package build

import (
	"errors"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/quoted"
	gopackages "golang.org/x/tools/go/packages"
)

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
