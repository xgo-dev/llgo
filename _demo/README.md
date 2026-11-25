# ESP32-S31 / ESP-IDF demos

One firmware per question. Two of them are comprehensive (many capabilities at
once, flashed to a board by hand), the rest are the smallest program that
exercises one capability plus its emulator gate.

| demo | what it answers | how it runs |
| --- | --- | --- |
| [`esp-idf-all`](esp-idf-all/) | Does everything work *together* on real silicon? Soft AP + DHCP + HTTP server, A2DP sink into the ES8311/NS4150B speaker path, a BLE HID-keyboard central on the same Bluedroid stack, and the codec self-test - one flash, one board session. | real board (`host/make-artifact.sh` → `idf.py build` → `esptool write-flash`) |
| [`esp-idf-wifi`](esp-idf-wifi/) | Is the whole network stack correct? Station join, lwIP TCP/UDP, TLS (both auth modes + a wrong-CA negative), WebSocket (plain, `wss` permissive, `wss` strict), reconnect, plus the memory ledger and the PSRAM/GC measurements. | real board, or esp-emu against the host peers |
| [`esp-idf-ap`](esp-idf-ap/) | Smallest soft-AP program: bring the AP up, watch a station associate, serve one request. | real board, or esp-emu (`host/run-acceptance.sh`) |
| [`esp-idf-ble`](esp-idf-ble/) | Smallest BLE central: init, scan, connect, disconnect, deinit. | esp-emu with a Bumble peer (`--ble-hci tcp:...`), no hardware needed |
| [`esp-idf-bt`](esp-idf-bt/) | Smallest classic-Bluetooth program: BTDM bring-up, SPP, and the A2DP sink data path. | esp-emu with the vendored Bumble classic peer |
| [`esp-idf-lpcore`](esp-idf-lpcore/) | Smallest LP-coprocessor program: HP/LP mailbox echo, LP-timer deep-sleep wake. | real board for the wake path (esp-emu models the LP RAM but not an executing LP hart) |
| [`esp-idf-lockcore`](esp-idf-lockcore/) | `runtime.LockCore` pins the llgo scheduler task to an HP core. | esp-emu for the pinning contract, real board for the timing |

Choosing:

- **Something is broken and you need to know which layer** → the minimal demo
  for that layer; it has the smallest possible surface and an emulator gate.
- **Everything works and you want one firmware that proves it** →
  `esp-idf-all`.
- **Network/TLS/WebSocket work** → `esp-idf-wifi`; it is the only demo with the
  full transport matrix and the memory ledger.

## Shared anatomy

Every demo has the same three-layer shape, and the layers are deliberately
separate:

```
<demo>/goapp/          the Go program (one file per area; see each README)
<demo>/idfproj/main/   the C shell: app_main.c, the extra linker script, and
                       the glue sources the build syncs from pkg/esp/*/glue
<demo>/idfproj/host/   make-artifact.sh: builds goapp with llgo, renames the
                       compiler-rt builtins, and merges host/llgo_syms.S so the
                       runtime's symbol providers live inside llgo_out.o
```

The glue C files under `idfproj/main/` are **build inputs**, not sources of
truth: `make-artifact.sh` copies them from `pkg/esp/*/glue/` on every run
(integration way 1 in [`pkg/esp/README.md`](../../pkg/esp/README.md)). Edit the
package template, never the copy.

## Memory on the console

Both comprehensive demos print a ledger line at every bring-up stage and then
periodically, so a long run shows the trend rather than only the watermarks:

```
[go] heap periodic internal free = 251724 min = 249512 | total free = ... | psram = ... | dma max = ...
```

`internal free` is the radio's headroom (WiFi and the Bluetooth stacks allocate
from internal RAM only), `min` is the low-water mark since boot - the number
that explains a later allocation failure - and `dma max` is the largest free
contiguous DMA block, which is what predicts whether the next packet still
fits.
