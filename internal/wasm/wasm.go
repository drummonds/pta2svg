//go:build js && wasm

// Package wasm is what the browser builds share: reading render options
// from a JS object, rendering a Model handed over as JSON text or as an
// object, and wrapping a renderer as a page function that throws on error.
package wasm

import (
	"bytes"
	"encoding/json"
	"errors"
	"syscall/js"

	"git.bytestone.uk/hum3/pta2svg"
)

// Renderer turns the page's argument into SVG under the given options.
type Renderer func(arg js.Value, opts js.Value) (string, error)

// Options reads {title, animate, layout} from a JS object. Anything else,
// undefined included, gives the zero Options.
func Options(v js.Value) pta2svg.Options {
	o := pta2svg.Options{}
	if v.Type() != js.TypeObject {
		return o
	}
	if t := v.Get("title"); t.Type() == js.TypeString {
		o.Title = t.String()
	}
	if a := v.Get("animate"); a.Type() == js.TypeBoolean {
		o.Animate = a.Bool()
	}
	if l := v.Get("layout"); l.Type() == js.TypeString {
		o.Layout = pta2svg.Layout(l.String())
	}
	return o
}

// RenderModel draws a pta2svg.Model given as JSON text or as a plain object.
func RenderModel(arg js.Value, opts js.Value) (string, error) {
	var raw string
	switch arg.Type() {
	case js.TypeString:
		raw = arg.String()
	case js.TypeObject:
		raw = js.Global().Get("JSON").Call("stringify", arg).String()
	default:
		return "", errors.New("renderModel: model must be JSON text or an object")
	}
	var m pta2svg.Model
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", err
	}
	d, err := m.Diagram()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := pta2svg.RenderDiagram(&buf, d, Options(opts)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Func wraps r as a page function taking (arg, opts) and returning the SVG
// string; a missing argument or a render error is thrown as a JS Error.
func Func(name string, r Renderer) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			throw(name + ": argument required")
		}
		opts := js.Undefined()
		if len(args) > 1 {
			opts = args[1]
		}
		svg, err := r(args[0], opts)
		if err != nil {
			throw(err.Error())
		}
		return svg
	})
}

// throw raises a JS Error; the caller's return value is discarded.
func throw(msg string) {
	panic(js.Error{Value: js.Global().Get("Error").New(msg)})
}
