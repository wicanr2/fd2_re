# 玩家第 17 戰戰後原生對話綁定證據

## 參考版本與定位

- `FD2.EXE`：357074 bytes；MD5 `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_017.bin`：4434 bytes；MD5 `f4fb40584d28ce20d61706b481f3f227`；
  SHA-256 `ece28ae2d45373c4265149f03bbbd80c7b9a0c36197712120ffb0990bdf3f2ed`。
- 位址均為IDA Pro 9.4的`FD2.EXE`線性位址。handler、roster_has(18)雙分支、
  60／61 frontiers、group3、ACTING50–53與JOIN16主證據為
  [`../fd2_ch16_post_ida.txt`](../fd2_ch16_post_ida.txt)。

## 已證實

- `0x23BBE`使用`FDTXT_017` index5，共4句：
  `FFEF/18, FFEE/0, FFEF/18, FFEE/11`；映射至`ch17.json` scene3 lines0..3。
- `0x23C2B`使用index7，共3句：`FFEF/18, FFEE/15, FFEF/18`；映射至
  scene4 lines1..3。
- `0x23C86`使用index6，共1句：`FFEF/16`；映射至scene4 line0。
- `0x23CB7`使用index8，共18句：
  `FFEE/0, FFEF/16, FFEE/4, FFEF/16, FFEE/0, FFEE/4, FFEF/16, FFEE/17,
  FFEF/16, FFEE/17, FFEF/16, FFEE/4, FFEE/0, FFEF/16, FFEE/0, FFEE/11,
  FFEE/3, FFEE/0`；映射至scene4 lines4..21。全量為4個caller、26句。
- roster有角色18時播放index5，經pan、group3與ACTING52；沒有角色18時先layout，
  播放index7，經ACTING50、pan、group3與ACTING51。兩路共同播放index6、
  ACTING53、index8，最後才JOIN16與chapter遞增。

## 執行契約與限制

- 二十六句逐筆保存control、operand、頁／列與glyph token，並與固定raw equality相等。
- 有角色18路徑保持60→61 slots、播放23句；無角色18路徑保持61→62 slots、播放22句。
  正式輸入完成opening、逐字、嘴型、分頁與closing後才能執行後續演出、JOIN、同步、
  `town_ch18`與存讀檔；禁止直接清除`g.dialog`。
- 缺caller context、唯一mapping、原始glyph、地圖底圖、頭像、roster provenance、
  group3或合法frontier時，在隊伍交易前失敗即關閉（fail-closed）。
- 完成後最高為`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊與town18原版輸入仍屬
  E2限制，不阻擋99%玩家可見相似門檻。
