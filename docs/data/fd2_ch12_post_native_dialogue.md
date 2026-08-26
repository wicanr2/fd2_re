# 玩家第 13 戰戰後原生對話綁定證據

## 參考版本

- `FD2.EXE`：357074 bytes；MD5
  `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_013.bin`：2360 bytes；MD5 `f6a60a04661f2308154b259a3fd4830a`；
  SHA-256 `1fde41ba41b05ae8d3ea1d9c5ad9f47382d9ef6029ff5fe840f5f4eefb22206f`。
  它是固定版 `FDTXT.DAT` 的解包資源；母檔雜湊仍以
  [`fd2-reference-files.json`](fd2-reference-files.json)為準。
- handler 位址是 IDA Pro 9.4 的 `FD2.EXE` 線性位址。dispatch、interior entry、
  caller 與共享尾段的主證據為
  [`fd2_ch12_post_dispatch_ida.txt`](fd2_ch12_post_dispatch_ida.txt)；本檔只附加
  逐句版面契約，不取代原始定位。

## 已證實

- table index12 進入 `0x2389f`，唯一 `sub_15F84` caller `0x238c8` 使用
  `FDTXT_013` index9。
- index9 依序解出12句，控制碼／operand 為：
  `FFEF/72, FFEF/72, FFEE/14, FFEF/72, FFEE/13, FFEF/72, FFEE/0,
  FFEE/4, FFEE/8, FFEE/9, FFEF/3, FFEE/0`。
- `count-aligned.json` 唯一映射為 `ch13.json` scene3 lines0..5，接 scene4
  lines0..5；場景邊界不得重排或重設同一 caller 的 utterance 序號。
- 對話完整結束後，原始順序才是 `0x238d0 sync_party`、共享尾段
  `0x237c8 JOIN3`、`0x231f2 chapter13`，最後由 campaign 進 `town_ch14`。

## 執行契約與限制

- 逐句資料必須保存原始 `FFEC`～`FFEF`、operand、`FFFE`、`FFFD`、頁／列及
  glyph token；12筆版面需與固定 raw 解碼逐筆相等。
- 多目標 mapping 依順序 lower 為兩個 `segments`。缺 caller context、mapping
  不唯一、分段總行數不等於12、缺 glyph／資產時失敗即關閉（fail-closed）。
- 正式回歸必須用與鍵盤共用的具型別輸入完成 opening、逐字、嘴型、分頁與
  closing；禁止以清除 `g.dialog` 略過。
- `0x36CD7(0x28)` 高階語意仍未知，且不屬本切片；不得以猜測名稱擴張 runtime。
- 完成後最高為 `RUNTIME-E1`。未修改 DOSBox 同狀態逐幀、精確音訊及 town14
  原版輸入仍是 E2 限制，不阻擋 99% 玩家可見相似門檻。
