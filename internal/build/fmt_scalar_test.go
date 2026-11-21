package build

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"golang.org/x/tools/go/ssa"
)

func TestSpecializeScalarFormats(t *testing.T) {
	tests := []struct {
		name        string
		opt         optlevel.Level
		helpers     bool
		debugger    bool
		wantScalars bool
	}{
		{name: "size", opt: optlevel.Oz, helpers: true, wantScalars: true},
		{name: "O2", opt: optlevel.O2, helpers: true},
		{name: "missing helpers", opt: optlevel.Oz},
		{name: "logical frames", opt: optlevel.Oz, helpers: true, debugger: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := literalFormatProgram(t, test.helpers)
			ctx := &context{progSSA: pkg.Prog, buildConf: &Config{OptLevel: test.opt}}
			ctx.frontendOptions.DebuggerFrames = test.debugger
			if err := recordPackageSSAInstructions(ctx); err != nil {
				t.Fatal(err)
			}
			for name, original := range map[string]string{
				"scalar": "Sprintf", "scalarPrint": "Printf", "scalarDefer": "Printf", "scalarGo": "Printf",
				"scalarDefault": "Sprintf", "scalarBadDefault": "Printf",
				"scalarPrintln": "Println", "scalarSprintln": "Sprintln",
				"scalarEmptyPrintln": "Println", "scalarComplexPrintln": "Println",
				"scalarSpread": "Printf", "scalarLiteralSpread": "Printf", "scalarInterface": "Printf",
				"scalarNamed": "Printf", "scalarNil": "Printf",
				"scalarTooFew": "Printf", "scalarTooMany": "Printf", "scalarBadType": "Printf", "scalarStar": "Printf",
				"scalarBadBool": "Printf", "scalarBadInt": "Printf", "scalarTrailingPercent": "Printf",
				"scalarIndex": "Printf", "scalarPrecision": "Printf", "scalarFlag": "Printf", "scalarLarge": "Printf",
				"scalarTruncated": "Printf", "scalarWriter": "Fprintf",
			} {
				want := original
				var wantFormat string
				if test.wantScalars {
					switch name {
					case "scalar":
						want, wantFormat = "llgoSprintfScalars", "id=%04X %% name=%8s valid=%t"
					case "scalarPrint", "scalarDefer", "scalarGo":
						want = "llgoPrintfScalars"
					case "scalarDefault":
						want, wantFormat = "llgoSprintfScalars", "%03d %5s %t"
					case "scalarPrintln":
						want, wantFormat = "llgoPrintfScalars", "%d %s %t\n"
					case "scalarSprintln":
						want, wantFormat = "llgoSprintfScalars", "%s\n"
					}
				}
				var common *ssa.CallCommon
				for _, block := range pkg.Func(name).Blocks {
					for _, instr := range block.Instrs {
						call, ok := instr.(ssa.CallInstruction)
						if ok && call.Common().StaticCallee() != nil {
							common = call.Common()
						}
					}
				}
				if common == nil {
					t.Fatalf("%s has no call", name)
				}
				if got := common.StaticCallee().Name(); got != want {
					t.Errorf("%s calls %q, want %q", name, got, want)
				} else if wantFormat != "" {
					if got := constant.StringVal(common.Args[0].(*ssa.Const).Value); got != wantFormat {
						t.Errorf("%s uses format %q, want %q", name, got, wantFormat)
					}
				}
			}
		})
	}
}

