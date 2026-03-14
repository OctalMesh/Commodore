package commands

import (
	"github.com/spf13/cobra"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	coreruntime "github.com/OctalMesh/Commodore/internal/core/runtime"
)

// Provider exposes runtime access for command execution.
type Provider interface {
	RuntimeService() *coreruntime.Service
	OperationOptions() coreruntime.OperationOptions
}

// Spec declares one command and its optional subcommands.
type Spec struct {
	Use       string
	Short     string
	Long      string
	Args      cobra.PositionalArgs
	Configure func(command *cobra.Command)
	RunE      func(provider Provider, arguments []string) error
	Children  []Spec
}

// Build builds one Cobra command tree from a spec.
func Build(provider Provider, spec Spec) *cobra.Command {
	command := &cobra.Command{
		Use:   spec.Use,
		Short: spec.Short,
		Long:  spec.Long,
		Args:  spec.Args,
		RunE: func(_ *cobra.Command, arguments []string) error {
			if spec.RunE == nil {
				return nil
			}
			return spec.RunE(provider, arguments)
		},
	}

	if spec.Configure != nil {
		spec.Configure(command)
	}

	for _, child := range spec.Children {
		command.AddCommand(Build(provider, child))
	}

	return command
}

// Core returns built-in top-level commands.
func Core() []Spec {
	return []Spec{Up(), Down(), Doctor(), Status(), Tree(), Signal(), Modules()}
}

// Maneuvers maps runtime maneuvers to top-level commands.
func Maneuvers(definitions []domain.ManeuverDefinition) []Spec {
	result := make([]Spec, 0, len(definitions))
	for _, definition := range definitions {
		maneuver := definition
		result = append(result, Spec{
			Use:   maneuver.Call,
			Short: maneuver.Description,
			RunE: func(provider Provider, _ []string) error {
				return provider.RuntimeService().Maneuver(provider.OperationOptions(), maneuver.Call)
			},
		})
	}

	return result
}
