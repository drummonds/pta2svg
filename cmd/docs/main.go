package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"git.bytestone.uk/hum3/pta2svg"
	"git.bytestone.uk/hum3/pta2svg/internal/highlight"
	"git.bytestone.uk/hum3/pta2svg/parse"
)

type fileData struct {
	Name   string
	Source template.HTML
	Model  string // the ledger as the JSON model renderModel takes
	SVG    template.HTML
}

type section struct {
	Title string
	Slug  string // html filename without extension
	Dir   string
	Files []fileData
}

var sections = []section{
	{Title: "Orange Trading Business", Slug: "oranges", Dir: "testdata/oranges"},
	{Title: "General Examples", Slug: "examples", Dir: "testdata/examples"},
}

// favicon is a to b: two account boxes in the diagram's colours joined by
// an arrow, as the diagrams draw a movement.
const favicon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
<rect x="2" y="12" width="22" height="40" rx="5" fill="#BCE" stroke="#555" stroke-width="3"/>
<rect x="40" y="12" width="22" height="40" rx="5" fill="#CDA" stroke="#555" stroke-width="3"/>
<path d="M24 32 H32" stroke="#555" stroke-width="5"/>
<polygon points="31,24 40,32 31,40" fill="#555"/>
<text x="13" y="40" font-family="sans-serif" font-size="24" font-weight="bold" fill="#333" text-anchor="middle">a</text>
<text x="51" y="40" font-family="sans-serif" font-size="24" font-weight="bold" fill="#333" text-anchor="middle">b</text>
</svg>
`

const indexTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<link rel="icon" href="favicon.svg" type="image/svg+xml">
<title>pta2svg — Demo</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.4/css/bulma.min.css">
</head>
<body>
<section class="section">
<div class="container">
  <h1 class="title">pta2svg — Demo</h1>
  <p class="subtitle">Generate SVG flowcharts from plain text accounting files</p>
  <div class="content">
  <div class="box">
    <h2 class="title is-4"><a href="try.html">Try it</a></h2>
    <p>Type a ledger and watch it drawn in the browser (WebAssembly build).</p>
  </div>
  {{with .Builds}}
  <div class="box">
    <h2 class="title is-4">Browser builds</h2>
    <p>The tree-sitter grammars behind <code>render</code> are most of the full build. A page that holds the ledger as data loads a build without them and calls <code>renderModel</code> with the JSON model; TinyGo shrinks that further.</p>
    <table class="table is-narrow">
      <thead><tr><th>file</th><th>compiler</th><th>parser</th><th>functions</th><th class="has-text-right">size</th><th class="has-text-right">gzipped</th><th class="has-text-right">of full</th></tr></thead>
      <tbody>
      {{range .}}
        <tr><td><code>{{.File}}</code></td><td>{{.Compiler}}</td><td>{{if .Parser}}yes{{else}}no{{end}}</td><td><code>{{.Functions}}</code></td><td class="has-text-right">{{.Size}}</td><td class="has-text-right">{{.Gzipped}}</td><td class="has-text-right">{{.OfFull}}</td></tr>
      {{end}}
      </tbody>
    </table>
  </div>
  {{end}}
  {{range .Sections}}
  <div class="box">
    <h2 class="title is-4"><a href="{{.Slug}}.html">{{.Title}}</a></h2>
    <ul>
    {{range .Files}}
      <li>{{.Name}}</li>
    {{end}}
    </ul>
  </div>
  {{end}}
  </div>
</div>
</section>
</body>
</html>
`

// tryTmpl is the in-browser demo: pta2svg.wasm renders what is typed.
const tryTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<link rel="icon" href="favicon.svg" type="image/svg+xml">
<title>pta2svg — Try it</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.4/css/bulma.min.css">
<style>
  textarea.source { font-family: monospace; font-size: 0.85rem; min-height: 16rem; }
  .svg-container { display: flex; align-items: center; justify-content: center; overflow-x: auto; min-height: 8rem; }
  .svg-container svg { max-width: 100%; height: auto; }
