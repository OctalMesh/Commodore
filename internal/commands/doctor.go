package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

type doctorResultMsg struct {
	results  []domain.ToolResult
	checkErr error
}

type doctorModel struct {
	spinner  spinner.Model
	svc      *service.DoctorService
	done     bool
	view     string
	checkErr error
}

func newDoctorModel(svc *service.DoctorService) doctorModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Purple)
	return doctorModel{spinner: s, svc: svc}
}

func (m doctorModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		results, err := m.svc.Check()
		return doctorResultMsg{results: results, checkErr: err}
	})
}

func (m doctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case doctorResultMsg:
		m.view = renderDoctorTable(msg.results)
		m.checkErr = msg.checkErr
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m doctorModel) View() string {
	if !m.done {
		return "  " + m.spinner.View() + "  " +
			stStep.Render("Checking environment...") + "\n"
	}
	return m.view
}

func renderDoctorTable(results []domain.ToolResult) string {
	cols := []ColDef{
		{Label: "TOOL", Width: 10},
		{Label: "CMD", Width: 8},
		{Label: "STATUS", Width: 10, Paint: paintDoctorStatus},
		{Label: "TYPE", Width: 10, Paint: paintDoctorType},
		{Label: "VERSION / INSTALL", Width: 52, Paint: paintDoctorInfo},
	}

	rows := make([][]string, len(results))
	for i, r := range results {
		info := r.Version
		if !r.Found {
			info = "-> " + r.Tool.InstallURL
		}
		typeStr := "optional"
		if r.Tool.Required {
			typeStr = "required"
		}
		statusStr := "found"
		if !r.Found {
			statusStr = "MISSING"
		}
		rows[i] = []string{r.Tool.Name, r.Tool.Cmd, statusStr, typeStr, info}
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + stHeader.Render("Environment Check") + "\n")
	sb.WriteString(BuildTable(cols, rows))
	sb.WriteString("\n")
	return sb.String()
}

func paintDoctorStatus(s string) string {
	if s == "found" {
		return lipgloss.NewStyle().Foreground(styles.Mint).Render(s)
	}
	return lipgloss.NewStyle().Foreground(styles.Error).Bold(true).Render(s)
}

func paintDoctorType(s string) string {
	if s == "required" {
		return lipgloss.NewStyle().Bold(true).Render(s)
	}
	return stMuted.Render(s)
}

func paintDoctorInfo(s string) string {
	if strings.HasPrefix(s, "-> ") {
		return lipgloss.NewStyle().Foreground(styles.Mint).Render(s)
	}
	return stMuted.Render(s)
}

// NewDoctorCmd returns a ready-to-use `doctor` command backed by svc.
func NewDoctorCmd(svc *service.DoctorService) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that all required tools are installed",
		Long: `doctor - environment diagnostics.

Verifies that all external tools required by OctalWeb are present
on PATH and reports their detected versions.

Required:  git, go, docker, tilt
Optional:  node, npm`,
		RunE: func(_ *cobra.Command, _ []string) error {
			m := newDoctorModel(svc)
			p := tea.NewProgram(m)

			final, err := p.Run()
			if err != nil {
				return err
			}

			dm, ok := final.(doctorModel)
			if !ok {
				return fmt.Errorf("unexpected model type: %T", final)
			}
			if dm.checkErr != nil {
				printBlank()
				printErr(dm.checkErr.Error())
				printBlank()
				return dm.checkErr
			}

			printBlank()
			printOk("All required tools are present.")
			printBlank()
			return nil
		},
	}
}
