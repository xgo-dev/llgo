/* Thin C glue between llgo-compiled Go code and the ESP-IDF lwip TCP socket
 * stack.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md. The main
 * component needs REQUIRES lwip esp_timer.
 *
 * Design point that differs from the other glues: the bounded-quantum model.
 * The llgo fiber scheduler is single-task, so nothing here may block
 * indefinitely. Every socket this glue creates is marked O_NONBLOCK at
 * birth; the Go side drives bounded retry loops over these primitives:
 *   - one non-blocking syscall attempt (recv/send/connect/accept),
 *   - one llgo_net_poll wait (lwip_select with a short timeout - the wait
 *     parks the FreeRTOS task on lwip's select semaphore, which yields the
 *     CPU and keeps the idle tasks, and the watchdog, fed),
 *   - a deadline check in Go.
 * The one deliberate exception is llgo_net_resolve: lwip's DNS client waits
 * on its own timers, which is bounded by lwip's DNS retry configuration, and
 * dotted-quad literals never enter DNS at all (dns.c checks ipaddr_aton
 * first).
 *
 * Error convention: every function returns 0 on success or the lwip errno
 * from the C global `errno` (lwip's socket layer sets it via set_errno).
 * llgo_net_resolve returns the netdb h_errno-family codes (200-213) on
 * failure. Nothing here is esp_err.
 *
 * Rules kept from the pkg/esp/README.md contract:
 *   (1) integer-and-pointer-only signatures - no float values cross the
 *       Go<->C link;
 *   (2) no large stack buffers and no heap allocation: the only buffer is
 *       the 256-byte NUL-terminator copy in llgo_net_resolve, bounded by a
 *       DNS name's maximum length;
 *   (3) no C-side state at all: every socket is an fd integer owned by the
 *       Go caller, so there is nothing to free and no leak paths;
 *   (4) no callbacks anywhere - lwip's asynchronous select semaphore is
 *       only ever waited on synchronously, inside one quantum-bounded call.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <errno.h>
#include <fcntl.h>
#include <stdint.h>
#include <string.h>

#include <lwip/sockets.h>
#include <lwip/netdb.h>

#include "esp_timer.h"

/* Create a TCP socket with O_NONBLOCK already set: the bounded-quantum
 * model in net.go never touches a blocking socket. */
int32_t llgo_net_socket(int32_t *out_fd)
{
    int fd = lwip_socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) {
        return errno;
    }
    int fl = lwip_fcntl(fd, F_GETFL, 0);
    if (fl < 0) {
        int e = errno;
        lwip_close(fd);
        return e;
    }
    if (lwip_fcntl(fd, F_SETFL, fl | O_NONBLOCK) < 0) {
        int e = errno;
        lwip_close(fd);
        return e;
    }
    *out_fd = fd;
    return 0;
}

/* One connect attempt. Returns 0 (connected), EINPROGRESS (net.go then
 * selects for writability and reads SO_ERROR), or another errno (immediate
 * failure, e.g. ENETUNREACH with no IP yet). ip is the network-order
 * address word, port the host-order port. */
int32_t llgo_net_connect(int32_t fd, uint32_t ip, int32_t port)
{
    struct sockaddr_in sa;
    memset(&sa, 0, sizeof(sa));
    sa.sin_family = AF_INET;
    sa.sin_port = htons((uint16_t)port);
    sa.sin_addr.s_addr = ip;
    if (lwip_connect(fd, (struct sockaddr *)&sa, sizeof(sa)) == 0) {
        return 0;
    }
    return errno;
}

/* Wait up to timeout_ms for the fd to become readable (want_write == 0) or
 * writable (want_write == 1). lwip_select also monitors the exception set:
 * a failed connect signals through it, and a peer reset arrives as a read
 * event, so net.go's SO_ERROR / recv handling sees both. *out_ready is 1
 * when the fd was flagged, 0 on timeout. The wait itself parks on lwip's
 * select semaphore - it yields the CPU. */
int32_t llgo_net_poll(int32_t fd, int32_t want_write, int32_t timeout_ms,
                      int32_t *out_ready)
{
    *out_ready = 0;
    if (timeout_ms <= 0) {
        timeout_ms = 1; /* net.go always passes a positive quantum */
    }
    fd_set rs, ws, es;
    FD_ZERO(&rs);
    FD_ZERO(&ws);
    FD_ZERO(&es);
    FD_SET(fd, &rs);
    FD_SET(fd, &es);
    fd_set *wp = NULL;
    if (want_write) {
        FD_SET(fd, &ws);
        wp = &ws;
    }
    struct timeval tv;
    tv.tv_sec = timeout_ms / 1000;
    tv.tv_usec = (timeout_ms % 1000) * 1000;
    int r = lwip_select(fd + 1, &rs, wp, &es, &tv);
    if (r < 0) {
        return errno;
    }
    *out_ready = (r > 0) ? 1 : 0;
    return 0;
}

/* Read and clear SO_ERROR: the deferred result of a non-blocking connect
 * (0 = connected, ECONNREFUSED / ETIMEDOUT / ... = why it failed). */
int32_t llgo_net_sock_error(int32_t fd, int32_t *out_err)
{
    int so_err = 0;
    socklen_t len = sizeof(so_err);
    if (lwip_getsockopt(fd, SOL_SOCKET, SO_ERROR, &so_err, &len) < 0) {
        return errno;
    }
    *out_err = so_err;
    return 0;
}

