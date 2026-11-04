package build

import (
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"go/ast"
	"go/importer"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/ssa/abi"
	llvm "github.com/xgo-dev/llvm"
)

var nativeComponentSelected = []string{"entry.s", "codebook.s", "helper.s", "ordinary.s"}

func nativeComponentFiles(t *testing.T) map[string]string {
	t.Helper()
	files := make(map[string]string)
	for _, name := range append([]string{"go.mod", "main.go"}, nativeComponentSelected...) {
		contents, err := os.ReadFile(filepath.Join("testdata", "plan9asm-native-component", name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(contents)
	}
	return files
}

func nativeComponentTableFiles(t *testing.T, slots int) map[string]string {
	t.Helper()
	files := nativeComponentFiles(t)
	call := "\tCALL sharedHelper(SB)\n"
	parts := strings.SplitN(files["codebook.s"], "\tCALL", 2)
	if len(parts) != 2 {
		t.Fatal("missing codebook source CALL boundary")
	}
	files["codebook.s"] = parts[0] + strings.Repeat(call, slots)
	return files
}

// The split form keeps the exact DATA directive but selects it as an independent
// assembly translation unit. Merely finding the file on disk grants no role.
func nativeComponentDataLayout(t *testing.T, files map[string]string, split bool) []string {
	t.Helper()
	selected := append([]string(nil), nativeComponentSelected...)
	if split {
		parts := strings.SplitN(files["entry.s"], "GLOBL", 2)
		if len(parts) != 2 {
			t.Fatal("missing original DATA/GLOBL boundary")
		}
		files["entry.s"] = parts[0]
		files["addresses.s"] = "#include \"textflag.h\"\nGLOBL" + parts[1]
		selected = append(selected, "addresses.s")
	}
	return selected
}

func nativeComponentPackage(t *testing.T, files map[string]string, selected []string, goos, arch, triple string) (*context, *aPackage) {
	t.Helper()
	ctx, apkg := nativeAddressFixturePackageAtPath(t, files, "example.com/plan9asm-native-component", goos, arch, triple)
	ctx.cTransformer = cabi.NewTransformer(ctx.prog, triple, "", true)
	pkg := apkg.Package
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	checker := types.Config{Sizes: pkg.TypesSizes, Importer: importer.Default()}
	typed, err := checker.Check(pkg.PkgPath, pkg.Fset, pkg.Syntax, info)
	if err != nil {
		t.Fatal(err)
	}
	pkg.Types, pkg.TypesInfo = typed, info
	pkg.OtherFiles = nil
	for _, name := range selected {
		pkg.OtherFiles = append(pkg.OtherFiles, filepath.Join(pkg.Dir, name))
	}
	if variable, ok := typed.Scope().Lookup("tableByte").(*types.Var); ok && types.Identical(variable.Type().Underlying(), types.Typ[types.Uint8]) {
		mod := apkg.LPkg.Module()
		global := llvm.AddGlobal(mod, mod.Context().Int8Type(), "callbackTable")
		global.SetInitializer(llvm.ConstNull(global.GlobalValueType()))
	}
	return ctx, apkg
}

// This reads ordinary relocatable-object bytes, not a disassembler or an
// instruction-translation fallback. A CALL table's physical slot size is part
// of its address contract even though the fixture never executes these slots.
func nativeComponentCodePrefix(t *testing.T, path, name string, size int) []byte {
	t.Helper()
	if object, err := elf.Open(path); err == nil {
		defer object.Close()
		symbols, err := object.Symbols()
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range symbols {
			if symbol.Name == name && int(symbol.Section) < len(object.Sections) && symbol.Section != elf.SHN_UNDEF {
				section := object.Sections[symbol.Section]
				data, err := section.Data()
				start := symbol.Value - section.Addr
				if err != nil || start > uint64(len(data)) || uint64(size) > uint64(len(data))-start {
					t.Fatalf("ELF code prefix: %v, symbol=%+v", err, symbol)
				}
				return append([]byte(nil), data[start:start+uint64(size)]...)
			}
		}
	} else if object, err := macho.Open(path); err == nil {
		defer object.Close()
		if object.Symtab == nil {
			t.Fatal("Mach-O object has no symbol table")
		}
		for _, symbol := range object.Symtab.Syms {
			if symbol.Name == "_"+name && symbol.Sect > 0 && int(symbol.Sect) <= len(object.Sections) {
				section := object.Sections[symbol.Sect-1]
				data, err := section.Data()
				start := symbol.Value - section.Addr
				if err != nil || start > uint64(len(data)) || uint64(size) > uint64(len(data))-start {
					t.Fatalf("Mach-O code prefix: %v, symbol=%+v", err, symbol)
				}
				return append([]byte(nil), data[start:start+uint64(size)]...)
			}
		}
	} else {
		t.Fatalf("not a native relocatable object: %s", path)
	}
	t.Fatalf("missing native code symbol %s in %s", name, path)
	return nil
}

func TestPlan9AsmNativeComponentObjects(t *testing.T) {
	for _, target := range []struct{ goos, arch, triple string }{
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"linux", "arm64", "aarch64-unknown-linux-gnu"},
		{"darwin", "amd64", "x86_64-apple-darwin"},
		{"darwin", "arm64", "arm64-apple-darwin"},
	} {
		t.Run(target.goos+"/"+target.arch, func(t *testing.T) {
			for _, slots := range []int{4, 2000} {
				t.Run(strconv.Itoa(slots), func(t *testing.T) {
					for _, layout := range []struct {
						name  string
						split bool
					}{{"inlineDATA", false}, {"splitDATA", true}} {
						t.Run(layout.name, func(t *testing.T) {
							files := nativeComponentTableFiles(t, slots)
							selected := nativeComponentDataLayout(t, files, layout.split)
							ctx, apkg := nativeComponentPackage(t, files, selected, target.goos, target.arch, target.triple)
							signatures, err := plan9asmSigsForPkg(ctx, apkg.PkgPath)
							ordinary := abi.PathOf(apkg.Types) + ".Ordinary"
							if _, present := signatures[ordinary]; err != nil || !present || len(signatures) != 1 {
								t.Fatalf("native components changed the actual Go declaration inventory: %v, %v", signatures, err)
							}
							objects, err := compilePkgSFiles(ctx, apkg, apkg.Package, false)
							for _, object := range objects {
								t.Cleanup(func() { os.Remove(object) })
							}
							if err != nil || len(objects) != len(selected) {
								t.Fatalf("selected component objects: %v, %v", objects, err)
							}
							for _, name := range []string{abi.PathOf(apkg.Types) + ".entry", "callbackTable"} {
								global := apkg.LPkg.Module().NamedGlobal(name)
								if global.IsNil() || !global.IsDeclaration() {
									t.Fatalf("native data/code address owner %s was not externalized", name)
								}
							}
							helperIR, err := os.ReadFile(apkg.ExportFile + "helper.s.ll")
							if err != nil {
								t.Fatal(err)
							}
							definitions := regexp.MustCompile(`(?m)^define [^\n@]*@([A-Za-z_][A-Za-z_0-9.]*)\(`).FindAllStringSubmatch(string(helperIR), -1)
							if len(definitions) != 1 {
								t.Fatalf("expected one native helper definition:\n%s", helperIR)
							}
							tableIR, err := os.ReadFile(apkg.ExportFile + "codebook.s.ll")
							if err != nil {
								t.Fatal(err)
							}
							if !strings.Contains(string(tableIR), "ptr @"+definitions[0][1]) {
								t.Fatalf("cross-file CALL does not reference the actual helper object:\n%s", tableIR)
							}
							templates := regexp.MustCompile(`asm sideeffect "([^"]*)"`).FindAllStringSubmatch(string(tableIR), -1)
							if len(templates) != 1 {
								t.Fatalf("expected one original codebook assembly body:\n%s", tableIR)
							}
							assembly := strings.NewReplacer(`\0A`, "\n", `\09`, "\t").Replace(templates[0][1])
							if regexp.MustCompile(`(?mi)^\s*ret(?:q)?(?:\s|$)`).MatchString(assembly) {
								t.Fatalf("codebook ending in CALL acquired a synthetic RET:\n%s", assembly)
							}
							slot := 5
							if target.arch == "arm64" {
								slot = 4
							}
							// compilePkgSFiles retains selected-file order, including the Go
							// declaration's separate typed object rather than skipping it.
							code := nativeComponentCodePrefix(t, objects[1], "callbackTable", slots*slot)
							for offset := 0; offset < len(code); offset += slot {
								if target.arch == "amd64" && code[offset] != 0xe8 {
									t.Fatalf("CALL slot at %d is not rel32: %x", offset, code)
								}
								if target.arch == "arm64" && binary.LittleEndian.Uint32(code[offset:])&0xfc000000 != 0x94000000 {
									t.Fatalf("CALL slot at %d is not BL: %x", offset, code)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestPlan9AsmNativeComponentRejectsUnprovedSources(t *testing.T) {
	base := nativeComponentFiles(t)
	for _, name := range []string{"unselected helper", "unreachable sibling", "duplicate bare symbol", "non-address alias", "alias write", "alias pointer escape", "non-byte alias", "foreign alias target", "typed Go CALL borrows native entry", "unselected DATA unit", "duplicate DATA units"} {
		t.Run(name, func(t *testing.T) {
			files := make(map[string]string, len(base)+1)
			for key, value := range base {
				files[key] = value
			}
			selected := append([]string(nil), nativeComponentSelected...)
			switch name {
			case "unselected DATA unit":
				selected = nativeComponentDataLayout(t, files, true)
				selected = selected[:len(selected)-1]
			case "duplicate DATA units":
				selected = nativeComponentDataLayout(t, files, true)
				files["duplicate-data.s"] = files["addresses.s"]
				selected = append(selected, "duplicate-data.s")
			case "unselected helper":
				selected = []string{"entry.s", "codebook.s", "ordinary.s"}
			case "unreachable sibling":
				files["rogue.s"] = "#include \"textflag.h\"\nTEXT rogue(SB),NOSPLIT|NOFRAME,$0-0\nRET\n"
				selected = append(selected, "rogue.s")
			case "duplicate bare symbol":
				files["duplicate.s"] = files["helper.s"]
				selected = append(selected, "duplicate.s")
			case "non-address alias":
				files["main.go"] = strings.Replace(files["main.go"], "uintptr(unsafe.Pointer(&tableByte))", "uintptr(tableByte)", 1) + "\nvar _ = unsafe.Pointer(nil)\n"
			case "alias write":
				files["main.go"] += "\nfunc writeCode() { tableByte = 0 }\n"
			case "alias pointer escape":
				files["main.go"] += "\nvar escapedCodePointer = &tableByte\n"
			case "non-byte alias":
				files["main.go"] = strings.Replace(files["main.go"], "var tableByte byte", "var tableByte uint64", 1)
			case "foreign alias target":
				files["main.go"] = strings.Replace(files["main.go"], "tableByte callbackTable", "tableByte absentCodeTable", 1)
			case "typed Go CALL borrows native entry":
				files["ordinary.s"] = strings.Replace(files["ordinary.s"], "\tRET", "\tCALL sharedHelper(SB)\n\tRET", 1)
			}
			ctx, apkg := nativeComponentPackage(t, files, selected, "linux", "amd64", "x86_64-unknown-linux-gnu")
			signatures, err := plan9asmSigsForPkg(ctx, apkg.PkgPath)
			if err != nil {
				return // Source ownership may fail before code generation.
			}
			ordinary := abi.PathOf(apkg.Types) + ".Ordinary"
			if _, present := signatures[ordinary]; !present || len(signatures) != 1 {
				t.Fatalf("invalid component replaced the real Go signature: %v", signatures)
			}
			// Signature discovery identifies Go-callable roles. It need not
			// guess unresolved external native providers before compilation.
			objects, err := compilePkgSFiles(ctx, apkg, apkg.Package, false)
			for _, object := range objects {
				t.Cleanup(func() { os.Remove(object) })
			}
			if err == nil {
				t.Fatalf("unproved component compiled: %v", objects)
			}
		})
	}
}

func TestPlan9AsmNativeComponentRechecksCompileSource(t *testing.T) {
	ctx, apkg := nativeComponentPackage(t, nativeComponentFiles(t), nativeComponentSelected, "linux", "amd64", "x86_64-unknown-linux-gnu")
	if _, err := plan9asmSigsForPkg(ctx, apkg.PkgPath); err != nil {
		t.Fatal(err)
	}
	// Signature success cannot turn the component into a stale source cache.
	// The selected helper no longer defines the referenced raw entry.
	changed := "#include \"textflag.h\"\nTEXT unrelated(SB),NOSPLIT|NOFRAME,$0-0\nRET\n"
	if err := os.WriteFile(filepath.Join(apkg.Dir, "helper.s"), []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	objects, err := compilePkgSFiles(ctx, apkg, apkg.Package, false)
	for _, object := range objects {
		t.Cleanup(func() { os.Remove(object) })
	}
	if err == nil {
		t.Fatalf("signature cache authorized changed component source: %v", objects)
	}
}

// The fixture must actually supply the byte-address Go declaration before its
// externalization is tested. Builtin byte and uint8 need not have equal Type
// object identity, even though they have the same Go storage type.
func TestPlan9AsmNativeComponentByteAliasFixture(t *testing.T) {
	_, apkg := nativeComponentPackage(t, nativeComponentFiles(t), nativeComponentSelected, "linux", "amd64", "x86_64-unknown-linux-gnu")
	variable, ok := apkg.Types.Scope().Lookup("tableByte").(*types.Var)
	if !ok {
		t.Fatal("missing actual Go byte alias declaration")
	}
	basic, ok := variable.Type().Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.Uint8 {
		t.Fatalf("actual byte alias type: %T %s", variable.Type(), variable.Type())
	}
	global := apkg.LPkg.Module().NamedGlobal("callbackTable")
	if global.IsNil() || global.GlobalValueType().TypeKind() != llvm.IntegerTypeKind || global.GlobalValueType().IntTypeWidth() != 8 || !global.Initializer().IsNull() {
		t.Fatalf("actual byte alias fixture global is missing or has wrong storage; Go type=%T %s", variable.Type(), variable.Type())
	}
}
