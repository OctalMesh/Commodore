package commands

// Doctor validates DAG, environments, and maneuvers.
func Doctor() Spec {
	return serviceAction(
		"doctor",
		"Validate DAG, environments, and maneuvers",
		func(provider Provider) error {
			return provider.RuntimeService().Doctor(provider.OperationOptions())
		},
	)
}
