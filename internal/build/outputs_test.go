//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/flash"
)

func TestBuildOutFmtsWithTarget(t *testing.T) {
	tests := []struct {
		name         string
		conf         *Config
		pkgName      string
		crossCompile crosscompile.Export
		wantOut      string // use empty string to indicate temp file
		wantBin      string
		wantHex      string
		wantImg      string
		wantUf2      string
		wantZip      string
	}{
		{
			name: "build with -o",
			conf: &Config{
				Mode:    ModeBuild,
				OutFile: "myapp",
				AppExt:  "",
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "",
			},
			wantOut: "myapp",
		},
		{
			name: "build hex format",
			conf: &Config{
				Mode:    ModeBuild,
				Target:  "esp32",
				AppExt:  ".elf",
				OutFmts: OutFmts{Hex: true},
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "esp32", // This will auto-set Bin: true
			},
			wantOut: "hello.elf",
			wantBin: "hello.bin", // Now expected due to esp32 BinaryFormat
			wantHex: "hello.hex",
		},
		{
			name: "emulator mode with bin format",
			conf: &Config{
				Mode:     ModeRun,
				Target:   "esp32",
				AppExt:   ".elf",
				Emulator: true,
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "esp32",
				Emulator:     "qemu-system-xtensa -machine esp32 -kernel {bin}",
			},
			wantBin: ".bin", // Should be temp file with .bin extension
		},
		{
			name: "flash command with hex format",
			conf: &Config{
				Mode:     ModeInstall,
				Target:   "esp32",
				AppExt:   ".elf",
				Emulator: false,
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "esp32", // This will auto-set Bin: true
				Device: flash.Device{
					Flash: flash.Flash{
						Method:  "command",
						Command: "esptool.py --chip esp32 write_flash 0x10000 {hex}",
					},
				},
			},
			wantBin: ".bin", // Expected due to esp32 BinaryFormat
			wantHex: ".hex", // Should be temp file with .hex extension
		},
		{
			name: "multiple formats specified",
			conf: &Config{
				Mode:   ModeBuild,
				Target: "esp32",
				AppExt: ".elf",
				OutFmts: OutFmts{
					Bin: true,
					Hex: true,
					Img: true,
					Uf2: true,
				},
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "esp32",
			},
			wantOut: "hello.elf",
			wantBin: "hello.bin",
			wantHex: "hello.hex",
			wantImg: "hello.img",
			wantUf2: "hello.uf2",
			wantZip: "", // Not specified
		},
		{
			name: "no formats specified",
			conf: &Config{
				Mode:    ModeBuild,
				Target:  "esp32",
				AppExt:  ".elf",
				OutFmts: OutFmts{}, // No formats specified
			},
			pkgName: "hello",
			crossCompile: crosscompile.Export{
				BinaryFormat: "esp32",
			},
			wantOut: "hello.elf",
			wantBin: "", // No bin file should be generated
			wantHex: "", // No hex file should be generated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine if multi-package (mock single package for simplicity)
			multiPkg := false

			result, err := buildOutFmts(tt.pkgName, tt.conf, multiPkg, &tt.crossCompile)
			if err != nil {
				t.Errorf("buildOutFmts() error = %v", err)
				return
			}

			// Check base output path
			if tt.wantOut != "" {
				if result.Out != tt.wantOut {
					t.Errorf("buildOutFmts().Out = %v, want %v", result.Out, tt.wantOut)
				}
			} else {
				// Should be a temp file
				if result.Out == "" || !strings.Contains(result.Out, tt.pkgName) {
					t.Errorf("buildOutFmts().Out should be temp file containing %v, got %v", tt.pkgName, result.Out)
				}
			}

			// Check format-specific paths
			checkFormatPath := func(actual, expected, formatName string) {
				if expected == "" {
					// Empty means no file should be generated
					if actual != "" {
						t.Errorf("buildOutFmts().%s = %v, want empty (no file generated)", formatName, actual)
					}
				} else if strings.HasPrefix(expected, ".") && !strings.Contains(expected[1:], ".") {
					// ".xxx" means temp file with .xxx extension
					if actual == "" {
						t.Errorf("buildOutFmts().%s = empty, want temp file with %s extension", formatName, expected)
					} else if !strings.HasSuffix(actual, expected) || !strings.Contains(actual, tt.pkgName) {
						t.Errorf("buildOutFmts().%s should be temp file with %s extension containing %v, got %v", formatName, expected, tt.pkgName, actual)
					}
				} else {
					// "aaa.xxx" means exact file name
					if actual != expected {
						t.Errorf("buildOutFmts().%s = %v, want %v", formatName, actual, expected)
					}
				}
			}

			checkFormatPath(result.Bin, tt.wantBin, "Bin")
			checkFormatPath(result.Hex, tt.wantHex, "Hex")
			checkFormatPath(result.Img, tt.wantImg, "Img")
			checkFormatPath(result.Uf2, tt.wantUf2, "Uf2")
			checkFormatPath(result.Zip, tt.wantZip, "Zip")
		})
	}
}

