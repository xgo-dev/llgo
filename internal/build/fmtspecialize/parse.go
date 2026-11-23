package fmtspecialize

import "strings"

type FormatPart struct {
	Text  string
	Raw   string
	Verb  byte
	Width int
	Zero  bool
}

// ParseFormat accepts sequential operands, escaped percent signs, and literal
// widths. It leaves argument selection, types and dialect checks to consumers.
func ParseFormat(text string) ([]FormatPart, bool) {
	var parts []FormatPart
	for i := 0; i < len(text); {
		end := strings.IndexByte(text[i:], '%')
		if end < 0 {
			parts = append(parts, FormatPart{Text: text[i:]})
			break
		}
		if end != 0 {
			parts = append(parts, FormatPart{Text: text[i : i+end]})
			i += end
		}
		start := i
		i++
		if i == len(text) {
			return nil, false
		}
		if text[i] == '%' {
			parts = append(parts, FormatPart{Text: "%", Raw: "%%"})
			i++
			continue
		}
		part := FormatPart{}
		if text[i] == '0' {
			part.Zero = true
			i++
		}
		widthStart := i
		for i < len(text) && '0' <= text[i] && text[i] <= '9' {
			part.Width = part.Width*10 + int(text[i]-'0')
			if i-widthStart >= 2 {
				return nil, false
			}
			i++
		}
		if i >= len(text) || !(text[i] >= 'a' && text[i] <= 'z' || text[i] >= 'A' && text[i] <= 'Z') {
			return nil, false
		}
		part.Verb, part.Raw = text[i], text[start:i+1]
		parts = append(parts, part)
		i++
	}
	return parts, true
}
