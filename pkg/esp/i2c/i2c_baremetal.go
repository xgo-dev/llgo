//go:build llgo && baremetal && !llgo.idf && (esp32c3 || esp32s31)

// The register-level I2C master backend: no ESP-IDF, no FreeRTOS, no glue
// template. It keeps the package's public API byte for byte (i2c.go) and
// implements it by programming the I2C controller's command list directly, so
// a plain `llgo build -target esp32s31` program can talk to a real bus.
//
// The backend is chosen by build tags, not by a runtime switch: llgo.idf
// builds keep the ESP-IDF driver path (i2c_idf.go), and every other build
// keeps the honest stub (i2c_stub.go) - except a baremetal llgo build of a
// chip that has a register table (i2c_regs_<chip>.go), which is this file.
// That is why the tag below names the chips: `baremetal` alone would also
// select this file on a Cortex-M, AVR or Xtensa target, which has neither the
// register window nor a RISC-V `csrr cycle`, so those targets must keep the
// stub.
//
// # What a transfer does
//
// One transfer is a sequence of command lists in COMD0..7: RSTART, the address
// byte and payload through the TX FIFO with a WRITE command, an optional
// repeated start for a register read, the READ command(s), and STOP. The
// controller runs a list once TRANS_START is written; the driver polls INT_RAW
// for the completion/NACK/timeout event, then drains the RX FIFO.
//
// The TX and RX FIFOs hold 32 bytes each, so a payload larger than one FIFO is
// split into several lists: the first carries the START (and the address
// byte), the last carries the STOP, and every list in between ends with an END
// command that holds the bus while the driver swaps the FIFO contents and
// starts the FSM again. The wire sees one transaction either way - no extra
// START or STOP - which is the split the ESP-IDF master driver makes for the
// same reason (esp_driver_i2c/i2c_master.c: s_i2c_write_command /
// s_i2c_read_command, "separated in to different transactions by interrupt").
//
// # Bounded waits
//
// Every wait is bounded twice: by a cycle-count deadline and by an iteration
// cap. The cycle counter is the honest bound on a chip where it runs (both
// supported SoCs clear mcountinhibit.CY at boot), and the iteration cap is
// what actually guarantees termination if a board's counter stays frozen - a
// frozen counter makes the deadline unreachable, so the cap must exist on its
// own. No path in this file spins forever.
package i2c

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/internal/mmio"
)

// esp_err codes this backend returns. They are the values the ESP-IDF driver
// reports for the same conditions, so a caller switching backends keeps its
// error handling: NACK is what i2c_master_probe returns for an absent device
// (ESP_ERR_NOT_FOUND), and a stuck bus is ESP_ERR_TIMEOUT.
const (
	espErrInvalidArg   = 0x102
	espErrInvalidState = 0x103
	espErrNotFound     = 0x105
	espErrTimeout      = 0x107
)

// The I2C register window. Both supported SoCs implement the same block at the
// same offsets (components/soc/esp32c3/register/soc/i2c_reg.h and the S31
// equivalent agree line for line); only the clock divider lives elsewhere, so
// the divider is the one thing the per-chip files own.
const (
	i2cRegSclLowPeriod   = 0x00 // I2C_SCL_LOW_PERIOD_REG   [8:0]
	i2cRegCtr            = 0x04 // I2C_CTR_REG
	i2cRegSr             = 0x08 // I2C_SR_REG
	i2cRegTo             = 0x0C // I2C_TO_REG               [4:0], enable bit 5
	i2cRegFifoConf       = 0x18 // I2C_FIFO_CONF_REG        rx/tx reset bits 12/13
	i2cRegData           = 0x1C // I2C_DATA_REG             FIFO data
	i2cRegIntRaw         = 0x20 // I2C_INT_RAW_REG
	i2cRegIntClr         = 0x24 // I2C_INT_CLR_REG
	i2cRegIntEna         = 0x28 // I2C_INT_ENA_REG
	i2cRegSdaHold        = 0x30 // I2C_SDA_HOLD_REG         [8:0]
	i2cRegSdaSample      = 0x34 // I2C_SDA_SAMPLE_REG       [8:0]
	i2cRegSclHighPeriod  = 0x38 // I2C_SCL_HIGH_PERIOD_REG  [8:0], wait [15:9]
	i2cRegSclStartHold   = 0x40 // I2C_SCL_START_HOLD_REG   [8:0]
	i2cRegSclRstartSetup = 0x44 // I2C_SCL_RSTART_SETUP_REG [8:0]
	i2cRegSclStopHold    = 0x48 // I2C_SCL_STOP_HOLD_REG    [8:0]
	i2cRegSclStopSetup   = 0x4C // I2C_SCL_STOP_SETUP_REG   [8:0]
	i2cRegComd0          = 0x58 // I2C_COMD0_REG, 4 bytes per command, 8 slots
	i2cRegSclSpConf      = 0x80 // I2C_SCL_SP_CONF_REG      bus-clear pulse count
)

