package wasmresume

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestDescriptorLinksAcrossModules(t *testing.T) {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()

	for _, triple := range []string{"wasm32-unknown-unknown", "wasm64-unknown-unknown"} {
		t.Run(triple, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()

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

			producer := ctx.NewModule("producer")
			producerOwned := true
			defer func() {
				if producerOwned {
					producer.Dispose()
				}
			}()
			configureWasmModule(producer, triple, targetData)

			i32 := ctx.Int32Type()
			sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
			callee := llvm.AddFunction(producer, "example.com/dep.callee", sig)
			callee.SetLinkage(llvm.LinkOnceAnyLinkage)
			markFunction(ctx, callee)
			block := ctx.AddBasicBlock(callee, "entry")
			builder := ctx.NewBuilder()
			defer builder.Dispose()
			builder.SetInsertPointAtEnd(block)
			builder.CreateRet(callee.Param(0))
			abi := newResumeABI(ctx, targetData)
			startType := llvm.FunctionType(
				abi.ptr, []llvm.Type{abi.ptr, i32}, false,
			)
			startDeclaration := llvm.AddFunction(
				producer, startEntryPrefix+callee.Name(), startType,
			)

			if _, err := lowerPrototype(producer, targetData); err != nil {
				t.Fatal(err)
			}
			descriptorName := descriptorPrefix + callee.Name()
			definedDescriptor := producer.NamedGlobal(descriptorName)
			if definedDescriptor.IsNil() || definedDescriptor.Initializer().IsNil() {
				t.Fatal("producer descriptor is not defined")
			}
			if got := definedDescriptor.Linkage(); got != llvm.LinkOnceAnyLinkage {
				t.Fatalf("producer descriptor linkage = %v, want linkonce", got)
			}
			if startDeclaration.IsDeclaration() ||
				startDeclaration.Linkage() != llvm.LinkOnceAnyLinkage {
				t.Fatal("producer did not define its predeclared start entry")
			}

			consumer := ctx.NewModule("consumer")
			defer consumer.Dispose()
			configureWasmModule(consumer, triple, targetData)
			calleeDeclaration := llvm.AddFunction(consumer, callee.Name(), sig)
			markFunction(ctx, calleeDeclaration)
			caller := llvm.AddFunction(consumer, "example.com/main.caller", sig)
			markFunction(ctx, caller)
			block = ctx.AddBasicBlock(caller, "entry")
			builder.SetInsertPointAtEnd(block)
			call := builder.CreateCall(sig, calleeDeclaration, []llvm.Value{caller.Param(0)}, "called")
			markCall(ctx, call)
			builder.CreateRet(call)

			if _, err := lowerPrototype(consumer, targetData); err != nil {
				t.Fatal(err)
			}
			referencedDescriptor := consumer.NamedGlobal(descriptorName)
			if referencedDescriptor.IsNil() || !referencedDescriptor.Initializer().IsNil() {
				t.Fatal("consumer descriptor is not an external declaration")
			}
			requireWasmObject(t, machine, producer)
			requireWasmObject(t, machine, consumer)

			if err := llvm.LinkModules(consumer, producer); err != nil {
				t.Fatal(err)
			}
			producerOwned = false
			linkedDescriptor := consumer.NamedGlobal(descriptorName)
			if linkedDescriptor.IsNil() || linkedDescriptor.Initializer().IsNil() {
				t.Fatal("linked descriptor remains unresolved")
			}
			if got := linkedDescriptor.Linkage(); got != llvm.LinkOnceAnyLinkage {
				t.Fatalf("linked descriptor linkage = %v, want linkonce", got)
			}
			if err := llvm.VerifyModule(consumer, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("verify linked module: %v\n%s", err, consumer.String())
			}
			requireWasmObject(t, machine, consumer)
		})
	}
}

func configureWasmModule(mod llvm.Module, triple string, targetData llvm.TargetData) {
	mod.SetTarget(triple)
	mod.SetDataLayout(targetData.String())
}

func requireWasmObject(t *testing.T, machine llvm.TargetMachine, mod llvm.Module) {
	t.Helper()
	object, err := machine.EmitToMemoryBuffer(mod, llvm.ObjectFile)
	if err != nil {
		t.Fatalf("emit %s: %v\n%s", mod.Target(), err, mod.String())
	}
	defer object.Dispose()
	if data := object.Bytes(); len(data) < 4 || string(data[:4]) != "\x00asm" {
		t.Fatalf("%s object does not have the WebAssembly header", mod.Target())
	}
}
