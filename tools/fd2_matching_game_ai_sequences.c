/* 人工智慧及場景完整C候選。固定原檔與IDA Pro 9.4原名、caller及bytes見主收據。
 * 名稱、型別與local資料布局只作產碼導航，不推定原作者宣告或未證實玩法。
 */
typedef struct {unsigned char bytes[16];} RawSixteen;
typedef struct {unsigned char bytes[9];} RawNine;
typedef struct {unsigned char bytes[25];} RawTwentyFive;
extern const RawSixteen unk_521F6,unk_52206;
extern const RawNine unk_52216,unk_5221F;
extern const RawTwentyFive unk_52228,unk_52241,unk_5225A;
extern const double dbl_50144;
extern unsigned char * volatile dword_53A45;
extern void *dword_53A51,*dword_53A69;
extern int dword_53C23,dword_53C27,dword_53C2B,dword_53C2F,dword_53C33,dword_53C37,dword_53C3B,dword_53C3F;
extern int dword_53C43,dword_53C47,dword_53C4B,dword_51A83,dword_53A79,dword_53BEF,dword_53C03;
extern int dword_53AA9,dword_53AAD,dword_53AB1,dword_53AB5,dword_53AB9,dword_53ABD;
extern unsigned char byte_53AF9;
extern unsigned char *sub_1EB05(int,int);
extern unsigned char *sub_4E516(int);
extern unsigned char *__cdecl sub_4E56C(int);
extern void *__cdecl sub_4E555(int);
extern void __cdecl sub_4DBFC(void *),__cdecl sub_4E040(void *,int,int,int,void *,void *);
extern void *malloc(unsigned);
extern void free(void *);
extern int sub_14B78(int,int,int,int),sub_1F0DC(int,int),sub_1E856(int,int,void *),sub_1B6B7(void *);
extern int sub_1B8A6(int),sub_14B16(void *),sub_14818(int,int,void *,int,int,int);
extern int sub_149F8(int,int,void *,int,int,int,int),sub_15880(int,int,void *),sub_1C269(int,void *);
extern int sub_1F183(int),sub_15DA2(int,void *,int,int),sub_31860(int,int);
extern int sub_15B77(int,int,const unsigned char *);
extern void sub_12D7B(int),sub_1F04A(int,int),sub_11CAC(int),sub_134E4(void),sub_1DB65(void);
extern void sub_1E7F6(void *,int,int,void *),sub_1E611(void *,int,int),sub_28A6C(int,int);
extern void sub_1AA1D(int,int,void *),sub_1E292(int),sub_1F882(void),sub_13536(void),sub_1F525(void);
extern void sub_375B2(int),sub_1366A(int),sub_112A5(int),sub_11506(void),sub_10B4E(int),sub_24336(void);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int),sub_1B8E7(int,int),sub_1C220(int);
extern void sub_233C6(void *,void *,void *,int,int,int,int,int,int,int,int);

#ifdef F1548E
int sub_1548E(int unit,int mode)
{
    unsigned char info[12];
    int *one,*two;
    int count,performed,adjacent;
    dword_51A83=0;sub_12D7B(unit);sub_14B78(dword_53C43,dword_53C47,unit,mode);
    dword_51A83=1;sub_12D7B(dword_53C4B);sub_1F04A(unit,dword_53C4B);
    if((int)byte_53AF9!=0){
        dword_51A83=0;sub_11CAC(0);dword_51A83=1;
        if(sub_1F0DC(unit,dword_53C4B)==1)sub_1F04A(dword_53C4B,unit);
        one=(int *)sub_1EB05(unit,dword_53C4B);two=one;
        sub_1E7F6((void *)0xa0000,320,dword_53C4B,one);
        if(sub_1F0DC(unit,dword_53C4B)==1){one+=2;sub_1E7F6((void *)0xa0000,320,unit,one);}
        sub_134E4();
        performed=sub_1E856(unit,dword_53C4B,two)!=0;
        adjacent=sub_1F0DC(unit,dword_53C4B)==1;
        if(performed&adjacent){
            sub_1F04A(unit,dword_53C4B);sub_1F04A(dword_53C4B,unit);
            sub_1E611(two,unit,dword_53C4B);two+=2;sub_1E856(dword_53C4B,unit,two);
        }
    }else sub_28A6C(unit,dword_53C4B);
    sub_134E4();count=sub_1B6B7(info);sub_11CAC(0);sub_1DB65();
    sub_1AA1D(dword_53C4B,count,info);sub_11CAC(0);sub_1E292(dword_53C4B);return 1;
}
#endif

