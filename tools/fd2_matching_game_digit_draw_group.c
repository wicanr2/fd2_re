/* 三個相鄰數字函式的完整C編譯單元候選。
 * 固定IDA 9.4原始位址、雜湊及caller見匹配主收據；宣告只作產碼導航。
 */
#ifdef DIGIT_DRAW_GROUP
#define DOWN_FORM_2
#include "DOWN.C"
#undef DOWN_FORM_2
extern unsigned char *dword_54147;
extern void *memmove(void *,const void *,unsigned);
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];

void sub_2D620(void *dest, int stride, volatile int index)
{
    unsigned source = (unsigned)dword_54147;
    int row;
    source += *(unsigned *)(source + 14);
    source += 4;
    row = index * 6;
    source += row;
    row = 0;
    goto condition;
next:
    memmove(dest, (const void *)source, 6);
    dest = (unsigned char *)dest + stride;
    source += 6;
    ++row;
condition:
    if (row < 9) goto next;
}
#endif
