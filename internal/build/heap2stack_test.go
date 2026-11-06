//go:build !llgo

package build

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// ABI coercions spill aggregates and reload their fields. Stress a long chain
// of these copies: asking Attributor to follow the memory uses before SROA can
// exhaust its analysis stack. The returned allocation must still escape.
// This also guards analysis cost; go test's timeout bounds a stalled pass.
func TestHeapToStackAggregateSpills(t *testing.T) {
	const copies = 1024
	var source strings.Builder
	source.WriteString("declare noalias ptr @alloc(i64) allocsize(0)\ndefine {ptr, ptr} @chain(ptr %type) {\nentry:\n")
	for i := range copies {
		fmt.Fprintf(&source, "%%slot%d = alloca {ptr, ptr}, align 8\n", i)
	}
	source.WriteString(`%p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
%pair0 = insertvalue {ptr, ptr} poison, ptr %p, 0
%pair = insertvalue {ptr, ptr} %pair0, ptr %type, 1
`)
	previous := "%pair"
	for i := range copies {
		fmt.Fprintf(&source, `store {ptr, ptr} %[2]s, ptr %%slot%[1]d
%%field%[1]d = getelementptr {ptr, ptr}, ptr %%slot%[1]d, i32 0, i32 0
%%a%[1]d = load ptr, ptr %%field%[1]d
%%fieldb%[1]d = getelementptr {ptr, ptr}, ptr %%slot%[1]d, i32 0, i32 1
%%b%[1]d = load ptr, ptr %%fieldb%[1]d
%%in%[1]d = insertvalue {ptr, ptr} poison, ptr %%a%[1]d, 0
%%out%[1]d = insertvalue {ptr, ptr} %%in%[1]d, ptr %%b%[1]d, 1
`, i, previous)
		previous = fmt.Sprintf("%%out%d", i)
	}
	fmt.Fprintf(&source, "ret {ptr, ptr} %s\n}\n", previous)
	for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
		t.Run(mode.String(), func(t *testing.T) {
			mod := parseWasmAggregateIR(t, source.String())
			prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch = "linux", "amd64"
			conf.OptLevel, conf.LTO = optlevel.Os, mode
			ctx := &context{prog: prog, buildConf: conf, passOpt: true}
			if err := optimizeLLVMModule(ctx, "aggregate-spills", mod); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction("chain").String()
			if strings.Contains(body, "alloca ") || strings.Count(body, "@alloc(") != 1 {
				t.Fatalf("aggregate spills must disappear while exactly one escaping heap allocation remains:\n%s", body)
			}
		})
	}
}
