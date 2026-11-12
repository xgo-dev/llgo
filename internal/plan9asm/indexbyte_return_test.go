package plan9asm

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func llFuncBody(ll, fnSig string) string {
	start := strings.Index(ll, fnSig)
	if start < 0 {
		return ""
	}
	rest := ll[start:]
	next := strings.Index(rest[len(fnSig):], "\ndefine ")
	if next < 0 {
		return rest
	}
	return rest[:len(fnSig)+next]
}

func TestPlan9AsmIndexByteStringReturnsResult(t *testing.T) {
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Fatal("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "linux", "arm64")

	sfile := filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s")
	source, err := os.ReadFile(sfile)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := TranslateFileForPkgWithOptions(pkg, sfile, "linux", "arm64", nil, TranslateOptions{AnnotateSource: true, GOROOT: goroot})
	if err != nil {
		t.Fatal(err)
	}
	body := llFuncBody(tr.LLVMIR, `define i64 @"internal/bytealg.IndexByteString"`)
	if body == "" {
		t.Fatalf("IndexByteString function not found in translated IR")
	}
	if containsTextSymbol(string(source), "indexbytebody<>") {
		if !strings.Contains(body, `call void @"internal/bytealg.indexbytebody"`) {
			t.Fatalf("expected helper call in IndexByteString:\n%s", body)
		}
		if !strings.Contains(body, "ptrtoint ptr %fp_ret_0 to i64") {
			t.Fatalf("expected helper result slot address setup in IndexByteString:\n%s", body)
		}
		if !strings.Contains(body, "load i64, ptr %fp_ret_0") {
			t.Fatalf("expected helper result slot load in IndexByteString:\n%s", body)
		}
		return
	}
	// Go 1.26 inlines indexbytebody into the ABIInternal entry point and
	// returns the result in R0, so there is no helper result slot to inspect.
	if !strings.Contains(body, "load i64, ptr %reg_R0") || !strings.Contains(body, "ret i64 %") {
		t.Fatalf("expected direct R0 result return in IndexByteString:\n%s", body)
	}
}
