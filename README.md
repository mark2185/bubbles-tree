## Tree

A Charm bubbletea model for a representation of a tree-like structure, I had a filesystem in mind when creating this.

The model supports out of the box navigating through the tree using the directional keys (and `hjkl`) and also expanding/collapsing directory nodes using `Enter`.

Different symbols and lipgloss styles can be configured for the basic elements of the tree.

## Examples

Check the `examples` folder where we have a minimal filesystem tree utility.

### [Tree](./examples/tree)

```sh
$ go run main.go -depth 3 ../../
 └─ ⊟ /some/path/bubbles-tree
    ├─   boilerplate.go
    ├─ ⊟ examples
    │  └─ ⊟ tree
    │     ├─   go.mod
    │     ├─   go.sum
    │     ├─   go.work
    │     └─   main.go
    ├─   go.mod
    ├─   go.sum
    ├─   node.go
    ├─   README.md
    ├─   symbols.go
    ├─   tree.go
    └─   tree_test.go
```
