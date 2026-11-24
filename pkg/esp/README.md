# pkg/esp — llgo × ESP-IDF binding packages

Formal llgo binding packages for the ESP-IDF drivers, running on
ESP32-S31 with the `esp32s31-idf` target (llgo compiles the program into a
relocatable object that an ESP-IDF component links into the firmware; see
`_demo/esp-idf-wifi/` for the full pipeline). Each package exposes an
ESP-IDF driver to Go through a thin C glue, so ESP peripherals become
usable from Go with hours-scale effort instead of a porting project:

| Package | IDF driver | Scope |
|---------|-----------|-------|
| `pkg/esp/wifi` | `esp_wifi` + `esp_netif` + `nvs_flash` | station bring-up, scan, connect, DHCP, heap watermarks |
| `pkg/esp/gpio` | `driver/gpio.h` | direction config, level set/get, toggle |
| `pkg/esp/i2c`  | `driver/i2c_master.h` (new driver) | master bus + device handles, write/read/write-read |
| `pkg/esp/i2s`  | `driver/i2s_std.h` (new driver) | std (Philips) mode TX/RX, 16/32-bit, mono/stereo |
| `pkg/esp/net`  | `lwip/sockets.h` + `lwip/netdb.h` | TCP dial/listen/accept/read/write, DNS resolve, bounded quantum waits |
| `pkg/esp/tls`  | `mbedtls/ssl.h` + `mbedtls/x509_crt.h` (4.1.1, PSA crypto) | TLS client dial (VERIFY_NONE/REQUIRED), record read/write, close_notify, bounded quantum waits |
| `pkg/esp/ws`   | — (no glue; pure Go RFC 6455 over a caller-supplied `net`/`tls` conn) | HTTP Upgrade handshake, masked frames, ping/pong, closing handshake; stdlib-free SHA-1/base64 |

## Package template

Every package here follows the same layout (`xxx` = package name; the
glue lives in a subdirectory because the go tool rejects C sources
inside a Go package, and a directory without Go files is invisible to
it):

```
pkg/esp/xxx/
├── xxx.go           Public API, no build tag. Go-idiomatic names over a
│                    thin wrapper; every function doc states the esp_err
│                    semantics.
├── xxx_idf.go       //go:build llgo.idf — bodyless //go:linkname externs only.
├── xxx_stub.go      //go:build !llgo.idf — host-build stubs, so `go build`,
│                    `go vet` and gopls stay clean with zero device code.
└── glue/
    └── xxx_glue.c   ESP-IDF C glue template. Compiled only under
                     ESP_PLATFORM (the whole body compiles away elsewhere),
                     meant to be copied into the consuming project's main
                     component.
```

The `llgo.idf` build tag is set automatically by the `esp32s31-idf`
target (`targets/esp32s31-idf.json`), never by hand. On the device build
the `_idf.go` file supplies the C symbol references; on every other build
the `_stub.go` file keeps the package compiling. Stubs return
`errUnsupported = 0x7F00`, a code between the real esp_err ranges, so it
can never be mistaken for one; `vTaskDelay`-style no-ops return plain zero
values.

Packages under pkg/esp are kept dependency-free (only stdlib/`unsafe`), so
each glue file can be adopted independently.

## Go ↔ C boundary rules

These rules are what keep the llgo fiber runtime safe across the link.
They are contract, not style.

1. **Integers and pointers only.** No float values ever cross the Go↔C
   link: llgo's RV32 codegen is soft-float while IDF compiles S31 with the
   FPU (ilp32f), and the artifact pipeline patches the object's float-ABI
   flag precisely because the boundary stays integer-only (see
   `_demo/esp-idf-wifi/idfproj/host/make-artifact.sh` step 4).
2. **Errors are raw `esp_err` codes as `int32`.** 0 = ESP_OK. No Go error
   wrapping at the boundary; mapping codes to names is a separate helper
   (`wifi.ErrName` pattern). IDF's `esp_err_t` is `int` — the glue
   signature is `int32_t` and the value passes through unmodified.
3. **No C→Go callbacks.** esp_event handlers, driver ISR callbacks and the
   like run on their own FreeRTOS tasks. The llgo fiber scheduler is
   single-task global state: an entry from a second FreeRTOS task into Go
   would corrupt it. Never register a C→Go trampoline.
