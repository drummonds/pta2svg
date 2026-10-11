//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"testing"
)

// The tree-sitter grammars run under wasm: parse, lay out and render goluca
// text to SVG inside the js/wasm runtime.
func TestRenderTextUnderWasm(t *testing.T) {
	src := "2026-01-31 balance Liability:Savings:alice 1000.00 GBP\n\n2026-01-31 * Interest\n  Expense:Interest -> Liability:Savings:alice \"interest\" 1.27 GBP\n"
	opts := js.Global().Get("Object").New()
	opts.Set("format", "goluca")
	opts.Set("title", "Interest")
	svg, err := renderText(js.ValueOf(src), opts)
	if err != nil {
		t.Fatalf("renderText: %v", err)
	}
	for _, want := range []string{"<svg", "<title>Interest</title>", "1001.27 GBP", "interest"} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg should contain %q", want)
		}
	}
}

func TestRenderTextDefaultsToPTA(t *testing.T) {
	svg, err := renderText(js.ValueOf("2026-01-01 * T\n  Asset:Cash -> Expense:Food lunch 10.00 GBP\n"), js.Undefined())
	if err != nil {
		t.Fatalf("renderText: %v", err)
	}
	if !strings.Contains(svg, "lunch") {
		t.Error("svg should contain the description")
	}
}

func TestRenderTextReportsErrors(t *testing.T) {
	opts := js.Global().Get("Object").New()
	opts.Set("format", "goluca")
	_, err := renderText(js.ValueOf("2026-01-01 * T\n  Asset:Cash -> Expense:Food lunch 10.00 GBP\n"), opts)
	if err == nil || !strings.Contains(err.Error(), "line") {
		t.Fatalf("pta text as goluca should fail naming a line, got %v", err)
	}
}
