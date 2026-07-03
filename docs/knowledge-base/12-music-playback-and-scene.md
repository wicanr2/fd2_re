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
| 18 | 0x26BB5(linear 0x025DB5,**boot 鏈 main→標題**)/ 0x2CFF5(linear 0x02C1F5,疑似 logo 落地,呼叫端未接) | **標題 / 開場曲**[驗:反組譯 boot 唯一呼叫 + 使用者實聽 2026-07-03 確認「登登登登磅礡」]。~~商店~~~~戰鬥~~ 兩個舊推定皆誤 |
| 11 | 0x2E0B6 / 0x31E06 / 0x3235D / 0x33217(最常用) | 世界地圖 / 一般場景 |
| 10 | 0x2DB34 / 0x2E0F9 | 城鎮 / 劇情 |
| 13 / 15 / 16 / 17 | 0x2E099 / 0x2E0E2 / 0x32344 / 0x31DF5 | 各特定場景 / 事件 |

> ⚠ 定案(2026-07-03,推翻兩次舊推定):track 18 = **標題 / 開場曲**。反組譯 boot 0x025DB5 唯一
> play_bgm(0,18) 在進標題 driver 前設定、全開場路徑不再換曲(bgm-title);使用者實聽確認=記憶中
> 「登登登登氣勢磅礡」那首。**2026-07-02 的「商店」與更早的「戰鬥」標記都是誤判**(商店=單聽檔案
> 觀感非溯源、戰鬥=doc12 早期推定);remake 曾把 018 誤掛 30 個戰鬥節點,真戰鬥/商店曲另 RE
> (章節曲表 0x51e63/0x51e81)。教訓:「推定場景」欄不可盡信,曲號→場景必須溯源到呼叫點,不能憑曲風印象。
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

## 戰鬥/商店 BGM(第15輪反組譯,續開場曲之後)

> 任務:釘死「戰鬥中」與「商店/城鎮」的真實曲號,取代 `gen_campaign.py` 現有的
> `BGM_BATTLE = "FDMUS_018"`(誤用,018 已在上節確認是 boot/標題曲)。
> 方法:規則62 靜態溯源——從 doc23 已知的兩張章節跳表 `0x51e63`/`0x51e81` dump 全 30 章 entry,
> 逐一核對呼叫端(`le_xref.py refs` + capstone 逐指令),窮舉走這兩張表與走常數 track 的呼叫點。

### 結論一:戰鬥 BGM = **每章查表**(`0x51e63`),非單一曲[驗]

全碼 5 處 `play_bgm` 走 `[0x53c03]`(章節)索引 `0x51e63` 表(byte,無查表偏移,索引=章節):

```
0x01047b   mov eax,[0x53c03]; movzx eax,[eax+0x1e63]; push eax; call 0x25977
0x025e4f   （緊接 call [eax*4+0x51d71] 章節戰前/劇情跳表之後)同上 movzx [eax+0x1e63]
0x025f26   （同上,另一分流路徑)
0x02610a   （同上,第三條分流路徑,esi=call 0x2cad7 結果)
0x026144   push -1;push 0;call 0x25977(停曲) → call 0x10010(戰場設置) → movzx [eax+0x1e63] → play_bgm
```

五處**全部**走 `0x51e63`,**沒有一處走 `0x51e81`**——確認 `0x51e63` = 戰鬥 BGM 表,`0x026144` 這條
(停曲 → 進戰場設置 0x10010 → 立刻重設曲號)是最直接的鐵證:曲號緊接在「進戰場」之後設定。

`0x51e63` 30 bytes(章節 0-29,dump 見 `extracted/exe_tables/bgm_chapter_tables.md`):

```
13 13 13 13 03 13 13 13 03 04 13 13 13 13 03 13 04 13 13 03 13 03 04 13 03 13 04 13 13 08
```

→ 換算 FDMUS(0x13=19、0x03=3、0x04=4、0x08=8):

