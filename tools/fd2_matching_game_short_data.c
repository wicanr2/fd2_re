/* 短資料讀寫、archive與局部資料C候選，八個預設3s完整匹配。
 * 固定原檔、IDA Pro 9.4線性位址、原名及caller見匹配主收據。
 * 運算寬度、呼叫順序及完整RET逐一比對，型別不推定為作者宣告。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A55;
extern void *dword_53A61;
extern int dword_53A79,dword_53BEF;
extern const char unk_5023B[],aFd2Tmp_0[];
extern void *malloc(unsigned);
extern void *fopen(const char *,const char *);
extern int fclose(void *);
extern void sub_37072(void *,unsigned,unsigned,void *);
extern void sub_12CEA(int,int);
extern void sub_4E63D(const void *,int,int,int,int,int);
extern void sub_4DEDA(const void *,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_1AA1D(int,int,const void *);
extern void sub_35822(int,int,int);
typedef struct { unsigned char data[3]; } RawThree;
typedef char RawThreeMustBe3[sizeof(RawThree)==3?1:-1];
extern const RawThree unk_52742;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F12D7B
void sub_12D7B(int unit)
{
    int offset=unit*80;
    unsigned char *record=(unsigned char *)((unsigned)dword_53A45+offset);
    int x=record[0];
    int y=record[1];
    sub_12CEA(x,y);
}
#endif

#ifdef F29117
void sub_29117(void)
{
    void *file=fopen(aFd2Tmp_0,unk_5023B);
    dword_53A61=malloc(207360);
    sub_37072(dword_53A61,1,207360,file);
    fclose(file);
}
#endif

#ifdef F2935B
void sub_2935B(const unsigned char *archive,int index,int third,int fourth,int fifth)
{
    const unsigned char *shape=archive+*(const unsigned *)(archive+index*4+8);
    int first=*(const unsigned short *)shape;
    int second=*(const unsigned short *)(shape+2);
    sub_4E63D(shape+9,first,second,third,fourth,fifth);
}
#endif

#ifdef F3415E
void sub_3415E(const unsigned char *archive,int index,int x,int stride,int delta,int y)
{
    const unsigned char *data=archive+*(const unsigned *)(archive+index*4+6);
    int position=y*stride+x+delta;
    sub_4DEDA(data,position,stride);
}
#endif

#ifdef F343E2
void sub_343E2(void)
{
    unsigned char *record=dword_53A45+1040;
    record[6]=1;
    MESSAGE(7);
}
#endif

#ifdef F34C1E
void sub_34C1E(void)
{
    unsigned char *record;
    MESSAGE(2);
    record=dword_53A45+960;
    record[52]=0;
    record=dword_53A45+1040;
    record[52]=0;
}
#endif

#ifdef F34CB3
void sub_34CB3(void)
{
    unsigned char *record=dword_53A45+1120;
    record[52]=131;
}
#endif

#ifdef F34F74
void sub_34F74(int unit)
{
    RawThree data=unk_52742;
    sub_1AA1D(unit,1,&data);
    MESSAGE(11);
}
#endif

#ifdef F352E2
void sub_352E2(void)
{
    unsigned char index=*(volatile unsigned char *)&dword_53BEF;
    index-=14;
    index+=index;
    sub_35822(2,11,index);
    index=*(volatile unsigned char *)&dword_53BEF;
    index-=14;
    index+=index;
    ++index;
    sub_35822(26,11,index);
}
#endif

#ifdef F35A2F
void sub_35A2F(void)
{
    dword_53A55[6]=*(volatile unsigned char *)&dword_53BEF;
}
#endif

#ifdef F35C22
/* 原始函式只有RET；標準compiler選項只控制自動stack check。 */
#pragma off (check_stack)
void sub_35C22(void) {}
#pragma on (check_stack)
#endif
