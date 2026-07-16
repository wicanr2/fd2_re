# 50 — 過場腳本系統:原版指令集 → remake Beat DSL(全 33 關通用)

> 結論整理(2026-07-04,doc47/48/49 三線收束)。使用者戰略:**第一關指令破解後,
> 後續 32 關全部機械可解**——因為所有章節 handler 用同一套原語指令集,差別只在參數。
> 本篇定義 remake 腳本系統如何一比一承接。

> **⭐ 本檔 = 過場 / acting / 走位機制的唯一主檔(鐵則:同主題知識集中一份,禁散落)。**
> 查「過場原語 / acting / 走位 / 面向 / 序章編排」**只讀本檔**;原始逐 beat 轉錄見 `doc47`(附錄性質)。
> 其他檔提到過場機制**一律只引用本檔、不複製內容**。

## 1. 原版過場機制最終結論(全部實證)

三層架構,各層職責與還原狀態:

| 層 | 原版實體 | 還原狀態 |
|---|---|---|
| **編排** | EXE 每章 handler(跳表 0x51d71[章] 戰前 / 0x51de9[章] 戰後),線性呼叫原語 | 序章 0x3231b 已全轉錄(doc47);其餘章可機械抽取(§3) |
| **對白** | FDTXT 章文本,0x15f84(idx) 逐條播 | 35 檔全解+1533 句精校 |
| **演出** | EXE acting 資源目錄 + direct-ID getter，0x1366a(id) 播放 | EXE 靜態 bank 106 entries；ACT99/100 已以 live unit diff 交叉驗證 |
| **走位** | **引擎逐格步進單位**(step 家族、路徑走位 0x13488，及 acting 正常 frame；見 §1.1/§1.2)+ 鏡頭鎖定跟隨 | 機制閉環;remake storyWalks+FollowWalk/acting player 同構 ✓ |

原語指令集(= 所有章節 handler 的「組合語言」):

| 原語 | 語意 | remake 對應 |
|---|---|---|
| `LOADCH` (0x205da) | 載章節地圖+文本(章節變數驅動) | 節點 map/script 欄 |
| `PAN(col,row)` (0x135dd) | 平滑鏡頭平移到格 | beat op:pan |
| `TXT(idx)` (0x15f84) | 播章文本第 idx 條(開框/頭像/翻頁) | beat op:dialog |
| `ACT(id)` (0x1366a) | 演出:批次設單位 pose；正常 frame 每拍走一格，特殊 frame 原地顯示 | beat op:act(acting_frames) |
| `SPAWN(g)` (0x10b4e) | 群組 g 登場 | beat op:spawn |
| `JOIN(char)` (0x112a5) | 角色入隊伍名冊 | beat op:join |
| `BGM(track)` (0x25977) | 配樂切換/停止 | beat op:bgm |
| **走位 STEP/路徑** (step家族 + 0x13488) | 引擎逐格步進單位(方向陣列 0下1左2上3右);詳見 §1.1 | beat op:walk |
| `PALFADE` (0x1f525) | 整幕 palette 淡入 | beat op:fade |
| `DELAY(ms)` (0x375b2) | 延遲 | beat op:delay |
| `DEACTIVATE_UNIT(slot)` (0x32975) | `unit[slot]+5 = 1`；bit0=1 是死亡／隱藏／未啟用，用於劇情退場 | beat op:deactivate_unit |
| `SPAWN_INTRO(g)` (0x32999) | 內呼叫 0x10b4e append group，再做 12-step reveal/present | beat op:spawn_intro |
| `RESET_POSE` (0x134e4) | 所有 materialized units pose=0，再 delay 20ms | beat op:reset_pose |
| `FOCUS_UNIT(slot)` (0x12d7b→0x12cea) | 讀 unit X/Y，先 X 後 Y 逐格移動游標；13×8 視窗在 X=2..10、Y=2..5 安全帶外才捲圖 | beat op:focus_unit；runtime 已照四個 step 函式 lower |
| `SYNC_PARTY` (0x11506) | 戰後以角色 ID 將 runtime battle unit 回寫 persistent roster，清暫態並恢復 HP/MP（§3.2） | beat op:sync_party；`partyRoster` 跨 battle/save 保留 |
| `GRANT_ITEM(item)` (0x1c220→0x1bb8c) | 依 runtime slot 順序找第一個我方且 8 格物品欄未滿的角色，放入 item ID | beat op:grant_item；角色 `Inventory` 隨 sync/save 保留 |

### 1.1 走位機制(step 家族 + 路徑走位;2026-07-05 釘死)

原版單位在格盤移動 = **引擎逐格步進**,由 4 方向 step 家族驅動:

| 函式 | pose 寫死(+3) | 方向 | 位移 |
|---|---|---|---|
| `0x12eaa` | 0 | 下 | Y+1 |
| `0x1300d` | 1 | 左 | X−1 |
| `0x13185` | 2 | 上 | Y−1 |
| `0x13315` | 3 | 右 | X+1 |

每個 step:讀單位、寫 pose(+3)=方向、增減 X(+0)/Y(+1) 一格、設 tick(+4) 次格滑動、同步捲鏡頭。

**通用走位原語 `0x13488(單位idx, 方向陣列指標, 步數)`**:迴圈讀方向陣列(每 byte 0下/1左/2上/3右)逐格呼叫對應 step = 任意路徑走位本體(全 EXE 2 caller:0x14ec4/0x18a52)。

**繪製公式 `0x127e0`**:單位畫在 `格 + tick × f(pose)`(pose=方向向量,tick 遞減→滑到定格)= 次格平滑內插。

**單位結構(0x50B/槽,`[0x53a45]+idx×0x50`)**:+0=X格、+1=Y格、+3=pose(=方向)、+4=tick倒數、+5=狀態旗標(bit0=1 死亡／隱藏，0 有效存活；bit7=已行動)、+8=角色ID。

**序章王座走位(handler 0x3231b)**:`0x13185` 直接 ×15 → 對話#0 → ×13 → 對話#1(「全上」特例,不經 0x13488)。停位(對原版兩截圖 + FDFIELD 守衛地標三角測量):第一次對話 **(8,21)**(守衛 (5,21)/(12,21) 左右緊鄰索爾)、最終 **(8,8)**(王前 3 格「最跟前」)。詳細轉錄見 doc47 §11。

**面向規則(所有劇本通用)**:dir/pose 預設 **0(下=面向玩家)**;**FDFIELD 不存面向**(出場資源每筆恰 6B=X,Y,portrait)→ 面向來自 zero-init。非 0 僅兩種:①走位者面向移動方向;②劇情主角對看。背景 NPC/守衛永遠 dir=0。

**remake 對映**:`storyWalkJob`(from→to 沿格線先長軸後短軸)+ `FollowWalk` 鏡頭跟隨 = 同構;`Actor.Dir` 預設 0;走完面向 `finalDir`。

### 1.2 acting 播放器 `0x1366a`(姿態 / 格位移幀動畫)

`0x1366a(id)` 播一筆 acting 資源:`[0x627d8+id*4]` 取資源,格式 `[幀數]+每幀{(拍數,N)+N×(單位idx,姿態)}`,逐 tick 寫 unit[+3]=pose、[+4]=tick,用繪製公式 0x127e0 + 重繪 0x11cac 畫出。

**[更正定論 2026-07-15,實機寫入中斷 + 播放器尾段]acting 有兩種模式；正常模式會搬格子**。
先前反組譯在 `0x13918 call 0x2c9ec` 處截斷，漏讀了正常模式尾段 `0x13930..0x13960`：

- 格式仍是 `u8 幀數 + 幀×{ u8 拍數, u8 N, N×(u8 單位idx, u8 pose) }`，**資料切分本來正確**。
- **bit7=0 正常模式**：低 7 位的每一拍都寫 `+3=pose`、以 `+4=tick(1..6)` 顯示滑動，然後在
  `0x13937..0x13949` 依 pose 寫邏輯格：`0:Y+1, 1:X-1, 2:Y-1, 3:X+1`。所以**低7位拍數 = 該 frame 的格數**。
