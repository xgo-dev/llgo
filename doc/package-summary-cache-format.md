# LLGo Package Summary Cache Format

## 概述

本文档定义 LLGo 包级摘要缓存（Package Summary Cache）的二进制文件格式。该格式服务于全图方法可达性分析（Whole-Program DCE），其核心设计目标是：

- **零反序列化**：文件通过 mmap 映射到内存，字节直接作为内存对象访问
- **O(1) 查询**：所有 per-symbol 数据通过 CSR 布局实现直接寻址
- **极低合并开销**：GlobalSummary 合并阶段只 intern 字符串表，不遍历重写所有边

设计参考了 Go 官方 linker（`cmd/internal/goobj`）的 `RelocIndex + Reloc` CSR 布局，并结合 LLGo 的实际需求做了简化。

---

## 文件布局

```
┌──────────────────────────────────────────────────────┐
│ Header                                               │
├──────────────────────────────────────────────────────┤
│ stringTable                                          │
├──────────────────────────────────────────────────────┤
│ Symbols                                              │
├──────────────────────────────────────────────────────┤
│ Edges          (CSR)                                 │
├──────────────────────────────────────────────────────┤
│ TypeChildren   (CSR)                                 │
├──────────────────────────────────────────────────────┤
│ MethodInfo     (CSR)                                 │
├──────────────────────────────────────────────────────┤
│ InterfaceInfo  (CSR)                                 │
├──────────────────────────────────────────────────────┤
│ ReflectBitmap                                        │
└──────────────────────────────────────────────────────┘
```

所有整数均为 **little-endian**。所有 section 4 字节对齐。

---

## Header

```
Header {
    Magic      [4]byte   // "LLPM"
    Version    uint32    // 当前版本 = 1
    SectionOffsets [8]uint32  // 各 section 在文件中的起始字节位置
                              // [0]=stringTable [1]=Symbols [2]=Edges
                              // [3]=TypeChildren [4]=MethodInfo
                              // [5]=InterfaceInfo [6]=ReflectBitmap [7]=reserved
}
```

Header 总大小固定 = 4 + 4 + 8×4 = **40 字节**。

`SectionOffsets` 让读取方能直接跳到任意 section，无需顺序扫描。

---

## stringTable section

```
stringTable {
    data []byte   // 所有字符串连续拼接的原始字节流
}
```

字符串不做任何结构化：直接把所有符号名拼接在一起。
`Symbols` section 里的每条 record 通过 `(NameOff, NameLen)` 引用对应字节区间。

---

## Symbols section

```
Symbols {
    NSyms   uint32
    Records [NSyms]SymbolRecord
}

SymbolRecord {        // 12 字节定长
    NameOff uint32    // 符号名在 stringTable 中的起始偏移
    NameLen uint32    // 符号名长度（字节）
    _       [4]byte   // 保留，对齐到 12 字节
}
```

**LocalSymbol = Records 数组的下标（uint32）**，在所有其他 section 中统一使用。

### 设计要点

- **内外部符号不做区分**：无论是本包定义的还是外部包引用的符号，都在此表中分配 LocalSymbol，写法完全一致。
- **不存 flag/属性**：符号的语义属性（是否是类型、是否是接口）通过它在哪些 section 中出现来隐式表达（见「符号语义识别」章节）。
- **幂等注册**：Builder 阶段同一个名字多次调用 `Symbol()` 返回相同的 LocalSymbol。

---

## CSR 布局（通用）

Edges、TypeChildren、MethodInfo、InterfaceInfo 四个 section 均采用 CSR（Compressed Sparse Row）布局：

```
{SectionName} {
    NSyms   uint32               // 符号数量（= Symbols.NSyms）
    Offsets [NSyms+1]uint32      // CSR offsets 数组，Offsets[NSyms] 为尾哨兵
    Data    [...]T               // 定长 record 的连续数组，T 因 section 而异
}
```

**查询 symbol i 的数据**：

```
start = Offsets[i]
end   = Offsets[i+1]
result = Data[start : end]   // 零拷贝切片
```

若 `start == end`，该符号在此 section 中无数据（正常情况，不代表错误）。

每个符号在 Offsets 数组中有且仅有一个 entry，**包括没有数据的符号**（它们的相邻 offset 相等）。这是 Go linker 的标准做法，每符号成本为 4 字节（一个 uint32），简单且查询 O(1)。

