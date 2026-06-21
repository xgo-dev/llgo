package metadata

import (
	"encoding/binary"
	"io"
	"sort"
)

const (
	magicV1  = "LLPS"
	version1 = 1
)

// WriteMeta serializes PackageMeta to the LLPS v1 binary format.
func (pm *PackageMeta) WriteMeta(w io.Writer) error {
	buf := make([]byte, 0, 4096)

	buf = append(buf, magicV1...)
	buf = binary.AppendUvarint(buf, version1)

	buf = binary.AppendUvarint(buf, uint64(len(pm.stringTable)))
	for _, s := range pm.stringTable {
		buf = binary.AppendUvarint(buf, uint64(len(s)))
		buf = append(buf, s...)
	}

	writeSymbolMap(&buf, pm.ordinaryEdges)
	writeSymbolMap(&buf, pm.typeChildren)
	writeInterfaceInfo(&buf, pm.interfaceInfo)
	writeSymbolMap(&buf, pm.useIface)
	writeUseIfaceMethod(&buf, pm.useIfaceMethod)
	writeMethodInfo(&buf, pm.methodInfo)
	writeUseNamedMethod(&buf, pm.useNamedMethod)
	writeReflectMethod(&buf, pm.reflectMethod)

	_, err := w.Write(buf)
	return err
}

func writeSymbolMap(buf *[]byte, m map[Symbol][]Symbol) {
	keys := sortedSymbolKeys(m)
	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, key := range keys {
		values := m[key]
		*buf = binary.AppendUvarint(*buf, uint64(key))
		*buf = binary.AppendUvarint(*buf, uint64(len(values)))
		for _, value := range values {
			*buf = binary.AppendUvarint(*buf, uint64(value))
		}
	}
}

func writeInterfaceInfo(buf *[]byte, m map[Symbol][]MethodSig) {
	keys := sortedSymbolKeys(m)
	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, iface := range keys {
		methods := m[iface]
		*buf = binary.AppendUvarint(*buf, uint64(iface))
		*buf = binary.AppendUvarint(*buf, uint64(len(methods)))
		for _, method := range methods {
			writeMethodSig(buf, method)
		}
	}
}

func writeMethodSig(buf *[]byte, sig MethodSig) {
	*buf = binary.AppendUvarint(*buf, uint64(sig.Name))
	*buf = binary.AppendUvarint(*buf, uint64(sig.MType))
}

func writeUseIfaceMethod(buf *[]byte, m map[Symbol][]IfaceMethodDemand) {
	keys := sortedSymbolKeys(m)
	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, owner := range keys {
		demands := m[owner]
		*buf = binary.AppendUvarint(*buf, uint64(owner))
		*buf = binary.AppendUvarint(*buf, uint64(len(demands)))
		for _, demand := range demands {
			*buf = binary.AppendUvarint(*buf, uint64(demand.Target))
			writeMethodSig(buf, demand.Sig)
		}
	}
}

func writeMethodInfo(buf *[]byte, m map[Symbol][]MethodSlot) {
	keys := sortedSymbolKeys(m)
	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, typ := range keys {
		slots := m[typ]
		*buf = binary.AppendUvarint(*buf, uint64(typ))
		*buf = binary.AppendUvarint(*buf, uint64(len(slots)))
		for _, slot := range slots {
			writeMethodSig(buf, slot.Sig)
			*buf = binary.AppendUvarint(*buf, uint64(slot.IFn))
			*buf = binary.AppendUvarint(*buf, uint64(slot.TFn))
		}
	}
}

func writeUseNamedMethod(buf *[]byte, m map[Symbol][]Name) {
	keys := sortedSymbolKeys(m)
	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, owner := range keys {
		names := m[owner]
		*buf = binary.AppendUvarint(*buf, uint64(owner))
		*buf = binary.AppendUvarint(*buf, uint64(len(names)))
		for _, name := range names {
			*buf = binary.AppendUvarint(*buf, uint64(name))
		}
	}
}

func writeReflectMethod(buf *[]byte, m map[Symbol]struct{}) {
	keys := make([]Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	*buf = binary.AppendUvarint(*buf, uint64(len(keys)))
	for _, owner := range keys {
		*buf = binary.AppendUvarint(*buf, uint64(owner))
	}
}

type symbolValueMap[V any] interface {
	~map[Symbol]V
}

func sortedSymbolKeys[M symbolValueMap[V], V any](m M) []Symbol {
	keys := make([]Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
