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
| **演出** | acting 資源(表 0x50~0x99 共 74 筆),0x1366a(id) 播放 | 74 筆全 dump+解碼(dosbox-x) |
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
| `REVEAL(n)` (0x32975/0x32999) | 攝影機 reveal 族(內部待展開,先當 pan 近似) | beat op:pan(近似) |

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

**單位結構(0x50B/槽,`[0x53a45]+idx×0x50`)**:+0=X格、+1=Y格、+3=pose(=方向)、+4=tick倒數、+5=狀態旗標(bit0存活/bit7已行動)、+8=角色ID。

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

**草地的剩餘追查點**：顯式 handler `ACT(0x65..0x69)` 的 bytes 沒有 bit7=0 frame 選 slot3/4，
但不能再推出「草地走位不在 acting」：slot3 X 的實機 `4→5` 寫入中斷已回溯至 acting 正常模式
`0x1391e call 0x2c9ec`。**待定位的是草地對話期間觸發該正常 acting 資源的間接來源／資源 ID，
不是 decoder 格式或位移機制。**

**[實機快照 2026-07-15]**：同一份 FD2.EXE 草地對話中，slot3 仍 `(4,46)`、slot4 已 `(7,46)`。
影片接近是真的改寫 unit[+0]/[+1]。首次 BPPM 命中會落在 thunk，不能採其 EIP；第二次命中的堆疊
才回溯到 `0x1391e`，是有效證據。

> 下列段落保留 roster／顯式 handler bytes 的史料背景；其中「acting 給面向、走位是另一套」的舊結論
> 已被本節更正，禁止引用。
2026-07-05 用 **dosbox unit-array dump**(`extracted/dosbox_dump/task_f/slots0_20_dialogue.bin`)解出 map32 roster:
**slot2=王座索爾(走王座那個)、slot3=草地索爾(4,46)、slot4=草地亞雷斯(13,47)、slot5-20=守衛、slot16/17=守衛**。
而 handler 草地段(disasm 0x323f5~0x3251c)呼叫演出 **0x64~0x69**,解碼後:0x65=units0-15 全 pose2(全體面上,定場)、
**0x66/0x67/0x69 動的是 unit16/17 = 守衛**、0x68=unit1。**沒有一筆動草地主角 slot3/4**;草地段也**無 step/0x13488 呼叫**。
⟹ 顯式 handler 的資源清單仍不足以解釋草地主角走位；**未解的是對話期間額外進入的 acting 呼叫來源／ID**，不是 acting 位移語意。

**[實機快照補證 2026-07-15]**:以同一份 `FD2.EXE`（`org_game/.../FD2.EXE` 與
`extracted/dosbox_dump/game_run/FD2.EXE` SHA-256 相同）跑到草地對話中，直接傾印 unit array：
slot3 索爾仍為 `(4,46)`，但 slot4 亞雷斯已為 **`(7,46)`**（初始 FDFIELD 為 `(13,47)`）。
因此草地影片中的接近是**真的改寫 unit[+0]/[+1] 邏輯格座標**，不是 pose、tick 或鏡頭造成的視覺假象。
第二次 `BPPM` 命中已由堆疊回溯到 acting 正常模式的 `0x1391e`（首次 thunk 命中仍不可採）；
因此剩下的工作是直接在 `0x1366a` entry 擷取實際 ID 與 caller，定位非 handler 顯式 ACT 的來源。

**[靜態排除 2026-07-15]**：對話 parser `0x15f84..0x164e3` 的所有直接與間接 control transfer
均不會到 `0x1366a`；FDTXT 控制碼不是這個額外 acting 的發動來源。草地 handler 的 `ACT(0x65..0x69)`
也都是直接呼叫且資料不選 slot3/4。故 runtime entry/caller trace 是下一個最小且有辨識力的實驗。

