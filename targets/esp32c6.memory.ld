# See "System Structure and Address Mapping" figure at
# https://www.espressif.com/sites/default/files/documentation/esp32-c6_technical_reference_manual_en.pdf

SRAM_START_ADDRESS = 0x40800000;
SRAM_LEN = 0x80000;

CPU_FREQUENCY = 40000000;
UART0_BAUD = 115200;

UART0_CLKDIV_REG = 0x60000014;
UART0_CLKDIV_VAL = CPU_FREQUENCY / UART0_BAUD;
UART0_STATUS = 0x6000001C;
UART0_TX_ADDR = 0x60000000;

PROVIDE ( USB_SERIAL_JTAG = 0x6000F000 );

MEMORY
{
  sram_seg  (RWX) : org = SRAM_START_ADDRESS, len = SRAM_LEN
}

REGION_ALIAS("iram_seg", sram_seg);
REGION_ALIAS("dram_seg", sram_seg);
INCLUDE "targets/esp32-riscv.app.elf.ld";
