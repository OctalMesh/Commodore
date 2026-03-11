/*
Package commands provides ready-to-use Cobra commands shared by all
OctalWeb CLIs (Foreman and Brigadiers).
Each NewXxxCmd() function returns a fully-wired *cobra.Command that callers
can attach to any root command.
*/
package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

var (
	stHeader  = lipgloss.NewStyle().Foreground(styles.Purple).Bold(true)
	stSep     = lipgloss.NewStyle().Foreground(styles.Muted)
	stStep    = lipgloss.NewStyle().Foreground(styles.Mint)
	stSuccess = lipgloss.NewStyle().Foreground(styles.Mint)
	stWarn    = lipgloss.NewStyle().Foreground(styles.Warn)
	stError   = lipgloss.NewStyle().Foreground(styles.Error)
	stFg      = lipgloss.NewStyle().Foreground(styles.Fg)
	stMuted   = lipgloss.NewStyle().Foreground(styles.Muted)
	stColHead = lipgloss.NewStyle().Foreground(styles.Fg).Bold(true)
)

func printHeader(label string) {
	fmt.Println()
	fmt.Println("  " + stHeader.Render(label))
	fmt.Println(stSep.Render("  " + strings.Repeat("─", 72)))
}

func printStep(msg string) { fmt.Println("  " + stStep.Render("->") + "  " + msg) }

func printOk(msg string) { fmt.Println("  " + stSuccess.Render("✓") + "  " + msg) }

func printWarn(msg string) { fmt.Println("  " + stWarn.Render("⚠") + "  " + msg) }

func printErr(msg string) {
	fmt.Fprintln(os.Stderr, "  "+stError.Render("✗")+"  "+msg)
}

func printInfo(msg string) { fmt.Println("    " + stMuted.Render(msg)) }

func printBlank() { fmt.Println() }

// ColDef describes a table column.
type ColDef struct {
	Label string
	Width int
	Paint func(string) string
}

func cellPad(s string, w int) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// BuildTable renders a CLI table with a header row.
func BuildTable(cols []ColDef, rows [][]string) string {
	total := 2
	for _, c := range cols {
		total += c.Width + 2
	}
	sep := stSep.Render("  " + strings.Repeat("─", total))

	var sb strings.Builder
	sb.WriteString(sep + "\n")
	sb.WriteString("  ")
	for _, c := range cols {
		sb.WriteString(cellPad(stColHead.Render(c.Label), c.Width))
		sb.WriteString("  ")
	}
	sb.WriteString("\n")
	sb.WriteString(sep + "\n")

	for _, row := range rows {
		sb.WriteString("  ")
		for i, c := range cols {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if c.Paint != nil {
				cell = c.Paint(cell)
			}
			sb.WriteString(cellPad(cell, c.Width))
			sb.WriteString("  ")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(sep)
	return sb.String()
}

// PrintTable prints the table to stdout.
func PrintTable(cols []ColDef, rows [][]string) { fmt.Println(BuildTable(cols, rows)) }
