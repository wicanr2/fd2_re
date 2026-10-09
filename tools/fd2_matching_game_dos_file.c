/* 已知DOS服務與檔案流程的C產碼候選。IDA 9.4原名、位址、bytes及caller見主收據。
 * 只比完整原始函式；保留unknown分類，不由呼叫慣例或用途推定第三方來源。
 */
extern int __cdecl int386(int,void *,void *);
extern int __cdecl open(const char *,int),__cdecl close(int);
extern int __cdecl filelength(int),__cdecl read(int,void *,unsigned);
extern int __cdecl strcmp(const char *,const char *);
extern int dword_52754;
extern unsigned dword_360FF;
extern const char aLx[];
typedef void *(__cdecl *Allocate)(unsigned);
typedef void (__cdecl *Release)(void *);
extern Allocate off_52758;
extern Release off_5275C;
extern int __cdecl sub_36284(unsigned,unsigned);
extern int __cdecl sub_36900(const char *);
extern unsigned __cdecl sub_36107(unsigned,unsigned,unsigned,void *,unsigned);
typedef char RawWidthsMustBe32[(sizeof(unsigned)==4 && sizeof(void *)==4)?1:-1];
/* 呼叫處保留EBX的相容compiler視角，不定義inline code；實際callee副作用另依證據。 */
#if defined(RAW_ALLOC_2) || defined(RAW_LENGTH_2) || defined(RAW_LOAD_2)
#pragma aux int386 parm caller [] value [eax] modify exact [eax ecx edx];
#pragma aux open parm caller [] value [eax] modify exact [eax ecx edx];
#pragma aux close parm caller [] value [eax] modify exact [eax ecx edx];
#pragma aux filelength parm caller [] value [eax] modify exact [eax ecx edx];
#pragma aux read parm caller [] value [eax] modify exact [eax ecx edx];
#endif
#pragma off (check_stack)

#if defined(RAW_RANGE_0) || defined(RAW_RANGE_1) || defined(RAW_RANGE_2)
int __cdecl sub_36284(unsigned first,unsigned last)
{
    struct {unsigned in[7],out[7];} local;
    unsigned start,end,length;
#ifdef RAW_RANGE_0
    start=first<last?first:last;end=first<last?last:first;
#elif defined(RAW_RANGE_1)
    if(first<last){start=first;end=last;}else{start=last;end=first;}
#else
    if(first<last)start=first;else start=last;
    if(first<last)first=last;end=first;
#endif
    length=end-start+1;
    local.in[0]=0x600;local.in[1]=start>>16;local.in[2]=start&0xffff;
    local.in[4]=length>>16;local.in[5]=length&0xffff;
    int386(0x31,local.in,local.out);return local.out[6]==0;
}
#endif

#if defined(RAW_ALLOC_0) || defined(RAW_ALLOC_1) || defined(RAW_ALLOC_2)
int __cdecl sub_361CC(unsigned paragraphs,unsigned *linear,unsigned *segment,unsigned *selector)
{
    struct {unsigned in[7],out[7];} local;
    int result;
    local.in[0]=0x100;local.in[1]=paragraphs;int386(0x31,local.in,local.out);
    if(local.out[6])result=0;
    else{
        *segment=local.out[0]<<16;*linear=(local.out[0]&0xffff)<<4;
        *selector=local.out[3]&0xffff;result=1;
    }
    if(result){
#ifdef RAW_ALLOC_1
        unsigned first=*segment>>12;sub_36284(first,paragraphs*16+first-1);
#else
        sub_36284(*segment>>12,paragraphs*16+(*segment>>12)-1);
#endif
    }
    return result;
}
#endif

#if defined(RAW_LENGTH_0) || defined(RAW_LENGTH_1) || defined(RAW_LENGTH_2)
int __cdecl sub_36900(const char *path)
{
#ifdef RAW_LENGTH_1
    register int handle,length;
#else
    int handle,length;
#endif
    dword_52754=0;handle=open(path,0x200);
    if(handle==-1){dword_52754=3;return -1;}
    length=filelength(handle);if(length==-1)dword_52754=5;
    close(handle);
    return length;
}
#endif

#if defined(RAW_LOAD_0) || defined(RAW_LOAD_1) || defined(RAW_LOAD_2)
void *__cdecl sub_36955(const char *path,void *dest)
{
#ifdef RAW_LOAD_1
    register int length,handle;
    register void *data;
#else
    int length,handle;
    void *data;
#endif
    dword_52754=0;length=sub_36900(path);
    if(length==-1){
open_failed:
        dword_52754=3;
failed:
        return (void *)0;
    }
    if(!dest)data=off_52758(length);else data=dest;
    if(!data){dword_52754=2;goto failed;}
    handle=open(path,0x200);
    if(handle==-1){off_5275C(data);goto open_failed;}
    if(read(handle,data,length)!=length){off_5275C(data);dword_52754=5;goto failed;}
    close(handle);return data;
}
#endif

#if defined(RAW_LX_0) || defined(RAW_LX_1)
unsigned __cdecl sub_36344(unsigned input,unsigned flags)
{
    struct {unsigned header[43],entry[6],offset,signature;} local;
    unsigned sum,position,index;
    int handle;
    local.signature=dword_360FF;
    sum=0;
    if(flags&1)handle=(int)input;
    else{handle=open((const char *)input,0x200);if(handle==-1)return 0;}
    sub_36107(handle,0x3c,flags,&local.offset,4);
    sub_36107(handle,local.offset,flags,&local.signature,2);
    if(strcmp((const char *)&local.signature,aLx)){close(handle);return 0;}
    sub_36107(handle,local.offset,flags,local.header,0xac);
    position=local.offset+local.header[16];index=0;goto condition;
next:
#ifdef RAW_LX_1
    position=sub_36107(handle,position,flags,local.entry,24);
#else
    sub_36107(handle,position,flags,local.entry,24);
#endif
    sum+=local.entry[0];++index;
condition:
    if(index<local.header[17])goto next;
    if(!(flags&1))close(handle);
    return local.header[17]*15+sum;
}
#endif