4. **Async events go through a mailbox.** The C side copies events into a
   ring (`wifi_glue.c`'s 16-slot SPSC queue) and Go polls it paced by
   `vTaskDelay(10)` = 100 ms — the delay also keeps the idle tasks fed for
   the watchdog. Overflow drops the newest event and is reported once.
   Synchronous driver calls that block on a driver semaphore (a WiFi scan,
   an I2C transfer timeout) are safe: they yield the CPU.
5. **No large stack buffers in glue.** Go goroutines run on 8 KiB fiber
   stacks. Bulk data (scan records, DMA-size sample buffers) lives in
   static storage on the C side or flows through Go heap slices whose
   first element is passed as a pointer; small config structs built inside
   one glue function are fine.
6. **IDF macros and config structs stay in C.**
   `WIFI_INIT_CONFIG_DEFAULT()`, `I2S_CHANNEL_DEFAULT_CONFIG()`,
   `I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG()`, `gpio_config_t` and friends
   embed Kconfig constants, enums and function pointers Go cannot spell.
   The glue builds them; Go passes scalars (pins, frequencies, widths,
   bools as 0/1).
7. **Flat structs only across the boundary.** When records must travel
   (WiFi scan results), the glue marshals into a fixed-layout struct with
   scalar and fixed-size-array fields; IDF types with bitfields or nested
   structs never cross.

### Handles

Driver handles (`i2c_master_bus_handle_t`, `i2s_chan_handle_t`, ...) are
opaque C pointers. They travel as integers: the Go side stores them in a
`uintptr` field (so the GC never scans them as pointers) and the glue
signature uses `uintptr_t`, casting back to the C handle type on entry.
The wifi package holds its netif handle purely C-side (`static`), which is
even thinner — prefer that when Go never needs the handle.

### linkname naming

C symbols are `llgo_<pkg>_<snake_verb>` (`llgo_wifi_scan_fetch`,
`llgo_gpio_set_level`, `llgo_i2c_new_bus`, `llgo_i2s_init_std`); project
or demo-local helpers prefix with the project (`llgo_demo_rdcycle`). Go
declares them in `xxx_idf.go` only, as
`//go:linkname name C.llgo_<pkg>_<verb>`. The runtime's own pattern is
`runtime/internal/runtime/idf_freertos.go` (`C.vTaskDelay`).

## Glue integration

`llgo_out.o` is the LAST item on the final link line and the ESP-IDF link
uses no `--start-group`, so each component archive is scanned only against
the undefined symbols known at its position. Glue symbols are referenced
solely from `llgo_out.o`; a plain archive would never pull them. Hence:

**The main component must be WHOLE_ARCHIVE**, keeping every glue object in
the link unconditionally (see the CMakeLists comment in
`_demo/esp-idf-wifi/idfproj/main/`). Two ways to get the glue compiled:

1. **Copy (demo pipeline).** Copy the `glue/*_glue.c` templates into the
   project's `main/` component and list them in `idf_component_register`'s
   `SRCS` with the matching `REQUIRES` (`esp_driver_gpio`,
   `esp_driver_i2c`, `esp_driver_i2s`, plus the wifi set for the wifi
   glue). This is what `_demo/esp-idf-wifi/idfproj/host/make-artifact.sh`
   does as a build step, so the repo stays the single source of truth and
   the IDF docker build needs no extra mounts.
2. **CMake reference.** Point the component's `SRCS` at the package paths
   directly (`SRCS ... ${LLGO_REPO}/pkg/esp/gpio/glue/gpio_glue.c`), with
   `LLGO_REPO` provided by the environment or a cache variable. Requires
   the llgo checkout to be visible where `idf.py` runs (e.g. a second
   `docker run -v` mount); no file copies.

Either way the templates' `#if defined(ESP_PLATFORM)` guard makes them
harmless if they ever leak into a non-IDF build.

## pkg/esp/net — TCP sockets under the bounded-quantum model

`pkg/esp/net` exposes lwip's TCP sockets (`Dial`, `Listen`, `Accept`,
`Read`, `Write`, `Close`, `Resolve` for names, `SetDeadline` for a
connection-wide deadline) over a glue whose inventory is deliberately
thin: `llgo_net_socket`, `llgo_net_connect`, `llgo_net_poll`,
`llgo_net_sock_error`, `llgo_net_recv`, `llgo_net_send`,
`llgo_net_shutdown`, `llgo_net_close`, `llgo_net_bind_listen`,
`llgo_net_accept`, `llgo_net_resolve`, `llgo_net_now_ms`. Every socket it creates is marked
`O_NONBLOCK` at birth (`fcntl` in the glue), because llgo's fiber scheduler
is single-task and one blocking C call would freeze the whole Go scheduler.

