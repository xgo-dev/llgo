# llgo × ESP-IDF BLE pipeline (ESP32-S31): NimBLE central + GATT client

A self-contained pipeline that drives ESP-IDF's NimBLE host from Go code
compiled by llgo: bring-up and sync, a scan that finds a peer, a connect, the
ATT MTU exchange, GATT discovery (16-bit and 128-bit services), read, write with
read-back, a notification subscription with two deliveries, unsubscription, the
disconnect and the stack shutdown - with the acceptance run automated under
esp-emu against a virtual BLE peripheral hosted by Google Bumble. The Go
bindings live in the llgo repository (`pkg/esp/ble`, consumed through the module
require + replace in `go.mod`); this directory owns the acceptance program and
the IDF project.

This is the GAP central role plus the GATT client. The peripheral role, pairing/
bonding, descriptor discovery and long read/write are later stages; see
`pkg/esp/README.md` for the package contract and the stages.

## Layout

```
esp-idf-ble/
├── go.mod                  Go module (module llgoble; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: Init → cancelled scan → scan for the
│                           peer → connect → MTU → discover services and
│                           characteristics → read → write + read-back →
│                           subscribe → 2 notifications → unsubscribe →
│                           write command + read-back → 128-bit service
│                           read/write → disconnect → Deinit
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
    │                         advertising peripheral with a GATT server on the
    │                         same LocalLink (16-bit echo + notify service,
    │                         128-bit echo service)
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

# 3. Host-side BLE peripheral. It serves the emulator's HCI stream on TCP 9544,
#    advertises as a connectable peripheral named llgo-peer, and hosts the GATT
#    server the probe discovers (0xFFF0 echo service with a read/write echo and
#    a notify characteristic, and a 128-bit echo service). Start it first: the
#    emulator connects out to it at boot. Every echo write is pushed back as a
#    notification carrying the same bytes, which is why the probe's log shows
#    its own payloads returning through the subscription.
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
[go] ble backend = nimble
[go] ble init ok
[c] ble glue: free heap 17080439, own address 00:11:22:33:44:55
[c] ble heap init free=17080439 min=17048408 | internal=309195 | psram=16771244
[go] ble scan cancel ok op=1
[go] ble scan start op=2
[c] ble heap scan start free=17080439 min=17048408 | internal=309195 | psram=16771244
[go] ble scan found name=llgo-peer addr=c0:ff:ee:00:00:01 addrtype=1 rssi=-50 advlen=11
[go] ble scan stop ok
[go] ble connect start op=4
[go] ble connect ok conn=65537
[c] ble heap connected free=17080439 min=17048276 | internal=309195 | psram=16771244
[go] ble mtu=256
[go] ble services count=4
[go] ble echo svc uuid=fff0 handle=14 end=19
[go] ble wide svc uuid=6c6c676f-1111-2222-3344-556677889900 handle=20 end=22
[go] ble echo chrs=2
[go] ble echo chr uuid=fff1 handle=16 props=14 read=1 write=1 notify=0
[go] ble echo chr uuid=fff2 handle=18 props=16 read=0 write=0 notify=1
[c] ble heap discovered free=17080419 min=17048124 | internal=309195 | psram=16771224
[go] ble read len=0
[go] ble write mode=1 len=7
[go] ble read back len=7 data=ping-00
[go] ble subscribe ok handle=18
[go] ble write mode=1 len=7
[go] ble notify n=1 handle=18 ind=0 len=7 data=ping-01
[go] ble write mode=1 len=7
[go] ble notify n=2 handle=18 ind=0 len=7 data=ping-02
[go] ble unsubscribe ok
[c] ble heap notified free=17080419 min=17048124 | internal=309195 | psram=16771224
[go] ble write nrsp mode=2 len=5
[go] ble read back nrsp len=5 data=norsp
[go] ble wide chrs=1
[go] ble wide chr uuid=6c6c676f-1111-2222-3344-556677889901 handle=22 props=14
[go] ble wide write mode=1 len=4
[go] ble wide read back len=4 data=u128
[go] ble disconnect done reason=531
[c] ble heap disconnected free=17080419 min=17048124 | internal=309195 | psram=16771224
[go] ble PASS
[go] ble deinit ok
[c] ble heap deinit free=17106843 min=17048124 | internal=332207 | psram=16774636
[go] ble done
```

The peer logs its own side, which is the cross-evidence that the connection was
real on both ends:

