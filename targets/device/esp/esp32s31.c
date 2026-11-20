/*
 * SPDX-FileCopyrightText: 2021-2024 Espressif Systems (Shanghai) CO LTD
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#ifndef _SOC_USB_SERIAL_JTAG_STRUCT_H_
#define _SOC_USB_SERIAL_JTAG_STRUCT_H_

#ifdef __cplusplus
extern "C" {
#endif
#include "stdint.h"

// note(zzy): Structure definition from ESP-IDF:
// components/soc/esp32s31/register/soc/usb_serial_jtag_struct.h
// The ESP32-S31 register layout matches the ESP32-C3 one: USB_SERIAL_JTAG
// sits at a different base address (bound by targets/esp32s31.memory.ld),
// but EP1, EP1_CONF and the status/interrupt bit positions are identical.
typedef volatile struct usb_serial_jtag_dev_s {
    union {
        struct {
            uint32_t rdwr_byte                     :    32;  /*Although only low 8-bits is valid, but change it to 32bits to avoid there's no read/modify/write behaviour*/ /*Write and read byte data to/from UART Tx/Rx FIFO through this field. When USB_SERIAL_JTAG_SERIAL_IN_EMPTY_INT is set  then user can write data (up to 64 bytes) into UART Tx FIFO. When USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT is set, user can check USB_SERIAL_JTAG_OUT_EP1_WR_ADDR and USB_SERIAL_JTAG_OUT_EP0_RD_ADDR to know how many data is received, then read that amount of data from UART Rx FIFO. */
        };
        uint32_t val;
    } ep1;
    union {
        struct {
            uint32_t wr_done : 1; /*Set this bit to indicate writing byte data to UART Tx FIFO is done.  This bit then
                                     stays 0 until data in UART Tx FIFO is read by the USB Host.*/
            uint32_t serial_in_ep_data_free
                : 1; /*1'b1: Indicate UART Tx FIFO is not full and data can be written into in. After writing
                        USB_SERIAL_JTAG_WR_DONE, this will be 1’b0 until the data is sent to the USB Host.*/
            uint32_t serial_out_ep_data_avail : 1; /*1'b1: Indicate there is data in UART Rx FIFO.*/
            uint32_t reserved3 : 29;               /*reserved*/
        };
        uint32_t val;
    } ep1_conf;
    union {
        struct {
            uint32_t jtag_in_flush_int_raw : 1; /*The raw interrupt bit turns to high level when a flush command is
                                                   received for IN endpoint 2 of JTAG.*/
            uint32_t sof_int_raw : 1; /*The raw interrupt bit turns to high level when a SOF frame is received.*/
            uint32_t serial_out_recv_pkt_int_raw : 1; /*The raw interrupt bit turns to high level when the Serial Port
                                                         OUT Endpoint received one packet.*/
            uint32_t serial_in_empty_int_raw
                : 1; /*The raw interrupt bit turns to high level when the Serial Port IN Endpoint is empty.*/
            uint32_t pid_err_int_raw : 1;  /*The raw interrupt bit turns to high level when a PID error is detected.*/
            uint32_t crc5_err_int_raw : 1; /*The raw interrupt bit turns to high level when a CRC5 error is detected.*/
            uint32_t crc16_err_int_raw
                : 1; /*The raw interrupt bit turns to high level when a CRC16 error is detected.*/
            uint32_t stuff_err_int_raw
                : 1; /*The raw interrupt bit turns to high level when a bit stuffing error is detected.*/
            uint32_t in_token_rec_in_ep1_int_raw
                : 1; /*The raw interrupt bit turns to high level when an IN token for IN endpoint 1 is received.*/
            uint32_t usb_bus_reset_int_raw
                : 1; /*The raw interrupt bit turns to high level when a USB bus reset is detected.*/
            uint32_t out_ep1_zero_payload_int_raw : 1; /*The raw interrupt bit turns to high level when OUT endpoint 1
                                                          received packet with zero payload.*/
            uint32_t out_ep2_zero_payload_int_raw : 1; /*The raw interrupt bit turns to high level when OUT endpoint 2
                                                          received packet with zero payload.*/
            uint32_t reserved12 : 20;                  /*reserved*/
        };
        uint32_t val;
    } int_raw;
    union {
        struct {
            uint32_t jtag_in_flush_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_JTAG_IN_FLUSH_INT interrupt.*/
            uint32_t sof_int_st : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_SOF_INT interrupt.*/
            uint32_t serial_out_recv_pkt_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT interrupt.*/
            uint32_t serial_in_empty_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_SERIAL_IN_EMPTY_INT interrupt.*/
            uint32_t pid_err_int_st : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_PID_ERR_INT interrupt.*/
            uint32_t crc5_err_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_CRC5_ERR_INT interrupt.*/
            uint32_t crc16_err_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_CRC16_ERR_INT interrupt.*/
            uint32_t stuff_err_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_STUFF_ERR_INT interrupt.*/
            uint32_t in_token_rec_in_ep1_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_IN_TOKEN_REC_IN_EP1_INT interrupt.*/
            uint32_t usb_bus_reset_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_USB_BUS_RESET_INT interrupt.*/
            uint32_t out_ep1_zero_payload_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_OUT_EP1_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t out_ep2_zero_payload_int_st
                : 1; /*The raw interrupt status bit for the USB_SERIAL_JTAG_OUT_EP2_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t reserved12 : 20; /*reserved*/
        };
        uint32_t val;
    } int_st;
    union {
        struct {
            uint32_t jtag_in_flush_int_ena
                : 1;                  /*The interrupt enable bit for the USB_SERIAL_JTAG_JTAG_IN_FLUSH_INT interrupt.*/
            uint32_t sof_int_ena : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_SOF_INT interrupt.*/
            uint32_t serial_out_recv_pkt_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT interrupt.*/
            uint32_t serial_in_empty_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_SERIAL_IN_EMPTY_INT interrupt.*/
            uint32_t pid_err_int_ena : 1;  /*The interrupt enable bit for the USB_SERIAL_JTAG_PID_ERR_INT interrupt.*/
            uint32_t crc5_err_int_ena : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_CRC5_ERR_INT interrupt.*/
            uint32_t crc16_err_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_CRC16_ERR_INT interrupt.*/
            uint32_t stuff_err_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_STUFF_ERR_INT interrupt.*/
            uint32_t in_token_rec_in_ep1_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_IN_TOKEN_REC_IN_EP1_INT interrupt.*/
            uint32_t usb_bus_reset_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_USB_BUS_RESET_INT interrupt.*/
            uint32_t out_ep1_zero_payload_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_OUT_EP1_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t out_ep2_zero_payload_int_ena
                : 1; /*The interrupt enable bit for the USB_SERIAL_JTAG_OUT_EP2_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t reserved12 : 20; /*reserved*/
        };
        uint32_t val;
    } int_ena;
    union {
        struct {
            uint32_t jtag_in_flush_int_clr
                : 1;                  /*Set this bit to clear the USB_SERIAL_JTAG_JTAG_IN_FLUSH_INT interrupt.*/
            uint32_t sof_int_clr : 1; /*Set this bit to clear the USB_SERIAL_JTAG_JTAG_SOF_INT interrupt.*/
            uint32_t serial_out_recv_pkt_int_clr
                : 1; /*Set this bit to clear the USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT interrupt.*/
            uint32_t serial_in_empty_int_clr
                : 1;                        /*Set this bit to clear the USB_SERIAL_JTAG_SERIAL_IN_EMPTY_INT interrupt.*/
            uint32_t pid_err_int_clr : 1;   /*Set this bit to clear the USB_SERIAL_JTAG_PID_ERR_INT interrupt.*/
            uint32_t crc5_err_int_clr : 1;  /*Set this bit to clear the USB_SERIAL_JTAG_CRC5_ERR_INT interrupt.*/
            uint32_t crc16_err_int_clr : 1; /*Set this bit to clear the USB_SERIAL_JTAG_CRC16_ERR_INT interrupt.*/
            uint32_t stuff_err_int_clr : 1; /*Set this bit to clear the USB_SERIAL_JTAG_STUFF_ERR_INT interrupt.*/
            uint32_t in_token_rec_in_ep1_int_clr
                : 1; /*Set this bit to clear the USB_SERIAL_JTAG_IN_TOKEN_IN_EP1_INT interrupt.*/
            uint32_t usb_bus_reset_int_clr
                : 1; /*Set this bit to clear the USB_SERIAL_JTAG_USB_BUS_RESET_INT interrupt.*/
            uint32_t out_ep1_zero_payload_int_clr
                : 1; /*Set this bit to clear the USB_SERIAL_JTAG_OUT_EP1_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t out_ep2_zero_payload_int_clr
                : 1;                  /*Set this bit to clear the USB_SERIAL_JTAG_OUT_EP2_ZERO_PAYLOAD_INT interrupt.*/
            uint32_t reserved12 : 20; /*reserved*/
        };
        uint32_t val;
    } int_clr;
    union {
        struct {
            uint32_t phy_sel : 1;             /*Select internal/external PHY. 1’b0: internal PHY, 1’b1: external PHY*/
            uint32_t exchg_pins_override : 1; /*Enable software control USB D+ D- exchange*/
            uint32_t exchg_pins : 1;          /*USB D+ D- exchange*/
            uint32_t vrefh : 2;               /*Control single-end input high threshold. 1.76V to 2V, step 80mV */
            uint32_t vrefl : 2;               /*Control single-end input low threshold. 0.8V to 1.04V, step 80mV*/
            uint32_t vref_override : 1;       /*Enable software control input  threshold*/
            uint32_t pad_pull_override : 1;   /*Enable software control USB D+ D- pullup pulldown*/
            uint32_t dp_pullup : 1;           /*Control USB D+ pull up.*/
            uint32_t dp_pulldown : 1;         /*Control USB D+ pull down.*/
            uint32_t dm_pullup : 1;           /*Control USB D- pull up.*/
            uint32_t dm_pulldown : 1;         /*Control USB D- pull down.*/
            uint32_t pullup_value : 1;        /*Control pull up value.*/
            uint32_t usb_pad_enable : 1;      /*Enable USB pad function.*/
            uint32_t reserved15 : 17;
        };
        uint32_t val;
    } conf0;
    union {
        struct {
            uint32_t test_enable : 1; /*Enable test of the USB pad*/
            uint32_t test_usb_oe : 1; /*USB pad oen in test*/
            uint32_t test_tx_dp : 1;  /*USB D+ tx value in test*/
            uint32_t test_tx_dm : 1;  /*USB D- tx value in test*/
            uint32_t reserved4 : 28;
        };
        uint32_t val;
    } test;
    union {
        struct {
            uint32_t in_fifo_cnt : 2;    /*JTAG in fifo counter.*/
            uint32_t in_fifo_empty : 1;  /*1: JTAG in fifo is empty.*/
            uint32_t in_fifo_full : 1;   /*1: JTAG in fifo is full.*/
            uint32_t out_fifo_cnt : 2;   /*JTAT out fifo counter.*/
            uint32_t out_fifo_empty : 1; /*1: JTAG out fifo is empty.*/
            uint32_t out_fifo_full : 1;  /*1: JTAG out fifo is full.*/
            uint32_t in_fifo_reset : 1;  /*Write 1 to reset JTAG in fifo.*/
            uint32_t out_fifo_reset : 1; /*Write 1 to reset JTAG out fifo.*/
            uint32_t reserved10 : 22;
        };
        uint32_t val;
    } jfifo_st;
    union {
        struct {
            uint32_t sof_frame_index : 11; /*Frame index of received SOF frame.*/
            uint32_t reserved11 : 21;
        };
        uint32_t val;
    } fram_num;
    union {
        struct {
            uint32_t in_ep0_state : 2;   /*State of IN Endpoint 0.*/
            uint32_t in_ep0_wr_addr : 7; /*Write data address of IN endpoint 0.*/
            uint32_t in_ep0_rd_addr : 7; /*Read data address of IN endpoint 0.*/
            uint32_t reserved16 : 16;    /*reserved*/
        };
        uint32_t val;
    } in_ep0_st;
    union {
        struct {
            uint32_t in_ep1_state : 2;   /*State of IN Endpoint 1.*/
            uint32_t in_ep1_wr_addr : 7; /*Write data address of IN endpoint 1.*/
            uint32_t in_ep1_rd_addr : 7; /*Read data address of IN endpoint 1.*/
            uint32_t reserved16 : 16;    /*reserved*/
        };
        uint32_t val;
    } in_ep1_st;
    union {
        struct {
            uint32_t in_ep2_state : 2;   /*State of IN Endpoint 2.*/
            uint32_t in_ep2_wr_addr : 7; /*Write data address of IN endpoint 2.*/
            uint32_t in_ep2_rd_addr : 7; /*Read data address of IN endpoint 2.*/
            uint32_t reserved16 : 16;    /*reserved*/
        };
        uint32_t val;
    } in_ep2_st;
    union {
        struct {
            uint32_t in_ep3_state : 2;   /*State of IN Endpoint 3.*/
            uint32_t in_ep3_wr_addr : 7; /*Write data address of IN endpoint 3.*/
            uint32_t in_ep3_rd_addr : 7; /*Read data address of IN endpoint 3.*/
            uint32_t reserved16 : 16;    /*reserved*/
        };
        uint32_t val;
    } in_ep3_st;
    union {
        struct {
            uint32_t out_ep0_state : 2; /*State of OUT Endpoint 0.*/
            uint32_t out_ep0_wr_addr
                : 7; /*Write data address of OUT endpoint 0. When USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT is detected.
                        there are USB_SERIAL_JTAG_OUT_EP0_WR_ADDR-2 bytes data in OUT EP0. */
            uint32_t out_ep0_rd_addr : 7; /*Read data address of OUT endpoint 0.*/
            uint32_t reserved16 : 16;     /*reserved*/
        };
        uint32_t val;
    } out_ep0_st;
    union {
        struct {
            uint32_t out_ep1_state : 2; /*State of OUT Endpoint 1.*/
            uint32_t out_ep1_wr_addr
                : 7; /*Write data address of OUT endpoint 1. When USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT is detected.
                        there are USB_SERIAL_JTAG_OUT_EP1_WR_ADDR-2 bytes data in OUT EP1.*/
            uint32_t out_ep1_rd_addr : 7;      /*Read data address of OUT endpoint 1.*/
            uint32_t out_ep1_rec_data_cnt : 7; /*Data count in OUT endpoint 1 when one packet is received.*/
            uint32_t reserved23 : 9;           /*reserved*/
        };
        uint32_t val;
    } out_ep1_st;
    union {
        struct {
            uint32_t out_ep2_state : 2; /*State of OUT Endpoint 2.*/
            uint32_t out_ep2_wr_addr
                : 7; /*Write data address of OUT endpoint 2. When USB_SERIAL_JTAG_SERIAL_OUT_RECV_PKT_INT is detected.
                        there are USB_SERIAL_JTAG_OUT_EP2_WR_ADDR-2 bytes data in OUT EP2.*/
            uint32_t out_ep2_rd_addr : 7; /*Read data address of OUT endpoint 2.*/
            uint32_t reserved16 : 16;     /*reserved*/
        };
        uint32_t val;
    } out_ep2_st;
    union {
        struct {
            uint32_t clk_en
                : 1; /*1'h1: Force clock on for register. 1'h0: Support clock only when application writes registers.*/
            uint32_t reserved1 : 31;
        };
        uint32_t val;
    } misc_conf;
    union {
        struct {
            uint32_t usb_mem_pd : 1;     /*1: power down usb memory.*/
            uint32_t usb_mem_clk_en : 1; /*1: Force clock on for usb memory.*/
            uint32_t reserved2 : 30;
        };
        uint32_t val;
    } mem_conf;
    uint32_t reserved_4c;
    uint32_t reserved_50;
    uint32_t reserved_54;
    uint32_t reserved_58;
    uint32_t reserved_5c;
    uint32_t reserved_60;
    uint32_t reserved_64;
    uint32_t reserved_68;
    uint32_t reserved_6c;
    uint32_t reserved_70;
    uint32_t reserved_74;
    uint32_t reserved_78;
    uint32_t reserved_7c;
    uint32_t date;
} usb_serial_jtag_dev_t;
extern usb_serial_jtag_dev_t USB_SERIAL_JTAG;