// CTR fields (i2c_reg.h I2C_CTR_REG).
const (
	ctrSdaForceOut = 1 << 0
	ctrSclForceOut = 1 << 1
	ctrMsMode      = 1 << 4
	ctrTransStart  = 1 << 5
	ctrClkEn       = 1 << 8
	ctrFsmRst      = 1 << 10
	ctrConfUpgate  = 1 << 11
)

// SR fields (i2c_reg.h I2C_SR_REG).
const (
	srBusBusy = 1 << 4
)

// INT_RAW / INT_CLR / INT_ENA bits (i2c_reg.h I2C_INT_RAW_REG).
const (
	intTransComplete   = 1 << 7
	intTimeOut         = 1 << 8
	intNack            = 1 << 10
	intArbitrationLost = 1 << 5
	// END_DETECT (i2c_reg.h:376): the FSM reached an END command, i.e. the
	// end of a command list that is not the end of the transaction.
	intEndDetect = 1 << 3
)

// FIFO_CONF fields.
const (
	fifoRxRst = 1 << 12
	fifoTxRst = 1 << 13
)

// The TX and RX FIFO depth, I2C_LL_FIFO_LEN on both SoCs
// (esp_hal_i2c/esp32c3/include/hal/i2c_ll.h:29,
// esp_hal_i2c/esp32s31/include/hal/i2c_ll.h:28). One command list can clock
// out or take in at most this many bytes.
const fifoSegmentLen = 32

// TO register fields.
const i2cToEnable = 1 << 5

// Command encoding: byte_num [7:0], ack_en bit 8, ack_exp bit 9, ack_val
// bit 10, op_code [13:11] (the i2c_ll_hw_cmd_t union in
// components/esp_hal_i2c/<chip>/include/hal/i2c_ll.h).
const (
	cmdOpRestart = 6
	cmdOpWrite   = 1
	cmdOpRead    = 3
	cmdOpStop    = 2
	cmdOpEnd     = 4

	cmdAckEn  = 1 << 8
	cmdAckVal = 1 << 10
)

// The bus clear: nine SCL pulses is what the I2C spec allows a slave to need
// to release a stuck SDA line, and it is the count the ESP-IDF driver uses
// (I2C_LL_RESET_SLV_SCL_PULSE_NUM_DEFAULT = 9).
const busClearPulses = 9

// Budget sizing: one poll iteration is a handful of instructions, so the
// iteration cap is the cycle budget divided by 24 with a floor, and the cycle
// budget itself comes from the bus time the transfer can legitimately take.
const (
	cyclesPerMs         = 160_000 // 160 MHz, the CPU clock of both SoCs
	pollCyclesPerIter   = 24
	transferSlackCycles = 160_000 // 1 ms of scheduling and clock skew
	// 9 SCL periods per byte, with a factor 20 for arbitration losses and
	// clock stretching.
	busCyclesPerByteFactor = 9 * 20
)

// budget is one bounded wait: a cycle-counter deadline plus an iteration cap.
type budget struct {
	expire uint32
	iters  int32
}

// transferBudget bounds one transfer of n bytes at speedHz.
func transferBudget(speedHz uint32, n int32) budget {
	perByte := uint64(cyclesPerMs*1000) * busCyclesPerByteFactor / uint64(speedHz)
	c := uint64(transferSlackCycles) + perByte*uint64(n)
	if max := uint64(cyclesPerMs) * DefaultTimeoutMs; c > max {
		c = max
	}
	return budget{expire: mmio.Cycles() + uint32(c), iters: int32(c/pollCyclesPerIter) + 64}
}

// expired reports whether the budget is spent, whichever bound trips first.
func (b *budget) expired() bool {
	if b.iters <= 0 {
		return true
	}
	b.iters--
	return int32(mmio.Cycles()-b.expire) >= 0
}

// busState is one open master port. The handle the public API carries is the
// port's register base address, so no state is needed to translate it back.
type busState struct {
	base uintptr
	port int32
	open bool
	// ctr is the value the controller's CTR register was left with (master
	// mode, forced output pins, no arbitration). TRANS_START re-writes it
	// with the start bit instead of reading the register back, which keeps
	// the start sequence off the read-modify-write path entirely.
	ctr uint32
}

