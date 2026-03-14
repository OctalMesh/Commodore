package runtime

import (
	"context"
	"fmt"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/logging"
	"github.com/OctalMesh/Commodore/internal/core/ports"
	"github.com/OctalMesh/Commodore/internal/core/runtime/actions"
)

// Options configures runtime service construction.
type Options struct {
	ConfigurationPath string
	WorkingDirectory  string
	Logger            logging.Logger

	NodeConfigLoader ports.NodeConfigLoader
	TiltReactor      ports.Reactor
	NativeReactor    ports.Reactor
	ProcessRunner    ports.ProcessRunner
}

// OperationOptions controls one orchestration capability call.
type OperationOptions = actions.OperationOptions

// Service is the API-first runtime service layer.
type Service struct {
	options Options
	logger  logging.Logger

	runtimeRoot *actions.RuntimeNode
	nodesByID   map[string]*actions.RuntimeNode
}

// NewService creates a runtime service.
func NewService(options Options) *Service {
	return &Service{
		options: options,
		logger:  options.Logger,
	}
}

// Initialize discovers configuration and builds runtime topology.
func (service *Service) Initialize() error {
	configurationPath, errorValue := service.discoverConfigurationPath()
	if errorValue != nil {
		return errorValue
	}

	runtimeRoot, nodesByID, errorValue := service.buildRuntimeTree(configurationPath, nil, "")
	if errorValue != nil {
		return errorValue
	}

	service.runtimeRoot = runtimeRoot
	service.nodesByID = nodesByID
	return nil
}

// RuntimeWorkingDirectory returns the discovered root path.
func (service *Service) RuntimeWorkingDirectory() string {
	if service.runtimeRoot == nil {
		return ""
	}
	return service.runtimeRoot.Node.GetPath()
}

// RootNodeID returns runtime root node identifier.
func (service *Service) RootNodeID() string {
	if service.runtimeRoot == nil {
		return ""
	}
	return service.runtimeRoot.Node.GetID()
}

// SetConfigurationPath updates explicit configuration path before initialization.
func (service *Service) SetConfigurationPath(configurationPath string) {
	service.options.ConfigurationPath = configurationPath
}

// RootManeuvers returns root node maneuvers for command surface generation.
func (service *Service) RootManeuvers() []domain.ManeuverDefinition {
	if service.runtimeRoot == nil {
		return nil
	}
	return service.ListNodeManeuvers(service.runtimeRoot.Node)
}

func (service *Service) EnsureInitialized() error {
	if service.runtimeRoot != nil {
		return nil
	}
	if initializeError := service.Initialize(); initializeError != nil {
		return initializeError
	}
	if service.runtimeRoot == nil {
		return fmt.Errorf("runtime root was not initialized")
	}
	return nil
}

func (service *Service) NormalizeOptions(options actions.OperationOptions) actions.OperationOptions {
	normalized := options
	if normalized.Environment == "" {
		normalized.Environment = "dev"
	}
	return normalized
}

func (service *Service) Root() *actions.RuntimeNode { return service.runtimeRoot }

func (service *Service) NodesByID() map[string]*actions.RuntimeNode { return service.nodesByID }

func (service *Service) Info(message string, arguments ...any) {
	service.logger.Info(message, arguments...)
}
func (service *Service) Warn(message string, arguments ...any) {
	service.logger.Warn(message, arguments...)
}
func (service *Service) Success(message string, arguments ...any) {
	service.logger.Success(message, arguments...)
}

func (service *Service) RunProcess(request ports.ProcessRunRequest) error {
	return service.options.ProcessRunner.Run(context.Background(), request)
}

func (service *Service) ListNodeManeuvers(node domain.OrchestrationNode) []domain.ManeuverDefinition {
	switch typedNode := node.(type) {
	case *domain.SquadronNode:
		return append([]domain.ManeuverDefinition(nil), typedNode.Maneuvers...)
	case *domain.UnitNode:
		return append([]domain.ManeuverDefinition(nil), typedNode.Maneuvers...)
	default:
		return nil
	}
}

func (service *Service) Up(options OperationOptions) error   { return actions.Up(service, options) }
func (service *Service) Down(options OperationOptions) error { return actions.Down(service, options) }
func (service *Service) Status(options OperationOptions) error {
	return actions.Status(service, options)
}
func (service *Service) Doctor(options OperationOptions) error {
	return actions.Doctor(service, options)
}
func (service *Service) Signal(options OperationOptions, arguments []string) error {
	return actions.Signal(service, options, arguments)
}
func (service *Service) Tree() error { return actions.Tree(service) }
func (service *Service) Maneuver(options OperationOptions, maneuverName string) error {
	return actions.Maneuver(service, options, maneuverName)
}
func (service *Service) ModulesStatus() error { return actions.ModulesStatus(service) }

// CollectTiltfiles returns ordered tiltfile paths for all Tilt-backed nodes
// that should participate in the current fleet operation, along with the
// project root directory.
// Parent Tiltfiles are preserved so module-level deployment definitions can be
// combined with unit-level image builds in a single generated Tiltfile.
// Used by the cobracli adapter to build a combined fleet Tiltfile.
func (service *Service) CollectTiltfiles(options OperationOptions) (tiltfilePaths []string, rootDir string, err error) {
	if err = service.EnsureInitialized(); err != nil {
		return
	}

	normalized := service.NormalizeOptions(options)
	orderedNodes, resolveErr := service.ResolveExecutionOrder(service.runtimeRoot, service.nodesByID)
	if resolveErr != nil {
		err = resolveErr
		return
	}

	seen := make(map[string]struct{})
	for _, node := range orderedNodes {
		reactor := service.SelectReactor(node.Node)
		if reactor == nil {
			continue
		}
		provider, ok := reactor.(ports.TiltfileProvider)
		if !ok {
			continue
		}
		tiltfilePath, workDir, resolvePathErr := provider.ResolveTiltfilePath(node.Node, normalized.Environment)
		if resolvePathErr != nil {
			continue
		}
		if !service.ShouldOperateOnNode(node.Node, normalized.Tags) && !service.hasOperableTiltfileDescendant(node, normalized.Environment, normalized.Tags) {
			continue
		}
		if _, duplicate := seen[tiltfilePath]; duplicate {
			continue
		}
		seen[tiltfilePath] = struct{}{}
		tiltfilePaths = append(tiltfilePaths, tiltfilePath)
		if rootDir == "" {
			rootDir = workDir
		}
	}

	if rootDir == "" && service.runtimeRoot != nil {
		rootDir = service.runtimeRoot.Node.GetPath()
	}
	return
}

// hasOperableTiltfileDescendant reports whether any direct or transitive child
// of node resolves to a Tiltfile for the given environment and matches the
// current tag selection.
func (service *Service) hasOperableTiltfileDescendant(node *actions.RuntimeNode, environment string, tags []string) bool {
	for _, child := range node.Children {
		reactor := service.SelectReactor(child.Node)
		if reactor == nil {
			continue
		}
		provider, ok := reactor.(ports.TiltfileProvider)
		if !ok {
			continue
		}
		if _, _, resolveErr := provider.ResolveTiltfilePath(child.Node, environment); resolveErr == nil && service.ShouldOperateOnNode(child.Node, tags) {
			return true
		}
		if service.hasOperableTiltfileDescendant(child, environment, tags) {
			return true
		}
	}
	return false
}
func (service *Service) ModulesInit() error   { return actions.ModulesInit(service) }
func (service *Service) ModulesUpdate() error { return actions.ModulesUpdate(service) }
func (service *Service) ModulesSync() error   { return actions.ModulesSync(service) }
