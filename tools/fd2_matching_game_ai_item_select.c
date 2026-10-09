/* sub_1567E完整C候選。沿用既有winner／target-list證據與原始欄位。
 * 保存契約及volatile只作相容產碼表示，不推定作者宣告。
 */
#ifdef AI_ITEM_SELECT_16
#define AI_ITEM_SELECT_10
#endif
#ifdef AI_ITEM_SELECT_17
#define AI_ITEM_SELECT_11
#endif
#ifdef AI_ITEM_SELECT_14
#define AI_ITEM_SELECT_10
#endif
#ifdef AI_ITEM_SELECT_15
#define AI_ITEM_SELECT_11
#endif
#ifdef AI_ITEM_SELECT_12
#define AI_ITEM_SELECT_10
#endif
#ifdef AI_ITEM_SELECT_13
#define AI_ITEM_SELECT_11
#endif
#ifdef AI_ITEM_SELECT_10
#define AI_ITEM_SELECT_8
#endif
#ifdef AI_ITEM_SELECT_11
#define AI_ITEM_SELECT_9
#endif
#if defined(AI_ITEM_SELECT_6) || defined(AI_ITEM_SELECT_7)
#define AI_ITEM_SELECT_3
#endif
#if defined(AI_ITEM_SELECT_8) || defined(AI_ITEM_SELECT_9)
#define AI_ITEM_SELECT_3
#endif
typedef struct {
    unsigned char targets[32];
    int count;
#if defined(AI_ITEM_SELECT_6) || defined(AI_ITEM_SELECT_7)
    unsigned char *volatile record;
#else
    unsigned char *record;
#endif
    int positions,x,y;
    unsigned char *work;
    int item,index,dx,dy;
} RawSelect72;
typedef char RawSelectMustBe72[sizeof(RawSelect72)==72?1:-1];
extern unsigned char *volatile dword_53A45;
extern void *dword_53A51;
extern int dword_53C33,dword_53C37,dword_53C3B,dword_53C3F;
extern void *sub_4E555(int),*malloc(unsigned);
extern void free(void *),sub_4DBFC(void *);
extern unsigned char *sub_4E56C(int);
extern int sub_1B8A6(int),sub_14B16(void *),sub_15880(int,int,void *);
extern int sub_14818(int,int,void *,int,int,int),sub_149F8(int,int,void *,int,int,int,int);
/* 4DBFC實際保存EBX／ESI／EDI，4E56C只改EAX／EDX。 */
#pragma aux sub_4DBFC modify exact [eax ebx ecx edx];
#ifndef AI_ITEM_SELECT_2
#pragma aux sub_4E56C modify exact [eax ebx ecx edx];
#endif
#ifdef AI_ITEM_SELECT_1
#define READ13 data[13]
#define READ17 data[17]
#else
#define READ13 ((int)*(volatile unsigned char *)(data+13))
#define READ17 ((int)*(volatile unsigned char *)(data+17))
#endif

