package runtime

import (
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/runtime/actions"
)

func (service *Service) CollectRuntimeNodes(runtime *actions.RuntimeNode) []*actions.RuntimeNode {
	result := make([]*actions.RuntimeNode, 0)
	queue := []*actions.RuntimeNode{runtime}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		queue = append(queue, current.Children...)
	}
	return result
}

func (service *Service) ResolveExecutionOrder(runtime *actions.RuntimeNode, nodesByID map[string]*actions.RuntimeNode) ([]*actions.RuntimeNode, error) {
	nodes := service.CollectRuntimeNodes(runtime)
	descendants := make([]domain.OrchestrationNode, 0)
	for _, nodeRuntime := range nodes {
		if nodeRuntime == runtime {
			continue
		}
		descendants = append(descendants, nodeRuntime.Node)
	}

	if len(descendants) == 0 {
		return []*actions.RuntimeNode{runtime}, nil
	}

	dependencyGraph, graphError := domain.BuildDependencyGraph(descendants)
	if graphError != nil {
		return nil, graphError
	}

	order, orderError := dependencyGraph.ResolveStartupOrder()
	if orderError != nil {
		return nil, orderError
	}

	resolved := make([]*actions.RuntimeNode, 0, len(order)+1)
	resolved = append(resolved, runtime)
	for _, node := range order {
		runtimeNodeByID := nodesByID[node.GetID()]
		if runtimeNodeByID == nil {
			continue
		}
		resolved = append(resolved, runtimeNodeByID)
	}

	return resolved, nil
}
