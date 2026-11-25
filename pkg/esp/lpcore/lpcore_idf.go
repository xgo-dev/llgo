//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/lpcore_glue.c - the same pattern as pkg/esp/gpio/gpio_idf.go and
// pkg/esp/wifi/wifi_idf.go. The symbols stay undefined in llgo_out.o by
// design; the IDF component archive provides the definitions at the final
// link.
package lpcore

import _ "unsafe" // for go:linkname

//go:linkname load C.llgo_lpcore_load
func load(prog *byte, length int32) int32

//go:linkname run C.llgo_lpcore_run
func run(wakeupSource int32, timerSleepUs int32) int32

//go:linkname stop C.llgo_lpcore_stop
func stop() int32

//go:linkname mailboxSend C.llgo_lpcore_mailbox_send
func mailboxSend(msg int32) int32

//go:linkname mailboxRecv C.llgo_lpcore_mailbox_recv
func mailboxRecv(out *int32) int32

// vTaskDelay yields the CPU for ticks. MailboxRecv's poll loop calls it
// between quanta: a busy loop starves the idle tasks and trips the task
// watchdog, and holding the driver's blocking receive instead would freeze
// the whole fiber scheduler for the length of the C-level wait.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// gosched hands the fiber scheduler to the next runnable goroutine (the
// runtime's runtime.Gosched, reached by linkname like the wifi package's
// poller so the runtime package is not imported - importing it would pull
// its finalizer/reflect machinery into the image). MailboxRecv yields
// through it between quanta so other goroutines keep running.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()
