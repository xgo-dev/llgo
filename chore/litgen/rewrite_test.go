package main

import (
	"strings"
	"testing"
)

var testGenerationConfig = generationConfig{allFunctions: true, checkGlobals: "smart"}

func TestRewriteSource_InsertsMainAndClosure(t *testing.T) {
	const src = `// LITTEST
package main

func main() {
	fn := func() {}
	fn()
}
`
	const ir = `define void @"example.com/p.main"() {
_llgo_0:
  %0 = call ptr @"example.com/p.main$1"()
  ret void
}

define void @"example.com/p.main$1"() {
_llgo_0:
  ret void
}

`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	mainCheck := `// CHECK-LABEL: define void @"{{.*}}/p.main"(){{.*}} {`
	mainDecl := "func main() {"
	if !strings.Contains(got, mainCheck) {
		t.Fatalf("main checks not inserted before func main:\n%s", got)
	}
	if strings.Index(got, mainCheck) > strings.Index(got, mainDecl) {
		t.Fatalf("main checks should appear before func main:\n%s", got)
	}
	closureCheck := "\t// CHECK-LABEL: define void @\"{{.*}}/p.main$1\"(){{.*}} {"
	closureStmt := "\tfn := func() {}"
	if !strings.Contains(got, closureCheck) {
		t.Fatalf("closure checks not inserted before func literal:\n%s", got)
	}
	if strings.Index(got, closureCheck) > strings.Index(got, closureStmt) {
		t.Fatalf("closure checks should appear before func literal:\n%s", got)
	}
}

func TestRewriteSource_AddsInitAndCheckEmptyAndSkipsHelpers(t *testing.T) {
	const src = `// LITTEST
package main

var x = 1

func main() {}
`
	const ir = `define void @"example.com/p.init"() {
_llgo_0:
  br i1 true, label %_llgo_1, label %_llgo_2

_llgo_1:
  ret void

_llgo_2:
  ret void
}

define i1 @"example.com/runtime/internal/runtime.strequal"(ptr %0, ptr %1) {
_llgo_0:
  ret i1 true
}

define void @"example.com/p.main"() {
_llgo_0:
  ret void
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	initCheck := `// CHECK-LABEL: define void @"{{.*}}/p.init"(){{.*}} {`
	if !strings.Contains(got, initCheck) {
		t.Fatalf("init checks not inserted before var decl:\n%s", got)
	}
	if strings.Index(got, initCheck) > strings.Index(got, "var x = 1") {
		t.Fatalf("init checks should appear before var decl:\n%s", got)
	}
	if !strings.Contains(got, "// CHECK-EMPTY:") {
		t.Fatalf("blank IR lines should use CHECK-EMPTY:\n%s", got)
	}
	if strings.Contains(got, "runtime.strequal") {
		t.Fatalf("runtime.strequal helper should be skipped:\n%s", got)
	}
}

func TestRewriteSource_PreservesIROrderWhenAnchorMovesBackward(t *testing.T) {
	const src = `// LITTEST
package main

var seed = 40

func add(x, y int) int {
	return x + y
}

func main() {}
`
	const ir = `define i64 @"example.com/p.add"(i64 %0, i64 %1) {
_llgo_0:
  %2 = add i64 %0, %1
  ret i64 %2
}

define void @"example.com/p.init"() {
_llgo_0:
  ret void
}

define void @"example.com/p.main"() {
_llgo_0:
  ret void
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	addCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.add"(`
	addSame := `// CHECK-SAME: i64 %[[TMP0:[0-9]+]], i64 %[[TMP1:[0-9]+]]){{.*}} {`
	initCheck := `// CHECK-LABEL: define void @"{{.*}}/p.init"(){{.*}} {`
	if strings.Index(got, addCheck) < 0 || strings.Index(got, addSame) < 0 || strings.Index(got, initCheck) < 0 {
		t.Fatalf("missing checks:\n%s", got)
	}
	if strings.Index(got, addCheck) > strings.Index(got, initCheck) {
		t.Fatalf("IR order should be preserved even if init anchor is earlier:\n%s", got)
	}
}

