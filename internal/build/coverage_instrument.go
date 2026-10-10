// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
)

// Hard-coded runtime package IDs belong to the selected Go coverage protocol.
// Read its table instead of duplicating a list that changes between releases.
func coveragePackageIDs(goroot string) (map[string]int, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(goroot, "src/internal/coverage/pkid.go"), nil, 0)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]int)
	var tableErr error
	ast.Inspect(file, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if !ok || len(v.Names) != 1 || v.Names[0].Name != "rtPkgs" || len(v.Values) != 1 {
			return true
		}
		if list, ok := v.Values[0].(*ast.CompositeLit); ok {
			for i, elt := range list.Elts {
				if lit, ok := elt.(*ast.BasicLit); ok {
					path, err := strconv.Unquote(lit.Value)
					if err == nil {
						ids[path] = -i - 2
						continue
					}
				}
				tableErr = fmt.Errorf("unrecognized runtime coverage package ID at index %d", i)
				return false
			}
		}
		return false
	})
	if tableErr != nil {
		return nil, tableErr
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("runtime coverage package ID table not found in %s", goroot)
	}
	return ids, nil
}

func addCoverageSource(p *packages.Package, conf *Config, suffix string, data []byte) string {
	hash := sha256.Sum256([]byte(p.ID))
	name := filepath.Join(p.Dir, fmt.Sprintf("z_llgo_cover_%x_%s.go", hash[:6], suffix))
	conf.Overlay[name] = data
	p.GoFiles = append(p.GoFiles, name)
	p.CompiledGoFiles = append(p.CompiledGoFiles, name)
	return name
}

func coverageInputs(p *packages.Package) (inputs []string, needsCgo bool) {
	compiled := make(map[string]bool, len(p.CompiledGoFiles))
	for _, file := range p.CompiledGoFiles {
		compiled[file] = true
	}
	// GoFiles includes original cgo sources. Instrument before cgo rewriting,
	// as cmd/go does: generated wrappers and argument-checking closures are
	// implementation details, not additional user statements to count.
	for _, file := range p.GoFiles {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		inputs = append(inputs, file)
		needsCgo = needsCgo || !compiled[file]
	}
	return
}

