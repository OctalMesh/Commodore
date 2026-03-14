package actions

import "context"

// Down stops the runtime root and subordinates in reverse dependency order.
func Down(engine Engine, options OperationOptions) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	normalized := engine.NormalizeOptions(options)
	orderedNodes, errorValue := engine.ResolveExecutionOrder(engine.Root(), engine.NodesByID())
	if errorValue != nil {
		return errorValue
	}

	for index := len(orderedNodes) - 1; index >= 0; index-- {
		orderedRuntime := orderedNodes[index]
		if !engine.ShouldOperateOnNode(orderedRuntime.Node, normalized.Tags) {
			continue
		}

		reactorAdapter := engine.SelectReactor(orderedRuntime.Node)
		if reactorAdapter == nil {
			continue
		}

		if stopError := reactorAdapter.StopNode(
			context.Background(),
			orderedRuntime.Node,
			normalized.Environment,
		); stopError != nil {
			engine.Warn("node %s failed to stop: %v", orderedRuntime.Node.GetID(), stopError)
			continue
		}
		engine.Success("stopped %s", orderedRuntime.Node.GetID())
	}

	return nil
}
