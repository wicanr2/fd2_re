# 25 — 戰場事件系統:章節 handler 與事件原語(反組譯)

> doc 24 §6.3 留的受阻項:`[0x53ecc]=1/2` 的 28 個寫入點(0x205c9–0x20c64)各對應哪種事件、整個「事件指令集」長怎樣。本篇挖完。
> **核心結論(且修正 doc 24 用詞)**:FD2 的戰場事件**不是 byte-code opcode VM**,而是**「每章一個編譯進 EXE 的 C handler 函式」**,放進第三張章節跳表 `0x51b19`,由戰場迴圈在每場結束時 `call [章節*4+0x51b19]` 觸發,handler 檢查條件後設 `[0x53ecc]`(1=中途事件 / 2=勝利)。
> 方法:call-graph(`callgraph_le.py`)+ fixup 跳表解析,全程比對既有結論(rulebook 62/63)。標 **[驗]/[推]/[阻]**。

## 1. 三張章節跳表(都以 `[0x53c03]` 章節索引)

挖事件系統時補齊了第三張跳表,與 doc 23 的兩張並列:

| 跳表 | linear | 用途 | dispatch 點(已驗證) |
|---|---|---|---|
| **戰場事件** | `0x51b19` | 每場戰鬥結束時呼叫,決定 `[0x53ecc]` | `0x1197b`(戰場迴圈 0x117e7 內):`mov eax,[0x53c03]; call [eax*4+0x51b19]` |
| 戰前/劇情 | `0x51d71` | 進章節前的 cutscene / 戰場設置 | 0x25f10、0x260f5、0x25e3a |
| 戰後/勝利 | `0x51de9` | 戰鬥勝利後的劇情 | 0x25e23 |

> 同一跳表 `0x51b19` 也被過場/世界地圖模組參照(0x1a950、0x1d8a3、0x1d96f、0x1d9ff)——即**戰場與過場共用同一套章節事件 handler**。[驗]

## 2. 為何是「handler 函式表」而非 byte-code

`0x1197b` 的 dispatch 用 **`[0x53c03]`(章節)** 當索引,不是讀腳本資料的 opcode。
跳表 30 個 entry 全部指向 code 段的函式入口(0x205b4–0x20bf5),不是資料偏移。
→ 漢堂沒做資料驅動的腳本 VM;**每一關的特殊事件邏輯,是工程師逐關手寫成 C 函式編進 EXE**。這是 1995 年常見做法(省去寫 VM + 編輯器的成本),代價是「改事件 = 改程式重編」。[驗]

## 3. `0x51b19` 全 30 章 handler 對映 [驗]

```
章: 0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29
hdl:D  a  D  D  D  D  D  D  D  b  D  c  d  D  e  f  g  h  i  j  k  L  m  D  n  o  L  L  p  q
```
- **D = default `0x205b4`**(11 章:0,2,3,4,5,6,7,8,10,13,23)——標準戰場,直接判勝利。
- 特殊 handler 18 個相異:a=0x206c5(章1)、b=0x20707(章9)、c=0x2073d、d=0x20765、e=0x20822、f=0x2084a、g=0x20872、h=0x208cf、i=0x20926、j=0x20957、k=0x20a51、**L=0x20a87(章21/26/27 共用)**、m=0x20aaf、n=0x20b14、o=0x20b3c、p=0x20b72、q=0x20bf5。

## 4. 事件原語(handler 共用的條件 / 動作函式)

把 18 個 handler 呼叫的函式統計出來,扣掉 stack-probe(0x36cd7),得到事件系統的「指令集」——雖是函式呼叫形式:

| 原語 | linear | 次數 | 作用 | 狀態 |
|---|---|---|---|---|
| **查單位旗標** | `0x3453e` | 36 | `unit_flag?(idx)`:`[0x53a45] + idx*0x50 + 5` 的 bit0=**存活(HP>0)**(使用者確認;初始=1,陣亡→0),bit7=已行動 | [驗 欄位] |
| **handler prologue** | `0x205be` | 15 | 預設 `[0x53ecc]=2` → 清 `[0x53ecc]=0` → `call 0x1088d`(載該章 FDTXT 文本/腳本) | [驗] |
| **繪事件畫面** | `0x15f84` | 6 | 全螢幕圖繪製(過場 / 事件畫面) | [驗] |
| 我方名冊查詢 | `0x33499(id)` | 1(章16) | `roster_has(id)`:查我方名冊 `[0x53bf7]`(32槽×0x50B)byte[+8]==id | [驗] |

