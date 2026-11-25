package build

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
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
			missing := filepath.Join(dir, "missing.txt")
			embedded.EmbedFiles = append(embedded.EmbedFiles, missing)
			err := ctx.collectPackageInputs(newManifestBuilder(), pkg)
			var pathErr *os.PathError
			if !errors.Is(err, os.ErrNotExist) || !errors.As(err, &pathErr) || pathErr.Path != missing {
				t.Fatalf("missing embed input error = %v", err)
			}
		})
	}
}

func TestEmbedPackageCacheInvalidation(t *testing.T) {
	isolateEmbedBuildCache(t)
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
		assertCachedEmbedProgram(t, root, want, wantHit)
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

func TestEmbedDirectoryCacheInvalidation(t *testing.T) {
	isolateEmbedBuildCache(t)
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
import (
    "embed"
    "io/fs"
)
//go:embed tree
var assets embed.FS
func Value() string {
    result := ""
    err := fs.WalkDir(assets, "tree", func(path string, entry fs.DirEntry, err error) error {
        if err != nil { return err }
        if entry.IsDir() {
            result += path + "/|"
            return nil
        }
        data, err := assets.ReadFile(path)
        if err != nil { return err }
        result += path + ":" + string(data) + "|"
        return nil
    })
    if err != nil { panic(err) }
    return result
}
`,
		"assets/tree/a.txt":        "a1",
		"assets/tree/nested/b.txt": "b1",
	})
	tree := filepath.Join(root, "assets", "tree")
	write := func(t *testing.T, name, content string) {
		t.Helper()
		path := filepath.Join(tree, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	steps := []struct {
		name    string
		change  func(*testing.T)
		want    string
		wantHit bool
	}{
		{name: "cold", want: "tree/|tree/a.txt:a1|tree/nested/|tree/nested/b.txt:b1|"},
		{name: "warm", want: "tree/|tree/a.txt:a1|tree/nested/|tree/nested/b.txt:b1|", wantHit: true},
		{name: "modify-file", change: func(t *testing.T) {
			rewriteEmbedFilePreservingTime(t, filepath.Join(tree, "a.txt"), "a2")
		}, want: "tree/|tree/a.txt:a2|tree/nested/|tree/nested/b.txt:b1|"},
		{name: "modify-nested-file", change: func(t *testing.T) {
			rewriteEmbedFilePreservingTime(t, filepath.Join(tree, "nested", "b.txt"), "b2")
		}, want: "tree/|tree/a.txt:a2|tree/nested/|tree/nested/b.txt:b2|"},
		{name: "add-file", change: func(t *testing.T) {
			write(t, "c.txt", "c1")
		}, want: "tree/|tree/a.txt:a2|tree/c.txt:c1|tree/nested/|tree/nested/b.txt:b2|"},
		{name: "add-nested-file", change: func(t *testing.T) {
			write(t, "nested/deep/d.txt", "d1")
		}, want: "tree/|tree/a.txt:a2|tree/c.txt:c1|tree/nested/|tree/nested/b.txt:b2|tree/nested/deep/|tree/nested/deep/d.txt:d1|"},
		{name: "delete-file", change: func(t *testing.T) {
			if err := os.Remove(filepath.Join(tree, "a.txt")); err != nil {
				t.Fatal(err)
			}
		}, want: "tree/|tree/c.txt:c1|tree/nested/|tree/nested/b.txt:b2|tree/nested/deep/|tree/nested/deep/d.txt:d1|"},
		{name: "rename-file", change: func(t *testing.T) {
			if err := os.Rename(filepath.Join(tree, "c.txt"), filepath.Join(tree, "renamed.txt")); err != nil {
				t.Fatal(err)
			}
		}, want: "tree/|tree/nested/|tree/nested/b.txt:b2|tree/nested/deep/|tree/nested/deep/d.txt:d1|tree/renamed.txt:c1|"},
		{name: "rename-directory", change: func(t *testing.T) {
			if err := os.Rename(filepath.Join(tree, "nested"), filepath.Join(tree, "moved")); err != nil {
				t.Fatal(err)
			}
		}, want: "tree/|tree/moved/|tree/moved/b.txt:b2|tree/moved/deep/|tree/moved/deep/d.txt:d1|tree/renamed.txt:c1|"},
		{name: "delete-directory", change: func(t *testing.T) {
			if err := os.RemoveAll(filepath.Join(tree, "moved")); err != nil {
				t.Fatal(err)
			}
		}, want: "tree/|tree/renamed.txt:c1|"},
		{name: "add-empty-directory", change: func(t *testing.T) {
			if err := os.Mkdir(filepath.Join(tree, "empty"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, want: "tree/|tree/renamed.txt:c1|", wantHit: true},
	}
	for _, step := range steps {
		if !t.Run(step.name, func(t *testing.T) {
			if step.change != nil {
				step.change(t)
			}
			reference := filepath.Join(t.TempDir(), "official")
			if runtime.GOOS == "windows" {
				reference += ".exe"
			}
			cmd := exec.Command("go", "build", "-o", reference, ".")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Go build: %v\n%s", err, out)
			}
			assertBuiltProgram(t, reference, step.want)
			assertCachedEmbedProgram(t, root, step.want, step.wantHit)
		}) {
			return
		}
	}
	t.Run("delete-last-file", func(t *testing.T) {
		if err := os.Remove(filepath.Join(tree, "renamed.txt")); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "official"), ".")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "no embeddable files") {
			t.Fatalf("Go build with empty embed directory: %v\n%s", err, out)
		}
		conf := multiBuildConfig()
		conf.OutFile = filepath.Join(t.TempDir(), "embedcache")
		if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: root}); err == nil {
			t.Fatal("llgo reused cached data after all embedded files were deleted")
		}
	})
}

func isolateEmbedBuildCache(t *testing.T) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	t.Setenv(llgoBuildCache, "1")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
}

func assertCachedEmbedProgram(t *testing.T, root, want string, wantHit bool) {
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
