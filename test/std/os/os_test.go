package os_test

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const osHelperEnv = "LLGO_TEST_OS_HELPER"

func TestOSHelperProcess(t *testing.T) {
	mode := os.Getenv(osHelperEnv)
	if mode == "" {
		return
	}
	switch mode {
	case "clearenv":
		if err := os.Setenv("LLGO_CLEAR_ENV_PROBE", "set"); err != nil {
			os.Exit(2)
		}
		os.Clearenv()
		if os.Getenv("LLGO_CLEAR_ENV_PROBE") != "" || len(os.Environ()) != 0 {
			os.Exit(2)
		}
	case "success":
	case "sleep":
		time.Sleep(time.Minute)
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func osHelperCommand(mode string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestOSHelperProcess$")
	cmd.Env = append(os.Environ(), osHelperEnv+"="+mode)
	return cmd
}

func startOSHelper(t *testing.T, mode string) *os.Process {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	proc, err := os.StartProcess(executable, []string{executable, "-test.run=^TestOSHelperProcess$"}, &os.ProcAttr{
		Env:   append(os.Environ(), osHelperEnv+"="+mode),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		t.Fatalf("StartProcess helper: %v", err)
	}
	return proc
}

func canonicalPath(p string) string {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return resolved
}

func TestChdir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Errorf("Chdir(%q) failed: %v", tmpDir, err)
	}

	newDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if canonicalPath(newDir) != canonicalPath(tmpDir) {
		t.Errorf("After Chdir, Getwd() = %q, want %q", newDir, tmpDir)
	}
}

func TestChmod(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chmod_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(testFile, 0600); err != nil {
		t.Errorf("Chmod failed: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("After Chmod(0600), mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestChtimes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chtimes_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	atime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mtime := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	if err := os.Chtimes(testFile, atime, mtime); err != nil {
		t.Errorf("Chtimes failed: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Errorf("After Chtimes, ModTime = %v, want %v", info.ModTime(), mtime)
	}
}

func TestClearenv(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		// Wasm cannot spawn a child process. Run the same assertions in the
		// guest and restore the environment before other tests execute.
		saved := os.Environ()
		defer func() {
			os.Clearenv()
			for _, entry := range saved {
				key, value, _ := strings.Cut(entry, "=")
				if err := os.Setenv(key, value); err != nil {
					t.Errorf("restore environment key %s: %v", key, err)
				}
			}
		}()
		if err := os.Setenv("LLGO_CLEAR_ENV_PROBE", "set"); err != nil {
			t.Fatal(err)
		}
		os.Clearenv()
		if os.Getenv("LLGO_CLEAR_ENV_PROBE") != "" || len(os.Environ()) != 0 {
			t.Fatal("Clearenv did not empty the environment")
		}
		return
	}
	// Clearenv mutates process-global state. Exercise it in a child so this test
	// cannot erase PATH and platform runtime variables needed by later tests.
	if output, err := osHelperCommand("clearenv").CombinedOutput(); err != nil {
		t.Fatalf("Clearenv helper: %v\n%s", err, output)
	}
}

func TestEnviron(t *testing.T) {
	os.Setenv("TEST_ENVIRON", "test_value")
	defer os.Unsetenv("TEST_ENVIRON")

	env := os.Environ()
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "TEST_ENVIRON=") {
			found = true
			if e != "TEST_ENVIRON=test_value" {
				t.Errorf("Environ contains %q, want TEST_ENVIRON=test_value", e)
			}
			break
		}
	}
	if !found {
		t.Error("TEST_ENVIRON not found in Environ()")
	}
}

func TestExecutable(t *testing.T) {
	exe, err := os.Executable()
	if runtime.GOARCH == "wasm" {
		if exe != "" || err == nil || err.Error() != "Executable not implemented for "+runtime.GOOS {
			t.Fatalf("Executable = %q, %v; want the Go wasm unsupported-operation contract", exe, err)
		}
		return
	}
	if err != nil {
		t.Errorf("Executable() failed: %v", err)
	}
	if exe == "" {
		t.Error("Executable() returned empty string")
	}
}

func TestExpand(t *testing.T) {
	mapper := func(s string) string {
		if s == "VAR" {
			return "value"
		}
		return ""
	}

	tests := []struct {
		input string
		want  string
	}{
		{"$VAR", "value"},
		{"${VAR}", "value"},
		{"prefix ${VAR} suffix", "prefix value suffix"},
		{"$UNKNOWN", ""},
	}

	for _, tt := range tests {
		got := os.Expand(tt.input, mapper)
		if got != tt.want {
			t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := os.ExpandEnv("prefix ${TEST_VAR} suffix")
	want := "prefix test_value suffix"
	if result != want {
		t.Errorf("ExpandEnv = %q, want %q", result, want)
	}
}

func TestGetenv(t *testing.T) {
	os.Setenv("TEST_GETENV", "test_value")
	defer os.Unsetenv("TEST_GETENV")

	if val := os.Getenv("TEST_GETENV"); val != "test_value" {
		t.Errorf("Getenv(TEST_GETENV) = %q, want test_value", val)
	}
	if val := os.Getenv("NONEXISTENT"); val != "" {
		t.Errorf("Getenv(NONEXISTENT) = %q, want empty", val)
	}
}

func TestLookupEnv(t *testing.T) {
	os.Setenv("TEST_LOOKUP", "test_value")
	defer os.Unsetenv("TEST_LOOKUP")

	val, ok := os.LookupEnv("TEST_LOOKUP")
	if !ok || val != "test_value" {
		t.Errorf("LookupEnv(TEST_LOOKUP) = (%q, %v), want (test_value, true)", val, ok)
	}

	_, ok = os.LookupEnv("NONEXISTENT")
	if ok {
		t.Error("LookupEnv(NONEXISTENT) returned ok=true, want false")
	}
}

func TestSetenvUnsetenv(t *testing.T) {
	if err := os.Setenv("TEST_SETENV", "value"); err != nil {
		t.Errorf("Setenv failed: %v", err)
	}
	if val := os.Getenv("TEST_SETENV"); val != "value" {
		t.Errorf("After Setenv, Getenv = %q, want value", val)
	}

	if err := os.Unsetenv("TEST_SETENV"); err != nil {
		t.Errorf("Unsetenv failed: %v", err)
	}
	if val := os.Getenv("TEST_SETENV"); val != "" {
		t.Errorf("After Unsetenv, Getenv = %q, want empty", val)
	}
}

func TestGetpid(t *testing.T) {
	pid := os.Getpid()
	if pid <= 0 {
		t.Errorf("Getpid() = %d, want > 0", pid)
	}
}

func TestGetppid(t *testing.T) {
	ppid := os.Getppid()
	if ppid <= 0 {
		t.Errorf("Getppid() = %d, want > 0", ppid)
	}
}

func TestGetuid(t *testing.T) {
	uid := os.Getuid()
	if runtime.GOOS == "windows" && uid != -1 {
		t.Errorf("Getuid() = %d, want -1 on Windows", uid)
	} else if runtime.GOOS != "windows" && uid < 0 {
		t.Errorf("Getuid() = %d, want >= 0", uid)
	}
}

func TestGeteuid(t *testing.T) {
	euid := os.Geteuid()
	if runtime.GOOS == "windows" && euid != -1 {
		t.Errorf("Geteuid() = %d, want -1 on Windows", euid)
	} else if runtime.GOOS != "windows" && euid < 0 {
		t.Errorf("Geteuid() = %d, want >= 0", euid)
	}
}

func TestGetgid(t *testing.T) {
	gid := os.Getgid()
	if runtime.GOOS == "windows" && gid != -1 {
		t.Errorf("Getgid() = %d, want -1 on Windows", gid)
	} else if runtime.GOOS != "windows" && gid < 0 {
		t.Errorf("Getgid() = %d, want >= 0", gid)
	}
}

func TestGetegid(t *testing.T) {
	egid := os.Getegid()
	if runtime.GOOS == "windows" && egid != -1 {
		t.Errorf("Getegid() = %d, want -1 on Windows", egid)
	} else if runtime.GOOS != "windows" && egid < 0 {
		t.Errorf("Getegid() = %d, want >= 0", egid)
	}
}

func TestGetgroups(t *testing.T) {
	groups, err := os.Getgroups()
	if runtime.GOOS == "windows" {
		if err == nil || len(groups) != 0 {
			t.Errorf("Getgroups() = %v, %v; want empty groups and unsupported error", groups, err)
		}
		return
	}
	if err != nil {
		t.Errorf("Getgroups() failed: %v", err)
	}
	if len(groups) == 0 {
		t.Error("Getgroups() returned empty slice")
	}
}

func TestGetpagesize(t *testing.T) {
	pagesize := os.Getpagesize()
	if pagesize <= 0 {
		t.Errorf("Getpagesize() = %d, want > 0", pagesize)
	}
}

func TestGetwd(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("Getwd() failed: %v", err)
	}
	if wd == "" {
		t.Error("Getwd() returned empty string")
	}
}

func TestHostname(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Errorf("Hostname() failed: %v", err)
	}
	if hostname == "" {
		t.Error("Hostname() returned empty string")
	}
}

func TestTempDir(t *testing.T) {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		t.Error("TempDir() returned empty string")
	}
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Errorf("TempDir() returned non-existent directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("TempDir() returned non-directory: %q", tmpDir)
	}
}

