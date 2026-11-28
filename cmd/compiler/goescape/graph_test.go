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
			scope := new(lifetime)
			box := g.location("box", scope)
			payload := g.location("payload", scope)
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
	scope := new(lifetime)
	p := g.location("p", scope)
	p.param = true
	r := g.location("result", scope)
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
		outer := new(lifetime)
		inner := &lifetime{parent: outer}
		object := g.location("iteration object", inner)
		scope := inner
		if retained {
			scope = outer
		}
		dst := g.location("user", scope)
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
	scope := new(lifetime)
	a := g.location("a", scope)
	b := g.location("b", scope)
	g.flow(a, b, -1)
	g.flow(b, a, -1)
	g.flow(&g.heap, a, 0)
	g.solve()
	if a.attrs&escapes == 0 || b.attrs&escapes == 0 {
		t.Fatal("cycle lost an escaping address")
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

func TestSiblingLoopScopesRetainObjects(t *testing.T) {
	g := newGraph()
	root := new(lifetime)
	first, second := &lifetime{parent: root}, &lifetime{parent: root}
	object := g.location("first loop object", first)
	user := g.location("second loop value", second)
	g.flow(user, object, -1)
	g.solve()
	if object.attrs&escapes == 0 {
		t.Fatal("separate iteration scopes were treated as one reusable lifetime")
	}
}
