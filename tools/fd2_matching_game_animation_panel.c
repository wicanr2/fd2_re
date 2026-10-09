/* 完整演出與介面資料流C候選。固定IDA 9.4原名、位址、bytes及consumer見主收據。
 * 原始EDI初值與作者宣告不猜補；局部型別、欄位及ABI視角只作產碼表示。
 */
extern unsigned char *dword_53A45,*dword_53A49,*dword_53A5D,*dword_53A55,*dword_53A61,*dword_54147;
extern void *dword_54107,*dword_54117;
extern int dword_54153,dword_5412F,dword_53A7D;
extern const char aBgDat[],aFiganiDat[],aTaiDat[],aFdshapDat[];
extern void *malloc(unsigned),*memset(void *,int,unsigned),*memmove(void *,const void *,unsigned);
extern void free(void *),sub_12E38(int,int,void *),sub_1F882(void),sub_1F525(void),sub_17AA9(int),sub_11CAC(int);
extern void *sub_111BA(const char *,void *,int),*sub_2BC9A(void *);
extern void sub_2A289(void *,int),sub_4E63D(void *,int,int,void *,int,int),sub_2935B(void *,int,void *,int,int);
extern void sub_11EB0(void *,int,void *,int,int,int),sub_25A96(void *,int,int);
extern void sub_29164(int,int,void *,void *,void *,void *,void *);
extern void sub_2EFB7(int,int,int *),sub_4DF4C(void *,void *,int),sub_4E8AF(void *,void *,int);
extern int sub_2EF8F(int,int);
extern void sub_187D6(void *,int,int,int,int),sub_15F84(int,int,void *,int,int,int,int,int,int);
typedef char RawPointerMustBe32[(sizeof(void *)==4 && sizeof(int)==4)?1:-1];
#if defined(PANEL_LOCAL_6) || defined(PANEL_LOCAL_7)
#define PANEL_LOCAL_5
#endif
#ifdef PANEL_LOCAL_7
extern int fd2_raw_number_result(void *,int,int,int,int);
#pragma aux fd2_raw_number_result "sub_187D6" parm caller [] value [eax] modify exact [eax ecx edx];
#endif
/* 允許EBX被修改的保守compiler契約；實際callee保存EBX，不宣稱作者原型。 */
#ifdef PANEL_LOCAL_5
#pragma aux sub_4DF4C parm caller [] modify exact [eax ebx ecx edx];
#pragma aux sub_4E8AF parm caller [] modify exact [eax ebx ecx edx];
#endif
#if defined(PANEL_LOCAL_3) || defined(PANEL_LOCAL_4) || defined(PANEL_LOCAL_5)
#define PANEL_LOCAL_2
#endif