// note(zzy): Functions below are based on ESP-IDF HAL layer:
// https://github.com/espressif/esp-idf/blob/release/v6.0/components/hal/esp32s31/include/hal/usb_serial_jtag_ll.h

/**
 * @brief  Flushes the TX buffer, that is, make it available for the
 *         host to pick up.
 *
 * @note  When fifo is full (with 64 byte), HW will flush the buffer automatically,
 *        if this function is called directly after, this effectively turns into a
 *        no-op. Because a 64-byte packet will be interpreted as a not-complete USB
 *        transaction, you need to transfer either more data or a zero-length packet
 *        for the data to actually end up at the program listening to the CDC-ACM
 *        serial port. To send a zero-length packet, call
 *        usb_serial_jtag_ll_txfifo_flush() again when
 *        usb_serial_jtag_ll_txfifo_writable() returns true.
 *
 * @return na
 */
static inline void usb_serial_jtag_ll_txfifo_flush(void) { USB_SERIAL_JTAG.ep1_conf.wr_done = 1; }

/**
 * @brief  Returns 1 if the USB_SERIAL_JTAG txfifo has room.
 *
 * @return 0 if no data available, 1 if data available
 */
static inline int usb_serial_jtag_ll_txfifo_writable(void) { return USB_SERIAL_JTAG.ep1_conf.serial_in_ep_data_free; }

