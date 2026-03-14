package ports

import "context"

// ProcessRunRequest describes one external process execution.
type ProcessRunRequest struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
}

// ProcessRunner executes external processes for orchestration actions.
type ProcessRunner interface {
	Run(executionContext context.Context, request ProcessRunRequest) error
}
