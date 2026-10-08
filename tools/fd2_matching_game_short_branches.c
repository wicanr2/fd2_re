/* 短分支及有界迴圈C候選。
 * 固定原檔、IDA Pro 9.4線性位址、原名與caller見匹配主收據。
 * 位元寬度、原始分支／資料流及完整RET皆比對，不推定作者型別。
 */
extern int dword_53ECC,dword_53EC4,dword_53A79,dword_53BEB,dword_53BEF;
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53AD5,*dword_53A55,*dword_54147;
extern void loc_205BE(void);
extern int sub_33499(int);
extern int sub_3453E(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_1CA89(int,int);
extern void sub_22AF6(int,int,int,int,int);
extern void sub_1DF58(void);
extern void sub_11DF2(int,int,int);
extern void sub_375B2(int);
extern void *memmove(void *,const void *,unsigned);
extern void sub_3419C(int,int,int);
extern void sub_35BBA(int);
extern void sub_1DB65(void);
extern int __cdecl outp(int,int);
extern void *sub_111BA(const char *,void *,int);
extern const char aFdotherDat[];
typedef struct { unsigned char bytes[6]; } RawSix;
typedef char RawSixMustBe6[sizeof(RawSix)==6?1:-1];
extern const RawSix unk_525D6;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F203BD
void sub_203BD(int first,int second,int third)
{
    int index=0;
    for(;index<256;++index){outp(968,index);outp(969,first);outp(969,second);outp(969,third);}
}
#endif
#ifdef F20872
void sub_20872(void)
{
    loc_205BE();
    if(sub_33499(18)!=0)return;
    if(sub_3453E(52)==0)return;
    MESSAGE(2);
    dword_53ECC=1;
}
#endif
#ifdef F20A51
void sub_20A51(void){loc_205BE();if(sub_3453E(16)||sub_3453E(17))dword_53ECC=1;}
#endif
#ifdef F20A87
void sub_20A87(void){loc_205BE();if(sub_3453E(1))dword_53ECC=1;}
#endif
#ifdef F20B14
void sub_20B14(void){loc_205BE();if(sub_3453E(16))dword_53ECC=1;}
#endif
#ifdef F20B3C
void sub_20B3C(void){loc_205BE();if(sub_3453E(1)||sub_3453E(2))dword_53ECC=1;}
#endif
#ifdef F22AA8
void sub_22AA8(int first,int second,int third,int fourth,int fifth)
{
    dword_53EC4=0;sub_1CA89(first,second);sub_22AF6(first,second,third,fourth,fifth);
    if(dword_53EC4!=0)sub_1DF58();
}
#endif
#ifdef F25052
void sub_25052(int index,int delay)
{
    for(;index>=0;--index){sub_11DF2(0,255,index);sub_375B2(delay);}
}
#endif
#ifdef F2BC9A
void *sub_2BC9A(const unsigned char *input)
{
    void *result=0;
    RawSix table=unk_525D6;
    if(input&&input[4])result=sub_111BA(aFdotherDat,result,table.bytes[input[4]-1]);
    return result;
}
#endif
#ifdef F2D620
void sub_2D620(unsigned char *dest,int stride,int index)
{
    unsigned char *source=dword_54147;
    int row;
    source+=*(unsigned *)(source+14)+4;
    source+=index*6;
    for(row=0;row<9;++row){memmove(dest,source,6);dest+=stride;source+=6;}
}
#endif
#ifdef F34594
void sub_34594(void)
{
    int state=*(volatile unsigned char *)(dword_53AD5+16);
    if(state!=0)return;
    sub_3419C(24,27,7);MESSAGE(3);dword_53AD5[16]=1;
}
#endif
#ifdef F347D9
void sub_347D9(void){if(sub_3453E(8)==0)MESSAGE(2);}
#endif
#ifdef F3499B
void sub_3499B(int unit)
{
    int offset=unit*80;
    int state=*(volatile unsigned char *)(offset+dword_53A45+6);
    if(state!=0){sub_3419C(9,27,0);dword_53AD5[16]=1;}
}
#endif
#ifdef F34A0E
void sub_34A0E(void)
{
    int index,offset;
    for(index=0;index<18;++index){offset=(index+10)*80;(dword_53A45+offset)[52]&=128;}
}
#endif
#ifdef F34E3B
void sub_34E3B(void)
{
    int state=*(volatile unsigned char *)(dword_53AD5+16);
    if(state!=0)return;
    sub_3419C(16,71,state);MESSAGE(1);dword_53AD5[16]=1;
}
#endif
#ifdef F35641
void sub_35641(int unit){if((dword_53A45+unit*80)[6])sub_3419C(39,44,0);}
#endif
#ifdef F35675
void sub_35675(int unit)
{
    if((dword_53A45+unit*80)[6]){sub_3419C(23,24,0);sub_3419C(53,56,0);}
}
#endif
#ifdef F35898
void sub_35898(void)
{
    unsigned char value;
    if(dword_53AD5[17]!=0)return;
    value=*(volatile unsigned char *)&dword_53BEF;
    ++value;dword_53A55[3]=value;dword_53AD5[17]=1;
}
#endif
#ifdef F3599B
void sub_3599B(void)
{
    if(dword_53AD5[16]!=0)return;
    dword_53A55[3]=*(volatile unsigned char *)&dword_53BEF;
    dword_53AD5[16]=1;
}
#endif
#ifdef F35B6B
void sub_35B6B(void)
{
    if(dword_53AD5[19]){MESSAGE(2);sub_35BBA(20);}
    ++dword_53AD5[19];
}
#endif
#ifdef F35BBA
void sub_35BBA(int unit)
{
    for(;unit<dword_53BEB;++unit){*(unsigned short *)(dword_53A45+unit*80+64)=0;}
    sub_1DB65();
}
#endif
