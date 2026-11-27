package callerlocation

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestTracker(t *testing.T) {
	for _, kind := range []string{"RecordPanicLocation", "RecordCallerLocation"} {
		for _, change := range []string{"none", "load", "store", "call", "indirect", "asm", "block", "kind", "function", "file", "line", "column", "scope", "inline", "missing debug"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				ctx := llvm.NewContext()
				defer ctx.Dispose()
				mod := ctx.NewModule("records")
				defer mod.Dispose()
				b := ctx.NewBuilder()
				defer b.Dispose()
				sig := llvm.FunctionType(ctx.VoidType(), nil, false)
				callee := llvm.AddFunction(mod, "record", sig)
				fn := llvm.AddFunction(mod, "f", llvm.FunctionType(ctx.VoidType(), []llvm.Type{llvm.PointerType(ctx.Int32Type(), 0)}, false))
				block := ctx.AddBasicBlock(fn, "entry")
				b.SetInsertPointAtEnd(block)
				location := Location{Kind: kind, Function: "f", File: "f.go", Line: 10}
				debug := llvm.DebugLoc{Line: 10, Col: 2}
				var tracker Tracker
				if tracker.Repeated(block, block.LastInstruction(), location, debug) {
					t.Fatal("first record was skipped")
				}
				tracker.Checkpoint(b.CreateCall(sig, callee, nil, ""))
				switch change {
				case "load":
					b.CreateLoad(ctx.Int32Type(), fn.Param(0), "")
				case "store":
					b.CreateStore(llvm.ConstInt(ctx.Int32Type(), 7, false), fn.Param(0))
				case "call":
					b.CreateCall(sig, llvm.AddFunction(mod, "ordinary", sig), nil, "")
				case "indirect":
					b.CreateCall(sig, fn.Param(0), nil, "")
				case "asm":
					b.CreateCall(sig, llvm.InlineAsm(sig, "", "", true, false, llvm.InlineAsmDialectATT, false), nil, "")
				case "block":
					block = ctx.AddBasicBlock(fn, "next")
					b.CreateBr(block)
					b.SetInsertPointAtEnd(block)
				case "kind":
					location.Kind = "other"
				case "function":
					location.Function = "other"
				case "file":
					location.File = "other.go"
				case "line":
					location.Line++
				case "column":
					debug.Col++
				case "scope":
					debug.Scope = ctx.MDString("scope")
				case "inline":
					debug.InlinedAt = ctx.MDString("inlined")
				case "missing debug":
					debug = llvm.DebugLoc{}
				}
				want := change == "none" || change == "load" || change == "store"
				if got := tracker.Repeated(block, block.LastInstruction(), location, debug); got != want {
					t.Fatalf("Repeated = %v, want %v", got, want)
				}
				b.CreateRetVoid()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestTrackerAdvancesAfterSkippedRecord(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("records")
	defer mod.Dispose()
	b := ctx.NewBuilder()
	defer b.Dispose()
	sig := llvm.FunctionType(ctx.VoidType(), nil, false)
	callee := llvm.AddFunction(mod, "record", sig)
	fn := llvm.AddFunction(mod, "f", sig)
	block := ctx.AddBasicBlock(fn, "entry")
	b.SetInsertPointAtEnd(block)
	ptr := b.CreateAlloca(ctx.Int32Type(), "")
	location := Location{Kind: "panic", File: "f.go", Line: 1}
	var tracker Tracker
	tracker.Repeated(block, block.LastInstruction(), location, llvm.DebugLoc{})
	tracker.Checkpoint(b.CreateCall(sig, callee, nil, ""))
	for range 4096 {
		last := b.CreateLoad(ctx.Int32Type(), ptr, "")
		if !tracker.Repeated(block, last, location, llvm.DebugLoc{}) {
			t.Fatal("redundant record was not skipped")
		}
		if tracker.cursor != last {
			t.Fatal("skipped record did not advance checkpoint; later checks would rescan the prefix")
		}
	}
	// A moved/deleted checkpoint must not let a new prefix inherit its record.
	tracker.Checkpoint(b.CreateLoad(ctx.Int32Type(), ptr, ""))
	tracker.cursor.EraseFromParentAsInstruction()
	if tracker.Repeated(block, ptr, location, llvm.DebugLoc{}) {
		t.Fatal("missing checkpoint was reused")
	}
	b.CreateRetVoid()
	if tracker.Repeated(block, block.LastInstruction(), location, llvm.DebugLoc{}) {
		t.Fatal("terminator was crossed")
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
