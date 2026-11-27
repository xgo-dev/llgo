package main

import _ "unsafe"

const LLGoFiles = "abi.c"

//go:linkname holdPayloadInC C.llgo_hardening_hold_payload
func holdPayloadInC(*payload) uint64

//go:linkname cHoldEntered C.llgo_hardening_hold_entered
func cHoldEntered() int32
