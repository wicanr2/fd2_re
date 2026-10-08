/* 相鄰事件函式的完整 C 產碼候選。
 * 固定輸入及 IDA Pro 9.4 線性位址見 fd2_matching_full_20261008.json。
 * 原始語意入口：fd2_command10_12_presentation_ida.txt及
 * fd2_end_turn_command13_owner_ida.txt；本檔不重開既有玩法證據。
 * 宣告與區域名稱只作產碼導覽，不推定為作者原始碼。
 * 共享返回尾段可能在其他函式，完整區間未匹配時不增加覆蓋。
 * INDEXED_ALL及QUAKE_GROUP保留完整C函式並重現共用呼叫／返回尾段。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A49, *dword_53A6D;
extern int dword_53AA9, dword_53AAD, dword_53B13, dword_53EC4;
extern const double dbl_501F0;
extern void *malloc(unsigned);
extern void __cdecl free(void *);
extern void *memmove(void *, const void *, unsigned);
extern int __cdecl abs(int);
extern double sqrt(double);
extern void sub_25A96(int, int, int);
extern void sub_21548(int, int, int, const unsigned char *);
extern void sub_2189A(int, int, int);
extern void sub_219AD(int, int, int, int, int, int, const unsigned char *);
extern void sub_21B18(int, int, int, const unsigned char *);
extern void sub_21EB1(int, int);
extern void sub_11EEE(void *, int, int, int, int, int);
extern void sub_11EB0(void *, int, const void *, int, int, int);
extern void sub_127A9(void);
extern void sub_11CAC(int);
extern void sub_1DF58(void);
extern void sub_4DB9C(const unsigned char *, int, void *);
extern void sub_1C4CC(int, int, int, const unsigned char *);
extern void sub_1C2DA(int, int, int, const unsigned char *);
extern void sub_1CA89(int, int);
extern int sub_1C8ED(int, int);
extern void sub_1E0DB(int, int, int);
extern void sub_1CAC7(int, int, int, const unsigned char *);
extern int sub_1C75E(int, int);
extern void sub_1E1DC(int);

#if defined(INDEXED_PAIR) || defined(INDEXED_ALL)
void sub_2111A(int unit, int count, const unsigned char *targets, int mode)
{
    int index;
    const unsigned char *target;
    int value;
    sub_1C4CC(unit, mode, count, targets);
    sub_1CAC7(unit, mode, count, targets);
    index = 0;
    goto condition;
next:
    ++index;
condition:
    if (index >= count) goto done;
    value = sub_1C75E(*(target = targets + index), mode);
    if (value) goto show;
    sub_1E1DC(*target);
    goto next;
show:
    sub_1E0DB(value, 94, *target);
    goto next;
done:
    sub_11CAC(0);
    sub_1DF58();
}
#endif

#ifdef QUAKE_GROUP
void sub_21527(int unit, int count, const unsigned char *targets)
{
    sub_21548(unit, 10, count, targets);
}
#define F21548
#include "QUAKE.C"
#endif

#if defined(F2185F) || defined(QUAKE_GROUP)
void sub_2185F(int unit, int count, const unsigned char *targets)
{
    sub_25A96(dword_53B13, 2, 1);
    sub_2189A(unit, 15, 10);
    sub_21548(unit, 11, count, targets);
}
#endif

#ifdef F2189A
void sub_2189A(int unit, int radius, int step)
{
    struct { int x, y; } center;
    unsigned char *record;
    unsigned char *buffer;
    unsigned char *shape;
    int index;
    record = unit * 80 + dword_53A45;
    center.x = (record[0] - dword_53AA9) * 24 + 12;
    center.y = (record[1] - dword_53AAD) * 24 + 18;
    buffer = malloc(153216);
    sub_11EEE(buffer + 0x8088, 456, 13, 8, dword_53AA9, dword_53AAD);
    index = 0;
    goto condition;
next:
    shape = (unsigned char *)(*(unsigned *)(dword_53A6D + index * 4 + 6)
                              + (unsigned)dword_53A6D);
    memmove(dword_53A49, buffer, 153216);
    sub_219AD(center.x, center.y, radius, 12, 0, 192, shape);
    sub_127A9();
    sub_11EB0((void *)0xa0504, 320, dword_53A49 + 0x8088, 456, 312, 192);
    radius += step;
    ++index;
condition:
    if (index < 10) goto next;
    free(buffer);
    sub_11CAC(0);
}
#endif

#ifdef F219AD
void sub_219AD(int x, int y, int radius, int scale, int row, int end,
               const unsigned char *shape)
{
    volatile int total_span;
    int radial;
    int width;
    int delta;
    int span;
    int left;
    int right;
    goto condition;
positive:
    span = width;
clamp_right:
    right = width + x;
    if (right >= 312) right = 312 - x;
    else right = width;
    total_span = span + right;
    sub_4DB9C(shape, total_span, dword_53A49 + 0x8088 + row * 456 + left);
next:
    ++row;
condition:
    if (row >= end) return;
    if (row <= y - radius) goto next;
    if (row >= y + radius) goto next;
    delta = abs(y - row);
    radial = radius * radius - delta * delta;
    width = (int)(sqrt((double)radial) * scale / dbl_501F0);
    left = x - width;
    if (left >= 0) goto positive;
    left = 0;
    span = x;
    goto clamp_right;
}
#endif

#if defined(F21A9E) || defined(QUAKE_GROUP)
void sub_21A9E(int unit, int count, const unsigned char *targets)
{
    sub_25A96(dword_53B13, 2, 1);
    sub_2189A(unit, 30, 16);
    sub_21548(unit, 12, count, targets);
}
#endif

#if defined(F21AD9) || defined(INDEXED_ALL)
void sub_21AD9(int unit, int count, const unsigned char *targets)
{
    sub_25A96(dword_53B13, 11, 1);
    sub_21EB1(1, 2);
    sub_21B18(unit, 13, count, targets);
}
#endif

#if defined(F21B18) || defined(INDEXED_PAIR) || defined(INDEXED_ALL)
void sub_21B18(int unit, int mode, int count, const unsigned char *targets)
{
    int index;
    int value;
    dword_53EC4 = 0;
    sub_1C4CC(unit, mode, count, targets);
    sub_1C2DA(unit, mode, count, targets);
    sub_1CA89(unit, mode);
    index = 0;
    goto condition;
next:
    value = sub_1C8ED(targets[index], mode);
    sub_1E0DB(value, 105, targets[index]);
    ++index;
condition:
    if (index < count) goto next;
    sub_11CAC(0);
    sub_1DF58();
}
#endif

#if defined(F21B99) || defined(INDEXED_ALL)
void sub_21B99(int unit, int count, const unsigned char *targets)
{
    sub_25A96(dword_53B13, 11, 1);
    sub_21EB1(2, 4);
    sub_21B18(unit, 14, count, targets);
}
#endif
