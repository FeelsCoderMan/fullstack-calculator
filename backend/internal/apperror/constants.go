package apperror

// Stack capture settings.
const (
	// maxStackFrames bounds the stack attached to unexpected errors.
	maxStackFrames = 32
	// skipToCaller skips runtime.Callers, capture and the exported constructor,
	// so the first frame is the code that called Internal or Panic.
	skipToCaller = 3
)

// Format strings.
const (
	errorWithCauseFormat = "%s: %v"
	stackFrameFormat     = "%s %s:%d"
	panicFormat          = "%w: %v"
)

// importPathSeparator separates the elements of a Go import path.
const importPathSeparator = "/"
