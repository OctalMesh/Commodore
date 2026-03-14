package commands

// Down stops node and subordinates in reverse DAG order.
func Down() Spec {
	return Spec{
		Use:   "down",
		Short: "Stop node and subordinates in reverse DAG order",
		RunE: func(provider Provider, _ []string) error {
			return runDown(provider.RuntimeService(), provider.OperationOptions())
		},
	}
}
