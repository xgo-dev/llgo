#pragma once

typedef struct {
	int a;
} s4;

typedef struct {
  int a;
  int b;
} s8;

typedef struct {
  int a;
  int b;
  int c;
} s12;

typedef struct {
  int a;
  int b;
  int c;
  int d;
} s16;

typedef struct {
  int a;
  int b;
  int c;
  int d;
  int e;
} s20;

extern int test_structs(s4* s4, s8* s8, s12* s12, s16* s16, s20* s20);
