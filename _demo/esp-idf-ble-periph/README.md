# llgo × ESP-IDF BLE pipeline (ESP32-S31): Bluedroid peripheral + GATT server

A self-contained pipeline that drives ESP-IDF's Bluedroid host from Go code
compiled by llgo - the **peripheral role** of `pkg/esp/ble`: the local GATT
server (CommitServices), the initial characteristic values (WriteCharValue),
advertising (StartAdvertising), the central's connection and writes as
mailbox events (EvPeriphConnect, EvCharWrite, EvSubscribed), server-side
notifications (Notify), and the disconnection - with the acceptance run
automated under esp-emu against a virtual BLE central hosted by Google
Bumble. The Go bindings live in the llgo repository (`pkg/esp/ble`, consumed
through the module require + replace in `go.mod`); this directory owns the
acceptance program and the IDF project.

This is the flipped counterpart of `_demo/esp-idf-ble`: there the firmware is
the NimBLE central and the host script advertises; here the firmware is the
Bluedroid peripheral and the host script scans. The sdkconfig choices are
mutually exclusive (one BT host per firmware), which is why the two roles
live in two demo directories sharing the `_demo/common` pipeline, and why the
central demo's golden stays untouched by this one.

## Layout

```
esp-idf-ble-periph/
├── go.mod                  Go module (module llgoblep; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: Init → SetSecurityParams → registry
│                           (2 services, 3 characteristics, one 128-bit) →
│                           CommitServices → LocalHandle → WriteCharValue →
│                           StartAdvertising → EvPeriphConnect → EvSubscribed →
│                           echo writes back as notifications → EvPeriphDisconnect
│                           → Deinit
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── golden/emu-ble-periph.txt  Golden for the [go] blep lines (+ the floater rule)
├── golden/bumble-central-run.log  The peer's own log of the same run (the
│                           cross-evidence that the link was real on both ends)
└── idfproj/                Minimal ESP-IDF project
    ├── sdkconfig.defaults  BLE-only: Bluedroid host from PSRAM, GATT server +
    │                       client, SMP, legacy advertising APIs, 8 KiB main
    │                       task, 240 MHz, PSRAM, 3.9 MiB app partition
    ├── host/
    │   ├── make-artifact.sh  symlink to the shared pipeline (_demo/common);
    │   │                     also syncs pkg/esp/ble/glue/ble_glue_bluedroid.c
    │   │                     into main/
    │   ├── artifact.conf    what this firmware links (the Bluedroid glue)
    │   └── bumble_central.py  Host-side BLE central: a Bumble virtual
    │                         controller for the emulator's HCI stream plus a
    │                         scanning central on the same LocalLink
    └── main/
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── demo_glue.c     Demo-local heap marks (both IDF pools)
        └── ble_glue.c      Bluedroid backend glue, synced from
                            github.com/xgo-dev/llgo/pkg/esp/ble/glue (gitignored)
```

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-idf` below), Docker image
`espressif/idf:latest`, esp-emu v1.2.0 (the first release with an HCI backend
for the S31 controller) and Python Bumble (`pip install bumble`; a venv is
enough). No Bluetooth hardware is needed.

```
# 1. Go → relocatable object (the script also syncs the pkg/esp/ble glue
#    template into main/ - run it before every idf.py build).
cd idfproj/host && ./make-artifact.sh /tmp/llgo-idf

# 2. IDF firmware (merge-bin takes -o relative to build/: the image lands in
#    build/merged-binary.bin, the ELF the emulator needs in build/llgo_idf_s31.elf)
cd .. && docker run --rm -v "$PWD":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Host-side BLE central. It serves the emulator's HCI stream on TCP 9545
#    (the central demo owns 9544, so parallel runs of the two demos do not
#    collide), scans for llgo-periph, connects, discovers, reads the initial
#    value, subscribes, writes twice (one acknowledged, one command),
#    read-backs, unsubscribes and disconnects. Start it first: the emulator
#    connects out to it at boot.
<venv>/bin/python host/bumble_central.py --port 9545 --timeout 100 &

# 4. Acceptance run. --elf is mandatory: the emulator intercepts the
#    firmware's BTDM HCI driver through the ELF's symbols, so the ELF must be
#    the unstripped build that produced the merged image.
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --ble-hci tcp:127.0.0.1:9545 \
    --console usb-jtag --timeout 90s --psram-size 16M \
    --exit-on "[go] blep done"
