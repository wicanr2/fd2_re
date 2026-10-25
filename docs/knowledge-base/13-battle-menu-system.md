# 13 — 戰場選單與行動系統

> 戰棋上選一個我方單位後,怎麼移動、怎麼跳出行動選單(攻擊/法術/道具/待機/狀態)、
> 游標怎麼操作、一個單位的「一回合」怎麼算完。第 3 輪反組譯 `FD2.EXE`。

## 一個單位的回合:行動狀態機

以下「AA／一回合」段落是早期 gameplay 摘要，並非目前已閉合的 native UI contract；現行 action chooser、
command grid、end-turn 與 campaign transition 以 SDD56／UI matrix57 的 raw evidence 為準。舊文件把行動狀態
誤掛到 `AA(0x0D)`；目前只保留已驗證的 record byte `+5` bit0／bit7 raw predicates：

```
+5 bit0: admission test `(+5 & 1)==0`／reject test `(+5 & 1)!=0`（raw mask only）
+5 bit7: test/set/clear `(+5 & 0x80)`（raw mask only）
```

回合流程:

```
已觀察的 caller fragments：raw `+5` mask gate → 可能的移動／target helper → action chooser；
各段是否必然串成完整回合、何時呼叫 writer，仍待 current UI/campaign evidence。
```

> 攻略 modify1 揭露的行動規則(改的就是這套狀態機):
> - #5/#6「行動後可再行動」的攻略說法不能直接映射成 raw writer；是否設 `+5 bit7` 需看具體 handler。
> - #7「移動後可施法,隨時可存檔」是攻略層摘要，不能當 native end-turn 或 command gate 證據。
> - 動作完畢旗標 `0x80` 的檢查見 `0x12717`(`test byte [..+5],0x80`)。

## 行動選單選項

早期「典型戰棋的移動／攻擊／法術／道具／待機／狀態」列舉不是原版證據，已刪除。原版 action
chooser 現有 E0 的四項只可寫為 `↑=attack`、`←=command`、`→=item`、`↓=wait/field`（見下方
`0x18d8c` switch）；各 action 的完整 availability gate 仍須按各 callee 分開證明，不能以通用 RPG
假設補齊。

### `Get_EasyMagic` — 法術 / 狀態面板(已反組譯 `0x18ED0+`)

EXE 內洩漏的函式名 `Get_EasyMagic`,位於 `0x18ED0` 起。舊文件曾把 runtime `unit+0x22..0x26` 誤標成 **5 組法術 bitfield**；constructor direct trace（`0x10f6b..0x10fa5`）已推翻此說：magic raw 是 `unit+0x1a..0x1d`；`+0x22..+0x27` 目前只保留 raw transient／modifier bytes，不能由局部 writer 直接命名成 AP/DP/HIT 或 status。以下只保留函式名與 UI caller 的已證實部分，法術欄位對映維持 partial：

```
native magic raw / menu state:
    由 `+0x1a..+0x1d` 與 `0x1cff0` command path 產生可選項
    （`+0x22..+0x27` 不可當法術 bitfield，也不自動等於 named stats/status）
```

→ 即玩家按「法術」時看到的面板：可用 command 與目前 HP/MP。繪字／數字用 `0x195D6`（在 320 寬 buffer 定位繪製）。可用 command 的 label、spell-family、個別 spell ID 對照仍待完成，不得由 raw bit 或舊 M1–M5 名稱推導。

## 游標 / 鍵盤操作(已反組譯)

### 2026-07-25 Docker Capstone recheck: `0x18890` action dispatcher

`0x18890` 是「單位移動後進入 action dispatch」的 caller，而非單純繪圖 helper：它以 `ebp` 計算 `[0x53a45] + unit*0x50`，讀 unit context，呼叫 `0x18d8c(ebp, ..., mode)` 取得 action result；result `0` 繼續等待，`-1` 走取消／返回，其他 result 進入 `0x13a44` target/action path。它同時串接 `0x13488` path-walk 與 `0x134e4` 完成通知，移動後重寫 unit X/Y bytes，再由 `0x13512`／`0x13a44` 進後續 action。

