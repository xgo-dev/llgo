package goescape

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"hash"
	"io"
	"sort"

	"golang.org/x/tools/go/ssa"
)

// Cache is optional local build-cache storage. I/O errors and invalid records
// fall back to analysis. Callbacks run on the coordinator, never backend workers.
type Cache struct {
	Namespace string
	// BodyKey optionally identifies the exact source/transforms of fn. It must
	// change whenever its SSA operations or operands change. Empty keys use the
	// complete SSA dump. Types, layout policies and dependencies remain hashed.
	BodyKey func(fn *ssa.Function) string
	Load    func(key string) ([]byte, error)
	Store   func(key string, data []byte) error
	Event   func(hit bool)
}

const cacheSchema = 1
const MaxCacheBytes = 64 << 20

type cacheFunction struct {
	fn      *ssa.Function
	name    string
	id      string
	key     string
	objects []ssa.Instruction
}

func (a *analyzer) componentCache(base string, functions map[*ssa.Function]cacheFunction, keys map[*ssa.Function]string, group []*ssa.Function) (string, []cacheFunction) {
	if base == "" {
		return "", nil
	}
	members := make(map[*ssa.Function]bool, len(group))
	var records []cacheFunction
	for _, fn := range group {
		members[fn] = true
		records = append(records, functions[fn])
	}
	sort.Slice(records, func(i, j int) bool { return records[i].id < records[j].id })
	for i := 1; i < len(records); i++ {
		if records[i-1].id == records[i].id {
			return "", nil
		}
	}
	h := sha256.New()
	fmt.Fprintln(h, base)
	deps := make(map[string]bool)
	for _, record := range records {
		fmt.Fprintf(h, "body %s %s\n", record.id, record.key)
		named := make(map[string]string)
		for _, dep := range a.dependencies(record.fn) {
			depKey := keys[dep]
			if members[dep] {
				depKey = functions[dep].key
			}
			if depKey == "" {
				return "", nil
			}
			id := functions[dep].id
			// Printed SSA names cannot distinguish replacement wrappers with
			// identical identities. A dependency set loses which operand uses
			// which wrapper, so do not cache an ambiguous caller or its users.
			name := functions[dep].name
			identity := id + " " + depKey
			if previous, ok := named[name]; ok && previous != identity {
				return "", nil
			}
			named[name] = identity
			deps[record.id+" -> "+id+" "+depKey] = true
		}
	}
	var sorted []string
	for dep := range deps {
		sorted = append(sorted, dep)
	}
	sort.Strings(sorted)
	for _, dep := range sorted {
		fmt.Fprintln(h, "dependency", dep)
	}
	pkgPath := "synthetic"
	owner := records[0].fn
	for owner.Pkg == nil && owner.Parent() != nil {
		owner = owner.Parent()
	}
	if owner.Pkg != nil {
		pkgPath = owner.Pkg.Pkg.Path()
	}
	// A stable package bucket lets the build layer batch filesystem I/O.
	bucket := digest([]byte(base + "|" + pkgPath))
	return bucket + "." + hex.EncodeToString(h.Sum(nil)), records
}

type cachedFunction struct {
	ID         string
	Summary    Summary
	StackBytes uint64
	Captures   []bool
	Objects    []Decision
}

type cachePayload struct {
	Schema, Version int
	Key             string
	Functions       []cachedFunction
}

