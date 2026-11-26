# ESP32-S31 application layout for the flash (XIP) target, booted through the
# ESP-IDF second-stage bootloader.
#
# The image writer emits one segment per non-empty output section, sorted by
# virtual address ascending, each preceded by an 8-byte segment header, all
# behind a 24-byte image header. The writable segment (VMAs at 0x2f000000)
# therefore comes first and its data starts at image offset 0x20, exactly
# where the bootloader reads the ESP-IDF application descriptor — provided by
# the .appdesc linker input. The two mapped segments follow in the fixed
# DROM->IROM slot order the bootloader asserts: .rodata, then .text.
#
# The bootloader maps each mapped segment's 64 KiB MMU pages such that the
# segment's virtual address and its flash offset (app partition base + data
# offset in the image file) agree modulo the MMU page size. Placing each
# mapped section at VMA = 0x40000000 + (its data offset in the image file)
# satisfies that unconditionally: the app partition starts at a 64 KiB
# boundary, so both sides reduce to the same file offset, and each segment
# maps linearly inside its page pair. The offsets chain from SIZEOF, which is
# exactly the byte count the writer stores per section.

STACK_SIZE = 16K;

/* Default entry point */
ENTRY(_start)
SECTIONS
{
  /* Writable segment 0; its first 256 bytes are the app descriptor read
     from image offset 0x20. */
  .data :
  {
    KEEP(*(.appdesc))
    _data_start = .;
    *(.data .data.* .gnu.linkonce.d.*)
    SORT(CONSTRUCTORS)
    *(.data1)
    PROVIDE(__global_pointer$ = . + 0x800);
    *(.srodata.cst16) *(.srodata.cst8) *(.srodata.cst4) *(.srodata .srodata.*)
    *(.sdata .sdata.* .gnu.linkonce.s.*)
    _edata = .; PROVIDE (edata = .);
    . = ALIGN(16);
  } > sram_seg

  IMAGE_HEADER_SIZE = 24;
  SEGMENT_HEADER_SIZE = 8;
  _rodata_off = IMAGE_HEADER_SIZE
              + SEGMENT_HEADER_SIZE
              + SIZEOF(.data)
              + SEGMENT_HEADER_SIZE;
  _text_off = _rodata_off
            + SIZEOF(.rodata)
            + SEGMENT_HEADER_SIZE;

  /* Read-only data: mapped segment 0 (DROM slot). */
  .rodata 0x40000000 + _rodata_off :
  {
    *(.rodata .rodata.* .gnu.linkonce.r.*)
    *(.rodata1)
    *(.sdata2 .sdata2.* .gnu.linkonce.s2.*)
    *(.sbss2 .sbss2.* .gnu.linkonce.sb2.*)
    . = ALIGN(4);
    __cpu_frequency = .;
    LONG(CPU_FREQUENCY);
    __uart0_clkdiv_reg = .;
    LONG(UART0_CLKDIV_REG);
    __uart0_clkdiv_val = .;
    LONG(UART0_CLKDIV_VAL);
    __uart0_tx_addr = .;
    LONG(UART0_TX_ADDR);
    __uart0_status = .;
    LONG(UART0_STATUS);
    . = ALIGN(4);
    PROVIDE_HIDDEN (__preinit_array_start = .);
    KEEP (*(.preinit_array))
    PROVIDE_HIDDEN (__preinit_array_end = .);
    PROVIDE_HIDDEN (__init_array_start = .);
    KEEP (*(SORT_BY_INIT_PRIORITY(.init_array.*) SORT_BY_INIT_PRIORITY(.ctors.*)))
    KEEP (*(.init_array EXCLUDE_FILE (*crtbegin.o *crtbegin?.o *crtend.o *crtend?.o ) .ctors))
    PROVIDE_HIDDEN (__init_array_end = .);
    PROVIDE_HIDDEN (__fini_array_start = .);
    KEEP (*(SORT_BY_INIT_PRIORITY(.fini_array.*) SORT_BY_INIT_PRIORITY(.dtors.*)))
    KEEP (*(.fini_array EXCLUDE_FILE (*crtbegin.o *crtbegin?.o *crtend.o *crtend?.o ) .dtors))
    PROVIDE_HIDDEN (__fini_array_end = .);
    KEEP (*crtbegin.o(.ctors))
    KEEP (*crtbegin?.o(.ctors))
    KEEP (*(EXCLUDE_FILE (*crtend.o *crtend?.o ) .ctors))
    KEEP (*(SORT(.ctors.*)))
    KEEP (*(.ctors))
    KEEP (*crtbegin.o(.dtors))
    KEEP (*crtbegin?.o(.dtors))
    KEEP (*(EXCLUDE_FILE (*crtend.o *crtend?.o ) .dtors))
    KEEP (*(SORT(.dtors.*)))
    KEEP (*(.dtors))
    . = ALIGN(4);
  } > flash_seg

  /* Code: mapped segment 1 (IROM slot). SUBALIGN(4) keeps the stored input
     placement alignment low so the explicit VMA (which mirrors the image
     file offset, and therefore cannot be rounded up) does not trip lld's
     output-alignment warning; RISC-V code placement has no functional
     16-byte requirement. */
  .text 0x40000000 + _text_off : SUBALIGN(4)
  {
    KEEP(*(.text._start))
    KEEP (*(SORT_NONE(.init)))
    *(.text .stub .text.* .gnu.linkonce.t.*)
    /* .gnu.warning sections are handled specially by elf32.em.  */
    *(.gnu.warning)
    KEEP (*(SORT_NONE(.fini)))
    PROVIDE (__etext = .);
    PROVIDE (_etext = .);
    PROVIDE (etext = .);
    . = ALIGN(4);
  } > flash_seg

  .bss    (NOLOAD)  :
  {
    __bss_start = .;
    *(.dynsbss)
    *(.sbss .sbss.* .gnu.linkonce.sb.*)
    *(.scommon)
    *(.dynbss)
    *(.bss .bss.* .gnu.linkonce.b.*)
    *(COMMON)
    . = ALIGN(. != 0 ? 32 / 8 : 1);

    . = ALIGN(32 / 8);
    . = ALIGN(32 / 8);
    _end = .; PROVIDE (end = .);
  } > sram_seg

  /* The stack sits at the very top of the application SRAM; the heap gets
     everything between .bss and the stack. */
  .stack ORIGIN(sram_seg) + LENGTH(sram_seg) - STACK_SIZE (NOLOAD) :
  {
    _stack_end = .;
    . += STACK_SIZE;
    __stack = .;
    . = ALIGN(16);
  } > sram_seg

  /* Check if data + heap + stack stays inside SRAM */
  ASSERT(_end <= _stack_end, "region SRAM overflowed by .data and .bss sections")

  /* Stabs debugging sections.  */
  .stab          0 : { *(.stab) }
  .stabstr       0 : { *(.stabstr) }
  .stab.excl     0 : { *(.stab.excl) }
  .stab.exclstr  0 : { *(.stab.exclstr) }
  .stab.index    0 : { *(.stab.index) }
  .stab.indexstr 0 : { *(.stab.indexstr) }
  .comment       0 : { *(.comment) }
  .gnu.build.attributes : { *(.gnu.build.attributes .gnu.build.attributes.*) }
  /* DWARF debug sections.  */
  .debug          0 : { *(.debug) }
  .line           0 : { *(.line) }
  .debug_srcinfo  0 : { *(.debug_srcinfo) }
  .debug_sfnames  0 : { *(.debug_sfnames) }
  .debug_aranges  0 : { *(.debug_aranges) }
  .debug_pubnames 0 : { *(.debug_pubnames) }
  .debug_info     0 : { *(.debug_info .gnu.linkonce.wi.*) }
  .debug_abbrev  0 : { *(.debug_abbrev) }
  .debug_line     0 : { *(.debug_line .debug_line.* .debug_line_end) }
  .debug_frame    0 : { *(.debug_frame) }
  .debug_str      0 : { *(.debug_str) }
  .debug_loc      0 : { *(.debug_loc) }
  .debug_macinfo  0 : { *(.debug_macinfo) }
  .debug_weaknames 0 : { *(.debug_weaknames) }
  .debug_funcnames 0 : { *(.debug_funcnames) }
  .debug_typenames 0 : { *(.debug_typenames) }
  .debug_varnames  0 : { *(.debug_varnames) }
  .debug_pubtypes 0 : { *(.debug_pubtypes) }
  .debug_macro    0 : { *(.debug_macro) }
  .debug_addr     0 : { *(.debug_addr) }
  .gnu.attributes 0 : { KEEP (*(.gnu.attributes)) }
  /DISCARD/ : { *(.note.GNU-stack) *(.gnu_debuglink) *(.gnu.lto_*)
                *(.eh_frame) *(.eh_frame_hdr) *(.eh_frame_entry) *(.eh_frame.*) }
}

_globals_start = ALIGN(_data_start, 4);
_globals_end = ALIGN(_end, 4);
_heapStart = _end;
_heapEnd = _stack_end;
_stack_top = __stack;
