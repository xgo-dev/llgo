package httptest_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func localServerClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	client := server.Client()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server transport has type %T, want *http.Transport", client.Transport)
	}
	transport = transport.Clone()
	// Browser Fetch cannot connect to js/wasm's in-module fake network. An
	// explicit dialer keeps these as end-to-end httptest server checks.
	transport.DialContext = (&net.Dialer{}).DialContext
	client.Transport = transport
	return client
}

// Test DefaultRemoteAddr constant
func TestDefaultRemoteAddr(t *testing.T) {
	if httptest.DefaultRemoteAddr != "1.2.3.4" {
		t.Errorf("DefaultRemoteAddr = %q, want 1.2.3.4", httptest.DefaultRemoteAddr)
	}
}

// Test NewRequest
func TestNewRequest(t *testing.T) {
	tests := []struct {
		method string
		target string
		body   io.Reader
	}{
		{"GET", "http://example.com/path", nil},
		{"POST", "http://example.com/submit", strings.NewReader("data")},
		{"PUT", "/api/resource", strings.NewReader("update")},
		{"DELETE", "/api/resource/1", nil},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.target, tt.body)
		if req == nil {
			t.Errorf("NewRequest(%q, %q, ...) returned nil", tt.method, tt.target)
			continue
		}

		if req.Method != tt.method {
			t.Errorf("Method = %q, want %q", req.Method, tt.method)
		}

		if tt.body != nil && req.Body == nil {
			t.Error("Body is nil when body was provided")
		}
	}
}

// Test NewRequestWithContext
func TestNewRequestWithContext(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, "GET", "http://example.com", nil)

	if req == nil {
		t.Fatal("NewRequestWithContext returned nil")
	}

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	if req.Context() != ctx {
		t.Error("Context not set correctly")
	}
}

// Test ResponseRecorder
func TestResponseRecorder(t *testing.T) {
	rec := httptest.NewRecorder()

	if rec == nil {
		t.Fatal("NewRecorder returned nil")
	}

	// Test initial state
	if rec.Code != 200 {
		t.Errorf("Initial Code = %d, want 200", rec.Code)
	}

	// Test WriteHeader (must be before Write)
	rec.WriteHeader(http.StatusCreated)
	if rec.Code != http.StatusCreated {
		t.Errorf("Code = %d, want %d", rec.Code, http.StatusCreated)
	}

	// Test Write
	n, err := rec.Write([]byte("Hello, World!"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 13 {
		t.Errorf("Write returned %d bytes, want 13", n)
	}

	// Test Body
	if rec.Body.String() != "Hello, World!" {
		t.Errorf("Body = %q, want 'Hello, World!'", rec.Body.String())
	}

	// Test Header
	rec.Header().Set("Content-Type", "text/plain")
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Error("Header not set correctly")
	}

	// Test Result
	result := rec.Result()
	if result.StatusCode != http.StatusCreated {
		t.Errorf("Result StatusCode = %d, want %d", result.StatusCode, http.StatusCreated)
	}
	result.Body.Close()

	// Test Flush
	rec.Flush()
	if !rec.Flushed {
		t.Error("Flushed not set to true after Flush")
	}
}

// Test ResponseRecorder with handler
func TestResponseRecorderWithHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	req := httptest.NewRequest("GET", "http://example.com", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Code = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Header().Get("X-Custom") != "test" {
		t.Error("Header not set by handler")
	}

	if rec.Body.String() != "response body" {
		t.Errorf("Body = %q, want 'response body'", rec.Body.String())
	}
}

// Test NewServer
func TestNewServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	if srv.URL == "" {
		t.Error("Server URL is empty")
	}

	// Make request to server
	resp, err := localServerClient(t, srv).Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test response" {
		t.Errorf("Body = %q, want 'test response'", body)
	}
}

// Test NewTLSServer
func TestNewTLSServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("tls response"))
	})

	srv := httptest.NewTLSServer(handler)
	defer srv.Close()

	if srv.URL == "" {
		t.Error("TLS Server URL is empty")
	}

	if !strings.HasPrefix(srv.URL, "https://") {
		t.Errorf("TLS Server URL = %q, want https:// prefix", srv.URL)
	}

	// Use server's client which has the right TLS config
	client := localServerClient(t, srv)
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "tls response" {
		t.Errorf("Body = %q, want 'tls response'", body)
	}
}

