/* 完整函式的運算元資料流候選。固定IDA 9.4原名、位址及caller見主收據。
 * 保留原始資料寬度、呼叫及操作數；局部宣告只作產碼表示。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A81,*dword_53A85;
extern int dword_53C67,dword_53EC8;
extern unsigned char byte_540FC,byte_540FD;
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];
extern const char aDatoDat[];
extern void *sub_111BA(const char *,void *,int);
extern void sub_168B6(void *,int,int,int,int,int),sub_4E8AF(void *,void *,int),sub_17FC0(int,void *);
extern void sub_1C4CC(int,int,int,void *),sub_1C2DA(int,int,int,void *),sub_1E1DC(int),sub_11CAC(int);
extern int sub_1C916(int,int);
extern void sub_1E0DB(int,int,int),sub_2935B(void *,int,int,int,int);

#if defined(OPERAND_PANEL_0) || defined(OPERAND_PANEL_1) || defined(OPERAND_PANEL_2)
#ifdef OPERAND_PANEL_0
#define ENTRY_ABI
#else
#define ENTRY_ABI __cdecl
#endif
void ENTRY_ABI sub_17EEF(int unit,unsigned char *dest)
{
#ifdef OPERAND_PANEL_2
    unsigned source;
#define offset source
#define RAW_ADDRESS(P) ((unsigned)(P))
#else
    unsigned char *source;
    int offset;
#define RAW_ADDRESS(P) (P)
#endif
    dword_53C67=3208;
#ifdef OPERAND_PANEL_0
    offset=unit*80;
#elif defined(OPERAND_PANEL_1)
    offset=unit*80;
#else
    offset=(unsigned)unit;offset=offset+(offset<<2);offset<<=4;
#endif
    dword_53A85=sub_111BA(aDatoDat,dword_53A85,(dword_53A45+offset)[7]);
    source=RAW_ADDRESS(dword_53A85+(int)dword_53A85[0]);
    sub_168B6(dest,320,5,7,5,5);sub_4E8AF(dest+dword_53C67,(unsigned char *)source,320);
    source=RAW_ADDRESS(dword_53A81);source+=*(int *)((unsigned char *)source+86);sub_4E8AF(dest+2332,(unsigned char *)source,320);
    source=RAW_ADDRESS(dword_53A81);source+=*(int *)((unsigned char *)source+90);sub_4E8AF(dest+30085,(unsigned char *)source,320);
    sub_17FC0(unit,dest);
}
#undef offset
#undef RAW_ADDRESS
#undef ENTRY_ABI
#endif

#if defined(OPERAND_STATE_0) || defined(OPERAND_STATE_1) || defined(OPERAND_STATE_2)
#ifdef OPERAND_STATE_0
#define ENTRY_ABI
#else
#define ENTRY_ABI __cdecl
#endif
void ENTRY_ABI sub_22AF6(int unit,int mode,int count,unsigned char *targets,int field)
{
    int index,target,offset,value,result,present;
    int kind;
    unsigned char *record,*marker,*target_pointer;
    sub_1C4CC(unit,mode,count,targets);sub_1C2DA(unit,mode,count,targets);
    index=0;goto check;
missing:
    sub_1E1DC(*target_pointer);
next:
    ++index;
check:
    if(index>=count)goto done;
    target=targets[index];offset=target*80;record=dword_53A45+offset;
    value=record[33];
#ifdef OPERAND_STATE_0
    kind=*(volatile unsigned char *)(record+32);if(kind>8&&kind<25)value+=30;
#elif defined(OPERAND_STATE_1)
    kind=*(volatile unsigned char *)(record+32);if(kind>8&&kind<25)value+=30;
#else
    {const unsigned read_kind=record[32];if((int)read_kind>8&&(int)read_kind<25)value+=30;}
#endif
    marker=record+field;
    present=*(volatile unsigned char *)marker;target_pointer=targets+index;
    if(!present)goto missing;
    result=sub_1C916(*target_pointer,10);sub_1E0DB(result,105,*target_pointer);
    *marker=0;dword_53EC8+=value*4;goto next;
done:
    sub_11CAC(0);
}
#undef kind
#undef ENTRY_ABI
#endif

#if defined(OPERAND_IDLE_0) || defined(OPERAND_IDLE_1) || defined(OPERAND_IDLE_2)
#ifdef OPERAND_IDLE_0
#define ENTRY_ABI
#else
#define ENTRY_ABI __cdecl
#endif
void ENTRY_ABI sub_2B9A1(unsigned char *data,int active,int x,int y)
{
    int offset,delay;
    int frame,frame_count;
    if(!active){byte_540FC=0;byte_540FD=0;return;}
    sub_2935B(data,byte_540FD,x,y,active);
    offset=*(int *)(data+byte_540FD*4+8);delay=data[offset+6];
    ++byte_540FC;
#define first_counter ((int)byte_540FC)
    if(first_counter>=delay){
        byte_540FC=0;++byte_540FD;
#ifdef OPERAND_IDLE_0
        offset=*(volatile unsigned char *)&byte_540FD;
        delay=*(volatile unsigned char *)data;
        if(offset>=delay)byte_540FD=0;
#elif defined(OPERAND_IDLE_1)
        offset=byte_540FD;
        delay=*(volatile unsigned char *)data;
        if(offset>=delay)byte_540FD=0;
#else
        offset=byte_540FD;
        delay=*(volatile unsigned char *)data;
        if(offset>=delay)byte_540FD=0;
#endif
    }
}
#undef first_counter
#undef ENTRY_ABI
#endif

#if defined(OPERAND_SLOT_0) || defined(OPERAND_SLOT_1) || defined(OPERAND_SLOT_2)
#ifdef OPERAND_SLOT_0
#define ENTRY_ABI
#else
#define ENTRY_ABI __cdecl
#endif
unsigned ENTRY_ABI sub_1B722(int unit,int slot)
{
#ifdef OPERAND_SLOT_2
    unsigned offset=unit;
    offset*=80;offset+=(unsigned)dword_53A45;
    offset=slot*2+offset;offset+=11;
    return *(unsigned char *)offset;
#else
    unsigned char *record,*cell;
    int offset=unit;
    offset*=80;record=dword_53A45;record+=offset;
    cell=record+2*slot;cell+=11;
    return *cell;
#endif
}
#undef ENTRY_ABI
#endif
