// Package uart drives the ESP32-S31's UART serial ports from llgo-compiled
// Go code.
//
// It follows the pkg/esp package template (see pkg/esp/README.md): three-file
// Go split plus a thin C glue template. The IDF driver (driver/uart.h, the
// esp_driver_uart component) is port-numbered global state - there is no
// handle object, so every binding here is a plain package-level function
// keyed by a Port value, and the C glue tracks which ports have been Init'd
// in static storage. Every call runs synchronously (no event queue, no
// mailbox): Write enqueues into the driver's TX ring buffer (its ISR feeds
// the hardware FIFO), Read polls the driver's RX ring buffer bounded by an
// explicit millisecond timeout.
//
// # Hardware facts this API is shaped by
//
// The S31 exposes five UART ports (SOC_UART_NUM = 5, 128-byte hardware FIFOs
// each). UART0 is the console: the boot ROM and the IDF default console
// (CONFIG_ESP_CONSOLE_UART_DEFAULT) drive it, so Init on port 0 is refused
// outright with ErrConsoleReserved rather than fighting the console driver
// for the pins. UART1..UART4 are free for application use; the demo and the
// typical board wiring use UART1 (the constants name 0..4, the glue validates
// against the chip's own UART_NUM_MAX).
//
// The binding installs each port with a 256-byte RX ring buffer and a
// 256-byte TX ring buffer (the driver refuses anything up to the 128-byte
// FIFO length), and no event queue - the synchronous-polling shape this
// package documents. There is deliberately no flow-control, RS485 or wakeup
// surface; add those when a board needs them.
//
// # Usage
//
//	if err := uart.Init(uart.UART1, 17, 18, 115200); err != nil { ... }  // 8N1
//	n, err := uart.Write(uart.UART1, []byte("hello"))
//	if k, err := uart.Available(uart.UART1); k > 0 { ... }
//	buf := make([]byte, 64)
//	n, err := uart.Read(uart.UART1, buf, 200) // up to 200 ms
//	if err := uart.FlushInput(uart.UART1); err != nil { ... }
//	if err := uart.SetBaud(uart.UART1, 9600); err != nil { ... }
//	if err := uart.Deinit(uart.UART1); err != nil { ... }
//
// # Read semantics
//
// Read is the idiomatic Go shape: (n>0, nil) for bytes received (partial
// fills are normal - it returns whatever arrived within the budget), and
// (0, ErrTimeout) when the budget elapsed with nothing received. The budget
// is milliseconds, quantized by the driver to FreeRTOS ticks (10 ms at the
// default 100 Hz tick): timeoutMs=0 is the zero-wait poll, reporting
// ErrTimeout when the RX buffer is empty. timeoutMs<0 is refused with
// ErrBadValue. There is no EOF concept on a UART (the line never "closes"),
// so an empty read is always a timeout, never io.EOF.
//
// # Blocking and the scheduler
//
// Read and FlushOutput block at most their millisecond budget (the driver
// parks the calling task on a semaphore and yields the CPU - safe for the
// llgo fiber scheduler). Write enqueues into the driver's TX ring buffer
// and returns as soon as the bytes are enqueued - the driver's own ISR
// feeds the hardware FIFO from there - so a normal write never touches the
// wire clock; it only blocks while the ring has no room for the payload,
// which a live line at any baud rate drains faster than message-size writes
// fill. FlushOutput gives a bounded wait for the full drain (FIFO empty).
//
// # Glue integration
//
// Copy glue/uart_glue.c into your project's main component (WHOLE_ARCHIVE
// required; add esp_driver_uart to the component REQUIRES); see
// pkg/esp/README.md for the two integration recipes.
package uart

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The package's internal error segment: 0x78xx, reserved for the uart
// binding's refusals (pkg/esp/README.md's segment table). esp_err codes
// from the IDF driver pass through unchanged; these values are outside
// every real esp_err range so a refusal made here can never be mistaken
// for one made there.
//
// The not-initialized reporting is the glue's job: before Init has run its
// static installed table is empty and every glue entry returns the segment
// value below, so a caller can tell "the port was never brought up" apart
// from the driver's own ESP_ERR_INVALID_STATE (0x103, e.g. a second
// install) - and the glue guard doubles as the crash guard that keeps an
// uninitialized port's NULL driver object out of the driver's hands.
const (
	errConsoleReserved = 0x7801
	errNotInit         = 0x7802
	errBadValue        = 0x7803
)

