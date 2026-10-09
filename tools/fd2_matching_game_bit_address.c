/* sub_13512完整36-byte C候選，固定IDA 9.4位址及原檔見匹配主收據。
 * 保留80-byte記錄、+5及bit7寫入；回傳表示只供產碼導航。
 * 原作者回傳宣告與未分級欄位語意仍未知。
 */
extern unsigned char * volatile dword_53A45;

#if defined(BIT_FORM_0) || defined(BIT_FORM_1) || defined(BIT_FORM_2) || defined(BIT_FORM_3) || defined(BIT_FORM_4) || defined(BIT_FORM_5)
#if defined(BIT_FORM_0) || defined(BIT_FORM_5)
void sub_13512(int unit)
#else
unsigned sub_13512(int unit)
#endif
{
#ifdef BIT_FORM_0
    unsigned offset = (unsigned)unit * 80;
    dword_53A45[offset + 5] |= 0x80;
#endif
#ifdef BIT_FORM_1
    unsigned offset = (unsigned)unit * 80;
    dword_53A45[offset + 5] |= 0x80;
    return offset;
#endif
#ifdef BIT_FORM_2
    unsigned offset = unit;
    offset *= 80;
    dword_53A45[offset + 5] |= 0x80;
    return offset;
#endif
#ifdef BIT_FORM_3
    typedef struct { unsigned char bytes[80]; } RawRecord;
    unsigned offset = (unsigned)unit * 80;
    RawRecord *record = (RawRecord *)dword_53A45;
    record[unit].bytes[5] |= 0x80;
    return offset;
#endif
#ifdef BIT_FORM_4
    unsigned offset = unit;
    offset *= 5;
    offset *= 16;
    dword_53A45[offset + 5] |= 0x80;
    return offset;
#endif
#ifdef BIT_FORM_5
    unsigned index = unit;
    unsigned offset = index << 2;
    offset += index;
    offset <<= 4;
    dword_53A45[offset + 5] |= 0x80;
#endif
}
#endif
