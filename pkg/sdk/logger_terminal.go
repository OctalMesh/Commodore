package sdk

import "github.com/OctalMesh/Commodore/internal/core/logging"

// NewTerminalLogger creates a logger that writes styled output to stdout/stderr.
func NewTerminalLogger() Logger {
	return logging.NewTerminalLogger()
}
