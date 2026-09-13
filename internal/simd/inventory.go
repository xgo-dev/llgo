/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package simd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"go/version"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// InventoryReferenceRevision is the reviewed official Go 1.27 source baseline.
// It is a comparison reference, not a claim about the selected GOROOT's Git
// revision. Sources and SourceSHA256 identify the bytes actually inspected.
const InventoryReferenceRevision = "2ee6421c51553e7164590445f96b123a858c1f4a"

const InventorySchema = 1

const inventoryPackage = "simd/archsimd"

// Inventory records source declarations, not compiler lowering coverage. A Go
// body can call other intrinsics; an absent body is only an intrinsic candidate.
// No entry in this inventory establishes LLVM, ABI, CPU, or runtime support.
type Inventory struct {
	Schema            int               `json:"schema"`
	ReferenceRevision string            `json:"reference_revision"`
	Package           string            `json:"package"`
	GoVersion         string            `json:"go_version"`
	SourceSHA256      string            `json:"source_sha256"`
	Sources           []InventorySource `json:"sources"`
	Targets           []InventoryTarget `json:"targets"`
}

// InventorySource identifies a package-relative source file. All non-test .go
// files are hashed, including files excluded by the audited build profiles, so
// a newly introduced OS/feature-specific source cannot escape drift detection.
type InventorySource struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

// InventoryTarget uses explicit audit profiles rather than the host's build
// tags: linux/amd64 v1, linux/arm64 v8.0, and wasip1/wasm, with SIMD enabled.
// The pinned archsimd API is OS-independent. Additional OS-specific APIs must
// expand these profiles when a source drift is reviewed.
type InventoryTarget struct {
	GOARCH  string           `json:"goarch"`
	GOOS    string           `json:"goos"`
	Entries []InventoryEntry `json:"entries"`
}

// InventoryEntry uses a stable package/type/method key without parameter names
// or source locations. Signature is a formatted source declaration with no
// function body or parameter/result names. Imported type qualifiers use import
// paths, not local aliases; this is not a resolved go/types type. The pinned
// package imports standard-library packages whose names match path basenames.
// Kind describes source only: type, variable, constant, go-body,
// intrinsic-candidate, or cpu-query. Body is present, absent, or not-applicable.
// LLGoStatus is unimplemented for bodyless candidates and unverified otherwise;
// neither an ordinary Go body nor a feature query implies compiler support.
type InventoryEntry struct {
	Key        string `json:"key"`
	Kind       string `json:"kind"`
	Signature  string `json:"signature"`
	Body       string `json:"body"`
	LLGoStatus string `json:"llgo_status"`
	Source     string `json:"source"`
	Line       int    `json:"line"`
}

// InventorySummary is the compact checked-in baseline. It stores counts and
// API digests instead of copying thousands of declarations. Generate a full
// Inventory and use CompareInventories for per-key addition/removal/signature
// diagnostics. A matching summary proves source inventory identity only.
type InventorySummary struct {
	Schema            int                      `json:"schema"`
	ReferenceRevision string                   `json:"reference_revision"`
	Package           string                   `json:"package"`
	GoLanguageVersion string                   `json:"go_language_version"`
	SourceSHA256      string                   `json:"source_sha256"`
	Sources           []InventorySource        `json:"sources"`
	Targets           []InventoryTargetSummary `json:"targets"`
}

type InventoryTargetSummary struct {
	GOARCH    string         `json:"goarch"`
	GOOS      string         `json:"goos"`
	APISHA256 string         `json:"api_sha256"`
	Entries   int            `json:"entries"`
	Kinds     map[string]int `json:"kinds"`
	Statuses  map[string]int `json:"llgo_statuses"`
}