---

## Edges section

```
Edges {
    NSyms   uint32
    Offsets [NSyms+1]uint32
    Data    []Edge
}

Edge {              // 12 字节定长
    Target  uint32  // LocalSymbol（EdgeOrdinary/UseIface/UseIfaceMethod）
                    // 或 Name 引用（UseNamedMethod，此时为 stringTable offset）
    Extra   uint32  // 仅 UseIfaceMethod 使用：目标接口的方法 index
                    // 其他 Kind 为 0
    Kind    uint8   // 边的种类（见下表）
    _       [3]byte // padding
}
```

### Kind 定义

| Kind | 值 | 含义 | Target | Extra |
|------|---|------|--------|-------|
| `EdgeOrdinary`       | 0 | 普通符号引用 | 目标 LocalSymbol | 0 |
| `EdgeUseIface`       | 1 | 该函数将 Target 类型转为接口 | 类型 LocalSymbol | 0 |
| `EdgeUseIfaceMethod` | 2 | 该函数调用了 Target 接口的某个方法 | 接口 LocalSymbol | 方法 index |
| `EdgeUseNamedMethod` | 3 | 该函数按名调用方法（MethodByName 常量） | stringTable offset | 0 |

### 说明

- `EdgeOrdinary` 涵盖所有普通符号引用：函数调用、全局变量引用、类型描述符引用等。
- OrdinaryEdges 来自两个来源：cl/ssa 编译阶段（类型/接口相关事实）和 build 层扫描 LLVM Module 指令（普通调用/引用关系）。
- 对应 Go linker 中 `R_CALL`、`R_ADDR` 等普通 reloc 类型（EdgeOrdinary），以及 `R_USEIFACE`、`R_USEIFACEMETHOD`、`R_USENAMEDMETHOD` marker reloc（其余三种 Kind）。

---

## TypeChildren section

```
TypeChildren {
    NSyms   uint32
    Offsets [NSyms+1]uint32
    Data    []uint32            // 子类型 LocalSymbol 的连续数组
}
```

记录类型描述符中包含的子类型引用：
- struct 类型 → 各字段的类型
- 指针类型 → 指向的元素类型
- slice/array 类型 → 元素类型
- map 类型 → key 和 value 类型
- chan 类型 → 元素类型

### 双重用途

1. **子类型传播**：当父类型被标记为 `usedInIface` 时，沿 TypeChildren 把 `usedInIface` 传播给所有子类型。
2. **隐式类型识别**：`TypeChildren[sym]` 非空，则 sym 是 composite type。**不需要额外 flag**。

对于没有子类型的 named primitive type（如 `type Foo int`），TypeChildren 为空，`usedInIface` 只能通过 `EdgeUseIface` 直接设置，无需通过父类型传播——这是正确的语义。

---

## MethodInfo section

```
MethodInfo {
    NSyms   uint32
    Offsets [NSyms+1]uint32
    Data    []MethodSlot
}

MethodSlot {        // 16 字节定长
    NameRef uint32  // 方法短名（stringTable offset）
    MType   uint32  // 方法函数类型符号（LocalSymbol）
    IFn     uint32  // 接口调用入口符号（LocalSymbol）
    TFn     uint32  // 类型方法入口符号（LocalSymbol）
}
```

- 槽位写入顺序与 Go runtime 中该类型的 `abi.Method` 表顺序**严格一致**，槽位 index 即 `abi.Method` 表中的位置。
- **MethodInfo 中有 entry 的符号即 ConcreteType**（有方法的具体类型），不需要额外 flag。

---

## InterfaceInfo section

```
InterfaceInfo {
    NSyms   uint32
    Offsets [NSyms+1]uint32
    Data    []MethodSig
}

MethodSig {         // 8 字节定长
    NameRef uint32  // 方法短名（stringTable offset）
    MType   uint32  // 方法函数类型符号（LocalSymbol）
}
```

- **InterfaceInfo 中有 entry 的符号即 Interface type**，不需要额外 flag。
- `EdgeUseIfaceMethod` 中的 `Extra` 字段（方法 index）直接索引此处对应接口的 Data 切片。

---

## ReflectBitmap section

```
ReflectBitmap {
    NSyms  uint32
    Bitmap [(NSyms+7)/8]uint8
}
```

