# 04 — 當年開發工具考證(1995)

> 從 `FD2.EXE` 與音效驅動的二進位指紋還原。漢堂國際開發《炎龍騎士團2》所用的技術棧。
> 全部 [已驗證] —— 字串證據見下。

## 結論一覽

| 環節 | 當年工具 | 證據 | 重製對應 |
|---|---|---|---|
| 編譯器 | **Watcom C/C++ 32-bit**(1988–1993 版，約 v9.5/v10) | `WATCOM C/C++32 Run-Time system. (c) Copyright by WATCOM International Corp. 1988-1993` | C++ 重製可直接讀演算法;Go 版重寫 |
| DOS 擴充器 | **Rational DOS/4GW**(© 1987–1993) | `RATIONAL DOS/4G` / `DOS/4G Copyright (C) Rational Systems, Inc. 1987 - 1993` | 重製不需要;原為跑 32-bit 保護模式 |
| 音效中介層 | **Miles Design AIL v3**(Audio Interface Library) | `AIL3DIG`(數位)/`AIL3MDI`(MIDI) | 用現代音訊管線取代 |
| 音樂格式 | **XMIDI**(Miles/AIL 專用) | `FORM…XDIR…CAT …XMID` 於 `FDMUS` 資源 | XMIDI 可轉標準 MIDI 再合成 |
| 數位音效 | 各音效卡 `.DIG` 取樣驅動 | `SBLASTER.DIG`/`SB16.DIG`/`ULTRA.DIG`… | — |
| 顯示模式 | **VGA mode 13h**(320×200, 256 色) | 圖像 header 320×200、768B 調色盤(6-bit) | — |
| 圖塊 | 24×24 像素為基本單位 | `FDSHAP_000` 標頭 `18 00 18 00` | — |
| 動畫工具 | **AFM(Animation File Manager)v1.00**,作者 **Lo Yuan Tsung**(羅元聰),1993/09/29 | `ANI` 資源 #0 版權橫幅 | 見 `06-animation-format.md` |

## 細節

### 編譯器:Watcom C/C++ 32-bit
1990 年代 DOS 遊戲主流。`FD2.EXE` 是 **LE(Linear Executable)** 格式，由 Watcom 連結器產出，
搭配 DOS/4GW 進保護模式。Run-Time 版權字串同時出現 1988-1992 與 1988-1993 兩段，
對應 Watcom C/C++ 9.x→10.0 過渡期(1993)。
> 意義:遊戲邏輯是 32-bit C 程式碼，反組譯(Ghidra/IDA)可得結構化函式 → 適合「反編當 oracle」。

### DOS 擴充器:Rational DOS/4GW
Watcom 隨附的精簡版 DOS/4G。`DOS4GW.EXE`(1993)負責把 16-bit DOS 切到 32-bit 平坦記憶體。
反組譯時要注意:`FD2.EXE` 內位址是保護模式線性位址，非 real-mode segment:offset。

### 音效:Miles Sound System(AIL v3)
1990 年代商用音訊中介層。架構:
- `.MDI` = 各裝置的 **MIDI/音樂**驅動(AdLib/OPL3/SB/MPU-401/MT-32/GUS…)。
- `.DIG` = 各裝置的**數位取樣**驅動。
- `AILDRVR.LST` = 驅動自動偵測規則(依環境變數 `BLASTER` 等選卡)。
- 音樂資料為 **XMIDI**:`FORM..XDIR`(目錄)+ `CAT ..XMID`(序列集)。XMIDI 是 MIDI 的擴充，
  有迴圈控制器等;有成熟工具可轉回標準 MIDI(如後輪寫 `tools/xmi2mid.py`)。

### 美術工具
無法從 binary 直接判定(未留檔名)。1990 年代台灣團隊常用 **Deluxe Paint** 類 256 色點陣工具 [假設]。
資產以 320×200 / 24×24、VGA 調色盤封裝，壓縮演算法待第 2 輪還原。

## 對重製的意義

- **規則/邏輯**:Watcom C → 反組譯可還原戰鬥/移動/AI 演算法(階段 3)。
- **音樂**:XMIDI 是已知格式 → 可抽出轉 MIDI，C++/Ebiten 兩版皆能用 SoundFont 合成。
- **美術**:VGA 13h + 調色盤 + 24×24 圖塊 → 解壓後轉 PNG 即可用於現代引擎。
