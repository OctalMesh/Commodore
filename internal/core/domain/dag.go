package domain

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

// DependencyGraph stores nodes and dependency edges.
//
// Edges are stored as dependencyID -> dependentID.
type DependencyGraph struct {
	nodesByID       map[string]OrchestrationNode
	edges           map[string]map[string]struct{}
	reverseEdges    map[string]map[string]struct{}
	dependencyCount map[string]int
}

// BuildDependencyGraph constructs and validates a graph from orchestration nodes.
func BuildDependencyGraph(nodes []OrchestrationNode) (*DependencyGraph, error) {
	if len(nodes) == 0 {
		return nil, &DependencyError{Message: "dependency graph cannot be built from an empty node set"}
	}

	graph := &DependencyGraph{
		nodesByID:       make(map[string]OrchestrationNode, len(nodes)),
		edges:           make(map[string]map[string]struct{}, len(nodes)),
		reverseEdges:    make(map[string]map[string]struct{}, len(nodes)),
		dependencyCount: make(map[string]int, len(nodes)),
	}

	tagIndex := make(map[string][]string)
	for _, node := range nodes {
		nodeIdentifier := node.GetID()
		if strings.TrimSpace(nodeIdentifier) == "" {
			return nil, &ValidationError{Message: "every node must have a non-empty ID"}
		}
		if _, alreadyExists := graph.nodesByID[nodeIdentifier]; alreadyExists {
			return nil, &ValidationError{Message: fmt.Sprintf("duplicate node ID %q in manifest", nodeIdentifier)}
		}

		graph.nodesByID[nodeIdentifier] = node
		graph.edges[nodeIdentifier] = map[string]struct{}{}
		graph.reverseEdges[nodeIdentifier] = map[string]struct{}{}
		graph.dependencyCount[nodeIdentifier] = 0

		for _, tag := range node.GetTags() {
			tagIndex[tag] = append(tagIndex[tag], nodeIdentifier)
		}
	}

	for _, node := range nodes {
		nodeIdentifier := node.GetID()
		for _, dependencySelector := range node.GetDependencies() {
			resolvedDependencies, errorValue := resolveDependencySelector(nodeIdentifier, dependencySelector, graph.nodesByID, tagIndex)
			if errorValue != nil {
				return nil, errorValue
			}

			for _, dependencyID := range resolvedDependencies {
				if dependencyID == nodeIdentifier {
					return nil, &DependencyError{Message: fmt.Sprintf("node %q cannot depend on itself", nodeIdentifier)}
				}

				if _, found := graph.edges[dependencyID][nodeIdentifier]; found {
					continue
				}

				graph.edges[dependencyID][nodeIdentifier] = struct{}{}
				graph.reverseEdges[nodeIdentifier][dependencyID] = struct{}{}
				graph.dependencyCount[nodeIdentifier]++
			}
		}
	}

	if _, errorValue := graph.ResolveStartupOrder(); errorValue != nil {
		return nil, errorValue
	}

	return graph, nil
}

// ResolveStartupOrder returns a stable topological order.
func (graph *DependencyGraph) ResolveStartupOrder() ([]OrchestrationNode, error) {
	temporaryDependencyCount := make(map[string]int, len(graph.dependencyCount))
	maps.Copy(temporaryDependencyCount, graph.dependencyCount)

	availableQueue := make([]string, 0, len(temporaryDependencyCount))
	for nodeIdentifier, count := range temporaryDependencyCount {
		if count == 0 {
			availableQueue = append(availableQueue, nodeIdentifier)
		}
	}
	sort.Strings(availableQueue)

	resolvedOrder := make([]OrchestrationNode, 0, len(graph.nodesByID))
	for len(availableQueue) > 0 {
		nodeIdentifier := availableQueue[0]
		availableQueue = availableQueue[1:]

		resolvedOrder = append(resolvedOrder, graph.nodesByID[nodeIdentifier])

		dependents := sortedMapKeys(graph.edges[nodeIdentifier])
		for _, dependentIdentifier := range dependents {
			temporaryDependencyCount[dependentIdentifier]--
			if temporaryDependencyCount[dependentIdentifier] == 0 {
				availableQueue = append(availableQueue, dependentIdentifier)
				sort.Strings(availableQueue)
			}
		}
	}

	if len(resolvedOrder) != len(graph.nodesByID) {
		cyclicNodes := make([]string, 0)
		for nodeIdentifier, count := range temporaryDependencyCount {
			if count > 0 {
				cyclicNodes = append(cyclicNodes, nodeIdentifier)
			}
		}
		sort.Strings(cyclicNodes)
		return nil, &DependencyError{Message: fmt.Sprintf("cycle detected in dependency graph involving: %s", strings.Join(cyclicNodes, ", "))}
	}

	return resolvedOrder, nil
}

