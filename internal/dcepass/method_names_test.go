package dcepass

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRefinedMethodNamesFromModules(t *testing.T) {
	ir := `
declare void @reflect_call(ptr, i64)

define void @known() {
entry:
  call void @reflect_call(ptr null, i64 0) #0
  call void @reflect_call(ptr null, i64 0) #1
  ret void
}

define void @partly_unknown() {
entry:
  call void @reflect_call(ptr null, i64 0) #0
  call void @reflect_call(ptr null, i64 0) #2
  ret void
}

define void @unmarked() {
entry:
  ret void
}

attributes #0 = { "llgo.reflect.methodbyname"="value" "llgo.reflect.methodbyname.names"="KeepB,KeepA" }
attributes #1 = { "llgo.reflect.methodbyname"="value" "llgo.reflect.methodbyname.names"="KeepC,KeepA" }
attributes #2 = { "llgo.reflect.methodbyname"="value" }
`
	path := filepath.Join(t.TempDir(), "names.ll")
	if err := os.WriteFile(path, []byte(ir), 0o644); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Dispose()

	got := RefinedMethodNamesFromModules([]llvm.Module{mod}, []string{
		"known", "partly_unknown", "unmarked", "missing",
	})
	want := map[string][]string{"known": {"KeepA", "KeepB", "KeepC"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RefinedMethodNamesFromModules() = %#v, want %#v", got, want)
	}
}
