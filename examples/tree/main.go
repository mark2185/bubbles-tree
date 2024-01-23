package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	tree "github.com/mark2185/bubbles-tree"
)

const RootPath = "/tmp"

type node struct {
	parent      *node
	path        string
	permissions string
	uid         string
	gid         string
	size        string
	state       tree.NodeState
	children    []*node
}

// to ensure it implements the interface
// TODO: does this even make sense? I'm not the go expert around here
var _ tree.Node = (*node)(nil)

func (n node) Prefix() string {
	return fmt.Sprintf("%s %s:%s %s", n.permissions, n.uid, n.gid, n.size)
}

func (n node) Parent() tree.Node {
	if n.parent == nil {
		return nil
	}
	return n.parent
}

func (n *node) SetState(st tree.NodeState) {
	n.state = st
}

const (
	Collapsed = ">"
	Expanded  = "v"
)

func (n node) Name() string {
	name := filepath.Base(n.path)
	if n.parent == nil {
		name = n.path
	}

	hints := n.state
	annotation := ""
	s := strings.Builder{}
	if hints&tree.NodeCollapsible == tree.NodeCollapsible {
		annotation = Expanded
		if hints&tree.NodeCollapsed == tree.NodeCollapsed {
			annotation = Collapsed
		}
	}

	if len(annotation) > 0 {
		fmt.Fprintf(&s, "%-2s%s", annotation, name)
	} else {
		fmt.Fprintf(&s, "%s", name)
	}

	return s.String()
}

func (n node) Children() tree.Nodes {
	nodes := make(tree.Nodes, len(n.children))
	for i, n := range n.children {
		nodes[i] = n
	}
	return nodes
}

func (n node) State() tree.NodeState {
	return n.state
}

func isUnixHiddenFile(name string) bool {
	return len(name) > 2 && (name[0] == '.' || name[:2] == "..")
}

func buildNodeTree(root string, maxDepth int) tree.Nodes {
	allNodes := make([]*node, 0)

	rootPath := func(p string) string {
		if p == "." {
			return root
		}
		return p
	}
	_ = fs.WalkDir(os.DirFS(root), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if isUnixHiddenFile(d.Name()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		cnt := len(strings.Split(p, string(os.PathSeparator)))
		if maxDepth != -1 && cnt > maxDepth {
			return fs.SkipDir
		}

		st := tree.NodeNone
		if d.IsDir() {
			st |= tree.NodeCollapsible
		}
		p = rootPath(p)
		parent := findNodeByPath(allNodes, rootPath(filepath.Dir(p)))

		node := &node{
			path:        p,
			state:       st,
			uid:         "1000",
			gid:         "1000",
			permissions: "-rwxrwxrwx",
		}

		if parent == nil {
			allNodes = append(allNodes, node)
		} else {
			node.parent = parent
			node.state |= tree.NodeCollapsed
			parent.children = append(parent.children, node)
		}
		return nil
	})

	nodes := make(tree.Nodes, len(allNodes))
	for i, n := range allNodes {
		nodes[i] = n
	}
	return nodes
}

func findNodeByPath(nodes []*node, path string) *node {
	for _, node := range nodes {
		if filepath.Clean(node.path) == filepath.Clean(path) {
			return node
		}
		if child := findNodeByPath(node.children, path); child != nil {
			return child
		}
	}
	return nil
}

type quittingTree struct {
	tree.Model
}

func (e quittingTree) Init() tea.Cmd {
	return e.Model.Init()
}

func (e quittingTree) Name() string {
	return e.Model.View()
}

func (e quittingTree) Update(m tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := m.(tea.KeyMsg); ok && key.Matches(msg, key.NewBinding(key.WithKeys("q"))) {
		return e, tea.Quit
	}
	var cmd tea.Cmd
	e.Model, cmd = e.Model.Update(m)
	return e, cmd
}

func main() {
	var depth int
	var style string
	flag.IntVar(&depth, "depth", 10, "The maximum depth to read the directory structure")
	flag.StringVar(&style, "style", "normal", "The style to use when drawing the tree: double, thick, rounded, edge, normal")
	flag.Parse()

	symbols := tree.DefaultSymbols()
	switch style {
	case "thick":
		symbols = tree.ThickSymbols()
	case "rounded":
		symbols = tree.RoundedSymbols()
	case "double":
		symbols = tree.DoubleSymbols()
	case "edge":
		symbols = tree.NormalEdgeSymbols()
	case "thickedge":
		symbols = tree.ThickEdgeSymbols()
	case "", "normal":
	default:
		fmt.Fprintf(os.Stderr, "invalid style type, using default 'normal'\n")
	}

	path := RootPath
	if flag.NArg() > 0 {
		abs, err := filepath.Abs(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
		path = abs
	}

	t := tree.New(buildNodeTree(path, depth))
	t.Symbols = symbols
	m := quittingTree{Model: t}

	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
