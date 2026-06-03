//go:build !llgo
// +build !llgo

package build

import (
	"testing"
)

func TestModuleHookReceivesMainPackageModule(t *testing.T) {
	conf := NewDefaultConf(ModeGen)

	counts := make(map[string]int)
	snapshots := make(map[string]string)
	conf.ModuleHook = func(pkg Package) {
		counts[pkg.PkgPath]++
		if _, ok := snapshots[pkg.PkgPath]; !ok {
			snapshots[pkg.PkgPath] = pkg.LPkg.String()
		}
	}

	pkgs, err := Do([]string{"../../cl/_testgo/print"}, conf)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 initial package, got %d", len(pkgs))
	}

	mainPkg := pkgs[0].PkgPath
	if counts[mainPkg] != 1 {
		t.Fatalf("expected hook to fire once for %s, got %d", mainPkg, counts[mainPkg])
	}
	if snapshots[mainPkg] == "" {
		t.Fatalf("expected non-empty module snapshot for %s", mainPkg)
	}
}
