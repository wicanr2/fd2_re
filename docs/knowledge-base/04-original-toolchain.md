# 04 — 當年開發工具考證(1995)

> 從固定雜湊 `FD2.EXE`、同版 `DOS4GW.EXE`、Miles 設定／驅動及 ANI 資源的
> 二進位指紋還原。每一項分開標示證據等級；工具家族已證實，不把版權年份
> 擅自提升為精確 compiler／linker 版本。

## 結論一覽

| 環節 | 當年工具 | 證據 | 重製對應 |
|---|---|---|---|
| 編譯器／runtime 家族 | **Watcom C/C++ 32-bit**；精確版本未知 | `FD2.EXE` 兩段 Watcom runtime 版權字串、Watcom 9.x FLIRT、`0x36CD7` stack runtime pattern | 已證實家族；v9.x→v10 只能列版本範圍強推論 |
| executable／linker 產物 | MZ stub＋**LE（Linear Executable）** | MZ `e_lfanew=0x28B8`，該處 magic `LE` | 已證實格式；「由哪一版 WLINK 產生」未知 |
| DOS 擴充器 | **Rational DOS/4GW 1.92**（1993-08-24 build） | 同版 `DOS4GW.EXE` 直接字串 `1.92`／`Aug 24 1993 16:10:14` | 已證實；重製不移植 |
| 音效中介層 | **Miles Design AIL 3.02，1995-01-18** | `DIG.INI`／`MDI.INI` 直接版本行；`SETSOUND.EXE` 同版 usage script | 已證實設定工具鏈；個別 API wrapper 仍依 caller 證實 |
| 音樂格式 | **XMIDI**(Miles/AIL 專用) | `FORM…XDIR…CAT …XMID` 於 `FDMUS` 資源 | XMIDI 可轉標準 MIDI 再合成 |
| 數位音效 | 各音效卡 `.DIG` 取樣驅動 | `SBLASTER.DIG`/`SB16.DIG`/`ULTRA.DIG`… | — |
| 顯示模式 | **VGA mode 13h**(320×200, 256 色) | 圖像 header 320×200、768B 調色盤(6-bit) | — |
| 圖塊 | 24×24 像素為基本單位 | `FDSHAP_000` 標頭 `18 00 18 00` | — |
| 動畫工具 | **AFM(Animation File Manager)v1.00**,作者 **Lo Yuan Tsung**(羅元聰),1993/09/29 | `ANI` 資源 #0 版權橫幅 | 見 `06-animation-format.md` |

## 細節

### 編譯器：Watcom C/C++ 32-bit

固定版 `FD2.EXE`（357,074 bytes、MD5
`b97caf2239a27a896069d03549d96e1e`、SHA-256
`222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`）具有：

- file offset `0x273`：`WATCOM C Run-Time system ... 1988-1992`；
- file offset `0x3D765`：`WATCOM C/C++32 Run-Time system ... 1988-1993`；
- MZ `e_lfanew=0x28B8`，該處為 `LE` magic；
- IDA Pro 9.4 的 Watcom 9.x FLIRT runtime 辨識；
- 已直接閉合的 `0x36CD7→0x36CEA→0x36D07` Watcom stack overflow runtime。

因此「Watcom C/C++ 32-bit 家族」與32-bit LE target為已證實。版權年份與 FLIRT
signature 不能唯一決定 compiler／linker patch level；舊文「約 v9.5/v10」改列為
**強推論版本範圍**，不得寫成精確工具版本。LE 格式也不能單獨證明是哪一版 WLINK。
> 意義:遊戲邏輯是 32-bit C 程式碼，反組譯(Ghidra/IDA)可得結構化函式 → 適合「反編當 oracle」。

### DOS 擴充器:Rational DOS/4GW
同版 `DOS4GW.EXE` 為244,716 bytes、MD5
`a75fdc120ae2fbb4cb2fd9e7b6a94858`、SHA-256
`bcba1fc7ea48469fcd97f7d33f2781db8903d90d93cda03505860177259dfaec`；內含
`DOS/4GW Protected Mode Run-time`、build `Aug 24 1993 16:10:14` 與版本 `1.92`。
因此本專案可把 DOS extender 精確綁定為 **Rational DOS/4GW 1.92**。
反組譯時要注意:`FD2.EXE` 內位址是保護模式線性位址，非 real-mode segment:offset。

### 音效：Miles Sound System／AIL 3.02
1990 年代商用音訊中介層。架構:
- `.MDI` = 各裝置的 **MIDI/音樂**驅動(AdLib/OPL3/SB/MPU-401/MT-32/GUS…)。
- `.DIG` = 各裝置的**數位取樣**驅動。
- `AILDRVR.LST` = 驅動自動偵測規則(依環境變數 `BLASTER` 等選卡)。
- 音樂資料為 **XMIDI**:`FORM..XDIR`(目錄)+ `CAT ..XMID`(序列集)。XMIDI 是 MIDI 的擴充，
  有迴圈控制器等;有成熟工具可轉回標準 MIDI(如後輪寫 `tools/xmi2mid.py`)。

`DIG.INI`（MD5 `6ea1dd6cfc3c6ef697d890606a2900b9`）與`MDI.INI`（MD5
`1eb7d3bb2f1f781b306c3a580fe3273d`）都直接寫
`Miles Design Audio Interface Library V3.02 of 18-Jan-95`；同版`SETSOUND.EXE`
為320,037 bytes、MD5 `acd891d19b109a068f05aa03236dbe95`。故 AIL 3.02 日期為
已證實，不再只寫模糊的「AIL v3」。`ADRV688.DIG`另直接署名`ES688 AIL 3.0 Driver`，
這是個別driver版本，不應反向覆蓋整套3.02設定工具鏈。

### 美術工具
無法從 binary 直接判定(未留檔名)。1990 年代台灣團隊常用 **Deluxe Paint** 類 256 色點陣工具 [假設]。
資產以 320×200 / 24×24、VGA 調色盤封裝，壓縮演算法待第 2 輪還原。

## 對重製的意義

- **規則/邏輯**:Watcom C → 反組譯可還原戰鬥/移動/AI 演算法(階段 3)。
- **音樂**:XMIDI是已知格式，可抽出轉MIDI並由現行Go／Ebiten音訊管線合成；
  早期C++／SDL2構想不是第二套可玩重製。
- **美術**:VGA 13h + 調色盤 + 24×24 圖塊 → 解壓後轉 PNG 即可用於現代引擎。
