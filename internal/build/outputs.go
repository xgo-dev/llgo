package build

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/firmware"
)

// OutputCfg contains the generated output paths and conversion configuration
type OutputCfg struct {
	OutPath   string // Final output file path
	IntPath   string // Intermediate file path (for two-stage conversion)
	OutExt    string // Output file extension
	FileFmt   string // File format (from conf.FileFormat or extracted from emulator)
	BinFmt    string // Binary format for firmware conversion (may have -img suffix)
	NeedFwGen bool   // Whether firmware image generation is needed
	DirectGen bool   // True if can generate firmware directly without intermediate file
}

const pclnSidecarSuffix = ".pclntab"

// pclnSidecarPath returns the conventional optional runtime-symbolization
// artifact owned by an executable output.
func pclnSidecarPath(executable string) string {
	return executable + pclnSidecarSuffix
}

func genTempOutputFile(prefix, ext string) (string, error) {
	tmpFile, err := os.CreateTemp("", prefix+"-*"+ext)
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	return tmpFile.Name(), nil
}

// setOutFmt sets the appropriate OutFmt based on format name
func setOutFmt(conf *Config, formatName string) {
	switch formatName {
	case "bin":
		conf.OutFmts.Bin = true
	case "hex":
		conf.OutFmts.Hex = true
	case "img":
		conf.OutFmts.Img = true
	case "uf2":
		conf.OutFmts.Uf2 = true
	case "zip":
		conf.OutFmts.Zip = true
	}
}

// buildOutFmts creates OutFmtDetails based on package, configuration and multi-package status
// determineBaseNameAndDir extracts the base name and directory from configuration
func determineBaseNameAndDir(pkgName string, conf *Config, multiPkg bool) (baseName, dir string) {
	switch conf.Mode {
	case ModeInstall:
		return pkgName, conf.BinPath
	case ModeBuild:
		if !multiPkg && conf.OutFile != "" {
			dir = filepath.Dir(conf.OutFile)
			baseName = strings.TrimSuffix(filepath.Base(conf.OutFile), conf.AppExt)
			if dir == "." {
				dir = ""
			}
			return baseName, dir
		}
		return pkgName, ""
	case ModeTest:
		if conf.OutFile != "" {
			// Handle -o flag for test mode
			if strings.HasSuffix(conf.OutFile, "/") || isDir(conf.OutFile) {
				// If OutFile ends in / or is a directory, write pkg.test in that directory
				// pkgName for test packages already includes .test suffix
				return pkgName, conf.OutFile
			}
			// Otherwise, use the specified file path
			dir = filepath.Dir(conf.OutFile)
			baseName = strings.TrimSuffix(filepath.Base(conf.OutFile), conf.AppExt)
			// Don't convert "." to "" for test mode with explicit output
			// This preserves the information that user specified an output file
			return baseName, dir
		}
		if conf.CompileOnly {
			// -c without -o: write pkg.test in current directory
			// pkgName for test packages already includes .test suffix
			return pkgName, "."
		}
		// Default test mode without -c or -o: use temp file
		return pkgName, ""
	}
	// Other modes (run, etc.)
	return pkgName, ""
}

// isDir checks if path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// applyPrefix applies build mode specific naming conventions
func applyPrefix(baseName string, buildMode BuildMode, target string, goos string) string {
	// Determine the effective OS for naming conventions
	effectiveGoos := goos
	if target != "" {
		// Embedded targets follow Linux conventions
		effectiveGoos = "linux"
	}

	switch buildMode {
	case BuildModeCArchive:
		// Static libraries: libname.a (add lib prefix if missing)
		if !strings.HasPrefix(baseName, "lib") {
			return "lib" + baseName
		}
		return baseName

	case BuildModeCShared:
		// Shared libraries: libname.so/libname.dylib (add lib prefix if missing, except on Windows)
		if effectiveGoos != "windows" && !strings.HasPrefix(baseName, "lib") {
			return "lib" + baseName
		}
		return baseName

	case BuildModeExe:
		// Executables: name or name.exe (no lib prefix)
		if strings.HasPrefix(baseName, "lib") {
			return strings.TrimPrefix(baseName, "lib")
		}
		return baseName
	}

	return baseName
}

