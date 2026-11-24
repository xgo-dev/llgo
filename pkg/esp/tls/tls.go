// Package tls binds llgo-compiled Go code to the ESP-IDF mbedtls TLS stack
// (mbedtls 4.1.1 in IDF v6.3-dev), giving a connected TCP socket (see the
// net package) a record layer: Dial connects, handshakes and verifies the
// peer (Config), then the connection exchanges plaintext through Read and
// Write while mbedtls encrypts and authenticates every byte.
//
// # Layout
//
// Same template as pkg/esp/net (glue lives in a subdirectory because the go
// tool rejects C sources inside a Go package):
//
//	tls.go          public API, no build tag
//	tls_idf.go      //go:build llgo.idf - bodyless //go:linkname externs
//	tls_stub.go     //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/tls_glue.c ESP-IDF C glue template, compiled only under ESP_PLATFORM
//
// The package is deliberately dependency-free (only stdlib/unsafe): the TCP
// primitives it needs (socket, connect, resolve, poll, shutdown, close,
// uptime clock) are the C symbols net_glue.c already defines, declared here
// again by linkname - so the glue set a TLS program links is tls_glue.c
// PLUS net_glue.c, without the Go package importing pkg/esp/net.
//
// # Usage
//
// After the wifi package reports EvGotIP (and, on a host peer, that peer is
// listening):
//
//	c, err := tls.Dial("192.168.4.1:9443", 10000,
//	    &tls.Config{ServerName: "device.internal", RootCAs: caPEM})
//	if err != nil { ... }
//	n, err := c.Write(msg)
//	n, err = c.Read(buf)
//	st := c.State() // negotiated version, cipher suite, peer certificate DER
//	err = c.Close()
//
// The ServerName is what the handshake verifies the server certificate
// against (SNI + CN/SAN match) and may differ from the dial address - the
// connection can target a NAT alias or load balancer while naming the
// service behind it. Empty ServerName means the host part of addr is the
// name.
//
// TLSDial(ip, port, timeoutMs, caPEM) is the pre-Config form, kept as a
// deprecated alias: its caPEM==nil behavior is VERIFY_NONE and its
// caPEM!=nil behavior verifies the chain and validity dates but NOT the
// host name (no SNI, no CN/SAN match) - the caller's identity check is its
// trust in the CA. New code should use Dial.
//
// # The bounded-quantum model (inherited from net)
//
// llgo's fiber scheduler runs on one FreeRTOS task, so a call that blocks in
// C would freeze the whole Go scheduler and starve the watchdog. The glue
// keeps the TCP socket non-blocking and every operation is a Go-side retry
// loop: one non-blocking attempt (one handshake step, one record
// read/write, one close_notify), then a 20 ms poll quantum on the socket -
// a wait that parks the FreeRTOS task on lwip's select semaphore and feeds
// the idle tasks - then a deadline check. The mbedtls layer reports what it
// wants through MBEDTLS_ERR_SSL_WANT_READ/WANT_WRITE and this package maps
// those to the socket quanta, so TLS 1.2 and 1.3 handshakes advance at the
// same pace the raw TCP operations do. Timeouts are accurate to within one
// quantum. There is no unbounded wait anywhere in the path.
//
// # Deadlines
//
// Dial takes an explicit timeoutMs (<= 0 is rejected with EINVAL - the
// model requires bounded waits) that bounds BOTH the TCP connect and the
// handshake as one total budget. Read and Write honor the deadline set by
// SetDeadline (relative ms; 0 = no deadline), with net.Conn semantics:
// with no deadline a Read gives up after one quantum with ErrWouldBlock
// and a Write reports (0, ErrWouldBlock) as soon as one quantum passes
// without progress; with a deadline both retry until it passes (Read then
// reports ErrTimeout, Write (0, ErrTimeout)).
//
// Deadlines are relative milliseconds - NOT stdlib's time.Time, a
// deliberate deviation (time.Time has never been validated on the
// bare-metal target and the bounded-quantum model does not need
// wall-clock arithmetic); everything else follows the io.Reader/io.Writer
// and net.Conn conventions.
//
// One semantic difference from raw TCP, inherent to TLS: Write accepts a
// buffer all-or-nothing. mbedtls encrypts the whole plaintext into one
// record (it never takes a partial plaintext), so a Write that could not
// complete has consumed NOTHING and the caller retries the whole buffer -
// there are no short positive writes in this package.
//
// # Certificates and host names
//
// Dial's Config chooses the verification mode, mirroring crypto/tls:
//
//   - RootCAs set (PEM): the trust anchor for MBEDTLS_SSL_VERIFY_REQUIRED.
//     It may hold one CA certificate or a chain of concatenated PEM blocks
//     (mbedtls parses them all; any failed member rejects the dial with
//     CaParseFailed before any bytes hit the network) - a single-CA PEM is
//     the documented and tested path. The server chain must verify against
//     that anchor, including validity dates, AND the ServerName must match
//     the certificate's CN/SAN entries (IP literals match IP SANs, names
//     match dNSName SANs); either failure ends the dial with
//     CertVerifyFailed.
//   - InsecureSkipVerify: MBEDTLS_SSL_VERIFY_NONE - the handshake runs and
//     the session key is still authenticated (an active MITM cannot read
//     the traffic), but the peer's identity is not checked. A RootCAs set
//     alongside is ignored, like crypto/tls.
//   - Anything else - including the zero Config - is REJECTED with
//     NoRootCAs before the dial touches the network. This is the
//     fail-closed rule: a bare-metal target has no system trust store, so
//     "no RootCAs given" cannot mean "use the system defaults"; it means
//     the caller forgot the identity check, and silent VERIFY_NONE (the
//     pre-Config default) would let a MITM holding any certificate of the
//     pinned CA through. Saying InsecureSkipVerify=true must stay the one
//     explicit way to dial without identity checking.
//
// MinVersion (VersionTLS12 / VersionTLS13; 0 = the mbedtls default) floors
// the negotiated protocol version.
//
// The deprecated TLSDial keeps its two legacy modes unchanged: caPEM==nil
// is VERIFY_NONE (today the SkipVerify shape; kept for the deterministic
// emulator acceptance run), caPEM!=nil is VERIFY_REQUIRED against that CA
// with NO host name check (the glue calls mbedtls_ssl_set_hostname(NULL),
// the explicit opt-out mbedtls demands before it allows certificate
// verification on a client without a hostname
// (MBEDTLS_ERR_SSL_CERTIFICATE_VERIFICATION_WITHOUT_HOSTNAME, -0x5D80);
// chain + validity are checked, the CN/SAN match is dropped). Its doc
// carries the warning; it exists so existing callers keep compiling while
// they migrate to Dial.
//
// ConnectionState reports what the session ended up with: Version and
// CipherSuite as mbedtls's own names ("TLSv1.3", "TLS1-3-AES-128-GCM-
// SHA256"), and PeerCerts with the peer certificate's DER encoding.
// mbedtls retains only the peer's LEAF certificate (the rest of the chain
// was verified and released), so PeerCerts holds one entry; it is stored
// regardless of authmode, so a VERIFY_NONE session can still be audited or
// pinned through it. State() copies everything out of mbedtls-owned memory,
// so the result stays valid across Close.
//
// The RNG behind the handshake is PSA Crypto random: mbedtls 4.1 wires the
// SSL layer to psa_generate_random and IDF v6.3-dev initializes PSA at
// startup (and feeds it from the hardware TRNG through esp_fill_random).
// The ctr_drbg of older mbedtls is a private header in 4.1.1, hence the
// PSA path. Under esp-emu the TRNG register reads back deterministic
// values, so emulated handshakes are reproducible (fine for acceptance;
// on hardware this is a real TRNG).
//
// # Error model
//
// Errors are Go error values of the shared pkg/esp/esperr type: Code is
// the raw pipeline code, Name its ErrName rendering. The handshake-phase
// and dial-policy failures have first-class sentinels - errors.Is can
// tell CertVerifyFailed (chain or CN/SAN mismatch), CaParseFailed,
// HandshakeTimeout, NoRootCAs, Handshake and friends apart from each
// other and from the transport sentinels ErrClosed/ErrTimeout/
// ErrWouldBlock. Three pass-through conventions on top, matching net:
//
//   - TCP-phase failures (socket, connect) surface as the raw lwip errno,
//     exactly as net.Dial reports them - ErrName names those values too.
//   - Dial failures inside mbedtls's own setup (config defaults, ssl_setup,
//     the rest of the glue's llgo_tls_new) pass through the raw negative
//     mbedtls code; the only 0x7E00 codes that phase produces are its own
//     CaParseFailed and PsaInitFailed. The Handshake code exists only on
//     the handshake phase, where the dial loop maps the codes its table
//     does not act on to it.
//   - Record-phase failures after the handshake surface as the raw
//     negative mbedtls code (e.g. -0x7780 fatal alert); mbedtls treats
//     such an error as fatal for the session, and so should the caller.
//     ErrName names the common ones. An orderly session end is ErrEOF
//     (close_notify arrived, or the TCP stream closed cleanly without
//     one) - the family's io.EOF counterpart, matching net.Conn.
//
// The 0x7E00 segment itself is now internal: the values live only inside
// this package (ErrName's table and the sentinel definitions); callers
// match on the sentinel errors.
//
// # Ownership
//
// The glue allocates one C-side context per connection (the mbedtls state:
// ssl context, ssl config, parsed CA) and hands it to Go as an opaque
// handle stored in a uintptr field - so the GC never scans it as a
// pointer. Close frees the context and the fd exactly once and is
// idempotent: a second Close, or any call on a nil or closed conn, reports
// ErrClosed. A failed dial frees everything before returning, so a failed
// dial leaks nothing.
//
// # Scheduling
//
// Same architecture property as the net package: the quantum waits are
// C-level select waits that give the llgo scheduler no cooperative yield
// point, so other goroutines do not run while one goroutine sits inside a
// TLS wait. Keep a TLS session on one goroutine, or hand control between
// fibers through channel operations at chunk boundaries - a channel
// send/receive is a real scheduler yield.
//
// # Glue integration
//
// The C glue is a template: copy glue/tls_glue.c into your project's main
// component (WHOLE_ARCHIVE required - see pkg/esp/README.md for both
// integration recipes and the boundary rules). A TLS program links BOTH
// glues: tls_glue.c needs REQUIRES mbedtls; the shared socket/clock
// primitives come from net_glue.c (REQUIRES lwip esp_timer).
package tls

