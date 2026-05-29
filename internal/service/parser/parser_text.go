package parser

import (
	"strings"
	"unicode"
)

func CleanText(text string) string {
	text = strings.ReplaceAll(text, "\ufeff", "")
	text = strings.ReplaceAll(text, "\u3000", " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = privateUse.ReplaceAllString(text, "")

	var b strings.Builder
	lastSpace := false
	newlines := 0
	for _, r := range text {
		if r == '\n' {
			newlines++
			lastSpace = false
			if newlines <= 2 {
				b.WriteRune(r)
			}
			continue
		}
		newlines = 0
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		lastSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func collapseInline(text string) string {
	lines := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' })
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(compact(lines), " ")
}

func compact(items []string) []string {
	var out []string
	for _, item := range items {
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}


