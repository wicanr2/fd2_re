/* sub_1366A與sub_1F894完整C群組，共用13994收尾。
 * 原始名字、raw欄位與呼叫順序保留。局部型別、pragma及作者宣告仍未知。
 */
#ifdef TITLE_ACTING_5
#define TITLE_ACTING_4
#endif
#ifdef TITLE_ACTING_2
typedef volatile unsigned char RawByte;
#else
typedef unsigned char RawByte;
#endif
typedef struct {int values[15];} RawRows60;
typedef struct {
    unsigned char directions[32],targets[32];
#ifdef TITLE_ACTING_5
    unsigned cycle:8;unsigned :24;
    unsigned count:8;unsigned :24;
    unsigned total:8;unsigned :24;
    unsigned index:8;unsigned :24;
    unsigned pose:8;unsigned :24;
    unsigned mode:8;unsigned :24;
    unsigned step:8;unsigned :24;
#else
    RawByte cycle;unsigned char p64[3];
    RawByte count;unsigned char p68[3];
    RawByte total;unsigned char p72[3];
    RawByte index;unsigned char p76[3];
    RawByte pose;unsigned char p80[3];
    RawByte mode;unsigned char p84[3];
    RawByte step;unsigned char p88[3];
#endif
} RawAct92;
typedef struct {
    RawRows60 rows;
    unsigned char *save;
    int done;
    void *screen,*menu;
    int frames,menucount;
    void *resource;
    RawByte event;unsigned char pad[3];
} RawTitle92;
typedef char RawActMustBe92[sizeof(RawAct92)==92?1:-1];
typedef char RawTitleMustBe92[sizeof(RawTitle92)==92?1:-1];
extern const RawRows60 unk_5204E;
extern const char aFdotherDat[],aRb_2[],aFd2Sav_3[];
extern unsigned char *dword_53A45,*dword_53A49,word_53A8D[];
extern void *dword_53A65;
extern int dword_53AA9,dword_53AAD,dword_53BEB,dword_53AFB;
extern unsigned char *sub_4E7F8(int);
extern void *sub_111BA(const char *,void *,int),*malloc(unsigned),*memset(void *,int,unsigned);
extern void *fopen(const char *,const char *);
extern int fclose(void *),int386(int,void *,void *),sub_10620(void);
extern unsigned sub_37072(void *,unsigned,unsigned,void *),sub_4DBB9(void *,unsigned);
extern void free(void *),sub_4DBD8(void *,unsigned);
extern void sub_17AA9(int),sub_11D40(int,int,int),sub_11DF2(int,int,int),j___delay(unsigned);
extern int sub_11CAC(int);
extern void sub_11EEE(void *,int,int,int,int,int),sub_11EB0(void *,int,void *,int,int,int);
extern void sub_127E0(int),sub_129EC(void),sub_4E031(void),sub_2C9EC(int);
extern void sub_4E63D(void *,int,int,void *,int,int),sub_1F525(void),sub_1F882(void);
extern void sub_20421(int,int,int),sub_1F81E(int,int,int),sub_1F73F(int,int,void *,int);
extern void sub_25A96(void *,int,int),sub_25B45(void *,int,int),sub_286BD(int,int,int,int,int,int);
extern void sub_16886(void *,int,void *,int),sub_1FF79(void *,int,int);
#ifdef TITLE_ACTING_3
/* 保守允許修改只作候選，原版實際保存契約另列。 */
#pragma aux sub_4E031 modify exact [eax ebx ecx edx];
#pragma aux sub_4E63D modify exact [eax ebx ecx edx];
#endif

