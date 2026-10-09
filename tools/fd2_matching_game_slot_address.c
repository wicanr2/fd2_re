/* sub_1B722，IDA 9.4線性位址0x1B722..0x1B750的完整C候選。
 * 固定原檔雜湊、caller及原始指令見匹配主收據。
 * 只供歷史-mf的32-bit模型產碼；整數保存地址的候選不宣稱portable C。
 * 原始名稱、80-byte步距、2-byte槽步距、+11及byte讀取保持。
 */
extern unsigned char * volatile dword_53A45;
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];

#if defined(SLOT_FORM_0) || defined(SLOT_FORM_1) || defined(SLOT_FORM_2) || defined(SLOT_FORM_3) || defined(SLOT_FORM_4) || defined(SLOT_FORM_5) || defined(SLOT_FORM_6) || defined(SLOT_FORM_7)
unsigned sub_1B722(int unit, int slot)
{
#ifdef SLOT_FORM_0
    unsigned offset = unit;
    offset *= 80;
    offset += (unsigned)dword_53A45;
    offset = slot * 2 + offset;
    offset += 11;
    return *(unsigned char *)offset;
#endif
#ifdef SLOT_FORM_1
    int offset = unit;
    offset *= 80;
    offset += (int)dword_53A45;
    slot *= 2;
    slot += offset;
    slot += 11;
    return *(unsigned char *)slot;
#endif
#ifdef SLOT_FORM_2
    unit *= 80;
    unit += (int)dword_53A45;
    slot *= 2;
    slot += unit;
    slot += 11;
    return *(unsigned char *)slot;
#endif
#ifdef SLOT_FORM_3
    typedef struct { unsigned char bytes[80]; } RawRecord;
    RawRecord *record = (RawRecord *)dword_53A45;
    unsigned char *cell = record[unit].bytes;
    cell += 2 * slot;
    cell += 11;
    return *cell;
#endif
#ifdef SLOT_FORM_4
    unsigned char *record;
    unsigned char *cell;
    int offset = unit;
    offset *= 80;
    record = offset + dword_53A45;
    cell = 2 * slot + record;
    cell += 11;
    return *cell;
#endif
#ifdef SLOT_FORM_5
    unsigned char *record;
    unsigned offset = unit;
    offset *= 80;
    record = (unsigned char *)(offset + (unsigned)dword_53A45);
    offset = slot * 2;
    offset += (unsigned)record;
    offset += 11;
    return *(unsigned char *)offset;
#endif
#ifdef SLOT_FORM_6
    unsigned char *record;
    unsigned char *cell;
    int offset = unit;
    offset *= 80;
    record = (unsigned char *)offset;
    record += (unsigned)dword_53A45;
    cell = record + 2 * slot;
    cell += 11;
    return *cell;
#endif
#ifdef SLOT_FORM_7
    unsigned char *record = (unsigned char *)(unit * 80);
    record += (unsigned)dword_53A45;
    return record[2 * slot + 11];
#endif
}
#endif
