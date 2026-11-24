# ESP-IDF soft AP on ESP32-S31 (llgo + `pkg/esp/wifi`)

A firmware that is a WiFi **access point**: it beacons an SSID, runs IDF's own
DHCP server, watches clients associate and lease addresses, serves them a TCP
session, deauthenticates them, and switches to APSTA mode - all through
`pkg/esp/wifi`, with the whole run automated under `esp-emu`.

The demo has two jobs: it is the **acceptance probe** for the AP half of the
package, and it is the runnable example for it. The station half (scan,
associate, DHCP client, auto-reconnect, TLS, WebSocket) lives in
`_demo/esp-idf-wifi` and is unchanged by this one.

## API shape

```go
// Configuration: one call describes the network.
type APConfig struct {
    SSID     string   // 1..32 bytes
    Password string   // "" for open, else 8..63 bytes
    Channel  uint8    // 1..14; 0 means 1 (never driver-chosen)
    Hidden   bool     // stop beaconing the SSID
    MaxSTA   int      // 1..MaxAPStations; 0 means DefaultMaxSTA (4)
    AuthMode AuthMode // AuthOpen / AuthWPA2PSK / AuthWPA3PSK / AuthWPA2WPA3PSK
}

func SetAP(cfg APConfig) error         // configure the AP iface (+ mode, see below)
func SetModeSTA() error                // esp_wifi_set_mode: STA     (explicit from now on)
func SetModeAP() error                 // esp_wifi_set_mode: AP
func SetModeAPSTA() error              // esp_wifi_set_mode: APSTA
func Start() error                     // esp_wifi_start: the AP comes up, EvAPStart follows
func Stop() error                      // esp_wifi_stop: AP_STOP follows, AP netif survives
func GetAPIP() (uint32, error)         // 192.168.4.1 (network order; IPString formats it)
func GetAPConfig() (APConfig, error)   // read back from the driver (password never returned)
func APStations() ([]Station, error)   // current clients: MAC, RSSI, leased IP
func DeauthStation(mac [6]uint8) error // deauthenticate one client by MAC

type Station struct { MAC [6]uint8; RSSI int8; IP uint32 }
```

Events (same mailbox, same `Subscribe`/`Next` fan-out, no station semantics
changed):

| kind | value | meaning | Event fields |
| --- | --- | --- | --- |
| `EvAPStart` | 7 | `WIFI_EVENT_AP_START`: the AP is beaconing | - |
| `EvAPStop` | 8 | `WIFI_EVENT_AP_STOP` | - |
| `EvAPStaConnected` | 9 | a client associated | `MAC`, `AID` |
| `EvAPStaDisconnected` | 10 | a client left (or was deauthenticated) | `MAC`, `AID`, `Aux` = reason |
| `EvAPStaIPAssigned` | 11 | the AP's DHCP server leased an address | `MAC`, `IP` |

`MaxAPStations` (15) is the S31's `ESP_WIFI_MAX_CONN_NUM`: `APStations` sizes its
buffer from the glue's `llgo_wifi_max_sta()`, and the glue rejects a `MaxSTA`
above the chip's limit with `ESP_ERR_INVALID_ARG`, so a C2 (4) neither
over-allocates nor silently accepts more than it can serve.

Station kinds keep their numbers (1..6) and their meanings, so
`EvStaDisconnected` still means *the station lost its AP* - which is why the
auto-reconnect machine never reacts to a client leaving the soft AP (the probe
verifies that: it arms the machine and reports `phase = idle, sta link events = 0`).

`Event` gained two fields for the AP side (`MAC [6]uint8`, `AID uint16`); the
MAC and AID ride in the event rather than in a follow-up query because a client
can associate and leave between any two queries - `APStations` reports who is
here *now*, the event reports who came or went. `DeauthStation` needs no
bookkeeping of its own: the driver answers with the AID
(`esp_wifi_ap_get_sta_aid`) or with `ESP_ERR_NOT_FOUND` (0x105), and the glue
passes either through.

### Modes: the union of the configured interfaces

