package actions

import (
	"fmt"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// Maneuver executes a named maneuver on the runtime root node.
func Maneuver(engine Engine, options OperationOptions, maneuverName string) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	normalized := engine.NormalizeOptions(options)
	runtimeRoot := engine.Root()
	action, errorValue := runtimeRoot.Node.ExecuteManeuverByName(maneuverName)
	if errorValue != nil {
		return errorValue
	}
	if len(action) == 0 {
		return &domain.ValidationError{Message: fmt.Sprintf("maneuver %q has empty action", maneuverName)}
	}

	effectiveEnvironments, errorValue := engine.BuildEffectiveEnvironments(runtimeRoot, normalized.Environment)
	if errorValue != nil {
		return errorValue
	}

	request := ports.ProcessRunRequest{Command: action[0], Args: action[1:], Dir: runtimeRoot.Node.GetPath(), Env: effectiveEnvironments[runtimeRoot.Node.GetID()].Variables}
	if runError := engine.RunProcess(request); runError != nil {
		return fmt.Errorf("maneuver %q failed for node %q: %w", maneuverName, runtimeRoot.Node.GetID(), runError)
	}

	return nil
}
