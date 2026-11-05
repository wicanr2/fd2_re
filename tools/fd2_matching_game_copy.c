/* 畫面複製及八格寫入的純 C 產碼候選。
 * 保留原始函式名與位移；作者的型別及變數宣告仍未知。
 * 只核對原始位元組，不提升其他 caller 的玩法或硬體語意。
 */
extern unsigned char *dword_53A45, *dword_53A49, *dword_53A71, *dword_53A85;
extern int dword_53AB9, dword_53ABD, dword_53C67;
extern void *memmove(void *, const void *, unsigned);
extern void sub_4E8AF(void *, const void *, unsigned);
extern void sub_4E8E1(void *, const void *, unsigned);

#ifdef VIDEO_COPY
void sub_16559(int index)
{
    int destination = 0xa0000;
    unsigned offset;
    const unsigned char *source;
    destination += dword_53C67;
    offset = ((unsigned *)dword_53A85)[index];
    source = dword_53A85 + offset;
    if (dword_53C67 != 0x9017) {
        sub_4E8AF((void *)destination, source, 320);
    } else {
        sub_4E8E1((void *)destination, source, 320);
    }
}
#endif

#ifdef RESTORE_72
void sub_17643(void)
{
    unsigned char *destination;
    int row;
    destination = dword_53A49 + 0x8088 + (dword_53AB9 - 1) * 24
                + (dword_53ABD - 1) * 10944;
    row = 0;
    goto condition;
next:
    memmove(destination, dword_53A71 + (row * 9) * 8, 72);
    destination += 456;
    ++row;
condition:
    if (row < 72) goto next;
}
#endif

#ifdef HCLIP_86
void sub_182AD(int position, unsigned char *output, const unsigned char *input)
{
    int source_offset;
    unsigned width;
    int row;
    width = 86;
    source_offset = 0;
    if (position < 0) {
        width += position;
        source_offset = -position;
        position = 0;
    }
    row = 0;
    goto condition;
next:
    memmove(output + 0x8c0 + position + row * 320,
            input + 0x8c5 + source_offset + row * 320, width);
    ++row;
condition:
    if (row < 86) goto next;
}
#endif

#ifdef VCLIP_86
void sub_18312(int position, unsigned char *output, const unsigned char *input)
{
    int rows = 86;
    int source_skip = 0;
    int row;
    if (position < 0) {
        rows += position;
        source_skip = -position;
        position = 0;
    }
    row = 0;
    goto condition;
next:
    memmove(output + 0x5c + position * 320 + row * 320,
            input + 0x91c + source_skip * 320 + row * 320, 223);
    ++row;
condition:
    if (row < rows) goto next;
}
#endif

#ifdef BOTTOM_102
void sub_1839B(int position, unsigned char *output, const unsigned char *input)
{
    int rows = 102;
    int row;
    if (position + rows >= 200) rows = 200 - position;
    row = 0;
    goto condition;
next:
    memmove(output + 5 + position * 320 + row * 320,
            input + 0x7585 + row * 320, 310);
    ++row;
condition:
    if (row < rows) goto next;
}
#endif

#ifdef PANEL_COPY
void sub_1AF99(const unsigned char *input, int position)
{
    int width = 170;
    int offset;
    int shift;
    int row;
    if (position > 4) {
        offset = 75;
    } else {
        shift = (4 - position) * 50;
        offset = shift + 75;
        if (shift + 245 > 320) width = 320 - offset;
    }
    row = 0;
    input += 0x2e8b;
    offset += 0x2e40;
    goto condition;
next:
    memmove(dword_53A49 + offset, input, width);
    input += 320;
    offset += 320;
    ++row;
condition:
    if (row < 117) goto next;
}
#endif

#ifdef INSERT_CELL
int sub_1BB8C(int unit, unsigned char value)
{
    unsigned char *record = dword_53A45 + 80 * unit;
    unsigned char *cell;
    int slot = 0;
    goto condition;
next:
    ++slot;
condition:
    if (slot >= 8) goto not_found;
    cell = record + 2 * slot + 10;
    if ((unsigned char)(cell[0] & 0x80) == 0) goto next;
    cell[0] = 0;
    cell[1] = value;
    return 1;
not_found:
    return -1;
}
#endif
