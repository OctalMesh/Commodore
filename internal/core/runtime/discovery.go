package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/runtime/actions"
)

func (service *Service) discoverConfigurationPath() (string, error) {
	if service.options.ConfigurationPath != "" {
		return domain.ResolveConfigurationPath(service.options.ConfigurationPath)
	}

	workingDirectory := service.options.WorkingDirectory
	if workingDirectory == "" {
		currentWorkingDirectory, errorValue := os.Getwd()
		if errorValue != nil {
			return "", fmt.Errorf("determining working directory: %w", errorValue)
		}
		workingDirectory = currentWorkingDirectory
	}

	candidatePaths := []string{workingDirectory}
	if discoveredPaths, errorValue := domain.DiscoverConfigurationPathsRecursively(workingDirectory); errorValue == nil {
		candidatePaths = append(candidatePaths, discoveredPaths...)
	}

	for _, candidatePath := range candidatePaths {
		resolvedPath, resolveError := domain.ResolveConfigurationPath(candidatePath)
		if resolveError == nil {
			return resolvedPath, nil
		}
	}

	return "", errors.New("could not locate .commodore; use --config")
}

func (service *Service) buildRuntimeTree(
	configurationPath string,
	reference *domain.NodeReference,
	namespacedID string,
) (*actions.RuntimeNode, map[string]*actions.RuntimeNode, error) {
	absoluteConfigurationPath, errorValue := filepath.Abs(configurationPath)
	if errorValue != nil {
		return nil, nil, fmt.Errorf("resolving configuration path %q: %w", configurationPath, errorValue)
	}

	configuration, loadError := service.options.NodeConfigLoader.LoadNodeConfiguration(absoluteConfigurationPath)
	if loadError != nil {
		return nil, nil, loadError
	}

	if reference != nil && reference.ID != "" && reference.ID != configuration.ID {
		return nil, nil, &domain.ValidationError{Message: fmt.Sprintf("manifest id %q does not match subordinate id %q", reference.ID, configuration.ID)}
	}

	configurationDirectory := filepath.Dir(absoluteConfigurationPath)
	localID := configuration.ID
	if reference != nil && reference.ID != "" {
		localID = reference.ID
	}

	resolvedNodeID := namespacedID
	if resolvedNodeID == "" {
		resolvedNodeID = localID
	}

	var node domain.OrchestrationNode
	switch configuration.Role {
	case domain.NodeRoleSquadron:
		node = &domain.SquadronNode{ID: resolvedNodeID, Path: configurationDirectory, Reactor: configuration.Reactor, Maneuvers: append([]domain.ManeuverDefinition(nil), configuration.Maneuvers...)}
	case domain.NodeRoleUnit:
		node = &domain.UnitNode{
			ID:                  resolvedNodeID,
			Path:                configurationDirectory,
			Reactor:             configuration.Reactor,
			Maneuvers:           append([]domain.ManeuverDefinition(nil), configuration.Maneuvers...),
			StandaloneExecution: reference == nil,
		}
	default:
		return nil, nil, &domain.ValidationError{Message: fmt.Sprintf("unsupported node role %q in %s", configuration.Role, absoluteConfigurationPath)}
	}

	if reference != nil {
		switch typedNode := node.(type) {
		case *domain.SquadronNode:
			typedNode.Tags = append([]string(nil), reference.Tags...)
			typedNode.After = append([]string(nil), reference.After...)
		case *domain.UnitNode:
			typedNode.Tags = append([]string(nil), reference.Tags...)
			typedNode.After = append([]string(nil), reference.After...)
		}
	}

	runtime := &actions.RuntimeNode{Node: node, LocalID: localID, Binary: resolveNodeBinary(configuration.Binary, localID)}
	if reference != nil {
		runtime.HealthCheck = reference.HealthCheck
	}
	nodesByID := map[string]*actions.RuntimeNode{node.GetID(): runtime}

	if configuration.Role != domain.NodeRoleSquadron {
		return runtime, nodesByID, nil
	}

	subordinateNamespaceByLocalID := make(map[string]string, len(configuration.Manifest))
	for _, subordinateReference := range configuration.Manifest {
		subordinateNamespaceByLocalID[subordinateReference.ID] = composeNamespacedNodeID(resolvedNodeID, subordinateReference.ID)
	}

	for _, subordinateReference := range configuration.Manifest {
		subordinateReference.After = rewriteDependencySelectors(subordinateReference.After, subordinateNamespaceByLocalID)
		subordinateInputPath := subordinateReference.Path
		if !filepath.IsAbs(subordinateInputPath) {
			subordinateInputPath = filepath.Join(configurationDirectory, subordinateReference.Path)
		}

		subordinateConfigurationPath, resolveError := domain.ResolveConfigurationPath(subordinateInputPath)
		if resolveError != nil {
			return nil, nil, &domain.DiscoveryError{
				Message: fmt.Sprintf("resolving subordinate %q at %q", subordinateReference.ID, subordinateInputPath),
				Cause:   resolveError,
			}
		}

		subordinateRuntime, subordinateNodeMap, buildError := service.buildRuntimeTree(
			subordinateConfigurationPath,
			&subordinateReference,
			subordinateNamespaceByLocalID[subordinateReference.ID],
		)
		if buildError != nil {
			return nil, nil, buildError
		}

		runtime.Children = append(runtime.Children, subordinateRuntime)
		for subordinateID, subordinateNode := range subordinateNodeMap {
			if _, exists := nodesByID[subordinateID]; exists {
				return nil, nil, &domain.ValidationError{Message: fmt.Sprintf("duplicate manifest id %q", subordinateID)}
			}
			nodesByID[subordinateID] = subordinateNode
		}
	}

	if squadronNode, ok := runtime.Node.(*domain.SquadronNode); ok {
		subordinates := make([]domain.OrchestrationNode, 0, len(runtime.Children))
		for _, childRuntime := range runtime.Children {
			subordinates = append(subordinates, childRuntime.Node)
		}
		squadronNode.Subordinates = subordinates
	}

	return runtime, nodesByID, nil
}

func composeNamespacedNodeID(parentNodeID string, localNodeID string) string {
	if parentNodeID == "" {
		return localNodeID
	}
	if localNodeID == "" {
		return parentNodeID
	}
	return parentNodeID + "-" + localNodeID
}

func rewriteDependencySelectors(
	dependencySelectors []string,
	namespaceByLocalNodeID map[string]string,
) []string {
	rewritten := make([]string, 0, len(dependencySelectors))
	for _, selector := range dependencySelectors {
		if strings.HasPrefix(selector, "$tag-") {
			rewritten = append(rewritten, selector)
			continue
		}
		if namespacedNodeID, found := namespaceByLocalNodeID[selector]; found {
			rewritten = append(rewritten, namespacedNodeID)
			continue
		}
		rewritten = append(rewritten, selector)
	}
	return rewritten
}

func resolveNodeBinary(configuredBinary string, localID string) string {
	if configuredBinary != "" {
		return configuredBinary
	}

	return localID
}