While the program has not picked a mode itself, the driver mode is the union of
the interfaces configured so far - `SetSTA` alone means STA (the pre-AP
behaviour), `SetAP` alone means AP, and the two together mean APSTA, in either
order. `SetModeSTA` / `SetModeAP` / `SetModeAPSTA` take the mode over
explicitly, and from then on the two `Set*` calls only configure their own
interface (and the driver rejects one whose interface the mode does not have -
`ESP_ERR_WIFI_MODE`, 0x3005, measured). A failed `SetAP` leaves no AP interface
behind: the config goes to the driver first, the AP netif (and its DHCP server)
is created only once the driver accepted it, so `GetAPIP`, `APStations` and
`GetAPConfig` keep answering `ESP_ERR_WIFI_MODE` for "no AP interface" instead
of handing out an address nobody configured.

### Glue version gate

The C symbols keep their names across package versions, so a project that
vendored an old `wifi_glue.c` would not fail the link - it would silently report
zeros for fields it does not know. `Setup` therefore compares the glue's
`llgo_wifi_glue_abi()` with the version this package speaks and returns
`ErrGlueABI` (0x7F01) on a mismatch before anything else runs.

## Build and run

Prerequisites: the llgo compiler (`/tmp/llgo-x` below), Docker image
`espressif/idf:latest` (IDF 6.3 for the S31), `esp-emu` v1.2.0, and a Go
toolchain for the host client.

```
# 1. Go -> IDF-linkable relocatable object. Also syncs the wifi and net glue
#    templates into main/ (gitignored build inputs) - run it before every
#    idf.py build.
cd idfproj/host && ./make-artifact.sh /tmp/llgo-x

# 2. Firmware. merge-bin takes -o relative to build/, so the merged image
#    lands in build/merged-binary.bin.
cd .. && docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
    bash -c 'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Automated acceptance: starts esp-emu, waits for the firmware's AP, dials
#    it through esp-emu's hostfwd from the host, and checks the reply.
cd host && ./run-acceptance.sh
```

`run-acceptance.sh` serializes against other `esp-emu` users on the machine
(`/tmp/esp-emu.lock`), releases the lock on every exit path, and writes the
console log to `host/logs/ap-acceptance.log` (`LOG=` overrides it).

The equivalent manual run, if you want to drive the client yourself:

```
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware idfproj/build/merged-binary.bin \
    --console uart0 --psram-size 16M \
    --net "user,hostfwd=tcp::18080-:8000" \
    --wifi-fw-ap-password llgoappass \
    --exit-on "[go] ap done" --timeout 150s
rmdir /tmp/esp-emu.lock

# in another terminal, once the log says "[go] ap ready":
go run ./idfproj/host/apclient -addr 127.0.0.1:18080
```

## Acceptance evidence

Recorded runs: `esp-emu` v1.2.0, `esp32s31-idf` firmware, 16 MiB PSRAM. The
`[go] ap ...` lines are byte-stable across runs (only the heap marks move, by a
few dozen bytes), and `host/golden/emu-ap-probe.txt` holds them:
`run-acceptance.sh` normalizes the four values that legitimately vary (the
initial station count, `aid`, `rssi`, `reason`), diffs everything else, and
prints `golden: MATCH` - or the diff - as part of the run.

The end-to-end chain, in the emulator's own words (IDF log lines interleaved
with the probe's):

```
I (470) esp_netif_lwip: DHCP server started on interface WIFI_AP_DEF with IP: 192.168.4.1
I (570) wifi:station: 02:e5:00:00:00:02 join, AID=1, bg, 20
I (570) esp_netif_lwip: DHCP server assigned IP to a client, IP is: 192.168.4.2
[go] ap pre-config get_ap_ip err = 12293 ESP_ERR_WIFI_MODE
[go] ap pre-config stations err = 12293 ESP_ERR_WIFI_MODE
[go] ap config ssid=llgo-ap ch = 6 maxsta = 2 auth = 3 hidden = 0
[go] ap config readback ok
[go] ap START
[go] ap ip = 192.168.4.1
[go] ap ip PASS
[go] ap STA_CONNECTED aid = 1 mac = 02:e5:00:00:00:02
[go] ap STA_IPASSIGNED ip = 192.168.4.2
[go] ap station mac = 02:e5:00:00:00:02 rssi = -30 ip = 192.168.4.2
[go] ap station list PASS
[go] ap tcp request = hello llgo-ap
[go] ap TCP PASS
[go] ap PASS
[go] ap STA_DISCONNECTED mac = 02:e5:00:00:00:02 reason = 2
[go] ap deauth PASS
[go] ap empty station list PASS
[go] ap deauth unknown mac err = 261 ESP_ERR_NOT_FOUND
[go] ap deauth unknown mac PASS
[go] ap STOP
[go] ap APSTA restart ok, ap ip = 192.168.4.1
[go] ap APSTA station interface started
[go] ap reconn: phase = idle sta link events = 0
[go] ap final stop ok
[go] ap mode-conflict set_ap err = 12293 ESP_ERR_WIFI_MODE
[go] ap mode-conflict PASS
[go] ap done
```

