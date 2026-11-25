// Package debuginfo owns the LLVM debug metadata builder and its lifecycle.
// Source-language type layout and SSA value tracking belong to the caller.
package debuginfo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/debugabi"
	"github.com/xgo-dev/llvm"
)

const (
	defaultDebugInfoVersion = 3
	defaultDwarfVersion     = 4
	// Stock LLDB has no maintained Go language plugin and limits frame-variable
	// inspection for DW_LANG_Go. Use its C language path for interoperable
	// access to LLGo's otherwise Go-shaped DWARF.
	dwarfSourceLanguageC llvm.DwarfLang = 1

	debuggerMarkerSymbol = debugabi.LegacyMarkerSymbol
)

// Config describes properties of the generated debug information. Optimized
// reports what the compilation pipeline does; it does not select any pass.
type Config struct {
	Producer       string
	Optimized      bool
	EmitCodeView   bool
	DebuggerRecord debugabi.Record
}

// Builder is a package-local owner of an LLVM DIBuilder. Finalize must be
// called before the module is verified, optimized, serialized, or disposed.
type Builder struct {
	impl      *llvm.DIBuilder
	module    llvm.Module
	config    Config
	files     map[string]llvm.Metadata
	finalized bool
}

// New creates a debug metadata builder and installs the module metadata LLVM
// requires for DWARF emission.
func New(module llvm.Module, config Config) *Builder {
	if config.Producer == "" {
		config.Producer = "LLGo"
	}
	b := &Builder{
		impl:   llvm.NewDIBuilder(module),
		module: module,
		config: config,
		files:  make(map[string]llvm.Metadata),
	}
	b.addModuleFlag(2, "Debug Info Version", defaultDebugInfoVersion)
	b.addModuleFlag(7, "Dwarf Version", defaultDwarfVersion)
	if config.EmitCodeView {
		// Keep DWARF enabled as well. LLVM can lower the same source metadata to
		// both formats: DWARF remains available to LLDB and Go-compatible
		// traceback consumers, while lld-link can merge CodeView into a PDB.
		b.addModuleFlag(2, "CodeView", 1)
	}
	// Do not declare wchar_size: Go has no wchar_t, and this C/C++ ABI
	// compatibility flag is not required for debug information. Let linked
	// C/C++ modules provide it; their value depends on the target and flags
	// such as -fshort-wchar, and imposing one here can reject valid LTO links.
	b.addModuleFlag(8, "PIC Level", 2)
	b.addModuleFlag(7, "uwtable", 1)
	b.addModuleFlag(7, "frame-pointer", 1)
	b.addDebuggerMarker()

	ctx := module.Context()
	module.AddNamedMetadataOperand("llvm.ident", ctx.MDNode([]llvm.Metadata{
		ctx.MDString(config.Producer + " Compiler"),
	}))
	return b
}

func (b *Builder) addDebuggerMarker() {
	ctx := b.module.Context()
	i8 := ctx.Int8Type()
	marker := llvm.AddGlobal(b.module, i8, debuggerMarkerSymbol)
	marker.SetInitializer(llvm.ConstInt(i8, 1, false))
	b.retainDebuggerGlobal(marker, debuggerMarkerSymbol)

	ptr := llvm.PointerType(i8, 0)
	retained := []llvm.Value{llvm.ConstBitCast(marker, ptr)}
	if b.config.DebuggerRecord.RecordVersion != 0 {
		raw, err := b.config.DebuggerRecord.MarshalBinary()
		if err != nil {
			panic(fmt.Sprintf("debuginfo: invalid debugger ABI record: %v", err))
		}
		values := make([]llvm.Value, len(raw))
		for i, value := range raw {
			values[i] = llvm.ConstInt(i8, uint64(value), false)
		}
		initializer := llvm.ConstArray(i8, values)
		record := llvm.AddGlobal(b.module, initializer.Type(), debugabi.NativeRecordSymbol)
		record.SetInitializer(initializer)
		record.SetAlignment(len(raw))
		b.retainDebuggerGlobal(record, debugabi.NativeRecordSymbol)
		retained = append(retained, llvm.ConstBitCast(record, ptr))
	}
	usedInit := llvm.ConstArray(ptr, retained)
	used := llvm.AddGlobal(b.module, usedInit.Type(), "llvm.used")
	used.SetInitializer(usedInit)
	used.SetLinkage(llvm.AppendingLinkage)
	used.SetSection("llvm.metadata")
}

