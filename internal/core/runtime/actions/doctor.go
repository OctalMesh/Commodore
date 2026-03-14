package actions

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/ui/widgets"
)

// Doctor validates topology and maneuver definitions.
func Doctor(engine Engine, options OperationOptions) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	normalized := engine.NormalizeOptions(options)
	runtimeRoot := engine.Root()

	globalRows := make([]doctorRow, 0, 2)
	doctorIssues := make([]string, 0)

	if _, errorValue := engine.ResolveExecutionOrder(runtimeRoot, engine.NodesByID()); errorValue != nil {
		globalRows = append(globalRows, doctorRow{Scope: "global", Check: "topology", Status: "fail", Details: errorValue.Error()})
		doctorIssues = append(doctorIssues, "topology: "+errorValue.Error())
	} else {
		globalRows = append(globalRows, doctorRow{Scope: "global", Check: "topology", Status: "ok", Details: "dependency graph resolved"})
	}

	if _, errorValue := engine.BuildEffectiveEnvironments(runtimeRoot, normalized.Environment); errorValue != nil {
		globalRows = append(globalRows, doctorRow{Scope: "global", Check: "environment", Status: "fail", Details: errorValue.Error()})
		doctorIssues = append(doctorIssues, "environment: "+errorValue.Error())
	} else {
		globalRows = append(globalRows, doctorRow{Scope: "global", Check: "environment", Status: "ok", Details: "effective env built for " + normalized.Environment})
	}

	nodeRows := make([]doctorRow, 0)
	nodes := engine.CollectRuntimeNodes(runtimeRoot)
	sort.Slice(nodes, func(left int, right int) bool {
		return nodes[left].Node.GetID() < nodes[right].Node.GetID()
	})

	for _, runtimeEntry := range nodes {
		if !engine.ShouldOperateOnNode(runtimeEntry.Node, normalized.Tags) {
			continue
		}

		invalidManeuvers := make([]string, 0)
		for _, maneuver := range engine.ListNodeManeuvers(runtimeEntry.Node) {
			if len(maneuver.Action) == 0 {
				invalidManeuvers = append(invalidManeuvers, maneuver.Call)
			}
		}

		row := doctorRow{
			Scope: runtimeEntry.Node.GetID(),
			Check: "node",
		}
		if len(invalidManeuvers) > 0 {
			row.Status = "fail"
			row.Details = "empty maneuver action: " + strings.Join(invalidManeuvers, ",")
			doctorIssues = append(doctorIssues, fmt.Sprintf("%s: %s", runtimeEntry.Node.GetID(), row.Details))
		} else {
			maneuverCount := len(engine.ListNodeManeuvers(runtimeEntry.Node))
			row.Status = "ok"
			row.Details = fmt.Sprintf("role=%s reactor=%s maneuvers=%d", resolveRoleLabel(runtimeEntry.Node), resolveReactorLabel(runtimeEntry.Node), maneuverCount)
		}

		nodeRows = append(nodeRows, row)
	}

	for _, line := range renderDoctorTable(globalRows, nodeRows, normalized) {
		engine.Info("%s", line)
	}

	if len(doctorIssues) > 0 {
		return &domain.ValidationError{Message: "doctor found issues: " + strings.Join(doctorIssues, "; ")}
	}

	engine.Success("doctor check passed for %s", runtimeRoot.Node.GetID())
	return nil
}

type doctorRow struct {
	Scope   string
	Check   string
	Status  string
	Details string
}

func renderDoctorTable(globalRows []doctorRow, nodeRows []doctorRow, options OperationOptions) []string {
	columns := []widgets.TableColumn{
		{Header: "SCOPE", MaxWidth: 32},
		{Header: "CHECK", MaxWidth: 14},
		{Header: "STATUS", MaxWidth: 8},
		{Header: "DETAILS", MaxWidth: 84},
	}

	data := make([][]string, 0, len(globalRows)+len(nodeRows))
	for _, row := range globalRows {
		data = append(data, []string{row.Scope, row.Check, row.Status, row.Details})
	}
	for _, row := range nodeRows {
		data = append(data, []string{row.Scope, row.Check, row.Status, row.Details})
	}

	lines := make([]string, 0, len(data)+2)
	lines = append(lines, "doctor: env="+options.Environment+" tags="+strings.Join(options.Tags, ","))
	lines = append(lines, widgets.RenderTable(columns, data)...)

	return lines
}
