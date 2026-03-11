/*
Package ports declares the outbound interfaces (driven ports) for the
Commodore SDK. Nothing here imports infrastructure packages -- only domain.
*/
package ports

import "github.com/OctalMesh/Commodore/internal/core/domain"

// Finder locates a project or module root by walking up the filesystem.
// Satisfied by adapters/rootdir.Finder and adapters/moduledir.Finder.
type Finder interface {
	Find() (string, error)
}

// ToolChecker verifies that external tools are installed and accessible on PATH.
type ToolChecker interface {
	CheckAll() []domain.ToolResult
}

// SubmoduleRepository manages git submodule operations.
type SubmoduleRepository interface {
	List(repoRoot string) ([]domain.Submodule, error)
	Init(repoRoot string) error
	Update(repoRoot string) error
	Sync(repoRoot string) error
}

// BinaryResolver resolves the full command argv needed to execute a module binary.
// The returned slice contains only the executor prefix -- callers append subcommand args.
type BinaryResolver interface {
	Resolve(repoRoot, cliRelPath, name string) ([]string, error)
}

// Runner executes an external process, inheriting the caller's stdin/stdout/stderr.
type Runner interface {
	Run(argv []string, workDir string) error
}

// ConfigLoader loads a .commodore configuration file from the given path.
type ConfigLoader interface {
	Load(path string) (domain.Config, error)
}

/*
CompatChecker verifies that an external binary was built with the Commodore SDK.
argv is the executor prefix returned by BinaryResolver (e.g. ["owc"] or
["go", "run", "/path/to/cli"]). The implementation runs argv + ["__compat"]
and validates the output.
*/
type CompatChecker interface {
	Check(argv []string) error
}
