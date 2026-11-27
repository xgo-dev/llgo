// Package i2c drives an ESP I2C master bus from llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md). Every driver call here is synchronous and blocking
// with a driver-side timeout, so there is no mailbox: each method maps to
// one glue call whose esp_err result is wrapped as the family's shared
// error value (nil = ESP_OK, the code stays inspectable on the error).
//
// # Backends
//
// Which driver sits behind the API is a build-tag decision, and the API is
// identical on all three:
//
//   - llgo.idf (an ESP-IDF project): glue/i2c_glue.c, which calls the IDF
//     new I2C master driver (i2c_idf.go).
//   - llgo build -target esp32c3|esp32s31 with no IDF: the register-level
//     baremetal backend (i2c_baremetal.go) programs the controller's command
//     list directly, so a plain llgo binary runs hardware I2C with no IDF and
//     no FreeRTOS in the link.
//   - any other build (host, Wasm, a baremetal chip without a register
//     table): the honest stub, which reports errUnsupported instead of
//     faking a transfer (i2c_stub.go).
//
// # Usage
//
//	bus, e := i2c.NewBus(8, 9)
//	dev, e := bus.Device(0x50, 100000)       // 7-bit address, SCL speed for this device
//	e = dev.Write([]byte{0x00})               // transmit
//	e = dev.WriteRead([]byte{0x00}, reg)      // transmit_receive
//	e = dev.Read(reg)                         // receive
//
// Handles are opaque C pointers carried in Go as uintptr integers, so the
// GC never scans them; the Go<->C boundary stays integer-and-pointer only.
//
// # Glue integration
//
// Copy glue/i2c_glue.c into your project's main component (WHOLE_ARCHIVE
// required); see pkg/esp/README.md for the two integration recipes.
package i2c

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// DefaultTimeoutMs bounds every transfer: the glue passes it as the
// xfer_timeout_ms wait argument of each i2c_master_* call — in this IDF
// release the per-transfer parameter is the only timeout knob (the bus
// config carries no timeout field), so a stuck bus aborts with
// ESP_ERR_TIMEOUT instead of hanging the llgo scheduler forever.
const DefaultTimeoutMs = 1000

// The ESP_ERR_INVALID_ARG value (0x102): the code the Go-side guards
// mirror when they reject a nil or zero-value bus/device before anything
// crosses into C. It stays an internal constant so the value remains
// comparable inside the package.
const errInvalidArg = 0x102

// ErrInvalidArg reports Go-side misuse - a nil or zero-value bus/device
// handle - caught before anything crosses into C. It is the package's
// semantic sentinel - match it with errors.Is - and it keeps the
// ESP_ERR_INVALID_ARG value (0x102) on Error.Code as the diagnostic
// annotation, so code-style checks against 0x102 keep working. The guard
// call sites wrap it with esperr.Wrap to name the operation that hit it.
var ErrInvalidArg = newSentinel("esp/i2c: invalid argument", errInvalidArg)

// Other esp_err codes the package reports as sentinels, so a caller can name
// the condition instead of the number. Each keeps its ESP code on Error.Code,
// which means errors.Is also matches the bare code-era literal.
const (
	errInvalidState = 0x103 // ESP_ERR_INVALID_STATE
	errTimeout      = 0x107 // ESP_ERR_TIMEOUT
	errNotFound     = 0x105 // ESP_ERR_NOT_FOUND
)

// ErrTimeout reports a transfer the driver abandoned because the bus did not
// finish it in time - a slave holding SCL low, or an address nobody answers on
// a bus with no pull-ups. It carries ESP_ERR_TIMEOUT (0x107); the transport
// family's esperr.ErrTimeout carries the lwip ETIMEDOUT (116) instead, because
// they are different layers' idioms for the same condition.
var ErrTimeout = newSentinel("esp/i2c: transfer timeout", errTimeout)

// ErrNotFound reports a NACK: the address byte reached the bus and no device
// acknowledged it. That is the answer a scan records for an empty address, and
// the code the ESP-IDF driver reports for the same condition
// (ESP_ERR_NOT_FOUND, 0x105).
var ErrNotFound = newSentinel("esp/i2c: address not acknowledged", errNotFound)

