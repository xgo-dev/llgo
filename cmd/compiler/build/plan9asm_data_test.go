package build

import (
	"strings"
	"testing"

	gllvm "github.com/xgo-dev/llvm"
)

func TestExternalizePlan9DataGlobals(t *testing.T) {
	ctx := gllvm.NewContext()
	defer ctx.Dispose()
	goMod := ctx.NewModule("go")
	defer goMod.Dispose()
	asmMod := ctx.NewModule("asm")
	defer asmMod.Dispose()
	td := gllvm.NewTargetData("e-p:64:64-i64:64-n32:64")
	defer td.Dispose()

	i64 := ctx.Int64Type()
	goGlobal := gllvm.AddGlobal(goMod, i64, "main.value")
	goGlobal.SetInitializer(gllvm.ConstNull(i64))
	bytes := gllvm.ArrayType(ctx.Int8Type(), 8)
	asmGlobal := gllvm.AddGlobal(asmMod, bytes, "main.value")
	asmGlobal.SetInitializer(gllvm.ConstNull(bytes))

	if err := externalizePlan9DataGlobals(goMod, asmMod, td); err != nil {
		t.Fatal(err)
	}
	if !goGlobal.IsDeclaration() || goGlobal.Linkage() != gllvm.ExternalLinkage {
		t.Fatalf("Go DATA target was not externalized:\n%s", goMod.String())
	}
}

func TestExternalizePlan9DataGlobalsRejectsConflicts(t *testing.T) {
	tests := []struct {
		name           string
		goTypeSize     int
		goValue        uint64
		asmThreadLocal bool
		want           string
	}{
		{name: "thread local", goTypeSize: 8, asmThreadLocal: true, want: "incompatible thread-local storage"},
		{name: "size", goTypeSize: 4, want: "Go size 4 but DATA size 8"},
		{name: "initializer", goTypeSize: 8, goValue: 1, want: "both a Go initializer and DATA"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := gllvm.NewContext()
			defer ctx.Dispose()
			goMod := ctx.NewModule("go")
			defer goMod.Dispose()
			asmMod := ctx.NewModule("asm")
			defer asmMod.Dispose()
			td := gllvm.NewTargetData("e-p:64:64-i64:64-n32:64")
			defer td.Dispose()

			goType := ctx.IntType(test.goTypeSize * 8)
			goGlobal := gllvm.AddGlobal(goMod, goType, "main.value")
			goGlobal.SetInitializer(gllvm.ConstInt(goType, test.goValue, false))
			asmType := gllvm.ArrayType(ctx.Int8Type(), 8)
			asmGlobal := gllvm.AddGlobal(asmMod, asmType, "main.value")
			asmGlobal.SetInitializer(gllvm.ConstNull(asmType))
			asmGlobal.SetThreadLocal(test.asmThreadLocal)

			err := externalizePlan9DataGlobals(goMod, asmMod, td)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("externalizePlan9DataGlobals() error = %v, want %q", err, test.want)
			}
		})
	}
}
