/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package clang

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/xtool/safesplit"
)

// Config represents clang configuration parameters.
type Config struct {
	CC                string   // Compiler to use (e.g., "clang", "clang++")
	CCArgs            []string // Arguments that are part of the CC command
	CXX               string   // C++ compiler to use
	CXXArgs           []string // Arguments that are part of the CXX command
	CCFLAGS           []string // Compiler flags for C/C++ compilation
	CFLAGS            []string // C-specific flags
	LDFLAGS           []string // Linker flags
	Linker            string   // Linker to use (e.g., "ld.lld", "avr-ld")
	LinkerArgs        []string // Arguments that are part of the linker command
	ResponseFileStyle ResponseFileStyle
}

// ResponseFileStyle identifies the tokenizer used by a compiler or linker.
// Clang's GNU driver uses GNU tokenization even on Windows, while clang-cl,
// lld-link, link.exe, and cl.exe use Windows command-line tokenization.
type ResponseFileStyle uint8

const (
	ResponseFileAuto ResponseFileStyle = iota
	ResponseFileGNU
	ResponseFileWindows
)

// NewConfig creates a new Config with the specified parameters.
func NewConfig(cc string, ccflags, cflags, ldflags []string, linker string) Config {
	return Config{
		CC:      cc,
		CCFLAGS: ccflags,
		CFLAGS:  cflags,
		LDFLAGS: ldflags,
		Linker:  linker,
	}
}

// Cmd represents a clang command with environment and configuration support.
type Cmd struct {
	app        string
	prefixArgs []string
	config     Config
	Dir        string
	Env        []string
	Verbose    bool
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

// New creates a new clang command with configuration.
func New(app string, config Config) *Cmd {
	return newWithArgs(app, nil, config)
}

func newWithArgs(app string, args []string, config Config) *Cmd {
	if app == "" {
		app = "clang"
	}
	return &Cmd{
		app:        app,
		prefixArgs: slices.Clone(args),
		config:     config,
		Env:        nil,
		Verbose:    false,
		Stdin:      nil,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
}

// NewCompiler creates a compiler command with proper flag merging.
func NewCompiler(config Config) *Cmd {
	app := "clang"
	if config.CC != "" {
		app = config.CC
	}
	return newWithArgs(app, config.CCArgs, config)
}

// NewCXXCompiler creates a C++ compiler command with proper flag merging.
func NewCXXCompiler(config Config) *Cmd {
	if config.CXX != "" {
		return newWithArgs(config.CXX, config.CXXArgs, config)
	}
	return NewCompiler(config)
}

// NewLinker creates a linker command with proper flag merging.
func NewLinker(config Config) *Cmd {
	if config.Linker != "" {
		return newWithArgs(config.Linker, config.LinkerArgs, config)
	}
	return NewCompiler(config)
}

// Compile executes a compilation command with merged flags.
func (c *Cmd) Compile(args ...string) error {
	flags := c.mergeCompilerFlags()
	allArgs := make([]string, 0, len(flags)+len(args))
	allArgs = append(allArgs, flags...)
	allArgs = append(allArgs, args...)
	return c.exec(allArgs...)
}

// Link executes a linking command with merged flags.
func (c *Cmd) Link(args ...string) error {
	return c.exec(c.linkArguments(args...)...)
}

// LinkArguments returns the effective driver arguments, including its command
// prefix and environment flags, without executing the linker. Inspecting these
// arguments preserves explicit user options when selecting linker defaults.
func (c *Cmd) LinkArguments(args ...string) []string {
	return slices.Concat(c.prefixArgs, c.linkArguments(args...))
}

func (c *Cmd) linkArguments(args ...string) []string {
	flags := c.mergeLinkerFlags()
	allArgs := make([]string, 0, len(flags)+len(args))
	allArgs = append(allArgs, flags...)
	allArgs = append(allArgs, args...)
	return resolveMSVCImportLibraries(c.Dir, allArgs)
}

// resolveMSVCImportLibraries lets clang's MSVC driver consume library names
// emitted for Unix-compatible -l flags. The driver lowers -lname to name.lib
// and lets the linker search every -L directory for that exact spelling; it
// does not probe libname.lib or libname.dll.a per directory. Preserve that
// behavior by preferring name.lib anywhere on the explicit search path, then
// replace -lname with the first alternative spelling that exists.
func resolveMSVCImportLibraries(baseDir string, args []string) []string {
	if !slices.ContainsFunc(args, func(arg string) bool {
		return strings.Contains(arg, "-windows-msvc")
	}) {
		return args
	}
	var dirs []string
	for i, arg := range args {
		switch {
		case arg == "-L" && i+1 < len(args):
			dirs = append(dirs, args[i+1])
		case strings.HasPrefix(arg, "-L") && len(arg) > 2:
			dirs = append(dirs, arg[2:])
		}
	}
	resolved := args
	changed := false
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-l") || len(arg) <= 2 || arg[2] == ':' {
			continue
		}
		name := arg[2:]
		if findLibrary(baseDir, dirs, name+".lib") != "" {
			continue
		}
		archive := findLibrary(baseDir, dirs, "lib"+name+".lib")
		if archive == "" {
			archive = findLibrary(baseDir, dirs, "lib"+name+".dll.a")
		}
		if archive != "" {
			if !changed {
				resolved = slices.Clone(args)
				changed = true
			}
			resolved[i] = archive
		}
	}
	return resolved
}

