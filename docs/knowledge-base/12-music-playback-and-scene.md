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

- **15 首 XMI**(見 `07-…`)對應不同場景:長曲(FDMUS_008 250 拍等)= 戰鬥 / 地圖 BGM;
  短曲(FDMUS_016 5 拍、FDMUS_017 15 拍)= 勝利 / 事件提示樂句(播一次不迴圈)。
- **場景 → 曲目對應表**:由遊戲流程碼在切場景時指定曲號。[推定] 應有一張「章節/場景 → FDMUS 序號」
  常數表;確切對應待反組譯場景載入碼補上(後輪)。

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
- 反組譯場景載入碼,dump「場景 → FDMUS 曲號」對應表。
- 確認短曲(勝利/事件)是否 loop_count=1(播一次)。
- SFX 取樣資源位置與「事件 → 音效」對應。
