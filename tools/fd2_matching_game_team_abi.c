/* 完整隊伍服務的普通C呼叫宣告候選。固定原檔、IDA Pro 9.4原名、caller及bytes見主收據。
 * 4E031原始15 bytes只修改AX且保存ESI，原作者宣告未知；__cdecl只作候選，不宣稱原版EBX被改寫。
 */
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53BF7,*dword_53A61,*dword_54147,*dword_53C5F;
extern int dword_53C57,dword_53C03,dword_53BFB,dword_53BF3,dword_53BDF,dword_54137;
extern int dword_53AD9,dword_53AE1,dword_53A7D;
extern unsigned char byte_526B9[],byte_52659[],byte_526A7[],byte_5265D;
extern unsigned char byte_51AAB,byte_53AF9,byte_51E61,byte_51E62;
#ifdef TEAM30_VOLATILE
extern const volatile short word_52669[];
#else
extern const short word_52669[];
#endif
extern const char aRb_6[],aFd2Sav_7[],aRb_7[],aFdiconB24_1[],aRb_8[],aFdiconB24_2[];
extern void *malloc(unsigned),*memmove(void *,const void *,unsigned),*memset(void *,int,unsigned);
extern void free(void *);
extern void *fopen(const char *,const char *);
extern int fclose(void *),sub_37072(void *,unsigned,unsigned,void *),sub_30550(void *,int);
extern int sub_309FF(void *),sub_30C22(int,void *),sub_31793(void *,void *),sub_311DC(int,void *,void *);
extern int sub_19953(void),sub_16C57(int),sub_11019(int,void *),sub_31860(int,int);
extern void sub_4DBD8(void *,unsigned),sub_2D31B(void),sub_1956B(int),sub_16559(int),sub_197E5(void);
extern void __cdecl sub_4E031(void);
extern int sub_4E4B9(int);
extern unsigned char *sub_4E48D(int);
extern void sub_2D516(int),sub_4E8AF(void *,void *,int),sub_187D6(void *,int,int,int,int);
extern void sub_25977(int,int),sub_2F4C6(void),sub_1B8E7(int,int),sub_2A2E8(int,int),sub_31602(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);
#define MESSAGE(INDEX) sub_15F84(dword_53A7D,INDEX,(void *)0xa94cc,320,205,76,74,19,1)

#ifdef TEAM313_ABI
void sub_31385(void)
{
    struct {unsigned char classes[32],units[32];int item;} local;
    struct TeamRecord {unsigned char raw[80];} *icons;
    unsigned char *record,*data;
    void *file;
    int count,response,unit,index,slot;
again:
    count=sub_31793(local.units,local.classes);
    if(!count){sub_1956B(byte_5265D);MESSAGE(591);sub_16559(count);sub_16C57(count);sub_2D31B();goto done;}
    sub_1956B(byte_5265D);MESSAGE(592);sub_16559(0);sub_16C57(1);sub_2D31B();sub_4E031();
    count=sub_311DC(count,local.units,local.classes);sub_2D31B();if(count==-1)goto done;
    sub_1956B(byte_5265D);unit=local.units[dword_53C57];local.item=local.classes[dword_53C57];
    record=dword_53A45+unit*80;dword_53AD9=(int)record[7]+1;MESSAGE(594);sub_4E031();
    response=sub_19953();sub_197E5();sub_2D31B();if(response==-1||dword_53C57!=0)goto again;
    if(local.item==52)slot=sub_31860(unit,90);
    else if(local.item>=50)slot=sub_31860(unit,byte_526A7[record[7]]);
    else goto change;
    sub_1B8E7(unit,slot);
change:
    sub_25977(16,1);sub_2A2E8(unit,local.item);sub_25977(11,0);
    data=sub_4E48D(local.item);record[32]=data[0];record[7]=local.item;
    if(dword_53A61)free(dword_53A61);
    file=fopen(aFdiconB24_2,aRb_8);dword_53BDF=0;index=0;goto icon_condition;
icon_next:
#if defined(TEAM313_TYPED)
    sub_11019((icons=(struct TeamRecord *)dword_53BF7,icons[index].raw[7]),file);
#else
    sub_11019((record=dword_53BF7,record[index*80+7]),file);
#endif
    ++index;
icon_condition:
    if(index<dword_53BFB)goto icon_next;
    fclose(file);sub_31602(unit);sub_4E031();goto again;
done:
    return;
}
#endif
