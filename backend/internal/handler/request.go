package handler

import (
	"bytes"
	"encoding/json"
	"strconv"

	"fullstack-calculator/backend/internal/service"
)

// Request bodies. Each endpoint binds its own type with bindJSON; the `json`
// tag is the property name clients send (and the "field" of a field error),
// the `binding` tag lists its rules in the order they are checked (see
// validationTagCodes for the error code of each rule). Fields are reported in
// declaration order, so declare them in api/openapi.yaml schema order.
//
// The precision bounds (min=0, max=15) mirror service.MinPrecision and
// service.MaxPrecision.
//
// Every field must be a number with a json name: bindJSON relies on
// number.UnmarshalJSON never failing. A new request type also goes into
// requestTypes (bind_test.go), which checks this and the rules.

// binaryRequest is the body of every two-operand operation
// (api/openapi.yaml BinaryOperationRequest).
type binaryRequest struct {
	LeftOperand  number `json:"leftOperand"  binding:"required,jsonnumber,finite"`
	RightOperand number `json:"rightOperand" binding:"required,jsonnumber,finite"`
	Precision    number `json:"precision"    binding:"omitempty,jsonnumber,finite,integer,min=0,max=15"`
}

// sqrtRequest is the body of sqrt (api/openapi.yaml UnaryOperationRequest). A negative
// operand is valid here; the service rejects it as a math error.
type sqrtRequest struct {
	Operand   number `json:"operand"   binding:"required,jsonnumber,finite"`
	Precision number `json:"precision" binding:"omitempty,jsonnumber,finite,integer,min=0,max=15"`
}

// number is a JSON number property of a request body.
//
// Its UnmarshalJSON never fails: it records what was sent, and the binding
// rules report a missing, mistyped or out-of-range value. A failing
// UnmarshalJSON would stop encoding/json at the first bad property, hiding the
// others; this way every problem is reported at once. The validator sees a
// number through numberValue.
type number struct {
	sent     bool    // the property is present and not null
	isNumber bool    // the value is a JSON number
	value    float64 // the parsed value; ±Inf when outside the float64 range
	raw      string  // the value as sent, when it is not a number
}

// UnmarshalJSON records data; see number.
func (n *number) UnmarshalJSON(data []byte) error {
	*n = number{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	// data is one value cut from a body that already decoded, so it decodes
	// again; v stays nil only for the JSON literal null.
	_ = dec.Decode(&v)
	if v == nil {
		return nil
	}
	n.sent = true
	num, ok := v.(json.Number)
	if !ok {
		n.raw = string(data)
		return nil
	}
	n.isNumber = true
	// A JSON number is valid float syntax, so the only possible error is
	// strconv.ErrRange, which comes with ±Inf: the finite rule rejects it.
	// Values below the float64 range underflow to 0, which is accepted
	// (api/openapi.yaml).
	n.value, _ = strconv.ParseFloat(num.String(), bitSize)
	return nil
}

// precision converts a validated precision property: absent or null means
// no rounding.
func (n number) precision() service.Precision {
	if !n.sent {
		return service.Precision{}
	}
	return service.RoundTo(int(n.value))
}
