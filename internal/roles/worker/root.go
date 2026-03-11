/*
Package worker provides the factory for a Worker CLI command tree.

A Worker is a single-service CLI that does not manage sub-processes. It
exposes only the custom shell commands declared in the "commands:" section
of its .commodore file, plus any extra *cobra.Command instances supplied by
the caller.
*/
package worker

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/app"
	"github.com/OctalMesh/Commodore/internal/core/domain"
)

/*
New builds and returns the fully-wired Cobra command tree for a Worker CLI.

cfg is the worker's configuration (role: worker). Commands declared in
cfg.Commands are registered as runnable cobra sub-commands.
version is the build version string injected at link time.
extra are any additional *cobra.Command instances the caller wants to add
beyond what is declared in the config.
*/
func New(cfg domain.Config, version string, extra ...*cobra.Command) *app.App {
	binary := cfg.Binary
	if binary == "" {
		binary = cfg.ID
	}

	root := &cobra.Command{
		Use:           binary,
		Short:         cfg.ID + " worker CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	for _, wc := range cfg.Commands {
		root.AddCommand(newExecCmd(wc))
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
newExecCmd returns a cobra command that runs wc.Exec as a shell command,
inheriting the current process's stdin / stdout / stderr.

The shell is "sh -c" on Unix and "cmd /c" on Windows. This matches the
behavior of common dev tooling (Makefile, npm scripts, etc.).
*/
func newExecCmd(wc domain.WorkerCommand) *cobra.Command {
	return &cobra.Command{
		Use:   wc.Name,
		Short: wc.Description,
		RunE: func(_ *cobra.Command, _ []string) error {
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", wc.Exec)
			} else {
				cmd = exec.Command("sh", "-c", wc.Exec)
			}
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

/*
newCompatCmd returns the hidden "__compat" sub-command present on every root
command built with the Commodore SDK.
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
