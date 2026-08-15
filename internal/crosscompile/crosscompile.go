package crosscompile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile/compile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/flash"
	"github.com/xgo-dev/llgo/internal/llvmpayload"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/targets"
	"github.com/xgo-dev/llgo/internal/xtool/llvm"
	envllvm "github.com/xgo-dev/llgo/xtool/env/llvm"
	gllvm "github.com/xgo-dev/llvm"
)

type Export struct {
	CC      string // C compiler executable
	CCArgs  []string
	CXX     string // C++ compiler executable
	CXXArgs []string
	CCFLAGS []string
	CFLAGS  []string
	LDFLAGS []string
	// CompilerIdentity values are stable version strings used to keep cached
	// packages separate when an explicit native toolchain changes.
	CCIdentity  string
	CXXIdentity string
	// Toolchain records the ABI and flag dialect selected for native-format
	// outputs. It is independent of the host shell and remains empty for named
	// embedded/WebAssembly targets that drive their linker directly.
	Toolchain NativeToolchain

	// Additional fields from target configuration
	BuildTags      []string
	GOOS           string
	GOARCH         string
	Libc           string
	Linker         string // Linker to use (e.g., "ld.lld", "avr-ld")
	LinkerArgs     []string
	LinkerIdentity string
	ExtraFiles     []string // Extra files to compile and link (e.g., .s, .c files)
	ClangRoot      string   // Root directory of custom clang installation
	ClangBinPath   string   // Path to clang binary directory

	LLVMTarget   string // LLVM Target
	CPU          string // LLVM target CPU used by external code generation and cache identity
	Features     string // LLVM target features used by external code generation and cache identity
	TargetABI    string // RISC-V Target ABI (e.g., "lp64", "lp64d")
	BinaryFormat string // Binary format (e.g., "elf", "esp", "uf2")
	FormatDetail string // For uf2, it's uf2FamilyID
	Emulator     string // Emulator command template (e.g., "qemu-system-arm -M {} -kernel {}")
	DebugInfo    DebugInfoPolicy

	// Flashing/Debugging configuration
	Device flash.Device // Device configuration for flashing/debugging
}

// NativeToolchain describes the externally visible ABI and the tools used to
// produce a native object. Keep these choices explicit: in particular, a
// Windows host does not imply either the MSVC or MinGW ABI.
type NativeToolchain struct {
	ABI            PlatformABI
	ObjectFormat   ObjectFormat
	Driver         DriverFlavor
	Linker         LinkerFlavor
	TargetTriple   string
	CRT            CRTFlavor
	CXXRuntime     CXXRuntimeFlavor
	SDKVersion     string
	CRTVersion     string
	ToolsetVersion string
}

type PlatformABI string

const (
	PlatformABIUnknown PlatformABI = ""
	PlatformABIGNU     PlatformABI = "gnu"
	PlatformABIDarwin  PlatformABI = "darwin"
	PlatformABIMsvc    PlatformABI = "msvc"
)

type ObjectFormat string

const (
	ObjectFormatUnknown ObjectFormat = ""
	ObjectFormatELF     ObjectFormat = "elf"
	ObjectFormatMachO   ObjectFormat = "macho"
	ObjectFormatCOFF    ObjectFormat = "coff"
)

type DriverFlavor string

const (
	DriverFlavorUnknown  DriverFlavor = ""
	DriverFlavorClangGNU DriverFlavor = "clang"
)

type LinkerFlavor string

const (
	LinkerFlavorUnknown  LinkerFlavor = ""
	LinkerFlavorELFLLD   LinkerFlavor = "ld.lld"
	LinkerFlavorMachO    LinkerFlavor = "ld64.lld"
	LinkerFlavorCOFFLLD  LinkerFlavor = "lld-link"
	LinkerFlavorMinGWLLD LinkerFlavor = "mingw-lld"
)

type CRTFlavor string

const (
	CRTFlavorUnknown CRTFlavor = ""
	CRTFlavorUCRT    CRTFlavor = "ucrt"
)

type CXXRuntimeFlavor string

const (
	CXXRuntimeUnknown CXXRuntimeFlavor = ""
	CXXRuntimeMSVC    CXXRuntimeFlavor = "msvc"
)

// DebugInfoPolicy describes how a selected linker handles debug information.
// Build orchestration consumes this typed capability instead of inferring it
// from a target name or linker executable.
type DebugInfoPolicy struct {
	AlwaysOmit        bool
	OmitLinkFlags     []string
	PreserveLinkFlags []string
}