// The package's sentinels - match them with errors.Is. Each keeps its
// segment code (ErrTimeout: the esp_err family's own timeout code, the
// value uart_wait_tx_done reports on an expired flush budget) on Error.Code
// as the diagnostic annotation, so errors.As and esperr.Code still expose
// the number the pipeline reported and a raw-code Error built anywhere from
// the same code matches through the errors.Is code path.
var (
	// ErrConsoleReserved reports an Init on port 0, the console UART. The
	// boot ROM and the IDF console own that port; the binding refuses it
	// before the driver is touched. Use UART1+ for application traffic.
	ErrConsoleReserved = newSentinel("esp/uart: console reserved", errConsoleReserved, "")
	// ErrNotInit reports a call on a port the glue has not brought up
	// (before Init, or after Deinit).
	ErrNotInit = newSentinel("esp/uart: not initialized", errNotInit, "")
	// ErrTimeout reports a Read budget that elapsed with no byte received
	// (the ESP_ERR_TIMEOUT domain, code 0x107).
	ErrTimeout = newSentinel("esp/uart: timeout", 0x107, "ESP_ERR_TIMEOUT")
	// ErrBadValue reports an argument this side refuses before the glue
	// runs (a negative timeout budget).
	ErrBadValue = newSentinel("esp/uart: bad value", errBadValue, "")
)

// newSentinel declares a semantic sentinel that keeps its segment code (and,
// where the code belongs to the shared esp_err family, its ErrName-style
// name) on the Error as the diagnostic annotation. esperr.New is text-only;
// the glue still reports the code, and the README segment table pins it.
func newSentinel(text string, code int32, name string) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	e.Name = name
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): the glue's refusals become the package sentinels, the esp_err
// timeout code becomes ErrTimeout, and every other code forwards as a bare
// esperr.Coded value - the driver's esp_err is the diagnostic, and the
// esp_err naming table lives in the wifi package.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case errConsoleReserved:
		return ErrConsoleReserved
	case errNotInit:
		return ErrNotInit
	case errBadValue:
		return ErrBadValue
	case 0x107:
		return ErrTimeout
	}
	return esperr.Coded(code, "")
}

// Port identifies a UART port by number. The S31 exposes five (SOC_UART_NUM
// = 5); the glue validates the range against the chip's own UART_NUM_MAX,
// so an out-of-range port reports the driver's ESP_ERR_INVALID_ARG (0x102)
// and never touches memory. UART0 is the console port - Init refuses it
// (ErrConsoleReserved); the application lives on UART1 and up.
type Port int32

const (
	UART0 Port = 0 // console port, refused by Init
	UART1 Port = 1 // the usual application port
	UART2 Port = 2
	UART3 Port = 3
	UART4 Port = 4
)

// WordLen selects a port's data-bit count; SetConfig takes the plain bit
// count (5..8) and the glue maps it onto the driver's encoding.
type WordLen int32

const (
	Word5 WordLen = 5
	Word6 WordLen = 6
	Word7 WordLen = 7
	Word8 WordLen = 8
)

// StopBits selects a port's stop-bit count; SetConfig takes 1 (one bit),
// 15 (1.5 bits) or 2, and the glue maps it onto the driver's encoding.
type StopBits int32

const (
	Stop1  StopBits = 1
	Stop15 StopBits = 15
	Stop2  StopBits = 2
)

// Parity selects a port's parity mode; SetConfig takes the zero value for
// "disabled" and the glue maps the rest onto the driver's encoding.
type Parity int32

const (
	ParityNone Parity = 0
	ParityEven Parity = 1
	ParityOdd  Parity = 2
)

// Config is the frame format SetConfig applies. Use DefaultConfig for the
// 8N1 the Init path installs.
type Config struct {
	// WordLen is the data-bit count: Word5..Word8. Default Word8.
	WordLen WordLen
	// StopBits is the stop-bit count: Stop1, Stop15 or Stop2. Default Stop1.
	StopBits StopBits
	// Parity is the parity mode: ParityNone, ParityEven or ParityOdd.
	// Default ParityNone.
	Parity Parity
}

// DefaultConfig returns the frame format Init installs (8 data bits, one
// stop bit, no parity).
func DefaultConfig() Config {
	return Config{WordLen: Word8, StopBits: Stop1, Parity: ParityNone}
}

