/* 完整搜尋、條件呼叫及畫面收尾C候選。
 * 固定FD2.EXE、IDA Pro 9.4線性位址、原始指令及caller見匹配主收據。
 * 宣告只用於還原產碼。未匹配候選及未證實用途不提升覆蓋或語意。
 */
extern int dword_53BFB,dword_53ECC,dword_53AA9,dword_53AAD,dword_53A79,dword_53A7D;
extern int dword_53C0F,dword_53C0B,dword_53C07,dword_53C57,dword_53C67,dword_5413F,dword_53AE1;
extern unsigned char * volatile dword_53BF7,* volatile dword_53A45;
extern unsigned char *dword_53AD5,*dword_53A55,*dword_53A49,*dword_53A6D,*dword_53A81,*dword_53A85;
extern unsigned char *dword_53C5B,*dword_53C5F,*dword_53C63;
extern int dword_53BEF;
extern unsigned char byte_52659,byte_53AFA,byte_540FC,byte_540FD;
extern const char aDatoDat[];
extern int sub_31860(int,int),sub_1B8A6(int),sub_1B722(int,int),sub_3453E(int),sub_1B83D(int,int);
extern void sub_1BB8C(int,int),sub_3419C(int,int,int),sub_35822(int,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_4DFCC(void),sub_4E031(void);
extern unsigned char sub_10620(void);
extern void sub_11DF2(int,int,int),sub_375B2(int),sub_11CAC(int),sub_1297D(void);
extern void sub_135DD(int,int),sub_32999(int),sub_1366A(int),sub_134E4(void),sub_10B4E(int);
extern void sub_1956B(int),sub_16559(int),sub_16C57(int),sub_2D31B(void);
extern int abs(int);
extern void sub_4E63D(void *,int,int,int,int,int),sub_187D6(int,int,int,int,int);
extern int sub_2E6B8(void);
extern void sub_17AED(int),sub_16E24(void),sub_17AA9(int);
extern void *sub_111BA(const char *,void *,int);
extern void *memmove(void *,const void *,unsigned);
extern void free(void *);
extern void sub_1974C(int,void *,void *),sub_11EEE(void *,int,int,int,int,int);
extern void sub_122DC(void),sub_127A9(void),sub_1ACF3(void *,int);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_22046(int,int,int,int,int,void *);
extern void sub_2935B(void *,int,int,int,int);
extern unsigned char *sub_4E56C(int);
extern int sub_4E893(void);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)
#define BIOS_WORD (*(volatile short *)0x46c)

