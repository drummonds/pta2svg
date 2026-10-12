package layout

import (
	"sort"

	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

// LR lays out accounts in left-to-right columns grouped by type.
type LR struct{}

// columnOrder defines L→R ordering of account types.
// Sources of funds on the left, Asset in the middle, Expense on the right.
var columnOrder = []model.AccountType{
	model.Equity,
	model.Income,
	model.Liability,
	model.Hub,
	model.Asset,
	model.Expense,
	model.External,
}

func (LR) Layout(d *model.Diagram, opts LayoutOptions) *Graph {
	g := &Graph{
		NodeMap: make(map[string]*Node),
	}

	// Group accounts by type
	groups := make(map[model.AccountType][]*model.Account)
	for _, a := range d.Accounts {
		groups[a.Type] = append(groups[a.Type], a)
	}
	// Sort within each group for deterministic output
	for _, accts := range groups {
		sort.Slice(accts, func(i, j int) bool {
			return accts[i].Name < accts[j].Name
		})
	}

	// Place nodes column by column
	col := 0
	maxY := 0.0
	for _, typ := range columnOrder {
		accts := groups[typ]
		if len(accts) == 0 {
			continue
		}
		for row, a := range accts {
			x := opts.MarginX + float64(col)*(opts.NodeW+opts.PadX)
			y := opts.MarginY + float64(row)*(opts.NodeH+opts.PadY)
			n := &Node{
				Account: a,
				X:       x,
				Y:       y,
				W:       opts.NodeW,
				H:       opts.NodeH,
			}
			g.Nodes = append(g.Nodes, n)
			g.NodeMap[a.Name] = n
			if y+opts.NodeH > maxY {
				maxY = y + opts.NodeH
			}
		}
		col++
	}

	// Compute canvas size
	g.Width = opts.MarginX*2 + float64(col)*(opts.NodeW+opts.PadX) - opts.PadX
	g.Height = maxY + opts.MarginY

	// Build edges
	idx := 0
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			fromNode := g.NodeMap[m.From.Name]
			toNode := g.NodeMap[m.To.Name]
			if fromNode == nil || toNode == nil {
				continue
			}
			g.Edges = append(g.Edges, Edge{
				From:     fromNode,
				To:       toNode,
				Movement: m,
				Index:    idx,
			})
			idx++
		}
	}
	routeAroundBoxes(g, opts)

	// Populate balances and commodities from diagram
	for name, bal := range d.Balances {
		if n, ok := g.NodeMap[name]; ok {
			n.Balance = bal
		}
	}
	// Infer commodity per node from movements
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

// routeAroundBoxes gives an arrow whose curve would pass through a box in
// a column between its ends a route around them instead, along a lane of
// its own below the boxes it passes. An end that is the lowest box in its
// column is left or entered through its bottom edge; otherwise the route
// runs through the gap beside it, so it never crosses a box beneath. The
// canvas grows to hold the lanes and their labels.
func routeAroundBoxes(g *Graph, opts LayoutOptions) {
	const laneStep = 36.0 // room for a two-line label between lanes
	lane := 0
	for i := range g.Edges {
		e := &g.Edges[i]
		sx, sy := e.From.X+e.From.W, e.From.Y+e.From.H/2
		ex, ey := e.To.X, e.To.Y+e.To.H/2
		if ex <= sx {
			continue // backward arrows keep their arc
		}
		top, bottom := min(sy, ey), max(sy, ey)
		blocked, lowest := false, 0.0
		for _, n := range g.Nodes {
			if n.X < sx || n.X+n.W > ex {
				continue
			}
			lowest = max(lowest, n.Y+n.H)
			if n.Y < bottom && n.Y+n.H > top {
				blocked = true
			}
		}
		if !blocked {
			continue
		}
		fromBelow, intoBelow := lowestInColumn(g, e.From), lowestInColumn(g, e.To)
		if fromBelow {
			lowest = max(lowest, e.From.Y+e.From.H)
		}
		if intoBelow {
			lowest = max(lowest, e.To.Y+e.To.H)
		}
		laneY := lowest + opts.PadY/2 + float64(lane)*laneStep
		lane++

		var pts [][2]float64
		if fromBelow {
			cx := e.From.X + e.From.W/2
			pts = append(pts, [2]float64{cx, e.From.Y + e.From.H}, [2]float64{cx, laneY})
		} else {
			out := sx + opts.PadX/2
			pts = append(pts, [2]float64{sx, sy}, [2]float64{out, sy}, [2]float64{out, laneY})
		}
		if intoBelow {
			cx := e.To.X + e.To.W/2
			pts = append(pts, [2]float64{cx, laneY}, [2]float64{cx, e.To.Y + e.To.H})
		} else {
			in := ex - opts.PadX/2
			pts = append(pts, [2]float64{in, laneY}, [2]float64{in, ey}, [2]float64{ex, ey})
		}
		e.Points = pts
		g.Height = max(g.Height, laneY+laneStep/2+opts.MarginY/2)
	}
}

// lowestInColumn reports whether no box in n's column sits below it.
func lowestInColumn(g *Graph, n *Node) bool {
	for _, o := range g.Nodes {
		if o.X == n.X && o.Y > n.Y {
			return false
		}
	}
	return true
}
