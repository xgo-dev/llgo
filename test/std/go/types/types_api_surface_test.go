package types_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"iter"
	"testing"
)

func countSeq[T any](seq iter.Seq[T]) int {
	n := 0
	for range seq {
		n++
	}
	return n
}

func TestObjectsAndBasicTypesAPI(t *testing.T) {
	src := `package p
import f "example.org/fixture/fmt"
type A = int
type N int
type S struct{ N }
type I interface{ M() int }
func (S) M() int { return 1 }
const C = 7
var V = C
func F(x int) int { return x }
func G[T ~int](v T) T { return v }
var _ = f.Sprintf("%d", C)
`
	info := &gotypes.Info{}
	checked := typeCheck(t, src, info)
	pkg := checked.pkg

	// Package methods.
	if pkg.String() == "" {
		t.Fatal("Package.String returned empty")
	}
	_ = pkg.GoVersion()
	pkg.SetName("p")
	pkg.MarkComplete()
	if !pkg.Complete() {
		t.Fatal("Package.Complete should be true after MarkComplete")
	}

	// Universe builtins and nil object.
	builtin, ok := gotypes.Universe.Lookup("len").(*gotypes.Builtin)
	if !ok {
		t.Fatal("Universe len is not *types.Builtin")
	}
	if builtin.Name() == "" || builtin.String() == "" || builtin.Id() == "" {
		t.Fatal("Builtin basic object methods returned empty")
	}
	_ = builtin.Exported()
	_ = builtin.Parent()
	_ = builtin.Pkg()
	_ = builtin.Pos()
	if builtin.Type() == nil {
		t.Fatal("Builtin.Type returned nil")
	}

	nilObj, ok := gotypes.Universe.Lookup("nil").(*gotypes.Nil)
	if !ok {
		t.Fatal("Universe nil is not *types.Nil")
	}
	if nilObj.Name() != "nil" || nilObj.String() == "" || nilObj.Type() == nil {
		t.Fatal("Nil object methods returned invalid values")
	}
	_ = nilObj.Exported()
	_ = nilObj.Id()
	_ = nilObj.Parent()
	_ = nilObj.Pkg()
	_ = nilObj.Pos()

	// PkgName via imported alias.
	var pkgName *gotypes.PkgName
	for ident, obj := range info.Defs {
		if ident != nil && ident.Name == "f" {
			if pn, ok := obj.(*gotypes.PkgName); ok {
				pkgName = pn
				break
			}
		}
	}
	if pkgName == nil {
		t.Fatal("PkgName for import alias not found")
	}
	if pkgName.Name() != "f" || pkgName.Imported() == nil {
		t.Fatal("PkgName methods returned invalid values")
	}
	_ = pkgName.Exported()
	_ = pkgName.Id()
	_ = pkgName.Parent()
	_ = pkgName.Pkg()
	_ = pkgName.Pos()
	_ = pkgName.String()
	_ = pkgName.Type()

	// Const / Func / TypeName / Var object methods.
	cObj := pkg.Scope().Lookup("C").(*gotypes.Const)
	if cObj.Val() == nil || cObj.Type() == nil || cObj.Name() != "C" {
		t.Fatal("Const object methods returned invalid values")
	}
	_ = cObj.Exported()
	_ = cObj.Id()
	_ = cObj.Parent()
	_ = cObj.Pkg()
	_ = cObj.Pos()
	_ = cObj.String()

	fObj := pkg.Scope().Lookup("F").(*gotypes.Func)
	if fObj.Signature() == nil || fObj.FullName() == "" || fObj.Scope() == nil {
		t.Fatal("Func object methods returned invalid values")
	}
	_ = fObj.Exported()
	_ = fObj.Id()
	_ = fObj.Origin()
	_ = fObj.Parent()
	_ = fObj.Pkg()
	_ = fObj.Pos()
	_ = fObj.String()
	_ = fObj.Type()

	tName := pkg.Scope().Lookup("N").(*gotypes.TypeName)
	if tName.Name() == "" || tName.Type() == nil || tName.String() == "" {
		t.Fatal("TypeName object methods returned invalid values")
	}
	_ = tName.Exported()
	_ = tName.Id()
	_ = tName.IsAlias()
	_ = tName.Parent()
	_ = tName.Pkg()
	_ = tName.Pos()

	named := tName.Type().(*gotypes.Named)
	if named.Obj() != tName || named.Underlying() == nil {
		t.Fatal("Named type methods returned invalid values")
	}
	_ = named.String()
	_ = named.TypeArgs()
	_ = named.TypeParams()
	_ = named.Origin()
	if named.NumMethods() > 0 {
		_ = named.Method(0)
	}
	_ = named.Methods()
	_ = named.NumMethods()

	fieldObj := pkg.Scope().Lookup("S").Type().Underlying().(*gotypes.Struct).Field(0)
	if !fieldObj.Anonymous() || !fieldObj.Embedded() || !fieldObj.IsField() {
		t.Fatal("embedded field var methods returned invalid values")
	}
	_ = fieldObj.Exported()
	_ = fieldObj.Id()
	_ = fieldObj.Origin()
	_ = fieldObj.Parent()
	_ = fieldObj.Pkg()
	_ = fieldObj.Pos()
	_ = fieldObj.String()
	_ = fieldObj.Type()

	// Label object.
	lbl := gotypes.NewLabel(token.NoPos, pkg, "L")
	if lbl.Name() != "L" || lbl.String() == "" {
		t.Fatal("Label methods returned invalid values")
	}
	_ = lbl.Exported()
	_ = lbl.Id()
	_ = lbl.Parent()
	_ = lbl.Pkg()
	_ = lbl.Pos()
	_ = lbl.Type()

	// Manual named type for AddMethod / SetTypeParams / SetUnderlying.
	tpConstraint := gotypes.NewInterfaceType(nil, nil)
	tpConstraint.Complete()
	tpObj := gotypes.NewTypeName(token.NoPos, pkg, "TP", nil)
	tp := gotypes.NewTypeParam(tpObj, tpConstraint)
	manualObj := gotypes.NewTypeName(token.NoPos, pkg, "Manual", nil)
	manualNamed := gotypes.NewNamed(manualObj, gotypes.Typ[gotypes.Invalid], nil)
	manualNamed.SetTypeParams([]*gotypes.TypeParam{tp})
	manualNamed.SetUnderlying(gotypes.NewStruct(nil, nil))
	recv := gotypes.NewVar(token.NoPos, pkg, "", manualNamed)
	meth := gotypes.NewFunc(token.NoPos, pkg, "Meth", gotypes.NewSignatureType(recv, nil, nil, nil, nil, false))
	manualNamed.AddMethod(meth)
	if manualNamed.Method(0) == nil || manualNamed.NumMethods() == 0 {
		t.Fatal("manual named type method set is empty")
	}

	// Basic/BasicKind/BasicInfo.
	basic := gotypes.Typ[gotypes.Int]
	if basic.Name() == "" || basic.String() == "" || basic.Kind() != gotypes.Int || basic.Info() == 0 || basic.Underlying() == nil {
		t.Fatal("Basic methods returned invalid values")
	}
	var _ gotypes.BasicKind = gotypes.Int
	var _ gotypes.BasicInfo = basic.Info()
}

