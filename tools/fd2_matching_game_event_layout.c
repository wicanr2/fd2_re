/* 完整事件來源布局候選。只恢復直接指令的資料流。
 * sub_212B9 是合成C導航符號；IDA沒有原始函式名或owner。
 * 其原始歸屬與作者宣告維持未知，不增加原始函式數量。
 */
extern int dword_53EC4;
extern void sub_1C4CC(int, int, int, const unsigned char *);
extern void sub_1C2DA(int, int, int, const unsigned char *);
extern void sub_1CAC7(int, int, int, const unsigned char *);
extern void sub_1CD17(int, int, int, const unsigned char *);
extern void sub_1CA89(int, int);
extern int sub_1C75E(int, int);
extern int sub_1C916(int, int);
extern void sub_1E0DB(int, int, int);
extern void sub_1E1DC(int);
extern void sub_11CAC(int);
extern void sub_1DF58(void);

#ifdef EVENT_FULL
void sub_21227(int, int, int, const unsigned char *);
void sub_213B7(int, int, int, const unsigned char *);

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

void sub_21206(int unit, int count, const unsigned char *targets)
{
    sub_21227(unit, 0, count, targets);
}

#define EFFECT_FUNCTION(NAME, EFFECT) \
void NAME(int unit, int mode, int count, const unsigned char *targets) \
{ \
    int index; \
    const unsigned char *target; \
    int value; \
    dword_53EC4 = 0; \
    sub_1C4CC(unit, mode, count, targets); \
    EFFECT(unit, mode, count, targets); \
    sub_1CA89(unit, mode); \
    index = 0; \
    goto condition; \
next: \
    ++index; \
condition: \
    if (index >= count) goto done; \
    value = sub_1C75E(*(target = targets + index), mode); \
    if (value) goto show; \
    sub_1E1DC(*target); \
    goto next; \
show: \
    sub_1E0DB(value, 94, *target); \
    goto next; \
done: \
    sub_11CAC(0); \
    sub_1DF58(); \
}

EFFECT_FUNCTION(sub_21227, sub_1CD17)
EFFECT_FUNCTION(sub_212B9, sub_1CD17)

#define EFFECT_WRAPPER(NAME, TARGET, MODE) \
void NAME(int unit, int count, const unsigned char *targets) \
{ TARGET(unit, MODE, count, targets); }

EFFECT_WRAPPER(sub_2134B, sub_21227, 1)
EFFECT_WRAPPER(sub_21364, sub_21227, 2)
EFFECT_WRAPPER(sub_2137D, sub_21227, 3)
EFFECT_WRAPPER(sub_21396, sub_213B7, 4)
EFFECT_FUNCTION(sub_213B7, sub_1CAC7)
EFFECT_WRAPPER(sub_21449, sub_213B7, 5)
EFFECT_WRAPPER(sub_21462, sub_213B7, 6)
EFFECT_WRAPPER(sub_2147B, sub_213B7, 7)
EFFECT_WRAPPER(sub_21494, sub_21227, 8)

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
