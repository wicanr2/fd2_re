/* sub_20C6F完整C候選。沿用既有物品owner證據，保持原始欄位與caller。
 * 宣告、局部聚合與pragma僅作相容產碼表示，不推定作者原始C。
 */
#ifdef ITEM_EFFECT_12
#define ITEM_EFFECT_3
#define ITEM_EFFECT_9
#endif
#ifdef ITEM_EFFECT_13
#define ITEM_EFFECT_12
#define ITEM_EFFECT_3
#define ITEM_EFFECT_9
#endif
#ifdef ITEM_EFFECT_14
#define ITEM_EFFECT_13
#define ITEM_EFFECT_12
#define ITEM_EFFECT_3
#define ITEM_EFFECT_9
#endif
#if defined(ITEM_EFFECT_15) || defined(ITEM_EFFECT_16)
#define ITEM_EFFECT_12
#define ITEM_EFFECT_3
#define ITEM_EFFECT_9
#endif
#ifdef ITEM_EFFECT_17
#define ITEM_EFFECT_12
#define ITEM_EFFECT_3
#define ITEM_EFFECT_9
#endif
typedef struct {unsigned char raw[100];int saved,power;} RawEffect108;
typedef char RawEffectMustBe108[sizeof(RawEffect108)==108?1:-1];
extern unsigned char *dword_53A45;
extern int dword_53EC4,dword_53EC8;
extern unsigned char *sub_4E56C(int);
extern int sub_1B722(int,int),sub_1C9DD(int,int),sub_1C75E(int,int),sub_1B6B7(void *);
extern void sub_1D4CB(void),sub_1D4F6(void),sub_1DF58(void),sub_1DB65(void);
extern void sub_211A4(int,int,unsigned char *,int),sub_22AF6(int,int,int,unsigned char *,int);
extern void sub_1B8E7(int,int),sub_21082(int,int,int,int,int,unsigned char *,int);
extern void sub_1C4CC(int,int,int,unsigned char *),sub_1C2DA(int,int,int,unsigned char *);
extern void sub_1E0DB(int,int,int),sub_1E1DC(int),sub_11CAC(int);
extern void sub_22997(int,int,unsigned char *),sub_22D1B(int,int,int,unsigned char *,int);
extern void sub_22866(int,int,unsigned char *),sub_22721(int,int,unsigned char *);
extern void sub_1CD17(int,int,int,unsigned char *),sub_2111A(int,int,unsigned char *,int);
extern void sub_2218A(int,int,unsigned char *),sub_1AA1D(int,int,void *);
#ifndef ITEM_EFFECT_2
/* 原版4E56C實際只改EAX／EDX；保守允許修改另列，不當作作者ABI。 */
#pragma aux sub_4E56C modify exact [eax ebx ecx edx];
#endif
#if defined(ITEM_EFFECT_9) || defined(ITEM_EFFECT_10)
/* 保守允許修改；原版函式保存契約另存證據，不推定作者宣告。 */
#pragma aux sub_1C9DD modify exact [eax ebx ecx edx];
#pragma aux sub_1C75E modify exact [eax ebx ecx edx];
#endif
#if defined(ITEM_EFFECT_9) || defined(ITEM_EFFECT_11)
#pragma aux sub_1E0DB modify exact [eax ebx ecx edx];
#pragma aux sub_1E1DC modify exact [eax ebx ecx edx];
#endif

