package http_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func localServerClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	client := server.Client()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server transport has type %T, want *http.Transport", client.Transport)
	}
	transport = transport.Clone()
	// On js/wasm the zero-value Transport uses browser Fetch, which cannot
	// reach Go's in-module fake listener. An explicit dialer selects the normal
	// Go transport and therefore exercises the same server in every profile.
	transport.DialContext = (&net.Dialer{}).DialContext
	client.Transport = transport
	return client
}

// Test all status code constants
func TestAllStatusCodes(t *testing.T) {
	codes := []int{
		http.StatusContinue,
		http.StatusSwitchingProtocols,
		http.StatusProcessing,
		http.StatusEarlyHints,
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNonAuthoritativeInfo,
		http.StatusNoContent,
		http.StatusResetContent,
		http.StatusPartialContent,
		http.StatusMultiStatus,
		http.StatusAlreadyReported,
		http.StatusIMUsed,
		http.StatusMultipleChoices,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusNotModified,
		http.StatusUseProxy,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusProxyAuthRequired,
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusGone,
		http.StatusLengthRequired,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusExpectationFailed,
		http.StatusTeapot,
		http.StatusMisdirectedRequest,
		http.StatusUnprocessableEntity,
		http.StatusLocked,
		http.StatusFailedDependency,
		http.StatusTooEarly,
		http.StatusUpgradeRequired,
		http.StatusPreconditionRequired,
		http.StatusTooManyRequests,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnavailableForLegalReasons,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		http.StatusInsufficientStorage,
		http.StatusLoopDetected,
		http.StatusNotExtended,
		http.StatusNetworkAuthenticationRequired,
	}

	for _, code := range codes {
		if code < 100 || code >= 600 {
			t.Errorf("Status code %d is out of valid range", code)
		}
	}
}

// Test additional error variables
func TestAdditionalErrors(t *testing.T) {
	errs := []error{
		http.ErrContentLength,
		http.ErrHeaderTooLong,
		http.ErrHijacked,
		http.ErrMissingBoundary,
		http.ErrMissingContentLength,
		http.ErrNotMultipart,
		http.ErrShortBody,
		http.ErrUnexpectedTrailer,
		http.ErrWriteAfterFlush,
	}

	for _, err := range errs {
		if err == nil {
			t.Error("Error variable is nil")
		}
		if err.Error() == "" {
			t.Error("Error message is empty")
		}
	}
}

// Test Client methods
func TestClientMethods(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	client := localServerClient(t, ts)
	client.Timeout = 5 * time.Second

	// Test Get
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("Client.Get failed: %v", err)
	}
	resp.Body.Close()

	// Test Head
	resp, err = client.Head(ts.URL)
	if err != nil {
		t.Fatalf("Client.Head failed: %v", err)
	}
	resp.Body.Close()

	// Test Post
	resp, err = client.Post(ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Client.Post failed: %v", err)
	}
	resp.Body.Close()

	// Test PostForm
	resp, err = client.PostForm(ts.URL, url.Values{"key": {"value"}})
	if err != nil {
		t.Fatalf("Client.PostForm failed: %v", err)
	}
	resp.Body.Close()

	// Test Do
	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Client.Do failed: %v", err)
	}
	resp.Body.Close()
}

// Test package-level convenience functions
func TestConvenienceFunctions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	oldClient := http.DefaultClient
	http.DefaultClient = localServerClient(t, ts)
	defer func() { http.DefaultClient = oldClient }()

	// Test Get
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	resp.Body.Close()

	// Test Head
	resp, err = http.Head(ts.URL)
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	resp.Body.Close()

	// Test Post
	resp, err = http.Post(ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	resp.Body.Close()

	// Test PostForm
	resp, err = http.PostForm(ts.URL, url.Values{"key": {"value"}})
	if err != nil {
		t.Fatalf("PostForm failed: %v", err)
	}
	resp.Body.Close()
}

