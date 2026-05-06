/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cltest

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/goplus/llgo/internal/semmeta"
)

// FormatSemMeta renders semantic metadata golden text in the stable
// grouped-by-key form used by meta-expect.txt fixtures.
func FormatSemMeta(info semmeta.ModuleInfo) string {
	var buf bytes.Buffer
	writeSymbolListSection(&buf, "UseIface", info.UseIface)
	writeIfaceMethodSection(&buf, "UseIfaceMethod", info.UseIfaceMethod)
	writeInterfaceInfoSection(&buf, "InterfaceInfo", info.InterfaceInfo)
	writeMethodInfoSection(&buf, "MethodInfo", info.MethodInfo)
	writeStringListSection(&buf, "UseNamedMethod", info.UseNamedMethod)
	writeReflectMethodSection(&buf, "ReflectMethod", info.ReflectMethod)
	return buf.String()
}

func writeSectionHeader(buf *bytes.Buffer, name string) {
	if buf.Len() != 0 {
		buf.WriteByte('\n')
	}
	fmt.Fprintf(buf, "[%s]\n", name)
}

func writeSymbolListSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol][]semmeta.Symbol) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSymbolKeys(data) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, value := range sortedSymbols(data[key]) {
			fmt.Fprintf(buf, "    %s\n", value)
		}
	}
}

func writeIfaceMethodSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol][]semmeta.IfaceMethodUse) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSymbolKeys(data) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, value := range sortedIfaceMethodDemands(data[key]) {
			fmt.Fprintf(buf, "    %s %s %s\n", value.Target, value.Sig.Name, value.Sig.MType)
		}
	}
}

func writeInterfaceInfoSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol][]semmeta.MethodSig) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSymbolKeys(data) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, value := range sortedMethodSigs(data[key]) {
			fmt.Fprintf(buf, "    %s %s\n", value.Name, value.MType)
		}
	}
}

func writeMethodInfoSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol][]semmeta.MethodSlot) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSymbolKeys(data) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, value := range data[key] {
			fmt.Fprintf(buf, "    %d %s %s %s %s\n", value.Index, value.Sig.Name, value.Sig.MType, value.IFn, value.TFn)
		}
	}
}

func writeStringListSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol][]string) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSymbolKeys(data) {
		fmt.Fprintf(buf, "%s:\n", key)
		for _, value := range sortedStrings(data[key]) {
			fmt.Fprintf(buf, "    %s\n", value)
		}
	}
}

func writeReflectMethodSection(buf *bytes.Buffer, name string, data map[semmeta.Symbol]struct{}) {
	if len(data) == 0 {
		return
	}
	writeSectionHeader(buf, name)
	for _, key := range sortedSetKeys(data) {
		fmt.Fprintf(buf, "%s\n", key)
	}
}

func sortedSymbolKeys[T any](m map[semmeta.Symbol]T) []semmeta.Symbol {
	keys := make([]semmeta.Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func sortedSetKeys(m map[semmeta.Symbol]struct{}) []semmeta.Symbol {
	return sortedSymbolKeys(m)
}

func sortedSymbols(values []semmeta.Symbol) []semmeta.Symbol {
	out := append([]semmeta.Symbol(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedIfaceMethodDemands(values []semmeta.IfaceMethodUse) []semmeta.IfaceMethodUse {
	out := append([]semmeta.IfaceMethodUse(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Target != b.Target {
			return a.Target < b.Target
		}
		if a.Sig.Name != b.Sig.Name {
			return a.Sig.Name < b.Sig.Name
		}
		return a.Sig.MType < b.Sig.MType
	})
	return out
}

func sortedMethodSigs(values []semmeta.MethodSig) []semmeta.MethodSig {
	out := append([]semmeta.MethodSig(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.MType < b.MType
	})
	return out
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
