# llgo × ESP-IDF WiFi pipeline (ESP32-S31)

A self-contained pipeline that drives the ESP-IDF `esp_wifi` stack from
Go code compiled by llgo: setup → start → scan → connect → DHCP, with the
acceptance run automated under esp-emu. The Go bindings live in the llgo
repository (`pkg/esp/{wifi,gpio,i2c,i2s,net}`), consumed through the module
require + replace in `go.mod`; this directory owns the acceptance program
and the IDF project. The Go heap's growth into PSRAM that the acceptance
program exercises is an llgo runtime feature, described under
[PSRAM and the Go heap](#psram-and-the-go-heap).

## Layout

```
esp-idf-wifi/
├── go.mod                  Go module (module llgoidf; requires + replaces
│                           github.com/xgo-dev/llgo, the binding packages' home)
├── goapp/main.go           Acceptance program (println + WiFi + Go heap probe
│                           + GPIO/I2C/I2S peripheral probes + TCP net probe)
├── goapp/hosted.go         Host-build push-linkname stubs for the FreeRTOS/C/
│                           tinygogc externs in main.go, so `go build` links
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt
    ├── sdkconfig.defaults  8 KiB main task, 240 MHz, WiFi tasks on CPU1, PSRAM
    ├── host/               Host-side artifact pipeline (one command)
    │   ├── make-artifact.sh  llgo build -> IDF-linkable relocatable object;
    │   │                     also syncs the pkg/esp/*/glue templates into main/
    │   ├── llgo_syms.S       GC heap + bootstrap stack window providers
    │   └── echosrv/          Host-side TCP echo peer for the net probe's emu
    │                         acceptance run (listens on 127.0.0.1:9000; the
    │                         emulator's NAT forwards 192.168.4.1:9000 to it)
    └── main/               The main component
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── app_main.c      app_main -> llgo entry; _write/_exit shims; Go
        │                    heap PSRAM budget (llgo_gc_ext_heap_cap)
        ├── demo_glue.c     Demo-local glue (cycle counter for the benchmarks)
        ├── *_glue.c        Package glues, synced from
        │                    github.com/xgo-dev/llgo/pkg/esp/*/glue at build time
        │                    (gitignored; boundary rules: pkg/esp/README.md)
        └── llgo_dram.ld    _globals_start/_globals_end for tinygogc
```

The Go binding packages themselves live in the llgo repo:
`github.com/xgo-dev/llgo/pkg/esp/{wifi,gpio,i2c,i2s,net}` - the wifi one grew
out of this demo's former `goapp/espwifi` directory (same API, same
mailbox glue; see `pkg/esp/README.md` for the package template, the
go↔C boundary rules and both glue integration recipes).

## Architecture

ESP-IDF/FreeRTOS owns the platform; llgo compiles `goapp` into a
relocatable object (`-target esp32s31-idf`, `ld.lld -r`) that the main
component links into the firmware. `app_main` calls the generated C entry,
which hands package init and `main.main` to the llgo fiber scheduler on the
FreeRTOS main task.

The Go ↔ C boundary follows three rules:

1. **Integer-and-pointer-only signatures.** llgo's RV32 codegen is
   soft-float; the object's ABI flag is patched to single-float because no
   float value ever crosses the link (see `make-artifact.sh` step 4).
2. **Events travel through a C-side mailbox, never callbacks.** esp_event
   handlers run on the event-loop task; the llgo fiber scheduler is
   single-task global state, so no second FreeRTOS task may enter Go. The
   handler copies into a 16-slot single-producer/single-consumer ring and Go
   polls it paced by `vTaskDelay(10)` = 100 ms — which also keeps the idle
   tasks fed for the watchdog. Overflow drops the newest event and is
   reported once.
3. **No large stack buffers in glue.** Go goroutines run on 8 KiB fiber
   stacks, so bulk data (scan records) lives in static storage, and
   `WIFI_INIT_CONFIG_DEFAULT()` stays in C (it embeds Kconfig constants and
   function pointers Go cannot spell).