func TestUserCacheDir(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() failed: %v", err)
	}
	if dir == "" {
		t.Fatal("UserCacheDir() returned empty string")
	}
}

func TestUserConfigDir(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() failed: %v", err)
	}
	if dir == "" {
		t.Fatal("UserConfigDir() returned empty string")
	}
}

func TestUserHomeDir(t *testing.T) {
	dir, err := os.UserHomeDir()
	if runtime.GOARCH == "wasm" && os.Getenv("HOME") == "" {
		if dir != "" || err == nil || err.Error() != "$HOME is not defined" {
			t.Fatalf("UserHomeDir without HOME = %q, %v", dir, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("UserHomeDir() failed: %v", err)
	}
	if dir == "" {
		t.Fatal("UserHomeDir() returned empty string")
	}
}

func TestIsPathSeparator(t *testing.T) {
	if runtime.GOOS == "windows" {
		if !os.IsPathSeparator('\\') {
			t.Error("IsPathSeparator('\\\\') should be true on Windows")
		}
	}
	if !os.IsPathSeparator('/') {
		t.Error("IsPathSeparator('/') should be true")
	}
	if os.IsPathSeparator('a') {
		t.Error("IsPathSeparator('a') should be false")
	}
}

func TestMkdir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	if err := os.Mkdir(newDir, 0755); err != nil {
		t.Errorf("Mkdir failed: %v", err)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("Stat after Mkdir failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("Mkdir did not create a directory")
	}
}

func TestMkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c")

	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}

	info, err := os.Stat(deepDir)
	if err != nil {
		t.Errorf("Stat after MkdirAll failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("MkdirAll did not create directories")
	}
}

func TestMkdirTemp(t *testing.T) {
	tmpDir := t.TempDir()
	tempDir, err := os.MkdirTemp(tmpDir, "pattern*")
	if err != nil {
		t.Errorf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	info, err := os.Stat(tempDir)
	if err != nil {
		t.Errorf("Stat after MkdirTemp failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("MkdirTemp did not create a directory")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "remove_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(testFile); err != nil {
		t.Errorf("Remove failed: %v", err)
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File still exists after Remove")
	}
}

func TestRemoveAll(t *testing.T) {
	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c")
	os.MkdirAll(deepDir, 0755)
	testFile := filepath.Join(deepDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	targetDir := filepath.Join(tmpDir, "a")
	if err := os.RemoveAll(targetDir); err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Error("Directory still exists after RemoveAll")
	}
}

func TestRename(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		t.Errorf("Rename failed: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file still exists after Rename")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Error("New file does not exist after Rename")
	}
}

func TestTruncate(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "truncate_test.txt")
	if err := os.WriteFile(testFile, []byte("long content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Truncate(testFile, 4); err != nil {
		t.Errorf("Truncate failed: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 4 {
		t.Errorf("After Truncate(4), size = %d, want 4", info.Size())
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "read_test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("ReadFile = %q, want %q", data, content)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_test.txt")
	content := []byte("write content")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Errorf("After WriteFile, content = %q, want %q", data, content)
	}
}

func TestReadDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Errorf("ReadDir failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadDir returned %d entries, want 3", len(entries))
	}

	for _, entry := range entries {
		if entry.Name() == "" {
			t.Error("Entry has empty name")
		}
	}
}

func TestStat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "stat_test.txt")
	content := []byte("test")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if info.Name() != "stat_test.txt" {
		t.Errorf("Stat().Name() = %q, want stat_test.txt", info.Name())
	}
	if info.Size() != int64(len(content)) {
		t.Errorf("Stat().Size() = %d, want %d", info.Size(), len(content))
	}
	if info.IsDir() {
		t.Error("Stat().IsDir() = true for file")
	}
}

