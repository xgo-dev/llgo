package env

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xgo-dev/llgo/internal/crosscompile/compile/libc"
	"github.com/xgo-dev/llgo/internal/crosscompile/compile/rtlib"
	internalenv "github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/targets"
	llvmenv "github.com/xgo-dev/llgo/xtool/env/llvm"
)

// Keep ordinary Go-only queries on the forwarding path, without inspecting
// LLGo installations, target files, or optional tools.
func extendedQuery(args []string) bool {
	hasName := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "LLGO_") || arg == "-target" || strings.HasPrefix(arg, "-target=") {
			return true
		}
		if !strings.HasPrefix(arg, "-") {
			hasName = true
		}
	}
	if hasName {
		return false
	}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name != "-json" && name != "-changed" {
			return false
		}
	}
	return true // no variables: include LLGo's environment in the full report
}

func runExtended(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("llgo env", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonMode := fs.Bool("json", false, "print JSON")
	changed := fs.Bool("changed", false, "print changed Go settings and explicitly set LLGo settings")
	write := fs.Bool("w", false, "write Go settings (LLGo fields are read-only)")
	unset := fs.Bool("u", false, "unset Go settings (LLGo fields are read-only)")
	target := fs.String("target", "", "inspect an embedded target without downloading or building")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *write || *unset {
		for _, arg := range fs.Args() {
			if strings.HasPrefix(arg, "LLGO_") {
				return fmt.Errorf("llgo env: %s is read-only; use the process environment to configure LLGo", strings.SplitN(arg, "=", 2)[0])
			}
		}
		if *target != "" {
			return fmt.Errorf("llgo env: -target cannot be combined with -w or -u")
		}
		return runGo(args, stdin, stdout, stderr)
	}
	vars, err := llgoVariables(*target)
	if err != nil {
		return err
	}
	names := fs.Args()
	all := len(names) == 0
	var goNames []string
	for _, name := range names {
		if !strings.HasPrefix(name, "LLGO_") {
			goNames = append(goNames, name)
		}
	}
	values := make(map[string]string)
	if all || len(goNames) != 0 {
		goArgs := []string{"-json"}
		if *changed {
			goArgs = append(goArgs, "-changed")
		}
		goArgs = append(goArgs, goNames...)
		var output bytes.Buffer
		if err := runGo(goArgs, stdin, &output, stderr); err != nil {
			return err
		}
		if err := json.Unmarshal(output.Bytes(), &values); err != nil {
			return fmt.Errorf("llgo env: decode Go environment: %w", err)
		}
	}
	if all {
		for name := range vars {
			names = append(names, name)
		}
	}
	type variableValue struct {
		name, value string
		set         bool
	}
	results := make([]variableValue, len(names))
	var pending sync.WaitGroup
	for index, name := range names {
		get, ok := vars[name]
		if !ok {
			if strings.HasPrefix(name, "LLGO_") && !*changed {
				results[index] = variableValue{name: name, set: true} // Go also returns an empty value for unknown names.
			}
			continue
		}
		if *changed {
			if _, set := os.LookupEnv(name); !set {
				continue
			}
		}
		results[index] = variableValue{name: name, set: true}
		pending.Add(1)
		go func(index int, get func() string) {
			defer pending.Done()
			results[index].value = get()
		}(index, get)
	}
	// A full report probes several independent LLVM and Clang properties. Run
	// their bounded subprocesses concurrently so slow diagnostics do not add up.
	pending.Wait()
	for _, result := range results {
		if result.set {
			values[result.name] = result.value
		}
	}
	if *jsonMode {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "\t")
		return encoder.Encode(values)
	}
	if !all {
		for _, name := range names {
			if value, ok := values[name]; ok {
				if _, err := fmt.Fprintln(stdout, value); err != nil {
					return err
				}
			}
		}
		return nil
	}
	names = names[:0]
	for name := range values {
		names = append(names, name)
	}
	// The merged report has no native Go-only order, so keep all Go and LLGo
	// fields in one deterministic alphabetical sequence.
	slices.Sort(names)
	for _, name := range names {
		if _, err := fmt.Fprintln(stdout, envAssignment(runtime.GOOS, name, values[name])); err != nil {
			return err
		}
	}
	return nil
}

