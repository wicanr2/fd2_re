/* sub_2C39B完整C候選；保留原始呼叫順序與A9514h目的位址。
 * 保守pragma允許修改不當作4E031／1956B實際clobber或作者宣告。
 */
extern int dword_53A79;
extern void sub_4E031(void),sub_1956B(int),sub_16559(int),sub_16C57(int),sub_2D31B(void);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#ifdef TERMINAL_DIALOGUE_1
#pragma aux sub_4E031 modify exact [eax ebx ecx edx];
#endif
#ifdef TERMINAL_DIALOGUE_2
#pragma aux sub_1956B modify exact [eax ebx ecx edx];
#endif
void sub_2C39B(int first,int second)
{
    sub_4E031();sub_1956B(first);sub_4E031();
    sub_15F84(dword_53A79,second,(void *)0xa9514,320,205,76,74,19,1);
    sub_16559(0);sub_16C57(0);sub_2D31B();sub_4E031();
}