這段 caller 沒有單獨證實 `0x18d8c` 的選項表與 result 編號，因此撤回「`0x18890` 只是選單繪圖」類推；目前 remake radial 方向 mapping 仍不得宣稱原版一致，直到 `0x18d8c` 本體和 menu/resource table 完成交叉引用。

### `0x18d8c` switch closure（2026-07-25, Docker Capstone）

`0x18d8c` 讀 `[0x3c57]` 後的四個分支已釘死：

| `[0x3c57]` | 方向 | native callee / 行為 | remake 對應 |
|---:|---|---|---|
| 0 | ↑ | `0x1f04a` → `0x28a6c`：攻擊目標／全螢幕攻擊演出 | 攻擊（case 0） |
| 1 | ← | `0x1cff0`：配置 320×200 演出 buffer、法術 command／演出 loop（`0x1c269` 失敗會 disable） | 法術選單（case 1） |
| 2 | → | `0x1bbdc`：item action loop；case 0 讀 item record `+0xd/+0x10/+0x12/+0x15`、做 target geometry，進 `0x20c6f` effect/target path；case 1 transfer uses `0x1bb8c` insertion + `0x1b8e7` removal；case 2 `0x1bffe` equips via `0x1c1c3` compatibility + `0x1c142` slot flags | 物品（case 2） |
| 3 | ↓ | `0x13fd4`（current HP≠max HP 且 raw `+0x25/+0x26==0` 時回復 `floor(maxHP/5)`）→ `0x190ac` 格子互動／寶物檢查 | 休息回復／格子互動（case 3） |

因此撤回舊 mapping「↑道具／←攻擊／→魔法／↓待機」。`0x1cff0` 的 command-`0x1e` spell path 已證實；family priority、damage/effect jump table 仍待完整拆解，本表不宣稱所有 effect 已完成。

### `0x1cff0` command evidence（2026-07-25, Docker Capstone）

`0x1cff0` 會以 `[0x3c57]` 選中的 byte 從 local command array 取出 `0x4e516` record。修正舊筆記：MP gate 是 record `+5`（`0x159fa` 的 direct compare），不是 `+4`；`+3/+4/+6` 在本 selector 的 geometry/control path 都有讀取，但未追到 field-name 前仍只能保留 raw offsets。已釘死的分支：command `0x17` 走特殊 target geometry（`0x14818`，使用 record `+6`）；command `0x1e` 走 `record+3-0x10` → `0x149f8` 的 spell-family path；其他 command 走 `0x2a6bd` 或 `0x1d6c8` jump-table effect path，最後統一回 `0x1aa1d`／`0x1d4f6` 收尾。

官方 IDA 9.4 的 `0x1d3f3` dispatch 再釘細一層：target confirm 後 command ID `0..8`、`0x18`、以及 `>=0x1c` 直接呼叫 `0x2a6bd(unit, commandID, target, scratch)`；ID `0x09..0x17` 與 `0x19..0x1b` 則先以 `0x1d6c8(commandID)` 做四輪 palette flicker，再呼叫 `funcs_1541f[commandID]` effect jump table。故悠妮 source mask 的 command 0 已能證實進 generic effect pipeline，卻尚未證實其效果名、傷害公式或等同 legacy `Spells[0]`；remake 對 unknown effect 必須繼續 fail-closed。

