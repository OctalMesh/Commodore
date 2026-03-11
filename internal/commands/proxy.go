package commands

import (
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/service"
)

/*
NewProxyCmd builds a Cobra command group for a brigadier (module proxy).
Each brigadier gets a named top-level command (e.g. "console") with "up" and
"down" sub-commands that delegate transparently to the module's own binary.
*/
func NewProxyCmd(b domain.Brigadier, svc *service.ProxyService) *cobra.Command {
	cliPath := b.Path + "/cli"

	cmd := &cobra.Command{
		Use:   b.ID,
		Short: "Manage the " + b.ID + " module",
		Long: "Delegates transparently to the standalone \"" + b.Binary + `" binary (` + cliPath + `/).

Resolution order:
  1. ` + b.Binary + ` found in PATH  (go install ./` + cliPath + `)
  2. ` + b.Binary + `[.exe] found in $GOPATH/bin or ~/go/bin
  3. ` + b.Binary + `[.exe] built locally in ` + cliPath + `/
  4. go run ` + cliPath + `  <- zero-setup fallback, always works`,
	}

	for _, sub := range []string{"up", "down"} {
		sub := sub
		cmd.AddCommand(&cobra.Command{
			Use:                sub,
			Short:              b.Binary + " " + sub,
			DisableFlagParsing: true,
			RunE: func(_ *cobra.Command, args []string) error {
				return svc.Exec(b, append([]string{sub}, args...))
			},
		})
	}

	return cmd
}
