package actions

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// Signal routes an action to one subordinate or broadcasts to direct subordinates.
func Signal(engine Engine, options OperationOptions, arguments []string) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}
	if len(arguments) == 0 {
		return &domain.SignalRoutingError{Message: "signal requires command arguments"}
	}

	normalized := engine.NormalizeOptions(options)
	runtimeRoot := engine.Root()
	targetByID := make(map[string]*RuntimeNode, len(runtimeRoot.Children)*2)
	for _, childRuntime := range runtimeRoot.Children {
		targetByID[childRuntime.Node.GetID()] = childRuntime
		targetByID[childRuntime.LocalID] = childRuntime
	}

	var targets []*RuntimeNode
	var commandName string
	var commandArguments []string
	if targetedRuntime, exists := targetByID[arguments[0]]; exists {
		if len(arguments) < 2 {
			return &domain.SignalRoutingError{Message: fmt.Sprintf("missing delegated command for subordinate %q", arguments[0])}
		}

		targets = []*RuntimeNode{targetedRuntime}
		commandName = arguments[1]
		commandArguments = arguments[2:]
	} else {
		if len(runtimeRoot.Children) == 0 {
			return &domain.SignalRoutingError{Message: fmt.Sprintf("node %q has no direct subordinates", runtimeRoot.Node.GetID())}
		}

		targets = append(targets, runtimeRoot.Children...)
		commandName = arguments[0]
		commandArguments = arguments[1:]
	}

	errorsList := make([]string, 0)
	for _, targetRuntime := range targets {
		if !engine.ShouldOperateOnNode(targetRuntime.Node, normalized.Tags) {
			continue
		}
		if errorValue := executeSubordinateCommand(engine, targetRuntime, normalized, commandName, commandArguments); errorValue != nil {
			errorsList = append(errorsList, fmt.Sprintf("%s: %v", targetRuntime.Node.GetID(), errorValue))
		}
	}
	if len(errorsList) > 0 {
		return &domain.SignalRoutingError{Message: strings.Join(errorsList, "; ")}
	}

	return nil
}

func executeSubordinateCommand(
	engine Engine,
	runtime *RuntimeNode,
	options OperationOptions,
	commandName string,
	arguments []string,
) error {
	runnableBinary, errorValue := resolveRunnableBinary(runtime.Binary)
	if errorValue != nil {
		return &domain.SignalRoutingError{Message: fmt.Sprintf("resolving runnable binary for subordinate %q", runtime.Node.GetID()), Cause: errorValue}
	}
	if runnableBinary != runtime.Binary {
		engine.Warn("subordinate %s binary %q was not found; using current executable %q", runtime.Node.GetID(), runtime.Binary, runnableBinary)
	}

	processArguments := []string{commandName}
	if options.Environment != "" {
		processArguments = append(processArguments, "--env", options.Environment)
	}
	if len(options.Tags) > 0 {
		processArguments = append(processArguments, "--tags", strings.Join(options.Tags, ","))
	}
	processArguments = append(processArguments, arguments...)

	return engine.RunProcess(ports.ProcessRunRequest{
		Command: runnableBinary,
		Args:    processArguments,
		Dir:     runtime.Node.GetPath(),
	})
}

func resolveRunnableBinary(configuredBinary string) (string, error) {
	if configuredBinary != "" {
		if resolvedBinary, errorValue := exec.LookPath(configuredBinary); errorValue == nil {
			return resolvedBinary, nil
		}
	}

	if executablePath, errorValue := os.Executable(); errorValue == nil && executablePath != "" {
		return executablePath, nil
	}

	if len(os.Args) > 0 && os.Args[0] != "" {
		if resolvedBinary, errorValue := exec.LookPath(os.Args[0]); errorValue == nil {
			return resolvedBinary, nil
		}
		return os.Args[0], nil
	}

	return "", fmt.Errorf("unable to resolve current executable")
}
