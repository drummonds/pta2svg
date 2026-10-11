// Package parse reads plain text accounting files into a pta2svg.Diagram.
//
// It lives apart from the root package because its tree-sitter grammars
// account for most of the size of a browser build; a page that already
// holds the ledger as data imports only pta2svg.
package parse

import (
	"fmt"
	"io"
	"strings"

	"git.bytestone.uk/hum3/pta2svg"
	"git.bytestone.uk/hum3/pta2svg/internal/parser"
)

// Format identifies a plain text accounting format.
type Format int

const (
	PTA       Format = iota // directional movements with free-text descriptions (.pta)
	Goluca                  // go-luca's format: quoted descriptions, directives (.goluca)
	Beancount               // beancount postings with signed amounts (.beancount)
)

func (f Format) String() string {
	switch f {
	case PTA:
		return "pta"
	case Goluca:
		return "goluca"
	case Beancount:
		return "beancount"
	}
	return fmt.Sprintf("Format(%d)", int(f))
}

// DetectFormat picks the format from a filename's extension. Unknown
// extensions are PTA.
func DetectFormat(filename string) Format {
	switch {
	case strings.HasSuffix(filename, ".goluca"):
		return Goluca
	case strings.HasSuffix(filename, ".beancount"):
		return Beancount
	default:
		return PTA
	}
}

// ParseFormat reads a format name as a user would type it: pta, goluca or
// beancount, in any case.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "pta":
		return PTA, nil
	case "goluca":
		return Goluca, nil
	case "beancount":
		return Beancount, nil
	}
	return 0, fmt.Errorf("unknown format %q (use pta, goluca or beancount)", s)
}

// Parse reads src in the given format.
func Parse(src []byte, format Format) (*pta2svg.Diagram, error) {
	f, err := internalFormat(format)
	if err != nil {
		return nil, err
	}
	return parser.Parse("", src, f)
}

// Render parses src in the given format and writes the diagram as SVG to w.
func Render(w io.Writer, src []byte, format Format, opts pta2svg.Options) error {
	d, err := Parse(src, format)
	if err != nil {
		return err
	}
	return pta2svg.RenderDiagram(w, d, opts)
}

func internalFormat(f Format) (parser.Format, error) {
	switch f {
	case PTA:
		return parser.FormatPTA, nil
	case Goluca:
		return parser.FormatGoluca, nil
	case Beancount:
		return parser.FormatBeancount, nil
	}
	return 0, fmt.Errorf("unknown format %v", f)
}