func nativeToolchain(goos string) NativeToolchain {
	switch goos {
	case "darwin":
		return NativeToolchain{
			ABI:          PlatformABIDarwin,
			ObjectFormat: ObjectFormatMachO,
			Driver:       DriverFlavorClangGNU,
			Linker:       LinkerFlavorMachO,
		}
	case "linux":
		return NativeToolchain{
			ABI:          PlatformABIGNU,
			ObjectFormat: ObjectFormatELF,
			Driver:       DriverFlavorClangGNU,
			Linker:       LinkerFlavorELFLLD,
		}
	case "windows":
		return NativeToolchain{
			ABI:          PlatformABIMsvc,
			ObjectFormat: ObjectFormatCOFF,
			Driver:       DriverFlavorClangGNU,
			Linker:       LinkerFlavorCOFFLLD,
		}
	default:
		return NativeToolchain{}
	}
}

func nativeDebugInfoPolicy(toolchain NativeToolchain) DebugInfoPolicy {
	switch toolchain.Linker {
	case LinkerFlavorMachO, LinkerFlavorELFLLD, LinkerFlavorMinGWLLD:
		return DebugInfoPolicy{OmitLinkFlags: []string{"-Wl,-S"}}
	case LinkerFlavorCOFFLLD:
		return DebugInfoPolicy{
			OmitLinkFlags:     []string{"-Wl,/debug:none"},
			PreserveLinkFlags: []string{"-Wl,/debug:dwarf"},
		}
	default:
		return DebugInfoPolicy{}
	}
}

// URLs and configuration that can be overridden for testing
var (
	wasiSdkUrl      = "https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-25/wasi-sdk-25.0-x86_64-macos.tar.gz"
	wasiMacosSubdir = "wasi-sdk-25.0-x86_64-macos"
)

var (
	espClangWindowsBaseUrl = "https://github.com/espressif/llvm-project/releases/download/esp-19.1.2_20250312"
	espClangWindowsVersion = "19.1.2_20250312"
)

const espClangWindowsPlatform = "x86_64-w64-mingw32"

// cacheRoot can be overridden for testing
var cacheRoot = env.LLGoCacheDir

var resolveESPClangArtifact = espClangArtifact

func espClangArtifact(payload llvmpayload.Manifest, platform string) (llvmpayload.Artifact, error) {
	if platform == espClangWindowsPlatform {
		filename := fmt.Sprintf("clang-esp-%s-%s.tar.xz", espClangWindowsVersion, platform)
		return llvmpayload.Artifact{
			Platform: platform,
			Version:  espClangWindowsVersion,
			URL:      espClangWindowsBaseUrl + "/" + filename,
		}, nil
	}
	return payload.Artifact(platform)
}

func cacheDir() string {
	return filepath.Join(cacheRoot(), "crosscompile")
}

// buildEnvMap creates a map of template variables for the current context
func buildEnvMap(llgoRoot string) map[string]string {
	envs := make(map[string]string)

	// Basic paths
	envs["root"] = llgoRoot
	envs["tmpDir"] = os.TempDir()

	// These will typically be set by calling code when actual values are known
	// envs["port"] = ""     // Serial port (e.g., "/dev/ttyUSB0", "COM3")
	// envs["hex"] = ""      // Path to hex file
	// envs["bin"] = ""      // Path to binary file
	// envs["img"] = ""      // Path to image file
	// envs["zip"] = ""      // Path to zip file

	return envs
}

// getCanonicalArchName returns the canonical architecture name for a target triple
func getCanonicalArchName(triple string) string {
	arch := strings.Split(triple, "-")[0]
	if arch == "arm64" {
		return "aarch64"
	}
	if strings.HasPrefix(arch, "arm") || strings.HasPrefix(arch, "thumb") {
		return "arm"
	}
	if arch == "mipsel" {
		return "mips"
	}
	return arch
}

// getMacOSSysroot returns the macOS SDK path using xcrun
func getMacOSSysroot() (string, error) {
	cmd := exec.Command("xcrun", "--sdk", "macosx", "--show-sdk-path")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getESPClangRoot returns the ESP Clang root directory, checking LLGoROOT first,
// then downloading if needed and platform is supported
func getESPClangRoot(forceEspClang bool) (clangRoot string, err error) {
	llgoRoot := env.LLGoROOT()

	// First check if clang exists in LLGoROOT
	espClangRoot := filepath.Join(llgoRoot, envllvm.CrosscompileClangPath)
	if _, err = os.Stat(espClangRoot); err == nil {
		clangRoot = espClangRoot
		return
	}

	if !forceEspClang {
		return "", nil
	}

	payload, err := llvmpayload.ForLLVMVersion(gllvm.Version)
	if err != nil {
		return "", err
	}

	// Try to download ESP Clang if platform is supported
	platformSuffix := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	if platformSuffix != "" {
		artifact, artifactErr := resolveESPClangArtifact(payload, platformSuffix)
		if artifactErr != nil {
			return "", artifactErr
		}
		cacheClangDir := filepath.Join(cacheRoot(), "crosscompile", "esp-clang-"+artifact.Version)
		if _, err = os.Stat(cacheClangDir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return
			}
			fmt.Fprintln(os.Stderr, "ESP Clang not found in LLGO_ROOT or cache, will download.")
			if err = checkDownloadAndExtractESPClang(artifact, cacheClangDir); err != nil {
				return
			}
		}
		clangRoot = cacheClangDir
		return
	}

	err = fmt.Errorf("ESP Clang not found in LLGoROOT and platform %s/%s is not supported for download", runtime.GOOS, runtime.GOARCH)
	return
}

