/*
 * Early panic diagnostics for the ESP RISC-V baremetal targets
 * (esp32c3, esp32s31): a polled UART0 FIFO writer plus a chained trap
 * handler, both usable before any runtime service exists - no stdio, no
 * allocator, no locks, no interrupt setup.
 *
 * Why bare registers instead of the ROM UART helpers: the ROM helper
 * symbols (uart_tx_one_char and friends) live in the mask ROM image, which
 * the application ELF does not link against - a direct call would be an
 * undefined symbol at link time. The polled FIFO write needs nothing but
 * the peripheral addresses, which the target linker scripts already
 * provide: __uart0_tx_addr / __uart0_status are rodata slots holding the
 * per-chip UART0 register addresses (targets/esp32c3.memory.ld,
 * targets/esp32s31.memory.ld), so one code path serves both chips.
 *
 * Why UART0 instead of the USB-Serial/JTAG FIFO (USB_SERIAL_JTAG symbol):
 * the ROM boot log already goes out on UART0, so the port is up everywhere
 * the firmware runs, while the emulators route USB-Serial/JTAG output only
 * after the firmware has selected that console - an early panic lands
 * before that point and its EP1 output never shows (observed under
 * esp-emu). UART0 is the same line the normal runtime output can reach
 * through the stdio path on targets wired that way; on USB-console boards
 * the banner needs a wired UART0 pin, which is the documented trade-off.
 *
 * The register layout is identical on both chips: the FIFO data port at
 * offset 0, STATUS at 0x1C with TXFIFO_CNT in bits 25:16
 * (components/soc/{esp32c3,esp32s31}/register/soc/uart_reg.h). The ROM has
 * already programmed the divider for its boot log, so no clock or baud
 * setup happens here.
 *
 * Host-side symbolization of the reported PCs (the linked ELF keeps all
 * symbols, so bare PCs are enough):
 *
 *   llvm-symbolizer --obj=app.elf 0x2f0049d2
 *
 * See earlypanic_baremetal.go for the Go-side report formatting.
 */

#include <stdint.h>

/* Linker-provided rodata slots: their content is the UART0 register base
 * and the STATUS register address (targets/esp32*.memory.ld emits them as
 * LONG(...) literals next to __cpu_frequency). */
extern const uint8_t __uart0_tx_addr[];
extern const uint8_t __uart0_status[];

#define UART_TXFIFO_CNT(status) (((status) >> 16) & 0x3ffu)

/* The TX FIFO drains at the baud rate; a boot-length ROM log can still sit
 * in it when the banner prints, so every byte waits for room, bounded.
 * A missing UART (nobody configured the pins) just times out and drops the
 * rest instead of hanging the panic report. */
#define UART_FIFO_ROOM 120u
#define UART_SPIN 500000

void llgo_early_write(const char *s, uint32_t n) {
    static int host_lost;
    volatile uint8_t *fifo;
    volatile uint32_t *status;
    uint32_t off = 0;
    if (host_lost || n == 0) {
        return;
    }
    fifo = (volatile uint8_t *)(uint32_t) * (const uint32_t *)&__uart0_tx_addr;
    status = (volatile uint32_t *)(uint32_t) * (const uint32_t *)&__uart0_status;
    while (off < n) {
        int spin = UART_SPIN;
        while (UART_TXFIFO_CNT(*status) >= UART_FIFO_ROOM && spin-- > 0) {
        }
        if (spin <= 0) {
            host_lost = 1; /* no UART behind the registers; drop the rest */
            return;
        }
        /* Byte-sized store to the FIFO data port: the peripheral advances
         * its internal write pointer per byte store, and wider stores
         * garble the stream (observed under esp-emu). */
        *fifo = (uint8_t)s[off];
        off++;
    }
}

/* ----------------------------------------------------------------------------
 * Chained trap handler: reports unexpected exceptions before letting the
 * boot-time handler run.
 *
 * The device files install their trap (semihosting-exit detection plus
 * chaining to the boot mtvec) from a .init_array constructor, which always
 * runs before any Go code. llgo_early_trap_install is called from the Go
 * runtime's init, so the handler saved below is always the device file's
 * handler and the semihosting exit path keeps working: this handler only
 * adds a report in front of exceptions that would otherwise vanish into the
 * ROM's exception loop.
 *
 * mcause 3 (breakpoint) is the newlib libsemihost exit EBREAK and interrupts
 * (mcause bit 31) chain silently: they are control flow, not failures, and
 * reporting them would corrupt the exit path and spam timer interrupts.
 * -------------------------------------------------------------------------- */

/* Fault-time register snapshot passed to llgoEarlyFaultReport. The layout is
 * mirrored by captureRecord in earlypanic_baremetal.go. */
struct llgo_early_fault_capture {
    uint32_t mepc;
    uint32_t mcause;
    uint32_t mtval;
    uint32_t s0;
    uint32_t sp;
};

/* llgoStr stringifies a macro so the token-pasted offsets render as text
 * inside the assembly string literal (same trick as llgoFiberStr in
 * _wrap/context_riscv32.c). */
