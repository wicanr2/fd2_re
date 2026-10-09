/* 完整價格服務的等價C乘積保存候選。固定原檔、IDA Pro 9.4原名、caller及bytes見主收據。
 * signed word乘byte的結果在int32範圍；型別與暫存值只作產碼導航，不推定作者宣告。
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

#if defined(PRICE_FORM_00) || defined(PRICE_FORM_01) || defined(PRICE_FORM_02) || defined(PRICE_FORM_03) || defined(PRICE_FORM_04) || defined(PRICE_FORM_05) || defined(PRICE_FORM_06) || defined(PRICE_FORM_07) || defined(PRICE_FORM_08) || defined(PRICE_FORM_09) || defined(PRICE_FORM_10) || defined(PRICE_FORM_11)
void sub_30DC3(void)
{
    unsigned char units[32],*record,*image;
    int count,response,price,index,unit,quantity;
again:
    count=sub_309FF(units);
    if(!count){sub_1956B(byte_5265D);MESSAGE(588);sub_16559(count);sub_16C57(count);sub_2D31B();goto done;}
    sub_1956B(byte_5265D);MESSAGE(589);sub_16559(0);sub_16C57(1);sub_2D31B();
    response=sub_30C22(count,units);sub_2D31B();if(response==-1)goto done;
    sub_1956B(byte_5265D);index=dword_53C57;unit=units[index];
    index=unit<<2;index+=unit;index<<=4;unit=(unsigned)dword_53A45;
    record=(unsigned char *)(index+unit);
    dword_53AD9=(int)record[8]+1;price=word_52669[record[32]];quantity=(int)record[33];
#if defined(PRICE_FORM_00)
    dword_53AE1=quantity*price;
#elif defined(PRICE_FORM_01)
    quantity*=price;dword_53AE1=quantity;
#elif defined(PRICE_FORM_02)
    price*=quantity;dword_53AE1=price;
#elif defined(PRICE_FORM_03)
    unit=quantity;unit*=price;dword_53AE1=unit;
#elif defined(PRICE_FORM_04)
    index=quantity;index*=price;dword_53AE1=index;
#elif defined(PRICE_FORM_05)
    response=quantity;response*=price;dword_53AE1=response;
#elif defined(PRICE_FORM_06)
    dword_53AE1=price*quantity;
#elif defined(PRICE_FORM_07)
    price=quantity*price;dword_53AE1=price;
#elif defined(PRICE_FORM_08)
    quantity=price*quantity;dword_53AE1=quantity;
#elif defined(PRICE_FORM_09)
    unit=price;unit*=quantity;dword_53AE1=unit;
#elif defined(PRICE_FORM_10)
    index=price;index*=quantity;dword_53AE1=index;
#elif defined(PRICE_FORM_11)
    response=price;response*=quantity;dword_53AE1=response;
#endif
    MESSAGE(590);response=sub_19953();sub_197E5();
    if(response==-1||dword_53C57!=0)goto back;
    if(dword_53BF3>=dword_53AE1)goto accept;
    sub_15F84(dword_53A7D,504,(void *)0xac44c,320,205,76,74,19,1);sub_16559(0);sub_16C57(1);
back:
    sub_2D31B();goto again;
accept:
    sub_2D516(dword_53AE1);record[5]=0;*(unsigned short *)(record+64)=*(unsigned short *)(record+66);
    image=dword_54147;image+=*(unsigned *)(image+10);sub_4E8AF(dword_53C5F+30405,image,320);
    sub_187D6(dword_53C5F+31696,320,dword_53BF3,31,8);sub_2D31B();
    sub_25977(17,1);sub_2F4C6();sub_25977(11,1);goto again;
done:
    return;
}
#endif
