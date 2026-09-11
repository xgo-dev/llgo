package types_test

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"
)

func TestInfoPopulation(t *testing.T) {
	src := `package sample

import "example.org/fixture/fmt"

type MyType struct{ X int }

func (m *MyType) String() string { return fmt.Sprintf("%d", m.X) }

func Use[T ~int](v T) T {
    return v
}

func Run() {
    var m MyType
    fmt.Println(m.String(), Use[int](42))
}`

	info := &types.Info{}
	checked := typeCheck(t, src, info)

	if obj := checked.pkg.Scope().Lookup("MyType"); obj == nil {
		t.Fatal("expected MyType in package scope")
	}
	if obj := checked.pkg.Scope().Lookup("Run"); obj == nil {
		t.Fatal("expected Run in package scope")
	}

	var typeObj *types.TypeName
	var methodObj *types.Func
	var typeParamObj *types.TypeName
	for ident, obj := range info.Defs {
		if ident == nil || obj == nil {
			continue
		}
		switch obj := obj.(type) {
		case *types.TypeName:
			switch ident.Name {
			case "MyType":
				typeObj = obj
			case "T":
				if ident.Obj != nil && ident.Obj.Kind == ast.Typ {
					typeParamObj = obj
				}
			}
		case *types.Func:
			if ident.Name == "String" {
				methodObj = obj
			}
		}
	}
	if typeObj == nil {
		t.Fatal("expected MyType definition in info.Defs")
	}
	if methodObj == nil {
		t.Fatal("expected String method definition in info.Defs")
	}
	if typeParamObj == nil {
		t.Fatal("expected type parameter T definition in info.Defs")
	}

	var pkgName *types.PkgName
	for ident, obj := range info.Uses {
		if ident != nil && ident.Name == "fmt" {
			if pn, ok := obj.(*types.PkgName); ok {
				pkgName = pn
				break
			}
		}
	}
	if pkgName == nil {
		t.Fatal("expected fmt package in info.Uses")
	}

	var methodSel *types.Selection
	for _, sel := range info.Selections {
		if sel.Kind() == types.MethodVal && sel.Obj() == methodObj {
			methodSel = sel
			break
		}
	}
	if methodSel == nil {
		t.Fatal("expected method selection for String")
	}
	if recv := methodSel.Recv().String(); !strings.Contains(recv, "MyType") {
		t.Fatalf("unexpected receiver for String: %s", recv)
	}

	var instCall types.TypeAndValue
	for expr, tv := range info.Types {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}
		index, ok := call.Fun.(*ast.IndexExpr)
		if !ok {
			continue
		}
		ident, ok := index.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "Use" {
			instCall = tv
			break
		}
	}
	if instCall.Type == nil {
		t.Fatal("expected type information for Use[int](42)")
	}
	if got, want := instCall.Type.String(), "int"; got != want {
		t.Fatalf("call result type = %s, want %s", got, want)
	}

	var inst types.Instance
	foundInstance := false
	for ident, instance := range info.Instances {
		if ident.Name == "Use" {
			inst = instance
			foundInstance = true
			break
		}
	}
	if !foundInstance {
		t.Fatal("expected generic instantiation for Use")
	}
	if got, want := inst.Type.String(), "func(v int) int"; got != want {
		t.Fatalf("instance type = %s, want %s", got, want)
	}

	var foundLocalScope bool
	for _, scope := range info.Scopes {
		if scope.Lookup("m") != nil {
			foundLocalScope = true
			break
		}
	}
	if !foundLocalScope {
		t.Fatal("expected a scope containing variable m")
	}

	var implicitPkg types.Object
	for node, obj := range info.Implicits {
		if spec, ok := node.(*ast.ImportSpec); ok && spec.Path != nil && strings.Contains(spec.Path.Value, "fmt") {
			implicitPkg = obj
			break
		}
	}
	if implicitPkg == nil {
		t.Fatal("expected implicit pkg object for fmt import")
	}
}
