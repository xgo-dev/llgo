package plan9asm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Distribution SDKs may omit VERSION while retaining the generated buildcfg
// source which embeds the selected compiler's version. Do not borrow the
// launcher's runtime version or invoke another Go binary on PATH.
func TestPreprocessSDKVersionFromSelectedBuildcfg(t *testing.T) {
	for _, test := range []struct{ version, want string }{
		{"go1.20.14", "old_arm64"},
		{"go1.26.10", "modern_arm64"},
		{"go1.27.2", "modern_arm64"},
	} {
		t.Run(test.version, func(t *testing.T) {
			pkg, file, opt := preprocessFixture(t)
			if err := os.Rename(filepath.Join(opt.GOROOT, "VERSION"), filepath.Join(opt.GOROOT, "unused-version")); err != nil {
				t.Fatal(err)
			}
			writePreprocessFile(t, filepath.Join(opt.GOROOT, "src", "internal", "buildcfg", "zbootstrap.go"),
				"package buildcfg\nconst version = `"+test.version+"`\n")
			opt.GOARM64 = "v8.1"
			t.Setenv("GOVERSION", "go1.99.0")
			t.Setenv("GOROOT", t.TempDir())
			got, err := PreprocessSourceForPkg(pkg, file,
				[]byte("#ifdef GOARM64_LSE\nmodern_arm64\n#else\nold_arm64\n#endif\n"), "linux", "arm64", opt)
			if err != nil || strings.TrimSpace(string(got)) != test.want {
				t.Fatalf("selected SDK version macros = %q, %v; want %s", got, err, test.want)
			}
		})
	}
}

func TestPreprocessOldSDKHasNoARM64FeatureMacros(t *testing.T) {
	for _, profile := range []string{"", "v8.0", "v8.1", "v9.5,lse,crypto"} {
		t.Run(profile, func(t *testing.T) {
			pkg, file, opt := preprocessFixture(t)
			writePreprocessFile(t, filepath.Join(opt.GOROOT, "VERSION"), "go1.22.2\n")
			opt.GOARM64 = profile
			output, err := PreprocessSourceForPkg(pkg, file,
				[]byte("#ifdef GOARM64_LSE\nWRONG\n#else\nOLD_SDK\n#endif\n"), "linux", "arm64", opt)
			if err != nil || string(output) != "OLD_SDK\n" {
				t.Fatalf("old SDK must retain its original macro namespace: %q, %v", output, err)
			}
		})
	}
}

func TestPreprocessSDKVersionPreservesAuthorityAndErrors(t *testing.T) {
	for _, test := range []struct{ name, bootstrap string }{
		{"missing", ""},
		{"wrong-package", "package unrelated\nconst version = `go1.26.10`"},
		{"missing-constant", "package buildcfg\nconst other = `go1.26.10`"},
		{"variable", "package buildcfg\nvar version = `go1.26.10`"},
		{"expression", "package buildcfg\nconst version = `go1.` + `26.10`"},
		{"duplicate", "package buildcfg\nconst version = `go1.26.10`\nconst version = `go1.27.2`"},
		{"multi-name", "package buildcfg\nconst version, other = `go1.26.10`, `ignored`"},
		{"malformed", "package buildcfg\nconst version ="},
		{"invalid-version", "package buildcfg\nconst version = `not-a-go-version`"},
		{"VERSION-is-directory", "package buildcfg\nconst version = `go1.26.10`"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg, file, opt := preprocessFixture(t)
			version := filepath.Join(opt.GOROOT, "VERSION")
			if err := os.Rename(version, version+".unused"); err != nil {
				t.Fatal(err)
			}
			if test.name == "VERSION-is-directory" {
				if err := os.Mkdir(version, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if test.bootstrap != "" {
				writePreprocessFile(t, filepath.Join(opt.GOROOT, "src", "internal", "buildcfg", "zbootstrap.go"), test.bootstrap)
			}
			if output, err := PreprocessSourceForPkg(pkg, file, []byte("RET\n"), "linux", "amd64", opt); err == nil || len(output) != 0 {
				t.Fatalf("invalid selected SDK metadata borrowed another version: %q, %v", output, err)
			}
		})
	}
}

func TestPreprocessSDKVersionFilePrecedesBootstrap(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "src", "internal", "buildcfg", "zbootstrap.go"), "invalid unused Go syntax")
	if output, err := PreprocessSourceForPkg(pkg, file, []byte("RET\n"), "linux", "amd64", opt); err != nil || strings.TrimSpace(string(output)) != "RET" {
		t.Fatalf("existing selected SDK VERSION lost precedence: %q, %v", output, err)
	}
}
