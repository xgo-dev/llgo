/* Thin C glue between llgo-compiled Go code and the ESP-IDF mbedtls TLS
 * stack (mbedtls 4.1.1 in IDF v6.3-dev).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md. The main
 * component needs REQUIRES mbedtls - plus the net glue in the same build,
 * because the socket/clock primitives this glue's caller drives (llgo_net_*)
 * are defined once in net_glue.c.
 *
 * Design: the glue owns one heap-allocated context per TLS session (the
 * mbedtls ssl context, ssl config and parsed CA) and hands it to Go as an
 * opaque uintptr handle. Every function is a SINGLE non-blocking attempt -
 * mbedtls 4.1 reports what it needs through MBEDTLS_ERR_SSL_WANT_READ/
 * WANT_WRITE when the BIO would block - and the Go side (tls.go) drives the
 * bounded-quantum retry loops over llgo_net_poll, exactly like the socket
 * calls of pkg/esp/net. No loop, no deadline logic lives here: the glue
 * stays stateless between attempts (all progress is inside mbedtls's own
 * context) and nothing can block the single FreeRTOS task the llgo
 * scheduler runs on.
 *
 * The crypto randomness comes from PSA Crypto: mbedtls 4.1 wires the SSL
 * layer to psa_generate_random, and IDF initializes PSA at startup from the
 * hardware TRNG (esp_fill_random, port/esp_hardware.c). This glue calls
 * psa_crypto_init() defensively (it is idempotent and cheap once done) and
 * maps a failure to TLS_ERR_PSA_INIT (0x7E06); the ctr_drbg of older mbedtls
 * releases is a private header in 4.1.1 and is not used.
 *
 * Rules kept from the pkg/esp/README.md contract:
 *   (1) integer-and-pointer-only signatures - no float values cross the
 *       Go<->C link;
 *   (2) no large stack buffers: the biggest on-stack object is a fixed
 *       config struct; the CA PEM copy and the context itself live on the
 *       heap, owned by the context;
 *   (3) ownership is explicit: llgo_tls_new allocates the context,
 *       llgo_tls_free releases everything (context, config, CA, PEM copy);
 *       every TLS failure path in tls.go frees what it allocated;
 *   (4) no callbacks into Go anywhere - the BIO callbacks below are called
 *       BY mbedtls, synchronously, inside one llgo_tls_* attempt, and only
 *       touch lwip sockets.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include <lwip/sockets.h>

#include "psa/crypto.h"
#include "mbedtls/ssl.h"
#include "mbedtls/x509_crt.h"

/* One TLS session: the mbedtls state plus the fd its BIO talks to. */
typedef struct {
    mbedtls_ssl_context ssl;
    mbedtls_ssl_config conf;
    mbedtls_x509_crt ca; /* used only when the CA PEM was given */
    int32_t fd;
} llgo_tls_ctx_t;

/* BIO callbacks on the non-blocking lwip socket, the shape mbedtls wants
 * for non-blocking operation: EAGAIN maps to WANT_WRITE/WANT_READ (the
 * caller retries), a 0-byte read signals the TCP peer closed (mbedtls turns
 * it into MBEDTLS_ERR_SSL_CONN_EOF), and any other failure is the negated
 * lwip errno - mbedtls treats every negative BIO return as fatal and hands
 * it to the caller unchanged. In the record phase (Read/Write/Close) that
 * means ECONNRESET and friends surface at the Go side with their real errno
 * values; the handshake phase is different - tls.go's dial loop maps every
 * non-WANT handshake code to a 0x7E00 TLS code, so a BIO error there
 * reaches Go as TLS_ERR_HANDSHAKE, not as its errno. */
static int llgo_tls_bio_send(void *ctx, const unsigned char *buf, size_t len)
{
    int32_t fd = ((llgo_tls_ctx_t *)ctx)->fd;
    int n = lwip_send((int)fd, buf, len, 0);
    if (n < 0) {
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            return MBEDTLS_ERR_SSL_WANT_WRITE;
        }
        return -errno;
    }
    return n;
}

static int llgo_tls_bio_recv(void *ctx, unsigned char *buf, size_t len)
{
    int32_t fd = ((llgo_tls_ctx_t *)ctx)->fd;
    int n = lwip_recv((int)fd, buf, len, 0);
    if (n < 0) {
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            return MBEDTLS_ERR_SSL_WANT_READ;
        }
        return -errno;
    }
    return n; /* 0 = the TCP peer closed; mbedtls maps this to CONN_EOF */
}

