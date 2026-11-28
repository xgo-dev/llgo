package build

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/fmtspecialize"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
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

func TestSpecializeScalarFormatsDynamicBuiltin(t *testing.T) {
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

func TestSpecializeScalarFormatsPrintfWithFiniteTypes(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		extraRoot  bool
		kinds      uint64
	}{
		{"dynamic format", `fmt.Printf(format, "value")`, false, 1 << (types.String - types.Bool)},
		{"phi", `var v any = 1.25; if cond { v = "value" }; fmt.Printf(format, v)`, false,
			1<<(types.String-types.Bool) | 1<<(types.Float64-types.Bool)},
		{"private parameter", `if cond { render("value") } else { render(1.25) }`, false,
			1<<(types.String-types.Bool) | 1<<(types.Float64-types.Bool)},
		{"address taken", `f := []func(any){render}; f[0]("value")`, false, 0},
		{"closure caller", `render("v"); (func() { render(1.5) })()`, false, 1<<(types.String-types.Bool) | 1<<(types.Float64-types.Bool)},
		{"entry parameter", `render("value")`, true, 0},
		{"unknown incoming branch", `var v any = "value"; if cond { v = input }; render(v)`, false, 0},
		{"named scalar method", `render(label(1))`, false, 0},
		{"mixed generic formatter", `render("value"); fmt.Printf("%v", input)`, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			fmtPkg := buildFormatTestPackage(t, prog, "fmt", `package fmt
type pp struct{}
func (*pp) printArg(_ any) string { return "generic" }
func Sprintf(format string, args ...any) string { return (&pp{}).printArg(args) }
func Printf(format string, args ...any) (int, error) { return len((&pp{}).printArg(args)), nil }
func llgoPrintfBuiltins(format string, kinds uint32, args ...any) (int, error) { return len(format), nil }
`, nil)
			main := buildFormatTestPackage(t, prog, "main", `package main
import "fmt"
type label int
func (label) String() string { return "label" }
var cond bool
var input any
var format string
func render(v any) { fmt.Printf("%v", v) }
func main() { `+tc.body+` }
`, fmtPkg.Pkg)
			prog.Build()
			plan := fmtspecialize.NewValuePlan(prog, nil)
			for fn := range ssautil.AllFunctions(prog) {
				plan.RecordFunction(fn)
				if fn.Pkg != main {
					continue
				}
				for _, block := range fn.Blocks {
					for _, inst := range block.Instrs {
						plan.Record(inst, main)
					}
				}
			}
			roots := []*ssa.Function{main.Func("main")}
			if tc.extraRoot {
				roots = append(roots, main.Func("render"))
			}
			applied := plan.ApplyStandalone(roots)
			if (len(applied) != 0) != (tc.kinds != 0) {
				t.Fatalf("selected specialization for %v", applied)
			}
			for _, call := range plan.BuiltinPrintfCalls(main) {
				mask, ok := constant.Uint64Val(call.Common().Args[1].(*ssa.Const).Value)
				if !ok || mask != tc.kinds {
					t.Fatalf("scalar mask = %#x, want %#x", mask, tc.kinds)
				}
			}
		})
	}
}
