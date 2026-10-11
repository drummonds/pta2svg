package parse

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"git.bytestone.uk/hum3/pta2svg"
)

// Every testdata file drawn from its text equals the same file drawn from
// its model after a trip through JSON: the model carries everything the
// renderer uses.
func TestModelRoundTripsTestdata(t *testing.T) {
	var files []string
	for _, pattern := range []string{"*/*.pta", "*/*.goluca", "*/*.beancount"} {
		m, err := filepath.Glob(filepath.Join("..", "testdata", pattern))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no testdata files found")
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(src, DetectFormat(file))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			raw, err := json.Marshal(pta2svg.ToModel(d))
			if err != nil {
				t.Fatal(err)
			}
			var m pta2svg.Model
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			back, err := m.Diagram()
			if err != nil {
				t.Fatalf("Diagram: %v", err)
			}
			var want, got bytes.Buffer
			if err := pta2svg.RenderDiagram(&want, d, pta2svg.Options{}); err != nil {
				t.Fatal(err)
			}
			if err := pta2svg.RenderDiagram(&got, back, pta2svg.Options{}); err != nil {
				t.Fatal(err)
			}
			if want.String() != got.String() {
				t.Error("SVG differs after the JSON round trip")
			}
		})
	}
}
