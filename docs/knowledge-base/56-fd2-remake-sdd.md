# 56 — FD2 remake 系統設計規格（SDD，2026-08-09）

> 本文件是重新開始 remake 前的設計闸門。目標不是把目前能啟動的 Ebiten demo 擴張成更多 placeholder，而是以可追溯的反組譯證據，重建原版的操作介面、戰間流程、資料與腳本引擎。未滿足證據與驗收條件的語意保持 fail-closed。
>
> **文件責任（2026-08-12）**：本檔保存系統契約、ABI、證據 gate 與精確設計，
> 不再從四千行歷史追加段落推算整體進度。最新「原版證據→可編輯資料→正式
> 執行期→玩家 E2」狀態與 canonical evidence，統一看
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)。本檔後段的日期條目是
> 設計沿革；與 `58` 衝突時先核對主證據，再更新兩者，不能重開整個子系統。

## 1. 目標與現況判定

### 1.1 目標

- 原版 30 關的 campaign、戰鬥、戰後城鎮／商店／教會／整備、存檔與結局可循環遊玩。
- 對話、事件、商店、部署、過場和 UI layout 都由外部資料／腳本驅動；新增戰役不需修改 Go runtime。
- UI 操作語意以原版為目標：游標、action overlay／command grid、射程／目標、對話框、狀態欄、商店、教會和戰後節點均須有可見且可測的操作入口；未取得 E0/E1/E2 證據的現有 UI 只算 approximation。
- native indexed renderer 與現代 RGBA/Ebiten 顯示層分離；未完成 native ABI 時不得用泛用淡出、PNG 或空白畫面冒充完成。

### 1.2 現況（2026-08-22 摘要；詳細狀態以 `58` 為準）

目前不是「沒有程式」，而是「有一個可跑的垂直切片，尚未達 remake」：`remake/cmd/fd2/main.go` 仍承擔 scene state、輸入 dispatch、戰鬥 UI、對話、town、shop、church、preparation 與 Draw；`internal/battle`、`internal/campaign`、`internal/ending`、`internal/figani` 已有可測的部分 primitive。這些 primitive 不等於原版 UI 或完整 campaign。

已存在但尚未全程驗收：story/cutscene BeatRunner、dialog 分頁／捲動、campaign
node、persistent roster、shop buy/sell/equip、church revive/class-change、
preparation quota、敵方 AI 窄 consumer 與 indexed ending prefix。2026-08-21
實跑戰後 audit 為24節點中24 active／0 blocked；玩家第23戰已以16＋70槽位
拓撲、raw ch22 indexed／resource adapter 與 `preparation_ch24` 存讀檔提升為 E1；玩家第24戰已以70＋16槽位
拓撲、raw ch23 兩段 indexed adapter 與 `preparation_ch25` 存讀檔提升為 E1；玩家第25戰已以62→70→71槽位
拓撲接通 raw ch24 post、`town_ch26` 與存讀檔 E1；玩家第29戰 raw ch28 post、
group9、持續隊伍與 `preparation_ch30` 存讀檔也已達 E1。故本檔較早的
「20 active／4 blocked」、「玩家第25戰仍阻擋」及「玩家第29戰仍失敗即關閉」
均已失效。
其餘主要缺口是完整玩家指令／法術／物品交易、相同 raw 狀態敵方回合、戰場與
戰間 UI、精確終局 renderer／輸入，以及一般玩家 E2；詳見 `58`。

### 1.3 進度停滯審計（2026-07-27）

最近一批 commit 的共同特徵是 AI、town/preparation、item 等單一 native offset
的 E0 raw slice 與文件勘誤；它們多數沒有接到 `main.go` 的 scene FSM、可見 UI、
campaign JSON transition 或玩家可操作測試。因此 worklist 的 `[x]` 數量會增加，
但玩家可走的原版等價流程沒有同步增加。這不是 Capstone、IDA 或 Docker blocker，
而是「證據切片完成」被誤當成「機制垂直切片完成」的流程問題。

根因有四個：`remake/cmd/fd2/main.go` 仍同時承擔 scene state、輸入、規則、繪圖
與 town/shop/church/preparation；UI-01…UI-12 多數只有 unit test 或 normalized
approximation，缺少同一 input trace 的 state trace、畫面 artifact 與原版 E2 對照；
30 章 postbattle→town/shop/church/preparation/save graph 尚未逐章驗收；歷史文件
曾把「格式／函式已解」寫成「系統已完成」。

從本節起，新的 RE 條目只有在同一輪明確指定 caller、資料契約、runtime consumer、
deterministic regression 與（若屬 UI）截圖／E2 trace，才可解除 implementation
gate；只有 E0 raw slice 的項目保持 `[~]`，不得再用新增同類 adapter 充當進度。
下一個主里程碑改為 UI-01→UI-08 的垂直操作鏈（title→dialog→battle→postbattle
hub→preparation/town），完成前暫停無 caller/consumer 的孤立 RE 擴張。

AI spell scoring raw slice：Docker Capstone/Hex-Rays 已閉合 `0x15b77` attack IDs0..12 的 HP threshold/`+0x08` branch、recovery IDs13..16 的 max/3→8、max/2→3 tiers/`+0x34 bit0` branch、ID17/18/19 的 `+0x22/+0x23/+0x24` zero flag score 3、ID20/21 的 nonzero `+0x25/+0x26` score 6、ID22 的 `+0x27` gate→`0x1c269` bit scan→6，以及 ID26/27 的 zero `+0x25/+0x26` score 4；`ScoreNativeAISpellAttack`／`ScoreNativeAISpellRecovery`／`ScoreNativeAISpellFlag`／`ScoreNativeAISpellZeroFlag`／`ScoreNativeAISpell22` 只接受 raw records，ID10..12 另要求 caller-supplied `0x1f183` gate。這些函式不命名欄位、不接 AI runtime 或 target UI。

`0x1598a` dispatcher 的 raw selection boundary 也已閉合：`unit+0x27==0` 後，`0x1c269` 產生 command bytes；每筆 command 先以 record `+5 <= unit+0x44` 過濾，再由 `0x4e040`/`0x14818` 產生目標候選，呼 `0x15b77(command,candidateCount,candidateBytes)` 評分。最大 score 勝；同分比較 command record `+0`，仍同分則保留先出現者。`battle.SelectNativeAISpellCandidate` 只保存此 score/tie-break 與 raw `(x,y,command)`，不代替 MP、target resolver、`+0x27` gate、UI 或施法執行。
`battle.NativeAvailableAICommandIDs` 另保存 dispatcher 前置 gate：raw `+0x27` 非零時不產生任何 AI command IDs；為避免把第五 command byte 的未知 physical IDs36..39 當可執行命令，仍只回傳已驗證的 0..35 records。

### AI unit/action call-graph boundary（E0 raw；E1 窄 runtime）

Canonical Docker Capstone 目前可重現的上層順序是：`0x1A4EB`／`0x1A58F` 的 phase-specific
setup 後進入 `0x1D80B`／`0x1D8BA` unit scans；每筆 `0x50`-byte record 經 raw `+6`、
`+5`、`+0x26` gates 後進 `0x13A9F`。`0x13A9F` 讀 `record+0x34 & 0x0f`，再依 raw
command nibble 分派 `0x14EF0`、`0x1598A`、`0x15311`、`0x1548E`；`0x14EF0` 內有
依序執行的 `0x14237` 物理、`0x1598A` command-mask 與 `0x1567E`
item-command 三個 producer。其 `0x14ef0..0x15055` raw 尾端再以
`[0x53c4f]`／`[0x53c23]`／`[0x53c33]` 門檻、record `+0x34 & 0x40`、
actor `+0x48`、target `+0x4a` 與必要時 `0x4e516([0x53c2f])` 分派
`0x1548e`／`0x15311`／`0x15055`；完整低階契約見
[`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt)。只有 `0x1598A` 內的 `0x15AD8→0x15B77`
負責該 producer 的 raw score/tie-break；`0x1567E` 改呼 `0x15880`。
這取代舊文件把 `0x15140` 稱作 AI entry 的說法。重製端仍只保存
`SelectNativeAI14EF0Tail` 的無副作用 raw 路由，缺 provenance 即失敗關閉；
2026-08-10 另以一個明確的 E1 窄切片讓 `NextAIPlan` 僅在原始 mode 2、完整
`0x4e555` 移動表、FDFIELD 地形／組成來源、物品幾何與
`0x14237` 評分來源齊全時消費物理候選。這個消費端（consumer）不代表
`0x14EF0` 前置選擇、其他 mode、命令／法術／物品交易或 `0x1548E` 演出已閉合；
其餘情況維持既有重製端相容路徑或在 raw provenance 缺失時失敗即關閉。
SDD 只授權保存上述原始呼叫拓撲（raw call topology）與這個標明範圍的執行期窄切片（runtime slice）；
截至 2026-08-10，`NextAIPlan` 另已接上 raw mode 0／1／3／4／5／7／9／10 的
窄 fallback、`0x14EF0` 的 command／item route，以及 mode 5 的 mutable event
state tail 與 raw sample audio 窄切片；未知 command、mode 11 雙動作、`0x13FD4` presentation、
`0x1548E` indexed 演出與一般玩家 E2 仍失敗即關閉。`+6` 的 raw camp code 已由 constructor 與 `0x14818` consumer 固定為
敵0／友1／己2；但完整 target transaction、movement/effect/UI 與 runtime
AI execution 仍是 fail-closed，不得由 normalized `aiActUnit` 反推 native parity。

物理候選的資料邊界另由 `battle.BuildNativeAIPhysicalAttackCandidates` 保存；
`0x14237` actor 端的 `0x1B83D→0x1B722→0x4E56C` 物品列來源則由
[`fd2_ai_physical_item_source_ida.txt`](../data/ida/fd2_ai_physical_item_source_ida.txt)
與 `battle.ResolveNativeAIPhysicalItemSource` 保存。這條來源已閉合，但不等同
command-mask 的 `0x4E516` table。

物理候選的資料邊界：
它只連接已證實的 row-major movement destinations、caller-provided
`0x14818` geometry、raw `+5/+6` target filter 與 detached record snapshots。
地形百分比修正、`0x1DEBE`、target `+8` 等輸入必須由明示 resolver 提供；
缺失時失敗即關閉。這是 E0/E1 窄資料橋，不是完整 command record loader、
正式 native AI planner、原生戰鬥演出或 UI consumer；`NextAIPlan` 只有上述
mode 2 來源完整時採用它，其餘模式仍不可宣稱原版一致。

上層掃描現已有可重現的完整指令產物
[`phase setup`](../data/fd2_ai_phase_setup_disasm.txt)、
[`unit scans`](../data/fd2_ai_unit_scan_disasm.txt) 與
[`mode dispatch`](../data/fd2_ai_mode_dispatch_disasm.txt)。`0x1D80B`
只處理 raw `+6==1` 並傳第二參數1。`0x1D8BA` 對 raw `+6==0` 先做
預選掃描：`0x1598A(unit,0)→0x1567E(unit,0)` 後，只有 signed
`[0x53C23]>=6` 或 `[0x53C33]>=6` 才呼叫 `0x13A9F(unit,0)`；隨後
`0x1D988` 再對同一 raw gate 做無此前置分數門檻的第二遍
`0x13A9F(unit,0)`。每筆 mode 收尾固定為
`selector1→0x13512→0x134E4→redraw`，然後才依序執行可選的 90 筆
全域事件表、無條件的 30 筆章節戰場事件處理器表與 `[0x53ECC]`
pending 碼檢查。合法 IDA Pro 9.4 已以相鄰邊界及原始指標確認兩張表的
筆數；在 IDA 直接資料交叉參照中，`0x13A44` 是 `[0x51A8F]`
唯一非重設寫入端。
第二掃描返回後
`0x1A5B9` 才增加 `[0x53BEF]`。這些證據禁止把 `0x1D8BA` 簡化為單遍
敵方行動迴圈。兩張表的結構、索引與呼叫順序已閉合，不代表 90／30 個
handler 的玩家可見效果全部閉合；pending 碼在本層也只保留 raw 數值。
結合既有 raw camp writer／consumer，`0x1D80B` 是友軍 camp1 單遍，
`0x1D8BA/0x1D988` 是敵軍 camp0 的預選＋第二遍；真正未知的是敵軍為何
需要兩遍，以及兩個分數門檻的完整玩法語意。缺 raw provenance 時仍不可
用 normalized Go `Camp` 猜補。

`fdother.PlanNativePhaseUnitScans` 把這三個 loop 保存成分離且有序的
具型別診斷計畫：selector1 單遍、selector0 預選、selector0 第二遍。預選要求
呼叫端明示每個 native unit 的 signed `[0x53C23]/[0x53C33]` 結果，任一
至少6才標記該遍會呼叫 `0x13A9F`；第二遍仍包含所有通過
`+6/+5/+0x26` gates 的 selector0 初始記錄。它只供無副作用診斷，不能
拿來直接執行，因為第一遍成功動作會改寫 bit7，第二遍必須重新讀 record。

`fdother.ExecuteNativePhaseUnitScans` 另保存逐筆 E0 執行契約：三遍均重新
判斷 raw gate；不合格記錄仍執行章節表與 pending 檢查；全域表後會重新
取得章節索引，且即使全域 handler 已設 pending，也要先完成章節 handler
才檢查退出。全域／章節索引分別限制為 0..89／0..29，缺回呼或越界即
失敗關閉。它仍只接受呼叫端提供的 action／handler 效果，不代表正式
production AI 已切換至原版執行器。直接證據見
[`fd2_ai_phase_callback_tables_ida.txt`](../data/fd2_ai_phase_callback_tables_ida.txt)。

敵軍兩遍的直接目的亦已由 producer／consumer 串起：`0x1598A` 的法術
候選最佳分數寫 `[0x53C23]`，`0x1567E` 的 item-command 候選最佳分數寫
`[0x53C33]`；優先遍任一 signed score `>=6` 才進 action dispatcher。
成功收尾 `0x13512` 設 raw bit7，第二遍的 `+5 & 0x81` admission 隨即
排除該記錄。因此這是原始分數門檻通過的 command 先行，不是雙動；不替分數
賦予高階玩法名稱。typed input 欄位
保留地址後綴，避免把 producer 類別擴張成某個特定 spell/item 效果。

`0x14237..0x145CC` 現已閉合為物理候選評分迴圈，而不是未知的 generic candidate
helper。它以 `0x145CD→0x4E040→0x146D1→0x14B16` 建立 row-major 候選格，
再以 `0x14818` 建立各格目標陣列；actor 與 target
的 raw `word +0x48/+0x4A` 會依各自 `0x12E38` 地形 control byte及
`0x51A12/0x51A2A` 百分比表修正。單一候選先算
`actor word48-target word4A`，`<=2` 拒絕；基本 priority=8，分數嚴格
`> target word40` 時分數×2且 priority=`0x12`。`0x1DEBE` 回傳1時再加
`actor word4A-target word48`；target raw `+8==0` 時以 signed toward-zero 規則×1.5。
選擇先比 priority，再比 score，完全同分保留先枚舉者，寫
`0x53C43/47/4B/4F`。`battle.ScoreNativePhysicalAttackCandidate` 保存 raw
單候選規則，`SelectNativePhysicalAttackCandidate` 保存 priority→score→穩定同分
選擇；`0x1DEBE`、raw `+8`、候選產生順序與完整執行仍未命名或接入 runtime。
合法 IDA 9.4 的 address-only report 交叉確認 `0x14237..0x145CC` 邊界、
三個 direct callers，以及 `0x1DEBE` 僅由該函式呼叫；新增的 IDA／Capstone
逐指令證據見 [`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt)。

`0x1548E..0x1567D` 則已更正為物理選擇結果執行邊界，不是路徑產生器：
它把 `0x53C43/47`、actor 與原始分組選擇值交給 `0x14B78`，以 `0x53C4B`
作 target，依 `0x53AF9` 選地圖呈現或 `0x28A6C(actor,target)` 戰鬥演出，
完成收尾後固定回傳1。其唯一 callers 是 `0x13E39/0x14F9B`，全函式沒有呼叫
舊筆記的 `0x4EE40/0x4F355`。實際尋路已定位為 `0x14B78→0x4E1A6`：
方向碼 `0/1/2/3=下/左/上/右`，使用原版成本列、`0x40/0x80` gate，
再以距離→XY 軸差→逐列先後選實際落點。無 action 時的一般 mode 0 備援是
`0x14121` mode-2 blocked-cell 搜索，失敗後才到 `0x13E9C` Manhattan
最近座標；舊 `0x15192` 假說撤回。上層 mode／selector 的完整遊戲命名仍未閉合。
合法 IDA 9.4 同樣確認 `0x1548E..0x1567D` 與兩個 direct callers。

`0x1567E` 的 command enumerator 也已由 canonical Docker Capstone 重讀：它以 unit index
算 `0x50`-byte record，先呼 `0x1B8A6` 取得 inventory 上界，再以
`record+0x0B+2*slot` 讀 item ID，經 `0x4E56C(item)` 取 row；row `+0x0D==0`
略過，`+0x10` 才是 command。
`command <= 0x0F` 走 `0x14818` geometry/target builder，`command > 0x0F` 轉
`command-0x10` 後走 `0x149F8` candidate builder。通過 candidate 後呼
`0x15880` score helper，最大值才寫 raw globals `0x53C33`（score）、`0x53C37/0x53C3B`
（candidate coordinates）、`0x53C3F`（inventory slot index）。執行端
`0x1507C→0x1B722(unit,slot)` 再解 item，`0x150C2` 才讀 row `+0x10`。這是 command enumeration／
selection boundary，不證明 item row 欄位名稱、法術效果、MP transaction 或完整 AI turn；
`battle.AIPlan.NativeScoredCommands` 仍只保存另一條 `0x1598A` 的 raw
provenance，不可拿來代替本函式的 item-command 列表。

`0x15880` score helper 的 raw ABI 也已閉合：它先以 `0x4E56C` 取得 item row，讀 row
`+0x0D` type 與 `+0x0E` word。type `5`／`0x0D` 逐候選讀 target `+0x40/+0x42`，以
current HP `<=maxHP/3` 產生8、否則 `<=maxHP/2` 產生3、其餘0；target record
`+0x34 bit7` 會將該分數乘3。type `0x14`／`0x15` 將 row word 交 `0x4E516`
取得 command word，type `0x18` 直接使用 row word；target HP `<=threshold` 累加
`0x12`，否則8，其他 type 回傳零。`ScoreNativeAIItemCommandTargets` 已保存這些
原始分支與邊界檢查。這些是 command selection 的數值分支，
不把 type 命名成治療、攻擊或 status，也不把 `+0x34 bit7` 命名成可見效果；item row
producer、target transaction、UI 與 runtime executor 仍由各自 evidence gate 控制。

`0x149F8` 則已確認為另一個 raw candidate scanner：它保存 caller 起點，依兩組座標比較
產生 ±X/±Y cardinal step，最多走 caller-supplied count；每一步先以 `[0x53AB1]/[0x53AB5]`
更新 cursor，檢查 map bounds `[0x53AC1]/[0x53AC5]`，再呼 `0x12C0D` 將格子解析為 unit index。
找到 record 後依 caller selector（raw `+6` 的 polarity gate）決定是否把 index 寫入 supplied
byte buffer；完成後恢復 cursor globals 並回傳 count。這證明它是座標／候選收集器，不是
damage、hit 或 spell-effect scorer；selector polarity、LOS／terrain semantics、buffer
ownership 與 command-30 caller contract 仍由各自 evidence gate 控制。

`0x1567E` 的 caller-specific 候選 ABI 現也已閉合。第一個 `0x14818` 從
actor 座標建立目的地 field：row `+0x10` command 作 mode、
`command>0x0F` 作 inner marker、target code 固定0；`0x14B16` 依
row-major 匯出非 `0xFF` 座標。逐目的地時，低 command 再以 row `+0x12`
作 effect mode，target code 在 selector 非零時保留 row `+0x11`，
selector0 時轉成 `row+0x11==0 ? 1 : 0`。高 command 則呼
`0x149F8(destination, actor, command-0x10, selector=0)`，固定只收 raw
camp0。`ScoreNativeAI1567E` 依 slot→destination→target 的原版順序評分，
strict `score>best` 才保存 `[0x53C33/37/3B/3F]` 對應值；零分不創造勝者。
map0＋item79 交叉 fixture 固定 score8、`(19,15)`、slot0，屬靜態 E0，
不是原版玩家路徑 E2。

## 2. 證據分級與反組譯規則

每個進入 runtime 的常數、座標、幀數、資源索引和 handler 語意都必須附證據：

### 2.0 原版版本基準

本 SDD 的位址只適用於 `FD2.EXE` 大小 `357074` 位元組、MD5
`b97caf2239a27a896069d03549d96e1e`、SHA-256
`222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
`FD2.EXE` 與目前解析器所使用資產的完整大小、MD5、SHA-256 存於
[`fd2-reference-files.json`](../data/fd2-reference-files.json)，可用
`tools/hash_fd2_reference.py <原版 FLAME2 目錄>` 唯讀重算。`FD2.SAV` 與
`FD2.TMP` 會隨遊戲狀態改變，刻意不列入固定基準。任何不同雜湊的發行版都
必須以特徵與呼叫關係重新定位，不得直接沿用本文件位址。

| 等級 | 來源 | 可否解除 implementation gate |
|---|---|---|
| E0 | 原版 EXE/DAT bytes、Docker `fd2-cap-local` Capstone、Ghidra/IDA call graph | 可以，需保留 offset、呼叫者與反組譯片段 |
| E1 | deterministic parser、pixel/byte regression、資產 round-trip | 可以，需能重跑且輸出穩定 |
| E2 | DOSBox/Xvfb 實機操作、逐幀截圖／輸入差分 | 可以，需保存 command、frame、artifact |
| E3 | 攻略、影片、視覺推論或 UX 慣例 | 只能列為假設，不得解除 native/handler gate |

本輪重新核對的已知更正：`0x16559` 是 DATO mouth-frame／glyph blit caller，`0x4ea2a` 才是 native glyph renderer；`0x2c435 push 0x1e`、`0x2c437 call 0x1088d` 會在 loader 內選 FDTXT archive resource #31，不能把 raw selector `0x1e` 或實體欄位直接命名成 ch30；`0x2c548` 有 `i=0→slot1、i=1→slot0` swap；`0x29164` 第一參數是 party unit index，TAI#3 是 7-byte transparent aux，不是可見台座。這些結論不可再由名稱外推 renderer 語意。

2026-07-28 visual audit correction：codec／原資源 fixture 的完成度不得再
寫成整體 UI parity。當時曾依 12 個主要界面的部分樣本提出三組工程估計，
但它們沒有完整的章節、畫面狀態與一般玩家路徑分母，現已撤回，不得再當成
專案完成度。最大落差仍包括preparation、
loadslots、ending、town variant1、variant2 selection5／未修改玩家路徑與商店其餘
child panels；不得因ch02已驗
切片而外推成全章覆蓋。完整分項和證據分級以doc57為準；README 已撤回把 raw `title.png`／
`dialogue.png` 稱為 remake runtime 對照的標示。

2026-08-09 已在隔離環境讀取共用知識庫的
`~/.codex/knowledge-base/fd2/reverse-engineering-gui-restoration.md` 與
`sources/claude/retro/ida-pro-9.4.md`。其中的 composition graph、E0/E1/E2
分層、非破壞性斷言索引、IDA xref 優先與 Docker 工具分工，現在作為本專案的
方法論檢查表；它們不是 FD2 原版證據，地址、資源與行為仍必須回到本儲存庫的
IDA／Capstone／DOSBox 產物。使用者已確認 `/home/anr2/ida_pro/ida94b1/idapro.hexlic`
為其合法持有的授權檔；官方 Docker image 的文字版 `/opt/ida-9.4/idat -h` 已以該檔
唯讀掛載驗證可啟動。不得使用同目錄既存的 `kg_patch` 設定、檔案或 Compose 掛載。

repo 提供不含 license／遊戲資料的 `tools/docker/fd2-ida.Dockerfile` 與
`tools/ida_export_fd2_xrefs.py`，供使用者授權的私有 IDA workspace 匯出 xref 後重跑。2026-07-26 已以使用者
合法的本機 IDA Docker image、臨時 overlay 的容器內 Python 3.12、唯讀遊戲檔與 `/tmp` IDA database 實跑；
`docs/data/ida/fd2_xrefs.json` 已由 IDA 9.4/Hex-Rays 產出。過程修正 IDA 9.4 移除的
`ida_xref.get_xref_type` API；export 現只保存 address/caller/function metadata，絕不提交 binary、database 或 license。
這份 report 可作 call-graph E0 交叉驗證，但不自行證明遊戲語意；語意仍須由指令與資料流佐證。

2026-08-13 的可重生全函式清冊由 `tools/ida_export_fd2_function_inventory.py`
只在授權 IDA Docker 匯出原始函式邊界、IDA 分析名稱、旗標與直接 caller；
`tools/compact_fd2_function_inventory.py` 產生受版控的
[`fd2_function_inventory.json`](../data/ida/fd2_function_inventory.json)。固定雜湊輸入
辨識1305函式，目前為產品36、Watcom runtime170、未知1099；語意只從
[`fd2_semantic_index.json`](../data/ida/fd2_semantic_index.json) 合併，雜湊不符、
註記未落在函式起點或缺推論等級／證據時直接拒絕。handler IR 同步拆成
`native_call`、`unresolved_native_call`、`unknown`；三者都保留原始定位，前兩者
另內嵌推論等級與證據，名稱本身不解除正式 runtime gate。

2026-07-29 起，repo 根目錄 `AGENTS.md` 是跨 session 的操作契約，
`CLAUDE.md` 只保存專案意圖並指向它。Docker container 必須是有界生命週期：
one-shot 一律 `docker run --rm`，Xvfb 等背景程序須由 trap 收回；每批 RE／測試／
抓圖後都要檢查並停止、刪除不用的 FD2 container。toolchain rebuild 後另盤點
FD2/dangling images，只保留每種仍使用工具鏈的一份可重現 image，不做會影響
其他專案的 global prune。Capstone 仍只准使用 `fd2-cap-local`，不得建立 host venv。

## 3. 目標架構

```text
Input adapters (keyboard/mouse/gamepad)
        ↓ normalized Commands
Scene FSM: title → story/cutscene → battle → postbattle → town/shop/church/preparation
        ↓                         ↘ save/load snapshot
Campaign/Script runner          Persistent party + flags + inventory
        ↓
Battle rules / target selection / AI / animation scheduler
        ↓
Indexed native surfaces + RGBA/Ebiten presentation adapter
        ↓
UI skin/assets (FDOTHER/FDTXT/DATO/FIGANI/TAI/FDFIELD) + audio
```

Runtime 不應再讓 `main.go` 同時決定資料模型、輸入、規則和像素座標。下一階段先定義 interfaces，再搬移；在搬移完成前不增加新的 hard-coded handler。

## 4. UI interaction contracts

每個 contract 都必須有 state、可見 render model、輸入 command、side effect、headless test 與一個實機／截圖 gate。

| ID | UI / 流程 | 必須還原的操作契約 | 目前狀態 |
|---|---|---|---|
| UI-01 | Title/main menu | 上下選擇、確認、取消、save/load、游標音效與 focus state | partial；`TitleMenuState`／`TitleSlotState` 已與 Ebiten 輸入共用並有 deterministic trace，仍缺原版逐幀 E2 對照與完整 boot/campaign 接線 |
| UI-02 | Battle field | 游標格、鏡頭、可移動格、高亮、單位 HUD、方向／面向 | partial；`native-map-ch01-original-video.png`（320×200）與正式 handler 截圖 `native-map-ch01-remake-handler.png`（640×400）不是同一狀態。新圖由 `story_ch00_handler` 的 73 拍快速時鐘保留 LOADCH、JOIN、SPAWN 與 battle handoff，並唯讀掛載原版 `FDOTHER/FDSHAP/FDICON`，再套用 IDA 已證實的 FDFIELD b1 selector；這修正舊 b0 映射造成的敵軍圖像錯誤，也排除舊直接 `battle_ch01` 除錯入口造成的單角色假象。舊圖的場上單位、游標與 HUD 差異不能作為目前渲染器缺陷證據；較早 E1 raw 相機／游標欄位也不等於畫面一致。2026-08-10 已以同一 `FD2.SAV`、相機、游標、回合與單位狀態建立 DOSBox／重製逐幀範圍比較，最近鄰縮放後內容區只剩 22 個畫布邊界差異像素，並以 [`battle-field-ch01-scoped-compare-20260810.png`](../figures/battle-field-ch01-scoped-compare-20260810.png) 固定三欄證據；記為 ch01 scoped E2 candidate，仍不得外推至其他章節、一般玩家 CONTINUE 或完整戰場介面。詳見 [`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json)。 |
| UI-03 | Action menu | move/attack/magic/item/status/wait/end-turn 的可見項、enable gate、取消回上一層 | partial；`0x1741c` 的 raw cell index 是 `3*firstArgument + 2*secondArgument`。battle wrapper `0x18d8c` 的第一表固定 `[0,1,2,3]`，所以第二表為 0 時 cells 為 `[0,3,6,9]`、為 1 時為 `[2,5,8,11]`；舊的乘數倒置／`[0,2,4,6]` 斷言已勘誤，詳見 [CONTINUE overlay IDA evidence](../data/ida/fd2_continue_action_overlay_ida.txt)。2026-08-22 IDA 進一步證實共用 `0x117E7` 在 `0x12C0D==-1` 時呼叫 `0x16F55`，故空游標 `[21,15,18,12]` 四格不再限於 chapter0 CONTINUE；正式 battle 現要求完整 FDOTHER #2 後開啟面板，只有 direction3／END 會走四幀命令框關閉，再以 DATO #75、FDOTHER #5/#2、FDTXT `0x1A3/0x1A4/0x19C` 跑6＋4展開、YES／NO、4＋5收合、接受／取消回覆逐字形發布、完整句後再等待十二個60 Hz畫格並復原來源；只有YES才進敵方回合。缺任一資產在關閉命令框前拒絕，其餘三格仍失敗即關閉。runtime 以 caller-owned lifecycle 呈現 opening `0..3` 與獨立 closing `0..3`，並延後 command/item/spell/attack/wait side effect，直到第四個 close present 完成；因 native loop 無 delay call，只宣稱順序／present count，不宣稱毫秒時長。[8-frame Xvfb artifact](../figures/action-overlay-open-close-remake.png) 由目前 source 與玩家 FDOTHER.DAT read-only mount 產生。native command grid 亦已定為 320×200、每欄四列，label `(18+100*col,103+22*row)`、MP 右側、↑↓ wrap/←→±4 bounded；scenario raw command mask 已可 materialize。Docker/Xvfb 以 player FDOTHER.DAT 捕捉的 [悠妮 command-0 grid](../figures/native-command-grid-remake.png) 證實 mask→label→palette/font→renderer 路徑，非 original visual diff。`fdother.CaptureActionOverlaySnapshot`／`RestoreActionOverlaySnapshot` 與 `ActionOverlaySnapshotOrigin` 現已對齊原版 `0x175a9/0x17643` 的 72×72（`0x1440`）索引快照、每列 `0x1c8` stride、游標各減一格的 owner，並有失敗即關閉回歸；詳見 [IDA snapshot evidence](../data/ida/fd2_action_overlay_snapshot_ida.txt)。現行 Ebiten adapter 尚未消費此快照（每幀由整幅場景重畫避免殘影），因此 native snapshot backup/restore 的正式 consumer、其餘三格 owner、精確 tick／音訊與逐章 DOSBox visual diff 仍未關閉。 |
| UI-04 | Target/range/item selector | 武器 min/max reach、法術 range/AOE、item兩欄四列、不可用目標灰化、確認／取消 | partial；command/item targets與 observed item effects已閉合。`0x1b9de/0x184c0` 固定 compact prefix、input、layout與raw icon IDs；`0x18409` 的12-frame open11→0/close0→11及left/upper/bottom clipped rectangles已有Ebiten adapter。2026-08-22 已以同一未修改 `FD2.SAV` 從標題 CONTINUE 正常操作至悠妮 command 0 目標模式，原版四次擷取證實 `0x51A97` 20相位 LUT 是動態生命週期；重製也由一般 X11 輸入抵達同一座標與command ID的正式modal，並停止覆蓋非原版診斷短訊。原版為窄 `PLAYER-E2`、重製為deterministic `RUNTIME-E1`，兩側時鐘相位不同，不能宣稱逐像素一致。tracked item Enter transaction已接，但indexed effect presentation、完整weapon/AOE/LOS與其他commands的DOSBox visual diff仍fail-closed |
| UI-05 | Dialog | 上／下框、portrait anchor、文字避讓、控制碼、分頁／捲動、嘴型、輸入鎖 | partial；`internal/dato.MouthState` 已按 `0x16D00` cadence 接入更新迴圈，native frame/資源與所有 speaker layout 未閉合 |
| UI-06 | Battle HUD | HP/MP/LV/name、面板 sprite、數字 cell、依游標避讓、palette/clip | partial；需以 FDOTHER/UI loader 和截圖差分驗收 |
| UI-07 | Postbattle | result → handler → reward/roster cleanup → town/shop/rest/preparation 或 ending；不可預設直連下一戰 | partial；campaign schema 與 bounded menu trace 可表達，`town_ch02→preparation_ch02→story_ch02_pre→battle_ch02` 已有可重播 trace。目前24個 postbattle節點全數已接 authored binding；玩家第29戰已達正式 RUNTIME-E1；各節點只代表重製端 E1 admission，不代表一般玩家 E2。玩家第17、18、20戰已依 raw ch16、ch17、ch19 的直接控制流程分別接入60／61→61／62、55與83→84 runtime frontier，並保留 `town_ch18`、`town_ch19`、`town_ch21` 及 save/load 邊界。未綁定節點原有的泛用 `sync_party→set_chapter` 會繞過 runtime guard，現已移除並以空 beats 失敗即關閉。逐關戰間畫面與一般玩家路徑仍不足；直接位址證據見 [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)、[`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)、[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)、[`fd2_ch12_post_dispatch_ida.txt`](../data/fd2_ch12_post_dispatch_ida.txt)、[`fd2_ch05_post_dispatch_ida.txt`](../data/fd2_ch05_post_dispatch_ida.txt) 與 [`fd2_post26_28_dispatch_ida.txt`](../data/fd2_post26_28_dispatch_ida.txt) |
| UI-08 | Town/hub | 可見選單、離開、shop/church/preparation 入口、BGM/SFX、持久隊伍 | partial；`campaign.MenuState` 已與 `choice/town` runtime 共用。ch02 variant0 的 [`selection0–5`](../figures/town-hub-six-selections-original-vs-remake.png) 都已達原版 DOSBox／source-built remake raw RGB 整幀相同；variant1與variant2 selection0–4 另有修改 LOAD 路徑 E2，兩組五項都與指定 pulse 640×400 整幀 AE=0。Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape回selection5亦有原版 input trace；shop/church/preparation 與 hotel raw route/return trace已接，仍需variant2 selection5 的 BIOS 掃描碼／Enter、未修改玩家路徑與逐章route E2 |
| UI-09 | Shop | buy/sell、商品／角色／slot 游標、裝備詢問、金錢／庫存原子更新、secret gate | partial；四條production owner已接原版indexed compositor。ch02 variant1/3/5 service、purchase list／Yes-No／不足金／收件者，以及sell roster／items／Yes-No均有route-patched原版／production同狀態整幀相同。賣出短劍後的五個成功原子影格、`0x2D3FF`向上金幣滾動11／36元及返回原actor名冊cycle0／1又有九組整幀相同；正式擁有者分成success、credit、inventory publish三個邊界，不能直接跳新金額或提前移除背包。service2名冊／面板與service3來源提示／名冊、物品、目的提示／名冊已有route-patched partial E2，動畫相位尚未同步。這些party畫面使用screenshot-only typed/raw bootstrap，不代表完整campaign/native save E2。正常campaign JOIN→LOADCH、四種mutation→town→JSON save/load及ch26祕密商店已有E1。尚待recipient scroll、no-recipient/full、service2動畫相位與mutation／restore、service3 mutation／empty／full、其他章節、原版存檔與未修改一般玩家E2；不再把service3五個穩定子面板列為未做 |
| UI-10 | Church | revive、class change、費率、候選過濾、確認／取消、缺資料 fail-closed | partial；class path 已對齊 `0x31385→0x31793→0x311DC→0x19953`：Lv>=20、portrait<0x12 且 !=7，三列可見候選、上下 bounded，special>optional>default 自動解析唯一 target，再以左右 Yes/No 確認。`0x31019` 的 FDICON＋四段 FDTXT row、FDOTHER#14 entry16 panel 與 `0x1974c` 六幀 opening 已成 indexed compositor。候選確認／取消會先跑 `0x2d31b` 五幀 closing＋source restore；`0x19953` 已接 FFFC 動態角色名、FDOTHER#2 cells16/17、48/49與51/52 normal/pulse、四幀 opening／`0x197e5` 四幀 choice closing，之後再跑 dialogue closing 五幀＋source restore，最後才 mutation／返回。所有幀只由 Draw acknowledgement 推進。`0x3072f` stable scene 已由FDOTHER#5 raw grid/four-mode digits、FDOTHER#14 entry1、DATO#131與FDTXT585/586合成；`0x2d669`四幀開關、closing source restore及`0x2d85f`兩-tick selected pulse均接runtime並有原版資源artifact。FD2.SAV、raw service0 command overlay與未接callee仍fail-closed |
| UI-11 | Preparation | 城鎮出發確認／無城鎮記錄詢問、依名冊門檻略過或進入部署、可選15／19筆另加固定 record0（總上場16／20）、取消、最終確認、進戰場 | partial；`0x2cad7` 與 `0x2d093→0x318ad` 的分流已接資料模型與原版提示。城鎮路徑保留實際 town frame，使用 FDTXT `0x201` 於 `(95,119)`；無城鎮路徑依 `0x2cc04` 清成黑畫面，使用 FDTXT `0x19a` 於 `(100,119)`，肯定結果在完整關框後才存檔。兩者都使用 DATO #75、FDOTHER #5 對話框、FDOTHER #2 Yes／No 與 6＋4＋兩 tick 脈動＋4＋5＋還原生命週期。`0x318ad/0x31e80` 的三區背景、計數、10 欄角色格、游標、彩色／灰色繪法、四向輸入與 `0x17fc0` 狀態已接；`0x320fc` 證實 selection byte i只重排 persistent record i+1，record0固定且不消耗quota。`0x1297d` 的有號 BIOS 低字差值與可見 `0,1,2,1` 待機週期亦已接。`0x31d3c` 最終確認沿用相同生命週期，文字為 FDTXT `0x292`，呈現原畫面後才處理結果。呼叫端也已閉合：城鎮出發 `0x2d16b` 收到 0 會退出，直接整備 `0x2ccd6` 收到 0 則重選。缺任一原始記錄或資源即退回。README 所列整備圖均為 E1 原始資源合成，不是 DOSBox 截圖或正常晚期戰役存檔。跨畫面初始相位、有效晚期存檔及原版實機差分仍缺。`0x1f42d` 屬戰場進入演出，不再列為選人視窗動畫 |
| UI-12 | Save/load | scene-safe boundary、campaign cursor、flags、party/inventory/equipment、version/checksum、四槽 selector | partial；remake title LOAD 已還原四槽 bounded selector 與原版 indexed compositor。合法 IDA 9.4 固定 reader `0x2602c..0x26098`、writer `0x30012` 及其僅有的 `0x2ccb6/0x2fd93` 戰間呼叫者；兩端只處理 metadata `+0..+9`。production 以綁定參考 EXE 雜湊與 `0x526b9` 的 editable gate table 將 raw chapter 1..29 還原到既有 town／preparation node，先完整驗證 persistent record→typed party、節點型別及重複 identity，再一次套用 campaign cursor、gold、party 與四個 raw option bytes；錯誤不部分 mutation、不誤轉 JSON loader。ch21／ch27 inventory postbattle gate 已在存檔前完成，LOAD 不重播。空槽及修改存檔 chapter1 有效槽畫面均與 DOSBox 全幀 RGB 相同。2026-08-22 未修改原版 CONTINUE 戰場按F5後畫面不變且FD2.SAV雜湊不變，動態支持戰場不能建立chapter slot；合成有效槽 restore 是 E1，仍缺正常打完戰鬥後由酒店／整備建立的有效槽 E2、CONTINUE current-battle、metadata `+10..+39` 其他可能 consumer、刪除／覆寫 |

### 4.1 教會轉職後的重製存檔邊界（RUNTIME-E1 契約）

原版 `0x31793` 目標表、`0x314A7..0x3157A` 寫入與職業顯示／裝備消費端已
固定轉職會改動的持續角色欄位；這裡不重新推定 `FD2.SAV` 位元組配置。重製自有
JSON 存檔只在離開教會、回到可編輯 town 節點後建立，並須保存正式
`applyChurchClassChange` 已發布的完整 typed roster，而不是由測試手動捏出轉職後角色。

現行整合回歸依序執行：town 選單進教會、`NativeClassChangeTarget` 解析唯一分支、
`applyChurchClassChange`、`leaveChurch`、安全 town 節點存檔、清除目前記憶狀態、再由
相同槽位讀回。讀回後至少核對 campaign cursor、membership／join order／deployment、
portrait、class、raw class、battle figure／map selector、MV、EXP、HP／MP、基礎與
裝備重算欄位、背包及裝備旗標；教會模式、候選陣列與戰場狀態不得跨節點殘留。

2026-08-22 實跑另發現讀檔前的 `churchMode`、候選分支與 indexed 工作原本會
殘留到 town；`loadGameFromSlot` 現在先原子清除這些未序列化暫態，再進入存檔節點。
這項契約只證明重製正式轉職交易可穿越自有 JSON 的 town 邊界，列
`RUNTIME-E1`。它不證明原版四槽 `FD2.SAV`、未修改 DOSBox 的轉職後存讀檔、精確
隨機成長序列或逐像素教會畫面；這些仍依 UI-10／UI-12 的既有 E2 門檻處理。

`0x50` persistent roster 的匯入邊界已開始由 raw snapshot 推進到具型別
view。合法 IDA Pro 9.4 重核 `0x112A5` constructor、`0x1145A` equipment
recompute 與 `0x17EEF→0x17FC0` status consumer，固定 `+5/+6/+7/+8`、
八個 item cells、五個 command bytes、race/class/level、六個 transient
bytes、base AP/DP、MV/EXP、DX、HP/MP 與 AP/DP/HIT/EV offsets。
`fdsave.PersistentRecord.View` 只投影這些已證實欄位並保存 signed word；
raw presentation key 與 identity 分開，尚未接名稱／class／sprite resolver
或 normalized `battle.Unit`。直接證據見
[`fd2_persistent_roster_ida.txt`](../data/fd2_persistent_roster_ida.txt)。

raw ch06 post（玩家第 7 戰）的分支已由 IDA Pro 9.4 主判讀與 Docker
Capstone 覆核：先 `sync_party`，只有 `[0x53ad5]+0x11 == 1` 才呼
`unit_inactive(43)`；inactive 走 dialog index5，active 才執行 `0x233c6`
layout、dialog index4、JOIN12。layout arrays 為
X=`[12,11,13,10,14,10,14,9,15]`、Y=`[4,4,4,5,5,6,6,7,7]`、
pose=`[0,0,0,3,1,3,1,3,1]`，另有 special slot43=`(12,7,pose2)`，
camera raw `(6,2)`。先前把 slot43 解讀成 96-slot 容量中的空白 record 是
錯誤斷言：map6 正確 runtime 順序是 party 9 人→group1 25 人（34 slots）→
event25 group2 10 人（slots34..43）。更早的 map6 event26 只在六個原始格子
的 selector0 路徑執行：觸發單位 raw `+6 != 0` 時，以 `0x3419C(9,27,0)`
保留 slots9..27 的 `+0x34` 高四位並清低四位，才寫 state16=1。`0x34924`
在 enemy turn10 先要求 state16==1，通過後依序 spawn2、pan `(16,10)`、
ACTING30、FDTXT index2，最後才寫 state17=1；ACTING30 本身直接引用
slots34..43。重製現於向左一步第七拍的既有 selector0 owner 執行 event26，
event25 同時要求 turn10／state16==1；未踏格反例不增援。戰後再以精確
34／44 frontier、state17==1 的 44-slot 細化與 raw byte5 bit0 接通
`postbattle_ch07_persist→town_ch08`，缺任一產生端即失敗關閉。直接證據見
[`fd2_ch06_post_event25_ida.txt`](../data/ida/fd2_ch06_post_event25_ida.txt)；
目前為 E1，尚缺未修改一般玩家路徑的 DOSBox E2。

> **歷史快照（2026-08-02；後續現況以本文件的 2026-08-09 稽核為準）**

2026-08-02 重新核對主迴圈後，撤回 2026-07-27 的錯誤同號索引斷言。
`0x25e23` 以目前 raw chapter 選 post-handler，handler 自己才增加章節；因此玩家第N戰必須執行
`ch(N-1)_post`。`postbattle_ch14_persist` 現改接 `ch13_post`，`postbattle_ch15_persist`
改接 `ch14_post`（native `0x239bd`，條件對話→sync→JOIN15→set_chapter15）。raw
`ch15_post` 實際屬於第16戰戰後；「`postbattle_ch16_persist` 仍 unbound」僅是
2026-08-02 的歷史狀態，已由 2026-08-09 的 E1 production 接線勘誤取代。
稽核工具不再把非空 binding 視為正確，而會比對這個已證實的零基索引關係；截至本輪，
`postbattle_ch04/05/08/09/10/11/12/13/18/19/24/25/29` 共13個既有同號 binding 會報
`active_index_mismatch`。逐章複核後，ch04/05/09/11/12/19/25 已安全移接至前一號 raw
binding；ch08/10/13/18/24/29 因未知呼叫、缺 mapping 或來源位址可疑而撤回錯接並失敗即關閉。
後續以 IDA 逐一補驗共享尾段並啟用 raw ch25、ch27 與 ch05 的正確 owner；**當時歷史稽核為**
13個 active、11個 blocked，沒有 mapping complete、index mismatch 或 inline bypass。

同輪重讀 `ch15_post` 已補足 layout evidence，但沒有解除 gate：native 先寫 slots `0..15`，再寫
special raw slot65=`(28,30,pose2)`、camera `(22,25)`，並由 acting resource49 操作 slot65；之後掃
slots66..73。IDA Pro 9.4 已確認 `sub_3453E` 以參數乘 `0x50` 後讀 runtime record `+5 & 1`，
所以 `0x42..0x49` 確為 slots66..73，不是角色或事件編號。inactive 計數、raw global
`[0x53bef] > 18` 與 record word `+0x42 >= 0x140` 已有受限條件原語；但一般玩家原版 runtime
shape 尚未擷取，故當時不接入 campaign runtime；此歷史 gate 已由 2026-08-09
四分支 E1 回歸解除，仍不代表一般玩家 E2。

條件模型現新增 `native_inactive_count_gt`：它只接受 address-independent 的明確 slot list 與
threshold，逐 slot 要求 native record byte5 provenance，再以 bit0 inactive count 做嚴格 `>` 比較；
缺 slot 或缺 raw byte5 不得退回 HP/OnField。這可表達 ch15 的第一個 predicate，但不替代同一
handler 的 raw global/record-word comparisons，因此當時 ch15 仍不可解除 implementation gate；
後續已由 production handler 與真實回歸補齊。

本輪再以 Docker Capstone 重讀 ch15 `0x23a0a..0x23b52`：`0x1a5b9` 明確對全域
`[0x53bef]` 做 `inc`，而 handler 在 `0x23a9a` 直接比較 `>0x12`；`0x23aad..0x23abb`
則從 `[0x53a45]` 取 runtime record 的 raw u16 `+0x42` 並比較 `>=0x140`。remake 現以
`State.NativeRoundCounter`（只在有 raw provenance 的載入狀態初始化／遞增）及
`Unit.NativeRecordWord42`／`HasNativeRecordWord42` 保存兩個來源；compiler/runtime 新增
`native_round_gt` 與 `native_record_word_gte`，缺 provenance 或 offset 不是 `0x42` 一律
fail-closed。這兩個 primitive 有獨立 compiler／BeatRunner regression；ch15 已有可編譯的
OR／else CFG，但 `[0x53a45]` 的一般玩家 runtime shape、JOIN-time persistent record 與 save
boundary 尚未全部閉合，因此當時 `postbattle_ch16_persist` 仍維持 unbound；此歷史語句已由
2026-08-09 的 production E1 勘誤取代，不宣稱已達 E2。

後續 producer trace 又閉合一層：constructor `0x10d7f..0x1100c` 在 `0x10fe9` 將生命值
輸入寫入新 runtime record 的 `+0x40` 與 `+0x42`，`0x10ff1`／`0x10ff9` 則將魔力輸入
寫入 `+0x44`／`+0x46`，最後才呼 `0x1b750` 重算衍生欄位。高階分支的生命值公式是
`u16(high[+2])*level`，魔力公式是 `high[+4]*level`；低階分支分別是
`u16(lower[+3])+lower_aux[+6]*(level-1)` 與
`u16(lower[+5])+lower_aux[+8]*(level-1)`。`tools/export_units.py` 因此在具備完整
原始表來源時導出 `native_record_word42` 與 `native_record_word46`；
`sync_native_selector_fields.py` 將兩者連同初始命令遮罩同步到 33 張地圖。載入器只在
欄位具備來源時，以 `word42` 初始化 `HP/MaxHP`、以 `word46` 初始化 `MP/MaxMP`；
舊式可編輯列缺欄位時仍沿用既有正規化數值，不從其反推原始欄位。表格不完整或 selector
未覆蓋時不輸出，相關原始消費端維持失敗即關閉。

sync boundary 也已補上來源傳遞：`syncPartyFromBattle` 的 snapshot 會保留
`NativeRecordWord42/HasNativeRecordWord42` 與
`NativeRecordWord46/HasNativeRecordWord46`；`applyPersistentStats` 在 LOADCH／戰場重建時
再把兩個 raw word 複製回 runtime，且不會由 `HP`／`MaxHP` 或 `MP`／`MaxMP` 反推。
`MapSelectorSlot` 則是每場戰鬥由 `0x11019` cache 重建的 runtime `unit+2`，不可跟著
persistent overlay 跨戰複製；只有其 raw `+7` key 可持續。此修補只關閉資料遺失邊界，
不代表目前所有 units JSON 都具備 constructor input，也不解除 ch15 handler 的 binding gate。

為保留 ch15 的實際 OR 控制流，條件模型另新增受限的 `native_any_of`：compiler 只允許
已閉合的 `native_round_gt` 與 `native_inactive_count_gt` 子條件；runtime 在任一 raw 子條件
已證實為真時才回 true，所有子條件都無法取得 provenance 時仍回 error。這是 compound gate
primitive，不是開放式腳本 expression language。ch15 candidate 已可編輯且可編譯；現行
campaign 不執行它的原因是 runtime／persistent／save 證據門檻，而非檔案擁有者問題。

另新增 [`ch15_post_cfg.json`](../../remake/assets/cutscenes/handlers/candidates/ch15_post_cfg.json) 作為
address-preserving candidate：它把 `0x23a9a` 的 `round>18 OR inactive_count>4` 與
`0x23aad` 的 `else +0x42>=0x140` 寫成 nested editable CFG，並保留 dialog/acting/JOIN/
set-chapter source addresses。2026-08-02 的 IDA 直接指令重核修正一個高風險錯置：
`0x23b1f` 會跳過 JOIN18，故 JOIN18 只在 `+0x42>=0x140` arm，不能放在共同尾端。
IDA 直接指令另閉合入口 producer：`sub_320FC` 只重排、不刪除 persistent records；
`sub_1088D` 依 FDFIELD header 建立固定 party slots，先完整複製 persistent records，不足才補
byte+5=1 的空 record，最後 `sub_10B4E(0)` 無條件 append 所有 group0 rows。raw ch15 對應
第16戰 map15：16個 party slots 加60筆 group0，故 candidate context 固定76。這是雜湊綁定
資料與直接指令的靜態閉合，尚非一般玩家 E2；JOIN-time persistent record、branch trace 與
campaign consumer 尚未閉合前，原始 `ch15_post.json` 與 `postbattle_ch16_persist` 都維持
fail-closed；這是歷史 gate，已由 2026-08-09 勘誤解除至 E1。
直接證據見 [`fd2_ch15_post_ida.txt`](../data/fd2_ch15_post_ida.txt)。

### UI restoration execution plan（2026-07-27）

UI 還原採「先操作契約、再 renderer fidelity」的垂直順序，不把單一 native offset
或一張漂亮截圖當成完成：

1. 先以同一條 deterministic trace 串起 `title → story → battle → postbattle → town/shop`；
   `TestUIShellVerticalTraceKeepsPostbattleTownAndShopBoundary` 已固定 battle win 必須經
   editable postbattle node，再進 town，shop 結束後回 town，不能直接進下一戰。
2. 對每個節點保存 state trace、可編輯 JSON 轉場、headless regression 與 screenshot artifact；
   原版沒有 E2 ground truth 的畫面維持 partial/blocked，不以 normalized UI 升級 native parity。
3. 再將 battle field、action overlay、command grid、target selector、dialog、HUD 依原版
   input/layout evidence 接入同一個 modal state stack；native target/effect 未閉合前 confirm
   必須 fail-closed。
4. 最後才做 indexed compositor、palette、FDOTHER/FDTXT/DATO 資源差分與逐章 campaign trace。

目前 SDD-3 已進入 `[~]`：title／campaign hub／shop 的 state chain 與既有截圖可重跑，
但 battle field/action/dialog 的同一路線畫面差分尚未完成。這是「可操作 shell 有進度、
原版 UI 尚未等價」的明確判定。

### UI acceptance gate

在 `UI-01…UI-12` 每項至少有一個 deterministic input script、預期 state trace 和 screenshot artifact；只通過 Go unit test 不算 UI 完成。截圖測試需記錄解析度、幀號、輸入序列，並比較 cursor/menu/dialog/panel 的 bounding boxes。無法取得原版 ground truth 的項目標為 blocked/assumption，不得用「看起來合理」關閉。

### UI-03 native command data contract（E0 partial）

原版 `0x1c269` 將 0x50-byte unit record 的 `+0x1a..+0x1e` bit array 展開為
`command_id = byte_index*8 + bit_index`（0..39）；`0x4e516(id)` 對應到
`0x619fd + id*7` 的 command record。construction path `0x10f7f/0x11399` 只 copy initial
4 bytes 至 `+0x1a..+0x1d`、清 `+0x1e`，而 `0x1d7fb` 可依 `id/8` 將 runtime bit OR 回 array。
`0x159fa` 另要求 `command_record[5] <= unit.current_mp`（unit `+0x44`）。

FDFIELD 26-byte roster 的 source bytes `b13..b16` 現由 `parse_field.py` 和
`export_units.py` 保留為 `initial_command_mask`；battle `Unit.NativeCommandMask` 以五個 bytes
materialize，並只提供原版 byte-major／low-bit-first 列舉與 bounded OR。這條管線刻意不覆蓋既有
`Spells` normalized list：後者是 legacy gameplay approximation，不能再被宣稱為原版 raw command source。

Scenario party override 也已採同一欄位：`PartyMember.initial_command_mask` 只接受空值（舊 editable
scenario）或精確四 bytes；`LoadScenario` 對其他長度 fail-closed，避免截斷後偽造另一個 command inventory。
`gen_campaign.py` 從 EXE `character_defaults.json` 依角色 index 帶入該 raw source，並已重產
`ch01..ch30.json`，且不覆寫既有手工驗證的 scenario 欄位。ch01 的悠妮為 `[1,0,0,0]`，其餘初始三人為全零，均直接來自
character-default table，絕非由 legacy `Spells` 反推。戰後 persistent snapshot 亦保留完整五-byte runtime
mask，因此 `0x1d7fb` 型 level-up OR 不會在 town/preparation 邊界遺失。

`0x4e516(id)` 的 backing bytes 對 `id=0..35` 與 EXE `spell.json` 7-byte rows 逐 byte 相同，故這個
已證實範圍的 record layout 可共用 `dmg:u16, hit:u8, dist:u8, range:u8, mp:u8, target:u8`；MP gate 是其
第 5 byte 的獨立直接證據。IDs 36..39 雖可由 pointer arithmetic 取到相鄰 7-byte data，FDTXT labels
卻是空字串／系統訊息，且所有 FDFIELD + character-default initial masks 實測最高只設到 ID 30。故不得把
36..39 加進 `SpellBook` 或宣稱第五 byte 的 dynamic path 已被實機素材證實。

runtime 對 native path 使用 `NativeCommandRecord`，不使用 normalized `Spell`：它將 bytes `+3/+4/+5/+6`
明確暴露為 `SelectionMode/EffectMode/MPCost/TargetCode`。loader 可讀現有的 physical export `spells.json`，但逐 row
重解 `raw` 的七個 bytes 並要求所有 JSON field 一致，且只接受連續 ID 0..35；任一 editable presentation 欄改壞、
缺列或未知 ID 都 fail-closed。Game bootstrap 只把這份 immutable book copy 到每個新 `battle.State.NativeCommandBook`；
它不取代 `SpellBook`，也尚未驅動 UI/effect。

選單 confirm 的 execution contract 必須再區分：`0x1cff0` 先完成 raw command ID 的 selector/target path，再由 ID 分派。`0..8`、`0x18`、`>=0x1c` 呼叫 `0x2a6bd(unit, id, target, scratch)`；`0x09..0x17` 與 `0x19..0x1b` 先走 `0x1d6c8(id)` 的四輪 palette flicker，之後才進 `funcs_1541f[id]` jump table。這證實 command 0 屬 generic pipeline，**不**證實它等同 normalized `Spells[0]`、也不允許在未解 callee 前為它填 damage/target contract。native-grid confirm 對無完整 effect trace 的 ID 必須維持 fail-closed。

`0x2a6bd` 的 command-0 entry 本身也不能被誤讀成 effect formula：它以 ID 作 presentation mode，command 0 不走 `>=0x20`／`0x18..0x1b` 的 special early branch，而採 generic compositor defaults，並經 `funcs_2ac25[0]=0x26152` 多輪繪製 320×200 battle buffers、FIGANI／FDOTHER cells、present/tick。這是已證實的 renderer boundary；HP、status、MP mutation 的責任仍需沿其後續 callee／caller 另行 dataflow 證明。

> 歷史快照（2026-07-26）：下段保留當時「尚未接 campaign」的研究邊界；它已被
> 2026-08-22 的正式 `battle_ch30→ending` E1 現況取代，不可再當成目前阻擋狀態。

2026-07-26 official IDA recheck further closes the `0x2c405 → 0x2c548` hand-off boundary. After the 500-pass phase-0 scroll, native code frees staging, allocates one `0x1f400` and two `0xfa00` buffers, loads `TAI.DAT` entry `3` and `FDOTHER.DAT` entry `0x38`, and blits the latter into the first indexed buffer before iterating party records from `[0x53bfb]-1`. This resource/buffer contract is now recorded in `assets/endings/native_2c548.json`/worklist; it does **not** authorize a PNG or generic-fade adapter. The later standalone `internal/ending.MontageCycle` now consumes the proven per-party scheduler and indexed renderer against original resources, but it remains separate from campaign／input handoff and does not claim general-player E2.

Runtime audit correction (2026-07-28): the prefix player already executes
frame12..108 and both 40/200-pass compositors, but its preview adapter retained
a single `queued` latch after the first `0x2c39b` block. That made the second
text call at `0x2bf1c` unreachable despite the recovered Player state machine.
The adapter now releases and resets the latch only after every page/line of the
current block is acknowledged, so both chapter26/final branches run in native
sequence. The preview also loads FDTXT resource31 and FDOTHER font resource4
and enables the existing `0x2c405` bridge. A player-archive integration test
now proves the complete recovered path stops at `0x2c548`, not the obsolete
first-text or `0x2c172` gates. This still does not authorize the montage or a
campaign terminal route.

The same IDA pass closes the previously omitted `0x29164` mirror branch. When `unit[+6]==0`, the native path at `0x2927e..0x29357` retains the 9-pass `stage=8..0` and `stage*6` DAC cadence, but addresses the primary FIGANI frame at `staging+0x140-stage*10`; `arg4==0` gates the extra TAI#3 and secondary-FIGANI draws. This is now an explicit `mirror_branch` record in the editable montage schema with a loader regression. It remains evidence-only: no PNG approximation or runtime renderer permission follows from the transcription.

`Montage.PlanMirrorFigureFade(unitSide,sideFlag)` exposes that schedule as a pure, testable plan (nine exact offsets and DAC deltas, with the `arg4==0` secondary/platform gate). The planner is deliberately not a pixel adapter and keeps the montage fail-closed.

> 歷史快照（2026-08-09）：下段的 standalone／terminal blocked 是當時狀態；正式
> campaign 已於後續接通來源約束的 E1 終局，現況與剩餘缺口以 `58` 為準。

2026-08-09 `0x2c548` standalone executor closure：新增
`internal/ending.LoadMontageCycleAssets` 與 `MontageCycle`，以原始
`FDOTHER#56` backdrop、`TAI#3`、`FDOTHER#5` grid、`FIGANI`、`DATO`、
`FDTXT_031/FDTXT_000` 和 FDOTHER#4 font 建立 provenance preflight；再依
IDA 證實的 slot swap、九段 mirror/non-mirror fade、20 次 secondary loop、
primary descriptor `+6` tick、portrait 220/440 loop、DATO mouth countdown、
五個 FDTXT destination 與最後 64 段 palette 收尾逐步產生 indexed VGA frame。
Docker 以玩家原始資源跑完兩個 unit side branch 的完整 cycle regression。
這是獨立可編輯 renderer，尚未接輸入事件映射或 campaign；`0x10620` 的
raw word 比較、`0x4e031` 的 `0x41a→0x41c` 複製，以及 `0x28a64` 的共用
清理尾端已另以固定位址證據收窄，但不提升為按鍵或戰役語意。campaign
terminal、一般玩家 E2 仍未閉合，因此不解除 `postbattle_ch29_persist` 的
失敗即關閉狀態。完整原始位址與工具雜湊見
[`fd2_ch29_montage_ida.txt`](../data/ida/fd2_ch29_montage_ida.txt)。

2026-08-09 raw input primitive correction：IDA／Capstone 已證實
`0x10620` 只比較 `word[0x41a]` 與 `word[0x41c]`，`0x4e031` 只做
`word[0x41c]=word[0x41a]`；`0x2c950`／`0x2c961` 是 ch29 montage 內的
呼叫點，但 `0x4e031` 也有其他 caller，不能命名成終局專用按鍵消費器。
`0x28a64` 是 `0x28784` 共用清理 epilogue，不是 campaign 返回 owner。
重製端新增 `fdsave.NativeBIOSKeyboardState`，只保存這兩個 raw word 的
比較／複製契約，測試不解碼按鍵、不接 generic skip。按鍵映射、外層事件、
`0x2c194..0x2c39a` handoff 與一般玩家 E2 仍失敗即關閉；證據見
[`fd2_ch29_input_cleanup_ida.txt`](../data/ida/fd2_ch29_input_cleanup_ida.txt)。

2026-08-09 ch29 post-montage tail raw schedule：IDA／Capstone 又閉合
`0x2c194..0x2c39a` 的資源與迴圈契約：FDOTHER #60 的前置 320×200 單影格、#58
的 20-entry frame table、#57 的 768-byte VGA 調色盤、`unit+6/+7/+0x56/+0x57` 寫入、
`0x28a6c(0,1)`、`0x11d40(0,255,0)`、`0x2935b`、20／78 tick、
`0x1f882`，以及最後 FDOTHER #59 的解碼與釋放。三組 20-byte 表以固定版
LE 線性位址 `0x525dc..0x52617`（object 2 檔案偏移 `0x523dc..0x52417`）輸出並帶雜湊；兩組 selector table 分別寫入
record0、record1 的 `+7`，並各自在 `<0x4c` 時即時計算該筆的 `+6=2`，
否則為 0（`0x50` stride，故 `[0x53a45]+0x56/+0x57` 是 record1，而非 record0
高位欄位），第三組寫入 `[0x540ff]`。`MontageTail.Plan` 只產生 raw
entry、不寫入 `battle.State`，也不命名欄位；近似模式另由 `MontageTailPlayer`
消費 20 組已驗證的 TAI／BG／FIGANI 資源、FIGANI descriptor `+6` 延遲與
FDOTHER #58 疊圖，最後保持 #59。這關閉「尾端完全未知」的過時斷言，但不是
精確 `0x28a6c` renderer，也不關閉輸入事件、campaign／town／
 shop／整備／save handoff 或 `postbattle_ch29_persist`；證據見
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)。

2026-08-12 loader／renderer 邊界追加證據：`0x1088d(0x1e)` 由 FDFIELD
#90/#91/#92 建立 31 筆 deployment runtime records；前綴逐筆複製 persistent
`0x50` records、覆寫位置／selector cache／狀態欄並執行 `0x1b750`，其餘筆數
只標成 inactive。`MontageTailLoaderBaseline` 已把這個 **post-loader baseline**
做成唯讀值拷貝與真實素材回歸，但 `0x2c548` 介於 loader 與 `0x2c2a6` 之間，
可能觀察或改動相同 runtime image；所以它明確不是 renderer admission。
IDA／Capstone 也已證明 `[0x540ff]!=0` 的 `0x28a6c` 仍載入 TAI／FIGANI／BG、
經 `0x29164→0x2939d` 合成並輸出 VGA，只略過 `[0x540ff]==0` 才呼叫的
`0x29f72` 一般戰鬥結果解析器。尚缺 `0x2c2a6` 呼叫當下完整 records/globals、
精確 `0x28a6c` renderer、原版輸入／時序及 E2；近似播放器不能解除忠實模式的
20 段失敗即關閉邊界。

2026-08-09 ch29 terminal caller correction：IDA／Capstone 直接固定
`0x25e23` 以 `[0x53c03]` 消費 raw table `0x51de9`；目前已證實
index26→`0x250cc`、index29→`0x25757`。`0x25757` 在 `0x25970→0x2bce5`
後於 `0x25975` 自迴圈；`0x250cc` 則依 `0x24b14(0x64)` 分成 success exit
與 ending self-loop。這修正「`0x25757` 可直接接 preparation」或「table index
等同玩家戰次」的過度斷言，但不提升 `0x24b14` 名稱、frame 語意、一般玩家
chapter provenance 或 campaign owner。完整 caller 證據見
[`fd2_ch29_terminal_callers_ida.txt`](../data/ida/fd2_ch29_terminal_callers_ida.txt)。

Correction: `[0x53a81]` in this call chain is `FDOTHER.DAT#5` (the dialogue-frame bank), not DATO. Official IDA shows `0x2c773` calling `0x168b6(destination=C, stride=0x140, arg8=5, argC=7, arg10=5, arg14=5)` to build that dialogue frame/grid; the later DATO pointer `[0x53a85]` is pasted by `0x4e8af`. This is a resource/layout boundary only; it does not authorize a single-static-portrait or guessed mouth cadence adapter.

`internal/dato` now provides the corresponding resource boundary: four-frame DATO LLLLLL parsing, the native `0x4e916` high-run codec, opaque-zero semantics, and bounds-checked indexed blit. `MouthState` preserves the verified `0x16D00` cadence as a pure tested adapter and is used by the dialogue update loop; complete DATO runtime resource binding and ending UI integration remain explicit gates. For the separate `0x24618` transition, `fdother` now preserves the native full-buffer seed and 456→320 viewport copy contract, but runtime LUT selection and indexed presentation are still fail-closed.

`fdother.PlanNativeDialogueFrameGrid` now transcribes all 49 `sub_1685c` calls for the proven `0x168b6` invocation (12 fixed cells, two `3×2` loops, and a `5×5` raw grid); `Montage.PlanDialogueFrameGrid()` delegates to it. **2026-07-27 correction:** the former ending-only formula omitted `a3=5` from `v6=dst+stride*a4+a3` and mixed byte/stride terms in several placements. The exact first cells are now offsets 2245/2328 (not 2240/2323), portrait-overwritten grid origin is 3208, and the final grid cell is 23752 (not 22812). The common planner exposes only resource indices and byte offsets; cell semantics and DATO mouth timing remain intentionally unnamed.

2026-07-28 `0x2c548` portrait-loop closure: direct Docker Capstone recheck of
`0x2c430`, `0x2c43f`, and `0x2c7cf..0x2c9a9` fixes the DATO paste at byte
offset `0x0c88` and the local countdown at zero initially. A zero value is
replaced by `(random_byte&0x1f)+0x28` without immediate decrement; otherwise it
is decremented first. Results below two choose DATO pointer-table entry 3,
otherwise entry 0. Nonzero party loop indexes run 220 iterations; loop index
zero (the swapped slot 1) runs 440, so it alone reaches tick 220 and switches
the epilogue to FDTXT_031 #45. `ComposeMontagePortraitFrame` now restores the
FDOTHER#5 grid, pastes the selected original DATO frame, and renders all five
FDTXT fields with the proven CD/4C/transparent glyph ABI. It accepts only the
observed FFFE line break and fails closed on other controls. Original-archive
regression exercises both normal and mouth/special-epilogue frames without
mutating the restore buffer.

Timing correction: the calls to `0x17aa9(1)` in the 20-frame secondary loop
and portrait loop consume one native BIOS tick, not one millisecond. The
editable schema and planner therefore use `frame_delay_ticks`; primary FIGANI
descriptor byte `+6` is likewise a tick count. This correction is local to the
montage and does not rewrite separately evidenced millisecond waits elsewhere.

Codec correction: the `0x1685c→0x4e9bb` path copies each selected `FDOTHER#5` cell's width×height bytes directly (`rep movsb`). It is not the `0x4e916` high-run codec used by DATO portraits. `fdother.ParseLMI1RawEntry` now preserves this separate contract and has a real entry-1 byte regression.

`RenderDialogueFrameGrid` now executes that narrow primitive against the 49 verified placements, writing literal zero bytes and preserving native overwrite order. It is only the C-buffer frame layer; DATO portrait paste, text glyphs, input, and ending runtime remain gated.

`RenderDialogueFrameGridResource` now runs the same contract against the player-provided `FDOTHER.DAT#5` entries 1..17, with missing assets failing closed. This verifies the raw resource boundary without promoting the frame bank into a guessed semantic UI renderer.

`RenderDATOFrameAt` and `dato.Frame.BlitAtOffset` now cover the separate opaque `0x4e8af` portrait paste with explicit stride/offset inputs. The caller must supply the recovered staging destination (the ending call site uses `staging+[0x53c67]`); the helper deliberately does not turn that global into a universal UI anchor or infer mouth timing.

`battle.RenderNativeItemPanelBaseResources` now executes the complete proven
`0x17eef` base composition from player archives: corrected 49-cell opaque raw
grid, DATO frame zero at `(8,10)`, then FDOTHER #5 entries 20/21 at `(92,7)`
and `(5,94)`. It stages all writes and commits atomically. The two large
entries use the newly explicit opaque `LMI1Entry.BlitOpaqueAt`, because
`0x4e8af` stores every decoded byte, including palette index zero. The older
statement that `LMI1Entry.BlitAt` represented a “transparent 0x4e8af” was
wrong; transparent callers retain the separate `BlitAt` API. `0x17fc0`
dynamic overlays are deliberately the following atomic pass; only Ebiten
presentation remains outside this base primitive.

Item text-helper correction: official IDA of `0x15f84/0x16559` resolves the
apparent `[0x53a85]` lifetime contradiction. Ordinary words call
`0x4ea2a([0x53a75],glyph,destination,stride,foreground,shadow,background)`;
boot `[0x53a75]` is FDOTHER #4's packed 16×16 1bpp font. `0x16559` instead
indexes the currently loaded DATO `[0x53a85]` and repastes a mouth frame for
dialogue control/animation. Therefore the old “`[0x53a85]` CJK glyph
container” assertion is deleted. For item panel calls, the proven glyph style
is foreground 205, shadow 76, background 0; control-bearing strings must
still fail closed.

Complete item-panel indexed compositor: `RenderNativeItemPanelData` executes
the full `0x17fc0` schedule over the recovered base. Bars preserve
`0x18795→0x17d6f` arithmetic and raw entries 23..30; numbers preserve
`0x1875d/0x187d6` comparison colours, zero padding, six-pixel advance and
overflow entries; icons preserve entries 53/54 and conditional 55..57; the
three text calls use FDTXT #0 plus FDOTHER #4 style 205/76/0 and reject any
control word. `RenderNativeItemPanelResources` selects DATO from record `+7`
and commits base plus data atomically. Synthetic tests cover subpass pixels,
zero bars and failure atomicity; player-archive regression covers the complete
panel. The reproducible 320×200 output is
[`item-panel-native-indexed.png`](../figures/item-panel-native-indexed.png),
generated by `cmd/fd2-item-panel-oracle`; it proves the indexed
resource/layout compositor. The separate runtime bridge below now consumes
that compositor for Ebiten input and 12-frame presentation.

Item-row/runtime bridge: `RenderNativeItemPanelRows` executes `0x184c0` over
the completed panel with compact raw-slot display, category/stat mixed-codec
icons, FDTXT `itemID+181`, selected/unselected foreground 201/205 and exact
stat-number origins. `NativeItemPanelRecordForUnit` materializes the required
80-byte subset only when raw `+6/+8`, `+0x1f/+0x20`, DATO selector and all
eight inventory cells have provenance; normalized HP/MP/AP/DP/DX/HIT/EV/MV,
level and integral EXP are copied only into their independently established
native offsets. The Ebiten adapter uses that record and player archives,
drives opening frames 11→0 and closing 0→11 with the recovered clipped
regions, and maps compact ↑/↓ plus ±4 left/right through
`AdvanceNativeItemSelector`. Missing evidence/assets explicitly retain the
legacy shell. Enter confirmation still refuses type zero and otherwise stops
before the unresolved effect/target transaction. The tracked FDFIELD map
rosters carry `+6/+8/+0x1f/+0x20` through
`sync_native_selector_fields.py`. Direct `0x112a5` disassembly proves JOIN id
selects the lower 24-byte record and writes record bytes0/1 to `+0x1f/+0x20`;
`0x31571..0x3157a` later rewrites only class `+0x20` and selector `+7`.
Scenario generation now projects the 32 cross-checked JOIN rows, persistent
overlay preserves the raw fields/inventory flags, and class change updates raw
class without fabricating a new constructor record. A real ch01 campaign asset
plus player archives passes the Ebiten preparation regression.

The first complete Enter family is now executable for tracked item IDs
198/199/200 (types 8/9/10) and IDs94/95/96 (types 17/18/19). Their rows
independently fix both target stages to mode zero and target code one; the
runtime still validates the actor as the confirmed candidate through
`NativeItemEffectTargets`. The Unit-level
`ApplyNativeItemBaseStatDeltaToUnit` preserves `0x21082` 16-bit wrapping over
raw base AP/DP/DX and `0x1b8e7` compact source removal, then the caller runs
the existing equipment recomputation. `ApplyNativeItemCapacityToUnit` adds
MaxHP/MaxMP without filling current values and applies the low-byte MV wrap
while preserving adjacent EXP, then performs the same compact removal.
Direct caller trace shows successful
`0x20c6f` is followed by `0x13512`; the bridge therefore sets raw `+5 bit7`,
the normalized acted projection, closes the panel and exits the action.
Missing AP/DP equipment-base provenance fails atomically. RNG effects and
non-self target presentations remain outside this completed slice.

`RenderMirrorFigureFadePass` now implements only the proven `0x292ad` indexed primitive: it requires a caller-preseeded 640-stride work surface, presents `work+0x140`, blits primary at `+0x140-stage*10`, conditionally blits secondary for `arg4==0`, and presents the same right viewport again. It validates TAI#3's transparent bytes but does not claim to render the unresolved DATO/portrait or complete montage.

Generic scheduler closure：`funcs_2ac25` 是 command-indexed function bank（ID0 entry `0x26152`）。`0x2a6bd` 先以 mode 0 呼該 entry 取得前段 step count；每一步先從 baseline 複製 320×200 至 640-stride work buffer，呼 mode 1，組合 actor FIGANI，再呼 mode 2，最後才複製至 VGA 並 `0x17aa9(1)` tick。每個 final target 另以 mode 3 取得 target-loop count，逐幀依序呼 mode 4、組合 target FIGANI、呼 mode 5後 present／tick；全目標完成後才是 mode 6 與逐幀 mode 7／8 尾段。`0x2b9a1` 並非未知 effect，它以 descriptor `frameIndex*4+8` 指向 frame的 byte+6 delay，遞增 `0x540fc`／`0x540fd` subframe counters並在上界 reset。這固定了 phase/order，仍不替每個 command entry 的視覺語意命名。

Generic presentation 的 BG selector 亦已閉合為 raw dataflow：`0x2a6bd` 呼 `0x2b5e1(finalCount, finalTargetArray)`，後者**倒序**掃 target slot，對該 unit cell 呼 `0x12e38`；若 raw `0x1f183` gate 不通、或累積 selector 為零，才以 decoded control byte+2 取代 selector，最後才餵 `0x111ba("BG.DAT", selector)`。`fdicon.NativeCommandBackgroundSelector` 保留該 strict pure rule。command ID 的 generic branch 不可被說成直接選 BG resource；selector 的高階地形／場景語意仍不命名。

Command 0 專屬 entry `0x26152` 另由 IDA 9.4 直接指令閉合：
mode 0／1／2 與 6／7／8 皆不產生 handler-owned 幀；mode 3 將七個
有效 counter 以 `0,-2,...,-12` 錯開並回傳 28，mode 4／5 再依
`0x523E1` 旗標、`0x523E8` 水平 offset 與 `0x52404` 垂直 row offset，
將 FDOTHER #18 或 #20 的 raw frames 0..15 異步疊到 640-stride work buffer。
每個元素 counter==3 時播放 FDOTHER #82 sub1。因此正式 runtime 契約是
「先完整解碼所需 frame bank／palette／raw target與view、28次draw acknowledgement，
再發布 MP／HP／acted 交易與清理」；任一前置缺失時維持失敗即關閉。
完整原始位址、bytes 與資源 owner 見
[`fd2_command0_presentation_ida.txt`](../data/ida/fd2_command0_presentation_ida.txt)。

正式 command 0 場景 owner 不能只消費上述28張target效果。`0x2A6BD` 的
caller contract 另要求：以 actor格與 `sub_2B5E1` final-target fold取得 BG／TAI
selector並依actor raw `+6`交換；在同一base畫actor與first-target兩個原生狀態欄；
以actor raw `+7`載`FIGANI base+0` idle及`base+2` effect（header word0時fallback
至`base+1`），再以每個target raw `+7`載idle。actor phase固定先跑
`sub_29164(...,mode0,...)`九張雙角色前導，再由`sub_2B659`於actor-effect raw
`+4==1`發布MP、更新狀態欄、套FDOTHER #3 entry11、播放sound bank sub0與六次
command DAC pulse；target phase才進`sub_26152`的七次sub1／七段HP。

因此 runtime admission 必須先完整預建前導、actor phase、全部target phase、
目標間轉場與尾端。章節初始selector、`0x1F183` gate、FDOTHER #3遮罩／DAC、
`sub_2BA22`或尾端任一尚缺時一律零交易，不得把command24 mode1前導、固定BG或
target-only效果冒充完整場景。完整位址與推論等級仍以同一主證據檔為準。

BG asset boundary：`BG.DAT` 是 LLLLLL archive；generic compositor 的前三個已知 layer #0/#1/#2 都是 `{u16 width,u16 height, 0x4e63d four-mode RLE}` single-frame payload，實測各為 320×100。`fdother.DecodeArchiveSingleFrame` 明確解這種無 frame-directory 的 archive entry，player-archive regression 對三個 layer 解入 320×100 indexed surface。它不替 `0x2b5e1` 的其他 raw selector 命名，也不自動把 current PNG background 當 native layer schedule。

### UI-03 action chooser availability contract（E0 partial）

`0x18d8c` 先清四個 dword，按固定順序傳給 `0x173e7/0x177fc`：`[+0]` attack、`[+4]`
native command、`[+8]` item、`[+12]` wait。`0x173e7` 選第一個值為 0 的方向，`0x177fc`
只允許落在值為 0 的方向；因此這些值是 disabled flag（非可用 count）。具體 E0 precondition 為：

- attack：`0x1b83d(actor,0)` 必須在 runtime inventory 的八個 2-byte slots 找到 flag `0x40`
  且 ID `<0x80` 的 entry；其 item record `+0xb/+0xc` 傳入 `0x14818` 後仍須產生 target，否則 `+0=1`。
  `battle.NativeEquippedInventorySlot` 已保存此 raw predicate；有 constructor flags 時 overlay 不再以
  normalized `Equipped` 取代它，缺 provenance 仍保留 legacy fallback。
- native command：`0x1c269(actor,0)` 必須枚舉至少一個 raw command bit，且 raw `unit+0x27==0`；任一失敗皆寫 `+4=1`。
  command 22 已證實會寫入此 duration byte；其遊戲名稱與所有 producer 尚未閉合，故 remake 只以
  `NativeTransient[5]` 保存並 gate raw command，**不得再說它等於 legacy `Sealed`**。
  選定 command 的 MP availability 另由 `0x159fa` 驗證 record `+5 <= unit+0x44`；`battle.NativeCommandAvailable`
  只在 raw bit、完整 0..35 command record 與 MP gate 同時成立時回 true，未知 36..39 bits 與 malformed book
  fail-closed，不把 selector gate 誤當 action-direction 或 target geometry。
- item：`0x1b8a6(actor)` 計數八 slot 中 flag `0x80` 未設的 entries；零個即 `+8=1`。
  `battle.NativeInventoryAvailableCount` 已保存此 raw count，overlay 在 constructor flags 存在時不再用
  `len(Inventory)` 取代它；沒有八格 provenance 的 legacy JSON 才保留明確標記的 approximation。
- wait：wrapper 未寫 `+12`，故在這條 chooser path 永遠可選。

既有 normalized `Spells`／`Sealed` 只保留給缺 raw command mask 的舊 editable scenario 相容 UI；它不得作為
FD2 native action gate 的證據，也不得覆蓋 raw mask 已存在時的 `unit+0x27` gate。remake 的 confirm path 已同樣
拒絕非零 disabled word，避免「灰色 cell 仍可 Enter 執行」。攻擊 geometry 與 item selector/effect 仍未閉合，native
overlay 維持 partial。

同樣地，`0x1b6b7` 不是 effect calculator：它掃 native runtime roster，只對符合 `+5/+0x31/+0x40` 後處理條件的 record 複製三 bytes（source `+0x31`）到 caller buffer；`0x1cff0` 再把此 buffer 交給 `0x1aa1d`。後者因此是 post-resolution 的訊息／掉落／互動處理層，不能拿來推回 command 0 的原始傷害或 status writer。三個 byte 現已閉合為 `{kind:u8,payload:u16le}`：kind 0/1 是物品／金錢、kind 2 dispatch 全域事件表、kind 3 走呈現分支。建構來源只含 FDFIELD `b22` 與 `b23..b24`；`b25` 不在 runtime `+0x31..+0x33`，舊 24-bit payload 解析已撤回。

玩家 table 的 IDs0..12 numeric damage writer 已閉合到 `0x1c75e(target, commandID)→0x1c81f(target, amount)`：前者取
`record.u16[+0] * resist_raw[unit+0x20] / 10` 為 base；constructor `0x10f7f/0x11399` 直接把 source
class byte 寫入 `unit+0x20`，故這是 target class-ID-indexed table，而非未明角色欄位。這些 handler 先呼
`0x4e893`，以 shared `uint16 state % 100 < record[+2]` 做命中門檻；命中才呼叫後者。`0x1c81f`
再呼一次 `0x4e893`，算 `damage = floor(base*0.9) + floor((state%100)*base/1000)`，
將 target `unit+0x40` 減去 damage，並 clamp 至 0，直接證實 `+0x40=current HP`、`+0x42=max HP`。
IDA `word_51f96` 的 loaded-data file offset 正是既有 `0x51d96` 職業魔抗表：每 class 的 4-byte row
低 byte 是 `resist_raw`（法師=7 即 30% magic resistance）。因此這個乘數的 raw ABI 與玩法名稱都已閉合，
並以 `NativeCommandDamage` 的獨立 resolver 實作及 regression 固定；它不共用 legacy normalized magic
resolver。`remake/assets/data/native_command_resistances.json` 是同一 raw table 的可編輯 runtime copy；target
geometry、動畫及 post-resolution 仍未閉合，故 UI 不得把已知數值公式誤擴張成完整 native effect。

玩家 dispatch 的可達性已重新核對：`0x1cff0` 對 IDs0..8 直接呼叫 `0x2a6bd`，沒有經 table 內的
`0x21227/0x213b7` wrappers；但 `0x2a6bd` 不是純 renderer：它先經 `sub_2b659` 的 MP event，final-target loop
直接以 array slot 和 command ID 呼叫 `0x1c75e(targetSlot, commandID)`。因此 IDs0..8 與 ID9 direct path、及
IDs10..12 compositor tail 都已閉合為同一 numeric/MP/raw-completion contract；dispatch 分流不表示 state effect
缺失，也不表示 renderer 等同。

`State.ExecuteNativeCommandDamage` 嚴格支援 IDs0..12，以 raw record、兩階段 target、class multiplier/hit/HP
clamp 和 success-only raw completion writer 做 bounded engine slice。`ExecuteBoundNativeCommand0`／raw-grid ID0 target slice
只接此 state core；缺 flags、record、candidate 或 resistance row 均在 mutation 前拒絕。專用 renderer、SFX、
post-resolution、其他 ID UI 與 screenshot oracle 仍未完成。

IDs13..16 是另一條已閉合的治療核心，不能併入上面的 damage route。其 jump-table handlers
`0x21AD9/0x21B99/0x2211C/0x22153` 各以 ID `13/14/15/16` 和各自的演出參數跳到共同
`0x21B18`；它在 generic target-confirm 後，以同一 final target array 呼叫專用 indexed 演出
`0x1C4CC/0x1C2DA`、再經 `0x1CA89(actor,id)` 扣 record `+5` MP。它逐 target 呼叫
`0x1C8ED(target,id)→0x1C916(target,record.u16+0)`：`+0x40` 增加
`floor(amount*9/10)+floor(rand()%100*amount/1000)`，上限 clamp 為 `+0x42`，並以
`0x1E0DB(...,0x69,target)` 顯示結果。這直接證實 IDs13..16 是 per-final-target HP restore（ID13 raw row 為
`dmg=70, +3=4, +4=0, mp=3, target=1`），但尚未把這個獨立 resolver、專用 renderer、SFX 或 UI 接入 remake；
在有對應 regression 前仍 fail-closed。

ID24 必須和上段嚴格分開。`funcs_1541f[24]` 雖然在 AI／自動執行的 `0x15311` 分派表中別名到
`0x22153`，使該表項把 **ID16** 傳入共同治療尾端；但玩家的 `0x1cff0` 明確將 `0x18` 直入
`0x2a6bd`，後者又以精確 ID24 分支至 `0x276ec`，完全不經 `funcs_1541f`。所以 table alias 不能當成
玩家 ID24 的效果或 MP ABI。玩家 `0x276ec` 的 state dataflow 已知：它選固定倍率 `15`，算
`trunc(actor.+0x48 * 15 / 10)`，逐 final target 扣 target `+0x4a` 後送入
`0x1c81f(target, amount)`；該共用 writer 再以其既有 90–99.9% RNG 路徑扣 `+0x40`、clamp 至零。
`0x276ec` 先經 `0x2b659`；該 event 對 ID24 在 actor effect frame raw `+4==1`
時以 `0x1ca89(actor,0x18)` 扣 record24 `+5` MP。target loop 會先暫存 total
delta並恢復 HP，但 `0x27c6d..0x27c89` 只有 ID28 使用分母8；ID24分母是1，
所以第一個 target effect raw `+4==1` 就發布完整 final delta，並非等份多段扣血。
default 轉職 selector32 的 resource98 提供一次 actor marker與一次 target marker，
header byte4=6 又經 `0x2bc9a` 選 FDOTHER #53 的 samples3／2。完整證據見
[`fd2_command24_presentation_ida.txt`](../data/ida/fd2_command24_presentation_ida.txt)。
AI alias 的設計意圖、完整 `0x29c90` 逐像素合成、精確 PCM 取樣率與 E2仍未關閉，
故不可冒充 ID16 heal 或借接 generic numeric／palette executor。

IDs17..19 是第三條 transient-modifier family，亦不能交給 damage/heal executor。ID17
`0x226EA→0x22721`、ID18 `0x2282F→0x22866`、ID19 `0x22960→0x22997` 都在 final target loop 中先拒絕
已設 flag 的 unit：17/18 在 `+0x22/+0x23` 為零時設 `rand()%4+2`，並分別對 `+0x48/+0x4a`
加 `__CHP(value*0.15+1)` 的 toward-zero increment；19 對 `+0x24` 同樣設 duration，並對 `+0x4c/+0x4e` 各加 15。
這與 `0x1b750` 對 `+0x48/+0x4a/+0x4c/+0x4e` 的 derived AP/DP/HIT/EV synthesis 相容，因而撤回先前把
這些 offsets 稱為 screen coordinates 的斷言。duration 的 tick/clear、玩家可見 status 名稱、專用演出與
remake state/UI 仍未閉合，不能據此補出 gameplay names。

這一族的 MP transaction 有一個不可泛化的細節：jump-table ID17 的 `0x226EA` 與 ID18 的
`0x2282F` 都直接呼叫 `0x1CA89(actor,0x12)`，而 raw records 17、18 的七個 bytes 都是
`00 00 00 04 02 05 01`。因此目前只可證實兩者在這個版本有相同 MP debit；不得從 wrapper index
推導「所有 handler 都把自身 command ID 傳給 `0x1CA89`」。ID19 則明確傳 `0x13`。這不改變其
modifier writer／duration 證據，但阻止錯誤泛化 command transaction ABI。

正式重製交易據此採原子發布契約：ID17 使用 record17 的 selector／target 欄位，但 MP debit
明確取 record18；ID18／19 分別取 record18／19。所有 final targets 必須先能投影成私有
`0x50`-byte runtime record，並在私有 buffer 完成 `ApplyNativeCommandModifier`；只有整批成功後，
才一併發布 `+0x22..+0x24`、`+0x48..+0x4e`、actor MP 與 acted。缺 record、target、raw target
provenance、16-bit derived word 或 MP 時均在 mutation 前失敗。此契約只關閉 state transaction
與敵方 mode 11 消費端；因專用 indexed renderer、status label、SFX 與玩家格狀確認仍無證據，
玩家 UI 不得據此開放 IDs17..19。

2026-08-22 的窄 IDA 補證已解除上句的 renderer gate，取代「沒有 renderer 證據」的舊狀態：
玩家 `0x1CFF0` 在 IDs17..19 的 target confirm 後先呼叫 `0x1D6C8(commandID)`；該函式播放
`0x25A96([0x53B13],0,1)`，亦即已證實的 `FDOTHER #88` sub0，然後固定四輪將 DAC palette
entry0 寫為三張36-byte tables 的 command RGB、等待 `0x17AA9(1)`、寫黑、再等待
`0x17AA9(1)`。三個 ID 的 raw six-bit RGB 均為 `(0x32,0x32,0x32)`。完整 table 已成為
`native_command_palette_flash.json`，正式玩家 owner 必須在 sample、baseline framebuffer、
黑色entry0、完整DAC、typed table與全部raw transaction provenance都通過後，才發布八個
Draw-ack phases；第八 phase 完成後才執行上述原子交易與range cleanup。AI `0x15311`直接進
effect table，不套用玩家 `0x1D6C8`。status名稱／icon、DOS精確tick及同狀態逐音訊E2仍未知。

同一玩家 `0x1CFF0→0x1D6C8` dispatch 也涵蓋 command 20–22；它們不可因 effect
family 不同就略過共同 sample／palette owner。正式入口在演出前必須完整驗證 command
record、final targets、MP、RNG、target HP bounds、raw interval與所有呈現資產；驗證失敗時
不得播放 sample、發布 palette、扣 MP 或改 target。八個 Draw-ack phases 完成後，20／21
才走既有 `+0x25/+0x26` clear／record10 restore，22 才走 `+0x27` application；完成後共用
range／selection cleanup。三個 status 名稱與 expiry feedback 尚未知，玩家訊息只能描述 raw
offset／交易結果，不得冒稱原版狀態名稱。AI `0x15311` 仍直接進 effect table，不套用此玩家演出。

同一 dispatch 的第二段 `0x19..0x1B` 對應 command 25–27，也必須走相同
sample／palette owner。ID25 的已證實 writer 是 raw `unit+5 bit7` clear；這個 bit
不可再投影成 Go `Unit.Acted`。正式 preflight 必須要求 target 的
`NativeRecordByte5` provenance，交易只清 `0x80` 且不改 target `Acted`。IDs26／27
分別沿既有 application route 寫 `+0x25/+0x26`；三者均在八個 Draw-ack phases 後
才扣 MP／發布 target mutation。command23仍有獨立 relocation renderer生命週期，不能只因共用
palette owner就視為整條玩家演出已接。

IDs20..21 共享另一條「flag-present 才生效」route：`0x22A85/0x22BC6→0x22AA8→0x22AF6` 各以
command ID 20/21 扣 MP，對每個 final target 讀 `+0x25/+0x26`。該 byte 為零時只走失敗 display；非零時呼叫
`0x1C916(target,10)` 的既有 HP-restore writer、清零該 byte，並顯示結果。這證實 raw gate、clear 與
HP writer，但尚未命名兩個 status，亦未接 engine/UI。ID22 是不同的 `0x22BE1→0x22CDA→0x22D1B` route：final
target 的 `+0x27` 必為零、class `+0x20` 不得為 `0x19/0x1a`、且 `rand()%100<0x32`，才以
`0x1C81F(target,10)` 固定扣 10 HP、顯示 damage，並寫 `rand()%4+2` 至 `+0x27`。它須獨立追蹤，不能併稱為 cure
或依 raw offsets 猜測 status name。

這六個 transient bytes 的 decrement 已由 official IDA 釘死，但 gate 仍是 raw ABI：已重跑的 caller `0x1A4D1`、`0x1A55E`、`0x1A797` 分別傳入 selector 1/0/2；`0x1A866` 只接受 `record+6 == selector` 且 `(record+5 & 1)==0` 的 record；不可把它改寫成 `Camp/OnField/Alive` normalized 條件。通過 gate 後依序對 `unit+0x22..+0x27` 的每個非零 byte decrement。任何一個 byte 變零時才顯示 expiry feedback 並呼叫 `0x1B750(unit)` 重算 derived fields；因此 ID17/18 的 AP/DP 增幅會在自己的 duration 歸零後由重算移除，其他 flag 不可因為共用 sweep 就被誤認為同一 status。這是 phase-based timer ABI，不是每次 action 或 frame 的 timer。`sub_17FC0` 另已證實 `+0x22..+0x24` 以數字 color base `0x77` 表示，`+0x25..+0x27` 才各畫 FDOTHER #5 entries `0x37..0x39`；高階 status labels 仍未命名。

同一場景流程中的 `0x1A7BD`/`0x1A7F1` 不是 transient selector 語意本身：前者在 `[0x53AF9] != 0` 時以 `0x111BA(0x1A4D,0,0x40)` 建立 resource handle 並寫 `[0x53B0F]`，後者釋放該 handle。`0x1A4EB` 與 `0x1A58F` 都採「setup → unit scan → release」順序；因此 selector→campaign phase 仍不可由這兩個 resource helper 推導。

Remake 已以 `Unit.NativeTransient[6]` 及 optional `NativeRecordByte5/6` 保留這段 raw ABI，並提供 bounded offset access（只接受 `0x22..0x27`）及 `State.TickNativeTransientsRaw(selector)`；FDFIELD b0→runtime `+6` 的 parser/exporter provenance 也已補上，缺少 raw gates 時仍 fail-closed。它刻意不呼叫 normalized `TickStatus` 或 legacy shared `BuffTurns`。selector 1→0／2 的正式 owner、`0x1B750` equipment recompute、FDTXT 481..486 到期提示，以及 `sub_17FC0` 的顏色／圖示 status panel consumer 均已接為 `RUNTIME-E1`；剩餘限制是高階名稱、精確 tick／音訊與一般玩家 E2。

Selector caller audit（Docker Capstone）已補上 raw 值但不替它們命名：`0x1a4d1` 以 `push 1` 呼叫
`0x1a866`，`0x1a55e` 以 `push 0` 呼叫，`0x1a797` 以 `push 2` 呼叫；三者各自位於不同
redraw/phase caller，不能直接映射成 Go `Camp` 或玩家／敵方回合。`0x1a30b` 內部另有
`record+6 == 2` 的 sweep，與上述 direct callers 分開。故 runtime 仍只提供 raw selector API，
不把 `completeTurn` 或 normalized camp 自動綁到任一 selector。

ID23 走 `0x1CFF0` 的 command-`0x17` special selector，不能套 generic two-stage target contract。其 handler
`0x2218A` 以 record23 扣 MP，並呼叫 `0x22253` 兩次：依 C stack ABI，第一次將 selected unit 的 runtime
`+0/+1` 寫為 `0xff/0xff`（以原座標作離場 indexed 演出），第二次直接寫為 selector cursor globals
`0x51CF9/0x51CFD`（並作入場演出）。因此已證實它是無 path traversal 的直接 grid-coordinate relocation；
落點 selection/legality、camera choreography、renderer 與 remake UI 尚未閉合，不能把它泛化成普通 move 或 generic
target effect。

重製端 command23 正式生命週期（2026-08-22）必須保持這條可見且原子的順序：目的地確認後，先在
私有 raw records 驗證完整 relocation transaction，再執行共用 `0x1D6C8` 的 #88 sub0 與八個
Draw-ack 色盤階段；其完成回呼啟動第一次 `0x22253(target,0xff,0xff,currentX,currentY)`，離場完成後
再啟動第二次 `0x22253(target,destX,destY,destX,destY)`。只有第二段完整結束後，才發布 record23 MP
debit、raw action bit、item 保留及 action cleanup。destination cursor、terrain／occupancy gate 仍由既有 mode-6
owner 負責。缺 command book、raw records、FDOTHER #88、DAC、完整 indexed map bundle 或任一 `0x22253`
前置條件時，必須在第一個 sample／frame 前失敗；演出期間不得接受其他玩家輸入。呈現器若在中途發生
不可預期錯誤，必須把 target coordinates 與 indexed work/VGA 回復到目的地確認前快照，不可留下
`0xff/0xff` 的半完成狀態。這項 E1 契約只還原已證實的 palette→disappear→appear→transaction ordering；
尚不宣稱 camera choreography 或逐幀／逐音訊 E2。

IDs25..27 也已由 jump-table 閉合。ID25 `0x22C04` 以 record25 扣 MP，僅對 final target 已有
`record+5` bit `0x80` 已設的項目清該 bit，直接保留 raw clear writer（不命名 acted/action-complete）。ID26
`0x22CBF` 與 ID27 `0x22E41` 分別將 command ID 和 flag offset `+0x25/+0x26` 傳給與 ID22 同一
`0x22CDA→0x22D1B` application helper，所以同樣受 zero flag、class、`rand()%100<50` gate，成功固定扣 10 HP
並寫 2..5 duration。這使 ID20→`+0x25` clear 與 ID26→`+0x25` apply、ID21→`+0x26` clear 與 ID27→`+0x26`
apply 成為 direct code-pairs；UI/status indicator 的獨立驗證現由
[`fd2_status_panel_transient_indicators_ida.txt`](../data/ida/fd2_status_panel_transient_indicators_ida.txt)
固定，不再借用 command writer 推導。

`State.ExecuteNativeCommand25` 現是另一個 non-UI, fail-closed engine slice：它只接受完整 raw book/flags 的
generic two-stage target contract，完成 record25 MP debit 後，對 final targets 的 raw `+5` bit `0x80` 作精確 clear-if-set，
最後才套用 actor raw bit writer。`Unit.Acted` 只是目前 engine projection，不是 native semantic。target invalid、缺 flags 或 MP 不足都在 mutation 前拒絕；它不使用 normalized CastArea，
也未開 native grid/UI、renderer 或 message feedback。

`State.ExecuteNativeCommandApplication` 現對 IDs22/26/27 提供另一條 strict non-UI core：以各自 record 建 generic
two-stage final targets、扣各自 MP；每個 target 只在 raw `+0x27/+0x25/+0x26` 為零、class 不為 `0x19/0x1a`、
`rand()%100<50` 時固定扣 10 HP，並寫 `rand()%4+2` 到同一 raw byte。已有 duration/class gate 的 target 不會
mutation，但 handler 已成功時仍遵循原版 MP debit/actor raw completion writer；unknown ID、缺 raw data 或 invalid target 在
mutation 前拒絕。此 route 不映射 legacy Poisoned/Paralyzed fields，UI/renderer 仍 fail-closed。

`State.ExecuteNativeCommandClearRestore` 對 IDs20/21 亦已接 strict non-UI core：各自 record 只供 target/MP；
final target 的 raw `+0x25/+0x26` 非零時，才以 **record10** 的 raw damage 呼 `ApplyNativeCommandRestore`，再清同一
raw byte。restore 精確算 `amount*9/10 + rand()%100*amount/1000`、HP cap，並分開報告 rolled value 與實際
HP delta，避免把原版 display number 誤當 mutation。empty flag 不 restore，但 successful handler 仍 debit MP/complete
actor；不映射 legacy named status/UI。

`State.ExecuteNativeCommandHeal` 現對 IDs13..16 接 strict non-UI core：每個 ID 只使用自己的 raw record
完成 generic two-stage targets、MP debit、並以同 record `u16 damage` 走 `0x1C916` restore/cap；成功後才執行
actor raw completion writer。它與 ID20/21「借 record10」的 clear/restore route 明確分開，並不因共用 restore primitive 而推論
`0x1C4CC/0x1C2DA` 專用演出、SFX、message 或 UI 已完成。

### UI-03 native command family implementation matrix（E0/E1 status）

下表是 raw command table ID，不是 legacy `Spell.ID` 的別名。`engine` 只表示 strict non-UI state core；
沒有 UI/renderer evidence 的列不得由 command grid 開放。這個 matrix 是每次擴充 effect 時的 fail-closed gate。

| IDs | 原版已驗 dataflow | engine 狀態 | UI / renderer 狀態 |
|---|---|---|---|
| 0–8 | `0x2A6BD→2B659/1C75E`，two-stage final targets、MP event、numeric hit/HP | `ExecuteNativeCommandDamage`；ID0 有 target slice | 僅 ID0 grid target；compositor/SFX/post-resolution 未接 |
| 9–12 | direct/`0x21548` tail → `1CA89→1C75E` | `ExecuteNativeCommandDamage` | 未接；numeric 共用不代表演出共用 |
| 13–16 | `0x21AD9…0x22153→21EB1→21B18→1C8ED/1C916` | `BuildNativeCommandHealPresentationSchedule`＋玩家 `ExecuteNativeCommandHeal`／AI `ExecuteNativeAICommandHeal` | 玩家與敵方 mode 11 的 16 張 FDOTHER #3 LUT 前段已接；AI 依 raw selector 重建 target array；後段索引畫面／數字佇列與格狀確認 E2 未接 |
| 17–19 | `0x1CFF0→1D6C8→226EA/2282F/22960`；#88 sub0、八個DAC phases、modifier writers與`+0x22..+0x24` duration已釘死 | `ExecuteNativeCommandModifier`／`ExecuteNativeAICommandModifier` 已以私有raw records原子發布target duration、derived words、MP與acted；ID17明確由record18扣MP。玩家與敵方mode 11、phase-expiry caller與 `sub_17FC0` status color consumer均已接 | 玩家grid＋八phase palette／sample、倒數、到期提示及status color已接E1；高階名稱、精確tick／逐音訊E2未接 |
| 20–21 | `0x1CFF0→1D6C8→22A85/22BC6→22AF6`，#88 sub0＋八個DAC phases，clear `+0x25/+0x26` 並借 record10 restore | `ExecuteNativeCommandClearRestore`；完整preflight後才允許玩家演出與交易 | 玩家grid＋八phase palette／sample已接E1；status名稱、expiry UI、精確tick／逐音訊E2未接 |
| 22 | `0x1CFF0→1D6C8→22BE1→22D1B`，#88 sub0＋八個DAC phases，class/RNG gate、base10 經第二 RNG 實際9 HP、第三 RNG write `+0x27` | `ExecuteNativeCommandApplication`；完整preflight後才允許玩家演出與交易 | 玩家grid＋八phase palette／sample已接E1；status名稱、expiry UI、精確tick／逐音訊E2未接 |
| 23 | `0x1CFF0→1D6C8→2218A→22253×2` special relocation selector | first target→mode-6 destination cursor、八段palette、離場／入場兩次完整indexed presenter及延後raw transaction已接 | 玩家正式`RUNTIME-E1`；缺同狀態逐幀／逐音訊E2與精確camera choreography核對 |
| 24 | 玩家 `2A6BD→276EC→2B659/1CA89→1C81F`：`actor +48 * 15/10 - target +4a`；selector32 resource98、FDOTHER #53 samples3／2、damage denominator1；AI table 另別名 `22153`，不可混用 | `ExecuteNativeCommand24`＋正式marker交易 | indexed actor／target base、前導、轉場與SFX已達E1；精確時序／音訊與E2待接 |
| 28, 29, 31 | 同玩家 `276EC` derived-strike route，倍率分別20、12、18；28的target writer分母8且略過`29C90`，29／31分母1並保留逐target轉場 | `ExecuteNativeCommandDerivedStrike`；28已修正為單一可達impact marker發布roll的1/8，29／31補有獨立倍率回歸；`BuildNativeCommandDerivedStrikeSchedule`保存command-specific header／marker／audio／base／prelude／transition契約 | typed schedule已就緒；正式indexed owner／SFX播放與E2未接，不得借用command24 resource98 |

指令 29 的玩家演出採獨立、原子化的多目標契約。只有 actor 的原始
`BattleFig=34`、FIGANI resource104、FDOTHER #50 sample1／4、完整 raw command
record 與所有 final targets 的 indexed background／idle 資產都可用時，正式 owner
才可啟動。actor phase 只播放一次，並在 actor marker 只扣一次 MP；之後每個 final
target 都各自播放 20 張 `0x29C90` 轉場、從 idle frame0 重設自己的游標，並在該
target marker 發布對應 HP。所有 target 完成後才設定 `Acted` 並發放死亡獎勵。
任一目標的資產、marker 或狀態發布失敗時，必須回復 actor 的 MP／`Acted` 及所有
已發布 target 的 HP／raw `+5`，不得保留部分提交。這項契約只接玩家指令 29；敵方
caller 的演出 owner、指令 28 的不同 prelude／無轉場路徑，以及指令 31 的 actor
selector 尚未證實，仍維持失敗即關閉。

2026-08-23 的正常取得路徑窄稽核進一步固定：runtime command-mask OR writer
`0x1D79C`只有`0x1E292`升級流程的單一direct caller；該流程依raw `+7`解析growth
row `+10`的`learn_idx`，再掃六組level／command pair。固定command-learn表與32筆
player constructor defaults都不含ID28／31。因此目前只能把兩者列為「沒有已證實
的一般玩家取得來源」的強推論，不宣稱死碼；它們不阻擋remake交付，也不准用
selector18排除猜測補演出。若未修改玩家動態路徑出現兩者、發現另一個mask writer，
或取得同一actor的raw `+7`與command bit，才重開正式owner。完整證據見
[`fd2_command28_31_reachability_ida.txt`](../data/ida/fd2_command28_31_reachability_ida.txt)。
| 30 | `1CFF0→14818→115B6` 先確認 record+3 candidate；再以 saved cursor→confirmed cursor 進 `149F8`，`count=record+3-16`、X-first cardinal line、只收 enemy，最後 `2A6BD→276EC` default倍率18 | `ExecuteNativeCommand30`（顯式兩 cursor、state-only final delta） | native cursor lifecycle／multi-hit／SFX／indexed UI 未接 |
| 25 | `0x1CFF0→1D6C8→22C04`，#88 sub0＋八個DAC phases後清raw target `+5 bit7` | `ExecuteNativeCommand25`要求raw +5 provenance，只清`0x80`而不混用`Acted` | 玩家grid＋palette／sample已接E1；原版feedback／逐音訊E2未接 |
| 26–27 | `0x1CFF0→1D6C8→22CBF/22E41→22D1B`，分別 write `+0x25/+0x26` | `ExecuteNativeCommandApplication`；完整preflight後才交易 | 玩家grid＋palette／sample已接E1；status名稱、expiry UI與逐音訊E2未接 |
| 32 | `2A6BD→27FC9→2111A→1C75E` numeric per-final-target；選單 MP gate已知但此 chain 未見 debit | `NativeCompoundCommandPlan(32)` 僅保存 raw callee 順序 | 未接 |
| 33 | `27FC9` 先清每 target `+25..+27`，再 `211A4(...,800)` restore | `NativeCompoundCommandPlan(33)` 僅保存 direct-clear 順序與 raw amount | 未接 |
| 34 | `27FC9` 依序呼 `22721/22866/22997`，嘗試三種 modifier writer | `NativeCompoundCommandPlan(34)` 僅保存三個 raw writer 順序 | 未接 |
| 35 | `27FC9` 依序以 IDs26/22/27 呼 `22D1B`，對 `+25/+27/+26` 三 application gates | `NativeCompoundCommandPlan(35)` 僅保存 marker offsets/呼叫順序 | 未接 |

實作和測試必須以本表逐 ID 更新。不得因 record bytes、label 或 generic dispatch 可見，就把未知 ID 送進
legacy `CastArea` 或宣稱整個 native command menu 已完成。

AI boundary correction：在 `0x1598a` 的 command-score path，`0x14818` 目前只可稱為 target-candidate
builder；候選建立後才進 `0x15B77` 的 family-specific score branches。任何文件或 adapter 都不得把
target builder 直接命名成傷害／命中評分，也不得把 `unit+0x22..+0x27`
的 raw bytes 直接命名成 AP/DP/HIT/status。

Runtime bridge 勘誤：`0x1C269` 將 `unit+0x1A..+0x1E` 的 set-bit
索引原樣交給 `0x1598A`；後者直接以同一索引呼叫 `0x4E516`、`0x15B77`，
選中後 `0x15311` 也直接消費 `[0x53C2F]`。這條路徑沒有
`command-0x10`；該轉換只存在於 `0x1567E` item-command 路徑。
`battle.AIPlan.NativeScoredCommands` 因此保存通過 raw `+0x27`、command
mask、36-record 與 MP gates 的已知原始索引 `0..35`，不再錯誤排除
`0..15`。它仍不填 `SpellID`、不選 target、不評分，也不執行 effect；
缺少完整 `NativeCommandBook` 時回傳 nil。直接指令見
[`fd2_ai_command_index_disasm.txt`](../data/fd2_ai_command_index_disasm.txt)。

`battle.NativeAIScoringRecords` 是下一層 fail-closed input boundary：
以既有原始物品面板 record 為底，補入 native map presentation
`+0/+1/+3/+4`、activity `+5` 與 AI mode bytes `+34..+36`；完整 roster
任一筆缺 provenance 就不回傳部分結果。map0 真實匯出 roster regression
已固定第一筆 `(1,3)`、enemy code 0、mode byte 0 與 HP 28。這仍是 E0
快照，未授權 normalized planner 使用，也不能取代原版動態候選／畫面 trace。

其後的 `battle.NativeAIScoredCommandCandidateGroups` 已保存
`0x1598A` 的 exact geometry contract：command `+3` 是 destination
budget，經原版 movement-cost row 與 caller-owned grid flags 產生
row-major cells；每格再以 command `+4` 建 target field，套 selector
導向的 raw target code 與 record `+5/+6` predicates，並保持 roster
index order。沒有 targets 的 cell 不進 score。map0 assets 已以
identity 103 actor、command #0、row 0 與 ally `(23,14)` anchor 通過。
這一層仍不做 `0x15B77` score／tie-break，也不接 production AI。

`battle.BuildNativeAIPhaseDiagnosticPlan` 現將這段已證實、無副作用的資料流
接到三遍掃描規則：每筆合格的 raw `+6==0` 記錄必須由呼叫者明確提供移動
成本列與可選的 `0x1F183` 閘門，並依 `0x1D8BA` 原序先執行
`ScoreNativeAI1598A(unit,0)`、再執行 `ScoreNativeAI1567E(unit,0)`；兩個
最大分數分別轉成 signed dword 後，才交給
`fdother.PlanNativePhaseUnitScans` 的 `>=6` 門檻。缺少、重複或不合格單位
的輸入均失敗即關閉。回傳值只包含診斷結果與三遍掃描計畫，不會呼叫
`0x13A9F`、兩張回呼表或處理 `[0x53ECC]` 的逐單位提早離開，因此不得接入
正式 `NextAIPlan` 或宣稱敵方回合已重現。

map0 交叉夾具使用真實名冊、目標旗標、地形、命令表、物品列及移動成本列，
但會排除其他 selector-zero 單位，並替 index23 注入 command0 與已追蹤
item79。它固定得到命令遮罩分數96、物品命令分數8及門檻通過，屬修改狀態的
E0 組合驗證，不是一般玩家 map0 動態狀態。

`sync_native_map_renderer_inputs.py` 現從每張 FDFIELD 構成格 `+2` 同步
`native_composition_event_bytes`，與封存 `+3`、FDSHAP
`native_terrain_control` 一起受到原始封存檔雜湊及 33 圖尺寸檢查。先前把
這個 `+2` 陣列命名成完整 target flags 是錯誤斷言：`0x4DBFC` 載入後先
執行 `+2 &= 0x1F`，`0x145CD→0x14625/0x146A7` 才依 caller selector 與
執行期 roster 加上 `0x40/0x80`。地圖 JSON 只保存不可變來源；
重製 `State` 不再保存跨命令的執行期目標旗標（live target flags）。map19 真實資產現提供1600格
event bytes（7格非零）；unit55 的 identity92、遮罩
`[4,0,0,8,0]`、MP288 可直接通過兩個單位評分器，結果皆為零且不創造勝者。
這是未修改分數輸入的 E0 負向錨點，仍不是完整 phase callback 或玩家路徑。

IDs32..35 的 `0x27fc9` 是一個獨立 multi-effect presentation wrapper，不能因為各 helper 已在其他 command
family 出現就直接重用既有 executor。direct static trace 已見：32 進 `0x2111a→0x1c75e`；33 對每個 final target
`memset(+0x25, 0, 3)` 後傳固定 `0x320` 給 `0x211a4→0x1c916`；34 連續呼
`0x22721/0x22866/0x22997`；35 連續呼 `0x22d1b(actor,26,...,+0x25)`、
`0x22d1b(actor,22,...,+0x27)`、`0x22d1b(actor,27,...,+0x26)`。先前「wrapper／helper 未見
`0x1ca89`」的負向斷言已撤回：`0x27fc9` 在 `0x28189` 進共同 presentation/effect routine `0x2b659`，而
`0x2b738..0x2b753` 會在其**載入 FIGANI container header 的 `byte+4 == 1`**時呼
`0x1ca89(actor, commandID)`。這個 resource 選擇已對 player class-19 path 關閉：command-learn row 19 提供
IDs32..35；原始可達來源是 portrait/visual group 4..7 的 optional class-change 與 group20 的初始 class19，
故 `group*3+1` 分別取 FIGANI #13/#16/#19/#22/#61。原 archive header byte4 依序為 `2/2/2/5/5`，全不等於 1；
而 `0x27fc9` 唯一 caller 是 `0x2a6bd`，`0x2b659` 是這條 presentation path 中唯一的 `0x1ca89` call site。
因此可證實這些**玩家可達 class-19 路徑不經已知 MP debit sink**，即使 record `+5` 的 selector gate 仍要求
76/52/28/36 MP。這不是「所有 runtime entity 免費」或 transaction rollback 的結論；AI／未盤點 runtime unit
visual group、其他 MP writer 與其餘 compound transaction 仍未閉合；下段只對已證實
class19玩家來源開放ID34，其他路徑保持 fail-closed。

The wrapper's only direct caller is `0x2a7ce`, entered from `0x2a6bd` when
the opaque command selector is `>=0x20`; it passes four caller-owned values
without a proven normalized type. Inside `0x27fc9`, resource setup and the
`0x29164`/`0x2b659` presentation chain precede the ID-specific raw operations,
then indexed redraw/present loops and resource cleanup run for all four IDs.
`battle.NativeCompoundCommandPlan` exposes only this verified order and raw
marker/amount bytes as editable data. `Callee==0` denotes ID33's three direct
byte clears, not a guessed helper. The plan does not execute, debit MP, choose
targets, or infer effect/status names.

指令34先形成一條受限的玩家狀態交易：只接受具完整raw class／selector provenance、
`NativeRecordClass==19`且`BattleFig`為已證實可達group 4／5／6／7／20的actor；record34
只作MP selector gate，不扣MP，因這五條固定資源路徑已證實全部繞過唯一已知
`0x1CA89` sink。正式交易以record34的兩階段final targets建立私有`0x50` records，
依`0x22721→0x22866→0x22997`順序連續套用17／18／19三個raw modifier，三段全部
成功後才一次發布`+0x22..+0x24`與`+0x48..+0x4E`、保留MP並設定actor `Acted`。
缺class／selector／target raw provenance、MP gate、RNG或任一stage時零修改。這只
關閉ID34 state transaction；`0x27FC9` indexed presentation、ID32／33／35交易、
AI／其他visual group與一般玩家E2仍失敗即關閉。

command 0 的 selector boundary 也已縮小：`0x1cff0` 對一般 record（非 command `0x17`／`0x1e` special
branch）先以 actor cell、`record[+3]`、`record[+6]` 呼叫 `0x14818`，把可選中心的 unit indices 寫進 caller
stack array；`0x115b6(mode=record[+6], count, array)` 作 cursor/confirm。confirm 成功後，它以**確認游標格**、
`record[+4]` 和同一 target code 再呼叫 `0x14818`，此第二個 candidate array/count 才傳入
`0x2a6bd(unit, commandID, count, array)`。這證實 command 0 的 selector 是 **per final-effect candidate**，
而非 legacy UI 的單格 `CastArea` contract。final array 本身不能單獨證明 numeric resolver；該部分已由後續
獨立的 `0x2a6bd→0x2b659/0x1c75e` direct dataflow closure建立。`0x14818` 的方向／形狀
與 target-code semantics 已有 raw closure：`dist<0x10` 經 native map/reach mask 決定可見格；`dist>=0x10`
使用十字線，半徑=`dist-0x10`（同 x 或同 y）。掃候選時必須在 grid geometry 內，並以 raw byte+5 active gate 及 target code 對 runtime
`unit+6` 做精確 predicate：`0: ==0`、`1: !=0`、`2: ==1`、`3: ==2`。constructor `0x10c50` 證實 `unit+6`
直接來自 FDFIELD `b0` camp（敵=0、友=1、己=2），故四個 code 分別是 enemy/non-enemy/ally/own；
`dist<0x10` 的 mask 已閉合為 `0x4e040→0x4e0dc/0x4e16e` 四方向
flood-fill：入口設定起點 budget=`dist`，遞迴器展開四鄰格，`0x4e16e`
對 grid flag bit `0x40` 阻擋；bit `0x80` 在正常扣除地形成本後把該格剩餘 budget 強制成零，使其成為可達終點，
不是零成本路徑。雖然 callee 支援 terrain-cost row，command selector 固定呼叫 `0x4e555(0)`，而
EXE `word_61646` row 0 的 20 bytes 全為 `1`；因此這條 native command contract 不套地形加權，而是避障的
cardinal range（無阻擋時才等於 Manhattan）。

`0x115b6` cursor-confirm gate is a separate contract from the `0x14818`
candidate list. Docker Capstone proves `0x14742` has exactly one caller,
`0x1175f`. For Enter/Space, raw target code 5 rejects before inspecting the
cell; FDFIELD selected-cell byte+3 equal to `0xff` also rejects before code4,
while code4 on a non-`0xff` cell accepts. Codes0..3 on a non-`0xff` cell call
`0x14742(cursorX,cursorY,radius,0,targetCode)`, where radius is `[0x51a83]`
decremented only when above one. The helper scans active raw records and
counts matching camps at strict Manhattan distance `< radius`; confirmation
accepts only when that count is nonzero. Code6 uses the already separate
relocation branch. `NativeCursorConfirmationAllowed` preserves these branch
orders and requires a complete raw roster. This direct reread also corrects
the old code2 predicate from `camp != 1` to `camp == 1`; tests assert target
identity rather than only list length.

`battle.NativeCommandTargetCells`／`NativeCommandTargets` 已把**一次** verified `0x14818` 呼叫做成獨立資料層：
caller 必須提供精確原版 grid flags，缺失或長度不符即 fail-closed；不重用現有 `map.json.cost`，並明確選定 first
selection stage (`actor,+3`) 或 confirmed effect stage (`cursor,+4`)。它覆蓋 four-way flood-fill、bit40/bit80、
cross branch 與四個 camp predicates。`NativeCommandEffectTargets` 進一步要求 confirmed unit 確在 first candidate
list，才以其 cell 與 `+4` 取 effect list，固定 generic two-stage contract；command cursor 的
highlight/confirm 已共用 selected raw command record，不再硬編 command 0 的 geometry；effect
renderer／legacy cast replacement 仍未開放。

候選 unit 的 active gate 也已按 raw provenance 收斂：當 roster 每筆都有 FDFIELD-derived byte+5 時，
`NativeCommandTargets`、`NativeAttackCandidates`、`NativeCommandEffectTargets` 與 command-30 cardinal
resolver 只採用 `byte+5 bit0 == 0`；不再以 HP 或 `OnField` 另造 alive predicate。舊 hand-built JSON／測試資料缺少
完整 raw roster 時才保留 normalized projection，明確標為 E1 compatibility boundary，不能宣稱 native state 已完全
materialize。

`battle.NativeAttackCandidates` 另保存 `0x14237` 的 caller-specific geometry：它先以 item-row
傳入的 raw `(a4=mode,a5=innerRadius)` 執行同一 `0x14818` grid pass，僅在 `mode<0x10` 時排除
Manhattan distance `< innerRadius` 的 marker cell；`mode>=0x10` 保留 native cross branch，不套
inner-radius。這個 adapter 不把 `+0x0b/+0x0c` 命名成 range min/max，也不宣稱已完成 item-row producer、LOS、UI
或 attack effect。

Provenance correction：`0x1088D` 以 `chapter*3` 呼叫 `0x111BA`，後者釋放
舊 `[0x53A51]`、依 FDFIELD.DAT offset table 配置精確資源大小並讀入可變
緩衝區。`0x4DBFC` 隨即逐格執行 tile high byte `&=3`、cell `+2 &=0x1F`
及 `+3=0xFF`。`0x4E040` 建立搜尋狀態、`0x4E0DC` 遞迴鄰格，而
`0x4E16E` 才是把 `+3` 當 remaining budget 並讀 live `+2` 的
`0x40/0x80` 消費端；這兩個高位元不是封存資料，而由
`0x145CD→0x14625/0x146A7` 依 roster 暫時寫入。`0x40` 阻擋進入，
`0x80` 在扣成本後把目的格 budget 強制歸零。

因此 `battle.Load` 只把完整 `native_composition_event_bytes` 載入
`State.NativeCompositionEventBytes`。`NativeCompositionBaseFlags` 保存
`0x4DBFC` 的 `&0x1F`，`NativeCommandRuntimeFlags` 保存已驗證的 roster
高位元 writer。合法 IDA Pro 9.4 再確認 `0x1598A`、`0x1567E`、
`0x1CFF0` 與 `0x1BBDC` 都在每次 `0x4E040/0x14818` 候選生命週期後呼叫
`0x4DBFC`，且它們不呼叫 `0x145CD`；因此 `State` 不得保存跨命令的執行期
切片（live slice）。正式命令每次從不可變事件位元組（event bytes）重建
低五位（low5）基底，缺來源時失敗即
關閉；只有 `0x145CD` 的直接呼叫者才可在自己的短生命週期加入 `0x40/0x80`。
上述函式邊界與交叉參照已由合法 IDA Pro 9.4 優先確認，再由 Capstone
逐指令交叉驗證；loader、配置器、constructor、writer 與 consumer 指令保存於
[`fd2_field_composition_lifecycle_disasm.txt`](../data/fd2_field_composition_lifecycle_disasm.txt)。
命令／AI 呼叫端（caller）的重設順序另保存於
[`fd2_ai_composition_flag_lifetime_disasm.txt`](../data/fd2_ai_composition_flag_lifetime_disasm.txt)。

### Native command MP transaction（E0 verified, UI unbound）

`0x21227`（generic command 0 route）在 candidate array 建立後、逐 target effect 前呼叫 `0x1CA89(actor, commandID)`；後者以
`0x4e516(commandID)` 取 record，讀 `byte+5`，直接從 runtime `unit+0x44` 扣除。可達性 gate 已在 selector
先比較 `currentMP >= record+5`，因此扣除不應在失敗 confirm 發生。`battle.SpendNativeCommandMP` 保留這個交易
contract：只接受 raw 0..255 cost，MP 不足／無 unit 一律不變更。它刻意不吃 normalized `Spell`，也尚未接 UI，直到
native candidate confirm、command 0 effect sequence 與原版 renderer 都能一起驗證。

### command 0 正式數值結果邊界（2026-08-22）

command 0 的「數值結果」與「原版 indexed 演出」必須分開計數。正式玩家路徑
`Game.confirm()` 只有在 `0x115B6` 對應的游標 gate 接受、完整 36 筆 command book、
composition baseline 與 class resistance table 都存在時，才可呼叫
`ExecuteBoundNativeCommand0`。成功交易必須一次完成 record 0 的 MP 扣除、
`0x1C75E→0x1C81F` 命中／傷害與目標 HP、actor acted、死亡獎勵、target field reset、
range mode 1 與 modal／selection 清理；任一前置缺失不得部分修改。這可由 Game 層
回歸提升為 `RUNTIME-E1 numeric`，但 `0x2A6BD→0x26152` compositor、sample、數字、
post-resolution 訊息及同狀態原版逐幀比較仍未接，因此不得把數值 E1 寫成
command 0 效果演出或完整 `PLAYER-E2`。

升級的 dynamic producer 現已閉合，但僅限資料層：native `0x1e292` 在 EXP 達門檻後增加 runtime
level，從 `unit+7` 所選 growth row 的 `learn_idx` 經 `0x4e4a2` 查 `0x626b3 + idx*12`，逐一比對最多六組
`(required_level, command_id)`；命中就呼叫 `0x1d79c` OR command bit，並顯示 FDTXT_000 #587「學會了！」。
`docs/data/exe_tables/command_learn.json` 已保存 20 張 raw table（`FF/FF` sentinel 不轉成假資料）。
growth-row 的**raw selector**是 direct ABI：`0x4e4d1(unit+7)=0x620a1+unit[+7]*11`，第 11 byte 就是
`learn_idx`。constructor `0x10d7f..0x10efc` 已閉合 FDFIELD roster `b1→unit+7`；這是 battle
FIGANI/DATO selector 的來源，並不使它和 map `unit+2` alias。remake `State.GainExp` 因此只在已注入這個
editable table 時，於剛達到的 level OR exact
command bit；`remake/assets/data/command_learn.json` 是 runtime copy，
`class_change_growth.json`則保留selector→`learn_idx`。`Game`在每個新battle state同時bind兩張表；缺
`HasBattleFig`、selector row或command row時不學習，且不再以`Portrait`直接索引command table。
selector32經row4在Lv4授予command24；optional selector50的row10不同，不可沿用。legacy standalone
`GainExp`與`Spells`都不補造結果。

### command 24 selector32 FIGANI演出契約（RUNTIME-E1 partial）

固定雜湊原版的`0x276EC`以actor raw `unit+7`取`selector*3+2`效果資源。正常
class-change selector32對應FIGANI資源98；它有15幀，header `(byte1=1, byte2=9,
byte4=6)`。frame4的raw `(1,3,2,0)`屬actor階段，繪製後發布MP並播放由byte4=6
映射的FDOTHER #53 sub3；frame10的raw `(1,2,2,0)`屬target階段，繪製後一次發布
完整傷害並播放sub2。command24分母為1，不能再稱為等分multi-hit。最後一幀後才發布
`Acted`。正式owner必須先驗證原始FIGANI、target idle、palette、panel、兩個sample、
single-target plan與selector32；任一缺漏時交易不得開始。

目前adapter以原始actor/effect與target idle indexed幀、原始delay及shake表執行，並接入
`0x29C90`兩段各10次的BG 0/1/2 viewport轉場。pure compositor接受caller預建的source／target
base；正式owner現以raw terrain control選actor／target BG，以`0x2B659`的
actor frame0..8、扣MP後source snapshot與重設後target idle建立indexed base，並以
`0x2A289→0x18C6D`原生狀態欄取代RGBA panel。尚未接的是`sub_29164`進入actor
phase前的九段figure fade，因此本切片只列
`RUNTIME-E1 partial`，不得宣稱逐幀原版一致；未修改原版的「轉職→Lv4學會→施放」一般玩家
連續E2仍是獨立驗收門檻。主證據見
[`fd2_command24_presentation_ida.txt`](../data/ida/fd2_command24_presentation_ida.txt)。

#### `0x2A289→0x18C6D` indexed戰鬥狀態欄契約

合法IDA 9.4已閉合狀態欄的完整窄consumer。位置只由runtime record raw `+6`
決定：零值在`(0,154)`，非零在`(171,4)`；raw chapter24且unit index17是強制
下方的直接例外。框為FDOTHER#5 LMI1 entry22的opaque 149×42 cell；HP／MP
bar使用raw cells23..30，兩／三位數使用frame entries31..52與93，姓名使用
FDTXT resource0的`record+8+1`及FDOTHER#4 16×16字模。所有座標、數值寬度、
bar公式與繪製順序以
[`fd2_battle_status_panel_ida.txt`](../data/ida/fd2_battle_status_panel_ida.txt)
為準。`battle.RenderNativeBattlePanel`現要求raw `+6/+8` provenance並在私有
320×200 indexed buffer完整成功後一次發布；缺panel、bar、digit、FDTXT、字模或
raw selector時保持目的地與command24 MP／HP／Acted不變。正式command24轉場的
actor／target base已消費此compositor，不再於20張indexed frame上疊兩塊RGBA panel；
仍不得由Camp、Name、Portrait或BattleFig猜測。

#### command24 `0x29C90`背景轉場契約

`0x29C90`已由合法IDA 9.4補閉合為兩段各10次的640-stride viewport compositor，
不是籠統「十列BG動畫」。第一段依resource `0,2,1`循環，把目前VGA從右半滑出；
第二段依`2,1,0`循環，把預建的target BG／status／idle從左半滑入。兩段viewport
起點皆為`32*9..32*0`。command24在`0x52363`初值為0，故單一target的BG resource
必定來自target格raw FDSHAP control byte2，不可用Camp或normalized terrain猜測。
函式內沒有BIOS tick wait；remake逐Draw呈現只屬平台pacing adapter。完整原始定位、
表bytes與限制見
[`fd2_command24_background_transition_ida.txt`](../data/ida/fd2_command24_background_transition_ida.txt)。

command24的source snapshot另由`0x276EC→0x29164→0x2B659`直接資料流固定：
actor BG與扣MP後的indexed actor panel為base；raw `unit+6!=0`且mode=1時保留
actor TAI於`(164,157)`，最後只疊resource98 frame8。`0x2B659`對command24不畫
第四參數idle，也不走DAC pulse。正式owner必須在damage plan建立後用私有actor
record的`MPAfter`預建此snapshot，避免轉場仍顯示扣款前MP。同批的`0x29164`
九段figure fade已由下列caller-specific契約接入；兩者仍分開測試，避免轉場
source snapshot掩蓋前導畫面缺失。
`sub_29C90`後的`sub_2B9A1(targetIdle,0,...)`另明確清零兩個idle globals且不
繪圖；target tail必須從idle frame0／repeat0重新開始，不能沿用actor phase內
雖有遞增但對command24不可見的局部idle相位。

`0x29164`的command24專屬typed契約不得復用終局的透明TAI#3 helper。caller固定
傳入`(actorIndex,1,actorIdle,firstTargetIdle,work640,actorBase320,actorTAI)`；因
第二參數為1，兩支都忽略firstTargetIdle。raw `unit+6==0`在work右半複製base，
將actorIdle frame0由`-80`逐十像素滑到0並present右viewport；raw非零則在左半
讓actorTAI與actorIdle由`+80`滑到0，stage0後把TAI固定寫回actorBase。九張畫面
各自以原始FDOTHER#0 DAC基線做`0x11D40`的`48,42,...,0` subtraction；函式內
沒有delay，所以正式owner只宣稱Draw順序，不宣稱毫秒。所有九張indexed pixels
與palette須在開始前一次預建；缺raw side、base、idle、非零分支TAI或DAC時零發布、
零交易。正式battle presenter現已在effect frame0之前逐Draw消費全部九張，並
保持九張期間MP、HP與acted不變；因此本窄切片達`RUNTIME-E1`。精確DOS tick／
音訊、未修改原版逐幀palette比對及「轉職→Lv4學會→施放」E2仍是獨立門檻。

remake 的可編輯資料模型必須至少表達這些 raw facts，而非固定四個 ring action：

```json
{
  "unit_command_mask": [0, 0, 0, 0, 0],
  "commands": [
    {
      "id": 0,
      "mp_cost": 0,
      "native_record": { "raw_hex": "00000000000000" },
      "label": null,
      "target_contract": null,
      "enabled_when": []
    }
  ]
}
```

`unit_command_mask` 必須是固定五 bytes；初始 source 可只填前四 bytes，但 runtime mutation 不得截斷
第 5 byte。可見 command 的最小 gate 是「該 bit set 且 `current_mp >= mp_cost`」。`label` 已有可編輯、
逐 slot 的原始證據：`docs/data/command_labels.json` 保留 FDTXT_000 的 physical index
`0x1b9+command_id` 與 decoded text。它只表示 native renderer 讀到的文字；空字串或系統訊息 slot
不能被提升為可選戰技。`target_contract`、其他 `enabled_when` 在未有 E0 producer／effect evidence 時維持
`null`／空集合，renderer 必須顯示未解析或禁用狀態，不得將 ID 猜成 attack/spell/item。驗收 test 應涵蓋 bit 0、7、8、31、32、39
的展開順序、MP 邊界（cost-1/cost）與 unknown ID fail-closed；只有在 ID→label/render/effect trace 完整後，
才可淘汰現有 four-way ring approximation。

原版 selector `0x1d51d` 對這份展開 list 使用**每欄四列、欄數可變**的 grid：↑／↓在線性 index 上
-1/+1 並 wrap，←／右只在合法時 -4/+4；Enter／Space 在重新檢查 MP gate 後確認，Esc cancel。`0x1ceed`
renderer 已釘 x=`0x12+0x64*floor(index/4)`、y=`0x67+0x16*(index%4)`，並以 `0x1b9+command_id` 作
`[0x53a7d]` label index；該常駐 table 是 FDTXT_000，且 `0x1b9..0x1e0` 的 40 physical strings
已由 raw resource 逐筆匯出。故 UI state 至少要有 `selected_index`、`rows_per_column=4`、
`visible_command_ids` 與 `cancel_parent`，並以 command count 而非固定四個 action 計算邊界。這是
selector／label ABI，並不證明每個 ID 可達、圖示或 effect；那些欄位仍保持 fail-closed，直到
producer／effect call graph 補齊。

2026-07-27 補上狀態層的窄接線：command grid confirm 只有在 `NativeCommandTargets` 通過後，
才允許已具備 raw executor 的 IDs `0,13–16,20–22,24–29,31` 進入 target cursor；ID30 的
special cardinal cursor、未知 ID、缺少 raw flags/record/resistance 均維持 fail-closed。這些 executor
只保存已證實的 MP/HP/raw-byte mutation，不能因此推導 command 名稱，也不代表 indexed effect
renderer、SFX、動畫或完整 target visual 已完成。

## 5. Campaign / postbattle 設計

每個 battle node 必須明確指定：

```json
{
  "on_win": "post_chNN",
  "on_lose": "lose_chNN",
  "persistent": ["roster_cleanup", "reward", "flags"],
  "next": "town_or_shop_or_preparation_or_ending"
}
```

`postbattle` 是一級可編輯 node，不是 `battle.on_win` 的隱含 callback。允許 `battle→town/rest`、`battle→shop`、`battle→church`、`battle→preparation`、`battle→ending`，也允許連戰區間明確沒有 town/shop。每個 transition 都需有 handler offset／資產／攻略旁證的 evidence list；只有攻略證據時仍標 E3。

Persistent party 的 transaction 順序固定為：結算結果 → reward/drop → transient status cleanup → MaxHP/MaxMP／equipment recompute → roster save → branch flags → 下一個 node。任何中途資料缺失都停在錯誤畫面，不自動跳到下一戰。

`syncPartyFromBattle`／`applyPersistentStats` 現在會在 raw provenance 存在時保存 `NativeRecordByte5/6`，並以
byte5 bit0 決定戰後 HP refill；缺 raw 才退回舊的 `OnField/Alive` projection。這個 fallback 仍是 E1，不能當成
原版 `0x11506` byte-for-byte compatibility；LOADCH 完整 raw record materialization 後才可移除。

### 5.0.1 Handler predicate boundary（E0 slice）

可編輯 handler 的 `if` 不是自由表達式。每種 predicate 都必須對應已反組譯的 native helper，並在 runtime 缺資料時 fail-closed。現有 `any_unit_inactive` 是 remake 對「指定 caller 讀取 runtime `record+5 & 1`」的投影名稱，不是宣稱全域生命欄位；`roster_has(char_id)` 對應 `0x33499` 對永久我方名冊 `[0x53bf7]` 的 `record[+8]` 掃描，`char_id` 僅限原版永久玩家 `0..31`，不得改以暫時出戰隊伍、portrait、NPC 或 story actor 推論。

目前 `cmd/fd2` 的 `any_unit_inactive` 在整個 runtime roster 都具 `HasNativeRecordByte5` 時已 strict 使用 raw predicate；只有舊／混合 authored JSON 缺 raw 時才使用 `OnField/Alive` 相容 projection，這是明確的 E1 gap，不是 native parity。constructor、已知 damage/death writer、revive writer、`deactivate_unit` 與 FDFIELD `+6` source 已同步 raw provenance；仍要補 zero-HP 初始 record 與所有 LOADCH 分支，並讓 strict binding 缺 raw 時 fail-closed。

ch14 pre-handler (`0x334d9`) 是已閉合的動態文字例：`0x33499(12)` 的回傳經 `xor al,1; mul 3` 得到 FDTXT_015 base index。因此有 char_id 12 時依序播放 index `0/1/2`，否則 `3/4/5`；中間仍依原順序 pan `(24,17)`、呼叫 acting 48、最後 focus slot 0。資料以 `handlers/ch14_pre.json` 的兩個結構化 `if roster_has` 與 address-keyed binding 表達，保留六個原始 dialog call-site，避免在 runtime 猜 EBX/EAX。

`layout_units` 是另一個必須 address-keyed 的 handler primitive。Official IDA 9.4 定義 `0x233c6..0x2345b`，並確認它被 15 個不同 post-handler caller 共用；其 call-site supplies X/Y byte arrays、slot range、pose source、optional special-slot placement，以及 focus/camera inputs。每個可播放 binding 必須保存完整 materialized `(slot,x,y,pose)` 與 camera 值；不得把任一 caller 的 table 位址、長度或 special-slot rule 泛化給其他關。缺少任一欄位的 layout 保持 compile issue／runtime fail-closed。

玩家第 8 戰的 raw ch07 post 是這項契約的完整 E1 切片。IDA Pro 9.4 與
Docker Capstone 共同固定 `0x234BB..0x235BC`：入口 runtime 不是舊情境所列的
groups 0／1／8／9／10，而是十名 party 後只附加 group 0 的十九筆 records，
因此最小 frontier 為 29，slot 28 才能保持 raw `+8==5`。event27 在回合 2..7
逐組附加兩筆，故戰後只接受 `29,31,33,35,37,39,41`。處理器依位址綁定
slots 0..9 與 slot 28 的佈位、ACTING 33／34、FDTXT_008 index 3／4，最後
執行 `JOIN5→sync_party→chapter 8` 並進 `town_ch09`。`0x23599` 的
`0x11D40(0,255,64)` 會把全部六位元 DAC 分量夾為 0，呼叫者再
`memset(0xA0000,0,0xFA00)`；因此只有這個 exact call site 可降低為完整黑色
覆蓋，其他 `0x11D40` 使用仍失敗即關閉。原始位址、資料表與限制見
[`fd2_ch07_post_ida.txt`](../data/ida/fd2_ch07_post_ida.txt)。尚缺未修改 DOSBox
一般玩家同狀態比較，不能提升為 E2 或逐像素一致。

玩家第 10 戰的 raw ch09 post 是另一個完整 E1 垂直切片。IDA Pro 9.4 主判讀與
Docker Capstone 覆核固定 `sub_235F9`（`0x235F9..0x23790`）：先執行
`0x1F882` 的 delta 0→63 共64次六位元 DAC 淡出，再由 `0x13536` 對全部
runtime records 執行 raw `+5 &= 0x7F`。後續不經 helper 的
`0x2362D..0x236E2` 直接寫入已轉成 address-keyed `direct_record_patch`：只改
slots 0..10 的 `+0/+1/+3`、slots 50／51 的 `+0/+1/+0x26`、slot52 與 slot5
的 `+5`，以及 camera／cursor／visible cursor；未列欄位保持不變。原始 offset
不從單一清零寫入提升為 gameplay 名稱，執行期也會在任何值域、provenance 或
frontier 錯誤時於寫入前原子拒絕。

第10戰戰前固定 group0 四十二筆，回合5的 event32 再 append group1 八筆；永久
隊伍是否包含凱麗 id12 使戰後 runtime frontier 成為60或61。這兩個數量由可編輯
roster producer 與回歸交叉支持，仍標強推論，不冒充未修改 DOSBox E2。
`0x236EE` 重繪後，`0x1F525` 依 delta 64→0 共65次恢復原始 DAC，接著播放
FDTXT_010 index4、ACTING37、index5，執行 sync、JOIN11、JOIN6、chapter10，
最後保留正式 `town_ch11`，不直接跳下一戰。`syncPartyFromBattle` 現在會在 typed
identity 尚未物化時讀 raw `+8`，並把該 identity 保存到 persistent snapshot，
避免 JOIN 後的原始身分被 legacy `Fig` 覆寫。完整非破壞性位址證據見
[`fd2_ch09_post_ida.txt`](../data/ida/fd2_ch09_post_ida.txt)；尚缺一般玩家同狀態
DOSBox 逐幀比較，因此不能宣稱 E2 或原版時序一致。

`0x24618..0x24754` is a separate map-transition compositor, not an actor `acting` decoder. Its callers include post-handler functions `0x33af1`/`0x33c9d`; it renders a 13×8 terrain region to an offscreen buffer, performs exactly nine strip-composite passes with a caller-supplied progression, then performs palette updates from 0 through 62 in steps of 2 (4 ms each). Its first two arguments feed tile geometry (`arg1*24+12`, `arg2*24+16`); the third starts a radial radius and the fourth increments that radius per pass. `0x22046` supplies a fixed scale of 16 to its two `0x219ad` radial LUT passes and derives its final rectangular radius as `trunc(radius*1.6)`. Its remaining constants are a pass row range `[start_y,end_y)=[0,192)`, not a source coordinate or blit width; the editable fields retain those names. A playable binding must either supply a verified transition adapter or fail closed—never lower it to `act`, `pan`, or an arbitrary fade.
The `0x22046` inner order is executable as a raw primitive: `fdother.BuildNativeIndexedTransitionPass` preserves its scale-16, second-radial start row, and final-rectangle `a2` alias; `fdother.ApplyIndexedTransitionPass` validates both radial specs and the final centered rectangle before applying the first LUT remap, requiring the caller-owned `0x127a9` redraw callback, then applying the second remap and rectangle LUT. `indexedmap.BuildNativeTerrainCells` materializes the exporter’s raw FDFIELD tile/high-byte arrays, and `indexedmap.ComposeNativeTransitionFrame` supplies the verified terrain→unit/foreground→pass→312×192 viewport composition with atomic work/VGA commit when all raw banks and controls are supplied. `loadNativeMapAssets` requires FDOTHER#3 LUT entries 1..9 and the exact 768-byte FDOTHER#0 six-bit DAC before exposing the all-or-nothing native map bundle. `fdother.BuildNativeIndexedTransitionSchedule` preserves the outer nine-pass FDOTHER#3 LUT index order `9..1`, caller radius progression, 5ms pass delay, 500ms tail hold, and `0x11df2` palette deltas `0..62` step 2 at 4ms; `NativeIndexedTransitionLUT` resolves those raw indices only against the 256-byte bank entries. Docker Capstone recheck of `0x11df2` ([saved disassembly](../data/fd2_11df2_palette_disasm.txt)) proves that every RGB component is reread from immutable `[0x53a65]`, then receives delta and an upper clamp at 63 before DAC output. These steps are baseline-derived range writes, not cumulative additions to the current DAC.

`cmd/fd2` now owns the playable adapter for this contract. Beat start preflights the complete field, tile-aligned camera, raw actor provenance, selector cache, LUTs, buffers and DAC before publishing a job; rejection leaves the existing map buffers unchanged. Each of the nine indexed presents must receive a real `Draw` acknowledgement, the 500ms tail is 30 update ticks, and each of the 32 cumulative DAC mutations must likewise be drawn before continuation. This preserves every native visual state and prevents a fast update loop from collapsing to the last frame. Because Ebiten's 60Hz host presentation cannot display distinct 5ms/4ms native writes at their original wall-clock cadence, the adapter deliberately uses one host present as the minimum duration and does **not** claim DOS timing parity. Raw ch28 post now consumes this adapter through the formal campaign binding and reaches `preparation_ch30`; this is `RUNTIME-E1`, not DOS timing parity or `PLAYER-E2`. The separate `0x2BCE5` terminal owner remains an ending-path question, not a blocker for this postbattle node.

Native battle-local event state is a separate editable boundary. `[0x53ad5]` points to a 32-byte table, but an index must not be named from a single reader. Entry 12 is set only on the successful `0x356bc` path after item `0xd0` is consumed; that path also runs its presentation, spawns FDFIELD group 1, JOINs char 31, and displays FDTXT #4. ch25 post later reads entry 12 to select its FDTXT base (`+5` or `+8`). Earlier official IDA analysis found only generic indirect dispatcher xrefs and therefore withheld a map-local binding; direct FDFIELD evidence now supersedes that limitation: map25 field slot2 maps selector1 at `(1,46)` to event61/`0x356b7`. The editable rule preserves item D0, entry12, spawn1, JOIN31 and text indices 2/3/4. Its presentation is typed as `FDOTHER.DAT` resource 45, 59 frames, destination offset 48356 in a 320-byte stride surface, transparent mode -1 and two native delay ticks per frame. FDTXT_026 strings 2 and 3 are portrait-less visible messages rather than zero-utterance controls, so the resource has 63 displayed utterances and count-aligns exactly with `ch26.json`; event61 string2/3/4 map to scene0 line10、scene0 line11、scene1 lines0–9. Chapter 26 now uses runtime append construction with only group0 initially materialized, keeping group1/char31 pending until this event. The battle core now separates planning from commit: planning verifies selector1, exact editable rule, native eight-cell inventory projection, entry12 and the pending group1/char31 identity without mutation; commit is rejected until exactly 59 frames were reported, then revalidates D0 before native compact removal, sets entry12, appends group1 and returns char31 to the campaign owner. This core boundary was the prerequisite for the runtime owner described below; it must not be substituted with a treasure or generic party predicate.

2026-07-29 執行期接線進一步縮小此邊界。直接呼叫證實 selector1 位於 AI 分派器 `0x13E77`，以及玩家 `0x18AEF/0x18B0C/0x18B66` 三個 action handler 成功返回臂；它不是 selector0 的向左格提交點。重製端以 `finishSuccessfulUnitAction` 作為唯一成功閘門，待命、攻擊、一般法術、原始指令、三種已閉合物品交易及 AI 都在實際 mutation 成功後才進 event61。攻擊與攻擊型法術把閘門保存於全螢幕演出的 `after`，直到演出結束才執行；目標不合法、取消及 executor 錯誤均不會抵達閘門。缺 D0 只排入可編輯 FDTXT2；成功臂先排 FDTXT3，合成目前 320×200 索引地圖，以每次呈現兩個 BIOS tick 累積貼上 FDOTHER#45 全 59 幀，再提交 D0／entry12／group1、持久化 JOIN31，最後排入 FDTXT4 十句。

同日的控制流勘誤補上正式 map25 runtime 來源。`0x10010` 不是一般章節進場函式，而是從 FD2.SAV plaintext `0x30c3` current-runtime header 恢復目前戰鬥；新遊戲與四槽讀檔的 pre-handler 從 `0x25ebb` 返回 0 後，main `0x25dce` 直接呼叫 `0x117e7`。ch25 pre-handler 已明示 `PAN(9,39)→FOCUS_UNIT(0)`，而 ch26 party slot0 的部署格是 `(15,46)`，因此戰鬥初始 native view 為 camera `(9,39)`、absolute cursor `(15,46)`、visible cursor `(6,7)`。玩家沿已閉合的原版 cursor state machine 移至 event61 `(1,46)`，會確定得到 camera `(0,39)`、visible cursor `(1,7)`；HUD anchor 因 row>5 且 column<3 寫成 `0xf2`。gate A 是 `0x173BA` 可切換且由 slot metadata/current-runtime header 保存的 raw 設定；現行 remake 尚無該選項，正式可達預設為1。`battle_ch26` 現已資料化此 pre-handler 最終狀態，真實資產 Docker regression 不再注入測試 view/HUD。這關閉 production E1 路徑，但未提供同 roster/event/tick 的 DOSBox 像素比較，仍不是 E2。selector1 的其他玩家動作與 AI 接線狀態，以前一段較晚的成功閘門勘誤為準。

The same failure exposed a repository-wide renderer-input omission: only map0 had `native_tile_blit_modes` and `native_terrain_control`; map1–32 silently loaded neither. The synchronized source is now explicit and reproducible. `tools/sync_native_map_renderer_inputs.py` verifies the immutable FDFIELD.DAT and FDSHAP.DAT against `fd2-reference-files.json`, then derives composition entry byte+3 and the map's terrain/control resource without rewriting unrelated editable fields. All 33 maps pass exact dimension, cell-count, tile-bound and control alignment checks. This raises the tactical map data boundary to E0 for all maps, but does not supply dynamic camera/cursor/HUD globals or prove any post-ch01 visual comparison.

`battle.ApplyNativeFieldModeEvent` 已提供 event59/60 的原始 mode-range
執行器：先驗證 selector、trigger byte6 provenance、完整目標範圍與每筆
byte34 provenance，全部成立後才原子式保留高四位並寫低四位 0。
合法 IDA 9.4 固定 `0x13488→0x1300D→0x13A44`：path byte1 的七拍
格步驟提交 runtime `x-1` 後才呼叫 selector0；`0x12E38` 以
`x + mapWidth*y` 交叉確認座標順序。`stepBattleWalk` 因此只在向左格
步驟提交後執行 event59/60，向右與其他方向不會泛化觸發。執行器仍拒絕
event61；後者只由上述 action handler 成功返回後的 selector1 共用閘門擁有。

Inventory gates are distinct from item-consuming event commands. Native `0x24b14(item)` scans only runtime slots `0..15` through `0x31860` and returns found/not-found; it neither filters camp/activity nor removes an item. In ch26 post, `0x24b14(0x64)` selects the sky-key success arm; that arm contains no `0x1b8e7` call and only later performs sync/chapter increment/persistent cleanup. The missing arm is a separate ending presentation path. Therefore an editable `inventory_gate` must preserve item `0x64`; it may not be lowered to a recipe, reward, or consume action.

`0x25052(start,delay_ms)` is an independently editable palette-ramp primitive: it emits inclusive descending `0x11df2(0,255,start..0)` baseline-derived updates, waiting after each update. The ch26 success arm calls `(5,80)`, `(4,80)`, `(3,80)`, then `(2,80)`, `(2,80)`, `(2,80)` interleaved with native waits. This is not a black fade and must preserve every delta including zero; compiler input is restricted to immediate `start∈[0,63]` and non-negative delay.

`0x1f882` 是獨立的原生 palette 淡出，不是 timing／vsync helper：它以
`ebx=0` 起始，對 delta 0..63（含兩端）各呼叫一次
`0x11d40(0,255,ebx)`，每次等待2ms。`0x1f525` 則由64遞減到0，共65次相同
baseline-minus-delta 寫入。handler compiler 分別保存為
`native_palette_fade_out{0→63,2ms}` 與 `native_palette_fade_in{64→0,2ms}`；
runtime 已用目前六位元 DAC baseline 與索引 framebuffer 執行，每一步都必須經
Draw acknowledgement 才前進。60Hz host 無法呈現 DOS 的2ms wall-clock cadence，
所以這只證明寫入序列與端點，不宣稱時序 E2。ch00 `0x3241f` 因 raw FDICON key
尚未閉合，仍是唯一明示的 RGBA E1 近似；其他 call site 不得沿用該 fallback。

raw ch19 post（玩家第20戰）已由 IDA Pro 9.4 與 Capstone 重新閉合。入口先以
四張 byte table 對 slots `0..15`、`52..60` 寫 `record+0/+1/+3`，並設定
camera／cursor `(26,31)`；resource59 使用 slots53–60，resource60在
`spawn(group1)` 後直接使用slot83。map19 group0為67筆，整備固定record0再選15筆，
故正常玩家區16筆、入口frontier83，group1一筆後為84。更重要的勘誤是
`0x23FFE/0x24005`：`round > 15` 會跳過整段group1、ACTING60–62、
FDTXT index14–16與JOIN28；只有`round <= 15`執行，JOIN25則在共同路徑。
正式binding已接`postbattle_ch20_persist→town_ch21`，決定性回歸覆蓋15／16邊界；
`ch20.json`亦撤回預載全部70筆FDFIELD records的舊拓撲，改由
`runtime_append_groups`只在開場追加group0，group1留給戰後條件分支；
證據仍為E1，未宣稱一般玩家DOSBox E2。詳見
[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)。

### 5.1 目前 editable graph audit（E1，不等同原版 E0）

`remake/assets/scenarios/campaign_full.json` 的 30 個 battle node 已逐一展開
`on_win`，並沿著 post/cutscene 節點走到第一個可操作戰間節點。這張表是目前 remake
的可編輯基線；只有標成「native 待核」的項目仍不可宣稱已還原原版 handler。

| battle | 勝利後第一個戰間節點 | 路線型態 | native 證據狀態 |
|---|---|---|---|
| 01 | `story_ch02` → `town_ch02` | 劇情→城鎮 | E0 hub gate；逐章文字／E2 待核 |
| 02–20 | `story/postbattle_chNN` → `town_ch(NN+1)` | 劇情／持久化→城鎮 | E0 hub gate；逐章 handler／E2 待核 |
| 21 | `story_ch21_post_sky_key_intro` → `inventory_recipe_ch21_sky_key` | 劇情→合成 gate（非直接下一戰） | gate E1；native 待核 |
| 22–24 | `postbattle_chNN_persist` → `preparation_ch(NN+1)` | 持久化→整備 | E0 preparation route；逐章 handler／E2 待核 |
| 25–26 | `postbattle_chNN_persist` → `town_ch(NN+1)` | 持久化→城鎮 | E0 hub gate；逐章 handler／E2 待核 |
| 27 | `inventory_gate_ch27_sky_key` → success/missing branch | 道具 gate→分支劇情 | gate E1；native 待核 |
| 28–29 | `postbattle_chNN_persist` → `preparation_ch(NN+1)` | 持久化→整備 | E0 preparation route；逐章 handler／E2 待核 |
| 30 | `ending` | 終局（不接下一戰） | ending renderer fail-closed |

因此不能以「battle node 有 `on_win`」推導下一節就是下一戰；town、shop、church、
preparation、inventory gate 與 ending 都必須留在 graph。下一個 SDD-2 子任務是以
原版 handler offset／DOSBox 操作逐列補 E0/E2 證據，並為每列加入 save/reload regression。

### 5.3 Native postbattle hub gate（E0，IDA 9.4）

`0x2D093` 是 `0x2CAD7` 的 selectable-hub 分支所到達的 raw selector
分派器，不是外層戰役迴圈的直接 callee。其 `[0x5412B]` option
`0→0x2FC85`、`1/3→0x2E341`、`4→0x3072F`；option 2 走確認及
`0x318AD` 整備路徑。酒店、武器／道具／神秘商店與教會等名稱具有
FDTXT 文字、資源及 mutation writer 旁證，但 address-level 主契約仍以
raw option→callee 為準。`0x2E341` 依 raw hub variant 選資源
12／29／63，`0x2FC85` 及各設施完成後可回到 hub；外層
`0x25DE5` 恢復前不會選下一戰 BGM。

原始 30-byte table 已固定 `byte_526B9[22..24]` 與 `[27..29]` 為 1，
`[0..21]` 與 `[25..26]` 為 0；`0x2CAD7` 中非零 entry 走
preparation-only 路徑，零 entry 才建立 selectable hub。chapter index
是原版下一戰索引，不是玩家畫面章號。每章文字、游標圖與 DOSBox 視覺
時序仍待 E2；不得把已證實的 hub／prepare 分支壓成直接下一戰。

`fdother.ResolveNativePostbattleRoute` now preserves this gate as editable
address-level data: nonzero `0x526b9[index]` entries return the raw
`0x318ad` preparation route before reading a hub option; zero entries map
options `0`, `1/3`, `2`, and `4` to `0x2fc85`, `0x2e341`, `0x318ad`, and
`0x3072f` respectively. It performs no scene call and does not label the
callees as hotel, shop, church, or leave; invalid index/option fails closed.

合法 IDA Pro 9.4 再固定 `0x2CAD7` 的 raw return 契約：直接整備
`0x318AD` 或 selectable hub 的子流程回傳 0 時，都在 `0x2CAD7` 內重複；
直接整備或 option 2 回傳非零時，`0x2CAD7` 回傳 raw 0；option
0／1／3／4 回傳非零時則回傳 raw 1。外層 `0x25DE5` 只在 raw 0 時繼續
章節索引 `0x51D71` 表。`fdother.ResolveNativePostbattleOutcome`
保存內部重複及 raw 0／1，不替 raw 1 命名某個設施、結局或離開原因。
直接證據見
[`fd2_postbattle_gate_outcome_ida.txt`](../data/fd2_postbattle_gate_outcome_ida.txt)。

The shop-family subscene is represented by
`fdother.ResolveNativeShopServiceRoute`: raw hub variants `3` and `5` select
FDOTHER resources `29` and `63`, while every other variant selects resource
`12`. The selector mapping is now independently closed beyond address level:
`0→0x2f0b0` purchase, `1→0x2f642` sell, `2→0x2f883` equip, and
`3→0x2f8ea` inventory transfer. These names come from the mutation writers,
not icon appearance: purchase inserts through `0x1bb8c`, optionally equips,
then `0x2d516` debits gold; sell computes `floor(3*price/4)`, credits through
`0x2d3ff`, removes through `0x1b8e7`, then recomputes; equip uses
`0x1c1c3→0x1c142→0x1b750` with no gold write; transfer uses the same proven
source/destination topology as the church caller. The resolver still returns
data only and never invokes a scene or bypasses the production UI gate.

The hidden-shop gate is not a persistent unlock flag. At
`0x2cd16→0x4e4b9(chapter)`, each town uses a 0x1f-byte record rooted at
`0x6238d`. The input loop compares the current normal selection with record
`+1` and the BIOS scan byte with record `+2`; only an exact pair writes
selection `5` at `0x2cef7`. The dispatch at `0x2d28c` then reaches
`0x2e341`, whose selection-5 branch uses resource63/variant5. Raw scans
`0x54..0x5d`, `0x5e..0x67`, and `0x68..0x71` are respectively
Shift/Ctrl/Alt-F1..F10. All 23 town records (player chapters2..22,26,27) are
editable as `native_secret_gate {selection,scan_code,to}` and runtime requires
the exact chord while the required normal option is selected. The chord does
not dispatch immediately: the hub redraws selection 5, and only a subsequent
Enter/Space reaches `0x2d093→0x2d28c`. Legacy
`SecretIf` and `found_secret_ch*` remain authored extension mechanisms only;
they do not establish native parity.

2026-08-20 的正式戰間回歸已把上述資料契約接到玩家第25戰：戰後
`62→70→71`、JOIN26／29、`town_ch26` 存讀檔之後，只有selection4＋Shift+F5
（BIOS scan `0x58`）能揭露selection5；ch02的Shift+F1（`0x54`）會拒絕。
後續確認進入`shop_ch26_secret` variant5，驗證商品195／207／40，並經四幀
離店動畫回到`town_ch26`、保留selection5及持續隊伍。這是重製端正常戰間
`RUNTIME-E1`，不是未修改原版同狀態`PLAYER-E2`。

The town scene owner is now closed at E1/production level. `0x2cd16` reads
record byte 0 and indexes the three-entry resource table at `0x526d7`,
selecting FDOTHER resources `11`, `61`, or `62`. `0x2cf71` redraws
FDOTHER#10 (62×26) at scene `(244,162)`, FDTXT indices
`0x1ef+selection` from `(252,168)`, and FDICON sprites `0,1,2,1` at the
variant/selection coordinate tables rooted at `0x52635/0x52647`. The scene
then copies only 312×192 to VGA `(4,4)` through `0x11eb0`; it is not a
320×200 top-left present.

All 23 editable town nodes now carry the raw `native_town_variant` value
0/1/2. `ComposeNativeTownFrame` consumes the original indexed resources and
fails closed for any missing asset, invalid variant, selection outside 0..5,
or pulse outside 0..3. Production uses the original right-key decrement and
left-key increment wrap over selections 0..4; hidden selection 5 remains
renderable and leaves through the same wrap rules. The pulse counter advances
after a signed BIOS low-word delta of four ticks and maps counter 3 back to
sprite 1. [`town-hub-remake.png`](../figures/town-hub-remake.png) is a
source-built runtime capture of ch02 variant0/selection0. The matching original
frame is now preserved as
[`town-hub-original-dosbox.png`](../figures/town-hub-original-dosbox.png),
captured after the ch00 postbattle handler through the original campaign gate.

### 變體二視覺證據邊界（2026-08-09）

以固定雜湊的原版 `FD2.EXE` 與原始 `FD2.SAV` 為來源，在 Docker 中建立只存在於
`/tmp` 的研究副本：slot0 只填入目前持久隊伍資料（current persistent roster），並將原始 metadata
chapter 設為 `5`、roster count 設為 `4`。標題→LOAD→slot0 的實際 DOSBox 時間線
載入後顯示城鎮變體2；這是修改存檔路徑，不能當成未修改一般玩家戰後 E2。
同一畫面以 320×200 內容用整數 `-scale` 放大至 640×400，再與
`FD2_CAMP_NODE=town_ch06`、`FD2_SHOT_TOWN_STATE=selection,pulse` 的重製端畫面
逐幀比較。正常 selection0–4 對應 pulse `0,1,2,1,0`，整幀絕對誤差均為零；
完整雜湊、時間線與修改副本雜湊見
[`native_town_variant2_e2.json`](../data/native_town_variant2_e2.json)，並列圖僅供
審查見 [`town-variant2-five-selections-original-vs-remake.png`](../figures/town-variant2-five-selections-original-vs-remake.png)。
selection5 的該城鎮 BIOS 掃描碼／後續 Enter，以及未修改一般玩家路徑仍然是待辦；不得把
本節外推成 23 個城鎮已完成。

### 變體一視覺證據邊界（2026-08-09）

以同一固定雜湊原版 `FD2.EXE`／`FD2.SAV` 建立 Docker `/tmp` 研究副本；slot0 只填入目前
持久隊伍資料，將 slot0 raw chapter 設為 `0x0b`、roster count 設為 `4`、currency 設為 `0`。
這條 LOAD 路徑觀察到原版 variant1，並非未修改一般玩家戰後路徑。原版內容裁為 320×200，
以 ImageMagick `-scale` 整數放大至 640×400；重製端使用 `FD2_CAMP_NODE=town_ch12` 與
`FD2_SHOT_TOWN_STATE=selection,pulse`。五個正常選項分別以實測 pulse `1,2,1,0,1` 配對，
五組未遮罩整幀 `compare -metric AE` 均為 `0`；這些 pulse 只表示本次畫面配對相位，不推論
原版計時器映射。完整雜湊、存檔欄位、時間線與限制見
[`native_town_variant1_e2.json`](../data/native_town_variant1_e2.json)，並列圖見
[`town-variant1-five-selections-original-vs-remake.png`](../figures/town-variant1-five-selections-original-vs-remake.png)。
selection5 的該城鎮 BIOS 掃描碼／後續 Enter、未修改一般玩家路徑與其餘城鎮仍維持失敗即關閉。
The first [`side-by-side`](../figures/town-hub-original-vs-remake.png) exposed
that the FDICON sprite could match while the label glyph did not. Capstone
re-read of `0x4ea2a` corrected the shared glyph shadow from the remake's
same-row `x-1` write to native next-row `(x-1,x)` writes. After that correction,
selection0/pulse2 hashes to `8a6a4b03946d1958d3af95fd4bd775c3`
in both original and remake; its
[`difference image`](../figures/town-hub-pixel-diff.png) is entirely black.
The original Left input then produces selection1 without resetting global
counter `0x54133`; [`selection1/pulse2`](../figures/town-hub-selection1-original-vs-remake.png)
also matches completely at `60a4791d60b32fd6efc82864afd63525`.
That first trace closed two E2 visual states and one Left transition. A
subsequent variant0 trace closes the remaining visual states:
selection2/3/4 match
`10017309d3c833c8e323e8739d624f8b`,
`0e1db5b95951230b3c13d1f0309296d2`, and
`1577fc5749410221497f512b52a12dbe`; Shift+F1-revealed selection5 matches
`e695d6cf391c45ccf4d2cf70096eb9bf`. Right `0→4` and Left `4→0` reproduce
the same selection4/0 frames without resetting the pulse. The
[`six-selection sheet`](../figures/town-hub-six-selections-original-vs-remake.png)
places original DOSBox states on top and remake states below. Remaining E2 is
variants 1/2 and secret-shop return, not variant0 steady redraw. The hidden
confirm path itself is now replayed through the unchanged town/shop owners:
Shift+F1 reveals selection5, the next Enter reaches variant5/resource63 with
DATO portrait `0x84`.

The ch02 variant5 service menu now has two whole-frame E2 pairs at gold 0.
The original and source-built remake raw RGB MD5 are
`12fad3c03096aae48098c8f9074370c7` for service0 phase0 and
`e5654e8ed03d1e4fd30b2c76106bb7a1` for phase2; both comparisons have
absolute pixel error count zero. The
[`side-by-side`](../figures/secret-shop-ch02-original-vs-remake.png) shows
the selected phase pair. `FD2_SHOT_SHOP_STATE=service,pulse,gold` is a
strict screenshot-only hook: it accepts only a stable claimed native shop
menu, service/pulse `0..3`, and gold `0..99999999`; it cancels only the
presentation opening job immediately before capture and never runs a
transaction.

This E2 exposed a production-only assertion error that the lower-level
fixture could not catch. `ComposeNativeShopServiceSteadyFrame` already maps
native phase `0..3` to cell variant `phase/2`, but its runtime caller also
passed `nativeShopUIPulse/2`. The double division made phase2/3 permanently
render the normal cell. The caller now passes the full phase and a production
consumer regression requires phase0 and phase2 frames to differ. Confirmation
widgets still receive their proven 0/1 cell variant; this correction is not
generalized to unrelated compositors.

The next original input trace closes the complete variant5 menu boundary:
Right reaches `0→1→2→3→0`, Left reaches `0→3`, and each of those six
320×200 frames matches a deterministic remake service/pulse with absolute
pixel error zero. Escape finishes the native closing lifecycle and restores
the town with hidden selection5 still visible; that frame also matches the
remake town selection5/pulse1 with zero error. The
[`two-row sheet`](../figures/secret-shop-ch02-services-return-original-vs-remake.png)
shows services0..3 and the restored town, original above remake.

This return E2 disproved the production boundary behavior: `enterNode()` reset
every town selector to0 after `leaveShop()`. `leaveShop()` now preserves the
proven native dispatch variant before clearing shop state and, only when the
editable next node is a town, restores variant1/3/5 as selection1/3/5.
Variant5 is directly confirmed by this DOSBox trace; variant1/3 follow the
already closed `0x2d093→0x2e341` dispatch mapping. Custom/non-native shops
retain selection0.

The ordinary ch02 shops close the remaining three-variant menu gate. Town
selection1 enters variant1/resource12 and selection3 enters
variant3/resource29 through the original input path. Ten steady samples per
shop show the same phase0/2 alternation as variant5 (the first variant3 sample
was still a transition and is excluded). Every admitted frame has whole-frame
absolute pixel error zero against the corresponding production state. The
selected-phase raw RGB MD5 pairs are variant1
`69003be54f47c221916c1ed89cf1d26f`, variant3
`dd5d80bb761cc87980dff066773f6763`, and variant5
`e5654e8ed03d1e4fd30b2c76106bb7a1`. The
[`three-variant sheet`](../figures/shop-variants-1-3-5-original-vs-remake.png)
places original DOSBox above remake. This closes the ch02 stable service-menu
variant/background/portrait/text/layout gate; child panels cannot inherit E2
from the parent menu.

The first child-panel trace now closes the ch02 weapon purchase-list steady
state and navigation. A strict screenshot-only
`FD2_SHOT_SHOP_PURCHASE_STATE=selection,start,gold` adapter admits only a
production-claimed native purchase mode with a valid goods selection and a
normalized even two-column window start; mismatched mode, variant, resource,
selection or window fails closed. Original input reaches selections
`0→1→3→2` by Right, Down and Left. After excluding the entry transient and
portrait animation, all four original 320×200 stable frames have absolute
pixel error zero against production. Their raw RGB MD5 values are
`1589cee3c068936f0beb6058cfd63991`,
`7480dbb0284b033b4e9ad8c8c7a8b78e`,
`3c0a2c935260b8ca80432b25b3600111`, and
`48d6182e261ebce574b08c4778b8a072` for selections 0, 1, 3, and 2.
The [`purchase-list sheet`](../figures/shop-purchase-ch02-selections-original-vs-remake.png)
places original above remake. This evidence covers only list steady rendering
and cursor input.

The next normal-path trace closes the purchase confirmation steady state and
its horizontal input. From ch02 weapon good0 the original closes the list and
opens the literal `布衣 / 50元 / 要不要啊？` question with Yes selected;
Right reaches No and Left returns to Yes. The screenshot-only
`FD2_SHOT_SHOP_CONFIRM_STATE=good,choice,pulse,gold` adapter requires a
production-claimed native shop, shared class assets, matching variant,
admitted portrait/background and a real editable good. Choice must be0/1,
pulse0..3 and gold0..99999999; invalid state fails closed and normal input
never reads the hook.

High-frequency original sampling separates the normal and visibly selected
pulse. The selected Yes raw RGB MD5 is
`7a07b1c064ca2c431bc97c798dcfd51e`; selected No is
`56f6ffb003e87cbc63d7a915ac4b5dd0`; each matches the corresponding
production 320×200 frame with absolute pixel error zero. The normal frame
also matches at `b8cce25df13447e73e1750a8b2edaf0f`. The
[`confirmation sheet`](../figures/shop-purchase-confirm-ch02-original-vs-remake.png)
shows selected Yes/No with original above remake.

The next gold0 trace closes the insufficient-gold stable feedback. After Yes,
the original restores the saved question region, appends `錢不夠！`, and
shows the mode-one wait marker. The strict screenshot-only
`FD2_SHOT_SHOP_INSUFFICIENT_STATE=good,gold` adapter accepts only an admitted
editable good whose price is greater than the supplied legal gold, performs
no debit or recipient mutation, and requires the final native compositor.
Docker Capstone revalidation found that `0x197e5` presents four inward choice
frames and then `0x19913..0x1994c` restores the saved 310×86 question region;
`0x16c57(1)` initially blits FDOTHER#5 cell18 and alternates cells18/19.
Using the fourth inward frame produced 563 absolute RGB error. The current
ch02 screenshot state deterministically recomposes the same question pixels
and adds the caller-owned wait-marker anchor, producing whole-frame error
zero. It does not yet implement a generic saved-buffer restore owner.
Original and production raw RGB MD5 are both
`6babcedfe2017a7457924c4df65ba7dc`; the
[`insufficient-gold sheet`](../figures/shop-purchase-insufficient-ch02-original-vs-remake.png)
places original left and remake right. Recipient/no-recipient/full/success,
sell, equip, and transfer still require their own same-state DOSBox traces.

Standalone equip has a separate scene contract from the optional equip prompt
inside purchase. `0x2f883` calls `0x2e6b8` for the two-column party roster,
closes that child, saves/restores the shop global around `0x1bffe`, and redraws
the shop only after the child returns. `0x1bffe` owns the full-screen native
item/status panel: `0x17e0b` saves the current VGA frame and opens
`0x17eef+0x184c0` through frames 11→0; `0x1b9de(actor,0)` accepts every
occupied item without the battle-use effect-type gate; `0x1c1c3` rejects an
incompatible row by jumping directly back to the selector with no feedback
owner; `0x1c142→0x1b750` updates the same-ID-category equipped bit and derived
stats, then rebuilds the panel in place. Exit alone runs frames 0→11 and
restores the saved shop framebuffer.

Production now preserves that ownership instead of reusing the purchase
recipient or priced facility-item widget. `EquipNativeCompactSlot` proves the
compact `Inventory` projection equals the occupied native eight-cell order
before mutation, retains ignored raw holes/stale bytes, and fails atomically
on divergence. The original item-panel indexed compositor and twelve-frame
clip schedule are reused directly; empty inventories may display the panel
and exit without inventing a message. DOSBox same-save/tick E2 remains open.

Inventory transfer `0x2f8ea` is a shared service callee: shop `0x2e341`
dispatches service index 3 to it, while church `0x3072f` dispatches raw index
1 to the same body. Neither caller owns a different transaction. The body
shows FDTXT512 before every source roster, FDTXT511 plus the selected source
name when no signed-nonnegative item cell exists, renders the compact source
items through `0x2e0bd→0x2dc55(mode=1)`, then shows FDTXT510 before a second
full-party roster. A destination with eight occupied raw flags receives the
variant-selected FDTXT506 plus its name; success alone executes
`0x1b722→0x1b8e7(source)→0x1bb8c(destination,item)→0x1b750(source)`.
There is no gold writer or confirmation box.

The second roster does not remove the source actor. When the source has fewer
than eight items, selecting oneself performs the literal remove-then-append:
the item moves to the first raw hole as unequipped and the source is
recomputed. Production now preserves this otherwise surprising branch for
both shop and church callers. `ValidateNativeInventoryProjection` requires
the compact inventory/equipped arrays to match signed raw-cell order before
mutation; destination-full checks raw flags rather than item bytes or
`len(Inventory)`. The shop owner returns item/destination cancel, empty/full
feedback, and success to the FDTXT512 source loop, while source-roster cancel
returns to the four-service menu.

The shared product/source-item renderer at `0x2dc55` is likewise split by its
actual mode byte rather than caller name. Mode zero displays item row `+0x13`
full price and is used by purchase. Nonzero mode displays
`trunc(3*price/4)` and is used by sell/transfer lists. The strict
`RenderNativeFacilityItemRows` preserves this distinction while the legacy
transfer wrapper remains mode-one compatibility; this does not yet close the
parent panel opening/closing lifecycle.

The purchase confirmation lifecycle is separately closed at E1; the preceding
ch02 trace additionally closes its steady Yes/No pulse and horizontal input
at E2, but not every opening/closing frame. The four
six-variant FDTXT tables copied by `0x2f0c5..0x2f0f4` are:
question `{1,502,1,439,1,439}`, insufficient gold
`{1,504,1,438,1,438}`, no eligible recipient
`{1,505,1,437,1,437}`, and optional equip
`{1,507,1,507,1,507}`. The question expands `FFFC` with
`FDTXT[itemID+181]`, expands `FFFA` with the decimal price, and then calls the
native `0x19953` Yes/No owner. The insufficient-gold branch does **not**
rebuild the dialogue overlay. Instruction-order revalidation corrects the
earlier framebuffer assertion: `0x2f2a9` first calls `0x197e5`, presents
all four choice-closing frames, and `0x19913..0x1994c` restores the saved
310×86 question region before return. Only then does `0x2f2d3` append
feedback at literal VGA `0xac44c`, framebuffer coordinate `(12,157)`;
`0x16c57(1)` adds the mode-one wait marker. The strict compositor therefore
must receive pixels equivalent to the restored question, not the fourth
inward frame, a steady confirmation, or a fresh dialogue. The current ch02
screenshot path reaches that target by deterministic recomposition; this
E2 does not prove generic saved-buffer ownership. Deterministic indexed
fixtures cover both states, while the ch02 gold0 stable frame additionally
has exact DOSBox E2. Production now owns the
product-list close, four-frame confirmation open/steady/close, bounded
Yes/No input, cancel return, insufficient wait, five-frame dialogue close,
and product-list reopen. Both recipient selectors and recipient-full feedback
are now production-owned as described below; successful insert/equip
animation and debit timing remain fail-closed.

Recipient selection has two distinct native owners and must not be normalized
into one roster widget. After the gold check, item type `>=0x20` sets the
native party count and enters `0x2e6b8`, the proven two-column/six-visible
roster. Item type `<0x20` instead passes the compatibility-filtered byte list
to `0x2e8cf→0x2ebe0`, a three-visible equipment comparison panel. The remake
now has a strict consumable-recipient compositor and production owner that accept only
`itemType>=0x20`; attempting to draw equipment recipients through it fails
closed. Its stateful cursor preserves the proven two-column six-visible
window, bounded `±1/±2` movement, six-frame open, five-frame close, and
cancel-to-product-loop. Its deterministic fixture uses the shop's own entry 16 panel,
FDICON map sprites, FDTXT identity names, and the recovered selected text
color.

If the selected recipient has exactly eight occupied native cells,
`0x2f36d` expands `word_5265f[hubVariant] =
{1,506,1,506,506,506}` with `unit[+7]+1`, opens the dialogue, waits in mode
one, and closes back to the product loop without inserting or debiting.
The purchase-specific compositor now preserves that variant/name ABI and has
an original-resource fixture. Production checks the eight raw flag/item cells
after selection, opens this message for a full recipient, waits for input,
then closes and returns without insert or debit.

The `<0x20` comparison renderer is now also executable. `0x2efb7` starts from
raw base AP `+0x37`, DP `+0x39`, and shared DX `+0x3e`, adds candidate item
words `+1/+5/+3/+7`, then retains equipped contributions only from the
opposite `type<=0x14` category. `0x2ebe0` compares those four results with
derived AP/DP/HIT/EV `+0x48/+0x4a/+0x4c/+0x4e`; `0x2ef8f` selects raw digit
bank 31/42/119 for equal/increase/decrease. Three rows are visible at
`y=117+26r`, with FDICON, selected FDTXT name, and four
current→candidate fields. The shop-resource offset provenance maps the panel
and labels to entries 16 and 18..22; six `0x1974c` opening frames and five
`0x2d31b` closing frames reuse the proven band schedule. Original-resource
regression and a deterministic indexed fixture cover the stable target.
Production has a runtime implementation of the filtered three-row cursor and
six/five-frame jobs; recipient cursor input and opening/closing timing still
have no DOSBox E2 evidence and must not be treated as lifecycle parity. A
correctness gate was added during integration: the generic item
panel record omitted native base AP/DP `+0x37/+0x39`, so equipment preview now
uses `NativeShopEquipmentRecordForUnit`, which requires
`EquipmentBaseSet` and writes those fields explicitly. Missing base/raw
provenance fails closed rather than previewing zero. At this point the
recipient targets and cancel/full branches have E1 production
implementations; recipient input/scroll 已達 E1，尚缺其 DOSBox E2；
no-recipient/full、success/debit 與各自 E2 仍是下列獨立 gate。

The production input path no longer mutates the equipment-recipient cursor
inline. `advanceNativeShopEquipmentRecipient` is the shared pure transition:
Up/Down are bounded, horizontal input is a no-op, simultaneous Up then Down
preserves the native update order, and `NativeThreeRowWindow` remains the sole
stateful viewport owner. Direct regressions cover the observed
selection0→Down→selection1→Up→selection0 trace, bounds, six-entry window
movement, and helper-level invalid count/selection/start rejection. The
production caller routes that rejection back to the purchase list before
indexing a recipient. This is E1 input-contract coverage only;
scrolling and lifecycle timing still require DOSBox E2.

The first same-state equipment-recipient trace now closes the stable ch02
selection0 frame, but not cursor input or opening/closing timing. Direct
`FD2_CAMP_NODE=shop_ch02_weapon` previously had no persistent party at all.
The screenshot-only `FD2_SHOT_PARTY_BINDING` bridge therefore accepts only a
compiled LOADCH binding with both `PartyScenario` and its recorded `PartyOrder`,
reorders and materializes that typed scenario, initializes equipment bases,
and rejects any unit missing identity, selector, race/class, raw inventory,
byte6, or base-stat provenance. The recipient hook
`FD2_SHOT_SHOP_EQUIPMENT_RECIPIENT_STATE=good,selection,start,cycle,gold`
then admits only an affordable real equipment good, a normalized three-row
window, and FDICON cycle0..2. Neither adapter contains character names.

E2 exposed three errors hidden by the earlier indexed fixture: ch01 authored
party rows omitted the DX projection consumed as `+0x3e`; HIT/EV number/arrow/result columns are each
three pixels right of the AP/DP columns; and every comparison arrow starts at
`y+dy+1`, not the earlier guessed offset. The ch01 DX values 2/2/1/2 are
E2-constrained projections obtained by cross-checking visible HIT/EV against
known equipment rows, not independently dumped raw `+0x3e` values. After adding
those projections and correcting the anchors, original DOSBox and production at
good0, selection0, start0, idle cycle1, gold1000 share raw RGB MD5
`28258fb3ce5bc42eb1c701a7792d193b` with whole-frame error zero. The
[`equipment-recipient sheet`](../figures/shop-equipment-recipient-ch02-original-vs-remake.png)
places original left and remake right. This proves one stable party/state
projection produced by a screenshot-only bridge; it does not prove normal
campaign persistence, native FD2.SAV compatibility, recipient
navigation/scroll, no-recipient/full feedback, or transaction success.

Normal campaign persistence has a separate production boundary. Native JOIN
creates persistent records before LOADCH; the remake previously recorded only
membership/order, while its first native-identity `sync_party` required a
pre-existing typed roster match and could therefore skip every initial member.
LOADCH now seeds only missing records after an established JOIN
membership/order, preserves all existing progression records, and refuses to
manufacture persistence during direct/debug replay. A regression using the
real ch00 scenario/order proves `[0,9,4,30]` reaches the ch02 cloth-equipment
recipient filter as `[0,9,4]`, then remains matchable by the first
native-identity sync. This closes the typed runtime bootstrap defect; it is not
native FD2.SAV compatibility or a full campaign input/E2 trace.

The shared `0x2f4c6` success helper must likewise be split by the caller's
raw hub variant. Shop variant 1/resource 12 applies entries 23..27 at
`(169,45)`, waiting two BIOS ticks per frame, then restores portrait mode 0.
Variant 3/resource 29 waits one tick, applies entry 23 once at `(148,39)`,
waits eight ticks, then restores the portrait. Variant 5/resource 63 applies
entries 23..29 at `(131,28)`, two ticks each, with no `0x16559(0)` restore.
Church variant 4 remains the separate nine-frame/palette case. Strict plans
and real-resource regressions now cover all three shop branches and reject
resource/variant mismatches.

The caller order is also explicit: inventory insert occurs first; equipment
types may ask and apply optional equip/recalculation; only then does
`0x2f4c6` present success, after which `0x2d516(price)` debits gold and control
returns to the product loop. Thus neither confirmation nor insertion alone
authorizes an early debit. The recipient owner deliberately returns
fail-closed if any transaction asset or raw provenance is missing. With all
inputs present, production stages a deep-copied unit, applies `0x1bb8c`
insertion first, and leaves gold unchanged. Consumables proceed directly to
success. Equipment opens FDTXT507 plus `0x19953`; Yes applies
`0x1c142→0x1b750`, while No/Escape preserves the inserted unequipped item.
Instruction revalidation confirms `0x1c142` categorizes replacement by item
ID `<0x80` versus `>=0x80`, writes removed equipped flags to exact zero and
the new raw slot to exact `0x40`; the normalized adapter now synchronizes all
eight raw flags with its compact equipped view.

Only after the staged unit is valid does production publish it and start the
variant-specific success timeline on the caller's bare shop framebuffer:
the purchase/equip dialogue has already closed, so retaining its blue grid or
question text is a renderer defect. Variant 1 presents five frames for two
BIOS ticks each plus a zero-duration portrait restore; variant 3 preserves
one pre-tick, one effect frame for eight post-ticks, then portrait restore;
variant 5 presents seven frames for two ticks each and has no portrait
restore. Gold remains unchanged throughout insertion, optional equip, and
that success presentation.

The former instant `FinalizeGood` callback was not native. Direct
`0x2d516..0x2d620` disassembly shows that the callback first subtracts the
amount from the balance, formats old/new values as eight digits, then animates
every differing digit downward (0 wraps to 9). Each value step uses nine
overlapping opaque 6x9 windows from current FDOTHER entry 2's 6x99 strip at
literal VGA `0xA7A90 = (16,98)`, one row above the stable eight-digit baseline
`(16,99)`, followed by `0x375b2(10)`. Production now starts this
10ms-per-phase odometer only after success completes, commits the new balance
before its first visible phase, and reopens the product list only after the
roll. `1000→950` therefore has 45 source phases; `1234→1134` has nine.
Regression asserts the 10/9/14 BIOS-tick success schedules, debit phase
geometry/direction/wrap, mutation boundary and product-loop return. The
wall-clock renderer may sample over 10ms source phases at a 60Hz display, so
it preserves elapsed timing but does not promise every source phase is
physically presented. DOSBox internal framebuffer capture found 16 atomic
debit samples across early/middle/final phases that each match the
source-built frame at full-screen AE=0. Five other captures interrupted
`0x2d620`'s row-by-row memmove and are partial writes, not atomic presentation
oracles.

2026-07-29 重新追查呼叫者緩衝區，已關閉原本固定AE=2的兩像素差異。
`0x1956b`先配置三份64000-byte緩衝區，把當下VGA完整複製到`[0x53c5f]`，
再由`0x168b6`建立對話格並以`0x4e8e1`加入DATO基底；`0x2d31b`最後用
`0x373c4(...,0xfa00)`完整恢復`[0x53c5f]`。購買呼叫者則在
`0x2f426`建立裝備詢問後才於`0x2f455`呼叫`0x16559(0)`，並在
`0x2f4a1`關框恢復後立即由`0x2f4a6`播放購買成功動畫；直到
`0x2f543`才再次覆蓋DATO第0幀。FDOTHER商店背景在
`(175,90)/(176,90)`的索引色值是190/190，DATO#128第0幀則是96/191；
現行`ComposeNativeShopBareScene`先前提前覆蓋DATO第0幀，正是兩點差異來源。
移除提前覆蓋後，26張DOSBox成功動畫抓圖中25張各自對上來源影格整幀AE=0；
唯一第15張只在效果區`(184,47)/(184,49)`差兩點，下一張同一來源影格即
AE=0，符合熱鍵中斷`0x16886`寫入的非原子畫面。成功動畫與扣款合成切片
均可升E2；正常未修改玩家路徑及其他商店子面板仍維持原有門檻。

The sibling hotel/preparation family is represented by
`fdother.ResolveNativeHotelServiceRoute`: `0x2fc85` loads raw resource `13`,
then selector `0/1/2` maps to `0x2ffa5/0x30012/0x301f4`; selector `3` first
reads the raw preparation input through `0x19953` and then enters `0x197e5`.
This preserves the observed two-call branch without naming the services or
executing the scene.

### Native shop sell production owner (E1/runtime, 2026-07-28)

Instruction-level revalidation of `0x2f642..0x2f87c` fixes the complete sell
state machine. It first enters the shared `0x2e6b8` two-column party roster.
After a recipient is chosen, the caller compacts all raw cells whose flag does
not have bit `0x80`; an empty source displays FDTXT509 with
`unit[+7]+1`, waits in mode one, closes, and returns to the same roster.
Non-empty sources enter `0x2e0bd(mode=1)→0x2df6b`: six visible items use the
shared panel but display `floor(3*row[+0x13]/4)`.

The two six-variant tables copied from `0x5272a/0x52736` are sell question
`{508,508,508,659,508,508}` and empty-source
`{509,509,509,509,509,509}`. The question expands selected item and sale
price, then uses the same four-frame `0x19953/0x197e5` Yes/No lifecycle.
No/Escape closes the dialogue and returns to the party roster, not the item
list. Yes closes the dialogue, presents the same variant-specific `0x2f4c6`
success timeline, then performs `0x2d3ff` credit,
`0x1b8e7` removal, and `0x1b750` recomputation in that order.

Production now owns this full roster→item→confirmation→success→commit loop.
It derives both displayed and committed prices from the tracked native effect
row rather than a second editable map. The commit is preflighted on a deep
copy and published only after the success timeline: gold changes before the
unit snapshot is published, matching the native writer order. The native
removal shifts subsequent raw cells left and marks the tail flag `0x80`;
the high-level `Unit` adapter canonicalizes the semantically ignored stale
tail item byte to `0xff`, so this projection must not be described as a
byte-identical FD2.SAV record dump. Missing eight-cell provenance fails
closed. Roster selection remains at the previous actor after an empty/cancel/
successful branch, matching the lack of a selector reset before `0x2f70a`.

### ch02 賣出子面板 E2 驗收契約（2026-08-22）

本切片不重開已閉合的 `0x2f642..0x2f87c` 語意，而是驗證其正式消費端。
原版 oracle 必須從固定雜湊 `FD2.EXE` 的可丟棄副本啟動；副本只允許套用
已記錄的三處戰鬥略過 patch（`0x117f3→call 0x205be`、
`0x117f8→jmp 0x1187a`、`0x205d5→jmp 0x206c3`），商店 handler、資源、
角色資料與鍵盤輸入不得修改。證據等級標成 route-patched E2，不得宣稱為
未修改的一般玩家戰後路徑。

正常輸入序列必須由 ch02 town selection0 進入 weapon shop，於 service0
以 Right 到 service1，再以 Enter 進入賣出角色名冊。至少保存下列原子狀態：

1. `sell_roster` selection0 的穩定整幀，以及一次普通方向鍵後的 selection1；
2. 在 selection0 以 Enter 進入 `sell_items`，保存第一個物品的穩定整幀；
3. 每張原版圖都必須與正式重製 owner 的同一 selection、window origin、
   icon／portrait 相位做完整 320×200 RGB 比較，不可遮罩差異；
4. screenshot-only bootstrap 若用於固定重製狀態，必須要求完整 typed/raw
   party provenance，且不能提升 campaign persistence 或 native save 等級；
5. 若任一原版狀態、相位或正式重製 projection 無法同狀態重現，UI-09 維持
   partial，差異須先分類為輸入、生命週期、資源、相位或 renderer，不得用
   固定 sleep、硬編姓名或猜測 ABI 關閉。

驗收結果：route-patched 原版依上述正常商店輸入抵達 `sell_roster` 與
`sell_items`。高頻樣本證實名冊 FDICON 使用三個可見 cycle；受限 adapter 因此
將 cycle 列為明確狀態，而未猜測正常 runtime cadence。以 `(175,90)` 人物區像素
同步後，名冊 selection0/cycle1 與索爾物品 selection0 均和正式重製 compositor
取得完整 320×200 RGB `AE=0`，raw RGB MD5 分別為
`62307f5918f1de723055f951a7e6dc6a`、`f5ff2d3650575a93bcc0d795fae7c4ea`。
證據見 [`shop-sell-child-ch02-e2.json`](../data/ui-traces/shop-sell-child-ch02-e2.json)。
這只關閉兩個 stable child states；selection1、問句、取消／成功生命週期與未修改
campaign/native save 路徑仍另列 gate。

下一個賣出 E2 切片沿用同一固定雜湊與 route-patched 限制，但必須由
`sell_roster` selection0 送普通 Right 抵達 selection1，再由 Left 回selection0，
不得以 renderer hook 取代輸入證據。之後以 Enter 選索爾、Enter 選第一件物品，
保存 `sell_confirm` 的 Yes selected 與一次 Right 後 No selected；兩個選項必須
分別與正式 `ComposeNativeShopSellQuestionBase`＋choice compositor 做同 pulse、
同人物相位的整幀比較。受限 adapter 只能選既有 raw inventory 中的 active item，
顯示價與提交價都必須來自已追蹤 effect row 的 75% 計算；缺 raw slot／flag、
非法 unit／item／choice／pulse 或 compositor 失敗時須原子回復。這一批只驗證
問句與選項，不得在尚未擷取 success timeline 前宣稱賣出提交演出已達 E2。

驗收結果：原版由 selection0 普通 Right 抵達悠妮，再 Left 回索爾；selection1
cycle1 與 production raw RGB MD5 同為 `d63d213c835f59c1a60428ef6a14d7ad`。
短劍問句顯示 37 元，Yes selected 與 Right 後的 No selected 分別在同人物相位、
pulse2 對上 production，raw RGB MD5 為
`38bd7527570c0ddbf819f19eeea71685`、
`6168c00b515ffe33e14a658bd7932d42`，三組完整 320×200 `AE=0`。
證據見 [`shop-sell-selection-confirm-ch02-e2.json`](../data/ui-traces/shop-sell-selection-confirm-ch02-e2.json)。
這關閉名冊 selection0↔1 與賣出 Yes／No stable/input E2；success timeline、
commit 後返回名冊及未修改 campaign/native save 仍未由本切片證明。

### ch02 賣出成功與返回名冊 E2 契約（2026-08-22）

本切片仍只驗證已閉合的 `0x2f642..0x2f87c` 正式消費端，不重新解釋
handler。原版沿用上一節固定雜湊與三處戰鬥略過 patch，並由同一正常商店
輸入走到短劍 Yes selected；按下普通 Enter 後，必須密集擷取下列連續邊界：

1. 四幀 Yes／No 收合與五幀對話收合之後，第一個 `0x2f4c6` 成功演出畫面；
2. variant1 成功演出的每個可見原子影格及最後一幀，並記錄相鄰樣本是否只是
   同一影格的停留，不把取樣頻率誤認成原版 present 次數；
3. `0x2d3ff` 金幣增加前後的第一個可見畫面，確認成功演出完成前金幣不變；
4. `0x1b8e7→0x1b750` 提交後返回 `sell_roster` 的第一個穩定畫面，確認角色
   selection 沒有被重設，並以重新進入該角色物品清單或正式重製狀態測試證明
   已售 slot 不再存在；畫面本身看不到的背包 mutation 不可只靠截圖宣稱；
5. 所有原版畫面只與正式 `nativeShopSuccessTimeline`／callback／
   `returnToNativeShopSellRoster` 所產生的同狀態比較。受限截圖入口只能選取這條
   production timeline 的既有 step 或完成 callback 後的 roster，不得另造成功
   影格、提前發布 staged unit／gold，或繞過 raw inventory provenance。

證據等級仍是 route-patched E2。若取樣只能證明成功動畫的共用原子影格，卻無法
固定 credit／commit／return 的相位，則只提升已直接對拍的子狀態；重製端的
mutation 與 JSON save/load 回歸維持 `RUNTIME-E1`，不得合併宣稱一般玩家完整 E2。

動態擷取首先推翻了「success callback 可以直接跳到新金額」的重製端近似：原版
短劍交易從金幣0開始，成功演出後可見 `0→11→33→34→35→37` 的
向上滾動，再返回索爾名冊；重新 Enter 後短劍已消失，只剩皮甲與藥草。合法
IDA Pro 9.4 隨後固定 `0x2d3ff`：它在畫第一個 6×9 window 前先增加全域金幣，
每個不同 digit 以 phase1開始，九個10 ms phase後才把舊 digit加一並十進位
wrap；共用 `0x2d620` 只負責九列6像素 blit。這與 `0x2d516` 的先減值、
phase9→1向下排程不是同一方向。直接證據見
[`fd2_shop_gold_credit_ida.txt`](../data/ida/fd2_shop_gold_credit_ida.txt)。

因此正式賣出擁有者現在分成三個不可交換的發布邊界：成功 timeline 只持有 staged
unit／gold；第一個 callback 先發布新 gold，啟動具型別 credit timeline，但仍不
發布背包；credit 最後一幀呈現後，第二個 callback 才發布 `0x1b8e7→0x1b750`
結果並返回原 actor selection。缺 success／credit任一資源或 raw inventory
provenance 時整筆交易失敗即關閉，不允許直接跳過可見滾動。

### 商店裝備收件者正常輸入 E1 契約（2026-08-22）

既有 `0x2F0B0` 購買 caller、三列收件者 compositor、滿欄／無合適角色訊息與
交易 owner 已有直接證據及原始資源回歸；本切片不重開這些位址，而是補正式
輸入到既有 owner 的消費鏈。正式 `Game.Update` 必須把收件者狀態同一拍的
Enter、Escape、Left、Right、Up、Down 收斂成一份不可變輸入值，再交給正式
收件者 consumer；測試亦只能呼叫這個 production consumer，不得直接改
`nativeShopMode` 冒充收件者輸入。非收件者 mode 必須由它拒絕，不能搶走其他 owner。

對至少六名具完整 native identity／selector／class／八格背包 provenance，且皆
符合商品裝備類別的可編輯 typed party，必須驗證：

1. `menu(service0)→purchase→confirm Yes→recipient_equipment` 只在四幀服務收合、
   六幀清單收合、四幀選項收合與五幀對話收合均經 Draw 確認後依序前進；工作
   存在時新輸入不得穿透；
2. Down 由 selection0 走到3時，三列 window 的 start 必須由0變1；再 Down 到5
   時 start為3，Up則依 stateful window 回退。Left／Right在裝備收件者不改選擇；
3. 選到八格皆 occupied 的角色後按 Enter，必須先播收件者五幀收合，再進
   `recipient_full` 六幀訊息；Enter／Escape 都以五幀收合回 purchase，不得扣金、
   改背包或留下 pending transaction；
4. 若所有角色都不符合該裝備類別，Yes 後必須進 `no_recipient` 六幀訊息；
   Enter／Escape 收合回 purchase，金額與隊伍維持原子不變；
5. 非滿欄角色沿同一路徑進 `equip_confirm`；接受後才播放正式 success／debit，
   發布 typed／raw 背包、裝備能力與金額。取消則不發布；
6. recipient Escape→purchase、purchase Escape→menu、menu Escape→town 的每個
   closing job 都必須完成才轉態，並清除未序列化商店暫態。任何 raw projection、
   原始資產或清單索引不足時失敗即關閉。

這只可提升正式收件者 input consumer 為 `RUNTIME-E1`。測試沒有注入作業系統
鍵盤事件，也沒有從完整 campaign 抵達商店；目前亦沒有可重播的未修改原版四人
以上合格收件者存檔。synthetic typed party、route-patched EXE 或截圖入口均不能
證明 recipient scroll／full／no-recipient 的一般玩家 `PLAYER-E2`。

### ch02 獨立裝備子面板 E2 契約（2026-08-22）

本切片沿用已閉合的 `0x2F883→0x1BFFE→0x17E0B`，不重新解釋 service2。
原版只可使用固定雜湊 `FD2.EXE` 的可丟棄副本，套用已驗證的三處戰鬥略過
route patch；標題、城鎮、武器店、service選擇、角色選擇與item panel仍使用普通
鍵盤及未修改商店程式／資源／party。證據等級因此最多是 route-patched E2。

第一批同狀態至少保存：

1. `town ch02→weapon shop→Right×2→service2 Enter` 後的角色名冊
   selection0／window start0穩定畫面；
2. 名冊按Enter選索爾，原版11→0展開完成後的完整item/status panel，item
   selection0為短劍；畫面需保留原始LV、HP／MP、DX／MV、HIT／AP、EV／DP、
   三項背包、裝備旗標與效果值，不得只比較名冊區；
3. 若再驗收mutation，必須由普通Enter選擇相容item後，比較同面板原地重建及
   Escape 0→11關閉／restore／返回原actor名冊；第一批stable state不能外推這些
   尚未逐幀對拍的邊界。

重製截圖入口只可接受 `mode,unit,item,start,gold`：它必須先以受限
`FD2_SHOT_PARTY_BINDING`建立完整typed/raw party，再呼叫production
`setupNativeShopEquipRoster`及`openNativeShopEquipPanel`；panel只能選該角色真實
occupied raw slot，並固定為production展開完成的step11。禁止注入姓名、record、
item ID、能力、畫面或動畫影格。任何window、inventory projection、原始資產或
effect row不足時整個入口失敗且回復原狀態。

原版與重製必須以320×200 raw RGB整幀比較並保存雜湊、AE、輸入時間線與限制。
即使兩個stable frame達AE=0，也只關閉service2名冊／panel stable E2；mutation、
restore、完整campaign／原版存檔及service3 transfer仍分開驗收。

2026-08-22 實測結果為 `PLAYER-E2 route-patched partial`，不能升格為上述完整
stable E2。正常標題／城鎮／商店／service2／角色輸入已取得名冊與索爾面板；
姓名、職業、金額、能力、三項物品、裝備效果與幾何均一致，但320×200整幀仍分別
有 `AE=1389` 與 `AE=1433`。名冊差異集中在四名角色的動畫精靈；面板差異包含
呈現相位與面板邊緣。重跑相位樣本時可拋棄沙箱已無法抵達同一城鎮簽章，因此
本輪停止，不猜測 sprite／panel renderer。證據與限制見
[`shop-standalone-equip-ch02-e2.json`](../data/ui-traces/shop-standalone-equip-ch02-e2.json)
及[`對照圖`](../figures/shop-standalone-equip-ch02-original-vs-remake.png)。

#### service2 裝備交易原子發布契約（2026-08-22）

`applyNativeShopEquipSelection` 不可先寫 `partyRoster` 再嘗試重建 item/status
panel。正式順序必須是：複製來源 unit→驗證 raw／compact 投影→在私有 unit 執行
`EquipNativeCompactSlot`／`0x1B750` 重算→以候選 unit 完整重建 panel buffers 與
Ebiten image→最後一次發布候選 unit。panel 所需 FDOTHER／FDTXT／DATO、effect
rows、palette 或任何 renderer 前置不足時，unit、背包旗標、能力、selection 與既有
panel buffers 必須全部不變。

成功後 item panel 留在 step11 並原地顯示新裝備與能力；只有玩家 Escape 才開始
0→11收合，完成後再以既有六幀 owner 返回同一角色名冊。這是正式 runtime 與重製
JSON 存檔的 E1 契約，不自行提升為原版 mutation／restore E2。

重製端已依此改為候選發布：測試刻意在相容物品、raw projection 與重算皆成功後
移除最終 palette，使 panel 建構於最後一步失敗；正式 owner 回傳失敗，來源 unit、
compact/raw 背包、裝備旗標、能力、item selection 與既有 panel image／buffers 全部
保持不變。恢復 palette 後同一輸入才一次發布新裝備與 AP，既有 0→11 收合、同角色
名冊六幀重開與 town JSON round-trip 回歸仍通過，裁決為 `RUNTIME-E1`。

本輪嘗試補原版 mutation E2 時，固定雜湊 route-patched 副本由標題 CONTINUE 載入
目前 `FD2.SAV／TMP` 後只進入戰場角色名冊／狀態面板；Return 在兩者間循環，Escape
返回標題，無法抵達先前 ch02 城鎮簽章。依有界停止原則不再重播或猜改快照，因此
原版 mutation／restore E2 仍未關閉，既有兩張 stable partial E2 也不受影響。

### ch02 物品轉移子面板 E2 契約（2026-08-22）

本切片只消費已閉合的共用 owner `0x2F8EA`，不重解函式本體。原版只可
使用固定雜湊 `FD2.EXE` 的可拋棄 route-patched 副本；戰鬥略過之後，標題、
城鎮、武器店、service3、來源角色、物品與目的角色仍由普通鍵盤輸入，
不得以修存檔或座標注入取代。證據最高只能為 route-patched E2。

第一批同狀態至少保存：

1. service3 Enter 後 FDTXT512「誰的東西呢？」的完整展開穩定幀；
2. Enter 後來源名冊 selection0／window start0；
3. 選索爾後，`0x2DC55(mode=1)` 物品清單 selection0／start0，且商店金額不變；
4. 選短劍後 FDTXT510「要給誰呢？」穩定幀；
5. Enter 後目的名冊 selection0／start0，來源索爾仍在全 party 名冊內。

重製截圖入口只可接受
`mode,source,item,selection,start,gold`，其中 `mode` 只能為
`intro|source|items|dest_prompt|dest`。入口必須先以
`FD2_SHOT_PARTY_BINDING` 建立完整 typed/raw party，再依序呼叫 production
`setupNativeShopTransfer`、來源 roster、物品清單與目的 roster owner。`source`與
`selection` 都是 `partyJoinOrder` 中的索引；`item` 只能選取該來源依 raw signed
flag 投影後的實際 compact item。禁止注入姓名、record、item ID、背包、
畫面或名冊像素。非法 window、缺 raw provenance、缺資源或合成器拒絕時，
所有公開狀態必須原子不變。

原版與重製以 320×200 raw RGB 整幀比較，保存輸入時間線、雜湊、AE 與限制。
第一批穩定幀不能外推 transfer mutation、self-transfer、empty/full feedback、
cancel/restore、原版 `FD2.SAV`、church caller 或未修改完整 campaign E2。

實測結果保存於
[`shop-transfer-ch02-e2.json`](../data/ui-traces/shop-transfer-ch02-e2.json)及
[`對照圖`](../figures/shop-transfer-ch02-original-vs-remake.png)。五個狀態都由
正常標題／城鎮／商店鍵盤輸入抵達：物品清單只差2像素；兩個提示各差88像素，
差異為翻頁箭頭的動畫相位；來源與目的名冊分別為 `AE=1391`、`AE=321`，差異
集中在角色小圖動畫相位。可見文字、名冊、物品、效果、價格與幾何一致，但
沒有同步原版動畫 tick，故裁決為 `PLAYER-E2 route-patched partial`，不可寫成
整幀相同或未修改一般玩家 E2。

### ch02 物品轉移成功交易 E2 契約（2026-08-22）

本切片沿用上述已閉合的 `0x2F8EA`、`0x1B8E7→0x1BB8C→0x1B750` 與
兩欄 selector，不新增原版語意。原版仍只在固定雜湊的可拋棄 route-patched
副本略過手動戰鬥；進入 service3 後必須用普通鍵盤選索爾、短劍、目的角色悠妮，
並等待正式 closing／opening 完成。成功驗收至少保存：

1. 目的名冊由 selection0 按 Right 到 selection1，再按 Enter；
2. 交易後自動返回 FDTXT512 來源提示，金幣仍為0；
3. 再選索爾時，物品清單不再含短劍；
4. 再選悠妮時，原有兩件物品後追加未裝備短劍；
5. 重製端同一 transaction 經 `leaveShop→town→JSON save/load` 後，來源／目的
   compact inventory、`Equipped`、八格 raw slots／flags、能力、金幣與隊伍拓撲不變。

重製截圖入口必須接受
`mode,source,item,destination,post_source,selection,start,gold`；`mode` 只可為
`success_intro|success_items`。入口先由 `FD2_SHOT_PARTY_BINDING` 建立完整
typed/raw party，在私有候選上依序呼叫 production transfer setup、來源 item
投影、`applyNativeShopTransfer` 與返回 loop；`success_items` 再由交易後的正式來源
roster／item-list owner 選取 `post_source`。禁止直接改背包、flags、能力、姓名、
物品或像素；任一索引、raw projection、資源或 compositor 拒絕時，公開 `Game`
必須原子不變。

此證據只涵蓋一筆 ch02 索爾→悠妮的成功交易與重製 JSON 邊界；不可外推
self-transfer、empty/full、取消／restore、church caller、原版 `FD2.SAV`、其他章節
或未修改完整 campaign E2。動畫相位仍須以實際 AE 誠實記錄，不為追逐 tick 猜改
renderer。

實測證據保存於
[`shop-transfer-success-ch02-e2.json`](../data/ui-traces/shop-transfer-success-ch02-e2.json)
及[`對照圖`](../figures/shop-transfer-success-ch02-original-vs-remake.png)。原版由目的
名冊 selection0 按 Right 選悠妮並 Enter，返回來源提示後再選索爾，只剩皮甲與
藥草；從該物品清單按 Escape 回 loop，再選悠妮，清單為長棍、長袍、未裝備短劍。
目的名冊／返回提示／索爾結果／悠妮結果的整幀差異依序為
`AE=1391／82／2／286`，可見交易資料與幾何一致，剩餘像素是角色、翻頁箭頭或
物品選取脈動相位。正式重製測試另以同一跨角色 transaction 穿越
`leaveShop→town_ch02→JSON save/load`，證實雙方 compact/raw 背包、裝備、能力、
金幣與隊伍順序保存。裁決為 `PLAYER-E2 route-patched partial` 加重製 JSON
`RUNTIME-E1`；仍不可宣稱完整 campaign 或原版存檔 E2。

### ch02 物品轉移目的取消與自我轉移契約（2026-08-22）

ch02 四名實際 party record 都至少有兩件物品，因此本切片不得把任何角色直接
注入成空背包。先關閉兩條由現況普通輸入直接可達的分支：

1. **目的取消**：索爾→短劍→目的提示→目的名冊後按 Escape；必須先播放正式
   五幀 roster closing 與來源還原，再開六幀 FDTXT512，背包、裝備、能力與金幣
   全部不變。輸入端與測試必須共用同一個取消 owner，不可讓測試直接寫 mode。
2. **自我轉移**：同一路徑在目的名冊直接 Enter 索爾。正式
   `applyNativeShopTransfer(sourceID)` 必須以同一 unit clone 執行 raw
   remove→append，再 `RecomputeEquipment`；短劍成為未裝備尾項，索爾物品順序由
   短劍／皮甲／藥草變成皮甲／藥草／短劍，金幣不變，完成後返回來源 loop。

原版以固定雜湊 route-patched 副本的普通鍵盤分別重播，重製畫面則只可使用
production owner：取消後的提示沿用無 mutation 的正式 compositor；自我轉移後
清單沿用 `FD2_SHOT_SHOP_TRANSFER_SUCCESS_STATE`，且 destination 必須等於 source。
兩條都需保存輸入時間線、320×200 raw RGB、AE 與限制。

這仍不證明 empty/full、跨 caller、原版 `FD2.SAV`、其他章節或未修改完整 campaign。
empty 必須由可追溯的多筆正常交易或合法 raw 前置狀態取得；full 必須有八個非負
raw flags。不得為了截圖直接清空或填滿背包。

重製端實作結果為 `RUNTIME-E1`：目的名冊 Escape 現只由
`beginNativeShopTransferDestinationCancel` 擁有，成功預檢五幀收合後才發布工作，
收合完成才呼叫既有來源 loop 建立六幀 FDTXT512；預檢失敗不再直接跳過動畫。
原始資產整合回歸證實取消前後角色、背包、裝備、能力與金幣完全不變，且錯誤
mode 不能啟動此 owner。取消返回後再沿正式 source→items→destination owner 選
來源本人，既有 raw remove→append／重算仍產生未裝備尾項。這批未新增 DOSBox
畫面，因此 self／destination-cancel 仍只到重製端 E1；上列原版同狀態 E2 gate
與 empty／full gate 均保留。

### 標題 LOAD 到原版章節槽的正式確認契約（2026-08-22）

標題四槽 selector 的 Enter／Space 必須只呼叫一個正式確認 owner。該 owner 先以
`nativeLoadSlotConfirmable` 驗證 envelope／checksum／空槽，再依來源分派原版
`FD2.SAV` 或重製 JSON；只有完整還原成功才離開 `loadslots`。空槽、竄改 envelope、
不支援章節、roster／identity 分歧或戰間節點進入失敗時，必須留在四槽畫面並顯示
錯誤，不得把標題 phase 誤設為遊戲中。

原版槽成功的重製端回歸至少要從 selector owner 消費 checksum-valid fixture，驗證
正確 `town/preparation` node、chapter、gold、typed/raw party、join order、HUD gate，
並清除舊 battle state／dialogue／selection。竄改 fixture 則要驗證 phase、campaign、
gold、party、battle state 與 handler chapter 全部原子不變。合成有效槽只證明
`RUNTIME-E1`；未修改原版由酒店／整備寫出的有效 `FD2.SAV` 與正常玩家 LOAD
仍是獨立 E2 gate。

重製端結果已達上述 `RUNTIME-E1`：`confirmTitleLoadSlot` 現是 Enter／Space 共用
owner；checksum-valid 合成槽由 `FD2_NATIVE_SAVE` 經正式 selector 分派到
`town_ch02`，一次發布悠妮 typed/raw record、join order、789 金幣、chapter1 與
HUD gate，並清除舊 battle state／selection。竄改 envelope 則留在 `loadslots`，
campaign、gold、party、battle state 與 handler chapter 均不變，且不再把 checksum
失敗誤報為一般空槽。這仍不是未修改原版有效槽 E2。

The `0x318ad` cap gate is now explicit in
`fdother.NativePreparationPartyLimit`: raw global `[0x53c03] <= 0x1a` yields
15, while values greater than `0x1a` yield 19. The adapter accepts a native
index rather than a human-facing chapter number, preventing a chapter-label
conversion from silently changing the original boundary.

The outer owner at `0x2d093` is essential to that interpretation. Town option
2 first presents FDTXT index `0x201` (“要進入戰場嗎？”); cancellation returns
from the facility.
After acceptance, native chapter indices below `0x1b` call `0x318ad` only when
`[0x53bfb] > 0x10`, while later indices call it only when
`[0x53bfb] > 0x14`. Because `0x318ad` renders `[0x53bfb]-1` selectable records,
these are exactly the 15/19 selectable-member boundaries. Smaller rosters skip
the deployment selector entirely. When selection is required, `0x318c7`
zeroes all 30 flags; the old remake policy that preselected the first 15/19
members was therefore wrong. A zero return from `0x318ad` cancels the facility;
only its accepted final confirmation departs. The editable campaign node now
stores an explicit cancellation target for town-backed preparation nodes.

IDA 在 `sub_320FC` 的直接指令進一步修正部署模型：目的索引從1開始，selection
byte `i` 讀取 persistent record `i+1`，目的 record0 從未被覆寫。因此 quota
不包含固定 record0：一般戰鬥上場 `1+15=16` 人，後期路徑則為 `1+19=20` 人。
舊重製錯把 record0 放入 `prepIDs`，消耗一個 quota，最多只能建立15／19筆戰場
玩家記錄。現在 `partyDeploy` 只保存可選 records，建立戰鬥時再由
`battlePartyMembers` 加入固定的 `partyJoinOrder[0]`。證據見
[`fd2_preparation_fixed_record_ida.txt`](../data/ida/fd2_preparation_fixed_record_ida.txt)。

The preparation-only branch at `0x2cad7` is different and must not share the
town prompt. A nonzero `0x526b9[index]` displays FDTXT index `0x19a`
(“要記錄戰況嗎？”); accepting invokes `0x30012(0)`, while declining skips the
save call. Both arms then enter `0x318ad`, and a zero return loops back into a
fresh selection pass. The editable nodes therefore use the battle-entry prompt
plus a town cancellation target only for town-backed routes; preparation-only
routes retain the save prompt and no town target.

The first full `0x31e80` trace narrows the neighboring UI contract: it reads
the caller-owned 30-byte selection table (`[selection+slot]`), counts selected
entries through `0x320ce`, and chooses the selected/unselected indexed blit
branch (`0x4deda` versus `0x4de56`) for each roster row. This body shows no
write to the selection table or persistent roster; it is a preview/presentation
consumer, not the Enter/toggle mutation primitive. The remake must therefore
keep `partyDeploy` mutation separate from this raw renderer boundary.

The preparation input wait loop is a separate raw boundary at `0x32004`,
called by `0x31a29`. It polls `0x10620`; when the DOS key word changes it
redraws through `0x31e80`, otherwise it reads the two-byte record at
`0x53a8d/0x53a8e` via `0x36d98`. The verified return-byte branches are
extended `0xe0/0x52` to `0x1c`, `[0x53a8d]==0x20` to `0x1c`, and—only
when neither earlier branch applies—`[0x53a8e]==0x53` to `1`, with the
helper's seeded default `0x10` otherwise.
The caller treats return `1` and `0x1c` as raw branch values before invoking
`0x320ce` and `0x320fc`; the remake captures only this byte contract in
`NormalizeNativePreparationKey` and does not assign key names, roster
mutation, or renderer semantics.

### Church service selector input/transition boundary (IDA E0, 2026-07-27)

Official IDA 9.4 decompilation closes the previously missing selector edge:
`0x3072f` calls `0x2d669(0)` to open the church menu, then `0x2d7bd()` to read the
selection. `0x2d7bd` accepts raw scancodes `75` (left) and `77` (right), updates
`[0x53c57]` with four-entry wrap (`0→3`, `3→0`), returns `1` on Enter/Space
(raw `28`/`57`), and returns `-1` on Escape (`1`). It does not use the up/down
bounded list contract used by character selection. After confirmation, the
caller dispatches raw selection `0→0x2ffa5`, `1→0x2f8ea`, `2→0x30dc3`, and
`3→0x31385`; these remain address-level service branches unless their own
callee semantics are independently proven.

`0x2d669` is the indexed church-menu transition: it snapshots a 64000-byte
buffer, clears a 104×20 region at x=201, y=169 with byte `0x4a`, then performs
four direction-dependent cell blits for each of four passes. Official IDA
recheck fixes their provenance as FDOTHER#14 LMI1 entries 3/5/7/9, all 24×20,
at base `(240,169)`. Both copied offset banks at `0x526da` and `0x526ea` are
the signed sequence `[-39,-13,13,39]`; the transition uses divisor `4-j` while
opening (`a1==0`) and `j+1` while closing. Each pass restores the cleared
snapshot, blits with transparent `0x4e9e4` at stride `0x140`, then copies the
frame to VGA. Contrary to the earlier direction assertion, only closing
(`a1!=0`) restores the cleared snapshot after its fourth presented pass;
opening leaves its fourth expanded frame visible.

`0x2d85f(0)` owns the steady mode-zero animation. It uses the paired
FDOTHER#14 cells `3/4,5/6,7/8,9/10`; all non-selected entries use the first
cell and the selected entry uses `pair + counter/2`. The two-bit counter starts
at 2, advances modulo four when BIOS low-word delta is at least two (or the
signed word wraps), and `0x2d9fe` redraws only the selected cell. The remake
now preserves the exact clear, four-pass transition, and steady selected-cell
composition as a fail-closed indexed primitive with original-resource
regression. It is not yet a complete church scene: `0x3072f`'s DATO face,
FDTXT585/586, entry1 decorations and chapter-dependent resource state must be
materialized together before this primitive may replace the authored runtime
fallback. No cell is assigned a service label from its position.

The two raw service branches share a second selector contract. `0x2e6b8` is a
roster/list selector used by `0x2ffa5` and `0x2f8ea`: left/right move by one,
up/down move by two, movement is bounded (no wrap), and the visible window
scrolls in two-entry increments once the cursor crosses its six-entry window.
Enter/Space return `1`, while Escape returns `-1`. `0x2df6b` is the same
bounded two-column selector for a caller-supplied list count and calls the
caller renderer after movement. These helpers are input/layout evidence only;
their caller-owned list entries do not establish service names.

`0x2f8ea` then builds a caller-local list by scanning the selected runtime
record's eight inventory cells and retaining cells whose signed flag byte is
non-negative. This includes both `0x40` equipped cells and `0x00` ordinary
cells; only bit-7-set (`0x80`) reserved cells are excluded. It enters a
second `0x2e0bd`/`0x2df6b` list and then a destination roster selector.
FDTXT510/511/512 are respectively「要給誰呢？」、「沒東西了！」、
「誰的東西呢？」; together with the writer below this closes the service as
item transfer. A previous document revision incorrectly attached `0x2f4c6`
feedback and `0x2d516` amount handling to this caller; the complete `0x2f8ea`
body calls neither function and performs no gold mutation, so that assertion
is deleted.

The writer is now closed at raw level: `0x1bb8c(destination,item)` scans the
destination's eight two-byte cells, finds the first cell whose flag byte is
negative, writes flag `0` and the supplied item byte, and returns `1`; a full
destination returns `-1` without mutation. In the `0x2f8ea` topology the
source cell is removed by `0x1b8e7` before this insertion, so the proven
operation is a source-to-destination inventory transfer with an unequipped
destination cell. The item ID and higher-level menu label remain raw; the
remake exposes `TransferNativeInventoryItem` as an atomic adapter but does
not silently wire it to an unnamed church menu branch.

The remake now exposes this proven transfer topology as an explicit church
mechanics slice: `transfer_source` → `transfer_item` → `transfer_dest`, with
bounded two-column cursor movement and atomic source/destination update. Its
source eligibility uses constructor-derived raw flags when `inventory_slots`
provenance is available; legacy JSON without that provenance retains a
conservative projection and is not native parity. Malformed or missing raw
provenance remains fail-closed for the native gate.

### Church revive service boundary (IDA E0, 2026-07-28)

`0x30dc3` rebuilds its candidate array through `0x309ff`; the predicate is
exactly `0x3453e(index) == 1`, i.e. roster record byte `+5` bit 0. HP,
`OnField`, and other normalized projections are not substitutes when raw
provenance is absent. `0x30a47` renders at most three rows selected by the
stateful `0x30c22` viewport. Each row contains the map sprite, FDTXT
`identity+1`, race `+140`, raw class `+150`, currency cell 15, and a five-digit
fee. The fee is `word_52669[record+0x20] * record+0x21`; no minimum level is
invented.

The dialogue branch is not the same lifecycle as class change. FDTXT590 uses
FFFC for the selected name, FFFA for the decimal fee, and FFFE for a 19-pixel
soft newline. After confirmation `0x197e5` closes only the YES/NO cells in four
frames. On insufficient gold, the caller writes FDTXT504 at VGA `(12,157)` in
the still-open question box, calls `0x16c57(1)` to wait with its marker, then
calls `0x2d31b` for the five-frame dialogue close. With no candidates it opens
FDTXT588, waits through `0x16c57(0)`, closes, and returns. The earlier
class-derived assertion that every revive confirmation performs an immediate
four-plus-five-frame close is deleted.

The remake implements the candidate list, dynamic confirmation, no-candidate,
and insufficient-gold indexed lifecycles with original-resource regression.
On success the hub selector is still 4, so `0x2f4c6` deterministically takes
case 4. It sequentially transparent-blits FDOTHER#14 entries 23 through 31 at
literal VGA `(147,32)`, waiting two BIOS ticks after each; it does not restore
the background between frames. It then applies baseline-derived DAC deltas
`0,2,...,62` and `62,60,...,0`, each with a 4ms delay. The intervening
`0x17aa9(10)` and trailing `0x17aa9(5)` wait against the helper's previous BIOS
latch, so their remaining duration is the latch interval minus the preceding
32×4ms DAC loop—not an additional ten/five-tick hold. Finally DATO mode 0 is
restored at `(118,4)`.

This pixel/palette sequence is now a monotonic indexed runtime timeline; its
zero-duration final portrait restore must be presented before control returns.
The two surrounding calls are `sub_25977(17,1)` and
`sub_25977(11,1)`. A subsequent instruction-level audit deletes the incorrect
PCM/SFX interpretation: `0x25977(track, loop_count)` loads the track-indexed
entry directly from FDMUS.DAT and passes its second argument to
`AIL_set_sequence_loop_count`. The revive branch therefore starts FDMUS track
17 once immediately before `0x2f4c6`, then starts track 11 once immediately
after the final portrait restore. `playBGMCount` now preserves both one-shot
transitions; it does not route them through the unrelated FDOTHER#31 UI SFX
bank.

The indexed runtime now follows the caller lifecycle rather than the earlier
authored stack. Source and destination use the shared six-entry roster. The
middle `0x2dc55(mode=1)` panel uses FDOTHER#14 entry16 and raw cell15; each
visible item uses columns at x=`10/158`, rows y=`119+26r`, FDTXT `181+itemID`,
category cells59/60/61, stat cells64..67 (or frame41), and a five-digit
`(3*word[itemRow+19])>>2` field with digit base119. Its viewport is stateful
and even, with six opening and five closing frames plus source restore.
Escaping either the item or destination selector returns to the source roster;
success returns directly to the source loop. A full destination does not return
silently: after the destination roster closes, `0x2f8ea` stores
`destination[+8]+1` as the dynamic name, selects FDTXT506 through
`word_5265f[hubSelector=4]`, opens the dialogue in six frames, expands its
leading FFFC with that name, waits through `0x16c57(1)`, closes in five frames,
then returns to the source roster. The runtime now preserves this lifecycle and
uses the exact eight raw inventory flags; missing flags or native identity fail
closed rather than projecting a normalized full state. The deterministic
original-resource oracle is
[`native-transfer-full-indexed.png`](../figures/native-transfer-full-indexed.png).

### 商店裝備／轉移返回城鎮的 JSON 存讀檔契約（2026-08-22）

本節不新增原版語意，而是把上方已閉合的正式交易擁有者（owner）接到重製自有 JSON
節點邊界。原版獨立裝備由 `0x2F883` 進入角色／物品面板，成功時以
`0x1C142→0x1B750` 更新八格裝備旗標與衍生能力；共用轉移 owner `0x2F8EA`
依 `0x1B8E7(source)→0x1BB8C(destination,item)→0x1B750(source)` 完成交易，
不寫金幣。主證據仍是本節前文與
[`fd2_runtime_equipment_recalc_1b750_ida.txt`](../data/fd2_runtime_equipment_recalc_1b750_ida.txt)，
不由存檔測試反向宣稱新的原版 ABI。

正式重製端契約如下：

1. 測試必須呼叫 `applyNativeShopEquipSelection` 與
   `applyNativeShopTransfer`，不得直接改寫 `partyRoster` 來冒充交易。
2. 成功交易必須經 `leaveShop` 返回 campaign authoring 指定的 town；只有 town
   節點才可寫入重製 JSON。存檔需保存緊密背包投影（compact inventory）、`Equipped`、八格
   `InventorySlots`／`NativeInventoryFlags`、裝備基礎值、重算後能力、名冊成員、
   加入順序、部署、金幣、章節與 campaign cursor。
3. 清空記憶中的持續隊伍後再讀檔，必須重建同一交易結果；裝備與轉移均不得因
   JSON round-trip 改變 raw 八格拓撲或衍生能力。
4. 商店選擇器（selector）、待提交 unit／gold、購買／賣出／裝備／轉移游標與列表、
   索引轉場（indexed transition）工作、暫存 item panel、舊 shop variant 等都不是節點邊界
   狀態。`loadGameFromSlot` 必須在 `enterNode` 前清除；即使讀檔前停在另一個
   商店子畫面，也不可污染還原後 town。
5. 本切片通過時只提升為重製 JSON 的 `RUNTIME-E1`。它不證明原版四槽
   `FD2.SAV` 位元組相容性（byte compatibility）、商店子面板（child panel）的 DOSBox E2，或所有章節
   商店均已完成一般玩家驗收。

### 商店購買／賣出返回城鎮的 JSON 存讀檔契約（2026-08-22）

本契約沿用上方已證實的交易順序。購買必須由 `stageNativeShopPurchase` 先以
`0x1BB8C` 拓撲發布 staged unit，跑完 variant-specific `0x2F4C6` 成功演出後，
才由 `0x2D516` 對應的金幣滾動擁有者提交扣款；賣出必須由
`beginNativeShopSellSuccess` 預檢深複本，跑完同一成功演出後，再依
`0x2D3FF→0x1B8E7→0x1B750` 順序發布金幣與名冊。確認框或成功演出開始都不是
可存檔的交易完成點。

兩條成功路徑各自必須在正式提交 callback 完成後，經 `leaveShop` 返回 town，
再寫入重製 JSON。冷讀檔後，購買需同時保存扣款後金額、插入物品、原始八格
位置與未裝備旗標；賣出需保存加款後金額、左移後緊密背包、尾格 `0x80` 空位
旗標、`0xFF` 正規化 item byte 與重算後能力。測試需使用正式 production owner，
不可用 `ReserveGood`／`SellNativeSlot` 的孤立規則測試代替。此結果仍只屬重製
JSON `RUNTIME-E1`，不提升原版 `FD2.SAV` 或一般玩家商店 E2。

The other raw branch, `0x2ffa5 → 0x17aed`, is a separate boundary. Direct
instruction decoding fixes `0x17aed` as a one-argument function; an apparent
second Hex-Rays argument was a decompiler artifact. Its body
allocates/copies three 64000-byte indexed buffers, calls `0x17e0b` to stage the
selected record, calls `0x16c57(0)` for the input wait, conditionally renders
the command/overlay path through `0x1ceed`, and executes repeated buffer
restore/redraw passes. The body contains no direct persistent roster,
inventory, gold, HP, or class writer. This rejects the previous "ability
service" wording. Data flow now supports the narrower name character
information/status presentation: `0x17e0b(actor)` builds the item/status panel,
then `0x1c269(actor,0)` gates an optional same-actor command/MP overlay rendered
by `0x1ceed(actor,-1,...)`. Instruction decoding confirms all three calls reuse
the single stack actor argument; the apparent Hex-Rays second parameter is an
artifact. The remake wires the caller-owned roster as FDOTHER#14 entry16 at
`(5,112)`, six visible
entries in two columns, FDICON at `(14+132c,117+26r)`, name at
`(40+132c,121+26r)`, selected/unselected palette `201/205`, shadow `76`,
bounded `±1/±2` input, six opening frames, five closing frames, then source
restore. Selecting an actor now runs the complete read-only presentation:
`0x17eef/0x17fc0+0x184c0(actor,-1)` opens in frames `11→0`; the first key wait
either closes immediately when the 40-bit command list is empty, or performs
seven bottom-panel close frames `0→6` followed by seven command-panel open
frames `6→0`. `0x1ceed(actor,-1)` renders FDTXT `441+commandID`, palette205,
FDOTHER#5 raw cell92 and the record `+5` MP cost with digit base42. The second
key wait closes frames `0→11`, restores the church source, and reopens the
roster. This service is presentation-only and performs no mutation; command
effects/targets remain the separate UI-03 execution workstream.

### 5.2 Native campaign loop ordering（E0，IDA 9.4）

IDA 對 `0x25DE5` 的直接控制流固定了可編輯戰役圖必須保存的外層順序。
`sub_25EBB` 完成標題／章節前置流程並回傳 0 後，main 才呼叫
`sub_117E7` 共享戰鬥控制器；
`[0x53ECC]==1` 時固定呼叫 `0x22E5C`，清除 pending 後繼續。
`0x22E5C` 的函式體只證實它載入 `FDOTHER.DAT` 資源 #79，做兩次呈現與
固定 tick；函式不讀章節索引。因此舊稱「第 1 章專屬世界地圖／中場」
缺少直接證據，已撤回。`[0x53ECC]==2` 時先停止 BGM，呼叫章節索引的
戰後處理器表 `funcs_25E23[dword_53C03]`，之後才呼叫 `sub_2CAD7()`。
若 `sub_2CAD7()` 回傳非零，迴圈走終止／返回路徑；只有回傳零時才呼叫
第二張章節索引表 `funcs_25E3A[dword_53C03]`，選擇
`byte_51E63[dword_53C03]` 作為後續 BGM，清除 pending 並恢復 driver。
表格各 entry 與 `0x2CAD7` 的玩家可見選單名稱仍是獨立證據工作，但此順序
已足以拒絕任何泛化的 `battle → next battle` 捷徑。重製轉場必須在下一戰
節點前保留明確的戰後處理器／選單 gate，即使高階節點名稱仍未閉合。

## 6. Reverse-engineering re-audit workstreams

### 6.0 Runtime/UI boundary（2026-07-26 audit）

現有 Ebiten runtime 已具地圖、游標、單位 HUD、四向 action shell、legacy spell list、dialog、town/shop/
church/preparation 與 save/load；這是 E1 playable shell，不是「UI 尚未存在」，也不是 original renderer。
`57-ui-evidence-matrix.md` 將它分成 UI-01…UI-12：其中 native command grid 的 layout/input/raw mask 有 E0
slice，但 item use、native target/effect、indexed transition、`unit_present`、four-slot native save UI 及大部分
DOSBox pixel differential 仍未閉合。所有文件提到 legacy `CastArea`、ring 或 `spell.json` 時，均只可稱
normalized/editable approximation，不能用來提升 native command 的完成度。

SDD 通過後按以下順序重審，不先補 renderer 猜測：

1. **Boot/menu/UI dispatch**：以 Ghidra/IDA 建立 call graph、keyboard scan、menu item table、resource loader；Docker Capstone 只作可重跑交叉驗證。
2. **Resource provenance**：把 FDOTHER/FDTXT/DATO/FIGANI/TAI/FDFIELD 的 loader、entry、palette、stride、clip 寫成 machine-readable bindings，並與 UI contract 對應。
   `0x22253` 會載入 FDOTHER immediate `0x51`（十進位 **81**）的 nested `LLLLLL` entry（outer 18710 bytes、directory first-word `0x12`；nested payload #1 為 9782 bytes），但完整 stack-slot trace 顯示此 local pointer 不傳入 `0x22470`／`0x22547`／`0x22656`，尾端只 free；它是 resource lifetime，**不是** pixel/frame source。`0x11eee` 是背景／tile redraw；boot 載入 FDOTHER #3 到 `0x53a6d`。FDOTHER #6 是 230-entry `LMI1` bank：`0x22470` 先以 entries `0x72..0x7c` 做 **11** 次 LMI present/tick（#0x72=12×21，#0x73..0x7b=20×22，+0x1f6=#0x7c=24×23）；`0x22547` 再倒序 #3 entries5→0 做 **6** 次 10ms remap present＋2 ticks；最後 `0x22656` 以 #3 entries0→9 做 **10** 次 remap present/tick，合計 27 次 present。其共用 compositor `0x22046` 有六個靜態 caller，並非只屬於 unit presentation：它兩次呼 `0x219ad`，後者以 `sqrt(radius²-dy²)*scale/10` 的 scanline span 作 in-place LUT remap；接著自身對第二個矩形範圍做同一 LUT remap。重新映射六個參數也更正舊斷言：unit-present 的 radius 固定11、scale固定16；`trunc((24*[0x53abd]+15)/5)*LUTIndex` 是 first-radial/final-rectangle **startY**，不是 radius。second radial從centerY開始，final rectangle水平半徑17。`NativeUnitPresentLUTPass/Frames`已保存完整6+10 geometry；`RunNativeUnitPresentLUTFrame`並固定每frame先restore完整`0x25680` snapshot，再執行first radial→mandatory object redraw→second radial→rectangle→present，禁止錯誤累積LUT。`indexedmap`現另有exact terrain-only snapshot、object-only redraw、312×192 viewport copy，以及atomic intro/LUT frame composers。snapshot ownership已閉合為同一allocation在`0x22547`由terrain-only轉成terrain+final-LMI，contract/release共用；不再列為未知 blocker。剩餘Ebiten blocker是從目前Game狀態一致提供原版`unit+3/+4` pose/motion、selector globals/BIOS-tick call timing與中間strip-copy bridge；缺任一仍不可用normalized PNG/Dir猜值。先前6-frame schema禁止接runtime。`internal/fdother.ArchiveEntry` 僅驗證 #81 nested raw boundary，不可把它寫成 layout、音訊或 frame table。
   **2026-08-21 現況勘誤**：上段「剩餘 Ebiten blocker」是 2026-07-26
   的歷史狀態。battle-state adapter 現已嚴格取得 `unit+3/+4`、selector、
   visible cursor 與 strip bridge，並先預算全部影格再發布。仍未閉合的是
   `0x33F78` 等 story/focus caller、第29戰 runtime topology 與原版 BIOS timing E2；
   舊六影格 schema 仍禁止接 runtime。

3. **Battle interaction**：追 action menu enable gates、weapon reach、spell inventory/targeting、end-turn 判定、HUD anchor；每一項先找 caller/data flow，再改 Go。

   Renderer boundary addendum: `0x127e0` chooses a camera-relative 24×24 object sprite and writes the current indexed buffer through either `0x4deda` (raw indexed RLE) or `0x4de56` (RLE palette-remap path). `0x127a9` then calls `0x129ec`, which performs further map/object overlay work on that same buffer. `0x129ec` iterates visible runtime units after their sprites, calls `0x12ac6` for the unit cell and its upper neighbour, and during a nonzero `unit+4` movement offset redraws one pose-dependent neighbour. `0x12ac6` only draws field entries whose resolved tile flag has bit 7 set, to `buffer+0x8088+(y-cameraY)*24*456+(x-cameraX)*24`; bit `0x08` adds `2*flip`, and its FDSHAP descriptor lookup is deliberately the offset-table entry `index+1` (`base+0x0a`). Its raw/alternate tile branch depends on the field entry's byte `+3`. The alternate `0x4dd52` branch is now closed as the same 24×24 four-mode RLE decoder with an explicit caller-supplied 256-entry index table, not an unknown visual effect. Loader `0x10937..0x1096f` obtains the image descriptor base `0x53a5d` and flag table `0x53a69` as the selected FDSHAP even/odd resource pair via `0x111ba`. `0x12ac6` selects its FDOTHER #3 LUT through the same `0x51a97[0x53c1f]` terrain phase table as `0x11eee`; that selector is closed. This is evidence for a foreground-terrain occlusion layer, not merely a redraw marker. The full scheduler remains incomplete, so an Ebiten adapter cannot claim native presentation.

   Asset boundary correction: FDSHAP four-mode decoding retains a separate opacity mask, and `export_engine_assets.py` writes RGBA `tileset.png` for the raw `0x4deda` preview path (opaque index 0 remains opaque). This is **not** a universal native compositor: `0x11eee` selects raw iff composition entry byte `+3==0xff`; otherwise `0x4dcc6` maps opaque source indices through a supplied LUT and, critically, maps mode-3 spans from the existing destination pixel through that LUT. The exporter preserves serialized `native_tile_blit_modes` only as FDFIELD provenance. It is not live renderer state: `0x4dbfc` overwrites every runtime byte+3 with `0xff`, then `0x14818` writes remaining-budget bytes for the active target grid and `0x122dc` mode6 clears selected cells. The compositor must not substitute the archive zeroes, alpha, or normalized highlighting for that lifecycle.

   Native terrain-frame contract: for each visible FDFIELD composition cell, `0x11eee` masks the tile ID to 10 bits and reads the selected FDSHAP terrain-control byte. Frame selection is priority-ordered: bit `0x08` adds `2*flip(0x53a40)`; otherwise bit `0x10` adds truncating `0x3c0b/2`; otherwise bit `0x04` adds `flip(0x53a40)`; otherwise it uses the base tile. It then performs the raw/LUT branch above. These are raw flag semantics only—names such as water/fire animation are not inferred.

   `fdicon.NativeTerrainFrameIndex` is the strict pure form of that selector: it accepts only a 10-bit tile and flip 0/1, preserves the native priority and signed toward-zero division, and returns a descriptor index rather than a rendered image.

   `fdicon.Bank.BlitNativeTerrainCell` composes the verified single-cell path: it selects the FDSHAP descriptor index, then uses raw `0x4deda` only for FDFIELD entry byte `+3==0xff` or `BlitLUT` otherwise. Its regression covers both branches and the mode-3 destination remap. It deliberately has no camera loop, LUT-phase selection or `0x129ec` foreground pass.

   `fdicon.Bank.BlitNativeTerrainRegion` now supplies the corresponding pure `0x11eee` row-major visible-cell pass. It accepts raw composition cells, the raw four-byte-per-tile FDSHAP control table, map origin and explicit destination/LUT; it validates map/control bounds before calling the single-cell compositor. `0x11cac` establishes the normal caller ABI as destination `buffer+0x8088`, stride 456, width 13, height 8, camera X/Y, followed by range overlay, unit layer, then foreground overlay. The pure region adapter does not schedule those later passes.

   Native overlay/selection-selector contract: `[0x51a83]` must not be modelled as a `0..5` GUI range enum. Official IDA data xrefs are retained in `docs/data/ida/fd2_51a83_xrefs.txt`. `0x122dc` dispatches values 1..5 to an **ordered** table of `0x126f7(x,y,descriptor)` calls; `fdother.NativeRangeOverlayPlacements` preserves all 1/1/5/13/21 calls respectively, including the value-3 centre descriptor `14` and value-5 repeated coordinates with distinct descriptors. Setup `0x10483` writes zero immediately before `0x11cac(1)`, so zero is an exact transient opening no-op; after the opening presentation, `0x105eb` calls `0x11cac(0)` and `0x1060c` writes the persistent interactive selector one. Value6 is not a draw table: it clears FDFIELD selected-cell byte+3. However, values above6 are valid runtime state: `0x15140/0x153b1/0x1bd14/0x1d188` zero-extend command/item record bytes and add two, and the verified spell table has `range=5/7/9`, producing selectors7/9/11. `0x122dc` draws nothing for these values, while `0x115b6` still passes `selector-1` into `0x14742` target legality. Therefore rendering and target validation share the raw dword but have different domains. `MaterializeNativeMapRangeMode` preserves the observed full `0..0x101` writer range; editable campaign state is restricted to the proven persistent selector one. The old `0..5` bound, persistent-selector-zero claim and doc37's “battle-message index” assertion are withdrawn.

   Steady-frame scheduler boundary: `indexedmap.ComposeFrame` is the first executable owner of the recovered normal order `terrain → range → unit → foreground → HUD → 0x11eb0 copy`. It requires an explicit HUD callback; omitting it fails before mutation, so callers cannot accidentally present a frame that skipped the native HUD position. `ComposeNativeFrame` is the non-approximation form: it binds `NativeFrameInput`'s recovered HUD resources/raw input directly to that position. Both compose into a private 456-stride work clone and only commit work/VGA after all layers and HUD succeed. **2026-07-28 correction:** direct Capstone at `0x11d12..0x11d36` proves width `0x138=312`, height `0xc0=192`, destination `0xA0504`, destination stride `0x140=320`; the copied viewport therefore occupies VGA `(4,4)..(315,195)`, not a 320×192 block at `(0,0)`. The compositor and regression now preserve the four-pixel border.

   `fdicon.NativeForegroundRedrawEligible` plus `NativeForegroundRedrawCells` are the corresponding pure `0x129ec` schedule primitives. A slot must pass the caller-specific raw `record+5 bit0` predicate and the raw `0x1f183` gate: `unit+7==0x1c` passes; other `unit+7` values are excluded for class `0x13` or race `4/5` (these values are deliberately not given visual/gameplay names). This corrects an earlier mistaken use of the word “group”: map sprite group is `unit+2`, not this field. Eligible slots then preserve the exact ordered calls `(x,y)`, `(x,y-1)`, then only for nonzero `unit+4` one neighbour selected by pose: 0→`(x,y+1)`, 1→`(x-1,y)`, 2→`(x,y-2)`, all other values→`(x+1,y)`. The coordinate helper intentionally returns off-map coordinates too, because native `0x12ac6` performs its own visibility/bounds gate. Neither primitive invokes a GUI renderer.

   Unit-present snapshot ownership correction (2026-07-27): `0x22253`
   allocates only one `0x25680` work snapshot. It first contains terrain-only
   output and is restored before each of the 11 intro LMI frames. At entry to
   `0x22547`, native blits final intro entry `#0x7c` into that same allocation
   once. Every one of the six contract and ten release LUT frames then restores
   this shared terrain+final-LMI snapshot. The coordinate rewrite and
   intervening strip-copy bridge mutate other state/buffers, not the snapshot.
   `ComposeNativeUnitPresentLUTSnapshot` now preserves this atomic phase
   boundary; the earlier allowance for unrelated contract/release snapshots
   is withdrawn.

   Unit-present bridge correction (2026-07-27): the often quoted “27
   presents” counts only full-viewport `0x11eb0` calls. After contract,
   `0x22547` returns FDOTHER #3 entry0 pointer+1; `0x22253` restores the shared
   snapshot and uses that LUT for one bridge-only `0x22046` remap/object
   redraw without `0x11eb0`. It then calls memmove
   `0x373c4(dest,src,24)` once per row from 456-stride work buffer to
   320-stride VGA and delays 10ms after every row. If targetY equals cameraY
   it copies 18 rows from the target row; otherwise it copies 24 rows beginning
   six pixels above the target. Therefore the observable schedule is 27
   full-viewport presents plus 18/24 progressive direct-VGA row writes.
   `NativeUnitPresentStripLayoutFor` and
   `RunNativeUnitPresentStripBridge` preserve exact offsets, strides,
   progressive visibility and preflighted bounds.
   `ComposeNativeUnitPresentStripBridge` binds the complete snapshot restore →
   bridge-only LUT/object redraw → direct-row sequence and intentionally never
   performs a full viewport copy.

   Bridge-LUT boundary correction: FDOTHER #3's real directory offsets are
   `0x66,0x166,0x266...`, exactly 256 bytes apart. Because `0x22547` returns
   entry0 pointer+1 while `0x22046` consumes a full 256-entry table, the bridge
   LUT is exactly `entry0[1:256] + entry1[0]`; it is not aligned LUT0 or LUT1.
   `NativeUnitPresentBridgeLUT` preserves this cross-entry view and real-archive
   regression rejects either aligned approximation.

   Five-argument caller ABI: `0x22253(unit,newX,newY,visualX,visualY)` renders
   intro/contract at the independent visual pair, then writes `newX/newY` to
   runtime record `+0/+1`, and only then runs bridge/release. Command23 first
   calls `new=0xff/0xff, visual=current` to disappear, then
   `new=visual=destination` to appear. The ending caller performs only the
   first form for unit1; scripted helpers use `new=visual`. Therefore neither
   pair should be generically renamed source/destination.
   `PlanNativeUnitPresentCall` preserves this byte ABI.

   `fdicon.Bank.BlitNativeForegroundLayer` now supplies the matching steady indexed layer. It applies those raw unit gates and schedule in roster order, then reproduces `0x12ac6`'s camera interval, foreground-control bit7, bit8 flip adjustment, `index+1` descriptor selection, `buffer+0x8088` placement, and raw versus LUT-transparent branch. It preflights the full selected set before a write. Coordinates that would index outside the supplied editable map are intentionally skipped rather than reading unchecked native memory; that is an explicit fail-closed adapter boundary. Scripted `0x1366a` composition, range overlay, HUD and VGA present remain separate.

   `fdicon.Bank.BlitNativeUnitLayer` now closes the intervening steady `0x127a9→0x127e0` layer as a pure indexed pass. It accepts only raw unit subset fields (`+2` slot, `+3` pose, `+4` movement offset, `+5` bit7 palette branch, `+0x26` base-frame flag) plus the preceding inactive gate, exact camera extents, global idle/moving cycles and pixel shift. It preserves the native visible bounds `X∈[camX−1,camX+maxX]`, `Y∈[camY−1,camY+maxY+1]`, negative-offset skip, slot→key pointer resolution, and raw versus palette-band blit. All selected entries are preflighted before the destination changes, so malformed editable selector input cannot yield a partial indexed frame. It is deliberately not an Ebiten adapter and does not schedule foreground/HUD/present.

   Caller boundary correction: native foreground is not confined to the steady `0x127a9` redraw. Official IDA 9.4 shows `0x1366a` also calls `0x129ec` after its step-specific `0x11eee` base-terrain redraw and per-slot `0x127e0` sprite loop, before `0x11eb0` and the later present/redraw calls. This path mutates runtime `unit+3` from its scripted step data while composing frames. The 106-entry `0x1366a` input bank is already decoded as editable acting frames; this evidence adds its indexed layer order. A future adapter must therefore schedule foreground occlusion in both steady and scripted-step frame paths. The final native presentation stages remain unmodelled.

   `0x11eb0` is now closed as a plain row-by-row `memmove`, not an unknown effect: the standard `0x11cac` caller copies width 312 × height 192 from `buffer+0x8088` (source stride 456) to VGA `0xA0504` (destination stride 320). `fdicon.CopyNativeIndexedRegion` preserves that explicit indexed-buffer contract with bounds validation.

   Selected-unit HUD boundary: `0x11cac` calls `0x1acf3` after terrain, range and unit/foreground layers but before the viewport copy. `0x1acf3` returns without drawing unless both raw display bytes `0x51aab` and `0x51aac` are nonzero. Gate A is not a constant: `0x10010` restores it from native-save plaintext offset `0x30d2`; gate B has separate UI writers. It first calls `0x12e38` on cursor globals: that helper is a terrain-cell resolver, yielding FDFIELD tile word masked to 10 bits, event low five bits, and the selected four-byte FDSHAP control record; `fdicon.NativeTerrainCursorInfoForCell` preserves this raw contract. Its control byte+1 indexes the verified `0x51a12`/`0x51a2a` terrain AP/DP table: 0→(+5,0), 1/5→(0,0), 2/3→(-5,+10), 4→(-5,-5). `battle.Load` derives the same byte per validated map cell and combat consumes it directly. The now-closed panel geometry is FDOTHER #5 LMI1 #130 (69×34) at `buffer + stride*157 + x`; both terrain and optional unit icons use the row-5 destination `stride*5+6`; AP signed-number path is at `stride*8+0x2b`, and DP at `stride*19+0x2b`; `0x1aeb1` chooses raw directory entry #0x83 (6×7) for a nonnegative table value or #0x84 (6×5) for a negative one, makes the value absolute, then calls the native decimal digit path at `+8`. These are literal hexadecimal immediates in `0x1aeb1`, not decimal 83/84. The resource artwork's semantic label remains unassigned. `x` is the raw static anchor (data initial value 1). Direct Docker Capstone reading of `0x1ad2a..0x1ad5f` confirms it is persistent: only visible cursor row `[0x53abd]>5` plus column `[0x53ab9]<3` writes `0xf2`; the same row plus column `>9` writes `1`; all other pairs retain the prior global. These two globals are camera-relative cursor coordinates, not dialogue-box width/height; doc14's older assertion has been removed. `battle.NativeMapHUDRuntimeState` preserves the two raw gate bytes and anchor only when an explicit save/scenario source materializes them. The optional unit icon is FDICON group `unit+2 * 12 + rawState`, with raw state 3 aliasing 1, blitted at `stride*5+6`; its current/max HP words `+0x40/+0x42` feed `0x1875d` at `stride*21+9` in raw mode 3. `fdicon.NativeMapHUDUnitFrameIndex` preserves only that selector. The global and icon/HP semantic names remain raw.

   `indexedmap.BlitNativeMapHUDPanel` is the executable first subpass only: it requires both recovered raw gates, validates FDOTHER #5 entry #130's 69×34 geometry, and transparently blits it at `NativeMapHUDLayoutFor(anchorX,456).Frame`. **Codec correction:** #130/#0x83/#0x84 are directory entries sent to `0x4e63d`; `DecodeNativeMapHUDFrames` uses `ParseLMI1FrameEntry`/four-mode `Frame`, not ordinary `ParseLMI1` (`0x4e916`) cells. Terrain/unit icons and digits remain explicitly out of this primitive; callers can use it as the required HUD callback in `ComposeFrame` without pretending the partial panel is a complete native HUD.

   `indexedmap.BlitNativeMapHUDSignedNumber` closes the immediately following `0x1aeb1` selector boundary: it accepts an already-recovered number origin, uses #0x83 (6×7) for `value>=0` or #0x84 (6×5) for `value<0`, then invokes a mandatory decimal callback at `origin+8` with the absolute value. The primitive commits atomically only after that callback succeeds. It neither supplies a decimal font nor decides the table value, AP/DP source, or number meaning; those remain caller/data-flow work.

   That callback is now closed for this call-site by `BlitNativeMapHUDTwoDigitNumber`: `0x1aeb1` supplies `0x187d6` glyph base `0x1f` and fixed width 2; `0x187d6` patches `%0.5d` to `%0.2d`, then calls `0x16886→0x4e63d` for glyph directory entries `0x1f+digit` at offsets `origin+8` and `origin+14`. Real FDOTHER #5 confirms digit entries #0x1f..#0x28 are 6×8 except #0x20 (digit 1) at 5×8; advance is nevertheless six pixels. The adapter rejects values outside `0..99` rather than silently rendering native's truncated first two characters. Number source/meaning remains unassigned.

   Terrain-icon subpass closure: immediately after `0x12e38(cursor)` fills its eight-byte local, `0x1acf3` reads local word0 (the masked 10-bit terrain descriptor), uses it as the selected FDSHAP bank offset-table index, and raw-blits through `0x4deda` to the panel row-5 destination `stride*5+6`. `indexedmap.BlitNativeMapHUDTerrainIcon` preserves exactly that raw descriptor input and destination, validating only editable bounds; it does not reuse texture previews or name the terrain category.

   Unit-icon subpass closure: if `0x12c0d(cursor)` returns a runtime unit index, `0x1acf3` uses `unit+2` as the global selector-cache slot, reads the raw global state counter (3 aliases 1), resolves that cached twelve-frame FDICON block and raw-blits it to panel `stride*5+6`. `indexedmap.BlitNativeMapHUDUnitIcon` preserves the cache slot/state boundary and makes no inference that slot is a character or portrait identity.

   Terrain AP/DP subpass closure: `0x1acf3` indexes its two static signed tables with the resolver's raw control byte+1: 0→(+5,0), 1/5→(0,0), 2/3→(-5,+10), 4→(-5,-5). `indexedmap.NativeMapHUDTerrainAPDP` keeps that bounded raw mapping, and `BlitNativeMapHUDTerrainAPDP` calls the verified signed two-digit renderer at the exact AP/DP layout origins atomically. The control byte's higher meaning and HP ratio path remain separate.

   HP subpass closure: after the optional unit icon, `0x1ae8e` zero-extends unit words `+0x40/+0x42`, passes `(destination,stride,current,maximum,3)` to `0x1875d`, and destination is the recovered HP origin. That helper selects glyph base #0x1f only when the two words are equal, otherwise #0x2a; `0x187d6` then formats **current** to exactly three digits at six-pixel advances. For current >999 it does not truncate: it blits `base+10` directly. Real FDOTHER #5 verifies #0x29 and #0x34 are 18×8 overflow frames, while both bases' digit #1 is 5×8 and all other digits 6×8. `indexedmap.BlitNativeMapHUDHP` preserves this raw comparison/unsigned-word boundary atomically; it does not call the unequal branch “damage” nor infer a percentage calculation.

   Full proven HUD assembly is now `indexedmap.BlitNativeMapHUD`: panel → terrain → AP → DP → optional icon → optional HP, in the direct `0x1ad72..0x1aea9` order and as one transaction. It accepts `NativeMapHUDInput` raw resolver outputs; `OptionalUnit=nil` represents no `0x12c0d` result or a post-lookup skip. The latter is closed as `NativeMapHUDOptionalUnitEligible`: raw `unit+7==0x79` skips; otherwise raw `unit+0x1f==0x0a && unit+6==1` skips. The three bytes retain raw names rather than a guessed character model. Closed display gates remain a no-op before resource validation.

   Constructor provenance correction: Docker Capstone plus official IDA 9.4 of `0x10d7f..0x10efc` proves runtime `unit+6` receives FDFIELD roster byte0 and `unit+7/+8` receives roster byte1; the existing editable `map_selector_key` and `battle_fig` fields therefore preserve those two raw sources. A further table trace closes the high-class branch: `0x10da4` computes `FDFIELD b1-0x44`, calls `0x4e4ff`, and that helper returns the 10-byte record at `0x61af9 + index*0xa`; `unit+0x1f/+0x20` are record bytes 0/1, while bytes 2/4/5/6/7/8 feed the other native fields. This is an EXE static table, not a proven DATO resource. The lower branch calls `0x4e4e8 → 0x61da1` (24-byte records) and `0x4e4d1 → 0x620a1` (11-byte records); `unit+0x1f/+0x20` come from the selected `0x61da1` record bytes 0/1. `0x619fd` belongs to the distinct `0x4e516` helper and is not part of this constructor path. Until both branch selectors and these raw table records are exported, optional unit/HP admission remains nil; portrait/class must not be used as a substitute.

   Export bridge: when supplied the paired FDSHAP terrain resource, `export_engine_assets.py` writes `native_terrain_control` (the complete raw four-byte records) alongside serialized per-cell `native_tile_blit_modes`. The latter proves field shape/provenance but is immediately replaced by the `0x4dbfc` runtime `0xff` fill; normalized `cost` remains a separate gameplay approximation.

   Runtime bridge: `battle.Load` accepts those two fields only when map dimensions, cell count, control-record alignment and every 10-bit tile index validate exactly; otherwise all native renderer/mechanics fields stay nil. It retains the raw tables and derives `NativeTerrainMoveCodes` from each tile's FDSHAP control byte+1. This is the authoritative combat AP/DP input; normalized `cost` is used only as a legacy incomplete-export fallback. The fields are not silently substituted into the current PNG/Ebiten path.

   Archive bridge: `fdother.DecodeSpriteBankResource` is the explicit LLLLLL-resource→24×24 four-mode-bank route for FDSHAP's evidenced even image resources. It deliberately returns only a `fdicon.Bank`: map/resource pairing and adjacent control resource selection remain caller-owned, preventing an image bank from being silently paired with a guessed terrain table.

   Map resource pairing closure: `DecodeMapTerrainResources` accepts an explicit map index N and loads only FDSHAP image #`2N` plus control bytes #`2N+1`; it rejects an inconsistent bank/control capacity. This replaces any future tile-count heuristic. Player archive map 0 regression fixes the concrete pair to 288 frames and 1200 control bytes.

   Production gate: `cmd/fd2.Game.loadMap` now attempts this complete original bundle (HUD FDOTHER frames, FDOTHER #1 range bank, explicit FDSHAP pair, FDICON.B24, FDOTHER #3 LUTs and palette) and stores it only when every decoder succeeds. The current PNG presentation remains unchanged until the indexed-to-Ebiten bridge consumes the bundle; missing or malformed original files therefore cannot create a half-native frame.

   Regression/harness closure (2026-07-26): Docker image `fd2-go-test-local` already contains Xvfb; running `GOMAXPROCS=1 GOFLAGS=-p=1 xvfb-run -a -s "-screen 0 1280x1024x24" go test ./...` passes every remake package. `cmd/fd2.assetPath` now searches cwd ancestors after the existing user-data/AppImage/executable layers, because Go runs package tests with cwd `cmd/fd2`; this fixes test/runtime asset resolution without weakening the editable-user override or fail-closed resource rules. The ch14 continuation-line assertion now follows FDTXT_015 count-aligned indices 2/5 (scene lines 4..12 / 4..8), and conditional ch16 SPAWN remains branch-local after LOADCH with no merged-slot assumption.

   Native unit table export boundary (2026-07-26): `tools/extract_native_unit_tables.py` reads the LE object through `le_xref` and emits only raw records: `high_class` `0x61af9` (68×10, helper `0x4e4ff`, selector `FDFIELD b1-0x44`), `lower_class` `0x61da1` (32×24, helper `0x4e4e8`, selector `FDFIELD b1` in the lower branch), and `lower_aux` `0x620a1` (68×11, helper `0x4e4d1`, same selector). Docker extraction against the real FD2.EXE validates all 68/32/68 records. The JSON deliberately keeps selector provenance and `bytes_hex` without assigning gameplay names; it is an editable RE fixture, not permission to substitute portrait/class or to enable HUD optional unit/HP.

   可編輯單位邊界：`tools/export_units.py` 接受可選的原始表 JSON，在建構器公式來源完整時可寫出 `native_constructor:{branch,index,record,aux_record}`、`native_record_word42` 與 `native_record_word46`。為避免每列複製完整表格，`tools/sync_native_selector_fields.py --native-tables` 只將已被消費的 race/class 原始位元組、初始命令遮罩、`word42` 與 `word46` 合併到 33 張地圖，保留其餘人工校正欄位。`battle.NativeConstructorTable` 仍是經驗證的可選稽核物件；記錄格式錯誤時失敗即關閉，不退回以 portrait/class 猜測。

   HUD raw-state closure: `sub_11cac` calls `sub_1297d` immediately before the native map compositor. Its `[0x53c0b]` state advances `3→0` only when signed `rawTimerTick([0x46c])-rawLastTimerTick([0x53c0f])` is negative or greater than four, then stores the new last tick; all other calls preserve it. `[0x46c]` is the low 16-bit BIOS timer tick, not a VGA scanline: `0x17aa9` performs a tick busy-wait with explicit 0x10000 wrap correction, and `0x16d00` uses the same word as a two-tick update gate. `indexedmap.AdvanceNativeMapHUDState` preserves the pure ABI. The actual runtime caller still owns timer/call timing, so the Ebiten optional unit icon remains fail-closed until those globals are materialized.

   Sprite-cycle correction (2026-07-27): the same `sub_1297d` always advances
   moving selector `[0x53c07]` on every call, independently of the gated idle
   selector above; both wrap 3→0. `AdvanceNativeMapSpriteCycles` now preserves
   the complete mutation and the HUD-only helper delegates to that single
   implementation. A successfully constructed `battle.State` now owns these
   three globals as `NativeMapCycleState`; legacy or partially materialized
   states fail closed. Runtime monotonic-clock BIOS tick/call timing is still
   not materialized.

   Terrain timing correction (2026-07-28): official IDA 9.4 and instruction
   level Capstone close `0x11eee`'s independent globals. With raw override
   `[0x51a93]==-1`, phase `[0x53c1f]` advances modulo 20 only when the
   sign-extended BIOS low word minus `[0x539f4]` is greater than two, or the
   current signed word is less than the latch; an override `0..19` writes the
   phase directly without updating the latch. This is not a per-compositor-call
   counter. `fdother.AdvanceNativeTerrainPhase` and battle-local
   `NativeTerrainPhaseState` preserve both paths. `0x11eee` separately toggles
   `[0x53a40]` once per new BIOS word, while `0x127e0` toggles independent
   `[0x53a04]` once when that unit-layer call first observes a new word.
   `NativeBinaryTickState` represents these two latches independently; neither
   is an alias of the terrain LUT phase or the idle/moving sprite cycles.

   Raw pose/motion lifecycle (2026-07-27): both player materialization
   `0x10a77..0x10aad` and FDFIELD spawn initialize runtime `+3/+4=0/0`.
   Direction entries `0x12eaa/0x1300d/0x13185/0x13315` write pose
   down/left/up/right, write motion `1..6` during each grid step, then mutate
   X/Y at the seventh boundary and clear motion to zero without restoring
   pose. `0x1366a` normal acting follows the same lifecycle; special frames
   only write pose. The remake now materializes an independent battle-local
   `NativeMapPresentationState` (`+0/+1/+3/+4`) together with each verified
   selector slot. Both ordinary grid walking and decoded acting advance raw
   motion `1..6` on the source cell and commit the destination on tick seven;
   placement and pose writers update the same state. `NativeUnitLayerEntry`
   admits a unit only when presentation, selector-slot, and record-byte
   provenance are all present. Persistent/scenario Dir is therefore not used
   as the constructor source, and normalized `Unit.X/Y/Dir/OffX/OffY` are not
   treated as aliases. This closes the state sequence, not wall-clock parity:
   the current Ebiten update cadence is not yet the original BIOS 18.2 Hz
   scheduler, and the indexed frame input/presentation bridge remains separate.

   Raw roster admission (2026-07-28): `NativeMapFrameRoster` now builds the
   unit and foreground arrays as one transaction from the battle state.
   Foreground admission additionally requires explicit `unit+7`, race and
   class provenance; the older `BattleFig=Fig` compatibility projection is
   tracked by `HasBattleFig=false` and cannot enter the indexed compositor.
   Any missing unit/foreground field rejects the entire snapshot, so one
   legacy record cannot create a mixed native/normalized frame.

   Strict runtime frame-input boundary (2026-07-28):
   `cmd/fd2.buildNativeMapFrameInput` now joins the all-or-nothing original
   banks, exact FDFIELD tile/blit-mode arrays, validated selector cache/raw
   roster, selected terrain LUT and the recovered cycles/flips into one
   `indexedmap.NativeFrameInput`. It requires the editable raw control table
   to equal the selected FDSHAP bytes and accepts only explicit tile-space
   camera, compositor-admissible selector `0..5`, cursor and complete HUD
   input. This narrower bound belongs only to the drawable frame adapter:
   battle runtime preserves the full `[0x51a83]` domain described above. It never
   derives those globals from the remake's 640×400 pixel camera, normalized
   reachability, selected unit or PNG state. This closes the composition-input
   admission transaction, not production presentation: the native 320×200
   camera lifecycle, HUD gate/anchor persistence and monotonic BIOS-clock
   caller still must be connected before replacing the visible map renderer.

   Campaign flow correction（2026-08-02 勘誤）：較早把 `ch29_post` 接到
   `postbattle_ch29_persist` 的說法已撤回。scenario `ch29` 載入 raw map28；依主迴圈的
   零起算 dispatch，第29戰應使用 raw `ch28_post`。這是2026-08-02當時的歷史 gate；
   後續 binding 已獲准並通過 `preparation_ch30` 存讀檔 `RUNTIME-E1`。
   raw `ch29_post` 的 LOADCH／persistent-roster 與 `0x2bce5` 證據仍保留，但其正確 owner
   必須和第30戰結局流程另行閉合，不能用已恢復函式本身推定 campaign 接點。

   Presentation bridge (strict gate): `drawNativeMapHUD` converts the verified 456-stride indexed buffer to a 320×200 paletted Ebiten image only when `NativeMapHUDRuntimeState`, selector cache/cycle and every selected-unit raw admission byte are present. It now draws panel/terrain/AP/DP plus the proven unit icon and `+0x40/+0x42` HP path together. The former hardcoded `DisplayGateA=true, DisplayGateB=true, AnchorX=1` partial path has been removed because native load can overwrite gate A. Missing provenance falls back before any native drawing. `NativeMapHUDPersistentState` now separates save-persistent gate A、process-persistent anchor 與 controller-owned gate B；custom save and native chapter restore preserve gate A, while battle entry materializes gate B only from the proven value 1. `battle_ch01`、`battle_ch26` and `battle_ch27` use editable `native_map_hud_inherited` together with their evidenced views. Exact fixed HUD bytes remain available only for explicit fixtures/snapshots; this inherited owner closes E1 state flow, not whole-campaign visual parity.

   HUD pointer-base correction (2026-07-28；歷史 E1 橋接，非目前正式 handler 畫面): direct Capstone at
   `0x11cfa..0x11d0a` proves the caller pushes stride `0x1c8` and
   `[0x53a49]+0x8088` into `0x1acf3`. Therefore every recovered HUD offset,
   including row157, is relative to the same viewport pointer used by terrain
   and the later `0x11eb0` copy. The earlier production adapter passed the
   allocation base, causing the HUD to land 72 rows／72 bytes away and disappear
   from the final bottom edge; its regression had encoded that wrong coordinate.
   `ComposeNativeFrame` now passes `work[0x8088:]`, keeps failure atomicity, and
   verifies the panel reaches VGA `(anchor+4,161)`. That paragraph's rebuilt
   [remake frame](../figures/native-map-ch01-remake.png) is the older
   `FD2_CAMP_NODE=battle_ch01` direct-entry artifact; it is retained only to
   preserve the pointer-base evidence and must not be read as the current
   story-handler composition. The original-video
   [HUD oracle](../figures/native-map-ch01-original-video.png) remains a separate
   320×200 reference. The reproducible extractor
   `tools/extract_fd2_video_frame.sh video/fd2-ch1.mp4 434.5 ...` first crops
   the recording's centered `(16,100,1408,880)` game viewport, then returns it
   to 320×200; direct whole-video scaling was a distorted oracle and is removed.
   The 434.5-second frame and that historical direct-entry frame shared camera
   `(1,13)`, absolute cursor `(8,15)`, visible cursor `(7,2)`, tree terrain and
   HUD `A -05 / D +10`.
   The screenshot hook formerly assigned only normalized `curX/curY`, leaving
   `NativeMapViewState` stale; it now drives `MoveNativeMapCursor` and persistent
   HUD-anchor updates. Roster/event presentation still differs, so these images
   prove only the pointer-base/camera/cursor/terrain/HUD bridge, not a full-frame
   pixel diff. The current formal handler image and its remaining battlefield
   differences are recorded in
   [`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json).

   Pre-handler→battle roster correction (2026-07-28): the remaining ch01
   roster mismatch was not a compositor omission. `ch00_pre` performs
   `LOADCH(map0)` and constructs the party first, then acting resource 0 moves
   runtime slots 0..3 upward for six normal beats. The authored scenario stores
   deploy cells in UI party order `0,4,9,30` with Y `[20,22,21,23]`, but handler
   construction first applies JOIN order `0,9,4,30`; its runtime Y therefore
   changes from `[20,21,22,23]` to `[14,15,16,17]`. Later SPAWN/ACT operations
   append the initial FDFIELD groups in the same runtime array. The original 434.5-second
   frame shows those post-ACT positions. The remake's old `resetBattle`
   unconditionally discarded `storyActors` and replayed scenario
   `on_battle_start`, returning the party to deployment cells; the comment
   that every pre-cutscene array must be cleared was therefore false.
   Runtime handoff now occurs only when the last handler LOADCH roster path and
   party-scenario path exactly equal the following battle node. It rebuilds
   native selector slots over the already moved/appended records, preserves the
   remaining FDFIELD roster for turn events, and marks the already represented
   `on_battle_start` event consumed. Direct node starts, retries, unrelated
   cutscenes, mismatched sources and non-runtime-append scenarios still rebuild
   normally. Regression fixes both the adopted `[14,15,16,17]` state and the
   direct-start `[20,22,21,23]` state.

   End-to-end runtime proof (2026-07-28): a bounded regression now compiles the
   actual `ch00_pre` binding, executes every BeatRunner job and dialogue boundary,
   crosses the real campaign fade into `battle_ch01`, and then checks the adopted
   state. The exact frontier is 12 records: party slots 0..3 in JOIN order
   `[0,9,4,30]`, group 1 in slots 4..7 and group 2 in slots 8..11. Party
   coordinates are `0:(7,14)`, `9:(10,15)`, `4:(8,16)`, `30:(11,17)`;
   recovered deactivate writer slot 9 has raw byte `+5 == 1`, while scenario
   pending groups 3..7 remain attached. This is runtime evidence beyond the
   earlier compiler-only “0 unresolved issues” claim.

   Production steady-frame and drawable-target slice (2026-07-28): ch01 now materializes the explicitly sourced persistent selector one, uses the original party-first/initial-group append constructor order, and consumes regenerated FDFIELD/FDSHAP raw map fields. Runtime composition byte+3 is initialized to `0xff` exactly as `0x4dbfc`; using serialized zeroes was an incorrect assertion that forced the whole steady map through the LUT branch and visibly washed out the palette. `nativeBIOSClock` supplies a battle-local signed BIOS low word at the PIT rate; one Update corresponds to the one `0x1297d` call at `0x11cb7`, while terrain phase and the two independent BIOS-word latches consume the same sample. `drawNativeMapFrame` executes the proven interactive `0x11cac` pipeline and FDOTHER#1 descriptor 0 supplies the native steady cursor; the former white rectangle approximation is removed. Command target entry materializes the first-stage `0x14818` remaining-budget grid and writes `record+4+2`; selectors 2–5 remain in the same production indexed compositor and use their exact ordered FDOTHER#1 call tables. Cancel and successful effect exit perform the `0x4dbfc` reset and restore selector one. Selector 6 is a separate field mutation and 7+ have no `0x122dc` draw table, so they still fall back; target flash and indexed effects remain incomplete.

   Palette assertion correction: `FDOTHER#0` is a VGA DAC palette and index zero is not globally transparent. `ParseVGAPalette` now returns 256 opaque entries; only blitter-specific adapters such as `RawCell.Paletted` clone the palette and make zero transparent when their native zero-source rule requires destination preservation. This prevents a full indexed VGA frame's four-pixel border from leaking the legacy renderer beneath it.

   Codec boundary: LMI1 is a directory, not one universal codec. `0x1acf3` sends #130 and its table-value-dependent hexadecimal #0x83/#0x84 entries to `0x4e63d`, the four-mode transparent RLE path. `fdother.ParseLMI1FrameEntry` / `DecodeLMI1FrameResource` preserve that explicit route and regression-decode the three player-archive entries at their verified geometries (69×34, 6×7, 6×5). `fdicon.NativeMapHUDLayoutFor` preserves the six strict 456-stride destinations and rejects an anchor whose 69-pixel frame does not fit the native 320-pixel viewport. These adapters intentionally do not reinterpret adjacent LMI1 cells handled by `0x4e916` or claim an Ebiten renderer.

   FIGANI placement bridge: `0x2935b` uses each frame header's signed X/Y, so runtime `assets/figani/meta.json` is placement data, not a hand-tuned animation hint. `cmd/fd2` regression reads every metadata resource from player-provided `FIGANI.DAT` and checks every exported `(X,Y)` against `internal/figani.DecodeResource`; a missing archive skips only that player-asset assertion. PNG rendering remains a presentation adapter, but cannot silently drift from the native frame coordinates.

   FIGANI scheduler boundary: official IDA at `0x2b9a1` shows `arg4==0` only clears the subframe counter and performs no render. On the advancing path, native code selects the current frame first, calls `0x2935b`, then reads descriptor `+6` as the delay; only after the rendered subframe reaches that delay does it reset subframe and wrap the frame index. `internal/figani.NativeScheduler.Step` implements this state machine as a pure, caller-owned primitive. It does not infer `0x2935b` presentation semantics or authorize an ending renderer.

   **2026-08-09 battle presentation bridge (E1):** `remake/assets/figani/delays.json` now
   records the descriptor `+6` delays for the 22 exported FIGANI resources, with a
   regression against the fixed player-provided `FIGANI.DAT` hash. The battle full-screen
   PNG adapter consumes `internal/figani.DisplayScheduler`, which scales only the display
   clock by explicit `FD2_BATTLE_FPT` and rejects a missing or mismatched delay/PNG pair.
   This closes frame selection and hold duration only; it does not authorize inferred hit
   timing, damage/HP commit, sound, palette flash, TAI placement, or E2 visual parity.

   Battle-entry split-slide boundary: official IDA at `0x1f42d`/`0x1f1cc` fixes FDOTHER#5 LMI1 entry `0x52`, stride 456, five offsets `100,75,50,25,0`, and placements `(85-offset,82)` plus `(165+offset,81)`. `fdother.NativeBattleEntrySplitSlideSteps`, clipped cell blit, and `RunNativeBattleEntrySplitSlide` preserve one present/restore pair per pass. Direct caller `0x1a30b` operates on battle records and the 456-stride battle surface; this is not evidence of a preparation-selection-window animation. MAP/TURN labels and native VGA restore remain caller-owned and fail-closed. A deterministic remake shell capture is tracked as [`preparation-remake.png`](../figures/preparation-remake.png) (Xvfb, `FD2_CAMP_NODE=preparation_ch02`, frame 30, 640×400); it is not an original visual oracle.

   Preparation record gate boundary: the official `0x1a866` loop reads only raw unit offsets `+0x25`, `+0x05`, `+0x06`, `+0x40`, `+0x42`; it accepts when `+0x25!=0`, selector equals `+0x06`, and `+0x05 bit0==0`, then writes `+0x40 := max(0,+0x40-(+0x42/10))` and stores the divisor. `fdother.ParseNativePreparationRecord`, `NativePreparationEligible`, and `NativePreparationAdjustedWord40` preserve this ABI without naming the fields as active/alive, deployment, coordinates, or gameplay stats.

   Preparation dispatch boundary: `0x1a813` computes each candidate as `base+3*i` for exactly 16 slots, compares bytes `+3` and `+5` to caller/global gates, then reads byte `+4` as an index into a separate function table and invokes it with zero. `fdother.FindNativePreparationDispatch` preserves the overlapping 3-byte stride and returns raw matches only; it does not invoke callbacks or assign event names.

   Preparation timer boundary: official `0x1a941` scans 0x50-byte records selected by `+6==selector` and `+5 bit0==0`, then decrements six bytes at `+0x22..+0x27`; only a nonzero byte that becomes zero emits the downstream redraw path, whose source argument is `0x1e1+counterIndex`. `fdother.TickNativePreparationTimers` preserves this in-place transition and returns raw expiry metadata without naming statuses or effects.

   Preparation input boundary: official `0x19953` maps scan codes `0xe0/0x52/0x1c/0x39` to return `1`, `0x01/0x53` to return `-1`, and updates raw cursor `[0x53c57]` to `0` for `0x4b` or `1` for `0x4d`; all other keys continue waiting. `fdother.ApplyNativePreparationInput` preserves this result/state contract, without labeling the two terminal returns as YES/NO.

   D8 scope correction: the official `0x1a30b` body contains no `0x15f84` text call and therefore does not prove MAP/TURN/ENEMY/FRIEND/NPC labels. Its first loop gates raw record bytes `+6==2`, `+5&0x81==0`, `+0x25==0`, `+0x26==0`, then advances word `+0x40` by `word+0x42/5` with an upper clamp before indexed redraw. `fdother.NativeBattleEntryStep` preserves this numeric transition only; the later `0x1f1cc/#0x52` slide is a separate indexed choreography.

   Shared-caller correction: official xrefs show `0x1a30b` is called from `0x135c5`, `0x17154`, and `0x17272`, not only from the battle-entry path. The latter callers sit beside FDTXT_000 `0x19c/0x1a4` interaction messages and `0x1728c` selector-flag handling. Therefore the raw transition must remain a reusable record primitive; it cannot be labeled or wired as a D8-only preparation action.

   Raw action-bit helpers: official `0x13512(index)` sets `record[index*0x50+5] |= 0x80`, while `0x13536` clears that bit across the record count. `battle.SetNativeRecordBit7` and `ClearNativeRecordBit7All` now preserve these byte-level mutations with bounds checks; they do not force a higher-level turn interpretation.

   Inventory-cell correction: official `0x1b8a6(unit)` scans exactly eight two-byte cells at `record+0x0a+2*i`; it increments a count whenever the flag byte bit7 is **clear**. The helper itself does not verify compactness or return a prefix length. Its callers use that numeric count as the upper bound for raw slots `0..count-1`; a malformed hole can therefore expose a stale item byte in the scanned range. Bit7 set is the reserved empty state consumed by `0x1bb8c`, so the former `free-slot count` assertion and `battle.NativeInventoryFreeSlotCount` name were wrong and are removed. `battle.NativeInventoryOccupiedCount` preserves only the exact count and ignores item-byte values.

   Inventory reservation boundary: official `0x1bb8c(unit,item)` scans those same eight cells, takes the first flag-bit7 reserved cell, clears its flag, writes the supplied item byte, and returns native success/failure (`1/-1`). `battle.AssignNativeReservedItem` reproduces this atomic raw mutation; no item category or shop meaning is inferred.

   Item-panel source/data boundary: official IDA 9.4 closes `0x17eef` as
   `0x168b6(dst,320,5,7,5,5)` for the frame at `(5,7)`, DATO selected by unit
   record byte `+7` at `(8,10)`, and FDOTHER #5 LMI1 directory entries 20/21
   (header offsets `+86/+90`) at `(92,7)` and `(5,94)`. The following
   `0x17fc0` schedule has two bar calls, four compared-number calls, eight raw
   number calls, three FDTXT calls and one base plus three conditional icons
   at fixed 320-stride destinations. `battle.NativeItemPanelBaseLayoutFor`
   and `NativeItemPanelDataPlanFor` preserve those source IDs, coordinates,
   record offsets, colors and primitive widths as data. This closes the
   reconstruction contract, not the indexed-to-Ebiten renderer; raw record
   offsets without independent semantic evidence remain unnamed.

   Correction: the `0x1ac62` loop is not a preparation command stream. Its caller `0x1aa1d` uses FDTXT_000 indices `0x1b0..0x1b3`, which decode to post-resolution loot/interaction messages (enemy item, full inventory, money), so the higher-level preparation label is withdrawn. The proven part is only a `base+3*i` `{kind:byte,payload:u16le}` stream with observed kind `0/1/2/3` branches; `fdother.ParseNativePostResolutionCommands` preserves it and refuses truncation without assigning event names.

   `internal/fdicon.Sprite.BlitLUT` now reproduces the `0x4dcc6` pixel contract as a pure indexed primitive: RLE source writes become `lut[source]`; mode-3 spans become `lut[destination]`; mode-1 dither holes remain unchanged. Its fixture regression covers all three effects. It deliberately accepts an explicit LUT and destination buffer only—the map palette-entry selector, frame scheduler and foreground pass remain separate adapters.

   `fdother.ParseLUTBank`/`DecodeLUTResource` now close the FDOTHER #3 input boundary: its `LMI1` directory contains 23 independent 256-byte tables, not UI cells. The original-archive regression verifies that count and every table length. This loader plus `BlitLUT` is sufficient for a caller that has an evidenced selector; it does not infer one.

   The default `0x11eee` selector is now evidenced too: static table `0x51a97` maps runtime phase `0x53c1f` 0..19 to FDOTHER #3 entries `[0,1,2,3,4,5,6,7,8,9,10,9,8,7,6,5,4,3,2,1]`. `fdother.NativeTerrainLUTIndex` makes the bounded sequence explicit. The separate explicit-override branch remains raw/unlabelled; no visual name is inferred for the cycle.

   Correction: the sprite pointer table is no longer opaque. `0x11019` builds/caches a twelve-pointer FDICON block from its raw key and resource arguments; complete IDA Pro 9.4 constructor evidence shows `0x10c50` passes FDFIELD **b1** plus its caller resource and writes the returned cache slot to `unit+2`. FDFIELD b0 independently becomes runtime `+6`; the same b1 is copied to `unit+7/+8`. `0x127e0` then selects `unit+2 × 12 + pose×3 + cycle`. Thus `unit+2` is a cache result, not a direct character/portrait field. This is distinct from battle `0x287b5..0x2884c`, which selects `FIGANI.DAT` by `unit+7 × 3`, even though scripted FDFIELD supplies both runtime selectors from the same raw b1. `export_units.py` and the editable map assets now preserve b1 as `map_selector_key` and `battle_fig`; missing older JSON retains an explicit `fig` fallback. `fig` remains an unseparated compatibility approximation. Cycle is global idle/moving animation state, not unit `+4`; that byte offsets the camera-relative placement. The remaining adapter boundary is the raw-key cache order, remap selection and layer order—not an invented FIGANI mapping. Direct evidence: [`fd2_fdicon_selector_constructor_ida.txt`](../data/ida/fd2_fdicon_selector_constructor_ida.txt).

   `fdicon.NativeSelectorCache` is the fail-closed data primitive for this ABI: `0x11019` keeps one process-global first-seen raw-key table (`0x53b17`/`0x53bdf`) and rejects non-byte keys in the remake. Its second argument is consumed only to materialize a new twelve-pointer block; both player `0x10a25` and scripted `0x10b69` load `FDICON.B24`, and cache lookup itself does not compare a resource pointer. Scripted FDFIELD source is explicit (**b1**, while b0 independently becomes native `+6`); player source is persistent `+7`. It deliberately does not map a slot to a character, portrait, or archive index; the remaining boundary is full mixed player/scripted construction order and indexed layer integration.

   The pointer-copy detail is now represented too: `KeyForSlot` reverses `unit+2` to the raw B24 key, and `SpriteForNativeSlot` then applies `key×12 + pose×3 + cycle`. This matches `0x11019`'s copied twelve-pointer block followed by `0x127e0`; it remains a process-global key cache and does not infer character identity.

   Runtime stores native map selection separately as optional `MapSelectorSlot`: its presence means an explicit native `unit+2` cache slot (including slot zero); absence must not fall back from legacy story/save `Fig`. This keeps the indexed compositor fail-closed while legacy UI remains compatible.

   The editable boundary now also carries optional `map_selector_key` (`MapSelectorKey` plus presence flag), the raw byte supplied to `0x11019` before slot allocation. `battle.MaterializeNativeMapSelectorSlots` accepts only an explicitly ordered construction batch and a process-global `fdicon.NativeSelectorCache`; it validates every key before allocating first-seen slots, then writes `MapSelectorSlot`. Missing/invalid keys leave both unit slots and cache untouched. Loader and renderer do not call it implicitly: the caller must first preserve the player-persistent then scripted-spawn order.

   `State.AppendNativeMapSelectorBatch` now supplies the atomic state seam for that proven order. It owns one process-global cache and appends only a fully valid batch; party `[9,4]` followed by scripted `[0,2,0]` produces slots `[0,1,2,3,2]`, while a missing-key batch changes neither runtime unit order nor cache. All 33 versioned map assets now carry the explicitly sourced scripted fields through `tools/sync_native_selector_fields.py --check`. The battle scenario path is connected: `spawn_party` materializes the party first, and every later `AppendGroup` uses the same cache for scripted groups. A malformed or provenance-free batch may retain legacy unit append for compatibility, but records `NativeMapSelectorError` and disables native selector resolution for the whole battle; story actors and direct-start/retry paths remain outside this E1 claim.

   Player-party construction is a separate proven source path: `0x1088d` copies each persistent 0x50-byte roster record from `[0x53bf7]` into the battle roster at `0x10a77`, then passes the copied record's `+7` byte and the chapter FDICON resource to `0x11019`; only its returned slot is written to runtime `unit+2` at `0x10aa2`. Map-script construction instead reaches `0x10c50` and supplies FDFIELD **b1**. These are distinct input locations to the same cache ABI. The remake must preserve explicit source provenance/order before it materializes slots, and must not derive either path from legacy `Fig`.

   The initial player source has one further closed edge: `JOIN` constructor `0x112a5(join_id)` writes `join_id` to both persistent record `+7` and `+8`; `0x33499` establishes `+8` as its character-ID lookup. Consequently a freshly joined player's raw FDICON key is its character ID when `0x10a77` later consumes `+7`. This is deliberately scoped to that writer and does not authorize a general character-ID/portrait/NPC selector alias or a default JSON fallback. It is mutable: class-change flow `0x314a7..0x3157a` writes its `0x31793`-resolved target byte to live roster `+7` after locating `0x53a45+slot×0x50`. The native persistence ABI is closed separately: post-handler `0x11506` matches by `+8` and copies the full 0x50-byte record runtime→persistent, so any native flow that invokes it persists `+7`. The remake now exposes optional editable `PartyMember.native_identity` and runtime `Unit.NativeIdentity`; sync uses this key only when explicitly present and skips unknown keys. Missing legacy fields retain the normalized Fig projection, so the path remains partial rather than byte-identical. The engine must carry an explicit raw key rather than infer it from identity fields.

   The party projection now applies this narrow player contract: fresh `PartyMember.Fig` seeds `BattleFig` and explicit `MapSelectorKey` only because it represents the verified JOIN identity; `campaign.ApplyClassChange` leaves stable `Fig` intact, updates `BattleFig`/raw key to its proven resolved target byte, and invalidates an old cache slot. Persistent overlay copies those split fields. This is state preservation only, not authorization to render arbitrary legacy `Fig` as native FDICON.

   Runtime selector bridge: `Scenario.ExecuteAction(spawn_party)` now materializes the party as one global-key batch; `AppendGroup` does the same for each later FDFIELD group, preserving party-first then group append order. A rejected editable batch records `State.NativeMapSelectorError` and preserves the legacy unit append, but disables native key resolution for the whole battle—there is no partial selector mix. The battle-unit draw path alone resolves `unit+2 slot → raw key` through that state cache; story/cutscene actors explicitly retain their editable `Fig` path. This is an exact selector adapter over the current PNG/Ebiten draw, **not** a claim that the native indexed buffer, palette branch, layer order, or HUD are now reproduced.

   `internal/fdicon` now decodes the 24×24 FDICON B24 container and preserves four-mode RLE transparency/dither plus both exact native blits: `0x4deda` raw indices and `0x4de56` opaque-index transform `(index&7)+0x18`. It has an original-asset 1680-sprite regression, but it is only an indexed asset primitive; no native frame schedule or UI handler is thereby enabled.

   `Bank.SpriteFor(key,pose,cycle)` enforces the recovered raw-key `key×12 + pose×3 + cycle` lookup inside one FDICON resource (pose 0..3, cycle 0..2). The renderer-facing formula remains `slot×12 + pose×3 + cycle`; `0x11019` performs the key→slot block copy. `NativeFrameIndex` captures the proven global idle/moving counters; battle `Fig` and `Dir` still provide only part of the runtime ABI, and no GUI integration is inferred.

   The raw/palette-band choice is also closed: `0x127e0` tests runtime unit `+5 bit7`; clear uses raw `0x4deda`, set uses `0x4de56` band. `fdicon.BlitForNativeFlags` makes that dependency explicit; it is not a guessed camp or LUT selection.

   Native placement is also represented as a pure byte-offset primitive: `NativePlacementOffset` preserves `0x127e0`'s `0x75d8 + (y-cameraY)*24*456 + (x-cameraX)*24 + unit[+4]*directionOffset` equation (pose 0/1/2/3 = down/left/up/right), with the `unit+0x26`-gated native 0-or-1 pixel shift. It deliberately does not claim an Ebiten coordinate or layer order.
4. **Campaign/postbattle**：逐關標記 battle end handler、town/shop/church/preparation/rest、persistent record append/reset、敗北路線；不能以章號順序推導。
5. **Native presentation**：完成 indexed off-screen/double-buffer、palette、透明 RLE、FIGANI/TAI/DATO compositing 後才接 Ebiten；任何 opaque segment 保持 fail-closed。

## 7. Milestones / gates

| Milestone | Deliverable | Gate |
|---|---|---|
| SDD-0 | 本文件、requirements matrix、證據分級、缺口清單 | 文件 review；無未標註的推測 |
| UI-1 | title→story→battle field→action menu→dialog→town/shop 的 shell vertical slice | command trace + headless tests + screenshots |
| UI-2 | battle target/range/end-turn/HUD 正確 | weapon/spell/menu RE evidence + differential tests |
| FLOW-1 | 全部 battle→postbattle→town/shop/church/preparation branches 可編輯 | 每章 transition matrix、save/reload regression |
| NATIVE-1 | indexed renderer primitives（含 ending／FIGANI／TAI） | byte/pixel regression；無 generic fallback |
| CONTENT-1 | ch01–ch30 script、資產、事件與結局完整 | campaign replay、無 load error、可編輯 round-trip |

## 8. Definition of done

- 所有 UI contract 有明確 state machine 和輸入測試；玩家不需要 debug key 才能完成基本流程。
- 所有戰後段落可在 campaign JSON 中看見；town/shop/rest/preparation 不被隱含吞掉。
- 所有 native 值有 E0/E1/E2 證據；E3 只存在於標註為 blocked 的文件，不進 runtime。
- Docker-only Capstone 與 Go regression 可重跑；`/tmp/fd2cap` 不存在，host Python 不安裝 Capstone。
- headless、畫面、存檔、reload、資源缺失 fail-closed 測試全綠；`git diff --check` 通過。

## 9. 2026-07-25 歷史決策（已 superseded）

當時曾凍結新的 handler／renderer 語意，先完成 UI evidence matrix；該凍結已由
後續 town/shop vertical slices解除。現行規則是：只有同時具備typed production
consumer、input/state trace、regression與適當E2 oracle的語意才能接入；孤立offset
或未證實高階名稱仍不得進runtime。

### 2026-07-26 — native phase dispatch raw boundary

Official Docker Capstone recheck of `0x1d80b` closes only its first loop's admission boundary: records are addressed as `[0x53a45] + unit*0x50`, bounded by `[0x53beb]`; a candidate must satisfy raw `+6 == 1`, `(+5 & 0x81) == 0`, and `+0x26 == 0`. The native caller then passes `(unitIndex, record+6)` to `0x13a9f`, which may set `[0x51a8f]` before the event and chapter function tables are called and `[0x53ecc]` is checked. `fdother.FindNativePhaseDispatchCandidates` preserves this as an offset-level, fail-closed planner. It intentionally does not invoke callbacks or assign event names; no campaign node may treat it as a completed phase/event renderer.

2026-07-29 的合法 IDA Pro 9.4 複核補正了這段歷史邊界：兩張表的尾段不是
只跟在「候選」後方，而是每一筆 record 都會到達；第二遍也會重新讀取
第一遍可能改寫的 `record+5 bit7`。因此靜態候選 plan 只能作診斷，
`fdother.ExecuteNativePhaseUnitScans` 才保存逐筆重判、90／30 筆表界、
固定回呼順序與 pending 提前退出的 E0 契約。各 handler 效果仍由呼叫端
提供，尚未接入正式 AI runtime。

The default chapter result boundary is now corrected at direct-instruction level. The
`0x205b4` function, also entered directly at its shared `0x205be` inner entry, first
writes `[0x53ecc]=2`, scans `[0x53a45]` for any record with raw `+6==0` and
`(+5&1)==0` and writes 0 when found, then lets record zero's `+5 bit0` overwrite the
result with 1. `battle.NativeBattleResultCode205B4` preserves that exact order and returns
only the numeric code. It does not name camp0, bit0, code1, or code2 as a gameplay outcome.
The adjacent `0x205da` is a separate reset/loader entry: `0x205d5` jumps directly to
`0x2067e`, while `0x205da` has its own direct callers and is the routine that clears globals
and calls `0x1088d`. Historical notes that described `0x205be` as clearing the result and
loading a chapter are invalid. Official IDA 9.4 groups `0x205be` into function
`0x205b4` and independently reproduces the same loop and overwrite order. A production victory/defeat transition still requires the
chapter handler, outer `[0x53ecc]` consumer, editable campaign node, and player-path evidence.

The same audit rechecked `0x1b8e7` because it is shared by class-change, item, and post-resolution callers. Official IDA 9.4 decompiles a uniform `sub_1B8E7(int unit, int slot)`: `memmove(record+0x0a+2*slot, record+0x0c+2*slot, 2*(7-slot))`, followed by `record+0x18=0x80`. `battle.RemoveNativeInventorySlot` now reproduces this byte-level removal, including the native stale item byte in the final cell. The previous claim about an unresolved third stack argument was incorrect and has been removed; higher-level callers still decide why a slot is removed.

The shared upstream `0x13a9f` is bounded at its raw mode boundary: after `record+5 & 5 == 0`, it reads `mode=(record+0x34)&0x0f`, bytes `+0x35/+0x36`, and byte `+0x3d`. `fdother.PlanNativeUnitMode` records those values plus the caller's unit/second argument and performs no callee invocation. Mode branches remain evidence-only; no battle, town, shop, or event meaning is assigned from the mode number.

The item-action effect dispatcher `0x20c6f` is transcribed at call-topology
level. It reads item-row byte `+0x0d` and word `+0x0e`; observed raw branches
are: `5/13→0x211a4`, `6→0x22af6` subcommand `0x14`,
`7→0x22af6/0x15`, `8→0x21082/0x37`, `9→0x21082/0x39`,
`10→0x21082/0x3e`, `11→0x1c4cc→0x1c2da` loop, `12→0x22997`,
`14→0x22d1b/0x1b`, `15→0x22866`, `16→0x22721`,
`17→0x21082/0x42`, `18→0x21082/0x46`, `19→0x21082/0x3b`,
`20/24→0x1c4cc→0x1cd17` loop, `21→0x2111a`,
`22→0x22d1b/0x16`, `23→0x2218a`. `NativeItemEffectRouteForType`
preserves the whole raw map; typed closures now supersede its opaque boundary
for 5/13 (HP restore with consume/retain split), 6/7 (consumable record-marker
clear plus HP restore), 8/9/10 (permanent base
AP/DP/DX), 11 (consumable MP restore), 12 (retained HIT/EV +15),
14/22 (retained marker application with randomized HP damage), 15/16
(retained derived DP/AP modifier), 17/18/19 (consumable max HP/max MP/MV
increase with type19 preserving EXP), 20/21/24 (retained reuse of a
row-selected native command damage record with distinct presentation paths),
and 23 (retained direct relocation with command23 MP debit). All observed
effect branches 5–24 now have typed post-confirm contracts; this does not
mean the item selector UI or indexed presentations are integrated.

Official IDA 9.4 also closes the small presentation helper `0x1e0db(value, digitBias, target)`: after a camera-bounds check it formats `value` as four decimal digits and appends four raw queue entries with position codes `2,7,12,17`, target index, and digit bytes; `0x1e1dc` writes a parallel four-byte queue from a global raw source. This is a presentation-queue ABI, not proof of HP/MP/damage/heal semantics. The adjacent `0x1debe(actor,x,y)` gate only checks active state, Manhattan adjacency, and equipped row byte `+0x0b <= 1`; it must not be promoted to a universal weapon max-range rule.

The remake has a data-only `battle.AppendNativePresentationDigits` adapter with right-alignment, bias, camera no-op, and raw position-code regression coverage. It deliberately stops before renderer, palette, SFX, or gameplay naming.

Official IDA 9.4 decompilation of the shared type-6/7 callee
`0x22af6(a1..a5)` iterates target indices from byte list `a4` and reads the
marker from the target runtime `record+a5`. A prior adapter incorrectly
modeled this as a parallel caller-owned `flags[]`; that assertion and API are
removed. A nonzero marker calls `0x1c916(target,10)` (base10 yields actual
9 HP restore), clears the same record byte, and accumulates
`4*effective(+0x21)`. Type6/7 select `a5=+0x25/+0x26`, then both jump through
`0x1b8e7` to consume their source slots. `ApplyNativeItemMarkerClearRestore`
preserves record-local marker mutation, sequential RNG, list order and atomic
source preflight; tracked IDs196/197 supply these routes. Status names and
presentation remain unknown.

Official IDA 9.4 decompilation of common callee `0x21082(a1..a7)` closes the corresponding word-write path: it reads one unit index from `a6`, adds the low 16 bits of `a2` to the word at `record+a3` (native wrap semantics), then calls `0x1b8e7(a1,a4)` to compact/remove the caller's inventory slot. `battle.ApplyNativeWordDeltaAndRemove` preserves explicit target/removal units, raw word offset, signed delta representation, and atomic bounds validation; it does not name the word or run renderer/effect callbacks.

The growth-marker callers around `0x22721/0x22866/0x22997` use `0x4e893`: 16-bit `rol3(state+0x9014)`, then `idiv 4` and the **remainder** in `EDX`, followed by `+2`. `fdother.NativeRNGStep`/`NativeRNGMarker` preserve this state transition and marker source. Any earlier interpretation as quotient-based growth marker is invalid and must not be used.

The shared state used by `0x4e893` is word `0x627b8`. LE fixup enumeration
finds exactly two references, the helper's own load and store. The address is
inside object 3's initialized pages and its executable image bytes are
`0x0000`; no save/load or chapter routine references it. Therefore its
verified lifecycle is process start at zero, then continuous mutation for the
life of that process, with no `FD2.SAV` persistence. The remake keeps this
`uint16` state separate from Go's normalized `math/rand` stream.

The Ebiten item action now commits the closed HP/MP families after the same
two-stage row-derived target validation: types 5/13 call the native HP restore
transaction and type 11 calls MP restore, preserving target-list order,
per-target RNG consumption, type-specific source retention/removal, compact
inventory cells, raw `+5 bit7`, and end-of-action state. Every runtime unit
must materialize a complete proven 0x50-byte item record before mutation; a
missing record rejects the entire transaction. Indexed effect presentation
and the remaining item families are still fail-closed.

Official IDA 9.4 decompilation of `0x22721(a1,count,indexBytes)` closes the first raw growth writer: it skips records whose `+0x22` marker is nonzero; for each zero marker it advances the shared RNG, writes `+0x22=(rng%4)+2`, computes `trunc(word(+0x48)*0.15+1)` using the native toward-zero `_CHP` helper, adds that low-word increment to `+0x48`, and accumulates `2*effective(+0x21)`. `battle.ApplyNativeRawWordStep` reproduces this batch mutation and score while leaving presentation and the `0x1317d` tail outside the adapter. It does not call the function for already-marked records or consume RNG for them.

The adjacent `0x22866` branch is byte-for-byte the same arithmetic with marker `+0x23` and word `+0x4a`; `battle.ApplyNativeRawWordStepAtOffsets` shares the implementation and regression without assigning either field a gameplay name.

The neighboring `0x22997` branch is a separate fixed-pair mutation: marker
`+0x24` is gated; successful units advance the same RNG and add `0x0f` to
derived HIT/EV `+0x4c/+0x4e`, then contribute
`2*effective(+0x21)` raw score. The type-12 item caller passes its final target
list to this helper and then goes directly to cleanup without `0x1b8e7`, so
the source is retained. `NativeItemHITEVStepRoute` /
`ApplyNativeItemHITEVStep` capture marker-gated RNG, 16-bit wrap, typed
HIT/EV increment and retention; tracked ID210 supplies this type. Presentation
and marker display name remain outside scope.

The `0x22d1b` path is separate from the word/pair modifier family. Its loop
skips a nonzero marker or class `0x19/0x1a`; otherwise the first RNG remainder
must be `<50`. It then calls `0x1c81f(unit,10)`, which consumes a **second**
RNG and applies `10*9/10 + (rng%100)*10/1000 = 9` HP damage, before a
**third** RNG writes `(rng%4)+2` to the marker. The earlier “two RNG draws /
fixed 10 HP damage” statement was incorrect and is removed.

`ApplyNativeRawApplication` preserves this three-draw sequence. Typed item
callers 14/22 select marker `+0x26/+0x27`, retain their source slots, and are
exposed by `ApplyNativeItemMarkerApplication`; tracked rows are ID212/57.
The marker/status display name and presentation remain fail-closed.

Official IDA 9.4 decompilation of `0x22253` closes the command-23 state write: after its indexed renderer work, it writes the supplied final `a13/a14` bytes to `record[+0]/record[+1]`. Caller `0x2218a` passes `0xff/0xff` as the pre-render pair and cursor globals as the final pair. `battle.SetNativeUnitCoordinateBytes` preserves only this raw write; movement pathfinding, camera, indexed presentation, and cursor semantics remain separate.

The item-type23 caller is now closed separately. `0x1bbdc` admits it only
when actor raw identity `+0x08==24` and max MP word `+0x46>=20`; the older
“class/level gate” label was wrong. After mode-6 destination confirmation,
`0x2218a` uses only the first target byte, calls `0x1ca89(actor,23)` (native
16-bit current-MP subtraction using command23 record cost20), and adds
`10*(target level + 30 when class is 9..24)` to the raw accumulator. It then
uses `0x22253` for the `0xff/0xff` exit and cursor-coordinate entry writes.
The item dispatcher does not call `0x1b8e7`, so tracked item ID101 is
retained; its row word1 is ignored by this handler.
`NativeItemRelocationRoute` / `ApplyNativeItemRelocation` preserve this
post-confirm state transaction, first-target behavior and MP wrap. Literal
target code 6's destination predicate is now executable too. Apart from the selected
target, any record at the destination with raw `+5 bit0==0` blocks admission.
The 29×20 table returned by `0x4e555(selector)` is exported as editable
`native_movement_cost_rows.json`; selector normally uses target class `+0x20`,
is overridden to 1 for `+7==0x1c`, or to 19 for the recovered `0x1f183`
class/race gate. The resolved terrain index must contain literal value20.
`NativeRelocationDestinationAllowed` preserves those gates and rejects
malformed tables/counts. Terrain-index production through `0x12e38` is already
the raw cursor/FDSHAP resolver boundary. Ebiten now keeps the selected first
target and opens a distinct destination cursor, admitting only cells accepted
by `NativeRelocationDestinationAllowed`. Both first-target cancel and
destination cancel return directly to the caller-owned item panel; the older
destination-to-first-target behavior was a remake invention and is removed.
2026-08-22 runtime correction: destination confirmation先在私有records驗證
command23 MP subtraction及raw coordinates，並要求精確逐格
`NativeTerrainMoveCodes` provenance；之後依序執行八段palette與兩次完整
27-present indexed renderer，第二次完成後才原子發布交易。任一前置缺件都在
第一個sample／frame前失敗。原版同狀態camera、逐幀與逐音訊E2仍是獨立驗收門檻。

The surrounding selector/grid lifecycle is caller-closed. Item target entry
writes the global selector as `row[+0x12]+2`, while its first target field is
built from `row[+0x10]`, the type-23 inner marker, and `row[+0x15]`.
Immediately after the first `0x115b6` returns, `0x4dbfc` resets all runtime
cell byte `+3` values and the global selector returns to `1`; the final target
list is then built from `row[+0x12]` and reset again. Type23 subsequently
passes literal target code `6` to `0x115b6` while the global selector remains
`1`: it does **not** materialize global selector 6. The indexed compositor
does preserve the separately verified global-selector-6 mutation primitive
(terrain draw, selected cell byte `+3=0`, then foreground), but its production
owner is a different, still-unintegrated battle presentation path.

2026-08-22 畫面所有權更正：物品第一階段目標欄位已由上述 row 欄位建立，且
正式 `0x11CAC` 索引組合器可消費可繪製的 global selector 1..5；這一層屬於
`RUNTIME-E1`，不等於物品效果演出或一般玩家 E2。舊的正規化地圖曾在原生
組合器失敗前另畫綠色目標、青色傳送目的地與橘色 command 0 半透明方塊；三者
都沒有原版畫面證據，且會在原生素材缺失時留下看似成功的替代介面，因此不再
作正式後備畫面。完整 raw 欄位、HUD、LUT 或 range sprite 任一缺失時，畫面與
交易都維持失敗即關閉；物品效果的索引演出、不可用目標外觀、取消鍵語意及
global selector 6 的真正 production owner 仍列為未知。

2026-08-22 一般玩家窄證據：以未修改 `FD2.SAV` 從標題 CONTINUE 進第一戰，
四名我方角色 HP 都等於 MaxHP；索爾在正常物品面板選到 item 192 草藥
（`HP 040`）後按 Return，0.3、1.3 與 4.3 秒都仍留在同一物品列，沒有進入
地圖 target modal。這只把「type 5 HP 回復物品在已證實候選中沒有任何
`HP < MaxHP` 角色時，保留 caller-owned item panel」提升為窄 `PLAYER-E2`。
重製入口應在發布 selector／清除面板前完成此 gate；type 13、MP／marker／能力、
傷害與 relocation 物品不可由本實驗外推。擷取、雜湊與裁切像素檢查見
[`native-item-herb-fullhp-original-e2.json`](../data/ui-traces/native-item-herb-fullhp-original-e2.json)。

Caller-scope correction (Docker Capstone, 2026-07-26): `0x22253` is shared by the chapter-ending/post handler at `0x250cc`, not command-23-only. That path calls it after `0x1c2da` with unit index `1`, pre-render bytes `0xff/0xff`, and the selected record's raw `+0/+1` bytes, then continues to `0x25089` cleanup and `0x2bce5` ending rendering. The remake therefore treats `SetNativeUnitCoordinateBytes` as a shared raw writer only; command-23 selector, ending layout, renderer, and campaign transition remain independent fail-closed contracts.

The `0x25348` branch audit further fixes the ending-only order: FDOTHER frames `0x0d`, `0x0e`, `0x0f` are presented around `0x1c2da`; the shared `0x22253` write for unit `1` follows with raw `+0/+1`, frame `0x10` follows, and then `0x25089→0x2bce5` enters the terminal self-loop. This is call-order evidence only. The `0x24b14` return and frame IDs remain unnamed, and this branch must not be used as a generic battle→town/shop transition.

The ch26 item gate is now closed as a raw read-only primitive. Docker Capstone shows `0x24b14(item)` scanning units `0..15`, while `0x31860(unit,item)` first calls `0x1b8a6` and then compares only raw slots `0..count-1` through `0x1b722` at `record+0x0b+2*slot`; it does not independently verify that those slots are occupied. `battle.FindNativeInventoryItemInUnit` and `FindNativeInventoryItem` preserve that count-sized scan, return the first raw `(unit,slot)` for an editable gate, and never remove or mutate a cell. `battle.NativeInventoryRecords` now materializes only the proven `InventorySlots` + `NativeInventoryFlags` cells for a complete 16-unit runtime roster; campaign `partyHasItemID` uses this raw gate when provenance is complete and retains normalized inventory only as an explicit fallback. The native `0x24b14` success/failure result is therefore not a recipe, reward, camp filter, or item-consumption proof; ch26's later success/missing presentation remains a separate handler branch.
Portrait text correction (official IDA, 2026-07-26): the epilogue selector at `0x2c8f7..0x2c8f9` is controlled by the outer `edi` portrait-loop counter, not a bitwise `|45` expression. It is `unit[+8]+0x0c` while `edi < 0xdc`, then fixed current-FDTXT index `0x2d`. `Montage.PlanPortraitText` preserves this exact branch and rejects short unit records; no text renderer is inferred from the mapping.

Persistent identity lookup is separately closed at `0x24bde`: Docker Capstone shows a caller-supplied count loop over the persistent `[0x53bf7]` array, stride `0x50`, comparing only the unsigned byte at record `+0x08`, with native boolean success/failure. `battle.FindNativePersistentIdentity` preserves the first raw index, explicit count/capacity validation, and read-only behavior. This is an identity-table primitive only; it does not rename `+8` as portrait, Fig, NPC, or a general character alias.

The adjacent `0x24d22(arg)` boundary remains evidence-only. IDA Pro 9.4 and
Capstone show `arg!=0` writing only its low byte to raw global `0x51a10` and
returning; `arg==0` instead allocates `latch*0x138` bytes, copies from
`0x53aff + (0xc0-latch)*0x138`, then copies rows in descending order
(`0xbf-latch` down through `0`) before a final `0x138`-byte copy and
`0x37416` free. The ch23 caller only takes the non-zero setter branch for
stages 2..14. Direct data xrefs for `0x51a10` remain local to the helper, but
the `dword_53C03==23` branch of `0x11eee`, reached by `0x11cac`, compares
`[0x46c]` with `[0x539f8]` at `0x120a7` and calls `0x24d22(0)` at `0x120af`
only after a BIOS-tick change, then stores the new tick at `0x120b9`. This
proves the exact tick gate and an indirect consumer and
supports the bounded annotation “312-byte staging row-rotation raw latch”;
it does not justify a generic frame, palette, or UI name. IDA also shows
`0x11cac(1)` and `0x11cac(0)` share an indexed chain through
`0x11eee`／`0x122dc`／`0x127a9`／`0x1acf3` and a 456-stride `0x11eb0` copy,
with only the zero argument calling `0x4dfcc`; its IDA `BYTE1(v2)=-32` and
Capstone `mov ah,0xe0` fix the palette write window at DAC indexes `0xe0..0xef`,
not the previously stated low-index window. The direct caller still does not read
`0x51a10`. The same ch23 audit fixes `0x17aa9(1)` as a DOS BIOS
tick wait and `0x11d40(0,255,ESI)` as twelve register-bound full-DAC updates per
outer stage, five stages total (`ESI=0..59`; no reset at the stage boundary). The
remake now has raw `RotateNativeCh23Rows` and
`ApplyNativeCh23PaletteCycle` primitives, plus two strictly validated
`native_ch23_loop` candidate beats that preserve every loop call site, the
30/12 repetition counts, register-shaped arguments, and stage values 2..14.
The runner now requires an explicit raw-latch callback for every non-zero
`0x24d22(stage)` setter and deliberately does **not** rotate rows by itself;
the `arg==0` copy belongs to the BIOS-tick-gated shared consumer. It therefore
fails closed while the raw state/latch adapter is absent, instead of turning a
handler schedule into an inferred renderer. The same
case-23 branch passes `0x53aff`, row stride `0x138`, and `0xc0` rows to
`0x11eb0`。重新核對 `0x10652..0x1088d` 後，raw 載入器的擁有者邊界已收窄：
當 `[0x53c03]==0x17` 時，`0x107dd` 精確配置 `0xea00` bytes 並寫入
`[0x53aff]`；`0x107ef..0x10804` 保留資源值 `0x1a4d`、raw 參數／索引
`0x2a` 與傳給 `0x111ba` 的既有 `[0x53b03]` handle；`0x10809..0x10820`
保留 `0x4e63d` 的原始堆疊形狀，接著 `0x10823..0x10831` 釋放並清零
`[0x53b03]`；`0x1083b` 呼叫 `0x24d22(0)`。這證明 raw ch23 staging
配置與清理邊界；IDA 的同函式資料表已將該載入值對到 `FDOTHER.DAT` archive
entry #42（輸入檔雜湊見 [`fd2-reference-files.json`](../data/fd2-reference-files.json)），
但不以推測名稱取代 #42 的 raw identity。`0x11EB0` 的共用尾端已證實將
312×192 複製到固定線性目的地 `0xA0504`。固定版二進位的
`byte_51A10` raw seed 是 `0x01`，但 `0x24c1e` 在每個內層 draw 之前已先
寫入 stage 2..14；因此舊規格把 handler 入口 latch 列為阻擋是錯誤斷言，
現已撤銷。`0x11cac` 內的共用 indexed 消費鏈是 `0x11eee→0x122dc→
0x127a9→0x1acf3→0x11eb0`，ch23 直接 call-site 是 `0x24c63` 與
`0x24cd3`。這是靜態 E1 consumer 證據；production adapter 與一般玩家 E2
仍是不同閘門。完整原始證據見
[`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt).

IDA data-xref 與函式資料流稽核在不替欄位命名的前提下收窄擁有者邊界：`0x53aff` 由
raw `0x10652..0x1088d` 載入器配置／清理，再交給 `0x11eee`／`0x24d22` 消費；
`0x51a10` 只在 `0x24d22` 內讀寫；`0x539f8` 只由 `0x11eee` tick gate 比較／
寫回。offset globals `0x53aed`、`0x53af1`、`0x53af5` 則由共用函式
`0x12eaa`、`0x1300d`、`0x13185`、`0x13315` 寫入，包含明示的 `0x18`／零值
哨兵與累加更新。2026-08-21 進一步證實這些值只在四個移動／呈現函式內暫時
寫入，每個函式在返回前都由 `0x12ff4` 或 `0x13153` 共同尾端清零；
`sub_24C1E` 既不在這些函式內，也不呼叫它們。因此只對 ch23 handler 的
quiescent loops，三個 offset 都以零進入 `sub_11EEE`。這項結論不授權建立
camera、viewport、framebuffer 或 `nativeMapWork` 高階別名，也不可外推到
角色移動中的其他 caller。

The narrow executable boundary is now represented by
`fdother.DecodeNativeCh23Stage`／`BlitNativeCh23Stage`: they require archive
entry #42, exact 312×192 geometry, a `0x138`-stride `0xea00` staging surface,
and transparent `0x4e63d` blitting, while leaving the indexed state/latch
adapter outside the production handler.

### raw ch23／玩家第 24 戰戰後 indexed adapter 規格（2026-08-21）

本節是 `postbattle_ch24_persist` 的實作閘門。它只使用上述已證實 raw contract，
不再重開入口 latch，也不把 DOS BIOS 時刻或三個 transient offset 猜成新資料欄位。

1. **具型別輸入**：只接受固定雜湊的 `FDOTHER.DAT` entry #42、單幀
   312×192、`0x138` stride／`0xea00` staging；目前戰鬥必須保有完整
   `nativeMapAssets`、`NativeMapViewState`、HUD、selector cache、
   `0x25680` work、320×200 VGA 與 768-byte DAC。任何一項缺少或形狀錯誤，
   在改動 campaign、party、clock、buffer 或 palette 前拒絕整拍。
2. **初始 staging 與 latch**：entry #42 先經已證實的透明 blit 建立私有
   staging。每個 outer stage 必須先執行 setter，再開始 inner draw；第一段
   stage 2..9 各 30 draw，第二段 stage 10..14 各 12 draw。固定 seed `0x01`
   不進正式狀態，也不得改變 stage-before-draw 順序。
3. **BIOS tick gate**：adapter 保留一個獨立的 signed low-word snapshot，
   每次 draw 使用既有 `nativeBIOSClock` 取樣。只有本次 raw tick 與 snapshot
   不同時，才以當前 latch 呼叫 `RotateNativeCh23Rows`，成功後將 snapshot
   更新成該 raw tick；相同時不旋轉。第一個 draw 的結果由同一 gate 決定，
   不硬編成一定旋轉或一定不旋轉。每個 draw 後的 `0x17aa9(1)` 仍以一個
   BIOS tick 的阻塞間隔推進；60 Hz 顯示只負責承載等待，不取代 raw tick 比較。
4. **indexed composition**：每次 draw 先把 staging 以 312-byte source stride
   複製到 work `+0x8088` 的 456-byte destination stride；三個 raw offset 對本
   handler 固定為零。接著沿既有 `ComposeNativeFrame` 執行 terrain→range→
   unit／foreground→HUD→312×192 viewport copy，保留透明 terrain 對 staging
   的實際消費，不用 RGBA 疊圖或靜態 PNG 近似。
5. **DAC 順序**：第一段 `0x11cac(1)` 不執行 ch23 的額外 palette cycle。
   第二段每個 draw 先用固定 baseline DAC 計算
   `max(component-rawESI,0)`，`rawESI` 從 0 連續到 59；再依 `0x11cac(0)`
   呼叫 palette cycle。該 cycle 另保留 `word_60000` snapshot：只有
   `uint16(rawTick-snapshot) >= 2` 時，才把 process-local 0..15 phase 加一、
   將 raw table `0x60003` 的 16 組 RGB 覆寫到 DAC `0xe0..0xef`，並更新
   snapshot；未達門檻時不覆寫。不得把五個 stage 的 ESI 各自重設、把每次
   draw 誤當一次 phase advance，或把高位 palette window 改成低位索引。
   handler 入口的 `word_60000`／`byte_60002` 精確值尚無一般玩家快照；E1
   沿用程序內相對 phase，並以 adapter 啟動時已 materialize 的 BIOS tick
   初始化 timer snapshot，明列為 E2 差距，不假稱首相位精確。
6. **非同步、原子與存檔邊界**：演出由單一 blocking job 逐次發布 draw，期間
   鎖住玩家輸入與 cutscene save；資產在第一個可見步驟前完整 preflight。
   後續任何 compose／palette／buffer 錯誤都回復 staging、work、VGA、DAC、
   native clock／tick snapshot、palette phase 與 beat cursor，且不得執行
   `sync_party` 或 chapter increment。
7. **戰役 handoff**：兩段 loop 完成後才允許 handler 的 `sync_party` 與
   `set_chapter(24)`，正式節點為
   `battle_ch24→postbattle_ch24_persist→preparation_ch25→story_ch25`。
   `preparation_ch25` 是存檔允許的 node boundary；不可誤接成
   `preparation_ch24`，也不可直接跳到下一場戰鬥。
8. **驗收**：E1 至少覆蓋 stage/setter-before-draw、240＋60 次 draw、每次
   一 tick wait、ESI 0..59、tick 相同／變化兩臂、row wrap、零 offset seed、
   DAC 操作順序、錯誤回復、cutscene save 拒絕、party sync 與
   `preparation_ch25` save/load。這些回歸通過後可建立 production binding；
   未修改原版同狀態影像／時序仍另標 PLAYER-E2，不反向阻擋已證實的 E1。

**2026-08-21 正式執行期結果（RUNTIME-E1）**：上述規格已由
`native_ch23_post.go`、`SeedNativeCh23Staging` 與 production binding 消費。
決定性測試覆蓋 setter-before-draw、240＋60 次 draw、相對 BIOS tick、兩拍
palette gate、ESI 0..59、失敗原子回復與 callback 邊界；正常戰役回歸另從
`battle_ch24` 的戰果確認進入 `postbattle_ch24_persist`，保留 map23 的70筆
FDFIELD＋16筆 LOADCH runtime，完成 `sync_party` 後進
`preparation_ch25` 並驗證存檔／讀檔。這解除的是正式 E1 gate；原版 handler
入口的程序相位、同狀態逐幀影像與實際時序仍屬 PLAYER-E2，不得寫成逐像素完成。

相鄰的 ch22 `0x2189a` 稽核遵循相同的非破壞性規則。IDA data-xref 顯示 caller
直接讀取共用 `0x53a49`／`0x53aa9`／`0x53aad`／`0x53a6d`，但沒有在此函式
寫入；同一批 globals 也被多個地圖、角色、轉場與 indexed caller 消費。
`0x53b03` 仍是 raw resource loader handle，`0x53b07`／`0x53b0b` 則由共用
呈現函式寫入，再由 indexed 路徑讀取。這只是擁有者邊界證據，不授權把任何
欄位別名成 camera、portrait、effect 或 framebuffer；`native_2189a_loop`
因此仍維持失敗即關閉。

The adjacent `0x24e80` handler contains one independent raw mutation loop: for runtime indices `0x10 <= i < caller_count`, records with byte `+0x07 == 0x1f` receive `+0=0x10` and `+1=0x06`. `battle.RewriteNativeMarker1F` preserves the explicit start/count, matching-marker-only writes, and bounds validation. The bytes remain unnamed and are not treated as roster identity or renderer state.

The mutation core after a selected `0x11506` pair is independently transcribed: copy runtime→persistent `0x50` bytes; clear persistent bytes `+0x22..+0x27`; mask persistent byte `+0x05` with `1`; if the result is not `1`, copy word `+0x42` to `+0x40`; always copy word `+0x46` to `+0x44`. `battle.ApplyNativePersistentRecordCopy` implements only this raw core with atomic bounds validation. The preceding `0x3453e` zero-identity/inactive gate and trailing `0x1145a` call remain outside the helper and outside remake sync semantics.

The preceding `0x3453e(index)` predicate is now closed independently: it returns exactly `record[index*0x50+0x05] & 1` and performs no write. `battle.NativeRecordByte5Bit0` preserves this mask/bounds contract without labeling the bit as acted, alive, or active; callers must still prove their own higher-level gate.

   The `0x1145a(persistentIndex)` tail is also data-flow closed: it starts signed base words at record `+0x37`, `+0x39`, and `+0x3e`; scans eight cells; only a cell flag with bit `0x40` set causes its item byte `+1` to index `0x4e56c`; effect words `+1`, `+5`, `+3`, and `+7` accumulate into raw destinations `+0x48`, `+0x4a`, `+0x4c`, and `+0x4e`. All 215 materialized rows now cross-check those little-endian words against normalized AP/HIT/DP/EV without a mismatch, fixing native accumulation order as AP/DP/HIT/EV. `battle.ApplyNativeEquipmentRecalc` preserves the raw operation with atomic bounds validation and 16-bit wrapping. Existing `campaign.RecomputeEquipment` remains a normalized projection; the cross-check closes these four equipment words, not the remaining effect fields or complete campaign byte identity.

The shared item-row helper `0x4e56c(item)` is now bounded at the proven arithmetic boundary: it returns a pointer at linear table base `0x602ad + item*0x17` (23-byte rows). Byte comparison fixes the corresponding EXE file view at `0x540ad`, one byte after the normalized/guide exporter view at `0x540ac`; each raw row consequently ends with the next normalized row's leading byte. `native_item_effect_rows.json` now materializes the 215 known selectors as a byte-exact prefix, and `LoadNativeItemEffectRowPrefix` enforces consecutive IDs and exact 23-byte rows. `battle.NativeItemEffectRowOffset` exposes only the table-relative offset for a byte-sized selector. The fixture is not proof that the native table ends at ID 214, and unnamed row fields remain disconnected from normalized `ItemStats`.

The type 8/9/0xa branch of `0x20c6f` is now closed beyond raw topology. The
`0x1145a` base/equipment data flow plus the 215-row raw-vs-normalized
cross-check identifies persistent offsets `+0x37/+0x39/+0x3e` as base
AP/DP/DX. Each branch passes row word `+0xe` to `0x21082`, permanently adds it
to the corresponding base stat, calls the proven synthesis path, and removes
the source slot. Known raw item IDs 198/199/200 carry amounts 9/9/7.
`battle.NativeItemWordDeltaRouteForType` exposes a typed AP/DP/DX contract;
presentation selectors and item labels remain outside this closure. The
shared callee's type17–19 routes are independently closed rather than
inheriting these base-stat labels.

Type17/18 pass row amount20 to max HP `+0x42` / max MP `+0x46`. Type19 passes
amount1 to word `+0x3b`; its caller saves byte `+0x3c` before `0x21082` and
restores it afterward. Existing class-change provenance identifies `+0x3b`
as MV and `+0x3c` as EXP, so the net operation is MV-byte +1 with EXP
unchanged. All three paths consume their source slots inside `0x21082`.
`NativeItemCapacityStepRoute` / `ApplyNativeItemCapacityStep` preserve these
typed mutations, type19's cross-byte save/restore, and atomic removal;
tracked IDs94/95/96 fix the amounts.

The `0x211a4(actor,count,targetBytes,amount)` ABI is now closed by official IDA 9.4 pseudocode: item-action caller
`0x20c6f(a1,a2,a3,a4)` passes `a3/a4` unchanged as count/list and supplies item-row word `+0x0e` as amount. The callee enters `0x1c4cc`/`0x1c2da`
with raw subcommand `0xd`, then iterates the byte list in order and calls
`0x1c916(target,amount)` before `0x1e0db`.
Canonical Capstone also finds a second direct caller, `0x285ed`, outside the
item dispatcher: under opaque selector `0x21` it prepares a byte list, passes
raw amount `0x320`, and reuses the same helper.  Therefore this is a shared
list-driven raw mutation/presentation primitive, not a type-5/13-only
function. That shared-callee fact does not erase the item caller's semantics:
`0x20c6f` type 5 and 13 both supply row word `+0x0e` to the proven current-HP
`+0x40` / max-HP `+0x42` restore path; after return, type 5 alone jumps through
`0x1b8e7` to consume its source slot, while type 13 goes directly to cleanup
and retains it. `battle.ApplyNativeItemHPRestore` preserves the sequential
RNG/mutation/score loop, full target/source preflight, and this consumption
branch. Item display names and presentation asset provenance remain
fail-closed.

Official IDA 9.4 pseudocode now closes the previously opaque presentation callers without naming their gameplay effect. `0x1c4cc(a1, subcommand, count, targetBytes)` copies three 33-byte global frame tables, snapshots the indexed 456-stride buffer, iterates `frame < frameCount[subcommand]`, selects a frame from `frameBank[subcommand]`, redraws each supplied target's visible 24×24 cell when it is inside the camera bounds, presents the 312×192 viewport, and emits only the observed subcommand/frame-specific SFX branches before a BIOS-tick wait. `0x1c2da(a1, subcommand, count, targetBytes)` starts the same presentation family with SFX index 1, redraws each target through the indexed pointer bank selected by `12*unitVisual + currentCycle` (with the native `cycle==3` remap), then performs five restore/present pairs before returning the saved buffer. `0x211a4` calls both with raw subcommand `13` before the per-target `0x1c916` mutation. This closes the caller ABI, frame ordering, camera bounds, and restore cadence only; the amount's gameplay meaning, upstream target-selection policy, SFX labels, and native renderer asset provenance remain opaque and fail-closed.

The same IDA pass closes the type `20/24` presentation loop at `0x1cd17(a1, subcommand, count, targetBytes)`: it copies a 30-byte frame-remap table, runs exactly ten frames, restores the saved indexed buffer before every frame, redraws each camera-visible target through `0x4dc34` using `7-(frame mod 8)` as the raw blend argument, presents the same 312×192 viewport, waits one BIOS tick, then restores the original frame. This is a distinct ten-frame path from `0x1c4cc`/`0x1c2da`; the presentation body itself performs no gameplay mutation. The later caller closure below proves the separate row-selected command-damage loop. The compatibility predicate used by the item selector is also exact at `0x1c1c3(actor,item)`: `0x4e53e(actor class)` supplies six raw item bytes and the predicate compares item-row byte `+0` against those six entries. The six-byte table and row byte remain opaque inputs; no class/equipment name is inferred.

The table provenance is now closed at `0x4e53e(class)`: it returns `0x6188a + class*7`, so the selector's six-byte comparison consumes bytes `row+0..row+5` and leaves `row+6` opaque. `battle.NativeClassCompatibilityRowOffset` and `NativeClassItemCompatible` preserve this address/length contract with bounds and short-row rejection. This is a raw selector adapter only; it does not expose a normalized class or item compatibility field.

The shared `0x1c916` HP mutation core is separately executable as
`battle.ApplyNativeRawHPRestore`: it advances the Docker-verified 16-bit RNG,
applies `amount*9/10 + (rng%100)*amount/1000`, clamps current HP `+0x40` to
max HP `+0x42`, and preserves the native raw score gate (`record+0x07 < 0x4b`,
class byte range `9..24` adds `0x1e`, score factor
`40*effective*delta/max`). The primitive alone does not imply an item; the
closed type5/13 caller contract above supplies that scope.

The sibling `0x1c9dd` MP mutation is captured by
`battle.ApplyNativeRawMPRestore`: the same RNG and amount arithmetic writes
current MP `+0x44` capped by max MP `+0x46`, while its score uses only
`record+0x21` (no HP routine's class-range bonus). The type-11 item caller is
now closed around that primitive: it skips a target with zero max MP without
advancing RNG, restores remaining targets in list order, then consumes the
source slot via `0x1b8e7`. `ApplyNativeItemMPRestore` preserves the whole
atomic transaction; tracked IDs206/207 supply amounts 80/200. Presentation
and display names remain outside this closure.

Types20/21/24 all pass the item-row word as the command ID to
`0x1c75e(target,commandID)` for every target and queue the numeric result
through `0x1e0db`. Types20/24 use the ten-frame `0x1cd17` presentation;
type21 reaches the distinct indexed `0x1cac7` helper through `0x2111a`.
Neither path mutates gameplay state before the shared damage loop. The
dispatcher performs neither `0x1ca89` command-MP debit nor `0x1b8e7`
inventory removal. Tracked type20 IDs11/56/60 select commands2/0/2, type21
IDs29/38/51/99 select 6/1/7/6, and type24 ID79 selects command3.
`NativeItemCommandDamageRoute` and `ApplyNativeItemCommandDamage` preserve
the presentation distinction, retained-source, no-MP-debit and sequential
target-damage contract without assigning item display names. A previous
adapter incorrectly substituted Go `math/rand` for both calls. Direct
Capstone at `0x1c7ed` and `0x1c869` proves both are `0x4e893`; the adapter,
player command 0 runtime, and item types20/21/24 now consume the same
process-lifetime `uint16` state (one step on miss, two on hit).

The Ebiten item target transaction now also executes types6/7, 12, 14–16 and
20/21/24. It synchronizes raw transient markers, HP, AP/DP/HIT/EV and compact
inventory back to Units; retained-source families remain retained. This is a
mutation/runtime closure only: each branch's indexed effect presentation is
still pending and must not be represented as restored.

The earlier attribution of a word subtract to `0x1cac7` was an address error:
that arithmetic belongs to `0x1ca89`, the independently verified command MP
debit helper. `battle.ApplyNativeRawWordSubtract` retains only that raw
arithmetic boundary; type21 does not call it.

The corrected common `0x22af6` primitive is captured by
`ApplyNativeRawFlagRestore(records,targets,markerOffset,rng)`: it preflights
record bounds, reads and clears marker bytes in those records, invokes the
proven HP restore only for nonzero markers, and adds the raw `effective*4`
accumulator. It no longer accepts or mutates a detached flag array.

Caller-level evidence around `0x24838` must remain separate from the raw lookup adapters. It first branches on `0x24b14(0x64)`; the success arm presents text `#8` and calls `0x112a5(0x16)`. It then branches on `0x24bde(0x12)`: hit presents text `#10`, acting `#0x48`, and `0x32975(0x11)`; miss branches on global count `0x53bef < 0x0f`, choosing text `#13` plus `0x112a5(0x13)` or text `#12` plus `0x32975(0x11)`. Shared sync/presentation follows. These are address/order facts only; no item, character, chapter, or NPC names are inferred from the immediates.

The downstream `0x32975(index)` mutation is independently closed: it computes the selected runtime record at `base + index*0x50` and writes byte `+0x05 = 1`, overwriting the entire byte. `battle.SetNativeRecordByte5One` preserves that overwrite and bounds behavior. It is intentionally separate from the `0x13512` bit7 setter and does not name byte5 as acted, turn, or action state.

### 2026-07-27 — constructor inventory flag materialization

Official IDA 9.4 pseudocode for `0x10c50` closes the constructor's eight raw
inventory flags. The first cell always receives `0x40`; if source byte 0 is
`0xff`, source byte 1 is placed in that first item cell and the second flag is
reserved `0x80`; otherwise the second flag is also `0x40`. Source bytes 2..7
copy to the remaining item cells, with flag `0x00` for a present item and
`0x80` for source `0xff`. `battle.NativeInventoryFlagsFromSource` preserves
only those byte writes, and `NativeInventoryCompactEligible` applies the
caller gate as a signed-byte test: `0x40` and `0x00` are eligible, `0x80` is
not. `Load`/`PartyUnits` retain these flags when the eight `inventory_slots`
source is present; legacy JSON remains a conservative projection. This fixes
the former incorrect “un-equipped only” description and does not assign an
item category or church service name.

### 2026-07-29 — 敵方行動模式的來源與可編輯資料契約

`0x10FB6..0x10FC8` 證實 FDFIELD 名冊列 `b17/b18/b19/b2` 依序建立
執行期記錄 `+0x34/+0x35/+0x36/+0x3D`。其中只有
`record+0x34 & 0x0F` 是 `0x13A9F` 的分派值；同一位元組的高四位及
個別 bit 另有使用者，不得把整個 byte 序列化成有名稱的單一列舉。

資料管線現在逐筆保存 `native_record_byte34/35/36`，`Unit` 以各自的
`HasNativeRecordByte*` 記錄來源完整性；`NativeAIModeRecordForUnit`
只有四個必要來源都存在時才建立原始記錄並交給
`fdother.PlanNativeUnitMode`。`SetNativeAIModeRange` 對應 `0x3419C`
的 inclusive range 與 `(old & 0xF0) | mode`，`SetNativeAIModeByte`
則保存章節處理器與 mode-5 完成路徑的整個 byte overwrite。

[`fdfield_native_ai_modes.json`](../data/fdfield_native_ai_modes.json)
固定同一份 FDFIELD.DAT 的 33 圖、1887 筆分布；低四位只出現
0、1、2、3、4、5、7、8、9、10。這項閉合只讓原始行為節點可編輯、
可驗證；`NextAIPlan` 尚未讀取它們，模式名稱與 0x13A9F 各分支的完整
遊戲語意仍採失敗即停止，不以 `set_ai:berserk` 或重製近似補猜。

模式 2／11 的控制流已有更窄的可執行契約：
`fdother.PlanNativeUnitMode2` 固定 `0x14EF0` 失敗後為
`0x14237→0x13C06→0x13FD4`，明確排除 `0x13E9C`；`PlanNativeUnitMode11`
保存 `[0x53C23]` 與 `[0x53C4F]` 兩個獨立 signed `>=6` gate，以及
第一段執行後仍會評估第二段的順序。`battle.ApplyNativeAIIdleRecovery`
另保存 `0x13FD4` 的 raw `+0x25/+0x26` 零值 gate 與
`min(currentHP+floor(maxHP/5),maxHP)`。玩家休息正式路徑也已刪除錯誤的
「至少回復 1」近似，改讀 `NativeTransient[3:5]`。

`fdother.PlanNativeUnitMode0`／`PlanNativeUnitMode1` 也只保存原始
`0x14EF0`／`0x14121`／`0x13E9C`／`0x13FD4` 的巢狀回傳順序；它們是
E0 控制流資料化，不是玩法名稱或正式 `NextAIPlan` 接線。缺少 raw 回傳
來源時必須失敗即關閉。

`fdother.PlanNativeUnitMode3/4/5/7/9/10` 同樣保存已證實的 lookup、
移動、事件狀態、座標比對與 raw writes；mode 5 的抵達後寫入只以
`EventRecordType` byte gate 表示。這些 helper 仍是 E0 控制流資料化，
不授權正式 AI planner、交易或呈現消費端。

模式 11 的唯一已知 runtime source 是全域事件 82 handler `0x35F92`：
battle-local state entry `+0x10` 等於 4 時，`0x36078` 透過
`0x3419C(20,20,11)` 只改單位索引 20 的低四位。它位於
`0x51B91 + 82*4`，且合法 IDA 顯示多個通用 dispatcher 交叉參照；
一般玩家觸發條件仍未閉合，33 張 FDFIELD 格子事件表也沒有 event 82，
不得命名成踩格、章節或人物事件。

事件 82 的資料來源稽核已再收窄：固定雜湊的 FDFIELD 全 33 圖中，
turn-events、格子事件、特殊寶物列與單位後處理列都沒有 payload 82；
四個 EXE 硬編碼 `0x1AA1D` 單列也只給 kind0 物品 D3、D5、0x65、0x0B。
因此目前沒有已知 serialized producer；但 runtime `+0x31..+0x33` 的所有
後續 writer 尚未閉合，SDD 仍將 event 82 標成未知可達性，而非 dead code。

寶物資料邊界則已完整版本化：`tools/sync_native_treasures.py` 從
FDFIELD composition/control 與 FDSHAP flags 重建全 33 圖
`treasure_slots`、`treasure_hidden` 和 16 槽 reward 定義。
type0/1 可執行物品／金錢；其他 type 保存為 `event` 與 `native_type`，
`ClaimTreasure` 在 handler 尚未資料化時拒絕取得且不標記 opened。

第一條特殊寶物 handler 已完成垂直切片。map25 slots0..4 共用 event58；
`0x354FE` 依 slot 查 `0x5274E` 五物品
`[0x1D,0x2B,0x33,0x3D,0x47]`，滿八格時不改狀態，成功則透過原始
inventory reservation primitive 加入對應物品，並共同關閉 slots0..4。
規則由 `extract_native_treasure_event_rules.py` 綁定 FD2.EXE 雜湊後輸出，
再嵌入 map25 editable asset；runtime 不含 handler 位址捷徑。

FDFIELD 控制段 `+0x33` 起的 32 bytes 已更正為 16 筆
`(event_id, selector)` 格子事件列。地圖構成 event-word low5 是 1-based
slot；只有 FDSHAP 地形控制 byte0 的 `0x20/0x40` 皆未設置時才採此解釋。
33 張可編輯 `map.json` 保存 `native_field_event_slots` 與
`native_field_events`；`battle.NativeFieldEventIDAt` 僅執行已證實的
slot、`0xFF` 與 selector gate，尚未猜測性 dispatch 未解 handler。

### 2026-07-29 — current-runtime snapshot 的原生名冊邊界

合法 IDA Pro 9.4 直接指令已閉合 `0x10010` 的五個 plaintext 區域：
`FD2.SAV+0x0000` 的 `0x08A3` bytes 是 FDFIELD 控制段的執行期映像，
`+0x08A3` 固定 `0x0A00` bytes 是 persistent roster，
`+0x12A3` 是由 header runtime count 限定的 runtime roster，
`+0x30A3` 是 32-byte battle-local event state，`+0x30C3` 是
18-byte header。
header `+0/+1/+9` 分別寫入
`[0x53BEF]/[0x53BEB]/[0x53BFB]`，因此是 turn counter、runtime count、
persistent count；舊工具把 `+0` 稱為 persistent count 的斷言已刪除。

`fdsave.InspectCurrentSnapshot` 現保存兩份原始 `0x50`-byte 名冊、
`0x0000..0x08A2` `NativeFieldControl`、`0x30A3..0x30C2` battle-local
event state 與
18-byte header，並對 runtime count `<=96`、persistent count `<=32`
採失敗即關閉。使用者目前
checksum-valid 的未修改快照實測得到 persistent identity
`[0,9,4,30]`，依固定角色表是索爾、悠妮、亞雷斯、蓋亞。此動態快照不是
固定版本 fixture，也不證明原生 LOAD／CONTINUE 已完成；identity/class
catalog 與正式 party materialization 已由下一段閉合。控制映像、runtime
selector／record、timing 與 future-group constructor 的個別 adapters 已由
下文閉合；整組 `battle.State` 到正式控制器的原子交接仍未閉合。
2026-08-02 以合法 IDA Pro 9.4 重查後撤回「`0x4E031` 是戰鬥驅動器」：
該函式只把 absolute `0x41A` 的 word 複製到 `0x41C`；把兩個 word 視為
BIOS head/tail、進而解釋為丟棄待處理輸入，最多是外部 ABI 的強推論，不能
當作函式本體或重製端的已證實語意。第三分支返回 0
後，main 才在 `0x25DCE` 呼叫並循環重入共享戰鬥控制器 `0x117E7`。
逐指令證據見
[`fd2_current_snapshot_ida.txt`](../data/fd2_current_snapshot_ida.txt) 與
[`fd2_continue_controller_117e7_ida.txt`](../data/fd2_continue_controller_117e7_ida.txt)。
下述具型別 adapters 現已分別閉合控制映像、runtime units、timing 與
future-group constructor；chapter0 的未改寫 live 排程也已有嚴格
pending-group 綁定及原版快照測試。尚未閉合的是 handler 改寫 turn byte 後的
通用 chapter pending-group binding，及把整組候選狀態一次發布到正式
`Game`／controller 的 handoff。
FDFIELD 控制映像的來源、固定容量與 consumers 見
[`fd2_current_field_control_ida.txt`](../data/fd2_current_field_control_ida.txt)。

後續 IDA／Capstone 勘誤證實 `[0x53AD5]` 是 main 以 `malloc(0x20)`
建立的 pointer，不是32 bytes相鄰 globals 的起點。current-save writer
`0x1A03D..0x1A04C` 與 CONTINUE reader `0x10319..0x10328` 對這個
heap table 做完全對稱的 `0x20`-byte copy；`0x190F1/0x19246` 等路徑
再以 event index 讀／寫 byte。既有 `battle.State.NativeEventState`
已保存 ch25 index12、ch06 index17、AI index16 等窄 raw predicates，
因此 `CurrentSnapshot.Raw30A3` 已更名為 `NativeEventState[32]`。
這不替其餘 index 命名，也不接通 CONTINUE。直接證據見
[`fd2_current_event_state_ida.txt`](../data/fd2_current_event_state_ida.txt)。

### 2026-07-30 — `0x10652` 章節輔助圖形勘誤

合法 IDA Pro 9.4 將 `0x10652..0x1088D` 固定為只有三個 caller 的函式：
CONTINUE `0x101D7`、完整章節 loader `0x108A6`、ch22 post `0x24A9A`。
它先釋放 `[0x53AFF]/[0x53B03]`，再只針對 raw chapter
`9/17/21–25/27–29` 載入或展開特定 FDOTHER 資源；其他章節只留下
已清空的緩衝區。FDFIELD、FDSHAP、FDTXT 與 roster 的完整載入仍由
`0x1088D` 負責。

因此 handler exporter 的舊 `load_ch_bg` 名稱已撤回，改為
`prepare_chapter_aux_graphics`。這個可編輯節點仍無正式執行期降階；
compiler 必須回報 issue，不能猜成背景切換、重繪或完整 LOADCH。
【截至2026-07-30的歷史狀態；`0x2189A`／`0x24B4D` buffer owner 已由
2026-08-21 後續證據部分取代。】當時 ch22 post 對應的 indexed buffer owner
尚未完成，故維持失敗即關閉。
直接證據見
[`fd2_chapter_aux_graphics_10652_ida.txt`](../data/fd2_chapter_aux_graphics_10652_ida.txt)。

同次 canonical handler 重生把 ch29 post 的 `0x25870→0x1088D` 從舊
`load_ch_text` 改回 `loadch`。compiler 已刪除舊名的相容降階；即使提供
完整 binding，`load_ch_text` 也必須回報 unresolved，避免把完整 FDFIELD、
FDSHAP、FDTXT、roster loader 再次縮窄成文字切換。

後續已新增綁定同一 FD2.EXE SHA-256 的
`native_character_catalog.json`，只保存 32 筆 persistent identity 名稱與
原版 class 0–28 文字。`MaterializeNativePersistentPartyRecord`
可把單筆 record 的身份、職業、能力值、inventory flags/items、command mask、
transient bytes 與已證實傳給 `0x11019` 的 raw `+7` map selector key
投影成 `battle.Unit`；不把該 key 猜成 portrait／Fig，也不填座標、
出場狀態、spells、attack range 或章節節點。合法 IDA Pro 9.4 已證實
`0x30B07..0x30B17` 與 `0x310B5..0x310C9` 都直接以 record `+0x20`
加 FDTXT 索引 150 顯示職業，沒有 0–26 上限；固定雜湊
`FDTXT_000.bin` 的索引 176／177／178 分別是「？？？」、兩個全形空格、
「？？？」。因此 class 27 必須保留原版空白文字，class 28 必須顯示
「？？？」；舊 `cls28`、`?`、「職業28」與「27／28 衝突」均是重製端
占位或錯誤斷言，已撤回。這只關閉 record materialization，不解除
LOAD／CONTINUE 的正式失敗即關閉閘門。可選的
`FD2_NATIVE_SAVE_FIXTURE` 整合測試已在 Docker 唯讀載入使用者
checksum-valid 原版快照，依實際順序成功產生索爾、悠妮、亞雷斯、蓋亞；
原版檔不進版控，缺 fixture 時測試明確略過。

### 2026-07-30 — CONTINUE runtime input preflight

合法 IDA Pro 9.4 與 Capstone 已閉合 `0x10010` 的 selector 重建：
runtime records 完整複製後，`0x1035C` 清 cache count，
`0x1036A..0x1039C` 依 current runtime record order 取 record `+7`
呼叫 `0x11019`，再覆寫 record `+2`。故 CONTINUE 的正確 first-seen
順序就是 current runtime list；存檔內舊 `+2` 不是可驗證真值，也不能
錯套新章 loader 的 persistent→FDFIELD group construction order。

`fdsave.BuildContinueRuntimeInput` 現接受 snapshot 與明確 resource
context（chapter、field dimensions、FDICON group count），原子驗證：

- chapter 與資源一致，runtime／persistent count 在固定容量內；
- FDFIELD control 的 unit count 不超過 `0x83+80*0x1A=0x8A3`；
- camera、cursor、visible cursor 符合 13×8 native viewport identity；
- 每筆 `+7` key 可由 FDICON group count 接受，並按原版順序重算 slot；
- active record 的 raw `+0/+1/+3/+4` 符合 field、pose 0..3、motion 0..6；
- field control、兩份 roster、event state 與 header 都深複製。

後續 IDA Pro 9.4 完整資料交叉參照與 Capstone 原始位元組覆核，又閉合
標題 CONTINUE 的 map presentation。`0x10483` 在開場重繪前明確寫
`[0x51A83]=0`，`0x1060C` 在 `0x10616→0x4E031` 前改寫為 `1`。
資料映像的 gate B `[0x51AAC]` 與 anchor `[0x51A0C]` 初值均為 `1`；
標題 caller `0x26124..0x26130` 沿路不改 gate B，anchor 則只由
`0x1AD2A..0x1AD5F` 依已恢復的 visible cursor 寫 `0xF2`／`1` 或保留。
因此 `ContinueMapPresentation` 現明確輸出開場／互動 range mode、
gate B 與有效 anchor，不再把這四值列為未知。此契約只適用標題
`0x26130` caller；不外推到戰場內 `0x1A251` caller。

runtime unit projection、map timing seed adapter 與完整 future-group constructor
transaction 現均已有嚴格 consumer。未改寫 live turn/event 可由
`MaterializeNativeContinuePendingGroups` 綁定，但原版存在多個 turn-byte
writer，尚未全部資料化成 slot／公式。因此 preflight 仍保留兩個尚待 caller
接管的 owners：通用 chapter pending-group binding，以及已知 `0x117E7`
對應的 remake `Game` controller handoff。因此
`ReadyForContinue()` 固定由 owner 清單決定，目前為 false；此 preflight
沒有 production owner，也不改 `battle.State`。直接證據見
[`fd2_continue_selector_rebuild_ida.txt`](../data/fd2_continue_selector_rebuild_ida.txt)
與
[`fd2_continue_map_presentation_ida.txt`](../data/fd2_continue_map_presentation_ida.txt)。

同輪 timing data-xref 又把 `map_timing` 拆成「已閉合種子」與「未接
執行期排程」。資料映像的 sprite cycles、terrain phase/latch、
terrain flip/latch、unit pixel-shift/latch 都是零，terrain override
`[0x51A93]` 是 `-1`；唯 sprite last tick `[0x53C0F]` 由 main
`0x25D83..0x25D8B` 在標題入口擷取 signed BIOS low word。故
`ContinueRuntimeContext` 必須明示 `TitleTimerTick`，
`ContinueMapTimingSeed` 才能無猜測保存首次 `0x11CAC` 前的完整種子。
`campaign.MaterializeNativeContinueMapTiming` 現已原子安裝這份 seed；
地圖合成器也把 `0x1297D` 與兩組 binary latch 的取樣移到每次實際
`0x11CAC` 等價合成交易，只有完整 frame 成功才同時發布 timing 與 pixels。
舊的每次 `Game.Update` 無條件推進已撤回。`0x10494` 與 `0x105ED` 中間的
固定演出／delay 排程仍由正式 CONTINUE handoff 負責，但不再是未知
`map_timing` 資料 owner。證據見
[`fd2_continue_map_timing_seed_ida.txt`](../data/fd2_continue_map_timing_seed_ida.txt)。

`NativeFieldControl` 也不再只以魔術偏移供後續程式直接索引。
`ContinueFieldControlView` 依已證實的 FDFIELD control layout 唯讀拆出
raw header、16 筆回合事件、16 筆格子事件、16 筆寶箱控制與
count-delimited 26-byte 單位列，完整 raw 映像仍同時保留並深複製。
IDA `0x10BCC` 的 `cmp index,[0x53BE3]; jge` 證實 raw `+2` 是排他的
有效筆數。使用者 current snapshot 的 chapter 0 控制前綴也與
FDFIELD resource 1 全同；雖該資源容納 31 列且 `+2=30`，第 31 列及
固定映像尾端都只保留 raw，不可因資源長度而當成 live unit。
後續完整 data-xref 與直接 writer 又證實這份 control 不是載入後的靜態
資源副本：`0x19357` 會改寫 chest value，`0x34AB4/0x34AC5` 及多個
chapter handler 會改寫 turn event bytes。因此 selected chapter 的
原始 resource 只能驗證身份與靜態來源，不能覆寫 current snapshot 的
live `NativeFieldControl`。同理，control rows 只供未來
`0x10B4E→0x10C50` group append；現有單位必須以 saved runtime
`0x50` records 及其原順序為準。
這不是 map composition：CONTINUE 另載 FDFIELD `3N` 資源，
`0x4DBFC` 才建立 live cell byte `+3`；舊 serialized map byte 不能
覆寫 current runtime。typed view 只關閉 control decoder，尚未完成
完整 runtime roster bridge。

`campaign.MaterializeNativeContinueFieldBoundary` 現已成為第一個嚴格
consumer。它不只檢查 `BuildContinueRuntimeInput` 的私有 marker，還會從
目前公開欄位重建 snapshot、重跑完整 preflight 並逐欄比對，拒絕建構後被
修改的 header、control、runtime record、selector 或 owner。外層另須先
選定相同 raw chapter、dimensions 與完整 field-event asset。所有檢查通過
後才一次安裝：

- exact live `NativeFieldControlRaw`、16 筆 turn/field/chest rows 與
  count-delimited 26-byte future-unit rows；
- `NativeEventState` 與 raw `NativeRoundCounter`；
- saved camera/cursor/visible cursor、HUD gates/anchor；
- 第一張 opening redraw 使用的 range mode `0`。

這個 adapter 不碰既有 `State.Units`，不重建 saved runtime `0x50`
records，不啟動 timing，也不把 range mode 提前推成 interactive `1`。
因此它已關閉原本籠統的 field boundary owner；saved runtime-record
materializer 與 `0x10C50→0x1B750` constructor transaction 也已在後續閉合。
未改寫排程的 rows／item table 綁定已由後述適配器完成；目前真正剩餘的是
把 handler 改寫的 live turn slots／group 公式資料化，使所有合法章節快照都
可重建 pending roster，以及一次發布 `Game`／controller handoff。
直接證據見
[`fd2_current_field_control_mutations_ida.txt`](../data/fd2_current_field_control_mutations_ida.txt)。

`campaign.MaterializeNativeContinueRuntimeUnits` 現已關閉前者的
lossless adapter。它要求上述 live field boundary 已安裝且逐筆 raw
record、key、slot 仍與驗證輸入全同；接著先在 detached roster 完整驗證
native camp `0/1/2`、class catalog、active presentation 與 first-seen
selector cache，任一錯誤均不改 `State`。成功時才依 saved runtime list
順序一次替換 `State.Units`，並保存：

- exact `+0/+1/+3/+4` presentation，存檔 `+2` 一律不信任；
- `+5/+6/+7/+8`、command mask、race/class、transient、
  `+34..+36`、`+42/+46` 及完整八格 inventory raw provenance；
- 所有已證實 signed stat words，及由 `+7` 在原順序重建的 selector slot。

2026-08-02 重讀 `0x117E7` 又補上高階投影邊界：玩家單位掃描以
`record+5 & 0x85` 作 admission，直接點選路徑另測 bit7；成功動作由
`0x13512` 設 bit7，回合相位由 `0x13536` 清除。因此
`Unit.Acted=(raw+5 bit7)` 可作這個控制器的行動准入投影；這不把整個
byte `+5` 命名為 acted，也不替 bit0、bit2 或完整換邊條件補語意。

這裡特別撤回把所有 runtime `+8` 都稱為角色 identity 的錯誤模型：
scripted constructor 會把 FDFIELD `b1` 同時寫入 runtime `+7/+8`；
只有 native camp 2 的 player record 依 persistent copy／lookup 契約將
`+8` 提升為 `NativeIdentity`。其餘單位只保存
`NativeRecordByte8`，名稱使用 class 顯示投影，不冒充角色身分。
`NativeItemPanelRecordForUnit` 也改優先消費這個 raw provenance；舊
player-only assets 的 `NativeIdentity` fallback 僅為相容邊界。
同一更正已套用到全部33份版本化 map unit assets：
`tools/sync_native_selector_fields.py` 現由 FDFIELD `b1` 輸出
`native_record_byte8`，並拒絕／移除舊 scripted `native_identity`；
scenario party 的 persistent identity 欄位不受影響。AI fixture 與
selector 測試也改以 raw +8 定位 scripted actor，不再靠錯誤語意名稱。

未改寫 live schedule 的窄切片已使用 checksum-valid 原版 `FD2.SAV` 在
Docker 唯讀整合測試：
chapter 0 的 12 筆 runtime records 全數依序 materialize，前四名 player
為索爾、悠妮、亞雷斯、蓋亞；敵方 record `+8=96` 保持 raw 且沒有
`HasNativeIdentity`。同一測試現再以 map0 的 24×24 資產綁定 groups 3..7
共15筆 future rows；groups 1/2 已在 runtime，10/11 未排程，均不混入。
舊測試註解把 map0 寫成31×24，已依版本化資產更正。這個 adapter 不啟動
map timing、不切 interactive range mode，也不發布 `Game` controller handoff；
正式 CONTINUE 仍因動態 pending-group binding 與原子 handoff 維持失敗即關閉。

### 2026-08-02 — CONTINUE 目前回合與 pending-group 綁定邊界

合法 IDA Pro 9.4 重新固定存檔與回合收束的直接順序。`0x117E7` 在未選中
單位時呼叫系統選單 `0x16F55`；raw selector 0 於 `0x16FED` 直接呼叫
會寫入 `FD2.SAV` 的 `0x19DF7`，隨即返回。這條存檔分支不呼叫
`0x1A30B`。玩家行動路徑則在章節 handler 後呼叫 `0x13565`；只有已無
合格 raw camp2 單位時才進 `0x1A30B`。後者依序以 selector 1、0 呼叫
`0x1A813`，在 `0x1A5B9` 增加 `[0x53BEF]`，後段再以 selector 2 呼叫
`0x1A813`。因此不能把三個 selector 合併成單一 `>=`：saved turn 的
selector1/0 尚待下一次 phase 收束；selector2 已在上一輪增加到該回合後的
尾端掃描。future roster 的門檻必須是 `turn > saved_turn`，或
`turn == saved_turn && selector in {0,1}`。目前沒有 turn1/selector2 的
可編輯 spawn；若將來出現，bootstrap 順序另證前失敗即關閉。
直接證據與推論分級見
[`fd2_continue_pending_schedule_ida.txt`](../data/fd2_continue_pending_schedule_ida.txt)。

`campaign.MaterializeNativeContinuePendingGroups` 現以失敗原子方式：

- 要求 raw chapter、scenario chapter/map、24×24 等實際資產尺寸、FDFIELD
  row count、live field/runtime adapters 及 native append ownership 全相符；
- 只接受可編輯 `on_turn_end`／once／turn／native event ID／逐呼叫
  source/via/raw gate 完整，且 live `(turn,event_id)` 恰好一列的排程；
- 只深複製上述尚未消費的 saved-turn selector0/1 與未來事件所引用的
  FDFIELD groups，另私有複製 item
  rows；已資料化格子事件的 spawn group 先要求 editable rule、地圖 slot、
  資產 selector 與存檔 live event 相符，再以其 once-state raw byte 判斷是否
  尚未消費。current Units、selector、live control、timing/view/HUD 均不變；
- live turn 已被 handler 改寫、group 重複排程、資產缺 constructor provenance
  或 source row 缺失時，在任何 `State` mutation 前拒絕。

全章機械式盤點確認46個 `spawn_group` action 均有排程來源，但只有
ch01/ch02/ch03/ch26 目前宣告 `runtime_append_groups`。event 27/54/57 以
`[0x53BEF]` 動態取 group，event47/49 另有 group formula；ch03/event9
還帶 runtime slot6 條件。另有 `0x34AB4/0x34AC5`、`0x358BA` 等多個
live turn-byte writer；先前列出的 `0x34AA7` 與 `0x358B5` 分別只是
round-counter load 與 `inc dl`，不是 store 指令。這些仍須增加固定 slot 與
公式資料，不能因 chapter0 測試通過就移除
`pending_group_binding` owner。現行適配器是可驗證的靜態排程垂直切片，
不是所有合法原版存檔的通用 CONTINUE 完成宣稱。

同輪開始縮小 `future_group_constructor`。完整 Docker Capstone
`0x10B4E..0x11018` 直接指令固定：

- caller 依 control `b21` 比對 group，按 control row 順序逐筆 append；
- position resource 是6-byte records，constructor 以相同 row index 讀
  X/Y word low bytes；`[0x53AFA]==0` 時會先以 `0x145CD(0/1)` 將所有
  `+5 bit0==0` runtime 單位所在格標 `0x40`，再掃全圖取 Manhattan
  最近的非占用格；同距離取 row-major 後者。非零才直接使用原座標；
- `b2→runtime +0x3D`，而 `b3/b20/b25` 在此函式沒有 reader；
- `b13..b16→+0x1A..+0x1D`、`b17..b19→+0x34..+0x36`、
  `b22..b24→+0x31..+0x33`；
- race/class 與 base words 來自 b1-selected EXE tables，不是 b2/b3。

這撤回 parser 歷史 `race=b2`／`cls=b3` 作為 native ABI 的錯誤暗示；
兩個欄位暫留只服務既有 normalized base-stat 近似。全部33張 map 現保存
exact six-byte `native_position_record`、`native_record_byte3d`、
三-byte `native_record_death_effect`、raw source `b3/b20/b25` 與
b1-selected constructor table record；loader 與 CONTINUE runtime
projection 亦保存對應 runtime raw 欄。

`NativeFutureGroupPlacement` 現以短生命週期 composition slice 精確執行
兩次 `0x145CD` writer、raw `[0x53AFA]` 分支、全圖 row-major Manhattan
搜尋及後者勝出的 tie rule。它不改寫不可變的
`NativeCompositionEventBytes`。舊 `nearestFree` 只是重製端半徑1..7
環狀呈現，已撤回其對映原版 constructor 的斷言。
`DecodeNativeFutureConstructorBase` 同時轉寫 high/lower table 兩分支的
race/class、base AP/DP/DX、mobility、HP/MP 16-bit writes；同步工具已把
1,885筆 selected records 寫入33張 map，並逐筆驗證 race/class/HP/MP。
`native_unit_tables.json` 現直接攜帶原版大小、MD5 與 SHA-256；同步工具
必須先與 `fd2-reference-files.json` 相符，且由原版重生的 JSON 已逐位元組
等同版本化檔案。
後續 official IDA Pro 9.4 對 `[0x53AFA]` 完整資料交叉參照，並由 Docker
Capstone 逐組重生直接指令：唯一 reader 是 `0x10CB7`；22 個 writer 恰為
11 組 `set 1 → push group → call 0x10B4E → reset 0`，沒有其他非零值或
未配對 writer。三組屬章節 handler（`0x32E50`、`0x331B2`、`0x33419`），
八組屬 global event call（event 3/6/7/15/25/35，以及 event 37 的兩次
call）。`0x32999` wrapper 本體與 caller 都沒有 writer，故其內部 call
明確讀零，而不是未知。

`HandlerBeat → Beat` 現以指標欄位 `raw_placement_gate` 保存逐 call-site
byte；原版 handler 缺欄位或超出 byte 範圍就產生 compile issue，不再以
group/camp 猜值。25 筆版本化 handler spawn 全數保存欄位，只有上述三筆為
1；`event_id_groups.json` 的34筆 call 也保存 source/via/gate，八筆為1。
`cmd/fd2` 遇到有此欄位的 handler beat 會呼叫
`AppendGroupWithNativePlacement`：先對完整 batch 預演 position row、兩次
occupancy writer 與逐列 append 後的新占用，再一次 materialize group，
失敗時 roster／units 不變。沒有 raw 欄位的擴充戰役 authored beat 才保留
明示的 normalized compatibility path。

global turn-event 現也完成可編輯資料與正式執行端的第一段接線：45 筆可解析
schedule 產生46個 `spawn_group` action（event 0 的 groups 3、7 因人工演出拆成
兩個 action），全部保存 `native_event_id`，並逐 call 保存 source、via 與
`raw_placement_gate`。排程可達的六筆 gate=1 精確為 ch01/event3、ch02/event6、
ch05/event15、ch07/event25、ch12/event35、ch13/event7；event37 的兩筆 gate=1
不在 `turn_events.json`，因此不能假造進任一關卡。產生器遇到同章同回合多個
spawn schedule 或既有 action 綁定不同 event id 時直接拒絕合併。

正式介面執行器改用有錯誤回傳的 `ExecuteActionChecked`。只有
`runtime_append_groups` 情境且具有完整 runtime roster 時，才逐 call 呼叫
`AppendGroupWithNativePlacement`；缺 roster 會失敗即關閉。尚未遷移的情境仍走
明示的正規化相容路徑（normalized compatibility path），不能算原版忠實度證據。
版本化 ch02 的 turn3/event6 已由真實 scenario action 驗證六名 group3 友軍
採 gate=1 的六筆原始位置列（position row）；錯誤路徑不呼叫回合完成回呼
（continuation），
因此不會在漏生增援後仍偷偷前進回合。
`spawn_group_with_intro` 現另保存 wrapper 返回後的確切 acting resource／
call-site；全域 event1/2 分別是 `0x342E7→ACTING(3)` 與
`0x3434F→ACTING(4)`。`0x32999` 本體不含 acting，而是用 FDOTHER #9 做固定
12 次索引合成／呈現。原版來源的 handler runner 現已用具型別排程承接12次
呈現：先以完整 FDFIELD 狀態在私有副本建構新增群組，再按舊／新槽位邊界合成
FDOTHER #9；第6、7、8次各自重建快照，第1次另觸發 FDOTHER #95，全部預檢
成功後才發布名冊。每次呈現都必須經過 `Draw` 確認，完成後才進入 caller 的
獨立 ACTING beat。缺素材、欄位、selector 或 framebuffer 幾何時，仍在名冊
變更前失敗即關閉。全域 event1/2 現由 `Game.advanceBattleEvent` 在低階 action
executor 之前辨識兩個已證實 caller；它以保留當下 selector cache 與動畫週期的
私有 battle state 預演 group4／5、12次呈現與 ACTING(3／4) slot frontier，成功
後才發布 units／roster／cache。scenario 以 `native_acting_resources` 明示可編輯
的 `map32.json` 轉錄來源。`ExecuteActionChecked` 本身仍刻意拒絕 intro call，因為
它沒有 `Draw`／音訊／非同步 continuation；所有其他來源、混合 call 或缺資源
情況仍失敗即關閉。
`0x10C50` 的 table-base projection、八格 inventory 初始化與 `0x1B750`
即時裝備／衍生能力重算，現已合成 `MaterializeNativeFutureConstructor`，並由
`AppendGroupWithNativePlacement` 在私有候選名冊完成全部建構、位置及 selector
預檢後才原子發布。`0x1B750` 另已由合法 IDA 與 Capstone 證實包含
`+0x22/+0x23/+0x24` modifier；它不等同於沒有這三條分支的 persistent
`0x1145A`。這解除 future-group constructor 的具型別 transaction 缺口；
CONTINUE 剩餘的是將所選章節的未登場 rows 與 item table 綁入同一候選
`State`，不能再把它記成「constructor 尚未反組譯」。完整 0x50-byte
逐位元組一致、其他 caller 與正式 CONTINUE handoff 仍維持失敗即關閉。

`gen_campaign.py` 的總戰役拓撲仍落後人工閉合的權威
`campaign_full.json`；本輪實測直接重生會把299節點降成293並遺失 handler／
介面／戰後路徑欄位。新增 `--scenarios-only` 供安全重生逐章情境，且以內容雜湊
回歸證明不改寫總表；未完成拓撲保真合併前，不可用預設模式覆蓋權威總表。
直接證據見
[`fd2_future_group_constructor_capstone.txt`](../data/fd2_future_group_constructor_capstone.txt)
與 [`fd2_future_group_raw_gate_ida.txt`](../data/fd2_future_group_raw_gate_ida.txt)。
`0x1B750` 的函式、呼叫者、binary64 1.15 與 x87 朝零捨入證據見
[`fd2_runtime_equipment_recalc_1b750_ida.txt`](../data/fd2_runtime_equipment_recalc_1b750_ida.txt)。

本輪依規則先嘗試官方 IDA 9.4 Docker；合法 license 與私有 home 均隔離
掛載，但 batch 回報 EULA 尚未接受且 IDAPython target 未啟用，沒有產生
新 IDA database 或 pseudocode。既有 repo IDA 證據仍保留；上述新欄位只
以完整直接指令與原始位元提升，不冒充新 IDA 結論。

### 2026-07-30 — 四槽 LOAD 的戰間還原擁有者

合法 IDA Pro 9.4 將 `0x30012..0x301F4` 固定為章節槽 writer，且只有
`0x2CCB6`（`0x2CAD7` 直接整備）與 `0x2FD93`（酒店流程）兩個直接
呼叫者。writer 與 `0x2602C..0x26098` reader 對稱處理固定
`0xA00` roster 及 metadata `+0..+9`；`+10..+39` 不由這兩條路徑或
`0x30437` renderer 生產／消費，但不外推成全程未使用。

因為存檔發生在戰間流程內，選槽 LOAD 也會重新進 `0x2CAD7`，合法
restore entry 是 raw `[0x53C03]` 對應的城鎮／整備，而不是上一戰
postbattle handler。raw chapter 是 zero-based，顯示與節點章數為
`raw+1`；`0x526B9` 只有 index `22..24`、`27..29` 非零，故這六筆
直接進 `preparation_ch23..25/28..30`，其餘可存 raw `1..21/25..26`
進 `town_ch02..22/26..27`。ch21 recipe 與 ch27 item gate 已在 writer
之前執行，LOAD 不重播。raw0 沒有 `town_ch01`，現行 importer 失敗即
關閉，不創造未證實的第一章存檔點。

`native_intermission_gate.json` 以參考 EXE SHA-256 與位址綁定這30筆
資料；`BuildNativeChapterSlotRestorePlan` 在任何 runtime mutation 前驗證
完整 table、chapter、node type、catalog、active roster 與 identity
唯一性。production title 成功後一次套用 fresh campaign flags、gold、
party order／roster、空 deployment、raw chapter 與 metadata `+6..+9`
保存值；後四者尚未全部接 gameplay consumer。合成 checksum-valid fixture
已驗證成功與錯誤無部分 mutation，完整 `campaign_full` raw1..29 逐筆
可達；使用者未修改原版四槽皆空，因此仍沒有一般玩家有效槽 E2。
直接指令與範圍限制見
[`fd2_native_chapter_slot_restore_ida.txt`](../data/fd2_native_chapter_slot_restore_ida.txt)。

### 2026-08-02 — event62／event63 動態回合列與敵軍 AI 前增援

**已證實**：固定雜湊的 map26 control resource `FDFIELD_079.bin` row0/row1
依序是 `ff 3f 00` 與 `ff 41 00`。它們是完整 control table 的原始列；舊
`parse_field.py` 因 `turn == 0xff` 直接略過，使 event63／65 的 producer
provenance 從可編輯資產消失。現在 `native_turn_event_controls.json` 保存全部
33張地圖各16列，`turn_events.json` 繼續只承載已啟用且已補 handler/group 的
策展排程；兩者不可互相覆寫，也不可把 `0xff` 當成第255回合。

**已證實**：event62 handler `0x35898..0x358C6` 在 state byte17 為零時，
由 `0x358BA` 把 `[0x53BEF]+1` 寫入 live row0 的 turn byte，再把 state17
設為1。`NativeTurnActivation` 因而明示 slot0、event63、raw camp0、delta1；
`ApplyNativeFieldTurnActivationEvent` 只有在地圖 selector、可編輯規則、完整
原始列、once-state 與可選的 CONTINUE raw image 全部一致時才原子提交。
缺列、重複目錄項、raw/typed 不一致與重複觸發都失敗即關閉。
正式 `Game.stepBattleWalk` 現在只在原版已證實的向左一步第七拍座標提交後，
由 selector0 辨識 event62 並呼叫這個 transaction；其他方向不泛化，規則錯誤
則寫入既有 `loadErr` 並停止動作。Xvfb 回歸證實前六拍不改 row/state，第七拍
才把 round8 的 row0 改成 turn9。這是重製端 E1 玩家操作路徑，不是 DOSBox E2。

**已證實**：合法 IDA Pro 9.4 將戰鬥初始化函式固定為
`sub_205DA`（`0x205DA..0x2067D`），29個章節開場呼叫者共用 `0x2066E`
的 `mov dword [0x53BEF],1`；Docker Capstone 對相同範圍得到相同指令。
因此固定雜湊地圖的新戰鬥載入會從版本化目錄安裝原生回合初始值1，CONTINUE
仍以快照的即時回合值覆蓋；手工狀態或缺少完整目錄時保持0並失敗即關閉。
載入器另鎖定整份目錄的 SHA-256，並要求 map 0–32 各自唯一且資源名、控制列
數量完整；竄改單列、寫入端、摘要字串或地圖身分都在安裝狀態前拒絕。
這只閉合 event62 所需的回合來源，不單獨提升事件63畫面到 E2。直接證據見
[`fd2_battle_round_seed_ida.txt`](../data/fd2_battle_round_seed_ida.txt)。

**已證實的勘誤**：`0x35822` 的 handler export 保存來源 `PUSH` 順序
`(group,y,x)`，不是舊文件與 compiler 使用的 `(x,y,group)`。helper 內才以
`0x35834` 呼叫 `0x135DD(x,y)`，並以 `0x35842` 呼叫 `0x10B4E(group)`。
ch27 的 `[6,16,0]` 現正確成為 group6／座標(0,16)，ch28 的 `[8,19,9]`
成為 group8／座標(9,19)；非對稱整合回歸防止 group/x 再被對調。
`event_id_groups.json` 也不再把 event63 記成空動作；它以
`staging_helper_0x35822` 保存兩個 group、座標、outer call-site 與內層
spawn source，並故意不讓現有一般 `spawn_group` consumer 接受這個 via。
同一個固定雜湊 extractor 也補出 event64／66／68／70／72 的 staging calls；
這只提升其 group／座標／call-site 為可重生 E0 資料，不代表各事件的觸發條件、
畫面 phase 或一般玩家可達性已完成。

**已證實並接線**：`sub_1A813(0)` 在 `0x1A554` 掃 raw camp0，接著才於
`0x1A58F` 呼叫敵軍 AI，`[0x53BEF]` 又到 `0x1A5B9` 才增加。因此 event63
不再借用 AI 返回後的通用 `on_turn_end`。ch27 以獨立
`native_turn_events` 保存 event63／raw camp0／handler `0x358C7`，並把
group1、group2 從開局 `initial_groups` 移到待增援 roster。正式 runner 先在
私人 state 依序預演兩次 `0x10B4E` constructor；第二組或 renderer input
不完整時，不發布第一組，也不啟動敵軍 AI。

**已證實的調色盤勘誤**：`0x35857..0x35863` 是
`0x11DF2(0,255,255)`，不是舊文件所稱的 delta0/no-op。它把 immutable
baseline 的所有六位元 DAC 分量飽和為63；200ms 後
`0x11DF2(0,255,0)` 恢復 baseline，再由 `0x11CAC(0)` 重畫。正式 runner
依 camera pan → group append → 300ms → 全白呈現 → 200ms → baseline
restore → redraw 的順序執行兩次，完成後才啟動 AI。完整 native
view/HUD provenance 存在時使用 indexed DAC。後續 IDA Pro／Capstone 已把
ch26_pre 返回 battle_ch27 時的 camera `(9,49)`、absolute cursor `(14,54)`、
visible cursor `(5,5)` 與 selector 0 閉合，並以 view-only 設定接入正式節點。
HUD 的持續擁有者已在 E1 閉合：main 返回後的 gate B 由 controller 以1進場；
gate A 由 custom save 與 native chapter slot restore 保存；anchor 是程序內狀態，
只有 `0x1ACF3` 在已證實的左右邊界條件下改寫，其他情況保留舊值。因此三者不能
猜成章節常數，但 `battle_ch27.native_map_hud_inherited` 可以從 `Game` 持續狀態
原子物化。event63 production regression 使用明確帶有 persistent raw `+0x42`
來源的凱麗 fixture 走 indexed path；它不把 ch27 可編輯 `hp=90` 反推成原始欄位。
這仍不是未修改一般玩家路徑的同狀態像素 E2。直接證據見
[`fd2_ch27_pre_view_ida.txt`](../data/fd2_ch27_pre_view_ida.txt)、
[`fd2_hud_persistence_ida.txt`](../data/fd2_hud_persistence_ida.txt) 與
[`fd2_join_constructor_word42_ida.txt`](../data/fd2_join_constructor_word42_ida.txt)。

campaign schema 因此不再強迫 `native_map_view` 與 `native_map_hud` 成對：
已證實的 view 可獨立存在，HUD 不可脫離 view；節點入口只接受已有直接證據的
selector 0／1。執行期先在私人候選狀態驗證 view、selector 與可選 HUD，再
一次發布，避免 HUD 失敗留下半套 view。

**尚未完成**：event63 的未修改 DOSBox 同回合逐幀比較、CONTINUE 進入
event63 前後的一般玩家路徑、該時點凱麗 runtime `+0x42` 的實際值，以及
event64／66／68／70／72 的各自 phase／handler consumer。JOIN `0x112A5`
的精確 fresh-record 公式現已由雜湊綁定的
`native_join_constructor.json` 與 `NativeJoinConstructorTable` 接入正常
JOIN→LOADCH 及 `scenario join_party` 首次 persistent roster 建立路徑；凱麗
id12 的 `+0x42=151` 不再由測試手填，章節產生器的近似 HP／MP 也不會被反推
成 raw 欄位。既有已閉合的 `sub_1145A` raw transaction 現已接入同一
materializer：先按 `0x112A5` 建八格 `[flag,item]`、base AP／DP／DX，再以
版本化的 byte-exact 物品列前綴重算 signed `+0x48/+0x4A/+0x4C/+0x4E`，任何所需 item row
缺失均不發布部分角色。凱麗 fresh JOIN 的 base AP／DP／DX 為 `80/69/10`，
重算後 AP／DP／HIT／EV 為 `100/79/110/15`。這仍只保存已證實交易欄位，
不宣稱完整保存 0x50-byte record 的未觸及 bytes。直接位址與指令見
[`fd2_join_constructor_word42_ida.txt`](../data/fd2_join_constructor_word42_ida.txt)
與 [`fd2_persistent_roster_ida.txt`](../data/fd2_persistent_roster_ida.txt)。

### 第 29 戰前置視圖交接規格

`story_ch29→ch28_pre` 必須把原版同一組全域的最終狀態資料化，
不可因重製端切換 campaign node 而回到零值或上一戰鏡頭。雜湊綁定的
map28 部署格、acting 86 與 `0x12D7B→0x12CEA` 安全帶規則閉合：

- camera `(9,56)`；absolute cursor `(15,63)`；visible cursor `(6,7)`；
- `range_mode=0`，來自最後一次 `0x135DD` 的 selector writer；
- HUD gate B 由主 controller 返回點物化為 1；gate A 與 anchor 只從
  `NativeMapHUDPersistentState` 繼承。

`battle_ch29` 以 `native_map_view` 與 `native_map_hud_inherited` 承載此契約。
六個視圖值、selector 或持續 HUD provenance 任一不完整時，indexed
renderer 與第 29 戰戰後演出均失敗即關閉（fail-closed）。正式回歸必須從
`story_ch29` 的 handler 播放到 `battle_ch29`，再完成
`postbattle_ch29_persist→preparation_ch30` 與存讀檔邊界；直接手填 HUD
或跳節點不能取代這條 E1 玩家路徑。原始指令與來源見
[`fd2_ch27_ch28_pre_owner_ida.txt`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt)。

戰後 handler 仍在原版同一組六個全域上連續運作。
`0x25511:0x135DD(9,8)` 只重定 camera 與 absolute cursor，保留戰鬥結束
當下的 visible cursor；`0x2551D:0x12CEA(15,10)` 再以安全帶規則
更新六欄。因此 beat runner 的 pan/focus 必須對已物化的
`battle.State.NativeMapViewState` 原子同步；不能只改 `Game.camX/camY`，
也不能用 `absolute-camera` 重算並覆蓋被 pan 故意保留的 visible 值。
後續 `0x24B4D` 必須消費更新後的 camera；這項同步失敗時不得只用
入口視圖勉強合成。證據見
[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。

## 2026-08-09 勘誤：raw ch15 post 已解除 production gate（E1）

前文「`postbattle_ch16_persist` 維持 unbound」是本輪之前的歷史狀態，現由
直接驗證取代，不刪除原段落以保留推翻原因。正式 handler
[`ch15_post.json`](../../remake/assets/cutscenes/handlers/ch15_post.json) 與 binding
[`ch15_post.json`](../../remake/assets/cutscenes/bindings/ch15_post.json) 已接入
`postbattle_ch16_persist`；原始位址仍保留在每個 beat，證據等級為「已證實」的
IDA 控制流轉錄，尚非未修改一般玩家 E2。

- `ch16.json` 現明確使用 `runtime_append_groups=true`、`initial_groups=[0]`，
  讓 LOADCH 的 16 個 persistent party records 先於 map15 group0 的 60 筆
  FDFIELD records，形成原版 `16+60=76` runtime slots；未使用 position resource
  的 86 作為 runtime count。
- Docker/Xvfb 真實回歸涵蓋四條互斥路徑：`round>18`、slots66..73 的
  raw `+5 bit0` inactive count `>4`、slot0 raw `+0x42<0x140`，以及
  `+0x42>=0x140` 的 dialog4→唯一 raw `+8=18`→JOIN18。前兩類與低 word42
  路徑不建立 JOIN18；成功路徑才建立 typed persistent roster record。
- 四路徑都在 handler 完成後進入 `town_ch17`，`handlerChapter=16` 且清除戰鬥
  暫態；JOIN18 路徑另在 town 邊界完成 save/load，確認隊伍順序與持久名冊恢復。
  失去 raw byte5、round、word42 或唯一 raw identity 時，既有執行器仍失敗即關閉。

本切片目前是 E1（原始位址、可編輯資料、重製端決定性垂直回歸）；尚缺未修改
一般玩家 DOSBox 的同狀態逐幀比較、完整第16戰輸入路徑及 `town_ch17` 的實機
E2。當時其餘 blocked postbattle 包含玩家第17、18、22、23、24、29戰；後續
第18戰已由下一節切片解除，現況以本文件最末稽核為準，不得因單一切片宣稱
整個戰役完成。

## 2026-08-09 勘誤：raw ch17 post 是玩家第18戰（E1）

前一節把「剩餘 blocked」列為玩家第17、18戰，是第17戰切片尚未完成時的
歷史狀態；本節以索引稽核與直接位址證據取代，不刪除前文以保留推翻原因。
`battle_ch18` 的 map17 才由 raw `ch17_post`（handler `0x23cd5`）消費；
`battle_ch17` 仍等待 raw `ch16_post` 的正式 binding。把 `ch17_post` 接到
`postbattle_ch17_persist` 會被稽核判定為 `active_index_mismatch`，已撤回。

- `sub_23CD5` 的 `0x23d39→sub_233C6` 直接參數閉合 slots 0..16 及 special
  slot17 `(25,8,pose1)`，相機為 `(432,96)`；IDA 原始位址、三組 0x11 位元組
  與 FD2.EXE 雜湊見 [`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)。
- acting directory 的資源56、57、58已由固定版本原版直接解碼，編輯稿位於
  [`ch17_post.json`](../../remake/assets/cutscenes/acting/ch17_post.json)，不
  保存指標或原始執行檔位元組。
- FDTXT_018 index10 明確跨越 `ch18.json` scene3 line9 與 scene4 lines0..10；
  binding 使用 `dialogue_overrides.segments`，禁止把跨場景字串壓成單一場景。
- map17 的 group0 37筆加 18個 persistent party records 形成55-slot runtime；
  JOIN21／7 的 current runtime array 沒有 raw +8 記錄，原版 sub_112A5 直接從
  角色資料表建構 persistent record。重製因此只接受已標示「強推論」的
  `native_join_base_units.json` 明示 map17 base，未知角色仍失敗即關閉，不能
  用 `Fig==char_id` 泛化。
- Docker/Xvfb E1 已驗證 `battle_ch18→postbattle_ch18→town_ch19`、演出、
  JOIN21／7、持久隊伍與 save/load。這不等於未修改一般玩家 DOSBox E2，也不
  解除玩家第17、22、23、24、29戰的 blocked 狀態。

（歷史快照；後續 `ch16_post` 接入已取代）本次索引修正當時的舊覆蓋統計
不再作為現況依據；
story/cutscene 稽核為 121 節點、9 個獨立 script、49 個 handler binding、
63 個 fallback。這些是覆蓋統計，不是完成百分比。

## 2026-08-09 勘誤：raw ch16 post 接入玩家第17戰（E1，最新）

前兩節的「玩家第17戰仍 blocked」是本輪之前的歷史狀態，已由本節直接驗證
取代；raw `ch17_post` 仍只屬玩家第18戰，不能交叉套用。
`sub_23B5F`（`0x23b5f..0x23cd5`）的兩條 `roster_has(18)` 分支已保存原始
指令順序：有18走 FDTXT_017 index5、PAN、group3、ACTING52；無18先走
layout、index7、ACTING50、PAN、group3、ACTING51；共同尾端依序為 index6、
ACTING53、index8、JOIN16、chapter17。共享 call site 的 ACTING immediate
以來源位址索引保存，沒有把兩條演出誤合併成單一語意。

- IDA Pro 9.4 的函式範圍、`sub_233C6` 的 16 格 table、特殊 slot52、camera
  與 FDTXT／acting 位址，連同固定版 FD2.EXE 雜湊，見
  [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)。資源50–53已轉成
  [`ch16_post.json`](../../remake/assets/cutscenes/acting/ch16_post.json)；每筆
  raw 位址仍保留在證據檔，語意分級為已證實，runtime frontier 則標為強推論。
- binding 限定 slot count `60` 或 `61`，compiler 靜態分支使用上界、runtime
  仍要求精確 count；有18路徑為 60→61，無18路徑為 61→62。測試明確建立
  ch16 pre 的 group1 與 battle event group2，再由 post handler materialize
  group3，避免把 position resource 的 86 筆記錄誤當 runtime slot 數。
- Docker/Xvfb 真實回歸已覆蓋兩條分支、ACTING／dialogue、JOIN16、
  `battle_ch17→postbattle_ch17→town_ch18` 及 town save/load；缺少 raw round、
  byte5 或唯一 identity 時仍失敗即關閉。這是 E1 垂直切片，未修改一般玩家
  DOSBox 同狀態 E2、完整第17戰輸入路徑與其餘 blocked 章節仍未完成。

這是 2026-08-09 當次稽核快照：24 個標準 postbattle 節點當時為
**19 active／5 blocked**；其後玩家第22戰曾使當時統計成為20 active／4 blocked，
其後2026-08-13又成為 **21 active／3 blocked**；以上皆為歷史數字，現況已是
2026-08-21後續接線的 **24 active／0 blocked**。現況以
[`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md) 為準。這些統計只表示
節點接線狀態，不是重製完成百分比。

## 2026-08-09 raw ch21 post 靜態證據（玩家第22戰；尚未接入）

授權 IDA Pro 9.4 已以固定雜湊的 `FD2.EXE` 重建一次性資料庫，閉合
`0x244b6` 的直接呼叫序列、`0x24512→sub_233C6` 的三組 16-byte raw layout
表、FDTXT_022 索引4/5/6、ACTING raw immediate 65/66、兩個 PAN、
`0x245ce→0x24618` 的九段 indexed transition（LUT 9→1、每段 5 ms、
固定目的地 `0xA0504`、500 ms 尾端與 0..62 調色盤序列）及
`0x1f882→sync→chapter22`。同一工具鏈再追出
`sub_1088D→sub_10B4E→sub_10C50` 的 map21 loader：固定資源 header 是
16 個持久名冊槽、70 筆控制列，map21 raw group0/1/2/3 的事件追加後候選
frontier 為 **66→72→73→79**（強推論）。完整位址、輸入雜湊、工具與
原始表見 [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)。

這提升了 runtime 拓撲證據，但不等於畫面或正式 handler 已閉合：IDA 已證實
`0x233c6` 的 16 格 layout、special slot72 與 raw camera 參數，Docker exporter
也已將 acting 65／66 轉錄為可編輯 frame 表；候選 binding 見
[`ch21_post_candidate.json`](../../remake/assets/cutscenes/bindings/ch21_post_candidate.json)
及 [`ch21_post.json`](../../remake/assets/cutscenes/acting/ch21_post.json)，屬 E1
資料消費證據，不是一般玩家畫面證據。由於候選 runtime frontier 是
**66→72→73→79**，而原版 `[0x53a45]` 的全域配置是 96 個 `0x50`-byte
槽、`[0x53beb]` 只是追加 count，不能把 66 寫成物理容量上限。候選仍未證明
slot72 在每一個宣告入口都已 materialize；編譯器因此以「最小已 materialize
frontier」作保守閘門，對此候選明確失敗即關閉並不產生 `runtime_context`，保留
原始 slot72 證據而不猜測短前沿的 record 內容或 renderer 消費。各 frontier
的 record 內容、原版 runtime trace 與 indexed 畫面狀態仍未閉合。`0x24618` 所消費的 raw 相對游標 globals
（`0x53ab9/0x53abd`）與第 21 戰呼叫點 `0x245ce` 的 Y+3 變換已由 IDA 固定；
重製端以呼叫位址核對這個來源專屬橋接（source-specific bridge），未證實來源或偏移會失敗即
關閉。仍未建立正式 binding。`postbattle_ch22_persist` 仍不得由 layout 表或
generated binding 猜接 `town_ch23`；缺證據時維持失敗即關閉。本節不宣稱
renderer parity 或一般玩家 E2。

## 2026-08-09 raw ch22 pre 靜態候選（玩家第23戰戰前；尚未接入）

合法 IDA Pro 9.4 與 Docker Capstone 以固定雜湊的 `FD2.EXE` 重讀
`0x336a0..0x338c0`。已證實 `0x336ab→0x205da` 載入 context、
`0x336b5→0x32975` 的 16 次 `EBX=0..15` loop、三個 PAN
`(14,32)/(14,29)/(14,13)`、`0x336e5→0x24618` 的 raw push
`[0x53abd]+5`／`[0x53ab9]+6`／`10`／`8`，以及 redraw、palette、
FDTXT_023 0..4、ACTING 68..70、group1 spawn 與 focus/reset 順序。完整
位址、工具、雜湊與證據等級見 [`fd2_ch22_pre_ida.txt`](../data/ida/fd2_ch22_pre_ida.txt)。

重製端已建立可編輯 handler 與研究 candidate binding：
[`ch22_pre.json`](../../remake/assets/cutscenes/handlers/ch22_pre.json) 與
[`ch22_pre_candidate.json`](../../remake/assets/cutscenes/bindings/ch22_pre_candidate.json)。
候選 compiler regression 保留 map22／70 slots／group1 24 rows、五個
FDTXT source、ACTING 68..70 與所有 raw call-site；`0x24618` runtime bridge
只接受來源 `0x336e5` 的 Y+5，並與第 21 戰 `0x245ce` 的 Y+3 明確分開。
這是 E1 的資料消費候選，不是正式 campaign binding；`story_ch23`、
`postbattle_ch23_persist→preparation_ch24`、indexed renderer、一般玩家 DOSBox E2
與戰後城鎮／商店／整備存檔流程仍保持失敗即關閉。

## 2026-08-21 raw ch22 post `0x2189a` 索引呈現轉接規格（玩家第23戰戰後；原語 E1）

合法 IDA Pro 9.4 與 Docker Capstone 以固定雜湊的 `FD2.EXE` 閉合
`0x24754` 戰後 handler 的三個 `0x2189a` 呼叫點（`0x24978`、`0x249c4`、
`0x24a10`）。IDA 偽代碼與指令層共同證實十次外層迴圈、`work+0x8088`、
stride 456、13×8 原始場景建立、312×192 呈現區域、320 呈現 stride，以及
`0x21914`／`0x21955`／`0x2195d`／`0x21986`／`0x219a3` 的巢狀呼叫順序。
原始 push 形狀與 caller 第九參數步進來源均保留於
[`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。

重製端已有 `native_2189a_loop` 可編輯原語，將三個呼叫點與所有 raw 引數
寫入 [`ch22_post.json`](../../remake/assets/cutscenes/handlers/ch22_post.json)，
並以 compiler regression 驗證幾何常數、巢狀位址及錯誤 payload 失敗即關閉。
2026-08-21 的 IDA 勘誤進一步證實外層三個實參依序是 runtime unit slot、初始
半徑與每輪半徑步進；因此執行期只接受 `(10,15,1)` 與 `(16,30,1)` 兩種已知
呼叫形狀，不把它泛化成肖像或任意特效。

正式轉接器必須符合下列契約，完成前不得接入 campaign：

1. runtime slot 必須存在且保有原版座標來源；以 `(unitX-cameraX)*24+12`、
   `(unitY-cameraY)*24+18` 建立畫面中心。缺 slot、相機、原始地圖或 indexed
   素材時，整個 beat 不得改變可見畫面或戰役狀態。
2. beat 開始時只建立一次 13×8、stride 456 的 terrain staging snapshot；十輪
   每輪先還原完整 snapshot，再依序取 FDOTHER resource3 的 LUT 0..9。每張 LUT
   必須恰為 256 bytes，任何一張缺失都在第一輪前失敗即關閉。
3. 第 `i` 輪半徑為 `initialRadius+i*radiusStep`；只在 312×192 viewport 內執行
   `0x219ad` 的向零截斷橫向半徑與 LUT remap，接著以既有 native unit/object
   規則重畫物件，最後將 312×192 區域發布到 320-stride indexed VGA surface。
   每輪發布一個可繪 frame；這是重製端 60 Hz 的 E1 排程，不宣稱 DOS 指令週期
   或逐像素時間完全一致。
4. 十輪成功後才執行等價 `0x11cac(0)` 的穩態 native frame；所有 staging 必須
   採先驗證後發布，錯誤時保留 beat 前的 work buffer、VGA surface 與 callback。
5. 相鄰 `0x24b4d(30)` 不是一般淡入。typed adapter 必須先以當下 raw
   FDFIELD／FDSHAP／camera 建立 `work+0x8088`、stride456 的13×9 terrain
   staging，再執行一次完整 `0x11cac(0)` 穩態畫面。之後第 `i` 輪只把
   `work+0x8088+456*(i&1)` 起始的312×192區塊複製到320-stride VGA，等待20 ms
   才進下一輪，共30輪；不得每輪重畫 terrain／unit，也不得用只有 timer 的
   `transitionRevealJob` 冒充。開始前產生完整候選 staging、穩態畫面與兩張交替
   viewport，任何失敗不得發布第一張；每輪必須經 Draw acknowledgement，完成後
   才執行 continuation。60 Hz 下每個20 ms等待採至少一個可見 frame 的 E1 排程，
   不宣稱 DOS 毫秒級時序完全一致。
6. `0x4dbfc` 以完整 `NativeMapEventGrid` 原地套用 `+1 &= 0x03`、
   `+2 &= 0x1f`、`+3 = 0xff`；必須先驗證 header 與長度，並以複本確保失敗時
   不留下半套修改。
7. `0x24a4b`／`0x24a65`／`0x24a7f` 的三次 `0x111ba` 已由直接 push、raw
   filename 與 loader ABI 閉合為 `FDFIELD.DAT#69`、`FDSHAP.DAT#46`、
   `FDSHAP.DAT#47`，並分別回寫 `[0x53a51]`、`[0x53a5d]`、`[0x53a69]`。
   typed binding 必須同時保存 archive 名稱、index、舊 handle owner 與回寫 owner，
   不能只留一個無檔名的 `resource_id`。三者須在候選 bundle 中一次解碼並驗證
   FDFIELD header／cell count、FDSHAP offset table／tile graphics／四位元組 control，
   成功後才原子取代目前 native terrain bundle；`0x4dbfc` 隨後只操作新 FDFIELD
   #69 的完整 raw grid。`0x11eee` 已由 IDA 證實直接消費這三個 owner。
8. `0x10652` 在本 handler 已將 chapter 設為23後，typed adapter 只接受 case23：
   配置59904-byte staging、解碼 `FDOTHER.DAT#42` 到312-stride buffer、釋放暫時
   resource，再執行 `0x24d22(0)` 對應的 indexed stage。不得沿用 `0x10652` 其他
   chapter branch，也不得只記錄「prepare」而不建立後續 ACT73 可消費的 buffer。
   `0x24d22` 的逐列 rotation 已有 raw ch23 post adapter 證據，可重用其 typed
   primitive。IDA 已補證 ACT73 的15-beat high-bit 路徑每 beat 呼叫
   `0x11cac(0)`，後者經 `0x11eee` case23 間接消費 `[0x53aff]`；正式 binding
   必須以同一份 decoded acting73 驗證這個 slot0／pose0／15-beat consumer，
   不得把 #42 另畫成不經 indexed compositor 的 RGBA 圖層。
9. campaign 驗收鏈為 `battle_ch23 → postbattle_ch23_persist → preparation_ch24
   → story_ch24`，並在 preparation node 驗證 JOIN19／22 分支成果、隊伍同步與
   save/load；三條 raw predicate 缺 provenance 時仍須停止。這只可提升為 E1，
   未修改原版的一般玩家同狀態畫面仍另列 E2。
10. `0x247b4→0x233c6` 的 layout 固定為 slots0..16 的17筆 table，加上 special
    slot17 `(21,21,2)`；camera 固定 `(14,14)`。X table 是
    `[20,20,18,19,20,21,22,18,22,18,19,21,22,19,20,21,19]`，Y table 是
    `[19,17,18,18,18,18,18,17,17,16,16,16,16,15,15,15,21]`，pose0..16
    全為0。binding 必須 materialize 全18筆，不得把舊「16×17」散文當成維度，
    也不得漏掉 special slot17 的最後覆寫。
11. runtime constructor 必須遵守 `0x1088d`：先以16個 persistent／部署 records
    建 slots0..15，再由 `0x10b4e` 依 group 追加 map22 FDFIELD records。
    map22 的70筆 raw groups 全 materialize 時 frontier 是86；舊 binding 的
    `slot_count=70` 只描述 map records，不能用於戰後 handler。重製端現有
    `initial_groups=0..9` 是「全部敵人已在場」的 authored 近似行為；本切片只把
    它搬到 party-first runtime append 路徑，保持目前玩家可見參戰集合不變，同時
    修正 slot identity。event52 在 rounds13／15／18／22 的精確追加時序另列
    `BLOCKED`，不得由本次86-slot E1 宣稱原版增援 parity。
12. 正式 binding 只接受 runtime slot count86、story viewport、上述 layout、
    acting71／72／73、三個帶 archive/index/owner 的 reload，以及所有已證實
    indexed／palette primitive。完整 handler 編譯不得有 issue；缺 original asset、
    raw predicate provenance、slot、camera、grid 或 staging 時仍在 campaign
    mutation 前停止。
13. register-bound `0x11df2(EBX,255,0)` 必須以來源位址展開：本 handler 的
    `0x24a24` 是 `EBX=0,2,...,62` 共32次，每次緊接 `0x24a2e` 的4 ms；相鄰
    ch28 `0x256eb` 是0..63，`0x25733` 與 ch29 `0x258cd` 才是62..0。
    compiler 展開 loop body 後必須消費緊鄰的單一 exported delay beat，不能把
    它再附加成第33／65／64次等待，也不能以一個「EBX一律下降」註解泛化。

`0x2189a` typed adapter 已依上述契約實作：compiler 將三個 raw immediate 固化到
typed payload，runtime 一次預驗證 LUT0..9、候選產生十個 effect frame 與一個
`0x11cac` 穩態 frame，逐幀發布後才執行 continuation；缺 LUT0 的回歸證實 buffer
維持不變。`0x4dbfc` 原子公開操作已與三資源候選 bundle 接線：三個 reload 完成
後只在候選 grid 執行 mask，FDOTHER #42 完整 preflight 成功才原子發布新
map／assets／state／staging；錯誤 archive、owner 或資源形狀不改 live state。在第7項 resource
`0x24b4d` typed adapter 也已建立13×9候選 staging、穩態 frame 與兩張 row-shifted
viewport，經30次 Draw acknowledgement／20 ms E1排程後才 continuation；缺第九列
時不修改既有 buffer。layout、runtime constructor 與正式 binding 現已依第10至13項
落地；正常玩家戰果確認回歸由16筆持續隊伍＋70筆 map records 建立86-slot
frontier，跑完正式 handler、同步隊伍後進 `preparation_ch24`，並在該節點通過
存檔／讀檔。因此本切片為 `RUNTIME-E1`；event52 的精確增援時序與未修改原版
同狀態畫面仍分別維持 `BLOCKED`／`PLAYER-E2` 待補。

同一輪又把 `0x247c6` 的 `cmp eax,-1`、`0x24840` 的 persistent lookup
分支，以及 `0x248b5` 的 `cmp [0x53bef],15; jl` 轉成可編輯巢狀 `if`：
`native_inventory_item_present(100)`、`native_persistent_identity_present(18)`
與 `native_round_lt(15)`。這些是原始 byte-level predicate，不是角色或物品
名稱；compiler 與 BeatRunner regression 只在完整 raw inventory／persistent
record provenance 存在時選臂，缺資料便停止。這改善了 handler 的控制流可編輯性，
正式 binding 現已以這三個 predicate 執行；缺 raw provenance 仍會失敗即關閉。
這不解除 event52 時序或一般玩家 E2 gate。

## 2026-08-09 raw ch24 post `0x24df2` 函式邊界與 owner 勘誤（E1）

固定版 `FD2.EXE` 的 IDA Pro 9.4 與 Docker Capstone 共同固定 handler table
index24 的 raw entry `0x14df2`（線性位址 `0x24df2`），以及相鄰 index25 的
`0x14e80`（線性位址 `0x24e80`）。完整輸入雜湊、檔案／線性位址與逐項指令見
[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。table index 只代表
raw dispatch，不直接代表玩家戰次。

`sub_24DF2` 的已證實順序為：FDTXT_025 index6 → `0x135dd` raw PAN
`(4,16)` → `0x10b4e(2)` → ACTING resource75 → FDTXT_025 index7 →
`0x112a5(0x1a)` persistent append → `0x11506` → push raw `0x1d` 跳入共享
`0x237c8` 尾段。Capstone 顯示 `0x237c6` 是 direct-entry 的 `push 0x0e`，
`0x237c8` 本身是 `call 0x112a5`；因此 `0x24e7b` 的中途跳入會跳過固定
14，讓 `0x112a5` 消費呼叫者壓入的 `0x1d=29`。ch12 的 `push 3→jmp 0x237c8`
與 ch14 的同型呼叫是交叉證據。固定角色表與 authored map／party 交叉確認
26 為聖寇拉斯、29 為亞奇梅吉；現行可編輯 handler 已保存為 `join` 26 與
`join` 29，原始位址仍保留在 `source.addr`。這是角色操作的強推論，並不等於
戰後 handler 已能直接消費目前重製端的 battle State。
補充追查已證實 `0x10b4e(2)` 會以 FDFIELD row `+0x15` 比對 group，逐筆呼叫
`0x10c50` 建立 runtime record；map24 resource 073 的 70 筆列中 group2 恰有
1 筆，故候選 binding 的 `spawn_groups["2"] = 1` 是可重生的列數投影。這只
閉合增援 materializer 的靜態列數；26／29 的 JOIN 角色判讀仍只到強推論，
不解除一般玩家與戰間節點的 E2 gate。

現有 map24、FDTXT_025、ACTING resource75、26／29 的 map／party 對照支持
「`0x24df2` 是玩家第25戰 post handler」；先前同號接到
`postbattle_ch24_persist → town_ch25` 已撤回。

**2026-08-13 勘誤與執行期閉合（E1）：**舊說把完整70筆 map control 預載後再加
16名隊員，誤造86-slot battle State。原版追加式拓撲其實是開戰先建立16名隊員與
group0的46筆，形成62；第6回合 event56 在`0x3549f`呼叫`0x10b4e(1)`，再追加
group1的8筆成為戰後入口70；`0x24df2`隨後呼叫`0x10b4e(2)`，追加唯一 group2
成為71，ACTING resource75才操作新建立的slot70。group255沒有任何初始或事件
materializer，不進 runtime。`ch25.json` 已改採 `runtime_append_groups`，正式
`postbattle_ch25_persist` 綁定 `ch24_post.json`；Docker／Xvfb 回歸實際走過
`battle_ch25 → postbattle_ch25 → town_ch26`，驗證 JOIN26／29、`sync_party`、
章節25與 town node-boundary save/load。這解除 runtime E1 gate；尚缺未修改一般
玩家原版同狀態 E2，不宣稱逐像素或完整玩家路徑一致。

## 2026-08-09 raw ch29 terminal body `0x2bce5→0x2c405`（E1）

合法 IDA Pro 9.4 與 Docker Capstone 以固定版 `FD2.EXE` 重讀終局函式完整範圍；
位址、函式邊界、雜湊與逐段 raw 順序見
[`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)。
`sub_2BCE5` 的兩個 caller 仍是 `0x2545d` 與 `0x25970`；後者在 `0x25975`
自迴圈，不能被寫成 campaign 返回。前綴已證實 resource `FDOTHER.DAT#0x36`、
frame0／frame9、`0x11df2` 的 `0x3f..0` ramp、三輪重複 ramp、frame
`0x0c..0x6c`、40 次與 200 次 indexed composite，以及於 `0x2c172` 呼叫
獨立 `sub_2C405` 的順序。

`sub_2C405` 另以 raw `0x1088d(0x1e)` 建立終局 staging，先跑 500 次
`0x11d40`／`0x11eb0` 迴圈，再於 `0x2c548` 配置 TAI／FDOTHER／FIGANI／DATO／
FDTXT 所需的 montage context，依 `[0x53bfb]-1` 逆序處理 runtime record。
這補足了「前綴已 lower」與「完整終局可接 campaign」之間的證據差距，但仍不
證實 `0x1e` 是玩家戰次或下一節點。

輸入方面，`0x10620` 只比較 absolute `word[0x41a]`／`word[0x41c]`，
`0x4e031` 只複製前者到後者；兩者都有其他 caller。`0x2c950`／`0x2c961`
因此只能保存 raw 字組變化閘門，不能猜成 Enter、Space、Esc 或 generic skip。
`MontageCycle`、`Player` 與 `MontageTail` 維持獨立可驗證執行器；indexed owner、
輸入事件、一般玩家 E2 及 battle→postbattle→town/shop/preparation/save
campaign handoff 仍失敗即關閉。

## 2026-08-09：戰間與指令格的窄回歸邊界

- 天空之鑰材料不足分支已由 `TestInventoryRecipeInsufficientReturnsToTownAndSaveLoad`
  實際驗證 `recipe→insufficient→town`，在 town22 等價節點保存後清除暫態並讀回
  持續隊伍；這只補可編輯戰間／存檔邊界，不解讀 raw `ch21_post`，也不提升一般玩家 E2。
- `TestNativeCommandGridMissingInputsRemainInert` 只驗證空 command ID 與越界選取值的
  純資料失敗即關閉契約；它不開放未知 command、效果或 renderer 語意。兩項回歸均在
  Docker 中執行，UI-03／UI-12 的其餘原版證據門檻維持不變。

## 2026-08-09 raw index28 `0x2548c`（玩家第29戰 post 候選；E1）

raw handler table index28 的 bytes `8c 54 01 00` 解析為線性位址 `0x2548c`，
與 index29 的 `0x25757` 是兩個不同函式。IDA／Capstone 證實 `sub_2548C`
由 `0x25e23` table consumer 呼叫，順序包含 FDTXT_029 string10／11／12／13／
14／15、`0x35bba(20)`、raw `0x10b4e(9)`、PAN raw `(9,8)`、
`0x12cea(10,15)`、`0x22253` 的五個 raw push、三段 `0x24b4d(20)` 過場、
三次 `0x35e5a`、兩段 `0x11df2` 調色盤迴圈，最後 `0x11506` 與
`inc [0x53c03]` 返回。逐項位址、雜湊與文字跨場景 mapping 見
[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。

map28（現行 `battle_ch29`）、FDTXT_029 count-aligned index10..15、raw
index28 與最後的 chapter-global increment 共同支持「玩家第29戰 post handler
候選」；不再把 index29 的 terminal self-loop 當成第29戰。【2026-08-21 勘誤】
`0x25535` 專用 `0x22253` battle-state presenter、`0x24B4D` 與 `0x35E5A`
indexed DAC presenter 已達窄 E1；仍由第29戰 runtime topology、dialog／pan binding、
正式 save flow 與未修改一般玩家 E2 阻擋，因此不能把現有
`postbattle_ch29_persist` 接到 `preparation_ch30`。campaign、城鎮／商店／整備／
存檔仍採 fail-closed；這一節不宣稱正式 binding。

補充 raw 邊界：`0x35bba(20)` 的完整 body 已固定為從 runtime index20 起，
逐筆清除 0x50-byte record 的 `+0x40`，再呼叫有七個其他 caller 的共用
`0x1db65`。`0x1db65` 讀取 raw `+0/+1/+5/+0x40` 並進入共用 indexed
呈現／更新鏈；這只刪除「`0x35bba` 完全未知」的過時斷言，不命名 `+0x40`
成 HP／狀態，也不把 `0x1db65` 接成 ch29 renderer 或 campaign owner。
indexed frame、一般玩家 E2 與 battle→postbattle→town／shop／整備／save
仍失敗即關閉。完整 raw 位址與雜湊見
[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。

## 2026-08-09：玩家近戰選單到結算的亂數邊界（E1 重製端）

玩家在戰場 action menu 確認近戰目標後，`Game` 現以注入的遊戲亂數呼叫
`battle.State.AttackWithRNG`，不再經由會消耗程序全域亂數且丟棄結算欄位的舊
`State.Attack` 介面。`AttackResult` 的未命中、暴擊、傷害與經驗資料會保留到
訊息／演出橋；缺少 `Game` 亂數來源時先停止，避免部分 mutation。這只證明
重製端「選單→型別結算→演出」的消費鏈與固定種子可重現性，不把青衫公式升格
為原版攻擊 ABI，也不宣稱命中表、劍技、完整經驗 UI 或一般玩家 E2 已閉合。
回歸涵蓋確定命中、缺少亂數的失敗即關閉，以及訊息保留未命中／暴擊／經驗。
正規化 `aiStep` 也共用這個亂數邊界，缺少來源時停止而不標記單位已行動；這只是
重製端 deterministic consistency，不能當成原版 native AI 回合執行證據。

## 2026-08-09：指令環疊加完整原生戰場畫面（E1 重製端）

Docker／Xvfb 實際抓圖發現：完整 `FDOTHER.DAT` 存在時，指令環與原始 command
grid 原先被錯誤排除在 `drawNativeMapFrame` admission 之外，短地圖會在畫面下方
留下黑帶。現在只有已 materialize 的完整 native frame 才允許 ring／command grid
疊加；缺資源仍走既有可玩回退，不以黑帶或猜測補畫原版資料。回歸鎖定 modal
admission，並保存目前 source 的
[完整戰場指令環畫面](../figures/action-overlay-native-remake-fullframe.png)。
這是重製端 renderer 接線修正，不是原版 DOSBox 逐像素差分。

## 2026-08-10：戰鬥狀態欄姓名字模消費端勘誤（E1）

全螢幕戰鬥狀態欄的框、血條與數字原本已使用原版素材，但姓名路徑仍使用
28 像素現代字型，造成 GitHub 戰鬥演出圖的字形與原版不一致。`35` 的 IDA／
Capstone 證據已確認 `0x18c6d→0x15f84→0x4ea2a` 消費 FDOTHER#4 的 16×16
1bpp glyph；重製端現以可版控的 `FDOTHER_004.bin` 與
`unicode_to_glyph.json` 逐字建立索引畫面，再以原版前景／陰影索引放大 2×。
缺少字模或 Unicode 對映時才退回既有 TTF，未知字元不會被猜測成其他 glyph。

Docker／Xvfb 實際產生 [`battle-native-name-remake.png`](../figures/battle-native-name-remake.png)，
完整輸入雜湊與命令見 [`battle-name-glyph-ch01.json`](../data/ui-traces/battle-name-glyph-ch01.json)。
這只關閉「狀態欄姓名字形」的重製端 E1 消費端差異；攻擊幀時序、傷害／閃紅、
台座、完整 command／spell／item presentation 及未修改一般玩家同狀態 E2 仍維持
失敗即關閉，不能把這張圖宣稱為整個戰鬥介面完成。

## 2026-08-09：戰鬥回合至戰後城鎮的 runtime 垂直切片（E1 重製端）

`Game.endTurn` 現在以實際 `endTurn → aiStep → finishTurn → completeTurn →
checkResult` 完成敵方回合；勝負結果停留在 battle node，只有玩家結果畫面的
Enter 才經 `confirmBattleResult` 進入可編輯的 postbattle cutscene。回歸中的
postbattle beat 先執行 `sync_party`，再執行章節標記並經淡出 callback 進 town，
因此不會把戰後隊伍同步或城鎮邊界折疊成「直接下一戰」。

`TestEndTurnEnemyPhaseResultEntersPostbattleCutsceneThenTown` 在 Docker／Xvfb
通過，並檢查持續隊伍快照與結果清除。這是最小可重現的 E1 runtime 邊界；敵方
目標選擇、原版逐章 handler、一般玩家 DOSBox E2 及 town/shop/save 的完整玩家
路徑仍未宣稱完成。

### 2026-08-10：戰場命中色盤脈衝採失敗即關閉

固定版 `FD2.EXE` 的 IDA／Capstone 證據已閉合 `0x2939d` 的條件式 DAC 分支：
只有 frame record `+4 == 1`、傷害步進完成，且 `0x29f72` 的兩個原始輸出欄位
非零時，才執行索引 0 的兩段色盤寫入與 20/40 毫秒等待。證據與可重生命令見
[`fd2_battle_impact_pulse_ida.txt`](../data/ida/fd2_battle_impact_pulse_ida.txt)。

重製端的正規化 `AttackResult` 尚無這些欄位的位址來源，故移除原先無條件的
RGBA 全畫面紅罩；目前只保留原版 impact 參考圖支持的守方紅色剪影，攻方維持
原本 FIGANI 幀，不把它或其他欄位猜成原始 DAC 脈衝。這是避免 GitHub 戰場圖
產生整片泛紅或攻方對稱染紅的安全修正；原始輸出轉接器、完整攻擊時序及一般
玩家 E2 仍是未解除閘門。

同一命中 fixture 另確認狀態欄的可見邊界：原版在 impact 開始時立即顯示
post-hit HP，重製端已移除沒有 raw provenance 的 8 tick 中間值；守方剪影 E1
近似色值對齊原版擷取 RGB `(190,0,0)`。正規化原版／重製／差異遮罩保存於
[`battle-impact-compare-20260810.png`](../figures/battle-impact-compare-20260810.png)，
逐 RGB 差異仍有 3933 像素，故這是兩項可見差異的窄修正，不是完整戰鬥 UI E2。

守方 FIGANI 待機幀也已撤回固定 `(prog/6)` 選幀，改由 descriptor `+6` 與
`FD2_BATTLE_FPT` 的純排程橋消費；攻守任一延遲表缺失即失敗即關閉。這只收緊
renderer 輸入契約，不能把排程本身解讀成命中、傷害或 DAC 語意。

## 2026-08-10：人工智慧與結局音訊的證據閉合更新

本輪把原始人工智慧與終局音訊拆成可驗證的邊界，沒有把尚未證實的演出語意
接入正式劇情：

- `0x14EF0` 的 runtime bridge 現在嚴格要求 `0x14237`、`0x1598A`、
  `0x1567E` 三個 producer 的原始結果，以及 actor／target `+0x34`、
  `+0x48/+0x4A` 與 command/item 來源；依 IDA 已證實的 raw tree 分派到
  `0x1548E`、`0x15311`、`0x15055`。這是 E1 窄執行切片，不是完整 AI 或 E2。
- mode 3／9 的 `0x12C60` raw `+0x35`→`+0x08` first-match、mode 5 的
  `0x15DF3` mutable event grid／field-control row／`+0x31..+0x33`／
  `0x53AD5`／`+0x34=7` state tail 已有可編輯資料與 transactional adapter。
  mode 5 的 `0x25B45` raw AIL sample（resource `[0x53EE8]`、index `12`、
  loop `1`）已接到同一 FDOTHER #31 導出的 `sfx_12.wav`，缺樣本時失敗即關閉；
  `0x17AA9` 不是 mode 5 的 direct caller。mode 11 的雙動作
  transaction，以及 `0x13FD4` 的 recovery presentation 仍 fail-closed。
- 指令、法術與物品決策只在 raw command／item effect route 完整時執行；
  unknown command、未閉合 relocation、零分勝者與 spell presentation 不得
  由 normalized 名稱或一般玩家 UI 推回。證據見
  [`fd2_ai_mode5_event_ida.txt`](../data/ida/fd2_ai_mode5_event_ida.txt)、
  [`fd2_ai_mode5_full_ida_20260810.txt`](../data/ida/fd2_ai_mode5_full_ida_20260810.txt) 與
  [`fd2_ai_mode11_13fd4_ida.txt`](../data/ida/fd2_ai_mode11_13fd4_ida.txt)。
- 終局播放器新增三個由 IDA 直接確認的事件：`sub_2C405` 前段
  `0x2C5CF→FDMUS_004`、`sub_2BCE5` 尾端 `0x2C1AC→play_bgm(-1)`，以及
  `0x2C1F5→FDMUS_018`。2026-08-12 起，唯一已獲 runtime 消費權的 cue 是
  `FDMUS_004`：它必須在精確 `0x2C548` 邊界，且只由
  `FD2_APPROXIMATE=1` 的嚴格 `native_ending_prefix` 最終節點消費；確認後
  回到可編輯結語。停曲與 `FDMUS_018` 仍僅保存為觀測事件。完整 `0x2BCE5`
  indexed montage、原始輸入交接、raw ch29 terminal owner 與一般玩家終局 E2
  仍未接通，因此不能宣稱原版結局演出或逐音符一致。

補充一個不改變執行期狀態的 mode 11 E0 邊界：
`SelectNativeAIMode11Transaction` 只驗證兩個 raw score 是否存在，並保留
`[0x53C23]`／`[0x53C4F]` 各自的 `>=6` 路由選擇；`0x14121` 以獨立的 mode 11
路由型別表示，避免與 `0x14EF0` 尾端混稱。它沒有 transaction owner、演出或
指令／法術／物品效果，因此不能解除 mode 11 的 E1／E2 閘門。

`PlanNativeAIIdleRecovery` 同樣只保存 `0x13FD4` 的 raw HP／gate 判定與
`max/5` 封頂結果，`ApplyNativeAIIdleRecovery` 才在同一 raw snapshot 上提交
寫入。這是可測的 state-only 邊界，不是 `0x12D7B` presentation、mode 2
handoff 或一般玩家回合證據。

### mode 11 直接控制流勘誤（E0）

合法 IDA Pro 9.4 與 Docker Capstone 已固定 `0x13A9F` 的 mode 11 分支從
`0x13E02` 直接呼叫 `0x1598A`，接著無條件呼叫 `0x14237`；它不是先經
`0x14EF0`。`[0x53C23] >= 6` 只控制 `0x15311`，`[0x53C4F] >= 6` 只控制
`0x1548E`，不足時才走 `0x14121`，且零回傳才進 `0x13FD4`。完整指令、函式
邊界、caller 與 raw writer 見
[`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt)。
重製端目前只保存無副作用的 E0 路由選擇，尚未接 transaction owner、演出或
一般玩家回合，因此維持失敗即關閉。

### `0x13FD4` presentation 邊界補證（E0）

完整指令重新固定 `0x13FD4..0x14120` 的三次 `0x17AA9(1)` 等候、兩次
`0x1DA16` raw 影格解碼，以及 `0x11EB0` 的 312×192、456→320 stride 緩衝區
拷貝；`0x19082` 另以未命名的原始 `a3==0` 作為 caller gate。證據見
[`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt)。
這只提升 raw presentation 邊界的 E0 證據；重製端仍只執行 state-only
`PlanNativeAIIdleRecovery`／`ApplyNativeAIIdleRecovery`，缺少同狀態原版逐幀
擷取時不得接入正式 renderer 或替參數命名為回血／音效效果。

## 2026-08-11：mode 11／`0x13FD4` 交易邊界與原版玩家回合錨點

本輪把兩個高影響 native 路徑整理成可測但不猜語意的交易邊界：

- `NativeAIMode11Stage`／`ExecuteNativeAIMode11Transaction` 僅執行已證實的
  stage 順序與 route 位址。第一段只有 raw `[0x53C23] >= 6` 才進
  `0x15311`；第二段永遠存在，由 raw `[0x53C4F] >= 6` 選 `0x1548E` 或
  `0x14121`。stage callback 缺失或失敗會停止後續階段。此 API 不提供
  command、physical、spell、item、movement 或 indexed effect 語意；正式 runtime
  owner 另由 `native_ai_mode11_runtime.go`／`native_ai_mode11_execute.go` 接線，
  因此不可把這個純 stage API 本身直接視為完整敵方 AI planner。
- `NativeAIIdleRecoveryPresentation`／
  `ApplyNativeAIIdleRecoveryWithPresentation` 保存 `0x13FD4` 已證實的
  sample、解碼、拷貝與 wait 參數；presentation callback 必須成功且不得改動
  raw record，才會以 `max/5` 封頂結果寫回 `+0x40`。缺 owner、callback 錯誤
  或 record 竄改會回復快照並失敗即關閉。該 API 保持原始 callback 契約；
  實際 indexed／音訊消費端由 `native_ai_idle_recovery.go` 另行提供，仍不替
  `[0x53EEC]` index 4 命名。
- Docker 的 `go test ./internal/battle -run 'NativeAIMode11|NativeAIIdleRecovery'`
  已通過；原版未修改複本另由 Docker DOSBox 走到第一戰第一個我方單位的
  玩家指令格，畫面與輸入時間線見
  [`native-player-turn-original.json`](../data/ui-traces/native-player-turn-original.json)。
  這是一般玩家回合 E2 原版錨點，不是重製端同狀態對照，也不解除敵方
  回合、逐幀／逐音訊 parity 或完整戰場 UI 的 E2 閘門。

### 2026-08-11 目前 owner 狀態勘誤

上一節的「尚未接線」是本輪開始前的邊界；目前已補上受 raw provenance
保護的 E1 消費端：

- mode 11 由 `native_ai_mode11_runtime.go` 建立兩段 stage，
  `native_ai_mode11_execute.go` 保持 `0x15311`→`0x1548E`／`0x14121` 的同單位
  continuation。`0x15311` 的已閉合 command／item family 重用既有 executor；
  `0x1548E` 使用 typed physical damage／FIGANI owner；缺 target、path、table
  或 raw provenance 時失敗即關閉。
- `0x13FD4` 由 `native_ai_idle_recovery.go` 消費 raw sample、indexed map
  compositor、FDICON selector 與修正後的 312×192 copy ABI，逐 frame 等待後才
  寫回 HP。音效只接受 `[0x53EEC]` index `4`／loop `1` 的現有 `sfx[4]`；
  `0x1DA16` mode、sample 高階名稱與色彩語意仍未知。
- 這些變更只把證據接到 E1 窄切片，不提升為原版一般玩家 E2、逐像素／逐音訊
  parity，也不替其他 `0x13FD4` caller 猜測玩法。驗收入口與剩餘 gate 以
  `docs/knowledge-base/91-worklist.md` 為準。

## 2026-08-11：玩家第22戰戰後→整備 E1 邊界

raw `ch21_post`（玩家第22戰戰後）已從候選提升為正式可編輯 binding：

- `remake/assets/scenarios/ch22.json` 明確使用 `runtime_append_groups`；正常
  整備選出的16名隊伍先建立，再依 map21 FDFIELD group0／1／2／3 的原始順序
  形成 66→72→73→79 的 runtime frontier。
- `remake/assets/cutscenes/bindings/ch21_post.json` 只接受 73 或79槽，因為
  raw `0x24512` 會直接寫 slot72；66／72槽在編譯或執行期停止，避免以未物化
  的記錄假裝演出可達。ACTING 65／66、三段 FDTXT_022、兩次 PAN、
  `0x24618` 九段 indexed transition、`0x1f882` 淡出、`sync_party` 與
  `set_chapter(22)` 均保留原始位址與可編輯 payload。
- `campaign_full.json` 的 `postbattle_ch22_persist` 現接至該 binding，完成後
  進入既有 `preparation_ch23`，不是直接跳到下一場戰鬥。Docker/Xvfb 回歸以
  group1+2（73）及 group1+2+3（79）兩個明確 frontier 實際消費 handler、
  持久隊伍與整備節點；原始資產採唯讀掛載。

這是 E1 的原始 handler→runtime→整備垂直切片，不是一般玩家 E2。未證實的
66／72-slot 結束狀態、玩家未修改 DOSBox 同狀態截圖、以及第23／24／25／29戰
仍維持失敗即關閉，不得由本切片推論完整戰役 parity。

## 2026-08-11：ch22 正式結果確認與整備存檔邊界（E1）

`TestChapter22BattleResultPreparationSaveLoadUsesProductionBoundaries` 將既有
ch22 runtime fixture 接到玩家可見的正式結果消費端：先以 73-slot（group1／2）
的已證實 frontier 建立 map21 狀態，再設定已完成的 battle result，呼叫
`Game.confirmBattleResult`；測試不直接呼叫 `Runner.Advance` 或 handler。既有
`postbattle_ch22_persist` binding 完成後進入 `preparation_ch23`，確認持久隊伍已
同步、battle array 已清除，並在該整備節點的隔離 XDG 目錄寫入自有 JSON 存檔。
重新透過 `loadGameFromSlot` 後，node、chapter、party roster、join order 與 deploy
狀態均保留，且不重新帶回 transient battle state。

這只關閉重製端 E1 的「戰鬥結果→戰後 handler→整備→存檔／讀檔」消費邊界；原版
未修改一般玩家 E2、同 raw 狀態的畫面／音訊差分、66／72-slot 入口，以及尚未綁定
的 ch23／25／29 戰後節點當時仍保持失敗即關閉；其中玩家第24戰已由
2026-08-21 raw ch23 adapter 勘誤為 E1。測試使用的資產唯讀掛載與
`--rm` Docker/Xvfb 命令列在遊戲測試報告中保存。

## 2026-08-11：玩家第23戰 ch22_pre 的 LOADCH 視圖來源已補證（E1）

本輪追加 IDA Pro 9.4／Docker 證據，閉合原先「`loadch` 後游標／視圖來源
尚未證實」的窄問題；歷史候選段落保留，不把舊的未知改寫成畫面完成：

- `0x205da` 在呼叫 `0x1088d` 後，明確將 `[0x53AA9]`／`[0x53AAD]`（鏡頭）、
  `[0x53AB1]`／`[0x53AB5]`（絕對游標）、`[0x53AB9]`／`[0x53ABD]`
  （可見游標）全部寫成零，接著呼叫 `0x11CAC(1)`。`0x135dd` 只讓鏡頭與
  絕對游標同步步進，沒有寫入可見游標；完整指令與雜湊見
  [`fd2_ch22_pre_view_reset_ida.txt`](../data/ida/fd2_ch22_pre_view_reset_ida.txt)。
- 因此第一次 PAN `(14,32)` 後，`0x336e5` 的 `[0x53AB9]+6`／
  `[0x53ABD]+5` 解析為可見相對 tile `(0,5)`；重製端以場景專用
  `NativeMapViewState` 載體保存六個原始全域，不把它冒充成 `battle.State`。
- 正式 `story_ch23` 已接 `assets/cutscenes/bindings/ch22_pre.json`；Docker＋Xvfb
  真實回歸涵蓋 70 筆 LOADCH roster、選定 16 人隊伍、16 次停用、三段 tile-step
  PAN、indexed transition 與進入 `battle_ch23`。`ch23.json` 沒有
  `runtime_append_groups` 的明確契約，故 handoff 會安全重建正式戰場，不宣稱
  handler 部分陣列與戰鬥共享；這是避免未證實語意接入的失敗即關閉邊界。
- 此項是靜態證據到重製 E1 窄切片，不是未修改一般玩家 DOSBox E2、逐像素／逐音訊
  parity，也不解除 `postbattle_ch23_persist` 的戰後城鎮／商店／整備／存檔 gate。

### 2026-08-11：CONTINUE current-runtime command grid E2 勘誤

重新檢查原版資料後，確認工作區內的 `FD2.SAV` 是可解碼的 current-runtime
快照，不是「完全沒有有效存檔」：固定雜湊的快照為 chapter0、12 筆 runtime
records、camera `(1,13)`、absolute cursor `(8,17)`、visible cursor `(7,4)`。
在 Docker DOSBox 以未修改輸入從開場按 Escape、標題 `CONTINUE`，再按 Enter
開啟游標單位的 command grid；兩個 320×200 client crop 與完整輸入時間線見
[`native-continue-current-runtime-e2.json`](../data/ui-traces/native-continue-current-runtime-e2.json)。

這項勘誤只把「沒有任何一般玩家 command grid oracle」修正為「章節0已有
E2 原版錨點」；沒有把章節0快照外推到 ch22／ch23，也沒有解除重製端
CONTINUE 的 pending-group／`Game` controller handoff 閘門。未修改原版與重製端
尚未同一 raw runtime 狀態逐像素配對，故 UI-03 仍為 partial。

### 2026-08-11：未修改原版敵方回合 E2 錨點

沿用同一份固定雜湊 `FD2.SAV`，在未修改原版 DOSBox 中實際執行
`CONTINUE → Return 開啟 command grid → Down 選 END → Return → Return(YES)`。
約 1 秒畫面顯示 `ENEMY PHASE`；約 10 秒仍在敵方回合，約 20 秒回到玩家操作狀態。
這是一般玩家輸入可達的原版 E2 邊界，輸入時間線、來源與 PNG 雜湊保存於
[`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)。

證據只閉合敵方回合的進入／持續／返回畫面，不把畫面變化命名成特定目標、移動
評分或命令效果；重製端尚未以同一 raw runtime 狀態完成逐幀配對，未知 AI 語意仍
失敗即關閉。`CAMPAIGN-POSTBATTLE-E2-FULL-PATH` 也仍需逐章保留戰後城鎮／商店／
整備／存檔節點的正常玩家路徑。

> 歷史快照（2026-08-20）：下段的 chapter0-only／泛用文字層限制已由
> 2026-08-22 共用 owner 與 indexed renderer 取代，只保留形成過程。

2026-08-20 已把同一輸入證據的狹義消費端接到正式重製路徑：只有 chapter0
current-runtime 空游標面板的 Down（內部 direction 3）可開啟 END 確認；四個原始
關閉畫面都完成後才顯示確認提示，Enter／Space 的 YES 才呼叫既有 `endTurn →
beginEnemyPhase → aiStep → finishTurn`，最後返回 `PLAYER PHASE`。Escape／Backspace
只取消且不改回合。其餘三個 `0x16f55` cell 的 owner 仍未知並失敗即關閉；目前確認
提示沿用重製端文字層，未接原版 indexed renderer，因此只提升為 `RUNTIME-E1`，
不宣稱重製端畫面 E2。

2026-08-21 靜態證據勘誤：上述 direction 3 不再只是 E2 輸入旁證。合法 IDA Pro
9.4 與 Docker Capstone 直接證實 `0x16F55` 在 `[0x53C57]==3` 時載入
`FDTXT_000[0x1A3]`，`0x19953` 成功且 choice 0 後載入 `[0x1A4]`，再呼叫
`delay(0xC8)` 與 `0x1A30B`。重製端因此改用兩段原文，並以60 Hz十二幀近似
200 ms 後才發布 `ENEMY PHASE`。同批也固定 command 13–16 的 indexed 演出 owner：
`0x21AD9/0x21B99/0x2211C/0x22153→0x21B18`；當時只為 command 13 加入玩家
cursor→MP／HP→action→range reset 的正式回歸，未猜接其畫格、調色盤或音效。
位址、雜湊、工具版本與限制見
[`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt)。

2026-08-22 共用 owner 勘誤：IDA Pro 9.4 直接指令固定 `sub_117E7` 由 main
`0x25DCE` 呼叫；玩家確認後先在 `0x118B3` 呼叫 `sub_12C0D`，回傳 `-1` 才在
`0x118C1` 進 `sub_16F55`，非負 index 則走單位分支。因此空游標四格系統面板
不是 chapter0 CONTINUE 專屬。正式 battle 的實作契約如下：

1. `sel==nil` 且目前格沒有 runtime unit 時，只有完整原始 FDOTHER #2 四格可供
   呈現才開啟面板；缺資產不建立不可見的互動熱區。
2. 面板沿用 `sub_16F55` 的 `[7,5,6,4]／[0,0,0,0]` raw tables、游標錨點與
   四幀 opening／closing；其餘三格仍不可由圖示猜 owner。
3. direction3／END 沿用已閉合的兩段 FDTXT、取消零修改與十二個60 Hz幀近似；
   只有延遲結束才發布敵方回合。
4. chapter0 CONTINUE 的一次性 opening-confirm（歷史擷取限制）只證明該存檔第一個 Return；
   面板本身改由共用 `nativeSystemCursorOverlay` 狀態承接。這是 `RUNTIME-E1`，
   本段當時尚未提升確認框畫面、DOS tick 或各章一般玩家路徑為 E2；問句畫面的
   後續同狀態比較與仍存限制，以緊接的 indexed renderer 小節為準。

主證據見
[`fd2_117e7_empty_cursor_system_overlay_ida.txt`](../data/ida/fd2_117e7_empty_cursor_system_overlay_ida.txt)。

#### END 確認框 indexed renderer（2026-08-22；RUNTIME-E1）

未修改原版 E2 與 IDA caller-specific 參數已把前述「確認提示沿用重製端文字層」
推翻。`0x16F55` 的 END 分支必須以一個全有或全無的播放器保存：目前320×200
indexed map source、FDOTHER #5 對話框格、DATO #75 frame0、FDTXT_000／字型、
FDOTHER #2 choices與目前 palette。正式執行期已依此契約接通；任何來源缺失時不得
顯示現代文字代用品，也不得關閉命令框或提交回合。

1. `0x1956B(0x4B)` 先從 source 發布六個對話框展開畫面；問句
   FDTXT `0x1A3` 使用 `(99,127)`、foreground `0xCD`、background `0x4C` 與
   19-pixel line step；緊接的`0x16559(0)`再寫一次DATO #75 frame0，使頭像擁有
   所有重疊像素。
2. `0x19953` 再發布四個選項展開畫面；stable choices沿用 cells 48/49與51/52、
   Left／Right選0／1且不環繞、BIOS低字delta>=2才前進pulse。每幀都由Draw
   acknowledgement推進。
3. Enter／Space在choice0接受；choice1或Escape／Backspace取消。兩條路徑都先
   跑`0x197E5`四個選項收合畫面。
4. 接受在`(99,146)`繪FDTXT `0x1A4`，取消同址繪`0x19C`。未修改原版一般玩家
   密集擷取證實`0x15F84`直接寫VGA期間的逐glyph中間內容可見；因此choice四幀
   收合後，每個普通glyph發布一個indexed frame，`FFFE`只換行。完整句發布後才
   等待十二個60 Hz remake frames近似原版`0xC8` ms，再依`0x196CB`發布五個
   dialogue收合畫面並復原source；不得先瞬間畫完整句再空等。
5. 只有接受分支在完整收合與source restore後進既有`endTurn`；取消分支保持
   turn、單位、roster與campaign零修改。DOS tick精確時序與音訊owner仍不外推。
6. question／accepted／canceled 為三份不可變且互不別名的320×200 frame；回覆
   預先合成不得污染問句或dialogue。一般X11正常玩家擷取已把三段文字疊印確認為
   真實重製缺陷，因此測試必須同時驗證像素內容與底層slice不共用。
7. DATO #75 的`0x9017`是`0x4E8E1`每列由右往左寫的最右端，不是左上角；
   80像素寬frame實際佔`0x9017-79..0x9017`並水平反轉。原版正常玩家擷取與
   `0x4E8E1`直接指令共同推翻舊左到右blit，故此修正必須落在共用DATO renderer。

位址、兩條分支原文、原版E2畫面雜湊與重開條件見
[`fd2_end_turn_confirmation_ui_ida.txt`](../data/ida/fd2_end_turn_confirmation_ui_ida.txt)。

同一份合法 `FD2.SAV` 的未修改原版一般玩家擷取與重製正式鍵盤路徑已完成
問句穩定畫面比較：肖像與問句子區域差異為0，整幀仍有1692個差異像素，集中於
YES脈動及戰場單位／游標動畫相位。因此原版一側列`PLAYER-E2`、重製一側仍列
決定性`RUNTIME-E1`，不宣稱全幀逐像素一致。畫面、雜湊、輸入時間線與限制見
[`native-end-turn-confirmation-original-vs-remake-e1.json`](../data/ui-traces/native-end-turn-confirmation-original-vs-remake-e1.json)。

接受與取消回覆另已以相同合法存檔、普通鍵盤事件及正式 campaign 路徑完成動態
配對：原版與重製都可見字數逐步增加，接受在完整句與等待後進敵方回合，取消則
回到玩家戰場。這閉合逐字形發布的內容順序與正常輸入可達性，但未鎖定相同動畫
相位、逐毫秒時序或音訊，故重製仍是 `RUNTIME-E1`。六狀態接觸表、雜湊與限制見
[`native-end-turn-response-progressive-original-vs-remake-e1.json`](../data/ui-traces/native-end-turn-response-progressive-original-vs-remake-e1.json)。

2026-08-21 後續閉合：`0x21EB1` 的 caller-specific 排程現已由 IDA／Capstone
固定為 FDOTHER #3 LUT 9→1九張擴張、200 ms、3→9七張收束、`0x11CAC(0)`
與200 ms尾停；中心來自鏡頭相對可見游標，半徑由四個 wrapper 的
`(1,2)/(2,4)/(8,4)/(6,6)` 決定。重製端以
`BuildNativeCommandHealPresentationSchedule` 配合既有嚴格 `0x22046`
compositor，在玩家 command 13–16 狀態變更前一次預先驗證全部 16 張。缺 baseline、
LUT、palette 或 raw view 時交易不發生；原版每張 5 ms 在 60 Hz 更新迴圈中採每張
至少一幀的順序近似。敵方 mode 11 不套玩家 confirmed-cursor gate，而是依既有
`0x15311` 證據在移動後重建 raw target array；後段數字佇列與 E2 仍失敗即關閉。
證據見
[`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt)。

#### command 13–16 後段演出規格（實作前契約）

本節是 [`fd2_command_numeric_tail_ida.txt`](../data/ida/fd2_command_numeric_tail_ida.txt)
的重製端消費規格；它不新增原版語意名稱。實作必須以一份具型別且不可變的
`NativeCommandHealTailSchedule`（實際 Go 名稱可等價調整）保存以下資料，禁止將
表格與時序常數分散寫入畫面控制器：

- command 13–16 共用 FDOTHER #6 起始 descriptor `0x39`、7 張連續畫格、第一張
  sample index `12`／loop `1`；每張先恢復完整 indexed baseline，再對所有可見的
  final target 合成同一張 descriptor。
- snapshot 切換段使用 sample index `1`／loop `1`、raw compositor 引數 `0xC0`，
  執行5組「原始 snapshot／重畫 target mask」切換，結尾保留原始 snapshot。
  重畫端依 `fdicon.Sprite.Mask`（不是 `Pixels!=0`）把24×24 source write spans填成
  `0xC0`，cycle3改用2，並寫在 target cell上方6列。`0xC0` 只以 raw 值傳遞；
  沒有原版畫面證據前不得命名其顏色或混合模式。
- 數值 writer 固定每個可見 target 四欄、水平 position code `2,7,12,17`、
  descriptor bias `0x69`，並沿用 `AppendNativePresentationDigits` 的右對齊契約。
- 數值 reader 固定22張 raw frame，垂直表為
  `15,15,15,15,7,3,1,0,0,1,3,7,15,15,11,9,8,8,9,11,15,15,15,15,15`；
  第 `q` 欄在第 `f` 張讀 `table[f+(q mod 4)]-3`，glyph 來自 FDOTHER #5。
  destination 保留原式
  `work+0x8088+24*(x-cameraX)+10944*(y-cameraY)+positionCode+
  456*(vertical-3)`，不得用重新估算的文字 baseline 取代。

正式順序固定為：既有 `0x21EB1` 16張前段 → FDOTHER #6七張 → 五組
snapshot→mask切換與結尾 snapshot → MP／HP transaction → 由每個 target 的實際 restore 回傳值
建立數值 queue → 完整 steady redraw → 22張數值演出 → 尾停 → action cleanup／
敵方 AI continuation。數字段 baseline 必須由 transaction後狀態重畫；不可沿用
`0x1C2DA` 的 mask work buffer或 transaction前的 HUD snapshot。
玩家與敵方 mode 11 共用這個視覺工作，但 target array 的 owner 保持分離：玩家只
使用已確認的 cursor target；敵方只使用 `0x15311` 既有 raw selector 重建結果，演出
期間不得重新規劃或改套玩家 gate。

資源與狀態發布採原子預檢：在任何 MP／HP mutation 前，必須一次確認完整 indexed
baseline、palette、FDOTHER #3前段 LUT、FDOTHER #6 descriptors `0x39..0x3F`、
FDOTHER #5 digits `0x69..0x72`、全部可見 target 座標及數值版面都可解碼。任一缺失
即失敗即關閉，不播放半套演出、不扣 MP、不改 HP，也不執行 action cleanup／AI
continuation。預檢後若畫面工作仍發生不可恢復錯誤，必須保存 `loadErr` 並停止後續
continuation；不得把已改變的狀態偽裝成成功完成。演出期間維持既有禁止存檔邊界。

時序只保存可觀察順序，不宣稱逐毫秒一致：原版7張前段的每張1 tick，以及數值段
每張2 ms，在60 Hz更新迴圈中都至少占一個可見 update；原版兩個200 ms停頓及數值
尾端500 ms統一以 `nativeDelayTicks` 換算。若未來 renderer 支援更高頻率，須另以
原版同狀態擷取驗證後才能提高時間忠實度。

實作驗收至少包含：typed schedule 的原始常數回歸；任一 #3／#5／#6 資源缺失時
玩家與敵方都零 mutation；玩家四個 command 與敵方 mode 11 的狀態邊界；AI 演出
期間不重新規劃；每 target 四欄、相位索引與右對齊；22張後回復 baseline並完成
500 ms尾停；存檔仍拒絕演出中狀態。這批只可提升為 `RUNTIME-E1`，未修改原版與
重製端同狀態逐幀／逐音訊比較完成前，`PLAYER-E2` 維持未完成。

### 2026-08-11：mode 2 `aiStep` 遊戲層消費端 E1

重製端新增兩個決定性回歸夾具：完整原始來源證據（raw provenance）的 mode 2 物理計畫
會由 `NextAIPlan` 進入 `aiStep`，實際消費移動路徑、FIGANI 攻擊演出擁有者（owner）
並完成回合；缺少移動成本資料列（movement-cost rows）時則在執行前停止，單位維持未行動。這只證明重製端的資料
契約與遊戲層 owner 已接通，不能替代未修改原版的同狀態 E2，也不為目標選擇、命令／
法術／道具效果或其他 AI mode 建立語意。一般玩家驗收仍要求固定 raw roster、鏡頭、
游標與 tick 的原版／重製逐幀比較，並保留戰後城鎮／商店／整備／存檔節點。

本輪也以該截圖快進器產生 [`native-battle-ch01-remake-e1.png`](../figures/native-battle-ch01-remake-e1.png)。
這張圖只作重製端 E1 執行期展示，完整環境、原版資產唯讀掛載與雜湊見
[`native-battle-ch01-remake-e1.json`](../data/ui-traces/native-battle-ch01-remake-e1.json)；
它不取代未修改原版一般玩家 E2 或同狀態逐幀比較。

## 2026-08-11：CONTINUE 戰場交接發布契約已補齊（E1）

本輪把已經過型別化驗證的 CONTINUE 載入結果接到重製端戰場控制器的
「發布邊界」，但沒有把標題流程尚未提供的值猜成完成：

- `MaterializeNativeContinueInteractiveBoundary` 只在欄位控制（field control）、
  執行期單位（runtime units）、待處理群組（pending groups）、地圖計時（map timing）、
  視圖（view）、HUD 與開場選擇器（selector）都成功後，將已證實的範圍模式
  （range mode）`0` 原子切換成返回控制器使用的 mode `1`。它不重播
  `Scenario.Setup`、不重建開場單位，也不修改未驗證的原始記錄（raw record）。
- `ValidateNativeContinueBattleHandoff` 檢查章節／地圖（chapter/map）、地圖尺寸、
  scenario 的 `runtime_append_groups`、保存的鏡頭／游標／可見游標
  （camera/cursor/visible cursor）、回合（round）、HUD 閘門（gates）、選擇器快取
  （selector cache）、隊伍名冊（roster）與待處理群組；任一欄缺失即失敗即關閉。
- `Game.publishNativeContinueBattle` 只接受上述通過的狀態，複製 campaign runner
  後一次發布 `Game.st`／`Game.sc`／current node，清除殘留對話、轉場、戰鬥事件與
  暫存地圖緩衝，再同步保存的鏡頭／游標。它刻意不呼叫 `resetBattle`，避免把
  CONTINUE 快照誤當成新戰鬥開場。

實際回歸 `TestNativeContinueBattlePublicationFromRealCurrentSnapshot` 讀取使用者
提供的 `FD2.SAV`（MD5 `409795ccebc2af340d5c74152c2d471c`，SHA-256
`6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f`），在 Docker
內解碼 chapter0、12 筆 current-runtime 執行期記錄，完成 map0/ch01 的型別化轉接器
（typed adapter）與 `Game` 發布；`TestMaterializeNativeContinueInteractiveBoundaryInstallsControllerMode`
及 `TestValidateNativeContinueBattleHandoffRejectsIncompleteAdapters` 也通過。
測試使用呼叫端明確提供的 `TitleTimerTick=0` 作為資料契約夾具，**不**宣稱標題
BIOS 時鐘、`0x10494/0x105ED` 重繪／延遲（redraw/delay）、逐像素或一般玩家 E2 已完成。

同日已補上正式標題呼叫端：`TitleMenuContinue` 只接受明確提供的
`FD2_NATIVE_SAVE` 與 `FD2_NATIVE_TITLE_TICK`，從可編輯戰役圖唯一解析
`scenario.chapter` 相符的 battle node，在私有 state 完成四個 adapter 後才發布；缺少
存檔、signed BIOS tick、資產或章節對映含糊時，標題保持不動並失敗即關閉。Docker／Xvfb
以 Escape×8、Down×2、Enter 走過標題續戰，產生重製端 E1 畫面
[`native-continue-current-runtime-remake-e1.png`](../figures/native-continue-current-runtime-remake-e1.png)，
條件與雜湊見
[`native-continue-current-runtime-remake-e1.json`](../data/ui-traces/native-continue-current-runtime-remake-e1.json)。

目前仍保持失敗即關閉的邊界：`FD2_NATIVE_TITLE_TICK` 是明確輸入的 signed BIOS tick，
尚未由跨平台執行期自行重建原版 `0x25D83..0x25D8B` 的時鐘／重繪時序；多章節動態待處理
群組寫入器／公式、未修改一般玩家同一 raw runtime 的重製／原版 E2、action 選取擁有者與
status/equipment panel、戰後城鎮／商店／整備／存檔全路徑，以及 mode 11、`0x13FD4`、
mode 5 的完整目標／指令／法術／道具人工智慧語意仍未解除。

## 2026-08-11：戰役 intermission 與 current-runtime 畫面配對更新

本輪完成一個可審查的戰役圖垂直切片：`battle_ch16` 勝利先進入有正式
`handler_binding` 的 `postbattle_ch16_persist`，再到 `town_ch17`；回歸實際走過武器店
返回、整備取消、再次整備確認，最後才進 `battle_ch17`。測試為
`TestCampaignFullChapter16BattlePreservesTownShopPreparationBoundary`，因此不能把
戰後節點折疊成直接下一戰；這仍不是 30 章一般玩家 E2。

同一份固定雜湊 `FD2.SAV` 的原版 CONTINUE E2 與重製端普通 X11 輸入畫面已形成配對基準，
但重製端仍使用明確的 `FD2_NATIVE_TITLE_TICK=0` 夾具，故重製端側維持 E1。原版 320×200
放大至 640×400 後，畫面 AE `164`、RMSE `50.2631`；證據與限制見
[`native-continue-current-runtime-remake-e2.json`](../data/ui-traces/native-continue-current-runtime-remake-e2.json)。

`drawRing` 另已補上 native 320×200→640×400 的呈現座標縮放，並以
`TestNativeActionOverlayAnchorUsesPresentationScale` 固定驗證；這只修正指令環位置，
不代表圖示可用性、指令／法術／道具效果或敵方人工智慧語意已完成。原版敵方回合 E2
錨點仍以 [`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)
為準，重製端同狀態敵方回合 E2 尚未解除。

## 2026-08-11：可玩近似模式與忠實證據模式分流

為配合重製目標「可玩且近似」，不再把逐像素／逐音訊 E2 當成唯一完成條件；但未知
原版語意仍不得悄悄成為正式規則。重製端新增明確的 `FD2_APPROXIMATE=1`：

- 尚未有正式 handler 的 `postbattle_*` 節點只同步已物化戰場隊伍，顯示戰後整理
  提示，玩家確認後沿編輯腳本的 `next` 進入既有城鎮或整備節點；不自行建立
  JOIN、獎勵、章節值或原版分支。
- 未設定旗標時維持忠實證據模式的失敗即關閉；`TestApproximatePostbattlePreservesAuthoredIntermissionBoundary`
  以 town／preparation 兩條邊界驗證同步與戰場狀態清除。
- `TestApproximateCampaignFullUnboundPostbattleBoundaries` 與
  `TestCampaignFullUnboundPostbattleDefaultsFailClosed` 再以 `campaign_full.json` 的
  第 23、24、29 戰實際節點驗證近似確認與預設停止。第25戰已由下游正式
  `ch24_post` E1 回歸取代此近似測試。
- `TestApproximateCampaignFullResultConfirmationKeepsUnboundIntermissions` 再從三個
  對應的 `battle_ch23/24/29` 設定勝利結果，走正式 `confirmBattleResult`，確認
  先停在近似戰後提示，玩家確認後才沿 authored `next` 進 preparation／town；這補上
  之前直接把 Runner 游標放在 postbattle 的測試與正式結果消費端之間的邊界。
- 近似模式只代表可玩的戰役銜接，不提升 E1 為 E2，也不改寫既有原版證據與推論
  等級；剩餘節點與證據以 [`91-worklist.md`](91-worklist.md) 最新稽核為準。

## 2026-08-11：mode 5 `aiStep` 遊戲層消費端 E1

重製端新增 `TestAIStepConsumesVerifiedMode5EventPlan` 與
`TestAIStepStopsMode5WithoutMovementProvenance`，把已驗證 raw event grid、
field-control row 與 movement-cost rows 的 mode 5 計畫接到 `aiStep`：單位
實際走到事件格後，原始 event state 由 `0` 寫成 `1`、地圖事件位元組清除、raw
`+0x34` 寫成 `7`，並完成該回合。缺少任一必要來源時，執行期仍停止且不標記行動。

測試以 `FD2_MUTE=1` 隔離容器沒有可播放 AIL sample 的環境差異；這不是跳過 mode 5
狀態提交，也不替事件命名成物品／法術／特殊效果。此切片只提升重製端遊戲層 owner
為 E1，尚未證明原版一般玩家目標選擇、完整敵方回合或同 raw 狀態 E2。

## 2026-08-11：mode 7 `aiStep` 遊戲層消費端 E1

重製端新增 `TestAIStepConsumesVerifiedMode7DestinationPlan` 與
`TestAIStepStopsMode7WithoutMovementProvenance`。測試只採用反組譯明示的 raw
`+0x35/+0x36` 目的地：`NextAIPlan` 產生 movement-only 路徑，`aiStep` 抵達相同落點
後才寫入 raw `+0x05=1`，並保留 `[0x51A83]` map-range 的來源標記；缺少 movement
rows 時在任何行走、攻擊、byte 寫入或回合變更前停止。

這是 `0x32975` raw writer 的重製端 E1 消費證據，不替 mode 7 命名為特定玩法，也不
證明原版目標選擇、完整敵方回合或一般玩家 E2。

## 2026-08-11：mode 3／9 `aiStep` 遊戲層消費端 E1

重製端新增 `TestAIStepConsumesVerifiedMode3AndMode9RawTargetPlans` 與
`TestAIStepStopsMode3AndMode9WithoutMovementProvenance`。兩個測試只採用 raw
`+0x08` 首筆查找與 movement rows：`NextAIPlan` 產生 movement-only 路徑並抵達
raw 目標鄰格；mode 3 提交已證實的 map-range write，mode 9 保留不寫入的分支，兩者
都不把 raw target 轉成攻擊。缺少 movement provenance 時在位置、回合與 map-range
變更前失敗即關閉。

這是 `0x12C60` lookup 的重製端 E1 消費證據，不命名目標選擇或 mode 3／9 的高階
玩法，也不證明原版一般玩家 E2。

## 2026-08-11：mode 4／10 `aiStep` 遊戲層消費端 E1

重製端新增 `TestAIStepConsumesVerifiedMode4AndMode10DestinationPlans` 與
`TestAIStepStopsMode4AndMode10WithoutMovementProvenance`。兩個測試只採用 raw
`+0x35/+0x36` 目的地，完成 movement-only 行走後提交 map-range write；不寫入
raw `+0x05`，也不建立攻擊。缺少 movement provenance 時在位置、回合或 map-range
變更前失敗即關閉；mode 4／10 的高階玩法仍未知。

## 2026-08-11：mode 0／8 `aiStep` 遊戲層消費端 E1

重製端新增 `TestAIStepConsumesVerifiedMode0NearestFallback`、
`TestAIStepStopsMode0WithoutMovementProvenance` 與
`TestAIStepConsumesVerifiedMode8Completion`。mode 0 只依 raw nearest fallback 建立
movement-only 路徑並完成回合，不寫入 map-range；mode 8 只驗證共同的 raw 行動完成
分支。mode 0 缺少 movement provenance 時在位置、回合或 raw 狀態變更前停止。

這些是重製端 E1 消費邊界，不替 mode 0／8 命名高階玩法；mode 1 的 blocked-coordinate
遊戲層消費端另見下節，原版一般玩家 E2 也未宣稱完成。

## 2026-08-11：mode 1 blocked-coordinate 遊戲層消費端 E1

`TestAIStepConsumesVerifiedMode1BlockedCoordinate` 與
`TestAIStepStopsMode1WithoutMovementProvenance` 已在 Docker／Xvfb 通過。測試沿用
`0x14121` 的 raw blocked-cell 來源、完整 runtime record、selector、地形／組成與
movement-cost rows：`NextAIPlan` 只接受唯一 raw 落點，`aiStep` 實際完成
movement-only 行走並結束回合，不建立攻擊、不寫入 mode 0 的 nearest fallback 或
map-range 狀態。缺少 movement／raw provenance 時，在位置、回合與 raw 狀態變更前
失敗即關閉。

這只證明 `0x14EF0→0x14121` 失敗備援的重製端 E1 消費邊界；blocked-coordinate 的
原版完整 producer／高階 mode 1 語意、一般玩家同狀態 E2 與其他 command／spell／item
路徑仍未閉合。

## 2026-08-11：`0x14EF0` command route 遊戲層消費端 E1

`TestAIStepConsumesVerified14EF0CommandRoute` 已在 Docker／Xvfb 通過。測試以完整
raw command book、command mask、selector、runtime record、地形／組成、movement-cost
與 class resistance provenance 建立三格戰場；`NextAIPlan` 由 `0x14EF0` 選出
`0x15311` command route，`aiStep` 先沿 raw destination 移動，再呼叫已驗證的
command 0 數值執行器，提交 MP 扣除、目標 HP 變更與回合完成。測試只驗證 raw route
與 numeric state owner，不替 command 0 命名法術或演出，也不把 synthetic fixture
當作原版一般玩家 E2。

`0x15055` 未知 item／relocation、未知 command／spell 的完整效果與 indexed 演出仍
維持失敗即關閉；缺少 command book、target 或 resistance provenance 不得退回正規化
AI。已核對的 type-5 item row 窄交易另見下節。

## 2026-08-11：`0x14EF0` type-5 item route 遊戲層消費端 E1

`TestAIStepConsumesVerified14EF0ItemRoute` 已在 Docker／Xvfb 通過。正向 fixture 使用
資產表中 item 192 的原始 23-byte row（row `+0x0d=5`、`+0x0e=40`、選擇／目標欄位
保持原值），並提供完整 command book、selector、runtime record、地形／組成、
movement-cost 與 `NativeTileBlitModes` provenance。`NextAIPlan` 實際選出
`0x14EF0→0x15055`，`aiStep` 完成目的地移動後交給正式 item owner：
`beginNativeTargetItem` 建立 raw 目標，`applyNativeTargetItem` 依 `0x211A4` type-5
交易回復目標 HP、消耗來源欄位並提交回合。`TestAIStepStops14EF0ItemRouteWithoutItemRows`
確認缺少 item rows 時在 HP、背包、行動旗標與回合變更前失敗即關閉。

這只閉合已核對 type-5／`0x211A4` 的一個重製端 E1 consumer；不替 item 192 命名玩法，
也不宣稱 type-5 以外的 item、`0x15055` relocation、未知 command／spell 演出或原版
一般玩家 E2。其餘物品與完整敵方回合仍依工作清單維持未完成。

## 2026-08-11：可編輯敵方／友軍 NPC 法術後備（fallback）E1

當 `NextAIPlan` 的原始 AI 路徑（route）未建立計畫且未回傳錯誤時，重製端才會從可編輯
`SpellBook` 與單位 `Spells` 建立法術計畫。已支援的治療、解除、再行動、輔助、
攻擊與狀態類法術，依可重現的目標分數、施放距離與移動路徑排序；未知、傳送或未有
明確數值語意的法術不會被猜測性接入。計畫由正式
`NextAIPlan → aiStep → CastArea` 路徑消費，完成 MP、效果、死亡獎勵與回合提交。

`TestNextAIPlanSelectsEditableHealSpellBeforePhysicalFallback`、
`TestNextAIPlanApproachesEditableAttackSpellTarget`、
`TestNextAIPlanAllyTargetsEnemyWithEditableAttackSpell` 與
`TestNextAIPlanUsesEditableInventorySpellMapping`、
`TestNextAIPlanSpellPathUsesStableCellTieBreak`、
`TestAIStepConsumesEditableHealSpellThroughProductionLoop` 與
`TestAIStepConsumesEditableAttackSpellAndMovesIntoRange` 鎖定這個正規化
（normalized）切片；`TestAIStepStopsSpellWithoutRNGBeforeMutation` 則確認缺少
決定性亂數來源時，在 MP、位置、HP 與回合變更前停止。原始 AI 路徑一旦回報
`NativeError`，仍不得回退至本後備路徑。

這是讓資料驅動戰役可實際運作的重製端 E1 行為，不是 `0x1598A` 原始評分、命令格、
特效、音效或一般玩家 E2 的證據；原始 AI 法術路徑與同狀態原版比較仍依工作清單
保持未完成。

## 2026-08-11：敵方回合多單位 loop 消費端 E1

`TestAIStepConsumesTwoVerifiedMode7ActorsBeforeFinishingTurn` 建立兩名都具完整 raw
mode-7 `+0x35/+0x36` 目的地、地形／組成與 movement-cost provenance 的敵方單位。正式
`NextAIPlan`／`aiStep` 先消費第一名 actor，提交其 `+0x05=1` 後才建立第二名 actor
的行走；第二名完成後只由共同 `finishTurn` 增加一次回合。測試同時確認第二名不會在
第一名尚未提交時提前消費，且沒有 attack／normalized fallback 介入。

`TestAIStepStopsTwoMode7ActorsWithoutMovementProvenance` 刪除 movement-cost rows，
第一名在任何行走／raw 寫入前失敗即關閉，第二名、兩者位置、行動旗標與回合均保持
不變。這只關閉重製端「多單位敵方回合 loop」的 E1 編排，不把 mode 7 raw byte 命名
成原版玩法，也不替原版多單位目標選擇、權重或同狀態 E2 增加語意。

## 2026-08-12：終局 party montage 與 raw 輸入變化的近似接線（E1）

固定雜湊 `FD2.EXE` 的合法 IDA Pro 9.4 資料庫，再由 Docker Capstone 5.0.3
交叉確認下列原始位址；完整位元組、工具版本與推論等級見
[`fd2_ch29_montage_ida.txt`](../data/ida/fd2_ch29_montage_ida.txt) 與
[`fd2_ch29_input_cleanup_ida.txt`](../data/ida/fd2_ch29_input_cleanup_ida.txt)。

- `0x2918f` 是 `test unit[+6],unit[+6]`，因此 `0x29164` 分支是零／非零，
  不是 `0/1` 枚舉。`MontageCycle` 現接受所有 nonzero raw value；真實 persistent
  party constructor 的 `+6=2` 不再被誤拒。
- portrait loop 每輪在 `0x2c946` 呼叫 `0x17aa9(1)`；該 helper 已證實比較
  `word[0x46c]` 的 wrap-aware 差值。55ms 只是 BIOS 約18.2Hz的**強推論近似**，
  不能寫成 E2 的精確時鐘。
- `0x2c950→0x10620` 的 raw word 差異若為非零，會在 `0x2c959` 把 outer counter
  寫成1、於 `0x2c961` 複製 raw word，且仍先走完當前 portrait；下一 outer loop
  改為 `j=0`。因此近似 runtime 只在 portrait loop 把「新輸入」作為這個 raw-change
  載體，完成當前角色後跳到 final loop。它沒有把任一鍵命名為原版 Enter、Space 或
  Escape。
- 最終 `ending` 節點仍受 `FD2_APPROXIMATE=1` 保護。抵達 `0x2c548` 時，runtime
  只從 persistent JOIN chronology 經現有 LOADCH 的 deployed-order 投影取 raw
  `+6/+7/+8/+0x20`，並以原始 `FDOTHER/TAI/FIGANI/DATO/FDTXT` 建立 montage。
  缺任何 raw provenance 或素材時，不建立半張畫面，保留明確的可編輯結語回退；
  direct preview 和預設忠實模式不跨越此 gate。
- 第 29 戰 `ch29_post` 的 `0x25970→0x2bce5` 仍是一個未編譯的原始 owner；
  截圖隊伍輔助程式會拒絕其不完整的 LOADCH，而不把它偷渡成 montage 的隊伍來源。
  因此上述持續隊伍（persistent roster）是近似終局節點既有的型別化資料載體，
  不是原版第 29 戰一般玩家交接（handoff）的宣稱。

回歸 `TestMontageCycleExecutesBothNativeSideBranchesAndFinalPaletteFade`、
`TestMontageCycleInputChangeFinishesCurrentPortraitThenJumpsToFinalLoop`、
`TestNativeEndingMontageRecordsUseOnlyPersistentRawProvenance` 與
`TestApproximateCampaignMontageStartsFromPersistentLoadCHOrder` 以玩家原始 archive
驗證 raw nonzero、輪播跳轉、資料拒絕與戰役 admission。這是重製端 E1 垂直切片，
不是未修改一般玩家終局路徑；`0x2c194` 尾段、精確 BIOS key code、FDMUS_018／停曲
owner、raw terminal owner、戰後／城鎮 handoff 與一般玩家 E2 仍失敗即關閉。

## 2026-08-12：終局 #59 定格與可選隊伍回顧（近似 E1 勘誤）

本節是對上述「`0x2c194` tail 完全沒有 consumer」舊現況的追加勘誤，不把未還原
的 20-entry 演出升格為完整終局。固定雜湊的 IDA／Capstone 直接指令與資源形狀現在
分開記錄：`0x2c1be` 的 FDOTHER #60 與 `0x2c357` 的 #59 都是 320×200 單影格；
`0x2c220` 的 #58 是傳給 `0x2935b` 的 20-entry frame table；`0x2c234` 的 #57 是
768-byte（256×3）VGA 調色盤，先寫入 `[0x53a65]` 再交給 `0x11d40`。先前把
`0x2935b` 的來源寫成 #57 是 resource index 錯置，已在
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)
追加勘誤。

`MontageTailAssets` 在 `FD2_APPROXIMATE=1`、`0x2c548` 的已 admission party montage
成功完成後，先驗證 #57/#58/#60/#59 的形狀與透明 RLE 邊界；`MontageTailPlayer`
再以 20 組 TAI／BG／FIGANI、descriptor `+6` 延遲與 #58 疊圖產生近似尾段，最後
呈現 #59 並保持終局定格。這條成功路徑不會再落入 generic ending；任一來源缺失時
在播放前失敗即關閉並保留可編輯結語回退。原始程式與近似 runtime 都在 20-entry
loop 前啟動 `FDMUS_018`，但其完整停曲、呼叫間隔與畫面同步尚未閉合，故只稱曲目與
大致階段相符，**不宣稱精確時序相同**。

外部第 30 戰片尾錄影顯示黑底金色 `THE END` 長時間停留；把 #59 對應成該畫面的
結論是**強推論／外部視覺旁證**，不是原版未修改一般玩家 E2，也不是逐像素比較。
擷取方法、時間窗、影片 URL、限制與原始位址交叉關係都記於
[`ch30-ending-youtube-visual-side-evidence.json`](../data/ui-traces/ch30-ending-youtube-visual-side-evidence.json)；
外部影片或影格沒有加入儲存庫。

為符合「終局停住讓玩家回味」的產品規則，定格是預設。Enter／空白鍵才會啟動一個
**重製延伸**：重播已 admission 的每位隊員 `MontageCycle` 最終狀態，完整一輪後再開始；
Enter／空白鍵／Esc 立即恢復 #59 定格。這些控制鍵不映射回 `0x10620` 的 BIOS word，
也不主張原版 terminal self-loop 有相同行為。

仍未解除的 gate 是 `0x28a6c(0,1)` 的精確 20-entry renderer、#60 的完整可見 owner、
palette fade／wait 的精確時序、raw `0x25970→0x2bce5` 一般玩家 campaign owner，
以及完整終局 E2。所有未解項繼續失敗即關閉，不能因 #59 定格或重製回顧功能而接成
正式戰役完成宣稱。

Docker／Xvfb 回歸 `TestMontageTailAssetsPreservePaletteFramesAndTerminalImage` 與
`TestApproximateCampaignMontageStartsFromPersistentLoadCHOrder` 的
`optional_party_outcome_review_loops_and_restores_terminal` 子案例，已以玩家原始 archive
確認資源形狀、終局定格、回顧重新起始與返回定格。新增的
`TestMontageTailPlayerUsesAllTwentySourceTransactionsThenHoldsFinal` 另以玩家原始
archive 驗證 20 組近似資源交易、延遲與最終 #59；不過測試以已完成的 montage state
驗證循環邊界，不能代替未修改一般玩家從第 30 戰走到終局的 E2。

## 2026-08-12：第 29／30 戰持續隊伍到結局的近似資料邊界（E1）

戰役圖原先已有 `battle_ch29 → postbattle_ch29_persist → preparation_ch30`，但只驗證
近似戰後同步與整備入口，沒有驗證該整備節點可安全存檔／讀檔。現在新增完整節點邊界
回歸：玩家勝利結果確認先停在未綁定戰後提示，確認後才清除戰場、進入
`preparation_ch30`，並以隔離的 JSON 存檔往返持續隊伍、加入順序與章節值。這仍是
重製端 E1；raw `ch28_post` 未閉合的 renderer 與原版一般玩家 E2 沒有因此升級。

另修正一個玩家可見的重製資料遺失：`battle_ch30.on_win` 原本直接進 `ending`，
`confirmBattleResult` 只移動戰役游標，而結局角色輪播只讀 `partyRoster`，不讀即將被
清除的最後戰場 `g.st`。因此第 30 戰獲得的等級、經驗與最終數值會停留在結局之外。
戰役資料現在可在「戰鬥勝利直接進 ending」的窄合約上設定
`approximate_sync_party_on_win`；只有 `FD2_APPROXIMATE=1` 才在移動游標前執行既有
`syncPartyFromBattle`，同步失敗或零筆持續隊伍身分（persistent identity）符合時，
保留原節點與勝利結果並停止。載入器拒絕把此欄位放在
非戰鬥、非勝利邊或非 ending 目標；忠實模式完全不消費它。

這個欄位只表達重製近似模式的終局資料保存需求，不命名 `0x25757`、`0x2bce5` 或
任何尚未閉合的原版 terminal handler 語意。原版 owner、精確終局輸入與一般玩家 E2
仍維持失敗即關閉。

**2026-08-21 勘誤：**上述是 2026-08-12 的過渡合約。現行欄位已改為
`ending_party_snapshot_on_win`，並由正式 `battle_ch30→ending` 在移動游標前
原子同步最後隊伍，不再以 `FD2_APPROXIMATE=1` 作為啟用條件。缺戰場或
零筆持續身分符合時仍保持勝利結果與當前節點。此變更只閉合
重製戰役的終局資料邊界，不提升原版 owner、精確 renderer／輸入與 E2。

## 2026-08-13：玩家第21戰天空之鑰固定演出（E0／E1）

本節是對較早「`0x24336` 尚未 lower」與 handler 匯出仍有一筆真正未知的勘誤；
舊時間序列紀錄保留其發現過程，不再作為現況判定。固定雜湊 `FD2.EXE` 由合法
IDA Pro 9.4 Docker 資料庫先確認函式邊界、唯一 caller 與直接呼叫圖，再由 Docker
Capstone 覆核原始指令。完整位址、工具、雜湊、資源形狀與推論等級見
[`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt)。

- `ch20_post` 在 `0x242C9` 無參數呼叫 `0x24336`；該函式先以
  `0x135DD(14,8)` 同步移動鏡頭與絕對游標、保持可見游標相對座標不變，保存
  320×200 VGA 基底，再載入 `FDOTHER.DAT #34`。
- 第0至68幀之後呼叫 `0x20421(0,15,0)` 播放 `ANI.DAT #0` 的全螢幕 AFM；
  `0x20421` 不是音訊 owner。接著以 `0x11DF2` 進行兩次全域調色盤變換，最後播放
  第69至100幀。前段每幀另呼叫 `0x4DFCC`，其已證實窄角色是依程序內相位循環
  更新 DAC `0xE0..0xEF`，不是泛用的整張調色盤 API。
- `native_ch20_sky_key_sequence` 只接受來源 `0x242C9`、目標 `0x24336`、零參數的
  具型別合約。正式執行期在改動畫面前預先驗證全部101幀、ANI 96幀與調色盤；
  缺資產、幀或來源不符時停止，不發布半套演出。
- map20 控制資料另固定 `16` 名部署隊伍加初始群組0的75筆前沿；第2、4、6、8回合
  依序追加群組1至4後為79、83、87、91，群組255不物化。`ch21.json` 因而使用
  `runtime_append_groups`，不再錯把80筆控制列全部當成戰鬥初始狀態。
- 完整回歸由 `campaign_full.json` 的玩家第21戰正式勝利確認開始，走過十句戰後對話、
  六素材原版怪癖配方、鑄造演出、JOIN24／23、持續隊伍同步、`town_ch22` 與
  存檔／讀檔邊界。這關閉重製端 E1 垂直切片，不等於一般玩家 E2。

尚未解除的是未修改原版同狀態逐幀／音訊配對、`0x4DFCC` 進入本函式前的程序內
第一相位，以及相鄰 `layout_units`、ACT63／64 的可見演出。重製端只保存已證實的
相對相位與順序，不宣稱 DOS 計時逐拍或逐像素一致；這些缺口不得觸發重新反組譯
`0x24336` 本體。

## 2026-08-21：`0x22253` indexed presenter 與 ch28 post caller 合約（E0 規格）

本節先固定執行介面，再允許修改 runtime。它沿用已閉合的共用 callee，不重解
`0x22253`：五參數 ABI 是 `(unit,newX,newY,visualX,visualY)`；可觀察排程為
11 張 FDOTHER #6 LMI intro、6 張 FDOTHER #3 contract、18／24 列 direct-VGA
bridge、10 張 FDOTHER #3 release。座標只能在第六張 contract 完成之後、bridge
開始之前寫入 runtime record `+0/+1`。

### 具型別拍與來源限制

正式 `Beat` 使用新的 `native_unit_present` payload，完整保存五參數，不沿用已被
反證的六影格 `HandlerUnitPresent` schema。可編輯數值仍須綁定 caller：

- `0x33F78` wrapper 只接受已證實的 `(slot,x,y)` immediate，先套用其 focus，
  再 lower 為 `(slot,x,y,x,y)`；
- raw ch28 post 的 `0x25535` 只接受原始壓棧
  `10,15,10,15,[0x53BEB]-1`，lower 為「執行當下最後一個 materialized slot」
  與 `(newX,newY,visualX,visualY)=(15,10,15,10)`；
- 其他 direct caller 必須各自補來源專屬 lowering，不得把動態最後槽位或兩組
  座標泛化成任意腳本功能。

### 原子預先驗證與發布

runtime 在改動 live state 前，必須以 private copies 完成以下預先驗證：

1. 固定版 FDOTHER #6 entries `0x72..0x7C`、FDOTHER #3 LUT0..9 與 bridge LUT
   `LUT0[1:256]+LUT1[0]` 全部存在且形狀正確；
2. native work/VGA、地形、unit、foreground、selector cache、camera、visible cursor、
   raw `+0/+1/+3/+4` presentation provenance 與目標 slot 完整；
3. 先用 mutation 前 state 建 terrain-only snapshot、11 張 intro、共享
   terrain+final-LMI snapshot 及6張 contract；
4. 再 clone battle state，只在 clone 寫入 `newX/newY`，用 mutation 後 state 建
   bridge 逐列 VGA 快照與10張 release；
5. 任一 frame、row、bounds、asset 或 compositor 失敗時，不發布第一張影格，
   live unit、work、VGA、DAC、camera 與 beat index 必須完全不變。

全部預先驗證成功後才建立 presenter job。每張 full-present 與每一列 bridge 都是
獨立 draw-ack 邊界；更新迴圈不可在該影格尚未實際畫出前自行跳過。第六張
contract 已畫出後，job 原子寫入 live unit 的 native map coordinates，再發布第一列
bridge；完成 release 後才執行 continuation。若 job 啟動後遇到 renderer 錯誤，必須
回復啟動前的 unit、work 與 VGA。

DOS BIOS tick 只可映射為重製端有界呈現等待；這是 E1 時序近似，不得宣稱 DOS
時鐘逐拍一致。`0x35E5A` 的 0..63／hold／62..0 DAC pulse 是另一個 typed presenter，
不得用 RGBA fade、純 delay 或本 adapter 代替。

### 戰役接線 gate

完成 generic presenter 不自動解除 `postbattle_ch29_persist`。正式 binding 還必須
證實玩家第29戰 post 入口的 materialized slot topology、group9 append 後
`[0x53BEB]-1` 的 consumer、所有對話／pan／palette payload、持續隊伍同步、
`preparation_ch30` 存讀檔，以及一般玩家戰果確認路徑。缺任一項時保持失敗即關閉；
不得用目前的近似戰後 fallback 或 direct-entry 測試冒充忠實模式 E1／E2。

### 2026-08-21 實作狀態

`native_unit_present` 現只接受來源 `0x25535` 的動態最後槽位 ABI，預先產生
11＋6＋18/24＋10個 work/VGA 快照；第17張 full-present 畫出後才提交座標。
決定性回歸覆蓋51-frame分支、mutation boundary、缺 FDOTHER #6 零修改與執行期
rollback。`native_palette_pulse` 另以 immutable baseline 執行127次 DAC 寫入，
保留第64步400 ms hold與最後 baseline restore。兩者是窄 `RUNTIME-E1`，不解除本節
戰役 topology／binding／E2 gate；`0x33F78` story/focus wrapper 也仍維持失敗即關閉。

## 2026-08-21 玩家第29戰 map28 runtime 拓撲契約

正式證據見
[`fd2_map28_runtime_topology_ida.txt`](../data/ida/fd2_map28_runtime_topology_ida.txt)。
固定版 `FD2.EXE` 的 `0x1088D→0x10B4E(0)` 與 raw ch28 pre-handler
`0x33DBA→0x35822(group8,9,19)` 共同固定一般戰鬥入口：20 筆持續隊伍先建立，
再追加 group8 的 56 筆，故 handler 到 battle 的基準 frontier 是 76；它不是把
FDFIELD 的 76 筆來源列全部正規化成 runtime。scenario 必須採
`runtime_append_groups`，正式 handoff 要保留既有 20＋group8 陣列，且不重播
`on_battle_start`。

map28 的 event75 只在已證實的 identity-9 branch 啟動 live row：它令 event74
從 group4 開始，依序逐回追加 groups4、5、6、7；event76 的 progress 到 4 時才
追加三筆 group1，並把其 base 留給 event79。event79 只消費該三筆，不追加群組。
raw ch28 post 最後由 `0x25505` 追加一筆 group9，再以 `[runtime_count]-1` 呈現。
groups2／3 雖存在於 immutable source roster，目前沒有 normal-chain producer；
event82 也沒有已證實的 live-row producer，因此兩者不得物化或用猜測補洞。

資料與 runtime 必須以原子方式保存上述 live-row mutation、context bytes 與群組
追加順序；任何缺 typed row、context record、來源群組或唯一 consumer 的情況均
失敗即關閉。完整 raw field view 只有在具 CONTINUE provenance 時才要求並雙寫，
不得因新戰鬥沒有該 raw view 就捏造來源。

### 2026-08-21 event75→74 實作狀態

- `DATA-READY`：`ch29.json` 已保存 event75 的 selector1、record `+6` gate、
  record `+8 == 9`、FDTXT_029 indices 0／1、event-state writes，以及 event74／76
  live-row activations；event74 另保存 state index16 的 dynamic group 4..7 與
  `0x35822(group,10,29)` 演出參數。
- `RUNTIME-E1`：成功動作共同提交點會啟動 event75；五句 index1 對話完成後才
  原子提交 state 與 live rows。event74 每次先私下 materialize 一個 group、建立
  鏡頭／白閃快照，發布後才將 state 加一及（group 4..6）重排下一回合；group7
  不再重排。缺 raw `+6/+8`、dormant row identity、來源 group 或 compositor 時
  維持零修改。
- 新戰鬥使用雜湊綁定的 typed turn rows；若 CONTINUE 邊界另提供完整 raw field
  view，則提交時必須先核對並同步該 view。兩者不可互相冒充 provenance。

### 2026-08-21 event76 實作狀態

- `DATA-READY`／`RUNTIME-E1`：event76 現由 raw-camp2 phase owner 在 round
  increment 後、玩家輸入前 dispatch。state17<4 的 repeat branch 原子設定slot1
  raw `+5` bit7、state increment與row1下一回合；state17==4則在index2對話後
  私下建構group1三筆、寫state21 base並啟動event79 row。
- final呈現沿用已閉合的 `0x35E5A` indexed presenter，順序固定為六次pulse、
  前兩次後各400ms，以及indices3、4、5、6對話；全部結束才回到player phase。
  測試將核心state preflight與indexed admission分開，不能用無原版資產的直接
  fixture冒充正式畫面已驗收。

### 2026-08-21 event79 實作狀態

event79 已達 `DATA-READY`／`RUNTIME-E1`：raw-camp0 owner使用同一個process-wide
`nativeRNGState`，只前進一次後選state21 base下兩個循環相鄰group1 slots，原子
設定raw `+5` bit7並把row2排到下一round。`0xFFFF` seed回歸鎖住第二目標不得先
發生16-bit overflow；缺第三筆provenance時RNG、row與units均保持不變。

本段撰寫時，正式 ch28 post binding、隊伍同步與 `preparation_ch30` 存讀檔尚未
完成，當時 `postbattle_ch29_persist` 仍是 `BLOCKED`；後續條目已取代這個歷史狀態。

### 2026-08-21 ch28 post raw 前置規格

合法 IDA Pro 9.4 補證已固定 `0x254C0→0x35BBA` 的 start slot20、
`0x254C8..0x254D8` 的 slot20 raw `+7/+8=0x7E`，以及 `0x35BBA` 清除
slots20..tail 的 raw word `+0x40` 後必定進入 `0x1DB65`。typed raw transaction
必須要求每筆 `+0/+1/+3/+5/+0x40` provenance並零修改失敗；一般戰役可使用
constructor 已物化的 typed view，`CONTINUE` 若帶完整 `0x50` projection則還要
逐欄一致。partial raw projection一律拒絕。可接受的 pre-post
frontier只允許已證實 producer形成的 `76/78/80/82/84/87`，不得退回固定slot93。
`0x1DB65` 的13＋6＋6呈現控制流已閉合；啟動載入與 consumer 證據亦已確認
`[0x53A81]` 是 `FDOTHER.DAT #5` LMI1 bank、疊圖 entries 為 `0x44..0x4F`，
而 `0x25A96([0x53EEC],3,1)` 從 `FDOTHER.DAT #31` UI SFX pool 播放 raw
sample index 3。typed presenter 必須保留這些 archive／index／argument 與每次
present 邊界；不得用未證實名稱取代。entries 的高階圖像身分、sample 3 的玩家
語意及同狀態像素／聽覺 E2 仍未閉合，因此本規格不授權 generic redraw，也不
自行解除正式戰後 binding。

## 2026-08-21 終局 20 段非零 renderer E1 契約

主證據為
[`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt)。
`0x2C2A6` 的20次呼叫已證實 `[0x540FF]` 全為非零，因此正式重製端不得再走
`0x28A6C` 的一般戰鬥結果、HP／狀態面板或 mutation 分支。每段的 typed input
固定為兩筆 loader record value copy、兩組直接 table `+6/+7` 覆寫、table2
`global_540ff`、TAI／BG與四組 FIGANI；全部20段須在發布第一幀前一次驗證。

每段呈現順序必須保留：

1. `0x29164` 的九段 `8..0` indexed 滑入與 `6*stage` palette；
2. record0 auxiliary＋record1 base 的第一個 `0x2939D`；
3. `[0x540FF]=1` 後 record1 auxiliary＋record0 base 的第二個 `0x2939D`；
4. caller-owned 20-tick等待、FDOTHER #58同 index overlay、78-tick等待與清畫面；
5. 20段後呈現 FDOTHER #59並永久停留，不進 town、save或下一章。

`0x2939D` 需依 FIGANI raw `+4/+5/+6/+7` 保留有效 frame、音效 marker、inner
present count與 z-order；animation header byte1 所屬的十段左右滑入亦不可省略。
目前 sound owner globals沒有已證實 writer，故 E1 renderer只保存 marker並不猜
sample。原版 1 tick 由重製端採既有約55 ms近似，不宣稱 DOS BIOS逐毫秒一致。

`0x1088D(0x1E)` baseline穿過 `0x2C405` 間接 helper後仍保持完整 records不變，
目前只有「函式內沒有直接 record store」的強推論，沒有動態 watchpoint。因此
正式 E1可以在使用者已允許近似的產品界線下消費該 baseline，但 UI／文件必須
標明來源約束 E1；不得提升為位元 exact、逐像素 parity或 `PLAYER-E2`。缺
persistent raw record、FDFIELD／FDICON／item table、任一20段資源或 baseline
一致性時，必須在第一幀前失敗即關閉且不得發布部分結局。

### 現行產品接線狀態

- `campaign_full.json` 的 `battle_ch30` 會先以 `ending_party_snapshot_on_win`
  同步最後戰場，再進入 `source_bound_e1_terminal_hold`。
- runtime 已會播放前綴、`0x2C548` 角色蒙太奇、現有的20段原資源
  視覺橋接，最後停在 FDOTHER #59。這是來源約束 E1，不再需要
  `FD2_APPROXIMATE=1`。
- 2026-08-22 direct operand 更正 FIGANI header：byte0為總幀數、byte1為前段
  旗標、byte2為前段幀數；不得把byte0／1當`u16 frame_count`。終局20段
  實際選到的80個 FIGANI 全部 byte1=0、byte2=byte0，故本批不進
  `0x29C90`／`0x29DED` 前段滑入；非本批的 byte1非零資源不可偷渡。
- 尾段 adapter 必須逐 raw `+6` inner present 執行，每次等待1 tick
  並推進配對 base scheduler；raw `+7 bit0` 決定 auxiliary／base 前後層。
  `+4!=0` 將位移相位重置為5，依 raw tables `0x5255F`／`0x52577`
  消費5→0；第二次配對在最後一個 `+4!=0` frame 完成後結束。
  effect 首次 inner present 對 base 使用 `0x4E63D(...,33)`：只將 RLE
  不透明像素覆寫為 palette index33，透明 span 仍保留背景。
  `+5` 只保存 raw marker，在 sound owner globals 未閉合前不播放猜測樣本。
- 兩次配對固定為 record0 auxiliary＋record1 base，然後 record1
  auxiliary＋record0 base；不得畫出被動畫者自己的 base。重製可以320×200
  indexed surface 呈現這個已證實配對，但仍不宣稱 DOS 逐像素／音訊 E2。

## 2026-08-22 空游標系統設定選單 E1 契約

主證據為
[`fd2_system_overlay_options_ida.txt`](../data/ida/fd2_system_overlay_options_ida.txt)。
固定雜湊 `FD2.EXE` 的 `0x16F55` outer selector 只有 index 2 呼叫
`0x1728C`；index 0 的巢狀存讀檔／離場與 index 1 的全軍行軍是不同交易，
不得由本段順便猜測接入。

`0x1728C` 的四個原始 byte 與畫面 cell 契約如下：

| selector | 原始欄位 | 值域 | cell（開／關） | 已證實消費端 |
|---|---|---|---|---|
| 0 | `0x51E61` | `0/1` | `54/57` | `0x25977` 音樂音量 gate |
| 1 | `0x51E62` | `0/1` | `60/63` | `0x25A96`、`0x25B45` PCM gate |
| 2 | `0x53AF9` | `0/1` | `66/69` | 戰鬥演出路徑 gate；替代演出 owner 尚未閉合 |
| 3 | `0x51AAB` | `0/1` | `72/75` | `0x1ACF3` 原生狀態列 gate |

typed state 必須在發布第一幀前驗證四值皆為 `0/1`，並要求
`FDOTHER #2` 至少具有76個可解碼 cell。方向鍵只改 selector；Enter 切換目前
欄位後重建精確 cell；Escape 走四次 closing present，再回到戰場。開／關音樂
需保留所選曲目以便重新開啟，音效欄位直接控制 PCM 發布；狀態列欄位同步正式
HUD persistence。`0x53AF9!=0` 時，在原版 abbreviated presentation owner 未閉合
前，重製端不得把完整 indexed 演出或任意快速動畫冒充替代路徑，應在交易前
失敗即關閉。

原版 `FD2.SAV` metadata `+14..+17` 保存這四個 byte；重製 JSON 存檔也必須
全數持久化。舊 JSON 缺欄位時採原版初始值 `0,1,1,1`，但顯式越界值必須拒絕。
目前完成本選單只可標成 `RUNTIME-E1`；尚缺同狀態原版逐幀與聽覺驗證，不能
提升為 `PLAYER-E2`。

## 2026-08-22 空游標巢狀系統選單前置規格

主證據為
[`fd2_nested_system_menu_ida.txt`](../data/ida/fd2_nested_system_menu_ida.txt)。
外層 selector0 必須先驗證 nested cells `36、39/41、42/44、45`，完成外層四次
closing present後才開巢狀面板。第二表 index1 由 runtime raw `+5` bit組合的
直接 writer 決定，index2 由 `FD2.SAV` 是否不存在決定；cell變體本身不再被泛化
命名成啟用／停用語意。缺raw provenance時不可猜值。巢狀Escape經四次closing
返回戰場，不得觸發任何交易。

selector0的`0x1B1E7`須一次預建320×200 indexed baseline、資訊畫面、12張展開與
12張收合；資訊畫面消費archive entries `0x85..0x88`、FDTXT、章節、round、
currency及三camp的typed count。等待輸入期間只推進已證實的animation clock與
DAC 224..239 pulse。任一資源或count provenance缺失時，巢狀面板不得先收掉。

selector1原版保存完整current-runtime，selector2以`0x10010`完整還原，selector3
回傳`-1`。現有JSON battle-node restart與原版current-runtime snapshot不是同一
交易，因此本段只授權nested selector與selector0資訊畫面，不授權把selector1／2
接到語意不同的save helper。selector3目的地後續已由本頁「巢狀離場規格」閉合；
本段當時的未知狀態不可再作為現況斷言。

實作現況：`NativeNestedSystemActionOverlayState`、
`ComposeNativeSystemInfoSurface`、`NativeSystemInfoCampCounts` 與
`NativeSystemInfoTransitionFrames` 已接到正式空游標鍵盤路徑。selector0會先完整
預建四個panel、六個十進位欄位、兩行FDTXT、12張展開與12張收合，再關閉巢狀
面板；等待任意鍵期間沿用已閉合的`0x4DFCC` DAC `0xE0..0xEF`循環。缺完整
runtime raw欄位、indexed VGA／DAC、panel、字型或文字時不發布第一幀。此切片列為
`RUNTIME-E1`；尚缺原版同狀態逐幀、BIOS tick相位與輸入flush的`PLAYER-E2`。

### selector1 「保存目前戰況」寫入契約

固定雜湊 `FD2.EXE` 的 `0x19F1A..0x1A198` 直接證實：玩家選 YES 後，
原版寫入 field control `0x8A3` bytes、persistent records `0xA00` bytes、
`runtime_count * 0x50` bytes、event state `0x20` bytes 及18-byte header，再重算
checksum、套用 XOR envelope 並覆寫 `FD2.SAV`。原始位址、caller、
FDTXT `0x19A/0x19B/0x19C` 與原始檔案交易見
[`fd2_nested_system_menu_ida.txt`](../data/ida/fd2_nested_system_menu_ida.txt)。

重製端必須把這項交易分成三層：

1. `fdsave` 純函式只將已完整驗證的 `CurrentSnapshot` 覆蓋到一份
   checksum-valid plaintext 副本。它要求 runtime、persistent count 與實際陣列
   一致，保留四個 chapter slots 與其他未命名 bytes，不從正規化角色
   反推不明 ABI。
2. `Game` 只可從一份已通過 native CONTINUE handoff、且至今仍保留
   完整 raw baseline 的戰場建立快照。地圖座標、面向、動作、生存／已行動
   bits、HP／MP、能力、物品與指令等已證實欄位，必須由目前
   typed state 回填原 raw 副本；任一欄位越界、陣列順序不一致或缺少
   provenance 時失敗即關閉。runtime 數量相對 baseline 改變時也必須拒絕寫入，
   直到增援／入隊記錄的 persistent raw 投影另行閉合；不可混寫新 runtime 與舊
   persistent 區。不允許把可空的 authored battle 假裝成原生快照。
3. 檔案擁有者先讀取與解碼現存 `FD2.SAV`，私有地建立新 plaintext、
   checksum 與 envelope，寫入同目錄暫存檔並同步完成後才原子取代。
   問句、YES／NO、成功／取消文字與面板收合的每一幀均由既有 indexed
   confirmation owner 呈現；只有確認分支會碰檔案，寫入失敗不得
   顯示成功文字或收掉可重試的戰場狀態。

本契約首先關閉 selector1；selector2 還需要將現有 title CONTINUE adapter
改造成可在現行 battle controller 內原子替換的 load transaction，不與 SAVE
同批猜測接入。兩者都不得落回重製 JSON 存檔。

### selector2「讀取目前戰況」發布契約

selector2 不另造第二套解碼器。它必須重用已閉合的 `Decode`、
`InspectCurrentSnapshot`、`BuildContinueRuntimeInput` 與五個 typed handoff，差別只在
caller：標題 CONTINUE 從標題時鐘取得呈現種子；戰場內 LOAD 從目前 battle clock
取得最近已物化的 signed low word。此值只維持既有動畫 delta 邊界；DOS BIOS、PIT
與硬體計時細節依公開規格視為平台介面，不再作為 remake 反組譯阻擋。

玩家確認前，應在私有 `Game`、map、scenario 與 `battle.State` 完成檔案解碼、章節
唯一映射、field／runtime／pending groups／view／HUD／options 與 controller handoff。
問句使用 FDTXT `0x19D`，接受回覆使用 `0x19E`，取消沿用共同 `0x19C`。只有 YES
且接受回覆已呈現後才以單次發布取代目前戰場；NO、檔案不存在、checksum 錯誤、
章節資產不唯一、時鐘來源不存在或任一 typed adapter 失敗，都不得清除目前戰況。
發布後的 `nativeCurrentSavePlain` 必須換成剛讀取的 plaintext，使下一次 SAVE 仍以
同一份完整 raw baseline 工作。

current SAVE 的 persistent 同步另採 identity-keyed 交易：既有 persistent records
先由 baseline 保留未知 bytes，再以 runtime／party 中具 `+8` 原始身分與完整欄位
provenance 的角色覆蓋已證實欄位；新 JOIN 只能使用已證實 constructor 產生的完整
raw record，並填入第一個未使用 persistent slot。身分重複、順序衝突、容量超過32、
找不到 constructor raw 或 runtime 增加不是已知 JOIN／增援時，整次 SAVE 失敗即關閉。

### `sub_1A866` 狀態倒數的正式回合邊界

三個直接 caller 已由固定雜湊的 IDA／Capstone 指令共同固定：`0x1A4D1→1` 位於
`0x1D80B` unit scan 前，`0x1A55E→0` 位於 `0x1D8BA` unit scan 前，
`0x1A797→2` 則在 phase 畫面收束後設定 raw range mode 1、進入玩家輸入前。
正式執行期依此順序遞減 raw `+0x22..+0x27`；歸零時再依 `0x1B750` 重新計算
derived AP／DP／HIT／EV。整批先在私有 runtime records 與 units 完成，缺少完整
原始投影或裝備重算資料時不發布任何變更。

現行重製 AI controller 將原版 selector 1／0 的兩段 unit scan 合併成單一非玩家
排程，無法在不重寫 scheduler 的情況下保持兩段 scan 交錯。其最小充分接法是在
合併排程前以單一交易依序執行 `1→0`，回到玩家輸入前再執行 `2`；這保存各 raw
record 的倒數邊界與原始 sweep 順序，但不是兩個 unit-scan 的逐指令等價實作。
selector 仍不得改名或映射成 normalized `Camp`。到期畫面依
[`fd2_transient_expiry_presentation_ida.txt`](../data/ida/fd2_transient_expiry_presentation_ida.txt)
消費目前 320×200 indexed map、raw `+7` 的 DATO 第一幀、FDOTHER #5 對話框、
FDTXT #481..486 與 `0x9F23` 文字位置；每筆以六幀展開、五幀收合並復原來源。
整批畫面先完整預建，缺背景、DATO、字型、文字或對話格時連倒數交易也不發布；
最後一幀完成繪製才續接敵方或玩家階段。這是來源約束的 `RUNTIME-E1`，尚未證明
原版精確停留 tick、音訊或一般玩家 E2。`+0x22..+0x27` 的完整遊戲名稱／圖示仍
未知，不能用 legacy `PoisonTurns`／`Sealed` 名稱冒充原版狀態列。

物品方面，`0x20C6F` 已恢復的 type 5..24 分派均已有正式交易消費端：HP／MP、
marker 清除與套用、HIT／EV、AP／DP、基礎能力／容量、command damage 與 relocation。
此處的剩餘缺口是 indexed effect presentation、原版取消／不可用目標畫面與同狀態
逐幀／逐音訊 E2，不再把「物品 effect 尚未接」列成數值交易缺口。未知 command、
複合技中只有ID34已具受限玩家state owner；ID32／33／35與尚無正式owner的法術仍各自失敗即關閉。

## 2026-08-22 巢狀離場規格

主證據為
[`fd2_system_exit_and_group_march_ida.txt`](../data/ida/fd2_system_exit_and_group_march_ida.txt)。
巢狀selector3必須沿用DATO #75、FDTXT `0x19F/0x1A0/0x19C`、共同YES／NO
四幀展開／收合、逐字形回覆與200 ms等待；任何資產不完整時須在巢狀四格收合前
拒絕。取消只收合對話並回戰場，不改戰鬥或戰役狀態。

接受路徑只有在問句、選擇框、接受回覆、200 ms及對話框收合都完成呈現後，才可
發布程式結束結果。重製端以`ebiten.Termination`離開主迴圈，讓既有defer／audio／
window清理接手；不得改成回標題、回城鎮、跳章或未經確認直接關閉。原版
`sub_25977(-1,1)`的精確BGM淡出仍屬音訊E2，不阻擋先以現有安全停止流程達
`RUNTIME-E1`。

實作現況：正式巢狀 selector3 已使用上述索引問句與逐字回覆，並把結束要求延後到
十二個60 Hz近似等待畫格及五個對話框收合畫格全部呈現後；只有接受分支會先停止
BGM並發布`ebiten.Termination`，取消與缺資產均保持遊戲狀態。此成果列為
`RUNTIME-E1`，不宣稱原版BGM淡出曲線、BIOS tick或逐幀相位`PLAYER-E2`。

同一證據亦閉合外層selector1的raw loop；本段當時尚未識別間接表的原始基址，
故不授權把它簡化為「逐一呼叫現有敵方AI」。後續補證與正式子集以緊接的
「外層 selector1 全軍移動規格」為準。

## 2026-08-22 外層 selector1 全軍移動規格

主證據仍為
[`fd2_system_exit_and_group_march_ida.txt`](../data/ida/fd2_system_exit_and_group_march_ida.txt)。
後續直接指令補證確認間接表基址是既有`0x51B91`，而 selector1 完全不讀
`sub_14B78`回傳值。typed planner 必須保存：runtime record原順序、
`(raw+5)&0x85==0 && raw+6==2`准入、共同目的座標、selector 1、
`sub_14B78`已閉合的mode0／mode1尋路與逐單位占位變更。

正式執行前必須在私有狀態投影預演目前可見的合格單位。每一個向左格提交後以
`NativeFieldEventIDAt(cell,1)`模擬`0x13A44→0x51A8F`；若事件非空且90-entry
handler沒有正式執行端，整批在關閉確認框前失敗，不得發布部分移動。無事件時
可依序播放移動，逐筆清raw `+3`、設raw `+5 bit7`／typed acted。

event61／event75 已各有具型別 Plan／Commit 與正式對話／演出 owner，故後續切片
可在走到事件格時暫停該筆 walk，轉交既有 owner，完成後從同一路徑下一格繼續。
這兩個 owner 的完整資產與 raw provenance 也必須在接受確認前由私有投影驗證；
不能先移動再發現 handler 不可執行。event61 會追加 group1 records，而新補證的
`0x170C6 cmp ebx,dword_53BEB` 與 `0x1100B inc dword_53BEB` 證實原版每輪重讀
動態 count；因此每筆完成後必須依目前 State 長度續規劃，不能永久消費確認時的
固定 steps。新 record 是否合格仍只由 raw gate 決定。取消、無合格單位、raw
provenance、成本列、地形、占位、事件或確認資產不完整時均不改狀態。

此切片完成後只能列`RUNTIME-E1`：原版FDTXT `0x1A1/0x1A2/0x19C`、200 ms、
逐單位路徑與無事件交易可閉合；90個全域事件效果、精確音訊／tick與一般玩家
同狀態比較仍各自保留門檻。

實作現況：正式空游標外層 selector1 已在確認框收合前建立私有逐單位 plan；
接受後依 record 順序播放路徑，完成每筆才發布座標、raw `+5 bit7`與 typed acted，
全批結束後進入共同回合收束。FDTXT、第一筆 runtime `+7` DATO肖像、取消、
200 ms與五幀對話收合均由正式鍵盤 owner 消費。任一路徑遇到尚無正式 handler 的
selector1格子事件時，在第一幀前整批拒絕。此為`RUNTIME-E1`，不是90事件全支援。

實作現況：`NativeSystemGroupMarchPlan` 會在深複製 State 上逐格模擬 event61／75；
event61 的私有 Commit 會追加 group1，使 planner 的動態 `len(Units)` 上限納入新
record，但不污染 live inventory、roster、event-state 或 turn rows。正式 walk 在
兩個事件格提交座標後暫停，由既有事件 owner 完成對話、演出及 mutation；成功
回呼再繼續該路徑與後續動態 record。事件 UI 素材也在確認框關閉前預檢。事件
owner 回報錯誤、事件綁定在預演後改變，或出現第三種 selector1 event 時，停止
整批且不得進入共同回合收束。這是既有已證實 handler 的組合，不授權新增猜測
renderer 或事件語意；正式資料目前只有 event61／75 兩筆 selector1 rule。
