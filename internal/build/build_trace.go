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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const buildTracePID = 1

// buildTraceEvent follows the Chrome Trace Event format accepted by both
// chrome://tracing and Perfetto.
type buildTraceEvent struct {
	Name     string         `json:"name"`
	Category string         `json:"cat,omitempty"`
	Phase    string         `json:"ph"`
	Time     int64          `json:"ts,omitempty"`
	Duration int64          `json:"dur,omitempty"`
	PID      int            `json:"pid"`
	TID      int            `json:"tid"`
	ID       uint64         `json:"id,omitempty"`
	Bind     string         `json:"bp,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
}

// buildTracer is owned by one Build invocation. The lane semaphore mirrors
// the build's effective package parallelism (-p, or GOMAXPROCS by default),
// so overlapping worker-lane events visualize the same concurrency budget
// used by SSA and isolated LLVM backend work.
type buildTracer struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	first    bool
	writeErr error

	lanes chan int
	// ssaSpans connects each completed Go SSA build to the corresponding
	// package backend without recording the transitive import graph.
	ssaSpans map[string]*buildTraceSpan

	nextFlowID uint64
	closeOnce  sync.Once
	closeErr   error
}

type buildTraceSpan struct {
	tracer *buildTracer
	name   string
	cat    string
	args   map[string]any
	start  time.Time
	lane   int
	worker bool

	end  time.Time
	once sync.Once
}

func startBuildTrace(path, dir string, parallelism int) (*buildTracer, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return nil, err
	}
	tracer := &buildTracer{
		file:     file,
		writer:   bufio.NewWriter(file),
		first:    true,
		lanes:    make(chan int, max(1, parallelism)),
		ssaSpans: make(map[string]*buildTraceSpan),
	}
	for lane := 1; lane <= cap(tracer.lanes); lane++ {
		tracer.lanes <- lane
	}
	if _, err := tracer.writer.WriteString("[\n"); err != nil {
		_ = file.Close()
		return nil, err
	}
	tracer.writeEvent(buildTraceEvent{
		Name:  "process_name",
		Phase: "M",
		PID:   buildTracePID,
		Args:  map[string]any{"name": "llgo build"},
	})
	tracer.writeEvent(buildTraceEvent{
		Name:  "thread_name",
		Phase: "M",
		PID:   buildTracePID,
		TID:   0,
		Args:  map[string]any{"name": "coordinator"},
	})
	for lane := 1; lane <= cap(tracer.lanes); lane++ {
		tracer.writeEvent(buildTraceEvent{
			Name:  "thread_name",
			Phase: "M",
			PID:   buildTracePID,
			TID:   lane,
			Args:  map[string]any{"name": fmt.Sprintf("worker %d", lane)},
		})
	}
	return tracer, nil
}

func (t *buildTracer) startCoordinator(name string, args map[string]any) *buildTraceSpan {
	if t == nil {
		return nil
	}
	return &buildTraceSpan{
		tracer: t,
		name:   name,
		cat:    "llgo.build",
		args:   args,
		start:  time.Now(),
	}
}

func (t *buildTracer) startPackageCoordinator(stage, pkgPath string) *buildTraceSpan {
	if t == nil {
		return nil
	}
	span := t.startCoordinator(stage+" "+pkgPath, map[string]any{
		"package": pkgPath,
		"stage":   stage,
	})
	span.cat = "llgo." + stage
	return span
}

// startWorker assigns one trace lane for the complete worker span. Callers
// must not nest worker spans and the build must not run more callers than its
// effective package parallelism; otherwise tracing could block real work.
func (t *buildTracer) startWorker(stage, pkgPath string) *buildTraceSpan {
	if t == nil {
		return nil
	}
	lane := <-t.lanes
	return &buildTraceSpan{
		tracer: t,
		name:   stage + " " + pkgPath,
		cat:    "llgo." + stage,
		args: map[string]any{
			"package": pkgPath,
			"stage":   stage,
		},
		start:  time.Now(),
		lane:   lane,
		worker: true,
	}
}

func (t *buildTracer) rememberSSA(pkgPath string, span *buildTraceSpan) {
	if t == nil || span == nil {
		return
	}
	t.mu.Lock()
	t.ssaSpans[pkgPath] = span
	t.mu.Unlock()
}

func (t *buildTracer) flowFromSSA(pkgPath string, to *buildTraceSpan) {
	if t == nil || to == nil {
		return
	}
	t.mu.Lock()
	from := t.ssaSpans[pkgPath]
	t.mu.Unlock()
	t.flow(from, to)
}

func (s *buildTraceSpan) setArg(name string, value any) {
	if s == nil {
		return
	}
	if s.args == nil {
		s.args = make(map[string]any)
	}
	s.args[name] = value
}

func (s *buildTraceSpan) done() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		s.end = time.Now()
		s.tracer.writeEvent(buildTraceEvent{
			Name:     s.name,
			Category: s.cat,
			Phase:    "X",
			Time:     traceMicros(s.start),
			Duration: max(0, traceMicros(s.end)-traceMicros(s.start)),
			PID:      buildTracePID,
			TID:      s.lane,
			Args:     s.args,
		})
		if s.worker {
			s.tracer.lanes <- s.lane
		}
	})
}

// flow records a dependency arrow from a completed producer to a started
// consumer, matching the flow events emitted by Go's build trace.
func (t *buildTracer) flow(from, to *buildTraceSpan) {
	if t == nil || from == nil || to == nil || from.end.IsZero() {
		return
	}
	t.mu.Lock()
	t.nextFlowID++
	id := t.nextFlowID
	t.mu.Unlock()
	name := from.name + " -> " + to.name
	t.writeEvent(buildTraceEvent{
		Name:     name,
		Category: "flow",
		Phase:    "s",
		Time:     traceMicros(from.end),
		PID:      buildTracePID,
		TID:      from.lane,
		ID:       id,
	})
	t.writeEvent(buildTraceEvent{
		Name:     name,
		Category: "flow",
		Phase:    "f",
		Time:     traceMicros(to.start),
		PID:      buildTracePID,
		TID:      to.lane,
		ID:       id,
		Bind:     "e",
	})
}

func (t *buildTracer) writeEvent(event buildTraceEvent) {
	if t == nil {
		return
	}
	data, err := json.Marshal(event)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.writeErr != nil {
		return
	}
	if err != nil {
		t.writeErr = err
		return
	}
	if !t.first {
		if _, err := t.writer.WriteString(",\n"); err != nil {
			t.writeErr = err
			return
		}
	}
	t.first = false
	if _, err := t.writer.Write(data); err != nil {
		t.writeErr = err
	}
}

func (t *buildTracer) close() error {
	if t == nil {
		return nil
	}
	t.closeOnce.Do(func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.writeErr == nil {
			_, t.writeErr = t.writer.WriteString("\n]\n")
		}
		if err := t.writer.Flush(); t.writeErr == nil {
			t.writeErr = err
		}
		if err := t.file.Close(); t.writeErr == nil {
			t.writeErr = err
		}
		t.closeErr = t.writeErr
	})
	return t.closeErr
}

func traceMicros(t time.Time) int64 {
	return t.UnixNano() / int64(time.Microsecond)
}