rmdir /tmp/esp-emu.lock
kill %1   # the peer
```

`mkdir /tmp/esp-emu.lock` is the machine-local discipline for running one
esp-emu instance at a time (the emulator uses host-wide resources).

Expected console output (interleaved with the IDF boot log; `[c]` lines are
the C side's heap and adapter report):

```
[go] blep: start
[go] blep backend = bluedroid
[go] blep init ok
[c] ble glue: free heap 17040051, own address 00:11:22:33:44:55
[c] ble heap init free=17040051 min=17005980 | internal=275815 | psram=16764236
[go] blep security ok
[go] blep registry svcs=2 chrs=3
[go] blep commit ok echo=42 notify=44 wide=48
[c] ble heap committed free=17038979 min=17005592 | internal=275287 | psram=16763692
[go] blep setvalue ok len=5
[go] blep adv start op=2
[go] blep adv ok
[c] ble heap advertising free=17038619 min=17005516 | internal=275287 | psram=16763332
[go] blep peer connected conn=65536 addr=c0:ff:ee:00:00:02 addrtype=1
[c] ble heap connected free=17036251 min=17002728 | internal=274759 | psram=16761492
[go] blep subscribed handle=44 cccd=1
[go] blep write n=1 handle=42 len=7 data=ping-00
[go] blep notify n=1 handle=44 len=7
[go] blep write n=2 handle=42 len=7 data=ping-01
[go] blep notify n=2 handle=44 len=7
[go] blep unsubscribed cccd=0
[go] blep disconnected conn=65536 reason=remote user terminated the connection
[c] ble heap disconnected free=17035943 min=17002635 | internal=274759 | psram=16761450
[go] blep PASS
[go] blep deinit ok
[c] ble heap deinit free=17074106 min=17002635 | internal=294446 | psram=16764311
[go] blep done
```

The peer logs its own side, which is the cross-evidence that the connection
was real on both ends (also kept as
`golden/bumble-central-run.log`):

```
[peer] scan found name=llgo-periph addr=00:11:22:33:44:55/P addrtype=0 rssi=-50
[peer] connected handle=1 peer=00:11:22:33:44:55/P mtu=23
[peer] service uuid=UUID-16:FFF0 chars=[UUID-16:FFF1 UUID-16:FFF2]
[peer] service uuid=6C6C676F-2222-3333-4444-556677889900 chars=[6C6C676F-2222-...889901]
[peer] discovered echo uuid=FFF1 props=<Properties.READ|WRITE_WITHOUT_RESPONSE|WRITE: 14>
[peer] discovered notify uuid=FFF2 props=<Properties.NOTIFY: 16>
[peer] read echo len=5 value=b'hello'
[peer] subscribed notify
[peer] echo write uuid=FFF1 len=7 value=b'ping-00'
[peer] notify n=1 len=7 value=b'ping-00'
[peer] echo write uuid=FFF1 len=7 value=b'ping-01'
[peer] notify n=2 len=7 value=b'ping-01'
[peer] read back len=7 value=b'ping-01'
[peer] wide write len=4 value=b'u128'
[peer] wide read back len=4 value=b'u128'
[peer] unsubscribed notify
[peer] disconnected reason=0x13
[peer] PASS: scan, connect, discover, read, subscribe, write, echo, unsubscribe, disconnect
```

The echo lines are the point: the peer wrote `ping-00`, the firmware reported
it as `EvCharWrite` and pushed it back through `Notify`, and the peer's own
subscription received exactly those bytes - write, value cache, CCCD, notify
and the event mailbox all worked in one observable chain. The `read echo
value=b'hello'` line is `WriteCharValue` proving itself: the peer's first
read is answered from the glue's static value cache, set by Go before the
advertisement even started.

What the probe proves, phase by phase:

| Phase | Line | What it establishes |
|---|---|---|
| bring-up | `[go] blep init ok` | Bluedroid enable + both interface registrations (GATT client and server) completed; `Backend()` names the stack |
| security | `[go] blep security ok` | `SetSecurityParams` accepted the pairing posture (bond + just-works) |
| registry | `[go] blep registry svcs=2 chrs=3` | `AddService`/`AddCharacteristic` built the registry: one 16-bit service (echo + notify), one 128-bit service (wide echo) |
| commit | `[go] blep commit ok echo=42 notify=44 wide=48` | `CommitServices` ran the stack's asynchronous creation to completion and pulled back the assigned value handles; `LocalHandle` returns them |
| initial value | `[go] blep setvalue ok len=5` + peer's `read echo value=b'hello'` | `WriteCharValue` filled the glue's value cache, and the peer's first read was answered from it - no Go involvement in the read path |
| advertise | `[go] blep adv ok` | the raw-payload advertisement (flags + complete local name) went through the stack's config chain and started; the peer found it by name |
| connect | `[go] blep peer connected conn=65536` | the central's connection arrived as `EvPeriphConnect` with the packed ConnID and the peer address |
| subscribe | `[go] blep subscribed handle=44 cccd=1` | the peer's CCCD write arrived as `EvSubscribed`, carrying the value handle and the raw CCCD value |
| echo | `[go] blep write n=1 data=ping-00` + `notify n=1`, then `ping-01` | each write (acknowledged request, then write command) landed in the value cache and the mailbox, and the probe's `Notify` pushed it back; the peer received both echoes byte-for-byte |
| unsubscribe | `[go] blep unsubscribed cccd=0` | the CCCD write of 0x0000, same event, value 0 |
| disconnect | `[go] blep disconnected reason=remote user terminated the connection` | the peer's disconnect arrived as `EvPeriphDisconnect` with the HCI reason, rendered by `DisconnectReasonName` |
| shutdown | `[go] blep deinit ok` | the stack released; internal heap returned to within ~19 KiB of its pre-BLE figure |

## Memory

Measured on this firmware (BLE-only Bluedroid, no WiFi, 64 KiB static GC
window, 16 MiB PSRAM, `CONFIG_BT_ALLOCATION_FROM_SPIRAM_FIRST=y`):

| Mark | IDF internal free | IDF PSRAM free |
|---|---:|---:|
| after `ble.Init` (controller + host + both interfaces) | 275,815 | 16,764,236 |
| after `CommitServices` | 275,287 | 16,763,692 |
| while advertising | 275,287 | 16,763,332 |
| while connected + echo | 274,759 | 16,761,492 |
| after `ble.Deinit` (stack released) | 294,446 | 16,764,311 |

The live Bluedroid stack costs roughly **31 KiB of internal SRAM** on this
target and returns almost all of it on shutdown (the residue is the pool
watermark, not a leak). The package's own peripheral state is static
`.bss`: the value caches, the declaration tables, the fourth mailbox ring and
the advertising buffers - about 3 KiB, which is why the init figure sits
below the NimBLE central firmware's (whose host allocates differently). These
are esp-emu figures for a BLE-only image; a WiFi+BLE provisioning build needs
the heap levers called out in the S31 Bluetooth research report.

## What the emulator covers, and what it does not

esp-emu 1.2.0 runs the real IDF Bluedroid host and intercepts the real
controller's HCI transport, so the whole stack above HCI is the production
code - the GATT server state machine, the SMP parameter plumbing, the
advertising configuration chain. What the emulator does not cover is the
radio: real timing (advertising intervals are the virtual controller's),
real pairing on air (the SMP handshake with an actual phone), and the
bond store on a real NVS partition. Those are the hardware-acceptance
items (`golden/` pins the emulated run; the report's 真机待验清单 tracks the
rest).

## NimBLE note

The `llgo_ble_*` ABI this demo links is implemented by both glue backends;
the NimBLE sibling (`pkg/esp/ble/glue/ble_glue_nimble.c`) implements the same
peripheral surface on its own stack APIs (a static `ble_gatt_svc_def` tree,
the GAP advertising calls, the access callback answering from the same
per-characteristic value caches). This demo pins the **Bluedroid** path in
emulation; a NimBLE peripheral firmware build (flip
`host/artifact.conf` to `ble_glue_nimble.c` and the sdkconfig's BT host
choice, plus `CONFIG_BT_NIMBLE_DYNAMIC_SERVICE=y` for post-sync service
registration) is the follow-up acceptance run tracked in the report.
