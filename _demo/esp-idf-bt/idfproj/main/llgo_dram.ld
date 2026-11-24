/* Extra linker script for the llgo component (appended to the generated
 * ESP-IDF scripts through the component's -T flag).
 *
 * tinygogc scans [globalsStart, globalsEnd) as a conservative root region
 * (gc_link.go gcMarkReachable). The ESP32-S31 generated script places every
 * component's writable data in .dram0.data and zero-initialized data in
 * .dram0.bss, so spanning both - including IDF's own data - covers the Go
 * globals no matter where the linker interleaves llgo_out.o. For a
 * conservative collector over-marking is safe: extra false roots only keep
 * unneeded blocks alive.
 *
 * The assignments are evaluated during layout, so this script's position in
 * the -T list does not matter.
 */

_globals_start = ADDR(.dram0.data);
_globals_end = ADDR(.dram0.bss) + SIZEOF(.dram0.bss);
