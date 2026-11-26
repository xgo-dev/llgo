package ssa

import (
	"go/importer"
	"go/token"
	"go/types"
	"runtime"
	"testing"
)

func TestMethodTypeMetadataGatedByDeadcodeDrop(t *testing.T) {
	Initialize(InitAll)
	rtPkg, err := importer.For("source", nil).Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}

	newNamedWithMethods := func(pkgPath string) *types.Named {
		goPkg := types.NewPackage(pkgPath, "demo")
		named := types.NewNamed(types.NewTypeName(token.NoPos, goPkg, "Worker", nil), types.NewStruct(nil, nil), nil)
		recv := types.NewVar(token.NoPos, goPkg, "w", named)
		sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
		named.AddMethod(types.NewFunc(token.NoPos, goPkg, "Run", sig))
		named.AddMethod(types.NewFunc(token.NoPos, goPkg, "helper", sig))
		return named
	}

	t.Run("globaldce only skips metadata map", func(t *testing.T) {
		prog := NewProgram(nil)
		defer prog.Dispose()
		prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
		prog.SetRuntime(func() *types.Package { return rtPkg })
		if prog.GoGlobalDCEEnabled() {
			t.Fatal("GoGlobalDCEEnabled() = true by default, want false")
		}
		prog.EnableGoGlobalDCE(true)
		if !prog.GoGlobalDCEEnabled() {
			t.Fatal("GoGlobalDCEEnabled() = false after EnableGoGlobalDCE(true)")
		}
		named := newNamedWithMethods("example.com/nodce")
		pkg := prog.NewPackage("demo", "example.com/nodce")
		fn := pkg.NewFunc("use", NoArgsNoRet, GoDecl)
		b := fn.MakeBody(1)
		b.abiType(named)
		b.Return()
		b.EndBuild()
		b.Dispose()

		if got := len(pkg.MethodTypeMetadata()); got != 0 {
			t.Fatalf("len(MethodTypeMetadata()) = %d without deadcodeDrop, want 0", got)
		}
	})

	t.Run("globaldce with deadcodeDrop records VFE entries", func(t *testing.T) {
		prog := NewProgram(nil)
		defer prog.Dispose()
		prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
		prog.SetRuntime(func() *types.Package { return rtPkg })
		prog.EnableGoGlobalDCE(true)
		prog.EnableDeadcodeDrop(true)

		named := newNamedWithMethods("example.com/withdce")
		pkg := prog.NewPackage("demo", "example.com/withdce")
		fn := pkg.NewFunc("use", NoArgsNoRet, GoDecl)
		b := fn.MakeBody(1)
		b.abiType(named)
		b.Return()
		b.EndBuild()
		b.Dispose()

		metaMap := pkg.MethodTypeMetadata()
		if len(metaMap) == 0 {
			t.Fatal("MethodTypeMetadata() is empty with GoGlobalDCE + DeadcodeDrop enabled")
		}
		name, _ := prog.abi.TypeName(named)
		got, ok := metaMap[name]
		if !ok {
			t.Fatalf("MethodTypeMetadata() missing %q, have keys %v", name, metaMap)
		}
		if got.VCallVisibility != vcallVisibilityLinkageUnit {
			t.Fatalf("VCallVisibility = %d, want %d", got.VCallVisibility, vcallVisibilityLinkageUnit)
		}
		// Exported method "Run" records 5 entries (capability + 4 reflect entries);
		// unexported "helper" records 1 capability entry.
		if len(got.Entries) != 6 {
			t.Fatalf("len(Entries) = %d, want 6: %+v", len(got.Entries), got.Entries)
		}
		// Empty metadata / empty name are ignored cleanly.
		pkg.recordMethodTypeMetadata("", got)
		pkg.recordMethodTypeMetadata("ignored", MethodTypeMeta{})
		if _, ok := pkg.MethodTypeMetadata()["ignored"]; ok {
			t.Fatal("empty MethodTypeMeta was recorded")
		}
	})
}
