package build

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestCoverageSetupErrors(t *testing.T) {
	t.Run("profile is directory", func(t *testing.T) {
		conf := &Config{Mode: ModeTest, Coverage: &CoverageConfig{Profile: t.TempDir()}}
		if c, err := newCoverageBuild(conf, commandEnv{}); err == nil {
			c.close()
			t.Fatal("accepted a directory as the output profile")
		}
	})
	t.Run("temporary directory unavailable", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		t.Setenv("TMPDIR", missing)
		t.Setenv("TMP", missing)
		if c, err := newCoverageBuild(&Config{Mode: ModeTest, Coverage: &CoverageConfig{}}, commandEnv{}); err == nil {
			c.close()
			t.Fatal("ignored temporary directory creation failure")
		}
	})
}

func TestCoverageRunErrors(t *testing.T) {
	dir := t.TempDir()
	c := &coverageBuild{
		dir:       filepath.Join(dir, "missing"),
		options:   CoverageConfig{Mode: "set", Profile: filepath.Join(dir, "merged.out")},
		goCommand: filepath.Join(dir, "missing-go"),
	}
	program := testProgram{app: filepath.Join(dir, "missing-test"), pkgName: "missing", coverage: true}
	conf := &Config{coverage: c, PrintCommands: true}
	var stdout, stderr bytes.Buffer
	if err := runCoveredTest(commandEnv{}, program, conf, &stdout, &stderr); err == nil {
		t.Fatal("ignored private directory creation failure")
	}
	c.dir = dir
	if err := runCoveredTest(commandEnv{}, program, conf, &stdout, &stderr); err == nil {
		t.Fatal("ignored test process start failure")
	}
	if !strings.Contains(stderr.String(), program.app) || !strings.Contains(stdout.String(), "FAIL\tmissing") {
		t.Fatalf("missing diagnostics: stdout=%s stderr=%s", &stdout, &stderr)
	}
	conf.TestJSON = true
	if err := runCoveredTest(commandEnv{}, program, conf, &stdout, &stderr); err == nil {
		t.Fatal("ignored test process start failure in JSON mode")
	}
	if err := c.merge(dir); err == nil {
		t.Fatal("accepted a directory as an input profile")
	}
	fragment := filepath.Join(dir, "fragment.out")
	if err := os.WriteFile(fragment, []byte("mode: set\np.go:1.1,1.2 1 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c.options.Profile = dir
	if err := c.merge(fragment); err == nil {
		t.Fatal("ignored merged profile write failure")
	}
	c.noTests = []*packages.Package{{PkgPath: "p"}}
	c.metaPaths = map[string]string{"p": filepath.Join(dir, "covmeta.missing")}
	if err := c.reportNoTests(&Config{}); err == nil {
		t.Fatal("ignored covdata failure")
	}
	c.metaPaths = nil
	if err := c.reportNoTests(&Config{TestJSON: true}); err == nil {
		t.Fatal("ignored no-test JSON converter failure")
	}
	t.Run("output closed", func(t *testing.T) {
		output, err := os.CreateTemp(dir, "closed-output")
		if err != nil {
			t.Fatal(err)
		}
		output.Close()
		original := os.Stdout
		os.Stdout = output
		defer func() { os.Stdout = original }()
		if err := c.reportNoTests(&Config{}); err == nil {
			t.Fatal("ignored report write failure")
		}
	})
}

func TestCoverageInstrumentErrors(t *testing.T) {
	for _, blocked := range []string{"package directory", "pkgcfg.json", "0", "0/p.go", "outfiles.txt", "go tool"} {
		t.Run(blocked, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "p.go")
			p := &packages.Package{
				ID:              "p",
				PkgPath:         "p",
				Name:            "p",
				Dir:             dir,
				GoFiles:         []string{input},
				CompiledGoFiles: []string{input},
			}
			hash := sha256.Sum256([]byte(p.ID))
			pkgdir := filepath.Join(dir, fmt.Sprintf("%x", hash[:12]))
			if err := os.MkdirAll(pkgdir, 0700); err != nil {
				t.Fatal(err)
			}
			c := &coverageBuild{dir: dir, options: CoverageConfig{Mode: "set"}}
			if blocked == "package directory" {
				c.dir = input
				if err := os.WriteFile(input, nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else if blocked == "0" {
				if err := os.WriteFile(filepath.Join(pkgdir, blocked), nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else if blocked != "go tool" {
				if err := os.MkdirAll(filepath.Join(pkgdir, blocked), 0700); err != nil {
					t.Fatal(err)
				}
			}
			base := map[string][]byte{input: []byte("package p\nfunc F() {}\n")}
			conf := &Config{Overlay: make(map[string][]byte)}
			if _, err := c.instrument(p, conf, nil, dir, 0, base); err == nil {
				t.Fatalf("ignored failure at %s", blocked)
			}
		})
	}
	if _, err := coveragePackageIDs(t.TempDir()); err == nil {
		t.Fatal("accepted missing runtime package IDs")
	}
	c := &coverageBuild{}
	if meta, err := c.instrument(&packages.Package{}, nil, nil, "", 0, nil); err != nil || meta != "" {
		t.Fatalf("empty package: metadata=%q error=%v", meta, err)
	}
}

func TestCoverageCgoReloadErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "p.go")
	if err := os.WriteFile(file, []byte("package p\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &coverageBuild{commands: commandEnv{environ: os.Environ()}}
	conf := &Config{}
	cfg := &packages.Config{Dir: dir}
	p := &packages.Package{ID: "missing", PkgPath: "command-line-arguments", GoFiles: []string{file}}
	if err := c.reloadCoverageCgo(p, conf, cfg, nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing package identity: %v", err)
	}
	p.ID = p.PkgPath
	if err := c.reloadCoverageCgo(p, conf, cfg, map[string][]byte{file: []byte("not Go")}); err == nil {
		t.Fatal("accepted invalid cgo source overlay")
	}
	cfg.Dir = filepath.Join(dir, "missing")
	if err := c.reloadCoverageCgo(p, conf, cfg, nil); err == nil {
		t.Fatal("ignored package loader failure")
	}
}

func TestCoveragePrepareErrors(t *testing.T) {
	dir := t.TempDir()
	c := &coverageBuild{dir: dir, options: CoverageConfig{Mode: "set"}, commands: commandEnv{dir: dir}}
	if err := c.prepare(nil, &packages.Config{}, &Config{}, dir); err == nil {
		t.Fatal("accepted missing GOROOT version")
	}
	c.options.Packages = "./..."
	cfg := &packages.Config{Dir: filepath.Join(dir, "missing"), Env: os.Environ()}
	if err := c.prepare(nil, cfg, &Config{}, runtime.GOROOT()); err == nil || !strings.Contains(err.Error(), "resolve -coverpkg") {
		t.Fatalf("missing coverpkg directory: %v", err)
	}
	c.options.Packages = ""
	if err := c.prepare(nil, cfg, &Config{}, runtime.GOROOT()); err == nil {
		t.Fatal("ignored support package loader failure")
	}
	cfg = &packages.Config{Mode: packages.NeedName, Dir: dir, Env: os.Environ()}
	p := &packages.Package{
		ID:              "example.org/p",
		PkgPath:         "example.org/p",
		Name:            "p",
		Dir:             dir,
		GoFiles:         []string{filepath.Join(dir, "missing.go")},
		CompiledGoFiles: []string{filepath.Join(dir, "missing.go")},
		Imports:         make(map[string]*packages.Package),
	}
	if err := c.prepare([]*packages.Package{p}, cfg, &Config{Mode: ModeTest}, runtime.GOROOT()); err == nil || !strings.Contains(err.Error(), "cover example.org/p:") {
		t.Fatalf("instrumentation failure lost package context: %v", err)
	}
}

func TestCoveragePrepareLegacyGraph(t *testing.T) {
	for _, mode := range []Mode{ModeTest, ModeBuild} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			dir := t.TempDir()
			p := &packages.Package{
				ID:      "example.org/p.test",
				PkgPath: "example.org/p.test",
				Name:    "main",
				Dir:     dir,
				Imports: make(map[string]*packages.Package),
			}
			c := &coverageBuild{
				dir:      dir,
				options:  CoverageConfig{Mode: "set", Packages: "unsafe"},
				commands: commandEnv{dir: dir},
			}
			// Metadata-only support packages model the old graph without cfile.
			// This checks wiring, not execution of an older Go runtime.
			cfg := &packages.Config{Mode: packages.NeedName, Dir: dir, Env: os.Environ()}
			conf := &Config{Mode: mode}
			if err := c.prepare([]*packages.Package{p}, cfg, conf, runtime.GOROOT()); err != nil {
				t.Fatal(err)
			}
			if len(conf.Overlay) != 1 || p.Imports["runtime/coverage"] == nil {
				t.Fatal("legacy main lost coverage support")
			}
			for file, source := range conf.Overlay {
				if _, err := parser.ParseFile(token.NewFileSet(), file, source, parser.AllErrors); err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(source, []byte("runtime/coverage")) {
					t.Fatal("legacy main uses the modern coverage hook")
				}
			}
		})
	}
}

func TestCoverageLegacyDeclarations(t *testing.T) {
	const body = "package p\nfunc F() {}\n"
	const declarations = "\nvar P uint32\nvar Counters [4]uint32\nvar Meta = [4]byte{1, 2, 3, 4}\n"
	for _, scenario := range []string{"valid", "missing input", "missing marker", "output directory"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "last.cover.go")
			output := filepath.Join(dir, "covervars.go")
			if scenario != "missing input" {
				source := body + declarations
				if scenario == "missing marker" {
					source = body
				}
				if err := os.WriteFile(input, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "output directory" {
				output = dir
			}
			err := extractLegacyCoverageVars("p", "P", input, output)
			if scenario != "valid" {
				if err == nil {
					t.Fatal("ignored legacy protocol/file error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for file, want := range map[string]string{input: body, output: "package p\n" + declarations} {
				got, err := os.ReadFile(file)
				if err != nil || string(got) != want {
					t.Fatalf("%s = %q, %v; want %q", file, got, err, want)
				}
				if _, err := parser.ParseFile(token.NewFileSet(), file, got, parser.AllErrors); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
