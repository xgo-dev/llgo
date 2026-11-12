#include <stddef.h>

size_t llgo_wasm_profile_pointer_size(void) { return sizeof(void *); }

size_t llgo_wasm_profile_long_size(void) { return sizeof(long); }

size_t llgo_wasm_profile_echo_size(size_t value) { return value; }

typedef struct {
  void *data;
  size_t size;
  int tag;
} profile_record;

size_t llgo_wasm_profile_record_size(void) { return sizeof(profile_record); }
size_t llgo_wasm_profile_record_offset(void) { return offsetof(profile_record, size); }

profile_record llgo_wasm_profile_record_round_trip(
    profile_record value, profile_record (*callback)(profile_record)) {
  return callback(value);
}

ptrdiff_t llgo_wasm_profile_signed_word(ptrdiff_t value) { return value; }

int llgo_wasm_profile_words(size_t *words) {
  if (words[0] != 7 || words[1] != 9) return 0;
  words[1] = 11;
  return 1;
}