import (
	"unsafe" // cString scans and copies the mbedtls-owned state strings

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// The default poll quantum: how long every retry loop waits for the socket
// between attempts. Same value the net package uses; one wait is long
// enough to keep the watchdog fed and short enough that timeout accuracy
// stays well under a tenth of a second.
const quantumMs = 20

// TLSConn is a connected TLS session. The C-side mbedtls context travels
// as an opaque uintptr handle (the GC must never scan it as a pointer),
// and fd is the underlying non-blocking TCP socket the quantum polls use.
// The deadline is an absolute uptime-ms value plus an explicit active flag:
// the flag - not a 0 sentinel - says whether a deadline is set, so a
// deadline that lands exactly on the 32-bit counter wrap (0) is still
// honored.
type TLSConn struct {
	h           uintptr
	fd          int32
	deadlineMs  uint32 // absolute uptime-ms deadline, meaningful when hasDeadline
	hasDeadline bool
}

// mbedtls error codes this package acts on by value. Values pinned from the
// mbedtls 4.1.1 headers IDF v6.3-dev ships (mbedtls/ssl.h, mbedtls/x509.h);
// the WANT pair changed sign value from mbedtls 3.x (-0x6880/-0x6890 there,
// -0x6900/-0x6880 here - WANT_WRITE sorts before WANT_READ).
const (
	mbErrWantRead         = -0x6900 // record layer wants more input
	mbErrWantWrite        = -0x6880 // record layer wants the socket writable
	mbErrPeerCloseNotify  = -0x7880 // peer sent the TLS close notification
	mbErrConnEOF          = -0x7280 // TCP closed cleanly without close_notify
	mbErrCertVerifyFailed = -0x2700 // server chain failed x509 verification
)

// TLS-layer error codes, the internal 0x7E00 segment (outside the lwip
// errno range, the DNS h_errno range and the esp_err range). Callers match
// on the sentinel errors below; the values themselves stay inside this
// package (ErrName's table and the sentinels).
const (
	tlsHandshakeTimeout  = 0x7E01
	tlsCertVerifyFailed  = 0x7E02
	tlsCaParseFailed     = 0x7E03
	tlsWantPollExhausted = 0x7E04
	tlsHandshakeFailure  = 0x7E05
	tlsPsaInitFailed     = 0x7E06
	tlsNoRootCAs         = 0x7E07
	tlsNoPeerCert        = 0x7E08
)

// Handshake-phase and dial-policy failures as sentinel errors: errors.Is
// tells the classes apart (CertVerifyFailed covers both the chain-verify
// failure and the ServerName CN/SAN mismatch; mbedtls reports both as
// X509_CERT_VERIFY_FAILED). See the package doc's error-model section for
// the pass-through conventions (lwip errnos from the TCP phase, raw
// negative mbedtls codes from the setup and record phases).
var (
	// ErrHandshakeTimeout: the handshake was still only making WANT progress
	// when the dial's total budget (timeoutMs) ran out. The TCP layer was
	// fine; the peer was too slow or the network too lossy.
	ErrHandshakeTimeout = &esperr.Error{Code: tlsHandshakeTimeout,
		Name: ErrName(tlsHandshakeTimeout)}
	// ErrCertVerifyFailed: VERIFY_REQUIRED and the server failed
	// verification: the chain does not verify against the CA (validity
	// included), or the ServerName does not match the certificate's CN/SAN
	// entries.
	ErrCertVerifyFailed = &esperr.Error{Code: tlsCertVerifyFailed,
		Name: ErrName(tlsCertVerifyFailed)}
	// ErrCaParseFailed: the CA PEM did not yield a usable certificate (hard
	// parse error, or mbedtls reported a failed member in the PEM - any
	// failed member rejects the whole parameter).
	ErrCaParseFailed = &esperr.Error{Code: tlsCaParseFailed,
		Name: ErrName(tlsCaParseFailed)}
	// ErrWantPollExhausted: Close's bounded close_notify wait ended before
	// the peer exchange completed. The fd and the context are released
	// regardless - Close never fails to clean up; the error only reports
	// that the graceful part did not finish.
	ErrWantPollExhausted = &esperr.Error{Code: tlsWantPollExhausted,
		Name: ErrName(tlsWantPollExhausted)}
	// ErrHandshake: any other fatal mbedtls error during the handshake
	// (protocol failure, alert from the peer, out of memory) - the dial
	// loop maps codes its table does not act on to this one. The setup
	// phase never produces it: there the raw negative mbedtls code is
	// passed through (see the error-model section above).
	ErrHandshake = &esperr.Error{Code: tlsHandshakeFailure,
		Name: ErrName(tlsHandshakeFailure)}
	// ErrPsaInitFailed: the glue's defensive psa_crypto_init call failed -
	// the handshake has no random source. IDF initializes PSA at boot, so
	// reaching this means the crypto subsystem itself is broken.
	ErrPsaInitFailed = &esperr.Error{Code: tlsPsaInitFailed,
		Name: ErrName(tlsPsaInitFailed)}
	// ErrNoRootCAs: the fail-closed dial. RootCAs was nil and
	// InsecureSkipVerify false, so there is no way to authenticate the
	// peer's identity - a bare-metal target has no system trust store to
	// fall back on, and the pre-Config default (silent VERIFY_NONE) is
	// exactly the hole a MITM with any certificate of the pinned CA would
	// walk through. Raised before the dial touches the network; the glue
	// carries the same refusal as a defense-in-depth check.
	ErrNoRootCAs = &esperr.Error{Code: tlsNoRootCAs,
		Name: ErrName(tlsNoRootCAs)}
	// ErrNoPeerCert: State() on a session that carries no peer certificate
	// - the handshake is not over (or the conn is already closed), or the
	// peer never sent one. (State reports the zero ConnectionState rather
	// than the error; the sentinel exists so the code stays nameable.)
	ErrNoPeerCert = &esperr.Error{Code: tlsNoPeerCert,
		Name: ErrName(tlsNoPeerCert)}
)

// The transport error sentinels, re-exported from the shared esperr base
// so callers can stay inside this package's namespace.
var (
	ErrClosed     = esperr.ErrClosed
	ErrTimeout    = esperr.ErrTimeout
	ErrWouldBlock = esperr.ErrWouldBlock
	ErrEOF        = esperr.ErrEOF
)

// errOf converts a pipeline code to the error the API returns (nil for 0).
// The sentinel codes map to their sentinel values; anything else (lwip
// errnos, negative mbedtls codes) becomes an *esperr.Error named through
// ErrName.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case ebadf:
		return ErrClosed
	case eagain:
		return ErrWouldBlock
	case etimedout:
		return ErrTimeout
	case tlsHandshakeTimeout:
		return ErrHandshakeTimeout
	case tlsCertVerifyFailed:
		return ErrCertVerifyFailed
	case tlsCaParseFailed:
		return ErrCaParseFailed
	case tlsWantPollExhausted:
		return ErrWantPollExhausted
	case tlsHandshakeFailure:
		return ErrHandshake
	case tlsPsaInitFailed:
		return ErrPsaInitFailed
	case tlsNoRootCAs:
		return ErrNoRootCAs
	case tlsNoPeerCert:
		return ErrNoPeerCert
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// Protocol versions for Config.MinVersion. The values are the TLS wire
// versions, which is both what crypto/tls's VersionTLS12/VersionTLS13 use
// and what mbedtls's mbedtls_ssl_protocol_version enum uses - one number
// serves all three.
const (
	VersionTLS12 uint16 = 0x0303
	VersionTLS13 uint16 = 0x0304
)

// Config is the dial-time TLS policy, the crypto/tls.Config fields a
// bare-metal client needs. The zero value dials NOTHING (NoRootCAs): a
// bare-metal target has no system trust store, so verification without an
// explicit trust anchor cannot succeed and must not silently degrade to
// VERIFY_NONE - set RootCAs to verify, or InsecureSkipVerify to state
// explicitly that identity checking is not wanted.
type Config struct {
	// ServerName is the host name the server certificate must match (the
	// SNI extension carries it too). Empty means the host part of the
	// addr passed to Dial. It exists separately from the dial address so
	// a NAT alias or load balancer can be targeted while naming the
	// service behind it - exactly this demo's acceptance shape: the
	// firmware dials the NAT alias 192.168.4.1:9443 and names the service
	// behind it.
	ServerName string
	// RootCAs is the PEM-encoded CA certificate (or chain of concatenated
	// PEM blocks) to verify the server chain against. Parsed with the
	// same x509_crt_parse path TLSDial uses; any failed member rejects the
	// dial with CaParseFailed.
	RootCAs []byte
	// InsecureSkipVerify disables ALL peer identity checking
	// (MBEDTLS_SSL_VERIFY_NONE): any certificate is accepted. RootCAs is
	// ignored when this is set, like crypto/tls.
	InsecureSkipVerify bool
	// MinVersion floors the negotiated protocol version (0 = the mbedtls
	// default; only VersionTLS12/VersionTLS13 are accepted, anything else
	// is EINVAL).
	MinVersion uint16
}

// ConnectionState describes one established TLS session, the
// crypto/tls.ConnectionState fields a record-layer client can act on.
// Version and CipherSuite are mbedtls's own names for what was negotiated
// (e.g. "TLSv1.3", "TLS1-3-AES-128-GCM-SHA256"); PeerCerts holds the peer
// certificate's DER encoding - mbedtls retains only the peer's leaf
// certificate, so it has exactly one entry after a completed handshake
// (stored regardless of authmode, so a VERIFY_NONE session can still be
// audited or pinned). One project-side requirement: IDF's mbedtls Kconfig
// defaults to MBEDTLS_SSL_KEEP_PEER_CERTIFICATE=n - the certificate is
// released right after verification (~4 kB heap saved per session) and
// PeerCerts comes back empty. Set CONFIG_MBEDTLS_SSL_KEEP_PEER_CERTIFICATE=y
// (the demo's sdkconfig.defaults does) to have State export the DER.
// Everything is copied out of mbedtls-owned memory
// at State() time and stays valid across Close.
type ConnectionState struct {
	Version     string
	CipherSuite string
	PeerCerts   [][]byte
}

// lwip errno values the TCP phase can surface, same values the net package
// pins (picolibc numbering - see net.go for the Linux divergences). They
// are redeclared here because the packages stay mutually dependency-free.
const (
	ebadf        = 9   // operation on a closed (or nil) connection
	eagain       = 11  // EWOULDBLOCK: no deadline set, nothing in one quantum
	einval       = 22  // invalid argument
	enametoolong = 91  // name longer than a DNS name may be
	econnreset   = 104 // peer reset the connection
	econnrefused = 111 // nothing listening at the far end
	eaddrinuse   = 112 // local address/port already in use
	enetunreach  = 114 // no route (typically: no IP yet)
	etimedout    = 116 // deadline (or TCP retransmit limit) passed
	ehostunreach = 118 // far host unreachable
	einprogress  = 119 // connect started, still in flight
	ealready     = 120 // connect already in flight
	shutRdwr     = 2   // SHUT_RDWR: both directions
)

// closeBudgetMs bounds Close's close_notify exchange: long enough for a
// live peer to answer within one quantum turn, short enough that Close
// stays a bounded call even against a vanished peer.
const closeBudgetMs = 500

// Dial connects a TCP socket to the host:port in addr, then runs the TLS
// client handshake with cfg's verification policy, all within timeoutMs
// milliseconds (<= 0 is rejected with EINVAL - the bounded-quantum model
// rejects waits without an explicit deadline).
//
// addr is "host:port"; host is a dotted-quad IP (parsed by lwip, no DNS
// involved) or a name (resolved through the lwip DNS client - the same
// Resolve-freeze caveat the net package documents). cfg may be nil only in
// combination with InsecureSkipVerify semantics - a nil cfg is the zero
// Config and REJECTED with NoRootCAs (see the package doc's fail-closed
// rule); there is no way to dial without naming a verification policy.
// Empty cfg.ServerName means the addr host is the verified name.
//
// On every failure path the fd is closed and the C context freed before
// Dial returns, so a failed dial leaks nothing. Errors: EINVAL for a bad
// addr or MinVersion, ENAMETOOLONG for a name over 255 bytes, ErrNoRootCAs
// for the missing trust anchor, the resolver's h_errno error for an
// unresolvable name, a lwip errno from the TCP phase (same as net.Dial),
// one of the 0x7E00-class sentinels from the phases above, or
// ErrHandshake for unmapped mbedtls failures.
func Dial(addr string, timeoutMs int32, cfg *Config) (*TLSConn, error) {
	if timeoutMs <= 0 {
		return nil, errOf(einval)
	}
	host, port, e := splitHostPort(addr)
	if e != 0 {
		return nil, errOf(e)
	}
	var cfgv Config
	if cfg != nil {
		cfgv = *cfg
	}
	if cfgv.MinVersion != 0 && cfgv.MinVersion != VersionTLS12 && cfgv.MinVersion != VersionTLS13 {
		return nil, errOf(einval)
	}
	if cfgv.RootCAs == nil && !cfgv.InsecureSkipVerify {
		return nil, ErrNoRootCAs
	}
	serverName := host
	if cfgv.ServerName != "" {
		serverName = cfgv.ServerName
	}
	if len(serverName) > 255 {
		return nil, errOf(enametoolong)
	}
	ip, e := resolveHost(host)
	if e != 0 {
		return nil, errOf(e)
	}
	return dialIP(ip, port, timeoutMs, serverName, cfgv.RootCAs,
		cfgv.InsecureSkipVerify, cfgv.MinVersion)
}

// TLSDial is the pre-Config dial form, kept as a deprecated alias for
// existing callers (the demo's ws probe included); new code should use
// Dial.
//
// WARNING: neither mode verifies the server's host name, so neither is
// proof against a MITM holding any certificate the CA signed - only
// tls.Dial with a ServerName adds the identity check.
//
//   - caPEM == nil: VERIFY_NONE - the session key is authenticated, the
//     peer's identity is not checked.
//   - caPEM != nil: VERIFY_REQUIRED against that PEM (same parse rules
//     and codes as Config.RootCAs), with mbedtls_ssl_set_hostname(NULL):
//     the chain and validity dates are checked, the CN/SAN match is not.
//
// Deprecated: use Dial with a Config (ServerName + RootCAs, or
// InsecureSkipVerify for the VERIFY_NONE shape).
func TLSDial(ip uint32, port uint16, timeoutMs int32, caPEM []byte) (*TLSConn, error) {
	if timeoutMs <= 0 {
		return nil, errOf(einval)
	}
	return dialIP(ip, port, timeoutMs, "", caPEM, caPEM == nil, 0)
}

// dialIP is the whole dial pipeline both entry points share: TCP connect,
// session setup, bounded handshake. hostname travels to the glue as the
// SNI + identity name ("" keeps the deprecated set_hostname(NULL) opt-out);
// insecure selects VERIFY_NONE; minVersion is a wire version or 0. deadline
// starts here so TCP connect + handshake share one total budget.
func dialIP(ip uint32, port uint16, timeoutMs int32, hostname string,
	ca []byte, insecure bool, minVersion uint16) (*TLSConn, error) {
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, errOf(e)
	}
	e := netConnect(fd, ip, int32(port))
	if e != 0 && e != einprogress && e != ealready {
		netClose(fd)
		return nil, errOf(e)
	}
	deadline := nowMs() + uint32(timeoutMs)
	if e != 0 { // connect in flight: select for writability, then SO_ERROR
		for {
			var ready int32
			if pe := netPoll(fd, 1, quantumMs, &ready); pe != 0 {
				netClose(fd)
				return nil, errOf(pe)
			}
			if ready == 1 {
				var soErr int32
				if se := netSockError(fd, &soErr); se != 0 {
					netClose(fd)
					return nil, errOf(se)
				}
				if soErr == 0 {
					break
				}
				netClose(fd)
				return nil, errOf(soErr)
			}
			if past(deadline) {
				netClose(fd)
				return nil, ErrTimeout
			}
		}
	}

	// TLS setup: config, CA parse, authmode, hostname, BIO, ssl_setup. The
	// strings travel as pointer/length pairs, a nil pointer for the empty
	// cases - the boundary passes integer and pointer pairs, and a Go
	// slice with no elements would pass a nil pointer either way.
	var caPtr *byte
	caLen := int32(0)
	if !insecure && len(ca) > 0 {
		caPtr = &ca[0]
		caLen = int32(len(ca))
	}
	var hostPtr *byte
	hostLen := int32(0)
	if hostname != "" {
		hostPtr = unsafe.StringData(hostname) // read-only bytes, len-bound
		hostLen = int32(len(hostname))
	}
	var h uintptr
	if e := tlsNew(fd, caPtr, caLen, hostPtr, hostLen,
		boolToInt32(insecure), int32(minVersion), &h); e != 0 {
		netClose(fd)
		return nil, errOf(e) // ErrCaParseFailed, ErrNoRootCAs or a raw negative mbedtls code (setup)
	}

	// Handshake by quanta: one handshake_step per loop, a poll quantum for
	// the WANT direction, the same total deadline as the TCP phase.
	for {
		r := tlsHandstep(h)
		if r == 0 {
			return &TLSConn{h: h, fd: fd}, nil
		}
		if r == mbErrWantRead || r == mbErrWantWrite {
			var ready int32
			w := int32(0)
			if r == mbErrWantWrite {
				w = 1
			}
			if pe := netPoll(fd, w, quantumMs, &ready); pe != 0 {
				tlsFree(h)
				netClose(fd)
				return nil, errOf(pe)
			}
			if past(deadline) {
				tlsFree(h)
				netClose(fd)
				return nil, ErrHandshakeTimeout
			}
			continue
		}
		if r == mbErrCertVerifyFailed {
			tlsFree(h)
			netClose(fd)
			return nil, ErrCertVerifyFailed
		}
		tlsFree(h)
		netClose(fd)
		return nil, ErrHandshake
	}
}

