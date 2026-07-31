//go:build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llvm"
)

func TestClosureEnvABIForTarget(t *testing.T) {
	tests := []struct {
		triple string
		want   closureEnvABI
	}{
		{"wasm32-unknown-wasip1", closureEnvExplicit},
		{"x86_64-unknown-linux", closureEnvNest},
		{"amd64-unknown-linux", closureEnvNest},
		{"riscv64-unknown-linux", closureEnvNest},
		{"armv7-unknown-linux-gnueabihf", closureEnvSwiftSelf},
		{"aarch64-unknown-linux", closureEnvNest},
		{"arm64-apple-macosx", closureEnvSwiftSelf},
		{"x86_64-pc-windows-gnu", closureEnvExplicit},
		{"x86_64-pc-windows-msvc", closureEnvExplicit},
		{"x86_64-w64-mingw32", closureEnvExplicit},
		{"mips64-unknown-linux", closureEnvExplicit},
	}
	for _, test := range tests {
		if got := closureEnvABIForTarget(test.triple); got != test.want {
			t.Errorf(
				"closureEnvABIForTarget(%q) = %d, want %d",
				test.triple, got, test.want,
			)
		}
	}
}

func TestClosureEnvBuildTag(t *testing.T) {
	tests := []struct {
		target *Target
		want   string
	}{
		{&Target{LLVMTarget: "x86_64-unknown-linux"}, "llgo_closure_env_nest"},
		{&Target{LLVMTarget: "arm64-apple-macosx"}, "llgo_closure_env_swiftself"},
		{&Target{GOOS: "linux", GOARCH: "arm", LLVMTarget: "wasm32-unknown-unknown"}, "llgo_closure_env_explicit"},
	}
	for _, test := range tests {
		if got := test.target.ClosureEnvBuildTag(); got != test.want {
			t.Errorf("ClosureEnvBuildTag() = %q, want %q", got, test.want)
		}
	}
}

func TestClosureEnvABIUsesPhysicalTarget(t *testing.T) {
	tests := []struct {
		name       string
		llvmTarget string
		want       closureEnvABI
	}{
		{"esp32", "xtensa", closureEnvExplicit},
		{"esp32c3", "riscv32-esp-elf", closureEnvNest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Embedded target configurations use GOARCH=arm for Go package
			// selection, while code generation uses a different physical ISA.
			prog := NewProgram(&Target{
				GOOS:       "linux",
				GOARCH:     "arm",
				Target:     test.name,
				LLVMTarget: test.llvmTarget,
			})
			defer prog.Dispose()
			if got := prog.closureEnvABI(); got != test.want {
				t.Fatalf("closureEnvABI() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestEnvFunctionKeepsSemanticSignatureAndZeroEnvNonNil(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	empty := types.NewStruct(nil, nil)
	env := types.NewParam(token.NoPos, nil, "$env", types.NewPointer(empty))
	entry := pkg.NewEnvFunc("example.com/p.entry", sig, InGo, env, false)
	if !entry.NeedsEnv() || entry.EnvType() == nil {
		t.Fatal("environment-bearing entry lost its separate env metadata")
	}
	if got := entry.Expr.raw.Type.(*types.Signature).Params().Len(); got != 0 {
		t.Fatalf("semantic entry signature has %d params, want 0", got)
	}
	eb := entry.MakeBody(1)
	eb.Return()

	out := types.NewTuple(types.NewVar(token.NoPos, nil, "", sig))
	makerSig := types.NewSignatureType(nil, nil, nil, nil, out, false)
	maker := pkg.NewFunc("example.com/p.make", makerSig, InGo)
	mb := maker.MakeBody(1)
	mb.Return(mb.MakeClosure(entry.Expr, nil))

	ir := pkg.String()
	if !strings.Contains(ir, `@"__llgo.moduleZeroSizedAlloc$"`) {
		t.Fatalf("zero-sized required env did not use the non-nil sentinel:\n%s", ir)
	}
	if strings.Contains(ir, "{ ptr @example.com/p.entry, ptr null }") {
		t.Fatalf("environment-bearing entry was represented with a nil env:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("closure-env module is invalid: %v\n%s", err, ir)
	}
}

func TestClosureEnvMetadataRejectsInvalidUses(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	env := types.NewParam(token.NoPos, nil, "$env", types.NewPointer(types.NewStruct(nil, nil)))

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "legacy hasFreeVars flag",
			fn: func() {
				pkg.NewFuncEx("legacy", sig, InGo, true, false)
			},
		},
		{
			name: "nil environment metadata",
			fn: func() {
				pkg.NewEnvFunc("nil-env", sig, InGo, nil, false)
			},
		},
		{
			name: "conflicting entry metadata",
			fn: func() {
				pkg.NewFunc("conflict", sig, InGo)
				pkg.NewEnvFunc("conflict", sig, InGo, env, false)
			},
		},
		{
			name: "environment from plain entry",
			fn: func() {
				pkg.NewFunc("plain-env", sig, InGo).Env()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			test.fn()
		})
	}
}

func TestWasmDynamicClosureUsesTwoExplicitTypedEdges(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	prog := NewProgram(&Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")

	params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]))
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	callerParams := types.NewTuple(
		types.NewVar(token.NoPos, nil, "fn", sig),
		types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]),
	)
	caller := pkg.NewFunc(
		"example.com/p.call",
		types.NewSignatureType(nil, nil, nil, callerParams, results, false),
		InGo,
	)
	b := caller.MakeBody(1)
	b.Return(b.Call(caller.Param(0), caller.Param(1)))

	ir := pkg.String()
	for _, want := range []string{
		"icmp ne ptr",
		"phi i32",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Wasm dynamic closure call missing %q:\n%s", want, ir)
		}
	}
	paramCounts := make(map[int]int)
	for block := caller.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue().IsAFunction().IsNil() {
				paramCounts[call.CalledFunctionType().ParamTypesCount()]++
			}
		}
	}
	if paramCounts[1] != 1 || paramCounts[2] != 1 {
		t.Fatalf("Wasm dynamic closure edges have parameter counts %v, want one 1-param and one 2-param call:\n%s", paramCounts, ir)
	}
	if strings.Contains(ir, " nest ") || strings.Contains(ir, " swiftself ") {
		t.Fatalf("Wasm dynamic closure call unexpectedly used a native env attribute:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Wasm dynamic closure module is invalid: %v\n%s", err, ir)
	}
}

