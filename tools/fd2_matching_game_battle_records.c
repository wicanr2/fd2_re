/* 戰鬥記錄與共用返回尾段的純 C 候選。
 * 保留原始名稱、欄位位移及呼叫目標。作者宣告仍未知。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned dword_53A51, dword_53A71;
extern unsigned char *dword_53A49;
extern int dword_53BEB, dword_53AC1, dword_53AC5, dword_51A83;
extern int dword_53A10, dword_53A14, dword_53EEC, dword_53AB9, dword_53ABD;
extern unsigned char byte_51AAC;
extern void sub_1A30B(void);
extern void sub_146A7(int, int);
extern void sub_16559(int);
extern void sub_25A96(int, int, int);
extern void sub_17AA9(int);
extern int sub_3453E(int);
extern int sub_1B722(int, int);
extern int sub_1BB8C(int, int);
extern void __cdecl free(void *);
extern void *malloc(unsigned);
extern void *memmove(void *, const void *, unsigned);
typedef struct { unsigned char byte_00, byte_01; } RawPair;
typedef char RawPairSizeMustBe2[sizeof(RawPair) == 2 ? 1 : -1];

#ifdef F13565
void sub_13565(void)
{
    int value = 1;
    int index = 0;
    unsigned char *record;
    int state;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) goto done;
    record = index * 80 + dword_53A45;
    if ((unsigned char)(record[5] & 0x81)) goto next;
    if (record[6] != 2) goto next;
    state = *(volatile unsigned char *)(record + 0x26);
    if (state != 0) goto next;
    value = 0;
    goto next;
done:
    if (value != 1) return;
    byte_51AAC = 0;
    dword_51A83 = 0;
    sub_1A30B();
    dword_51A83 = 1;
    byte_51AAC = 1;
}
#endif

#ifdef F14625
void sub_14625(int x, int y)
{
    unsigned char *cell;
    unsigned offset;
    if (x) sub_146A7(x - 1, y);
    if (y) sub_146A7(x, y - 1);
    if (x < dword_53AC1 - 1) sub_146A7(x + 1, y);
    if (y < dword_53AC5 - 1) sub_146A7(x, y + 1);
    offset = (x + y * dword_53AC1) * 4;
    cell = (unsigned char *)(offset + dword_53A51);
    cell += 6;
    *cell |= 0x40;
}
#endif

#ifdef F14B16
int sub_14B16(RawPair *output)
{
    int count = 0;
    unsigned char *cell = (unsigned char *)(dword_53A51 + 7);
    volatile int y = 0;
    int x;
    goto outer_condition;
outer_next:
    ++y;
outer_condition:
    x = y;
    if (x >= dword_53AC5) goto done;
    x = 0;
    goto inner_condition;
inner_next:
    cell += 4;
    ++x;
inner_condition:
    if (x >= dword_53AC1) goto outer_next;
    if (*cell == 255) goto inner_next;
    output->byte_00 = x;
    output->byte_01 = y;
    ++output;
    ++count;
    goto inner_next;
done:
    return count;
}
#endif

#ifdef F164E8
void sub_164E8(void)
{
    int index;
    ++dword_53A14;
    if (dword_53A14 == 2) {
        ++dword_53A10;
        if (dword_53A10 == 4) dword_53A10 = 0;
        index = dword_53A10;
        if (index == 3) index = 1;
        sub_16559(index);
        dword_53A14 = 0;
    }
    sub_25A96(dword_53EEC, 2, 1);
    sub_17AA9(1);
}
#endif

#ifdef F175A9
void sub_175A9(void)
{
    const unsigned char *source;
    int row;
    if (dword_53A71) free((void *)dword_53A71);
    dword_53A71 = (unsigned)malloc(5184);
    source = dword_53A49 + 0x8088 + (dword_53AB9 - 1) * 24
           + (dword_53ABD - 1) * 10944;
    row = 0;
    goto condition;
next:
    memmove((void *)(dword_53A71 + (row * 9) * 8), source, 72);
    source += 456;
    ++row;
condition:
    if (row < 72) goto next;
}
#endif

#ifdef COUNTS_GROUP
int sub_1B5F1(int value)
{
    int total = 0;
    int index = 0;
    unsigned char *record;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) return total;
    record = index * 80 + dword_53A45;
    if (record[6] != value) goto next;
    if (record[7] == 121) goto next;
    if (record[0x1f] == 10) goto next;
    if (sub_3453E(index)) goto next;
    ++total;
    goto next;
}

int sub_1B653(unsigned char *output)
{
    int total = 0;
    int index = 0;
    unsigned char *record;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) return total;
    record = index * 80 + dword_53A45;
    if ((unsigned char)(record[5] & 1)) goto next;
    if (record[0x31] != 3) goto next;
    if ((int)*(unsigned short *)(record + 0x40) > 0) goto next;
    memmove(output + total * 3, record + 0x31, 3);
    ++total;
    goto next;
}

int sub_1B6B7(unsigned char *output)
{
    int total = 0;
    int index = 0;
    unsigned char *record;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) return total;
    record = index * 80 + dword_53A45;
    if ((unsigned char)(record[5] & 1)) goto next;
    if (record[0x31] == 255) goto next;
    if ((int)*(unsigned short *)(record + 0x40) > 0) goto next;
    memmove(output + total * 3, record + 0x31, 3);
    ++total;
    goto next;
}
#endif

#ifdef F1B83D
int sub_1B83D(int unit, int category)
{
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    unsigned char *cell;
    goto condition;
next:
    ++index;
condition:
    if (index >= 8) goto not_found;
    cell = record + index * 2;
    cell += 10;
    if ((unsigned char)(*cell & 0x40) == 0) goto next;
    if (category == 0 && cell[1] < 128) goto found;
    if (category == 0) goto next;
    if (cell[1] < 128) goto next;
found:
    return index;
not_found:
    return -1;
}
#endif

#ifdef F1B8A6
int sub_1B8A6(int unit)
{
    int total = 0;
    unsigned char *record = unit * 80 + dword_53A45;
    int index = 0;
    unsigned cell;
    goto condition;
next:
    ++index;
condition:
    if (index >= 8) return total;
    cell = (unsigned)record + index * 2;
    cell += 10;
    if ((unsigned char)(*(unsigned char *)cell & 0x80)) goto next;
    ++total;
    goto next;
}
#endif

#ifdef F1C142
void sub_1C142(int unit, int slot)
{
    unsigned char *record = unit * 80 + dword_53A45;
    int value = sub_1B722(unit, slot);
    int index = 0;
    unsigned char *cell;
    goto condition;
next:
    ++index;
condition:
    if (index >= 8) goto done;
    cell = record + index * 2;
    cell += 10;
    if ((unsigned char)(*cell & 0x40) == 0) goto next;
    if (value < 128 && cell[1] < 128) goto clear;
    if (value < 128) goto next;
    if (cell[1] < 128) goto next;
clear:
    *cell = 0;
    goto next;
done:
    record[slot * 2 + 10] = 0x40;
}
#endif

#ifdef F1C220
void sub_1C220(int item)
{
    int index = 0;
    goto condition;
next:
    ++index;
condition:
    if (index >= dword_53BEB) return;
    if ((index * 80 + dword_53A45)[6] != 2) goto next;
    if (sub_1BB8C(index, item) == -1) goto next;
}
#endif