| 章節 | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 |
|---|---|---|---|---|---|---|---|---|---|---|
| BGM | 019 | 019 | 019 | 019 | 003 | 019 | 019 | 019 | 003 | 004 |
| 章節 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 |
| BGM | 019 | 019 | 019 | 019 | 003 | 019 | 004 | 019 | 019 | 003 |
| 章節 | 20 | 21 | 22 | 23 | 24 | 25 | 26 | 27 | 28 | 29 |
| BGM | 019 | 003 | 004 | 019 | 003 | 019 | 004 | 019 | 019 | 008 |

`FDMUS_019` 是壓倒性主曲(30 章裡 18 章用它),`FDMUS_003`/`004`/`008` 穿插在特定章節。
**給 remake:`gen_campaign.py` 的 `BGM_BATTLE` 應改成這張 30-entry 表(依章節索引),不是單一常數。**

### 結論二:商店/城鎮 BGM = **track 10(`FDMUS_010`)**,常數非查表[驗]

兩處 `push 0;push 0xa;call 0x25977`(track=10 字面值):

- **`linear 0x2cd34`**:前置 `push [0x53c03]; call 0x4e4b9` → `al=[eax]`(取「目前章節城鎮資料」)
  → `call 0x1f882`(vsync 輔助)→ `play_bgm(0,10)`。與函式 `0x2d098`(下條)共用同一個
  `0x4e4b9([0x53c03])` helper 開場,判斷是「進城鎮」的共用前置動作。
- **`linear 0x2d2f9`**:在函式 `0x2d098`–`0x2d316`(依全域 `[0x412b]` 分支的場景轉場邏輯)尾端。
  該函式一進入就 `play_bgm(0,-1)` 停曲,依 `[0x412b]` 值(0/4/3/預設)先暫放 track 13/11/15/14
  其中一首、呼叫對應子場景載入函式(`0x2fc85`/`0x3072f`/`0x2e341`),**但除了 `[0x412b]==2` 這條
  例外分支,其餘所有路徑最終都會再呼叫一次 `play_bgm(0,10)` 才 `ret`**,同時把 doc13 記錄的
  「單位陣列 `[0x53a45]`」歸零(離開戰鬥狀態的訊號)。也就是說 13/11/15/14 只是這個轉場函式內部
  的過渡/中繼曲,**最終收斂的城鎮曲固定是 10**。
- **商店是城鎮的子選單,不另開新曲**:doc13 記錄的「神秘商店方向鍵游標」判定碼在 `0x2dbf3`,
  與上述城鎮函式群在同一 code 區塊(`0x2cd00`–`0x2e200`),窮舉該區塊內全部 `call 0x25977`
  只有 track 10/11/13/14/15 五種立即數(見上),**沒有第二個「商店專屬」track**——商店畫面沿用
  進城鎮時已設定好的 track 10,不會另外呼叫 play_bgm。
- 32 處 `play_bgm` 呼叫中,track=18(boot/標題,已在上節釘死)**從未出現在這個城鎮/商店 code 區塊**,
  確認舊 remake `BGM_STORY`(已是 `FDMUS_010`,寫對了)不受影響,**問題只在 `BGM_BATTLE` 誤用 018**。

### `0x51e81`(第二張表,語意待定)[推]

