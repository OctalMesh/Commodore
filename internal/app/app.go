/*
Package app provides the App type that wraps a Cobra root command and implements
a consistent, styled Execute()/Run() entry-point for all Commodore-based CLIs.

All error handling lives here so individual commands only need to return errors
and main() requires no boilerplate.
*/
package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

var (
	stErr  = lipgloss.NewStyle().Foreground(styles.Error)
	stMute = lipgloss.NewStyle().Foreground(styles.Muted)
)

// ErrSilent can be returned from any cobra RunE to signal that the command
// already printed its own error output. App.Run() will still exit with code 1
// but will not print an additional error message.
var ErrSilent = errors.New("silent error")

// App wraps a Cobra root command with a styled error handler.
type App struct {
	Root *cobra.Command
}

/*
Run executes the CLI. Errors are printed to stderr in a styled format.
The process exits with code 1 on any error.

Callers should use this as their entire main() body:

	foreman.New(cfg, version).Run()
*/
func (a *App) Run() {
	if err := a.Root.Execute(); err != nil {
		if !errors.Is(err, ErrSilent) {
			fmt.Fprintln(os.Stderr,
				"  "+stErr.Render("x")+"  "+stErr.Render(err.Error()),
			)
			fmt.Fprintln(os.Stderr,
				"  "+stMute.Render("Run '"+a.Root.Name()+" --help' for usage."),
			)
		}
		os.Exit(1)
	}
}
