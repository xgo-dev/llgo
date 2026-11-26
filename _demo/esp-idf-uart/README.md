# llgo × ESP-IDF UART (ESP32-S31)

A self-contained pipeline that drives the ESP32-S31's UART serial ports from
Go code compiled by llgo, with the acceptance run automated under esp-emu.
The Go binding lives in the llgo repository (`pkg/esp/uart`), consumed through
the module require + replace in `go.mod`; this directory owns the acceptance
program and the IDF project.

## Chip facts this demo is built on

- **Ports**: the S31 ships **five** UART ports (SOC_UART_NUM = 5, 128-byte
  hardware FIFOs each). UART0 is the console - the boot ROM and the IDF
  default console drive it - so the binding refuses `Init` on port 0 outright
  (`ErrConsoleReserved`, a binding-side refusal, not a driver verdict). The
  demo runs the application traffic on **UART1**, routed through the GPIO
  matrix onto GPIO17 (TX) and GPIO18 (RX): free header pins on
  Function-CoreBoard-1 (GPIO6/7 are Ethernet; GPIO10/12 belong to the touch
  demo).
- **Install shape**: the glue installs each port with a 256-byte RX ring
  buffer and a 256-byte TX ring buffer and **no event queue** - the
  synchronous-polling shape `pkg/esp/uart` documents. The TX ring is what
  keeps `Write` enqueue-and-return: the driver's own ISR feeds the hardware
  FIFO from the ring, so a write never waits on the wire clock (without a TX
  ring, `uart_write_bytes` parks on a FIFO-space semaphore that only the TX
  interrupt gives - a stalled line would hold the single llgo task forever).
- **Read timeout**: `Read`'s budget is milliseconds, quantized by the driver
  to FreeRTOS ticks (10 ms at the default 100 Hz). An expired budget with no
  byte received is the idiomatic `(0, ErrTimeout)` - `ErrTimeout` rides the
  esp_err family's own `ESP_ERR_TIMEOUT` (0x107), the code `uart_wait_tx_done`
  reports for an expired flush budget too.

## Layout

```
esp-idf-uart/
├── go.mod                  Go module (module llgouart; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe (console + not-init refusals;
│                           Init→Write→Available→Read timeout→FlushInput→
│                           FlushOutput→SetConfig→SetBaud→Deinit→post-deinit
│                           refusal; no FreeRTOS extern, so no host stub file)
├── idfproj/                Minimal ESP-IDF project
│   ├── sdkconfig.defaults  8 KiB main task, 240 MHz, PSRAM (no driver-specific
│   │                       switches: the UART driver is a plain REQUIRES)
│   ├── partitions.csv      4 MB layout, 3.9375 MiB factory app (generated)
│   ├── host/
│   │   ├── make-artifact.sh  symlink to _demo/common; llgo build ->
│   │                        IDF-linkable relocatable object; also syncs
│   │                        pkg/esp/uart/glue into main/
│   │   └── artifact.conf    GLUES="uart"
│   └── main/
│       ├── CMakeLists.txt  Sources + esp_driver_uart REQUIRES + WHOLE_ARCHIVE
│       ├── app_main.c      app_main -> llgo entry (generated from _demo/common)
│       ├── demo_glue.c     Demo-local glue (empty: no C helpers needed)
│       └── uart_glue.c     Package glue, synced from pkg/esp/uart/glue at
│                           build time (gitignored)
├── golden/emu-uart-probe.txt  Golden for the [go] uart lines (no floater)
└── README.md
```

## Build and run

Requirements: the llgo compiler (build with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-uart ./cmd/llgo`
from the llgo repo), docker with `espressif/idf:latest`, and esp-emu
(`~/.local/bin/esp-emu`, v1.2.0). The runtime's vendored riscv32 libffi
archive is a gitignored build input: copy it from a working checkout
(`runtime/internal/clite/ffi/riscv32/libffi.a`) if the tree is fresh.

```bash
cd _demo/esp-idf-uart

# 1. Go -> relocatable object (also syncs pkg/esp/uart/glue into main/).
cd idfproj
LLGO_ROOT=<llgo-repo> ./host/make-artifact.sh /tmp/llgo-uart

# 2. IDF build (set-target first on a fresh tree; delete sdkconfig if you
#    edit sdkconfig.defaults, so the defaults re-apply).
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
    bash -c 'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'
cd ..

# 3. Acceptance run (one esp-emu instance per machine: the lock).
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --console usb-jtag --timeout 120s \
    --exit-on "[go] uart PASS"
rmdir /tmp/esp-emu.lock
```

The ELF matters as much as the binary here: the emulator reads the
application's DWARF from it, and a re-linked firmware without its ELF next to
it changes what the emulator can report.

## Acceptance

The run passes when the `[go] uart` lines, stripped of CR, match
`golden/emu-uart-probe.txt` exactly:

```
grep -E '^\[go\] uart' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-uart-probe.txt)
```

There is no floater rule: the probe prints no wall-clock values. Every value
is a build fact (port, pins, baud rates, the 9-byte payload) or an esperr
pipeline code printed in decimal (30721 = 0x7801 console refusal, 30722 =
0x7802 not-init refusal, 263 = 0x107 timeout) - see Fidelity notes.

## Fidelity notes (measured, esp-emu v1.2.0)

The emulator models no UART peer and no wire. The driver path runs green -
`uart_driver_install` (its object, both ring buffers, the interrupt
allocation), `uart_param_config`, `uart_set_pin` (GPIO matrix routing),
`uart_write_bytes` (TX ring enqueue), the word-length/parity/stop-bits
writes, `uart_set_baudrate`, both flushes, `uart_driver_delete` all return
ESP_OK - but nothing is ever received:

- `available n=0` on every poll: the RX ring stays empty. The written 9 bytes
  sit in the TX ring/FIFO with nowhere to go.
- `read n=0 err=263 timeout`: the Read budget is a real 20-tick semaphore
  wait under the emulator's scheduler, expiring into `ErrTimeout` - the
  idiomatic `(0, ErrTimeout)` shape, exercised end to end.
- `flush output err=0`: `uart_wait_tx_done` finds the (fixed-state) TX FIFO
  already empty and returns immediately. An expired budget would print 263;
  the golden accepts only what actually happened.

There is no UART loopback or bridging model in the emulator to make the
written bytes come back, so the receive-data path (bytes arriving, ring
contents, partial fills) stays a hardware item - see the checklist. The
PASS gate proves the Go→glue→driver→register link; it does not prove a wire.

## Hardware bring-up checklist (not yet verified - emulator run only)

- [ ] **Loopback**: jumper GPIO17 (TX) to GPIO18 (RX), run the same probe.
      Expect `available n=9` and the read-back to return the 9 payload bytes
      (`read n=9 err=0 ok`) - a partial fill of the 16-byte buffer, proving
      bytes arrive, ring contents and the partial-fill shape.
- [ ] Two-board exchange at the probe's two rates (115200 then 9600 after
      `SetBaud`): confirm the rate change is visible on the wire (an
      oscilloscope or a peer that re-syncs).
- [ ] Frame formats: `SetConfig` to 7E2/8O1 against a peer configured the
      same; confirm parity/stop errors surface as the driver's framing
      behavior, not silent garbage.
- [ ] Back-pressure: write enough payload to fill the 256-byte TX ring
      against a slow peer; confirm `Write` blocks and resumes (and that
      `FlushOutput` bounds a stalled drain with `ErrTimeout`).
- [ ] The console guard on a UART0-console board: confirm `Init(UART0, ...)`
      refuses and the console keeps printing throughout the run.