/**
 * @brief  Write byte to the USB_SERIAL_JTAG txfifo. Only writes bytes as long / if there
 *         is room in the buffer.
 *
 * @param  buf The data buffer.
 * @param  wr_len The data length needs to be written.
 *
 * @return Amount of bytes actually written. May be less than wr_len.
 */
static inline int usb_serial_jtag_ll_write_txfifo(const uint8_t *buf, uint32_t wr_len) {
    int i;
    for (i = 0; i < (int)wr_len; i++) {
        if (!USB_SERIAL_JTAG.ep1_conf.serial_in_ep_data_free)
            break;
        USB_SERIAL_JTAG.ep1.rdwr_byte = buf[i];
    }
    return i;
}

// ============================================================================
// Boot-time HP watchdog shutdown
// ============================================================================
// The ROM bootloader arms the RWDT (RTC watchdog) in flashboot mode with an
// ~8 s timeout and arms MWDT0 flashboot protection as well. Bare-metal images
// boot straight from the ROM with no second-stage bootloader, so nothing ever
// reconfigures them and any task that legitimately runs longer than ~8 s gets
// reset mid-run. Mirror what the ESP-IDF second-stage bootloader does in
// bootloader_config_wdt(): unlock with the WDT key, clear the enable and
// flashboot bits, feed, and re-lock.
//
// Register addresses and bit positions follow ESP-IDF:
//   components/soc/<chip>/register/soc/{rtc_cntl,timer_group,rtc_wdt}_reg.h
//   components/esp_hal_wdt/<chip>/include/hal/{rwdt,mwdt}_ll.h
// The constructor runs from crt0's __libc_init_array, before main and before
// the Go runtime starts scheduling, which is the earliest hook the device C
// file can install.

