//go:build go1.26

package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientConn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		io.WriteString(w, request.Host+request.URL.Path)
	}))
	defer server.Close()

	transport := new(http.Transport)
	address := strings.TrimPrefix(server.URL, "http://")
	connection, err := transport.NewClientConn(t.Context(), "http", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	if connection.Err() != nil || connection.Available() != 1 || connection.InFlight() != 0 {
		t.Fatalf("initial state: available=%d in-flight=%d err=%v", connection.Available(), connection.InFlight(), connection.Err())
	}
	if err := connection.Reserve(); err != nil {
		t.Fatal(err)
	}
	if connection.Available() != 0 || connection.InFlight() != 1 {
		t.Fatalf("reserved state: available=%d in-flight=%d", connection.Available(), connection.InFlight())
	}
	connection.Release()
	if connection.Available() != 1 || connection.InFlight() != 0 {
		t.Fatalf("released state: available=%d in-flight=%d", connection.Available(), connection.InFlight())
	}

	var stateChanges atomic.Int32
	connection.SetStateHook(func(*http.ClientConn) {
		stateChanges.Add(1)
	})
	request, err := http.NewRequest(http.MethodGet, "http://example.test/resource", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := connection.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); got != "example.test/resource" {
		t.Fatalf("response body = %q, want %q", got, "example.test/resource")
	}
	if stateChanges.Load() == 0 {
		t.Fatal("state hook was not called after the request completed")
	}

	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if connection.Err() == nil || connection.Available() != 0 || connection.InFlight() != 0 {
		t.Fatalf("closed state: available=%d in-flight=%d err=%v", connection.Available(), connection.InFlight(), connection.Err())
	}
}