- **bit7=1 特殊顯示**(`0x137c7/0x1370b`)：只寫 `+3`、重繪，**不搬格子**；低7位是原地顯示/節奏次數。

實機佐證：草地末段對 slot3 X 設保護模式寫入中斷，捕到 `4→5`；堆疊回溯到正常模式
`0x1391e call 0x2c9ec`。因此 `decode_acting` 先前「acting 只設面向」的輸出語意錯，正確模型是：
**acting 的正常 frame = `pose` 指定方向、`beat` 指定走幾格；特殊 frame = 原地姿態。**

`+4(tick)`仍是每一格內的平滑內插；step 家族與 `0x13488` 仍是另一套顯式逐格控制器，但不再是唯一走位來源。

**草地走位已由 direct bank 解完（2026-07-15）**：舊 decoder 錯移 48 entries，才會把
ACT101..105 誤讀成 slots16/17。正確 direct resources 明確操作草地角色：ACT101 slot4 left×3；
ACT102 slot4 left×2/up×1/left×1；ACT103 special slot4；ACT104 special slot3；ACT105 讓 slot3
right×2/down×1/right×6，並讓 slot4 down×1/right×4。這與實機 slot4 `(13,47)→(7,46)` 快照
一致，不存在「對話 parser 額外觸發未知 ACT」或「重填 slots16/17」的未解來源。

**acting 靜態來源與 live table 規則（2026-07-15 更正）**：raw bytes 不在外部 DAT；
`FD2.EXE file+0x565d8` 是 106×u32 offset directory，entry 範圍 `0..105`，資料位址為
`file+0x53e00+offset`。舊 `0x207718`、高 ID 74 筆與 `id−48` window 是錯 context dump，全部撤回。
ACT99 的 normal-core trace 已證 getter immediate=`0x2077d8`、`table[99]=0x208493`、
bytes=`01 06 01 02 02`，所以 id99 直接解為 slot2 向上六格。ACT100 隨後以同一 getter 及 handler
id100 命中，slot2 實際 Y `8→18`。getter 是全域 direct-ID table，不存在 chapter-local window。

## 2. remake Beat DSL(campaign 節點新形態)

story 節點升級為 **cutscene 節點**:`beats:[{op,args…}]` 順序執行,一比一對映原語:

```json
{ "type": "cutscene", "map": "assets/maps/map32", "script": "assets/story/ch00_palace.json",
  "beats": [
    { "op": "walk",   "fig": 0, "y": 21, "follow": true },
    { "op": "dialog", "line": 0 },
    { "op": "walk",   "fig": 0, "y": 8,  "follow": true },
    { "op": "dialog", "line": 1, "count": 18 }
  ] }
```
> ⚠ 範例依原版截圖 + FDFIELD 守衛地標實測修正:第一次對話停 **(8,21)**、最終停 **(8,8)**;
> 兩段之間是 **walk**(引擎逐格步進 + follow 鏡頭跟隨,§1.1),**不是** scroll/pan——早期「scroll rows」
> 與「walk 到 (8,8) 一次」都已撤回(§1.1 停位、doc47 §11)。

原則:
- beats 序列直接照抄章 handler 轉錄(doc47 §3/§7 即 ch1 的 beats 來源),**參數用 handler 實值/dosbox 實測,不外推**。
- `walk`=remake 為已量測／顯式路徑保留的高階走位拍(可選 follow)；`act`=原版 acting frame，
  正常 frame 可含位移、special frame 才是原地姿態。
- 對白×演出**交錯**天然支援(beats 是平面序列,不再「一幕一段」)。
- 舊 story 節點(Lines/Scene/Actors)保留相容,逐步遷移。

### 2.1 remake 的 acting frame 轉錄(2026-07-15)

`Beat` 新增 `acting_frames`，不收錄原始 bytes，只轉錄已解出的行為；可直接對應 `0x1366a` 資源：

```json
{ "op": "act", "acting_frames": [
  { "beats": 3, "units": [{"fig": 0, "pose": 3}] },
  { "beats": 8, "special": true, "units": [{"fig": 4, "pose": 1}] }
] }
```

- 第一幀（`special` 省略／false）= 索爾面右、每 beat 走一格，共右移 3 格；每格有 7 tick 內插。
- 第二幀（`special:true`）= 亞雷斯原地面左，維持 8 tick。
- decoded handler 的 `units[]` 必須寫 `slot`（原版 FDFIELD/unit-array index）；同一 Fig 可有多個
  守衛，若以 Fig 尋找必然可能移錯第一人。`loadch` adapter 會保留完整 roster 順序（連不可見／死亡
  slot 也不濾掉）來保證 `slot` 對位。`fig` 只保留給沒有原版 roster 的舊手寫場景相容，**不得**用來
  轉錄已 decode 的 `0x1366a` 資源。
- 主要 live regression 是 `0x32343 / ACT99`：slot2 `up×6`（Y42→36），再接兩段
  `scroll_step(slot2)` 的 `up×15`、`up×13`（Y36→21→8；每格七 ticks）。
- `0x323f5 / ACT100` 是 slot2 `down×10`（Y8→18）；`0x32461 / ACT102` 則是 slot4
  `left×2 → up×1 → left×1`。所有 decoded target 都用原始 slot，不以重複 Fig 猜角色。

**acting resource library（2026-07-15）**：`assets/cutscenes/acting/map32.json`（歷史檔名）轉錄
EXE 靜態 bank 的 **106** entries（`0..105`）為 editable `{beats,special,units:[{slot,pose}]}`，
沒有原始 pointer。其 key 就是 getter 的 direct resource ID；handler source address 仍作 call-site 稽核。
`HandlerBinding.acting_resources` 的 mapping 必須以 source address 限制；不得再以舊 high-ID window
或 `id−48` 偷套另一個資源到同一 call-site。
binding 也完整 transcription 三次 `loadch`（32→map32/ch00_palace，31→map31/ch00_meadow，
0→map0/ch01）。direct resource 90..105 的最大 slot 僅 4，舊 resource99/100=slot61/60 已撤回。
`LoadCHState.slot_count` 現為必填，compiler 會拒絕任何超過當前已載入 roster projection 的 acting，
runtime 也驗證實際 roster 長度，避免沉默略過。map32 的 99..105（slot ≤4）均可安全播放；
完整轉錄；原版 DOSBox 的 map32 `unitcount=21` 已在多個 snapshot 相同，故用
`tools/export_runtime_roster.py` 從 FDFIELD export 機械生成 21-slot
`cutscenes/rosters/map32_runtime.json`，不把 battle export 的後 9 個 placeholder 混進原版 slot
identity。

**外部 overlay／資源載入排除（2026-07-15，靜態＋DOSBox-X I/O trace）**：曾懷疑 acting 或 handler
另藏於 `.DAT`／`FD2.TMP`，再被載入 text/code 區；此路已查明並排除。FD2.EXE 的 LE object #1
`0x10000..0x4ebd8` flags=`0x2045` 是 executable code；object #2/#3 flags=`0x2043` 是 RW data。
acting directory `0x627d8` 位於 **object #3 的 file-backed initialized data**（file+`0x565d8`），
payload bank 同在該 object（file+`0x53e00`），不是 runtime copy。getter `0x4e7f8` 唯一操作是
`mov eax,[0x627d8+id*4]`，沒有呼叫檔案 loader。

