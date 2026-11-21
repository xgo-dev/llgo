// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both. The image only
// fits the 512 KiB S31 SRAM and the XIP flash layouts; the 256 KiB ESP32-C3
// IRAM target cannot hold it. Baremetal exposes no argv and no filesystem,
// so the probes only assert that Args is a slice, Getenv misses, and the
// libc stubs report errors for nonexistent paths.
package main

import "os"

func main() {
	println("args-slice", len(os.Args) >= 0)
	println("getenv-miss", os.Getenv("LLGO_NO_SUCH_VAR_XYZ") == "")
	f, err := os.Create("/nonexistent-zz-dir/probe.txt")
	println("create-miss", f == nil, err != nil)
	_, sterr := os.Stat("/nonexistent-zz-dir/probe.txt")
	println("stat-miss", sterr != nil)
}
