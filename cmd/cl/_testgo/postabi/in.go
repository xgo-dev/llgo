// LITTEST: POST-ABI darwin/arm64 linux/amd64 wasip1/wasm
// CHECK: target triple = "
// DARWIN-ARM64-SAME: {{.*}}-apple-{{.*}}"
// LINUX-AMD64-SAME: {{.*}}-linux{{.*}}"
// WASIP1-WASM-SAME: wasm32-unknown-wasip1"
package main

type result struct {
	a, b, c int
}

// CHECK-LABEL: define void @main.values(
// CHECK-SAME: ptr sret(%main.result)
func values() result {
	return result{1, 2, 3}
}

func main() {
	_ = values()
}
