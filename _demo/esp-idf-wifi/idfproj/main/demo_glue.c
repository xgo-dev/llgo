/* Demo-local C glue: symbols the acceptance program uses that belong to no
 * pkg/esp package. The package glues (wifi/gpio/i2c/i2s) are synced into
 * this directory from the pkg/esp package glue templates by
 * host/make-artifact.sh; this file is
 * the only glue kept in the repo, and only because it is demo-specific.
 *
 * Boundary rules (see pkg/esp/README.md) apply here unchanged.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>

/* Lower 32 bits of the CPU cycle counter, for on-device benchmarks. It wraps
 * every 17.9 s at 240 MHz, so callers time short intervals with differences. */
uint32_t llgo_demo_rdcycle(void)
{
    uint32_t cycles;
    __asm__ volatile ("csrr %0, cycle" : "=r"(cycles));
    return cycles;
}

#else /* !ESP_PLATFORM */

typedef int llgo_demo_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
