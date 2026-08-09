# 56 — FD2 remake 系統設計規格（SDD，2026-08-09）

> 本文件是重新開始 remake 前的設計闸門。目標不是把目前能啟動的 Ebiten demo 擴張成更多 placeholder，而是以可追溯的反組譯證據，重建原版的操作介面、戰間流程、資料與腳本引擎。未滿足證據與驗收條件的語意保持 fail-closed。

## 1. 目標與現況判定

### 1.1 目標

- 原版 30 關的 campaign、戰鬥、戰後城鎮／商店／教會／整備、存檔與結局可循環遊玩。
- 對話、事件、商店、部署、過場和 UI layout 都由外部資料／腳本驅動；新增戰役不需修改 Go runtime。
- UI 操作語意以原版為目標：游標、action overlay／command grid、射程／目標、對話框、狀態欄、商店、教會和戰後節點均須有可見且可測的操作入口；未取得 E0/E1/E2 證據的現有 UI 只算 approximation。
- native indexed renderer 與現代 RGBA/Ebiten 顯示層分離；未完成 native ABI 時不得用泛用淡出、PNG 或空白畫面冒充完成。

### 1.2 現況（以 2026-08-09 working tree 與程式碼為準）

目前不是「沒有程式」，而是「有一個可跑的垂直切片，尚未達 remake」：`remake/cmd/fd2/main.go` 仍承擔 scene state、輸入 dispatch、戰鬥 UI、對話、town、shop、church、preparation 與 Draw；`internal/battle`、`internal/campaign`、`internal/ending`、`internal/figani` 已有可測的部分 primitive。這些 primitive 不等於原版 UI 或完整 campaign。

已存在但必須重新驗收：story/cutscene BeatRunner、dialog 分頁／捲動、campaign node、persistent roster、shop buy/sell/equip、church revive/class-change、preparation quota、indexed ending prefix。明確缺口包括：原版選單完整 dispatch、可見的回合結束流程、武器射程、完整 spell effects/演出、HUD 避讓、完整 UI sprite/layout、所有 postbattle branch、native ending montage。現行戰後稽核為 19 active／5 blocked；玩家第 22、23、24、25、29 戰仍失敗即關閉，已接切片也只達重製端 E1，不能當成一般玩家 E2 或完整 30 章完成。

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

### AI unit/action call-graph boundary（E0 raw, runtime 未開放）

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
這取代舊文件把 `0x15140` 稱作 AI entry 的說法。重製端只保存
`SelectNativeAI14EF0Tail` 的無副作用 raw 路由，缺 provenance 即失敗關閉，
不接 `NextAIPlan`。SDD 只授權保存上述 raw call topology；
`+6` 的 raw camp code 已由 constructor 與 `0x14818` consumer 固定為
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
缺失時失敗即關閉。這是 E0 diagnostic bridge，不是 command record loader、
正式 AI planner、戰鬥執行器或 UI consumer；`NextAIPlan` 仍維持近似路徑。

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

本輪重新核對的已知更正：`0x16559` 是 DATO mouth-frame／glyph blit caller，`0x4ea2a` 才是 native glyph renderer；FDTXT `0x2c469` 前的 `0x1088d(30)` 會選 archive resource #31，不能直接把實體欄位命名成 ch30；`0x2c548` 有 `i=0→slot1、i=1→slot0` swap；`0x29164` 第一參數是 party unit index，TAI#3 是 7-byte transparent aux，不是可見台座。這些結論要在新工具鏈重跑後才能再擴展，不可由名稱推導 renderer 語意。

