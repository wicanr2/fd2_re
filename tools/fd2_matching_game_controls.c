/* 主程式控制與事件呼叫的純 C 產碼候選。
 * 原始名稱、呼叫位址與欄位位移依固定 IDA 9.4 匯出。
 * 名稱只供導航；原作者宣告及未分級玩法語意仍未知。
 */
extern unsigned char * volatile dword_53A45;
extern int dword_53A0C, dword_53A2C, dword_53C57;
extern int dword_53ECC, dword_53BEF;
extern void __cdecl sub_4E9BB(int, const unsigned char *, int);
extern void __cdecl sub_4E63D(const unsigned char *, int, int, int, int, int);
extern void sub_187D6(int, int, int, int, int);
extern void sub_17D6F(int, int, int, int);
extern const unsigned char *__cdecl sub_4E516(int);
extern void sub_1C916(int, int);
extern void loc_205BE(void);
extern int sub_3453E(int);
extern void sub_21227(int, int, int, int);
extern void sub_213B7(int, int, int, int);
extern void sub_21548(int, int, int, int);

#ifdef F10620
int sub_10620(void)
{
    int result = 0;
    volatile int first = *(volatile short *)0x41a;
    if (*(volatile short *)0x41c != first) result = 1;
    return result;
}
#endif

#ifdef F13460
void sub_13460(void)
{
again:
    if (*(volatile short *)0x46c == dword_53A0C) goto again;
    dword_53A0C = *(volatile short *)0x46c;
}
#endif

#ifdef F1685C
void sub_1685C(int first, int second, const unsigned char *base, int index)
{
    base += ((const unsigned *)(base + 6))[index];
    sub_4E9BB(first, base, second);
}
#endif

#ifdef F16886
void sub_16886(int first, int second, const unsigned char *base, int index)
{
    base += ((const unsigned *)(base + 6))[index];
    sub_4E63D(base, 0, 0, first, second, -1);
}
#endif

#ifdef F173E7
void sub_173E7(const int *values)
{
    dword_53C57 = 0;
    goto condition;
next:
    ++dword_53C57;
condition:
    if (dword_53C57 >= 4) return;
    if (values[dword_53C57] != 0) goto next;
}
#endif

#ifdef F17AA9
void sub_17AA9(int limit)
{
    int difference;
    dword_53A2C = *(volatile short *)0x46c;
again:
    difference = *(volatile short *)0x46c - dword_53A2C;
    if (difference < 0) difference += 0x10000;
    if (difference < limit) goto again;
    dword_53A2C = *(volatile short *)0x46c;
}
#endif

#ifdef F1875D
void sub_1875D(int first, int second, int third, int other, int fifth)
{
    int value = 42;
    if (third == other) value = 31;
    sub_187D6(first, second, third, value, fifth);
}
#endif

#ifdef F18795
void sub_18795(int first, int second, int third, int current, int maximum)
{
    int width;
    if (maximum == 0) return;
    if (current != 0) width = current * 101 / maximum + 1;
    else width = 0;
    sub_17D6F(first, second, width, third);
}
#endif

#ifdef F1C8ED
void sub_1C8ED(int first, int index)
{
    const unsigned char *data = sub_4E516(index);
    sub_1C916(first, *(const short *)data);
}
#endif

#ifdef F1CA89
void sub_1CA89(int unit, int index)
{
    const unsigned char *data = sub_4E516(index);
    unsigned char *record = dword_53A45 + unit * 80;
    int value = *(unsigned short *)(record + 0x44);
    value -= data[5];
    *(unsigned short *)(record + 0x44) = value;
}
#endif

#ifdef F1F183
int sub_1F183(int unit)
{
    int offset = unit << 2;
    const unsigned char *record;
    int value;
    offset += unit;
    offset <<= 4;
    record = (const unsigned char *)(unsigned)offset;
    record += (unsigned)dword_53A45;
    if (record[7] == 28) goto no;
    if (record[0x20] == 19) goto yes;
    value = record[0x1f];
    if (value == 4) goto yes;
    if (value != 5) goto no;
yes:
    return 1;
no:
    return 0;
}
#endif

#ifdef F206C5
void sub_206C5(void)
{
    int index;
    loc_205BE();
    index = 5;
    goto condition;
next:
    ++index;
condition:
    if (index >= 11) goto all;
    if ((unsigned char)((index * 80 + dword_53A45)[5] & 1) == 0) return;
    goto next;
all:
    dword_53ECC = 1;
}
#endif

#ifdef F20707
void sub_20707(void)
{
    loc_205BE();
    if (sub_3453E(50) || sub_3453E(51)) dword_53ECC = 1;
}
#endif

#ifdef F2073D
void sub_2073D(void)
{
    loc_205BE();
    if (sub_3453E(14)) dword_53ECC = 1;
}
#endif

#ifdef F20822
void sub_20822(void)
{
    loc_205BE();
    if (sub_3453E(64)) dword_53ECC = 1;
}
#endif

#ifdef F2084A
void sub_2084A(void)
{
    loc_205BE();
    if (sub_3453E(65)) dword_53ECC = 1;
}
#endif

#ifdef F20926
void sub_20926(void)
{
    loc_205BE();
    if (dword_53BEF > 6 && sub_3453E(64)) dword_53ECC = 1;
}
#endif

#ifdef F21206
void sub_21206(int first, int second, int third)
{
    sub_21227(first, 0, second, third);
}
#endif

#ifdef F21396
void sub_21396(int first, int second, int third)
{
    sub_213B7(first, 4, second, third);
}
#endif

#ifdef F21527
void sub_21527(int first, int second, int third)
{
    sub_21548(first, 10, second, third);
}
#endif
