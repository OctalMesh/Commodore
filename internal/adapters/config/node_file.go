package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// NodeFileLoader loads squadron/unit YAML files used by the new orchestration runtime.
type NodeFileLoader struct{}

// NewNodeLoader creates a new NodeFileLoader.
func NewNodeLoader() *NodeFileLoader {
	return &NodeFileLoader{}
}

// LoadNodeConfiguration loads and parses a squadron/unit configuration file.
func (loader *NodeFileLoader) LoadNodeConfiguration(configurationPath string) (domain.NodeConfiguration, error) {
	configurationData, readError := os.ReadFile(configurationPath)
	if readError != nil {
		return domain.NodeConfiguration{}, &ports.ConfigLoadError{
			Message: fmt.Sprintf("reading configuration %s", configurationPath),
			Cause:   readError,
		}
	}

	var configuration domain.NodeConfiguration
	if parseError := yaml.Unmarshal(configurationData, &configuration); parseError != nil {
		return domain.NodeConfiguration{}, &ports.ConfigLoadError{
			Message: fmt.Sprintf("parsing configuration %s", configurationPath),
			Cause:   parseError,
		}
	}

	if configuration.Role != domain.NodeRoleSquadron && configuration.Role != domain.NodeRoleUnit {
		return domain.NodeConfiguration{}, &ports.ConfigLoadError{
			Message: fmt.Sprintf("unsupported role %q in %s", configuration.Role, configurationPath),
		}
	}

	if configuration.ID == "" {
		return domain.NodeConfiguration{}, &ports.ConfigLoadError{
			Message: fmt.Sprintf("missing id in %s", configurationPath),
		}
	}

	return configuration, nil
}
