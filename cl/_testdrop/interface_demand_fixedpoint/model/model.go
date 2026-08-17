package model

import "github.com/xgo-dev/llgo/cl/_testdrop/interface_demand_fixedpoint/flow"

type Runner struct{}

func NewRunner() Runner {
	return Runner{}
}

//go:noinline
func (Runner) Run() int {
	// This static call is only reachable after Runner.Run is dynamically kept
	// for the api.First.Run interface demand.
	return flow.Step(41)
}

//go:noinline
func (Runner) Drop() int {
	panic("Runner.Drop should be unreachable")
}
