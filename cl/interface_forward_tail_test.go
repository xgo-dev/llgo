package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestInterfaceForwardTailRecoverIR(t *testing.T) {
	const source = `package forwardtail
type Scalar interface{ Read() int }
type Forward struct{ Scalar }
func scalar(v Forward) Scalar { return v }
func ordinary(v Scalar) int { return v.Read() }
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
