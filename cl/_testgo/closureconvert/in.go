package main

type Request struct {
	Function RouteFunction
}

type RouteFunction func(*Request)

type Route struct {
	Function RouteFunction
}

type Container struct {
	router []Route
}

func (c *Container) HandleWithFilter() {
	_ = func(req *Request) {}
}

func InstrumentRouteFunc(routeFunc RouteFunction) {
}

func main() {
	InstrumentRouteFunc(func(req *Request) {})
	println("ok")
}
