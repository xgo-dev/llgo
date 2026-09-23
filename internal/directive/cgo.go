package directive

import (
	"go/ast"
	"strconv"
	"strings"
)

type DynamicImport struct{ Local, Alias string }
type Cgo struct {
	LDFlags []string
	Imports []DynamicImport
}

func (p *Cgo) add(c *ast.Comment) {
	for _, line := range splitCommentLines(c.Text) {
		switch {
		case strings.HasPrefix(line, "go:cgo_ldflag"):
			p.LDFlags = append(p.LDFlags, splitDirectiveArgs(strings.TrimSpace(strings.TrimPrefix(line, "go:cgo_ldflag")))...)
		case strings.HasPrefix(line, "go:cgo_import_dynamic"):
			toks := splitDirectiveArgs(strings.TrimSpace(strings.TrimPrefix(line, "go:cgo_import_dynamic")))
			if len(toks) == 0 {
				continue
			}
			local, alias := toks[0], toks[0]
			if len(toks) > 1 && toks[1] != "" {
				alias = toks[1]
			}
			if local != "" && alias != "" && local != alias {
				p.Imports = append(p.Imports, DynamicImport{local, alias})
			}
		}
	}
}
func CollectCgo(files []*File) (out Cgo) {
	for _, f := range files {
		out.LDFlags = append(out.LDFlags, f.Cgo.LDFlags...)
		out.Imports = append(out.Imports, f.Cgo.Imports...)
	}
	return
}
func splitCommentLines(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitDirectiveArgs(s string) []string {
	fields := strings.Fields(strings.TrimSpace(s))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if u, err := strconv.Unquote(f); err == nil {
			out = append(out, u)
		} else {
			out = append(out, f)
		}
	}
	return out
}