func TestNativeDynamicClosureIdentityBarrierSurvivesO2(t *testing.T) {
	for _, pipeline := range []string{"default<O2>", "lto<O2>"} {
		t.Run(pipeline, func(t *testing.T) {
			testNativeDynamicClosureIdentityBarrier(t, pipeline)
		})
	}
}

func testNativeDynamicClosureIdentityBarrier(t *testing.T, pipeline string) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")

	params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]))
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	plain := pkg.NewFunc("plain", sig, InGo)
	pb := plain.MakeBody(1)
	pb.Return(plain.Param(0))
	caller := newMatrixCaller(pkg, "callPlain", sig, func(b Builder) Expr {
		return b.MakeClosure(plain.Expr, nil)
	})

	mod := pkg.Module()
	pbo := llvm.NewPassBuilderOptions()
	defer pbo.Dispose()
	if err := mod.RunPasses(pipeline, prog.TargetMachine(), pbo); err != nil {
		t.Fatalf("run %s pipeline: %v", pipeline, err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify optimized module: %v\n%s", err, mod.String())
	}

	barriers, hiddenCalls := 0, 0
	for block := caller.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			call := instruction.IsACallInst()
			if call.IsNil() {
				if instruction.InstructionOpcode() == llvm.ICmp {
					t.Fatalf("optimized native call retained an env == nil check:\n%s", mod.String())
				}
				continue
			}
			if !call.CalledValue().IsAInlineAsm().IsNil() {
				barriers++
				continue
			}
			if call.GetCallSiteEnumAttribute(1, llvm.AttributeKindID("nest")).IsNil() {
				continue
			}
			hiddenCalls++
			if !call.CalledValue().IsAFunction().IsNil() {
				t.Fatalf("O2 devirtualized hidden-env call to no-env entry:\n%s", mod.String())
			}
		}
	}
	if barriers != 1 || hiddenCalls != 1 {
		t.Fatalf("optimized native call has %d barriers and %d hidden calls, want one each:\n%s", barriers, hiddenCalls, mod.String())
	}
}

