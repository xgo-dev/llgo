//go:build go1.27

package build

import (
	"bytes"
	"debug/pe"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
)

func windowsStdCallCommand(t *testing.T, binary string, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	if err := runRunnerCommand(commandEnv{environ: os.Environ()}, binary, args,
		runnerDetails{phase: "windows-stdcall-bridge", timeout: time.Minute}, &output, &output); err != nil {
		t.Fatalf("actual command %q: %v\n%s", append([]string{binary}, args...), err, output.String())
	}
	return output.String()
}

func windowsStdCallTools(t *testing.T) (clang, readobj string) {
	t.Helper()
	bin := llvmenv.New("").BinDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	clang, readobj = filepath.Join(bin, "clang"+ext), filepath.Join(bin, "llvm-readobj"+ext)
	for _, binary := range []string{clang, readobj} {
		version := windowsStdCallCommand(t, binary, "--version")
		if !strings.Contains(version, "version 22.") {
			t.Fatal("the real bridge requires LLVM22 tools", binary, version)
		}
	}
	return
}

// Actual COFF/unwind evidence is separate from Windows execution. Both CRT
// profiles must encode a stable frame pointer and the correct stack probe ABI.
func TestWindowsStdCallDynamicThunkCOFF(t *testing.T) {
	clang, readobj := windowsStdCallTools(t)
	assembly := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime", "_wrap", "syscall_windows.S")
	adapter := filepath.Join(filepath.Dir(assembly), "stdcall_windows_go127.c")
	for _, arch := range []struct {
		name, triple string
		machine      uint16
	}{
		{"386", "i686", pe.IMAGE_FILE_MACHINE_I386},
		{"amd64", "x86_64", pe.IMAGE_FILE_MACHINE_AMD64},
		{"arm64", "aarch64", pe.IMAGE_FILE_MACHINE_ARM64},
	} {
		for _, abi := range []string{"w64-windows-gnu", "pc-windows-msvc"} {
			t.Run(arch.name+"/"+abi, func(t *testing.T) {
				object := filepath.Join(t.TempDir(), "thunk.obj")
				windowsStdCallCommand(t, clang, "--target="+arch.triple+"-"+abi, "-c", assembly, "-o", object)
				file, err := pe.Open(object)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				if file.Machine != arch.machine {
					t.Fatal("bridge object has the wrong actual target", file.Machine)
				}
				probe := "__chkstk"
				if abi == "w64-windows-gnu" && arch.name != "arm64" {
					probe = "___chkstk_ms"
				}
				found := false
				for _, symbol := range file.Symbols {
					found = found || symbol.Name == probe && symbol.SectionNumber == 0
				}
				if !found {
					t.Fatal("dynamic frame does not reference its actual CRT stack probe", probe)
				}
				objdump := filepath.Join(filepath.Dir(readobj), "llvm-objdump")
				if runtime.GOOS == "windows" {
					objdump += ".exe"
				}
				arguments := []string{"--disassemble", "--no-show-raw-insn"}
				if arch.name != "arm64" {
					arguments = append(arguments, "--x86-asm-syntax=intel")
				}
				arguments = append(arguments, object)
				windowsStdCallRequirePreservedRegisters(t, arch.name, windowsStdCallCommand(t, objdump, arguments...))
				bridgeObject := filepath.Join(t.TempDir(), "bridge.obj")
				windowsStdCallCommand(t, clang, "--target="+arch.triple+"-"+abi,
					"-ffreestanding", "-std=c11", "-c", adapter, "-o", bridgeObject)
				bridge, err := pe.Open(bridgeObject)
				if err != nil {
					t.Fatal(err)
				}
				defer bridge.Close()
				if bridge.Machine != arch.machine {
					t.Fatal("C adapter uses another machine ABI", bridge.Machine)
				}
				for _, name := range []string{"llgo_windows_stdcall_go127", "llgo_windows_stdcall_go127_addr"} {
					if arch.name == "386" {
						name = "_" + name
					}
					defined := false
					for _, symbol := range bridge.Symbols {
						defined = defined || symbol.Name == name && symbol.SectionNumber > 0
					}
					if !defined {
						t.Fatal("C adapter lacks its actual C ABI entry", name)
					}
				}
				if arch.name == "386" {
					return // x86 uses its preserved EBP chain, not x64/ARM64 .pdata.
				}
				for _, name := range []string{".pdata", ".xdata"} {
					section := file.Section(name)
					if section == nil || section.Size == 0 {
						t.Fatal("dynamic frame lacks real exception metadata", name)
					}
				}
				unwind := windowsStdCallCommand(t, readobj, "--unwind", object)
				if arch.name == "amd64" && !strings.Contains(unwind, "FrameRegister: RBP") {
					t.Fatal("dynamic x64 frame is not recoverable through RBP", unwind)
				}
				if arch.name == "arm64" && !strings.Contains(unwind, "mov sp, fp") && !strings.Contains(unwind, "mov fp, sp") {
					t.Fatal("dynamic ARM64 frame lacks its decoded frame-pointer operation", unwind)
				}
			})
		}
	}
}

