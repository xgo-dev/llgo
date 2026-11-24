# llgo × ESP-IDF BLE pipeline (ESP32-S31): NimBLE central

A self-contained pipeline that drives ESP-IDF's NimBLE host from Go code
compiled by llgo: bring-up and sync, a scan that finds a peer, a connect, a
disconnect, and the stack shutdown - with the acceptance run automated under
esp-emu against a virtual BLE peripheral hosted by Google Bumble. The Go
bindings live in the llgo repository (`pkg/esp/ble`, consumed through the module
require + replace in `go.mod`); this directory owns the acceptance program and
the IDF project.

This is the central half of GAP only. GATT client discovery/read/write/
notifications, the peripheral role and pairing are later stages; see
`pkg/esp/README.md` for the package contract and the stages.

## Layout

```
esp-idf-ble/
├── go.mod                  Go module (module llgoble; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: Init → cancelled scan → scan for the
│                           peer → connect → disconnect → Deinit
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── golden/emu-ble-probe.txt  Golden for the [go] ble lines (+ the floater rule)
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt
    ├── sdkconfig.defaults  BLE-only: NimBLE host from PSRAM, central+observer
    │                       roles, one connection, GATT server off, 8 KiB main
    │                       task, 240 MHz, PSRAM, 3.9 MiB app partition
    ├── partitions.csv
    ├── host/
    │   ├── make-artifact.sh  llgo build → IDF-linkable relocatable object; also
    │   │                     syncs pkg/esp/ble/glue/ble_glue_nimble.c into main/
    │   ├── llgo_syms.S       GC window + bootstrap stack providers (window size
    │   │                     from LLGO_GC_WINDOW_KB, default 64 KiB)
    │   └── bumble_peer.py    Host-side BLE peripheral: a Bumble virtual
    │                         controller for the emulator's HCI stream plus an
    │                         advertising peripheral on the same LocalLink
    └── main/
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── app_main.c      app_main → llgo entry; _write/_exit shims; Go heap
        │                   PSRAM budget (llgo_gc_ext_heap_cap)
        ├── demo_glue.c     Demo-local heap marks (both IDF pools)
        ├── ble_glue.c      NimBLE backend glue, synced from
        │                   github.com/xgo-dev/llgo/pkg/esp/ble/glue (gitignored)
        └── llgo_dram.ld    _globals_start/_globals_end for tinygogc
```

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-idf` below), Docker image
`espressif/idf:latest`, esp-emu v1.2.0 (the first release with an HCI backend
for the S31 controller) and Python Bumble (`pip install bumble`; a venv is
enough). No Bluetooth hardware is needed.

```
# 1. Go → relocatable object (prints the remaining undefined symbols; the
#    llgo_ble_* glue set must be in that list - the script echoes it). It also
#    syncs the pkg/esp/ble glue template into main/ - run it before every
#    idf.py build.
cd idfproj/host && ./make-artifact.sh /tmp/llgo-idf

# 2. IDF firmware (merge-bin takes -o relative to build/: the image lands in
#    build/merged-binary.bin, the ELF the emulator needs in build/llgo_idf_s31.elf)
cd .. && docker run --rm -v "$PWD":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Host-side BLE peripheral. It serves the emulator's HCI stream on TCP 9544
#    and advertises as a connectable peripheral named llgo-peer. Start it
#    first: the emulator connects out to it at boot.
<venv>/bin/python host/bumble_peer.py --port 9544 --timeout 90 &

# 4. Acceptance run. --elf is mandatory: the emulator intercepts the
#    firmware's BTDM HCI driver through the ELF's symbols, so the ELF must be
#    the unstripped build that produced the merged image.
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --elf idfproj/build/llgo_idf_s31.elf \
    --ble-hci tcp:127.0.0.1:9544 \
    --console usb-jtag --timeout 60s --psram-size 16M \
    --exit-on "[go] ble done"
