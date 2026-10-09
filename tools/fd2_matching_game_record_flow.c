/* 三個完整記錄查詢C候選，固定IDA 9.4原始位址及雜湊見匹配主收據。
 * 保留原始讀取寬度、步距、全域寫入及返回路徑，未匹配不提高覆蓋。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7, *dword_53C1B;
extern int dword_53BEB, dword_53BFB;
extern int sub_3453E(int), sub_1B83D(int,int), sub_1B722(int,int), abs(int);
extern unsigned char *sub_4E56C(int);

#ifdef F1F183
int sub_1F183(int unit)
{
    unsigned offset = unit;
    const unsigned char *record;
    int value;
    offset *= 5;
    offset *= 16;
    record = dword_53A45;
    record += offset;
    if (record[7] == 28) goto no;
    if (record[32] == 19) goto yes;
    value = record[31];
    if (value == 4) goto yes;
    if (value != 5) goto no;
yes:
    return 1;
no:
    return 0;
}
#endif

#ifdef F12C60
int sub_12C60(int value)
{
    unsigned char *record = dword_53A45;
    int index;
    dword_53C1B = 0;
    index = 0;
    goto first_check;
first_next:
    record += 80;
    ++index;
first_check:
    if (index >= dword_53BEB) goto first_done;
    if ((int)record[8] != value) goto first_next;
    dword_53C1B = record;
    if (sub_3453E(index)) goto first_next;
    return index;
first_done:
    if (dword_53C1B) goto unavailable;
    record = dword_53BF7;
    index = 0;
    goto second_check;
second_next:
    record += 80;
    ++index;
second_check:
    if (index >= dword_53BFB) goto unavailable;
    if ((int)record[8] != value) goto second_next;
    dword_53C1B = record;
    goto second_next;
unavailable:
    return -1;
}
#endif

#ifdef F1DEBE
int sub_1DEBE(int unit, int x, int y)
{
    unsigned offset = unit;
    unsigned char *record;
    int first, second, slot;
    offset *= 5;
    offset *= 16;
    record = dword_53A45;
    record += offset;
    if ((int)record[38] == 0) goto adjacent;
unavailable:
    return -1;
adjacent:
    first = x - (int)record[0];
    first = abs(first);
    second = y - (int)record[1];
    second = abs(second);
    if (first + second != 1) goto unavailable;
    slot = sub_1B83D(unit, 0);
    if (slot == -1) return slot;
    if ((int)sub_4E56C(sub_1B722(unit, slot))[11] > 1) goto unavailable;
    return 1;
}
#endif
