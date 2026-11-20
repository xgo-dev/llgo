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

	"github.com/xgo-dev/llgo/internal/crosscompile/compile"
	"github.com/xgo-dev/llgo/internal/crosscompile/compile/libc"
	"github.com/xgo-dev/llgo/internal/crosscompile/compile/rtlib"
	"github.com/xgo-dev/llgo/internal/env"
)

// for testing, in testing env, we use fake path, it will cause downloading failure
var needSkipDownload = false

var llvmVersionPattern = regexp.MustCompile(`[0-9]+\.[0-9]+(?:\.[0-9]+)?`)

func compilerVersionCacheKey(versionOutput string, payloadContract []byte) (string, error) {
	var versionLine string
	for _, line := range strings.Split(versionOutput, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(strings.ToLower(line), "version") {
			continue
		}
		if llvmVersionPattern.MatchString(line) {
			versionLine = line
			break
		}
	}
	version := llvmVersionPattern.FindString(versionLine)
	if version == "" {
		return "", fmt.Errorf("parse compiler version from %q", strings.TrimSpace(versionOutput))
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

func compiledLibraryDir(baseDir string, config compile.LibConfig, compiledLibraryKey string) string {
	return filepath.Join(baseDir, config.String()+"-"+compiledLibraryKey)
}

// getLibcCompileConfigByName retrieves libc compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getLibcCompileConfigByName(baseDir, libcName, target, mcpu, compiledLibraryKey string) (outputDir string, cfg compile.CompileConfig, err error) {
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
	outputDir = compiledLibraryDir(baseDir, config, compiledLibraryKey)
	if needSkipDownload {
		return outputDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLib(config.Url, sourceDir, config.ResourceSubDir); err != nil {
		return
	}

	return outputDir, compileConfig, nil
}

// rtCompileConfigByName builds the rtlib compile configuration by name.
// Pure path construction: it never downloads or compiles.
func rtCompileConfigByName(baseDir, rtName, target string) (config compile.LibConfig, compileConfig compile.CompileConfig, err error) {
	switch rtName {
	case "compiler-rt":
		config = rtlib.GetCompilerRTConfig()
		sourceDir := filepath.Join(baseDir, config.String())
		compileConfig = rtlib.GetCompilerRTCompileConfig(sourceDir, target)
	default:
		err = fmt.Errorf("unsupported rt: %s", rtName)
	}
	return
}

// rtlibCacheKeyGroups returns the rtlib source-file list as cache-key flag
// groups. The archive identity must cover the source file list and the
// contents of repo-vendored sources (unlike the version-pinned downloads,
// those can change between builds): the compile step only stats the archive,
// so a changed list or file would otherwise keep serving a stale archive.
func rtlibCacheKeyGroups(baseDir, rtName, target string) ([][]string, error) {
	_, compileConfig, err := rtCompileConfigByName(baseDir, rtName, target)
	if err != nil {
		return nil, err
	}
	root := env.LLGoROOT()
	groups := make([][]string, 0, len(compileConfig.Groups))
	for _, group := range compileConfig.Groups {
		files := append([]string(nil), group.Files...)
		for _, file := range group.Files {
			if root == "" || !strings.HasPrefix(file, root) {
				continue // downloaded sources are pinned by the version tag
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			sum := sha256.Sum256(content)
			files = append(files, fmt.Sprintf("content:%x", sum))
		}
		groups = append(groups, files)
	}
	return groups, nil
}

// getRTCompileConfigByName retrieves runtime library compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getRTCompileConfigByName(baseDir, rtName, target, compiledLibraryKey string) (outputDir string, compileConfig compile.CompileConfig, err error) {
	if rtName == "" {
		err = fmt.Errorf("rt name cannot be empty")
		return
	}
	config, compileConfig, err := rtCompileConfigByName(baseDir, rtName, target)
	if err != nil {
		return
	}

	outputDir = compiledLibraryDir(baseDir, config, compiledLibraryKey)
	if needSkipDownload {
		return
	}

	sourceDir := filepath.Join(baseDir, config.String())
	if err = checkDownloadAndExtractLib(config.Url, sourceDir, config.ResourceSubDir); err != nil {
		return
	}

	return outputDir, compileConfig, nil
}
