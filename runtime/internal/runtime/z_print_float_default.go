//go:build !llgo || !llgo.esp32c3.printf_float

package runtime

func requirePrintfFloat() {}
