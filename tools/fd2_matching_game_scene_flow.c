/* 較大場景與畫面流程C候選。原版版本、IDA Pro 9.4線性位址及caller見主收據。
 * 原名、原始owner及全部bytes保留，宣告只作產碼導覽。
 * 230F2內231BC／231F9只建立C入口，不改IDA邊界或另計原始函式。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A49,*dword_53A5D,*dword_53A61,*dword_53A81,*dword_53AD5;
extern unsigned char *dword_5413B,*dword_53BF7;
extern void *dword_53EEC;
extern int dword_53A79,dword_53A7D,dword_53C03,dword_5412B,dword_54133,dword_51A83;
extern int dword_53AA9,dword_53AAD,dword_53AB1,dword_53AB5,dword_53AB9,dword_53ABD;
extern int dword_53AD9,dword_53AE1,dword_53BFB;
extern unsigned char *dword_5410B[];
extern const unsigned char byte_52635[],byte_52647[];
extern const char aFdotherDat[];
extern unsigned char *sub_4E4B9(int),*sub_4E4E8(int),*sub_4E4D1(int),*sub_4E48D(int);
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void *sub_111BA(const char *,void *,int),*sub_15F0E(void *,void *,int,int,int,int);
extern void sub_4E8AF(void *,void *,int),sub_4DEDA(void *,void *,int),sub_4E809(int,void *,void *);
extern void sub_4E63D(void *,int,int,void *,int,int),sub_11EB0(void *,int,void *,int,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_11D40(int,int,int),sub_17AA9(int),sub_1F42D(int,int);
extern void sub_11EEE(void *,int,int,int,int,int),sub_127A9(void),sub_4E031(void);
extern void sub_1956B(int),sub_16559(int),sub_16C57(int),sub_196CB(void),sub_12263(void);
extern int sub_1B8A6(int),sub_3453E(int),sub_33499(int),sub_31860(int,int);
extern void sub_12E38(int,int,void *),sub_1BB8C(int,int),sub_1B8E7(int,int);
extern void sub_25A96(void *,int,int),sub_1DA16(void *,int,int,int,int);
extern void sub_2935B(void *,int,void *,int,int),sub_2A289(void *,int);
extern void sub_233C6(void *,void *,void *,int,int,int,int,int,int,int,int);
extern void sub_11506(void),sub_1366A(int),sub_112A5(int),sub_10B4E(int),sub_135DD(int,int);
extern void sub_12D7B(int);
extern void sub_1F882(void),sub_13536(void),sub_11CAC(int),sub_1F525(void),sub_375B2(int);
extern int sub_1E529(unsigned short *,void *,int,int);
extern void sub_1B750(int),sub_2D31B(void),sub_1145A(int);
typedef struct {unsigned char bytes[5];} RawFive;
typedef struct {unsigned char bytes[7];} RawSeven;
typedef struct {unsigned char bytes[11];} RawEleven;
typedef struct {unsigned char bytes[16];} RawSixteen;
typedef struct {unsigned char bytes[17];} RawSeventeen;
typedef char RawSizesMustMatch[sizeof(RawFive)==5&&sizeof(RawSeven)==7&&sizeof(RawEleven)==11&&sizeof(RawSixteen)==16&&sizeof(RawSeventeen)==17?1:-1];
extern const RawFive unk_5274E;
extern const RawSeven unk_520BA,unk_520C1,unk_520C8,unk_520CF,unk_520D6,unk_520DD;
extern const RawEleven unk_52113,unk_5211E;
extern const RawSixteen unk_521A3,unk_521B3;
extern const RawSeventeen unk_521C3,unk_521D4,unk_521E5;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)
#define FULL_VIEW() sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192)

#ifdef F2CF71
void sub_2CF71(void)
{
    unsigned char value=sub_4E4B9(dword_53C03)[0];
    unsigned char *image,*base;
    int selector;
    memmove(dword_53A49,dword_53A5D,153216);sub_4E8AF(dword_53A49+107020,dword_5413B,456);
    sub_15F84(dword_53A7D,495+dword_5412B,dword_53A49+109764,456,205,76,74,19,0);
    selector=dword_54133;if(selector==3)selector=1;
    base=dword_53A61;image=base+*(int *)(base+selector*4);
    sub_4DEDA(image,dword_53A49+32904+(int)byte_52635[(int)value*6+dword_5412B]+(int)byte_52647[(int)value*6+dword_5412B]*456,456);
    FULL_VIEW();
}
#endif
#ifdef F1F30A
void sub_1F30A(int value)
{
    int scale=17,index=16;
    unsigned char *work=malloc(64000);
    memset(work,0,64000);
    for(;index>=0;--index){
        sub_4E809(scale,work,dword_53A49);
        free(sub_15F0E(dword_53A81,work,320,89,86,value));
        free(sub_15F0E(dword_53A81,work,320,169,86,81));
        sub_11D40(16,255,index);memmove((void *)0xa0000,work,64000);sub_17AA9(1);--scale;
    }
    free(work);sub_11EEE(dword_53A49+32904,456,13,8,dword_53AA9,dword_53AAD);sub_127A9();
    for(index=0;index<=4;++index)sub_1F42D(index*25,value);
}
#endif
#ifdef F1F1CC
void sub_1F1CC(int value)
{
    int scale=1,index;
    unsigned char *work=malloc(64000);
    memmove(work,(void *)0xa0000,64000);
    for(index=4;index>=0;--index)sub_1F42D(index*25,value);
    sub_1F42D(1,value);sub_1F42D(0,value);
    memmove(dword_53A49,work,64000);memset(work,0,64000);
    for(index=0;index<16;++index){
        sub_4E809(scale,work,dword_53A49);
        free(sub_15F0E(dword_53A81,work,320,89,86,value));
        free(sub_15F0E(dword_53A81,work,320,169,86,81));
        sub_11D40(16,255,index);memmove((void *)0xa0000,work,64000);sub_17AA9(1);++scale;
    }
    free(work);sub_4E031();
}
#endif
#ifdef F354FE
void sub_354FE(int unit)
{
    RawFive choices=unk_5274E;
    unsigned char result[8],index;
    int offset;
    sub_4E031();offset=unit;offset*=80;sub_1956B((dword_53A45+offset)[7]);
    if(sub_1B8A6(unit)==8){
        sub_15F84(dword_53A7D,480,(void *)0xa9f23,320,205,76,74,19,1);
        sub_16559(0);sub_16C57(0);sub_196CB();
    }else{
        sub_12E38(dword_53AB1,dword_53AB5,result);index=result[2];dword_53AD9=(int)choices.bytes[index]+181;
        sub_15F84(dword_53A7D,422,(void *)0xa9f23,320,205,76,74,19,1);
        sub_16559(0);sub_16C57(0);sub_1BB8C(unit,choices.bytes[index]);sub_196CB();
        for(index=0;index<5;++index)dword_53AD5[index]=1;
        sub_12263();
    }
}
#endif
#ifdef F13FD4
int sub_13FD4(int unit)
{
    unsigned address=unit*80;
    int current,maximum,state;
    unsigned char *record;
    address+=(unsigned)dword_53A45;record=(unsigned char *)address;
    current=*(unsigned short *)(record+64);maximum=*(unsigned short *)(record+66);
    if(current==maximum||(int)*(volatile unsigned char *)(record+37)||(state=*(volatile unsigned char *)(record+38)))return 0;
    dword_51A83=state;sub_12D7B(unit);sub_17AA9(1);sub_25A96(dword_53EEC,4,1);
    sub_1DA16(dword_53A49+32904,456,unit,2,253);FULL_VIEW();sub_17AA9(1);
    sub_1DA16(dword_53A49+32904,456,unit,0,0);FULL_VIEW();sub_17AA9(1);
    current+=maximum/5;if(current>maximum)current=maximum;
    *(unsigned short *)(record+64)=(unsigned short)current;dword_51A83=1;return 1;
}
#endif
#ifdef F29C90
void sub_29C90(int unit,void *animation,unsigned char *one,unsigned char *two,void *image)
{
    int index,divisor=3;
    for(index=9;index>=0;--index){
        sub_4E63D(dword_5410B[index%divisor],0,50,two,640,-1);
        sub_11EB0((void *)0xa0000,320,two+index*32,640,320,200);
    }
    memset(two,0,128000);memset(one,0,64000);sub_4E63D(image,0,50,one,320,-1);sub_2A289(one,unit);
    sub_11EB0(two,640,one,320,320,200);sub_2935B(animation,0,two,640,-1);
    for(index=9;index>=0;--index){
        sub_4E63D(dword_5410B[(index+2)%divisor],0,50,two+320,640,-1);
        sub_11EB0((void *)0xa0000,320,two+index*32,640,320,200);
    }
}
#endif
#ifdef F23CD5
void sub_23CD5(void)
{
    RawSeventeen first=unk_521C3,second=unk_521D4,third=unk_521E5;
    sub_11506();sub_233C6(first.bytes,second.bytes,third.bytes,0,16,17,25,8,1,18,4);
    MESSAGE(7);dword_51A83=0;sub_1366A(56);MESSAGE(8);dword_51A83=0;
    sub_1366A(57);MESSAGE(9);dword_51A83=0;sub_1366A(58);MESSAGE(10);
    sub_112A5(21);sub_112A5(7);++dword_53C03;
}
#endif
#ifdef F356B7
void sub_356B7(int unit)
{
    int offset,slot,index;
    void *image;
    if(dword_53AD5[12])return;
    offset=unit*80;sub_1956B((dword_53A45+offset)[7]);slot=sub_31860(unit,208);
    if(slot==-1){
        sub_15F84(dword_53A79,2,(void *)0xa951f,320,205,76,74,19,1);
        sub_16559(0);sub_16C57(0);sub_196CB();return;
    }
    sub_1B8E7(unit,slot);sub_15F84(dword_53A79,3,(void *)0xa951f,320,205,76,74,19,1);
    sub_16C57(0);sub_196CB();image=sub_111BA(aFdotherDat,0,45);
    for(index=0;index<59;++index){sub_2935B(image,index,(void *)0xabce4,320,-1);sub_17AA9(2);}
    free(image);dword_53AD5[12]=1;sub_12263();sub_10B4E(1);sub_112A5(31);MESSAGE(4);
}
#endif
#ifdef F23B5F
void sub_23B5F(void)
{
    RawSixteen first=unk_521A3,second=unk_521B3;
    int result;
    sub_11506();result=sub_33499(18);
    if(result){MESSAGE(5);dword_51A83=0;sub_135DD(17,14);sub_10B4E(3);sub_1366A(52);}
    else{
        sub_233C6(first.bytes,second.bytes,(void *)result,result,15,52,23,23,2,17,17);
        MESSAGE(7);dword_51A83=0;sub_1366A(50);sub_135DD(17,14);sub_10B4E(3);sub_1366A(51);
    }
    MESSAGE(6);sub_1366A(53);MESSAGE(8);sub_112A5(16);++dword_53C03;
}
#endif
#ifdef F29DED
void sub_29DED(int unit,void *animation,void *extra,unsigned char *one,unsigned char *two,void *image)
{
    int index,divisor=3;
    unsigned char *dest=two+320;
    for(index=1;index<10;++index){
        sub_4E63D(dword_5410B[index%divisor],0,50,dest,640,-1);
        sub_11EB0((void *)0xa0000,320,two+index*32,640,320,200);
    }
    memset(two,0,128000);memset(one,0,64000);sub_4E63D(image,0,50,one,320,-1);
    sub_4E63D(extra,164,157,one,320,-1);sub_2A289(one,unit);
    sub_11EB0(dest,640,one,320,320,200);sub_2935B(animation,0,dest,640,-1);
    for(index=1;index<=10;++index){
        sub_4E63D(dword_5410B[(index+1)%divisor],0,50,two,640,-1);
        sub_11EB0((void *)0xa0000,320,two+index*32,640,320,200);
    }
}
#endif
#ifdef F31602
void sub_31602(int unit)
{
    int offset=unit*80,selector,extra;
    unsigned char *record=dword_53A45+offset,*growth;
    growth=sub_4E4D1(record[7]);sub_4E031();sub_1956B(record[7]);dword_53AD9=(int)record[32]+150;
    sub_15F84(dword_53A7D,595,(void *)0xa951f,320,205,76,74,19,1);sub_16559(0);sub_4E031();
    selector=sub_1E529((unsigned short *)(record+55),growth,490,1);
    selector=sub_1E529((unsigned short *)(record+57),growth+2,491,selector);
    selector=sub_1E529((unsigned short *)(record+62),growth+4,492,selector);
    selector=sub_1E529((unsigned short *)(record+66),growth+6,493,selector);
    selector=sub_1E529((unsigned short *)(record+70),growth+8,494,selector);
    extra=sub_4E48D(record[7])[1];
    if(extra){
        dword_53AE1=extra;
        sub_15F84(dword_53A7D,596,(void *)(0xa951f+selector*6080),320,205,76,74,19,1);
        sub_16C57(0);record[59]+=*(volatile unsigned char *)&dword_53AE1;
    }
    sub_1B750(unit);sub_2D31B();record[33]=1;record[60]=0;
    *(unsigned short *)(record+64)=*(unsigned short *)(record+66);
    *(unsigned short *)(record+68)=*(unsigned short *)(record+70);sub_4E031();
}
#endif
#ifdef F235F9
void sub_235F9(void)
{
    RawEleven ys,xs;
    unsigned char *record;
    int index,offset;
    xs=unk_52113;ys=unk_5211E;sub_1F882();sub_13536();
    index=0;goto check;
copy:
    offset=index*80;record=dword_53A45+offset;record[0]=xs.bytes[index];record[1]=ys.bytes[index];record[3]=2;++index;
check:
    if(index<11)goto copy;
    record=dword_53A45+4000;record[0]=15;record[1]=35;record[38]=0;
    record=dword_53A45+4080;record[0]=14;record[1]=35;record[38]=0;
    record=dword_53A45+4160;record[0]=16;record[1]=35;record[5]=0;
    record=dword_53A45+400;record[5]=0;
    dword_51A83=0;dword_53AA9=9;dword_53AAD=34;dword_53AB1=9;dword_53AB5=34;dword_53AB9=0;dword_53ABD=0;
    sub_11CAC(1);sub_1F525();sub_375B2(200);MESSAGE(4);dword_51A83=0;sub_1366A(37);MESSAGE(5);
    sub_11506();sub_112A5(11);sub_112A5(6);++dword_53C03;
}
#endif
#ifdef F230F2
void sub_230F2(void)
{
    RawSeven first=unk_520BA,second=unk_520C1,third=unk_520C8;
    int result;
    sub_11506();result=sub_3453E(6);
    if(result)MESSAGE(6);
    else{sub_233C6(first.bytes,second.bytes,third.bytes,result,6,result,result,result,result,2,result);MESSAGE(7);sub_112A5(2);}
    ++dword_53C03;
}
void sub_231BC(void){MESSAGE(4);sub_11506();++dword_53C03;}
void sub_231F9(void)
{
    RawSeven first=unk_520CF,second=unk_520D6,third=unk_520DD;
    sub_233C6(first.bytes,second.bytes,third.bytes,0,6,41,12,8,0,6,4);MESSAGE(9);sub_112A5(10);sub_11506();++dword_53C03;
}
#endif
#ifdef F112A5
void sub_112A5(int id)
{
    int offset=dword_53BFB*80,level,hp_increment,hp,mp,base_first,base_second,base_third,index;
    unsigned char *dest=dword_53BF7+offset,*base,*source,*growth;
    base=sub_4E4E8(id);source=base;growth=sub_4E4D1(id);level=base[2];
    hp_increment=(level-1)*(int)growth[6];hp=hp_increment+*(unsigned short *)(base+3);
    mp=(level-1)*(int)growth[8]+*(unsigned short *)(base+5);
    base_first=*(unsigned short *)(base+18);base_second=*(unsigned short *)(base+20);base_third=*(unsigned short *)(base+22);
    dest[5]=0;dest[6]=2;dest[7]=(unsigned char)id;dest[8]=(unsigned char)id;dest[9]=0;dest[10]=64;
    dest[11]=base[12];dest[12]=64;dest[13]=base[13];
    for(index=0;index<4;++index){dest[14+index*2]=source[14+index]==255?128:0;dest[15+index*2]=source[14+index];}
    dest[22]=128;dest[24]=128;memmove(dest+26,source+8,4);dest[30]=0;dest[31]=source[0];dest[32]=source[1];dest[33]=(unsigned char)level;
    memset(dest+34,0,6);dest[49]=255;
    *(unsigned short *)(dest+55)=(unsigned short)(base_first+(unsigned short)growth[0]*level);
    *(unsigned short *)(dest+57)=(unsigned short)(base_second+(unsigned short)growth[2]*level);
    dest[59]=source[7];dest[60]=0;
    *(unsigned short *)(dest+62)=(unsigned short)(base_third+(unsigned short)growth[4]*level);
    *(unsigned short *)(dest+64)=(unsigned short)hp;*(unsigned short *)(dest+66)=(unsigned short)hp;
    *(unsigned short *)(dest+68)=(unsigned short)mp;*(unsigned short *)(dest+70)=(unsigned short)mp;
    sub_1145A(dword_53BFB);++dword_53BFB;
}
#endif
