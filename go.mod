module github.com/xgo-dev/llgo

go 1.27.0

require (
	github.com/coder/websocket v1.8.15
	github.com/goplus/cobra v1.9.12 //xgo:class
	github.com/goplus/gogen v1.26.6
	github.com/goplus/lib v0.7.1
	github.com/goplus/mod v0.22.1
	github.com/mattn/go-tty v0.0.8
	github.com/qiniu/x v1.19.3
	github.com/xgo-dev/llgo/runtime v0.0.0-00010101000000-000000000000
	github.com/xgo-dev/llvm v0.10.2
	github.com/xgo-dev/plan9asm v0.6.2
	go.bug.st/serial v1.6.4
	go.yaml.in/yaml/v3 v3.0.6
	golang.org/x/mod v0.41.0
	golang.org/x/sys v0.48.0
	golang.org/x/tools v0.51.0
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sync v0.23.0 // indirect
)

replace github.com/xgo-dev/llgo/runtime => ./runtime

// v0.51.0 plus the generic-method wrapper fix; remove when upstream includes it.
replace golang.org/x/tools => github.com/xgo-dev/tools v0.51.1

// Intrinsic bindings from xgo-dev/llvm#60; remove this pin once available upstream.
replace github.com/xgo-dev/llvm => github.com/cpunion/llvm v0.9.4-0.20261107154507-3bd4a8b983df
