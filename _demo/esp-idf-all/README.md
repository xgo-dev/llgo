# esp-idf-all — the comprehensive llgo verification firmware for ESP32-S31

One firmware exercising every capability at once, so a single board session
answers all of them: soft AP + HTTP server, an A2DP sink streaming through
the ES8311 / NS4150B analog path, a BLE HID-keyboard central host on the
SAME dual-mode Bluedroid stack, and the codec self-test (sine → mic sweep →
loopback). Heap watermarks with a DMA-block metric print at every stage.

## Bring-up sequence (what the firmware does, in order)

1. `bt.Init()` — controller in BTDM mode, Bluedroid up, name `llgo-audio`,
   discoverable. The classic host comes first; everything BLE joins it.
2. `ble.Init()` — the project's patched `ble_glue.c` detects the running
   stack and registers only its own callbacks (Bluedroid is a singleton; a
   second `esp_bluedroid_init` would fail). Init order bt → ble is the
   firmware's contract.
3. `bt.A2DPEnable` — the A2DP sink's decoded-PCM I2S pipeline on
   BCLK=53 / WS=55 / DOUT=56, MCLK=52 (one-line local `a2dp_glue.c` patch;
   `A2DPConfig` has no MCLK field yet — recorded as a pkg/esp/bt gap).
4. WiFi soft AP `llgo-ap` / `llgoappass`, channel 6, 192.168.4.1, DHCP
   server; an HTTP server on port 8000 answers `GET /` with `llgo-ap-ok`.
5. ES8311 codec init over I2C (0x18) with the audited register sequence,
   then the self-test: 3 s of 440 Hz sine through the speaker, a 4-config
   mic gain sweep with RMS statistics, a loopback replay of the capture,
   then the codec is reconfigured for A2DP rates and the sink re-enabled.
   The self-test pauses A2DP (the I2S clock pins are exclusive), ~20 s.
6. From then on: A2DP byte counters every 2 s (≈352800 B per 2 s at
   44.1 kHz stereo), BLE scan rounds every 10 s, the HTTP server and the
   AP station lifecycle run concurrently. The amp pin follows the A2DP
   link once the codec is up.

## Hardware prerequisites

- ESP32-S31-Function-CoreBoard-1: PSRAM fitted (16 MB octal on this board),
  4 MB flash, ES8311 codec + NS4150B amp, UART0 console on the USB bridge.
- **Codec_CE note**: the board's ES8311 CE pin sits at the schematic's
  default (low). This is Espressif's own I2C address-select scheme and
  selects address 0x18 — it is NOT a disable. No fly-wire is needed; the
  codec answers at 0x18 (chip ID 0xFD=0x83 / 0xFE=0x11, verified both by
  bit-bang and by the i2c_master driver).
- The old board's UART0 bridge asserts DTR/RTS on open; every serial reader
  here opens with both deasserted or the chip resets into download mode.

## Operating the verification (user checklist)

1. **WiFi**: join `llgo-ap` (password `llgoappass`), open
   `http://192.168.4.1:8000/` — the page shows `llgo-ap-ok`. The console
   logs `STA_CONNECTED`, the DHCP `LEASE`, the station table and the HTTP
   line. Every station is deauthenticated 40 s after it associated (the
   kick step runs on its own; rejoin for another window).
2. **A2DP**: pair/connect the phone to `llgo-audio` and play. The console
   logs the connect, the negotiated codec config (`rate=44100 ch=2`) and
   the byte counters — the written delta tracks ≈352800 B per 2 s tick,
   `drop+` stays ≈0, and the speaker plays (amp is on while a phone is
   connected and the codec is initialized).
3. **BLE keyboard**: put a Bluetooth keyboard in pairing mode. The console
   logs `KEYBOARD-FOUND`, the pairing result, the GATT walk, and every key
   press decoded (`kb:report n = 1 handle = 39 => a`). Measured on this
   board: 5.1 s from connect to usable, of which 3.6 s is the pairing and
   the rest is the service walk and the one subscription the reports need.
   After that the keyboard reboots onto its stored bond and needs no
   pairing-mode step at all: every round first asks
   `ble.BondedPeers()` and connects straight to the stored address
   (`kb:bonded peer stored - connecting without a scan`), which is the only
   way back in - a bonded peer stops advertising.
4. **Audio self-test** (runs by itself ~20 s after boot): the speaker plays
   a 440 Hz sine (`SINE-START`/`SINE-DONE`), the mic sweep statistics and
   the 3 s capture land in the console (`MIC-RMS ... PASS`), and the
   capture plays back through the speaker (`LOOPBACK-START/DONE`).

### What the keyboard bring-up had to get right

Four things failed in order on real hardware and each one is fixed in the
package now, so a different keyboard does not have to rediscover them:

