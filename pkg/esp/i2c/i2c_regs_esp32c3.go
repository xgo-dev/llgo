//go:build esp32c3

// Register facts the baremetal I2C backend needs for the ESP32-C3. Every value
// is transcribed from the ESP-IDF SOC headers (paths relative to
// components/soc/esp32c3/ or the chip's HAL), with the line it comes from.
//
//	register/soc/reg_base.h:30          DR_REG_I2C_EXT_BASE = 0x60013000
//	include/soc/soc.h:26                REG_I2C_BASE(i) = DR_REG_I2C_EXT_BASE + i*0x14000
//	include/soc/soc_caps.h:182          SOC_I2C_NUM = 1
//	register/soc/i2c_reg.h:817          I2C_CLK_CONF_REG (0x54) - divider lives here
//	register/soc/reg_base.h:6           DR_REG_SYSTEM_BASE = 0x600c0000
//	register/soc/system_reg.h:76,320    SYSTEM_PERIP_CLK_EN0 0x010, PERIP_RST_EN0 0x018
//	register/soc/system_reg.h:223,465   SYSTEM_I2C_EXT0_CLK_EN / _RST = bit 7
//	register/soc/reg_base.h:23,27       GPIO 0x60004000, IO_MUX 0x60009000
//	register/soc/gpio_reg.h:1322,3882   GPIO_FUNC0_IN_SEL_CFG 0x154, OUT_SEL_CFG 0x554
//	include/soc/gpio_sig_map.h:87-90    I2CEXT0_SCL_IN/OUT 53, I2CEXT0_SDA_IN/OUT 54
package i2c

import "github.com/xgo-dev/llgo/pkg/esp/internal/mmio"

const (
	// i2cPortCount is SOC_I2C_NUM (soc_caps.h:182).
	i2cPortCount = 1

	// i2cPortBaseC3 is REG_I2C_BASE(0) (soc.h:26). The block stride is
	// 0x14000 on this chip, but there is only one master port.
	i2cPortBaseC3 uintptr = 0x60013000

	// i2cSourceHz is the frequency of the I2C source clock: the default
	// source is XTAL (clk_tree_defs.h:265 I2C_CLK_SRC_DEFAULT =
	// SOC_MOD_CLK_XTAL), a 40 MHz crystal (clk_tree_defs.h:122).
	i2cSourceHz uint32 = 40_000_000

	// i2cSCLWaitUS is I2C_LL_SCL_WAIT_US_VAL_DEFAULT
	// (esp_hal_i2c/esp32c3/include/hal/i2c_ll.h:84): how long a slave may
	// hold SCL low before the controller gives up. 2 ms converts to a TO
	// register value of 17, i.e. 2^17 source ticks = 3.3 ms.
	i2cSCLWaitUS uint32 = 2000

	// The I2C clock divider and the source select live inside the I2C
	// block on this chip (i2c_reg.h:817 I2C_CLK_CONF_REG).
	i2cRegClkConf = 0x54
	i2cClkDivNum  = 0xFF    // I2C_SCLK_DIV_NUM [7:0]
	i2cClkActiveB = 1 << 21 // I2C_SCLK_ACTIVE (b21, i2c_reg.h:817-830)
	// The APB clock gate and the module reset live in the SYSTEM block.
	systemBase        uintptr = 0x600c0000 // reg_base.h:6
	systemPeripClkEn0         = 0x010      // system_reg.h:76
	systemPeripRstEn0         = 0x018      // system_reg.h:320
	systemI2CExt0Bit  uint32  = 1 << 7     // system_reg.h:223,465

	// i2cHwFsmRst reports whether the controller's FSM_RST bit is usable
	// for error recovery. The C3's HAL deliberately declines it ("FSM_RST
	// only resets the FSM, not using it", esp_hal_i2c/esp32c3/include/hal/
	// i2c_ll.h:32), so timeout recovery pulses the SYSTEM-block module
	// reset and re-runs the bring-up instead (see fsmReset).
	i2cHwFsmRst = false

	// GPIO block bases for the matrix and pad configuration.
	gpioBaseI2C  uintptr = 0x60004000 // reg_base.h:23
	ioMuxBaseI2C         = 0x60009000 // reg_base.h:27

	// GPIO matrix registers. On this chip the output selector is indexed
	// by pin (gpio_reg.h:3882 GPIO_FUNC0_OUT_SEL_CFG_REG 0x554 + 4*pin)
	// and the input selector by signal (gpio_reg.h:1322
	// GPIO_FUNC0_IN_SEL_CFG_REG 0x154 + 4*sig).
	gpioFuncOutSelBase = 0x554
	gpioFuncInSelBase  = 0x154
	// Output selector field: OUT_SEL [7:0] (gpio_reg.h:3882-3905). The
	// selector's OE_SEL bit stays clear, which is what hands the pad's
	// output enable to the peripheral instead of GPIO_ENABLE.
	gpioOutSelMask uint32 = 0xFF
	// Input selector fields: FUNC_IN_SEL [4:0] (the GPIO number),
	// FUNC_IN_INV_SEL bit 5, SIG_IN_SEL bit 6 (1 = route via the matrix)
	// (gpio_reg.h:1322-1340).
	gpioInSelMask uint32 = 0x1F
	gpioSigInSelB        = 1 << 6
	// GPIO_PINn_REG (gpio_reg.h:170 + 4*n), PAD_DRIVER bit 2: 1 = open
	// drain, the only mode an I2C pin may use.
	gpioPinRegBase = 0x74
	gpioPinODB     = 1 << 2
	// I/O MUX: FUN_PU bit 8, FUN_IE bit 9, MCU_SEL [14:12]
	// (io_mux_reg.h:31-65); PIN_FUNC_GPIO = 1 (io_mux_reg.h:120).
	i2cIOMuxFunPU    uint32 = 1 << 8
	i2cIOMuxFunIE    uint32 = 1 << 9
	i2cIOMuxMcuSel          = 0x7 << 12
	i2cMcuSelShiftBy        = 12
	i2cPinFuncGPIO   uint32 = 1
	// Pad register addressing: gpio0 at IO_MUX+0x04, then 4 bytes per pad
	// (io_mux_reg.h:163-...), 22 pads, no gaps (soc_caps.h:156,166).
	i2cIOMuxFirstOff = 0x04
	i2cIOMuxStride   = 4
	i2cPinCount      = 22

	// GPIO matrix signal indices (gpio_sig_map.h:87-90).
	i2cSclSigC3 = 53 // I2CEXT0_SCL_IN_IDX / I2CEXT0_SCL_OUT_IDX
	i2cSdaSigC3 = 54 // I2CEXT0_SDA_IN_IDX / I2CEXT0_SDA_OUT_IDX
)