// getESPClangPlatform returns the platform suffix for ESP Clang downloads
func getESPClangPlatform(goos, goarch string) string {
	if goos == "windows" && (goarch == "amd64" || goarch == "arm64") {
		// Espressif publishes an x86-64 Windows host toolchain. Windows on
		// ARM64 runs it through the system's x64 emulation layer.
		return espClangWindowsPlatform
	}
	platform, _ := llvmpayload.PlatformSuffix(goos, goarch)
	return platform
}

// ldFlagsFromFileName extracts the library name from a filename for use in linker flags
// For example, "libmath.a" becomes "math" for use with "-lmath"
func ldFlagsFromFileName(fileName string) string {
	return strings.TrimPrefix(strings.TrimSuffix(fileName, ".a"), "lib")
}

func lldLTOOptFlag(level optlevel.Level) (string, error) {
	switch level {
	case optlevel.O0, optlevel.O1, optlevel.O2, optlevel.O3:
		return "--lto-" + level.Name(), nil
	case optlevel.Os, optlevel.Oz:
		// ld.lld only accepts numeric LTO optimization levels. Clang maps its
		// size-oriented modes to O2 for the link-time optimization pipeline.
		return "--lto-O2", nil
	default:
		return "", fmt.Errorf("invalid LTO optimization level %q", level)
	}
}

// compileWithConfig compiles libraries according to the provided configuration
// and returns the necessary linker flags for linking against the compiled libraries
func compileWithConfig(
	compileConfig compile.CompileConfig,
	outputDir string, options compile.CompileOptions,
) (ldflags []string, err error) {
	if err = os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create compiled library cache %q: %w", outputDir, err)
	}
	ldflags = append(ldflags, "-nostdlib", "-L"+outputDir)

	for _, group := range compileConfig.Groups {
		err = group.Compile(outputDir, options)
		if err != nil {
			break
		}
		if filepath.Ext(group.OutputFileName) == ".o" {
			continue
		}
		ldflags = append(ldflags, "-l"+ldFlagsFromFileName(group.OutputFileName))
	}
	return
}

// ltoLinkerOptFlag maps LLGo's optimization level to lld's numeric LTO
// optimizer level. lld accepts only O0 through O3 here; Os/Oz are carried by
// the optsize/minsize function attributes in LLGo's bitcode and use O2 as
// LLVM's corresponding speed level.
func ltoLinkerOptFlag(level optlevel.Level) string {
	switch level {
	case optlevel.O0, optlevel.O1, optlevel.O2, optlevel.O3:
		return "--lto-" + level.Name()
	case optlevel.Os, optlevel.Oz:
		return "--lto-O2"
	default:
		return ""
	}
}

func nativeLLDFlags(toolchain NativeToolchain, level optlevel.Level, ltoMode lto.Mode) []string {
	flags := []string{"-fuse-ld=lld"}
	if toolchain.Linker == LinkerFlavorCOFFLLD {
		flags = append(flags,
			"-Wl,/errorlimit:0",
			// Go requires distinct functions to have distinct PCs. lld-link
			// enables identical COMDAT folding through /opt:icf, so keep it
			// disabled even when dead-code elimination is enabled.
			"-Wl,/opt:noicf",
		)
	} else {
		flags = append(flags,
			"-Wl,--error-limit=0",
			// lld's safe mode still folds llgo-emitted same-body functions.
			"-Wl,--icf=none",
		)
	}
	if !ltoMode.Enabled() {
		return flags
	}

	flags = append(flags, ltoMode.ClangFlag())
	if toolchain.Linker == LinkerFlavorCOFFLLD {
		flags = append(flags, "-Wl,/opt:lldlto="+coffLTOLevel(level))
	} else if optFlag := ltoLinkerOptFlag(level); optFlag != "" {
		flags = append(flags, "-Wl,"+optFlag)
	}
	return flags
}

func coffLTOLevel(level optlevel.Level) string {
	switch level {
	case optlevel.O0:
		return "0"
	case optlevel.O1:
		return "1"
	case optlevel.O3:
		return "3"
	default:
		// lld-link accepts only 0 through 3. -Os and -Oz remain encoded in
		// the input IR through optsize/minsize attributes; use its normal
		// optimization pipeline for the link-wide setting.
		return "2"
	}
}