#define ESP_WDT_WKEY 0x50D83AA1u

#define ESP_WDT_EN (1u << 31)
#define ESP_WDT_CONF_UPDATE (1u << 22)
#define ESP_RWDT_FLASHBOOT_EN (1u << 12)
#define ESP_MWDT_FLASHBOOT_EN (1u << 14)

// RWDT is a dedicated LP block; MWDT0 in timer group 0.
#define ESP_RWDT_CONFIG0_REG (0x20801000u + 0x00)
#define ESP_RWDT_FEED_REG (0x20801000u + 0x14)
#define ESP_RWDT_WPROTECT_REG (0x20801000u + 0x18)
#define ESP_MWDT_CONFIG0_REG (0x20580000u + 0x48)
#define ESP_MWDT_FEED_REG (0x20580000u + 0x60)
#define ESP_MWDT_WPROTECT_REG (0x20580000u + 0x64)

// LP_CLKRST_HPCORE0_RESET_CTRL.HPCORE0_SW_RESET: software reset of the
// boot core (S31 has no RTC_CNTL_OPTIONS0-style system reset register).
#define ESP_SOFT_RESET_REG (0x20701000u + 0x3C)
#define ESP_SOFT_RESET_BIT (1u << 20)

// USB_SERIAL_JTAG base: ep1_conf.serial_in_ep_data_free lives at +0x4 bit 1.
#define ESP_USB_SJ_BASE 0x20391000u

