/* AN385 exposes separate 4 MiB ZBT SRAM banks for code and data. */
MEMORY
{
    FLASH_TEXT (rx) : ORIGIN = 0x00000000, LENGTH = 4M
    RAM (xrw)       : ORIGIN = 0x20000000, LENGTH = 4M
}

_stack_size = 16K;
INCLUDE "targets/arm.ld"

_heapStart = _heap_start;
_heapEnd = _heap_end;
_stack_end = ADDR(.stack);
