package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// ExecRunner executes processes via os/exec.
type ExecRunner struct{}

// NewExecRunner creates a process runner backed by os/exec.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{}
}

func (runner *ExecRunner) Run(executionContext context.Context, request ports.ProcessRunRequest) error {
	if request.Command == "" {
		return &ports.ReactorExecutionError{Message: "process command cannot be empty"}
	}

	command := exec.CommandContext(executionContext, request.Command, request.Args...)
	command.Dir = request.Dir
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if len(request.Env) > 0 {
		command.Env = append(os.Environ(), request.Env...)
	}

	if runError := command.Run(); runError != nil {
		return fmt.Errorf("running %s: %w", request.Command, runError)
	}

	return nil
}
