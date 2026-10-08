/* 主程式區的匹配來源候選，原始定位與用途分類依 IDA 匯出。
 * 只描述直接指令中的 raw 存取，不把候選來源當成原作者宣告。
 * 回傳型別只描述原始 EAX 結果，不推定原作者的回傳宣告。
 */
extern unsigned char *dword_53A45;
extern int dword_53BEB;
extern int dword_53AC1;
extern unsigned char *dword_53A51;
extern void sub_375B2(int value);
extern int sub_3453E(int index);
extern void sub_127E0(int index);
extern void sub_129EC(void);
extern void sub_11D40(int a, int b, int c);
extern void *memmove(void *destination, const void *source, unsigned size);
extern void sub_4E92C(void *a, void *b, unsigned size);
extern void __cdecl free(void *pointer);
extern unsigned char byte_53AF9;
extern void *dword_53B0F;
extern void *dword_53B13;
extern const char aFdotherDat[];
extern void *sub_111BA(const char *name, int mode, int resource);
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
int sub_13512(int unit)
{
    int index = unit;
    int offset = index << 2;
    offset += index;
    offset <<= 4;
    dword_53A45[offset + 5] |= 0x80;
    return offset;
}
#endif

#ifdef CLEAR_BIT7
void sub_13536(void)
{
    int unit;
    for (unit = 0; unit < dword_53BEB; ++unit) {
        dword_53A45[80 * unit + 5] &= 0x7f;
    }
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    unsigned char *record;
    unsigned char *cell;
    int offset = unit;
    offset *= 80;
    record = dword_53A45;
    record += offset;
    cell = record + 2 * slot;
    cell += 11;
    return *cell;
}
#endif

#ifdef SET_BYTE5
int sub_32975(int unit)
{
    int offset = 80 * unit;
    dword_53A45[offset + 5] = 1;
    return offset;
}
#endif

#ifdef GET_BIT0
int sub_3453E(int unit)
{
    unsigned char value = dword_53A45[80 * unit + 5];
    value &= 1;
    return value;
}
#endif

#ifdef RANGE_LOW_BYTE
void sub_3419C(int first, int last, int value)
{
    int unit;
    for (unit = first; unit <= last; ++unit) {
        unsigned char *record = dword_53A45 + 80 * unit;
        record[52] = (record[52] & 0xf0) | (unsigned char)value;
    }
}
#endif

#ifdef COPY_WORDS
unsigned sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; unit < dword_53BFB; ++unit) {
        unsigned char *record = dword_53BF7 + 80 * unit;
        unsigned short value;
        record[5] = 0;
        value = *(unsigned short *)(record + 66);
        *(unsigned short *)(record + 64) = value;
        value = *(unsigned short *)(record + 70);
        *(unsigned short *)(record + 68) = value;
    }
    return unit;
}
#endif

#ifdef COMPARE_VALUE
int sub_2EF8F(int a, int b)
{
    if (a == b) return 31;
    if (a < b) return 42;
    return 119;
}
#endif

#ifdef CLEAR3
void sub_134E4(void)
{
    unsigned char *record = dword_53A45;
    int index;
    for (index = 0; index < dword_53BEB; ++index) {
        record[3] = 0;
        record += 80;
    }
    sub_375B2(20);
}
#endif

#ifdef CACHE_INIT
void sub_1D4CB(void)
{
    dword_53B13 = 0;
    dword_53B13 = sub_111BA(aFdotherDat, 0, 80);
}
#endif

#ifdef CONDITIONAL_INIT
void sub_1A7BD(void)
{
    if (byte_53AF9) {
        dword_53B0F = 0;
        dword_53B0F = sub_111BA(aFdotherDat, 0, 64);
    }
}
#endif

#ifdef ROWCOPY
void sub_11EB0(unsigned char *destination, int destination_step,
               const unsigned char *source, int source_step,
               unsigned size, int rows)
{
    int row = 0;
    goto condition;
next:
    memmove(destination, source, size);
    destination += destination_step;
    source += source_step;
    ++row;
condition:
    if (row >= rows) goto finished;
    goto next;
finished:
    return;
}
#endif

#ifdef INVENTORY
void *sub_1B8E7(int unit, int slot)
{
    unsigned char *record = dword_53A45 + 80 * unit;
    void *result = memmove(record + 2 * slot + 10,
                          record + 2 * slot + 12, 2 * (7 - slot));
    record[24] = 0x80;
    return result;
}
#endif

#ifdef RELEASE
void sub_15E71(void *a, void *b, unsigned size)
{
    sub_4E92C(a, b, size);
    free(a);
}
#endif

#ifdef CONDITIONAL_FREE
void __cdecl sub_1A7F1(void)
{
    if (byte_53AF9) free(dword_53B0F);
}
#endif

#ifdef REDRAW
void sub_127A9(void)
{
    int index = 0;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) goto finished;
    if (sub_3453E(index)) goto next;
    sub_127E0(index);
    goto next;
finished:
    sub_129EC();
}
#endif

#ifdef CYCLE
void sub_1F525(void)
{
    int value;
    for (value = 64; value >= 0; --value) {
        sub_11D40(0, 255, value);
        sub_375B2(2);
    }
}
#endif

#ifdef MASK
void sub_146A7(int x, int y)
{
    int index = y * dword_53AC1;
    unsigned char *entry;
    index += x;
    index <<= 2;
    entry = dword_53A51;
    entry += index;
    entry += 6;
    *entry |= 0x80;
}
#endif
