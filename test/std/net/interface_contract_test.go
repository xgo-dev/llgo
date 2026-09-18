package net_test

import (
	"errors"
	"net"
	"runtime"
	"testing"
)

// GOROOT's interface_stub.go exposes empty interface/address tables on both
// js/wasm and wasip1/wasm. Validate that contract instead of skipping the tests.
func checkWasmInterfaces(t *testing.T, ifaces []net.Interface, err error) bool {
	t.Helper()
	if runtime.GOARCH != "wasm" {
		return false
	}
	if err != nil || len(ifaces) != 0 {
		t.Fatalf("wasm Interfaces = %v, %v; want empty list, nil", ifaces, err)
	}
	return true
}

func checkMissingWasmInterface(t *testing.T, iface *net.Interface, err error) {
	t.Helper()
	var op *net.OpError
	if iface != nil || !errors.As(err, &op) || op.Op != "route" || op.Net != "ip+net" || op.Err == nil || op.Err.Error() != "no such network interface" {
		t.Fatalf("wasm interface lookup = %v, %v; want route/ip+net no-such-interface error", iface, err)
	}
}