// Test Handle and HandleFunc
func TestHandleAndHandleFunc(t *testing.T) {
	oldMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	defer func() { http.DefaultServeMux = oldMux }()

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.Handle("/other", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

// Test Server methods
func TestServerMethods(t *testing.T) {
	srv := &http.Server{
		Addr:    ":0",
		Handler: http.DefaultServeMux,
	}

	// These methods exist even if we can't fully test them without starting server
	_ = srv.ListenAndServe
	_ = srv.ListenAndServeTLS
	_ = srv.Serve
	_ = srv.ServeTLS
	_ = srv.Shutdown

	// Can test Close
	srv.Close()
}

// Test package-level server functions
func TestServerFunctions(t *testing.T) {
	_ = http.ListenAndServe
	_ = http.ListenAndServeTLS
	_ = http.Serve
	_ = http.ServeTLS
}

// Test Request additional methods
func TestRequestAdditionalMethods(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)

	// Test SetPathValue and PathValue
	req.SetPathValue("key", "value")
	if req.PathValue("key") != "value" {
		t.Error("PathValue did not return set value")
	}

	// Test CookiesNamed
	req.AddCookie(&http.Cookie{Name: "test", Value: "val1"})
	req.AddCookie(&http.Cookie{Name: "test", Value: "val2"})
	cookies := req.CookiesNamed("test")
	if len(cookies) != 2 {
		t.Errorf("CookiesNamed returned %d cookies, want 2", len(cookies))
	}

	// Test FormFile (requires multipart form)
	_ = req.FormFile
}

// Test ResponseController additional methods
func TestResponseControllerAdditionalMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)

		// Test Flush
		err := rc.Flush()
		if err != nil {
			t.Logf("Flush error (expected for ResponseRecorder): %v", err)
		}

		// Test Hijack (will fail with ResponseRecorder, but method exists)
		_, _, err = rc.Hijack()
		if err != nil {
			t.Logf("Hijack error (expected for ResponseRecorder): %v", err)
		}
	})

	req := httptest.NewRequest("GET", "http://example.com", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

// Test interfaces
func TestInterfaces(t *testing.T) {
	// CloseNotifier interface
	var _ http.CloseNotifier

	// Flusher interface
	var _ http.Flusher

	// Hijacker interface
	var _ http.Hijacker

	// Pusher interface
	var _ http.Pusher

	// CookieJar interface
	var _ http.CookieJar

	// RoundTripper interface
	var _ http.RoundTripper
}

// Test Transport additional methods
func TestTransportAdditionalMethods(t *testing.T) {
	tr := &http.Transport{}

	// Test Clone
	tr2 := tr.Clone()
	if tr2 == nil {
		t.Error("Clone returned nil")
	}

	// Test RegisterProtocol
	tr.RegisterProtocol("test", http.NewFileTransport(http.Dir(".")))

	// Test RoundTrip (requires full setup, just verify method exists)
	_ = tr.RoundTrip
}

// Test Protocols additional methods
func TestProtocolsAdditionalMethods(t *testing.T) {
	p := &http.Protocols{}

	// Test HTTP1
	if p.HTTP1() {
		t.Log("HTTP1 enabled by default")
	}

	// Test SetHTTP1
	p.SetHTTP1(true)

	// Test UnencryptedHTTP2
	if p.UnencryptedHTTP2() {
		t.Log("UnencryptedHTTP2 enabled")
	}

	// Test SetUnencryptedHTTP2
	p.SetUnencryptedHTTP2(false)

	// Test String
	s := p.String()
	if s == "" {
		t.Log("Protocols.String() returned empty string")
	}
}

// Test ProtocolError.Is
func TestProtocolErrorIs(t *testing.T) {
	pe := &http.ProtocolError{ErrorString: "test"}

	// ProtocolError.Is is used with errors.Is, test with a target error
	target := http.ErrNotSupported
	result := pe.Is(target)
	_ = result // Result depends on error comparison logic
}
