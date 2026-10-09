/* 完整整備畫面轉換C候選。原始IDA 9.4位址、原名、bytes及caller見主收據。
 * 保留有號除法、表格複製與工作緩衝順序，導覽型別不推定作者宣告。
 */
typedef struct { int offsets[4]; } RawScreenOffsets;
extern const RawScreenOffsets unk_526DA;
extern unsigned char *dword_53A49,*dword_54147;
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void sub_4E9E4(void *,void *,int);

#ifdef F2D669
void sub_2D669(int mode)
{
    struct { RawScreenOffsets table; unsigned char *buffer; } local;
    unsigned char *base,*image;
    int row,frame,next,index,offset;
    local.table=unk_526DA;
    local.buffer=malloc(64000);
    memmove(local.buffer,(void *)0xa0000,64000);
    row=0;goto row_condition;
row_next:
    memset(local.buffer+(row+169)*320+201,74,104);++row;
row_condition:
    if(row<20)goto row_next;
    memmove(dword_53A49,local.buffer,64000);
    base=dword_53A49+54320;frame=0;goto frame_condition;
frame_next:
    memmove((void *)0xa0000,dword_53A49,64000);frame=next;
frame_condition:
    if(frame>=4)goto finish;
    memmove(dword_53A49,local.buffer,64000);
    index=0;goto item_condition;
item_next:
    ++index;
item_condition:
    next=frame+1;
    if(index>=4)goto frame_next;
    image=dword_54147;image+=*(int *)(image+(index*2+3)*4+6);
    offset=index*4;
    if(mode==0)next=4-frame;
    sub_4E9E4(base+*(int *)((unsigned char *)&local.table+offset)/next,image,320);
    goto item_next;
finish:
    if(mode!=0)memmove((void *)0xa0000,local.buffer,64000);
    free(local.buffer);
}
#endif
