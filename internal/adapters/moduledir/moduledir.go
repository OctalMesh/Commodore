/*
Package moduledir implements ports.Finder for a brigadier module root.
It walks up the filesystem looking for a directory that has cli/, dev/, or
apps/ as children but does NOT have a modules/ child (which would be the
repo root, not a module root).
*/
package moduledir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Finder locates the module root for a named brigadier module.
type Finder struct { name string }

// New returns a Finder for the given module name (used in error messages).
func New(name string) *Finder { return &Finder{name: name} }

// Find walks upward from cwd to locate the module root.
func (f *Finder) Find() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("moduledir: getwd: %w", err)
	}

	dir := cwd
	for {
		if isModuleRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New(
				"module root for " + f.name + " not found; run this command from inside the module directory",
			)
		}
		dir = parent
	}
}

func isModuleRoot(dir string) bool {
	// Must have a cli/ sub-directory.
	if _, err := os.Stat(filepath.Join(dir, "cli")); err != nil {
		return false
	}
	// Must have at least one of apps/, services/, dev/.
	hasContent := false
	for _, child := range []string{"apps", "services", "dev"} {
		if _, err := os.Stat(filepath.Join(dir, child)); err == nil {
			hasContent = true
			break
		}
	}

	if !hasContent {
		return false
	}

	// Must NOT have modules/ (that would be the repo root).
	if _, err := os.Stat(filepath.Join(dir, "modules")); err == nil {
		return false
	}

	return true
}