The full discipline (handles as uintptr, linkname naming, the package
template, both glue integration recipes, how to add a new peripheral) is
fixed in [`pkg/esp/README.md`](../../pkg/esp/README.md).

Every binding returns the raw `esp_err` code (0 = ESP_OK);
`wifi.ErrName` maps the codes this pipeline can hit to their IDF names.

### Why the main component is WHOLE_ARCHIVE

`llgo_out.o` is the last item on the final link line and the ESP-IDF link
uses no `--start-group`, so each archive is scanned only against the
undefined symbols known at its position. The glue (`wifi_glue.c`) is
referenced solely from `llgo_out.o`, and a plain archive would never pull
it — the same reason the `llgo_syms.S` runtime providers are merged inside
`llgo_out.o` itself. `WHOLE_ARCHIVE` keeps every main-component object in
the link unconditionally. (The baseline demo resolved `vTaskDelay` from
`libmain.a` only by accident: `app_main.c` itself references FreeRTOS
symbols and was pulled early by the IDF startup, and its undefineds then
pulled `libfreertos.a` in a later archive round.)

## Build and run

Prerequisites: llgo compiler (`/tmp/llgo-wifi` below), Docker image
`espressif/idf:latest`, `esp-emu` v1.2.0 on the host (the first release
with the user-mode NAT the net probe relies on).

```
# 0. Station credentials: copy the placeholder and fill in myssid/mypassword
#    (the emulator's built-in soft AP). Without this file the glue compiles
#    empty strings and the run stays scan-only: it never associates, so a
#    --exit-on "[go] net PASS" acceptance would run to the 150 s timeout
#    instead of exiting.
cp idfproj/main/wifi_creds.h.example idfproj/main/wifi_creds.h
#    ... edit in myssid / mypassword ...

# 1. Go -> relocatable object (prints the remaining undefined symbols;
#    llgo_wifi_* / llgo_gpio_* / llgo_i2c_* / llgo_i2s_* / llgo_net_* /
#    vTaskDelay / stdio must all be in that list - the script echoes the
#    llgo_net_* set separately). It also syncs the pkg/esp/*/glue templates
#    into main/ - run it before every idf.py build.
cd idfproj/host && ./make-artifact.sh /tmp/llgo-wifi

# 2. IDF firmware (merge-bin takes -o relative to the build/ directory:
#    the image lands in build/merged-binary.bin)
cd .. && docker run --rm -v "$PWD":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Back at the demo root: host-side echo peer for the net probe (echoes
#    everything back until the firmware closes; see idfproj/host/echosrv).
#    The firmware connects to the emulator's NAT gateway alias
#    192.168.4.1:9000, which lands on this listener.
cd .. && go run ./idfproj/host/echosrv 127.0.0.1:9000 &

# 4. Acceptance run under esp-emu (emu's built-in soft AP is myssid/mypassword;
#    --psram-size matches the board's 16 MiB PSRAM, without it the Go heap
#    cannot grow). The PASS gate exits as soon as the net echo verifies;
#    a full-coverage run (resolve + loopback + every heap mark) exits on
#    "[go] wifi done" instead - the PASS-gate string kills the run before
#    those best-effort items execute.
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware build/merged-binary.bin \
    --console usb-jtag --timeout 150s --psram-size 16M \
    --wifi-ssid myssid --wifi-password mypassword \
    --exit-on "[go] net PASS"
#    ... or: --exit-on "[go] wifi done"  (full coverage)
rmdir /tmp/esp-emu.lock
kill %1   # echosrv
```

