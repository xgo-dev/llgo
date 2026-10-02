package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestCompileDebugMetadata(t *testing.T) {
	const source = `package debugcompile

type item struct { value int }

func inspect(items [2]item, seed int) int {
	x := seed + 1
	var local [1]item
	if x > 0 {
		items[0].value = x
		local[0].value = x
		seed = x
	} else {
		seed = x
	}
	seed = 42
	return items[0].value + local[0].value + seed
}

var anonymous = func(seed int) int {
	value := seed + 1
	return value
}

func snapshots(n int) complex128 {
	var result complex128
	for i := 0; i < n; i++ {
		value := complex(float64(i), float64(i+1))
		result += value
	}
	return result
}

func inspectAddressable(seed int) [2]int {
	backed := [2]int{seed, seed + 1}
	backed[0] = 42
	return backed
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "debug_compile.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("debugcompile", "debugcompile"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}

	prog := newLLSSAProgForTarget(t, &llssa.Target{
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		OptLevel: optlevel.O0,
	})
	defer prog.Dispose()
	if err := prepareTestSyntax(prog, ssaPkg, []*ast.File{file}, Options{}); err != nil {
		t.Fatal(err)
	}
	prog.Directives().Freeze()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{
		Debug:        true,
		DebugSymbols: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("debug module is invalid: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"#dbg_declare",
		"#dbg_value",
		"DILexicalBlock",
		`name: "items", arg: 1`,
		`name: "seed", arg: 2`,
		`name: "x"`,
		`name: "local"`,
		`name: "value"`,
		"isOptimized: false",
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("debug module is missing %q:\n%s", want, ir)
		}
	}
	assertDebugRecords(t, ir, `name: "items", arg: 1`, true, false)
	assertDebugRecords(t, ir, `name: "seed", arg: 2`, true, false)
	assertDebugRecords(t, ir, `name: "backed"`, true, false)
	assertDebugHomeStores(t, ir, pkg.Module().NamedFunction("debugcompile.inspect").String(), `name: "seed", arg: 2`, 4)
	assertDebugLoopSnapshots(t, pkg.Module().NamedFunction("debugcompile.snapshots"))

	optimizedProg := newLLSSAProgForTarget(t, &llssa.Target{
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		OptLevel: optlevel.O2,
	})
	defer optimizedProg.Dispose()
	optimizedPkg, _, err := newPackageEx(optimizedProg, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{Debug: true, DebugSymbols: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(optimizedPkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("optimized debug module is invalid: %v\n%s", err, optimizedPkg.Module().String())
	}
	optimizedIR := optimizedPkg.Module().String()
	assertDebugRecords(t, optimizedIR, `name: "items", arg: 1`, true, false)
	assertDebugRecords(t, optimizedIR, `name: "seed", arg: 2`, false, true)
	// Addressable aggregates must keep their real storage at O2 too. A value
	// DebugRef after the field update must not replace it with a snapshot.
	assertDebugRecords(t, optimizedIR, `name: "local"`, true, false)
	assertDebugRecords(t, optimizedIR, `name: "backed"`, true, false)
}

func assertDebugLoopSnapshots(t *testing.T, function llvm.Value) {
	t.Helper()
	if function.IsNil() {
		t.Fatal("missing snapshot loop function")
	}
	entry := function.EntryBasicBlock()
	allocations, loopStores := 0, 0
	for block := entry; !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			switch inst.InstructionOpcode() {
			case llvm.Alloca:
				allocations++
				if block != entry {
					t.Fatalf("debug snapshot repeatedly reserves stack space in a loop:\n%s", function.String())
				}
			case llvm.Store:
				if block != entry {
					loopStores++
				}
			}
		}
	}
	if allocations == 0 || loopStores == 0 {
		t.Fatalf("snapshot loop needs fixed homes and per-iteration value updates:\n%s", function.String())
	}
}

func assertDebugHomeStores(t *testing.T, ir, functionIR, variable string, minimum int) {
	t.Helper()
	variableID := debugVariableID(t, ir, variable)
	re := regexp.MustCompile(`#dbg_declare\(ptr ([^,]+), ` + regexp.QuoteMeta(variableID) + `,`)
	match := re.FindStringSubmatch(functionIR)
	if len(match) != 2 {
		t.Fatalf("debug home for %q not found:\n%s", variable, ir)
	}
	stores := 0
	for _, line := range strings.Split(functionIR, "\n") {
		if strings.Contains(line, "store ") && strings.Contains(line, ", ptr "+match[1]+",") {
			stores++
			if strings.Contains(line, "!dbg") {
				t.Fatalf("debug home store for %q has a source location: %s", variable, line)
			}
		}
	}
	if stores < minimum {
		t.Fatalf("debug home for %q has %d stores, want at least %d\n%s", variable, stores, minimum, ir)
	}
}

func assertDebugRecords(t *testing.T, ir, variable string, wantDeclare, wantValue bool) {
	t.Helper()
	variableID := debugVariableID(t, ir, variable)
	var declare, value bool
	for _, line := range strings.Split(ir, "\n") {
		if !strings.Contains(line, variableID+",") {
			continue
		}
		declare = declare || strings.Contains(line, "#dbg_declare")
		value = value || strings.Contains(line, "#dbg_value")
	}
	if declare != wantDeclare || value != wantValue {
		t.Fatalf("debug records for %q: declare=%v value=%v, want declare=%v value=%v\n%s",
			variable, declare, value, wantDeclare, wantValue, ir)
	}
}

func debugVariableID(t *testing.T, ir, variable string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^(![0-9]+) = !DILocalVariable\(` + regexp.QuoteMeta(variable))
	match := re.FindStringSubmatch(ir)
	if len(match) != 2 {
		t.Fatalf("debug variable %q not found:\n%s", variable, ir)
	}
	return match[1]
}
