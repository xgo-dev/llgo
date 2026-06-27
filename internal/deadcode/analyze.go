package deadcode

import (
	"fmt"
	"go/token"
	"os"
	"sort"
	"time"

	"github.com/goplus/llgo/internal/meta"
)

type ifaceMethodKey struct {
	iface meta.Symbol
	sig   meta.GMethodSig
}

type ifaceMethodName struct {
	iface meta.Symbol
	name  meta.Name
}

type methodID struct {
	owner meta.Symbol
	slot  int
}

type methodRef struct {
	owner    meta.Symbol
	slot     int
	slotInfo meta.GMethodSlot
}

type pass struct {
	info *meta.GlobalSummary

	methodImplKeys    map[methodID][]ifaceMethodKey
	methodRefs        map[meta.GMethodSig][]meta.Symbol // sig → []iface (built eagerly)
	ifaceMethodCounts map[meta.Symbol]int               // iface → unique method name count
	typeSymbols       map[meta.Symbol]struct{}

	reachable        map[meta.Symbol]struct{}
	usedInIface      map[meta.Symbol]struct{}
	processedIfaceTy map[meta.Symbol]struct{}
	workQueue        []meta.Symbol

	ifaceMethod        map[ifaceMethodKey]struct{}
	genericIfaceMethod map[meta.Name]struct{}
	reflectSeen        bool

	markableMethods []methodRef
	markedMethods   map[methodID]struct{}
	liveSlots       map[meta.Symbol][]int
}

// Analyze returns live ABI method slot indexes by concrete type symbol name.
func Analyze(info *meta.GlobalSummary, rootNames []string) map[string][]int {
	roots := make([]meta.Symbol, 0, len(rootNames))
	for _, name := range rootNames {
		if sym, ok := info.LookupSymbol(name); ok {
			roots = append(roots, sym)
		}
	}

	liveSlots := deadcode(info, roots)
	out := make(map[string][]int, len(liveSlots))
	for typ, slots := range liveSlots {
		name := info.SymbolName(typ)
		sorted := append([]int(nil), slots...)
		sort.Ints(sorted)
		out[name] = sorted
	}
	return out
}

func deadcode(info *meta.GlobalSummary, roots []meta.Symbol) map[meta.Symbol][]int {
	d := &pass{
		info:               info,
		methodImplKeys:     make(map[methodID][]ifaceMethodKey),
		methodRefs:         make(map[meta.GMethodSig][]meta.Symbol),
		ifaceMethodCounts:  make(map[meta.Symbol]int),
		typeSymbols:        make(map[meta.Symbol]struct{}),
		reachable:          make(map[meta.Symbol]struct{}),
		usedInIface:        make(map[meta.Symbol]struct{}),
		processedIfaceTy:   make(map[meta.Symbol]struct{}),
		ifaceMethod:        make(map[ifaceMethodKey]struct{}),
		genericIfaceMethod: make(map[meta.Name]struct{}),
		markedMethods:      make(map[methodID]struct{}),
		liveSlots:          make(map[meta.Symbol][]int),
	}
	d.buildMethodRefs()

	for _, root := range roots {
		d.markReachable(root)
	}

	for {
		d.flood()
		changed := d.methodMarkingLoop()
		if len(d.workQueue) == 0 && !changed {
			return d.liveSlots
		}
	}
}

// buildMethodRefs builds the methodRefs reverse index (sig → []iface) from all
// interfaces. This is cheap (tens of interfaces, hundreds of sigs) and must be
// eager — every concrete type needs it to check implementation relationships.
//
// Concrete type methodImplKeys are NOT computed here. They are built lazily in
// computeMethodImplKeys when a type first enters usedInIface.
func (d *pass) buildMethodRefs() {
	t0 := time.Now()
	for _, iface := range d.info.Interfaces() {
		d.typeSymbols[iface] = struct{}{}
		seenNames := make(map[meta.Name]struct{})
		for _, sig := range d.info.InterfaceMethods(iface) {
			d.methodRefs[sig] = appendSymbolUnique(d.methodRefs[sig], iface)
			if _, ok := seenNames[sig.Name]; ok {
				continue
			}
			seenNames[sig.Name] = struct{}{}
			d.ifaceMethodCounts[iface]++
		}
	}
	t1 := time.Now()
	fmt.Fprintf(os.Stderr, "[dce] methodRefs index: interfaces=%d total_sigs=%d %v\n",
		len(d.info.Interfaces()), len(d.methodRefs), t1.Sub(t0))
}

