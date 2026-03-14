package widgets

import (
	"sort"
	"strings"
)

// TreeNode is a generic tree render node.
type TreeNode struct {
	Label    string
	Children []TreeNode
}

// RenderTree renders children of root into an ASCII tree.
// It returns "(none)" when root has no children.
func RenderTree(root TreeNode) string {
	if len(root.Children) == 0 {
		return "(none)"
	}

	children := append([]TreeNode(nil), root.Children...)
	sort.Slice(children, func(left int, right int) bool {
		return children[left].Label < children[right].Label
	})

	lines := make([]string, 0)
	for index, child := range children {
		appendTreeLines(&lines, child, "", index == len(children)-1)
	}

	return strings.Join(lines, "\n")
}

func appendTreeLines(lines *[]string, node TreeNode, prefix string, isLast bool) {
	branch := "|- "
	nextPrefix := prefix + "|  "
	if isLast {
		branch = "`- "
		nextPrefix = prefix + "   "
	}

	*lines = append(*lines, prefix+branch+node.Label)

	children := append([]TreeNode(nil), node.Children...)
	sort.Slice(children, func(left int, right int) bool {
		return children[left].Label < children[right].Label
	})

	for index, child := range children {
		appendTreeLines(lines, child, nextPrefix, index == len(children)-1)
	}
}