func TestRewriteSource_AddsReferencedNumericGlobalsAtTop(t *testing.T) {
	const src = `// LITTEST
package main

func main() {}
`
	const ir = `@0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1
@1 = private unnamed_addr constant [3 x i8] c"%s\00", align 1
@"example.com/p.named" = global i64 1

define void @"example.com/p.main"() {
_llgo_0:
  call void @puts(ptr @0)
  call void @printf(ptr @1)
  ret void
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `// CHECK: {{^}}@[[GLOB0:[0-9]+]] = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1{{$}}`) {
		t.Fatalf("missing numeric global @0:\n%s", got)
	}
	if !strings.Contains(got, `// CHECK: {{^}}@[[GLOB1:[0-9]+]] = private unnamed_addr constant [3 x i8] c"%s\00", align 1{{$}}`) {
		t.Fatalf("missing numeric global @1:\n%s", got)
	}
	if strings.Contains(got, `// CHECK: {{^}}@"{{.*}}/p.named" = global i64 1{{$}}`) {
		t.Fatalf("named globals should not be emitted by default:\n%s", got)
	}
	if strings.Index(got, `// CHECK: {{^}}@[[GLOB0:[0-9]+]] = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1{{$}}`) > strings.Index(got, "func main()") {
		t.Fatalf("global checks should be placed before first declaration:\n%s", got)
	}
}

func TestRewriteSource_PreservesDeclarationDirectiveAdjacency(t *testing.T) {
	const src = `// LITTEST
package main

import _ "unsafe"

//go:linkname cSqrt C.sqrt
func cSqrt(float64) float64

func callSqrt(x float64) float64 {
	println("sqrt")
	return cSqrt(x)
}
`
	const ir = `@0 = private unnamed_addr constant [4 x i8] c"sqrt"

declare double @sqrt(double)

define double @"example.com/p.callSqrt"(double %0) {
_llgo_0:
  call void @"example.com/runtime/internal/runtime.PrintString"(ptr @0)
  %1 = call double @sqrt(double %0)
  ret double %1
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	want := `// LITTEST
package main

import _ "unsafe"

// CHECK: {{^}}@[[GLOB0:[0-9]+]] = private unnamed_addr constant [4 x i8] c"sqrt"{{$}}

//go:linkname cSqrt C.sqrt
func cSqrt(float64) float64

// CHECK-LABEL: define double @"{{.*}}/p.callSqrt"(
// CHECK-SAME: double %[[TMP0:[0-9]+]]){{.*}} {
// CHECK-NEXT: _llgo_[[BB0:[0-9]+]]:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(ptr @[[GLOB0]])
// CHECK-NEXT:   %[[TMP1:[0-9]+]] = call double @sqrt(double %[[TMP0]])
// CHECK-NEXT:   ret double %[[TMP1]]
// CHECK-NEXT: }

func callSqrt(x float64) float64 {
	println("sqrt")
	return cSqrt(x)
}
`
	if got != want {
		t.Fatalf("rewriteSource mismatch:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestRewriteSource_PreservesDirectiveBeforeInlineClosure(t *testing.T) {
	const funcIR = `define ptr @"example.com/p.makeFn"() {
_llgo_0:
  ret ptr @"example.com/p.makeFn$1"
}

define void @"example.com/p.makeFn$1"() {
_llgo_0:
  ret void
}
`
	const initIR = `@0 = private unnamed_addr constant [3 x i8] c"fn"

define void @"example.com/p.init"() {
_llgo_0:
  store ptr @"example.com/p.init$1", ptr null
  ret void
}

define void @"example.com/p.init$1"() {
_llgo_0:
  ret void
}
`
	tests := []struct {
		name      string
		src       string
		ir        string
		adjacency string
	}{
		{
			name: "function declaration",
			src: `// LITTEST
package main

//go:noinline
func makeFn() func() { return func() {} }
`,
			ir:        funcIR,
			adjacency: "//go:noinline\nfunc makeFn",
		},
		{
			name: "variable declaration",
			src: `// LITTEST
package main

//llgo:tls
var fn = func() {}
`,
			ir:        initIR,
			adjacency: "//llgo:tls\nvar fn",
		},
		{
			name: "variable specification",
			src: `// LITTEST
package main

var (
	//llgo:tls
	fn = func() {}
)
`,
			ir:        initIR,
			adjacency: "\t//llgo:tls\n\tfn =",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := rewriteSource(test.src, "in.go", "example.com/p", "example.com", test.ir, testGenerationConfig)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, test.adjacency) {
				t.Fatalf("declaration directive separated from declaration:\n%s", got)
			}
			if !strings.Contains(got, "$1") {
				t.Fatalf("closure checks not inserted:\n%s", got)
			}
		})
	}
}

func TestRewriteSource_SharesInitClosureCountsAcrossDecls(t *testing.T) {
	const src = `// LITTEST
package main

var a = func() int { return 1 }()
var b = func() int { return 2 }()
`
	const ir = `define void @"example.com/p.init"() {
_llgo_0:
  %0 = call i64 @"example.com/p.init$1"()
  %1 = call i64 @"example.com/p.init$2"()
  ret void
}

define i64 @"example.com/p.init$1"() {
_llgo_0:
  ret i64 1
}

define i64 @"example.com/p.init$2"() {
_llgo_0:
  ret i64 2
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, testGenerationConfig)
	if err != nil {
		t.Fatal(err)
	}
	firstCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.init$1"(){{.*}} {`
	secondCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.init$2"(){{.*}} {`
	firstVar := "var a = func() int { return 1 }()"
	secondVar := "var b = func() int { return 2 }()"
	if strings.Index(got, firstCheck) > strings.Index(got, firstVar) {
		t.Fatalf("first init closure should be anchored before first var decl:\n%s", got)
	}
	if strings.Index(got, secondCheck) > strings.Index(got, secondVar) {
		t.Fatalf("second init closure should be anchored before second var decl:\n%s", got)
	}
}

