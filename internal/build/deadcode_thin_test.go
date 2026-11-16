package build

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/xtool/ar"
	"github.com/xgo-dev/llvm"
)

func TestDeadcodeThinArchivePolicy(t *testing.T) {
	for _, ctx := range []*context{nil, {}} {
		if useDeadcodeThinArchives(ctx) {
			t.Fatal("missing build configuration accepted")
		}
	}
	if !buildenv.Dev || runtime.GOOS != "darwin" {
		t.Skip("development macOS experiment")
	}
	base := Config{DeadcodeDrop: true, LTO: lto.Thin, BuildMode: BuildModeExe,
		Goos: runtime.GOOS, Goarch: runtime.GOARCH, DebugArtifactMode: DebugArtifactNone}
	for _, mode := range []Mode{ModeBuild, ModeTest} {
		ctx := &context{buildConf: &base, mode: mode, initial: []*packages.Package{{}}}
		if !useDeadcodeThinArchives(ctx) {
			t.Fatalf("mode %v: single-root size build excluded", mode)
		}
		if !useDeadcodeReflectNames(ctx) {
			t.Fatalf("mode %v: Thin backend enabled without shared analysis", mode)
		}
		strong := base
		strong.LTO = lto.Off
		strongCtx := &context{buildConf: &strong, mode: mode, initial: ctx.initial}
		if !useDeadcodeReflectNames(strongCtx) || useDeadcodeThinArchives(strongCtx) {
			t.Fatalf("mode %v: native Strong did not retain analysis without Thin rewriting", mode)
		}
		ctx.initial = append(ctx.initial, &packages.Package{})
		if useDeadcodeThinArchives(ctx) {
			t.Fatalf("mode %v: multiple roots accepted", mode)
		}
	}
	for name, modify := range map[string]func(*Config){
		"full LTO": func(c *Config) { c.LTO = lto.Full },
		"shared":   func(c *Config) { c.BuildMode = BuildModeCShared },
		"DWARF":    func(c *Config) { c.DebugArtifactMode = DebugArtifactEmbedded },
		"IR":       func(c *Config) { c.GenLL = true },
		"target":   func(c *Config) { c.Target = "wasi" },
		"no drop":  func(c *Config) { c.DeadcodeDrop = false },
	} {
		conf := base
		modify(&conf)
		if useDeadcodeThinArchives(&context{buildConf: &conf, mode: ModeBuild, initial: []*packages.Package{{}}}) {
			t.Errorf("accepted %s", name)
		}
	}
}

func TestDeadcodeThinArchivePreservesCacheAndNativeMembers(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mach-O archive format")
	}
	llssa.Initialize(llssa.InitAll)
	llvmCtx := llvm.NewContext()
	defer llvmCtx.Dispose()
	mod := llvmCtx.NewModule("methods")
	defer mod.Dispose()
	mod.SetTarget(llvm.DefaultTargetTriple())
	addMethodTypeGlobal(mod, "_llgo_pkg.T")
	original := llvm.WriteThinLTOBitcodeToMemoryBuffer(mod)
	defer original.Dispose()

	nativeMod := llvmCtx.NewModule("native")
	defer nativeMod.Dispose()
	nativeMod.SetTarget(mod.Target())
	f := llvm.AddFunction(nativeMod, "native_member", llvm.FunctionType(llvmCtx.VoidType(), nil, false))
	b := llvmCtx.NewBuilder()
	b.SetInsertPointAtEnd(llvmCtx.AddBasicBlock(f, "entry"))
	b.CreateRetVoid()
	b.Dispose()
	target, err := llvm.GetTargetFromTriple(mod.Target())
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(mod.Target(), "generic", "", llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	defer machine.Dispose()
	native, err := machine.EmitToMemoryBuffer(nativeMod, llvm.ObjectFile)
	if err != nil {
		t.Fatal(err)
	}
	defer native.Dispose()
	dir := t.TempDir()
	input := filepath.Join(dir, "cache.a")
	if err := llvm.WriteArchive(input, mod.Target(), []llvm.ArchiveMember{
		llvm.NewArchiveMemberFromMemoryBuffer("methods.o", original),
		llvm.NewArchiveMemberFromMemoryBuffer("native.o", native),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
		crossCompile: crosscompile.Export{LLVMTarget: mod.Target()}}
	out, work, err := rewriteDeadcodeThinArchives(ctx, []string{input}, map[string][]int{"_llgo_pkg.T": {0}}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(work)
	if len(out) != 1 || out[0] == input {
		t.Fatalf("archive was not replaced: %v", out)
	}
	after, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("cached input changed: %v", err)
	}
	output, err := os.Open(out[0])
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	r, err := ar.NewReader(output)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var order []string
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimRight(header.Name, "\x00")
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		seen[name] = true
		if !strings.HasPrefix(name, "__.SYMDEF") {
			order = append(order, name)
		}
		switch name {
		case "native.o":
			if !bytes.Equal(data, native.Bytes()) {
				t.Fatal("native member changed")
			}
		case "methods.o":
			path := filepath.Join(dir, "rewritten.bc")
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			parsed, err := llvmCtx.ParseBitcodeFile(path)
			if err != nil {
				t.Fatal(err)
			}
			defer parsed.Dispose()
			g := parsed.NamedGlobal("_llgo_pkg.T")
			if g.Linkage() != llvm.WeakODRLinkage {
				t.Fatal("descriptor linkage changed")
			}
			slots := g.Initializer().Operand(1)
			if slots.Operand(0).Operand(3).Name() != "pkg.T.M" ||
				slots.Operand(1).Operand(3).Name() != llssa.PkgRuntime+".unreachableMethod" {
				t.Fatalf("unexpected rewritten slots: %s", slots)
			}
			if err := llvm.VerifyModule(parsed, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !seen["native.o"] || !seen["methods.o"] {
		t.Fatalf("missing archive members: %v", seen)
	}
	if strings.Join(order, ",") != "methods.o,native.o" {
		t.Fatalf("archive member order changed: %v", order)
	}
	// A second program using both slots must still see the unmodified cache.
	second, secondWork, err := rewriteDeadcodeThinArchives(ctx, []string{input}, map[string][]int{"_llgo_pkg.T": {0, 1}}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(secondWork)
	if second[0] != input {
		t.Fatal("a program using both methods rewrote the original archive")
	}
}
