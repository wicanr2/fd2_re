/* 原版 +0x1B8E7 呼叫前後使用 EBX 的 ABI 候選。
 * IDA 的 __cdecl 標記只作導覽；機器碼及 callee 保存契約才是證據。
 */
extern unsigned char *dword_53A45;
#ifdef CALLEE_CDECL
extern void *__cdecl memmove(void *, const void *, unsigned);
#else
extern void *memmove(void *, const void *, unsigned);
#endif

#ifdef OWNER_CDECL
void *__cdecl sub_1B8E7(int unit, int slot)
#else
void *sub_1B8E7(int unit, int slot)
#endif
{
    unsigned char *record = dword_53A45 + 80 * unit;
    void *result = memmove(record + 2 * slot + 10,
                          record + 2 * slot + 12, 2 * (7 - slot));
    record[24] = 0x80;
    return result;
}
