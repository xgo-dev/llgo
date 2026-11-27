package build

import (
	"go/version"
	"strings"
)

// Go before 1.27 adds the MinGW-only -mthreads flag while preparing Windows
// cgo metadata. Keep the selected MSVC target and C type layout, but ask Clang
// to discard that obsolete driver argument for this metadata subprocess only.
func cgoMetadataEnvironment(environ []string, goversion, triple string) []string {
	commands := commandEnv{environ: environ}
	if commands.lookup("CGO_ENABLED") != "1" || !version.IsValid(goversion) ||
		version.Compare(goversion, "go1.27") >= 0 || !strings.Contains(triple, "-windows-msvc") {
		return environ
	}
	overrides := commands.lookup("CCC_OVERRIDE_OPTIONS")
	if overrides == "" {
		// Clang's leading # suppresses the override diagnostic, which would
		// otherwise pollute cgo's compiler output.
		overrides = "#"
	}
	return withEnv(environ, "CCC_OVERRIDE_OPTIONS="+overrides+" x-mthreads")
}
