package logger

import "os"

// Category names a logger. It is emitted as the "logger" attribute and used
// in log file names. Keep the set small: it is a low-cardinality Loki label.
type Category string

// Logger categories.
const (
	CategoryServer Category = "server"
	CategoryHTTP   Category = "http"
)

// Categories returns every known category.
func Categories() []Category {
	return []Category{CategoryServer, CategoryHTTP}
}

// ServiceName identifies this service in every log record.
const ServiceName = "calculator-backend"

// Attribute keys. Unexported on purpose: callers attach data only through the
// typed constructors in attrs.go.
const (
	keyService    = "service"
	keyLogger     = "logger"
	keyError      = "error"
	keyAddr       = "addr"
	keyTimeout    = "timeout"
	keyMethod     = "method"
	keyPath       = "path"
	keyStatus     = "status"
	keyDurationMS = "duration_ms"
	keyDir        = "dir"
	keyRequestID  = "request_id"
	keyCode       = "code"
	keyStack      = "stack"
)

// Log file settings (development-only file output).
const (
	fileNameFormat = "%s-%s.log"
	fileTimeLayout = "20060102T150405Z"
	dirPerm        = 0o755
	filePerm       = 0o644
	fileFlags      = os.O_CREATE | os.O_WRONLY | os.O_APPEND
)
