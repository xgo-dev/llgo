package crosscompile

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// NativeToolchainInput contains Go-compatible native compiler command
// prefixes and external-linker flags. CC, CXX, and ExternalLinker are parsed
// command prefixes: element zero is the executable and the remainder are
// arguments that must precede LLGo's generated flags. ExternalFlags is the
// plain flag list parsed from -extldflags.
type NativeToolchainInput struct {
	CC             []string
	CXX            []string
	ExternalLinker []string
	ExternalFlags  []string
	Dir            string
	Environ        []string
	// ResolveWindows selects a Windows native-format toolchain when linked
	// output is produced for another architecture or from a non-Windows host.
	// Pure IR generation leaves it false so multi-target golden tests do not
	// depend on an installed compiler's architecture or ABI profile.
	ResolveWindows bool
}

type toolIdentity struct {
	target  string
	version string
}

type toolProbe func(command []string, input NativeToolchainInput) (toolIdentity, error)

const windowsGNUABIBuildTag = "llgo_windows_gnu"

func resolveWindowsToolchain(goarch string, input NativeToolchainInput, probe toolProbe) (Export, error) {
	if _, err := windowsTargetTriple(goarch, PlatformABIMsvc); err != nil {
		return Export{}, err
	}
	// Probe an implicit Clang before adding target flags. Official Windows LLVM
	// defaults to MSVC while MSYS2 CLANG64 defaults to GNU/MinGW, so either
	// native installation works without requiring CC/CXX environment settings.
	// The compiler target selects the profile; the invoking shell does not.
	implicitCC := len(input.CC) == 0
	cc := slices.Clone(input.CC)
	if implicitCC {
		cc = []string{"clang"}
	}

	ccIdentity, err := probe(cc, input)
	if err != nil {
		return Export{}, fmt.Errorf("probe CC %q: %w", cc[0], err)
	}
	toolchainArch := goarch
	if implicitCC {
		// The default compiler identifies the installed Windows ABI profile,
		// while GOARCH remains the target-architecture authority. This lets an
		// x64 host Clang target ARM64 or 386 without requiring CC solely to add
		// --target. Explicit CC commands remain architecture-checked below.
		toolchainArch, err = goarchForTargetTriple(ccIdentity.target)
		if err != nil {
			return Export{}, fmt.Errorf("CC %q: %w", cc[0], err)
		}
	}
	toolchain, err := windowsToolchainForTriple(toolchainArch, ccIdentity.target)
	if err != nil {
		return Export{}, fmt.Errorf("CC %q: %w", cc[0], err)
	}
	if implicitCC && toolchainArch != goarch {
		toolchain.TargetTriple, err = windowsTargetTriple(goarch, toolchain.ABI)
		if err != nil {
			return Export{}, err
		}
	}
	if err := requireClangGNUDriver("CC", cc, ccIdentity); err != nil {
		return Export{}, err
	}
	if implicitCC {
		cc = append(cc, "--target="+toolchain.TargetTriple)
		if toolchain.ABI == PlatformABIMsvc {
			// The standalone LLVM and native dependency profile use the dynamic
			// MSVC runtime. Explicit compiler commands remain the caller's choice.
			cc = append(cc, "-fms-runtime-lib=dll")
		}
	}
	// Visual Studio's developer variables identify only the MSVC dependency
	// profile. A GNU/MinGW profile must obtain its CRT, C++ runtime, sysroot,
	// and version identity from its own setup instead of inheriting the host's
	// MSVC installation and cache identity.
	if toolchain.ABI == PlatformABIMsvc {
		toolchain.SDKVersion = windowsEnvironmentValue(input.Environ, "WindowsSDKVersion")
		toolchain.CRTVersion = windowsEnvironmentValue(input.Environ, "UCRTVersion")
		toolchain.ToolsetVersion = windowsEnvironmentValue(input.Environ, "VCToolsVersion")
	}

	// An unset CXX follows the profile selected by CC. This is important when
	// an explicit CC selects MinGW: the host's default clang++ target must not
	// silently switch the C++ half of the build back to MSVC.
	cxxDefault := []string{"clang++", "--target=" + toolchain.TargetTriple}
	if toolchain.ABI == PlatformABIMsvc {
		cxxDefault = append(cxxDefault, "-fms-runtime-lib=dll")
	}
	cxx := commandOrDefault(input.CXX, cxxDefault...)
	cxxIdentity, err := probe(cxx, input)
	if err != nil {
		return Export{}, fmt.Errorf("probe CXX %q: %w", cxx[0], err)
	}
	cxxToolchain, err := windowsToolchainForTriple(goarch, cxxIdentity.target)
	if err != nil {
		return Export{}, fmt.Errorf("CXX %q: %w", cxx[0], err)
	}
	if err := requireClangGNUDriver("CXX", cxx, cxxIdentity); err != nil {
		return Export{}, err
	}
	if err := compatibleWindowsToolchains(toolchain, cxxToolchain); err != nil {
		return Export{}, fmt.Errorf("CC and CXX are incompatible: %w", err)
	}

	export := Export{
		CC:          cc[0],
		CCArgs:      slices.Clone(cc[1:]),
		CXX:         cxx[0],
		CXXArgs:     slices.Clone(cxx[1:]),
		CCIdentity:  ccIdentity.version,
		CXXIdentity: cxxIdentity.version,
		Toolchain:   toolchain,
		LLVMTarget:  toolchain.TargetTriple,
	}
	if toolchain.ABI == PlatformABIGNU {
		// x86-64 libffi distinguishes its GNUW64 and Win64 interfaces even
		// though both toolchain profiles use the native Windows C ABI. Keep
		// that implementation detail automatic and in the package cache key.
		export.BuildTags = []string{windowsGNUABIBuildTag}
	}
	if len(input.ExternalLinker) != 0 {
		linkerIdentity, err := probe(input.ExternalLinker, input)
		if err != nil {
			return Export{}, fmt.Errorf("probe -extld %q: %w", input.ExternalLinker[0], err)
		}
		linkerToolchain, err := windowsToolchainForTriple(goarch, linkerIdentity.target)
		if err != nil {
			return Export{}, fmt.Errorf("-extld %q: %w", input.ExternalLinker[0], err)
		}
		if err := requireClangGNUDriver("-extld", input.ExternalLinker, linkerIdentity); err != nil {
			return Export{}, err
		}
		if err := compatibleWindowsToolchains(toolchain, linkerToolchain); err != nil {
			return Export{}, fmt.Errorf("-extld is incompatible with CC: %w", err)
		}
		export.Linker = input.ExternalLinker[0]
		export.LinkerArgs = slices.Clone(input.ExternalLinker[1:])
		export.LinkerIdentity = linkerIdentity.version
	}
	export.LDFLAGS = slices.Clone(input.ExternalFlags)
	return export, nil
}

