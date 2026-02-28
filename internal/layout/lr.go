package layout

import (
	"sort"

	"github.com/drummonds/pta2svg/internal/model"
)

// LR lays out accounts in left-to-right columns grouped by type.
type LR struct{}

// columnOrder defines L→R ordering of account types.
var columnOrder = []model.AccountType{
	model.Income,
	model.Expense,
	model.Liability,
	model.Asset,
	model.Equity,
	model.External,
}

func (LR) Layout(d *model.Diagram, opts LayoutOptions) *Graph {
	g := &Graph{
		NodeMap: make(map[string]*Node),
	}

	// Group accounts by type
	groups := make(map[model.AccountType][]*model.Account)
	for _, a := range d.Accounts {
		groups[a.Type] = append(groups[a.Type], a)
	}
	// Sort within each group for deterministic output
	for _, accts := range groups {
		sort.Slice(accts, func(i, j int) bool {
			return accts[i].Name < accts[j].Name
		})
	}

	// Place nodes column by column
	col := 0
	maxY := 0.0
	for _, typ := range columnOrder {
		accts := groups[typ]
		if len(accts) == 0 {
			continue
		}
		for row, a := range accts {
			x := opts.MarginX + float64(col)*(opts.NodeW+opts.PadX)
			y := opts.MarginY + float64(row)*(opts.NodeH+opts.PadY)
			n := &Node{
				Account: a,
				X:       x,
				Y:       y,
				W:       opts.NodeW,
				H:       opts.NodeH,
			}
			g.Nodes = append(g.Nodes, n)
			g.NodeMap[a.Name] = n
			if y+opts.NodeH > maxY {
				maxY = y + opts.NodeH
			}
		}
		col++
	}

	// Compute canvas size
	g.Width = opts.MarginX*2 + float64(col)*(opts.NodeW+opts.PadX) - opts.PadX
	g.Height = maxY + opts.MarginY

	// Build edges
	idx := 0
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			fromNode := g.NodeMap[m.From.Name]
			toNode := g.NodeMap[m.To.Name]
			if fromNode == nil || toNode == nil {
				continue
			}
			g.Edges = append(g.Edges, Edge{
				From:     fromNode,
				To:       toNode,
				Movement: m,
				Index:    idx,
			})
			idx++
		}
	}

	return g
}
