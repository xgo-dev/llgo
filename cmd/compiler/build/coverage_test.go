package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"golang.org/x/tools/cover"
)

func coverageTestConfig(t *testing.T, mode Mode) *Config {
	t.Helper()
	// Most cases check the coverage protocol, not LLVM optimization. The LTO
	// cases opt into O2 explicitly; native core CI also exercises default Os.
	conf := NewDefaultConf(mode)
	conf.OptLevel = optlevel.O0
	return conf
}

func TestCoverageRegistration(t *testing.T) {
	const source = `package p

var GoCoverP uint32
var GoCover_0 [4]uint32
var GoCover_1 [5]uint32
var GoCoverM = [4]byte{1, 2, 3, 4}
var unusedA, unusedB int
func unused() {}
`
	fix := coverFixupConfig{
		MetaVar:            "GoCoverM",
		MetaLen:            4,
		MetaHash:           "00112233445566778899aabbccddeeff",
		Strategy:           "normal",
		CounterPrefix:      "GoCover",
		PkgIdVar:           "GoCoverP",
		CounterMode:        "atomic",
		CounterGranularity: "perblock",
	}
	data, err := coverageRegistration([]byte(source), fix, "example.org/p", -1)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"var GoCoverCounters = [...]struct",
		"{&GoCover_0[0], uint64(len(GoCover_0))}",
		"{&GoCover_1[0], uint64(len(GoCover_1))}",
		"var GoCoverP = GoCoverregister(",
		"runtime.registerCoverage",
	} {
		if !bytes.Contains(data, []byte(required)) {
			t.Errorf("missing %q:\n%s", required, data)
		}
	}
	for name, change := range map[string]func(*coverFixupConfig){
		"strategy":    func(f *coverFixupConfig) { f.Strategy = "future" },
		"granularity": func(f *coverFixupConfig) { f.CounterGranularity = "perfunc" },
		"hash":        func(f *coverFixupConfig) { f.MetaHash = "not hex" },
		"hash length": func(f *coverFixupConfig) { f.MetaHash = "00" },
		"mode":        func(f *coverFixupConfig) { f.CounterMode = "future" },
		"pkg ID":      func(f *coverFixupConfig) { f.PkgIdVar = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := fix
			change(&invalid)
			if _, err := coverageRegistration([]byte(source), invalid, "p", -1); err == nil {
				t.Fatal("accepted unsupported fixup protocol")
			}
		})
	}
	if _, err := coverageRegistration([]byte("bad Go"), fix, "p", -1); err == nil {
		t.Fatal("accepted malformed generated source")
	}
}

func TestCoveragePackageIDs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src/internal/coverage/pkid.go")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"package coverage\nvar rtPkgs = [...]string{\"runtime\", \"internal/abi\"}\n",
		"package coverage\nvar rtPkgs = [...]string{someConstant}\n",
		"package coverage\n",
	} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		ids, err := coveragePackageIDs(root)
		if strings.Contains(source, `"runtime"`) {
			if err != nil || ids["runtime"] != -2 || ids["internal/abi"] != -3 {
				t.Fatalf("IDs = %v, %v", ids, err)
			}
		} else if err == nil {
			t.Fatalf("accepted incompatible table: %s", source)
		}
	}
}

func TestCoverageMainTemplates(t *testing.T) {
	for name, source := range map[string]string{
		"build":        coverageBuildMain,
		"build legacy": coverageBuildMainLegacy + coverageLegacyExitSupport,
		"current":      fmt.Sprintf(coverageTestMain, "set", "", []string{"p"}),
		"go120":        legacyCoverageMain("count", " in ./...", true),
		"go121":        legacyCoverageMain("atomic", "", false),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parser.ParseFile(token.NewFileSet(), "main.go", source+coverageExitSupport, parser.AllErrors); err != nil {
				t.Fatal(err)
			}
			if name == "go120" && strings.Contains(source, "llgoCoverTearDown, llgoCoverSnapshot") {
				t.Fatal("Go 1.20 uses the newer snapshot callback ABI")
			}
		})
	}
	body, err := addCoverageImport([]byte("package p\nfunc F() {}\n"), "_", "example.org/counters")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "p.go", body, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	if _, err := addCoverageImport([]byte("not Go"), "_", "p"); err == nil {
		t.Fatal("accepted invalid instrumented source")
	}
	source := []byte("//line /original/p.go:20\npackage p\n\nfunc F() {}\n")
	body, err = addCoverageImport(source, "_", "example.org/counters")
	if err != nil {
		t.Fatal(err)
	}
	var positions []token.Position
	for _, input := range [][]byte{source, body} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "generated.go", input, 0)
		if err != nil {
			t.Fatal(err)
		}
		decl := file.Decls[len(file.Decls)-1]
		positions = append(positions, fset.Position(decl.Pos()))
	}
	if positions[0].String() != positions[1].String() {
		t.Fatalf("added import shifted source location: %v", positions)
	}
	atomic := []byte("//line /original/p.go:20\npackage p; import _cover_atomic_ \"sync/atomic\"\nfunc F() {}\n")
	body, err = addCoverageImport(atomic, "_", "example.org/counters")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "p.go", body, parser.AllErrors); err != nil {
		t.Fatalf("atomic coverage import: %v\n%s", err, body)
	}
}

