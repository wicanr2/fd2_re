/* 主程式區的匹配來源候選，原始定位與用途分類依 IDA 匯出。
 * 只描述直接指令中的 raw 存取，不把候選來源當成原作者宣告。
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
