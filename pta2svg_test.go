package pta2svg

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func lunch() *Diagram {
	d := NewDiagram()
	from := d.GetOrCreateAccount("Asset:Cash")
	to := d.GetOrCreateAccount("Expense:Food")
	d.Transactions = []Transaction{{Date: "2026-01-01", Flag: "*", Movements: []Movement{{From: from, To: to, Amount: 10, Commodity: "GBP", Description: "lunch"}}}}
	d.Opening["Asset:Cash"] = 100
	d.ComputeBalances()
	return d
}

func TestRenderDiagramBuiltWithoutText(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderDiagram(&buf, lunch(), Options{Animate: true}); err != nil {
		t.Fatalf("RenderDiagram: %v", err)
	}
	for _, want := range []string{"Cash", "90.00 GBP", "lunch", "animation"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("svg should contain %q", want)
		}
	}
}

func TestRenderDiagramUnknownLayout(t *testing.T) {
	err := RenderDiagram(&bytes.Buffer{}, lunch(), Options{Layout: "circular"})
	if err == nil {
		t.Fatal("unknown layout should be an error")
	}
}

func TestRenderDiagramSugiyama(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderDiagram(&buf, lunch(), Options{Layout: Sugiyama}); err != nil {
		t.Fatalf("RenderDiagram sugiyama: %v", err)
	}
}

// nodeXs is the x of every node box in the SVG, in drawing order.
func nodeXs(svg string) []float64 {
	var xs []float64
	const mark = `class="node" transform="translate(`
	for s := svg; ; {
		i := strings.Index(s, mark)
		if i < 0 {
			return xs
		}
		var x, y float64
		fmt.Sscanf(s[i+len(mark):], "%f,%f)", &x, &y)
		xs = append(xs, x)
		s = s[i+len(mark):]
	}
}

// A long label on an arrow widens the gap between the boxes it runs
// between, so the text does not sit over them; a short one leaves the
// default gap. Both layouts space their columns by the same gap.
func TestLongArrowLabelsWidenTheGap(t *testing.T) {
	for _, l := range []Layout{LR, Sugiyama} {
		gap := func(code string) float64 {
			d := lunch()
			d.Transactions[0].Movements[0].Description = code
			var buf bytes.Buffer
			if err := RenderDiagram(&buf, d, Options{Layout: l}); err != nil {
				t.Fatalf("%s: %v", l, err)
			}
			xs := nodeXs(buf.String())
			if len(xs) != 2 {
				t.Fatalf("%s: %d node boxes, want 2", l, len(xs))
			}
			return max(xs[0], xs[1]) - min(xs[0], xs[1])
		}
		short, long := gap("lunch"), gap("LDAS:FTDP:ACRU interest applied for the month")
		if short != 140+80 {
			t.Errorf("%s: a short label leaves the boxes %v apart, want the box width plus the default gap 220", l, short)
		}
		if long <= short {
			t.Errorf("%s: a long label leaves the boxes %v apart, want wider than %v", l, long, short)
		}
	}
}
