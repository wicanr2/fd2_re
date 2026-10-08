/* 中型狀態、搜尋與資料讀取C候選。固定原檔、IDA Pro 9.4線性位址及caller見主收據。
 * 保留原名、欄位偏移與全部返回路徑；宣告只作產碼導覽。
 * exit(1)由1005E原始指令確認，不猜補外部共用錯誤尾段。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A49,*dword_53A55,*dword_53A59,*dword_53A61,*dword_53C5B,*dword_53C5F,*dword_53C63,*dword_53A81;
extern int dword_53EC4,dword_53EC8,dword_53ECC,dword_53BEF,dword_53C03,dword_53AD9,dword_53A79,dword_53A7D,dword_54127,dword_53A51,dword_53BFF,dword_53BE3,dword_51A83;
extern void *dword_53EEC;
extern unsigned char word_53A8D[];
extern void *memmove(void *,const void *,unsigned),*memset(void *,int,unsigned),*malloc(unsigned);
extern void free(void *),exit(int);
extern void sub_182AD(int,void *,void *),sub_18312(int,void *,void *),sub_1839B(int,void *,void *);
extern void sub_1CA89(int,int),sub_1C4CC(int,int,int,void *),sub_1C2DA(int,int,int,void *),sub_1E1DC(int),sub_1DF58(void);
extern void sub_11CAC(int),sub_17AA9(int),sub_4E031(void),loc_205BE(void);
extern void sub_1685C(int,int,void *,int),sub_15F84(int,int,void *,int,int,int,int,int,int);
extern int sub_3453E(int),sub_10620(void),sub_4E893(void),sub_1F183(int),sub_1C81F(int,int),sub_1C916(int,int);
extern void sub_1E0DB(int,int,int),sub_1956B(int),sub_16559(int),sub_16C57(int),sub_2D31B(void);
extern unsigned char *sub_4E516(int);
extern void sub_31E80(int,int,int,int),sub_17EEF(int,void *),sub_184C0(int,int,void *),sub_18409(int,void *,void *,void *);
extern void sub_25A96(void *,int,int),sub_4E63D(void *,int,int,void *,int,int),sub_2935B(void *,int,void *,int,int);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_24D22(int),sub_11D40(int,int,int),sub_11506(void);
extern int __cdecl int386(int,void *,void *);
extern void *fopen(const char *,const char *);
extern int fclose(void *),fseek(void *,long,int),printf(const char *,...);
extern unsigned sub_37072(void *,unsigned,unsigned,void *),fwrite(const void *,unsigned,unsigned,void *);
extern void *sub_111BA(const char *,void *,int);
extern void sub_10C50(int,void *);
extern unsigned char *dword_5410B[];
extern const char aRb_12[],aFdiconB24_6[],aFileFdiconB24E[],aFdfieldDat[],aWb_2[],aFd2Tmp_1[];
extern const char aRb_13[],aFileNotFoundS[],aOutOfMemoryAtL[];
typedef struct {int values[28];} RawCoefficients;
typedef char RawCoefficientsMustBe112[sizeof(RawCoefficients)==112?1:-1];
extern const RawCoefficients unk_51F96;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F18409
void sub_18409(int phase,void *work,void *source,void *saved)
{
    memmove(work,saved,64000);
    if(phase>=6)sub_182AD(5-(phase*16-96),work,source);else sub_182AD(5,work,source);
    if(phase<9&&phase>2)sub_18312(7-(phase*16-48),work,source);
    else if(phase<=2)sub_18312(7,work,source);
    if(phase<6)sub_1839B(phase*16+94,work,source);
    memmove((void *)0xa0000,work,64000);
}
#endif
#ifdef F22C04
void sub_22C04(int unit,int count,unsigned char *targets)
{
    int index=0,target,offset,value,kind;
    unsigned char *record;
    dword_53EC4=0;sub_1CA89(unit,25);sub_1C4CC(unit,25,count,targets);sub_1C2DA(unit,25,count,targets);
    goto check;
next:
    ++index;
check:
    if(index>=count)goto done;
    target=targets[index];offset=target*80;record=dword_53A45+offset;
    if(!(record[5]&128)){sub_1E1DC(target);goto next;}
    record[5]&=127;value=record[33];kind=record[32];if(kind>8&&kind<25)value+=30;
    dword_53EC8+=value*8;goto next;
done:
    sub_11CAC(0);if(dword_53EC4)sub_1DF58();
}
#endif
#ifdef F1E739
void sub_1E739(int x,int y,int count)
{
    int index=0;
    if(count>0){
        sub_1685C(x,y,dword_53A81,23);
        for(index=1;index<count;++index)sub_1685C(x+index,y,dword_53A81,24);
        sub_1685C(x+index,y,dword_53A81,25);
        if(count>=70)return;
        for(++index;index<=70;++index)sub_1685C(x+index,y,dword_53A81,index<70?29:30);
    }else{
        for(index=0;index<=69;++index)sub_1685C(x+index,y,dword_53A81,index<69?29:30);
    }
}
#endif
#ifdef F20765
void sub_20765(void)
{
    unsigned char missing=0;
    int index,state;
    loc_205BE();index=0;goto check;
next:
    ++index;
check:
    if(index>=12)goto done;
    if(sub_3453E(index+15))goto next;
    missing=1;goto next;
done:
    state=*(volatile unsigned char *)&missing;
    if(!state){dword_53ECC=1;MESSAGE(10);}
    if(dword_53BEF>5&&sub_3453E(59)){dword_53ECC=1;MESSAGE(2);}
}
#endif
#ifdef F1C75E
int sub_1C75E(int unit,int id)
{
    RawCoefficients coefficients=unk_51F96;
    int offset=unit*80,kind,amount,chance;
    unsigned char *data;
    kind=(dword_53A45+offset)[32];data=sub_4E516(id);
    amount=(int)*(short *)data*coefficients.values[kind]/10;chance=data[2];
    if(id>9&&id<13&&sub_1F183(unit))return 0;
    if(sub_4E893()%100>=chance)return 0;
    return sub_1C81F(unit,amount);
}
#endif
#ifdef F31DBE
int sub_31DBE(int count,unsigned char value)
{
    unsigned char found=0;
    int index=0,offset,state;
    goto check;
next:
    ++index;
check:
    if(index>=count)goto done;
    offset=(index+1)*80;if((int)(dword_53A45+offset)[8]!=(int)value)goto next;
    found=1;goto next;
done:
    state=*(volatile unsigned char *)&found;
    if(!state){
        sub_1956B(75);dword_53AD9=(int)value+1;
        sub_15F84(dword_53A7D,657,(void *)0xa951f,320,205,76,74,19,1);
        sub_16559(0);dword_53A51=1;sub_16C57(0);dword_53A51=0;sub_2D31B();
    }
    return found;
}
#endif
#ifdef F32004
int sub_32004(int first,int second,int third,int fourth)
{
    int key;
    while(!sub_10620()){
        if((int)*(volatile short *)0x46c==dword_54127)continue;
        dword_54127=*(volatile short *)0x46c;sub_31E80(first,second,third,fourth);
        memmove((void *)0xa0000,dword_53C63,64000);
    }
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);
    key=word_53A8D[1];if(key==224||key==82||(int)word_53A8D[0]==32)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#endif
#ifdef F22AF6
void sub_22AF6(int unit,int mode,int count,unsigned char *targets,int field)
{
    int index,target,offset,value,kind,result,present;
    unsigned char *record,*marker,*target_pointer;
    sub_1C4CC(unit,mode,count,targets);sub_1C2DA(unit,mode,count,targets);
    index=0;goto check;
missing:
    sub_1E1DC(*target_pointer);
next:
    ++index;
check:
    if(index>=count)goto done;
    target=targets[index];offset=target*80;record=dword_53A45+offset;
    value=record[33];kind=*(volatile unsigned char *)(record+32);if(kind>8&&kind<25)value+=30;
    marker=record+field;present=*(volatile unsigned char *)marker;target_pointer=targets+index;
    if(!present)goto missing;
    result=sub_1C916(*target_pointer,10);sub_1E0DB(result,105,*target_pointer);
    *marker=0;dword_53EC8+=value*4;goto next;
done:
    sub_11CAC(0);
}
#endif
#ifdef F17E0B
void sub_17E0B(int unit)
{
    int index;
    dword_53C5B=malloc(64000);dword_53C5F=malloc(64000);dword_53C63=malloc(64000);
    memmove(dword_53C5F,(void *)0xa0000,64000);memmove(dword_53C63,dword_53C5F,64000);
    sub_17EEF(unit,dword_53C63);sub_184C0(unit,-1,dword_53C63);
    index=11;goto check;
next:
    --index;
check:
    if(index<0)goto done;
    if(index==11||index==5)sub_25A96(dword_53EEC,5,1);
    sub_18409(index,dword_53C5B,dword_53C63,dword_53C5F);goto next;
done:
    sub_4E031();
}
#endif
#ifdef F111BA
void *sub_111BA(const char *name,void *old,int id)
{
    void *file;
    int *header,start;
    void *data;
    if(old)free(old);
    file=fopen(name,aRb_13);
    if(!file){printf(aFileNotFoundS,name);exit(1);}
    header=malloc(8);fseek(file,id*4+6,0);sub_37072(header,1,8,file);
    start=header[0];dword_53BFF=header[1]-start;free(header);data=malloc(dword_53BFF);
    if(!data){printf(aOutOfMemoryAtL,name,id);exit(1);}
    fseek(file,start,0);sub_37072(data,1,dword_53BFF,file);fclose(file);return data;
}
#endif
#ifdef F2A5D0
void sub_2A5D0(unsigned char *data,void *work,int count)
{
    int sequence=0,frame=0,index=0,pattern=0,offset,frames,divisor=3;
    for(index=0;index<count;++index){
        memset(work,0,128000);pattern=(pattern+1)%divisor;
        sub_4E63D(dword_5410B[pattern],0,50,work,640,-1);sub_2935B(data,sequence,work,640,-1);
        sub_11EB0((void *)0xa0000,320,work,640,320,200);
        offset=*(int *)(data+sequence*4+8);frames=data[offset+6];++frame;
        if(frame==frames){frame=0;++sequence;if(sequence==(int)data[0])sequence=0;}
        sub_17AA9(1);
    }
}
#endif
#ifdef F10B4E
void sub_10B4E(int value)
{
    void *file=fopen(aFdiconB24_6,aRb_12);
    int index=0;
    if(!file){*(unsigned short *)word_53A8D=3;int386(16,word_53A8D,word_53A8D);printf(aFileFdiconB24E);exit(1);}
    dword_53A59=sub_111BA(aFdfieldDat,dword_53A59,dword_53C03*3+2);
    goto check;
next:
    ++index;
check:
    if(index>=dword_53BE3)goto done;
    if((int)(dword_53A55+index*26+131)[21]!=value)goto next;
    sub_10C50(index,file);goto next;
done:
    fclose(file);free(dword_53A59);dword_53A59=0;
    file=fopen(aFd2Tmp_1,aWb_2);fwrite(dword_53A61,1,207360,file);fclose(file);
}
#endif
#ifdef F24C1E
void sub_24C1E(void)
{
    int shade=0,rows,step;
    MESSAGE(2);dword_51A83=shade;
    for(rows=2;rows<10;++rows){sub_24D22(rows);for(step=0;step<30;++step){sub_11CAC(1);sub_17AA9(1);}}
    MESSAGE(3);dword_51A83=0;
    for(;rows<15;++rows){sub_24D22(rows);for(step=0;step<12;++step){sub_11D40(0,255,shade);sub_11CAC(0);sub_17AA9(1);++shade;}}
    memset((void *)0xa0000,0,64000);sub_11506();++dword_53C03;
}
#endif
