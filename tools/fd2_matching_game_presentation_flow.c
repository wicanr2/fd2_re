/* 三個完整呈現函式C候選；固定原檔、IDA 9.4原始位址與caller見匹配主收據。
 * 保留全域讀取、byte寫入、呼叫順序與返回邊，不提升玩法或硬體時序語意。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A85, *dword_53AD5, *dword_53A55;
extern int dword_53C67, dword_53BEF, dword_53BEB, dword_53A79;
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];
extern int abs(int);
extern void sub_4E8AF(void *, const void *, unsigned), sub_4E8E1(void *, const void *, unsigned);
extern void sub_13512(int), sub_10B4E(int), sub_35E5A(void), sub_375B2(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F1F04A
void sub_1F04A(int first, int second)
{
    unsigned char *a = (unsigned char *)((unsigned)first * 80);
    unsigned char *base, *b;
    int dx, dy;
    base = dword_53A45;
    a += (unsigned)base;
    b = (unsigned char *)((unsigned)second * 80);
    b += (unsigned)base;
    dx = abs((int)a[0] - (int)b[0]);
    dy = abs((int)a[1] - (int)b[1]);
    if (dx <= dy) goto vertical;
    if ((int)a[0] <= (int)b[0]) goto left;
    a[3] = 1;
    return;
left:
    a[3] = 3;
    return;
vertical:
    if ((int)a[1] <= (int)b[1]) goto up;
    a[3] = 2;
    return;
up:
    a[3] = 0;
}
#endif

#ifdef F16559
void sub_16559(int index)
{
    unsigned destination = 0xa0000;
    unsigned offset;
    const unsigned char *base, *source;
    destination += *(volatile unsigned *)&dword_53C67;
    offset = *(volatile int *)&index;
    base = dword_53A85;
    offset = ((const unsigned *)base)[offset];
    source = base + offset;
    if (dword_53C67 == 0x9017) goto alternate;
    sub_4E8AF((void *)destination, source, 320);
    return;
alternate:
    sub_4E8E1((void *)destination, source, 320);
}
#endif

#ifdef F35D60
void sub_35D60(void)
{
    int state = *(volatile unsigned char *)(dword_53AD5 + 17);
    volatile unsigned char index;
    if (state == 4) goto sequence;
    sub_13512(1);
    ++dword_53AD5[17];
    dword_53A55[6] = (unsigned char)((unsigned)dword_53BEF + 1);
    return;
sequence:
    MESSAGE(2);
    sub_10B4E(1);
    dword_53AD5[21] = (unsigned char)((unsigned)dword_53BEB - 3);
    dword_53A55[9] = *(volatile unsigned char *)&dword_53BEF;
    sub_35E5A();
    sub_375B2(400);
    sub_35E5A();
    sub_375B2(400);
    index = 3;
    goto check;
next:
    sub_35E5A();
    MESSAGE(state);
    ++index;
check:
    state = index;
    if (state <= 6) goto next;
}
#endif
