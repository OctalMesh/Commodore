/*
Package brigadier provides the factory for a Brigadier CLI command tree.

A Brigadier is a module CLI (owc, ows, owl, owd, ...) that manages a single
OctalMesh module. It provides the common up / down / hire commands out of the
box; callers may add extra custom commands via the extra variadic parameter.
*/
package brigadier

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/adapters/config"
	"github.com/OctalMesh/Commodore/internal/adapters/moduledir"
	"github.com/OctalMesh/Commodore/internal/app"
	"github.com/OctalMesh/Commodore/internal/commands"
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
)

/*
New builds and returns the fully-wired Cobra command tree for a Brigadier CLI.

cfg is the module's configuration (role: brigadier). The ID field is used
to locate the module root directory at runtime.
version is the build version string injected at link time.
extra are any additional *cobra.Command instances the caller wants to register
(e.g. module-specific commands beyond the standard up / down / hire).
*/
func New(cfg domain.Config, version string, extra ...*cobra.Command) *app.App {
	finder := moduledir.New(cfg.ID)
	loader := config.New()

	// Services
	svcHire := service.NewHireWorkerService(finder, loader)

	// Root command
	binary := cfg.Binary
	if binary == "" {
		binary = cfg.ID
	}

	root := &cobra.Command{
		Use:           binary,
		Aliases:       cfg.Aliases,
		Short:         "OctalMesh " + cfg.ID + " module CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(commands.NewUpCmd(cfg.ID, cfg, finder))
	root.AddCommand(commands.NewDownCmd(cfg.ID, cfg, finder))

	if len(cfg.Workers) > 0 {
		root.AddCommand(commands.NewHireWorkerCmd(cfg.ID, cfg.Workers, svcHire))
	}

	for _, cmd := range extra {
		root.AddCommand(cmd)
	}

	root.AddCommand(newCompatCmd())

	root.SetErrPrefix(fmt.Sprintf("  %s  ", "\x1b[31mx\x1b[0m"))
	root.SetOut(os.Stdout)

	return &app.App{Root: root}
}

/*
newCompatCmd returns the hidden "__compat" sub-command added to every root
command built with the Commodore SDK. Running "binary __compat" lets a parent
CLI verify that the binary is Commodore-compatible before delegating work.
*/
func newCompatCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "__compat",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "commodore:sdk:github.com/OctalMesh/Commodore")
		},
	}
}
