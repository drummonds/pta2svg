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
}

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
	// The gap between columns is at least the room the widest arrow
	// label needs, so the text sits between the boxes, not over them.
	lo := layout.DefaultOptions()
	lo.PadX = max(lo.PadX, render.LabelWidth(d))
	g := layout.GraphFromDiagram(d, l, lo)
	return render.Render(w, g, render.Options{Title: opts.Title, Animate: opts.Animate})
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
