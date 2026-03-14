package reactor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// TiltReactor executes Tilt-based blueprints.
type TiltReactor struct{}

// NewTiltReactor creates a TiltReactor.
func NewTiltReactor() *TiltReactor {
	return &TiltReactor{}
}

func (reactor *TiltReactor) StartNode(
	executionContext context.Context,
	node domain.OrchestrationNode,
	environmentName string,
	effectiveEnvironment domain.EnvironmentDefinition,
) error {
	_ = effectiveEnvironment
	tiltfilePath, workingDirectory, errorValue := resolveTiltBlueprint(node, environmentName)
	if errorValue != nil {
		return errorValue
	}

	command := exec.CommandContext(executionContext, "tilt", "ci", "--file", tiltfilePath)
	command.Dir = workingDirectory
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if runError := command.Run(); runError != nil {
		return &ports.ReactorExecutionError{
			Message: fmt.Sprintf("tilt start failed for node %q", node.GetID()),
			Cause:   runError,
		}
	}

	return nil
}

func (reactor *TiltReactor) StopNode(
	executionContext context.Context,
	node domain.OrchestrationNode,
	environmentName string,
) error {
	tiltfilePath, workingDirectory, errorValue := resolveTiltBlueprint(node, environmentName)
	if errorValue != nil {
		return errorValue
	}

	command := exec.CommandContext(executionContext, "tilt", "down", "--file", tiltfilePath)
	command.Dir = workingDirectory
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if runError := command.Run(); runError != nil {
		return &ports.ReactorExecutionError{
			Message: fmt.Sprintf("tilt stop failed for node %q", node.GetID()),
			Cause:   runError,
		}
	}

	return nil
}

func (reactor *TiltReactor) CheckNodeHealth(
	executionContext context.Context,
	node domain.OrchestrationNode,
) error {
	_ = executionContext
	_ = node
	return nil
}

func (reactor *TiltReactor) ResolveStartupPlan(node domain.OrchestrationNode) ([][]domain.OrchestrationNode, error) {
	descendants := node.CollectDescendantNodes()
	if len(descendants) == 0 {
		return [][]domain.OrchestrationNode{{node}}, nil
	}

	graph, graphError := domain.BuildDependencyGraph(descendants)
	if graphError != nil {
		return nil, graphError
	}

	return graph.ResolveStartupWaves()
}

func (reactor *TiltReactor) RouteSignalToSubordinate(
	executionContext context.Context,
	node domain.OrchestrationNode,
	signal ports.SignalRequest,
) ([]ports.SignalResult, error) {
	_ = executionContext
	_ = node
	_ = signal
	return nil, &ports.ReactorExecutionError{Message: "tilt reactor does not route signals directly"}
}

// ResolveTiltfilePath implements ports.TiltfileProvider so the fleet
// orchestrator can collect tiltfile paths to build a combined Tiltfile.
func (reactor *TiltReactor) ResolveTiltfilePath(node domain.OrchestrationNode, environmentName string) (string, string, error) {
	return resolveTiltBlueprint(node, environmentName)
}

func resolveTiltBlueprint(node domain.OrchestrationNode, environmentName string) (string, string, error) {
	workingDirectory := node.GetPath()

	var reactorConfiguration domain.ReactorDefinition
	switch typedNode := node.(type) {
	case *domain.SquadronNode:
		reactorConfiguration = typedNode.Reactor
	case *domain.UnitNode:
		reactorConfiguration = typedNode.Reactor
	default:
		return "", "", &ports.ReactorExecutionError{Message: fmt.Sprintf("node %q does not support tilt blueprints", node.GetID())}
	}

	for _, blueprint := range reactorConfiguration.Blueprints {
		if blueprint.EnvironmentName != environmentName {
			continue
		}
		if blueprint.Path == "" {
			continue
		}

		tiltfilePath := blueprint.Path
		if !filepath.IsAbs(tiltfilePath) {
			tiltfilePath = filepath.Join(workingDirectory, blueprint.Path)
		}
		return tiltfilePath, workingDirectory, nil
	}

	return "", "", &ports.ReactorExecutionError{Message: fmt.Sprintf("tilt blueprint for env %q is missing path in node %q", environmentName, node.GetID())}
}