// devState is one bound device. The public Device handle is a 1-based index
// into this table, which keeps the handle a small integer the GC never scans
// while the address and speed stay in Go memory.
type devState struct {
	bus   int32
	addr  uint8
	speed uint32
	open  bool
}

// The bus table is one entry per master port the chip has (i2cPortCount), so
// two Bus values on a single-port chip cannot alias one controller: the second
// NewBus refuses instead of reaching the same register block and overwriting
// the first bus's pin matrix.
var (
	baremetalBuses [2]busState // i2cPortCount is 1 or 2 on every supported SoC
	baremetalDevs  [4]devState
)

// busByHandle finds the open bus behind a public-API handle.
func busByHandle(handle uintptr) *busState {
	for i := range baremetalBuses {
		if baremetalBuses[i].open && baremetalBuses[i].base == handle {
			return &baremetalBuses[i]
		}
	}
	return nil
}

// busIndex is busByHandle plus the table index the devices refer to.
func busIndex(handle uintptr) int32 {
	for i := range baremetalBuses {
		if baremetalBuses[i].open && baremetalBuses[i].base == handle {
			return int32(i)
		}
	}
	return -1
}

// devByHandle finds the open device behind a public-API handle.
func devByHandle(handle uintptr) *devState {
	if handle == 0 || handle > uintptr(len(baremetalDevs)) {
		return nil
	}
	d := &baremetalDevs[handle-1]
	if !d.open {
		return nil
	}
	return d
}

// newBus installs the controller on the pins: I/O MUX to the GPIO function,
// open-drain pads, GPIO matrix routing to the port's SDA/SCL signals, then the
// controller's own clock, mode and bus-clear. The bus keeps the internal
// pull-ups on (an idle I2C line must be high); 400 kHz still wants external
// pull-ups, and the bus is released by delBus.
func newBus(handle *uintptr, sda int32, scl int32) int32 {
	if !i2cPinOK(sda) || !i2cPinOK(scl) || sda == scl {
		return espErrInvalidArg
	}
	idx := int32(-1)
	for i := int32(0); i < int32(i2cPortCount) && i < int32(len(baremetalBuses)); i++ {
		if !baremetalBuses[i].open {
			idx = i
			break
		}
	}
	if idx < 0 {
		return espErrInvalidState
	}
	port := idx
	base := i2cPortBase(port)

	i2cEnablePortClock(port)
	routePin(port, sda, true)
	routePin(port, scl, false)

	// Controller setup, mirroring i2c_hal_master_init (esp_hal_i2c/
	// i2c_hal.c:38-52): master mode; SDA/SCL_FORCE_OUT set - the HAL reaches
	// it through i2c_ll_enable_pins_open_drain, which sets both bits, and
	// the open drain that matters on the wire is the pad's PAD_DRIVER set in
	// routePin; arbitration off (single master); MSB first; NACK when the
	// RX FIFO fills.
	// One full write of the wanted value, no read-modify-write: CTR is
	// never read back here (or anywhere - TRANS_START re-writes this value
	// with the start bit), so the configuration is exactly these bits:
	// master mode, pins forced out for the wire, clock on, config latched;
	// arbitration off, MSB first, RX-full ACK level 0 (all reset values
	// this write enforces regardless of what the block powers up with).
	ctr := uint32(ctrMsMode|ctrSdaForceOut|ctrSclForceOut) |
		ctrClkEn | ctrConfUpgate
	mmio.Write32(base+i2cRegCtr, ctr)
	mmio.Write32(base+i2cRegIntEna, 0)
	mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
	resetFifos(base)

	// A slave that was reset mid-byte can hold SDA low; nine SCL pulses
	// release it before the first transfer.
	clearBus(base)

	baremetalBuses[idx] = busState{base: base, port: port, open: true, ctr: ctr}
	*handle = base
	return 0
}

