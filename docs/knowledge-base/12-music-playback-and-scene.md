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

### `play_bgm(flag, track)` — 已反組譯(`0x25977`)

> 修正(第15輪):先前記錄的位址 `0x26777` 是誤植——實測該處是一段動畫幀計數器程式碼
> (`inc byte ptr [ebx]` 一類),與音樂無關。`play_bgm` 真正位址是 **`0x25977`**
> (與 doc23 §5 offset 總表一致),cdecl `push flag; push track; call 0x25977`,
> track 取自 `[esp+8]`。已用 `tools/disasm_le.py dis 0x25977 60` 逐指令核對。

切場景換曲走同一個函式 `play_bgm`(`0x25977`):

```
play_bgm(flag, track):
    if track == 目前曲號 [0x1A11]:  直接 return(同曲不重播)
    [0x1A11] = track                 記錄目前曲號
    if track == -1 (0xFF):           停曲(釋放序列,call 0x3BBD4)
    else:                            釋放舊曲 → 從 FDMUS.DAT 載入第 track 個資源(call 0x11FBA)
                                     → AIL init + start sequence(handle [0x3BFF], buffer [0x3EE0])
```

- **`[0x1A11]`**(linear `0x51a11`,obj2 data base `0x50000`)= 目前播放曲號(全域)。**`track = -1`** = 停止音樂(場景轉換常先停曲)。
- `track` 即 **FDMUS.DAT 的資源索引**(見 `07-…`;偶數位多為 3-byte 分隔,奇數位為 XMI)——
  第15輪逐指令核對 `0x25977` 內 `push dword ptr [esp+8]; push 0x1a79("FDMUS.DAT"); call 0x111ba`,
  **索引直接 = track,無查表/偏移轉換**,`FDMUS_NNN.bin`(`unpack_dat.py` 命名)即為 track=NNN。[驗]

### 場景 → 曲號(實測 32 處 `play_bgm` 呼叫的 track 立即數)

| track | 呼叫位置(file offset) | 推定場景 |
|---|---|---|
| `-1`(停曲) | 0x1B049 / 0x1B31D / 0x26F28 / 0x335BB … 多處 | 場景切換前停曲 |
| 4 | 0x2D3CF(ANI 播放器一帶) | 片頭 / 過場 |
| 18 | 0x26BB5(linear 0x025DB5,**boot 鏈 main→標題**)/ 0x2CFF5(linear 0x02C1F5,疑似 logo 落地,呼叫端未接) | ~~商店~~ → **第15輪反組譯:兩處呼叫都在開機/標題附近,查無商店呼叫點,「商店」待複核(見下方§開場/標題 BGM)** |
| 11 | 0x2E0B6 / 0x31E06 / 0x3235D / 0x33217(最常用) | 世界地圖 / 一般場景 |
| 10 | 0x2DB34 / 0x2E0F9 | 城鎮 / 劇情 |
| 13 / 15 / 16 / 17 | 0x2E099 / 0x2E0E2 / 0x32344 / 0x31DF5 | 各特定場景 / 事件 |

> ⚠ 修正(2026-07-02):track 18 經使用者實聽 = **商店音樂**,非戰鬥——「推定場景」欄不可盡信,
> 逐曲確認以「實際遊玩聽辨」為準(戰鬥曲號待聽辨)。
> 機制已確定:遊戲流程碼在切場景時呼叫 `play_bgm(_, track)` 指定曲號。各 track 的「確切場景名」
> 需把呼叫端函式對應到遊戲狀態(片頭 / 世界圖 / 城鎮 / 戰鬥 / 劇情)後補全;上表為依呼叫位置的推定。
> 短曲(勝利 / 事件提示)應以非迴圈方式播放(loop_count=1),待確認。

## 開場/標題 BGM(第15輪反組譯)

> 任務:釘死「開場過場/標題畫面」播的是哪首曲,取代 remake 現在的猜測值 `FDMUS_004`。
> 方法:規則62 靜態溯源——從已驗證的 boot 鏈(doc23)`main 0x25bf4 → 0x25ebb → title_seq 0x1f894`
> 逐指令反組譯,找鏈上實際的 `play_bgm` 呼叫,並窮舉全 32 處 `play_bgm` 呼叫的 track 立即數
> 逐一核對(排除誤判)。

### 結論:**boot/title BGM = track `0x12`(18)= `FDMUS_018`**[驗]

呼叫點 `linear 0x025db5`(main 內,`0x25bf4` 函式體,`0x25dbd` 呼叫 `0x25ebb` 標題/選單 driver **之前**):

```
0x025db1  push    0
0x025db3  push    0x12          ; track = 0x12 = 18
0x025db5  call    0x25977       ; play_bgm(0, 18)
0x025dba  add     esp, 8
0x025dbd  call    0x25ebb       ; ← 緊接著進標題序列 + 新遊戲/讀檔分流
```

