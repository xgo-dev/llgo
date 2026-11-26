/* Thin C glue between llgo-compiled Go code and the ESP-IDF NVS API.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there. The integration recipes and the boundary
 * rules (integer-and-pointer-only signatures, esp_err passed through as
 * int32, IDF macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * No malloc/free anywhere: NVS keeps its own driver state, and this glue
 * adds only two static staging buffers (the value read buffer and the key
 * table below) - capacities the Go side documents as MaxValue and MaxKeys.
 * Everything else lives on the stack in 16-byte name locals.
 *
 * Every nvs_* call here is synchronous (a cache hit or one flash read), so
 * no mailbox is needed. Outside an ESP-IDF build the whole body compiles
 * away, so stray globbing (hosted builds, IDE tasks, over-broad CMake file
 * lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdbool.h>
#include <stdint.h>
#include <string.h>

#include "nvs.h"
#include "nvs_flash.h"

/* The storage behind the Go contract (nvs.go): MaxValue and MaxKeys must
 * keep matching these, so the "binding capacity" errors cannot hide a
 * disagreement between the two files. */
#define LG_NVS_KEY_MAX      15
#define LG_NVS_VALUE_CAP    512
#define LG_NVS_MAX_KEYS     32

/* This package's own 0x75xx-segment codes (nvs.go): conditions with no IDF
 * equivalent - the two static staging buffers' capacity, refused instead of
 * silently truncating. */
#define LG_NVS_ERR_TOO_LONG       0x7501
#define LG_NVS_ERR_TOO_MANY_KEYS  0x7502

/* One stored value at a time: GetString/GetBlob read into it and the Go
 * side copies out before returning. Single-fiber use is the caller's
 * contract (see the pkg/esp scheduling notes). */
static uint8_t s_value[LG_NVS_VALUE_CAP];

/* Keys() results: up to LG_NVS_MAX_KEYS names of up to LG_NVS_KEY_MAX
 * bytes each, NUL-terminated, one contiguous table Go copies by row. */
static uint8_t s_keys[LG_NVS_MAX_KEYS][LG_NVS_KEY_MAX + 1];

/* Copy a Go string (pointer + length, not NUL-terminated) into a 16-byte
 * NUL-terminated name local. The Go side applies the same rules before
 * calling (validName in nvs.go); this stays as the honest C-side guard. */
static int32_t lg_nvs_name(const uint8_t *src, int32_t len, char *dst)
{
    if (len < 1 || len > LG_NVS_KEY_MAX) {
        return ESP_ERR_INVALID_ARG;
    }
    memcpy(dst, src, (size_t)len);
    dst[len] = '\0';
    return ESP_OK;
}

int32_t llgo_nvs_flash_init(void)
{
    esp_err_t e = nvs_flash_init();
    /* The IDF recovery recipe (same as the wifi glue): a partition the
     * driver cannot mount - out of free pages, or written by a newer NVS
     * format - is erased and initialized fresh. That erase WIPES EVERY
     * NAMESPACE; it only runs when nothing was readable anyway. */
    if (e == ESP_ERR_NVS_NO_FREE_PAGES || e == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        if ((e = nvs_flash_erase()) != ESP_OK) return e;
        e = nvs_flash_init();
    }
    return e;
}

int32_t llgo_nvs_flash_deinit(void)
{
    return nvs_flash_deinit();
}

int32_t llgo_nvs_open(const uint8_t *ns, int32_t nsLen, uintptr_t *outHandle)
{
    char name[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(ns, nsLen, name);
    if (e != ESP_OK) return e;
    nvs_handle_t h;
    e = nvs_open(name, NVS_READWRITE, &h);
    if (e == ESP_OK) *outHandle = (uintptr_t)h;
    return e;
}

void llgo_nvs_close(uintptr_t handle)
{
    nvs_close((nvs_handle_t)handle);
}

int32_t llgo_nvs_get_u8(uintptr_t h, const uint8_t *key, int32_t keyLen, uint8_t *out)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_get_u8((nvs_handle_t)h, k, out);
}

int32_t llgo_nvs_get_i32(uintptr_t h, const uint8_t *key, int32_t keyLen, int32_t *out)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_get_i32((nvs_handle_t)h, k, out);
}

int32_t llgo_nvs_get_u64(uintptr_t h, const uint8_t *key, int32_t keyLen, uint64_t *out)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_get_u64((nvs_handle_t)h, k, out);
}

/* The IDF read is two-phase (a NULL-buffer length query, then the read);
 * both phases run inside this one call, so one boundary crossing carries a
 * full Get and the length can never race the read. Strings over the value
 * buffer's capacity are refused with the binding's own code - the value
 * stays intact in NVS - rather than truncated or misreported as an IDF
 * condition. For strings the length includes IDF's terminator (Go drops
 * it); for blobs it is the exact stored length. */
