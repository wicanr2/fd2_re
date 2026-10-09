/* sub_35F6F完整35-byte C候選，固定IDA 9.4線性位址及原檔見匹配主收據。
 * 保留原始兩個byte寫入及一個byte讀取；宣告只供產碼導航。
 */
extern unsigned char *dword_53AD5, *dword_53A55;
extern int dword_53BEF;

#if defined(STATE_FORM_0) || defined(STATE_FORM_1) || defined(STATE_FORM_2) || defined(STATE_FORM_3) || defined(STATE_FORM_4) || defined(STATE_FORM_5)
void sub_35F6F(void)
{
    unsigned char value;
    unsigned char *record;
    ++dword_53AD5[16];
#ifdef STATE_FORM_0
    value = *(volatile unsigned char *)&dword_53BEF;
    ++value;
    record = dword_53A55;
    record[3] = value;
#endif
#ifdef STATE_FORM_1
    value = *(volatile unsigned char *)&dword_53BEF;
    ++value;
    dword_53A55[3] = value;
#endif
#ifdef STATE_FORM_2
    record = dword_53A55;
    value = *(volatile unsigned char *)&dword_53BEF;
    ++value;
    record[3] = value;
#endif
#ifdef STATE_FORM_3
    dword_53A55[3] = (unsigned char)((unsigned)dword_53BEF + 1);
#endif
#ifdef STATE_FORM_4
    dword_53A55[3] = *(volatile unsigned char *)&dword_53BEF + 1;
#endif
#ifdef STATE_FORM_5
    value = *(unsigned char *)&dword_53BEF;
    value += 1;
    dword_53A55[3] = value;
#endif
}
#endif
