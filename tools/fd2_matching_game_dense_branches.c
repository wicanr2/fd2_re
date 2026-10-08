/* 多分支、搜尋及裁切C候選。固定原檔、IDA Pro 9.4線性位址與caller見主收據。
 * 原名與偏移保留。宣告及變數只作產碼導覽，不推定作者型別或欄位用途。
 * 12DAC的初始ESI來源未明，本批不以未初始化C變數猜補。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7,*dword_53A49,*dword_53A51,*dword_53AD5,*dword_53C1B,*dword_53A81,*dword_54137;
extern int dword_53BEB,dword_53BFB,dword_53AC1,dword_53AC5,dword_53AA9,dword_53AAD;
extern int dword_53ECC,dword_53A79,dword_5412B,dword_53C57,dword_51A83,dword_51A8F,dword_53C03;
extern void *dword_53EEC;
extern short word_539F0,word_539F2;
extern unsigned char word_53A8D[];
extern int sub_3453E(int),sub_10620(void),sub_17898(int,int *),sub_2D85F(int);
extern void sub_12E38(int,int,void *),sub_3419C(int,int,int),sub_25A96(void *,int,int),sub_13A9F(int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_16886(void *,int,int,int),sub_1685C(int,int,void *,int);
extern void *memmove(void *,const void *,unsigned);
extern void sub_4DFCC(void),sub_4E031(void),sub_11CAC(int);
extern int __cdecl int386(int,void *,void *);
extern void (*const funcs_1199C[])(int);
extern void (*const funcs_1197B[])(int);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F20AAF
void sub_20AAF(void)
{
    if(sub_3453E(0)||sub_3453E(1)||sub_3453E(16)||sub_3453E(17))dword_53ECC=1;
    if(sub_3453E(18))dword_53ECC=2;
}
#endif
#ifdef F2D392
int sub_2D392(unsigned char *output)
{
    int limit=8,start,index=0,count=0;
    unsigned char *item;
    if(dword_5412B==1){limit=12;start=3;}else if(dword_5412B==3)start=15;else start=23;
    for(;;){
        if(index>=limit)return count;
        item=dword_54137+start+index;
        if((int)*item==255)return count;
        output[count]=*item;++count;++index;
    }
}
#endif
#ifdef F12263
void sub_12263(void)
{
    unsigned char info[8],flag,*cell;
    unsigned short code;
    int row=0,column,offset;
    goto row_check;
next_row:
    ++row;
row_check:
    if(row>=dword_53AC5)return;
    column=0;goto column_check;
next_column:
    ++column;
column_check:
    if(column>=dword_53AC1)goto next_row;
    sub_12E38(column,row,info);code=*(unsigned short *)(info+2);flag=info[4];flag&=96;
    if(flag!=32||!(int)dword_53AD5[code])goto next_column;
    offset=(dword_53AC1*row+column)*4;cell=dword_53A51+offset;cell+=4;
    ++*(unsigned short *)cell;cell[2]=0;goto next_column;
}
#endif
#ifdef F20B72
void sub_20B72(void)
{
    if(dword_53AD5[18]&&dword_53AD5[19]&&dword_53AD5[20])dword_53ECC=2;
    if(sub_3453E(0))dword_53ECC=1;
    if(sub_3453E(1)){MESSAGE(9);dword_53ECC=1;}
}
#endif
#ifdef F12C60
int sub_12C60(int value)
{
    unsigned char *record=dword_53A45;
    int index=0;
    dword_53C1B=0;
    goto first_check;
first_next:
    record+=80;++index;
first_check:
    if(index>=dword_53BEB)goto first_done;
    if((int)record[8]!=value)goto first_next;
    dword_53C1B=record;if(sub_3453E(index))goto first_next;
    return index;
first_done:
    if(dword_53C1B)goto unavailable;
    record=dword_53BF7;index=0;goto second_check;
second_next:
    record+=80;++index;
second_check:
    if(index>=dword_53BFB)goto unavailable;
    if((int)record[8]!=value)goto second_next;
    dword_53C1B=record;goto second_next;
unavailable:
    return -1;
}
#endif
#ifdef F1FF79
void sub_1FF79(int value,int selected,int count)
{
    int mode=1;if(!selected)mode=2;
    sub_16886((void *)0xacd81,320,value,mode);
    if(count>1){mode=3;if(selected==1)mode=4;sub_16886((void *)0xad8c1,320,value,mode);}
    if(count>2){mode=5;if(selected==2)mode=6;sub_16886((void *)0xae401,320,value,mode);}
}
#endif
#ifdef F1B019
void sub_1B019(unsigned char *dest,int phase)
{
    int count=17,dest_row=19,source_row,index,offset;
    if(phase<3)return;
    if(phase>7)source_row=dest_row;
    else{
        source_row=19-(4-(phase-3))*6;
        if(source_row<0){dest_row-=source_row;count+=source_row;source_row=0;}
    }
    dest+=dest_row*320+109;offset=source_row*320+109;
    for(index=0;index<count;++index){memmove(dword_53A49+offset,dest,102);dest+=320;offset+=320;}
}
#endif
#ifdef F34716
void sub_34716(void)
{
    unsigned char missing=0;
    int index=7;
    sub_3419C(7,36,7);MESSAGE(8);goto check;
next:
    ++index;
check:
    if(index>=37)goto done;
    if(sub_3453E(index))goto next;
    missing=1;goto next;
done:
    if((int)missing==1)sub_15F84(dword_53A79,11,(void *)0xa0000,320,205,76,74,19,(int)missing);
}
#endif
#ifdef F177FC
int sub_177FC(int value,int *flags)
{
    int key=sub_17898(value,flags);
    if(key==1)return -1;
    if(key==57||key==28)return 1;
    if(key==72){if(!flags[0])dword_53C57=0;return 0;}
    if(key==80){if(!flags[3])dword_53C57=3;return 0;}
    if(key==75){if(!flags[1])dword_53C57=1;return 0;}
    if(key==77&& !flags[2])dword_53C57=2;
    return 0;
}
#endif
#ifdef F17D6F
void sub_17D6F(int x,int y,int count,int glyph)
{
    int index=0;
    if(count){
        sub_1685C(x,y,dword_53A81,glyph);
        for(index=1;index<count;++index)sub_1685C(x+index,y,dword_53A81,glyph+1);
        sub_1685C(x+index,y,dword_53A81,glyph+2);
    }else{
        for(index=1;index<=102;++index){
            if(index<=101)sub_1685C(x+index,y,dword_53A81,29);
            else sub_1685C(x+index,y,dword_53A81,30);
        }
    }
}
#endif
#ifdef F1B14B
void sub_1B14B(unsigned char *dest,int phase)
{
    int count=15,start,index,offset;
    if(phase<8)return;
    if(phase>12)start=172;
    else{
        int delta=(4-(phase-8))*4;start=delta+172;
        if(delta+187>200)count=start-200;
    }
    dest+=55169;offset=start*320+129;
    for(index=0;index<count;++index){memmove(dword_53A49+offset,dest,63);dest+=320;offset+=320;}
}
#endif
#ifdef F1EC2A
void sub_1EC2A(int *output,int unit)
{
    int offset=unit*80,next;
    unsigned char *record=dword_53A45+offset;
    output[0]=((int)record[0]-dword_53AA9)*24+4;output[1]=((int)record[1]-dword_53AAD)*24;
    if((int)record[3]<2){
        next=output[1]-18;if(next<0)output[1]+=5;else output[1]=next;
        if(output[0]+108>319){output[0]-=86;return;}
        output[0]+=28;return;
    }
    if(output[1]+37>199)output[1]+=5;else output[1]+=22;
    if(output[0]-86<0){output[0]+=28;return;}
    output[0]-=88;
}
#endif
#ifdef F1B0AD
void sub_1B0AD(unsigned char *dest,int phase)
{
    int count=16,start,index,offset;
    if(phase<5)return;
    if(phase>9)start=155;
    else{
        int delta=(4-(phase-5))*9;start=delta+155;
        if(delta+171>200)count=start-200;
    }
    dest+=49675;offset=start*320+75;
    for(index=0;index<count;++index){memmove(dword_53A49+offset,dest,170);dest+=320;offset+=320;}
}
#endif
#ifdef F11AA8
int sub_11AA8(void)
{
    int key;
    while(!sub_10620()){
        sub_4DFCC();word_539F0=*(volatile short *)0x46c;
        if((int)word_539F0==(int)word_539F2)continue;
        sub_11CAC(0);word_539F2=*(volatile short *)0x46c;
    }
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);
    key=word_53A8D[1];if(key==224||key==82)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#endif
#ifdef F2D7BD
int sub_2D7BD(void)
{
    int result=0,key;
    do{
        key=sub_2D85F(0);
        if(key==75){sub_25A96(dword_53EEC,0,1);--dword_53C57;if(dword_53C57<0)dword_53C57=3;}
        else if(key==77){sub_25A96(dword_53EEC,0,1);++dword_53C57;if(dword_53C57>3)dword_53C57=0;}
        else if(key==28||key==57)result=1;
        else if(key==1)result=-1;
    }while(!result);
    return result;
}
#endif
#ifdef F1D80B
void sub_1D80B(void)
{
    int index=0,offset,side;
    unsigned char *record;
    dword_51A83=0;goto check;
next:
    ++index;
check:
    if(index>=dword_53BEB)return;
    dword_51A83=0;sub_4E031();offset=index*80;record=dword_53A45+offset;dword_51A8F=255;
    side=*(volatile unsigned char *)(record+6);
    if(side==1&&!(record[5]&129)&&!(int)*(volatile unsigned char *)(record+38))sub_13A9F(index,side);
    if(dword_51A8F!=255)funcs_1199C[dword_51A8F](index);
    funcs_1197B[dword_53C03](index);
    if(!dword_53ECC)goto next;
}
#endif
