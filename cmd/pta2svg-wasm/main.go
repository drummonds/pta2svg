//go:build js && wasm

// Command pta2svg-wasm is the browser build of pta2svg. Loaded with Go's
// wasm_exec.js, it installs one function on the page:
//
//	pta2svg.render(text, {format, title, animate, layout}) -> svg string
//
// format is "pta" (default), "goluca" or "beancount"; layout is "lr"
// (default) or "sugiyama". A parse or layout error is thrown as a JS Error.
package main

import (
	"bytes"
	"syscall/js"

	"git.bytestone.uk/hum3/pta2svg"
)

func main() {
	js.Global().Set("pta2svg", map[string]any{
		"render": js.FuncOf(render),
	})
	select {} // keep the Go runtime alive for callbacks
}

func render(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return throw("pta2svg.render: text argument required")
	}
	var opts js.Value
	if len(args) > 1 {
		opts = args[1]
	}
	svg, err := renderText(args[0].String(), opts)
	if err != nil {
		return throw(err.Error())
	}
	return svg
}

// renderText is the JS-free core, so a test can drive it with plain values.
func renderText(text string, opts js.Value) (string, error) {
	o := pta2svg.Options{}
	if opts.Type() == js.TypeObject {
		if v := opts.Get("format"); v.Type() == js.TypeString {
			f, err := pta2svg.ParseFormat(v.String())
			if err != nil {
				return "", err
			}
			o.Format = f
		}
		if v := opts.Get("title"); v.Type() == js.TypeString {
			o.Title = v.String()
		}
		if v := opts.Get("animate"); v.Type() == js.TypeBoolean {
			o.Animate = v.Bool()
		}
		if v := opts.Get("layout"); v.Type() == js.TypeString {
			o.Layout = pta2svg.Layout(v.String())
		}
	}
	var buf bytes.Buffer
	if err := pta2svg.Render(&buf, []byte(text), o); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// throw raises a JS Error; the caller's return value is discarded.
func throw(msg string) any {
	js.Global().Get("Error").New(msg).Call("toString")
	panic(js.Error{Value: js.Global().Get("Error").New(msg)})
}
