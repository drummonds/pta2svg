package layout

import (
	"github.com/drummonds/pta2svg/internal/model"
	"github.com/nulab/autog"
	"github.com/nulab/autog/graph"
)

// Sugiyama uses the nulab/autog layered layout algorithm.
// The autog engine produces a top-to-bottom layout; we swap X/Y
// to get a left-to-right flow matching the existing LR convention.
type Sugiyama struct{}

func (Sugiyama) Layout(d *model.Diagram, opts LayoutOptions) *Graph {
	g := &Graph{
		NodeMap: make(map[string]*Node),
	}

	// Collect edges for autog. Each movement becomes a directed edge.
	// autog creates nodes implicitly from edge endpoints.
	var edges [][]string
	seen := make(map[[2]string]bool)
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			key := [2]string{m.From.Name, m.To.Name}
			if !seen[key] {
				edges = append(edges, []string{m.From.Name, m.To.Name})
				seen[key] = true
			}
		}
	}

	// Guard: autog panics on empty graph
	if len(edges) == 0 {
		return g
	}

	// Build per-node sizes. autog layout is top-down, so we swap W/H
	// (node width in TD becomes height in LR, and vice versa).
	sizes := make(map[string]graph.Size, len(d.Accounts))
	for name := range d.Accounts {
		sizes[name] = graph.Size{W: opts.NodeH, H: opts.NodeW}
	}

	result := autog.Layout(
		graph.EdgeSlice(edges),
		autog.WithNodeSize(sizes),
		autog.WithLayerSpacing(opts.PadX),
		autog.WithNodeSpacing(opts.PadY),
		autog.WithPositioning(autog.PositioningBrandesKoepf),
		autog.WithEdgeRouting(autog.EdgeRoutingPolyline),
	)

	// Convert autog nodes → layout.Node, swapping X/Y for LR.
	for _, n := range result.Nodes {
		acct := d.Accounts[n.ID]
		if acct == nil {
			continue
		}
		node := &Node{
			Account: acct,
			X:       opts.MarginX + n.Y, // swap: autog Y → our X
			Y:       opts.MarginY + n.X, // swap: autog X → our Y
			W:       n.H,                // swap: autog H → our W
			H:       n.W,                // swap: autog W → our H
		}
		g.Nodes = append(g.Nodes, node)
		g.NodeMap[n.ID] = node
	}

	// Convert autog edges → layout.Edge with routed points.
	idx := 0
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			fromNode := g.NodeMap[m.From.Name]
			toNode := g.NodeMap[m.To.Name]
			if fromNode == nil || toNode == nil {
				continue
			}

			// Find matching autog edge for route points
			var points [][2]float64
			var arrowStart bool
			for _, ae := range result.Edges {
				if ae.FromID == m.From.Name && ae.ToID == m.To.Name {
					// Swap X/Y in each point
					points = make([][2]float64, len(ae.Points))
					for i, p := range ae.Points {
						points[i] = [2]float64{opts.MarginX + p[1], opts.MarginY + p[0]}
					}
					arrowStart = ae.ArrowHeadStart
					break
				}
			}

			g.Edges = append(g.Edges, Edge{
				From:           fromNode,
				To:             toNode,
				Movement:       m,
				Index:          idx,
				Points:         points,
				ArrowHeadStart: arrowStart,
			})
			idx++
		}
	}

	// Compute canvas size from node positions
	maxX, maxY := 0.0, 0.0
	for _, n := range g.Nodes {
		if n.X+n.W > maxX {
			maxX = n.X + n.W
		}
		if n.Y+n.H > maxY {
			maxY = n.Y + n.H
		}
	}
	g.Width = maxX + opts.MarginX
	g.Height = maxY + opts.MarginY

	// Populate balances and commodities
	for name, bal := range d.Balances {
		if n, ok := g.NodeMap[name]; ok {
			n.Balance = bal
		}
	}
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			if m.Commodity == "" {
				continue
			}
			if n, ok := g.NodeMap[m.From.Name]; ok && n.Commodity == "" {
				n.Commodity = m.Commodity
			}
			if n, ok := g.NodeMap[m.To.Name]; ok && n.Commodity == "" {
				n.Commodity = m.Commodity
			}
		}
	}

	return g
}
