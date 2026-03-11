package commands

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
	"github.com/OctalMesh/Commodore/internal/ui/styles"
)

// NewModulesCmd returns a ready-to-use `modules` command backed by svc.
func NewModulesCmd(svc *service.ModulesService) *cobra.Command {
	root := &cobra.Command{
		Use:   "modules",
		Short: "Manage git submodules (status, init, update, sync)",
		Long: `modules - git submodule management.

Provides a convenient wrapper around git submodule operations for
the OctalWeb monorepo.

Sub-commands:
  status   Show the current state of all submodules
  init     Initialize / clone all available submodules
  update   Pull latest commits from each submodule's tracking branch
  sync     Synchronize remote URLs from .gitmodules`,
	}

	root.AddCommand(newModulesStatusCmd(svc))
	root.AddCommand(newModulesInitCmd(svc))
	root.AddCommand(newModulesUpdateCmd(svc))
	root.AddCommand(newModulesSyncCmd(svc))

	return root
}

func newModulesStatusCmd(svc *service.ModulesService) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current state of all git submodules",
		RunE: func(_ *cobra.Command, _ []string) error {
			mods, err := svc.Status()
			if err != nil {
				return err
			}

			printHeader("Submodule Status")

			if len(mods) == 0 {
				printBlank()
				printWarn("No submodules found.")
				printInfo("Run this command from inside the OctalWeb repository root.")
				printBlank()
				return nil
			}

			cols := []ColDef{
				{Label: "NAME", Width: 12},
				{Label: "PATH", Width: 24},
				{Label: "COMMIT", Width: 10},
				{Label: "STATE", Width: 10, Paint: paintSubState},
				{Label: "BRANCH", Width: 18},
			}

			rows := make([][]string, len(mods))
			for i, m := range mods {
				rows[i] = []string{
					m.Name,
					m.Path,
					m.Commit,
					string(m.State),
					strings.Trim(m.Detail, "()"),
				}
			}

			printBlank()
			PrintTable(cols, rows)
			printBlank()

			for _, m := range mods {
				if m.State == domain.SubmoduleMissing {
					printInfo(`Modules marked "missing" have not been initialized.`)
					printInfo("Run: ow modules init")
					printBlank()
					break
				}
			}
			return nil
		},
	}
}

func newModulesInitCmd(svc *service.ModulesService) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize / clone all available git submodules",
		Long: `modules init

Runs:  git submodule update --init --recursive`,
		RunE: func(_ *cobra.Command, _ []string) error {
			printBlank()
			printStep(stFg.Render("git submodule update --init --recursive"))
			printBlank()

			if err := svc.Init(); err != nil {
				printBlank()
				printWarn("Some submodules could not be initialized.")
				printInfo("Private modules require separate repository access.")
				printBlank()
				return nil
			}

			printBlank()
			printOk("Submodules initialized.")
			printBlank()
			return nil
		},
	}
}

func newModulesUpdateCmd(svc *service.ModulesService) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Pull latest commits from each submodule's tracking branch",
		Long: `modules update

Runs:  git submodule update --remote --merge`,
		RunE: func(_ *cobra.Command, _ []string) error {
			printBlank()
			printStep(stFg.Render("git submodule update --remote --merge"))
			printBlank()

			if err := svc.Update(); err != nil {
				printBlank()
				printWarn("Update encountered errors (see git output above).")
				printBlank()
				return nil
			}

			printBlank()
			printOk("Submodules updated.")
			printBlank()
			return nil
		},
	}
}

func newModulesSyncCmd(svc *service.ModulesService) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync submodule remote URLs from .gitmodules",
		Long: `modules sync

Runs:  git submodule sync --recursive`,
		RunE: func(_ *cobra.Command, _ []string) error {
			printBlank()
			printStep(stFg.Render("git submodule sync --recursive"))
			printBlank()

			if err := svc.Sync(); err != nil {
				printBlank()
				printWarn("Sync encountered errors (see git output above).")
				printBlank()
				return nil
			}

			printBlank()
			printOk("Submodule URLs synced.")
			printBlank()
			return nil
		},
	}
}

func paintSubState(s string) string {
	switch domain.SubmoduleState(s) {
	case domain.SubmodulePresent:
		return lipgloss.NewStyle().Foreground(styles.Mint).Render(s)
	case domain.SubmoduleMissing:
		return lipgloss.NewStyle().Foreground(styles.Error).Render(s)
	case domain.SubmoduleOutdated:
		return lipgloss.NewStyle().Foreground(styles.Warn).Render(s)
	case domain.SubmoduleConflict:
		return lipgloss.NewStyle().Foreground(styles.Purple).Render(s)
	default:
		return stMuted.Render(s)
	}
}
