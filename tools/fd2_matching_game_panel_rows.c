/* 0x1B0AD 完整158-byte原始區間的C候選。
 * 原始IDA 9.4函式名、caller及bytes綁定匹配主收據的固定FD2.EXE。
 * raw常數、memmove方向與每列步距保持；局部型別不宣稱原作者宣告。
 */
extern unsigned char *dword_53A49;
extern void *memmove(void *, const void *, unsigned);

#if defined(PANEL_FORM_0) || defined(PANEL_FORM_1) || defined(PANEL_FORM_2) || defined(PANEL_FORM_3)
void sub_1B0AD(unsigned char *dest, int phase)
{
#ifdef PANEL_FORM_0
    int count = 16, start, offset, index;
#else
    int count = 16, start, index, offset;
#endif
    if (phase < 5) return;
    if (phase > 9) start = 155;
    else {
        int delta = (4 - (phase - 5)) * 9;
        start = delta + 155;
        if (delta + 171 > 200) count = start - 200;
    }
#if defined(PANEL_FORM_0)
    dest += 49675;
    offset = start * 320 + 75;
#elif defined(PANEL_FORM_1)
    index = start;
    index *= 320;
    dest += 49675;
    offset = index + 75;
#elif defined(PANEL_FORM_2)
    start *= 320;
    dest += 49675;
    offset = start + 75;
#else
    offset = start;
    offset *= 320;
    dest += 49675;
    offset += 75;
#endif
    for (index = 0; index < count; ++index) {
        memmove(dword_53A49 + offset, dest, 170);
        dest += 320;
        offset += 320;
    }
}
#endif
