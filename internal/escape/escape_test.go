//go:build !llgo

package escape

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/littest"
	"github.com/xgo-dev/llvm"
)

func TestTransformModule(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "pointer-info" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join("testdata", entry.Name())
			input := filepath.Join(dir, "in.txt")
			buffer, err := llvm.NewMemoryBufferFromFile(input)
			if err != nil {
				t.Fatal(err)
			}
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod, err := ctx.ParseIR(buffer)
			if err != nil {
				t.Fatalf("parse %s: %v", input, err)
			}
			defer mod.Dispose()
			result := TransformModule(mod, true)

			output := filepath.Join(dir, "out.txt")
			want, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			spec := littest.Spec{Path: output, Text: string(want), Mode: littest.ModeLiteral}
			if err := littest.Check(spec, mod.String()); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}

			diagnostics := filepath.Join(dir, "diagnostics.txt")
			wantDiagnostics, err := os.ReadFile(diagnostics)
			if os.IsNotExist(err) {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			spec = littest.Spec{Path: diagnostics, Text: string(wantDiagnostics), Mode: littest.ModeLiteral}
			if got := formatParameterSummaries(result); littest.Check(spec, got) != nil {
				t.Fatalf("diagnostics mismatch:\n%s", got)
			}
		})
	}
}

func formatParameterSummaries(result Result) string {
	var out strings.Builder
	for _, param := range result.Parameters {
		fmt.Fprintf(&out, "%s param=%d heap=%d mutator=%d callee=%d", param.Function, param.Parameter, param.HeapLevel, param.MutatorLevel, param.CalleeLevel)
		for _, leak := range param.Results {
			fmt.Fprintf(&out, " result=%d level=%d", leak.Result, leak.Level)
		}
		out.WriteByte('\n')
	}
	return out.String()
}
