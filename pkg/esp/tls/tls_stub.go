//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported and
// nowMs is a stub. Nothing here is meant to run on a host.
package tls

// errUnsupported is what every stub below returns: it lies outside the TLS
// code segment (0x7E00), the lwip errno range and the DNS h_errno range, so
// it can never be mistaken for a real code. (The net package's stubs use
// the same value; the packages are independent, the number is simply a
// shared convention for "no device here".)
const errUnsupported = 0x7F00

func tlsNew(fd int32, caPEM *byte, caLen int32, host *byte, hostLen int32,
	verifyNone int32, minVersion int32, h *uintptr) int32 {
	_, _, _, _, _, _, _, _ = fd, caPEM, caLen, host, hostLen, verifyNone, minVersion, h
	return errUnsupported
}

func tlsVersion(h uintptr) *byte { _ = h; return nil }

func tlsCiphersuite(h uintptr) *byte { _ = h; return nil }

func tlsPeerCert(h uintptr, buf *byte, cap int32, outLen *int32) int32 {
	_, _, _, _ = h, buf, cap, outLen
	return errUnsupported
}

func tlsHandstep(h uintptr) int32 { _ = h; return errUnsupported }

func tlsRead(h uintptr, buf *byte, length int32, n *int32) int32 {
	_, _, _, _ = h, buf, length, n
	return errUnsupported
}

func tlsWrite(h uintptr, buf *byte, length int32, n *int32) int32 {
	_, _, _, _ = h, buf, length, n
	return errUnsupported
}

func tlsCloseNotify(h uintptr) int32 { _ = h; return errUnsupported }

func tlsFree(h uintptr) { _ = h }

func netSocket(fd *int32) int32 { _ = fd; return errUnsupported }

func netConnect(fd int32, ip uint32, port int32) int32 {
	_, _, _ = fd, ip, port
	return errUnsupported
}

func netResolve(host *byte, length int32, ip *uint32) int32 {
	_, _, _ = host, length, ip
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

func netShutdown(fd int32, how int32) int32 {
	_, _ = fd, how
	return errUnsupported
}

func netClose(fd int32) int32 { _ = fd; return errUnsupported }

func nowMs() uint32 { return 0 }
