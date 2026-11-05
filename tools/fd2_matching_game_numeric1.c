extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
int sub_13512(int unit)
{
    int offset = unit;
    unsigned int address = (unsigned int)dword_53A45;
    offset <<= 2;
    offset += unit;
    offset <<= 4;
    address += (unsigned int)offset;
    address += 5;
    *(unsigned char *)address |= 0x80;
    return offset;
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    unsigned int offset = (unsigned int)unit;
    unsigned int address = (unsigned int)dword_53A45;
    offset <<= 2;
    offset += (unsigned int)unit;
    offset <<= 4;
    address += offset;
    offset = (unsigned int)slot;
    offset <<= 1;
    offset += 11;
    address += offset;
    return *(unsigned char *)address;
}
#endif

#ifdef COPY_WORDS
unsigned sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; unit < dword_53BFB; ++unit) {
        unsigned int offset = (unsigned int)unit;
        unsigned int address = (unsigned int)dword_53BF7;
        offset <<= 2;
        offset += (unsigned int)unit;
        offset <<= 4;
        address += offset;
        *(unsigned char *)(address + 5) = 0;
        *(unsigned short *)(address + 64) = *(unsigned short *)(address + 66);
        *(unsigned short *)(address + 68) = *(unsigned short *)(address + 70);
    }
    return unit;
}
#endif
