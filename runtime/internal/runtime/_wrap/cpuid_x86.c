#include <cpuid.h>
#include <stddef.h>

/* Query CPUID leaf 1 EDX bit 26 (SSE2) and ECX bit 25 (AES-NI) on x86/x86-64. */
int llgo_cpu_has_aes(void)
{
    unsigned int eax = 0, ebx = 0, ecx = 0, edx = 0;
    if (__get_cpuid_max(0, NULL) < 1)
        return 0;
    if (!__get_cpuid(1, &eax, &ebx, &ecx, &edx))
        return 0;
    return (edx & (1u << 26)) != 0 && (ecx & (1u << 25)) != 0;
}
