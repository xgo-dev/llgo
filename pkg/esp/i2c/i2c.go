// Package i2c drives ESP-IDF's new I2C master driver (i2c_master.h) from
// llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md). Every driver call here is synchronous and blocking
// with a driver-side timeout, so there is no mailbox: each method maps to
// one glue call that returns the raw esp_err code (0 = ESP_OK).
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

// DefaultTimeoutMs bounds every transfer: the glue passes it as the
// xfer_timeout_ms wait argument of each i2c_master_* call — in this IDF
// release the per-transfer parameter is the only timeout knob (the bus
// config carries no timeout field), so a stuck bus aborts with
// ESP_ERR_TIMEOUT instead of hanging the llgo scheduler forever.
const DefaultTimeoutMs = 1000

// ESP_ERR_INVALID_ARG: returned for Go-side misuse (nil or zero-value
// bus/device) before anything crosses into C.
const ErrInvalidArg = 0x102

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
// property, chosen when a Device is bound (see Bus.Device). The clock source
// and glitch filter stay in the C glue, and the internal pull-ups are left
// off: the IDF docs recommend external ones for real buses.
// On error the esp_err code is returned and the Bus is nil.
func NewBus(sda, scl Pin) (*Bus, int32) {
	var b Bus
	if e := newBus(&b.handle, int32(sda), int32(scl)); e != 0 {
		return nil, e
	}
	return &b, 0
}

// Device binds a 7-bit-addressed peripheral to the bus at the given SCL
// speed (the new driver's clock rate is per device, not per bus). On error
// the esp_err code is returned and the Device is nil.
func (b *Bus) Device(addr uint8, speedHz int32) (*Device, int32) {
	if b == nil || b.handle == 0 {
		return nil, ErrInvalidArg
	}
	var d Device
	if e := addDevice(b.handle, &d.handle, uint32(addr), uint32(speedHz)); e != 0 {
		return nil, e
	}
	return &d, 0
}

// Write transmits data (i2c_master_transmit). Empty data is a no-op.
func (d *Device) Write(data []byte) int32 {
	if d == nil || d.handle == 0 {
		return ErrInvalidArg
	}
	if len(data) == 0 {
		return 0
	}
	return transmit(d.handle, &data[0], int32(len(data)), DefaultTimeoutMs)
}

// Read receives len(data) bytes (i2c_master_receive). Empty data is a no-op.
func (d *Device) Read(data []byte) int32 {
	if d == nil || d.handle == 0 {
		return ErrInvalidArg
	}
	if len(data) == 0 {
		return 0
	}
	return receive(d.handle, &data[0], int32(len(data)), DefaultTimeoutMs)
}

// WriteRead performs a transmit followed by a repeated-start receive
// (i2c_master_transmit_receive), the usual register-read pattern.
// Whichever half is empty is skipped.
func (d *Device) WriteRead(write, read []byte) int32 {
	if d == nil || d.handle == 0 {
		return ErrInvalidArg
	}
	switch {
	case len(write) > 0 && len(read) > 0:
		return transmitReceive(d.handle, &write[0], int32(len(write)),
			&read[0], int32(len(read)), DefaultTimeoutMs)
	case len(read) > 0:
		return receive(d.handle, &read[0], int32(len(read)), DefaultTimeoutMs)
	case len(write) > 0:
		return transmit(d.handle, &write[0], int32(len(write)), DefaultTimeoutMs)
	}
	return 0
}

// Close removes the device from its bus (i2c_master_bus_rm_device).
func (d *Device) Close() int32 {
	if d == nil || d.handle == 0 {
		return ErrInvalidArg
	}
	e := rmDevice(d.handle)
	d.handle = 0
	return e
}

// Close deletes the bus (i2c_del_master_bus). All its devices must have
// been Closed first, or the driver returns an error.
func (b *Bus) Close() int32 {
	if b == nil || b.handle == 0 {
		return ErrInvalidArg
	}
	e := delBus(b.handle)
	b.handle = 0
	return e
}
