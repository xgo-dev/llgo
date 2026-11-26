# llgo × ESP-IDF ADC oneshot (ESP32-S31)

A self-contained pipeline that drives the ESP32-S31's on-chip analog
converter (the `esp_hal_ana_conv` ADC, in the IDF's oneshot mode) from Go
code compiled by llgo, with the acceptance run automated under esp-emu. The
Go binding lives in the llgo repository (`pkg/esp/adc`), consumed through
the module require + replace in `go.mod`; this directory owns the acceptance
program and the IDF project.

## Chip facts this demo is built on

Everything below is read off the IDF headers the toolchain ships
(`espressif/idf:latest` = v6.3-dev; `soc/adc_channel.h`,
`esp_hal_ana_conv`, `esp_adc`). The S31's converter is **not** the classic
ESP32-generation SAR ADC, and several classic recipes do not apply:

- **The analog pads are GPIO42..57** - one contiguous run of sixteen pads.
  ADC1 owns GPIO42..49 (channels 0..7), ADC2 owns GPIO50..57 (channels
  0..7). The board-header pins below 42 (GPIO1..9, the analog-looking pins
  of other ESP32 generations) are **not** ADC pads on this chip. The
  binding pins the pad set as its Go-side guard (`ErrBadPin`).
- **One attenuation.** `SOC_ADC_ATTEN_NUM` is 1: the converter implements
  exactly `ADC_ATTEN_DB_0`. There is no 2.5/6/11 dB tap and no "11 dB
  full-scale ~3.1 V" recipe; anything else is refused with `ErrBadAtten`.
- **Fixed 17-bit width.** The oneshot conversion produces a 17-bit result
  (`ADC_LL_RTC_MIN_BITWIDTH = ADC_LL_RTC_MAX_BITWIDTH = 17`); the driver
  also accepts the DEFAULT placeholder (0) and expands it to 17. There is
  no 12-bit mode. `DefaultOpts()` returns the explicit 17.
- **The raw value is a 17-bit comparator vector**, not a magnitude code:
  even channels are the N side and odd channels the P side of the
  converter's differential pairs. Read returns the vector as-is;
  ReadMillivolts is what carries physical meaning.
- **Calibration is LINE-FITTING**, the only scheme this chip compiles with
  (`ADC_CALI_SCHEME_LINE_FITTING_SUPPORTED`). The classic channel-blind
  `adc_cali_raw_to_voltage` is **not supported** on the S31 (the IDF marks
  it `ESP_ERR_NOT_SUPPORTED` there); the conversion goes through the
  channel-aware `adc_cali_chan_raw_to_voltage`, which keeps separate N/P
  characteristics. The glue creates one calibration handle per unit,
  lazily on the first ReadMillivolts, and releases it at Deinit.
- **The calibration eFuse regime is surfaced, not hidden.** When the
  chip's calibration eFuse block is burnt (block version >= 2 minor), the
  scheme corrects with the burnt characteristics; when it is not, the IDF
  falls back to the nominal scale factors and the scheme still succeeds -
  the driver never fails "missing eFuse" on this chip. `CaliEfuseVersion`
  reports which regime a board is in (0 = nominal, >= 1 = burnt). The
  emulator's chip has no block burnt (ver=0); the binding never invents a
  failure or a number the driver did not produce.
- **ADC2 and WiFi.** ADC2 shares the analog front end with the radio's
  power detector and the S31 has no arbiter: while WiFi is active, ADC2
  conversion results are not guaranteed. ADC1 (GPIO42..49) is unaffected -
  read sensors through ADC1 when the radio is up. Contention the driver
  does detect surfaces as `ESP_ERR_TIMEOUT` (0x107).
- **Lifecycle:** Init (claim the unit, fix the width) -> Config per pad
  (io->channel map + per-channel config) -> Read / ReadMillivolts. A read
  of a pad nobody configured reports `ESP_ERR_INVALID_STATE` (0x103) from
  the glue's bookkeeping; a second Init reports 0x103 (the family
  precedent); Deinit releases the unit and every later call on it reports
  `ErrNotInit` (0x7001).

## Layout

```
esp-idf-adc/
├── go.mod                  Go module (module llgoadc; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe (efuse regime → unit init → double-init
│                           refusal → pad configs + refusals → raw reads ×5 +
│                           millivolt reads ×5 on GPIO42, one round each on GPIO43
│                           → unconfigured-pad refusal → deinit refusals → teardown)
├── goapp/hosted.go         Host-build push-linkname stubs for the FreeRTOS
│                           externs in main.go, so `go build` links
├── idfproj/                Minimal ESP-IDF project
│   ├── sdkconfig.defaults  8 KiB main task, 240 MHz, PSRAM (no driver-specific
│   │                       switches: the oneshot ADC driver is a plain REQUIRES)
│   ├── partitions.csv      4 MB layout, 3.9375 MiB factory app (generated)
│   ├── host/
│   │   ├── make-artifact.sh  symlink to _demo/common; llgo build ->
│   │                       IDF-linkable relocatable object; also syncs
│   │                       pkg/esp/adc/glue into main/
│   │   └── artifact.conf    GLUES="adc"
│   └── main/
│       ├── CMakeLists.txt  Sources + esp_adc/esp_hal_ana_conv REQUIRES +
│       │                   WHOLE_ARCHIVE
│       ├── app_main.c      app_main -> llgo entry (generated from _demo/common)
│       ├── demo_glue.c     Demo-local glue (empty: no C helpers needed)
│       └── adc_glue.c      Package glue, synced from pkg/esp/adc/glue at
│                           build time (gitignored)
├── golden/emu-adc-probe.txt  Golden for the [go] adc lines (no floater)
└── README.md
```

