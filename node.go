package tree

type NodeState uint16

// Node represents the base model for the elements of the Treeish implementation
type Node interface {
	// Name should return the name of the node
	Name() string
	// Prefix should return the metadata of the file, e.g. permissions, size, uid, gid
	Prefix() string
	// Parent should return the parent of the current node, or nil if a root node.
	Parent() Node
	// Children should return a list of Nodes which represent the children of the current node.
	Children() Nodes
	// State should return the annotation for the current node, which are used for computing various display states.
	State() NodeState
	// SetState self-explanatory
	SetState(NodeState)
}

// Nodes is a slice of Node elements, usually representing the children of a Node.
type Nodes []Node

const (
	NodeNone NodeState = 0

	// NodeSelected hints that the current node should be rendered as selected
	NodeSelected = 1 << iota
	// NodeCollapsible hints that the current node can be collapsed
	NodeCollapsible
	// NodeCollapsed hints that the current node is collapsed
	NodeCollapsed
	// NodeHidden hints that the current node is not going to be displayed, e.g. when applying filters
	NodeHidden
	// NodeLastChild shows the node to be the last in the children list
	NodeLastChild
	// NodeHasPreviousSibling shows if the node has siblings
	NodeHasPreviousSibling
)

// at returns the i-th non hidden node
// should be the same as ns.flatten()[i], but more performant (exits early)
func (ns Nodes) at(i int) Node {
	j := 0
	for _, n := range ns {
		if isHidden(n) {
			continue
		}
		if j == i {
			return n
		}

		if isExpanded(n) {
			if nn := n.Children().at(i - j - 1); nn != nil {
				return nn
			}
			j += countNodesBelow(n)
		}
		j++
	}
	return nil
}

// countNodesBelow returns the number of all nodes below the given one
func countNodesBelow(n Node) int {
	count := 0
	for _, child := range n.Children() {
		count += countNodesBelow(child)
	}
	return count
}

// getDepth traverses through Parents (upwards) until it reaches nil
func getDepth(n Node) int {
	d := 0
	for {
		if n == nil || n.Parent() == nil {
			break
		}
		d++
		n = n.Parent()
	}
	return d
}

// flatten returns a flat slice of all non-hidden and expanded Nodes
func (ns Nodes) flatten() Nodes {
	res := Nodes{}
	for _, n := range ns {
		if isHidden(n) {
			continue
		}
		res = append(res, n)
		if isCollapsible(n) && isExpanded(n) {
			res = append(res, n.Children().flatten()...)
		}
	}
	return res
}

// Is checks if the given state is set
func (s NodeState) Is(st NodeState) bool {
	return s&st == st
}

func isHidden(n Node) bool {
	return n.State().Is(NodeHidden)
}

func isExpanded(n Node) bool {
	return !n.State().Is(NodeCollapsed)
}

func isCollapsible(n Node) bool {
	return n.State().Is(NodeCollapsible)
}

func isLastNode(n Node) bool {
	return n.State().Is(NodeLastChild)
}

func isSelected(n Node) bool {
	return n.State().Is(NodeSelected)
}

func hasPreviousSibling(n Node) bool {
	return n.State().Is(NodeHasPreviousSibling)
}

func hasChildren(n Node) bool {
	return len(n.Children()) > 0
}
