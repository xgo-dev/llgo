# Memory layout obtained from the ESP32-C3 technical reference manual
# ("System Structure and Address Mapping"):
# https://www.espressif.com/sites/default/files/documentation/esp32-c3_technical_reference_manual_en.pdf
#
# SRAM is 400 KiB. The instruction bus maps it at 0x4037C000 (with a 16 KiB
# ICACHE at the bottom) and the data bus maps the first 256 KiB at
# 0x3FC80000. The shared esp32-riscv.app.elf.ld therefore reserves a shadow
# gap inside DRAM equal to the whole IRAM footprint, which caps a program at
# roughly half of SRAM. That cap cannot hold a reflect-scale program.
#
# This layout pins .text/.rodata to the TOP of the instruction window
# instead and keeps DRAM below it. The two regions stay physically disjoint,
# so no shadow gap is needed and the program can use almost all of SRAM
# (text+rodata upper bound of 256 KiB plus 128 KiB of writable data).

ICACHE_SIZE = 0x4000;
# Skip possible ICACHE area
IRAM_START_ADDRESS = 0x4037C000 + ICACHE_SIZE;
# Writable sections live in the low SRAM the data bus can address; text and
# read-only data sit above them, inside the same physical SRAM.
DRAM_START_ADDRESS = 0x3FC80000;
DRAM_LEN = 0x20000;
IRAM_TEXT_ORG = IRAM_START_ADDRESS + DRAM_LEN;
IRAM_LEN = 0x64000 - ICACHE_SIZE - DRAM_LEN;

CPU_FREQUENCY = 20000000;
UART0_BAUD = 115200;

UART0_CLKDIV_REG = 0x60000014;
UART0_CLKDIV_VAL = CPU_FREQUENCY / UART0_BAUD;
UART0_STATUS = 0x6000001C;
UART0_TX_ADDR = 0x60000000;

PROVIDE ( USB_SERIAL_JTAG = 0x60043000 );

MEMORY
{
  iram_seg  (RX) : org = IRAM_TEXT_ORG, len = IRAM_LEN
  dram_seg  (RW) : org = DRAM_START_ADDRESS, len = DRAM_LEN
}

INCLUDE "targets/esp32c3.app.elf.ld";
