package env

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	LLGoCompilerPkg    = "github.com/goplus/llgo"
	LLGoRuntimePkgName = "runtime"
	LLGoRuntimePkg     = LLGoCompilerPkg + "/" + LLGoRuntimePkgName
	envFileName        = "/internal/env/env.go"
)

func GOROOT() (string, error) {
	return GOROOTWithEnv(nil)
}

func GOROOTWithEnv(env []string) (string, error) {
	vals, err := GoEnvWithEnv(env, "GOROOT")
	if err != nil {
		return "", err
	}
	return vals[0], nil
}

func GOVERSIONWithEnv(env []string) (string, error) {
	vals, err := GoEnvWithEnv(env, "GOVERSION")
	if err != nil {
		return "", err
	}
	return vals[0], nil
}

func GOROOTAndGOVERSIONWithEnv(env []string) (goroot, goversion string, err error) {
	vals, err := GoEnvWithEnv(env, "GOROOT", "GOVERSION")
	if err != nil {
		return "", "", err
	}
	return vals[0], vals[1], nil
}

var (
	goEnvCache     sync.Map
	llgoRootWarned sync.Map
)

func GoEnvWithEnv(env []string, vars ...string) ([]string, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("go env requires at least one variable")
	}
	cacheKey := goEnvCacheKey(env, vars)
	if cached, ok := goEnvCache.Load(cacheKey); ok {
		vals := cached.([]string)
		return append([]string(nil), vals...), nil
	}
	args := append([]string{"env"}, vars...)
	cmd := exec.Command("go", args...)
	if len(env) != 0 {
		cmd.Env = env
	}
	var out bytes.Buffer
	var buf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%s, %w", buf.String(), err)
	}
	got := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(got) != len(vars) {
		return nil, fmt.Errorf("go env returned %d values for %d variables", len(got), len(vars))
	}
	for i := range got {
		got[i] = strings.TrimSpace(got[i])
	}
	goEnvCache.Store(cacheKey, append([]string(nil), got...))
	return got, nil
}

func goEnvCacheKey(env []string, vars []string) string {
	var b strings.Builder
	b.WriteString(os.Getenv("PATH"))
	b.WriteByte('\x00')
	for _, v := range vars {
		b.WriteString(v)
		b.WriteByte('\x00')
	}
	b.WriteByte('\x00')
	if len(env) == 0 {
		env = os.Environ()
	}
	for _, entry := range env {
		b.WriteString(entry)
		b.WriteByte('\x00')
	}
	return b.String()
}

func LLGoCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(userCacheDir, "llgo")
}

func LLGoRuntimeDir() string {
	root := LLGoROOT()
	if root != "" {
		return filepath.Join(root, LLGoRuntimePkgName)
	}
	return ""
}

func LLGoROOT() string {
	llgoRootEnv := os.Getenv("LLGO_ROOT")
	if llgoRootEnv != "" {
		if root, ok := isLLGoRoot(llgoRootEnv); ok {
			return root
		}
		fmt.Fprintf(os.Stderr, "WARNING: LLGO_ROOT is not a valid LLGO root: %s\n", llgoRootEnv)
	}
	// Get executable path
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve any symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	// Check if parent directory is bin
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" {
		// Get parent directory of bin
		root := filepath.Dir(dir)
		if root, ok := isLLGoRoot(root); ok {
			return root
		}
	}
	if Devel() {
		root, err := getRuntimePkgDirByCaller()
		if err != nil {
			return ""
		}
		if root, ok := isLLGoRoot(root); ok {
			if _, loaded := llgoRootWarned.LoadOrStore(root, true); !loaded {
				fmt.Fprintln(os.Stderr, "WARNING: Using LLGO root for devel: "+root)
			}
			return root
		}
	}
	return ""
}

func isLLGoRoot(root string) (string, bool) {
	if root == "" {
		return "", false
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	// Check for go.mod
	data, err := os.ReadFile(filepath.Join(root, LLGoRuntimePkgName, "go.mod"))
	if err != nil {
		return "", false
	}
	// Check module name
	if !strings.Contains(string(data), "module "+LLGoRuntimePkg+"\n") {
		return "", false
	}
	return root, true
}

func getRuntimePkgDirByCaller() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot get caller")
	}
	if !strings.HasSuffix(file, envFileName) {
		return "", fmt.Errorf("wrong caller")
	}
	// check file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return "", fmt.Errorf("file %s not exists", file)
	}
	modPath := strings.TrimSuffix(file, envFileName)
	if st, err := os.Stat(modPath); os.IsNotExist(err) || !st.IsDir() {
		return "", fmt.Errorf("not llgo compiler root: %s", modPath)
	}
	return modPath, nil
}