這是 boot 鏈上**唯一**一次 `play_bgm` 呼叫——用 `callgraph_le.py edges` 逐一檢查
`title_seq(0x1f894)`、其子函式 `0x1f81e`、轉場 `0x286bd`、靜態插播 `0x1f73f`(doc23 §2.4⑥)、
`0x25b45`(main 前段 init),**均無第二次 `play_bgm` 呼叫**——曲號在整條「開機→標題可互動」
路徑上只被設定這一次,不會被後續覆蓋。[驗]

### 與 doc12 舊「track18=商店」推定的衝突(需使用者複核)

本節第74行(2026-07-02 實聽修正)記錄「track18 經實聽 = 商店」。**本輪反組譯與此衝突**,理由:

- 全 32 處 `play_bgm` 呼叫中,`track=0x12`(字面 push,非查表動態值)只出現在**兩處**:
  上述 boot 呼叫(`0x025db5`),以及 `0x02c1f5`(見下「未接來源」)。
  **沒有任何一個店鋪/城鎮場景的呼叫點 push 18**——動態查表呼叫(依章節取曲,呼叫點見
  `0x01047b`/`0x025e4f`/`0x025f26`/`0x02610a`/`0x026144`,經 `0x51e63` 章節→曲號表)實測前 30 章
  entry 值域為 `{0x03,0x04,0x08,0x13}`,**無 `0x12`**;另一張表 `0x51e81` 值域
  `{0x01,0x06,0x08,0x0c}` 亦無 `0x12`。也就是說 track18 在靜態碼裡**只跟 boot/標題相關**,
  沒有任何一條路徑會在「進商店」時 push 18。
- 舊推定的依據是**單獨聽 `FDMUS_018.ogg` 這個檔案本身的觀感**(`聽辨清單.md`「播放→請填」的流程),
  不是對照到一個已驗證的商店呼叫點——換言之是憑印象判斷曲風,不是溯源驗證。
- 使用者記憶「開場登登登登氣勢磅礡進場」(短促、有衝擊感的進場)——`FDMUS_018` 長度 137 拍
  (doc07)不算短曲,但 boot 呼叫是**唯一**候選,靜態證據沒有第二個選項。

**建議**:請針對「這是開場標題那一刻的音樂」重新聽一次 `extracted/music_ogg/FDMUS_018.ogg`
(而非泛泛判斷「像不像商店曲」),再決定是否推翻本輪結論。若複聽仍覺得不像標題曲,則衝突本身
值得記一筆(可能代表這條 boot 呼叫另有隱情,如「先鋪一段序曲、螢幕還沒淡入就先起音樂」的
慣例),而非直接否定靜態證據。

### 未接來源(第15輪新發現,[推],未接上 caller):`0x2bcea` 的 track18→track4 序列

反組譯 `play_bgm` 全部呼叫端時,另外發現一個**與 boot 鏈脫鉤**(`callgraph_le.py callers`
以 main 為種子重建可達集,直接 call/jmp 掃描 46782 條可達指令內找不到任何指令呼叫它)、
但內容高度像「片頭/標題」的函式 `0x2bcea`:依序畫多張全螢幕圖(FDOTHER.DAT 動態索引)、
中途 `[0x53c03]`(章節變數)借用值 `0x1a`(26,doc23 已知章節變數會被借用成特殊值,如
序章借 `0x20`/`0x1f`)分支選圖,尾段:

```
0x02c1ac  call 0x25977(1,-1)     ; 停曲
0x02c1c9..0x02c1e9  載入+blit 一張動態索引的 FDOTHER 圖
0x02c1ec  call 0x1f525           ; 調色盤淡入
0x02c1f5  call 0x25977(0,0x12)   ; ★ 起 track18 = FDMUS_018
...(續畫兩張角色相關圖、跑一段陣列迴圈)...
0x02c5cf  call 0x25977(0,4)      ; ★ 切換到 track4 = FDMUS_004
```

「停曲→淡入新圖→起曲」這個模式與 doc23 §2.4①(dosbox 實拍:紅閃光→『2』縮放定位→
白閃光→色板收斂)的節奏相符,**如果**這函式真的會被執行,它暗示的敘事是「track18 是
logo 落地那一瞬的短促起手曲,隨後畫面接續(角色/城堡等更多分鏡)時換成 track4 撐場」——
與舊 doc12「track4=片頭/過場」的推定並不衝突,反而互補。但**呼叫端未找到**(不在
`0x51d71`/`0x51de9` 兩張章節跳表內,也不在任何已知直接 call/jmp),不排除是尚未觸發的
死碼/備用分支,**不可當結論使用,僅記錄留給下一輪追查**(方向:查是否經事件腳本
(doc19)/其他資料驅動 dispatch 表呼叫,而非本輪已窮舉的直接 call)。

### 給 remake 的結論值

`remake/cmd/fd2/title.go` 開場 BGM 應改用 **`FDMUS_018`**(對應 boot 鏈唯一確認呼叫);
`FDMUS_004` 若要留用,建議改配「開場過場/序章故事分鏡」(呼應舊 track4=片頭/過場 推定,
且 `0x2bcea`(若證實會執行)顯示 track4 接在 track18 之後),而非標題畫面本身。
兩者是否真的先後銜接(18→4)仍待 `0x2bcea` 呼叫端釘死後才能定案。

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
