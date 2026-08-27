# 玩家第 6 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_006.bin`：3208 bytes；MD5 `c55deffd0244e3b48e69711c8b07cfe5`；
  SHA-256 `493d9e3f04d582f1a516d22444a0a41fb39cff3dcfc9a7125ef0c414c050a8c3`。
- 位址均為IDA Pro 9.4的`FD2.EXE`線性位址。handler、JOIN13、group3、PAN、
  ACTING27、共享尾段與chapter transition主證據為
  [`../fd2_ch05_post_dispatch_ida.txt`](../fd2_ch05_post_dispatch_ida.txt)。

## 已證實

- 唯一caller `0x231E5`使用`FDTXT_006` index6，共19句：
  `FFEF/13, FFEE/8, FFEF/13, FFEE/10, FFEF/13, FFEE/10, FFEF/13, FFEE/4,
  FFEF/13, FFEE/0, FFEE/10, FFEE/0, FFEF/13, FFEE/1, FFEE/8, FFEE/4,
  FFEE/8, FFEE/0, FFEF/13`；映射至`ch06.json` scene6 lines0..18。
- 原始順序為JOIN13、group3 append、PAN raw `(5,14)`、ACTING27、index6的19句、
  `sync_party`、chapter6。`0x232E3→0x231DF`是共享尾段跳轉，不是遺失return。
- 正式runtime入口為40 slots，group3後為41 slots；本切片沒有條件分支或未知call。

## 執行契約與限制

- 十九句逐筆保存control、operand、頁／列與glyph token，並與固定raw equality相等。
- 正式輸入完成opening、逐字、嘴型、分頁與closing後才能同步並進`town_ch07`；
  禁止直接清除`g.dialog`。JOIN13必須先於group3，且完整persistent record須通過存讀檔。
- 缺caller context、唯一mapping、原始glyph、地圖底圖、頭像、group3 provenance或
  40-slot frontier時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊與town07原版輸入仍屬
  E2限制，不阻擋99%玩家可見相似門檻。
