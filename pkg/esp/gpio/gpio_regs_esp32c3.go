//go:build esp32c3

// Register facts the baremetal GPIO backend needs for the ESP32-C3. Every
// value below is transcribed from the ESP-IDF SOC headers (paths relative to
// components/soc/esp32c3/) together with the line it comes from; nothing here
// is from memory.
//
//	register/soc/reg_base.h:23         DR_REG_GPIO_BASE   = 0x60004000
//	register/soc/reg_base.h:27         DR_REG_IO_MUX_BASE = 0x60009000
//	register/soc/gpio_reg.h:50,58,66   GPIO_OUT_REG 0x4, W1TS 0x8, W1TC 0xC
//	register/soc/gpio_reg.h:90,98      GPIO_ENABLE_W1TS 0x24, W1TC 0x28
//	register/soc/gpio_reg.h:114        GPIO_IN_REG 0x3C
//	register/soc/io_mux_reg.h:163-...  PERIPHS_IO_MUX_*_U offsets
//	register/soc/io_mux_reg.h:120      PIN_FUNC_GPIO = 1
//	include/soc/soc_caps.h:156         SOC_GPIO_PIN_COUNT = 22
package gpio

// gpioBaremetalChip names the SoC in the backend's diagnostics.
const gpioBaremetalChip = "esp32c3"

const (
	// gpioBase is DR_REG_GPIO_BASE (reg_base.h:23).
	gpioBase uintptr = 0x60004000
	// ioMuxBase is DR_REG_IO_MUX_BASE (reg_base.h:27).
	ioMuxBase uintptr = 0x60009000

	// GPIO register offsets inside gpioBase.
	gpioRegOut       = 0x04 // GPIO_OUT_REG (gpio_reg.h:50)
	gpioRegOutW1TS   = 0x08 // GPIO_OUT_W1TS_REG (gpio_reg.h:58)
	gpioRegOutW1TC   = 0x0C // GPIO_OUT_W1TC_REG (gpio_reg.h:66)
	gpioRegEnableW1S = 0x24 // GPIO_ENABLE_W1TS_REG (gpio_reg.h:90)
	gpioRegEnableW1C = 0x28 // GPIO_ENABLE_W1TC_REG (gpio_reg.h:98)
	gpioRegIn        = 0x3C // GPIO_IN_REG (gpio_reg.h:114)

	// GPIO pins 0..21 exist and all of them are valid digital GPIOs
	// (soc_caps.h:156,166: SOC_GPIO_PIN_COUNT = 22, valid mask = all 22).
	gpioPinCount = 22
	// gpioPinGapMask marks the pad slots that exist in the pad map but are
	// not usable GPIOs. The C3 has none (soc_caps.h:166).
	gpioPinGapMask uint64 = 0

	// I/O MUX register addressing: gpio0 sits at IO_MUX+0x04 and every
	// later pad is 4 bytes further (io_mux_reg.h:163-...: XTAL_32K_P_U
	// 0x04, XTAL_32K_N_U 0x08, GPIO2_U 0x0C, ..., GPIO21_U 0x58).
	ioMuxFirstOffset = 0x04
	ioMuxStride      = 4
)
