package cl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestInterfaceForwardTailRecoverIR(t *testing.T) {
	const source = `package forwardtail
type Scalar interface{ Read() int64 }
type Forward struct{ Scalar }
func scalar(v Forward) Scalar { return v }
func pointer(v *Forward) Scalar { return v }
func ordinary(v Scalar) int64 { return v.Read() }
type Record struct{ A, B int }
type Composite interface{ Read() Record }
type CompositeForward struct{ Composite }
func composite(v CompositeForward) Composite { return v }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")
			body := llvmFunction(t, ir, "forwardtail.Forward.Read")
			if !strings.Contains(body, "StartRecoverFrameAlias") {
				t.Fatalf("forwarder lost the direct-defer token:\n%s", body)
			}
			if target.GOARCH != "wasm" {
				if strings.Contains(body, "EndRecoverFrameAlias") || strings.Contains(body, "IfacePtrDataBuffer") {
					t.Fatalf("forwarder retains post-call cleanup or borrowed frame storage:\n%s", body)
				}
			} else if !strings.Contains(body, "EndRecoverFrameAlias") {
				t.Fatalf("Wasm must retain its existing scope policy:\n%s", body)
			}
			if target.GOARCH == "386" {
				pointer := llvmFunction(t, ir, "forwardtail.(*Forward).Read")
				if !strings.Contains(pointer, "musttail call i64") {
					t.Fatalf("x86 pointer adapter cannot reuse incoming stack slots:\n%s", pointer)
				}
			}
			composite := llvmFunction(t, ir, "forwardtail.CompositeForward.Read")
			if !strings.Contains(composite, "EndRecoverFrameAlias") {
				t.Fatalf("composite forwarder lost conservative cleanup:\n%s", composite)
			}
			ordinary := llvmFunction(t, ir, "forwardtail.ordinary")
			if !strings.Contains(ordinary, "IfacePtrDataBuffer") {
				t.Fatalf("ordinary calls lost their borrowed receiver:\n%s", ordinary)
			}
		})
	}
}

func TestInterfaceForwardWindows386TailAssembly(t *testing.T) {
	llc, err := exec.LookPath("llc")
	if err != nil {
		t.Skip("LLVM llc is unavailable; target IR remains covered")
	}
	opt, err := exec.LookPath("opt")
	if err != nil {
		t.Skip("LLVM opt is unavailable; target IR remains covered")
	}
	ir := compileInterfaceBoxIR(t, `package forwardtail
type Scalar interface{ Read() int64 }
type Forward struct{ Scalar }
func pointer(v *Forward) Scalar { return v }
`, ssa.InstantiateGenerics, &llssa.Target{GOOS: "windows", GOARCH: "386"}, false)
	directory := t.TempDir()
	input, optimized := filepath.Join(directory, "input.ll"), filepath.Join(directory, "optimized.ll")
	if err := os.WriteFile(input, []byte(ir), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(opt, "-O2", "-S", input, "-o", optimized).CombinedOutput(); err != nil {
		t.Fatalf("optimize: %v\n%s", err, output)
	}
	output, err := exec.Command(llc, "-O2", optimized, "-o", "-").CombinedOutput()
	if err != nil {
		t.Fatalf("emit Windows/x86 machine assembly: %v\n%s", err, output)
	}
	start := strings.Index(string(output), `"_forwardtail.(*Forward).Read":`)
	if start < 0 {
		t.Fatalf("missing pointer forwarder:\n%s", output)
	}
	body := string(output[start:])
	if end := strings.Index(body, "# -- End function"); end >= 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "jmpl\t*") || strings.Contains(body, "calll\t*") {
		t.Fatalf("pointer forwarder still builds an indirect-call stack chain:\n%s", body)
	}
}
