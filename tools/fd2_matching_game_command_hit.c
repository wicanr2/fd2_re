/* sub_1C75E完整C候選。保留原始112-byte局部表、raw職業索引與signed算式。
 * 既有規則語意不重開；聚合、volatile與pragma不推定作者原始C。
 */
#ifdef COMMAND_HIT_5
#define COMMAND_HIT_3
#endif
#if defined(COMMAND_HIT_6) || defined(COMMAND_HIT_7) || defined(COMMAND_HIT_8) || defined(COMMAND_HIT_9)
#define COMMAND_HIT_4
#endif
typedef struct {int values[28];} RawTable112;
typedef struct {unsigned char raw[80];} RawActor80;
typedef struct {RawTable112 table;int threshold;} RawHit116;
typedef char RawHitMustBe116[sizeof(RawHit116)==116?1:-1];
extern const int unk_51F96[28];
extern unsigned char *dword_53A45,*sub_4E516(int);
extern int sub_4E893(void),sub_1F183(int),sub_1C81F(int,int);
extern void *memcpy(void *,const void *,unsigned);
#pragma intrinsic(memcpy)
#ifdef COMMAND_HIT_1
#pragma aux sub_4E516 modify exact [eax ebx ecx edx];
#endif

int sub_1C75E(int actor,int command)
{
#ifdef COMMAND_HIT_3
    int local[29];
#define VALUES local
#define THRESHOLD local[28]
#else
    RawHit116 local;
#define VALUES local.table.values
#define THRESHOLD local.threshold
#endif
    int rawclass,amount;
    unsigned char *info;
#if defined(COMMAND_HIT_4) || defined(COMMAND_HIT_5)
    *(RawTable112 *)VALUES=*(const RawTable112 *)unk_51F96;
#else
    memcpy(VALUES,unk_51F96,112);
#endif
#ifdef COMMAND_HIT_8
    info=dword_53A45+actor*80;rawclass=info[32];
#elif defined(COMMAND_HIT_9)
    rawclass=((RawActor80 *)dword_53A45)[actor].raw[32];
#elif defined(COMMAND_HIT_6)
    info=(unsigned char *)(actor<<2);info+=(unsigned)actor;
    info=(unsigned char *)((unsigned)info<<4);
    rawclass=dword_53A45[(unsigned)info+32];
#elif defined(COMMAND_HIT_7)
    info=(unsigned char *)(actor*80);rawclass=dword_53A45[(unsigned)info+32];
#else
    rawclass=dword_53A45[actor*80+32];
#endif
    info=sub_4E516(command);
#ifdef COMMAND_HIT_2
    amount=*(volatile short *)info*VALUES[rawclass-1]/10;
#else
    amount=*(short *)info*VALUES[rawclass-1]/10;
#endif
    THRESHOLD=info[2];
    if(command>9 && command<13 && sub_1F183(actor)!=0)return 0;
    if(sub_4E893()%100<THRESHOLD)return sub_1C81F(actor,amount);
    return 0;
}
