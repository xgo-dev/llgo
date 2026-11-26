// Package esperr defines the one error representation shared by every
// pkg/esp package. Before the error-model unification the family carried
// failures as raw int32 codes in three different shapes (a negative errno
// from Write, an (int32, int32) pair from Read, a bare esp_err from the
// peripheral drivers); now every public call reports an error value, and
// this package is the type behind it.
//
// # The Go shape
//
// An Error is text-first, code-annotated: the words a Go user reads come
// first, the raw pipeline number rides along as the diagnostic. Build
// errors only through the three constructors - they are the family's
// whole vocabulary:
//
//   - New(text) declares a package's semantic sentinel (ErrNotReady,
//     ErrBusy, ...): an *Error carrying only the human text, usable as a
//     package-level var and matched by pointer identity through the
//     standard errors.Is chain (it plays the role errors.New plays for
//     the stdlib, without importing it - see the dependency note below).
//   - Coded(code, name) forwards a raw pipeline code (an lwip errno, a
//     DNS h_errno, a negative mbedtls code, an esp_err): no semantic
//     text - a forwarded code is not translatable prose, the number is
//     the diagnostic - so Error() renders the code-era formats.
//   - Wrap(op, err) adds the operation context ("i2c: write transaction")
//     in front of any error, the "op: " prefix of stdlib practice.
//
// A sentinel produced by New carries no code. When a condition also has
// a historical pipeline code (an internal segment value the C glue still
// reports, or a transport sentinel's errno), the declaring package keeps
// that code on the sentinel's exported Code field - the code stays
// inspectable through errors.As and esperr.Code while the text becomes
// the user-facing form.
//
// # Matching
//
// Three matching paths, all standard errors.Is:
//
//   - sentinel pointer: a value unwrapped to the sentinel itself (direct
//     return, or through Wrap) matches by identity;
//   - pipeline code: any *Error matches a sentinel (or literal) with the
//     same non-zero Code, the code-era compatibility path - code built
//     before the sentinels existed keeps matching unchanged;
//   - zero-code errors (New sentinels) never match by code - two
//     code-free sentinels must not alias each other's identity.
//
// errors.As retrieves the *Error to read Code and Name off any of these.
//
// # The four transport sentinels
//
// ErrClosed, ErrTimeout, ErrWouldBlock and ErrEOF are the cross-package
// vocabulary of the transport family (net, tls, ws). They are re-exported
// there (as ErrClosed/ErrTimeout/ErrWouldBlock/ErrEOF) so callers can stay
// inside one package's namespace; they are the same values. The first
// three carry the lwip errnos that have always reported those conditions
// (EBADF 9, ETIMEDOUT 116, EAGAIN 11) on their Code field, so an Error
// built from a raw pipeline code still satisfies errors.Is against them
// by code; ErrEOF carries a meta-segment value (see below) because the
// condition has no C-side code.
//
// The deadline model that produces these conditions is unchanged by the
// error migration: deadlines stay relative milliseconds (time.Time has
// never been validated on the bare-metal target and the bounded-quantum
// model does not need it) - a deliberate, documented deviation from
// stdlib net.Conn.
//
// # Dependency discipline: no imports
//
// The package imports nothing and has no build tags, so every pkg/esp
// package (including the stdlib-free ws) can depend on it on every build,
// hosted or target, without dragging in glue or stdlib image weight. That
// rule now explicitly covers the stdlib errors and fmt packages too: the
// errors.Is/As machinery lives in the caller (tests, application code on
// the host side), not here, and New's comparable sentinel is a pointer to
// this package's own Error value (a few lines, equivalent to what
// errors.New hands back); Wrap renders by string concatenation, not
// fmt.Errorf. Everything this package needs to speak the standard error
// conventions - Error, Unwrap, Is - is methods on its own type.
package esperr

// The transport-family sentinels: semantic text first, the historical
// pipeline code kept on Code as the annotation (the errnos above), so the
// old code-based matching against raw-errno errors keeps working. Re-
// exported by net, tls and ws (as ErrClosed/ErrTimeout/ErrWouldBlock/ErrEOF).
var (
	// ErrClosed reports operating on a closed (or nil) connection: a
	// second Close, or any call after one. The code is EBADF (9), the
	// value the transports always reported for the condition; unlike the
	// raw-errno era the sentinel makes it testable (errors.Is) instead of
	// comparable only as a magic number.
	ErrClosed = newSentinel("closed", 9, "EBADF")
	// ErrTimeout reports a deadline that passed with the operation
	// unfinished (lwip ETIMEDOUT, 116).
	ErrTimeout = newSentinel("timeout", 116, "ETIMEDOUT")
	// ErrWouldBlock reports the bounded model's poll-style give-up: a
	// read (or accept) with no deadline set that found nothing within one
	// 20 ms quantum, or a write the socket stopped accepting within one
	// quantum (lwip EAGAIN/EWOULDBLOCK, 11). Only possible without a
	// deadline; with one, the same stall ends in ErrTimeout instead.
	ErrWouldBlock = newSentinel("would block", 11, "EAGAIN")
	// ErrEOF reports an orderly end of the byte stream: the peer closed a
	// TCP/TLS connection cleanly (net/tls Read). It is the family's
	// counterpart of stdlib io.EOF, kept in-family because importing the
	// stdlib io package would drag the errors/reflect machinery into the
	// bare-metal image (the pkg/esp image-size rule; the ws package
	// documents the history). It satisfies errors.Is within the family,
	// and io.Reader-shaped stdlib consumers (bufio, io.Copy) work
	// unchanged - they stop on any non-nil error.
	ErrEOF = newSentinel("EOF", metaEOF, "EOF")
)

