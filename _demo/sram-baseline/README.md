# ESP32-S31 baremetal SRAM baseline

The baremetal half of the SRAM budget: no ESP-IDF, no WiFi, just llgo's own
fiber runtime on the 512 KiB DIRAM window (`esp32s31-basic` target). Paired
with `_demo/esp-idf-wifi`, which measures the same program shape with the
IDF heap pool and the WiFi stack in play.

## Layout

```
sram-baseline/
├── go.mod            Go module (module srambase)
├── main.go           Probe: workload + tinygogc accounting prints
└── build_flash.py    ELF -> ESP image at flash offset 0x2000 (esp-emu input)
```

## What the probe does

Runs the workload the WiFi demo runs before its WiFi stack comes up
(strings, slices, goroutine + channel, prints), adds the WiFi pipeline's
live-set shape (20 scan records, ssid/pass strings), then five rounds of
8 KiB allocate / release / `runtime.GC()`, printing the tinygogc accounting
at every node:

- `_heapStart`/`_heapEnd` and the bootstrap stack window via
  `//go:linkname` on the linker script symbols;
- `runtime.ReadMemStats` (backed by `tinygogc.ReadGCStats`): `Sys` is the GC
  window the runtime reserved, `HeapInuse`/`HeapIdle` split that window,
  `GCSys` is collector metadata, `StackSys` is the linker bootstrap stack
  window (`StackInuse` measures that context only — while `main` runs it
  executes on a GC-allocated 8 KiB fiber stack that counts in `HeapInuse`).

## Build and run

The repo's S31 emu suite pins `esp32s31-basic` for SRAM-only runs; the plain
`esp32s31` target (device C + newlib variant, same linker window) traps in
`_exit` before its first print under esp-emu, so it is not used here.

```
/tmp/llgo-wifi build -target esp32s31-basic -o app.elf .
python3 build_flash.py app.elf flash.bin
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware flash.bin --console uart0 \
    --timeout 30s --exit-on "[go] sram done"
rmdir /tmp/esp-emu.lock
```

Static segments come from the linked ELF:

```
llvm-size -A app.elf        # .text / .rodata / .stack / .data / .bss
llvm-nm app.elf | grep _heap   # _heapStart / _heapEnd window bounds
```

## Closing the ledger

The linker script places code, the NOLOAD `.stack` window, `.data` and
`.bss` consecutively from `0x2f000000`, then `_heapStart = _end`; `_heapEnd`
is the top of the 512 KiB DIRAM window. The runtime reserves the first
2 KiB after `_heapStart` (unwrappable libc allocation) and uses the rest as
the GC window, so:

```
static segments + 2048 reserve + GC window = 524288 B (512 KiB)
```

Measured on this probe: 108280 + 2048 + 413960 = 524288.
