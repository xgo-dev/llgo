// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

package sig

//llgo:skip BoringCrypto FIPSOnly StandardCrypto

// The signature functions are pure no-ops on every LLGo target: BoringCrypto
// is never linked, so the linked-in signature only needs to exist. The bodies
// live here in Go; the stdlib declarations they replace are assembly-backed
// (dropped by the skip rule above) and the arch .s files are blanked via the
// sourcePatchAsmPkgs entry.
func BoringCrypto() {}

// FIPSOnly indicates that package crypto/tls/fipsonly is present.
func FIPSOnly() {}

// StandardCrypto indicates that standard Go crypto is present.
func StandardCrypto() {}
