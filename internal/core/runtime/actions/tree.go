package actions

import (
	"fmt"
	"strings"

	"github.com/OctalMesh/Commodore/internal/ui/widgets"
)

// Tree logs discovered subordinate hierarchy.
func Tree(engine Engine) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	runtimeRoot := engine.Root()
	structure := widgets.RenderTree(buildTreeWidgetRoot(runtimeRoot))
	if structure == "(none)" {
		engine.Info("(none)")
		return nil
	}

	rootLabel := runtimeRoot.LocalID
	if rootLabel == "" {
		rootLabel = runtimeRoot.Node.GetID()
	}
	if runtimeRoot.Node.GetID() != rootLabel {
		rootLabel = fmt.Sprintf("%s (%s)", rootLabel, runtimeRoot.Node.GetID())
	}

	engine.Info("%s", rootLabel)
	for _, line := range strings.Split(structure, "\n") {
		engine.Info("%s", line)
	}
	return nil
}

func buildTreeWidgetRoot(rootRuntime *RuntimeNode) widgets.TreeNode {
	if rootRuntime == nil {
		return widgets.TreeNode{}
	}

	rootLabel := rootRuntime.LocalID
	if rootLabel == "" {
		rootLabel = rootRuntime.Node.GetID()
	}

	children := make([]widgets.TreeNode, 0, len(rootRuntime.Children))
	for _, child := range rootRuntime.Children {
		children = append(children, buildTreeWidgetNode(child))
	}

	return widgets.TreeNode{Label: rootLabel, Children: children}
}

func buildTreeWidgetNode(node *RuntimeNode) widgets.TreeNode {
	if node == nil {
		return widgets.TreeNode{}
	}

	label := node.LocalID
	if node.LocalID != node.Node.GetID() {
		label = fmt.Sprintf("%s (%s)", node.LocalID, node.Node.GetID())
	}

	children := make([]widgets.TreeNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, buildTreeWidgetNode(child))
	}

	return widgets.TreeNode{Label: label, Children: children}
}
