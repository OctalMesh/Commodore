/*
Package sdk is the public entry point for building Commodore-based CLI binaries.

All internal adapters, services, and domain types are hidden behind this surface.
*/
package sdk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/adapters/config"
	"github.com/OctalMesh/Commodore/internal/adapters/moduledir"
	"github.com/OctalMesh/Commodore/internal/adapters/rootdir"
	"github.com/OctalMesh/Commodore/internal/app"
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/roles/brigadier"
	"github.com/OctalMesh/Commodore/internal/roles/foreman"
	"github.com/OctalMesh/Commodore/internal/roles/worker"
)

/*
Config holds the compile-time metadata for a CLI binary.
Only ID, Binary, and Version are required; everything else (workers, engine,
brigadiers, ...) is discovered at runtime from the nearest .commodore file.
 */
type Config struct {
	// ID is the module / service identifier, e.g. "console".
	// Must match the id: field in .commodore.
	ID string
	// Binary is the executable name, e.g. "owc".
	Binary string
	// Version is the build version injected via -ldflags.
	Version string
}

/*
Context is passed to custom command Run functions and provides helpers for
running shell commands in the correct working directory.
 */
type Context struct {
	// Root is the absolute path to the project / module root.
	Root string
}

/*
Shell runs cmd in a shell (sh -c on Unix, cmd /c on Windows) with the
process's stdin/stdout/stderr attached and the working directory set to Root.
 */
func (c *Context) Shell(cmd string) error {
	var sh *exec.Cmd
	if runtime.GOOS == "windows" {
		sh = exec.Command("cmd", "/c", cmd)
	} else {
		sh = exec.Command("sh", "-c", cmd)
	}
	sh.Dir = c.Root
	sh.Stdin = os.Stdin
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr
	if err := sh.Run(); err != nil {
		return fmt.Errorf("shell: %w", err)
	}
	return nil
}

// Command is a custom cobra command to attach to the CLI.
type Command struct {
	// Name is the cobra Use string, e.g. "migrate".
	Name string
	// Description is the Short help text.
	Description string
	// Run is the command handler. Return nil on success; any non-nil error is
	// printed and causes a non-zero exit. Return sdk.ErrSilent if the handler
	// already printed its own error message.
	Run func(ctx *Context) error
}

/*
App is a ready-to-run Commodore CLI. Obtain one via NewForeman, NewBrigadier,
or NewWorker; then optionally call AddCommand before calling Run.
 */
type App struct {
	// buildFn is called lazily the first time Run is invoked. It receives all
	// queued extra commands and returns the fully-wired internal App.
	buildFn func(extras ...*cobra.Command) *app.App
	// initErr captures any error from discovery/config-loading so that Run can
	// print it and exit with a non-zero code without panicking.
	initErr error
	// ctx is used by AddCommand to bind Shell() to the discovered root.
	ctx *Context
	// pending custom commands queued via AddCommand.
	pending []*cobra.Command
}

/*
AddCommand registers a custom command that will be added to the CLI's command
tree when Run is called. Must be called before Run.
 */
func (a *App) AddCommand(cmd Command) {
	a.pending = append(a.pending, toCobra(cmd, a.ctx))
}

/*
Run executes the CLI. It handles all errors internally (printing styled
messages and calling os.Exit on failure) and never returns.
 */
func (a *App) Run() {
	// Respond to the compat probe immediately, before any discovery happens.
	// This lets a parent CLI verify SDK compatibility even when the binary is
	// run outside its own module directory.
	if len(os.Args) == 2 && os.Args[1] == "__compat" {
		fmt.Println("commodore:sdk:github.com/OctalMesh/Commodore")
		os.Exit(0)
	}
	if a.initErr != nil {
		fmt.Fprintln(os.Stderr, "  \x1b[31mx\x1b[0m  "+a.initErr.Error())
		os.Exit(1)
	}
	a.buildFn(a.pending...).Run()
}

/*
NewForeman constructs an App for a Foreman role (platform root CLI, e.g. ow).

Discovery: walks up from cwd to the repo root (directory containing modules/
and cli/), then loads .commodore from that root.
Validation: the loaded config must have role: foreman and id matching cfg.ID.
 */
func NewForeman(cfg Config) *App {
	finder := rootdir.New()
	root, err := finder.Find()
	if err != nil {
		return errApp(err)
	}

	dcfg, err := loadAndValidate(root, domain.RoleForeman, cfg.ID)
	if err != nil {
		return errApp(err)
	}

	// Hardcoded binary takes precedence over .commodore.
	if cfg.Binary != "" {
		dcfg.Binary = cfg.Binary
	}

	return &App{
		ctx: &Context{Root: root},
		buildFn: func(extras ...*cobra.Command) *app.App {
			return foreman.New(dcfg, cfg.Version, extras...)
		},
	}
}

/*
NewBrigadier constructs an App for a Brigadier role (module CLI, e.g. owc).

Discovery: walks up from cwd to the module root (directory containing cli/
and at least one of apps/, services/), then loads .commodore from that root.
Validation: the loaded config must have role: brigadier and id matching cfg.ID.
 */
func NewBrigadier(cfg Config) *App {
	finder := moduledir.New(cfg.ID)
	root, err := finder.Find()
	if err != nil {
		return errApp(err)
	}

	dcfg, err := loadAndValidate(root, domain.RoleBrigadier, cfg.ID)
	if err != nil {
		return errApp(err)
	}

	if cfg.Binary != "" {
		dcfg.Binary = cfg.Binary
	}

	return &App{
		ctx: &Context{Root: root},
		buildFn: func(extras ...*cobra.Command) *app.App {
			return brigadier.New(dcfg, cfg.Version, extras...)
		},
	}
}

/*
NewWorker constructs an App for a Worker role.

Discovery: reads .commodore from the current working directory.
Validation: the loaded config must have role: worker and id matching cfg.ID.
 */
func NewWorker(cfg Config) *App {
	cwd, err := os.Getwd()
	if err != nil {
		return errApp(fmt.Errorf("worker: getwd: %w", err))
	}

	dcfg, err := loadAndValidate(cwd, domain.RoleWorker, cfg.ID)
	if err != nil {
		return errApp(err)
	}

	if cfg.Binary != "" {
		dcfg.Binary = cfg.Binary
	}

	return &App{
		ctx: &Context{Root: cwd},
		buildFn: func(extras ...*cobra.Command) *app.App {
			return worker.New(dcfg, cfg.Version, extras...)
		},
	}
}

// errApp returns an App that will print initErr and exit when Run is called.
func errApp(err error) *App {
	return &App{initErr: err, ctx: &Context{}}
}

// loadAndValidate loads .commodore from dir and checks that role and id match.
func loadAndValidate(dir string, role domain.Role, id string) (domain.Config, error) {
	path := filepath.Join(dir, ".commodore")
	dcfg, err := config.New().Load(path)
	if err != nil {
		return domain.Config{}, fmt.Errorf("could not load .commodore at %s: %w", path, err)
	}
	if dcfg.Role != role {
		return domain.Config{}, fmt.Errorf(
			".commodore role mismatch: expected %q, got %q (path: %s)",
			role, dcfg.Role, path,
		)
	}
	if dcfg.ID != id {
		return domain.Config{}, fmt.Errorf(
			".commodore id mismatch: expected %q, got %q (path: %s)",
			id, dcfg.ID, path,
		)
	}
	return dcfg, nil
}

// toCobra converts an sdk.Command into a *cobra.Command.
func toCobra(cmd Command, ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   cmd.Name,
		Short: cmd.Description,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(ctx)
		},
	}
}
