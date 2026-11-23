package cl

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
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
	if err := CompleteTypeRepresentations(prog, fset, pkg, files, live); err != nil {
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
const CopySize = unsafe.Sizeof(Copy{})
type Bound [CopySize]byte
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
				if size != 16 || pkg.Scope().Lookup("Bound").Type().Underlying().(*types.Array).Len() != 16 {
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

func TestNativeRepresentationRejectsLostScope(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", "package p; type Copy Native", 0)
	if err != nil {
		t.Fatal(err)
	}
	spec := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	// A one-shot importer retained checked declarations, but lost their scope.
	// An unresolved RHS must never silently select Go storage for Copy.
	pkg := types.NewPackage("example.com/lostscope", "p")
	st := types.NewStruct(nil, nil)
	copyType := types.NewNamed(types.NewTypeName(spec.Name.Pos(), pkg, "Copy", nil), st, nil)
	pkg.Scope().Insert(copyType.Obj())
	pkg.MarkComplete()
	prog.SetTypeCallConv(pkg.Path()+".Native", llssa.CDecl)
	err = registerTypeRepresentations(prog, fset, pkg, []*ast.File{file}, nil)
	if err == nil || !strings.Contains(err.Error(), "resolve representation of") || !strings.Contains(err.Error(), "no position") {
		t.Fatalf("lost scope must fail before publication, got %v", err)
	}
	// A failed query must not poison later queries after scope restoration.
	types.NewScope(pkg.Scope(), file.Pos(), file.End(), "restored file scope")
	native := types.NewNamed(types.NewTypeName(0, pkg, "Native", nil), st, nil)
	pkg.Scope().Insert(native.Obj())
	if err := CompleteTypeRepresentations(prog, fset, pkg, []*ast.File{file}, nil); err != nil {
		t.Fatal(err)
	}
	if got := prog.RepresentationOf(copyType); got.Storage != llssa.NativeRepresentation || got.Origin != native.Obj() {
		t.Fatalf("restored scope did not restore native provenance: %+v", got)
	}
}

func TestNativeRepresentationSnapshotDropsCheckerBindings(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, info := checkRepresentationPackage(t, prog, "example.com/frozen", `package frozen
//llgo:cdecl
type Native struct { Fn func(); Tail uint64 }
type Copy Native
func local() { type Local Copy; var _ Local }
`, nil, true)
	var local types.Type
	for id, obj := range info.Defs {
		if id.Name == "Local" {
			local = obj.Type()
		}
	}
	copyType := pkg.Scope().Lookup("Copy").Type()
	native := pkg.Scope().Lookup("Native").(*types.TypeName)
	// Replacing both maps catches any source callback retained by the backend.
	info.Types = nil
	info.Defs = nil
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	localName := local.(*types.Named).Obj()
	instantiatedLocal := types.NewNamed(types.NewTypeName(localName.Pos(), pkg, localName.Name(), nil), local.Underlying(), nil)
	for _, typ := range []types.Type{copyType, local, instantiatedLocal} {
		if backend.RepresentationOf(typ).Origin != native {
			t.Fatal("snapshot lost the origin after discarding source bindings")
		}
		raw := backend.Type(typ, llssa.GoDecl)
		if backend.SizeOf(raw) != 16 || backend.OffsetOf(raw, 1) != 8 {
			t.Fatal("backend layout still depends on source bindings")
		}
	}
}

func TestNativeRepresentationCompletionRejectsMissingBindings(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", `package p
//llgo:cdecl
type Native struct { F func() }
type Copy Native`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	bindings := info.Types
	info.Types = nil
	if err := ParsePkgSyntaxWithOptions(prog, fset, pkg, []*ast.File{file}, Options{}, info); err == nil || !strings.Contains(err.Error(), "missing RHS binding") {
		t.Fatalf("incomplete snapshot accepted: %v", err)
	}
	if !prog.TypeRepresentationsPending(pkg) {
		t.Fatal("failed completion published a partial snapshot")
	}
	info.Types = bindings
	if err := CompleteTypeRepresentations(prog, fset, pkg, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	if prog.RepresentationOf(pkg.Scope().Lookup("Copy").Type()).Storage != llssa.NativeRepresentation {
		t.Fatal("failed completion poisoned the retry")
	}
}