// splitHostPort splits "host:port" (the only address form this stack
// speaks: IPv4 dotted quads and DNS names - no IPv6, no service names).
// A second colon in the host is EINVAL rather than a resolve error, so a
// misplaced IPv6 literal fails loudly at the call instead of inside the
// resolver. port must be decimal 1..65535; host may be at most 255 bytes
// (ENAMETOOLONG - the DNS name limit).
func splitHostPort(addr string) (string, uint16, int32) {
	i := -1
	for j := len(addr) - 1; j >= 0; j-- {
		if addr[j] == ':' {
			i = j
			break
		}
	}
	if i <= 0 || i == len(addr)-1 {
		return "", 0, einval
	}
	host := addr[:i]
	for k := 0; k < len(host); k++ {
		if host[k] == ':' {
			return "", 0, einval
		}
	}
	ps := addr[i+1:]
	if len(ps) > 5 {
		return "", 0, einval
	}
	port := 0
	for k := 0; k < len(ps); k++ {
		if ps[k] < '0' || ps[k] > '9' {
			return "", 0, einval
		}
		port = port*10 + int(ps[k]-'0')
	}
	if port == 0 || port > 65535 {
		return "", 0, einval
	}
	if len(host) > 255 {
		return "", 0, enametoolong
	}
	return host, uint16(port), 0
}

