# pkg/esp — llgo × ESP-IDF binding packages

Formal llgo binding packages for the ESP-IDF drivers, running on
ESP32-S31 with the `esp32s31-idf` target (llgo compiles the program into a
relocatable object that an ESP-IDF component links into the firmware; see
`_demo/esp-idf-wifi/` for the full pipeline). Each package exposes an
ESP-IDF driver to Go through a thin C glue, so ESP peripherals become
usable from Go with hours-scale effort instead of a porting project:

| Package | IDF driver | Scope |
|---------|-----------|-------|
| `pkg/esp/wifi` | `esp_wifi` + `esp_netif` + `nvs_flash` | station and soft-AP bring-up (STA / AP / APSTA), scan, connect, DHCP client and server, AP station list + deauth, auto-reconnect state machine, heap watermarks |
| `pkg/esp/gpio` | `driver/gpio.h` | direction config, level set/get, toggle |
| `pkg/esp/i2c`  | `driver/i2c_master.h` (new driver) | master bus + device handles, write/read/write-read |
| `pkg/esp/i2s`  | `driver/i2s_std.h` (new driver) | std (Philips) mode TX/RX, 16/32-bit, mono/stereo |
| `pkg/esp/net`  | `lwip/sockets.h` + `lwip/netdb.h` | TCP dial/listen/accept/read/write, DNS resolve, bounded quantum waits |
| `pkg/esp/tls`  | `mbedtls/ssl.h` + `mbedtls/x509_crt.h` (4.1.1, PSA crypto) | TLS client dial with SNI/host verification (Config: RootCAs/InsecureSkipVerify/MinVersion; deprecated VERIFY_NONE/REQUIRED TLSDial), record read/write, ConnectionState, close_notify, bounded quantum waits |
| `pkg/esp/ws`   | — (no glue; pure Go RFC 6455 over a caller-supplied `net`/`tls` conn) | HTTP Upgrade handshake, masked frames, ping/pong, closing handshake; stdlib-free SHA-1/base64 |
| `pkg/esp/ble`  | NimBLE host (`host/ble_gap.h`, `nimble_port.h`), or Bluedroid (`esp_bt.h` + `esp_gap_ble_api.h` + `esp_gattc_api.h`) | BLE **central**: bring-up and sync, scan, connect, cancel, disconnect; GATT client (service/characteristic discovery, read, write, CCCD subscribe, notifications, MTU); three-tier event mailbox, generation-tagged Operation/Connection ids. Two interchangeable C glues behind the same ABI (`ble_glue_nimble.c` / `ble_glue_bluedroid.c`), selected by the firmware's sdkconfig |
| `pkg/esp/bt`   | Bluedroid Classic (`esp_spp_api.h` SPP over RFCOMM; `esp_a2dp_api.h` A2DP sink) | Classic Bluetooth bring-up (BTDM controller + Bluedroid host), device name/discoverability, SPP server (listen/accept one serial connection), `SPPConn.Read/Write/Close`, A2DP **sink** with the decoded-PCM audio path kept entirely in C glue (BTC callback → SPSC ring → pump task → I2S; Go does the control plane only), event mailbox with fan-out subscriptions |
| `pkg/esp/lpcore` | `ulp_lp_core.h` + `lp_core_mailbox.h` (ulp component) | LP (low-power) coprocessor: load an LP program into LP RAM (32 KiB ceiling + the project's own reservation), run/stop with any of the six wakeup sources, single-word mailbox send/receive; the LP core keeps running through HP deep sleep (deep-sleep entry stays in the app's own glue) |
| `pkg/esp/opus` | `opus.h` (managed component `78/esp-opus`, vanilla upstream Opus compiled fixed-point `-DFIXED_POINT=1 -DDISABLE_FLOAT_API`) | S16 PCM ↔ Opus packets: encoder (application, bitrate, complexity, frame-duration ctls; lookahead query) and decoder (duration read off the packet, packet-loss concealment on a nil frame); synchronous C-glue calls behind a generation-tagged handle table - no mailbox, no task; negative OPUS_* errors pass through named, binding-side refusals live in the 0x79xx segment |
| `pkg/esp/wakenet` | ESP-SR WakeNet (`model_path.h`, `esp_wn_iface.h`, `esp_wn_models.h`) | Wake-word detection: model-partition enumeration, one engine instance (wn9_hilexin by default), the audio path (I2S RX or a test source) and the 3–6.4 ms/frame detect loop kept entirely on the glue's own FreeRTOS task, detections and task errors as a bounded event ring Go polls, sticky `Detected()`, per-call timing stats |

| `pkg/esp/touch` | `driver/touch_sens.h` (touch hardware v3, `esp_driver_touch_sens`) | touch sensor controller: single-instance bring-up (Init→Config→Start lifecycle), 14 channels mapped onto GPIO6..19, channel data + hardware benchmark reads, active-threshold events (ISR callbacks → static ring → Go poll), benchmark recalibration, calibration flow documented on the package |
| `pkg/esp/temp` | `driver/temperature_sensor.h` (`esp_driver_tsens`) | built-in temperature sensor (±1 °C): install/start/read/stop/uninstall lifecycle, range-span config, float reading carried across the boundary as integer milli-degrees |
| `pkg/esp/nvs` | `nvs.h` + `nvs_flash.h` | non-volatile key-value store: namespaces as read-write handles, typed values (u8/i32/u64/string/blob), Delete/Commit, `Keys()` over the entry iterator, partition-wide `ReadStats()`; explicit-commit durability boundary, static-buffer capacity contract (0x75xx segment) |