#if defined(ITEM_EFFECT_4) || defined(ITEM_EFFECT_5) || defined(ITEM_EFFECT_6) || defined(ITEM_EFFECT_7) || defined(ITEM_EFFECT_8) || defined(ITEM_EFFECT_9) || defined(ITEM_EFFECT_10) || defined(ITEM_EFFECT_11)
void sub_20C6F(int actor,int item,int count,unsigned char *targets)
#else
void sub_20C6F(int actor,int item,volatile int count,unsigned char *targets)
#endif
{
#ifdef ITEM_EFFECT_3
    unsigned char scratch[100];int saved,power;
#define SCRATCH scratch
#define SAVED saved
#define POWER power
#else
    RawEffect108 local;
#define SCRATCH local.raw
#define SAVED local.saved
#define POWER local.power
#endif
#if defined(ITEM_EFFECT_14) || defined(ITEM_EFFECT_15) || defined(ITEM_EFFECT_16)
    int i,value,selected;
#endif
    unsigned char *row,*record,*target;
#ifdef ITEM_EFFECT_1
    int type;
#else
    unsigned char type;
#endif
#if !defined(ITEM_EFFECT_14) && !defined(ITEM_EFFECT_15) && !defined(ITEM_EFFECT_16)
    int i,value,selected;
#endif
    sub_1D4CB();dword_53EC4=0;
    row=sub_4E56C(sub_1B722(actor,item));
#ifdef ITEM_EFFECT_7
    POWER=*(volatile unsigned short *)(row+14);
#else
    POWER=*(unsigned short *)(row+14);
#endif
    type=row[13];
    if(type==5 || type==13){
        sub_211A4(actor,count,targets,POWER);
        if(type==5)sub_1B8E7(actor,item);
    }
    else if(type==6){
        sub_22AF6(actor,20,count,targets,37);
        if(dword_53EC4!=0)sub_1DF58();
        sub_1B8E7(actor,item);
    }
    else if(type==7){
        sub_22AF6(actor,21,count,targets,38);
        if(dword_53EC4!=0)sub_1DF58();
        sub_1B8E7(actor,item);
    }
    else if(type==8){sub_21082(actor,POWER,55,item,count,targets,17);}
    else if(type==9){sub_21082(actor,POWER,57,item,count,targets,18);}
    else if(type==10){sub_21082(actor,POWER,62,item,count,targets,19);}
    else if(type==11){
        sub_1C4CC(actor,13,count,targets);sub_1C2DA(actor,13,count,targets);
        i=0;goto revive_check;
revive_next:
        value=sub_1C9DD(value,POWER);sub_1E0DB(value,105,*target);
revive_advance:
        ++i;
revive_check:
#ifdef ITEM_EFFECT_8
        if(i>=*(volatile int *)&count)goto revive_done;
#else
        if(i>=count)goto revive_done;
#endif
        target=targets+i;value=*target;
#ifdef ITEM_EFFECT_17
        record=dword_53A45+value*80;
        if(*(unsigned short *)(record+70))goto revive_next;
#elif defined(ITEM_EFFECT_16)
        if((*(unsigned *)(dword_53A45+value*80+68)>>16)!=0)goto revive_next;
#elif defined(ITEM_EFFECT_15)
        if(*(unsigned short *)(dword_53A45+value*80+70)!=0)goto revive_next;
#elif defined(ITEM_EFFECT_13)
        record=dword_53A45+value*80;
        if((*(unsigned *)(record+68)>>16)!=0)goto revive_next;
#elif defined(ITEM_EFFECT_6)
        {
            int offset=value;offset=(offset<<2)+offset;offset<<=4;
            if(*(volatile unsigned short *)(dword_53A45+offset+70)!=0)goto revive_next;
        }
#elif defined(ITEM_EFFECT_5) || defined(ITEM_EFFECT_7) || defined(ITEM_EFFECT_8)
        record=dword_53A45+value*80;
        if(*(volatile unsigned short *)(record+70)!=0)goto revive_next;
#else
        record=dword_53A45+value*80;
        if(*(unsigned short *)(record+70)!=0)goto revive_next;
#endif
        sub_1E1DC(value);goto revive_advance;
revive_done:
        sub_11CAC(0);sub_1DF58();sub_1B8E7(actor,item);
    }
    else if(type==12){sub_22997(actor,count,targets);}
    else if(type==14){sub_22D1B(actor,27,count,targets,38);}
    else if(type==15){sub_22866(actor,count,targets);}
    else if(type==16){sub_22721(actor,count,targets);}
    else if(type==17){sub_21082(actor,POWER,66,item,count,targets,13);}
    else if(type==18){sub_21082(actor,POWER,70,item,count,targets,13);}
    else if(type==19){
        record=dword_53A45+targets[0]*80;SAVED=record[60];
        sub_21082(actor,POWER,59,item,count,targets,19);record[60]=(unsigned char)SAVED;
    }
    else if(type==20 || type==24){
        sub_1C4CC(actor,POWER,count,targets);sub_1CD17(actor,POWER,count,targets);
        i=0;goto damage_check;
damage_next:
        sub_1E0DB(value,94,*target);
damage_advance:
        ++i;
damage_check:
#ifdef ITEM_EFFECT_8
        if(i>=*(volatile int *)&count)goto damage_done;
#else
        if(i>=count)goto damage_done;
#endif
        target=targets+i;value=sub_1C75E(*target,POWER);
        if(value!=0)goto damage_next;
        sub_1E1DC(*target);goto damage_advance;
damage_done:
        sub_11CAC(0);sub_1DF58();
    }
    else if(type==21){sub_2111A(actor,count,targets,POWER);}
    else if(type==22){sub_22D1B(actor,22,count,targets,39);}
    else if(type==23){sub_2218A(actor,count,targets);}
    dword_53EC8=0;sub_1D4F6();selected=sub_1B6B7(SCRATCH);sub_1DB65();sub_1AA1D(actor,selected,SCRATCH);
}
