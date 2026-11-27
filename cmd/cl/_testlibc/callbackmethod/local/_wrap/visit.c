#include <stdint.h>

typedef struct {
  int32_t kind;
  int32_t data;
} llgo_node;

typedef int32_t (*llgo_visitor)(llgo_node, void *);

int32_t llgo_visit_node(llgo_node node, llgo_visitor visitor, void *data) {
  return visitor(node, data) + 1;
}