後續 direct dataflow 已閉合 command 0 的數值 writer：`0x2a6bd` 對每個 target 呼叫 `0x1c75e(target,commandID)`；它讀 command record `u16+0`／byte `+2`，以 target class ID（constructor 寫入 unit `+0x20`）索引 `word_51f96` 形成 base。命中擲骰是 `0x1c7ed call 0x4e893` 後取 shared uint16 state `%100 < record[+2]`；命中後 `0x1c81f` 在 `0x1c869` 再呼同一 RNG，以 90% 加 0..9.9% random 的整數值扣 target `+0x40`，clamp 0。舊 adapter 使用 Go `math/rand` 是錯誤替代，已刪除。`word_51f96` 的 loaded file offset=`0x51d96`，就是 `resist_crit.json` 每職業的 `resist_raw`（每 row 4 bytes）；因此 base=`record.dmg*resist_raw/10`，等價於既有職業魔抗百分比。這是 native damage/HP ABI；command0 effect name、target family與 normalized spell equivalence 仍未證實。

同一條 caller dataflow 排除了「confirm 只傳回一個格子」的猜測：一般 record 先以 actor cell 的 `+3/+6` 呼叫 `0x14818` 產生可選中心 stack candidate-index array，`0x115b6` 接收 `(mode=+6,count,array)` 做游標／確認；成功後 `0x1cff0` 以**確認游標格**的 `+4/+6` 第二次呼叫 `0x14818`，`0x2a6bd` 接到的是這個第二 array/count，並逐 element 呼叫 `0x1c75e`。故 command0 有 per-final-effect-candidate 傷害；`+6` target-code 與 `0x14818` geometry 的現行 direct closure如下。現有單格 normalized target UI 仍不是原版證明。

`0x14818` 本體現已釘其 raw geometry：record `+3 < 0x10` 時交由 `0x4e555` 產生 map/reach mask；`+3 >= 0x10`
時只 mark 同 x 或同 y、距離不超過 `(+3-0x10)` 的十字格。接著掃 roster，略過 inactive unit 與 mask=`0xff` 格，再用 record `+6` 篩 runtime `unit+6`：code 0 要 `==0`、1 要 `!=0`、2 要 `==1`、3 要 `==2`。`0x10c50` constructor 已證實 `unit+6` 是 FDFIELD `b0` camp（0敵/1友/2己）；故 code 0=敵、1=非敵、2=友、3=己。**2026-07-28 correction：**先前把 code2 寫成 `!=1`／非友是分支方向抄反，已由 `0x149b0..0x149be` 與 `0x147d5..0x147e1` direct Capstone 撤回。

上述 `dist<0x10` 不是無條件 Manhattan：`0x4e555(0)` 回傳 20-byte cost row，真正的 `0x4e040` 以 `dist`
做四方向 flood-fill，grid flag `0x40` 不可通過；`0x80` 不是零成本，`0x4e19a`
會在扣除 terrain cost 後把該格的 remaining-budget byte 強制寫成 0，因此該格可達但不能形成
zero-cost chain。對 command selector 而言 row index 固定
為 0，EXE `word_61646` 的該 20-byte row 全為 1；故不套地形加權，但障礙仍可使可達格少於 Manhattan circle。
remake `InCastRange` 仍不能作為 native contract，除非也帶入同一 grid flags。

`0x149f8` 的本體語意是 target-candidate builder：依起點／方向步進 `count` 次，做地圖邊界檢查，呼叫 `0x12c0d` 取 unit，依 selector 篩選 `unit+6` 狀態後把 unit index 寫入輸出陣列。一般 AI caller 的 selector-family 對照仍待 RE；但玩家 ID30 已由 `0x1d339` 的完整 push 順序閉合為一條不同的 special line（如下）。

