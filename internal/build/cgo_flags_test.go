package build

import (
	"fmt"
	"go/build"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCgoPreambleFlagsMatchGoPackagePaths(t *testing.T) {
	for index, flags := range []string{
		`-Iheaders -I "include space" -Llibs -L "lib space"`,
		`-I"${SRCDIR}/include space" -L"${SRCDIR}/lib space" -DROOT=${SRCDIR}`,
	} {
		for _, kind := range []string{"CPPFLAGS", "CFLAGS", "CXXFLAGS", "LDFLAGS"} {
			t.Run(fmt.Sprintf("%s/case%d", kind, index), func(t *testing.T) {
				dir := t.TempDir()
				filename := filepath.Join(dir, "fixture.go")
				preamble := "#cgo " + kind + ": " + flags + "\n"
				source := "package fixture\n/*\n" + preamble + "*/\nimport \"C\"\n"
				if err := os.WriteFile(filename, []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
				oracle := build.Default
				oracle.CgoEnabled = true
				pkg, err := oracle.ImportDir(dir, 0)
				if err != nil {
					t.Fatalf("original Go preamble: %v", err)
				}
				_, decls, err := parseCgoPreamble(token.Position{Filename: filename, Line: 2}, preamble)
				if err != nil {
					t.Fatal(err)
				}
				cflags, cxxflags, ldflags := collectCgoFlags(decls)
				got, want := cflags, pkg.CgoCFLAGS
				switch kind {
				case "CPPFLAGS":
					want = pkg.CgoCPPFLAGS
				case "CXXFLAGS":
					got, want = cxxflags, pkg.CgoCXXFLAGS
				case "LDFLAGS":
					got, want = ldflags, pkg.CgoLDFLAGS
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("preamble flags = %q; original Go package-relative flags = %q", got, want)
				}
			})
		}
	}
}

func TestCgoDirectiveFlagsMatchGoQuoting(t *testing.T) {
	for _, flags := range []string{
		`-I"include space" -DNAME='two words'`,
		`-Iinclude\ space -DNAME=left' middle 'right`,
		`-framework Cocoa -weak_framework Metal`,
		`-DNAME="a\ b" -DOTHER='c\ d'`,
		`-O2 '-DNAME=with -flag text'`,
		"-I\tinclude -O2",
		`-I"unterminated`,
		`-Iunfinished\`,
	} {
		t.Run(flags, func(t *testing.T) {
			for _, kind := range []string{"CPPFLAGS", "CFLAGS", "CXXFLAGS", "LDFLAGS"} {
				t.Run(kind, func(t *testing.T) {
					dir := t.TempDir()
					// go/build also resolves relative include paths. Use an
					// absolute path to isolate its independent quoting oracle.
					flags := strings.ReplaceAll(flags, "include", filepath.ToSlash(filepath.Join(dir, "include")))
					source := "package fixture\n/*\n#cgo " + kind + ": " + flags + "\n*/\nimport \"C\"\n"
					if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(source), 0o600); err != nil {
						t.Fatal(err)
					}
					oracle := build.Default
					oracle.CgoEnabled = true
					pkg, wantErr := oracle.ImportDir(dir, 0)
					decls, err := parseCgoDecl("#cgo " + kind + ": " + flags)
					if (err == nil) != (wantErr == nil) {
						t.Fatalf("parse error = %v; original Go error = %v", err, wantErr)
					}
					if wantErr != nil {
						return
					}
					cflags, cxxflags, ldflags := collectCgoFlags(decls)
					got, want := cflags, pkg.CgoCFLAGS
					switch kind {
					case "CPPFLAGS":
						want = pkg.CgoCPPFLAGS
					case "CXXFLAGS":
						got, want = cxxflags, pkg.CgoCXXFLAGS
					case "LDFLAGS":
						got, want = ldflags, pkg.CgoLDFLAGS
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("flags = %q; original Go flags = %q", got, want)
					}
				})
			}
		})
	}
}

func TestSplitCgoDirectiveFlagsPreservesLiteralTokens(t *testing.T) {
	// go/build.ImportDir additionally applies safeCgoName, which rejects
	// literal quotes and empty arguments after tokenization. Check the lexer
	// separately: it must preserve intentional bytes, not strip every quote.
	got, err := splitCgoDirectiveFlags(`-DNAME="a\"b" -DOTHER='c\\d' ""`)
	want := []string{`-DNAME=a"b`, `-DOTHER=c\d`, ""}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("literal tokens = %q, %v; want %q", got, err, want)
	}
}