func TestLstat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lstat_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(testFile)
	if err != nil {
		t.Errorf("Lstat failed: %v", err)
	}
	if info.Name() != "lstat_test.txt" {
		t.Errorf("Lstat().Name() = %q, want lstat_test.txt", info.Name())
	}
}

func TestSameFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "same_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	info1, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if !os.SameFile(info1, info2) {
		t.Error("SameFile returned false for same file")
	}
}

func TestSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Relative targets remain within a WASI preopen capability; an absolute
	// host path is not necessarily a valid symlink target in a guest.
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	linkTarget, err := os.Readlink(link)
	if err != nil {
		t.Errorf("Readlink failed: %v", err)
	}
	if linkTarget != "target.txt" {
		t.Errorf("Readlink = %q, want target.txt", linkTarget)
	}
	if data, err := os.ReadFile(link); err != nil || string(data) != "content" {
		t.Errorf("ReadFile through symlink = %q, %v", data, err)
	}
}

func TestLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Link(oldPath, newPath); err != nil {
		t.Fatalf("Link failed: %v", err)
	}

	info1, err := os.Stat(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(info1, info2) {
		t.Error("Linked files are not the same")
	}
}

func TestChown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chown_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := os.Chown(testFile, -1, -1)
	if runtime.GOOS == "wasip1" {
		if !errors.Is(err, syscall.ENOSYS) {
			t.Errorf("Chown = %v, want ENOSYS on wasip1", err)
		}
	} else if runtime.GOOS == "windows" {
		var pathError *os.PathError
		if err == nil || !errors.As(err, &pathError) {
			t.Errorf("Chown = %v, want a PathError wrapping unsupported Windows operation", err)
		}
	} else if err != nil {
		t.Errorf("Chown(-1, -1) failed: %v", err)
	}
}