int32_t llgo_nvs_get_str(uintptr_t h, const uint8_t *key, int32_t keyLen,
                         int32_t *outLen, const uint8_t **outBuf)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    size_t len = 0;
    e = nvs_get_str((nvs_handle_t)h, k, NULL, &len);
    if (e != ESP_OK) return e;
    if (len > LG_NVS_VALUE_CAP) return LG_NVS_ERR_TOO_LONG;
    e = nvs_get_str((nvs_handle_t)h, k, (char *)s_value, &len);
    if (e != ESP_OK) return e;
    *outLen = (int32_t)len;
    *outBuf = s_value;
    return ESP_OK;
}

int32_t llgo_nvs_get_blob(uintptr_t h, const uint8_t *key, int32_t keyLen,
                          int32_t *outLen, const uint8_t **outBuf)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    size_t len = 0;
    e = nvs_get_blob((nvs_handle_t)h, k, NULL, &len);
    if (e != ESP_OK) return e;
    if (len > LG_NVS_VALUE_CAP) return LG_NVS_ERR_TOO_LONG;
    e = nvs_get_blob((nvs_handle_t)h, k, s_value, &len);
    if (e != ESP_OK) return e;
    *outLen = (int32_t)len;
    *outBuf = s_value;
    return ESP_OK;
}

int32_t llgo_nvs_set_u8(uintptr_t h, const uint8_t *key, int32_t keyLen, uint8_t v)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_set_u8((nvs_handle_t)h, k, v);
}

int32_t llgo_nvs_set_i32(uintptr_t h, const uint8_t *key, int32_t keyLen, int32_t v)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_set_i32((nvs_handle_t)h, k, v);
}

int32_t llgo_nvs_set_u64(uintptr_t h, const uint8_t *key, int32_t keyLen, uint64_t v)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_set_u64((nvs_handle_t)h, k, v);
}

/* nvs_set_str has no length parameter - IDF reads the value to a
 * terminator - so the Go side appends one to a copy of the string and the
 * glue passes the pointer through (a NUL inside the value would truncate
 * there; Go refuses those before the call). */
int32_t llgo_nvs_set_str(uintptr_t h, const uint8_t *key, int32_t keyLen, const uint8_t *val)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_set_str((nvs_handle_t)h, k, (const char *)val);
}

int32_t llgo_nvs_set_blob(uintptr_t h, const uint8_t *key, int32_t keyLen,
                          const uint8_t *val, int32_t valLen)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_set_blob((nvs_handle_t)h, k, val, (size_t)valLen);
}

int32_t llgo_nvs_erase_key(uintptr_t h, const uint8_t *key, int32_t keyLen)
{
    char k[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(key, keyLen, k);
    if (e != ESP_OK) return e;
    return nvs_erase_key((nvs_handle_t)h, k);
}

int32_t llgo_nvs_commit(uintptr_t h)
{
    return nvs_commit((nvs_handle_t)h);
}

/* List the namespace's keys. The iterator walks the default partition's
 * entries filtered by namespace name and NVS_TYPE_ANY, so a namespace that
 * has no entries (opened but never written) returns ESP_ERR_NVS_NOT_FOUND
 * - the same "namespace does not exist" signal a read-only nvs_open gives.
 * A namespace larger than the static table refuses with the binding's own
 * code instead of returning a silent prefix. The partition label "nvs" is
 * IDF's default NVS partition and the one every pkg/esp partition table
 * ships. */
int32_t llgo_nvs_keys(const uint8_t *ns, int32_t nsLen,
                      int32_t *outCount, const uint8_t **outTable)
{
    char name[LG_NVS_KEY_MAX + 1];
    int32_t e = lg_nvs_name(ns, nsLen, name);
    if (e != ESP_OK) return e;
    nvs_iterator_t it = NULL;
    e = nvs_entry_find("nvs", name, NVS_TYPE_ANY, &it);
    if (e != ESP_OK) return e;
    int32_t n = 0;
    bool overflow = false;
    nvs_entry_info_t info;
    esp_err_t res = ESP_OK;
    while (it != NULL && res == ESP_OK) {
        nvs_entry_info(it, &info);
        if (n < LG_NVS_MAX_KEYS) {
            memcpy(s_keys[n], info.key, LG_NVS_KEY_MAX + 1);
            s_keys[n][LG_NVS_KEY_MAX] = '\0';
            n++;
        } else {
            overflow = true;
        }
        res = nvs_entry_next(&it);
    }
    nvs_release_iterator(it);
    if (overflow) return LG_NVS_ERR_TOO_MANY_KEYS;
    *outCount = n;
    *outTable = &s_keys[0][0];
    return ESP_OK;
}

int32_t llgo_nvs_stats(int32_t *used, int32_t *freeEntries,
                       int32_t *available, int32_t *total, int32_t *nsCount)
{
    nvs_stats_t st;
    esp_err_t e = nvs_get_stats(NULL, &st);
    if (e != ESP_OK) return e;
    *used = (int32_t)st.used_entries;
    *freeEntries = (int32_t)st.free_entries;
    *available = (int32_t)st.available_entries;
    *total = (int32_t)st.total_entries;
    *nsCount = (int32_t)st.namespace_count;
    return ESP_OK;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_nvs_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
