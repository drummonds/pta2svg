package parse

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.bytestone.uk/hum3/pta2svg"
)

var update = flag.Bool("update", false, "rewrite testdata/golden from the current renderer")

// Every testdata ledger draws to the SVG recorded in testdata/golden, byte
// for byte. The two variants between them cover both layouts, the title and
// the animation. Run with -update to accept a deliberate change.
func TestRenderMatchesGolden(t *testing.T) {
	variants := []struct {
		name string
		opts pta2svg.Options
	}{
		{"lr", pta2svg.Options{Layout: pta2svg.LR}},
		{"sugiyama-animated", pta2svg.Options{Title: "Golden", Animate: true, Layout: pta2svg.Sugiyama}},
	}
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
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		for _, v := range variants {
			name := filepath.Base(filepath.Dir(file)) + "-" + base + "." + v.name + ".svg"
			t.Run(name, func(t *testing.T) {
				var got bytes.Buffer
				if err := Render(&got, src, DetectFormat(file), v.opts); err != nil {
					t.Fatalf("Render: %v", err)
				}
				golden := filepath.Join("..", "testdata", "golden", name)
				if *update {
					if err := os.WriteFile(golden, got.Bytes(), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("%v (run go test ./parse -update to create it)", err)
				}
				if !bytes.Equal(got.Bytes(), want) {
					t.Errorf("SVG differs from %s", golden)
				}
			})
		}
	}
}
