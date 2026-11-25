package esperr

import (
	"errors"
	"testing"
)

// TestRendering pins Error() output for all three construction shapes,
// exactly:
//
//   - New sentinel without code: the bare text;
//   - sentinel (or any Error) with text + historical code: "text (code N)";
//   - Coded with a name: the code-era "NAME (N)";
//   - Coded without a name: the code-era "esp error N".
func TestRendering(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{New("esp/x: not ready"), "esp/x: not ready"},
		{&Error{Code: 29184}, "esp error 29184"},
		{&Error{Code: 116, Name: "ETIMEDOUT"}, "ETIMEDOUT (116)"},
		{Coded(0x7F00, ""), "esp error 32512"},
		{Coded(0x102, "ESP_ERR_INVALID_ARG"), "ESP_ERR_INVALID_ARG (258)"},
		{ErrTimeout, "timeout (code 116)"},
		{ErrClosed, "closed (code 9)"},
		{ErrWouldBlock, "would block (code 11)"},
		{ErrEOF, "EOF (code 31745)"},
		// A code attached to a New sentinel renders text-first with the
		// annotation.
		{withCode(New("esp/x: broken"), 0x7201), "esp/x: broken (code 29185)"},
		// Negative codes (mbedtls) keep the minus in both rendering eras.
		{Coded(-0x2700, ""), "esp error -9984"},
		{Coded(-0x2700, "MBEDTLS_ERR_X509_CERT_VERIFY_FAILED"), "MBEDTLS_ERR_X509_CERT_VERIFY_FAILED (-9984)"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
	}
}

// TestOldLiteralsRenderUnchanged pins the backward-compatibility promise:
// the pre-migration construction &esperr.Error{Code: X, Name: Y} - no text
// field - renders exactly the code-era formats it always did.
func TestOldLiteralsRenderUnchanged(t *testing.T) {
	if got := (&Error{Code: 116, Name: "ETIMEDOUT"}).Error(); got != "ETIMEDOUT (116)" {
		t.Errorf("old named literal = %q, want %q", got, "ETIMEDOUT (116)")
	}
	if got := (&Error{Code: 29184}).Error(); got != "esp error 29184" {
		t.Errorf("old unnamed literal = %q, want %q", got, "esp error 29184")
	}
}

// TestSentinelPointerIs pins the standard chain: a sentinel returned
// directly (or reached through Wrap) matches itself by pointer identity,
// and two distinct code-free sentinels never match each other.
func TestSentinelPointerIs(t *testing.T) {
	notReady := New("esp/x: not ready")
	busy := New("esp/x: busy")
	if !errors.Is(notReady, notReady) {
		t.Error("errors.Is(sentinel, itself) = false, want true")
	}
	if errors.Is(notReady, busy) {
		t.Error("two distinct code-free sentinels matched - zero-code Is leak")
	}
	if errors.Is(Wrap("x: op", notReady), busy) {
		t.Error("Wrap(sentinel) matched a different sentinel")
	}
	if !errors.Is(Wrap("x: op", notReady), notReady) {
		t.Error("errors.Is(Wrap(op, sentinel), sentinel) = false, want true")
	}
	// A failed walk against a root sentinel must terminate (the sentinel
	// field points at the error itself; a self-UNwrapping Unwrap would
	// loop forever here).
	if errors.Is(notReady, errors.New("foreign")) {
		t.Error("code-free sentinel matched a foreign error")
	}
}

// TestCodeIs pins the code-era compatibility path: an Error built from a
// raw pipeline code matches any *Error carrying the same non-zero code -
// including the transport sentinels, which keep their errnos on Code.
func TestCodeIs(t *testing.T) {
	if !errors.Is(Coded(116, "ETIMEDOUT"), ErrTimeout) {
		t.Error("Coded(116) did not match ErrTimeout by code")
	}
	if !errors.Is(ErrTimeout, Coded(116, "")) {
		t.Error("ErrTimeout did not match Coded(116) by code (Is is directional)")
	}
	if !errors.Is(&Error{Code: 9, Name: "EBADF"}, ErrClosed) {
		t.Error("raw EBADF literal did not match ErrClosed by code")
	}
	if !errors.Is(&Error{Code: metaEOF}, ErrEOF) {
		t.Error("raw metaEOF literal did not match ErrEOF by code")
	}
	if !errors.Is(Coded(11, ""), ErrWouldBlock) {
		t.Error("Coded(11) did not match ErrWouldBlock by code")
	}
	if errors.Is(Coded(116, ""), ErrClosed) {
		t.Error("Coded(116) matched ErrClosed - different codes must not match")
	}
	// The transport sentinels keep matching each other's old literal
	// spellings but not one another.
	if errors.Is(ErrWouldBlock, ErrTimeout) {
		t.Error("ErrWouldBlock matched ErrTimeout")
	}
}

