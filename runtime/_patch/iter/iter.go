// llgo:skipall
package iter

// Seq is an iterator over sequences of individual values.
//
// When called as seq(yield), seq calls yield(v) for each value v in the
// sequence, stopping early if yield returns false.
type Seq[V any] func(yield func(V) bool)

// Seq2 is an iterator over sequences of pairs of values.
//
// When called as seq(yield), seq calls yield(k, v) for each pair (k, v) in the
// sequence, stopping early if yield returns false.
type Seq2[K, V any] func(yield func(K, V) bool)

// Pull converts the “push-style” iterator sequence seq into a “pull-style”
// iterator accessed by the two functions next and stop.
//
// This is a llgo-specific implementation that avoids runtime coroutines.
// It follows the stdlib contract: next/stop must not be called concurrently.
func Pull[V any](seq Seq[V]) (next func() (V, bool), stop func()) {
	type none struct{}

	resume := make(chan none) // resumes the producer (like coroswitch)
	out := make(chan V)       // yielded values
	stopCh := make(chan none) // closed to request stop
	doneCh := make(chan none) // closed when producer returns

	var (
		yieldNext  bool
		finished   bool
		panicValue any
	)

	go func() {
		defer close(doneCh)
		// Match runtime.newcoro: do not start until the first next/stop.
		<-resume
		select {
		case <-stopCh:
			close(out)
			return
		default:
		}

		defer func() {
			if p := recover(); p != nil {
				panicValue = p
			}
			close(out)
		}()

		yield := func(v V) bool {
			select {
			case <-stopCh:
				return false
			default:
			}
			if !yieldNext {
				panic("iter.Pull: yield called again before next")
			}
			yieldNext = false
			out <- v
			<-resume
			select {
			case <-stopCh:
				return false
			default:
				return true
			}
		}

		seq(yield)
	}()

	next = func() (v V, ok bool) {
		if finished {
			return
		}
		if yieldNext {
			panic("iter.Pull: next called again before yield")
		}
		yieldNext = true
		resume <- none{}

		v, ok = <-out
		if !ok {
			finished = true
			if panicValue != nil {
				p := panicValue
				panicValue = nil
				panic(p)
			}
			return
		}
		return v, true
	}

	stop = func() {
		if finished {
			return
		}
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
		resume <- none{}
		<-doneCh
		finished = true
		if panicValue != nil {
			p := panicValue
			panicValue = nil
			panic(p)
		}
	}

	return next, stop
}

// Pull2 is like Pull but for Seq2.
// Like Pull, next/stop must not be called concurrently.
func Pull2[K, V any](seq Seq2[K, V]) (next func() (K, V, bool), stop func()) {
	type none struct{}
	type pair struct {
		k K
		v V
	}

	resume := make(chan none)
	out := make(chan pair)
	stopCh := make(chan none)
	doneCh := make(chan none)

	var (
		yieldNext  bool
		finished   bool
		panicValue any
	)

	go func() {
		defer close(doneCh)
		<-resume
		select {
		case <-stopCh:
			close(out)
			return
		default:
		}

		defer func() {
			if p := recover(); p != nil {
				panicValue = p
			}
			close(out)
		}()

		yield := func(k K, v V) bool {
			select {
			case <-stopCh:
				return false
			default:
			}
			if !yieldNext {
				panic("iter.Pull2: yield called again before next")
			}
			yieldNext = false
			out <- pair{k: k, v: v}
			<-resume
			select {
			case <-stopCh:
				return false
			default:
				return true
			}
		}

		seq(yield)
	}()

	next = func() (k K, v V, ok bool) {
		if finished {
			return
		}
		if yieldNext {
			panic("iter.Pull2: next called again before yield")
		}
		yieldNext = true
		resume <- none{}

		p, ok := <-out
		if !ok {
			finished = true
			if panicValue != nil {
				p := panicValue
				panicValue = nil
				panic(p)
			}
			return
		}
		return p.k, p.v, true
	}

	stop = func() {
		if finished {
			return
		}
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
		resume <- none{}
		<-doneCh
		finished = true
		if panicValue != nil {
			p := panicValue
			panicValue = nil
			panic(p)
		}
	}

	return next, stop
}