func TestClosureObjectCallMatrixAcrossTransports(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs)
	tests := []struct {
		name string
		tgt  *Target
		abi  closureEnvABI
		attr string
	}{
		{
			name: "nest",
			tgt:  &Target{GOOS: "linux", GOARCH: "amd64"},
			abi:  closureEnvNest,
			attr: "nest",
		},
		{
			name: "swiftself",
			tgt:  &Target{GOOS: "darwin", GOARCH: "arm64"},
			abi:  closureEnvSwiftSelf,
			attr: "swiftself",
		},
		{
			name: "wasm-explicit",
			tgt:  &Target{GOOS: "wasip1", GOARCH: "wasm"},
			abi:  closureEnvExplicit,
		},
		{
			name: "xtensa-explicit",
			tgt: &Target{
				GOOS: "linux", GOARCH: "arm", Target: "esp32", LLVMTarget: "xtensa",
			},
			abi: closureEnvExplicit,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := NewProgram(test.tgt)
			defer prog.Dispose()
			if got := prog.closureEnvABI(); got != test.abi {
				t.Fatalf("closure transport = %d, want %d", got, test.abi)
			}
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("p", "example.com/p")

			params := types.NewTuple(types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]))
			results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
			sig := types.NewSignatureType(nil, nil, nil, params, results, false)

			plain := pkg.NewFunc("plainGo", sig, InGo)
			pb := plain.MakeBody(1)
			pb.Return(plain.Param(0))

			cfn := pkg.NewFunc("plainC", sig, InC)
			cb := cfn.MakeBody(1)
			cb.Return(cfn.Param(0))

			captured := newMatrixEnvEntry(pkg, "capturedClosure", sig,
				types.NewStruct(
					[]*types.Var{types.NewField(token.NoPos, nil, "capture", types.Typ[types.Int], false)},
					nil,
				),
			)
			empty := newMatrixEnvEntry(pkg, "emptyClosure", sig, types.NewStruct(nil, nil))
			nilMethod := newMatrixEnvEntry(pkg, "nilReceiverMethodValue", sig,
				types.NewStruct(
					[]*types.Var{types.NewField(
						token.NoPos, nil, "receiver", types.NewPointer(types.Typ[types.Int]), false,
					)},
					nil,
				),
			)

			rawMethod := types.NewFunc(token.NoPos, nil, "M",
				types.NewSignatureType(nil, nil, nil, params, results, false),
			)
			rawIface := types.NewInterfaceType([]*types.Func{rawMethod}, nil).Complete()
			ifaceMethodValue := newMatrixEnvEntry(pkg, "interfaceMethodValue", sig,
				types.NewStruct(
					[]*types.Var{types.NewField(token.NoPos, nil, "receiver", rawIface, false)},
					nil,
				),
			)

			type callCase struct {
				caller Function
				entry  Function
			}
			callers := map[string]callCase{}
			callers["plain-go"] = callCase{
				caller: newMatrixCaller(pkg, "callPlainGo", sig, func(b Builder) Expr {
					return checkExpr(plain.Expr, prog.Type(sig, InGo).RawType(), b)
				}),
				entry: plain,
			}
			callers["plain-c"] = callCase{
				caller: newMatrixCaller(pkg, "callPlainC", sig, func(b Builder) Expr {
					return checkExpr(cfn.Expr, prog.Type(sig, InGo).RawType(), b)
				}),
				entry: cfn,
			}
			callers["captured-closure"] = callCase{
				caller: newMatrixCaller(pkg, "callCapturedClosure", sig, func(b Builder) Expr {
					return b.MakeClosure(captured.Expr, []Expr{prog.Val(7)})
				}),
				entry: captured,
			}
			callers["zero-sized-closure"] = callCase{
				caller: newMatrixCaller(pkg, "callEmptyClosure", sig, func(b Builder) Expr {
					return b.MakeClosure(empty.Expr, nil)
				}),
				entry: empty,
			}
			callers["nil-receiver-method-value"] = callCase{
				caller: newMatrixCaller(pkg, "callNilReceiverMethodValue", sig, func(b Builder) Expr {
					return b.MakeClosure(nilMethod.Expr, []Expr{prog.Nil(prog.Pointer(prog.Int()))})
				}),
				entry: nilMethod,
			}
			callers["interface-method-value"] = callCase{
				caller: newMatrixCaller(pkg, "callInterfaceMethodValue", sig, func(b Builder) Expr {
					return b.MakeClosure(ifaceMethodValue.Expr, []Expr{prog.Zero(prog.Type(rawIface, InGo))})
				}),
				entry: ifaceMethodValue,
			}

			for name, call := range callers {
				assertDynamicMatrixCall(t, call.caller, call.entry, name, test.attr)
			}
			assertMatrixEntryAttribute(t, plain, "", test.attr)
			assertMatrixEntryAttribute(t, cfn, "", test.attr)
			for _, entry := range []Function{captured, empty, nilMethod, ifaceMethodValue} {
				assertMatrixEntryAttribute(t, entry, entry.Name(), test.attr)
			}

			// Direct interface invocation is not a funcval. Its receiver is one
			// ordinary ABI argument, with no env branch or hidden attribute.
			ifaceCaller := newMatrixInterfaceCaller(t, prog, pkg, rawIface)
			assertDirectInterfaceMatrixEdge(t, ifaceCaller, test.attr)
			ifaceRoutine := newMatrixInterfaceRoutine(t, prog, pkg, rawIface)
			assertDirectInterfaceMatrixEdge(t, ifaceRoutine, test.attr)

			ir := pkg.String()
			for _, want := range []string{
				"{ ptr @plainGo, ptr null }",
				"{ ptr @plainC, ptr null }",
				`@"__llgo.moduleZeroSizedAlloc$"`,
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("%s matrix missing %q:\n%s", test.name, want, ir)
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%s matrix module is invalid: %v\n%s", test.name, err, ir)
			}
		})
	}
}

