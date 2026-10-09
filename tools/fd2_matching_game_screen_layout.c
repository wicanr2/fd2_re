/* 2D669完整C布局候選。IDA 9.4固定輸入及原始caller見匹配主收據。
 * 沿用原始名稱、20-byte局部資料及有號除法；宣告僅作產碼表示。
 */
typedef struct { int offsets[4]; } RawScreenOffsets;
extern const RawScreenOffsets unk_526DA;
extern unsigned char *dword_53A49,*dword_54147;
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void sub_4E9E4(void *,void *,int);

#if defined(SCREEN_LAYOUT) || defined(SCREEN_LAYOUT_1) || defined(SCREEN_LAYOUT_2) || defined(SCREEN_LAYOUT_3) || defined(SCREEN_LAYOUT_4) || defined(SCREEN_LAYOUT_5) || defined(SCREEN_LAYOUT_6)
#ifdef SCREEN_LAYOUT_4
void sub_2D669(volatile int mode)
#else
void sub_2D669(int mode)
#endif
{
    unsigned char *buffer;
    RawScreenOffsets table;
    unsigned char *base,*image;
    int frame,offset;
#ifdef SCREEN_LAYOUT_1
    long next;
#elif defined(SCREEN_LAYOUT_3)
    unsigned next;
#else
    int next;
#endif
#ifdef SCREEN_LAYOUT_2
    long index;
#elif defined(SCREEN_LAYOUT_5)
    unsigned index;
#else
    int index;
#endif
#ifdef SCREEN_LAYOUT_6
    int row;
#else
#define row index
#endif
    table=unk_526DA;
    buffer=(unsigned char *)malloc(64000);
    memmove(buffer,(void *)0xa0000,64000);
    row=0;goto row_condition;
row_next:
    memset(buffer+(row+169)*320+201,74,104);++row;
row_condition:
    if((int)row<20)goto row_next;
    memmove(dword_53A49,buffer,64000);
    base=dword_53A49+54320;frame=0;goto frame_condition;
frame_next:
    memmove((void *)0xa0000,dword_53A49,64000);frame=next;
frame_condition:
    if(frame>=4)goto finish;
    memmove(dword_53A49,buffer,64000);
    index=0;goto item_condition;
item_next:
    ++index;
item_condition:
    next=frame+1;
    if((int)index>=4)goto frame_next;
    offset=index*2+3;
    image=dword_54147;image+=*(int *)(image+offset*4+6);
    offset=index*4;
    if(mode==0){
        sub_4E9E4(base+*(int *)((unsigned char *)&table+offset)/(int)(next=4-frame),image,320);
        goto item_next;
    }
    sub_4E9E4(base+*(int *)((unsigned char *)&table+offset)/(int)next,image,320);
    goto item_next;
finish:
    if(mode!=0)memmove((void *)0xa0000,buffer,64000);
    free(buffer);
}
#undef row
#endif
