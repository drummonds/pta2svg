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
	d := model.NewDiagram()
	input, sales := d.GetOrCreateAccount("Equity:Input"), d.GetOrCreateAccount("Income:Sales")
	cash, purchases := d.GetOrCreateAccount("Asset:Cash"), d.GetOrCreateAccount("Expense:Purchases")
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: input, To: cash, Amount: 25},
		{From: cash, To: purchases, Amount: 25},
		{From: sales, To: cash, Amount: 37.5},
	}}}
	assertClearRoutes(t, LR{}.Layout(d, DefaultOptions()))
}

// An arrow back to an earlier column, or to a box in its own column, is
// routed like a column-jumping one rather than arced off the canvas or
// through its own box. The budget ledger has both: Bank pays the
// Mortgage to its left and the Savings beneath it.
func TestLRRoutesBackwardAndSameColumnArrows(t *testing.T) {
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
	assertClearRoutes(t, LR{}.Layout(d, DefaultOptions()))
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