**[追蹤準備 2026-07-15]**：在草地實機的 CS dump 以 entry 簽章
`68 88 00 00 00 … 53 56 57 55 83 EC 5C` 搜到真正入口 `CS=0158:1C966A`（不是曾誤算的
`1C766A`）。dynamic core 下該 code breakpoint 沒有可靠停住，即使流程已跨過整段草地；這與 DOSBox-X
對 dynamic core 的 breakpoint/step 警告一致。後續必須以 **normal core** 重跑至草地，於此入口讀
`SS:ESP` 的 return address 與 `SS:ESP+4` 的 acting ID；在未抓到這兩值前，不宣稱額外資源是哪一筆。

**[資料面直接證實 2026-07-05,靜態讀 dump 非 dosbox]**:讀 74 筆解碼 acting(`extracted/dosbox_dump/acting_decoded/`),
草地幕 0x65~0x69 資料 = 每幀 (拍數, unit, pose):0x65=units 0-15 全 pose2(上,定場,拍數5)、
0x66/0x67=units 16&17 面向循環、**0x69(帶他離開)=unit16 設 pose3(右,領頭面向)**。這只能描述
這幾筆**顯式**資源，不能再用來否定對話期間觸發的正常 acting 位移。

**acting 資源(74 筆,id 0x50–0x99)**:執行期 base=0x207718，已全 dump+解碼。原始靜態容器仍未定位；
但正常／特殊 frame 語意已足以讓 remake 實作播放器。

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
- `fig` 是 remake 既有角色／sprite 組 id，無法對上的原版 unit index 必須標待 RE，不可硬填。
- 舊 `poses`／`pose_frames` 仍可用於尚未轉錄的近似場景，但新的原版 acting 不得再降級成它。

> **系統界線(2026-07-04,doc52):本 DSL 只承接「戰前/戰後過場編排」(handler 0x3231b 族,系統 A)。
> 「戰鬥中回合事件對話」(哈諾第 3~4 回合等)是另一套系統 B(跳表 0x51b19 / battle_events.json,
> doc26),由 `battle.Scenario.Events` + `Fire(on_turn_end)` 承接,邊打邊觸發,不進 cutscene beats。
> 兩套並存不混——把戰鬥對白塞進開場一次播完是先前的架構錯誤。**

## 3. 全 33 關機械破解管線

1. **`tools/dump_chapter_beats.py`(已完成,2026-07-04)**:走跳表 0x51d71/0x51de9 30 entry,
   對每支 handler 跑「push/call 配對抽取」(doc47 §7 的方法,已驗證),輸出
   `docs/data/chapter_beats/chNN_{pre,post}.json`(原語+參數序列,機器可讀)。
   **ch0(序章)逐項核對 doc47 §7:73 個 call beats 完全吻合,含迴圈自動偵測精確抓到
   `fade_step(2)×15`/`×13`**(不是人工事先寫死,是 parser 自動偵測 push→call→inc→cmp→jl
   模式解出重複次數)。
2. 轉換器:beats JSON + 章文本 + FDFIELD roster → campaign 節點(cutscene beats)。
3. 引擎 BeatRunner:main.go 依序執行 beats(pan/dialog/walk/act/spawn/join/bgm/fade/delay)。
4. 驗收:每章過場對照 dosbox 錄影(規則 65,對 reference 非內部訊號)。

### 3.1 原語覆蓋率(全 30 章,2026-07-04)

全 629 beats(含 loadch_var 這類非 call 記錄)中,**已知原語 483 筆、未知 146 筆(覆蓋率 76.8%)**,
未知原語 30 種位址,集中在**戰後(post)handler**(戰役流程控制/中場銜接族,跟序章那種純過場
敘事不同族)。逐一淺層反組譯定性(前 40 條指令,看讀寫哪些已知變數/呼叫哪些已知函式):

