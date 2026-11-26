// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "strings"

func main() {
	s := "Hello, Go World"
	println("idx", strings.Index(s, "Go"))
	println("contains", strings.Contains(s, "Go"), strings.Contains(s, "go"))
	println("edge", strings.HasPrefix(s, "Hello"), strings.HasSuffix(s, "World"))
	parts := strings.Split("a,b,c", ",")
	println("split", len(parts), parts[0], parts[2])
	println("join", strings.Join(parts, "-"))
	println("upper", strings.ToUpper("abc123"))
	println("repeat", strings.Repeat("ab", 3))
	println("replace", strings.Replace("aaa", "a", "b", 2))
	println("trim", strings.TrimSpace("  hi  "))
	fs := strings.Fields("x  y z")
	println("fields", len(fs), fs[1])
	println("count", strings.Count("banana", "an"))
	println("builder", func() string {
		var b strings.Builder
		b.WriteString("bu")
		b.WriteString("ild")
		return b.String()
	}())
}
