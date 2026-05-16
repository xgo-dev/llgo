package metadata

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// FormatMeta writes a human-readable text representation of pm to w.
// The output is intended for golden-file comparison in tests.
func FormatMeta(w io.Writer, pm *PackageMeta) {
	if pm == nil {
		return
	}
	table := pm.stringTable
	str := func(s Symbol) string {
		if int(s) < len(table) {
			return table[s]
		}
		return fmt.Sprintf("?%d", s)
	}

	sortSymbolKeys := func(m map[Symbol][]Symbol) []Symbol {
		keys := make([]Symbol, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return str(keys[i]) < str(keys[j]) })
		return keys
	}

	// TypeChildren
	if len(pm.TypeChildren) > 0 {
		fmt.Fprintln(w, "[TypeChildren]")
		for _, k := range sortSymbolKeys(pm.TypeChildren) {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.TypeChildren[k] {
				fmt.Fprintf(w, "    %s\n", str(v))
			}
		}
		fmt.Fprintln(w)
	}

	// InterfaceInfo
	if len(pm.InterfaceInfo) > 0 {
		fmt.Fprintln(w, "[InterfaceInfo]")
		ifaceKeys := make([]Symbol, 0, len(pm.InterfaceInfo))
		for k := range pm.InterfaceInfo {
			ifaceKeys = append(ifaceKeys, k)
		}
		sort.Slice(ifaceKeys, func(i, j int) bool { return str(ifaceKeys[i]) < str(ifaceKeys[j]) })
		for _, k := range ifaceKeys {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.InterfaceInfo[k] {
				fmt.Fprintf(w, "    %s %s\n", str(v.Name), str(v.MType))
			}
		}
		fmt.Fprintln(w)
	}

	// UseIface
	if len(pm.UseIface) > 0 {
		fmt.Fprintln(w, "[UseIface]")
		for _, k := range sortSymbolKeys(pm.UseIface) {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.UseIface[k] {
				fmt.Fprintf(w, "    %s\n", str(v))
			}
		}
		fmt.Fprintln(w)
	}

	// UseIfaceMethod
	if len(pm.UseIfaceMethod) > 0 {
		fmt.Fprintln(w, "[UseIfaceMethod]")
		ownerKeys := make([]Symbol, 0, len(pm.UseIfaceMethod))
		for k := range pm.UseIfaceMethod {
			ownerKeys = append(ownerKeys, k)
		}
		sort.Slice(ownerKeys, func(i, j int) bool { return str(ownerKeys[i]) < str(ownerKeys[j]) })
		for _, k := range ownerKeys {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.UseIfaceMethod[k] {
				fmt.Fprintf(w, "    %s %s %s\n", str(v.Target), str(v.Sig.Name), str(v.Sig.MType))
			}
		}
		fmt.Fprintln(w)
	}

	// MethodInfo
	if len(pm.MethodInfo) > 0 {
		fmt.Fprintln(w, "[MethodInfo]")
		typeKeys := make([]Symbol, 0, len(pm.MethodInfo))
		for k := range pm.MethodInfo {
			typeKeys = append(typeKeys, k)
		}
		sort.Slice(typeKeys, func(i, j int) bool { return str(typeKeys[i]) < str(typeKeys[j]) })
		for _, k := range typeKeys {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.MethodInfo[k] {
				fmt.Fprintf(w, "    %s %s %s %s\n", str(v.Sig.Name), str(v.Sig.MType), str(v.IFn), str(v.TFn))
			}
		}
		fmt.Fprintln(w)
	}

	// UseNamedMethod
	if len(pm.UseNamedMethod) > 0 {
		fmt.Fprintln(w, "[UseNamedMethod]")
		for _, k := range sortSymbolKeys(pm.UseNamedMethod) {
			fmt.Fprintf(w, "%s:\n", str(k))
			for _, v := range pm.UseNamedMethod[k] {
				fmt.Fprintf(w, "    %s\n", str(v))
			}
		}
		fmt.Fprintln(w)
	}

	// ReflectMethod
	if len(pm.ReflectMethod) > 0 {
		fmt.Fprintln(w, "[ReflectMethod]")
		keys := make([]Symbol, 0, len(pm.ReflectMethod))
		for k := range pm.ReflectMethod {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return str(keys[i]) < str(keys[j]) })
		for _, k := range keys {
			fmt.Fprintln(w, str(k))
		}
		fmt.Fprintln(w)
	}
}

// MetaString returns a formatted string representation of pm.
func MetaString(pm *PackageMeta) string {
	var sb strings.Builder
	FormatMeta(&sb, pm)
	return sb.String()
}
