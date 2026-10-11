//go:build js && wasm

// Command pta2svg-wasm-lite is the small browser build of pta2svg: no text
// parsers, so a page that already holds the ledger as data does not pay
// for the tree-sitter grammars. Loaded with Go's wasm_exec.js, it installs
// one function on the page:
//
//	pta2svg.renderModel(model, {title, animate, layout}) -> svg string
//
// model is a pta2svg.Model as JSON text or as an object. A model or layout
// error is thrown as a JS Error.
package main

import (
	"syscall/js"

	"git.bytestone.uk/hum3/pta2svg/internal/wasm"
)

func main() {
	js.Global().Set("pta2svg", map[string]any{
		"renderModel": wasm.Func("pta2svg.renderModel", wasm.RenderModel),
	})
	select {} // keep the Go runtime alive for callbacks
}
