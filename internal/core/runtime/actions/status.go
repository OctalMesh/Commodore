package actions

import (
	"sort"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/ui/widgets"
)

// Status reports runtime node statuses for current scope.
func Status(engine Engine, options OperationOptions) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	normalized := engine.NormalizeOptions(options)
	nodes := engine.CollectRuntimeNodes(engine.Root())
	rows := make([]statusRow, 0, len(nodes))
	for _, nodeRuntime := range nodes {
		if !engine.ShouldOperateOnNode(nodeRuntime.Node, normalized.Tags) {
			continue
		}

		rows = append(rows, statusRow{
			NodeID:      nodeRuntime.Node.GetID(),
			LocalID:     nodeRuntime.LocalID,
			Role:        resolveRoleLabel(nodeRuntime.Node),
			Reactor:     resolveReactorLabel(nodeRuntime.Node),
			Binary:      resolveBinaryLabel(nodeRuntime.Node, nodeRuntime.Binary),
			Healthcheck: resolveHealthcheckLabel(nodeRuntime.HealthCheck),
			Tags:        strings.Join(nodeRuntime.Node.GetTags(), ","),
			Path:        nodeRuntime.Node.GetPath(),
		})
	}

	if len(rows) == 0 {
		engine.Info("status: no nodes matched current scope (env=%s tags=%s)", normalized.Environment, strings.Join(normalized.Tags, ","))
		return nil
	}

	sort.Slice(rows, func(left int, right int) bool {
		return rows[left].NodeID < rows[right].NodeID
	})

	for _, line := range renderStatusTable(rows, normalized) {
		engine.Info("%s", line)
	}

	return nil
}

type statusRow struct {
	NodeID      string
	LocalID     string
	Role        string
	Reactor     string
	Binary      string
	Healthcheck string
	Tags        string
	Path        string
}

func resolveRoleLabel(node domain.OrchestrationNode) string {
	switch node.GetRole() {
	case domain.NodeRoleSquadron:
		return string(domain.NodeRoleSquadron)
	default:
		return string(domain.NodeRoleUnit)
	}
}

func resolveReactorLabel(node domain.OrchestrationNode) string {
	switch typed := node.(type) {
	case *domain.SquadronNode:
		if typed.Reactor.Provider == "" {
			return string(domain.ReactorProviderTilt)
		}
		return string(typed.Reactor.Provider)
	case *domain.UnitNode:
		if typed.Reactor.Provider == "" {
			return string(domain.ReactorProviderTilt)
		}
		return string(typed.Reactor.Provider)
	default:
		return "-"
	}
}

func resolveBinaryLabel(node domain.OrchestrationNode, binary string) string {
	if node.GetRole() != domain.NodeRoleSquadron {
		return "-"
	}
	if strings.TrimSpace(binary) == "" {
		return "-"
	}
	return binary
}

func resolveHealthcheckLabel(definition *domain.HealthCheckDefinition) string {
	if definition == nil {
		return "no"
	}
	if len(definition.Test) == 0 {
		return "invalid"
	}
	return "yes"
}

func renderStatusTable(rows []statusRow, options OperationOptions) []string {
	columns := []widgets.TableColumn{
		{Header: "NODE", MaxWidth: 30},
		{Header: "LOCAL", MaxWidth: 20},
		{Header: "ROLE", MaxWidth: 10},
		{Header: "REACTOR", MaxWidth: 12},
		{Header: "BINARY", MaxWidth: 28},
		{Header: "HC", MaxWidth: 7},
		{Header: "TAGS", MaxWidth: 24},
		{Header: "PATH", MaxWidth: 42},
	}

	data := make([][]string, 0, len(rows))
	for _, row := range rows {
		data = append(data, []string{
			row.NodeID,
			row.LocalID,
			row.Role,
			row.Reactor,
			row.Binary,
			row.Healthcheck,
			row.Tags,
			row.Path,
		})
	}

	lines := make([]string, 0, len(data)+2)
	lines = append(lines, "status: env="+options.Environment+" tags="+strings.Join(options.Tags, ","))
	lines = append(lines, widgets.RenderTable(columns, data)...)

	return lines
}
