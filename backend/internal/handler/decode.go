package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"fullstack-calculator/backend/internal/apperror"
	"fullstack-calculator/backend/internal/service"
)

// binaryRequest is a validated two-operand request body.
type binaryRequest struct {
	left, right float64
	precision   service.Precision
}

// decodeBinaryRequest reads {"leftOperand", "rightOperand", "precision"?} and
// reports every invalid field at once, in a fixed order: leftOperand,
// rightOperand, precision, then unknown properties alphabetically.
func decodeBinaryRequest(c *gin.Context) (binaryRequest, error) {
	fields, err := decodeObject(c)
	if err != nil {
		return binaryRequest{}, err
	}
	r := fieldReader{fields: fields}
	req := binaryRequest{
		left:      r.number(fieldLeftOperand),
		right:     r.number(fieldRightOperand),
		precision: r.precision(fieldPrecision),
	}
	r.rejectUnknown()
	if err := r.err(); err != nil {
		return binaryRequest{}, err
	}
	return req, nil
}

// decodeObject reads the body as one JSON object, split into its raw
// properties so each can be validated on its own. It rejects a wrong
// Content-Type, a body over maxBodyBytes, malformed JSON, a JSON value that
// isn't an object, and trailing data after the object.
func decodeObject(c *gin.Context) (map[string]json.RawMessage, error) {
	// Media types are case-insensitive (RFC 9110 section 8.3.1).
	if ct := c.ContentType(); !strings.EqualFold(ct, binding.MIMEJSON) {
		return nil, fmt.Errorf("content type %q: %w", ct, apperror.ErrUnsupportedMediaType)
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, fmt.Errorf("read body: %w", apperror.ErrBodyTooLarge)
		}
		return nil, apperror.Internal(fmt.Errorf("read body: %w", err))
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	var fields map[string]json.RawMessage
	if err := dec.Decode(&fields); err != nil {
		return nil, fmt.Errorf("decode body: %w: %w", err, apperror.ErrInvalidJSON)
	}
	if fields == nil { // the body was the JSON literal null
		return nil, fmt.Errorf("decode body: not an object: %w", apperror.ErrInvalidJSON)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode body: data after the object: %w", apperror.ErrInvalidJSON)
	}
	return fields, nil
}

// fieldReader validates the properties of a decoded object, collecting every
// problem instead of stopping at the first. Each read removes the property, so
// whatever is left at the end is unknown.
type fieldReader struct {
	fields map[string]json.RawMessage
	errs   []apperror.FieldError
}

// number reads a required JSON number that fits in a float64. Values below
// the float64 range underflow to 0, which is accepted (api/openapi.yaml).
func (r *fieldReader) number(name string) float64 {
	v, ok := r.take(name)
	if !ok {
		r.fail(name, apperror.CodeFieldRequired)
		return 0
	}
	num, ok := v.(json.Number)
	if !ok {
		r.fail(name, apperror.CodeFieldInvalidType)
		return 0
	}
	f, err := strconv.ParseFloat(num.String(), bitSize)
	if err != nil { // a JSON number is valid syntax, so this is strconv.ErrRange
		r.fail(name, apperror.CodeFieldOutOfRange)
		return 0
	}
	return f
}

// precision reads an optional integer number of decimal places. Absent or
// null means no rounding. 2.0 counts as the integer 2.
func (r *fieldReader) precision(name string) service.Precision {
	v, ok := r.take(name)
	if !ok {
		return service.Precision{}
	}
	num, ok := v.(json.Number)
	if !ok {
		r.fail(name, apperror.CodeFieldInvalidType)
		return service.Precision{}
	}
	f, err := strconv.ParseFloat(num.String(), bitSize)
	switch {
	case err != nil:
		r.fail(name, apperror.CodeFieldOutOfRange)
	case f != math.Trunc(f):
		r.fail(name, apperror.CodeFieldInvalidType)
	case f < service.MinPrecision || f > service.MaxPrecision:
		r.fail(name, apperror.CodeFieldOutOfRange)
	default:
		return service.RoundTo(int(f))
	}
	return service.Precision{}
}

// rejectUnknown reports every property no read consumed, alphabetically.
func (r *fieldReader) rejectUnknown() {
	for _, name := range slices.Sorted(maps.Keys(r.fields)) {
		r.fail(name, apperror.CodeFieldUnknown)
	}
}

// err returns a validation error listing every problem, or nil.
func (r *fieldReader) err() error {
	if len(r.errs) == 0 {
		return nil
	}
	return apperror.Validation(r.errs...)
}

// take removes the property and returns its decoded value, with numbers kept
// as json.Number. It reports false when the property is missing or null.
func (r *fieldReader) take(name string) (any, bool) {
	raw, ok := r.fields[name]
	if !ok {
		return nil, false
	}
	delete(r.fields, name)

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	// raw is one value cut from a body that already decoded, so it decodes
	// again; v stays nil (null) only for the JSON literal null.
	_ = dec.Decode(&v)
	return v, v != nil
}

func (r *fieldReader) fail(name string, code apperror.Code) {
	r.errs = append(r.errs, apperror.FieldError{Field: name, Code: code})
}
