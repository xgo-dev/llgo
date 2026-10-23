package build

import (
	"go/build"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCgoDirectiveFlagsMatchGoQuoting(t *testing.T) {
	for _, flags := range []string{
		`-I"include space" -DNAME='two words'`,
		`-Iinclude\ space -DNAME=left' middle 'right`,
		`-framework Cocoa -weak_framework Metal`,
		`-DNAME="a\"b" -DOTHER='c\\d'`,
		`-O2 "" '-DNAME=with -flag text'`,
		"-I\tinclude -O2",
		`-I"unterminated`,
		`-Iunfinished\`,
	} {
		t.Run(flags, func(t *testing.T) {
			for _, kind := range []string{"CPPFLAGS", "CFLAGS", "CXXFLAGS", "LDFLAGS"} {
				t.Run(kind, func(t *testing.T) {
					dir := t.TempDir()
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
