#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wasi/api.h>

/* Go keeps 64-bit string and slice slots on Wasm32. The data address in
 * each slot is a 32-bit linear-memory pointer. */
struct string_result {
    uint64_t data;
    int64_t size;
};

struct slice_result {
    uint64_t data;
    int64_t len;
    int64_t cap;
};

/* libc owns the argv and environment bytes for the process lifetime.
 * The compact link rejects any surviving Go-heap dependency. */
struct slice_result llgo_pinned_argv;
extern char **environ;

struct pinned_env_entry {
    struct string_result value;
    char *key;
    uint32_t key_size;
};
static struct pinned_env_entry *environment;
static size_t env_count;

static void llgo_pinned_snapshot_process(int argc, char **argv) {
    if (argc > 0 && argv != NULL) {
        if ((size_t)argc > SIZE_MAX / sizeof(struct string_result)) {
            exit(71);
        }
        struct string_result *strings = malloc(argc * sizeof(struct string_result));
        if (strings == NULL) {
            exit(71);
        }
        int i = 0;
        for (; i < argc && argv[i] != NULL; ++i) {
            strings[i].data = (uintptr_t)argv[i];
            strings[i].size = strlen(argv[i]);
        }
        llgo_pinned_argv = (struct slice_result){(uintptr_t)strings, i, argc};
    }

    size_t count = 0;
    if (environ == NULL) {
        return;
    }
    while (environ[count] != NULL) {
        count++;
    }
    if (count == 0) {
        return;
    }
    if (count > SIZE_MAX / sizeof(struct pinned_env_entry)) {
        exit(71);
    }
    struct pinned_env_entry *table = malloc(count * sizeof(struct pinned_env_entry));
    if (table == NULL) {
        exit(71);
    }
    environment = table;
    for (size_t i = 0; i < count; ++i) {
        char *entry = environ[i];
        char *sep = strchr(entry, '=');
        if (sep != NULL && sep != entry) {
            table[env_count++] = (struct pinned_env_entry){
                {(uintptr_t)(sep + 1), strlen(sep + 1)},
                entry,
                (uint32_t)(sep - entry)
            };
        }
    }
}

const struct string_result *llgo_pinned_lookup(const char *key, uint32_t size) {
    if (size != 0) {
        for (size_t i = 0; i < env_count; ++i) {
            struct pinned_env_entry *entry = &environment[i];
            if (size == entry->key_size && memcmp(key, entry->key, size) == 0) {
                return &entry->value;
            }
        }
    }
    return NULL;
}

typedef uint32_t (*traceback_parser)(const char *text, uint32_t size);

static int preopen_traceback_level(traceback_parser parse) {
    const struct string_result *env = llgo_pinned_lookup("GOTRACEBACK", 11);
    return parse(env ? (const char *)(uintptr_t)env->data : NULL, env ? (uint32_t)env->size : 0);
}

static void print_preopen_error(const char *operation, __wasi_errno_t error, const char *trace, traceback_parser parse,
                                const struct string_result *messages, uint32_t message_count) {
    fputs("panic: ", stderr);
    fputs(operation, stderr);
    fputs(": ", stderr);
    if (error < message_count && messages[error].size) {
        const struct string_result *message = &messages[error];
        fwrite((const char *)(uintptr_t)message->data, 1, (size_t)message->size, stderr);
    } else {
        char decimal[5];
        char *end = decimal + sizeof(decimal);
        char *p = end;
        do {
            *--p = '0' + error % 10;
            error /= 10;
        } while (error != 0);
        fputs("errno ", stderr);
        fwrite(p, 1, end - p, stderr);
    }
    fputs("\n\n", stderr);
    int traceback = preopen_traceback_level(parse);
    if (traceback != 0) {
        fputs("goroutine 1 [running]:\n", stderr);
        fputs(trace, stderr);
    }
    if (traceback == 2) {
        abort();
    }
    exit(2);
}

void llgo_wasi_compact_startup(int argc, char **argv, traceback_parser parse,
                                      const char *prestat_trace, const char *prestat_name_trace,
                                      const struct string_result *messages, uint32_t message_count) {
    llgo_pinned_snapshot_process(argc, argv);
    for (__wasi_fd_t fd = 0; fd < 3; fd++) {
        __wasi_fdstat_t stat;
        if (__wasi_fd_fdstat_get(fd, &stat) == __WASI_ERRNO_SUCCESS) {
            __wasi_errno_t ignored = __wasi_fd_fdstat_set_flags(fd, stat.fs_flags | __WASI_FDFLAGS_NONBLOCK);
            (void)ignored;
        }
    }

    uint8_t local_name[256];
    uint8_t *name = local_name;
    uint32_t capacity = sizeof(local_name);
    for (__wasi_fd_t fd = 3; ; fd++) {
        __wasi_prestat_t prestat = {0};
        __wasi_errno_t error = __wasi_fd_prestat_get(fd, &prestat);
        if (error == __WASI_ERRNO_BADF) {
            break;
        }
        if (error == __WASI_ERRNO_NOTDIR || prestat.tag != __WASI_PREOPENTYPE_DIR) {
            continue;
        }
        if (error != __WASI_ERRNO_SUCCESS) {
            print_preopen_error("fd_prestat", error, prestat_trace, parse, messages, message_count);
        }
        uint32_t size = prestat.u.dir.pr_name_len;
        if (size > capacity) {
            if (name != local_name) {
                free(name);
            }
            name = malloc(size);
            if (name == NULL) {
                __builtin_trap();
            }
            capacity = size;
        }
        error = __wasi_fd_prestat_dir_name(fd, name, size);
        if (error != __WASI_ERRNO_SUCCESS) {
            print_preopen_error("fd_prestat_dir_name", error, prestat_name_trace, parse, messages, message_count);
        }
    }
    if (name != local_name) {
        free(name);
    }
}
