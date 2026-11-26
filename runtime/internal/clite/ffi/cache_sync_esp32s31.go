//go:build tinygo.riscv32 && esp32s31

package ffi

const riscv32CacheSyncFiles = "; _wrap/cache_sync_esp32s31.c"
const riscv32CacheSyncLinkFlags = " --wrap=__clear_cache"
