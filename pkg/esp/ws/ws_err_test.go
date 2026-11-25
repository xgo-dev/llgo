package ws

// Host-build tests for the ws error shape (the framing/handshake behavior
// itself is pinned in ws_test.go). The sentinels are text-first now: these
// tests pin the rendering with the historical 0x7D00 segment code as the
// annotation, the ErrName entries the code-era diagnostics read, code-era
// matching against raw segment literals, errOf's identity mapping, and the
// re-export identity of the transport sentinels.
import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestSentinelCatalog pins the eight WS-layer sentinels: the text-first
// rendering with the code annotation, the code and ErrName fields, and
// code-era matching against a raw 0x7Dxx literal.
func TestSentinelCatalog(t *testing.T) {
	cases := []struct {
		err  *esperr.Error
		text string
		code int32
		name string
	}{
		{ErrHandshake, "ws: malformed handshake response (code 32001)",
			wsErrHandshake, "WS_ERR_HANDSHAKE"},
		{ErrNotWS, "ws: not a websocket upgrade (code 32002)",
			wsErrNotWS, "WS_ERR_NOT_WEBSOCKET"},
		{ErrAcceptKey, "ws: accept key mismatch (code 32003)",
			wsErrAcceptKey, "WS_ERR_ACCEPT_KEY"},
		{ErrTooBig, "ws: frame too large (code 32004)",
			wsErrTooBig, "WS_ERR_FRAME_TOO_BIG"},
		{ErrProtocol, "ws: protocol error (code 32005)",
			wsErrProtocol, "WS_ERR_PROTOCOL"},
		{ErrPeerClosed, "ws: peer closed (code 32006)",
			wsErrPeerClosed, "WS_ERR_PEER_CLOSED"},
		{ErrConnDead, "ws: connection dead (code 32007)",
			wsErrConnDead, "WS_ERR_CONN_DEAD"},
		{ErrNoDeadline, "ws: no deadline support (code 32008)",
			wsErrNoDeadline, "WS_ERR_NO_DEADLINE"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.text {
			t.Errorf("Error() = %q, want %q", got, c.text)
		}
		if c.err.Code != c.code || c.err.Name != c.name {
			t.Errorf("%p: Code/Name = %d/%q, want %d/%q", c.err,
				c.err.Code, c.err.Name, c.code, c.name)
		}
		if !errors.Is(c.err, &esperr.Error{Code: c.code}) {
			t.Errorf("errors.Is(sentinel, raw %#x Error) = false - the code path is broken", c.code)
		}
		if errOf(c.code) != error(c.err) {
			t.Errorf("errOf(%#x) did not return the sentinel by identity", c.code)
		}
	}
}

// TestTransportSentinelsReexport pins the re-export identity of the four
// transport sentinels (the package values ARE the shared esperr values -
// the pointer-equality checks in ws_test.go rely on it).
func TestTransportSentinelsReexport(t *testing.T) {
	if ErrClosed != esperr.ErrClosed || ErrTimeout != esperr.ErrTimeout ||
		ErrWouldBlock != esperr.ErrWouldBlock || ErrEOF != esperr.ErrEOF {
		t.Fatal("the re-exported sentinels are not the shared esperr values")
	}
}

// TestErrOfForwardsBareCoded pins the pass-through convention: an lwip
// errno from a transport-shaped code path forwards as a bare named Coded
// value - the code is the diagnostic, the *esperr.Error fields stay
// directly readable. (The ws package pins only the four transport errnos;
// 111 is ECONNREFUSED, one an echo transport can plausibly surface.)
func TestErrOfForwardsBareCoded(t *testing.T) {
	err := errOf(111)
	e, ok := err.(*esperr.Error)
	if !ok || *e != *esperr.Coded(111, ErrName(111)) {
		t.Fatalf("errOf(111) is not a bare Coded value: %v", err)
	}
	if got, want := err.Error(), "ECONNREFUSED (111)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	var as *esperr.Error
	if !errors.As(err, &as) || as.Code != 111 || as.Name != "ECONNREFUSED" {
		t.Fatal("errors.As did not surface the *esperr.Error with code and name")
	}
	if got, want := errOf(77).Error(), "errno 77 (77)"; got != want {
		t.Fatalf("errOf(77).Error() = %q, want %q", got, want)
	}
	// The transport sentinel codes map by identity.
	if errOf(ebadf) != ErrClosed || errOf(eagain) != ErrWouldBlock ||
		errOf(etimedout) != ErrTimeout {
		t.Fatal("the transport sentinel codes do not map to the re-exported values")
	}
}

// TestClosedSentinelShape pins the defensive ErrClosed: the family's
// transport sentinel, bare (no op prefix - the pointer-equality contract
// the framing tests rely on), with the text-first rendering.
func TestClosedSentinelShape(t *testing.T) {
	var w *WSConn
	if _, err := w.ReadMessage(make([]byte, 1)); err != ErrClosed {
		t.Fatalf("nil WSConn ReadMessage = %v, want the bare ErrClosed sentinel", err)
	}
	if err := w.Close(1000); err != ErrClosed {
		t.Fatalf("nil WSConn Close = %v, want the bare ErrClosed sentinel", err)
	}
	if got, want := ErrClosed.Error(), "closed (code 9)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if got := esperr.Code(ErrClosed); got != ebadf {
		t.Fatalf("esperr.Code(ErrClosed) = %d, want the EBADF 9 annotation", got)
	}
}
