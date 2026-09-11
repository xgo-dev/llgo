package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Full auditing deliberately has no compatibility allowlist. Source exclusions
// and failures remain unresolved until reviewed individually; independently
// executable host-side suites are named explicitly in the report.
type fullPackage struct {
	Package string `json:"package"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Tests   int    `json:"passed_top_level_tests"`
}

type selectedPackage struct {
	Dir                       string
	TestGoFiles, XTestGoFiles []string
	Error                     *struct{ Err string }
}

func discoverFull(root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(root, "test"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			seen[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, err
}

func fullProfile(name string) (profile, error) {
	switch name {
	case "GJS":
		return profile{Name: name, GOOS: "js"}, nil
	case "GWASI":
		return profile{Name: name, GOOS: "wasip1"}, nil
	default:
		return selectProfile(name)
	}
}

func fullCommand(p profile, goCmd, llgo, goRoot, pkg string) command {
	args := []string{"test", "-v", "-count=1", "-timeout=60s"}
	env := map[string]string{}
	program := llgo
	if p.Reference {
		program = goCmd
		args = append(args, "-exec="+strconv.Quote(filepath.Join(goRoot, "lib", "wasm", "go_"+p.GOOS+"_wasm_exec")))
		env["GOWASIRUNTIME"] = "wasmtime"
	}
	if p.Target != "" {
		args = append(args, "-target", p.Target, "-emulator")
	} else {
		env["GOOS"], env["GOARCH"], env["CGO_ENABLED"] = p.GOOS, "wasm", "0"
	}
	args = append(args, "./"+pkg)
	// Keep the LLGo package cache local to this job. Repeated stdlib builds can
	// reuse compilation while every test binary still executes with count=1.
	if !p.Reference {
		env["LLGO_BUILD_CACHE"] = "on"
	}
	// GNU timeout bounds compilation as well as execution, including children.
	return command{"timeout", append([]string{"--kill-after=10s", "5m", program}, args...), env}
}

func testWitness(pkg selectedPackage) (string, error) {
	for _, file := range append(append([]string{}, pkg.TestGoFiles...), pkg.XTestGoFiles...) {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pkg.Dir, file), nil, 0)
		if err != nil {
			return "", err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") && fn.Name.Name != "TestMain" && fn.Type.Params.NumFields() == 1 {
				return fn.Name.Name, nil
			}
		}
	}
	return "", errors.New("no top-level test witness; examples/benchmarks require separate accounting")
}

func runFull(name, reportPath, goCmd, llgo string, shard, shards int) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	return runFullAt(root, name, reportPath, goCmd, llgo, shard, shards, executeStructured, execute)
}

func runFullAt(root, name, reportPath, goCmd, llgo string, shard, shards int, structured, run func(string, command) ([]byte, error)) error {
	if reportPath == "" || shards < 1 || shard < 0 || shard >= shards {
		return errors.New("full audit requires report and valid shard/shards")
	}
	p, err := fullProfile(name)
	if err != nil {
		return err
	}
	names, err := discoverFull(root)
	if err != nil {
		return err
	}
	entries := make([]fullPackage, len(names))
	for i, pkg := range names {
		entries[i] = fullPackage{Package: pkg, Status: "not-run"}
		if i%shards != shard {
			entries[i].Status = "other-shard"
		}
	}
	result := "incomplete"
	save := func() error {
		data, err := json.MarshalIndent(struct {
			Profile  string        `json:"profile"`
			Result   string        `json:"result"`
			Shard    int           `json:"shard"`
			Shards   int           `json:"shards"`
			Packages []fullPackage `json:"packages"`
		}{name, result, shard, shards, entries}, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(reportPath, append(data, '\n'), 0644)
	}
	if err := save(); err != nil {
		return err
	}
	data, err := structured(root, command{goCmd, []string{"env", "GOROOT"}, nil})
	if err != nil {
		return err
	}
	goRoot := strings.TrimSpace(string(data))
	listArgs := []string{"list", "-e", "-json"}
	if !p.Reference {
		listArgs = append(listArgs, "-tags=llgo")
	}
	listArgs = append(listArgs, "./test/...")
	data, err = structured(root, command{goCmd, listArgs, map[string]string{"GOOS": p.GOOS, "GOARCH": "wasm", "CGO_ENABLED": "0"}})
	if err != nil {
		return err
	}
	selected := map[string]selectedPackage{}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var pkg selectedPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, pkg.Dir)
		if err != nil {
			return err
		}
		selected[filepath.ToSlash(rel)] = pkg
	}
	if err := os.MkdirAll(reportPath+".logs", 0755); err != nil {
		return err
	}
	failures := 0
	for i := range entries {
		e := &entries[i]
		if e.Status == "other-shard" {
			continue
		}
		pkg, exists := selected[e.Package]
		switch {
		case e.Package == "test/goroot":
			e.Status, e.Reason = "separate-suite", "host-side target runner is executed by the wasm GOROOT acceptance jobs"
		case !exists || len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) == 0:
			e.Status, e.Reason = "source-excluded", "no tests selected by source context; applicability not yet established"
		case pkg.Error != nil:
			e.Status, e.Reason = "fail", pkg.Error.Err
		default:
			witness, err := testWitness(pkg)
			if err != nil {
				e.Status, e.Reason = "unresolved", err.Error()
				break
			}
			fmt.Printf("%s %s\n", name, e.Package)
			out, runErr := run(root, fullCommand(p, goCmd, llgo, goRoot, e.Package))
			if err := os.WriteFile(filepath.Join(reportPath+".logs", strings.ReplaceAll(e.Package, "/", "_")+".log"), out, 0644); err != nil {
				return err
			}
			if runErr == nil {
				e.Tests, runErr = validateOutput(out, witness)
			}
			e.Status = "pass"
			if runErr != nil {
				e.Status, e.Reason = "fail", runErr.Error()
			}
		}
		if e.Status != "pass" && e.Status != "separate-suite" {
			failures++
		}
		fmt.Printf("%s %s: %s %s\n", name, e.Package, e.Status, e.Reason)
		if err := save(); err != nil {
			return err
		}
	}
	result = "pass"
	if failures != 0 {
		result = "fail"
	}
	if err := save(); err != nil {
		return err
	}
	if failures != 0 {
		return fmt.Errorf("%s shard %d/%d: %d unresolved/failed packages", name, shard, shards, failures)
	}
	return nil
}