</style>
<script src="wasm_exec.js"></script>
</head>
<body>
<section class="section">
<div class="container">
  <nav class="breadcrumb">
    <ul><li><a href="index.html">Home</a></li><li class="is-active"><a>Try it</a></li></ul>
  </nav>
  <h1 class="title">Try it</h1>
  <p class="subtitle">pta2svg compiled to WebAssembly: the diagram is drawn in your browser as you type.</p>
  <div class="columns">
    <div class="column is-5">
      <div class="field is-grouped">
        <div class="control">
          <div class="select"><select id="format">
            <option value="goluca">goluca</option>
            <option value="pta">pta</option>
            <option value="beancount">beancount</option>
            <option value="json">json model</option>
          </select></div>
        </div>
        <div class="control">
          <div class="select"><select id="layout">
            <option value="lr">lr</option>
            <option value="sugiyama">sugiyama</option>
          </select></div>
        </div>
        <div class="control"><label class="checkbox"><input type="checkbox" id="animate"> animate</label></div>
      </div>
      <textarea id="source" class="textarea source" spellcheck="false">{{.Example}}</textarea>
      <p id="status" class="help">Loading pta2svg.wasm…</p>
    </div>
    <div class="column is-7">
      <div id="output" class="svg-container box"></div>
    </div>
  </div>
</div>
</section>
<script>
(async () => {
  const status = document.getElementById("status");
  const output = document.getElementById("output");
  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(fetch("pta2svg.wasm"), go.importObject);
  go.run(result.instance); // installs window.pta2svg
  const draw = () => {
    const opts = {
      format: document.getElementById("format").value,
      layout: document.getElementById("layout").value,
      animate: document.getElementById("animate").checked,
    };
    const src = document.getElementById("source").value;
    try {
      output.innerHTML = opts.format === "json" ? pta2svg.renderModel(src, opts) : pta2svg.render(src, opts);
      status.textContent = "";
      status.className = "help";
    } catch (e) {
      status.textContent = String(e.message || e);
      status.className = "help is-danger";
    }
  };
  // Switching between text and the JSON model swaps in the example in that form.
  const examples = {text: {{.Example}}, json: {{.Model}}};
  let shown = document.getElementById("format").value === "json" ? "json" : "text";
  document.getElementById("format").addEventListener("input", (e) => {
    const want = e.target.value === "json" ? "json" : "text";
    if (want !== shown) {
      examples[shown] = document.getElementById("source").value;
      document.getElementById("source").value = examples[want];
      shown = want;
    }
  });
  for (const id of ["source", "format", "layout", "animate"]) {
    document.getElementById(id).addEventListener("input", draw);
  }
  draw();
})();
</script>
</body>
</html>
`

const pageTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<link rel="icon" href="favicon.svg" type="image/svg+xml">
<title>pta2svg — {{.Title}}</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.4/css/bulma.min.css">
<style>
  pre.source { background: #1e1e2e; color: #cdd6f4; padding: 1rem; border-radius: 6px; overflow-x: auto; font-size: 0.85rem; }
  .hl-keyword  { color: #cba6f7; }
  .hl-string   { color: #a6e3a1; }
  .hl-type     { color: #f9e2af; }
  .hl-number   { color: #fab387; }
  .hl-constant { color: #89dceb; }
  .hl-operator { color: #94e2d5; }
  .hl-comment  { color: #6c7086; }
  .svg-container { display: flex; align-items: center; justify-content: center; overflow-x: auto; }
  .svg-container svg { max-width: 100%; height: auto; }
</style>
</head>
<body>
<section class="section">
<div class="container">
  <nav class="breadcrumb">
    <ul><li><a href="index.html">Home</a></li><li class="is-active"><a>{{.Title}}</a></li></ul>
  </nav>
  <h1 class="title">{{.Title}}</h1>
  {{range .Files}}
  <div class="box mb-5">
    <h2 class="subtitle is-4">{{.Name}}</h2>
    <div class="columns">
      <div class="column is-5">
        <h3 class="heading">PTA Source</h3>
        <pre class="source">{{.Source}}</pre>
        <details class="mt-3">
          <summary class="heading">JSON model</summary>
          <pre class="source">{{.Model}}</pre>
        </details>
      </div>
      <div class="column is-7">
        <h3 class="heading">SVG Output</h3>
        <div class="svg-container">{{.SVG}}</div>
      </div>
    </div>
  </div>
  {{end}}
</div>
</section>
</body>
</html>
`

