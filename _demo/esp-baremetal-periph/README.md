# ESP baremetal peripherals (no ESP-IDF)

The runnable artefact behind `pkg/esp`'s non-IDF backend: a plain
`llgo build -target ...` binary that drives GPIO and hardware I2C by writing
registers, with **no ESP-IDF and no FreeRTOS in the link**. The `_demo/esp-idf-*`
projects cover the other half (the IDF driver through the C glue); this one is
the same two API calls a baremetal program makes and nothing else.

## What it does

Each check prints one fixed line:

| line | meaning |
| --- | --- |
| `gpio loopback ok` / `gpio loopback mismatch high=.. low=..` | GPIO3 is set to `Output`, driven high, read back, driven low, read back |
| `i2c scan found=N` | `Bus.Scan(0x08, 0x77)` - one address-only probe per usable 7-bit address |
| `i2c scan addr=0x..` | one line per address that acknowledged |
| `i2c register read reg=0x00 val=0x..` | only when the scan found something: read register 0x00 of the first device |
| `i2c bulk write len=41 ok` / `i2c bulk read len=40 ok` / `i2c bulk verify ok` | one payload wider than the 32-byte FIFO each way, written to registers 0x00..0x27 and compared byte for byte |

The GPIO check is the pad's own read-back (the input buffer stays on for an
`Output` pin, which is what `pkg/esp/gpio` documents), so it needs no extra
wiring. The I2C check uses the ESP32-C3 devkit pin pair, SDA = GPIO8 and
SCL = GPIO9.

## Build

```
llgo build -target esp32s31 -obin -o app.bin .     # SRAM image, flash at 0x2000
llgo build -target esp32c3  -obin -o app.bin .     # C3
llgo build -target esp32s31 -o app.elf .           # ELF, for the emulator
```

Nothing here imports ESP-IDF, so there is no `idf.py`, no component tree and no
`llgo.idf` build tag involved.

## Run under esp-emu

The emulator wants a merged 4 MiB flash dump, which the SRAM-baseline demo's
script produces from the linked ELF:

```
python3 ../sram-baseline/build_flash.py app.elf flash.bin
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware flash.bin \
    --console usb-jtag --timeout 90s
rmdir /tmp/esp-emu.lock
```

`--console usb-jtag` is not optional for the full `esp32s31` target: its device C
sends the Go console to the USB-Serial/JTAG console, and forcing `--console uart0`
swallows every print (`esp32s31-basic` targets the UART and does print under
`--console uart0`).

Compare the output against the golden line for line (the console emits `\r\n`,
so strip the CR; the emulator's own `[INFO ...]` log lines and the ROM banner are
not part of the comparison):

```
esp-emu ... | grep -E '^(esp-baremetal-periph|gpio loopback|i2c scan|i2c register|i2c bulk)' \
    | tr -d '\r' | diff -u golden/esp32s31.txt -
```

## What the emulator proves, and what it does not

Verified under `esp-emu` (see the run in the repository's report for this
change):

- the whole register sequence executes without a fault: clock gate, pin matrix,
  bus clear, command list, `INT_RAW` polling, FIFO drain;
- `gpio loopback ok` - **only** that the output-enable/level register
  sequence runs to completion without a fault. The S31 model mirrors
  `GPIO_OUT` into `GPIO_IN` unconditionally (a pin that was never set to
  output still reads high), so this line does **not** prove the direction
  register took effect, and it cannot catch a read-modify-write on a
  write-only register. On real hardware the same line needs the pad's
  input path to actually see the driven level.
- `i2c scan found=1` with `addr=0x50` - the S31 model **does** model the ACK
  bit on this path: an absent address raises `NACK` (the driver reports
  `esp/i2c: address not acknowledged`, `esperr.Code` 0x105) for a probe, a
  write, a read and a write-read alike, and only 0x50 answers. A command list
  that mis-encoded the write command would ACK everywhere or nowhere, so this
  is a real check of the command encoding and of the NACK diagnosis;
- `i2c register read reg=0x00 val=0xff` - the model fills the RX FIFO, so the
  read path (repeated start, `READ` command, NACK on the last byte, FIFO drain)
  runs too. Measured against the model: a `Write([]byte{0x00, 0x11})` to 0x50
  followed by a `WriteRead([]byte{0x00}, buf)` returns 0x11, so the emulator
  implements a byte-addressed register file at 0x50 and 0xFF is its erased
  value - a model reading, not a real part's.

**Not** verified, because the emulator cannot:

- real bus timing. The S31 model does not model the clock at all, so 100 kHz vs
  400 kHz, setup/hold times and the 9-pulse bus clear are unobservable. The
  timing numbers are transcribed from the ESP-IDF HAL and unit-tested on the
  host (`pkg/esp/i2c/i2c_timing_test.go`); the waveform itself is unverified.
- a real device's behaviour: register maps, clock stretching, arbitration.
- the ESP32-C3 at all: `esp-emu`'s C3 model implements no peripheral MMIO and
  no clock, so the C3 build is compile-checked only. That includes the
  post-timeout recovery: on the C3 the controller has no hardware FSM reset
  (the C3 `i2c_ll.h` does not define `I2C_LL_SUPPORT_HW_FSM_RST`), so the
  driver's timeout path does the block-level reset the IDF uses there
  (`i2c_ll_reset_register` + re-init + bus clear, mirroring
  `s_i2c_hw_fsm_reset` in `esp_driver_i2c/i2c_master.c:111-146`). Whether
  the controller comes back cleanly after a timeout on a real C3 is
  **unverified**, not known-good.

## What real hardware needs

- **Pull-ups.** The backend leaves the internal pull-ups on, which is enough to
  see idle-high on a short bench wire, but a 400 kHz bus needs external 2.2-4.7
  kΩ pull-ups to SDA and SCL. A bus with no pull-ups at all reads as
  permanently busy and every transfer times out.
- **Wiring for the scan.** SDA to the device's SDA, SCL to its SCL, and a common
  ground. `Bus.Scan` reports an address only if the device acknowledges its
  address byte, so a device that is present but unpowered is a gap in the
  result, not an error.
- **A separate loopback pin if you want a two-pin check.** The built-in check
  reads the pad back, which - like every GPIO line here - neither proves the
  pad wiring nor the direction register (the model mirrors `GPIO_OUT` into
  `GPIO_IN` unconditionally, see Golden); it only shows the register sequence
  completes. Wiring GPIO3 to another input pin is the real end-to-end test.
- **Flash offsets.** `llgo build -obin` produces the application image for flash
  offset 0x2000 (`targets/esp32s31.json`'s `flash-command`), not a bootloader.

## Golden

`golden/esp32s31.txt` holds the exact application output of the run above
(emulator log lines filtered out). The lines are fixed: the GPIO line reports
`ok`, the scan reports exactly one device at 0x50, and the register read reports
0xFF. A change in any of them is a change in behaviour, not noise - the run is
deterministic across repeated emulator invocations.
