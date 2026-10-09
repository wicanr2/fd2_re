/* 隊伍服務、選擇面板與既有輸入輪詢C候選。固定IDA 9.4原名、位址、bytes與caller見主收據。
 * 獨立暫存值及共用計數值只作產碼表示，不推定作者宣告或新的玩法。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7,*dword_53A61;
extern int dword_53C57,dword_53BFB,dword_53BDF,dword_53AD9,dword_53A7D;
extern unsigned char byte_526A7[],byte_5265D;
extern const char aRb_8[],aFdiconB24_2[];
extern void free(void *);
extern void *fopen(const char *,const char *);
extern int fclose(void *),sub_31793(void *,void *),sub_311DC(int,void *,void *);
extern int sub_19953(void),sub_16C57(int),sub_11019(int,void *),sub_31860(int,int);
extern void sub_2D31B(void),sub_1956B(int),sub_16559(int),sub_197E5(void);
#if defined(CLASS_STACK_0) || defined(CLASS_STACK_4)
extern void __cdecl sub_4E031(void);
#else
extern void sub_4E031(void);
#ifdef CLASS_STACK_1
#pragma aux sub_4E031 modify exact [ax];
#elif defined(CLASS_STACK_2)
#pragma aux sub_4E031 modify exact [eax];
#else
#pragma aux sub_4E031 modify exact [eax ecx edx];
#endif
#endif
extern unsigned char *sub_4E48D(int);
extern void sub_25977(int,int),sub_1B8E7(int,int),sub_2A2E8(int,int),sub_31602(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A7D,INDEX,(void *)0xa94cc,320,205,76,74,19,1)
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];

/* 既有呼叫的暫存器快取視角，不定義inline code，不推定原作者原型。
 * 原名／位址與callee保存證據見主收據；實際呼叫及完整code須相同。
 */
#ifdef CLASS_STACK_4
extern int fd2_close_status_view(int);
#pragma aux fd2_close_status_view "sub_2D31B" parm caller [esi] value [esi] modify exact [eax ecx edx];
extern int fd2_icon_cache_view(int,int,void *);
#pragma aux fd2_icon_cache_view "sub_11019" parm caller [esi] [] value [esi] modify exact [eax ecx edx esi];
#endif
#ifdef SELECTOR_PARAM_4
extern int fd2_key_cache_view(int,void *,int);
#pragma aux fd2_key_cache_view "sub_2D85F" parm caller [ebx] [esi] [] value [eax] modify exact [eax ecx edx];
#endif

#if defined(CLASS_STACK_0) || defined(CLASS_STACK_1) || defined(CLASS_STACK_2) || defined(CLASS_STACK_3) || defined(CLASS_STACK_4)
void sub_31385(void)
{
#if defined(CLASS_STACK_2) || defined(CLASS_STACK_3)
    int scalar_item;
    struct {unsigned char classes[32],units[32];} local;
#define ITEM_VALUE (*(int *)&scalar_item)
#else
    struct {unsigned char classes[32],units[32];int item;} local;
#define ITEM_VALUE local.item
#endif
    unsigned char *record,*data;
    void *file;
    int count,response,unit,slot;
#if defined(CLASS_STACK_1) || defined(CLASS_STACK_3)
#define ICON_INDEX count
#else
    int index;
#define ICON_INDEX index
#endif
again:
    count=sub_31793(local.units,local.classes);
    if(!count){sub_1956B(byte_5265D);MESSAGE(591);sub_16559(count);sub_16C57(count);sub_2D31B();goto done;}
    sub_1956B(byte_5265D);MESSAGE(592);sub_16559(0);sub_16C57(1);sub_2D31B();sub_4E031();
#ifdef CLASS_STACK_4
    if(fd2_close_status_view(sub_311DC(count,local.units,local.classes))==-1)goto done;
#else
    count=sub_311DC(count,local.units,local.classes);sub_2D31B();if(count==-1)goto done;
#endif
    sub_1956B(byte_5265D);unit=local.units[dword_53C57];ITEM_VALUE=local.classes[dword_53C57];
    record=dword_53A45+unit*80;dword_53AD9=(int)record[7]+1;MESSAGE(594);sub_4E031();
    response=sub_19953();sub_197E5();sub_2D31B();if(response==-1||dword_53C57!=0)goto again;
    if(ITEM_VALUE==52)slot=sub_31860(unit,90);
    else if(ITEM_VALUE>=50)slot=sub_31860(unit,byte_526A7[record[7]]);
    else goto change;
    sub_1B8E7(unit,slot);
change:
    sub_25977(16,1);sub_2A2E8(unit,ITEM_VALUE);sub_25977(11,0);
    data=sub_4E48D(ITEM_VALUE);record[32]=data[0];record[7]=ITEM_VALUE;
    if(dword_53A61)free(dword_53A61);
    file=fopen(aFdiconB24_2,aRb_8);dword_53BDF=0;ICON_INDEX=0;goto icon_condition;
icon_next:
#ifdef CLASS_STACK_4
    ICON_INDEX=fd2_icon_cache_view(ICON_INDEX,(record=dword_53BF7,record[ICON_INDEX*80+7]),file);
#else
    sub_11019((record=dword_53BF7,record[ICON_INDEX*80+7]),file);
#endif
    ++ICON_INDEX;
icon_condition:
    if(ICON_INDEX<dword_53BFB)goto icon_next;
    fclose(file);sub_31602(unit);sub_4E031();goto again;
done:
    return;
}
#undef ITEM_VALUE
#undef ICON_INDEX
#endif

