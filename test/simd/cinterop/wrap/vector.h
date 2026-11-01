#ifndef LLGO_TEST_VECTOR_H
#define LLGO_TEST_VECTOR_H
#include <stddef.h>
#include <stdint.h>

typedef float llgo_f4 __attribute__((vector_size(16)));
typedef double llgo_d2 __attribute__((vector_size(16)));
typedef uint32_t llgo_u4 __attribute__((vector_size(16)));
typedef uint8_t llgo_b16 __attribute__((vector_size(16)));
typedef llgo_f4 llgo_f4_alias;
typedef float llgo_plain4[4];
typedef struct { llgo_f4 a, b; } llgo_vpair;
typedef struct { llgo_f4 a; llgo_b16 b; } llgo_vmixed;
typedef struct { llgo_f4 v[3]; } llgo_vtriple;
typedef struct { unsigned char tag; llgo_f4 v; unsigned char tail; } llgo_vbox;
typedef struct { llgo_plain4 v; } llgo_plain_box;

static __attribute__((noinline)) llgo_f4 llgo_vmake(float a, float b, float c, float d) { return (llgo_f4){a,b,c,d}; }
static __attribute__((noinline)) llgo_f4 llgo_vadd(llgo_f4 a, llgo_f4 b) { return a+b; }
static __attribute__((noinline)) llgo_u4 llgo_vxor(llgo_u4 a, llgo_u4 b) { return a^b; }
static __attribute__((noinline)) llgo_b16 llgo_vbytes(llgo_b16 a) { return a+(llgo_b16){1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16}; }
static __attribute__((noinline)) llgo_d2 llgo_vdouble(llgo_d2 a, llgo_d2 b) { return a-b; }
static __attribute__((noinline)) llgo_f4_alias llgo_valias(llgo_f4_alias v) { return v; }
static __attribute__((noinline)) float llgo_vlane(llgo_f4 a, int n) { return a[n]; }
static __attribute__((noinline)) float llgo_vsum(llgo_f4 v) { return v[0]+2*v[1]+3*v[2]+4*v[3]; }
static __attribute__((noinline)) float llgo_pair_sum(llgo_vpair p) { return llgo_vsum(p.a)+10*llgo_vsum(p.b); }
static __attribute__((noinline)) llgo_vpair llgo_pair_swap(llgo_vpair p) { return (llgo_vpair){p.b,p.a}; }
static __attribute__((noinline)) llgo_vmixed llgo_mixed_swap(llgo_vmixed p) { p.a += (llgo_f4){1,2,3,4}; p.b += (llgo_b16){16,15,14,13,12,11,10,9,8,7,6,5,4,3,2,1}; return p; }
static __attribute__((noinline)) llgo_vtriple llgo_triple_rotate(llgo_vtriple p) { return (llgo_vtriple){{p.v[1],p.v[2],p.v[0]}}; }
static __attribute__((noinline)) float llgo_box_read(const llgo_vbox *p) { return p->tag+llgo_vsum(p->v)+p->tail; }
static __attribute__((noinline)) llgo_vbox llgo_box_value(llgo_vbox p) { p.tag++; p.v += (llgo_f4){1,2,3,4}; p.tail+=2; return p; }
static __attribute__((noinline)) float llgo_plain_value(llgo_plain_box p) { return p.v[0]+2*p.v[1]+3*p.v[2]+4*p.v[3]; }
// Seven vector registers leave too little room for the pair on ARM64. The
// whole pair must spill, and the following vector must use the correct slot.
static __attribute__((noinline)) float llgo_pair_spill(llgo_f4 a,llgo_f4 b,llgo_f4 c,llgo_f4 d,llgo_f4 e,llgo_f4 f,llgo_f4 g,llgo_vpair pair,llgo_f4 tail) {
 return llgo_vsum(a)+2*llgo_vsum(b)+3*llgo_vsum(c)+4*llgo_vsum(d)+5*llgo_vsum(e)+6*llgo_vsum(f)+7*llgo_vsum(g)+8*llgo_pair_sum(pair)+9*llgo_vsum(tail);
}
static __attribute__((noinline)) llgo_f4 llgo_many_vectors(llgo_f4 a,llgo_f4 b,llgo_f4 c,llgo_f4 d,llgo_f4 e,llgo_f4 f,llgo_f4 g,llgo_f4 h,llgo_f4 i,llgo_f4 j) { return a+2*b+3*c+4*d+5*e+6*f+7*g+8*h+9*i+10*j; }
static __attribute__((noinline)) uint64_t llgo_box_layout(void) { return sizeof(llgo_vbox)*10000+offsetof(llgo_vbox,v)*100+_Alignof(llgo_vbox); }
static llgo_f4 llgo_vector_global = {3,5,7,11};
static __attribute__((noinline)) llgo_f4 llgo_vglobal(void) { return llgo_vector_global; }
#endif
