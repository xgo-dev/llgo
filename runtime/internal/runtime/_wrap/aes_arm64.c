/* Armv8 AES instructions, using the generic Go uint32 key schedule layout.
 * Calls are guarded by internal/cpu.ARM64.HasAES (including GODEBUG).
 * Keep the optional instructions in target-specific, non-inlined functions
 * so full LTO cannot move them onto the portable fallback path. */
#if defined(__aarch64__)
#include <arm_neon.h>
#include <stdint.h>

#define AES_TARGET __attribute__((target("aes")))
#define AES_ENTRY AES_TARGET __attribute__((noinline))

static AES_TARGET uint32_t subword(uint32_t w)
{
    /* Repeated columns make ShiftRows an identity, leaving just SubBytes. */
    uint8x16_t v = vreinterpretq_u8_u32(vdupq_n_u32(w));
    return vgetq_lane_u32(vreinterpretq_u32_u8(vaeseq_u8(v, vdupq_n_u8(0))), 0);
}

static uint8x16_t roundkey(const uint32_t *p)
{
    return vrev32q_u8(vld1q_u8((const uint8_t *)p));
}

AES_ENTRY void llgo_aes_expand_arm64(const uint8_t *key, uint32_t size,
                                     uint32_t *enc, uint32_t *dec)
{
    uint32_t nk = size / 4, rounds = nk + 6, words = 4 * (rounds + 1);
    for (uint32_t i = 0; i < nk; ++i) {
        enc[i] = (uint32_t)key[4*i] << 24 | (uint32_t)key[4*i+1] << 16 |
                 (uint32_t)key[4*i+2] << 8 | key[4*i+3];
    }
    uint32_t rc = 1;
    for (uint32_t i = nk; i < words; ++i) {
        uint32_t t = enc[i-1];
        if (i % nk == 0) {
            t = subword((t << 8) | (t >> 24)) ^ (rc << 24);
            rc = (rc << 1) ^ ((rc >> 7) * 0x11b);
        } else if (nk == 8 && i % nk == 4) {
            t = subword(t);
        }
        enc[i] = enc[i-nk] ^ t;
    }
    for (uint32_t r = 0; r <= rounds; ++r) {
        uint8x16_t k = roundkey(enc + 4 * (rounds-r));
        if (r != 0 && r != rounds)
            k = vaesimcq_u8(k);
        vst1q_u8((uint8_t *)(dec + 4*r), vrev32q_u8(k));
    }
}

AES_ENTRY void llgo_aes_encrypt_arm64(const uint32_t *key, uint32_t rounds,
                                      uint8_t *dst, const uint8_t *src)
{
    uint8x16_t s = vld1q_u8(src);
    for (uint32_t r = 0; r < rounds-1; ++r)
        s = vaesmcq_u8(vaeseq_u8(s, roundkey(key + 4*r)));
    s = vaeseq_u8(s, roundkey(key + 4*(rounds-1)));
    vst1q_u8(dst, veorq_u8(s, roundkey(key + 4*rounds)));
}

AES_ENTRY void llgo_aes_decrypt_arm64(const uint32_t *key, uint32_t rounds,
                                      uint8_t *dst, const uint8_t *src)
{
    uint8x16_t s = vld1q_u8(src);
    for (uint32_t r = 0; r < rounds-1; ++r)
        s = vaesimcq_u8(vaesdq_u8(s, roundkey(key + 4*r)));
    s = vaesdq_u8(s, roundkey(key + 4*(rounds-1)));
    vst1q_u8(dst, veorq_u8(s, roundkey(key + 4*rounds)));
}

AES_ENTRY void llgo_aes_ctr_arm64(const uint32_t *key, uint32_t rounds,
                                  uint8_t *dst, const uint8_t *src,
                                  uint64_t lo, uint64_t hi, uint32_t blocks)
{
    uint8x16_t s[8], input[8];
    for (uint32_t i = 0; i < blocks; ++i) {
        uint64_t l = lo + i;
        uint64x2_t counter = vdupq_n_u64(0);
        counter = vsetq_lane_u64(hi + (l < lo), counter, 0);
        counter = vsetq_lane_u64(l, counter, 1);
        s[i] = vrev64q_u8(vreinterpretq_u8_u64(counter));
        /* Load every source block before any store, as ctrBlocksN requires. */
        input[i] = vld1q_u8(src + 16*i);
    }
    for (uint32_t r = 0; r < rounds-1; ++r) {
        uint8x16_t k = roundkey(key + 4*r);
        for (uint32_t i = 0; i < blocks; ++i)
            s[i] = vaesmcq_u8(vaeseq_u8(s[i], k));
    }
    uint8x16_t penultimate = roundkey(key + 4*(rounds-1));
    uint8x16_t last = roundkey(key + 4*rounds);
    for (uint32_t i = 0; i < blocks; ++i) {
        s[i] = veorq_u8(vaeseq_u8(s[i], penultimate), last);
        vst1q_u8(dst + 16*i, veorq_u8(s[i], input[i]));
    }
}
#endif
