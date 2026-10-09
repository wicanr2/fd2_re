/* 完整人工智慧點位與執行期欄位清除C重建來源及負例。
 * 固定IDA 9.4原名、線性位址、原始bytes及caller見匹配主收據。
 * 宣告與局部中間值只作產碼表示；既有語意分級保持。
 */
extern unsigned char * volatile dword_53A45;
extern void *dword_53A51,*dword_53A69;
extern int dword_53C23,dword_53C27,dword_53C2B,dword_53C2F,dword_53BEB;
extern void *__cdecl sub_4E555(int);
extern unsigned char *sub_4E516(int);
extern void __cdecl sub_4DBFC(void *),__cdecl sub_4E040(void *,int,int,int,void *,void *);
extern void *malloc(unsigned);
extern void free(void *),sub_1DB65(void);
extern int sub_1C269(int,void *),sub_14B16(void *),sub_14818(int,int,void *,int,int,int);
extern int sub_15B77(int,int,const unsigned char *);
typedef char AddressWidthMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];

#if defined(AI_POINT_0) || defined(AI_POINT_1) || defined(AI_POINT_2) || defined(AI_POINT_3)
int sub_1598A(int unit,int mode)
{
#if defined(AI_POINT_2) || defined(AI_POINT_3)
#ifdef AI_POINT_2
    volatile int scalar_dx,scalar_targetcount;
#else
    int scalar_dx,scalar_targetcount;
#endif
#define POINT_DX (*(int *)&scalar_dx)
#define POINT_COUNT (*(int *)&scalar_targetcount)
#else
#define POINT_DX local.dx
#define POINT_COUNT local.targetcount
#endif
    struct {
        unsigned char targets[32],list[12];
        int mp,count,y,x;
        void *grid;
        int positions;
        unsigned char *work;
        int best,dy;
#if defined(AI_POINT_2) || defined(AI_POINT_3)
#elif defined(AI_POINT_1)
        volatile int targetcount;
#else
        int targetcount;
#endif
#if !defined(AI_POINT_2) && !defined(AI_POINT_3)
        int dx;
#endif
    } local;
    unsigned char *record,*data,*position;
    int index,point,team,score;
    unsigned address;
    dword_53C23=0;local.grid=sub_4E555(0);record=dword_53A45+unit*80;
    local.x=record[0];local.y=record[1];local.work=malloc(400);local.mp=*(unsigned short *)(record+68);
    local.count=sub_1C269(unit,local.list);if(!local.count||(int)record[39]!=0)goto zero;
    index=0;goto item_condition;
item_next:
    ++index;
item_condition:
    if(index>=local.count)goto free_work;
    data=sub_4E516(local.list[index]);if((int)data[5]>local.mp)goto item_next;
    sub_4E040(local.grid,local.x,local.y,data[3],dword_53A51,dword_53A69);
    local.positions=sub_14B16(local.work);sub_4DBFC(dword_53A51);point=0;goto point_condition;
point_next:
    ++point;
point_condition:
    if(point>=local.positions)goto item_next;
#if defined(AI_POINT_0) || defined(AI_POINT_3)
    address=point*2;address+=(unsigned)local.work;position=(unsigned char *)address;
#ifdef AI_POINT_0
    POINT_DX=position[0];local.dy=position[1];
#else
    local.dy=(POINT_DX=position[0],position[1]);
#endif
#elif defined(AI_POINT_1)
    POINT_DX=((unsigned char *)((unsigned)local.work+point*2))[0];
    local.dy=((unsigned char *)((unsigned)local.work+point*2))[1];
#else
    local.dy=(POINT_DX=((unsigned char *)((unsigned)local.work+point*2))[0],
        ((unsigned char *)((unsigned)local.work+point*2))[1]);
#endif
    if(mode==0){if((int)data[6]==0)team=1;else team=0;}else team=data[6];
#if defined(AI_POINT_2) || defined(AI_POINT_3)
    *(volatile int *)&POINT_COUNT=sub_14818(POINT_DX,local.dy,local.targets,data[4],0,team);
#else
    POINT_COUNT=sub_14818(POINT_DX,local.dy,local.targets,data[4],0,team);
#endif
    sub_4DBFC(dword_53A51);if(!POINT_COUNT)goto point_next;
    score=sub_15B77(local.list[index],POINT_COUNT,local.targets);
    if(score>dword_53C23)goto accept;
    if(score!=dword_53C23||*(unsigned short *)data<=local.best)goto point_next;
accept:
    dword_53C23=score;dword_53C27=POINT_DX;dword_53C2B=local.dy;dword_53C2F=local.list[index];
    local.best=*(unsigned short *)data;goto point_next;
free_work:
    free(local.work);
zero:
    return 0;
}
#undef POINT_DX
#undef POINT_COUNT
#endif

#if defined(RESET_WORD_0) || defined(RESET_WORD_1) || defined(RESET_WORD_2) || defined(RESET_WORD_3)
void sub_35BBA(int unit)
{
    int offset;
    unsigned char *record;
    for(;unit<dword_53BEB;++unit){
#ifdef RESET_WORD_0
        *(unsigned short *)(dword_53A45+unit*80+64)=0;
#else
#ifdef RESET_WORD_2
        offset=unit;offset*=5;offset*=16;
#else
        offset=unit*80;
#endif
        record=dword_53A45;
#ifdef RESET_WORD_3
        record+=offset;*(unsigned short *)(record+64)=0;
#else
        *(unsigned short *)(record+offset+64)=0;
#endif
#endif
    }
    sub_1DB65();
}
#endif
