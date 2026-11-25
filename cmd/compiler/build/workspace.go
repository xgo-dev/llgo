package build

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/quoted"
	"golang.org/x/mod/modfile"
	gopackages "golang.org/x/tools/go/packages"
)

// isolateNestedModuleWorkspace disables implicit parent go.work discovery when
// building inside a standalone nested Go module that is not listed in that
// workspace's use directives. Explicit GOWORK settings are always respected.
func isolateNestedModuleWorkspace(dir string, environ []string) []string {
	if (commandEnv{environ: environ}).lookup("GOWORK") != "" {
		return environ
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return environ
	}
	var modDir, workFile string
	for cur := absDir; ; cur = filepath.Dir(cur) {
		if modDir == "" {
			if info, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil && !info.IsDir() {
				modDir = cur
			}
		}
		if workFile == "" {
			if info, err := os.Stat(filepath.Join(cur, "go.work")); err == nil && !info.IsDir() {
				workFile = filepath.Join(cur, "go.work")
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur || (modDir != "" && workFile != "") {
			break
		}
	}
	if modDir == "" || workFile == "" {
		return environ
	}
	workDir := filepath.Dir(workFile)
	rel, err := filepath.Rel(workDir, modDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return environ
	}
	data, err := os.ReadFile(workFile)
	if err != nil {
		return environ
	}
	wf, err := modfile.ParseWork(workFile, data, nil)
	if err != nil {
		return environ
	}
	canonDir := func(p string) string {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Clean(resolved)
		}
		return filepath.Clean(p)
	}
	wantMod := canonDir(modDir)
	for _, u := range wf.Use {
		if u == nil || u.Path == "" {
			continue
		}
		useDir := u.Path
		if !filepath.IsAbs(useDir) {
			useDir = filepath.Join(workDir, filepath.FromSlash(useDir))
		}
		if canonDir(useDir) == wantMod {
			return environ
		}
	}
	return withEnv(environ, "GOWORK=off")
}

// go/packages queries release tags with GO111MODULE=off and no build flags.
// A module flag in GOFLAGS breaks that query. Move effective GOFLAGS (including
// persisted GOENV values) into explicit driver flags, before caller flags so
// their precedence is unchanged. A whitespace value suppresses GOENV fallback.
func applyPackageLoadFlags(conf *packages.Config, goflags string) error {
	flags, err := quoted.Split(goflags)
	if err != nil {
		return err
	}
	conf.BuildFlags = append(flags, conf.BuildFlags...)
	conf.Env = withEnv(conf.Env, "GOFLAGS= ")
	return nil
}

// Runtime implementation packages belong to the compiler's module. An
// explicit GOWORK applies even after changing directories, and GOFLAGS can
// select a caller modfile or vendor tree. Override those choices through
// command-line flags, which also take precedence over persistent GOENV flags.
// Only module selection is isolated: retain build tags, source overlays, and
// export package directories so both loads use the same compilation inputs.
func runtimePackageConfig(caller *packages.Config, dir string) packages.Config {
	conf := *caller
	conf.Dir = dir
	conf.Env = withEnv(caller.Env, "GOWORK=off", "GO111MODULE=on")
	conf.BuildFlags = append(slices.Clone(caller.BuildFlags), "-mod=readonly", "-modfile=")
	return conf
}

// List errors are attached to package roots even when Load returns nil error.
// Check before test filtering can discard a failing root. Go frontend export
// diagnostics beginning with '# ' are intentionally deferred: LLGo parses and
// checks patched sources itself and may support sources that gc cannot build.
func initialPackageListErrors(roots []*packages.Package) error {
	var failures []error
	for _, pkg := range roots {
		for _, err := range pkg.Errors {
			if err.Kind == gopackages.ListError && !strings.HasPrefix(err.Msg, "# ") {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