and on the host side, the client that dialled `127.0.0.1:18080`:

```
[host] ap client connected to 127.0.0.1:18080
[host] ap client reply = llgo-ap-ok
[host] ap client PASS
```

What that pair proves: the emulator's station associated with the firmware's
AP, the **firmware's own DHCP server** leased it `192.168.4.2`, and a TCP
connection opened on the host reached a listener inside the firmware over that
link and came back with the expected payload. Nothing in the firmware implements
DHCP or ARP; both are IDF's soft-AP stack.

The negative lines are checked claims, not prose: the three AP queries report
`ESP_ERR_WIFI_MODE` before any AP was configured, deauthenticating a MAC nobody
associated with reports the driver's `ESP_ERR_NOT_FOUND`, and an explicitly
chosen STA mode makes the driver refuse the AP configuration with
`ESP_ERR_WIFI_MODE`. All three codes were read off the wire, and the package's
documentation quotes them.

The comparison against the golden is part of the script: it normalizes the four
values that legitimately vary between runs (the initial station count, `aid`,
`rssi`, `reason`) and diffs everything else byte for byte. `--no-golden` skips
that and only checks the PASS markers.

## The DHCP server

The package does not implement DHCP and does not configure it: the server comes
with the AP interface. `SetAP` creates the default AP netif
(`esp_netif_create_default_wifi_ap()`), which

- owns the static address `192.168.4.1` (`GetAPIP` reads it back), and
- carries IDF's DHCP server (`CONFIG_LWIP_DHCPS`, on by default; the demo's
  `sdkconfig.defaults` spells it out).

The server starts leasing when the interface comes up on `WIFI_EVENT_AP_START`
(the log line above), and reports each lease as `IP_EVENT_ASSIGNED_IP_TO_CLIENT`
- which is the event the glue maps to `EvAPStaIPAssigned`, and the record
`APStations` reads back through `esp_wifi_ap_get_sta_list_with_ip`. A project
that wants a different pool, lease time or static assignments changes it with
`esp_netif_dhcps_*` on the same netif; nothing in `pkg/esp/wifi` gets in the way.

## Memory ledger (measured)

IDF internal-SRAM free bytes at the probe's marks, from a recorded run. The
first four rows reproduce exactly across runs; from `tcp` on, the figures wobble
by up to a few dozen bytes (driver and lwIP bookkeeping), which is why the golden
excludes the heap marks:

| mark | internal free | Δ from previous mark |
| --- | --- | --- |
| baseline (before `Setup`) | 307263 | - |
| `init` (after `Setup`) | 258231 | **−49032** (esp_wifi_init: driver tasks, buffers) |
| `ap-config` (after `SetAP`) | 257495 | **−736** (AP netif + its default config) |
| `ap-start` (after `Start`) | 246943 | **−10552** (beacons, AP state, DHCP server up) |
| `tcp` (one client associated, leased, TCP session) | 246359 | −584 |
| `ap-stop` (after `Stop`) | 255663 | **+9304** released |
| `apsta` (APSTA, AP back up, station configured) | 244275 | −11388 from `ap-stop` |
| `done` (after the second `Stop`) | 255663 | +11388, back to the `ap-stop` level |

- **AP mode costs 11 288 bytes (≈11 KiB) of internal SRAM on top of the
  station-sized driver**: `SetAP` + `Start` (measured), of which the network
  interface itself is 736 bytes and the rest is driver AP state, beacons and the
  DHCP server. A station-only firmware never pays it: the AP netif is created by
  `SetAP`, not by `Setup`.
