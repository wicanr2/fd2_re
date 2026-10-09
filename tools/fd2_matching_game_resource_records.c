/* 完整資源、隊伍記錄與介面C候選。固定原檔及IDA Pro 9.4完整區間見主收據。
 * 保留原名、呼叫、原始讀寫寬度；型別與local布局只作產碼導航。
 */
typedef struct {unsigned char bytes[16];} RawSixteen;
typedef struct {unsigned char bytes[30];} RawThirty;
extern const RawSixteen unk_52183,unk_52193;
extern const RawThirty unk_51F15;
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7,*dword_53A61,*dword_53A49;
extern int dword_53BFB,dword_53BDF,dword_539EC,dword_53B17[],dword_53C57;
extern int dword_53C0B,dword_53AA9,dword_53AAD,dword_51A87,dword_51A8B;
extern int dword_53A79,dword_53BEF,dword_53C03,dword_51A83,dword_5412B;
extern void *dword_53EEC,*dword_53B13,*dword_54147;
extern const char aRb_10[],aFdiconB24_4[];
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*fopen(const char *,const char *);
extern void free(void *);
extern int fclose(void *),fseek(void *,long,int);
extern unsigned sub_37072(void *,unsigned,unsigned,void *);
extern int sub_16C57(int),sub_3453E(int);
extern int sub_11019(int,void *);
extern unsigned char *sub_4E56C(int);
extern void sub_184C0(int,int,void *),sub_25A96(void *,int,int),sub_17AA9(int),sub_375B2(int);
extern void __cdecl sub_4DDD7(void *,void *,int,int);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_11506(void),sub_1366A(int),sub_112A5(int);
extern void sub_233C6(void *,void *,int,int,int,int,int,int,int,int,int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
extern void sub_16886(void *,int,void *,int),sub_16559(int),sub_11DF2(int,int,int),sub_4E031(void);

#ifdef F11019
int sub_11019(int key,void *file)
{
    struct {int offsets[13];int length;} local;
    unsigned char *table;
    unsigned offset;
    int index,position,header=1920;
    fseek(file,6,0);table=malloc(6720);sub_37072(table,1,6720,file);
    index=0;goto copy_condition;
copy_next:
    local.offsets[index]=*(unsigned *)(table+(key*12+index)*4);++index;
copy_condition:
    if(index<13)goto copy_next;
    local.length=local.offsets[12]-local.offsets[0];free(table);
    if(dword_53BDF==0){
        dword_53B17[0]=key;dword_53A61=malloc(207360);
        fseek(file,local.offsets[0],0);sub_37072(dword_53A61+header,1,local.length,file);
        index=0;goto initial_condition;
initial_next:
        offset=local.offsets[index]-local.offsets[0]+header;
        ((unsigned *)dword_53A61)[index]=offset;++index;
initial_condition:
        if(index<12)goto initial_next;
        ++dword_53BDF;dword_539EC=local.length+header;position=0;goto done;
    }
    index=0;goto search_condition;
search_next:
    ++index;
search_condition:
    if(index>=dword_53BDF)goto append;
    if(key!=dword_53B17[index])goto search_next;
    position=index;goto done;
append:
    dword_53B17[index]=key;fseek(file,local.offsets[0],0);
    sub_37072(dword_53A61+dword_539EC,1,local.length,file);
    index=0;goto append_condition;
append_next:
    offset=local.offsets[index]-local.offsets[0];offset+=dword_539EC;
    ((unsigned *)dword_53A61)[dword_53BDF*12+index]=offset;++index;
append_condition:
    if(index<12)goto append_next;
    dword_539EC+=local.length;position=dword_53BDF;dword_53BDF=position+1;
done:
    return position;
}
#endif

#ifdef F1B9DE
int sub_1B9DE(int unit,int mode)
{
    int count=0,index,key,last;
    unsigned char *record,*data;
    sub_184C0(unit,dword_53C57,(void *)0xa0000);record=dword_53A45+unit*80;
    index=0;goto condition;
next:
    ++index;
condition:
    if(index>=8)goto input;
    if((unsigned char)(record[index*2+10]&0x80)==0)++count;
    goto next;
input:
    key=sub_16C57(0);last=count-1;
    if(key==72){
        if(dword_53C57==0){sub_25A96(dword_53EEC,0,1);dword_53C57=last;goto zero;}
        sub_25A96(dword_53EEC,0,1);--dword_53C57;return 0;
    }
    if(key==80){
        if(last==dword_53C57){sub_25A96(dword_53EEC,0,1);dword_53C57=0;goto zero;}
        sub_25A96(dword_53EEC,0,1);++dword_53C57;return 0;
    }
    if(key==75){
        if(dword_53C57<4)goto zero;
        sub_25A96(dword_53EEC,0,1);dword_53C57-=4;return 0;
    }
    if(key==77){
        if(dword_53C57>3)goto zero;
        count-=4;if(count<=dword_53C57)goto zero;
        sub_25A96(dword_53EEC,0,1);dword_53C57+=4;return 0;
    }
    if(key==28||key==57){
        if(mode==0)return 1;
        data=sub_4E56C(record[dword_53C57*2+11]);if((int)data[13]!=0)return 1;return 0;
    }
    if(key==1)return -1;
zero:
    return 0;
}
#endif

#ifdef F1C2DA
#pragma pack (4)
void sub_1C2DA(int ignored,int color,int count,const unsigned char *units)
{
    struct {RawThirty colors;unsigned char *row,*saved;} local;
    int index,x,y,frame;
    unsigned char *record,*dest,*sprite;
    local.colors=unk_51F15;sub_25A96(dword_53B13,1,1);
    local.saved=malloc(153216);memmove(local.saved,dword_53A49,153216);
    index=0;goto condition;
next:
    ++index;
condition:
    if(index>=count)goto restore;
    record=dword_53A45+(int)units[index]*80;x=record[0];y=record[1];frame=record[2];
    if(x<dword_53AA9-1||x>dword_53AA9+dword_51A87)goto next;
    if(y<dword_53AAD-1||y>dword_53AAD+dword_51A8B+1)goto next;
    local.row=dword_53A49+32904+(x-dword_53AA9)*24;
    dest=local.row+(y-dword_53AAD)*10944-2736;
    frame*=12;if(dword_53C0B==3)frame+=2;else frame+=dword_53C0B;
    sprite=dword_53A61+((unsigned *)dword_53A61)[frame];
    sub_4DDD7(sprite,dest,456,local.colors.bytes[color]);goto next;
restore:
    index=0;goto restore_condition;
restore_next:
    sub_11EB0((void *)0xa0504,320,local.saved+32904,456,312,192);sub_17AA9(1);
    sub_11EB0((void *)0xa0504,320,dword_53A49+32904,456,312,192);sub_17AA9(1);++index;
restore_condition:
    if(index<5)goto restore_next;
    sub_11EB0((void *)0xa0504,320,local.saved+32904,456,312,192);free(local.saved);
}
#pragma pack ()
#endif

#ifdef F23A0A
void sub_23A0A(void)
{
    struct {RawSixteen second,first;unsigned count,flag;} local;
    int index;
    *(unsigned char *)&local.flag=0;local.first=unk_52183;local.second=unk_52193;
    *(unsigned char *)&local.count=0;
    sub_233C6(&local.first,&local.second,0,0,15,65,28,30,2,22,25);
    index=0;goto condition;
next:
    ++index;
condition:
    if(index>=8)goto decision;
    if(sub_3453E(index+66))++*(unsigned char *)&local.count;
    goto next;
decision:
    if((int)*(unsigned char *)&local.count>4)*(unsigned char *)&local.flag=1;
    sub_11506();
    if(dword_53BEF>18||(int)*(unsigned char *)&local.flag==1||*(unsigned short *)(dword_53A45+66)<320){
        sub_15F84(dword_53A79,2,(void *)0xa0000,320,205,76,74,19,1);
        dword_51A83=0;sub_1366A(49);
        sub_15F84(dword_53A79,3,(void *)0xa0000,320,205,76,74,19,1);
    }else{sub_15F84(dword_53A79,4,(void *)0xa0000,320,205,76,74,19,1);sub_112A5(18);}
    ++dword_53C03;
}
#endif

#ifdef F2F4C6
void sub_2F4C6(void)
{
    int index;
    if(dword_5412B==1){
        index=0;goto first_condition;
first_next:
        sub_16886((void *)0xa38e9,320,dword_54147,index+23);sub_17AA9(2);++index;
first_condition:
        if(index<5)goto first_next;
        goto clear;
    }
    if(dword_5412B==3){
        sub_17AA9(1);sub_16886((void *)0xa3154,320,dword_54147,23);sub_17AA9(8);
clear:
        sub_16559(0);goto done;
    }
    if(dword_5412B==4){
        sub_16559(3);sub_17AA9(2);index=0;goto second_condition;
second_next:
        sub_16886((void *)0xa2893,320,dword_54147,index+23);sub_17AA9(2);++index;
second_condition:
        if(index<9)goto second_next;
        index=0;goto fade_condition;
fade_next:
        sub_11DF2(0,255,index);sub_375B2(4);index+=2;
fade_condition:
        if(index<64)goto fade_next;
        sub_17AA9(10);index=62;goto return_condition;
return_next:
        sub_11DF2(0,255,index);sub_375B2(4);index-=2;
return_condition:
        if(index>=0)goto return_next;
        sub_17AA9(5);goto clear;
    }
    if(dword_5412B==5){
        index=0;goto third_condition;
third_next:
        sub_16886((void *)0xa2383,320,dword_54147,index+23);sub_17AA9(2);++index;
third_condition:
        if(index<7)goto third_next;
    }
done:
    sub_4E031();
}
#endif

#ifdef F321C8
void sub_321C8(unsigned char id)
{
    unsigned char selected=0,next=2;
    int index;
    unsigned char *copy,*chosen,*record;
    int offset;
    void *file;
    index=1;goto find_condition;
find_next:
    ++index;
find_condition:
    if(index>=dword_53BFB)goto found;
    if((int)dword_53A45[index*80+8]==(int)id)selected=index;
    goto find_next;
found:
    copy=malloc(2560);chosen=copy;memmove(copy,dword_53BF7,2560);
    memmove(dword_53BF7+80,chosen+=(int)selected*80,80);
    index=1;goto copy_condition;
copy_next:
    ++index;
copy_condition:
    if(index>=dword_53BFB)goto reload;
    if(index==(int)selected)goto copy_next;
    memmove(dword_53BF7+(int)next*80,copy+index*80,80);++next;goto copy_next;
reload:
    free(copy);free(dword_53A61);file=fopen(aFdiconB24_4,aRb_10);dword_53BDF=0;
    index=0;goto icons_condition;
icons_next:
    record=dword_53BF7;offset=index<<2;offset+=index;offset<<=4;
    sub_11019(record[offset+7],file);++index;
icons_condition:
    if(index<dword_53BFB)goto icons_next;
    fclose(file);
}
#endif
