# Memory layout for the ESP32-S31 flash (XIP) target, booted through the
# ESP-IDF second-stage bootloader.
#
# The bootloader maps the app's flash-resident segments into the 64 KiB-page
# XIP window at 0x40000000 and copies the RAM segments into SRAM, then jumps
# to the app entry. Code and read-only data therefore live in flash_seg, and
# the application keeps the SRAM below the ROM stack top for its writable
# data, heap and stack.

SRAM_START_ADDRESS = 0x2F000000;
SRAM_LEN = 0x2F07CFB0 - 0x2F000000;
FLASH_BASE_ADDRESS = 0x40000000;
FLASH_LEN = 0x400000; /* 4 MiB flash image */

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
  flash_seg (RX)  : org = FLASH_BASE_ADDRESS, len = FLASH_LEN
}

REGION_ALIAS("iram_seg", flash_seg);
REGION_ALIAS("dram_seg", sram_seg);
INCLUDE "targets/esp32s31-flash.app.elf.ld";
