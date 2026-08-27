# 玩家第 24 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_024.bin`：1722 bytes；MD5 `9acf528661a32fc12d6f42eb4eedc3c5`；
  SHA-256 `6a705b7a455393175e273c0ad5a7931acd9b53763e0fe73a2f5c552f26bb00e1`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、兩段 staging、
  BIOS tick gate、DAC 更新、同步與 chapter transition 主證據為
  [`fd2_ch23_post_ida.txt`](fd2_ch23_post_ida.txt)。

## 已證實

- caller `0x24C4C` 使用 `FDTXT_024` index2，共3句：
  `FFEE/21, FFEE/24, FFEE/20`。唯一 count-aligned 映射跨場景：
  `ch24.json` scene0 line14，接 scene1 lines0..1。
- caller `0x24CAD` 使用 index3，共8句：
  `FFEE/8, FFEE/0, FFEE/9, FFEE/5, FFEE/4, FFEE/5, FFEE/24, FFEE/21`；
  對應 scene1 lines2..9。
- 兩個索引合計11句。原始順序為index2三句、stage2..9各30次tick-gated draw、
  index3八句、stage10..14各12次DAC／draw／tick、`sync_party`、chapter24。

## 執行契約與限制

- 十一句逐筆保存 control、operand、頁／列與 glyph token，並與固定原始資料相等；
  index2必須保存跨場景有序 segments，不可把三句誤塞入單一場景。
- 正式戰果確認後，必須以正式故事輸入完成 opening、逐字、嘴型、分頁與 closing；
  禁止直接清除 `g.dialog`。兩段對話各自完成後，才可進入相鄰的原生 staging loop；
  最後才能同步隊伍並進 `preparation_ch25`。
- binding 明示只接受86-slot topology並領回節點入口保存的battle view；不得以固定座標
  或測試專用欄位取代這個正式交接。
- 缺 caller context、唯一映射、原始 glyph、86-slot topology、selector cache、戰場視圖、
  FDOTHER #42 staging、clock或indexed buffer時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為 `RUNTIME-E1`；未修改 DOSBox 同狀態、精確 BIOS tick／音訊與第25戰
  整備原版輸入仍屬 E2 限制，不阻擋玩家可見99%相似門檻。
