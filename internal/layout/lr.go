package layout

import (
	"cmp"
	"slices"
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
	routeArrows(g, opts)

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

// A box side an arrow meets.
type side int

const (
	rightSide side = iota
	leftSide
	bottomSide
)

// end is where an arrow meets a box: the side, and the order along it
// (the other end's position, so arrows fan out without crossing).
type end struct {
	node *Node
	side side
	key  float64
	at   [2]float64
}

// track is a vertical run through the gap beside a column, kept close to
// the column side it serves: edge is that side's x, dir -1 for the gap to
// its left and +1 for the gap to its right.
type track struct {
	edge float64
	dir  float64
	key  float64
	x    float64
}

// path is an arrow's route before its ends and tracks are placed: a
// straight hop, a bracket through one track, or a run along a lane below
// the boxes, reached by the bottom of a box or by a track.
type path struct {
	edge           *Edge
	from, to       *end
	bracket        *track
	lane           bool
	laneY          float64
	fromRun, toRun *track
}

// routeArrows gives every arrow its route. A forward arrow with nothing in
// the way is a straight hop between facing sides. An arrow to a box in its
// own column is a bracket through the gap to the column's left. An arrow
// back to an earlier column, or forward through a box in a column between,
// runs along a lane of its own below the boxes it passes, leaving and
// entering through a box's bottom when nothing sits beneath it and through
// the gap beside it otherwise. Arrows meeting a box side are spread along
// it and runs through a gap each get their own track. The canvas grows to
// hold the lanes and their labels.
func routeArrows(g *Graph, opts LayoutOptions) {
	const (
		laneStep      = 36.0 // room for a two-line label between lanes
		laneClearance = 24.0 // room above the first lane for its description
	)
	if len(g.Nodes) == 0 {
		return
	}
	centreY := func(n *Node) float64 { return n.Y + n.H/2 }
	centreX := func(n *Node) float64 { return n.X + n.W/2 }

	var paths []*path
	var ends []*end
	var tracks []*track
	newEnd := func(n *Node, s side, key float64) *end {
		e := &end{node: n, side: s, key: key}
		ends = append(ends, e)
		return e
	}
	// A track serves the end it leaves or enters and shares its key, so a
	// lower end runs nearer the box and no stub crosses another run.
	leftOf := func(e *end) *track {
		t := &track{edge: e.node.X, dir: -1, key: e.key}
		tracks = append(tracks, t)
		return t
	}
	rightOf := func(e *end) *track {
		t := &track{edge: e.node.X + e.node.W, dir: 1, key: e.key}
		tracks = append(tracks, t)
		return t
	}

	lane := 0
	for i := range g.Edges {
		e := &g.Edges[i]
		from, to := e.From, e.To
		p := &path{edge: e}
		paths = append(paths, p)
		switch {
		case from.X == to.X:
			p.from = newEnd(from, leftSide, centreY(to))
			p.to = newEnd(to, leftSide, centreY(from))
			p.bracket = leftOf(p.from)
			continue
		case to.X > from.X && !blocked(g, from, to):
			p.from = newEnd(from, rightSide, centreY(to))
			p.to = newEnd(to, leftSide, centreY(from))
			continue
		}

		forward := to.X > from.X
		left, right := min(from.X, to.X), max(from.X+from.W, to.X+to.W)
		lowest := 0.0
		for _, n := range g.Nodes {
			if n.X >= left && n.X+n.W <= right && n != from && n != to {
				lowest = max(lowest, n.Y+n.H)
			}
		}
		fromBelow, intoBelow := lowestInColumn(g, from), lowestInColumn(g, to)
		if fromBelow {
			lowest = max(lowest, from.Y+from.H)
		}
		if intoBelow {
			lowest = max(lowest, to.Y+to.H)
		}
		p.lane = true
		p.laneY = lowest + laneClearance + float64(lane)*laneStep
		lane++
		g.Height = max(g.Height, p.laneY+laneStep/2+opts.MarginY/2)

		// A side end sorts below the side's straight hops, deeper lanes lower.
		switch {
		case fromBelow:
			p.from = newEnd(from, bottomSide, centreX(to))
		case forward:
			p.from = newEnd(from, rightSide, p.laneY)
			p.fromRun = rightOf(p.from)
		default:
			p.from = newEnd(from, leftSide, p.laneY)
			p.fromRun = leftOf(p.from)
		}
		switch {
		case intoBelow:
			p.to = newEnd(to, bottomSide, centreX(from))
		case forward:
			p.to = newEnd(to, leftSide, p.laneY)
			p.toRun = leftOf(p.to)
		default:
			p.to = newEnd(to, rightSide, p.laneY)
			p.toRun = rightOf(p.to)
		}
	}

	placeEnds(ends)
	placeTracks(tracks, opts)

	for _, p := range paths {
		f, t := p.from.at, p.to.at
		switch {
		case p.bracket != nil:
			x := p.bracket.x
			p.edge.Points = [][2]float64{f, {x, f[1]}, {x, t[1]}, t}
		case !p.lane:
			p.edge.Points = [][2]float64{f, t}
		default:
			pts := [][2]float64{f}
			if p.fromRun != nil {
				pts = append(pts, [2]float64{p.fromRun.x, f[1]}, [2]float64{p.fromRun.x, p.laneY})
			} else {
				pts = append(pts, [2]float64{f[0], p.laneY})
			}
			if p.toRun != nil {
				pts = append(pts, [2]float64{p.toRun.x, p.laneY}, [2]float64{p.toRun.x, t[1]})
			} else {
				pts = append(pts, [2]float64{t[0], p.laneY})
			}
			p.edge.Points = append(pts, t)
		}
	}
}

// placeEnds spreads the arrows meeting each box side evenly along it, in
// order of their keys.
func placeEnds(ends []*end) {
	type key struct {
		n *Node
		s side
	}
	groups := map[key][]*end{}
	var order []key
	for _, e := range ends {
		k := key{e.node, e.side}
		if groups[k] == nil {
			order = append(order, k)
		}
		groups[k] = append(groups[k], e)
	}
	for _, k := range order {
		es := groups[k]
		slices.SortStableFunc(es, func(a, b *end) int { return cmp.Compare(a.key, b.key) })
		n := k.n
		for i, e := range es {
			f := float64(i+1) / float64(len(es)+1)
			switch k.s {
			case rightSide:
				e.at = [2]float64{n.X + n.W, n.Y + f*n.H}
			case leftSide:
				e.at = [2]float64{n.X, n.Y + f*n.H}
			case bottomSide:
				e.at = [2]float64{n.X + f*n.W, n.Y + n.H}
			}
		}
	}
}

// placeTracks sets the runs beside each column side out from it, the
// highest key nearest, spaced to fit the margin when the side is the
// diagram's outermost.
func placeTracks(tracks []*track, opts LayoutOptions) {
	type side struct{ edge, dir float64 }
	groups := map[side][]*track{}
	var order []side
	for _, t := range tracks {
		k := side{t.edge, t.dir}
		if groups[k] == nil {
			order = append(order, k)
		}
		groups[k] = append(groups[k], t)
	}
	for _, k := range order {
		ts := groups[k]
		slices.SortStableFunc(ts, func(a, b *track) int { return cmp.Compare(b.key, a.key) })
		step := min(8, opts.MarginX/float64(len(ts)+1))
		for i, t := range ts {
			t.x = k.edge + k.dir*float64(i+1)*step
		}
	}
}

// blocked reports whether a hop from from's right side to to's left side
// would pass through a box in a column between them.
func blocked(g *Graph, from, to *Node) bool {
	sx, ex := from.X+from.W, to.X
	sy, ey := from.Y+from.H/2, to.Y+to.H/2
	top, bottom := min(sy, ey), max(sy, ey)
	for _, n := range g.Nodes {
		if n.X >= sx && n.X+n.W <= ex && n.Y < bottom && n.Y+n.H > top {
			return true
		}
	}
	return false
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