// Audit the real assembled instructions, not just SEH declarations. This
// thunk needs only these preserved registers; any additional nonvolatile use
// requires an explicit save/restore review rather than silently clobbering it.
func windowsStdCallRequirePreservedRegisters(t *testing.T, arch, disassembly string) {
	t.Helper()
	instructions, err := windowsStdCallDecodeInstructions(arch, disassembly)
	if err != nil {
		t.Fatal(err)
	}
	var operands, code []string
	for _, instruction := range instructions {
		operands = append(operands, instruction.operands)
		code = append(code, instruction.opcode+" "+instruction.operands)
	}
	text := strings.Join(operands, "\n")
	compact := strings.Join(strings.Fields(strings.Join(code, "\n")), "")
	if arch == "arm64" {
		for _, match := range windowsStdCallARMGP.FindAllStringSubmatch(text, -1) {
			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			if n >= 20 && n <= 28 {
				t.Fatal("assembled ARM64 thunk uses an unpreserved nonvolatile register", match[0], disassembly)
			}
		}
		if windowsStdCallARMSIMD.MatchString(text) {
			t.Fatal("assembled ARM64 thunk uses an unpreserved SIMD register", disassembly)
		}
		for _, pattern := range []string{
			`strx19,\[sp,#(?:16|0x10)\]`, `ldrx19,\[sp,#(?:16|0x10)\]`,
			`stpx29,x30,\[sp,#-(?:32|0x20)\]!`, `ldpx29,x30,\[sp\],#(?:32|0x20)`,
			`movx29,sp`, `movsp,x29`,
		} {
			if !regexp.MustCompile(pattern).MatchString(compact) {
				t.Fatal("assembled ARM64 thunk lost a real save/restore instruction", pattern, disassembly)
			}
		}
		return
	}
	registers := []string{"ebp", "ebx", "esi", "edi"}
	if arch == "amd64" {
		registers = []string{"rbp"}
		if regexp.MustCompile(`\b(?:[re](?:bx|si|di)|r1[2-5][dwb]?|xmm(?:[6-9]|1[0-5]))\b`).MatchString(text) {
			t.Fatal("assembled x64 thunk uses an unpreserved nonvolatile register", disassembly)
		}
	}
	for _, register := range registers {
		if strings.Count(compact, "push"+register) != 1 || strings.Count(compact, "pop"+register) != 1 {
			t.Fatal("assembled thunk lacks an exact preserved-register save/restore pair", register, disassembly)
		}
	}
}

var (
	windowsStdCallARMGP   = regexp.MustCompile(`\b[xw]([0-9]+)\b`)
	windowsStdCallARMSIMD = regexp.MustCompile(`\b[vdqs](?:8|9|1[0-5])\b`)
)

type windowsStdCallInstruction struct {
	opcode, operands string
}