// The esperr meta segment: codes for Go-level conditions that have no
// C-side pipeline code. 0x7C00 sits below the ws (0x7Dxx) and tls (0x7Exx)
// internal segments and outside the lwip errno, h_errno and esp_err
// ranges.
const (
	metaEOF = 0x7C01
)

// newSentinel builds a transport sentinel: text first, the historical
// code and ErrName-style name kept as the annotation. Package-internal -
// other packages declare their sentinels with New (and set Code when the
// condition has a historical segment value; see the package doc).
func newSentinel(text string, code int32, name string) *Error {
	e := &Error{Code: code, Name: name, text: text}
	e.sentinel = e // the "this is a semantic root" mark
	return e
}

// Error is the error value every pkg/esp package returns. Build it through
// New / Coded / Wrap (see the package doc), or match existing values with
// errors.Is / errors.As against the sentinels. The zero fields Code and
// Name stay exported and writable: a package that must keep a historical
// pipeline code on a New-built sentinel sets Error.Code directly.
type Error struct {
	// Code is the raw pipeline code (lwip errno, h_errno, negative
	// mbedtls code, esp_err, or a package-internal 0x7Dxx/0x7Exx value),
	// kept as the diagnostic annotation. 0 means the error carries no
	// pipeline code (a New-built sentinel for a purely Go-level
	// condition).
	Code int32
	// Name is the code's ErrName-style name, or "" when the producer has
	// no name table entry. It appears only in the code-era rendering
	// (below); a text-first error shows its text instead.
	Name string
	// text is the human-readable form (a sentinel's semantic text).
	// Empty on a forwarded-code error, which renders the code-era way.
	text string
	// sentinel marks the semantic root of an error chain: New sets it to
	// the error itself, everything else leaves it nil (Unwrap ends the
	// chain there).
	sentinel error
}

// Error renders the human text first when the error carries one ("text",
// plus " (code N)" when a pipeline code rides along), falling back to the
// code-era formats for forwarded codes: "NAME (CODE)" with a name from the
// producer's ErrName table, "esp error CODE" without. The decimal
// formatting is this package's own itoa - no strconv (the import rule).
func (e *Error) Error() string {
	if e.text != "" {
		if e.Code != 0 {
			return e.text + " (code " + itoa(e.Code) + ")"
		}
		return e.text
	}
	if e.Name != "" {
		return e.Name + " (" + itoa(e.Code) + ")"
	}
	return "esp error " + itoa(e.Code)
}

// Unwrap exposes the sentinel an error carries, so errors.Is walks the
// standard chain to it. A New-built sentinel is its own root (the sentinel
// field points at the error itself); unwrapping a root returns nil, so a
// failed errors.Is walk terminates instead of chasing its own tail.
func (e *Error) Unwrap() error {
	if s, ok := e.sentinel.(*Error); ok && s == e {
		return nil
	}
	return e.sentinel
}

// Is makes errors.Is match by pipeline code, so an Error built anywhere
// from the raw code satisfies errors.Is against the sentinel that carries
// it (ErrWouldBlock against EAGAIN 11, and so on) without a wrapping step.
// Zero never matches: a code-free New sentinel has no code to match on,
// and letting 0 == 0 through would make every such sentinel alias every
// other one.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok || e.Code == 0 || t.Code == 0 {
		return false
	}
	return e.Code == t.Code
}

// New declares a semantic sentinel: an *Error carrying only the human
// text, with no pipeline code. Use it for a package's internal conditions
// (ErrNotReady, ErrBusy, ...), assigned to a package-level var - the
// returned pointer is the errors.Is target (comparable by identity, the
// role errors.New plays without the import). A condition that also has a
// historical pipeline code keeps it on the sentinel's Code field; see the
// package doc.
func New(text string) error {
	e := &Error{text: text}
	e.sentinel = e // the "this is a semantic root" mark
	return e
}

// Coded forwards a raw pipeline code (lwip errno, esp_err, negative
// mbedtls code, h_errno) as the error value a binding returns (nil is the
// caller's job: a zero code means ESP_OK/success and never becomes an
// error). A forwarded code is not translatable prose - the number is the
// diagnostic - so there is no semantic text and Error() renders the
// code-era formats: "NAME (N)" with a name, "esp error N" without.
func Coded(code int32, name string) *Error {
	return &Error{Code: code, Name: name}
}

// Wrap adds the operation context op ("i2c: write transaction": package
// prefix + operation, lowercase) in front of err. The result renders
// "op: err" and unwraps to err, so errors.Is / errors.As / esperr.Code all
// walk through it. A nil err wraps to nil, so Wrap is safe on the success
// path of a `return Wrap(op, err)` chain.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &opError{op: op, next: err}
}

// opError is Wrap's value: an operation prefix around the next error in
// the chain. Unexported - errors cross this package's boundary as error.
type opError struct {
	op   string
	next error
}

func (e *opError) Error() string { return e.op + ": " + e.next.Error() }

func (e *opError) Unwrap() error { return e.next }

// Code returns the pipeline code an error carries: 0 for nil, the code of
// the *Error it is or wraps (through opError prefixes and sentinel
// chains), and -1 for anything else (a foreign error has no esp pipeline
// code; callers that must tell one apart reliably should type-assert
// *Error instead). A New-built sentinel without an attached code reports
// 0 - the same value as success; errors that mean failure always carry a
// non-zero code or a sentinel identity of their own.
func Code(err error) int32 {
	for hop := 0; err != nil && hop < 3; hop++ {
		if e, ok := err.(*Error); ok {
			return e.Code
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	if err == nil {
		return 0
	}
	return -1
}

// itoa formats v in decimal (negatives with a leading minus). No strconv:
// the package stays import-free.
func itoa(v int32) string {
	neg := v < 0
	x := uint32(v)
	if neg {
		x = -uint32(v)
	}
	if x == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
