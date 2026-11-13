package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAESMachineCode(t *testing.T) {
	Initialize(InitAll)
	for _, tc := range []struct {
		arch, os, instruction string
		arm, x86              string
	}{
		{"arm64", "linux", "aese", "", ""},
		{"arm64", "darwin", "aese", "", ""},
		{"arm64", "windows", "aese", "", ""},
		{"amd64", "linux", "aesenc", "", ""},
		{"amd64", "windows", "aesenc", "", ""},
		{"386", "linux", "aesenc", "", ""},
		{"386", "linux", "aesenc", "", "softfloat"},
		{"arm", "linux", "aese", "", ""},
		{"arm", "linux", "aese", "5,softfloat", ""},
		{"ppc64", "linux", "vcipher", "", ""},
		{"ppc64le", "linux", "vcipher", "", ""},
		{"riscv64", "linux", "aes64esm", "", ""},
	} {
		t.Run(tc.os+"-"+tc.arch+tc.arm+tc.x86, func(t *testing.T) {
			target := &Target{GOOS: tc.os, GOARCH: tc.arch, GOARM: tc.arm, GO386: tc.x86}
			if _, err := llvm.GetTargetFromTriple(target.Spec().Triple); err != nil {
				// Match the existing storage/ABI matrix: the official Windows
				// LLVM SDK omits PPC. The artifact-producing conformance lane
				// must still provide every target, including both PPC endians.
				if runtime.GOOS == "windows" && os.Getenv("LLGO_AES_CROSS_DIR") == "" && (tc.arch == "ppc64" || tc.arch == "ppc64le") {
					t.Skipf("Windows LLVM build lacks %s: %v", tc.arch, err)
				}
				t.Fatal(err)
			}
			prog := NewProgram(target)
			defer prog.Dispose()
			pkg := prog.NewPackage("aesprobe", "aesprobe")
			ptr := types.NewPointer(types.Typ[types.Uint8])
			params := types.NewTuple(
				types.NewParam(token.NoPos, nil, "dst", ptr),
				types.NewParam(token.NoPos, nil, "src", ptr),
				types.NewParam(token.NoPos, nil, "key", ptr),
			)
			sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
			var first Function
			for op := AESEncryptRound; op <= AESXORBlock; op++ {
				fn := pkg.NewFunc(fmt.Sprintf("aes_op_%d", op), sig, CDecl)
				b := fn.MakeBody(1)
				b.AESRound(op, fn.Param(0), fn.Param(1), fn.Param(2))
				b.Return()
				b.EndBuild()
				if op == AESEncryptRound {
					first = fn
				}
			}
			// An aligned typed stack slot must vanish into the AES round chain.
			chain := pkg.NewFunc("aes_local_chain", sig, CDecl)
			cb := chain.MakeBody(1)
			state := cb.Convert(prog.Pointer(prog.Byte()), cb.AESAlloca(16))
			cb.AESRound(AESEncryptRound, state, chain.Param(1), chain.Param(2))
			for range 12 {
				cb.AESRound(AESEncryptRound, state, state, chain.Param(2))
			}
			cb.AESRound(AESEncryptLast, chain.Param(0), state, chain.Param(2))
			cb.Return()
			cb.EndBuild()
			if !strings.Contains(chain.impl.String(), "alloca [1 x <16 x i8>], align 16") {
				t.Fatalf("AES buffer is not statically typed and aligned:\n%s", chain.impl.String())
			}
			u32 := types.Typ[types.Uint32]
			sub := pkg.NewFunc("aes_subword", types.NewSignatureType(nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "x", u32)),
				types.NewTuple(types.NewParam(token.NoPos, nil, "", u32)), false), CDecl)
			b := sub.MakeBody(1)
			b.Return(b.AESSubWord(sub.Param(0)))
			b.EndBuild()
			baseline := pkg.NewFunc("aes_baseline", sig, CDecl)
			b = baseline.MakeBody(1)
			b.Call(first.Expr, baseline.Param(0), baseline.Param(1), baseline.Param(2))
			b.Return()
			b.EndBuild()
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			pbo := llvm.NewPassBuilderOptions()
			defer pbo.Dispose()
			if err := mod.RunPasses("default<O3>", prog.TargetMachine(), pbo); err != nil {
				t.Fatal(err)
			}
			if body := chain.impl.String(); strings.Contains(body, "alloca ") || strings.Count(body, "store ") != 1 {
				t.Fatalf("AES state was not promoted to registers:\n%s", body)
			}
			for _, attr := range first.impl.GetFunctionAttributes() {
				if !attr.IsString() || attr.GetStringKind() != "target-features" {
					continue
				}
				features := strings.Split(attr.GetStringValue(), ",")
				for _, feature := range features {
					if (tc.arch == "386" && (feature == "-sse" || feature == "-sse2" || feature == "+soft-float")) ||
						(tc.arch == "arm" && strings.HasPrefix(strings.TrimLeft(feature, "+-"), "armv") && feature != "+armv8-a") {
						t.Fatalf("AES function retained baseline requirement %q", feature)
					}
				}
			}
			// Full-module optimization must leave a real dispatch boundary.
			if !strings.Contains(baseline.impl.String(), "call void @aes_op_0") {
				t.Fatalf("AES instructions escaped into the baseline caller:\n%s", baseline.impl.String())
			}
			for _, attr := range baseline.impl.GetFunctionAttributes() {
				if attr.IsString() && attr.GetStringKind() == "target-features" && strings.Contains(attr.GetStringValue(), "+aes") {
					t.Fatal("baseline caller acquired the AES CPU requirement")
				}
			}
			asm, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer asm.Dispose()
			if !strings.Contains(strings.ToLower(string(asm.Bytes())), tc.instruction) {
				t.Fatalf("missing hardware instruction %s", tc.instruction)
			}
			// Optional artifacts feed the QEMU conformance runner without
			// involving a C frontend in any AES implementation operation.
			if dir := os.Getenv("LLGO_AES_CROSS_DIR"); dir != "" && tc.os == "linux" && tc.arm == "" && tc.x86 == "" {
				obj, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
				if err != nil {
					t.Fatal(err)
				}
				defer obj.Dispose()
				if err := os.WriteFile(filepath.Join(dir, tc.arch+".o"), obj.Bytes(), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, tc.arch+".ll"), []byte(mod.String()), 0644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestAESBuilderRejectsInvalidConstants(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		call       func()
	}{
		{"buffer size", "aesAlloca: expected 16 or 128 bytes", func() { var b Builder; b.AESAlloca(32) }},
		{"round opcode", "aesRound: invalid operation", func() { var b Builder; b.AESRound(AESOp(255), Expr{}, Expr{}, Expr{}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := fmt.Sprint(recover()); got != tc.want {
					t.Fatalf("panic = %q, want %q", got, tc.want)
				}
			}()
			tc.call()
		})
	}
}
