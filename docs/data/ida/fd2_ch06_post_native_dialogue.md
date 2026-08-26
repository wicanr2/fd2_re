# 玩家第 7 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_007.bin`：2022 bytes；MD5 `5038c14a9dc2e71dcbca84859b0f0607`；
  SHA-256 `5d6556ce04c644c71e1fd8070167dffe34a9b20ebdfa8e291bc80167463720ca`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、event25 producer、
  34／44-slot frontier、版面與條件式 JOIN12 的主證據為
  [`fd2_ch06_post_event25_ida.txt`](fd2_ch06_post_event25_ida.txt)。

## 已證實

- `0x2337F` 使用 `FDTXT_007` index4，共8句，control／operand依序為
  `FFEF/12, FFEE/0, FFEF/12, FFEE/13, FFEF/12, FFEE/13, FFEF/12, FFEE/4`；
  映射至 `ch07.json` scene4 lines0..7。
- `0x233B2` 使用 index5，共4句，依序為
  `FFEE/4, FFEE/0, FFEE/10, FFEE/8`；映射至scene5 lines0..3。全量為
  2個唯一caller、12句。`0x233B2`雖出現在兩條互斥CFG分支，仍是同一原始
  caller／index／context，不得重複計數。
- state17==1、44 slots且slot43 active時，順序為`sync_party`→layout→index4→
  JOIN12→chapter7→`town_ch08`；state17 producer缺失或slot43 inactive時播放index5，
  不執行JOIN12。兩條分支都只播放其中一個caller。

## 執行契約與限制

- 十二句逐筆保存control、operand、頁／列與glyph token，並與固定raw equality相等。
- 正式輸入完成opening、逐字、嘴型、分頁與closing後才能JOIN、同步並進城鎮；禁止
  直接清除`g.dialog`。缺caller context、唯一mapping、原始glyph、地圖底圖、頭像、
  state17 producer或精確frontier時，在隊伍交易前失敗即關閉（fail-closed）。
- 對話binding不得把map6的state16／state17或slot43窄語意提升為跨關通用欄位。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊與town08原版輸入仍屬
  E2限制，不阻擋99%玩家可見相似門檻。