int sub_1567E(int actor,int mode)
{
    RawSelect72 local;
    unsigned char *data;
    unsigned char radius,kind;
    int point,targetcount,team,score,offset;
    dword_53C33=0;sub_4E555(0);
#if defined(AI_ITEM_SELECT_8) || defined(AI_ITEM_SELECT_9)
#ifdef AI_ITEM_SELECT_16
    data=(unsigned char *)(actor*80);
#elif defined(AI_ITEM_SELECT_17)
    data=(unsigned char *)(unsigned)actor;data=(unsigned char *)((int)data*80);
#elif defined(AI_ITEM_SELECT_14)
    data=(unsigned char *)((unsigned)*(volatile int *)&actor*80);
#elif defined(AI_ITEM_SELECT_15)
    data=(unsigned char *)((unsigned)actor*80);
#else
#if defined(AI_ITEM_SELECT_10) || defined(AI_ITEM_SELECT_11)
    data=(unsigned char *)(unsigned)*(volatile int *)&actor;
#else
    data=(unsigned char *)(unsigned)actor;
#endif
#if defined(AI_ITEM_SELECT_12) || defined(AI_ITEM_SELECT_13)
    data=(unsigned char *)(((unsigned)data<<2)+(unsigned)data);
#else
    data=(unsigned char *)((unsigned)data+((unsigned)data<<2));
#endif
    data=(unsigned char *)((unsigned)data<<4);
#endif
    local.x=(local.record=dword_53A45+(unsigned)data)[0];
    local.y=(*(unsigned char *volatile *)&local.record)[1];
#elif defined(AI_ITEM_SELECT_6) || defined(AI_ITEM_SELECT_7)
    data=(unsigned char *)(unsigned)actor;offset=(unsigned)data<<2;data+=offset;
    data=(unsigned char *)((unsigned)data<<4);local.record=dword_53A45+(unsigned)data;
#elif defined(AI_ITEM_SELECT_3) || defined(AI_ITEM_SELECT_4) || defined(AI_ITEM_SELECT_5)
    offset=actor;{int scaled=offset<<2;offset+=scaled;}offset<<=4;
    local.record=(unsigned char *)((unsigned)dword_53A45+offset);
#else
    offset=actor<<2;offset+=actor;offset<<=4;
    local.record=(unsigned char *)(unsigned)offset;local.record+=(unsigned)dword_53A45;
#endif

#if !defined(AI_ITEM_SELECT_8) && !defined(AI_ITEM_SELECT_9)
    local.x=local.record[0];local.y=local.record[1];
#endif
    local.work=(unsigned char *)malloc(400);
#if defined(AI_ITEM_SELECT_8) || defined(AI_ITEM_SELECT_9)
    score=sub_1B8A6(actor);local.count=score;
#ifdef AI_ITEM_SELECT_8
    if(score==0)return score;
#else
    if(score==0)return 0;
#endif
#else
    local.count=sub_1B8A6(actor);
#if defined(AI_ITEM_SELECT_3) || defined(AI_ITEM_SELECT_5)
    if(local.count==0)return local.count;
#elif defined(AI_ITEM_SELECT_4)
    if(local.count==0)return 0;
#else
    if(local.count==0)goto zero;
#endif
#endif
    local.index=0;goto item_check;
item_next:
    ++local.index;
item_check:
    if(local.index>=local.count)goto free_work;
    local.item=local.record[local.index*2+11];data=sub_4E56C(local.item);
    kind=0;radius=data[16];if((int)radius>15){radius=1;kind=radius;}
    if(READ13==0)goto item_next;
    sub_14818(local.x,local.y,0,radius,kind,0);
    local.positions=sub_14B16(local.work);sub_4DBFC(dword_53A51);point=0;goto point_check;
point_next:
    ++point;
point_check:
    if(point>=local.positions)goto item_next;
#if defined(AI_ITEM_SELECT_3) || defined(AI_ITEM_SELECT_4)
    {
        unsigned char *position=local.work+point*2;
#if defined(AI_ITEM_SELECT_7) || defined(AI_ITEM_SELECT_8) || defined(AI_ITEM_SELECT_9)
        offset=position[0];local.dx=offset;local.dy=position[1];
#else
        targetcount=position[0];local.dx=targetcount;local.dy=position[1];
#endif
    }
#else
    local.dx=local.work[point*2];local.dy=local.work[point*2+1];
#endif
    if(mode==0){if(READ17==0)team=1;else team=0;}else team=READ17;
    radius=data[16];
#if defined(AI_ITEM_SELECT_10) || defined(AI_ITEM_SELECT_11)
    if((int)radius>15)targetcount=sub_149F8(local.dx,local.dy,local.targets,local.x,local.y,(int)radius-16,0);
#elif defined(AI_ITEM_SELECT_8) || defined(AI_ITEM_SELECT_9)
    offset=radius;
    if(offset>15){offset-=16;targetcount=sub_149F8(local.dx,local.dy,local.targets,local.x,local.y,offset,0);}
#elif defined(AI_ITEM_SELECT_6) || defined(AI_ITEM_SELECT_7)
    targetcount=radius;
    if(targetcount>15){targetcount-=16;targetcount=sub_149F8(local.dx,local.dy,local.targets,local.x,local.y,targetcount,0);}
#else
    if((int)radius>15)targetcount=sub_149F8(local.dx,local.dy,local.targets,local.x,local.y,(int)radius-16,0);
#endif
    else targetcount=sub_14818(local.dx,local.dy,local.targets,data[18],0,team);
    sub_4DBFC(dword_53A51);if(!targetcount)goto point_next;
    score=sub_15880(local.item,targetcount,local.targets);if(score<=dword_53C33)goto point_next;
    dword_53C33=score;dword_53C37=local.dx;dword_53C3B=local.dy;dword_53C3F=local.index;
    goto point_next;
free_work:
    free(local.work);
zero:
    return 0;
}
