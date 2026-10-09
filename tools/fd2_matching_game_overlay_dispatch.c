/* 固定IDA9.4的sub_122DC完整產碼候選；原名與原始offset保留。
 * 模式與座標只按原指令表示，既有游標／覆蓋層語意不重新命名。
 */
extern int dword_51A83,dword_53AB1,dword_53AB5,dword_53AC1;
extern unsigned char *dword_53A51;
extern void sub_126F7(int,int,int);
#define AT(DX,DY,N) sub_126F7(dword_53AB1+(DX),dword_53AB5+(DY),N)
#ifdef OVERLAY_DISPATCH_1
#define BEGIN(N) case N:
#define NEXT(N) break;case N:
#define END break;}
#else
#define BEGIN(N) if(dword_51A83==N){
#ifdef OVERLAY_DISPATCH_2
#define NEXT(N) return;}if(dword_51A83==N){
#else
#define NEXT(N) }else if(dword_51A83==N){
#endif
#define END }
#endif
void sub_122DC(void)
{
#ifdef OVERLAY_DISPATCH_1
    switch(dword_51A83){
#endif
    BEGIN(1) AT(0,0,0);
    NEXT(2) AT(0,0,1);
    NEXT(3)
        AT(0,0,14);AT(0,-1,2);AT(-1,0,3);AT(1,0,4);AT(0,1,5);
    NEXT(4)
        AT(0,0,1);AT(0,-2,2);AT(-2,0,3);AT(2,0,4);AT(0,2,5);
        AT(-1,-1,6);AT(1,-1,7);AT(-1,1,8);AT(1,1,9);
        AT(0,-1,10);AT(-1,0,11);AT(1,0,12);AT(0,1,13);
    NEXT(5)
        AT(0,0,1);AT(0,-3,2);AT(-3,0,3);AT(3,0,4);AT(0,3,5);
        AT(-1,-2,6);AT(-2,-1,6);AT(1,-2,7);AT(2,-1,7);
        AT(-1,2,8);AT(-2,1,8);AT(1,2,9);AT(2,1,9);
        AT(0,-2,10);AT(-2,0,11);AT(2,0,12);AT(0,2,13);
        AT(-1,-1,15);AT(1,-1,16);AT(-1,1,17);AT(1,1,18);
    NEXT(6)
        dword_53A51[(dword_53AB5*dword_53AC1+dword_53AB1)*4+7]=0;
    END
}
