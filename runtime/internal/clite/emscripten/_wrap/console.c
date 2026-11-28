#include <stddef.h>
#include <stdint.h>

typedef struct {
  uintptr_t data;
  size_t size;
} llgo_console_iovec;

__attribute__((import_module("wasi_snapshot_preview1"), import_name("fd_write")))
extern uint32_t llgo_console_fd_write(uint32_t fd, uintptr_t buffers,
                                      size_t count, uintptr_t written);

void llgo_console_write(const void *data, uint32_t size) {
  const llgo_console_iovec iov = {(uintptr_t)data, size};
  size_t written;
  llgo_console_fd_write(2, (uintptr_t)&iov, 1, (uintptr_t)&written);
}

void llgo_console_byte(uint8_t data) {
  llgo_console_write(&data, 1);
}
