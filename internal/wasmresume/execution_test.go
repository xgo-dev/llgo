package wasmresume

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLowerExecutesRequiredWasmProfiles(t *testing.T) {
	wasmLD, err := exec.LookPath("wasm-ld")
	if err != nil {
		t.Skip("wasm-ld is not installed")
	}
	node, nodeErr := exec.LookPath("node")
	wasmtime, wasmtimeErr := exec.LookPath("wasmtime")

	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()

	tests := []struct {
		name   string
		triple string
		run    func(*testing.T, string) ([]byte, error)
	}{
		{
			name:   "J32",
			triple: "wasm32-unknown-emscripten",
			run: func(t *testing.T, wasm string) ([]byte, error) {
				if nodeErr != nil {
					t.Skip("node is not installed")
				}
				script := `const fs=require("fs");WebAssembly.instantiate(fs.readFileSync(process.argv[1])).then(({instance})=>console.log(instance.exports["run.state.machine"]()))`
				return exec.Command(node, "-e", script, wasm).CombinedOutput()
			},
		},
		{
			name:   "J64",
			triple: "wasm64-unknown-emscripten",
			run: func(t *testing.T, wasm string) ([]byte, error) {
				if nodeErr != nil {
					t.Skip("node is not installed")
				}
				script := `const fs=require("fs");WebAssembly.instantiate(fs.readFileSync(process.argv[1])).then(({instance})=>console.log(instance.exports["run.state.machine"]()))`
				return exec.Command(node, "-e", script, wasm).CombinedOutput()
			},
		},
		{
			name:   "P1",
			triple: "wasm32-unknown-wasip1",
			run: func(t *testing.T, wasm string) ([]byte, error) {
				if wasmtimeErr != nil {
					t.Skip("wasmtime is not installed")
				}
				return exec.Command(wasmtime, "run", "--invoke", "run.state.machine", wasm).CombinedOutput()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := buildExecutableWasmResumeObject(t, test.triple)
			dir := t.TempDir()
			objectPath := filepath.Join(dir, "resume.o")
			wasmPath := filepath.Join(dir, "resume.wasm")
			if err := os.WriteFile(objectPath, object, 0o600); err != nil {
				t.Fatal(err)
			}
			linkArgs := []string{
				"--no-entry",
				"--export=run.state.machine",
				"-o", wasmPath,
				objectPath,
			}
			if strings.HasPrefix(test.triple, "wasm64-") {
				linkArgs = append([]string{"-mwasm64"}, linkArgs...)
			}
			if output, err := exec.Command(wasmLD, linkArgs...).CombinedOutput(); err != nil {
				t.Fatalf("link %s: %v\n%s", test.name, err, output)
			}
			output, err := test.run(t, wasmPath)
			if err != nil {
				if test.name == "J64" &&
					strings.Contains(string(output), "invalid table elements limits flags") {
					version, _ := exec.Command(node, "--version").CombinedOutput()
					t.Skipf(
						"node %s does not support LLVM wasm64 table limits",
						strings.TrimSpace(string(version)),
					)
				}
				t.Fatalf("execute %s: %v\n%s", test.name, err, output)
			}
			fields := strings.Fields(string(output))
			if len(fields) == 0 || fields[len(fields)-1] != "14" {
				t.Fatalf("%s result = %q, want 14", test.name, output)
			}
		})
	}
}

func buildExecutableWasmResumeObject(t *testing.T, triple string) []byte {
	t.Helper()
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule(triple)
	defer mod.Dispose()

	target, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(
		triple, "", "", llvm.CodeGenLevelNone, llvm.RelocDefault, llvm.CodeModelDefault,
	)
	defer machine.Dispose()
	targetData := machine.CreateTargetData()
	defer targetData.Dispose()
	mod.SetTarget(triple)
	mod.SetDataLayout(targetData.String())

	i32 := ctx.Int32Type()
	sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	callee := llvm.AddFunction(mod, "callee", sig)
	markFunction(ctx, callee)
	block := ctx.AddBasicBlock(callee, "entry")
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(block)
	builder.CreateRet(builder.CreateAdd(callee.Param(0), llvm.ConstInt(i32, 1, false), "result"))

	suspend := llvm.AddFunction(mod, SuspendSymbol, llvm.FunctionType(ctx.VoidType(), nil, false))
	markFunction(ctx, suspend)
	caller := llvm.AddFunction(mod, "caller", sig)
	markFunction(ctx, caller)
	block = ctx.AddBasicBlock(caller, "entry")
	builder.SetInsertPointAtEnd(block)
	call := builder.CreateCall(sig, callee, []llvm.Value{caller.Param(0)}, "called")
	markCall(ctx, call)
	suspendCall := builder.CreateCall(suspend.GlobalValueType(), suspend, nil, "")
	markCall(ctx, suspendCall)
	builder.CreateRet(builder.CreateMul(call, llvm.ConstInt(i32, 2, false), "result"))

	lowered, err := lowerPrototype(mod, targetData)
	if err != nil {
		t.Fatal(err)
	}
	var root loweredState
	for _, state := range lowered {
		if state.layout.plan.function == caller {
			root = state
			break
		}
	}
	if root.entry.IsNil() {
		t.Fatal("caller state machine was not emitted")
	}
	defineStateMachineHarness(mod, targetData, root, []llvm.Value{
		llvm.ConstInt(i32, 6, false),
	})
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify %s executable: %v\n%s", triple, err, mod.String())
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := mod.RunPasses("default<O2>", machine, options); err != nil {
		t.Fatalf("optimize %s executable: %v\n%s", triple, err, mod.String())
	}
	buffer, err := machine.EmitToMemoryBuffer(mod, llvm.ObjectFile)
	if err != nil {
		t.Fatalf("emit %s executable: %v\n%s", triple, err, mod.String())
	}
	defer buffer.Dispose()
	return append([]byte(nil), buffer.Bytes()...)
}
