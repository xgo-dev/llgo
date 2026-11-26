# llgo × ESP-IDF core affinity (ESP32-S31): runtime.LockCore acceptance

A self-contained pipeline that proves `runtime.LockCore(id int) error` on the
esp32s31-idf target: the Go scheduler lives on the FreeRTOS main task, and
LockCore pins that task to the requested HP core at runtime - verified through
both the kernel's own pin getter (xTaskGetCoreID) and the executing core
(esp_cpu_get_core_id) after a scheduling point, plus the out-of-range
rejection. The layout copies `_demo/esp-idf-bt` (make-artifact.sh → llgo_out.o
→ IDF main component), stripped to the core-affinity surface.

## Why the pin goes through a demo-side C shim

The design's first candidate, `vTaskCoreAffinitySet`, belongs to the Amazon
SMP FreeRTOS kernel (components/freertos/FreeRTOS-Kernel-SMP); its Kconfig
entry is `depends on IDF_TARGET_ESP32`, so an S31 firmware never links it.
The kernel S31 actually runs (components/freertos/FreeRTOS-Kernel) pins a
task once, at creation: `xTaskCreatePinnedToCore` writes `TCB.xCoreID`, and
no kernel API writes that field afterwards (verified against
espressif/idf:latest = v6.3.0; TCB_t is private to tasks.c and no public or
esp_private header exposes a setter).

`idfproj/main/lockcore_glue.c` is therefore the pin. It writes
`TCB.xCoreID` of the calling task directly, with three layers of protection:

1. **Version gate** - the file refuses to compile against any IDF release
   other than 6.3.0, the one the offset below was extracted from.
2. **Provenance** - `LLGO_TCB_XCOREID_OFFSET` came from the DWARF of this
   exact build's `tasks.c` object, not from guessing (recipe below).
3. **Self-check, fail closed** - before writing, the bytes at the offset must
   read back as the same core id the public `xTaskGetCoreID` getter returns;
   after writing, the getter must confirm the new id, otherwise the write is
   reverted and refused. `runtime.LockCore` additionally reads the getter
   itself before reporting success, and a layout drift surfaces as an error,
   never as memory corruption.

The Go side (runtime/internal/runtime/lockcore_idf.go) binds
`xTaskGetCurrentTaskHandle` / `xTaskGetCoreID` as plain kernel symbols and
`C.llgoLockCore` as a link-environment symbol - the same bodyless-linkname
pattern as `idf_freertos.go`'s `vTaskDelay`. The runtime itself ships a
weak `llgoLockCore` default (honest refusal), so a firmware without the
glue still links and `runtime.LockCore` reports its documented error; this
demo's `main/lockcore_glue.c` defines the strong override that really
pins.

## Extracting LLGO_TCB_XCOREID_OFFSET for a new IDF version

After `idf.py set-target esp32s31 && idf.py build` (any esp32s31 project
with the same FreeRTOS config family), dump the member offset of `xCoreID`
in `tskTCB` from the kernel object's DWARF and update the `#define`:

```
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest bash -c '
  riscv32-esp-elf-objdump --dwarf=info \
    build/esp-idf/freertos/CMakeFiles/__idf_freertos.dir/FreeRTOS-Kernel/tasks.c.obj \
  | grep -B1 -A3 "DW_AT_name.*xCoreID"'
```

The `DW_AT_data_member_location` of that member is the offset. The glue's
`_Static_assert` keeps it 4-byte aligned, and the self-check catches any
mismatch at runtime.

## Layout

```
esp-idf-lockcore/
├── go.mod                  Go module (module llgolockcore; requires + replaces
│                           github.com/xgo-dev/llgo)
├── goapp/main.go           Acceptance probe: start (pin=0 core=0) → LockCore(1)
│                           → getter + executing core read back 1 → LockCore(0)
│                           → read back 0 → LockCore(9) refused → PASS
├── golden/emu-lockcore.txt Golden for the [go] lockcore lines (fully
│                           deterministic; no floaters)
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt
    ├── sdkconfig.defaults  240 MHz, main-task stack headroom, llgo-compatible
    │                       stack guards, 3.9 MiB app partition - and
    │                       deliberately NO CONFIG_ESP_MAIN_TASK_AFFINITY pin:
    │                       the probe boots on the CPU0 default so LockCore(1)
    │                       has a real move to perform
    ├── partitions.csv
    ├── host/
    │   ├── make-artifact.sh  llgo build → IDF-linkable relocatable object
    │   └── llgo_syms.S       GC window + bootstrap stack providers
    │                         (llgoFixStreams legacy no-op kept for
    │                         link compatibility)
    └── main/
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── app_main.c      app_main → llgo entry; _write/_exit shims
        ├── llgo_dram.ld    _globals_start/_globals_end assignments
        └── lockcore_glue.c llgoLockCore pin shim + llgoCurrentCore wrapper
```

## Build and run

```
cd idfproj
./host/make-artifact.sh /path/to/llgo        # Go → llgo_out.o (host side)
docker run --rm -v "$PWD":/project -w /project espressif/idf:latest \
  bash -c 'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'
~/.local/bin/esp-emu --chip esp32s31 --firmware merged-binary.bin \
  --elf build/llgo_idf_lockcore.elf --console uart0
```

Acceptance check (strip CR from the USB-serial console stream first):

```
grep -E '^\[go\] lockcore' <emu log> | tr -d '\r' | diff - <(grep -v '^#' golden/emu-lockcore.txt)
```
The right side strips the golden's leading comment block (the diff target is
the output lines only).