func llgoVariables(targetName string) (map[string]func() string, error) {
	root := internalenv.LLGoROOT()
	cacheDir, _ := os.UserCacheDir()
	if cacheDir != "" {
		cacheDir = filepath.Join(cacheDir, "llgo")
	}
	value := func(s string) func() string { return func() string { return s } }
	variables := map[string]func() string{
		"LLGO_ROOT":                   value(root),
		"LLGO_VERSION":                value(internalenv.Version()),
		"LLGO_BUILD_TIME":             value(internalenv.BuildTime()),
		"LLGO_RUNTIME_DIR":            value(joinIfSet(root, internalenv.LLGoRuntimePkgName)),
		"LLGO_RUNTIME_PKG":            value(internalenv.LLGoRuntimePkg),
		"LLGO_CACHE_DIR":              value(cacheDir),
		"LLGO_PKG_CACHE_DIR":          value(joinIfSet(cacheDir, "build")),
		"LLGO_CROSSCOMPILE_DIR":       value(joinIfSet(root, "crosscompile")),
		"LLGO_CROSSCOMPILE_CACHE_DIR": value(joinIfSet(cacheDir, "crosscompile")),
		"LLGO_TARGETS_DIR":            value(joinIfSet(root, "targets")),
	}
	for _, name := range []string{
		"LLGO_AR", "LLGO_BUILD_CACHE", "LLGO_FULL_RPATH", "LLGO_FUNCINFO",
		"LLGO_FUNCINFO_SITES", "LLGO_LLDB", "LLGO_OPTIMIZE", "LLGO_PCLNPOST",
		"LLGO_PLAN9ASM_PKGS", "LLGO_SHADOW_STACK", "LLGO_STDIO_NOBUF",
		"LLGO_TRACE", "LLGO_WASI_THREADS", "LLGO_WASM_RUNTIME",
	} {
		name := name
		variables[name] = func() string { return os.Getenv(name) }
	}
	variables["LLGO_PICOLIBC_SOURCE_DIR"] = value(librarySourceDir(cacheDir, libc.GetPicolibcConfig().String()))
	variables["LLGO_NEWLIB_ESP32_SOURCE_DIR"] = value(librarySourceDir(cacheDir, libc.GetNewlibESP32Config().String()))
	variables["LLGO_COMPILER_RT_SOURCE_DIR"] = value(librarySourceDir(cacheDir, rtlib.GetCompilerRTConfig().String()))

	llvm := newLLVMInfo()
	variables["LLGO_LLVM_CONFIG"] = value(llvm.config)
	variables["LLGO_LLVM_BINDIR"] = func() string { return llvm.field("--bindir") }
	variables["LLGO_LLVM_LIBDIR"] = func() string { return llvm.field("--libdir") }
	variables["LLGO_LLVM_INCLUDEDIR"] = func() string { return llvm.field("--includedir") }
	variables["LLGO_LLVM_VERSION"] = func() string { return llvm.field("--version") }
	variables["LLGO_CLANG"] = func() string { return llvm.tool("clang") }
	variables["LLGO_CLANGXX"] = func() string { return llvm.tool("clang++") }
	variables["LLGO_CLANG_VERSION"] = func() string { return commandLine(llvm.tool("clang"), "--version") }
	variables["LLGO_CLANG_TARGET"] = func() string { return commandLine(llvm.tool("clang"), "-dumpmachine") }

	// These are the packaged locations. An empty result means this LLGo
	// installation does not contain the optional payload; env never downloads
	// it or guesses which cached artifact a future build would select.
	variables["LLGO_EMBEDDED_CLANG_DIR"] = value(existingDir(joinIfSet(root, llvmenv.CrosscompileClangPath)))
	variables["LLGO_WASI_LIBC_DIR"] = value(existingDir(joinIfSet(root, "crosscompile", "wasi-libc")))

	if targetName == "" {
		return variables, nil
	}
	config, err := targets.NewDefaultResolver().Resolve(targetName)
	if err != nil {
		return nil, fmt.Errorf("llgo env -target %s: %w", targetName, err)
	}
	addTargetVariables(variables, config, existingDir(joinIfSet(root, llvmenv.CrosscompileClangPath)))
	return variables, nil
}

