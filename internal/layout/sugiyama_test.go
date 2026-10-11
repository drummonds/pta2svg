package layout_test

import (
	"os"
	"strings"
	"testing"

	"git.bytestone.uk/hum3/pta2svg/internal/layout"
	"git.bytestone.uk/hum3/pta2svg/internal/parser"
)

func TestSugiyamaBasic(t *testing.T) {
	src := []byte(`2024-01-01 * Investment
  Equity:Input -> Asset:Cash investment 25.00 GBP

2024-01-02 * Buy oranges
  Asset:Cash -> Expense:Purchases oranges 20.00 GBP

2024-01-03 * Sell oranges
  Income:Sales -> Asset:Cash market sales 37.50 GBP
`)
	d, err := parser.Parse("test.pta", src, parser.FormatPTA)
	if err != nil {
		t.Fatal("parse:", err)
	}
	if len(d.Accounts) == 0 {
		t.Fatal("expected accounts, got 0")
	}

	g := layout.GraphFromDiagram(d, layout.Sugiyama{}, layout.DefaultOptions())
	if len(g.Nodes) == 0 {
		t.Fatal("expected nodes, got 0")
	}
	if len(g.Edges) == 0 {
		t.Fatal("expected edges, got 0")
	}
	if g.Width <= 0 || g.Height <= 0 {
		t.Fatalf("bad canvas size: %.0fx%.0f", g.Width, g.Height)
	}

	// Every edge should have routed points
	for _, e := range g.Edges {
		if len(e.Points) < 2 {
			t.Errorf("edge %s->%s: expected >=2 points, got %d",
				e.From.Account.Name, e.To.Account.Name, len(e.Points))
		}
	}
}

func TestSugiyamaEmpty(t *testing.T) {
	d, _ := parser.Parse("empty.pta", []byte{}, parser.FormatPTA)
	g := layout.GraphFromDiagram(d, layout.Sugiyama{}, layout.DefaultOptions())
	if len(g.Nodes) != 0 {
		t.Errorf("expected 0 nodes for empty diagram, got %d", len(g.Nodes))
	}
}

// autog does not return its nodes in a stable order; the layout must, or
// the SVG (node and legend order) changes from run to run.
func TestSugiyamaNodeOrderIsStable(t *testing.T) {
	src, err := os.ReadFile("../../testdata/oranges/04_buy_on_credit.goluca")
	if err != nil {
		t.Fatal(err)
	}
	order := func() string {
		d, err := parser.Parse("x.goluca", src, parser.FormatGoluca)
		if err != nil {
			t.Fatal("parse:", err)
		}
		g := layout.GraphFromDiagram(d, layout.Sugiyama{}, layout.DefaultOptions())
		var names []string
		for _, n := range g.Nodes {
			names = append(names, n.Account.Name)
		}
		return strings.Join(names, ",")
	}
	want := order()
	for range 50 {
		if got := order(); got != want {
			t.Fatalf("node order changed:\n%s\n%s", want, got)
		}
	}
}