| `pkg/esp/espnow` | `esp_now.h` (inside the esp_wifi library; rides a started WiFi driver) | Espressif's connectionless low-latency MAC-layer protocol: bring-up over the wifi package's radio (STA/AP started, no association), peer list (add/delete/known/count, optional per-peer LMK encryption + PMK), `Send` with the driver's synchronous verdict + asynchronous `SendResult.Wait` confirmation per destination MAC, `Broadcast`, recv frames through a static ring + pump goroutine into `SetRecvHandler`, `SetChannel` for unassociated nodes, protocol-version and own-MAC queries |


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

Packages under pkg/esp are kept dependency-free (no third-party code, no
`fmt`/`strconv`; the transports additionally avoid everything stdlib but
`io`/`unsafe`), so each glue file can be adopted independently. The one
intra-family import every package may take is `pkg/esp/esperr` - the
shared error base, pure Go, no glue, no build tags.

## Error model

Every pkg/esp package reports failures as Go `error` values built from
the shared `pkg/esp/esperr` base. The model is Go-idiomatic: each
package defines text-first sentinels, wraps them with an operation
prefix, and callers match with `errors.Is` and inspect with
`errors.As`:

```go
var ErrTooLong = esperr.New("esp/nvs: value too long")   // package sentinel
esperr.Wrap("nvs: set string", nvs.ErrTooLong)           // guard with op context
esperr.Coded(code, name)                                 // forwarded pipeline code
```

- `esperr.New(text)` — a semantic sentinel: binding-side refusals
  (not initialized, bad argument, closed, ...). `Error()` renders the
  text plus the diagnostic code annotation when the sentinel carries
  one (`"esp/nvs: value too long (code 29953)"`).
- `esperr.Coded(code, name)` — a forwarded pipeline code (lwip errno,
  h_errno, negative mbedtls code, esp_err) with its ErrName rendering;
  `Error()` keeps the historical `"NAME (CODE)"` / `"esp error CODE"`
  form, and the code stays inspectable via `.Code` or `esperr.Code(err)`,
  so old code that matched numbers (or named them through `net.ErrName`,
  `tls.ErrName`, `ws.ErrName`, `wifi.ErrName`) keeps working.
- `esperr.Wrap(op, err)` — an operation prefix (`"tls: handshake: ..."`)
  via a standard `Unwrap` chain.

`errors.Is` matches by sentinel identity; every sentinel additionally
carries its historical segment code as the `.Code` annotation, and a
forwarded raw code satisfies `errors.Is` against the transport
sentinels by code:

```go
esperr.ErrClosed     // EBADF 9       - a second Close, or a call on a closed conn
esperr.ErrTimeout    // ETIMEDOUT 116 - a deadline passed
esperr.ErrWouldBlock // EAGAIN 11     - poll-style give-up without a deadline
```

These are re-exported by `net`, `tls` and `ws`. `net`/`tls` `Read`
signal an orderly peer close with `esperr.ErrEOF` (the family's io.EOF
counterpart, kept in-family because the stdlib `io` package drags the
errors/reflect machinery into the bare-metal image; UDP has no EOF -
`(0, nil)` there is a zero-length datagram). I/O shapes follow stdlib:
`Read(p) (int, error)`, `Write(p) (int, error)` with partial writes
reporting `(n, err)`, and `Close` is idempotent (`ErrClosed`).

Deliberate deviations, documented on each package: deadlines stay relative
milliseconds (`SetDeadline(ms int32)`, not `time.Time` - never validated
on the bare-metal target and unneeded by the bounded-quantum model);
`Listener.Accept` without a `SetDeadline` is a poll-style call that gives
up after one quantum with `ErrWouldBlock`; `tls.Write` stays
all-or-nothing (record-layer property). The peripheral and wifi drivers
wrap plain esp_err values (Name empty or from `wifi.ErrName`; the esp_err
family keeps its own timeout code `0x107`, surfaced as `wifi.ErrTimeout`).
Historical code segments, one per package. Since the Go-shape error
migration `errors.Is` matches by sentinel identity (each package owns
its sentinel values), so two packages' errors can never be confused;
the segment codes below remain only as the diagnostic `.Code`
annotation each sentinel carries:

