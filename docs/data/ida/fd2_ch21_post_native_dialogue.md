# 玩家第 22 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_022.bin`：2742 bytes；MD5 `0d15cc2083ab89bc1ba6f22afcee2157`；
  SHA-256 `be1f18e7b69ecfd899707cc37ff82767deaf7f5c30502bcc3ee1786e500f622c`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、layout、ACTING、PAN、
  indexed transition、調色盤淡出、共享尾段與 chapter transition 主證據為
  [`fd2_ch21_post_ida.txt`](fd2_ch21_post_ida.txt)。

## 已證實

- caller `0x24539` 使用 `FDTXT_022` index4，共3句：
  `FFEE/21, FFEE/24, FFEE/21`。
- caller `0x2456A` 使用 index5，共1句：`FFEE/24`。
- caller `0x245A7` 使用 index6，共7句：
  `FFEE/24, FFEE/21, FFEE/4, FFEE/0, FFEE/24, FFEE/8, FFEE/24`。
- 三個索引合計11句，依唯一的 count-aligned 映射分別對應 `ch22.json`
  scene2 lines0..2、scene2 line3、scene2 lines4..10。
- 原始順序為layout、index4三句、ACTING65、index5一句、PAN、ACTING66、
  index6七句、PAN、indexed transition、延遲、調色盤淡出、`sync_party`、chapter22。

## 執行契約與限制

- 十一句逐筆保存 control、operand、頁／列與 glyph token，並與固定原始資料相等。
- 正式戰果確認後，必須以正式故事輸入完成 opening、逐字、嘴型、分頁與 closing；
  禁止直接清除 `g.dialog`。三段對話完成後才能執行 indexed transition、同步隊伍並進
  `preparation_ch23`。
- 缺 caller context、唯一映射、原始 glyph、73／79-slot frontier、戰場游標來源或
  indexed 畫面資產時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為 `RUNTIME-E1`；未修改 DOSBox 同狀態、精確音訊與第23戰整備原版輸入
  仍屬 E2 限制，不阻擋玩家可見99%相似門檻。

## 2026-08-27 現況勘誤

較早的 [`fd2_ch21_post_ida.txt`](fd2_ch21_post_ida.txt) 仍記錄「沒有正式 binding」；
該說法只描述當時狀態，現已由正式 `ch21_post.json`、73／79-slot runtime guard、
indexed transition cursor bridge 及戰果→整備→存讀檔回歸取代。這不改變原始位址、
bytes 或推論等級，也不重開已閉合的 renderer／handler。
