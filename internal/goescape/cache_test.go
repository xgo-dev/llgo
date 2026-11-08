package goescape

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/types"
	"reflect"
	"testing"

	"golang.org/x/tools/go/ssa"
)

type memoryCache struct {
	data         map[string][]byte
	hits, misses int
	cache        Cache
}

func newMemoryCache() *memoryCache {
	m := &memoryCache{data: make(map[string][]byte)}
	m.cache = Cache{Namespace: "test-target", Load: func(key string) ([]byte, error) { return m.data[key], nil }, Store: func(key string, data []byte) error { m.data[key] = append([]byte(nil), data...); return nil }, Event: func(hit bool) {
		if hit {
			m.hits++
		} else {
			m.misses++
		}
	}}
	return m
}

func assertSameFacts(t *testing.T, a, b *Result) {
	t.Helper()
	other := make(map[string]*ssa.Function)
	for fn := range b.summaries {
		other[fn.String()] = fn
	}
	if len(a.summaries) != len(b.summaries) {
		t.Fatal("summary inventory changed")
	}
	for fn, summary := range a.summaries {
		current := other[fn.String()]
		if current == nil || !reflect.DeepEqual(summary, b.summaries[current]) || a.StackBytes(fn) != b.StackBytes(current) {
			t.Fatalf("summary or budget changed for %s", fn)
		}
		for i, v := range fn.FreeVars {
			if a.CaptureByValue(v) != b.CaptureByValue(current.FreeVars[i]) {
				t.Fatal("capture representation changed")
			}
		}
		for i, block := range fn.Blocks {
			for j, site := range block.Instrs {
				x, ok := a.LookupDecision(site)
				y, found := b.LookupDecision(current.Blocks[i].Instrs[j])
				if ok != found || x != y {
					t.Fatalf("placement changed for %s: %+v / %+v", site, x, y)
				}
			}
		}
	}
	var first, second bytes.Buffer
	level := func(*ssa.Function) int { return 2 }
	a.WriteReport(&first, level)
	b.WriteReport(&second, level)
	if first.String() != second.String() {
		t.Fatal("cache changed diagnostics")
	}
}

func TestCacheRebindsCompleteFacts(t *testing.T) {
	const source = `func use(*int){};func f(x int)func()int{p:=new(int);use(p);y:=x+1;return func()int{return y}}`
	m := newMemoryCache()
	config := Config{Cache: &m.cache, StackLayout: testStackLayout, MaxStackBytes: 64}
	first := Analyze([]*ssa.Function{sourceFunction(t, source)}, config)
	if m.hits != 0 || m.misses == 0 || len(m.data) == 0 {
		t.Fatal("cold cache not populated")
	}
	m.hits, m.misses = 0, 0
	second := Analyze([]*ssa.Function{sourceFunction(t, source)}, config)
	if m.hits == 0 || m.misses != 0 {
		t.Fatalf("fresh SSA did not reuse facts: %d/%d", m.hits, m.misses)
	}
	assertSameFacts(t, first, second)
	config.Cache = nil
	assertSameFacts(t, first, Analyze([]*ssa.Function{sourceFunction(t, source)}, config))
}

func TestCacheReusesUnchangedComponents(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		hits, misses        int
		heap                bool
	}{
		{"caller", `func dep(*int){};func other(p *int)int{return *p};func f(x int){dep(&x)}`, `func dep(*int){};func other(p *int)int{return *p};func f(x int){if x==0{return};dep(&x)}`, 2, 1, false},
		{"dependency", `var sink *int;func f(x int){dep(&x)};func other(p *int)int{return *p};func dep(p *int){}`, `var sink *int;func f(x int){dep(&x)};func other(p *int)int{return *p};func dep(p *int){sink=p}`, 1, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMemoryCache()
			first := sourceFunction(t, tc.before)
			Analyze([]*ssa.Function{first, first.Pkg.Func("other")}, Config{Cache: &m.cache})
			m.hits, m.misses = 0, 0
			fn := sourceFunction(t, tc.after)
			r := Analyze([]*ssa.Function{fn, fn.Pkg.Func("other")}, Config{Cache: &m.cache})
			if m.hits != tc.hits || m.misses != tc.misses {
				t.Fatalf("hits/misses=%d/%d want %d/%d", m.hits, m.misses, tc.hits, tc.misses)
			}
			if d := r.Decision(sitesOf[*ssa.Alloc](fn)[0]); d.Heap != tc.heap {
				t.Fatalf("stale dependency effects: %+v", d)
			}
		})
	}
}

func TestCacheNamedTypeAndTargetInvalidation(t *testing.T) {
	m := newMemoryCache()
	first := sourceFunction(t, `type T struct{V *int};func f(v T)T{return v}`)
	r := Analyze([]*ssa.Function{first}, Config{Cache: &m.cache})
	if s, _ := r.Summary(first); s.Parameters[0].Results[0] != 1 {
		t.Fatal("missing pointer result flow")
	}
	m.hits, m.misses = 0, 0
	second := sourceFunction(t, `type T struct{V uintptr};func f(v T)T{return v}`)
	r = Analyze([]*ssa.Function{second}, Config{Cache: &m.cache})
	if m.hits != 0 || m.misses != 1 {
		t.Fatal("named type body did not invalidate cache")
	}
	if s, _ := r.Summary(second); s.Parameters[0].Results[0] != 0 {
		t.Fatal("stale pointer classification")
	}
	m.cache.Namespace = "other-target"
	m.hits, m.misses = 0, 0
	Analyze([]*ssa.Function{second}, Config{Cache: &m.cache})
	if m.hits != 0 || m.misses != 1 {
		t.Fatal("target identity did not invalidate cache")
	}
}

