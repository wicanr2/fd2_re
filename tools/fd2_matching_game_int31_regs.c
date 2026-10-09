/* sub_36255完整C候選。只保留INT31h、0101h與原始REGS欄位寬度。
 * 未從常數推定服務高階語意或原作者型別，caller另存主收據。
 */
#pragma off (check_stack)
extern int int386(int,void *,void *);
#ifdef INT31_REGS_1
int sub_36255(unsigned first,unsigned second,volatile unsigned third)
#else
int sub_36255(unsigned first,unsigned second,unsigned third)
#endif
{
    unsigned words[14];
    words[0]=0x101;
#ifdef INT31_REGS_2
    {
        unsigned value=third;
        words[3]=value&0xffff;
    }
#else
    words[3]=third&0xffff;
#endif
    return int386(0x31,words,words+7);
}
