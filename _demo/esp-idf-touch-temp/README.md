# llgo × ESP-IDF touch + temperature sensors (ESP32-S31)

A self-contained pipeline that drives the ESP32-S31's touch sensor (hardware
v3) and built-in temperature sensor from Go code compiled by llgo, with the
acceptance run automated under esp-emu. The Go bindings live in the llgo
repository (`pkg/esp/touch`, `pkg/esp/temp`), consumed through the module
require + replace in `go.mod`; this directory owns the acceptance program and
the IDF project.

## Chip facts this demo is built on

- **Touch**: the S31 ships touch hardware **v3** (SOC_TOUCH_SENSOR_VERSION 3)
  with 14 channels (ids 0..13). The IDF driver is the new
  `driver/touch_sens.h` API (the `esp_driver_touch_sens` component) - the
  legacy `driver/touch_pad.h` API only exists for the esp32/s2/s3
  generations and is not available on this target. Channels map onto fixed
  GPIOs:

  | chan | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 |
  |------|---|---|---|---|---|---|---|---|---|---|----|----|----|----|
  | GPIO | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 |

  On the Function-CoreBoard-1 board GPIO6/7 are claimed by Ethernet, so the
  demo scans Chan4 (GPIO10) and Chan6 (GPIO12) - both on header pins a
  finger can reach. The v3 hardware keeps **one measurement datapath per
  channel**: every scan yields the channel data (the charge-discharge count,
  IIR-smoothed per the filter config; with `SmoothNoFilter` it is the raw
  count) and the benchmark (the hardware-maintained baseline). Touch
  detection is relative: a channel is active while
  `data - benchmark > ActiveThreshold`, and the driver raises active/inactive
  interrupts the glue funnels into a ring Go polls.

- **Temperature**: built in, ±1 °C accuracy. The IDF v6.3 driver lifecycle is
  install/enable/disable/uninstall (`driver/temperature_sensor.h`, the
  `esp_driver_tsens` component); the binding names it Install/Start/Stop/
  Uninstall and passes every driver error through unadorned. The configured
  range span (`DefaultOpts`: -10..80 °C) selects the sensor's internal
  range/dacdiv steps.

- **Deep-sleep touch wake**: the hardware supports touch wake-up
  (TOUCH_SUPPORT_SLEEP_WAKEUP), but the S31 Function-CoreBoard's EXT1 pin
  restrictions make deep-sleep touch wake unavailable on this board. Reading
  the sensor while running works (this demo). The LP-domain touch usage is a
  forward-looking note only - see `pkg/esp/lpcore` for the LP coprocessor the
  sensor could pair with.

## Layout

```
esp-idf-touch-temp/
├── go.mod                  Go module (module llgotouchtemp; requires + replaces
│                           github.com/xgo-dev/llgo, the binding packages' home)
├── goapp/main.go           Acceptance probe (temp install→start→5 reads→stop→
│                           uninstall; touch init→config→start→10 scan rounds with
│                           data/baseline/delta + event drain → recalibrate →
│                           stop→deinit)
├── goapp/hosted.go         Host-build push-linkname stubs for the FreeRTOS
│                           externs in main.go, so `go build` links
├── idfproj/                Minimal ESP-IDF project
│   ├── sdkconfig.defaults  8 KiB main task, 240 MHz, PSRAM (no driver-specific
│   │                       switches: both drivers are plain REQUIRES)
│   ├── partitions.csv      4 MB layout, 3.9375 MiB factory app (generated)
│   ├── host/
│   │   ├── make-artifact.sh  symlink to _demo/common; llgo build ->
│     │                      IDF-linkable relocatable object; also syncs
│   │                      pkg/esp/{touch,temp}/glue into main/
│   │   └── artifact.conf    GLUES="touch temp"
│   └── main/
│       ├── CMakeLists.txt  Sources + esp_driver_touch_sens/esp_driver_tsens
│       │                   REQUIRES + WHOLE_ARCHIVE
│       ├── app_main.c      app_main -> llgo entry (generated from _demo/common)
│       ├── demo_glue.c     Demo-local glue (empty: no C helpers needed)
│       ├── touch_glue.c    Package glue, synced from pkg/esp/touch/glue at
│       │                   build time (gitignored)
│       └── temp_glue.c     Package glue, synced from pkg/esp/temp/glue
│                           (gitignored)
├── golden/emu-touch-temp-probe.txt  Golden for the [go] tt lines (no floater)
└── README.md
```

## Build and run