// routePin connects one pad to the port's SDA or SCL signal.
func routePin(port int32, pin int32, isSDA bool) {
	sig := i2cSclSig(port)
	if isSDA {
		sig = i2cSdaSig(port)
	}

	// I/O MUX: hand the pad to the GPIO block (MCU_SEL = PIN_FUNC_GPIO),
	// input buffer on, internal pull-up on, pull-down off.
	mux := ioMuxBaseI2C + i2cIOMuxFirstOff + uintptr(pin)*i2cIOMuxStride
	mmio.Clear32(mux, i2cIOMuxMcuSel)
	mmio.Set32(mux, i2cPinFuncGPIO<<i2cMcuSelShiftBy|i2cIOMuxFunIE|i2cIOMuxFunPU)

	// Open-drain pad: the pad only pulls low, which is what an I2C line
	// needs from both ends (gpio_od_enable in the IDF pin setup,
	// esp_driver_i2c/i2c_common.c:328).
	mmio.Set32(gpioBaseI2C+gpioPinRegBase+uintptr(pin)*4, gpioPinODB)
	// No GPIO_ENABLE write here. The output selector below keeps OE_SEL
	// clear (its reset value), which routes the pad's output enable from
	// the I2C peripheral through the matrix - exactly what the ESP-IDF
	// driver leaves it at (i2c_common.c: input enable, open drain, func
	// select, both matrix connections; no output-enable step).

	// Output matrix: the pad drives the port's *_OUT signal.
	mmio.Write32(gpioBaseI2C+gpioFuncOutSelBase+uintptr(pin)*4, sig&gpioOutSelMask)
	// Input matrix: the port's *_IN signal reads this pad. The input
	// selector is indexed by signal, not by pin, and has to be told to
	// route through the matrix at all.
	mmio.Write32(gpioBaseI2C+gpioFuncInSelBase+uintptr(sig)*4,
		uint32(pin)&gpioInSelMask|gpioSigInSelB)
}

// clearBus runs the controller's 9-pulse bus clear and waits (bounded) for it
// to finish.
func clearBus(base uintptr) {
	mmio.Write32(base+i2cRegSclSpConf, busClearPulses<<1|1)
	mmio.Set32(base+i2cRegCtr, ctrConfUpgate)
	b := budget{expire: mmio.Cycles() + 5*cyclesPerMs, iters: 5*cyclesPerMs/pollCyclesPerIter + 64}
	for {
		if mmio.Read32(base+i2cRegSclSpConf)&1 == 0 {
			return
		}
		if b.expired() {
			return
		}
	}
}

// resetFifos drops whatever a previous transfer left in either FIFO.
func resetFifos(base uintptr) {
	mmio.Set32(base+i2cRegFifoConf, fifoTxRst|fifoRxRst)
	mmio.Clear32(base+i2cRegFifoConf, fifoTxRst|fifoRxRst)
}

// programTiming writes the SCL timing registers for one bus speed. The
// divider differs per chip (inside the I2C block on the C3, in the clock tree
// on the S31), so that half lives in the per-chip files.
func programTiming(base uintptr, port int32, speedHz uint32) {
	t := busTimingFor(i2cSourceHz, speedHz)
	i2cApplyClockDiv(port, t.clkDiv)
	mmio.Write32(base+i2cRegSclLowPeriod, (t.low-1)&0x1FF)
	mmio.Write32(base+i2cRegSclHighPeriod, t.high&0x1FF|(t.waitHigh&0x7F)<<9)
	mmio.Write32(base+i2cRegSdaHold, (t.sdaHold-1)&0x1FF)
	mmio.Write32(base+i2cRegSdaSample, (t.sdaSampl-1)&0x1FF)
	mmio.Write32(base+i2cRegSclRstartSetup, (t.setup-1)&0x1FF)
	mmio.Write32(base+i2cRegSclStopSetup, (t.setup-1)&0x1FF)
	mmio.Write32(base+i2cRegSclStartHold, (t.hold-1)&0x1FF)
	mmio.Write32(base+i2cRegSclStopHold, (t.hold-1)&0x1FF)
	// The hardware timeout bounds how long a slave may hold SCL low (clock
	// stretching), which has nothing to do with the SCL period, so it does
	// not come from the timing table: IDF programs it from
	// I2C_LL_SCL_WAIT_US_VAL_DEFAULT right after the timing registers
	// (esp_driver_i2c/i2c_master.c: i2c_hal_master_set_scl_timeout_val).
	mmio.Write32(base+i2cRegTo, sclTimeoutVal(i2cSourceHz, i2cSCLWaitUS)&0x1F|i2cToEnable)
}

// waitIdle waits (bounded) for the bus to become idle before a transfer.
func waitIdle(base uintptr, b *budget) bool {
	for {
		if mmio.Read32(base+i2cRegSr)&srBusBusy == 0 {
			return true
		}
		if b.expired() {
			return false
		}
	}
}

// addDevice binds an address and a bus speed. The speed is programmed per
// transfer, so several devices may share a bus at different rates.
func addDevice(bus uintptr, handle *uintptr, addr uint32, speedHz uint32) int32 {
	if addr > 0x7F {
		return espErrInvalidArg
	}
	if !validSpeed(speedHz) {
		return espErrInvalidArg
	}
	idx := busIndex(bus)
	if idx < 0 {
		return espErrInvalidState
	}
	for i := range baremetalDevs {
		if baremetalDevs[i].open {
			continue
		}
		baremetalDevs[i] = devState{bus: idx, addr: uint8(addr), speed: speedHz, open: true}
		*handle = uintptr(i + 1)
		return 0
	}
	return espErrInvalidState
}

