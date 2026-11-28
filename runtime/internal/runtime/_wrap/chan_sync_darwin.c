/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0.
 */

#include <dlfcn.h>
#include <stddef.h>
#include <stdint.h>

typedef int (*channel_wait)(void *, uint64_t, size_t, uint32_t);
typedef int (*channel_wake)(void *, size_t, uint32_t);

static channel_wait wait_on_address;
static channel_wake wake_by_address;

int llgo_bind_channel_address_wait(void) {
    /*
     * Both APIs are present starting with macOS 14.4. Runtime lookup
     * keeps the linked executable compatible with older systems and SDKs.
     */
    wait_on_address = (channel_wait)dlsym(RTLD_DEFAULT, "os_sync_wait_on_address");
    wake_by_address = (channel_wake)dlsym(RTLD_DEFAULT, "os_sync_wake_by_address_any");
    return wait_on_address != NULL && wake_by_address != NULL;
}

int llgo_channel_address_wait(uint32_t *notified) {
    return wait_on_address(notified, 0, sizeof(*notified), 0);
}

int llgo_channel_address_wake(uint32_t *notified) {
    return wake_by_address(notified, sizeof(*notified), 0);
}