- **`max_connection` is not a start-up memory factor** on this IDF (measured,
  A/B on the shipped build): `MaxSTA = 4` vs `MaxSTA = 2` moved `ap-start` by
  12 bytes (246955 vs 246943) - about 6 bytes per admitted station, i.e. the
  driver reserves no per-station buffer at start. Per-client cost shows up with
  the client: the −584 bytes between `ap-start` and `tcp` are one association
  plus one lwIP TCP session (the split is not separable in this run - the
  socket's buffers are the plausible larger part).
- **Nothing in AP mode touched PSRAM**: the psram figure stayed at 16 774 112
  bytes through the whole run. The Go heap is a separate 64 KiB static window
  that grows into PSRAM segments; the AP path allocates only small Go values
  (an `Event`, a `[]Station`, request/reply buffers).
- `HeapMin()` is the aggregate watermark (`esp_get_minimum_free_heap_size`), so
  it tracks total heap including PSRAM and is dominated by boot; the per-stage
  ledger above is the useful one.

## Station-path regression

The two packages share one mailbox, one driver handle and one Go API, so the
station half is regression-tested against the same change set: `_demo/esp-idf-wifi`
builds and links unchanged against this package (in a scratch copy, so the demo
tree itself is untouched), and its acceptance run passes with its own golden
(`net` + `net udp` + `wifi reconn` lines byte-identical): the station still
scans, associates, gets its DHCP lease, echoes TCP, and drives the auto-reconnect
machine through every transition. The AP probe white-boxes the part that matters
for AP mode: with the reconnect machine armed, `sta link events = 0` and
`phase = idle` after the whole AP episode, i.e. soft-AP traffic never wakes it.

## Emulator coverage boundary

Measured under `esp-emu` v1.2.0 (`docs/guides/NETWORKING.md`, "Firmware running
its own soft AP"): the emulator joins a firmware's own AP as a station, leases
an address from the firmware's DHCP server, and forwards `hostfwd` TCP rules
into the firmware through that link. **This demo is covered by that:**

- AP beacons/association and the WPA2-PSK handshake with
  `--wifi-fw-ap-password` (`[go] ap STA_CONNECTED`, `AID = 1`),
- the firmware's DHCP server leasing `192.168.4.2` (`DHCP server assigned IP to
  a client` in the IDF log, `[go] ap STA_IPASSIGNED`, and the same address in
  `APStations`),
- host → firmware TCP through the AP link (`[go] ap tcp request`,
  `[go] ap TCP PASS`, host client PASS),
- `WIFI_EVENT_AP_START/AP_STOP/AP_STACONNECTED/AP_STADISCONNECTED` and the DHCP
  lease event reaching the mailbox,
- `DeauthStation` (the emulator's station is deauthenticated and reported back
  with reason 2),
- mode switching to APSTA without losing the AP config, and the station
  interface coming up (`[go] ap APSTA station interface started`).

**Not modelable in this emulator, so not verified here:**

- **Hidden SSIDs**: `--wifi-fw-ap-password`'s documentation is explicit that a
  hidden-SSID AP is not joined by the emulator's station. `Hidden` is therefore
  passed through to the driver (and read back by `GetAPConfig`) but never
  exercised with a client; a hidden beacon is a hardware/real-client item.
- **WPA2/WPA3 transition and WPA3-SAE-only APs**: the emulator's station joins
  only open and WPA2-PSK firmware APs (its networking guide lists exactly those,
  and excludes "WPA3-SAE-only"), so a WPA3 AP auth mode is configured and read
  back by `GetAPConfig` but never joined here. The package's `AuthWPA3PSK` /
  `AuthWPA2WPA3PSK` paths reach the driver and stop there.
- **More than one client, roaming, power-save, RSSI behaviour**: the emulator
  gives one station. `MaxSTA` above 1, `APStations` with several records, and
  the per-client cost are unverified; an A/B run showed `max_connection` costs
  nothing at start, so the real multi-client question is throughput, not RAM.
- **Concurrent APSTA networking**: an APSTA firmware that actually associates
  would, under esp-emu's default addressing, put its station on the same
  `192.168.4.0/24` as its own AP (esp-emu warns about exactly that:
  `the firmware's own AP (192.168.4.1/24) shares the emulator's subnet`). The
  probe therefore configures the station and never connects. A meaningful APSTA
  routing test needs `--net net=10.0.2.0/24` and a real second network.
- **A client that stays connected while the AP is shot down / power-cycle
  behaviour**: the deauth path covers an intentional removal, not a link loss.
- **RF**: RSSI values (`rssi = -30`) are the emulator's synthetic figure.

## Pending verification on hardware

1. A real phone/laptop joining the AP: DHCP lease, DNS/NTP-less captive-portal
   behaviour, and `APStations` showing its MAC/IP.
2. `Hidden = true` (beacon only, no probe response with the SSID) with a real
   client.
3. WPA3-SAE and WPA2/WPA3 transition auth modes with real clients, including one
   that does not support PMF (it must fail to join `AuthWPA3PSK`).
4. Several simultaneous clients against `MaxSTA`, and `DeauthStation` on each.
5. APSTA with the station on a different subnet than the AP, verifying routing
   between the two interfaces.
6. Long-run stability with clients associating/leaving in a loop: mailbox
   overflow reporting (the 16-slot ring drops and reports once) and the driver's
   own AID recycling under `DeauthStation`.

## Glue surface (for folding into `pkg/esp/README.md`)

`pkg/esp/wifi/glue/wifi_glue.c` gained these symbols (all integer-and-pointer
only, per the boundary rules):

```
llgo_wifi_glue_abi(void)                               2: the contract version Setup checks
llgo_wifi_max_sta(void)                                ESP_WIFI_MAX_CONN_NUM (15 on S31, 4 on C2)
llgo_wifi_stop(void)                                   esp_wifi_stop
llgo_wifi_set_mode(mode)                               1 STA, 2 AP, 3 APSTA
llgo_wifi_set_ap(ssid, ssid_len, pass, pass_len,
                 channel, hidden, max_sta, auth_mode)  set_config(WIFI_IF_AP) + AP netif
llgo_wifi_get_ap_config(ssid, ssid_max, ssid_len,
                        ch, hidden, max_sta, auth_mode) esp_wifi_get_config(WIFI_IF_AP)
llgo_wifi_get_ap_ip(ip)                                AP netif address (error if no AP iface)
llgo_wifi_ap_sta_list(out, max, count)                 get_sta_list + get_sta_list_with_ip
llgo_wifi_deauth_sta(mac)                              esp_wifi_ap_get_sta_aid + deauth_sta
```

`llgo_wifi_poll` keeps its name and gained the `mac`/`aid` out-parameters, which
is exactly why `llgo_wifi_glue_abi` exists: a stale glue copy would answer zeros
instead of failing the link. Facts worth carrying into the family README:

- The mailbox slot is `{kind, aux, ip, mac[6], aid}`. The producer publishes
  `s_wr` with a release store after filling the slot and the consumer acquires
  it before reading; the consumer publishes `s_rd` the same way after copying
  the slot out. Only plain stores and fences are used - no read-modify-write, so
  nothing pulls in libatomic on RV32IMC - and the drop counter is reported by
  the consumer's own last-seen snapshot instead of being cleared by it.
- `esp_wifi_ap_get_sta_aid(mac, &aid)` is public in IDF 6.3 (`esp_wifi.h`) and
  is what turns a MAC into the AID that `esp_wifi_deauth_sta` wants; no MAC→AID
  bookkeeping is needed. A MAC that is not associated makes the driver report
  `ESP_ERR_NOT_FOUND` (0x105).
- `esp_wifi_ap_get_sta_list_with_ip` lives in `esp_wifi_ap_get_sta_list.h` (not
  `esp_wifi.h`) and is the DHCP-lease/ARP lookup behind `APStations`.
- The DHCP lease arrives as `IP_EVENT_ASSIGNED_IP_TO_CLIENT` (the id the pre-5.2
  stack called `WIFI_EVENT_AP_STAIPASSIGNED`), not as a `WIFI_EVENT`.
- `llgo_wifi_sta_t` and `llgo_ap_info_t` carry `_Static_assert`s for their
  layout, and `wifi_test.go` pins the Go side with `unsafe.Sizeof/Offsetof` -
  both halves of the boundary are checked from one place each.

## Not done

- No `Deinit` (`esp_wifi_deinit`) counterpart: the package keeps its one-shot
  `Setup` model, and `Stop` is the teardown this demo exercises.
- No IPv6 for the AP (`esp_netif_create_default_wifi_ap` leaves SLAAC/RA to
  lwIP's own defaults; nothing in the package configures it), and no DNS server
  or captive-portal plumbing.
- No WPA-Enterprise, WPS or mesh support.
- No per-client traffic accounting, RSSI events (`WIFI_EVENT_AP_PROBEREQRECVED`
  is not mapped) or `IP_EVENT_ASSIGNED_IP_TO_CLIENT`'s DHCP hostname.
