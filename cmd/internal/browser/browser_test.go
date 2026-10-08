//go:build !llgo

package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestExtensionJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable")
	}
	command := exec.Command(node, "--test", filepath.Join("extension", "plugin_test.js"), filepath.Join("extension", "goroutines_test.js"), filepath.Join("testdata", "page_test.js"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("language extension tests: %v\n%s", err, output)
	}
}

func TestSessionServesArtifactIndexSchemaAndSources(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	dir := t.TempDir()
	source := filepath.Join(dir, "fixture.c")
	artifact := filepath.Join(dir, "fixture.wasm")
	if err := os.WriteFile(source, []byte("int answer(void) {\n  int value = 42;\n  return value;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compileBrowserFixture(t, source, artifact)
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err = wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, raw, 0o755); err != nil {
		t.Fatal(err)
	}

	session, err := StartSession(artifact, nil, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, route := range []string{"/", "/fixture.wasm", "/__llgo/debug-index.json", "/__llgo/debug-schema.json"} {
		response, err := http.Get(strings.TrimSuffix(session.URL, "/") + route)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || len(body) == 0 {
			t.Fatalf("GET %s = status %d bytes %d, %v", route, response.StatusCode, len(body), readErr)
		}
	}
	response, err := http.Get(strings.TrimSuffix(session.URL, "/") + "/__llgo/plugin-ready")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent || session.PluginReady() != 1 {
		t.Fatalf("plugin-ready = status %d, count %d", response.StatusCode, session.PluginReady())
	}
	response, err = http.Get(strings.TrimSuffix(session.URL, "/") + "/__llgo/debug-index.json")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var index map[string]any
	if err := json.NewDecoder(response.Body).Decode(&index); err != nil {
		t.Fatal(err)
	}
	if index["contract"] != "llgo.browser.debug" {
		t.Fatalf("browser index contract = %v", index["contract"])
	}
	for _, field := range []string{"sources", "lines", "functions", "variables", "types"} {
		if _, ok := index[field].([]any); !ok {
			t.Fatalf("browser index field %q is %T, want array", field, index[field])
		}
	}
	for _, field := range []string{"host", "origin"} {
		request, err := http.NewRequest(http.MethodGet, session.URL+"__llgo/debug-index.json", nil)
		if err != nil {
			t.Fatal(err)
		}
		if field == "host" {
			request.Host = "other.example"
		} else {
			request.Header.Set("Origin", "https://other.example")
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("foreign %s accepted: %d", field, response.StatusCode)
		}
	}

	for _, from := range []string{session.extensionOrigin, "chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", session.extensionOrigin + "/"} {
		request, _ := http.NewRequest(http.MethodGet, session.URL+"__llgo/debug-index.json", nil)
		request.Header.Set("Origin", from)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if from == session.extensionOrigin {
			if response.StatusCode != http.StatusOK || response.Header.Get("Access-Control-Allow-Origin") != from {
				t.Fatalf("session extension rejected: status=%d headers=%v", response.StatusCode, response.Header)
			}
		} else if response.StatusCode != http.StatusForbidden || response.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("foreign extension origin accepted: %q status=%d headers=%v", from, response.StatusCode, response.Header)
		}
	}

	for id, sourcePath := range session.Bundle.SourceFiles {
		response, err := http.Get(strings.TrimSuffix(session.URL, "/") + "/__llgo/source/" + id)
		if err != nil {
			t.Fatal(err)
		}
		contents, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("GET source %q (%q) = %d, %v", id, sourcePath, response.StatusCode, readErr)
		}
		if filepath.Base(sourcePath) == "fixture.c" && !bytes.Contains(contents, []byte("return value")) {
			t.Fatalf("served fixture source = %q", contents)
		}
	}
}

func TestWriteExtensionAndChromeVersion(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "extension")
	key, origin, err := extensionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	session := &Session{extensionKey: key, extensionOrigin: origin}
	if err := session.WriteExtension(directory); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"manifest.json", "devtools.html", "goroutines.js", "plugin.js"} {
		if info, err := os.Stat(filepath.Join(directory, name)); err != nil || info.Size() == 0 {
			t.Fatalf("extension file %s: %v, %+v", name, err, info)
		}
	}
	for input, want := range map[string]string{
		"Google Chrome 150.0.1":             "150",
		"Google Chrome for Testing 151.0.1": "151",
		"Google Chrome Canary 152.0.1":      "152",
		"Chromium 153.0.1":                  "153",
	} {
		if match := chromeVersionPattern.FindStringSubmatch(input); len(match) != 2 || match[1] != want {
			t.Fatalf("Chrome version match for %q = %v, want %s", input, match, want)
		}
	}
}

