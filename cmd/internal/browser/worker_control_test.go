// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package browser

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func workerTestClient(t *testing.T, serve func(context.Context, *websocket.Conn)) (context.Context, *cdpClient) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		serve(ctx, conn)
		<-ctx.Done()
	}))
	client, err := newCDPClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		cancel()
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.conn.CloseNow()
		cancel()
		server.Close()
	})
	return ctx, client
}

func TestWorkerSnapshotDrainsTargetsConcurrently(t *testing.T) {
	const targets = 3
	ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
		var commands [targets]cdpMessage
		// No reply until every request arrives: a serial drain would stall.
		for i := range commands {
			if wsjson.Read(ctx, conn, &commands[i]) != nil {
				return
			}
		}
		for _, command := range commands {
			_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)})
		}
	})
	control := &workerControl{}
	control.client.Store(client)
	control.state.Store(&workerSnapshot{
		Enabled: true, Ready: true, Paused: true, StopID: 1,
		Targets: targets, sessions: []string{"page", "worker1", "worker2"},
	})
	session := &Session{}
	session.workers.Store(control)
	if state := session.checkedWorkerState(ctx); !state.Paused {
		t.Fatalf("concurrent drain rejected an unchanged stop: %+v", state)
	}
}

func TestWorkerAttachExitPreservesOtherPauseCommands(t *testing.T) {
	for _, phase := range []string{"initialize", "pause"} {
		t.Run(phase, func(t *testing.T) {
			observed := make(chan cdpMessage, 2)
			ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
				for {
					var command cdpMessage
					if wsjson.Read(ctx, conn, &command) != nil {
						return
					}
					if command.SessionID == "exiting" {
						_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Error: &cdpError{Message: "Target closed"}})
						continue
					}
					observed <- command
					_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)})
					if command.Method == "Runtime.evaluate" {
						return
					}
				}
			})
			control := &workerControl{}
			group := workerGroup{stopping: true, ready: true, targets: map[string]*controlledTarget{
				"page": {paused: true}, "running": {},
			}}
			var commands []cdpMessage
			if phase == "initialize" {
				var err error
				commands, err = control.attachTarget(ctx, client, &group, "page-id", cdpMessage{
					Method: "Target.attachedToTarget", SessionID: "page",
					Params: json.RawMessage(`{"sessionId":"exiting","targetInfo":{"targetId":"child","type":"worker"}}`),
				})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				group.targets["exiting"] = &controlledTarget{}
				commands = []cdpMessage{{Method: "Debugger.pause", SessionID: "exiting"}}
			}
			if err := control.executeCommands(ctx, client, &group, commands); err != nil {
				t.Fatal(err)
			}
			if group.targets["exiting"] != nil {
				t.Fatal("exited worker remains in group")
			}
			for _, method := range []string{"Debugger.pause", "Runtime.evaluate"} {
				select {
				case command := <-observed:
					if command.Method != method || command.SessionID != "running" {
						t.Fatalf("surviving worker command = %+v", command)
					}
				case <-ctx.Done():
					t.Fatal("dropped the surviving worker's pause command")
				}
			}
		})
	}
}

func TestCDPOverflowFailsInsteadOfBlockingReplies(t *testing.T) {
	ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
		for i := 0; i <= cdpEventCapacity; i++ {
			if wsjson.Write(ctx, conn, cdpMessage{Method: "Debugger.resumed", SessionID: "worker"}) != nil {
				return
			}
		}
	})
	err := client.call(ctx, "worker", "Runtime.getIsolateId", map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), "event queue overflow") {
		t.Fatalf("overflow did not invalidate the connection: %v", err)
	}
}

func TestCDPDialErrorOmitsDiscoveryToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newCDPClient(ctx, "ws://127.0.0.1:1/devtools/browser/test-discovery-token")
	if err == nil || !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "test-discovery-token") {
		t.Fatalf("connection error exposes the endpoint or loses its cause: %v", err)
	}
}

