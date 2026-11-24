// Package esperr defines the one error representation shared by every
// pkg/esp package. Before the error-model unification the family carried
// failures as raw int32 codes in three different shapes (a negative errno
// from Write, an (int32, int32) pair from Read, a bare esp_err from the
// peripheral drivers); now every public call reports an error value, and
// this package is the type behind it.
//
// An Error carries two fields:
//
//   - Code is the raw pipeline code, unchanged from the pre-error era:
//     an lwip errno (the net/tls/ws transports; picolibc numbering),
//     a DNS h_errno (Resolve), a negative mbedtls code (the tls setup
//     and record phases), an esp_err (the peripheral and wifi drivers),
//     or one of the package-internal segments (0x7Dxx ws, 0x7Exx tls).
//     Code stays inspectable, so existing code that matched numeric
//     values (or named them through net.ErrName, tls.ErrName, ws.ErrName,
//     wifi.ErrName) keeps working unchanged.
//   - Name is the human-readable name the producing package's ErrName
//     table gives the code ("EAGAIN", "TLS_ERR_CERT_VERIFY",
//     "ESP_ERR_INVALID_ARG", ...), filled in when the package builds the
//     error. Error() renders "NAME (CODE)", or "esp error CODE" when no
//     name was attached.
//
// The sentinels below are the cross-package vocabulary of the transport
// family (net, tls, ws): ErrWouldBlock, ErrTimeout, ErrClosed and ErrEOF.
// The first three carry the lwip errno values that have always reported
// those conditions (EAGAIN 11, ETIMEDOUT 116, EBADF 9), so an Error built
// from a raw pipeline code satisfies errors.Is against the sentinel with
// no wrapping step - Is matches by code; ErrEOF carries a meta-segment
// value (see below) because the condition has no C-side code.
//
// The deadline model that produces these conditions is unchanged by the
// error migration: deadlines stay relative milliseconds (time.Time has
// never been validated on the bare-metal target and the bounded-quantum
// model does not need it) - a deliberate, documented deviation from
// stdlib net.Conn.
//
// The package imports nothing and has no build tags, so every pkg/esp
// package (including the stdlib-free ws) can depend on it on every build,
// hosted or target, without dragging in glue or stdlib image weight.
package esperr

// The transport-family sentinels. Re-exported by net, tls and ws (as
// ErrClosed/ErrTimeout/ErrWouldBlock/ErrEOF) so callers can stay inside one
// package's namespace; they are the same values.
var (
	// ErrClosed reports operating on a closed (or nil) connection: a
	// second Close, or any call after one. The code is EBADF (9), the
	// value the transports always reported for the condition; unlike the
	// raw-errno era the sentinel makes it testable (errors.Is) instead of
	// comparable only as a magic number.
	ErrClosed = &Error{Code: 9, Name: "EBADF"}
	// ErrTimeout reports a deadline that passed with the operation
	// unfinished (lwip ETIMEDOUT, 116).
	ErrTimeout = &Error{Code: 116, Name: "ETIMEDOUT"}
	// ErrWouldBlock reports the bounded model's poll-style give-up: a
	// read (or accept) with no deadline set that found nothing within one
	// 20 ms quantum, or a write the socket stopped accepting within one
	// quantum (lwip EAGAIN/EWOULDBLOCK, 11). Only possible without a
	// deadline; with one, the same stall ends in ErrTimeout instead.
	ErrWouldBlock = &Error{Code: 11, Name: "EAGAIN"}
	// ErrEOF reports an orderly end of the byte stream: the peer closed a
	// TCP/TLS connection cleanly (net/tls Read). It is the family's
	// counterpart of stdlib io.EOF, kept in-family because importing the
	// stdlib io package would drag the errors/reflect machinery into the
	// bare-metal image (the pkg/esp image-size rule; the ws package
	// documents the history). It satisfies errors.Is within the family,
	// and io.Reader-shaped stdlib consumers (bufio, io.Copy) work
	// unchanged - they stop on any non-nil error.
	ErrEOF = &Error{Code: metaEOF, Name: "EOF"}
)

// The esperr meta segment: codes for Go-level conditions that have no
// C-side pipeline code. 0x7C00 sits below the ws (0x7Dxx) and tls (0x7Exx)
// internal segments and outside the lwip errno, h_errno and esp_err
// ranges.
const (
	metaEOF = 0x7C01
)

// Error is the error value every pkg/esp package returns. Construct it
// directly (packages do, filling Name from their ErrName table) or match
// with errors.Is against the sentinels.
type Error struct {
	// Code is the raw pipeline code (lwip errno, h_errno, negative
	// mbedtls code, esp_err, or a package-internal 0x7Dxx/0x7Exx value).
	Code int32
	// Name is the code's ErrName-style name, or "" when the producer has
	// no name table entry (the peripheral packages leave it empty; the
	// code itself stays the stable identifier).
	Name string
}

// Error renders "NAME (CODE)", or "esp error CODE" without a name.
func (e *Error) Error() string {
	if e.Name != "" {
		return e.Name + " (" + itoa(e.Code) + ")"
	}
	return "esp error " + itoa(e.Code)
}

// Is makes errors.Is match by pipeline code, so an Error built anywhere
// from the raw code satisfies errors.Is against the sentinel that carries
// it (ErrWouldBlock against EAGAIN 11, and so on) without a wrapping step.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.Code == t.Code
}

// Code returns the pipeline code an error carries: 0 for nil, the code of
// an *Error it is or wraps, and -1 for anything else (a foreign error has
// no esp pipeline code; callers that must tell one apart reliably should
// type-assert *Error instead).
func Code(err error) int32 {
	for hop := 0; err != nil && hop < 2; hop++ {
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
