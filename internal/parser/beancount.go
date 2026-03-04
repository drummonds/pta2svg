package parser

import (
	"fmt"
	"math"
	"strings"

	"github.com/drummonds/gotreesitter"
	"github.com/drummonds/pta2svg/internal/model"
)

// walkBeancount walks a beancount parse tree and populates the diagram.
func walkBeancount(root *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	txSeq := 0
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil || child.Type(lang) != "transaction" {
			continue
		}
		tx, err := parseBeancountTransaction(child, lang, src, d, txSeq)
		if err != nil {
			return err
		}
		d.Transactions = append(d.Transactions, tx)
		txSeq++
	}
	return nil
}

type bcPosting struct {
	account  string
	amount   float64
	elided   bool
	currency string
}

func parseBeancountTransaction(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram, seq int) (model.Transaction, error) {
	var tx model.Transaction
	var postings []bcPosting

	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "date":
			tx.Date = child.Text(src)
		case "txn":
			tx.Flag = child.Text(src)
		case "payee":
			tx.Payee = strings.Trim(child.Text(src), "\"")
		case "narration":
			if tx.Payee == "" {
				tx.Payee = strings.Trim(child.Text(src), "\"")
			}
		case "posting":
			p, err := parseBeancountPosting(child, lang, src)
			if err != nil {
				return tx, fmt.Errorf("line %d: %w", child.StartPoint().Row+1, err)
			}
			postings = append(postings, p)
		}
	}

	// Resolve elided amount
	if err := resolveElidedAmount(postings); err != nil {
		return tx, fmt.Errorf("line %d: %w", node.StartPoint().Row+1, err)
	}

	// Convert postings to movements
	tx.Movements = postingsToMovements(postings, tx.Date, tx.Payee, seq, d)
	return tx, nil
}

func parseBeancountPosting(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) (bcPosting, error) {
	var p bcPosting
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "account":
			p.account = child.Text(src)
		case "incomplete_amount":
			amt, cur, err := parseBeancountAmount(child, lang, src)
			if err != nil {
				return p, err
			}
			p.amount = amt
			p.currency = cur
		}
	}
	if p.account != "" && p.currency == "" {
		p.elided = true
	}
	return p, nil
}

func parseBeancountAmount(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) (float64, string, error) {
	var amount float64
	var currency string
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "number":
			val, err := parseAmountStr(child.Text(src))
			if err != nil {
				return 0, "", fmt.Errorf("invalid number %q: %w", child.Text(src), err)
			}
			amount = val
		case "unary_number_expr":
			val, err := parseUnaryExpr(child, lang, src)
			if err != nil {
				return 0, "", err
			}
			amount = val
		case "currency":
			currency = child.Text(src)
		}
	}
	return amount, currency, nil
}

func parseUnaryExpr(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) (float64, error) {
	neg := false
	var val float64
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "minus":
			neg = true
		case "number":
			v, err := parseAmountStr(child.Text(src))
			if err != nil {
				return 0, err
			}
			val = v
		}
	}
	if neg {
		val = -val
	}
	return val, nil
}

func resolveElidedAmount(postings []bcPosting) error {
	elidedIdx := -1
	sum := 0.0
	for i, p := range postings {
		if p.elided {
			if elidedIdx >= 0 {
				return fmt.Errorf("multiple elided postings")
			}
			elidedIdx = i
		} else {
			sum += p.amount
		}
	}
	if elidedIdx >= 0 {
		postings[elidedIdx].amount = -sum
		postings[elidedIdx].elided = false
		// Inherit currency from other postings
		for _, p := range postings {
			if p.currency != "" {
				postings[elidedIdx].currency = p.currency
				break
			}
		}
	}
	return nil
}

func postingsToMovements(postings []bcPosting, date, payee string, seq int, d *model.Diagram) []model.Movement {
	var negatives, positives []bcPosting
	for _, p := range postings {
		if p.amount < 0 {
			negatives = append(negatives, p)
		} else if p.amount > 0 {
			positives = append(positives, p)
		}
	}

	// 2 postings: direct movement (negative→positive)
	if len(postings) == 2 && len(negatives) == 1 && len(positives) == 1 {
		return []model.Movement{{
			From:      d.GetOrCreateAccount(negatives[0].account),
			To:        d.GetOrCreateAccount(positives[0].account),
			Amount:    math.Abs(negatives[0].amount),
			Commodity: positives[0].currency,
		}}
	}

	// 3+ postings: use hub
	hub := d.GetOrCreateHubAccount(date, payee, seq)
	var moves []model.Movement
	for _, p := range negatives {
		moves = append(moves, model.Movement{
			From:      d.GetOrCreateAccount(p.account),
			To:        hub,
			Amount:    math.Abs(p.amount),
			Commodity: p.currency,
		})
	}
	for _, p := range positives {
		moves = append(moves, model.Movement{
			From:      hub,
			To:        d.GetOrCreateAccount(p.account),
			Amount:    p.amount,
			Commodity: p.currency,
		})
	}
	return moves
}
