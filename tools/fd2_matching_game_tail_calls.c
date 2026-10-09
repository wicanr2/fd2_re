/* 完整事件處理器的C尾呼叫表示。原始旗標寬度、常數與callee保留。
 * 附加語意沿既有事件證據，不由導覽名稱推定原作者宣告。
 */
extern unsigned char byte_53AFA;
extern int dword_53A79;
extern void sub_135DD(int,int),sub_10B4E(int),sub_1366A(int),sub_134E4(void);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#ifdef TAIL_EVENT_34C76
void sub_34C76(void)
{
    sub_135DD(12,5);byte_53AFA=1;sub_10B4E(2);byte_53AFA=0;
    sub_1366A(42);sub_134E4();
}
#endif
#ifdef TAIL_EVENT_35487
void sub_35487(void)
{
    sub_135DD(6,40);sub_10B4E(1);sub_1366A(74);
    sub_15F84(dword_53A79,5,(void *)0xa0000,320,205,76,74,19,1);
    sub_134E4();
}
#endif