#if defined(ANIM_LOCAL_0) || defined(ANIM_LOCAL_1) || defined(ANIM_LOCAL_2) || defined(ANIM_LOCAL_3)
/* 原始29164保存／還原EDI，此視角只描述call之後可讀的EDI，不推定值為零。 */
extern int fd2_raw_edi_view(int,int,void *,void *,void *,void *,void *);
#pragma aux fd2_raw_edi_view "sub_29164" parm caller [] value [edi] modify exact [eax ecx edx edi];
#ifdef ANIM_LOCAL_3
void sub_28784(int,int);
#pragma aux sub_28784 parm caller [edi] [] modify exact [eax ecx edx gs];
void sub_28784(int index,int unit)
#else
void sub_28784(int unit)
#endif
{
    struct {unsigned char coordinates[8];int background;void *tai,*figure,*tai_base,*figure_base,*saved,*source;} local;
    unsigned char *record,*animation,*frame;
    void *work;
    int kind,figure_index;
#ifndef ANIM_LOCAL_3
    int index;
#endif
    dword_54107=0;record=dword_53A45+unit*80;kind=record[7];free(dword_53A49);free(dword_53A5D);dword_53A5D=0;
#ifdef ANIM_LOCAL_1
#define STORE(P,VALUE) (*(void * volatile *)&(P)=(VALUE))
#else
#define STORE(P,VALUE) ((P)=(VALUE))
#endif
    STORE(local.saved,malloc(64000));local.source=local.saved;work=malloc(128000);memset(local.saved,0,64000);
    sub_12E38(record[0],record[1],local.coordinates);local.background=local.coordinates[6];
    STORE(local.tai_base,sub_111BA(aTaiDat,0,3));local.tai=local.tai_base;
    figure_index=kind*3;STORE(local.figure_base,sub_111BA(aFiganiDat,0,figure_index));local.figure=local.figure_base;
    animation=sub_111BA(aFiganiDat,0,++figure_index);sub_1F882();
    dword_54107=sub_111BA(aBgDat,dword_54107,local.background);sub_2A289(local.saved,unit);
    sub_4E63D(dword_54107,0,50,local.saved,320,-1);dword_54117=sub_2BC9A(animation);
#ifdef ANIM_LOCAL_3
    sub_29164(unit,1,local.figure_base,local.figure_base,work,local.saved,local.tai_base);
#elif defined(ANIM_LOCAL_2)
    sub_29164(unit,1,local.figure_base,local.figure_base,work,local.saved,local.tai_base);index=0;
#else
    index=fd2_raw_edi_view(unit,1,local.figure_base,local.figure_base,work,local.saved,local.tai_base);
#endif
    goto frame_check;
frame_draw:
    sub_11EB0(work,320,local.source,320,320,200);sub_2935B(animation,index,work,320,-1);
    sub_11EB0((void *)0xa0000,320,work,320,320,200);sub_17AA9(frame[6]);++index;
frame_check:
    if(index>=(int)animation[0])goto frame_done;
    frame=animation+*(int *)(animation+index*4+8);
    if(frame[5])sub_25A96(dword_54117,frame[5],1);goto frame_draw;
frame_done:
    free(local.source);free(work);free(local.figure);free(animation);free(dword_54107);free(local.tai);
    dword_53A49=malloc(153216);dword_53A5D=sub_111BA(aFdshapDat,dword_53A5D,dword_53A55[0]*2);
    sub_17AA9(6);sub_1F882();memset((void *)0xa0000,0,64000);sub_11CAC(1);
    sub_25A96(dword_54117,-1,1);if(dword_54117)free(dword_54117);sub_1F525();
}
#undef STORE
#endif

