/* sub_3396A完整C候選；原始#88與selector1僅屬此過場owner。
 * 既有#80一般玩家指令證據與硬體時序停止線保持。
 */
extern void *dword_53B13,*dword_53A49;
extern int dword_53A79;
extern const char aFdotherDat[];
extern void sub_205DA(void),sub_135DD(int,int),sub_24B4D(int),j___delay(unsigned);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_25A96(void *,int,int),sub_12D7B(int),sub_1D4F6(void);
extern void *sub_111BA(const char *,void *,int),*memset(void *,int,unsigned);
#define MESSAGE(N) sub_15F84(dword_53A79,N,(void *)0xa0000,320,205,76,74,19,1)
void sub_3396A(void)
{
    sub_205DA();dword_53B13=0;dword_53B13=sub_111BA(aFdotherDat,0,88);
    sub_135DD(5,0);MESSAGE(1);memset(dword_53A49,0,153216);
    sub_25A96(dword_53B13,1,1);sub_24B4D(20);j___delay(600);
    sub_25A96(dword_53B13,1,1);sub_24B4D(20);j___delay(600);
    sub_25A96(dword_53B13,1,1);sub_24B4D(20);j___delay(600);
    sub_25A96(dword_53B13,1,1);sub_24B4D(60);
    MESSAGE(2);sub_12D7B(0);sub_1D4F6();
}