/* One recv attempt on the non-blocking socket. *out_n is the byte count:
 * > 0 data, == 0 orderly peer close (net.go surfaces both with err 0),
 * < 0 never happens here (failures come back as the returned errno). */
int32_t llgo_net_recv(int32_t fd, uint8_t *buf, int32_t len, int32_t *out_n)
{
    int n = lwip_recvfrom(fd, buf, (size_t)len, 0, NULL, NULL);
    if (n < 0) {
        return errno;
    }
    *out_n = n;
    return 0;
}

/* One send attempt; *out_n is what the socket accepted (short writes are
 * normal once lwip's send buffers fill; net.go loops the rest). */
int32_t llgo_net_send(int32_t fd, const uint8_t *buf, int32_t len,
                      int32_t *out_n)
{
    int n = lwip_send(fd, buf, (size_t)len, 0);
    if (n < 0) {
        return errno;
    }
    *out_n = n;
    return 0;
}

/* Shutdown one or both directions (how: 0 = RD, 1 = WR, 2 = RDWR, POSIX
 * values). net.go's Close shuts RDWR first so a stalled peer can never
 * hold the close open; the wrapper stays separate so callers that want a
 * half-close before their own Close have the primitive. */
int32_t llgo_net_shutdown(int32_t fd, int32_t how)
{
    if (lwip_shutdown(fd, how) < 0) {
        return errno;
    }
    return 0;
}

/* Release the fd. After the RDWR shutdown this cannot wait on the peer:
 * lwip frees the PCB on the spot. */
int32_t llgo_net_close(int32_t fd)
{
    if (lwip_close(fd) < 0) {
        return errno;
    }
    return 0;
}

/* Bind to the wildcard address (every local IP: the station address and,
 * with LWIP_NETIF_LOOPBACK, 127.0.0.1) and start listening with the given
 * backlog. SO_REUSEADDR first: without it lwip keeps the closed listener's
 * PCB in TIME_WAIT-ish lingering state, so a development restart that
 * re-binds the same port fails with EADDRINUSE. */
int32_t llgo_net_bind_listen(int32_t fd, int32_t port, int32_t backlog)
{
    struct sockaddr_in sa;
    int one = 1;
    memset(&sa, 0, sizeof(sa));
    sa.sin_family = AF_INET;
    sa.sin_port = htons((uint16_t)port);
    sa.sin_addr.s_addr = INADDR_ANY;
    if (lwip_setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one)) < 0) {
        return errno;
    }
    if (lwip_bind(fd, (struct sockaddr *)&sa, sizeof(sa)) < 0) {
        return errno;
    }
    if (lwip_listen(fd, backlog) < 0) {
        return errno;
    }
    return 0;
}

/* One accept attempt; the accepted socket is put into O_NONBLOCK too, so
 * every socket the Go side ever sees follows the same quantum model. A
 * failed F_GETFL/F_SETFL closes the connection and propagates the errno,
 * mirroring llgo_net_socket: an accepted socket must never silently stay
 * in blocking mode. The peer address is not handed back - a device probe
 * talks to known peers. */
int32_t llgo_net_accept(int32_t fd, int32_t *out_conn_fd)
{
    struct sockaddr_in sa;
    socklen_t len = sizeof(sa);
    int conn = lwip_accept(fd, (struct sockaddr *)&sa, &len);
    if (conn < 0) {
        return errno;
    }
    int fl = lwip_fcntl(conn, F_GETFL, 0);
    if (fl < 0) {
        int e = errno;
        lwip_close(conn);
        return e;
    }
    if (lwip_fcntl(conn, F_SETFL, fl | O_NONBLOCK) < 0) {
        int e = errno;
        lwip_close(conn);
        return e;
    }
    *out_conn_fd = conn;
    return 0;
}

/* Name resolution through lwip's DNS client. host is NOT NUL-terminated on
 * the Go side, so it is copied into a bounded stack buffer (DNS names are
 * at most 255 bytes; net.go rejects anything longer). Dotted-quad literals
 * return immediately (lwip parses them before consulting DNS); real names
 * block the caller on lwip's DNS semaphore until its timers answer - the
 * one documented freeze in this package. Only AF_INET results are accepted
 * (this package hands out IPv4 words); anything else - including a NULL
 * hostent with h_errno left at 0 - reports HOST_NOT_FOUND rather than a
 * fabricated success. Failures return the netdb h_errno family
 * (HOST_NOT_FOUND 210, TRY_AGAIN 213, ...), not errno. */
int32_t llgo_net_resolve(const uint8_t *host, int32_t host_len,
                         uint32_t *out_ip)
{
    if (host_len <= 0 || host_len > 255) {
        return EINVAL;
    }
    char name[256];
    memcpy(name, host, (size_t)host_len);
    name[host_len] = '\0';
    struct hostent *h = lwip_gethostbyname(name);
    if (h == NULL || h->h_addrtype != AF_INET || h->h_addr_list[0] == NULL) {
        return h_errno != 0 ? h_errno : HOST_NOT_FOUND;
    }
    /* h_addr_list[0] is the network-order address word net.go stores. */
    memcpy(out_ip, h->h_addr_list[0], 4);
    return 0;
}

/* Monotonic uptime in milliseconds for the Go-side deadline bookkeeping.
 * esp_timer runs at microsecond resolution and is independent of the
 * FreeRTOS tick rate. The 32-bit ms counter wraps after ~49.7 days; net.go
 * compares deadlines with unsigned subtraction, which stays correct
 * across the wrap. */
uint32_t llgo_net_now_ms(void)
{
    return (uint32_t)(esp_timer_get_time() / 1000);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_net_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
