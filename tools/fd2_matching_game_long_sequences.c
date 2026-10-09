/* 已閉合開場／戰後演出的完整C產碼候選。
 * 固定IDA 9.4原名、位址、bytes、caller及既有語意入口見主收據；不重開玩法或硬體時序。
 */
extern int dword_53C03,dword_53A79,dword_51A83,dword_53AFB,dword_53BF3,dword_53BEB;
extern unsigned char *dword_53A45;
extern void sub_205DA(void),sub_1F525(void),sub_134E4(void),sub_11506(void),sub_35E5A(void);
extern void sub_135DD(int,int),sub_1366A(int),sub_13185(int),sub_25977(int,int);
extern void sub_10B4E(int),sub_32975(int),sub_32999(int),sub_112A5(int),sub_11CAC(int),sub_12D7B(int);
extern void sub_35BBA(int),sub_12CEA(int,int),sub_22253(int,int,int,int,int),sub_24B4D(int);
extern void j___delay(unsigned),sub_11DF2(int,int,int);
extern void *memset(void *,int,unsigned);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A79,INDEX,(void *)0xa0000,320,205,76,74,19,1)

#if defined(INTRO_SEQ_0) || defined(INTRO_SEQ_1) || defined(INTRO_SEQ_2)
void sub_3231B(void)
{
#ifdef INTRO_SEQ_2
    int counter;
#else
    unsigned char counter;
#endif
#ifdef INTRO_SEQ_1
#define COUNTER_VALUE ((int)*(volatile unsigned char *)&counter)
#else
#define COUNTER_VALUE ((int)counter)
#endif
    dword_53C03=32;sub_205DA();sub_135DD(3,34);sub_1366A(99);
    counter=0;goto first_check;
first_next:
    sub_13185(2);++counter;
first_check:
    if(COUNTER_VALUE<15)goto first_next;
    MESSAGE(0);dword_51A83=0;
    counter=0;goto second_check;
second_next:
    sub_13185(2);++counter;
second_check:
    if(COUNTER_VALUE<13)goto second_next;
    MESSAGE(1);dword_51A83=0;sub_25977(-1,0);dword_53AFB=1;sub_1366A(100);dword_53AFB=0;
    sub_135DD(0,43);sub_25977(11,0);sub_1F525();sub_1366A(101);
    MESSAGE(2);dword_51A83=0;sub_1366A(102);
    MESSAGE(3);dword_51A83=0;sub_1366A(103);
    MESSAGE(4);dword_51A83=0;sub_1366A(104);
    MESSAGE(5);dword_53AFB=1;dword_51A83=0;sub_1366A(105);dword_53AFB=0;
    dword_53C03=31;sub_205DA();dword_51A83=0;sub_135DD(5,42);sub_10B4E(1);sub_1366A(90);
    MESSAGE(0);dword_51A83=0;sub_1366A(91);
    MESSAGE(1);dword_51A83=0;sub_1366A(92);
    MESSAGE(2);dword_51A83=0;sub_10B4E(3);sub_135DD(4,41);
    MESSAGE(3);dword_51A83=0;sub_1366A(93);
    MESSAGE(4);dword_51A83=0;sub_32975(2);sub_10B4E(5);
    MESSAGE(5);dword_51A83=0;sub_1366A(94);
    MESSAGE(6);dword_51A83=0;sub_1366A(95);
    MESSAGE(7);dword_51A83=0;sub_1366A(96);
    MESSAGE(8);dword_51A83=0;sub_1366A(97);
    MESSAGE(9);sub_25977(-1,0);dword_51A83=0;dword_53AFB=1;sub_1366A(98);dword_53AFB=0;
    dword_53C03=0;sub_112A5(0);sub_112A5(9);sub_112A5(4);sub_112A5(30);
    sub_205DA();dword_51A83=0;sub_135DD(4,12);sub_1366A(0);j___delay(200);
    MESSAGE(0);dword_51A83=0;j___delay(200);
    sub_135DD(0,0);sub_32999(1);sub_1366A(1);sub_135DD(0,15);sub_32999(2);sub_1366A(2);
    MESSAGE(1);dword_51A83=0;j___delay(200);sub_1366A(5);sub_32975(9);sub_11CAC(0);j___delay(100);
    MESSAGE(2);sub_134E4();sub_12D7B(0);dword_53BF3=0;
}
#undef COUNTER_VALUE
#endif

#if defined(POST_SEQ_0) || defined(POST_SEQ_1) || defined(POST_SEQ_2)
void sub_2548C(void)
{
#ifdef POST_SEQ_1
    int counter;
#define RECORD ((unsigned char *)counter)
#else
    int counter;
    unsigned char *record;
#define RECORD record
#endif
    MESSAGE(10);sub_35BBA(20);
#ifdef POST_SEQ_1
    counter=(int)dword_53A45;counter+=1600;
#else
    record=dword_53A45+1600;
#endif
    RECORD[7]=126;RECORD[8]=126;MESSAGE(11);
    sub_10B4E(9);sub_135DD(9,8);sub_12CEA(15,10);sub_22253(dword_53BEB-1,15,10,15,10);
    MESSAGE(12);dword_51A83=0;
    sub_24B4D(20);j___delay(600);sub_24B4D(20);j___delay(600);sub_24B4D(20);
    MESSAGE(13);dword_51A83=0;
    sub_24B4D(20);j___delay(200);sub_24B4D(20);j___delay(200);sub_24B4D(20);
    MESSAGE(14);dword_51A83=0;
    sub_24B4D(20);j___delay(200);sub_24B4D(20);j___delay(100);sub_24B4D(40);j___delay(200);
    sub_35E5A();j___delay(300);sub_35E5A();j___delay(300);sub_35E5A();j___delay(300);MESSAGE(15);
#ifdef POST_SEQ_2
    counter=dword_53BEB;counter^=counter;
#else
    counter=0;
#endif
    goto fade_up_check;
fade_up_next:
    sub_11DF2(0,255,counter);j___delay(4);++counter;
fade_up_check:
    if(counter<64)goto fade_up_next;
    memset((void *)0xa0000,0,64000);j___delay(800);
    counter=62;goto fade_down_check;
fade_down_next:
    sub_11DF2(0,255,counter);j___delay(4);--counter;
fade_down_check:
    if(counter>=0)goto fade_down_next;
    sub_11506();++dword_53C03;
}
#undef RECORD
#endif
