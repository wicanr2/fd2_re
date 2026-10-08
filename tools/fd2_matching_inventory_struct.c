/* 0x1B8E7 的等價結構索引候選。原作者的結構宣告仍未知。
 * offset +0x0a 的八格 word、+0x18 的尾 flag、0x50 stride
 * 依既有原始 IDA 指令建模；不接正式重製路徑。
 */
#pragma pack(1)
typedef struct {
    unsigned char raw_prefix[10];
    union {
        unsigned short slots[8];
        struct {
            unsigned short first_seven[7];
            unsigned char raw_flag;
            unsigned char raw_item;
        } last;
    } inventory;
    unsigned char raw_suffix[54];
} Record;
extern Record *dword_53A45;
extern void *__cdecl memmove(void *, const void *, unsigned);

#ifdef RET_VOID
void __cdecl sub_1B8E7(int unit, int slot)
#else
void *__cdecl sub_1B8E7(int unit, int slot)
#endif
{
    Record *record = &dword_53A45[unit];
#ifdef RET_VOID
    memmove(&record->inventory.slots[slot], &record->inventory.slots[slot + 1], 2 * (7 - slot));
#else
    void *result = memmove(&record->inventory.slots[slot], &record->inventory.slots[slot + 1], 2 * (7 - slot));
#endif
    record->inventory.last.raw_flag = 0x80;
#ifndef RET_VOID
    return result;
#endif
}
