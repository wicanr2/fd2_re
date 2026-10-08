/* 全域寫入與線性呼叫C候選，五個預設3s完整匹配，33FAF未匹配。
 * 固定原檔、IDA Pro 9.4線性位址、原始名稱及caller見匹配主收據。
 * 全部全域寫入、參數轉發及RET均比較；不由名稱推定原作用途。
 */
extern int dword_53EC4, dword_53A79, dword_53C03;
extern unsigned char byte_53A44;
extern void sub_1CA89(int,int);
extern void sub_22721(int,int,int);
extern void sub_22866(int,int,int);
extern void sub_22997(int,int,int);
extern void sub_22D1B(int,int,int,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_11506(void);

#ifdef F226EA
void sub_226EA(int first,int second,int third)
{
    dword_53EC4=0;
    sub_1CA89(first,18);
    sub_22721(first,second,third);
}
#endif

#ifdef F2282F
void sub_2282F(int first,int second,int third)
{
    dword_53EC4=0;
    sub_1CA89(first,18);
    sub_22866(first,second,third);
}
#endif

#ifdef F22960
void sub_22960(int first,int second,int third)
{
    dword_53EC4=0;
    sub_1CA89(first,19);
    sub_22997(first,second,third);
}
#endif

#ifdef F22CDA
void sub_22CDA(int first,int second,int third,int fourth,int fifth)
{
    dword_53EC4=0;
    sub_1CA89(first,second);
    sub_22D1B(first,second,third,fourth,fifth);
}
#endif

#ifdef F22EF6
void sub_22EF6(void)
{
    sub_15F84(dword_53A79,9,(void *)0xa0000,320,205,76,74,19,1);
    sub_11506();
    dword_53C03=1;
}
#endif

#ifdef F33FAF
void sub_33FAF(void)
{
    byte_53A44=1;
}
#endif
