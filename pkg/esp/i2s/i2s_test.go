package i2s

import (
	"errors"
	"testing"
)

// The host stub must be honest: every binding reports the unsupported code
// rather than a fabricated success, so a host build can never look like it
// moved data through an I2S port that does not exist.
func TestStubIsHonest(t *testing.T) {
	if _, err := NewChannel(true); !errors.Is(err, errOf(errUnsupported)) {
		t.Fatalf("NewChannel err = %v, want unsupported", err)
	}
	if _, err := NewChannelRole(true, RoleSlave); !errors.Is(err, errOf(errUnsupported)) {
		t.Fatalf("NewChannelRole err = %v, want unsupported", err)
	}
}

// Role is part of the channel-creation contract: it selects the clock role
// and cannot be changed afterwards, so the zero value must stay the master
// (the shape every existing caller relies on).
func TestRoleZeroIsMaster(t *testing.T) {
	var r Role
	if r != RoleMaster {
		t.Fatalf("zero Role = %d, want RoleMaster", r)
	}
	if RoleSlave == RoleMaster {
		t.Fatal("RoleSlave aliases RoleMaster")
	}
}
