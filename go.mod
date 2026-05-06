module github.com/goplus/llgo

go 1.24

require (
	github.com/goplus/cobra v1.9.12 //xgo:class
	github.com/goplus/gogen v1.22.0
	github.com/goplus/lib v0.3.1
	github.com/goplus/llgo/runtime v0.0.0-00010101000000-000000000000
	github.com/goplus/mod v0.20.2
	github.com/marcinbor85/gohex v0.0.0-20210308104911-55fb1c624d84
	github.com/mattn/go-tty v0.0.8
	github.com/qiniu/x v1.17.0
	github.com/sigurn/crc16 v0.0.0-20240131213347-83fcde1e29d1
	github.com/xgo-dev/llvm v0.9.0
	github.com/xgo-dev/plan9asm v0.3.0
	go.bug.st/serial v1.6.4
	go.yaml.in/yaml/v3 v3.0.4
	golang.org/x/mod v0.27.0
	golang.org/x/tools v0.36.0
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace github.com/goplus/llgo/runtime => ./runtime
