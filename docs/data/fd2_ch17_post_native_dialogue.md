# 玩家第 18 戰戰後原生對話綁定證據

## 參考版本與位址基準

- `FD2.EXE`：357074 bytes；MD5
  `b97caf2239a27a896069d03549d96e1e`；SHA-256
  `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
- `FDTXT_018` 是固定版 `FDTXT.DAT` 的第 18 個解包資源；`FDTXT.DAT`：
  120502 bytes；MD5 `fe5c487ce4313485f1da9d48d35b05f9`；SHA-256
  `a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69`。
- handler 位址均為 IDA Pro 9.4 的 `FD2.EXE` 線性位址；原始函式名稱、位址與
  caller 順序的主證據是
  [`fd2_ch17_post_ida.txt`](fd2_ch17_post_ida.txt)。本檔只附加可編輯對話的
  映射語意，不取代原始定位資訊。

## 已證實

- `sub_23CD5` 先於 `0x23d15` 呼叫 `sub_11506` 同步隊伍，才開始戰後版面與
  對話；不可把同步延後到對話或入隊之後。
- 四個 `sub_15F84` caller 依序為：

  | caller | `FDTXT_018` index | 原始句數 | 可編輯目標 |
  |---|---:|---:|---|
  | `0x23d60` | 7 | 5 | `ch18.json` scene 3 lines 0..4 |
  | `0x23d9b` | 8 | 2 | `ch18.json` scene 3 lines 5..6 |
  | `0x23dd6` | 9 | 2 | `ch18.json` scene 3 lines 7..8 |
  | `0x23e11` | 10 | 12 | scene 3 line 9，接 scene 4 lines 0..10 |

- 合計為四個 caller、21 句。`count-aligned.json` 對 index 10 保存兩個有序
  `targets`；這不是文字推測，而是原始句數與可編輯逐句索引的唯一對齊結果。
- index 10 必須編譯成兩個有序 `segments`，但其 12 筆原始控制碼版面仍屬同一
  caller，順序不得因場景邊界重設或重排。

## 執行契約與證據等級

- 原始 `FFEC`～`FFEF` 控制碼、operand、`FFFE` 換行、`FFFD` 分頁及原始 glyph
  token 由 `FDTXT_018` 直接解碼，屬已證實。
- `ch18.json` 的場景與行號是可編輯投影；只有逐句數量與順序對齊屬已證實，
  現代場景名稱本身不是原版 ABI。
- 缺 caller context、無唯一 count-aligned mapping、分段行數總和不等於原始
  句數、缺 glyph 或不支援的控制碼時，一律失敗即關閉（fail-closed）。
- 本切片完成後最高為重製執行期 E1；未修改原版的一般玩家同狀態比較仍是 E2
  限制，不阻擋 99% 玩家可見相似度的重製收尾。
