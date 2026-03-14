package cobracli

import (
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/adapters/cli/cobracli/commands"
	coreruntime "github.com/OctalMesh/Commodore/internal/core/runtime"
)

// ExternalCommand describes one custom root command extension.
type ExternalCommand struct {
	Use   string
	Short string
	RunE  func() error
}

// Adapter builds Cobra command surfaces over the runtime service API.
type Adapter struct {
	root        *cobra.Command
	runtime     *coreruntime.Service
	configPath  string
	environment string
	tags        []string
}

// NewAdapter creates a Cobra adapter over runtime service capabilities.
func NewAdapter(binary string, version string, runtime *coreruntime.Service) *Adapter {
	adapter := &Adapter{runtime: runtime, environment: "dev"}

	adapter.root = &cobra.Command{
		Use:           binary,
		Short:         "Commodore orchestration CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	adapter.root.PersistentFlags().StringVarP(&adapter.environment, "env", "e", "dev", "Execution environment name (dev, staging, ...)")
	adapter.root.PersistentFlags().StringVarP(&adapter.configPath, "config", "c", "", "Path to Commodore configuration file")
	adapter.root.PersistentFlags().StringSliceVarP(&adapter.tags, "tags", "t", nil, "Filter operations by tags")

	for _, spec := range commands.Core() {
		adapter.root.AddCommand(commands.Build(adapter, spec))
	}

	for _, spec := range commands.Maneuvers(runtime.RootManeuvers()) {
		adapter.root.AddCommand(commands.Build(adapter, spec))
	}

	return adapter
}

// AddExternalCommand appends a custom command at root level.
func (adapter *Adapter) AddExternalCommand(command ExternalCommand) {
	adapter.root.AddCommand(commands.Build(adapter, commands.Spec{
		Use:   command.Use,
		Short: command.Short,
		RunE: func(_ commands.Provider, _ []string) error {
			if command.RunE == nil {
				return nil
			}
			return command.RunE()
		},
	}))
}

// Execute runs the root Cobra command.
func (adapter *Adapter) Execute() error {
	return adapter.root.Execute()
}

func (adapter *Adapter) operationOptions() coreruntime.OperationOptions {
	return coreruntime.OperationOptions{Environment: adapter.environment, Tags: append([]string(nil), adapter.tags...)}
}

// RuntimeService exposes runtime service to the command provider interface.
func (adapter *Adapter) RuntimeService() *coreruntime.Service {
	return adapter.runtime
}

// OperationOptions exposes current operation options to the command provider interface.
func (adapter *Adapter) OperationOptions() coreruntime.OperationOptions {
	return adapter.operationOptions()
}