#ifdef F24B14
int sub_24B14(int value)
{
    int index;
    index=0;goto check;
advance:
    ++index;
check:
    if(index>=16)return -1;
    if(sub_31860(index,value)==-1)goto advance;
    return 1;
}
#endif
#if defined(F24BDE) || defined(F33499)
#ifdef F24BDE
int sub_24BDE(int value)
#else
int sub_33499(int value)
#endif
{
    int index,offset;
    index=0;goto check;
advance:
    ++index;
check:
    if(index>=dword_53BFB)return 0;
    offset=index*80;
    if((int)(dword_53BF7+offset)[8]!=value)goto advance;
    return 1;
}
#endif
#ifdef F309FF
int sub_309FF(unsigned char *output)
{
    unsigned char count=0;
    int index;
    index=0;goto check;
advance:
    ++index;
check:
    if(index>=dword_53BFB)return count;
    if(sub_3453E(index)!=1)goto advance;
    output[count]=(unsigned char)index;++count;goto advance;
}
#endif
#ifdef F31860
int sub_31860(int unit,int value)
{
    int count=sub_1B8A6(unit),index;
    if(!count)goto unavailable;
    index=0;goto check;
advance:
    ++index;
check:
    if(index>=count)goto unavailable;
    if(sub_1B722(unit,index)!=value)goto advance;
    return index;
unavailable:
    return -1;
}
#endif
#ifdef F35AB8
void sub_35AB8(int unit)
{
    unit*=80;
    if(!(dword_53A45+unit)[6])return;
    if(dword_53AD5[17])return;
    if(!dword_53AD5[18])return;
    dword_53A55[9]=*(volatile unsigned char *)&dword_53BEF;
    dword_53AD5[17]=1;
}
#endif
#ifdef F1E5C0
void sub_1E5C0(int limit)
{
    int initial=BIOS_WORD;
    unsigned char stop;
    do{
        sub_4DFCC();stop=sub_10620();
        if(BIOS_WORD-initial>=limit||BIOS_WORD<initial)stop=1;
    }while(!stop);
    sub_4E031();
}
#endif
#ifdef F208CF
void sub_208CF(void)
{
    if(sub_3453E(0)||sub_3453E(16)||sub_3453E(17))dword_53ECC=1;
    if(sub_3453E(52))dword_53ECC=2;
}
#endif
#ifdef F35E5A
void sub_35E5A(void)
{
    int index;
    for(index=0;index<64;++index){sub_11DF2(0,255,index);sub_375B2(8);}
    sub_375B2(400);
    for(index=62;index>=0;--index){sub_11DF2(0,255,index);sub_375B2(8);}
}
#endif
#ifdef F342B5
void sub_342B5(void)
{
    sub_135DD(11,16);sub_32999(4);sub_4E031();sub_11CAC(1);
    sub_1366A(3);sub_134E4();MESSAGE(4);
}
#endif
#ifdef F2C39B
void sub_2C39B(int unit,int text)
{
    sub_4E031();sub_1956B(unit);sub_4E031();
    sub_15F84(dword_53A79,text,(void *)0xa9514,320,205,76,74,19,1);
    sub_16559(0);sub_16C57(0);sub_2D31B();sub_4E031();
}
#endif
#ifdef F1AEB1
void sub_1AEB1(int x,int y,int value)
{
    int selector=131;
    unsigned char *base;
    if(value<0){selector=132;value=abs(value);}
    base=dword_53A81;
    base+=*(unsigned *)(base+selector*4+6);
    sub_4E63D(base,0,0,x,y,-1);
    sub_187D6(x+8,y,value,31,2);
}
#endif
#ifdef F2FFA5
void sub_2FFA5(void)
{
    int result,again,saved;
    dword_5413F=dword_53BFB;
    do{
        result=sub_2E6B8();again=result;sub_2D31B();saved=dword_53C67;
        if(result!=-1){
            sub_17AED(dword_53C57);dword_53C67=saved;
            dword_53A85=sub_111BA(aDatoDat,dword_53A85,byte_52659);
        }
    }while(again!=-1);
}
#endif
#ifdef F34DCD
void sub_34DCD(int unit)
{
    int state;
    if(unit!=0)return;
    if(sub_1B8A6(unit)==8)return;
    state=*(volatile unsigned char *)(dword_53AD5+16);
    if(state)return;
    sub_1BB8C(unit,89);MESSAGE(11);dword_53AD5[16]=1;
}
#endif
#ifdef F1297D
void sub_1297D(void)
{
    if(BIOS_WORD-dword_53C0F>4||BIOS_WORD-dword_53C0F<0){
        ++dword_53C0B;if(dword_53C0B==4)dword_53C0B=0;dword_53C0F=BIOS_WORD;
    }
    ++dword_53C07;if(dword_53C07==4)dword_53C07=0;
}
#endif
#ifdef F35A48
void sub_35A48(void){MESSAGE(4);sub_35822(14,7,2);MESSAGE(6);dword_53AD5[18]=1;}
#endif
#ifdef F34E90
void sub_34E90(void)
{
    int index,offset;
    MESSAGE(6);
    for(index=64;index<=73;++index){offset=index*80;(dword_53A45+offset)[53]=0;}
    sub_3419C(64,73,3);sub_3419C(35,49,0);
}
#endif
#if defined(F2D31B) || defined(F196CB)
#ifdef F2D31B
void sub_2D31B(void)
#else
void sub_196CB(void)
#endif
{
    int index;
    for(index=1;index<6;++index)sub_1974C(index*13+112,dword_53C5B,dword_53C63);
    memmove((void *)0xa0000,dword_53C5F,64000);
    free(dword_53C5B);free(dword_53C5F);free(dword_53C63);
#ifdef F196CB
    sub_11CAC(0);
#endif
}
#endif
#ifdef F34924
void sub_34924(void)
{
    if((int)*(volatile unsigned char *)(dword_53AD5+16)!=1)return;
    byte_53AFA=1;sub_10B4E(2);byte_53AFA=0;
    sub_135DD(16,10);sub_1366A(30);MESSAGE(2);dword_53AD5[17]=1;
}
#endif
#ifdef F20BF5
void sub_20BF5(void)
{
    if(sub_3453E(20))dword_53ECC=2;
    if(sub_3453E(0))dword_53ECC=1;
    if(sub_3453E(1)){MESSAGE(7);dword_53ECC=1;}
}
#endif
#ifdef F1AF1E
void sub_1AF1E(unsigned char *dest,int phase)
{
    int count=170,source_offset=75,dest_offset=75,index;
    if(phase<5){
        source_offset=75-(4-phase)*50;
        if(source_offset<0){count+=source_offset;dest_offset-=source_offset;source_offset=0;}
    }
    dest+=dest_offset+11840;source_offset+=11840;
    for(index=0;index<117;++index){memmove(dword_53A49+source_offset,dest,count);dest+=320;source_offset+=320;}
}
#endif
#ifdef F344C2
void sub_344C2(void)
{
    if(sub_3453E(6))return;
    sub_10B4E(2);sub_135DD(3,0);sub_375B2(800);sub_135DD(3,17);sub_375B2(200);MESSAGE(4);
}
#endif
#ifdef F2B9A1
void sub_2B9A1(unsigned char *data,int active,int x,int y)
{
    int offset,count;
    if(!active){byte_540FC=0;byte_540FD=0;return;}
    sub_2935B(data,byte_540FD,x,y,active);
    offset=*(int *)(data+byte_540FD*4+8);
    count=data[offset+6];
    ++byte_540FC;
    if((int)byte_540FC>=count){
        byte_540FC=0;++byte_540FD;
        offset=*(volatile unsigned char *)&byte_540FD;
        count=*(volatile unsigned char *)data;
        if(offset>=count)byte_540FD=0;
    }
}
#endif
#ifdef F24B4D
void sub_24B4D(int count)
{
    int index;
    sub_11EEE(dword_53A49+32904,456,13,9,dword_53AA9,dword_53AAD);sub_11CAC(0);
    for(index=0;index<count;++index){
        sub_11EB0((void *)0xa0504,320,dword_53A49+32904+(index&1)*456,456,312,192);
        sub_375B2(20);
    }
}
#endif
#ifdef F11CAC
void sub_11CAC(int flag)
{
    sub_1297D();if(!flag)sub_4DFCC();
    sub_11EEE(dword_53A49+32904,456,13,8,dword_53AA9,dword_53AAD);
    sub_122DC();sub_127A9();sub_1ACF3(dword_53A49+32904,456);
    sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192);
}
#endif
#ifdef F22656
void sub_22656(int unused1,int unused2,unsigned char *source,int x,int y)
{
    int index;
    unsigned char *entry,*base;
    for(index=0;index<10;++index){
        base=dword_53A6D;entry=base+*(unsigned *)(base+index*4+6);
        memmove(dword_53A49,source,153216);
        sub_22046(x,y,11,0,192,entry);
        sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192);sub_17AA9(1);
    }
}
#endif
#ifdef F1E529
int sub_1E529(unsigned short *dest,const unsigned char *range,int text,int selector)
{
    int value=range[0],difference=(int)range[1]-value;
    if(difference)difference=sub_4E893()%difference;
    value+=difference;dword_53AE1=value;
    if(value){
        if(selector==3){--selector;sub_16E24();}
        sub_4E031();
        sub_15F84(dword_53A7D,text,(void *)(0xa951f+selector*6080),320,205,76,74,19,1);
        *dest+=(unsigned short)dword_53AE1;++selector;
    }
    return selector;
}
#endif
#ifdef F1DEBE
int sub_1DEBE(int unit,int x,int y)
{
    unsigned char *record=dword_53A45+unit*80;
    int first,second,slot;
    if(record[38])return -1;
    first=x-(int)record[0];first=abs(first);
    second=y-(int)record[1];second=abs(second);
    if(first+second!=1)return -1;
    slot=sub_1B83D(unit,0);if(slot==-1)return slot;
    if((int)sub_4E56C(sub_1B722(unit,slot))[11]>1)return -1;
    return 1;
}
#endif
#ifdef F34422
void sub_34422(void)
{
    int index,offset;
    sub_135DD(9,1);sub_375B2(100);byte_53AFA=1;sub_10B4E(3);byte_53AFA=0;
    sub_1366A(13);sub_375B2(200);MESSAGE(4);
    for(index=5;index<11;++index){offset=index*80;(dword_53A45+offset)[53]=26;(dword_53A45+offset)[54]=15;}
}
#endif
