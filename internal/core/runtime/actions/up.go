package actions

import (
	"context"
)

// Up starts the runtime root and subordinates in dependency order.
func Up(engine Engine, options OperationOptions) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	normalized := engine.NormalizeOptions(options)
	runtimeRoot := engine.Root()
	effectiveEnvironments, errorValue := engine.BuildEffectiveEnvironments(runtimeRoot, normalized.Environment)
	if errorValue != nil {
		return errorValue
	}

	orderedNodes, errorValue := engine.ResolveExecutionOrder(runtimeRoot, engine.NodesByID())
	if errorValue != nil {
		return errorValue
	}

	for _, orderedRuntime := range orderedNodes {
		if !engine.ShouldOperateOnNode(orderedRuntime.Node, normalized.Tags) {
			continue
		}

		reactorAdapter := engine.SelectReactor(orderedRuntime.Node)
		if reactorAdapter == nil {
			continue
		}

		if startError := reactorAdapter.StartNode(
			context.Background(),
			orderedRuntime.Node,
			normalized.Environment,
			effectiveEnvironments[orderedRuntime.Node.GetID()],
		); startError != nil {
			engine.Warn("node %s failed to start: %v", orderedRuntime.Node.GetID(), startError)
			continue
		}

		if healthError := engine.VerifyNodeHealth(orderedRuntime, normalized.Environment); healthError != nil {
			engine.Warn("node %s health check failed: %v", orderedRuntime.Node.GetID(), healthError)
			continue
		}

		engine.Success("started %s", orderedRuntime.Node.GetID())
	}

	return nil
}
