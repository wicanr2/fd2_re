extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
int sub_13512(int unit)
{
    int index = (int)(unsigned int)unit;
    unsigned int base = (unsigned int)(unsigned long)dword_53A45;
    int scaled = (int)((unsigned int)index << 2);
    scaled += (int)(unsigned int)unit;
    scaled = (int)((unsigned int)scaled << 4);
    *(unsigned char *)(base + (unsigned int)scaled + 5U) |= 0x80;
    return scaled;
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    unsigned int index = (unsigned int)(int)unit;
    unsigned int base = (unsigned int)(unsigned long)dword_53A45;
    index = (index << 2) + (unsigned int)(int)unit;
    index <<= 4;
    base += index;
    index = (unsigned int)(int)slot;
    index = (index << 1) + 11U;
    return *(unsigned char *)(base + index);
}
#endif

#ifdef COPY_WORDS
unsigned sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; (unsigned int)unit < (unsigned int)dword_53BFB; ++unit) {
        unsigned int index = (unsigned int)unit;
        unsigned int base = (unsigned int)(unsigned long)dword_53BF7;
        index = (index << 2) + (unsigned int)unit;
        index <<= 4;
        base += index;
        *(unsigned char *)(base + 5U) = 0;
        *(unsigned short *)(base + 64U) = *(unsigned short *)(base + 66U);
        *(unsigned short *)(base + 68U) = *(unsigned short *)(base + 70U);
    }
    return unit;
}
#endif