/* Allocate and set up the TLS client session for an already-connected
 * non-blocking socket: defaults, optional CA parse + VERIFY_REQUIRED (else
 * VERIFY_NONE), BIO wiring, ssl_setup. The CA PEM must be copied before
 * parsing: mbedtls's PEM path requires a NUL terminator (x509_crt_parse
 * detects PEM by checking buf[buflen-1] == '\0') and the Go slice is not
 * terminated. Returns 0 with *out_ctx set, TLS_ERR_CA_PARSE (0x7E03) for an
 * unusable CA, TLS_ERR_PSA_INIT (0x7E06) if PSA Crypto does not start, or a
 * negative mbedtls code for the remaining setup failures (passed through
 * unchanged - only tls.go's handshake loop maps unmapped codes to
 * TLS_ERR_HANDSHAKE). */
int32_t llgo_tls_new(int32_t fd, const uint8_t *ca_pem, int32_t ca_len,
                     uintptr_t *out_ctx)
{
    if (psa_crypto_init() != PSA_SUCCESS) {
        return 0x7E06; /* TLS_ERR_PSA_INIT - no random source for the handshake */
    }

    llgo_tls_ctx_t *c = malloc(sizeof(*c));
    if (c == NULL) {
        return MBEDTLS_ERR_SSL_ALLOC_FAILED;
    }
    c->fd = fd;
    mbedtls_ssl_init(&c->ssl);
    mbedtls_ssl_config_init(&c->conf);
    mbedtls_x509_crt_init(&c->ca);

    int ret = mbedtls_ssl_config_defaults(&c->conf, MBEDTLS_SSL_IS_CLIENT,
                                          MBEDTLS_SSL_TRANSPORT_STREAM,
                                          MBEDTLS_SSL_PRESET_DEFAULT);
    if (ret != 0) {
        mbedtls_ssl_free(&c->ssl);
        mbedtls_ssl_config_free(&c->conf);
        free(c);
        return ret;
    }
    if (ca_pem != NULL && ca_len > 0) {
        char *pem = malloc((size_t)ca_len + 1);
        if (pem == NULL) {
            mbedtls_ssl_free(&c->ssl);
            mbedtls_ssl_config_free(&c->conf);
            free(c);
            return MBEDTLS_ERR_SSL_ALLOC_FAILED;
        }
        memcpy(pem, ca_pem, (size_t)ca_len);
        pem[ca_len] = '\0';
        ret = mbedtls_x509_crt_parse(&c->ca, (const unsigned char *)pem,
                                     (size_t)ca_len + 1);
        free(pem);
        if (ret != 0) {
            /* < 0: hard parse error. > 0: mbedtls counts failed members
             * and KEEPS the ones that parsed - free them before tearing
             * the context down (free on an empty chain is a no-op). The
             * PEM may hold one CA or a chain of concatenated PEM blocks;
             * any failed member rejects the whole parameter. */
            mbedtls_x509_crt_free(&c->ca);
            mbedtls_ssl_free(&c->ssl);
            mbedtls_ssl_config_free(&c->conf);
            free(c);
            return 0x7E03; /* TLS_ERR_CA_PARSE */
        }
        mbedtls_ssl_conf_ca_chain(&c->conf, &c->ca, NULL);
        mbedtls_ssl_conf_authmode(&c->conf, MBEDTLS_SSL_VERIFY_REQUIRED);
    } else {
        mbedtls_ssl_conf_authmode(&c->conf, MBEDTLS_SSL_VERIFY_NONE);
    }
    mbedtls_ssl_set_bio(&c->ssl, c, llgo_tls_bio_send, llgo_tls_bio_recv,
                        NULL);
    /* Explicitly opt out of hostname verification: mbedtls 4.x refuses a
     * VERIFY_REQUIRED handshake on a client that never called
     * mbedtls_ssl_set_hostname (CERTIFICATE_VERIFICATION_WITHOUT_HOSTNAME,
     * -0x5D80) unless the caller shows intent - the documented way is a
     * set_hostname call with NULL, which keeps chain + validity checks and
     * only drops the CN/SAN match. This package's API has no hostname
     * parameter, so the caller's identity check is its trust in the CA. */
    mbedtls_ssl_set_hostname(&c->ssl, NULL);
    if ((ret = mbedtls_ssl_setup(&c->ssl, &c->conf)) != 0) {
        mbedtls_x509_crt_free(&c->ca);
        mbedtls_ssl_free(&c->ssl);
        mbedtls_ssl_config_free(&c->conf);
        free(c);
        return ret;
    }
    *out_ctx = (uintptr_t)c;
    return 0;
}