#ifdef F1567E
int sub_1567E(int unit,int mode)
{
    struct {
        unsigned char targets[32];
        int count;
        unsigned char *record;
        int positions,x,y;
        unsigned char *work;
        int item,index,dx,dy;
    } local;
    unsigned char *data;
    unsigned char radius,kind;
    int point,targetcount,team,score,offset;
    dword_53C33=0;sub_4E555(0);offset=unit<<2;offset+=unit;offset<<=4;
    local.record=(unsigned char *)(unsigned)offset;local.record+=(unsigned)dword_53A45;
    local.x=local.record[0];local.y=local.record[1];local.work=malloc(400);
    local.count=sub_1B8A6(unit);if(local.count==0)goto zero;
    local.index=0;goto item_condition;
item_next:
    ++local.index;
item_condition:
    if(local.index>=local.count)goto free_work;
    local.item=local.record[local.index*2+11];data=sub_4E56C(local.item);
    kind=0;radius=data[16];if((int)radius>15){radius=1;kind=radius;}
    if((int)data[13]==0)goto item_next;
    sub_14818(local.x,local.y,0,radius,kind,0);
    local.positions=sub_14B16(local.work);sub_4DBFC(dword_53A51);point=0;goto point_condition;
point_next:
    ++point;
point_condition:
    if(point>=local.positions)goto item_next;
    local.dx=local.work[point*2];local.dy=local.work[point*2+1];
    if(mode==0){if((int)data[17]==0)team=1;else team=0;}else team=data[17];
    if((int)data[16]>15)targetcount=sub_149F8(local.dx,local.dy,local.targets,local.x,local.y,(int)data[16]-16,0);
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
#endif

#ifdef F1598A
int sub_1598A(int unit,int mode)
{
    struct {
        unsigned char targets[32],list[12];
        int mp,count,y,x;
        void *grid;
        int positions;
        unsigned char *work;
        int best,dy,targetcount,dx;
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
    address=point*2;address+=(unsigned)local.work;position=(unsigned char *)address;
    local.dx=position[0];local.dy=position[1];
    if(mode==0){if((int)data[6]==0)team=1;else team=0;}else team=data[6];
    local.targetcount=sub_14818(local.dx,local.dy,local.targets,data[4],0,team);
    sub_4DBFC(dword_53A51);if(!local.targetcount)goto point_next;
    score=sub_15B77(local.list[index],local.targetcount,local.targets);
    if(score>dword_53C23)goto accept;
    if(score!=dword_53C23||*(unsigned short *)data<=local.best)goto point_next;
accept:
    dword_53C23=score;dword_53C27=local.dx;dword_53C2B=local.dy;dword_53C2F=local.list[index];
    local.best=*(unsigned short *)data;goto point_next;
free_work:
    free(local.work);
zero:
    return 0;
}
#endif

#ifdef F15B77
int sub_15B77(int item,int count,const unsigned char *targets)
{
    volatile int divisor;
    struct {int power,hp,value;} local;
    unsigned char *record;
    int result=0,index,unit,maximum;
    local.power=*(unsigned short *)sub_4E516(item);
    if(item<13){
        index=0;goto heal_condition;
heal_next:
        ++index;
heal_condition:
        if(index>=count)goto done;
        if(item>=10&&sub_1F183(targets[index]))goto heal_next;
        record=dword_53A45+(int)targets[index]*80;
        if(*(unsigned short *)(record+64)<local.power)local.value=24;else local.value=8;
        if((int)record[8]==0)local.value=(int)(local.value*dbl_50144);
        result+=local.value;goto heal_next;
    }
    if(item<17){
        index=0;goto hp_condition;
hp_next:
        ++index;
hp_condition:
        if(index>=count)goto done;
        record=dword_53A45+(int)targets[index]*80;local.hp=*(unsigned short *)(record+64);
        maximum=*(unsigned short *)(record+66);divisor=3;
        if(maximum/divisor>local.hp)local.value=8;
        else if(maximum/2>local.hp)local.value=3;else local.value=0;
        if((unsigned char)(record[52]&1)!=0)local.value<<=1;
        result+=local.value;goto hp_next;
    }
    if(item<20){result=sub_15DA2(count,(void *)targets,item+17,3);goto done;}
    if(item==20){
        index=0;goto flag25_condition;
flag25_next:
        ++index;
flag25_condition:
        if(index>=count)goto done;
        if((int)dword_53A45[(int)targets[index]*80+37]!=0)result+=6;goto flag25_next;
    }
    if(item==21){
        index=0;goto flag26_condition;
flag26_next:
        ++index;
flag26_condition:
        if(index>=count)goto done;
        if((int)dword_53A45[(int)targets[index]*80+38]!=0)result+=6;goto flag26_next;
    }
    if(item==22){
        index=0;goto flag27_condition;
flag27_next:
        ++index;
flag27_condition:
        if(index>=count)goto done;
        unit=targets[index];if((int)dword_53A45[unit*80+39]!=0)goto flag27_next;
        if(sub_1C269(unit,0))result+=6;goto flag27_next;
    }
    if(item==26)result=sub_15DA2(count,(void *)targets,37,4);
    else if(item==27)result=sub_15DA2(count,(void *)targets,38,4);
done:
    return result;
}
#endif

#ifdef F23E74
void sub_23E74(void)
{
    RawSixteen first;
    RawSixteen second;
    RawNine third;
    RawNine fourth;
    unsigned char *record;
    int index,scaled,offset,base;
    first=unk_521F6;second=unk_52206;third=unk_52216;fourth=unk_5221F;
    sub_1F882();sub_13536();index=0;goto first_condition;
first_next:
    scaled=index<<2;offset=index+scaled;offset<<=4;base=(unsigned)dword_53A45;
    record=(unsigned char *)(base+offset);record[0]=first.bytes[index];record[1]=second.bytes[index];record[3]=1;++index;
first_condition:
    if(index<16)goto first_next;
    index=0;goto second_condition;
second_next:
    offset=index+52;scaled=offset<<2;offset+=scaled;offset<<=4;
    base=(unsigned)dword_53A45;record=(unsigned char *)(base+offset);
    record[0]=third.bytes[index];record[1]=fourth.bytes[index];record[3]=3;++index;
second_condition:
    if(index<9)goto second_next;
    dword_51A83=0;dword_53AA9=26;dword_53AAD=31;dword_53AB1=26;dword_53AB5=31;dword_53AB9=0;dword_53ABD=0;
    sub_11CAC(1);sub_1F525();sub_375B2(200);
    sub_15F84(dword_53A79,11,(void *)0xa0000,320,205,76,74,19,1);
    dword_51A83=0;sub_1366A(59);sub_15F84(dword_53A79,12,(void *)0xa0000,320,205,76,74,19,1);
    dword_51A83=0;sub_112A5(25);sub_11506();
    if(dword_53BEF<=15){
        sub_10B4E(1);sub_1366A(60);sub_15F84(dword_53A79,14,(void *)0xa0000,320,205,76,74,19,1);
        dword_51A83=0;sub_1366A(61);sub_15F84(dword_53A79,15,(void *)0xa0000,320,205,76,74,19,1);
        dword_51A83=0;sub_1366A(62);sub_15F84(dword_53A79,16,(void *)0xa0000,320,205,76,74,19,1);sub_112A5(28);
    }
    sub_15F84(dword_53A79,13,(void *)0xa0000,320,205,76,74,19,1);++dword_53C03;
}
#endif

#ifdef F240FA
void sub_240FA(void)
{
    struct {unsigned item,index;} slots;
#define item (*(unsigned char *)&slots.item)
#define index (*(unsigned char *)&slots.index)
    RawTwentyFive first,second,third;
    int count,position;
    first=unk_52228;second=unk_52241;third=unk_5225A;
    count=0;
    sub_233C6(&first,&second,&third,0,24,25,23,14,1,14,10);
    sub_15F84(dword_53A79,5,(void *)0xa0000,320,205,76,74,19,1);
    item=209;goto item_condition;
item_next:
    ++item;
item_condition:
    if((int)item>=215)goto decision;
    index=0;goto unit_condition;
unit_next:
    ++index;
unit_condition:
    if((int)index>=16)goto item_next;
    if(sub_31860(index,item)!=-1)++count;goto unit_next;
decision:
    if(count!=6)goto other;
    item=209;goto remove_item_condition;
remove_item_next:
    ++item;
remove_item_condition:
    if((int)item>=215)goto reward;
    index=0;goto remove_unit_condition;
remove_unit_next:
    ++index;
remove_unit_condition:
    if((int)index>=16)goto remove_item_next;
    position=sub_31860(index,item);if(position!=-1)sub_1B8E7(index,position);goto remove_unit_next;
reward:
    sub_1C220(100);sub_15F84(dword_53A79,7,(void *)0xa0000,320,205,76,74,19,1);
    dword_51A83=0;sub_1366A(63);sub_15F84(dword_53A79,8,(void *)0xa0000,320,205,76,74,19,1);
    dword_51A83=0;sub_1366A(64);sub_15F84(dword_53A79,9,(void *)0xa0000,320,205,76,74,19,1);
    sub_24336();sub_15F84(dword_53A79,10,(void *)0xa0000,320,205,76,74,19,1);goto done;
other:
    sub_15F84(dword_53A79,6,(void *)0xa0000,320,205,76,74,19,1);
done:
    sub_112A5(24);sub_112A5(23);sub_11506();++dword_53C03;
}
#undef item
#undef index
#endif
