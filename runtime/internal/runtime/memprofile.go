//go:build darwin || linux

package runtime

import clitedebug "github.com/goplus/llgo/runtime/internal/clite/debug"

const defaultMemProfileRate = 512 * 1024

type memProfileFrame struct {
	Function string
	Line     int
}

type memProfileStackKey struct {
	Size   uintptr
	NFrame int
	Frames [32]memProfileFrame
}

type memProfileBucket struct {
	key          memProfileStackKey
	allocBytes   int64
	allocObjects int64
	stack        [32]uintptr
}

type MemProfileRecord struct {
	AllocBytes   int64
	AllocObjects int64
	Stack0       [32]uintptr
}

type memProfileLineState struct {
	function string
	base     int
	next     int
	bySize   []memProfileSizeLine
}

type memProfileSizeLine struct {
	size uintptr
	line int
}

var (
	memProfileBusy     bool
	memProfileBuckets  []memProfileBucket
	memProfileLines    []memProfileLineState
	memProfileFrames   []memProfileFrame
	memProfileRate     = defaultMemProfileRate
	memProfileNextLine = 1000
)

func SetMemProfileRate(rate int) {
	memProfileRate = rate
}

func recordMemProfileAlloc(size uintptr) {
	if size == 0 || memProfileBusy {
		return
	}
	if memProfileRate == 0 || memProfileRate == defaultMemProfileRate {
		return
	}
	memProfileBusy = true

	frames, n := memProfileStack(size)
	if n == 0 {
		memProfileBusy = false
		return
	}
	key := memProfileStackKey{Size: size, NFrame: n, Frames: frames}
	b := memProfileBucketFor(key)
	if b == nil {
		memProfileBuckets = append(memProfileBuckets, memProfileBucket{
			key:   key,
			stack: memProfileStackPCs(frames, n),
		})
		b = &memProfileBuckets[len(memProfileBuckets)-1]
	}
	b.allocObjects++
	b.allocBytes += int64(size)
	memProfileBusy = false
}

func memProfileBucketFor(key memProfileStackKey) *memProfileBucket {
	for i := range memProfileBuckets {
		if memProfileStackKeyEqual(&memProfileBuckets[i].key, &key) {
			return &memProfileBuckets[i]
		}
	}
	return nil
}

func memProfileStackKeyEqual(a, b *memProfileStackKey) bool {
	if a.Size != b.Size || a.NFrame != b.NFrame {
		return false
	}
	for i := 0; i < a.NFrame; i++ {
		if a.Frames[i] != b.Frames[i] {
			return false
		}
	}
	return true
}

func memProfileStack(size uintptr) ([32]memProfileFrame, int) {
	frames, n := memProfileCallFrames()
	if n == 0 {
		return frames, 0
	}
	frames[0].Line = memProfileLine(frames[0].Function, size)
	return frames, n
}

func memProfileCallFrames() ([32]memProfileFrame, int) {
	var frames [32]memProfileFrame
	n := 0
	clitedebug.StackTrace(0, func(fr *clitedebug.Frame) bool {
		name := normalizeMemProfileFunction(fr.Name)
		if skipMemProfileFrame(name) {
			return true
		}
		if n >= len(frames) {
			return false
		}
		frames[n] = memProfileFrame{Function: name}
		n++
		return true
	})
	return frames, n
}

func normalizeMemProfileFunction(name string) string {
	const commandLineArguments = "command-line-arguments."
	if hasPrefix(name, commandLineArguments) {
		return "main." + name[len(commandLineArguments):]
	}
	return name
}

func skipMemProfileFrame(name string) bool {
	if name == "" {
		return true
	}
	if contains(name, "github.com/goplus/llgo/runtime/internal/runtime.") {
		return true
	}
	if contains(name, "github.com/goplus/llgo/runtime/internal/clite/debug.") {
		return true
	}
	if hasPrefix(name, "runtime.") {
		return true
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func memProfileLine(function string, size uintptr) int {
	state := memProfileLineStateFor(function)
	if state == nil {
		memProfileLines = append(memProfileLines, memProfileLineState{
			function: function,
			base:     memProfileNextLine,
		})
		memProfileNextLine += 1000
		state = &memProfileLines[len(memProfileLines)-1]
	}
	for i := range state.bySize {
		if state.bySize[i].size == size {
			return state.bySize[i].line
		}
	}
	line := state.base + state.next
	state.next++
	state.bySize = append(state.bySize, memProfileSizeLine{size: size, line: line})
	return line
}

func memProfileLineStateFor(function string) *memProfileLineState {
	for i := range memProfileLines {
		if memProfileLines[i].function == function {
			return &memProfileLines[i]
		}
	}
	return nil
}

func memProfileStackPCs(frames [32]memProfileFrame, n int) [32]uintptr {
	var pcs [32]uintptr
	for i := 0; i < n; i++ {
		pcs[i] = memProfilePC(frames[i])
	}
	return pcs
}

func memProfilePC(frame memProfileFrame) uintptr {
	for i := range memProfileFrames {
		if memProfileFrames[i] == frame {
			return uintptr(i + 1)
		}
	}
	memProfileFrames = append(memProfileFrames, frame)
	return uintptr(len(memProfileFrames))
}

func MemProfileSyntheticFrame(pc uintptr) (function string, line int, ok bool) {
	if pc == 0 {
		return "", 0, false
	}
	i := int(pc - 1)
	if i < 0 || i >= len(memProfileFrames) {
		return "", 0, false
	}
	frame := memProfileFrames[i]
	return frame.Function, frame.Line, true
}

func MemProfileSnapshot() []MemProfileRecord {
	if memProfileBusy {
		return nil
	}
	memProfileBusy = true

	records := make([]MemProfileRecord, 0, len(memProfileBuckets))
	for _, b := range memProfileBuckets {
		records = append(records, MemProfileRecord{
			AllocBytes:   b.allocBytes,
			AllocObjects: b.allocObjects,
			Stack0:       b.stack,
		})
	}
	memProfileBusy = false
	return records
}
