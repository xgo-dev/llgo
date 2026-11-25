//go:build !llgo.idf

// Host-build tests for the net error surface. A host build has no glue, so
// nothing here claims a socket exists: the tests pin what the pure-Go layer
// owns - the transport-sentinel re-export identity (the package values ARE
// the shared esperr sentinels), the errOf mapping (sentinel codes by
// identity, everything else as a bare named Coded forward), the
// honest-stub contract, and the error shape throughout (errors.Is by
// sentinel identity and by code, the code annotation reachable through
// esperr.Code and errors.As).
package net

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestTransportSentinelsReexport pins the re-export identity: the four
// package-level sentinels ARE the shared esperr values (same pointers, so
// pointer comparison across packages and errors.Is behave identically),
// with the historical lwip errnos kept on Code as the annotation and the
// base's text-first rendering.
func TestTransportSentinelsReexport(t *testing.T) {
	if ErrClosed != esperr.ErrClosed || ErrTimeout != esperr.ErrTimeout ||
		ErrWouldBlock != esperr.ErrWouldBlock || ErrEOF != esperr.ErrEOF {
		t.Fatal("the re-exported sentinels are not the shared esperr values")
	}
	cases := []struct {
		err  error
		code int32
		text string
	}{
		{ErrClosed, 9, "closed (code 9)"},
		{ErrTimeout, 116, "timeout (code 116)"},
		{ErrWouldBlock, 11, "would block (code 11)"},
		{ErrEOF, 0x7C01, "EOF (code 31745)"},
	}
	for _, c := range cases {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %d, want %d", c.err, got, c.code)
		}
		if got := c.err.Error(); got != c.text {
			t.Errorf("Error() = %q, want %q", got, c.text)
		}
		// The code path: a raw errno error matches its sentinel by code.
		if !errors.Is(&esperr.Error{Code: c.code}, c.err) {
			t.Errorf("errors.Is(raw code-%d Error, %v) = false", c.code, c.err)
		}
	}
	if errors.Is(ErrWouldBlock, ErrTimeout) {
		t.Error("ErrWouldBlock matched ErrTimeout - sentinel isolation broken")
	}
}

// TestErrOfMapping pins errOf: the transport sentinel codes map by
// identity (the caller gets the re-exported pointer back), everything else
// forwards as a bare *esperr.Error named through ErrName - the lwip
// errnos, the resolver's h_errno family and the unnamed fallback alike.
func TestErrOfMapping(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	if errOf(ebadf) != ErrClosed || errOf(eagain) != ErrWouldBlock ||
		errOf(etimedout) != ErrTimeout {
		t.Fatal("the sentinel codes do not map to the re-exported values by identity")
	}
	err := errOf(econnreset)
	if !errors.Is(err, &esperr.Error{Code: econnreset}) {
		t.Fatal("the forwarded ECONNRESET did not match a raw code-104 Error")
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != econnreset || e.Name != "ECONNRESET" {
		t.Fatal("errors.As did not surface the *esperr.Error with code and name")
	}
	if got, want := err.Error(), "ECONNRESET (104)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	// The resolver's h_errno family forwards the same way (Resolve's
	// failure path).
	if got, want := errOf(210).Error(), "HOST_NOT_FOUND (210)"; got != want {
		t.Fatalf("errOf(210).Error() = %q, want %q", got, want)
	}
	// A code with no table entry renders the errno fallback.
	if got, want := errOf(999).Error(), "errno 999 (999)"; got != want {
		t.Fatalf("errOf(999).Error() = %q, want %q", got, want)
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, the driver path reports errUnsupported (0x7F00) instead of
// faking a socket, as a bare Coded value that matches the code-era literal.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	c, err := ListenUDP(nil)
	if !errors.Is(err, unsupported) {
		t.Fatalf("ListenUDP on a host build = %v, want errUnsupported from the stub", err)
	}
	if c != nil {
		t.Fatal("ListenUDP on a host build returned a conn, want nil")
	}
	if got := esperr.Code(err); got != errUnsupported {
		t.Fatalf("esperr.Code = %#x, want 0x7F00", got)
	}
	// The guard paths still report the transport sentinels on a nil conn.
	var u *UDPConn
	cerr := u.Close()
	if cerr != ErrClosed {
		t.Fatalf("nil UDPConn Close = %v, want the ErrClosed sentinel", cerr)
	}
	if got, want := cerr.Error(), "closed (code 9)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	var tc *TCPConn
	if _, err := tc.Read(make([]byte, 1)); err != ErrClosed {
		t.Fatalf("nil TCPConn Read = %v, want the ErrClosed sentinel (bare, no op prefix)", err)
	}
	var l *Listener
	if _, err := l.Accept(); err != ErrClosed {
		t.Fatalf("nil Listener Accept = %v, want the ErrClosed sentinel", err)
	}
}
