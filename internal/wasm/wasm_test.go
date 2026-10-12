//go:build js && wasm

package wasm

import (
	"errors"
	"reflect"
	"strings"
	"syscall/js"
	"testing"

	"git.bytestone.uk/hum3/pta2svg"
)

const interestJSON = `{"opening":{"Liability:Savings:alice":1000},"transactions":[{"date":"2026-01-31","flag":"*","payee":"Interest","movements":[{"from":"Expense:Interest","to":"Liability:Savings:alice","amount":1.27,"commodity":"GBP","description":"interest"}]}]}`

func TestRenderModelFromJSONString(t *testing.T) {
	opts := js.Global().Get("Object").New()
	opts.Set("title", "Interest")
	svg, err := RenderModel(js.ValueOf(interestJSON), opts)
	if err != nil {
		t.Fatalf("RenderModel: %v", err)
	}
	for _, want := range []string{"<svg", "<title>Interest</title>", "£1,001.27", "interest"} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg should contain %q", want)
		}
	}
}

func TestRenderModelFromObject(t *testing.T) {
	obj := js.Global().Get("JSON").Call("parse", interestJSON)
	svg, err := RenderModel(obj, js.Undefined())
	if err != nil {
		t.Fatalf("RenderModel: %v", err)
	}
	if !strings.Contains(svg, "£1,001.27") {
		t.Error("an object argument should render like its JSON")
	}
}

func TestRenderModelRejectsBadInput(t *testing.T) {
	if _, err := RenderModel(js.ValueOf(`{"transactions": [`), js.Undefined()); err == nil {
		t.Error("malformed JSON should be an error")
	}
	if _, err := RenderModel(js.ValueOf(`{"transactions":[{"date":"2026-01-01","movements":[{"to":"Expense:Food","amount":1}]}]}`), js.Undefined()); err == nil || !strings.Contains(err.Error(), "from") {
		t.Errorf("a movement without from should fail naming the field, got %v", err)
	}
	if _, err := RenderModel(js.ValueOf(42), js.Undefined()); err == nil {
		t.Error("a number is neither JSON text nor a model object")
	}
}

func TestOptions(t *testing.T) {
	obj := js.Global().Get("Object").New()
	obj.Set("title", "T")
	obj.Set("animate", true)
	obj.Set("layout", "sugiyama")
	gbp := js.Global().Get("Object").New()
	gbp.Set("symbol", "£")
	gbp.Set("decimals", 2)
	gbp.Set("thousands", ",")
	commodities := js.Global().Get("Object").New()
	commodities.Set("GBP", gbp)
	obj.Set("commodities", commodities)
	palette := js.Global().Get("Object").New()
	palette.Set("Asset", "#d1f0e2")
	obj.Set("palette", palette)
	obj.Set("shape", "stadium")
	strokes := js.Global().Get("Object").New()
	strokes.Set("Income", 3)
	obj.Set("strokes", strokes)
	obj.Set("balanceBelowName", true)
	obj.Set("style", "afp")
	got := Options(obj)
	want := pta2svg.Options{Title: "T", Animate: true, Layout: pta2svg.Sugiyama,
		Commodities: map[string]pta2svg.Commodity{"GBP": {Symbol: "£", Decimals: 2, Thousands: ","}},
		Palette:     map[string]string{"Asset": "#d1f0e2"},
		Shape:       pta2svg.Stadium, Strokes: map[string]float64{"Income": 3},
		BalanceBelowName: true, Style: "afp"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Options = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(Options(js.Undefined()), pta2svg.Options{}) {
		t.Error("undefined options should be the zero Options")
	}
}

// The page function throws a JS Error instead of crashing the Go runtime.
func TestFuncThrowsOnError(t *testing.T) {
	f := Func("test", func(arg, _ js.Value) (string, error) {
		if arg.Truthy() {
			return "ok", nil
		}
		return "", errors.New("boom")
	})
	if got := f.Invoke(true); got.String() != "ok" {
		t.Errorf("Invoke(true) = %v, want ok", got)
	}
	if msg := thrown(t, f, false); !strings.Contains(msg, "boom") {
		t.Errorf("thrown message %q should carry the error", msg)
	}
	if msg := thrown(t, f); !strings.Contains(msg, "argument required") {
		t.Errorf("thrown message %q should say the argument is required", msg)
	}
}

// thrown calls f with args, which must throw a JS Error, and returns its
// message. The throw is caught in JS, as a page would: TinyGo's wasm target
// cannot recover the Go panic that Invoke turns a throw into.
func thrown(t *testing.T, f js.Value, args ...any) string {
	t.Helper()
	e := catcher.Invoke(append([]any{f}, args...)...)
	if !e.InstanceOf(js.Global().Get("Error")) {
		t.Fatal("call should have thrown a JS Error")
	}
	return e.Get("message").String()
}

// catcher(f, ...args) is what f(...args) throws, or undefined.
var catcher = js.Global().Get("Function").New("f", "...args",
	"try { f(...args); } catch (e) { return e; }")
