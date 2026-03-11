package commands

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
	"github.com/OctalMesh/Commodore/internal/ui/tui"
)

/*
NewDownCmd returns a ready-to-use `down` command.
See NewUpCmd for parameter documentation.
 */
func NewDownCmd(brand string, cfg domain.Config, finder ports.Finder) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the " + brand + " dev environment",
		Long: brand + ` down - stop the dev environment.

Sends SIGTERM to the running Tilt process and waits for a clean shutdown.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := finder.Find()
			if err != nil {
				return err
			}

			tiltfilePath, err := resolveTiltfile(root, cfg)
			if err != nil {
				return err
			}

			argv := []string{"tilt", "down", "--file", tiltfilePath}

			m := tui.NewDownModel(brand, staticFinder(root), argv)
			p := tea.NewProgram(m)

			final, err := p.Run()
			if err != nil {
				return fmt.Errorf("tui: %w", err)
			}

			finalModel, ok := final.(tui.Model)
			if !ok {
				return fmt.Errorf("unexpected model type: %T", final)
			}

			if finalModel.LegacyMode() {
				legacy := exec.Command("tilt", "down", "--file", tiltfilePath)
				legacy.Dir = root
				legacy.Stdin = os.Stdin
				legacy.Stdout = os.Stdout
				legacy.Stderr = os.Stderr
				return legacy.Run()
			}

			if ferr := finalModel.FinalErr(); ferr != nil {
				return ferr
			}
			return nil
		},
	}
}