func TestTypeRelationsScopeAndWriters(t *testing.T) {
	src := `package p
type I interface{ M() int }
type T struct{}
func (T) M() int { return 1 }
func H[T ~int|~string](v T) T { return v }
var _ I = T{}
var _ = T{}.M()
`
	info := &gotypes.Info{}
	checked := typeCheck(t, src, info)
	pkg := checked.pkg

	iType := pkg.Scope().Lookup("I").Type().Underlying().(*gotypes.Interface)
	tType := pkg.Scope().Lookup("T").Type()
	if !gotypes.Implements(tType, iType) || !gotypes.AssignableTo(tType, iType) {
		t.Fatal("T should implement/assign to I")
	}
	if !gotypes.AssertableTo(iType, tType) {
		t.Fatal("interface value should be assertable to T")
	}
	if !gotypes.Comparable(tType) {
		t.Fatal("T should be comparable")
	}
	if !gotypes.IsInterface(iType) {
		t.Fatal("IsInterface should be true for I")
	}
	_ = gotypes.Satisfies(tType, iType)
	_ = gotypes.Id(pkg, "X")
	_ = gotypes.Identical(gotypes.Typ[gotypes.Int], gotypes.Typ[gotypes.Int])
	_ = gotypes.IdenticalIgnoreTags(
		gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, nil, "X", gotypes.Typ[gotypes.Int], false)}, []string{`a:"1"`}),
		gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, nil, "X", gotypes.Typ[gotypes.Int], false)}, []string{`b:"2"`}),
	)

	// Scope API.
	root := gotypes.NewScope(nil, token.Pos(1), token.Pos(100), "root")
	child := gotypes.NewScope(root, token.Pos(10), token.Pos(20), "child")
	v := gotypes.NewVar(token.Pos(11), pkg, "x", gotypes.Typ[gotypes.Int])
	root.Insert(v)
	if !root.Contains(token.Pos(11)) || root.Lookup("x") == nil {
		t.Fatal("Scope basic lookup/contains failed")
	}
	if root.Child(0) != child || countSeq(root.Children()) != 1 || root.NumChildren() != 1 {
		t.Fatal("Scope child APIs returned invalid values")
	}
	if root.Innermost(token.Pos(12)) != child {
		t.Fatal("Scope.Innermost should return child scope")
	}
	if s, obj := root.LookupParent("x", token.Pos(15)); obj != v || s != root {
		t.Fatal("Scope.LookupParent failed")
	}
	if root.Len() != 1 || len(root.Names()) != 1 {
		t.Fatal("Scope Len/Names mismatch")
	}
	_ = root.Pos()
	_ = root.End()
	if root.String() == "" {
		t.Fatal("Scope.String returned empty")
	}
	var sb bytes.Buffer
	root.WriteTo(&sb, 0, true)
	if sb.Len() == 0 {
		t.Fatal("Scope.WriteTo wrote empty output")
	}

	// MethodSet/Selection APIs.
	ms := gotypes.NewMethodSet(tType)
	if ms.Len() == 0 || ms.At(0).Obj().Name() != "M" {
		t.Fatal("method set should contain M")
	}
	if ms.Lookup(pkg, "M") == nil || countSeq(ms.Methods()) == 0 || ms.String() == "" {
		t.Fatal("MethodSet lookup/methods/string failed")
	}
	var sel *gotypes.Selection
	for _, s := range info.Selections {
		sel = s
		break
	}
	if sel == nil {
		t.Fatal("expected at least one Selection")
	}
	if sel.Obj() == nil || sel.Recv() == nil || sel.Type() == nil {
		t.Fatal("Selection core methods returned nil")
	}
	_ = sel.Index()
	_ = sel.Indirect()
	_ = sel.Kind()
	_ = sel.String()
	_ = gotypes.SelectionString(sel, gotypes.RelativeTo(pkg))
	var _ gotypes.SelectionKind = sel.Kind()

	// WriteExpr / WriteType / WriteSignature / CheckExpr.
	expr, err := parser.ParseExpr("1 + 2")
	if err != nil {
		t.Fatalf("ParseExpr: %v", err)
	}
	var eb bytes.Buffer
	gotypes.WriteExpr(&eb, expr)
	if eb.Len() == 0 {
		t.Fatal("WriteExpr wrote empty output")
	}
	var tb bytes.Buffer
	gotypes.WriteType(&tb, tType, gotypes.RelativeTo(pkg))
	if tb.Len() == 0 {
		t.Fatal("WriteType wrote empty output")
	}
	var sbuf bytes.Buffer
	gotypes.WriteSignature(&sbuf, pkg.Scope().Lookup("H").Type().(*gotypes.Signature), gotypes.RelativeTo(pkg))
	if sbuf.Len() == 0 {
		t.Fatal("WriteSignature wrote empty output")
	}
	checkInfo := &gotypes.Info{Types: map[ast.Expr]gotypes.TypeAndValue{}}
	if err := gotypes.CheckExpr(checked.fset, pkg, token.NoPos, expr, checkInfo); err != nil {
		t.Fatalf("CheckExpr: %v", err)
	}
}