Subnet note: the probe hardcodes the emulator's NAT addresses
(192.168.4.1 gateway alias, station 192.168.4.2), which are esp-emu's
defaults. esp-emu v1.2.0 prints a startup WARN that a future release
changes the default subnet to 10.0.2.0/24 (`the emulator's network
defaults to 192.168.4.0/24 ... a future release changes the default to
10.0.2.0/24. Add net=192.168.4.0/24 to --net to keep it, or
net=10.0.2.0/24 to switch now`); if that lands, pass
`--net 192.168.4.0/24` or the probe's hardcoded addresses stop matching.

Expected console output (interleaved with the IDF boot log):

```
[c] wifi glue: IDF v6.3-dev-..., free heap ... bytes
[go] wifi setup err=0
[go] wifi STA_START
[go] wifi scan n = 1
[go] wifi ap ssid=myssid ch = 1 rssi = -30
[go] wifi scan done
[go] wifi STA_CONNECTED
[go] wifi GOT_IP ip=192.168.4.2
[go] wifi PASS ip=192.168.4.2
```

`main.go` runs the whole acceptance sequence: one-shot setup, event mailbox,
station config (`myssid`/`mypassword`), start, blocking scan with the AP list
print, connect with up to three attempts (disconnect reason codes arrive
through the mailbox), and the DHCP wait. A wrong password ends in three
`[go] wifi disconnect reason = 202` lines (auth fail) and
`[go] wifi FAIL no_ap`.

After the heap probes, the scan-only run exercises the three peripheral
packages (`[go] periph ...` lines): a GPIO set→get→toggle round-trip on
GPIO4, an I2C master bus install plus one read from an absent address (the
emulator does not model ACK/NACK, so the read reports `err = 0` and a
`0xFF` byte there; on hardware it fails with an explicit error within the
1 s driver timeout), and an I2S TX channel
alloc → std init → enable → one silence write. Each probe prints its own
esp_err code; the `probeGPIO`/`probeI2C`/`probeI2S` switches in `main.go`
skip a peripheral that hangs a given target.

After the association (the net probe needs the DHCP lease), `netProbe`
exercises `pkg/esp/net` over the emulator's NAT (`[go] net ...` lines):
it dials the NAT gateway alias 192.168.4.1:9000 - which the emulator
forwards to the host's 127.0.0.1:9000, i.e. the echosrv above - writes a
256-byte pseudo-random pattern, reads it back and compares; that echo is
the `[go] net PASS` gate and the acceptance exit marker. Two best-effort
items follow: `Resolve("localhost")` through the emulator's DNS proxy,
and a strictly sequential loopback Listen/Accept self-connect on
127.0.0.1. Every wait is bounded (20 ms select quanta under explicit
deadlines), so a missing peer surfaces as an errno, never a hang; see
`pkg/esp/README.md` for the bounded-quantum model and the scheduling
note that keeps this probe on one goroutine.

Without credentials the run is scan-only: after the scan, and with PSRAM
enabled, the program keeps 6 MiB live in the Go heap with a collection
after every block, verifies it, starts goroutines on top of it, and
returns:

```
[go] wifi: scan-only, no credentials compiled in
...
[go] wifi scan done
[go] bench static heap: 4x 16 KiB | internal blocks =4 at2f028680(alloc102196write98313read114700cycles) | psram blocks =0 | gc cycles = 3764814
[go] psram: budget = 75% of the free spiram (auto)
[go] gc heap before psram probe usable = 258096 reserved = 262144 segments = 1 | psram bytes = 0 (c side = 0 )
[go] psram: live = 6291456 bytes in 96 blocks | checksum = 16ca9dc5 mismatches = 0
[go] psram: 6 MiB live and verified across collections
[go] bench psram segments: 12x 16 KiB | internal blocks =0 | psram blocks =12 at505f0cc0(alloc101683write98374read114761cycles) | gc cycles = 177030373
[go] psram: 8 of 8 goroutines started with the static window full ran on a psram stack
[go] bench word read internal (16 KiB at 2e000b04): 20487 cycles, 5 per word
[go] bench word read psram (16 KiB at 50700c1c): 20487 cycles, 5 per word
[go] wifi: scan-only run complete, no association attempted
[c] gc ext heap: cap = 4294967295, bytes = 7340032 | idf internal free = 57999, psram free = 9434088
[c] app_main: llgo main returned, deleting main task
```

