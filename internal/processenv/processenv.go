package processenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Get returns the last value for key, matching the convention used when an
// exec.Cmd environment contains duplicate entries.
func Get(environ []string, key string) string {
	prefix := key + "="
	for i := len(environ) - 1; i >= 0; i-- {
		if strings.HasPrefix(environ[i], prefix) {
			return strings.TrimPrefix(environ[i], prefix)
		}
	}
	return ""
}

// Command constructs a command whose path lookup, environment and working
// directory all come from the same snapshot. A nil environ preserves the
// standard os/exec process-inheritance behavior.
func Command(environ []string, dir, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if environ == nil {
		return cmd
	}
	cmd.Env = slices.Clone(environ)
	path, err := LookPath(environ, dir, name)
	cmd.Path = path
	cmd.Err = err
	return cmd
}

// LookPath searches file using PATH from environ. Relative PATH entries are
// interpreted from dir, which is where the resulting command will run.
func LookPath(environ []string, dir, file string) (string, error) {
	if environ == nil {
		return exec.LookPath(file)
	}
	if strings.ContainsRune(file, os.PathSeparator) {
		path := file
		if !filepath.IsAbs(path) && dir != "" {
			path = filepath.Join(dir, path)
		}
		if executable(path) {
			return path, nil
		}
		return "", exec.ErrNotFound
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" && filepath.Ext(file) == "" {
		if pathExt := Get(environ, "PATHEXT"); pathExt != "" {
			extensions = filepath.SplitList(strings.ToLower(pathExt))
		}
	}
	for _, pathDir := range filepath.SplitList(Get(environ, "PATH")) {
		if pathDir == "" {
			pathDir = "."
		}
		if !filepath.IsAbs(pathDir) && dir != "" {
			pathDir = filepath.Join(dir, pathDir)
		}
		for _, extension := range extensions {
			candidate := filepath.Join(pathDir, file+extension)
			if executable(candidate) {
				return candidate, nil
			}
		}
	}
	return "", exec.ErrNotFound
}

func executable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return runtime.GOOS == "windows" || info.Mode()&0o111 != 0
}
