package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
	"github.com/OctalMesh/Commodore/internal/core/runtime/actions"
)

func (service *Service) VerifyNodeHealth(runtime *actions.RuntimeNode, _ string) error {
	reactorAdapter := service.SelectReactor(runtime.Node)
	if reactorAdapter == nil {
		return nil
	}
	if runtime.HealthCheck == nil {
		return reactorAdapter.CheckNodeHealth(context.Background(), runtime.Node)
	}

	testCommand := append([]string(nil), runtime.HealthCheck.Test...)
	if len(testCommand) == 0 {
		return &domain.ValidationError{Message: fmt.Sprintf("healthcheck for node %q has empty test", runtime.Node.GetID())}
	}

	retries := runtime.HealthCheck.Retries
	if retries <= 0 {
		retries = 1
	}
	timeout := runtime.HealthCheck.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	interval := runtime.HealthCheck.Interval
	if interval <= 0 {
		interval = time.Second
	}

	var lastError error
	for attempt := 1; attempt <= retries; attempt++ {
		runContext, cancel := context.WithTimeout(context.Background(), timeout)
		request := ports.ProcessRunRequest{Command: testCommand[0], Args: testCommand[1:], Dir: runtime.Node.GetPath()}
		lastError = service.options.ProcessRunner.Run(runContext, request)
		cancel()
		if lastError == nil {
			service.logger.Info("healthcheck passed for %s (attempt %d/%d)", runtime.Node.GetID(), attempt, retries)
			return nil
		}
		service.logger.Info("healthcheck attempt %d/%d failed for %s", attempt, retries, runtime.Node.GetID())
		if attempt < retries {
			time.Sleep(interval)
		}
	}

	return &ports.ReactorExecutionError{Message: fmt.Sprintf("healthcheck failed for node %q after %d attempts", runtime.Node.GetID(), retries), Cause: lastError}
}
