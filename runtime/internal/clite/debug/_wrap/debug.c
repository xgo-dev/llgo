#if defined(__linux__)
#define UNW_LOCAL_ONLY
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <features.h>
#endif

#include <dlfcn.h>
#include <libunwind.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#ifdef __APPLE__
#include <limits.h>
#include <libgen.h>
#endif

typedef struct {
    char *function;
    char *file;
    int line;
    void *entry;
} LlgoSymbolInfo;

void *llgo_address() {
    return __builtin_return_address(0);
}

int llgo_addrinfo(void *addr, Dl_info *info) {
    return dladdr(addr, info);
}

static char *llgo_strdup_range(const char *s, size_t n) {
    char *out = (char *)malloc(n + 1);
    if (out == NULL) {
        return NULL;
    }
    memcpy(out, s, n);
    out[n] = '\0';
    return out;
}

static char *llgo_strdup_or_null(const char *s) {
    if (s == NULL || s[0] == '\0') {
        return NULL;
    }
    return strdup(s);
}

static void llgo_trim_newline(char *s) {
    if (s == NULL) {
        return;
    }
    size_t n = strlen(s);
    while (n > 0 && (s[n - 1] == '\n' || s[n - 1] == '\r')) {
        s[--n] = '\0';
    }
}

static int llgo_file_exists(const char *path) {
    struct stat st;
    return path != NULL && stat(path, &st) == 0;
}

static char *llgo_shell_quote(const char *s) {
    size_t extra = 2;
    for (const char *p = s; *p != '\0'; p++) {
        extra += (*p == '\'') ? 4 : 1;
    }
    char *out = (char *)malloc(extra + 1);
    if (out == NULL) {
        return NULL;
    }
    char *q = out;
    *q++ = '\'';
    for (const char *p = s; *p != '\0'; p++) {
        if (*p == '\'') {
            memcpy(q, "'\\''", 4);
            q += 4;
        } else {
            *q++ = *p;
        }
    }
    *q++ = '\'';
    *q = '\0';
    return out;
}

static int llgo_parse_location(const char *loc, LlgoSymbolInfo *out) {
    if (loc == NULL || loc[0] == '\0' || strcmp(loc, "??:0:0") == 0) {
        return 0;
    }
    const char *last = strrchr(loc, ':');
    if (last == NULL) {
        return 0;
    }
    const char *prev = last;
    while (prev > loc) {
        prev--;
        if (*prev == ':') {
            break;
        }
    }
    if (*prev != ':' || prev == loc) {
        return 0;
    }
    int line = atoi(prev + 1);
    if (line <= 0) {
        return 0;
    }
    free(out->file);
    out->file = llgo_strdup_range(loc, (size_t)(prev - loc));
    out->line = line;
    return out->file != NULL;
}

static uintptr_t llgo_symbol_addr(void *addr, const Dl_info *dli) {
#ifdef __APPLE__
    if (dli != NULL && dli->dli_fbase != NULL) {
        return (uintptr_t)addr - (uintptr_t)dli->dli_fbase + 0x100000000ULL;
    }
#endif
    return (uintptr_t)addr;
}

static char *llgo_dsym_path(const char *obj) {
#ifdef __APPLE__
    if (obj == NULL || obj[0] == '\0') {
        return NULL;
    }
    const char *base = strrchr(obj, '/');
    base = base == NULL ? obj : base + 1;
    size_t n = strlen(obj) + strlen(".dSYM/Contents/Resources/DWARF/") + strlen(base) + 1;
    char *path = (char *)malloc(n);
    if (path == NULL) {
        return NULL;
    }
    snprintf(path, n, "%s.dSYM/Contents/Resources/DWARF/%s", obj, base);
    if (llgo_file_exists(path)) {
        return path;
    }
    free(path);
    n = strlen(obj) + strlen(".dSYM") + 1;
    path = (char *)malloc(n);
    if (path == NULL) {
        return NULL;
    }
    snprintf(path, n, "%s.dSYM", obj);
    if (llgo_file_exists(path)) {
        return path;
    }
    free(path);
#else
    (void)obj;
#endif
    return NULL;
}

static int llgo_run_symbolizer(const char *obj, uintptr_t addr, LlgoSymbolInfo *out) {
    if (obj == NULL || obj[0] == '\0') {
        return 0;
    }
    char *qobj = llgo_shell_quote(obj);
    if (qobj == NULL) {
        return 0;
    }
	char cmd[4096];
	snprintf(cmd, sizeof(cmd),
        "llvm-symbolizer --functions=linkage --inlining=false --no-demangle --obj=%s 0x%llx 2>/dev/null",
        qobj, (unsigned long long)addr);
    free(qobj);

    FILE *fp = popen(cmd, "r");
    if (fp == NULL) {
        return 0;
    }
    char function[1024];
    char location[2048];
    int got = fgets(function, sizeof(function), fp) != NULL;
    got = got && fgets(location, sizeof(location), fp) != NULL;
    pclose(fp);
    if (!got) {
        return 0;
    }
    llgo_trim_newline(function);
    llgo_trim_newline(location);
    if (function[0] != '\0' && strcmp(function, "??") != 0) {
        free(out->function);
        out->function = strdup(function);
    }
    return llgo_parse_location(location, out);
}

int llgo_symbolize(void *addr, LlgoSymbolInfo *out) {
    memset(out, 0, sizeof(*out));
    Dl_info dli;
    memset(&dli, 0, sizeof(dli));
    if (dladdr(addr, &dli) != 0) {
        out->function = llgo_strdup_or_null(dli.dli_sname);
        out->entry = dli.dli_saddr;
    }
    uintptr_t saddr = llgo_symbol_addr(addr, &dli);
    char *dsym = llgo_dsym_path(dli.dli_fname);
    int ok = 0;
    if (dsym != NULL) {
        ok = llgo_run_symbolizer(dsym, saddr, out);
        free(dsym);
    }
    if (!ok) {
        ok = llgo_run_symbolizer(dli.dli_fname, saddr, out);
    }
    return (out->function != NULL || out->file != NULL) ? 1 : 0;
}

void llgo_symbolinfo_free(LlgoSymbolInfo *info) {
    if (info == NULL) {
        return;
    }
    free(info->function);
    free(info->file);
    memset(info, 0, sizeof(*info));
}

void llgo_stacktrace(int skip, void *ctx, int (*fn)(void *ctx, void *pc, void *offset, void *sp, char *name)) {
    unw_cursor_t cursor;
    unw_context_t context;
    unw_word_t offset, pc, sp;
    char fname[256];
    unw_getcontext(&context);
    unw_init_local(&cursor, &context);
    int depth = 0;
    while (unw_step(&cursor) > 0) {
        if (depth < skip) {
            depth++;
            continue;
        }
        if (unw_get_reg(&cursor, UNW_REG_IP, &pc) == 0) {
            unw_get_proc_name(&cursor, fname, sizeof(fname), &offset);
            unw_get_reg(&cursor, UNW_REG_SP, &sp);
            if (fn(ctx, (void*)pc, (void*)offset, (void*)sp, fname) == 0) {
                return;
            }
        }
    }
}
