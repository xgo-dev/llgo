package streamlib

type ServerStream interface {
	Context() string
}

type BidiStreamingServer[Req, Res any] interface {
	ServerStream
}

type GenericServerStream[Req, Res any] struct {
	ServerStream
}
