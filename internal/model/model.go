package model

import "fmt"

// AccountType classifies accounts for coloring and layout.
type AccountType int

const (
	Asset AccountType = iota
	Liability
	Equity
	Income
	Expense
	Hub      // synthetic hub for multi-posting beancount transactions
	External // catch-all for unknown prefixes
)

func (t AccountType) String() string {
	switch t {
	case Asset:
		return "Asset"
	case Liability:
		return "Liability"
	case Equity:
		return "Equity"
	case Income:
		return "Income"
	case Expense:
		return "Expense"
	case Hub:
		return "Hub"
	default:
		return "External"
	}
}

// Fill returns the SVG fill color for this account type.
func (t AccountType) Fill() string {
	switch t {
	case Asset:
		return "#CDA"
	case Liability:
		return "#DDD"
	case Equity:
		return "#EEB"
	case Income:
		return "#BCE"
	case Expense:
		return "#FCC"
	case Hub:
		return "#FEE"
	default:
		return "#ECECFF"
	}
}

// Account represents a named ledger account.
type Account struct {
	Name      string      // full path e.g. "Asset:Bank"
	ShortName string      // display name e.g. "Bank"
	Type      AccountType // inferred from first path segment
}

// Movement represents a flow of money between two accounts.
type Movement struct {
	From        *Account
	To          *Account
	Amount      float64
	Commodity   string
	Description string
	Linked      bool // prefixed with + for animation grouping
}

// Transaction groups movements under a date.
type Transaction struct {
	Date      string // YYYY-MM-DD
	Flag      string // * or !
	Payee     string
	Movements []Movement
}

// Diagram is the top-level parsed result.
type Diagram struct {
	Accounts     map[string]*Account // keyed by full name
	Transactions []Transaction
	Balances     map[string]float64 // net balance per account
}

// NewDiagram creates an empty diagram.
func NewDiagram() *Diagram {
	return &Diagram{
		Accounts: make(map[string]*Account),
		Balances: make(map[string]float64),
	}
}

// GetOrCreateAccount returns the account for name, creating it if needed.
func (d *Diagram) GetOrCreateAccount(name string) *Account {
	if a, ok := d.Accounts[name]; ok {
		return a
	}
	a := &Account{
		Name:      name,
		ShortName: shortName(name),
		Type:      inferType(name),
	}
	d.Accounts[name] = a
	return a
}

// GetOrCreateHubAccount returns a hub account with a unique name based on
// the transaction date, payee, and sequence number.
func (d *Diagram) GetOrCreateHubAccount(date, payee string, seq int) *Account {
	name := fmt.Sprintf("hub:%s:%s:%d", date, payee, seq)
	if a, ok := d.Accounts[name]; ok {
		return a
	}
	a := &Account{
		Name:      name,
		ShortName: payee,
		Type:      Hub,
	}
	d.Accounts[name] = a
	return a
}

// ComputeBalances calculates net balances from all movements.
func (d *Diagram) ComputeBalances() {
	d.Balances = make(map[string]float64)
	for _, tx := range d.Transactions {
		for _, m := range tx.Movements {
			d.Balances[m.From.Name] -= m.Amount
			d.Balances[m.To.Name] += m.Amount
		}
	}
}

func shortName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == ':' {
			return name[i+1:]
		}
	}
	return name
}

func inferType(name string) AccountType {
	// Extract first segment before ':'
	prefix := name
	for i, c := range name {
		if c == ':' {
			prefix = name[:i]
			break
		}
	}
	switch prefix {
	case "Asset", "Assets":
		return Asset
	case "Liability", "Liabilities":
		return Liability
	case "Equity":
		return Equity
	case "Income", "Revenue":
		return Income
	case "Expense", "Expenses":
		return Expense
	default:
		return External
	}
}
