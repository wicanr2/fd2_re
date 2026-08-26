# 玩家第 16 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_016.bin`：3258 bytes；MD5 `f8d041321824c90c86d3527e21dfca0c`；
  SHA-256 `6b702dd3a5d91d119d63ceda2470e1f33e10d6f7a385dff82fbc451ac590eaa9`。
- 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址。handler、76-slot frontier、
  round／inactive／word42分支、ACTING49與JOIN18主證據為
  [`../fd2_ch15_post_ida.txt`](../fd2_ch15_post_ida.txt)。

## 已證實

- `0x23ADC`使用`FDTXT_016` index2，共3句：
  `FFEF/18, FFEE/15, FFEF/18`；映射至`ch16.json` scene1 lines1..3。
- `0x23B17`使用index3，共5句：
  `FFEE/0, FFEE/4, FFEE/13, FFEE/4, FFEE/0`；映射至scene1 lines4..8。
- `0x23B40`使用index4，共15句：
  `FFEF/18, FFEE/5, FFEF/18, FFEE/0, FFEE/4, FFEF/18, FFEE/0, FFEE/12,
  FFEE/0, FFEF/18, FFEE/15, FFEE/4, FFEE/13, FFEF/18, FFEE/0`；映射至
  scene1 lines9..23。全量為3個caller、23句。
- round>18或slots66..73 inactive count>4時，依序播放index2、ACTING49、index3，
  共8句且不JOIN18。兩者皆不成立時，slot0 word42<`0x140`不播放這三個caller也
  不JOIN；word42>=`0x140`才播放index4的15句並JOIN18。

## 執行契約與限制

- 二十三句逐筆保存control、operand、頁／列與glyph token，並與固定raw equality相等。
- 正式輸入完成opening、逐字、嘴型、分頁與closing後才能ACTING、JOIN、同步及進城鎮；
  禁止直接清除`g.dialog`。四條分支都必須保持76-slot persistent-first topology，並
  驗證`town_ch17`存讀檔。
- 缺caller context、唯一mapping、原始glyph、地圖底圖、頭像、round／inactive／
  word42 provenance或76-slot frontier時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊與town17原版輸入仍屬
  E2限制，不阻擋99%玩家可見相似門檻。
