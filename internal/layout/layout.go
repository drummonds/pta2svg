package layout

import (
	"github.com/drummonds/pta2svg/internal/model"
)

// Node represents a positioned account box in the SVG.
type Node struct {
	Account *model.Account
	X, Y    float64
	W, H    float64
}

// Edge represents a positioned flow arrow.
type Edge struct {
	From, To *Node
	Movement model.Movement
	Index    int // for staggered animation delay
}

// Graph holds the positioned nodes and edges for rendering.
type Graph struct {
	Nodes   []*Node
	Edges   []Edge
	Width   float64
	Height  float64
	NodeMap map[string]*Node // keyed by account name
}

// LayoutOptions configures the layout engine.
type LayoutOptions struct {
	NodeW   float64
	NodeH   float64
	PadX    float64
	PadY    float64
	MarginX float64
	MarginY float64
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() LayoutOptions {
	return LayoutOptions{
		NodeW:   140,
		NodeH:   50,
		PadX:    80,
		PadY:    30,
		MarginX: 40,
		MarginY: 40,
	}
}

// Layouter positions nodes and routes edges.
type Layouter interface {
	Layout(d *model.Diagram, opts LayoutOptions) *Graph
}

// GraphFromDiagram builds a Graph using the given Layouter.
func GraphFromDiagram(d *model.Diagram, l Layouter, opts LayoutOptions) *Graph {
	return l.Layout(d, opts)
}