// i2cPortBase is the register block of one master port.
func i2cPortBase(port int32) uintptr { return i2cPortBaseC3 }

// i2cSclSig / i2cSdaSig are the GPIO matrix signal indices; the C3 has a
// single port, so the port argument is unused.
func i2cSclSig(port int32) uint32 { return i2cSclSigC3 }

func i2cSdaSig(port int32) uint32 { return i2cSdaSigC3 }

// i2cPinOK reports whether pin can carry an I2C signal on this chip.
func i2cPinOK(pin int32) bool { return pin >= 0 && pin < i2cPinCount }

// i2cEnablePortClock gates the I2C APB clock on and pulses the module reset
// through the SYSTEM block, mirroring i2c_ll_enable_bus_clock and
// i2c_ll_reset_register (esp_hal_i2c/esp32c3/include/hal/i2c_ll.h:161,184).
func i2cEnablePortClock(port int32) {
	mmio.Set32(systemBase+systemPeripClkEn0, systemI2CExt0Bit)
	mmio.Set32(systemBase+systemPeripRstEn0, systemI2CExt0Bit)
	mmio.Clear32(systemBase+systemPeripRstEn0, systemI2CExt0Bit)
}

// i2cResetPort pulses the I2C module's reset bit in the SYSTEM block (the
// i2c_ll_reset_register unwind, i2c_ll.h:160-165: REG_I2C_EXT0_RST = 1 then
// 0; system_reg.h:465-467). One I2C on this chip, so the bit is fixed.
func i2cResetPort(port int32) {
	mmio.Set32(systemBase+systemPeripRstEn0, systemI2CExt0Bit)
	mmio.Clear32(systemBase+systemPeripRstEn0, systemI2CExt0Bit)
}

// i2cApplyClockDiv programs the source clock and the divider inside the I2C
// block (i2c_reg.h:817): SCLK = XTAL (40 MHz) / div, controller clock on.
func i2cApplyClockDiv(port int32, div uint32) {
	// One store carries all three fields: divider in [7:0], source select
	// 0 = XTAL (bit 20 stays clear), controller clock active (bit 21).
	mmio.Write32(i2cPortBase(port)+i2cRegClkConf, (div-1)&i2cClkDivNum|i2cClkActiveB)
}
