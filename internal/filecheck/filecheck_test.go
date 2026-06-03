package filecheck

import (
	"regexp"
	"strings"
	"testing"
)

func TestMatchSupportsLabelRegexAndNext(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK-LABEL: func main
func main() {
	// CHECK: value {{[0-9]+}}
	// CHECK-NEXT: done
}
`
	input := "func main\nvalue 42\ndone\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchPreservesPatternWhitespace(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK:  value 42 
`
	input := " value 42 \n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchSupportsEmptyLine(t *testing.T) {
	spec := `// LITTEST
package main

func main() {
	// CHECK: value 42
	// CHECK-EMPTY:
	// CHECK: done
}
`
	input := "value 42\n\ndone\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchSupportsWholeLineChecks(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK-LINE: begin
// CHECK-LINE: end
`
	input := "begin\nmiddle\nend\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestCheckLabelRequiresWholeLineMatch(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK-LABEL: begin
`
	err := Match("test.go", spec, "prefix begin\n")
	if err == nil {
		t.Fatal("Match succeeded unexpectedly")
	}
}

func TestMatchSupportsSameAndNot(t *testing.T) {
	spec := `// LITTEST
package main

func main() {
	// CHECK: value
	// CHECK-SAME: 42
	// CHECK-NOT: forbidden
	// CHECK: done
}
`
	input := "value 42\ndone\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchRejectsForbiddenPattern(t *testing.T) {
	spec := `// LITTEST
package main

func main() {
	// CHECK: ok
	// CHECK-NOT: panic
	// CHECK: done
}
`
	err := Match("test.go", spec, "ok\npanic\ndone\n")
	requireErrContains(t, err, "forbidden text")
}

func TestMatchDoesNotCarryCheckNotAcrossLabels(t *testing.T) {
	spec := `// LITTEST
package main

// CHECK-LABEL: first
// CHECK: ok
// CHECK-NOT: panic
// CHECK-LABEL: second
// CHECK: panic
`
	input := "first\nok\nsecond\npanic\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestHasDirectives(t *testing.T) {
	ok, err := HasDirectives("// CHECK: value\n")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("HasDirectives returned false")
	}

	ok, err = HasDirectives("value\n")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("HasDirectives returned true unexpectedly")
	}
}

func TestHasDirectivesPropagatesDirectiveErrors(t *testing.T) {
	_, err := HasDirectives("// CHECK: {{[invalid\n")
	requireErrContains(t, err, "unterminated '{{' in pattern")
}

func TestHasDirectivesIgnoresNonSlashSlashComments(t *testing.T) {
	ok, err := HasDirectives("; CHECK: value\n")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("HasDirectives returned true unexpectedly")
	}
}

func TestMatchSupportsCRLF(t *testing.T) {
	spec := "// LITTEST\r\npackage main\r\n\r\n// CHECK-LABEL: begin\r\n// CHECK-NEXT: done\r\n"
	input := "begin\r\ndone\r\n"
	if err := Match("test.go", spec, input); err != nil {
		t.Fatal(err)
	}
}

func TestMatchReportsDirectiveAndPatternErrors(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want string
	}{
		{
			name: "empty pattern",
			spec: "// CHECK:\n",
			want: "empty pattern",
		},
		{
			name: "unterminated regex",
			spec: "// CHECK: {{[0-9]+\n",
			want: "unterminated '{{' in pattern",
		},
		{
			name: "invalid regex",
			spec: "// CHECK: {{(}}\n",
			want: "error parsing regexp",
		},
		{
			name: "unknown directive",
			spec: "// CHECK-BOGUS: value\n",
			want: "unknown directive CHECK-BOGUS",
		},
		{
			name: "empty pattern not allowed",
			spec: "// CHECK-EMPTY: value\n",
			want: "CHECK-EMPTY must not have a pattern",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Match("test.go", tc.spec, "")
			requireErrContains(t, err, tc.want)
		})
	}
}

func TestMatchReportsPreconditionErrors(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want string
	}{
		{
			name: "next",
			spec: "// CHECK-NEXT: value\n",
			want: "CHECK-NEXT requires a prior positive match",
		},
		{
			name: "same",
			spec: "// CHECK-SAME: value\n",
			want: "CHECK-SAME requires a prior positive match",
		},
		{
			name: "empty",
			spec: "// CHECK-EMPTY:\n",
			want: "CHECK-EMPTY requires a prior positive match",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Match("test.go", tc.spec, "value\n")
			requireErrContains(t, err, tc.want)
		})
	}
}

func TestMatchReportsSearchFailures(t *testing.T) {
	cases := []struct {
		name  string
		spec  string
		input string
		want  string
	}{
		{
			name:  "next wrong line",
			spec:  "// CHECK: value\n// CHECK-NEXT: done\n",
			input: "value\nother\n",
			want:  `CHECK-NEXT "done" did not match`,
		},
		{
			name:  "same wrong line",
			spec:  "// CHECK: value\n// CHECK-SAME: done\n",
			input: "value other\n",
			want:  `CHECK-SAME "done" did not match`,
		},
		{
			name:  "empty non-empty line",
			spec:  "// CHECK: value\n// CHECK-EMPTY:\n",
			input: "value\nother\n",
			want:  "CHECK-EMPTY expected input:2 to be empty",
		},
		{
			name:  "empty past end",
			spec:  "// CHECK: value\n// CHECK-EMPTY:\n",
			input: "value",
			want:  "CHECK-EMPTY expected an empty line after input:1",
		},
		{
			name:  "trailing not checks remaining input",
			spec:  "// CHECK: value\n// CHECK-NOT: forbidden\n",
			input: "value\nforbidden\n",
			want:  "forbidden text",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Match("test.go", tc.spec, tc.input)
			requireErrContains(t, err, tc.want)
		})
	}
}

func TestSearchHelpers(t *testing.T) {
	re := regexp.MustCompile("x")

	if got := trimDirectivePattern("\tvalue"); got != "value" {
		t.Fatalf("trimDirectivePattern(tab) = %q", got)
	}
	if got := trimDirectivePattern("value"); got != "value" {
		t.Fatalf("trimDirectivePattern(plain) = %q", got)
	}

	if !before(pos{line: 0, col: 1}, pos{line: 1, col: 0}) {
		t.Fatal("before returned false unexpectedly")
	}
	if !before(pos{line: 0, col: 1}, pos{line: 0, col: 2}) {
		t.Fatal("before returned false unexpectedly")
	}

	if _, ok := findForward([]string{"ab", "x"}, pos{line: 0, col: 10}, re); !ok {
		t.Fatal("findForward did not clamp and continue")
	}
	if _, ok := findLine([]string{"x"}, -1, 0, re); ok {
		t.Fatal("findLine matched unexpectedly")
	}
	if _, ok := findLine([]string{"x"}, 0, 10, re); ok {
		t.Fatal("findLine matched unexpectedly")
	}
	if _, _, ok := findForbidden([]string{"abc"}, pos{line: 0, col: 2}, pos{line: 0, col: 1}, re); ok {
		t.Fatal("findForbidden matched unexpectedly")
	}
}

func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}