func TestGenericsAliasTypeCollectionsAndImporters(t *testing.T) {
	src := `package p
import _ "example.org/fixture/fmt"
type A = int
type Box[T any] struct{ V T }
func Make[T any](v T) Box[T] { return Box[T]{V: v} }
`
	info := &gotypes.Info{}
	checked := typeCheck(t, src, info)
	pkg := checked.pkg

	// Alias API.
	aliasObj := pkg.Scope().Lookup("A").Type().(*gotypes.Alias)
	if aliasObj.Obj() == nil || aliasObj.Origin() == nil || aliasObj.Rhs() == nil || aliasObj.Underlying() == nil || aliasObj.String() == "" {
		t.Fatal("Alias methods returned invalid values")
	}
	aliasObj.SetTypeParams(nil)
	_ = aliasObj.TypeArgs()
	_ = aliasObj.TypeParams()

	// Signature/TypeParam/TypeList/TypeParamList/Term/Union APIs.
	makeSig := pkg.Scope().Lookup("Make").Type().(*gotypes.Signature)
	if makeSig.Recv() != nil || makeSig.RecvTypeParams() != nil || makeSig.Underlying() == nil || makeSig.String() == "" {
		t.Fatal("Signature methods returned invalid values")
	}
	tparams := makeSig.TypeParams()
	if tparams.Len() != 1 || tparams.At(0) == nil || countSeq(tparams.TypeParams()) != 1 {
		t.Fatal("TypeParamList methods returned invalid values")
	}
	tp := tparams.At(0)
	if tp.Obj() == nil || tp.Constraint() == nil || tp.Index() != 0 || tp.String() == "" || tp.Underlying() == nil {
		t.Fatal("TypeParam methods returned invalid values")
	}
	tp.SetConstraint(tp.Constraint())

	termA := gotypes.NewTerm(true, gotypes.Typ[gotypes.Int])
	termB := gotypes.NewTerm(false, gotypes.Typ[gotypes.String])
	union := gotypes.NewUnion([]*gotypes.Term{termA, termB})
	if union.Len() != 2 || union.Term(0) == nil || countSeq(union.Terms()) != 2 || union.String() == "" || union.Underlying() == nil {
		t.Fatal("Union methods returned invalid values")
	}
	if termA.Type() == nil || termA.String() == "" || !termA.Tilde() {
		t.Fatal("Term methods returned invalid values")
	}

	// Instantiate generic named type and inspect TypeList.
	box := pkg.Scope().Lookup("Box").Type()
	inst, err := gotypes.Instantiate(nil, box, []gotypes.Type{gotypes.Typ[gotypes.Int]}, false)
	if err != nil {
		t.Fatalf("Instantiate(Box[int]): %v", err)
	}
	instNamed, ok := inst.(*gotypes.Named)
	if !ok {
		t.Fatalf("instantiated type = %T, want *types.Named", inst)
	}
	targs := instNamed.TypeArgs()
	if targs == nil || targs.Len() != 1 || targs.At(0) != gotypes.Typ[gotypes.Int] || countSeq(targs.Types()) != 1 {
		t.Fatal("TypeList methods returned invalid values")
	}

	// Array / Chan / Map / Pointer / Slice / Tuple / Struct APIs.
	arr := gotypes.NewArray(gotypes.Typ[gotypes.Int], 2)
	ch := gotypes.NewChan(gotypes.SendOnly, gotypes.Typ[gotypes.String])
	mp := gotypes.NewMap(gotypes.Typ[gotypes.String], gotypes.Typ[gotypes.Int])
	ptr := gotypes.NewPointer(gotypes.Typ[gotypes.Int])
	sli := gotypes.NewSlice(gotypes.Typ[gotypes.Byte])
	tuple := gotypes.NewTuple(gotypes.NewVar(token.NoPos, nil, "a", gotypes.Typ[gotypes.Int]))
	st := gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(token.NoPos, nil, "X", gotypes.Typ[gotypes.Int], false)}, nil)
	if arr.Elem() == nil || arr.String() == "" || arr.Underlying() == nil {
		t.Fatal("Array methods returned invalid values")
	}
	if ch.Elem() == nil || ch.String() == "" || ch.Underlying() == nil {
		t.Fatal("Chan methods returned invalid values")
	}
	if mp.Elem() == nil || mp.String() == "" || mp.Underlying() == nil {
		t.Fatal("Map methods returned invalid values")
	}
	if ptr.Elem() == nil || ptr.String() == "" || ptr.Underlying() == nil {
		t.Fatal("Pointer methods returned invalid values")
	}
	if sli.Elem() == nil || sli.String() == "" || sli.Underlying() == nil {
		t.Fatal("Slice methods returned invalid values")
	}
	if tuple.At(0) == nil || tuple.String() == "" || tuple.Underlying() == nil || countSeq(tuple.Variables()) != 1 {
		t.Fatal("Tuple methods returned invalid values")
	}
	if st.NumFields() != 1 || countSeq(st.Fields()) != 1 || st.String() == "" || st.Underlying() == nil {
		t.Fatal("Struct methods returned invalid values")
	}

	// StdSizes.Offsetsof.
	std := &gotypes.StdSizes{WordSize: 8, MaxAlign: 8}
	if len(std.Offsetsof([]*gotypes.Var{st.Field(0)})) != 1 {
		t.Fatal("StdSizes.Offsetsof returned invalid length")
	}

	// Interface methods not already touched.
	emb := gotypes.NewInterfaceType([]*gotypes.Func{
		gotypes.NewFunc(token.NoPos, pkg, "E", gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)),
	}, nil)
	emb.Complete()
	iface := gotypes.NewInterfaceType([]*gotypes.Func{
		gotypes.NewFunc(token.NoPos, pkg, "M", gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)),
	}, []gotypes.Type{emb})
	iface.MarkImplicit()
	iface.Complete()
	if iface.NumEmbeddeds() > 0 {
		_ = iface.Embedded(0)
		_ = iface.EmbeddedType(0)
		_ = countSeq(iface.EmbeddedTypes())
	}
	if iface.ExplicitMethod(0) == nil || countSeq(iface.ExplicitMethods()) == 0 || iface.Method(0) == nil || countSeq(iface.Methods()) == 0 {
		t.Fatal("Interface method APIs returned invalid values")
	}
	_ = iface.IsComparable()
	_ = iface.IsImplicit()
	_ = iface.Empty()
	_ = iface.NumEmbeddeds()
	_ = iface.NumExplicitMethods()
	if iface.String() == "" || iface.Underlying() == nil {
		t.Fatal("Interface string/underlying returned invalid values")
	}

	// Info helper methods and TypeAndValue helpers.
	for id := range info.Defs {
		if id != nil {
			_ = info.ObjectOf(id)
			_ = info.TypeOf(id)
		}
	}
	for id, obj := range info.Uses {
		if id != nil {
			_ = info.ObjectOf(id)
			_ = info.TypeOf(id)
			_ = obj
		}
	}
	for node := range info.Implicits {
		if spec, ok := node.(*ast.ImportSpec); ok {
			_ = info.PkgNameOf(spec)
		}
	}
	for _, tv := range info.Types {
		_ = tv.Addressable()
		_ = tv.Assignable()
		_ = tv.HasOk()
		_ = tv.IsBuiltin()
		_ = tv.IsNil()
		_ = tv.IsType()
		_ = tv.IsValue()
		_ = tv.IsVoid()
	}

	// Importer / ImporterFrom / ImportMode / Qualifier.
	var imp gotypes.Importer = testImporter(t)
	if _, err := imp.Import("example.org/fixture/fmt"); err != nil {
		t.Fatalf("Importer.Import(fmt): %v", err)
	}
	impFrom, ok := imp.(gotypes.ImporterFrom)
	if !ok {
		t.Fatal("fixture importer should implement types.ImporterFrom")
	}
	if _, err := impFrom.ImportFrom("example.org/fixture/fmt", "", 0); err != nil {
		t.Fatalf("ImporterFrom.ImportFrom(fmt): %v", err)
	}
	var _ gotypes.ImportMode = 0
	var _ gotypes.ChanDir = gotypes.SendRecv
	q := gotypes.RelativeTo(pkg)
	if q(pkg) != "" {
		t.Fatalf("Qualifier for same package should be empty, got %q", q(pkg))
	}
	var _ gotypes.Qualifier = q

	// Type aliases/vars required by API surface.
	var _ *gotypes.Checker
	var _ *gotypes.Context
	var _ gotypes.Importer
	var _ gotypes.ImporterFrom
	_ = gotypes.DefPredeclaredTestFuncs

	// Checker.Files
	fileset := token.NewFileSet()
	file, err := parser.ParseFile(fileset, "checker.go", "package checker\nvar X = 1\n", 0)
	if err != nil {
		t.Fatalf("ParseFile(checker): %v", err)
	}
	checkPkg := gotypes.NewPackage("example.org/checker", "checker")
	checkInfo := &gotypes.Info{}
	ensureInfoMaps(checkInfo)
	checker := gotypes.NewChecker(&gotypes.Config{Importer: testImporter(t)}, fileset, checkPkg, checkInfo)
	if err := checker.Files([]*ast.File{file}); err != nil {
		t.Fatalf("Checker.Files: %v", err)
	}

	// ArgumentError methods.
	argErr := &gotypes.ArgumentError{Err: errors.New("arg")}
	if argErr.Error() == "" {
		t.Fatal("ArgumentError.Error returned empty")
	}
	_ = argErr.Unwrap()

	// Initializer and Initializer.String.
	if len(checkInfo.InitOrder) > 0 {
		init := checkInfo.InitOrder[0]
		if init.String() == "" {
			t.Fatal("Initializer.String returned empty")
		}
	}
}
