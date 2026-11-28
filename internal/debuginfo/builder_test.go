package debuginfo

import (
	"debug/dwarf"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llvm"
)

func TestBuilderLinksWithClangWcharABI(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is required to verify C/Go debug module linking")
	}
	for _, tt := range []struct {
		name       string
		target     string
		wcharSize  int
		shortWchar bool
	}{
		{name: "windows_amd64", target: "x86_64-pc-windows-msvc", wcharSize: 2},
		{name: "windows_386", target: "i686-pc-windows-msvc", wcharSize: 2},
		{name: "windows_arm64", target: "aarch64-pc-windows-msvc", wcharSize: 2},
		{name: "windows_gnu", target: "x86_64-w64-windows-gnu", wcharSize: 2},
		{name: "windows_gnu_386", target: "i686-w64-windows-gnu", wcharSize: 2},
		{name: "darwin", target: "aarch64-apple-darwin", wcharSize: 4},
		{name: "linux", target: "x86_64-unknown-linux-gnu", wcharSize: 4},
		{name: "linux_short_wchar", target: "x86_64-unknown-linux-gnu", wcharSize: 2, shortWchar: true},
		{name: "wasm", target: "wasm32-unknown-wasip1", wcharSize: 4},
		{name: "cortex_m", target: "thumbv7m-unknown-unknown-eabi", wcharSize: 4},
		{name: "avr", target: "avr", wcharSize: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Compile without system headers or libraries so this exercises real
			// target ABI flags on every host, including Windows cross-targets.
			path := filepath.Join(t.TempDir(), "c.ll")
			args := []string{"--target=" + tt.target, "-x", "c", "-g", "-S", "-emit-llvm", "-o", path, "-"}
			if tt.shortWchar {
				args = append(args, "-fshort-wchar")
			}
			cmd := exec.Command(clang, args...)
			cmd.Stdin = strings.NewReader("int c_wchar_size(void) { return sizeof(__WCHAR_TYPE__); }")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("compile C module: %v\n%s", err, out)
			}
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			buf, err := llvm.NewMemoryBufferFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			cModule, err := ctx.ParseIR(buf)
			if err != nil {
				t.Fatalf("parse C module: %v", err)
			}
			wantFlag := fmt.Sprintf(`!{i32 1, !"wchar_size", i32 %d}`, tt.wcharSize)
			cIR := cModule.String()
			wcharSize := cModule.NamedFunction("c_wchar_size").FirstBasicBlock().LastInstruction().Operand(0)
			if wcharSize.IsAConstantInt().IsNil() || wcharSize.ZExtValue() != uint64(tt.wcharSize) {
				t.Fatalf("C module has the wrong wchar_t size:\n%s", cIR)
			}
			if strings.Contains(cIR, `!"wchar_size"`) && !strings.Contains(cIR, wantFlag) {
				t.Fatalf("C module has an unexpected wchar_size flag:\n%s", cIR)
			}
			// LLVM 23 omits this flag for the target's default wchar_t ABI.
			// Keep explicit flags in the link test as well: precompiled C
			// bitcode and -fshort-wchar modules still carry this ABI contract.
			if !strings.Contains(cIR, `!"wchar_size"`) {
				(&Builder{module: cModule}).addModuleFlag(1, "wchar_size", tt.wcharSize)
			}

			module := ctx.NewModule("go-debug-test")
			defer module.Dispose()
			module.SetTarget(cModule.Target())
			module.SetDataLayout(cModule.DataLayout())
			builder := New(module, Config{EmitCodeView: strings.Contains(tt.target, "windows")})
			builder.CompileUnit("main.go", "/src/example")
			builder.Finalize()

			// LLVM's LTO linker must accept the Go debug module and retain the
			// ABI selected by the C compiler. LinkModules consumes cModule.
			if err := llvm.LinkModules(module, cModule); err != nil {
				t.Fatalf("link C and Go debug modules: %v", err)
			}
			if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("linked module is invalid: %v\n%s", err, module.String())
			}
			if module.NamedFunction("c_wchar_size").IsNil() || module.NamedGlobal(debuggerMarkerSymbol).IsNil() {
				t.Fatalf("linked module lost C code or the Go debugger marker:\n%s", module.String())
			}
			if !strings.Contains(module.String(), wantFlag) {
				t.Fatalf("linked module lost C ABI flag %s:\n%s", wantFlag, module.String())
			}
		})
	}
}

