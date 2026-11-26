// Package spi drives the ESP32-S31 general-purpose SPI master (SPI2_HOST,
// the GPSPI the IDF docs call the "general purpose" SPI) from llgo-compiled
// Go code.
//
// It follows the pkg/esp package template (see pkg/esp/README.md): a
// three-file Go split plus a thin C glue template. The S31 exposes one
// general-purpose SPI master (SPI1_HOST is the flash controller), so the C
// glue owns the bus and device handles in static storage and every binding
// here is a plain package-level function whose esp_err result is wrapped as
// the family's shared error value (nil = ESP_OK, the code stays inspectable
// on the error).
//
// # Usage
//
//	if err := spi.Init(11, 12, 13, 10, spi.Freq10MHz); err != nil { ... }
//	n, err := spi.Transfer(cmd, resp)        // full duplex, one transaction
//	err = spi.Write([]byte{0x9F})            // MOSI only, CS toggles
//	err = spi.WriteRead(cmd, buf)            // write half, then read half
//	if err := spi.AcquireBus(); err != nil { ... }  // hold CS across calls
//	defer spi.ReleaseBus()
//	if err := spi.Deinit(); err != nil { ... }
//
// # Full-duplex semantics
//
// A SPI master always clocks a byte on MOSI for every byte it samples on
// MISO, so the three data operations differ in what they do with the two
// lines and in how many transactions they use:
//
//   - Transfer(tx, rx) is ONE full-duplex transaction of
//     n = max(len(tx), len(rx)) bytes: tx is clocked out (zero-padded to n
//     bytes - a shorter tx costs one small heap allocation, or pass a
//     pre-padded buffer to avoid it), rx receives the first len(rx) of the
//     n bytes sampled. This is the shape for devices that talk while
//     listening (full-duplex streams, status-during-transfer).
//   - Write(tx) is ONE MOSI-only transaction; MISO samples are discarded
//     (the driver skips the MISO phase).
//   - WriteRead(tx, rx) is TWO back-to-back transactions: the tx bytes
//     clocked out, then len(rx) bytes received with the MOSI line idle.
//     This is the register-read shape (send a command, clock out the
//     answer). CS deasserts between the two halves unless the bus is held
//     by AcquireBus - devices that require CS low across the whole
//     command+response sequence (most flash-like devices) must wrap the
//     call in AcquireBus/ReleaseBus.
//
// Transfer with a nil tx is a receive-only transaction (MOSI idles);
// Transfer with a nil rx is a transmit-only one (MISO discarded). A call
// whose buffers are both empty is a no-op.
//
// # CS control
//
// By default CS (when Init got a cs pin >= 0) is hardware-driven: asserted
// at the start and deasserted at the end of every transaction. AcquireBus
// takes the driver's bus-acquire lock and pins CS: from the first transfer
// after the acquire, CS stays asserted across every transfer until
// ReleaseBus, which drops CS with one final zero-phase transaction before
// handing the bus back. On a bus held this way, Transfer/Write/WriteRead
// compose into arbitrary multi-transaction sequences under one CS select -
// the classic erase/page-program sequences of flash-class devices.
//
// # DMA
//
// The bus always initializes with DMA enabled (SPI_DMA_CH_AUTO), so
// transfers are not capped at the 64-byte hardware FIFO: the driver's
// default ceiling is 4092 bytes per transaction (bus max_transfer_sz
// default; a larger transaction reports ESP_ERR_INVALID_ARG). Caller
// slices flow straight into the driver: IDF v6.3 automatically stages
// DMA-unfriendly buffers (unaligned, internal/PSRAM placement) through
// temporary DMA-capable copies when the transaction descriptor does not
// opt out (SPI_TRANS_DMA_BUFFER_ALIGN_MANUAL is never set here), so the
// glue needs no alignment staging of its own. One property stays this
// binding's job: the SPI DMA engine writes MISO data in whole 32-bit
// words (the RX descriptor size is the byte length rounded up to four),
// so a rx slice whose length is not a multiple of four is transferred
// through one word-rounded staging copy - the hardware never writes past
// a Go buffer.
//
// # Frequency
//
// The S31 feeds GPSPI from a source the driver can pre-divide down to at
// most 80 MHz into the peripheral, so 80 MHz is the working-frequency
// ceiling. The binding accepts exactly the integer power-of-two divisions
// of that ceiling - Freq80MHz, Freq40MHz, Freq20MHz, Freq10MHz - the four
// rates the driver's integer divider reproduces exactly; anything else is
// refused with ErrBadFrequency instead of being silently rounded to a
// neighbor. SetFrequency rebinds the device at the new rate (remove +
// re-add, the driver has no in-place speed change that survives all
// paths), which also re-picks the CS slot.
//
// # Concurrency
//
// llgo runs goroutines cooperatively on one FreeRTOS task, so every glue
// call here is exclusive by construction: a polling transfer occupies the
// scheduler until it returns (real transactions complete in microseconds
// to a few milliseconds). The APIs are all synchronous; there is no
// callback, no ISR, no mailbox. AcquireBus/ReleaseBus are the transaction
// lock for manual CS sequencing, not a concurrency lock - do not call
// them from two goroutines that both expect to hold the bus.
//
// # Glue integration
//
// Copy glue/spi_glue.c into your project's main component (WHOLE_ARCHIVE
// required; add esp_driver_spi to the component REQUIRES); see
// pkg/esp/README.md for the two integration recipes.
package spi

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The package's internal error segment: 0x71xx, reserved for the spi
// binding's refusals (pkg/esp/README.md's segment table; 0x72 is temp,
// 0x73 touch, 0x74 espnow, 0x75 nvs, 0x76-0x78 bt, 0x79 opus, 0x7A
// wakenet, 0x7B ble, 0x7C+ the shared/meta segments). esp_err codes from
// the IDF driver pass through unchanged; these values are outside every
// real esp_err range so a refusal made here can never be mistaken for one
// made there.
//
// The not-init reporting is the glue's job: before Init has run its static
// device handle is NULL and every glue entry returns the segment value
// below, so a caller can tell "the package was never brought up" apart
// from the driver's own ESP_ERR_INVALID_STATE (0x103). ErrBadFrequency and
// ErrBusy are refused on both sides: the Go guards below catch the misuse
// before anything crosses into C, and the glue repeats the same segment
// values defensively.
const (
	errNotInit      = 0x7101
	errBadFrequency = 0x7102
	errBusy         = 0x7103
)