func TestCacheRejectsIncompleteAndCorruptFacts(t *testing.T) {
	for _, mutate := range []struct {
		name  string
		apply func(*cachePayload)
	}{
		{"version", func(p *cachePayload) { p.Version-- }},
		{"schema", func(p *cachePayload) { p.Schema++ }},
		{"missing placement", func(p *cachePayload) { p.Functions[0].Objects = nil }},
		{"summary dimensions", func(p *cachePayload) { p.Functions[0].Summary.Parameters = nil }},
		{"stack limit", func(p *cachePayload) { p.Functions[0].StackBytes = 65 }},
		{"capture layout", func(p *cachePayload) { p.Functions[0].Captures = []bool{true} }},
		{"invalid borrow", func(p *cachePayload) { p.Functions[0].Objects[0].ReadOnly = true }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			fn := sourceFunction(t, `func f(x int)*int{return &x}`)
			m := newMemoryCache()
			config := Config{Cache: &m.cache, MaxStackBytes: 64, StackLayout: testStackLayout}
			want := Analyze([]*ssa.Function{fn}, config)
			for key, data := range m.data {
				var env cacheEnvelope
				json.Unmarshal(data, &env)
				var payload cachePayload
				json.Unmarshal(env.Payload, &payload)
				mutate.apply(&payload)
				env.Payload, _ = json.Marshal(payload)
				env.Digest = digest(env.Payload)
				m.data[key], _ = json.Marshal(env)
			}
			m.hits, m.misses = 0, 0
			got := Analyze([]*ssa.Function{fn}, config)
			if m.hits != 0 || m.misses != 1 {
				t.Fatalf("invalid facts accepted: %d/%d", m.hits, m.misses)
			}
			assertSameFacts(t, want, got)
		})
	}
	for _, data := range [][]byte{[]byte(`{"Digest":"bad","Payload":{}}`), []byte(`{"Digest":"x"} trailing`), nil} {
		fn := sourceFunction(t, `func f(x int)*int{return &x}`)
		m := newMemoryCache()
		m.cache.Load = func(string) ([]byte, error) { return data, nil }
		r := Analyze([]*ssa.Function{fn}, Config{Cache: &m.cache})
		if m.hits != 0 || !r.Decision(sitesOf[*ssa.Alloc](fn)[0]).Heap {
			t.Fatal("corrupt data bypassed analysis")
		}
	}
}

func TestCacheIOFailureIsConservative(t *testing.T) {
	fn := sourceFunction(t, `func f(x int)*int{return &x}`)
	cache := &Cache{Load: func(string) ([]byte, error) { return nil, errors.New("unreadable") }, Store: func(string, []byte) error { return errors.New("read-only") }}
	if d := Analyze([]*ssa.Function{fn}, Config{Cache: cache}).Decision(sitesOf[*ssa.Alloc](fn)[0]); !d.Heap {
		t.Fatal("I/O failure lost escape", d)
	}
}

func TestCacheSeparatesEqualNamesWithDifferentCallees(t *testing.T) {
	// Source replacements can create distinct method wrappers with the same
	// printed identity. Equal local bodies still need their actual dependency
	// keys, rather than a global name lookup or disabling every cache entry.
	const borrow = `var sink *int;func f(x int)int{dep(&x);return x};func dep(p *int){}`
	const retain = `var sink *int;func f(x int)int{dep(&x);return x};func dep(p *int){sink=p}`
	m := newMemoryCache()
	for range 2 {
		first, second := sourceFunction(t, borrow), sourceFunction(t, retain)
		r := Analyze([]*ssa.Function{first, second}, Config{Cache: &m.cache})
		if r.Decision(sitesOf[*ssa.Alloc](first)[0]).Heap || !r.Decision(sitesOf[*ssa.Alloc](second)[0]).Heap {
			t.Fatal("names replaced actual call edges")
		}
	}
	if m.hits == 0 {
		t.Fatal("equal printed names disabled unrelated cache reuse")
	}
}

func TestCachePolicyInvalidation(t *testing.T) {
	for _, policy := range []string{"heap", "layout", "external"} {
		t.Run(policy, func(t *testing.T) {
			source := `func use(p *int)int{return *p};func f(x int)int{return use(&x)}`
			if policy == "external" {
				source = `func use(*int);func f(x int){use(&x)}`
			}
			fn := sourceFunction(t, source)
			m := newMemoryCache()
			changed := false
			config := Config{Cache: &m.cache, MaxStackBytes: 8, StackLayout: func(ssa.Instruction, types.Type) (uint64, uint64) {
				if policy == "layout" && changed {
					return 16, 8
				}
				return 8, 8
			}, ForceHeap: func(ssa.Instruction, types.Type) string {
				if policy == "heap" && changed {
					return "backend constraint"
				}
				return ""
			}, External: func(f *ssa.Function) (Summary, bool) {
				s := emptySummary(f)
				if changed {
					s.Parameters[0].Heap = 1
				}
				return s, true
			}}
			first := Analyze([]*ssa.Function{fn}, config)
			if first.Decision(sitesOf[*ssa.Alloc](fn)[0]).Heap {
				t.Fatal("missing baseline borrow")
			}
			changed = true
			m.hits, m.misses = 0, 0
			after := Analyze([]*ssa.Function{fn}, config)
			if m.misses == 0 || !after.Decision(sitesOf[*ssa.Alloc](fn)[0]).Heap {
				t.Fatal("stale policy proof")
			}
		})
	}
}
