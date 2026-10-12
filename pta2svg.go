// Package pta2svg draws a ledger as an SVG flow diagram: the accounts as
// boxes carrying their balances, the movements as arrows.
//
// Callers that hold the ledger as data build a Diagram and hand it to
// RenderDiagram. Plain text accounting files are parsed by the parse
// subpackage, which is kept separate because its tree-sitter grammars
// dominate the size of a browser build:
//
//	err := parse.Render(w, src, parse.Goluca, pta2svg.Options{Title: "January"})
package pta2svg

import (
	"fmt"
	"io"

	"git.bytestone.uk/hum3/pta2svg/internal/layout"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
	"git.bytestone.uk/hum3/pta2svg/internal/render"
)

// Layout names a layout engine.
type Layout string

const (
	LR       Layout = "lr"       // left-to-right columns by account type (default)
	Sugiyama Layout = "sugiyama" // layered graph layout
)

// Options controls layout and rendering. The zero value renders with the
// left-to-right layout, no title and no animation.
type Options struct {
	Title   string // SVG <title>
	Animate bool   // CSS flow animation along the arrows
	Layout  Layout // layout engine; empty means LR

	// Commodities is how each commodity's amounts are written, by code,
	// over the defaults: GBP, USD, EUR and JPY with their symbols and
	// thousands separated ("-£1,234.56", "¥123,456"), any other commodity
	// separated with its code after ("1,234.50 XYZ"). {"GBP": {Suffix:
	// " GBP", Decimals: 2}} writes "1234.56 GBP".
	Commodities map[string]Commodity
	// Palette is the fill of each account type's boxes by type name
	// ("Asset", "Liability", "Equity", "Income", "Expense", "Hub",
	// "External"); a type not named keeps pta2svg's own colour.
	Palette map[string]string

	// Shape is the outline of every box: Rounded (the zero value) or
	// Stadium, a pill.
	Shape Shape
	// Strokes is the stroke width of each account type's boxes by type
	// name, as Palette; a type not named keeps the default 1.5.
	Strokes map[string]float64
	// BalanceBelowName writes each balance in bold beneath the account
	// name rather than small in the box's corner.
	BalanceBelowName bool
	// Style is a preset of the three above. "afp", the default, is the
	// look of bytestone.uk/afp: stadium boxes with the P&L types (Income
	// and Expense) stroked at 3 and the balance beneath the name. A Shape
	// or Strokes set alongside it applies on top. "plain" is no preset:
	// the three fields alone.
	Style string
}

// Shape is the outline of an account's box.
type Shape = render.Shape

const (
	Rounded = render.Rounded // rectangle with slightly rounded corners (default)
	Stadium = render.Stadium // pill: the ends fully rounded
)

// withStyle is o with its Style's preset filled in under the fields the
// caller set.
func (o Options) withStyle() (Options, error) {
	switch o.Style {
	case "plain":
		return o, nil
	case "", "afp":
		if o.Shape == Rounded {
			o.Shape = Stadium
		}
		strokes := map[string]float64{"Income": 3, "Expense": 3}
		for t, w := range o.Strokes {
			strokes[t] = w
		}
		o.Strokes = strokes
		o.BalanceBelowName = true
		return o, nil
	}
	return o, fmt.Errorf("unknown style %q (use afp or plain)", o.Style)
}

// Commodity is how a commodity's amounts are written.
type Commodity = render.Commodity

// The ledger model, for callers that build a Diagram without text.
type (
	Diagram     = model.Diagram
	Account     = model.Account
	AccountType = model.AccountType
	Movement    = model.Movement
	Transaction = model.Transaction
)

// Account types, inferred from the first segment of an account's path.
const (
	Asset     = model.Asset
	Liability = model.Liability
	Equity    = model.Equity
	Income    = model.Income
	Expense   = model.Expense
	External  = model.External
)

// NewDiagram returns an empty Diagram. Add accounts with GetOrCreateAccount,
// append Transactions, seed Opening balances, then call ComputeBalances.
func NewDiagram() *Diagram { return model.NewDiagram() }

// RenderDiagram lays out d and writes it as SVG to w. The diagram's
// Balances are drawn as given, so call ComputeBalances first if it was
// built by hand.
func RenderDiagram(w io.Writer, d *Diagram, opts Options) error {
	l, err := layouter(opts.Layout)
	if err != nil {
		return err
	}
	if opts, err = opts.withStyle(); err != nil {
		return err
	}
	// The gap between columns is at least the room the widest arrow
	// label needs, so the text sits between the boxes, not over them.
	ro := render.Options{
		Title: opts.Title, Animate: opts.Animate, Commodities: opts.Commodities, Palette: opts.Palette,
		Shape: opts.Shape, Strokes: opts.Strokes, BalanceBelowName: opts.BalanceBelowName,
	}
	lo := layout.DefaultOptions()
	lo.PadX = max(lo.PadX, render.LabelWidth(d, ro))
	g := layout.GraphFromDiagram(d, l, lo)
	return render.Render(w, g, ro)
}

func layouter(l Layout) (layout.Layouter, error) {
	switch l {
	case "", LR:
		return layout.LR{}, nil
	case Sugiyama:
		return layout.Sugiyama{}, nil
	}
	return nil, fmt.Errorf("unknown layout %q (use lr or sugiyama)", string(l))
}
