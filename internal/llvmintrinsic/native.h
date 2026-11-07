#ifndef LLGO_INTRINSIC_NATIVE_H
#define LLGO_INTRINSIC_NATIVE_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  void *type;
  void *declaration;
  char *name;
  char *error;
  unsigned *immargs;
  size_t immarg_count;
} LLGOIntrinsicResult;

LLGOIntrinsicResult llgo_intrinsic_resolve(unsigned id, void *type,
                                           void *module);
void llgo_intrinsic_dispose(LLGOIntrinsicResult result);
uint64_t llgo_intrinsic_array_length(void *type);
unsigned llgo_intrinsic_scalar_bits(void *type);
int llgo_intrinsic_vector_shape(void *type, unsigned *minimum, int *scalable);
void *llgo_intrinsic_vector_type(void *element, unsigned minimum, int scalable);
int llgo_intrinsic_integer_range(void *value, uint64_t low, uint64_t high,
                                 int unsigned_range);
unsigned llgo_intrinsic_pointer_index_bits(void *type, const char *layout);

// Linked metadata is queried rather than maintaining a copy of LLVM's table.
typedef struct {
  const char *kind;
  unsigned argument;
  unsigned argument_kind;
  unsigned extra;
  int scalable;
} LLGOIntrinsicDescriptor;
unsigned llgo_intrinsic_count(void);
int llgo_intrinsic_is_target(unsigned id);
const char *llgo_intrinsic_base_name(unsigned id, size_t *length);
size_t llgo_intrinsic_descriptors(unsigned id, LLGOIntrinsicDescriptor *out);
void *llgo_intrinsic_sample(void *context, unsigned id, unsigned variant);

#ifdef __cplusplus
}
#endif
#endif
