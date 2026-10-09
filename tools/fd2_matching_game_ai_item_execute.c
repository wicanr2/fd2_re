/* sub_15055完整C候選。原始欄位、36-byte局部與既有target-list consumer保留。
 * 聚合／volatile只作產碼表示，不推定作者宣告或新增效果語意。
 */
#if defined(AI_ITEM_EXEC_8) || defined(AI_ITEM_EXEC_9)
#define AI_ITEM_EXEC_6
#endif
#ifdef AI_ITEM_EXEC_6
#define AI_ITEM_EXEC_4
#define AI_ITEM_EXEC_2
#endif
#ifdef AI_ITEM_EXEC_7
#define AI_ITEM_EXEC_3
#define AI_ITEM_EXEC_2
#endif
#ifdef AI_ITEM_EXEC_1
typedef struct {unsigned char raw[32];volatile unsigned char byte32;unsigned char tail[3];} RawFrame36;
#define BYTE32 local.byte32
#else
typedef struct {unsigned char raw[36];} RawFrame36;
#define BYTE32 local.raw[32]
#endif
typedef char RawFrameMustBe36[sizeof(RawFrame36)==36?1:-1];
extern unsigned char *dword_53A45;
extern void *dword_53A51;
extern int dword_53C3F,dword_53C57,dword_53C37,dword_53C3B,dword_51A83;
extern int dword_53AB1,dword_53AB5,dword_53AC1,dword_53AC5,dword_53C1F,dword_53EC8;
extern int sub_1B722(int,int);
extern unsigned char *sub_4E56C(int);
extern int sub_149F8(int,int,void *,int,int,int,int),sub_14818(int,int,void *,int,int,int);
extern void sub_12D7B(int),sub_12CEA(int,int),sub_4DBFC(void *),j___delay(unsigned);
extern void sub_1F04A(int,int),sub_11CAC(int),sub_17AA9(int),sub_28784(int),sub_2189A(int,int,int);
extern void sub_11DF2(int,int,int),sub_20C6F(int,int,int,void *),sub_134E4(void);
#if defined(AI_ITEM_EXEC_6) || defined(AI_ITEM_EXEC_7)
/* 4E56C實際只改EAX／EDX；此處保守允許修改，不當作作者ABI。 */
#pragma aux sub_4E56C modify exact [eax ebx ecx edx];
#endif
#if defined(AI_ITEM_EXEC_4) || defined(AI_ITEM_EXEC_5)
/* 保守允許修改；原版4DBFC實際保存EBX／ESI／EDI，不推定作者ABI。 */
#pragma aux sub_4DBFC modify exact [eax ebx ecx edx];
#endif
#if defined(AI_ITEM_EXEC_3) || defined(AI_ITEM_EXEC_4)
#define READ17 ((int)*(volatile unsigned char *)(item+17))
#else
#define READ17 item[17]
#endif

int sub_15055(int actor,int mode)
{
    RawFrame36 local;
    unsigned char *record=dword_53A45+actor*80,*item;
    int option,count,step;
    dword_53C57=sub_1B722(actor,dword_53C3F);item=sub_4E56C(dword_53C57);
    if(mode==0){if(READ17==0)option=1;else option=0;}else option=READ17;
    sub_12D7B(actor);BYTE32=item[16];
    if(BYTE32>15)count=sub_149F8(dword_53C37,dword_53C3B,local.raw,record[0],record[1],BYTE32-16,0);
    else count=sub_14818(dword_53C37,dword_53C3B,local.raw,item[18],0,option);
    sub_4DBFC(dword_53A51);j___delay(200);dword_51A83=item[18]+2;
    if(BYTE32<16){sub_12CEA(dword_53C37,dword_53C3B);goto execute;}
    sub_1F04A(actor,local.raw[0]);sub_11CAC(1);sub_17AA9(1);sub_17AA9(2);sub_28784(actor);
    sub_2189A(actor,80,-4);j___delay(200);step=64;goto fade_check;
fade_next:
    sub_11DF2(0,255,step);j___delay(4);--step;
fade_check:
    if(step>=0)goto fade_next;
    dword_51A83=6;BYTE32-=16;
#ifdef AI_ITEM_EXEC_2
    {
        int delta=dword_53C37-dword_53AB1;
        dword_53C37=dword_53AB1+BYTE32*delta;
    }
#else
    dword_53C37=dword_53AB1+(dword_53C37-dword_53AB1)*BYTE32;
#endif
    if(dword_53C37>=dword_53AC1)dword_53C37=dword_53AC1-1;
    else if(dword_53C37<0)dword_53C37=0;
#ifdef AI_ITEM_EXEC_8
    step=dword_53C3B-dword_53AB5;step*=BYTE32;dword_53C3B=dword_53AB5+step;
#elif defined(AI_ITEM_EXEC_9)
    option=dword_53C3B-dword_53AB5;option*=BYTE32;dword_53C3B=dword_53AB5+option;
#elif defined(AI_ITEM_EXEC_6) || defined(AI_ITEM_EXEC_7)
    dword_53C3B=(dword_53C3B-dword_53AB5)*BYTE32+dword_53AB5;
#else
    dword_53C3B=dword_53AB5+(dword_53C3B-dword_53AB5)*BYTE32;
#endif
    if(dword_53C3B>=dword_53AC5)dword_53C3B=dword_53AC5-1;
    else if(dword_53C3B<0)dword_53C3B=0;
    dword_53C1F=0;sub_12CEA(dword_53C37,dword_53C3B);dword_51A83=0;
    step=1;goto flash_check;
flash_next:
    dword_53C1F=step;sub_11CAC(1);sub_17AA9(1);++step;
flash_check:
    if(step<9)goto flash_next;
    sub_4DBFC(dword_53A51);sub_17AA9(2);sub_12D7B(local.raw[0]);
execute:
    sub_20C6F(actor,dword_53C3F,count,local.raw);sub_134E4();dword_53EC8=0;return 0;
}