// ResolveStartupWaves returns dependency-safe execution groups.
func (graph *DependencyGraph) ResolveStartupWaves() ([][]OrchestrationNode, error) {
	temporaryDependencyCount := make(map[string]int, len(graph.dependencyCount))
	maps.Copy(temporaryDependencyCount, graph.dependencyCount)

	currentWaveNodeIdentifiers := make([]string, 0)
	for nodeIdentifier, count := range temporaryDependencyCount {
		if count == 0 {
			currentWaveNodeIdentifiers = append(currentWaveNodeIdentifiers, nodeIdentifier)
		}
	}
	sort.Strings(currentWaveNodeIdentifiers)

	startupWaves := make([][]OrchestrationNode, 0)
	processedNodeCount := 0

	for len(currentWaveNodeIdentifiers) > 0 {
		currentWave := make([]OrchestrationNode, 0, len(currentWaveNodeIdentifiers))
		nextWaveCandidates := make(map[string]struct{})

		for _, nodeIdentifier := range currentWaveNodeIdentifiers {
			currentWave = append(currentWave, graph.nodesByID[nodeIdentifier])
			processedNodeCount++

			for _, dependentIdentifier := range sortedMapKeys(graph.edges[nodeIdentifier]) {
				temporaryDependencyCount[dependentIdentifier]--
				if temporaryDependencyCount[dependentIdentifier] == 0 {
					nextWaveCandidates[dependentIdentifier] = struct{}{}
				}
			}
		}

		startupWaves = append(startupWaves, currentWave)
		currentWaveNodeIdentifiers = sortedMapKeys(nextWaveCandidates)
	}

	if processedNodeCount != len(graph.nodesByID) {
		return nil, &DependencyError{Message: "cycle detected while resolving startup waves"}
	}

	return startupWaves, nil
}

func resolveDependencySelector(
	nodeIdentifier string,
	dependencySelector string,
	nodesByID map[string]OrchestrationNode,
	tagIndex map[string][]string,
) ([]string, error) {
	if dependencySelector == "" {
		return nil, &DependencyError{Message: fmt.Sprintf("node %q declares an empty dependency selector", nodeIdentifier)}
	}

	if !strings.HasPrefix(dependencySelector, "$tag-") {
		if _, found := nodesByID[dependencySelector]; !found {
			return nil, &DependencyError{Message: fmt.Sprintf("node %q references unknown dependency ID %q", nodeIdentifier, dependencySelector)}
		}
		return []string{dependencySelector}, nil
	}

	tagName := strings.TrimPrefix(dependencySelector, "$tag-")
	if tagName == "" {
		return nil, &DependencyError{Message: fmt.Sprintf("node %q has invalid dependency selector %q", nodeIdentifier, dependencySelector)}
	}

	taggedNodeIdentifiers, found := tagIndex[tagName]
	if !found || len(taggedNodeIdentifiers) == 0 {
		return nil, &DependencyError{Message: fmt.Sprintf("node %q references unknown tag selector %q", nodeIdentifier, dependencySelector)}
	}

	uniqueIdentifiers := make(map[string]struct{}, len(taggedNodeIdentifiers))
	for _, taggedNodeIdentifier := range taggedNodeIdentifiers {
		uniqueIdentifiers[taggedNodeIdentifier] = struct{}{}
	}

	return sortedMapKeys(uniqueIdentifiers), nil
}

func sortedMapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
