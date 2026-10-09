/* 完整顯示、事件派送及選單C候選。
 * 固定原檔與IDA Pro 9.4原名、caller、指令及原始具名參照見匹配主收據。
 * 所有型別及local資料布局只作產碼導航，不提升作者宣告或未知欄位語意。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A51,*dword_53A85,*dword_53C5B,*dword_53C5F,*dword_53C63;
extern int dword_53BEB,dword_51A83,dword_51A8F,dword_53C03,dword_53ECC,dword_53C23,dword_53C33;
extern int dword_53C27,dword_53C2B,dword_53C2F,dword_53C4B,dword_53EC8,dword_53C67;
extern int dword_53C57,dword_5412F;
extern void *dword_53EEC;
extern unsigned char byte_53AF9;
extern const char aDatoDat[];
extern int abs(int),sub_14B78(int,int,int,int),sub_14818(int,int,void *,int,int,int);
extern void sub_12D7B(int),sub_12CEA(int,int),sub_11CAC(int),sub_375B2(int);
extern unsigned char *__cdecl sub_4E516(int);
extern void __cdecl sub_4DBFC(void *);
extern void sub_2A6BD(int,int,int,void *),sub_1D4CB(void),sub_1D4F6(void),sub_1DB65(void);
extern int sub_1B653(void *);
extern void sub_1AA1D(int,int,void *),sub_1598A(int,int),sub_1567E(int,int),sub_13A9F(int,int);
extern void sub_4E031(void),sub_168B6(void *,int,int,int,int,int),sub_1974C(int,void *,void *);
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*sub_111BA(const char *,void *,int);
extern void sub_4E8E1(void *,void *,int);
extern void (*funcs_1541F[])(int,int,void *);
extern void (__cdecl *funcs_1199C[])(int),(__cdecl *funcs_1197B[])(int);
extern int sub_2D85F(int);
extern void sub_25A96(void *,int,int),sub_2E19B(void),sub_2E26C(void),sub_2DC55(int,void *,int,void *,int);

#ifdef F13E9C
int sub_13E9C(int unit,int team)
{
    struct {int result,besty,x,y,bestx,best;} local;
    int index,cx,cy,first,distance,offset;
    unsigned char *record;
    local.best=65535;local.bestx=-1;local.result=0;
    offset=unit<<2;offset+=unit;offset<<=4;
    record=(unsigned char *)(unsigned)offset;record+=(unsigned)dword_53A45;
    local.x=record[0];local.y=record[1];index=0;goto condition;
next:
    ++index;
condition:
    if(index>=dword_53BEB)goto done;
    record=dword_53A45+index*80;
    if(team==0&&(int)record[6]!=0)goto accept;
    if(team==0||(int)record[6]!=0)goto next;
accept:
    cx=record[0];cy=record[1];first=abs(local.x-cx);distance=first+abs(local.y-cy);
    if(distance>=local.best)goto next;
    local.bestx=cx;local.besty=cy;local.best=distance;goto next;
done:
    if(local.bestx==-1)return 0;
    if(local.bestx==local.x&&local.besty==local.y)return local.result;
    dword_51A83=0;sub_12D7B(unit);
    if(sub_14B78(local.bestx,local.besty,unit,team))local.result=1;
    dword_51A83=1;return local.result;
}
#endif

#ifdef F15311
int sub_15311(int unit,int mode)
{
    struct {unsigned char targets[32];unsigned char info[12];int count;} local;
    unsigned char *data=sub_4E516(dword_53C2F);
    int team,count,total;
    if(mode==0){if((int)data[6]==0)team=1;else team=0;}else team=data[6];
    if(dword_53C23<6)goto zero;
    sub_12D7B(unit);
    count=sub_14818(dword_53C27,dword_53C2B,local.targets,data[4],0,team);local.count=count;
    sub_4DBFC(dword_53A51);sub_375B2(200);
    dword_51A83=(int)data[4]+2;sub_12CEA(dword_53C27,dword_53C2B);
    dword_51A83=0;sub_11CAC(0);
    if(dword_53C2F<10&&(int)byte_53AF9==0)sub_2A6BD(unit,dword_53C2F,count,local.targets);
    else{sub_1D4CB();funcs_1541F[dword_53C2F](unit,local.count,local.targets);sub_1D4F6();}
    total=sub_1B653(local.info);sub_11CAC(0);sub_1DB65();
    sub_1AA1D(dword_53C4B,total,local.info);sub_11CAC(0);
    dword_53EC8=0;dword_51A83=0;return 1;
zero:
    return 0;
}
#endif

#ifdef F1956B
void sub_1956B(int image)
{
    unsigned char *data;
    int index;
    dword_53C5B=malloc(64000);dword_53C5F=malloc(64000);dword_53C63=malloc(64000);
    memmove(dword_53C5F,(void *)0xa0000,64000);memmove(dword_53C63,dword_53C5F,64000);
    sub_168B6(dword_53C63,320,5,112,19,5);
    if(image==128)dword_53C67=4283;
    else if(image==129)dword_53C67=1707;
    else if(image==130)dword_53C67=3939;
    else if(image==131)dword_53C67=1398;
    else if(image==132)dword_53C67=3644;
    else dword_53C67=36887;
    data=sub_111BA(aDatoDat,dword_53A85,image);dword_53A85=data;
    data+=(int)data[0];sub_4E8E1(dword_53C63+dword_53C67,data,320);
    index=5;goto condition;
next:
    sub_1974C(index*13+112,dword_53C5B,dword_53C63);--index;
condition:
    if(index>=0)goto next;
}
#endif

#ifdef F1D8BA
void sub_1D8BA(void)
{
    int index,team,value,offset;
    unsigned char *record;
    index=0;goto first_condition;
first_next:
    ++index;
first_condition:
    if(index>=dword_53BEB)goto second;
    dword_51A83=0;sub_4E031();
    value=index<<2;offset=index+value;offset<<=4;value=(unsigned)dword_53A45;
    record=(unsigned char *)(value+offset);dword_51A8F=255;
    team=record[6];
    if(team!=0)goto first_dispatch;
    if((unsigned char)(record[5]&0x81)!=0)goto first_dispatch;
    if((int)record[38]!=0)goto first_dispatch;
    sub_1598A(index,team);sub_1567E(index,0);
    if(dword_53C23>=6||dword_53C33>=6)sub_13A9F(index,0);
first_dispatch:
    if(dword_51A8F!=255)funcs_1199C[dword_51A8F](index);
    funcs_1197B[dword_53C03](index);if(dword_53ECC)goto done;goto first_next;
second:
    index=0;goto second_condition;
second_next:
    ++index;
second_condition:
    if(index>=dword_53BEB)goto done;
    sub_4E031();
    value=index<<2;offset=index+value;offset<<=4;value=(unsigned)dword_53A45;
    record=(unsigned char *)(value+offset);dword_51A8F=255;
    team=record[6];
    if(team!=0)goto second_dispatch;
    if((unsigned char)(record[5]&0x81)!=0)goto second_dispatch;
    if((int)record[38]!=0)goto second_dispatch;
    sub_13A9F(index,team);
second_dispatch:
    if(dword_51A8F!=255)funcs_1199C[dword_51A8F](index);
    funcs_1197B[dword_53C03](index);if(!dword_53ECC)goto second_next;
done:
    return;
}
#endif

#ifdef F2DF6B
int sub_2DF6B(int count,void *data,unsigned char event)
{
    register int limit=count;
    register void *items=data;
    int result=0,key;
again:
    key=sub_2D85F(1);
    if(key==77){
        if(limit-1==dword_53C57)goto check;
right:
        sub_25A96(dword_53EEC,0,1);++dword_53C57;
down_check:
        if(dword_53C57-dword_5412F>=6){dword_5412F+=2;sub_2E19B();}
draw:
        sub_2DC55(limit,items,dword_53C57,(void *)0xa0000,(int)event);goto check;
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
        if(limit-2<=dword_53C57)goto check;
        sub_25A96(dword_53EEC,0,1);dword_53C57+=2;goto down_check;
    }
    if(key==28||key==57)result=1;
    else if(key==1)result=-1;
check:
    if(!result)goto again;
    return result;
}
#endif
