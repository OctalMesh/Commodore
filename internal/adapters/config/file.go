/*
Package config implements ports.ConfigLoader by reading a YAML file from disk.
*/
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/OctalMesh/Commodore/internal/core/domain"
)

// FileLoader satisfies ports.ConfigLoader.
type FileLoader struct {}

// New creates a new FileLoader.
func New() *FileLoader { return &FileLoader{} }

// Load reads and parses the .commodore YAML file at path.
func (l *FileLoader) Load(path string) (domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Config{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg domain.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	return cfg, nil
}