// resolveHost maps a dotted-quad string or DNS name to the in-memory
// network-order address form net.IPv4 produces, through the same
// llgo_net_resolve primitive net.Resolve wraps: a dotted quad parses
// inside lwip without touching DNS, a real name goes through the lwip DNS
// client (which blocks the calling task on a driver semaphore and freezes
// the llgo scheduler until lwip's own DNS timers finish - the net package
// documents the caveat; prefer numeric IPs on time-critical paths).
// Failures return the same h_errno-family codes as net.Resolve.
func resolveHost(host string) (uint32, int32) {
	if host == "" {
		return 0, einval
	}
	if len(host) > 255 {
		return 0, enametoolong
	}
	buf := make([]uint8, len(host))
	copy(buf, host)
	var ip uint32
	if e := netResolve(&buf[0], int32(len(buf)), &ip); e != 0 {
		return 0, e
	}
	return ip, 0
}

// boolToInt32 turns a Go bool into the int32 flag the C boundary carries.
func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// Read reads up to len(p) bytes of decrypted plaintext and returns the
// count plus an error. Reads are short: one call returns what fits
// the caller's buffer from the record mbedtls holds; the caller loops until
// it has what it needs. The outcomes:
//
//	n > 0, nil          n bytes of plaintext arrived
//	0, ErrEOF           the peer closed the session in order: TLS
//	                    close_notify arrived (or the TCP stream closed
//	                    cleanly without one)
//	0, ErrWouldBlock    no deadline set and nothing arrived within one
//	                    quantum - a poll-style read
//	0, ErrTimeout       a deadline was set and passed with no data
//
// Any other error is the raw negative mbedtls code (a fatal session error;
// see ErrName for the common names). A zero-length p is a no-op success
// (it would otherwise be indistinguishable from a peer close).
func (c *TLSConn) Read(p []byte) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		var n int32
		e := tlsRead(c.h, &p[0], int32(len(p)), &n)
		if e == 0 {
			if n == 0 {
				return 0, ErrEOF
			}
			return int(n), nil
		}
		if e == mbErrPeerCloseNotify || e == mbErrConnEOF {
			return 0, ErrEOF
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			if pe := c.pace(e == mbErrWantWrite); pe != 0 {
				return 0, errOf(pe)
			}
			continue
		}
		return 0, errOf(e)
	}
}

