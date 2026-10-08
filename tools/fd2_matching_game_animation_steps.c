/* 完整演出步進及資源控制的純C候選。
 * 固定原檔、IDA Pro 9.4原名、caller、完整bytes見匹配主收據。
 * 型別與區域名稱只作產碼導覽，不推定作者宣告或新增硬體時序契約。
 */
extern int dword_53A30,dword_53A34,dword_53A38,dword_53A3C;
extern void *dword_53AD1,*dword_53ED0,*dword_53EE0,*dword_5411F;
extern int dword_53BFF,dword_53BF3,dword_540BA[16];
extern unsigned char byte_51A11,byte_53EF0,byte_51E61,byte_540FB,byte_540FA;
extern const char aFdmusDat[],a08d_1[],a08d_2[];
typedef struct {unsigned char bytes[16];} RawSixteen;
extern const RawSixteen unk_52539;
extern void sub_1EC2A(int *,int),sub_15E71(void *,void *,int),sub_375B2(int);
extern int sub_1F0DC(int,int);
extern void *sub_15F0E(void *,void *,int,int,int,int),*sub_111BA(const char *,void *,int);
extern void __cdecl free(void *);
extern int __cdecl sub_36316(void *,unsigned);
extern void sub_3ADD4(void *,int,int),sub_3AC0B(void *),sub_3AAA5(void *,void *,int);
extern void sub_3AB9E(void *),sub_3AE56(void *,int);
extern void sub_2935B(void *,int,void *,int,int),sub_25A96(void *,int,int),sub_25B45(void *,int,int);
extern void sub_2D620(void *,int,int);
extern int __cdecl sprintf(char *,const char *,...);

#ifdef F1EB05
int *sub_1EB05(int first,int second)
{
    int index,frame;
    void *one,*two;
    sub_1EC2A(&dword_53A30,second);
    if(sub_1F0DC(first,second)==1)sub_1EC2A(&dword_53A38,first);
    else dword_53A38=-1;
    index=0;goto condition;
next:
    ++index;
condition:
    if(index>=10)goto done;
    frame=index+39;
    one=sub_15F0E(dword_53AD1,(void *)0xa0000,320,dword_53A30,dword_53A34,frame);
    if(dword_53A38!=-1)two=sub_15F0E(dword_53AD1,(void *)0xa0000,320,dword_53A38,dword_53A3C,frame);
    sub_375B2(25);
    if(index>=9)goto next;
    sub_15E71(one,(void *)0xa0000,320);
    if(dword_53A38!=-1)sub_15E71(two,(void *)0xa0000,320);
    goto next;
done:
    free(one);if(dword_53A38!=-1)free(two);
    return &dword_53A30;
}
#endif

#ifdef F25977
void sub_25977(int cue,int loops)
{
    if((int)byte_51A11==cue)return;
    byte_51A11=cue;
    if(cue==-1){sub_3ADD4(dword_53ED0,0,4000);return;}
    if((int)byte_53EF0!=0){
    if(dword_53EE0)sub_3AC0B(dword_53ED0);
    if(cue==-1)return;
    dword_53EE0=sub_111BA(aFdmusDat,dword_53EE0,cue);
    sub_36316(dword_53EE0,dword_53BFF);
    sub_3AAA5(dword_53ED0,dword_53EE0,0);sub_3AB9E(dword_53ED0);
    if(byte_51E61){
        if(cue!=16&&cue!=17){sub_3ADD4(dword_53ED0,0,0);sub_3ADD4(dword_53ED0,127,2000);}
        else sub_3ADD4(dword_53ED0,127,0);
    }else sub_3ADD4(dword_53ED0,0,0);
    sub_3AE56(dword_53ED0,loops);
    }
}
#endif

#ifdef F274B0
int sub_274B0(int ignored,void *data,void *dest,int width,unsigned char event)
{
    RawSixteen local=unk_52539;
    int stride=width;
    int index,result=0,value=event;
    if(value==0){
        index=0;goto init_condition;
init_next:
        dword_540BA[index]=-index*2;++index;
init_condition:
        if(index<16)goto init_next;
        return 3;
    }
    if(value==3)return 34;
    if(value==6)return 2;
    if(value!=2&&value!=5)goto zero;
    index=0;goto condition;
next:
    ++index;
condition:
    if(index>=16)goto done;
    if(dword_540BA[index]>=0&&dword_540BA[index]<8)
        sub_2935B(data,dword_540BA[index]+(int)local.bytes[index],dest,stride,-1);
    if(dword_540BA[index]==0)sub_25A96(dword_5411F,1,1);
    if(dword_540BA[index]==4)sub_25B45(dword_5411F,2,1);
    ++dword_540BA[index];if(dword_540BA[index]==4)result=1;
    goto next;
done:
    return result;
zero:
    return 0;
}
#endif

#ifdef F275D6
int sub_275D6(int ignored,void *data,void *dest,int width,unsigned char event)
{
    int value=event;
    if(value==0){byte_540FB=0;byte_540FA=1;return 20;}
    if(value==3)return 60;
    if(value==6)return 20;
    if(value==1||value==7){
        if(byte_540FB==0)sub_2935B(data,byte_540FB,dest,width,-1);
        byte_540FB^=1;return 0;
    }
    if(value==4){sub_2935B(data,0,dest,width,-1);return 0;}
    if(value!=5)return 0;
    sub_2935B(data,(int)byte_540FA/2,dest,width,-1);
    value=byte_540FA;
    if(value==6)sub_25A96(dword_5411F,1,1);
    else if(value==36)sub_25B45(dword_5411F,2,1);
    ++byte_540FA;value=byte_540FA;
    if(value<44&&value>16)return 1;
    return 0;
}
#endif

#ifdef F2D3FF
void sub_2D3FF(int difference)
{
    struct {
        unsigned char current[20];
        unsigned char old[20];
        unsigned char step[20];
    } digits;
    int index,digit,equal,amount,base,pattern,before,after;
    sprintf((char *)digits.old,a08d_1,dword_53BF3);
    index=0;goto old_condition;
old_next:
    digits.old[index]-=48;++index;
old_condition:
    if(index<8)goto old_next;
    dword_53BF3+=difference;
    sprintf((char *)digits.current,a08d_2,dword_53BF3);
    index=0;goto current_condition;
current_next:
    digits.current[index]-=48;++index;
current_condition:
    if(index<8)goto current_next;
again:
    equal=1;
    index=0;goto compare_condition;
compare_zero:
    digits.step[index]=0;
compare_next:
    ++index;
compare_condition:
    if(index>=8)goto compare_done;
    before=digits.old[index];after=digits.current[index];
    if(before==after)goto compare_zero;
    digits.step[index]=1;equal=0;goto compare_next;
compare_done:
    if(equal)goto check;
    index=0;goto frame_condition;
frame_next:
    sub_375B2(10);++index;
frame_condition:
    if(index>=9)goto check;
    digit=0;goto digit_condition;
digit_next:
    ++digit;
digit_condition:
    if(digit>=8)goto frame_next;
    amount=digits.step[digit];
    if(!amount)goto digit_next;
    base=digits.old[digit];pattern=base<<3;pattern+=base;pattern+=amount;
    sub_2D620((void *)(0xa7a90+digit*6),320,pattern);
    ++digits.step[digit];
    if(digits.step[digit]!=10)goto digit_next;
    ++digits.old[digit];if(digits.old[digit]==10)digits.old[digit]=0;
    goto digit_next;
check:
    if(!equal)goto again;
}
#endif
