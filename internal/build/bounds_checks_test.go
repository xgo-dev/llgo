package build

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const boundsChecksFixture = "./testdata/boundschecks"

func TestDisableBoundsChecksIR(t *testing.T) {
	checked := boundsChecksModuleIR(t, false)
	unchecked := boundsChecksModuleIR(t, true)

	for _, helper := range []string{"PanicIndex", "StringSlice2", "NewSlice2", "NewSlice3Bounds", "PanicSliceConvert"} {
		if !strings.Contains(checked, helper) {
			t.Errorf("default IR does not contain bounds helper %q", helper)
		}
		if strings.Contains(unchecked, helper) {
			t.Errorf("-B IR unexpectedly contains bounds helper %q", helper)
		}
	}

	for _, function := range []string{"indexString", "indexSlice", "indexArray", "indexArrayPointer"} {
		body := llvmFunctionBody(t, unchecked, function)
		if strings.Contains(body, "PanicIndex") {
			t.Errorf("-B %s contains an index bounds check:\n%s", function, body)
		}
	}
	for _, function := range []string{"sliceString", "sliceSlice", "sliceArray", "sliceArrayPointer", "sliceThree"} {
		body := llvmFunctionBody(t, unchecked, function)
		if strings.Contains(body, "StringSlice2") || strings.Contains(body, "NewSlice2") || strings.Contains(body, "NewSlice3Bounds") {
			t.Errorf("-B %s contains a slice bounds helper:\n%s", function, body)
		}
	}
	for _, function := range []string{"shortSliceToArrayPointer", "shortSliceToArrayValue"} {
		checkedBody := llvmFunctionBody(t, checked, function)
		if !strings.Contains(checkedBody, "PanicSliceConvert") {
			t.Errorf("default %s does not contain its conversion bounds check:\n%s", function, checkedBody)
		}
		uncheckedBody := llvmFunctionBody(t, unchecked, function)
		if strings.Contains(uncheckedBody, "PanicSliceConvert") {
			t.Errorf("-B %s contains a conversion bounds check:\n%s", function, uncheckedBody)
		}
	}
	if body := llvmFunctionBody(t, unchecked, "shortSliceToArrayValue"); !strings.Contains(body, "load [4 x i8]") {
		t.Errorf("slice-to-array value conversion does not dereference its converted pointer:\n%s", body)
	}

	for _, function := range []string{"indexArrayPointer", "sliceArrayPointer"} {
		body := llvmFunctionBody(t, unchecked, function)
		if !strings.Contains(body, "AssertNilDeref") {
			t.Errorf("-B %s lost its *array nil check:\n%s", function, body)
		}
	}
	for _, width := range []string{"zext i8", "zext i16", "zext i32"} {
		if !strings.Contains(unchecked, width) {
			t.Errorf("-B IR does not retain integer-width conversion %q", width)
		}
	}
	for _, function := range []string{"makeUnsafeString", "makeUnsafeSlice"} {
		body := llvmFunctionBody(t, unchecked, function)
		if !strings.Contains(body, "AssertRuntimeError") {
			t.Errorf("-B %s lost mandatory unsafe builtin checks:\n%s", function, body)
		}
	}
}

func TestDisableBoundsChecksLegalResultsMatchDefault(t *testing.T) {
	wantFields := []string{"98", "30", "10", "40", "bc", "2", "3", "2", "3", "2", "3", "2", "3", "10", "40", "20", "30"}
	checked := runBinary(t, buildBoundsChecksBinary(t, false))
	unchecked := runBinary(t, buildBoundsChecksBinary(t, true))
	if checked != unchecked {
		t.Fatalf("default and -B output differ:\ndefault %q\n-B      %q", checked, unchecked)
	}
	if fields := strings.Fields(unchecked); !reflect.DeepEqual(fields, wantFields) {
		t.Fatalf("legal -B results = %v, want %v", fields, wantFields)
	}
}

func TestDisableBoundsChecksShortSliceConversionsDoNotPanic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bounds-disabled")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = path
	conf.DisableBoundsChecks = true
	if _, err := Do([]string{"./testdata/boundschecks_convert_short"}, conf); err != nil {
		t.Fatalf("build short conversion fixture with bounds checks disabled: %v", err)
	}
	if output := runBinary(t, path); !reflect.DeepEqual(strings.Fields(output), []string{"1", "4", "1", "4"}) {
		fields := strings.Fields(output)
		t.Fatalf("short conversions with bounds checks disabled = %v, want [1 4 1 4]; output %q", fields, output)
	}
}

func TestDisableBoundsChecksRetainsRequiredPanics(t *testing.T) {
	output := runBinary(t, buildBoundsChecksBinaryFrom(t, "./testdata/boundschecks_required", true))
	want := []string{"true", "true", "true", "true"}
	if fields := strings.Fields(output); !reflect.DeepEqual(fields, want) {
		t.Fatalf("required -B panics = %v, want %v; output %q", fields, want, output)
	}
}

func boundsChecksModuleIR(t *testing.T, disable bool) string {
	t.Helper()
	conf := NewDefaultConf(ModeGen)
	conf.DisableBoundsChecks = disable
	var ir string
	conf.ModuleHook = func(pkg Package) {
		module := pkg.LPkg.String()
		if strings.Contains(module, "main.indexString(") || strings.Contains(module, ".indexString\"(") {
			ir = module
		}
	}
	pkgs, err := Do([]string{boundsChecksFixture}, conf)
	if err != nil {
		t.Fatalf("generate bounds-check IR (disabled=%v): %v", disable, err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatalf("generate bounds-check IR (disabled=%v): packages = %#v", disable, pkgs)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	if ir == "" {
		t.Fatalf("generate bounds-check IR (disabled=%v): fixture module was not observed", disable)
	}
	return ir
}

func buildBoundsChecksBinary(t *testing.T, disable bool) string {
	t.Helper()
	return buildBoundsChecksBinaryFrom(t, boundsChecksFixture, disable)
}

func buildBoundsChecksBinaryFrom(t *testing.T, fixture string, disable bool) string {
	t.Helper()
	name := "checked"
	if disable {
		name = "unchecked"
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = path
	conf.DisableBoundsChecks = disable
	if _, err := Do([]string{fixture}, conf); err != nil {
		t.Fatalf("build bounds-check fixture (disabled=%v): %v", disable, err)
	}
	return path
}

func llvmFunctionBody(t *testing.T, module, name string) string {
	t.Helper()
	markers := []string{"." + name + "\"(", "." + name + "("}
	for _, marker := range markers {
		markerAt := 0
		for {
			next := strings.Index(module[markerAt:], marker)
			if next < 0 {
				break
			}
			markerAt += next
			lineStart := strings.LastIndex(module[:markerAt], "\n") + 1
			if strings.HasPrefix(module[lineStart:markerAt], "define ") {
				end := strings.Index(module[markerAt:], "\n}")
				if end < 0 {
					t.Fatalf("end of LLVM definition for %q not found", name)
				}
				return module[lineStart : markerAt+end+2]
			}
			markerAt += len(marker)
		}
	}
	t.Fatalf("LLVM definition for %q not found", name)
	return ""
}
