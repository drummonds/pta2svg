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
	Title   string
	Animate bool
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
  .edge text { font-family: sans-serif; font-size: 11px; fill: #555; text-anchor: middle; }
  .legend text { font-family: sans-serif; font-size: 11px; fill: #333; dominant-baseline: central; }
  .legend rect { stroke: #999; stroke-width: 1; rx: 3; ry: 3; }`

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
	if opts.Animate {
		b.WriteString(animation)
	}
	b.WriteString("\n</style>\n")

	// The legend lists each account type once, in order of first appearance.
	type legendEntry struct{ fill, label string }
	var legend []legendEntry
	seenTypes := make(map[model.AccountType]bool)

	for _, n := range g.Nodes {
		fill := n.Account.Type.Fill()
		fmt.Fprintf(&b, "\n<g class=\"node\" transform=\"translate(%v,%v)\">", n.X, n.Y)
		fmt.Fprintf(&b, "\n  <rect width=\"%v\" height=\"%v\" fill=\"%s\"/>", n.W, n.H, fill)
		fmt.Fprintf(&b, "\n  <text x=\"%v\" y=\"%v\">%s</text>", n.W/2, n.H/2, n.Account.ShortName)
		if n.Balance != 0 || n.Commodity != "" {
			fmt.Fprintf(&b, "\n  <text class=\"balance\" x=\"%.0f\" y=\"%.0f\" dx=\"-4\" dy=\"-4\">%s</text>",
				n.W, n.H, formatAmount(n.Balance, n.Commodity))
		}
		b.WriteString("\n</g>")

		if !seenTypes[n.Account.Type] {
			seenTypes[n.Account.Type] = true
			legend = append(legend, legendEntry{fill, n.Account.Type.String()})
		}
	}
	b.WriteString("\n")

	for _, e := range g.Edges {
		var pd string
		var midX, midY float64

		if len(e.Points) >= 2 {
			// Use routed path from layout engine
			pd = polylinePath(e.Points, e.ArrowHeadStart)
			// Label at midpoint of the polyline
			mid := e.Points[len(e.Points)/2]
			midX, midY = mid[0], mid[1]
		} else {
			// Fallback: compute path from node positions
			sx := e.From.X + e.From.W // right edge
			sy := e.From.Y + e.From.H/2
			ex := e.To.X // left edge
			ey := e.To.Y + e.To.H/2
			pd = pathD(sx, sy, ex, ey)
			midX = (sx + ex) / 2
			midY = (sy + ey) / 2
		}

		fmt.Fprintf(&b, "\n<g class=\"edge\">\n  <path d=\"%s\"", pd)
		if opts.Animate {
			fmt.Fprintf(&b, " style=\"animation-delay: %vs\"", float64(e.Index)*0.3)
		}
		b.WriteString("/>")
		if d := e.Movement.Description; d != "" {
			fmt.Fprintf(&b, "\n  <text x=\"%v\" y=\"%v\">%s</text>", midX, midY-8, d)
		}
		fmt.Fprintf(&b, "\n  <text x=\"%v\" y=\"%v\">%s</text>\n</g>",
			midX, midY+14, formatAmount(e.Movement.Amount, e.Movement.Commodity))
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

func formatAmount(amount float64, commodity string) string {
	s := fmt.Sprintf("%.2f", amount)
	if commodity != "" {
		s += " " + commodity
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
// its formatted amount, estimated from the character count at the edge
// font, with clearance either side. Zero when there are no movements.
func LabelWidth(d *model.Diagram) float64 {
	longest := 0
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			longest = max(longest, len([]rune(m.Description)), len(formatAmount(m.Amount, m.Commodity)))
		}
	}
	if longest == 0 {
		return 0
	}
	return float64(longest)*charWidth + labelClearance
}