func TestCoverageOptions(t *testing.T) {
	dir := t.TempDir()
	commands := commandEnv{dir: dir}
	if got, err := newCoverageBuild(&Config{}, commands); got != nil || err != nil {
		t.Fatalf("disabled coverage = %v, %v", got, err)
	}
	for _, mode := range []string{"", "set", "count", "atomic"} {
		t.Run("mode="+mode, func(t *testing.T) {
			conf := &Config{
				Mode: ModeTest,
				Coverage: &CoverageConfig{
					Mode:    mode,
					Profile: "cover.out",
				},
			}
			c, err := newCoverageBuild(conf, commands)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			if mode == "" {
				mode = "set"
			}
			if c.options.Mode != mode || c.options.Profile != filepath.Join(dir, "cover.out") {
				t.Fatalf("resolved options: %+v", c.options)
			}
			data, err := os.ReadFile(c.options.Profile)
			if err != nil || string(data) != "mode: "+mode+"\n" {
				t.Fatalf("profile = %q, %v", data, err)
			}
		})
	}
	for name, conf := range map[string]*Config{
		"bad mode": {
			Mode:     ModeTest,
			Coverage: &CoverageConfig{Mode: "invalid"},
		},
		"run": {
			Mode:     ModeRun,
			Coverage: &CoverageConfig{},
		},
		"build profile": {
			Mode:     ModeBuild,
			Coverage: &CoverageConfig{Profile: "cover.out"},
		},
		"shared library": {
			Mode:      ModeBuild,
			BuildMode: BuildModeCShared,
			Coverage:  &CoverageConfig{},
		},
		"target": {
			Mode:     ModeTest,
			Target:   "wasi",
			Coverage: &CoverageConfig{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if c, err := newCoverageBuild(conf, commands); err == nil {
				c.close()
				t.Fatal("expected error")
			}
		})
	}
	t.Run("compile only", func(t *testing.T) {
		profile := filepath.Join(dir, "must-not-exist.out")
		c, err := newCoverageBuild(&Config{
			Mode:        ModeTest,
			CompileOnly: true,
			Coverage:    &CoverageConfig{Profile: profile},
		}, commands)
		if err != nil {
			t.Fatal(err)
		}
		defer c.close()
		if _, err := os.Stat(profile); !os.IsNotExist(err) {
			t.Fatalf("-c created a profile: %v", err)
		}
	})
}

