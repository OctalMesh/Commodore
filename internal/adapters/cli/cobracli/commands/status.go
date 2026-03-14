package commands

// Status shows status for node and selected descendants.
func Status() Spec {
	return serviceAction(
		"status",
		"Show status for node and selected descendants",
		func(provider Provider) error {
			return provider.RuntimeService().Status(provider.OperationOptions())
		},
	)
}
