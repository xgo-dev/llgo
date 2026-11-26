/*
 * SPDX-FileCopyrightText: 2025 Espressif Systems (Shanghai) CO LTD
 *
 * SPDX-License-Identifier: Apache-2.0
 */

/* The LP core program of the llgo lpcore acceptance demo: a mailbox echo
 * server. It mirrors the HP side's wire format (goapp/main.go: one 32-bit
 * word per message, lpcore.Pack's (cmd, arg) halves): every ping
 * (cmd 1, arg) gets a pong (cmd 2, arg XOR 0xA55A) - a reply value only the
 * LP core's own code can produce, which is what the HP probe's golden
 * checks.
 *
 * Built by the IDF ULP build system (ulp_embed_binary in main/CMakeLists.txt,
 * per the IDF lp_mailbox example) into lp_core_main.bin, which the HP
 * firmware links and loads into LP RAM at probe time. The S31 LP core has no
 * ROM: the HP side writes the boot address before the core starts
 * (ulp_lp_core_run does this), so the program's reset vector lands where the
 * ULP build's linker script puts it.
 *
 * All mailbox operations here use short or infinite waits: this core has
 * nothing else to do, and the HP side drives the pacing with its own bounded
 * poll. No prints: lp_core_printf would need the LP UART setup, and the echo
 * verdict belongs to the HP side's golden anyway. Progress instead goes into
 * the four exported markers below - plain LP RAM words the HP probe reads
 * through the generated lp_core_main.h (ulp_ prefixed externs), which is how
 * the demo (and any hardware bring-up) tells apart "the LP core never ran"
 * from "the mailbox path broke".
 */

#include <stdint.h>

#include "esp_err.h"
#include "ulp_lp_core_mailbox.h"

#define LG_CMD_PING  1
#define LG_CMD_PONG  2
#define LG_ARG_MASK  0xFFFFu
#define LG_REPLY_XOR 0xA55Au

/* Progress markers, exported to the HP side by the ULP build (lp_core_main.h).
 * All four land in LP RAM's zero-initialized .bss, so their pre-run state is
 * unambiguous: zero means "not reached". */
uint32_t lp_echo_boot;   /* 0xC0DEC0DE once main() entered                */
uint32_t lp_echo_mbinit; /* lp_core_mailbox_init's esp_err (0 = ok)       */
uint32_t lp_echo_poll;   /* the entry poll's esp_err (0 = word was there,
                          * 7 = ESP_ERR_TIMEOUT, i.e. mailbox empty)      */
uint32_t lp_echo_count;  /* blocking receives completed                   */

static lp_mailbox_t s_mailbox;

int main(void)
{
    lp_echo_boot = 0xC0DEC0DEu;

    lp_echo_mbinit = (uint32_t)lp_core_mailbox_init(&s_mailbox, NULL);
    if (lp_echo_mbinit != ESP_OK) {
        /* No mailbox means no protocol: park (the HP side's marker readout
         * reports the failure) rather than spin on a dead path. */
        while (1) {
        }
    }

    /* The entry poll: one zero-tick receive before any HP traffic. It
     * proves the mailbox's read path answers from the LP side (the result
     * is ESP_ERR_TIMEOUT on an empty mailbox), separating "the LP core
     * reads the controller" from "the wake/ACK path works". */
    lp_message_t msg = 0;
    lp_echo_poll = (uint32_t)lp_core_mailbox_receive(s_mailbox, &msg, 0);


    while (1) {
        if (lp_core_mailbox_receive(s_mailbox, &msg, -1) != ESP_OK) {
            continue;
        }
        lp_echo_count++;
        uint32_t cmd = ((uint32_t)msg >> 16) & LG_ARG_MASK;
        if (cmd != LG_CMD_PING) {
            continue;
        }
        uint32_t arg = (uint32_t)msg & LG_ARG_MASK;
        uint32_t reply = ((uint32_t)LG_CMD_PONG << 16) | ((arg ^ LG_REPLY_XOR) & LG_ARG_MASK);
        lp_core_mailbox_send(s_mailbox, (lp_message_t)reply, -1);
    }

    return 0; /* not reached */
}
