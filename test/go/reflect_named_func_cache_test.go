package gotest

import (
	"reflect"
	"testing"
)

type reflectColdNamedFunc func() int

func (*reflectColdNamedFunc) Marker() {}

func TestReflectNamedFuncConcurrentPublication(t *testing.T) {
	const workers = 32
	start := make(chan struct{})
	results := make(chan reflect.Type, workers)
	for i := 0; i < workers; i++ {
		go func() {
			<-start
			ft := reflect.TypeOf(reflectColdNamedFunc(nil))
			pt := reflect.PointerTo(ft)
			if reflect.New(ft).Type() != pt || pt.Elem() != ft {
				results <- nil
				return
			}
			results <- pt
		}()
	}
	close(start)
	want := reflect.TypeOf((*reflectColdNamedFunc)(nil))
	for i := 0; i < workers; i++ {
		if got := <-results; got != want {
			t.Fatalf("concurrent named function pointer = %v, want canonical %v", got, want)
		}
	}
	if !reflect.New(want.Elem()).Type().Implements(reflect.TypeOf((*interface{ Marker() })(nil)).Elem()) {
		t.Fatal("published pointer lost its method set")
	}
}

type reflectBenchNamedFunc[T any] func()

func reflectBenchFuncTypes[T any]() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(reflectBenchNamedFunc[T](nil)),
		reflect.TypeOf(reflectBenchNamedFunc[*T](nil)),
		reflect.TypeOf(reflectBenchNamedFunc[[]T](nil)),
		reflect.TypeOf(reflectBenchNamedFunc[map[int]T](nil)),
	}
}

var reflectBenchPointerSink reflect.Type

func BenchmarkReflectNamedFuncPointer(b *testing.B) {
	factories := []func() []reflect.Type{
		reflectBenchFuncTypes[[1]byte], reflectBenchFuncTypes[[2]byte],
		reflectBenchFuncTypes[[3]byte], reflectBenchFuncTypes[[4]byte],
		reflectBenchFuncTypes[[5]byte], reflectBenchFuncTypes[[6]byte],
		reflectBenchFuncTypes[[7]byte], reflectBenchFuncTypes[[8]byte],
		reflectBenchFuncTypes[[9]byte], reflectBenchFuncTypes[[10]byte],
		reflectBenchFuncTypes[[11]byte], reflectBenchFuncTypes[[12]byte],
		reflectBenchFuncTypes[[13]byte], reflectBenchFuncTypes[[14]byte],
		reflectBenchFuncTypes[[15]byte], reflectBenchFuncTypes[[16]byte],
	}
	for _, tc := range []struct {
		name string
		n    int
	}{{"4-types", 1}, {"16-types", 4}, {"64-types", 16}} {
		b.Run(tc.name, func(b *testing.B) {
			var typ reflect.Type
			for _, factory := range factories[:tc.n] {
				for _, ft := range factory() {
					typ = ft
					reflectBenchPointerSink = reflect.PointerTo(ft)
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				reflectBenchPointerSink = reflect.PointerTo(typ)
			}
		})
	}
}