// buildOutputPath creates the final output path from baseName, dir and other parameters
func buildOutputPath(baseName, dir string, conf *Config, multiPkg bool, appExt string) (string, error) {
	baseName = applyPrefix(baseName, conf.BuildMode, conf.Target, conf.Goos)

	if dir != "" {
		// dir == "." means current directory (explicit user specification)
		// other dir values are actual directory paths
		if dir == "." {
			return baseName + appExt, nil
		}
		return filepath.Join(dir, baseName+appExt), nil
	} else if (conf.Mode == ModeBuild && multiPkg) || (conf.Mode != ModeBuild && conf.Mode != ModeInstall && conf.Mode != ModeTest) {
		return genTempOutputFile(baseName, appExt)
	} else if conf.Mode == ModeTest && !conf.CompileOnly && conf.OutFile == "" {
		// Only create temp file for test mode when:
		// - Not compile-only (-c flag not set)
		// - No output file specified (-o flag not set)
		return genTempOutputFile(baseName, appExt)
	} else {
		return baseName + appExt, nil
	}
}

func buildOutFmts(pkgName string, conf *Config, multiPkg bool, crossCompile *crosscompile.Export) (*OutFmtDetails, error) {
	details := &OutFmtDetails{}

	// Determine base name and directory
	baseName, dir := determineBaseNameAndDir(pkgName, conf, multiPkg)

	// Build output path
	outputPath, err := buildOutputPath(baseName, dir, conf, multiPkg, conf.AppExt)
	if err != nil {
		return nil, err
	}
	details.Out = outputPath
	if conf.PCLNMode == PCLNExternal {
		details.PCLN = pclnSidecarPath(details.Out)
	}

	if conf.Target == "" {
		// Native target - we're done
		return details, nil
	}

	needRun := slices.Contains([]Mode{ModeRun, ModeTest, ModeCmpTest, ModeInstall}, conf.Mode)

	// Check emulator format if emulator mode is enabled
	outFmt := ""
	if needRun {
		if conf.Emulator {
			if crossCompile.Emulator != "" {
				outFmt = firmware.ExtractFileFormatFromCommand(crossCompile.Emulator)
			}
		} else {
			if crossCompile.Device.Flash.Method == "command" {
				outFmt = firmware.ExtractFileFormatFromCommand(crossCompile.Device.Flash.Command)
			}
		}
	}
	if outFmt != "" {
		setOutFmt(conf, outFmt)
	}

	// Check binary format and set corresponding format
	if crossCompile.BinaryFormat != "" && needRun {
		envName := firmware.BinaryFormatToEnvName(crossCompile.BinaryFormat)
		if envName != "" {
			setOutFmt(conf, envName)
		}
	}

	// Generate format-specific paths based on base output
	if details.Out != "" {
		base := strings.TrimSuffix(details.Out, filepath.Ext(details.Out))

		if conf.OutFmts.Bin || conf.OutFmts.Img || conf.OutFmts.Hex {
			details.Bin = base + ".bin"
		}
		if conf.OutFmts.Hex {
			details.Hex = base + ".hex"
		}
		if conf.OutFmts.Img {
			details.Img = base + ".img"
		}
		if conf.OutFmts.Uf2 {
			details.Uf2 = base + ".uf2"
		}
		if conf.OutFmts.Zip {
			details.Zip = base + ".zip"
		}
	}

	return details, nil
}

// ToEnvMap converts OutFmtDetails to a map for template substitution
func (details *OutFmtDetails) ToEnvMap() map[string]string {
	envMap := make(map[string]string)

	if details.Out != "" {
		envMap[""] = details.Out
		envMap["out"] = details.Out
		envMap["elf"] = details.Out // alias for compatibility
	}
	if details.Bin != "" {
		envMap["bin"] = details.Bin
	}
	if details.Hex != "" {
		envMap["hex"] = details.Hex
	}
	if details.Img != "" {
		envMap["img"] = details.Img
	}
	if details.Uf2 != "" {
		envMap["uf2"] = details.Uf2
	}
	if details.Zip != "" {
		envMap["zip"] = details.Zip
	}
	if details.PCLN != "" {
		envMap["pclntab"] = details.PCLN
	}

	return envMap
}

func defaultAppExt(conf *Config) string {
	// Handle build mode specific extensions first
	switch conf.BuildMode {
	case BuildModeCArchive:
		return ".a"
	case BuildModeCShared:
		switch conf.Goos {
		case "windows":
			return ".dll"
		case "darwin":
			return ".dylib"
		default:
			return ".so"
		}
	case BuildModeExe:
		// For executable mode, handle target-specific logic
		if conf.Target != "" {
			if strings.HasPrefix(conf.Target, "wasi") || strings.HasPrefix(conf.Target, "wasm") {
				return ".wasm"
			}
			return ".elf"
		}

		switch conf.Goos {
		case "windows":
			return ".exe"
		case "wasi", "wasip1", "js":
			return ".wasm"
		}
		return ""
	}

	// This should not be reached, but kept for safety
	return ""
}
