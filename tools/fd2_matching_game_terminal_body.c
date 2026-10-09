/* sub_2BCE5完整C候選。固定IDA原名、偏移及既有終局證據另存主收據。
 * 局部聚合／volatile只作相容產碼表示，不推定原作者型別或硬體時序。
 */
typedef struct {unsigned char bytes[20];} RawTwenty;
extern const RawTwenty unk_525DC,unk_525F0,unk_52604;
#if defined(TERMINAL_BODY_6) || defined(TERMINAL_BODY_7)
#define TERMINAL_BODY_3
#endif
#ifdef TERMINAL_BODY_4
#define TERMINAL_BODY_2
#endif
typedef struct {
    RawTwenty third,second,first;
#ifdef TERMINAL_BODY_2
    volatile int remainder;
    void *volatile resource39,*volatile resourceOther,*volatile screen;
    volatile int left,right;
#else
    int remainder;
    void *resource39,*resourceOther,*screen;
    int left,right;
#endif
} TerminalFrame;
typedef char TerminalFrameMustBe84[sizeof(TerminalFrame)==84?1:-1];
extern int dword_53C03,dword_540FF;
extern unsigned char *dword_53A45;
extern void *dword_53A65;
extern const char aFdotherDat[];
extern void *malloc(unsigned),*memset(void *,int,unsigned),*memmove(void *,const void *,unsigned);
extern void free(void *);
extern void *sub_111BA(const char *,void *,int);
extern void sub_2935B(void *,int,void *,int,int),sub_2C39B(int,int),sub_2C405(void);
extern void sub_1F882(void),sub_1F525(void),j___delay(unsigned),sub_20421(int,int,int);
extern void sub_11DF2(int,int,int),sub_11D40(int,int,int),sub_17AA9(int),sub_25977(int,int);
extern void sub_11EB0(void *,int,const void *,int,int,int),sub_4E63D(void *,int,int,void *,int,int);
extern void sub_28A6C(int,int);
#if defined(TERMINAL_BODY_6) || defined(TERMINAL_BODY_7)
/* 保守允許修改，原版4E63D保存EBX的既有證據保持；不當作實際clobber。 */
#pragma aux sub_4E63D modify exact [eax ebx ecx edx];
#endif
#define VGA ((void *)0xa0000)
#define DRAW(IMAGE,INDEX,DEST,STRIDE) sub_2935B(IMAGE,INDEX,DEST,STRIDE,-1)
#ifdef TERMINAL_BODY_7
#define UNIT_FIELD(N) ((buffer=dword_53A45)[N])
#else
#define UNIT_FIELD(N) dword_53A45[N]
#endif

