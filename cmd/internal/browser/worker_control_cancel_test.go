// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package browser

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestWorkerSnapshotDrainsSurvivorsAfterTargetExit(t *testing.T) {
	const targets = 3
	failed := make(chan struct{})
	release := make(chan struct{})
	ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
		var commands [targets]cdpMessage
		for i := range commands {
			if wsjson.Read(ctx, conn, &commands[i]) != nil {
				return
			}
		}
		if wsjson.Write(ctx, conn, cdpMessage{
			ID: commands[0].ID, Error: &cdpError{Message: "Target closed"},
		}) != nil {
			return
		}
		close(failed)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		for _, command := range commands[1:] {
			if wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)}) != nil {
				return
			}
		}
	})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	control := &workerControl{}
	control.client.Store(client)
	control.state.Store(&workerSnapshot{
		Enabled: true, Ready: true, Paused: true, StopID: 1,
		Targets: targets, sessions: []string{"exiting", "page", "worker"},
	})
	session := &Session{}
	session.workers.Store(control)
	result := make(chan workerSnapshot, 1)
	go func() { result <- session.checkedWorkerState(ctx) }()
	select {
	case <-failed:
	case <-ctx.Done():
		t.Fatal("snapshot requests did not reach the debugger")
	}
	select {
	case state := <-result:
		t.Fatalf("snapshot returned before surviving requests drained: %+v", state)
	case <-time.After(50 * time.Millisecond):
	}
	// The helper's lifetime will close the connection after this test, so let
	// the server return all replies before checking the rejected snapshot.
	unblock()
	select {
	case state := <-result:
		if state.Paused {
			t.Fatal("accepted a snapshot containing an exited target")
		}
	case <-ctx.Done():
		t.Fatal("snapshot did not finish after all replies")
	}
}

// Block exactly one post-handshake socket write, after websocket has installed
// its context cancellation hook. No large payload or socket-buffer timing is
// needed to exercise cancellation during a frame write.
type gatedCDPConn struct {
	net.Conn
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (c *gatedCDPConn) Write(data []byte) (int, error) {
	if c.armed.CompareAndSwap(true, false) {
		close(c.entered)
		select {
		case <-c.release:
		case <-c.closed:
			return 0, net.ErrClosed
		}
	}
	return c.Conn.Write(data)
}

func (c *gatedCDPConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func TestCDPCancelDuringWritePreservesSharedConnection(t *testing.T) {
	t.Run("request", func(t *testing.T) { testCDPCancelDuringWrite(t, false) })
	t.Run("connection", func(t *testing.T) { testCDPCancelDuringWrite(t, true) })
}

func testCDPCancelDuringWrite(t *testing.T, cancelConnection bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	observed := make(chan cdpMessage, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var command cdpMessage
			if wsjson.Read(ctx, conn, &command) != nil {
				return
			}
			select {
			case observed <- command:
			case <-ctx.Done():
				return
			}
		}
	}))
	defer server.Close()
	var wire *gatedCDPConn
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		wire = &gatedCDPConn{
			Conn: conn, entered: make(chan struct{}),
			release: make(chan struct{}), closed: make(chan struct{}),
		}
		return wire, nil
	}}
	defer transport.CloseIdleConnections()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	releaseWrite := sync.OnceFunc(func() { close(wire.release) })
	defer releaseWrite()
	client := &cdpClient{ctx: ctx, conn: conn, pending: make(map[uint64]cdpPending), done: make(chan struct{})}
	wire.armed.Store(true)
	request, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	result := make(chan error, 1)
	go func() {
		result <- client.call(request, "exiting", "Runtime.getIsolateId", map[string]any{}, nil)
	}()
	select {
	case <-wire.entered:
	case <-ctx.Done():
		t.Fatal("request did not enter its socket write")
	}
	if cancelConnection {
		cancel()
		select {
		case <-wire.closed:
		case <-time.After(5 * time.Second):
			t.Fatal("connection shutdown did not interrupt its blocked write")
		}
	} else {
		cancelRequest()
		select {
		case <-wire.closed:
			t.Fatal("canceling one CDP request closed the shared transport")
		case <-time.After(50 * time.Millisecond):
		}
	}
	releaseWrite()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled call = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled request did not return")
	}
	client.mu.Lock()
	pending := len(client.pending)
	client.mu.Unlock()
	if pending != 0 {
		t.Fatalf("canceled call retains %d pending requests", pending)
	}
	if cancelConnection {
		return
	}
	if err := client.wake(ctx, "surviving"); err != nil {
		t.Fatalf("cannot send to surviving worker: %v", err)
	}
	for _, want := range []string{"exiting", "surviving"} {
		select {
		case command := <-observed:
			if command.SessionID != want {
				t.Fatalf("command = %+v, want session %s", command, want)
			}
		case <-ctx.Done():
			t.Fatalf("server did not receive command for %s", want)
		}
	}
}

func TestCDPCanceledRequestDoesNotPoisonOtherTargets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, cancelRequest := context.WithCancel(ctx)
	var rejected bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var command cdpMessage
			if wsjson.Read(ctx, conn, &command) != nil {
				return
			}
			if wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)}) != nil {
				return
			}
		}
	}))
	defer server.Close()
	client, err := newCDPClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.conn.CloseNow()
	cancelRequest()
	if err := client.call(request, "exiting", "Runtime.getIsolateId", map[string]any{}, nil); errors.Is(err, context.Canceled) {
		rejected = true
	}
	if !rejected {
		t.Fatal("an already canceled request was dispatched")
	}
	if err := client.call(ctx, "page", "Runtime.getIsolateId", map[string]any{}, nil); err != nil {
		t.Fatalf("canceled request poisoned a surviving target: %v", err)
	}
}
