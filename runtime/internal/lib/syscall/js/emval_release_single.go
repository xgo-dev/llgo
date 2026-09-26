//go:build js && wasm && !llgo.wasm.workers

package js

func emvalOwner() int { return 0 }

func releaseEmval(handle uintptr, _ int) {
	cEmvalDecref(handle)
}

func pollEmvalReleases() {}