func TestBuildOutFmtsNativeTarget(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		multiPkg bool
		outFile  string
		binPath  string
		appExt   string
		goos     string
		pkgName  string
		wantOut  string
	}{
		// ModeBuild scenarios
		{
			name:     "build single pkg no outfile macos",
			mode:     ModeBuild,
			multiPkg: false,
			outFile:  "",
			appExt:   "",
			goos:     "darwin",
			pkgName:  "hello",
			wantOut:  "hello",
		},
		{
			name:     "build single pkg no outfile linux",
			mode:     ModeBuild,
			multiPkg: false,
			outFile:  "",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "hello",
		},
		{
			name:     "build single pkg no outfile windows",
			mode:     ModeBuild,
			multiPkg: false,
			outFile:  "",
			appExt:   ".exe",
			goos:     "windows",
			pkgName:  "hello",
			wantOut:  "hello.exe",
		},
		{
			name:     "build single pkg with outfile",
			mode:     ModeBuild,
			multiPkg: false,
			outFile:  "myapp",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "myapp",
		},
		{
			name:     "build single pkg with outfile and ext",
			mode:     ModeBuild,
			multiPkg: false,
			outFile:  "myapp.exe",
			appExt:   ".exe",
			goos:     "windows",
			pkgName:  "hello",
			wantOut:  "myapp.exe",
		},
		{
			name:     "build multi pkg",
			mode:     ModeBuild,
			multiPkg: true,
			outFile:  "",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "", // Should be temp file
		},
		// ModeInstall scenarios
		{
			name:     "install single pkg macos",
			mode:     ModeInstall,
			multiPkg: false,
			outFile:  "",
			binPath:  "/go/bin",
			appExt:   "",
			goos:     "darwin",
			pkgName:  "hello",
			wantOut:  "/go/bin/hello",
		},
		{
			name:     "install single pkg windows",
			mode:     ModeInstall,
			multiPkg: false,
			outFile:  "",
			binPath:  "C:/go/bin", // Use forward slashes for cross-platform compatibility
			appExt:   ".exe",
			goos:     "windows",
			pkgName:  "hello",
			wantOut:  "C:/go/bin/hello.exe",
		},
		{
			name:     "install multi pkg",
			mode:     ModeInstall,
			multiPkg: true,
			outFile:  "",
			binPath:  "/go/bin",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "/go/bin/hello",
		},
		// Other modes should use temp files
		{
			name:     "run mode",
			mode:     ModeRun,
			multiPkg: false,
			outFile:  "",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "", // Should be temp file
		},
		{
			name:     "test mode",
			mode:     ModeTest,
			multiPkg: false,
			outFile:  "",
			appExt:   "",
			goos:     "linux",
			pkgName:  "hello",
			wantOut:  "", // Should be temp file
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &Config{
				Mode:    tt.mode,
				OutFile: tt.outFile,
				BinPath: tt.binPath,
				AppExt:  tt.appExt,
				Target:  "", // Native target
			}

			crossCompile := &crosscompile.Export{}

			result, err := buildOutFmts(tt.pkgName, conf, tt.multiPkg, crossCompile)
			if err != nil {
				t.Errorf("buildOutFmts() error = %v", err)
				return
			}

			// Check base output path
			if tt.wantOut != "" {
				if result.Out != tt.wantOut {
					t.Errorf("buildOutFmts().Out = %v, want %v", result.Out, tt.wantOut)
				}
			} else {
				// Should be a temp file
				if result.Out == "" || !strings.Contains(result.Out, tt.pkgName) {
					t.Errorf("buildOutFmts().Out should be temp file containing %v, got %v", tt.pkgName, result.Out)
				}
			}

			// For native targets, no format files should be generated
			if result.Bin != "" {
				t.Errorf("buildOutFmts().Bin = %v, want empty for native target", result.Bin)
			}
			if result.Hex != "" {
				t.Errorf("buildOutFmts().Hex = %v, want empty for native target", result.Hex)
			}
			if result.Img != "" {
				t.Errorf("buildOutFmts().Img = %v, want empty for native target", result.Img)
			}
			if result.Uf2 != "" {
				t.Errorf("buildOutFmts().Uf2 = %v, want empty for native target", result.Uf2)
			}
			if result.Zip != "" {
				t.Errorf("buildOutFmts().Zip = %v, want empty for native target", result.Zip)
			}
		})
	}
}