```
[peer] virtual controller listening on tcp:9544, peer address C0:FF:EE:00:00:01 name 'llgo-peer'
[peer] gatt server: echo FFF1 (handle 0x0010), notify FFF2 (handle 0x0012, cccd 0x0013), 128-bit 6c6c676f-1111-2222-3344-556677889901 (handle 0x0016)
[peer] advertising as C0:FF:EE:00:00:01 (static random), connectable, name 'llgo-peer'
[peer] central connected handle=1 peer=00:11:22:33:44:55/P mtu=23
[peer] PASS: a central connected
[peer] mtu update handle=1 mtu=256
[peer] echo write uuid=FFF1 len=7 value=b'ping-00'
[peer] subscription handle=0x0012 notify=1 indicate=0
[peer] echo write uuid=FFF1 len=7 value=b'ping-01'
[peer] echo write uuid=FFF1 len=7 value=b'ping-02'
[peer] subscription handle=0x0012 notify=0 indicate=0
[peer] echo write uuid=6c6c676f-1111-2222-3344-556677889901 len=4 value=b'u128'
[peer] central disconnected reason=0x13
```

The subscription lines are the peer's side of the CCCD writes: `notify=1` after
`SubscribeChar`, `notify=0` after `Unsubscribe`.

What the probe proves, phase by phase:

| Phase | Line | What it establishes |
|---|---|---|
| bring-up | `[go] ble init ok` | `nimble_port_init` (controller init + BLE enable + host init) returned ESP_OK and the host posted its sync event through the mailbox |
| cancel | `[go] ble scan cancel ok op=1` | a discovery can be abandoned; the command queue keeps the following `StartScan` strictly after it |
| scan | `[go] ble scan found ...` | the mailbox delivered a real advertising report: address, address type, RSSI and the 11-byte payload (9-byte name + AD header), payload copied out of the HCI mbuf |
| connect | `[go] ble connect ok conn=65537` | `ble_gap_connect` completed; the ConnID is the packed `(generation << 16) | handle` form |
| MTU | `[go] ble mtu=256` | the exchange the glue starts on connect finished (`EvMtu`), and `MTU(conn)` reads the same value back from the stack - the peer's own log shows `mtu update ... mtu=256` |
| discovery | `[go] ble services count=4` + the two lookup lines | `ble_gattc_disc_all_svcs` reported the peer's GAP and GATT services plus both scripted ones; the probe finds its 16-bit and 128-bit service by UUID, so both UUID widths parsed out of the wire |
| characteristics | `[go] ble echo chrs=2` + one `read=/write=/notify=` line per characteristic | `ble_gattc_disc_all_chrs` over the discovered handle range; `CanRead/CanWrite/CanNotify` applied to the property bits the peer declared |
| read | `[go] ble read len=0` | `ble_gattc_read` of the echo characteristic, still at the peer's initial (empty) value |
| write | `[go] ble write mode=1 len=7` | `Write` chose the acknowledged ATT Write Request from the properties (`PropWrite` set) and its response arrived through the mailbox |
| read back | `[go] ble read back len=7 data=ping-00` | the peer stored the probe's bytes and returned them |
| subscribe | `[go] ble subscribe ok handle=18` | the CCCD write (value handle + 1) completed; the peer logs `notify=1` |
| notifications | `[go] ble notify n=1 ... data=ping-01`, `n=2 ... data=ping-02` | two notifications reached the characteristic subscription through the mailbox, each carrying the bytes the probe wrote to the echo characteristic |
| unsubscribe | `[go] ble unsubscribe ok` | the CCCD write of 0x0000 completed; the peer logs `notify=0` |
| write command | `[go] ble write nrsp mode=2 len=5` + `read back nrsp ... data=norsp` | `WriteWithoutResponse` submitted an ATT Write Command (no acknowledgement, so the call reports the submission only) and the peer applied it: the read-back returns those bytes |
| 128-bit path | `[go] ble wide chr uuid=...889901` + write/read-back `u128` | the same read/write/read-back through a vendor 128-bit service and characteristic |
| disconnect | `[go] ble disconnect done reason=531` | `ble_gap_terminate` on that ConnID took the connection down; `531` = `0x213`, the peer-initiated reason tagged with NimBLE's HCI base |
| shutdown | `[go] ble deinit ok` | `nimble_port_stop` + `nimble_port_deinit` released the controller and host, and the internal heap returned to within ~1.5 KiB of its pre-BLE figure |

## Memory

Measured on this firmware (BLE-only, no WiFi, 64 KiB static GC window, 16 MiB
PSRAM, `CONFIG_BT_NIMBLE_MEM_ALLOC_MODE_EXTERNAL=y`):