> **更正(doc 26 補完)**:早先把各章 ×1 的 `0x33499` 等列為「增援/對話動作」,經逐關反組譯確認 **handler 無動作函式**——`0x33499` 是條件查詢(roster_has),其餘是控制流;handler 只「條件→設碼+繪圖」。增援/對話在碼1後的世界地圖/章節跳表流程。

關鍵狀態變數:
- **`[0x53a45]`** = 戰場單位陣列基底(每單位 0x50 byte;doc 23)。事件條件多半在查它。[驗]
- **`[0x53bef]`** = **回合數**(戰場開始=1、`inc`、handler `cmp N`)→ 「第 N 回合觸發」類事件。[驗](doc 26)
- **`[0x53ec8]`** = 累積計數(`add reg`+每 tick `clamp` 99,0x11959;非回合數)→ 語意待定。[推]
- **`[0x53ecc]`** = 輸出碼(1=中途事件 / 2=勝利 / 0=續打),戰役迴圈據此分派(doc 24 §6)。[驗]

## 5. 實例:章 1 handler `0x206c5` [驗]

```
0x206d0  call 0x205be             ; prologue:載章節文本、預設碼
0x206d5  edx = 5                   ; 迴圈單位 5..10
0x206dd  cmp edx,0xb; jge 0x206fb
0x206ed  eax=[0x53a45]            ; 單位陣列
0x206f2  test byte[ebx+eax+5],1    ; 查單位 #edx 狀態 bit0
0x206f7  je 0x20705                ; 有一個不滿足 → 跳出
0x206f9  (續迴圈)
0x206fb  mov [0x53ecc],1           ;★ 單位 5..10 全滿足 → 觸發中途事件
0x20718  push 0x32; call 0x3453e   ; 另查單位 #0x32(50)狀態
0x2073...  push 0x33; call 0x3453e ; 查單位 #0x33(51)
```
即:**章 1 的腳本邏輯 = 「若單位 5–10 群狀態旗標(+5 bit0)都成立 → 觸發劇情事件;再依單位 50/51 旗標分支」**。
這就是一條「事件指令」的真身——一段檢查單位狀態的硬編碼條件 + 設 `[0x53ecc]`。

> 單位 byte(+5):**bit0=存活(HP>0)、bit7(0x80)=已行動**(使用者確認)。回合數 [0x53bef] inc=我方全動+敵方AI全動完一輪。

## 6. 完整事件流(串起 doc 23/24/25)

```
戰場迴圈 0x117e7
  ├ 跑戰鬥(行動 0x18890…)
  ├ [0x53bef] 回合數(開始=1,回合切換 inc,inc 點 0x1a5b9);[0x53ec8] 累積計數 clamp 99
  ├ call [章節*4+0x51b19]           ; 章節戰場事件 handler(勝負判定,見上)
  └ call 0x1a813(camp_filter)       ; ★turn_events 消費點(見 §6.1),3 處呼叫點各帶 camp=1/0/2
       └ default 0x205b4 → [0x53ecc]=2(敵滅→勝利)
              │
戰役迴圈(doc 24 §6)讀 [0x53ecc]
  ├ ==1 → 世界地圖/中場 0x22e5c(章1專屬固定過場,非資料驅動,見 §6.1 修正) → 清0 → 續打
  └ ==2 → 戰後跳表 0x51de9[章節] → 結局判定 0x2cad7 → 下一章跳表 0x51d71[章節]
```

## 6.1 turn_events.event_id → group 消費機制(已解,取代先前 [阻] 的 `0x22e5c` 猜測)