// errBusy is the watchdog refusal: another transfer still owns the bus.
var errBusy = newSentinel("esp/i2c: bus busy", errInvalidState)

// newSentinel declares a semantic sentinel that keeps its historical
// pipeline code on Error.Code as the annotation (esperr.New is text-only;
// the number stays inspectable for the drivers' own copies of the same
// condition).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// errOf turns one backend result into the error this package returns (nil for
// ESP_OK). The three conditions with a package sentinel map to it - so
// errors.Is names them while the ESP code stays on Error.Code - and every
// other code forwards as a bare esperr.Coded value: the driver's esp_err is
// the diagnostic, and the esp_err naming table lives in the wifi package.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case errTimeout:
		return ErrTimeout
	case errNotFound:
		return ErrNotFound
	case errInvalidState:
		return errBusy
	}
	return esperr.Coded(code, "")
}

// Pin identifies a GPIO by number (same numbering as the gpio package;
// each pkg/esp package is kept dependency-free, so this is a separate type).
type Pin int32

// Bus is an I2C master bus handle. Create it with NewBus; the zero-value
// type is not usable.
type Bus struct {
	handle uintptr // opaque C i2c_master_bus_handle_t
}

// Device is one peripheral address bound to a Bus (i2c_master_dev_handle_t).
type Device struct {
	handle uintptr
}

// NewBus installs a master bus on the given pins. The bus itself carries no
// clock speed: in the new I2C master driver the SCL frequency is a per-device
// property, chosen when a Device is bound (see Bus.Device). Both backends
// leave the internal pull-ups on (an idle I2C line must be high), which is
// enough for a short bench wire; the IDF docs recommend external ones for a
// real bus. The clock source and the SCL/SDA glitch filter stay at their
// reset values.
// On error the Bus is nil and the error carries the esp_err code.
func NewBus(sda, scl Pin) (*Bus, error) {
	var b Bus
	if e := newBus(&b.handle, int32(sda), int32(scl)); e != 0 {
		return nil, errOf(e)
	}
	return &b, nil
}

// Device binds a 7-bit-addressed peripheral to the bus at the given SCL
// speed (the new driver's clock rate is per device, not per bus). On error
// the esp_err code is returned and the Device is nil. Calling it on a nil
// or zero-value Bus reports ErrInvalidArg under the "i2c: bind device"
// operation.
func (b *Bus) Device(addr uint8, speedHz int32) (*Device, error) {
	if b == nil || b.handle == 0 {
		return nil, esperr.Wrap("i2c: bind device", ErrInvalidArg)
	}
	var d Device
	if e := addDevice(b.handle, &d.handle, uint32(addr), uint32(speedHz)); e != 0 {
		return nil, errOf(e)
	}
	return &d, nil
}

// Write transmits data (i2c_master_transmit). Empty data is a no-op.
// Calling it on a nil or zero-value Device reports ErrInvalidArg under the
// "i2c: write" operation. A payload larger than the controller's FIFO is
// split into FIFO-sized command lists inside the transfer - the wire sees one
// transaction with one START and one STOP - so there is no length limit at
// this level (the baremetal backend chunks; the IDF driver does the same).
func (d *Device) Write(data []byte) error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: write", ErrInvalidArg)
	}
	if len(data) == 0 {
		return nil
	}
	return errOf(transmit(d.handle, &data[0], int32(len(data)), DefaultTimeoutMs))
}

// Read receives len(data) bytes (i2c_master_receive). Empty data is a
// no-op. Calling it on a nil or zero-value Device reports ErrInvalidArg
// under the "i2c: read" operation.
func (d *Device) Read(data []byte) error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: read", ErrInvalidArg)
	}
	if len(data) == 0 {
		return nil
	}
	return errOf(receive(d.handle, &data[0], int32(len(data)), DefaultTimeoutMs))
}

