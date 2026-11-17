//go:build wasm && llgo.wasm.gc.linear && !nogc && !baremetal

package runtime

import (
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"unsafe"
)

// A sampled synthetic PC must remain resolvable after its originating G exits.
// Intern source locations globally without retaining the G's stack or GLS.
// Numeric PC/entry words are encoded to avoid conservative heap roots.
type heapProfilePCFrame struct {
	byPC, bySource  *heapProfilePCFrame
	pcKey, entryKey uintptr
	function, file  string
	line, startLine int
}

var heapProfilePCs struct {
	once         psync.Once
	mu           psync.Mutex
	pcs, sources [256]*heapProfilePCFrame
}

func initHeapProfilePCs() { heapProfilePCs.mu.Init(nil) }

func profileSourceHash(f CallerFrame) uintptr {
	h := uintptr(f.Line) ^ uintptr(f.StartLine)<<8
	for i := 0; i < len(f.Function); i++ {
		h = (h ^ uintptr(f.Function[i])) * 16777619
	}
	for i := 0; i < len(f.File); i++ {
		h = (h ^ uintptr(f.File[i])) * 16777619
	}
	return h % 256
}

func pinHeapProfilePCs(pcs []uintptr) {
	for i, pc := range pcs {
		frame, ok := FrameForPC(pc)
		if !ok {
			continue
		}
		heapProfilePCs.once.Do(initHeapProfilePCs)
		heapProfilePCs.mu.Lock()
		index := profileSourceHash(frame)
		var found *heapProfilePCFrame
		for node := heapProfilePCs.sources[index]; node != nil; node = node.bySource {
			if node.function == frame.Function && node.file == frame.File && node.line == frame.Line && node.startLine == frame.StartLine {
				found = node
				break
			}
		}
		if found == nil {
			found = (*heapProfilePCFrame)(AllocRoot(unsafe.Sizeof(heapProfilePCFrame{})))
			canonical := nextCallerPCBase(nil) | callersPCValue
			found.pcKey = ^canonical
			found.entryKey = ^frame.Entry
			found.function, found.file = frame.Function, frame.File
			found.line, found.startLine = frame.Line, frame.StartLine
			found.bySource = heapProfilePCs.sources[index]
			heapProfilePCs.sources[index] = found
			pcIndex := (canonical &^ callerPCMask) % 256
			found.byPC = heapProfilePCs.pcs[pcIndex]
			heapProfilePCs.pcs[pcIndex] = found
		}
		pcs[i] = ^found.pcKey
		heapProfilePCs.mu.Unlock()
	}
}

func heapProfileFrameForPC(pc uintptr) (CallerFrame, bool) {
	heapProfilePCs.once.Do(initHeapProfilePCs)
	heapProfilePCs.mu.Lock()
	defer heapProfilePCs.mu.Unlock()
	key := ^(pc&^callerPCMask | callersPCValue)
	for node := heapProfilePCs.pcs[(pc&^callerPCMask)%256]; node != nil; node = node.byPC {
		if node.pcKey == key {
			return CallerFrame{PC: pc, Entry: ^node.entryKey, Function: node.function, File: node.file, Line: node.line, StartLine: node.startLine}, true
		}
	}
	return CallerFrame{}, false
}