> **修正**:§8 先前把 `0x22e5c` 列為「turn_events 消費點,待解」。call-graph 驗證(`callgraph_le.py callers 0x22e5c`)
> 顯示它唯一 caller 是 `0x25de5`(戰役主迴圈:`[0x53ecc]==1` 時呼叫),且函式體內只操作固定的圖片/文字資源
> (`0x51a4d`、`0x520ba` 等常數位址),不觸碰 FDFIELD 控制段——**它是「第1章專屬、寫死的中場過場演出」**,
> 與 turn_events 資料無關。真正消費點在下面的 `0x1a813`。[驗]

**呼叫鏈(3 處呼叫點,camp 過濾)**:

| 呼叫點 | camp 參數 | 時機 |
|---|---|---|
| `0x1a4c7` | 1(ally) | 玩家(ally)回合結束後 |
| `0x1a554` | 0(enemy) | 敵方(AI)回合結束後 |
| `0x1a78d` | 2(special) | 同一世界地圖/戰場迴圈的另一檢查點 |

**`0x1a813(camp_filter)`**(turn_events 掃描迴圈):

```
0x01a813  ...
0x01a828  cmp ebx, 0x10            ; 迴圈 16 筆(FDFIELD turn_events[16])
0x01a831  mov ecx, [0x53a55]       ; ecx = FDFIELD 控制段 runtime 基底
0x01a83e  add eax, ecx             ; eax = ecx + i*3        (3B/筆,對齊 parse_field.py)
0x01a840  movzx edx, byte[eax+3]   ; edx = turn_events[i].turn   (ecx+3=跳過3B header)
0x01a844  cmp edx, [0x53bef]       ; == 目前回合數?
0x01a84a  jne next
0x01a84c  movzx edx, byte[eax+5]   ; edx = turn_events[i].camp
0x01a850  cmp edx, esi             ; == camp_filter(呼叫端傳入)?
0x01a852  jne next
0x01a854  movzx eax, byte[eax+4]   ; eax = turn_events[i].event_id ★
0x01a858  push 0
0x01a85a  call [eax*4 + 0x51b91]   ; ★★★ event_id → handler 跳表(58 entry,id 0-57 全域)
```

即:`turn==目前回合` 且 `camp==呼叫端 filter` 的記錄,取其 `event_id`,呼叫 `[event_id*4+0x51b91]`。
`0x51b91` 是**與 §1 的 `0x51b19`(章節×30)不同的另一張跳表**——**全域 event_id×58**,`jtab` 解出全 58 entry(0x341db…0x354dd)。[驗]

**event_id handler 內部(同 §2 結論:仍是硬編碼函式,非 byte-code)**:handler 呼叫兩個 spawn 原語:

| 原語 | linear | 作用 |
|---|---|---|
| `spawn_group(group_id)` | `0x10b4e` | 掃 FDFIELD 控制段 units 陣列(`[0x53a55]+0x83 + k*0x1a`,stride 26B=FDFIELD 單位記錄大小,欄位 `+0x15`=b21=group)找 `unit.group==group_id` 者啟用(offset `0x83`=3+48+32+48,正好是 header+turn_events+保留+chests,對齊 `parse_field.py` 的 units 起點)。[驗] |
| `spawn_group_with_intro(group_id)` | `0x32999` | 先繪 portrait(`0x51a4d`)+ 對話文字,再內部呼叫 `0x10b4e(group_id)`。用於敵方頭目類「先喊話再出場」。[驗] |

`group_id` 引數**通常是 handler 裡的字面常數**(如 event_id0 呼叫 `push 3;call 0x10b4e` 和 `push 7;call 0x10b4e` → 兩個 group);
少數 handler(event_id 27/54/57)用**動態值 `[0x53bef]`(目前回合數本身)當 group_id**——對應 `turn_events.json` 中同一
`event_id` 在連續多回合重複出現(如 map7/章8 的 event_id27 於 turn 2-7 各出現一次):每回合觸發同一 handler,
但 group_id=當下回合數,達成「每回合多放一波、group 編號＝回合數」的遞增增援。[驗]

**工具**:`tools/extract_event_id_groups.py` 走訪 58 個 handler 自身的 basic-block 鏈(call 過站不進入、遇 ret 停止,
避免線性 sweep 漂移),擷取 `push <group_id>; call spawn_group[_with_intro]`,輸出 `docs/data/event_id_groups.json`。
58 個 handler 中 20 個解出字面 group、5 個解出動態(`$turn_counter`)、其餘 33 個掃描視窗內無 spawn 呼叫
(推測為純對話/AI模式切換/目標判定類事件,無新單位進場)。[驗/推]

