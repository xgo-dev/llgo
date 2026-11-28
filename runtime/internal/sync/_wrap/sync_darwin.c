/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0.
 */

#include <dlfcn.h>
#include <errno.h>
#include <pthread.h>
#include <stddef.h>
#include <stdint.h>

typedef int (*llgo_wait_fn)(void *, uint64_t, size_t, uint32_t);
typedef int (*llgo_wait_timeout_fn)(void *, uint64_t, size_t, uint32_t, uint32_t, uint64_t);
typedef int (*llgo_wake_fn)(void *, size_t, uint32_t);

/* OS_CLOCK_MACH_ABSOLUTE_TIME from <os/clock.h> */
#define LLGO_OS_CLOCK_MACH_ABSOLUTE_TIME 32u

static pthread_once_t llgo_darwin_sync_once = PTHREAD_ONCE_INIT;
static llgo_wait_fn llgo_wait_addr_fn;
static llgo_wait_timeout_fn llgo_wait_addr_timeout_fn;
static llgo_wake_fn llgo_wake_addr_any_fn;
static llgo_wake_fn llgo_wake_addr_all_fn;

static void llgo_darwin_sync_init(void)
{
    llgo_wait_addr_fn = (llgo_wait_fn)dlsym(RTLD_DEFAULT, "os_sync_wait_on_address");
    llgo_wait_addr_timeout_fn = (llgo_wait_timeout_fn)dlsym(RTLD_DEFAULT, "os_sync_wait_on_address_with_timeout");
    llgo_wake_addr_any_fn = (llgo_wake_fn)dlsym(RTLD_DEFAULT, "os_sync_wake_by_address_any");
    llgo_wake_addr_all_fn = (llgo_wake_fn)dlsym(RTLD_DEFAULT, "os_sync_wake_by_address_all");
}

int llgo_darwin_has_wait_uint32(void)
{
    pthread_once(&llgo_darwin_sync_once, llgo_darwin_sync_init);
    return llgo_wait_addr_fn != NULL && llgo_wake_addr_any_fn != NULL;
}

int llgo_darwin_has_wait_uint32_timeout(void)
{
    pthread_once(&llgo_darwin_sync_once, llgo_darwin_sync_init);
    return llgo_wait_addr_fn != NULL &&
           llgo_wait_addr_timeout_fn != NULL &&
           llgo_wake_addr_any_fn != NULL &&
           llgo_wake_addr_all_fn != NULL;
}

int llgo_darwin_wait_uint32(uint32_t *addr, uint32_t value)
{
    if (llgo_wait_addr_fn == NULL)
        return ENOPROTOOPT;
    int saved_errno = errno;
    int ret = llgo_wait_addr_fn(addr, (uint64_t)value, sizeof(*addr), 0);
    int err = ret < 0 ? errno : 0;
    errno = saved_errno;
    return err;
}

int llgo_darwin_wait_uint32_timeout(uint32_t *addr, uint32_t value, uint64_t timeout_ns)
{
    if (llgo_wait_addr_timeout_fn == NULL)
        return ENOPROTOOPT;
    if (timeout_ns == 0)
        return ETIMEDOUT;
    int saved_errno = errno;
    int ret = llgo_wait_addr_timeout_fn(addr, (uint64_t)value, sizeof(*addr), 0,
                                        LLGO_OS_CLOCK_MACH_ABSOLUTE_TIME, timeout_ns);
    int err = ret < 0 ? errno : 0;
    errno = saved_errno;
    return err;
}

void llgo_darwin_wake_uint32(uint32_t *addr)
{
    if (llgo_wake_addr_any_fn == NULL)
        return;
    int saved_errno = errno;
    llgo_wake_addr_any_fn(addr, sizeof(*addr), 0);
    errno = saved_errno;
}

void llgo_darwin_wake_uint32_all(uint32_t *addr)
{
    if (llgo_wake_addr_all_fn == NULL)
        return;
    int saved_errno = errno;
    llgo_wake_addr_all_fn(addr, sizeof(*addr), 0);
    errno = saved_errno;
}
