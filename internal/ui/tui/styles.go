/*
Package tui provides the shared BubbleTea TUI widget.
Styles live here so callers can reference them for post-TUI output.
*/
package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

// Exported styles - available to callers for post-TUI output.
var (
	StyleBrand         = lipgloss.NewStyle().Foreground(styles.Purple).Bold(true)
	StyleStep          = lipgloss.NewStyle().Foreground(styles.Mint)
	StyleMuted         = lipgloss.NewStyle().Foreground(styles.Muted)
	StyleError         = lipgloss.NewStyle().Foreground(styles.Error)
	StyleSuccess       = lipgloss.NewStyle().Foreground(styles.Mint)
	StyleSpinner       = lipgloss.NewStyle().Foreground(styles.Purple)
	StyleLog           = lipgloss.NewStyle().Foreground(styles.Fg)
	StyleStatusRunning = lipgloss.NewStyle().Foreground(styles.Mint)
	StyleTabActive     = lipgloss.NewStyle().Foreground(styles.Mint).Bold(true)
	StyleTab           = lipgloss.NewStyle().Foreground(styles.Muted)
	StyleWarn          = lipgloss.NewStyle().Foreground(styles.Warn)
)