/* One handshake step. 0 = the handshake is complete; WANT_READ/WANT_WRITE
 * keep the loop alive; every other return is a fatal mbedtls code the
 * caller maps (X509_CERT_VERIFY_FAILED -> TLS_ERR_CERT_VERIFY and so on).
 * All state lives in the ssl context, so retrying later resumes exactly
 * where this attempt stopped.
 *
 * mbedtls's handshake_step reports a bare 0 for its very first call too:
 * from the HELLO_REQUEST state it only advances an internal state marker
 * without touching the network, so ssl.h says to keep stepping while
 * mbedtls_ssl_is_handshake_over reports 0. This wrapper absorbs that
 * transition (reporting it as WANT_READ - the loop simply polls again) so
 * that a 0 return here always means the handshake is genuinely over. */
int32_t llgo_tls_handstep(uintptr_t ctx)
{
    llgo_tls_ctx_t *c = (llgo_tls_ctx_t *)ctx;
    if (mbedtls_ssl_is_handshake_over(&c->ssl)) {
        return 0;
    }
    int r = mbedtls_ssl_handshake_step(&c->ssl);
    if (r == 0) {
        /* The step completed; if it was the last one the handshake is now
         * over (report success at once), otherwise it was an internal
         * transition (HELLO_REQUEST -> CLIENT_HELLO) that touched no
         * network - report WANT_READ so the Go loop keeps polling. */
        if (mbedtls_ssl_is_handshake_over(&c->ssl)) {
            return 0;
        }
        return MBEDTLS_ERR_SSL_WANT_READ;
    }
    return r;
}

/* One read attempt: 0 with *out_n set on plaintext, WANT_* when the record
 * layer needs the socket, or a fatal negative mbedtls code (CONN_EOF and
 * PEER_CLOSE_NOTIFY included - tls.go maps both to the orderly-close
 * sentinel). mbedtls returns what fits the caller's buffer and keeps the
 * rest of the record buffered for the next call: short reads are normal. */
int32_t llgo_tls_read(uintptr_t ctx, uint8_t *buf, int32_t len,
                      int32_t *out_n)
{
    llgo_tls_ctx_t *c = (llgo_tls_ctx_t *)ctx;
    int r = mbedtls_ssl_read(&c->ssl, buf, (size_t)len);
    if (r > 0) {
        *out_n = r;
        return 0;
    }
    return r; /* mbedtls documents no success-with-zero-bytes return */
}

/* One write attempt: 0 (the whole buffer went into one record) or a WANT_*
 * / fatal negative code - mbedtls accepts plaintext all-or-nothing, so
 * there is no partial-write case to report. */
int32_t llgo_tls_write(uintptr_t ctx, const uint8_t *buf, int32_t len,
                       int32_t *out_n)
{
    llgo_tls_ctx_t *c = (llgo_tls_ctx_t *)ctx;
    int r = mbedtls_ssl_write(&c->ssl, buf, (size_t)len);
    if (r == len) {
        *out_n = r;
        return 0;
    }
    return r;
}

/* One close_notify attempt. 0 = the notification is out; WANT_* keeps
 * Close's bounded wait alive; anything else is fatal and the caller tears
 * the session down regardless. */
int32_t llgo_tls_close_notify(uintptr_t ctx)
{
    llgo_tls_ctx_t *c = (llgo_tls_ctx_t *)ctx;
    return mbedtls_ssl_close_notify(&c->ssl);
}

/* Release the whole context: ssl state, config, parsed CA, the struct. The
 * fd is NOT closed here - it belongs to the Go caller, which shuts it down
 * in both directions first (see TLSConn.Close). */
void llgo_tls_free(uintptr_t ctx)
{
    llgo_tls_ctx_t *c = (llgo_tls_ctx_t *)ctx;
    if (c == NULL) {
        return;
    }
    mbedtls_x509_crt_free(&c->ca);
    mbedtls_ssl_free(&c->ssl);
    mbedtls_ssl_config_free(&c->conf);
    free(c);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_tls_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
