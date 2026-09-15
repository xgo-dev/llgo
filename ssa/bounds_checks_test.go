//go:build !llgo

package ssa_test

import (
	"go/types"
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestBoundsCheckModesIR(t *testing.T) {
	checked := boundsCheckModeIR(t, false)
	unchecked := boundsCheckModeIR(t, true)

	for _, helper := range []string{
		"PanicIndex",
		"StringSlice2",
		"NewSlice2",
		"NewSlice3Bounds",
		"PanicSliceConvert",
	} {
		if !strings.Contains(checked, helper) {
			t.Errorf("checked IR does not contain %q", helper)
		}
		if strings.Contains(unchecked, helper) {
			t.Errorf("unchecked IR contains %q", helper)
		}
	}
	if !strings.Contains(unchecked, "AssertNilDeref") {
		t.Error("unchecked *array slice lost its nil check")
	}
	panicPath := regexp.MustCompile(`(?m)^[ \t]+br i1 %[0-9]+, label %(_llgo_[0-9]+), label %_llgo_[0-9]+\n\n(_llgo_[0-9]+):.*\n[ \t]+call void @"[^"]*PanicIndex"\([^\n]*\)\n[ \t]+br label %(_llgo_[0-9]+)$`)
	match := panicPath.FindStringSubmatch(checked)
	if len(match) == 0 || match[1] != match[2] || match[2] != match[3] {
		t.Error("checked IR does not isolate PanicIndex in a non-returning failure branch")
	}
	if got := strings.Count(unchecked, "select i1"); got < 4 {
		t.Errorf("unchecked IR contains %d select operations, want at least 4", got)
	}
}

func TestSignedIndexUsesUnsignedCompare(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	t.Cleanup(prog.Dispose)

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	byteArray := types.NewArray(types.Typ[types.Byte], 4)
	byteArrayPtr := types.NewPointer(byteArray)
	params := types.NewTuple(
		types.NewVar(0, nil, "slice", byteSlice),
		types.NewVar(0, nil, "array", byteArrayPtr),
		types.NewVar(0, nil, "idx", types.Typ[types.Int]),
		types.NewVar(0, nil, "uidx", types.Typ[types.Uint]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg := prog.NewPackage("bounds", "example.com/bounds")
	fn := pkg.NewFunc("signedIndex", sig, ssa.InGo)
	b := fn.MakeBody(1)
	b.IndexAddr(fn.Param(0), fn.Param(2))
	b.IndexAddr(fn.Param(1), fn.Param(2))
	b.IndexAddr(fn.Param(0), fn.Param(3))
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	if strings.Contains(ir, "icmp slt") {
		t.Errorf("signed index still emits a separate min check:\n%s", ir)
	}
	if strings.Contains(ir, "or i1") {
		t.Errorf("signed index still ors min and max checks:\n%s", ir)
	}
	// All three IndexAddr sites use function parameters, so indexNeedsCheck
	// cannot fold any of them away.
	if got := strings.Count(ir, "icmp uge"); got != 3 {
		t.Errorf("signed/unsigned index IR contains %d icmp uge, want 3:\n%s", got, ir)
	}
	if !strings.Contains(ir, "PanicIndex\"") {
		t.Errorf("signed index IR does not call PanicIndex:\n%s", ir)
	}
	if !strings.Contains(ir, "PanicIndexU\"") {
		t.Errorf("unsigned index IR does not call PanicIndexU:\n%s", ir)
	}
}

func TestWideIndexBoundsCheck386(t *testing.T) {
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: "386"})
	t.Cleanup(prog.Dispose)

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	params := types.NewTuple(
		types.NewVar(0, nil, "slice", byteSlice),
		types.NewVar(0, nil, "unsigned", types.Typ[types.Uint64]),
		types.NewVar(0, nil, "signed", types.Typ[types.Int64]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg := prog.NewPackage("bounds", "example.com/bounds")
	fn := pkg.NewFunc("wideIndex", sig, ssa.InGo)
	b := fn.MakeBody(1)
	b.IndexAddr(fn.Param(0), fn.Param(1))
	b.IndexAddr(fn.Param(0), fn.Param(2))
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	for _, want := range []string{
		"icmp uge i64",
		"lshr i64",
		"PanicExtendIndexU\"",
		"PanicExtendIndex\"",
		"trunc i64",
		"getelementptr inbounds i8",
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("386 wide-index IR does not contain %q:\n%s", want, ir)
		}
	}
	if strings.Index(ir, "icmp uge i64") > strings.Index(ir, "trunc i64") {
		t.Errorf("386 wide index is truncated before its bounds comparison:\n%s", ir)
	}
}

func boundsCheckModeIR(t *testing.T, disable bool) string {
	t.Helper()
	prog := ssatest.NewProgram(t, nil)
	t.Cleanup(prog.Dispose)
	prog.DisableBoundsChecks(disable)

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	byteArray := types.NewArray(types.Typ[types.Byte], 4)
	byteArrayPtr := types.NewPointer(byteArray)
	params := types.NewTuple(
		types.NewVar(0, nil, "str", types.Typ[types.String]),
		types.NewVar(0, nil, "slice", byteSlice),
		types.NewVar(0, nil, "array", byteArrayPtr),
		types.NewVar(0, nil, "low", types.Typ[types.Int]),
		types.NewVar(0, nil, "high", types.Typ[types.Int]),
		types.NewVar(0, nil, "max", types.Typ[types.Int]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg := prog.NewPackage("bounds", "example.com/bounds")
	fn := pkg.NewFunc("modes", sig, ssa.InGo)
	b := fn.MakeBody(1)

	str := fn.Param(0)
	slice := fn.Param(1)
	array := fn.Param(2)
	low := fn.Param(3)
	high := fn.Param(4)
	max := fn.Param(5)
	none := ssa.Expr{}

	b.Index(str, low, nil)
	b.IndexAddr(slice, low)
	b.Slice(str, low, high, none)
	b.Slice(slice, low, high, none)
	b.Slice(slice, low, high, max)
	b.Slice(array, none, none, none)
	b.SliceToArrayPointer(slice, prog.Type(byteArrayPtr, ssa.InGo))
	b.Return()
	b.EndBuild()
	return pkg.String()
}
