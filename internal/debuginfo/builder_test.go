package debuginfo

import (
	"debug/dwarf"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

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
	if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Windows debug module is invalid: %v\n%s", err, module.String())
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
