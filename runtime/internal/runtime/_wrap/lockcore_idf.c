/* Core-pin hook behind runtime.LockCore (lockcore_idf.go) on ESP-IDF targets.
 *
 * Weak default: returns 1 ("not provided") so a firmware without a
 * core-affinity glue still links, and runtime.LockCore reports an explicit
 * error instead of failing the IDF link. A project that implements the pin -
 * on the S31 IDF kernel the only path is writing the kernel-internal task
 * affinity field, since no public runtime setter exists (vTaskCoreAffinitySet
 * belongs to the Amazon SMP kernel, ESP32 targets only) - defines a strong
 * llgoLockCore in its IDF component. A strong definition overrides this weak
 * one; the whole-archive main-component pattern keeps the strong object in
 * the link unconditionally. See _demo/esp-idf-lockcore/main/lockcore_glue.c
 * for the reference implementation and its self-check discipline.
 *
 * Return contract with lockcore_idf.go: 0 = pinned, 1 = not provided by this
 * firmware, anything else = the implementing shim refused.
 *
 * The runtime's C sources are compiled without target-specific defines, so
 * the guard keys on the architecture ESP-IDF targets share; whether the file
 * is built at all is decided by the LLGoFiles chain in z_rt.go.
 */
#if defined(__riscv) && __riscv_xlen == 32

__attribute__((weak)) int llgoLockCore(int core)
{
    (void)core;
    return 1;
}

#endif
