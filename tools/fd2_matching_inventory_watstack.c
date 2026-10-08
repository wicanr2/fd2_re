/* 0x1B8E7：結構索引配合已觀察到的 EBX 保存契約。
 * 原作者宣告仍未知；此檔只作 C 產碼候選，原始 bytes 在既有 IDA 匯出。
 */
#pragma pack(1)
typedef struct {
    unsigned char prefix[10];
    union {
        unsigned short slots[8];
        struct {
            unsigned short first[7];
            unsigned char flag;
            unsigned char item;
        } tail;
    } inventory;
    unsigned char suffix[54];
} Record;
extern Record *dword_53A45;
extern void *memmove(void *, const void *, unsigned);

#ifdef RET_VOID
void sub_1B8E7(int unit, int slot)
#else
void *sub_1B8E7(int unit, int slot)
#endif
{
    Record *record = &dword_53A45[unit];
#ifdef RET_VOID
    memmove(&record->inventory.slots[slot], &record->inventory.slots[slot + 1], 2 * (7 - slot));
#else
    void *result = memmove(&record->inventory.slots[slot], &record->inventory.slots[slot + 1], 2 * (7 - slot));
#endif
    record->inventory.tail.flag = 0x80;
#ifndef RET_VOID
    return result;
#endif
}