// ScanInventory reads the selected GOROOT without invoking its compiler or
// importing its internal packages. It requires a Go 1.27 VERSION file, applies
// filename/build constraints, and never inherits host experiment/feature tags.
func ScanInventory(goroot string) (*Inventory, error) {
	versionBytes, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return nil, fmt.Errorf("simd inventory: read GOROOT VERSION: %w", err)
	}
	goVersion := strings.TrimSpace(strings.SplitN(string(versionBytes), "\n", 2)[0])
	if version.Lang(goVersion) != "go1.27" {
		return nil, fmt.Errorf("simd inventory: requires Go 1.27 source, found %q", goVersion)
	}
	dir := filepath.Join(goroot, "src", "simd", "archsimd")
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("simd inventory: read archsimd: %w", err)
	}
	inv := &Inventory{
		Schema: InventorySchema, ReferenceRevision: InventoryReferenceRevision,
		Package: inventoryPackage, GoVersion: goVersion,
	}
	sourceBytes := make(map[string][]byte)
	for _, file := range dirEntries {
		name := file.Name()
		if file.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("simd inventory: read %s: %w", name, err)
		}
		sourceBytes[name] = data
		inv.Sources = append(inv.Sources, InventorySource{name, inventorySHA256(data)})
	}
	if len(inv.Sources) == 0 {
		return nil, fmt.Errorf("simd inventory: no archsimd Go sources")
	}
	inv.SourceSHA256 = inventoryJSONHash(inv.Sources)
	for _, arch := range []string{"amd64", "arm64", "wasm"} {
		ctx := inventoryBuildContext(goroot, arch)
		// Match constraints against exactly the bytes being hashed and parsed,
		// even if another process modifies a source file during the scan.
		ctx.OpenFile = func(path string) (io.ReadCloser, error) {
			data, ok := sourceBytes[filepath.Base(path)]
			if !ok {
				return nil, os.ErrNotExist
			}
			return io.NopCloser(bytes.NewReader(data)), nil
		}
		target := InventoryTarget{GOARCH: arch, GOOS: ctx.GOOS}
		seen := make(map[string]bool)
		for _, source := range inv.Sources {
			match, err := ctx.MatchFile(dir, source.File)
			if err != nil {
				return nil, fmt.Errorf("simd inventory: %s/%s: %w", arch, source.File, err)
			}
			if !match {
				continue
			}
			entries, err := inventoryFile(source.File, sourceBytes[source.File])
			if err != nil {
				return nil, fmt.Errorf("simd inventory: %s/%s: %w", arch, source.File, err)
			}
			for _, entry := range entries {
				if seen[entry.Key] {
					return nil, fmt.Errorf("simd inventory: duplicate %s API %s", arch, entry.Key)
				}
				seen[entry.Key] = true
				target.Entries = append(target.Entries, entry)
			}
		}
		if len(target.Entries) == 0 {
			return nil, fmt.Errorf("simd inventory: no public declarations for %s", arch)
		}
		slices.SortFunc(target.Entries, func(a, b InventoryEntry) int { return strings.Compare(a.Key, b.Key) })
		inv.Targets = append(inv.Targets, target)
	}
	return inv, nil
}

func inventoryBuildContext(goroot, arch string) build.Context {
	ctx := build.Context{
		GOROOT: goroot, GOARCH: arch, GOOS: "linux", Compiler: "gc",
		ToolTags: []string{"goexperiment.simd"},
	}
	for n := 1; n <= 27; n++ {
		ctx.ReleaseTags = append(ctx.ReleaseTags, fmt.Sprintf("go1.%d", n))
	}
	switch arch {
	case "amd64":
		ctx.ToolTags = append(ctx.ToolTags, "amd64.v1")
	case "arm64":
		ctx.ToolTags = append(ctx.ToolTags, "arm64.v8.0")
	case "wasm":
		ctx.GOOS = "wasip1"
	}
	return ctx
}