func windowsEnvironmentValue(environ []string, name string) string {
	for i := len(environ) - 1; i >= 0; i-- {
		key, value, ok := strings.Cut(environ[i], "=")
		if ok && strings.EqualFold(key, name) {
			return strings.TrimRight(strings.TrimSpace(value), `\/`)
		}
	}
	return ""
}

func commandOrDefault(command []string, defaults ...string) []string {
	if len(command) == 0 {
		return slices.Clone(defaults)
	}
	return slices.Clone(command)
}

func requireClangGNUDriver(setting string, command []string, identity toolIdentity) error {
	for _, arg := range command {
		name := strings.TrimSuffix(strings.ToLower(filepath.Base(arg)), ".exe")
		if name == "clang-cl" || arg == "--driver-mode=cl" || strings.HasPrefix(arg, "--driver-mode=cl=") {
			return fmt.Errorf("%s %q uses clang-cl flag syntax; LLGo currently requires Clang's GNU-compatible driver", setting, command[0])
		}
	}
	if strings.Contains(strings.ToLower(identity.version), "clang") {
		return nil
	}
	return fmt.Errorf("%s %q uses an unsupported driver (%s); Windows builds currently require Clang's GNU-compatible driver", setting, command[0], identity.version)
}

func compatibleWindowsToolchains(a, b NativeToolchain) error {
	if a.ABI != b.ABI || a.TargetTriple != b.TargetTriple || a.CRT != b.CRT || a.CXXRuntime != b.CXXRuntime {
		return fmt.Errorf("profiles differ: %s versus %s", describeWindowsToolchain(a), describeWindowsToolchain(b))
	}
	return nil
}

