// Deliberate crash glue for ./_testgo/earlypanicfault: a load from a
// permanently unmapped address raises a load access fault (mcause 5) the
// moment llgo_trigger_fault runs. 0xE0000004 sits outside every window of
// both esp32c3 and esp32s31, so neither chip nor emulator can alias it. A
// load is used instead of a store because the esp32c3 emulator silently
// accepts stores to unmapped addresses (they corrupt aliased RAM instead of
// faulting), while loads always fault.
void llgo_trigger_fault(void) {
    *(volatile int *)0xE0000004;
}
