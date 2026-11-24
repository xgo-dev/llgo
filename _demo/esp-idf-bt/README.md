# llgo × ESP-IDF classic Bluetooth pipeline (ESP32-S31): Bluedroid SPP server

A self-contained pipeline that drives ESP-IDF's Bluedroid classic Bluetooth
stack from Go code compiled by llgo: controller + host bring-up, the device
name and discoverability, an SPP (Serial Port Profile) server on RFCOMM, the
echo of a peer's payload with byte-pattern verification, the peer-initiated
close and the stack shutdown - with the acceptance run automated under esp-emu
against a virtual classic Bluetooth peer hosted by Google Bumble. The Go
bindings live in the llgo repository (`pkg/esp/bt`, consumed through the module
require + replace in `go.mod`); this directory owns the acceptance program and
the IDF project.

This is the SPP server (acceptor) role. Other classic profiles (A2DP, HFP) and
the SPP client/initiator role are later stages; the lifecycle, event mailbox
and error domain in `pkg/esp/bt` are profile-shared, so those stages add API
surface without redesign. See `pkg/esp/README.md` for the package contract.

## Layout

```
esp-idf-bt/
├── go.mod                  Go module (module llgobt; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: Init → SetName → SetDiscoverable →
│                           SPPListen (SCN 1) → SPPAccept → echo loop with
│                           pattern check → peer close → Deinit
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── golden/emu-bt-probe.txt Golden for the [go] bt lines (+ the floater rule)
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt
    ├── sdkconfig.defaults  Classic-only: Bluedroid + BTDM controller + SPP,
    │                       no BLE host, Bluedroid pools from PSRAM, 240 MHz,
    │                       PSRAM, 3.9 MiB app partition
    ├── partitions.csv
    ├── host/
    │   ├── make-artifact.sh  llgo build → IDF-linkable relocatable object; also
    │   │                     syncs pkg/esp/bt/glue/bt_glue_bluedroid.c into main/
    │   ├── llgo_syms.S       GC window + bootstrap stack providers (window size
    │   │                     from LLGO_GC_WINDOW_KB, default 64 KiB)
    │   └── bumble_classic_peer.py  Host-side classic peer: a Bumble virtual
    │                         controller for the emulator's HCI stream plus the
    │                         SPP client that discovers the firmware's record
    │                         over SDP and drives it (a copy of esp-emulator's
    │                         tools/bumble_classic_peer.py; the docstring names
    │                         the one delta)
    └── main/
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── app_main.c      app_main → llgo entry; _write/_exit shims; Go heap
        │                   PSRAM budget (llgo_gc_ext_heap_cap)
        ├── demo_glue.c     Demo-local heap marks (both IDF pools)
        ├── bt_glue.c       Bluedroid backend glue, synced from
        │                   github.com/xgo-dev/llgo/pkg/esp/bt/glue (gitignored)
        └── llgo_dram.ld    _globals_start/_globals_end for tinygogc
```

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-x` below), Docker image
`espressif/idf:latest` (the only supported build environment - it pins the
IDF v6.3-dev API the glue codes against), esp-emu v1.2.0 (the first release
with a classic Bluetooth HCI backend), and a Python venv with `bumble` for the
host peer.

```
# 1. Compile the Go probe into the IDF-linkable object. The script also syncs
#    pkg/esp/bt/glue/bt_glue_bluedroid.c into main/, so run it before idf.py.
cd _demo/esp-idf-bt
LLGO_ROOT=$(cd ../.. && pwd) bash idfproj/host/make-artifact.sh /tmp/llgo-x

