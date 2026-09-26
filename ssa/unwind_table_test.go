//go:build !llgo

package ssa

import (
	"bytes"
	"debug/pe"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWindowsUnwindTablesForNoUnwindFunctions(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			pkg := prog.NewPackage("p", "example.com/p")
			for _, bg := range []Background{InGo, InC} {
				name := "go_frame"
				if bg == InC {
					name = "c_export_frame"
				}
				fn := pkg.NewFunc(name, NoArgsNoRet, bg)
				fn.impl.AddFunctionAttr(prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("nounwind"), 0))
				b := prog.ctx.NewBuilder()
				b.SetInsertPointAtEnd(prog.ctx.AddBasicBlock(fn.impl, "entry"))
				slot := b.CreateAlloca(prog.ctx.Int64Type(), "slot")
				b.CreateStore(llvm.ConstInt(prog.ctx.Int64Type(), 1, false), slot).SetVolatile(true)
				b.CreateRetVoid()
				b.Dispose()
			}
			mod := pkg.Module()
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			obj, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			defer obj.Dispose()
			coff, err := pe.NewFile(bytes.NewReader(obj.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer coff.Close()
			if section := coff.Section(".pdata"); section == nil || section.Size == 0 {
				t.Fatalf("optimized nounwind functions lost Windows unwind metadata:\n%s", mod.String())
			}
		})
	}
}
