package parser

import (
	"strings"
	"testing"

	"github.com/drummonds/pta2svg/internal/model"
)

func TestParseSimple(t *testing.T) {
	input := `2024-01-15 * Salary deposit
  Income:Salary -> Asset:Bank salary payment 5000.00 GBP

2024-01-20 * Pay rent
  Asset:Bank -> Expense:Rent monthly rent 1200.00 GBP
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(d.Transactions) != 2 {
		t.Fatalf("got %d transactions, want 2", len(d.Transactions))
	}

	tx0 := d.Transactions[0]
	if tx0.Date != "2024-01-15" {
		t.Errorf("tx0.Date = %q, want 2024-01-15", tx0.Date)
	}
	if tx0.Flag != "*" {
		t.Errorf("tx0.Flag = %q, want *", tx0.Flag)
	}
	if tx0.Payee != "Salary deposit" {
		t.Errorf("tx0.Payee = %q, want 'Salary deposit'", tx0.Payee)
	}
	if len(tx0.Movements) != 1 {
		t.Fatalf("tx0 has %d movements, want 1", len(tx0.Movements))
	}

	m := tx0.Movements[0]
	if m.From.Name != "Income:Salary" {
		t.Errorf("from = %q, want Income:Salary", m.From.Name)
	}
	if m.To.Name != "Asset:Bank" {
		t.Errorf("to = %q, want Asset:Bank", m.To.Name)
	}
	if m.Amount != 5000.0 {
		t.Errorf("amount = %f, want 5000.0", m.Amount)
	}
	if m.Commodity != "GBP" {
		t.Errorf("commodity = %q, want GBP", m.Commodity)
	}
	if m.Description != "salary payment" {
		t.Errorf("description = %q, want 'salary payment'", m.Description)
	}
}

func TestParseLinked(t *testing.T) {
	input := `2024-02-01 * Multi-leg
  +Income:Salary -> Asset:Bank salary 3000.00 GBP
  +Asset:Bank -> Expense:Tax tax 600.00 GBP
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(d.Transactions) != 1 {
		t.Fatalf("got %d transactions, want 1", len(d.Transactions))
	}
	for i, m := range d.Transactions[0].Movements {
		if !m.Linked {
			t.Errorf("movement %d not linked", i)
		}
	}
}

func TestParseArrowVariants(t *testing.T) {
	tests := []struct {
		name  string
		arrow string
	}{
		{"dash-arrow", "->"},
		{"double-slash", "//"},
		{"unicode", "→"},
		{"gt", " > "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "2024-01-01 * Test\n  A:X " + tt.arrow + " B:Y 100.00\n"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(d.Transactions[0].Movements) != 1 {
				t.Fatal("expected 1 movement")
			}
			m := d.Transactions[0].Movements[0]
			if m.From.Name != "A:X" || m.To.Name != "B:Y" {
				t.Errorf("got %s -> %s", m.From.Name, m.To.Name)
			}
		})
	}
}

func TestParseNoTrailingBlank(t *testing.T) {
	input := "2024-01-01 * Test\n  Asset:Cash -> Expense:Food lunch 25.50"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(d.Transactions) != 1 {
		t.Fatalf("got %d transactions, want 1", len(d.Transactions))
	}
}

func TestBalances(t *testing.T) {
	input := `2024-01-01 * Test
  Income:Salary -> Asset:Bank 1000.00
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if d.Balances["Income:Salary"] != -1000.0 {
		t.Errorf("Income:Salary balance = %f, want -1000", d.Balances["Income:Salary"])
	}
	if d.Balances["Asset:Bank"] != 1000.0 {
		t.Errorf("Asset:Bank balance = %f, want 1000", d.Balances["Asset:Bank"])
	}
}

func TestAccountTypes(t *testing.T) {
	input := `2024-01-01 * Types
  Income:Salary -> Asset:Bank 100.00
  Asset:Bank -> Expense:Food 50.00
  Asset:Bank -> Liability:Loan 30.00
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	checks := map[string]model.AccountType{
		"Income:Salary":  model.Income,
		"Asset:Bank":     model.Asset,
		"Expense:Food":   model.Expense,
		"Liability:Loan": model.Liability,
	}
	for name, want := range checks {
		a := d.Accounts[name]
		if a == nil {
			t.Errorf("account %q not found", name)
			continue
		}
		if a.Type != want {
			t.Errorf("%s.Type = %v, want %v", name, a.Type, want)
		}
	}
}
