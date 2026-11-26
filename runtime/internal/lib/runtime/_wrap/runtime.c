#include <stdint.h>

#if defined(__APPLE__)
// The patched runtime provides os.runtime_args even when package os is not
// linked. Darwin's external linker diagnoses its reference to
// os.executablePath before dead stripping the unused function. Provide a weak,
// zero-valued Go string so runtime-only shared/static libraries remain
// linkable; package os supplies the strong definition when it is present.
struct llgo_go_string {
    const char *p;
    intptr_t n;
};
struct llgo_go_string llgo_os_executable_path __asm("_os.executablePath")
    __attribute__((weak));
#else
/* Keep this translation unit non-empty on non-Darwin Unix builds. */
typedef int llgo_runtime_c_nonempty;
#endif
