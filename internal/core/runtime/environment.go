package runtime

import (
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/runtime/actions"
)

func (service *Service) BuildEffectiveEnvironments(runtime *actions.RuntimeNode, environmentName string) (map[string]domain.EnvironmentDefinition, error) {
	result := make(map[string]domain.EnvironmentDefinition)

	var build func(current *actions.RuntimeNode, inherited domain.EnvironmentDefinition) error
	build = func(current *actions.RuntimeNode, inherited domain.EnvironmentDefinition) error {
		effective, errorValue := current.Node.BuildEffectiveEnvironmentByName(environmentName, inherited)
		if errorValue != nil {
			return errorValue
		}
		result[current.Node.GetID()] = effective

		for _, child := range current.Children {
			if childError := build(child, effective); childError != nil {
				return childError
			}
		}
		return nil
	}

	if errorValue := build(runtime, domain.EnvironmentDefinition{Name: environmentName}); errorValue != nil {
		return nil, errorValue
	}
	return result, nil
}
