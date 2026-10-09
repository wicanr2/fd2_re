/* 數字動畫完整C候選。固定原檔與IDA Pro 9.4完整區間見匹配主收據。
 * 60-byte局部資料及三個20-byte陣列來自原始堆疊使用；型別只作產碼導航。
 */
extern int dword_53BF3;
extern const char a08d_1[],a08d_2[];
extern int __cdecl sprintf(char *,const char *,...);
extern void sub_375B2(int),sub_2D620(void *,int,int);

#ifdef F2D3FF
void sub_2D3FF(int difference)
{
    struct {
        unsigned char current[20];
        unsigned char old[20];
        unsigned char step[20];
    } digits;
    int index,digit,equal,amount,base,pattern,before,after;
    sprintf((char *)digits.old,a08d_1,dword_53BF3);
    index=0;goto old_condition;
old_next:
    digits.old[index]-=48;++index;
old_condition:
    if(index<8)goto old_next;
    dword_53BF3+=difference;
    sprintf((char *)digits.current,a08d_2,dword_53BF3);
    index=0;goto current_condition;
current_next:
    digits.current[index]-=48;++index;
current_condition:
    if(index<8)goto current_next;
again:
    equal=1;
    index=0;goto compare_condition;
compare_zero:
    digits.step[index]=0;
compare_next:
    ++index;
compare_condition:
    if(index>=8)goto compare_done;
    before=digits.old[index];after=digits.current[index];
    if(after==before)goto compare_zero;
    digits.step[index]=1;equal=0;goto compare_next;
compare_done:
    if(equal)goto check;
    index=0;goto frame_condition;
frame_next:
    sub_375B2(10);++index;
frame_condition:
    if(index>=9)goto check;
    digit=0;goto digit_condition;
digit_next:
    ++digit;
digit_condition:
    if(digit>=8)goto frame_next;
    amount=digits.step[digit];
    if(!amount)goto digit_next;
    base=digits.old[digit];pattern=base<<3;pattern+=base;pattern+=amount;
    sub_2D620((void *)(0xa7a90+digit*6),320,pattern);
    ++digits.step[digit];
    if(digits.step[digit]!=10)goto digit_next;
    ++digits.old[digit];if(digits.old[digit]==10)digits.old[digit]=0;
    goto digit_next;
check:
    if(!equal)goto again;
}
#endif