// WriteRead performs a transmit followed by a repeated-start receive
// (i2c_master_transmit_receive), the usual register-read pattern.
// Whichever half is empty is skipped. Calling it on a nil or zero-value
// Device reports ErrInvalidArg under the "i2c: write transaction"
// operation.
func (d *Device) WriteRead(write, read []byte) error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: write transaction", ErrInvalidArg)
	}
	switch {
	case len(write) > 0 && len(read) > 0:
		return errOf(transmitReceive(d.handle, &write[0], int32(len(write)),
			&read[0], int32(len(read)), DefaultTimeoutMs))
	case len(read) > 0:
		return errOf(receive(d.handle, &read[0], int32(len(read)), DefaultTimeoutMs))
	case len(write) > 0:
		return errOf(transmit(d.handle, &write[0], int32(len(write)), DefaultTimeoutMs))
	}
	return nil
}

// Close removes the device from its bus (i2c_master_bus_rm_device).
// Idempotent, like the net package's conn Close: the first call performs
// the rm and consumes the device (the handle is zeroed whether or not the
// driver reports an error); any later call on the same Device - and a
// Close on a nil or zero-value Device - reports ErrClosed (esperr.ErrClosed)
// under the "i2c: close device" operation.
func (d *Device) Close() error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: close device", esperr.ErrClosed)
	}
	e := rmDevice(d.handle)
	d.handle = 0
	return errOf(e)
}

// Close deletes the bus (i2c_del_master_bus). All its devices must have
// been Closed first, or the driver reports an error. Idempotent, like the
// net package's conn Close, with the zeroed handle as the per-Bus closed
// mark: the first call consumes the handle (whether or not the driver
// reports an error), and any later call on the same Bus - and a Close on
// a nil or zero-value Bus - reports ErrClosed (esperr.ErrClosed) under
// the "i2c: close bus" operation. Other Bus instances are unaffected:
// the mark lives in each value's own handle field.
func (b *Bus) Close() error {
	if b == nil || b.handle == 0 {
		return esperr.Wrap("i2c: close bus", esperr.ErrClosed)
	}
	e := delBus(b.handle)
	b.handle = 0
	return errOf(e)
}

// WriteReg writes one 8-bit register value, the usual two-byte register write
// (i2c_master_transmit with a two-byte payload).
func (d *Device) WriteReg(reg, value uint8) error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: write register", ErrInvalidArg)
	}
	return errOf(transmit(d.handle, &[]uint8{reg, value}[0], 2, DefaultTimeoutMs))
}

// ReadReg reads len(buf) bytes starting at an 8-bit register address, the
// usual write-register-then-repeated-start-read pattern
// (i2c_master_transmit_receive). An empty buf is a no-op.
func (d *Device) ReadReg(reg uint8, buf []byte) error {
	if d == nil || d.handle == 0 {
		return esperr.Wrap("i2c: read register", ErrInvalidArg)
	}
	if len(buf) == 0 {
		return nil
	}
	return errOf(transmitReceive(d.handle, &reg, 1, &buf[0], int32(len(buf)), DefaultTimeoutMs))
}

// Scan probes every 7-bit address in [first, last] and returns the ones that
// acknowledged, in ascending order. Addresses outside 0x08..0x77 are reserved
// by the I2C specification and are skipped even when the range covers them.
//
// A probe is the address byte on its own: the bus reports exactly one thing,
// the ACK bit, so a scan disturbs nothing. A refused address is a short
// transaction - the NACK comes back as soon as the address byte has been
// clocked out - so an empty range scans fast; only a bus stuck low (no
// pull-ups, a slave holding SCL) walks into the transfer timeout.
//
// The call reports an error only when the bus itself is unusable (nil or
// closed Bus, or a backend failure); an address that does not answer is not an
// error, it is a gap in the result.
func (b *Bus) Scan(first, last uint8) ([]uint8, error) {
	if b == nil || b.handle == 0 {
		return nil, esperr.Wrap("i2c: scan", ErrInvalidArg)
	}
	if first > last {
		return nil, esperr.Wrap("i2c: scan", ErrInvalidArg)
	}
	var found []uint8
	for addr := first; ; addr++ {
		if addr >= 0x08 && addr <= 0x77 {
			switch e := probe(b.handle, uint32(addr), DefaultTimeoutMs); e {
			case 0:
				found = append(found, addr)
			case errNotFound, errInvalidState, errTimeout:
				// Nobody answered, or the bus is stuck: either way this
				// address is not a device.
			default:
				return found, errOf(e)
			}
		}
		if addr == last {
			break
		}
	}
	return found, nil
}