通用 loader `0x111ba` 則是 `fopen("rb") → seek(index*4+6) → read(start/end) → malloc(length)
→ seek(start) → read(payload) → fclose`；destination 是 heap，呼叫端只把 pointer 存進 data globals，
沒有寫入 object #1。DOSBox-X `-log-fileio` 從新遊戲跑至 map32 草地對話的實測序列為：載
`FDTXT_033`、`FDFIELD` entries 96/97/98、`FDSHAP` entries 64/65、`FDICON.B24`，其後演出／對話只按
speaker 開 `DATO.DAT`（另有 BGM 的 `FDMUS.DAT`）；**沒有** FDOTHER／ANI／FIGANI／FD2.TMP read
發生在 acting 播放期。`FD2.TMP` 在同次 trace 只有建立並寫入 207360 bytes，沒有 read-back。

外部資料裡唯一可稱為「事件資料」的是 FDFIELD 的 turn/event/group metadata；event handler code
仍由 EXE `0x51b91[event_id]` 跳表選取。FDTXT 是 editable 對話來源，但不含 staging opcode。
所以 remake 的正確拆分仍是：handler／acting 從 EXE 機械轉成 editable script，FDTXT 提供文字，
FDFIELD 提供地圖 roster/group/turn-event 資料；不要再尋找不存在的 external C-handler overlay。

**序章 ACT inventory（2026-07-15）**：`ch00_pre` 共 20 個 `0x1366a` call，現已 **20/20 完整解碼**。
加上 `scroll_step`、`focus_unit` 等 native primitive 後，`ch00_pre` 整體也已完整 lower，compiler
regression 要求 **0 unresolved issues**。
王座／草地階段是 `0x32343/0x323f5/
0x32426/0x32461/0x3249c/0x324d7/0x3251c`（ID 99–105）；map31 是
`0x3255f/0x3259a/0x325d5/0x32657/0x326d7/0x32712/0x3274d/0x32788/0x327d9`
（source ID 90–98）。direct entries 的 slots 都在 0–4；舊 74-entry dump 不得再作為 slot projection
依據。map0 四 call 也使用同一 direct-ID bank：

- `0x3283a→ACT0`：normal `up×6` 同時作用 slots 0–3；再由四個 special frame 讓 slot0
  `left→right→left→down`（8/8/8/4 ticks）。
- `0x328a5→ACT1`：slots 4–7 全部 `down×1`。
- `0x328c5→ACT2`：slots 8–11 依三個 normal frame 位移，再 special 4 ticks 定場。
- `0x3290d→ACT5`：**slot9 `down×4`**；slot9 是 group2 第二名海盜，不是悠妮。

runtime slot identity 亦閉環：JOIN 首次順序 `0,9,4,30` = 索爾、悠妮、亞雷斯、蓋亞，故 party
先建 slots 0–3；SPAWN group1 append 四海盜成 slots 4–7，group2 再 append 成 slots 8–11。
map0 `loadch` binding 另保存 editable `party_order:[0,9,4,30]`：正常 campaign 用它核對實際 JOIN
chronology，direct/debug replay 沒有歷史時則以它重建正確 slots，不回退到 battle UI 的 authored order。
`map0_slots.bin` 直接顯示 ACT0 後四人為 `(7,14)/(10,15)/(8,16)/(11,17)`，回推前一幀部署格正是
`(7,20)/(10,21)/(8,22)/(11,23)`。

**map31 runtime-array 反證（2026-07-15）**：用原版 headless DOSBox-X 從開場逐秒送 Enter；第
100/120 個確認點截圖均已是 map31 密林（第 140 個則已進 map0），故場景時點不含糊。在第 100 點以
`MEMDUMPBIN DS 24B2F0 1900` 首次取得的是 map32 stale allocation，不能當 map31 roster。隨後從
spawn writer `0x10c50` 的 runtime-relocated 指令讀出 `[*0x19CA45]`，在同一密林 checkpoint 的值為
**`0x2499EC`**；在**該 checkpoint** dump 80×0x50 後只有 slot 0–4 是有效人物：索爾、亞雷斯、**商店店員的
portrait／scene-actor ID 75（不是我方名冊 charID，亦不是第二個悠妮）**、蓋亞、悠妮。序章文本與
`JOIN` 呼叫均只支持 charID 9 為悠妮；她在這幕是先倒地、後由演出喚醒的角色，不能把同格的
商店店員 portrait 75 猜成她。75 不在 0–31 的可入隊角色表，故不得進入 party/JOIN adapter。
slot 5 起在此時點尚不是 unit 結構。acting player `0x137dd/0x13891/0x13975` 也明確以同一
`[*0x53a45]+slot×0x50` 取目標。direct ACT90..98 的 max slot 是 `1,1,0,3,4,3,3,1,4`：前三筆
在 group1 後的 2 actors、ACT93 在 group3 後的 4 actors、後五筆在 group5 後的 5 actors，全部合法。
舊 slots8/25–71 來自錯位 table，不能再據此補 72-slot roster。map0 0/1/2/5 亦來自同一 direct bank。

**SPAWN 資料流改定（2026-07-15，靜態反組譯）**：不再以鍵盤節奏撞 `decode_acting` entry。
`0x10b4e(group)` 掃 `FDFIELD` unit record（`[0x53a55]+0x83+k×0x1a`，`+0x15=group`），每筆命中都 call
`0x10c50`；後者以 **`[0x53beb]`** 為 destination slot，寫完整 `0x50`-byte unit，最後
`inc [0x53beb]`（`0x10c69..0x10c81`、`0x10ffd..0x1100b`）。原版語意因此是「依 FDFIELD 原順序
**append** matching group 到 runtime unit array」，不是先建完整 roster 再把 `OnField` 打開。
map31 的 handler 只 spawn group 1、3、5，而這三組在 FDFIELD 正好是索爾／亞雷斯、商店店員／蓋亞、
悠妮五筆；所以 checkpoint 的 runtime slots 0–4 正好成立。這是可重現的靜態＋實機交叉證據，並且推翻
舊的「保留所有 FDFIELD slot」adapter。remake 已依此改為：可選 persistent party 先 materialize，
再 materialize group0，後續 spawn append group；map32 runtime roster 的 21 筆都為 group0，故既有
map32 slot fixtures 不受影響。map0 則由 `party_scenario` + JOIN chronology 先建立四人，讓 ACT0/1/2/5
與 group1/2 append 的 slot identity 完整成立。

**map31 ACT(90..98) 最終更正（2026-07-15）**：direct entries 的 max slot 依序為
`1,1,0,3,4,3,3,1,4`。handler spawn group1/3/5 後 active count 依序為 2→4→5，因此每個 call
都只操作當時已 materialize 的 actor。舊 `timing_only:true` 會錯誤清空真動作，已全部移除。

**entry breakpoint 的正確界線（2026-07-15）**：不要把 `0x1366a` 的 runtime 位址
`0x1C966A` 當成 handler 的 relocation base。normal-core 在一個 ACT(102) 函式入口擷取的 stack
第一個 return address 是 **`0x1E8466`**；它對應該次載入狀態下原檔 `0x32461 call 0x1366a` 的
return `0x32466`，故當次 handler base 為 **`0x1B6000`**。ACT function 本體在另一段 runtime
image（`0x1366a→0x1C966A`），兩者不可混算。把這個樣本外推成 map31 `ACT(95)` 的
`0x1E8712`、並在 map31 `LOADCH` 前預設 breakpoint 的實驗**沒有命中且流程已越過 ACT(95)**；
因此 handler code base 可能隨載入階段重配置，尚不可當跨 `LOADCH` 的通用公式。這條逐鍵撞 entry 的
路徑已由上面的 table compare + SPAWN/renderer 靜態資料流取代；其失敗位址不能用來建立 roster 結論。
但在真正 entry `0158:1c966a` 設單一 normal-core code breakpoint、逐次讀 `SS:ESP` 的事件式 trace
是有效方法；ACT99 已以它取得 `0x2077d8/table[99]` 的直接證據。淘汰的是「猜下一個 caller
runtime 位址」與錯 context table dump，不是 entry breakpoint 本身。
- 舊 `poses`／`pose_frames` 仍可用於尚未轉錄的近似場景，但新的原版 acting 不得再降級成它。

