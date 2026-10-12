package render

import (
	"bytes"
	"fmt"
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
	if !strings.Contains(svg, "£5,000.00") {
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
	// Written as pounds by default, "-£1,234,567.89" is 14 characters,
	// longer than "lunch".
	if got, want := LabelWidth(d, Options{}), 14*charWidth+labelClearance; got != want {
		t.Errorf("LabelWidth = %v, want %v for the 14-character amount", got, want)
	}
	// Written as the caller describes, "-1234567.89 quid" is 16.
	quid := Options{Commodities: map[string]Commodity{"GBP": {Suffix: " quid", Decimals: 2}}}
	if got, want := LabelWidth(d, quid), 16*charWidth+labelClearance; got != want {
		t.Errorf("LabelWidth = %v, want %v for the 16-character amount", got, want)
	}
}

// Money is written for people: the common currencies with their symbol
// before the number and after its sign, thousands separated, yen without
// decimals; any other commodity grouped the same way with its code
// after. A commodity the caller describes is written their way instead,
// symbol, suffix, decimals and separator as given. A palette
// fills an account type's boxes, legend swatch included, in the caller's
// colour; a type the palette leaves out keeps its own.
func TestCommodityFormatAndPalette(t *testing.T) {
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
		{-2500, "USD", "-$2,500.00"},
		{5, "EUR", "€5.00"},
		{123456, "JPY", "¥123,456"},
		{-1234.5, "XYZ", "-1,234.50 XYZ"},
		{5, "", "5.00"},
	} {
		if got := (Options{}).formatAmount(c.amount, c.commodity); got != c.want {
			t.Errorf("formatAmount(%v, %q) = %q, want %q", c.amount, c.commodity, got, c.want)
		}
	}
	opts := Options{
		Commodities: map[string]Commodity{"GBP": {Symbol: "£", Decimals: 2, Thousands: ","}, "JPY": {Symbol: "¥", Decimals: 0}, "EUR": {Suffix: " EUR", Decimals: 2}},
		Palette:     map[string]string{"Asset": "#d1f0e2", "Expense": "#ccfaf3"},
	}
	for _, c := range []struct {
		amount    float64
		commodity string
		want      string
	}{
		{123456, "JPY", "¥123456"},
		{-5, "EUR", "-5.00 EUR"},
	} {
		if got := opts.formatAmount(c.amount, c.commodity); got != c.want {
			t.Errorf("described formatAmount(%v, %q) = %q, want %q", c.amount, c.commodity, got, c.want)
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
	if want := ` x="` + strconv.FormatFloat(wantX, 'f', -1, 64) + `"`; !strings.Contains(buf.String(), want) {
		t.Errorf("label not at the path's middle %v:\n%s", wantX, buf.String())
	}
}

// A routed edge's label is centred on the point halfway along its path by
// length, not on a bend, and is drawn with a halo so the arrow passes
// behind the text rather than through it.
func TestRoutedEdgeLabelCentredHalfwayAlongPath(t *testing.T) {
	d := model.NewDiagram()
	a, b := d.GetOrCreateAccount("Income:Salary"), d.GetOrCreateAccount("Expense:Tax")
	m := model.Movement{From: a, To: b, Amount: 800, Commodity: "GBP", Description: "PAYE"}
	from, to := &layout.Node{Account: a, W: 140, H: 50}, &layout.Node{Account: b, X: 60, Y: 300, W: 140, H: 50}
	// 100 across then 300 down: halfway along the 400 is (100,100), not the bend at (100,0).
	g := &layout.Graph{Nodes: []*layout.Node{from, to}, Width: 400, Height: 400, Edges: []layout.Edge{
		{From: from, To: to, Movement: m, Points: [][2]float64{{0, 0}, {100, 0}, {100, 300}}},
	}}
	var buf bytes.Buffer
	if err := Render(&buf, g, Options{}); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	var ys []float64
	for _, line := range strings.Split(svg, "\n") {
		if !strings.Contains(line, ">PAYE<") && !strings.Contains(line, ">£800.00<") {
			continue
		}
		var x, y float64
		if _, err := fmt.Sscanf(line[strings.Index(line, "x="):], `x="%g" y="%g"`, &x, &y); err != nil {
			t.Fatalf("label %q: %v", line, err)
		}
		if x != 100 {
			t.Errorf("label %q at x=%v, want 100", line, x)
		}
		ys = append(ys, y)
	}
	if len(ys) != 2 || (ys[0]+ys[1])/2 != 100 || ys[0] >= ys[1] {
		t.Errorf("label lines at y=%v, want description above amount, centred on 100", ys)
	}
	if !strings.Contains(svg, "paint-order: stroke") {
		t.Errorf("routed edge labels have no halo:\n%s", svg)
	}
}

// When the point halfway along a routed edge falls on a horizontal
// stretch, the label sits either side of the line, the description above
// and the amount below, each clear of the line by the text's height, so
// neither hides the arrow.
func TestRoutedEdgeLabelStraddlesHorizontalStretch(t *testing.T) {
	d := model.NewDiagram()
	a, b := d.GetOrCreateAccount("Income:Salary"), d.GetOrCreateAccount("Asset:Bank")
	m := model.Movement{From: a, To: b, Amount: 3200, Commodity: "GBP", Description: "net pay"}
	from, to := &layout.Node{Account: a, W: 140, H: 50}, &layout.Node{Account: b, X: 300, W: 140, H: 50}
	// 20 down, 260 across, 20 up: halfway is (140,120), on the lane.
	g := &layout.Graph{Nodes: []*layout.Node{from, to}, Width: 500, Height: 200, Edges: []layout.Edge{
		{From: from, To: to, Movement: m, Points: [][2]float64{{10, 100}, {10, 120}, {270, 120}, {270, 100}}},
	}}
	var buf bytes.Buffer
	if err := Render(&buf, g, Options{}); err != nil {
		t.Fatal(err)
	}
	pos := map[string][2]float64{}
	for _, line := range strings.Split(buf.String(), "\n") {
		for _, label := range []string{">net pay<", ">£3,200.00<"} {
			if strings.Contains(line, label) {
				var x, y float64
				if _, err := fmt.Sscanf(line[strings.Index(line, "x="):], `x="%g" y="%g"`, &x, &y); err != nil {
					t.Fatalf("label %q: %v", line, err)
				}
				pos[label] = [2]float64{x, y}
			}
		}
	}
	desc, amount := pos[">net pay<"], pos[">£3,200.00<"]
	if desc[0] != 140 || amount[0] != 140 {
		t.Errorf("labels at x=%v,%v, want 140", desc[0], amount[0])
	}
	// The description's baseline is above the line; the amount's top,
	// 11px font above its baseline, is below it.
	if !(desc[1] <= 116 && amount[1]-11 >= 122) {
		t.Errorf("description at y=%v and amount at y=%v are not clear of the line at 120", desc[1], amount[1])
	}
}

// Formatting options, all off by default: a stadium shape rounds each box
// to a pill, a stroke width per account type draws that type's boxes
// heavier, and the balance can sit in bold, centred, beneath the name.
func TestBoxFormatting(t *testing.T) {
	d := model.NewDiagram()
	a, b := d.GetOrCreateAccount("Income:Salary"), d.GetOrCreateAccount("Asset:Bank")
	d.Transactions = []model.Transaction{{Movements: []model.Movement{{From: a, To: b, Amount: 10, Commodity: "GBP"}}}}
	d.ComputeBalances()
	g := layout.LR{}.Layout(d, layout.DefaultOptions())
	render := func(o Options) string {
		var buf bytes.Buffer
		if err := Render(&buf, g, o); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	rect := func(svg, fill string) string {
		for _, line := range strings.Split(svg, "\n") {
			if strings.Contains(line, "<rect") && strings.Contains(line, `fill="`+fill+`"`) && strings.Contains(line, `height="50"`) {
				return line
			}
		}
		t.Fatalf("no box filled %s:\n%s", fill, svg)
		return ""
	}

	plain := render(Options{})
	// The stylesheet's rx and stroke-width outrank attributes, so a box's
	// own shape and stroke must be in its inline style to show.
	if r := rect(plain, model.Income.Fill()); strings.Contains(r, "style=") {
		t.Errorf("default box carries formatting: %s", r)
	}
	if !strings.Contains(plain, `class="balance"`) {
		t.Error("default balance is not in the corner")
	}

	styled := render(Options{Shape: Stadium, Strokes: map[string]float64{"Income": 3}, BalanceBelowName: true})
	if r := rect(styled, model.Income.Fill()); !strings.Contains(r, `style="rx: 25px; ry: 25px; stroke-width: 3px"`) {
		t.Errorf("income box not a heavy stadium: %s", r)
	}
	if r := rect(styled, model.Asset.Fill()); !strings.Contains(r, `style="rx: 25px; ry: 25px"`) {
		t.Errorf("asset box not a plain-stroked stadium: %s", r)
	}
	if strings.Contains(styled, `class="balance"`) || !strings.Contains(styled, `class="balance-below"`) {
		t.Errorf("balance not beneath the name:\n%s", styled)
	}
	if !strings.Contains(styled, ".balance-below { ") || !strings.Contains(styled, "font-weight: bold") {
		t.Error("balance beneath the name is not bold")
	}
}