type cacheEnvelope struct {
	Digest  string
	Payload json.RawMessage
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// Fingerprint all analysis inputs, rather than names/signatures alone. Named
// type bodies can change pointer flow without changing size or spelling.
func (a *analyzer) cacheIdentity() (string, []cacheFunction) {
	if a.config.Cache == nil {
		return "", nil
	}
	var functions []cacheFunction
	for fn := range a.functions {
		name := fn.String()
		id := fmt.Sprintf("%s|%s|%s", name, types.TypeString(fn.Signature, qualifier), fn.Prog.Fset.Position(fn.Pos()))
		functions = append(functions, cacheFunction{fn: fn, name: name, id: id})
	}
	sort.Slice(functions, func(i, j int) bool { return functions[i].id < functions[j].id })
	h := sha256.New()
	fmt.Fprintf(h, "%d|%d|%q|%d\n", cacheSchema, Version, a.config.Cache.Namespace, a.config.MaxStackBytes)
	base := hex.EncodeToString(h.Sum(nil))
	var body bytes.Buffer
	typeKeys := make(map[types.Type][]byte)
	for i := range functions {
		h = sha256.New()
		item := &functions[i]
		fn := item.fn
		fmt.Fprintf(h, "%q %v\n", item.id, a.bodies[fn])
		hashType(h, fn.Signature, typeKeys)
		if !a.bodies[fn] {
			if a.config.ExternalKey != nil {
				if key := a.config.ExternalKey(fn); key != "" {
					item.key = key
					continue
				}
			}
			// Include replacement/intrinsic contracts, even with visible stubs.
			s, valid := Summary{}, false
			if a.config.External != nil {
				s, valid = a.config.External(fn)
			}
			valid = valid && validSummary(fn, s)
			encoded, _ := json.Marshal(s)
			fmt.Fprintf(h, "external %v %s\n", valid, encoded)
			item.key = hex.EncodeToString(h.Sum(nil))
			continue
		}
		bodyKey := ""
		if a.config.Cache.BodyKey != nil {
			bodyKey = a.config.Cache.BodyKey(fn)
		}
		if bodyKey != "" {
			fmt.Fprintf(h, "source %q\n", bodyKey)
		} else {
			body.Reset()
			ssa.WriteFunction(&body, fn)
			h.Write(body.Bytes())
		}
		for _, param := range fn.Params {
			hashType(h, param.Type(), typeKeys)
		}
		for _, v := range fn.FreeVars {
			hashType(h, v.Type(), typeKeys)
			fmt.Fprintf(h, "capture %v\n", a.result.CaptureByValue(v))
		}
		for _, block := range fn.Blocks {
			for _, site := range block.Instrs {
				if value, ok := site.(ssa.Value); ok {
					hashType(h, value.Type(), typeKeys)
				}
				for _, operand := range site.Operands(nil) {
					if operand != nil && *operand != nil {
						hashType(h, (*operand).Type(), typeKeys)
					}
				}

				switch value := site.(type) {
				case *ssa.MakeInterface:
					indirect := !directInterface(value.X.Type())
					if a.config.IndirectInterface != nil {
						indirect = a.config.IndirectInterface(value.X.Type())
					}
					fmt.Fprintln(h, "interface", indirect)
				case *ssa.TypeAssert:
					indirect := !directInterface(value.AssertedType)
					if a.config.IndirectInterface != nil {
						indirect = a.config.IndirectInterface(value.AssertedType)
					}
					fmt.Fprintf(h, "assert %T %v\n", value.AssertedType.Underlying(), indirect)
				}
				typ := a.objectType(site)
				if typ == nil {
					continue
				}
				item.objects = append(item.objects, site)
				if a.config.ForceHeap != nil {
					fmt.Fprintf(h, "heap %q\n", a.config.ForceHeap(site, typ))
				}
				if a.config.StackLayout != nil {
					size, align := a.config.StackLayout(site, typ)
					fmt.Fprintf(h, "stack %d %d\n", size, align)
				}
				if a.config.LocalStackLayout != nil {
					size, align := a.config.LocalStackLayout(site, typ)
					fmt.Fprintf(h, "local %d %d\n", size, align)
				}
			}
		}
		item.key = hex.EncodeToString(h.Sum(nil))
	}
	return base, functions
}

func qualifier(pkg *types.Package) string { return pkg.Path() }

// Fingerprint the type properties queried by this field-insensitive analysis.
// Following pointees or interface method signatures recursively is unnecessary:
// dereferenced values have their own SSA types, and resolved method bodies are
// covered by component dependency keys. Backend storage constraints are hashed
// separately. New type queries in the analyzer must update this schema/version.
func hashType(h hash.Hash, t types.Type, keys map[types.Type][]byte) {
	if t == nil {
		return
	}
	if key, ok := keys[t]; ok {
		h.Write(key)
		return
	}
	var key bytes.Buffer
	fmt.Fprintf(&key, "type %T pointers=%v direct=%v\n", t.Underlying(), containsPointers(t), directInterface(t))
	switch typ := t.Underlying().(type) {
	case *types.Basic:
		fmt.Fprintln(&key, "basic", typ.Kind())
	case *types.Array:
		fmt.Fprintln(&key, "array", typ.Len())
	case *types.Struct:
		fmt.Fprintln(&key, "fields", typ.NumFields())
	case *types.Tuple:
		fmt.Fprintln(&key, "tuple", typ.Len())
	case *types.Interface:
		for i := 0; i < typ.NumMethods(); i++ {
			m := typ.Method(i)
			fmt.Fprintln(&key, "method", m.Id())
		}
	case *types.Slice:
		b, ok := typ.Elem().Underlying().(*types.Basic)
		fmt.Fprintln(&key, "byte-slice", ok && b.Kind() == types.Byte)
	}
	keys[t] = key.Bytes()
	h.Write(key.Bytes())
}

func decodeCache(data []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing cache data")
	}
	return nil
}