func TestLchown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lchown_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := os.Lchown(testFile, -1, -1)
	if runtime.GOOS == "wasip1" {
		if !errors.Is(err, syscall.ENOSYS) {
			t.Errorf("Lchown = %v, want ENOSYS on wasip1", err)
		}
	} else if runtime.GOOS == "windows" {
		var pathError *os.PathError
		if err == nil || !errors.As(err, &pathError) {
			t.Errorf("Lchown = %v, want a PathError wrapping unsupported Windows operation", err)
		}
	} else if err != nil {
		t.Errorf("Lchown(-1, -1) failed: %v", err)
	}
}

func TestDirFS(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := "test.txt"
	if err := os.WriteFile(filepath.Join(tmpDir, testFile), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	fsys := os.DirFS(tmpDir)
	f, err := fsys.Open(testFile)
	if err != nil {
		t.Errorf("DirFS().Open failed: %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Errorf("DirFS read %q, want content", data)
	}
}

func TestCopyFS(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0644)

	fsys := os.DirFS(srcDir)
	if err := os.CopyFS(dstDir, fsys); err != nil {
		t.Errorf("CopyFS failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "file.txt"))
	if err != nil {
		t.Errorf("CopyFS result read failed: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("CopyFS copied %q, want content", data)
	}
}

func TestPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if runtime.GOARCH == "wasm" {
		if r != nil || w != nil || !errors.Is(err, syscall.ENOSYS) {
			t.Fatalf("Pipe = %v, %v, %v; want nil, nil, ENOSYS on wasm", r, w, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	defer r.Close()
	defer w.Close()

	testData := []byte("test data")
	go func() {
		w.Write(testData)
		w.Close()
	}()

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Errorf("Read from pipe failed: %v", err)
	}
	if string(buf[:n]) != string(testData) {
		t.Errorf("Pipe read %q, want %q", buf[:n], testData)
	}
}

func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file_ops.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if f.Name() != testFile {
		t.Errorf("File.Name() = %q, want %q", f.Name(), testFile)
	}

	n, err := f.Write([]byte("test"))
	if err != nil || n != 4 {
		t.Errorf("Write failed: n=%d, err=%v", n, err)
	}

	n2, err := f.WriteString(" string")
	if err != nil || n2 != 7 {
		t.Errorf("WriteString failed: n=%d, err=%v", n2, err)
	}

	if err := f.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	offset, err := f.Seek(0, io.SeekStart)
	if err != nil || offset != 0 {
		t.Errorf("Seek failed: offset=%d, err=%v", offset, err)
	}

	buf := make([]byte, 11)
	n, err = f.Read(buf)
	if err != nil || string(buf[:n]) != "test string" {
		t.Errorf("Read failed: got %q, err=%v", buf[:n], err)
	}

	info, err := f.Stat()
	if err != nil {
		t.Errorf("File.Stat failed: %v", err)
	}
	if info.Size() != 11 {
		t.Errorf("File.Stat().Size() = %d, want 11", info.Size())
	}

	if err := f.Truncate(4); err != nil {
		t.Errorf("File.Truncate failed: %v", err)
	}

	if fd := f.Fd(); fd == 0 {
		t.Error("File.Fd() returned 0")
	}

	if err := f.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "open_test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(testFile)
	if err != nil {
		t.Errorf("Open failed: %v", err)
	}
	defer f.Close()

	buf := make([]byte, len(content))
	n, err := f.Read(buf)
	if err != nil || string(buf[:n]) != string(content) {
		t.Errorf("Read after Open got %q, want %q", buf[:n], content)
	}
}

func TestOpenFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "openfile_test.txt")

	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Errorf("OpenFile failed: %v", err)
	}
	defer f.Close()

	if _, err := f.WriteString("content"); err != nil {
		t.Errorf("WriteString after OpenFile failed: %v", err)
	}
}

func TestCreateTemp(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "pattern*")
	if err != nil {
		t.Errorf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if !strings.HasPrefix(filepath.Base(f.Name()), "pattern") {
		t.Errorf("CreateTemp name %q doesn't match pattern", f.Name())
	}
}

func TestFileReadAt(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readat_test.txt")
	content := []byte("0123456789")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	buf := make([]byte, 3)
	n, err := f.ReadAt(buf, 5)
	if err != nil || string(buf[:n]) != "567" {
		t.Errorf("ReadAt(5) = %q, err=%v, want 567", buf[:n], err)
	}
}

func TestFileWriteAt(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "writeat_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	f.Write([]byte("0000000000"))
	n, err := f.WriteAt([]byte("abc"), 3)
	if err != nil || n != 3 {
		t.Errorf("WriteAt failed: n=%d, err=%v", n, err)
	}

	f.Seek(0, io.SeekStart)
	buf := make([]byte, 10)
	f.Read(buf)
	if string(buf) != "000abc0000" {
		t.Errorf("After WriteAt, content = %q, want 000abc0000", buf)
	}
}

func TestFileChdir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	f, err := os.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Chdir(); err != nil {
		t.Errorf("File.Chdir failed: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if canonicalPath(wd) != canonicalPath(tmpDir) {
		t.Errorf("After File.Chdir, Getwd = %q, want %q", wd, tmpDir)
	}
}

func TestFileChdirRestoresRelativeDirectory(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	orig, err := os.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer orig.Close()

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := orig.Chdir(); err != nil {
		t.Fatalf("File.Chdir failed to restore a directory opened as .: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if canonicalPath(wd) != canonicalPath(origDir) {
		t.Errorf("After File.Chdir, Getwd = %q, want %q", wd, origDir)
	}
}

func TestFileChmod(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chmod_file_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Chmod(0600); err != nil {
		t.Errorf("File.Chmod failed: %v", err)
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("After File.Chmod, mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestFileReadDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("2"), 0644)

	f, err := os.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entries, err := f.ReadDir(-1)
	if err != nil {
		t.Errorf("File.ReadDir failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("File.ReadDir returned %d entries, want 2", len(entries))
	}
}

func TestFileReaddir(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)

	f, err := os.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	infos, err := f.Readdir(-1)
	if err != nil {
		t.Errorf("File.Readdir failed: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("File.Readdir returned %d infos, want 1", len(infos))
	}
}

func TestFileReaddirnames(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("1"), 0644)

	f, err := os.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Errorf("File.Readdirnames failed: %v", err)
	}
	if len(names) != 1 || names[0] != "file1.txt" {
		t.Errorf("File.Readdirnames = %v, want [file1.txt]", names)
	}
}

func TestErrorFunctions(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "/nonexistent", Err: syscall.ENOENT}
	if !os.IsNotExist(err) {
		t.Error("IsNotExist should return true for ENOENT")
	}

	err2 := &os.PathError{Op: "open", Path: "/exists", Err: syscall.EEXIST}
	if !os.IsExist(err2) {
		t.Error("IsExist should return true for EEXIST")
	}

	err3 := &os.PathError{Op: "open", Path: "/denied", Err: syscall.EACCES}
	if !os.IsPermission(err3) {
		t.Error("IsPermission should return true for EACCES")
	}

	if os.IsNotExist(nil) {
		t.Error("IsNotExist should return false for nil")
	}
}

func TestNewSyscallError(t *testing.T) {
	err := os.NewSyscallError("test_syscall", syscall.EINVAL)
	if err == nil {
		t.Error("NewSyscallError returned nil")
	}
	if !strings.Contains(err.Error(), "test_syscall") {
		t.Errorf("NewSyscallError().Error() = %q, should contain test_syscall", err.Error())
	}
}

func TestSyscallError(t *testing.T) {
	serr := &os.SyscallError{
		Syscall: "open",
		Err:     syscall.EINVAL,
	}

	errStr := serr.Error()
	if !strings.Contains(errStr, "open") {
		t.Errorf("SyscallError.Error() = %q, should contain open", errStr)
	}

	unwrapped := serr.Unwrap()
	if unwrapped != syscall.EINVAL {
		t.Errorf("SyscallError.Unwrap() = %v, want EINVAL", unwrapped)
	}
}

func TestLinkError(t *testing.T) {
	lerr := &os.LinkError{
		Op:  "symlink",
		Old: "/old",
		New: "/new",
		Err: syscall.EEXIST,
	}

	errStr := lerr.Error()
	if !strings.Contains(errStr, "symlink") || !strings.Contains(errStr, "/old") {
		t.Errorf("LinkError.Error() = %q, should contain op and old path", errStr)
	}

	unwrapped := lerr.Unwrap()
	if unwrapped != syscall.EEXIST {
		t.Errorf("LinkError.Unwrap() = %v, want EEXIST", unwrapped)
	}
}

func TestFindProcess(t *testing.T) {
	pid := os.Getpid()
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Errorf("FindProcess(%d) failed: %v", pid, err)
	}
	if proc == nil {
		t.Error("FindProcess returned nil process")
	}
}

func TestConstants(t *testing.T) {
	if os.DevNull == "" {
		t.Error("DevNull constant is empty")
	}

	if os.PathSeparator == 0 {
		t.Error("PathSeparator is zero")
	}

	if os.PathListSeparator == 0 {
		t.Error("PathListSeparator is zero")
	}

	openFlags := []int{os.O_RDONLY, os.O_WRONLY, os.O_APPEND, os.O_EXCL, os.O_SYNC, os.O_TRUNC}
	for _, flag := range openFlags {
		_ = flag
	}

	seekModes := []int{os.SEEK_SET, os.SEEK_CUR, os.SEEK_END}
	for _, mode := range seekModes {
		_ = mode
	}
}

func TestFileMode(t *testing.T) {
	modes := []os.FileMode{
		os.ModeDir,
		os.ModeAppend,
		os.ModeExclusive,
		os.ModeTemporary,
		os.ModeSymlink,
		os.ModeDevice,
		os.ModeNamedPipe,
		os.ModeSocket,
		os.ModeSetuid,
		os.ModeSetgid,
		os.ModeCharDevice,
		os.ModeSticky,
		os.ModeIrregular,
		os.ModeType,
		os.ModePerm,
	}

	for _, mode := range modes {
		_ = mode
	}
}

func TestErrorConstants(t *testing.T) {
	errors := []error{
		os.ErrInvalid,
		os.ErrPermission,
		os.ErrExist,
		os.ErrNotExist,
		os.ErrClosed,
		os.ErrNoDeadline,
		os.ErrDeadlineExceeded,
		os.ErrProcessDone,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error constant is nil")
		}
	}
}