// computeMethodImplKeys lazily builds methodImplKeys for a single concrete type
// that has entered usedInIface. Called at most once per type.
func (d *pass) computeMethodImplKeys(typ meta.Symbol, slots []meta.GMethodSlot) {
	if _, done := d.methodImplKeys[methodID{owner: typ, slot: 0}]; done {
		// Already computed — check skip by looking at slot 0. If slot 0 has
		// an entry, the whole type was processed (we always process all slots
		// of a type at once).
		// Fine print: a type with 0 slots will never reach here because
		// markUsedInIface only calls this when slots is non-empty.
		return
	}
	// Mark the type and compute all slots at once.
	d.typeSymbols[typ] = struct{}{}
	impls := make(map[meta.Symbol]int)
	seen := make(map[ifaceMethodName]struct{})

	for _, slot := range slots {
		sig := meta.GMethodSig{Name: slot.Name, MType: slot.MType}
		for _, iface := range d.methodRefs[sig] {
			key := ifaceMethodName{iface: iface, name: slot.Name}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			impls[iface]++
		}
	}

	for slotIndex, slot := range slots {
		id := methodID{owner: typ, slot: slotIndex}
		sig := meta.GMethodSig{Name: slot.Name, MType: slot.MType}
		for _, iface := range d.methodRefs[sig] {
			if impls[iface] == d.ifaceMethodCounts[iface] {
				key := ifaceMethodKey{iface: iface, sig: sig}
				d.methodImplKeys[id] = append(d.methodImplKeys[id], key)
			}
		}
	}
}

func (d *pass) flood() {
	for len(d.workQueue) > 0 {
		sym := d.popWork()

		if d.info.HasReflectMethod(sym) {
			d.reflectSeen = true
		}

		_, usedInIface := d.usedInIface[sym]
		for _, dst := range d.info.OrdinaryEdges(sym) {
			if usedInIface {
				d.markTypeUsedInIface(dst)
			}
			d.markReachable(dst)
		}

		for _, typ := range d.info.UseIface(sym) {
			d.markUsedInIface(typ)
		}

		for _, demand := range d.info.UseIfaceMethod(sym) {
			key := ifaceMethodKey{iface: demand.Target, sig: demand.Sig}
			d.ifaceMethod[key] = struct{}{}
		}

		for _, name := range d.info.UseNamedMethod(sym) {
			d.genericIfaceMethod[name] = struct{}{}
		}

		if _, used := d.usedInIface[sym]; used {
			if _, processed := d.processedIfaceTy[sym]; !processed {
				d.processedIfaceTy[sym] = struct{}{}
				slots := d.info.MethodSlots(sym)
				if len(slots) > 0 {
					d.computeMethodImplKeys(sym, slots)
				}
				for slot, slotInfo := range slots {
					d.markableMethods = append(d.markableMethods, methodRef{
						owner:    sym,
						slot:     slot,
						slotInfo: slotInfo,
					})
				}
			}
		}
	}
}

func (d *pass) methodMarkingLoop() bool {
	changed := false
	rem := d.markableMethods[:0]

	for _, method := range d.markableMethods {
		if d.shouldKeep(method) {
			if d.markMethod(method) {
				changed = true
			}
			continue
		}
		rem = append(rem, method)
	}

	d.markableMethods = rem
	return changed
}

func (d *pass) shouldKeep(method methodRef) bool {
	if d.reflectSeen && token.IsExported(d.info.Name(method.slotInfo.Name)) {
		return true
	}

	if _, ok := d.genericIfaceMethod[method.slotInfo.Name]; ok {
		return true
	}

	id := methodID{owner: method.owner, slot: method.slot}
	for _, key := range d.methodImplKeys[id] {
		if _, ok := d.ifaceMethod[key]; ok {
			return true
		}
	}
	return false
}

func (d *pass) markMethod(method methodRef) bool {
	id := methodID{owner: method.owner, slot: method.slot}
	if _, ok := d.markedMethods[id]; ok {
		return false
	}
	d.markedMethods[id] = struct{}{}
	d.liveSlots[method.owner] = append(d.liveSlots[method.owner], method.slot)

	d.markReachable(method.slotInfo.MType)
	d.markReachable(method.slotInfo.IFn)
	d.markReachable(method.slotInfo.TFn)
	return true
}

func (d *pass) markReachable(sym meta.Symbol) {
	if _, ok := d.reachable[sym]; ok {
		return
	}
	d.reachable[sym] = struct{}{}
	d.workQueue = append(d.workQueue, sym)
}

func (d *pass) markUsedInIface(typ meta.Symbol) {
	if _, ok := d.usedInIface[typ]; ok {
		return
	}
	d.usedInIface[typ] = struct{}{}
	if _, ok := d.reachable[typ]; ok {
		d.workQueue = append(d.workQueue, typ)
	}
	for _, child := range d.info.TypeChildren(typ) {
		d.markUsedInIface(child)
	}
}

func (d *pass) markTypeUsedInIface(sym meta.Symbol) {
	if _, ok := d.typeSymbols[sym]; ok {
		d.markUsedInIface(sym)
		return
	}
	// Lazy check: interfaces are in typeSymbols, concrete types are detected
	// by the presence of MethodSlots.
	if len(d.info.MethodSlots(sym)) > 0 {
		d.typeSymbols[sym] = struct{}{}
		d.markUsedInIface(sym)
	}
}

func (d *pass) popWork() meta.Symbol {
	sym := d.workQueue[0]
	copy(d.workQueue, d.workQueue[1:])
	d.workQueue = d.workQueue[:len(d.workQueue)-1]
	return sym
}

func appendSymbolUnique(items []meta.Symbol, item meta.Symbol) []meta.Symbol {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

