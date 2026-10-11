package pta2svg

import (
	"fmt"
	"sort"

	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

// Model is the wire form of a Diagram: the ledger as data, serialised as
// JSON by a producer that already holds it, so no text is parsed. Balances
// are not carried; Diagram computes them.
//
//	{"accounts": [{"name": "Asset:Cash", "short": "Cash", "type": "Asset"}],
//	 "opening": {"Asset:Cash": 100},
//	 "transactions": [{"date": "2026-01-01", "flag": "*",
//	   "movements": [{"from": "Asset:Cash", "to": "Expense:Food",
//	                  "amount": 10, "commodity": "GBP", "description": "lunch"}]}]}
type Model struct {
	Accounts     []ModelAccount     `json:"accounts,omitempty"`
	Opening      map[string]float64 `json:"opening,omitempty"`
	Transactions []ModelTransaction `json:"transactions"`
}

// ModelAccount names an account. Accounts are created from the movements
// that reference them, with short name and type inferred from the path, so
// an entry is only needed to override that inference.
type ModelAccount struct {
	Name  string `json:"name"`
	Short string `json:"short,omitempty"`
	Type  string `json:"type,omitempty"` // Asset, Liability, Equity, Income, Expense, Hub or External
}

type ModelTransaction struct {
	Date      string          `json:"date"`
	Flag      string          `json:"flag,omitempty"`
	Payee     string          `json:"payee,omitempty"`
	Movements []ModelMovement `json:"movements"`
}

type ModelMovement struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	Amount      float64 `json:"amount"`
	Commodity   string  `json:"commodity"`
	Description string  `json:"description,omitempty"`
	Linked      bool    `json:"linked,omitempty"`
}

// ToModel returns d as a Model, accounts sorted by name.
func ToModel(d *Diagram) *Model {
	m := &Model{Transactions: make([]ModelTransaction, 0, len(d.Transactions))}
	for name, a := range d.Accounts {
		m.Accounts = append(m.Accounts, ModelAccount{Name: name, Short: a.ShortName, Type: a.Type.String()})
	}
	sort.Slice(m.Accounts, func(i, j int) bool { return m.Accounts[i].Name < m.Accounts[j].Name })
	if len(d.Opening) > 0 {
		m.Opening = make(map[string]float64, len(d.Opening))
		for name, v := range d.Opening {
			m.Opening[name] = v
		}
	}
	for _, tx := range d.Transactions {
		mt := ModelTransaction{Date: tx.Date, Flag: tx.Flag, Payee: tx.Payee, Movements: make([]ModelMovement, 0, len(tx.Movements))}
		for _, mv := range tx.Movements {
			mt.Movements = append(mt.Movements, ModelMovement{
				From: mv.From.Name, To: mv.To.Name, Amount: mv.Amount, Commodity: mv.Commodity,
				Description: mv.Description, Linked: mv.Linked,
			})
		}
		m.Transactions = append(m.Transactions, mt)
	}
	return m
}

// Diagram builds the Diagram m describes, with balances computed. It fails
// on an account type it does not know or a movement missing from or to.
func (m *Model) Diagram() (*Diagram, error) {
	d := NewDiagram()
	for _, a := range m.Accounts {
		acct := d.GetOrCreateAccount(a.Name)
		if a.Short != "" {
			acct.ShortName = a.Short
		}
		if a.Type != "" {
			t, err := parseAccountType(a.Type)
			if err != nil {
				return nil, fmt.Errorf("account %q: %w", a.Name, err)
			}
			acct.Type = t
		}
	}
	for name, v := range m.Opening {
		d.Opening[name] = v
	}
	for i, mt := range m.Transactions {
		tx := Transaction{Date: mt.Date, Flag: mt.Flag, Payee: mt.Payee, Movements: make([]Movement, 0, len(mt.Movements))}
		for j, mv := range mt.Movements {
			if mv.From == "" {
				return nil, fmt.Errorf("transaction %d movement %d: from is empty", i+1, j+1)
			}
			if mv.To == "" {
				return nil, fmt.Errorf("transaction %d movement %d: to is empty", i+1, j+1)
			}
			tx.Movements = append(tx.Movements, Movement{
				From: d.GetOrCreateAccount(mv.From), To: d.GetOrCreateAccount(mv.To),
				Amount: mv.Amount, Commodity: mv.Commodity, Description: mv.Description, Linked: mv.Linked,
			})
		}
		d.Transactions = append(d.Transactions, tx)
	}
	d.ComputeBalances()
	return d, nil
}

func parseAccountType(s string) (AccountType, error) {
	for _, t := range []AccountType{Asset, Liability, Equity, Income, Expense, model.Hub, External} {
		if t.String() == s {
			return t, nil
		}
	}
	return 0, fmt.Errorf("unknown type %q", s)
}
