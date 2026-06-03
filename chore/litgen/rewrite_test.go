package main

import (
	"strings"
	"testing"
)

func TestRewriteSource_InsertsMainClosureAndStub(t *testing.T) {
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

define linkonce void @"__llgo_stub.example.com/p.main$1"(ptr %0) {
_llgo_0:
  tail call void @"example.com/p.main$1"()
  ret void
}
`
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir)
	if err != nil {
		t.Fatal(err)
	}
	mainCheck := `// CHECK-LABEL: define void @"{{.*}}/p.main"() {`
	mainDecl := "func main() {"
	if !strings.Contains(got, mainCheck) {
		t.Fatalf("main checks not inserted before func main:\n%s", got)
	}
	if strings.Index(got, mainCheck) > strings.Index(got, mainDecl) {
		t.Fatalf("main checks should appear before func main:\n%s", got)
	}
	closureCheck := "\t// CHECK-LABEL: define void @\"{{.*}}/p.main$1\"() {"
	closureStmt := "\tfn := func() {}"
	if !strings.Contains(got, closureCheck) {
		t.Fatalf("closure checks not inserted before func literal:\n%s", got)
	}
	if strings.Index(got, closureCheck) > strings.Index(got, closureStmt) {
		t.Fatalf("closure checks should appear before func literal:\n%s", got)
	}
	if !strings.Contains(got, `// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/p.main$1"(ptr %0) {`) {
		t.Fatalf("stub checks missing:\n%s", got)
	}
	if strings.Index(got, `// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/p.main$1"(ptr %0) {`) < strings.Index(got, "func main()") {
		t.Fatalf("stub checks should be appended after source:\n%s", got)
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
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir)
	if err != nil {
		t.Fatal(err)
	}
	initCheck := `// CHECK-LABEL: define void @"{{.*}}/p.init"() {`
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
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir)
	if err != nil {
		t.Fatal(err)
	}
	addCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.add"(i64 %0, i64 %1) {`
	initCheck := `// CHECK-LABEL: define void @"{{.*}}/p.init"() {`
	if strings.Index(got, addCheck) < 0 || strings.Index(got, initCheck) < 0 {
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
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1`) {
		t.Fatalf("missing numeric global @0:\n%s", got)
	}
	if !strings.Contains(got, `// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"%s\00", align 1`) {
		t.Fatalf("missing numeric global @1:\n%s", got)
	}
	if strings.Contains(got, `// CHECK-LINE: @"{{.*}}/p.named" = global i64 1`) {
		t.Fatalf("named globals should not be emitted by default:\n%s", got)
	}
	if strings.Index(got, `// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1`) > strings.Index(got, "func main()") {
		t.Fatalf("global checks should be placed before first declaration:\n%s", got)
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
	got, err := rewriteSource(src, "in.go", "example.com/p", "example.com", ir)
	if err != nil {
		t.Fatal(err)
	}
	firstCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.init$1"() {`
	secondCheck := `// CHECK-LABEL: define i64 @"{{.*}}/p.init$2"() {`
	firstVar := "var a = func() int { return 1 }()"
	secondVar := "var b = func() int { return 2 }()"
	if strings.Index(got, firstCheck) > strings.Index(got, firstVar) {
		t.Fatalf("first init closure should be anchored before first var decl:\n%s", got)
	}
	if strings.Index(got, secondCheck) > strings.Index(got, secondVar) {
		t.Fatalf("second init closure should be anchored before second var decl:\n%s", got)
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
