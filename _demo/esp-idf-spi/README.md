# llgo × ESP-IDF SPI master (ESP32-S31)

A self-contained pipeline that drives the ESP32-S31's general-purpose SPI
master (GPSPI2, `SPI2_HOST`) from Go code compiled by llgo, with the
acceptance run automated under esp-emu. The Go binding lives in the llgo
repository (`pkg/esp/spi`), consumed through the module require + replace in
`go.mod`; this directory owns the acceptance program and the IDF project.

## Chip facts this demo is built on

- The S31 exposes **one** general-purpose SPI master (`SPI2_HOST` = the GPSPI
  the IDF docs call "general purpose"; `SPI1_HOST` is the flash controller).
  The binding is therefore a single-instance, package-level API: the C glue
  owns the bus and the one device in static storage.
- The bus always initializes **with DMA** (`SPI_DMA_CH_AUTO`), so transfers
  are not capped at the 64-byte hardware FIFO; the driver's default
  per-transaction ceiling is 4092 bytes (a larger transaction reports
  ESP_ERR_INVALID_ARG). Caller slices flow straight into the driver: IDF
  v6.3 stages DMA-unfriendly buffers (unaligned, PSRAM placement) through
  its own temporary copies when the transaction does not opt out
  (`SPI_TRANS_DMA_BUFFER_ALIGN_MANUAL` is never set). One property stays
  Go-side: the SPI DMA engine writes MISO data in whole 32-bit words (the RX
  descriptor size is the byte length rounded up to four), so a rx slice
  whose length is not a multiple of four goes through one word-rounded
  staging copy - the hardware never writes past a Go buffer. The probe
  exercises that path with its `transfer rx5` round.
- The clock source (BBPLL) reaches the peripheral at most 80 MHz, and the
  divider is integer-only. The binding accepts exactly the power-of-two
  divisions 80/40/20/10 MHz - the rates the divider reproduces **exactly** -
  and refuses everything else with `ErrBadFrequency` instead of silently
  rounding (the driver alone would round a request down to the nearest
  step). `ActualFreqKHz` echoes the driver's math back; under the emulator
  it reads 10000/20000 kHz for the probe's rates.
