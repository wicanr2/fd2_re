/* 事件呼叫與目標列表的純 C 候選。
 * 原始函式名及呼叫目標保留；型別與變數名只供導航。
 */
extern volatile unsigned dword_53A45;
extern int dword_53EC4;
extern void sub_1C4CC(int, int, int, const unsigned char *);
extern void sub_1C2DA(int, int, int, const unsigned char *);
extern void sub_1CAC7(int, int, int, const unsigned char *);
extern int sub_1C75E(int, int);
extern int sub_1C916(int, int);
extern void sub_1E0DB(int, int, int);
extern void sub_1E1DC(int);
extern void sub_11CAC(int);
extern void sub_1DF58(void);
extern void sub_1B750(int);
extern void sub_1B8E7(int, int);
extern void sub_1CA89(int, int);

#ifdef F21082
void sub_21082(int unit, int value, int offset, int slot, int count,
               const unsigned char *targets, int mode)
{
    unsigned address;
    sub_1C4CC(unit, mode, count, targets);
    sub_1C2DA(unit, mode, count, targets);
    address = targets[0] * 80 + dword_53A45;
    address += offset;
    *(unsigned short *)address += (unsigned short)value;
    sub_1E0DB(value, 94, targets[0]);
    sub_11CAC(0);
    sub_1DF58();
    sub_1B750(unit);
    sub_1B8E7(unit, slot);
}
#endif

#ifdef EVENT_PAIR
void sub_2111A(int unit, int count, const unsigned char *targets, int mode)
{
    int index;
    const unsigned char *target;
    int value;
    sub_1C4CC(unit, mode, count, targets);
    sub_1CAC7(unit, mode, count, targets);
    index = 0;
    goto condition;
next:
    ++index;
condition:
    if (index >= count) goto done;
    value = sub_1C75E(*(target = targets + index), mode);
    if (value) goto show;
    sub_1E1DC(*target);
    goto next;
show:
    sub_1E0DB(value, 94, *target);
    goto next;
done:
    sub_11CAC(0);
    sub_1DF58();
}

void sub_211A4(int unit, int count, const unsigned char *targets, int value)
{
    int index;
    int result;
    sub_1C4CC(unit, 13, count, targets);
    sub_1C2DA(unit, 13, count, targets);
    index = 0;
    goto condition;
next:
    result = sub_1C916(targets[index], value);
    sub_1E0DB(result, 105, targets[index]);
    ++index;
condition:
    if (index < count) goto next;
    sub_11CAC(0);
    sub_1DF58();
}
#endif

#ifdef F214AD
void sub_214AD(int unit, int count, const unsigned char *targets)
{
    int value;
    dword_53EC4 = 0;
    sub_1C4CC(unit, 9, count, targets);
    sub_1CA89(unit, 9);
    value = sub_1C75E(targets[0], 9);
    if (value) goto show;
    sub_1E1DC(targets[0]);
    goto done;
show:
    sub_1E0DB(value, 94, targets[0]);
done:
    sub_11CAC(0);
    sub_1DF58();
}
#endif