func (c *coverageBuild) instrument(
	p *packages.Package,
	conf *Config,
	cfg *packages.Config,
	goroot string,
	pkgID int,
	baseOverlay map[string][]byte,
) (string, error) {
	inputs, needsCgo := coverageInputs(p)
	if len(inputs) == 0 {
		return "", nil
	}
	hash := sha256.Sum256([]byte(p.ID))
	dir := filepath.Join(c.dir, fmt.Sprintf("%x", hash[:12]))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	pcfg := coverPkgConfig{
		PkgPath:      p.PkgPath,
		PkgName:      p.Name,
		Local:        p.PkgPath == "command-line-arguments",
		Granularity:  "perblock",
		OutConfig:    filepath.Join(dir, "fixup.json"),
		EmitMetaFile: filepath.Join(dir, fmt.Sprintf("covmeta.%x", hash[:12])),
	}
	if p.Module != nil {
		pcfg.ModulePath = p.Module.Path
	}
	data, err := json.Marshal(pcfg)
	if err != nil {
		return "", err
	}
	configFile := filepath.Join(dir, "pkgcfg.json")
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return "", err
	}
	outputs := []string{filepath.Join(dir, "covervars.go")}
	toolInputs := slices.Clone(inputs)
	for i, input := range inputs {
		outputs = append(outputs, filepath.Join(dir, fmt.Sprintf("%d.cover.go", i)))
		if source, ok := baseOverlay[input]; ok {
			inputDir := filepath.Join(dir, fmt.Sprint(i))
			if err := os.MkdirAll(inputDir, 0700); err != nil {
				return "", err
			}
			toolInputs[i] = filepath.Join(inputDir, filepath.Base(input))
			if err := os.WriteFile(toolInputs[i], source, 0600); err != nil {
				return "", err
			}
		}
	}
	outlist := filepath.Join(dir, "outfiles.txt")
	toolOutputs := outputs
	if c.go120 {
		// Go 1.20 appends coverage declarations to the last instrumented
		// source file; Go 1.21+ writes a separate covervars.go first.
		toolOutputs = outputs[1:]
	}
	if err := os.WriteFile(outlist, []byte(strings.Join(toolOutputs, "\n")), 0600); err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("GoCover_%x_", hash[:6])
	args := []string{
		"tool", "cover",
		"-pkgcfg=" + configFile,
		"-mode=" + c.options.Mode,
		"-var=" + prefix,
		"-outfilelist=" + outlist,
	}
	cmd := c.commands.configure(exec.Command(filepath.Join(goroot, "bin", "go"), append(args, toolInputs...)...))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go tool cover: %w\n%s", err, output)
	}
	data, err = os.ReadFile(pcfg.OutConfig)
	if err != nil {
		return "", err
	}
	var fix coverFixupConfig
	if err := json.Unmarshal(data, &fix); err != nil {
		return "", err
	}
	if c.go120 {
		if err := extractLegacyCoverageVars(p.Name, fix.PkgIdVar, outputs[len(outputs)-1], outputs[0]); err != nil {
			return "", err
		}
	}
	vars, err := os.ReadFile(outputs[0])
	if err != nil {
		return "", err
	}
	if pkgID == 0 {
		pkgID = -1
	}
	vars, err = coverageRegistration(vars, fix, p.PkgPath, pkgID)
	if err != nil {
		return "", err
	}
	// Prepend the registration initializer. Go's initializer dependency order
	// also places it before any instrumented function that reads the package ID.
	oldFiles := slices.Clone(p.CompiledGoFiles)
	p.CompiledGoFiles = nil
	var counterPackage string
	if hasAltPkgForTarget(conf, p.PkgPath) {
		// Whole-package replacements may discard the original globals and
		// init, while generic functions still refer to original-source counters.
		// Keep those counters in a separate, unpatched package. Its dependency
		// initializes the registry before either implementation can use it.
		counterPackage = fmt.Sprintf("llgo.coverage/%x", hash[:12])
		counter := &packages.Package{
			ID:         counterPackage,
			PkgPath:    counterPackage,
			Name:       "llgocover",
			Dir:        p.Dir,
			Module:     p.Module,
			ExportFile: filepath.Join(dir, "counters.a"),
			Imports:    map[string]*packages.Package{"unsafe": p.Imports["unsafe"]},
		}
		vars = bytes.Replace(vars, []byte("package "+p.Name), []byte("package llgocover"), 1)
		addCoverageSource(counter, conf, "vars", vars)
		p.Imports[counterPackage] = counter
	} else {
		addCoverageSource(p, conf, "vars", vars)
	}
	for i, file := range inputs {
		body, err := os.ReadFile(outputs[i+1])
		if err != nil {
			return "", err
		}
		body = bytes.ReplaceAll(body, []byte(toolInputs[i]), []byte(inputs[i]))
		if counterPackage != "" {
			alias := "_"
			if bytes.Contains(body, []byte(prefix)) {
				alias = "llgoCoverage"
				body = bytes.ReplaceAll(body, []byte(prefix), []byte(alias+"."+prefix))
			}
			body, err = addCoverageImport(body, alias, counterPackage)
			if err != nil {
				return "", err
			}
		}
		if needsCgo {
			conf.Overlay[file] = body
		} else {
			addCoverageSource(p, conf, fmt.Sprint(i), body)
		}
	}
	if needsCgo {
		// Regenerate only this package's cgo outputs using the original build
		// flags, target environment and overlays. Do not parse/type-check here
		// or replace graph identities/imports; the shared frontend does that.
		if err := c.reloadCoverageCgo(p, conf, cfg, baseOverlay); err != nil {
			return "", err
		}
	} else {
		for _, file := range oldFiles {
			if !slices.Contains(inputs, file) {
				p.CompiledGoFiles = append(p.CompiledGoFiles, file)
			}
		}
	}
	if meta, err := os.Stat(pcfg.EmitMetaFile); os.IsNotExist(err) {
		// Go 1.20/1.21 do not emit standalone metadata for packages without
		// tests. Do not give their runtime a manifest of nonexistent files.
		return "", nil
	} else if err != nil {
		return "", err
	} else if meta.Size() == 0 {
		// Like cmd/go's WriteCoverMetaFilesFile, omit packages with no
		// functions. Their empty fragments are not valid metadata files.
		return "", nil
	}
	return pcfg.EmitMetaFile, nil
}

// Go 1.20 appends counter declarations to the last instrumented input. Separate
// them so registration runs before any user initializer reads the package ID.
func extractLegacyCoverageVars(pkgName, pkgIDVar, input, output string) error {
	body, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	marker := []byte("\nvar " + pkgIDVar + " uint32\n")
	pos := bytes.Index(body, marker)
	if pos < 0 {
		return fmt.Errorf("Go 1.20 coverage declarations not found")
	}
	vars := append([]byte("package "+pkgName+"\n"), body[pos:]...)
	if err := os.WriteFile(output, vars, 0600); err != nil {
		return err
	}
	return os.WriteFile(input, body[:pos], 0600)
}

