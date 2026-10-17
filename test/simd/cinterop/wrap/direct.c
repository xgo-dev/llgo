#include "vector.h"

#if defined(__aarch64__)
#include <arm_neon.h>
#else
#include <xmmintrin.h>
#endif

llgo_f4 llgo_direct_add(llgo_f4 a, llgo_f4 b) {
#if defined(__aarch64__)
    return vaddq_f32(a,b);
#else
    return _mm_add_ps(a,b);
#endif
}
llgo_f4 llgo_direct_callback(llgo_f4 (*fn)(llgo_f4,llgo_f4), llgo_f4 a, llgo_f4 b) { return fn(a,b); }
extern llgo_f4 llgo_test_vector_export(llgo_f4,llgo_f4);
llgo_f4 llgo_direct_export(llgo_f4 a, llgo_f4 b) { return llgo_test_vector_export(a,b); }
float llgo_direct_box_read(const llgo_vbox *p) { return llgo_box_read(p); }
llgo_vbox llgo_direct_box(llgo_vbox p) { return llgo_box_value(p); }
llgo_vpair llgo_direct_pair(llgo_vpair p) { return llgo_pair_swap(p); }
llgo_vpair llgo_direct_pair_callback(llgo_vpair (*fn)(llgo_vpair), llgo_vpair p) { return fn(p); }