1. **Bonding needs its parameters spelled out.** Bluedroid keeps no usable
   defaults for `ESP_BLE_SM_MAX_KEY_SIZE` (and needs the IO capability, the
   key-distribution masks and the auth mode). With any of them unset the
   pairing reports success while distributing no key: `AUTH_CMPL` arrives,
   `cm->key_present` is 0, nothing lands in NVS, and the peer has to re-pair
   on every reconnect. `key_present` is also not the bond test - use
   `esp_ble_get_bond_device_num()`, which this firmware prints as
   `bond_count=N` next to the auth result.
2. **Every address crosses the Go/C boundary in stack order.** The event
   path swaps it (`bw_addr_swap`); a bond listing that memcpys Bluedroid's
   `bd_addr` straight through hands Connect a reversed address, which fails
   to establish with `rsn=0x100`.
3. **A bonded peer does not advertise.** The reconnect path has to go
   through the stored bond, tried every round rather than once per boot, so
   a keyboard power-cycled mid-session is picked up within 10 s.
4. **Notification registry size.** A HID keyboard exposes more notifying
   characteristics than the default `CONFIG_BT_GATTC_NOTIF_REG_MAX` of 4
   (`Max Notification Reached, registration failed`); this firmware sets 16
   and subscribes only to the first input report, which is where every key
   press arrives - each extra CCCD write is roughly 1.7 s of ATT traffic.

Disconnect reasons print as text (`ble.DisconnectReasonName`), so a link
that drops says why: `kb:disconnect reason = connection timeout (supervision
timeout) after 2 reports` means the peer stopped hearing from us, not that
the host tore the link down.

## Serial evidence markers (grep table)

| marker | meaning |
|---|---|
| `[go] bt init ok` / `[go] ble init ok` / `[go] a2dp enable ok` | stack bring-ups |
| `[go] ap START ip = 192.168.4.1` | soft AP up |
| `[go] ap STA_CONNECTED` / `LEASE` / `stations n =` | client lifecycle + table |
| `[go] http GET / ... -> 200 llgo-ap-ok` | the HTTP evidence line |
| `[go] ap DEAUTH` | the 40 s kick |
| `[go] codec ES8311-OK` | codec answered, registers read back |
| `[go] codec SINE-START` / `MIC-RMS` / `LOOPBACK-DONE` | self-test points |
| `[go] a2dp t = N s tot+ = ...` | the 2 s data-plane counter |
| `[go] kb KEYBOARD-FOUND` / `PASSKEY` / `ENC ok` / `kb report` | keyboard flow |
| `[go] heap <stage> ... dma max =` | the memory ledger |
| `[c] ble glue: PASSKEY ...` / `auth cmpl ok` | C-side pairing events |

## Build & flash

```
cd idfproj/host && ./make-artifact.sh            # Go -> llgo_out.o (glues synced)
cd .. && docker run --rm -v "$PWD":/project -w /project \
  espressif/idf:latest bash -c \
  'idf.py build && idf.py merge-bin -o merged-binary.bin'
# flash (board lock /tmp/s31-board.lock held):
/Users/ma6174/.local/bin/esptool --chip esp32s31 --port /dev/cu.usbserial-110 \
  --baud 460800 write-flash 0x0 build/merged-binary.bin
```

`LLGO_BT_TRACE=1 ./make-artifact.sh` adds stage prints around the classic
stack bring-up (diagnostic; the hang it once chased turned out to be the
assert below, not a stuck C call).

## Real-hardware discipline baked into this firmware

Each of these was a measured failure mode on this exact board; the code and
the sdkconfig both carry them. Reuse the rules, not just the fixes.

1. **C-call stack budget.** The fiber scheduler multiplexes all goroutines
   on one FreeRTOS task and swaps SP onto 2 KiB GC-heap fiber stacks. Deep
   C paths — the BLE host's BTC message deep copies (`esp_ble_gap_*`), the
   esp_wifi API-lock calls — need several KiB of C stack. This firmware
   therefore runs every BLE and esp_wifi control call on the MAIN goroutine
   (`controlLoop`), whose fiber gets 8 KiB; the background fibers stick to
   the light drivers their own probes proved (mailbox polls, lwip poll,
   i2c/i2s). A keyboard scan submitted from a 2 KiB background fiber
   corrupted the GC heap and died inside the runtime's goroutine status
   machine (`invalid goroutine status transition`). Raising
   `-goroutine-stack-size` is NOT a workaround: the runtime scales the main
   fiber to 4x the same value, moving its allocation into the heap at a
   point where the firmware boot-loops.
