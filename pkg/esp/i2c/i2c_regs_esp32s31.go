//go:build esp32s31

// Register facts the baremetal I2C backend needs for the ESP32-S31. Every
// value is transcribed from the ESP-IDF SOC headers (paths relative to
// components/soc/esp32s31/ or the chip's HAL), with the line it comes from.
//
//	register/soc/reg_base.h:26,27       DR_REG_I2C0_BASE 0x20385000, I2C1 0x20386000
//	register/soc/i2c_reg.h:13           REG_I2C_BASE(i) = DR_REG_I2C0_BASE + i*0x1000
//	include/soc/soc_caps.h:304          SOC_I2C_NUM = 3 (I2C0, I2C1, LP_I2C)
//	register/soc/reg_base.h:79          DR_REG_HP_SYS_CLKRST_BASE = 0x20587000
//	register/soc/hp_sys_clkrst_reg.h:2296,2357  I2C0_CTRL0 0xdc, I2C1_CTRL0 0xe0
//	register/soc/reg_base.h:73,74       IO_MUX 0x20582000, GPIO 0x20583000
//	register/soc/gpio_reg.h:6326,15803  FUNC0_IN_SEL_CFG 0x2f4, OUT_SEL_CFG 0xaf4
//	include/soc/gpio_sig_map.h:148-155  I2C0_SCL_PAD_IN/OUT 68, SDA 69 (I2C1 70/71)
package i2c

import "github.com/xgo-dev/llgo/pkg/esp/internal/mmio"

const (
	// i2cPortCount covers the two HP master ports. LP_I2C (the third
	// SOC_I2C_NUM instance) is a different controller with its own
	// register block and is not driven by this backend.
	i2cPortCount = 2

	// Port bases (reg_base.h:26,27); the block stride is 0x1000
	// (i2c_reg.h:13).
	i2cPortBase0 uintptr = 0x20385000
	i2cPortBase1 uintptr = 0x20386000

	// i2cSourceHz is the frequency of the I2C source clock: the default
	// source is XTAL (clk_tree_defs.h:360 I2C_CLK_SRC_DEFAULT =
	// SOC_MOD_CLK_XTAL), a 40 MHz crystal (clk_tree_defs.h:137).
	i2cSourceHz uint32 = 40_000_000

	// i2cSCLWaitUS is I2C_LL_SCL_WAIT_US_VAL_DEFAULT
	// (esp_hal_i2c/esp32s31/include/hal/i2c_ll.h:88): how long a slave may
	// hold SCL low before the controller gives up. 2.5 ms converts to a TO
	// register value of 17 on both chips, i.e. 2^17 source ticks = 3.3 ms.
	i2cSCLWaitUS uint32 = 2500

	// On this chip the divider and the source select are NOT in the I2C
	// block (there is no I2C_CLK_CONF register at 0x54 here): they live in
	// the HP_SYS_CLKRST block, one CTRL0 register per port
	// (hp_sys_clkrst_reg.h:2296 / :2357).
	clkrstBase     uintptr = 0x20587000 // reg_base.h:79
	clkrstI2C0Ctrl         = 0xdc       // hp_sys_clkrst_reg.h:2296
	clkrstI2C1Ctrl         = 0xe0       // hp_sys_clkrst_reg.h:2357

	// i2cHwFsmRst: the S31's controller has a usable hardware FSM reset
	// (the HAL defines no I2C_LL_SUPPORT_HW_FSM_RST exclusion for this
	// chip; IDF's timeout recovery here is i2c_ll_master_fsm_rst), so
	// fsmReset takes the short path.
	i2cHwFsmRst           = true
	clkrstApbClkEn uint32 = 1 << 0 // REG_I2Cx_APB_CLK_EN
	clkrstRstEn    uint32 = 1 << 1 // REG_I2Cx_RST_EN
	clkrstSrcSelB  uint32 = 1 << 3 // REG_I2Cx_CLK_SRC_SEL (0 = XTAL)
	clkrstClkEn    uint32 = 1 << 4 // REG_I2Cx_CLK_EN
	clkrstDivShift        = 5      // REG_I2Cx_CLK_DIV_NUM [12:5]
	clkrstDivMask  uint32 = 0xFF

	// GPIO block bases for the matrix and pad configuration.
	gpioBaseI2C  uintptr = 0x20583000 // reg_base.h:74
	ioMuxBaseI2C         = 0x20582000 // reg_base.h:73

	// GPIO matrix registers: output selector indexed by pin
	// (gpio_reg.h:15803, 0xaf4 + 4*pin), input selector indexed by signal
	// (gpio_reg.h:6326, 0x2f4 + 4*sig). The S31 widened both selectors
	// relative to the C3, so the masks are chip-local.
	gpioFuncOutSelBase = 0xaf4
	gpioFuncInSelBase  = 0x2f4
	// Output selector field: OUT_SEL [8:0] (gpio_reg.h:15803-15850). The
	// selector's OE_SEL bit stays clear, which is what hands the pad's
	// output enable to the peripheral instead of GPIO_ENABLE (gpio_reg.h:
	// 15832: "0: Use output enable signal from peripheral").
	gpioOutSelMask uint32 = 0x1FF
	// Input selector fields: FUNC_IN_SEL [7:0] (the GPIO number),
	// FUNC_IN_INV_SEL bit 8, SIG_IN_SEL bit 9 (1 = route via the matrix)
	// (gpio_reg.h:6326-6360).
	gpioInSelMask uint32 = 0xFF
	gpioSigInSelB        = 1 << 9
	// GPIO_PINn_REG (gpio_reg.h:845 + 4*n), PAD_DRIVER bit 2: 1 = open
	// drain (gpio_reg.h:862-868).
	gpioPinRegBase = 0xf4
	gpioPinODB     = 1 << 2
	// I/O MUX: FUN_PU bit 8, FUN_IE bit 9, MCU_SEL [14:12]
	// (io_mux_reg.h:42-65); PIN_FUNC_GPIO = 1 (io_mux_reg.h:140).
	i2cIOMuxFunPU    uint32 = 1 << 8
	i2cIOMuxFunIE    uint32 = 1 << 9
	i2cIOMuxMcuSel          = 0x7 << 12
	i2cMcuSelShiftBy        = 12
	i2cPinFuncGPIO   uint32 = 1
	// Pad register addressing: gpio0 at IO_MUX+0x00, then 4 bytes per pad
	// (io_mux_reg.h:148-...), 63 pads (soc_caps.h:228). GPIO29 and GPIO41
	// are holes in the digital pad map (soc_caps.h:242) and are rejected
	// by i2cPinOK.
	i2cIOMuxFirstOff = 0x00
	i2cIOMuxStride   = 4
	i2cPinCount      = 62
	i2cPinGapMask    = uint64(1)<<29 | uint64(1)<<41

	// GPIO matrix signal indices (gpio_sig_map.h:148-155), per port.
	i2cSclSigPort0 = 68 // I2C0_SCL_PAD_IN_IDX / OUT_IDX
	i2cSdaSigPort0 = 69 // I2C0_SDA_PAD_IN_IDX / OUT_IDX
	i2cSclSigPort1 = 70 // I2C1_SCL_PAD_IN_IDX / OUT_IDX
	i2cSdaSigPort1 = 71 // I2C1_SDA_PAD_IN_IDX / OUT_IDX
)

