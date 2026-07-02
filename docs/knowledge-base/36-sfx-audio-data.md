# 36 — 音效(SFX)資料位置與格式

> 回答「音效資料存在哪、什麼格式」。音樂(BGM)見 `07`/`12`;`.DIG`/`.MDI` 是 Miles AIL 的**驅動程式**
> (依音效卡型號選一份載入),不是遊戲資料,見 `01` §6 / `04`。本篇反組譯 `FD2.EXE` 找出「遊戲真正播放
> 的數位音效樣本存在哪個檔案、哪個 index、什麼編碼」。

## 結論(一句話)

**《炎龍騎士團2》有數位音效,儲存在 `FDOTHER.DAT`(而非獨立音效檔),以巢狀 `LLLLLL` 容器裝著多個
「8-bit unsigned mono raw PCM」樣本,播放走 Miles AIL 的 `AIL_*_sample` 系列 API(非 `.DIG` 驅動本身
含資料 ── `.DIG` 只是硬體訪問層)。**

## 證據鏈(反組譯,file offset = linear,已用 capstone 交叉驗證)

### 1. 找到「播 SFX」函式:`0x26896` / `0x26945`

戰場命中/選單游標等處呼叫的通用「播放樣本」函式(`table_ptr, index, priority` 三參數):

```
0x026896  play_sfx_a(table, index, priority):
0x0268c7    push  [0x53ee4]            ; SFX 播放 handle A
0x0268cd    call  0x3a2b5              ; 取狀態/index 檢查(-1 則跳過)
0x0268dc    mov   eax, index
0x0268e0    shl   eax, 2               ; index*4 → 進表
0x0268e3    add   eax, table_ptr
0x0268eb    add   edx, [eax+6]         ; edx = table_ptr + entry.start   ← 樣本起始位址
0x0268f2    mov   edx, [eax+6]
0x0268f5    mov   eax, [eax+0xa]       ; entry.end
0x0268f8    sub   eax, edx             ; len = entry.end - entry.start
0x0268fd    push  [0x53ee4]
0x026903    call  0x39fd1              ; AIL_init_sample(handle)
0x02690b    push  len
0x02690e    push  addr
0x026912    push  [0x53ee4]
0x026918    call  0x3a144              ; AIL_set_sample_address(handle, addr, len)
0x026920    push  priority
0x026924    push  [0x53ee4]
0x02692a    call  0x3a55e              ; AIL_set_sample_loop_count(handle, priority)
0x026932    push  [0x53ee4]
0x026938    call  0x3a248              ; AIL_start_sample(handle)
```

`0x026945` 是同一份函式的第二份拷貝,操作另一個獨立 handle `[0x53ee8]`(雙聲道,兩個 SFX 可疊播)。

**呼叫端內嵌函式名確認鏈**(每個 `AIL_*` wrapper 內都有 debug-trace,push 一個字串位址再呼叫
`0x3ff1b`;字串位址用 fixup 解回實際文字,逐一比對):

| wrapper 位址 | 內嵌字串(file offset → 文字) | 對應 AIL API |
|---|---|---|
| `0x39fd1` | `0x505f8` → `"AIL_init_sample(0x%X)"` | `AIL_init_sample` |
| `0x3a144` | `0x50632` → `"AIL_set_sample_address(0x%X,0x%X,%u)"` | `AIL_set_sample_address` |
| `0x3a55e` | `0x5073b` → `"AIL_set_sample_loop_count(0x%X,%d)"` | `AIL_set_sample_loop_count` |
| `0x3a248` | `0x50679` → `"AIL_start_sample(0x%X)"` | `AIL_start_sample` |

(同段程式碼另找到 `0x658` → `"AIL_set_sample_file(...)"`、`0x658` 一帶的 `"AIL_set_sample_type(...)"` 等
其餘樣本 API 字串,證實 EXE 內確有完整 AIL digital-sample 呼叫鏈,而非只有驅動殼子。)

### 2. 找到「表從哪來」:一次性 init 載入 `FDOTHER.DAT` 資源 `#31`(`0x1f`)

`[0x53eec]`(上面 `table_ptr`)在啟動 init 區(緊接在 `play_bgm` 之後,`0x26a63`)被設定:

```
0x026a63  push  0x1f                   ; 資源 index = 31
0x026a65  push  [0x53eec]              ; 舊指標(釋放用)
0x026a6b  push  0x1a4d                 ; ->0x51a4d = "FDOTHER.DAT"(字串已確認)
0x026a70  call  0x11fba                ; 通用「載容器資源」函式(與 play_bgm 載 FDMUS.DAT 用同一支)
0x026a78  mov   [0x53eec], eax         ; 存起來
```

同一段 init 另外把 `FDOTHER.DAT` 資源 `#1`、`#2` 分別載進 `[0x53a4d]`、`[0x53a89]`(這兩個非 SFX 用途,
與 doc14 已知的「疑框角/箭頭」推測一致,留待後續確認,不影響本篇結論)。

`[0x53eec]` 被 code 讀取(`push dword ptr [0x53eec]` 系列)共 **58 處**,散布在選單/游標/對話等一般 UI
流程 ── 符合「UI 通用音效池」的角色。

### 3. 戰鬥/攻擊音效:另一張表,同一個檔案,動態 index

戰場動畫更新函式(`0x28ee0` 一帶,單位受擊/攻擊演出)另外從 `FDOTHER.DAT` 動態載入:

```
0x028f1a  mov   eax, [esp+0x50]
0x028f1e  add   eax, 0x21              ; index = 某資料表[unit/attack]+0x21(依攻擊種類變化)
0x028f21  push  eax
0x028f24  push  0x1a4d                 ; "FDOTHER.DAT"
0x028f29  call  0x11fba
0x028f33  mov   [0x5411f], 0
...
0x028f41  movzx eax, [esp+eax-0xc]     ; 第二個動態 index(byte,依攻擊資料)
0x028f49  push  0x1a4d                 ; "FDOTHER.DAT"
0x028f4e  call  0x11fba
0x028f56  mov   [0x5411f], eax
```

`[0x5411f]` 再被 `0x268c7`/`0x26976` 形式的 `play_sfx` 呼叫(32 處 xref,全落在 `0x27000`–`0x2c600`
戰鬥演出程式碼範圍內)。**結論:戰鬥音效(揮擊/受擊等)也走 `FDOTHER.DAT`,但 index 依攻擊/單位資料
動態決定**(不是固定 index),对应「每種招式配自己的音效」設計 ── index 表本身待第 9 輪逐招對照。

### 4. 直接解包驗證:`FDOTHER.DAT` 資源 #31 = 巢狀容器,內容是原始 PCM

```
$ python3 tools/unpack_dat.py --list FDOTHER.DAT | grep '^    31'
    31  0x000fe4f3       31771
```

抽出這段 bytes,開頭即 `LLLLLL` magic → 資源 #31 本身**又是一個 `LLLLLL` 容器**(嵌套,與 doc23
記錄的資源 #7 巢狀容器同一手法),內含 **14 個子樣本**:

| sub# | 長度(bytes) | 特徵 |
|---|---|---|
| 0 | 160 | 極短,值域窄(`0x77`–`0x90`) → 短促 UI 提示音(游標移動量級) |
| 1 | 3322 | 值集中 `0x80` 附近 | 
| 2 | 667 | 短 |
| 3 | 6324 | 最長,值域寬(`0x67`–`0x8f`) → 較長效果音 |
| 4 | 3438 | 值域最寬(`0x58`–`0x90`) |
| 5–6 | 3058, 3058 | 幾乎全 `0x80`(近似靜音段) |
| 7 | 160 | 與 #0 同長度、不同波形 |
| 8 | 2652 | 全 `0x80` 開頭 |
| 9 | 390 | 短、值窄 |
| 10 | 3304 | 全 `0x80` 開頭 |
| 11 | 2442 | 漸變波形 |
| 12 | 2734 | 全 `0x80` 開頭 |
| 13 | 0 | 空(目錄結尾哨兵) |

所有子樣本 byte 值都集中在 **`0x80`(128)附近上下擺動**(unsigned 8-bit PCM 的靜音中心值恰為
128) ── **這就是無壓縮、無檔頭的 8-bit unsigned mono raw PCM**,與 Miles AIL digital sample API
(`AIL_set_sample_address` + `AIL_set_sample_type`)直接吃裸 PCM buffer 的用法完全吻合:不需要 WAV/VOC
檔頭,取樣率/位元深度另由 `AIL_set_sample_type`/`AIL_set_sample_playback_rate` 呼叫端設定(未逐一反組譯
其立即數,取樣率待後輪用實機錄音比對確認,推定 11025Hz ── 1995 年 AIL 遊戲常見預設值)。