bit `i` 为 1 表示 symbol `i` 触发了无法静态确定方法名的反射调用（`reflect.Type.Method(index)`、非常量 `MethodByName` 等）。

查询：

```go
func hasReflectMethod(sym LocalSymbol) bool {
    return bitmap[sym/8] & (1 << (sym%8)) != 0
}
```

---

## 符号语义识别（无 flag 设计）

DCE 阶段所有符号类型判断**通过 section 存在性隐式表达**，不依赖任何 flag 字段：

| 判断 | 实现 |
|------|------|
| sym 是 composite type | `TypeChildren.Offsets[i] != TypeChildren.Offsets[i+1]` |
| sym 是 concrete type（有方法） | `MethodInfo.Offsets[i] != MethodInfo.Offsets[i+1]` |
| sym 是 interface type | `InterfaceInfo.Offsets[i] != InterfaceInfo.Offsets[i+1]` |
| sym 使用了反射 | `ReflectBitmap bit i == 1` |

这个设计对齐了当前 MVP `analyze.go` 中 `TypeChildren`、`MethodSlots`、`InterfaceMethods` 的使用方式，无需引入额外的属性机制。

---

## mmap 读取

```go
type PackageMeta struct {
    raw  []byte   // mmap 映射的字节区域，或 Builder.Build() 产出的字节缓冲

    // 解析 Header 后缓存的各 section 起始偏移（构造时一次性计算）
    strOff      uint32
    symOff      uint32
    edgesOff    uint32
    childrenOff uint32
    methodOff   uint32
    ifaceOff    uint32
    reflectOff  uint32

    nsyms uint32   // 符号数量，各 section 共享
}

// 从文件 mmap
func ReadMeta(path string) (*PackageMeta, error) {
    f, _ := os.Open(path)
    raw, _ := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
    pm := &PackageMeta{raw: raw, file: f}
    pm.parseHeader()
    return pm, nil
}

// 从 Builder 直接构造（Cache MISS 路径）
func (b *Builder) Build() (*PackageMeta, error) {
    raw := b.serialize()   // Builder 内部 map → wire format
    pm := &PackageMeta{raw: raw}
    pm.parseHeader()
    return pm, nil
}

// 落盘
func (pm *PackageMeta) Bytes() []byte { return pm.raw }

// 查询示例（零分配，直接切 mmap 区域）
func (pm *PackageMeta) TypeChildren(sym LocalSymbol) []LocalSymbol {
    section := pm.raw[pm.childrenOff:]
    // nsyms := binary.LittleEndian.Uint32(section[:4])
    offsetsBase := section[4:]
    start := binary.LittleEndian.Uint32(offsetsBase[sym*4:])
    end   := binary.LittleEndian.Uint32(offsetsBase[(sym+1)*4:])
    if start == end { return nil }
    dataBase := offsetsBase[(pm.nsyms+1)*4:]
    return unsafe.Slice((*LocalSymbol)(unsafe.Pointer(&dataBase[start*4])), end-start)
}
```

两条路径（Cache HIT / Cache MISS）产出同一个 `*PackageMeta` 类型，后续 GlobalSummary 合并完全透明。

---

## GlobalSummary 合并（Lazy Remap）

合并阶段**只做字符串 intern，不遍历重写边**：

```go
type GlobalSummary struct {
    intern    map[string]Symbol      // 字符串 → 全局 Symbol ID（O(unique_symbols) 建立）
    locToGlb  [][]Symbol             // [pkgIdx][localSym] → global Symbol
    pkgs      []*PackageMeta         // 各包原始数据保留，不拷贝

    interfaces    []Symbol           // InterfaceInfo 有 entry 的全局 Symbol
    concreteTypes []Symbol           // MethodInfo 有 entry 的全局 Symbol
}

func NewGlobalSummary(pkgs []*PackageMeta) *GlobalSummary {
    g := &GlobalSummary{pkgs: pkgs}

    // 第一步：intern 所有包的字符串表，建立 locToGlb 映射
    // 成本 = O(所有包字符串总数) = O(unique_symbols 数量级)
    // 不触碰任何 Edge
    for pkgIdx, pkg := range pkgs {
        g.locToGlb[pkgIdx] = make([]Symbol, pkg.NSyms())
        for localID := range pkg.NSyms() {
            name := pkg.SymbolName(LocalSymbol(localID))
            gid := g.intern.GetOrInsert(name)
            g.locToGlb[pkgIdx][localID] = gid
        }
    }

    // 第二步：收集 interfaces 和 concreteTypes
    // 扫 InterfaceInfo 和 MethodInfo 的 key，remap 到全局 Symbol
    g.buildTypeIndex()
    return g
}

// 查询时按需翻译（lazy），不预先重写
func (g *GlobalSummary) Edges(sym Symbol) iter.Seq[Edge] {
    return func(yield func(Edge) bool) {
        pkgIdx, localSym := g.ownerOf(sym)
        pkg := g.pkgs[pkgIdx]
        for _, edge := range pkg.Edges(localSym) {
            // 翻译 Target 的 LocalSymbol → global Symbol
            globalEdge := Edge{
                Target: g.locToGlb[pkgIdx][edge.Target],
                Extra:  edge.Extra,
                Kind:   edge.Kind,
            }
            if !yield(globalEdge) { return }
        }
    }
}
```