func TestDebugPageCanSkipDevToolsHandshake(t *testing.T) {
	page := debugPage("/fixture.wasm", "")
	for _, want := range []string{
		"__llgoLanguageExtensionReady",
		"llgo-devtools",
		"DevTools extension did not become ready",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("debug page does not contain %q", want)
		}
	}
}

func TestChromeLanguageExtension(t *testing.T) {
	chrome := os.Getenv("LLGO_BROWSER_CHROME")
	requestedArtifact := os.Getenv("LLGO_BROWSER_DEBUG_ARTIFACT")
	if chrome == "" || requestedArtifact == "" {
		t.Skip("LLGO_BROWSER_CHROME and LLGO_BROWSER_DEBUG_ARTIFACT are required")
	}
	if _, _, err := Find(chrome); err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	artifact := prepareBrowserArtifact(t, requestedArtifact)
	session, err := StartSession(artifact, nil, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	profile := t.TempDir()
	profileData := filepath.Join(profile, "profile")
	extension := filepath.Join(profile, "extension")
	if err := session.WriteExtension(extension); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	chromeArgs := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=" + profileData,
		"--no-first-run", "--no-default-browser-check",
		"--disable-background-networking", "--disable-breakpad", "--disable-default-apps",
		"--password-store=basic",
		"--disable-extensions-except=" + extension,
		"--load-extension=" + extension,
		"--auto-open-devtools-for-tabs",
		session.URL,
	}
	if runtime.GOOS == "darwin" {
		chromeArgs = append([]string{"--use-mock-keychain"}, chromeArgs...)
	}
	if os.Getenv("LLGO_BROWSER_CHROME_NO_SANDBOX") == "1" {
		chromeArgs = append([]string{"--no-sandbox"}, chromeArgs...)
	}
	if os.Getenv("LLGO_BROWSER_CHROME_GUI") == "" {
		chromeArgs = append([]string{"--headless=new"}, chromeArgs...)
	}
	command := exec.Command(chrome, chromeArgs...)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	defer func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()
	deadline := time.Now().Add(30 * time.Second)
	for (session.PluginReady() == 0 || session.RuntimeReady() == 0) && time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("Chromium exited before the language extension associated the module: %v\n%s", err, output.String())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if session.PluginReady() == 0 || session.RuntimeReady() == 0 {
		t.Fatalf("Chrome did not complete LLGo module registration/instantiation (index=%d plugin-ready=%d runtime-ready=%d)\ntargets: %s\n%s",
			session.PluginRequests(), session.PluginReady(), session.RuntimeReady(),
			chromeTargets(profileData), output.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	probe := exec.CommandContext(ctx, "node", filepath.Join("testdata", "chrome_session.mjs"), profileData, session.URL)
	if report, err := probe.CombinedOutput(); err != nil {
		t.Fatalf("real browser debugger session: %v\n%s\ntargets: %s", err, report, chromeTargets(profileData))
	} else {
		t.Logf("%s", report)
	}
}

func TestChromeWithoutLanguageExtension(t *testing.T) {
	chrome := os.Getenv("LLGO_BROWSER_CHROME")
	if chrome == "" {
		t.Skip("LLGO_BROWSER_CHROME is required")
	}
	if _, _, err := Find(chrome); err != nil {
		t.Fatal(err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	artifact := prepareBrowserArtifact(t, "fixture")
	session, err := StartSession(artifact, nil, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	args := []string{
		"--headless=new", "--remote-debugging-port=0",
		"--user-data-dir=" + filepath.Join(t.TempDir(), "profile"),
		"--no-first-run", "--no-default-browser-check", "--disable-default-apps",
		"--disable-background-networking", "--disable-breakpad",
		"--disable-extensions", "--password-store=basic",
		session.URL + "?llgo-devtools=disabled",
	}
	if runtime.GOOS == "darwin" {
		args = append([]string{"--use-mock-keychain"}, args...)
	}
	if os.Getenv("LLGO_BROWSER_CHROME_NO_SANDBOX") == "1" {
		args = append([]string{"--no-sandbox"}, args...)
	}
	var output bytes.Buffer
	command := exec.Command(chrome, args...)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	defer func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}()
	deadline := time.Now().Add(30 * time.Second)
	for session.RuntimeReady() == 0 && time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("Chromium exited before fallback WebAssembly instantiation: %v\n%s", err, output.String())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if session.RuntimeReady() == 0 {
		t.Fatalf("fallback page did not instantiate WebAssembly without the extension:\n%s", output.String())
	}
	if session.PluginRequests() != 0 || session.PluginReady() != 0 {
		t.Fatalf("fallback unexpectedly used the LLGo extension: index=%d ready=%d",
			session.PluginRequests(), session.PluginReady())
	}
}

func prepareBrowserArtifact(t *testing.T, requested string) string {
	t.Helper()
	if requested != "fixture" && requested != "fixture-external" {
		return requested
	}
	external := requested == "fixture-external"
	dir := t.TempDir()
	source := filepath.Join(dir, "fixture.c")
	artifact := filepath.Join(dir, "fixture.wasm")
	if err := os.WriteFile(source, []byte("int answer(void) {\n  int value = 42;\n  return value;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compileBrowserFixture(t, source, artifact)
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err = wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	if external {
		if err := os.WriteFile(filepath.Join(dir, "fixture debug.wasm"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		raw, err = wasmdebug.Externalize(raw, "fixture%20debug.wasm")
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(artifact, raw, 0o755); err != nil {
		t.Fatal(err)
	}
	return artifact
}

func chromeTargets(profile string) string {
	data, err := os.ReadFile(filepath.Join(profile, "DevToolsActivePort"))
	if err != nil {
		return err.Error()
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "DevToolsActivePort is empty"
	}
	response, err := http.Get("http://127.0.0.1:" + fields[0] + "/json/list")
	if err != nil {
		return err.Error()
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return err.Error()
	}
	return string(contents)
}

func compileBrowserFixture(t *testing.T, source, artifact string) {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is unavailable")
	}
	command := exec.Command(clang,
		"--target=wasm32-unknown-unknown", "-O0", "-g", "-nostdlib",
		"-Wl,--no-entry", "-Wl,--export=answer", "-o", artifact, source,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile WebAssembly fixture on %s/%s: %v\n%s", runtime.GOOS, runtime.GOARCH, err, output)
	}
}

func TestBrowserChromeOptions(t *testing.T) {
	for _, args := range [][]string{
		{"--user-data-dir=/real/profile"}, {"--disable-extensions-except=/other"},
		{"--load-extension=/other"}, {"--disable-web-security"}, {"--no-sandbox"},
		{"https://other.example"}, {"--window-size", "900,700"}, {"--", "--disable-web-security"},
	} {
		if err := validateChromeArgs(args); err == nil {
			t.Fatalf("unsafe browser arguments accepted: %v", args)
		}
	}
	if err := validateChromeArgs([]string{"--window-size=900,700", "--headless=new", "--start-maximized"}); err != nil {
		t.Fatal(err)
	}
	environment := map[string]string{"ProgramFiles": "system64", "ProgramFiles(x86)": "system32", "LOCALAPPDATA": "user"}
	getenv := func(key string) string { return environment[key] }
	candidates := chromeCandidates("windows", "", getenv)
	for _, root := range []string{"system64", "system32", "user"} {
		for _, vendor := range []string{"Google/Chrome", "Chromium"} {
			want := filepath.Join(root, filepath.FromSlash(vendor), "Application", "chrome.exe")
			found := false
			for _, candidate := range candidates {
				found = found || candidate == want
			}
			if !found {
				t.Fatalf("Windows discovery omitted %q: %v", want, candidates)
			}
		}
	}
	for _, goos := range []string{"windows", "darwin", "linux"} {
		environment["LLGO_CHROME"] = "environment-choice"
		if got := chromeCandidates(goos, "explicit-choice", getenv); len(got) != 1 || got[0] != "explicit-choice" {
			t.Fatalf("explicit browser selection fell back: %v", got)
		}
		if got := chromeCandidates(goos, "", getenv); len(got) != 1 || got[0] != "environment-choice" {
			t.Fatalf("environment browser selection fell back: %v", got)
		}
	}
}

func TestSessionRestrictsRecordedSourceFiles(t *testing.T) {
	// Real DWARF claims a readable source outside the artifact directory.
	sourceDir, artifactDir := t.TempDir(), t.TempDir()
	source := filepath.Join(sourceDir, "private.c")
	if err := os.WriteFile(source, []byte("int answer(void) { return 42; }"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(artifactDir, "fixture.wasm")
	compileBrowserFixture(t, source, artifact)
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err = wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, mapped := range []bool{false, true} {
		var mappings []browserdebug.PathMapping
		if mapped {
			mappings = []browserdebug.PathMapping{{From: sourceDir, To: sourceDir}}
		}
		session, err := StartSession(artifact, mappings)
		if err != nil {
			t.Fatal(err)
		}
		for _, indexed := range session.Bundle.Index.Sources {
			response, err := http.Get(strings.TrimSuffix(session.URL, "/") + indexed.URL)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if mapped {
				if response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("return 42")) {
					t.Fatalf("mapped source unavailable: status=%d body=%q", response.StatusCode, body)
				}
			} else if indexed.Local || response.StatusCode != http.StatusNotFound {
				t.Fatalf("untrusted source exposed: %+v status=%d", indexed, response.StatusCode)
			}
		}
		session.Close()
	}
}
