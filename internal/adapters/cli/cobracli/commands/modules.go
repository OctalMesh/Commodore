package commands

// Modules manages git submodules.
func Modules() Spec {
	return Spec{
		Use:   "modules",
		Short: "Manage git submodules",
		Children: []Spec{
			serviceAction("status", "Show git submodule status", func(provider Provider) error {
				return provider.RuntimeService().ModulesStatus()
			}),
			serviceAction("init", "Initialize git submodules", func(provider Provider) error {
				return provider.RuntimeService().ModulesInit()
			}),
			serviceAction("update", "Update git submodules", func(provider Provider) error {
				return provider.RuntimeService().ModulesUpdate()
			}),
			serviceAction("sync", "Synchronize git submodule URLs", func(provider Provider) error {
				return provider.RuntimeService().ModulesSync()
			}),
		},
	}
}