// Write submits the whole buffer for encryption and returns (len(p), nil)
// on success. mbedtls accepts plaintext all-or-nothing (one record, no
// partial plaintext), so the failure contract is simpler than TCP's -
// every failure consumes NOTHING and the caller retries the whole buffer:
//
//	len(p), nil        the buffer is encrypted and handed to the socket
//	                   (mbedtls may still be retrying the record's bytes
//	                   inside its own BIO loop on later calls)
//	0, ErrWouldBlock   no deadline set and one quantum passed without
//	                   progress; retry the whole buffer later
//	0, ErrTimeout      a deadline was set and passed
//	0, err             the poll reported a socket error (lwip errno) or a
//	                   raw fatal mbedtls code - the session is unusable
//	                   either way
func (c *TLSConn) Write(p []byte) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		var n int32
		e := tlsWrite(c.h, &p[0], int32(len(p)), &n)
		if e == 0 {
			return int(n), nil // the whole buffer, or nothing on the error paths below
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			if pe := c.pace(e == mbErrWantWrite); pe != 0 {
				return 0, errOf(pe) // nothing accepted: caller retries all of p
			}
			continue
		}
		return 0, errOf(e)
	}
}

// Close ends the session: it tries to send the TLS close notification
// within a bounded 500 ms wait (ErrWantPollExhausted if the wait ends
// first), then frees the C context and shuts the TCP socket down in both
// directions - the fd is ALWAYS released, whatever the report. Close is
// idempotent: a second Close on the same conn (or any call on a closed or
// nil conn) reports ErrClosed. mbedtls treats a failed close_notify as
// recoverable (the session is over either way); the TCP shutdown means a
// stalled peer can never hold the close open.
func (c *TLSConn) Close() error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	fd := c.fd
	h := c.h
	c.fd = -1
	c.h = 0
	report := int32(0)
	deadline := nowMs() + closeBudgetMs
	for {
		e := tlsCloseNotify(h)
		if e == 0 {
			break
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			var ready int32
			w := int32(0)
			if e == mbErrWantWrite {
				w = 1
			}
			if pe := netPoll(fd, w, quantumMs, &ready); pe != 0 {
				report = pe
				break
			}
			if past(deadline) {
				report = tlsWantPollExhausted
				break
			}
			continue
		}
		report = e // fatal mbedtls code: record it, tear down anyway
		break
	}
	tlsFree(h)
	netShutdown(fd, shutRdwr)
	if e := netClose(fd); e != 0 && report == 0 {
		return errOf(e)
	}
	return errOf(report)
}

