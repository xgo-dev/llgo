//go:build linux || darwin

package gotest

import "testing"

func enterRecoverableFaultTest(t *testing.T) bool { return true }

func checkRecoveredFaultAddress(t *testing.T, err error, address *byte) {}
