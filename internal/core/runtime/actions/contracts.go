package actions

import (
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// OperationOptions controls one orchestration capability call.
type OperationOptions struct {
	Environment string
	Tags        []string
}

// RuntimeNode stores resolved orchestration state for one node.
type RuntimeNode struct {
	Node        domain.OrchestrationNode
	LocalID     string
	Binary      string
	Children    []*RuntimeNode
	HealthCheck *domain.HealthCheckDefinition
}

// Engine defines orchestration internals required by actions.
type Engine interface {
	EnsureInitialized() error
	NormalizeOptions(options OperationOptions) OperationOptions
	Root() *RuntimeNode
	NodesByID() map[string]*RuntimeNode
	ResolveExecutionOrder(runtime *RuntimeNode, nodesByID map[string]*RuntimeNode) ([]*RuntimeNode, error)
	BuildEffectiveEnvironments(runtime *RuntimeNode, environmentName string) (map[string]domain.EnvironmentDefinition, error)
	CollectRuntimeNodes(runtime *RuntimeNode) []*RuntimeNode
	ShouldOperateOnNode(node domain.OrchestrationNode, tags []string) bool
	SelectReactor(node domain.OrchestrationNode) ports.Reactor
	VerifyNodeHealth(runtime *RuntimeNode, environment string) error
	RunProcess(request ports.ProcessRunRequest) error
	ListNodeManeuvers(node domain.OrchestrationNode) []domain.ManeuverDefinition
	Info(message string, arguments ...any)
	Warn(message string, arguments ...any)
	Success(message string, arguments ...any)
}
