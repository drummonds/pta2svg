package highlight

import (
	"html"
	"html/template"
	"path/filepath"

	"github.com/drummonds/gotreesitter"
	"github.com/drummonds/gotreesitter/grammars"
)

// HighlightHTML returns syntax-highlighted HTML for the given source file.
// Highlighted tokens are wrapped in <span class="hl-{capture}"> elements.
// All text is HTML-escaped.
func HighlightHTML(filename string, src []byte) (template.HTML, error) {
	ext := filepath.Ext(filename)
	entry := grammars.DetectLanguage(ext)
	if entry == nil {
		// No grammar — return plain escaped text
		return template.HTML(html.EscapeString(string(src))), nil
	}

	lang := entry.Language()
	h, err := gotreesitter.NewHighlighter(lang, entry.HighlightQuery)
	if err != nil {
		return "", err
	}

	ranges := h.Highlight(src)

	var buf []byte
	pos := uint32(0)
	for _, r := range ranges {
		// Emit unhighlighted gap
		if r.StartByte > pos {
			buf = append(buf, html.EscapeString(string(src[pos:r.StartByte]))...)
		}
		// Emit highlighted span — use top-level capture name (before any dot)
		cls := topCapture(r.Capture)
		buf = append(buf, `<span class="hl-`...)
		buf = append(buf, cls...)
		buf = append(buf, `">`...)
		buf = append(buf, html.EscapeString(string(src[r.StartByte:r.EndByte]))...)
		buf = append(buf, `</span>`...)
		pos = r.EndByte
	}
	// Emit trailing unhighlighted text
	if int(pos) < len(src) {
		buf = append(buf, html.EscapeString(string(src[pos:]))...)
	}

	return template.HTML(buf), nil
}

// topCapture returns the part before the first dot, e.g. "punctuation.bracket" → "punctuation".
func topCapture(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i]
		}
	}
	return s
}
