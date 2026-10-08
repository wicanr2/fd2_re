/* sub_190AC完整1215-byte C候選與徑向函式的返回段來源。
 * 固定輸入、IDA Pro 9.4線性位址及原始名稱見匹配主收據。
 * 玩法語意沿fd2_treasure_input_20260907.json，不重開已閉合規則。
 * 宣告及框架名稱只用於產碼，未匹配時不增加覆蓋。
 */
typedef struct { short word_00, word_02; unsigned char byte_04, bytes_05[3]; } RawTreasureInfo;
typedef struct { RawTreasureInfo info; int old_item, saved_item, flags; } RawTreasureFrame;
typedef char RawTreasureFrameMustBe20[sizeof(RawTreasureFrame) == 20 ? 1 : -1];
typedef struct { unsigned char data[80]; } RawTreasureRecord;
extern unsigned char * volatile dword_53A45;
extern unsigned char *dword_53A55, *dword_53AD5;
extern int dword_53AB1, dword_53AB5, dword_53A7D, dword_53C57, dword_53EEC;
extern int dword_53AD9, dword_53ADD, dword_53AE1, dword_53BF3;
extern unsigned char sub_12E38(int, int, void *);
extern void sub_4E031(void);
extern void sub_1956B(int);
extern void sub_15F84(int, int, void *, int, int, int, int, int, int);
extern void sub_16559(int);
extern int sub_19953(void);
extern void sub_197E5(void);
extern void sub_25B45(int, int, int);
extern void sub_375B2(int);
extern int sub_1BB8C(int, int);
extern void sub_16C57(int);
extern void sub_196CB(void);
extern void sub_12263(void);
extern int sub_1B932(int, int);
extern int sub_1B722(int, int);
extern void sub_1B8E7(int, int);
typedef void (*RawUnitFunction)(int);
extern RawUnitFunction funcs_1199C[];
#define MESSAGE(INDEX, SURFACE) sub_15F84(dword_53A7D, INDEX, (void *)SURFACE, 320, 205, 76, 74, 19, 1)

#if defined(F190AC) || defined(RADIAL_GROUP)
void sub_190AC(int unit)
{
    RawTreasureFrame frame;
    int grid;
    int item;
    int kind;
    RawTreasureRecord *record;
    sub_12E38(dword_53AB1, dword_53AB5, &frame.info);
    grid = frame.info.word_02;
    frame.flags = frame.info.byte_04;
    if ((unsigned char)(frame.flags & 0x60) == 0) goto done;
    kind = *(volatile unsigned char *)(dword_53AD5 + grid);
    if (kind != 0) goto done;
    record = (RawTreasureRecord *)dword_53A45 + unit;
    sub_4E031();
    sub_1956B(record->data[7]);
    if ((unsigned char)(frame.flags & 0x20)) MESSAGE(421, 0xa9f23);
    else MESSAGE(428, 0xa9f23);
    sub_16559(0);
    item = sub_19953();
    sub_197E5();
    if (item != 1 || dword_53C57 != 0) goto cancel;
    sub_25B45(dword_53EEC, 12, item);
    sub_375B2(300);
    item = *(unsigned short *)(dword_53A55 + grid * 3 + 0x54);
    kind = *(volatile unsigned char *)(dword_53A55 + grid * 3 + 0x53);
    if (kind != 0) goto other;
    dword_53AD9 = item + 181;
    if ((unsigned char)(frame.flags & 0x20)) MESSAGE(422, 0xab6e3);
    else MESSAGE(429, 0xab6e3);
    if (sub_1BB8C(unit, item) == -1) goto full;
    dword_53AD5[grid] = 1;
    sub_16559(0);
    sub_16C57(0);
    sub_196CB();
finish:
    sub_12263();
    goto done;
full:
    sub_16559(0);
    sub_16C57(0);
    sub_196CB();
    sub_375B2(100);
    sub_1956B(record->data[7]);
    MESSAGE(423, 0xa9f23);
    sub_16559(0);
    frame.saved_item = sub_19953();
    sub_197E5();
    if (frame.saved_item != 1 || dword_53C57 != 0) goto declined;
    sub_196CB();
    if (!sub_1B932(unit, 0)) goto unavailable;
    frame.saved_item = sub_1B722(unit, dword_53C57);
    frame.old_item = frame.saved_item;
    sub_1B8E7(unit, dword_53C57);
    sub_1BB8C(unit, item);
    *(unsigned short *)(dword_53A55 + grid * 3 + 0x54) = frame.old_item;
    sub_375B2(100);
    sub_1956B(record->data[7]);
    dword_53ADD = frame.saved_item + 181;
    MESSAGE(425, 0xa9f23);
    sub_16559(0);
    sub_16C57(0);
    goto redraw_done;
unavailable:
    sub_375B2(100);
    sub_1956B(record->data[7]);
    MESSAGE(424, 0xa9f23);
    sub_375B2(200);
    goto redraw_done;
declined:
    MESSAGE(424, 0xab6e3);
    goto redraw_done;
other:
    if (kind != 1) goto dispatch;
    if (item != 0) goto amount;
    if ((unsigned char)(frame.flags & 0x20)) MESSAGE(427, 0xab6e3);
    else MESSAGE(431, 0xab6e3);
    goto amount_shown;
amount:
    dword_53AE1 = item;
    if ((unsigned char)(frame.flags & 0x20)) MESSAGE(426, 0xab6e3);
    else MESSAGE(430, 0xab6e3);
amount_shown:
    sub_16559(0);
    sub_16C57(0);
    sub_196CB();
    dword_53BF3 += dword_53AE1;
    dword_53AD5[grid] = 1;
    goto finish;
dispatch:
    sub_375B2(200);
    sub_196CB();
    funcs_1199C[item](unit);
done:
    return;
cancel:
    sub_375B2(100);
    MESSAGE(412, 0xab6e3);
    sub_375B2(200);
redraw_done:
    sub_196CB();
    goto done;
}
#endif

#ifdef RADIAL_GROUP
#define F219AD
#include "EFFECT.C"
#endif