2026-07-28 visual audit correction：codec／原資源 fixture 的完成度不得再
寫成整體 UI parity。依 12 個主要界面逐項比較 repo DOSBox／錄影 oracle、
source-rebuild screenshot 與外部原版畫面後，目前 asset/codec 可重現約
75–85%、可操作 state flow 約50–55%，但玩家操作畫面的視覺還原約
40–45%。35–40%是同日較早、尚未計入ch02 town/shop indexed production
與DOSBox E2的初估，已由doc57分項矩陣取代。最大落差仍包括preparation、
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
| UI-02 | Battle field | 游標格、鏡頭、可移動格、高亮、單位 HUD、方向／面向 | partial；HUD 固定錨點與完整 native sprite 未閉合 |
| UI-03 | Action menu | move/attack/magic/item/status/wait/end-turn 的可見項、enable gate、取消回上一層 | partial；原版 action overlay 的 battle cell table（enabled `[0,2,4,6]`／disabled `[3,5,7,9]`）、open/close 四幀 byte-offset、以可視 cursor column/row 算出的 framebuffer anchor 已閉合。runtime 現以 caller-owned lifecycle 呈現 opening `0..3` 與獨立 closing `0..3`，並延後 command/item/spell/attack/wait side effect，直到第四個 close present 完成；因 native loop 無 delay call，只宣稱順序／present count，不宣稱毫秒時長。[8-frame Xvfb artifact](../figures/action-overlay-open-close-remake.png) 由目前 source 與玩家 FDOTHER.DAT read-only mount 產生。native command grid 亦已定為 320×200、每欄四列，label `(18+100*col,103+22*row)`、MP 右側、↑↓ wrap/←→±4 bounded；scenario raw command mask 已可 materialize。Docker/Xvfb 以 player FDOTHER.DAT 捕捉的 [悠妮 command-0 grid](../figures/native-command-grid-remake.png) 證實 mask→label→palette/font→renderer 路徑，非 original visual diff。`fdother.CaptureActionOverlaySnapshot`／`RestoreActionOverlaySnapshot` 與 `ActionOverlaySnapshotOrigin` 現已對齊原版 `0x175a9/0x17643` 的 72×72（`0x1440`）索引快照、每列 `0x1c8` stride、游標各減一格的 owner，並有失敗即關閉回歸；詳見 [IDA snapshot evidence](../data/ida/fd2_action_overlay_snapshot_ida.txt)。現行 Ebiten adapter 尚未消費此快照（每幀由整幅場景重畫避免殘影），因此 native backup/restore 的正式 runtime consumer、完整 native gate 與 DOSBox visual diff 仍未關閉。舊 PNG ring 是缺原始 asset 時的 fallback。 |
| UI-04 | Target/range/item selector | 武器 min/max reach、法術 range/AOE、item兩欄四列、不可用目標灰化、確認／取消 | partial；command/item targets與 observed item effects已閉合。`0x1b9de/0x184c0` 固定 compact prefix、input、layout與raw icon IDs；`0x18409` 的12-frame open11→0/close0→11及left/upper/bottom clipped rectangles已有Ebiten adapter。tracked item Enter transaction已接，但indexed effect presentation、完整weapon/AOE/LOS與DOSBox visual diff仍fail-closed |
| UI-05 | Dialog | 上／下框、portrait anchor、文字避讓、控制碼、分頁／捲動、嘴型、輸入鎖 | partial；`internal/dato.MouthState` 已按 `0x16D00` cadence 接入更新迴圈，native frame/資源與所有 speaker layout 未閉合 |
| UI-06 | Battle HUD | HP/MP/LV/name、面板 sprite、數字 cell、依游標避讓、palette/clip | partial；需以 FDOTHER/UI loader 和截圖差分驗收 |
| UI-07 | Postbattle | result → handler → reward/roster cleanup → town/shop/rest/preparation 或 ending；不可預設直連下一戰 | partial；campaign schema 與 bounded menu trace 可表達，`town_ch02→preparation_ch02→story_ch02_pre→battle_ch02` 已有可重播 trace。標準 postbattle 現有19個節點接入零起算 owner 的 authored binding，另5個維持 blocked；玩家第17、18、20戰已依 raw ch16、ch17、ch19 的直接控制流程分別接入60／61→61／62、55與83→84 runtime frontier，並保留 `town_ch18`、`town_ch19`、`town_ch21` 及 save/load 邊界。五個未綁定節點原有的泛用 `sync_party→set_chapter` 會繞過 runtime guard，現已移除；所有未綁定標準節點均以空 beats 失敗即關閉。這批只達 E1，逐關戰間畫面與一般玩家路徑仍不足；直接位址證據見 [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)、[`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)、[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)、[`fd2_ch12_post_dispatch_ida.txt`](../data/fd2_ch12_post_dispatch_ida.txt)、[`fd2_ch05_post_dispatch_ida.txt`](../data/fd2_ch05_post_dispatch_ida.txt) 與 [`fd2_post26_28_dispatch_ida.txt`](../data/fd2_post26_28_dispatch_ida.txt) |
| UI-08 | Town/hub | 可見選單、離開、shop/church/preparation 入口、BGM/SFX、持久隊伍 | partial；`campaign.MenuState` 已與 `choice/town` runtime 共用。ch02 variant0 的 [`selection0–5`](../figures/town-hub-six-selections-original-vs-remake.png) 都已達原版 DOSBox／source-built remake raw RGB 整幀相同；variant1與variant2 selection0–4 另有修改 LOAD 路徑 E2，兩組五項都與指定 pulse 640×400 整幀 AE=0。Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape回selection5亦有原版 input trace；shop/church/preparation 與 hotel raw route/return trace已接，仍需variant2 selection5 的 BIOS 掃描碼／Enter、未修改玩家路徑與逐章route E2 |
| UI-09 | Shop | buy/sell、商品／角色／slot 游標、裝備詢問、金錢／庫存原子更新、secret gate | partial；stable scene、四項service menu及purchase/sell/standalone-equip/transfer四條production owner已接原版indexed compositor。equip為角色roster後切入完整item/status panel；transfer保存FDTXT512/511/510/506與raw remove→append/recalc。ch02 variant1/3/5 service0 selected phase、variant5四service/wrap/Escape return、weapon purchase-list四個selection、Yes/No、gold0不足金與gold1000裝備收件者selection0/cycle1均達原版DOSBox／production remake同狀態raw RGB整幀相同；recipient E2使用screenshot-only party bootstrap，DX為E2約束的projection而非直接raw dump。正常campaign JOIN→LOADCH首次typed roster已接runtime regression，但尚非完整playthrough E2或native FD2.SAV。另修正pulse double-`/2`、返回selection0、choice-close frame ownership與比較欄位geometry。尚待recipient input/scroll、no-recipient/full/success、sell/equip/transfer與其他章節E2 |
| UI-10 | Church | revive、class change、費率、候選過濾、確認／取消、缺資料 fail-closed | partial；class path 已對齊 `0x31385→0x31793→0x311DC→0x19953`：Lv>=20、portrait<0x12 且 !=7，三列可見候選、上下 bounded，special>optional>default 自動解析唯一 target，再以左右 Yes/No 確認。`0x31019` 的 FDICON＋四段 FDTXT row、FDOTHER#14 entry16 panel 與 `0x1974c` 六幀 opening 已成 indexed compositor。候選確認／取消會先跑 `0x2d31b` 五幀 closing＋source restore；`0x19953` 已接 FFFC 動態角色名、FDOTHER#2 cells16/17、48/49與51/52 normal/pulse、四幀 opening／`0x197e5` 四幀 choice closing，之後再跑 dialogue closing 五幀＋source restore，最後才 mutation／返回。所有幀只由 Draw acknowledgement 推進。`0x3072f` stable scene 已由FDOTHER#5 raw grid/four-mode digits、FDOTHER#14 entry1、DATO#131與FDTXT585/586合成；`0x2d669`四幀開關、closing source restore及`0x2d85f`兩-tick selected pulse均接runtime並有原版資源artifact。FD2.SAV、raw service0 command overlay與未接callee仍fail-closed |
| UI-11 | Preparation | 城鎮出發確認／無城鎮記錄詢問、依名冊門檻略過或進入部署、可選15／19筆另加固定 record0（總上場16／20）、取消、最終確認、進戰場 | partial；`0x2cad7` 與 `0x2d093→0x318ad` 的分流已接資料模型與原版提示。城鎮路徑保留實際 town frame，使用 FDTXT `0x201` 於 `(95,119)`；無城鎮路徑依 `0x2cc04` 清成黑畫面，使用 FDTXT `0x19a` 於 `(100,119)`，肯定結果在完整關框後才存檔。兩者都使用 DATO #75、FDOTHER #5 對話框、FDOTHER #2 Yes／No 與 6＋4＋兩 tick 脈動＋4＋5＋還原生命週期。`0x318ad/0x31e80` 的三區背景、計數、10 欄角色格、游標、彩色／灰色繪法、四向輸入與 `0x17fc0` 狀態已接；`0x320fc` 證實 selection byte i只重排 persistent record i+1，record0固定且不消耗quota。`0x1297d` 的有號 BIOS 低字差值與可見 `0,1,2,1` 待機週期亦已接。`0x31d3c` 最終確認沿用相同生命週期，文字為 FDTXT `0x292`，呈現原畫面後才處理結果。呼叫端也已閉合：城鎮出發 `0x2d16b` 收到 0 會退出，直接整備 `0x2ccd6` 收到 0 則重選。缺任一原始記錄或資源即退回。README 所列整備圖均為 E1 原始資源合成，不是 DOSBox 截圖或正常晚期戰役存檔。跨畫面初始相位、有效晚期存檔及原版實機差分仍缺。`0x1f42d` 屬戰場進入演出，不再列為選人視窗動畫 |
| UI-12 | Save/load | scene-safe boundary、campaign cursor、flags、party/inventory/equipment、version/checksum、四槽 selector | partial；remake title LOAD 已還原四槽 bounded selector 與原版 indexed compositor。合法 IDA 9.4 固定 reader `0x2602c..0x26098`、writer `0x30012` 及其僅有的 `0x2ccb6/0x2fd93` 戰間呼叫者；兩端只處理 metadata `+0..+9`。production 以綁定參考 EXE 雜湊與 `0x526b9` 的 editable gate table 將 raw chapter 1..29 還原到既有 town／preparation node，先完整驗證 persistent record→typed party、節點型別及重複 identity，再一次套用 campaign cursor、gold、party 與四個 raw option bytes；錯誤不部分 mutation、不誤轉 JSON loader。ch21／ch27 inventory postbattle gate 已在存檔前完成，LOAD 不重播。空槽及修改存檔 chapter1 有效槽畫面均與 DOSBox 全幀 RGB 相同；合成有效槽 restore 是 E1，仍缺未修改一般玩家有效槽 E2、CONTINUE current-battle、metadata `+10..+39` 其他可能 consumer、刪除／覆寫 |

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
`0x2c194..0x2c39a` 的資源與迴圈契約：FDOTHER #60 的前置解碼、#58／#57
的 20-entry raw table loop、`unit+6/+7/+0x56/+0x57` 寫入、
`0x28a6c(0,1)`、`0x11d40(0,255,0)`、`0x2935b`、20／78 tick、
`0x1f882`，以及最後 FDOTHER #59 的解碼與釋放。三組 20-byte 表以固定版
原始位址 `0x525dc..0x52617` 輸出並帶雜湊；`MontageTail.Plan` 只產生 raw
entry，不寫入 `battle.State`、不呈現畫面，也不命名欄位。這關閉「尾端完全未知」
的過時斷言，但不關閉 indexed resource owner、輸入事件、campaign／town／
 shop／整備／save handoff 或 `postbattle_ch29_persist`；證據見
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)。

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

Generic scheduler closure：`funcs_2ac25` 是 command-indexed function bank（ID0 entry `0x26152`）。`0x2a6bd` 先以 mode 0 呼該 entry 取得 animation step count，接著在 640-stride off-screen buffer 的逐 step loop 呼 mode 2、`0x11eb0` copy 320×200 至 VGA、`0x17aa9(1)` tick、再呼 mode 1；收尾的雙 buffer path 還會呼 mode 4。`0x2b9a1` 並非未知 effect，它以 descriptor `frameIndex*4+8` 指向 frame的 byte+6 delay，遞增 `0x540fc`／`0x540fd` subframe counters並在上界 reset。這固定了 phase/order，仍不替每個 command entry 的視覺語意命名。

Generic presentation 的 BG selector 亦已閉合為 raw dataflow：`0x2a6bd` 呼 `0x2b5e1(finalCount, finalTargetArray)`，後者**倒序**掃 target slot，對該 unit cell 呼 `0x12e38`；若 raw `0x1f183` gate 不通、或累積 selector 為零，才以 decoded control byte+2 取代 selector，最後才餵 `0x111ba("BG.DAT", selector)`。`fdicon.NativeCommandBackgroundSelector` 保留該 strict pure rule。command ID 的 generic branch 不可被說成直接選 BG resource；selector 的高階地形／場景語意仍不命名。

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
`0x276ec` 先經 `0x2b659`；該 event 對 ID24 以 `0x1ca89(actor,0x18)` 扣 record24 `+5` MP。原版為了
多段演出會先暫存 total delta、把 HP 復原，再以等份遞減回最終值；state-only remake 因此可一次套用相同
最終 delta。AI alias 的設計意圖與 native UI/SFX/timing 仍未以 remake regression 關閉，故不可冒充 ID16 heal
或接入 generic numeric executor。

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

IDs20..21 共享另一條「flag-present 才生效」route：`0x22A85/0x22BC6→0x22AA8→0x22AF6` 各以
command ID 20/21 扣 MP，對每個 final target 讀 `+0x25/+0x26`。該 byte 為零時只走失敗 display；非零時呼叫
`0x1C916(target,10)` 的既有 HP-restore writer、清零該 byte，並顯示結果。這證實 raw gate、clear 與
HP writer，但尚未命名兩個 status，亦未接 engine/UI。ID22 是不同的 `0x22BE1→0x22CDA→0x22D1B` route：final
target 的 `+0x27` 必為零、class `+0x20` 不得為 `0x19/0x1a`、且 `rand()%100<0x32`，才以
`0x1C81F(target,10)` 固定扣 10 HP、顯示 damage，並寫 `rand()%4+2` 至 `+0x27`。它須獨立追蹤，不能併稱為 cure
或依 raw offsets 猜測 status name。

這六個 transient bytes 的 decrement 已由 official IDA 釘死，但 gate 仍是 raw ABI：已重跑的 caller `0x1A4D1`、`0x1A55E`、`0x1A797` 分別傳入 selector 1/0/2；`0x1A866` 只接受 `record+6 == selector` 且 `(record+5 & 1)==0` 的 record；不可把它改寫成 `Camp/OnField/Alive` normalized 條件。通過 gate 後依序對 `unit+0x22..+0x27` 的每個非零 byte decrement。任何一個 byte 變零時才顯示 expiry feedback 並呼叫 `0x1B750(unit)` 重算 derived fields；因此 ID17/18 的 AP/DP 增幅會在自己的 duration 歸零後由重算移除，其他 flag 不可因為共用 sweep 就被誤認為同一 status。這是 phase-based timer ABI，不是每次 action 或 frame 的 timer；status labels/UI icon 仍未命名。

同一場景流程中的 `0x1A7BD`/`0x1A7F1` 不是 transient selector 語意本身：前者在 `[0x53AF9] != 0` 時以 `0x111BA(0x1A4D,0,0x40)` 建立 resource handle 並寫 `[0x53B0F]`，後者釋放該 handle。`0x1A4EB` 與 `0x1A58F` 都採「setup → unit scan → release」順序；因此 selector→campaign phase 仍不可由這兩個 resource helper 推導。

Remake 已以 `Unit.NativeTransient[6]` 及 optional `NativeRecordByte5/6` 保留這段 raw ABI，並提供 bounded offset access（只接受 `0x22..0x27`）及 `State.TickNativeTransientsRaw(selector)`；FDFIELD b0→runtime `+6` 的 parser/exporter provenance 也已補上，缺少 raw gates 時仍 fail-closed。它刻意不呼叫 normalized `TickStatus` 或 legacy shared `BuffTurns`，也尚未自行接 campaign equipment recompute；expiry consumer/UI 必須先帶入 `0x1B750` 對應的資料依賴才能開放。

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

IDs25..27 也已由 jump-table 閉合。ID25 `0x22C04` 以 record25 扣 MP，僅對 final target 已有
`record+5` bit `0x80` 已設的項目清該 bit，直接保留 raw clear writer（不命名 acted/action-complete）。ID26
`0x22CBF` 與 ID27 `0x22E41` 分別將 command ID 和 flag offset `+0x25/+0x26` 傳給與 ID22 同一
`0x22CDA→0x22D1B` application helper，所以同樣受 zero flag、class、`rand()%100<50` gate，成功固定扣 10 HP
並寫 2..5 duration。這使 ID20→`+0x25` clear 與 ID26→`+0x25` apply、ID21→`+0x26` clear 與 ID27→`+0x26`
apply 成為 direct code-pairs；仍不以此取代 UI/status icon 的獨立驗證。

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
| 13–16 | `0x21AD9…0x22153→21B18→1C8ED/1C916` | `ExecuteNativeCommandHeal` | 專用 animation/SFX/grid confirm 未接 |
| 17–19 | `0x226EA/2282F/22960` modifier writers、`+0x22..+0x24` duration；`__CHP` toward-zero 已釘死 | `ApplyNativeCommandModifier` 已接 raw writer；`ApplyNativeRuntimeEquipmentRecalc` 已保存 `0x1B750` 的 exact 1.15／朝零及HIT/EV+15。command transaction與phase-expiry caller仍未接 | 未接 |
| 20–21 | `0x22A85/22BC6→22AF6`，clear `+0x25/+0x26` 並借 record10 restore | `ExecuteNativeCommandClearRestore` | 未接 |
| 22 | `0x22BE1→22D1B`，class/RNG gate、base10 經第二 RNG 實際9 HP、第三 RNG write `+0x27` | `ExecuteNativeCommandApplication` | 未接 |
| 23 | `0x2218A→22253` special relocation selector | 已接 first target → mode-6 destination cursor；27-present indexed renderer 未接 | 已接 raw MP/座標 transaction |
| 24 | 玩家 `2A6BD→276EC→2B659/1CA89→1C81F`：`actor +48 * 15/10 - target +4a`；AI table 另別名 `22153`，不可混用 | `ExecuteNativeCommand24`（state-only final delta） | multi-hit／SFX／native UI 未接 |
| 28, 29, 31 | 同玩家 `276EC` derived-strike route，倍率分別 20、12、18；各自 record MP/一般 two-stage selector | `ExecuteNativeCommandDerivedStrike` | multi-hit／SFX／native UI 未接 |
| 30 | `1CFF0→14818→115B6` 先確認 record+3 candidate；再以 saved cursor→confirmed cursor 進 `149F8`，`count=record+3-16`、X-first cardinal line、只收 enemy，最後 `2A6BD→276EC` default倍率18 | `ExecuteNativeCommand30`（顯式兩 cursor、state-only final delta） | native cursor lifecycle／multi-hit／SFX／indexed UI 未接 |
| 25 | `0x22C04` clear target acted bit | `ExecuteNativeCommand25` | 未接 |
| 26–27 | `0x22CBF/22E41→22D1B`，分別 write `+0x25/+0x26` | `ExecuteNativeCommandApplication` | 未接 |
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
visual group、其他 MP writer 與 compound effect ordering 仍未閉合，engine 保持 fail-closed。

The wrapper's only direct caller is `0x2a7ce`, entered from `0x2a6bd` when
the opaque command selector is `>=0x20`; it passes four caller-owned values
without a proven normalized type. Inside `0x27fc9`, resource setup and the
`0x29164`/`0x2b659` presentation chain precede the ID-specific raw operations,
then indexed redraw/present loops and resource cleanup run for all four IDs.
`battle.NativeCompoundCommandPlan` exposes only this verified order and raw
marker/amount bytes as editable data. `Callee==0` denotes ID33's three direct
byte clears, not a guessed helper. The plan does not execute, debit MP, choose
targets, or infer effect/status names.

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

升級的 dynamic producer 現已閉合，但僅限資料層：native `0x1e292` 在 EXP 達門檻後增加 runtime
level，從 portrait growth row 的 `learn_idx` 經 `0x4e4a2` 查 `0x626b3 + idx*12`，逐一比對最多六組
`(required_level, command_id)`；命中就呼叫 `0x1d79c` OR command bit，並顯示 FDTXT_000 #587「學會了！」。
`docs/data/exe_tables/command_learn.json` 已保存 20 張 raw table（`FF/FF` sentinel 不轉成假資料）。
growth-row 的**raw selector**是 direct ABI：`0x4e4d1(unit+7)=0x620a1+unit[+7]*11`，第 11 byte 就是
`learn_idx`。constructor `0x10d7f..0x10efc` 已閉合 FDFIELD roster `b1→unit+7`；這是 battle
FIGANI/DATO selector 的來源，並不使它和 map `unit+2` alias。remake `State.GainExp` 因此只在已注入這個
editable table 時，於剛達到的 level OR exact
command bit；`remake/assets/data/command_learn.json` 是 runtime copy，`Game` 在每個新 battle state bind
同一張 table。legacy standalone `GainExp` 與 `Spells` 都不補造結果。

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

`cmd/fd2` now owns the playable adapter for this contract. Beat start preflights the complete field, tile-aligned camera, raw actor provenance, selector cache, LUTs, buffers and DAC before publishing a job; rejection leaves the existing map buffers unchanged. Each of the nine indexed presents must receive a real `Draw` acknowledgement, the 500ms tail is 30 update ticks, and each of the 32 cumulative DAC mutations must likewise be drawn before continuation. This preserves every native visual state and prevents a fast update loop from collapsing to the last frame. Because Ebiten's 60Hz host presentation cannot display distinct 5ms/4ms native writes at their original wall-clock cadence, the adapter deliberately uses one host present as the minimum duration and does **not** claim DOS timing parity. This transition adapter is executable only for an isolated, fully provisioned candidate; the current `postbattle_ch29_persist` has no active campaign binding and remains fail-closed until its later `0x2bce5` terminal renderer and owner index are complete.

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
implementations; recipient input/scroll, no-recipient/full, success/debit and
their DOSBox E2 remain separate gates below.

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

   Selected-unit HUD boundary: `0x11cac` calls `0x1acf3` after terrain, range and unit/foreground layers but before the viewport copy. `0x1acf3` returns without drawing unless both raw display bytes `0x51aab` and `0x51aac` are nonzero. Gate A is not a constant: `0x10010` restores it from native-save plaintext offset `0x30d2`; gate B has separate UI writers. It first calls `0x12e38` on cursor globals: that helper is a terrain-cell resolver, yielding FDFIELD tile word masked to 10 bits, event low five bits, and the selected four-byte FDSHAP control record; `fdicon.NativeTerrainCursorInfoForCell` preserves this raw contract. Its control byte+1 indexes the verified `0x51a12`/`0x51a2a` terrain AP/DP table: 0→(+5,0), 1/5→(0,0), 2/3→(-5,+10), 4→(-5,-5). `battle.Load` derives the same byte per validated map cell and combat consumes it directly. The now-closed panel geometry is FDOTHER #5 LMI1 #130 (69×34) at `buffer + stride*157 + x`, terrain icon at `+6`, AP signed-number path at `stride*8+0x2b`, and DP at `stride*19+0x2b`; `0x1aeb1` chooses raw directory entry #0x83 (6×7) for a nonnegative table value or #0x84 (6×5) for a negative one, makes the value absolute, then calls the native decimal digit path at `+8`. These are literal hexadecimal immediates in `0x1aeb1`, not decimal 83/84. The resource artwork's semantic label remains unassigned. `x` is the raw static anchor (data initial value 1). Direct Docker Capstone reading of `0x1ad2a..0x1ad5f` confirms it is persistent: only visible cursor row `[0x53abd]>5` plus column `[0x53ab9]<3` writes `0xf2`; the same row plus column `>9` writes `1`; all other pairs retain the prior global. These two globals are camera-relative cursor coordinates, not dialogue-box width/height; doc14's older assertion has been removed. `battle.NativeMapHUDRuntimeState` preserves the two raw gate bytes and anchor only when an explicit save/scenario source materializes them. The optional unit icon is FDICON group `unit+2 * 12 + rawState`, with raw state 3 aliasing 1, blitted at `stride*5+6`; its current/max HP words `+0x40/+0x42` feed `0x1875d` at `stride*21+9` in raw mode 3. `fdicon.NativeMapHUDUnitFrameIndex` preserves only that selector. The global and icon/HP semantic names remain raw.

   `indexedmap.BlitNativeMapHUDPanel` is the executable first subpass only: it requires both recovered raw gates, validates FDOTHER #5 entry #130's 69×34 geometry, and transparently blits it at `NativeMapHUDLayoutFor(anchorX,456).Frame`. **Codec correction:** #130/#0x83/#0x84 are directory entries sent to `0x4e63d`; `DecodeNativeMapHUDFrames` uses `ParseLMI1FrameEntry`/four-mode `Frame`, not ordinary `ParseLMI1` (`0x4e916`) cells. Terrain/unit icons and digits remain explicitly out of this primitive; callers can use it as the required HUD callback in `ComposeFrame` without pretending the partial panel is a complete native HUD.

   `indexedmap.BlitNativeMapHUDSignedNumber` closes the immediately following `0x1aeb1` selector boundary: it accepts an already-recovered number origin, uses #0x83 (6×7) for `value>=0` or #0x84 (6×5) for `value<0`, then invokes a mandatory decimal callback at `origin+8` with the absolute value. The primitive commits atomically only after that callback succeeds. It neither supplies a decimal font nor decides the table value, AP/DP source, or number meaning; those remain caller/data-flow work.

   That callback is now closed for this call-site by `BlitNativeMapHUDTwoDigitNumber`: `0x1aeb1` supplies `0x187d6` glyph base `0x1f` and fixed width 2; `0x187d6` patches `%0.5d` to `%0.2d`, then calls `0x16886→0x4e63d` for glyph directory entries `0x1f+digit` at offsets `origin+8` and `origin+14`. Real FDOTHER #5 confirms digit entries #0x1f..#0x28 are 6×8 except #0x20 (digit 1) at 5×8; advance is nevertheless six pixels. The adapter rejects values outside `0..99` rather than silently rendering native's truncated first two characters. Number source/meaning remains unassigned.

   Terrain-icon subpass closure: immediately after `0x12e38(cursor)` fills its eight-byte local, `0x1acf3` reads local word0 (the masked 10-bit terrain descriptor), uses it as the selected FDSHAP bank offset-table index, and raw-blits through `0x4deda` to panel `+6`. `indexedmap.BlitNativeMapHUDTerrainIcon` preserves exactly that raw descriptor input and destination, validating only editable bounds; it does not reuse texture previews or name the terrain category.

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
   零起算 dispatch，第29戰應使用 raw `ch28_post`。該 binding 尚未獲准啟用，因此
   `postbattle_ch29_persist` 現保持未綁定並由 runtime guard 停在 `preparation_ch30` 前。
   raw `ch29_post` 的 LOADCH／persistent-roster 與 `0x2bce5` 證據仍保留，但其正確 owner
   必須和第30戰結局流程另行閉合，不能用已恢復函式本身推定 campaign 接點。

   Presentation bridge (strict gate): `drawNativeMapHUD` converts the verified 456-stride indexed buffer to a 320×200 paletted Ebiten image only when `NativeMapHUDRuntimeState`, selector cache/cycle and every selected-unit raw admission byte are present. It now draws panel/terrain/AP/DP plus the proven unit icon and `+0x40/+0x42` HP path together. The former hardcoded `DisplayGateA=true, DisplayGateB=true, AnchorX=1` partial path has been removed because native load can overwrite gate A. Missing provenance falls back before any native drawing. `NativeMapHUDPersistentState` now separates save-persistent gate A、process-persistent anchor 與 controller-owned gate B；custom save and native chapter restore preserve gate A, while battle entry materializes gate B only from the proven value 1. `battle_ch01`、`battle_ch26` and `battle_ch27` use editable `native_map_hud_inherited` together with their evidenced views. Exact fixed HUD bytes remain available only for explicit fixtures/snapshots; this inherited owner closes E1 state flow, not whole-campaign visual parity.

   HUD pointer-base correction (2026-07-28): direct Capstone at
   `0x11cfa..0x11d0a` proves the caller pushes stride `0x1c8` and
   `[0x53a49]+0x8088` into `0x1acf3`. Therefore every recovered HUD offset,
   including row157, is relative to the same viewport pointer used by terrain
   and the later `0x11eb0` copy. The earlier production adapter passed the
   allocation base, causing the HUD to land 72 rows／72 bytes away and disappear
   from the final bottom edge; its regression had encoded that wrong coordinate.
   `ComposeNativeFrame` now passes `work[0x8088:]`, keeps failure atomicity, and
   verifies the panel reaches VGA `(anchor+4,161)`. The original-video
   [HUD oracle](../figures/native-map-ch01-original-video.png) and rebuilt
   [remake frame](../figures/native-map-ch01-remake.png) both show the panel at
   the lower edge. The reproducible extractor
   `tools/extract_fd2_video_frame.sh video/fd2-ch1.mp4 434.5 ...` first crops
   the recording's centered `(16,100,1408,880)` game viewport, then returns it
   to 320×200; direct whole-video scaling was a distorted oracle and is removed.
   The 434.5-second frame and remake now share camera `(1,13)`, absolute cursor
   `(8,15)`, visible cursor `(7,2)`, tree terrain and HUD `A -05 / D +10`.
   The screenshot hook formerly assigned only normalized `curX/curY`, leaving
   `NativeMapViewState` stale; it now drives `MoveNativeMapCursor` and persistent
   HUD-anchor updates. Roster/event presentation still differs, so these images
   prove camera/cursor/terrain/HUD alignment but not a full-frame pixel diff.

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

   Correction: the sprite pointer table is no longer opaque. `0x11019` builds/caches a twelve-pointer FDICON block from its raw key and resource arguments; `0x10c50` passes FDFIELD `b0` plus its caller resource and writes the returned cache slot to `unit+2`. `0x127e0` then selects `unit+2 × 12 + pose×3 + cycle`. Thus `unit+2` is a cache result, not a direct character/portrait field. This is distinct from battle `0x287b5..0x2884c`, which selects `FIGANI.DAT` by `unit+7 × 3`. Constructor `0x10d7f..0x10efc` closes FDFIELD `b1→unit+7`, so `export_units.py` writes `battle_fig`; missing older JSON retains an explicit `fig` fallback. `fig` remains an unseparated map-`+2` approximation. Cycle is global idle/moving animation state, not unit `+4`; that byte offsets the camera-relative placement. The remaining adapter boundary is the resource/key→slot materialization, remap selection and layer order—not an invented FIGANI mapping.

   `fdicon.NativeSelectorCache` is the fail-closed data primitive for this ABI: `0x11019` keeps one process-global first-seen raw-key table (`0x53b17`/`0x53bdf`) and rejects non-byte keys in the remake. Its second argument is consumed only to materialize a new twelve-pointer block; both player `0x10a25` and scripted `0x10b69` load `FDICON.B24`, and cache lookup itself does not compare a resource pointer. Scripted FDFIELD source is explicit (`b0`, also native camp `+6`); player source is persistent `+7`. It deliberately does not map a slot to a character, portrait, or archive index; the remaining boundary is full mixed player/scripted construction order and indexed layer integration.

   The pointer-copy detail is now represented too: `KeyForSlot` reverses `unit+2` to the raw B24 key, and `SpriteForNativeSlot` then applies `key×12 + pose×3 + cycle`. This matches `0x11019`'s copied twelve-pointer block followed by `0x127e0`; it remains a process-global key cache and does not infer character identity.

   Runtime stores native map selection separately as optional `MapSelectorSlot`: its presence means an explicit native `unit+2` cache slot (including slot zero); absence must not fall back from legacy story/save `Fig`. This keeps the indexed compositor fail-closed while legacy UI remains compatible.

   The editable boundary now also carries optional `map_selector_key` (`MapSelectorKey` plus presence flag), the raw byte supplied to `0x11019` before slot allocation. `battle.MaterializeNativeMapSelectorSlots` accepts only an explicitly ordered construction batch and a process-global `fdicon.NativeSelectorCache`; it validates every key before allocating first-seen slots, then writes `MapSelectorSlot`. Missing/invalid keys leave both unit slots and cache untouched. Loader and renderer do not call it implicitly: the caller must first preserve the player-persistent then scripted-spawn order.

   `State.AppendNativeMapSelectorBatch` now supplies the atomic state seam for that proven order. It owns one process-global cache and appends only a fully valid batch; party `[9,4]` followed by scripted `[0,2,0]` produces slots `[0,1,2,3,2]`, while a missing-key batch changes neither runtime unit order nor cache. All 33 versioned map assets now carry the explicitly sourced scripted fields through `tools/sync_native_selector_fields.py --check`. The battle scenario path is connected: `spawn_party` materializes the party first, and every later `AppendGroup` uses the same cache for scripted groups. A malformed or provenance-free batch may retain legacy unit append for compatibility, but records `NativeMapSelectorError` and disables native selector resolution for the whole battle; story actors and direct-start/retry paths remain outside this E1 claim.

   Player-party construction is a separate proven source path: `0x1088d` copies each persistent 0x50-byte roster record from `[0x53bf7]` into the battle roster at `0x10a77`, then passes the copied record's `+7` byte and the chapter FDICON resource to `0x11019`; only its returned slot is written to runtime `unit+2` at `0x10aa2`. Map-script construction instead reaches `0x10c50` and supplies FDFIELD `b0`. These are distinct inputs to the same cache ABI. The remake must preserve explicit source provenance/order before it materializes slots, and must not derive either path from legacy `Fig`.

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
Only destination confirmation commits command23 MP subtraction plus raw
coordinates. It requires the exact per-cell
`NativeTerrainMoveCodes` provenance and fails closed without it. The native
27-present indexed renderer remains a separate integration gate.

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
但不證明 #42 的圖形用途。`0x11EB0` 的共用尾端已證實將 312×192
複製到固定線性目的地 `0xA0504`；仍未知的是 ch23 中間目的地偏移與
固定版二進位的 `byte_51A10` raw seed 已由 IDA／Capstone 交叉讀出為
`0x01`；但 `0x24c1e` 會在迴圈間寫入 stage 2..14，因此 handler 入口的
執行期 latch 值與第一個 zero-argument `0x24d22(0)` 時序仍未知。IDA 現在
proves the shared indexed consumer chain inside `0x11cac` (`0x11eee` →
`0x122dc` → `0x127a9` → `0x1acf3` → `0x11eb0`), including ch23 call-sites
`0x24c63` and `0x24cd3`; this is static consumer evidence, not a claim that
the remake has a runnable adapter. Runtime entry latch state, `dword_53C03` lifetime,
the `0x53AED`／`0x53AF5`／`0x53AF1` runtime offset lifetime, full
`0x53aff` staging lifetime, raw state mapping, and
normal-player E2 still lack proof, so
`postbattle_ch23_persist` remains fail-closed. Detailed evidence is in
[`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt).

