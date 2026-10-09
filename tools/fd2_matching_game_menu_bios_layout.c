/* 2D85F完整C候選。固定IDA 9.4原名、位址、bytes與caller見主收據。
 * 保留BIOS字讀取、有號比較、原始等待與輸入流程；不提升硬體時序語意。
 */
typedef struct { int values[4]; } RawSixteen;
extern const RawSixteen unk_526EA;
extern unsigned char *dword_54147,word_53A8D[];
extern int dword_53C57,dword_54133;
extern int sub_4E893(void),sub_10620(void),int386(int,void *,void *);
extern void sub_2D9FE(void *,int),sub_4E9E4(void *,void *,int),sub_16559(int);

#if defined(MENU_BIOS_0) || defined(MENU_BIOS_1) || defined(MENU_BIOS_2) || defined(MENU_BIOS_3)
int sub_2D85F(volatile int mode)
{
#if defined(MENU_BIOS_0) || defined(MENU_BIOS_1)
#ifdef MENU_BIOS_0
    struct {RawSixteen positions;unsigned tick,flash;} local;
#else
    struct {RawSixteen positions;short tick,pad;unsigned flash;} local;
#endif
#define positions local.positions
#define tick local.tick
#define flash (*(unsigned char *)&local.flash)
#else
    volatile unsigned char flash;
#ifdef MENU_BIOS_2
    short tick;
#else
    unsigned tick;
#endif
    RawSixteen positions;
#endif
    short now;
    int random,wait,index,frame,before,difference;
    unsigned char *base,*image;
    flash=0;positions=unk_526EA;
    tick=*(volatile unsigned short *)0x46c;
    random=sub_4E893();wait=random%50+8;base=(unsigned char *)0xad430;dword_54133=2;
    sub_2D9FE(&positions,mode);
    if(mode==0){
        index=0;goto draw_condition;
draw_next:
        image=dword_54147;image+=*(unsigned *)(image+frame*4+6);
        sub_4E9E4(base+positions.values[index],image,320);++index;
draw_condition:
        if(index>=4)goto again;
        frame=index*2+3;if(index==dword_53C57)++frame;
        goto draw_next;
    }
again:
    now=*(volatile short *)0x46c;before=(short)tick;difference=now-before;
    if(difference<2&&now>=before)goto input;
    ++dword_54133;if(dword_54133==4)dword_54133=0;sub_2D9FE(&positions,mode);
    if((int)flash!=0){
        sub_16559(0);random=sub_4E893();wait=random%30+2;flash=0;
    }else{
        random=wait;--wait;if(!random){sub_16559(3);flash=1;}
    }
    tick=*(volatile unsigned short *)0x46c;
input:
    if(!sub_10620())goto again;
    word_53A8D[1]=16;int386(22,word_53A8D,word_53A8D);
    if((int)word_53A8D[1]==224||(int)word_53A8D[1]==82)word_53A8D[1]=28;
    if((int)word_53A8D[1]==83)word_53A8D[1]=1;
    return word_53A8D[1];
}
#undef positions
#undef tick
#undef flash
#endif
