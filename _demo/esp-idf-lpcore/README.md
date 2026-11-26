# llgo × ESP-IDF LP core pipeline (ESP32-S31)

A self-contained pipeline that drives the ESP32-S31's LP (low-power)
coprocessor core from Go code compiled by llgo: load an LP program into LP
RAM, run it, exchange single-word messages over the hardware mailbox, stop
it - with the acceptance run automated under esp-emu. The Go binding lives in
the llgo repository (`pkg/esp/lpcore`), consumed through the module require +
replace in `go.mod`; this directory owns the acceptance program, the LP core
C program, and the IDF project.

## Layout

```
esp-idf-lpcore/
├── go.mod                  Go module (module llgoldpcore; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe (Load → Run → markers → mailbox
│                           echo → Stop; deep-sleep stretch gated by
│                           probeDeepSleep, off for the emulator - see
│                           Fidelity notes)
├── goapp/hosted.go         Host-build push-linkname stubs for the FreeRTOS/C
│                           externs in main.go, so `go build` links
├── idfproj/                Minimal ESP-IDF project
│   ├── CMakeLists.txt
│   ├── sdkconfig.defaults  8 KiB main task, 240 MHz, PSRAM, and the LP core:
│   │                       CONFIG_ULP_COPROC_ENABLED + TYPE_LP_CORE +
│   │                       RESERVE_MEM=8192
│   ├── partitions.csv      4 MB layout, 3.9375 MiB factory app
│   ├── host/
│   │   ├── make-artifact.sh  llgo build -> IDF-linkable relocatable object;
│   │                         also syncs pkg/esp/lpcore/glue into main/
│   │   └── llgo_syms.S       GC heap + bootstrap stack window providers
│   └── main/               The main component
│       ├── CMakeLists.txt  Sources + ulp_embed_binary + WHOLE_ARCHIVE
│       ├── app_main.c      app_main -> llgo entry; _write/_exit shims
│       ├── demo_glue.c     Demo-local glue: embedded LP program accessors,
│       │                   LP progress markers, LP RAM readback, the
│       │                   deep-sleep stretch bindings (esp_sleep)
│       ├── lpcore_glue.c   Package glue, synced from
│       │                   pkg/esp/lpcore/glue at build time (gitignored)
│       ├── lp_core/main.c  The LP core program: a mailbox echo server, with
│       │                   progress markers exported to the HP side
│       └── llgo_dram.ld    _globals_start/_globals_end for tinygogc
├── golden/emu-lpcore-probe.txt  Golden for the [go] lp lines (no floater)
└── README.md
```

## Architecture

