/* sub_1B722完整46-byte候選，沿用已匹配13512的5乘16表示線索。
 * 固定IDA 9.4原始位址／雜湊見匹配主收據；未匹配不提高覆蓋。
 */
extern unsigned char * volatile dword_53A45;
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];

#if defined(SLOT_SCALE_0) || defined(SLOT_SCALE_1)
unsigned sub_1B722(int unit, int slot)
{
    unsigned offset = unit;
    unsigned char *record;
    unsigned char *cell;
    offset *= 5;
    offset *= 16;
    record = dword_53A45;
    record += offset;
#ifdef SLOT_SCALE_0
    cell = record + 2 * slot;
#else
    cell = (unsigned char *)(2 * slot);
    cell += (unsigned)record;
#endif
    cell += 11;
    return *cell;
}
#endif