func setTestRuntime(t *testing.T, prog Program) {
	t.Helper()
	fset := token.NewFileSet()
	imp := packages.NewImporter(fset)
	runtimePkg, err := imp.Import(PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	prog.SetRuntime(runtimePkg)
}

func newMatrixEnvEntry(pkg Package, name string, sig *types.Signature, envStruct *types.Struct) Function {
	env := types.NewParam(token.NoPos, nil, "$env", types.NewPointer(envStruct))
	entry := pkg.NewEnvFunc(name, sig, InGo, env, false)
	b := entry.MakeBody(1)
	b.Return(entry.Param(0))
	return entry
}

func newMatrixCaller(pkg Package, name string, sig *types.Signature, value func(Builder) Expr) Function {
	caller := pkg.NewFunc(name, sig, InGo)
	b := caller.MakeBody(1)
	funcval := value(b)
	slot := b.AllocaT(funcval.Type)
	b.Store(slot, funcval)
	b.Return(b.Call(b.Load(slot), caller.Param(0)))
	return caller
}

func assertDynamicMatrixCall(t *testing.T, caller, entry Function, name, attrName string) {
	t.Helper()
	paramCounts := make(map[int]int)
	barrierCalls := 0
	var envCall llvm.Value
	for block := caller.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			call := instruction.IsACallInst()
			if call.IsNil() {
				continue
			}
			called := call.CalledValue()
			if !called.IsAInlineAsm().IsNil() {
				barrierCalls++
				continue
			}
			if !called.IsAFunction().IsNil() && called.Name() != entry.impl.Name() {
				continue
			}
			count := call.CalledFunctionType().ParamTypesCount()
			paramCounts[count]++
			if count == 2 {
				envCall = call
			}
		}
	}
	if attrName == "" {
		if barrierCalls != 0 {
			t.Fatalf("%s explicit call has %d native identity barriers, want none", name, barrierCalls)
		}
		if paramCounts[1] != 1 || paramCounts[2] != 1 {
			t.Fatalf("%s explicit edges have parameter counts %v, want one no-env and one env edge", name, paramCounts)
		}
		entry := caller.impl.FirstBasicBlock()
		if term := entry.LastInstruction(); term.IsNil() || term.InstructionOpcode() != llvm.Br || term.OperandsCount() != 3 {
			t.Fatalf("%s explicit call has no env != nil branch", name)
		}
	} else {
		if barrierCalls != 1 {
			t.Fatalf("%s native call has %d identity barriers, want one", name, barrierCalls)
		}
		if paramCounts[1] != 0 || paramCounts[2] != 1 {
			t.Fatalf("%s native calls have parameter counts %v, want one hidden-env edge", name, paramCounts)
		}
		for block := caller.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
				if instruction.InstructionOpcode() == llvm.ICmp {
					t.Fatalf("%s native call retained an env == nil check", name)
				}
			}
		}
	}
	for _, candidate := range []string{"nest", "swiftself"} {
		kind := llvm.AttributeKindID(candidate)
		got := !envCall.GetCallSiteEnumAttribute(1, kind).IsNil()
		if got != (candidate == attrName) {
			t.Fatalf("%s env edge %s attribute = %v, want %v", name, candidate, got, candidate == attrName)
		}
	}
}