> **系統界線(2026-07-04,doc52):本 DSL 只承接「戰前/戰後過場編排」(handler 0x3231b 族,系統 A)。
> 「戰鬥中回合事件對話」(哈諾第 3~4 回合等)是另一套系統 B(跳表 0x51b19 / battle_events.json,
> doc26),由 `battle.Scenario.Events` + `Fire(on_turn_end)` 承接,邊打邊觸發,不進 cutscene beats。
> 兩套並存不混——把戰鬥對白塞進開場一次播完是先前的架構錯誤。**

## 3. 全戰役 handler 機械破解管線

1. **`tools/dump_chapter_beats.py` + `tools/export_handler_scripts.py`(2026-07-15)**：走
   跳表 `0x51d71/0x51de9` 的 30 組 pre/post entry，對每支 handler 做已驗證的
   push/call 配對抽取，再正規化為 **60 個可編輯檔案**
   `remake/assets/cutscenes/handlers/chNN_{pre,post}.json` 與 manifest。
   每個 beat 都保留 EXE 原始 call-site (`source.addr`) 作稽核證據；原語呼叫則轉成
   `loadch/pan/dialog/act/spawn/spawn_intro/deactivate_unit/reset_pose/focus_unit/join/bgm/scroll_step/palette_fade/delay` 等可讀 op。
   尚未證實的 native call **保留為 `unknown`**，不可靜默丟失或猜譯。ch0(序章)仍逐項
   對照 doc47 §7；其中迴圈由 parser 自動辨識，精確匯成
   `scroll_step(unit_slot:2,repeat:15)` 與 `repeat:13`。**slot 不是方向**：它是 0x13185
   跟隨／捲動畫面所跟的原版 unit slot，必須由 binding 指向 remake actor。2026-07-16 修正
   exporter 的 handler 邊界假設：Watcom 會讓多個 entry `jmp` 到下一個 entry 之外、甚至較低位址的
   deduplicated shared tail；現在只追顯式 CFG edge（不追普通 fallthrough），local body 依位址排序後
   再依發現順序接 external blocks。60 支 handler 因而由 **624 增至 701 個 top-level beats**，
   `ch01_pre` 等過去被截掉的最後 dialog/focus 不再消失；兩個合成 CFG regression 固定跨下一 entry
   與 backwards tail 的順序。
2. 編譯層：`campaign.CompileHandlerScript` 將 editable handler script + 已驗證的章文本／
   FDFIELD roster mapping → map-specific runtime `Beat`。它能直接 lower `delay` 與已證實的
   `bgm`（`track=-1` → `bgm_stop`）；`loadch` 則要求同一個 address-keyed override 同時提供
   **原版零起算 resource chapter + map + roster + editable story script**，並 lower 成原子的
   `Beat.LoadCH`。BeatRunner 先驗證名冊／文本，再切圖；若 binding 有 `party_scenario`，先依 JOIN
   chronology 建立 persistent party slots，接著才按 FDFIELD group materialize/append 到 `storyActors`；
   缺任何一項就 issue／runtime fail-closed，絕不把 `0x205da` 偷降級為 map-only
   或 no-op。handler 檔名是**零起算 jump-table index**：`ch05_pre` 在 table index 5，driver 呼叫時
   global chapter 也是 5，因此 `0x33155` 選的是 map5 / FDTXT_006 / `ch06.json`；舊 binding 把它
   誤接到玩家第五章 map4/FDTXT_005，已撤下 campaign consumer。shared-tail 修正也證明它不是
   「唯一 loadch beat」，後面還有 `dialog #0 @0x3320c → focus slot0 @0x33142`；在第六章 persistent
   party JOIN chronology 尚未逐項釘死前保持 compile issue，不能假裝 complete。
　　`pan/dialog/act` 一律要求**以 `source.addr` 鍵控的**
　　顯式 mapper，分別避免猜 grid→pixel、FDTXT idx→譯文行、acting id→角色。`spawn`
　　現已由 loadch roster 的 FDFIELD group 直接 lower；`join` 只接受原版 0–31 的我方名冊
　　charID，並保存成跨關 membership；`palette_fade` lower 為 `fade(out:false)`；`scroll_step`
　　保留 slot/repeat/每格七 ticks，`focus_unit` 保留原版逐格游標安全帶。真正未知或動態參數仍產生
　　帶 source address 的 compile issue，不能假裝成可執行效果。

**SPAWN runtime adapter（2026-07-15）**：`LOADCH` 保留 editable FDFIELD records；可選 persistent
party 先依 JOIN order materialize，接著才 materialize group0。`Beat.spawn(group)` 按原 FDFIELD order
append 尚未 materialize 的同 group records，對應
`0x10b4e→0x10c50` 的 `unit_count` 寫入方式。ch00 binding 已能 lower map31 三個 call-site
`0x32555/0x32610/0x3269c`。這解釋已驗證的 5-slot map31 checkpoint；ACT(90–98) 現直接操作
slots 0–4，完整保留 frame movement/special timing。
`0x32999(group)` 已由完整函式本體重反組譯確認為同一個 append constructor 加 12-step present loop；
BeatRunner 的 `spawn_intro` 先 materialize group、保留 12 個顯示 step，再進下一個 ACT。`0x32975(slot)`
則明確 lower 成 `deactivate_unit`：它寫 bit0=1，是死亡／隱藏／未啟用的劇情退場，不是 camera reveal。
`JOIN` 已可 lower 原版 0–31 player charID 並保存 party membership；NPC portrait（例如商店店員 75）
一律拒絕。
   `remake/assets/cutscenes/bindings/` 的 `HandlerBinding` 則是這個顯式 mapper 的可編輯
   JSON 表示；其 override 以 call-site 位址為 key。`ch00_pre.json` 已收入已驗證的王座／草地
   全七個 PAN、19 個 FDTXT call、全部 20 個 ACT、spawn/spawn_intro、activate/reset/redraw、
   player JOIN、兩段 scroll_step 與尾端 focus_unit。完整 ch00 compiler regression 為 **0 issues**。

**ch00 對白群組已具體轉錄（2026-07-15）**：原版一個 `0x15f84` 呼叫可包含多名說話者，故
`HandlerDialog.lines` 可展開為多個 runtime dialog beat。binding 以 source 位址保存
`0x32382 → 王座 line 0–5`、`0x323cb → line 6–18`，草地則是
`0x3244d → line 0–4`、`0x32488 → line 5–8`、`0x324c3 → line 9`、
`0x324fe → line 10–21`。這些群組由 FDTXT_033 offset table 直接解碼，不是按譯文段落猜切；
草地兩段實際座標與時點見 doc55 §2.1。
同一 binding 現亦透過 count-aligned index 接上 FDTXT_032 的十個 call-site（37 editable lines）與
FDTXT_001 的三個 call-site（19 editable lines）；compiler regression 逐 source address 驗證每組行數。

**全戰役台詞索引資料層（2026-07-15）**：`tools/export_story_index_map.py` 會讀原始 FDTXT
offset table，僅在一份原始資源的「logical utterance」總數與一份 `assets/story/*.json` 的
flattened lines **完全一致**時，產生 `remake/assets/cutscenes/dialogue-index/count-aligned.json`。
目前有 **73** 個 handler dialog call-site context 可機械映射（60 份 handler skeleton 全量生成）；另有
**73** 個 call-site 因 source context／唯一 mapping 未證實而只列 diagnostics、不得猜補。映射 key 是
`source_dat + script + string_index`：FDTXT_032/033 在不同章／序章 context 有重用，不能只以
FDTXT index 當全域 key。`campaign.LoadStoryIndexMap` 會驗證每個映射的全部計數、連續 line
range 與 context，再提供只讀 `Lookup`；此層尚不越權把跨 scene 原字串強行 lower 成 runtime
dialog beats。

