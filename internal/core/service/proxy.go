package service

import (
	"fmt"
	"path/filepath"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

/*
ProxyService resolves a brigadier binary, verifies its compatibility, and
delegates execution to it with the supplied arguments.
 */
type ProxyService struct {
	finder   ports.Finder
	resolver ports.BinaryResolver
	runner   ports.Runner
	compat   ports.CompatChecker
}

// NewProxyService constructs a ProxyService.
func NewProxyService(
	finder ports.Finder,
	resolver ports.BinaryResolver,
	runner ports.Runner,
	compat ports.CompatChecker,
) *ProxyService {
	return &ProxyService{finder: finder, resolver: resolver, runner: runner, compat: compat}
}

/*
Exec resolves the binary for brigadier, runs the Commodore compatibility check,
and executes it with args. The working directory is set to the module root
(the parent of cliPath in the repo).
*/
func (s *ProxyService) Exec(b domain.Brigadier, args []string) error {
	repoRoot, err := s.finder.Find()
	if err != nil {
		return fmt.Errorf("proxy: locating repo root: %w", err)
	}

	cliPath := filepath.Join(b.Path, "cli")
	argv, err := s.resolver.Resolve(repoRoot, filepath.ToSlash(cliPath), b.Binary)
	if err != nil {
		return fmt.Errorf("proxy: resolving binary %q: %w", b.Binary, err)
	}

	if err := s.compat.Check(argv); err != nil {
		return fmt.Errorf("proxy: compat check: %w", err)
	}

	argv = append(argv, args...)
	workDir := filepath.Join(repoRoot, filepath.FromSlash(b.Path))
	if err := s.runner.Run(argv, workDir); err != nil {
		return fmt.Errorf("proxy: exec %s: %w", b.Binary, err)
	}

	return nil
}