// ErrNotInit reports an spi call made before Init. It is the package's
// semantic sentinel - match it with errors.Is - and it keeps the glue's
// 0x7101 segment code on Error.Code as the diagnostic annotation, so
// errors.As and esperr.Code still expose the number the C pipeline
// reported and code-style checks against the segment value keep working.
var ErrNotInit = newSentinel("esp/spi: not initialized", errNotInit)

// ErrBadFrequency reports a frequency outside the four exact rates the
// binding accepts (Freq10MHz .. Freq80MHz). The Go guard rejects the call
// before anything crosses into C; the glue repeats the same segment value
// defensively. Like ErrNotInit it keeps the segment code on Error.Code.
var ErrBadFrequency = newSentinel("esp/spi: bad frequency", errBadFrequency)

// ErrBusy reports the bus transaction lock in a state the call cannot
// proceed from: AcquireBus on an already-held bus, ReleaseBus without a
// held bus, or SetFrequency while the bus is held (the rebind must not
// happen under a device that has the bus pinned). It keeps the glue's
// 0x7103 segment code on Error.Code.
var ErrBusy = newSentinel("esp/spi: bus busy", errBusy)

// newSentinel declares a semantic sentinel that keeps its historical
// segment code on Error.Code as the annotation (esperr.New is text-only;
// the glue still reports the code, and the README segment table pins it).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): the glue's segment-value refusals become the ErrNotInit /
// ErrBadFrequency / ErrBusy sentinels, every other code forwards as a bare
// esperr.Coded value - the driver's esp_err is the diagnostic, and the
// esp_err naming table lives in the wifi package.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case errNotInit:
		return ErrNotInit
	case errBadFrequency:
		return ErrBadFrequency
	case errBusy:
		return ErrBusy
	}
	return esperr.Coded(code, "")
}

// Pin identifies a GPIO by number (same numbering as the gpio package;
// each pkg/esp package is kept dependency-free, so this is a separate
// type). NoCS selects running the bus without a CS line (three-wire mode:
// the device is selected by other means, and AcquireBus/ReleaseBus stay
// pure transaction locks with no CS to pin).
type Pin int32

