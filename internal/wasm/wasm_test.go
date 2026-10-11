//go:build js && wasm

package wasm

import (
	"errors"
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
	for _, want := range []string{"<svg", "<title>Interest</title>", "1001.27 GBP", "interest"} {
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
	if !strings.Contains(svg, "1001.27 GBP") {
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
	got := Options(obj)
	want := pta2svg.Options{Title: "T", Animate: true, Layout: pta2svg.Sugiyama}
	if got != want {
		t.Errorf("Options = %+v, want %+v", got, want)
	}
	if Options(js.Undefined()) != (pta2svg.Options{}) {
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
	if msg := thrown(t, func() { f.Invoke(false) }); !strings.Contains(msg, "boom") {
		t.Errorf("thrown message %q should carry the error", msg)
	}
	if msg := thrown(t, func() { f.Invoke() }); !strings.Contains(msg, "argument required") {
		t.Errorf("thrown message %q should say the argument is required", msg)
	}
}

// thrown runs call, which must throw a JS Error, and returns its message.
func thrown(t *testing.T, call func()) (msg string) {
	t.Helper()
	defer func() {
		e, ok := recover().(js.Error)
		if !ok {
			t.Fatal("call should have thrown a JS Error")
		}
		msg = e.Value.Get("message").String()
	}()
	call()
	t.Fatal("call returned without throwing")
	return ""
}