func addTargetVariables(variables map[string]func() string, config *targets.Config, embeddedClangDir string) {
	value := func(s string) func() string { return func() string { return s } }
	extraBin := joinIfSet(embeddedClangDir, "bin")
	fields := map[string]string{
		"LLGO_TARGET":               config.Name,
		"LLGO_TARGET_GOOS":          config.GOOS,
		"LLGO_TARGET_GOARCH":        config.GOARCH,
		"LLGO_TARGET_WASM_PROFILE":  config.WasmProfile,
		"LLGO_TARGET_WASM_PROVIDER": config.WasmProvider,
		"LLGO_TARGET_LLVM_TARGET":   config.LLVMTarget,
		"LLGO_TARGET_CPU":           config.CPU,
		"LLGO_TARGET_FEATURES":      config.Features,
		"LLGO_TARGET_BUILD_TAGS":    strings.Join(config.BuildTags, " "),
		"LLGO_TARGET_LIBC":          config.Libc,
		"LLGO_TARGET_RTLIB":         config.RTLib,
		"LLGO_TARGET_LINKER":        config.Linker,
		"LLGO_TARGET_LINKER_SCRIPT": config.LinkerScript,
		"LLGO_TARGET_CFLAGS":        strings.Join(config.CFlags, " "),
		"LLGO_TARGET_LDFLAGS":       strings.Join(config.LDFlags, " "),
		"LLGO_TARGET_EXTRA_FILES":   strings.Join(config.ExtraFiles, " "),
		"LLGO_TARGET_EMULATOR":      config.Emulator,
		"LLGO_TARGET_GDB":           strings.Join(config.GDB, " "),
		"LLGO_TARGET_FLASH_METHOD":  config.FlashMethod,
		"LLGO_TARGET_FLASH_COMMAND": config.FlashCommand,
		"LLGO_TARGET_SERIAL":        config.Serial,
		"LLGO_TARGET_SERIAL_PORTS":  strings.Join(config.SerialPort, " "),
		"LLGO_TARGET_LINKER_PATH":   resolveTool(firstWord(config.Linker), extraBin),
		"LLGO_TARGET_EMULATOR_PATH": resolveTool(firstWord(config.Emulator), extraBin),
		"LLGO_TARGET_GDB_PATH":      resolveFirstTool(config.GDB, extraBin),
		"LLGO_TARGET_FLASH_TOOL":    resolveTool(firstWord(config.FlashCommand), extraBin),
	}
	for name, field := range fields {
		variables[name] = value(field)
	}
}

func librarySourceDir(cacheDir, name string) string {
	return joinIfSet(cacheDir, "crosscompile", name)
}

func firstWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func resolveFirstTool(commands []string, extraBin string) string {
	for _, command := range commands {
		if path := resolveTool(firstWord(command), extraBin); path != "" {
			return path
		}
	}
	return ""
}

func resolveTool(name, extraBin string) string {
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		for _, path := range toolCandidates(runtime.GOOS, filepath.Dir(name), filepath.Base(name)) {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		return ""
	}
	if extraBin != "" {
		for _, path := range toolCandidates(runtime.GOOS, extraBin, name) {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	path, _ := exec.LookPath(name)
	return path
}

type llvmInfo struct {
	config string
	mu     sync.Mutex
	fields map[string]func() string
	tools  map[string]func() string
}

func newLLVMInfo() *llvmInfo {
	return &llvmInfo{config: llvmenv.ConfigBin(), fields: make(map[string]func() string), tools: make(map[string]func() string)}
}

func (p *llvmInfo) field(flag string) string {
	p.mu.Lock()
	get := p.fields[flag]
	if get == nil {
		get = sync.OnceValue(func() string { return commandLine(p.config, flag) })
		p.fields[flag] = get
	}
	p.mu.Unlock()
	return get()
}

func (p *llvmInfo) tool(name string) string {
	p.mu.Lock()
	get := p.tools[name]
	if get == nil {
		get = sync.OnceValue(func() string {
			if binDir := p.field("--bindir"); binDir != "" {
				for _, path := range toolCandidates(runtime.GOOS, binDir, name) {
					if _, err := os.Stat(path); err == nil {
						return path
					}
				}
			}
			value, _ := exec.LookPath(name)
			return value
		})
		p.tools[name] = get
	}
	p.mu.Unlock()
	return get()
}

func toolCandidates(goos, dir, name string) []string {
	path := filepath.Join(dir, name)
	if goos == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return []string{path, path + ".exe"}
	}
	return []string{path}
}

func commandLine(name string, args ...string) string {
	if name == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(output))
	if line, _, ok := strings.Cut(value, "\n"); ok {
		return strings.TrimSpace(line)
	}
	return value
}

func existingDir(path string) string {
	if path == "" {
		return ""
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	return ""
}

func joinIfSet(base string, elem ...string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(append([]string{base}, elem...)...)
}

func envAssignment(goos, name, value string) string {
	value = sanitizeValue(value)
	if goos == "windows" {
		var escaped strings.Builder
		for _, char := range value {
			if char == '\r' || char == '\n' || (!unicode.IsGraphic(char) && !unicode.IsSpace(char)) {
				escaped.WriteRune(unicode.ReplacementChar)
				continue
			}
			switch char {
			case '%':
				escaped.WriteString("%%")
			case '<', '>', '|', '&', '^':
				escaped.WriteByte('^')
				escaped.WriteRune(char)
			default:
				escaped.WriteRune(char)
			}
		}
		return "set " + name + "=" + escaped.String()
	}
	return name + "='" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func sanitizeValue(value string) string {
	return strings.Map(func(char rune) rune {
		if char == '\r' || char == '\n' || (!unicode.IsGraphic(char) && !unicode.IsSpace(char)) {
			return unicode.ReplacementChar
		}
		return char
	}, value)
}
