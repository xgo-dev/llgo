#include <errno.h>
#include <fcntl.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

extern int __llgo_argc;
extern char **__llgo_argv;

static void *read_pcln(uint32_t *size) {
    if (__llgo_argc < 1 || !__llgo_argv || !__llgo_argv[0]) return NULL;
    const char *name = __llgo_argv[0];
    const size_t length = strlen(name);
    const char *directory = name[0] == '/' ? NULL : getenv("PWD");
    const size_t prefix = directory && directory[0] == '/' ? strlen(directory) : 0;
    if (length > 4096 || prefix > 4096) return NULL;
    char *path = malloc(prefix + 1 + length + sizeof(".pclntab"));
    if (!path) return NULL;
    if (prefix) { memcpy(path, directory, prefix); path[prefix] = '/'; }
    size_t offset = prefix ? prefix + 1 : 0;
    memcpy(path + offset, name, length);
    memcpy(path + offset + length, ".pclntab", sizeof(".pclntab"));
    int fd = open(path, O_RDONLY | O_NONBLOCK);
    free(path);
    if (fd < 0) return NULL;
    struct stat st;
    int result = fstat(fd, &st);
    if (result || !S_ISREG(st.st_mode) || st.st_size < 256 || st.st_size > (64 << 20)) {
        close(fd); return NULL;
    }
    uint8_t *buffer = malloc((size_t)st.st_size);
    if (!buffer) { close(fd); return NULL; }
    offset = 0;
    while (offset < (size_t)st.st_size) {
        ssize_t n = read(fd, buffer + offset, (size_t)st.st_size - offset);
        if (n < 0 && errno == EINTR) continue;
        if (n <= 0) { free(buffer); close(fd); return NULL; }
        offset += (size_t)n;
    }
    close(fd);
    *size = (uint32_t)st.st_size;
    return buffer;
}

uint32_t llgo_pcln_pointer_size(void) { return sizeof(void *); }

void llgo_pcln_load(void **out, uint32_t *size) {
    *size = 0;
    *out = read_pcln(size);
}
