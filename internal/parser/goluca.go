package parser

import (
	"fmt"

	"github.com/drummonds/gotreesitter"
	"github.com/drummonds/pta2svg/internal/model"
)

// walkGoluca walks a goluca parse tree and populates the diagram.
func walkGoluca(root *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) error {
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil || child.Type(lang) != "transaction" {
			continue
		}
		tx, err := parseGolucaTransaction(child, lang, src, d)
		if err != nil {
			return err
		}
		d.Transactions = append(d.Transactions, tx)
	}
	return nil
}

func parseGolucaTransaction(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte, d *model.Diagram) (model.Transaction, error) {
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
			m, err := parseDirectionalMovement(child, lang, src, d, true)
			if err != nil {
				return tx, fmt.Errorf("line %d: %w", child.StartPoint().Row+1, err)
			}
			tx.Movements = append(tx.Movements, m)
		}
	}
	return tx, nil
}
