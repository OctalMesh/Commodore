package sdk

import (
	"fmt"
	"os"

	"github.com/OctalMesh/Commodore/internal/adapters/cli/cobracli"
	"github.com/OctalMesh/Commodore/internal/adapters/config"
	"github.com/OctalMesh/Commodore/internal/adapters/process"
	"github.com/OctalMesh/Commodore/internal/adapters/reactor"
	coreruntime "github.com/OctalMesh/Commodore/internal/core/runtime"
)

const compatibilityMarker = "commodore:sdk:github.com/OctalMesh/Commodore"

// Commander is the primary SDK facade that bootstraps a Commodore command tree.
type Commander struct {
	options         Options
	logger          Logger
	runtimeService  *coreruntime.Service
	commandAdapter  *cobracli.Adapter
	pendingCommands []*Command
}

// NewCommander creates a Commander with lazy configuration discovery.
func NewCommander(options Options) *Commander {
	logger := options.Logger
	if logger == nil {
		logger = NewTerminalLogger()
	}

	runtimeService := coreruntime.NewService(coreruntime.Options{
		ConfigurationPath: options.ConfigurationPath,
		WorkingDirectory:  options.WorkingDirectory,
		Logger:            logger,
		NodeConfigLoader:  config.NewNodeLoader(),
		TiltReactor:       reactor.NewTiltReactor(),
		NativeReactor:     reactor.NewNativeReactor(),
		ProcessRunner:     process.NewExecRunner(),
	})

	return &Commander{
		options:        options,
		logger:         logger,
		runtimeService: runtimeService,
	}
}

// AddCommand registers a custom command to be attached during Execute.
func (commander *Commander) AddCommand(command *Command) {
	if command == nil {
		return
	}

	commander.pendingCommands = append(commander.pendingCommands, command)
}

// Logger returns the Commander logger.
func (commander *Commander) Logger() Logger {
	return commander.logger
}

// Execute discovers configuration, builds the command tree, and executes it.
func (commander *Commander) Execute() error {
	if len(os.Args) == 2 && os.Args[1] == "__compat" {
		fmt.Println(compatibilityMarker)
		return nil
	}

	arguments := os.Args[1:]
	configurationPath := parseConfigurationFlag(arguments)
	if configurationPath != "" {
		commander.runtimeService.SetConfigurationPath(configurationPath)
	}

	needsRuntime := requiresRuntimeInitialization(arguments)
	if needsRuntime {
		if initializeError := commander.runtimeService.Initialize(); initializeError != nil {
			return initializeError
		}
	}

	binary := commander.options.Binary
	if binary == "" && needsRuntime {
		binary = commander.runtimeService.RootNodeID()
	}
	if binary == "" {
		binary = "commodore"
	}

	if commander.commandAdapter == nil {
		commander.commandAdapter = cobracli.NewAdapter(binary, commander.options.Version, commander.runtimeService)
		for _, command := range commander.pendingCommands {
			customCommand := command
			commander.commandAdapter.AddExternalCommand(cobracli.ExternalCommand{
				Use:   customCommand.Use,
				Short: customCommand.Short,
				RunE: func() error {
					if customCommand.RunE == nil {
						return nil
					}
					return customCommand.RunE(Context{
						WorkingDirectory: commander.runtimeService.RuntimeWorkingDirectory(),
						logger:           commander.logger,
					})
				},
			})
		}
	}

	return commander.commandAdapter.Execute()
}

func parseConfigurationFlag(arguments []string) string {
	for index := range arguments {
		argument := arguments[index]
		if (argument == "--config" || argument == "-c") && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

func requiresRuntimeInitialization(arguments []string) bool {
	if len(arguments) == 0 {
		return false
	}

	for _, argument := range arguments {
		switch argument {
		case "-h", "--help", "help", "-v", "--version", "version", "completion", "__complete", "__completeNoDesc":
			return false
		}
	}

	return true
}
