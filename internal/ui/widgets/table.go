package widgets

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	lipglosstable "github.com/charmbracelet/lipgloss/table"

	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

// TableColumn defines one reusable table column.
type TableColumn struct {
	Header   string
	MaxWidth int
}

// RenderTable renders rows to styled text lines using lipgloss/table.
func RenderTable(columns []TableColumn, rows [][]string) []string {
	if len(columns) == 0 {
		return nil
	}

	headers := make([]string, 0, len(columns))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Fg)
	for _, column := range columns {
		headers = append(headers, headerStyle.Render(clipText(column.Header, column.MaxWidth)))
	}

	normalizedRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		normalized := make([]string, len(columns))
		for columnIndex := range columns {
			if columnIndex < len(row) {
				normalized[columnIndex] = clipText(row[columnIndex], columns[columnIndex].MaxWidth)
			}
		}
		normalizedRows = append(normalizedRows, normalized)
	}

	bodyStyle := lipgloss.NewStyle().Foreground(styles.Muted)

	rendered := lipglosstable.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Muted)).
		Headers(headers...).
		Rows(normalizedRows...).
		StyleFunc(func(_ int, _ int) lipgloss.Style { return bodyStyle }).
		String()

	rendered = strings.TrimRight(rendered, "\n")
	if rendered == "" {
		return nil
	}

	return strings.Split(rendered, "\n")
}

func clipText(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return value
	}
	if utf8.RuneCountInString(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return firstRunes(value, maxWidth)
	}
	return firstRunes(value, maxWidth-3) + "..."
}

func firstRunes(value string, count int) string {
	if count <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= count {
		return value
	}
	return string(runes[:count])
}
