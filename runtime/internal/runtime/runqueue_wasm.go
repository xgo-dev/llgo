//go:build llgo && wasm && (js || (wasip1 && !llgo.wasi_threads))

package runtime

func (gp *g) RunqueueNext() *g {
	return gp.context.platform.runqNext
}

func (gp *g) SetRunqueueNext(next *g) {
	gp.context.platform.runqNext = next
}

func (gp *g) RunqueueQueued() bool {
	return gp.context.platform.runqQueued
}

func (gp *g) SetRunqueueQueued(queued bool) {
	gp.context.platform.runqQueued = queued
}
