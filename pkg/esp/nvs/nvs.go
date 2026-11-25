// Package nvs exposes ESP-IDF's NVS (non-volatile storage) key-value store
// to llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md). Every NVS call is synchronous and fast (an in-RAM
// cache hit, or a flash read on first access after boot), so there is no
// mailbox: each method maps to one glue call whose esp_err result is
// wrapped as the family's shared error value (nil = ESP_OK, the code stays
// inspectable on the error).
//
// # Usage
//
//	if e := nvs.Init(); e != nil { ... }        // nvs_flash_init, idempotent
//	store, e := nvs.Open("wifi")                // namespaces partition the key space
//	e = store.SetString("ssid", "home")         // stage in the NVS cache
//	e = store.Commit()                          // make it durable (see below)
//	ssid, e := store.GetString("ssid")          // read back
//	e = store.Delete("ssid")                    // erase one key
//	e = store.Commit()
//
// # Explicit Commit
//
// IDF's NVS stages every write in RAM and only writes flash on
// nvs_commit, which is where durability comes from. This API keeps that
// boundary honest: Set*/Delete return after touching the cache, and a
// Commit is what makes the batch survive power loss. An auto-commit layer
// was considered and rejected: committing per call multiplies page writes
// (flash wear) for a sequence of Sets that is one durable batch, and the
// one-line convenience is trivially the caller's (Set; Commit). The docs
// say "staged in the NVS cache" everywhere a call stops short of Commit.
//
// # Value shapes and limits
//
// Keys are Go strings of 1..15 bytes without a NUL byte (NVS's
// NVS_KEY_NAME_MAX_SIZE-1), enforced Go-side with ESP_ERR_INVALID_ARG.
// GetString/GetBlob read through the glue's static value buffer, so a
// stored string or blob longer than MaxValue bytes reads back as
// ErrTooLong - the binding's read cap, not an IDF condition; writes have
// no binding cap (IDF's own ESP_ERR_NVS_VALUE_TOO_LONG applies past what a
// page holds). Keys() lists the namespace's keys through a static table,
// refusing with ErrTooManyKeys past MaxKeys rather than truncating
// silently.
//
// # Error model
//
// Errors are *esperr.Error values. The esp domain passes raw esp_err
// through - the ESP_ERR_NVS_* family (0x1101..0x1110, named by ErrName)
// plus the generic codes - so code stays comparable and inspectable. The
// binding's own conditions - the two static staging buffers' capacity,
// refused instead of silently truncating - live in the 0x75xx segment,
// this package's own (see the segment plan below). The stub tag sits above
// everything:
//
//	esp domain     raw esp_err from the glue (NVS_* 0x11xx family, the
//	               generic ESP_ERR_* codes)
//	Go domain      0x75xx - this package's own capacity conditions
//	               (ErrTooLong, ErrTooManyKeys)
//	stub tag       0x7F00 - a host build (every binding reports it)
//
// The 0x75xx segment: the family's internal segments are 0x76xx bt,
// 0x77xx A2DP, 0x78xx reserved for further classic-audio profiles, 0x79xx
// opus, 0x7Axx wakenet, 0x7Bxx ble, 0x7Cxx the esperr meta segment, 0x7Dxx
// ws, 0x7Exx tls and 0x7Fxx the stub tag; 0x75xx is the lowest unclaimed
// segment.
//
// # Glue integration
//
// Copy glue/nvs_glue.c into your project's main component (WHOLE_ARCHIVE
// required); see pkg/esp/README.md for the two integration recipes. The
// glue allocates nothing (no malloc/free across the llgo link): the value
// buffer and the key table are static.
package nvs

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// The binding's capacity limits. They mirror the glue's static storage
// (LG_NVS_VALUE_CAP, LG_NVS_MAX_KEYS in nvs_glue.c) - the glue is the
// storage, these are the documented contract.
const (
	// MaxKeyLen is the longest key or namespace name: NVS_KEY_NAME_MAX_SIZE
	// (16) minus the terminator.
	MaxKeyLen = 15
	// MaxValue is the longest string or blob a Get reads back, in bytes.
	// Longer stored values read as ErrTooLong (the glue's static value
	// buffer is the bound). Writes are not capped by the binding.
	MaxValue = 512
	// MaxKeys is the most keys Keys() will return for one namespace; more
	// refuse with ErrTooManyKeys.
	MaxKeys = 32
)

// ErrInvalidArg is the code errOf wraps for Go-side misuse (a nil or
// closed Namespace, an empty / over-long / NUL-bearing key or namespace),
// refused before anything crosses into C; it stays a plain constant so the
// value remains comparable.
const ErrInvalidArg = 0x102