func nativeSectionFlags(toolchain NativeToolchain) (ccflags, ldflags []string) {
	switch toolchain.ObjectFormat {
	case ObjectFormatMachO:
		return nil, []string{"-Xlinker", "-dead_strip"}
	case ObjectFormatCOFF:
		if toolchain.ABI == PlatformABIGNU {
			return []string{"-fdata-sections", "-ffunction-sections"}, []string{
				"-fdata-sections",
				"-ffunction-sections",
				"-Wl,--gc-sections",
			}
		}
		return []string{"-fdata-sections", "-ffunction-sections"}, []string{
			"-fdata-sections",
			"-ffunction-sections",
			"--rtlib=compiler-rt",
			"-Wl,/opt:ref",
			// UCRT defines printf-family entry points inline in its headers.
			// LLGo C linknames use the traditional external symbols supplied by
			// this Microsoft compatibility import library.
			"-llegacy_stdio_definitions",
		}
	default:
		return []string{"-fdata-sections", "-ffunction-sections"}, []string{
			"-fdata-sections",
			"-ffunction-sections",
			"-Xlinker",
			"--gc-sections",
			"-latomic",
			// libpthread & libdl is built-in since glibc 2.34 (2021-08-01);
			// retain these flags for older supported systems.
			"-lpthread",
			"-ldl",
		}
	}
}

func use(goos, goarch string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool) (Export, error) {
	return useWithGOARM(goos, goarch, "", wasiThreads, forceEspClang, level, ltoMode, goGlobalDCE)
}

func useWithGOARM(goos, goarch, goarm string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool) (export Export, err error) {
	return useWithGOARMAndToolchain(goos, goarch, goarm, wasiThreads, forceEspClang, level, ltoMode, goGlobalDCE, NativeToolchainInput{})
}

