package httputil_test

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

type bytePool struct{}

func (bytePool) Get() []byte { return make([]byte, 0, 1024) }
func (bytePool) Put([]byte)  {}

func localServerClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	client := server.Client()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server transport has type %T, want *http.Transport", client.Transport)
	}
	transport = transport.Clone()
	transport.DialContext = (&net.Dialer{}).DialContext
	client.Transport = transport
	return client
}

func TestDumpHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/path?q=1", strings.NewReader("abc"))
	dumpReq, err := httputil.DumpRequest(req, true)
	if err != nil {
		t.Fatalf("DumpRequest: %v", err)
	}
	if !strings.Contains(string(dumpReq), "POST http://example.com/path?q=1") {
		t.Fatalf("unexpected DumpRequest output: %q", dumpReq)
	}

	outReq, _ := http.NewRequest(http.MethodGet, "http://example.com/hello", nil)
	dumpOut, err := httputil.DumpRequestOut(outReq, false)
	if err != nil {
		t.Fatalf("DumpRequestOut: %v", err)
	}
	if !strings.Contains(string(dumpOut), "GET /hello") {
		t.Fatalf("unexpected DumpRequestOut output: %q", dumpOut)
	}

	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
	}
	resp.Header.Set("X-Test", "1")
	dumpResp, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatalf("DumpResponse: %v", err)
	}
	if !strings.Contains(string(dumpResp), "200 OK") {
		t.Fatalf("unexpected DumpResponse output: %q", dumpResp)
	}
}

func TestChunkedReaderWriter(t *testing.T) {
	var buf bytes.Buffer
	w := httputil.NewChunkedWriter(&buf)
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("chunked write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("chunked close: %v", err)
	}

	r := httputil.NewChunkedReader(bytes.NewReader(buf.Bytes()))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("chunked read: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("chunked roundtrip = %q, want %q", got, "hello")
	}
}

func TestProxyRequestHelpers(t *testing.T) {
	in := httptest.NewRequest(http.MethodGet, "http://in.example/dir?a=1", nil)
	out := in.Clone(in.Context())
	pr := &httputil.ProxyRequest{In: in, Out: out}

	target, err := url.Parse("http://upstream.example/base")
	if err != nil {
		t.Fatalf("Parse target: %v", err)
	}
	pr.SetURL(target)
	if !strings.Contains(pr.Out.URL.Path, "/base/dir") {
		t.Fatalf("SetURL path = %q", pr.Out.URL.Path)
	}

	pr.SetXForwarded()
	if pr.Out.Header.Get("X-Forwarded-Host") == "" {
		t.Fatal("X-Forwarded-Host not set")
	}
}

func TestReverseProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := r.Method + " " + r.URL.Path
		if n, err := io.WriteString(w, body); err != nil || n != len(body) {
			t.Errorf("WriteString response = (%d, %v), want (%d, nil)", n, err, len(body))
		}
	}))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("Parse backend URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = localServerClient(t, backend).Transport

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://front.example/direct", nil)
	proxy.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("direct ServeHTTP status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "GET /direct") {
		t.Fatalf("unexpected direct ServeHTTP body: %q", rr.Body.String())
	}

	front := httptest.NewServer(proxy)
	defer front.Close()

	resp, err := localServerClient(t, front).Get(front.URL + "/echo")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "GET /echo") {
		t.Fatalf("unexpected proxy response: %q", body)
	}
}

func TestPublicAPISymbols(t *testing.T) {
	if httputil.ErrPersistEOF == nil || httputil.ErrClosed == nil || httputil.ErrPipeline == nil || httputil.ErrLineTooLong == nil {
		t.Fatal("expected exported errors to be non-nil")
	}

	_ = httputil.DumpRequest
	_ = httputil.DumpRequestOut
	_ = httputil.DumpResponse
	_ = httputil.NewChunkedReader
	_ = httputil.NewChunkedWriter
	_ = httputil.NewClientConn
	_ = httputil.NewProxyClientConn
	_ = httputil.NewSingleHostReverseProxy
	_ = httputil.NewServerConn

	var _ httputil.BufferPool = bytePool{}

	_ = httputil.ClientConn{}
	_ = httputil.ProxyRequest{}
	_ = httputil.ReverseProxy{}
	_ = httputil.ServerConn{}
}

