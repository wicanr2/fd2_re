# 42 — RE 已記錄 vs remake 已實作:落差稽核

> **文件狀態（2026-08-12）**：本檔是機制落差的歷史專題快照，表內 `✅／🟡／❌`
> 不再代表目前總進度。最新分層狀態統一看
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)，精確系統契約看
> [`56`](56-fd2-remake-sdd.md)，玩家可見差距看 [`57`](57-ui-evidence-matrix.md)，
> 有效下一步只看 [`91`](91-worklist.md) 檔首。保留本表是為了追溯早期 code／RE
> 差距，不得因本表某列仍寫缺少而直接重新反組譯。
>
> 目的:逐一核對「RE knowledge-base 已記錄的機制」與「remake 程式碼實際做了什麼」,列出落差與優先度。
> 方法:每項機制先讀對應 doc,再 grep/讀 `remake/internal/battle`、`remake/cmd/fd2` 的實作,以 code 為準,不憑印象判定。
> 2026-07-25 重新校正本表：撤回已被後續 code 推翻的「零命中／完全沒有」斷言。序章主角隊進場(staging)由另一 agent 處理,本篇不重複列。
> 狀態符號:✅已實作(含公式/資料對齊) 🟡部分(做了一半或簡化) ❌缺(RE 有記錄,remake 未做)。

## 2026-07-27 進度停滯審計

近期 commit 主要增加 E0 raw adapters 與斷言撤回，沒有等比例增加可操作的
campaign/UI 路徑。根因不是反組譯工具不足，而是 E0→runtime→E2 的串接缺口：
`main.go` 仍是 monolithic scene/input/draw owner；UI contract 缺 deterministic
input/state trace 與 screenshot gate；30 章 postbattle graph 尚未逐章驗收；worklist
勾選數也容易把函式級成果誤讀成玩法完成。

本表後續採用「垂直閉環」判定：raw slice 若沒有 caller/data contract、runtime
consumer、regression，且 UI 項目沒有 E2 artifact，只能列為 🟡，不得提升為完成。
下一個優先工作不是再開新的孤立 offset，而是把 title→dialog→battle→postbattle
hub→preparation/town 做成可重播 input trace；item effect、AI runtime 等新 RE
只有在能直接供應該垂直鏈時才解除 fail-closed。

> **2026-07-26 native-command correction**：本表中 legacy `magic.go`／`CastArea` 的舊逐招
> 勾選，不能再當作原版 command runtime 的完成宣告。權威逐 ID dataflow 與 strict engine 邊界是
> SDD `56 §UI-03`，UI 證據則是 `57 §UI-03`；未有 E0 target、transaction、renderer 證據的 ID
> 必須維持 fail-closed。下列舊表列已改為只描述 normalized approximation，不再宣稱與原版同義。

## 總表

