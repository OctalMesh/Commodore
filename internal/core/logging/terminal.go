package logging

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"

	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

var (
	logStyleInfo    = lipgloss.NewStyle().Foreground(styles.Muted)
	logStyleWarn    = lipgloss.NewStyle().Foreground(styles.Warn)
	logStyleSuccess = lipgloss.NewStyle().Foreground(styles.Mint)
	logStyleError   = lipgloss.NewStyle().Foreground(styles.Error)
)

type terminalLogger struct{}

// NewTerminalLogger creates a logger that writes styled output to stdout/stderr.
func NewTerminalLogger() Logger {
	return terminalLogger{}
}

func (terminalLogger) Info(message string, arguments ...any) {
	line := fmt.Sprintf(message, arguments...)
	_, _ = fmt.Fprintln(os.Stdout, logStyleInfo.Render("  "+line))
}

func (terminalLogger) Warn(message string, arguments ...any) {
	line := fmt.Sprintf(message, arguments...)
	_, _ = fmt.Fprintln(os.Stderr, logStyleWarn.Render("  ⚠ "+line))
}

func (terminalLogger) Success(message string, arguments ...any) {
	line := fmt.Sprintf(message, arguments...)
	_, _ = fmt.Fprintln(os.Stdout, logStyleSuccess.Render("  ✓ "+line))
}

func (terminalLogger) Error(message string, arguments ...any) {
	line := fmt.Sprintf(message, arguments...)
	_, _ = fmt.Fprintln(os.Stderr, logStyleError.Render("  ✗ "+line))
}