#ifdef TITLE_ACTING_1
int sub_1366A(int resource)
#else
void sub_1366A(int resource)
#endif
{
    RawAct92 local;
    unsigned char *cursor,*record,*saved;
    int unit;
#ifdef TITLE_ACTING_4
    record=sub_4E7F8(resource);local.total=*record;local.step=0;cursor=record+1;
#else
    cursor=sub_4E7F8(resource);local.total=*cursor;local.step=0;++cursor;
#endif
    goto event_check;
event_next:
    ++local.step;
event_check:
    if(local.step>=local.total)goto acting_done;
    local.mode=cursor[0];local.count=cursor[1];local.index=0;cursor+=2;
    goto pair_check;
pair_next:
    local.targets[local.index]=cursor[0];local.directions[local.index]=cursor[1];cursor+=2;++local.index;
pair_check:
    if(local.index<local.count)goto pair_next;
    if(!(local.mode&128))goto normal;
    local.mode&=127;
    if(local.mode!=0)goto fixed_pose;
    sub_17AA9(1);sub_11EEE(dword_53A49+0x8088,456,13,8,dword_53AA9,dword_53AAD);
    saved=dword_53A49;unit=0;goto unit_check;
unit_next:
    ++local.index;
unit_target_check:
    if(local.index>=local.count)goto unit_draw;
    if(unit!=local.targets[local.index])goto unit_next;
    dword_53A49=saved-0x1560;record[3]=local.directions[local.index];goto unit_next;
unit_draw:
    if(!(record[5]&1))sub_127E0(unit);
    dword_53A49=saved;++unit;
unit_check:
    if(unit>=dword_53BEB)goto units_done;
    record=dword_53A45+unit*80;local.index=0;goto unit_target_check;
units_done:
    sub_129EC();sub_11EB0((void *)0xA0504,320,dword_53A49+0x8088,456,312,192);
    sub_17AA9(2);sub_11CAC(0);sub_4E031();goto event_next;
fixed_pose:
    local.index=0;goto fixed_check;
fixed_next:
    dword_53A45[local.targets[local.index]*80+3]=local.directions[local.index];++local.index;
fixed_check:
    if(local.index<local.count)goto fixed_next;
    local.index=0;goto fixed_wait_check;
fixed_wait_next:
    sub_11CAC(0);sub_17AA9(1);sub_4E031();++local.index;
fixed_wait_check:
    if(local.index<local.mode)goto fixed_wait_next;
    goto event_next;
normal:
    local.cycle=0;goto cycle_check;
cycle_next:
    ++local.cycle;
cycle_check:
    if(local.cycle>=local.mode)goto event_next;
    local.pose=1;goto pose_check;
pose_next:
    dword_53A45[local.targets[local.index]*80+3]=local.directions[local.index];
    dword_53A45[local.targets[local.index]*80+4]=local.pose;++local.index;
pose_target_check:
    if(local.index<local.count)goto pose_next;
    if(dword_53AFB!=0 && dword_53AFB!=64){++dword_53AFB;sub_11CAC(1);sub_11D40(0,255,dword_53AFB);}
    else sub_11CAC(0);
    sub_17AA9(1);sub_4E031();++local.pose;
pose_check:
    if(local.pose>=7)goto move_start;
    sub_2C9EC(local.targets[0]);local.index=0;goto pose_target_check;
move_start:
    local.index=0;goto move_check;
move_next:
    if(local.directions[local.index]==0)++record[1];
    else if(local.directions[local.index]==1)--record[0];
    else if(local.directions[local.index]==3)++record[0];
    else --record[1];
    record[4]=0;++local.index;
move_check:
    if(local.index>=local.count)goto cycle_next;
    record=dword_53A45+local.targets[local.index]*80;goto move_next;
acting_done:
#ifdef TITLE_ACTING_1
    return sub_11CAC(1);
#else
    sub_11CAC(1);
#endif
}

int sub_1F894(void)
{
    RawTitle92 local;
    unsigned char *scroll,*row,*meta;
    void *file;
    int selected,index,scan,last;
    local.menucount=1;local.done=0;selected=0;local.menu=0;local.frames=12;local.event=0;
    local.rows=unk_5204E;
    local.resource=sub_111BA(aFdotherDat,0,77);memset((void *)0xA0000,0,64000);
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,76);sub_11D40(0,255,64);
    local.screen=sub_111BA(aFdotherDat,0,74);sub_4E63D(local.screen,0,0,(void *)0xA0000,320,-1);
    sub_1F525();sub_17AA9(1);sub_17AA9(30);sub_1F882();
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,99);memset((void *)0xA0000,0,64000);sub_11D40(0,255,0);
    sub_20421(3,90,1);sub_1F882();memset((void *)0xA0000,0,64000);
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,101);sub_11D40(0,255,64);
    scroll=(unsigned char *)malloc(0x396c0);memset(scroll,0,0x396c0);
    index=0;goto images_check;
images_next:
    local.screen=sub_111BA(aFdotherDat,local.screen,index+69);
    sub_4E63D(local.screen,0,index*147,scroll,320,-1);++index;
images_check:
    if(index<5)goto images_next;
    sub_4E031();if(dword_53A45)free(dword_53A45);dword_53A45=(unsigned char *)malloc(160);
    index=535;goto row_check;
row_next:
    --index;
