/* 已閉合戰後流程的完整C候選。固定IDA9.4原名、位址、bytes及consumer見主收據。
 * 17-byte局部備份及全域副作用只作產碼表示，不重開既有戰後語意。
 */
typedef struct {unsigned char bytes[17];} RawSeventeen;
typedef char RawSeventeenMustBe17[sizeof(RawSeventeen)==17?1:-1];
extern const RawSeventeen unk_522A3,unk_522B4,unk_522C5;
extern int dword_53A79,dword_51A83,dword_53BEF,dword_53C03;
extern void *dword_53A51,*dword_53A5D,*dword_53A69;
extern const char aFdfieldDat[],aFdshapDat[];
extern void sub_233C6(void *,void *,void *,int,int,int,int,int,int,int,int);
extern int sub_24B14(int),sub_24BDE(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_112A5(int),sub_1366A(int),sub_32975(int),sub_11506(void),sub_2189A(int,int,int);
extern void sub_24B4D(int),j___delay(unsigned),sub_11DF2(int,int,int);
extern void *sub_111BA(const char *,void *,int);
extern void sub_4DBFC(void *),sub_10652(void),sub_135DD(int,int);
#define MESSAGE_WITH(CONTEXT,INDEX) sub_15F84(CONTEXT,INDEX,(void *)0xa0000,320,205,76,74,19,1)
#define MESSAGE(INDEX) MESSAGE_WITH(dword_53A79,INDEX)

void sub_24754(void)
{
    RawSeventeen first=unk_522A3,second=unk_522B4,third=unk_522C5;
    int shade;
    sub_233C6(&first,&second,&third,0,16,17,21,21,2,14,14);
    if(sub_24B14(100)!=-1){MESSAGE(8);sub_112A5(22);}
    else{MESSAGE(9);dword_51A83=0;sub_1366A(71);}
    if(sub_24BDE(18)){
        MESSAGE(10);dword_51A83=0;sub_1366A(72);sub_32975(17);MESSAGE(11);
    }else if(dword_53BEF>=15){
        MESSAGE(12);dword_51A83=0;sub_1366A(72);sub_32975(17);
    }else{MESSAGE(13);sub_112A5(19);}
    sub_11506();
#ifdef POST_DIALOGUE_1
    MESSAGE_WITH((++dword_53C03,dword_53A79),14);
#elif defined(POST_DIALOGUE_2)
    MESSAGE_WITH(dword_53A79+0*++dword_53C03,14);
#else
    ++dword_53C03;MESSAGE(14);
#endif
    j___delay(400);sub_2189A(1,15,10);sub_24B4D(30);
    MESSAGE(15);j___delay(400);sub_2189A(1,15,10);sub_24B4D(30);
    MESSAGE(16);j___delay(400);sub_2189A(1,30,16);
    shade=0;goto fade_check;
fade_next:
    sub_11DF2(0,255,shade);j___delay(4);shade+=2;
fade_check:
    if(shade<64)goto fade_next;
    dword_53A51=sub_111BA(aFdfieldDat,dword_53A51,69);
    dword_53A5D=sub_111BA(aFdshapDat,dword_53A5D,46);
    dword_53A69=sub_111BA(aFdshapDat,dword_53A69,47);
    sub_4DBFC(dword_53A51);sub_10652();sub_135DD(14,29);sub_11DF2(0,255,0);
    sub_1366A(73);sub_135DD(14,14);sub_1366A(73);sub_1366A(73);MESSAGE(17);
}