// rmDevice drops a device binding. The bus keeps running.
func rmDevice(dev uintptr) int32 {
	d := devByHandle(dev)
	if d == nil {
		return espErrInvalidArg
	}
	*d = devState{}
	return 0
}

// delBus releases the port and its devices.
func delBus(bus uintptr) int32 {
	idx := busIndex(bus)
	if idx < 0 {
		return espErrInvalidArg
	}
	for i := range baremetalDevs {
		if baremetalDevs[i].open && baremetalDevs[i].bus == idx {
			baremetalDevs[i] = devState{}
		}
	}
	baremetalBuses[idx] = busState{}
	return 0
}

// transmit writes data to the device.
func transmit(dev uintptr, data *uint8, length int32, timeoutMs int32) int32 {
	d := devByHandle(dev)
	if d == nil {
		return espErrInvalidArg
	}
	if length <= 0 {
		return espErrInvalidArg
	}
	return xfer(d, data, length, nil, 0)
}

// receive reads len(data) bytes from the device.
func receive(dev uintptr, data *uint8, length int32, timeoutMs int32) int32 {
	d := devByHandle(dev)
	if d == nil {
		return espErrInvalidArg
	}
	if length <= 0 {
		return espErrInvalidArg
	}
	return xfer(d, nil, 0, data, length)
}

// transmitReceive writes then reads with a repeated start in between.
func transmitReceive(dev uintptr, write *uint8, writeLen int32,
	read *uint8, readLen int32, timeoutMs int32) int32 {
	d := devByHandle(dev)
	if d == nil {
		return espErrInvalidArg
	}
	if writeLen < 0 || readLen < 0 || writeLen+readLen == 0 {
		return espErrInvalidArg
	}
	return xfer(d, write, writeLen, read, readLen)
}

// probe asks whether any device answers at addr: the address byte with no
// payload, so the only thing the bus can report is the ACK bit.
func probe(bus uintptr, addr uint32, timeoutMs int32) int32 {
	if addr > 0x7F {
		return espErrInvalidArg
	}
	idx := busIndex(bus)
	if idx < 0 {
		return espErrInvalidState
	}
	d := devState{bus: idx, addr: uint8(addr), speed: SpeedStandard}
	return xfer(&d, nil, 0, nil, 0)
}

// xfer runs one transaction on the device's port.
func xfer(d *devState, wr *uint8, wrLen int32, rd *uint8, rdLen int32) int32 {
	b := &baremetalBuses[d.bus]
	base := b.base
	budget := transferBudget(d.speed, wrLen+rdLen+1)

	if !waitIdle(base, &budget) {
		return espErrTimeout
	}
	programTiming(base, b.port, d.speed)
	resetFifos(base)
	mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
	// The timing writes above land in the controller's shadow registers;
	// CONF_UPGATE latches them, and must be set before TRANS_START.
	mmio.Set32(base+i2cRegCtr, ctrConfUpgate)

	t := transfer{base: base, port: b.port, addr: uint32(d.addr) << 1, budget: &budget, ctrShadow: b.ctr, speed: d.speed}

	// A write-only transfer (and the address-only probe) walks the write
	// phase and stops. Anything with a read goes through readPhase, which
	// merges the write payload into the read's command lists (see there).
	if rdLen == 0 {
		return t.writePhase(wr, wrLen)
	}
	return t.readPhase(wr, wrLen, rd, rdLen)
}

// transfer carries one transaction through its command lists.
type transfer struct {
	base      uintptr
	port      int32
	addr      uint32 // the 8-bit address byte, W bit clear
	budget    *budget
	ctrShadow uint32 // the bus's configured CTR value, see busState.ctr
	speed     uint32 // this transfer's bus speed, for the timing re-init
}

// segment is one command list: a few commands, the bytes to preload into the
// TX FIFO ahead of it, and how the list ends. A list that ends with STOP
// finishes the transaction; every other list ends with an END command, whose
// execution raises END_DETECT and leaves the bus held for the next list.
type segment struct {
	// Eight slots is the worst case: a done merged list is RESTART + payload
	// WRITE + RESTART + address WRITE + ACK-run READ + NACK-byte READ + STOP
	// (7), and run writes the trailing END one past the last command. The
	// controller has exactly 8 slots (COMD0..7).
	cmds  [8]uint32
	ncmd  int
	addr  uint32 // when nonzero, this address byte leads the FIFO fill
	tx    *uint8 // payload bytes to preload
	txn   int32
	addr2 uint32 // when nonzero, this address byte follows the payload in the FIFO
	last  bool
}

