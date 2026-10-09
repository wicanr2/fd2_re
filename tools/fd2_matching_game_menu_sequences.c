/* 介面步進、輸入與交易完整C候選。固定原檔及IDA Pro 9.4完整區間見主收據。
 * 保留原始名稱、欄位寬度、呼叫與控制順序；名稱及型別只作產碼導航。
 */
typedef struct {int values[10];} RawForty;
typedef struct {int values[4];} RawSixteen;
typedef struct {short values[6];} RawTwelve;
extern const RawForty unk_52511;
extern const RawSixteen unk_526EA;
extern const RawTwelve unk_5272A,unk_52736;
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53C5B,*dword_53C5F,*dword_53C63,*dword_54147,*dword_53A61;
extern int dword_53C57,dword_5412F,dword_53BFB,dword_54133,dword_5413F,dword_5412B;
extern int dword_54097[4],dword_540A7[4],dword_53AD9,dword_53AE1,dword_53A7D;
extern unsigned char byte_540B7,byte_540B8,byte_540B9,byte_52659[],word_53A8D[];
extern void *dword_5411F,*dword_53EEC;
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned);
extern int sub_4E893(void),sub_10620(void),int386(int,void *,void *);
extern int sub_2E6B8(void),sub_2DF6B(int,void *,int),sub_16C57(int),sub_19953(void);
extern int sub_1B722(int,int);
extern unsigned char *sub_4E56C(int);
extern void sub_2935B(void *,int,void *,int,int),sub_25A96(void *,int,int),sub_25B45(void *,int,int);
extern void sub_2D9FE(void *,int),sub_4E9E4(void *,void *,int),sub_4E8AF(void *,void *,int);
extern void sub_2EA90(int,void *),sub_1974C(int,void *,void *),sub_16559(int);
extern int sub_2D85F(int);
extern void sub_2E19B(void),sub_2E26C(void),sub_2D31B(void),sub_2E0BD(int,void *,int);
extern void sub_1956B(int),sub_197E5(void),sub_2F4C6(void),sub_2D3FF(int),sub_1B8E7(int,int),sub_1B750(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);

#ifdef F272B8
int sub_272B8(int unit,void *image,void *dest,int stride,unsigned char event)
{
    int result=0,index,value,divisor,limit;
    RawForty local=unk_52511;
    if((int)dword_53A45[unit*80+6]==0){
        index=0;goto shift_condition;
shift_next:
        local.values[index]+=130;++index;
shift_condition:
        if(index<10)goto shift_next;
    }
    value=event;
    if(value==0){
        index=0;goto init_condition;
init_next:
        dword_54097[index]=-index*3;dword_540A7[index]=index;++index;
init_condition:
        if(index<4)goto init_next;
        byte_540B7=index;byte_540B8=0;byte_540B9=0;return 2;
    }
    if(value==3)return 32;
    if(value==6){byte_540B8=1;return 16;}
    if(value!=2&&value!=5&&value!=8)goto zero;
    ++byte_540B9;divisor=2;byte_540B9=(int)byte_540B9%divisor;
    index=0;limit=10;goto condition;
next:
    ++index;
condition:
    if(index>=3)goto done;
    if(dword_54097[index]>=0&&dword_54097[index]<5)
        sub_2935B(image,dword_54097[index],(unsigned char *)dest+local.values[dword_540A7[index]],stride,-1);
    if((int)byte_540B9!=0)goto next;
    if(dword_54097[index]!=1)goto advance;
    if(index==0)sub_25A96(dword_5411F,1,1);
    else if(index==1)sub_25B45(dword_5411F,index,index);
advance:
    ++dword_54097[index];if(dword_54097[index]==2)result=1;
    if(dword_54097[index]!=7||(int)byte_540B8!=0)goto next;
    ++byte_540B7;byte_540B7=(int)byte_540B7%limit;
    dword_540A7[index]=byte_540B7;dword_54097[index]=0;goto next;
done:
    return result;
zero:
    return 0;
}
#endif

#ifdef F2D85F
int sub_2D85F(int mode)
{
    struct {RawSixteen positions;unsigned tick,flash;} local;
    int random,wait,index,frame,now,before,difference;
    unsigned char *base;
    *(unsigned char *)&local.flash=0;local.positions=unk_526EA;
    local.tick=*(volatile unsigned short *)0x46c;
    random=sub_4E893();wait=random%50+8;base=(unsigned char *)0xad430;dword_54133=2;
    sub_2D9FE(&local.positions,mode);
    if(mode==0){
        index=0;goto draw_condition;
draw_next:
        frame=index*2+3;if(index==dword_53C57)++frame;
        sub_4E9E4(base+local.positions.values[index],
            dword_54147+((unsigned *)(dword_54147+6))[frame],320);++index;
draw_condition:
        if(index<4)goto draw_next;
    }
again:
    now=*(volatile short *)0x46c;before=(short)local.tick;difference=now-before;
    if(difference<2&&now>=before)goto input;
    ++dword_54133;if(dword_54133==4)dword_54133=0;sub_2D9FE(&local.positions,mode);
    if((int)*(unsigned char *)&local.flash!=0){
        sub_16559(0);random=sub_4E893();wait=random%30+2;*(unsigned char *)&local.flash=0;
    }else{
        random=wait;--wait;if(!random){sub_16559(3);*(unsigned char *)&local.flash=1;}
    }
    local.tick=*(volatile unsigned short *)0x46c;
input:
    if(!sub_10620())goto again;
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);
    if((int)word_53A8D[1]==224||(int)word_53A8D[1]==82)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#endif

