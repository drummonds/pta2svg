package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"git.bytestone.uk/hum3/pta2svg"
	"git.bytestone.uk/hum3/pta2svg/internal/highlight"
)

type fileData struct {
	Name   string
	Source template.HTML
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

const indexTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
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
  {{range .}}
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
    try {
      output.innerHTML = pta2svg.render(document.getElementById("source").value, opts);
      status.textContent = "";
      status.className = "help";
    } catch (e) {
      status.textContent = String(e.message || e);
      status.className = "help is-danger";
    }
  };
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

	// Write index page
	if err := writePage("docs/index.html", indexTmpl, sections); err != nil {
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
	if err := writePage("docs/try.html", tryTmpl, struct{ Example string }{string(example)}); err != nil {
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
	if err := pta2svg.Render(&buf, src, pta2svg.Options{Format: pta2svg.DetectFormat(path)}); err != nil {
		return fileData{}, fmt.Errorf("render: %w", err)
	}

	highlighted, err := highlight.HighlightHTML(path, src)
	if err != nil {
		return fileData{}, fmt.Errorf("highlight: %w", err)
	}

	return fileData{
		Name:   filepath.Base(path),
		Source: highlighted,
		SVG:    template.HTML(buf.String()),
	}, nil
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