| 機制 | RE doc 出處 | remake 狀態 | 證據 | 優先度 |
|---|---|---|---|---|
| 物理攻擊:基礎傷害 AP−DP | doc02 §4.1 | ✅ | `combat.go:12` `dmg := a.EffectiveAP() - d.EffectiveDP()` | — |
| 物理攻擊:**地形 AP/DP% 修正** | native `0x1acf3`、`0x51a12/0x51a2a`、doc11 | ✅原版逐格資料 | `map.json` 已有 raw `tiles`+`native_terrain_control`; `battle.Load` 只在完整驗證後導出逐格 FDSHAP control byte+1，`TerrainAPDPPct` 直接採 static table：0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5)。舊／不完整 map 才保留 Cost fallback，相容性而非原版語意。 | 低 |
| 物理攻擊:**暴擊(DP÷2)** | doc02 §4.1「暴擊時 DP=守方DP/2」 | ✅ | `combat.go AttackWithRNG`:`CritPct>0 && rng.Intn(100)<a.CritPct` 觸發後 `dp/=2`,順序照 notes.md(先減半再套地形%);`CritPct` 來源 `resist_crit.json`(EXE 0x5219B,已與 doc02 §7.2 逐職業交叉驗證吻合) | — |
| 物理攻擊:**命中率 (HIT−EV)%** | doc02 §4.1 | 🟡已補(HIT/EV 為近似值) | `model.go EffectiveHIT/EffectiveEV` + `combat.go rollsHitPct`;**HIT/EV 兩個基礎值本身是固定近似值(export_units.py DEFAULT_HIT=90/DEFAULT_EV=5)**,因為 doc03 明確記載這是「衍生值(由上面計算,直接改無效)」而非「敵/友單位 10B」表的原始欄位,且 remake 尚無裝備系統可提供真正來源(item.json 的 hit/ev 是掛在武器/防具上)——**doc42 原敘述「只是匯出腳本未取用」不完全準確,實際是來源表本身缺這兩欄**,見 export_units.py 檔頭更正說明 | 中(HIT/EV 真值待裝備系統) |
| 物理攻擊:**傷害隨機化(0.9×max~max−1)** | doc02 §4.1 | ✅ | `combat.go AttackWithRNG` 呼叫 `magic.go randomizeAmount`(與法術共用同一公式) | — |
| native IDs24/28/29/31 derived strike | SDD56 UI-03 | 🟡 strict state-only | `ExecuteNativeCommand24`／`ExecuteNativeCommandDerivedStrike` 已依 `0x276EC` 的 verified multiplier 寫 final HP delta；two-stage UI、multi-hit/SFX 未接。legacy `CastArea` 不是證據 | 高 |
| normalized spell attack/heal/hit | doc02；legacy `magic.go` | 🟡 approximation | `CastArea` 有可玩結算，但 native command ID、target geometry、effect family 和 renderer 沒有逐項閉合；不得以其數字證明原版法術完成 | 高 |
| native IDs17–19 modifier | SDD56 UI-03 | 🟡 raw recompute adapter | 已驗 `+0x22..+0x24` raw writer／duration；`ApplyNativeRuntimeEquipmentRecalc` 已保存 `0x1B750` 的 binary64 1.15、x87朝零、HIT/EV+15與裝備累加。尚未把 phase expiry、command transaction及presentation接成正式玩法；不能把 legacy Buff 視為同一機制 | 高 |
| native IDs20–22、25–27 clear/application | SDD56 UI-03 | 🟡 strict state-only | 已有 raw clear/application executors；status name、native UI、完整 tick/expiry 對照未閉合 | 高 |
| native ID23 relocation | SDD56 UI-03 | ⚠️ | first-target、mode6 destination legality、MP/座標 transaction已接；`0x22253` 27-present indexed presentation仍缺 | 中 |
| native IDs32–35 compound | SDD56 UI-03 | ❌ | static helper order 已知，但 MP transaction、rollback、UI/SFX 未閉合；禁止以 legacy combo 實作宣稱完成 | 高 |
| 經驗值公式(攻擊/恢復/各系術) | doc02 §4.5 | 🟡 normalized approximation | legacy `growth.go`／`CastArea` 的獎勵不證明 native command 逐 ID 的 EXP route；IDs22/32–35 等仍缺原版 transaction/effect evidence | 高 |
| 升級(每 100 經驗一級、成長亂數) | doc02 §2/§4.6/§7.2 | ✅已補(worklist 第 9 輪) | `growth.go GainExp`/`applyLevelUpGrowth`,門檻 100(doc03 0x43),可連續跨級;`growthTable` 為 doc02 §7.2 顯示值與 EXE 升級成長表(`docs/data/exe_tables/growth.json` 0x55EA1)交叉比對後的精確版(63 列全比對成功,見該檔案頭註解),非估計值。`Unit` 新增 `Exp`/`ExpPerLevel`/`DX` 欄;`ExpPerLevel`(攻擊經驗公式的「守方每級經驗」)來源 EXE 敵/友單位表,由 `export_units.py` 新增 `ex` 欄接上,34 份本機 `map*_units.json` 資產已重新匯出;查無成長資料的單位(如無名雜兵)等級仍照門檻演進但不套用屬性成長,誠實標記非靜默丟棄。**升級是否立即回滿新增 HP**doc 未明講,採較合理的 RPG 慣例並於 `growth.go` 註解誠實標記為假設 | — |
| 敵方 AI：物理落點、尋路與目標評分 | doc11（2026-07-29 recheck；2026-08-09 mode2／candidate bridge／item-source 勘誤） | 🟡 | `0x145CD→0x4E040→0x146D1→0x14B16` 已閉合可達落點；`0x14237` 已閉合逐落點／目標評分；`0x1B83D→0x1B722→0x4E56C` actor 物品列來源亦已由 IDA 與 `ResolveNativeAIPhysicalItemSource` 保存。`BuildNativeAIPhysicalAttackCandidates` 將 row-major 落點、caller-provided `0x14818` geometry、raw `+5/+6` target filter 與 detached record snapshots 串成 E0 候選資料鏈，但 score resolver、`0x1DEBE/+8`、完整 mode/selector/turn 語意與 production runtime 接線仍未閉合；不可把物品列來源誤稱為 `0x4E516` command record。`0x14B78→0x4E1A6→0x13488` 已閉合方向碼、地形成本、路徑與實際落點排序；FDFIELD `b17/b18/b19` → runtime `+0x34/+0x35/+0x36` 的來源、33 圖分布及保留高四位的範圍 writer 已資料化。mode 2 現確認為 `0x14EF0` 失敗後 `0x14237→0x13C06→0x13FD4`，不走最近座標或 `0x13E9C`；mode 11 的兩個 signed score gates 與唯一已知 runtime writer 亦已閉合。`0x13FD4` 是 raw `+0x25/+0x26` gate 的 `floor(maxHP/5)` 回復，已同步正式休息路徑。一般 mode 0 才會先走 `0x14121` blocked-cell 搜索，再以 `0x13E9C` Manhattan 最近 raw opposite group 備援；舊 `0x15192` 說法撤回。現行 `aiTargets` 與 `aiApproachPath` 仍是重製近似，不等於上述原版鏈 | 高 |
| 敵方 AI:**施法決策**(法師/僧侶主動用攻擊術/補血術) | direct disasm 證實兩條不同 producer：`0x1567E` 枚舉 inventory slot→item row command，候選交 `0x15880`；`0x1598A` 枚舉 unit command mask，候選交 `0x15B77`。兩條 producer 均已有具型別實作；前者 command>0x0F 才做 `command-0x10`，兩套 ranking 不可合併 | 🟡 | `BuildNativeAIPhaseDiagnosticPlan` 已依原序計算兩個分數；`ExecuteNativePhaseUnitScans` 另保存逐筆重判、90／30 筆回呼順序與 pending 提前退出。但 `0x13A9F` 及各 handler 效果仍未接入，同一原版執行期狀態的動態 trace、MP／交易與呈現仍缺，因此未接 `NextAIPlan` | 高；補動態原版 trace、phase／執行交易與 runtime Cast |
| 敵方 AI:**使用道具** | `0x1567E→0x15880` 已證實會依 raw inventory slot 與 item row command 做預選；這證明 AI 有物品指令候選，不等於每種道具效果都已命名或可執行 | 🟡（僅靜態預選） | `ScoreNativeAI1567E` 已閉合數值 producer；`aiActUnit` 尚未接物品交易、消耗、效果與呈現，故正式執行仍維持失敗即關閉 | 高；補執行消費端、物品交易及同狀態原版驗證 |
| 移動地形成本(森林/沼澤耗 MV) | doc02 §3.1 | ✅ | `move.go MoveCost` 讀 `map.json` 的 `cost` 陣列(worklist 第8輪「地形屬性接線」) | — |
| **地形攻防加成** | native `0x1acf3`、`0x51a12/0x51a2a` | ✅原版逐格資料 | 同上「物理攻擊:地形 AP/DP% 修正」條：完整 raw map 直接用 FDSHAP control byte+1；0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5)。這取代 doc02 對一般地面 DP-5 的舊摘要；Cost 僅是 legacy fallback。 | 低 |
| 裝備欄 / 裝備加成 AP/DP/HIT/EV | doc02 §5、doc32 §1/§5 | 🟡 | `Unit` 有 `Inventory`/`Equipped`/`InventorySlots`；`campaign.EquipItem`、`RecomputeEquipment` 與 shop equip UI 已接線。HIT/EV 真值與戰鬥全鏈仍需對照原版欄位，不能再說「無裝備欄」 | 高；補 native stat source 與 battle integration |
| 道具使用效果(藥水回血、卷軸) | doc02 §5.13 | ❌ | battle action 的 item command 仍是未實作入口；但 inventory/reward/shop/equip 並非全缺 | 高 |
| 裝備自帶法術(不耗MP、無經驗) | doc02 §4.6、doc32 | 🟡 | 裝備資料與裝備重算已存在，裝備法術在 `Cast`/action menu 尚未接成獨立不耗 MP 路徑 | 中 |
| 轉職系統(Lv20+教會、轉職道具→最高職業) | doc02 §7.1、doc32 §4「[阻] 轉職系統」；official IDA `0x31385/0x31793/0x311DC/0x19953` | 🟡 | `church` UI、`ClassChangeCandidates`、單一 target resolution（special>optional>default）、Yes/No confirmation、`ApplyClassChange` 與 growth table 已存在；exact indexed renderer、fee／數值實機差分仍待 | 中 |
| legacy 中毒/麻痺/封咒與 Buff 的回合處理 | doc02 §6.4 | 🟡 normalized approximation | `model.go TickStatus`,`main.go:1962` 每回合結尾呼叫；但 native `0x1A866` 實際對 raw `+0x22..+0x27` 逐 camp、逐 byte 遞減，只有歸零才重算，尚不能宣稱 `TickStatus` 的 named status／每回合語意等同原版 | 高（native transient/UI/expiry recompute） |
| legacy 中毒每回合 −10% HP | doc02 §6.4 | 🟡 normalized approximation | `TickStatus` 使用 `dmg := u.MaxHP/10`；現已知 native command transient `+0x25..+0x27` 不能直接命名為 legacy Poison/Paralyze/Seal，故此不是原版 native status closure | 高 |
| legacy Buff(魔刃/魔鎧/風行)到期清除 | doc02 §6.4 | 🟡 normalized approximation | `TickStatus` 的共享 `BuffTurns` 歸零清空；native IDs17..19 則分別寫 `+0x22/+0x23/+0x24`、2..5 camp phases、並依 `0x1B750` 重算衍生值，shared timer 不等同原版 | 高 |
| 對話嘴型動畫(m0閉/m3開,doc14 0x16d00) | doc14、doc40 | 🟡 | `main.go`/`internal/dato.MouthState` 已對齊每 2 frame、開嘴 1 tick、`rand()%30+2` cadence，並可載入四幀 DATO；但 `0x168b6` dialogue-frame/grid、完整 resource binding、speaker layout 與 runtime dialogue parity 尚未閉合，不能列為完整 ✅ | 中 |
| 法術施放演出(命中/傷害畫面) | doc35、doc37 | 🟡 approximation | remake 攻擊型法術(`sp.Target==0`)重用 `newAtkAnim`，治療型只有文字；這是目前 runtime 缺口，**不是原版「無獨立法術特效」的結論**。原版僅證實 `0x28784` 不以 spell-id 選另一段 FIGANI；`0x2a6bd` command presentation dispatcher 與命中／效果層仍待閉合。 | 中(補 native presentation/effect path) |
| 全螢幕攻擊 FIGANI 幀節拍 | doc06、doc35；IDA `0x2b9a1` | 🟡 E1 呈現橋 | `remake/assets/figani/delays.json` 保存固定版本原始 descriptor `+6`；`internal/figani.DisplayScheduler` 與 `cmd/fd2` 逐幀回歸已接入 PNG 攻擊幀。缺配對即失敗即關閉；命中幀、HP／傷害時機、音效、台座與 E2 畫面仍未閉合，不能列為完整戰鬥演出。 | 中（補 indexed／一般玩家畫面） |
| 商店(一般商品) | `0x2e341→0x2f0b0/0x2f642/0x2f883/0x2f8ea`、doc56 UI-09 | 🟡 | purchase、sell、standalone equip與transfer四條production縱切已閉合。ch02 variant1/3/5主選單、weapon purchase-list四個selection、purchase Yes/No、gold0不足金與gold1000裝備收件者selection0/cycle1已有全幀DOSBox E2；該recipient E2仍使用screenshot-only bootstrap。正常campaign已另修JOIN→LOADCH首次typed roster seeding，ch00 scenario/order可進ch02候選`[0,9,4]`，direct replay不造persistent state；這是runtime regression，不是native FD2.SAV或完整playthrough E2。transfer保存512/511/510/506訊息、source/item/destination/full loop、raw remove→append→source recalc與self-transfer。sell仍只宣稱gameplay parity，不宣稱save byte parity；custom variant0保留generic fallback | 高；補recipient input/scroll、no-recipient/full/success、sell/equip/transfer與其他章節同狀態E2 |
| 城鎮hub畫面／selector | `0x2cd16/0x2cf71/0x11eb0`、FDOTHER#10/#11/#61/#62、FDTXT `0x1ef+selection`、FDICON0..2 | 🟡 production／ch02 variant0 E2；variant1/2 selection0–4 修改LOAD E2 | 23個town保存raw variant0/1/2；production已接原版背景、label、六組座標、`0,1,2,1` pulse與312×192→VGA `(4,4)`。ch02 variant0 selection0–5、Left/Right wrap與hidden reveal均有全幀DOSBox E2；另以固定雜湊原版修改LOAD副本取得variant1與variant2，兩者selection0–4都與指定pulse整幀AE=0，詳見[`native_town_variant1_e2.json`](../data/native_town_variant1_e2.json)與[`native_town_variant2_e2.json`](../data/native_town_variant2_e2.json)。這不是未修改一般玩家路徑，也不涵蓋selection5 | 高；補variant2 selection5 BIOS掃描碼／Enter、未修改玩家路徑與其他章節capture |
| 祕密商店進入gate | `0x2cde0..0x2cef7`、town table `0x6238d`、`0x2d28c` selection5→`0x2e341` | 🟡 production／ch02 E2 | 撤回舊✅、persistent-flag等價及「chord立即進店」說：原版以每章0x1f-byte record的`+1`目前選項、`+2` BIOS Shift/Ctrl/Alt-F1..F10 scan同時命中，當次只把selection改為5並重畫；後續Enter才dispatch。23筆gate已進editable `native_secret_gate`；ch02 chord→confirm→variant5與Escape return已有DOSBox E2；legacy `SecretIf`／`found_secret_ch*`只保留擴充相容 | 高；其餘22個town逐章chord/route E2 |
| 商店賣出(原價 75 折) | doc02 §4.6、native shop callee audit | 🟡 | `campInput` shop sell mode、`campaign.SellSlot`、price×3/4 與 inventory/equipment recompute 已實作；這只證明 normalized transaction，原版 service callee、indexed menu/cancel semantics尚未閉合 | 中 |
| 存檔/讀檔 | doc19、doc56 UI-12 | 🟡（自有格式；原生匯入部分完成） | `save.go`保存campaign節點／旗標／金幣／道具／typed persistent party與deploy/order；四槽 bounded selector 與原版 indexed compositor 已接，空槽達 E2，修改存檔有效槽只作排版 oracle。`FD2_NATIVE_SAVE` 可驗 checksum、顯示四槽 metadata，並對空槽／未支援還原的有效槽保持 selector；仍缺 native roster→normalized party、一般玩家成功 restore 與完整 playthrough，不能標成原版機制✅ | 高 |
| BGM 播放 | doc12 | ✅ | `audio.go playBGM`,同曲不重播/換曲釋放語意對齊 `0x26777` | — |
| SFX(命中/陣亡/選單音) | doc36 | ✅(池對照為近似值,doc36 已註記真實 attack_id→sfx 池對照未 RE 完成) | `audio.go loadSFX/playSFX`,`main.go` 多處呼叫 | — |
| 出場人數上限(前27章16人/末3章20人) | doc02 §4.6、native `0x2d093→0x318ad` | 🟡 | remake 已把可選人數門檻資料化為一般 `party_limit=15`、末路線 `party_limit=19`；外層已證實小名冊完全略過選人，超過門檻才進全零勾選表。這是 native 0-based cap 的目前證據，不把持久記錄總數16/20與可選上限15/19混為同一欄。仍待完整 native deployment cursor／overflow 行為與實機 UI 對照 | 中 |

## RE 側也需要補的缺口(非 remake 落差,附帶記錄)

- **施法入口已找到，但 producer 必須分開**：`0x154D1` 仍不是入口；
  `0x1567E→0x15880` 是 inventory item-command 預選，
  `0x1598A→0x15B77` 是 unit command-mask 預選；`0x15055/0x150F1`
  是執行消費端。兩者不能共用未證實的 ranking 或槽／command 身分。
- **doc32 §4 三個 `[阻]` 項目(裝備加成精確公式、物品使用效果碼、轉職系統機制)本身就還沒反組譯完**,remake 的裝備/道具/轉職缺口有一部分要等 RE 補完才能對照實作,不能單純算「remake 沒做」。

## 已移除的歷史統計與排序

早期的 33 項計數、worklist 第 8 輪增減帳與「前 5 項」排序已被後續程式與 RE 證據取代，
會與上方逐列現況互相矛盾，故不保留在本文件。需要歷史 provenance 請查 Git；當前優先序
由 `56` SDD 的 evidence gate、`57` UI matrix 與 `91` worklist 決定。
