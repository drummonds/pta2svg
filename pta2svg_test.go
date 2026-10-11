package pta2svg

import (
	"bytes"
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