`0x115b6` 的 confirm loop 已直接讀 `0x53ab1/0x53ab5`，並以兩值比對 runtime unit `+0/+1`；方向鍵 helpers 改寫同一對 globals。因此它們是這條 selector 的 cursor cell，而不是相機 scroll。ID30 的 `0x1d287..0x1d354` 先以 record `+3=0x14` 的 normal `0x14818` candidate list 確認一個 enemy cursor，保存確認前 cursor `(savedX,savedY)`，再傳確認後 cursor `(confirmedX,confirmedY)` 給 `0x149f8`。callee 的七個 args 已可靜態對應：output、confirmed X/Y、saved X/Y、`count=record+3-0x10=4`、selector=1。它從 saved cell **先走一步**，若 X 不同則只走 X（savedX>confirmedX 為 −1，否則 +1）；僅 X 相同才以同規則走 Y；兩座標相同落入原始 `<=` 的 +Y branch。每一步由 `0x12c0d` 取得第一個 active unit，selector=1 僅收 native camp=0（Go `Enemy`），越界只是略過。故 ID30 是「保存游標→確認游標」的最多四格 cardinal line，不是 actor→confirmed 的 generic ray，也不走第二次 `0x14818`。`ExecuteNativeCommand30` 保留顯式兩cursor規則；正式玩家入口已接保存游標、確認、原生／HD分段打擊及FDOTHER49音效，受控繪製、F2與保存冷讀通過。現況及重跑入口見[58](58-fd2-exe-re-coverage.md)、[61](61-modern-theme-runtime-spec.md)與[主證據](../data/ida/fd2_command30_player_acquisition_20261025.json)。

### `0x1bbdc` item action evidence（2026-07-25, Docker Capstone）

物品 action 不是「完全不存在」：`0x1b932→0x1b9de` 是八格 raw
inventory 的 **兩欄×四列** selector。它只計算 signed flag非負的 occupied
prefix；原版 insertion/removal 維持 compact prefix，不把 raw hole 畫成
固定空白列。↑/↓沿 prefix linear wrap（slot3→4會跨欄），←/→以±4切欄；
Enter/Space confirm、Esc cancel。battle-use mode另外拒絕 selected item
row `+0x0d==0`，非 battle mode不套此 gate。`0x1bb8c` 只把 item 插入
第一個空槽；case1 與 `0x1b8e7` 串成 transfer。case2 進 `0x1bffe`，
由 `0x1c1c3` compatibility 與 `0x1c142` 設 `0x40` equipped flag，再
呼 `0x1b750` 重算。case0 的 target/effect ABI現已閉合；remake item
Enter仍未接 transaction。

`0x184c0` 的 original geometry 也已固定：display index `n` 的 label
在 `x=42+150*(n/4), y=103+22*(n%4)`，第五筆從 `(192,103)` 開始；
category icon在 `x-29,y-2`，item文字取 FDTXT index `itemID+181`，
selected/unselected color raw 201/205。row byte0 `<0x15`／`<0x20`／其他
選 icon59/60/61，equipped再+3；右側 stat icon/value 依同 byte與
type5/11 分流。`NativeItemSelectorCells` 保存 compact layout與raw icon
IDs，但 indexed blit/字型與 opening/closing animation 尚未接 GUI。

