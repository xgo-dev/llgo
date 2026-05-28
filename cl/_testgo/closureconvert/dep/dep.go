package dep

type HTTPRequest struct{}

type ResponseWriter interface{}

type Handler interface {
	ServeHTTP(ResponseWriter, *HTTPRequest)
}

type HandlerFunc func(ResponseWriter, *HTTPRequest)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *HTTPRequest) {
	f(w, r)
}

type Request struct {
	Request       *HTTPRequest
	selectedRoute *Route
}

type Response struct {
	ResponseWriter
}

type RouteFunction func(*Request, *Response)

type Route struct {
	Function RouteFunction
}

type FilterChain struct {
	Filters []FilterFunction
	Target  RouteFunction
}

func (f *FilterChain) ProcessFilter(req *Request, resp *Response) {
	f.Target(req, resp)
}

type FilterFunction func(*Request, *Response, *FilterChain)

type Container struct {
	router           RouteSelector
	containerFilters []FilterFunction
}

type RouteSelector interface {
	SelectRoute([]*WebService, *HTTPRequest) (*WebService, *Route, error)
}

type WebService struct {
	routes []Route
}

func (c *Container) Handle(pattern string, handler Handler) {
}

func (c *Container) HandleWithFilter(pattern string, handler Handler) {
	f := func(httpResponse ResponseWriter, httpRequest *HTTPRequest) {
		if len(c.containerFilters) == 0 {
			handler.ServeHTTP(httpResponse, httpRequest)
			return
		}
		chain := FilterChain{Filters: c.containerFilters, Target: func(req *Request, resp *Response) {
			handler.ServeHTTP(resp, req.Request)
		}}
		chain.ProcessFilter(&Request{Request: httpRequest}, &Response{ResponseWriter: httpResponse})
	}
	c.Handle(pattern, HandlerFunc(f))
}