func (s *segment) add(cmd uint32) {
	s.cmds[s.ncmd] = cmd
	s.ncmd++
}

// run clears the previous list's completion bit, preloads the FIFO, writes the
// command list, starts the FSM and waits for this list's own completion
// signal. The clear happens first: it must not be interleaved with the FIFO
// load it is meant to precede.
func (s *segment) run(t *transfer) int32 {
	base := t.base
	mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
	// The previous list consumed its bytes; resetting both FIFO pointers
	// makes the next fill start at slot 0. At a list boundary the FIFOs are
	// empty, so nothing queued is lost (the RX side was drained right after
	// the last READ, the TX side was emptied by the last WRITE).
	resetFifos(base)
	if s.addr != 0 {
		putFifo(base, uint8(s.addr))
	}
	for i := int32(0); i < s.txn; i++ {
		putFifo(base, byteAt(s.tx, i))
	}
	if s.addr2 != 0 {
		putFifo(base, uint8(s.addr2))
	}
	for i := 0; i < s.ncmd; i++ {
		mmio.Write32(base+i2cRegComd0+uintptr(i)*4, s.cmds[i])
	}
	// The FSM walks the slots until it finds one that is not a command, so
	// every stale non-zero slot past the list is live ammunition: a leftover
	// WRITE or STOP from a longer list corrupts this one. A list that ends
	// with STOP needs no marker after it - the STOP is the terminator, and
	// zeroing the rest clears the stale slots. A bus-held list (ending in
	// END) takes the explicit END marker: IDF terminates its command lists
	// the same way, by writing hw_end_cmd right after the last command
	// (esp_driver_i2c/i2c_master.c: s_i2c_write_command). An END written
	// after a STOP breaks the emulator's merged reads (it fires a spurious
	// NACK after the transaction completed), so STOP lists get zeroes only.
	if s.last {
		for i := s.ncmd; i < 8; i++ {
			mmio.Write32(base+i2cRegComd0+uintptr(i)*4, 0)
		}
	} else {
		if s.ncmd < 8 {
			mmio.Write32(base+i2cRegComd0+uintptr(s.ncmd)*4, cmdOpEnd<<11)
		}
		for i := s.ncmd + 1; i < 8; i++ {
			mmio.Write32(base+i2cRegComd0+uintptr(i)*4, 0)
		}
	}
	// IDF order (i2c_ll_update then i2c_ll_start_trans): latch the new
	// command list into the controller first, then kick off the transfer.
	// Writing conf_upgate and trans_start in the same store races: the
	// emulator starts the list before the latched commands land and runs a
	// stale list ("command list ended without STOP/END").
	mmio.Write32(base+i2cRegCtr, t.ctrShadow|ctrConfUpgate)
	mmio.Write32(base+i2cRegCtr, t.ctrShadow|ctrTransStart)
	if s.last {
		return waitTransfer(t)
	}
	return waitSegment(t)
}

// writePhase clocks out the address byte and wrLen payload bytes, ending
// with STOP. A payload longer than one FIFO is split into several lists; the
// lists that keep the bus across a FIFO swap get their END from segment.run,
// which appends it to every list that is not the transaction's last.
func (t *transfer) writePhase(wr *uint8, wrLen int32) int32 {
	idx := int32(0)
	for first := true; ; first = false {
		var seg segment
		if first {
			seg.add(cmdOpRestart << 11)
			seg.addr = t.addr // write direction: W bit clear
		}
		// The address byte takes one of the FIFO's 32 slots on the list
		// that carries it.
		room := int32(fifoSegmentLen)
		if first {
			room--
		}
		n := wrLen - idx
		if n > room {
			n = room
		}
		if n > 0 {
			seg.tx = offsetByte(wr, idx)
			seg.txn = n
		}
		// The address byte counts as part of the write's byte_num: the
		// sum is an addition, not a bitwise OR (n | 1 silently drops the
		// address byte for odd n).
		byteNum := uint32(n)
		if first {
			byteNum++
		}
		seg.add(cmdOpWrite<<11 | cmdAckEn | byteNum)
		idx += n
		done := idx == wrLen
		if done {
			seg.add(cmdOpStop << 11)
			seg.last = true
		}
		if e := seg.run(t); e != 0 {
			return e
		}
		if done {
			return 0
		}
	}
}

