//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported and
// nowMs is a stub. Nothing here is meant to run on a host.
package net

// errUnsupported is what every stub below returns: it lies outside both the
// lwip errno range and the DNS h_errno range so it can never be mistaken
// for a real code.
const errUnsupported = 0x7F00

func netSocket(fd *int32) int32 { _ = fd; return errUnsupported }

func netConnect(fd int32, ip uint32, port int32) int32 {
	_, _, _ = fd, ip, port
	return errUnsupported
}

func netPoll(fd int32, wantWrite int32, timeoutMs int32, ready *int32) int32 {
	_, _, _, _ = fd, wantWrite, timeoutMs, ready
	*ready = 0
	return errUnsupported
}

func netSockError(fd int32, err *int32) int32 {
	_, _ = fd, err
	return errUnsupported
}

func netRecv(fd int32, buf *uint8, length int32, n *int32) int32 {
	_, _, _, _ = fd, buf, length, n
	return errUnsupported
}

func netSend(fd int32, buf *uint8, length int32, n *int32) int32 {
	_, _, _, _ = fd, buf, length, n
	return errUnsupported
}

func netShutdown(fd int32, how int32) int32 {
	_, _ = fd, how
	return errUnsupported
}

func netClose(fd int32) int32 { _ = fd; return errUnsupported }

func netBindListen(fd int32, port int32, backlog int32) int32 {
	_, _, _ = fd, port, backlog
	return errUnsupported
}

func netAccept(fd int32, connFd *int32) int32 {
	_, _ = fd, connFd
	return errUnsupported
}

func netResolve(host *uint8, hostLen int32, ip *uint32) int32 {
	_, _, _ = host, hostLen, ip
	return errUnsupported
}

func nowMs() uint32 { return 0 }
