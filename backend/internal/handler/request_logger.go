package handler

import (
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"fullstack-calculator/backend/internal/apperror"
	"fullstack-calculator/backend/internal/logger"
)

var validRequestID = regexp.MustCompile(requestIDPattern)

// RequestLogger writes exactly one record per request (docs/logging.md): after
// the handler chain finishes it logs method, path, status, duration and
// request ID, plus the code, error and (for unexpected failures) stack that
// writeError recorded. Nothing else logs per request, so nothing is logged
// twice.
//
// It also assigns the request ID (echoed in X-Request-ID) and recovers
// panics: a panic becomes a 500 error.internal response and is reported in the
// same single record, with the panic's stack.
func RequestLogger(accessLog *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		id := requestID(c.GetHeader(HeaderRequestID))
		c.Header(HeaderRequestID, id)

		defer func() {
			v := recover()
			if v != nil {
				// Called directly in the deferred function so the stack
				// includes the panic site.
				panicErr := apperror.Panic(v)
				if !c.Writer.Written() {
					writeError(c, panicErr)
				} else {
					c.Set(contextKeyFailure, failure{err: panicErr, appErr: panicErr, code: panicErr.Code})
				}
			}

			logRequest(c, accessLog, id, time.Since(start))

			// net/http uses ErrAbortHandler to abort a response on purpose;
			// it must keep propagating (it is logged above, once).
			if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(v)
			}
		}()

		c.Next()
	}
}

func logRequest(c *gin.Context, accessLog *slog.Logger, id string, elapsed time.Duration) {
	path, status := c.Request.URL.Path, c.Writer.Status()
	attrs := []slog.Attr{
		logger.Method(c.Request.Method),
		logger.Path(path),
		logger.Status(status),
		logger.Duration(elapsed),
		logger.RequestID(id),
	}

	f, failed := recordedFailure(c)
	if failed {
		attrs = append(attrs,
			logger.Code(string(f.code)),
			logger.Err(f.err),
			logger.Stack(f.appErr.Stack()),
		)
	}

	accessLog.LogAttrs(c.Request.Context(), requestLevel(path, status, failed && f.code == apperror.CodeInternal),
		msgRequestCompleted, attrs...)
}

// requestLevel picks the record's level (docs/logging.md section 5): server
// failures are always errors, successful health probes are debug, and
// everything else, client errors included, is info.
func requestLevel(path string, status int, internal bool) slog.Level {
	switch {
	case internal || status >= http.StatusInternalServerError:
		return slog.LevelError
	case path == PathHealth && status < http.StatusBadRequest:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// requestID reuses a well-formed incoming ID and otherwise generates one.
func requestID(incoming string) string {
	if validRequestID.MatchString(incoming) {
		return incoming
	}
	return rand.Text()
}