// SetDeadline sets the session's I/O deadline to ms milliseconds from now,
// or clears it with 0. Both Read and Write honor it until SetDeadline
// changes it. ms < 0 is EINVAL.
func (c *TLSConn) SetDeadline(ms int32) error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	if ms < 0 {
		return errOf(einval)
	}
	if ms == 0 {
		c.hasDeadline = false
		return nil
	}
	c.hasDeadline = true
	c.deadlineMs = nowMs() + uint32(ms)
	return nil
}

// State reports the established session's negotiated parameters and the
// peer certificate, for auditing and pinning: Version and CipherSuite as
// mbedtls's names, PeerCerts with the peer's leaf certificate in DER form
// (see ConnectionState). Everything is copied out of the mbedtls-owned
// context at call time, so the result outlives the conn. On a nil or
// closed conn - or a session whose handshake never completed - it returns
// the zero ConnectionState (no panic, no error: the zero Version and the
// empty PeerCerts already say "nothing to report").
func (c *TLSConn) State() ConnectionState {
	var st ConnectionState
	if c == nil || c.fd < 0 {
		return st
	}
	st.Version = cString(tlsVersion(c.h))
	st.CipherSuite = cString(tlsCiphersuite(c.h))
	var n int32
	if e := tlsPeerCert(c.h, nil, 0, &n); e != 0 || n <= 0 {
		return st // no peer cert (NoPeerCert): version/cipher still stand
	}
	der := make([]byte, n)
	if e := tlsPeerCert(c.h, &der[0], n, &n); e != 0 {
		return st
	}
	st.PeerCerts = [][]byte{der}
	return st
}

