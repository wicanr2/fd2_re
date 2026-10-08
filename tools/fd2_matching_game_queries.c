/* 查表、方向及反向遍歷的純 C 產碼候選。
 * 原始名稱、位移與呼叫目標依 IDA 固定輸入；作者宣告仍未知。
 * 只還原直接指令的資料流，不將變數名當成玩法證據。
 */
extern unsigned char *dword_53A45;
extern int dword_53C03;
extern unsigned char byte_52363[];
extern const unsigned char *__cdecl sub_4E53E(int index);
extern const unsigned char *__cdecl sub_4E56C(int index);
extern int abs(int value);
extern void sub_18C6D(int position, int stride, int unit);
extern unsigned char sub_12E38(int x, int y, void *output);
extern int sub_1F183(int unit);
extern void sub_1E739(int position, int y, int width);
extern void *memset(void *destination, int value, unsigned count);

#ifdef RATIO_WORDS
void sub_1E7F6(int x, int y, int unit, const int *box)
{
    unsigned char *record = dword_53A45 + 80 * unit;
    int current = *(unsigned short *)(record + 0x40);
    int maximum = *(unsigned short *)(record + 0x42);
    if (current > 0) {
        sub_1E739((box[1] + 6) * y + x + box[0] + 7,
                  y, current * 69 / maximum + 1);
    }
}
#endif

#ifdef FILL_SQUARE
void sub_1F6EF(int x, int y, int value, int size)
{
    int fill = value;
    int limit = size;
    unsigned char *destination = (unsigned char *)(y * 320 + 0xa0000);
    int row;
    destination += x;
    row = 0;
    goto condition;
next:
    memset(destination, fill, limit - 1);
    destination += 320;
    ++row;
condition:
    if (row < limit - 1) goto next;
}
#endif

#ifdef MEMBER_SIX
int sub_1C1C3(int unit, int item)
{
    int key = dword_53A45[80 * unit + 0x20];
    const unsigned char *entries = sub_4E53E(key);
    const unsigned char *data = sub_4E56C(item);
    int value = data[0];
    int index = 0;
    int current;
    goto condition;
next:
    ++index;
condition:
    if (index >= 6) goto not_found;
    current = entries[index];
    if (value != current) goto next;
    return 1;
not_found:
    return 0;
}
#endif

#ifdef FACING_BYTES
void sub_1F04A(int first, int second)
{
    unsigned char *a = 80 * first + dword_53A45;
    unsigned char *b = 80 * second + dword_53A45;
    int dx, dy;
    dx = abs((int)a[0] - (int)b[0]);
    dy = abs((int)a[1] - (int)b[1]);
    if (dx > dy) {
        if ((int)a[0] > (int)b[0]) {
            a[3] = 1;
            return;
        }
        a[3] = 3;
        return;
    }
    if ((int)a[1] > (int)b[1]) {
        a[3] = 2;
        return;
    }
    a[3] = 0;
}
#endif

#ifdef BOX_POSITION
void sub_2A289(int offset, int unit)
{
    int value = dword_53A45[80 * unit + 6];
    int position;
    if (!value) {
        position = 0xc080;
    } else {
        position = 0x5ab;
    }
    if (dword_53C03 == 24 && unit == 17) position = 0xc080;
    sub_18C6D(position + offset, 320, unit);
}
#endif

typedef struct {
    short word_00, word_02;
    unsigned char bytes_04[4];
} RawInfo;
typedef char InfoSizeMustBe8[sizeof(RawInfo) == 8 ? 1 : -1];

#ifdef REVERSE_INFO
int sub_2B5E1(int count, const unsigned char *indices)
{
    const unsigned char *list = indices;
    int value = byte_52363[dword_53C03];
    int index = count;
    unsigned char *record;
    RawInfo info;
next:
    --index;
    if (index < 0) goto finished;
    record = 80 * list[index] + dword_53A45;
    sub_12E38(record[0], record[1], &info);
    if (sub_1F183(list[index]) && value) goto next;
    value = (unsigned short)info.word_02;
    goto next;
finished:
    return value;
}
#endif
