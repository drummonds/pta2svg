package layout_test

import (
	"testing"

	"github.com/drummonds/pta2svg/internal/layout"
	"github.com/drummonds/pta2svg/internal/parser"
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
