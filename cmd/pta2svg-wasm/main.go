//go:build js && wasm

// Command pta2svg-wasm is the full browser build of pta2svg. Loaded with
// Go's wasm_exec.js, it installs two functions on the page:
//
//	pta2svg.render(text, {format, title, animate, layout}) -> svg string
//	pta2svg.renderModel(model, {title, animate, layout})   -> svg string
//
// format is "pta" (default), "goluca" or "beancount"; layout is
// "sugiyama" (default) or "lr"; model is a pta2svg.Model as JSON text or as an
// object. A parse, model or layout error is thrown as a JS Error.
//
// The text parsers are most of this binary's size; pta2svg-wasm-lite
// installs renderModel alone.
package main

import (
	"bytes"
	"errors"
	"syscall/js"

	"git.bytestone.uk/hum3/pta2svg/internal/wasm"
	"git.bytestone.uk/hum3/pta2svg/parse"
)

func main() {
	js.Global().Set("pta2svg", map[string]any{
		"render":      wasm.Func("pta2svg.render", renderText),
		"renderModel": wasm.Func("pta2svg.renderModel", wasm.RenderModel),
	})
	select {} // keep the Go runtime alive for callbacks
}

// renderText parses the text in opts.format, PTA by default.
func renderText(arg js.Value, opts js.Value) (string, error) {
	if arg.Type() != js.TypeString {
		return "", errors.New("pta2svg.render: text must be a string")
	}
	format := parse.PTA
	if opts.Type() == js.TypeObject {
		if v := opts.Get("format"); v.Type() == js.TypeString {
			f, err := parse.ParseFormat(v.String())
			if err != nil {
				return "", err
			}
			format = f
		}
	}
	var buf bytes.Buffer
	if err := parse.Render(&buf, []byte(arg.String()), format, wasm.Options(opts)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
