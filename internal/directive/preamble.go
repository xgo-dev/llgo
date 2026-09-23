package directive

import (
	"fmt"
	"go/token"
	"strings"
)

// Preamble is the C source attached to a single unsafe import. Its #cgo lines
// are recognized and parsed at discovery; target filtering and pkg-config
// execution remain build-driver operations.
type Preamble struct {
	Pos   token.Pos
	Lines []PreambleLine
}
type PreambleLine struct {
	Text string
	Cgo  *CgoCommand
}
type CgoCommand struct {
	Tag, Flag, Args string
	Err             error
}

func PreambleLines(text string) []PreambleLine {
	var out []PreambleLine
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		r := PreambleLine{Text: line}
		if strings.HasPrefix(line, "#cgo ") {
			c := ParseCgoCommand(line)
			r.Cgo = &c
		}
		out = append(out, r)
	}
	return out
}
func ParseCgoCommand(line string) (r CgoCommand) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		r.Err = fmt.Errorf("invalid cgo format: %v", line)
		return
	}
	decl := strings.TrimSpace(line[:idx])
	r.Args = strings.TrimSpace(line[idx+1:])
	parts := strings.SplitN(decl, " ", 2)
	if len(parts) < 2 {
		r.Err = fmt.Errorf("invalid cgo directive: %v", line)
		return
	}
	remaining := strings.TrimSpace(parts[1])
	if last := strings.LastIndex(remaining, " "); last >= 0 {
		r.Tag = strings.TrimSpace(remaining[:last])
		r.Flag = strings.TrimSpace(remaining[last+1:])
	} else {
		r.Flag = remaining
	}
	return
}
