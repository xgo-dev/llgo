package build

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestEmbedFilesChangePackageFingerprint(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		name := "package"
		if alternate {
			name = "alternate"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			first := filepath.Join(dir, "page.css")
			second := filepath.Join(dir, "outline.js")
			for _, path := range []string{first, second} {
				if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			embedded := &packages.Package{EmbedFiles: []string{first, second}}
			pkg := &aPackage{Package: &packages.Package{PkgPath: "example.com/p"}}
			if alternate {
				pkg.AltPkg = &packages.Cached{Package: embedded}
			} else {
				pkg.EmbedFiles = embedded.EmbedFiles
				embedded = pkg.Package
			}
			ctx := &context{mode: ModeGen, buildConf: &Config{}, sfilesCache: map[string][]string{pkg.PkgPath: nil}}
			fingerprint := func() string {
				t.Helper()
				m := newManifestBuilder()
				if err := ctx.collectPackageInputs(m, pkg); err != nil {
					t.Fatal(err)
				}
				return m.Fingerprint()
			}
			before := fingerprint()
			slices.Reverse(embedded.EmbedFiles)
			if fingerprint() != before {
				t.Fatal("embed input order changed the package fingerprint")
			}
			rewriteEmbedFilePreservingTime(t, first, "v2")
			if fingerprint() == before {
				t.Fatal("changing only embedded content did not invalidate the package cache")
			}
			embedded.EmbedFiles = append(embedded.EmbedFiles, filepath.Join(dir, "missing.txt"))
			if err := ctx.collectPackageInputs(newManifestBuilder(), pkg); err == nil || !strings.Contains(err.Error(), "embed") {
				t.Fatalf("missing embed input error = %v", err)
			}
		})
	}
}

func TestEmbedPackageCacheInvalidation(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "1")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
	root := writeMultiBuildModule(t, map[string]string{
		"main.go": `package main
import "example.com/multibuild/wrapper"
func main() { println(wrapper.Value()) }
`,
		"wrapper/wrapper.go": `package wrapper
import "example.com/multibuild/assets"
func Value() string { return assets.Value() }
`,
		"assets/assets.go": `package assets
import "embed"
//go:embed files/page.html files/page.css files/live.js files/outline.js tree/*.txt
var assets embed.FS
//go:embed files/outline.js
var script string
//go:embed files/page.css
var style []byte
func Value() string {
    result := ""
    for _, name := range []string{"page.html", "page.css", "live.js", "outline.js"} {
        data, err := assets.ReadFile("files/" + name)
        if err != nil { panic(err) }
        result += string(data) + "|"
    }
    result += script + "|" + string(style) + "|"
    entries, err := assets.ReadDir("tree")
    if err != nil { panic(err) }
    for _, entry := range entries {
        data, err := assets.ReadFile("tree/" + entry.Name())
        if err != nil { panic(err) }
        result += entry.Name() + ":" + string(data) + "|"
    }
    return result
}
`,
		"assets/files/page.html":  "html-v1",
		"assets/files/page.css":   "css-v1",
		"assets/files/live.js":    "live-v1",
		"assets/files/outline.js": "outline-v1",
		"assets/tree/a.txt":       "a",
	})
	build := func(want string, wantHit bool) {
		t.Helper()
		conf := NewDefaultConf(ModeBuild)
		conf.PCLNMode = PCLNNone
		conf.BuildParallelism = 2
		conf.OutFile = filepath.Join(t.TempDir(), "embedcache")
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: root})
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, pkg := range pkgs {
			if pkg.PkgPath == "example.com/multibuild/assets" || pkg.PkgPath == "example.com/multibuild/wrapper" {
				found++
				if pkg.CacheHit != wantHit {
					t.Errorf("%s CacheHit = %v, want %v", pkg.PkgPath, pkg.CacheHit, wantHit)
				}
			}
		}
		if found != 2 {
			t.Fatalf("found %d fixture dependencies, want 2", found)
		}
		assertBuiltProgram(t, conf.OutFile, want)
	}
	const original = "html-v1|css-v1|live-v1|outline-v1|outline-v1|css-v1|a.txt:a|"
	const updated = "html-v1|css-v2|live-v1|outline-v2|outline-v2|css-v2|a.txt:a|"
	build(original, false)
	build(original, true)
	rewriteEmbedFilePreservingTime(t, filepath.Join(root, "assets", "files", "page.css"), "css-v2")
	rewriteEmbedFilePreservingTime(t, filepath.Join(root, "assets", "files", "outline.js"), "outline-v2")
	build(updated, false)
	added := filepath.Join(root, "assets", "tree", "b.txt")
	if err := os.WriteFile(added, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	build(updated+"b.txt:b|", false)
	if err := os.Remove(added); err != nil {
		t.Fatal(err)
	}
	build(updated, true)
}

func rewriteEmbedFilePreservingTime(t *testing.T, path, content string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(content)) != info.Size() {
		t.Fatal("embed rewrite must preserve file size")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
}
