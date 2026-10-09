/* 三個完整查詢函式的C區塊排列候選。
 * 固定原檔、IDA 9.4線性位址、原始名稱及caller見匹配主收據。
 * 失敗尾端依原始jge目標保留，未匹配不提高覆蓋或玩法語意。
 */
extern int dword_53BFB;
extern unsigned char * volatile dword_53BF7;
extern int sub_31860(int, int);

#ifdef F24B14
int sub_24B14(int value)
{
    int index = 0;
    goto check;
advance:
    ++index;
check:
    if (index >= 16) goto unavailable;
    if (sub_31860(index, value) == -1) goto advance;
    return 1;
unavailable:
    return -1;
}
#endif

#if defined(F24BDE) || defined(F33499)
#ifdef F24BDE
int sub_24BDE(int value)
#else
int sub_33499(int value)
#endif
{
    int index, offset;
    index = 0;
    goto check;
advance:
    ++index;
check:
    if (index >= dword_53BFB) goto unavailable;
    offset = index * 80;
    if ((int)(dword_53BF7 + offset)[8] != value) goto advance;
    return 1;
unavailable:
    return 0;
}
#endif
