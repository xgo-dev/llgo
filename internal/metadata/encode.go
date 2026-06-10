package metadata

import (
	"encoding/binary"
	"io"
)

const (
	magicV1  = "LLPS"
	version1 = 1
)

// WriteMeta serializes PackageMeta to the LLPS v1 binary format.
func (pm *PackageMeta) WriteMeta(w io.Writer) error {
	buf := make([]byte, 0, 4096)

	// header: magic + version
	buf = append(buf, magicV1...)
	buf = binary.AppendUvarint(buf, version1)

	// stringTable
	buf = binary.AppendUvarint(buf, uint64(len(pm.stringTable)))
	for _, s := range pm.stringTable {
		buf = binary.AppendUvarint(buf, uint64(len(s)))
		buf = append(buf, s...)
	}

	// OrdinaryEdges
	writeOrdinaryEdges(&buf, pm.OrdinaryEdges)

	// TypeChildren
	writeTypeChildren(&buf, pm.TypeChildren)

	// InterfaceInfo
	writeInterfaceInfo(&buf, pm.InterfaceInfo)

	// UseIface
	writeUseIface(&buf, pm.UseIface)

	// UseIfaceMethod
	writeUseIfaceMethod(&buf, pm.UseIfaceMethod)

	// MethodInfo
	writeMethodInfo(&buf, pm.MethodInfo)

	// UseNamedMethod
	writeUseNamedMethod(&buf, pm.UseNamedMethod)

	// ReflectMethod
	writeReflectMethod(&buf, pm.ReflectMethod)

	_, err := w.Write(buf)
	return err
}

func writeOrdinaryEdges(buf *[]byte, m map[Symbol][]Symbol) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for src, dsts := range m {
		*buf = binary.AppendUvarint(*buf, uint64(src))
		*buf = binary.AppendUvarint(*buf, uint64(len(dsts)))
		for _, dst := range dsts {
			*buf = binary.AppendUvarint(*buf, uint64(dst))
		}
	}
}

func writeTypeChildren(buf *[]byte, m map[Symbol][]Symbol) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for parent, children := range m {
		*buf = binary.AppendUvarint(*buf, uint64(parent))
		*buf = binary.AppendUvarint(*buf, uint64(len(children)))
		for _, child := range children {
			*buf = binary.AppendUvarint(*buf, uint64(child))
		}
	}
}

func writeInterfaceInfo(buf *[]byte, m map[Symbol][]MethodSig) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for iface, methods := range m {
		*buf = binary.AppendUvarint(*buf, uint64(iface))
		*buf = binary.AppendUvarint(*buf, uint64(len(methods)))
		for _, ms := range methods {
			writeMethodSig(buf, ms)
		}
	}
}

func writeMethodSig(buf *[]byte, ms MethodSig) {
	*buf = binary.AppendUvarint(*buf, uint64(ms.Name))
	*buf = binary.AppendUvarint(*buf, uint64(ms.MType))
}

func writeUseIface(buf *[]byte, m map[Symbol][]Symbol) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for owner, types := range m {
		*buf = binary.AppendUvarint(*buf, uint64(owner))
		*buf = binary.AppendUvarint(*buf, uint64(len(types)))
		for _, t := range types {
			*buf = binary.AppendUvarint(*buf, uint64(t))
		}
	}
}

func writeUseIfaceMethod(buf *[]byte, m map[Symbol][]IfaceMethodDemand) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for owner, demands := range m {
		*buf = binary.AppendUvarint(*buf, uint64(owner))
		*buf = binary.AppendUvarint(*buf, uint64(len(demands)))
		for _, d := range demands {
			*buf = binary.AppendUvarint(*buf, uint64(d.Target))
			writeMethodSig(buf, d.Sig)
		}
	}
}

func writeMethodInfo(buf *[]byte, m map[Symbol][]MethodSlot) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for typ, slots := range m {
		*buf = binary.AppendUvarint(*buf, uint64(typ))
		*buf = binary.AppendUvarint(*buf, uint64(len(slots)))
		for _, slot := range slots {
			writeMethodSig(buf, slot.Sig)
			*buf = binary.AppendUvarint(*buf, uint64(slot.IFn))
			*buf = binary.AppendUvarint(*buf, uint64(slot.TFn))
		}
	}
}

func writeUseNamedMethod(buf *[]byte, m map[Symbol][]Symbol) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for owner, names := range m {
		*buf = binary.AppendUvarint(*buf, uint64(owner))
		*buf = binary.AppendUvarint(*buf, uint64(len(names)))
		for _, name := range names {
			*buf = binary.AppendUvarint(*buf, uint64(name))
		}
	}
}

func writeReflectMethod(buf *[]byte, m map[Symbol]struct{}) {
	*buf = binary.AppendUvarint(*buf, uint64(len(m)))
	for owner := range m {
		*buf = binary.AppendUvarint(*buf, uint64(owner))
	}
}
