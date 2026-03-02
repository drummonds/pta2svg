package main

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drummonds/pta2svg/internal/layout"
	"github.com/drummonds/pta2svg/internal/parser"
	"github.com/drummonds/pta2svg/internal/render"
)

type fileData struct {
	Name   string
	Source string
	SVG    template.HTML
}

const tmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>pta2svg — Demo</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.4/css/bulma.min.css">
<style>
  pre.source { background: #1e1e2e; color: #cdd6f4; padding: 1rem; border-radius: 6px; overflow-x: auto; font-size: 0.85rem; }
  .svg-container { display: flex; align-items: center; justify-content: center; overflow-x: auto; }
  .svg-container svg { max-width: 100%; height: auto; }
</style>
</head>
<body>
<section class="section">
<div class="container">
  <h1 class="title">pta2svg — Demo</h1>
  <p class="subtitle">Generate SVG flowcharts from plain text accounting files</p>
  {{range .}}
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
	files, err := filepath.Glob("testdata/*.pta")
	if err != nil {
		fmt.Fprintf(os.Stderr, "glob: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(files)

	var data []fileData
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			os.Exit(1)
		}

		d, err := parser.Parse(strings.NewReader(string(src)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", path, err)
			os.Exit(1)
		}

		g := layout.GraphFromDiagram(d, layout.LR{}, layout.DefaultOptions())

		var buf bytes.Buffer
		err = render.Render(&buf, g, render.Options{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "render %s: %v\n", path, err)
			os.Exit(1)
		}

		data = append(data, fileData{
			Name:   filepath.Base(path),
			Source: html.EscapeString(string(src)),
			SVG:    template.HTML(buf.String()),
		})
	}

	t, err := template.New("page").Parse(tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "template: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	out, err := os.Create("docs/index.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if err := t.Execute(out, data); err != nil {
		fmt.Fprintf(os.Stderr, "execute: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("wrote docs/index.html")
}
