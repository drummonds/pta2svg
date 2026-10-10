package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every file under testdata is a published example, so each must parse in
// the format its extension declares and draw at least one movement.
func TestTestdataParses(t *testing.T) {
	var files []string
	for _, pattern := range []string{"*/*.goluca", "*/*.pta", "*/*.beancount"} {
		matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", pattern))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no testdata files found")
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(path, src, FormatAuto)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(d.Transactions) == 0 {
				t.Fatal("no transactions parsed")
			}
			for _, tx := range d.Transactions {
				if len(tx.Movements) == 0 {
					t.Errorf("transaction %s %q has no movements", tx.Date, tx.Payee)
				}
			}
		})
	}
}

// A file the grammar rejects is reported as an error naming the line, not
// silently drawn as an empty diagram.
func TestParseReportsSyntaxError(t *testing.T) {
	input := "2024-01-15 * Salary deposit\n  Income:Salary -> Asset:Bank salary payment 5000.00 GBP\n"
	_, err := Parse("test.goluca", []byte(input), FormatGoluca)
	if err == nil {
		t.Fatal("expected a parse error for an unquoted goluca description, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "line 2") {
		t.Errorf("error should name line 2, got %q", got)
	}
}
