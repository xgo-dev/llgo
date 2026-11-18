package build

import (
	"go/constant"
	"go/types"
	"testing"

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
				specializeFormat(call, true)
				if got := call.Common().StaticCallee().Name(); got != "Sprintf" {
					t.Fatalf("specialized a call with %s: %s", name, got)
				}
				return
			}
			t.Fatal("missing fmt.Sprintf call")
		})
	}
}
