# 玩家第 20 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5
  `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_020.bin`：3836 bytes；MD5 `fcbfb5783bc0b00cbe2ebff08986408c`；
  SHA-256 `64711c85deab36737fea9c12fb7b1280e6dfb3a596865cc3e133c18b61fca1c8`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、分支、slot frontier、
  JOIN 與直接記錄寫入的主證據是
  [`fd2_ch19_post_ida.txt`](fd2_ch19_post_ida.txt)；本檔只附加逐句版面及正式
  消費契約，不取代原始定位。

## 已證實：六個 caller 與 29 句

| caller | `FDTXT_020` index | 句數 | `ch20.json` 目標 |
|---|---:|---:|---|
| `0x23fa2` | 11 | 10 | scene1 lines9..18 |
| `0x23fdd` | 12 | 3 | scene2 lines0..2 |
| `0x2403e` | 14 | 4 | scene3 lines0..3 |
| `0x24079` | 15 | 3 | scene3 lines4..6 |
| `0x240b4` | 16 | 7 | scene3 lines7..13 |
| `0x240e5` | 13 | 2 | scene2 lines3..4 |

- 全量合計29句；`count-aligned.json` 對每個 index 均只有一個目標，句數與原始
  utterance count 相等。
- index11／12／13位於共同路徑，所以 round15與round16都會播放15句。
- index14／15／16只位於 `native_round_gt(15)` 的 else arm；round15播放全部29句，
  round16略過這14句，不可為了測試全量而強迫執行互斥分支。

## 已證實：交易與對話順序

- 共同前段：index11 → ACTING59 → index12 → JOIN25 → `sync_party`。
- round <=15 分支：spawn group1 → ACTING60 → index14 → ACTING61 → index15 →
  ACTING62 → index16 → JOIN28。
- 合流後才播放 index13，再執行 chapter20並由 campaign 進 `town_ch21`。
- `sync_party` 位於 JOIN25之後、可選JOIN28之前；正式回歸必須保留這個看似不對稱
  但有直接指令支持的順序，不得移到所有對話尾端。

## 執行契約與限制

- 每句保存原始 control、operand、`FFFE`／`FFFD`、頁／列與 glyph token；固定 raw
  equality 必須逐筆相等。
- 缺 caller context、唯一 mapping、原始 glyph、對話框／頭像／地圖底圖，或句數
  不等時，在任何 JOIN／同步／章節交易前失敗即關閉（fail-closed）。
- 正式 round15／16 回歸使用與鍵盤共用的具型別輸入完成 opening、逐字、嘴型、
  分頁與 closing；禁止直接清除 `g.dialog`。
- 完成後最高為 `RUNTIME-E1`。未修改 DOSBox 同狀態、精確音訊及 town21 原版輸入
  仍是 E2 限制，不阻擋 99% 玩家可見相似門檻。