static void esp_disable_hp_wdt(void) {
  volatile uint32_t *rwdt_config0 = (volatile uint32_t *)ESP_RWDT_CONFIG0_REG;
  volatile uint32_t *rwdt_feed = (volatile uint32_t *)ESP_RWDT_FEED_REG;
  volatile uint32_t *rwdt_wprotect = (volatile uint32_t *)ESP_RWDT_WPROTECT_REG;
  volatile uint32_t *mwdt_config0 = (volatile uint32_t *)ESP_MWDT_CONFIG0_REG;
  volatile uint32_t *mwdt_feed = (volatile uint32_t *)ESP_MWDT_FEED_REG;
  volatile uint32_t *mwdt_wprotect = (volatile uint32_t *)ESP_MWDT_WPROTECT_REG;

  // RWDT (RTC watchdog): disarm and drop flashboot mode.
  *rwdt_wprotect = ESP_WDT_WKEY;
  *rwdt_config0 &= ~(ESP_WDT_EN | ESP_RWDT_FLASHBOOT_EN);
  *rwdt_feed = 1;
  *rwdt_wprotect = 0;

  // MWDT0: disarm and drop flashboot mode. The config registers latch
  // asynchronously on this silicon, so request an update explicitly.
  *mwdt_wprotect = ESP_WDT_WKEY;
  *mwdt_config0 = (*mwdt_config0 & ~(ESP_WDT_EN | ESP_MWDT_FLASHBOOT_EN)) |
                  ESP_WDT_CONF_UPDATE;
  *mwdt_feed = 1;
  *mwdt_wprotect = 0;
}