rmdir /tmp/esp-emu.lock
kill %1   # the peer
```

`mkdir /tmp/esp-emu.lock` is the machine-local discipline for running one
esp-emu instance at a time (the emulator uses host-wide resources).

Expected console output (interleaved with the IDF boot log; `[c]` lines are the
C side's heap and adapter report):

```
[go] ble: start
[go] ble init ok
[c] ble glue: free heap 17082967, own address 00:11:22:33:44:55
[c] ble heap init free=17082967 min=17050936 | internal=311723 | psram=16771244
[go] ble scan cancel ok op=1
[go] ble scan start op=2
[c] ble heap scan start free=17082967 min=17050936 | internal=311723 | psram=16771244
[go] ble scan found name=llgo-peer addr=c0:ff:ee:00:00:01 addrtype=1 rssi=-50 advlen=11
[go] ble scan stop ok
[go] ble connect start op=4
[go] ble connect ok conn=65537
[c] ble heap connected free=17082967 min=17050936 | internal=311723 | psram=16771244
[go] ble disconnect done reason=531
[c] ble heap disconnected free=17082967 min=17050936 | internal=311723 | psram=16771244
[go] ble PASS
[go] ble deinit ok
[c] ble heap deinit free=17109371 min=17050848 | internal=334735 | psram=16774636
[go] ble done
```

The peer logs its own side, which is the cross-evidence that the connection was
real on both ends:

```
[peer] virtual controller listening on tcp:9544, peer address C0:FF:EE:00:00:01 name 'llgo-peer'
[peer] advertising as C0:FF:EE:00:00:01 (static random), connectable, name 'llgo-peer'
[peer] central connected handle=1 peer=00:11:22:33:44:55/P
[peer] central disconnected reason=0x13
```

What the probe proves, phase by phase:

| Phase | Line | What it establishes |
|---|---|---|
| bring-up | `[go] ble init ok` | `nimble_port_init` (controller init + BLE enable + host init) returned ESP_OK and the host posted its sync event through the mailbox |
| cancel | `[go] ble scan cancel ok op=1` | a discovery can be abandoned; the command queue keeps the following `StartScan` strictly after it |
| scan | `[go] ble scan found ...` | the mailbox delivered a real advertising report: address, address type, RSSI and the 11-byte payload (9-byte name + AD header), payload copied out of the HCI mbuf |
| connect | `[go] ble connect ok conn=65537` | `ble_gap_connect` completed; the ConnID is the packed `(generation << 16) | handle` form |
| disconnect | `[go] ble disconnect done reason=531` | `ble_gap_terminate` on that ConnID took the connection down; `531` = `0x213`, the peer-initiated reason tagged with NimBLE's HCI base |
| shutdown | `[go] ble deinit ok` | `nimble_port_stop` + `nimble_port_deinit` released the controller and host, and the internal heap returned to within ~1.5 KiB of its pre-BLE figure |

## Memory

Measured on this firmware (BLE-only, no WiFi, 64 KiB static GC window, 16 MiB
PSRAM, `CONFIG_BT_NIMBLE_MEM_ALLOC_MODE_EXTERNAL=y`):

| Mark | IDF internal free | IDF PSRAM free |
|---|---:|---:|
| after `ble.Init` (controller + host + host task) | 311,723 | 16,771,244 |
| while connected | 311,723 | 16,771,244 |
| after `ble.Deinit` (stack released) | 334,735 | 16,774,636 |

So the live BLE stack costs roughly **23 KiB of internal SRAM** and ~3.4 KiB of
PSRAM on this target, and the shutdown returns it. These are esp-emu figures for
a BLE-only image; they are not a WiFi+BLE budget (that combination needs the
heap levers in the S31 Bluetooth research report, and no llgo firmware has been
measured in that shape yet).

## What the emulator covers, and what it does not

esp-emu 1.2.0 runs the real IDF NimBLE host and the real controller
initialization, and it takes over the firmware's BTDM HCI transport through the
ELF (`--ble-hci`): the HCI command/event/ACL stream goes to a Bumble virtual
controller. Everything above HCI is therefore real firmware code:

* covered: controller init/enable/disable, host init, host task and event queue
  ordering, the GAP procedures (discovery, connect, terminate), the mailbox and
  fan-out, HCI command/event flow, the ConnID/generation bookkeeping, and heap
  accounting for the stack.
* **not** covered (pending real-hardware verification, no claim made here): the
  radio itself (RF calibration, real RSSI, channel map, coexistence), pairing and
  bonding (SM, NVS-backed bond store), MTU exchange and data-length extension,
  more than one connection (`CONFIG_BT_NIMBLE_MAX_CONNECTIONS=1` in this demo),
  notifications/indications (no GATT stage yet), flash/cache operations
  concurrent with BLE activity, and the controller's event-pool behavior under
  real interrupt load.

Two emulator gaps are worth naming, both measured while building this pipeline:

* a connect attempt that nobody answers **never fails** under esp-emu: with a
  3 s `duration_ms` it was still pending 15 s later, because the virtual
  controller does not model the controller-side connect-duration timeout. The
  probe therefore only exercises the answered path; the expiring-attempt path
  (which must report a non-zero `EvConnect` status) is a hardware item.
* the peer script has to drop Bumble's extended-advertising feature bit, so
  the emulator cannot exercise extended advertising at all (see the note
  below).

Mailbox overflow has likewise only been reasoned about, not observed: the
emulator never produces advertising traffic dense enough to overrun the rings.

### Emulator and peer notes

* **The S31 HCI path is interceptable.** The emulator recognizes the firmware's
  BTDM transport (`hci_driver_host_cmd_tx`, `hci_driver_host_acl_tx`,
  `hci_driver_host_callback_register`, `ble_stack_enable/disable`, ...) and logs
  `BLE: 12 functions intercepted (S31)`. NimBLE's `nimble_port_init` therefore
  works unchanged under emulation.
* **The peer must advertise legacy-only.** Bumble's `Device.start_advertising`
  prefers the extended advertising commands whenever the controller reports
  `LE_EXTENDED_ADVERTISING`, and Bumble's scanning controller then reports every
  packet as `HCI_LE_EXTENDED_ADVERTISING_REPORT` — including legacy `ADV_IND`.
  A firmware whose NimBLE host is built without `CONFIG_BT_NIMBLE_EXT_ADV` scans
  with the legacy procedure (`ble_gap_disc`) and only parses the legacy
  `LE Advertising Report` event, so it sees nothing. `bumble_peer.py` drops that
  feature bit from both of its controllers, which makes the peer send real
  `ADV_IND` and the scanner report through the legacy event.
* **NimBLE over the BTDM OSAL.** On this chip `struct ble_npl_event` wraps
  `struct btdm_osal_event` (`nimble_npl_os.h`), so every initialized event holds
  one element of the controller's `s_btdm_osal_ev_pool`. A command-queue event
  that is never `ble_npl_event_deinit`ed makes `esp_bt_controller_disable`
  assert in `btdm_osal_elem_mempool_deinit` — which is what the glue's `Deinit`
  returns before it stops the controller, and why the probe exercises the
  shutdown instead of assuming it.

## Golden

`golden/emu-ble-probe.txt` holds every `[go] ble` line of the run, with the
floater rule for the values that come from the peer script or the controller
(address, address type, RSSI, ConnID, disconnection reason) documented in the
file's header. The `[c]` lines are excluded: heap byte counts drift run to run
and the "own address" line reports the address the host controller presents.

The two runs recorded for this demo were byte-identical on the `[go] ble` lines
after normalizing the floaters.

## Notes

* Only this demo's `sdkconfig.defaults` is BLE-shaped; `_demo/esp-idf-wifi` keeps
  its WiFi configuration untouched so its acceptance chain stays valid.
* The main component is `WHOLE_ARCHIVE` for the reason every demo here is: the
  glue is referenced only from `llgo_out.o`, which the IDF link processes last
  and without `--start-group` (see `pkg/esp/README.md`).
* Host builds compile entirely against the stub side (`go build`, `go vet`,
  gopls): `pkg/esp/ble` through its `ble_stub.go`, `goapp/main.go` through
  `goapp/hosted.go`.
* A firmware built without the NimBLE host still links a program that imports
  `pkg/esp/ble`: the glue's other branch defines every symbol and returns
  `ESP_ERR_NOT_SUPPORTED`, which the package surfaces as `ble.ErrUnsupported` —
  never a forged success.
