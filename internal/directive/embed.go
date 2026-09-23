package directive

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"
)

// Embed retains parsed patterns and the original parse error until the consumer
// reaches the same diagnostic phase as before.
type Embed struct {
	Patterns []string
	Present  bool
	Err      error
}

func scanEmbed(doc *ast.CommentGroup) Embed {
	p, h, e := parseEmbedPatterns(doc)
	return Embed{p, h, e}
}
func EmbedPatterns(groups ...*Group) (patterns []string, present bool, err error) {
	for _, g := range groups {
		present = present || g.Embed.Present
		if g.Embed.Err != nil {
			return nil, present, g.Embed.Err
		}
		patterns = append(patterns, g.Embed.Patterns...)
	}
	return
}
func parseEmbedPatterns(docs ...*ast.CommentGroup) (patterns []string, hasDirective bool, err error) {
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			if c == nil {
				continue
			}
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			args, ok := EmbedDirective(line)
			if !ok {
				continue
			}
			hasDirective = true
			if args == "" {
				return nil, hasDirective, fmt.Errorf("invalid //go:embed: missing pattern")
			}
			fields, err := SplitEmbedArgs(args)
			if err != nil {
				return nil, hasDirective, err
			}
			for _, f := range fields {
				if uq, err := strconv.Unquote(f); err == nil {
					patterns = append(patterns, uq)
				} else {
					if len(f) > 0 && (f[0] == '"' || f[0] == '`') {
						return nil, hasDirective, fmt.Errorf("invalid //go:embed quoted pattern %q", f)
					}
					patterns = append(patterns, f)
				}
			}
		}
	}
	return patterns, hasDirective, nil
}

func EmbedDirective(line string) (args string, ok bool) {
	if !strings.HasPrefix(line, "go:embed") {
		return "", false
	}
	if len(line) == len("go:embed") {
		return "", true
	}
	ch := line[len("go:embed")]
	if ch != ' ' && ch != '\t' {
		return "", false
	}
	return strings.TrimSpace(line[len("go:embed"):]), true
}

func SplitEmbedArgs(s string) ([]string, error) {
	var out []string
	for i := 0; i < len(s); {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		start := i
		if s[i] == '"' || s[i] == '`' {
			quote := s[i]
			i++
			closed := false
			for i < len(s) {
				if s[i] == quote {
					i++
					closed = true
					break
				}
				if quote == '"' && s[i] == '\\' && i+1 < len(s) {
					i += 2
					continue
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("invalid //go:embed quoted pattern")
			}
			out = append(out, s[start:i])
			continue
		}
		for i < len(s) && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		out = append(out, s[start:i])
	}
	return out, nil
}

// IsEmbedComment retains the loader's line-comment-only recognition rule.
func IsEmbedComment(c *ast.Comment) bool {
	if c == nil || !strings.HasPrefix(c.Text, "//") {
		return false
	}
	_, ok := EmbedDirective(strings.TrimSpace(strings.TrimPrefix(c.Text, "//")))
	return ok
}