### 6.1.1 ch21/ch22 動態增援(event_id 47/49)eax 來源解密 [驗]

gen-campaign v4 report 留下 6 筆「`groups: [$reg_or_mem(eax)]`」——`spawn_group(eax)` 的 `eax` 來自暫存器計算,
非字面常數,擷取器放棄。反組譯 event_id 47(`0x35112`,ch21/map20)與 event_id 49(`0x351e9`,ch22/map21)的
handler 本體,兩者在 `call 0x10b4e` 前的計算完全同構:

```
0x03511c  mov  eax, [0x53bef]   ; eax = 回合數(turn counter)
0x035121  mov  edx, eax
0x035123  sar  edx, 0x1f        ; edx = eax 符號位擴散(正數→0,負數→-1)
0x035126  sub  eax, edx         ; 負數時 +1(修正無條件捨去→趨零捨去)
0x035128  sar  eax, 1           ; eax >>= 1(算術右移)
0x03512a  push eax
0x03512b  call 0x10b4e          ; spawn_group(eax)
```

這是編譯器產生「有號數除以 2」的標準慣用法(`sar edx,31; sub eax,edx; sar eax,1`),保證負數也趨零捨去;
`[0x53bef]` 恆為正(回合數 1,2,3…),故實際等價於 **`group_id = turn_counter DIV 2`(無條件捨去)**。
event_id 49(0x351e9)同一段位移量(handler+0x0a..0x12)逐位元組相同,同一公式。[驗]

**ch21(map0→map20)/ch22(map21)套公式,對照 ground truth(`remake/assets/maps/map20|21_units.json` 實際存在的 group)**:

| 章 | turn | event_id | 公式算出 group | map units.json 該 group 存在? | camp |
|---|---|---|---|---|---|
| 21(map20) | 2 | 47 | 1 | ✓(4 單位) | enemy |
| 21(map20) | 4 | 47 | 2 | ✓(4 單位) | enemy |
| 21(map20) | 6 | 47 | 3 | ✓(4 單位) | enemy |
| 21(map20) | 8 | 47 | 4 | ✓(4 單位) | enemy |
| 22(map21) | 3 | 49 | 1 | ✓(6 單位) | enemy |
| 22(map21) | 7 | 49 | 3 | ✓(6 單位) | enemy |

map20 實際 group 集合 `{0,1,2,3,4,255}`、map21 `{0,1,2,3,255}`——`turn/2` 算出的 1/2/3/4 與 1/3 恰好全部落在
存在的 group 內,且 map21 的 group2 對應另一筆已解出的字面事件(event_id50,turn5,camp=special,groups=[2],
own 陣營)不衝突,6/6 全部吻合。**已用 `tools/extract_event_id_groups.py` 的同款 basic-block walk 手動核對兩
handler 反組譯,非猜測**;`docs/data/turn_events.json` 對應 6 筆已把 `groups` 從 `$reg_or_mem(eax)` 換成算出的
整數,並補 `group_formula` 欄位記錄機制。

> 與 §6.1「5 個解出動態(`$turn_counter`,即 group=回合數本身)」是**不同公式**:event27/54/57 是 `group=turn`,
> event47/49 是 `group=turn÷2`——同樣的「回合數驅動遞增 group」設計母題,但除以 2 是因為這兩章每 2 回合才觸發
> 一次(turn events 只登記偶數/隔輪回合),group 编號仍要對齊「第幾波」而非「第幾回合」。

**map0/章1 ground truth 全部驗證通過(4/4)**:對照 `remake/assets/scenarios/ch01.json`(青衫核對過的正解)——

