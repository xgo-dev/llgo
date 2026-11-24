package plan9asm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPreprocessSourceInputBoundaries(t *testing.T) {
	for _, mutation := range []string{
		"nil-package", "empty-file", "invalid-target", "expanded-limit", "absolute-include",
		"escaping-include", "overlay-aliases", "include-total-limit", "disk-file-limit", "wasm-macros", "unreviewed-sdk-version",
	} {
		t.Run(mutation, func(t *testing.T) {
			pkg, file, opt := preprocessFixture(t)
			source, goos, arch := []byte("RET\n"), "linux", "amd64"
			want := ""
			switch mutation {
			case "nil-package":
				pkg, want = nil, "nil package"
			case "empty-file":
				file, want = "", "package and filename"
			case "invalid-target":
				goos, want = "", "explicit valid target"
			case "expanded-limit":
				opt.SourcePreprocessed = true
				source, want = bytes.Repeat([]byte(" "), maxAssemblyOutputBytes+1), "output bound"
			case "absolute-include":
				source = []byte("#include " + strconv.Quote(filepath.ToSlash(filepath.Join(pkg.Dir, "header.h"))) + "\n")
				want = "package-relative"
			case "escaping-include":
				source, want = []byte("#include \"../../outside.h\"\n"), "escapes package/module/SDK roots"
			case "overlay-aliases":
				header := filepath.Join(pkg.Dir, "header.h")
				alias := pkg.Dir + string(filepath.Separator) + "." + string(filepath.Separator) + "header.h"
				opt.Overlay = map[string][]byte{header: []byte("RET\n"), alias: []byte("NOP\n")}
				source, want = []byte("#include \"header.h\"\n"), "conflicting assembly overlay aliases"
			case "include-total-limit":
				// Each include fits, but repeated reads still consume the total input budget.
				opt.Overlay = map[string][]byte{filepath.Join(pkg.Dir, "header.h"): bytes.Repeat([]byte(" "), maxAssemblySourceBytes)}
				source = []byte(strings.Repeat("#include \"header.h\"\n", 5))
				want = "includes exceed their input byte bound"
			case "disk-file-limit":
				writePreprocessFile(t, filepath.Join(pkg.Dir, "header.h"), strings.Repeat(" ", maxAssemblySourceBytes+1))
				source, want = []byte("#include \"header.h\"\n"), "source exceeds its input byte bound"
			case "wasm-macros":
				goos, arch = "js", "wasm"
				t.Setenv("GOWASM", "satconv,signext")
				source = []byte("#ifdef GOWASM_satconv\nWRONG\n#else\nRET\n#endif\n")
			case "unreviewed-sdk-version":
				writePreprocessFile(t, filepath.Join(opt.GOROOT, "VERSION"), "go1.99.1\n")
				want = "requires an audited observed Go"
			}
			got, err := PreprocessSourceForPkg(pkg, file, source, goos, arch, opt)
			if want == "" {
				if err != nil || string(got) != "RET\n" {
					t.Fatalf("target namespace changed: %q, %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), want) || len(got) != 0 {
				t.Fatalf("source boundary %q: got %q, %v; want %q and no output", mutation, got, err, want)
			}
		})
	}
}

func TestPreprocessFileAndTextSelectionBoundaries(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	writePreprocessFile(t, file, "// no entry\n")
	if got, err := PreprocessFileForPkg(pkg, filepath.Base(file), "linux", "amd64", opt); err != nil || HasTextAsmSource(got) {
		t.Fatal("relative package file selection changed", string(got), err)
	}
	for _, files := range [][]string{nil, {file}} {
		if found, err := HasAnyTextAsmForPkg(pkg, files, "linux", "amd64", opt); err != nil || found {
			t.Fatal("a no-TEXT selection invented an entry", found, err)
		}
	}
	if _, err := PreprocessFileForPkg(pkg, "", "linux", "amd64", opt); err == nil {
		t.Fatal("empty selected filename accepted")
	}
	if got, err := TranslateFileForPkgWithOptions(pkg, "", "linux", "amd64", nil, opt); err == nil || got != nil {
		t.Fatal("translation bypassed the same filename boundary", got, err)
	}
	if found, err := HasAnyTextAsmForPkg(pkg, []string{file + ".missing"}, "linux", "amd64", opt); err == nil || found {
		t.Fatal("missing selected file was treated as no assembly", found, err)
	}
}

func TestPreprocessSDKExperimentRegistrationBoundaries(t *testing.T) {
	const main = `package main
func main() {
 if objabi.LookupPkgSpecial(ctxt.Pkgpath).AllowAsmABI {
  for _, exp := range buildcfg.Experiment.Enabled() { flags.D = append(flags.D, "GOEXPERIMENT_"+exp) }
 }
}`
	for _, test := range []struct{ name, main, roles, tag, want string }{
		{"empty-tag", main, "", "goexperiment.", "invalid selected Go experiment tag"},
		{"uppercase-tag", main, "", "goexperiment.JSON", "invalid selected Go experiment tag"},
		{"punctuation-tag", main, "", "goexperiment.json_v2", "invalid selected Go experiment tag"},
		{"missing-main", "", "", "", "read selected SDK assembler registration"},
		{"malformed-main", "package", "", "", "expected"},
		{"duplicate-registration", main + "\nvar duplicate = \"GOEXPERIMENT_\"\n", "", "", "experiment registration"},
		{"wrong-condition", strings.Replace(main, "objabi.LookupPkgSpecial(ctxt.Pkgpath).AllowAsmABI", "true", 1), "", "", "package-role condition"},
		{"missing-roles", main, "", "", "read selected SDK assembler registration"},
		{"malformed-roles", main, "package", "", "expected"},
		{"duplicate-roles", main, "var allowAsmABIPkgs = []string{\"runtime\"}\nvar allowAsmABIPkgs = []string{}", "", "ambiguous"},
		{"array-roles", main, "var allowAsmABIPkgs = [1]string{\"runtime\"}", "", "nonliteral selected SDK assembler package role list"},
		{"wrong-element", main, "var allowAsmABIPkgs = []int{1}", "", "invalid selected SDK assembler package role list"},
		{"empty-path", main, "var allowAsmABIPkgs = []string{\"\"}", "", "invalid selected SDK assembler package role"},
		{"duplicate-path", main, "var allowAsmABIPkgs = []string{\"runtime\",\"runtime\"}", "", "invalid selected SDK assembler package role"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg, file, opt := preprocessFixture(t)
			pkg.PkgPath = "runtime"
			opt.ToolTags = []string{"goexperiment.jsonv2"}
			if test.tag != "" {
				opt.ToolTags = []string{test.tag}
			}
			if test.main != "" {
				writePreprocessFile(t, filepath.Join(opt.GOROOT, "src/cmd/asm/main.go"), test.main)
			}
			if test.roles != "" {
				roles := test.roles
				if roles != "package" {
					roles = "package objabi\n" + roles
				}
				writePreprocessFile(t, filepath.Join(opt.GOROOT, "src/cmd/internal/objabi/pkgspecial.go"), roles)
			}
			got, err := PreprocessSourceForPkg(pkg, file, []byte("RET\n"), "linux", "amd64", opt)
			if err == nil || !strings.Contains(err.Error(), test.want) || len(got) != 0 {
				t.Fatalf("SDK registration %s: %q, %v; want %q and no output", test.name, got, err, test.want)
			}
		})
	}
}

func TestPreprocessOverlayAliasesRequireIdenticalSource(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	alias := pkg.Dir + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Base(file)
	for _, source := range [][]byte{nil, {}, []byte("RET\n")} {
		opt.Overlay = map[string][]byte{file: source, alias: source}
		got, err := ReadFileWithOverlay(opt.Overlay, file)
		if source == nil {
			if err == nil || !strings.Contains(err.Error(), "deleted by overlay") {
				t.Fatal("identical deletion aliases fell back to disk", err)
			}
		} else if err != nil || !bytes.Equal(got, source) {
			t.Fatal("identical source aliases changed the selection", got, err)
		}
	}
	for _, other := range [][]byte{nil, []byte("NOP\n")} {
		opt.Overlay = map[string][]byte{file: []byte{}, alias: other}
		if _, err := ReadFileWithOverlay(opt.Overlay, file); err == nil || !strings.Contains(err.Error(), "conflicting") {
			t.Fatal("distinct empty/deleted/source aliases were silently selected", err)
		}
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal(fmt.Errorf("overlay tests unexpectedly wrote the selected source: %w", err))
	}
}
