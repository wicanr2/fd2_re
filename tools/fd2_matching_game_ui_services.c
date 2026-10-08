/* 較大介面、地圖輔助及存檔服務C候選。原始版本、IDA Pro 9.4線性位址與caller見主收據。
 * 原名、偏移與完整區間保留；C型別只作產碼導覽，不推定作者宣告。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A49,*dword_53A51,*dword_53A59,*dword_53A5D,*dword_53A61,*dword_53A69,*dword_53A6D,*dword_53A81,*dword_53BF7;
extern int dword_53AA9,dword_53AAD,dword_53AC1,dword_51A87,dword_51A8B,dword_53A40,dword_53C1F;
extern int dword_53C17,dword_53C13,dword_53A79,dword_53A7D,dword_54153,dword_53BFB,dword_5412F;
extern int dword_51A83,dword_53C03,dword_53BEF,dword_53BF3,dword_53C57;
extern int dword_53B07,dword_53AB9,dword_53AED,dword_53AF5,dword_53AB1,dword_53AB5;
extern unsigned char word_53A8D[],byte_51AAB,byte_53AF9,byte_51E61,byte_51E62,byte_52659;
extern const unsigned char byte_51A97[],byte_526B9[];
extern const char aRb_5[],aFd2Sav_5[],aWb_1[],aFd2Sav_6[];
extern void sub_4DFCC(void),sub_1297D(void),sub_127A9(void),sub_13460(void),sub_11506(void),sub_2D31B(void);
extern int sub_10620(void),sub_33499(int),sub_1B5F1(int),sub_30550(void *,unsigned char);
extern void sub_11EEE(void *,int,int,int,int,int),sub_1ACF3(void *,int),sub_179D5(void *,void *);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_4DEDA(void *,void *,int),sub_4DD52(void *,void *,int,void *),sub_4DF4C(void *,void *,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int),sub_16886(void *,int,int,int),sub_187D6(int,int,int,int,int);
extern void sub_2C9EC(int),sub_13A44(int,int,int),sub_1C220(int),sub_135DD(int,int),sub_10B4E(int),sub_375B2(int),sub_1366A(int),sub_112A5(int);
extern unsigned char *sub_4E48D(int);
extern int __cdecl int386(int,void *,void *);
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned),*fopen(const char *,const char *);
extern void free(void *),sub_4DBD8(void *,int),sub_16559(int),sub_16C57(int),sub_1956B(int);
extern unsigned sub_37072(void *,unsigned,unsigned,void *),fwrite(const void *,unsigned,unsigned,void *),sub_4DBB9(void *,int);
extern int fclose(void *);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)
#define FULL_VIEW() sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192)
#define PLAIN_TEXT(INDEX,DEST,COLOR) sub_15F84(dword_53A7D,INDEX,DEST,320,COLOR,76,0,0,0)

#ifdef F17898
int sub_17898(void *selected,void *flags)
{
    int delta,key;
    unsigned short clock;
    while(!sub_10620()){
        sub_4DFCC();clock=*(volatile unsigned short *)0x46c;delta=(short)clock;delta-=dword_53C17;
        if(delta>3||delta<0){++dword_53C13;if(dword_53C13==2)dword_53C13=0;dword_53C17=*(volatile short *)0x46c;}
        sub_1297D();sub_11EEE(dword_53A49+32904,456,13,8,dword_53AA9,dword_53AAD);
        sub_127A9();sub_1ACF3(dword_53A49+32904,456);sub_179D5(selected,flags);FULL_VIEW();
    }
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);key=word_53A8D[1];
    if(key==224||key==82)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#endif
#ifdef F12AC6
void sub_12AC6(unsigned char *dest,int x,int y)
{
    int offset,index,flags,selector;
    unsigned short tile;
    unsigned char *cell,*base,*image,*mask,*target;
    if(dword_53AA9-1>x||dword_53AA9+dword_51A87<x||dword_53AAD-1>y||dword_53AAD+dword_51A8B+1<y||y<0)return;
    offset=(y*dword_53AC1+x)*4;cell=dword_53A51+offset+4;
    tile=*(unsigned short *)cell;tile&=1023;index=tile;flags=dword_53A69[index*4];
    if(flags&8)index+=dword_53A40*2;
    if(!(flags&128))return;
    base=dword_53A5D;image=base+*(int *)(base+index*4+10);
    selector=byte_51A97[dword_53C1F];base=dword_53A6D;mask=base+*(int *)(base+selector*4+6);
    target=dest+32904+(y-dword_53AAD)*10944+(x-dword_53AA9)*24;
    if((int)cell[3]==255)sub_4DEDA(image,target,456);else sub_4DD52(image,target,456,mask);
}
#endif
#ifdef F2EA90
void sub_2EA90(int selected,unsigned char *dest)
{
    int selector=dword_54153,count=dword_53BFB,index=0,unit,column,row,color;
    unsigned char *record,*base,*image;
    if(selector==3)selector=1;
    if(count>6){count=6;if(dword_5412F+6>dword_53BFB)count=5;}
    for(index=0;index<count;++index){
        unit=dword_5412F+index;record=dword_53A45+unit*80;
        column=(index%2)*132;row=(index/2)*26+117;
        base=dword_53A61;image=base+*(int *)(base+unit*48+selector*4);
        sub_4DF4C(image,dest+row*320+14+column,320);
        color=205;if(dword_5412F+index==selected)color=201;
        PLAIN_TEXT((int)record[8]+1,dest+(row+4)*320+40+column,color);
    }
}
#endif
#ifdef F1300D
void sub_1300D(int unit)
{
    int shift=0,camera=0,position,source_offset=32904,step;
    unsigned char *record=dword_53A45+unit*80;
    position=record[0];record[3]=1;dword_53B07=6;
    if(position-dword_53AA9<2&&dword_53AA9!=0){shift=-4;camera=-1;}else --dword_53AB9;
    for(step=1;step<7;++step){
        sub_2C9EC(unit);sub_4DFCC();record[4]=(unsigned char)step;sub_1297D();
        dword_53AED=24;dword_53AF5+=shift;dword_53B07+=camera;
        sub_11EEE(dword_53A49+32880,456,14,8,dword_53AA9-1,dword_53AAD);
        dword_53AED=0;sub_127A9();source_offset+=shift;
        sub_11EB0((void *)0xa0504,320,dword_53A49+source_offset,456,312,192);sub_13460();
    }
    --position;record[0]=(unsigned char)position;dword_53AA9+=camera;--dword_53AB1;
    record[4]=0;dword_53AF5=0;dword_53B07=0;sub_13A44(dword_53AB1,dword_53AB5,0);
}
#endif
#ifdef F22F37
void sub_22F37(void)
{
    unsigned char found=0;
    unsigned char *base=dword_53A45;
    int index=5,offset,state;
    goto check;
next:
    ++index;
check:
    if(index>=11)goto done;
    offset=index*80;if(!(base[offset+5]&1))goto next;
    found=1;goto next;
done:
    state=*(volatile unsigned char *)&found;
    if(!state){MESSAGE(6);sub_1C220(198);}else MESSAGE(7);
    sub_135DD(14,2);sub_10B4E(4);sub_375B2(100);sub_1366A(14);MESSAGE(8);dword_51A83=0;
    sub_375B2(200);sub_1366A(15);MESSAGE(9);dword_51A83=0;sub_375B2(200);sub_135DD(14,1);
    sub_375B2(200);sub_1366A(16);sub_375B2(200);MESSAGE(10);sub_112A5(8);sub_11506();dword_53C03=2;
}
#endif
#ifdef F31019
void sub_31019(int count,unsigned char *dest,int selected,unsigned char *units,unsigned char *commands)
{
    int selector=dword_54153,limit=count,index,unit,row,color;
    unsigned char *record,*base,*image,*text;
    if(selector==3)selector=1;if(limit>3)limit=3;
    for(index=0;index<limit;++index){
        unit=units[dword_5412F+index];record=dword_53A45+unit*80;row=index*26+117;
        base=dword_53A61;image=base+*(int *)(base+unit*48+selector*4);
        sub_4DF4C(image,dest+row*320+14,320);
        color=205;if(dword_5412F+index==selected)color=201;
        text=dest+(row+4)*320;
        PLAIN_TEXT((int)record[8]+1,text+40,color);PLAIN_TEXT((int)record[32]+150,text+130,color);PLAIN_TEXT(593,text+175,color);
        image=sub_4E48D(commands[dword_5412F+index]);PLAIN_TEXT((int)image[0]+150,text+239,color);
    }
}
#endif
#ifdef F1B41D
void sub_1B41D(unsigned base,int pitch)
{
    int offset24,offset159,index;
    sub_16886((void *)(base+109+pitch*19),pitch,(int)dword_53A81,133);
    sub_16886((void *)(base+75+pitch*37),pitch,(int)dword_53A81,134);
    sub_16886((void *)(base+75+pitch*155),pitch,(int)dword_53A81,135);
    sub_16886((void *)(base+129+pitch*172),pitch,(int)dword_53A81,136);
    offset24=pitch*24;
    sub_187D6((int)(base+143+offset24),pitch,dword_53C03+1,42,2);
    sub_187D6((int)(base+188+offset24),pitch,dword_53BEF,42,3);
    sub_187D6((int)(base+140+pitch*176),pitch,dword_53BF3,31,8);
    offset159=pitch*159;
    sub_187D6((int)(base+120+offset159),pitch,sub_1B5F1(0),42,2);
    sub_187D6((int)(base+182+offset159),pitch,sub_1B5F1(2),42,2);
    sub_187D6((int)(base+228+offset159),pitch,sub_1B5F1(1),42,2);
    index=dword_53C03*2+597;
    sub_15F84(dword_53A7D,index,(void *)(base+80+pitch*61),pitch,205,76,0,19,0);
    if(dword_53C03==16&&sub_33499(18)==0)index-=2;
    ++index;sub_15F84(dword_53A7D,index,(void *)(base+80+pitch*116),pitch,205,76,0,19,0);
}
#endif
#ifdef F30012
void sub_30012(unsigned char option)
{
    unsigned char *data=malloc(22987),*slot;
    void *file;
    int result,again;
    file=fopen(aFd2Sav_5,aRb_5);
    if(file){sub_37072(data,1,22987,file);sub_4DBD8(data,22987);fclose(file);}else memset(data,255,22987);
    dword_53C57=0;
    do{
        again=sub_30550(data,option);result=again;
        if(result!=-1){
            slot=data+12587+dword_53C57*2600;memmove(slot,dword_53BF7,2560);slot+=2560;
            slot[0]=(unsigned char)dword_53C03;slot[1]=(unsigned char)dword_53BFB;*(int *)(slot+2)=dword_53BF3;
            slot[6]=byte_51AAB;slot[7]=byte_53AF9;slot[8]=byte_51E61;slot[9]=byte_51E62;
            file=fopen(aFd2Sav_6,aWb_1);*(unsigned *)(data+22983)=sub_4DBB9(data,22987);sub_4DBD8(data,22987);
            fwrite(data,1,22987,file);fclose(file);sub_4DBD8(data,22987);
            if(!byte_526B9[dword_53C03]){
                sub_2D31B();sub_1956B(byte_52659);sub_15F84(dword_53A7D,660,(void *)0xa94cc,320,205,76,74,19,1);
                sub_16559(0);sub_16C57(0);
            }
        }
        sub_2D31B();
    }while(again!=-1);
    free(data);
}
#endif
