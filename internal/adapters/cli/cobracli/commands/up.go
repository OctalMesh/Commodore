package commands

import "github.com/spf13/cobra"

// Up starts node and subordinates using DAG order.
func Up() Spec {
	var hud bool

	return Spec{
		Use:   "up",
		Short: "Start node and subordinates using DAG order",
		Long: `up - start the development environment.

Launches an interactive TUI: spinner during environment checks, then
streams Tilt output into a scrollable viewport with a live status bar.

If the current configuration is a root unit (standalone execution),
reactor.standalone blueprints/environments are resolved before reactor defaults.

Key bindings while running:
  space      open Tilt browser UI (http://localhost:10350)
  t          switch to legacy terminal mode (full Tilt TUI)
  ctrl+c     gracefully stop Tilt and exit
  ↑/↓ j/k   scroll log  |  g / G  top / bottom  |  pgup/pgdn  page
  [ ]        switch tabs when multiple services are running`,
		Configure: func(command *cobra.Command) {
			command.Flags().BoolVar(&hud, "hud", true, "open Tilt's browser HUD (default: true)")
		},
		RunE: func(provider Provider, _ []string) error {
			return runUp(provider.RuntimeService(), provider.OperationOptions(), hud)
		},
	}
}
