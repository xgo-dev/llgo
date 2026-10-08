// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package browser launches LLGo WebAssembly debug sessions in Chromium.
package browser

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

const MinimumChromeMajor = 123

//go:embed extension/manifest.json
var extensionManifest []byte

//go:embed extension/devtools.html
var extensionPage []byte

//go:embed extension/plugin.js
var extensionPlugin []byte

//go:embed extension/goroutines.js
var extensionGoroutines []byte

//go:embed page.html
var sessionPage string

var chromeVersionPattern = regexp.MustCompile(`(?:Chrome(?: for Testing| Canary)?|Chromium)\s+(\d+)\.`)

type Options struct {
	Chrome       string
	ChromeArgs   []string
	SourceMaps   []browserdebug.PathMapping
	SourceRoots  []string
	KeepProfile  bool
	ProfilePath  string
	DisableTools bool
}

// Run validates artifact, starts a loopback-only debug server, installs the
// LLGo extension in an isolated profile, and waits for Chromium to exit.
func Run(artifact string, options Options, stdin io.Reader, stdout, stderr io.Writer) error {
	if err := validateChromeArgs(options.ChromeArgs); err != nil {
		return err
	}
	path, version, err := Find(options.Chrome)
	if err != nil {
		return err
	}
	session, err := StartSession(artifact, options.SourceMaps, options.SourceRoots...)
	if err != nil {
		return fmt.Errorf("llgo debug: %w", err)
	}
	defer session.Close()

	profile, profileCleanup, err := prepareProfile(options)
	if err != nil {
		return err
	}
	defer profileCleanup()
	extensionPath := filepath.Join(profile, "llgo-extension")
	if err := session.WriteExtension(extensionPath); err != nil {
		return fmt.Errorf("llgo debug: prepare browser extension: %w", err)
	}

	args := append(append([]string(nil), options.ChromeArgs...), []string{
		"--user-data-dir=" + profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-breakpad",
		"--disable-default-apps",
		"--password-store=basic",
		"--disable-extensions-except=" + extensionPath,
		"--load-extension=" + extensionPath,
	}...)
	if runtime.GOOS == "darwin" {
		// An isolated profile must not wait for a system Keychain prompt before
		// loading its command-line extension and first navigation.
		args = append(args, "--use-mock-keychain")
	}
	if !options.DisableTools {
		args = append(args, "--auto-open-devtools-for-tabs")
	}
	sessionURL := session.URL
	if options.DisableTools {
		sessionURL += "?llgo-devtools=disabled"
	}
	args = append(args, sessionURL)
	fmt.Fprintf(stderr, "llgo debug: Chromium %d; browser session %s\n", version, session.URL)
	command := exec.Command(path, args...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("llgo debug: Chromium session: %w", err)
	}
	return nil
}

func prepareProfile(options Options) (string, func(), error) {
	if options.ProfilePath != "" {
		path, err := filepath.Abs(options.ProfilePath)
		if err != nil {
			return "", func() {}, err
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", func() {}, err
		}
		return path, func() {}, nil
	}
	path, err := os.MkdirTemp("", "llgo-browser-debug-")
	if err != nil {
		return "", func() {}, fmt.Errorf("llgo debug: create Chromium profile: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(path) }
	if options.KeepProfile {
		cleanup = func() {}
	}
	return path, cleanup, nil
}

// Find resolves and validates a Chromium-family executable.
func Find(configured string) (string, int, error) {
	candidates := chromeCandidates(runtime.GOOS, configured, os.Getenv)
	seen := make(map[string]bool)
	var failures []string
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		output, err := exec.Command(path, "--version").CombinedOutput()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		match := chromeVersionPattern.FindStringSubmatch(string(output))
		if len(match) != 2 {
			failures = append(failures, fmt.Sprintf("%s: cannot parse %q", path, strings.TrimSpace(string(output))))
			continue
		}
		major, _ := strconv.Atoi(match[1])
		if major < MinimumChromeMajor {
			failures = append(failures, fmt.Sprintf("%s: version %d is older than %d", path, major, MinimumChromeMajor))
			continue
		}
		return path, major, nil
	}
	detail := ""
	if len(failures) != 0 {
		detail = ": " + strings.Join(failures, "; ")
	}
	return "", 0, fmt.Errorf("llgo debug: Chromium %d or newer is required; use -chrome or LLGO_CHROME%s", MinimumChromeMajor, detail)
}

func chromeCandidates(goos, configured string, getenv func(string) string) []string {
	// An explicitly selected executable must fail visibly rather than launch
	// a different installed browser after a typo or unsupported version.
	if configured != "" {
		return []string{configured}
	}
	if configured = getenv("LLGO_CHROME"); configured != "" {
		return []string{configured}
	}
	switch goos {
	case "darwin":
		return []string{
			"/Applications/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
	case "windows":
		candidates := []string{"chrome.exe", "chromium.exe"}
		for _, name := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if root := getenv(name); root != "" {
				candidates = append(candidates,
					filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
					filepath.Join(root, "Chromium", "Application", "chrome.exe"))
			}
		}
		return candidates
	default:
		return []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"}
	}
}

func validateChromeArgs(args []string) error {
	for _, arg := range args {
		name, _, hasValue := strings.Cut(arg, "=")
		switch name {
		case "--window-size", "--window-position":
			if !hasValue {
				return fmt.Errorf("llgo debug: browser argument %q requires --name=value", arg)
			}
		case "--start-maximized", "--headless":
			// Presentation options cannot replace the profile, extension or
			// web security policy. Values must be in the same argument.
		default:
			return fmt.Errorf("llgo debug: browser argument %q is unsupported; use --window-size, --window-position, --start-maximized or --headless", arg)
		}
	}
	return nil
}

// extensionIdentity creates a fresh unpacked-extension ID for this session.
// Only the public key is retained; it identifies the manifest, not a signing
// credential. Chromium derives the ID from the first 128 bits of its SHA-256.
func extensionIdentity() (key, origin string, err error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		return "", "", err
	}
	hash := sha256.Sum256(public)
	id := make([]byte, 32)
	for index, value := range hash[:16] {
		id[2*index] = 'a' + value>>4
		id[2*index+1] = 'a' + value&15
	}
	return base64.StdEncoding.EncodeToString(public), "chrome-extension://" + string(id), nil
}