// The binding's own codes, the 0x75xx segment: capacity conditions of the
// glue's static staging buffers, which refuse (errors.Is-able) where
// truncating would lie.
var (
	// ErrTooLong reports a stored string or blob whose length the glue's
	// value buffer cannot carry (see MaxValue). The value stays intact in
	// NVS; store it under a shape the binding reads (SplitBlob-like
	// chunking is the caller's choice).
	ErrTooLong = &esperr.Error{Code: 0x7501, Name: "NVS_ERR_TOO_LONG"}
	// ErrTooManyKeys reports a namespace with more keys than the glue's
	// key table holds (see MaxKeys). Keys() refuses instead of returning a
	// silent prefix.
	ErrTooManyKeys = &esperr.Error{Code: 0x7502, Name: "NVS_ERR_TOO_MANY_KEYS"}
)

// errOf wraps a raw esp_err code as the error value this package returns
// (nil for ESP_OK), named through ErrName so a printed error reads like
// the wifi package's.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// ErrName maps the esp_err codes this pipeline can hit to their IDF names.
func ErrName(code int32) string {
	switch code {
	case 0:
		return "ESP_OK"
	case -1:
		return "ESP_FAIL"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case ErrInvalidArg:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x104:
		return "ESP_ERR_INVALID_SIZE"
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case 0x1101:
		return "ESP_ERR_NVS_NOT_INITIALIZED"
	case 0x1102:
		return "ESP_ERR_NVS_NOT_FOUND"
	case 0x1103:
		return "ESP_ERR_NVS_TYPE_MISMATCH"
	case 0x1104:
		return "ESP_ERR_NVS_READ_ONLY"
	case 0x1105:
		return "ESP_ERR_NVS_NOT_ENOUGH_SPACE"
	case 0x1108:
		return "ESP_ERR_NVS_REMOVE_FAILED"
	case 0x110B:
		return "ESP_ERR_NVS_INVALID_STATE"
	case 0x110D:
		return "ESP_ERR_NVS_NO_FREE_PAGES"
	case 0x110E:
		return "ESP_ERR_NVS_VALUE_TOO_LONG"
	case 0x1110:
		return "ESP_ERR_NVS_NEW_VERSION_FOUND"
	case 0x7501:
		return "NVS_ERR_TOO_LONG"
	case 0x7502:
		return "NVS_ERR_TOO_MANY_KEYS"
	case 0x7F00:
		return "NVS_ERR_UNSUPPORTED"
	}
	return "esp_err 0x" + hex6(uint32(code))
}

// hex6 formats the low 24 bits as fixed-width hex (esp_err codes fit).
func hex6(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 0; i < 6; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}

// Init initializes the NVS flash layer (nvs_flash_init on the default
// "nvs" partition). It is idempotent: calling it again on an initialized
// layer reports nil. On a partition the driver cannot mount - out of free
// pages, or written by a newer NVS format - the glue applies the IDF
// recovery recipe, nvs_flash_erase followed by another init. That erase
// WIPES EVERY NAMESPACE and runs only when nothing is readable anyway.
func Init() error {
	return errOf(flashInit())
}

// Deinit releases the NVS layer (nvs_flash_deinit). After it, Open and
// ReadStats report ESP_ERR_NVS_NOT_INITIALIZED until the next Init; live
// Namespace handles keep their cache but a Commit reports the same code.
func Deinit() error {
	return errOf(flashDeinit())
}

// Namespace is one NVS namespace opened read-write (nvs_open
// NVS_READWRITE). Namespaces partition the key space, so two components
// can both use a "ssid" key without colliding. Create one with Open; the
// zero-value type is not usable - every method on a nil or zero-value
// Namespace (or one after Close) reports ESP_ERR_INVALID_ARG, and Close
// itself reports esperr.ErrClosed, the i2c package's handle idiom.
type Namespace struct {
	handle uintptr // opaque nvs_handle_t (uint32 in IDF, GC-invisible)
	name   string  // the namespace's own name, for Keys' iterator
}

// Open opens (creating on first write if needed - NVS_READWRITE) the named
// namespace in the default "nvs" partition. The name obeys the key rules
// (1..MaxKeyLen bytes, no NUL); violations report ESP_ERR_INVALID_ARG
// without crossing into C. On error the Namespace is nil and the error
// carries the esp_err code.
func Open(namespace string) (*Namespace, error) {
	if !validName(namespace) {
		return nil, errOf(ErrInvalidArg)
	}
	var ns Namespace
	ns.name = namespace
	if e := openNs(unsafe.StringData(namespace), int32(len(namespace)), &ns.handle); e != 0 {
		return nil, errOf(e)
	}
	return &ns, nil
}

