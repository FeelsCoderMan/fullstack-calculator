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
	"reflect"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"fullstack-calculator/backend/internal/apperror"
)

// bindJSON reads the request body into a T and checks it against T's
// `binding` rules.
//
// A body that can't be read as one JSON object fails with a request error
// (415, 413 or 400). Anything wrong with its properties fails with one
// validation error (422) listing every problem: T's fields in declaration
// order, then unknown properties alphabetically.
func bindJSON[T any](c *gin.Context, validate *validator.Validate) (T, error) {
	var req T
	props, err := readJSONObject(c)
	if err != nil {
		return req, err
	}

	// Decode only the properties T declares, matched by exact name:
	// encoding/json alone would also accept "LEFTOPERAND" for leftOperand.
	known := make(map[string]json.RawMessage, len(props))
	for _, name := range jsonNames(reflect.TypeFor[T]()) {
		if raw, ok := props[name]; ok {
			known[name] = raw
			delete(props, name)
		}
	}
	body, err := json.Marshal(known)
	if err != nil {
		return req, apperror.Internal(fmt.Errorf("bind body: %w", err))
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, apperror.Internal(fmt.Errorf("bind body: %w", err))
	}

	fieldErrs, err := fieldErrors(validate.Struct(req))
	if err != nil {
		return req, err
	}
	for _, name := range slices.Sorted(maps.Keys(props)) { // what is left is unknown
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: name, Code: apperror.CodeFieldUnknown})
	}
	if len(fieldErrs) > 0 {
		return req, apperror.Validation(fieldErrs...)
	}
	return req, nil
}

// readJSONObject reads the body as one JSON object, split into its raw
// properties. It rejects a wrong Content-Type, a body over maxBodyBytes,
// malformed JSON, a JSON value that isn't an object, and trailing data after
// the object.
func readJSONObject(c *gin.Context) (map[string]json.RawMessage, error) {
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
	var props map[string]json.RawMessage
	if err := dec.Decode(&props); err != nil {
		return nil, fmt.Errorf("decode body: %w: %w", err, apperror.ErrInvalidJSON)
	}
	if props == nil { // the body was the JSON literal null
		return nil, fmt.Errorf("decode body: not an object: %w", apperror.ErrInvalidJSON)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decode body: data after the object: %w", apperror.ErrInvalidJSON)
	}
	return props, nil
}

// validationTagCodes maps each binding rule to the field code reported when
// it fails. A rule used in a request type must be listed here
// (TestValidationTagsHaveCodes).
var validationTagCodes = map[string]apperror.Code{
	tagRequired:   apperror.CodeFieldRequired,
	tagJSONNumber: apperror.CodeFieldInvalidType,
	tagInteger:    apperror.CodeFieldInvalidType,
	tagFinite:     apperror.CodeFieldOutOfRange,
	tagMin:        apperror.CodeFieldOutOfRange,
	tagMax:        apperror.CodeFieldOutOfRange,
}

// fieldErrors converts the result of validator.Struct into field errors, one
// per invalid field. A rule without a code is a bug and fails as internal.
func fieldErrors(err error) ([]apperror.FieldError, error) {
	if err == nil {
		return nil, nil
	}
	invalid, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return nil, apperror.Internal(fmt.Errorf("validate body: %w", err))
	}
	out := make([]apperror.FieldError, 0, len(invalid))
	for _, fe := range invalid {
		code, ok := validationTagCodes[fe.Tag()]
		if !ok {
			return nil, apperror.Internal(fmt.Errorf("validate %s: rule %q: %w", fe.Field(), fe.Tag(), errNoFieldCode))
		}
		out = append(out, apperror.FieldError{Field: fe.Field(), Code: code})
	}
	return out, nil
}

// newValidator returns the validator for request bodies: it reads `binding`
// tags, names fields by their JSON property, sees a number through
// numberValue and knows the number rules (jsonnumber, finite, integer).
func newValidator() (*validator.Validate, error) {
	v := validator.New()
	v.SetTagName(tagKeyBinding)
	v.RegisterTagNameFunc(jsonName)
	v.RegisterCustomTypeFunc(numberValue, number{})
	rules := map[string]validator.Func{
		tagJSONNumber: isJSONNumber,
		tagFinite:     isFinite,
		tagInteger:    isInteger,
	}
	for tag, fn := range rules {
		if err := v.RegisterValidation(tag, fn); err != nil {
			return nil, fmt.Errorf("register validation %q: %w", tag, err)
		}
	}
	return v, nil
}

// numberValue is what the validator checks for a number: nil when it was not
// sent (fails required, skipped by omitempty), the raw text when it is not a
// JSON number (fails jsonnumber), and otherwise a pointer to its value, so
// that required accepts 0 and min/max compare the number.
func numberValue(v reflect.Value) any {
	n, _ := v.Interface().(number) // registered for number only
	switch {
	case !n.sent:
		return nil
	case !n.isNumber:
		return n.raw
	default:
		return &n.value
	}
}

// isJSONNumber reports whether the field holds a number (see numberValue).
func isJSONNumber(fl validator.FieldLevel) bool {
	return fl.Field().Kind() == reflect.Float64
}

// isFinite reports whether the number fits in a float64.
func isFinite(fl validator.FieldLevel) bool {
	return isJSONNumber(fl) && !math.IsInf(fl.Field().Float(), 0)
}

// isInteger reports whether the number has no fractional part; 2.0 counts as
// the integer 2.
func isInteger(fl validator.FieldLevel) bool {
	return isFinite(fl) && fl.Field().Float() == math.Trunc(fl.Field().Float())
}

// jsonNames returns the JSON property names of t's fields, in declaration
// order.
func jsonNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		if name := jsonName(t.Field(i)); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// jsonName returns the JSON property name of a struct field, or "" when the
// field is not in the JSON form (json:"-").
func jsonName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get(tagKeyJSON), jsonTagSeparator)
	if name == jsonTagSkip {
		return ""
	}
	return name
}