func main() {
	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	for i := range sections {
		if err := loadSection(&sections[i]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", sections[i].Dir, err)
			os.Exit(1)
		}
	}

	if err := os.WriteFile("docs/favicon.svg", []byte(favicon), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "favicon: %v\n", err)
		os.Exit(1)
	}

	// Write index page
	builds, err := measureBuilds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "builds: %v\n", err)
		os.Exit(1)
	}
	if err := writePage("docs/index.html", indexTmpl, struct {
		Builds   []build
		Sections []section
	}{builds, sections}); err != nil {
		fmt.Fprintf(os.Stderr, "index: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote docs/index.html")

	// Write the in-browser demo page
	example, err := os.ReadFile("testdata/examples/movement.goluca")
	if err != nil {
		fmt.Fprintf(os.Stderr, "example: %v\n", err)
		os.Exit(1)
	}
	model, err := jsonModel(example, parse.Goluca)
	if err != nil {
		fmt.Fprintf(os.Stderr, "example: %v\n", err)
		os.Exit(1)
	}
	if err := writePage("docs/try.html", tryTmpl, struct{ Example, Model string }{string(example), model}); err != nil {
		fmt.Fprintf(os.Stderr, "try: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote docs/try.html")

	// Write section pages
	for _, s := range sections {
		path := fmt.Sprintf("docs/%s.html", s.Slug)
		if err := writePage(path, pageTmpl, s); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func loadSection(s *section) error {
	var files []string
	for _, pattern := range []string{"*.goluca", "*.pta"} {
		matches, err := filepath.Glob(filepath.Join(s.Dir, pattern))
		if err != nil {
			return err
		}
		files = append(files, matches...)
	}
	sort.Strings(files)

	for _, path := range files {
		fd, err := renderFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		s.Files = append(s.Files, fd)
	}
	return nil
}

func renderFile(path string) (fileData, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return fileData{}, err
	}

	var buf bytes.Buffer
	if err := parse.Render(&buf, src, parse.DetectFormat(path), pta2svg.Options{}); err != nil {
		return fileData{}, fmt.Errorf("render: %w", err)
	}

	model, err := jsonModel(src, parse.DetectFormat(path))
	if err != nil {
		return fileData{}, err
	}

	highlighted, err := highlight.HighlightHTML(path, src)
	if err != nil {
		return fileData{}, fmt.Errorf("highlight: %w", err)
	}

	return fileData{
		Name:   filepath.Base(path),
		Source: highlighted,
		Model:  model,
		SVG:    template.HTML(buf.String()),
	}, nil
}

// build is a browser build as the index page lists it.
type build struct {
	File, Compiler, Functions string
	Parser                    bool
	Size, Gzipped, OfFull     string
}

// measureBuilds sizes the wasm files task wasm:build left in docs/, raw
// and gzipped, against the full build. None when they have not been built.
func measureBuilds() ([]build, error) {
	builds := []build{
		{File: "pta2svg.wasm", Compiler: "Go", Parser: true, Functions: "render, renderModel"},
		{File: "pta2svg-lite.wasm", Compiler: "Go", Functions: "renderModel"},
		{File: "pta2svg-tiny.wasm", Compiler: "TinyGo", Functions: "renderModel"},
	}
	var full float64
	for i := range builds {
		b, err := os.ReadFile(filepath.Join("docs", builds[i].File))
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var z bytes.Buffer
		w, _ := gzip.NewWriterLevel(&z, gzip.BestCompression)
		w.Write(b)
		w.Close()
		if i == 0 {
			full = float64(len(b))
		}
		builds[i].Size, builds[i].Gzipped = megabytes(len(b)), megabytes(z.Len())
		builds[i].OfFull = fmt.Sprintf("%.0f%%", 100*float64(len(b))/full)
	}
	return builds, nil
}

func megabytes(n int) string { return fmt.Sprintf("%.1f MB", float64(n)/1e6) }

// jsonModel is the ledger src as the indented JSON model.
func jsonModel(src []byte, format parse.Format) (string, error) {
	d, err := parse.Parse(src, format)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	b, err := json.MarshalIndent(pta2svg.ToModel(d), "", "  ")
	if err != nil {
		return "", fmt.Errorf("model: %w", err)
	}
	return string(b), nil
}

func writePage(path, tmplStr string, data any) error {
	t, err := template.New("page").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}
