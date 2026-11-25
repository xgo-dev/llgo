# llgo × ESP-IDF NVS acceptance (ESP32-S31)

A self-contained pipeline that drives the `pkg/esp/nvs` key-value binding
from Go code compiled by llgo: write a wifi credential pair plus one value
of every shape (u8/i32/u64/string/blob), pin every error path to the code
IDF actually returns, then prove flash persistence the hard way - the
probe commits a stage counter, soft-restarts the chip (`esp_restart`), and
each subsequent boot only advances if the previous boot's Commits survived
inside the flash. The Go binding lives in the llgo repository
(`pkg/esp/nvs`, consumed through the module require + replace in `go.mod`);
this directory owns the acceptance program and the IDF project.

No peer process, no network, no board: the acceptance run is one esp-emu
boot with a self-contained, restart-driven probe - the smallest full-stack
gate of the storage side of the pkg/esp family.

## What is under test

| segment | what it proves |
| --- | --- |
| write | `Init` (fresh-flash format), namespace `wifi` with `ssid`/`pass` strings + one `Commit`; namespace `app` with u8/i32/u64/string/blob round-trips read back immediately (cache) |
| errors | `wrong-type` (i32 read of a u8 key), `missing-key`, `key-too-long`, `key-empty`, `value-nul` (all refused Go-side), `read-cap` (a 600-byte string reads back as the binding's own `NVS_ERR_TOO_LONG` 0x7501, value intact), `deleted-key`, `delete-absent`, `namespace-void` (`Keys()` on a never-written namespace) |
| keys | `Keys()` on both written namespaces, sorted by the probe so the golden never depends on NVS slot order |
| stats | `ReadStats()` - the partition-wide entry ledger (floated in the golden) |
| persist | stage counter in namespace `eternal`: write → `Commit` → `esp_restart` → verify (boots = 1) → `Commit` → restart → re-verify (boots = 2) → PASS. Two soft restarts, every committed value compared byte-for-byte, the deleted key still absent |

The `wrong-type` check pins measured IDF v6.3 behavior, not the doc ideal:
`nvs_page.cpp`'s hash-list indexes items by (namespace, type, key), so an
i32 read of a u8-stored key misses the hash and reports
`ESP_ERR_NVS_NOT_FOUND` before the page scan could return
`ESP_ERR_NVS_TYPE_MISMATCH`.

## Layout

```
esp-idf-nvs/
├── go.mod                  Go module (module llgonvs; requires + replaces
│                           github.com/xgo-dev/llgo, the binding package's home)
├── goapp/main.go           Acceptance probe: stage-keyed restart cycle
│                           (write → verify → re-verify → PASS) + all
│                           error paths + keys + stats
├── goapp/hosted.go         Host-build push-linkname stubs, so `go build` links
├── golden/emu-nvs-probe.txt  Golden for the [go] nvs lines (+ floater rule)
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt      (generated from _demo/common/)
    ├── sdkconfig.defaults  PSRAM, 240 MHz, 3.9 MiB app partition (the
    │                       common partitions.csv carries the NVS partition
    │                       this probe exercises: nvs, 0x9000, 0x6000)
    ├── partitions.csv      (generated)
    ├── main/
    │   ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + REQUIRES nvs_flash
    │   ├── demo_glue.c     C-side heap ledger; checked in
    │   ├── nvs_glue.c      pkg/esp/nvs glue template, synced at build time
    │   │                   (gitignored): the static value buffer and key
    │   │                   table behind the binding's capacity contract
    │   └── app_main.c, llgo_dram.ld  (generated from _demo/common/)
    └── host/
        ├── make-artifact.sh  (symlink to the shared pipeline)
        ├── artifact.conf   GLUES=nvs, default 64 KiB GC window, default
        │                   stacks (every glue call is a shallow NVS access)
        └── llgo_syms.S     (generated)
```

## Build and run

Prerequisites: an llgo compiler (`/tmp/llgo-nvsc` below, built with
`LLVM_CONFIG=/opt/homebrew/opt/llvm/bin/llvm-config go build -o /tmp/llgo-nvsc ./cmd/llgo`),
the vendored riscv32 libffi archive at
`runtime/internal/clite/ffi/riscv32/libffi.a` (gitignored, built per that
directory's README), Docker image `espressif/idf:latest` (pins the IDF
v6.3-dev API), esp-emu v1.2.0.

```
# 1. Compile the Go probe into the IDF-linkable object (also syncs the glue
#    and the scaffold into idfproj/main/, so run it before idf.py).
cd _demo/esp-idf-nvs
LLGO_ROOT=$(cd ../.. && pwd) bash idfproj/host/make-artifact.sh /tmp/llgo-nvsc

# 2. IDF firmware (set-target pins the chip on the first build; merge-bin
#    lands the image at build/merged-binary.bin).
docker run --rm -v "$PWD/idfproj":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. Acceptance run under the emulator (the probe exits on its done marker;
#    three boots happen inside this one process).
timeout 150 ~/.local/bin/esp-emu --chip esp32s31 \
    --firmware idfproj/build/merged-binary.bin --console uart0 \
    --psram-size 16M --timeout 120s --exit-on "[go] nvs done" | tee /tmp/nvs-emu.log

# 4. Golden check.
grep -E '^\[go\] nvs' /tmp/nvs-emu.log | tr -d '\r' \
    | sed -E 's/used = [0-9]+ free = [0-9]+ avail = [0-9]+ total = [0-9]+ ns = [0-9]+/used = * free = * avail = * total = * ns = */' \
    | diff - <(grep -v '^#' golden/emu-nvs-probe.txt)
```

Expected console output (interleaved with the IDF boot log; `[c]` lines are
the C side's heap marks; one boot sequence of the three shown):

```
[go] nvs stage0-fresh err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs wifi ok ssid pass committed
[go] nvs roundtrip ok u8 i32 u64 str blob64
[go] nvs wrong-type err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs missing-key err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs key-too-long err = 0x0102 ESP_ERR_INVALID_ARG
[go] nvs key-empty err = 0x0102 ESP_ERR_INVALID_ARG
[go] nvs value-nul err = 0x0102 ESP_ERR_INVALID_ARG
[go] nvs read-cap ok 600-in 512-max err = 0x7501 NVS_ERR_TOO_LONG
[go] nvs deleted-key err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs delete-absent err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs namespace-void err = 0x1102 ESP_ERR_NVS_NOT_FOUND
[go] nvs keys app n = 6 blob i32 str t8 u64 u8
[go] nvs keys wifi n = 2 pass ssid
[go] nvs stats used = 18 free = 738 avail = 612 total = 756 ns = 4
[c] nvs heap stage0-done internal=337835 psram=16774636
[go] nvs stage0 done
...
[go] nvs verify ok boots = 1
[go] nvs stage1 done
...
[go] nvs verify ok boots = 2
[c] nvs heap done internal=337835 psram=16774636
[go] nvs PASS
[go] nvs done
```

## Cross-process persistence (esp-emu --save-state)

The single run above proves persistence across `esp_restart` inside one
emulator process. The emulator also round-trips the whole flash through a
file across processes: run the same firmware with `--save-state` and an
early exit marker (the probe stops after the stage-0 write), then boot the
saved image in a second esp-emu process - the fresh process picks up at
stage 1 and walks the same verify → stage-2 → PASS path:

```
cp idfproj/build/merged-binary.bin /tmp/nvs-state.bin
esp-emu --chip esp32s31 --firmware /tmp/nvs-state.bin --console uart0 \
    --psram-size 16M --save-state --exit-on "[go] nvs stage0 done"
esp-emu --chip esp32s31 --firmware /tmp/nvs-state.bin --console uart0 \
    --psram-size 16M --exit-on "[go] nvs done"     # runs stages 1 and 2, PASS
```

`--save-state` rewrites the firmware file with the full 4 MiB flash image
(a 696 KiB merged binary grows to 4 MiB); the NVS partition entries are
visible in the file at 0x9000 (`eternal`/`wifi` namespaces, `ssid` =
`llgo-home`), which is the byte-level cross-evidence that the commit path
lands in the modeled flash.

## Emulator fidelity notes (measured, esp-emu 1.2.0, chip esp32s31)

- **NVS is modeled faithfully for this surface**: `nvs_flash_init` formats
  the partition on first boot and is idempotent after; Set/Get/Erase/
  Commit behave like the IDF contract; the entry iterator (`Keys`) walks
  real committed entries; the stats ledger reports live numbers.
- **Flash persists across `esp_restart`** (twice per acceptance run) and
  across emulator processes via `--save-state` - the strongest persistence
  claims this environment can make without real silicon.
- Not modeled here / not claimed: flash wear characteristics (erase
  cycling, page A/B failure handling under power loss), NVS encryption
  (no key partition in this layout), multi-partition setups, concurrent
  access from a second task. See the binding's README section for the
  hardware-only verification list.
- The probe is restart-bounded by construction: if a boot finds the stage
  key missing it reports `stage0-fresh` and loops - an emulator flash
  regression shows up as a repeated stage-0 sequence until the run
  timeout, never as a hang.
