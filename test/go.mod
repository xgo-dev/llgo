module github.com/xgo-dev/llgo/test

go 1.27.0

require (
	github.com/goplus/lib v0.7.1
	github.com/xgo-dev/llgo/cmd v0.0.0-00010101000000-000000000000
	github.com/xgo-dev/llgo/runtime v0.0.0-00010101000000-000000000000
	github.com/xgo-dev/llvm v0.10.4
	go.yaml.in/yaml/v3 v3.0.6
	golang.org/x/mod v0.43.0
	golang.org/x/sys v0.48.0
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-tty v0.0.8 // indirect
	github.com/xgo-dev/llgo v0.0.0-00010101000000-000000000000 // indirect
	github.com/xgo-dev/plan9asm v0.6.2 // indirect
	go.bug.st/serial v1.6.4 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
)

replace github.com/xgo-dev/llgo => ..

replace github.com/xgo-dev/llgo/cmd => ../cmd

replace github.com/xgo-dev/llgo/runtime => ../runtime

// v0.51.0 plus the generic-method wrapper fix; remove when upstream includes it.
replace golang.org/x/tools => github.com/xgo-dev/tools v0.51.1
