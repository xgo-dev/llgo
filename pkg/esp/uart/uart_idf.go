//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/uart_glue.c - the same pattern as pkg/esp/gpio/gpio_idf.go and
// runtime/internal/runtime/idf_freertos.go. The symbols stay undefined in
// llgo_out.o by design; the IDF component archive provides the definitions
// at the final link.
package uart

import _ "unsafe" // for go:linkname

//go:linkname uartInit C.llgo_uart_init
func uartInit(port int32, txPin int32, rxPin int32, baud int32) int32

//go:linkname uartSetBaud C.llgo_uart_set_baud
func uartSetBaud(port int32, baud int32) int32

//go:linkname uartSetConfig C.llgo_uart_set_config
func uartSetConfig(port int32, wordLen int32, stopBits int32, parity int32) int32

//go:linkname uartWrite C.llgo_uart_write
func uartWrite(port int32, data *byte, length int32, outN *int32) int32

//go:linkname uartRead C.llgo_uart_read
func uartRead(port int32, buf *byte, length int32, timeoutMs int32, outN *int32) int32

//go:linkname uartAvailable C.llgo_uart_available
func uartAvailable(port int32, outN *int32) int32

//go:linkname uartFlushInput C.llgo_uart_flush_input
func uartFlushInput(port int32) int32

//go:linkname uartFlushOutput C.llgo_uart_flush_output
func uartFlushOutput(port int32, timeoutMs int32) int32

//go:linkname uartDeinit C.llgo_uart_deinit
func uartDeinit(port int32) int32
