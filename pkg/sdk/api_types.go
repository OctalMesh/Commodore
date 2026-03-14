package sdk

import (
	"fmt"
	"os"
	"os/exec"
	stdruntime "runtime"

	"github.com/OctalMesh/Commodore/internal/core/logging"
)

// Options configures Commander initialization and execution behavior.
type Options struct {
	// Version sets the root command version shown by Cobra.
	Version string
	// Binary overrides the binary name from configuration.
	Binary string
	// ConfigurationPath explicitly points to a squadron/unit configuration file.
	ConfigurationPath string
	// WorkingDirectory overrides runtime discovery start path.
	WorkingDirectory string
	// Logger sets the logger implementation used by Context.Log and Commander.Logger.
	Logger Logger
}

// Logger is the public logging contract used by the Commander facade.
type Logger = logging.Logger

// Context is passed to custom command handlers.
type Context struct {
	// WorkingDirectory is the discovered execution root for this Commander.
	WorkingDirectory string
	logger           Logger
}

// Log returns the configured logger for this command context.
func (commandContext Context) Log() Logger {
	return commandContext.logger
}

// Shell executes command in a shell in the discovered working directory.
func (commandContext Context) Shell(command string) error {
	var shellCommand *exec.Cmd
	if stdruntime.GOOS == "windows" {
		shellCommand = exec.Command("cmd", "/c", command)
	} else {
		shellCommand = exec.Command("sh", "-c", command)
	}

	shellCommand.Dir = commandContext.WorkingDirectory
	shellCommand.Stdin = os.Stdin
	shellCommand.Stdout = os.Stdout
	shellCommand.Stderr = os.Stderr

	if errorValue := shellCommand.Run(); errorValue != nil {
		return fmt.Errorf("executing shell command %q: %w", command, errorValue)
	}

	return nil
}

// Command describes a custom command attached to the generated CLI.
type Command struct {
	// Use defines the command invocation name.
	Use string
	// Short defines the command summary shown in help output.
	Short string
	// RunE executes command logic and returns an error on failure.
	RunE func(context Context) error
}
