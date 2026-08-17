package flow

import "github.com/xgo-dev/llgo/cl/_testdrop/interface_demand_fixedpoint/api"

//go:noinline
func Step(n int) int {
	// This creates the api.Second.Next demand, but Step itself is reached only
	// through the ordinary edge from dynamically kept Runner.Run.
	return api.UseSecond(Worker{N: n})
}

type Worker struct {
	N int
}

//go:noinline
func (w Worker) Next() int {
	// Keeping Worker.Next exposes another ordinary edge, which then reaches a
	// third interface method demand.
	return api.UseThird(Finisher{N: w.N})
}

//go:noinline
func (w Worker) Drop() int {
	panic("Worker.Drop should be unreachable")
}

type Finisher struct {
	N int
}

//go:noinline
func (f Finisher) Done() int {
	return finalValue(f.N)
}

//go:noinline
func (f Finisher) Drop() int {
	panic("Finisher.Drop should be unreachable")
}

//go:noinline
func finalValue(n int) int {
	return n + 1
}
