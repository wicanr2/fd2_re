/* sub_1DB65完整857-byte目標的C候選。沿用既有狀態／標記證據。
 * 保留record+5整位元組寫入、word+64與原始畫面指標公式。
 * 局部型別、回傳型別與C宣告只是產碼表示，不推定作者原始碼。
 */
#ifdef RECORD_RESOLVE_6
#define RECORD_RESOLVE_5
#define RESOLVE_EAX_INPUT
#define RESOLVE_FLAG_COMPARE
#endif
#ifdef RECORD_RESOLVE_7
#define RECORD_RESOLVE_5
#define RESOLVE_EAX_INPUT
#endif
#if defined(RECORD_RESOLVE_8) || defined(RECORD_RESOLVE_9)
#define RECORD_RESOLVE_5
#define RESOLVE_EAX_INPUT
#define RESOLVE_DIRECT_RETURN
#define RESOLVE_FRAME_INDEX
#endif
#if defined(RECORD_RESOLVE_9) || defined(RECORD_RESOLVE_10)
#define RECORD_RESOLVE_5
#define RESOLVE_FLAG_BITFIELD
#define RESOLVE_FRAME_INDEX
#endif
typedef struct {unsigned char *points[30];unsigned char *buffer;} RawResolve124;
typedef struct {unsigned char bit0:1;unsigned char :7;} RawFlag1;
typedef char RawFlagMustBe1[sizeof(RawFlag1)==1?1:-1];
typedef char RawResolveMustBe124[sizeof(RawResolve124)==124?1:-1];
extern unsigned char *dword_53A45,*dword_53A49,*dword_53A81;
extern void *dword_53EEC;
extern int dword_53BEB,dword_53AA9,dword_53AAD,dword_51A87,dword_51A8B;
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned);
extern void free(void *),sub_127E0(int),sub_129EC(void),sub_127A9(void);
extern void sub_11EEE(void *,int,int,int,int,int),sub_11EB0(void *,int,void *,int,int,int);
extern void sub_17AA9(int),sub_25A96(void *,int,int),sub_4E85B(void *,void *,int);
extern int sub_11CAC(int);
#if defined(RECORD_RESOLVE_3) || defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
/* 保守允許修改，不當作原作實際clobber或作者ABI。 */
#pragma aux sub_4E85B modify exact [eax ebx ecx edx];
#endif
#if defined(RECORD_RESOLVE_2) || defined(RECORD_RESOLVE_3) || defined(RECORD_RESOLVE_4)
#define RAW64 (*(unsigned *)(record+64)&65535U)
#else
#define RAW64 (*(unsigned short *)(record+64))
#endif