// TestAs retrieves the *Error through the chain, so callers can read the
// code annotation off a wrapped error.
func TestAs(t *testing.T) {
	wrapped := Wrap("i2c: write transaction", Coded(0x107, "ESP_ERR_TIMEOUT"))
	var e *Error
	if !errors.As(wrapped, &e) {
		t.Fatalf("errors.As through Wrap = false, want the *Error")
	}
	if e.Code != 0x107 {
		t.Errorf("Code = %d, want 0x107", e.Code)
	}
	if e.Name != "ESP_ERR_TIMEOUT" {
		t.Errorf("Name = %q, want ESP_ERR_TIMEOUT", e.Name)
	}
	// Through a Wrap around a code-carrying sentinel too.
	var s *Error
	if !errors.As(Wrap("x: op", withCode(New("esp/x: broken"), 0x7201)), &s) || s.Code != 0x7201 {
		t.Errorf("errors.As through Wrap(sentinel-with-code) failed: %v", s)
	}
}

// TestUnwrapChainDepth pins the chain shapes: Error unwraps to nothing
// (forwarded code) or to its root (sentinel), opError unwraps one level,
// and nested Wraps stack prefixes.
func TestUnwrapChainDepth(t *testing.T) {
	inner := Coded(5, "EIO")
	if u := inner.Unwrap(); u != nil {
		t.Errorf("Coded error unwrapped to %v, want nil", u)
	}
	sent := New("esp/x: not ready").(*Error)
	if u := sent.Unwrap(); u != nil {
		t.Errorf("root sentinel unwrapped to %v, want nil (chain ends here)", u)
	}
	w1 := Wrap("x: read", inner).(*opError)
	if u := w1.Unwrap(); u != inner {
		t.Errorf("opError.Unwrap = %v, want the wrapped error", u)
	}
	w2 := Wrap("x: outer", w1).(*opError)
	if got, want := w2.Error(), "x: outer: x: read: EIO (5)"; got != want {
		t.Errorf("nested Wrap Error() = %q, want %q", got, want)
	}
	if u := w2.Unwrap(); u != w1 {
		t.Errorf("outer opError.Unwrap = %v, want the inner opError", u)
	}
	// errors.Is walks any depth of the chain.
	if !errors.Is(w2, inner) {
		t.Error("errors.Is through two Wraps = false, want true")
	}
}

// TestWrapNil pins the nil safety: wrapping a nil error is nil, so a
// `return Wrap(op, err)` tail never invents a failure.
func TestWrapNil(t *testing.T) {
	if err := Wrap("x: op", nil); err != nil {
		t.Errorf("Wrap(op, nil) = %v, want nil", err)
	}
}

// TestCodeFunc pins esperr.Code across every chain shape: nil, direct
// *Error, Wrap(-ed) errors, code-carrying sentinels, code-free sentinels
// (0), and foreign errors (-1).
func TestCodeFunc(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int32
	}{
		{"nil", nil, 0},
		{"direct", Coded(0x7F00, ""), 0x7F00},
		{"named literal", &Error{Code: 0x7401, Name: "X"}, 0x7401},
		{"one Wrap", Wrap("x: read", Coded(11, "EAGAIN")), 11},
		{"two Wraps", Wrap("x: a", Wrap("x: b", Coded(9, "EBADF"))), 9},
		{"transport sentinel", ErrTimeout, 116},
		{"code-free sentinel", New("esp/x: busy"), 0},
		{"sentinel with code", withCode(New("esp/x: broken"), 0x7201), 0x7201},
		{"wrap of code-free sentinel", Wrap("x: op", New("esp/x: busy")), 0},
		{"foreign", errors.New("boom"), -1},
		{"foreign wrapped", Wrap("x: op", errors.New("boom")), -1},
	}
	for _, c := range cases {
		if got := Code(c.err); got != c.want {
			t.Errorf("Code(%s) = %d, want %d", c.name, got, c.want)
		}
	}
}

// TestCodeFreeSentinelsDoNotAlias pins the identity guarantee behind the
// zero-code Is guard: code-free sentinels match nothing but themselves,
// even when compared against each other or against wrapped values.
func TestCodeFreeSentinelsDoNotAlias(t *testing.T) {
	a := New("esp/x: not ready")
	b := New("esp/x: busy")
	if errors.Is(Coded(5, "EIO"), a) || errors.Is(Coded(5, "EIO"), b) {
		t.Error("a coded error matched a code-free sentinel")
	}
	if errors.Is(Wrap("x: op", a), b) {
		t.Error("Wrap(a) matched b through the chain")
	}
}

// withCode attaches a historical pipeline code to a New-built sentinel -
// the pattern the packages use when a condition has a segment value to
// keep inspectable (tests use it for the same reason).
func withCode(err error, code int32) error {
	e, _ := err.(*Error)
	e.Code = code
	return e
}
