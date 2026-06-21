package gotest

import "testing"

func TestMapUpdateRHSNilDerefOrder(t *testing.T) {
	var sink bool

	tests := []struct {
		name string
		run  func(map[int]int)
	}{
		{
			name: "assign",
			run: func(m map[int]int) {
				var p *int
				m[0] = *p
			},
		},
		{
			name: "add assign",
			run: func(m map[int]int) {
				var p *int
				m[0] += *p
			},
		},
		{
			name: "multi assign",
			run: func(m map[int]int) {
				var p *int
				sink, m[0] = sink, *p
			},
		},
		{
			name: "receive",
			run: func(m map[int]int) {
				var p *chan int
				m[0], sink = <-(*p)
			},
		},
		{
			name: "type assert",
			run: func(m map[int]int) {
				var p *interface{}
				m[0], sink = (*p).(int)
			},
		},
		{
			name: "map index",
			run: func(m map[int]int) {
				var p *map[int]int
				m[0], sink = (*p)[0]
			},
		},
		{
			name: "divide",
			run: func(m map[int]int) {
				var z int
				m[0] /= z
			},
		},
		{
			name: "slice index",
			run: func(m map[int]int) {
				var a []int
				m[0] = a[0]
			},
		},
		{
			name: "remainder",
			run: func(m map[int]int) {
				var z int
				m[0] %= z
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[int]int{}
			expectPanic(t, func() {
				tt.run(m)
			})
			if len(m) != 0 {
				t.Fatalf("map insert happened before RHS panic: len=%d", len(m))
			}
		})
	}
}

func TestMapUpdateAppendRHSOrder(t *testing.T) {
	var sink bool

	tests := []struct {
		name string
		run  func(map[int][]int)
	}{
		{
			name: "append nil deref",
			run: func(m map[int][]int) {
				var p *int
				m[0] = append(m[0], *p)
			},
		},
		{
			name: "multi assign append nil deref",
			run: func(m map[int][]int) {
				var p *int
				sink, m[0] = !sink, append(m[0], *p)
			},
		},
		{
			name: "append before nil deref",
			run: func(m map[int][]int) {
				var p *int
				m[0], _ = append(m[0], 0), *p
			},
		},
		{
			name: "slice bounds",
			run: func(m map[int][]int) {
				m[0] = m[0][:1]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[int][]int{}
			expectPanic(t, func() {
				tt.run(m)
			})
			if len(m) != 0 {
				t.Fatalf("map insert happened before RHS panic: len=%d", len(m))
			}
		})
	}
}

// Mirrors fixedbugs/issue23017: LHS addresses are evaluated before stores,
// then stores proceed left-to-right even when a later store panics.
func TestMultipleAssignmentMapUpdateBeforeNilStore(t *testing.T) {
	m := map[int]int{}
	var p *int

	expectPanic(t, func() {
		m[2], *p = 42, 2
	})
	if len(m) != 1 {
		t.Fatalf("map length after panic = %d, want 1", len(m))
	}
	if got := m[2]; got != 42 {
		t.Fatalf("m[2] after panic = %d, want 42", got)
	}
}

func TestMultipleAssignmentOrderBeforeLaterPanic(t *testing.T) {
	t.Run("slice index", func(t *testing.T) {
		m := map[int]int{}
		p := []int{}

		expectPanic(t, func() {
			m[2], p[1] = 2, 2
		})
		if len(m) != 1 {
			t.Fatalf("map length after panic = %d, want 1", len(m))
		}
		if got := m[2]; got != 2 {
			t.Fatalf("m[2] after panic = %d, want 2", got)
		}
	})

	t.Run("nil field store", func(t *testing.T) {
		type P struct{ i int }
		m := map[int]int{}
		var p *P

		expectPanic(t, func() {
			m[2], p.i = 3, 2
		})
		if len(m) != 1 {
			t.Fatalf("map length after panic = %d, want 1", len(m))
		}
		if got := m[2]; got != 3 {
			t.Fatalf("m[2] after panic = %d, want 3", got)
		}
	})

	t.Run("nil map after store", func(t *testing.T) {
		var m map[int]int
		var a int
		p := &a

		expectPanic(t, func() {
			*p, m[2] = 5, 2
		})
		if got := *p; got != 5 {
			t.Fatalf("*p after panic = %d, want 5", got)
		}
	})

	t.Run("nil map before store", func(t *testing.T) {
		var m map[int]int
		var g int

		expectPanic(t, func() {
			m[0], g = 1, 2
		})
		if g != 0 {
			t.Fatalf("store after nil map assignment happened: g=%d", g)
		}
	})
}

func TestMultipleAssignmentAddressUsesPreAssignmentValue(t *testing.T) {
	t.Run("field", func(t *testing.T) {
		type T struct{ i int }
		var x T
		p := &x

		p, p.i = new(T), 4
		if x.i != 4 {
			t.Fatalf("old pointee field = %d, want 4", x.i)
		}
		if p.i != 0 {
			t.Fatalf("new pointee field = %d, want 0", p.i)
		}
	})

	t.Run("nested field", func(t *testing.T) {
		type T struct{ x struct{ y int } }
		var x T
		p := &x

		p, p.x.y = new(T), 7
		if x.x.y != 7 {
			t.Fatalf("old nested field = %d, want 7", x.x.y)
		}
		if p.x.y != 0 {
			t.Fatalf("new nested field = %d, want 0", p.x.y)
		}
	})

	t.Run("nil replacement after address evaluation", func(t *testing.T) {
		type T *struct{ x struct{ y int } }
		x := struct{ y int }{}
		q := T(&struct{ x struct{ y int } }{x})
		p := q

		p, p.x.y = nil, 7
		if q.x.y != 7 {
			t.Fatalf("old nested field = %d, want 7", q.x.y)
		}
		if p != nil {
			t.Fatal("p should be nil after assignment")
		}
	})

	t.Run("swap", func(t *testing.T) {
		x, y := 1, 2
		x, y = y, x
		if x != 2 || y != 1 {
			t.Fatalf("swap = %d, %d; want 2, 1", x, y)
		}
	})
}

func expectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
}