// i2cPortBase is the register block of one master port.
func i2cPortBase(port int32) uintptr {
	if port == 1 {
		return i2cPortBase1
	}
	return i2cPortBase0
}

// i2cClkCtrlAddr is the HP_SYS_CLKRST control register of one port.
func i2cClkCtrlAddr(port int32) uintptr {
	if port == 1 {
		return clkrstBase + clkrstI2C1Ctrl
	}
	return clkrstBase + clkrstI2C0Ctrl
}

// i2cSclSig / i2cSdaSig are the GPIO matrix signal indices of a port.
func i2cSclSig(port int32) uint32 {
	if port == 1 {
		return i2cSclSigPort1
	}
	return i2cSclSigPort0
}

func i2cSdaSig(port int32) uint32 {
	if port == 1 {
		return i2cSdaSigPort1
	}
	return i2cSdaSigPort0
}

// i2cPinOK reports whether pin can carry an I2C signal on this chip.
func i2cPinOK(pin int32) bool {
	if pin < 0 || pin >= i2cPinCount {
		return false
	}
	return i2cPinGapMask&(1<<uint(pin)) == 0
}

// i2cEnablePortClock gates the port's APB clock on and pulses its module
// reset through HP_SYS_CLKRST, mirroring i2c_ll_enable_bus_clock and
// i2c_ll_reset_register (esp_hal_i2c/esp32s31/include/hal/i2c_ll.h:155,172).
func i2cEnablePortClock(port int32) {
	ctrl := i2cClkCtrlAddr(port)
	mmio.Set32(ctrl, clkrstApbClkEn)
	mmio.Set32(ctrl, clkrstRstEn)
	mmio.Clear32(ctrl, clkrstRstEn)
}

// i2cResetPort pulses the port's reset bit in its HP_SYS_CLKRST control
// register (REG_I2Cx_RST_EN, hp_sys_clkrst_reg.h fields on the I2C0/1_CTRL0
// words), the same bit i2cEnablePortClock exercises.
func i2cResetPort(port int32) {
	ctrl := i2cClkCtrlAddr(port)
	mmio.Set32(ctrl, clkrstRstEn)
	mmio.Clear32(ctrl, clkrstRstEn)
}

// i2cApplyClockDiv programs the source select, the divider and the controller
// clock in the port's HP_SYS_CLKRST control register: SCLK = XTAL (40 MHz) /
// div. There is no divider inside the S31 I2C block, so a driver that only
// writes the I2C register window runs at the undivided source clock or not at
// all.
func i2cApplyClockDiv(port int32, div uint32) {
	ctrl := i2cClkCtrlAddr(port)
	mmio.Clear32(ctrl, clkrstSrcSelB) // 0 = XTAL
	mmio.Clear32(ctrl, clkrstDivMask<<clkrstDivShift)
	mmio.Set32(ctrl, ((div-1)&clkrstDivMask)<<clkrstDivShift)
	mmio.Set32(ctrl, clkrstClkEn)
}
