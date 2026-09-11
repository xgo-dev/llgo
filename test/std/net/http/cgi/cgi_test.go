package cgi_test

import (
	"bytes"
	"log"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
)

const cgiHelperEnv = "LLGO_TEST_CGI_HELPER"

func TestCGIHelperProcess(t *testing.T) {
	if os.Getenv(cgiHelperEnv) != "1" {
		return
	}
	_, err := os.Stdout.WriteString("Status: 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"method=" + os.Getenv("REQUEST_METHOD") + " query=" + os.Getenv("QUERY_STRING") + "\r\n")
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRequestFromMap(t *testing.T) {
	req, err := cgi.RequestFromMap(map[string]string{
		"REQUEST_METHOD":  "POST",
		"SERVER_PROTOCOL": "HTTP/1.1",
		"HTTP_HOST":       "example.com",
		"REQUEST_URI":     "/cgi-bin/app?x=1&y=2",
		"SCRIPT_NAME":     "/cgi-bin/app",
		"QUERY_STRING":    "x=1&y=2",
		"REMOTE_ADDR":     "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("RequestFromMap: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("Method = %q, want POST", req.Method)
	}
	if req.URL.Path != "/cgi-bin/app" {
		t.Fatalf("URL.Path = %q, want /cgi-bin/app", req.URL.Path)
	}
	if req.URL.RawQuery != "x=1&y=2" {
		t.Fatalf("URL.RawQuery = %q, want x=1&y=2", req.URL.RawQuery)
	}
	if req.Host != "example.com" {
		t.Fatalf("Host = %q, want example.com", req.Host)
	}
}

func TestRequestWithoutCGIEnv(t *testing.T) {
	t.Setenv("REQUEST_METHOD", "")
	if _, err := cgi.Request(); err == nil {
		t.Fatal("expected cgi.Request to fail without CGI environment")
	}
}

func TestPublicAPISymbols(t *testing.T) {
	_ = cgi.Request
	_ = cgi.RequestFromMap
	_ = cgi.Serve

	_ = cgi.Handler{}
}

func TestHandlerServeHTTP(t *testing.T) {
	var logs bytes.Buffer
	h := &cgi.Handler{
		Path:   os.Args[0],
		Root:   "/cgi-bin",
		Dir:    t.TempDir(),
		Args:   []string{"-test.run=^TestCGIHelperProcess$"},
		Env:    []string{cgiHelperEnv + "=1"},
		Logger: log.New(&logs, "", 0),
	}
	req := httptest.NewRequest("GET", "http://example.com/cgi-bin/app.sh?x=1&y=2", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	if runtime.GOARCH == "wasm" {
		// CGI process execution is unavailable in Go's wasm profiles. Exercise
		// and check the handler's HTTP error response, not a fictitious child.
		if res.StatusCode != 500 || !strings.Contains(strings.ToLower(logs.String()), "not implemented") {
			t.Fatalf("unsupported CGI execution: status=%d log=%q", res.StatusCode, logs.String())
		}
		if w.Body.Len() != 0 {
			t.Fatalf("unexpected CGI error body: %q", w.Body.String())
		}
		return
	}
	if res.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "method=GET") || !strings.Contains(body, "query=x=1&y=2") {
		t.Fatalf("unexpected body: %q", body)
	}
}