func TestBuildOutFmtsPCLN(t *testing.T) {
	tests := []struct {
		name string
		conf *Config
		want string
	}{
		{
			name: "embedded",
			conf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: "app", PCLNMode: PCLNEmbedded},
		},
		{
			name: "none",
			conf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: "app", PCLNMode: PCLNNone},
		},
		{
			name: "external",
			conf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: "app", PCLNMode: PCLNExternal},
			want: "app.pclntab",
		},
		{
			name: "external suffix follows executable extension",
			conf: &Config{Mode: ModeBuild, BuildMode: BuildModeExe, OutFile: "dist/app.exe", AppExt: ".exe", PCLNMode: PCLNExternal},
			want: "dist/app.exe.pclntab",
		},
		{
			name: "external install",
			conf: &Config{Mode: ModeInstall, BuildMode: BuildModeExe, BinPath: "/go/bin", PCLNMode: PCLNExternal},
			want: "/go/bin/hello.pclntab",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildOutFmts("hello", tt.conf, false, &crosscompile.Export{})
			if err != nil {
				t.Fatal(err)
			}
			if got.PCLN != tt.want {
				t.Fatalf("buildOutFmts().PCLN = %q, want %q", got.PCLN, tt.want)
			}
		})
	}
}

func TestOutFmtDetailsToEnvMapIncludesPCLN(t *testing.T) {
	details := &OutFmtDetails{Out: "app", PCLN: "app.pclntab"}
	if got := details.ToEnvMap()["pclntab"]; got != details.PCLN {
		t.Fatalf("ToEnvMap()[pclntab] = %q, want %q", got, details.PCLN)
	}
	if _, ok := (&OutFmtDetails{Out: "app"}).ToEnvMap()["pclntab"]; ok {
		t.Fatal("ToEnvMap() contains pclntab for an empty PCLN output")
	}
}