func TestStdFiles(t *testing.T) {
	if os.Stdin == nil {
		t.Error("Stdin is nil")
	}
	if os.Stdout == nil {
		t.Error("Stdout is nil")
	}
	if os.Stderr == nil {
		t.Error("Stderr is nil")
	}

	if os.Stdin.Name() != "/dev/stdin" && os.Stdin.Name() != "stdin" && os.Stdin.Name() != "/dev/fd/0" {
		t.Logf("Stdin.Name() = %q", os.Stdin.Name())
	}
}

func TestArgs(t *testing.T) {
	if os.Args == nil {
		t.Error("Args is nil")
	}
	if len(os.Args) == 0 {
		t.Error("Args is empty")
	}
}

func TestIsTimeout(t *testing.T) {
	if os.IsTimeout(nil) {
		t.Error("IsTimeout should return false for nil")
	}

	serr := &os.SyscallError{
		Syscall: "test",
		Err:     syscall.ETIMEDOUT,
	}
	if !serr.Timeout() {
		t.Error("SyscallError.Timeout() should return true for ETIMEDOUT")
	}
}

func TestFileChown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "chown_file_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = f.Chown(-1, -1)
	if runtime.GOOS == "wasip1" {
		if !errors.Is(err, syscall.ENOSYS) {
			t.Errorf("File.Chown = %v, want ENOSYS on wasip1", err)
		}
	} else if runtime.GOOS == "windows" {
		var pathError *os.PathError
		if err == nil || !errors.As(err, &pathError) {
			t.Errorf("File.Chown = %v, want a PathError wrapping unsupported Windows operation", err)
		}
	} else if err != nil {
		t.Errorf("File.Chown(-1, -1) failed: %v", err)
	}
}

