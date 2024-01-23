package tree

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"golang.org/x/exp/constraints"
)

// Model is the Bubble Tea model for this user interface.
type Model struct {
	root  Node
	nodes Nodes // all nodes

	view viewport.Model

	focus  bool // could be useful, currently unused
	cursor int

	KeyMap  KeyMap
	Styles  Styles
	Symbols Symbols
}

// New initializes a new Model
// It sets the default content, keymap, styles, and symbols.
func New(ns Nodes) Model {
	// TODO: maybe assert that Nodes isn't empty or something
	root := ns[0]
	root.SetState(root.State() | NodeSelected) // we're selecting the first row by default

	m := Model{
		root:  root,
		nodes: ns.flatten(),

		view: viewport.New(0, 0),

		KeyMap:  DefaultKeyMap(),
		Styles:  DefaultStyles(),
		Symbols: DefaultSymbols(),
	}

	// rendering all nodes, every single one of them expanded as the inital state
	initialContent := m.renderAllNodes()
	m.view.SetContent(
		lipgloss.JoinVertical(lipgloss.Left, initialContent...),
	)

	return m
}

// just to wrap my head around it easier
var noop tea.Cmd = nil

func (m Model) Init() tea.Cmd {
	return noop
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		// TODO: never actually rendered, but might be useful one day
		return m, noop
	}

	var cmd tea.Cmd = nil
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
		// TODO: what if the screen shrinks and the currently selected node
		// isn't visible anymore?
		return m, nil
	case tea.KeyMsg:
		// so we can toggle it if need be
		previouslySelectedNode := m.cursor

		switch {
		case key.Matches(msg, m.KeyMap.Expand):
			// this requires rerendering all of the nodes
			m.ToggleExpand()
			renderedRows := m.renderAllNodes()
			m.view.SetContent(
				lipgloss.JoinVertical(lipgloss.Left, renderedRows...),
			)
			return m, noop
		case key.Matches(msg, m.KeyMap.LineUp):
			cmd = m.MoveUp(1)
		case key.Matches(msg, m.KeyMap.LineDown):
			cmd = m.MoveDown(1)
		case key.Matches(msg, m.KeyMap.PageUp):
			cmd = m.MoveUp(m.view.Height)
		case key.Matches(msg, m.KeyMap.PageDown):
			cmd = m.MoveDown(m.view.Height)
		case key.Matches(msg, m.KeyMap.HalfPageUp):
			cmd = m.MoveUp(m.view.Height / 2)
		case key.Matches(msg, m.KeyMap.HalfPageDown):
			cmd = m.MoveDown(m.view.Height / 2)
		case key.Matches(msg, m.KeyMap.GotoTop):
			cmd = m.GotoTop()
		case key.Matches(msg, m.KeyMap.GotoBottom):
			cmd = m.GotoBottom()
		}

		newlySelectedNode := m.cursor
		// TODO: this requires a viewport fork
		m.view.ReplaceLine(previouslySelectedNode, m.renderNode(m.nodes.at(previouslySelectedNode)))
		m.view.ReplaceLine(newlySelectedNode, m.renderNode(m.nodes.at(newlySelectedNode)))
	}

	return m, cmd
}

func (m Model) View() string {
	return m.view.View()
}

func (m *Model) setCursor(newCursorPos int) tea.Cmd {
	// nothing changes if nothing changes
	if cursorNotMoved := newCursorPos == m.cursor; cursorNotMoved {
		return noop
	}

	// deselect the old one
	previous := m.currentNode()
	// TODO: this should actually be AND with !NodeSelected, but go complains
	// that ^NodeSelected overflows
	previous.SetState(previous.State() ^ NodeSelected)

	// move cursor
	m.cursor = newCursorPos

	// select the new one
	current := m.currentNode()
	current.SetState(current.State() | NodeSelected)

	return noop
}

// currentNode returns the currently selected node.
func (m Model) currentNode() Node {
	return m.nodes.at(m.cursor)
}

func (m Model) AllNodes() Nodes {
	return m.nodes
}

