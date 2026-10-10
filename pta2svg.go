// Package pta2svg draws plain text accounting as an SVG flow diagram: the
// accounts as boxes carrying their balances, the movements as arrows.
//
// Text in a supported format is parsed, laid out and rendered in one call:
//
//	err := pta2svg.Render(w, src, pta2svg.Options{Format: pta2svg.Goluca, Title: "January"})
//
// Callers that hold the ledger as data rather than text build a Diagram
// and hand it to RenderDiagram.
package pta2svg

import (
	"fmt"
	"io"
	"strings"

	"git.bytestone.uk/hum3/pta2svg/internal/layout"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
	"git.bytestone.uk/hum3/pta2svg/internal/parser"
	"git.bytestone.uk/hum3/pta2svg/internal/render"
)

// Format identifies a plain text accounting format.
type Format int

const (
	PTA       Format = iota // directional movements with free-text descriptions (.pta)
	Goluca                  // go-luca's format: quoted descriptions, directives (.goluca)
	Beancount               // beancount postings with signed amounts (.beancount)
)

func (f Format) String() string {
	switch f {
	case PTA:
		return "pta"
	case Goluca:
		return "goluca"
	case Beancount:
		return "beancount"
	}
	return fmt.Sprintf("Format(%d)", int(f))
}

// DetectFormat picks the format from a filename's extension. Unknown
// extensions are PTA.
func DetectFormat(filename string) Format {
	switch {
	case strings.HasSuffix(filename, ".goluca"):
		return Goluca
	case strings.HasSuffix(filename, ".beancount"):
		return Beancount
	default:
		return PTA
	}
}

// ParseFormat reads a format name as a user would type it: pta, goluca or
// beancount, in any case.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "pta":
		return PTA, nil
	case "goluca":
		return Goluca, nil
	case "beancount":
		return Beancount, nil
	}
	return 0, fmt.Errorf("unknown format %q (use pta, goluca or beancount)", s)
}

// Layout names a layout engine.
type Layout string

const (
	LR       Layout = "lr"       // left-to-right columns by account type (default)
	Sugiyama Layout = "sugiyama" // layered graph layout
)

// Options controls parsing, layout and rendering. The zero value renders PTA
// text with the left-to-right layout, no title and no animation.
type Options struct {
	Format  Format // text format; ignored by RenderDiagram
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

// Parse reads src in the given format.
func Parse(src []byte, format Format) (*Diagram, error) {
	f, err := internalFormat(format)
	if err != nil {
		return nil, err
	}
	return parser.Parse("", src, f)
}

// Render parses src and writes the diagram as SVG to w.
func Render(w io.Writer, src []byte, opts Options) error {
	d, err := Parse(src, opts.Format)
	if err != nil {
		return err
	}
	return RenderDiagram(w, d, opts)
}

// RenderDiagram lays out d and writes it as SVG to w. The diagram's
// Balances are drawn as given, so call ComputeBalances first if it was
// built by hand.
func RenderDiagram(w io.Writer, d *Diagram, opts Options) error {
	l, err := layouter(opts.Layout)
	if err != nil {
		return err
	}
	g := layout.GraphFromDiagram(d, l, layout.DefaultOptions())
	return render.Render(w, g, render.Options{Title: opts.Title, Animate: opts.Animate})
}

func internalFormat(f Format) (parser.Format, error) {
	switch f {
	case PTA:
		return parser.FormatPTA, nil
	case Goluca:
		return parser.FormatGoluca, nil
	case Beancount:
		return parser.FormatBeancount, nil
	}
	return 0, fmt.Errorf("unknown format %v", f)
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
