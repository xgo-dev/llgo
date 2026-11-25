// Freestanding so layout checks need Clang's target support, not target SDKs.
typedef __SIZE_TYPE__ AuditSizeT;
typedef __PTRDIFF_TYPE__ AuditSsizeT;
typedef __INTPTR_TYPE__ AuditIntptrT;
typedef __UINTPTR_TYPE__ AuditUintptrT;
typedef __WCHAR_TYPE__ AuditWcharT;
typedef _Complex float AuditComplex64;
typedef _Complex double AuditComplex128;
typedef enum { AuditZero, AuditOne } AuditEnum;
typedef struct { char C; double D; char Tail; } AuditRecord;

#define AUDIT_TYPES(X) \
 X(Bool, _Bool) X(Char, char) X(Uchar, unsigned char) \
 X(Short, short) X(Ushort, unsigned short) \
 X(Int, int) X(Uint, unsigned int) \
 X(Long, long) X(Ulong, unsigned long) \
 X(LongLong, long long) X(UlongLong, unsigned long long) \
 X(Float, float) X(Double, double) X(Complex64, AuditComplex64) X(Complex128, AuditComplex128) \
 X(Pointer, void*) X(SizeT, AuditSizeT) X(SsizeT, AuditSsizeT) \
 X(IntptrT, AuditIntptrT) X(UintptrT, AuditUintptrT) \
 X(WcharT, AuditWcharT) X(Enum, AuditEnum) X(Record, AuditRecord)