func TestFileReadFrom(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readfrom_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	src := strings.NewReader("test data from reader")
	n, err := f.ReadFrom(src)
	if err != nil {
		t.Errorf("File.ReadFrom failed: %v", err)
	}
	if n != int64(len("test data from reader")) {
		t.Errorf("File.ReadFrom wrote %d bytes, want %d", n, len("test data from reader"))
	}
}

func TestFileWriteTo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "writeto_test.txt")
	content := []byte("test data")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var buf strings.Builder
	n, err := f.WriteTo(&buf)
	if err != nil {
		t.Errorf("File.WriteTo failed: %v", err)
	}
	if n != int64(len(content)) || buf.String() != string(content) {
		t.Errorf("File.WriteTo wrote %d bytes: %q, want %d: %q", n, buf.String(), len(content), content)
	}
}

func TestFileSetDeadline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "deadline_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	deadline := time.Now().Add(time.Second)
	for name, set := range map[string]func(time.Time) error{
		"SetDeadline":      f.SetDeadline,
		"SetReadDeadline":  f.SetReadDeadline,
		"SetWriteDeadline": f.SetWriteDeadline,
	} {
		err := set(deadline)
		if runtime.GOARCH == "wasm" && runtime.GOOS == "js" {
			// Go's js/wasm poller accepts deadlines on regular files.
			if err != nil {
				t.Errorf("File.%s = %v, want nil on js/wasm", name, err)
			}
		} else if !errors.Is(err, os.ErrNoDeadline) {
			t.Errorf("File.%s = %v, want ErrNoDeadline for a regular file", name, err)
		}
	}
}

