package plan9asm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	llvm "github.com/xgo-dev/llvm"
	asm "github.com/xgo-dev/plan9asm"
)

var nativeTailLabels = regexp.MustCompile(`\bLnative_(?:func_0|0_label_[0-9]+)\b`)
var nativeTailPseudoSP = regexp.MustCompile(`\bSP\b`)

// lowerNativeTailTransfers preserves physical register and stack semantics at
// an indirect assembly tail transfer. The typed translator's virtual registers
// cannot describe code that changes SP and returns through a saved native LR.
// Keep a normal LLVM entry/epilogue around one indivisible native asm region:
// LLVM saves its callee-saved registers, and the region supplies the ABI0 frame.
// Guest code must restore the entry FP/SP before returning through the saved LR.
func lowerNativeTailTransfers(mod llvm.Module, src []byte, sigs map[string]asm.FuncSig, resolve func(string) string, goos, goarch string, bdwgc bool) error {
	if goarch != "arm64" || !asm.SupportsNativeTarget(goos, goarch) {
		return nil
	}
	// GoModuleTranslation exposes names and signatures, but no parsed File.
	// Reparse the small assembly input for its physical instruction body.
	file, err := asm.Parse(asm.ArchARM64, string(src))
	if err != nil {
		return err
	}
	for _, fn := range file.Funcs {
		if !hasIndirectTailTransfer(fn.Instrs) {
			continue
		}
		name := resolve(fn.Sym)
		// The Go binder can deliberately filter TEXT bodies (for example a
		// syscall entry supplied by the runtime). Do not lower those again.
		entry := mod.NamedFunction(name)
		if entry.IsNil() || entry.IsDeclaration() {
			continue
		}
		sig, ok := sigs[name]
		if !ok {
			return fmt.Errorf("native tail transfer %s: missing Go declaration", name)
		}
		if err := lowerNativeTailTransfer(mod, fn, sig, goos); err != nil {
			return fmt.Errorf("native tail transfer %s: %w", name, err)
		}
		if bdwgc {
			wrapNativeTailBDWGC(mod, mod.NamedFunction(sig.Name))
		}
	}
	return nil
}

// wrapNativeTailBDWGC snapshots the native stack before foreign code can switch
// SP. BDWGC must scan that saved stack, rather than the interval from a guest
// heap stack to the pthread stack top. All arguments stay rooted in the active
// caller frame; the foreign region performs no managed Go calls/allocations.
func wrapNativeTailBDWGC(mod llvm.Module, native llvm.Value) {
	ctx := mod.Context()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	name := native.Name()
	native.SetName(name + "$native")
	visibility, linkage := native.Visibility(), native.Linkage()
	native.SetVisibility(llvm.DefaultVisibility)
	native.SetLinkage(llvm.InternalLinkage)
	entry := llvm.AddFunction(mod, name, native.GlobalValueType())
	entry.SetVisibility(visibility)
	entry.SetLinkage(linkage)
	entry.SetFunctionCallConv(native.FunctionCallConv())
	native.ReplaceAllUsesWith(entry)
	var params []llvm.Type
	for i := 0; i < native.ParamsCount(); i++ {
		params = append(params, native.Param(i).Type())
	}
	dataType := ctx.StructType(params, false)
	callbackType := llvm.FunctionType(ptr, []llvm.Type{ptr}, false)
	callback := llvm.AddFunction(mod, name+"$blocked", callbackType)
	callback.SetLinkage(llvm.InternalLinkage)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(callback, "entry"))
	var args []llvm.Value
	for i, typ := range params {
		field := b.CreateStructGEP(dataType, callback.Param(0), i, "")
		args = append(args, b.CreateLoad(typ, field, ""))
	}
	call := b.CreateCall(native.GlobalValueType(), native, args, "")
	call.SetInstructionCallConv(native.FunctionCallConv())
	// The GC transition changes function effects; forward only the return
	// and parameter contracts to the public entry and the native call.
	for i := 0; i <= len(params); i++ {
		for _, attr := range native.GetAttributesAtIndex(i) {
			entry.AddAttributeAtIndex(i, attr)
			call.AddCallSiteAttribute(i, attr)
		}
	}
	b.CreateRet(llvm.ConstNull(ptr))

	b.SetInsertPointAtEnd(ctx.AddBasicBlock(entry, "entry"))
	data := b.CreateAlloca(dataType, "native.args")
	for i := range params {
		field := b.CreateStructGEP(dataType, data, i, "")
		b.CreateStore(entry.Param(i), field)
	}
	blockingType := llvm.FunctionType(ptr, []llvm.Type{ptr, ptr}, false)
	blocking := mod.NamedFunction("GC_do_blocking")
	if blocking.IsNil() {
		blocking = llvm.AddFunction(mod, "GC_do_blocking", blockingType)
	}
	b.CreateCall(blockingType, blocking, []llvm.Value{callback, data}, "")
	b.CreateRetVoid()
}

