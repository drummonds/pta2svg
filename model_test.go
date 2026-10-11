package pta2svg

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const lunchJSON = `{"accounts":[{"name":"Asset:Cash","short":"Cash","type":"Asset"},{"name":"Expense:Food","short":"Food","type":"Expense"}],"opening":{"Asset:Cash":100},"transactions":[{"date":"2026-01-01","flag":"*","movements":[{"from":"Asset:Cash","to":"Expense:Food","amount":10,"commodity":"GBP","description":"lunch"}]}]}`

func TestModelJSONShape(t *testing.T) {
	got, err := json.Marshal(ToModel(lunch()))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != lunchJSON {
		t.Errorf("model JSON\n got %s\nwant %s", got, lunchJSON)
	}
}

func TestModelAccountsAreOptional(t *testing.T) {
	var m Model
	src := `{"transactions":[{"date":"2026-01-01","movements":[{"from":"Asset:Cash","to":"Expense:Food","amount":10,"commodity":"GBP"}]}]}`
	if err := json.Unmarshal([]byte(src), &m); err != nil {
		t.Fatal(err)
	}
	d, err := m.Diagram()
	if err != nil {
		t.Fatalf("Diagram: %v", err)
	}
	if d.Accounts["Asset:Cash"].Type != Asset || d.Accounts["Asset:Cash"].ShortName != "Cash" {
		t.Errorf("type and short name should be inferred from the path, got %+v", d.Accounts["Asset:Cash"])
	}
	if d.Balances["Expense:Food"] != 10 {
		t.Errorf("balances should be computed on load, got %v", d.Balances)
	}
}

func TestModelAccountOverridesInference(t *testing.T) {
	m := Model{
		Accounts:     []ModelAccount{{Name: "Pot:Holiday", Short: "Hols", Type: "Asset"}},
		Transactions: []ModelTransaction{{Date: "2026-01-01", Movements: []ModelMovement{{From: "Pot:Holiday", To: "Expense:Travel", Amount: 1, Commodity: "GBP"}}}},
	}
	d, err := m.Diagram()
	if err != nil {
		t.Fatalf("Diagram: %v", err)
	}
	if a := d.Accounts["Pot:Holiday"]; a.Type != Asset || a.ShortName != "Hols" {
		t.Errorf("listed account should override inference, got %+v", a)
	}
}

func TestModelDiagramErrors(t *testing.T) {
	cases := map[string]Model{
		"unknown type": {Accounts: []ModelAccount{{Name: "A:B", Type: "Pot"}}},
		"empty from":   {Transactions: []ModelTransaction{{Date: "2026-01-01", Movements: []ModelMovement{{To: "Expense:Food", Amount: 1}}}}},
		"empty to":     {Transactions: []ModelTransaction{{Date: "2026-01-01", Movements: []ModelMovement{{From: "Asset:Cash", Amount: 1}}}}},
	}
	for name, m := range cases {
		_, err := m.Diagram()
		if err == nil {
			t.Errorf("%s: want an error", name)
			continue
		}
		field := strings.Fields(name)[1]
		if !strings.Contains(err.Error(), field) {
			t.Errorf("%s: error should name the field %q, got %v", name, field, err)
		}
	}
}

// A Diagram with a synthetic hub account (as the beancount parser builds)
// survives the trip through JSON and draws the same SVG.
func TestModelRoundTripKeepsHubAccounts(t *testing.T) {
	d := NewDiagram()
	hub := d.GetOrCreateHubAccount("2026-01-02", "Shop", 0)
	cash := d.GetOrCreateAccount("Asset:Cash")
	food := d.GetOrCreateAccount("Expense:Food")
	d.Transactions = []Transaction{{Date: "2026-01-02", Flag: "*", Payee: "Shop", Movements: []Movement{
		{From: cash, To: hub, Amount: 12, Commodity: "GBP", Linked: true},
		{From: hub, To: food, Amount: 12, Commodity: "GBP", Description: "groceries", Linked: true},
	}}}
	d.ComputeBalances()

	raw, err := json.Marshal(ToModel(d))
	if err != nil {
		t.Fatal(err)
	}
	var m Model
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	back, err := m.Diagram()
	if err != nil {
		t.Fatalf("Diagram: %v", err)
	}
	var want, got bytes.Buffer
	if err := RenderDiagram(&want, d, Options{Animate: true}); err != nil {
		t.Fatal(err)
	}
	if err := RenderDiagram(&got, back, Options{Animate: true}); err != nil {
		t.Fatal(err)
	}
	if want.String() != got.String() {
		t.Error("SVG differs after the JSON round trip")
	}
}
