package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/drummonds/pta2svg/internal/layout"
	"github.com/drummonds/pta2svg/internal/model"
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