func TestCoverageProfileSurvivesInvocationClone(t *testing.T) {
	dir := t.TempDir()
	conf := &Config{
		Mode:     ModeTest,
		Coverage: &CoverageConfig{Mode: "count", Profile: "cover.out"},
	}
	first, err := newCoverageBuild(conf, commandEnv{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	first.close()
	profile := filepath.Join(dir, "cover.out")
	const fragment = "example.go:1.1,1.2 1 1\n"
	f, err := os.OpenFile(profile, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(fragment); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	child := conf.clone()
	second, err := newCoverageBuild(child, commandEnv{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	second.close()
	data, err := os.ReadFile(profile)
	if err != nil || string(data) != "mode: count\n"+fragment {
		t.Fatalf("profile after child initialization = %q, %v", data, err)
	}
}

func TestCoverageFuzzFlags(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		wantErr bool
	}{
		{[]string{"-test.fuzz=Fuzz"}, true},
		{[]string{"--test.fuzz=Fuzz"}, true},
		{[]string{"-test.fuzz", "Fuzz"}, true},
		{[]string{"--test.fuzz", "Fuzz"}, true},
		{[]string{"-test.fuzzworker"}, false},
		{[]string{"--", "-test.fuzz=Fuzz"}, false},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			for _, profile := range []string{"", "cover.out"} {
				dir := t.TempDir()
				conf := &Config{
					Mode:     ModeTest,
					RunArgs:  tc.args,
					Coverage: &CoverageConfig{Profile: profile},
				}
				c, err := newCoverageBuild(conf, commandEnv{dir: dir})
				if c != nil {
					c.close()
				}
				if tc.wantErr && profile != "" {
					const want = "cannot use -coverprofile flag with -fuzz flag"
					if err == nil || err.Error() != want {
						t.Fatalf("error = %v; want %s", err, want)
					}
					if _, err := os.Stat(filepath.Join(dir, profile)); !os.IsNotExist(err) {
						t.Fatalf("invalid arguments created a profile: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCoverageFailureResult(t *testing.T) {
	if err := coverageTestFailure([]error{nil}); err != ErrTestFailed {
		t.Fatalf("test failure = %v", err)
	}
	buildErr := errors.New("build failed")
	if err := coverageTestFailure([]error{nil, buildErr}); !errors.Is(err, buildErr) {
		t.Fatalf("lost build error: %v", err)
	}
}

func TestCoveragePrivateFlags(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "cover.out")
	const previous = "existing profile must survive invalid arguments\n"
	if err := os.WriteFile(profile, []byte(previous), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"-test.gocoverdir", "--test.gocoverdir",
		"-test.coverprofile", "--test.coverprofile",
	} {
		for _, args := range [][]string{{name + "=elsewhere"}, {name, "elsewhere"}, {name + "="}} {
			conf := &Config{
				Mode:     ModeTest,
				Coverage: &CoverageConfig{Profile: profile},
				RunArgs:  args,
			}
			c, err := newCoverageBuild(conf, commandEnv{dir: dir})
			if c != nil {
				c.close()
			}
			if err == nil || !strings.Contains(err.Error(), name+" is reserved") {
				t.Fatalf("private arguments %v: %v", args, err)
			}
			data, err := os.ReadFile(profile)
			if err != nil || string(data) != previous {
				t.Fatalf("invalid arguments truncated profile: %q, %v", data, err)
			}
		}
	}
	for name, conf := range map[string]*Config{
		"disabled": {
			Mode:    ModeTest,
			RunArgs: []string{"-test.gocoverdir=elsewhere"},
		},
		"compile only": {
			Mode:        ModeTest,
			CompileOnly: true,
			Coverage:    &CoverageConfig{},
			RunArgs:     []string{"-test.gocoverdir=elsewhere"},
		},
		"positional": {
			Mode:     ModeTest,
			Coverage: &CoverageConfig{},
			RunArgs:  []string{"--", "-test.gocoverdir=elsewhere", "-test.coverprofile=elsewhere"},
		},
		"similar name": {
			Mode:     ModeTest,
			Coverage: &CoverageConfig{},
			RunArgs:  []string{"-test.gocoverdirectory=elsewhere"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, err := newCoverageBuild(conf, commandEnv{dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if c != nil {
				c.close()
			}
		})
	}
}

func TestCoverageNoMetadata(t *testing.T) {
	c := &coverageBuild{
		noTests: []*packages.Package{{PkgPath: "example.org/empty"}},
	}
	for _, compileOnly := range []bool{false, true} {
		output := captureCoverageOutput(t, func() {
			if err := c.reportNoTests(&Config{CompileOnly: compileOnly}); err != nil {
				t.Fatal(err)
			}
		})
		want := "?   \texample.org/empty\t[no test files]\n"
		if compileOnly {
			want = ""
		}
		if string(output) != want {
			t.Fatalf("no-metadata output = %q; want %q", output, want)
		}
	}
}

func TestCoverageParallelProfiles(t *testing.T) {
	dir := t.TempDir()
	c, err := newCoverageBuild(&Config{
		Mode:     ModeTest,
		Coverage: &CoverageConfig{Profile: "cover.out", OutputDir: dir},
	}, commandEnv{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	c.manifest = []byte(`{"ImportPaths":[],"MetaFileFragments":[]}`)
	const runs = 16
	var wg sync.WaitGroup
	for i := range runs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			args, profile, err := c.runArgs([]string{"-test.run=Test"})
			if err != nil {
				t.Error(err)
				return
			}
			if len(args) != 3 || args[2] != "-test.run=Test" {
				t.Errorf("args: %v", args)
			}
			manifest, err := os.ReadFile(filepath.Join(filepath.Dir(profile), "metafiles.txt"))
			if err != nil || !bytes.Equal(manifest, c.manifest) {
				t.Errorf("manifest: %s, %v", manifest, err)
			}
			data := fmt.Sprintf("mode: set\np%d.go:1.1,2.1 1 1\n", i)
			if err := os.WriteFile(profile, []byte(data), 0600); err != nil {
				t.Error(err)
				return
			}
			if err := c.merge(profile); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(c.options.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "mode: ") != 1 || bytes.Count(data, []byte("\n")) != runs+1 {
		t.Fatalf("invalid merged profile:\n%s", data)
	}
	if err := c.merge(filepath.Join(dir, "missing.out")); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.out")
	if err := os.WriteFile(bad, []byte("mode: atomic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.merge(bad); err == nil {
		t.Fatal("accepted a mismatched profile mode")
	}
	if err := os.WriteFile(bad, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.merge(bad); err != nil {
		t.Fatalf("empty profile from an interrupted test: %v", err)
	}
}

func TestCoverageInputs(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.go")
	cgo := filepath.Join(dir, "cgo.go")
	testFile := filepath.Join(dir, "plain_test.go")
	translated := filepath.Join(dir, "translated-cache")
	translatedTest := filepath.Join(dir, "test-cache")
	generated := filepath.Join(dir, "wrappers-cache")
	p := &packages.Package{
		GoFiles:         []string{plain, cgo, testFile},
		CompiledGoFiles: []string{plain, generated, translated, testFile, translatedTest},
	}
	got, needsCgo := coverageInputs(p)
	if !needsCgo || !reflect.DeepEqual(got, []string{plain, cgo}) {
		t.Fatalf("coverage inputs = %v, needs cgo = %v", got, needsCgo)
	}
	p.GoFiles = []string{plain, testFile}
	got, needsCgo = coverageInputs(p)
	if needsCgo || !reflect.DeepEqual(got, []string{plain}) {
		t.Fatalf("ordinary inputs = %v, needs cgo = %v", got, needsCgo)
	}
}

func TestCoverageSourceOverlay(t *testing.T) {
	for _, cgo := range []bool{false, true} {
		t.Run(fmt.Sprintf("cgo=%v", cgo), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "p.go")
			source := []byte("package p\nfunc F() int { return 73 }\n")
			if cgo {
				source = []byte("package p\n// static int f() { return 73; }\nimport \"C\"\nfunc F() int { return int(C.f()) }\n")
			}
			base := map[string][]byte{input: source}
			for i := range 1000 {
				base[filepath.Join(dir, "unrelated", fmt.Sprintf("%d.go", i))] = []byte("package unrelated\n")
			}
			conf := &Config{
				Mode:     ModeTest,
				Coverage: &CoverageConfig{},
				Overlay:  make(map[string][]byte),
			}
			commands := commandEnv{dir: dir, environ: os.Environ()}
			c, err := newCoverageBuild(conf, commands)
			if err != nil {
				t.Fatal(err)
			}
			defer c.close()
			p := &packages.Package{
				ID:              "command-line-arguments",
				PkgPath:         "command-line-arguments",
				Name:            "p",
				Dir:             dir,
				GoFiles:         []string{input},
				CompiledGoFiles: []string{input},
			}
			if cgo {
				p.CompiledGoFiles = nil
			}
			meta, err := c.instrument(p, conf, &packages.Config{Dir: dir}, runtime.GOROOT(), 0, base)
			if err != nil {
				t.Fatal(err)
			}
			if meta == "" || len(p.CompiledGoFiles) == 0 {
				t.Fatal("overlay source was not instrumented")
			}
			if len(conf.Overlay) != 2 {
				t.Fatalf("worker returned %d entries; want only counters and instrumented source", len(conf.Overlay))
			}
			if len(base) != 1001 || !bytes.Equal(base[input], source) {
				t.Fatal("instrumentation modified the shared input overlay")
			}
		})
	}
}

func TestCoverageCgo(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs native cgo tests")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	const fixture = "./internal/build/testdata/coveragecgo"
	for _, mode := range []string{"set", "count", "atomic"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			goProfile := filepath.Join(dir, "go.out")
			cmd := exec.Command("go", "test", "-count=1", "-covermode="+mode, "-coverprofile="+goProfile, fixture)
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go test: %v\n%s", err, output)
			}
			llgoProfile := filepath.Join(dir, "llgo.out")
			conf := coverageTestConfig(t, ModeTest)
			conf.Coverage = &CoverageConfig{Mode: mode, Profile: llgoProfile}
			if _, err := Build(Invocation{Args: []string{fixture}, Dir: root, Config: conf}); err != nil {
				t.Fatal(err)
			}
			compareCoverageProfiles(t, goProfile, llgoProfile)
		})
	}
}

func TestCoverageAgainstGo(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs native test binaries")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	fingerprints := make(map[string]string)
	for _, mode := range []string{"set", "count", "atomic"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			fixture := filepath.Join(root, "internal/build/testdata/coverage/basic")
			goProfile := filepath.Join(dir, "go.out")
			cmd := exec.Command("go", "test", "-count=1", "-covermode="+mode, "-coverprofile="+goProfile, ".")
			cmd.Dir = fixture
			goOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go test: %v\n%s", err, goOutput)
			}
			conf := coverageTestConfig(t, ModeTest)
			conf.CompileOnly = true
			conf.OutFile = filepath.Join(dir, "basic.test")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			conf.Coverage = &CoverageConfig{Mode: mode}
			inv := Invocation{Args: []string{"."}, Dir: fixture, Config: conf}
			pkgs, err := Build(inv)
			if err != nil {
				t.Fatal(err)
			}
			for _, pkg := range pkgs {
				if pkg.Name == "basic" {
					if previous := fingerprints[pkg.Fingerprint]; previous != "" {
						t.Fatalf("%s shares a cache key with %s", mode, previous)
					}
					fingerprints[pkg.Fingerprint] = mode
				}
			}
			// One repeat proves cache reuse; the first build of each mode above
			// independently checks counter-mode fingerprint isolation.
			if mode == "set" {
				pkgs, err = Build(inv)
				if err != nil {
					t.Fatal(err)
				}
				var found bool
				for _, pkg := range pkgs {
					if pkg.Name == "basic" {
						found = true
						if !pkg.CacheHit {
							t.Error("identical covered package missed the archive cache")
						}
					}
				}
				if !found {
					t.Fatal("covered package not built")
				}
			}
			llgoProfile := filepath.Join(dir, "llgo.out")
			cmd = exec.Command(conf.OutFile, "-test.coverprofile="+llgoProfile)
			llgoOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("llgo test binary: %v\n%s", err, llgoOutput)
			}
			for _, output := range [][]byte{goOutput, llgoOutput} {
				if !bytes.Contains(output, []byte("coverage: 50.0% of statements")) {
					t.Fatalf("unexpected coverage: %s", output)
				}
			}
			compareCoverageProfiles(t, goProfile, llgoProfile)
			if mode == "atomic" {
				checkCoverageParallelExecution(t, conf.OutFile, fixture, llgoProfile)
			}
		})
	}
	conf := coverageTestConfig(t, ModeTest)
	conf.OutFile = filepath.Join(t.TempDir(), "plain.test")
	pkgs, err := Build(Invocation{
		Args:   []string{"./internal/build/testdata/coverage/basic"},
		Dir:    root,
		Config: conf,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		if pkg.Name == "basic" && fingerprints[pkg.Fingerprint] != "" {
			t.Fatal("ordinary package reused an instrumented archive key")
		}
	}
}

func checkCoverageParallelExecution(t *testing.T, app, dir, reference string) {
	t.Helper()
	// Both processes use an already-linked binary. A rendezvous during Build
	// would incorrectly treat a slow peer's compilation/linking as serialization.
	commands := commandEnv{
		dir:     dir,
		environ: withEnv(os.Environ(), "LLGO_COVER_BARRIER="+t.TempDir()),
	}
	conf := &Config{
		Mode:             ModeTest,
		BuildParallelism: 2,
		Coverage: &CoverageConfig{
			Mode:    "atomic",
			Profile: filepath.Join(t.TempDir(), "parallel.out"),
		},
	}
	c, err := newCoverageBuild(conf, commands)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	conf.coverage = c
	programs := []testProgram{
		{app: app, pkgDir: dir, pkgName: "first", coverage: true},
		{app: app, pkgDir: dir, pkgName: "second", coverage: true},
	}
	var stdout, stderr bytes.Buffer
	if result := runNativeTestPrograms(commands, programs, conf, &stdout, &stderr); result.failed || result.skipped != 0 {
		t.Fatalf("parallel covered tests: %+v\n%s\n%s", result, &stdout, &stderr)
	}
	want, err := cover.ParseProfiles(reference)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range want {
		for i := range profile.Blocks {
			profile.Blocks[i].Count *= 2
		}
	}
	got, err := cover.ParseProfiles(c.options.Profile)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("parallel profile lost counters: got %+v, want %+v, error %v", got, want, err)
	}
}

func compareCoverageProfiles(t *testing.T, wantFile, gotFile string) {
	t.Helper()
	want, err := cover.ParseProfiles(wantFile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := cover.ParseProfiles(gotFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Fatalf("coverage mismatch\nwant: %s\ngot: %s", wantJSON, gotJSON)
	}
}

func TestCoverageLTO(t *testing.T) {
	if testing.Short() {
		t.Skip("builds optimized test binaries")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, mode := range []lto.Mode{lto.Thin, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := coverageTestConfig(t, ModeTest)
			conf.Coverage = &CoverageConfig{}
			conf.LTO = mode
			conf.OptLevel = optlevel.O2
			conf.OutFile = filepath.Join(t.TempDir(), "basic.test")
			_, err := Build(Invocation{
				Args:   []string{"./internal/build/testdata/coverage/basic"},
				Dir:    root,
				Config: conf,
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCoverageMultiPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs several test binaries")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	pattern := "./internal/build/testdata/coverage/..."
	for _, selection := range []string{"", pattern} {
		t.Run("coverpkg="+selection, func(t *testing.T) {
			dir := t.TempDir()
			goProfile := filepath.Join(dir, "go.out")
			args := []string{"test", "-count=1", "-coverprofile=" + goProfile}
			if selection != "" {
				args = append(args, "-coverpkg="+selection)
			}
			cmd := exec.Command("go", append(args, pattern)...)
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go test: %v\n%s", err, output)
			}
			conf := coverageTestConfig(t, ModeTest)
			conf.BuildParallelism = 2
			conf.Coverage = &CoverageConfig{
				Packages: selection,
				Profile:  filepath.Join(dir, "llgo.out"),
			}
			conf.TestJSON = true
			conf.BuildTrace = filepath.Join(dir, "trace.json")
			output := captureCoverageOutput(t, func() {
				if _, err := Build(Invocation{Args: []string{pattern}, Dir: root, Config: conf}); err != nil {
					t.Error(err)
				}
			})
			compareCoverageProfiles(t, goProfile, conf.Coverage.Profile)
			trace, err := os.ReadFile(conf.BuildTrace)
			if err != nil {
				t.Fatal(err)
			}
			var events []buildTraceEvent
			if err := json.Unmarshal(trace, &events); err != nil {
				t.Fatal(err)
			}
			var instrumented bool
			for _, event := range events {
				if strings.HasPrefix(event.Name, "coverage ") && event.Phase == "X" {
					instrumented = true
				}
			}
			if !instrumented {
				t.Fatal("build trace does not include coverage instrumentation")
			}
			var passed int
			for _, line := range bytes.Split(bytes.TrimSpace(output), []byte("\n")) {
				var event struct {
					Action  string
					Test    string
					Elapsed *float64
				}
				if err := json.Unmarshal(line, &event); err != nil {
					t.Fatalf("invalid test2json event: %s: %v", line, err)
				}
				if event.Action == "pass" && event.Test == "" {
					passed++
					// test2json rounds to milliseconds; zero is a valid duration.
					if event.Elapsed == nil || *event.Elapsed < 0 {
						t.Errorf("missing or invalid package elapsed time: %s", line)
					}
				}
			}
			if passed != 4 {
				t.Fatalf("got %d passed test packages:\n%s", passed, output)
			}
		})
	}
}

func captureCoverageOutput(t *testing.T, run func()) []byte {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = f
	defer func() {
		os.Stdout = old
		f.Close()
	}()
	run()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	output, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func TestCoverageAPIsAndExit(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs covered test binaries")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, mode := range []string{"set", "atomic"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			conf := coverageTestConfig(t, ModeTest)
			conf.Coverage = &CoverageConfig{Mode: mode}
			conf.CompileOnly = true
			conf.OutFile = filepath.Join(dir, "api.test")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			_, err := Build(Invocation{
				Args:   []string{"./internal/build/testdata/coverage/api"},
				Dir:    root,
				Config: conf,
			})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(conf.OutFile)
			cmd.Env = append(os.Environ(), "LLGO_COVER_TEST_APIS=1")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("coverage APIs: %v\n%s", err, output)
			}
			cmd = exec.Command(conf.OutFile)
			cmd.Env = append(os.Environ(), "LLGO_COVER_EARLY_EXIT=1", "GOCOVERDIR="+dir)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("early exit: %v\n%s", err, output)
			}
			cmd = exec.Command("go", "tool", "covdata", "percent", "-i="+dir)
			output, err := cmd.CombinedOutput()
			if err != nil || !bytes.Contains(output, []byte("coverage: 66.7% of statements")) {
				t.Fatalf("early-exit counters: %v\n%s", err, output)
			}
			profile := filepath.Join(dir, "failed.out")
			cmd = exec.Command(conf.OutFile, "-test.run=TestFailureProfile", "-test.coverprofile="+profile)
			cmd.Env = append(os.Environ(), "LLGO_COVER_TEST_FAIL=1")
			if output, err := cmd.CombinedOutput(); err == nil || !bytes.Contains(output, []byte("intentional coverage failure")) {
				t.Fatalf("failure fixture: %v\n%s", err, output)
			}
			profiles, err := cover.ParseProfiles(profile)
			if err != nil || len(profiles) == 0 {
				t.Fatalf("missing failure profile: %v", err)
			}
		})
	}
}

func TestCoverageFailedPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs failing covered tests")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("LLGO_COVER_TEST_FAIL", "1")
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "failed.out")
			conf := coverageTestConfig(t, ModeTest)
			conf.Coverage = &CoverageConfig{Profile: profile}
			conf.TestJSON = jsonOutput
			conf.RunArgs = []string{"-test.run=TestFailureProfile"}
			output := captureCoverageOutput(t, func() {
				_, err := Build(Invocation{
					Args:   []string{"./internal/build/testdata/coverage/api"},
					Dir:    root,
					Config: conf,
				})
				if err != ErrTestFailed {
					t.Fatalf("test failure = %v", err)
				}
			})
			if !bytes.Contains(output, []byte("intentional coverage failure")) ||
				!bytes.Contains(output, []byte("coverage: 66.7% of statements")) {
				t.Fatalf("missing failure or coverage report:\n%s", output)
			}
			if bytes.Contains(output, []byte("exit code")) {
				t.Fatalf("LLGo-specific exit diagnostic in test output:\n%s", output)
			}
			if jsonOutput {
				failed := false
				for _, line := range bytes.Split(bytes.TrimSpace(output), []byte("\n")) {
					var event struct {
						Action string
						Test   string
					}
					if err := json.Unmarshal(line, &event); err != nil {
						t.Fatalf("non-JSON output: %s", line)
					}
					failed = failed || event.Action == "fail" && event.Test == ""
				}
				if !failed {
					t.Fatalf("missing package failure event:\n%s", output)
				}
			}
			profiles, err := cover.ParseProfiles(profile)
			if err != nil || len(profiles) == 0 {
				t.Fatalf("failed package profile: %v", err)
			}
		})
	}
}

func TestCoverageStandardLibrary(t *testing.T) {
	if testing.Short() {
		t.Skip("instruments the standard-library dependency graph")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	profile := filepath.Join(dir, "cover.out")
	conf := coverageTestConfig(t, ModeTest)
	conf.Coverage = &CoverageConfig{
		Mode:     "atomic",
		Packages: "all",
		Profile:  profile,
	}
	_, err = Build(Invocation{
		Args:   []string{"./internal/build/testdata/coverage/basic"},
		Dir:    root,
		Config: conf,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"tool", "cover", "-func=" + profile},
		{"tool", "cover", "-html=" + profile, "-o=" + filepath.Join(dir, "cover.html")},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("profile has invalid source coordinates: %v\n%s", err, output)
		}
	}
}
