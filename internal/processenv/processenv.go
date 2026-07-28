package processenv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Context contains the process-derived inputs used to resolve paths and launch
// subprocesses for one request.
type Context struct {
	Dir string
	Env []string
}

// Clone returns an independent copy of the context.
func (c Context) Clone() Context {
	c.Env = slices.Clone(c.Env)
	return c
}

// Capture resolves omitted process inputs once. A nil environ snapshots the
// current process environment; an empty non-nil environ remains empty.
func Capture(dir string, environ []string) (Context, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return Context{}, fmt.Errorf("get working directory: %w", err)
		}
	}
	env := slices.Clone(environ)
	if environ == nil {
		env = os.Environ()
	}
	return Context{Dir: dir, Env: env}, nil
}

// Abs resolves path relative to the context working directory.
func (c Context) Abs(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir, path)
}

// Get returns the last value for key in the context environment.
func (c Context) Get(key string) string {
	if c.Env == nil {
		return os.Getenv(key)
	}
	return Get(c.Env, key)
}

// Lookup returns the last value for key in the context environment and whether
// it was present.
func (c Context) Lookup(key string) (string, bool) {
	if c.Env == nil {
		return os.LookupEnv(key)
	}
	return Lookup(c.Env, key)
}

// Get returns the last value for key, matching the convention used when an
// exec.Cmd environment contains duplicate entries.
func Get(environ []string, key string) string {
	value, _ := Lookup(environ, key)
	return value
}

// Lookup returns the last value for key and whether it was present.
func Lookup(environ []string, key string) (string, bool) {
	prefix := key + "="
	for i := len(environ) - 1; i >= 0; i-- {
		if strings.HasPrefix(environ[i], prefix) {
			return strings.TrimPrefix(environ[i], prefix), true
		}
	}
	return "", false
}

// Command constructs a command whose path lookup, environment and working
// directory all come from the same snapshot. A nil environ inherits the
// process environment while still applying dir.
func Command(environ []string, dir, name string, args ...string) *exec.Cmd {
	return (Context{Dir: dir, Env: environ}).Command(name, args...)
}

// Command constructs a command using the context environment and directory.
func (c Context) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = c.Dir
	if c.Env == nil {
		return cmd
	}
	cmd.Env = slices.Clone(c.Env)
	path, err := c.LookPath(name)
	cmd.Path = path
	cmd.Err = err
	return cmd
}

// LookPath searches file using PATH from environ. A match through a relative
// PATH entry is resolved against dir but returned with exec.ErrDot, matching
// the standard library safeguard against executing from a working directory.
func LookPath(environ []string, dir, file string) (string, error) {
	return (Context{Dir: dir, Env: environ}).LookPath(file)
}

// LookPath searches file using PATH from the context environment.
func (c Context) LookPath(file string) (string, error) {
	if c.Env == nil {
		return exec.LookPath(file)
	}
	if strings.ContainsRune(file, os.PathSeparator) {
		path := file
		if !filepath.IsAbs(path) && c.Dir != "" {
			path = filepath.Join(c.Dir, path)
		}
		if executable(path) {
			return path, nil
		}
		return "", exec.ErrNotFound
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" && filepath.Ext(file) == "" {
		if pathExt := c.Get("PATHEXT"); pathExt != "" {
			extensions = filepath.SplitList(strings.ToLower(pathExt))
		}
	}
	for _, pathDir := range filepath.SplitList(c.Get("PATH")) {
		if pathDir == "" {
			pathDir = "."
		}
		relative := !filepath.IsAbs(pathDir)
		if relative && c.Dir != "" {
			pathDir = filepath.Join(c.Dir, pathDir)
		}
		for _, extension := range extensions {
			candidate := filepath.Join(pathDir, file+extension)
			if executable(candidate) {
				if relative {
					return candidate, exec.ErrDot
				}
				return candidate, nil
			}
		}
	}
	return "", exec.ErrNotFound
}