// ============================================================================
// System Call Implementation for Bare-Metal ESP32-S31
// ============================================================================
// note(zzy): _write() implementation is inspired by ESP-IDF VFS layer:
// https://github.com/espressif/esp-idf/blob/release/v6.0/components/esp_driver_usb_serial_jtag/src/usb_serial_jtag_vfs.c

#include <errno.h>
#include <sys/types.h>

// The EP1 TX FIFO holds one 64-byte packet. A full 64-byte packet is treated
// as an incomplete USB transaction by CDC-ACM hosts (it only completes when a
// further packet or a zero-length packet follows), so chunks are capped at 63
// bytes and every chunk ends with an explicit wr_done flush.
#define USB_SJ_CHUNK_MAX 63

// After wr_done, the writable flag only returns once the host has picked the
// packet up, which takes at most a few USB frames (~1 ms each). A missing host
// (no terminal attached) never returns the flag, so every wait is bounded and
// degrades to dropped output instead of hanging the program.
#define USB_SJ_FLUSH_SPIN 500000

/**
 * @brief Push buf into the TX FIFO, handing full packets to the host as needed.
 */
static void usb_serial_jtag_push(const uint8_t *buf, size_t len) {
    // Once a timeout has proved that no host drains the FIFO, skip future
    // writes instead of re-running the full spin for every following chunk.
    static int host_lost;
    if (host_lost) {
        return;
    }
    size_t off = 0;
    while (off < len) {
        off += usb_serial_jtag_ll_write_txfifo(buf + off, len - off);
        if (off < len) {
            usb_serial_jtag_ll_txfifo_flush();
            int spin = USB_SJ_FLUSH_SPIN;
            while (!usb_serial_jtag_ll_txfifo_writable() && spin-- > 0) {
                // Busy wait until the host drains the packet
            }
            if (spin <= 0) {
                host_lost = 1;
                return; // no host is reading; drop the remainder and future output
            }
        }
    }
}

