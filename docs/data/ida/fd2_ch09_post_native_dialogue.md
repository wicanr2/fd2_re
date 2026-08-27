# 玩家第 10 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_010.bin`：4494 bytes；MD5 `57c3da8cddf0496473ac76dfe8b9f1be`；
  SHA-256 `17a2838e4abd4b1624953b415a8b1e38e35d2eb26c4a89a339e591dc33dba007`。
- 位址均為IDA Pro 9.4的`FD2.EXE`線性位址。handler、sparse record patch、
  DAC淡出／淡入、ACTING37、JOIN11／6與60／61 frontier主證據為
  [`fd2_ch09_post_ida.txt`](fd2_ch09_post_ida.txt)。

## 已證實

- `0x23727`使用`FDTXT_010` index4，共19句：
  `FFEE/8, FFED/50, FFEE/13, FFED/50, FFEE/0, FFED/51, FFED/50, FFEE/13,
  FFED/51, FFEE/13, FFED/51, FFEE/13, FFED/51, FFED/50, FFED/51, FFED/50,
  FFED/51, FFED/50, FFEC/51`。
- `0x23762`使用index5，共16句：
  `FFEE/8, FFED/50, FFEE/8, FFED/50, FFEE/8, FFED/50, FFEE/4, FFEE/0,
  FFED/50, FFEE/8, FFEE/6, FFED/50, FFEE/6, FFEE/4, FFEE/0, FFEE/4`。
  兩者依count-aligned manifest映射至`ch10.json`，全量為2個caller、35句。
- 原始順序為DAC delta0..63淡出、清`+5 bit7`、只寫明列的sparse欄位與視圖、
  DAC delta64..0淡入、index4、ACTING37、index5、sync、JOIN11、JOIN6、chapter10。

## 執行契約與限制

- 三十五句逐筆保存control、operand、頁／列與glyph token，並與固定raw equality相等。
- 正式輸入完成opening、逐字、嘴型、分頁與closing後才能ACTING、JOIN、同步及進
  `town_ch11`；禁止直接清除`g.dialog`。60／61兩個強推論E1 frontier都必須通過，
  並驗證兩筆JOIN的持續隊伍與存讀檔。
- sparse patch只可寫原始`+0/+1/+3/+5/+0x26`；缺caller context、唯一mapping、
  原始glyph、地圖底圖、頭像、DAC baseline、patch provenance或合法frontier時，
  在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊與town11原版輸入仍屬
  E2限制，不阻擋99%玩家可見相似門檻。