#ifdef F2E6B8
int sub_2E6B8(void)
{
    int result=0,index,key;
    unsigned char *data;
    dword_53C5B=malloc(64000);dword_53C5F=malloc(64000);dword_53C63=malloc(64000);
    memmove(dword_53C5F,(void *)0xa0000,64000);memmove(dword_53C63,dword_53C5F,64000);
    dword_5412F=result;dword_53C57=result;
    data=dword_54147;data+=*(unsigned *)(data+70);sub_4E8AF(dword_53C63+35845,data,320);
    sub_2EA90(dword_53C57,dword_53C63);
    index=5;goto show_condition;
show_next:
    sub_1974C(index*13+112,dword_53C5B,dword_53C63);--index;
show_condition:
    if(index>=0)goto show_next;
again:
    key=sub_2D85F(3);
    if(key==77){
        if(dword_53BFB-1==dword_53C57)goto check;
        sub_25A96(dword_53EEC,0,1);++dword_53C57;
down_check:
        if(dword_53C57-dword_5412F>=6){dword_5412F+=2;sub_2E19B();}
draw:
        sub_2EA90(dword_53C57,(void *)0xa0000);goto check;
    }
    if(key==75){
        if(dword_53C57==0)goto check;
        sub_25A96(dword_53EEC,0,1);--dword_53C57;
up_check:
        if(dword_53C57<dword_5412F){dword_5412F-=2;sub_2E26C();}goto draw;
    }
    if(key==72){
        if(dword_53C57<2)goto check;
        sub_25A96(dword_53EEC,0,1);dword_53C57-=2;goto up_check;
    }
    if(key==80){
        if(dword_53BFB-2<=dword_53C57)goto check;
        sub_25A96(dword_53EEC,0,1);dword_53C57+=2;goto down_check;
    }
    if(key==28||key==57)result=1;
    else if(key==1)result=-1;
check:
    if(!result)goto again;
    return result;
}
#endif

#ifdef F2F642
void sub_2F642(void)
{
    RawTwelve first,second;
    unsigned char items[8],*record,*data,*slot;
    int unit,index,count,selection,item,response;
    first=unk_5272A;second=unk_52736;goto choose_unit;
again:
    sub_2D31B();
choose_unit:
    dword_5413F=dword_53BFB;response=sub_2E6B8();sub_2D31B();if(response==-1)goto done;
    unit=dword_53C57;record=dword_53A45+unit*80;count=0;index=0;goto item_condition;
item_next:
    ++index;
item_condition:
    if(index>=8)goto item_list;
    slot=record+index*2;slot+=10;
    if((unsigned char)(slot[0]&128)!=0)goto item_next;
    items[count]=slot[1];++count;goto item_next;
item_list:
    if(count==0){
        dword_53AD9=(int)dword_53A45[unit*80+7]+1;sub_1956B(byte_52659[dword_5412B]);
        sub_15F84(dword_53A7D,second.values[dword_5412B],(void *)0xa94cc,320,205,76,74,19,1);
        sub_16559(count);sub_16C57(1);goto again;
    }
    dword_53C57=0;dword_5412F=0;sub_2E0BD(count,items,1);dword_5413F=count;
    if(sub_2DF6B(count,items,1)==-1)goto again;
    selection=dword_53C57;sub_2D31B();item=sub_1B722(unit,dword_53C57);dword_53AD9=item+181;
    data=sub_4E56C(item);dword_53AE1=(int)*(unsigned short *)(data+19)*3/4;
    sub_1956B(byte_52659[dword_5412B]);
    sub_15F84(dword_53A7D,first.values[dword_5412B],(void *)0xa94cc,320,205,76,74,19,1);
    sub_16559(0);response=sub_19953();sub_197E5();
    if(response==-1||dword_53C57==1)goto again;
    sub_2D31B();sub_2F4C6();sub_2D3FF(dword_53AE1);sub_1B8E7(unit,selection);sub_1B750(unit);goto choose_unit;
done:
    return;
}
#endif
