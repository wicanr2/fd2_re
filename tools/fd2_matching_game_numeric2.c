extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
int sub_13512(int unit)
{
    unsigned long base = (unsigned long)dword_53A45;
    unsigned long index = (unsigned long)(unsigned int)unit;
    index = (index << 2) + (unsigned int)unit;
    index <<= 4;
    *(unsigned char *)(base + index + 5UL) |= 0x80;
    return (int)index;
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    unsigned long base = (unsigned long)dword_53A45;
    unsigned long index = (unsigned long)(unsigned int)unit;
    index = (index << 2) + (unsigned int)unit;
    index <<= 4;
    base += index;
    index = (unsigned long)(unsigned int)slot;
    index = (index << 1) + 11UL;
    return *(unsigned char *)(base + index);
}
#endif

#ifdef COPY_WORDS
unsigned sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; unit < dword_53BFB; ++unit) {
        unsigned long base = (unsigned long)dword_53BF7;
        unsigned long index = (unsigned long)unit;
        index = (index << 2) + unit;
        index <<= 4;
        base += index;
        *(unsigned char *)(base + 5UL) = 0;
        *(unsigned short *)(base + 64UL) = *(unsigned short *)(base + 66UL);
        *(unsigned short *)(base + 68UL) = *(unsigned short *)(base + 70UL);
    }
    return unit;
}
#endif