## Build and run

Requirements: the llgo compiler (build with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-adc ./cmd/llgo`
from the llgo repo), docker with `espressif/idf:latest`, and esp-emu
(`~/.local/bin/esp-emu`, v1.2.0).

```bash
cd _demo/esp-idf-adc

# 1. Go -> relocatable object (also syncs pkg/esp/adc/glue into main/).
cd idfproj
LLGO_ROOT=<llgo-repo> ./host/make-artifact.sh /tmp/llgo-adc

# 2. IDF build (set-target first on a fresh tree; delete sdkconfig if you
#    edit sdkconfig.defaults, so the defaults re-apply). The merge-bin is
#    esptool directly - `idf.py merge-bin -f raw @flash_args` mis-orders
#    the raw format on this pipeline.
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
    bash -c 'idf.py set-target esp32s31 && idf.py build && \
    python -m esptool --chip esp32s31 merge-bin -o merged-binary.bin -f raw \
      --flash-mode dio --flash-freq 80m --flash-size 4MB \
      0x2000 build/bootloader/bootloader.bin \
      0x8000 build/partition_table/partition-table.bin \
      0x10000 build/llgo_idf_s31.bin && \
    mv -f merged-binary.bin build/merged-binary.bin'
cd ..

# 3. Acceptance run (one esp-emu instance per machine: the lock).
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --console usb-jtag --timeout 120s \
    --exit-on "[go] adc PASS"
rmdir /tmp/esp-emu.lock
```

The ELF matters as much as the binary here: the emulator reads the
application's DWARF from it, and a re-linked firmware without its ELF next to
it changes what the emulator can report.

## Acceptance

The run passes when the `[go] adc` lines, stripped of CR, match
`golden/emu-adc-probe.txt` exactly:

```
grep -E '^\[go\] adc' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-adc-probe.txt)
```

There is no floater rule: the probe prints no wall-clock values. Every value
is a build fact (pads, efuse regime, esp_err verdicts, sentinel codes) or
the emulator's reading shape - see Fidelity notes.

## Fidelity notes (measured, esp-emu v1.2.0)

The oneshot ADC path is modeled deep enough that the **whole driver path
runs green**: unit claim + handle allocation, the io->channel map, the
per-channel config, the oneshot conversion itself (the done event fires -
the read path has no timeout on this chip, it spins until then, and the
emulator delivers), the weight-accumulator bypass that exposes the 17-bit
comparator vector, the lazy line-fitting scheme creation (which on the
unburnt emu chip takes the nominal-scale fallback) and the channel-aware
millivolt conversion for both the N and the P side, plus calibration and
unit handle release at Deinit - all return esp_err 0, measured through this
binding in the acceptance run.

What the emulator cannot do is put a signal on a pad: with nothing driving
GPIO42/43, every conversion returns the emulator's fixed floating-input
vector (raw=67729, a stable 17-bit value) and the nominal model converts it
to 0 mV. The golden pins those values exactly because they are stable and
they pin the read path; they are not analog data. On hardware the same
rounds carry readings that track the applied voltage - verify the
millivolts against a known source on the board.

## Hardware bring-up checklist (not yet verified - emulator run only)

- [ ] Apply a known voltage (e.g. from the board's regulated rails through a
      divider) to GPIO42 and confirm `ReadMillivolts` reports it within the
      line-fitting scheme's tolerance; repeat on GPIO43 (the P side) to
      exercise the other characteristics.
- [ ] Check `CaliEfuseVersion` on a production board: >= 1 expected where
      the calibration block is burnt; record the observed accuracy in both
      regimes.
- [ ] ADC2 sanity: read a GPIO50..57 pad with the radio down, then with WiFi
      scanning/associated - confirm the documented "not guaranteed while
      WiFi is active" boundary and the `ESP_ERR_TIMEOUT` (0x107) contention
      surface.
- [ ] Cross-pad crosstalk at 17 bits: read a grounded pad next to a driven
      one and record the comparator vector's behavior (the differential-pair
      readout is new silicon; characterization data is thin).
- [ ] Long-run stability: thousands of oneshot reads (sensor polling shape)
      watching for drift or lock contention with the temperature sensor
      (the ADC shares its power domain with PWDET/TempSensor on this chip).
