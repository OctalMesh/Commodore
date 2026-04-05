package reactor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// NativeReactor executes inline command actions from native blueprints.
type NativeReactor struct{}

// NewNativeReactor creates a NativeReactor.
func NewNativeReactor() *NativeReactor {
	return &NativeReactor{}
}

func (reactor *NativeReactor) StartNode(
	executionContext context.Context,
	node domain.OrchestrationNode,
	environmentName string,
	effectiveEnvironment domain.EnvironmentDefinition,
) error {
	action, workingDirectory, blueprintSource, errorValue := resolveNativeAction(node, environmentName)
	if errorValue != nil {
		return errorValue
	}

	_, _ = fmt.Fprintf(os.Stdout, "  info: selected %s reactor blueprint for node %q (env=%s)\n", blueprintSource, node.GetID(), environmentName)

	environmentVariables, errorValue := buildEnvironmentVariables(workingDirectory, effectiveEnvironment)
	if errorValue != nil {
		return errorValue
	}

	command := exec.CommandContext(executionContext, action[0], action[1:]...)
	command.Dir = workingDirectory
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = append(os.Environ(), environmentVariables...)

	if runError := command.Run(); runError != nil {
		return &ports.ReactorExecutionError{
			Message: fmt.Sprintf("native start failed for node %q", node.GetID()),
			Cause:   runError,
		}
	}

	return nil
}

func (reactor *NativeReactor) StopNode(
	executionContext context.Context,
	node domain.OrchestrationNode,
	environmentName string,
) error {
	_ = executionContext
	_ = node
	_ = environmentName
	return nil
}

func (reactor *NativeReactor) CheckNodeHealth(
	executionContext context.Context,
	node domain.OrchestrationNode,
) error {
	_ = executionContext
	_ = node
	return nil
}

func (reactor *NativeReactor) ResolveStartupPlan(node domain.OrchestrationNode) ([][]domain.OrchestrationNode, error) {
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

func (reactor *NativeReactor) RouteSignalToSubordinate(
	executionContext context.Context,
	node domain.OrchestrationNode,
	signal ports.SignalRequest,
) ([]ports.SignalResult, error) {
	_ = executionContext
	_ = node
	_ = signal
	return nil, &ports.ReactorExecutionError{Message: "native reactor does not route signals directly"}
}

func resolveNativeAction(node domain.OrchestrationNode, environmentName string) ([]string, string, domain.BlueprintSelectionSource, error) {
	unitNode, ok := node.(*domain.UnitNode)
	if !ok {
		return nil, "", domain.BlueprintSelectionSourceDefault, &ports.ReactorExecutionError{Message: fmt.Sprintf("node %q is not a unit for native execution", node.GetID())}
	}

	workingDirectory := unitNode.Path
	primaryBlueprints, fallbackBlueprints, primarySource := unitNode.ResolveBlueprintListsForExecution()

	action, found := findNativeBlueprintAction(primaryBlueprints, environmentName)
	if found {
		return action, workingDirectory, primarySource, nil
	}

	action, found = findNativeBlueprintAction(fallbackBlueprints, environmentName)
	if found {
		return action, workingDirectory, domain.BlueprintSelectionSourceDefault, nil
	}

	return nil, "", domain.BlueprintSelectionSourceDefault, &ports.ReactorExecutionError{Message: fmt.Sprintf("native blueprint for env %q is missing action in node %q", environmentName, node.GetID())}
}

func findNativeBlueprintAction(
	blueprints []domain.ReactorBlueprint,
	environmentName string,
) ([]string, bool) {
	for _, blueprint := range blueprints {
		if blueprint.EnvironmentName != environmentName || len(blueprint.Action) == 0 {
			continue
		}

		return append([]string(nil), blueprint.Action...), true
	}

	return nil, false
}

func buildEnvironmentVariables(workingDirectory string, effectiveEnvironment domain.EnvironmentDefinition) ([]string, error) {
	environmentVariables := make([]string, 0, len(effectiveEnvironment.Variables))
	environmentVariables = append(environmentVariables, effectiveEnvironment.Variables...)

	for _, environmentFile := range effectiveEnvironment.Files {
		filePath := environmentFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(workingDirectory, environmentFile)
		}

		parsedVariables, parseError := parseEnvironmentFile(filePath)
		if parseError != nil {
			return nil, parseError
		}

		environmentVariables = append(environmentVariables, parsedVariables...)
	}

	return environmentVariables, nil
}

func parseEnvironmentFile(filePath string) (variables []string, err error) {
	fileHandle, openError := os.Open(filePath)
	if openError != nil {
		if os.IsNotExist(openError) {
			return nil, nil
		}
		return nil, &ports.ReactorExecutionError{Message: fmt.Sprintf("opening env file %s", filePath), Cause: openError}
	}
	defer func() {
		if closeError := fileHandle.Close(); closeError != nil && err == nil {
			err = &ports.ReactorExecutionError{Message: fmt.Sprintf("closing env file %s", filePath), Cause: closeError}
		}
	}()

	variables = make([]string, 0)
	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "=") {
			variables = append(variables, line)
		}
	}

	if scanError := scanner.Err(); scanError != nil {
		return nil, &ports.ReactorExecutionError{Message: fmt.Sprintf("reading env file %s", filePath), Cause: scanError}
	}

	return variables, nil
}
