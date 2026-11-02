package main

import (
	"os"
	"runtime"
	"time"
)

type object struct {
	id  int
	pad [64]byte
}

type namedPointer *object
type markerError string

func (e markerError) Error() string { return string(e) }

// These globals are never read. Each still keeps a finalized allocation alive,
// even when the pointer is nested in an aggregate or closure environment.
var globalPointer = makeGlobalRoot()
var globalArray = [1]*object{makeGlobalRoot()}
var globalStruct = struct{ P *object }{makeGlobalRoot()}
var globalSlice = []*object{makeGlobalRoot()}
var globalMap = map[int]*object{0: makeGlobalRoot()}
var globalMapKey = map[*object]int{makeGlobalRoot(): 0}
var globalInterface any = makeGlobalRoot()
var globalClosure = func() func() *object {
	p := makeGlobalRoot()
	return func() *object { return p }
}()

var aliasSource = makeGlobalRoot()
var globalAliasMap = map[int]*object{0: aliasSource}

func init() {
	// The copied pointer remains a root after its source variable is cleared.
	aliasSource = nil
}

//go:noinline
func makeGlobalRoot() *object {
	p := new(object)
	runtime.SetFinalizer(p, func(*object) { panic("unread global GC root was dropped") })
	return p
}

var events = make(chan int, 64)

func (p *object) finish() error {
	events <- p.id
	return markerError("discarded finalizer result")
}

//go:noinline
func dynamic(obj, fn any) { runtime.SetFinalizer(obj, fn) }

//go:noinline
func register() {
	for i := 0; i < 8; i++ {
		// The closure is the only remaining owner of this captured allocation.
		seed := new(int)
		*seed = 100
		p := &object{id: i}
		runtime.SetFinalizer(p, func(p *object) { events <- *seed + p.id })

		q := &object{id: 200 + i}
		runtime.SetFinalizer(q, (*object).finish)

		n := namedPointer(&object{id: 300 + i})
		runtime.SetFinalizer(n, func(p namedPointer) { events <- p.id })

		// Both paths share the registry, including replacement and cancellation.
		r := &object{id: 400 + i}
		dynamic(r, func(*object) { panic("replaced dynamic finalizer ran") })
		runtime.SetFinalizer(r, nil)
		runtime.SetFinalizer(r, func(p *object) { events <- p.id })

		s := &object{id: 500 + i}
		runtime.SetFinalizer(s, func(*object) { panic("replaced direct finalizer ran") })
		dynamic(s, nil)
		dynamic(s, func(p *object) { events <- p.id })

		c := &object{id: 600 + i}
		runtime.SetFinalizer(c, func(*object) { panic("canceled direct finalizer ran") })
		dynamic(c, nil)
		c = &object{id: 700 + i}
		dynamic(c, func(*object) { panic("canceled dynamic finalizer ran") })
		runtime.SetFinalizer(c, nil)
	}
}

func main() {
	if len(os.Args) > 1 {
		runtime.SetFinalizer((*object)(nil), nil)
		panic("nil finalizer object was accepted")
	}
	registered := make(chan struct{})
	go func() {
		register()
		close(registered)
	}()
	<-registered
	deadline := time.Now().Add(45 * time.Second)
	var seen [6]bool
	count := 0
	for count < 5 {
		if time.Now().After(deadline) {
			println("finalizer groups:", seen[1], seen[2], seen[3], seen[4], seen[5])
			panic("finalizer callback did not run")
		}
		runtime.GC()
		for len(events) != 0 {
			id := <-events
			group := id / 100
			if group < 1 || group > 5 || id%100 >= 8 {
				panic("finalizer closure or argument was corrupted")
			}
			if !seen[group] {
				seen[group] = true
				count++
			}
		}
		// Emscripten's asynchronous host must regain control to schedule work.
		time.Sleep(10 * time.Millisecond)
	}
	for i := 0; i < 3; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	println("wasm direct finalizers passed")
}