2. **No task stacks in PSRAM.** `CONFIG_FREERTOS_TASK_CREATE_ALLOW_EXT_MEM`
   must be `=n` (its Kconfig default under SPIRAM is y): while it is on,
   every flash cache-disabled section asserts the running task's stack is
   sane, and a llgo fiber stack that lands in a PSRAM heap segment (the
   collector grows segments as the heap fills) makes every NVS write abort
   the chip. The 40 KiB static GC window keeps the bootstrap, the main
   fiber and all goroutine stacks in internal SRAM for the whole run.
3. **esp_wifi query gating.** Querying the driver (station table,
   deauth) inside or right after a join/leave/DHCP event burst freezes the
   firmware: the driver's API lock is held across the burst and a baremetal
   C call cannot yield out of it. Every query here waits until the AP has
   been quiet for 5 s, and the deauth kick never queries afterwards.
4. **net.Listener.Accept throttling.** The net package's accept loop never
   Goscheds (its poll parks the single scheduler task and the retry runs
   immediately), so an idle listener in an unbounded wait starves every
   other goroutine. The HTTP loop takes 1 s bounded accepts with 200 ms
   yields between.
5. **No `time.Sleep` from background fibers.** The fiber sleeper list is
   unproven under concurrency in this repo; this firmware's only wait
   primitive is the busy-Gosched bounded wait (`yieldMs`) — the shape every
   verified probe uses.
6. **DMA block monitoring.** The WiFi driver needs ~1.6 KiB contiguous
   DMA-capable internal blocks per packet; the ledger prints
   `heap_caps_get_largest_free_block(MALLOC_CAP_DMA)` because that number,
   not the free totals, predicts whether the radio keeps working under a
   client join.

## Internal-RAM budget (what the sdkconfig deliberately drops)

With everything at IDF defaults the dual-mode host plus the WiFi driver
leave ~32 KB internal at a2dp-init and `wifi.Setup` fails with
`ESP_ERR_NO_MEM`. The dropped pools (none of them used by this firmware):
GATTS server, extended/periodic advertising, WiFi AMPDU BA sessions,
static RX buffers 10→6. `CONFIG_SPIRAM_MALLOC_ALWAYSINTERNAL=4096` keeps
only small allocations in internal SRAM. `BT_CTRL_RUN_IN_FLASH_ONLY` (the
controller-lib flash variant that frees ~20 KiB of DIRAM on other targets)
is NOT available for esp32s31 in IDF v6.3 — the lib set ships no flash
variant (`libbtdm_app_flash.a` absent) and the Kconfig has no symbol for
this target.

Measured ledger (this firmware, final config): baseline 164 KB internal /
81920 B DMA block → bt-init 80 KB / 31744 → a2dp-init 51 KB / 21504 →
ap-start 13 KB / ~13 KB. The remaining margin covers one phone join + one
HTTP session; heavier AP traffic would want the AMPDU windows back once a
bigger flash (or the controller flash lib) relieves the budget.

## Project layout

- `goapp/` — one file per area, all in `package main`; every subsystem
  degrades independently (a FAIL line never stops the others):
  - `main.go` — entry, bring-up order, the cooperative control loop
    (wifi events + one keyboard step per iteration + the memory cadence).
  - `wifi_ap.go` — soft AP, station bookkeeping, the kick timer, the HTTP
    server.
  - `bt_a2dp.go` — the Bluedroid bring-up shared by both roles and the A2DP
    sink loop.
  - `ble_kb.go` — BLE central state machine: stored-bond reconnect, scan,
    connect, pairing, HID walk, report decoding.
  - `audio.go` — ES8311 register work, the I2S self-test, the sine, the mic
    sweep, the loopback, and the statistics.
  - `mem.go` — the heap ledger and its periodic cadence.
  - `log.go` — the IDF-style `I (tick) tag: message` emitter and the small
    number/hex formatters.
- `idfproj/main/ble_glue.c` — checked-in copy of
  `pkg/esp/ble/glue/ble_glue_bluedroid.c` with two local patches: the
  shared-stack bring-up and the BLE pairing flow (SM posture: DisplayOnly
  IO / MITM / SC / bonding; SEC_REQ accepted, PASSKEY printed, NUMCMP
  auto-confirmed, AUTH_CMPL tracked, stale bonds removed; plus the two
  `llgo_ble_security` / `llgo_ble_conn_sec` passthroughs and the 15-30 ms
  HID link parameters). See the file header for the exact delta.
- `idfproj/main/a2dp_glue.c` — synced at build time with the one-line MCLK
  patch (assert-guarded in `host/make-artifact.sh`).
- All other glues (`wifi net bt i2c i2s gpio`) are verbatim build-time
  copies of the `pkg/esp/*/glue` templates.
- `idfproj/host/llgo_syms.S` — llgo runtime symbol providers (40 KiB GC
  window, 4 KiB bootstrap stack window).
