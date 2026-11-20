# Memory layout obtained from the ESP32-S31 datasheet and ESP-IDF soc headers.
# SRAM is 512 KiB mapped identically on the IRAM and DRAM buses (DIRAM); the
# mask ROM sits above it at 0x2F800000 and the flash windows start at
# 0x40000000. Only the internal SRAM is used here.

SRAM_START_ADDRESS = 0x2F000000;
SRAM_LEN = 0x80000;

CPU_FREQUENCY = 40000000;
UART0_BAUD = 115200;

UART0_CLKDIV_REG = 0x2038A014;
UART0_CLKDIV_VAL = CPU_FREQUENCY / UART0_BAUD;
UART0_STATUS = 0x2038A01C;
UART0_TX_ADDR = 0x2038A000;

PROVIDE ( USB_SERIAL_JTAG = 0x20391000 );

MEMORY
{
  sram_seg  (RWX) : org = SRAM_START_ADDRESS, len = SRAM_LEN
}

REGION_ALIAS("iram_seg", sram_seg);
REGION_ALIAS("dram_seg", sram_seg);
INCLUDE "targets/esp32-riscv.app.elf.ld";