Requirements: the llgo compiler (build with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-tt-bin ./cmd/llgo`
from the llgo repo), docker with `espressif/idf:latest`, and esp-emu
(`~/.local/bin/esp-emu`, v1.2.0).

```bash
cd _demo/esp-idf-touch-temp

# 1. Go -> relocatable object (also syncs pkg/esp/{touch,temp}/glue into main/).
cd idfproj
LLGO_ROOT=<llgo-repo> ./host/make-artifact.sh /tmp/llgo-tt-bin

# 2. IDF build (set-target first on a fresh tree; delete sdkconfig if you
#    edit sdkconfig.defaults, so the defaults re-apply).
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
    bash -c 'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'
cd ..

# 3. Acceptance run (one esp-emu instance per machine: the lock).
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --console usb-jtag --timeout 60s \
    --exit-on "[go] tt PASS"
rmdir /tmp/esp-emu.lock
```

The ELF matters as much as the binary here: the emulator reads the
application's DWARF from it, and a re-linked firmware without its ELF next to
it changes what the emulator can report.

## Acceptance

The run passes when the `[go] tt` lines, stripped of CR, match
`golden/emu-touch-temp-probe.txt` exactly:

```
grep -E '^\[go\] tt' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-touch-temp-probe.txt)
```

There is no floater rule: the probe prints no wall-clock values. Every value
is a build fact (channel ids, GPIO map, threshold), an emu-constant sensor
reading, or a derived zero - see Fidelity notes.

## Fidelity notes (measured, esp-emu v1.2.0)

Neither peripheral is modeled by the emulator; the numbers below are what
the (real, unmodified) IDF driver code reports when its registers sit in the
emulator's unmapped/zero state:

- **Touch**: the full driver path runs green - controller allocation, the
  callback interrupt allocation, channel allocation, filter + benchmark
  configuration, scan start/stop, `touch_channel_read_data` all return
  ESP_OK - but every reading is 0 (data, baseline, delta). The measurement
  hardware that would count charge-discharge oscillations does not exist in
  the emulator, and zero counts can never cross an active threshold, so no
  event lines appear either. The PASS gate proves the Go→glue→driver→register
  link; it does not prove capacitance sensing.
- **Temperature**: same story - install/enable/read/disable/uninstall all
  return ESP_OK, but the conversion table maps the fixed register state to
  one constant value, so all five reads print `milli=32000` (32.0 °C).
- On hardware the touch lines read charge-discharge counts (tens of
  thousands with the default sample config; a pad touch moves `delta` by
  hundreds) and the temperature lines read the die temperature at ±1 °C.

## Calibration flow (hardware)

1. Power up with the pads untouched; run the probe's bring-up
   (Init → Config → Start).
2. Let the benchmark settle (a few hundred ms), then read `Baseline(ch)` -
   that is the reference for touch detection.
3. During runtime, judge touch by the ratio, not the absolute count:
   `per-mille = delta * 1000 / baseline`. With the default sample config a
   finger adds several per-mille; start judging "touched" above ~20-50
   per-mille and tune on the real pads.
4. When drift accumulates (temperature, aging), call `Recalibrate(ch)` with
   the pad untouched - it resets the benchmark to the current data. The
   hardware benchmark also tracks slowly on its own (IIR_16 by default), so
   a *held* touch fades back into the baseline: that is the expected
   behavior, detect press transitions (the events) rather than level.

`ActiveThreshold` is in channel-data units and scales with the sample
config's gain (`DivNum`, `ChargeTimes`): change the gain, re-tune the
threshold.

## Hardware bring-up checklist (not yet verified - emulator run only)

- [ ] Touch readings on the real pads: expected tens-of-thousands counts
      with `DefaultOpts`; confirm `data`/`base` move and `delta` responds to
      a finger on Chan4 (GPIO10) / Chan6 (GPIO12).
- [ ] Sensitivity tuning: `ActiveThreshold` per pad, `DivNum` if readings
      saturate, `ChargeTimes` for the count scale.
- [ ] Threshold events: touch a pad, expect `touch event ch=... active` then
      `... inactive` lines from the event ring; verify hysteresis/debounce
      feel (DefaultFilter's 2/2).
- [ ] Temperature vs a room thermometer: expect ±1 °C agreement with the
      -10..80 °C span; check a second span (e.g. 0..50) installs and reads
      consistently.
- [ ] `SetThreshold` stop→reconfigure→restart cycle on hardware: confirm the
      scan resumes and the benchmark rebuilds.
- [ ] LP-domain touch (forward-looking): pairing the touch sensor with
      `pkg/esp/lpcore` for low-power scanning is unimplemented - design
      space, not a binding gap.
