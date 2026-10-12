package render

import (
	"bytes"
	"strings"
	"testing"

	"git.bytestone.uk/hum3/pta2svg/internal/layout"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

func TestRenderBasic(t *testing.T) {
	d := model.NewDiagram()
	income := d.GetOrCreateAccount("Income:Salary")
	bank := d.GetOrCreateAccount("Asset:Bank")
	d.Transactions = []model.Transaction{
		{
			Date: "2024-01-01", Flag: "*",
			Movements: []model.Movement{
				{From: income, To: bank, Amount: 5000, Commodity: "GBP", Description: "salary"},
			},
		},
	}

	g := layout.LR{}.Layout(d, layout.DefaultOptions())

	var buf bytes.Buffer
	err := Render(&buf, g, Options{Title: "Test"})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain <svg")
	}
	if !strings.Contains(svg, "<title>Test</title>") {
		t.Error("output should contain title")
	}
	if !strings.Contains(svg, "Salary") {
		t.Error("output should contain account name 'Salary'")
	}
	if !strings.Contains(svg, "5000.00 GBP") {
		t.Error("output should contain amount")
	}
	if !strings.Contains(svg, "salary") {
		t.Error("output should contain description")
	}
	if !strings.Contains(svg, "legend") {
		t.Error("output should contain legend")
	}
}

func TestRenderAnimate(t *testing.T) {
	d := model.NewDiagram()
	a := d.GetOrCreateAccount("Asset:Cash")
	b := d.GetOrCreateAccount("Expense:Food")
	d.Transactions = []model.Transaction{
		{
			Date: "2024-01-01", Flag: "*",
			Movements: []model.Movement{
				{From: a, To: b, Amount: 25},
			},
		},
	}

	g := layout.LR{}.Layout(d, layout.DefaultOptions())

	var buf bytes.Buffer
	err := Render(&buf, g, Options{Animate: true})
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "@keyframes flow") {
		t.Error("animated output should contain @keyframes")
	}
	if !strings.Contains(svg, "animation-delay") {
		t.Error("animated output should contain animation-delay")
	}
}

// LabelWidth is the room the widest arrow label needs: the longer of a
// movement's description and its formatted amount, estimated from the
// character count at the edge font, with clearance either side.
func TestLabelWidth(t *testing.T) {
	d := model.NewDiagram()
	a, b := d.GetOrCreateAccount("Asset:Cash"), d.GetOrCreateAccount("Expense:Food")
	if got := LabelWidth(d); got != 0 {
		t.Errorf("no movements: %v, want 0", got)
	}
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: a, To: b, Amount: 10, Commodity: "GBP", Description: "lunch"},
		{From: a, To: b, Amount: -1234567.89, Commodity: "GBP"},
	}}}
	// "-1234567.89 GBP" is 15 characters, longer than "lunch".
	if got, want := LabelWidth(d), 15*charWidth+labelClearance; got != want {
		t.Errorf("LabelWidth = %v, want %v for the 15-character amount", got, want)
	}
}
