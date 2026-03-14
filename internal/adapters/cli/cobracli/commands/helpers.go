package commands

type providerAction func(provider Provider) error

func serviceAction(use string, short string, action providerAction) Spec {
	return Spec{
		Use:   use,
		Short: short,
		RunE: func(provider Provider, _ []string) error {
			if action == nil {
				return nil
			}
			return action(provider)
		},
	}
}