func hasIndirectTailTransfer(instrs []asm.Instr) bool {
	for _, ins := range instrs {
		if (ins.Op == "JMP" || ins.Op == "B" || ins.Op == "BR") && len(ins.Args) == 1 {
			arg := ins.Args[0]
			if arg.Kind == asm.OpReg || (arg.Kind == asm.OpMem && arg.Mem.Base != "" && arg.Mem.Base != "PC") {
				return true
			}
		}
	}
	return false
}

func lowerNativeTailTransfer(mod llvm.Module, fn asm.Func, sig asm.FuncSig, goos string) error {
	last := fn.Instrs[len(fn.Instrs)-1]
	if (last.Op != "JMP" && last.Op != "B" && last.Op != "BR") || len(last.Args) != 1 {
		return fmt.Errorf("requires a terminal indirect branch")
	}
	if hasIndirectTailTransfer(fn.Instrs[:len(fn.Instrs)-1]) {
		return fmt.Errorf("requires a single terminal indirect branch")
	}
	target := last.Args[0].Reg
	if last.Args[0].Kind == asm.OpMem && last.Args[0].Mem.Off == 0 && last.Args[0].Mem.Index == "" {
		target = last.Args[0].Mem.Base
	}
	reg, err := strconv.Atoi(strings.TrimPrefix(string(target), "R"))
	if err != nil || reg < 0 || reg > 28 || reg == 18 {
		return fmt.Errorf("unsupported tail register %s", target)
	}
	argc := len(sig.Args)
	if fn.FrameSize != 0 || fn.ArgSize != int64(argc*8) || argc > 8 || sig.Ret != asm.Void || len(sig.Frame.Results) != 0 || len(sig.Frame.Params) != argc || len(sig.ArgRegs) != 0 {
		return fmt.Errorf("requires a zero-frame declaration with up to eight word arguments and no results")
	}
	flags := strings.Split(fn.Instrs[0].Raw, ",")
	if len(flags) != 3 {
		return fmt.Errorf("malformed TEXT directive: %s", fn.Instrs[0].Raw)
	}
	seenFlags := map[string]bool{}
	for _, flag := range strings.Split(flags[1], "|") {
		flag = strings.TrimSpace(flag)
		if flag != "NOSPLIT" && flag != "NOFRAME" {
			return fmt.Errorf("unsupported TEXT flag %q", flag)
		}
		seenFlags[flag] = true
	}
	if !seenFlags["NOSPLIT"] || !seenFlags["NOFRAME"] {
		return fmt.Errorf("requires NOSPLIT|NOFRAME")
	}
	for i, param := range sig.Frame.Params {
		if (sig.Args[i] != asm.I64 && sig.Args[i] != asm.Ptr) || param.Offset != int64(i*8) || param.Index != i || param.Field >= 0 || param.Type != sig.Args[i] {
			return fmt.Errorf("argument %d requires a scalar word or pointer slot", i)
		}
	}

	var source strings.Builder
	source.WriteString("TEXT bridge<>(SB), NOSPLIT|NOFRAME, $0-0\n")
	stackTouched := false
	for _, ins := range fn.Instrs[1 : len(fn.Instrs)-1] {
		if nativeTailPseudoSP.MatchString(ins.Raw) {
			return fmt.Errorf("unsupported pseudo SP operand: %s", ins.Raw)
		}
		if ins.Op == asm.OpLABEL {
			fmt.Fprintf(&source, "%s:\n", ins.Args[0].Sym)
			continue
		}
		args := make([]string, len(ins.Args))
		for i, arg := range ins.Args {
			if arg.Kind == asm.OpFP {
				// ABI0 reserves the first word for LR. Only input loads before
				// physical stack use are supported by this bounded bridge.
				if ins.Op != "MOVD" || i != 0 || stackTouched || arg.FPOffset < 0 || arg.FPOffset%8 != 0 || arg.FPOffset >= int64(argc*8) {
					return fmt.Errorf("unsupported ABI0 frame access: %s", ins.Raw)
				}
				args[i] = fmt.Sprintf("%d(RSP)", arg.FPOffset+8)
			} else {
				args[i] = arg.String()
				if arg.Kind == asm.OpReg && arg.Reg == "SP" {
					args[i] = "RSP"
				}
				args[i] = strings.ReplaceAll(args[i], "(SP)", "(RSP)")
			}
			if arg.Kind == asm.OpReg && arg.Reg == "SP" || arg.Kind == asm.OpMem && arg.Mem.Base == "SP" {
				stackTouched = true
			}
		}
		fmt.Fprintf(&source, "%s %s\n", ins.Op, strings.Join(args, ", "))
	}
	code, _, err := asm.TranslateNativeSource([]byte(source.String()), asm.NativeOptions{GOOS: goos, GOARCH: "arm64"})
	if err != nil {
		return err
	}
	// This region is embedded in a function's current text section. Give its
	// local labels an LLVM instance ID so cloning/inlining cannot duplicate them.
	var body strings.Builder
	for _, line := range strings.Split(code, "\n") {
		if line == ".text" || strings.HasPrefix(line, ".section .note.GNU-stack") || strings.HasPrefix(line, ".type ") || strings.HasPrefix(line, ".size ") {
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	// The native emitter handles the checked instruction body. The terminal
	// transfer is the bridge boundary itself and deliberately remains physical.
	fmt.Fprintf(&body, "br x%d\n", reg)
	assembly := strings.ReplaceAll(body.String(), "$", "$$")
	// NativeSource uses only these two generated label families for a single
	// function (entry plus ordinary source labels).
	assembly = nativeTailLabels.ReplaceAllStringFunc(assembly, func(label string) string {
		return label + "${:uid}"
	})

	entry := mod.NamedFunction(sig.Name)
	if entry.IsNil() || entry.ParamsCount() != argc {
		return fmt.Errorf("missing typed LLVM entry")
	}
	// Replace the virtual body as a unit: deleting its basic blocks one by
	// one would leave branch/PHI references into already-deleted blocks.
	old := entry
	old.SetName(sig.Name + ".virtual")
	entry = llvm.AddFunction(mod, sig.Name, old.GlobalValueType())
	entry.SetLinkage(old.Linkage())
	entry.SetVisibility(old.Visibility())
	entry.SetFunctionCallConv(old.FunctionCallConv())
	for i := -1; i <= argc; i++ {
		for _, attr := range old.GetAttributesAtIndex(i) {
			entry.AddAttributeAtIndex(i, attr)
		}
	}
	old.ReplaceAllUsesWith(entry)
	old.EraseFromParentAsFunction()
	ctx := mod.Context()
	// A caller with dynamic/overaligned allocas can reserve x19 as its frame
	// base. Even an explicit x19 clobber cannot protect it after inlining.
	// Keep this physical ABI boundary in its own ordinary LLVM function.
	entry.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(entry, "native.tail"))
	var inputs []llvm.Value
	var words []llvm.Type
	var constraints []string
	for i := 0; i < argc; i++ {
		v := entry.Param(i)
		if v.Type().TypeKind() == llvm.PointerTypeKind {
			v = b.CreatePtrToInt(v, ctx.Int64Type(), "")
		}
		inputs = append(inputs, v)
		words = append(words, ctx.Int64Type())
		constraints = append(constraints, fmt.Sprintf("={x%d}", i))
	}
	for i := 0; i < argc; i++ {
		constraints = append(constraints, fmt.Sprint(i))
	}
	// Inputs are also outputs: the guest may overwrite them. Darwin reserves
	// x18; Linux permits guest clobbers. FP/SP must be restored by the guest.
	for i := argc; i <= 28; i++ {
		if i != 18 || goos != "darwin" {
			constraints = append(constraints, fmt.Sprintf("~{x%d}", i))
		}
	}
	constraints = append(constraints, "~{lr}")
	for i := 0; i < 32; i++ {
		constraints = append(constraints, fmt.Sprintf("~{v%d}", i))
	}
	constraints = append(constraints, "~{cc}", "~{memory}")
	// BL gives the raw body a return continuation inside this asm region.
	// Its saved LR therefore reaches stack cleanup before LLVM's epilogue,
	// including after a guest stack switch or host-call resume.
	frame := (8 + argc*8 + 15) &^ 15
	var prefix strings.Builder
	fmt.Fprintf(&prefix, "sub sp, sp, #%d\n", frame)
	for i := 0; i < argc; i++ {
		fmt.Fprintf(&prefix, "str x%d, [sp, #%d]\n", i, 8+i*8)
	}
	fmt.Fprintf(&prefix, "bl Lnative_func_0${:uid}\nadd sp, sp, #%d\nb Ltail_done${:uid}\n", frame)
	assembly = prefix.String() + assembly + "\nLtail_done${:uid}:\n"
	result := ctx.VoidType()
	if argc == 1 {
		result = ctx.Int64Type()
	} else if argc > 1 {
		result = ctx.StructType(words, false)
	}
	ty := llvm.FunctionType(result, words, false)
	inline := llvm.InlineAsm(ty, assembly, strings.Join(constraints, ","), true, false, llvm.InlineAsmDialectATT, false)
	b.CreateCall(ty, inline, inputs, "")
	b.CreateRetVoid()
	return nil
}
