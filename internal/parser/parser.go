package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drummonds/gotreesitter"
	"github.com/drummonds/gotreesitter/grammars"
	"github.com/drummonds/pta2svg/internal/model"
)

// Parse parses src in the given format and returns a Diagram.
// If format is FormatAuto, the format is detected from the filename extension.
func Parse(filename string, src []byte, format Format) (*model.Diagram, error) {
	if format == FormatAuto {
		format = DetectFormat(filename)
	}

	var ext string
	switch format {
	case FormatPTA:
		ext = ".pta"
	case FormatGoluca:
		ext = ".goluca"
	case FormatBeancount:
		ext = ".beancount"
	default:
		return nil, fmt.Errorf("unsupported format")
	}

	// Grammars require a trailing newline to emit the last node.
	if len(src) > 0 && src[len(src)-1] != '\n' {
		src = append(src, '\n')
	}

	entry := grammars.DetectLanguage(ext)
	if entry == nil {
		return nil, fmt.Errorf("no grammar for %s", ext)
	}
	lang := entry.Language()
	p := gotreesitter.NewParser(lang)
	tree, err := p.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse: %w", err)
	}
	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("empty parse tree")
	}

	d := model.NewDiagram()
	switch format {
	case FormatPTA:
		err = walkPTA(root, lang, src, d)
	case FormatGoluca:
		err = walkGoluca(root, lang, src, d)
	case FormatBeancount:
		err = walkBeancount(root, lang, src, d)
	}
	if err != nil {
		return nil, err
	}

	d.ComputeBalances()
	return d, nil
}

// parseAmountStr parses an amount string, stripping commas.
func parseAmountStr(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}