// WriteExtension materializes the embedded unpacked extension.
func (s *Session) WriteExtension(directory string) error {
	var manifest map[string]any
	if err := json.Unmarshal(extensionManifest, &manifest); err != nil {
		return err
	}
	manifest["key"] = s.extensionKey
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	for name, data := range map[string][]byte{
		"manifest.json": manifestJSON,
		"devtools.html": extensionPage,
		"plugin.js":     extensionPlugin,
		"goroutines.js": extensionGoroutines,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type Session struct {
	URL             string
	Listener        net.Listener
	Server          *http.Server
	Bundle          *browserdebug.Bundle
	extensionKey    string
	extensionOrigin string
	pluginRequests  atomic.Uint64
	pluginReady     atomic.Uint64
	runtimeReady    atomic.Uint64
}

// StartSession starts the loopback HTTP portion of a browser debug session.
// It is exported so headless acceptance tests can exercise exactly the same
// artifact, sidecar, source, schema, and page routes as the interactive path.
func StartSession(artifact string, mappings []browserdebug.PathMapping, sourceRoots ...string) (*Session, error) {
	bundle, err := browserdebug.Load(artifact, mappings, sourceRoots...)
	if err != nil {
		return nil, err
	}
	main, err := os.ReadFile(bundle.MainPath)
	if err != nil {
		return nil, err
	}
	indexJSON, err := json.Marshal(bundle.Index)
	if err != nil {
		return nil, err
	}
	key, extensionOrigin, err := extensionIdentity()
	if err != nil {
		return nil, fmt.Errorf("create browser extension identity: %w", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start browser debug server: %w", err)
	}
	origin := "http://" + listener.Addr().String()
	mainRoute := "/" + filepath.Base(bundle.MainPath)
	mainURLPath := "/" + url.PathEscape(filepath.Base(bundle.MainPath))
	session := &Session{URL: origin + "/", Listener: listener, Bundle: bundle, extensionKey: key, extensionOrigin: extensionOrigin}

	files := map[string]servedFile{
		mainRoute:      {data: main, contentType: "application/wasm"},
		"/favicon.ico": {data: nil, contentType: "image/x-icon"},
	}
	// Execute the compiler's actual Emscripten host. Synthesizing imports or
	// loading Go's wasm_exec.js loses LLGo's Asyncify/EH/worker host contract.
	glueURL := ""
	stem := strings.TrimSuffix(bundle.MainPath, filepath.Ext(bundle.MainPath))
	for _, suffix := range []string{".mjs", ".js"} {
		data, readErr := os.ReadFile(stem + suffix)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			listener.Close()
			return nil, fmt.Errorf("read Emscripten host: %w", readErr)
		}
		name := filepath.Base(stem + suffix)
		files["/"+name] = servedFile{data: data, contentType: "text/javascript; charset=utf-8"}
		glueURL = "/" + url.PathEscape(name)
		break
	}
	if glueURL != "" {
		fsHost, readErr := os.ReadFile(filepath.Join(env.LLGoROOT(), "targets", "wasm_fs.js"))
		if readErr != nil {
			listener.Close()
			return nil, fmt.Errorf("read browser filesystem host: %w", readErr)
		}
		files["/wasm_fs.js"] = servedFile{data: fsHost, contentType: "text/javascript; charset=utf-8"}
	}
	if bundle.SymbolsPath != bundle.MainPath {
		reference, ok, err := wasmdebug.ExternalURL(main)
		if err != nil || !ok {
			listener.Close()
			return nil, fmt.Errorf("read external WebAssembly DWARF URL: %w", err)
		}
		parsed, _ := url.Parse(reference)
		sidecarRoute, err := url.PathUnescape("/" + parsed.EscapedPath())
		if err != nil {
			listener.Close()
			return nil, err
		}
		sidecar, err := os.ReadFile(bundle.SymbolsPath)
		if err != nil {
			listener.Close()
			return nil, err
		}
		files[sidecarRoute] = servedFile{data: sidecar, contentType: "application/wasm"}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		setDebugHeaders(response)
		if request.URL.Path == "/" {
			response.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(response, debugPage(mainURLPath, glueURL))
			return
		}
		if file, ok := files[request.URL.Path]; ok {
			response.Header().Set("Content-Type", file.contentType)
			response.Header().Set("Content-Length", strconv.Itoa(len(file.data)))
			_, _ = response.Write(file.data)
			return
		}
		http.NotFound(response, request)
	})
	mux.HandleFunc("/__llgo/debug-index.json", func(response http.ResponseWriter, _ *http.Request) {
		session.pluginRequests.Add(1)
		setDebugHeaders(response)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(indexJSON)
	})
	mux.HandleFunc("/__llgo/debug-schema.json", func(response http.ResponseWriter, _ *http.Request) {
		setDebugHeaders(response)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(debugabi.SchemaV1())
	})
	mux.HandleFunc("/__llgo/plugin-ready", func(response http.ResponseWriter, request *http.Request) {
		setDebugHeaders(response)
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		session.pluginReady.Add(1)
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/__llgo/runtime-ready", func(response http.ResponseWriter, request *http.Request) {
		setDebugHeaders(response)
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		session.runtimeReady.Add(1)
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/__llgo/source/", func(response http.ResponseWriter, request *http.Request) {
		setDebugHeaders(response)
		id := strings.TrimPrefix(request.URL.Path, "/__llgo/source/")
		contents, err := bundle.ReadSource(id)
		if err != nil {
			http.NotFound(response, request)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = response.Write(contents)
	})

	server := &http.Server{
		Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// Sources belong to this loopback session. Do not expose them to
			// arbitrary websites through CORS or a rebinding Host header.
			if request.Host != listener.Addr().String() {
				http.Error(response, "invalid session host", http.StatusForbidden)
				return
			}
			if from := request.Header.Get("Origin"); from != "" && from != origin {
				if from != session.extensionOrigin {
					http.Error(response, "invalid session origin", http.StatusForbidden)
					return
				}
				response.Header().Set("Access-Control-Allow-Origin", from)
				response.Header().Set("Vary", "Origin")
			}
			mux.ServeHTTP(response, request)
		}),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	session.Server = server
	go func() {
		_ = server.Serve(listener)
	}()
	return session, nil
}

// PluginRequests reports how often Chrome's Language Extension requested the
// session index. PluginReady is the stronger end-to-end readiness signal.
func (s *Session) PluginRequests() uint64 {
	if s == nil {
		return 0
	}
	return s.pluginRequests.Load()
}

// PluginReady reports how often Chrome's Language Extension completed all
// module, index, build-identity, and debugger-schema validation.
func (s *Session) PluginReady() uint64 {
	if s == nil {
		return 0
	}
	return s.pluginReady.Load()
}

// RuntimeReady reports how often the inspected page completed WebAssembly
// instantiation. It is independent of whether DevTools or the extension ran.
func (s *Session) RuntimeReady() uint64 {
	if s == nil {
		return 0
	}
	return s.runtimeReady.Load()
}

type servedFile struct {
	data        []byte
	contentType string
}

func setDebugHeaders(response http.ResponseWriter) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	response.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
}

func debugPage(modulePath, gluePath string) string {
	config, _ := json.Marshal(map[string]string{"module": modulePath, "glue": gluePath})
	return strings.Replace(sessionPage, "__LLGO_DEBUG_CONFIG__", string(config), 1)
}

func (s *Session) Close() error {
	if s == nil || s.Server == nil {
		return nil
	}
	context, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.Server.Shutdown(context)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