func useWithGOARMAndToolchain(goos, goarch, goarm string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool, nativeInput NativeToolchainInput) (export Export, err error) {
	targetTriple := llvm.GetTargetTripleWithGOARM(goos, goarch, goarm)
	llgoRoot := env.LLGoROOT()
	nativePlatformToolchain := usesNativePlatformToolchain(
		runtime.GOOS, runtime.GOARCH, goos, goarch, nativeInput.ResolveWindows,
	)

	// Linked Windows output resolves its compiler and dependencies as one
	// coherent profile below, including when the LLGo host is macOS or Linux.
	// Do not inspect an unrelated embedded Clang before that profile resolves.
	var clangRoot string
	if !(nativePlatformToolchain && goos == "windows") {
		clangRoot, err = getESPClangRoot(forceEspClang)
		if err != nil {
			return
		}
	}

	// Set ClangRoot and CC if clang is available
	export.ClangRoot = clangRoot
	if clangRoot != "" {
		export.CC = filepath.Join(clangRoot, "bin", "clang++")
	} else {
		export.CC = "clang++"
	}

	if nativePlatformToolchain {
		if goos == "windows" {
			export, err = resolveWindowsToolchain(goarch, nativeInput, probeNativeTool)
			if err != nil {
				return
			}
			targetTriple = export.Toolchain.TargetTriple
			// Native Windows tool selection is authoritative. An embedded
			// toolchain cached below LLGO_ROOT must not leak include or library
			// paths into this profile.
			clangRoot = ""
			export.ClangRoot = ""
		} else {
			export.Toolchain = nativeToolchain(goos)
		}
		export.DebugInfo = nativeDebugInfoPolicy(export.Toolchain)
		// not cross compile
		// Set up basic flags for non-cross-compile
		externalLDFlags := slices.Clone(export.LDFLAGS)
		export.LDFLAGS = []string{
			"-target", targetTriple,
			"-Qunused-arguments",
			"-Wno-unused-command-line-argument",
		}
		export.LDFLAGS = append(export.LDFLAGS, nativeLLDFlags(export.Toolchain, level, ltoMode)...)
		if clangRoot != "" {
			clangLib := filepath.Join(clangRoot, "lib")
			clangInc := filepath.Join(clangRoot, "include")
			export.CFLAGS = append(export.CFLAGS, "-I"+clangInc)
			export.LDFLAGS = append(export.LDFLAGS, "-L"+clangLib)
			// Add platform-specific rpath flags
			switch goos {
			case "darwin":
				export.LDFLAGS = append(export.LDFLAGS, "-Wl,-rpath,"+clangLib)
			case "linux":
				export.LDFLAGS = append(export.LDFLAGS, "-Wl,-rpath,"+clangLib)
			case "windows":
				// Windows doesn't support rpath, DLLs should be in PATH or same directory
			default:
				// For other Unix-like systems, try the generic rpath
				export.LDFLAGS = append(export.LDFLAGS, "-Wl,-rpath,"+clangLib)
			}
		}
		export.CCFLAGS = []string{
			level.Flag(),
			"-target", targetTriple,
			"-Qunused-arguments",
			"-Wno-unused-command-line-argument",
			// Keep frame pointers in C code too: the runtime's physical
			// unwinder walks fault-site chains through C frames (Go keeps
			// them via the "frame-pointer"="non-leaf" attribute; x86-64 C
			// would omit them at -O by default).
			"-fno-omit-frame-pointer",
		}
		if ltoMode.Enabled() {
			export.CCFLAGS = append(export.CCFLAGS, ltoMode.ClangFlag())
		}
		if ltoMode == lto.Full && goGlobalDCE {
			export.CCFLAGS = append(export.CCFLAGS, "-fvirtual-function-elimination", "-fwhole-program-vtables")
		}

		// Add sysroot for macOS only
		if goos == "darwin" {
			sysrootPath, sysrootErr := getMacOSSysroot()
			if sysrootErr != nil {
				err = fmt.Errorf("failed to get macOS SDK path: %w", sysrootErr)
				return
			}
			export.CCFLAGS = append(export.CCFLAGS, []string{"--sysroot=" + sysrootPath}...)
			export.LDFLAGS = append(export.LDFLAGS, []string{"--sysroot=" + sysrootPath}...)
		}

		ccflags, ldflags := nativeSectionFlags(export.Toolchain)
		export.CCFLAGS = append(export.CCFLAGS, ccflags...)
		export.LDFLAGS = append(export.LDFLAGS, ldflags...)
		export.LDFLAGS = append(export.LDFLAGS, externalLDFlags...)
		return
	}
	if goarch != "wasm" {
		return
	}
	export.DebugInfo.OmitLinkFlags = []string{"-Wl,-S"}

	// Configure based on GOOS
	switch goos {
	case "wasip1":
		// Set wasiSdkRoot path
		wasiSdkRoot := filepath.Join(llgoRoot, "crosscompile", "wasi-libc")

		// If not exists in LLGoROOT, download and use cached wasiSdkRoot
		if _, err = os.Stat(wasiSdkRoot); err != nil {
			sdkDir := filepath.Join(cacheDir(), llvm.GetTargetTriple(goos, goarch))
			if wasiSdkRoot, err = checkDownloadAndExtractWasiSDK(sdkDir); err != nil {
				return
			}
		}
		// WASI-SDK configuration
		triple := "wasm32-wasip1"
		if wasiThreads {
			triple = "wasm32-wasip1-threads"
		}

		// Set up flags for the WASI-SDK or wasi-libc
		sysrootDir := filepath.Join(wasiSdkRoot, "share", "wasi-sysroot")
		libclangDir := filepath.Join(wasiSdkRoot, "lib", "clang", "19")
		includeDir := filepath.Join(sysrootDir, "include", triple)
		libDir := filepath.Join(sysrootDir, "lib", triple)

		// Use system clang and sysroot of wasi-sdk
		// Add compiler flags
		export.CCFLAGS = []string{
			level.Flag(),
			"-target", targetTriple,
			"--sysroot=" + sysrootDir,
			"-resource-dir=" + libclangDir,
			"-matomics",
			"-mbulk-memory",
		}
		export.CFLAGS = []string{
			"-I" + includeDir,
			"-Qunused-arguments",
			"-Wno-unused-command-line-argument",
		}
		// Add WebAssembly linker flags
		export.LDFLAGS = append(export.LDFLAGS, export.CCFLAGS...)
		export.LDFLAGS = append(export.LDFLAGS, []string{
			"-Wno-override-module",
			"-Wl,--error-limit=0",
			"-L" + libDir,
			"-Wl,--allow-undefined",
			"-Wl,--import-memory,", // unknown import: `env::memory` has not been defined
			"-Wl,--export-memory",
			"-Wl,--initial-memory=67108864", // 64MB
			"-mbulk-memory",
			"-mmultimemory",
			"-z", "stack-size=10485760", // 10MB
			"-Wl,--export=malloc", "-Wl,--export=free",
			"-lc",
			"-lcrypt",
			"-lm",
			"-lrt",
			"-lutil",
			"-lsetjmp",
			"-lwasi-emulated-mman",
			"-lwasi-emulated-getpid",
			"-lwasi-emulated-process-clocks",
			"-lwasi-emulated-signal",
			"-fwasm-exceptions",
			"-mllvm", "-wasm-enable-sjlj",
		}...)
		// Add thread support if enabled
		if wasiThreads {
			export.CCFLAGS = append(
				export.CCFLAGS,
				"-pthread",
			)
			export.LDFLAGS = append(export.LDFLAGS, export.CCFLAGS...)
			export.LDFLAGS = append(
				export.LDFLAGS,
				"-lwasi-emulated-pthread",
				"-lpthread",
			)
		}

	case "js":
		targetTriple := "wasm32-unknown-emscripten"
		// Emscripten configuration using system installation
		// Specify emcc as the compiler
		export.CC = "emcc"
		// Add compiler flags
		export.CCFLAGS = []string{
			level.Flag(),
			"-target", targetTriple,
			"-Qunused-arguments",
			"-Wno-unused-command-line-argument",
		}
		export.CFLAGS = []string{}
		// Add WebAssembly linker flags for Emscripten
		export.LDFLAGS = []string{
			"-target", targetTriple,
			"-Wno-override-module",
			"-Wl,--error-limit=0",
			"-s", "ALLOW_MEMORY_GROWTH=1",
			"-Wl,--allow-undefined",
			// "-Wl,--import-memory,",
			// "-Wl,--export-memory",
			// "-Wl,--initial-memory=67108864", // 64MB
			// "-mbulk-memory",
			// "-mmultimemory",
			// "-z", "stack-size=10485760", // 10MB
			// "-Wl,--export=malloc", "-Wl,--export=free",
		}
		export.LDFLAGS = append(export.LDFLAGS, []string{
			"-sENVIRONMENT=web,worker",
			"-DPLATFORM_WEB",
			"-sEXPORT_KEEPALIVE=1",
			"-sEXPORT_ES6=1",
			"-sALLOW_MEMORY_GROWTH=1",
			"-sRESERVED_FUNCTION_POINTERS=1",
			"-sEXPORTED_RUNTIME_METHODS=cwrap,allocateUTF8,stringToUTF8,UTF8ToString,FS,setValue,getValue",
			"-sWASM=1",
			"-sEXPORT_ALL=1",
			"-sASYNCIFY=1",
			"-sSTACK_SIZE=5242880", // 50MB
		}...)

	default:
		err = errors.New("unsupported GOOS for WebAssembly: " + goos)
		return
	}
	return
}