`HandlerBinding` 可用 `story_index_map` + 按 `source.addr` 的 `dialogue_contexts` 接上這份
資料；context 明列 `source_dat` 與 story-relative `script`，所以同 FDTXT 重用不會串場。若同一
位址另有手寫 `dialog` override（如 ch00 草地的 `upper`），override 優先；如果原字串跨 scene，
compiler 保持 issue，等待 scene-loading adapter，不能偷當成同一 scene 的 dialog。ch01_pre 是第一個
只靠嚴格索引映射產生三組對白的 binding fixture。compiler lower 後的 runtime `Beat` 仍保留
`script/scene/scene_index` context；BeatRunner 會以 `scene_index`（可處理 null/reused label）載入那個
editable scene，不能回退到 enclosing Node 的 lines 而播錯 `loadch` context。這只解決已編譯 dialog 的
文字選擇；handler 整體仍須所有 map/roster/acting 原語完成 binding 才能宣告可忠實播放。
3. 引擎 BeatRunner：依序執行已證實的 runtime beats
   (pan/dialog/walk/act/spawn/spawn_intro/deactivate_unit/reset_pose/redraw/join/bgm/fade/delay)。其 `acting_frames` 已可精確播放已
   解的 0x1366a 格式；handler 腳本不直接把 EXE 位址交給引擎。
4. 驗收：每章過場對照 DOSBox 錄影（規則 65，對 reference 不對內部訊號），並以 Go
   loader test 驗證 60 份腳本全可讀取、每一筆均帶 source address。

### 3.1 原語覆蓋率(全 30 章,2026-07-04)

舊 raw dump 共 629 beats（含 `loadch_var` 這類非-call 記錄）中，重分類後已知原語 496 筆、未知
133 筆（78.9%）。新版 editable 匯出將 `loadch_var + loadch_call` 合併為一個 `loadch`，
因此要以各檔 `diagnostics.unknown_ops` 和 `_manifest.json` 計數，而不可拿兩種格式的
beat 總數直接相比。2026-07-15 branch 結構化前全量匯出為 **626 個 flat editable beats**；完整辨識 `0x11506`
並重新生成 24 個 post handler 後，保留的 `unknown` calls 已由 **133 降至 109**；再定案
`0x1c220` 的兩個 caller 後降至 **107**。5 個 handler
是已驗證的空 handler，仍保留檔案與 handler metadata。
未知原語 28 種位址，集中在**戰後(post)handler**（戰役流程控制／中場
銜接族，跟序章那種純過場敘事不同族）。逐一淺層反組譯定性（前 40 條指令，看讀寫哪些已知
變數／呼叫哪些已知函式）：

| 位址 | 次數 | 淺層定性(證據) |
|---|---|---|
| `0x11506` | 24 | **戰後 runtime→persistent roster 同步**（完整 body 已驗，見 §3.2）：雙迴圈以 `+8` charID 配對 battle `[0x53a45]` 與 persistent `[0x53bf7]`，複製完整 `0x50`-byte record、清戰場暫態、復原存活者 HP 與全員 MP，最後重算裝備衍生值。ID 0 特例是 `unit_inactive(runtime_idx) != 0` 時跳過配對。 |
| `0x233c6` | 15 | **批次寫入單位陣列 X/Y 座標+初始 pose**:迴圈對 `unitbase+idx*0x50` 寫 `+0`(從 edi 陣列讀 X)、`+1`(從 ebp 陣列讀 Y)、`+3`(<4 的小常數,疑初始 pose)。疑是「roster/FDFIELD own 展開寫入戰場陣列」的初始化實作,呼應 doc47/48 單位結構 `+0=X,+1=Y,+3=pose` 定案。 |
| `0x24b4d` | 15 | **畫面過渡效果**:push 鏡頭 `[0x53aa9]/[0x53aad]` 呼叫 `0x11eee`(地形重繪)+`0x11cac`(主重繪)+迴圈呼叫 `0x11eb0`(present)+`DELAY(20ms)`。與 acting bit7 特殊模式分支(doc47 §9)看到的同一組呼叫序列相同,疑是該分支背後共用的「reveal/漸現」子程序。 |
| `0x11df2` | 12 | **VGA 調色盤處理**(**推翻 team-lead「疑 0x11cac 同族」的猜測**):操作 `[0x53a65]`(新變數,調色盤資料表?)+呼叫 `0x37795`(push 常數 `0x3c8`/`0x3c9`——VGA DAC 索引/資料 I/O port 位址),跟 `0x11cac`(畫面重繪)不同族,是獨立的調色盤/淡變數值計算函式。 |

其餘 24 種(次數 1~8)未逐一反組譯,清單見 `docs/data/chapter_beats/_stats.json`。

### 3.2 戰後 `0x11506`：runtime→persistent roster 同步（2026-07-15，完整 body 已驗）

`0x11506` 出現在 **24 個 post/victory handler caller**。它不是 roster 查詢；每次戰後會外層掃
runtime battle array `[0x53a45]`（`0..[0x53beb)`），內層掃 persistent player roster `[0x53bf7]`
（`0..[0x53bfb)`），以 unit `+8` 的角色 ID 配對。ID 0 另呼叫 `unit_inactive(runtime_idx)`；反組譯的
精確分支是 **回傳非零（bit0=inactive/dead）便跳回內層迴圈、不做 copy**，只有回傳零的有效存活索爾才落入 copy。

對每一個配對，原版依序：

- 以 `0x373c4` 把完整 **`0x50` bytes 從 runtime unit 複製到 persistent unit**；方向不可反過來。
- 將 persistent `+0x22..+0x27` **六 bytes 清零**，並把 `+5` state flags 收斂為 bit0（inactive/dead）而不帶走
  戰場 path／行動等 transient state。
- 若 bit0=0（active/alive），將 HP current `+0x40` 回填為 HP max `+0x42`；bit0=1 的陣亡／inactive 單位保留其零 HP。
  無論存活與否，都將 MP current `+0x44` 回填為 MP max `+0x46`。
- 呼叫 `0x1145a(persistent_index)`：由 persistent record 的 base 值（`+0x37/+0x39/+0x3e`）起算，逐一
  累加已裝備欄位（`+0x0a` 起、bit `0x40`）所指 item 的數值，寫回 `+0x48/+0x4a/+0x4c/+0x4e`。
  故此 call 是**裝備衍生能力值重算**，必須在 copy/清理後做，不能把舊戰場快取直接跨關沿用。

remake 的 first complete consumer 是 `assets/cutscenes/handlers/ch00_post.json`：原版 `FDTXT_001` #9
展開為 13 句 editable `dialog` 後，接 `sync_party`，再 `set_chapter(1)`；
`assets/cutscenes/bindings/ch00_post.json` 供 `campaign_full.json:story_ch02` 載入。runtime 將快照存入
`Game.partyRoster`，下一張 battle/cutscene materialize 玩家時按 stable `Fig`/charID 覆蓋可持久的
能力、HP、MP、EXP 與 spells，但保留新場景的部署座標／group／on-field 狀態；`saveData.PartyRoster`
與 `Chapter` 一併 JSON serialize/restore。因此 battle→post handler→下一章／讀檔 的持久資料鏈已接通，
而不是只在當前戰場記憶體做畫面效果。remake 目前會同步所有 `JOIN` 成員（包含 ID 0），因為尚未
重製原版 ID 0 可能使用的另一條持久化路徑；這是刻意標記的 projection 差異，不能宣稱該特例已 1:1。

### 3.3 戰後獎勵 `0x1c220(item_id)`：第一個有空位的我方角色（2026-07-15）