func TestWorkerGroupStopResumeAndLifecycle(t *testing.T) {
	g := workerGroup{targets: make(map[string]*controlledTarget), ready: true}
	g.attach("page", false)
	g.attach("worker", false)
	commands := g.event("Debugger.paused", "worker")
	if len(commands) != 1 || commands[0].Method != "Debugger.pause" || commands[0].SessionID != "page" {
		t.Fatalf("pause commands = %+v", commands)
	}
	if g.snapshot().Paused {
		t.Fatal("published shared-memory stop before the other target paused")
	}
	g.event("Debugger.paused", "page")
	stopped := g.snapshot()
	if !stopped.Paused || stopped.StopID == 0 {
		t.Fatalf("snapshot = %+v", stopped)
	}
	if got := g.event("Debugger.paused", "page"); len(got) != 0 || g.snapshot().StopID != stopped.StopID {
		t.Fatal("duplicate notification changed a stable stop")
	}
	commands = g.attach("new-worker", false)
	if g.snapshot().Paused || g.snapshot().StopID == stopped.StopID || len(commands) != 1 {
		t.Fatal("new worker did not invalidate the stop and request a pause")
	}
	g.event("Target.detachedFromTarget", "new-worker")
	if !g.snapshot().Paused {
		t.Fatal("departed worker prevents inspection of surviving workers")
	}
	commands = g.event("Debugger.resumed", "worker")
	if g.snapshot().Paused || len(commands) != 1 || commands[0].Method != "Debugger.resume" {
		t.Fatalf("resume = %+v, %+v", commands, g.snapshot())
	}
	g.event("Debugger.resumed", "page")
	if g.stopping {
		t.Fatal("coordinated resume started another stop")
	}
}

func TestWorkerGroupHoldsNewWorkerUntilResume(t *testing.T) {
	g := workerGroup{targets: map[string]*controlledTarget{
		"page": {paused: true},
	}, stopping: true, ready: true, epoch: 3}
	commands := g.attach("worker", true)
	if len(commands) != 0 || !g.snapshot().Paused || g.epoch != 4 {
		t.Fatalf("new worker left the startup barrier: %+v, %+v", commands, g.snapshot())
	}
	commands = g.event("Debugger.resumed", "page")
	if len(commands) != 1 || commands[0].Method != "Runtime.runIfWaitingForDebugger" ||
		commands[0].SessionID != "worker" || g.snapshot().Paused {
		t.Fatalf("resume did not release the new worker: %+v, %+v", commands, g.snapshot())
	}
	if g.targets["worker"].resumePending || g.targets["worker"].waiting {
		t.Fatal("startup release incorrectly waits for a Debugger.resumed event")
	}
}

func TestWorkerGroupStepWhileResumeIsInFlight(t *testing.T) {
	g := workerGroup{targets: map[string]*controlledTarget{
		"page": {paused: true}, "worker": {paused: true},
	}, stopping: true, ready: true}
	g.event("Debugger.resumed", "worker")
	g.event("Debugger.paused", "worker") // A short step completes first.
	commands := g.event("Debugger.resumed", "page")
	if len(commands) != 1 || commands[0].Method != "Debugger.pause" {
		t.Fatalf("late resume was not paused again: %+v", commands)
	}
	if g.snapshot().Paused {
		t.Fatal("reported quiescence before the late resume was stopped")
	}
	g.event("Debugger.paused", "page")
	if !g.snapshot().Paused {
		t.Fatal("step did not produce a new consistent stop")
	}
}

func TestWorkerGroupContinueWhilePauseIsInFlight(t *testing.T) {
	g := workerGroup{targets: make(map[string]*controlledTarget), ready: true}
	g.attach("page", false)
	g.attach("worker", false)
	g.event("Debugger.paused", "worker")
	g.event("Debugger.resumed", "worker")
	commands := g.event("Debugger.paused", "page")
	if len(commands) != 1 || commands[0].Method != "Debugger.resume" || g.stopping {
		t.Fatalf("late auxiliary pause restarted the group stop: %+v", commands)
	}
	g.event("Debugger.resumed", "page")
	if g.snapshot().Paused || g.stopping {
		t.Fatal("continue left the group stopped")
	}
}

func TestCDPReplyAndEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx := r.Context()
		var command cdpMessage
		if wsjson.Read(ctx, conn, &command) != nil {
			return
		}
		_ = wsjson.Write(ctx, conn, cdpMessage{Method: "Debugger.paused", SessionID: "worker"})
		_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{"value":42}`)})
		_ = wsjson.Read(ctx, conn, &command) // Wait for the client to close.
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := newCDPClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.conn.CloseNow()
	var result struct{ Value int }
	if err := client.call(ctx, "worker", "Debugger.enable", map[string]any{}, &result); err != nil || result.Value != 42 {
		t.Fatalf("reply = %+v, %v", result, err)
	}
	event := <-client.events
	if event.Method != "Debugger.paused" || event.SessionID != "worker" {
		t.Fatalf("event = %+v", event)
	}
}

func TestWorkerSnapshotRejectsUnprocessedResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var command cdpMessage
		if wsjson.Read(r.Context(), conn, &command) != nil {
			return
		}
		_ = wsjson.Write(r.Context(), conn, cdpMessage{Method: "Debugger.resumed", SessionID: "worker"})
		_ = wsjson.Write(r.Context(), conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)})
		_ = wsjson.Read(r.Context(), conn, &command)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := newCDPClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.conn.CloseNow()
	control := &workerControl{}
	control.client.Store(client)
	control.state.Store(&workerSnapshot{
		Enabled: true, Ready: true, Paused: true, StopID: 3,
		Targets: 1, sessions: []string{"worker"},
	})
	session := &Session{}
	session.workers.Store(control)
	if state := session.checkedWorkerState(ctx); state.Paused {
		t.Fatal("accepted an old snapshot while a resume notification awaited processing")
	}
}

func TestCDPWorkerExitCancelsPendingRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var command cdpMessage
		if wsjson.Read(r.Context(), conn, &command) != nil {
			return
		}
		// Worker destruction can drop the command reply altogether.
		_ = wsjson.Write(r.Context(), conn, cdpMessage{
			Method: "Target.detachedFromTarget", Params: json.RawMessage(`{"sessionId":"worker"}`),
		})
		_ = wsjson.Read(r.Context(), conn, &command)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := newCDPClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.conn.CloseNow()
	err = client.call(ctx, "worker", "Runtime.getIsolateId", map[string]any{}, nil)
	if err == nil || !closedCDPTarget(err) {
		t.Fatalf("pending request after worker exit: %v", err)
	}
}

func TestWorkerSnapshotDrainsRepliesAfterTargetExit(t *testing.T) {
	firstReply := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
		var commands [2]cdpMessage
		for i := range commands {
			if wsjson.Read(ctx, conn, &commands[i]) != nil {
				return
			}
		}
		_ = wsjson.Write(ctx, conn, cdpMessage{ID: commands[0].ID, Error: &cdpError{Message: "Target closed"}})
		close(firstReply)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		_ = wsjson.Write(ctx, conn, cdpMessage{ID: commands[1].ID, Result: json.RawMessage(`{}`)})
		var command cdpMessage
		if wsjson.Read(ctx, conn, &command) == nil {
			_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)})
		}
	})
	control := &workerControl{}
	control.client.Store(client)
	control.state.Store(&workerSnapshot{
		Enabled: true, Ready: true, Paused: true, StopID: 1,
		Targets: 2, sessions: []string{"page", "worker"},
	})
	session := &Session{}
	session.workers.Store(control)
	result := make(chan workerSnapshot, 1)
	go func() { result <- session.checkedWorkerState(ctx) }()
	select {
	case <-firstReply:
	case <-ctx.Done():
		t.Fatal("snapshot requests did not reach the debugger")
	}
	select {
	case <-result:
		t.Fatal("target exit canceled another in-flight snapshot request")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case state := <-result:
		if state.Paused {
			t.Fatal("accepted a snapshot after one target exited")
		}
	case <-ctx.Done():
		t.Fatal("snapshot requests did not finish")
	}
	if err := client.call(ctx, "page", "Runtime.getIsolateId", map[string]any{}, nil); err != nil {
		t.Fatalf("surviving target lost its debugger transport: %v", err)
	}
}

func TestCDPCanceledWritePreservesOtherTargets(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, client := workerTestClient(t, func(ctx context.Context, conn *websocket.Conn) {
		conn.SetReadLimit(64 << 20)
		_, reader, err := conn.Reader(ctx)
		if err != nil {
			return
		}
		var first [1]byte
		if _, err := io.ReadFull(reader, first[:]); err != nil {
			return
		}
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return
		}
		var command cdpMessage
		if wsjson.Read(ctx, conn, &command) == nil {
			_ = wsjson.Write(ctx, conn, cdpMessage{ID: command.ID, Result: json.RawMessage(`{}`)})
		}
	})
	request, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		// Keep the server's first request outstanding until cancellation.
		result <- client.call(request, "exiting", "Runtime.evaluate", map[string]any{
			"expression": strings.Repeat(" ", 32<<20),
		}, nil)
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("request did not reach the server")
	}
	cancel()
	close(release)
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled request: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("canceled request did not finish")
	}
	if err := client.call(ctx, "surviving", "Runtime.getIsolateId", map[string]any{}, nil); err != nil {
		t.Fatalf("canceling one request closed the shared connection: %v", err)
	}
}