合并成本从 **O(total_edges)** 降为 **O(unique_symbols)**，消除前面测量到的 42ms 全局合并开销。

---

## Cache 集成流程

```
buildPkg(pkg):
    metaPath = cachePath(actionID, ".meta")

    if exists(metaPath):
        ── Cache HIT ──
        aPkg.Meta = ReadMeta(metaPath)    // mmap，~微秒级，无反序列化
    else:
        ── Cache MISS ──
        builder = NewBuilder()
        ret = cl.NewPackageEx(..., builder)      // cl/ssa 填入类型相关事实
        extractOrdinaryEdges(builder, ret.Module) // 扫 IR 补充普通引用边
        aPkg.Meta = builder.Build()
        writeFile(metaPath, aPkg.Meta.Bytes())    // 落盘，下次命中用
        runLLVMPipeline(ret.Module)               // 正常 LLVM 编译流程

    allMetas.append(aPkg.Meta)

全图阶段:
    gs = NewGlobalSummary(allMetas)    // O(unique_symbols) intern
    liveSlots = deadcode.Analyze(gs)   // DCE 算法，~毫秒级
    applyDCE(liveSlots)
```

---

## 版本与兼容性

- `Version` 字段标识格式版本，当前为 1。
- 读到不支持的 version，直接放弃此缓存，回退到重新编译路径。
- 格式 version 作为编译器内嵌常量参与 actionID 计算，version 升级自动 invalidate 旧缓存。

---

## 与 MVP 的对比

| 维度 | MVP | 本格式 |
|------|-----|--------|
| 读取方式 | 逐字节反序列化（uvarint） | mmap 零拷贝 |
| 合并方式 | 遍历全部边重写 global ID（O(edges)，~42ms） | 只 intern 字符串（O(symbols)，~微秒） |
| 反序列化成本 | ~50ms（read + decode） | ~微秒（mmap syscall） |
| 符号属性 | `map[Symbol]struct{}` in memory | 通过 section 存在性隐式表达，零额外存储 |
| TypeChildren | 独立 section + 函数递归展开 | 独立 section + worklist（无 Go 函数递归） |
| 是否类型判断 | `typeSymbols map` | `TypeChildren[i] 非空` |
| ReflectMethod | 独立 section | bitmap（每符号 1 bit） |
| UseIface/UseIfaceMethod/UseNamedMethod | 独立 section | 合并进 Edges，Kind 字段区分 |

---

## 空间估算

以中等规模包（1000 符号、5000 边、200 类型、20 接口）为例：

| Section | 大小 |
|---------|------|
| Header | 40 B |
| stringTable | ~50 KB |
| Symbols | 12 × 1000 = 12 KB |
| Edges offsets | 4 × 1001 = ~4 KB |
| Edges data | 12 × 5000 = 60 KB |
| TypeChildren offsets | 4 × 1001 = ~4 KB |
| TypeChildren data | 4 × ~500 = ~2 KB |
| MethodInfo offsets | 4 × 1001 = ~4 KB |
| MethodInfo data | 16 × 1000 = 16 KB |
| InterfaceInfo offsets | 4 × 1001 = ~4 KB |
| InterfaceInfo data | 8 × 80 = ~1 KB |
| ReflectBitmap | ~125 B |
| **合计** | **~157 KB / 包** |

100 个包约 ~15 MB，mmap 无压力。