完整 body `0x1c220..0x1c268` 只有一個參數。它由 runtime slot 0 起掃到 `[0x53beb)`，只處理
unit `+6 == 2`（原版我方 camp），並呼叫 `0x1bb8c(unit_idx,item_id)`。後者逐一檢查
unit `+0x0a` 起的 **8 個 2-byte inventory slots**：slot 第一 byte 的 bit7 set 代表可用空欄；命中後
把第一 byte 清零、把 item ID 寫到第二 byte並回傳 1。八格都沒有空位則回傳 -1，外層繼續找下一個
我方角色；所有我方皆滿時函式靜默結束。

全 EXE 只有兩個 caller，兩者都在 post handler：

- ch01 post `0x22f9f`：item `0xC6`（198，力量藥水）。
- ch20 post `0x24224`：item `0x64`（100，天空之鑰）。

remake lower 為 editable `grant_item(item_id)`；`battle.Unit.Inventory` 保存 item IDs、容量固定 8，
`sync_party` 深複製到 persistent roster，JSON save/load 亦會保留。這比原先只有名稱字串的全隊商店
購物暫存更接近原版角色物品欄，也替後續裝備／消耗品系統留下可資料化的 stable ID。

注意 ch01 post 尚不能因此直接宣稱 complete：FDTXT_002 的 61 utterances 已由 §3.5 補齊並解除
全部 5 個 post-dialog binding，但 pan×2 / ACT14..16 / SPAWN4 的 post-battle roster context 仍有
**6 個 compile issues**。存活分支已由 §3.4 保存，但在這些場景 binding 補齊前仍不得接進 campaign
假裝可播放。

### 3.4 `ch01_post` inactive diamond：structured `if any_unit_inactive`（2026-07-16 更正）

原版 `0x22f44` 從 slot 5 起，`0x22f52 cmp edx,0xb` 限定掃 slots **5..10**；
`0x22f61 test byte [unit+5],1` 累積 inactive/dead bit，最後 `0x22f71 jne 0x22fa9` 分流：

- 任一村民死亡／inactive：只播 FDTXT #7（call-site `0x22fc8`），不給獎勵。
- 六名村民全部 active/alive：播 #6（`0x22f92`），再 `grant_item(0xC6)`（`0x22f9f`）。
- `0x22fd0` 匯合後才共同執行 pan / SPAWN4 / ACT14..16 / 後續對白。

舊 exporter 依地址排序 call，會把 #6、送物品、#7 錯誤串成三拍。現在 extractor 依 Watcom 的
「byte accumulator + fixed-slot loop + test/jne diamond + forward merge-jmp」指令形狀辨認，不硬寫
handler 地址；editable IR 產生 `if`、`condition:any_unit_inactive`、`then`、`else`，compiler 會遞迴
resolve **兩臂**，任一臂缺 binding 就整個 branch fail closed。runtime 先驗證所有指定 slots 都存在，
再只把選中的 arm splice 到當前拍後；使用新 slice，不會改寫 campaign node backing array，共同 tail
維持一次。dialogue-binding exporter 與 unknown diagnostics 也已改成遞迴走 arm。

manifest 在 shared-tail CFG 修正後全量為 **701 個 top-level beats**，unknown 為 **108**；structured
`if` 的 then/else 仍需遞迴計入 diagnostics，不能只看頂層 beat 數。

### 3.5 FDTXT_002 61 utterances 與 dynamic speaker slot（2026-07-16）

原始 17 strings 的 logical utterance counts 是：
`[1,3,6,10,9,1,1,1,15,7,1,1,1,1,1,1,1]`，合計 **61**；舊 `ch02.json`
只有 53 lines。差額不是八句連續 postbattle 對白：完全漏掉的是 #5 與 #11..16 共 7 句，另 #6/#7
兩個互斥獎勵字串被以 `/` 壓在同一 line，再少一個結構位置。現在 story scenes 為 20+12+23+6=61，
count-aligned index 可精確建立；ch01 post 使用的 #6..10 展開為 **1+1+15+7+1=25 lines**，五個
dialog call-sites 全部 resolve，generated dialogue contexts 73→78、skipped 73→68。

另有一個重要 speaker 修正：FDTXT 的 `FFED operand` 是 runtime unit direct index（doc40），不是全域
角色 ID。map1 前 5 slots 是 party，slots5..10 六名村民的 DATO portraits 為 134/133 交錯；因此
#6/#7 的 operand 6 是 DATO133 村民，不是萊汀，#11..16 也是六名村民倒下短句，不是洛娜到瑪琳。
#4 的「救命啊」是 slot8/DATO133 村民，「往東南逃」回話是 slot7/DATO134 村民。editable story
新增 `speaker_slot` 保存直接索引、`speaker` 保存已知 DATO audit value；runtime 必須從當前 battle unit
或 materialized cutscene unit 的 `Portrait` 解析，slot/state 不存在就 fail closed。`upper` 同時保存
FFED/FFEF 上框與 FFEC/FFEE 下框，
避免再用角色編號猜框位。

### 3.6 ch01 post 完整接線：canonical runtime slots + postbattle context（2026-07-16）

先前 `battle.Load` 把 map1 全 40 筆 FDFIELD records 放入 `State.Units`，`Scenario.Setup` 只切
`OnField`，再把五名 party append 到 slots40..44；group4 希莉亞更因誤列 `initial_groups` 而提前在
slot22。這不只是畫面差異：存活 diamond 的 slots5..10 會混入敵軍，FFED speaker slot 也會讀錯人。

ch02 現改用 `runtime_append_groups` constructor 模式：FDFIELD records 留在 source roster，canonical
runtime array 依事件實際 materialize：party slots0..4、group1 村民 slots5..10、group2 slots11..20；
turn3 的 SPAWN3 才 append slots21..26，戰後 SPAWN4 才 append 希莉亞為 **slot27**。group255 padding
不再污染 runtime slots，也不計入尚待出場敵軍。`State.AppendGroup` 是無進場滑動的原版 `0x10b4e`
投影；戰鬥增援 `SpawnGroup` 在同一 append 後另加既有動畫。

post handler binding 新增明確 `runtime_context`：進入時必須已有 27 slots，group4 cardinality=1，並
切到原版 13×8 的 story viewport；compiler 由這個 count 驗證 branch/ACT，在 SPAWN4 後把可用 slot
frontier 推到 28。runtime 不建立第二份 story copy：PAN、SPAWN、ACT、FFED、grant、JOIN、sync_party
全都作用於同一個 `g.st`，因此希莉亞加入後也能被後續 persistent roster 同步保存。

兩個 PAN 已由 `0x135dd` 完整 body 定案：參數是 camera grid origin，原版 tile=24px；map1 27×21、
viewport 13×8，故 `(14,2)→(336,48)`、`(14,1)→(336,24)`，其中 col14 也正好是 X 最大 origin。
這兩點仍以 address-keyed binding 保存；沒有把所有章節未驗證座標改成全域猜測公式。ACT14 使 slot27
由 `(22,4)` 下移到 `(22,6)`，ACT15 special 操作 party slots0..4，ACT16 再使 slot27 到 `(15,0)`；
均有 headless runtime regression。實機 Xvfb 也已跑過完整 branch→reward→PAN→SPAWN→ACT14，並在
FDTXT #8 首句停住截圖確認希莉亞出場與原版窄視窗。

campaign 現為 `battle_ch02.on_win → story_ch02_post → town_ch03 → preparation_ch03 → story_ch03`。因目前 save format 只保存節點
邊界、不保存 completed battle array，戰後 handler 進行中會明確拒絕 F5；到下一節點即可正常存檔，
避免讀檔後缺 runtime slots 而必然失敗。

### 3.7 ch02 pre 與 FDTXT_003 39 utterances（2026-07-16）