func windowsStdCallDecodeInstructions(arch, disassembly string) ([]windowsStdCallInstruction, error) {
	// llvm-objdump is invoked with --no-show-raw-insn. Only an address and
	// instruction may occupy these lines; file headers and symbol labels do not.
	line := regexp.MustCompile(`(?m)^\s*[0-9a-f]+:\s+([^\n]+)$`)
	opcode := regexp.MustCompile(`^[a-z][a-z0-9_.]*$`)
	symbol := regexp.MustCompile(`<[^>]*>`)
	var result []windowsStdCallInstruction
	for _, match := range line.FindAllStringSubmatch(strings.ToLower(disassembly), -1) {
		text, _, _ := strings.Cut(match[1], "//")
		text, _, _ = strings.Cut(text, ";")
		if arch != "arm64" {
			text, _, _ = strings.Cut(text, "#") // ARM64 '#' denotes an immediate.
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		if !opcode.MatchString(fields[0]) {
			return nil, fmt.Errorf("unrecognized actual thunk instruction %q", text)
		}
		operands := symbol.ReplaceAllString(strings.Join(fields[1:], " "), "")
		operands = regexp.MustCompile(`\bfp\b`).ReplaceAllString(operands, "x29")
		operands = regexp.MustCompile(`\blr\b`).ReplaceAllString(operands, "x30")
		result = append(result, windowsStdCallInstruction{fields[0], operands})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("actual thunk disassembly contains no decoded instructions")
	}
	return result, nil
}

func TestWindowsStdCallDisassemblyOperandBoundary(t *testing.T) {
	for _, test := range []struct {
		name, text string
		gp, simd   bool
	}{
		{"address and symbol", "00000000 <d8>:\n d8: ldr x0, [x1] // d8 x20\n dc: bl 0xd8 <d8> ; x20", false, false},
		{"real SIMD", " d8: fmov d8, d0", false, true},
		{"real GP", " d8: ldr x20, [x19]\n dc: blr x20", true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			instructions, err := windowsStdCallDecodeInstructions("arm64", test.text)
			if err != nil {
				t.Fatal(err)
			}
			gp, simd := false, false
			for _, instruction := range instructions {
				simd = simd || windowsStdCallARMSIMD.MatchString(instruction.operands)
				for _, match := range windowsStdCallARMGP.FindAllStringSubmatch(instruction.operands, -1) {
					index, err := strconv.Atoi(match[1])
					if err != nil {
						t.Fatal(err)
					}
					gp = gp || index >= 20 && index <= 28
				}
			}
			if gp != test.gp || simd != test.simd {
				t.Fatalf("decoded operand violation GP/SIMD=%v/%v, want %v/%v", gp, simd, test.gp, test.simd)
			}
		})
	}
}

// The actual C adapter executes against a mock native thunk on the host. It
// proves only the five/six-word conversion, address and error contracts.
func TestWindowsStdCallGo127HostBridge(t *testing.T) {
	clang, _ := windowsStdCallTools(t)
	dir := t.TempDir()
	adapter := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime", "_wrap", "stdcall_windows_go127.c")
	if _, err := os.Stat(adapter); err != nil {
		t.Fatal("missing real five-word Windows runtime bridge", err)
	}
	source := fmt.Sprintf("#include %q\n%s", filepath.ToSlash(adapter), windowsStdCallHostFixture)
	input := assemblyLanguageTestWrite(t, dir, "main.c", source)
	binary := filepath.Join(dir, "host-bridge")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	windowsStdCallCommand(t, clang, "-std=c11", "-Wall", "-Wextra", "-Werror", input, "-o", binary)
	windowsStdCallCommand(t, binary)
}

