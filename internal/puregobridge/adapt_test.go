package puregobridge

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAdaptRejectsUnqualifiedProfiles(t *testing.T) {
	for _, test := range []struct {
		path, version, os, arch string
		cgo                     bool
	}{
		{"example.com/purego", "v0.9.1", "linux", "amd64", true},
		{"github.com/ebitengine/purego", "v0.9.0", "linux", "amd64", true},
		{"github.com/ebitengine/purego", "v0.9.2", "linux", "amd64", true},
		{"github.com/ebitengine/purego", "v0.9.1", "darwin", "amd64", true},
		{"github.com/ebitengine/purego", "v0.9.1", "linux", "arm64", true},
		{"github.com/ebitengine/purego", "v0.9.1", "linux", "amd64", false},
	} {
		if changes, err := Adapt(test.path, test.version, test.os, test.arch, test.cgo, nil); err == nil || changes != nil {
			t.Fatalf("accepted unqualified profile %+v", test)
		}
	}
	if changes, err := Adapt("github.com/ebitengine/purego", "v0.9.1", "linux", "amd64", true, nil); err == nil || changes != nil {
		t.Fatal("accepted absent module source")
	}
}

func TestVerifySourceRejectsPartialOrChangedSource(t *testing.T) {
	data := []byte("package checked\n")
	want := map[string]string{"checked.go": fmt.Sprintf("%x", sha256.Sum256(data))}
	if err := verifySource(fstest.MapFS{"checked.go": {Data: data}}, want); err != nil {
		t.Fatal(err)
	}
	for _, source := range []fstest.MapFS{
		{},
		{"checked.go": {Data: []byte("package changed\n")}},
	} {
		if err := verifySource(source, want); err == nil {
			t.Fatal("accepted absent or changed source")
		}
	}
}

func TestBackendIsExplicitAndWiresNativeSource(t *testing.T) {
	data, err := assets.ReadFile("backend/syscall_sysv.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "syscall_sysv.go", data, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name.Name != "purego" {
		t.Fatal("wrong adapter package")
	}
	functions := map[string]bool{}
	constants := map[string]string{}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			functions[decl.Name.Name] = decl.Body != nil
		case *ast.GenDecl:
			if decl.Tok != token.CONST {
				continue
			}
			for _, spec := range decl.Specs {
				value := spec.(*ast.ValueSpec)
				if len(value.Names) == 1 && len(value.Values) == 1 {
					if lit, ok := value.Values[0].(*ast.BasicLit); ok {
						constants[value.Names[0].Name] = lit.Value
					}
				}
			}
		}
	}
	for _, name := range []string{"runtime_cgocall", "syscall_syscall15X", "NewCallback"} {
		if !functions[name] {
			t.Fatalf("adapter lacks implementation of %s", name)
		}
	}
	if constants["LLGoPackage"] != `"llgo"` || constants["LLGoFiles"] != `"_llgo_compat/sysv_amd64.S"` {
		t.Fatal("native source must retain Go package initialization, including dlfcn function values and the argument pool")
	}
	if strings.Contains(string(data), "runtime.cgocall") || strings.Contains(string(data), "crosscall2") {
		t.Fatal("explicit adapter must not alias an upstream Go runtime entry")
	}
}

func TestAdaptRejectsPartialAndChangedQualifiedSource(t *testing.T) {
	for _, source := range []fstest.MapFS{
		{},
		{"cgo.go": {Data: []byte("package purego\n")}},
	} {
		changes, err := Adapt("github.com/ebitengine/purego", "v0.9.1", "linux", "amd64", true, source)
		if err == nil || changes != nil || !strings.Contains(err.Error(), "cgo.go") {
			t.Fatalf("unverified source produced changes: %+v, %v", changes, err)
		}
	}
}