#if defined(PANEL_LOCAL_0) || defined(PANEL_LOCAL_1) || defined(PANEL_LOCAL_2)
#define SOURCE(OFFSET) (dword_54147+*(int *)(dword_54147+(OFFSET)))
#ifdef PANEL_LOCAL_7
int sub_2EBE0(int count,unsigned char *list,int mode,int selected,unsigned char *dest)
#elif defined(PANEL_LOCAL_5)
void sub_2EBE0(int count,unsigned char *list,int mode,int selected,unsigned char *dest)
#elif defined(PANEL_LOCAL_4)
void sub_2EBE0(volatile int count,unsigned char * volatile list,volatile int mode,volatile int selected,unsigned char * volatile dest)
#elif defined(PANEL_LOCAL_3)
void sub_2EBE0(int count,unsigned char *list,int mode,volatile int selected,unsigned char * volatile dest)
#elif defined(PANEL_LOCAL_2)
void sub_2EBE0(int count,unsigned char *list,int mode,int selected,unsigned char * volatile dest)
#else
void sub_2EBE0(int count,unsigned char *list,int mode,int selected,unsigned char *dest)
#endif
{
    struct {int current[4];unsigned char *row4,*row13;int ratio,count,mode,max1,max0,max3,max2,y;} local;
    unsigned char *record,*source,*lower,*upper;
    int index,unit,color,ratio;
#ifdef PANEL_LOCAL_7
    int result;
#endif
#ifdef PANEL_LOCAL_6
    unsigned source_address;
#define GLYPH(DEST,OFFSET) do {source=SOURCE(OFFSET);sub_4E8AF(DEST,source,320);} while(0)
#else
#define GLYPH(DEST,OFFSET) sub_4E8AF(DEST,SOURCE(OFFSET),320)
#endif
    local.mode=dword_54153;if(local.mode==3)local.mode=1;
#ifdef PANEL_LOCAL_7
    result=count;local.count=result;
#else
    local.count=count;
#endif
    if(local.count>3)local.count=3;
    index=0;goto row_check;
row_next:
    ++index;
row_check:
    if(index>=local.count)goto done;
    unit=list[dword_5412F+index];sub_2EFB7(unit,mode,local.current);
    record=dword_53A45+unit*80;
    local.max0=*(unsigned short *)(record+72);local.max1=*(unsigned short *)(record+74);
    local.max2=*(unsigned short *)(record+76);local.max3=*(unsigned short *)(record+78);
    local.y=index*26+117;
#ifdef PANEL_LOCAL_6
    source_address=(unsigned)dword_53A61+unit*48;
    source_address=*(volatile unsigned *)(source_address+local.mode*4);
    source_address+=(unsigned)dword_53A61;source=(void *)source_address;
#else
    source=dword_53A61+*(int *)(dword_53A61+unit*48+local.mode*4);
#endif
    sub_4DF4C(source,dest+local.y*320+14,320);
    color=205;if(dword_5412F+index==selected)color=201;
#ifdef PANEL_LOCAL_2
    sub_15F84(dword_53A7D,(int)record[8]+1,(local.row4=dest+(local.y+4)*320)+40,320,color,76,0,0,0);
#else
    local.row4=dest+(local.y+4)*320;
    sub_15F84(dword_53A7D,(int)record[8]+1,local.row4+40,320,color,76,0,0,0);
#endif
    ratio=sub_2EF8F(local.max0,local.current[0]);
#ifdef PANEL_LOCAL_2
    GLYPH((lower=dest+(local.y+3)*320)+122,78);
#else
    lower=dest+(local.y+3)*320;
    GLYPH(lower+122,78);sub_187D6(lower+137,320,local.max0,ratio,3);
#endif
#ifdef PANEL_LOCAL_2
    sub_187D6(lower+137,320,local.max0,ratio,3);
#endif
    GLYPH(local.row4+157,94);sub_187D6(lower+165,320,local.current[0],ratio,3);
#ifdef PANEL_LOCAL_1
#define STORE_RATIO(VALUE) (*(volatile int *)&local.ratio=(VALUE))
#else
#define STORE_RATIO(VALUE) (local.ratio=(VALUE))
#endif
    STORE_RATIO(sub_2EF8F(local.max1,local.current[1]));
#ifdef PANEL_LOCAL_2
    GLYPH((upper=dest+(local.y+12)*320)+122,82);sub_187D6(upper+137,320,local.max1,local.ratio,3);
    GLYPH((local.row13=dest+(local.y+13)*320)+157,94);
#else
    upper=dest+(local.y+12)*320;
    GLYPH(upper+122,82);sub_187D6(upper+137,320,local.max1,local.ratio,3);
    local.row13=dest+(local.y+13)*320;
    GLYPH(local.row13+157,94);
#endif
    sub_187D6(upper+165,320,local.current[1],local.ratio,3);
    STORE_RATIO(sub_2EF8F(local.max2,local.current[2]));
    GLYPH(lower+196,86);sub_187D6(lower+214,320,local.max2,local.ratio,3);
    GLYPH(local.row4+234,94);sub_187D6(lower+242,320,local.current[2],local.ratio,3);
    ratio=sub_2EF8F(local.max3,local.current[3]);
    GLYPH(upper+196,90);sub_187D6(upper+214,320,local.max3,ratio,3);
    GLYPH(local.row13+234,94);
#ifdef PANEL_LOCAL_7
    result=fd2_raw_number_result(upper+242,320,local.current[3],ratio,3);
#else
    sub_187D6(upper+242,320,local.current[3],ratio,3);
#endif
    goto row_next;
done:
#ifdef PANEL_LOCAL_7
    return result;
#else
    return;
#endif
}
#undef SOURCE
#undef STORE_RATIO
#undef GLYPH
#endif