// NoCS is the cs argument for a bus without a CS line (three-wire mode).
const NoCS Pin = -1

// The four working frequencies the binding accepts: the 80 MHz peripheral
// ceiling divided by 1, 2, 4 and 8 - the power-of-two divisions the
// driver's integer clock divider reproduces exactly. Any other value is
// refused with ErrBadFrequency rather than silently rounded to a
// neighbor. (The driver itself rounds any request down to the nearest
// divider step; the binding refuses the inexact ones.)
const (
	Freq10MHz uint32 = 10_000_000
	Freq20MHz uint32 = 20_000_000
	Freq40MHz uint32 = 40_000_000
	Freq80MHz uint32 = 80_000_000
)

// validFreq reports whether hz is one of the four exact rates.
func validFreq(hz uint32) bool {
	switch hz {
	case Freq10MHz, Freq20MHz, Freq40MHz, Freq80MHz:
		return true
	}
	return false
}

// The GPIO range the glue accepts for the four pins, mirroring the gpio
// package's [0, 64) (the GPIO matrix routing the driver performs is
// undefined outside the chip's pad range, and the glue's own guard repeats
// the same bound). cs additionally accepts NoCS (-1).
func validPin(p Pin) bool { return p >= 0 && p < 64 }

// Init brings the bus up: initializes SPI2_HOST with DMA (SPI_DMA_CH_AUTO)
// on the given MOSI/MISO/SCLK pins, adds the one device this binding
// drives (SPI mode 0, hardware CS on cs, or no CS line when cs is NoCS)
// at clock rate hz, and leaves everything idle until the first transfer.
//
// hz must be one of Freq10MHz/Freq20MHz/Freq40MHz/Freq80MHz (ErrBadFrequency
// otherwise). A second Init without Deinit reports the driver's
// ESP_ERR_INVALID_STATE (0x103). The driver can also report ESP_ERR_NO_MEM
// (0x101, DMA channel or state memory), ESP_ERR_INVALID_ARG (0x102) for
// pins it cannot route, and ESP_ERR_NOT_FOUND (0x105) if its CS slots are
// exhausted (never with one device).
func Init(mosi, miso, sclk, cs Pin, hz uint32) error {
	if !validFreq(hz) {
		return errOf(errBadFrequency)
	}
	if !validPin(mosi) || !validPin(miso) || !validPin(sclk) ||
		(cs != NoCS && !validPin(cs)) {
		return errOf(0x102) // ESP_ERR_INVALID_ARG, before anything crosses into C
	}
	return errOf(init_(int32(mosi), int32(miso), int32(sclk), int32(cs), hz))
}

// SetFrequency rebinds the one device at a new clock rate (see Init for
// the accepted values and the driver errors this can report). The bus
// stays initialized; only the device is removed and re-added, so a
// transfer never runs concurrently (llgo schedules cooperatively) and the
// rebind is atomic from the caller's view. While the bus is held by
// AcquireBus this reports ErrBusy - release first.
func SetFrequency(hz uint32) error {
	if !validFreq(hz) {
		return errOf(errBadFrequency)
	}
	return errOf(setFrequency(hz))
}

// Transfer runs one full-duplex transaction of n = max(len(tx), len(rx))
// bytes (see the package doc for the exact semantics). It returns n, the
// number of bytes clocked. Both buffers empty is a no-op returning
// (0, nil). A transaction longer than the driver's per-transaction ceiling
// (the bus default is 4092 bytes with DMA) reports the driver's
// ESP_ERR_INVALID_ARG (0x102).
//
// Two staging rules keep the DMA engine inside the caller's buffers (see
// the package doc): a tx shorter than n is zero-padded through one fresh
// buffer, and a rx whose length is not a multiple of four goes through
// one word-rounded staging copy (pass multiples of four to skip it).
func Transfer(tx, rx []byte) (int, error) {
	n := len(tx)
	if len(rx) > n {
		n = len(rx)
	}
	if n == 0 {
		return 0, nil
	}
	if len(tx) < n { // zero-pad the MOSI half to n bytes
		pad := make([]byte, n)
		copy(pad, tx)
		tx = pad
	}
	rxp := rxStaged(rx)
	if e := transfer(&tx[0], int32(len(tx)), rxp, int32(len(rx))); e != 0 {
		return 0, errOf(e)
	}
	copyBackStaged(rx)
	return n, nil
}