// MoveUp moves the selection up by any number of rows.
// It can not go above the first row.
func (m *Model) MoveUp(n int) tea.Cmd {
	if cursorAtTop := m.cursor == 0; cursorAtTop {
		return noop
	}

	minCursorPos := 0
	newCursorPos := max(m.cursor-n, minCursorPos)

	viewTop, _ := m.view.VisibleLineIndices()
	if cursorBrokeLimit := newCursorPos < viewTop; cursorBrokeLimit {
		// gotta move the view to follow the cursor
		m.view.LineUp(newCursorPos - m.cursor)
	}
	return m.setCursor(newCursorPos)
}

// MoveDown moves the selection down by any number of rows.
// It can not go below the last row.
func (m *Model) MoveDown(n int) tea.Cmd {
	maxCursorPos := m.view.TotalLineCount() - 1
	if cursorAtBottom := m.cursor == maxCursorPos; cursorAtBottom {
		return noop
	}

	newCursorPos := min(m.cursor+n, maxCursorPos)

	_, viewBottom := m.view.VisibleLineIndices()
	if cursorBrokeLimit := viewBottom < newCursorPos; cursorBrokeLimit {
		// gotta move the view to follow the cursor
		m.view.LineDown(newCursorPos - m.cursor)
	}
	return m.setCursor(newCursorPos)
}

// GotoTop moves the selection to the first row.
func (m *Model) GotoTop() tea.Cmd {
	return m.MoveUp(m.view.TotalLineCount())
}

// GotoBottom moves the selection to the last row.
func (m *Model) GotoBottom() tea.Cmd {
	return m.MoveDown(m.view.TotalLineCount())
}

// ToggleExpand toggles the expanded state of the node pointed at by m.cursor
func (m *Model) ToggleExpand() {
	n := m.currentNode()
	if !isCollapsible(n) {
		return
	}
	n.SetState(n.State() ^ NodeCollapsed)
}

// SetWidth sets the width of the viewport of the tree.
func (m *Model) SetWidth(w int) {
	m.view.Width = w
}

// SetHeight sets the height of the viewport of the tree.
func (m *Model) SetHeight(h int) {
	// TODO: make sure the currently selected node is still visible
	m.view.Height = h
}

// Height returns the viewport height of the tree.
func (m Model) Height() int {
	return m.view.Height
}

// Width returns the viewport width of the tree.
func (m Model) Width() int {
	return m.view.Width
}

// YOffset returns the viewport vertical scroll position of the tree.
func (m Model) YOffset() int {
	return m.view.YOffset
}

// SetYOffset sets Y offset of the tree's viewport.
func (m *Model) SetYOffset(n int) {
	m.view.SetYOffset(n)
}

// ScrollPercent returns the amount scrolled as a float between 0 and 1.
func (m Model) ScrollPercent() float64 {
	return m.view.ScrollPercent()
}

// Cursor returns the index of the selected row.
func (m Model) Cursor() int {
	return m.cursor
}

