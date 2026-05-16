package crosscompile

import (
	"fmt"
	"path/filepath"

	"github.com/goplus/llgo/internal/crosscompile/compile"
	"github.com/goplus/llgo/internal/crosscompile/compile/libc"
	"github.com/goplus/llgo/internal/crosscompile/compile/mquickjs"
	"github.com/goplus/llgo/internal/crosscompile/compile/rtlib"
)

// for testing, in testing env, we use fake path, it will cause downloading failure
var needSkipDownload = false

// getLibcCompileConfigByName retrieves libc compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getLibcCompileConfigByName(baseDir, libcName, target, mcpu string) (outputDir string, cfg compile.CompileConfig, err error) {
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

	if err = checkDownloadAndExtractLib(config.Url, libcDir, config.ResourceSubDir); err != nil {
		return
	}

	return libcDir, compileConfig, nil
}

// getRTCompileConfigByName retrieves runtime library compilation configuration by name
// Returns the actual libc output dir, compilation config and err
func getRTCompileConfigByName(baseDir, rtName, target string) (outputDir string, cfg compile.CompileConfig, err error) {
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

	if err = checkDownloadAndExtractLib(config.Url, rtDir, config.ResourceSubDir); err != nil {
		return
	}

	return rtDir, compileConfig, nil
}

// getJSLibCompileConfigByName retrieves JavaScript engine compilation configuration by name
// Returns the actual jslib output dir, compilation config and err
func getJSLibCompileConfigByName(baseDir, jslibName, target string) (outputDir string, cfg compile.CompileConfig, err error) {
	if jslibName == "" {
		err = fmt.Errorf("jslib name cannot be empty")
		return
	}
	var jslibDir string
	var config compile.LibConfig
	var compileConfig compile.CompileConfig

	switch jslibName {
	case "mquickjs":
		config = mquickjs.GetMquickjsConfig()
		jslibDir = filepath.Join(baseDir, config.String())
		compileConfig = mquickjs.GetMquickjsCompileConfig(jslibDir, target)
	default:
		err = fmt.Errorf("unsupported jslib: %s", jslibName)
		return
	}
	if needSkipDownload {
		return jslibDir, compileConfig, err
	}

	if err = checkDownloadAndExtractLib(config.Url, jslibDir, config.ResourceSubDir); err != nil {
		return
	}

	// Generate headers for mquickjs (mquickjs_atom.h and mqjs_stdlib.h)
	if jslibName == "mquickjs" {
		if err = mquickjs.GenerateHeaders(jslibDir); err != nil {
			return
		}
	}

	return jslibDir, compileConfig, nil
}
