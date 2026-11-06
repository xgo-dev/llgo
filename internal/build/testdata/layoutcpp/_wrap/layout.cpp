#include <cstddef>
#include <cstdint>

struct Empty {};
struct Pad { Empty e; uint8_t b; };
struct NoPad { uint64_t x; [[no_unique_address]] Empty e; };
struct Pair { [[no_unique_address]] Empty a; [[no_unique_address]] Empty b; };
struct __attribute__((packed)) Packed { uint8_t b; uint32_t x; uint8_t z; };
struct __attribute__((packed)) Lane { uint32_t x; uint8_t tail; };
struct PackedArray { Lane lanes[2]; };
#pragma pack(push, 2)
struct Pack2 { uint8_t b; uint32_t x; uint16_t y; };
#pragma pack(pop)

extern "C" int32_t llgo_layout_callback_impl(int32_t n) { return n + 7; }
extern "C" {
int32_t (*llgo_layout_callback)(int32_t) = llgo_layout_callback_impl;
}

extern "C" uint64_t llgo_layout_metric(int32_t i) {
    static const uint64_t values[] = {
        sizeof(Pad), alignof(Pad), offsetof(Pad, b),
        sizeof(NoPad), sizeof(NoPad[2]),
        sizeof(Pair), offsetof(Pair, b),
        sizeof(Packed), alignof(Packed), offsetof(Packed, x), offsetof(Packed, z),
        sizeof(Pack2), alignof(Pack2), offsetof(Pack2, x), offsetof(Pack2, y),
    };
    return values[i];
}
extern "C" Packed llgo_layout_round_packed(Packed p) {
    p.b += 1; p.x += 2; p.z += 3; return p;
}
extern "C" Pack2 llgo_layout_round_pack2(Pack2 p) {
    p.b += 1; p.x += 2; p.y += 3; return p;
}
extern "C" PackedArray llgo_layout_round_array(PackedArray p) {
    p.lanes[0].x += 1; p.lanes[1].x += 2; return p;
}
