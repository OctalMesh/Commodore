/*
Package styles defines the shared color palette for all OctalWeb CLIs.
Import this package to stay visually consistent across Foreman and Brigadiers.
*/
package styles

import "github.com/charmbracelet/lipgloss"

// Color tokens - referenced by both TUI and plain output helpers.
var (
	Purple   = lipgloss.Color("#9B8DCA") // soft lavender - brand / accent
	Mint     = lipgloss.Color("#7ECFB0") // mint green    - success / running
	Graphite = lipgloss.Color("#3D404D") // dark graphite - backgrounds
	Fg       = lipgloss.Color("#E2DFF0") // near-white    - log / body text
	Muted    = lipgloss.Color("#6B6E7F") // graphite grey - secondary / hints
	Error    = lipgloss.Color("#E07B7B") // soft red      - errors
	Warn     = lipgloss.Color("#E0C97B") // amber         - warnings
)
