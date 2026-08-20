# 58 — `FD2.EXE` 反組譯覆蓋與重製閉合矩陣

> 更新基準：2026-08-20 工作樹。這是判斷「還要不要反組譯」與「重製還缺哪一層」
> 的唯一現況入口；它不取代位址證據、系統設計、介面矩陣或歷史交接。
>
> 原版基準：`FD2.EXE`，357074 位元組，MD5
> `b97caf2239a27a896069d03549d96e1e`，SHA-256
> `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
> 雜湊不同時，本頁所有位址都只能當特徵線索，不能直接沿用。

## 一、先說結論

目前**不能誠實宣稱整支 `FD2.EXE` 已完成多少百分比**，也不能用文件數、測試數或
匯出器的 `unknown_ops` 數量代替。原因如下：

1. 現已有 IDA Pro 9.4 重生的 1,305 筆函式清冊；Watcom FLIRT 分出170筆
   runtime，受版控語意索引分出31筆產品函式，其餘1,104筆仍未
   分類；尚未排完 DOS/4GW、Miles AIL 與所有一般函式庫，因此仍沒有可信的
   「重製相關函式」分母。清冊見
   [`fd2_function_inventory.json`](../data/ida/fd2_function_inventory.json)。
2. 目前反組譯以玩家可見功能為目標，並不需要逐一重寫編譯器函式庫或驅動程式。
   「整支 EXE 每個函式都命名」不是重製完成條件。
3. `chapter_beats` 現將呼叫拆成已分級 `native_call`、已知 callee 但未閉合
   caller/runtime 的 `unresolved_native_call`，以及真正 `unknown`；任何單一計數
   都不能直接視為重製完成度。
4. 原版語意已解、可編輯資料已建、正式執行期已消費、未修改一般玩家路徑已驗證，
   是四個不同閘門。過去最常見的誤判，就是把第一個閘門完成寫成整個功能完成。

以重製需要的玩家可見子系統來看，**資產格式與許多底層原語的反組譯覆蓋已高；
戰役處理器、敵方人工智慧、指令／法術／物品與終局仍是部分閉合；正式執行期與
一般玩家第二級證據（E2）明顯落後於靜態反組譯。** 因此後續預設不再全面重做
反組譯，而是只補下表中明確缺少的欄位。

## 二、完成度的固定語言

每個主題都分開記錄五個欄位，不再使用單一 `[x]` 表示「完成」。

| 欄位 | 問題 | 可標示的狀態 |
|---|---|---|
| 原版證據 | 位址、呼叫者、寫入端、消費端與控制流是否閉合？ | 閉合／部分／未知／不適用 |
| 可編輯資料 | 原版硬編碼是否已轉成具型別 JSON、腳本或規則？ | 就緒／部分／缺少／不適用 |
| 正式執行期 | 正式 campaign／battle／UI 路徑是否真的消費？ | E1／部分／未接／失敗即關閉 |
| 玩家驗證 | 未修改原版與重製是否在同狀態、正常輸入路徑驗證？ | E2／部分／缺少／不適用 |
| 下一個缺口 | 下一步屬於反組譯、實作、動態原版、視覺或發行？ | 必須明列一類 |

本頁的「閉合」只適用於該格描述的問題，不可外推到整個子系統。E0／E1／E2 的
詳細證據規則仍以 [`56` 系統設計規格](56-fd2-remake-sdd.md)為準。

## 三、目前覆蓋矩陣

| 子系統 | 原版證據 | 可編輯資料 | 正式執行期 | 玩家驗證 | 目前裁決與下一步 |
|---|---|---|---|---|---|
| 檔案版本、容器與主要資產格式 | 閉合 | 就緒 | E1 | 部分 | `.DAT`、圖像、FDTXT／字型、AFM／FIGANI、XMIDI、地圖與多張 EXE 表已有雜湊與重生工具。剩餘多是消費端、音訊時序或個別執行期改寫，不應重解容器格式。 |
| 開機、標題、LOAD、CONTINUE、存檔 | 部分 | 部分 | 部分 E1 | 原版錨點部分 E2 | 四槽 envelope、checksum、名冊與部分戰間落點已接；current-runtime 有原版與重製各自錨點，但尚缺同一 raw 狀態配對、未修改有效槽完整路徑與部分控制器交接。下一步以動態原版／執行期整合為主。 |
| 對話、頭像與過場原語 | 部分偏高 | 部分就緒 | 部分 E1 | 部分 | `0x15F84`、`0x1366A`、基本 pan／acting／spawn／join 等不應再從零重解。仍須處理 caller-specific layout、indexed renderer、文字分支與實際 chapter binding。 |
| 30 個 raw chapter 的戰前／戰後處理器 | 部分 | 60 份 handler script；部分 binding | 部分 E1 | 缺完整 E2 | 舊83個 raw unknown 已拆成79個已證實窄呼叫、4個已知但 caller／執行期未閉合的呼叫，已沒有未分類 call site。`0x24336` 與玩家第25戰 raw ch24 post 已達 E1，但三個晚期戰後節點及多個 caller-specific 畫面仍未完成；仍須逐章追蹤「handler→戰鬥→戰後→城鎮／整備→存檔」。 |
| 可編輯戰役與持續隊伍 | 部分 | 121 個 story／cutscene 節點；9 個 scripted、53 個 handler-bound、59 個 fallback | 部分 E1 | 缺完整 E2 | 24 個 postbattle 節點中目前21 active、3 blocked（玩家第23、24、29戰戰後）。玩家第25戰已以62→70→71原始槽位拓撲接到`town_ch26`並通過JOIN26／29與存讀檔 E1；仍缺一般玩家 E2。近似模式只維持其餘可玩戰間，不提升忠實度。 |
| 戰鬥資料、移動、公式、勝敗與成長 | 部分偏高 | 部分就緒 | E1 | 部分 | 多項公式與地形資料已有具型別實作；命中／閃避來源、部分經驗交易、回合事件與原版逐狀態驗證仍不完整。需要針對缺欄位補 producer／consumer，不重解已閉合的 AP−DP 等公式。 |
| 玩家指令、法術、物品與交易 | 部分 | 部分 | 部分 E1／部分失敗即關閉 | 缺完整 E2 | command mask、若干 ID、MP／物品交易與 selector 邊界已解；未知 command、複合技、法術／物品完整效果、rollback、演出與 UI 尚未閉合。這是目前真正需要局部反組譯與實作並行的區域。 |
| 敵方人工智慧 | 底層控制流部分偏高；高階交易部分 | mode／候選／部分 fallback 已資料化 | 多個窄 E1 consumer | 只有原版敵方回合邊界 E2 | `0x13FD4`、`0x14EF0`、mode 5／11 等既有函式邊界與窄 owner 不應反覆重解。缺的是完整 target／command／spell／item transaction、同一 raw 狀態的重製端配對與一般玩家效果；詳見 [`11`](11-enemy-ai.md)。 |
| 戰場 HUD、指令格、輸入與戰鬥演出 | 部分 | 部分 | 部分 E1 | 少量畫面 E2／多數缺少 | 有 native frame、command overlay、姓名字模、命中色盤與部分 FIGANI consumer；整體操作狀態機、圖示可用性、相同戰況及演出時序仍未完成。完成度只由 [`57` 介面矩陣](57-ui-evidence-matrix.md)判定。 |
| 城鎮、祕密商店、商店、教會與整備 | 部分偏高 | 部分就緒 | 多個正式 E1 consumer | ch02 若干狀態 E2；其餘部分 | 個別 menu、購買、轉移、復活、轉職與整備已有窄切片。第25戰正式勝利、持續隊伍與存讀檔後，`town_ch26` 已以章節專屬 selection4＋Shift+F5（BIOS `0x58`）揭露selection5、進入variant5、驗證三項商品並經四幀離店返回；錯用ch02的`0x54`會拒絕。這是正常戰間E1，不是未修改原版E2；其餘章節入口、native save、recipient scroll與完整交易仍缺。 |
| 音樂與音效 | 格式閉合；owner／時序部分 | 部分就緒 | 部分 E1 | 逐音訊 E2 缺少 | XMIDI、兩類音源與部分曲目／樣本 owner 已知。重點是精確播放時機、停止／切換、效果同步與三平台播放，不需要重解 XMIDI 格式。 |
| 終局與結局 | 部分 | 近似排程已資料化 | 近似 E1；忠實路徑部分失敗即關閉 | 缺一般玩家終局 E2 | `0x2BCE5` 前綴、部分 montage／音訊／資源尾段已有證據；`0x28A6C` 是戰鬥、事件與終局共用的雙 record renderer，缺的是終局 caller `0x2C2A6` 當下 records／globals、精確輸出、終端輸入與正式 handoff，不是重解整個 callee。 |
| DOS/4GW、Watcom runtime、Miles 驅動與一般函式庫 | 第一輪分類 | 不適用 | 只在行為外露時處理 | 不適用 | IDA 清冊1305函式中170筆由 Watcom FLIRT 標成 runtime；其餘未分類不能都算產品程式。後續只擴充分級索引，不把函式庫未命名算成 remake 缺口。 |
| 三平台打包與推廣片 | 不適用 | 部分 | 尚未達發行閘門 | 缺完整玩家驗收 | 這不是反組譯問題。待核心戰役、操作 UI、結局與代表性一般玩家路徑關閉後，再做 Linux／Windows／macOS 打包與影片。 |

## 四、目前可重生的數字，以及不能怎麼解讀

2026-08-13 以唯讀 Docker 實跑現有稽核：

- IDA Pro 9.4 對固定雜湊 `FD2.EXE` 辨識1,305個函式；受版控語意索引32筆，
  分類結果為產品31、runtime170、未知1,104。`0x37416` 同時由 FLIRT 命名為
  `free` 並由索引標成 runtime；沒有因此重複計數。
- 60份 raw handler script 原有83個 `unknown` call site、23個 target。重生後為
  79個 `native_call`（21個 target）與4個 `unresolved_native_call`
  （`0x22253`、`0x2BCE5` 各2次），真正 `unknown` 為0。每筆
  具名呼叫皆保存原始位址／PUSH 順序、推論等級與證據檔；編譯器仍逐 caller
  驗證並失敗即關閉，故這是**工具債清除**，不是玩法完成率。
- `campaign_full.json`：121 個 story／cutscene 節點；9 個 scripted、53 個
  handler-bound、59 個 fallback。fallback 可能是撤退、傳聞或尚未接線故事，不能全部
  視為同一種缺口。
- postbattle audit：24 個節點，21 active、3 blocked；mapping gap 為 act 6、dialog 6、
  layout 1、pan 3，已無未分類 native semantics。active 也只表示 admission gate 可通過，不代表
  該章一般玩家 E2 或逐像素一致。

上述數字只能用來定位工作，**禁止相加或換算成遊戲完成百分比**。
重算時只更新本節與上方矩陣；README、SDD、介面矩陣與工作清單只連回本頁，不再
各自保存一份會漂移的「目前總數」。舊日期條目若保留當時數字，必須明寫為歷史快照。

## 五、已知位址的「不要重做」索引

下表指定目前主證據與可重開條件。舊交接、SDD 附錄或 exporter 仍寫 `unknown`，都不能
單獨成為重做理由。

| 位址／家族 | 現有主證據 | 已閉合範圍 | 仍可做的工作 |
|---|---|---|---|
| `0x15F84` | [`29`](29-remake-extensible-event-system.md)、各 handler 直接指令 | 對話呼叫與索引基礎 | caller-specific 分頁、版面與 E2；不重解基本函式角色 |
| `0x1366A` | [`50`](50-cutscene-script-system-design.md)、`chapter_beats` | acting 呼叫原語 | 個別資源、場景時序與畫面；不重解原語本身 |
| `0x11DF2` | [`fd2_11df2_palette_disasm.txt`](../data/fd2_11df2_palette_disasm.txt) | palette range/delta helper | caller 時序與畫面；應回填 exporter |
| `0x1F882`、`0x25052` | [`91`](91-worklist.md) 對應直接指令證據 | 兩種不同 palette ramp | runtime renderer／caller E2；不再稱 vsync |
| `0x13FD4` | [`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt) | raw gate、回復與窄 presentation owner | 同狀態交易／逐幀逐音訊 E2；不重解函式邊界 |
| `0x14EF0` | [`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt) | producer 順序與尾端 dispatch | 未知 command／效果／完整 transaction；不重解既有 dispatch |
| `0x14237` | [`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt) | 物理候選評分窄切片 | 完整 planner、target transaction 與 E2 |
| `0x15311`／`0x1548E` | [`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt) | mode 11 兩段 owner／順序 | 未知 command、完整演出與 E2 |
| `0x24618` | chapter-specific IDA 證據；例如 [`ch22`](../data/ida/fd2_ch22_pre_ida.txt)、[`ch27/28`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt) | indexed transition 核心與部分 caller payload | 新 caller 必須另證參數／view；不得把已知 callee 當全新未知 |
| `0x22253` | [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt) | 共用11＋6＋10排程 | caller payload、renderer 組合、資產與同狀態 E2；維持 `unresolved_native_call` |
| `0x24B14` | ch26 post 直接證據與 [`91`](91-worklist.md) | 天空之鑰 inventory gate | 兩臂視覺／效果；不重解搜尋條件 |
| `0x24336` | [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt) | 玩家第21戰戰後天空之鑰固定演出：`FDOTHER #34`、`ANI #0`、調色盤與延遲順序；正式重製端 E1 已消費 | 補未修改原版同狀態 E2、第一個動態調色盤相位及相鄰 `layout_units`／ACT63／64；不重解函式本體 |
| `0x135DD`、`0x20421`、`0x4DFCC` | 同上及既有 palette／AFM 證據 | `0x24336` 使用的鏡頭移動、全螢幕 AFM 與高色階相位循環窄角色 | 新 caller 另證參數與時序；不可把 `0x20421` 誤稱音訊或把 `0x4DFCC` 推成一般調色盤 API |
| `0x2BCE5` | [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)、[`montage tail`](../data/ida/fd2_ch29_post_montage_tail_ida.txt) | 終局前綴與部分尾段 | 正式 owner、完整 handoff、一般玩家 E2 |
| `0x28A6C` | [`35`](35-battle-animation-rendering.md)、[`montage tail`](../data/ida/fd2_ch29_post_montage_tail_ida.txt) | 共用雙 runtime-record renderer；直接 caller 含戰鬥、事件與終局 | 只補終局 `0x2C2A6` call-time records／globals、精確輸出與輸入；不得再稱終局專屬 callee |

只有四種情況可重開「已閉合範圍」：原版雜湊不同、原始指令或跳表直接反證、執行期
同狀態結果矛盾，或現有證據缺少它聲稱已具有的 writer／consumer。單純找不到舊筆記、
舊文件仍寫未知、IDA 自訂名稱不同、或 exporter 沒更新，都不是重開理由。

## 六、後續工作必須先分類

### 真正需要局部反組譯

- 依玩家可見 blocker 擴充既有 IDA 清冊的產品／runtime／driver 分類；不為了提高
  百分比替剩餘1,104筆未知函式猜名稱。
- 三個 blocked postbattle 節點的實際未知 branch／native semantics；已知 helper 只補
  caller payload，不重讀 callee。
- 未知 command／spell／item transaction 與高階效果 owner。
- `0x28A6C` 精確 renderer、終端輸入、`0x2BCE5` 正式 owner／handoff。
- global event table 58..89、event82 producer，以及其他會阻擋一般玩家流程的 handler。

### 不需要再反組譯，應轉實作或工具修正

- 維護 `native_call`／`unresolved_native_call`／`unknown` 三態與分級證據；目前
  handler 匯出已沒有真正 `unknown` call site，剩餘4筆只追 caller／執行期 gate，
  不重解已知 helper。
- 把已知資料接進正式 UI、campaign、save、audio consumer，補原子失敗與 regression。
- 依 [`57`](57-ui-evidence-matrix.md)完成戰場、指令、城鎮、教會、整備的輸入狀態機與畫面。
- 讓完整戰役保留戰後城鎮／商店／整備，不用直接跳下一戰的測試捷徑。

### 需要原版動態驗證，而不是更多靜態反組譯

- 同一 raw save／章節／回合下的原版與重製敵方回合配對。
- 晚期三個 blocked postbattle、其餘章節祕密商店的一般玩家 E2、有效槽 LOAD
  與跨章 save/load。
- 第30戰勝利、結局動畫、終端輸入與定格的一般玩家路徑。
- 戰場 HUD、command grid、法術／物品與命中演出的同狀態影像／音訊時序。

## 七、文件責任與更新規則

- 本頁只回答整體覆蓋、主證據與下一個缺欄位。
- [`56`](56-fd2-remake-sdd.md)只保存系統契約、證據規則與精確資料／執行期設計。
- [`57`](57-ui-evidence-matrix.md)是 UI 與玩家可見差距的唯一狀態表。
- [`91`](91-worklist.md)只在檔首列有效佇列；其後舊勾選是歷史工作記錄。
- [`SESSION-HANDOFF`](SESSION-HANDOFF-2026-07-06.md)只保存時間序列與勘誤，不能決定現況。
- `docs/data/ida/` 與 `docs/data/fd2_*` 是位址主證據；`chapter_beats`、binding 與測試是
  產物或消費端，不得冒充原始二進位證據。

關閉一項工作時，先更新本頁相應欄位與 canonical evidence，再更新專題文件、UI 矩陣
或工作佇列。若只新增一份散文筆記、卻沒有更新本頁，該工作不得被視為新的完成進度。