func inventoryFile(name string, data []byte) ([]InventoryEntry, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, data, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	if file.Name.Name != "archsimd" {
		return nil, fmt.Errorf("expected package archsimd, found %s", file.Name.Name)
	}
	// Include imported type identity in signatures, so changing an import path
	// with an unchanged alias is API drift; renaming the alias alone is not.
	imports := make(map[string]string)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			return nil, fmt.Errorf("dot imports require type-aware inventory resolution")
		}
		imports[name] = importPath
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok {
				if importPath, ok := imports[ident.Name]; ok {
					selector.X = &ast.Ident{NamePos: ident.NamePos, Name: importPath}
				}
			}
		}
		return true
	})
	var result []InventoryEntry
	add := func(key, kind, body, status string, node ast.Node, pos token.Pos) error {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return err
		}
		result = append(result, InventoryEntry{
			Key: inventoryPackage + "." + key, Kind: kind, Signature: buf.String(),
			Body: body, LLGoStatus: status, Source: name, Line: fset.Position(pos).Line,
		})
		return nil
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			key, kind, body, status := d.Name.Name, "go-body", "present", "unverified"
			if d.Body == nil {
				kind, body, status = "intrinsic-candidate", "absent", "unimplemented"
			}
			if d.Recv != nil {
				recv, err := inventoryReceiver(d.Recv.List[0].Type)
				if err != nil {
					return nil, err
				}
				key = recv + "." + key
				if (recv == "X86Features" || recv == "ARM64Features") &&
					d.Type.Params.NumFields() == 0 && d.Type.Results.NumFields() == 1 {
					if typ, ok := d.Type.Results.List[0].Type.(*ast.Ident); ok && typ.Name == "bool" {
						kind = "cpu-query"
					}
				}
			}
			fn := *d
			ft := *d.Type
			ft.Params, ft.Results = inventoryUnnamed(ft.Params), inventoryUnnamed(ft.Results)
			fn.Recv, fn.Type, fn.Body = inventoryUnnamed(d.Recv), &ft, nil
			if err := add(key, kind, body, status, &fn, d.Pos()); err != nil {
				return nil, err
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						decl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{s}}
						if err := add(s.Name.Name, "type", "not-applicable", "unverified", decl, s.Pos()); err != nil {
							return nil, err
						}
					}
				case *ast.ValueSpec:
					for i, name := range s.Names {
						if !name.IsExported() {
							continue
						}
						kind := "variable"
						if d.Tok == token.CONST {
							kind = "constant"
						}
						value := *s
						value.Names = []*ast.Ident{name}
						if len(s.Values) == len(s.Names) {
							value.Values = []ast.Expr{s.Values[i]}
						}
						decl := &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{&value}}
						if err := add(name.Name, kind, "not-applicable", "unverified", decl, s.Pos()); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}
	return result, nil
}

func inventoryReceiver(expr ast.Expr) (string, error) {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name, nil
	case *ast.StarExpr:
		return inventoryReceiver(e.X)
	case *ast.IndexExpr:
		return inventoryReceiver(e.X)
	case *ast.IndexListExpr:
		return inventoryReceiver(e.X)
	case *ast.ParenExpr:
		return inventoryReceiver(e.X)
	}
	return "", fmt.Errorf("unsupported method receiver %T", expr)
}

func inventoryUnnamed(list *ast.FieldList) *ast.FieldList {
	if list == nil {
		return nil
	}
	copy := *list
	copy.List = nil
	for _, field := range list.List {
		for range max(1, len(field.Names)) {
			f := *field
			f.Names = nil
			copy.List = append(copy.List, &f)
		}
	}
	return &copy
}

// Summary omits locations from the API digest. Source text changes, including
// bodies, comments, and imports, are independently visible in the source hash.
func (inv *Inventory) Summary() InventorySummary {
	summary := InventorySummary{
		Schema: inv.Schema, ReferenceRevision: inv.ReferenceRevision, Package: inv.Package,
		GoLanguageVersion: version.Lang(inv.GoVersion), SourceSHA256: inv.SourceSHA256, Sources: inv.Sources,
	}
	for _, target := range inv.Targets {
		s := InventoryTargetSummary{GOARCH: target.GOARCH, GOOS: target.GOOS,
			Entries: len(target.Entries), Kinds: make(map[string]int), Statuses: make(map[string]int)}
		entries := slices.Clone(target.Entries)
		for i := range entries {
			entry := &entries[i]
			s.Kinds[entry.Kind]++
			s.Statuses[entry.LLGoStatus]++
			entry.Source, entry.Line = "", 0
		}
		s.APISHA256 = inventoryJSONHash(entries)
		summary.Targets = append(summary.Targets, s)
	}
	return summary
}

func inventorySHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func inventoryJSONHash(value any) string {
	// All callers pass structs containing only strings, integers, slices, and
	// string-keyed maps. encoding/json is deterministic for these values.
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return inventorySHA256(data)
}

