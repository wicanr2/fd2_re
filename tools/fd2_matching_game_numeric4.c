extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
int sub_13512(int unit)
{
    int address = (int)dword_53A45;
    int index = unit;
    index = (index << 2) + unit;
    index <<= 4;
    address += index;
    *(unsigned char *)(address + 5) |= 0x80;
    return index;
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    int address = (int)dword_53A45;
    int index = unit;
    index = (index << 2) + unit;
    index <<= 4;
    address += index;
    index = slot;
    index += index;
    address += index;
    address += 11;
    return *(unsigned char *)address;
}
#endif

#ifdef COPY_WORDS
unsigned sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; unit < dword_53BFB; ++unit) {
        int address = (int)dword_53BF7;
        int index = unit;
        index = (index << 2) + unit;
        index <<= 4;
        address += index;
        *(unsigned char *)(address + 5) = 0;
        *(unsigned short *)(address + 64) = *(unsigned short *)(address + 66);
        *(unsigned short *)(address + 68) = *(unsigned short *)(address + 70);
    }
    return unit;
}
#endif