#ifdef RESOLVE_EAX_INPUT
int sub_1DB65(int);
/* 只將未在零筆路徑定義的EAX顯式攜入，不推定作者參數。 */
#pragma aux sub_1DB65 parm caller [eax] value [eax] modify exact [eax ecx edx gs];
int sub_1DB65(int result)
#elif defined(RECORD_RESOLVE_1)
void sub_1DB65(void)
#else
int sub_1DB65(void)
#endif
{
    RawResolve124 local;
    unsigned char *record,*saved,*frame;
    int count,index,phase,x,y;
#ifdef RESOLVE_FRAME_INDEX
    int image;
#endif
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
    /* 零筆路徑的回傳暫存器未在原始本體定義，不猜補零。 */
#ifndef RESOLVE_EAX_INPUT
    int result;
#endif
    int hp;
#endif
    count=0;index=0;goto collect_check;
collect_next:
    ++index;
collect_check:
    if(index>=dword_53BEB)goto collected;
    record=dword_53A45+index*80;x=record[0];y=record[1];
#ifdef RESOLVE_FLAG_BITFIELD
    if(((RawFlag1 *)(record+5))->bit0)goto collect_next;
#elif defined(RESOLVE_FLAG_COMPARE)
    if((record[5]&1)!=0)goto collect_next;
#else
    if(record[5]&1)goto collect_next;
#endif
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
    hp=(int)RAW64;if(hp)goto collect_next;
#else
    if(RAW64)goto collect_next;
#endif
    if(x<dword_53AA9-1)goto collect_next;
    if(x>dword_53AA9+dword_51A87)goto collect_next;
    if(y<dword_53AAD-1)goto collect_next;
    if(y>dword_53AAD+dword_51A8B+1)goto collect_next;
    local.points[count]=dword_53A49+0x8088+(x-1-dword_53AA9)*24+(y-1-dword_53AAD)*10944-0xab0;
    ++count;goto collect_next;
collected:
    if(count==0){
        index=0;goto empty_check;
empty_next:
        ++index;
empty_check:
        if(index>=dword_53BEB)goto finished;
        record=dword_53A45+index*80;
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
        hp=(int)RAW64;if(hp)goto empty_next;
#else
        if(RAW64)goto empty_next;
#endif
        record[5]=1;goto empty_next;
    }else{
        phase=0;goto turn_check;
turn_present:
        sub_129EC();sub_11EB0((void *)0xA0504,320,dword_53A49+0x8088,456,312,192);
        sub_17AA9(1);++phase;
turn_check:
        if(phase>=13)goto mark_start;
        sub_11EEE(dword_53A49+0x8088,456,13,8,dword_53AA9,dword_53AAD);
        index=0;goto draw_check;
draw_next:
        ++index;
draw_check:
        if(index>=dword_53BEB)goto turn_present;
        record=dword_53A45+index*80;
#ifdef RESOLVE_FLAG_BITFIELD
        if(((RawFlag1 *)(record+5))->bit0)goto draw_next;
#elif defined(RESOLVE_FLAG_COMPARE)
        if((record[5]&1)!=0)goto draw_next;
#else
        if(record[5]&1)goto draw_next;
#endif
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
        hp=(int)RAW64;if(!hp)record[3]=(unsigned char)(phase%4);
#else
        if(!RAW64)record[3]=(unsigned char)(phase%4);
#endif
        sub_127E0(index);goto draw_next;
mark_start:
        index=0;goto mark_check;
mark_next:
        ++index;
mark_check:
        if(index>=dword_53BEB)goto effect_start;
        record=dword_53A45+index*80;
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
        hp=(int)RAW64;if(hp)goto mark_next;
#else
        if(RAW64)goto mark_next;
#endif
        record[5]=1;goto mark_next;
effect_start:
        local.buffer=(unsigned char *)malloc(0x25680);
        sub_11EEE(local.buffer+0x8088,456,13,8,dword_53AA9,dword_53AAD);
        saved=dword_53A49;dword_53A49=local.buffer;sub_127A9();dword_53A49=saved;
        sub_25A96(dword_53EEC,3,1);phase=0;goto first_check;
first_draw:
#ifdef RESOLVE_FRAME_INDEX
        image=phase+68;frame=dword_53A81+*(unsigned *)(dword_53A81+image*4+6);
#else
        frame=dword_53A81+*(unsigned *)(dword_53A81+(phase+68)*4+6);
#endif
        sub_4E85B(local.points[index],frame,456);++index;
first_target_check:
        if(index<count)goto first_draw;
        sub_11EB0((void *)0xA0504,320,dword_53A49+0x8088,456,312,192);sub_17AA9(1);++phase;
first_check:
        if(phase>=6)goto second_start;
        index=0;goto first_target_check;
second_start:
        phase=6;goto second_check;
second_draw:
#ifdef RESOLVE_FRAME_INDEX
        image=phase+68;frame=dword_53A81+*(unsigned *)(dword_53A81+image*4+6);
#else
        frame=dword_53A81+*(unsigned *)(dword_53A81+(phase+68)*4+6);
#endif
        sub_4E85B(local.points[index],frame,456);++index;
second_target_check:
        if(index<count)goto second_draw;
        sub_11EB0((void *)0xA0504,320,dword_53A49+0x8088,456,312,192);sub_17AA9(1);++phase;
second_check:
        if(phase>=12)goto effect_done;
        memmove(dword_53A49,local.buffer,0x25680);index=0;goto second_target_check;
effect_done:
        free(local.buffer);
#ifdef RESOLVE_DIRECT_RETURN
        return sub_11CAC(0);
#elif defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
        result=sub_11CAC(0);
#else
        sub_11CAC(0);
#endif
    }
finished:
#if defined(RECORD_RESOLVE_4) || defined(RECORD_RESOLVE_5)
    return result;
#else
    ;
#endif
}