## Credentials

Station credentials are compiled in from `idfproj/main/wifi_creds.h`, a
file kept out of version control (the `.gitignore` in `idfproj/` lists
`wifi_creds.h`, which covers `idfproj/main/wifi_creds.h`;
`wifi_creds.h.example` is the placeholder). The glue falls back to empty
strings when the file is absent, and an empty SSID selects the scan-only
run (`[go] wifi: scan-only, no credentials compiled in`): the driver is
set to station mode, scans once, never configures or associates with an
AP, and the program returns after the Go heap probe. The Go side reads
the values through `wifi.DefaultSTA()` - nothing is hardcoded in
committed code.

## Memory: two non-overlapping layers

- **llgo GC heap (tinygogc)**: a static 256 KiB window in `.dram0.bss`
  (`host/llgo_syms.S`: 2 KiB runtime reserve + 262,144 bytes + 4 KiB
  bootstrap-stack window). It sits below the IDF heap's `_heap_start` and
  is never managed by the IDF allocator, so it never shows up in
  `esp_get_free_heap_size()`. Once the window is used up, the collector
  takes further segments from the PSRAM heap (see
  [PSRAM and the Go heap](#psram-and-the-go-heap)).
- **IDF heap pool**: everything `esp_get_free_heap_size()` /
  `esp_get_minimum_free_heap_size()` report. The WiFi stack lives entirely
  here (~50 KiB at init, ~55 KiB steady-state connected).

`wifi.HeapFree()`/`wifi.HeapMin()` expose the watermarks and
`wifi.HeapInternalFree()`/`wifi.HeapSpiramFree()` the two pools;
`main.go` prints them at every pipeline stage (`baseline`, `init`,
`start`, `scan`, before and after the PSRAM probe, `dhcp`,
`steady+10s`), so a shrinking internal heap is attributable to the WiFi
stack or IDF itself - never to the Go runtime. PSRAM taken by the Go heap
is reported separately (`psram bytes = N (c side = N)`).

## PSRAM and the Go heap

With PSRAM enabled (`CONFIG_SPIRAM=y` in `sdkconfig.defaults`) the llgo GC
heap is not limited to its static 256 KiB window: when the window is used
up, tinygogc asks IDF for more memory
(`heap_caps_malloc(size, MALLOC_CAP_SPIRAM|MALLOC_CAP_8BIT)`) and links it
in as a further heap segment (runtime:
`runtime/internal/runtime/tinygogc/gc_segment_idf.go`).

- **Enabling.** `CONFIG_SPIRAM=y` is all it takes. The collector probes
  `heap_caps_get_free_size(MALLOC_CAP_SPIRAM)` the first time the window is
  exhausted; less than 256 KiB free (no PSRAM, or `CONFIG_SPIRAM` off)
  means "no external heap": the Go heap stays in the static window and an
  exhausted window fails exactly as before. `CONFIG_SPIRAM_IGNORE_NOTFOUND=y`
  (set in `sdkconfig.defaults`) keeps the firmware bootable on boards without
  PSRAM chips: IDF then reports zero SPIRAM and the same static-window
  behavior applies.
- **Segment sizes.** 1 MiB, 2 MiB, 4 MiB, then 4 MiB per segment, each
  enlarged when the allocation that triggered the growth needs more. A
  failed `heap_caps_malloc` halves the request, down to 64 KiB; below that
  the growth fails and the collector takes its out-of-memory path.
- **Budget.** How much of PSRAM the Go heap may take is the C variable
  `llgo_gc_ext_heap_cap`, read once, when the collector first grows:

  | value                     | meaning                                          |
  |---------------------------|--------------------------------------------------|
  | `(uintptr_t)-1` (default) | 75% of the PSRAM free at the first growth        |
  | `0`                       | disabled: the Go heap stays in its static window |
  | any other byte count      | at most this many bytes of PSRAM                 |

  The runtime ships a weak definition; an application overrides it with a
  strong one, as `app_main.c` does through the `LLGO_GC_PSRAM_CAP` macro.
- **Ledger.** After every successful segment the collector publishes the
  bytes taken so far in `llgo_gc_ext_bytes`. `app_main.c` prints it next
  to the budget and both IDF pools when `main` returns
  (`[c] gc ext heap: cap = ..., bytes = ... | idf internal free = ..., psram free = ...`),
  and the Go side prints the same number through `tinygogc.HeapStats()`
  (`[go] gc heap ... usable/reserved/segments | psram bytes = N (c side = N)`).

Measured under esp-emu (16 MiB PSRAM, scan-only run, bytes; the exact IDF
internal figures drift run-to-run with the linked driver set — P5 measures
58,135 after the scan vs P4's 62,055 before the peripheral drivers joined):

| state                              | IDF internal free | IDF PSRAM free | Go heap reserved | segments | PSRAM taken |
|------------------------------------|------------------:|---------------:|-----------------:|---------:|------------:|
| after the scan, static window only |            58,135 |     16,774,132 |          262,144 |        1 |           0 |
| 6 MiB live, automatic budget       |            58,135 |      9,434,088 |        7,602,144 |        4 |   7,340,032 |
| 6 MiB live, budget 6.5 MiB         |            58,135 |      9,958,376 |        7,077,856 |        4 |   6,815,744 |

The internal IDF heap is untouched: the price of Go heap growth is paid
from PSRAM only. With PSRAM disabled the run prints
`[go] psram: external heap unavailable (spiram free = 0 ), the Go heap stays in its static window`
and goes on unchanged.

Cycles per 16 KiB block (esp-emu, `rdcycle`): alloc/write/read cost
102,196/98,313/114,700 in the static window and
101,683/98,374/114,700 in a PSRAM segment, and a word-wise read takes
5 cycles per word in both. **The emulator does not model PSRAM latency**,
so these numbers show the allocator's overhead only; real PSRAM access
costs have to be measured on hardware. What the emulator does show is the
collector: one stop-the-world collection costs 3.8 M cycles with the
static window alone and 70-180 M with 6 MiB live in a 7 MiB heap.

Known limitations:

- Only the `esp32s31-idf` target grows into PSRAM. The baremetal
  `esp32s31` target has no `heap_caps` allocator and keeps the static
  window alone.
- Goroutine (fiber) stacks are allocated from the Go heap, so once the
  window is exhausted new goroutine stacks land in PSRAM too (the scan-only
  run starts eight at that point and reports where their stacks are).
- Collections are stop-the-world and walk every segment: the pause grows
  with the live set, on top of slower PSRAM access on real hardware.
- The budget is read at the first growth and never revisited; memory
  handed to the Go heap is never returned to IDF.

## Notes

- The WiFi stack's tasks are pinned to CPU1
  (`CONFIG_ESP_WIFI_TASK_PINNED_TO_CORE_1`) so scan/association bursts never
  compete with the llgo scheduler on CPU0.
- The IDF heap must absorb the WiFi allocations (~100-150 KiB) next to the
  static 256 KiB llgo GC heap; the glue prints the free heap before setup to
  make that budget visible.
- Host builds compile entirely against the stub side (`go build`, `go vet`,
  gopls): `pkg/esp/*` through their `_stub.go` files, `goapp/main.go`
  through `goapp/hosted.go`, whose push linknames define the FreeRTOS/C
  externs inertly on the host.
- `goapp/main.go` triggers collections through
  `//go:linkname gcCollect .../tinygogc.GC` instead of `runtime.GC()`: in
  an `esp32s31-idf` build `import "runtime"` drags in the runtime's
  finalizer support and with it libffi, whose C sources do not compile for
  this target.
