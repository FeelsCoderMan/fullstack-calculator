package middleware

import (
	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/apperror"
)

// Failure is what a failed request reports to RequestLogger, which logs it
// once, in the request record (docs/logging.md).
type Failure struct {
	// Err is the error as returned; its text has the full wrap context.
	Err error
	// AppErr is Err resolved to an application error, for its stack.
	AppErr *apperror.Error
	// Code is the code the client actually received.
	Code apperror.Code
}

// RecordFailure attaches f to the request for RequestLogger. The handler's
// error writer calls it; nothing else needs to.
func RecordFailure(c *gin.Context, f Failure) {
	c.Set(contextKeyFailure, f)
}

// recordedFailure returns the Failure recorded for this request.
func recordedFailure(c *gin.Context) (Failure, bool) {
	v, ok := c.Get(contextKeyFailure)
	if !ok {
		return Failure{}, false
	}
	f, ok := v.(Failure)
	return f, ok
}
