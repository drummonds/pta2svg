// Command pta2svg draws a plain text accounting file as an SVG flow diagram.
package main

import (
	"flag"
	"fmt"
	"os"

	"git.bytestone.uk/hum3/pta2svg"
	"git.bytestone.uk/hum3/pta2svg/parse"
)

func main() {
	output := flag.String("o", "", "output file (default: stdout)")
	animate := flag.Bool("animate", false, "enable CSS flow animations")
	title := flag.String("title", "", "SVG title")
	layoutName := flag.String("layout", "sugiyama", "layout engine: sugiyama, lr")
	formatStr := flag.String("format", "auto", "input format: auto, pta, goluca, beancount")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: pta2svg [flags] <file>\n")
		os.Exit(1)
	}

	if err := run(flag.Arg(0), *output, *formatStr, pta2svg.Options{
		Title:   *title,
		Animate: *animate,
		Layout:  pta2svg.Layout(*layoutName),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(filename, output, formatStr string, opts pta2svg.Options) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	format := parse.DetectFormat(filename)
	if formatStr != "auto" {
		if format, err = parse.ParseFormat(formatStr); err != nil {
			return err
		}
	}

	w := os.Stdout
	if output != "" {
		if w, err = os.Create(output); err != nil {
			return err
		}
		defer w.Close()
	}
	return parse.Render(w, src, format, opts)
}
