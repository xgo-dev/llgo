package cl

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

type representationImports map[string]*types.Package

func (p representationImports) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	return p[path], nil
}

func checkRepresentationPackage(t *testing.T, prog llssa.Program, path, source string, imports representationImports, liveInfo bool) (*types.Package, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path+"/p.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage(path, file.Name.Name)
	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	files := []*ast.File{file}
	var live *types.Info
	if liveInfo {
		live = info
	}
	if err := ParsePkgSyntaxWithOptions(prog, fset, pkg, files, Options{}, live); err != nil {
		t.Fatal(err)
	}
	config := &types.Config{Importer: imports, Sizes: prog.TypeSizes(types.SizesFor("gc", "amd64"))}
	if err := types.NewChecker(config, fset, pkg, info).Files(files); err != nil {
		t.Fatal(err)
	}
	return pkg, info
}

func TestNativeRepresentationFollowsBoundDefinitions(t *testing.T) {
	for _, liveInfo := range []bool{true, false} {
		name := "completed-scope"
		if liveInfo {
			name = "checker-info"
		}
		t.Run(name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg, info := checkRepresentationPackage(t, prog, "example.com/native", `package native
import "unsafe"
//llgo:cdecl
type Native struct { Fn func(); Tail uint64 }
func (Native) Method() {}
type First Native
type Copy First
type Parenthesized (Copy)
type Alias = Native
type ThroughAlias Alias
type Independent struct { Fn func(); Tail uint64 }
//llgo:cdecl
type Generic[T any] struct { Fn func(); Tail T }
type GenericCopy[T any] Generic[T]
type Instance Generic[uint64]
type ThroughTemplate GenericCopy[uint64]
//llgo:cdecl
type Recursive struct { Fn func(); Tail uint64; Next *Recursive }
type RecursiveCopy Recursive
//llgo:cdecl
type Callback func()
type CallbackCopy Callback
const CopySize = unsafe.Sizeof(Copy{})
func local() {
 type Native struct { Fn func(); Tail uint64 }
 type LocalCopy Copy
 var _ Native
 var _ LocalCopy
}
`, nil, liveInfo)
			native := pkg.Scope().Lookup("Native").Type()
			for _, name := range []string{"First", "Copy", "Parenthesized", "Alias", "ThroughAlias", "Instance", "ThroughTemplate"} {
				typ := pkg.Scope().Lookup(name).Type()
				if rep := prog.RepresentationOf(typ); rep.Storage != llssa.NativeRepresentation || rep.Origin == nil {
					t.Fatalf("%s representation = %+v", name, rep)
				}
				raw := prog.Type(typ, llssa.GoDecl)
				if size, offset := prog.SizeOf(raw), prog.OffsetOf(raw, 1); size != 16 || offset != 8 {
					t.Fatalf("%s physical size/offset = %d/%d, want 16/8", name, size, offset)
				}
			}
			copyType := pkg.Scope().Lookup("Copy").Type()
			if native.Underlying() != copyType.Underlying() {
				t.Fatal("probe no longer exercises shared underlying objects")
			}
			if types.NewMethodSet(native).Len() != 1 || types.NewMethodSet(copyType).Len() != 0 {
				t.Fatal("representation inheritance changed Go method sets")
			}
			if liveInfo {
				size, _ := constant.Int64Val(pkg.Scope().Lookup("CopySize").(*types.Const).Val())
				if size != 16 {
					t.Fatalf("source CopySize = %d, want 16", size)
				}
			}
			for id, obj := range info.Defs {
				typeObj, ok := obj.(*types.TypeName)
				if !ok {
					continue
				}
				local := typeObj.Parent() != pkg.Scope()
				if id.Name == "Independent" || id.Name == "Native" && local {
					if got := prog.SizeOf(prog.Type(obj.Type(), llssa.GoDecl)); got != 24 {
						t.Fatalf("ordinary %s has size %d, want 24", id.Name, got)
					}
				}
				if id.Name == "LocalCopy" && prog.RepresentationOf(obj.Type()).Storage != llssa.NativeRepresentation {
					t.Fatal("local derived type lost native representation")
				}
			}
			for _, names := range [][2]string{{"Recursive", "RecursiveCopy"}, {"Callback", "CallbackCopy"}} {
				a := prog.Type(pkg.Scope().Lookup(names[0]).Type(), llssa.GoDecl)
				b := prog.Type(pkg.Scope().Lookup(names[1]).Type(), llssa.GoDecl)
				if prog.SizeOf(a) != prog.SizeOf(b) {
					t.Fatalf("%s changed inherited storage", names[1])
				}
			}

			consumer, _ := checkRepresentationPackage(t, prog, "example.com/consumer", `package consumer
import renamed "example.com/native"
type Copy renamed.Copy
type Again Copy
type Instance renamed.Generic[uint64]
`, representationImports{"example.com/native": pkg}, liveInfo)
			backend := prog.NewBackendProgram()
			defer backend.Dispose()
			for _, name := range []string{"Copy", "Again", "Instance"} {
				typ := backend.Type(consumer.Scope().Lookup(name).Type(), llssa.GoDecl)
				if backend.SizeOf(typ) != 16 || backend.OffsetOf(typ, 1) != 8 {
					t.Fatalf("cross-package %s lost native representation in backend", name)
				}
			}
			var workers sync.WaitGroup
			for range 4 {
				workers.Go(func() {
					for range 50 {
						if prog.RepresentationOf(copyType).Origin != native.(*types.Named).Obj() {
							t.Error("concurrent query lost representation origin")
						}
					}
				})
			}
			workers.Wait()
		})
	}
}
