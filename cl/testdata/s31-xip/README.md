# ESP32-S31 flash (XIP) boot assets

Second-stage boot binaries for the flash (XIP) baremetal tests in
`cl/embed_s31_test.go` (`TestS31XIPReflectSmoke`, `TestS31XIPFmtConsole`).
The tests merge these images with the application image into a 4 MiB flash
dump and boot it under esp-emu through the ESP-IDF second-stage bootloader;
the SRAM-only target (`esp32s31-basic`) does not use them.

| File | Size | SHA-256 |
| --- | ---: | --- |
| `bootloader.bin` | 21552 | `e4ec8f63c207599198de39cbf1a72b89d1db7441669dc184fb8037063a81a917` |
| `partition-table.bin` | 3072 | `7f00b6c042a89b15b0cac534f82ed988caf29278ff5700b0c511eb1b5bb7c820` |

Source: built with the ESP-IDF `espressif/idf` Docker image for the esp32s31
target (`docker run espressif/idf idf.py build`, bootloader + partition table
outputs of a default project). The tests only need *any* valid esp32s31
bootloader/partition-table pair: llgo's XIP app image is self-contained
aside from second-stage boot, so these binaries are regenerated only when a
bootloader change is deliberately adopted. CI wires them in through the
`.github/workflows/baremetal-emu.yml` environment
(`LLGO_S31_XIP_BOOTLOADER`, `LLGO_S31_XIP_PARTTABLE`).

Note: `.gitignore` ignores `*.bin`, so these two fixtures are force-added
(`git add -f`) and stay tracked afterwards.