func TestFileSyscallConn(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sysconn_test.txt")

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	conn, err := f.SyscallConn()
	if err != nil {
		t.Fatalf("File.SyscallConn: %v", err)
	}
	if conn == nil {
		t.Fatal("File.SyscallConn returned nil without error")
	}
	var descriptor uintptr
	if err := conn.Control(func(fd uintptr) { descriptor = fd }); err != nil {
		t.Fatalf("RawConn.Control: %v", err)
	}
	if descriptor != f.Fd() {
		t.Errorf("RawConn descriptor = %d, want File.Fd %d", descriptor, f.Fd())
	}
}

func TestStartProcess(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		proc, err := os.StartProcess("/wasm-child", []string{"/wasm-child"}, &os.ProcAttr{})
		if proc != nil || !errors.Is(err, syscall.ENOSYS) {
			t.Fatalf("StartProcess = %v, %v; want nil, ENOSYS on wasm", proc, err)
		}
		return
	}
	proc := startOSHelper(t, "success")
	state, err := proc.Wait()
	if err != nil {
		t.Fatalf("Process.Wait failed: %v", err)
	}
	if state == nil {
		t.Fatal("Process.Wait returned nil state")
	}
	if !state.Success() || !state.Exited() || state.ExitCode() != 0 {
		t.Errorf("process state = %v, want successful exit code 0", state)
	}
	if pid := state.Pid(); pid <= 0 {
		t.Errorf("ProcessState.Pid() = %d, want > 0", pid)
	}
	if str := state.String(); str == "" {
		t.Error("ProcessState.String() returned empty")
	}
	if sys := state.SystemTime(); sys < 0 {
		t.Errorf("ProcessState.SystemTime() = %v, want >= 0", sys)
	}
	if user := state.UserTime(); user < 0 {
		t.Errorf("ProcessState.UserTime() = %v, want >= 0", user)
	}
	if state.Sys() == nil || state.SysUsage() == nil {
		t.Error("ProcessState system data is nil")
	}
}

func TestProcessSignal(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		testWasmForeignProcess(t, false)
		return
	}
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Release()

	var sig os.Signal = syscall.Signal(0)
	err = proc.Signal(sig)
	if runtime.GOOS == "windows" {
		if err == nil {
			t.Error("Process.Signal(0) succeeded on Windows, want unsupported-operation error")
		}
	} else if err != nil {
		t.Errorf("Process.Signal(0): %v", err)
	}
}

func TestProcessKill(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		testWasmForeignProcess(t, true)
		return
	}
	proc := startOSHelper(t, "sleep")
	if err := proc.Kill(); err != nil {
		t.Fatalf("Process.Kill failed: %v", err)
	}
	state, err := proc.Wait()
	if err != nil {
		t.Fatalf("Process.Wait after Kill: %v", err)
	}
	if state == nil || state.Success() {
		t.Errorf("ProcessState after Kill = %v, want unsuccessful exit", state)
	}
}