## `.DIG` / `SAMPLE.*` 的角色釐清(避免與資料混淆)

| 檔案 | 角色 |
|---|---|
| `*.DIG`(`SBLASTER.DIG` 等) | Miles AIL **數位輸出驅動**(依音效卡選一份載入),純程式碼,不含遊戲音效資料 |
| `SAMPLE.BNK` | 標準 Miles SDK 隨附的 **AdLib 樂器名稱庫**(`ADLIB-` magic + 樂器名字串表),FM 合成器音色定義,與遊戲 SFX 資料無關 |
| `SAMPLE.AD` / `SAMPLE.OPL` | 與 `SAMPLE.BNK` 配套的偏移索引表,同屬 AIL SDK 標準檔案 |
| `FDOTHER.DAT` 資源 `#31`、動態 index(`+0x21`/`[+eax-0xc]`) | **真正的遊戲 SFX 樣本資料**(本篇結論) |

`SAMPLE.BNK/AD/OPL` 三檔的內容(偏移遞增表 + 樂器名字串)是 Miles Sound System 開發套件的**通用隨附
檔**,不含 `FLAME2` 專屬資料 ── 判斷依據:內容與已知其他 Miles/AIL 遊戲的同名檔案結構特徵一致(遞增
uint32 表 + `ADLIB-`/樂器名字串),且遊戲程式碼中未找到任何對 `"SAMPLE.BNK"`/`"SAMPLE.AD"`/`"SAMPLE.OPL"`
字串的參照(僅 `.MDI`/`.DIG` driver 內部會用到,由 AIL library 自己讀取,不是 `FD2.EXE` 遊戲邏輯呼叫)。

## 重製對應(SDL2 / Ebiten)

| 原版 | 現代 |
|---|---|
| `FDOTHER.DAT` #31(巢狀容器,14 個 8-bit unsigned PCM) | 逐個解出轉 WAV(補標準 44-byte RIFF 檔頭,8-bit unsigned mono),或轉 OGG |
| `AIL_init_sample` → `set_sample_address` → `set_sample_loop_count` → `start_sample` | SDL_mixer `Mix_LoadWAV` + `Mix_PlayChannel` |
| 雙 handle(`[0x53ee4]`/`[0x53ee8]`) | 兩個 SDL_mixer channel(允許疊播) |
| 戰鬥音效動態 index(`+0x21`) | 待逐招對照後,做 `attack_id → sfx_index` 對照表 |

## 待辦(後輪)

- [ ] `FDOTHER.DAT` #31 的 14 個子樣本逐一對照「UI 事件」(游標/確認/取消/選單開合等),目前僅
  index 0/1 已知走 `[0x53eec]` 表且用於游標移動一類 UI 呼叫(呼叫端待逐一標注)。
- [ ] 戰鬥動態 index 表(`[esp+0x50]+0x21`、`[esp+eax-0xc]`)還原成「攻擊/招式 → SFX 子樣本」對照。
- [ ] 取樣率/位元深度以 `AIL_set_sample_type`/`playback_rate` 呼叫端立即數反查確認(目前用產業常見值
  11025Hz/8-bit 推定,未逐一反組譯驗證)。
- [ ] `FDOTHER.DAT` #1、#2(同一次 init 載入,`[0x53a4d]`/`[0x53a89]`)用途未確認,與本篇 SFX 無關但
  同批載入,值得一併釐清避免誤判。

## 工具

`tools/le_xref.py`(字串/call xref)+ `tools/unpack_dat.py`(容器解包,含巢狀容器直接可用同一支
`parse_directory` 邏輯手動遞迴)。本輪額外用 capstone(`docker fd2-cap` 已內建)做逐指令交叉驗證 ──
專案既有 `tools/disasm_le.py` 在部分 range 有指令邊界誤判問題(如 `0x25a96` 一帶會錯位),遇到可疑
反組譯結果時改用 capstone 從已知正確的函式起點(如 `play_bgm=0x26777`)線性反組譯校正。