func TestGeneralizeDefineLine_WildcardsAttrsBeforeBrace(t *testing.T) {
	line := `define void @"example.com/p.main"() local_unnamed_addr #0 {`
	got := generalizeDefineLine(line, "example.com")
	want := `define void @"{{.*}}/p.main"(){{.*}} {`
	if got != want {
		t.Fatalf("generalizeDefineLine = %q, want %q", got, want)
	}
}

func TestGeneralizeClosureEnvAttrs(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{
			`define void @"example.com/nest.swiftself"(ptr swiftself %env) {`,
			`define void @"example.com/nest.swiftself"(ptr {{(nest|swiftself)}} %env) {`,
		},
		{
			`  call void %fn(ptr nest %env, ptr %arg)`,
			`  call void %fn(ptr {{(nest|swiftself)}} %env, ptr %arg)`,
		},
		{
			`@0 = private constant [14 x i8] c"nest swiftself"`,
			`@0 = private constant [14 x i8] c"nest swiftself"`,
		},
		{
			`@nest = global ptr @swiftself`,
			`@nest = global ptr @swiftself`,
		},
	}
	for _, test := range tests {
		if got := generalizeClosureEnvAttrs(test.line); got != test.want {
			t.Errorf("generalizeClosureEnvAttrs(%q) = %q, want %q", test.line, got, test.want)
		}
	}
}

func TestScrubIRLineGeneralizesCgoSymbolHash(t *testing.T) {
	line := `  %0 = load ptr, ptr @main._cgo_96608f8de8c8_Cfunc__Cmalloc, align 8`
	got := scrubIRLine(line)
	want := `  %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc__Cmalloc, align 8`
	if got != want {
		t.Fatalf("scrubIRLine = %q, want %q", got, want)
	}
}

func TestScrubIRLineEscapesFileCheckVariableSyntax(t *testing.T) {
	line := `  call void @"main.(*Slice[[]int,int]).Append"()`
	got := scrubIRLine(line)
	want := `  call void @"main.(*Slice{{\[\[}}]int,int]).Append"()`
	if got != want {
		t.Fatalf("scrubIRLine = %q, want %q", got, want)
	}
}

func TestGeneralizeModulePath_ReplacesOnlyQuotedSegments(t *testing.T) {
	line := `  %0 = getelementptr inbounds %"go/example.Type", ptr @"go/example.fn"`
	got := generalizeModulePath(line, "go")
	want := `  %0 = getelementptr inbounds %"{{.*}}/example.Type", ptr @"{{.*}}/example.fn"`
	if got != want {
		t.Fatalf("generalizeModulePath = %q, want %q", got, want)
	}
}

func TestGeneralizeModulePath_IgnoresEscapedQuotes(t *testing.T) {
	line := "  !0 = !{!\"prefix \\\"quoted\\\" suffix\", !\"go/example.fn\"}"
	got := generalizeModulePath(line, "go")
	want := "  !0 = !{!\"prefix \\\"quoted\\\" suffix\", !\"{{.*}}/example.fn\"}"
	if got != want {
		t.Fatalf("generalizeModulePath = %q, want %q", got, want)
	}
}