func findLibrary(baseDir string, dirs []string, name string) string {
	for _, dir := range dirs {
		path := filepath.Join(dir, name)
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// mergeCompilerFlags merges environment CCFLAGS/CFLAGS with config flags.
func (c *Cmd) mergeCompilerFlags() []string {
	var flags []string

	// Add environment CCFLAGS
	if envCCFlags := os.Getenv("CCFLAGS"); envCCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCCFlags)...)
	}

	// Add environment CFLAGS
	if envCFlags := os.Getenv("CFLAGS"); envCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCFlags)...)
	}

	// Add config CCFLAGS
	flags = append(flags, c.config.CCFLAGS...)

	// Add config CFLAGS
	flags = append(flags, c.config.CFLAGS...)

	return flags
}

// mergeLinkerFlags merges environment CCFLAGS/LDFLAGS with config flags.
func (c *Cmd) mergeLinkerFlags() []string {
	var flags []string

	// Add environment CCFLAGS (for linker)
	if envCCFlags := os.Getenv("CCFLAGS"); envCCFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envCCFlags)...)
	}

	// Add environment LDFLAGS
	if envLDFlags := os.Getenv("LDFLAGS"); envLDFlags != "" {
		flags = append(flags, safesplit.SplitPkgConfigFlags(envLDFlags)...)
	}

	// Add config LDFLAGS
	flags = append(flags, c.config.LDFLAGS...)

	return flags
}

// exec executes the clang command with given arguments.
func (c *Cmd) exec(args ...string) error {
	if len(c.prefixArgs) != 0 {
		allArgs := make([]string, 0, len(c.prefixArgs)+len(args))
		allArgs = append(allArgs, c.prefixArgs...)
		args = append(allArgs, args...)
	}
	responseFile := ""
	if useResponseFile(c.app, args) {
		var err error
		responseFile, err = writeResponseFile(args, c.responseFileStyle())
		if err != nil {
			return fmt.Errorf("write clang response file: %w", err)
		}
		defer os.Remove(responseFile)
		args = []string{"@" + responseFile}
	}
	cmd := exec.Command(c.app, args...)
	cmd.Dir = c.Dir
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "%v\n", cmd)
	}
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	if c.Env != nil {
		cmd.Env = c.Env
	}
	return cmd.Run()
}

// Windows CreateProcess limits a command line to 32,767 UTF-16 code units.
// UTF-8 byte length is a conservative upper bound for UTF-16 length, and this
// lower threshold leaves room for executable-path and quoting overhead added
// by os/exec.
const windowsCommandLineLimit = 30 * 1024

