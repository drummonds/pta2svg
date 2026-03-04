package main

import (
	"bytes"
	"fmt"
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

const pageTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>pta2svg — {{.Title}}</title>
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
	files, err := filepath.Glob(filepath.Join(s.Dir, "*.pta"))
	if err != nil {
		return err
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

	d, err := parser.Parse(path, src, parser.FormatAuto)
	if err != nil {
		return fileData{}, fmt.Errorf("parse: %w", err)
	}

	g := layout.GraphFromDiagram(d, layout.LR{}, layout.DefaultOptions())

	var buf bytes.Buffer
	if err := render.Render(&buf, g, render.Options{}); err != nil {
		return fileData{}, fmt.Errorf("render: %w", err)
	}

	// Replace ASCII arrow with unicode for display
	display := strings.ReplaceAll(string(src), "->", "→")

	return fileData{
		Name:   filepath.Base(path),
		Source: display,
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