func TestBuildOutFmtsBuildModes(t *testing.T) {
	tests := []struct {
		name        string
		pkgName     string
		buildMode   BuildMode
		outFile     string
		mode        Mode
		target      string
		goos        string
		appExt      string
		expectedOut string
	}{
		// C-Archive tests
		{
			name:        "c_archive_build_linux",
			pkgName:     "mylib",
			buildMode:   BuildModeCArchive,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      ".a",
			expectedOut: "libmylib.a",
		},
		{
			name:        "c_archive_build_with_outfile",
			pkgName:     "mylib",
			buildMode:   BuildModeCArchive,
			outFile:     "custom.a",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      ".a",
			expectedOut: "libcustom.a",
		},
		{
			name:        "c_archive_build_with_path",
			pkgName:     "mylib",
			buildMode:   BuildModeCArchive,
			outFile:     "build/custom.a",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      ".a",
			expectedOut: "build/libcustom.a",
		},

		// C-Shared tests
		{
			name:        "c_shared_build_linux",
			pkgName:     "mylib",
			buildMode:   BuildModeCShared,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      ".so",
			expectedOut: "libmylib.so",
		},
		{
			name:        "c_shared_build_windows",
			pkgName:     "mylib",
			buildMode:   BuildModeCShared,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "windows",
			appExt:      ".dll",
			expectedOut: "mylib.dll",
		},
		{
			name:        "c_shared_build_darwin",
			pkgName:     "mylib",
			buildMode:   BuildModeCShared,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "darwin",
			appExt:      ".dylib",
			expectedOut: "libmylib.dylib",
		},
		{
			name:        "c_shared_embedded_target",
			pkgName:     "mylib",
			buildMode:   BuildModeCShared,
			outFile:     "",
			mode:        ModeBuild,
			target:      "rp2040",
			goos:        "windows",
			appExt:      ".so", // embedded follows linux rules
			expectedOut: "libmylib.so",
		},

		// Executable tests
		{
			name:        "exe_build_linux",
			pkgName:     "myapp",
			buildMode:   BuildModeExe,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      "",
			expectedOut: "myapp",
		},
		{
			name:        "exe_build_windows",
			pkgName:     "myapp",
			buildMode:   BuildModeExe,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "windows",
			appExt:      ".exe",
			expectedOut: "myapp.exe",
		},
		{
			name:        "exe_remove_lib_prefix",
			pkgName:     "libmyapp",
			buildMode:   BuildModeExe,
			outFile:     "",
			mode:        ModeBuild,
			target:      "",
			goos:        "linux",
			appExt:      "",
			expectedOut: "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &Config{
				Mode:      tt.mode,
				BuildMode: tt.buildMode,
				Target:    tt.target,
				Goos:      tt.goos,
				OutFile:   tt.outFile,
				AppExt:    tt.appExt,
			}

			crossCompile := &crosscompile.Export{}
			result, err := buildOutFmts(tt.pkgName, conf, false, crossCompile)

			if err != nil {
				t.Fatalf("buildOutFmts failed: %v", err)
			}

			if result.Out != tt.expectedOut {
				t.Errorf("buildOutFmts(%q, buildMode=%v, target=%q, goos=%q) = %q, want %q",
					tt.pkgName, tt.buildMode, tt.target, tt.goos, result.Out, tt.expectedOut)
			}
		})
	}
}

func TestApplyBuildModeNaming(t *testing.T) {
	tests := []struct {
		name      string
		baseName  string
		buildMode BuildMode
		target    string
		goos      string
		expected  string
	}{
		// Executable tests
		{
			name:      "exe_linux",
			baseName:  "myapp",
			buildMode: BuildModeExe,
			target:    "",
			goos:      "linux",
			expected:  "myapp",
		},
		{
			name:      "exe_remove_lib_prefix",
			baseName:  "libmyapp",
			buildMode: BuildModeExe,
			target:    "",
			goos:      "linux",
			expected:  "myapp",
		},

		// C-Archive tests
		{
			name:      "c_archive_linux",
			baseName:  "mylib",
			buildMode: BuildModeCArchive,
			target:    "",
			goos:      "linux",
			expected:  "libmylib",
		},
		{
			name:      "c_archive_existing_prefix",
			baseName:  "libmylib",
			buildMode: BuildModeCArchive,
			target:    "",
			goos:      "linux",
			expected:  "libmylib",
		},

		// C-Shared tests
		{
			name:      "c_shared_linux",
			baseName:  "mylib",
			buildMode: BuildModeCShared,
			target:    "",
			goos:      "linux",
			expected:  "libmylib",
		},
		{
			name:      "c_shared_windows",
			baseName:  "mylib",
			buildMode: BuildModeCShared,
			target:    "",
			goos:      "windows",
			expected:  "mylib", // Windows doesn't use lib prefix
		},
		{
			name:      "c_shared_embedded_rp2040",
			baseName:  "mylib",
			buildMode: BuildModeCShared,
			target:    "rp2040",
			goos:      "darwin",
			expected:  "libmylib", // embedded follows linux rules
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyPrefix(tt.baseName, tt.buildMode, tt.target, tt.goos)
			if result != tt.expected {
				t.Errorf("applyBuildModeNaming(%q, %v, %q, %q) = %q, want %q",
					tt.baseName, tt.buildMode, tt.target, tt.goos, result, tt.expected)
			}
		})
	}
}
