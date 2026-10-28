// ccgobridge produces an explicit source snapshot for audited CCGo ABI shapes.
// It never edits the module cache or enables unsafe rewrites in the compiler.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/ccgobridge"
	"github.com/xgo-dev/llgo/internal/puregobridge"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

func main() {
	dir := flag.String("dir", ".", "module directory whose dependency graph is adapted")
	out := flag.String("out", "", "new output directory (must not exist)")
	profile := flag.String("profile", "ccgo", "explicit ABI profile: ccgo or purego-v0.9.1")
	runtimeDir := flag.String("runtime", "", "LLGo runtime module directory (required by purego profile)")
	flag.Parse()
	if err := runProfile(*dir, *out, flag.Args(), *profile, *runtimeDir); err != nil {
		fmt.Fprintln(os.Stderr, "ccgobridge:", err)
		os.Exit(1)
	}
}

func run(dir, out string, patterns []string) error {
	return runProfile(dir, out, patterns, "ccgo", "")
}

func runProfile(dir, out string, patterns []string, profile, runtimeDir string) error {
	if out == "" || len(patterns) == 0 {
		return fmt.Errorf("usage: ccgobridge -dir module-dir -out new-dir package [package ...]")
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absOut); !os.IsNotExist(err) {
		return fmt.Errorf("output directory must not exist: %s", absOut)
	}
	if profile != "ccgo" && profile != "purego-v0.9.1" {
		return fmt.Errorf("unknown ABI profile %q", profile)
	}
	envCommand := exec.Command("go", "env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "GOVERSION", "GOMOD")
	envCommand.Dir = dir
	envData, err := envCommand.Output()
	if err != nil {
		return err
	}
	var environment map[string]string
	if err := json.Unmarshal(envData, &environment); err != nil {
		return err
	}
	if profile == "purego-v0.9.1" {
		if runtimeDir == "" {
			return fmt.Errorf("purego profile requires -runtime pointing to the LLGo runtime module")
		}
		runtimeDir, err = filepath.Abs(runtimeDir)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(runtimeDir, "go.mod"))
		if err != nil || modfile.ModulePath(data) != "github.com/xgo-dev/llgo/runtime" {
			return fmt.Errorf("-runtime is not the LLGo runtime module: %s", runtimeDir)
		}
	}
	pkgs, err := packages.Load(&packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedModule | packages.NeedEmbedFiles,
	}, patterns...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(pkgs) != 0 {
		return fmt.Errorf("source graph did not type-check; no overlay written")
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages matched")
	}
	if profile == "ccgo" {
		if err := validateExportOwners(pkgs); err != nil {
			return err
		}
	}

	type receipt struct {
		Package string
		Module  *packages.Module
		Stats   ccgobridge.Stats
	}
	var receipts []receipt
	files := make(map[string][]byte)
	removed := make(map[string]bool)
	modules := make(map[string]*packages.Module)
	for _, pkg := range pkgs {
		if pkg.Module == nil {
			return fmt.Errorf("%s is not in a module", pkg.PkgPath)
		}
		if pkg.Module.Main {
			return fmt.Errorf("%s is in the main module; this tool stages dependencies only", pkg.PkgPath)
		}
		modules[pkg.Module.Path] = pkg.Module
		if profile == "purego-v0.9.1" {
			if pkg.PkgPath != "github.com/ebitengine/purego" {
				return fmt.Errorf("purego profile must select github.com/ebitengine/purego, got %s", pkg.PkgPath)
			}
			changes, err := puregobridge.Adapt(pkg.Module.Path, pkg.Module.Version,
				environment["GOOS"], environment["GOARCH"], environment["CGO_ENABLED"] == "1", os.DirFS(pkg.Module.Dir))
			if err != nil {
				return err
			}
			for name, content := range changes.Files {
				files[filepath.Join(pkg.Module.Dir, name)] = content
			}
			for _, name := range changes.Remove {
				removed[filepath.Join(pkg.Module.Dir, name)] = true
			}
			receipts = append(receipts, receipt{Package: pkg.PkgPath, Module: pkg.Module})
			continue
		}
		stats, err := ccgobridge.Rewrite(pkg.Fset, pkg.Types, pkg.TypesInfo, pkg.Syntax)
		if err != nil {
			return err
		}
		receipts = append(receipts, receipt{pkg.PkgPath, pkg.Module, stats})
		if stats == (ccgobridge.Stats{}) {
			continue
		}
		for _, file := range pkg.Syntax {
			name := pkg.Fset.Position(file.Pos()).Filename
			var source bytes.Buffer
			if err := format.Node(&source, pkg.Fset, file); err != nil {
				return fmt.Errorf("format %s: %w", name, err)
			}
			files[name] = source.Bytes()
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("no recognized CCGo ABI sites; no overlay written")
	}
	// Replacing a module also replaces sibling packages that were not selected
	// for rewriting. Preserve every such package used by the application, not
	// merely dependencies of the selected exporter package.
	application, err := packages.Load(&packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedModule | packages.NeedEmbedFiles,
	}, "./...")
	if err != nil {
		return err
	}
	if packages.PrintErrors(application) != 0 {
		return fmt.Errorf("application dependency graph is incomplete; no snapshot written")
	}
	graph := append(append([]*packages.Package(nil), pkgs...), application...)

	// Go rejects overlays below GOMODCACHE. Stage just the selected target's
	// actual package files and use a private modfile instead of editing cached
	// dependencies or copying their test fixtures and other target variants.
	inputMod := environment["GOMOD"]
	data, err := os.ReadFile(inputMod)
	if err != nil {
		return err
	}
	mainMod, err := modfile.Parse(inputMod, data, nil)
	if err != nil {
		return err
	}
	// A modfile outside the application must not change the meaning of any
	// pre-existing local replacement, including unrelated dependencies.
	for _, replacement := range mainMod.Replace {
		if replacement.New.Version == "" && !filepath.IsAbs(replacement.New.Path) {
			path := filepath.Join(filepath.Dir(inputMod), replacement.New.Path)
			if err := mainMod.AddReplace(replacement.Old.Path, replacement.Old.Version, path, ""); err != nil {
				return err
			}
		}
	}
	if profile == "purego-v0.9.1" {
		const runtimeModule = "github.com/xgo-dev/llgo/runtime"
		if err := mainMod.AddRequire(runtimeModule, "v0.0.0-00010101000000-000000000000"); err != nil {
			return err
		}
		if err := mainMod.AddReplace(runtimeModule, "", runtimeDir, ""); err != nil {
			return err
		}
	}
	staged := make(map[string][]byte)
	var stageErr error
	stageFile := func(src, root, dest string) {
		if stageErr != nil || removed[src] {
			return
		}
		rel, err := filepath.Rel(root, src)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			stageErr = fmt.Errorf("package file %s is outside module %s", src, root)
			return
		}
		contents, ok := files[src]
		if !ok {
			contents, err = os.ReadFile(src)
			if err != nil {
				stageErr = err
				return
			}
		}
		staged[filepath.Join(dest, rel)] = contents
	}
	for path, module := range modules {
		key := sha256.Sum256([]byte(path))
		dest := filepath.Join(absOut, fmt.Sprintf("module-%x", key[:8]))
		for _, replacement := range append([]*modfile.Replace(nil), mainMod.Replace...) {
			if replacement.Old.Path == path {
				if err := mainMod.DropReplace(path, replacement.Old.Version); err != nil {
					return err
				}
			}
		}
		if err := mainMod.AddReplace(path, "", dest, ""); err != nil {
			return err
		}
		stageFile(filepath.Join(module.Dir, "go.mod"), module.Dir, dest)
		if _, err := os.Stat(filepath.Join(module.Dir, "go.sum")); err == nil {
			stageFile(filepath.Join(module.Dir, "go.sum"), module.Dir, dest)
		}
		packages.Visit(graph, func(pkg *packages.Package) bool {
			if pkg.Module != nil && pkg.Module.Path == path {
				for _, group := range [][]string{pkg.GoFiles, pkg.OtherFiles, pkg.EmbedFiles} {
					for _, src := range group {
						stageFile(src, module.Dir, dest)
					}
				}
			}
			return true
		}, nil)
		for src := range files {
			if strings.HasPrefix(src, module.Dir+string(filepath.Separator)) {
				stageFile(src, module.Dir, dest)
			}
		}
	}
	if stageErr != nil {
		return stageErr
	}
	modData, err := mainMod.Format()
	if err != nil {
		return err
	}
	staged[filepath.Join(absOut, "build.mod")] = modData
	if sums, err := os.ReadFile(strings.TrimSuffix(inputMod, ".mod") + ".sum"); err == nil {
		staged[filepath.Join(absOut, "build.sum")] = sums
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(absOut, 0o755); err != nil {
		return err
	}
	for dest, contents := range staged {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, contents, 0o644); err != nil {
			return err
		}
	}
	for name, value := range map[string]any{
		"adaptations.json": receipts,
		"environment.json": map[string]any{"profile": profile, "go": environment},
	} {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(absOut, name), append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("Build with -modfile=%s\n", filepath.Join(absOut, "build.mod"))
	fmt.Println("Adaptation counts are not build/link/run proof. Validate with both Go and LLGo.")
	return nil
}

// Compare declaration identities, not types.Func pointer addresses: separate
// package loaders may represent the same imported declaration with different
// objects. Generic instances are already rejected by ExportedFunctions.
func validateExportOwners(pkgs []*packages.Package) error {
	owners := make(map[string]string)
	for _, pkg := range pkgs {
		functions, err := ccgobridge.ExportedFunctions(pkg.Fset, pkg.TypesInfo, pkg.Syntax)
		if err != nil {
			return err
		}
		for _, fn := range functions {
			identity := fn.Pkg().Path() + "." + fn.Name()
			if owner, exists := owners[identity]; exists && owner != pkg.PkgPath {
				return fmt.Errorf("CCGo cross-package export of %s by %s and %s cannot preserve function-pointer identity; no snapshot written",
					identity, owner, pkg.PkgPath)
			}
			owners[identity] = pkg.PkgPath
		}
	}
	return nil
}
