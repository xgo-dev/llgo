/* S31 closures are written into executable DIRAM on the executing hart.
 * Synchronize its instruction stream after libffi emits a trampoline.
 * compiler-rt's generic baremetal RISC-V implementation aborts instead.
 */
void __wrap___clear_cache(void *start, void *end) {
  (void)start;
  (void)end;
  /* fence.i synchronizes all prior instruction-memory stores on this hart;
   * the ISA has no range-selective form. The target's conservative feature
   * list omits Zifencei, so enable it only for this verified instruction. */
  __asm__ volatile(".option push\n.option arch, +zifencei\nfence.i\n.option pop"
                   ::: "memory");
}
