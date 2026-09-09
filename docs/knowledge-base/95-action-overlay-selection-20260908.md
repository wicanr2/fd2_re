# 95 — 指令環的選取提示、可用性閘門與回合橫幅（2026-09-08）

## 固定輸入與工具

`FD2.EXE`：357074 位元組，MD5 `b97caf2239a27a896069d03549d96e1e`，
SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
以下位址全部是 DOS LE loader 線性位址，不是檔案偏移。

第一工具為 IDA Pro 9.4（`fd2-ida-authorized-local:latest`，
`tools/ida_probe_functions.py`，`FD2_IDA_ADDRESSES="0x1741c 0x173e7 0x177fc 0x17898 0x179d5 0x18d8c"`）；
第二套獨立驗證為 Capstone 5.0.3（`fd2-cap-local:latest`，`tools/disasm_le.py`）。
兩者的函式邊界與直接指令完全一致。可審查的最小匯出為
[`fd2_action_overlay_selection_20260908.json`](../data/ida/fd2_action_overlay_selection_20260908.json)。
沒有更名任何 IDA 資料庫物件。

## 已證實的函式分工

| 原始名稱 | 邊界 | 角色 |
|---|---|---|
| `sub_173E7` | `0x173e7..0x1741c` | 把 `[0x53c57]` 設成第一個 availability 為零的方向 |
| `sub_1741C` | `0x1741c..0x175a9` | 展開動畫的四格呈現；格號 `3*first + 2*second` |
| `sub_177FC` | `0x177fc..0x17898` | 方向／確認／取消輸入 |
| `sub_17898` | `0x17898..0x179d5` | 等待按鍵的迴圈；翻動閃爍相位並每輪呼叫 `sub_179D5` |
| `sub_179D5` | `0x179d5..0x17aa9` | 穩態重繪；對選中方向的格號加上閃爍相位 |
| `sub_18D8C` | `0x18d8c..0x190ac` | 戰鬥指令的 caller；建表後依 `[0x53c57]` 分派 |

`sub_177FC` 的 IDA 交叉參照有五個 caller：`0x16FA9`、`0x17334`、`0x18EEA`、
`0x19EC6`、`0x1BC89`。因此下列三條規則屬於整個 overlay chooser，不只戰鬥指令。

## 選中格是閃爍的第三個變體

`sub_179D5` 的 `0x17A60..0x17A8B`（bytes `89 F3 8B 44 24 2C 8B 14 98 89 D0 C1 E0 02
29 D0 89 C2 8B 44 24 30 8B 04 98 01 C0 01 D0 3B 35 57 3C 00 00 75 A9 03 05 13
3C 00 00 EB A1`）逐方向計算 `3*first + 2*second`，接著 `cmp esi, dword_53C57`；
只有等於目前選擇的方向才 `add eax, dword_53C13`。

`[0x53C13]` 由 `sub_17898` 的 `0x178BF..0x178F8` 維護：讀 BIOS 低字 `[0x46C]`，
減去 `[0x53C17]`，差值大於 3 或為負就把相位在 0 與 1 之間翻動並更新時間戳。
一個 BIOS tick 是 1193182/65536 Hz，所以週期是四個 tick，約 219.7 ms。

FDOTHER#2 的前十二格因此是每個動作一組三格：
攻擊 `0/1/2`、指令 `3/4/5`、物品 `6/7/8`、待機 `9/10/11`，
依序為「可用」「可用且選中」「不可用」。dosgolem 原版收據
（`work/ch01-town-parity-20260908/oracle-live-r11/`）第 235 幀的物品格畫的是
帶亮點的 cell 7，第 236 幀回到 cell 6，與上述指令一致。

重製端原本只用 `3*direction + 2*availability`，永遠取不到 `+1` 的變體，
所以切換方向時畫面上完全沒有提示。

## 方向鍵只接受可用的方向

`sub_177FC` 的 `0x17835..0x17897` 對四個掃描碼各檢查自己那一格：
`0x48` 檢查 `[ebx+0]`、`0x4B` 檢查 `[ebx+4]`、`0x4D` 檢查 `[ebx+8]`、
`0x50` 檢查 `[ebx+0xC]`；不為零就直接跳到共用的 `xor eax,eax` 尾端，
`[0x53C57]` 不變。也就是說原版按下不可用方向時沒有任何反應，
選擇不會停在紅底格上。

起始選擇由 `sub_173E7` 保證同一個不變量：從 0 起找第一個 availability 為零的
方向，四個都不可用時停在 4。`sub_18D8C` 在 `0x18E53` 與 `0x18ED6` 各呼叫一次，
第二次是在 availability 表更新之後。

重製端原本讓方向鍵自由改寫 `ringSel`，再於確認時顯示「此指令目前不可用」；
而且開環時硬寫 1，起始選擇可能落在原版不會停留的格上。

## 回合橫幅被原生整幀蓋掉

`drawNativeMapFrame` 是覆蓋整個畫面的 320×200 索引幀 ×2 blit。
`Draw` 原本在它之前呼叫 `drawPhaseBanner`，因此只要原生整幀被採用，
`PLAYER PHASE`／`ENEMY PHASE` 橫幅每一幀都會被立刻蓋掉。
v.1.0.19 的一般玩家路徑錄影（進入 `battle_ch01` 前後 6 秒、10 fps、
亮度提高 2.2 倍）確認整段淡入沒有任何可見橫幅。

橫幅本身是重製端的近似呈現，原版自己的回合字樣尚未解出；
把它移到原生整幀之後只恢復可見性，不宣稱與原版一致。

## 已套用的修正

- `fdother`：新增 `ActionOverlaySelectionBlink`（`[0x53C13]`／`[0x53C17]`）、
  `SelectedCellIndex`、`ActionOverlayInitialDirection`、
  `ActionOverlayAcceptsDirection`，四項各有直接指令依據與單元測試。
- `drawNativeActionOverlay`：穩態重繪對選中且可用的方向改用 `base + 相位`。
  展開／收合動畫由 `sub_1741C`／`sub_176B4` 擁有，兩者都沒有這個加法，
  因此只在 `actionOverlayBlocksInput()` 為假時套用。
- 四個 overlay chooser 的方向鍵改走同一個閘門，帶各自的 availability 表。
- 戰鬥開環改由 `ActionOverlayInitialDirection` 決定起始方向。
- `drawPhaseBanner` 移到原生整幀與移動資訊面板之後。

## 尚未關閉

- 取消一層退回指令環時，原版是否每次都重跑 `sub_173E7`，取決於 `0x18890`
  的重進場語意，尚未閉合；目前保留上一個選擇，只有它已不可用才回到第一個
  可用方向。
- 閃爍相位以 `nativeMapClock` 的 BIOS 低字推進，與原版同一個計數來源，
  但尚未做同狀態逐幀對拍。
- 攻擊可用性的 producer（`0x1B8A6`、`0x1C269`、unit `+0x27`）不在本輪範圍。
