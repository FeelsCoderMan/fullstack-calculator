package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/service"
)

// Calculator is the arithmetic the HTTP layer serves. service.Calculator
// implements it; tests substitute a fake to reach failures the real one can't
// produce.
type Calculator interface {
	Add(a, b float64, p service.Precision) (float64, error)
	Subtract(a, b float64, p service.Precision) (float64, error)
	Multiply(a, b float64, p service.Precision) (float64, error)
	Divide(a, b float64, p service.Precision) (float64, error)
	Power(a, b float64, p service.Precision) (float64, error)
	Sqrt(x float64, p service.Precision) (float64, error)
}

// binaryOperation is one two-operand Calculator method, e.g. calc.Add.
type binaryOperation func(a, b float64, p service.Precision) (float64, error)

// unaryOperation is one one-operand Calculator method, e.g. calc.Sqrt.
type unaryOperation func(x float64, p service.Precision) (float64, error)

// resultResponse is the body of a successful operation: data only.
type resultResponse struct {
	Result float64 `json:"result"`
}

// binary serves a two-operand operation: it validates the body, runs op and
// writes {"result": ...}, or ends the request with writeError.
func binary(op binaryOperation) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := decodeBinaryRequest(c)
		if err != nil {
			writeError(c, err)
			return
		}
		result, err := op(req.left, req.right, req.precision)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, resultResponse{Result: result})
	}
}

// unary serves a one-operand operation: it validates the body, runs op and
// writes {"result": ...}, or ends the request with writeError.
func unary(op unaryOperation) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := decodeUnaryRequest(c)
		if err != nil {
			writeError(c, err)
			return
		}
		result, err := op(req.operand, req.precision)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, resultResponse{Result: result})
	}
}
