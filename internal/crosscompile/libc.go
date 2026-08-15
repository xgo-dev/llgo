package crosscompile

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goplus/llgo/internal/crosscompile/compile"
	"github.com/goplus/llgo/internal/crosscompile/compile/libc"
	"github.com/goplus/llgo/internal/crosscompile/compile/rtlib"
)

// for testing, in testing env, we use fake path, it will cause downloading failure
var needSkipDownload = false

var llvmVersionPattern = regexp.MustCompile(`[0-9]+\.[0-9]+(?:\.[0-9]+)?`)

func compilerVersionCacheKey(versionOutput string, payloadContract []byte) (string, error) {
	versionLine := strings.TrimSpace(versionOutput)
	if i := strings.IndexByte(versionLine, '\n'); i >= 0 {
		versionLine = versionLine[:i]
	}
	version := llvmVersionPattern.FindString(versionLine)
	if version == "" {
		return "", fmt.Errorf("parse compiler version from %q", versionLine)
	}
	identity := append([]byte(versionLine+"\x00"), payloadContract...)
	digest := sha256.Sum256(identity)
	return fmt.Sprintf("llvm-%s-%x", version, digest[:6]), nil
}

func compilerCacheKey(cc string) (string, error) {
	output, err := exec.Command(cc, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query compiler %q version: %w: %s", cc, err, strings.TrimSpace(string(output)))
	}
	payloadContract, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(cc)), "LLGO-LLVM-MANIFEST.txt"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read compiler payload contract: %w", err)
	}
	return compilerVersionCacheKey(string(output), payloadContract)
}

func compiledLibraryCacheKey(compilerKey string, flagGroups ...[]string) string {
	identity := compilerKey
	for _, flags := range flagGroups {
		identity += "\x00" + strings.Join(flags, "\x00") + "\x01"
	}
	digest := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("%s-%x", compilerKey, digest[:6])
}

func compiledLibraryDir(baseDir string, config compile.LibConfig, compilerKey string) string {
	return filepath.Join(baseDir, config.String()+"-"+compilerKey)
}

// getLibcCompileConfigByName retrieves libc compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getLibcCompileConfigByName(baseDir, libcName, target, mcpu, compilerKey string) (outputDir string, cfg compile.CompileConfig, err error) {
	if libcName == "" {
		err = fmt.Errorf("libc name cannot be empty")
		return
	}
	var sourceDir string
	var config compile.LibConfig
	var compileConfig compile.CompileConfig

	switch libcName {
	case "picolibc":
		config = libc.GetPicolibcConfig()
		sourceDir = filepath.Join(baseDir, config.String())
		compileConfig = libc.GetPicolibcCompileConfig(sourceDir, target)
	case "newlib-esp32":
		config = libc.GetNewlibESP32Config()
		sourceDir = filepath.Join(baseDir, config.String())
		compileConfig = libc.GetNewlibESP32CompileConfig(sourceDir, target, mcpu)
	default:
		err = fmt.Errorf("unsupported libc: %s", libcName)
		return
	}
	outputDir = compiledLibraryDir(baseDir, config, compilerKey)
	if needSkipDownload {
		return outputDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLib(config.Url, sourceDir, config.ResourceSubDir); err != nil {
		return
	}

	return outputDir, compileConfig, nil
}

// getRTCompileConfigByName retrieves runtime library compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getRTCompileConfigByName(baseDir, rtName, target, compilerKey string) (outputDir string, cfg compile.CompileConfig, err error) {
	if rtName == "" {
		err = fmt.Errorf("rt name cannot be empty")
		return
	}
	var sourceDir string
	var config compile.LibConfig
	var compileConfig compile.CompileConfig

	switch rtName {
	case "compiler-rt":
		config = rtlib.GetCompilerRTConfig()
		sourceDir = filepath.Join(baseDir, config.String())
		compileConfig = rtlib.GetCompilerRTCompileConfig(sourceDir, target)
	default:
		err = fmt.Errorf("unsupported rt: %s", rtName)
		return
	}
	outputDir = compiledLibraryDir(baseDir, config, compilerKey)
	if needSkipDownload {
		return outputDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLib(config.Url, sourceDir, config.ResourceSubDir); err != nil {
		return
	}

	return outputDir, compileConfig, nil
}
