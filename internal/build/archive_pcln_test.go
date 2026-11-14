package build

import (
	"bytes"
	"encoding/binary"
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
				if mode == lto.Off {
					// A valid target object without our section is not a mapping
					// artifact, even when it has no functions to describe.
					empty := prog.NewPackage("empty", "example.com/empty")
					buf, err := emitPCLNObject(ctx, pkg, empty.Module())
					if err != nil {
						t.Fatal(err)
					}
					_, err = readPCLNObject(buf.Bytes())
					buf.Dispose()
					if err == nil || !strings.Contains(err.Error(), "PCLN section") {
						t.Fatalf("object without mappings: %v", err)
					}
				}
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

func TestLoadPackagePCLNRejectsCorruptPayload(t *testing.T) {
	// A minimal Wasm custom section lets these checks exercise the complete
	// archive -> object -> payload loader without relying on a host object ABI.
	wasm := func(payload []byte) []byte {
		section := append([]byte{9}, []byte("llgo.pcln")...)
		section = append(section, payload...)
		raw := []byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0, 0}
		raw = binary.AppendUvarint(raw, uint64(len(section)))
		return append(raw, section...)
	}
	for _, tc := range []struct {
		name, want string
		object     []byte
	}{
		{"object", "read PCLN object", []byte("not an object")},
		{"short length", "invalid PCLN payload length", wasm([]byte{1})},
		{"oversized length", "invalid PCLN payload length", wasm(binary.LittleEndian.AppendUint64(nil, ^uint64(0)))},
		{"invalid JSON", "decode PCLN", wasm(append(binary.LittleEndian.AppendUint64(nil, 1), '{'))},
		{"wrong JSON type", "decode PCLN", wasm(append(binary.LittleEndian.AppendUint64(nil, 2), '[', ']'))},
		{"empty mappings with padding", "", wasm(append(binary.LittleEndian.AppendUint64(nil, 2), '{', '}', 0, 0))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pkg.a")
			raw := fmt.Appendf(nil, "!<arch>\n%-16s%-12d%-6d%-6d%-8o%-10d`\n", packagePCLNMember+"/", 0, 0, 0, 0644, len(tc.object))
			raw = append(raw, tc.object...)
			if len(tc.object)%2 != 0 {
				raw = append(raw, '\n')
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			pkg := &aPackage{Package: &packages.Package{PkgPath: "p"}, ArchiveFile: path, pcln: &packagePCLN{functions: []funcInfoRecord{{symbol: "stale"}}}}
			err := loadPackagePCLN(pkg)
			if tc.want == "" {
				if err != nil || pkg.pcln == nil || len(pkg.pcln.functions) != 0 || len(pkg.pcln.lines) != 0 {
					t.Fatalf("valid empty mappings: %v, %#v", err, pkg.pcln)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) || pkg.pcln != nil {
				t.Fatalf("corrupt artifact retained mappings: %v, %#v", err, pkg.pcln)
			}
		})
	}
	for _, path := range []string{"", filepath.Join(t.TempDir(), "absent.a")} {
		pkg := &aPackage{Package: &packages.Package{PkgPath: "p"}, ArchiveFile: path, pcln: &packagePCLN{}}
		err := loadPackagePCLN(pkg)
		if (err != nil) != (path != "") || pkg.pcln != nil {
			t.Fatalf("archive %q: %v, %#v", path, err, pkg.pcln)
		}
	}
	for _, raw := range [][]byte{
		// Empty and truncated names, and an unterminated name length.
		{'\x00', 'a', 's', 'm', 1, 0, 0, 0, 0, 0},
		{'\x00', 'a', 's', 'm', 1, 0, 0, 0, 0, 2, 9, 'x'},
		{'\x00', 'a', 's', 'm', 1, 0, 0, 0, 0, 1, 0x80},
	} {
		if _, err := readPCLNObject(raw); err == nil || !strings.Contains(err.Error(), "custom section name") {
			t.Fatalf("malformed Wasm name accepted: %v", err)
		}
	}
}

func TestEmitPackagePCLNFailure(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"})
	defer prog.Dispose()
	for _, tempFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("tempFailure=%v", tempFailure), func(t *testing.T) {
			dir := t.TempDir()
			tmp := dir
			if tempFailure {
				tmp = filepath.Join(dir, "missing")
			}
			for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(key, tmp)
			}
			ctx := &context{prog: prog, buildConf: &Config{}, crossCompile: crosscompile.Export{CC: filepath.Join(dir, "missing-compiler")}}
			pkg := &aPackage{Package: &packages.Package{PkgPath: "p"}, LPkg: prog.NewPackage("p", "p")}
			if err := emitPackagePCLN(ctx, pkg); err == nil || !strings.Contains(err.Error(), "emit PCLN object for p") {
				t.Fatalf("failed emission was not reported: %v", err)
			}
			if len(pkg.ObjBuffers) != 0 {
				pkg.disposeArchiveBuffers()
				t.Fatal("failed emission published an archive member")
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("temporary artifacts leaked: %v, %v", entries, err)
			}
		})
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
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	pkg.LPkg = prog.NewPackage("stale", "stale")
	pkg.LPkg.EmitFuncInfo("stale.F", "stale.F", "stale.go", 1, 1)
	pkg.LPkg.EmitPCLineInfo(1, "stale.F", "stale.go", 1, 1)
	if len(packageFuncInfo(pkg)) != 0 || len(packagePCLineInfo(pkg)) != 0 {
		t.Fatal("missing archive mappings fell back to the reconstructed frontend")
	}
}