`ch02_pre`（zero-based table index2，玩家第三章）的 16 source beats 已完整 lower：LOADCH chapter2、
三段 `0x135dd` PAN `(3,17)→(72,408)`、`(3,6)→(72,144)`、`(3,17)→(72,408)` 都採 X-first
`tile_step`；ACT18 操作 party slots0..5，SPAWN1 append map2 group1 九人後，ACT17 操作 slot6 鐵諾，
ACT19 操作 slots7..14 八敵再回 slot6。尾端 `0x32fad` 跳到 shared block，播放 FDTXT #3、reset、
focus slot0。party construction 依 JOIN caller chronology 是 `[0,9,4,30,1,8]`，不是 scenario UI
排列 `[0,4,9,30,1,8]`；map2 battle 也改成 runtime append，避免 19 筆 group255 padding 汙染 slots。

舊 `ch03.json` 只有33 lines，原始 FDTXT_003 十個 strings 的 logical counts 是
`[2,1,4,7,7,1,5,10,1,1]`，合計39。缺的不是空白 padding，而是原版 battle turn3 hard-code #4
後六句（鐵諾追問、葛雷指使、希莉亞揭露卡蘿線索、約下令攻擊）；已補回 editable scene0 lines15..20，
故索引 exporter 現能機械得到 **39/39 count-aligned**，diagnostics 9→8、generated contexts 81→83、
skipped 89→87。pre #0..3 精確展開 2+1+4+7=14 dialogs，因此完整 runtime 為26 beats、0 issues；
campaign 的 `story_ch03` 已由章標 stub 改接 authored `ch02_pre.json` 後再進 battle_ch03。

2026-07-16 完整 writer/death/revive body 已釘死方向：有效 constructor `0x10eed` 寫 bit0=0，
HP 歸零路徑 `0x1dc61/0x1dd4c` 寫1，復活 `0x30f9c` 再清0；所以 `0x3453e` 是
`unit_inactive`，不是 `unit_alive`。ch03 turn3 `0x344d8 jne` 表示 slot6 死亡／inactive 就跳過；
只有鐵諾仍 active/alive 才 SPAWN2、PAN、delay 並播 #4。scenario 已增加 editable
`when:{turn:3,unit_slot_active:6}` 防止死亡路徑誤生援軍；FDTXT_003 #4 七句已接入同一
battle event，並以原始 portrait 77/2/77/8/2/8/77 顯示。PAN `(3,0)→(3,17)` 與 800/200ms
現已由通用 `battleEventRun` 依 editable action 原序播放：SPAWN2 → PAN grid(3,0) → delay800ms →
PAN grid(3,17) → delay200ms → 七句對白。`Scenario.TriggerActions` 只評估/標記 once 並回傳 actions，
runtime 再逐項執行；battle runner 與 campaign BeatRunner 分離，避免完成事件時誤推進 campaign。
PAN 由 map2 24px tile 轉成 `(72,0)/(72,408)`，X-first tile step；等待精確為 48/12 ticks。
最後一句清空前不會增 Turn 或 tick 毒／buff，finishTurn 重入亦不會重複 SPAWN。事件期間改用原版
320×200（13×8格）離屏視野再放大2倍，Xvfb frame120 已確認第二個 PAN 後畫面無寬視野黑區且先播
portrait77 的第一句。

`ch02_post` 真 CFG 是：`sync_party → if slot6 inactive {#6 哀悼} else
{0x233c6 layout + #7 + JOIN(2)} → set_chapter(3)`。single-slot diamond、`layout_units` 與 15/27-slot
runtime frontier 已於 §3.8 完成，不再把互斥兩臂串播。
#5/#8/#9 不在這支 post handler CFG，不可因 39/39 mapping 就強塞進去。

### 3.8 ch02 post：single-slot diamond + `layout_units`（2026-07-16）

extractor 現依可重用的 `push slot; call 0x3453e; test; je/jne; 兩臂同 merge`
指令形狀復原 single-slot branch，不硬寫 handler 位址。ch02 post 因此編輯成：

- `sync_party`；
- `if any_unit_inactive([6])`：死亡臂播 FDTXT_003 #6 五句哀悼；
- active 臂執行 `layout_units`、#7 十句、`JOIN(2)`；
- common tail 的 `inc [0x53c03]` 現保留為唯一 `set_chapter(3)`。

`0x233c6` 以 call-site binding 保存 slots0..6 的絕對 `(X,Y,pose)`：
`[(8,3,2),(7,3,2),(9,3,2),(6,2,3),(10,2,1),(8,4,2),(8,1,0)]`；鏡頭 grid `(2,0)`
投影為 pixel `(48,0)`。compiler 把該原語完整 lower 為 layout、redraw、palette fade-in、delay200ms。
由於 turn3 group2 可能因鐵諾死亡而未生成，post 入口不能假定單一 count；
`runtime_context.slot_counts:[15,27]` 會在 runtime 只接受這兩個原版 frontier，compiler 以共同最小
frontier 15 驗證 layout 的 slots0..6。campaign 現已接成
`battle_ch03 → story_ch03_post → town_ch04 → preparation_ch04 → story_ch04`。

同輪也把 29 支 post handler 的 `inc [chapter]` 保留成 editable `set_chapter`，並將 15 個
`0x233c6` caller 從 unknown 分類為需 address-keyed layout binding 的已知原語。全 60 支 manifest 現為
**725 top-level beats / 93 unknown calls**；「已命名」不等於其他 14 個 layout caller 已猜好座標，未有各章證據的
binding 仍會 fail closed。

### 3.9 戰後不是直接下一戰：`0x2cad7` town / preparation 契約（2026-07-16）

勝利 driver 的原版順序是 `0x25e1e` 取目前 `[0x53c03]`、`0x25e23` 呼叫
`0x51de9[current]` post handler，接著 `0x25e2a call 0x2cad7`執行戰間流程，最後才在
`0x25e35..0x25e3a` 取已更新的章節呼叫 `0x51d71[next]` pre handler。因此完整契約是：

```text
battle[current] → post[current] → intermission[next] → pre[next] → battle[next]
```

`0x2cad7` 在 `0x2cbf2..0x2cbfe` 測試 `byte[chapter+0x526b9]`。以 LE object #2 映射回
FD2.EXE file offset `0x524b9` 的 30-byte 章表為：零起算 chapter **0..21 = town**、
**22..24 = preparation**、**25..26 = town**、**27..29 = preparation**。這與
`docs/data/shops.json` 只有玩家章 **2..22、26、27** 完全一致；商店表的 chapter 是「下一場要進入的
玩家章」，例如第2章羅德鎮位於 `battle_ch01` 戰後。舊 generator 把 `shop_chNN` 接在
`battle_chNN` 後的 off-by-one 已修正。青衫攻略 `references/text/fd2.md:1196` 亦明言第22章開始
連戰四場才再有商店，`:1472` 則記第27章是最後買賣處。

town 路徑在 `0x2cd04` 以 `0x4e4b9(chapter)` 讀城鎮資料，並以 FDTXT_000
`0x1ef..0x1f3` 顯示**酒店、武器店、出口、道具店、教會**（可讀轉錄見
`extracted/story/full_story_auto.md:450-455`）；隱藏 selection 5 是神秘商店。`0x2d28c`
將酒店分派到 `0x2fc85`、教會到 `0x3072f`、武器／道具／神秘商店到 `0x2e341`；
設施結束後回 town hub，不是強迫「武器店→道具店→離城」的線性鏈。選出口會以
FDTXT_000 `0x201` 詢問「要進入戰場嗎？」，然後進 `0x318ad` 出戰隊伍整備。

無 town 的 table=1 路徑也不會直接進戰：`0x2cc04` 顯示 FDTXT_000 `0x19a`
「要記錄戰況嗎？」，允許存檔後同樣進 `0x318ad` 隊伍整備。remake 因此新增可編輯
`town`、`preparation`、`church` 節點；town 五個可見設施與隱藏神秘商店皆回 hub，
出口才進 preparation。`TestCampaignFullPostBattleTownContractMatchesOriginalShopChapters`
對全戰役驗證：商店章集合為 `[2..22,26,27]`、`battle_ch(N-1)` 必先到對應 town，
各設施必回 hub，章23/24/25/28/29/30 無 town 但必有 preparation，且第30章勝利才進 ending；
`TestRunnerTownUsesVisibleOptionOutcome` 則固定 town option 轉移。

