package main

import (
	"github.com/goplus/llgo/cl/_testgo/closureconvert/dep"
)

func InstrumentRouteFunc(routeFunc dep.RouteFunction) dep.RouteFunction {
	return dep.RouteFunction(func(req *dep.Request, response *dep.Response) {
		routeFunc(req, response)
	})
}

func main() {
	InstrumentRouteFunc(func(req *dep.Request, response *dep.Response) {})
	println("ok")
}