// Init installs the driver on a port and brings it up at baud with the
// default 8N1 frame format on the given pins: a 256-byte RX ring buffer and
// a 256-byte TX ring buffer (see the package doc's blocking note), no event
// queue (this package polls). Pass -1 for a pin to leave the port's current
// mapping for that pin unchanged (the driver's UART_PIN_NO_CHANGE).
//
// Port 0 is the console and is refused with ErrConsoleReserved before the
// driver is touched. An out-of-range port, a non-positive baud or pins
// outside the chip's GPIO range report the driver's ESP_ERR_INVALID_ARG
// (0x102); a port already installed reports the driver's
// ESP_ERR_INVALID_STATE (0x103) - Deinit first to re-install.
func Init(p Port, txPin, rxPin int32, baud int32) error {
	if p == UART0 {
		return esperr.Wrap("uart: init", ErrConsoleReserved)
	}
	return esperr.Wrap("uart: init", errOf(uartInit(int32(p), txPin, rxPin, baud)))
}

// SetBaud changes a port's baud rate on the fly (the frame format stays as
// configured). Zero or negative baud is refused with the driver's
// ESP_ERR_INVALID_ARG. Before Init the glue reports ErrNotInit.
func SetBaud(p Port, baud int32) error {
	return esperr.Wrap("uart: set baud", errOf(uartSetBaud(int32(p), baud)))
}

// SetConfig applies a frame format to a running port. A value outside the
// WordLen/StopBits/Parity tables is refused with the driver's
// ESP_ERR_INVALID_ARG before anything is applied; before Init the glue
// reports ErrNotInit.
func SetConfig(p Port, c Config) error {
	return esperr.Wrap("uart: set config", errOf(uartSetConfig(int32(p), int32(c.WordLen), int32(c.StopBits), int32(c.Parity))))
}

// Write enqueues data into the port's driver TX ring buffer (the driver's
// ISR feeds the hardware FIFO from there) and returns the byte count written
// (all of data). An empty slice returns (0, nil) without touching the port.
// Before Init the glue reports ErrNotInit; a driver-level failure reports
// (0, err).
func Write(p Port, data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	var n int32
	if err := errOf(uartWrite(int32(p), &data[0], int32(len(data)), &n)); err != nil {
		return 0, esperr.Wrap("uart: write", err)
	}
	return int(n), nil
}

// Read receives into buf, waiting up to timeoutMs for data. It returns
// (n, nil) with the bytes received - a partial fill is normal, the budget
// is not consumed waiting for a full buf - and (0, ErrTimeout) when the
// budget elapsed with nothing received. timeoutMs is quantized to FreeRTOS
// ticks (10 ms at the default 100 Hz), so timeoutMs=0 is the zero-wait
// poll; a negative budget is refused with ErrBadValue. An empty buf
// returns (0, nil) without touching the port. Before Init the glue reports
// ErrNotInit.
func Read(p Port, buf []byte, timeoutMs int32) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	if timeoutMs < 0 {
		return 0, esperr.Wrap("uart: read", ErrBadValue)
	}
	var n int32
	if err := errOf(uartRead(int32(p), &buf[0], int32(len(buf)), timeoutMs, &n)); err != nil {
		return 0, esperr.Wrap("uart: read", err)
	}
	if n == 0 {
		return 0, esperr.Wrap("uart: read", ErrTimeout)
	}
	return int(n), nil
}

// Available reports how many received bytes sit in the port's driver RX
// ring buffer right now (a Read of that size returns without waiting).
// Before Init the glue reports ErrNotInit.
func Available(p Port) (int, error) {
	var n int32
	if err := errOf(uartAvailable(int32(p), &n)); err != nil {
		return 0, esperr.Wrap("uart: available", err)
	}
	return int(n), nil
}

// FlushInput discards everything in the port's RX ring buffer and hardware
// FIFO - the resync point after a garbage burst. Before Init the glue
// reports ErrNotInit.
func FlushInput(p Port) error {
	return esperr.Wrap("uart: flush input", errOf(uartFlushInput(int32(p))))
}

// FlushOutput waits up to timeoutMs for the port's hardware FIFO to drain
// to the wire (a bounded uart_wait_tx_done). An expired budget reports the
// driver's ESP_ERR_TIMEOUT, surfaced as ErrTimeout; a negative budget is
// refused with ErrBadValue. Before Init the glue reports ErrNotInit.
func FlushOutput(p Port, timeoutMs int32) error {
	if timeoutMs < 0 {
		return esperr.Wrap("uart: flush output", ErrBadValue)
	}
	return esperr.Wrap("uart: flush output", errOf(uartFlushOutput(int32(p), timeoutMs)))
}

// Deinit uninstalls the port's driver and releases its buffers; the port
// returns to the pre-Init state and Init can run again (the driver reports
// ESP_ERR_INVALID_STATE for a second install, ErrNotInit for every call
// here after Deinit). Before Init the glue reports ErrNotInit.
func Deinit(p Port) error {
	return esperr.Wrap("uart: deinit", errOf(uartDeinit(int32(p))))
}