func usesNativePlatformToolchain(hostGOOS, hostGOARCH, targetGOOS, targetGOARCH string, resolveWindows bool) bool {
	if targetGOOS == "windows" && resolveWindows {
		return true
	}
	return hostGOOS == targetGOOS && hostGOARCH == targetGOARCH
}

// UseTarget loads configuration from a target name (e.g., "rp2040", "wasi")
func UseTarget(targetName string, level optlevel.Level, ltoMode lto.Mode) (export Export, err error) {
	resolver := targets.NewDefaultResolver()

	config, err := resolver.Resolve(targetName)
	if err != nil {
		return export, fmt.Errorf("failed to resolve target %s: %w", targetName, err)
	}

	target := config.LLVMTarget
	if target == "" {
		return export, fmt.Errorf("target '%s' does not have a valid LLVM target triple", targetName)
	}

	cpu := config.CPU
	if cpu == "" {
		return export, fmt.Errorf("target '%s' does not have a valid CPU configuration", targetName)
	}

	// Espressif's Windows toolchain only ships the ESP backends. Use the
	// full MSYS2 LLVM distribution for other embedded targets (for example
	// ARM and AVR), while retaining the established ESP toolchain selection
	// on Unix hosts and for ESP targets.
	var clangRoot string
	if useSystemClangForTarget(runtime.GOOS, target, config.BuildTags) {
		export.CC = "clang++"
	} else {
		var clangErr error
		clangRoot, clangErr = getESPClangRoot(true)
		if clangErr != nil {
			err = clangErr
			return
		}
		export.ClangRoot = clangRoot
		export.CC = filepath.Join(clangRoot, "bin", "clang++")
	}

	// Convert target config to Export - only export necessary fields
	export.BuildTags = config.BuildTags
	export.GOOS = config.GOOS
	export.GOARCH = config.GOARCH
	export.ExtraFiles = config.ExtraFiles
	export.LLVMTarget = config.LLVMTarget
	export.CPU = config.CPU
	export.Features = config.Features
	export.TargetABI = config.TargetABI
	export.BinaryFormat = config.BinaryFormat
	export.FormatDetail = config.FormatDetail()
	export.Emulator = config.Emulator
	export.DebugInfo.AlwaysOmit = true

	// Set flashing/debugging configuration
	export.Device = flash.Device{
		Serial:     config.Serial,
		SerialPort: config.SerialPort,
		Flash: flash.Flash{
			Method:            config.FlashMethod,
			Command:           config.FlashCommand,
			Flash1200BpsReset: config.Flash1200BpsReset == "true",
		},
		MSD: flash.MSD{
			VolumeName:   config.MSDVolumeName,
			FirmwareName: config.MSDFirmwareName,
		},
		OpenOCD: flash.OpenOCD{
			Interface: config.OpenOCDInterface,
			Transport: config.OpenOCDTransport,
			Target:    config.OpenOCDTarget,
		},
	}

	// Build environment map for template variable expansion
	envs := buildEnvMap(env.LLGoROOT())

	// Convert LLVMTarget, CPU, Features to CCFLAGS/LDFLAGS
	// ICF off for Go pc-identity semantics (see the non-cross flags above).
	ldflags := []string{"-S", "--icf=none"}
	ccflags := []string{level.Flag()}
	cflags := []string{"-Wno-override-module", "-Qunused-arguments", "-Wno-unused-command-line-argument"}
	clangTarget := clangDriverTargetForHost(runtime.GOOS, config.LLVMTarget, config.BuildTags)
	if clangTarget != "" {
		cflags = append(cflags, "--target="+clangTarget)
		ccflags = append(ccflags, "--target="+clangTarget)
	}
	// Expand template variables in cflags
	expandedCFlags := env.ExpandEnvSlice(config.CFlags, envs)
	cflags = append(cflags, expandedCFlags...)

	if config.Linker == "ld.lld" && ltoMode.Enabled() {
		if optFlag := ltoLinkerOptFlag(level); optFlag != "" {
			ldflags = append(ldflags, optFlag)
		}
		cflags = append(cflags, ltoMode.ClangFlag())
		ccflags = append(ccflags, ltoMode.ClangFlag())
	}

	// The following parameters are inspired by tinygo/builder/library.go
	// Handle CPU configuration
	if cpu != "" {
		// X86 has deprecated the -mcpu flag, so we need to use -march instead.
		// However, ARM has not done this.
		if strings.HasPrefix(target, "i386") || strings.HasPrefix(target, "x86_64") {
			ccflags = append(ccflags, "-march="+cpu)
		} else if strings.HasPrefix(target, "avr") {
			ccflags = append(ccflags, "-mmcu="+cpu)
		} else {
			ccflags = append(ccflags, "-mcpu="+cpu)
		}

		// For ld.lld linker, also add CPU info to linker flags
		if config.Linker == "ld.lld" {
			ldflags = append(ldflags, "-mllvm", "-mcpu="+cpu)
		}
	}

	// Handle architecture-specific flags
	canonicalArch := getCanonicalArchName(target)
	switch canonicalArch {
	case "arm":
		if strings.Split(target, "-")[2] == "linux" {
			ccflags = append(ccflags, "-fno-unwind-tables", "-fno-asynchronous-unwind-tables")
		} else {
			ccflags = append(ccflags, "-fshort-enums", "-fomit-frame-pointer", "-mfloat-abi=soft", "-fno-unwind-tables", "-fno-asynchronous-unwind-tables")
		}
	case "avr":
		// AVR defaults to C float and double both being 32-bit. This deviates
		// from what most code (and certainly compiler-rt) expects. So we need
		// to force the compiler to use 64-bit floating point numbers for
		// double.
		ccflags = append(ccflags, "-mdouble=64")
	case "riscv32":
		// Check llvm-target to distinguish ESP RISC-V chips from others
		// ESP series (riscv32-esp-elf) only supports RV32IMC (no A/D/F extensions)
		// Other RISC-V32 targets support RV32IMAC (with A extension)
		if config.LLVMTarget == "riscv32-esp-elf" {
			ccflags = append(ccflags, "-march=rv32imc")
		} else {
			ccflags = append(ccflags, "-march=rv32imac")
		}
		ccflags = append(ccflags, "-fforce-enable-int128")
	case "riscv64":
		ccflags = append(ccflags, "-march=rv64gc")
		// codegen option should be added to ldflags for lto
		ldflags = append(ldflags, "-mllvm", "-march=rv64gc")
	case "mips":
		ccflags = append(ccflags, "-fno-pic")
	}

	// Handle soft float
	if strings.Contains(config.Features, "soft-float") || strings.Contains(strings.Join(config.CFlags, " "), "soft-float") {
		// Use softfloat instead of floating point instructions. This is
		// supported on many architectures.
		ccflags = append(ccflags, "-msoft-float")
	} else {
		if strings.HasPrefix(target, "armv5") {
			// On ARMv5 we need to explicitly enable hardware floating point
			// instructions: Clang appears to assume the hardware doesn't have a
			// FPU otherwise.
			ccflags = append(ccflags, "-mfpu=vfpv2")
		}
	}

	// Handle Features
	if config.Features != "" {
		// Only add -mllvm flags for non-WebAssembly linkers
		if config.Linker == "ld.lld" {
			ldflags = append(ldflags, "-mllvm", "-mattr="+config.Features)
		}
	}

	// Handle code generation configuration
	if config.CodeModel != "" {
		ccflags = append(ccflags, "-mcmodel="+config.CodeModel)
		if ltoMode.Enabled() {
			// codegen option should be added to ldflags for lto
			ldflags = append(ldflags, "-mllvm", "-code-model="+config.CodeModel)
		}
	}
	if config.TargetABI != "" {
		ccflags = append(ccflags, "-mabi="+config.TargetABI)
		if ltoMode.Enabled() {
			// codegen option should be added to ldflags for lto
			ldflags = append(ldflags, "-mllvm", "-target-abi="+config.TargetABI)
		}
	}
	if config.RelocationModel != "" {
		switch config.RelocationModel {
		case "pic":
			ccflags = append(ccflags, "-fPIC")
		case "static":
			ccflags = append(ccflags, "-fno-pic")
		}
	}

	// Handle Linker - keep it for external usage
	if config.Linker != "" {
		export.Linker = config.Linker
		if clangRoot != "" {
			export.Linker = filepath.Join(clangRoot, "bin", config.Linker)
		}
	}
	if config.LinkerScript != "" {
		ldflags = append(ldflags, "-T", config.LinkerScript)
	}
	var libcIncludeDir []string
	var compiledLibraryKey string
	if config.Libc != "" || config.RTLib != "" {
		var compilerKey string
		compilerKey, err = compilerCacheKey(export.CC)
		if err != nil {
			return
		}
		compiledLibraryKey = compiledLibraryCacheKey(compilerKey, ccflags, ldflags)
	}
	ldflags = append(ldflags, "-L", env.LLGoROOT()) // search targets/*.ld

	if config.Libc != "" {
		var outputDir string
		var libcLDFlags []string
		var compileConfig compile.CompileConfig
		baseDir := filepath.Join(cacheRoot(), "crosscompile")

		outputDir, compileConfig, err = getLibcCompileConfigByName(baseDir, config.Libc, config.LLVMTarget, config.CPU, compiledLibraryKey)
		if err != nil {
			return
		}
		libcLDFlags, err = compileWithConfig(compileConfig, outputDir, compile.CompileOptions{
			CC:      export.CC,
			Linker:  export.Linker,
			CCFLAGS: ccflags,
			LDFLAGS: ldflags,
		})
		if err != nil {
			return
		}
		cflags = append(cflags, compileConfig.ExportCFlags...)
		ldflags = append(ldflags, libcLDFlags...)

		libcIncludeDir = compileConfig.ExportCFlags
		export.Libc = config.Libc
	}

	if config.RTLib != "" {
		var outputDir string
		var rtLibLDFlags []string
		var compileConfig compile.CompileConfig
		baseDir := filepath.Join(cacheRoot(), "crosscompile")

		outputDir, compileConfig, err = getRTCompileConfigByName(baseDir, config.RTLib, config.LLVMTarget, compiledLibraryKey)
		if err != nil {
			return
		}
		rtLibLDFlags, err = compileWithConfig(compileConfig, outputDir, compile.CompileOptions{
			CC:      export.CC,
			Linker:  export.Linker,
			CCFLAGS: ccflags,
			LDFLAGS: ldflags,
			CFLAGS:  libcIncludeDir,
		})
		if err != nil {
			return
		}
		ldflags = append(ldflags, rtLibLDFlags...)
	}

	// Combine with config flags and expand template variables
	export.CFLAGS = cflags
	export.CCFLAGS = ccflags
	expandedLDFlags := env.ExpandEnvSlice(config.LDFlags, envs)
	export.LDFLAGS = append(ldflags, expandedLDFlags...)

	return export, nil
}