func TestBuilderLifecycleAndModuleMetadata(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	module := ctx.NewModule("debug-test")
	defer module.Dispose()

	builder := New(module, Config{Producer: "LLGo", Optimized: true})
	cu := builder.CompileUnit("main.go", "/src/example")
	file := builder.File("/src/example/main.go")
	if got := builder.File("/src/example/dir/../main.go"); got != file {
		t.Fatal("File did not reuse metadata for the same cleaned path")
	}

	voidType := builder.CreateSubroutineType(llvm.DISubroutineType{File: file})
	subprogram := builder.CreateFunction(cu, llvm.DIFunction{
		Name:         "main.main",
		LinkageName:  "main.main",
		File:         file,
		Line:         1,
		ScopeLine:    1,
		Type:         voidType,
		IsDefinition: true,
	})
	function := llvm.AddFunction(module, "main.main", llvm.FunctionType(ctx.VoidType(), nil, false))
	function.SetSubprogram(subprogram)
	irBuilder := ctx.NewBuilder()
	defer irBuilder.Dispose()
	block := llvm.AddBasicBlock(function, "entry")
	irBuilder.SetInsertPointAtEnd(block)
	irBuilder.CreateRetVoid()

	builder.Finalize()
	builder.Finalize()
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("finalized debug module is invalid: %v\n%s", err, module.String())
	}

	ir := module.String()
	for _, want := range []string{
		`!llvm.dbg.cu`,
		`!DICompileUnit(language: DW_LANG_C`,
		`producer: "LLGo"`,
		`isOptimized: true`,
		`@__llgo_debugger_marker_v1 = linkonce_odr hidden constant i8 1`,
		`@llvm.used = appending global [1 x ptr] [ptr @__llgo_debugger_marker_v1], section "llvm.metadata"`,
		`!{i32 7, !"Dwarf Version", i32 4}`,
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("module is missing %q:\n%s", want, ir)
		}
	}
}

func TestBuilderWindowsDebuggerMarkerUsesComdat(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	module := ctx.NewModule("windows-debug-test")
	defer module.Dispose()
	module.SetTarget("aarch64-pc-windows-msvc")

	builder := New(module, Config{})
	builder.CompileUnit("main.go", "/src/example")
	builder.Finalize()

	marker := module.NamedGlobal(debuggerMarkerSymbol)
	if marker.IsNil() {
		t.Fatal("debugger marker is missing")
	}
	if got := marker.Comdat().SelectionKind(); got != llvm.AnyComdatSelectionKind {
		t.Fatalf("debugger marker COMDAT selection = %v, want any", got)
	}
	if got := marker.DLLStorageClass(); got != llvm.DLLExportStorageClass {
		t.Fatalf("debugger marker DLL storage class = %v, want export", got)
	}
	if got := marker.Visibility(); got != llvm.DefaultVisibility {
		t.Fatalf("debugger marker visibility = %v, want default", got)
	}
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Windows debug module is invalid: %v\n%s", err, module.String())
	}
	if ir := module.String(); !strings.Contains(ir, `@__llgo_debugger_marker_v1 = linkonce_odr dllexport constant i8 1`) {
		t.Fatalf("Windows debugger marker is not exported:\n%s", ir)
	}
}

func TestBuilderCodeViewRetainsDWARF(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	module := ctx.NewModule("windows-codeview-test")
	defer module.Dispose()
	module.SetTarget("x86_64-pc-windows-msvc")

	builder := New(module, Config{EmitCodeView: true})
	builder.CompileUnit("main.go", "/src/example")
	builder.Finalize()

	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Windows CodeView module is invalid: %v\n%s", err, module.String())
	}
	ir := module.String()
	for _, want := range []string{
		`!{i32 2, !"CodeView", i32 1}`,
		`!{i32 7, !"Dwarf Version", i32 4}`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Windows debug module is missing %q:\n%s", want, ir)
		}
	}
}

func TestStructuredDebuggerRecordTargets(t *testing.T) {
	for _, triple := range []string{"aarch64-apple-darwin", "x86_64-unknown-linux", "aarch64-pc-windows-msvc"} {
		t.Run(triple, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			module := ctx.NewModule("record")
			defer module.Dispose()
			module.SetTarget(triple)
			builder := New(module, Config{DebuggerRecord: debugabi.NewRecord(8, debugabi.ByteOrderLittle)})
			builder.CompileUnit("main.go", "/src")
			builder.Finalize()
			if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			record := module.NamedGlobal(debugabi.NativeRecordSymbol)
			if record.IsNil() || record.Alignment() != debugabi.RecordSize {
				t.Fatal("structured debugger record missing or misaligned")
			}
			if strings.Contains(triple, "windows") && record.DLLStorageClass() != llvm.DLLExportStorageClass {
				t.Fatal("COFF record is not exported")
			}
			if !strings.Contains(module.String(), "[2 x ptr] [ptr @__llgo_debugger_marker_v1, ptr @__llgo_debugger_abi_v1]") {
				t.Fatal("both debugger records must survive code generation")
			}
		})
	}
}

