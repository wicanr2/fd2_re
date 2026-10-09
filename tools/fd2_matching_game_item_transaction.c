/* 2F642完整C候選。IDA 9.4固定輸入、原始位址及caller見匹配主收據。
 * 保留原始名稱、欄位寬度與全部控制流程；宣告只作產碼表示。
 */
typedef struct {short values[6];} RawTwelve;
extern const RawTwelve unk_5272A,unk_52736;
extern unsigned char * volatile dword_53A45;
extern int dword_5413F,dword_53BFB,dword_53C57,dword_53AD9,dword_5412B,dword_5412F,dword_53AE1,dword_53A7D;
extern unsigned char byte_52659[];
extern int sub_2E6B8(void),sub_2DF6B(int,void *,int),sub_16C57(int),sub_19953(void),sub_1B722(int,int);
extern unsigned char *sub_4E56C(int);
extern void sub_2D31B(void),sub_1956B(int),sub_16559(int),sub_2E0BD(int,void *,int),sub_197E5(void),sub_2F4C6(void);
extern void sub_2D3FF(int),sub_1B8E7(int,int),sub_1B750(int);
extern void sub_15F84(int,int,void *,int,int,int,int,int,int);

#if defined(ITEM_INDEX_0) || defined(ITEM_INDEX_1) || defined(ITEM_INDEX_2) || defined(ITEM_INDEX_3)
void sub_2F642(void)
{
    RawTwelve first,second;
    unsigned char items[8],*record,*data,*slot;
    int unit,index,count,selection,item,response;
    first=unk_5272A;second=unk_52736;goto choose_unit;
again:
    sub_2D31B();
choose_unit:
    dword_5413F=dword_53BFB;response=sub_2E6B8();sub_2D31B();if(response==-1)goto done;
    unit=dword_53C57;
#ifdef ITEM_INDEX_1
    index=unit;
#endif
    record=dword_53A45+unit*80;count=0;
#ifdef ITEM_INDEX_0
    index=0;
#elif defined(ITEM_INDEX_1)
    index^=unit;
#elif defined(ITEM_INDEX_2)
    index=unit^unit;
#else
    index=unit;index-=unit;
#endif
    goto item_condition;
item_next:
    ++index;
item_condition:
    if(index>=8)goto item_list;
    slot=record+index*2;slot+=10;
    if((unsigned char)(slot[0]&128)!=0)goto item_next;
    items[count]=slot[1];++count;goto item_next;
item_list:
    if(count==0){
        dword_53AD9=(int)dword_53A45[unit*80+7]+1;sub_1956B(byte_52659[dword_5412B]);
        sub_15F84(dword_53A7D,second.values[dword_5412B],(void *)0xa94cc,320,205,76,74,19,1);
        sub_16559(count);sub_16C57(1);goto again;
    }
    dword_53C57=0;dword_5412F=0;sub_2E0BD(count,items,1);dword_5413F=count;
    if(sub_2DF6B(count,items,1)==-1)goto again;
    selection=dword_53C57;sub_2D31B();item=sub_1B722(unit,dword_53C57);dword_53AD9=item+181;
    data=sub_4E56C(item);dword_53AE1=(int)*(unsigned short *)(data+19)*3/4;
    sub_1956B(byte_52659[dword_5412B]);
    sub_15F84(dword_53A7D,first.values[dword_5412B],(void *)0xa94cc,320,205,76,74,19,1);
    sub_16559(0);response=sub_19953();sub_197E5();
    if(response==-1||dword_53C57==1)goto again;
    sub_2D31B();sub_2F4C6();sub_2D3FF(dword_53AE1);sub_1B8E7(unit,selection);sub_1B750(unit);goto choose_unit;
done:
    return;
}
#endif
