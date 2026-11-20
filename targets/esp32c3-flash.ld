/* ESP32-C3 ROM entry points used by targets/device/esp/esp32c3.S.
 * Source: ESP-IDF components/esp_rom/esp32c3/ld/esp32c3.rom.ld.
 */
PROVIDE(Cache_Invalidate_ICache_All = 0x400004d8);
PROVIDE(Cache_Suspend_ICache = 0x40000524);
PROVIDE(Cache_Resume_ICache = 0x40000528);
PROVIDE(Cache_MMU_Init = 0x4000055c);
PROVIDE(Cache_Dbus_MMU_Set = 0x40000564);

PROVIDE(USB_SERIAL_JTAG = 0x60043000);

MEMORY
{
    iram_seg (rx) : ORIGIN = 0x40380000, LENGTH = 400K
    dram_seg (rw) : ORIGIN = 0x3FC80000, LENGTH = 256K
    irom_seg (rx) : ORIGIN = 0x42000000, LENGTH = 4M
    drom_seg (r)  : ORIGIN = 0x3C000000, LENGTH = 4M
}

ENTRY(call_start_cpu0)

SECTIONS
{
    /* The mask ROM loads the boot code before the flash cache is enabled. */
    .iram :
    {
        _iram_start = .;
        KEEP(*(.text._start))
        KEEP(*(.init))
        *(.iram*)
        . = ALIGN(4);
        _iram_end = .;
    } > iram_seg

    /* IRAM and DRAM alias the same SRAM. Skip the boot code's storage. */
    .dram0.dummy (NOLOAD) : { . += _iram_end - _iram_start; } > dram_seg
    .stack (NOLOAD) : ALIGN(16)
    {
        _stack_end = .;
        . += 16K;
        __stack = .;
    } > dram_seg

    .data :
    {
        _data_start = .;
        *(.data .data.* .data1 .gnu.linkonce.d.* .dram*)
        SORT(CONSTRUCTORS)
    } > dram_seg
    .sdata :
    {
        PROVIDE(__global_pointer$ = . + 0x800);
        *(.srodata .srodata.* .sdata .sdata.* .gnu.linkonce.s.*)
        _edata = .;
        PROVIDE(edata = .);
    } > dram_seg
    .bss (NOLOAD) :
    {
        __bss_start = .;
        *(.dynsbss .sbss .sbss.* .gnu.linkonce.sb.* .scommon)
        *(.dynbss .bss .bss.* .gnu.linkonce.b.* COMMON)
        . = ALIGN(4);
        _end = .;
        PROVIDE(end = .);
    } > dram_seg

    /* Keep the checksummed ROM image before the first XIP section. Leave
     * room for its segment headers, alignment, checksum and digest. The boot
     * code maps physical flash offsets at 64 KiB intervals.
     */
    _flash_image_end = ALIGN(SIZEOF(.iram) + SIZEOF(.data) + SIZEOF(.sdata) + 1K, 64K);
    .flash.text (ORIGIN(irom_seg) + _flash_image_end) :
    {
        *(.text .text.* .stub .gnu.linkonce.t.*)
        KEEP(*(.fini))
        PROVIDE(__etext = .);
        PROVIDE(_etext = .);
        PROVIDE(etext = .);
    } > irom_seg

    .flash.rodata (ORIGIN(drom_seg) + ALIGN(ADDR(.flash.text) - ORIGIN(irom_seg) + SIZEOF(.flash.text), 16)) :
    {
        *(.rodata .rodata.* .rodata1 .gnu.linkonce.r.*)
        *(.sdata2 .sdata2.* .gnu.linkonce.s2.*)
        *(.sbss2 .sbss2.* .gnu.linkonce.sb2.*)
        . = ALIGN(4);
        __cpu_frequency = .;    LONG(20000000);
        __uart0_clkdiv_reg = .; LONG(0x60000014);
        __uart0_clkdiv_val = .; LONG(20000000 / 115200);
        __uart0_tx_addr = .;    LONG(0x60000000);
        __uart0_status = .;     LONG(0x6000001C);
    } > drom_seg

    .eh_frame_hdr : { *(.eh_frame_hdr .eh_frame_entry .eh_frame_entry.*) } > drom_seg
    .eh_frame : { KEEP(*(.eh_frame)) *(.eh_frame.*) } > drom_seg
    .preinit_array :
    {
        PROVIDE_HIDDEN(__preinit_array_start = .);
        KEEP(*(.preinit_array))
        PROVIDE_HIDDEN(__preinit_array_end = .);
    } > drom_seg
    .init_array :
    {
        PROVIDE_HIDDEN(__init_array_start = .);
        KEEP(*(SORT_BY_INIT_PRIORITY(.init_array.*) SORT_BY_INIT_PRIORITY(.ctors.*)))
        KEEP(*(.init_array .ctors))
        PROVIDE_HIDDEN(__init_array_end = .);
    } > drom_seg
    .fini_array :
    {
        PROVIDE_HIDDEN(__fini_array_start = .);
        KEEP(*(SORT_BY_INIT_PRIORITY(.fini_array.*) SORT_BY_INIT_PRIORITY(.dtors.*)))
        KEEP(*(.fini_array .dtors))
        PROVIDE_HIDDEN(__fini_array_end = .);
    } > drom_seg

    /DISCARD/ : { *(.note.GNU-stack .gnu_debuglink .gnu.lto_*) }
}

_globals_start = ALIGN(_data_start, 4);
_globals_end = ALIGN(_end, 4);
_heapStart = _end;
_heapEnd = ORIGIN(dram_seg) + LENGTH(dram_seg);
_stack_top = __stack;
