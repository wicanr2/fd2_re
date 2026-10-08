/* 短服務及callback C候選。固定原檔、IDA Pro 9.4原名、caller及bytes見主收據。
 * 純C產碼比對；不因位址區段、呼叫慣例或名字就提升第三方runtime分類。
 */
extern unsigned char *dword_53AD5,*dword_53A55;
extern int dword_53BEF;
extern int sub_4E893(void);
extern void sub_13512(int);
typedef void *(__cdecl *Allocate)(unsigned);
typedef void (__cdecl *Release)(void *);
extern Allocate off_52758;
extern Release off_5275C;
extern int __cdecl sub_36284(unsigned,unsigned),__cdecl sub_362F1(unsigned,unsigned);
extern int __cdecl sub_36316(void *,unsigned),__cdecl sub_3632D(void *,unsigned);
extern void * __cdecl memcpy(void *,const void *,unsigned);
extern long __cdecl sub_3CC00(int,long,int);
extern int __cdecl read(int,void *,unsigned);

#ifdef F35EBE
void sub_35EBE(void){dword_53AD5[19]=1;}
#endif
#ifdef F35ED2
void sub_35ED2(void){dword_53AD5[20]=1;}
#endif
#ifdef F35EE6
void sub_35EE6(void)
{
    unsigned char value=*(volatile unsigned char *)&dword_53BEF;
    int random;
    volatile int base;
    ++value;dword_53A55[9]=value;random=sub_4E893();base=dword_53AD5[21];
    sub_13512(random%3+base);sub_13512((random+1)%3+(int)dword_53AD5[21]);
}
#endif
#ifdef F35F6F
void sub_35F6F(void)
{
    unsigned char value;
    unsigned address;
    ++dword_53AD5[16];value=*(volatile unsigned char *)&dword_53BEF;++value;
    address=(unsigned)dword_53A55;*(unsigned char *)(address+3)=value;
}
#endif
#ifdef EMPTY_STACK_GROUP
void sub_360D8(void){}
void sub_360E3(void){}
void sub_360EA(void){}
void sub_360F1(void){}
void sub_360F8(void){}
#endif

#pragma off (check_stack)
#ifdef F36107
unsigned __cdecl sub_36107(unsigned input,unsigned position,unsigned flags,void *dest,unsigned length)
{
    unsigned next=position+length;
    if(flags&1)memcpy(dest,(void *)(input+position),length);
    else{sub_3CC00((int)input,position,0);read((int)input,dest,length);}
    return next;
}
#endif
#ifdef F3615E
Allocate __cdecl sub_3615E(Allocate value)
{
    Allocate old=off_52758;off_52758=value;return old;
}
#endif
#ifdef F3616E
Release __cdecl sub_3616E(Release value)
{
    Release old=off_5275C;off_5275C=value;return old;
}
#endif
#ifdef F3617E
void *__cdecl sub_3617E(unsigned length)
{
    void *result=off_52758(length);if(result)sub_36316(result,length);return result;
}
#endif
#ifdef F361A5
void __cdecl sub_361A5(void *input,unsigned length)
{
    if(input){sub_3632D(input,length);off_5275C(input);}
}
#endif
#ifdef F36316
int __cdecl sub_36316(void *input,unsigned length)
{
    return sub_36284((unsigned)input,(unsigned)input+length);
}
#endif
#ifdef F3632D
int __cdecl sub_3632D(void *input,unsigned length)
{
    return sub_362F1((unsigned)input,(unsigned)input+length);
}
#endif
#pragma on (check_stack)