// TODO: put this in some utilities file maybe
// btw it's copied from samber/lo
func clamp[T constraints.Ordered](value T, min T, max T) T {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

// Focused returns the focus state of the tree.
func (m Model) Focused() bool {
	return m.focus
}

// Focus focuses the tree, allowing the user to move around the tree nodes.
// interact.
func (m *Model) Focus() {
	m.focus = true
}

// Blur blurs the tree, preventing selection or movement.
func (m *Model) Blur() {
	current := m.currentNode()
	current.SetState(current.State() ^ NodeSelected)
	m.focus = false
}

// When we render the tree symbols we consider them as a grid of maxDepth width
// Each pos in the grid corresponds to a space or a tree-depth-indicating symbol
// TODO: good luck
func (m Model) getTreeSymbolForPos(n Node, pos int, maxDepth int) string {
	if n == nil {
		// TODO: find out how can this happen? ( Luka M. 2024-01-21 )
		panic("getting tree symbol for nil node")
	}
	s := m.Styles.Symbol
	if hasPaddingAtPos(n, pos, maxDepth) {
		return Padding(s, m.Symbols, pos)
	}
	if pos < maxDepth {
		return RenderConnector(s, m.Symbols, pos)
	}
	if isLastNode(n) {
		return RenderTerminator(s, m.Symbols, pos)
	}
	return RenderStarter(s, m.Symbols, pos)
}

// hasPaddingAtPos computes if a node of given given depth needs padding in the tree-like view
// TODO: good luck
func hasPaddingAtPos(n Node, depth int, maxDepth int) bool {
	if n == nil {
		return true
	}
	if depth > maxDepth {
		return true
	}
	if depth == maxDepth {
		return false
	}
	parentInPos := maxDepth - depth
	for i := 0; i < parentInPos; i++ {
		if n = n.Parent(); n == nil {
			return true
		}
	}
	return isLastNode(n)
}

// TODO: good luck
func (m Model) renderSymbolsForSingleLineNode(n Node) string {
	nodeDepth := getDepth(n)

	prefix := strings.Builder{}
	for pos := 0; pos <= nodeDepth; pos++ {
		prefix.WriteString(m.getTreeSymbolForPos(n, pos, nodeDepth))
	}
	return prefix.String()
}

// TODO: good luck
func (m Model) renderPrefixForMultiLineNode(t Node, lineCount int) string {
	maxDepth := getDepth(t)

	s := m.Styles.Symbol

	prefix := strings.Builder{}

	connectsBottom := isLastNode(t)
	for line := 0; line < lineCount; line++ {
		for lvl := 0; lvl <= maxDepth-1; lvl++ {
			prefix.WriteString(m.getTreeSymbolForPos(t, lvl, maxDepth))
		}
		if line == 0 {
			prefix.WriteString(RenderStarter(s, m.Symbols, maxDepth))
			if lineCount > 1 {
				prefix.WriteRune('\n')
			}
		} else if line == lineCount-1 {
			if !connectsBottom {
				prefix.WriteString(RenderTerminator(s, m.Symbols, maxDepth))
			} else {
				prefix.WriteString(RenderConnector(s, m.Symbols, maxDepth))
			}
		} else {
			prefix.WriteString(RenderConnector(s, m.Symbols, maxDepth))
			prefix.WriteRune('\n')
		}
	}

	return prefix.String()
}

// TODO: good luck
func (m *Model) render() []string {
	if m.view.Height+m.view.Width == 0 {
		return nil
	}

	return m.renderNodes(m.AllNodes())
}

const Ellipsis = "…"

// SetStyles sets the tree Styles.
func (m *Model) SetStyles(s Styles) {
	m.Styles = s
}

// TODO: good luck
func (m *Model) renderNode(n Node) string {
	if n == nil {
		// TODO: find out how can this happen? ( Luka M. 2024-01-21 )
		panic("trying to render nil node")
		// return ""
	}

	// TODO: multiline content issue will be solved when viewport gets horizontal scrolling (https://github.com/charmbracelet/bubbles/issues/145)
	// the prefix consists of custom Prefix function + tree-like symbols (depth, branching)
	prefix := n.Prefix() + m.renderSymbolsForSingleLineNode(n)

	prefixWidth := lipgloss.Width(prefix)
	nameWidth := m.Width() - prefixWidth
	style := m.Styles.Line
	if isSelected(n) {
		style = m.Styles.Selected
	}
	render := style.Width(nameWidth).MaxWidth(nameWidth - 1).Render
	name := n.Name()
	if lipgloss.Width(name) > nameWidth {
		name = truncate.StringWithTail(name, uint(nameWidth-1), Ellipsis)
	}
	node := lipgloss.JoinHorizontal(lipgloss.Left, prefix, render(name))
	// TODO: I don't like this approach, renderNode should render only the given node!
	// if isExpanded(n) && hasChildren(n) {
	// renderedChildren := m.renderNodes(n.Children())
	// node = lipgloss.JoinVertical(lipgloss.Top, node, lipgloss.JoinVertical(lipgloss.Left, renderedChildren...))
	// }

	return node
}

// renderAllNodes returns a string representation for each node
// both the prefix, tree-like symbols and name, omitting hidden nodes
// TODO: good luck
func (m Model) renderAllNodes() []string {
	return m.renderNodes(m.AllNodes())
}

// TODO: good luck
func (m Model) renderNodes(ns Nodes) []string {
	rendered := []string{}
	for i, n := range ns {
		if isHidden(n) {
			continue
		}

		hints := NodeNone
		if i > 0 {
			hints |= NodeHasPreviousSibling
		}
		if hasChildren(n) {
			hints |= NodeCollapsible
		}
		if i == len(ns)-1 {
			hints |= NodeLastChild
		}

		n.SetState(n.State() | hints)
		if out := m.renderNode(n); len(out) > 0 {
			rendered = append(rendered, out)
		}
	}

	return rendered
}
