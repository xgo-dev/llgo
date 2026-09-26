//go:build llgo && js && wasm && llgo.wasm.workers

package js

import (
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
)

type pendingEmvalRelease struct {
	next   *pendingEmvalRelease
	handle uintptr
	owner  int
}

var pendingEmvalReleases struct {
	mutex wasmsync.Mutex
	head  *pendingEmvalRelease
}

func emvalOwner() int {
	return llruntime.SchedulerProcID()
}

func releaseEmval(handle uintptr, owner int) {
	// This record crosses both a physical-worker boundary and the scheduler's
	// host-event boundary. Keep it in the runtime's explicit root set until the
	// owning realm has consumed it; a Go heap allocation is not reliably rooted
	// while the scheduler hands the detached list to a new G. Always enqueue,
	// even on the owner worker: a finalizer must not enter the JavaScript host
	// ABI directly from the GC finalizer goroutine.
	release := (*pendingEmvalRelease)(llruntime.AllocRoot(unsafe.Sizeof(pendingEmvalRelease{})))
	*release = pendingEmvalRelease{handle: handle, owner: owner}
	// Install the poll before publishing, so a worker waking for this release
	// can observe it immediately. The worker profile keeps this hook installed.
	ensureCallbackPoll()
	pendingEmvalReleases.mutex.Lock(nil)
	release.next = pendingEmvalReleases.head
	pendingEmvalReleases.head = release
	pendingEmvalReleases.mutex.Unlock()

	// The finalizer goroutine may belong to any worker, whereas an emval
	// handle belongs to the realm that created it.
	llruntime.WakeWasmCallbackPoll()
}

func pollEmvalReleases() {
	owner := llruntime.SchedulerProcID()
	var ready *pendingEmvalRelease
	pendingEmvalReleases.mutex.Lock(nil)
	link := &pendingEmvalReleases.head
	for *link != nil {
		release := *link
		if release.owner != owner {
			link = &release.next
			continue
		}
		*link = release.next
		release.next = ready
		ready = release
	}
	pendingEmvalReleases.mutex.Unlock()
	if ready != nil {
		// pollEmvalReleases runs on the scheduler's system fiber. Start a G in
		// the same worker realm before touching sync-backed callback state.
		go drainEmvalReleases(ready)
	}
}

func drainEmvalReleases(ready *pendingEmvalRelease) {
	for ready != nil {
		release := ready
		ready = release.next
		release.next = nil
		cEmvalDecref(release.handle)
		llruntime.FreeRoot(unsafe.Pointer(release))
	}
}
