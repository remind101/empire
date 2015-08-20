package kinesumer

const (
	ECrit  = "crit"
	EError = "error"
	EWarn  = "warn"
	EInfo  = "info"
	EDebug = "debug"
)

type Error struct {
	// One of "crit", "error", "warn", "info", "debug"
	severity string
	message  string
	origin   error
}

func NewError(severity, message string, origin error) *Error {
	return &Error{
		severity: severity,
		message:  message,
		origin:   origin,
	}
}

func (e *Error) Severity() string {
	return e.severity
}

func (e *Error) Origin() error {
	return e.origin
}

func (e *Error) Error() string {
	if e.origin == nil {
		return e.message
	} else {
		return e.message + " from " + e.origin.Error()
	}
}