func TestClientConnMethods(t *testing.T) {
	sconn, cconn := net.Pipe()
	defer sconn.Close()
	defer cconn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		br := bufio.NewReader(sconn)

		req1, err := http.ReadRequest(br)
		if err != nil {
			t.Errorf("server ReadRequest #1: %v", err)
			return
		}
		if err := req1.Body.Close(); err != nil {
			t.Errorf("req1.Body.Close: %v", err)
			return
		}
		if req1.URL.Path != "/one" {
			t.Errorf("path #1 = %q, want /one", req1.URL.Path)
			return
		}
		resp1 := "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"
		if n, err := io.WriteString(sconn, resp1); err != nil || n != len(resp1) {
			t.Errorf("server WriteString #1 = (%d, %v), want (%d, nil)", n, err, len(resp1))
			return
		}

		req2, err := http.ReadRequest(br)
		if err != nil {
			t.Errorf("server ReadRequest #2: %v", err)
			return
		}
		if err := req2.Body.Close(); err != nil {
			t.Errorf("req2.Body.Close: %v", err)
			return
		}
		if req2.URL.Path != "/two" {
			t.Errorf("path #2 = %q, want /two", req2.URL.Path)
			return
		}
		resp2 := "HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\npong"
		if n, err := io.WriteString(sconn, resp2); err != nil || n != len(resp2) {
			t.Errorf("server WriteString #2 = (%d, %v), want (%d, nil)", n, err, len(resp2))
		}
	}()

	cc := httputil.NewClientConn(cconn, nil)
	req1, _ := http.NewRequest(http.MethodGet, "http://example.com/one", nil)
	if err := cc.Write(req1); err != nil {
		t.Fatalf("ClientConn.Write: %v", err)
	}
	if cc.Pending() < 0 {
		t.Fatalf("ClientConn.Pending = %d, want >= 0", cc.Pending())
	}
	resp1, err := cc.Read(req1)
	if err != nil {
		t.Fatalf("ClientConn.Read: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	if err := resp1.Body.Close(); err != nil {
		t.Fatalf("resp1.Body.Close: %v", err)
	}
	if string(body1) != "ok" {
		t.Fatalf("Read body = %q, want %q", body1, "ok")
	}

	req2, _ := http.NewRequest(http.MethodGet, "http://example.com/two", nil)
	resp2, err := cc.Do(req2)
	if err != nil {
		t.Fatalf("ClientConn.Do: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	if err := resp2.Body.Close(); err != nil {
		t.Fatalf("resp2.Body.Close: %v", err)
	}
	if string(body2) != "pong" {
		t.Fatalf("Do body = %q, want %q", body2, "pong")
	}

	hijackedConn, hijackedReader := cc.Hijack()
	if hijackedConn == nil || hijackedReader == nil {
		t.Fatal("ClientConn.Hijack returned nil")
	}
	if err := hijackedConn.Close(); err != nil {
		t.Fatalf("hijackedConn.Close: %v", err)
	}
	<-done
}

func TestClientConnCloseAndProxyConn(t *testing.T) {
	sconn1, cconn1 := net.Pipe()
	cc1 := httputil.NewClientConn(cconn1, nil)
	if err := cc1.Close(); err != nil {
		t.Fatalf("ClientConn.Close: %v", err)
	}
	if err := sconn1.Close(); err != nil {
		t.Fatalf("sconn1.Close: %v", err)
	}

	sconn2, cconn2 := net.Pipe()
	cc2 := httputil.NewProxyClientConn(cconn2, nil)
	if err := cc2.Close(); err != nil {
		t.Fatalf("ProxyClientConn.Close: %v", err)
	}
	if err := sconn2.Close(); err != nil {
		t.Fatalf("sconn2.Close: %v", err)
	}
}

func TestServerConnMethods(t *testing.T) {
	sconn, cconn := net.Pipe()
	defer sconn.Close()
	defer cconn.Close()

	sc := httputil.NewServerConn(sconn, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		reqText := "GET /x HTTP/1.1\r\nHost: example.com\r\n\r\n"
		if n, err := io.WriteString(cconn, reqText); err != nil || n != len(reqText) {
			t.Errorf("client WriteString request = (%d, %v), want (%d, nil)", n, err, len(reqText))
			return
		}
		br := bufio.NewReader(cconn)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Errorf("client ReadResponse: %v", err)
			return
		}
		body, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
			return
		}
		if string(body) != "srv" {
			t.Errorf("response body = %q, want %q", body, "srv")
			return
		}
	}()

	req, err := sc.Read()
	if err != nil {
		t.Fatalf("ServerConn.Read: %v", err)
	}
	if req.URL.Path != "/x" {
		t.Fatalf("request path = %q, want /x", req.URL.Path)
	}
	if sc.Pending() < 0 {
		t.Fatalf("ServerConn.Pending = %d, want >= 0", sc.Pending())
	}

	resp := &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: 3,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader("srv")),
	}
	resp.Header.Set("Content-Length", "3")
	resp.Header.Set("Content-Type", "text/plain")
	if err := sc.Write(req, resp); err != nil {
		t.Fatalf("ServerConn.Write: %v", err)
	}
	<-done

	hc, hr := sc.Hijack()
	if hc == nil || hr == nil {
		t.Fatal("ServerConn.Hijack returned nil")
	}
	if err := hc.Close(); err != nil {
		t.Fatalf("hijacked server conn close: %v", err)
	}
}

func TestServerConnClose(t *testing.T) {
	sconn, cconn := net.Pipe()
	sc := httputil.NewServerConn(sconn, nil)
	if err := sc.Close(); err != nil {
		t.Fatalf("ServerConn.Close: %v", err)
	}
	if err := cconn.Close(); err != nil {
		t.Fatalf("cconn.Close: %v", err)
	}
}
