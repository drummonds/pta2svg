package render

import (
	"embed"
	"fmt"
	"io"
	"math"
	"text/template"

	"github.com/drummonds/pta2svg/internal/layout"
)

//go:embed svg.tmpl
var tmplFS embed.FS

var svgTmpl = template.Must(
	template.New("svg.tmpl").Funcs(template.FuncMap{
		"half": func(v float64) float64 { return v / 2 },
	}).ParseFS(tmplFS, "svg.tmpl"),
)

// Options controls rendering.
type Options struct {
	Title   string
	Animate bool
}

// templateData is passed to the SVG template.
type templateData struct {
	Width   float64
	Height  float64
	Title   string
	Animate bool
	Nodes   []nodeData
	Edges   []edgeData
}

type nodeData struct {
	X, Y, W, H float64
	Fill       string
	Label      string
}

type edgeData struct {
	PathD     string
	Label     string
	LabelX    float64
	LabelY    float64
	AnimDelay float64
}

// Render writes the SVG to w.
func Render(w io.Writer, g *layout.Graph, opts Options) error {
	data := templateData{
		Width:   g.Width,
		Height:  g.Height,
		Title:   opts.Title,
		Animate: opts.Animate,
	}

	for _, n := range g.Nodes {
		data.Nodes = append(data.Nodes, nodeData{
			X:     n.X,
			Y:     n.Y,
			W:     n.W,
			H:     n.H,
			Fill:  n.Account.Type.Fill(),
			Label: n.Account.ShortName,
		})
	}

	for _, e := range g.Edges {
		sx := e.From.X + e.From.W // right edge
		sy := e.From.Y + e.From.H/2
		ex := e.To.X // left edge
		ey := e.To.Y + e.To.H/2

		pd := pathD(sx, sy, ex, ey)

		label := formatAmount(e.Movement.Amount, e.Movement.Commodity)
		if e.Movement.Description != "" {
			label = e.Movement.Description + " " + label
		}

		data.Edges = append(data.Edges, edgeData{
			PathD:     pd,
			Label:     label,
			LabelX:    (sx + ex) / 2,
			LabelY:    (sy+ey)/2 - 8,
			AnimDelay: float64(e.Index) * 0.3,
		})
	}

	return svgTmpl.Execute(w, data)
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

func formatAmount(amount float64, commodity string) string {
	s := fmt.Sprintf("%.2f", amount)
	if commodity != "" {
		s += " " + commodity
	}
	return s
}