func (a *analyzer) loadCache(key string, functions []cacheFunction) *Result {
	cache := a.config.Cache
	if cache == nil {
		return nil
	}
	var result *Result
	defer func() {
		if cache.Event != nil {
			cache.Event(result != nil)
		}
	}()
	if key == "" || cache.Load == nil {
		return nil
	}
	data, err := cache.Load(key)
	if err != nil || len(data) > MaxCacheBytes {
		return nil
	}
	var envelope cacheEnvelope
	if decodeCache(data, &envelope) != nil || digest(envelope.Payload) != envelope.Digest {
		return nil
	}
	var payload cachePayload
	if decodeCache(envelope.Payload, &payload) != nil || payload.Schema != cacheSchema || payload.Version != Version || payload.Key != key || len(payload.Functions) != len(functions) {
		return nil
	}
	loaded := &Result{objects: make(map[ssa.Instruction]Decision), summaries: make(map[*ssa.Function]Summary), stackBytes: make(map[*ssa.Function]uint64), captures: a.result.captures}
	// Component records contain inspected bodies only; opaque contracts are
	// represented in dependency keys, never in this positional inventory.
	for index, item := range functions {
		fn := item.fn
		facts := payload.Functions[index]
		if facts.ID != item.id || !validSummary(fn, facts.Summary) || len(facts.Objects) != len(item.objects) || len(facts.Captures) != len(fn.FreeVars) {
			return nil
		}
		if a.config.MaxStackBytes != 0 && facts.StackBytes > a.config.MaxStackBytes {
			return nil
		}
		for i, v := range fn.FreeVars {
			if facts.Captures[i] != a.result.CaptureByValue(v) {
				return nil
			}
		}
		for i, site := range item.objects {
			d := facts.Objects[i]
			if d.ReadOnly {
				if _, ok := site.(*ssa.Convert); !ok || d.Heap {
					return nil
				}
			}
			loaded.objects[site] = d
		}
		loaded.summaries[fn] = facts.Summary
		loaded.stackBytes[fn] = facts.StackBytes
	}
	result = loaded
	return result
}

func (a *analyzer) storeCache(key string, functions []cacheFunction) {
	cache := a.config.Cache
	if key == "" || cache == nil || cache.Store == nil {
		return
	}
	payload := cachePayload{Schema: cacheSchema, Version: Version, Key: key}
	for _, item := range functions {
		fn := item.fn
		facts := cachedFunction{ID: item.id, Summary: a.result.summaries[fn], StackBytes: a.result.StackBytes(fn)}
		for _, v := range fn.FreeVars {
			facts.Captures = append(facts.Captures, a.result.CaptureByValue(v))
		}
		for _, site := range item.objects {
			facts.Objects = append(facts.Objects, a.result.Decision(site))
		}
		payload.Functions = append(payload.Functions, facts)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	data, err := json.Marshal(cacheEnvelope{Digest: digest(encoded), Payload: encoded})
	if err == nil && len(data) <= MaxCacheBytes {
		cache.Store(key, data)
	}
}
