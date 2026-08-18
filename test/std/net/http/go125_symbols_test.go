//go:build go1.25

package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossOriginProtection(t *testing.T) {
	protection := http.NewCrossOriginProtection()
	if err := protection.AddTrustedOrigin("https://trusted.example"); err != nil {
		t.Fatal(err)
	}
	protection.AddInsecureBypassPattern("POST /health")

	request := httptest.NewRequest(http.MethodPost, "http://service.example/write", nil)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	request.Header.Set("Origin", "https://evil.example")
	if err := protection.Check(request); err == nil {
		t.Fatal("Check accepted a cross-origin POST")
	}
	request.Header.Set("Origin", "https://trusted.example")
	if err := protection.Check(request); err != nil {
		t.Fatalf("Check rejected a trusted origin: %v", err)
	}

	bypass := httptest.NewRequest(http.MethodPost, "http://service.example/health", nil)
	bypass.Header.Set("Sec-Fetch-Site", "cross-site")
	if err := protection.Check(bypass); err != nil {
		t.Fatalf("Check rejected a bypass pattern: %v", err)
	}

	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	recorder := httptest.NewRecorder()
	protection.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "http://service.example/write", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("same-origin handler status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	recorder = httptest.NewRecorder()
	rejected := httptest.NewRequest(http.MethodPost, "http://service.example/write", nil)
	rejected.Header.Set("Sec-Fetch-Site", "cross-site")
	protection.Handler(http.NotFoundHandler()).ServeHTTP(recorder, rejected)
	if recorder.Code != http.StatusTeapot {
		t.Fatalf("deny handler status = %d, want %d", recorder.Code, http.StatusTeapot)
	}
}
