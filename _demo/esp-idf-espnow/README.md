# ESP-NOW on ESP32-S31 (llgo + `pkg/esp/espnow`)

A firmware that drives Espressif's ESP-NOW protocol from Go code compiled by
llgo: WiFi bring-up in the documented order, then the whole `pkg/esp/espnow`
surface - peer-list lifecycle, broadcast sends with asynchronous
confirmations, the encryption entry points, and a full Deinit/re-Init cycle -
with the run automated under `esp-emu`.

The demo has two jobs: it is the **acceptance probe** for the package, and it
is the runnable example for it. The radio's lifecycle belongs to
`pkg/esp/wifi`; this demo shows the ordering contract in action
(`wifi.Setup() -> SetModeSTA() -> Start()`, *then* `espnow.Init()`).

## API shape

```go
// Bring-up (after the radio is started; a connection is NOT required).
func Init() error                       // glue ABI check + esp_now_init + cb registration; starts the pump
func Deinit() error
func SetChannel(ch int) error           // esp_wifi_set_channel, secondary NONE; 1..14

// Peer list (the driver's, 20 entries max).
type PeerConfig struct {
    MAC     [6]uint8
    Channel uint8   // 0 = current channel
    If      IfIdx   // IfSTA / IfAP
    Encrypt bool    // needs LMK; consumes a CONFIG_ESP_WIFI_ESPNOW_MAX_ENCRYPT_NUM slot
    LMK     [16]uint8
}
func AddPeer(cfg PeerConfig) error      // broadcast peer: PeerConfig{MAC: BroadcastMAC}
func DeletePeer(mac [6]uint8) error
func PeerIsKnown(mac [6]uint8) bool
func PeerCount() (PeerNum, error)       // {Total, Encrypted}

// Send: synchronous driver verdict + asynchronous confirmation.
func Send(peer [6]uint8, data []byte) (*SendResult, error) // 1..250 bytes
func Broadcast(data []byte) (*SendResult, error)           // Send(BroadcastMAC, data)
func (r *SendResult) Wait(timeoutMs int32) (SendStatus, error) // SendSuccess/SendFail or ErrTimeout

// Receive: the pump delivers into the handler; data is a fresh copy.
func SetRecvHandler(fn func(from [6]uint8, data []byte))

// Encryption.
func SetPMK(key [16]uint8) error        // the LMKs' root, set before encrypted traffic
func SetLMK(peer [6]uint8, key [16]uint8) error // rotate one peer's key

// Identity and diagnostics.
func Version() (uint32, error)          // 1 = v1.0 (250 B frames), 2 = v2.0
func OwnMAC(ifIdx IfIdx) [6]uint8
func MACString(mac [6]uint8) string
func Dropped() uint32                   // ring-overflow loss counter
```

Events (the package's own table, registered in `pkg/esp/README.md`):

| kind | value | record | goes to |
| --- | --- | --- | --- |
| `EvRecv` | 1 | source MAC, RSSI, payload | `SetRecvHandler` |
| `EvSendDone` | 2 | destination MAC, send status | the pending `SendResult` (oldest first per MAC) |
| `EvOverflow` | -1 | ring dropped records | the `Dropped()` counter |

## Build and run

```sh
# one-time: build the llgo compiler from this checkout
LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config \
    go build -o /tmp/llgo-espnow-compiler ./cmd/llgo   # from the llgo repo root

cd idfproj
./host/make-artifact.sh /tmp/llgo-espnow-compiler   # llgo -> relocatable + glue sync
idf.py build
idf.py merge-bin -o merged-binary.bin
./host/run-acceptance.sh                            # esp-emu run + golden check
```

## Emulator fidelity (measured, esp-emu 1.2.0 + IDF 6.1)

A pure-C probe (no llgo involved) established what the emulator does with
esp_now before the binding was validated through it:

- **Modeled**: `esp_now_init` / `deinit` / re-init (ESP_OK), callback
  registration, peer management - `add_peer` (OK, and the duplicate's
  `ESP_ERR_ESPNOW_EXIST`), `is_peer_exist`, `get_peer_num`,
  `del_peer` (OK, and the second one's `ESP_ERR_ESPNOW_NOT_FOUND`), `set_pmk`
  - and `esp_now_send`: broadcast sends queue (ESP_OK) and the **send
  callback fires** with `ESP_NOW_SEND_SUCCESS`, repeatedly, including after
  a deinit/re-init cycle. `esp_now_send` before init / after deinit reports
  `ESP_ERR_ESPNOW_NOT_INIT`; a send to an unregistered MAC reports
  `ESP_ERR_ESPNOW_NOT_FOUND`.
- **Not modeled (per the demo probe and the binding's honest SKIP lines)**:
  over-the-air frames. Two esp-emu instances broadcasting every 300 ms each
  see only their own sends confirmed and `recv = 0` on both sides - the
  emulator has no cross-instance radio and no virtual ESP-NOW peer.

Consequently the Go acceptance run exercises the whole **control plane**
through the binding (bring-up order, peer lifecycle, refusals, broadcast
confirmations through the pump, PMK/LMK, re-init) and registers the recv
handler, but never claims a frame was received: the receive path and the
radio-level semantics are hardware items. The Go probe's full log is in
`idfproj/host/logs/espnow-acceptance.log`, the C probe's in
`idfproj/host/logs/espnow-c-probe.log`.

## Hardware checklist (what esp-emu cannot answer)

- [ ] Two boards, unassociated, same channel: broadcast/unicast delivery,
  recv handler payload fidelity, RSSI sanity.
- [ ] Unicast send confirmation semantics: `ESP_NOW_SEND_SUCCESS` with an
  ACKing peer vs `ESP_NOW_SEND_FAIL` with a dead peer; the SendResult FIFO
  under real confirmation timing (sub-100 ms, i.e. inside one pump quantum).
- [ ] Latency budget: esp_now_send → callback delta on hardware (the
  protocol's headline number; the pump adds at most one quantum).
- [ ] Encrypted pair: PMK + LMK on both sides, frames received and
  decrypted, `ESP_ERR_ESPNOW_FULL` after `CONFIG_ESP_WIFI_ESPNOW_MAX_ENCRYPT_NUM`
  encrypted peers.
- [ ] Channel roaming: `SetChannel` on both sides mid-run, traffic follows.
- [ ] WiFi coexistence: espnow send burst while the station is associated
  and passing traffic (throughput/reorder effects, send-queue `NO_MEM`
  backpressure).
- [ ] Range/rate behavior at the PHY defaults; per-peer rate config remains
  unexposed (see the package docs).

## Notes

- The probe's PASS criteria are the driver's own documented codes under
  emulation; nothing is asserted about frames on the air.
- `idfproj/host/make-artifact.sh` is the shared pipeline
  (`_demo/common/make-artifact.sh`); this demo's `artifact.conf` selects
  `GLUES="wifi espnow"`.
- The recv ring holds 16 slots (~4.3 KiB .bss). A burst that outruns the
  pump is dropped newest-first and counted (`Dropped()`), the datagram
  discipline - not an error.
