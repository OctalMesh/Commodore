package ports

import (
	"context"
	"fmt"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/logging"
)

// ReactorExecutionError classifies reactor-level runtime failures.
type ReactorExecutionError struct {
	Message string
	Cause   error
}

func (errorValue *ReactorExecutionError) Error() string {
	if errorValue == nil {
		return "reactor execution error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *ReactorExecutionError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// ConfigLoadError classifies YAML loader failures.
type ConfigLoadError struct {
	Message string
	Cause   error
}

func (errorValue *ConfigLoadError) Error() string {
	if errorValue == nil {
		return "configuration load error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *ConfigLoadError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// Logger is the outbound logging contract for orchestration services.
type Logger = logging.Logger

// NodeConfigLoader loads squadron/unit YAML configuration files.
type NodeConfigLoader interface {
	LoadNodeConfiguration(configurationPath string) (domain.NodeConfiguration, error)
}

// SignalRequest carries a delegated command.
type SignalRequest struct {
	EnvironmentName     string
	TargetSubordinateID string
	CommandName         string
	Arguments           []string
}

// SignalResult captures one delegated execution outcome.
type SignalResult struct {
	TargetNodeID string
	Succeeded    bool
	Error        error
}

// TiltfileProvider can be implemented by a Reactor that knows its tiltfile path.
// The cobracli adapter uses this to build a combined fleet Tiltfile.
type TiltfileProvider interface {
	ResolveTiltfilePath(node domain.OrchestrationNode, environmentName string) (tiltfilePath, workingDir string, err error)
}

// Reactor defines provider-neutral orchestration behavior.
//
// Implementations include TiltReactor and NativeReactor in adapters.
type Reactor interface {
	StartNode(
		executionContext context.Context,
		node domain.OrchestrationNode,
		environmentName string,
		effectiveEnvironment domain.EnvironmentDefinition,
	) error

	StopNode(
		executionContext context.Context,
		node domain.OrchestrationNode,
		environmentName string,
	) error

	CheckNodeHealth(
		executionContext context.Context,
		node domain.OrchestrationNode,
	) error

	ResolveStartupPlan(
		node domain.OrchestrationNode,
	) ([][]domain.OrchestrationNode, error)

	RouteSignalToSubordinate(
		executionContext context.Context,
		node domain.OrchestrationNode,
		signal SignalRequest,
	) ([]SignalResult, error)
}