30 bytes(章節 0-29):`0c 0c 01 0c 06 0c 0c 01 06 04 0c 01 0c 01 06 0c 08 0c 0c 06 0c 06 08 0c 06 01 08 01 01 08`
→ FDMUS 001/004/006/008/012。**只在一處被讀**:`linear 0x1a57a`(函式 `0x1a4fc`–`0x1a620`),
與 `0x51e63` 在同一函式內先後各播一次——流程是「比較 table1[ch] vs table2[ch],不同就停曲 →
播 table2[ch](資源 0x52 一帶,疑似部隊編成/軍情畫面)→ 再比較一次、不同就停曲 → 播 table1[ch]
(資源 0x50 一帶,接著畫 9 格圖示,疑似出擊前的隊伍/部署畫面)」。**推測 `0x51e81` = 戰前準備
/部隊編成畫面的曲**(與戰鬥曲 `0x51e63` 前後銜接,同一章節通常同組但少數章節不同——
`cmp ebx,eax; je` 那段邏輯正是在處理「準備畫面曲跟戰鬥曲一樣就不重播,不一樣才停曲重放」)。
**不是商店/城鎮曲**——本輪窮舉的商店/城鎮 code 區塊完全不讀這張表。此表語意未 100% 釘死,
標 [推],留給下一輪如需要「戰前準備畫面 BGM」時複核(方法:找 `0x1a4fc` 函式的 caller,
釘死是哪個具體選單畫面)。

## MT-32 真 ROM 合成校音(第15輪)

> 任務:用使用者提供的**真 Roland MT-32 v1.07 ROM**(`/home/anr2/cht/mt32/`,本機,不外流)重錄全 15 首
> BGM,排查「氣勢偏弱」的主觀回報是否為技術缺陷(reverb 沒開?增益太低?)。

### 結論:不是 reverb 沒開,是 3 首曲跨曲峰值沒對齊

**① Reverb 排查——已排除**:`munt-smf2wav` 沒有「關閉 reverb」的旗標,預設輸出就是「乾聲+濕聲」
混音(對應 MT-32 開機預設 Reverb=ON、Room、Time/Level 中值)。用 `-w 4 -w 5` 把純 reverb-wet
stream 單獨 dump 出來實測(FDMUS_018):非靜音,RMS≈1510(16-bit 全幅 32767 的 4.6%),
證實**混音裡確實有殘響尾音**,不是漏開。原始 XMI 轉出的 MIDI 也完全沒有 CC91(reverb send)
或 sysex 覆寫——代表遊戲原始資料就是「吃 MT-32 開機預設殘響」,munt 預設行為正確還原此意圖。

**② 增益排查——找到真根因**:用 `ffmpeg -af volumedetect` 逐曲量測舊版(`extracted/music_ogg/`)
15 首峰值(max_volume)與均量(mean_volume):

| 曲目 | 舊版 mean/max(dB) | 備註 |
|---|---|---|
| FDMUS_011 | -15.3 / **-2.9** | 明顯離頂,浪費近 3dB |
| FDMUS_013 | -20.4 / **-3.5** | 離頂最多,聽感最弱 |
| FDMUS_014 | -18.3 / **-2.1** | 離頂 |
| 其餘 12 首 | 各異 / **0.0(頂滿)** | 本來就已用滿峰值,音量差是曲子本身編制疏密不同(非 bug) |

**FDMUS_011/013/014 這 3 首從沒把 munt 渲染出的動態空間用滿**——這才是「聽起來氣勢弱」的實際
可測差異,其餘 12 首峰值早已頂到 0dBFS,音量落差是**音樂編制本身**(疏密/聲部數)造成,
不是管線缺陷,不該用動態壓縮硬拉平(會失真、不還原)。

### 修法:線性峰值正規化(non-destructive,不用 loudnorm 動態壓縮)

`tools/export_music_ogg.sh` 新增:量測每曲 munt 原始 WAV 的 `max_volume`,計算單一線性增益值
補到目標峰值(預設 **-1.0dBFS**,非 0dBFS——libvorbis 有損編碼會有 inter-sample true-peak
overshoot,量測 v2 成品發現同一目標峰值編碼後實際落在 -1.3~0.0dBFS 之間飄動,故用 -1dBFS
留安全餘裕、避免解碼後削頂失真)。**只做等比例增益,不動態壓縮**——已頂滿的曲子相對安靜,
離頂的曲子按原比例補滿,曲子內部的強弱對比完全保留。

