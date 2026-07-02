# 12 — 音樂播放與場景切換機制

> 遊戲怎麼放背景音樂、怎麼在「場景切換(片頭→城鎮→戰場→劇情)」時換曲。
> 第 3 輪反組譯 `FD2.EXE`(AIL API 字串)+ 設定檔。音樂資料格式見 `07-music-xmidi-format.md`。

## 音訊中介層:Miles AIL V3.02 (1995-01-18)

原版不直接碰音效卡,全透過 **Miles Design Audio Interface Library (AIL) V3.02**。
EXE 內含完整 AIL API 呼叫(`AIL_startup`、`AIL_call_driver`、timer、sequence、sample…)。

兩條音訊管線:
- **音樂(MIDI/XMIDI)**:`.MDI` 驅動 + sequence API。
- **數位音效(取樣)**:`.DIG` 驅動 + sample API。

驅動選擇:啟動時讀 `MDI.INI` / `DIG.INI`(本作預設 Sound Blaster,IO 220h),AIL 載入對應 `.MDI`/`.DIG`。
玩家可用 `SETSOUND.EXE` 改設定。

## 背景音樂播放流程(XMIDI sequence)

確定的 AIL 序列 API(EXE 字串):

```
AIL_startup()                          遊戲啟動,初始化 AIL + 載驅動
AIL_allocate_sequence_handle(drvr)     配置一個序列 handle
AIL_init_sequence(handle, xmi, seq#)   把某首 XMI(從 FDMUS.DAT 取出)載入 handle
AIL_set_sequence_loop_count(h, n)      設迴圈次數(BGM 設無限循環)
AIL_set_sequence_volume(h, vol, ms)    音量(可漸入)
AIL_start_sequence(h)                  開始播放
AIL_sequence_status(h)                 查詢播放狀態(是否結束)
AIL_stop_sequence(h) / AIL_end_sequence(h)  停止
AIL_set_sequence_tempo(h, tempo, ms)   速度(可漸變)
AIL_map_sequence_channel(...)          聲道對映
```

播一首 BGM = 「從 `FDMUS.DAT` 取出該曲 XMI → `AIL_init_sequence` → 設迴圈 → `AIL_start_sequence`」。

## 場景切換時的換曲

切換場景(片頭 / 世界地圖 / 城鎮 / 戰場 / 劇情對話 / 勝敗)時:

```
1. AIL_stop_sequence(目前 BGM)            停掉舊曲(可先漸出音量)
2. 載入新場景資料(地圖 FDFIELD.DAT、背景 BG.DAT、單位 sprite…)
3. 從 FDMUS.DAT 取該場景對應曲 → AIL_init_sequence → AIL_start_sequence
```

### `play_bgm(flag, track)` — 已反組譯(`0x26777`)

切場景換曲走同一個函式 `play_bgm`(`0x26777`):

```
play_bgm(flag, track):
    if track == 目前曲號 [0x1A11]:  直接 return(同曲不重播)
    [0x1A11] = track                 記錄目前曲號
    if track == -1 (0xFF):           停曲(釋放序列,call 0x3BBD4)
    else:                            釋放舊曲 → 從 FDMUS.DAT 載入第 track 個資源(call 0x11FBA)
                                     → AIL init + start sequence(handle [0x3BFF], buffer [0x3EE0])
```

- **`[0x1A11]`** = 目前播放曲號(全域)。**`track = -1`** = 停止音樂(場景轉換常先停曲)。
- `track` 即 **FDMUS.DAT 的資源索引**(見 `07-…`;偶數位多為 3-byte 分隔,奇數位為 XMI)。

### 場景 → 曲號(實測 32 處 `play_bgm` 呼叫的 track 立即數)

| track | 呼叫位置(file offset) | 推定場景 |
|---|---|---|
| `-1`(停曲) | 0x1B049 / 0x1B31D / 0x26F28 / 0x335BB … 多處 | 場景切換前停曲 |
| 4 | 0x2D3CF(ANI 播放器一帶) | 片頭 / 過場 |
| 18 | 0x26BB5 / 0x2CFF5 | **商店**(FDMUS_018;使用者實聽確認,推翻先前「戰鬥」推定) |
| 11 | 0x2E0B6 / 0x31E06 / 0x3235D / 0x33217(最常用) | 世界地圖 / 一般場景 |
| 10 | 0x2DB34 / 0x2E0F9 | 城鎮 / 劇情 |
| 13 / 15 / 16 / 17 | 0x2E099 / 0x2E0E2 / 0x32344 / 0x31DF5 | 各特定場景 / 事件 |

> ⚠ 修正(2026-07-02):track 18 經使用者實聽 = **商店音樂**,非戰鬥——「推定場景」欄不可盡信,
> 逐曲確認以「實際遊玩聽辨」為準(戰鬥曲號待聽辨)。
> 機制已確定:遊戲流程碼在切場景時呼叫 `play_bgm(_, track)` 指定曲號。各 track 的「確切場景名」
> 需把呼叫端函式對應到遊戲狀態(片頭 / 世界圖 / 城鎮 / 戰鬥 / 劇情)後補全;上表為依呼叫位置的推定。
> 短曲(勝利 / 事件提示)應以非迴圈方式播放(loop_count=1),待確認。

## 音效(SFX)

戰鬥打擊 / 施法 / 選單操作音用數位取樣,走 `.DIG` 管線:
`AIL_init_sample` → `AIL_start_sample`,音色資料在 `RAP10.DIG` 等(實為驅動)/ 遊戲內取樣資源。
[推定] SFX 取樣的存放與索引待確認。

## 重製對應(SDL2 / Ebiten)

| 原版 | 現代 |
|---|---|
| AIL XMIDI sequence | XMI→MID(`tools/xmi2mid.py`)+ SoundFont 合成(SDL2_mixer / oto + 軟體合成) |
| loop_count 無限 | 音樂 loop 播放 |
| 場景換曲 stop+start | 場景管理器在切 state 時 crossfade BGM |
| `.DIG` 取樣 SFX | WAV/OGG 音效 |

## 待辦(後輪)
- ✅ 「場景 → FDMUS 曲號」對映:`play_bgm`(0x26777)+ 32 處呼叫 track 已反組譯(見上表)。
- 把各 track 呼叫端對應到確切遊戲狀態名(片頭/世界圖/城鎮/戰鬥/劇情)。
- 確認短曲(勝利/事件)是否 loop_count=1(播一次)。
- SFX 取樣資源位置與「事件 → 音效」對應。

> 工具:`tools/le_xref.py`(解析 LE object/fixup,做字串 xref 與相對呼叫端搜尋)——
> 本輪 scene→track 即用它從 `FDMUS.DAT` 字串 xref 追到 `play_bgm`。
