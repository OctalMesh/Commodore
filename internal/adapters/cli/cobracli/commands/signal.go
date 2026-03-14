package commands

import "github.com/spf13/cobra"

// Signal delegates command to subordinate or broadcasts to direct subordinates.
func Signal() Spec {
	return Spec{
		Use:   "signal [subordinate-id] [command-or-maneuver] [args...]",
		Short: "Delegate command to subordinate or broadcast to direct subordinates",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(provider Provider, arguments []string) error {
			return provider.RuntimeService().Signal(provider.OperationOptions(), arguments)
		},
	}
}
