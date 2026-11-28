//go:build (darwin || linux) && !baremetal && go1.27 && !go1.28

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package http

import "time"

// wroteRequest is called by Go 1.27's persistConn.readLoop before returning a
// connection to the idle pool. Keep the write-completion check from
// src/net/http/transport.go and reject bytes the read loop already buffered
// beyond the response. The check must precede handing the connection to a
// waiting request, which would allow the bytes to be mistaken for its reply.
func (pc *persistConn) wroteRequest() bool {
	if pc.br.Buffered() != 0 {
		return false
	}
	select {
	case err := <-pc.writeErrCh:
		return err == nil
	default:
	}

	// Allow the writer time to publish its result. An early response can
	// arrive while the request is still being written.
	t := time.NewTimer(maxWriteWaitBeforeConnReuse)
	defer t.Stop()
	select {
	case err := <-pc.writeErrCh:
		return err == nil
	case <-t.C:
		return false
	}
}
