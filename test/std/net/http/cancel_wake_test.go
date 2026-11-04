//go:build darwin || linux

package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTransportCancelWakesPoll(t *testing.T) {
	for i := 0; i < 8; i++ {
		entered, cancelled := make(chan struct{}), make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(entered)
			<-r.Context().Done()
			close(cancelled)
		}))
		transport := &http.Transport{Proxy: nil}
		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			resp, err := transport.RoundTrip(req)
			if resp != nil {
				resp.Body.Close()
			}
			done <- err
		}()
		<-entered
		cancel()
		timer := time.NewTimer(750 * time.Millisecond)
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("RoundTrip: %v, want context.Canceled", err)
			}
		case <-timer.C:
			t.Error("RoundTrip did not wake promptly after cancellation")
		}
		timer.Stop()
		select {
		case <-cancelled:
		case <-time.After(750 * time.Millisecond):
			t.Error("server did not observe cancellation promptly")
		}
		transport.CloseIdleConnections()
		server.Close()
	}
}