// Give both marker records the same target-specific retention semantics.
func (b *Builder) retainDebuggerGlobal(value llvm.Value, symbol string) {
	value.SetGlobalConstant(true)
	value.SetLinkage(llvm.LinkOnceODRLinkage)
	if strings.Contains(strings.ToLower(b.module.Target()), "windows") {
		comdat := b.module.Comdat(symbol)
		comdat.SetSelectionKind(llvm.AnyComdatSelectionKind)
		value.SetComdat(comdat)
		value.SetDLLStorageClass(llvm.DLLExportStorageClass)
	} else {
		value.SetVisibility(llvm.HiddenVisibility)
	}
}

func (b *Builder) addModuleFlag(behavior int, name string, value int) {
	ctx := b.module.Context()
	b.module.AddNamedMetadataOperand("llvm.module.flags", ctx.MDNode([]llvm.Metadata{
		llvm.ConstInt(ctx.Int32Type(), uint64(behavior), false).ConstantAsMetadata(),
		ctx.MDString(name),
		llvm.ConstInt(ctx.Int32Type(), uint64(value), false).ConstantAsMetadata(),
	}))
}

func (b *Builder) checkOpen() {
	if b == nil || b.impl == nil || b.finalized {
		panic("debuginfo: use after Finalize")
	}
}

// Finalize resolves temporary metadata and releases the underlying DIBuilder.
// It is idempotent so callers can safely defer it across error paths.
func (b *Builder) Finalize() {
	if b == nil || b.finalized {
		return
	}
	b.impl.Finalize()
	b.impl.Destroy()
	b.impl = nil
	b.finalized = true
}

// CompileUnit creates this module's compile unit.
func (b *Builder) CompileUnit(filename, dir string) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateCompileUnit(llvm.DICompileUnit{
		Language:       dwarfSourceLanguageC,
		File:           filename,
		Dir:            dir,
		Producer:       b.config.Producer,
		Optimized:      b.config.Optimized,
		RuntimeVersion: 1,
	})
}

// File returns one metadata node per cleaned source path.
func (b *Builder) File(filename string) llvm.Metadata {
	b.checkOpen()
	key := filename
	if key != "" {
		key = filepath.Clean(key)
	}
	if file, ok := b.files[key]; ok {
		return file
	}
	dir, file := filepath.Split(key)
	metadata := b.impl.CreateFile(file, dir)
	b.files[key] = metadata
	return metadata
}

func (b *Builder) CreateLexicalBlock(scope llvm.Metadata, desc llvm.DILexicalBlock) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateLexicalBlock(scope, desc)
}

func (b *Builder) CreateFunction(scope llvm.Metadata, desc llvm.DIFunction) llvm.Metadata {
	b.checkOpen()
	desc.Optimized = b.config.Optimized
	return b.impl.CreateFunction(scope, desc)
}

func (b *Builder) CreateGlobalVariableExpression(scope llvm.Metadata, desc llvm.DIGlobalVariableExpression) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateGlobalVariableExpression(scope, desc)
}

func (b *Builder) CreateAutoVariable(scope llvm.Metadata, desc llvm.DIAutoVariable) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateAutoVariable(scope, desc)
}

func (b *Builder) CreateParameterVariable(scope llvm.Metadata, desc llvm.DIParameterVariable) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateParameterVariable(scope, desc)
}

func (b *Builder) CreateBasicType(desc llvm.DIBasicType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateBasicType(desc)
}

func (b *Builder) CreatePointerType(desc llvm.DIPointerType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreatePointerType(desc)
}

func (b *Builder) CreateSubroutineType(desc llvm.DISubroutineType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateSubroutineType(desc)
}

func (b *Builder) CreateStructType(scope llvm.Metadata, desc llvm.DIStructType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateStructType(scope, desc)
}

func (b *Builder) CreateReplaceableCompositeType(scope llvm.Metadata, desc llvm.DIReplaceableCompositeType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateReplaceableCompositeType(scope, desc)
}

func (b *Builder) CreateMemberType(scope llvm.Metadata, desc llvm.DIMemberType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateMemberType(scope, desc)
}

func (b *Builder) CreateArrayType(desc llvm.DIArrayType) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateArrayType(desc)
}

func (b *Builder) CreateTypedef(desc llvm.DITypedef) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateTypedef(desc)
}

func (b *Builder) CreateExpression(ops []uint64) llvm.Metadata {
	b.checkOpen()
	return b.impl.CreateExpression(ops)
}

func (b *Builder) InsertDeclareAtEnd(value llvm.Value, variable, expression llvm.Metadata, loc llvm.DebugLoc, block llvm.BasicBlock) {
	b.checkOpen()
	b.impl.InsertDeclareAtEnd(value, variable, expression, loc, block)
}

func (b *Builder) InsertValueAtEnd(value llvm.Value, variable, expression llvm.Metadata, loc llvm.DebugLoc, block llvm.BasicBlock) {
	b.checkOpen()
	b.impl.InsertValueAtEnd(value, variable, expression, loc, block)
}
