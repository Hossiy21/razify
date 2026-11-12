package reporter

import "os"

const (
	// ExitClean indicates all checks passed without violations or errors.
	ExitClean = 0

	// ExitViolation indicates schema violations or secret leaks were detected.
	ExitViolation = 1

	// ExitError indicates a fatal operational error (missing file, bad args, I/O error).
	ExitError = 2
)

// ExitHandler manages standardized exit code resolution with optional overrides.
type ExitHandler struct {
	OverrideViolationCode int
}

// NewExitHandler creates an ExitHandler. A negative overrideCode leaves ExitViolation unchanged.
func NewExitHandler(overrideCode int) *ExitHandler {
	return &ExitHandler{
		OverrideViolationCode: overrideCode,
	}
}

// ResolveViolationCode returns the appropriate exit code for violations.
func (h *ExitHandler) ResolveViolationCode() int {
	if h != nil && h.OverrideViolationCode >= 0 {
		return h.OverrideViolationCode
	}
	return ExitViolation
}

// ExitWithCode terminates the process with the given exit code.
func ExitWithCode(code int) {
	os.Exit(code)
}

