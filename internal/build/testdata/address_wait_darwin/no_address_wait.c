#include <dlfcn.h>
#include <string.h>

static void *lookup(void *handle, const char *name)
{
    if (strncmp(name, "os_sync_", sizeof("os_sync_") - 1) == 0)
        return NULL;
    return dlsym(handle, name);
}

// Model macOS versions that do not export the address-wait functions.
__attribute__((used, section("__DATA,__interpose")))
static const struct {
    void *(*replacement)(void *, const char *);
    void *(*original)(void *, const char *);
} interposer = { lookup, dlsym };