// pace waits one poll quantum (a select that yields the CPU) and reports
// how the bounded wait ended: 0 = retry the TLS attempt now, eagain = no
// deadline is set and nothing became ready within the quantum, etimedout =
// the session deadline passed. A select failure surfaces as its errno -
// the next TLS attempt would report the same dead socket anyway.
func (c *TLSConn) pace(wantWrite bool) int32 {
	w := int32(0)
	if wantWrite {
		w = 1
	}
	var ready int32
	if e := netPoll(c.fd, w, quantumMs, &ready); e != 0 {
		return e
	}
	if c.hasDeadline && past(c.deadlineMs) {
		return etimedout
	}
	if ready == 0 && !c.hasDeadline {
		return eagain
	}
	return 0
}

// past reports whether an absolute uptime-ms deadline has passed. The
// unsigned subtraction compares safely across the 32-bit millisecond
// counter wrap - including a deadline that wraps through 0, which is why
// "no deadline" is carried by TLSConn.hasDeadline instead of a 0-value
// sentinel.
func past(deadline uint32) bool {
	return uint32(nowMs()-deadline) < 1<<31
}

// ErrName maps the codes this pipeline can hit to their names. Three
// families share one namespace: the 0x7E00 TLS codes this package defines,
// the negative mbedtls codes the setup and record phases pass through, and
// the lwip errno values (picolibc numbering - see net.ErrName for the
// Linux divergences) the TCP phase can surface. A negative value is tried
// against the mbedtls table first, then resolved through the errno table by
// its absolute value (net.ErrName's pattern) - so Write's -EAGAIN contract
// reports "EAGAIN" instead of a hex code. A code from no family prints as
// its value.
func ErrName(code int32) string {
	if code < 0 {
		switch code {
		case mbErrWantRead:
			return "MBEDTLS_ERR_SSL_WANT_READ"
		case mbErrWantWrite:
			return "MBEDTLS_ERR_SSL_WANT_WRITE"
		case mbErrPeerCloseNotify:
			return "MBEDTLS_ERR_SSL_PEER_CLOSE_NOTIFY"
		case mbErrConnEOF:
			return "MBEDTLS_ERR_SSL_CONN_EOF"
		case mbErrCertVerifyFailed:
			return "MBEDTLS_ERR_X509_CERT_VERIFY_FAILED"
		case -0x7780:
			return "MBEDTLS_ERR_SSL_FATAL_ALERT_MESSAGE"
		case -0x7700:
			return "MBEDTLS_ERR_SSL_UNEXPECTED_MESSAGE"
		case -0x7300:
			return "MBEDTLS_ERR_SSL_DECODE_ERROR"
		case -0x7200:
			return "MBEDTLS_ERR_SSL_INVALID_RECORD"
		case -0x7180:
			return "MBEDTLS_ERR_SSL_INVALID_MAC"
		case -0x7080:
			return "MBEDTLS_ERR_SSL_FEATURE_UNAVAILABLE"
		case -0x7000:
			return "MBEDTLS_ERR_SSL_CRYPTO_IN_PROGRESS"
		case -0x7A00:
			return "MBEDTLS_ERR_SSL_BAD_CERTIFICATE"
		case -0x7500:
			return "MBEDTLS_ERR_SSL_UNSUPPORTED_EXTENSION"
		case -0x7800:
			return "MBEDTLS_ERR_SSL_UNRECOGNIZED_NAME"
		case -0x7480:
			return "MBEDTLS_ERR_SSL_NO_CLIENT_CERTIFICATE"
		case -0x004C:
			return "MBEDTLS_ERR_NET_RECV_FAILED"
		case -0x004E:
			return "MBEDTLS_ERR_NET_SEND_FAILED"
		case -0x004F:
			return "MBEDTLS_ERR_NET_CONN_RESET"
		}
		// Not a named mbedtls code. Values in the errno range are the
		// negated lwip errnos the Write contract and the BIO pass through
		// (abs pattern above, 134 = the largest errno named below);
		// anything larger is a raw mbedtls code, printed in its
		// conventional hex form.
		if v := -code; v <= 134 {
			code = v
		} else {
			return "mbedtls -0x" + hex(uint32(v))
		}
	}
	switch code {
	case 0:
		return "OK"
	case tlsHandshakeTimeout:
		return "TLS_ERR_HANDSHAKE_TIMEOUT"
	case tlsCertVerifyFailed:
		return "TLS_ERR_CERT_VERIFY"
	case tlsCaParseFailed:
		return "TLS_ERR_CA_PARSE"
	case tlsWantPollExhausted:
		return "TLS_ERR_WANT_POLL_EXHAUSTED"
	case tlsHandshakeFailure:
		return "TLS_ERR_HANDSHAKE"
	case tlsPsaInitFailed:
		return "TLS_ERR_PSA_INIT"
	case tlsNoRootCAs:
		return "TLS_ERR_NO_ROOT_CAS"
	case tlsNoPeerCert:
		return "TLS_ERR_NO_PEER_CERT"
	case 4:
		return "EINTR"
	case 5:
		return "EIO"
	case 9:
		return "EBADF"
	case 11:
		return "EAGAIN"
	case 12:
		return "ENOMEM"
	case 13:
		return "EACCES"
	case 22:
		return "EINVAL"
	case 32:
		return "EPIPE"
	case 91:
		return "ENAMETOOLONG"
	case 95:
		return "EOPNOTSUPP"
	case 104:
		return "ECONNRESET"
	case 105:
		return "ENOBUFS"
	case 106:
		return "EAFNOSUPPORT"
	case 107:
		return "EPROTOTYPE"
	case 108:
		return "ENOTSOCK"
	case 109:
		return "ENOPROTOOPT"
	case 110:
		return "ESHUTDOWN"
	case 111:
		return "ECONNREFUSED"
	case 112:
		return "EADDRINUSE"
	case 113:
		return "ECONNABORTED"
	case 114:
		return "ENETUNREACH"
	case 115:
		return "ENETDOWN"
	case 116:
		return "ETIMEDOUT"
	case 117:
		return "EHOSTDOWN"
	case 118:
		return "EHOSTUNREACH"
	case 119:
		return "EINPROGRESS"
	case 120:
		return "EALREADY"
	case 121:
		return "EDESTADDRREQ"
	case 122:
		return "EMSGSIZE"
	case 123:
		return "EPROTONOSUPPORT"
	case 125:
		return "EADDRNOTAVAIL"
	case 126:
		return "ENETRESET"
	case 127:
		return "EISCONN"
	case 128:
		return "ENOTCONN"
	case 134:
		return "ENOTSUP"
	}
	return "errno " + dec(uint32(code))
}

// dec formats v in decimal (fmt is not imported in pkg/esp packages).
func dec(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// hex formats v in lowercase hex (mbedtls error codes are conventionally
// printed in hex; mbedtls -0x6900 names the code unambiguously).
func hex(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for v > 0 {
		i--
		d := v & 15
		if d < 10 {
			buf[i] = byte('0' + d)
		} else {
			buf[i] = byte('a' + d - 10)
		}
		v >>= 4
	}
	return string(buf[i:])
}

// cString copies a NUL-terminated C string into a Go string (nil reads as
// empty). The scan bounds itself to nothing but the terminator - the two
// mbedtls-owned strings this feeds (get_version, get_ciphersuite) are
// short static names, and State() runs only after a completed handshake.
func cString(p *byte) string {
	if p == nil {
		return ""
	}
	n := int32(0)
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}