/**
 * @brief System call: write to file descriptor
 *
 * For bare-metal ESP32-S31, only stdout (1) and stderr (2) are supported
 * via USB Serial JTAG. Other file descriptors return error.
 *
 * Output is batched into stack chunks of up to 63 bytes instead of writing
 * byte by byte: each chunk needs one wr_done handshake with the host, so
 * batching removes most of that per-byte overhead. Chunks flush on newline
 * for interactive output, and _exit flushes anything left over.
 *
 * @param fd File descriptor (only 1 and 2 supported)
 * @param buf Data buffer to write
 * @param count Number of bytes to write
 * @return Number of bytes written, or -1 on error
 */
ssize_t _write(int fd, const void *buf, size_t count) {
    if (fd != 1 && fd != 2) {
        errno = EBADF;
        return -1;
    }

    const uint8_t *data = (const uint8_t *)buf;
    uint8_t chunk[USB_SJ_CHUNK_MAX];
    size_t n = 0;

    for (size_t i = 0; i < count; i++) {
        uint8_t c = data[i];
        size_t need = (c == '\n') ? 2 : 1; // LF expands to CRLF

        if (n + need > sizeof(chunk)) {
            usb_serial_jtag_push(chunk, n);
            usb_serial_jtag_ll_txfifo_flush();
            n = 0;
        }
        if (c == '\n') {
            chunk[n++] = '\r';
        }
        chunk[n++] = c;

        if (c == '\n') {
            usb_serial_jtag_push(chunk, n);
            usb_serial_jtag_ll_txfifo_flush();
            n = 0;
        }
    }

    if (n > 0) {
        usb_serial_jtag_push(chunk, n);
        usb_serial_jtag_ll_txfifo_flush();
    }

    return (ssize_t)count;
}

// ============================================================================
// Exit path fallback for real hardware
// ============================================================================
// _exit comes from newlib's libsemihost archive (RISC-V semihosting
// SYS_EXIT: a0=0x18 SYS_EXIT, a1=0x20026 ADP_Stopped_ApplicationExit, with
// the uncompressed slli/ebreak/srai trap sequence). esp-emu and QEMU
// (-semihosting) intercept that EBREAK and terminate cleanly.
//
// On real hardware there is no semihosting host: the EBREAK raises a
// breakpoint exception (mcause=3). This device file installs a chained trap
// at boot that recognizes the semihosting sequence by the magic slli in
// front of the faulting EBREAK and performs an RTC soft reset, instead of
// falling into the ROM exception handler (which spins until a watchdog
// reset). Every other exception/interrupt chains to the handler that was
// in mtvec at boot, so ordinary traps keep their existing behavior.
//
// The link order puts the libc archives before the extra-file objects, so a
// strong _exit here would collide with the archive's definition; the trap
// below is the device-side complement to the archive's exit.

// The semihosting pair opens with the uncompressed 0x01f01013 (slli zero,
// zero, 0x1f). Its high halfword 0x01f0 identifies the sequence: the trap
// loads it with a 2-byte-aligned lhu because RISC-V only guarantees 2-byte
// instruction alignment, so the sequence may start at an address that is not
// 4-byte aligned when preceded by compressed instructions.
#define ESP_SEMIHOST_SLII_HIGH 0x01f0u