**The quantum loop.** Each operation is a Go-side retry loop: one
non-blocking syscall attempt, then a 20 ms wait in `llgo_net_poll` (a
`lwip_select` wrapper kept for reuse) — a wait that parks the FreeRTOS task
on lwip's select semaphore, which yields the CPU and keeps the idle tasks
(and the watchdog) fed — then a deadline check. Timeouts are accurate to
within one quantum. Deadlines come from the call (`Dial`/`Accept`
`timeoutMs`, `<= 0` rejected with EINVAL) or from the connection
(`SetDeadline`, relative ms, 0 = none; with none set, a `Read` gives up
after one quantum with EAGAIN and a `Write` returns a short count). There
is no unbounded wait in the path except the single documented freeze,
`Resolve` on a real name: lwip's DNS client waits on its own timers
(dotted-quad literals return instantly, parsed before DNS is consulted).

**Error domain.** Errors are lwip errno values (0 = success), not
`esp_err` — that is what lwip's socket layer sets — and `Resolve` returns
the netdb `h_errno` family (200–213). Mind the numbering: the ESP-IDF
build compiles picolibc's `errno.h`, whose values above 103 differ from
Linux (`ETIMEDOUT` = 116, `EINPROGRESS` = 119, `ENOTCONN` = 128; `net.ErrName`
knows them, and the shim probe printed the firmware's own values to pin
the table). A negative return from `Write` is `-errno`.

**Scheduling.** The quantum waits are C-level: lwip_select gives the llgo
scheduler no cooperative yield, so goroutines do not interleave while one
of them sits in a net wait - the same scheduler architecture property the
wifi package's vTaskDelay pacing has, and now measured under esp-emu: a
dial goroutine and an accept main, each bounded to 3 s, ran strictly
serially (the accept main consumed its whole 3026 ms window and timed out;
the starved dial goroutine finished its connect in 0 ms once the accept
side reached a channel join). The watchdog stays fed; the pattern to avoid
is two fibers spinning toward each other (a dial in one goroutine, the
accept in another). Keep a session on one goroutine, or hand control
through channel operations, which are real yields; the demo's loopback
probe runs strictly sequentially (dial → accept → exchange), which also
matches the protocol — the handshake completes into the listener's backlog
before Accept runs.

**Integration.** Same two recipes as every package here; the main
component needs `REQUIRES lwip esp_timer` (the glue reads
`esp_timer_get_time` for the Go-side deadline clock). For an emu
acceptance run, start the host-side peer first:
`go run ./idfproj/host/echosrv` (TCP echo on 127.0.0.1:9000) — the
emulator's user-mode NAT aliases its gateway address (192.168.4.1 in the
default subnet) to the host, forwarding guest connections to the same
port on the host's loopback.

## pkg/esp/tls — TLS client sessions over the same bounded-quantum model

`pkg/esp/tls` adds a record layer on top of a connected TCP socket:
`TLSDial(ip, port, timeoutMs, caPEM)` connects, parses the optional CA PEM
and handshakes as a client; `TLSConn.Read`/`Write`/`Close`/`SetDeadline`
then speak plaintext. The glue inventory is deliberately thin:
`llgo_tls_new` (config defaults, CA parse + authmode, BIO wiring,
ssl_setup), `llgo_tls_handstep` (one `mbedtls_ssl_handshake_step`),
`llgo_tls_read`/`llgo_tls_write` (one record attempt),
`llgo_tls_close_notify`, `llgo_tls_free`. Each call is a single
non-blocking attempt; the Go side drives the quantum loops over
`llgo_net_poll` exactly like the net package — the WANT_READ/WANT_WRITE
mbedtls reports map onto the same socket quanta. The shared socket/clock
primitives (`llgo_net_socket`, `llgo_net_connect`, `llgo_net_poll`,
`llgo_net_sock_error`, `llgo_net_shutdown`, `llgo_net_close`,
`llgo_net_now_ms`) are declared by linkname in BOTH packages and defined
once in `net_glue.c`: a TLS program links both glues, but the packages
stay mutually dependency-free at the Go level (each adopts independently).