#define llgoStr_(x) #x
#define llgoStr(x)  llgoStr_(x)

/* Trap frame layout (128 bytes, 16-byte aligned):
 *   0..67:  ra, t0-t6, a0-a7, s0          (17 words)
 *   68..71: unused
 *   72..91: struct llgo_early_fault_capture (5 words)
 */
#define TRAP_FRAME_RA_OFF   0
#define TRAP_FRAME_T0_OFF   4
#define TRAP_FRAME_T1_OFF   8
#define TRAP_FRAME_T2_OFF   12
#define TRAP_FRAME_A0_OFF   16
#define TRAP_FRAME_A1_OFF   20
#define TRAP_FRAME_A2_OFF   24
#define TRAP_FRAME_A3_OFF   28
#define TRAP_FRAME_A4_OFF   32
#define TRAP_FRAME_A5_OFF   36
#define TRAP_FRAME_A6_OFF   40
#define TRAP_FRAME_A7_OFF   44
#define TRAP_FRAME_S0_OFF   48
#define TRAP_FRAME_T3_OFF   52
#define TRAP_FRAME_T4_OFF   56
#define TRAP_FRAME_T5_OFF   60
#define TRAP_FRAME_T6_OFF   64
#define TRAP_FRAME_CAP_OFF  72

/* `used` keeps both statics in the object: their only readers are the
 * naked trap's assembly, which the optimizer cannot see (same trick as
 * esp_prev_mtvec in targets/device/esp/esp32s31.c). */
__attribute__((used)) static uint32_t llgo_early_prev_mtvec;
__attribute__((used)) static uint32_t llgo_early_reporting;

void llgoEarlyFaultReport(void *cap); // exported by earlypanic_baremetal.go

/* Vectored dispatch: this core model runs mtvec in vectored mode - a
 * direct-mode write reads back with the low bits forced to 11 and exceptions
 * dispatch to base + 4*cause. A bare handler installed direct-mode therefore
 * executes from handler+4*cause on any exception (mid-instruction garbage:
 * illegal stores and compressed float stores, observed under esp-emu). The
 * install therefore points mtvec at a 32-entry jump table, every entry
 * entering the same common handler, so every cause lands on real code. The
 * table is 128 bytes so the whole cause range stays inside it. */
__attribute__((naked, used, aligned(128), section(".text.llgo_early_trap_table")))
static void llgo_early_trap_vectors(void) {
    __asm__ volatile(
        /* 32 four-byte jumps: vectored dispatch lands on base + 4*cause, so
         * every entry must keep the 4-byte stride (norvc blocks the
         * link-time relaxation into 2-byte c.j). */
        ".option push\n"
        ".option norvc\n"
        ".rept 32\n"
        "jal zero, llgo_early_trap_common\n"
        ".endr\n"
        ".option pop\n");
}