func (c *coverageBuild) reloadCoverageCgo(p *packages.Package, conf *Config, cfg *packages.Config, baseOverlay map[string][]byte) error {
	loader := *cfg
	loader.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles
	loader.Tests = p.ForTest != ""
	loader.Env = c.commands.environ
	// Only cgo needs a merged overlay for a new package load. Keep dependency
	// patches visible to that loader without returning them as worker output.
	loader.Overlay = make(map[string][]byte, len(baseOverlay)+len(conf.Overlay))
	maps.Copy(loader.Overlay, baseOverlay)
	maps.Copy(loader.Overlay, conf.Overlay)
	patterns := []string{p.PkgPath}
	if p.PkgPath == "command-line-arguments" {
		patterns = p.GoFiles
	}
	loaded, err := packages.LoadMetadata(&loader, patterns...)
	if err != nil {
		return err
	}
	for _, updated := range loaded {
		if updated.ID != p.ID {
			continue
		}
		if len(updated.Errors) != 0 {
			return fmt.Errorf("regenerate covered cgo source: %v", updated.Errors)
		}
		p.CompiledGoFiles = updated.CompiledGoFiles
		return nil
	}
	return fmt.Errorf("covered cgo package %q not found", p.ID)
}

func addCoverageImport(source []byte, alias, path string) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "cover.go", source, parser.PackageClauseOnly)
	if err != nil {
		return nil, err
	}
	pos := fset.PositionFor(file.Name.End(), false).Offset
	// Atomic instrumentation inserts "; import sync/atomic" immediately
	// after the package name. Keep that semicolon with the package clause.
	if pos < len(source) && source[pos] == ';' {
		pos++
	}
	original := fset.Position(fset.File(file.Pos()).Pos(pos))
	// Keep the added import readable without shifting user-code locations.
	// The cover tool's source already carries an original-file line directive.
	decl := fmt.Sprintf("\nimport %s %q\n//line %s:%d",
		alias, path, original.Filename, original.Line)
	if original.Column != 0 {
		decl += ":" + strconv.Itoa(original.Column)
	}
	decl += "\n"
	return slices.Concat(source[:pos:pos], []byte(decl), source[pos:]), nil
}

func coverageRegistration(source []byte, fix coverFixupConfig, pkg string, pkgID int) ([]byte, error) {
	if fix.Strategy != "normal" || fix.CounterGranularity != "perblock" {
		return nil, fmt.Errorf("unsupported Go coverage strategy %q/%q", fix.Strategy, fix.CounterGranularity)
	}
	hash, err := hex.DecodeString(fix.MetaHash)
	if err != nil || len(hash) != 16 {
		return nil, fmt.Errorf("invalid coverage metadata hash %q", fix.MetaHash)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "covervars.go", source, 0)
	if err != nil {
		return nil, err
	}
	prefix := fix.CounterPrefix
	var counters strings.Builder
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok || len(v.Names) != 1 {
				continue
			}
			if _, ok := v.Type.(*ast.ArrayType); ok && strings.HasPrefix(v.Names[0].Name, prefix) {
				fmt.Fprintf(&counters, "{&%s[0], uint64(len(%s))},\n", v.Names[0].Name, v.Names[0].Name)
			}
		}
	}
	mode := map[string]int{"set": 1, "count": 2, "atomic": 3}[fix.CounterMode]
	if mode == 0 {
		return nil, fmt.Errorf("unsupported coverage mode %q", fix.CounterMode)
	}
	extra := fmt.Sprintf(coverageRegisterDecl, prefix, prefix)
	var h [16]byte
	copy(h[:], hash)
	initializer := fmt.Sprintf(coverageRegisterInit,
		prefix, counters.String(), fix.PkgIdVar, prefix, fix.MetaVar, fix.MetaLen, h, pkg, pkgID, mode, prefix)
	declaration := "var " + fix.PkgIdVar + " uint32"
	if !strings.Contains(string(source), declaration) {
		return nil, fmt.Errorf("coverage package ID variable %q not found", fix.PkgIdVar)
	}
	body := strings.Replace(string(source), declaration, initializer, 1)
	pos := strings.Index(body, "package "+file.Name.Name) + len("package "+file.Name.Name)
	body = body[:pos] + "\nimport \"unsafe\"\n" + body[pos:] + extra
	return format.Source([]byte(body))
}

const coverageRegisterDecl = `
//go:linkname %[1]sregister runtime.registerCoverage
func %[2]sregister(
	meta unsafe.Pointer,
	size uint32,
	hash [16]byte,
	pkgpath string,
	pkgid int,
	mode uint8,
	counters []struct {
		Counters *uint32
		Len uint64
	},
) uint32
`

const coverageRegisterInit = `var %sCounters = [...]struct {
	Counters *uint32
	Len uint64
}{
	%s
}

var %s = %sregister(
	unsafe.Pointer(&%s[0]),
	%d,
	%#v,
	%q,
	%d,
	%d,
	%sCounters[:],
)`