| turn_events(map0) | event_id | handler | 解出 group | ch01.json 正解 | 結果 |
|---|---|---|---|---|---|
| T3, camp=ally | 0 | 0x341db | {3, 7} | `hano_hawat_join`: groups[3,7] | ✓ |
| T4, camp=enemy | 1 | 0x342b5 | {4} | `enemy_reinforce`: groups[4] | ✓ |
| T5, camp=enemy | 2 | 0x3431d | {5} | `pirate_boss`: groups[5] | ✓ |
| T6, camp=ally | 3 | 0x34377 | {6} | `coast_guard`: groups[6] | ✓ |

group 數字、單筆 vs 雙筆(T3 兩組)、觸發回合、camp 全部吻合。`docs/data/turn_events.json` 已補上
`groups`/`handler` 欄位(全 30 章)。

## 7. 對重製(Go/Ebiten ScenarioRunner)的意義

- **原版事件 = 編進 EXE 的每章 handler**,無法像資料般直接搬。重製要嘛逐章反組譯 handler 邏輯重寫,
  要嘛(建議)用**資料驅動 DSL** 取代:把「條件(單位群狀態 / 回合數)→ 動作(增援 / 對話 / 勝負)」表達成
  campaign.json 的事件節點(對映 doc 19 腳本系統)。
- 已抽出的原語正好是 DSL 的詞彙:`when unit[i].flag` / `when turn>=N` → `spawn` / `dialogue` / `win` / `event`。
- default handler(11 章)= 最簡單的「殲滅即勝」,重製預設規則即可;18 個特殊 handler 是需要逐關重建的事件腳本。

## 7.5 戰場單位有兩個來源:FDFIELD roster vs 事件進場(2026-06-28 證實)

掃全 12 關 FDFIELD `own`(己方)roster,**沒有任何一關含索爾(id0)/亞雷斯(4)等主角**。
**第1章「初試身手」= 海邊第一戰 = map0**(敵方 portrait 盜賊96 + 海盜頭目97 + 海防/援軍76 + 103;
與青衫第1章敵方完全對應),其 own roster **只有哈諾(1)+哈瓦特(3)**,且這兩人也不是第一回合就在場
(青衫:第3回合己方結束才從房子出來)。索爾/亞雷斯/妮雅/蓋亞**完全不在 roster**。
→ 戰場單位是**雙來源**:

| 來源 | 內容 | 機制 |
|---|---|---|
| **FDFIELD roster** | 每關的敵人 + 部分配角/ally(各關 map<N>) | 開場擺位 **或** 由事件按回合放出(哈諾/哈瓦特雖在 roster,T3 才進場) |
| **隊伍名冊 `[0x53bf7]` + 事件** | 玩家主角隊(索爾/亞雷斯/妮雅/蓋亞) | **由 pre-battle cutscene / 事件腳本動態進場** |

**第1章開場演出(實機)**:索爾/亞雷斯/妮雅/蓋亞**從戰場邊緣移動進入中央 → 觸發對話**,
之後才進入可操作戰鬥。即第一戰不是「擺好棋子開打」,而是 scripted 入場 cutscene。
全關進場時序見 doc 28 第1章(青衫 ground truth):開場主角隊 → T3 哈諾/哈瓦特(友) → T4 敵援軍(右下) → T5 海盜頭目+屬下(左下) → T6 警備隊/海防隊(右上,友,立即行動)。

> ⚠ 更正(2026-06-29):前一版誤把第一戰寫成「序章 ch0=map2」。**第一戰是 map0**;map2 敵方為 [76,77],屬另一關。map↔章節對應應以**各關敵方單位特徵對青衫各章**核對,別套「map=章節×3+2」公式(未對齊實際關卡)。

**對 remake 的意義**:`export_units.py` 目前只導 FDFIELD roster,**玩家主角隊要另從隊伍名冊注入**,
且 roster 內單位也可能帶「進場回合」(哈諾/哈瓦特 T3);第一戰要實作「單位從邊緣走入 + 對話」的進場演出
(事件腳本 DSL 的 `spawn`+`move`+`dialogue`+`when turn>=N` 節點),不能只靠 map_units.json。

## 7.5.1 序章主角隊進場 staging:**戰場進場**直接定位,**cutscene 幕內**另有走位(2026-07-04 範圍修正)

