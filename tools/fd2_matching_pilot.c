/* Matching decompilation 的三函式試驗，僅供編譯比對。
 * 原始位址與指令：work/matching-pilot-20261008/ida-pilot.json。
 * 主證據入口：docs/data/ida/fd2_matching_pilot_20261008.json。
 * 不含原版機器碼、內嵌組合語言或正式 remake 執行路徑。
 */

extern unsigned short word_627B8;
extern unsigned char *dword_53A45;
extern int dword_53BEB;
extern void __cdecl sub_14625(int x, int y);
extern void *__cdecl memmove(void *destination, const void *source, unsigned length);

#if defined(FD2_PILOT_RNG)
#if defined(FD2_RNG_PROMOTED)
unsigned sub_4E893(void)
#else
unsigned short sub_4E893(void)
#endif
{
    unsigned short value = word_627B8 + 0x9014;
    value = (value << 1) | (value >> 15);
    value = (value << 1) | (value >> 15);
    value = (value << 1) | (value >> 15);
    word_627B8 = value;
    return value;
}
#endif

#if defined(FD2_PILOT_OCCUPANCY)
void __cdecl sub_145CD(int mode)
{
    unsigned char *record = dword_53A45;
    int index;
    for (index = 0; index < dword_53BEB; ++index) {
        if (!(record[5] & 1) &&
            ((mode == 0 && record[6] != 0) ||
             (mode != 0 && record[6] == 0))) {
            sub_14625(record[0], record[1]);
        }
        record += 80;
    }
}
#endif

#if defined(FD2_PILOT_INVENTORY)
void *__cdecl sub_1B8E7(int unit, int slot)
{
    unsigned char *record = dword_53A45 + 80 * unit;
    void *result = memmove(record + 2 * slot + 10,
                           record + 2 * slot + 12,
                           2 * (7 - slot));
    record[24] = 0x80;
    return result;
}
#endif
