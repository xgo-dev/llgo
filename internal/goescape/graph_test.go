package goescape

import "testing"

func TestObjectAndPayloadLifetimes(t *testing.T) {
	for _, tc := range []struct {
		name         string
		depth        int
		box, payload bool
	}{
		{"publish box", -1, true, true},
		{"copy payload", 0, false, true},
		{"read scalar", 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGraph()
			fn, scope := new(function), new(lifetime)
			box := g.location("box", fn, scope)
			payload := g.location("payload", fn, scope)
			g.flow(box, payload, -1)
			g.flow(&g.heap, box, tc.depth)
			g.solve()
			if got := box.attrs&escapes != 0; got != tc.box {
				t.Errorf("box escapes=%v, want %v", got, tc.box)
			}
			if got := payload.attrs&escapes != 0; got != tc.payload {
				t.Errorf("payload escapes=%v, want %v", got, tc.payload)
			}
		})
	}
}

func TestParameterEffectsAndResults(t *testing.T) {
	g := newGraph()
	fn, scope := new(function), new(lifetime)
	p := g.location("p", fn, scope)
	p.param = true
	r := g.location("result", fn, scope)
	r.result = 1
	g.flow(r, p, 0)
	g.flow(&g.mutator, p, 1)
	g.flow(&g.callee, p, 2)
	g.solve()
	if p.leaks.heap != 0 || p.leaks.results[0].depth() != 0 || p.leaks.mutator.depth() != 1 || p.leaks.callee.depth() != 2 {
		t.Fatalf("effects=%+v", p.leaks)
	}
	if p.attrs&escapes != 0 {
		t.Fatal("returning a parameter value escaped its local storage")
	}
}

func TestLoopRetentionAndPersistence(t *testing.T) {
	for _, retained := range []bool{false, true} {
		g := newGraph()
		fn := new(function)
		outer := new(lifetime)
		inner := &lifetime{parent: outer}
		object := g.location("iteration object", fn, inner)
		scope := inner
		if retained {
			scope = outer
		}
		dst := g.location("user", fn, scope)
		dst.attrs = persists
		g.flow(dst, object, -1)
		g.solve()
		if (object.attrs&escapes != 0) != retained {
			t.Fatalf("retained=%v attrs=%v", retained, object.attrs)
		}
		if object.attrs&persists == 0 {
			t.Fatal("lost persistence")
		}
	}
}

func TestAddressCyclesConverge(t *testing.T) {
	g := newGraph()
	fn, scope := new(function), new(lifetime)
	a := g.location("a", fn, scope)
	b := g.location("b", fn, scope)
	g.flow(a, b, -1)
	g.flow(b, a, -1)
	g.flow(&g.heap, a, 0)
	g.solve()
	if a.attrs&escapes == 0 || b.attrs&escapes == 0 {
		t.Fatal("cycle lost an escaping address")
	}
}

func TestClosureReturnLifetime(t *testing.T) {
	for _, lost := range []bool{false, true} {
		g := newGraph()
		outer := new(function)
		child := &function{parent: outer, resultsLost: lost}
		object := g.location("outer object", outer, new(lifetime))
		result := g.location("closure result", child, new(lifetime))
		result.result = 1
		g.flow(result, object, -1)
		g.solve()
		if (object.attrs&escapes != 0) != lost {
			t.Fatalf("lost=%v attrs=%v", lost, object.attrs)
		}
	}
}

func TestLeakDepthSaturatesConservatively(t *testing.T) {
	var l leak
	l.add(1000000)
	if l.depth() != 65534 {
		t.Fatal(l)
	}
	l.add(2)
	l.add(3)
	if l.depth() != 2 {
		t.Fatal(l)
	}
}

func TestUnknownClosureCallsRevisitResults(t *testing.T) {
	g := newGraph()
	outer := new(function)
	child := &function{parent: outer}
	object := g.location("outer object", outer, new(lifetime))
	result := g.location("closure result", child, new(lifetime))
	result.result = 1
	child.results = []*location{result}
	environment := g.location("closure environment", outer, new(lifetime))
	environment.closure = child
	g.flow(result, object, -1)
	g.solve()
	if object.attrs&escapes != 0 {
		t.Fatal("tracked result escaped before an unknown call")
	}
	g.flow(&g.callee, environment, -1)
	g.solve()
	if !child.resultsLost || object.attrs&escapes == 0 {
		t.Fatal("unknown closure call did not invalidate the returned alias proof")
	}
}

func TestSiblingLoopScopesRetainObjects(t *testing.T) {
	g := newGraph()
	fn := new(function)
	root := new(lifetime)
	first, second := &lifetime{parent: root}, &lifetime{parent: root}
	object := g.location("first loop object", fn, first)
	user := g.location("second loop value", fn, second)
	g.flow(user, object, -1)
	g.solve()
	if object.attrs&escapes == 0 {
		t.Fatal("separate iteration scopes were treated as one reusable lifetime")
	}
}
