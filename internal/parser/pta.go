package parser

import (
	"fmt"
	"strings"

	"github.com/drummonds/gotreesitter"
	"github.com/drummonds/pta2svg/internal/model"
)

// walkPTA walks a PTA parse tree and populates the diagram.
func walkPTA(root *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil || child.Type(lang) != "transaction" {
			continue
		}
		tx, err := parsePTATransaction(child, lang, src, d)
		if err != nil {
			return err
		}
		d.Transactions = append(d.Transactions, tx)
	}
	return nil
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
	m.From = d.GetOrCreateAccount(from.Text(src))
	m.To = d.GetOrCreateAccount(to.Text(src))

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
