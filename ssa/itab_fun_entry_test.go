package ssa

import (
	"go/types"
	"runtime"
	"strings"
	"testing"

	"go/importer"

	"github.com/xgo-dev/llvm"
)

// TestItabFunEntryDataWord pins the data-word itab calling convention:
// interface method calls forward the raw interface data word instead of
// routing through the kind-dispatch helper, nil interfaces take a branchy
// guard, and itab fun slots hold the method itself for direct non-pointer
// receivers while boxed receivers get a check-free $dfn adapter.
func TestItabFunEntryDataWord(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkgTypes := types.NewPackage("example.com/itabentry", "itabentry")

	methSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	meth := types.NewFunc(0, pkgTypes, "M", methSig)
	rawIface := types.NewInterfaceType([]*types.Func{meth}, nil)
	rawIface.Complete()

	// B is an empty struct: boxed in interfaces, so its entry is a $dfn
	// adapter that loads through the (never nil) box pointer.
	bNamed := types.NewNamed(types.NewTypeName(0, pkgTypes, "B", nil), types.NewStruct(nil, nil), nil)
	bRecv := types.NewVar(0, pkgTypes, "", bNamed)
	bNamed.AddMethod(types.NewFunc(0, pkgTypes, "M", types.NewSignatureType(bRecv, nil, nil, nil, nil, false)))
	// W wraps a pointer field: direct interface word, so the method itself
	// (its Tfn) is the entry.
	wStruct := types.NewStruct([]*types.Var{
		types.NewField(0, pkgTypes, "p", types.NewPointer(types.Typ[types.Int]), false),
	}, nil)
	wNamed := types.NewNamed(types.NewTypeName(0, pkgTypes, "W", nil), wStruct, nil)
	wRecv := types.NewVar(0, pkgTypes, "", wNamed)
	wNamed.AddMethod(types.NewFunc(0, pkgTypes, "M", types.NewSignatureType(wRecv, nil, nil, nil, nil, false)))

	pkg := prog.NewPackageEx("itabentry", "example.com/itabentry", true)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := pkg.NewFunc("make", sig, GoDecl)
	b := fn.MakeBody(1)
	for _, concrete := range []types.Type{bNamed, wNamed} {
		iface := b.MakeInterface(prog.Type(rawIface, GoDecl), prog.Zero(prog.Type(concrete, GoDecl)))
		b.Call(b.Imethod(iface, meth))
	}
	b.Return()
	b.EndBuild()
	pkg.MaterializePreserveSyms()

	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := pkg.String()
	if strings.Contains(ir, "IfacePtrData") {
		t.Fatalf("interface calls must not route through the kind-dispatch helper:\n%s", ir)
	}

	itabLines := map[string]string{}
	for _, line := range strings.Split(ir, "\n") {
		if !strings.HasPrefix(line, `@"_llgo_itab$`) || !strings.Contains(line, `= weak_odr constant`) {
			continue
		}
		for _, concrete := range []string{"B", "W"} {
			if strings.Contains(line, `@"_llgo_example.com/itabentry.`+concrete+`"`) {
				itabLines[concrete] = line
			}
		}
	}
	if len(itabLines) != 2 {
		t.Fatalf("missing static itab templates for B and W:\n%s", ir)
	}
	for _, concrete := range []string{"B", "W"} {
		line, ok := itabLines[concrete]
		if !ok {
			t.Fatalf("missing %s itab template:\n%s", concrete, ir)
		}
		entry := `M$dfn"`
		if concrete == "W" {
			entry = `M"`
		}
		if !strings.Contains(line, entry) {
			t.Fatalf("%s itab must hold %q as the fun entry:\n%s", concrete, entry, line)
		}
	}

	// The boxed adapter carries no nil check: one block, one call, straight
	// through to the method.
	dfn := pkg.FuncOf("example.com/itabentry.B.M$dfn")
	if dfn == nil {
		t.Fatalf("missing boxed-receiver adapter:\n%s", ir)
	}
	if got := dfn.impl.BasicBlocksCount(); got != 1 {
		t.Fatalf("boxed adapter must be a single tail block, got %d blocks:\n%s", got, dfn.impl.String())
	}
	calls := 0
	for inst := dfn.impl.EntryBasicBlock().FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if inst.InstructionOpcode() != llvm.Call {
			continue
		}
		calls++
		callee := inst.CalledValue().IsAFunction()
		if callee.IsNil() || callee.Name() != "example.com/itabentry.B.M" {
			t.Fatalf("boxed adapter must forward to B.M:\n%s", dfn.impl.String())
		}
	}
	if calls != 1 {
		t.Fatalf("boxed adapter must contain exactly one call, got %d:\n%s", calls, dfn.impl.String())
	}

	// The caller extracts the raw data word, guards the itab branchily and
	// invokes the loaded fun pointer directly.
	entry := fn.impl.EntryBasicBlock()
	for inst := entry.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if inst.InstructionOpcode() != llvm.Call {
			continue
		}
		callee := inst.CalledValue().IsAFunction()
		if !callee.IsNil() && strings.Contains(callee.Name(), "IfacePtrData") {
			t.Fatalf("caller must not call the kind-dispatch helper:\n%s", fn.impl.String())
		}
	}
	branch := entry.LastInstruction()
	if branch.InstructionOpcode() != llvm.Br || branch.SuccessorsCount() != 2 {
		t.Fatalf("nil interface guard must branch on the hot path:\n%s", fn.impl.String())
	}
	cond := branch.Operand(0)
	if cond.IsAInstruction().IsNil() || cond.InstructionOpcode() != llvm.ICmp {
		t.Fatalf("nil interface guard must be an icmp:\n%s", fn.impl.String())
	}
	// The data word extract lives in the guard's success block: the cold block
	// panics before touching anything but the itab word.
	dataExtract := false
	success := branch.Successor(1)
	for inst := success.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if inst.InstructionOpcode() == llvm.ExtractValue && strings.Contains(inst.String(), ", 1") {
			dataExtract = true
			break
		}
	}
	if !dataExtract {
		t.Fatalf("caller must forward the raw data word:\n%s", fn.impl.String())
	}
}
