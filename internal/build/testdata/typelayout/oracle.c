#include "types.h"
#define DEFINE(Name, T) \
 T value_##Name; \
 T array_##Name[2]; \
 const unsigned long long size_##Name = sizeof(T); \
 const unsigned long long align_##Name = _Alignof(T);
AUDIT_TYPES(DEFINE)
DEFINE(LongDouble, long double)
#define COUNT(Name, T) + 1
const unsigned long long audit_type_count = 0 AUDIT_TYPES(COUNT);
const unsigned long long offset_Record_D = __builtin_offsetof(AuditRecord, D);
const unsigned long long offset_Record_Tail = __builtin_offsetof(AuditRecord, Tail);