玩家第27章（零起算 post `ch26_post`）的天空之鑰分支已先接成 editable campaign gate：
`battle_ch27 → inventory_gate_ch27_sky_key`。`0x25186 call 0x24b14(item 0x64)` 的完整 body
只掃 runtime unit records slots 0..15，不檢查 camp／active；找到鑰匙才走
`story_ch27_post_sky_key_success` 的 `sync_party → set_chapter(27)`，再停在 `preparation_ch28`；
回傳 `-1` 則走獨立 `ending_ch27_no_sky_key`，對應 `0x2545d call 0x2bce5` 壞結局。
runtime 在沒有 active battle array 時另查 persistent roster，是為 node-boundary save/load 做的 remake
projection，不冒充 `0x24b14` 的 byte-exact 行為。Load 會拒絕缺 `item_id` 或缺任一 arm 的 gate；
runtime／campaign tests 同時固定 slots 0..15、present/missing、成功 sync、save round-trip 與第28章整備邊界。

「取得天空之鑰」也已接入，但完整 body 揭露的規則比攻略口語的「六件全齊」更怪：零起算
`ch20_post`（玩家第21章戰後）在 `0x2418a..0x241cd` 對每個 item `0xD1..0xD6` 與每個
runtime slot 0..15 呼叫 `0x31860`；每個 `(item,slot)` 只要該角色至少持有一件就加一，最後
`cmp ebx,6` 要求**命中組合總數恰為 6**。所以同一素材分散兩名角色會算2；D1兩人持有、
D2..D5各一人、完全沒有D6仍會湊到6而成功，反之正常六種都有但任一種分散兩人形成7組反而失敗。
這個原版怪癖由 `inventory_recipe` 與 regression 原樣保存，不能擅自正規化成 set membership。

成功時 `0x241d6..0x24220` 仍以 item→slot 順序，在每個命中 pair 移除該角色的第一件，然後
`0x24224` grant `0x64`；失敗完全不改 inventory。campaign 現為
`battle_ch21 → story_ch21_post_sky_key_intro(#5十句) → inventory_recipe_ch21_sky_key`，
成功臂使用 editable FDTXT_021 #7..#10 全16句，失敗臂使用 #6 全4句；兩臂共同
JOIN24/JOIN23、sync_party、set_chapter(21)，最後都回 `town_ch22`，沒有跳過商店／整備。
原 handler 的 `layout_units`、ACT63/64 與 `0x24336` 鑄造動畫尚未 lower，所以目前是完整文字、
物品與持久化分支，不宣稱視覺演出 1:1。六素材的正常取得資料與 runtime 接線見下一節。

### 3.10 天空之鑰六素材：人物 defaults、寶箱與特殊死亡 reward（2026-07-16）

六件材料已由攻略、EXE 與 FDFIELD 三方交叉固定，不能簡化成章末 grant：

- D1 黃金徽章：索菲亞 EXE 人物出場 defaults（`FD2.EXE file+0x55CA9`，IT 從 +12，
  `[0x36,0xA7,0xD1]`）。ch10 FDFIELD 場景 record 只有前兩件，證明 D1 是 JOIN 命名角色時由 EXE
  default 帶入；`character_defaults.json` 與 ch11 editable party 已保存。
- D2 星之眼：map10 `(18,37)`，terrain166／chest slot0，control reward `00 D2 00`。
- D6 火之眼：map12 `(38,18)`，terrain118／slot8，reward `00 D6 00`。
- D4 暗之眼：map19 `(30,7)`，terrain110 的 `0x40` hidden flag／slot7，reward `00 D4 00`。
- D3 光之眼：map14 unit58，group1、position `(33,1)`、inventory `[0x0F,0x88,0xD3]`；其
  death effect `type2,id39` dispatch 到 `0x34F74`，該 handler 將 `00 D3 00` 傳入 `0x1AA1D` reward。
- D5 冰之眼：map16 unit0 水魔神，inventory `[0x54,0x82,0xD5]`；`type2,id41 → 0x34FF0`，
  handler 將 `00 D5 00` 傳入同一 reward dispatcher。

composition 每格仍是 `(terrain_word,event_word)`；EXE `0x12E38` 使用
`terrain=word0&0x3FF`、`slot=lowbyte(word1)&0x1F`。terrain control byte0 的 `0x20/0x40`
分別是普通寶箱／隱藏物，reward 必須用 slot 關聯 control `+0x53+slot*3`，不能把已過濾的 chest
清單依序 zip。匯出管線現保存 row-major `treasure_slots[]`（`-1` 才是無寶物；slot0 合法）、
`treasure_hidden[]`、帶 slot 的 chests、unit inventory、raw death_effect 與 lowered death_reward。

取得手勢也由 caller 鎖定：`0x190AC` 唯一由行動選單第四項「下／休息」的 `0x1908B` 呼叫；
不是踩格即取，也不是物品指令。未移動先回 MaxHP 20%，之後才檢查當格寶物。item 必須放入當前
active unit 的8格 inventory，滿欄不設 opened；gold 進隊伍金庫；無 camp 限制，所以敵人也可開。
remake 的 `ClaimTreasure` 與待機 UI 已按此實作，普通／隱藏只差視覺。已開普通箱原版會 terrain+1，
hidden 不改圖；普通開箱換圖仍待接。敵方 API 可取，但現有簡化 AI 尚不會以寶箱為尋路目標，
所以「獸人搶 D2/D6 後逃走」的 AI/離場事件仍是下一個戰鬥事件切片，不可宣稱已 1:1。

D3/D5 不採「搬走敵人全部 inventory」的猜法，而使用上述特殊死亡 handler 明示的單一 item reward；
物理、魔法、毒／狀態死亡共用 once-only reward path。擊殺者滿欄時，現階段先投影到其他己方空格，
待物品給予 UI 完成後再還原原版提示。材料取得後的五個關鍵戰場（ch11/13/15/17/20）最先接上 editable
`postbattle_chNN_persist`；隨後全 campaign 稽核抓出另有19條非終局戰勝路徑直接進 town／preparation，
而 `enterNode` 會在該邊界清空 battle state，會實際遺失本戰能力與 inventory。這19條也已各接同一
可編輯契約：`sync_party → set_chapter → 原本 town/preparation`。現在 chapter1..29 的每條正常續關路徑
在第一個戰間節點前均恰有一次同步；chapter30 仍保留原版終局 `ending`。因此不會為保存物品而跳過
商店／教會／整備。ch21 既有配方兩臂則仍共同 sync 後回 town22。

## 4. 未解(低優先)+ 工具紀律

- ~~acting 原始靜態容器未定位~~：已定位為 `FD2.EXE file+0x565d8` 的 106×u32 offset directory，
  資料位址=`file+0x53e00+offset`；getter 以 ID 直接索引。見 §1.2/doc48 §6。
- ~~草地「亞雷斯撞見」主角走位未解~~：direct ACT101..105 已完整解釋 slots3/4 位移，remake
  使用原 acting frames，不再以手寫 storyWalk 外推。
- **[工具紀律]反組譯語意必看完整 body**：`0x32999` 完整 body 證實為 spawn_group_with_intro；
  `0x12d7b` 不是戰鬥 log，而是讀 unit X/Y 後呼叫 `0x12cea` 逐格 focus。call-site 名稱或淺層 caller
  不足以定性，extractor 只收已由 callee body 證實的名稱。

## 5. 版權界線

beats JSON=行為事實轉錄(指令+參數),同 scenarios 屬原創整理可入庫;
acting 資源原始 bytes、dump 檔=版權物,永久留 extracted/(gitignore)。
