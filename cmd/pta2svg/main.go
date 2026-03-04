package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/drummonds/pta2svg/internal/layout"
	"github.com/drummonds/pta2svg/internal/parser"
	"github.com/drummonds/pta2svg/internal/render"
)

func main() {
	output := flag.String("o", "", "output file (default: stdout)")
	animate := flag.Bool("animate", false, "enable CSS flow animations")
	title := flag.String("title", "", "SVG title")
	layoutName := flag.String("layout", "lr", "layout engine")
	formatStr := flag.String("format", "auto", "input format: auto, pta, goluca, beancount")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: pta2svg [flags] <file>\n")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	format, err := parser.FormatFromString(*formatStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	d, err := parser.Parse(filename, src, format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	var l layout.Layouter
	switch *layoutName {
	case "lr":
		l = layout.LR{}
	default:
		fmt.Fprintf(os.Stderr, "unknown layout: %s\n", *layoutName)
		os.Exit(1)
	}

	g := layout.GraphFromDiagram(d, l, layout.DefaultOptions())

	w := os.Stdout
	if *output != "" {
		w, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer w.Close()
	}

	err = render.Render(w, g, render.Options{
		Title:   *title,
		Animate: *animate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}
}