#ifdef PANEL_LOCAL_8
extern void fd2_raw_icon(unsigned,unsigned,int),fd2_raw_glyph(unsigned,unsigned,int);
extern int fd2_raw_number(unsigned,int,int,int,int);
extern void fd2_raw_caption(int,int,unsigned,int,int,int,int,int,int);
#pragma aux fd2_raw_icon "sub_4DF4C" parm caller [] modify exact [eax ebx ecx edx];
#pragma aux fd2_raw_glyph "sub_4E8AF" parm caller [] modify exact [eax ebx ecx edx];
#pragma aux fd2_raw_number "sub_187D6" parm caller [] value [eax] modify exact [eax ecx edx];
#pragma aux fd2_raw_caption "sub_15F84" parm caller [] modify exact [eax ecx edx];
#define RAW_SOURCE(OFFSET) (*(unsigned *)(dword_54147+(OFFSET))+(unsigned)dword_54147)
int sub_2EBE0(int count,unsigned list,int mode,int selected,unsigned dest)
{
    int current[4];
    unsigned row4,row13;
    int other_ratio,limit,view,max1,max0,max3,max2,y;
    int result,index,unit,color,ratio;
    unsigned record,source,lower,upper;
    view=dword_54153;if(view==3)view=1;result=count;limit=count;if(limit>3)limit=3;
    for(index=0;index<limit;++index){
        unit=*(unsigned char *)(list+dword_5412F+index);sub_2EFB7(unit,mode,current);
        record=(unsigned)dword_53A45+unit*80;
        max0=*(unsigned short *)(record+72);max1=*(unsigned short *)(record+74);
        max2=*(unsigned short *)(record+76);max3=*(unsigned short *)(record+78);y=26*index+117;
        source=(unsigned)dword_53A61+*(unsigned *)(dword_53A61+unit*48+4*view);
        fd2_raw_icon(source,dest+320*y+14,320);
        color=205;if(dword_5412F+index==selected)color=201;
        row4=dest+320*(y+4);fd2_raw_caption(dword_53A7D,*(unsigned char *)(record+8)+1,row4+40,320,color,76,0,0,0);
        ratio=sub_2EF8F(max0,current[0]);lower=dest+320*(y+3);
        fd2_raw_glyph(lower+122,RAW_SOURCE(78),320);fd2_raw_number(lower+137,320,max0,ratio,3);
        fd2_raw_glyph(row4+157,RAW_SOURCE(94),320);fd2_raw_number(lower+165,320,current[0],ratio,3);
        other_ratio=sub_2EF8F(max1,current[1]);upper=dest+320*(y+12);
        fd2_raw_glyph(upper+122,RAW_SOURCE(82),320);fd2_raw_number(upper+137,320,max1,other_ratio,3);
        row13=dest+320*(y+13);fd2_raw_glyph(row13+157,RAW_SOURCE(94),320);fd2_raw_number(upper+165,320,current[1],other_ratio,3);
        other_ratio=sub_2EF8F(max2,current[2]);
        fd2_raw_glyph(lower+196,RAW_SOURCE(86),320);fd2_raw_number(lower+214,320,max2,other_ratio,3);
        fd2_raw_glyph(row4+234,RAW_SOURCE(94),320);fd2_raw_number(lower+242,320,current[2],other_ratio,3);
        ratio=sub_2EF8F(max3,current[3]);
        fd2_raw_glyph(upper+196,RAW_SOURCE(90),320);fd2_raw_number(upper+214,320,max3,ratio,3);
        fd2_raw_glyph(row13+234,RAW_SOURCE(94),320);result=fd2_raw_number(upper+242,320,current[3],ratio,3);
    }
    return result;
}
#undef RAW_SOURCE
#endif
