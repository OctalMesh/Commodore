package commands

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/app"
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
	"github.com/OctalMesh/Commodore/internal/ui/styles"
	"github.com/OctalMesh/Commodore/internal/ui/tui"
)

var (
	stHireHeader = lipgloss.NewStyle().Foreground(styles.Error)
	stHireName   = lipgloss.NewStyle().Foreground(styles.Mint).Bold(true)
	stHireMuted  = lipgloss.NewStyle().Foreground(styles.Muted)
	stHireHint   = lipgloss.NewStyle().Foreground(styles.Fg)
)

/*
NewHireBrigadierCmd returns the "hire" command for a Foreman CLI.

The first argument is the brigadier id; all subsequent arguments are forwarded
to the brigadier binary unchanged. Flag parsing is disabled so flags meant for
the brigadier are not intercepted by cobra.

Example:

	ow hire console up
	ow hire console hire app-web
*/
func NewHireBrigadierCmd(brigadiers []domain.Brigadier, svc *service.ProxyService) *cobra.Command {
	return &cobra.Command{
		Use:   "hire [brigadier-id] [args...]",
		Short: "Delegate a command to a named brigadier",
		Long:  "hire delegates execution to the named brigadier binary.\n\nAll arguments after the brigadier id are forwarded unchanged.",
		Example: "  ow hire console up\n" +
			"  ow hire console hire app-web",
		DisableFlagParsing: true,
		Args:               cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			for _, b := range brigadiers {
				if b.ID == id {
					return svc.Exec(b, args[1:])
				}
			}
			printHireBrigadierError(cmd, id, brigadiers)
			return app.ErrSilent
		},
	}
}

/*
NewHireWorkerCmd returns the "hire" command for a Brigadier CLI.

The first argument is the worker id. The command resolves the worker's
Tiltfile path via svc and launches the standard Tilt TUI in the worker's
working directory.

Example:

	owc hire app-web
*/
func NewHireWorkerCmd(brand string, workers []domain.Worker, svc *service.HireWorkerService) *cobra.Command {
	return &cobra.Command{
		Use:     "hire [worker-id]",
		Short:   "Start a specific worker's dev environment",
		Long:    "hire starts the Tilt dev environment for a single worker.",
		Example: "  owc hire app-web",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tiltfile, workerRoot, err := svc.WorkerTiltfile(workers, args[0])
			if err != nil {
				printHireWorkerError(cmd, args[0], workers, err)
				return app.ErrSilent
			}

			argv := []string{
				"tilt", "up",
				"--file", tiltfile,
				"--hud=false",
				"--stream=true",
			}

			// workerFinder returns the pre-resolved worker root so the TUI
			// sets the correct working directory for the tilt process.
			wf := workerFinder(workerRoot)
			m := tui.NewModel(brand+" / "+args[0], wf, argv)
			p := tea.NewProgram(m)

			final, err := p.Run()
			if err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			finalModel, ok := final.(tui.Model)
			if !ok {
				return fmt.Errorf("unexpected model type: %T", final)
			}
			if ferr := finalModel.FinalErr(); ferr != nil {
				return ferr
			}
			return nil
		},
	}
}

/*
printHireBrigadierError prints a styled error message when a brigadier is not
found, listing all available brigadiers with their descriptions.
 */
func printHireBrigadierError(cmd *cobra.Command, id string, brigadiers []domain.Brigadier) {
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  "+stHireHeader.Render("x")+"  "+stHireHeader.Render(fmt.Sprintf("brigadier %q not found", id)))
	fmt.Fprintln(w)

	if len(brigadiers) > 0 {
		fmt.Fprintln(w, "  "+stHireMuted.Render("Available brigadiers:"))
		nameW := maxIDWidth(brigadiersIDs(brigadiers))
		for _, b := range brigadiers {
			pad := strings.Repeat(" ", nameW-len(b.ID)+2)
			fmt.Fprintln(w, "    "+stHireName.Render(b.ID)+pad+stHireMuted.Render(b.Path))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n",
		stHireHint.Render(fmt.Sprintf("Usage: %s hire <brigadier-id> [args...]", cmd.Root().Name())))
	fmt.Fprintf(w, "  %s\n",
		stHireMuted.Render(fmt.Sprintf("Tip:   %s hire %s up",
			cmd.Root().Name(), firstOrDefault(brigadiersIDs(brigadiers), "console"))))
	fmt.Fprintln(w)
	os.Stderr.Sync() //nolint
}

// printHireWorkerError prints a styled error when a worker id is not found.
func printHireWorkerError(cmd *cobra.Command, id string, workers []domain.Worker, origErr error) {
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  "+stHireHeader.Render("x")+"  "+stHireHeader.Render(origErr.Error()))
	fmt.Fprintln(w)

	if len(workers) > 0 {
		fmt.Fprintln(w, "  "+stHireMuted.Render("Available workers:"))
		nameW := maxIDWidth(workersIDs(workers))
		for _, wk := range workers {
			pad := strings.Repeat(" ", nameW-len(wk.ID)+2)
			fmt.Fprintln(w, "    "+stHireName.Render(wk.ID)+pad+stHireMuted.Render(wk.Path))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n",
		stHireHint.Render(fmt.Sprintf("Usage: %s hire <worker-id>", cmd.Root().Name())))
	fmt.Fprintln(w)
	os.Stderr.Sync() //nolint
}

// workerFinder is a tui.Finder that always returns the same pre-resolved path.
type workerFinder string

func (f workerFinder) Find() (string, error) { return string(f), nil }

func brigadiersIDs(bs []domain.Brigadier) []string {
	ids := make([]string, len(bs))
	for i, b := range bs {
		ids[i] = b.ID
	}
	return ids
}

func workersIDs(ws []domain.Worker) []string {
	ids := make([]string, len(ws))
	for i, w := range ws {
		ids[i] = w.ID
	}
	return ids
}

func maxIDWidth(ids []string) int {
	maxW := 0
	for _, id := range ids {
		if len(id) > maxW {
			maxW = len(id)
		}
	}
	return maxW
}

func firstOrDefault(ids []string, def string) string {
	if len(ids) > 0 {
		return ids[0]
	}
	return def
}