// Test NewUnstartedServer
func TestNewUnstartedServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(handler)

	// Server should not be started yet
	if srv.URL != "" {
		t.Error("Unstarted server has non-empty URL")
	}

	// Start the server
	srv.Start()
	defer srv.Close()

	if srv.URL == "" {
		t.Error("Started server URL is empty")
	}

	// Make request to started server
	resp, err := localServerClient(t, srv).Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// Test Server methods
func TestServerMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Test URL
	if srv.URL == "" {
		t.Error("Server URL is empty")
	}

	// Test Listener
	if srv.Listener == nil {
		t.Error("Server Listener is nil")
	}

	// Test Config
	if srv.Config == nil {
		t.Error("Server Config is nil")
	}

	// Test Client
	client := localServerClient(t, srv)
	if client == nil {
		t.Error("Server Client is nil")
	}

	// Test Certificate (only for TLS servers)
	tlsSrv := httptest.NewTLSServer(handler)
	defer tlsSrv.Close()

	if tlsSrv.Certificate() == nil {
		t.Error("TLS Server Certificate is nil")
	}

	// Test CloseClientConnections
	srv.CloseClientConnections()
}

// Test ResponseRecorder HeaderMap field
func TestResponseRecorderHeaderMap(t *testing.T) {
	rec := httptest.NewRecorder()

	// HeaderMap should be accessible
	if rec.HeaderMap == nil {
		t.Error("HeaderMap is nil")
	}

	rec.HeaderMap.Set("X-Test", "value")
	if rec.Header().Get("X-Test") != "value" {
		t.Error("HeaderMap not synced with Header()")
	}
}

// Test ResponseRecorder Code field
func TestResponseRecorderCodeField(t *testing.T) {
	rec := httptest.NewRecorder()

	// Initial code should be 200
	if rec.Code != 200 {
		t.Errorf("Initial Code = %d, want 200", rec.Code)
	}

	rec.WriteHeader(404)
	if rec.Code != 404 {
		t.Errorf("Code after WriteHeader(404) = %d, want 404", rec.Code)
	}
}

// Test Server StartTLS method
func TestServerStartTLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(handler)
	srv.StartTLS()
	defer srv.Close()

	if !strings.HasPrefix(srv.URL, "https://") {
		t.Errorf("Server URL = %q, want https:// prefix", srv.URL)
	}

	client := localServerClient(t, srv)
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()
}

// Test that ResponseRecorder implements http.ResponseWriter
func TestResponseRecorderImplementsResponseWriter(t *testing.T) {
	var _ http.ResponseWriter = httptest.NewRecorder()
}

// Test that ResponseRecorder implements http.Flusher
func TestResponseRecorderImplementsFlusher(t *testing.T) {
	rec := httptest.NewRecorder()
	var w http.ResponseWriter = rec
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
		if !rec.Flushed {
			t.Error("Flush did not set Flushed flag")
		}
	} else {
		t.Error("ResponseRecorder does not implement http.Flusher")
	}
}

// Test ResponseRecorder.WriteString
func TestResponseRecorderWriteString(t *testing.T) {
	rec := httptest.NewRecorder()

	n, err := rec.WriteString("test string")
	if err != nil {
		t.Fatalf("WriteString failed: %v", err)
	}
	if n != 11 {
		t.Errorf("WriteString returned %d, want 11", n)
	}
	if rec.Body.String() != "test string" {
		t.Errorf("Body = %q, want 'test string'", rec.Body.String())
	}
}

// Test multiple Write calls
func TestMultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()

	rec.Write([]byte("Hello, "))
	rec.Write([]byte("World!"))

	if rec.Body.String() != "Hello, World!" {
		t.Errorf("Body = %q, want 'Hello, World!'", rec.Body.String())
	}
}

// Test Server with custom listener address
func TestServerCustomAddr(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(handler)
	// Server will use default listener
	srv.Start()
	defer srv.Close()

	if srv.URL == "" {
		t.Error("Server URL is empty after Start")
	}
}