IDA data-xref 稽核已在不替欄位命名的前提下收窄擁有者邊界：`0x53aff` 由
raw `0x10652..0x1088d` 載入器配置／清理，再交給 `0x11eee`／`0x24d22` 消費；
`0x51a10` 只在 `0x24d22` 內讀寫；`0x539f8` 只由 `0x11eee` tick gate 比較／
寫回。offset globals `0x53aed`、`0x53af1`、`0x53af5` 則由共用函式
`0x12eaa`、`0x1300d`、`0x13185`、`0x13315` 寫入，包含明示的 `0x18`／零值
哨兵與累加更新。這只是靜態 E1 擁有者證據，不授權建立 camera、viewport、
framebuffer 或 `nativeMapWork` 別名，因此 ch23 production adapter 仍失敗即關閉。

The narrow executable boundary is now represented by
`fdother.DecodeNativeCh23Stage`／`BlitNativeCh23Stage`: they require archive
entry #42, exact 312×192 geometry, a `0x138`-stride `0xea00` staging surface,
and transparent `0x4e63d` blitting, while leaving the indexed state/latch
adapter outside the production handler.

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
ch22 post 對應的 indexed buffer owner 尚未完成前維持失敗即關閉。
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

本次稽核後，24 個標準 postbattle 節點為 **19 active／5 blocked**；story/cutscene
為 121 節點、9 個獨立 script、49 個 handler binding、63 個 fallback。剩餘
blocked 為玩家第22、23、24、25、29戰；這些統計是覆蓋範圍，不是重製完成百分比。

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
`postbattle_ch22_persist→town_ch23`、indexed renderer、一般玩家 DOSBox E2
與戰後城鎮／商店／整備存檔流程仍保持失敗即關閉。

