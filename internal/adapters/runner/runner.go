/*
Package runner implements ports.Runner using os/exec.
*/
package runner

import (
	"fmt"
	"os"
	"os/exec"
)

// OSRunner implements ports.Runner.
type OSRunner struct {}

// New returns an OSRunner.
func New() *OSRunner { return &OSRunner{} }

// Run executes argv[0] with argv[1:] as arguments, inheriting stdin/stdout/stderr.
func (r *OSRunner) Run(argv []string, workDir string) error {
	if len(argv) == 0 {
		return fmt.Errorf("runner: empty argv")
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if workDir != "" {
		cmd.Dir = workDir
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", argv[0], err)
	}

	return nil
}