// Bounded host-drain window in the exit trap (same rationale as
// USB_SJ_FLUSH_SPIN: a few USB frames, and it never runs under esp-emu/QEMU
// because the semihosting host terminates the process at the EBREAK).
#define ESP_TRAP_DRAIN_SPIN 1500000

// Previous (ROM) mtvec, saved at boot for chaining. `used` keeps the
// definition in the object: the only reader is the naked trap's assembly,
// which the optimizer cannot see.
__attribute__((used)) static uint32_t esp_prev_mtvec;

__attribute__((naked, aligned(4))) static void esp_trap_handler(void) {
    __asm__ volatile(
        "addi sp, sp, -24\n"
        "sw a0, 0(sp)\n"
        "sw a1, 4(sp)\n"
        "sw a2, 8(sp)\n"
        "sw t0, 12(sp)\n"
        "sw t1, 16(sp)\n"
        "csrr a0, mcause\n"
        "bltz a0, 2f\n" // interrupts chain to the previous handler
        "li a1, 3\n" // EXCAUSE_BREAKPOINT
        "bne a0, a1, 2f\n"
        "csrr a0, mepc\n"
        "lhu a1, -2(a0)\n" // high half of the slli in front of the EBREAK
        "li a2, %0\n"
        "bne a1, a2, 2f\n" // not our semihosting exit: chain
        // Semihosting exit without a semihosting host. Give the USB host a
        // bounded window to drain the last flushed packet (ep1_conf
        // serial_in_ep_data_free), then soft reset.
        "li a0, %1\n"
        "li a3, %2\n"
        "3: lw a1, 4(a0)\n"
        "andi a1, a1, 2\n" // serial_in_ep_data_free
        "bnez a1, 4f\n"
        "addi a3, a3, -1\n"
        "bnez a3, 3b\n"
        "4: li a0, %3\n"
        "lw a1, 0(a0)\n"
        "li a2, %4\n"
        "or a1, a1, a2\n"
        "sw a1, 0(a0)\n"
        "1: wfi\n" // reset takes effect; park if it somehow does not
        "j 1b\n"
        "2: la t0, esp_prev_mtvec\n"
        "lw t0, 0(t0)\n"
        "sw t0, 20(sp)\n" // park the chain target above the saved set
        "lw a0, 0(sp)\n"
        "lw a1, 4(sp)\n"
        "lw a2, 8(sp)\n"
        "lw t0, 12(sp)\n"
        "lw t1, 16(sp)\n"
        "lw t0, 20(sp)\n" // reload the chain target last: t0 jumps
        "addi sp, sp, 24\n"
        "jr t0\n"
        :
        : "i"(ESP_SEMIHOST_SLII_HIGH), "i"(ESP_USB_SJ_BASE), "i"(ESP_TRAP_DRAIN_SPIN),
          "i"(ESP_SOFT_RESET_REG), "i"(ESP_SOFT_RESET_BIT));
}

// Boot hook: disarm the ROM-armed watchdogs and install the exit trap.
// crt0 runs .init_array constructors before main, so this happens before
// any Go code - the earliest hook the device C file can install.
__attribute__((constructor)) static void esp_boot_early(void) {
    esp_disable_hp_wdt();
    uint32_t prev;
    __asm__ volatile("csrr %0, mtvec" : "=r"(prev));
    // Chaining assumes the boot-time mtvec is a direct-mode handler (the ROM
    // installs one). If a vectored handler ever ran before this constructor,
    // the chained jump would lose its per-cause offset dispatch.
    esp_prev_mtvec = prev & ~3u;
    __asm__ volatile("csrw mtvec, %0" : : "r"(esp_trap_handler));
}

#ifdef __cplusplus
}
#endif

#endif /*_SOC_USB_SERIAL_JTAG_STRUCT_H_ */
