#include <errno.h>
#include <stdio.h>
#include "in.h"

int test_structs(s4* s4, s8* s8, s12* s12, s16* s16, s20* s20) {
	errno = EACCES;
  printf("s4.a: %d\n", s4->a);
  printf("s8.a: %d, s8.b: %d\n", s8->a, s8->b);
  printf("s12.a: %d, s12.b: %d, s12.c: %d\n", s12->a, s12->b, s12->c);
  printf("s16.a: %d, s16.b: %d, s16.c: %d, s16.d: %d\n", s16->a, s16->b, s16->c, s16->d);
  printf("s20.a: %d, s20.b: %d, s20.c: %d, s20.d: %d, s20.e: %d\n", s20->a, s20->b, s20->c, s20->d, s20->e);

  return s4->a + s8->a + s8->b + s12->a + s12->b + s12->c + s16->a + s16->b + s16->c + s16->d + s20->a + s20->b + s20->c + s20->d + s20->e;
}
