/* FIGANI 計數推進的完整 C 候選。
 * 固定 FD2.EXE、IDA 9.4 線性位址 0x2B9A1、caller 與原始 bytes
 * 見 docs/data/ida/fd2_matching_full_20261008.json。
 * 名稱保留原始定位，局部宣告只供產碼導航，不推定作者型別。
 */
extern unsigned char byte_540FC, byte_540FD;
extern void sub_2935B(unsigned char *, int, int, int, int);

#if defined(IDLE_FORM_0) || defined(IDLE_FORM_1) || defined(IDLE_FORM_2) || defined(IDLE_FORM_3) || defined(IDLE_FORM_4) || defined(IDLE_FORM_5)
void sub_2B9A1(unsigned char *data, int active, int x, int y)
{
    int offset, count;
    if (!active) {
        byte_540FC = 0;
        byte_540FD = 0;
        return;
    }
    sub_2935B(data, byte_540FD, x, y, active);
    offset = *(int *)(data + byte_540FD * 4 + 8);
    count = data[offset + 6];
    ++byte_540FC;
    if ((int)byte_540FC >= count) {
        byte_540FC = 0;
        ++byte_540FD;
#ifdef IDLE_FORM_0
        offset = byte_540FD;
        count = data[0];
        if (offset >= count) byte_540FD = 0;
#endif
#ifdef IDLE_FORM_1
        if ((int)byte_540FD >= (int)data[0]) byte_540FD = 0;
#endif
#ifdef IDLE_FORM_2
        if ((int)data[0] <= (int)byte_540FD) byte_540FD = 0;
#endif
#ifdef IDLE_FORM_3
        count = byte_540FD;
        offset = data[0];
        if (count >= offset) byte_540FD = 0;
#endif
#ifdef IDLE_FORM_4
        offset = *(volatile unsigned char *)&byte_540FD;
        count = data[0];
        if (offset >= count) byte_540FD = 0;
#endif
#ifdef IDLE_FORM_5
        offset = byte_540FD;
        count = *(volatile unsigned char *)data;
        if (offset >= count) byte_540FD = 0;
#endif
    }
}
#endif