// InventoryChange reports source drift separately from API/support drift.
type InventoryChange struct {
	Target string `json:"target,omitempty"`
	Key    string `json:"key"`
	Kind   string `json:"kind"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// CompareInventories reports additions, removals, signatures, body availability,
// classification, and recorded support status independently. Body contents and
// source-only changes are detected through source hashes, not misreported as
// function-signature changes.
func CompareInventories(before, after *Inventory) []InventoryChange {
	changes := compareInventoryHeaders(before.Summary(), after.Summary())
	index := func(inv *Inventory) map[string]InventoryEntry {
		entries := make(map[string]InventoryEntry)
		for _, target := range inv.Targets {
			for _, entry := range target.Entries {
				entries[target.GOOS+"/"+target.GOARCH+" "+entry.Key] = entry
			}
		}
		return entries
	}
	a, b := index(before), index(after)
	for id, old := range a {
		target, key, _ := strings.Cut(id, " ")
		current, ok := b[id]
		if !ok {
			changes = append(changes, InventoryChange{target, key, "removed", old.Signature, ""})
			continue
		}
		for _, field := range []struct{ kind, old, current string }{
			{"signature", old.Signature, current.Signature},
			{"classification", old.Kind, current.Kind},
			{"body", old.Body, current.Body},
			{"llgo-status", old.LLGoStatus, current.LLGoStatus},
		} {
			if field.old != field.current {
				changes = append(changes, InventoryChange{target, key, field.kind, field.old, field.current})
			}
		}
	}
	for id, current := range b {
		if _, ok := a[id]; !ok {
			target, key, _ := strings.Cut(id, " ")
			changes = append(changes, InventoryChange{target, key, "added", "", current.Signature})
		}
	}
	return sortInventoryChanges(changes)
}

// CompareInventorySummaries checks the compact pinned baseline. For precise
// changed API names, compare full inventories from the two source trees.
func CompareInventorySummaries(before, after InventorySummary) []InventoryChange {
	changes := compareInventoryHeaders(before, after)
	index := func(summary InventorySummary) map[string]InventoryTargetSummary {
		result := make(map[string]InventoryTargetSummary)
		for _, target := range summary.Targets {
			result[target.GOOS+"/"+target.GOARCH] = target
		}
		return result
	}
	a, b := index(before), index(after)
	for target, old := range a {
		current, ok := b[target]
		if !ok || inventoryJSONHash(old) != inventoryJSONHash(current) {
			changes = append(changes, InventoryChange{target, "api", "inventory", old.APISHA256, current.APISHA256})
		}
	}
	for target, current := range b {
		if _, ok := a[target]; !ok {
			changes = append(changes, InventoryChange{target, "api", "inventory", "", current.APISHA256})
		}
	}
	return sortInventoryChanges(changes)
}

func compareInventoryHeaders(before, after InventorySummary) []InventoryChange {
	var changes []InventoryChange
	for _, field := range []struct{ name, before, after string }{
		{"schema", fmt.Sprint(before.Schema), fmt.Sprint(after.Schema)},
		{"reference_revision", before.ReferenceRevision, after.ReferenceRevision},
		{"package", before.Package, after.Package},
		{"go_language_version", before.GoLanguageVersion, after.GoLanguageVersion},
		{"source_sha256", before.SourceSHA256, after.SourceSHA256},
	} {
		if field.before != field.after {
			changes = append(changes, InventoryChange{"", field.name, "provenance", field.before, field.after})
		}
	}
	old := make(map[string]string)
	for _, source := range before.Sources {
		old[source.File] = source.SHA256
	}
	for _, source := range after.Sources {
		if old[source.File] != source.SHA256 {
			changes = append(changes, InventoryChange{"", source.File, "source", old[source.File], source.SHA256})
		}
		delete(old, source.File)
	}
	for file, hash := range old {
		changes = append(changes, InventoryChange{"", file, "source", hash, ""})
	}
	return changes
}

func sortInventoryChanges(changes []InventoryChange) []InventoryChange {
	slices.SortFunc(changes, func(a, b InventoryChange) int {
		return strings.Compare(a.Target+" "+a.Key+" "+a.Kind, b.Target+" "+b.Key+" "+b.Kind)
	})
	return changes
}
