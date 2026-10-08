/* 較完整的遊戲函式產碼候選。
 * 原始位址、呼叫端及位元組依 IDA 匯出；作者型別與變數名仍為未知。
 * 名稱保存原始定位與位移，不替玩法語意分級。
 */
extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BEB;
extern int dword_53AB1, dword_53AB5, dword_53AB9, dword_53ABD;
extern int dword_53AA9, dword_53AAD, dword_53AC1, dword_53AC5;
extern int dword_51A83, dword_51A87, dword_51A8B, dword_51A8F;
extern unsigned char *dword_53A49, *dword_53A4D, *dword_53A51;
extern unsigned char *dword_53A55, *dword_53A69;
extern void sub_11CAC(int value);
extern int sub_3453E(int unit);
extern unsigned char *__cdecl sub_4E56C(int index);
extern void __cdecl sub_4DEDA(const void *source, void *destination, unsigned size);
extern unsigned char sub_12E38(int x, int y, void *output);

#ifdef CURSOR_Y_DEC
void sub_11B48(void)
{
    if (!dword_53AB5) goto redraw;
    if (dword_53ABD < 2 && dword_53AAD) {
        --dword_53AB5;
        --dword_53AAD;
        goto redraw;
    }
    --dword_53AB5;
    --dword_53ABD;
    if (!dword_51A83) goto finished;
redraw:
    sub_11CAC(0);
finished:
    return;
}
#endif

#ifdef CURSOR_Y_INC
void sub_11B9B(void)
{
    if (dword_53AB5 == dword_53AC5 - 1) goto redraw;
    if (dword_53ABD > 5 && dword_53AAD != dword_53AC5 - 8) {
        ++dword_53AB5;
        ++dword_53AAD;
        goto redraw;
    }
    ++dword_53AB5;
    ++dword_53ABD;
    if (!dword_51A83) goto finished;
redraw:
    sub_11CAC(0);
finished:
    return;
}
#endif

#ifdef CURSOR_X_INC
void sub_11BFA(void)
{
    if (dword_53AB1 == dword_53AC1 - 1) goto redraw;
    if (dword_53AB9 > 10 && dword_53AA9 != dword_53AC1 - 13) {
        ++dword_53AB1;
        ++dword_53AA9;
        goto redraw;
    }
    ++dword_53AB1;
    ++dword_53AB9;
    if (!dword_51A83) goto finished;
redraw:
    sub_11CAC(0);
finished:
    return;
}
#endif

#ifdef CURSOR_X_DEC
void sub_11C59(void)
{
    if (!dword_53AB1) goto redraw;
    if (dword_53AB9 < 2 && dword_53AA9) {
        --dword_53AB1;
        --dword_53AA9;
        goto redraw;
    }
    --dword_53AB1;
    --dword_53AB9;
    if (!dword_51A83) goto finished;
redraw:
    sub_11CAC(0);
finished:
    return;
}
#endif

#ifdef FIND_RECORD
int sub_12C0D(void)
{
    unsigned char *record = dword_53A45;
    int unit = 0;
    int a, b;
    goto condition;
next:
    record += 80;
    ++unit;
condition:
    if (unit >= dword_53BEB) goto not_found;
    a = record[0];
    b = record[1];
    if (a != dword_53AB1) goto next;
    if (b != dword_53AB5) goto next;
    if (sub_3453E(unit)) goto next;
    return unit;
not_found:
    return -1;
}
#endif

#ifdef SUM_FIELD
int sub_15DA2(int count, const unsigned char *indices, int offset, int increment)
{
    int field_offset, step, length;
    int sum, index;
    int unit;
    int value;
    unsigned char *record;
    length = count;
    field_offset = offset;
    step = increment;
    sum = 0;
    index = 0;
    goto condition;
next:
    ++index;
condition:
    if (index >= length) goto finished;
    unit = indices[index];
    unit *= 80;
    record = dword_53A45;
    record += unit;
    value = record[field_offset];
    if (!value) sum += step;
    goto next;
finished:
    return sum;
}
#endif

#ifdef DERIVED_WORDS
void sub_1145A(int unit)
{
    unsigned char *record = dword_53BF7 + 80 * unit;
    volatile int values[4];
    int base;
    int slot;
    unsigned char *cell;
    unsigned char *data;
    values[1] = *(short *)(record + 0x37);
    values[2] = *(short *)(record + 0x39);
    base = *(short *)(record + 0x3e);
    values[3] = base;
    values[0] = base;
    slot = 0;
    goto condition;
next:
    ++slot;
condition:
    if (slot >= 8) goto finished;
    cell = record + 2 * slot + 10;
    if ((unsigned char)(cell[0] & 0x40) == 0) goto next;
    data = sub_4E56C(cell[1]);
    values[1] += *(short *)(data + 1);
    values[2] += *(short *)(data + 5);
    values[0] += *(short *)(data + 3);
    values[3] += *(short *)(data + 7);
    goto next;
finished:
    *(short *)(record + 0x48) = values[1];
    *(short *)(record + 0x4a) = values[2];
    *(short *)(record + 0x4c) = values[0];
    *(short *)(record + 0x4e) = values[3];
}
#endif

#ifdef BLIT_CELL
void sub_126F7(int x, int y, int index)
{
    int offsets[1];
    const unsigned char *source;
    if (x < dword_53AA9) goto finished;
    if (x >= dword_53AA9 + dword_51A87) goto finished;
    if (y < dword_53AAD) goto finished;
    if (y >= dword_53AAD + dword_51A8B) goto finished;
    offsets[0] = (y - dword_53AAD) * 10944 + (x - dword_53AA9) * 24;
    source = dword_53A4D;
    source += *(unsigned *)(source + 4 * index + 6);
    sub_4DEDA(source, dword_53A49 + offsets[0] + 0x8088, 456);
finished:
    return;
}
#endif

typedef struct {
    short word_00, word_02;
    unsigned char byte_04, byte_05, byte_06, byte_07;
} RawInfo;
typedef char InfoSizeMustBe8[sizeof(RawInfo) == 8 ? 1 : -1];

#ifdef READ_INFO
unsigned char sub_12E38(int x, int y, void *output)
{
    unsigned index = y * dword_53AC1;
    unsigned char *entry;
    RawInfo *info = output;
    short word, value;
    volatile int offset;
    unsigned char last;
    unsigned char *data;
    index += x;
    index <<= 2;
    entry = dword_53A51;
    entry += index;
    entry += 4;
    word = *(short *)entry & 0x3ff;
    value = entry[2] & 0x1f;
    info->word_00 = word;
    info->word_02 = value;
    offset = 4 * word;
    data = dword_53A69;
    data += offset;
    info->byte_04 = data[0];
    info->byte_05 = data[1];
    info->byte_06 = data[2];
    last = data[3];
    info->byte_07 = last;
    return last;
}
#endif

#ifdef CONDITIONAL_INFO
void sub_13A44(int x, int y, int value)
{
    RawInfo info;
    int index;
    unsigned char *entry;
    int a, b;
    sub_12E38(x, y, &info);
    if ((unsigned char)(info.byte_04 & 0x60) != 0) goto finished;
    index = (unsigned short)info.word_02;
    if (!index) goto finished;
    --index;
    index += index;
    entry = dword_53A55;
    entry += index;
    a = entry[0x33];
    b = entry[0x34];
    if (a == 255) goto finished;
    if (b != value) goto finished;
    dword_51A8F = a;
finished:
    return;
}
#endif
