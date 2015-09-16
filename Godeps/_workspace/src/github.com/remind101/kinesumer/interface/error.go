package kinesumeriface

const (
	ECrit  = "crit"
	EError = "error"
	EWarn  = "warn"
	EInfo  = "info"
	EDebug = "debug"
)

type Error interface {
	Severity() string
	Origin() error
	Error() string
}