row_check:
    if(index<0)goto scroll_done;
    sub_11EB0((void *)0xA0000,320,scroll+index*320,320,320,200);
    if(index==535)sub_1F525();row=scroll+index*320;
    if(index==25){
        sub_1F81E(0,15,0);sub_11EB0((void *)0xA0000,320,row,320,320,200);
        dword_53A65=sub_111BA(aFdotherDat,dword_53A65,101);sub_1F525();
    }else if(index==330){sub_1F882();sub_1F81E(4,90,99);sub_1F81E(5,50,0);}
    else if(index==210){sub_1F882();sub_1F81E(6,90,99);sub_1F81E(7,50,0);}
    else if(index==110){sub_1F882();sub_1F81E(8,90,99);}
    else if(index==450)sub_1F73F(100,99,scroll,index);
    else if(index==10)sub_1F73F(75,76,scroll,index);
    if(index==local.rows.values[local.event]){
        local.frames=0;sub_25A96(local.resource,0,1);
        dword_53A65=sub_111BA(aFdotherDat,dword_53A65,102);sub_11D40(0,255,0);++local.event;
    }
    if(local.frames==11){dword_53A65=sub_111BA(aFdotherDat,dword_53A65,101);sub_11D40(0,255,0);}
    ++local.frames;j___delay(30);if(index==0)j___delay(1000);
    if(sub_10620()==0)goto row_next;
scroll_done:
    index=40;goto fade_out_check;
fade_out_next:
    sub_286BD(0,255,index,63,0,0);j___delay(8);--index;
fade_out_check:
    if(index>=0)goto fade_out_next;
    j___delay(100);sub_4E031();free(scroll);free(local.screen);
    local.menu=sub_111BA(aFdotherDat,local.menu,7);
    dword_53A65=sub_111BA(aFdotherDat,dword_53A65,8);memset((void *)0xA0000,0,64000);sub_11D40(0,255,0);
    sub_20421(1,15,1);sub_25B45(local.resource,3,1);sub_11DF2(0,255,64);
    sub_16886((void *)0xA0000,320,local.menu,0);
    index=0;goto fade_in_check;
fade_in_next:
    sub_286BD(0,255,index,56,60,63);j___delay(8);++index;
fade_in_check:
    if(index<=40)goto fade_in_next;
    sub_4E031();file=fopen(aFd2Sav_3,aRb_2);
    if(file){
        local.save=(unsigned char *)malloc(0x59cb);sub_37072(local.save,1,0x59cb,file);fclose(file);
        sub_4DBD8(local.save,0x59cb);
#ifdef TITLE_ACTING_4
        scroll=local.save+0x30c3;
#else
        meta=local.save+0x30c3;
#endif
        if(sub_4DBB9(local.save,0x59cb)==*(unsigned *)(local.save+0x59c7)){
            local.menucount=2;
#ifdef TITLE_ACTING_4
            if(scroll[2]!=255)local.menucount=3;
#else
            if(meta[2]!=255)local.menucount=3;
#endif
        }
        free(local.save);
    }
    sub_1FF79(local.menu,selected,local.menucount);
#ifdef TITLE_ACTING_4
key_check:
    if(local.done!=0)goto key_done;
#else
    goto key_check;
#endif
key_next:
    sub_1FF79(local.menu,selected,local.menucount);word_53A8D[1]=16;int386(0x16,word_53A8D,word_53A8D);
    scan=word_53A8D[1];last=local.menucount-1;
    if(scan==72){sub_25A96(local.resource,2,1);if(selected==0)selected=last;else --selected;}
    else if(scan==80){sub_25A96(local.resource,2,1);if(selected==last)selected=0;else ++selected;}
#ifdef TITLE_ACTING_4
    else {last=word_53A8D[0];if(last==13 || last==32 || scan==224 || scan==82){
        sub_25A96(local.resource,1,1);local.done=1;
    }}
    goto key_check;
key_done:
#else
    else if(word_53A8D[0]==13 || word_53A8D[0]==32 || scan==224 || scan==82){
        sub_25A96(local.resource,1,1);local.done=1;
    }
key_check:
    if(local.done==0)goto key_next;
#endif
    index=0;goto flash_check;
flash_next:
    sub_1FF79(local.menu,-1,local.menucount);j___delay(80);
    sub_1FF79(local.menu,selected,local.menucount);j___delay(80);++index;
flash_check:
    if(index<4)goto flash_next;
    sub_1F882();memset((void *)0xA0000,0,64000);free(local.menu);
    sub_25A96(local.resource,-1,1);free(local.resource);return selected;
}
