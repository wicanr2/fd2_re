/* 完整讀檔C布局試驗。固定原檔、IDA Pro 9.4原名、caller及bytes見主收據。
 * 導覽名稱、型別與slot位移不推定作者宣告，完整區間未匹配不增加覆蓋。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7,*dword_53A61,*dword_54147,*dword_53C5F;
extern int dword_53C57,dword_53C03,dword_53BFB,dword_53BF3,dword_53BDF,dword_54137;
extern int dword_53AD9,dword_53AE1,dword_53A7D;
extern unsigned char byte_526B9[],byte_52659[],byte_526A7[],byte_5265D;
extern unsigned char byte_51AAB,byte_53AF9,byte_51E61,byte_51E62;
extern const short word_52669[];
extern const char aRb_6[],aFd2Sav_7[],aRb_7[],aFdiconB24_1[],aRb_8[],aFdiconB24_2[];
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void *fopen(const char *,const char *);
extern int fclose(void *),sub_37072(void *,unsigned,unsigned,void *),sub_30550(void *,int);
extern int sub_309FF(void *),sub_30C22(int,void *),sub_31793(void *,void *),sub_311DC(int,void *,void *);
extern int sub_19953(void),sub_16C57(int),sub_11019(int,void *),sub_31860(int,int);
extern void sub_4DBD8(void *,unsigned),sub_2D31B(void),sub_1956B(int),sub_16559(int),sub_197E5(void),sub_4E031(void);
extern int sub_4E4B9(int);
extern unsigned char *sub_4E48D(int);
extern void sub_2D516(int),sub_4E8AF(void *,void *,int),sub_187D6(void *,int,int,int,int);
extern void sub_25977(int,int),sub_2F4C6(void),sub_1B8E7(int,int),sub_2A2E8(int,int),sub_31602(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A7D,INDEX,(void *)0xa94cc,320,205,76,74,19,1)

#if defined(LOAD_PTR) || defined(LOAD_STRUCT) || defined(LOAD_SCALE_FIRST)
void sub_301F4(void)
{
    unsigned char *buffer=malloc(22987),*slot;
    void *file;
    int choice,response,index,offset;
    unsigned char *roster;
    struct LoadRecord { unsigned char raw[80]; } *records;
    file=fopen(aFd2Sav_7,aRb_6);
    if(file){sub_37072(buffer,1,22987,file);sub_4DBD8(buffer,22987);fclose(file);}
    else memset(buffer,255,22987);
    dword_53C57=0;
again:
    choice=sub_30550(buffer,1);response=choice;sub_2D31B();if(choice==-1)goto check;
    slot=buffer+12587+dword_53C57*2600;
    if((int)slot[2560]==255)goto check;
    if(byte_526B9[slot[2560]]!=0){sub_1956B(byte_52659[0]);MESSAGE(479);goto message_done;}
    memmove(dword_53BF7,slot,2560);slot+=2560;
    dword_53C03=slot[0];dword_53BFB=slot[1];dword_53BF3=*(unsigned *)(slot+2);
    byte_51AAB=slot[6];byte_53AF9=slot[7];byte_51E61=slot[8];byte_51E62=slot[9];
    if(dword_53A61)free(dword_53A61);
    file=fopen(aFdiconB24_1,aRb_7);dword_53BDF=0;index=0;goto icon_condition;
icon_next:
#if defined(LOAD_PTR)
    sub_11019((roster=dword_53BF7,offset=((index<<2)+index)<<4,
        roster[offset+7]),file);
#elif defined(LOAD_STRUCT)
    sub_11019((records=(struct LoadRecord *)dword_53BF7,records[index].raw[7]),file);
#else
    sub_11019((offset=((index<<2)+index)<<4,roster=dword_53BF7,
        roster[offset+7]),file);
#endif
    ++index;
icon_condition:
    if(index<dword_53BFB)goto icon_next;
    fclose(file);dword_54137=sub_4E4B9(dword_53C03);response=-1;
    sub_1956B(byte_52659[0]);MESSAGE(478);
message_done:
    sub_16559(0);sub_16C57(0);sub_2D31B();
check:
    if(response!=-1)goto again;
    free(buffer);
}
#endif
