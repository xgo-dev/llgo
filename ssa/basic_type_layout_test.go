package ssa

import (
	"fmt"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Normalize named structs to their actual element types, retaining packing.
func basicLLVMShape(t llvm.Type) string {
	if t.TypeKind() != llvm.StructTypeKind {
		return t.String()
	}
	parts := []string{}
	for _, f := range t.StructElementTypes() {
		parts = append(parts, basicLLVMShape(f))
	}
	shape := "{" + strings.Join(parts, ", ") + "}"
	if t.IsStructPacked() {
		shape = "<" + shape + ">"
	}
	return shape
}

func TestGoBasicTypeLLVMStorage(t *testing.T) {
	Initialize(InitAll)
	targets := []struct {
		name, os, arch, triple, profile string
		word                            uint64
	}{
		{"linux-amd64", "linux", "amd64", "x86_64-unknown-linux-gnu", "", 8},
		{"linux-386", "linux", "386", "i386-unknown-linux-gnu", "", 4},
		{"linux-arm64", "linux", "arm64", "aarch64-unknown-linux-gnu", "", 8},
		{"linux-arm", "linux", "arm", "armv7-unknown-linux-gnueabihf", "", 4},
		{"darwin-amd64", "darwin", "amd64", "x86_64-apple-darwin", "", 8},
		{"darwin-arm64", "darwin", "arm64", "arm64-apple-darwin", "", 8},
		{"windows-amd64", "windows", "amd64", "x86_64-pc-windows-msvc", "", 8},
		{"windows-386", "windows", "386", "i686-pc-windows-msvc", "", 4},
		{"windows-arm64", "windows", "arm64", "aarch64-pc-windows-msvc", "", 8},
		{"wasm-j32", "js", "wasm", "wasm32-unknown-emscripten", "j32", 8},
		{"wasm-j64", "js", "wasm", "wasm64-unknown-emscripten", "j64", 8},
		{"wasm-w32", "wasip1", "wasm", "wasm32-unknown-wasip1", "w32", 8},
		{"linux-riscv64", "linux", "riscv64", "riscv64-unknown-linux-gnu", "", 8},
		{"linux-ppc64", "linux", "ppc64", "powerpc64-unknown-linux-gnu", "", 8},
		{"linux-ppc64le", "linux", "ppc64le", "powerpc64le-unknown-linux-gnu", "", 8},
		{"linux-s390x", "linux", "s390x", "s390x-unknown-linux-gnu", "", 8},
		{"linux-mips", "linux", "mips", "mips-unknown-linux-gnu", "", 4},
		{"linux-mipsle", "linux", "mipsle", "mipsel-unknown-linux-gnu", "", 4},
		{"linux-mips64", "linux", "mips64", "mips64-unknown-linux-gnu", "", 8},
		{"linux-mips64le", "linux", "mips64le", "mips64el-unknown-linux-gnu", "", 8},
		{"linux-loong64", "linux", "loong64", "loongarch64-unknown-linux-gnu", "", 8},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			if _, err := llvm.GetTargetFromTriple(target.triple); err != nil {
				// The official Windows LLVM archive omits several extra
				// backends. Keep the core hosted targets mandatory.
				if runtime.GOOS == "windows" {
					switch target.arch {
					case "ppc64", "ppc64le", "s390x", "mips", "mipsle", "mips64", "mips64le", "loong64":
						t.Skipf("LLVM build lacks %s: %v", target.arch, err)
					}
				}
				t.Fatal(err)
			}
			prog := NewProgram(&Target{GOOS: target.os, GOARCH: target.arch, LLVMTarget: target.triple, WasmProfile: target.profile})
			defer prog.Dispose()
			// Match the production frontend configuration, but never use this
			// Sizes object (or Program.SizeOf) as the expected or observed size.
			prog.TypeSizes(types.SizesFor("gc", target.arch))
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("audit", "audit")
			wordIR := fmt.Sprintf("i%d", target.word*8)
			stringIR := "{ptr, " + wordIR + "}"
			if target.profile == "j32" || target.profile == "w32" {
				stringIR = "{{ptr, i32}, i64}"
			}
			cases := []struct {
				name  string
				kind  types.BasicKind
				shape string
				size  uint64
			}{
				{"bool", types.Bool, "i1", 1},
				{"int8", types.Int8, "i8", 1}, {"uint8", types.Uint8, "i8", 1}, {"byte", types.Uint8, "i8", 1},
				{"int16", types.Int16, "i16", 2}, {"uint16", types.Uint16, "i16", 2},
				{"int32", types.Int32, "i32", 4}, {"uint32", types.Uint32, "i32", 4}, {"rune", types.Int32, "i32", 4},
				{"int64", types.Int64, "i64", 8}, {"uint64", types.Uint64, "i64", 8},
				{"int", types.Int, wordIR, target.word}, {"uint", types.Uint, wordIR, target.word}, {"uintptr", types.Uintptr, wordIR, target.word},
				{"float32", types.Float32, "float", 4}, {"float64", types.Float64, "double", 8},
				{"complex64", types.Complex64, "{float, float}", 8}, {"complex128", types.Complex128, "{double, double}", 16},
				{"string", types.String, stringIR, 2 * target.word},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					raw := types.Typ[tc.kind]
					typ := prog.Type(raw, GoDecl)
					// Inspect the global actually emitted by LLGo. Do not
					// substitute a hand-constructed LLVM type for the result.
					g := pkg.NewVarEx("audit."+tc.name, prog.Pointer(typ))
					physical := g.impl.GlobalValueType()
					g.impl.SetInitializer(llvm.ConstNull(physical))
					td := prog.TargetData()
					shape := basicLLVMShape(physical)
					size := td.TypeAllocSize(physical)
					array := pkg.NewVarEx("audit.array_"+tc.name, prog.Pointer(prog.Type(types.NewArray(raw, 2), GoDecl)))
					arrayType := array.impl.GlobalValueType()
					array.impl.SetInitializer(llvm.ConstNull(arrayType))
					arraySize := td.TypeAllocSize(arrayType)
					wantAlign := tc.size
					switch tc.kind {
					case types.Complex64:
						wantAlign = 4
					case types.Complex128:
						wantAlign = 8
					case types.String:
						wantAlign = target.word
					}
					// Raw LLVM scalar ABI alignment, not a substitute for Go
					// field alignment. i386 System V uses four-byte i64/double
					// ABI alignment; MSVC i686 keeps eight-byte ABI alignment.
					if target.name == "linux-386" && wantAlign == 8 {
						wantAlign = 4
					}
					align := uint64(td.ABITypeAlignment(physical))
					offsets := []uint64{}
					if physical.TypeKind() == llvm.StructTypeKind {
						for i := range physical.StructElementTypes() {
							offsets = append(offsets, td.ElementOffset(physical, i))
						}
					}
					if shape != tc.shape || size != tc.size || arraySize != 2*tc.size {
						t.Errorf("LLVM storage=%s bytes=%d array=%d; want %s bytes=%d array=%d", shape, size, arraySize, tc.shape, tc.size, 2*tc.size)
					}
					if align != wantAlign {
						t.Errorf("LLVM ABI alignment=%d, want %d", align, wantAlign)
					}
					// String's second field is its Go-word-sized length.
					if tc.kind == types.String && (len(offsets) != 2 || offsets[1] != target.word) {
						t.Errorf("string field offsets=%v", offsets)
					}
				})
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Error(err)
			}
		})
	}
}
