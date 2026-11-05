package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestCrossPackageInlineWithCachedDependency(t *testing.T) {
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
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/crossinline\n\ngo 1.24\n",
		"dep/arith.go": `package dep
var Counter int
func Next() uint64 {Counter++;return uint64(Counter)}
func Sum(x,y uint64)uint64{return x+y+2}
func Split(x uint64)(uint64,bool){return x+1,x>3}
//go:noinline
func Kept(x uint64)uint64{return x+1}
`,
		"main.go": `package main
import "example.com/crossinline/dep"
import "example.com/crossinline/bridge"
const offset=2
var value=bridge.Sum
//go:noinline
func indirect(fn func(uint64,uint64)uint64,x,y uint64)uint64{return fn(x,y)}
func main(){
 if bridge.Sum(dep.Next(),dep.Next())!=3+offset || dep.Counter!=2 {panic("argument evaluation")}
 for x:=uint64(0);x<100;x++ {
  if bridge.Sum(x,3)!=x+3+offset || indirect(value,x,3)!=x+3+offset {panic("function value")}
  n,b:=dep.Split(x);if n!=x+1 || b!=(x>3){panic("tuple")}
  if dep.Kept(x)!=x+1 {panic("noinline")}
 }
 println("cross inline ok")
}
`,
		"bridge/sum.go": `package bridge
import "example.com/crossinline/dep"
func Sum(x,y uint64)uint64{return dep.Sum(x,y)}
`,
	}
	for name, source := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for phase := 0; phase < 3; phase++ {
		if phase == 2 {
			// Change only the dependency's private implementation. The bridge
			// source is unchanged, but its inlined body must be rebuilt too.
			for name, source := range map[string]string{
				"dep/arith.go": strings.Replace(files["dep/arith.go"], "x+y+2", "x+y+1002", 1),
				"main.go":      strings.Replace(files["main.go"], "offset=2", "offset=1002", 1),
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		conf := NewDefaultConf(ModeBuild)
		conf.OptLevel, conf.LTO = optlevel.O2, lto.Off
		conf.BuildParallelism = 2
		conf.OutFile = filepath.Join(dir, fmt.Sprintf("program-%d", phase))
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		imported := false
		conf.ModuleHook = func(pkg Package) {
			if pkg.PkgPath != "example.com/crossinline" || pkg.LPkg == nil {
				return
			}
			fn := pkg.LPkg.Module().NamedFunction("example.com/crossinline/dep.Sum")
			imported = !fn.IsNil() && !fn.IsDeclaration() && fn.Linkage() == llvm.AvailableExternallyLinkage
		}
		pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		if !imported {
			t.Fatalf("phase %d: caller lacks dependency inline body", phase)
		}
		found := 0
		for _, pkg := range pkgs {
			if pkg.PkgPath == "example.com/crossinline/dep" || pkg.PkgPath == "example.com/crossinline/bridge" {
				found++
				if pkg.CacheHit != (phase == 1) {
					t.Fatalf("phase %d: dependency CacheHit=%v", phase, pkg.CacheHit)
				}
			}
		}
		if found != 2 {
			t.Fatal("dependency or importing bridge missing")
		}
		out, err := exec.Command(conf.OutFile).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "cross inline ok" {
			t.Fatalf("run: %v\n%s", err, out)
		}
	}
}
