#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
extern void aes_op_0(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_1(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_2(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_3(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_4(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_5(uint8_t *, const uint8_t *, const uint32_t *);
extern void aes_op_6(uint8_t *, const uint8_t *, const uint32_t *);
extern uint32_t aes_subword(uint32_t);
static void xor_key(uint8_t s[16], const uint32_t *key) {
  aes_op_5(s, s, key);
}
static void expand(const uint8_t *key, int size, uint32_t enc[60],
                   uint32_t dec[60]) {
  int nk = size / 4, rounds = nk + 6, words = 4 * (rounds + 1);
  for (int i = 0; i < nk; i++)
    enc[i] = (uint32_t)key[4 * i] << 24 | (uint32_t)key[4 * i + 1] << 16 |
             (uint32_t)key[4 * i + 2] << 8 | key[4 * i + 3];
  uint32_t rc = 1;
  for (int i = nk; i < words; i++) {
    uint32_t t = enc[i - 1];
    if (i % nk == 0) {
      t = aes_subword(t << 8 | t >> 24) ^ (rc << 24);
      rc = (rc << 1) ^ ((rc >> 7) * 0x11b);
    } else if (nk == 8 && i % nk == 4)
      t = aes_subword(t);
    enc[i] = enc[i - nk] ^ t;
  }
  for (int r = 0; r <= rounds; r++) {
    memcpy(dec + 4 * r, enc + 4 * (rounds - r), 16);
    if (r > 0 && r < rounds)
      aes_op_4((uint8_t *)(dec + 4 * r), (uint8_t *)(dec + 4 * r), NULL);
  }
}
static void block(uint8_t s[16], const uint32_t *key, int rounds, int dec) {
  if (!PREKEY)
    xor_key(s, key);
  for (int r = 0; r < rounds - 1; r++) {
    int index = r + !PREKEY;
    if (dec)
      aes_op_2(s, s, key + 4 * index);
    else
      aes_op_0(s, s, key + 4 * index);
  }
  int last = rounds - PREKEY;
  if (dec)
    aes_op_3(s, s, key + 4 * last);
  else
    aes_op_1(s, s, key + 4 * last);
  if (PREKEY)
    xor_key(s, key + 4 * rounds);
}
static void dump(const uint8_t *s) {
  for (int i = 0; i < 16; i++)
    printf("%02x", s[i]);
  puts("");
}
static uint8_t mul(uint8_t a, uint8_t b) {
  uint8_t r = 0;
  for (int i = 0; i < 8; i++) {
    if (b & 1)
      r ^= a;
    a = (uint8_t)((a << 1) ^ ((a >> 7) * 0x1b));
    b >>= 1;
  }
  return r;
}
int main(void) {
  uint8_t x[17], y[17], want_xor[16];
  for (int i = 0; i < 16; i++) {
    x[i + 1] = i * 7;
    y[i + 1] = i * 11;
    want_xor[i] = x[i + 1] ^ y[i + 1];
  }
  aes_op_6(x + 1, x + 1, (const uint32_t *)(y + 1));
  if (memcmp(x + 1, want_xor, 16)) {
    puts("XORBlock failed");
    return 1;
  }

  if (aes_subword(0x00102030) != 0x63cab704) {
    printf("SubWord failed: %08x\n", aes_subword(0x00102030));
    return 1;
  }
  for (unsigned i = 0; i < 16; i++) {
    uint32_t in[4], got[4], want[4];
    for (unsigned j = 0; j < 4; j++) {
      in[j] = 0x0103070bU * (1 + i * 3 + j * 7);
      uint8_t a = in[j] >> 24, b = in[j] >> 16, c = in[j] >> 8, d = in[j];
      want[j] =
          (uint32_t)(mul(a, 14) ^ mul(b, 11) ^ mul(c, 13) ^ mul(d, 9)) << 24 |
          (uint32_t)(mul(a, 9) ^ mul(b, 14) ^ mul(c, 11) ^ mul(d, 13)) << 16 |
          (uint32_t)(mul(a, 13) ^ mul(b, 9) ^ mul(c, 14) ^ mul(d, 11)) << 8 |
          (mul(a, 11) ^ mul(b, 13) ^ mul(c, 9) ^ mul(d, 14));
    }
    aes_op_4((uint8_t *)got, (uint8_t *)in, NULL);
    if (memcmp(got, want, 16)) {
      printf("InverseMix failed: got=%08x want=%08x\n", got[0], want[0]);
      return 1;
    }
  }
  const uint8_t expected[3][16] = {
      {0x69, 0xc4, 0xe0, 0xd8, 0x6a, 0x7b, 0x04, 0x30, 0xd8, 0xcd, 0xb7, 0x80,
       0x70, 0xb4, 0xc5, 0x5a},
      {0xdd, 0xa9, 0x7c, 0xa4, 0x86, 0x4c, 0xdf, 0xe0, 0x6e, 0xaf, 0x70, 0xa0,
       0xec, 0x0d, 0x71, 0x91},
      {0x8e, 0xa2, 0xb7, 0xca, 0x51, 0x67, 0x45, 0xbf, 0xea, 0xfc, 0x49, 0x90,
       0x4b, 0x49, 0x60, 0x89}};
  for (int test = 0; test < 3; test++) {
    uint8_t key[32], plain[16], buf[17];
    uint32_t enc[60], dec[60];
    int size = 16 + 8 * test, rounds = size / 4 + 6;
    for (int i = 0; i < size; i++)
      key[i] = i;
    for (int i = 0; i < 16; i++)
      plain[i] = i * 0x11;
    expand(key, size, enc, dec);
    memcpy(buf + 1, plain, 16);
    block(buf + 1, enc, rounds, 0);
    if (memcmp(buf + 1, expected[test], 16)) {
      printf("AES-%d Encrypt failed: ", size * 8);
      dump(buf + 1);
      return 1;
    }
    block(buf + 1, dec, rounds, 1);
    if (memcmp(buf + 1, plain, 16)) {
      printf("AES-%d Decrypt failed: ", size * 8);
      dump(buf + 1);
      return 1;
    }
  }
  puts("PASS: SubWord, InvMix, AES-128/192/256, unaligned in-place "
       "encrypt/decrypt");
  return 0;
}