func describeWindowsToolchain(toolchain NativeToolchain) string {
	return fmt.Sprintf("target=%s ABI=%s CRT=%s C++=%s", toolchain.TargetTriple, toolchain.ABI, toolchain.CRT, toolchain.CXXRuntime)
}

func windowsToolchainForTriple(goarch, target string) (NativeToolchain, error) {
	target = strings.ToLower(strings.TrimSpace(strings.Split(target, "\n")[0]))
	if target == "" {
		return NativeToolchain{}, fmt.Errorf("compiler reported an empty target triple")
	}
	if strings.Contains(target, "cygwin") || strings.Contains(target, "cygnus") || strings.Contains(target, "msys") {
		return NativeToolchain{}, fmt.Errorf("target %q uses the unsupported MSYS/Cygwin POSIX-emulation ABI", target)
	}
	arch, err := goarchForTargetTriple(target)
	if err != nil {
		return NativeToolchain{}, err
	}
	if arch != goarch {
		return NativeToolchain{}, fmt.Errorf("target %q is %s, want GOARCH=%s", target, arch, goarch)
	}

	toolchain := NativeToolchain{
		ObjectFormat: ObjectFormatCOFF,
		Driver:       DriverFlavorClangGNU,
	}
	switch {
	case strings.Contains(target, "windows-msvc"):
		toolchain.ABI = PlatformABIMsvc
		toolchain.Linker = LinkerFlavorCOFFLLD
		toolchain.CRT = CRTFlavorUCRT
		toolchain.CXXRuntime = CXXRuntimeMSVC
	case strings.Contains(target, "windows-gnu"), strings.Contains(target, "mingw32"):
		toolchain.ABI = PlatformABIGNU
		toolchain.Linker = LinkerFlavorMinGWLLD
	default:
		return NativeToolchain{}, fmt.Errorf("target %q is not an explicit Windows MSVC or GNU/MinGW target", target)
	}
	toolchain.TargetTriple, err = windowsTargetTriple(goarch, toolchain.ABI)
	if err != nil {
		return NativeToolchain{}, err
	}
	return toolchain, nil
}

func windowsTargetTriple(goarch string, abi PlatformABI) (string, error) {
	var arch string
	switch goarch {
	case "386":
		arch = "i686"
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return "", fmt.Errorf("unsupported Windows GOARCH %q", goarch)
	}
	switch abi {
	case PlatformABIMsvc:
		return arch + "-pc-windows-msvc", nil
	case PlatformABIGNU:
		return arch + "-w64-windows-gnu", nil
	default:
		return "", fmt.Errorf("unsupported Windows ABI %q", abi)
	}
}

func goarchForTargetTriple(target string) (string, error) {
	arch, _, _ := strings.Cut(target, "-")
	switch arch {
	case "i386", "i486", "i586", "i686", "x86":
		return "386", nil
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("target %q has unsupported Windows architecture %q", target, arch)
	}
}

func probeNativeTool(command []string, input NativeToolchainInput) (toolIdentity, error) {
	run := func(arg string) ([]byte, error) {
		args := make([]string, 0, len(command))
		args = append(args, command[1:]...)
		args = append(args, arg)
		cmd := exec.Command(command[0], args...)
		cmd.Dir = input.Dir
		if input.Environ != nil {
			cmd.Env = input.Environ
		}
		return cmd.CombinedOutput()
	}

	targetOutput, targetErr := run("-dumpmachine")
	if targetErr != nil || len(bytes.TrimSpace(targetOutput)) == 0 {
		targetOutput, targetErr = run("--print-target-triple")
	}
	if targetErr != nil {
		return toolIdentity{}, fmt.Errorf("read target triple: %w: %s", targetErr, strings.TrimSpace(string(targetOutput)))
	}
	versionOutput, err := run("--version")
	if err != nil {
		return toolIdentity{}, fmt.Errorf("read compiler identity: %w: %s", err, strings.TrimSpace(string(versionOutput)))
	}
	return toolIdentity{
		target:  firstOutputLine(targetOutput),
		version: firstOutputLine(versionOutput),
	}, nil
}

func firstOutputLine(output []byte) string {
	line, _, _ := bytes.Cut(bytes.TrimSpace(output), []byte{'\n'})
	return strings.TrimSpace(strings.TrimSuffix(string(line), "\r"))
}
