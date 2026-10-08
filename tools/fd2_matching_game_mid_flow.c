/* 中型完整函式C候選。固定原檔及IDA Pro 9.4線性位址見匹配主收據。
 * 原始名稱、caller、資料寬度、具名參照及全部返回路徑保留。
 * 宣告只作產碼導覽，未匹配不提升覆蓋、作者型別或欄位用途。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53AD5,*dword_53A55,*dword_53A49,*dword_53A65,*dword_53A81,*dword_53A85;
extern unsigned char *dword_53AFF,*dword_53AD1,*dword_54147;
extern unsigned char *dword_53C5B,*dword_53C5F,*dword_53C63;
extern int dword_53C57,dword_53C67,dword_53C03,dword_53BEF,dword_53BEB,dword_51A83;
extern int dword_53A79,dword_53EC8,dword_51CF9,dword_51CFD,dword_53AA9,dword_53AAD;
extern int dword_540FF,dword_53EE4,dword_53EE8;
extern void *dword_53B13;
extern unsigned char byte_53EF1,byte_51E62,byte_51A10;
extern const unsigned char byte_51AAD[],byte_51AD1[],byte_51AF5[];
extern const char aFdotherDat[],aDatoDat[];
extern const double dbl_501F8,dbl_50200;
extern double cos(double),sin(double);
extern int abs(int),sub_1B83D(int,int),sub_1B722(int,int),sub_1B9DE(int,int),sub_33499(int);
extern unsigned char *sub_4E56C(int);
extern int sub_3453E(int),sub_4E893(void);
extern void sub_17E0B(int),sub_18409(int,void *,void *,void *);
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void sub_205DA(void),sub_134E4(void),sub_4E031(void),sub_11506(void);
extern void sub_135DD(int,int),sub_1366A(int),sub_12D7B(int),sub_11CAC(int),sub_112A5(int),sub_10B4E(int);
extern void sub_35822(int,int,int),sub_35BBA(int),sub_17AA9(int),sub_375B2(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_394B5(int),sub_391D1(int),sub_39344(int,void *,int),sub_3975E(int,int),sub_39448(int);
extern int __cdecl outp(int,int);
extern void sub_1CA89(int,int),sub_22253(int,int,int,int,int),sub_12CEA(int,int);
extern void *sub_111BA(const char *,void *,int);
extern void sub_168B6(void *,int,int,int,int,int),sub_4E8AF(void *,void *,int),sub_17FC0(int,void *);
extern void sub_25A96(void *,int,int),sub_4E85B(void *,void *,int),sub_127A9(void);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_4E63D(void *,int,int,void *,int,int);
extern void sub_1F882(void),sub_1F525(void),sub_1974C(int,void *,void *);
extern void sub_2DC55(int,int,int,void *,unsigned char);
extern void sub_1956B(int),sub_16559(int),sub_16C57(int),sub_196CB(void),sub_13512(int),sub_35E5A(void);
typedef struct {unsigned char bytes[9];} RawNine;
typedef char RawNineMustBe9[sizeof(RawNine)==9?1:-1];
extern const RawNine unk_520E4,unk_520ED,unk_520F6;
extern void sub_233C6(void *,void *,void *,int,int,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F1F0DC
int sub_1F0DC(int unit,int other)
{
    int offset=unit*80,other_offset,first,second,slot,kind;
    unsigned char *base=dword_53A45,*one=base+offset,*two;
    other_offset=other*80;two=base+other_offset;
    if((int)two[38])return -1;
    first=(int)one[0]-(int)two[0];first=abs(first);
    second=(int)one[1]-(int)two[1];second=abs(second);
    if(first+second!=1)return -1;
    slot=sub_1B83D(other,0);if(slot==-1)return slot;
    kind=sub_4E56C(sub_1B722(other,slot))[11];
    if(kind!=1)return -1;
    return kind;
}
#endif
#ifdef F1B932
int sub_1B932(int unit,int value)
{
    int result,index;
    sub_17E0B(unit);dword_53C57=0;
    do{result=sub_1B9DE(unit,value);}while(!result);
    for(index=0;index<=11;++index)sub_18409(index,dword_53C5B,dword_53C63,dword_53C5F);
    memmove((void *)0xa0000,dword_53C5F,64000);
    free(dword_53C5B);free(dword_53C5F);free(dword_53C63);
    return result!=-1;
}
#endif
#ifdef F3327D
void sub_3327D(void)
{
    int index,offset;
    sub_205DA();
    for(index=0;index<11;++index){offset=index*80;(dword_53A45+offset)[3]=2;}
    sub_135DD(6,0);MESSAGE(0);dword_51A83=0;sub_1366A(35);MESSAGE(1);sub_12D7B(0);sub_134E4();
}
#endif
#if defined(F25A96) || defined(F25B45)
#ifdef F25A96
#define HANDLE dword_53EE4
void sub_25A96(void *input,int index,int loops)
#else
#define HANDLE dword_53EE8
void sub_25B45(void *input,int index,int loops)
#endif
{
    unsigned char *data=input;
    void * volatile sample;
    volatile int length;
    if(!byte_53EF1||!byte_51E62||dword_540FF)return;
    sub_394B5(HANDLE);if(index==-1)return;
    sample=data+*(int *)(data+index*4+6);
    length=*(int *)(data+index*4+10)-*(int *)(data+index*4+6);
    sub_391D1(HANDLE);sub_39344(HANDLE,sample,length);sub_3975E(HANDLE,loops);sub_39448(HANDLE);
}
#undef HANDLE
#endif
#ifdef F358EA
void sub_358EA(void)
{
    int state=*(volatile unsigned char *)(dword_53AD5+16);
    if(state==1){
        sub_15F84(dword_53A79,state,(void *)0xa0000,320,205,76,74,19,state);
        sub_35822(9,44,3);sub_35822(0,9,4);sub_35822(17,9,5);dword_51A83=1;
    }else if(state==2){MESSAGE(state);sub_35BBA(16);}
    ++dword_53AD5[16];
}
#endif
#ifdef F35112
void sub_35112(void)
{
    sub_10B4E(dword_53BEF/2);sub_17AA9(1);
    sub_135DD(0,0);sub_17AA9(8);sub_135DD(28,0);sub_17AA9(8);
    sub_135DD(28,32);sub_17AA9(8);sub_135DD(0,32);sub_17AA9(8);
    if(dword_53BEF==2)MESSAGE(3);
}
#endif
#ifdef F286BD
void sub_286BD(int first,int limit,int step,unsigned char r,unsigned char g,unsigned char b)
{
    int index,offset,denominator=40;
    for(index=first;index<limit;++index){
        outp(968,index);offset=index*3;
        outp(969,((int)dword_53A65[offset]-(int)r)*step/denominator+r);
        outp(969,((int)dword_53A65[offset+1]-(int)g)*step/denominator+g);
        outp(969,((int)dword_53A65[offset+2]-(int)b)*step/denominator+b);
    }
}
#endif
#ifdef F334D9
void sub_334D9(void)
{
    unsigned char byte_index;
    int index;
    sub_205DA();byte_index=(unsigned char)sub_33499(12);byte_index^=1;byte_index*=3;
    MESSAGE(index=byte_index);sub_135DD(24,17);MESSAGE(index+1);dword_51A83=0;
    sub_1366A(48);index+=2;MESSAGE(index);sub_12D7B(0);
}
#endif
#ifdef F2218A
void sub_2218A(int actor,int unused,unsigned char *target)
{
    int offset,value,kind;
    unsigned char *record;
    sub_12D7B(target[0]);sub_1CA89(actor,23);
    offset=(int)target[0]*80;record=dword_53A45+offset;
    value=record[33];kind=record[32];if(kind>8&&kind<25)value+=30;
    dword_53EC8+=value*10;
    sub_22253(target[0],255,255,record[0],record[1]);dword_51A83=0;
    sub_12CEA(dword_51CF9,dword_51CFD);
    sub_22253(target[0],dword_51CF9,dword_51CFD,dword_51CF9,dword_51CFD);dword_51A83=1;
}
#endif
#ifdef F24D22
void sub_24D22(int rows)
{
    unsigned char *temporary;
    int index;
    if(rows){byte_51A10=(unsigned char)rows;return;}
    temporary=malloc((int)byte_51A10*312);
    memmove(temporary,dword_53AFF+(192-(int)byte_51A10)*312,(int)byte_51A10*312);
    for(index=191-(int)byte_51A10;index>=0;--index){
        memmove(dword_53AFF+index*312+(int)byte_51A10*312,dword_53AFF+index*312,312);
    }
    memmove(dword_53AFF,temporary,(int)byte_51A10*312);free(temporary);
}
#endif
#ifdef F17EEF
void sub_17EEF(int unit,unsigned char *dest)
{
    unsigned char *source;
    int offset;
    dword_53C67=3208;offset=unit*80;
    dword_53A85=sub_111BA(aDatoDat,dword_53A85,(dword_53A45+offset)[7]);
    source=dword_53A85+(int)dword_53A85[0];
    sub_168B6(dest,320,5,7,5,5);sub_4E8AF(dest+dword_53C67,source,320);
    source=dword_53A81;source+=*(int *)(source+86);sub_4E8AF(dest+2332,source,320);
    source=dword_53A81;source+=*(int *)(source+90);sub_4E8AF(dest+30085,source,320);
    sub_17FC0(unit,dest);
}
#endif
#ifdef F1D6C8
void sub_1D6C8(int selector)
{
    int index;
    sub_25A96(dword_53B13,0,1);
    for(index=0;index<4;++index){
        outp(968,0);outp(969,byte_51AAD[selector]);outp(969,byte_51AD1[selector]);outp(969,byte_51AF5[selector]);
        sub_17AA9(1);outp(968,0);outp(969,0);outp(969,0);outp(969,0);sub_17AA9(1);
    }
}
#endif
#ifdef F22470
void sub_22470(int x,int y,unsigned char *source)
{
    int index,horizontal,vertical;
    unsigned char *entry,*base,*dest;
    for(index=0;index<11;++index){
        base=dword_53AD1;entry=base+*(int *)(base+(index+114)*4+6);
        memmove(dword_53A49,source,153216);
        horizontal=(x-dword_53AA9)*24;dest=dword_53A49+32904+horizontal;
        vertical=(y-dword_53AAD)*10944;
        sub_4E85B(dest+vertical+456,entry,456);sub_127A9();
        sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192);sub_17AA9(1);
    }
}
#endif
#ifdef F341DB
void sub_341DB(void)
{
    sub_112A5(1);sub_10B4E(3);sub_135DD(5,8);sub_11CAC(1);sub_375B2(100);
    sub_1366A(7);sub_4E031();MESSAGE(11);dword_51A83=0;
    sub_10B4E(7);sub_11CAC(1);sub_375B2(100);sub_1366A(8);sub_4E031();MESSAGE(3);sub_134E4();
}
#endif
#ifdef F232E8
void sub_232E8(void)
{
    RawNine first=unk_520E4,second=unk_520ED,third=unk_520F6;
    int result;
    sub_11506();
    if((int)*(volatile unsigned char *)(dword_53AD5+17)==1){
        result=sub_3453E(43);
        if(!result){sub_233C6(first.bytes,second.bytes,third.bytes,result,8,43,12,7,2,6,2);MESSAGE(4);sub_112A5(12);goto done;}
    }
    MESSAGE(5);
done:
    ++dword_53C03;
}
#endif
#ifdef F2E0BD
void sub_2E0BD(int first,int second,unsigned char mode)
{
    unsigned char *base;
    int index;
    dword_53C5B=malloc(64000);dword_53C5F=malloc(64000);dword_53C63=malloc(64000);
    memmove(dword_53C5F,(void *)0xa0000,64000);memmove(dword_53C63,dword_53C5F,64000);
    base=dword_54147;base+=*(int *)(base+70);sub_4E8AF(dword_53C63+35845,base,320);
    sub_2DC55(first,second,dword_53C57,dword_53C63,mode);
    for(index=5;index>=0;--index)sub_1974C(index*13+112,dword_53C5B,dword_53C63);
}
#endif
#ifdef F1F73F
void sub_1F73F(int image,int palette,unsigned char *buffer,int row)
{
    void *source;
    sub_1F882();memset((void *)0xa0000,0,64000);
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,palette);
    source=sub_111BA(aFdotherDat,0,image);sub_4E63D(source,0,0,(void *)0xa0000,320,-1);
    sub_1F525();sub_17AA9(1);sub_17AA9(6);sub_1F882();
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,101);
    sub_11EB0((void *)0xa0000,320,buffer+row*320,320,320,200);sub_1F525();
}
#endif
#ifdef F34A7A
void sub_34A7A(void)
{
    unsigned char *record,value;
    int index,offset;
    for(index=12;index<34;++index){offset=index*80;(dword_53A45+offset)[52]=0;}
    value=*(volatile unsigned char *)&dword_53BEF;++value;dword_53A55[3]=value;
    value=*(volatile unsigned char *)&dword_53BEF;value+=2;dword_53A55[6]=value;
    record=dword_53A45+880;
    record[5]=0;record[6]=1;record[7]=6;record[8]=6;record[49]=255;record[52]=128;*(unsigned short *)(record+64)=1;
    MESSAGE(2);sub_10B4E(1);MESSAGE(3);dword_53EC8=0;dword_53AD5[16]=2;
}
#endif
#ifdef F35C79
void sub_35C79(int unit)
{
    unsigned char *record;
    int kind;
    unsigned char value;
    unit*=80;record=dword_53A45+unit;
    if(!record[6]||dword_53AD5[17])return;
    kind=*(volatile unsigned char *)(record+8);
    if(kind!=9){
        sub_1956B(record[7]);sub_15F84(dword_53A79,0,(void *)0xa951f,320,205,76,74,19,1);
        sub_16559(0);sub_16C57(0);sub_196CB();
    }else{
        MESSAGE(1);dword_53AD5[17]=1;
        value=*(volatile unsigned char *)&dword_53BEF;++value;dword_53A55[6]=value;
        dword_53AD5[16]=4;dword_53A55[3]=*(volatile unsigned char *)&dword_53BEF;
    }
}
#endif
#ifdef F35D60
void sub_35D60(void)
{
    int state=*(volatile unsigned char *)(dword_53AD5+17);
    unsigned char value,index;
    if(state!=4){
        sub_13512(1);++dword_53AD5[17];
        value=*(volatile unsigned char *)&dword_53BEF;++value;dword_53A55[6]=value;return;
    }
    MESSAGE(2);sub_10B4E(1);value=*(volatile unsigned char *)&dword_53BEB;value-=3;dword_53AD5[21]=value;
    dword_53A55[9]=*(volatile unsigned char *)&dword_53BEF;
    sub_35E5A();sub_375B2(400);sub_35E5A();sub_375B2(400);
    for(index=3;index<=6;++index){sub_35E5A();MESSAGE(index);}
}
#endif
#ifdef F21DB2
void sub_21DB2(int amount,unsigned char index,unsigned short *xs,unsigned short *ys,unsigned char *values,int x,int y)
{
    int radius=sub_4E893()%64*amount/64-1;
    int angle=sub_4E893()%360;
    double radians=(double)angle*dbl_501F8;
    double scaled=(double)radius;
    xs[index]=(unsigned short)(int)(cos(radians)*scaled+(double)x);
    ys[index]=(unsigned short)(int)(sin(radians)*scaled+(double)y+dbl_50200);
    values[index]=(unsigned char)(sub_4E893()%8+1);
}
#endif