func TestGenerationConfigForSourceProtectsManualChecks(t *testing.T) {
	_, autogenerated, err := generationConfigForSource("// LITTEST\npackage main\n", commandOptions{globals: "smart"})
	if autogenerated {
		t.Fatal("manual source reported as autogenerated")
	}
	if err == nil || !strings.Contains(err.Error(), "refusing to replace manual CHECK lines") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerationConfigForSourceUsesEmbeddedArguments(t *testing.T) {
	src := `// LITTEST
// NOTE: Assertions have been autogenerated by chore/litgen UTC_ARGS: --function=run(?:\$[0-9]+)? --check-globals=none
package main
`
	cfg, autogenerated, err := generationConfigForSource(src, commandOptions{globals: "smart"})
	if err != nil {
		t.Fatal(err)
	}
	if !autogenerated {
		t.Fatal("autogenerated note was not detected")
	}
	if cfg.allFunctions || len(cfg.functions) != 1 || cfg.functions[0] != `run(?:\$[0-9]+)?` {
		t.Fatalf("unexpected function config: %+v", cfg)
	}
	if cfg.checkGlobals != "none" {
		t.Fatalf("checkGlobals = %q, want none", cfg.checkGlobals)
	}
	if !cfg.matchesFunction("main.run$1", "example.com/p") || cfg.matchesFunction("main.main", "example.com/p") {
		t.Fatalf("embedded function filter does not select only run closures")
	}
}

func TestSetAutogeneratedNoteIsCanonicalAndIdempotent(t *testing.T) {
	cfg := generationConfig{functions: []string{"run", `run\$1`}, checkGlobals: "smart"}
	src := "// LITTEST\n// NOTE: Assertions have been autogenerated by chore/litgen UTC_ARGS: --all-functions --check-globals=all\npackage main\n"
	wantNote := "// NOTE: Assertions have been autogenerated by chore/litgen UTC_ARGS: --function=run --function=run\\$1 --check-globals=smart"
	got := setAutogeneratedNote(src, cfg)
	if strings.Count(got, autogeneratedNote) != 1 || !strings.Contains(got, wantNote) {
		t.Fatalf("unexpected autogenerated note:\n%s", got)
	}
	if again := setAutogeneratedNote(got, cfg); again != got {
		t.Fatalf("setAutogeneratedNote is not idempotent:\n--- first ---\n%s--- second ---\n%s", got, again)
	}
}

func TestRewriteSourceFiltersFunctions(t *testing.T) {
	const src = `// LITTEST
package main

func keep() {}
func drop() {}
`
	const ir = `define void @main.keep() {
_llgo_0:
  ret void
}
define void @main.drop() {
_llgo_0:
  ret void
}
`
	cfg := generationConfig{functions: []string{"keep"}, checkGlobals: "none"}
	if err := cfg.compile(); err != nil {
		t.Fatal(err)
	}
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "@main.keep") || strings.Contains(got, "@main.drop") {
		t.Fatalf("function filter mismatch:\n%s", got)
	}
}

func TestRewriteSourceSmartGlobalsFollowSelectedFunctions(t *testing.T) {
	const src = `// LITTEST
package main

func keep() {}
func drop() {}
`
	const ir = `@0 = private unnamed_addr constant [4 x i8] c"keep"
@1 = private unnamed_addr constant [4 x i8] c"drop"

define void @main.keep() {
_llgo_0:
  call void @print(ptr @0)
  ret void
}

define void @main.drop() {
_llgo_0:
  call void @print(ptr @1)
  ret void
}
`
	cfg := generationConfig{functions: []string{"keep"}, checkGlobals: "smart"}
	if err := cfg.compile(); err != nil {
		t.Fatal(err)
	}
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `c"keep"`) {
		t.Fatalf("selected function global is missing:\n%s", got)
	}
	if strings.Contains(got, `c"drop"`) {
		t.Fatalf("unselected function global leaked into smart checks:\n%s", got)
	}
}

func TestIRGeneralizerAbstractsNumberingButNotStrings(t *testing.T) {
	g := newIRGeneralizer()
	g.startFunction()
	lines := []string{
		`define void @main.f(i64 %0) {`,
		`_llgo_0:`,
		`  %1 = call ptr @use(i64 %0, ptr @12)`,
		`  call void @print(ptr @12) ; text "%0 @12 _llgo_0"`,
		`  br label %_llgo_0`,
	}
	var got []string
	for _, line := range lines {
		got = append(got, g.generalizeFunctionLine(line))
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{
		`%[[TMP0:[0-9]+]]`, `%[[TMP1:[0-9]+]]`, `%[[TMP0]]`,
		`@[[GLOB12:[0-9]+]]`, `@[[GLOB12]]`,
		`_llgo_[[BB0:[0-9]+]]`, `%_llgo_[[BB0]]`,
		`"%0 @12 _llgo_0"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("generalized output missing %q:\n%s", want, joined)
		}
	}
}
