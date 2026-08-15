package deadcode

import (
	"go/token"
	"sort"

	"github.com/goplus/llgo/internal/meta"
)

type ifaceMethodKey struct {
	iface meta.Symbol
	sig   meta.MethodSig
}

type methodID struct {
	owner meta.Symbol
	slot  int
}

type methodRef struct {
	owner    meta.Symbol
	slot     int
	slotInfo meta.MethodSlot
}

type pass struct {
	info *meta.GlobalSummary

	methodImplKeys    map[methodID][]ifaceMethodKey
	methodRefs        map[meta.MethodSig][]meta.Symbol // sig → []iface (built eagerly)
	ifaceMethodCounts map[meta.Symbol]int              // iface → unique method name count
	reachable         map[meta.Symbol]struct{}
	usedInIface       map[meta.Symbol]struct{}
	processedIfaceTy  map[meta.Symbol]struct{}
	workQueue         []meta.Symbol

	ifaceMethod        map[ifaceMethodKey]struct{}
	genericIfaceMethod map[meta.Name]struct{}
	reflectSeen        bool

	markableMethods []methodRef
	liveSlots       map[meta.Symbol][]int
}

// Plan is the link-specific semantic result consumed by a backend rewrite.
// The package metadata remains analyzer-independent; this structure is the
// boundary between whole-program planning and LLVM module transformation.
type Plan struct {
	LiveSlots map[string][]int
}

// BuildPlan computes the conservative Go method liveness plan for one link.
// rootNames are final linker-visible roots, not package-local source names.
func BuildPlan(info *meta.GlobalSummary, rootNames []string) Plan {
	return Plan{LiveSlots: analyze(info, rootNames)}
}

// Analyze returns live ABI method slot indexes by concrete type symbol name.
func Analyze(info *meta.GlobalSummary, rootNames []string) map[string][]int {
	return BuildPlan(info, rootNames).LiveSlots
}

func analyze(info *meta.GlobalSummary, rootNames []string) map[string][]int {
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
		methodRefs:         make(map[meta.MethodSig][]meta.Symbol),
		ifaceMethodCounts:  make(map[meta.Symbol]int),
		reachable:          make(map[meta.Symbol]struct{}),
		usedInIface:        make(map[meta.Symbol]struct{}),
		processedIfaceTy:   make(map[meta.Symbol]struct{}),
		ifaceMethod:        make(map[ifaceMethodKey]struct{}),
		genericIfaceMethod: make(map[meta.Name]struct{}),
		liveSlots:          make(map[meta.Symbol][]int),
	}
	d.buildMethodRefs()

	// Seed the initial reachability flood with entry-point roots.
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
	for _, iface := range d.info.Ifaces() {
		seenNames := make(map[meta.Name]struct{})
		for _, sig := range d.info.IfaceMethods(iface) {
			d.methodRefs[sig] = append(d.methodRefs[sig], iface)
			if _, ok := seenNames[sig.Name]; ok {
				continue
			}
			seenNames[sig.Name] = struct{}{}
			d.ifaceMethodCounts[iface]++
		}
	}
}

// computeMethodImplKeys lazily builds methodImplKeys for a single concrete type
// that has entered usedInIface. Called at most once per type.
func (d *pass) computeMethodImplKeys(typ meta.Symbol, slots []meta.MethodSlot) {
	// Compute all slots at once.
	impls := make(map[meta.Symbol]int)

	// For example, if typ has Read and Write slots, Read matches Reader and
	// ReaderWriter while Write matches Writer and ReaderWriter. The resulting
	// counts are Reader=1, Writer=1, and ReaderWriter=2.
	for _, slot := range slots {
		sig := meta.MethodSig{Name: slot.Name, MType: slot.MType}
		for _, iface := range d.methodRefs[sig] {
			impls[iface]++
		}
	}

	// Record keys only for interfaces that typ implements completely. For the
	// example above, the resulting methodImplKeys are:
	//	Read slot:  Reader, ReaderWriter
	//	Write slot: Writer, ReaderWriter
	for slotIndex, slot := range slots {
		id := methodID{owner: typ, slot: slotIndex}
		sig := meta.MethodSig{Name: slot.Name, MType: slot.MType}
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
		if !d.info.HasFacts(sym) {
			continue
		}

		for _, dst := range d.info.OrdinaryEdges(sym) {
			d.markReachable(dst)
		}

		for _, demand := range d.info.FuncDemands(sym) {
			switch demand.Kind {
			case meta.DemandReflectMethod:
				d.reflectSeen = true
			case meta.DemandUseIface:
				d.markUsedInIface(demand.Target)
			case meta.DemandIfaceMethod:
				key := ifaceMethodKey{iface: demand.Target, sig: demand.Sig}
				d.ifaceMethod[key] = struct{}{}
			case meta.DemandNamedMethod:
				d.genericIfaceMethod[demand.MethodName] = struct{}{}
			}
		}

		if _, used := d.usedInIface[sym]; used {
			if _, processed := d.processedIfaceTy[sym]; !processed {
				d.processedIfaceTy[sym] = struct{}{}
				slots := d.info.MethodSlots(sym)
				// Build interface implementation links lazily: only types that
				// enter an interface need method matching.
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
			d.markMethod(method)
			changed = true
			continue
		}
		rem = append(rem, method)
	}

	d.markableMethods = rem
	return changed
}

// shouldKeep reports whether a concrete method slot is demanded. Reflection
// and MethodByName demands are conservative shortcuts handled first.
//
// For ordinary interface calls, method liveness is the intersection of three
// sets:
//  1. Concrete types in the interface domain. A type enters this set when a
//     reachable conversion stores it in an interface, or when TypeChildren
//     propagate that status from another such type. Only methods of these
//     types are added to markableMethods.
//  2. Reachable interface-method demands, such as ReaderWriter.Read. These are
//     recorded in ifaceMethod while flooding reachable functions.
//  3. Complete interface implementation relationships. methodImplKeys records
//     that the concrete owner implements the whole interface and that this
//     slot matches the demanded interface method by name and method type.
//
// The loop below intersects sets 2 and 3; membership in set 1 is guaranteed by
// the construction of markableMethods.
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

func (d *pass) markMethod(method methodRef) {
	d.liveSlots[method.owner] = append(d.liveSlots[method.owner], method.slot)

	// The method descriptor is itself a type descriptor, and reflection can
	// reach its parameter and result types through operations such as
	// reflect.Type.Method(i).Type.Out(j). Traverse those child types with
	// UsedInIface set; otherwise a reflected result such as U from T.Make can
	// be reachable without U's methods participating in a later MethodByName.
	d.markUsedInIface(method.slotInfo.MType)
	d.markReachable(method.slotInfo.MType)
	d.markReachable(method.slotInfo.IFn)
	d.markReachable(method.slotInfo.TFn)
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

func (d *pass) popWork() meta.Symbol {
	sym := d.workQueue[0]
	copy(d.workQueue, d.workQueue[1:])
	d.workQueue = d.workQueue[:len(d.workQueue)-1]
	return sym
}
