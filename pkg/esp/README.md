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
