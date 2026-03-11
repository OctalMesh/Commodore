package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

/*
HireWorkerService resolves the Tilt entry-point for a named worker so that
the brigadier can start it in isolation.
 */
type HireWorkerService struct {
	finder ports.Finder
	loader ports.ConfigLoader
}

// NewHireWorkerService constructs a HireWorkerService.
func NewHireWorkerService(finder ports.Finder, loader ports.ConfigLoader) *HireWorkerService {
	return &HireWorkerService{finder: finder, loader: loader}
}

/*
WorkerTiltfile looks up the worker with the given id inside workers, then
resolves the absolute path to its Tiltfile and its working directory.

Resolution order for the Tiltfile path:
 1. engine.config_path value from the worker's own .commodore file.
 2. Fallback: <worker-root>/dev/Tiltfile.
*/
func (s *HireWorkerService) WorkerTiltfile(workers []domain.Worker, id string) (tiltfile, workerRoot string, err error) {
	moduleRoot, err := s.finder.Find()
	if err != nil {
		return "", "", fmt.Errorf("hire: locating module root: %w", err)
	}

	var found *domain.Worker
	for i := range workers {
		if workers[i].ID == id {
			found = &workers[i]
			break
		}
	}
	if found == nil {
		names := make([]string, len(workers))
		for i, w := range workers {
			names[i] = w.ID
		}
		return "", "", fmt.Errorf(
			"worker %q not found; available: %s",
			id, strings.Join(names, ", "),
		)
	}

	workerRoot = filepath.Join(moduleRoot, filepath.FromSlash(found.Path))

	// Prefer the worker's own .commodore engine config.
	cfg, loadErr := s.loader.Load(filepath.Join(workerRoot, ".commodore"))
	if loadErr == nil && cfg.Engine.ConfigPath != "" {
		return filepath.Join(workerRoot, filepath.FromSlash(cfg.Engine.ConfigPath)), workerRoot, nil
	}

	// Fallback to the conventional location.
	return filepath.Join(workerRoot, "dev", "Tiltfile"), workerRoot, nil
}
