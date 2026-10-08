/* 原始 IDA 位址的最小 C 切片。全檔分類仍依既有函式清冊。
 * 原始名稱只作定位；以下只命名已直接看見的 raw 存取，不推定玩法語意。
 */
extern unsigned char *dword_53AD5;
extern unsigned long off_52758;
extern unsigned long off_5275C;
extern unsigned long dword_52754;

#if defined(H13)
void sub_35EBE(void) { dword_53AD5[0x13] = 1; }
#elif defined(H14)
void sub_35ED2(void) { dword_53AD5[0x14] = 1; }
#elif defined(S58)
unsigned long sub_3615E(unsigned long value)
{
    unsigned long previous = off_52758;
    off_52758 = value;
    return previous;
}
#elif defined(S5C)
unsigned long sub_3616E(unsigned long value)
{
    unsigned long previous = off_5275C;
    off_5275C = value;
    return previous;
}
#elif defined(G54)
unsigned long sub_368FA(void) { return dword_52754; }
#endif
