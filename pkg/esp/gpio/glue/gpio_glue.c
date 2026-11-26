/* Thin C glue between llgo-compiled Go code and the ESP-IDF GPIO driver.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there. The integration recipes and the boundary
 * rules (integer-and-pointer-only signatures, esp_err passed through as
 * int32, IDF macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * All the driver calls here are synchronous register operations, so no
 * mailbox is needed: the glue returns as soon as the driver call does.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>

#include "driver/gpio.h"

/* Go-side Pin modes (gpio.go): Input = 0, Output = 1. Output also enables
 * the input buffer (GPIO_MODE_INPUT_OUTPUT) so a Set->Get round-trip reads
 * back the driven level; a plain GPIO_MODE_OUTPUT leaves the input buffer
 * off and gpio_get_level would not reflect the output. */
#define LG_GPIO_INPUT 0
#define LG_GPIO_OUTPUT 1

int32_t llgo_gpio_set_direction(int32_t pin, int32_t mode, int32_t pull)
{
    /* gpio.go promises [0, 64) on both sides: 1ULL << pin is undefined
     * behavior outside that range, so reject it here with the same code the
     * IDF driver would return for an invalid GPIO number. */
    if (pin < 0 || pin >= 64) {
        return ESP_ERR_INVALID_ARG;
    }
    gpio_mode_t m = (mode == LG_GPIO_OUTPUT) ? GPIO_MODE_INPUT_OUTPUT : GPIO_MODE_INPUT;
    gpio_config_t cfg = {
        .pin_bit_mask = 1ULL << pin,
        .mode = m,
        .pull_up_en = (pull == 1) ? GPIO_PULLUP_ENABLE : GPIO_PULLUP_DISABLE,
        .pull_down_en = (pull == 2) ? GPIO_PULLDOWN_ENABLE : GPIO_PULLDOWN_DISABLE,
    };
    esp_err_t e = gpio_config(&cfg);
    if (e != ESP_OK) return e;
    return gpio_set_direction((gpio_num_t)pin, m);
}

int32_t llgo_gpio_set_level(int32_t pin, int32_t level)
{
    return gpio_set_level((gpio_num_t)pin, (uint32_t)level);
}

int32_t llgo_gpio_get_level(int32_t pin)
{
    return gpio_get_level((gpio_num_t)pin);
}

/* Invert an output pin. gpio_toggle_level exists on newer IDF releases, but
 * get+set is version-portable and the glue stays thin either way. */
int32_t llgo_gpio_toggle(int32_t pin)
{
    return gpio_set_level((gpio_num_t)pin,
                          gpio_get_level((gpio_num_t)pin) ? 0u : 1u);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_gpio_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
