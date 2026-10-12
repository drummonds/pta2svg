package render

import (
	"fmt"
	"io"
	"math"
	"strings"

	"git.bytestone.uk/hum3/pta2svg/internal/layout"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

// Options controls rendering.
type Options struct {
	Title       string
	Animate     bool
	Commodities map[string]Commodity // how each commodity's amounts are written; others as "1.27 GBP"
	Palette     map[string]string    // box fill by account type name ("Asset"); others keep their own

	Shape            Shape              // box outline; the zero value is a rounded rectangle
	Strokes          map[string]float64 // box stroke width by account type name; others keep the default
	BalanceBelowName bool               // balance in bold beneath the name rather than small in the corner
}

// Shape is the outline of an account's box.
type Shape string

const (
	Rounded Shape = ""        // rectangle with slightly rounded corners
	Stadium Shape = "stadium" // pill: the ends fully rounded
)

// Commodity is how a commodity's amounts are written: the symbol before
// the number and after its sign, the decimals shown, the separator
// between thousands, if any, and the suffix after the number.
type Commodity struct {
	Symbol    string `json:"symbol,omitempty"`
	Decimals  int    `json:"decimals"`
	Thousands string `json:"thousands,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
}

// currencies are written with their symbols unless the caller says
// otherwise.
var currencies = map[string]Commodity{
	"GBP": {Symbol: "£", Decimals: 2, Thousands: ","},
	"USD": {Symbol: "$", Decimals: 2, Thousands: ","},
	"EUR": {Symbol: "€", Decimals: 2, Thousands: ","},
	"JPY": {Symbol: "¥", Decimals: 0, Thousands: ","},
}

// fill is the colour an account type's boxes are drawn in: the palette's
// when it names the type, else the type's own.
func (o Options) fill(t model.AccountType) string {
	if f, ok := o.Palette[t.String()]; ok {
		return f
	}
	return t.Fill()
}

// commodity is how amounts in code are written: as the caller described
// it, else as the currency is, else to two places, thousands separated,
// with the code after.
func (o Options) commodity(code string) Commodity {
	if c, ok := o.Commodities[code]; ok {
		return c
	}
	if c, ok := currencies[code]; ok {
		return c
	}
	c := Commodity{Decimals: 2, Thousands: ","}
	if code != "" {
		c.Suffix = " " + code
	}
	return c
}

// formatAmount writes an amount in its commodity.
func (o Options) formatAmount(amount float64, code string) string {
	c := o.commodity(code)
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	s := fmt.Sprintf("%.*f", c.Decimals, amount)
	whole, frac, _ := strings.Cut(s, ".")
	if c.Thousands != "" {
		var groups []string
		for len(whole) > 3 {
			groups = append([]string{whole[len(whole)-3:]}, groups...)
			whole = whole[:len(whole)-3]
		}
		whole = strings.Join(append([]string{whole}, groups...), c.Thousands)
	}
	if frac != "" {
		whole += "." + frac
	}
	return sign + c.Symbol + whole + c.Suffix
}

const defs = `
<defs>
  <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
    <polygon points="0 0, 10 3.5, 0 7" fill="#666"/>
  </marker>
</defs>
<style>
  .node rect { stroke: #999; stroke-width: 1.5; rx: 6; ry: 6; }
  .node text { font-family: sans-serif; font-size: 13px; fill: #333; text-anchor: middle; dominant-baseline: central; }
  .node .balance { font-size: 10px; fill: #666; text-anchor: end; dominant-baseline: auto; }
  .edge path { fill: none; stroke: #666; stroke-width: 1.5; marker-end: url(#arrowhead); }
  .edge text { font-family: sans-serif; font-size: 11px; fill: #555; text-anchor: middle; paint-order: stroke; stroke: #fff; stroke-width: 4px; stroke-linejoin: round; }
  .edge text.on-path { dominant-baseline: central; }
  .legend text { font-family: sans-serif; font-size: 11px; fill: #333; dominant-baseline: central; }
  .legend rect { stroke: #999; stroke-width: 1; rx: 3; ry: 3; }`

const balanceBelow = `
  .node text.name-above { dominant-baseline: auto; }
  .node .balance-below { font-size: 12px; font-weight: bold; fill: #333; dominant-baseline: hanging; }`

const animation = `
  @keyframes flow {
    from { stroke-dashoffset: 100; }
    to   { stroke-dashoffset: 0; }
  }
  .edge path { stroke-dasharray: 6 3; animation: flow 1.5s linear infinite; }`

// Render writes the SVG to w. It is plain string building rather than
// text/template, which TinyGo's reflection cannot run. Coordinates print
// with fmt's %v, the shortest exact form.
func Render(w io.Writer, g *layout.Graph, opts Options) error {
	const legendHeight = 40.0
	width, height := g.Width, g.Height+legendHeight

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %v %v" width="%v" height="%v">`, width, height, width, height)
	if opts.Title != "" {
		fmt.Fprintf(&b, "\n<title>%s</title>", opts.Title)
	}
	b.WriteString(defs)
	if opts.BalanceBelowName {
		b.WriteString(balanceBelow)
	}
	if opts.Animate {
		b.WriteString(animation)
	}
	b.WriteString("\n</style>\n")

	// The legend lists each account type once, in order of first appearance.
	type legendEntry struct{ fill, label string }
	var legend []legendEntry
	seenTypes := make(map[model.AccountType]bool)

	for _, n := range g.Nodes {
		fill := opts.fill(n.Account.Type)
		fmt.Fprintf(&b, "\n<g class=\"node\" transform=\"translate(%v,%v)\">", n.X, n.Y)
		// A box's own shape and stroke go in its inline style, which
		// outranks the stylesheet's .node rect.
		var style []string
		if opts.Shape == Stadium {
			style = append(style, fmt.Sprintf("rx: %vpx; ry: %vpx", n.H/2, n.H/2))
		}
		if w, ok := opts.Strokes[n.Account.Type.String()]; ok {
			style = append(style, fmt.Sprintf("stroke-width: %vpx", w))
		}
		fmt.Fprintf(&b, "\n  <rect width=\"%v\" height=\"%v\" fill=\"%s\"", n.W, n.H, fill)
		if len(style) > 0 {
			fmt.Fprintf(&b, " style=\"%s\"", strings.Join(style, "; "))
		}
		b.WriteString("/>")
		hasBalance := n.Balance != 0 || n.Commodity != ""
		switch {
		case opts.BalanceBelowName && hasBalance:
			fmt.Fprintf(&b, "\n  <text class=\"name-above\" x=\"%v\" y=\"%v\">%s</text>", n.W/2, n.H/2-2, n.Account.ShortName)
			fmt.Fprintf(&b, "\n  <text class=\"balance-below\" x=\"%v\" y=\"%v\">%s</text>",
				n.W/2, n.H/2+2, opts.formatAmount(n.Balance, n.Commodity))
		default:
			fmt.Fprintf(&b, "\n  <text x=\"%v\" y=\"%v\">%s</text>", n.W/2, n.H/2, n.Account.ShortName)
			if hasBalance {
				fmt.Fprintf(&b, "\n  <text class=\"balance\" x=\"%.0f\" y=\"%.0f\" dx=\"-4\" dy=\"-4\">%s</text>",
					n.W, n.H, opts.formatAmount(n.Balance, n.Commodity))
			}
		}
		b.WriteString("\n</g>")

		if !seenTypes[n.Account.Type] {
			seenTypes[n.Account.Type] = true
			legend = append(legend, legendEntry{fill, n.Account.Type.String()})
		}
	}
	b.WriteString("\n")

	for _, e := range g.Edges {
		if len(e.Points) >= 2 {
			// A routed edge's label is centred on the point halfway along
			// it: either side of the line where that is a horizontal
			// stretch, else on the line, haloed so the arrow passes behind.
			midX, midY, flat := halfway(e.Points)
			writeEdgePath(&b, e, polylinePath(e.Points, e.ArrowHeadStart), opts)
			if flat {
				writeLabelsAround(&b, e, midX, midY, opts)
				continue
			}
			amountY := midY
			if d := e.Movement.Description; d != "" {
				fmt.Fprintf(&b, "\n  <text class=\"on-path\" x=\"%v\" y=\"%v\">%s</text>", midX, midY-7, d)
				amountY = midY + 7
			}
			fmt.Fprintf(&b, "\n  <text class=\"on-path\" x=\"%v\" y=\"%v\">%s</text>\n</g>",
				midX, amountY, opts.formatAmount(e.Movement.Amount, e.Movement.Commodity))
			continue
		}

		// Unrouted: a curve between the boxes' facing sides, the label
		// above and below its middle.
		sx := e.From.X + e.From.W // right edge
		sy := e.From.Y + e.From.H/2
		ex := e.To.X // left edge
		ey := e.To.Y + e.To.H/2
		writeEdgePath(&b, e, pathD(sx, sy, ex, ey), opts)
		writeLabelsAround(&b, e, (sx+ex)/2, (sy+ey)/2, opts)
	}

	if len(legend) > 0 {
		fmt.Fprintf(&b, "\n<g class=\"legend\" transform=\"translate(20,%.0f)\" dy=\"-30\">", height)
		for i, l := range legend {
			fmt.Fprintf(&b, "\n  <rect x=\"%d\" y=\"-30\" width=\"14\" height=\"14\" fill=\"%s\"/>", i*100, l.fill)
			fmt.Fprintf(&b, "\n  <text x=\"%d\" y=\"-23\">%s</text>", i*100+18, l.label)
		}
		b.WriteString("\n</g>")
	}
	b.WriteString("\n</svg>\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// writeEdgePath opens an edge's group and draws its arrow along path d.
func writeEdgePath(b *strings.Builder, e layout.Edge, d string, opts Options) {
	fmt.Fprintf(b, "\n<g class=\"edge\">\n  <path d=\"%s\"", d)
	if opts.Animate {
		fmt.Fprintf(b, " style=\"animation-delay: %vs\"", float64(e.Index)*0.3)
	}
	b.WriteString("/>")
}

// writeLabelsAround writes an edge's description above (x,y) and its
// amount below, and closes the edge's group.
func writeLabelsAround(b *strings.Builder, e layout.Edge, x, y float64, opts Options) {
	if d := e.Movement.Description; d != "" {
		fmt.Fprintf(b, "\n  <text x=\"%v\" y=\"%v\">%s</text>", x, y-8, d)
	}
	fmt.Fprintf(b, "\n  <text x=\"%v\" y=\"%v\">%s</text>\n</g>",
		x, y+14, opts.formatAmount(e.Movement.Amount, e.Movement.Commodity))
}

// halfway is the point half the length of the polyline pts along it, and
// whether the stretch it falls on is horizontal.
func halfway(pts [][2]float64) (x, y float64, flat bool) {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += math.Hypot(pts[i][0]-pts[i-1][0], pts[i][1]-pts[i-1][1])
	}
	rest := total / 2
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		seg := math.Hypot(b[0]-a[0], b[1]-a[1])
		if seg > 0 && rest <= seg {
			f := rest / seg
			return a[0] + f*(b[0]-a[0]), a[1] + f*(b[1]-a[1]), a[1] == b[1]
		}
		rest -= seg
	}
	return pts[0][0], pts[0][1], false
}

// pathD generates an SVG path from (sx,sy) to (ex,ey).
// Straight line if flowing left-to-right; cubic Bezier otherwise.
func pathD(sx, sy, ex, ey float64) string {
	if ex > sx+10 {
		// forward flow — gentle curve
		cx := (sx + ex) / 2
		return fmt.Sprintf("M%.1f,%.1f C%.1f,%.1f %.1f,%.1f %.1f,%.1f",
			sx, sy, cx, sy, cx, ey, ex, ey)
	}
	// backward or same-column — arc around
	dx := math.Abs(ex - sx)
	offset := math.Max(60, dx/2+30)
	return fmt.Sprintf("M%.1f,%.1f C%.1f,%.1f %.1f,%.1f %.1f,%.1f",
		sx, sy, sx+offset, sy-offset, ex-offset, ey-offset, ex, ey)
}

// polylinePath builds an SVG path from routed edge points.
// If arrowHeadStart is true, the points are reversed so the arrow marker
// (always at the path end in SVG) appears at the logical start.
func polylinePath(pts [][2]float64, arrowHeadStart bool) string {
	if len(pts) == 0 {
		return ""
	}
	if arrowHeadStart {
		// Reverse so marker-end lands at the logical arrow start
		rev := make([][2]float64, len(pts))
		for i, p := range pts {
			rev[len(pts)-1-i] = p
		}
		pts = rev
	}
	s := fmt.Sprintf("M%.1f,%.1f", pts[0][0], pts[0][1])
	for _, p := range pts[1:] {
		s += fmt.Sprintf(" L%.1f,%.1f", p[0], p[1])
	}
	return s
}

// The edge font is 11px sans-serif; charWidth is the room a character of
// it takes on average, labelClearance the space left between a label and
// the boxes either side of it.
const (
	charWidth      = 6.5
	labelClearance = 20.0
)

// LabelWidth is the room the widest arrow label in d needs between the
// boxes it runs between: the longer of each movement's description and
// its amount as opts writes it, estimated from the character count at the
// edge font, with clearance either side. Zero when there are no movements.
func LabelWidth(d *model.Diagram, opts Options) float64 {
	longest := 0
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			longest = max(longest, len([]rune(m.Description)), len([]rune(opts.formatAmount(m.Amount, m.Commodity))))
		}
	}
	if longest == 0 {
		return 0
	}
	return float64(longest)*charWidth + labelClearance
}