// The fast path depends on the shape of the compiler-generated argument
// array. If it gains other uses or writes, retain the normal formatter.
func TestScalarFormatsRequireKnownArgumentStorage(t *testing.T) {
	for name, change := range map[string]func(*ssa.Slice, *ssa.IndexAddr){
		"shared slice": func(slice *ssa.Slice, _ *ssa.IndexAddr) {
			refs := slice.Referrers()
			*refs = append(*refs, (*refs)[0])
		},
		"shared slot": func(_ *ssa.Slice, slot *ssa.IndexAddr) {
			refs := slot.Referrers()
			*refs = append(*refs, (*refs)[0])
		},
		"variable index": func(_ *ssa.Slice, slot *ssa.IndexAddr) {
			slot.Index = slot.Parent().Params[0]
		},
		"unknown index": func(_ *ssa.Slice, slot *ssa.IndexAddr) {
			slot.Index = ssa.NewConst(constant.MakeInt64(-1), types.Typ[types.Int])
		},
		"unknown slot use": func(slice *ssa.Slice, slot *ssa.IndexAddr) {
			(*slot.Referrers())[0] = slice
		},
	} {
		t.Run(name, func(t *testing.T) {
			pkg := literalFormatProgram(t, true)
			for _, inst := range pkg.Func("scalar").Blocks[0].Instrs {
				call, ok := inst.(*ssa.Call)
				if !ok {
					continue
				}
				slice := call.Common().Args[1].(*ssa.Slice)
				array := slice.X.(*ssa.Alloc)
				slot := (*array.Referrers())[0].(*ssa.IndexAddr)
				change(slice, slot)
				fmtspecialize.RewriteCall(call, true)
				if got := call.Common().StaticCallee().Name(); got != "Sprintf" {
					t.Fatalf("specialized a call with %s: %s", name, got)
				}
				return
			}
			t.Fatal("missing fmt.Sprintf call")
		})
	}
}

func TestSpecializeBuiltinDynamicFormat(t *testing.T) {
	for _, test := range []struct {
		name, operands, otherOperands string
		kinds                         []types.BasicKind
	}{
		{"builtin values", ", 17, 1.5, false, \"label\"", ", 18, byte(5), rune('界')",
			[]types.BasicKind{types.Bool, types.Int, types.Float64, types.String, types.Uint8, types.Int32}},
		{"empty arguments", "", "", nil},
		{"named value", ", named(3)", ", 18", nil},
		{"interface value", ", value", ", 18", nil},
		{"unsafe pointer", ", pointer", ", 18", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			prog.CreatePackage(types.Unsafe, nil, nil, true)
			fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
type pp struct{}
func (*pp) printArg(_ any) string { return "generic" }
func Sprintf(format string, args ...any) string { return (&pp{}).printArg(args) }
func llgoSprintfBuiltins(format string, kinds uint32, args ...any) string { return format }
`, nil)
			main := buildFormatTestPackage(t, prog, "main", `package main
import "fmt"
import "unsafe"
type named int
func render(format string, value any, pointer unsafe.Pointer) string {
	return fmt.Sprintf(format`+test.operands+`) + fmt.Sprintf(format`+test.otherOperands+`)
}
`, fmtPkg.Pkg)
			prog.Build()
			render := main.Func("render")
			plan := fmtspecialize.NewValuePlan(prog, nil)
			var formatted []*ssa.Call
			for _, instr := range render.Blocks[0].Instrs {
				if call, ok := instr.(*ssa.Call); ok && call.Common().StaticCallee() == fmtPkg.Func("Sprintf") {
					formatted = append(formatted, call)
					plan.Record(call, main)
				}
			}
			if len(formatted) != 2 {
				t.Fatalf("found %d Sprintf calls, want two", len(formatted))
			}
			plan.ApplyStandalone([]*ssa.Function{render})
			var wantMask uint64
			for _, kind := range test.kinds {
				wantMask |= 1 << (kind - types.Bool)
			}
			for _, call := range formatted {
				common := call.Common()
				gotBuiltin := common.StaticCallee() == fmtPkg.Func("llgoSprintfBuiltins")
				if gotBuiltin != (wantMask != 0) {
					t.Fatalf("builtin formatter selected = %v, want %v", gotBuiltin, wantMask != 0)
				}
				if gotBuiltin {
					mask := common.Args[1].(*ssa.Const)
					if got, ok := constant.Uint64Val(mask.Value); !ok || got != wantMask {
						t.Fatalf("builtin kind mask = %v, want %d", mask, wantMask)
					}
				} else if common.StaticCallee() != fmtPkg.Func("Sprintf") || len(common.Args) != 2 {
					t.Fatalf("fallback did not restore call: %v", common)
				}
			}
		})
	}
}
