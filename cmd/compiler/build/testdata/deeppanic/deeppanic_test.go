package deeppanic

import (
	"runtime/debug"
	"testing"
)

func TestDeepPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		deepCall(250)
	})
}

func TestVeryDeepPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		deepCall(5000)
	})
}

func TestMediumPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		deepCall(70)
	})
}

func TestCCallbackPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		callCGoCallback()
	})
}

func TestDeepCCallbackPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		deepCCallback(250)
	})
}

func TestVeryDeepCCallbackPanic(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		deepCCallback(5000)
	})
}

func TestDeepInsideCCallbackPanic(t *testing.T) {
	callbackDepth = 250
	t.Run("child", func(t *testing.T) {
		callCGoCallback()
	})
}

//go:noinline
func deepCCallback(depth int) {
	if depth == 0 {
		callCGoCallback()
		return
	}
	deepCCallback(depth - 1)
}

func TestDeepFault(t *testing.T) {
	debug.SetPanicOnFault(true)
	deepFault(250)
}
func TestVeryDeepFault(t *testing.T) {
	debug.SetPanicOnFault(true)
	deepFault(5000)
}
func TestDeepCallbackFault(t *testing.T) {
	debug.SetPanicOnFault(true)
	callbackFault, callbackDepth = true, 250
	callCGoCallback()
}

func TestForeignCallbackPanic(t *testing.T) {
	callbackDepth = 250
	callForeignThreadCallback()
}

func TestForeignCallbackFault(t *testing.T) {
	callbackFault, callbackDepth = true, 250
	callForeignThreadCallback()
}

func TestDynamicCallbackPanic(t *testing.T) {
	callbackDepth = 250
	callDynamicLibraryCallback()
}
func TestDynamicCallbackFault(t *testing.T) {
	callbackFault, callbackDepth = true, 250
	callDynamicLibraryCallback()
}