/* The common body: reached from every vector entry. */
__attribute__((naked, used)) static void llgo_early_trap_common(void) {
    __asm__ volatile(
        /* Stash every register the report call could clobber. The chained
         * handler re-derives its state from the CSRs, but it must still
         * observe the fault-time register values, so everything below is
         * restored before the chain jump. */
        "addi sp, sp, -128\n"
        "sw ra, " llgoStr(TRAP_FRAME_RA_OFF) "(sp)\n"
        "sw t0, " llgoStr(TRAP_FRAME_T0_OFF) "(sp)\n"
        "sw t1, " llgoStr(TRAP_FRAME_T1_OFF) "(sp)\n"
        "sw t2, " llgoStr(TRAP_FRAME_T2_OFF) "(sp)\n"
        "sw a0, " llgoStr(TRAP_FRAME_A0_OFF) "(sp)\n"
        "sw a1, " llgoStr(TRAP_FRAME_A1_OFF) "(sp)\n"
        "sw a2, " llgoStr(TRAP_FRAME_A2_OFF) "(sp)\n"
        "sw a3, " llgoStr(TRAP_FRAME_A3_OFF) "(sp)\n"
        "sw a4, " llgoStr(TRAP_FRAME_A4_OFF) "(sp)\n"
        "sw a5, " llgoStr(TRAP_FRAME_A5_OFF) "(sp)\n"
        "sw a6, " llgoStr(TRAP_FRAME_A6_OFF) "(sp)\n"
        "sw a7, " llgoStr(TRAP_FRAME_A7_OFF) "(sp)\n"
        "sw s0, " llgoStr(TRAP_FRAME_S0_OFF) "(sp)\n"
        "sw t3, " llgoStr(TRAP_FRAME_T3_OFF) "(sp)\n"
        "sw t4, " llgoStr(TRAP_FRAME_T4_OFF) "(sp)\n"
        "sw t5, " llgoStr(TRAP_FRAME_T5_OFF) "(sp)\n"
        "sw t6, " llgoStr(TRAP_FRAME_T6_OFF) "(sp)\n"
        /* Report only genuine exceptions, once; everything else chains. */
        "csrr a0, mcause\n"
        "bltz a0, 1f\n" /* interrupts chain silently */
        "li t0, 3\n"
        "beq a0, t0, 1f\n" /* semihosting exit EBREAK chains silently */
        "lui t0, %hi(llgo_early_reporting)\n"
        "lw t1, %lo(llgo_early_reporting)(t0)\n"
        "bnez t1, 1f\n"
        "li t1, 1\n"
        "sw t1, %lo(llgo_early_reporting)(t0)\n"
        /* Fill the capture record: mepc, mcause, mtval, s0, sp. */
        "csrr a1, mepc\n"
        "csrr a2, mtval\n"
        "sw a1, " llgoStr(TRAP_FRAME_CAP_OFF + 0) "(sp)\n"
        "sw a0, " llgoStr(TRAP_FRAME_CAP_OFF + 4) "(sp)\n"
        "sw a2, " llgoStr(TRAP_FRAME_CAP_OFF + 8) "(sp)\n"
        "lw a1, " llgoStr(TRAP_FRAME_S0_OFF) "(sp)\n"
        "sw a1, " llgoStr(TRAP_FRAME_CAP_OFF + 12) "(sp)\n"
        "mv a1, sp\n"
        "addi a1, a1, 128\n" /* the interrupted sp, not the trap frame */
        "sw a1, " llgoStr(TRAP_FRAME_CAP_OFF + 16) "(sp)\n"
        "mv a0, sp\n"
        "addi a0, a0, " llgoStr(TRAP_FRAME_CAP_OFF) "\n"
        "call llgoEarlyFaultReport\n"
        "sw zero, %lo(llgo_early_reporting)(t0)\n"
        /* A reported exception is fatal: exit through the same semihosting
         * sequence the runtime's exit uses. Emulators intercept it and
         * terminate with the console drained; on real hardware the device
         * file's trap recognizes the sequence (mcause 3 chains silently
         * through this handler) and parks the core cleanly instead of
         * falling into the ROM exception loop. */
        "li a0, 23\n"
        "li a1, 0x20026\n"
        ".option push\n"
        ".option norvc\n"
        "slli zero, zero, 0x1f\n"
        "ebreak\n"
        "srai zero, zero, 7\n"
        ".option pop\n"
        /* No semihosting host and no device file: park instead of resuming
         * a state that just faulted. */
        "wfi\n"
        "j .\n"
        "1:\n"
        /* Restore everything except t0, which carries the chain target and
         * is reloaded last; the chained handler saves its own registers
         * before using any. */
        "lw ra, " llgoStr(TRAP_FRAME_RA_OFF) "(sp)\n"
        "lw a0, " llgoStr(TRAP_FRAME_A0_OFF) "(sp)\n"
        "lw a1, " llgoStr(TRAP_FRAME_A1_OFF) "(sp)\n"
        "lw a2, " llgoStr(TRAP_FRAME_A2_OFF) "(sp)\n"
        "lw a3, " llgoStr(TRAP_FRAME_A3_OFF) "(sp)\n"
        "lw a4, " llgoStr(TRAP_FRAME_A4_OFF) "(sp)\n"
        "lw a5, " llgoStr(TRAP_FRAME_A5_OFF) "(sp)\n"
        "lw a6, " llgoStr(TRAP_FRAME_A6_OFF) "(sp)\n"
        "lw a7, " llgoStr(TRAP_FRAME_A7_OFF) "(sp)\n"
        "lw s0, " llgoStr(TRAP_FRAME_S0_OFF) "(sp)\n"
        "lw t1, " llgoStr(TRAP_FRAME_T1_OFF) "(sp)\n"
        "lw t2, " llgoStr(TRAP_FRAME_T2_OFF) "(sp)\n"
        "lw t3, " llgoStr(TRAP_FRAME_T3_OFF) "(sp)\n"
        "lw t4, " llgoStr(TRAP_FRAME_T4_OFF) "(sp)\n"
        "lw t5, " llgoStr(TRAP_FRAME_T5_OFF) "(sp)\n"
        "lw t6, " llgoStr(TRAP_FRAME_T6_OFF) "(sp)\n"
        "addi sp, sp, 128\n"
        /* Chain interrupts and breakpoints to the handler installed before
         * us, entered at its base (with the device file linked that is its
         * direct-mode trap; without one, the crt0 vector table's first
         * entry). */
        "lui t0, %hi(llgo_early_prev_mtvec)\n"
        "lw t0, %lo(llgo_early_prev_mtvec)(t0)\n"
        "andi t0, t0, -4\n"
        "jr t0\n");
}

void llgo_early_trap_install(void) {
    uint32_t prev;
    __asm__ volatile("csrr %0, mtvec" : "=r"(prev));
    llgo_early_prev_mtvec = prev;
    __asm__ volatile("csrw mtvec, %0"
                     :
                     : "r"((uint32_t)(intptr_t)&llgo_early_trap_vectors));
    llgo_early_reporting = 0;
}
