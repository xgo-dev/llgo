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
	Load      func(key string) ([]byte, error)
	Store     func(key string, data []byte) error
	Event     func(hit bool)
}

const cacheSchema = 1
const MaxCacheBytes = 64 << 20

type cacheFunction struct {
	fn      *ssa.Function
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
		for _, dep := range dependencies(record.fn) {
			if members[dep] {
				deps[record.id+" -> "+functions[dep].id+" "+functions[dep].key] = true
				continue
			}
			if keys[dep] == "" {
				return "", nil
			}
			deps[record.id+" -> "+functions[dep].id+" "+keys[dep]] = true
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
		id := fmt.Sprintf("%s|%s|%s", fn.String(), types.TypeString(fn.Signature, qualifier), fn.Prog.Fset.Position(fn.Pos()))
		functions = append(functions, cacheFunction{fn: fn, id: id})
	}
	sort.Slice(functions, func(i, j int) bool { return functions[i].id < functions[j].id })
	h := sha256.New()
	fmt.Fprintf(h, "%d|%d|%q|%d\n", cacheSchema, Version, a.config.Cache.Namespace, a.config.MaxStackBytes)
	base := hex.EncodeToString(h.Sum(nil))
	for i := range functions {
		h = sha256.New()
		seenTypes := make(map[types.Type]bool)
		item := &functions[i]
		fn := item.fn
		fmt.Fprintf(h, "%q %v\n", item.id, a.bodies[fn])
		hashType(h, fn.Signature, seenTypes)
		if !a.bodies[fn] {
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
		fn.WriteTo(h)
		for _, param := range fn.Params {
			hashType(h, param.Type(), seenTypes)
		}
		for _, v := range fn.FreeVars {
			hashType(h, v.Type(), seenTypes)
			fmt.Fprintf(h, "capture %v\n", a.result.CaptureByValue(v))
		}
		for _, block := range fn.Blocks {
			for _, site := range block.Instrs {
				if value, ok := site.(ssa.Value); ok {
					hashType(h, value.Type(), seenTypes)
				}
				for _, operand := range site.Operands(nil) {
					if operand != nil && *operand != nil {
						hashType(h, (*operand).Type(), seenTypes)
					}
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

func hashType(h hash.Hash, t types.Type, seen map[types.Type]bool) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	fmt.Fprintf(h, "type %q\n", types.TypeString(t, qualifier))
	switch v := t.(type) {
	case *types.Named:
		for i := 0; i < v.TypeArgs().Len(); i++ {
			hashType(h, v.TypeArgs().At(i), seen)
		}
		hashType(h, v.Underlying(), seen)
	case *types.Alias:
		hashType(h, v.Rhs(), seen)
	case *types.Pointer:
		hashType(h, v.Elem(), seen)
	case *types.Array:
		hashType(h, v.Elem(), seen)
	case *types.Slice:
		hashType(h, v.Elem(), seen)
	case *types.Map:
		hashType(h, v.Key(), seen)
		hashType(h, v.Elem(), seen)
	case *types.Chan:
		hashType(h, v.Elem(), seen)
	case *types.Struct:
		for i := 0; i < v.NumFields(); i++ {
			hashType(h, v.Field(i).Type(), seen)
		}
	case *types.Signature:
		if v.Recv() != nil {
			hashType(h, v.Recv().Type(), seen)
		}
		hashType(h, v.Params(), seen)
		hashType(h, v.Results(), seen)
	case *types.Tuple:
		for i := 0; i < v.Len(); i++ {
			hashType(h, v.At(i).Type(), seen)
		}
	case *types.Interface:
		for i := 0; i < v.NumMethods(); i++ {
			hashType(h, v.Method(i).Type(), seen)
		}
	case *types.TypeParam:
		hashType(h, v.Constraint(), seen)
	}
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
	index := 0
	for _, item := range functions {
		fn := item.fn
		if !a.bodies[fn] {
			continue
		}
		facts := payload.Functions[index]
		index++
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
		if !a.bodies[fn] {
			continue
		}
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
