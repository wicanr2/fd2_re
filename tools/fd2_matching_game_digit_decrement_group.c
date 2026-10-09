/* 相鄰數字更新的完整C編譯單元候選。
 * 固定IDA 9.4原始位址、雜湊與caller見匹配主收據，型別只作產碼導航。
 */
#if defined(DIGIT_DECREMENT_GROUP) || defined(DOWN_FORM_0) || defined(DOWN_FORM_1) || defined(DOWN_FORM_2) || defined(DOWN_FORM_3) || defined(DOWN_FORM_4) || defined(DOWN_FORM_5)
#define DIGIT_FORM_11
#include "DIGIT.C"
#undef DIGIT_FORM_11
extern const char a08d[], a08d_0[];

void sub_2D516(int difference)
{
    struct {
        unsigned char current[20];
        unsigned char step[20];
        unsigned char old[20];
    } digits;
    int index, digit, equal, amount, base, pattern, before, after;
    sprintf((char *)digits.old, a08d, dword_53BF3);
    index = 0;
    goto old_condition;
old_next:
    digits.old[index] -= 48;
    ++index;
old_condition:
    if (index < 8) goto old_next;
    dword_53BF3 -= difference;
    sprintf((char *)digits.current, a08d_0, dword_53BF3);
    index = 0;
    goto current_condition;
current_next:
    digits.current[index] -= 48;
    ++index;
current_condition:
    if (index < 8) goto current_next;
again:
    equal = 1;
    index = 0;
    goto compare_condition;
compare_zero:
    digits.step[index] = 0;
compare_next:
    ++index;
compare_condition:
    if (index >= 8) goto compare_done;
    before = digits.old[index];
    after = digits.current[index];
    if (before == after) goto compare_zero;
    digits.step[index] = 9;
    equal = 0;
    --digits.old[index];
    if ((int)digits.old[index] != 255) goto compare_next;
    digits.old[index] = 9;
    goto compare_next;
compare_done:
    if (equal) goto check;
    index = 0;
    goto frame_condition;
frame_next:
    sub_375B2(10);
    ++index;
frame_condition:
    if (index >= 9) goto check;
    digit = 0;
    goto digit_condition;
digit_next:
    ++digit;
digit_condition:
    if (digit >= 8) goto frame_next;
    amount = digits.step[digit];
    if (!amount) goto digit_next;
    base = digits.old[digit];
#if defined(DOWN_FORM_1)
    amount += 9 * base;
    --amount;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, amount);
#elif defined(DOWN_FORM_2)
    pattern = amount;
    pattern += 9 * base;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, pattern - 1);
#elif defined(DOWN_FORM_3)
    pattern = amount + 9 * base - 1;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, pattern);
#elif defined(DOWN_FORM_4)
    amount += 9 * base;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, amount - 1);
#elif defined(DOWN_FORM_5)
    pattern = amount;
    pattern += 9 * base - 1;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, pattern);
#else
    pattern = amount;
    pattern += 9 * base;
    --pattern;
    sub_2D620((void *)(0xa7a90 + digit * 6), 320, pattern);
#endif
    --digits.step[digit];
    goto digit_next;
check:
    if (!equal) goto again;
}
#endif
