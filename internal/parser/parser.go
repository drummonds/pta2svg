package parser

import (
	"fmt"
	"strconv"
	"strings"

	"git.bytestone.uk/hum3/gotreesitter"
	"git.bytestone.uk/hum3/gotreesitter/grammars"
	"git.bytestone.uk/hum3/pta2svg/internal/model"
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
	if root.HasError() {
		return nil, syntaxError(root, lang, src)
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

// syntaxError describes the first error or missing node in the tree.
func syntaxError(root *gotreesitter.Node, lang *gotreesitter.Language, src []byte) error {
	if n := firstErrorNode(root); n != nil {
		text := n.Text(src)
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[:i]
		}
		if n.IsMissing() {
			return fmt.Errorf("line %d: missing %s", n.StartPoint().Row+1, n.Type(lang))
		}
		return fmt.Errorf("line %d: unexpected %q", n.StartPoint().Row+1, text)
	}
	return fmt.Errorf("syntax error")
}

func firstErrorNode(n *gotreesitter.Node) *gotreesitter.Node {
	if n.IsError() || n.IsMissing() {
		// Prefer the innermost error so the line points at the fault.
		for i := 0; i < n.ChildCount(); i++ {
			if c := firstErrorNode(n.Child(i)); c != nil {
				return c
			}
		}
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if c := firstErrorNode(n.Child(i)); c != nil {
			return c
		}
	}
	return nil
}
