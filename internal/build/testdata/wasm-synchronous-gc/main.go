package main

var saved map[string][]int

//go:noinline
func retained(n int) []int {
	return []int{n}
}

//go:noinline
func churn(n int) int {
	held := retained(n)
	saved = make(map[string][]int)
	saved["keep"] = []int{41, n}
	// Retain one short-lived allocation at a time so the collector can run
	// while the other value stays reachable through the map.
	for i := range 60000 {
		saved["discard"] = make([]int, 256+i%10)
		saved["discard"][0] = i
	}
	return saved["keep"][0] + saved["keep"][1] + saved["discard"][0] + held[0]
}

func main() {
	if churn(1) != 60042 {
		println("live data lost")
		return
	}
	println("wasm synchronous GC ok")
}
