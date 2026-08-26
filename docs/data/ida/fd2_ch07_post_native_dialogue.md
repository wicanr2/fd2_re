# 玩家第 8 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_008.bin`：1896 bytes；MD5 `b4152116d5778d70804ef0ea0e6f255c`；
  SHA-256 `26fc6f2f6692d84bdce790525989c55bd4655cd9549d8a1a8bdc28dec52e2c02`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、合法 frontiers、版面、
  演出、全黑與尾段主證據為 [`fd2_ch07_post_ida.txt`](fd2_ch07_post_ida.txt)。

## 已證實

- `0x23525` 使用 `FDTXT_008` index3，共5句，control／operand 依序為
  `FFEE/8, FFEF/5, FFEE/8, FFEF/5, FFEE/8`；映射至 `ch08.json` scene3 lines0..4。
- `0x23560` 使用 index4，共3句，依序為 `FFEE/0, FFEE/4, FFEE/0`；映射至
  同一 scene lines5..7。全量為2個caller、8句。
- 原始順序為 layout → index3 → ACTING33 → index4 → ACTING34 → 全DAC黑與
  framebuffer清除 → JOIN5 → `sync_party` → chapter8 → `town_ch09`。
- 合法入口 slot counts 為29、31、33、35、37、39、41；對話資料不得改動 event27
  形成的奇數 frontier，也不得假造 groups1／8／9／10。

## 執行契約與限制

- 八句逐筆保存 control、operand、頁／列、`FFFE`／`FFFD`與 glyph token，並與固定
  raw equality 相等。
- 正式輸入完成 opening、逐字、嘴型、分頁與 closing 後才能繼續 ACTING／全黑／
  JOIN／同步；禁止直接清除 `g.dialog`。
- 缺 caller context、唯一 mapping、原始 glyph、地圖底圖、頭像或任何合法 frontier
  provenance 時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確DAC時序及town09原版輸入
  仍屬E2限制，不阻擋99%玩家可見相似門檻。
