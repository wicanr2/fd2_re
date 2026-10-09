/* sub_1B8A6完整65-byte C候選；原始定位及caller見固定IDA 9.4匹配主收據。
 * 保留80-byte記錄、八槽、2-byte步距、+10及bit7測試。
 * 局部型別只供產碼導航，未匹配不提升語意或作者宣告。
 */
extern unsigned char * volatile dword_53A45;

#if defined(COUNT_FORM_0) || defined(COUNT_FORM_1) || defined(COUNT_FORM_2) || defined(COUNT_FORM_3) || defined(COUNT_FORM_4) || defined(COUNT_FORM_5) || defined(COUNT_FORM_6) || defined(COUNT_FORM_7)
int sub_1B8A6(int unit)
{
    int total = 0;
#ifdef COUNT_FORM_0
    unsigned cell;
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
#elif defined(COUNT_FORM_1)
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    unsigned char *cell;
#elif defined(COUNT_FORM_2)
    unsigned char *record = unit * 80 + dword_53A45;
    unsigned cell;
    int index = 0;
#elif defined(COUNT_FORM_3)
    unsigned char *record = unit * 80 + dword_53A45;
    unsigned cell;
    int index = unit;
    index = 0;
#elif defined(COUNT_FORM_4)
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    int cell;
#elif defined(COUNT_FORM_5)
    unsigned char *record = unit * 80 + dword_53A45;
    unsigned index = 0;
    unsigned cell;
#elif defined(COUNT_FORM_6)
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    int cell;
#else
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    unsigned cell;
#endif
    goto condition;
next:
    ++index;
condition:
    if ((int)index >= 8) return total;
#ifdef COUNT_FORM_1
    cell = index * 2 + record;
    cell += 10;
    if ((unsigned char)(*cell & 0x80)) goto next;
#elif defined(COUNT_FORM_4)
    cell = index * 2;
    cell += (int)record;
    cell += 10;
    if ((unsigned char)(*(unsigned char *)cell & 0x80)) goto next;
#elif defined(COUNT_FORM_6)
    cell = index;
    cell += index;
    cell += (int)record;
    cell += 10;
    if ((unsigned char)(*(unsigned char *)cell & 0x80)) goto next;
#elif defined(COUNT_FORM_7)
    cell = index * 2;
    cell += (unsigned)record;
    cell += 10;
    if ((unsigned char)(*(unsigned char *)cell & 0x80)) goto next;
#else
    cell = (unsigned)record + index * 2;
    cell += 10;
    if ((unsigned char)(*(unsigned char *)cell & 0x80)) goto next;
#endif
    ++total;
    goto next;
}
#endif