// Close releases the handle (nvs_close; the staged-but-uncommitted cache
// of this namespace is dropped - commit first). Idempotent, like the net
// package's conn Close: the first call performs the close and consumes the
// handle (zeroed whether or not anything went wrong); any later call on
// the same Namespace - and a Close on a nil or zero-value Namespace -
// reports ErrClosed (esperr.ErrClosed).
func (ns *Namespace) Close() error {
	if ns == nil || ns.handle == 0 {
		return esperr.ErrClosed
	}
	closeNs(ns.handle)
	ns.handle = 0
	return nil
}

// validName applies the NVS name rules to keys and namespaces alike:
// 1..MaxKeyLen bytes, no NUL byte (the C side terminates; a NUL would
// truncate the name).
func validName(s string) bool {
	if len(s) < 1 || len(s) > MaxKeyLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return false
		}
	}
	return true
}

// keyArgs validates a key and returns its boundary form (pointer into the
// string's bytes, length): the glue NUL-terminates into a 16-byte local
// before every nvs call. A nil pointer never crosses: validName has
// already refused the empty cases.
func keyArgs(key string) (*byte, int32, error) {
	if !validName(key) {
		return nil, 0, errOf(ErrInvalidArg)
	}
	return unsafe.StringData(key), int32(len(key)), nil
}

// GetU8 reads an 8-bit unsigned value (nvs_get_u8). An absent key reports
// ESP_ERR_NVS_NOT_FOUND - and so does a key stored under a different
// primitive shape: IDF's page hash-list indexes items by (namespace, type,
// key), so a mismatched-type read usually misses the hash and returns
// ESP_ERR_NVS_NOT_FOUND before the type check could report
// ESP_ERR_NVS_TYPE_MISMATCH (measured, IDF v6.3).
func (ns *Namespace) GetU8(key string) (uint8, error) {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return 0, e
	}
	if ns == nil || ns.handle == 0 {
		return 0, errOf(ErrInvalidArg)
	}
	var v uint8
	if e := getU8(ns.handle, kp, kl, &v); e != 0 {
		return 0, errOf(e)
	}
	return v, nil
}

// GetI32 reads a 32-bit signed value (nvs_get_i32). See GetU8 for the
// error shapes.
func (ns *Namespace) GetI32(key string) (int32, error) {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return 0, e
	}
	if ns == nil || ns.handle == 0 {
		return 0, errOf(ErrInvalidArg)
	}
	var v int32
	if e := getI32(ns.handle, kp, kl, &v); e != 0 {
		return 0, errOf(e)
	}
	return v, nil
}

// GetU64 reads a 64-bit unsigned value (nvs_get_u64). See GetU8 for the
// error shapes.
func (ns *Namespace) GetU64(key string) (uint64, error) {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return 0, e
	}
	if ns == nil || ns.handle == 0 {
		return 0, errOf(ErrInvalidArg)
	}
	var v uint64
	if e := getU64(ns.handle, kp, kl, &v); e != 0 {
		return 0, errOf(e)
	}
	return v, nil
}

// GetString reads a string value (nvs_get_str) as a fresh Go string. The
// glue runs the IDF two-phase read inside one call - a length query, then
// a copy into its static value buffer - and Go copies the bytes out, so
// nothing of C memory outlives the call. The length counts the stored
// string including IDF's terminator; a stored string longer than MaxValue
// bytes reports ErrTooLong with the value left intact.
func (ns *Namespace) GetString(key string) (string, error) {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return "", e
	}
	if ns == nil || ns.handle == 0 {
		return "", errOf(ErrInvalidArg)
	}
	var n int32
	var buf *byte
	if e := getStr(ns.handle, kp, kl, &n, &buf); e != 0 {
		return "", errOf(e)
	}
	if n > 0 {
		n-- // drop IDF's terminator from the Go string
	}
	return string(unsafe.Slice(buf, int(n))), nil
}

// GetBlob reads a binary value (nvs_get_blob) as a fresh byte slice. The
// read path is GetString's (one glue call, static buffer, immediate copy);
// the length is the blob's exact stored length. Same ErrTooLong bound.
func (ns *Namespace) GetBlob(key string) ([]byte, error) {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return nil, e
	}
	if ns == nil || ns.handle == 0 {
		return nil, errOf(ErrInvalidArg)
	}
	var n int32
	var buf *byte
	if e := getBlob(ns.handle, kp, kl, &n, &buf); e != 0 {
		return nil, errOf(e)
	}
	out := make([]byte, int(n))
	copy(out, unsafe.Slice(buf, int(n)))
	return out, nil
}