// readPhase clocks in rdLen bytes, with the write payload (a register
// address, usually) merged into the same command lists: the list that
// carries the last write slice also carries the repeated start, the
// read-direction address byte and the first read commands, transmit-receive
// style. The esp-emu I2C model only delivers read data to such a list - a
// list of RESTART + address WRITE + READs standing alone completes without
// data (its regfile parts serve reads from a register pointer written in
// the same transaction) - and a merged list is what IDF builds for
// i2c_master_transmit_receive as well.
//
// A pure read (wr == nil) clocks the read-direction address byte from the
// transaction's first WRITE command, the classic current-address read.
// Real hardware accepts it; the emulator completes it without data, so an
// emu-verified caller must go through transmitReceive (Device.ReadReg).
//
// Every byte but the last is ACKed; the last is NACKed so the slave
// releases SDA before the STOP. ACK runs are FIFO-sized lists, and the
// final NACK byte rides in the last list whenever it fits (byte_num is
// capped at fifoSegmentLen-1 for it, the same bound the ESP-IDF driver
// uses).
func (t *transfer) readPhase(wr *uint8, wrLen int32, rd *uint8, rdLen int32) int32 {
	ackBytes := rdLen - 1
	widx := int32(0)

	// Write lists that precede the merged one: the first carries the
	// write-direction address byte (its byte_num counts it), later ones
	// carry payload only. Each ends held (END) - the bus stays in this
	// transaction. mergedWriteMax leaves room for the read-direction
	// address byte that shares the final list's FIFO (32 = up to 30 payload
	// + 2 address bytes).
	const mergedWriteMax = 30
	for wr != nil && wrLen-widx > mergedWriteMax {
		var seg segment
		first := widx == 0
		n := wrLen - widx
		if first {
			seg.add(cmdOpRestart << 11)
			seg.addr = t.addr // write direction: W bit clear
			n = fifoSegmentLen - 1
		} else if n > fifoSegmentLen {
			n = fifoSegmentLen
		}
		seg.tx = offsetByte(wr, widx)
		seg.txn = n
		byteNum := uint32(n)
		if first {
			byteNum++
		}
		seg.add(cmdOpWrite<<11 | cmdAckEn | byteNum)
		widx += n
		if e := seg.run(t); e != 0 {
			return e
		}
	}

	// The merged list: the last write slice (if any), then the repeated
	// start, the read-direction address byte and the first ACK run in one
	// command list. The RESTART + address WRITE + READ sequence always
	// shares a list with the READ commands; a pure read skips the write
	// slice and clocks the read-direction address from the list's first
	// WRITE command.
	ridx := int32(0)
	started := widx > 0 // preceding lists already sent RSTART + payload
	addrRSent := false
	for {
		var seg segment
		if !started {
			seg.add(cmdOpRestart << 11)
			started = true
		}
		if wr != nil && wrLen > widx {
			first := widx == 0
			if first {
				seg.addr = t.addr // write direction: W bit clear
			}
			n := wrLen - widx
			seg.tx = offsetByte(wr, widx)
			seg.txn = n
			// The address byte counts as part of this write's byte_num (an
			// addition, not a bitwise OR - n | 1 drops the address byte for
			// odd n).
			byteNum := uint32(n)
			if first {
				byteNum++
			}
			seg.add(cmdOpWrite<<11 | cmdAckEn | byteNum)
			widx += n
			// This list also opens the read direction: repeated start, then
			// the read-direction address byte clocked after the payload.
			seg.add(cmdOpRestart << 11)
			seg.addr2 = t.addr | 1 // read direction, clocked after the payload
			addrRSent = true
			seg.add(cmdOpWrite<<11 | cmdAckEn | 1)
		} else if !addrRSent {
			// The read direction has not been opened yet and this list has
			// no write slice: a pure read, or a write that exactly filled
			// preceding lists. Open it here - with a repeated start when the
			// write phase ended in an earlier held list.
			if widx > 0 {
				seg.add(cmdOpRestart << 11)
			}
			seg.addr = t.addr | 1 // read direction: R bit set
			addrRSent = true
			seg.add(cmdOpWrite<<11 | cmdAckEn | 1)
		}
		// Continuation lists (read direction already open, no write slice)
		// carry READ commands only: no address, no WRITE. The transaction
		// is still open (the previous list ended with END) and the slave
		// keeps clocking data out.
		n := ackBytes - ridx
		if n > fifoSegmentLen-1 {
			n = fifoSegmentLen - 1
		}
		done := ridx+n == ackBytes
		if n > 0 {
			seg.add(cmdOpRead<<11 | uint32(n))
		}
		if done {
			seg.add(cmdOpRead<<11 | cmdAckVal | 1)
			seg.add(cmdOpStop << 11)
			seg.last = true
		}
		if e := seg.run(t); e != 0 {
			return e
		}
		for i := int32(0); i < n; i++ {
			setByteAt(rd, ridx+i, uint8(mmio.Read32(t.base+i2cRegData)))
		}
		ridx += n
		if done {
			// The NACKed byte lands after the ACK run.
			setByteAt(rd, ridx, uint8(mmio.Read32(t.base+i2cRegData)))
			return 0
		}
	}
}

