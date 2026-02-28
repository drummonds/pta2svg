package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/drummonds/pta2svg/internal/model"
)

type state int

const (
	betweenTransactions state = iota
	inTransaction
)

// Parse reads a .pta file and returns a Diagram.
func Parse(r io.Reader) (*model.Diagram, error) {
	d := model.NewDiagram()
	scanner := bufio.NewScanner(r)
	st := betweenTransactions
	var cur model.Transaction
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		switch st {
		case betweenTransactions:
			line = strings.TrimSpace(line)
			if line == "" || line[0] == '#' {
				continue
			}
			tx, err := parseHeader(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			cur = tx
			st = inTransaction

		case inTransaction:
			if strings.TrimSpace(line) == "" {
				d.Transactions = append(d.Transactions, cur)
				cur = model.Transaction{}
				st = betweenTransactions
				continue
			}
			if !strings.HasPrefix(line, "  ") {
				return nil, fmt.Errorf("line %d: expected indented movement or blank line", lineNum)
			}
			m, err := parseMovement(strings.TrimSpace(line), d)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			cur.Movements = append(cur.Movements, m)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// finalize last transaction if no trailing blank
	if st == inTransaction && len(cur.Movements) > 0 {
		d.Transactions = append(d.Transactions, cur)
	}

	d.ComputeBalances()
	return d, nil
}

// parseHeader parses "2024-01-15 * Salary deposit"
func parseHeader(line string) (model.Transaction, error) {
	var tx model.Transaction
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return tx, fmt.Errorf("invalid header: %q", line)
	}
	tx.Date = parts[0]
	// validate date looks like YYYY-MM-DD
	if len(tx.Date) != 10 || tx.Date[4] != '-' || tx.Date[7] != '-' {
		return tx, fmt.Errorf("invalid date: %q", tx.Date)
	}
	tx.Flag = parts[1]
	if tx.Flag != "*" && tx.Flag != "!" {
		return tx, fmt.Errorf("invalid flag %q (expected * or !)", tx.Flag)
	}
	if len(parts) == 3 {
		tx.Payee = parts[2]
	}
	return tx, nil
}

// parseMovement parses "[+]from -> to [description] amount [commodity]"
func parseMovement(line string, d *model.Diagram) (model.Movement, error) {
	var m model.Movement

	// check linked prefix
	if strings.HasPrefix(line, "+") {
		m.Linked = true
		line = line[1:]
	}

	// find arrow
	arrowIdx, arrowLen := findArrow(line)
	if arrowIdx < 0 {
		return m, fmt.Errorf("no arrow found in movement: %q", line)
	}

	fromName := strings.TrimSpace(line[:arrowIdx])
	rest := strings.TrimSpace(line[arrowIdx+arrowLen:])

	if fromName == "" {
		return m, fmt.Errorf("empty source account")
	}

	// rest is: "to [description words...] amount [commodity]"
	// Strategy: split into tokens, account name is first token (no spaces in account names),
	// amount is last or second-to-last numeric token, commodity is last if non-numeric.
	tokens := strings.Fields(rest)
	if len(tokens) < 2 {
		return m, fmt.Errorf("need at least destination and amount: %q", rest)
	}

	toName := tokens[0]
	tokens = tokens[1:]

	// Find amount: scan from right. Last token might be commodity.
	var commodity string
	amountStr := tokens[len(tokens)-1]
	amountVal, err := parseAmount(amountStr)
	if err != nil {
		// last token is commodity, second-to-last is amount
		if len(tokens) < 2 {
			return m, fmt.Errorf("cannot parse amount from %q", rest)
		}
		commodity = amountStr
		amountStr = tokens[len(tokens)-2]
		amountVal, err = parseAmount(amountStr)
		if err != nil {
			return m, fmt.Errorf("cannot parse amount %q: %w", amountStr, err)
		}
		tokens = tokens[:len(tokens)-2]
	} else {
		tokens = tokens[:len(tokens)-1]
	}

	// remaining tokens are description
	description := strings.Join(tokens, " ")

	m.From = d.GetOrCreateAccount(fromName)
	m.To = d.GetOrCreateAccount(toName)
	m.Amount = amountVal
	m.Commodity = commodity
	m.Description = description
	return m, nil
}

// findArrow returns the index and length of the first arrow in line.
func findArrow(line string) (int, int) {
	// Check multi-char arrows first
	for _, arrow := range []string{"->", "//", "→"} {
		if idx := strings.Index(line, arrow); idx >= 0 {
			return idx, len(arrow)
		}
	}
	// Single char '>' — but must be surrounded by spaces to avoid matching account names
	for i, c := range line {
		if c == '>' {
			// check it's space-delimited
			if i > 0 && line[i-1] == ' ' && i+1 < len(line) && line[i+1] == ' ' {
				return i, 1
			}
		}
	}
	return -1, 0
}

func parseAmount(s string) (float64, error) {
	// strip commas for thousand separators
	s = strings.ReplaceAll(s, ",", "")
	// must start with digit or minus
	if len(s) == 0 {
		return 0, fmt.Errorf("empty")
	}
	if !unicode.IsDigit(rune(s[0])) && s[0] != '-' {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	return strconv.ParseFloat(s, 64)
}
