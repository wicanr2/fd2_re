/* 查詢、捲動與座標轉發的完整C編譯單元候選。
 * 固定IDA 9.4原始位址、雜湊及caller見匹配主收據。
 * 沿用既有C來源，不推定原作者的函式分檔或宣告。
 */
#ifdef MAP_QUERY_EXTENDED
#define MAP_QUERY_GROUP
#include "MAP.C"
#undef MAP_QUERY_GROUP

void sub_12D7B(int unit)
{
    int offset = unit, x, y;
    unsigned char *record;
    offset *= 80;
    record = dword_53A45;
    record += offset;
    x = record[0];
    y = record[1];
    sub_12CEA(x, y);
}
#endif
