package build

import (
	"bytes"
	stdcontext "context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Exercise the production registry operations with isolated collector state.
// Counting metadata reads checks scaling without timing limits or depending on
// conservative collection of objects belonging to the running test itself.
func TestWasmFinalizerCandidates(t *testing.T) {
	fset := token.NewFileSet()
	path := filepath.Join("..", "..", "runtime", "internal", "runtime", "tinygogc", "finalizer.go")
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{
		"encodeFinalizerAddress": true, "preserveFinalizableObjects": true,
		"finalizerObjectBlock": true, "finalizerObjectState": true,
		"candidateForObject": true, "earlierFinalizerForObject": true,
		"hasCandidateFinalizer": true, "finalizerObjectBlocked": true,
		"queueCallbacksForObject": true,
	}
	var source bytes.Buffer
	source.WriteString(finalizerCandidateTestSource)
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if !wanted[decl.Name.Name] {
				continue
			}
			delete(wanted, decl.Name.Name)
		case *ast.GenDecl:
			if decl.Tok == token.IMPORT {
				continue
			}
		}
		if err := format.Node(&source, fset, decl); err != nil {
			t.Fatal(err)
		}
		source.WriteByte('\n')
	}
	if len(wanted) != 0 {
		t.Fatalf("missing collector operations: %v", wanted)
	}
	path = filepath.Join(t.TempDir(), "candidates_test.go")
	if err := os.WriteFile(path, source.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-timeout=25s", path)
	cmd.WaitDelay = time.Second
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("finalizer registry checks: %v\n%s", err, output)
	}
}

const finalizerCandidateTestSource = `package candidates
import (
  "testing"
  "unsafe"
)
const (
  blockStateFree uint8 = iota
  blockStateHead
  blockStateTail
  blockStateMark
  endBlock = 4096
  blockBytes = 16
)
var heap []byte
var states [endBlock]uint8
var metadataReads int
func gcStateOf(block uintptr) uint8 { metadataReads++; return states[block] }
func gcAddressOf(block uintptr) uintptr { return uintptr(unsafe.Pointer(&heap[block*blockBytes])) }
func blockFromAddr(addr uintptr) uintptr { return (addr - gcAddressOf(0)) / blockBytes }
// These objects have no edges. Real dependency/cycle marking is covered by the
// Wasm lifecycle fixture; this model isolates candidate and queue traversal.
func startMark(block uintptr) { states[block] = blockStateMark }
func finishMark() {}
func resetCollector() {
  heap = make([]byte, endBlock*blockBytes)
  for i := range states { states[i] = blockStateHead }
  finalizers, readyFinalizers = nil, nil
  metadataReads = 0
}
func recordFor(block uintptr, kind finalizerKind) *finalizerRecord {
  return &finalizerRecord{
    objectKey: encodeFinalizerAddress(gcAddressOf(block)),
    object: encodeFinalizerAddress(gcAddressOf(block)+3),
    kind: kind, callback: func(unsafe.Pointer) {},
  }
}
func readyCounts(t *testing.T) (finals, cleanups int) {
  t.Helper()
  seen := make(map[*finalizerRecord]bool)
  for r := readyFinalizers; r != nil; r = r.readyNext {
    if seen[r] || r.state != finalizerQueued || r.next != nil || uintptr(r.ready) != ^r.object {
      t.Fatal("invalid or duplicate queued record")
    }
    seen[r] = true
    if r.kind == objectFinalizer { finals++ } else { cleanups++ }
  }
  return
}
func TestEmptyRegistry(t *testing.T) {
  resetCollector()
  preserveFinalizableObjects()
  if metadataReads != 0 { t.Fatalf("empty registry inspected %d heap blocks", metadataReads) }
}
func TestInterleavedRecords(t *testing.T) {
  resetCollector()
  // A has two cleanups separated by B's finalizer. C has only a cleanup.
  records := []*finalizerRecord{
    recordFor(1, objectFinalizer), recordFor(7, objectCleanup),
    recordFor(1, objectCleanup), recordFor(7, objectFinalizer),
    recordFor(1, objectCleanup), recordFor(13, objectCleanup),
  }
  for i := 0; i+1 < len(records); i++ { records[i].next = records[i+1] }
  finalizers = records[0]
  preserveFinalizableObjects()
  if metadataReads > 100 { t.Fatalf("six records inspected %d heap blocks", metadataReads) }
  if f, c := readyCounts(t); f != 2 || c != 1 { t.Fatalf("ready finalizers/cleanups = %d/%d, want 2/1", f, c) }
  pending := 0
  for r := finalizers; r != nil; r = r.next {
    pending++
    if pending > 3 || r.kind != objectCleanup || r.candidate {
      t.Fatal("cleanup was not deferred to a later collection")
    }
  }
  if pending != 3 { t.Fatalf("pending cleanups = %d, want 3", pending) }
  // Model completed finalizers and objects becoming unreachable again.
  readyFinalizers = nil
  states[1], states[7] = blockStateHead, blockStateHead
  metadataReads = 0
  preserveFinalizableObjects()
  if metadataReads > 100 { t.Fatalf("three records inspected %d heap blocks", metadataReads) }
  if f, c := readyCounts(t); f != 0 || c != 3 { t.Fatalf("later ready finalizers/cleanups = %d/%d, want 0/3", f, c) }
  if finalizers != nil { t.Fatal("queue traversal lost pending records") }
}
`