func useResponseFile(app string, args []string) bool {
	return useResponseFileForGOOS(runtime.GOOS, app, args)
}

func useResponseFileForGOOS(goos, app string, args []string) bool {
	if goos != "windows" {
		return false
	}
	length := len(app)
	for _, arg := range args {
		length += 1 + len(arg)
	}
	return length > windowsCommandLineLimit
}

func (c *Cmd) responseFileStyle() ResponseFileStyle {
	if c.config.ResponseFileStyle != ResponseFileAuto {
		return c.config.ResponseFileStyle
	}
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(c.app)), ".exe")
	switch name {
	case "cl", "clang-cl", "link", "lld-link":
		return ResponseFileWindows
	default:
		return ResponseFileGNU
	}
}

func writeResponseFile(args []string, style ResponseFileStyle) (name string, err error) {
	file, err := os.CreateTemp("", "llgo-clang-*.rsp")
	if err != nil {
		return "", err
	}
	return writeResponseFileTo(file, args, style)
}

func writeResponseFileTo(file *os.File, args []string, style ResponseFileStyle) (name string, err error) {
	name = file.Name()
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(name)
		}
	}()

	var content strings.Builder
	for index, arg := range args {
		if index != 0 {
			content.WriteByte(' ')
		}
		if style == ResponseFileWindows {
			writeWindowsResponseArg(&content, arg)
		} else {
			writeGNUResponseArg(&content, arg)
		}
	}
	content.WriteByte('\n')
	_, err = io.WriteString(file, content.String())
	return name, err
}

// writeGNUResponseArg quotes one argument for LLVM's GNU response-file
// tokenizer. Within double quotes both backslash and quote must be escaped.
func writeGNUResponseArg(out *strings.Builder, arg string) {
	out.WriteByte('"')
	for _, char := range arg {
		if char == '"' || char == '\\' {
			out.WriteByte('\\')
		}
		out.WriteRune(char)
	}
	out.WriteByte('"')
}

// writeWindowsResponseArg quotes one argument using the CommandLineToArgvW
// convention consumed by Clang, lld-link, and link.exe on Windows. Backslashes
// are preserved unless they precede a quote or the closing delimiter; blindly
// doubling every backslash would corrupt ordinary paths such as C:\\src.
func writeWindowsResponseArg(out *strings.Builder, arg string) {
	out.WriteByte('"')
	backslashes := 0
	for _, char := range arg {
		if char == '\\' {
			backslashes++
			continue
		}
		if char == '"' {
			writeBackslashes(out, backslashes*2+1)
			out.WriteByte('"')
		} else {
			writeBackslashes(out, backslashes)
			out.WriteRune(char)
		}
		backslashes = 0
	}
	writeBackslashes(out, backslashes*2)
	out.WriteByte('"')
}

func writeBackslashes(out *strings.Builder, count int) {
	for range count {
		out.WriteByte('\\')
	}
}

// CheckLinkArgs validates linking arguments by attempting a test compile.
func (c *Cmd) CheckLinkArgs(cmdArgs []string, wasm bool) error {
	// Create a temporary file with appropriate extension
	extension := ""
	if wasm {
		extension = ".wasm"
	} else if runtime.GOOS == "windows" {
		extension = ".exe"
	}

	tmpFile, err := os.CreateTemp("", "llgo_check*"+extension)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpFile.Close()
	tmpPath := tmpFile.Name()

	// Make sure to delete the temporary file when done
	defer os.Remove(tmpPath)

	// Set up compilation arguments
	args := append([]string{}, cmdArgs...)
	args = append(args, []string{"-x", "c", "-o", tmpPath, "-"}...)
	src := "int main() {return 0;}"
	srcIn := strings.NewReader(src)
	c.Stdin = srcIn

	// Execute the command with linker flags
	return c.Link(args...)
}