> **範圍修正(2026-07-04,doc46 開場時間軸影片證據後)**:本節原標題/結論「無行軍動畫」**範圍過大,需修正**。
> 下方 2026-07-03 的反組譯+dosbox 複驗證實的是**「戰場進場(map0 spawn)= 直接定位」**——這條在
> `docs/knowledge-base/46-ch1-opening-timeline.md` §4(模板匹配 remake `battle_ch01` 開局位置)**再次複驗仍成立**,
> 沒有推翻。**被推翻的是「序章全程無走位」這個更廣的推論**:doc46 用影片逐幀證據(0.5 秒間隔連續抽幀)
> 找到至少兩處明確的**跨多幀角色位移**,發生在**序幕 cutscene 畫面本身**(map31/map32 複合場景,非戰場):
> ①王座廳對白說完,索爾 sprite 沿紅毯走下場(~1.5 秒,`ch1_trans_t1_sheet.png`)
> ②後山密林「比劍邀約」轉「發現悠妮與蓋亞」之間,索爾+亞雷斯用多幀畫面逐步接近悠妮/蓋亞
> (FDFIELD 出場位置證實兩組座標相距 14 格,非同格瞬移,`ch1_trans_t4_sheet.png`)。
> **這兩處走位用的是哪個 EXE 機制,尚未重新反組譯**(不在下方 0x3231b 三原語表的已知範圍內,
> 可能是同一 handler 內未逐一展開的呼叫、也可能在其他尚未追的子程序);remake 對應實作見
> `remake/internal/campaign/campaign.go`(`Actor.FromX/FromY/WalkFrames` 進場走位、`Node.ExitWalk` 退場走位)
> 與 `remake/cmd/fd2/main.go`(`storyWalkJob`),重用既有 `OffX/OffY` 插值,不等待 EXE 機制查清才動手
> (影片已是可直接落地的 ground truth,見 doc46)。**下方 2026-07-03 內容保留原文(歷史記錄),
> 但讀者請以本段範圍修正為準**:「直接定位無行軍動畫」只在**戰場單位進場**成立,不適用於**序幕
> cutscene 畫面內**的角色走位。

playtest 反饋 #3 指出「序章劇本 staging 機制沒 RE 完整」,且使用者記憶「索爾一行人一開始走到地圖中央」。
本節用**靜態反組譯 0x3231b 本體 + dosbox 全程重跑序章開場**兩路收斂,把 §7.5 的「事件進場」講清楚「怎麼進場」。

**靜態反組譯(`tools/disasm_le.py range/dis`,`fd2-cap` docker)**:章節0 handler `0x3231b`(跳表 `0x51d71[0]`)
本體是一長串**線性**呼叫序列(對白段 `0x1366a` + 場景重繪 `0x15f84` + 少量特效),逐一過站無分支迴圈,
其中出現三種「群組登場」原語,語意各不相同:

| 呼叫 | 語意 | 對單位座標的影響 |
|---|---|---|
| `0x10b4e(group_id)` | **直接 spawn**(第1章序章內見於 group 1/3/5 等) | 無——單位直接出現在其 FDFIELD/roster 定位,無任何逐幀移動 |
| `0x13185(unit_idx)`(序章開場呼叫 2 次迴圈,共 15+13=28 次) | **攝影機平移**(讀寫 `[0x53aa9]`/`[0x53aad]` camX/camY,doc 25 §7.6 同一對變數) | 無——單位不動,是**鏡頭**在移動;每次呼叫只把攝影機原點挪一格,配合後續 `0x15f84` 逐 frame 重繪 |
| `0x32999(group_id)`(序章呼叫 2 次,group 1/2)`spawn_group_with_intro` | 先用 `0x111ba` 載頭像+`0x1366a` 播對白(「先喊話再出場」),內部再呼叫 `0x10b4e` spawn,**接著逐一掃描新 spawn 的單位**,若座標落在目前攝影機視窗([0x53aa9]/[0x53aad] ± [0x51a87]/[0x51a8b])內就用 `0x4e85b` blit 出來,最後 `0x11eb0` present | 無——同樣是單位**已經**定位好,只是「有沒有進攝影機視窗」決定畫不畫;視覺上像「隨鏡頭捲動逐個冒出來」,但每個單位本身沒有位置插值 |

