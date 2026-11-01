package build

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// These are whole reviewed Go1.26/1.27 sources, not an instruction allowlist.
// The expander's 26 PCs are private AX -> Z1/Z2 entries. Only after replacing
// both actual callers may we remove the table; no native ABI is inferred here.
var gcScanReviewedSources = map[string][]string{
	"expand_amd64.s": {
		"e04d1663dbe0cc0d20d429e24b301c9046b472ab5ca8fecfcb46640c55b4e554",
		"17f53567f6e395bbd03006cd912fb05d6ad4832b384c0f08ece1d10d5e483d1e",
	},
	"scan_amd64.s": {
		"1123e5160bb6920843be16d6d218a64352b34ce0f6b8ce2df9ed34c68fd642fb",
		"bb791b67d854cf352a0edd432cf86026b9dd8ae1d405105bc874ed27b93150a2",
	},
	"filter_amd64.s":  {"d75bc88f985ffca0e5d1fa0ea53ebc69478b8a4db4bd4618babda1a0f510ab75"},
	"expand_amd64.go": {"7e2a099fab42af62a4e7340425ffdb9cfb9b2a91b3886d29ee018e1ddddbbb84"},
	"scan_amd64.go": {
		"af4691e471daace679116cde58a47a4c9c435c525f91c45477da1d7a0d2669da",
		"274b655592508439929493836be2abbd6ae5011902582c53a7f31a3a11242017",
		"310d5ce0828d619645236caeba957118b1b232e35d8c2baa95176c2d84397dcb",
	},
	"filter_amd64.go": {"21e405b81d2200a049db1131aa6891d58cfd86aea41e3b3d2560dfa164e23e9c"},
}

func gcScanSourcePatchView(dir string, entries []os.DirEntry, buildCtx build.Context,
	read func(string) ([]byte, error), base map[string][]byte, body []byte, skips map[string]struct{},
) (map[string][]byte, error) {
	const marker = "// replaced by LLGo source patch\n"
	assemblyNames := []string{"expand_amd64.s", "scan_amd64.s", "filter_amd64.s"}
	view := make(map[string][]byte)
	reused := 0
	for _, name := range assemblyNames {
		filename := filepath.Join(dir, name)
		source, err := read(filename)
		if err != nil {
			return nil, err
		}
		if string(source) == marker {
			reused++
		}
		view[filename] = []byte(marker)
	}
	if reused != 0 && (reused != len(assemblyNames) ||
		!bytes.Equal(base[filepath.Join(dir, "z_llgo_patch_scan_amd64.go")], body)) {
		return nil, fmt.Errorf("incomplete or changed generated gc scan owner")
	}
	wantSkips := map[string]struct{}{
		"ExpandAVX512": {}, "scanSpanPackedAVX512": {}, "FilterNilAVX512": {},
		"llgoGCScanFactor": {}, "gcExpandersAVX512": {},
	}
	if !reflect.DeepEqual(skips, wantSkips) {
		return nil, fmt.Errorf("gc scan owner changed the exact declaration selection")
	}
	seen := make(map[string]bool)
	pcDeclarations := 0
	var declarations [][]byte
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_test.s") ||
			(!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
			continue
		}
		match, err := buildCtx.MatchFile(dir, name)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		if strings.HasSuffix(name, ".s") && gcScanReviewedSources[name] == nil {
			return nil, fmt.Errorf("new unowned gc scan assembly %s", name)
		}
		filename := filepath.Join(dir, name)
		source, err := read(filename)
		if err != nil {
			return nil, err
		}
		if reused != 0 && gcScanReviewedSources[name] != nil {
			original, err := os.ReadFile(filename)
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(name, ".go") {
				filtered, _, err := filterSourcePatchFile(original, skips)
				if err != nil || !bytes.Equal(source, filtered) {
					return nil, fmt.Errorf("changed generated gc scan declaration view %s", name)
				}
			}
			source = original
		}
		if hashes := gcScanReviewedSources[name]; hashes != nil {
			digest := fmt.Sprintf("%x", sha256.Sum256(source))
			accepted := false
			for _, hash := range hashes {
				accepted = accepted || digest == hash
			}
			if !accepted {
				return nil, fmt.Errorf("unreviewed complete gc scan source %s (%s)", name, digest)
			}
			seen[name] = true
		}
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		declarations = append(declarations, source)
		originalFile, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range originalFile.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range decl.Specs {
					if value, ok := spec.(*ast.ValueSpec); ok {
						for _, ident := range value.Names {
							if ident.Name == "gcExpandersAVX512" {
								pcDeclarations++
							}
						}
					}
				}
			}
		}
		filtered, _, err := filterSourcePatchFile(source, skips)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), filename, filtered, 0)
		if err != nil {
			return nil, err
		}
		referencesPCs := false
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && ident.Name == "gcExpandersAVX512" {
				referencesPCs = true
			}
			return true
		})
		if referencesPCs {
			return nil, fmt.Errorf("remaining Go consumer of private expander PCs in %s", name)
		}
	}
	if len(seen) != len(gcScanReviewedSources) || pcDeclarations != 1 {
		return nil, fmt.Errorf("incomplete gc scan source group")
	}
	originals, err := atomicSourcePatchFunctions(declarations)
	if err != nil {
		return nil, err
	}
	replacements, err := atomicSourcePatchFunctions([][]byte{body})
	if err != nil {
		return nil, err
	}
	if len(replacements) != 4 || originals["llgoGCScanFactor"] != nil {
		return nil, fmt.Errorf("gc scan replacement changed its complete body group")
	}
	if helper := replacements["llgoGCScanFactor"]; helper == nil || helper.Body == nil || len(helper.Body.List) == 0 {
		return nil, fmt.Errorf("gc scan owner lacks its actual bounded size-class helper")
	}
	for _, name := range []string{"ExpandAVX512", "scanSpanPackedAVX512", "FilterNilAVX512"} {
		before, after := originals[name], replacements[name]
		if before == nil || before.Body != nil || after == nil || after.Body == nil || len(after.Body.List) == 0 {
			return nil, fmt.Errorf("gc scan %s lacks its original declaration or real Go body", name)
		}
		want, err := atomicSourcePatchSignature(before.Type)
		if err != nil {
			return nil, err
		}
		got, err := atomicSourcePatchSignature(after.Type)
		if err != nil || !reflect.DeepEqual(want, got) || after.Recv != nil || after.Type.TypeParams.NumFields() != 0 {
			return nil, fmt.Errorf("gc scan %s replacement changed its exact signature", name)
		}
	}
	return view, nil
}
