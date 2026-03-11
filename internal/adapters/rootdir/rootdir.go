/*
Package rootdir implements ports.Finder by walking up the directory tree.
The repository root is identified by the presence of both "modules/" and "cli/"
as direct children of a candidate directory.
*/
package rootdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound is returned when no repository root can be located.
var ErrNotFound = errors.New(
	"OctalWeb repository root not found: no directory containing both 'modules/' and 'cli/' was found",
)

// Finder implements ports.Finder for the repository root.
type Finder struct {}

// New returns a Finder.
func New() *Finder { return &Finder{} }

// Find walks upward from the current working directory to the repo root.
func (f *Finder) Find() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("rootdir: getwd: %w", err)
	}
	current, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("rootdir: abs: %w", err)
	}
	for {
		if isRoot(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNotFound
		}
		current = parent
	}
}

func isRoot(dir string) bool {
	for _, child := range []string{"modules", "cli"} {
		if _, err := os.Stat(filepath.Join(dir, child)); err != nil {
			return false
		}
	}

	return true
}