func useSystemClangForTarget(hostGOOS, targetTriple string, buildTags []string) bool {
	if hostGOOS != "windows" || strings.HasPrefix(targetTriple, "xtensa") {
		return false
	}
	for _, tag := range buildTags {
		if tag == "esp" {
			return false
		}
	}
	return true
}

// clangDriverTargetForHost returns the target spelling accepted by the host
// Clang driver. LLGo's Unix ESP toolchains use the historical "xtensa"
// spelling, but Espressif's official Windows distribution selects its Xtensa
// multilibs using the canonical GCC-compatible triple.
func clangDriverTargetForHost(hostGOOS, llvmTarget string, buildTags []string) string {
	if hostGOOS == "windows" && llvmTarget == "xtensa" {
		for _, tag := range buildTags {
			if tag == "esp" {
				return "xtensa-esp-unknown-elf"
			}
		}
	}
	return llvmTarget
}

// Use extends the original Use function to support target-based configuration
// If targetName is provided, it takes precedence over goos/goarch
func Use(goos, goarch, targetName string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool) (export Export, err error) {
	return UseWithGOARM(goos, goarch, "", targetName, wasiThreads, forceEspClang, level, ltoMode, goGlobalDCE)
}

// UseWithGOARM is Use with an explicit Go ARM architecture setting. The
// setting affects native GOARCH=arm clang and linker triples; named targets
// retain their target configuration's LLVM triple.
func UseWithGOARM(goos, goarch, goarm, targetName string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool) (export Export, err error) {
	return UseWithGOARMAndToolchain(goos, goarch, goarm, targetName, wasiThreads, forceEspClang, level, ltoMode, goGlobalDCE, NativeToolchainInput{})
}

// UseWithGOARMAndToolchain is UseWithGOARM with explicit Go-compatible native
// compiler commands. Named -target configurations intentionally ignore these
// host commands and preserve their existing toolchain selection.
func UseWithGOARMAndToolchain(goos, goarch, goarm, targetName string, wasiThreads, forceEspClang bool, level optlevel.Level, ltoMode lto.Mode, goGlobalDCE bool, nativeInput NativeToolchainInput) (export Export, err error) {
	if targetName != "" && !strings.HasPrefix(targetName, "wasm") && !strings.HasPrefix(targetName, "wasi") {
		return UseTarget(targetName, level, ltoMode)
	}
	return useWithGOARMAndToolchain(goos, goarch, goarm, wasiThreads, forceEspClang, level, ltoMode, goGlobalDCE, nativeInput)
}