## 2026-08-09 raw ch22 post `0x2189a` 索引呈現原語（玩家第23戰戰後；E1）

合法 IDA Pro 9.4 與 Docker Capstone 以固定雜湊的 `FD2.EXE` 閉合
`0x24754` 戰後 handler 的三個 `0x2189a` 呼叫點（`0x24978`、`0x249c4`、
`0x24a10`）。IDA 偽代碼與指令層共同證實十次外層迴圈、`work+0x8088`、
stride 456、13×8 原始場景建立、312×192 呈現區域、320 呈現 stride，以及
`0x21914`／`0x21955`／`0x2195d`／`0x21986`／`0x219a3` 的巢狀呼叫順序。
原始 push 形狀與 caller 第九參數步進來源均保留於
[`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。

重製端新增 `native_2189a_loop` 可編輯原語，將三個呼叫點與所有 raw 引數
寫入 [`ch22_post.json`](../../remake/assets/cutscenes/handlers/ch22_post.json)，
並以 compiler regression 驗證幾何常數、巢狀位址及錯誤 payload 失敗即關閉。
執行器尚未具備可證實的 indexed state／buffer adapter，遇到此原語會停止並
回報未完成；沒有把它猜成肖像、特效或一般 renderer，也沒有接到
`postbattle_ch22_persist→town_ch23`。`0x24b14`／`0x24bde` 的 persistent
record 高階意義、一般玩家 runtime frontier、DOSBox E2 與戰後城鎮／商店／
整備／存檔路徑仍是強推論或未知，維持 fail-closed。

同一輪又把 `0x247c6` 的 `cmp eax,-1`、`0x24840` 的 persistent lookup
分支，以及 `0x248b5` 的 `cmp [0x53bef],15; jl` 轉成可編輯巢狀 `if`：
`native_inventory_item_present(100)`、`native_persistent_identity_present(18)`
與 `native_round_lt(15)`。這些是原始 byte-level predicate，不是角色或物品
名稱；compiler 與 BeatRunner regression 只在完整 raw inventory／persistent
record provenance 存在時選臂，缺資料便停止。這改善了 handler 的控制流可編輯性，
但不解除 indexed renderer、`postbattle_ch22_persist` 或戰間節點 gate。

## 2026-08-09 raw ch24 post `0x24df2` 函式邊界與 owner 勘誤（E1）

固定版 `FD2.EXE` 的 IDA Pro 9.4 與 Docker Capstone 共同固定 handler table
index24 的 raw entry `0x14df2`（線性位址 `0x24df2`），以及相鄰 index25 的
`0x14e80`（線性位址 `0x24e80`）。完整輸入雜湊、檔案／線性位址與逐項指令見
[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。table index 只代表
raw dispatch，不直接代表玩家戰次。

`sub_24DF2` 的已證實順序為：FDTXT_025 index6 → `0x135dd` raw PAN
`(4,16)` → `0x10b4e(2)` → ACTING resource75 → FDTXT_025 index7 →
`0x112a5(0x1a)` persistent append → `0x11506` → push raw `0x1d` 跳入共享
`0x237c8` 尾段。共享尾段另執行文字 index3、`0x11506`、`0x112a5(0x0e)`，再
跳至 `0x231f2`。`0x112a5` 的 append 形狀已證實，固定角色表也已辨識
`0x1a`（26）為聖寇拉斯、`0x0e`（14）為珊；但兩次 append 是否代表永久
JOIN、臨時演出或其他隊伍操作，以及 `0x1d` 的章節／分支語意，仍未知，不能
由名稱或 table index 猜測升格。
補充追查已證實 `0x10b4e(2)` 會以 FDFIELD row `+0x15` 比對 group，逐筆呼叫
`0x10c50` 建立 runtime record；map24 resource 073 的 70 筆列中 group2 恰有
1 筆，故候選 binding 的 `spawn_groups["2"] = 1` 是可重生的列數投影。這只
閉合增援 materializer 的靜態列數，不替已辨識的角色索引 `0x1a/0x0e` 賦予 JOIN 身分，
也不解除一般玩家與戰間節點的 E2 gate。

現有 map24、FDTXT_025 與 ACTING resource75 的交叉證據支持「`0x24df2` 是玩家
第25戰 post handler 候選」；先前同號接到 `postbattle_ch24_persist → town_ch25`
已撤回。重製端仍維持 `postbattle_ch24_persist` fail-closed，尚未把
`postbattle_ch25_persist → town_ch26` 當成已驗證正式接線。缺少未修改一般玩家
路徑、持續隊伍與戰後城鎮／商店／整備／存檔的 E2 時，這一節只屬靜態 E1 證據。

現行 `ch24_post.json` 只保留 `0x112a5` 的兩個 raw append 呼叫（immediate
`0x1a` 與 `0x0e`），不再把它們改名成 JOIN 26／29；對應 binding 仍是候選資料，
`postbattle_ch25_persist` 已撤回正式 handler binding，編譯器會對兩個未知操作
產生問題並停止執行。

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
候選」；不再把 index29 的 terminal self-loop 當成第29戰。但 `0x35bba`、
`0x12cea`、`0x22253`、`0x24b4d`、`0x35e5a` 的完整 indexed renderer／工作區
擁有者尚未閉合，也沒有未修改一般玩家的 E2，因此不能把現有
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