func TestWindowsStdCallGo127CompleteOwner(t *testing.T) {
	const pkg = "internal/runtime/syscall/windows"
	runtimeDir := env.LLGoRuntimeDir()
	dir := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkg))
	patch := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(pkg), "syscall_windows_go127.go")
	body, err := os.ReadFile(patch)
	if err != nil {
		t.Fatal("missing complete Go1.27 Windows source owner", err)
	}
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			ctx := sourcePatchBuildContext{goos: "windows", goarch: arch, goversion: runtime.Version()}
			apply := func(base map[string][]byte) (map[string][]byte, error) {
				changed, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, runtime.GOROOT(), pkg, ctx)
				if err == nil && !changed {
					return nil, fmt.Errorf("missing complete source owner")
				}
				return overlay, err
			}
			view, err := apply(nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, other := range []string{"386", "amd64", "arm64"} {
				filename := filepath.Join(dir, "asm_windows_"+other+".s")
				_, present := view[filename]
				if present != (other == arch) || present && string(view[filename]) != "// replaced by LLGo source patch\n" {
					t.Fatal("the owner did not replace exactly its complete selected assembly", other)
				}
			}
			generated := filepath.Join(dir, "z_llgo_patch_syscall_windows_go127.go")
			file, err := parser.ParseFile(token.NewFileSet(), generated, view[generated], 0)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
					seen[fn.Name.Name] = true
				}
			}
			if !seen["StdCall"] || !seen["AsmStdCallAddr"] {
				t.Fatal("selected public entries lack real Go bodies", seen)
			}
			if _, err := apply(view); err != nil {
				t.Fatal("complete owner reuse changed its contract", err)
			}
			original, err := os.ReadFile(filepath.Join(dir, "syscall_windows.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				name string
				base map[string][]byte
			}{
				{"missing body", map[string][]byte{patch: bytes.Replace(body, []byte("func StdCall("), []byte("func MissingStdCall("), 1)}},
				{"changed layout", map[string][]byte{filepath.Join(dir, "syscall_windows.go"): append(bytes.Clone(original), []byte("\nvar extraStdCall = StdCallInfo{}\n")...)}},
				{"new TEXT", map[string][]byte{filepath.Join(dir, "asm_windows_"+arch+".s"): []byte("TEXT ·Other(SB),$0-0\nRET\n")}},
				{"remaining private consumer", map[string][]byte{filepath.Join(dir, "syscall_windows.go"): append(bytes.Clone(original), []byte("\nfunc extra() { asmstdcall(nil) }\n")...)}},
			} {
				t.Run(test.name, func(t *testing.T) {
					if _, err := apply(test.base); err == nil {
						t.Fatal("accepted an incomplete or changed original source owner")
					}
				})
			}
			ctx.goversion = "go1.26.10"
			if changed, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), pkg, ctx); err != nil || changed {
				t.Fatal("Go1.27 owner changed the six-word legacy selection", err)
			}
		})
	}
}

const windowsStdCallHostFixture = `
static uintptr_t expected_n;
static uintptr_t expected_args;
static unsigned calls;

void llgo_windows_syscall(struct llgo_windows_syscall_info *call)
{
    if (call->fn != 0x1234 || call->n != expected_n || call->args != expected_args)
        __builtin_trap();
    calls++;
    call->r1 = 0x81726354;
    call->r2 = 0x19283746;
    call->err = (uintptr_t)-1;
}

int main(void)
{
    uintptr_t arguments[1026] = {0};
    const uintptr_t sizes[] = {0, 8, 12, 42, 43, 1026};
    for (unsigned i = 0; i < sizeof(sizes) / sizeof(sizes[0]); i++) {
        struct {
            uintptr_t before;
            struct llgo_windows_stdcall_info info;
            uintptr_t after;
        } block = {0x73928a51, {0x1234, sizes[i], (uintptr_t)arguments, 0, 0xa5a5}, 0x73928a51};
        expected_n = sizes[i];
        expected_args = (uintptr_t)arguments;
        uint32_t (*entry)(struct llgo_windows_stdcall_info *) =
            (uint32_t (*)(struct llgo_windows_stdcall_info *))llgo_windows_stdcall_go127_addr();
        if (entry != llgo_windows_stdcall_go127 || entry(&block.info) != UINT32_MAX)
            return 1;
        if (block.info.fn != 0x1234 || block.info.n != sizes[i] ||
            block.info.args != expected_args || block.info.r1 != 0x81726354)
            return 2;
#if defined(__aarch64__)
        if (block.info.r2 != 0xa5a5)
            return 3;
#else
        if (block.info.r2 != 0x19283746)
            return 3;
#endif
        if (block.before != 0x73928a51 || block.after != 0x73928a51)
            return 4;
    }
    return calls == 6 ? 0 : 5;
}
`
