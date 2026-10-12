package render

import (
	"bytes"
	"strconv"
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
	if got := LabelWidth(d, Options{}); got != 0 {
		t.Errorf("no movements: %v, want 0", got)
	}
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: a, To: b, Amount: 10, Commodity: "GBP", Description: "lunch"},
		{From: a, To: b, Amount: -1234567.89, Commodity: "GBP"},
	}}}
	// "-1234567.89 GBP" is 15 characters, longer than "lunch".
	if got, want := LabelWidth(d, Options{}), 15*charWidth+labelClearance; got != want {
		t.Errorf("LabelWidth = %v, want %v for the 15-character amount", got, want)
	}
	// Written as pounds, "-£1,234,567.89" is 14 characters.
	pounds := Options{Commodities: map[string]Commodity{"GBP": {Symbol: "£", Decimals: 2, Thousands: ","}}}
	if got, want := LabelWidth(d, pounds), 14*charWidth+labelClearance; got != want {
		t.Errorf("LabelWidth = %v, want %v for the 14-character amount", got, want)
	}
}

// A commodity the caller describes is written its way: the symbol before
// the number and after its sign, the decimals given, the thousands
// separated; one the caller does not is written as before. A palette
// fills an account type's boxes, legend swatch included, in the caller's
// colour; a type the palette leaves out keeps its own.
func TestCommodityFormatAndPalette(t *testing.T) {
	opts := Options{
		Commodities: map[string]Commodity{"GBP": {Symbol: "£", Decimals: 2, Thousands: ","}, "JPY": {Symbol: "¥", Decimals: 0}},
		Palette:     map[string]string{"Asset": "#d1f0e2", "Expense": "#ccfaf3"},
	}
	for _, c := range []struct {
		amount    float64
		commodity string
		want      string
	}{
		{1234567.891, "GBP", "£1,234,567.89"},
		{-1.27, "GBP", "-£1.27"},
		{0, "GBP", "£0.00"},
		{999, "GBP", "£999.00"},
		{1000, "GBP", "£1,000.00"},
		{123456, "JPY", "¥123456"},
		{-5, "EUR", "-5.00 EUR"},
		{5, "", "5.00"},
	} {
		if got := opts.formatAmount(c.amount, c.commodity); got != c.want {
			t.Errorf("formatAmount(%v, %q) = %q, want %q", c.amount, c.commodity, got, c.want)
		}
	}

	d := model.NewDiagram()
	a, b, e := d.GetOrCreateAccount("Asset:Cash"), d.GetOrCreateAccount("Liability:Savings"), d.GetOrCreateAccount("Expense:Interest")
	d.Transactions = []model.Transaction{{Movements: []model.Movement{
		{From: a, To: b, Amount: -1000, Commodity: "GBP", Description: "deposit"},
		{From: e, To: b, Amount: -1.27, Commodity: "GBP", Description: "interest"},
	}}}
	d.ComputeBalances()
	g := layout.LR{}.Layout(d, layout.DefaultOptions())
	var buf bytes.Buffer
	if err := Render(&buf, g, opts); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	for _, want := range []string{"-£1,000.00", "-£1.27", `fill="#d1f0e2"`, `fill="#ccfaf3"`, `fill="` + model.Liability.Fill() + `"`} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg lacks %q:\n%s", want, svg)
		}
	}
	if strings.Contains(svg, "GBP") || strings.Contains(svg, model.Asset.Fill()) {
		t.Errorf("svg still carries the commodity code or the default asset fill:\n%s", svg)
	}
}

// A routed edge's label sits at the middle of its path: for a straight
// two-point edge, halfway along it, not at its end.
func TestRoutedEdgeLabelSitsMidPath(t *testing.T) {
	d := model.NewDiagram()
	a, b := d.GetOrCreateAccount("Expense:Interest"), d.GetOrCreateAccount("Liability:Savings")
	m := model.Movement{From: a, To: b, Amount: -1.27, Commodity: "GBP", Description: "LDAS:FTDP:ACRU"}
	d.Transactions = []model.Transaction{{Movements: []model.Movement{m}}}
	g := layout.Sugiyama{}.Layout(d, layout.DefaultOptions())
	if len(g.Edges) != 1 || len(g.Edges[0].Points) != 2 {
		t.Fatalf("edges %+v; want one straight routed edge", g.Edges)
	}
	p := g.Edges[0].Points
	wantX := (p[0][0] + p[1][0]) / 2
	var buf bytes.Buffer
	if err := Render(&buf, g, Options{}); err != nil {
		t.Fatal(err)
	}
	if want := `<text x="` + strconv.FormatFloat(wantX, 'f', -1, 64) + `"`; !strings.Contains(buf.String(), want) {
		t.Errorf("label not at the path's middle %v:\n%s", wantX, buf.String())
	}
}
