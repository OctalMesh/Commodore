/*
Package compat implements ports.CompatChecker.

Every binary built with the Commodore SDK exposes a hidden "__compat"
sub-command that prints a well-known marker line to stdout. The checker
runs that sub-command and validates the output before delegating any
real work to the binary.
*/
package compat

import (
	"fmt"
	"os/exec"
	"strings"
)

// marker is the prefix every Commodore-compatible binary prints when run
// with the "__compat" sub-command.
const marker = "commodore:sdk:"

// Checker implements ports.CompatChecker.
type Checker struct {}

// New returns a Checker.
func New() *Checker { return &Checker{} }

/*
Check runs argv + ["__compat"] and verifies the output starts with the
known Commodore SDK marker. It returns nil for "go run ..." prefixes so
that the zero-install fallback path is never blocked.
*/
func (c *Checker) Check(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("compat: empty argv")
	}

	// Skip the check for the "go run" zero-install fallback -- running the
	// full compilation just to verify compatibility would be too slow.
	if argv[0] == "go" {
		return nil
	}

	checkArgv := make([]string, len(argv)+1)
	copy(checkArgv, argv)
	checkArgv[len(argv)] = "__compat"

	out, err := exec.Command(checkArgv[0], checkArgv[1:]...).Output()
	if err != nil {
		// Older or third-party binaries may not support __compat; treat as
		// incompatible only when the binary exited non-zero AND produced no
		// recognizable output.  An exit error with matching output is fine.
		if !strings.HasPrefix(strings.TrimSpace(string(out)), marker) {
			return fmt.Errorf(
				"binary %q is not Commodore-compatible.\n"+
					"Make sure it was built with the Commodore SDK and reinstalled.\n"+
					"(hint: run './commodore install <cli-path> <name>')",
				argv[0],
			)
		}
	}

	if !strings.HasPrefix(strings.TrimSpace(string(out)), marker) {
		return fmt.Errorf(
			"binary %q does not identify itself as a Commodore SDK binary.\n"+
				"Expected output prefix %q from '__compat' sub-command.",
			argv[0], marker,
		)
	}

	return nil
}
