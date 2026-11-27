package llvmgen

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLocationRecordInsertionPoint(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("foo.f", NoArgsNoRet, GoDecl)
	b := fn.MakeBody(2)
	defer b.Dispose()
	callee := pkg.NewFunc("record", NoArgsNoRet, GoDecl)
	emitted := 0
	record := func() {
		b.EmitLocationRecord("panic", "foo.f", "foo.go", 1, func() Expr {
			emitted++
			return b.Call(callee.Expr)
		})
	}
	record()
	record()
	if emitted != 1 {
		t.Fatal("tail duplicate was not skipped")
	}
	b.Jump(fn.Block(1))
	b.SetBlock(fn.Block(1))
	record()
	b.Return()
	// Phi lowering and other rewrites can revisit an already terminated block.
	// Do not reuse the remembered record when inserting before its terminator.
	b.SetBlockEx(fn.Block(1), BeforeLast, true)
	record()
	record()
	if emitted != 4 {
		t.Fatalf("emitted %d records, want 4", emitted)
	}
	b.EndBuild()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestLocationRecordAfterClosureContext(t *testing.T) {
	for _, synthetic := range []bool{false, true} {
		name := "block"
		if synthetic {
			name = "synthetic block"
		}
		t.Run(name, func(t *testing.T) {
			prog := NewProgram(nil)
			defer prog.Dispose()
			pkg := prog.NewPackage("foo", "foo")
			fields := []*types.Var{types.NewField(0, nil, "x", types.Typ[types.Int], false)}
			env := types.NewParam(0, nil, "$env", types.NewPointer(types.NewStruct(fields, nil)))
			fn := pkg.NewEnvFunc("foo.f", NoArgsNoRet, GoDecl, env, false)
			b := fn.MakeBody(2)
			defer b.Dispose()
			b.Jump(fn.Block(1))
			b.SetBlock(fn.Block(1))
			callee := pkg.NewFunc("record", NoArgsNoRet, GoDecl)
			check := func() {
				block := b.impl.GetInsertBlock()
				fn.FreeVar(b, 0) // Hoists the first environment load to the entry block.
				if b.impl.GetInsertBlock() != block {
					t.Fatal("closure context changed the insertion block")
				}
				emitted := 0
				for range 2 {
					b.EmitLocationRecord("panic", "foo.f", "foo.go", 1, func() Expr {
						emitted++
						return b.Call(callee.Expr)
					})
				}
				if emitted != 1 {
					t.Fatalf("emitted %d records after closure context load, want 1", emitted)
				}
			}
			if synthetic {
				b.IfThen(prog.BoolVal(true), check)
			} else {
				check()
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLocationRecordDebugPositions(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("foo", "foo")
	pkg.InitDebug("foo", "foo", token.NewFileSet())
	fn := pkg.NewFunc("foo.f", NoArgsNoRet, GoDecl)
	b := fn.MakeBody(1)
	defer b.Dispose()
	pos := token.Position{Filename: "foo.go", Line: 10, Column: 2}
	b.DebugFunction(fn, nil, pos, pos)
	callee := pkg.NewFunc("record", NoArgsNoRet, GoDecl)
	emitted := 0
	record := func() {
		b.EmitLocationRecord("panic", "foo.f", pos.Filename, pos.Line, func() Expr {
			emitted++
			return b.Call(callee.Expr)
		})
	}
	record()
	record()
	loc := b.diLocation
	loc.Col++
	b.setDebugLocation(loc)
	record()
	if emitted != 2 {
		t.Fatalf("emitted %d records, want two distinct debugger positions", emitted)
	}
	b.Return()
	b.EndBuild()
	pkg.FinalizeDebug()
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestLocationRecordCallbackBarrier(t *testing.T) {
	for _, change := range []string{"call", "split before", "split after"} {
		t.Run(change, func(t *testing.T) {
			prog := NewProgram(nil)
			defer prog.Dispose()
			pkg := prog.NewPackage("foo", "foo")
			fn := pkg.NewFunc("foo.f", NoArgsNoRet, GoDecl)
			b := fn.MakeBody(1)
			defer b.Dispose()
			record := pkg.NewFunc("record", NoArgsNoRet, GoDecl)
			helper := pkg.NewFunc("helper", NoArgsNoRet, GoDecl)
			split := func() {
				next := fn.MakeBlock()
				b.Jump(next)
				b.SetBlock(next)
			}
			emitted := 0
			for range 3 {
				b.EmitLocationRecord("panic", "foo.f", "foo.go", 1, func() Expr {
					if emitted == 0 && change == "split before" {
						split()
					}
					update := b.Call(record.Expr)
					// Calls and block changes inside the callback must remain barriers.
					if change == "call" {
						b.Call(helper.Expr)
					} else if emitted == 0 && change == "split after" {
						split()
					}
					emitted++
					return update
				})
			}
			want := 2 // A block change resets history; the third update is redundant.
			if change == "call" {
				want = 3 // Every update is followed by a call barrier.
			}
			if emitted != want {
				t.Fatalf("emitted %d records, want %d", emitted, want)
			}
			b.Return()
			b.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