- `SetFrequency` rebinds the device (remove + re-add - the driver has no
  in-place speed change), refused with `ErrBusy` while AcquireBus holds the
  bus. `AcquireBus`/`ReleaseBus` take the driver's bus-acquire lock and pin
  CS across the held transfers (IDF's `SPI_TRANS_CS_KEEP_ACTIVE`); the
  release drops CS with one zero-phase transaction before handing the bus
  back. Note the driver refuses finite acquire waits outright ("acquire
  finite time not supported now"), so the lock wait is `portMAX_DELAY` -
  uncontended in an llgo program (one FreeRTOS task, one device).
- Transfers ride the driver's polling path with a bounded completion wait
  (`spi_device_polling_end`, 300 ms): a real transaction completes in
  microseconds to a few milliseconds with the llgo task spinning inside
  (exclusive by construction under the cooperative scheduler), and a stuck
  transaction - reachable only where the peripheral is absent - ends with
  ESP_ERR_TIMEOUT instead of hanging forever.

## Layout

```
esp-idf-spi/
├── go.mod                  Go module (module llgospi; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe (init → actual freq → set-freq rounds
│                           → write/transfer/writeread incl. pad + staged-rx rounds
│                           → refusal rounds → CS-pinned session → deinit)
├── goapp/hosted.go         Host-build push-linkname stubs for the FreeRTOS
│                           externs in main.go, so `go build` links
├── idfproj/                Minimal ESP-IDF project
│   ├── sdkconfig.defaults  8 KiB main task, 240 MHz, PSRAM (no driver-specific
│   │                       switches: the SPI master driver is a plain REQUIRES)
│   ├── partitions.csv      4 MB layout, 3.9375 MiB factory app (generated)
│   ├── host/
│   │   ├── make-artifact.sh  symlink to _demo/common; llgo build ->
│   │                       IDF-linkable relocatable object; also syncs
│   │                       pkg/esp/spi/glue into main/
│   │   └── artifact.conf    GLUES="spi"
│   └── main/
│       ├── CMakeLists.txt  Sources + esp_driver_spi REQUIRES + WHOLE_ARCHIVE
│       ├── app_main.c      app_main -> llgo entry (generated from _demo/common)
│       ├── demo_glue.c     Demo-local glue (empty: no C helpers needed)
│       └── spi_glue.c      Package glue, synced from pkg/esp/spi/glue at
│                           build time (gitignored)
├── golden/emu-spi-probe.txt  Golden for the [go] spi lines (no floater)
└── README.md
```

## Build and run

Requirements: the llgo compiler (build with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-spi ./cmd/llgo`
from the llgo repo), docker with `espressif/idf:latest`, and esp-emu
(`~/.local/bin/esp-emu`, v1.2.0).

```bash
cd _demo/esp-idf-spi

# 1. Go -> relocatable object (also syncs pkg/esp/spi/glue into main/).
cd idfproj
LLGO_ROOT=<llgo-repo> ./host/make-artifact.sh /tmp/llgo-spi

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
    --exit-on "[go] spi PASS"
rmdir /tmp/esp-emu.lock
```

The ELF matters as much as the binary here: the emulator reads the
application's DWARF from it, and a re-linked firmware without its ELF next to
it changes what the emulator can report.

## Acceptance

The run passes when the `[go] spi` lines, stripped of CR, match
`golden/emu-spi-probe.txt` exactly:

```
grep -E '^\[go\] spi' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-spi-probe.txt)
```

There is no floater rule: the probe prints no wall-clock values. Every value
is a build fact (pins, kHz, esp_err verdicts, sentinel codes) or the
received-byte shape - see Fidelity notes.

## Fidelity notes (measured, esp-emu v1.2.0)

The GPSPI master is modeled deep enough that the **whole driver path runs
green**: bus init with DMA channel allocation, device add, the clock
divider, polling transactions (tx-only, full duplex, rx-only - the
trans-done interrupt fires), the bus-acquire lock, CS-pinned keep-active
transactions, the zero-phase CS drop, device remove and bus free, all
return esp_err 0 (measured first with a pure-C probe of the same calls,
then through the binding in this acceptance run).

What the emulator cannot do is put a slave on the bus, so the one honest
caveat is the received data: with nothing driving MISO, the sampled line
floats high and **every received byte reads ff**. The rx= lines in the
golden pin the buffer shapes and the receive path (lengths, the zero-pad
round, the word-rounded staging round, the two-half WriteRead, the held
session); the ff bytes themselves are the floating line, not device data.
On hardware the same rounds carry the actual device answer (a JEDEC ID in
response to 0x9F, register data in response to a command) - verify them on
the board.

## Hardware bring-up checklist (not yet verified - emulator run only)

- [ ] Wire a real device (flash chip, sensor) to MOSI 11 / MISO 12 / SCLK 13
      / CS 10 and read its ID through `WriteRead` under `AcquireBus`: expect
      the device's answer bytes in rx instead of ff.
- [ ] Frequency sweep: `Freq80MHz`/`Freq40MHz` on a device that tolerates it
      (GPIO-matrix-routed pins add input delay; IDF's guidance on >8 MHz and
      `input_delay_ns` applies - the binding uses GPIO matrix routing, not
      IOMUX, for arbitrary pins).
- [ ] CS integrity on a scope: hardware CS toggling per transaction in the
      plain rounds; CS staying low across a held session (write → transfer →
      writeread) and rising exactly once at `ReleaseBus` (the zero-phase
      drop), with no extra clock edges during the drop.
- [ ] Large transfers near the 4092-byte ceiling, and the driver's
      ESP_ERR_INVALID_ARG just past it.
- [ ] DMA behavior with PSRAM-resident buffers (the Go heap lives in PSRAM):
      IDF v6.3 stages them through internal DMA-capable copies; confirm
      throughput and stability on the board.
- [ ] `SetFrequency` under load: the remove/re-add rebind keeps transfers
      correct across the switch (the CS slot is re-picked).
