package crosscompile

import (
	"fmt"
	"path/filepath"

	"github.com/goplus/llgo/internal/crosscompile/compile"
	"github.com/goplus/llgo/internal/crosscompile/compile/libc"
	"github.com/goplus/llgo/internal/crosscompile/compile/rtlib"
	"github.com/goplus/llgo/internal/processenv"
)

// for testing, in testing env, we use fake path, it will cause downloading failure
var needSkipDownload = false

// getLibcCompileConfigByName retrieves libc compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getLibcCompileConfigByName(baseDir, libcName, target, mcpu string) (outputDir string, cfg compile.CompileConfig, err error) {
	return getLibcCompileConfigByNameWithContext(baseDir, libcName, target, mcpu, processenv.Context{})
}

func getLibcCompileConfigByNameWithContext(baseDir, libcName, target, mcpu string, process processenv.Context) (outputDir string, cfg compile.CompileConfig, err error) {
	if libcName == "" {
		err = fmt.Errorf("libc name cannot be empty")
		return
	}
	var libcDir string
	var config compile.LibConfig
	var compileConfig compile.CompileConfig

	switch libcName {
	case "picolibc":
		config = libc.GetPicolibcConfig()
		libcDir = filepath.Join(baseDir, config.String())
		compileConfig = libc.GetPicolibcCompileConfig(libcDir, target)
	case "newlib-esp32":
		config = libc.GetNewlibESP32Config()
		libcDir = filepath.Join(baseDir, config.String())
		compileConfig = libc.GetNewlibESP32CompileConfig(libcDir, target, mcpu)
	default:
		err = fmt.Errorf("unsupported libc: %s", libcName)
		return
	}
	if needSkipDownload {
		return libcDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLibWithContext(config.Url, libcDir, config.ResourceSubDir, process); err != nil {
		return
	}

	return libcDir, compileConfig, nil
}

// getRTCompileConfigByName retrieves runtime library compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getRTCompileConfigByName(baseDir, rtName, target string) (outputDir string, cfg compile.CompileConfig, err error) {
	return getRTCompileConfigByNameWithContext(baseDir, rtName, target, processenv.Context{})
}

func getRTCompileConfigByNameWithContext(baseDir, rtName, target string, process processenv.Context) (outputDir string, cfg compile.CompileConfig, err error) {
	if rtName == "" {
		err = fmt.Errorf("rt name cannot be empty")
		return
	}
	var rtDir string
	var config compile.LibConfig
	var compileConfig compile.CompileConfig

	switch rtName {
	case "compiler-rt":
		config = rtlib.GetCompilerRTConfig()
		rtDir = filepath.Join(baseDir, config.String())
		compileConfig = rtlib.GetCompilerRTCompileConfig(rtDir, target)
	default:
		err = fmt.Errorf("unsupported rt: %s", rtName)
	}
	if needSkipDownload {
		return rtDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLibWithContext(config.Url, rtDir, config.ResourceSubDir, process); err != nil {
		return
	}

	return rtDir, compileConfig, nil
}