func assertMatrixEntryAttribute(t *testing.T, fn Function, name, attrName string) {
	t.Helper()
	for _, candidate := range []string{"nest", "swiftself"} {
		kind := llvm.AttributeKindID(candidate)
		got := !fn.impl.GetEnumAttributeAtIndex(1, kind).IsNil()
		want := name != "" && candidate == attrName
		if got != want {
			t.Fatalf("%s definition %s attribute = %v, want %v", fn.Name(), candidate, got, want)
		}
	}
}

func newMatrixInterfaceCaller(t *testing.T, prog Program, pkg Package, rawIface *types.Interface) Function {
	t.Helper()
	namedIface := types.NewNamed(types.NewTypeName(token.NoPos, nil, "MatrixIface", nil), rawIface, nil)
	recv := types.NewVar(token.NoPos, nil, "recv", namedIface)
	method := rawIface.Method(0)
	recvMethod := types.NewFunc(
		token.NoPos,
		nil,
		method.Name(),
		types.NewSignatureType(recv, nil, nil, method.Type().(*types.Signature).Params(),
			method.Type().(*types.Signature).Results(), false),
	)
	params := types.NewTuple(
		types.NewVar(token.NoPos, nil, "recv", namedIface),
		types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]),
	)
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
	caller := pkg.NewFunc("callInterfaceDirect",
		types.NewSignatureType(nil, nil, nil, params, results, false), InGo,
	)
	b := caller.MakeBody(1)
	b.Return(b.Call(b.Imethod(caller.Param(0), recvMethod), caller.Param(1)))
	return caller
}

func newMatrixInterfaceRoutine(t *testing.T, prog Program, pkg Package, rawIface *types.Interface) Function {
	t.Helper()
	namedIface := types.NewNamed(types.NewTypeName(token.NoPos, nil, "RoutineIface", nil), rawIface, nil)
	recv := types.NewVar(token.NoPos, nil, "recv", namedIface)
	method := rawIface.Method(0)
	recvMethod := types.NewFunc(
		token.NoPos,
		nil,
		method.Name(),
		types.NewSignatureType(recv, nil, nil, method.Type().(*types.Signature).Params(),
			method.Type().(*types.Signature).Results(), false),
	)
	params := types.NewTuple(
		types.NewVar(token.NoPos, nil, "recv", namedIface),
		types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int]),
	)
	owner := pkg.NewFunc("startInterfaceRoutine",
		types.NewSignatureType(nil, nil, nil, params, nil, false), InGo,
	)
	b := owner.MakeBody(1)
	invoke := b.Imethod(owner.Param(0), recvMethod)
	startRecord := prog.Struct(invoke.Type, prog.Int())
	routineExpr := pkg.routine(startRecord, invoke, Builder.Call, 1)
	b.Return()
	return pkg.FuncOf(routineExpr.Name())
}

func assertDirectInterfaceMatrixEdge(t *testing.T, caller Function, attrName string) {
	t.Helper()
	var indirect []llvm.Value
	for block := caller.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			call := instruction.IsACallInst()
			if !call.IsNil() && call.CalledValue().IsAFunction().IsNil() {
				indirect = append(indirect, call)
			}
		}
	}
	if len(indirect) != 1 || indirect[0].CalledFunctionType().ParamTypesCount() != 2 {
		t.Fatalf("direct interface invocation has %d indirect edges, want one receiver+arg edge", len(indirect))
	}
	for _, candidate := range []string{"nest", "swiftself"} {
		kind := llvm.AttributeKindID(candidate)
		if attr := indirect[0].GetCallSiteEnumAttribute(1, kind); !attr.IsNil() {
			t.Fatalf("direct interface invocation incorrectly carries %s (transport %q)", candidate, attrName)
		}
	}
}