| Segment | Package | Values |
|---------|---------|--------|
| `0x72xx` | `temp` | `ErrNotInstalled` (`0x7201`, the glue's refusal for calls before Install) |
| `0x73xx` | `touch` | `ErrNotInit` (`0x7301`, the glue's refusal for calls before Init) |
| `0x74xx` | `espnow` | `ErrGlueABI` (`0x7401`; `0x7402+` reserved) |
| `0x76xx` | `bt` | `ErrGlueABI`, `ErrNotReady`, `ErrBusy` (`0x7700+` = the A2DP domain; `0x7800+` reserved for HFP) |
| `0x7Bxx` | `ble` | `ErrTimeout`, `ErrNotReady`, `ErrUnsupported`, `ErrBadUUID`, `ErrBadValue`, `ErrBusy`, `ErrBadProperty`, `ErrNotFound`, `ErrTooLong` |
| `0x7C01` | `esperr` | the meta error (a wrapper with no pipeline code) |
| `0x7Dxx` | `ws` | `ErrProtocol`, `ErrClosed`… |
| `0x7Exx` | `tls` | `ErrCertVerifyFailed`, `ErrHandshake`… |
| `0x7Fxx` | stubs & glue contracts | `0x7F00` = the shared `errUnsupported` every `_stub.go` reports; `0x7F01` = `wifi.ErrGlueABI` (glue/binding version mismatch) |

## Go ↔ C boundary rules

These rules are what keep the llgo fiber runtime safe across the link.
They are contract, not style.

1. **Integers and pointers only.** No float values ever cross the Go↔C
   link: llgo's RV32 codegen is soft-float while IDF compiles S31 with the
   FPU (ilp32f), and the artifact pipeline patches the object's float-ABI
   flag precisely because the boundary stays integer-only (see
   `_demo/esp-idf-wifi/idfproj/host/make-artifact.sh` step 4).
2. **Errors cross as raw codes; the Go API wraps them.** The C glue
   signatures return `int32_t` codes unchanged (`esp_err_t` is `int`, lwip
   sets errnos, mbedtls returns negative codes) - mapping codes to names is
   a separate helper (`wifi.ErrName` pattern). The Go API converts at the
   boundary: every package returns the shared `pkg/esp/esperr` error value
   (`*esperr.Error{Code, Name}`), `nil` for success. See "Error model"
   below.
3. **No C→Go callbacks.** esp_event handlers, driver ISR callbacks and the
   like run on their own FreeRTOS tasks. The llgo fiber scheduler is
   single-task global state: an entry from a second FreeRTOS task into Go
   would corrupt it. Never register a C→Go trampoline.
4. **Async events go through a mailbox.** The C side copies events into a
   ring (`wifi_glue.c`'s 16-slot SPSC queue) and Go polls it paced by
   `vTaskDelay(10)` = 100 ms — the delay also keeps the idle tasks fed for
   the watchdog. Overflow drops the newest event and is reported once. The
   ble package refines that shape instead of copying the pacing: three
   rings (control results vs notifications vs scan payload), a bounded
   batch per pass, one real tick of rest, and wait deadlines measured in
   milliseconds (see its section).
   Synchronous driver calls that block on a driver semaphore (a WiFi scan,
   an I2C transfer timeout) are safe: they yield the CPU.
5. **No large stack buffers in glue.** Goroutines run on small fiber stacks
   (2 KiB on this target, `default-stack-size` in the target JSON; the main
   fiber gets 4×). Bulk data (scan records, station lists, DMA-size sample
   buffers) lives in static storage on the C side or flows through Go heap
   slices whose first element is passed as a pointer; small config structs
   built inside one glue function are fine. Record structs that fan out
   through Go channels must stay small for the same reason - see the ble
   package's payload bound.
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

**Error domain.** Errors are esperr values carrying lwip errnos (0 wraps
to nil) - not esp_err, since that is what lwip's socket layer sets - and
`Resolve` returns the netdb `h_errno` family (200–213). Mind the numbering:
the ESP-IDF build compiles picolibc's `errno.h`, whose values above 103
differ from Linux (`ETIMEDOUT` = 116, `EINPROGRESS` = 119, `ENOTCONN` = 128;
`net.ErrName` knows them, and the shim probe printed the firmware's own
values to pin the table). The transport sentinels (`ErrClosed`,
`ErrTimeout`, `ErrWouldBlock`) are re-exported here; Read reports an
orderly peer close as `io.EOF`, and Write reports partial progress as
`(n, err)` instead of the old negative-errno single value.

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
`Dial(addr, timeoutMs, cfg)` connects, resolves the addr host when it is a
name, and handshakes as a client under cfg's policy (`Config{ServerName,
RootCAs, InsecureSkipVerify, MinVersion}`); `TLSConn.Read`/`Write`/`Close`/
`SetDeadline`/`State` then speak plaintext and report the session. The
deprecated `TLSDial(ip, port, timeoutMs, caPEM)` keeps its two legacy
modes (VERIFY_NONE on nil, chain+validity-only on non-nil - neither
verifies the host name) while existing callers migrate. The glue
inventory is deliberately thin:
`llgo_tls_new` (config defaults, min version, CA parse + authmode,
hostname, BIO wiring, ssl_setup), `llgo_tls_handstep` (one
`mbedtls_ssl_handshake_step`), `llgo_tls_read`/`llgo_tls_write` (one
record attempt), `llgo_tls_close_notify`, `llgo_tls_version`/
`llgo_tls_ciphersuite` (static negotiated-name strings),
`llgo_tls_peer_cert` (peer leaf DER: size query, then copy; needs the
project's `CONFIG_MBEDTLS_SSL_KEEP_PEER_CERTIFICATE=y` - IDF's default is
off, mbedtls then drops the certificate after verification and State
reports version/cipher with an empty PeerCerts),
`llgo_tls_free`. Each call is a single
non-blocking attempt; the Go side drives the quantum loops over
`llgo_net_poll` exactly like the net package — the WANT_READ/WANT_WRITE
mbedtls reports map onto the same socket quanta. The shared socket/clock
primitives (`llgo_net_socket`, `llgo_net_connect`, `llgo_net_resolve`,
`llgo_net_poll`, `llgo_net_sock_error`, `llgo_net_shutdown`,
`llgo_net_close`, `llgo_net_now_ms`) are declared by linkname in BOTH
packages and defined once in `net_glue.c`: a TLS program links both
glues, but the packages stay mutually dependency-free at the Go level
(each adopts independently).

**mbedtls 4.1 specifics** (IDF v6.3-dev): the SSL layer's randomness is
PSA Crypto random (`psa_generate_random`), which IDF wires to the hardware
TRNG (`esp_fill_random`) and initializes at startup — no `conf_rng`, and
the ctr_drbg of older mbedtls is a private header in 4.1.1. VERIFY_REQUIRED
requires an explicit `mbedtls_ssl_set_hostname` call or the handshake
fails with `MBEDTLS_ERR_SSL_CERTIFICATE_VERIFICATION_WITHOUT_HOSTNAME`
(-0x5D80). `Dial` always provides one (the ServerName, defaulting to the
addr host) - the name drives both the SNI extension and the CN/SAN match,
so a full-identity check is the default; only the deprecated TLSDial opts
out with `set_hostname(NULL)`: chain + validity are still checked, the
name match is dropped. `mbedtls_x509_crt_parse`'s PEM path needs the NUL
terminator counted INSIDE the length, so the glue parses a heap copy; the
hostname travels the same way. And the first
`handshake_step` only advances HELLO_REQUEST → CLIENT_HELLO returning 0
without network I/O — the glue wraps it so 0 always means "handshake over".

**Error domain.** TLS-phase failures are sentinel esperr errors
(`ErrHandshakeTimeout`/`ErrCertVerifyFailed`/`ErrCaParseFailed`/
`ErrWantPollExhausted`/`ErrHandshake`/`ErrPsaInitFailed`/`ErrNoRootCAs`/
`ErrNoPeerCert`; the underlying 0x7E00 codes are internal, named by
`tls.ErrName`); the TCP phase surfaces lwip errnos exactly like
`net.Dial`; failures inside mbedtls's own dial setup (config defaults,
`ssl_setup`) and record-phase fatals carry the raw negative mbedtls code.
The fail-closed dial rule: `RootCAs` nil with `InsecureSkipVerify` false
is rejected with `ErrNoRootCAs` before any bytes hit the network (bare
metal has no system trust store to fall back on - verification without an
explicit anchor cannot succeed and must not silently degrade to
VERIFY_NONE); the glue refuses the same combination defensively.
Read/Write honor the connection deadline with net.Conn semantics
(ErrWouldBlock poll-read without a deadline, ErrTimeout at expiry, io.EOF
at an orderly session end); TLS Write is all-or-nothing (no partial
plaintext), so "nothing accepted" returns (0, ErrWouldBlock) and the
caller retries the whole buffer. `Close` sends close_notify under a
bounded 500 ms wait (`ErrWantPollExhausted` if it cannot finish), always
releases the fd, and is idempotent (`ErrClosed`).

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
(`Read/Write/Close` in the family's `(int, error)` shapes) - Go interfaces
are structural, so the real transports satisfy it without the package
importing them - plus the optional `Deadliner` interface
(`SetDeadline(ms int32) error`), which `*TCPConn` and `*TLSConn` both
implement as methods. It imports NOTHING but the family's shared error
base `pkg/esp/esperr` (no stdlib at all), so it
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
`ReadMessage` reports `ErrTimeout` and `WriteMessage` (0, `ErrTimeout`);
without one, `ReadMessage` is a poll-style read (`ErrWouldBlock`
after one quantum) whose framing state machine resumes mid-header or
mid-payload on the next call. Errors are sentinel esperr values
(`ErrHandshake`/`ErrNotWS`/`ErrAcceptKey`/`ErrTooBig`/`ErrProtocol`/
`ErrPeerClosed`/`ErrConnDead`/`ErrNoDeadline`, each carrying its internal
0x7D00-segment code, named by `ws.ErrName`
alongside the lwip errno table); transport failures pass through
unchanged, and calls on a closed session report ErrClosed like the
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
automatically inside the read path; a pong is reported as a `(0, nil)`
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
`_demo/esp-idf-wifi/goapp/`, one file per probe area) runs against
`idfproj/host/wssrv` - a host peer that sniffs each connection and
serves the raw echo, the TLS echo and both WebSocket endpoints on the
same two ports (9000/9443), so one process backs the whole
wifi->net->tls->ws acceptance chain.

## pkg/esp/ble — BLE central with a GATT client, two interchangeable host stacks

`pkg/esp/ble` exposes a BLE **central** over a host stack chosen at firmware
build time: `Init` (controller init + BLE enable + host init, the host task,
and the wait for the stack's sync event), `Ready`, `Deinit`, `StartScan` /
`StopScan` (legacy discovery: interval, window, passive, duplicate filter,
optional duration), `Connect` (bounded by the controller's own attempt
timeout), `Cancel` and `Disconnect`. `Subscribe` / `Subscription.Next`
deliver the stack's asynchronous results. On top of the connection sits a
GATT client: `Services` / `Characteristics` (16- and 128-bit UUIDs,
normalized to the Bluetooth base form so both spellings of the same UUID
compare equal), `Read`, `Write` (with response or not, chosen from the
characteristic's properties or picked explicitly), `SubscribeChar` /
`Unsubscribe` (write the CCCD, deliver notifications through the ordinary
fan-out) and `MTU` (the glue requests the exchange right after a connection
comes up). `_demo/esp-idf-ble` drives the whole surface under esp-emu
against a Bumble GATT server.

The **peripheral role** shares the same package and ABI: advertising
(`StartAdvertising` composes the AD payloads in Go - flags, complete or
shortened local name, caller-supplied structures - and the glue pushes them
through the stack's legacy raw-data chain), the local GATT server (a Go-side
registry built by `AddService`/`AddCharacteristic` and pushed once by
`CommitServices`; the glue holds static declaration tables and one value
cache per characteristic), the pairing posture (`SetSecurityParams`), and
the peer's actions as events - `EvPeriphConnect`/`EvPeriphDisconnect` on the
control tier, `EvCharWrite`/`EvSubscribed` on the peripheral's own payload
ring, so an advertising or write flood can never displace them. A peer's
read is answered straight from the value cache in the glue's callback - the
GATT server must answer synchronously on the stack's task, where the llgo
scheduler cannot be entered - and `WriteCharValue` fills the same cache from
Go. `Notify`/`Indicate` send server-side updates; `LocalHandle` maps a UUID
to the handle the stack assigned. `_demo/esp-idf-ble-periph` drives the whole
peripheral surface under esp-emu against a Bumble central (Bluedroid backend,
the path whose server half is fully implemented; the NimBLE sibling
implements the same ABI over `ble_gatts_add_dynamic_svcs` and needs
`CONFIG_BT_NIMBLE_DYNAMIC_SERVICE=y`).

**The Go API is backend-neutral; the sdkconfig picks the stack.** IDF ships
two mutually exclusive BLE hosts and a firmware selects exactly one
(`choice BT_HOST`). The package compiles against a single C ABI - the
`llgo_ble_*` symbols, the 188-byte event record, the four-ring mailbox and
every event kind - and two glue files implement it:
`glue/ble_glue_nimble.c` (NimBLE) and `glue/ble_glue_bluedroid.c`
(Bluedroid; Classic Bluetooth needs Bluedroid, so BLE-only Bluedroid
firmware keeps the door open for a later `pkg/esp/bt` profile in the same
image). The project copies ONE glue into its main component; each file's
first `#if` turns "configured the other stack" into a compile-time `#error`
naming the right file. No Go build tags: the Go build runs before the IDF
configuration exists, so two switches could only drift apart. Semantic
differences (Bluedroid's synchronous `esp_bluedroid_enable`, its folded
discovery model, NO_RSP as a write type rather than a separate call, MSB-first
BD addresses, no per-procedure error callbacks on disconnect) are absorbed
inside the glue; `ble.go` is unchanged between backends, and `Backend()`
reports which one is linked. Both backends pass the same demo golden under
esp-emu (the parity runs against the NimBLE baseline golden are in the
demo's README).

**Control calls never block.** Each command is queued and executed on the
NimBLE host task through the host's own event queue (`ble_npl_eventq_put` on
`ble_hs_evq_get()`), so no GAP or GATT call ever runs on the goroutine that
issued it and the llgo scheduler is never held across a driver wait. The Go
side gets an `OpID` back immediately; the outcome - success, a refusal, or a
controller result - arrives as an event. A returned error therefore only
means "could not even be submitted" (`ErrNotReady`, a full command queue;
`ErrBusy`, another GATT procedure still in flight - the ATT bearer carries
one outstanding request anyway), never a forged result. GATT calls are
synchronous for the caller: each submits, then waits (bounded by
`GattTimeoutMs`) for the mailbox record that completes its `OpID`.

**Three-tier mailbox.** Events split by rate and size. The 8-slot control
ring carries sync, reset, scan-done, connect, disconnect, command
rejections and the GATT completions (discovery status, read values, write
status, the negotiated MTU). The 8-slot notification ring carries
server-pushed notifications and indications - 134 bytes per slot, payload
bounded by `MaxData` = 128 bytes, the largest record that fits the
target's 2 KiB goroutine fiber stacks across every hop a fan-out record
takes (a notification larger than `MaxData` is dropped whole and counted,
never truncated; a read larger than that reports `BLE_HS_EMSGSIZE` - raise
`-goroutine-stack-size` and the glue's `BW_DATA_MAX` together to lift it).
The 16-slot scan ring carries scan results (address, type, RSSI, bounded
31-byte legacy payload). All three are single-producer (the host task)
single-consumer (the poller); ring indices are published with C11
`__atomic` release stores and consumed with acquire loads, so the records
stay intact regardless of core assignment or compiler reordering. A flood
in one tier cannot starve another. Overflow drops the newest record and is
reported once as `EvOverflow` with the dropped count. One poller goroutine
drains up to 32 records per pass, hands the scheduler away, then rests
exactly one FreeRTOS tick, and fans every record out to all matching
subscriptions - a slow subscriber loses its own events, never anyone
else's. A bounded `Next` measures its deadline in milliseconds against the
glue's timer (accuracy one tick), not in quanta. Discovery results
themselves are not mailbox traffic: they land in C-side fixed tables (16
services / 32 characteristics, a peer with more ends the discovery with
`BLE_HS_EMSGSIZE` rather than losing records silently) that the Go side
copies out on completion.

**Generation-tagged ids.** `OpID` comes from a monotonic counter and is never
reused, so a late completion from an earlier scan or connect cannot be
mistaken for the current one. `ConnID` packs the controller's HCI connection
handle with a generation counter (`(generation << 16) | handle`), because
handles are reused: a `ConnID` from a previous connection is refused with
`StatusENOENT` instead of addressing whatever that handle became.

**Three error spaces.** `esperr.Error.Code` stays inspectable and `errors.Is`
matches by code, so the package deliberately separates them: the esp domain
(raw esp_err, below 0x4000 - `Init`/`Deinit`), the nimble domain
(`NimbleBase` 0x8000 | status - every BLE_HS_*, ATT 0x1xx, HCI 0x2xx and L2CAP
0x3xx status as the stack reports it; `Status` extracts the raw value and
`ErrStatus` rebuilds the error value), and this package's own 0x7Bxx codes
(`ErrTimeout`, `ErrNotReady`, `ErrUnsupported`, `ErrBadUUID`, `ErrBadValue`,
`ErrBusy`, `ErrBadProperty`, `ErrNotFound`, `ErrTooLong`). NimBLE's host
status 0x02 (`BLE_HS_EALREADY`), an HCI error 0x02 and an esp_err 0x02 are
three different conditions, and the tag is what keeps them apart.

**Lifecycle is blocking, and that matters.** `Init` and `Deinit` are the only
calls that hold the scheduler: `nimble_port_init` brings the controller up, and
`nimble_port_stop` waits for the host task to service the stop (it must not run
on the host task itself). One measured trap is worth keeping: on this chip
NimBLE's `ble_npl_event` wraps `btdm_osal_event` (`nimble_npl_os.h`), so every
initialized event holds one element of the controller's `s_btdm_osal_ev_pool`.
The glue's command event is returned with `ble_npl_event_deinit` before
`nimble_port_deinit` - left out, `esp_bt_controller_disable` asserts in
`btdm_osal_elem_mempool_deinit`, which under esp-emu shows up as an abort right
after the last print of the program.

**Unsupported paths are honest.** A host build returns `ErrUnsupported` from
every call. A firmware compiled without the NimBLE host still links a program
that imports this package (the glue's other branch defines every symbol) and
reports `ESP_ERR_NOT_SUPPORTED`, surfaced as `ErrUnsupported` - verified under
emp-emu with a `CONFIG_BT_ENABLED=n` build of the demo.

**Integration.** The main component needs `REQUIRES bt esp_timer nvs_flash`
(NimBLE headers, the deadline clock, and the bond store's partition, which the
glue initializes but never formats). `_demo/esp-idf-ble` is the worked example,
including the host-side Bumble peripheral that gives the emulator something to
discover.

**Verification status.** Covered under esp-emu 1.2.0 (the emulator intercepts
the S31 BTDM HCI transport through the ELF and forwards it to a Bumble virtual
controller): controller init/enable, host sync, discovery, connect, GATT
service/characteristic discovery, read, write with and without response,
CCCD subscribe/unsubscribe, delivered notifications, the negotiated MTU,
terminate, the mailbox and fan-out, and clean shutdown, all driven by
`_demo/esp-idf-ble`'s probe with cross-checks against the peer's own logs
(subscription state changes and written values seen by the Bumble server).
**Not** covered and claimed now: the radio itself (RF calibration, real
RSSI, channel map, coexistence), pairing/bonding, the indication path (the
code shares the notification ring and the host sends the ATT confirmation,
but the emu peer only notifies), payloads beyond `MaxData` (drop and count
is implemented, never exercised with real oversized traffic), descriptor
discovery, long read/write, more than one connection, flash/cache
operations concurrent with BLE activity, and controller event-pool behavior
under real interrupt load. Two measured emulator gaps narrow what the run can
say: an unanswered connect attempt never times out there (the virtual
controller does not model the controller-side connect-duration limit), and the
host peer must advertise legacy-only, so extended advertising and its 1650-byte
payloads are out of reach.

## pkg/esp/espnow — Espressif's connectionless ESP-NOW protocol over a started WiFi driver

`pkg/esp/espnow` exposes the esp_now stack (which lives inside the esp_wifi
library and rides its driver): `Init` / `Deinit`, `SetChannel`, the peer
list (`AddPeer(PeerConfig)` / `DeletePeer` / `PeerIsKnown` / `PeerCount`),
sends (`Send(peer, data) (*SendResult, error)`, `Broadcast`), the receive
side (`SetRecvHandler(func(from [6]uint8, data []byte))`) and the
crypto entry points (`SetPMK`, `SetLMK`). `Version` reports the ESP-NOW
protocol generation the firmware speaks; `OwnMAC(ifIdx)` returns the
address a peer must add; `MACString` renders MACs; `Dropped` reports the
ring's honest loss counter.

**Bring-up order is the caller's contract.** esp_now needs the WiFi driver
initialized AND started (station or soft-AP, unassociated is fine - that is
the protocol's usual mode), but this package deliberately does not touch
esp_wifi's lifecycle: pkg/esp/wifi owns the radio. The program runs
`wifi.Setup() -> wifi.SetModeSTA() -> wifi.Start()` first and calls
`espnow.Init()` after; `_demo/esp-idf-espnow` is the runnable example.
`esp_now_init` before the driver runs reports ESP_ERR_ESPNOW_INTERNAL
(0x306a) - Init surfaces it as any other driver error.

**Callbacks → static ring → pump.** Both driver callbacks (frame received,
send confirmed) fire on the WiFi task - the same no-callbacks-into-Go,
no-malloc-in-callbacks discipline every async pkg/esp package follows (the
WiFi task must never enter the llgo fiber scheduler, and llgo's --wrap
redirects malloc/free to the lock-free tinygogc allocator that only the
llgo task may enter). The glue memcpy's into a 16-slot static SPSC ring
(16 × ~276 B ≈ 4.3 KiB .bss; slots hold the full 250-byte v1.0 payload),
and a single pump goroutine - started once, when Init succeeds, never on a
host build - drains it paced by `vTaskDelay(10)` = 100 ms quanta, the
wifi package's exact cadence: drain, dispatch, `Gosched`, rest.
Overflow drops the newest record and counts it; `Dropped()` reads the
counter (there is no callback for overflow - the recv handler's signature
carries frames, and a loss counter is the honest form).

Event kinds (this package's own table, starting at 1 - never aliased with
wifi's or ble's kinds; every pkg/esp event table belongs to its package):

| kind | value | record | goes to |
| --- | --- | --- | --- |
| `EvRecv` | 1 | source MAC, RSSI in Aux, payload in the pump's staging window | `SetRecvHandler(from, data)` - data is a fresh copy per delivery |
| `EvSendDone` | 2 | destination MAC, `esp_now_send_status_t` in Aux | the pending `SendResult` for that MAC (oldest first) |
| `EvOverflow` | -1 | the ring dropped records | the `Dropped()` counter (no callback) |

**Send splits into two verdicts, like the driver.** `esp_now_send` is
synchronous and only queues the frame; the ACK-level outcome arrives later
on the send callback. `Send` returns the driver's synchronous verdict as
its error (`ESP_ERR_ESPNOW_NOT_FOUND` for an unregistered peer, `ARG` for
a payload outside 1..MaxDataLen, `NO_MEM` when the driver's send queue is
momentarily full, `IF`/`CHAN` for an interface/channel mismatch) and a
`*SendResult` whose `Wait(timeoutMs)` resolves the asynchronous one
(`(SendSuccess, nil)` / `(SendFail, nil)` / `(SendFail, ErrTimeout)`,
bounds measured in 100 ms quanta). The callback carries no sequence id -
only the destination MAC - so results resolve oldest-first per MAC (the
driver serializes a peer's frames, so statuses arrive in send order).
Resolve every result with a bounded Wait: a result whose callback never
fires keeps its place at the head, and the next confirmation resolves it
first - but a Wait that timed out leaves its channel in place, so the
queue never mis-delivers into a Wait that already returned. A zero-value
or nil result `Wait`s to `ErrTimeout` instead of hanging.

**Peers and encryption.** `AddPeer` takes a `PeerConfig{MAC, Channel, If,
Encrypt, LMK}` - the zero value beyond the MAC inherits the radio's state
(the broadcast peer is exactly that). `Encrypt` without a non-zero LMK is
refused client-side with ESP_ERR_ESPNOW_ARG (the driver would accept it
and silently send plaintext). Encrypted peers consume the Kconfig
`CONFIG_ESP_WIFI_ESPNOW_MAX_ENCRYPT_NUM` slots (the driver is the
authority; a surplus reports ESP_ERR_ESPNOW_FULL); `SetPMK` arms the
primary master key that protects the LMKs on the air, `SetLMK` rotates a
peer's key. Key agreement/provisioning is out of scope. Not exposed (the
driver has it; add when needed): per-peer PHY rate
(`esp_now_set_peer_rate_config`), the v2.0 frame format (`Version`
reports the generation; Send stays at the v1.0 250-byte ceiling so both
generations receive), the switch-channel/remain-on-channel pair, wake
window, user OUI.

**Errors.** The family's esperr model: raw codes wrapped with names from
`espnow.ErrName` (the shared esp_err family, the WiFi driver codes
SetChannel can surface, and the whole ESP_ERR_ESPNOW segment 0x3065..0x306D).
The package's own internal code is `0x7401` (`ErrGlueABI`, the
stale-glue-template guard Init checks via `llgo_espnow_glue_abi`); the
stub's `0x7F00` errUnsupported is the family-wide value.

**Emulator fidelity (esp-emu 1.2.0, measured with a pure-C probe first,
logs in the demo README).** The esp_now API surface is modeled: init,
deinit, re-init, callback registration, peer management (including the
EXIST/FULL/NOT_FOUND/ARG refusals and the encrypted-peer slot accounting),
PMK/LMK calls, `esp_wifi_set_channel` while unassociated, and broadcast
sends - `esp_now_send` queues them and the send callback fires with
ESP_NOW_SEND_SUCCESS. What an emulator cannot do is OTA frames: two
esp-emu instances do not hear each other (both sides of a two-instance
broadcast run see their own sends confirmed and recv = 0), and there is no
virtual ESP-NOW peer. The receive path and true delivery semantics
(collision behavior, real RSSI, ACK outcomes for unicast to a live peer,
encrypted frames over the air, range/rate) are hardware items - see the
demo README's checklist. The demo probe runs the whole control plane
through the binding under esp-emu; it does not pretend a frame was
received.

**Integration.** The main component needs
`REQUIRES esp_wifi esp_netif esp_event nvs_flash` (the esp_now header
ships in the esp_wifi component; the wifi glue brings NVS/netif/event-loop
up before espnow.Init). `_demo/esp-idf-espnow` is the worked example.

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
`_demo/esp-idf-wifi/goapp/` (`[go] periph ...` console lines, and
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

## pkg/esp/bt — Classic Bluetooth (SPP + A2DP sink) over the Bluedroid host

`pkg/esp/bt` brings up the BTDM controller and the Bluedroid host for
**Classic Bluetooth**: `Init` / `Deinit` / `Ready` / `Backend()`, `SetName`,
`SetDiscoverable`, and an SPP server: `SPPListen(SPPOptions{SCN, Name})`
registers the Serial Port Profile (SDP record + RFCOMM listener), and
`SPPAccept(timeoutMs)` waits for the connection event and returns an
`*SPPConn` with `Read(p, timeoutMs) / Write(p, timeoutMs) / Close()`,
`Handle()` and `AddrString()` (Bluedroid order: most-significant octet
first). After a connection closes the server keeps listening - the next
`SPPAccept` catches the following peer. Events (`EvSPPInit`, `EvSPPListen`,
`EvSPPConnOpen`, `EvSPPConnClose`, `EvSPPCong`, `EvSPPStop`, `EvUninit`,
`EvACLDisconnect`, `EvOverflow`) arrive through the same Subscribe /
fan-out mailbox shape the ble package uses. `Read` reports a peer close as
`esperr.ErrEOF` after the buffered bytes are drained first; `Write` shards
payloads to the RFCOMM MTU (990 B) and honors the stack's congestion
signal (`EvSPPCong` edges retry a `NO_MEM` write, deadline-bounded). The
package's own sentinel codes live in the 0x76xx segment (`0x7700+` = the
A2DP domain, `0x7800+` reserved for HFP).

**SPP data bypasses the mailbox.** The control ring carries lifecycle
events only (8 slots, ~192 B records); serial data lives in its own ring of
8 x 1 KiB static slots handed over per-slot by a release/acquire `used`
flag, and `Read` pulls it with one C call - kilobyte payloads never cross
the 2 KiB goroutine fiber stacks or a channel.

**Lifecycle includes a shutdown handshake.** Bluedroid's `esp_spp_deinit`
only posts its stop request, and `esp_bluedroid_disable` tears down the
host thread while ACL disconnects may still be completing - a disconnect
event landing after that free aborts (`osi_thread_post` assert), and a dead
peer hangs `disable` forever. The glue therefore owns the teardown: an
UNINIT semaphore handshake with the BTC task, and an ACL-link counter
(`EvACLDisconnect` decrements it) that `Deinit` waits on (bounded) before
disabling the host. Measured whole-stack cost on the demo probe: the BT
stack holds ~70.5 KB of internal SRAM while active (controller OSAL pools
resist PSRAM placement); Go heap lives in PSRAM as usual.

**Integration.** The main component needs the Bluedroid side of the `bt`
component (`REQUIRES bt esp_timer nvs_flash`) with
`CONFIG_BT_BLUEDROID_ENABLED=y`, `CONFIG_BT_CLASSIC_ENABLED=y` and
`CONFIG_BT_SPP_ENABLED=y` (the glue's `#error` guards name the missing
switch). `_demo/esp-idf-bt` is the worked example, including the vendored
`bumble_classic_peer.py` the emulator pairs against (SDP browse, RFCOMM
DLC, a payload echo, then a deliberate ACL disconnect so `Deinit` has
something finishable to wait for).

**A2DP sink — the audio path never enters Go.** `A2DPEnable(A2DPConfig{pins,
DMA sizes})` starts the sink profile and the AVRCP TG/CT pair and brings up
an I2S std TX channel on the given pins. From there the decoded audio flows
entirely inside the glue: the BTC task's data callback (the IDF stack
decodes SBC internally - the callback receives PCM S16 with the sample
tate and channel count negotiated in the AUDIO_CFG event) copies into a
32 KiB SPSC ring and wakes a dedicated pump task, which reconfigures the
I2S clock/slots on a codec change and writes the DMA queue. The callback
path never blocks - a full ring drops the whole block and counts it
(`A2DPStats()` exposes total/written/dropped) - because blocking the BTC
task stalls the whole Bluedroid host. Go only drives the control plane:
enable/disable, absolute volume over AVRCP CT (`A2DPSetVolume`, the
peer-side volume arrives as `EvA2DPVolume`), the four event kinds
(EvA2DPConnected/Disconnected/AudioCfg/Volume = 9..12, reusing the existing
`Event` fields: Aux carries the sample rate, the channel count or the 0..127
volume; the AUDIO_CFG event may arrive before EvA2DPConnected - the BTC
queue order - and `AudioCfg` pins both values). The glue lives in a
sister file, `glue/bt_glue_a2dp.c`, next to `bt_glue_bluedroid.c`: the two
share the mailbox (`ctrl_push`, `s_ready`) and the lifecycle (the A2DP
profile-down joins the common teardown handshake), so a project copying
the bt glues copies both (copying one is a loud link error, not silent
breakage). Needs `CONFIG_BT_A2DP_ENABLE=y`. The emulator run drives the
real path end to end (a Bumble AVDTP peer streams SBC; the probe pins the
decoded byte count exactly - 3,525,120 bytes = 44.1 kHz × 2 ch × 2 B ×
20 s) - the radio-level audio qualities (listening quality, clock-drift
glitches over long playbacks, AVRCP volume round-trips) stay physical-board
items.

## pkg/esp/nvs — non-volatile key-value storage over the IDF NVS layer

`pkg/esp/nvs` exposes ESP-IDF's NVS store: `Init` (nvs_flash_init on the
default partition, idempotent, with the driver's erase-and-retry recovery
for a partition it cannot mount - an erase that wipes every namespace and
only runs when nothing was readable), `Deinit`, and per-namespace handles:
`Open(name)` returns a read-write `*Namespace` whose `GetU8`/`GetI32`/
`GetU64`/`GetString`/`GetBlob`/`SetU8`/`SetI32`/`SetU64`/`SetString`/
`SetBlob`/`Delete`/`Commit`/`Keys` cover the store's surface, plus a
partition-wide `ReadStats()` ledger (used/free/available/total entries,
namespace count). Handles follow the i2c idiom: a method on a nil, zero or
Closed `Namespace` reports ESP_ERR_INVALID_ARG, `Close` is idempotent and
reports the family's `esperr.ErrClosed` afterwards. Every call is a
synchronous NVS access (a cache hit or one flash read) - no mailbox, and
the default goroutine stacks hold.

**Commit is explicit.** IDF stages every write in RAM and writes flash on
nvs_commit; the API keeps that boundary honest instead of hiding it behind
an auto-commit. The alternative was weighed and rejected: auto-committing
per Set multiplies page writes for what is logically one durable batch
(flash wear), and the convenience is one line of caller code (`Set`;
`Commit`). Callers batch, then Commit once.

**Capacity is a binding contract, not an IDF limit.** GetString/GetBlob
read through the glue's static value buffer (512 B) and Keys through a
static key table (32 names) - the glue allocates nothing (no malloc/free
across the llgo link). Longer values and larger namespaces are REFUSED
with the package's own 0x75xx-segment codes (`ErrTooLong`, `ErrTooManyKeys`)
rather than truncated or misreported: raw esp_err passes through otherwise
(the ESP_ERR_NVS_* family 0x1101..0x1110, named by `nvs.ErrName`), and
these two conditions have no honest IDF equivalent. Writes have no
binding-side cap; IDF's own ESP_ERR_NVS_VALUE_TOO_LONG applies past what a
page holds.

**One measured deviation from the IDF docs, pinned by the demo probe:**
reading a key back under a different primitive shape (an i32 read of a
u8-stored key) reports ESP_ERR_NVS_NOT_FOUND, not the documented
ESP_ERR_NVS_TYPE_MISMATCH - `nvs_page.cpp`'s hash-list indexes items by
(namespace, type, key), so a mismatched read misses the hash and
short-circuits before the page scan that would classify it (IDF v6.3-dev;
the nvs.go docs say so, and `_demo/esp-idf-nvs` pins the code).

**Emulator fidelity (esp-emu 1.2.0, measured through the binding by
`_demo/esp-idf-nvs`).** NVS is modeled faithfully for the whole surface:
init/format on fresh flash, the typed round-trips, the entry iterator, the
stats ledger - and, the point of the demo, flash persists across
`esp_restart` (the probe walks write → verify → re-verify across two soft
restarts and gates the run on PASS) and across emulator processes via
`--save-state` (the whole 4 MiB flash image round-trips through the
firmware file; the committed entries are visible in the file at the NVS
partition offset). Not modeled and not claimed: flash wear (erase cycling,
page-failure recovery under power loss), NVS encryption, multi-partition
setups, concurrent access from a second task.

**Integration.** The main component needs `REQUIRES nvs_flash`; copy
`glue/nvs_glue.c` into the project's main component (WHOLE_ARCHIVE), the
same two recipes as every package here.
