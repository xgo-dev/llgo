//go:build !llgo.idf

// Host-build tests for the temperature sensor surface. A host build has no
// sensor and no glue, so nothing here claims hardware exists: the tests pin
// what the pure-Go layer owns - the default configuration (against the IDF
// default macro), the internal error segment, and the honest-stub contract -
// and that every driver binding reports errUnsupported instead of faking a
// sensor.
package temp

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestDefaults pins the default configuration to the IDF default macro the
// glue mirrors (TEMPERATURE_SENSOR_CONFIG_DEFAULT(-10, 80)).
func TestDefaults(t *testing.T) {
	o := DefaultOpts()
	if o.RangeMinC != -10 || o.RangeMaxC != 80 || o.AllowPD {
		t.Fatalf("DefaultOpts = %+v, want the TEMPERATURE_SENSOR_CONFIG_DEFAULT(-10, 80) values", o)
	}
}

// TestErrorSegment pins the binding's internal segment value: 0x7201 sits
// outside every real esp_err range (the drivers' codes top out around
// 0x10x/0x2xx/0x6xx), so a refusal made at the glue can never be mistaken
// for one made by the IDF driver.
func TestErrorSegment(t *testing.T) {
	if ErrNotInstalled != 0x7201 {
		t.Fatalf("ErrNotInstalled = 0x%X, want the temp segment value 0x7201", ErrNotInstalled)
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every binding reports errUnsupported (0x7F00) instead of faking
// a sensor, and both read shapes carry the same error.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	if err := Install(DefaultOpts()); !errors.Is(err, unsupported) {
		t.Fatalf("Install on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := Start(); !errors.Is(err, unsupported) {
		t.Fatalf("Start on a host build = %v, want errUnsupported", err)
	}
	if _, err := Read(); !errors.Is(err, unsupported) {
		t.Fatalf("Read on a host build = %v, want errUnsupported", err)
	}
	if _, err := ReadMilliC(); !errors.Is(err, unsupported) {
		t.Fatalf("ReadMilliC on a host build = %v, want errUnsupported", err)
	}
	if err := Stop(); !errors.Is(err, unsupported) {
		t.Fatalf("Stop on a host build = %v, want errUnsupported", err)
	}
	if err := Uninstall(); !errors.Is(err, unsupported) {
		t.Fatalf("Uninstall on a host build = %v, want errUnsupported", err)
	}
}
