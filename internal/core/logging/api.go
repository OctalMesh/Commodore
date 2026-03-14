package logging

// Logger defines the unified logging contract for Commodore runtime and ports.
//
// Methods are intentionally level-specific for readable call sites and
// consistent terminal styling.
type Logger interface {
	Info(message string, arguments ...any)
	Success(message string, arguments ...any)
	Warn(message string, arguments ...any)
	Error(message string, arguments ...any)
}