修法效果(舊 mean → v2 mean):FDMUS_011 -15.3→-13.4dB(+1.9)、FDMUS_013 -20.4→-17.9dB(+2.5)、
FDMUS_014 -18.3→-17.0dB(+1.3);其餘 12 首因目標峰值從 0→-1dBFS,mean 普遍降約 1dB(安全餘裕的
代價,聽感差異可忽略,換來不削頂)。

> ⚠ 踩過的坑:第一版腳本用 `grep -oE '\-?[0-9.]+' | head -1` 從 ffmpeg log 行抓 `max_volume`,
> 但該行開頭有 `[Parsed_volumedetect_0 @ 0x5f073f...]` hex 位址,`head -1` 誤抓位址裡的數字
> 當成峰值,算出離譜增益(部分曲不升反降)。改用 `sed -E 's/.*max_volume:[[:space:]]*(-?[0-9.]+) dB.*/\1/'`
> 錨定欄位名稱後才修正。

### v1.07 vs CM-32L 對照(FDMUS_018 標題曲,同一 MIDI、同峰值正規化目標)

| ROM | mean(dB) | RMS(dB) | Crest factor | 高頻(>4kHz)mean(dB) |
|---|---|---|---|---|
| MT-32 v1.07 | -10.4 | -10.36 | 3.70 | -34.5 |
| CM-32L v1.02 | -9.9 | -9.94 | 3.43 | -34.5 |

CM-32L 版整體 RMS 高約 0.4dB、crest factor 較低(3.43 對 3.70,代表能量分布更連續、較不「一波一波」)——
量測上支撐「CM-32L 音色較飽滿」的印象,但**高頻能量完全相同**(不是更明亮/更亮眼,是更「厚」)。
差異幅度不大(<0.5dB RMS 級),CM-32L 是加分但非戲劇性改變;根因是 CM-32L 出廠音色表比 MT-32
更豐富,同一 program-change 號碼在兩顆 ROM 對應到不同(通常更飽滿)的出廠音色。

### ROM 正名(munt 要求固定檔名)

munt 用 `-m <目錄>` 掃描目錄內容,靠檔名比對版本(非用戶自訂路徑),故渲染前需在暫存目錄放
`MT32_CONTROL.ROM`+`MT32_PCM.ROM`(或 `CM32L_CONTROL.ROM`+`CM32L_PCM.ROM`)正名複本/symlink,
指向 `/home/anr2/cht/mt32/` 內實際版本檔(如 `MT32_CONTROL.1987-10-07.v1.07.ROM`)。

### 全量產指令

```bash
# MT-32(15 首,峰值正規化至 -1.0dBFS,預設)
./tools/export_music_ogg.sh extracted/raw/FDMUS <ROM正名目錄> extracted/music_ogg_v2 4 -1.0 mt32
# CM-32L 對照(machine-id 換 cm32l,ROM 目錄換 CM32L_CONTROL.ROM+CM32L_PCM.ROM)
./tools/export_music_ogg.sh extracted/raw/FDMUS <CM32L正名目錄> extracted/music_ogg_v2_cm32l 4 -1.0 cm32l
```

輸出:`extracted/music_ogg_v2/`(本機,15 首 mt32 + `FDMUS_018_cm32l.ogg` 對照樣本;皆不入庫)。

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
- ✅ 開場/標題 BGM = track 18(見上節);戰鬥 BGM = 每章查表 `0x51e63`;商店/城鎮 BGM = track 10(見上節)。
- 把各 track 呼叫端對應到確切遊戲狀態名(片頭/世界圖/城鎮/戰鬥/劇情)。
- 確認短曲(勝利/事件)是否 loop_count=1(播一次)。
- SFX 取樣資源位置與「事件 → 音效」對應。
- `0x51e81` 表語意([推]戰前準備/部隊編成曲)未 100% 釘死,待查 `0x1a4fc` 函式 caller 確認觸發場景。

> 工具:`tools/le_xref.py`(解析 LE object/fixup,做字串 xref 與相對呼叫端搜尋)——
> 本輪 scene→track 即用它從 `FDMUS.DAT` 字串 xref 追到 `play_bgm`。
