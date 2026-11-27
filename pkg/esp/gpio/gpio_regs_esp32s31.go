//go:build esp32s31

// Register facts the baremetal GPIO backend needs for the ESP32-S31. Every
// value below is transcribed from the ESP-IDF SOC headers (paths relative to
// components/soc/esp32s31/) together with the line it comes from.
//
//	register/soc/reg_base.h:74         DR_REG_GPIO_BASE   = 0x20583000
//	register/soc/reg_base.h:73         DR_REG_IO_MUX_BASE = 0x20582000
//	register/soc/gpio_reg.h:37,53,69   GPIO_OUT_REG 0x4, W1TS 0x8, W1TC 0xC
//	register/soc/gpio_reg.h:201,218    GPIO_ENABLE_W1TS 0x38, W1TC 0x3C
//	register/soc/gpio_reg.h:333        GPIO_IN_REG 0x64
//	register/soc/io_mux_reg.h:148-...  PERIPHS_IO_MUX_U_* offsets
//	register/soc/io_mux_reg.h:140      PIN_FUNC_GPIO = 1
//	include/soc/soc_caps.h:228,242     SOC_GPIO_PIN_COUNT = 62, valid mask
//	                                   excludes BIT29 and BIT41
package gpio

// gpioBaremetalChip names the SoC in the backend's diagnostics.
const gpioBaremetalChip = "esp32s31"

const (
	// gpioBase is DR_REG_GPIO_BASE (reg_base.h:74).
	gpioBase uintptr = 0x20583000
	// ioMuxBase is DR_REG_IO_MUX_BASE (reg_base.h:73).
	ioMuxBase uintptr = 0x20582000

	// GPIO register offsets inside gpioBase. The S31 GPIO block moved the
	// enable and input registers relative to the C3, so these are the S31
	// values, not a copy of the C3 table.
	gpioRegOut       = 0x04 // GPIO_OUT_REG (gpio_reg.h:37)
	gpioRegOutW1TS   = 0x08 // GPIO_OUT_W1TS_REG (gpio_reg.h:53)
	gpioRegOutW1TC   = 0x0C // GPIO_OUT_W1TC_REG (gpio_reg.h:69)
	gpioRegEnableW1S = 0x38 // GPIO_ENABLE_W1TS_REG (gpio_reg.h:201)
	gpioRegEnableW1C = 0x3C // GPIO_ENABLE_W1TC_REG (gpio_reg.h:218)
	gpioRegIn        = 0x64 // GPIO_IN_REG (gpio_reg.h:333)

	// Pads 0..61 are GPIOs; 29 and 41 are holes in the pad map
	// (soc_caps.h:228 SOC_GPIO_PIN_COUNT = 62, soc_caps.h:242 valid mask
	// = all 62 minus BIT29 and BIT41).
	gpioPinCount = 62
	// gpioPinGapMask marks those two holes.
	gpioPinGapMask uint64 = 1<<29 | 1<<41

	// I/O MUX register addressing: gpio0 sits at IO_MUX+0x00 and every
	// later pad is 4 bytes further (io_mux_reg.h:148-...:
	// U_XTAL32K_HYS_P 0x0, U_XTAL32K_HYS_N 0x4, U_GPIO2 0x8, ...,
	// U_GPIO62 0xF8).
	ioMuxFirstOffset = 0x00
	ioMuxStride      = 4
)