func testWasmForeignProcess(t *testing.T, kill bool) {
	t.Helper()
	// WASI treats even signal 0 to self as process termination. Use a foreign
	// PID to test its no-such-process contract without terminating the suite.
	proc, err := os.FindProcess(os.Getpid() + 1000)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Release()
	if kill {
		err = proc.Kill()
	} else {
		err = proc.Signal(syscall.Signal(0))
	}
	want := error(syscall.ENOSYS)
	if runtime.GOOS == "wasip1" {
		want = os.ErrProcessDone
	}
	if !errors.Is(err, want) {
		t.Fatalf("foreign process signal (kill=%v) = %v, want %v", kill, err, want)
	}
	if state, err := proc.Wait(); state != nil || !errors.Is(err, syscall.ENOSYS) {
		t.Fatalf("foreign process Wait = %v, %v; want nil, ENOSYS", state, err)
	}
}

func TestRoot(t *testing.T) {
	tmpDir := t.TempDir()

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	defer root.Close()

	if name := root.Name(); name != tmpDir {
		t.Errorf("Root.Name() = %q, want %q", name, tmpDir)
	}

	testFile := "test.txt"
	f, err := root.Create(testFile)
	if err != nil {
		t.Errorf("Root.Create failed: %v", err)
	}
	if f != nil {
		f.WriteString("content")
		f.Close()
	}

	f, err = root.Open(testFile)
	if err != nil {
		t.Errorf("Root.Open failed: %v", err)
	}
	if f != nil {
		f.Close()
	}

	info, err := root.Stat(testFile)
	if err != nil {
		t.Errorf("Root.Stat failed: %v", err)
	} else if info.Name() != testFile {
		t.Errorf("Root.Stat().Name() = %q, want %q", info.Name(), testFile)
	}

	info, err = root.Lstat(testFile)
	if err != nil {
		t.Errorf("Root.Lstat failed: %v", err)
	} else if info.Name() != testFile {
		t.Errorf("Root.Lstat().Name() = %q, want %q", info.Name(), testFile)
	}

	subDir := "subdir"
	err = root.Mkdir(subDir, 0755)
	if err != nil {
		t.Errorf("Root.Mkdir failed: %v", err)
	}

	subRoot, err := root.OpenRoot(subDir)
	if err != nil {
		t.Errorf("Root.OpenRoot failed: %v", err)
	}
	if subRoot != nil {
		subRoot.Close()
	}

	f, err = root.OpenFile(testFile, os.O_RDONLY, 0)
	if err != nil {
		t.Errorf("Root.OpenFile failed: %v", err)
	}
	if f != nil {
		f.Close()
	}

	fsys := root.FS()
	if fsys == nil {
		t.Error("Root.FS() returned nil")
	}

	err = root.Remove(testFile)
	if err != nil {
		t.Errorf("Root.Remove failed: %v", err)
	}
}

func TestNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "newfile_test.txt")

	fd, err := openNewFileDescriptor(testFile)
	if err != nil {
		t.Fatalf("open raw file descriptor: %v", err)
	}
	f2 := os.NewFile(fd, testFile)
	if f2 == nil {
		t.Fatal("NewFile returned nil")
	}
	// NewFile takes sole ownership of the raw descriptor. In particular, do
	// not create it from another *os.File: two owners for one Windows HANDLE
	// leave the second finalizer able to close a runtime event after the
	// numeric value is reused.
	defer f2.Close()
	if f2.Name() != testFile {
		t.Errorf("NewFile().Name() = %q, want %q", f2.Name(), testFile)
	}
}

func TestOpenInRoot(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := "test.txt"
	fullPath := filepath.Join(tmpDir, testFile)
	if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenInRoot(tmpDir, testFile)
	if err != nil {
		t.Fatalf("OpenInRoot: %v", err)
	}
	if f != nil {
		defer f.Close()
		buf := make([]byte, 7)
		n, _ := f.Read(buf)
		if string(buf[:n]) != "content" {
			t.Errorf("OpenInRoot read %q, want content", buf[:n])
		}
	}
}

func TestDirEntryAndFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "entry_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var entry os.DirEntry
	for _, e := range entries {
		if e.Name() == "entry_test.txt" {
			entry = e
			break
		}
	}

	if entry == nil {
		t.Fatal("File not found in ReadDir")
	}

	info, err := entry.Info()
	if err != nil {
		t.Errorf("DirEntry.Info failed: %v", err)
	}

	var _ os.FileInfo = info
	var _ os.FileMode = info.Mode()
}

func TestExit(t *testing.T) {
	_ = os.Exit
}
