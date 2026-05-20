package deadcode

import (
	"go/token"
	"sort"

	"github.com/goplus/llgo/internal/metadata"
)

type ifaceMethodKey struct {
	iface metadata.Symbol
	sig   metadata.MethodSig
}

type ifaceMethodName struct {
	iface metadata.Symbol
	name  metadata.Name
}

type methodID struct {
	owner metadata.Symbol
	slot  int
}

type methodRef struct {
	owner    metadata.Symbol
	slot     int
	slotInfo metadata.MethodSlot
}

type pass struct {
	info *metadata.GlobalSummary

	methodImplKeys map[methodID][]ifaceMethodKey

	reachable        map[metadata.Symbol]struct{}
	usedInIface      map[metadata.Symbol]struct{}
	processedIfaceTy map[metadata.Symbol]struct{}
	workQueue        []metadata.Symbol

	ifaceMethod        map[ifaceMethodKey]struct{}
	genericIfaceMethod map[metadata.Name]struct{}
	reflectSeen        bool

	markableMethods []methodRef
	markedMethods   map[methodID]struct{}
	liveSlots       map[metadata.Symbol][]int
}

// Analyze returns live ABI method slot indexes by concrete type symbol name.
func Analyze(info *metadata.GlobalSummary, rootNames []string) map[string][]int {
	roots := make([]metadata.Symbol, 0, len(rootNames))
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

func deadcode(info *metadata.GlobalSummary, roots []metadata.Symbol) map[metadata.Symbol][]int {
	d := &pass{
		info:               info,
		methodImplKeys:     make(map[methodID][]ifaceMethodKey),
		reachable:          make(map[metadata.Symbol]struct{}),
		usedInIface:        make(map[metadata.Symbol]struct{}),
		processedIfaceTy:   make(map[metadata.Symbol]struct{}),
		ifaceMethod:        make(map[ifaceMethodKey]struct{}),
		genericIfaceMethod: make(map[metadata.Name]struct{}),
		markedMethods:      make(map[methodID]struct{}),
		liveSlots:          make(map[metadata.Symbol][]int),
	}
	d.buildMethodImplKeys()

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

func (d *pass) buildMethodImplKeys() {
	methodRefs := make(map[metadata.MethodSig][]metadata.Symbol)
	ifaceMethodCounts := make(map[metadata.Symbol]int)
	for _, iface := range d.info.Interfaces() {
		seenNames := make(map[metadata.Name]struct{})
		for _, sig := range d.info.InterfaceMethods(iface) {
			methodRefs[sig] = appendSymbolUnique(methodRefs[sig], iface)
			if _, ok := seenNames[sig.Name]; ok {
				continue
			}
			seenNames[sig.Name] = struct{}{}
			ifaceMethodCounts[iface]++
		}
	}

	for _, typ := range d.info.ConcreteTypes() {
		slots := d.info.MethodSlots(typ)
		impls := make(map[metadata.Symbol]int)
		seen := make(map[ifaceMethodName]struct{})

		for _, slot := range slots {
			for _, iface := range methodRefs[slot.Sig] {
				key := ifaceMethodName{iface: iface, name: slot.Sig.Name}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				impls[iface]++
			}
		}

		for slotIndex, slot := range slots {
			id := methodID{owner: typ, slot: slotIndex}
			for _, iface := range methodRefs[slot.Sig] {
				if impls[iface] == ifaceMethodCounts[iface] {
					key := ifaceMethodKey{iface: iface, sig: slot.Sig}
					d.methodImplKeys[id] = append(d.methodImplKeys[id], key)
				}
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

		for _, dst := range d.info.OrdinaryEdges(sym) {
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
				for slot, slotInfo := range d.info.MethodSlots(sym) {
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
	if d.reflectSeen && token.IsExported(d.info.Name(method.slotInfo.Sig.Name)) {
		return true
	}

	if _, ok := d.genericIfaceMethod[method.slotInfo.Sig.Name]; ok {
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

	d.markReachable(method.slotInfo.Sig.MType)
	d.markReachable(method.slotInfo.IFn)
	d.markReachable(method.slotInfo.TFn)
	return true
}

func (d *pass) markReachable(sym metadata.Symbol) {
	if _, ok := d.reachable[sym]; ok {
		return
	}
	d.reachable[sym] = struct{}{}
	d.workQueue = append(d.workQueue, sym)
}

func (d *pass) markUsedInIface(typ metadata.Symbol) {
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

func (d *pass) popWork() metadata.Symbol {
	sym := d.workQueue[0]
	copy(d.workQueue, d.workQueue[1:])
	d.workQueue = d.workQueue[:len(d.workQueue)-1]
	return sym
}

func appendSymbolUnique(items []metadata.Symbol, item metadata.Symbol) []metadata.Symbol {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
