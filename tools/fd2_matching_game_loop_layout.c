/* 有界迴圈的完整C候選。固定原檔、IDA Pro 9.4原名、caller及bytes見主收據。
 * 明示原始初始閘門及位址計算順序；型別與名稱只作產碼導航。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_54147;
extern void sub_11DF2(int,int,int),sub_375B2(int);
extern void *memmove(void *,const void *,unsigned);

#ifdef F25052
void sub_25052(int initial,int duration)
{
    int delay=duration;
    int index=initial;
    goto condition;
next:
    sub_11DF2(0,255,index);sub_375B2(delay);--index;
condition:
    if(index>=0)goto next;
}
#endif

#ifdef F2D620
void sub_2D620(unsigned char *dest,int stride,int index)
{
    unsigned char *source=dword_54147;
    int row;
    source+=*(unsigned *)(source+14);source+=4;source+=index*6;
    row=0;goto condition;
next:
    memmove(dest,source,6);dest+=stride;source+=6;++row;
condition:
    if(row<9)goto next;
}
#endif

#ifdef F34A0E
void sub_34A0E(void)
{
    int index,offset;
    index=0;goto condition;
next:
    offset=(index+10)*80;(dword_53A45+offset)[52]&=128;++index;
condition:
    if(index<18)goto next;
}
#endif