ESP-IDF/FreeRTOS owns the platform; llgo compiles `goapp` into a relocatable
object (`-target esp32s31-idf`, `ld.lld -r`) that the main component links
into the firmware; `app_main` calls the generated C entry. The LP core
program is a separate C source (`main/lp_core/main.c`) that the IDF ULP
build system compiles with the LP RISC-V toolchain (`ulp_embed_binary` in
`main/CMakeLists.txt`, the pattern of IDF's `lp_mailbox` example) into
`lp_core_main.bin`, which the HP firmware links. The probe reads that
embedded binary through `demo_glue.c` and hands the bytes to
`lpcore.Load` - the same path a real program uses.

The HP ↔ LP wire format is one 32-bit mailbox word per message in
`lpcore.Pack`'s (cmd, arg) halves: HP sends `(1, i)`, the LP core answers
`(2, i XOR 0xA55A)` - a reply value only code running on the LP core can
produce, so the round trip is the whole mailbox story in one check.

The LP program also maintains four exported progress markers (plain LP RAM
words, `ulp_lp_echo_*` in the generated `lp_core_main.h`): a boot magic
written at `main()` entry, the mailbox-init result, the entry poll's result,
and the receive counter. The probe prints them right after the settle wait,
which is how any environment (emulator or hardware bring-up) tells apart
"the LP core never ran" from "the mailbox path broke".

## Build and run

Requirements: the llgo compiler (build with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-lp ./cmd/llgo`
from the llgo repo), docker with `espressif/idf:latest`, and esp-emu
(`~/.local/bin/esp-emu`, v1.2.0).

```bash
cd _demo/esp-idf-lpcore

# 1. Go -> relocatable object (also syncs pkg/esp/lpcore/glue into main/).
cd idfproj
LLGO_ROOT=<llgo-repo> ./host/make-artifact.sh /tmp/llgo-lp

# 2. IDF build (set-target first on a fresh tree; delete sdkconfig if you
#    edit sdkconfig.defaults, so the defaults re-apply).
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
    bash -c 'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'
cd ..

# 3. Acceptance run (one esp-emu instance per machine: the lock).
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --console usb-jtag --timeout 90s \
    --exit-on "[go] lp PASS"
rmdir /tmp/esp-emu.lock
```

The ELF matters as much as the binary here: the emulator reads the
application's DWARF from it (its `[INFO machine]` line says so), and a
re-linked firmware without its ELF next to it changes what the emulator can
report.

## Acceptance

Two rounds of step 3. Each round passes when the `[go] lp` lines, stripped
of CR, match `golden/emu-lpcore-probe.txt` exactly:

```
grep -E '^\[go\] lp' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-lpcore-probe.txt)
```

There is no floater rule: the probe prints no wall-clock values, no
counters and no addresses. Every value is a build or protocol fact, and the
markers line is the run's fidelity statement - pinning it is the point.
Under esp-emu the run is one boot (see below); on hardware with
`probeDeepSleep = true` the golden grows a second boot's `markers2` /
`wake ok` / `PASS` lines - appended, never edited.

## Fidelity notes (esp-emu v1.2.0, chip esp32s31, IDF v6.3-dev)

Measured with this demo's probe (2026-11); the upstream README documents LP
core + sleep modeling for the ESP32-C6, and lists the S31 as early
bring-up - the measurements agree:

- **LP core: not modeled.** The whole HP-side driver surface works
  (`ulp_lp_core_load_binary` returns OK, `ulp_lp_core_run` returns OK,
  `ulp_lp_core_stop` is a no-op), but the LP RISC-V hart never executes:
  the program's first store - the `0xC0DEC0DE` boot marker, the very first
  instruction of `main()` - never lands (`[go] lp markers boot=0x00000000
  ...`), so the mailbox echo cannot happen and the probe reports the skip
  line instead of exchanges. Whatever the HP side writes to the LP-core
  boot/wake registers, the emulator does not start a second hart from it.
- **LP RAM: modeled as HP-visible memory.** After `Load`, the first 32
  bytes read back from `RTC_SLOW_MEM` (the LP RAM base at `SOC_RTC_DATA_LOW`,
  0x2E000000) word-for-word (`[go] lp lpram ok`) - the load's memcpy lands
  in real emulated memory; it is the execution that has no model.
- **Deep sleep: not modeled.** `esp_deep_sleep_start` spins inside
  `pmu_sleep_start`'s hardware-wait loop (the PMU never reports the sleep
  transition) until the interrupt watchdog aborts the firmware
  (`Interrupt wdt timeout on CPU0`, measured in run 3). The stretch phase
  therefore ships gated off (`probeDeepSleep = false` in `goapp/main.go`);
  the probe still exercises `Run(WakeLPTimer)`'s driver path and
  `esp_sleep_enable_timer_wakeup` when the flag is on, but only hardware
  can close the sleep → timer-wake → `ESP_SLEEP_WAKEUP_TIMER` loop today.
- **The HP-side esp_err surface is faithful**: the oversize load guard
  returns the driver's `ESP_ERR_INVALID_SIZE`, and every driver call's
  return code matches the v6.3-dev source (checked against
  `components/ulp/lp_core/lp_core.c` in the container).
- No probe call hangs the emulator by itself; the only abort in this demo's
  history is the deep-sleep entry above, which is why that phase is a flag.

What the acceptance run under esp-emu proves, precisely: the llgo → C glue
→ IDF ULP-driver path for Load (with real LP RAM), Run, Stop and the
client-side error contracts, end to end in the real firmware image. What it
cannot prove until the emulator grows an S31 LP hart (or the same firmware
runs on hardware): the mailbox exchanges and the deep-sleep wake, whose
hardware expectations are documented above and in the golden's comments.

## Core affinity (runtime.LockCore) and this demo

The llgo runtime ships a weak `llgoLockCore` default (an honest refusal),
so this demo - which never pins - needs no shim of its own. A project that
wants real core pinning defines a strong `llgoLockCore` override in its
main component; `_demo/esp-idf-lockcore` carries that full implementation
for its own acceptance probe.

## Hardware notes (measured on a real S31 board)

The probe was flashed to a real ESP32-S31 (USJ console, `probeDeepSleep`
flipped locally, esptool write-flash of the merged image). Measured:

- **The LP core executes.** `markers boot=0xC0DEC0DE mbinit=0` - the LP
  program booted (no LP ROM; the HP-written boot address worked) and its
  own `lp_core_mailbox_init` succeeded. `poll=263` is `ESP_ERR_TIMEOUT`
  (0x107) - the entry poll's honest answer for an empty mailbox.
- **HP deep-sleep entry works.** With the stretch phase enabled, the USB
  console dropped exactly at `esp_deep_sleep_start` (the USJ peripheral
  lives in the HP power domain). The emulator cannot model this.
- **The full mailbox round trip works: 8/8 echo exchanges pass.** With the
  bounded blocking send, the probe ran the whole protocol on hardware:
  every pong argument came back exactly `i XOR 0xA55A` for i = 0..7 (a
  reply value only LP-core code can compute), through stop, stop-twice
  and the reload. The earlier zero-tick run (whose slot-cursor wipe
  motivated the bounded send) never got past the first exchange.
- **LP-timer wake-up after deep sleep is verified.** On an ESP32-S31-Function-CoreBoard
  (whose onboard USB-to-UART bridge keeps UART0 alive across deep sleep, unlike the
  USJ port) the probe ran the full loop: `deep sleep now` → `rst:0x5
  (PMU_SYS_PWR_DOWN_RESET)` → `wakeup_cause=16 ESP_SLEEP_WAKEUP_TIMER` (the IDF
  sleep API itself attributing the reset to the configured timer) → the post-wake
  marker check reading the intact `0xC0DEC0DE` LP-RAM boot marker (the LP domain
  stays powered in deep sleep) → `PASS`. The first boot correctly reported
  `wakeup_cause=1 ESP_SLEEP_WAKEUP_UNDEFINED`, distinguishing power-on from timer
  wake. An earlier attempt over the USJ port alone never showed any of this: the
  USJ peripheral does not re-enumerate after wake, so deep-sleep observation needs
  a console on UART0.
- **The mailbox needs a bounded blocking send.** The driver's HP-side ACK
  arrives through the mailbox interrupt + semaphore; a zero-tick send
  returns before that machinery can ever deliver the ACK, and each send
  advances the TX slot cursor, so a caller-level retry storm wraps the
  four slots and wipes messages the LP core has not consumed (measured).
  The package glue therefore sends with `pdMS_TO_TICKS(500)` - one
  bounded blocking wait per message - and the Go doc on `MailboxSend`
  carries the scheduler-freeze caveat.

The committed probe still ships `probeDeepSleep = false` (the emulator's
golden stays single-boot) and the UART0 console; the hardware run's local
flips (USJ console, flag true) are exactly that - local, not committed.
