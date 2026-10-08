/* 七個完整線性呼叫包裝C，預設3s產碼逐位元組匹配。
 * 固定原檔、IDA Pro 9.4線性位址、原始名稱及caller見匹配主收據。
 * 每個候選包含全部push、call、cleanup及RET；未匹配不增加覆蓋。
 * 宣告只用於產碼，不推定原作者型別或callee玩法語意。
 */
extern int dword_53A79;
extern void sub_22AA8(int,int,int,int,int);
extern void sub_22CDA(int,int,int,int,int);
extern void sub_12CEA(int,int);
extern void sub_22253(int,int,int,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_34A0E(int);
extern void sub_3419C(int,int,int);
extern void sub_10B4E(int);
extern void sub_112A5(int);
extern void sub_35BBA(int);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#ifdef F22A85
void sub_22A85(int first,int second,int third)
{
    sub_22AA8(first,20,second,third,37);
}
#endif

#ifdef F22BE1
void sub_22BE1(int first,int second,int third)
{
    sub_22CDA(first,22,second,third,39);
}
#endif

#ifdef F33F78
void sub_33F78(int first,int second,int third)
{
    sub_12CEA(second,third);
    sub_22253(first,second,third,second,third);
}
#endif

#ifdef F34A3C
void sub_34A3C(int value)
{
    MESSAGE(2);
    sub_34A0E(value);
}
#endif

#ifdef F34F02
void sub_34F02(void)
{
    MESSAGE(8);
    sub_3419C(16,34,0);
}
#endif

#ifdef F350CC
void sub_350CC(void)
{
    sub_10B4E(1);
    MESSAGE(1);
    sub_112A5(27);
}
#endif

#ifdef F35321
void sub_35321(void)
{
    MESSAGE(5);
    sub_35BBA(18);
}
#endif