**mbedtls 4.1 specifics** (IDF v6.3-dev): the SSL layer's randomness is
PSA Crypto random (`psa_generate_random`), which IDF wires to the hardware
TRNG (`esp_fill_random`) and initializes at startup — no `conf_rng`, and
the ctr_drbg of older mbedtls is a private header in 4.1.1. VERIFY_REQUIRED
requires an explicit `mbedtls_ssl_set_hostname` call or the handshake
fails with `MBEDTLS_ERR_SSL_CERTIFICATE_VERIFICATION_WITHOUT_HOSTNAME`
(-0x5D80); the glue opts out with `set_hostname(NULL)` — chain + validity
are still checked, no CN/SAN matching happens (the API has no hostname
parameter). `mbedtls_x509_crt_parse`'s PEM path needs the NUL terminator
counted INSIDE the length, so the glue parses a heap copy. And the first
`handshake_step` only advances HELLO_REQUEST → CLIENT_HELLO returning 0
without network I/O — the glue wraps it so 0 always means "handshake over".

**Error domain.** TLS-phase failures use the 0x7E00 segment
(`HandshakeTimeout`/`CertVerifyFailed`/`CaParseFailed`/`WantPollExhausted`/
`Handshake`/`PsaInitFailed`, named by `tls.ErrName`); the TCP phase surfaces
lwip errnos exactly like `net.Dial`; failures inside mbedtls's own dial
setup (config defaults, `ssl_setup`) and record-phase fatals pass the raw
negative mbedtls code through. Read/Write honor the connection deadline
with net.Conn semantics (EAGAIN poll-read without a deadline, ETIMEDOUT at
expiry); TLS Write is all-or-nothing (no partial plaintext), so "nothing
accepted" returns -EAGAIN and the caller retries the whole buffer. `Close`
sends close_notify under a bounded 500 ms wait (`WantPollExhausted` if it
cannot finish) and always releases the fd.

**Memory.** One live TLS connection holds 24-25 KiB of internal SRAM at the
default fragment lengths (IN 16 KiB + OUT 4 KiB buffers from
`CONFIG_MBEDTLS_SSL_IN/OUT_CONTENT_LEN`, plus the ssl/config/CA state —
all measured internal under esp-emu with SPIRAM enabled: the >16 KiB IN
buffer still lands on the internal heap, PSRAM untouched); after Close
everything returns and the free watermark recovers (four consecutive
sessions recover to within 8 B, logs/p6bfix-soak.log). Measured on the demo
probe's live-connection heap marks: WiFi steady-state internal free
53,655 B → 28,819 B with the VERIFY_NONE session open (24,244 B held),
27,723 B with VERIFY_REQUIRED (25,012 B held; the parsed CA adds the ~0.75
KiB difference).
That leaves ~28 KiB of headroom over the WiFi steady state, so the demo
keeps the default fragment lengths; lowering
`CONFIG_MBEDTLS_SSL_IN/OUT_CONTENT_LEN` would shrink this further at the
price of record-size compatibility (Go's crypto/tls does not negotiate
RFC 6066 max-fragment-length, so a shrunken IN buffer is only safe
against peers that never send records larger than the buffer).

**Scheduling.** Same architecture property as net: the quantum waits are
C-level selects and give the llgo scheduler no yield point. Keep a TLS
session on one goroutine, or hand control through channel operations.

**Emulator fidelity (esp-emu 1.2.0, measured with a pure-C probe before
the binding, logs p6b-shim-emu*).** Everything the TLS handshake needs is
modeled: a full TLS 1.2 client handshake (TLS-ECDHE-RSA-AES128-GCM-SHA256)
against a host crypto/tls peer passes with the IDF default hardware
crypto accelerators enabled AND in a pure-software A/B config — no
fidelity gap, no software fallback needed. `esp_random` returns non-zero
values (the PSA random path is real), and the guest wall clock sits at
the epoch (`time()` = 1): certificates must be valid from 1970, which the
demo's gencert.sh does (`-not_before 19700101000000Z`, 100-year window).
The demo acceptance re-verified the whole stack through the binding
itself (`[go] tls ...`).

