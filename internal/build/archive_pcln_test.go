package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestPackagePCLNArchive(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct{ goos, goarch, triple string }{
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"darwin", "arm64", "arm64-apple-darwin"},
		{"windows", "amd64", "x86_64-pc-windows-msvc"},
		{"windows", "386", "i686-pc-windows-msvc"},
		{"wasip1", "wasm", "wasm32-unknown-wasip1"},
		{"wasip1", "wasm", "wasm64-unknown-wasip1"},
	} {
		for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
			t.Run(target.triple+"/"+mode.String(), func(t *testing.T) {
				prog := llssa.NewProgram(&llssa.Target{GOOS: target.goos, GOARCH: target.goarch, LLVMTarget: target.triple})
				defer prog.Dispose()
				src := prog.NewPackage("p", "example.com/p")
				src.EmitFuncInfoFlags("p.Trace.__llgo_fmv_avx2", "p.Trace", "路径/p.go", 10, 2, 1)
				src.EmitPCLineInfo(1<<63+7, "p.Trace.__llgo_fmv_avx2", "路径/p.go", 14, 3)
				wantFuncs := readFuncInfo(src.Module())
				wantLines := readPCLineInfo(src.Module())
				ctx := &context{prog: prog, buildConf: &Config{Goos: target.goos, Goarch: target.goarch, LTO: mode, PCLNMode: PCLNEmbedded}, crossCompile: crosscompile.Export{LLVMTarget: target.triple, CC: clang, CCFLAGS: []string{"--target=" + target.triple, "-flto"}}}
				pkg := &aPackage{Package: &packages.Package{ID: "p", PkgPath: "example.com/p", Name: "p"}, LPkg: src}
				// Mix the ordinary package object/bitcode and the always-native carrier.
				buf, _, err := emitObjectToMemoryBuffer(ctx, src)
				if err != nil {
					t.Fatal(err)
				}
				pkg.ObjBuffers = append(pkg.ObjBuffers, packageArchiveBuffer{"p.o", buf})
				defer pkg.disposeArchiveBuffers()
				if err := emitPackagePCLN(ctx, pkg); err != nil {
					t.Fatal(err)
				}
				if err := normalizeToArchive(ctx, pkg, false); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(pkg.ArchiveFile)
				raw, err := readPCLNArchiveMember(pkg.ArchiveFile)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.HasPrefix(raw, []byte{'B', 'C', 0xc0, 0xde}) {
					t.Fatal("metadata carrier is LTO bitcode")
				}
				for _, hit := range []bool{false, true} {
					// A reconstructed module and any previous parse must not override the
					// actual archive, regardless of cache status or backend disposal.
					loaded := &aPackage{Package: pkg.Package, ArchiveFile: pkg.ArchiveFile, CacheHit: hit, LPkg: prog.NewPackage("stale", "stale"), pcln: &packagePCLN{functions: []funcInfoRecord{{symbol: "stale"}}}}
					if err := loadPackagePCLN(loaded); err != nil {
						t.Fatal(err)
					}
					loaded.LPkg = nil
					if got := collectFuncInfo([]Package{loaded}); !reflect.DeepEqual(got, wantFuncs) {
						t.Fatalf("functions = %#v, want %#v", got, wantFuncs)
					}
					if got := collectPCLineInfo([]Package{loaded}); !reflect.DeepEqual(got, wantLines) {
						t.Fatalf("lines = %#v, want %#v", got, wantLines)
					}
				}
				ctx.buildConf.PCLNMode = PCLNNone
				if err := emitPackagePCLN(ctx, pkg); err != nil || len(pkg.ObjBuffers) != 0 {
					t.Fatalf("PCLN none emitted a carrier: %v", err)
				}
			})
		}
	}
}

func TestPCLNArchiveRejectsIncompleteArtifacts(t *testing.T) {
	member := func(name string, data []byte) []byte {
		header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n", name, 0, 0, 0, 0644, len(data))
		result := append([]byte(header), data...)
		if len(data)%2 != 0 {
			result = append(result, '\n')
		}
		return result
	}
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"empty", nil},
		{"missing", []byte("!<arch>\n")},
		{"header", []byte("!<arch>\nshort")},
		{"truncated", append([]byte("!<arch>\n"), member(packagePCLNMember+"/", []byte("abc"))[:61]...)},
		{"duplicate", append(append([]byte("!<arch>\n"), member(packagePCLNMember+"/", []byte("a"))...), member(packagePCLNMember+"/", []byte("b"))...)},
		{"bad BSD name", append([]byte("!<arch>\n"), member("#1/999", []byte("a"))...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pkg.a")
			if err := os.WriteFile(path, tc.raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := readPCLNArchiveMember(path); err == nil {
				t.Fatal("incomplete archive accepted")
			}
		})
	}
	for _, raw := range [][]byte{nil, []byte("not an object"), []byte("\x7fELF"), []byte("\x00asm"), []byte("\x00asm\x01\x00\x00\x00\x00\xff")} {
		if _, err := readPCLNObject(raw); err == nil {
			t.Fatalf("invalid object accepted: %q", raw)
		}
	}
	// The loader must not silently reconstruct mappings after a missing artifact.
	path := filepath.Join(t.TempDir(), "missing.a")
	if err := os.WriteFile(path, []byte("!<arch>\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{PkgPath: "p"}, ArchiveFile: path, pcln: &packagePCLN{functions: []funcInfoRecord{{symbol: "stale"}}}}
	if err := loadPackagePCLN(pkg); err == nil || !strings.Contains(err.Error(), "missing "+packagePCLNMember) || pkg.pcln != nil {
		t.Fatalf("missing artifact retained old mappings: %v", err)
	}
}
