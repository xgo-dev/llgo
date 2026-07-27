package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goplus/llgo/internal/processenv"
)

// BuildRequest contains the process-derived inputs used by one build. Empty
// Dir and a nil Env preserve the command-line behavior by snapshotting the
// current working directory and environment once, before the build starts.
type BuildRequest struct {
	Args   []string
	Config *Config
	Dir    string
	Env    []string
}

type processSnapshot struct {
	Dir string
	Env []string
}

func (p processSnapshot) command(name string, args ...string) *exec.Cmd {
	return processenv.Command(p.Env, p.Dir, name, args...)
}

func (p processSnapshot) path(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(p.Dir, path)
}

func (p processSnapshot) resolveOutputs(out *OutFmtDetails) {
	out.Out = p.path(out.Out)
	out.PCLN = p.path(out.PCLN)
	out.Bin = p.path(out.Bin)
	out.Hex = p.path(out.Hex)
	out.Img = p.path(out.Img)
	out.Uf2 = p.path(out.Uf2)
	out.Zip = p.path(out.Zip)
}

func snapshotProcess(req BuildRequest) (processSnapshot, error) {
	dir := req.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return processSnapshot{}, fmt.Errorf("get working directory: %w", err)
		}
	}
	env := slices.Clone(req.Env)
	if req.Env == nil {
		env = os.Environ()
	}
	return processSnapshot{Dir: dir, Env: env}, nil
}

func envValue(environ []string, key string) (string, bool) {
	return processenv.Lookup(environ, key)
}

func withEnv(environ []string, values ...string) []string {
	keys := make(map[string]struct{}, len(values))
	for _, value := range values {
		if key, _, ok := strings.Cut(value, "="); ok {
			keys[key] = struct{}{}
		}
	}
	ret := make([]string, 0, len(environ)+len(values))
	for _, value := range environ {
		key, _, ok := strings.Cut(value, "=")
		// Ignore malformed entries: exec.Cmd requires KEY=VALUE strings.
		if _, replace := keys[key]; ok && replace {
			continue
		}
		if ok {
			ret = append(ret, value)
		}
	}
	return append(ret, values...)
}
