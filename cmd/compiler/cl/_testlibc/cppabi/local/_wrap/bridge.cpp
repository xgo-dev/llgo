#include <cstdint>

extern "C" std::int64_t llgo_test_cpp_add(std::int64_t left,
                                           std::int64_t right) {
  return left + right;
}
