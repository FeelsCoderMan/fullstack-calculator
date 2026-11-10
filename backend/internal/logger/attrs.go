package logger

import (
	"log/slog"
	"time"
)

// Typed attribute constructors. Log only through these, with
// (*slog.Logger).LogAttrs, so every key is a known constant and every value has
// the right type; loose key/value pairs (logger.Info(msg, "k", v)) are not used.

// Err records an error's message. A nil error yields an empty attribute, which
// slog handlers drop.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.String(keyError, err.Error())
}

// Addr records a network address.
func Addr(addr string) slog.Attr { return slog.String(keyAddr, addr) }

// Timeout records a duration setting in Go notation (e.g. "10s").
func Timeout(d time.Duration) slog.Attr { return slog.String(keyTimeout, d.String()) }

// Method records an HTTP method.
func Method(method string) slog.Attr { return slog.String(keyMethod, method) }

// Path records a URL path.
func Path(path string) slog.Attr { return slog.String(keyPath, path) }

// Status records an HTTP status code.
func Status(code int) slog.Attr { return slog.Int(keyStatus, code) }

// Duration records an elapsed time in fractional milliseconds.
func Duration(d time.Duration) slog.Attr {
	return slog.Float64(keyDurationMS, float64(d)/float64(time.Millisecond))
}

// RequestID records the ID that correlates a request's records (and the
// X-Request-ID response header).
func RequestID(id string) slog.Attr { return slog.String(keyRequestID, id) }

// Code records an error code sent to the client (e.g. "error.internal").
func Code(code string) slog.Attr { return slog.String(keyCode, code) }

// Stack records a stack trace as an array of frames, keeping the record on
// one JSON line. A nil or empty stack yields an empty attribute, which slog
// handlers drop.
func Stack(frames []string) slog.Attr {
	if len(frames) == 0 {
		return slog.Attr{}
	}
	return slog.Any(keyStack, frames)
}

// Dir records a filesystem directory.
func Dir(dir string) slog.Attr { return slog.String(keyDir, dir) }

func service() slog.Attr { return slog.String(keyService, ServiceName) }

func category(c Category) slog.Attr { return slog.String(keyLogger, string(c)) }
