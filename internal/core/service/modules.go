package service

import (
	"fmt"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// ModulesService wraps git submodule operations for the OctalWeb monorepo.
type ModulesService struct {
	repo   ports.SubmoduleRepository
	finder ports.Finder
}

// NewModulesService constructs a ModulesService.
func NewModulesService(repo ports.SubmoduleRepository, finder ports.Finder) *ModulesService {
	return &ModulesService{repo: repo, finder: finder}
}

// Status returns the current state of all git submodules.
func (s *ModulesService) Status() ([]domain.Submodule, error) {
	root, err := s.finder.Find()
	if err != nil {
		return nil, fmt.Errorf("modules status: %w", err)
	}

	mods, err := s.repo.List(root)
	if err != nil {
		return nil, fmt.Errorf("listing submodules: %w", err)
	}

	return mods, nil
}

// Init runs "git submodule update --init --recursive".
func (s *ModulesService) Init() error {
	root, err := s.finder.Find()

	if err != nil {
		return fmt.Errorf("modules init: %w", err)
	}

	if err := s.repo.Init(root); err != nil {
		return fmt.Errorf("modules init: %w", err)
	}

	return nil
}

func (s *ModulesService) Update() error {
	root, err := s.finder.Find()

	if err != nil {
		return fmt.Errorf("modules update: %w", err)
	}

	if err := s.repo.Update(root); err != nil {
		return fmt.Errorf("modules update: %w", err)
	}

	return nil
}

func (s *ModulesService) Sync() error {
	root, err := s.finder.Find()

	if err != nil {
		return fmt.Errorf("modules sync: %w", err)
	}

	if err := s.repo.Sync(root); err != nil {
		return fmt.Errorf("modules sync: %w", err)
	}

	return nil
}