# 2. IDF firmware (set-target pins the chip: the llgo artifact is compiled
#    for esp32s31-idf, and idf.py's default esp32 would fail the link with an
#    ELF-class mismatch; merge-bin takes -o relative to build/: the image
#    lands in build/merged-binary.bin, the ELF the emulator needs in
#    build/llgo_idf_s31.elf)
docker run --rm -v "$PWD/idfproj":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Start the host peer BEFORE the emulator: it owns the virtual BR/EDR
#    controller the emulator connects out to (tcp-server :9544), discovers the
#    firmware's SPP record over SDP, connects, sends 25 x 512 bytes
#    (pattern 0x30 + (i % 10) repeated per 512-byte record), then disconnects
#    the RFCOMM DLC and the ACL. BUMBLE_SPP_PASS on its console is the
#    peer-side completion marker. The peer is this tree's host/ copy of
#    esp-emulator's bumble_classic_peer.py with one delta (see its docstring):
#    it drops the ACL before exiting, because the firmware can never observe
#    the process vanishing and Bluedroid cannot tear down under an open ACL.
/tmp/bumble-venv/bin/python idfproj/host/bumble_classic_peer.py spp --wait 6 &

# 4. Acceptance run. --elf is mandatory: the emulator intercepts the
#    firmware's BTDM HCI driver through the ELF's symbols, so the ELF must be
#    the unstripped build that produced the merged image.
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --ble-hci tcp:127.0.0.1:9544 \
    --console usb-jtag --timeout 120s --psram-size 16M \
    --exit-on "[go] bt done"
rmdir /tmp/esp-emu.lock
kill %1   # the peer
```

`mkdir /tmp/esp-emu.lock` is the machine-local discipline for running one
esp-emu instance at a time (the emulator uses host-wide resources).

Expected console output (interleaved with the IDF boot log; `[c]` lines are
the C side's heap marks):

```
[go] bt: start
[go] bt backend = bluedroid
[go] bt init ok
[c] bt heap init free=... min=... | internal=... | psram=...
[go] bt name ok
[go] bt discoverable ok
[go] bt spp listen scn=1
[go] bt accept wait
[go] bt accept ok addr=aa:bb:cc:dd:ee:01
[go] bt echo rx=12800 tx=12800 pat=ok
[go] bt peer closed eof
[go] bt PASS
[go] bt deinit ok
[go] bt done
```

## Acceptance

Two rounds of step 3+4 (a fresh emulator boot each time). Each round passes
when every `[go] bt` line, with the floater rule applied, matches
`golden/emu-bt-probe.txt`:

```
grep -E '^\[go\] bt' <emu log> | tr -d '\r' \
    | diff - <(grep -v '^#' golden/emu-bt-probe.txt)
```

The golden has no floaters: the peer's BR/EDR address is pinned by the peer
script and the firmware renders it in Bluedroid's storage order (most
significant octet first), so even the `addr=` value is exact - which is what
pins the render direction.

The peer side independently reports `BUMBLE_SPP_PASS` when its DLC saw the
whole payload and the disconnect landed cleanly; both sides passing is the
round's verdict.

## Troubleshooting

- **glue `#error` at CMake configure** - the synced `main/bt_glue.c` refused
  the sdkconfig: the classic SPP lines of `sdkconfig.defaults` did not make it
  into `sdkconfig`. Delete `sdkconfig` and rebuild so the defaults re-apply.
- **`BT_ERR_GLUE_ABI` from Init** - `main/bt_glue.c` and `pkg/esp/bt` drifted
  apart: re-run `make-artifact.sh` so the glue template is re-synced.
- **`[go] bt accept err=... ESP_ERR_TIMEOUT`** - the peer never paged: check
  that the bumble process is up (its `[*] virtual controller on :9544` line
  precedes the emulator's start) and that no other process holds port 9544.
- **abort in `osi_thread_post` or a hang right after `[go] bt PASS`** - the
  stack was torn down while an ACL link was still up: the disconnect
  completion then lands on host threads Bluedroid already freed (abort), or a
  disconnect that can never complete stalls `esp_bluedroid_disable` (hang).
  Deinit serializes profile teardown and waits for the ACL count to reach
  zero, but a peer that holds the link forever (an exited virtual controller
  without the ACL disconnect - see the peer's DELTA note) can still stall it.
- **boot loop under esp-emu** - a Go fatal path parks in `__wrap__exit`
  instead of restarting; read the `[c] _exit(...)` diagnostics at the end of
  the console log.
