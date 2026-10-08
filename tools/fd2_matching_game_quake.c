/* sub_21548 的完整 C 產碼候選，尚未逐位元組匹配。
 * 依據：fd2_matching_full_20261008.json 綁定的 IDA Pro 9.4 匯出，
 * IDA 線性位址 0x21548..0x2185f；原檔版本見該收據 input。
 * Frame只保存原始堆疊位移；型別與變數名不推定為作者宣告。
 * 遊戲語意：強推論；候選的行為等價性尚未獨立執行驗證。
 */
typedef struct { int values[3]; } RawTriplet;
typedef struct { unsigned short word_00, word_02; unsigned char bytes_04[4]; } RawInfo;
typedef struct {
    unsigned char *buffers[4];
    RawTriplet phase;
    RawTriplet x;
    RawTriplet y;
    RawInfo info;
    unsigned char *saved_shape;
} RawFrame;
typedef char FrameSizeMustBe64[sizeof(RawFrame) == 64 ? 1 : -1];
extern const RawTriplet unk_52096, unk_520A2, unk_520AE;
extern unsigned char *dword_53A5D, *dword_53A49;
extern unsigned char * volatile dword_53A45;
extern int dword_53EC4, dword_53AC1, dword_53AC5, dword_53BEB;
extern int dword_53AA9, dword_53AAD, dword_51A87, dword_51A8B;
extern int dword_53B13;
extern const char aOutOfMemoryAtE[];
extern void *malloc(unsigned);
extern void __cdecl free(void *);
extern int __cdecl printf(const char *, ...);
extern void exit(int);
/* Watcom 的不返回宣告，未提供組語、暫存器配置或機器碼。 */
#pragma aux exit aborts;
extern unsigned char *sub_1399C(void);
extern void sub_1CA89(int, int);
extern unsigned char sub_12E38(int, int, void *);
extern void sub_1F558(int, int, int, void *);
extern void sub_11EB0(void *, int, const void *, int, int, int);
extern void sub_127E0(int);
extern void sub_375B2(int);
extern void sub_25A96(int, int, int);
extern void sub_11CAC(int);
extern int sub_1C75E(int, int);
extern void sub_1E0DB(int, int, int);
extern void sub_1E1DC(int);
extern void sub_1DF58(void);

#ifdef F21548
void sub_21548(int unit, int mode, int count, const unsigned char *targets)
{
    RawFrame frame;
    unsigned char *screen;
    unsigned char **table;
    int column, index, value;
    const unsigned char *target;
    frame.x = unk_52096;
    frame.y = unk_520A2;
    frame.phase = unk_520AE;
    frame.saved_shape = dword_53A5D;
    dword_53A5D = sub_1399C();
    dword_53EC4 = 0;
    sub_1CA89(unit, mode);
    screen = malloc(64000);
    if (!screen) goto failed;
    frame.buffers[1] = malloc(153216);
    if (!frame.buffers[1]) goto failed;
    frame.buffers[2] = malloc(153216);
    if (!frame.buffers[2]) goto failed;
    goto allocated;
failed:
    printf(aOutOfMemoryAtE);
    exit(0);
allocated:
    frame.buffers[0] = dword_53A49;
    frame.buffers[3] = frame.buffers[1];
    table = malloc(16384);
    index = 0;
    goto outer_condition;
outer_next:
    ++index;
outer_condition:
    if (index >= dword_53AC5) goto phases;
    column = 0;
    goto inner_condition;
inner_next:
    sub_12E38(column, index, &frame.info);
    table[index * 64 + column] = dword_53A5D + 6 + frame.info.word_00 * 576;
    ++column;
inner_condition:
    if (column < dword_53AC1) goto inner_next;
    goto outer_next;
phases:
    column = 0;
    goto phase_condition;
phase_next:
    ++column;
phase_condition:
    if (column >= 3) goto presents;
    dword_53A49 = screen;
    sub_1F558(dword_53AA9 * 3072 + dword_51A87 * 1536 + frame.x.values[column],
              dword_53AAD * 3072 + dword_51A8B * 1536 + frame.y.values[column],
              frame.phase.values[column], table);
    sub_11EB0(frame.buffers[column] + 0x8088, 456,
              dword_53A49 + 0x504, 320, 312, 192);
    dword_53A49 = frame.buffers[column];
    index = 0;
    goto unit_condition;
unit_next:
    ++index;
unit_condition:
    if (index >= dword_53BEB) goto phase_next;
    if ((unsigned char)((index * 80 + dword_53A45)[5] & 1)) goto unit_next;
    sub_127E0(index);
    goto unit_next;
presents:
    index = 0;
    goto present_condition;
present_next:
    ++index;
present_condition:
    if (index >= 60) goto release;
    if (index < 43 && index % 6 == 0) sub_25A96(dword_53B13, 13, 1);
    sub_11EB0((void *)0xa0504, 320, frame.buffers[index % 4] + 0x8088, 456, 312, 192);
    sub_375B2(10);
    goto present_next;
release:
    free(screen);
    free(frame.buffers[1]);
    free(frame.buffers[2]);
    dword_53A49 = frame.buffers[0];
    free(dword_53A5D);
    dword_53A5D = frame.saved_shape;
    sub_11CAC(0);
    index = 0;
    goto target_condition;
target_next:
    ++index;
target_condition:
    if (index >= count) goto done;
    value = sub_1C75E(*(target = targets + index), mode);
    if (value) goto show;
    sub_1E1DC(*target);
    goto target_next;
show:
    sub_1E0DB(value, 94, *target);
    goto target_next;
done:
    sub_11CAC(0);
    sub_1DF58();
}
#endif
