/* 80-byte 記錄的產碼候選。
 * 位移與存取寬度取自 IDA 原始指令；作者的型別宣告仍為未知。
 * 欄位名稱只保存原始位移，不附加玩法語意。
 */
#pragma pack(1)
typedef struct {
    unsigned char byte_00;
    unsigned char byte_01;
} RawCell;

typedef struct {
    unsigned char bytes_00[5];
    unsigned char byte_05;
    unsigned char bytes_06[4];
    RawCell cells_0A[8];
    unsigned char bytes_1A[26];
    unsigned char byte_34;
    unsigned char bytes_35[11];
    unsigned short word_40;
    unsigned short word_42;
    unsigned short word_44;
    unsigned short word_46;
    unsigned char bytes_48[8];
} RawRecord;
#pragma pack()
typedef char RecordSizeMustBe80[sizeof(RawRecord) == 80 ? 1 : -1];

extern unsigned char *dword_53A45;
extern unsigned char *dword_53BF7;
extern int dword_53BFB;

#ifdef SET_BIT7
void sub_13512(int unit)
{
    ((RawRecord *)dword_53A45)[unit].byte_05 |= 0x80;
}
#endif

#ifdef SLOT_BYTE
unsigned sub_1B722(int unit, int slot)
{
    return ((RawRecord *)dword_53A45)[unit].cells_0A[slot].byte_01;
}
#endif

#ifdef COPY_WORDS
void sub_25089(void)
{
    unsigned char unit;
    for (unit = 0; unit < dword_53BFB; ++unit) {
        RawRecord *record = (RawRecord *)dword_53BF7 + unit;
        record->byte_05 = 0;
        record->word_40 = record->word_42;
        record->word_44 = record->word_46;
    }
}
#endif
