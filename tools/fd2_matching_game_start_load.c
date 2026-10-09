/* sub_25EBB完整C候選。存檔偏移與欄位只按原始讀寫寬度表示。
 * 原始名稱、LE表基址及caller見主收據；不推定原作者型別。
 */
#pragma pack(1)
typedef struct {
    unsigned char byte0,byte1;
    int dword2;
    unsigned char byte6,byte7,byte8,byte9;
} RawSavedMeta;
typedef struct {unsigned char bytes[2560];RawSavedMeta meta;unsigned char tail[30];} RawSavedSlot;
#pragma pack()
typedef char SavedMetaMustBe10[sizeof(RawSavedMeta)==10?1:-1];
typedef char SavedSlotMustBe2600[sizeof(RawSavedSlot)==2600?1:-1];
extern int dword_53C03,dword_53BFB,dword_53C57,dword_53BF3;
extern void *dword_53A65,*dword_54147,*dword_53BF7;
extern unsigned char byte_51AAC,byte_51AAB,byte_53AF9,byte_51E61,byte_51E62,byte_51E63[];
extern const char aFdotherDat[],aFd2Sav_4[],unk_50220[];
extern void (*funcs_25E3A[])(void);
extern int sub_1F894(void),sub_30550(void *,int),sub_2CAD7(void);
extern void sub_1F882(void),sub_4E031(void),sub_2D31B(void),sub_10010(void);
extern void sub_25977(int,int),sub_11D40(int,int,int),sub_4DBD8(void *,int);
extern void *sub_111BA(const char *,void *,int),*malloc(unsigned),*memset(void *,int,unsigned),*memmove(void *,const void *,unsigned);
extern void *fopen(const char *,const char *);
extern unsigned sub_37072(void *,unsigned,unsigned,void *);
extern int fclose(void *);
extern void free(void *);
#if defined(START_LOAD_3) || defined(START_LOAD_4)
#define START_LOAD_1
#endif

int sub_25EBB(void)
{
    int mode,selected;
#ifdef START_LOAD_1
    unsigned char *cursor,*buffer;
#else
    unsigned char *buffer,*cursor;
#endif
    RawSavedMeta *meta;
    void *file;
    mode=sub_1F894();
    if(mode==0){
        sub_1F882();dword_53C03=0;dword_53A65=sub_111BA(aFdotherDat,dword_53A65,0);
        dword_53BFB=0;byte_51AAC=0;funcs_25E3A[dword_53C03]();
        sub_25977(byte_51E63[dword_53C03],0);byte_51AAC=1;sub_4E031();return 0;
    }else if(mode==1){
        dword_54147=sub_111BA(aFdotherDat,dword_54147,13);sub_1F882();
        dword_53A65=sub_111BA(aFdotherDat,dword_53A65,0);
        memset((void *)0xa0000,0,64000);sub_11D40(0,255,0);
        cursor=malloc(22987);buffer=cursor;file=fopen(aFd2Sav_4,unk_50220);
        if(file){sub_37072(cursor,1,22987,file);sub_4DBD8(cursor,22987);fclose(file);}
        else memset(cursor,255,22987);
        dword_53C57=0;
select_again:
        selected=sub_30550(buffer,0);
        if(selected!=-1){
#ifdef START_LOAD_1
            cursor=buffer+12587+dword_53C57*2600;
#elif defined(START_LOAD_2)
            cursor=(unsigned char *)((RawSavedSlot *)(buffer+12587)+dword_53C57);
#else
            cursor=buffer+dword_53C57*2600+12587;
#endif
            memmove(dword_53BF7,cursor,2560);cursor+=2560;meta=(RawSavedMeta *)cursor;
            dword_53C03=meta->byte0;dword_53BFB=meta->byte1;dword_53BF3=meta->dword2;
            byte_51AAB=meta->byte6;byte_53AF9=meta->byte7;byte_51E61=meta->byte8;byte_51E62=meta->byte9;
            if(dword_53C03==255)selected=0;
        }
        sub_2D31B();if(selected==0)goto select_again;
        free(buffer);free(dword_54147);dword_54147=0;
        if(selected==1){
            byte_51AAC=0;
#ifdef START_LOAD_4
            if((selected=sub_2CAD7())==0){funcs_25E3A[dword_53C03]();sub_25977(byte_51E63[dword_53C03],selected);}
#elif defined(START_LOAD_3)
            selected=sub_2CAD7();
            if(selected==0){funcs_25E3A[dword_53C03]();sub_25977(byte_51E63[dword_53C03],0);}
#else
            selected=sub_2CAD7();
            if(selected==0){funcs_25E3A[dword_53C03]();sub_25977(byte_51E63[dword_53C03],selected);}
#endif
            byte_51AAC=1;
        }
        sub_4E031();return selected;
    }
    sub_25977(-1,0);sub_10010();sub_25977(byte_51E63[dword_53C03],0);return 0;
}
