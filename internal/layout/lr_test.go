package layout

import (
	"testing"

	"github.com/drummonds/pta2svg/internal/model"
)

func TestLRLayout(t *testing.T) {
	d := model.NewDiagram()
	income := d.GetOrCreateAccount("Income:Salary")
	bank := d.GetOrCreateAccount("Asset:Bank")
	expense := d.GetOrCreateAccount("Expense:Rent")

	d.Transactions = []model.Transaction{
		{
			Date: "2024-01-01", Flag: "*",
			Movements: []model.Movement{
				{From: income, To: bank, Amount: 5000},
				{From: bank, To: expense, Amount: 1200},
			},
		},
	}

	g := LR{}.Layout(d, DefaultOptions())

	if len(g.Nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("got %d edges, want 2", len(g.Edges))
	}

	// Income and Expense should be in earlier columns than Asset
	incomeNode := g.NodeMap["Income:Salary"]
	expenseNode := g.NodeMap["Expense:Rent"]
	assetNode := g.NodeMap["Asset:Bank"]

	if incomeNode.X >= assetNode.X {
		t.Errorf("Income column (%f) should be left of Asset column (%f)", incomeNode.X, assetNode.X)
	}
	if expenseNode.X >= assetNode.X {
		t.Errorf("Expense column (%f) should be left of Asset column (%f)", expenseNode.X, assetNode.X)
	}
}

func TestLRCanvasSize(t *testing.T) {
	d := model.NewDiagram()
	d.GetOrCreateAccount("Asset:Cash")

	g := LR{}.Layout(d, DefaultOptions())

	if g.Width <= 0 || g.Height <= 0 {
		t.Errorf("canvas size should be positive: %fx%f", g.Width, g.Height)
	}
}
