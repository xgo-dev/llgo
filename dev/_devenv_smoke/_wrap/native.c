#include <gc.h>
#include <ffi.h>
#include <openssl/crypto.h>
#include <sqlite3.h>
#include <uv.h>
#include <zlib.h>

int llgo_check_native_dependencies(void) {
    ffi_cif cif;
    if (GC_malloc(16) == NULL) return 1;
    if (ffi_prep_cif(&cif, FFI_DEFAULT_ABI, 0, &ffi_type_void, NULL) != FFI_OK) return 2;
    if (OpenSSL_version_num() == 0) return 3;
    if (sqlite3_libversion_number() == 0) return 4;
    if (uv_version() == 0) return 5;
    if (zlibVersion()[0] == '\0') return 6;
    return 0;
}