// offsetByte walks a caller-supplied buffer the way byteAt does.
func offsetByte(p *uint8, n int32) *uint8 {
	return (*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n)))
}

// waitTransfer polls the interrupt raw register for the final command list's
// outcome. waitSegment does the same for the lists that end in END instead.
func waitTransfer(t *transfer) int32 {
	return waitEvent(t, intTransComplete)
}

// waitSegment polls for the end of a command list that leaves the bus held
// for the next one. Both completion bits are accepted, the way the IDF
// interrupt handler treats MST_COMPLETE and END_DETECT alike
// (i2c_master.c: "int_mask & I2C_LL_INTR_MST_COMPLETE ||
// int_mask & I2C_LL_INTR_END_DETECT"): the TRM ties END_DETECT to the END
// command, and a chip or model may fold the two together for a list that
// returns the FSM to its wait state.
func waitSegment(t *transfer) int32 {
	return waitEvent(t, intTransComplete|intEndDetect)
}

// waitEvent polls INT_RAW until one of the failure bits or the expected
// completion bit shows up. Order matters: the controller can raise NACK
// together with a completion bit (the STOP still runs after a NACK), and the
// NACK is the diagnosis the caller needs.
func waitEvent(t *transfer, done uint32) int32 {
	base := t.base
	for {
		raw := mmio.Read32(base + i2cRegIntRaw)
		switch {
		case raw&intArbitrationLost != 0:
			mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
			settleBus(base)
			return espErrInvalidState
		case raw&intNack != 0:
			mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
			settleBus(base)
			return espErrNotFound
		case raw&intTimeOut != 0:
			mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
			fsmReset(t)
			return espErrTimeout
		case raw&done != 0:
			mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
			return 0
		}
		if t.budget.expired() {
			mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
			fsmReset(t)
			return espErrTimeout
		}
	}
}

// settleBus waits (bounded) for the bus to go idle before reporting an error.
// A NACK ends the data phase but the STOP the controller is already clocking
// out is still on the wire, and a caller that reconfigures pins or powers
// down right after the error would otherwise do it mid-transaction - the same
// unwind the ESP-IDF driver makes (i2c_master.c: after a NACK it loops on
// i2c_ll_is_bus_busy before quitting).
func settleBus(base uintptr) {
	idle := budget{expire: mmio.Cycles() + 5*cyclesPerMs, iters: 5*cyclesPerMs/pollCyclesPerIter + 64}
	waitIdle(base, &idle)
}

// fsmReset aborts a transfer the controller will not finish on its own, so
// the next transfer starts from a clean FSM. On a SoC with a hardware FSM
// reset (the S31) that is one CTR bit. The C3 has none - its HAL
// deliberately declines I2C_LL_SUPPORT_HW_FSM_RST ("FSM_RST only resets
// the FSM, not using it", esp_hal_i2c/esp32c3/include/hal/i2c_ll.h:32) - so
// there the whole block pulses through the SYSTEM reset bit and the
// bring-up runs again, the unwind IDF's s_i2c_hw_fsm_reset makes for the
// same chips (esp_driver_i2c/i2c_master.c:112-146): reset, master init,
// clock source + timing, interrupt state, bus clear.
func fsmReset(t *transfer) {
	base := t.base
	if i2cHwFsmRst {
		mmio.Set32(base+i2cRegCtr, ctrFsmRst)
		resetFifos(base)
		return
	}
	i2cResetPort(t.port)
	mmio.Write32(base+i2cRegCtr, t.ctrShadow)
	mmio.Write32(base+i2cRegIntEna, 0)
	mmio.Write32(base+i2cRegIntClr, 0xFFFFFFFF)
	programTiming(base, t.port, t.speed)
	resetFifos(base)
	clearBus(base)
}

// putFifo queues one byte for transmission.
func putFifo(base uintptr, v uint8) {
	mmio.Write32(base+i2cRegData, uint32(v))
}

// byteAt / setByteAt walk the caller's buffer. The public API hands the
// backend a *uint8 pointing at the first element of a Go slice, which lives in
// ordinary (GC-managed or static) memory, so a plain load or store is the right
// access - only the register window needs volatile.
func byteAt(p *uint8, i int32) uint8 {
	return *(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i)))
}

func setByteAt(p *uint8, i int32, v uint8) {
	*(*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i))) = v
}