opening/closing schedule 本身現已閉合：`0x17e0b` 依 frame11→0 開啟，
`0x1b932` 依0→11關閉，每幀都先從 saved 64000-byte buffer重建，再由
`0x18409` 組三塊 320-stride region並present。left panel來源
`(5,7,86,86)` 在 frame6後每幀向左clip16px，frame11只剩11px；
upper來源 `(92,7,223,86)` 在 frame3後向上clip16px，frame9起消失；
bottom來源 `(5,94,310,102)` 從 y94每幀下移16px，frame6起消失。
`NativeItemPanelSchedule` 保存十二幀與exact clipped rectangles。
`0x17eef` 的 source construction 也已閉合：先以
`0x168b6(dst,320,5,7,5,5)` 在 `(5,7)` 建 5×5 框，unit record `+7`
選 DATO portrait並貼到 `(8,10)`；`[0x53a81]+86/+90` 是 FDOTHER #5
LMI1 directory entries 20/21，分別貼到 `(92,7)` 與 `(5,94)`。
`0x17fc0` 的兩條 bar、四個 compared-number、八個 raw-number、三段
FDTXT及四組 icon destination/record-offset schedule 已由
`NativeItemPanelBaseLayoutFor`／`NativeItemPanelDataPlanFor` 資料化並測試。
`RenderNativeItemPanelResources` 已以玩家 FDOTHER/FDTXT/DATO archive完成
corrected 49-cell grid、portrait frame0、entries20/21，以及 `0x17fc0`
bar/digit/icon/FDTXT dynamic overlay，整張 atomic commit。三條 codec
嚴格分離：raw opaque cells、four-mode transparent frames、FDOTHER#4
1bpp font。`RenderNativeItemPanelRows` 另完成 `0x184c0` compact rows：
category/stat icons、item FDTXT `ID+181`、selected color201、unselected205
與 stat value座標。可重現 oracle 見
[`item-panel-native-indexed.png`](../figures/item-panel-native-indexed.png)。
Ebiten runtime現以 `NativeItemPanelRecordForUnit` 嚴格要求 raw `+6/+8`、
`+0x1f/+0x20`、DATO selector與8格inventory provenance，接完整 compositor、
compact ↑↓/←→ input及 opening11→0/closing0→11 clipped presentation；
缺證據／缺archive即回 legacy shell。`0x112a5` 已證實 JOIN id直接選
lower constructor record，寫 `+0x1f/+0x20`；轉職只改 `+0x20`與`+7`。
30章scenario、persistent overlay及class-change writer現保留這組獨立raw
provenance，正常ch01 campaign integration已可準備原版面板。現在剩 item
Enter的大多數 effect/target transaction；尚未獨立證實的 raw offsets 仍不命名。

第一組完整 Enter transaction 已接：tracked IDs198/199/200 的 type8/9/10
與 IDs94/95/96 的 type17/18/19 均以 row selection/effect mode0、
target code1 經兩段 `0x14818` 驗證 actor self-target。前組依 `0x21082`
對 raw base AP/DP/DX 做16-bit wrap加值；後組對 MaxHP/MaxMP/MV 加值，
其中 MV保存相鄰EXP byte，Max值增加不回填current值。兩組都依
`0x1b8e7` compact移除來源。成功後 `0x1bbdc` 直接呼 `0x13512`，故 runtime
同時設 raw `+5 bit7`、normalized `Acted`並退出action；不是回到item menu。
AP/DP缺 equipment-base provenance時整筆拒絕，不以有效值冒充raw base。

`0x20c6f` 已再以 Docker Capstone 展開：它依 item `+0xd` type 分派至多個原生 effect routines（例如 type `5/0xd→0x211a4`、`6/7→0x22af6`、`8/9/0xa→0x21082`、`0xe/0xf/0x10→0x22d1b/0x22866/0x22721`、`0x15→0x2111a`、`0x17→0x2218a`）。其中 type5/13 已定案為以 row `+0xe` 恢復 target-list HP：type5 隨後經 `0x1b8e7` 消耗來源 slot，type13 不移除來源；這是 effect 與 consumption contract，不推測道具顯示名稱。其餘尚未閉合的 routine 仍不可直接映射成藥水／卷軸規則。

其中 type `8/9/0xa→0x21082` 已由 raw table 全表對齊與 `0x1145a`
base/equipment data flow 閉合：item row `+0xe` 分別永久增加 persistent
base AP(`+0x37`)、DP(`+0x39`)、DX(`+0x3e`)，呼叫 `0x1b750` 重算後移除
來源 slot；已知 IDs 198/199/200 的量為 9/9/7。顯示 selector 仍未命名，
也不能把此語意套到共用 callee 的 type17–19。type6/7 已閉合成
consumable marker-clear HP restore：分別讀 target runtime record
`+0x25/+0x26`；nonzero 才以 base amount10 經 `0x1c916` 實際恢復
9 HP、清同一 record marker，最後消耗來源 slot。tracked IDs196/197。
舊 adapter 將 marker 當 parallel `flags[]` 的實作已修正；status 名稱未知。

