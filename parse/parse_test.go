package parse

import (
	"bytes"
	"strings"
	"testing"

	"git.bytestone.uk/hum3/pta2svg"
)

const ptaSrc = "2026-01-31 balance Liability:Savings:alice 1000.00 GBP\n\n2026-01-31 * Interest\n  Expense:Interest -> Liability:Savings:alice interest 1.27 GBP\n"
const golucaSrc = "2026-01-31 * Interest\n  Expense:Interest -> \"Liability:Savings:Alice Lovelace:easy-access\" \"interest\" 1.27 GBP\n"

func TestRenderPTA(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, []byte(ptaSrc), PTA, pta2svg.Options{Title: "Interest"}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	svg := buf.String()
	for _, want := range []string{"<svg", "<title>Interest</title>", "alice", "£1,001.27", "interest"} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg should contain %q", want)
		}
	}
}

func TestRenderGoluca(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, []byte(golucaSrc), Goluca, pta2svg.Options{}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "easy-access") {
		t.Error("svg should contain the destination account's short name")
	}
}

func TestRenderSyntaxError(t *testing.T) {
	err := Render(&bytes.Buffer{}, []byte(ptaSrc), Goluca, pta2svg.Options{})
	if err == nil || !strings.Contains(err.Error(), "line") {
		t.Fatalf("pta text parsed as goluca should fail naming a line, got %v", err)
	}
}

func TestRenderPassesOptionsThrough(t *testing.T) {
	err := Render(&bytes.Buffer{}, []byte(ptaSrc), PTA, pta2svg.Options{Layout: "circular"})
	if err == nil {
		t.Fatal("unknown layout should be an error")
	}
}

func TestParseExposesTheModel(t *testing.T) {
	d, err := Parse([]byte(ptaSrc), PTA)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Transactions) != 1 {
		t.Fatalf("got %d transactions", len(d.Transactions))
	}
	if d.Accounts["Liability:Savings:alice"].Type != pta2svg.Liability {
		t.Error("account type should be Liability")
	}
	if d.Opening["Liability:Savings:alice"] != 1000 {
		t.Error("opening balance should be seeded")
	}
}

func TestDetectFormat(t *testing.T) {
	for file, want := range map[string]Format{"a.pta": PTA, "a.goluca": Goluca, "a.beancount": Beancount, "a.txt": PTA} {
		if got := DetectFormat(file); got != want {
			t.Errorf("DetectFormat(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	for s, want := range map[string]Format{"pta": PTA, "Goluca": Goluca, "beancount": Beancount} {
		got, err := ParseFormat(s)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %v, %v", s, got, err)
		}
	}
	if _, err := ParseFormat("ledger"); err == nil {
		t.Error("unknown format should be an error")
	}
}
