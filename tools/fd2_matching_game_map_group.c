/* 相鄰查詢／地圖捲動的完整C編譯單元候選。
 * 12CEA的返回邊指向12C60尾端；原始位址、雜湊與語意分級見匹配主收據。
 * 只檢驗原始編譯布局，不重開已閉合的鍵盤清理或地圖語意。
 */
#ifdef MAP_QUERY_GROUP
#define F12C60
#include "REC.C"
#undef F12C60

extern int dword_53AB1, dword_53AB5, dword_51A83;
extern void sub_11CAC(int), sub_17AA9(int);
extern void sub_11C59(void), sub_11BFA(void), sub_11B48(void), sub_11B9B(void);
extern void __cdecl sub_4E031(void);

void sub_12CEA(int x, int y)
{
    sub_11CAC(0);
x_check:
    if (x == dword_53AB1) goto y_check;
    if (x < dword_53AB1) sub_11C59();
    else sub_11BFA();
    if (dword_51A83 == 0) goto x_done;
    if (dword_51A83 == 6) goto x_done;
    sub_17AA9(1);
x_done:
    sub_4E031();
    goto x_check;
y_check:
    if (y == dword_53AB5) return;
    if (y < dword_53AB5) sub_11B48();
    else sub_11B9B();
    if (dword_51A83 == 0) goto y_done;
    if (dword_51A83 == 6) goto y_done;
    sub_17AA9(1);
y_done:
    sub_4E031();
    goto y_check;
}
#endif
