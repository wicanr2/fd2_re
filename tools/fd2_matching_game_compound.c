/* 230F2原始IDA owner的三個C導覽入口。
 * 反序編譯讓compiler選擇共用尾段，程式碼布局由標準COFF重定位處理。
 * 不提供原版機器碼或指令，不改原始owner，也不推定作者的來源順序。
 */
extern int dword_53A79,dword_53C03;
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_11506(void),sub_112A5(int);
extern int sub_3453E(int);
extern void sub_233C6(void *,void *,void *,int,int,int,int,int,int,int,int);
typedef struct {unsigned char bytes[7];} RawSeven;
typedef char RawSevenMustBe7[sizeof(RawSeven)==7?1:-1];
extern const RawSeven unk_520BA,unk_520C1,unk_520C8,unk_520CF,unk_520D6,unk_520DD;
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

void sub_231F9(void)
{
    RawSeven first=unk_520CF,second=unk_520D6,third=unk_520DD;
    sub_233C6(first.bytes,second.bytes,third.bytes,0,6,41,12,8,0,6,4);
    MESSAGE(9);sub_112A5(10);sub_11506();++dword_53C03;
}

void sub_231BC(void)
{
    MESSAGE(4);sub_11506();++dword_53C03;
}

void sub_230F2(void)
{
    RawSeven first=unk_520BA,second=unk_520C1,third=unk_520C8;
    int result;
    sub_11506();result=sub_3453E(6);
    if(result)MESSAGE(6);
    else{
        sub_233C6(first.bytes,second.bytes,third.bytes,result,6,result,result,result,result,2,result);
        MESSAGE(7);sub_112A5(2);
    }
    ++dword_53C03;
}
