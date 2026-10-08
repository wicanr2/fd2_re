/* 短資料流程、數字、捲動及cue C候選。原版版本、IDA Pro 9.4位址與caller見主收據。
 * 原名及偏移保留，不用假定ABI或原作型別補差異。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A49;
extern int dword_5204A,dword_53AA9,dword_53AAD,dword_51A87,dword_51A8B,dword_53EC4,dword_53BFB,dword_53C67,dword_53ECC,dword_53A79;
extern unsigned char byte_53DFC[],byte_53C6C[],byte_53D34[],byte_540FE;
extern const unsigned char byte_526A7[];
extern void *dword_53A81,*dword_53EEC;
extern int sub_31860(int,int),sub_1F183(int),sub_3453E(int),sprintf(char *,const char *,...);
extern void sub_16886(void *,int,int,int),loc_205BE(void),sub_375B2(int),sub_25A96(void *,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void *memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
typedef union {int word;unsigned char bytes[4];} RawFour;
typedef struct {char bytes[6];} RawSix;
typedef struct {unsigned char bytes[29];} RawTwentyNine;
typedef char RawSizesMustMatch[sizeof(RawFour)==4&&sizeof(RawSix)==6&&sizeof(RawTwentyNine)==29?1:-1];
extern const RawSix a05d;
extern const RawTwentyNine unk_52618;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F1E1DC
void sub_1E1DC(int unit)
{
    RawFour values;
    int offset=unit*80,x,y,index=0,position;
    unsigned char *base,*record,value;
    values.word=dword_5204A;base=dword_53A45;record=base+offset;x=record[0];y=record[1];
    if(x<=dword_53AA9-1||x>=dword_53AA9+dword_51A87||y<dword_53AAD-1||y>dword_53AAD+dword_51A8B)return;
    goto check;
next:
    ++index;
check:
    if(index>=4)goto done;
    position=dword_53EC4+index;value=(unsigned char)index;value*=5;
    if(index==1)value+=3;else value+=2;
    byte_53D34[position]=value;position=dword_53EC4+index;byte_53DFC[position]=(unsigned char)unit;byte_53C6C[position]=values.bytes[index];goto next;
done:
    dword_53EC4+=4;
}
#endif
#ifdef F187D6
void sub_187D6(unsigned char *dest,int pitch,int value,int glyph,int width)
{
    RawSix format=a05d;
    char text[20];
    int index;
    if(value<0)value=0;
    if(width==3&&value>999)sub_16886(dest,pitch,(int)dword_53A81,glyph+10);
    else if(width==2&&value>99)sub_16886(dest,pitch,(int)dword_53A81,93);
    else{
        format.bytes[3]=(char)(width+48);sprintf(text,format.bytes,value);
        for(index=0;index<width;++index)sub_16886(dest+index*6,pitch,(int)dword_53A81,(int)(unsigned char)text[index]+glyph-48);
    }
}
#endif
#ifdef F31793
int sub_31793(unsigned char *indices,unsigned char *codes)
{
    unsigned char count=0;
    int index=0,offset,id;
    unsigned char *record,*output,value;
    goto check;
next:
    ++index;
check:
    if(index>=dword_53BFB)return count;
    offset=index*80;record=dword_53A45+offset;id=record[7];
    if((int)record[33]<20||id>=18||id==7)goto next;
    indices[count]=(unsigned char)index;value=(unsigned char)id;value+=32;output=codes+(int)count;*output=value;
    if(sub_31860(index,byte_526A7[id])!=-1)*output=(unsigned char)(id+50);
    if(id==9&&sub_31860(index,90)!=-1)codes[count]=52;
    ++count;goto next;
}
#endif
#ifdef F2E19B
void sub_2E19B(void)
{
    int cycle,row,offset;
    for(cycle=0;cycle<3;++cycle){
        for(row=1;row<74;++row){offset=row*320;memmove((void *)(0xa8fca+offset),(void *)(0xa974a+offset),284);}
        for(row=0;row<6;++row){offset=row*320;memset((void *)(0xaec4a+offset),73,284);}
        sub_375B2(10);
    }
    for(row=0;row<72;++row){offset=row*320;memmove((void *)(0xa8fca+offset),(void *)(0xa99ca+offset),284);}
    for(row=0;row<8;++row){offset=row*320;memset((void *)(0xae9ca+offset),73,284);}
}
#endif
#ifdef F2E26C
void sub_2E26C(void)
{
    int cycle,row,offset;
    for(cycle=0;cycle<3;++cycle){
        for(row=72;row>=0;--row){offset=row*320;memmove((void *)(0xa974a+offset),(void *)(0xa8fca+offset),284);}
        for(row=0;row<6;++row){offset=row*320;memset((void *)(0xa8fca+offset),73,284);}
        sub_375B2(10);
    }
    for(row=71;row>=0;--row){offset=row*320;memmove((void *)(0xa99ca+offset),(void *)(0xa8fca+offset),284);}
    for(row=0;row<8;++row){offset=row*320;memset((void *)(0xa8fca+offset),73,284);}
}
#endif
#ifdef F2FB9F
void sub_2FB9F(int first,int second,unsigned char *source,int step)
{
    int initial_x=first-step*160,y=second-step*100,row,x,column,source_row;
    unsigned char *dest=dword_53A49,*input;
    memset(dest,0,64000);
    for(row=0;row<200;++row){
        source_row=y/128*320;
        if(y>=0&&y<25600){
            input=source+source_row;x=initial_x;
            for(column=0;column<320;++column){if(x>=0&&x<40960)dest[column]=input[x/128];x+=step;}
        }
        y+=step;dest+=320;
    }
}
#endif
#ifdef F16E24
void sub_16E24(void)
{
    unsigned char *base;
    int cycle,row;
    if(dword_53C67!=1832&&dword_53C67!=36887)return;
    base=(unsigned char *)(dword_53C67==1832?0xa0b4f:0xa951f);
    for(cycle=0;cycle<5;++cycle){
        for(row=1;row<72;++row)memmove(base+row*320-1,base+(row+3)*320-1,208);
        memset(base+23040,74,208);
    }
    for(row=0;row<72;++row)memmove(base+row*320-1,base+(row+4)*320-1,208);
    memset(base+23040,74,208);
}
#endif
#ifdef F2C9EC
void sub_2C9EC(int unit)
{
    RawTwentyNine values=unk_52618;
    volatile int divisor;
    int offset,value,cue;
    if(sub_1F183(unit)){divisor=6;cue=10;}
    else{
        offset=unit*80;value=values.bytes[(dword_53A45+offset)[32]];
        if(value==0){divisor=6;cue=9;}else if(value==1){divisor=4;cue=9;}else{divisor=9;cue=11;}
    }
    if((int)byte_540FE%divisor==0)sub_25A96(dword_53EEC,cue,1);
    ++byte_540FE;
}
#endif
#ifdef F20957
void sub_20957(void)
{
    unsigned char missing=0;
    int index,state;
    loc_205BE();index=38;goto first_check;
first_next:
    ++index;
first_check:
    if(index>=46)goto first_done;
    if(sub_3453E(index+15))goto first_next;
    missing=1;goto first_next;
first_done:
    state=*(volatile unsigned char *)&missing;
    if(!state){dword_53ECC=1;MESSAGE(10);}
    if(sub_3453E(0)||sub_3453E(52))dword_53ECC=1;
    missing=0;index=21;goto second_check;
second_next:
    ++index;
second_check:
    if(index>=37)goto second_done;
    if(sub_3453E(index+15))goto second_next;
    missing=1;goto second_next;
second_done:
    index=46;goto third_check;
third_next:
    ++index;
third_check:
    if(index>=68)goto done;
    if(sub_3453E(index+15))goto third_next;
    missing=1;goto third_next;
done:
    state=*(volatile unsigned char *)&missing;
    if(!state)dword_53ECC=2;
}
#endif
