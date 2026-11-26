//go:build !llgo.idf

// Host-build tests for the tls error surface. A host build has no glue, so
// nothing here claims a session exists: the tests pin what the pure-Go
// layer owns - the text-first sentinel catalog (each with its historical
// 0x7E00 segment code and ErrName entry as annotations), the fail-closed
// dial guard (sentinel under the "tls: dial" op), the closed-conn guards
// (the transport sentinel under their op prefixes), the bare Coded forwards
// for the forwarded codes, the honest-stub contract, and the error shape
// throughout (errors.Is by sentinel identity and by code, the annotation
// reachable through esperr.Code and errors.As).
package tls

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestSentinelCatalog pins the eight handshake-phase / dial-policy
// sentinels: the text-first rendering with the code annotation, the code
// and ErrName fields the code-era diagnostics read, code-era matching
// against a raw segment literal, and errOf's identity mapping.
func TestSentinelCatalog(t *testing.T) {
	cases := []struct {
		err  *esperr.Error
		text string
		code int32
		name string
	}{
		{ErrHandshakeTimeout, "tls: handshake timeout (code 32257)",
			tlsHandshakeTimeout, "TLS_ERR_HANDSHAKE_TIMEOUT"},
		{ErrCertVerifyFailed, "tls: certificate verify failed (code 32258)",
			tlsCertVerifyFailed, "TLS_ERR_CERT_VERIFY"},
		{ErrCaParseFailed, "tls: CA parse failed (code 32259)",
			tlsCaParseFailed, "TLS_ERR_CA_PARSE"},
		{ErrWantPollExhausted, "tls: close notify wait exhausted (code 32260)",
			tlsWantPollExhausted, "TLS_ERR_WANT_POLL_EXHAUSTED"},
		{ErrHandshake, "tls: handshake failed (code 32261)",
			tlsHandshakeFailure, "TLS_ERR_HANDSHAKE"},
		{ErrPsaInitFailed, "tls: PSA crypto init failed (code 32262)",
			tlsPsaInitFailed, "TLS_ERR_PSA_INIT"},
		{ErrNoRootCAs, "tls: no root CAs (code 32263)",
			tlsNoRootCAs, "TLS_ERR_NO_ROOT_CAS"},
		{ErrNoPeerCert, "tls: no peer certificate (code 32264)",
			tlsNoPeerCert, "TLS_ERR_NO_PEER_CERT"},
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

// TestNoRootCAsGuard pins the fail-closed dial: a nil RootCAs without
// InsecureSkipVerify is refused before the dial touches the network, as
// ErrNoRootCAs under the "tls: dial" op - matched by identity, by its
// 0x7E07 code annotation against a code-era literal, and through
// esperr.Code / errors.As past the Wrap.
func TestNoRootCAsGuard(t *testing.T) {
	_, err := Dial("127.0.0.1:9443", 1000, nil)
	if !errors.Is(err, ErrNoRootCAs) {
		t.Fatalf("zero-config Dial = %v, want the ErrNoRootCAs sentinel", err)
	}
	if !errors.Is(err, &esperr.Error{Code: tlsNoRootCAs}) {
		t.Fatal("the guard error did not match a raw 0x7E07 Error (code path broken)")
	}
	if got := esperr.Code(err); got != tlsNoRootCAs {
		t.Fatalf("esperr.Code = %#x, want 0x7E07", got)
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != tlsNoRootCAs || e.Name != "TLS_ERR_NO_ROOT_CAS" {
		t.Fatal("errors.As did not surface the *esperr.Error with the annotation")
	}
	if got, want := err.Error(), "tls: dial: tls: no root CAs (code 32263)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	// The glue's defense-in-depth refusal (the same code from tlsNew) maps
	// to the bare sentinel: errors.Is matches it without any wiring.
	if !errors.Is(errOf(tlsNoRootCAs), ErrNoRootCAs) {
		t.Fatal("errOf(0x7E07) did not return the sentinel by identity")
	}
}

// TestClosedGuards pin the nil/closed-conn guards: the family's transport
// sentinel (matched by identity and by its EBADF code) under the operation
// that hit it, so the rendering is "tls: <op>: closed (code 9)".
func TestClosedGuards(t *testing.T) {
	cases := []struct {
		name string
		err  error
		op   string
	}{
		{"nil conn Read", func() error { _, e := (*TLSConn)(nil).Read(make([]byte, 1)); return e }(), "tls: read"},
		{"nil conn Write", func() error { _, e := (*TLSConn)(nil).Write([]byte{0}); return e }(), "tls: write"},
		{"nil conn Close", (*TLSConn)(nil).Close(), "tls: close"},
		{"nil conn SetDeadline", (*TLSConn)(nil).SetDeadline(100), "tls: set deadline"},
	}
	for _, c := range cases {
		if !errors.Is(c.err, ErrClosed) {
			t.Errorf("%s = %v, want the ErrClosed sentinel through the op Wrap", c.name, c.err)
		}
		if !errors.Is(c.err, esperr.ErrClosed) {
			t.Errorf("%s did not match esperr.ErrClosed - the re-export identity is broken", c.name)
		}
		if got := esperr.Code(c.err); got != 9 {
			t.Errorf("%s: esperr.Code = %d, want the EBADF 9 annotation", c.name, got)
		}
		if got := c.err.Error(); len(got) < len(c.op) || got[:len(c.op)] != c.op {
			t.Errorf("%s Error() = %q, want the %q op prefix", c.name, got, c.op)
		}
	}
	if got, want := (*TLSConn)(nil).Close().Error(), "tls: close: closed (code 9)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if errors.Is((*TLSConn)(nil).Close(), esperr.ErrTimeout) {
		t.Fatal("the closed error matched ErrTimeout - sentinel isolation broken")
	}
}

// TestErrOfForwardsBareCoded pins the pass-through conventions: negative
// mbedtls codes, lwip errnos and the negated-errno Write-contract shape all
// forward as bare named Coded values - the code is the diagnostic, no op
// wrapper, the *esperr.Error fields stay directly readable.
func TestErrOfForwardsBareCoded(t *testing.T) {
	cases := []struct {
		code int32
		text string
	}{
		{-0x7780, "MBEDTLS_ERR_SSL_FATAL_ALERT_MESSAGE (-30592)"},
		{-0x2700, "MBEDTLS_ERR_X509_CERT_VERIFY_FAILED (-9984)"},
		{econnreset, "ECONNRESET (104)"},
		{-eagain, "EAGAIN (-11)"},
		{210, "errno 210 (210)"},
	}
	for _, c := range cases {
		err := errOf(c.code)
		e, ok := err.(*esperr.Error)
		if !ok || *e != *esperr.Coded(c.code, ErrName(c.code)) {
			t.Errorf("errOf(%d) is not a bare Coded value: %v", c.code, err)
		}
		if got := err.Error(); got != c.text {
			t.Errorf("errOf(%d).Error() = %q, want %q", c.code, got, c.text)
		}
	}
	// The transport sentinel codes map by identity.
	if errOf(etimedout) != ErrTimeout || errOf(ebadf) != ErrClosed {
		t.Fatal("the transport sentinel codes do not map to the re-exported values")
	}
}

// TestTransportSentinelsReexport pins the re-export identity of the four
// transport sentinels (the package values ARE the shared esperr values).
func TestTransportSentinelsReexport(t *testing.T) {
	if ErrClosed != esperr.ErrClosed || ErrTimeout != esperr.ErrTimeout ||
		ErrWouldBlock != esperr.ErrWouldBlock || ErrEOF != esperr.ErrEOF {
		t.Fatal("the re-exported sentinels are not the shared esperr values")
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, the dial reports errUnsupported (0x7F00) from the resolver stub
// instead of faking a handshake, as a bare Coded value that matches the
// code-era literal.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	c, err := Dial("127.0.0.1:9443", 1000, &Config{InsecureSkipVerify: true})
	if !errors.Is(err, unsupported) {
		t.Fatalf("Dial on a host build = %v, want errUnsupported from the stub", err)
	}
	if c != nil {
		t.Fatal("Dial on a host build returned a conn, want nil")
	}
	if got := esperr.Code(err); got != errUnsupported {
		t.Fatalf("esperr.Code = %#x, want 0x7F00", got)
	}
	if got, want := err.Error(), "errno 32512 (32512)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