void sub_2BCE5(void)
{
#ifdef TERMINAL_BODY_1
    volatile TerminalFrame local;
#else
    TerminalFrame local;
#endif
#ifdef TERMINAL_BODY_5
    register unsigned char *buffer,*screen;
#else
    unsigned char *buffer,*screen;
#endif
    void *image,*savedImage;
    int counter,shade,outer,remainder;
    local.right=290;local.left=80;local.resource39=0;local.resourceOther=0;
    local.first=unk_525DC;local.second=unk_525F0;local.third=unk_52604;
    buffer=malloc(128000);screen=malloc(64000);local.screen=screen;memset(screen,0,64000);
    image=sub_111BA(aFdotherDat,0,54);savedImage=image;DRAW(image,0,screen,320);
    sub_1F882();memmove(VGA,screen,64000);sub_1F525();j___delay(1000);sub_20421(2,100,0);
    sub_11DF2(0,255,63);memmove(VGA,screen,64000);DRAW(image,9,VGA,320);
    shade=63;goto fade_check;
fade_next:
    sub_11DF2(0,255,shade);j___delay(4);--shade;
fade_check:
    if(shade>=0)goto fade_next;
    j___delay(2000);
    if(dword_53C03==26)sub_2C39B(4,17);
    else{sub_2C39B(37,2);sub_2C39B(21,3);sub_2C39B(26,4);sub_2C39B(105,5);sub_2C39B(32,6);}
    j___delay(500);outer=0;goto triple_check;
triple_next:
    sub_11DF2(0,255,shade);j___delay(4);--shade;
triple_inner_check:
    if(shade>=0)goto triple_next;
    j___delay(200);++outer;
triple_check:
    if(outer<3){shade=63;goto triple_inner_check;}
    counter=12;goto montage_check;
montage_next:
    DRAW(savedImage,counter,VGA,320);j___delay(20);++counter;
montage_check:
    if(counter<109)goto montage_next;
    memmove(VGA,local.screen,64000);
    if(dword_53C03==26){sub_2C39B(21,18);sub_2C39B(24,19);sub_2C39B(26,20);}
    else sub_2C39B(45,7);
    j___delay(2000);counter=0;goto drift_check;
drift_next:
    j___delay(20);sub_11EB0(VGA,320,buffer+160,640,320,200);++counter;
drift_check:
    if(counter>=40)goto drift_done;
    sub_11EB0(buffer+160,640,local.screen,320,320,200);
    DRAW(savedImage,(remainder=counter%4)+1,buffer+local.right,640);
    DRAW(savedImage,remainder+=5,buffer+local.left,640);
    local.left+=2;local.right-=2;
    if(counter>=25)goto drift_next;
    local.right-=2;goto drift_next;
drift_done:
    memmove(buffer,local.screen,64000);DRAW(savedImage,1,buffer,320);DRAW(savedImage,5,buffer,320);
    sub_11EB0(VGA,320,buffer,320,320,200);
    if(dword_53C03==26){sub_2C39B(32,21);sub_2C39B(36,22);sub_2C39B(32,23);}
    else{sub_2C39B(32,8);sub_2C39B(36,9);}
    shade=0;counter=0;goto expand_check;
expand_next:
    sub_11D40(0,255,shade);++counter;
expand_check:
    if(counter>=200)goto expand_done;
    memmove(buffer,local.screen,64000);
    DRAW(savedImage,(local.remainder=counter%4)+1,buffer,320);
    DRAW(savedImage,local.remainder+5,buffer,320);j___delay(20);
    sub_11EB0(VGA,320,buffer,320,320,200);
    if(counter<=135)goto expand_next;
    ++shade;goto expand_next;
expand_done:
    free(buffer);memset(VGA,0,64000);sub_2C405();memset(VGA,0,64000);sub_25977(-1,1);sub_17AA9(50);
#if defined(TERMINAL_BODY_3) || defined(TERMINAL_BODY_4) || defined(TERMINAL_BODY_5)
    buffer=sub_111BA(aFdotherDat,local.resourceOther,60);sub_4E63D(buffer,0,0,VGA,320,-1);sub_1F525();
#else
    image=sub_111BA(aFdotherDat,local.resourceOther,60);sub_4E63D(image,0,0,VGA,320,-1);sub_1F525();
#endif
    sub_25977(18,0);sub_17AA9(80);sub_1F882();memset(VGA,0,64000);
#if defined(TERMINAL_BODY_3) || defined(TERMINAL_BODY_4) || defined(TERMINAL_BODY_5)
    local.resourceOther=sub_111BA(aFdotherDat,buffer,58);
#else
    local.resourceOther=sub_111BA(aFdotherDat,image,58);
#endif
    local.resource39=sub_111BA(aFdotherDat,local.resource39,57);
    savedImage=dword_53A65;counter=0;goto records_check;
record_invalid:
    UNIT_FIELD(6)=0;
record_body:
    UNIT_FIELD(7)=local.first.bytes[counter];
    if(local.second.bytes[counter]<76)UNIT_FIELD(86)=2;else UNIT_FIELD(86)=0;
    UNIT_FIELD(87)=local.second.bytes[counter];dword_540FF=local.third.bytes[counter];sub_28A6C(0,1);
    dword_53A65=local.resource39;sub_11D40(0,255,0);sub_17AA9(20);
    DRAW(local.resourceOther,counter,VGA,320);sub_17AA9(78);sub_1F882();memset(VGA,0,64000);++counter;
records_check:
    if(counter>=20)goto records_done;
    dword_53A65=savedImage;
    if(local.first.bytes[counter]>=76)goto record_invalid;
    UNIT_FIELD(6)=2;goto record_body;
records_done:
    dword_53A65=savedImage;free(local.resource39);sub_17AA9(50);
#if defined(TERMINAL_BODY_3) || defined(TERMINAL_BODY_4) || defined(TERMINAL_BODY_5)
    buffer=sub_111BA(aFdotherDat,local.resourceOther,59);sub_4E63D(buffer,0,0,VGA,320,-1);sub_1F525();free(buffer);
#else
    image=sub_111BA(aFdotherDat,local.resourceOther,59);sub_4E63D(image,0,0,VGA,320,-1);sub_1F525();free(image);
#endif
}