| Mark | IDF internal free | IDF PSRAM free |
|---|---:|---:|
| after `ble.Init` (controller + host + host task) | 309,195 | 16,771,244 |
| while connected | 309,195 | 16,771,244 |
| after discovery (two service lists on the Go heap) | 309,195 | 16,771,224 |
| after the notification phase | 309,195 | 16,771,224 |
| after `ble.Deinit` (stack released) | 332,207 | 16,774,636 |

So the live BLE stack costs roughly **23 KiB of internal SRAM** and ~3.4 KiB of
PSRAM on this target, and the shutdown returns it. The GATT client adds no
measurable runtime heap: its discovery tables, notification ring, read slot and
write slot are static (about 2.5 KiB of `.bss`, which is why the init figure is
~2.5 KiB below the GAP-only stage's), and the only Go allocations are the two
short service/characteristic slices (20 bytes of PSRAM in this run). These are esp-emu figures for
a BLE-only image; they are not a WiFi+BLE budget (that combination needs the
heap levers in the S31 Bluetooth research report, and no llgo firmware has been
measured in that shape yet).

## What the emulator covers, and what it does not

esp-emu 1.2.0 runs the real IDF NimBLE host and the real controller
initialization, and it takes over the firmware's BTDM HCI transport through the
ELF (`--ble-hci`): the HCI command/event/ACL stream goes to a Bumble virtual
controller. Everything above HCI is therefore real firmware code:

* covered: controller init/enable/disable, host init, host task and event queue
  ordering, the GAP procedures (discovery, connect, terminate), the ATT MTU
  exchange, the GATT client procedures (service and characteristic discovery,
  read, write, CCCD write) and the notifications the peer pushes, the mailbox
  and fan-out, HCI command/event/L2CAP-ACL flow, the ConnID/generation
  bookkeeping, and heap accounting for the stack.
* **not** covered (pending real-hardware verification, no claim made here): the
  radio itself (RF calibration, real RSSI, channel map, coexistence), pairing and
  bonding (SM, NVS-backed bond store), data-length extension, more than one
  connection (`CONFIG_BT_NIMBLE_MAX_CONNECTIONS=1` in this demo),
  indications (the peer only notifies; the indication path shares the
  notification ring and the host confirms for us), notifications larger than
  `ble.MaxData`, flash/cache operations concurrent with BLE activity, and the
  controller's event-pool behavior under real interrupt load.

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
emulator never produces advertising traffic dense enough, or notifications large
enough, to overrun the rings or the payload bound.

## Payload bound and the goroutine stack

`ble.MaxData` (128 bytes) bounds the payload one mailbox record carries. That is
**not** the ATT MTU: the glue negotiates 256, so a peer may legally notify 253
bytes. The bound is the goroutine stack. Every record travels through a channel
and through the non-blocking receive `select` of `Subscription.Next`, and those
frames hold several copies of it; `targets/esp32s31-idf.json` gives goroutines
2 KiB fiber stacks, and a record of 316 bytes (MaxData 253) overflows the stack
during `Init` - measured: the firmware aborted with a load-access fault before
its first `[go] ble init ok`, while the 188-byte record MaxData 128 produces runs
the whole acceptance. What does not fit is never truncated: an oversized
notification is dropped whole and counted in `EvOverflow`, and an oversized read
reports `BLE_HS_EMSGSIZE`. A firmware built with a larger
`-goroutine-stack-size` can raise `MaxData` and the glue's `BW_DATA_MAX`
together.

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
floater rule for the values that come from the peer script, the controller or
the peer's attribute table (address, address type, RSSI, ConnID, disconnection
reason, negotiated MTU, service/characteristic UUIDs and handles, counts, and
the echo characteristic's initial value) documented in the file's header. The `[c]` lines are excluded: heap byte counts drift run to run
and the "own address" line reports the address the host controller presents.

The two runs recorded for this demo were byte-identical on the `[go] ble` lines
after normalizing the floaters.

## Notes

* Two boards sharing one PC's USB power rail can brown each other out: the
current surge when a connection's radio bursts start pulls the shared 5V rail
low enough to reset both boards at once (one board's USB-UART bridge drops
off the bus, the other takes a full POWERON reset). Device-verified in both
directions: with one board on an independent supply, the same central
firmware that "crashed at read" every time runs the whole acceptance chain
to `[go] ble PASS`. The package has no role in it; keep two radios on
separate power when doing connection-heavy tests on a bench.
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
