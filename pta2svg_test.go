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
	for _, want := range []string{"Cash", "£90.00", "lunch", "animation"} {
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

// The afp style is the look of bytestone.uk/afp in one option, and the
// default: stadium boxes, the P&L account types stroked heavier, the
// balance in bold beneath the name. It is the same drawing as asking for
// the three separately, which still apply on top of it. "plain" is the
// drawing without it; an unknown style is an error.
func TestStyleAFP(t *testing.T) {
	render := func(o Options) (string, error) {
		var buf bytes.Buffer
		err := RenderDiagram(&buf, lunch(), o)
		return buf.String(), err
	}
	byDefault, err := render(Options{})
	if err != nil {
		t.Fatal(err)
	}
	afp, _ := render(Options{Style: "afp"})
	if byDefault != afp {
		t.Error("the default is not the afp style")
	}
	separately, _ := render(Options{Style: "plain", Shape: Stadium, Strokes: map[string]float64{"Income": 3, "Expense": 3}, BalanceBelowName: true})
	if afp != separately {
		t.Errorf("afp style differs from its three options:\n%s\nvs\n%s", afp, separately)
	}
	plain, _ := render(Options{Style: "plain"})
	if plain == afp || strings.Contains(plain, "style=") || !strings.Contains(plain, `class="balance"`) {
		t.Errorf("plain style carries formatting:\n%s", plain)
	}
	heavier, _ := render(Options{Strokes: map[string]float64{"Expense": 5}})
	if !strings.Contains(heavier, `stroke-width: 5px`) {
		t.Errorf("caller's stroke does not apply over the style:\n%s", heavier)
	}
	if _, err := render(Options{Style: "mermaid"}); err == nil || !strings.Contains(err.Error(), "mermaid") {
		t.Errorf("unknown style: err = %v, want one naming it", err)
	}
}

// The layered (Sugiyama) layout is the default; LR is asked for by name.
// A salary fanning out to a bank and to tax lays out differently in each.
func TestDefaultLayoutIsSugiyama(t *testing.T) {
	d := NewDiagram()
	salary, bank, tax := d.GetOrCreateAccount("Income:Salary"), d.GetOrCreateAccount("Asset:Bank"), d.GetOrCreateAccount("Expense:Tax")
	d.Transactions = []Transaction{{Movements: []Movement{
		{From: salary, To: bank, Amount: 3200, Commodity: "GBP"},
		{From: salary, To: tax, Amount: 800, Commodity: "GBP"},
	}}}
	d.ComputeBalances()
	render := func(o Options) string {
		var buf bytes.Buffer
		if err := RenderDiagram(&buf, d, o); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	if render(Options{}) != render(Options{Layout: Sugiyama}) {
		t.Error("the default layout is not Sugiyama")
	}
	if render(Options{}) == render(Options{Layout: LR}) {
		t.Error("the default layout draws the same as LR; the test cannot tell them apart")
	}
}