| 位址 | 次數 | 淺層定性(證據) |
|---|---|---|
| `0x11506` | 24 | **roster↔戰場單位雙迴圈配對**:外層走 `[0x53a45]`(戰場全單位,0..`[0x53beb]`)、內層走 `[0x53bf7]`(我方名冊,0..`[0x53bfb]`),比對兩邊 `+8`(角色ID)相同;若角色ID=0(索爾)額外呼叫 `unit_alive(idx)` 確認存活才算命中。疑「找 roster 成員在戰場的槽位/檢查特定角色是否已在場」,跟 `roster_has`/`unit_alive` 同族但更底層。 |
| `0x233c6` | 15 | **批次寫入單位陣列 X/Y 座標+初始 pose**:迴圈對 `unitbase+idx*0x50` 寫 `+0`(從 edi 陣列讀 X)、`+1`(從 ebp 陣列讀 Y)、`+3`(<4 的小常數,疑初始 pose)。疑是「roster/FDFIELD own 展開寫入戰場陣列」的初始化實作,呼應 doc47/48 單位結構 `+0=X,+1=Y,+3=pose` 定案。 |
| `0x24b4d` | 15 | **畫面過渡效果**:push 鏡頭 `[0x53aa9]/[0x53aad]` 呼叫 `0x11eee`(地形重繪)+`0x11cac`(主重繪)+迴圈呼叫 `0x11eb0`(present)+`DELAY(20ms)`。與 acting bit7 特殊模式分支(doc47 §9)看到的同一組呼叫序列相同,疑是該分支背後共用的「reveal/漸現」子程序。 |
| `0x11df2` | 12 | **VGA 調色盤處理**(**推翻 team-lead「疑 0x11cac 同族」的猜測**):操作 `[0x53a65]`(新變數,調色盤資料表?)+呼叫 `0x37795`(push 常數 `0x3c8`/`0x3c9`——VGA DAC 索引/資料 I/O port 位址),跟 `0x11cac`(畫面重繪)不同族,是獨立的調色盤/淡變數值計算函式。 |

其餘 26 種(次數 1~8)未逐一反組譯,清單見 `docs/data/chapter_beats/_stats.json`。

## 4. 未解(低優先)+ 工具紀律

- **acting 原始靜態容器未定位**(格式已知 + 執行期 74 筆已 dump;§1.2)。**低優先**:remake 動畫近似即可,不 RE 也能做(使用者確認,需要時給影片)。
- **草地「亞雷斯撞見」主角走位 = 深層未解**(§1.2 定論,dump 佐證):handler 草地段呼叫演出 0x64~0x69,
  但解碼 + roster(dump)證實那些**動守衛(16/17)+設面向,不動草地主角 slot3/4**;草地段也無 step/0x13488。
  ⟹ 索爾(slot3)/亞雷斯(slot4)走位的驅動源**未定位**(不在 handler acting、不在 step 家族)。
  roster 已解(slot3=草地索爾/slot4=草地亞雷斯,dump `task_f/slots0_20_dialogue.bin`)。
  remake 目前用 storyWalk grid-walk 對齊 doc55 影片量測值(可玩、視覺對得上);精確原版驅動待更大 RE。
- **[工具紀律]sonnet 不適合反組譯語意判讀**:實測開場 handler 解碼,呼叫序列抄對但 7 原語猜錯 6(0x32999 錯猜「鏡頭滾動」實為 spawn_group_with_intro、0x10b4e 錯猜「發話者」實為 spawn、0x12d7b 錯猜「清場」實為印戰鬥 log),還誤說「森林不在 handler / handler 提早結束」(**與 doc47 §3 矛盾:森林=0x3231b Part2,早已轉錄**)。反組譯語意判讀交旗艦自己做 + 反查 KB,sonnet 至多機械 grep 且結果必覆核。

## 5. 版權界線

beats JSON=行為事實轉錄(指令+參數),同 scenarios 屬原創整理可入庫;
acting 資源原始 bytes、dump 檔=版權物,永久留 extracted/(gitignore)。
