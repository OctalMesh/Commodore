/*
Package resolver implements ports.BinaryResolver.

Resolution order for a module binary:
 1. $PATH / %PATH%   – developer ran `go install` or placed binary in PATH.
 2. $GOPATH/bin      – standard Go install destination.
 3. <repo>/<cliPath> – local `go build` output.
 4. go run <src>     – zero-setup fallback; always works where Go is installed.
*/
package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BinaryResolver implements ports.BinaryResolver.
type BinaryResolver struct {}

// New returns a BinaryResolver.
func New() *BinaryResolver { return &BinaryResolver{} }

// Resolve returns the argv prefix for executing the named binary.
// repoRoot is the absolute repo root path; cliRelPath is e.g. "modules/console/cli".
func (r *BinaryResolver) Resolve(repoRoot, cliRelPath, name string) ([]string, error) {
	// 1. PATH
	if p, err := exec.LookPath(name); err == nil {
		return []string{p}, nil
	}

	// 2. $GOPATH/bin and ~/go/bin
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		if p := findBinary(filepath.Join(gopath, "bin"), name); p != "" {
			return []string{p}, nil
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := findBinary(filepath.Join(home, "go", "bin"), name); p != "" {
			return []string{p}, nil
		}
	}

	// Module availability check (skip for CLIs not inside modules/)
	cliFromSlash := filepath.FromSlash(cliRelPath)
	if modRoot, ok := moduleRoot(cliFromSlash); ok {
		gitMarker := filepath.Join(repoRoot, modRoot, ".git")
		if _, err := os.Stat(gitMarker); err != nil {
			modName := filepath.Base(modRoot)
			return nil, fmt.Errorf(
				"module %q (%s) has not been initialized.\n\n"+
					"Run:  git submodule update --init %s",
				modName, modRoot, filepath.ToSlash(modRoot),
			)
		}
	}

	// 3. Local pre-built binary
	cliSrcDir := filepath.Join(repoRoot, cliFromSlash)
	if p := findBinary(cliSrcDir, name); p != "" {
		return []string{p}, nil
	}

	// 4. go run <src> (zero-setup fallback)
	if _, err := os.Stat(filepath.Join(cliSrcDir, "main.go")); err == nil {
		fmt.Fprintf(os.Stderr, "  \x1b[33m⚠\x1b[0m  binary %q not found - falling back to 'go run' (slower startup)\n", name)
		return []string{"go", "run", cliSrcDir}, nil
	}

	return nil, fmt.Errorf(
		"%q CLI source not found at %s\nQuick setup:  cd %s && go build -o %s .",
		name, cliSrcDir, cliSrcDir, name,
	)
}

func findBinary(dir, name string) string {
	for _, candidate := range binaryCandidates(name) {
		p := filepath.Join(dir, candidate)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func binaryCandidates(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{name + ".exe", name}
	}

	return []string{name}
}

/*
moduleRoot returns the module root path relative to the repo root for CLI
paths that live inside "modules/"; returns ("", false) for the root CLI.

	moduleRoot("modules/console/cli") -> ("modules/console", true)
	moduleRoot("cli")                 -> ("",                false)
 */
func moduleRoot(cliPath string) (string, bool) {
	sep := string(filepath.Separator)
	if !strings.HasPrefix(cliPath, "modules"+sep) {
		return "", false
	}
	return filepath.Dir(cliPath), true
}
