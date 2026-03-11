/*
Package foreman provides the factory for the Foreman CLI command tree.

The Foreman is the root CLI (ow / octalweb) that manages the full
OctalMesh platform. It reads the brigadiers list from domain.Config and
wires up both proxy shortcut commands and the "hire" delegation command
for each module alongside the built-in up / doctor / modules commands.
*/
package foreman

import (
	"fmt"
	"os"
	"strings"

	"github.com/OctalMesh/Commodore/internal/adapters/compat"
	"github.com/OctalMesh/Commodore/internal/adapters/resolver"
	"github.com/OctalMesh/Commodore/internal/adapters/rootdir"
	"github.com/OctalMesh/Commodore/internal/adapters/runner"
	"github.com/OctalMesh/Commodore/internal/adapters/submodule"
	"github.com/OctalMesh/Commodore/internal/adapters/toolcheck"
	"github.com/OctalMesh/Commodore/internal/app"
	"github.com/OctalMesh/Commodore/internal/commands"
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
	"github.com/spf13/cobra"
)

/*
usageTemplate is a custom Cobra usage template that renders brigadier proxy
commands in a dedicated "Modules" section, separate from built-in commands.
 */
const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Commands:{{range .Commands}}{{if and (not .Hidden) (not (index .Annotations "brigadier"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Modules:{{range .Commands}}{{if and (not .Hidden) (index .Annotations "brigadier")}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

/*
New builds and returns the fully-wired Cobra command tree for the Foreman CLI.

cfg is the parsed .commodore configuration (role: foreman).
version is the build version string injected at link time.
extra are any additional *cobra.Command instances the caller wants to register
(e.g. custom commands not provided by the SDK).
 */
func New(cfg domain.Config, version string, extra ...*cobra.Command) *app.App {
	finder := rootdir.New()
	checker := toolcheck.New()
	sub := submodule.New()
	res := resolver.New()
	run := runner.New()
	cmp := compat.New()

	// Services
	svcDoctor := service.NewDoctorService(checker)
	svcModules := service.NewModulesService(sub, finder)
	svcProxy := service.NewProxyService(finder, res, run, cmp)

	// Root command
	binary := cfg.Binary
	if binary == "" {
		binary = "ow"
	}

	root := &cobra.Command{
		Use:           binary,
		Aliases:       cfg.Aliases,
		Short:         "OctalWeb platform management CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.SetUsageTemplate(usageTemplate)

	root.AddCommand(commands.NewUpCmd(cfg.ID, cfg, finder))
	root.AddCommand(commands.NewDownCmd(cfg.ID, cfg, finder))
	root.AddCommand(commands.NewDoctorCmd(svcDoctor))
	root.AddCommand(commands.NewModulesCmd(svcModules))
	root.AddCommand(commands.NewHireBrigadierCmd(cfg.Brigadiers, svcProxy))

	for _, b := range cfg.Brigadiers {
		cmd := commands.NewProxyCmd(b, svcProxy)
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations["brigadier"] = "true"
		root.AddCommand(cmd)
	}

	for _, cmd := range extra {
		root.AddCommand(cmd)
	}

	root.AddCommand(newCompatCmd())

	root.SetErrPrefix(fmt.Sprintf("  %s  ", "\x1b[31mx\x1b[0m"))
	root.SetOut(os.Stdout)

	// Register cobra template functions used in usageTemplate.
	// Accepts bool OR string (index .Annotations returns string, .Hidden returns bool).
	cobra.AddTemplateFunc("not", func(v interface{}) bool {
		if v == nil {
			return true
		}
		if b, ok := v.(bool); ok {
			return !b
		}
		if s, ok := v.(string); ok {
			return s == ""
		}
		return false
	})
	_ = strings.TrimRight // ensure strings is used

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