**Integration.** The main component needs `REQUIRES mbedtls esp_hw_support`
in addition to the net glue set (`lwip esp_timer`); `esp_hw_support` is a
defensive REQUIRES - the glue compiles against mbedtls/psa headers only
(the TRNG wiring behind `psa_generate_random` lives inside IDF's mbedtls
port), and the entry documents that dependency. For an emu acceptance
run, start the host TLS peer first: `go run ./idfproj/host/tlssrv` (TLS
echo on 127.0.0.1:9443, certs from `idfproj/host/gencert.sh`) — the NAT
forwards the guest's 192.168.4.1:9443 to it.

## pkg/esp/ws — RFC 6455 WebSocket client, pure Go, no glue

`pkg/esp/ws` adds the WebSocket protocol on top of a transport the caller
already dialed: `HandshakeURL(c, host, path, timeoutMs)` runs the HTTP
Upgrade handshake over a `*net.TCPConn` (ws://) or a `*tls.TLSConn`
(wss://), and the returned `WSConn` offers `ReadMessage`, `WriteMessage`,
`Ping`, `Close(code)`, `SetDeadline`, `PongSeen`, `CloseCode` and
`LastOpcode`. The package defines its own `Conn` interface
(`Read/Write/Close` with the pkg/esp int32 signatures) - Go interfaces are
structural, so the real transports satisfy it without the package
importing them - plus the optional `Deadliner` interface
(`SetDeadline(ms int32) int32`), which `*TCPConn` and `*TLSConn` both
implement as methods. It imports NOTHING - not even `unsafe` - so it
compiles unchanged on a host build, runs under `go test` with an
in-memory transport, and contributes no C glue: a ws program links the
net/tls glues it already needs and nothing else, and `make-artifact.sh`
needs no new checklist entry.

**Why it carries its own SHA-1 and base64 (image size, not availability).**
The handshake cannot exist without SHA-1 and base64 (the accept key is
`base64(SHA-1(key + GUID))`). When this section was written, the
standard-library versions were unlinkable on esp32s31-idf - any probe
importing `crypto/sha1`, `encoding/base64`, `bufio`, `io`, `bytes`,
`strings` or `strconv` died at the libffi LLGoFiles preprocessing step
(`stdlib.h not found`; all of them reach `internal/reflectlite`, which
llgo maps to its own reflect layer, which dragged the libffi C glue into
the build). That gap is fixed since - the glue is freestanding-header-only
now and the FIPS entropy scratch buffer is patched to an embedded-friendly
size - but the ws package keeps its own primitives anyway: importing
`crypto/sha1` through the stdlib reflect machinery pulled a probe from
~1.19 MB to 2.24 MB of app image, while `hash.go` (~200 lines) delivers
the same handshake cryptography in a few hundred bytes. The package
therefore implements RFC 3174 SHA-1 and RFC 4648 base64 itself
(`hash.go`), parses the handshake response with a hand-rolled buffered
reader (no bufio), and formats its
error names without strconv. `SHA1Sum`/`Base64StdEncode`/`Base64StdDecode`
and `AcceptKey` are exported: a pkg/esp program on a lean target has a
dependency-free SHA-1/base64 without paying the stdlib image cost, and
the acceptance probe pins them against the FIPS 180-1 / RFC 6455 /
RFC 4648 known values in the firmware log itself.

**Deadlines are delegated, never invented.** The package has no clock
and starts no goroutines. HandshakeURL applies `timeoutMs` through the
transport's SetDeadline for the handshake and clears it afterwards; a
transport without a SetDeadline method cannot bound the handshake and is
rejected with `ErrNoDeadline`. With a session deadline set,
`ReadMessage` returns ETIMEDOUT and `WriteMessage` a negative
-ETIMEDOUT; without one, `ReadMessage` is a poll-style read (EAGAIN
after one quantum) whose framing state machine resumes mid-header or
mid-payload on the next call. Error codes use the 0x7D00 segment
(`ErrHandshake`/`ErrNotWS`/`ErrAcceptKey`/`ErrTooBig`/`ErrProtocol`/
`ErrPeerClosed`/`ErrConnDead`/`ErrNoDeadline`, named by `ws.ErrName`
alongside the lwip errno table); transport failures pass through
unchanged, and calls on a closed session return EBADF like the
transports. A framing violation (masked server frame, RSV bits set,
fragmented control frame) closes the session on the spot - a misaligned
byte stream has no recovery, and no close frame is sent: answering with
the RFC 6455 section 7.1.7 1002 close is a SHOULD the package
deliberately skips, because a misaligned stream cannot promise a
well-formed closing exchange.

**Scope and randomness.** RFC 6455 baseline only: no subprotocols, no
permessage-deflate (RSV bits are a protocol error), client frames always
masked, one frame per ReadMessage (a fragmented message arrives as
consecutive reads, `LastOpcode` reporting each fragment's opcode; a
frame bigger than the caller's buffer is drained and reported as
`ErrTooBig`, leaving the session usable). Pings are answered
automatically inside the read path; a pong is reported as a `(0, 0)`
ReadMessage event with `PongSeen()` true, so a ping/pong check never
blocks on a data frame that never comes. Two receive-side waivers, both
intentional and documented: text frames are NOT validated as UTF-8 (the
RFC 6455 section 8.1 client MUST is waived - whoever sends `OpText`
guarantees the bytes; use `OpBinary` for arbitrary bytes), and a peer's
close status code is echoed verbatim even when it is outside the valid
ranges (the strict answer would be a 1002; a code-less close is echoed
as an empty payload, since a code below 1000 must never go on the wire).
The client key and the
per-frame mask keys come from an in-package xorshift64* PRNG -
`Seed(s)` it once with the best entropy the board offers (uptime,
ADC noise); unseeded it is deterministic: wire-valid and fine on the
bench, predictable through an intermediary.

**Memory and scheduling.** Measured on the acceptance build: a live
plain session holds 388 bytes of Go heap (the `WSConn` struct itself is
on the order of ~180 bytes of that, the 125-byte control/drain staging
buffer included; the rest is the handshake's residual allocations), plus
transient frame buffers, on top of whatever the transport holds
(a plain TCP conn is negligible; a wss conn adds the ~24 KiB TLS ledger
of the tls package). Post-close the transport returns everything. The
bounded waits are the transports' C-level select quanta - the same
scheduler property as net/tls: keep one session on one goroutine.

**Integration and acceptance.** No glue to copy, no REQUIRES beyond
what net/tls already bring. The demo probe (`[go] ws ...` in
`_demo/esp-idf-wifi/goapp/main.go`) runs against
`idfproj/host/wssrv` - a host peer that sniffs each connection and
serves the raw echo, the TLS echo and both WebSocket endpoints on the
same two ports (9000/9443), so one process backs the whole
wifi->net->tls->ws acceptance chain.

## Adding a new peripheral

Template checklist (0.5–3 days per driver, dominated by emu/device
validation):

1. `pkg/esp/<xxx>/<xxx>.go` — Go API over thin wrappers; document each
   call's esp_err semantics; decide handle ownership (Go `uintptr` vs
   C-side static) and whether the driver needs a mailbox (only for
   asynchronous event sources — ISR/event-queue drivers).
2. `<xxx>_idf.go` — `//go:build llgo.idf` externs, `C.llgo_<xxx>_*` names.
3. `<xxx>_stub.go` — `//go:build !llgo.idf`, `errUnsupported` returns.
4. `<xxx>_glue.c` — template with the `ESP_PLATFORM` guard; macros and
   config structs C-side; mailbox only if the driver is an async event
   source.
5. Hosted `gofmt -l`, `go vet ./pkg/esp/...`, `go build ./pkg/esp/...`
   must be clean.
6. Probe + acceptance under esp-emu (and hardware when available): print
   per-call esp_err codes; record the emulator's peripheral fidelity in
   the probe's output and flip the probe's skip switch if a call hangs the
   emulator.

## Emulator fidelity notes (esp-emu 1.2.0, chip esp32s31)

The emulator models WiFi (softAP) and PSRAM faithfully; peripheral
fidelity is documented by the acceptance probe in
`_demo/esp-idf-wifi/goapp/main.go` (`[go] periph ...` console lines, and
`[go] net ...` for the network items below). Measured on S31, 16 MiB
PSRAM variant (2026-11):

- **GPIO**: modeled. Output direction + set→get→toggle round-trip passes
  (`gpio.Pin(4)`).
- **I2C**: modeled at the install/transfer level - master bus install on
  GPIO 8/9 and a receive complete with `err = 0`. Two glue-level lessons
  came out of the probe: the bus config's `clk_source` must be spelled
  out (`I2C_CLK_SRC_DEFAULT`) because S31's only I2C sources are XTAL and
  RC_FAST, so a zeroed union is an invalid source (`ESP_ERR_INVALID_ARG`),
  and the emulator does not model ACK/NACK: a read from an absent address
  succeeds with `0xFF` bytes instead of a driver error. Real transfers
  against real devices must be validated on hardware.
- **I2S**: modeled. TX channel alloc, std (Philips) init, enable and a
  1024-byte write all succeed.

### Network (lwip sockets)

Measured first with a pure-C shim probe over the emulator's NAT
(workflow: host echo server on 127.0.0.1:9000, firmware dials the
gateway alias 192.168.4.1:9000), then re-verified through
`pkg/esp/net` itself in the Go acceptance probe (`[go] net ...`):

- **WiFi association + DHCP**: modeled; the station acquires 192.168.4.2
  from the emulator's softAP/NAT. The earlier scan-only acceptance runs
  never exercised this path; the network work was the first to need it.
- **`fcntl` O_NONBLOCK**: modeled; `F_GETFL`/`F_SETFL` round-trips.
- **Short `select`**: modeled; a 50 ms select returns 0 after 59 ms.
- **`SO_RCVTIMEO`**: modeled; a blocking recv returns EAGAIN (11) after
  49 ms with a 50 ms timeout. The package does not need it (the primary
  model is non-blocking + select quanta) but it works as a fallback.
- **Loopback (127.0.0.1)**: modeled (IDF enables `LWIP_NETIF_LOOPBACK`),
  against a loopback-bound listener, a wildcard (`INADDR_ANY`) listener,
  and via the station's own address (192.168.4.2); full
  connect/accept/echo round-trips.
- **DNS**: dotted-quad literals resolve instantly (lwip parses them
  before consulting DNS); `localhost` resolves through the emulator's
  DNS proxy to 127.0.0.1 in <1 ms.
- **NAT to host**: modeled; a guest connection to 192.168.4.1:9000
  forwards to the host's 127.0.0.1:9000 and echoes (the Go acceptance
  run verifies 256 pseudo-random bytes both ways, `echoed 256 bytes`
  on the echosrv log as the host-side cross-evidence).
- **errno numbering**: picolibc's, not Linux's — the C probe printed
  `ECONNRESET=104 ECONNREFUSED=111 ETIMEDOUT=116 EINPROGRESS=119
  EALREADY=120 EISCONN=127 ENOTCONN=128 EADDRINUSE=112 ENETUNREACH=114
  ENOBUFS=105`, and lwip's `h_errno` family is 200–213
  (`HOST_NOT_FOUND=210`, `TRY_AGAIN=213`).
- **Scheduler note**: a net quantum wait is a C-level select; llgo
  fibers do not interleave inside it (see the net section above). Not an
  emulator fidelity gap — the same holds on hardware.

No probe hangs or crashes the emulator, and the whole run is bounded:
every driver path used by the probes has a driver-side or emulator-side
timeout, so a missing peripheral model surfaces as an esp_err code (a
missing network peer as an errno) — never a hang. The probe switches
(`probeGPIO`/`probeI2C`/`probeI2S`/`probeNet` consts in `main.go`) exist
only as a contingency for a target where one does.

Two non-peripheral notes from the same run: the default single-app
partition (1 MiB) overflows once the I2C/I2S driver code joins the link
(app ~1.03 MiB), so the demo ships a custom `partitions.csv` with a
3.9375 MiB (0x3F0000) factory app; and the long PSRAM GC loop legitimately starves
IDLE0 - the task-watchdog warnings in the console log are the same
baseline noise as before the peripherals were added.