**三者共通點:全程沒有任何「單位座標逐幀 +1/路徑插值」的迴圈。** 移動的是攝影機,不是單位精靈——這與
doc 35 攻擊演出「無 runtime 縮放/無翻轉,景深燒在素材」的結論同源:**原版能省的動畫運算一律省,
靠鏡頭運鏡或素材本身,不做角色位移插值。**

**dosbox 實機複驗(220+ 張連拍,`extracted/story/staging_dosbox/seq/`,本機不入庫)**:全新遊戲 → 標題 START →
throne room 父子送別 → 黑幕轉場 → 草地小憩(索爾/亞雷斯對話,`proof_01_field_rest.png`)→ 比劍邀約(兩人
靠近,暗轉,疑為另一段小型過場非本節重點)→ 悠妮/蓋亞加入(失憶對話「我們是從哪裡來的?」)→ 是否赴
馬拉大陸的爭論 → 海盜堵路對峙(`proof_02_pirate_prebattle.png`,索爾/悠妮/蓋亞已在最終戰鬥位置,3 海盜
+1 機械兵已在各自位置)→ 指令環開戰(`proof_03_battle_command_ring.png`,HP/MP 狀態欄出現,單位位置與
上一張幾乎相同)。**逐幀比對:每個場景切換都是「背景/對話框瞬間換」的硬切,場景內單位座標在連續多張
截圖間完全靜止;從對峙畫面到開戰指令環,單位位置沒有位移**——與靜態反組譯結論一致:主角隊(及本章
遇到的海盜/機械兵)都是**直接定位**,沒有觀察到任何「單位行走/行軍」動畫。

**結論(2026-07-03 原文;§7.5.1 開頭範圍修正已標記何處不再成立,見上方)**:
1. **remake 現行做法(main.go focusOnParty 純鏡頭對準 + event.go spawn_party 直接定位)已忠實**,#3 **不是 bug**,不需要補行軍動畫。——**此點對「戰場進場」仍成立**;但序幕 cutscene 本身(map31/32)需要走位動畫,已於 Phase 2 補上(見上方範圍修正段)。
2. ~~玩家記憶「一行人走到地圖中央」查無實據~~ **此點已被 doc46 影片證據部分推翻**:序幕 cutscene 內(非戰場)確實有角色跨幀走位(見上方範圍修正),雖然仍不是「世界地圖/道路移動」那種大範圍場景,但也不是「鏡頭動、人沒動」的錯覺——原文「最可能的解釋是攝影機平移錯覺」這個判斷本身是錯的,以 doc46 為準。
3. 先前 event.go 註解一度誤引「doc 25 §7.5.1 dosbox 實機證實[世界地圖走位]」,但該小節當時並不存在——本節即補上正確內容並修正該註解的引用鏈(§7.5.1 = 本節,無世界地圖佐證)。此點不受本次修正影響。

## 7.6 戰場視窗固定 13×8 格,原版無「地圖比視窗窄」的清背景邏輯(2026-07-03,反組譯)

remake 內部畫布 640×400(2x hi-res,tile 維持原生 24px),map0(24×24 格)在此畫布下比可視寬度窄
(576<640),右緣露出「畫面外」的區域;為了確認 remake 該填什麼色,反組譯原版戰場重繪鏈:

- **主重繪函式 `0x11cac`**(每幀呼叫,`0x10010`「戰場設置」進場後的主迴圈 call):依序呼叫
  `0x1297d`(捲動動畫計數器)→ `0x11eee`(地形層)→ `0x122dc`(移動範圍高亮疊圖)→ `0x127a9`(單位層,
  含 `0x129ec` 收尾)→ `0x1acf3`(游標/UI)→ `0x11eb0`(present)。work buffer stride **0x1c8=456**
  (與 doc 35 攻擊演出的 0x280=640 不同,tactical 另有獨立合成 buffer)。
