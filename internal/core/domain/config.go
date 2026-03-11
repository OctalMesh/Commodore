/*
Package domain contains the core entities for the Commodore SDK.
No external dependencies -- pure Go types.
*/
package domain

// Role identifies the role of a CLI binary in the OctalWeb platform.
type Role string

const (
	// RoleForeman is the platform root CLI (ow / octalweb).
	RoleForeman Role = "foreman"
	// RoleBrigadier is a module CLI (owc, ows, owl, owd, ...).
	RoleBrigadier Role = "brigadier"
	// RoleWorker is a single-service CLI that exposes only custom commands.
	RoleWorker Role = "worker"
)

// Engine describes the orchestration engine used to run the environment.
type Engine struct {
	// Type is the engine identifier, e.g. "tilt".
	Type string `yaml:"type"`
	// ConfigPath is the path to the engine config file relative to the project
	// root, e.g. "./dev/Tiltfile".
	ConfigPath string `yaml:"config_path"`
}

// Brigadier describes a module CLI managed by a Foreman.
type Brigadier struct {
	// ID is the module identifier used as the cobra sub-command name,
	// e.g. "console".
	ID string `yaml:"id"`
	// Path is the module root relative to the repo root, e.g. "modules/console".
	Path string `yaml:"path"`
	// Binary is the module CLI executable name, e.g. "owc".
	// Optional: if empty the brigadier is invoked via "go run".
	Binary string `yaml:"binary"`
}

// Worker describes a runnable service unit managed by a Brigadier.
type Worker struct {
	// ID is the worker identifier used as the cobra sub-command name,
	// e.g. "app-web".
	ID string `yaml:"id"`
	// Path is the worker root relative to the module root, e.g. "apps/web".
	Path string `yaml:"path"`
}

// WorkerCommand is a custom shell command registered on a Worker CLI.
type WorkerCommand struct {
	// Name is the cobra command name, e.g. "migrate".
	Name string `yaml:"name"`
	// Description is a short human-readable summary shown in help.
	Description string `yaml:"description"`
	// Exec is the shell command to run, e.g. "go run ./scripts/migrate.go".
	Exec string `yaml:"exec"`
	// Style controls output rendering: "minimal" (raw, default) or "tui".
	Style string `yaml:"style"`
}

// Config is the parsed .commodore configuration file.
type Config struct {
	// Role is "foreman", "brigadier", or "worker".
	Role Role `yaml:"role"`
	// ID is the human-readable project / module / service name, e.g. "octalweb".
	ID string `yaml:"id"`
	// Binary is the CLI executable name, e.g. "ow" or "owc".
	// Optional: defaults to ID at runtime when not set.
	Binary string `yaml:"binary"`
	// Aliases are additional names the binary responds to, e.g. ["octalweb"].
	Aliases []string `yaml:"aliases"`
	// Standalone flags a brigadier that operates without a foreman context.
	// When true the module entry-point works as a self-contained CLI.
	Standalone bool `yaml:"standalone"`
	// Engine describes how to start the dev environment.
	Engine Engine `yaml:"engine"`
	// Brigadiers lists the module CLIs managed by this foreman.
	// Only relevant when Role == RoleForeman.
	Brigadiers []Brigadier `yaml:"brigadiers"`
	// Workers lists the service units managed by this brigadier.
	// Only relevant when Role == RoleBrigadier.
	Workers []Worker `yaml:"workers"`
	// Commands lists the custom shell commands available on this worker.
	// Only relevant when Role == RoleWorker.
	Commands []WorkerCommand `yaml:"commands"`
}