// Write clocks out tx in one MOSI-only transaction (MISO discarded) and
// returns len(tx). Empty tx is a no-op returning (0, nil).
func Write(tx []byte) (int, error) {
	if len(tx) == 0 {
		return 0, nil
	}
	return len(tx), errOf(write(&tx[0], int32(len(tx))))
}

// WriteRead runs the register-read sequence: the tx bytes clocked out in
// one transaction, then len(rx) bytes received in a second one with the
// MOSI line idle. CS deasserts between the halves unless AcquireBus holds
// the bus (see the package doc) - flash-like devices that need CS low
// across the whole sequence must be driven under an acquire. Whichever
// half is empty is skipped; both empty is a no-op. The rx word-rounding
// staging of Transfer applies to the read half as well.
func WriteRead(tx, rx []byte) error {
	tlen, rlen := len(tx), len(rx)
	if tlen == 0 && rlen == 0 {
		return nil
	}
	var txp *byte
	if tlen > 0 {
		txp = &tx[0]
	}
	rxp := rxStaged(rx)
	if e := writeRead(txp, int32(tlen), rxp, int32(rlen)); e != 0 {
		return errOf(e)
	}
	copyBackStaged(rx)
	return nil
}

// The rx DMA staging: the SPI DMA engine writes MISO data in whole words
// (the RX descriptor size is the byte length rounded up to four), so a rx
// whose length is not a multiple of four is received into a fresh
// word-rounded buffer whose extra tail bytes belong to the staging
// allocation alone. rxStaged returns the buffer to hand the glue (the
// slice's own first-element pointer when no staging is needed) and
// copyBackStaged moves the received bytes into the caller's slice. Both
// are no-ops for nil/empty rx and for word-multiple lengths.
var s_stagedRx []byte

func rxStaged(rx []byte) *byte {
	s_stagedRx = nil
	if len(rx) == 0 || len(rx)%4 == 0 {
		return rxPtr(rx)
	}
	staged := make([]byte, len(rx)+4-len(rx)%4)
	s_stagedRx = staged
	return &staged[0]
}

func copyBackStaged(rx []byte) {
	if s_stagedRx != nil {
		copy(rx, s_stagedRx)
		s_stagedRx = nil
	}
}

// rxPtr returns the Go slice's first-element pointer in a form the
// boundary rules allow (nil for an empty slice), keeping the
// integer-and-pointer-only signature discipline.
func rxPtr(rx []byte) *byte {
	if len(rx) == 0 {
		return nil
	}
	return &rx[0]
}

// ActualFreqKHz reports the working frequency the driver's clock divider
// actually produces for the device, in kHz (spi_device_get_actual_freq).
// The configured rate is exact for the four accepted values, so this
// echoes the configuration - the call exists for diagnostics and for
// probes to pin the driver's clock math.
func ActualFreqKHz() (int32, error) {
	var khz int32
	if e := actualFreq(&khz); e != 0 {
		return 0, errOf(e)
	}
	return khz, nil
}

// takes the driver's bus-acquire lock and pins CS (when the bus
// has a CS line): from the first transfer after this call, CS stays
// asserted across every transfer until ReleaseBus. While held, other
// would-be masters of the peripheral are locked out (there are none in an
// llgo program - the lock is what makes the driver allow the CS-pinned
// transactions), and SetFrequency reports ErrBusy. A second AcquireBus on
// a held bus reports ErrBusy. The underlying lock wait is the driver's
// portMAX_DELAY - it refuses finite waits outright - and it is
// uncontended in an llgo program (one FreeRTOS task, one device), so it
// returns immediately in practice.
func AcquireBus() error {
	return errOf(acquireBus())
}

// ReleaseBus ends an AcquireBus session: it drops CS with one final
// zero-phase transaction (deasserting whatever the held sequence left
// asserted) and hands the bus back. ReleaseBus without a held bus reports
// ErrBusy.
func ReleaseBus() error {
	return errOf(releaseBus())
}

// Deinit tears the bus down: removes the device (its CS slot goes back to
// the driver) and frees the bus with its DMA channel. After a successful
// Deinit the package is back at the pre-Init state and Init can run again.
// Deinit while the bus is held reports ErrBusy (release first); Deinit
// before Init reports ErrNotInit (0x7101) from the glue.
func Deinit() error {
	return errOf(deinit())
}