type11 已由同一 dispatcher loop 閉合為 MP restore consumable：row `+0xe`
是 amount，target max MP `+0x46==0` 時跳過且不前進 RNG；其餘依 list 順序
由 `0x1c9dd` 增加 current MP `+0x44`、cap `+0x46`，最後經 `0x1b8e7`
消耗來源 slot。已知 raw IDs206/207 amounts=80/200；顯示名稱仍不猜測。

type12 則重用 `0x22997`：target marker `+0x24` 非零時跳過且不耗 RNG；
成功時寫 `(rng%4)+2` 並把 derived HIT/EV `+0x4c/+0x4e` 各加 15。
dispatcher 直接 cleanup、不移除來源 slot。raw fixture 目前只有 ID210
走此 type；道具名稱與 marker 的 UI 顯示仍未閉合。

type15/16 也是 retained modifier：type15 以 marker `+0x23` gate，將
derived DP `+0x4a` 增加 `trunc(DP×0.15+1)`；type16 對 marker `+0x22`
與 derived AP `+0x48` 做同式。marker 已存在就不耗 RNG，兩條 dispatcher
都不移除來源；tracked rows 是 ID213/214。

type14/22 共用 `0x22d1b` 且保留來源。marker (`+0x26/+0x27`) 必須為零、
class 不能是 `0x19/0x1a`；第一 RNG `%100<50` 才呼
`0x1c81f(target,10)`。注意 10 是 damage base amount：`0x1c81f` 再消耗
第二 RNG，原生整數公式在 amount=10 時實際減 9 HP；第三 RNG 才寫
`(rng%4)+2` marker。tracked rows 是 ID212/57。marker 的 status 名稱未知。

type17–19 雖共用 `0x21082`，欄位已由獨立 producer/consumer 閉合：
type17 max HP `+0x42` +20、type18 max MP `+0x46` +20；type19 對
`+0x3b` 做 word +1，但 caller 保存並恢復 `+0x3c` EXP，因此結果只增加
MV byte、EXP 不變。三條都由 callee 移除來源 slot；tracked IDs94/95/96。

type20/21/24 的 row word 不是恢復量或 MP 扣值，而是交給 `0x1c75e`
的 command ID。type20/24 先走 `0x1cd17` 十幀 indexed presentation；
type21 由 `0x2111a→0x1cac7` 走另一套 presentation，之後三者都依
target list 呼叫既有 command damage writer。dispatcher 不呼叫
`0x1ca89`、也不移除來源 slot。tracked type20 IDs11/56/60 對 commands
2/0/2，type21 IDs29/38/51/99 對 6/1/7/6，type24 ID79 對 command3。
先前把 `0x1cac7` 說成 MP subtract 是地址混淆，已撤回；真正的 command
MP debit helper 是 `0x1ca89`。

type23（`0x17→0x2218a`）也已閉合 post-confirm transaction。item selector
要求 actor runtime `+0x08==24` 且 max MP `+0x46>=20`；舊稱
class/level gate 是錯的。handler 只取 target list 第一筆，以
`0x1ca89(actor,command23)` 直接按 command record `+5` 扣 current MP
（原生 16-bit wrap），再依 target class `+0x20`／level `+0x21` 增加
raw accumulator。兩次 `0x22253` 先以原座標演出並寫 `0xff/0xff`，
再寫 destination cursor globals 至 target `+0/+1`；不走 path traversal，
dispatcher 也不移除來源物品。tracked item ID101 的 row word1 不被
handler 使用。`NativeItemRelocationRoute`／executor 保存這些 state
mutation。mode-6 落點 predicate 也已資料化：除 selected target 外，
任何同座標且 raw `+5 bit0==0` 的 record 都阻擋；movement selector
通常取 target class `+0x20`，`+7==0x1c` 改1，否則 class `0x13` 或
race `+0x1f∈{4,5}` 改19；`0x4e555` 對應 29×20 table 的目的地 terrain
entry 必須等於20。indexed renderer 與 Ebiten UI 尚未接。

