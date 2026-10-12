package layout

import (
	"testing"

	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

func TestLRLayout(t *testing.T) {
	d := model.NewDiagram()
	income := d.GetOrCreateAccount("Income:Salary")
	bank := d.GetOrCreateAccount("Asset:Bank")
	expense := d.GetOrCreateAccount("Expense:Rent")

	d.Transactions = []model.Transaction{
		{
			Date: "2024-01-01", Flag: "*",
			Movements: []model.Movement{
				{From: income, To: bank, Amount: 5000},
				{From: bank, To: expense, Amount: 1200},
			},
		},
	}

	g := LR{}.Layout(d, DefaultOptions())

	if len(g.Nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("got %d edges, want 2", len(g.Edges))
	}

	// Income should be left of Asset, Expense should be right of Asset
	incomeNode := g.NodeMap["Income:Salary"]
	expenseNode := g.NodeMap["Expense:Rent"]
	assetNode := g.NodeMap["Asset:Bank"]

	if incomeNode.X >= assetNode.X {
		t.Errorf("Income column (%f) should be left of Asset column (%f)", incomeNode.X, assetNode.X)
	}
	if expenseNode.X <= assetNode.X {
		t.Errorf("Expense column (%f) should be right of Asset column (%f)", expenseNode.X, assetNode.X)
	}
}

func TestLRCanvasSize(t *testing.T) {
	d := model.NewDiagram()
	d.GetOrCreateAccount("Asset:Cash")

	g := LR{}.Layout(d, DefaultOptions())

	if g.Width <= 0 || g.Height <= 0 {
		t.Errorf("canvas size should be positive: %fx%f", g.Width, g.Height)
	}
}

// An arrow that jumps a column is routed around the boxes in between
// rather than through them, and its route stays on the canvas. Here
// Equity:Input -> Asset:Cash would cross Income:Sales on the same row.
func TestLRRoutesAroundIntermediateBoxes(t *testing.T) {
	assertClearRoutes(t, LR{}.Layout(oranges(), DefaultOptions()))
}

// oranges is the third orange-trading step: an investor's cash, oranges
// bought, oranges sold.
func oranges() *model.Diagram {
	d := model.NewDiagram()
	input, sales := d.GetOrCreateAccount("Equity:Input"), d.GetOrCreateAccount("Income:Sales")
	cash, purchases := d.GetOrCreateAccount("Asset:Cash"), d.GetOrCreateAccount("Expense:Purchases")
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: input, To: cash, Amount: 25},
		{From: cash, To: purchases, Amount: 25},
		{From: sales, To: cash, Amount: 37.5},
	}}}
	return d
}

// An arrow back to an earlier column, or to a box in its own column, is
// routed like a column-jumping one rather than arced off the canvas or
// through its own box. The budget ledger has both: Bank pays the
// Mortgage to its left and the Savings beneath it.
func TestLRRoutesBackwardAndSameColumnArrows(t *testing.T) {
	assertClearRoutes(t, LR{}.Layout(budget(), DefaultOptions()))
}

// assertClearRoutes checks every arrow in g: one that is not a forward
// hop is routed; no route crosses a box other than its own ends; every
// point is on the canvas.
func assertClearRoutes(t *testing.T, g *Graph) {
	t.Helper()
	for _, e := range g.Edges {
		name := e.From.Account.Name + " -> " + e.To.Account.Name
		pts := e.Points
		if len(pts) == 0 {
			if e.To.X <= e.From.X {
				t.Errorf("%s runs backward or within its column but is not routed", name)
				continue
			}
			// Unrouted: a curve within the box between its ends' facing sides.
			pts = [][2]float64{{e.From.X + e.From.W, e.From.Y + e.From.H/2}, {e.To.X, e.To.Y + e.To.H/2}}
		}
		for i := 1; i < len(pts); i++ {
			for _, n := range g.Nodes {
				if n == e.From || n == e.To {
					continue
				}
				if crossesBox(pts[i-1], pts[i], n) {
					t.Errorf("%s crosses %s on segment %v-%v", name, n.Account.Name, pts[i-1], pts[i])
				}
			}
		}
		for _, p := range pts {
			if p[0] < 0 || p[0] > g.Width || p[1] < 0 || p[1] > g.Height {
				t.Errorf("%s point %v off the %vx%v canvas", name, p, g.Width, g.Height)
			}
		}
	}
}

