//go:build llgo && baremetal && tinygo.riscv32 && llgo.idf

package runtime

// The SPIRAM budget of the collector (tinygogc/gc_segment_idf.go) is a pair of
// weak C symbols so that an application can override the budget with a strong
// definition. The tag deliberately omits !nogc: the symbols must exist as weak
// definitions even under nogc, where nothing references them, so that a
// project compiled with nogc can still define the budget without a link
// surprise. The C file is inert there (no references, no emitted code).
const psramIdfLLGoFiles = "; _wrap/psram_idf.c"