選單游標導航在 `0x1864D` 一帶,用 PC 方向鍵掃描碼:

| 掃描碼 | 鍵 | 行為 |
|---|---|---|
| `0x48` | ↑ | 上移選項 |
| `0x50` | ↓ | `[ebx+0xc]==0` 時 `[0x3C57]=3` |
| `0x4B` | ← | `[ebx+4]==0` 時 `[0x3C57]=1` |
| `0x4D` | → | `[ebx+8]==0` 時 `[0x3C57]=2` |

- **`[0x3C57]`** = 目前選到的選項編號(全域)。↑→0、(其餘方向)→1/2/3。
- `[ebx]` / `[ebx+4]` / `[ebx+8]` / `[ebx+0xC]` = 上/左/右/下各方向是否有可移動到的相鄰選項
  (選單為 2D 排列,該方向無項目(旗標≠0)就不動)。
- 「方向鍵移到神祕商店」(攻略 #16,`0x2DBF3` 等多處)= 商店畫面的方向鍵游標邊界判定。

### 確認 / 取消鍵(已反組譯 `0x18610`)

選單主迴圈 `call 0x18698` 取鍵後:

| 鍵 | 掃描碼 | 行為 |
|---|---|---|
| **Enter** | `0x1C` | 確認 / 選定(函式回傳 1) |
| **Space** | `0x39` | 同 Enter,確認 / 選定 |
| **ESC** | `0x01` | 取消 / 返回(`0x18698` 回傳 1 → 外層回傳 −1 = 取消) |
| ↑ ↓ ← → | `0x48 0x50 0x4B 0x4D` | 移動游標 `[0x3C57]`(受各方向啟用旗標 gate) |

→ **「ESC 取消、Enter / Space 確認」的離開語意在原版即成立**(ESC 比較在 EXE 出現 72 處 = 各層皆可退)。

## 相關全域 / 結構

- `[0x3C57]` 選單游標選項;`[0x3C43]`/`[0x3C47]`/`[0x3C4B]`/`[0x3C4F]` AI 最佳落點/目標(見 11-…)。
- 單位陣列 `[0x3A45]`,每單位 0x50,數量 `[0x3BEB]`;已驗證 action/inactive predicates 在 record `+5`，其餘欄位保持 raw。

## 重製對應(SDL2 / Ebiten)

| 原版 | 現代 |
|---|---|
| 舊 AA 行動狀態(00/01/80) | 已撤回；目前只保存 raw `+5` bit0/bit7 predicates，end-turn 組合條件仍待 evidence |
| flood-fill 移動範圍 | BFS/Dijkstra(MV + 地形成本)→ 高亮可走格 |
| `[0x3C57]` 選單游標 + 方向鍵 | 選單元件,↑↓←→ 導航、Enter 確認、ESC 取消(見離開鐵則) |
| Get_EasyMagic | 依 native magic raw／command inventory 動態產生法術選單（`+0x22..+0x24` 不是 bitfield） |
| 選目標範圍判定 | 共用攻擊/法術 range + LOS 計算 |

## 待辦(後輪)
- 反組譯選單繪製與選項表,確認確切選項與排列(2D 位置)。
- ✅ 按鍵綁定(Enter/Space 確認、ESC 取消、方向鍵)— 已反組譯。
- [~] `Get_EasyMagic`(0x18ED0) 的 UI caller 已定位；magic raw=`+0x1a..+0x1d`、`+0x22..+0x24` 僅保留 raw transient/modifier bytes，完整 bit 展開與高階欄位語意仍待重審。
- 各選項的 enable gate 條件(已移動 / MP / 道具 / 攻擊範圍內有無目標)的完整反組譯。
- command bits → 可選 command／spell ID 的完整展開（法術編號見 `02`/`03`）。