// Arrows meeting a box are spread along its side rather than meeting at
// one point, and no two arrows run along the same stretch of line, so
// each can be followed on its own.
func TestLRSpreadsArrowEnds(t *testing.T) {
	g := LR{}.Layout(budget(), DefaultOptions())
	ends := map[[2]float64]string{}
	for _, e := range g.Edges {
		name := e.From.Account.Name + " -> " + e.To.Account.Name
		if len(e.Points) < 2 {
			t.Fatalf("%s has no route", name)
		}
		for _, p := range [][2]float64{e.Points[0], e.Points[len(e.Points)-1]} {
			if other, ok := ends[p]; ok {
				t.Errorf("%s and %s meet at %v", name, other, p)
			}
			ends[p] = name
		}
	}
	for i, a := range g.Edges {
		for _, b := range g.Edges[i+1:] {
			for j := 1; j < len(a.Points); j++ {
				for k := 1; k < len(b.Points); k++ {
					if overlap(a.Points[j-1], a.Points[j], b.Points[k-1], b.Points[k]) {
						t.Errorf("%s -> %s and %s -> %s share a stretch: %v-%v, %v-%v",
							a.From.Account.Name, a.To.Account.Name, b.From.Account.Name, b.To.Account.Name,
							a.Points[j-1], a.Points[j], b.Points[k-1], b.Points[k])
					}
				}
			}
		}
	}
}

// A run up or down the gap beside a column keeps close to the box it
// serves, clear of the middle of the gap where the hops' labels sit.
func TestLRRunsKeepClearOfGapMiddle(t *testing.T) {
	opts := DefaultOptions()
	g := LR{}.Layout(budget(), opts)
	for _, e := range g.Edges {
		for i := 1; i < len(e.Points); i++ {
			a, b := e.Points[i-1], e.Points[i]
			if a[0] != b[0] {
				continue
			}
			for _, n := range g.Nodes {
				// The middle third of the gap right of n.
				lo, hi := n.X+n.W+opts.PadX/3, n.X+n.W+2*opts.PadX/3
				if a[0] > lo && a[0] < hi && n.X+n.W+opts.PadX <= g.Width-opts.MarginX {
					t.Errorf("%s -> %s runs at x=%v, mid-gap right of %s", e.From.Account.Name, e.To.Account.Name, a[0], n.Account.Name)
				}
			}
		}
	}
}

// A lane runs far enough below the boxes it passes under for its label,
// written above the line, to clear them.
func TestLRLanesLeaveRoomForLabels(t *testing.T) {
	g := LR{}.Layout(oranges(), DefaultOptions())
	for _, e := range g.Edges {
		for i := 1; i < len(e.Points); i++ {
			a, b := e.Points[i-1], e.Points[i]
			if a[1] != b[1] {
				continue
			}
			for _, n := range g.Nodes {
				under := max(a[0], b[0]) > n.X && min(a[0], b[0]) < n.X+n.W && a[1] > n.Y+n.H
				if under && a[1]-(n.Y+n.H) < 20 {
					t.Errorf("%s -> %s lane at y=%v is %v below %s", e.From.Account.Name, e.To.Account.Name, a[1], a[1]-(n.Y+n.H), n.Account.Name)
				}
			}
		}
	}
}

// overlap reports whether two horizontal or vertical segments lie on the
// same line and share more than a point.
func overlap(a1, a2, b1, b2 [2]float64) bool {
	for axis := range 2 {
		other := 1 - axis
		if a1[axis] == a2[axis] && b1[axis] == b2[axis] && a1[axis] == b1[axis] {
			lo := max(min(a1[other], a2[other]), min(b1[other], b2[other]))
			hi := min(max(a1[other], a2[other]), max(b1[other], b2[other]))
			if hi-lo > 1 {
				return true
			}
		}
	}
	return false
}

// budget is the household budget example: a salary paid into a bank,
// which pays a mortgage to its left, savings beneath it and three
// expenses, one of them also paid from the salary.
func budget() *model.Diagram {
	d := model.NewDiagram()
	salary, bank := d.GetOrCreateAccount("Income:Salary"), d.GetOrCreateAccount("Asset:Bank")
	mortgage, savings := d.GetOrCreateAccount("Liability:Mortgage"), d.GetOrCreateAccount("Asset:Savings")
	food, tax, utilities := d.GetOrCreateAccount("Expense:Food"), d.GetOrCreateAccount("Expense:Tax"), d.GetOrCreateAccount("Expense:Utilities")
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: salary, To: bank, Amount: 3200},
		{From: salary, To: tax, Amount: 800},
		{From: bank, To: mortgage, Amount: 1100},
		{From: bank, To: food, Amount: 95},
		{From: bank, To: savings, Amount: 500},
		{From: bank, To: utilities, Amount: 145},
		{From: bank, To: tax, Amount: 180},
	}}}
	return d
}

// crossesBox reports whether the segment a-b passes through n's box,
// tested by sampling along it.
func crossesBox(a, b [2]float64, n *Node) bool {
	for i := 0; i <= 100; i++ {
		f := float64(i) / 100
		x, y := a[0]+f*(b[0]-a[0]), a[1]+f*(b[1]-a[1])
		if x > n.X && x < n.X+n.W && y > n.Y && y < n.Y+n.H {
			return true
		}
	}
	return false
}
