/* 0x1B8E7 的型別與運算式產碼試驗，僅供匹配工具使用。
 * 原始 bytes/caller：docs/data/ida/fd2_matching_pilot_original_20261008.json。
 * ADDR_INT 與 RET_INT 是候選來源形式，不宣稱原作者使用此型別。
 */
extern unsigned char *dword_53A45;
#ifdef RET_VOID
extern void *__cdecl memmove(void *, const void *, unsigned);
void __cdecl sub_1B8E7(int unit, int slot)
#elif defined(RET_INT)
extern int __cdecl memmove(void *, const void *, unsigned);
int __cdecl sub_1B8E7(int unit, int slot)
#else
extern void *__cdecl memmove(void *, const void *, unsigned);
void *__cdecl sub_1B8E7(int unit, int slot)
#endif
{
#ifdef MUL_SHIFTS
    unsigned offset = (((unsigned)unit << 2) + unit) << 4;
#else
    unsigned offset = 80 * unit;
#endif
#ifdef ADDR_INT
    unsigned record = (unsigned)dword_53A45 + offset;
#ifdef RET_VOID
    memmove((void *)(record + 2 * slot + 10),
            (void *)(record + 2 * slot + 12), 2 * (7 - slot));
#elif defined(RET_INT)
    int result = memmove((void *)(record + 2 * slot + 10),
                        (void *)(record + 2 * slot + 12), 2 * (7 - slot));
#else
    void *result = memmove((void *)(record + 2 * slot + 10),
                          (void *)(record + 2 * slot + 12), 2 * (7 - slot));
#endif
    *(unsigned char *)(record + 24) = 0x80;
#else
    unsigned char *record = dword_53A45 + offset;
#ifdef RET_VOID
    memmove(record + 2 * slot + 10,
            record + 2 * slot + 12, 2 * (7 - slot));
#elif defined(RET_INT)
    int result = memmove(record + 2 * slot + 10,
                        record + 2 * slot + 12, 2 * (7 - slot));
#else
    void *result = memmove(record + 2 * slot + 10,
                          record + 2 * slot + 12, 2 * (7 - slot));
#endif
    record[24] = 0x80;
#endif
#ifndef RET_VOID
    return result;
#endif
}