- **地形層 `0x11eee`**:對「一般章節」(排除 9/0x18/0x19/0x1c/0x1d 世界地圖類與 0x11/0x15/0x16/0x17/0x1b
  過場標題類,那幾類走不同的 BG 貼圖分支),直接落入逐格迴圈(inline,0x12164–0x1222f):
  ```
  for row in 0..<8:              ; 高度計數硬編碼 8(0x011cd4 push 8)
    for col in 0..<13:           ; 寬度計數硬編碼 0xd=13(0x011cd6 push 0xd)
      idx = (row+camY)*[0x53ac1](地圖寬) + (col+camX)   ; camX/camY = [0x53aa9]/[0x53aad](捲動原點)
      entry = FDFIELD[idx*4 + [0x3a51]]                  ; [0x3a51] = FDFIELD.DAT 載入基底(le_xref 驗證)
      call 0x4deda / 0x4dcc6(dst, tile_sprite, stride)    ; 無條件 blit,兩者僅差是否轉透明色 0xFF
  ```
  **13×8 格 × 24px = 312×192px**,與 present 呼叫的視窗尺寸(`0x011d12 push 0xc0` / `0x011d17 push 0x138`
  = 192/312)吻合 → **原版戰場視窗永遠固定 13×8 格,不隨地圖尺寸縮放**。
- **關鍵:整個地形迴圈裡沒有任何 `memset`/fillrect/rect 原語**(比對戰鬥演出 doc35 §3 同一結論:
  「無 fillrect/circle」)——每格永遠呼叫 blit,**沒有「地圖格不存在就清某色」的分支**。
- **驗證:原版全 34 張地圖沒有一張窄於視窗**(`extracted/maps/maps_metadata.json` 全量掃描:
  最小寬 18 格 map2、最小高 20 格 map3,皆 ≥ 13×8)。→ **「地圖比視窗窄」在原版從未觸發過**,
  因此原版壓根沒有為這個情境寫清背景/邊框邏輯,不是「找不到」,是**這段代碼從未被需要過**。
  remake 會撞到這情境,純粹是 remake 自己選了比原版視窗寬的 FOV(640px、tile 仍 24px 原生尺寸
  ≈ 26.7 格可視寬,大於任何原版地圖或原版 13 格視窗)——**這是 remake 設計決策造成的新情境,不是
  移植失真**。
- **remake 對齊**:`cmd/fd2/main.go` 的 `screen.Fill(黑)` 在地圖繪製前打底,合理(無原版行為可循,
  黑色純粹是視覺乾淨的預設,非「原版就是這樣填」)。已截圖確認(`extracted/remake_shots/map0_edge_test.png`)
  map0 右緣為乾淨黑邊,非殘影黃白。

## 8. 受阻 / 待續

- **[已解,見 §6.1]** ~~`turn_events.event_id → group` 對應機制(先前疑 `0x22e5c`,未解)~~ →
  真正消費點是 `0x1a813`(3 呼叫點 camp filter)+ 全域 `event_id` 跳表 `0x51b91`(58 entry)+
  spawn 原語 `0x10b4e`/`0x32999`;`0x22e5c` 只是第1章專屬固定過場,與 turn_events 無關。
  map0/章1 ground truth 4/4 驗證通過,`docs/data/turn_events.json` 已補 `groups` 欄。
- **[已解,見 doc 26]** ~~18 handler 逐章語意 + 動作函式~~ → 全挖完:handler 無動作函式(只條件→設碼+繪圖);條件原語 `unit_flag`/`roster_has`/回合;機器可讀 `docs/data/battle_events.json`。
- **[修正]** byte(+5)bit0 **非陣亡**(初始化=1);= 狀態旗標(存活/在場?)語意待驗。回合數=`[0x53bef]`(非 `[0x53ec8]`,後者為累積計數)。詳見 doc 26。
- **修正 doc 24**:§6 稱「事件腳本解譯器(大函式 0x205c9–0x20c64)」用詞不精確 → 實為**章節戰場事件 handler 表 0x51b19,各 handler 在 0x205b4–0x20bf5**(非單一解譯器,非 byte-code)。已於 doc 24 §6.3 附註。

> 相關:doc 23(三大狀態 + 兩跳表)· doc 24(戰役迴圈 + [0x53ecc] 狀態機)· doc 19(腳本系統設計)· doc 11(AI)。工具:`tools/callgraph_le.py`、`tools/disasm_le.py`。
