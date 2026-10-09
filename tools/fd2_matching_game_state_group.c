/* 既有狀態處理的暫存值及四函式共用尾段C候選。
 * 固定IDA 9.4原名、位址、bytes與caller見主收據；既有遊戲語意不重開。
 */
extern unsigned char * volatile dword_53A45;
extern int dword_53EC4,dword_53EC8;
extern void sub_1C4CC(int,int,int,void *),sub_1C2DA(int,int,int,void *),sub_1E1DC(int);
extern int sub_1C916(int,int);
extern void sub_1E0DB(int,int,int),sub_1CA89(int,int),sub_1DF58(void);
extern int sub_11CAC(int);
#if defined(STATE_GROUP_0) || defined(STATE_GROUP_1) || defined(STATE_GROUP_2) || defined(STATE_GROUP_3) || defined(STATE_GROUP_4) || defined(STATE_GROUP_5) || defined(STATE_GROUP_6) || defined(STATE_GROUP_7) || defined(STATE_GROUP_8) || defined(STATE_GROUP_9) || defined(STATE_GROUP_10)
#define WITH_STATE_GROUP
#endif
#ifdef STATE_GROUP_4
#define STATE_VALUE_11
#elif defined(STATE_GROUP_5)
#define STATE_VALUE_12
#elif defined(STATE_GROUP_6)
#define STATE_VALUE_13
#elif defined(STATE_GROUP_7)
#define STATE_VALUE_14
#elif defined(STATE_GROUP_8)
#define STATE_VALUE_15
#elif defined(STATE_GROUP_9)
#define STATE_VALUE_16
#elif defined(STATE_GROUP_10)
#define STATE_VALUE_17
#endif
#if defined(STATE_VALUE_3) || defined(STATE_GROUP_2)
#define ENTRY_RESULT int
#else
#define ENTRY_RESULT void
#endif
ENTRY_RESULT sub_22AF6(int,int,int,unsigned char *,int);
#ifdef STATE_GROUP_3
extern int fd2_state_entry_view(int,int,int,unsigned char *,int);
#pragma aux fd2_state_entry_view "sub_22AA8" parm caller [] value [eax] modify exact [eax ecx edx];
#define WRAPPER_RESULT int
#else
#define WRAPPER_RESULT void
#endif

#ifdef WITH_STATE_GROUP
void sub_22AA8(int,int,int,unsigned char *,int);
WRAPPER_RESULT sub_22A85(int first,int second,unsigned char *third)
{
#ifdef STATE_GROUP_3
    return fd2_state_entry_view(first,20,second,third,37);
#else
    sub_22AA8(first,20,second,third,37);
#endif
}
void sub_22AA8(int first,int mode,int count,unsigned char *targets,int field)
{
    dword_53EC4=0;sub_1CA89(first,mode);sub_22AF6(first,mode,count,targets,field);
    if(dword_53EC4)sub_1DF58();
}
#endif

ENTRY_RESULT sub_22AF6(int unit,int mode,int count,unsigned char *targets,int field)
{
    int index,target,offset,value,result,present;
#if defined(STATE_VALUE_1) || defined(STATE_GROUP_1)
    register int kind;
#elif defined(STATE_VALUE_2)
    unsigned char kind;
#elif defined(STATE_VALUE_8)
    short kind;
#elif defined(STATE_VALUE_9)
    long kind;
#elif defined(STATE_VALUE_10)
    unsigned short kind;
#else
    int kind;
#endif
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
#if defined(STATE_VALUE_11)
    kind=*(volatile unsigned char *)(record+32);if(kind>8&&kind<25)value+=30;
#elif defined(STATE_VALUE_12)
    if((int)record[32]>8&&(int)record[32]<25)value+=30;
#elif defined(STATE_VALUE_13)
    if((int)record[32]>8){if((int)record[32]<25)value+=30;}
#elif defined(STATE_VALUE_14)
    kind=record[32];if(kind<=8)goto kind_done;if(kind>=25)goto kind_done;value+=30;
kind_done:
    ;
#elif defined(STATE_VALUE_15)
    kind=*(unsigned *)(record+32)&255;if(kind>8&&kind<25)value+=30;
#elif defined(STATE_VALUE_16)
    {struct RawFields {unsigned char prefix[32],raw32,raw33;};
     kind=((struct RawFields *)record)->raw32;if(kind>8&&kind<25)value+=30;}
#elif defined(STATE_VALUE_17)
    {struct RawBits {unsigned prefix[8];unsigned raw32:8;unsigned raw33:8;};
     kind=((struct RawBits *)record)->raw32;if(kind>8&&kind<25)value+=30;}
#elif defined(STATE_VALUE_5)
    result=record[32];if(result>8&&result<25)value+=30;
#elif defined(STATE_VALUE_6)
    present=record[32];if(present>8&&present<25)value+=30;
#elif defined(STATE_VALUE_7)
    offset=record[32];if(offset>8&&offset<25)value+=30;
#elif defined(STATE_VALUE_4)
    if((kind=record[32])>8&&kind<25)value+=30;
#else
    kind=record[32];if((int)kind>8&&(int)kind<25)value+=30;
#endif
    marker=record+field;present=*(volatile unsigned char *)marker;target_pointer=targets+index;
    if(!present)goto missing;
    result=sub_1C916(*target_pointer,10);sub_1E0DB(result,105,*target_pointer);
    *marker=0;dword_53EC8+=value*4;goto next;
done:
#if defined(STATE_VALUE_3) || defined(STATE_GROUP_2)
    return sub_11CAC(0);
#else
    sub_11CAC(0);
#endif
}

#ifdef WITH_STATE_GROUP
WRAPPER_RESULT sub_22BC6(int first,int second,unsigned char *third)
{
#ifdef STATE_GROUP_3
    return fd2_state_entry_view(first,21,second,third,38);
#else
    sub_22AA8(first,21,second,third,38);
#endif
}
#endif