// SetU8 stages an 8-bit unsigned value (nvs_set_u8) in the namespace's
// NVS cache; Commit makes it durable.
func (ns *Namespace) SetU8(key string, v uint8) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	return errOf(setU8(ns.handle, kp, kl, v))
}

// SetI32 stages a 32-bit signed value (nvs_set_i32).
func (ns *Namespace) SetI32(key string, v int32) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	return errOf(setI32(ns.handle, kp, kl, v))
}

// SetU64 stages a 64-bit unsigned value (nvs_set_u64).
func (ns *Namespace) SetU64(key string, v uint64) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	return errOf(setU64(ns.handle, kp, kl, v))
}

// SetString stages a string value (nvs_set_str). IDF's API reads the value
// to a terminator, so the Go side appends one to a copy of the string; a
// value containing a NUL byte would truncate there and is refused with
// ESP_ERR_INVALID_ARG instead. No binding-side length cap (see the error
// model above).
func (ns *Namespace) SetString(key string, v string) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	for i := 0; i < len(v); i++ {
		if v[i] == 0 {
			return errOf(ErrInvalidArg)
		}
	}
	buf := make([]byte, len(v)+1) // the terminator IDF reads to
	copy(buf, v)
	return errOf(setStr(ns.handle, kp, kl, &buf[0]))
}

// SetBlob stages a binary value (nvs_set_blob) - the (pointer, length)
// shape, NUL bytes welcome. No binding-side length cap.
func (ns *Namespace) SetBlob(key string, v []byte) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	var vp *byte
	if len(v) > 0 {
		vp = &v[0]
	}
	return errOf(setBlob(ns.handle, kp, kl, vp, int32(len(v))))
}

// Delete erases one key (nvs_erase_key) in the namespace's cache; Commit
// makes the erasure durable. An absent key reports
// ESP_ERR_NVS_NOT_FOUND.
func (ns *Namespace) Delete(key string) error {
	kp, kl, e := keyArgs(key)
	if e != nil {
		return e
	}
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	return errOf(eraseKey(ns.handle, kp, kl))
}

// Commit flushes the namespace's staged writes to flash (nvs_commit).
// Without it nothing is durable; with it, the whole batch since the last
// Commit lands in one page write. See the package docs for why this API
// exposes IDF's boundary instead of hiding it behind an auto-commit.
func (ns *Namespace) Commit() error {
	if ns == nil || ns.handle == 0 {
		return errOf(ErrInvalidArg)
	}
	return errOf(commit(ns.handle))
}

// Keys lists the namespace's keys (nvs_entry_find / nvs_entry_info over
// the default partition, filtered to this namespace and NVS_TYPE_ANY).
// The glue copies the names into a static table that holds MaxKeys of
// them; a larger namespace reports ErrTooManyKeys rather than a silent
// prefix. A namespace that does not exist (opened but never written) has
// no entries: the iterator comes back ESP_ERR_NVS_NOT_FOUND.
func (ns *Namespace) Keys() ([]string, error) {
	if ns == nil || ns.handle == 0 {
		return nil, errOf(ErrInvalidArg)
	}
	if !validName(ns.name) {
		return nil, errOf(ErrInvalidArg)
	}
	var n int32
	var tbl *byte
	if e := listKeys(unsafe.StringData(ns.name), int32(len(ns.name)), &n, &tbl); e != 0 {
		return nil, errOf(e)
	}
	raw := unsafe.Slice(tbl, int(n)*(MaxKeyLen+1))
	out := make([]string, int(n))
	for i := range out {
		chunk := raw[i*(MaxKeyLen+1) : (i+1)*(MaxKeyLen+1)]
		for j, b := range chunk {
			if b == 0 {
				out[i] = string(chunk[:j])
				break
			}
		}
	}
	return out, nil
}

// Stats is a flash-wide observation (nvs_get_stats on the default
// partition): the entry ledger of the whole partition, not of one
// namespace. Read it with ReadStats. Entries are 32-byte slots; a variable-length string or blob
// occupies one entry per 32 bytes of payload. Available is what new
// entries can actually use (free minus the reserved tail every page
// keeps).
type Stats struct {
	Used       int32 // entries holding live data
	Free       int32 // never-written entries, reserved tail included
	Available  int32 // entries a new value can use (Free minus reserved)
	Total      int32 // all entries the partition provides
	Namespaces int32 // namespaces with at least one entry
}

// ReadStats reads the ledger. A partition that was never initialized
// reports ESP_ERR_NVS_NOT_INITIALIZED.
func ReadStats() (Stats, error) {
	var s Stats
	if e := getStats(&s.Used, &s.Free, &s.Available, &s.Total, &s.Namespaces); e != 0 {
		return Stats{}, errOf(e)
	}
	return s, nil
}