func TestBuilderMetadataOperations(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	module := ctx.NewModule("debug-operations")
	defer module.Dispose()

	builder := New(module, Config{Optimized: true})
	cu := builder.CompileUnit("main.go", "/src/example")
	file := builder.File("/src/example/main.go")
	basic := builder.CreateBasicType(llvm.DIBasicType{
		Name:       "int",
		SizeInBits: 64,
		Encoding:   llvm.DW_ATE_signed,
	})
	pointer := builder.CreatePointerType(llvm.DIPointerType{
		Pointee:     basic,
		SizeInBits:  64,
		AlignInBits: 64,
		Name:        "*int",
	})
	temporary := builder.CreateReplaceableCompositeType(cu, llvm.DIReplaceableCompositeType{
		Tag:         dwarf.TagStructType,
		Name:        "pair",
		File:        file,
		Line:        1,
		SizeInBits:  128,
		AlignInBits: 64,
	})
	member := builder.CreateMemberType(temporary, llvm.DIMemberType{
		Name:        "left",
		File:        file,
		Line:        1,
		SizeInBits:  64,
		AlignInBits: 64,
		Type:        basic,
	})
	structure := builder.CreateStructType(cu, llvm.DIStructType{
		Name:        "pair",
		File:        file,
		Line:        1,
		SizeInBits:  128,
		AlignInBits: 64,
		Elements:    []llvm.Metadata{member},
	})
	temporary.ReplaceAllUsesWith(structure)
	array := builder.CreateArrayType(llvm.DIArrayType{
		SizeInBits:  128,
		AlignInBits: 64,
		ElementType: basic,
		Subscripts:  []llvm.DISubrange{{Count: 2}},
	})
	typedef := builder.CreateTypedef(llvm.DITypedef{
		Type:        array,
		Name:        "ints",
		File:        file,
		Line:        1,
		Context:     cu,
		AlignInBits: 64,
	})
	subroutine := builder.CreateSubroutineType(llvm.DISubroutineType{
		File:       file,
		Parameters: []llvm.Metadata{{}, pointer},
	})
	subprogram := builder.CreateFunction(cu, llvm.DIFunction{
		Name:         "main.main",
		LinkageName:  "main.main",
		File:         file,
		Line:         1,
		ScopeLine:    1,
		Type:         subroutine,
		IsDefinition: true,
	})
	lexical := builder.CreateLexicalBlock(subprogram, llvm.DILexicalBlock{
		File: file,
		Line: 2,
	})
	auto := builder.CreateAutoVariable(lexical, llvm.DIAutoVariable{
		Name:           "local",
		File:           file,
		Line:           2,
		Type:           typedef,
		AlwaysPreserve: true,
	})
	pairAuto := builder.CreateAutoVariable(lexical, llvm.DIAutoVariable{
		Name:           "pair",
		File:           file,
		Line:           2,
		Type:           structure,
		AlwaysPreserve: true,
	})
	parameter := builder.CreateParameterVariable(subprogram, llvm.DIParameterVariable{
		Name:           "param",
		File:           file,
		Line:           1,
		Type:           pointer,
		AlwaysPreserve: true,
		ArgNo:          1,
	})
	expression := builder.CreateExpression(nil)
	globalExpression := builder.CreateGlobalVariableExpression(cu, llvm.DIGlobalVariableExpression{
		Name:        "global",
		LinkageName: "main.global",
		File:        file,
		Line:        1,
		Type:        pointer,
		Expr:        expression,
		AlignInBits: 64,
	})
	global := llvm.AddGlobal(module, ctx.Int64Type(), "main.global")
	global.SetInitializer(llvm.ConstInt(ctx.Int64Type(), 0, false))
	global.AddMetadata(0, globalExpression)

	functionType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{llvm.PointerType(ctx.Int64Type(), 0)}, false)
	function := llvm.AddFunction(module, "main.main", functionType)
	function.SetSubprogram(subprogram)
	irBuilder := ctx.NewBuilder()
	defer irBuilder.Dispose()
	block := llvm.AddBasicBlock(function, "entry")
	irBuilder.SetInsertPointAtEnd(block)
	home := irBuilder.CreateAlloca(ctx.Int64Type(), "local")
	pairType := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int64Type()}, false)
	pairHome := irBuilder.CreateAlloca(pairType, "pair")
	loc := llvm.DebugLoc{Line: 2, Scope: lexical}
	builder.InsertDeclareAtEnd(home, auto, expression, loc, block)
	builder.InsertDeclareAtEnd(pairHome, pairAuto, expression, loc, block)
	builder.InsertValueAtEnd(function.Param(0), parameter, expression, loc, block)
	irBuilder.CreateRetVoid()

	builder.Finalize()
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("metadata operations produced an invalid module: %v\n%s", err, module.String())
	}
	for _, want := range []string{"#dbg_declare", "#dbg_value", `name: "pair"`, `name: "ints"`} {
		if !strings.Contains(module.String(), want) {
			t.Errorf("module is missing %q:\n%s", want, module.String())
		}
	}
}

func TestBuilderRejectsUseAfterFinalize(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	module := ctx.NewModule("debug-finalized")
	defer module.Dispose()

	builder := New(module, Config{})
	builder.CompileUnit("main.go", ".")
	builder.Finalize()
	defer func() {
		if recover() == nil {
			t.Fatal("File after Finalize did not panic")
		}
	}()
	builder.File("main.go")
}
