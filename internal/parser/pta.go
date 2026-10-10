package parser

import (
	"fmt"
	"strings"

	"git.bytestone.uk/hum3/gotreesitter"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
)

// walkPTA walks a PTA parse tree and populates the diagram.
func walkPTA(root *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "transaction":
			tx, err := parsePTATransaction(child, lang, src, d)
			if err != nil {
				return err
			}
			d.Transactions = append(d.Transactions, tx)
		case "directive":
			if err := parseDirective(child, lang, src, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseDirective handles the directives shared by pta and goluca. Only the
// balance directive affects the diagram; the rest are skipped.
func parseDirective(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil || child.Type(lang) != "balance_directive" {
			continue
		}
		if err := parseBalanceDirective(child, lang, src, d); err != nil {
			return fmt.Errorf("line %d: %w", child.StartPoint().Row+1, err)
		}
	}
	return nil
}

// parseBalanceDirective records the balance an account holds before the
// movements: DATE balance ACCOUNT AMOUNT [COMMODITY].
func parseBalanceDirective(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	var account *model.Account
	var amount float64
	var seen bool
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "account":
			account = d.GetOrCreateAccount(accountName(child.Text(src)))
		case "amount":
			val, err := parseAmountStr(child.Text(src))
			if err != nil {
				return fmt.Errorf("invalid amount %q: %w", child.Text(src), err)
			}
			amount, seen = val, true
		}
	}
	if account == nil || !seen {
		return fmt.Errorf("balance directive needs an account and an amount")
	}
	d.Opening[account.Name] = amount
	return nil
}

// accountName returns the account name as written, without the quotes that
// a name with spaces is wrapped in.
func accountName(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

func parsePTATransaction(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) (model.Transaction, error) {
	var tx model.Transaction
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "header":
			parsePTAHeader(child, lang, src, &tx)
		case "movement":
			m, err := parseDirectionalMovement(child, lang, src, d, false)
			if err != nil {
				return tx, fmt.Errorf("line %d: %w", child.StartPoint().Row+1, err)
			}
			tx.Movements = append(tx.Movements, m)
		}
	}
	return tx, nil
}

func parsePTAHeader(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, tx *model.Transaction) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "date":
			tx.Date = child.Text(src)
		case "flag":
			tx.Flag = child.Text(src)
		case "payee":
			tx.Payee = child.Text(src)
		}
	}
}

// parseDirectionalMovement parses a PTA/goluca movement node.
// If quotedDesc is true, the description comes from a "description" node (goluca);
// otherwise it's assembled from "word" nodes (PTA).
func parseDirectionalMovement(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram, quotedDesc bool) (model.Movement, error) {
	var m model.Movement
	var words []string

	from := node.ChildByFieldName("from", lang)
	to := node.ChildByFieldName("to", lang)
	if from == nil || to == nil {
		return m, fmt.Errorf("movement missing from/to accounts")
	}
	m.From = d.GetOrCreateAccount(accountName(from.Text(src)))
	m.To = d.GetOrCreateAccount(accountName(to.Text(src)))

	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "linked_prefix":
			m.Linked = true
		case "amount":
			val, err := parseAmountStr(child.Text(src))
			if err != nil {
				return m, fmt.Errorf("invalid amount %q: %w", child.Text(src), err)
			}
			m.Amount = val
		case "commodity":
			m.Commodity = child.Text(src)
		case "word":
			if !quotedDesc {
				words = append(words, child.Text(src))
			}
		case "description":
			if quotedDesc {
				text := child.Text(src)
				m.Description = strings.Trim(text, "\"")
			}
		}
	}

	if !quotedDesc {
		m.Description = strings.Join(words, " ")
	}

	return m, nil
}
