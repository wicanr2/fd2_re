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
