# llgo × ESP-IDF WiFi pipeline (ESP32-S31)

A self-contained pipeline that drives the ESP-IDF `esp_wifi` stack from
Go code compiled by llgo: setup → start → scan → connect → DHCP, with the
acceptance run automated under esp-emu. Nothing here modifies the llgo
repository itself; the Go binding lives in this directory's own module.

## Layout

```
esp-idf-wifi/
├── go.mod                  Go module (module llgoidf)
├── goapp/main.go           Acceptance program (println + WiFi sequence)
├── goapp/espwifi/          The Go binding package
│   ├── wifi.go             Public API, mailbox wait helper, esp_err names
│   ├── wifi_idf.go         //go:build llgo.idf - bodyless linkname externs
│   └── wifi_stub.go        //go:build !llgo.idf - host-build stubs
└── idfproj/                Minimal ESP-IDF project
    ├── CMakeLists.txt
    ├── sdkconfig.defaults  8 KiB main task, 240 MHz, WiFi tasks on CPU1
    ├── host/               Host-side artifact pipeline (one command)
    │   ├── make-artifact.sh  llgo build -> IDF-linkable relocatable object
    │   └── llgo_syms.S       GC heap + bootstrap stack window providers
    └── main/               The main component
        ├── CMakeLists.txt  Sources + WHOLE_ARCHIVE + link options
        ├── app_main.c      app_main -> llgo entry; _write/_exit shims
        ├── wifi_glue.c     esp_wifi C glue (setup/config/events/mailbox)
        └── llgo_dram.ld    _globals_start/_globals_end for tinygogc
```

## Architecture

ESP-IDF/FreeRTOS owns the platform; llgo compiles `goapp` into a
relocatable object (`-target esp32s31-idf`, `ld.lld -r`) that the main
component links into the firmware. `app_main` calls the generated C entry,
which hands package init and `main.main` to the llgo fiber scheduler on the
FreeRTOS main task.

The Go ↔ C boundary follows three rules:

1. **Integer-and-pointer-only signatures.** llgo's RV32 codegen is
   soft-float; the object's ABI flag is patched to single-float because no
   float value ever crosses the link (see `make-artifact.sh` step 4).
2. **Events travel through a C-side mailbox, never callbacks.** esp_event
   handlers run on the event-loop task; the llgo fiber scheduler is
   single-task global state, so no second FreeRTOS task may enter Go. The
   handler copies into a 16-slot single-producer/single-consumer ring and Go
   polls it paced by `vTaskDelay(10)` = 100 ms — which also keeps the idle
   tasks fed for the watchdog. Overflow drops the newest event and is
   reported once.
3. **No large stack buffers in glue.** Go goroutines run on 8 KiB fiber
   stacks, so bulk data (scan records) lives in static storage, and
   `WIFI_INIT_CONFIG_DEFAULT()` stays in C (it embeds Kconfig constants and
   function pointers Go cannot spell).

Every binding returns the raw `esp_err` code (0 = ESP_OK);
`espwifi.ErrName` maps the codes this pipeline can hit to their IDF names.

### Why the main component is WHOLE_ARCHIVE

`llgo_out.o` is the last item on the final link line and the ESP-IDF link
uses no `--start-group`, so each archive is scanned only against the
undefined symbols known at its position. The glue (`wifi_glue.c`) is
referenced solely from `llgo_out.o`, and a plain archive would never pull
it — the same reason the `llgo_syms.S` runtime providers are merged inside
`llgo_out.o` itself. `WHOLE_ARCHIVE` keeps every main-component object in
the link unconditionally. (The baseline demo resolved `vTaskDelay` from
`libmain.a` only by accident: `app_main.c` itself references FreeRTOS
symbols and was pulled early by the IDF startup, and its undefineds then
pulled `libfreertos.a` in a later archive round.)

## Build and run

Prerequisites: llgo compiler (`/tmp/llgo-wifi` below), Docker image
`espressif/idf:latest`, `esp-emu` on the host.

```
# 1. Go -> relocatable object (prints the remaining undefined symbols;
#    llgo_wifi_* / vTaskDelay / stdio must all be in that list)
cd idfproj/host && ./make-artifact.sh /tmp/llgo-wifi

# 2. IDF firmware
cd .. && docker run --rm -v "$PWD":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o build/llgo_flash.bin'

# 3. Acceptance run under esp-emu (emu's built-in soft AP is myssid/mypassword)
while ! mkdir /tmp/esp-emu.lock; do sleep 5; done
~/.local/bin/esp-emu --chip esp32s31 --firmware build/llgo_flash.bin \
    --console uart0 --timeout 60s \
    --wifi-ssid myssid --wifi-password mypassword \
    --exit-on "[go] wifi STA_START"
rmdir /tmp/esp-emu.lock
```

Expected console output (interleaved with the IDF boot log):

```
[c] wifi glue: IDF v6.3-dev-..., free heap ... bytes
[go] wifi setup err=0
[go] wifi STA_START
[go] done
```

The Go sequence in `main.go` runs the acceptance stages: setup, event
mailbox, station config (`myssid`/`mypassword`), start, STA_START. Scan and
connect/DHCP stages extend the same sequence.

## Notes

- The WiFi stack's tasks are pinned to CPU1
  (`CONFIG_ESP_WIFI_TASK_PINNED_TO_CORE_1`) so scan/association bursts never
  compete with the llgo scheduler on CPU0.
- The IDF heap must absorb the WiFi allocations (~100-150 KiB) next to the
  static 256 KiB llgo GC heap; the glue prints the free heap before setup to
  make that budget visible.
- Host builds of `goapp/espwifi` compile against `wifi_stub.go` (`go build`,
  `go vet`, gopls); `goapp/main.go` itself is llgo-only because its FreeRTOS
  linknames have no host stubs.
