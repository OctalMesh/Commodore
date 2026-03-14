package commands

// Tree shows discovered subordinate structure.
func Tree() Spec {
	return serviceAction(
		"tree",
		"Show discovered subordinate structure",
		func(provider Provider) error {
			return provider.RuntimeService().Tree()
		},
	)
}
