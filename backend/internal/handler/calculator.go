package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

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
	Percentage(a, b float64, p service.Precision) (float64, error)
	Sqrt(x float64, p service.Precision) (float64, error)
}

// binaryOperation is one two-operand Calculator method, e.g. calc.Add.
type binaryOperation func(a, b float64, p service.Precision) (float64, error)

// resultResponse is the body of a successful operation: data only.
type resultResponse struct {
	Result float64 `json:"result"`
}

// calculatorHandler serves the calculator operations.
type calculatorHandler struct {
	calc     Calculator
	validate *validator.Validate
}

func newCalculatorHandler(calc Calculator, validate *validator.Validate) *calculatorHandler {
	return &calculatorHandler{calc: calc, validate: validate}
}

// registerHandlers registers every operation on rg, the versioned API group.
func (h *calculatorHandler) registerHandlers(rg gin.IRoutes) {
	rg.POST(routeAdd, h.add)
	rg.POST(routeSubtract, h.subtract)
	rg.POST(routeMultiply, h.multiply)
	rg.POST(routeDivide, h.divide)
	rg.POST(routePower, h.power)
	rg.POST(routePercentage, h.percentage)
	rg.POST(routeSqrt, h.sqrt)
}

func (h *calculatorHandler) add(c *gin.Context)        { h.binary(c, h.calc.Add) }
func (h *calculatorHandler) subtract(c *gin.Context)   { h.binary(c, h.calc.Subtract) }
func (h *calculatorHandler) multiply(c *gin.Context)   { h.binary(c, h.calc.Multiply) }
func (h *calculatorHandler) divide(c *gin.Context)     { h.binary(c, h.calc.Divide) }
func (h *calculatorHandler) power(c *gin.Context)      { h.binary(c, h.calc.Power) }
func (h *calculatorHandler) percentage(c *gin.Context) { h.binary(c, h.calc.Percentage) }

func (h *calculatorHandler) sqrt(c *gin.Context) {
	req, err := bindJSON[sqrtRequest](c, h.validate)
	if err != nil {
		writeError(c, err)
		return
	}
	result, err := h.calc.Sqrt(req.Operand.value, req.Precision.precision())
	writeResult(c, result, err)
}

// binary serves a two-operand operation: it binds a binaryRequest and runs op.
func (h *calculatorHandler) binary(c *gin.Context, op binaryOperation) {
	req, err := bindJSON[binaryRequest](c, h.validate)
	if err != nil {
		writeError(c, err)
		return
	}
	result, err := op(req.LeftOperand.value, req.RightOperand.value, req.Precision.precision())
	writeResult(c, result, err)
}

// writeResult ends an operation: {"result": ...} on success, otherwise the
// error response from writeError.
func writeResult(c *gin.Context, result float64, err error) {
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resultResponse{Result: result})
}