#if defined(SELECTOR_PARAM_0) || defined(SELECTOR_PARAM_1) || defined(SELECTOR_PARAM_2) || defined(SELECTOR_PARAM_3) || defined(SELECTOR_PARAM_4) || defined(SELECTOR_PARAM_5)
extern int sub_2D85F(int);
extern void *dword_53EEC;
extern int dword_5412F;
extern void sub_25A96(void *,int,int),sub_2E19B(void),sub_2E26C(void);
#if defined(SELECTOR_PARAM_2)
extern void sub_2DC55(int,unsigned,int,void *,int);
#elif defined(SELECTOR_PARAM_3)
extern void sub_2DC55(int,int,int,void *,int);
#else
extern void sub_2DC55(int,void *,int,void *,int);
#endif
#ifdef SELECTOR_PARAM_5
int sub_2DF6B(void *raw_count,int raw_data,unsigned char event)
#define LIMIT ((int)(unsigned)raw_count)
#define ITEMS ((void *)raw_data)
#elif defined(SELECTOR_PARAM_2)
int sub_2DF6B(int count,unsigned raw_data,unsigned char event)
#define LIMIT count
#define ITEMS raw_data
#elif defined(SELECTOR_PARAM_3)
int sub_2DF6B(int count,int raw_data,unsigned char event)
#define LIMIT count
#define ITEMS raw_data
#else
int sub_2DF6B(int count,void *data,unsigned char event)
#if defined(SELECTOR_PARAM_1) || defined(SELECTOR_PARAM_4)
#define LIMIT count
#define ITEMS data
#endif
#endif
{
#ifdef SELECTOR_PARAM_0
    register int limit=count;
    register void *items=data;
#define LIMIT limit
#define ITEMS items
#endif
    int result=0,key;
again:
#ifdef SELECTOR_PARAM_4
    key=fd2_key_cache_view(count,data,1);
#else
    key=sub_2D85F(1);
#endif
    if(key==77){
        if(LIMIT-1==dword_53C57)goto check;
        sub_25A96(dword_53EEC,0,1);++dword_53C57;
down_check:
        if(dword_53C57-dword_5412F>=6){dword_5412F+=2;sub_2E19B();}
draw:
        sub_2DC55(LIMIT,ITEMS,dword_53C57,(void *)0xa0000,(int)event);goto check;
    }
    if(key==75){
        if(dword_53C57==0)goto check;
        sub_25A96(dword_53EEC,0,1);--dword_53C57;
up_check:
        if(dword_53C57<dword_5412F){dword_5412F-=2;sub_2E26C();}
        goto draw;
    }
    if(key==72){
        if(dword_53C57<2)goto check;
        sub_25A96(dword_53EEC,0,1);dword_53C57-=2;goto up_check;
    }
    if(key==80){
        if(LIMIT-2<=dword_53C57)goto check;
        sub_25A96(dword_53EEC,0,1);dword_53C57+=2;goto down_check;
    }
    if(key==28||key==57)result=1;
    else if(key==1)result=-1;
check:
    if(!result)goto again;
    return result;
}
#undef LIMIT
#undef ITEMS
#endif

#if defined(INPUT_ESI_0) || defined(INPUT_ESI_1) || defined(INPUT_ESI_2)
extern int sub_10620(void);
extern void sub_4DFCC(void),sub_11CAC(int);
extern unsigned char word_53A8D[];
extern int __cdecl int386(int,void *,void *);
#ifdef INPUT_ESI_0
int sub_12DAC(void)
#else
int sub_12DAC(int);
#ifdef INPUT_ESI_1
#pragma aux sub_12DAC parm caller [esi] value [eax] modify exact [eax ecx edx gs];
#else
#pragma aux sub_12DAC parm caller [esi] value [eax] modify exact [eax ebx ecx edx gs];
#endif
int sub_12DAC(int previous)
#endif
{
#ifdef INPUT_ESI_0
    /* 原始ESI在首次更新前已被讀取，本控制不加入猜測初值。 */
    int previous;
#endif
    int key;
again:
    if(sub_10620())goto ready;
    sub_4DFCC();if(previous==*(volatile short *)0x46c)goto again;
    sub_11CAC(0);previous=*(volatile short *)0x46c;goto again;
ready:
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);
    key=word_53A8D[1];if(key==224||key==82)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#endif
