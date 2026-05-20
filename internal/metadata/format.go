package metadata

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// FormatMeta writes a stable human-readable representation for tests.
func FormatMeta(w io.Writer, pm *PackageMeta) {
	if pm == nil {
		return
	}

	sym := func(s Symbol) string {
		if int(s) < len(pm.stringTable) {
			return pm.stringTable[s]
		}
		return fmt.Sprintf("?%d", s)
	}
	name := func(n Name) string {
		if int(n) < len(pm.stringTable) {
			return pm.stringTable[n]
		}
		return fmt.Sprintf("?%d", n)
	}
	keys := func(m map[Symbol][]Symbol) []Symbol { return sortedFormatKeys(m, sym) }

	if len(pm.typeChildren) > 0 {
		fmt.Fprintln(w, "[TypeChildren]")
		for _, key := range keys(pm.typeChildren) {
			fmt.Fprintf(w, "%s:\n", sym(key))
			for _, child := range pm.typeChildren[key] {
				fmt.Fprintf(w, "    %s\n", sym(child))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.interfaceInfo) > 0 {
		fmt.Fprintln(w, "[InterfaceInfo]")
		for _, iface := range sortedFormatKeys(pm.interfaceInfo, sym) {
			fmt.Fprintf(w, "%s:\n", sym(iface))
			for _, method := range pm.interfaceInfo[iface] {
				fmt.Fprintf(w, "    %s %s\n", name(method.Name), sym(method.MType))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.ordinaryEdges) > 0 {
		fmt.Fprintln(w, "[OrdinaryEdges]")
		for _, key := range keys(pm.ordinaryEdges) {
			fmt.Fprintf(w, "%s:\n", sym(key))
			for _, dst := range pm.ordinaryEdges[key] {
				fmt.Fprintf(w, "    %s\n", sym(dst))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.useIface) > 0 {
		fmt.Fprintln(w, "[UseIface]")
		for _, key := range keys(pm.useIface) {
			fmt.Fprintf(w, "%s:\n", sym(key))
			for _, typ := range pm.useIface[key] {
				fmt.Fprintf(w, "    %s\n", sym(typ))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.useIfaceMethod) > 0 {
		fmt.Fprintln(w, "[UseIfaceMethod]")
		for _, owner := range sortedFormatKeys(pm.useIfaceMethod, sym) {
			fmt.Fprintf(w, "%s:\n", sym(owner))
			for _, demand := range pm.useIfaceMethod[owner] {
				fmt.Fprintf(w, "    %s %s %s\n", sym(demand.Target), name(demand.Sig.Name), sym(demand.Sig.MType))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.methodInfo) > 0 {
		fmt.Fprintln(w, "[MethodInfo]")
		for _, typ := range sortedFormatKeys(pm.methodInfo, sym) {
			fmt.Fprintf(w, "%s:\n", sym(typ))
			for i, slot := range pm.methodInfo[typ] {
				fmt.Fprintf(w, "    %d %s %s %s %s\n", i, name(slot.Sig.Name), sym(slot.Sig.MType), sym(slot.IFn), sym(slot.TFn))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.useNamedMethod) > 0 {
		fmt.Fprintln(w, "[UseNamedMethod]")
		for _, owner := range sortedFormatKeys(pm.useNamedMethod, sym) {
			fmt.Fprintf(w, "%s:\n", sym(owner))
			for _, methodName := range pm.useNamedMethod[owner] {
				fmt.Fprintf(w, "    %s\n", name(methodName))
			}
		}
		fmt.Fprintln(w)
	}

	if len(pm.reflectMethod) > 0 {
		owners := sortedFormatSetKeys(pm.reflectMethod, sym)

		fmt.Fprintln(w, "[ReflectMethod]")
		for _, owner := range owners {
			fmt.Fprintln(w, sym(owner))
		}
		fmt.Fprintln(w)
	}
}

// MetaString returns the formatted metadata string.
func MetaString(pm *PackageMeta) string {
	var sb strings.Builder
	FormatMeta(&sb, pm)
	return sb.String()
}

func sortedFormatKeys[V any](m map[Symbol]V, name func(Symbol) string) []Symbol {
	keys := make([]Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return name(keys[i]) < name(keys[j]) })
	return keys
}

func sortedFormatSetKeys(m map[Symbol]struct{}, name func(Symbol) string) []Symbol {
	keys := make([]Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return name(keys[i]) < name(keys[j]) })
	return keys
}
