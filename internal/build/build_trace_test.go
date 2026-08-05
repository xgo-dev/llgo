//go:build !llgo

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildTraceWritesChromeEvents(t *testing.T) {
	dir := t.TempDir()
	tracer, err := startBuildTrace("trace.json", dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	buildSpan := tracer.startCoordinator("build", map[string]any{"packages": []string{"./..."}})
	preSpan := tracer.startPackageCoordinator("pre", "example.com/p")
	preSpan.done()
	ssaSpan := tracer.startWorker("ssa", "example.com/p")
	ssaSpan.done()
	tracer.rememberSSA("example.com/p", ssaSpan)
	backendSpan := tracer.startWorker("backend+publish", "example.com/p")
	backendSpan.setArg("class", "isolated")
	tracer.flowFromSSA("example.com/p", backendSpan)
	backendSpan.done()
	buildSpan.done()
	if err := tracer.close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	var events []buildTraceEvent
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("trace is not valid Chrome Trace JSON: %v\n%s", err, data)
	}
	var metadata, complete, packageCoordinator, flowStart, flowEnd bool
	for _, event := range events {
		switch event.Phase {
		case "M":
			metadata = true
		case "X":
			complete = true
			if event.Name == "pre example.com/p" {
				packageCoordinator = event.Category == "llgo.pre" && event.TID == 0
			}
			if event.Name == "backend+publish example.com/p" {
				if got := event.Args["class"]; got != "isolated" {
					t.Fatalf("backend class = %v, want isolated", got)
				}
			}
		case "s":
			flowStart = true
		case "f":
			flowEnd = event.Bind == "e"
		}
	}
	if !metadata || !complete || !packageCoordinator || !flowStart || !flowEnd {
		t.Fatalf("trace phases: metadata=%v complete=%v packageCoordinator=%v flowStart=%v flowEnd=%v", metadata, complete, packageCoordinator, flowStart, flowEnd)
	}
}

func TestBuildTraceWorkerLanesRespectParallelism(t *testing.T) {
	tracer, err := startBuildTrace(filepath.Join(t.TempDir(), "trace.json"), "", 2)
	if err != nil {
		t.Fatal(err)
	}
	var active atomic.Int32
	var maximum atomic.Int32
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			span := tracer.startWorker("backend", string(rune('a'+i)))
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			active.Add(-1)
			span.done()
		}()
	}
	wg.Wait()
	if got := maximum.Load(); got > 2 {
		t.Fatalf("maximum active traced workers = %d, want <= 2", got)
	}
	if err := tracer.close(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildTraceRefusesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.json")
	if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := startBuildTrace(path, dir, 1); err == nil {
		t.Fatal("startBuildTrace(existing file) succeeded")
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "keep" {
		t.Fatalf("existing output = %q, %v; want unchanged", got, err)
	}
}

func TestBuildTraceDisabledAndErrors(t *testing.T) {
	tracer, err := startBuildTrace("", t.TempDir(), 1)
	if err != nil || tracer != nil {
		t.Fatalf("disabled trace = (%v, %v), want (nil, nil)", tracer, err)
	}
	var span *buildTraceSpan
	span.setArg("ignored", true)
	span.done()
	tracer.startCoordinator("ignored", nil).done()
	tracer.startPackageCoordinator("ignored", "example.com/p").done()
	tracer.startWorker("ignored", "example.com/p").done()
	tracer.rememberSSA("example.com/p", span)
	tracer.flowFromSSA("example.com/p", span)
	tracer.flow(span, span)
	if err := tracer.close(); err != nil {
		t.Fatalf("closing disabled trace: %v", err)
	}

	if _, err := startBuildTrace(filepath.Join(t.TempDir(), "missing", "trace.json"), "", 1); err == nil {
		t.Fatal("startBuildTrace in missing directory succeeded")
	}

	tracer, err = startBuildTrace(filepath.Join(t.TempDir(), "trace.json"), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	tracer.writeEvent(buildTraceEvent{
		Name:  "invalid",
		Phase: "X",
		Args:  map[string]any{"function": func() {}},
	})
	if err := tracer.close(); err == nil {
		t.Fatal("closing trace after JSON encoding error succeeded")
	}
}
