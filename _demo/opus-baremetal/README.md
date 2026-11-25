# pkg/esp/opus baremetal (no-ESP-IDF) acceptance · ESP32-S31

pkg/esp/opus with its vendored fixed-point Opus sources (`pkg/esp/opus/csrc/`,
compiled by the llgo toolchain through `LLGoFiles`) driving a synthetic-tone
encode → decode → SNR probe. No ESP-IDF component manager, no IDF project:
`llgo build -target esp32s31` is the whole pipeline. The paired IDF acceptance
run is `_demo/esp-idf-opus` — same probe shape, same gates, so the two build
paths stay honest against each other.

## What is under test

The probe is the ESP-IDF acceptance's tone48k segment: 48 kHz mono, music
application, 48 kbps, 50 × 20 ms frames, SNR compared against the encode
input shifted by the codec's own lookahead, plus packet-loss concealment and
the encoder ctls. Measured under esp-emu, both baremetal targets land exactly
on the IDF run's numbers (SNR 39.3 dB, max error 399, RMS 91, delay 312,
encoded bytes 6101) — the vendored clang build and the IDF gcc build of the
same source tree are quality-identical.

## Two firmware layouts

| target | code | SRAM heap left | measured load |
| --- | --- | --- | --- |
| `esp32s31` (SRAM) | 313,056 B loaded into the 512 KiB DIRAM window | ~188 KiB (the 501,248 B budget minus the load) | boots from flash offset 0x2000 |
| `esp32s31-flash` (XIP) | code executes from the flash mapping; SRAM carries data/bss/heap | most of 512 KiB | boots through the ESP-IDF second-stage bootloader from offset 0x10000 |

The SRAM budget is the binding one: 313 KiB of load leaves ~188 KiB for the
heap, and the probe itself is sized to it (a two-frame reference ring for the
aligned compare instead of the IDF demo's whole-input copy; the codec states,
the 64 KiB main fiber and the probe's buffers total ~115 KiB). A real
consumer that streams audio should prefer the XIP target — the codec's code
stops competing with the heap for SRAM.

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-opus`, built with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-opus ./cmd/llgo`),
esp-emu v1.2.0. The XIP merge also needs the bootloader and partition table
of any ESP-IDF build for esp32s31 (the paths below take them from the paired
demo's build directory).

```
# SRAM layout: boots from flash offset 0x2000
/tmp/llgo-opus build -o app.elf -target esp32s31 -goroutine-stack-size 16384 .
python3 build_flash.py app.elf flash.bin
timeout 150 ~/.local/bin/esp-emu --chip esp32s31 --firmware flash.bin --timeout 120s

# XIP layout: boots through the IDF second-stage bootloader (offset 0x10000)
/tmp/llgo-opus build -o appxip.elf -target esp32s31-flash -goroutine-stack-size 16384 .
python3 build_flash.py appxip.elf flashxip.bin --xip \
    ../esp-idf-opus/idfproj/build/bootloader/bootloader.bin \
    ../esp-idf-opus/idfproj/build/partition_table/partition-table.bin
timeout 150 ~/.local/bin/esp-emu --chip esp32s31 --firmware flashxip.bin \
    --timeout 120s --console usb-jtag
```

Expected console (both layouts):

```
[go] opusbm start
[go] opusbm states enc1 = 24716 dec1 = 18136 enc2 = 29628 dec2 = 26856
[go] opusbm validation ok rate44100-refused
[go] opusbm seg tone48k frames = 50 enc = 6101 dec = 48000 snr = 39.3 dB max = 399 rms = 91 delay = 312
[go] opusbm plc conceal = 960 of 960
[go] opusbm plc ok resync = 960
[go] opusbm seg tone48k PASS
[go] opusbm ctl ok bad-arg frame-table length-guard
[go] opusbm PASS
[go] opusbm done
```

## Stacks

Same contract as the IDF demo: `-goroutine-stack-size 16384` (main fiber
64 KiB) — opus_encode/opus_decode are deep C calls with alloca scratch in the
tens of KiB, and the fiber guard only fires at cooperative swap points, so a
default 16 KiB main fiber silently corrupts the heap below its floor.
