# 58 — `FD2.EXE` 反組譯覆蓋與重製閉合矩陣

> **第一輪停止條件（2026-08-27）**：完整反編譯不是第一輪 remake 的交付門檻。
> 已有最小充分證據並形成正式玩家可見消費端的位址不重做；剩餘 unknown 只有在
> 會造成95%代表性抽樣失敗、破壞核心資料或違反失敗即關閉時才成為阻擋。原版
> oracle 未知、逐週期硬體差異及無正常producer的分支留作證據限制或可選考古，
> 不得僅因覆蓋率不足重開。抽樣完成定義見[`REMAKE-STATUS.md`](../REMAKE-STATUS.md)。
> **2026-08-28 交付閘門**：Docker 清冊檢查為60／60、五層最低配額全達成、
> `first_round_complete=true`、0 integrity errors。這關閉第一輪代表性抽樣，
> 不改寫下方原版 RE 證據分級，也不把未知函式或未達 E2 的畫面升格為已證實。

> 更新基準：2026-08-27 工作樹。這是判斷「還要不要反組譯」與「重製還缺哪一層」
> 的唯一現況入口；它不取代位址證據、系統設計、介面矩陣或歷史交接。
>
> 原版基準：`FD2.EXE`，357074 位元組，MD5
> `b97caf2239a27a896069d03549d96e1e`，SHA-256
> `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
> 雜湊不同時，本頁所有位址都只能當特徵線索，不能直接沿用。

## 一、先說結論

> **第29戰產品裁決（2026-08-26）：** 依專案採用的玩家可見 **99% 相似門檻**，
> 第29戰已列為 **remake 已完成**：正式 `CONTINUE` 入口、76-slot 戰況、玩家／敵方
> 回合交接、戰果、raw ch28 post、持續隊伍、19人整備／冷讀檔與第30戰接縫均已有
> `RUNTIME-E1`，並有未修改原版的單次回合候選錨點。第三方存檔缺少「第29戰勝利
> 當下由原版 writer 建槽」的完整來源，以及逐幀／精確音訊 `PLAYER-E2`，只保留為
> **證據限制與可選 polish**，不再阻擋 remake，也不得據此重開已閉合 RE。

目前**不能誠實宣稱整支 `FD2.EXE` 已完成多少百分比**，也不能用文件數、測試數或
匯出器的 `unknown_ops` 數量代替。原因如下：

1. 現已有 IDA Pro 9.4 重生的 1,305 筆函式清冊；Watcom FLIRT 與受版控
   runtime 註記合計分出175筆 runtime，受版控語意索引分出61筆產品函式，
   其餘1,069筆仍未
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

### 1,073 筆未知函式的交付影響

2026-08-27 已由授權 IDA Pro 9.4 從固定雜湊原檔重新分析。第一次重生基準為
`1,305 = 37 product + 170 runtime + 1,098 unknown`、語意註記38筆；同日再將
全專案 Markdown／文字證據與1,098個 unknown 範圍交叉比對，發現508個函式起點
曾被精確提及、524個函式範圍內至少有一筆位址足跡。這只證明專案曾留下線索，
不是語意證明；raw 匯出、呼叫位址或歷史猜測都不能單獨解除 unknown。

本輪只選 caller、consumer、raw writer／控制流與受版控證據均已閉合的十筆窄
語意回填，其中九筆是產品函式，`0x375B2` 是轉跳至 `0x3DCCD` 的 delay thunk。
授權 IDA Pro 9.4 由固定雜湊原檔再次從零匯出後，現行清冊為
`1,305 = 46 product + 171 runtime + 1,088 unknown`，語意註記48筆。新增分類
保留 `sub_...`／`j___delay` 原名、線性位址、直接 caller 與證據等級；未達此門檻
的其餘足跡仍維持 unknown。`1,089` 是舊數字次序誤寫，不是另一版清冊；
`1,104` 則是更早語意索引尚未重生前的歷史數字。

使用者要求先判讀「研究過但尚未登錄」後，現已加入可重跑的
[`fd2_unknown_footprints.json`](../data/ida/fd2_unknown_footprints.json)與產生器
`tools/audit_fd2_unknown_footprints.py`。在前批十筆移出 unknown 後的1,088筆中，
498筆有精確起點足跡、514筆有 range 足跡；其中337筆同時命中現況知識文件與
直接證據產物。人工複核第一批337筆後，再接受十二筆窄產品語意：FDFIELD record
materializer、FDICON cache、玩家戰場 controller、兩個 AIL sample wrappers、
戰後戰間 gate、兩個 raw/event constructors、輸入清理、程序 RNG 與上下框 portrait
blitters。授權 IDA 9.4 再次從固定雜湊原檔重生後，現況為
`1,305 = 58 product + 171 runtime + 1,076 unknown`、語意註記60筆；剩餘足跡為
486個精確起點、502個 range，第一優先人工審查降為325筆。

`0x36CD7` 起初因高命中而證據產物未閉合被保守保留；後續應使用者追問，以 IDA
直接核對其16-byte body、唯一下一層與失敗 consumer，現已證實
`0x36CD7→0x36CEA→0x36D07` 是 Watcom stack-demand／overflow-check runtime：
比較 prospective ESP、process stack lower limit 與 SS selector，失敗時輸出
`Stack Overflow!` 並以1退出。它不是舊文件所稱的逐頁 guard-page probe，也沒有
遊戲 side effect。三段回填後當時為
`1,305 = 58 product + 174 runtime + 1,073 unknown`、語意註記63筆；541個
prologue call sites 可從產品原語統計排除，但 caller 後續真正 frame 與產品 calls
仍須分析。通用 pattern 見[`59-watcom-stack-runtime-patterns.md`](59-watcom-stack-runtime-patterns.md)。

這也說明目前可以打包驗證而不必先命名整支 EXE：封包驗證檢查的是受版控資產、
可編輯資料、正式 Go／Ebiten 執行期、存讀檔與代表性玩家垂直切片能否形成自洽
產物；它不會執行原版 EXE，也不會自動證明未分類 helper 的語意。只要某個玩家
切片所依賴的原版資料、規則、介面與狀態邊界已有證據及失敗即關閉行為，該切片
便能建置與抽驗。反之，能產生三平台封包只證明交付物完整性，不能提升其餘
1,069筆 unknown，也不能取代未修改原版同狀態的 `PLAYER-E2`。

同日再依工具鏈指紋分流最高 fan-in 候選：`sub_3EEDA` 雖有160個直接 code xrefs，
本體只回傳 `dword_52BE6`；該全域由 Miles IRQ 8 timer 初始化 owner 清零，於中斷
handler 進入／離開時加減，AIL shutdown 等 caller 以非零結果避開背景中斷內的
診斷／檔案操作。故窄語意「回傳 Miles AIL timer IRQ 背景中斷巢狀狀態」已閉合，
精確 API 名稱 `AIL_background` 仍只列強推論。現況為
`1,305 = 58 product + 175 runtime + 1,072 unknown`、語意註記64筆；這是回填
handler 三筆 callee 前的歷史狀態，證據見
[`fd2_ail_background_3eeda_ida.txt`](../data/ida/fd2_ail_background_3eeda_ida.txt)。
這個案例建立後續固定順序：先考證 compiler／linker／extender／middleware，再以
writer／consumer 分類高 fan-in helper，不用 unknown 總數驅動無限 RE。

`unknown` 只表示目前沒有受版控證據足以分類，不能批次改名成 DOS、PIT、DAC、
驅動或遊戲邏輯。第一輪重製採下列交付分類，而不是繼續追求命名率：

- 已定位為 DOS BIOS、DOS/4GW、PIT、DAC、DMA、Miles AIL 或硬體忙等的時序，依
  公開規格與成熟模擬器契約實作可重現近似；只保存遊戲選用的 cue、參數、順序與
  播放完成閘門，不把逐週期一致列為阻擋。
- 只影響裝飾性抖動、嘴型、粒子、終局未初始化殘留或其他不改變玩法結果的亂數，
  可使用隔離且可重現的重製端亂數；不得冒稱原版 RNG parity，也不阻擋第一輪。
- 命中、傷害、暴擊、成長、AI 目標／同分裁決、法術／物品效果與掉落等會改變
  玩家結果的亂數仍是玩法契約，不能因為同樣叫 RNG 就略過；抽樣矛盾時才局部追查
  writer、consumer 與狀態提交邊界。
- 其餘未知函式預設為「尚未分類、非交付阻擋」。只有正常玩家抽樣顯示它會改變
  玩家結果、破壞資料，或使失敗即關閉無法維持時，才升級成局部 RE 工作。

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
| 檔案版本、容器與主要資產格式 | 閉合 | 部分 | 部分 E1 | 部分 | `.DAT`、圖像、FDTXT／字型、AFM／FIGANI、XMIDI、地圖與多張 EXE 表已有雜湊與重生工具，不應重解容器格式。2026-08-28 全量匯出實測得到1,005個raw subresources、125張一般PNG、264／409組FIGANI、136組頭像與33張地圖。FIGANI有2,118張indexed frame、2,118張mask、264份animation metadata及409份resource status；另將23筆已證實的FDOTHER內嵌動畫輸出408張frame、408張mask、23份metadata與23份status。BG／TAI各57筆均已建立狀態文件，#0..55共112張indexed frame與112張mask，兩個0-byte #56維持blocked。FDTXT另有34份lossless token JSON＋1份zero-byte blocked，FDOTHER#4輸出1,824 glyph atlas／metadata；FDICON完整輸出1,680張indexed frame、1,680張source mask、1,680張remap mask及metadata。逐檔manifest現為13,048筆（12,024 exported、1,005 intentionally_raw、19 blocked）。112筆surface已逐pixel／mask／final blit與archive decoder一致；FDTXT#31逐word、字型逐bit、FDICON全1,680張三層一致。戰鬥指令0–3／5–9／24／29／32–35、party montage的TAI#3透明gate及終局20段tail均只讀分離surface；LOAD、職業／教會共用資產、共用道具panel與party montage亦只讀分離FDTXT／字型；戰場、整備、職業／教會、城鎮與終局tail baseline只讀分離FDICON。第23戰重載的FDFIELD#69已由`map23/map.json`完整四byte組合格逐位元組重建，原封存檔不可讀時仍可staging。正式FDTXT、FDICON、`figani.DecodeResource`與TAI direct archive production caller歸零，達各自窄`RUNTIME-E1`。production runtime仍直接讀其他`.DAT`；音效、AFM／ANI、終局FDFIELD#90..92、FDSHAP與其他UI用途未全量閉合，故整體仍是部分。契約與實測清冊見[`60`](60-editor-separated-assets-spec.md)及[`asset-export-audit-20260828.json`](../data/asset-export-audit-20260828.json)。 |

> **2026-08-26 晚期四槽 LOAD 勘誤：** 同一固定雜湊 `fd2last.sav` 現已由重製
> 正式 selector 還原 slot 0 的29人、60金幣與 raw metadata，並進
> `preparation_ch30`。聚焦回歸先證明舊 `raw+1` 映射會誤進
> `preparation_ch29`；原始 `0x526B9` 表保持不變，玩家可見例外改由具證據來源的
> 可編輯覆寫保存。後續同檔已由正式標題事件擁有者走完 LOAD、19次自動前進選取、
> 最終確認與 ch29 pre，20名部署者由 persistent records 物化後抵達 `battle_ch30`；
> authored scenario 漏列 identity 3 不再造成 mismatch。此批達 `RUNTIME-E1`，第三方
> 來源仍不升完整 `PLAYER-E2`。同一正式長鏈現又由 END／YES 進入未刪減的第30戰
> 敵軍回合並交還玩家控制；測試揭露 persistent record 的 raw `+0x34/+0x35/+0x36`
> 曾在 typed party 邊界遺失，現已從固定槽保留到 AI scorer。沒有新增或猜測高階語意。
| 開機、標題、LOAD、CONTINUE、存檔 | 部分 | 部分 | E1 | 原版錨點部分 E2 | 穩定標題選單由目前原始碼重新擷取後，320×200與原版 oracle 達整幀 `AE=0/64000`。`sub_1F894` 的535→0、每列30ms、`450/330/210/110/25/10`插播與返回同一捲動視窗已由 IDA 9.4 canonical 證據 [`fd2_title_scroll_schedule_ida.txt`](../data/ida/fd2_title_scroll_schedule_ida.txt) `RE-CLOSED`，正式 runtime 也已改成同一交錯排程。2026-08-27再由相同canonical caller接入AFM 3前的`FDOTHER #74 + #76`漢堂發行商畫面：8＋103＋8幀近似，runtime第60幀與indexed oracle達640×400 `AE=0/256000`，新增後仍自然抵達AE0選單。舊DOSBox `frame_000`不是發行商標誌，故本幕只到`RUNTIME-E1`。先前誤稱wipe／logo揭示的後段現由IDA直接指令閉合為兩次`sub_286BD`索引色盤內插，正式runtime依半開區間0..254播放紅幕→真實ANI #1→近白標題；相近原版淡入影格MAE為0.077／255，高於99%玩家可見門檻。剩餘標題缺口限於精確音訊與原版runtime E2。四槽 envelope、checksum、名冊與部分戰間落點已接；標題 selector 的正式確認 owner 現以 checksum-valid 合成槽完整還原 `town_ch02`、typed/raw party、gold、chapter、HUD gate並清除舊 battle state，竄改 envelope 原子留在選槽。current-runtime CONTINUE 與巢狀 SAVE／LOAD 已有正式 E1：LOAD 先完成私有候選 handoff，YES 後才原子替換；SAVE 保留四槽與未命名 bytes，非我方增援不占 persistent slot，具 constructor／identity 證據的新我方 JOIN 追加完整 raw record。重製 JSON LOAD 現另在發布前驗證 JOIN 順序、membership、部署與 materialized roster 的拓撲一致性；錯誤存檔不會部分改寫現行遊戲。新 `Game` 冷讀 `preparation_ch30` 後可保留完整加入順序並繼續終局。外部2003年第30戰候選已由未修改原版普通 `CONTINUE` 進場，重製亦以同檔還原33筆場上單位與31人持續隊伍；fixed-hash `fd2004` 候選也由普通 CONTINUE 進玩家第29戰並完成一輪控制權交接。另有 checksum-valid `fd2021` raw chapter `0x1c` 槽由未修改原版普通 LOAD 連續走過 save-NO、19人整備、戰前劇情至可操作第30戰。三者均非本專案從頭產生，只列候選E2。第29戰勝利當下由原版 writer 建槽的完整來源、delete／overwrite及跨章同狀態差分只列證據限制，不再阻擋99%相似門檻；長程遊玩改由使用者人工回報，不列代理工作項目。 |
| 對話、頭像與過場原語 | 部分偏高 | ch00_pre 97句、ch00_post 13句、ch01_pre 20句、ch05_post 19句、ch06_post 12句、ch07_post 8句、ch09_post 35句、ch12_post 12句、ch15_post 23句、ch16_post 26句、ch17_post 21句、ch19_post 29句、ch20_post 30句、ch21_post 11句、ch22_post 56句、ch23_post 11句、ch24_post 18句、ch25_post 41句與ch27_post 5句原生版面就緒 | 部分 E1 | 部分 | 十九個已接切片均由原始控制碼建成具型別頁面。136組DATO頭像已重生為544張indexed PNG；resource26四幀與archive逐indexed-pixel一致，archive不可讀仍通過。同一分離loader已接故事對話、暫態、group march、整備、教會轉職主介面、商店、物品／狀態面板、終局preview與montage；production直接DATO archive caller由10處降至0處。第21戰後六組分支呼叫保存30句逐句版面；成功臂正式消費26句、ACT63／64、天空之鑰動畫、同步、`town_ch22`與存讀檔。第26戰後兩分支分別消費18／33句並進`town_ch27`；第28戰後五句則進`preparation_ch29`。第24戰後兩個呼叫端以正式輸入播放11句並保持86-slot跨場景長鏈。只重開E2、其他archive家族consumer或其他呼叫端綁定，不重做已閉合renderer。 |
| 30 個 raw chapter 的戰前／戰後處理器 | 部分 | 60 份 handler script；部分 binding | 部分 E1 | 缺完整 E2 | 舊83個 raw unknown 已重生為93個已證實窄呼叫、0 unresolved、0 unknown；數量增加是補登既有 callee 後的分類展開，不是新增呼叫位置。玩家第29戰 raw ch28 post 現已以綁定的視圖／HUD、`0x35BBA→0x1DB65`、group9、`0x22253`、`0x24B4D`、`0x35E5A`、隊伍同步與 `preparation_ch30` 存讀檔達成 E1；最終戰前 raw ch29 pre 亦已綁定 `LOADCH`、21句對話與七次 `0x33F78` 原生 staging。未證實高階圖像／樣本名稱與一般玩家 E2 仍保留。 |
| 可編輯戰役與持續隊伍 | 部分 | 121 個 story／cutscene 節點；9 個 scripted、57 個 handler-bound、55 個 fallback | 部分 E1 | 缺完整 E2 | 24 個 postbattle 節點目前全部 active；admission blocked 為0。玩家第29戰正常 `story_ch29→battle_ch29` 入口現物化76-slot frontier與已證實視圖／HUD，戰果確認後播放 raw ch28 post、追加group9、同步持續隊伍，再進`preparation_ch30`並通過存讀檔。正式連續回歸現由全新 `Game` 冷讀該整備存檔，再走19人選擇、`story_ch30` 的21句對話／七次 staging，並以 ch29_pre 最後 focus 物化 `battle_ch30` view／HUD；party→group0 的 handler rows 經 raw origin 扣除後補 groups1–3，所有33筆皆具 indexed selector／presentation。回歸再消費完整 END→YES 介面、敵方回合、勝利、終局文字閘門與角色蒙太奇。最終戰部署成員保留戰後更新，未部署成員保留冷讀狀態，終局回顧依完整 JOIN 時序涵蓋全隊。外部固定雜湊候選於未修改原版第29、30戰各自完成單次 `CONTINUE→END→YES→ENEMY PHASE→玩家控制`；重製也以同一第29戰候選從正式標題事件完成 END／YES、實際敵軍行動與回合交還，保留76筆 runtime／31人 persistent。晚期有效槽另由普通 LOAD 走過19人整備與戰前劇情至第30戰控制權。這些仍受第三方來源與停用音訊限制，也尚未證明原版第29戰勝利、writer建槽至第30戰的同一連續程序。 |
| 戰鬥資料、移動、公式、勝敗與成長 | 部分偏高 | 部分就緒 | E1 | 部分 | 多項公式與地形資料已有具型別實作；命中／閃避來源、部分經驗交易、回合事件與原版逐狀態驗證仍不完整。需要針對缺欄位補 producer／consumer，不重解已閉合的 AP−DP 等公式。 |
| 玩家指令、法術、物品與交易 | 部分 | 部分 | 部分 E1／部分失敗即關閉 | 缺完整 E2 | command mask、若干 ID、MP／物品交易與 selector 邊界已解；物品第一階段 raw target field 及 drawable selector 1..5 現由正式 `0x11CAC` 索引畫面消費，缺原生 HUD／LUT／range sprite 時不再顯示綠／青／橘色猜測後備層。未修改第一戰存檔另以正常輸入證實四名我方皆滿 HP 時，item 192 草藥確認後至少4.3秒仍留在物品面板；type 5 現只在已證實候選中存在 `HP < MaxHP` 時發布 target modal，形成窄原版 `PLAYER-E2`／重製 `RUNTIME-E1`。這不包含受傷狀態原版 target modal、其他 item type、indexed item effect、disabled target 外觀、原版取消鍵或 global selector 6 owner。共用 `0x117E7` 在 `0x12C0D==-1` 時進 `0x16F55`。direction3→END 現以原版 DATO #75、FDOTHER #5/#2、FDTXT `0x1A3/0x1A4/0x19C` 跑完6＋4展開、YES／NO、4＋5收合、接受／取消逐字形回覆、來源復原與十二個60 Hz畫格近似，只有YES才進`0x1A30B`；普通鍵盤正式路徑與原版密集擷取已形成動態配對，精確時序／音訊仍未閉合。缺任一資產時在命令框關閉前拒絕。command 13–16 的 `0x21EB1→0x22046` 16張 FDOTHER #3 LUT 演出已轉成 typed schedule，玩家與敵方 mode 11 正式入口均先演出再交易；AI 依 `0x15311` 在移動後重建 raw target array。command 17–19 的 raw modifier transaction已由玩家與敵方mode 11正式消費；ID17依原版由record18扣MP。玩家command 17–22與25–27均依`0x1D6C8`先播放#80 selector0與八個commandColor／black DAC phases，再進各自handler的effect／mask／結果尾段；交易只在handler專屬Draw邊界發布。20／21才清`+0x25/+0x26`並借record10 restore，22／26／27才經class／RNG gate寫各自raw marker，25只清final target的raw `+5 bit7`且全成功時沒有空數字段。command 24 的正常 selector32 路徑現依資源98的15幀 raw schedule，在frame4發布MP並播放FDOTHER #53 sub3，在frame10發布單一完整傷害並播放sub2，最後一幀才標記行動完成；兩標記間依raw terrain control選BG，播放`0x29C90`兩段各10次的640-stride viewport滑動。`0x2A289→0x18C6D`的entry22框、HP／MP bar、數字與raw姓名亦已接進actor／target indexed base，轉場不再重疊RGBA雙panel。缺原始FIGANI、BG、terrain control、target idle、palette、panel cells／font／text、sample或raw selector即零交易。升級學習端亦已修正為`unit+7→growth byte10 learn_idx→command_learn`，不再誤用portrait直接查表。`sub_2B659` actor base、扣MP後source snapshot與`sub_2B9A1` target idle reset已接；`sub_29164`九段雙分支角色／TAI滑入與DAC減算已接。共用短音效層現保留每個疊播 player 至自然結束、每幀回收並於程式退出關閉，避免既有 raw cue 在 `Play()` 後立即失去生命週期；這只關閉現代引擎播放可靠性，精確音訊與一般玩家E2仍缺，故只列partial。AI不套用玩家palette owner。缺baseline／DAC／table／sample／records／target／MP／RNG時交易不發生。ID33／34／35現只對具raw class19及selector4／5／6／7／20的玩家來源開放：record33／34／35分別做52／28／36 MP gate，但已證實來源均不在此分支扣款；33在私有records清`+0x25..+0x27`後以固定`0x320`走`0x211A4`回復，34依`0x22721→0x22866→0x22997`完成三段，35依`0x22D1B`以command26／22／27及`+0x25/+0x27/+0x26`完成三段。三者都先在私有records完成全部stage，正式command grid→target confirm回歸已通過。ID33／34／35均已另接`0x27FC9`正式indexed owner；ID34／35依三段mask／數字段邊界逐段發布並可整批回復。score／EXP、AI／其他visual group仍失敗即關閉。其他未知 command、狀態高階名稱、精確 DOS tick／音訊與完整 E2 仍未閉合；phase-expiry caller 與其 FDTXT／DATO／redraw／recalc 消費順序已由 [`fd2_transient_expiry_presentation_ida.txt`](../data/ida/fd2_transient_expiry_presentation_ida.txt) RE-CLOSED，selector 1→0／2 的倒數、歸零重算與 raw 同步已達 `RUNTIME-E1`，indexed 到期訊息已達 `RUNTIME-E1`；status colors／entries `0x37..0x39` 已由正式角色面板消費；精確 tick／音訊、高階名稱與一般玩家 E2 仍待。 |
| 指令28／29／31校正 | 閉合caller分歧；28／31正常取得來源未見 | command29資產與typed schedule就緒 | 28／31校正後數值E1；29玩家indexed owner E1 | 29缺E2；28／31非阻擋 | `0x276EC`已固定三支renderer／分母分歧。command29已由正式玩家confirm消費selector34／resource104並逐target原子發布。另以IDA固定command-mask OR writer只有level-up direct caller；固定learn table與32筆player defaults都不授予28／31，故「一般玩家無已證實取得來源」列強推論，不猜selector、不把它們當交付阻擋，也不冒稱死碼。主證據見[`fd2_command28_29_31_presentation_ida.txt`](../data/ida/fd2_command28_29_31_presentation_ida.txt)與[`fd2_command28_31_reachability_ida.txt`](../data/ida/fd2_command28_31_reachability_ida.txt)。 |
| 敵方人工智慧 | 底層控制流部分偏高；高階交易部分 | mode／候選／fallback 已資料化 | 正常 producer 的 E1 consumer 已接 | 只有原版敵方回合邊界 E2 | `0x13FD4`、`0x14EF0`、mode 5／11 等既有函式邊界與窄 owner 不應反覆重解。command 9 現以 `0x15311` producer target 走raw-side-zero indexed owner；10–12走`funcs_1541F`的60幀owner；17–22及26–27也已由同跳表進各自wrapper內建的effect／mask／numeric tail，並在Draw邊界原子發布raw-selector target transaction。`0x15055` 現保存並原子消費 `0x1567E` winner 的完整 raw target list；正常正分 type 5／13／20／21／24 的數值交易與caller-specific indexed演出均達 E1，不再借用玩家游標重算。type 5／13走`0x211A4`，type 20／24走`0x1CD17`，type 21走`0x1CAC7`四組toggle；三家都在各自Draw邊界後原子發布，保留音效、來源消耗差異與完整回復。2026-08-26末關動態診斷抓出三個重製consumer缺陷：物理helper現持有完整item table；`0x15311`保持actor原地並只保存effect destination，只有`0x1548E`消費`0x14B78`移動路徑；BattleFig 126／56所需FIGANI 379／168則由玩家唯讀archive嚴格按需解碼，兩筆全數驗證後才原子發布。雜湊鎖定的第三方末關候選現已在同一未修改原版程序以普通鍵盤完成`CONTINUE→END→YES→ENEMY PHASE`、敵方實際演出，並由後續Return開啟索爾狀態面板，直接證實玩家操作權恢復；這取代「兩次證據不能合併」的舊限制，但仍只屬第三方存檔候選E2且沒有音訊證據。mode 2 的 `0x14237` 無候選現由 `0x13C0F→0x13FD4` 正式 owner 消費：accepted gate第三個Draw後才加HP，拒絕gate零修改進共用收尾，缺資產仍零交易。敵方25缺正常AI producer而維持失敗即關閉。下一步只剩精確tick／音訊、同一raw狀態配對與一般玩家效果，不再重解已閉合owner。詳見 [`11`](11-enemy-ai.md)。 |
| 戰場 HUD、指令格、輸入與戰鬥演出 | 部分 | 部分 | 部分 E1 | 少量畫面 E2／多數缺少 | 有 native frame、command overlay、姓名字模、命中色盤與部分 FIGANI consumer。2026-08-22 已以同一未修改存檔由標題正常操作至悠妮 command 0 目標模式：原版四相位動態 LUT 為窄 `PLAYER-E2`，重製普通 X11 路徑達同座標／ID modal 為 `RUNTIME-E1`；時鐘相位未同步，故不是逐像素 parity。command 0 現由正式 Game confirm 接入 `0x2A6BD→0x29164→0x2B659→0x26152` 的完整預建與逐 Draw 發布：九段滑入、施術者效果、28 幀／7 元素錯開目標效果、七段 HP 與 LUT 尾段均達 indexed `RUNTIME-E1`，缺素材或 raw provenance 時在 MP／HP 前失敗即關閉。command 6 亦已接正式玩家與敵方 owner，涵蓋 common 前導／actor、全目標 orbit、九幀目標間過場、五段 HP 與尾段，達 indexed `RUNTIME-E1`；#87 單幀多呼叫混音仍只近似。兩者仍缺原版／重製同狀態逐幀、逐音訊 E2。整體操作狀態機、圖示可用性、其他 commands、相同戰況及演出時序仍未完成。完成度只由 [`57` 介面矩陣](57-ui-evidence-matrix.md)判定。 |

> **2026-08-26 第30戰重製人工智慧補證：** 同一固定晚期槽已由正式 LOAD、19人
> 整備、END／YES 進入未刪減敵軍回合，至少一名敵軍完成正式計畫後，完整回合交回
> `PLAYER PHASE`。首輪失敗揭露 persistent raw `+0x34/+0x35/+0x36` 在 typed party
> 邊界遺失；現已直接從槽記錄保留至 AI scorer，不替三欄附加未證實高階名稱。
> 此批為 `RUNTIME-E1`，第三方來源與停用音訊仍不升完整 `PLAYER-E2`。

> **2026-08-26 晚期整備同狀態勘誤：** 固定 `fd2last.sav` 現有只在截圖模式
> 啟用的正式 LOAD 擷取入口；它先由 `confirmTitleLoadSlot(0)` 還原29人／60金幣，
> 再走既有記錄提示 owner 進 `preparation_ch30`，不手工建立名冊或 campaign cursor。
> 舊截圖入口在擷取前仍推進圖像相位，所9255／3763／9255不是有效的
> 三相位比較；「灰階角色 selector／RLE／調色盤差異」斷言已被28格相位0
> 逐格 `AE=0` 否定。IDA 另證實兩組數字使用 FDOTHER #5 entries 31..40
> 與42..51；凍結相位並接通第二套字形後，固定初始狀態達
> `AE=0/64000`。本批仍只是重製 `RUNTIME-E1`，不外推完整整備E2。主紀錄見
> [`native-load-ch29-preparation-original-remake-e1.json`](../data/ui-traces/native-load-ch29-preparation-original-remake-e1.json)。

2026-08-26 晚期戰場勘誤：舊第30戰重製候選圖未提供玩家自備
`FDOTHER.DAT`，因此原生資產組拒絕載入並走PNG fallback；其洋紅地形與構圖不能
再當作`0x11CAC` renderer缺陷。補齊既有`FD2_ORIGINAL_FDOTHER`契約後，普通X11
鍵盤由標題`CONTINUE`抵達`battle_ch30`／round12／camera `(16,16)`／cursor
`(21,20)`，正式indexed六階段輸出19筆active、18筆camera-admitted與8281個
unit-stage寫入像素，foreground／HUD覆蓋該批像素均為0。這關閉舊PNG fallback與
unit覆蓋假說。後續合法IDA Pro 9.4閉合`0x10652→0x11EEE→0x4EB90`：raw chapter
28／29會先以`FDOTHER #55`和16-byte列偏移表建立312×192底面，再覆蓋會保留目的像素
的terrain tiles。正式runtime已接此typed底面並維持原子失敗；同狀態16相位比較的
最佳raw phase 10先由`AE=16281/64000`降至`AE=3242/64000`。後續又閉合
`0x11CAC(0)→0x4DFCC`的BIOS兩tick gate與DAC `0xE0..0xEF`滑動表；正式runtime接通後，
同一typed狀態的合法aux phase10／palette phase0達`AE=0/64000`。第三方存檔、固定
title tick與精確音訊仍只到`RUNTIME-E1`／候選E2；主紀錄見
[`native-battle-ch30-original-candidate.json`](../data/ui-traces/native-battle-ch30-original-candidate.json)。

2026-08-26 正常玩家接續勘誤：原版caller只要求確認`CONTINUE`當下的signed
16-bit timer seed；重製正式標題現由跨平台18.2065Hz單調時鐘近似器自行提供，
`FD2_NATIVE_TITLE_TICK`只保留為決定性測試覆寫。未設定環境變數的早期實檔與外部
第30戰候選都已通過正式title publication回歸。既有畫面擷取本身仍使用固定tick，
所以其證據分級不變；但「玩家必須設定timer環境變數」已不再是runtime阻擋。
同日Docker／Xvfb正式GUI再完整播放開場，不設定`FD2_NOCUT`或timer覆寫，只送普通
`Down、Down、Return`，於frame7202抵達相同第30戰狀態；圖與旁車已加入主紀錄。
再送一次`Return`的獨立重播於frame6300消費opening confirm並開啟共用indexed空游標
操作面板，證明正式GUI已把戰場操作權交給玩家，而非只停在載入完成狀態。

2026-08-24 補證：`0x525AF` 是 command 0..9 的 HP 分段除數表；typed
傷害計畫已不再把 command0 的七段套給全部 ID。command6 使用五段，並通過
決定性發布／越界拒絕回歸；其 12 張 target compositor 已由下述正式 Game owner
消費，這個舊的「尚未接 owner」狀態已失效。

同日續補：command6 的 7 張前導 orbit、12 張 target、7 張尾段 orbit 已全部
轉成具型別 layer／sequence 與 indexed compositor，包含 side 分流及前五個
marker 才發布 HP 的契約。正式 Game owner 與正常玩家／敵方 producer 現已接入；
逐呼叫音訊時序及一般玩家原版配對仍未閉合。

同日 `sub_2BA22` 唯一 caller／八參數 ABI／九幀相鄰目標水平過場亦由
IDA／Capstone 閉合並完成 typed sequence/compositor。控制流勘誤同時固定：
前導是 mode1→actor→mode2，尾段是 mode7→actor/target→mode8；舊有依 call
位址排序的相反說法已撤回。正式 owner 現不再缺多目標畫面資料，但仍須一次
預建 common prelude／actor phase、音訊與所有目標後才能提升 `RUNTIME-E1`；
下段所述工作已滿足這項門檻。

command6 handler-owned 全批次預建器現已實作並以兩目標 fixture 驗證
`7 + 12×target + 9×boundary + 7` 的形狀、每目標五個 HP stage，以及 malformed
晚期輸入零 partial output。正式 Game owner 現將 common `0x29164/0x2B659`、
#87 sub0..3、所有 final targets 與玩家／敵方 continuation 一次預檢後接入；
MP、五段 HP、`Acted` 與 RNG 均在對應 Draw 邊界發布，失敗時整批回復，達 indexed
`RUNTIME-E1`。目前 #87 mode5 的多次 raw 呼叫以每呈現畫格一次播放近似，未宣稱
逐呼叫混音、精確取樣率或一般玩家 `PLAYER-E2`。

**2026-08-25 第 8 號指令補證（`RE-CLOSED`／`DATA-READY`）**：IDA Pro 9.4
與 Capstone 已固定 `sub_274B0` 的精確邊界 `0x274B0..0x275D6`；緊接的
`sub_275D6` 是另一函式，不得混入。mode 0 以 `0,-2,...,-30` 初始化 16 個
`dword`，故舊索引 `0x540BA..0x540C9` 已更正為 `0x540BA..0x540F9`；mode 2／5
皆會消費 `0x52539` 的 16 個 frame base、在遞增前 counter 0／4 呼叫音訊
sub1／sub2，並在遞增後 counter 4 回傳數值標記。FDOTHER #28／#30 均有精確
32 幀，#90 提供共同 actor sub0 與 handler sub1／sub2。typed planner 已保存
3／34／2 frame 單一 target 契約及 16 個 HP stage；entry 不回收 counter，故
多目標在 caller 補證前失敗即關閉。正式 Game owner 現已由玩家確認與敵方
mode 11 共用，完整預建 common／actor／handler／tail，並逐 Draw 發布 MP、16 段
HP、`Acted` 與 RNG；失敗整批回復，達 indexed `RUNTIME-E1`。一般玩家 E2 尚未接，
不能外推為完整原版一致。主證據見
[`fd2_command8_presentation_ida.txt`](../data/ida/fd2_command8_presentation_ida.txt)。

**2026-08-25 第 9 號雙路徑補證（雙方 `RUNTIME-E1`）**：
IDA Pro 9.4 與 Capstone 固定玩家 `0x214AD→0x1C4CC/0x1DF58` 與敵方
`0x15311→0x2A6BD→0x275D6` 是不同 compositor。玩家 typed 規格保存 #6
87..113、raw sample selector14／15、單一 target、#5 命中／未命中 descriptors
與22張結果；`sub_1D4CB` 已證實載入 #80，selector14／15 亦有實檔子樣本。敵方 entry 保存 mode
20／60／20、toggle、counter1..61、#90 sub0..2、#44恰31幀、27個raw marker與
前20個HP stage，以及11／8張 actor slide。`0x525B9` 只複製九筆，故敵方
raw side非零 ID9 讀未初始化 padding；正式 owner 僅接受 raw side0。

敵方 mode11 現使用窄 `PlanNativeAICommandDamageSingleTarget` 消費 `0x15311`
已選定的 producer target，不再錯套玩家陣營 target-code geometry。所有119張
handler frame、common actor／tail 與原始資產先完整預建，才逐 Draw 發布 MP、
20段HP、`Acted`與RNG；錯誤整筆回復。其後重核 `[0x53B13]` writer，證實
`sub_1D4CB` 載入 FDOTHER #80；玩家正式 map owner 現依序執行 #80 selector0、
八段色盤、#6 的27張效果、selector14／15、原子 MP／HP、#5 的22張結果與
500ms hold，完成後才標記行動。一般玩家／敵方同狀態E2仍待。主證據見
[`fd2_command9_player_ai_presentation_ida.txt`](../data/ida/fd2_command9_player_ai_presentation_ida.txt)。
| 城鎮、祕密商店、商店、教會與整備 | 部分偏高 | 部分就緒 | 多個正式 E1 consumer | ch02 若干狀態 E2；其餘部分 | 個別 menu、購買、賣出、轉移、復活、轉職與整備已有窄切片；ch02 賣出已由正常商店輸入走完角色／物品／Yes-No、成功、向上金幣滾動及返回名冊，九組 route-patched 原版／正式重製畫面皆整幀 AE=0。獨立裝備 service2 另由正常商店輸入取得名冊與索爾面板，但動畫相位未同步，整幀仍為 AE=1389／1433；正式交易現先在私有 unit 完成 raw 裝備、重算與 panel 重建，最後才發布，深層 renderer 失敗不再污染 roster／能力／既有 panel，達 `RUNTIME-E1`。service3 物品轉移除五個選擇狀態外，現也由正常商店輸入完成索爾短劍→悠妮，返回 loop 後原版索爾只剩皮甲／藥草、悠妮追加未裝備短劍；四個成功交易畫面 AE=1391／82／2／286。可見內容與幾何一致，剩餘差異是角色、翻頁箭頭或選取脈動相位，故仍列 route-patched partial E2。重製同一跨角色交易又穿越 `town_ch02` JSON 冷讀檔，保存雙方 compact/raw 背包、裝備、能力、金幣與隊伍順序。service3現由單一具型別輸入consumer承接Ebiten鍵盤；正式menu Right×3後逐Draw走完empty、full、目的取消與self-transfer，full／cancel保持角色與金幣原子不變，自我轉移仍依raw remove→append／重算，達production-input `RUNTIME-E1`。裝備收件者以六名具完整 raw provenance 的 typed party，從正式 menu→purchase→Yes 進三列面板，走過 scroll、滿欄／無合適角色原子返回與成功裝備／扣款，亦達 `RUNTIME-E1`。正式`church_ch02`主選單固定`selection=0,pulse=2,gold=1000`後，640×400 runtime最近鄰縮回320×200與現行原始資源oracle達`AE=0/64000`；舊oracle的320點差異已確認為過時圖片並替換，這仍只提升`RUNTIME-E1`而非DOSBox E2。教會轉職與其他商店 mutation 也能返回 town，再穿越重製 JSON 存讀檔。第25戰後的 `town_ch26` 祕密商店 E1 亦已接通。這些不等於未修改一般玩家戰間 E2；其餘章節入口、原版存檔、recipient scroll／no-recipient／full、service2 原版 mutation／restore畫面，以及 transfer empty／full、self／destination-cancel 與 church caller 的**未修改原版同狀態 E2**仍缺；重製正式 consumer 不再列為缺口。 |
| 音樂與音效 | 格式閉合；owner／時序部分 | 部分就緒 | 部分 E1 | 逐音訊 E2 缺少 | XMIDI、兩類音源與部分曲目／樣本 owner 已知。共用短音效播放器現保留疊播 voice 至自然結束並於退出清理；玩家自備 MT-32 OGG 的終局 `FDMUS_004→stop→FDMUS_018` 也已在無聲 Docker 實際解碼、建立及切換播放器。這仍不證明人耳輸出、裝置延遲或三平台音訊；重點是精確播放時機、效果同步與真機播放，不需要重解 XMIDI 格式。 |
| 終局與結局 | 部分偏高 | 來源約束排程已資料化 | E1；第27戰missing與正式 `battle_ch30→ending` 已接各自原版文字臂，最終戰另接終端定格／隊伍回顧 | 缺一般玩家終局 E2 | `0x250CC`缺天空之鑰臂的`0x2545D→0x2BCE5`已由正式inventory gate進chapter26來源約束前綴，消費`FDTXT_027` index17..20，不再只顯示通用結語。`sub_2C39B`的caller頭像現與FDTXT逐句speaker分離：chapter29 index2..7直接保存`FFEC..FFEF`控制碼、operand與頁面，runtime不再把整個block錯畫成同一個caller頭像；chapter26無內嵌speaker的四句才沿用caller arg0。正式runtime已把typed speaker／editable text接入19×5 indexed owner，包含開框、逐字、嘴型、輸入等待、收框與source restore；只保留精確時序及一般玩家E2。`0x2BCE5` 前綴、`0x2C548` 角色蒙太奇、20段尾段與 FDOTHER #59 定格則已由最終戰campaign以原始資源與持續raw roster消費；資產或provenance不足時整批失敗即關閉。定格預設永久停留；Enter／Space可進入重製端明示的隊伍最終狀態循環，Enter／Space／Escape返回同一原版定格。成功路徑已移除來源等級與按鍵說明等現代疊圖，只在除錯HUD顯示。`Game.Update`及第29戰後冷讀長鏈回歸現共用單一終局輸入owner：raw-change略過、定格進回顧、Escape返回均不再直接改旗標；原版scan code仍維持未知。全新 `Game` 由最終整備冷讀檔到永久定格／回顧的有界回歸，已逐人核對 JOIN 順序與 persistent raw `+6/+7/+8/+0x20`，關閉重製端連續性；它仍不是未修改原版的完整動態 oracle。20段實際80個FIGANI已證實全部header byte1=0；runtime現逐raw `+6` inner present、`+7 bit0`層序、`+4`位移／palette33與最後effect終止、base scheduler執行兩次交叉配對。`0x2939D`的3%外層預算已閉合到未初始化`var_4C→var_44→record+0x40` consumer；它不是穩定終局重播契約，正式重製不模擬且不再列交付阻擋。尚缺原版 caller `0x2C2A6` 當下完整動態狀態、精確音訊時序與一般玩家原版owner／E2，不是重解整個`0x28A6C`。 |
| DOS/4GW、Watcom runtime、Miles 驅動與一般函式庫 | 第一輪分類 | 不適用 | 只在行為外露時處理 | 不適用 | IDA 清冊1305函式中170筆由 Watcom FLIRT 標成 runtime；其餘未分類不能都算產品程式。後續只擴充分級索引，不把函式庫未命名算成 remake 缺口。 |
| 三平台打包與推廣片 | 不適用 | 部分 | 尚未達發行閘門 | 缺完整玩家驗收 | 這不是反組譯問題。待核心戰役、操作 UI、結局與代表性一般玩家路徑關閉後，再做 Linux／Windows／macOS 打包與影片。 |

**2026-08-26 service2 正式輸入補證**：獨立裝備現由 typed consumer 從 service
selection 2 走完角色名冊、原版 item scan code、相容／不相容交易、空背包、panel
收合、同角色名冊重開與返回 menu。交易與 panel 仍維持候選一次發布；原版
mutation／restore 同狀態 E2 未提升。

**2026-08-26 教會入口補證**：`0x3072F` 四項服務不再只有分散 callee 回歸；
正式鍵盤與測試現共用 typed menu consumer，並把服務發布延後至四段關框及
source restore 之後。raw index 0 的名冊、狀態、指令面板與返回名冊也已沿同一
typed consumer 完整往返。此項為 `RUNTIME-E1`，church caller 同狀態 E2 仍缺。
同一教會 input owner 又已覆蓋共用 `0x2F8EA` 的 source／item／destination／full，
以正式狀態驗證跨角色成功、自我轉移、取消與滿欄零交易。
raw index 2 復活亦已沿 typed input 覆蓋候選→確認→成功／不足金／取消→empty／menu，
成功交易才啟動既有 track 21→indexed timeline→track 14 owner。
raw index 3 轉職也已沿 typed input 覆蓋候選、唯一 target 確認、取消、缺表拒絕與
成功完整 persistent unit 發布；四項教會服務至此均有正式 input owner。

**2026-08-25 ID32 現況勘誤**：上表長列保留了本批開始時「ID32失敗即關閉」的
歷史快照文字；現況由[`fd2_command32_transaction_ida.txt`](../data/ida/fd2_command32_transaction_ida.txt)、
[`fd2_command32_35_presentation_ida.txt`](../data/ida/fd2_command32_35_presentation_ida.txt)
與[`fd2_command32_tail_presentation_ida.txt`](../data/ida/fd2_command32_tail_presentation_ida.txt)
取代。ID32已由正式grid→confirm接到受限class19玩家indexed owner及原子交易，達
`RUNTIME-E1`；仍失敗即關閉的是score／EXP、AI、其他visual group與一般玩家E2。

**2026-08-25 transient名稱與敵方mode2現況勘誤**：FDTXT_000 #481..486已直接
固定 `+0x22..+0x27` 的攻擊力／防禦力／速度增加效果、毒性、痲痺與封咒到期文字；
玩家可見分類不再未知，高階enum或圖示名稱不是runtime阻擋。mode2的`0x14237`
無候選亦已正式接入`0x13C0F→0x13FD4`；accepted gate第三個Draw後才加HP，拒絕gate
零修改完成單位，缺presentation仍零交易。上方玩家指令列末尾較早的「狀態高階名稱
仍未閉合」由本段取代；剩餘只屬精確音訊／時序與一般玩家E2。

**2026-08-25 ID33 現況勘誤**：上表長列的「三者只關閉state transaction」同樣是
歷史快照。ID33現已由正式grid→confirm接到#66／#92共用段與`0x211A4`專用尾段，
達受限class19玩家`RUNTIME-E1`；ID34／35亦已接三段正式owner。主證據與重開
條件見本頁`0x211A4..0x21206`列。

**2026-08-23 敵方物理提交補強（17–19尾段句已於2026-08-25勘誤）**：mode 2與mode 11 `0x1548E`正式入口現先預檢
攻方attack FIGANI、守方idle FIGANI與descriptor delay，全部可建立排程後才消耗
RNG並發布HP／死亡獎勵；缺素材維持零交易。另確認敵方17–19由`0x15311`直接進
effect table，原版不經玩家專用`0x1D6C8` palette owner，故不把玩家八相位演出
誤接到AI。後續raw跳表補證另證實三個wrapper仍消費各自的handler內建尾段，不能
據此維持state-only；以後文2026-08-25勘誤為準。

**2026-08-25 敵方17–19尾段勘誤**：`0x1541F`的loaded table `0x51D01`
indices17／18／19分別是`0x226EA／0x2282F／0x22960`；三個wrapper會進已閉合的
`0x22721／0x22866／0x22997` effect／mask／numeric tail。敵方不使用玩家
`0x1D6C8`八相位palette，但正式owner必須呈現對應單段尾段並在mask／結果hold邊界
發布交易；主證據為[`fd2_command34_tail_presentation_ida.txt`](../data/ida/fd2_command34_tail_presentation_ida.txt)。

**2026-08-25 敵方20–22尾段勘誤**：同一loaded table indices20／21／22已由
IDA Pro 9.4與raw bytes閉合為`0x22A85／0x22BC6／0x22BE1`；wrapper分別進
`0x22AF6` clear/restore或`0x22D1B` application，三者都消費`0x1C4CC`效果、
`0x1C2DA`遮罩、結果queue與`0x1DF58`數字段。正式敵方owner現以raw selector
重建target array、原生16-bit RNG私下預算；mask完成後才發布MP／HP／marker／RNG，
22張數字段與500 ms尾停後才發布`Acted`。ID20原始資產端到端與取消回復聚焦
回歸已通過，達`RUNTIME-E1`。後續同批已讓玩家20–22以confirmed-cursor target
plan保留`0x1D6C8`八相位palette，再串同一handler tail；原始資產ID20測試固定
完整Draw／發布／完成邊界，玩家與敵方皆達`RUNTIME-E1`。主證據為
[`fd2_command20_22_player_ai_presentation_ida.txt`](../data/ida/fd2_command20_22_player_ai_presentation_ida.txt)。

**2026-08-25 指令25–27尾段補證（`RE-CLOSED`／`DATA-READY`）**：raw
`funcs_1541F` entries25／26／27已閉合為`0x22C04／0x22CBF／0x22E41`，同時由
敵方`0x1541F`與玩家`0x1D479`間接消費。25具#6 `0xBF`起13幀、sample5、mask
`0xC0`；只在raw `+5 bit7`未設時加入failure queue，全成功時不呼叫數字段。
26／27各具#6 `0x8A`起9幀／sample3與`0x9E`起12幀／sample2，兩者走
`0x22D1B`的effect／mask／成功或失敗queue／numeric tail。玩家25–27及敵方26／27
的正式owner現均已依mask Draw邊界原子發布，25全成功時省略空數字段，達
`RUNTIME-E1`；敵方25缺正常AI scoring producer，維持失敗即關閉。玩家25與敵方26
原始資產端到端回歸已通過。主證據為
[`fd2_command25_27_player_ai_presentation_ida.txt`](../data/ida/fd2_command25_27_player_ai_presentation_ida.txt)。

**2026-08-23 敵方 command0 演出接線**：`0x15311`在ID `<10`且
`Raw53AF9==0`時進`0x2A6BD`；正式敵方ID0已改用既有完整
`0x2A6BD→0x26152` presenter，不再直接發布state-only傷害。預建失敗時MP、HP、
RNG與`Acted`均不變；完成後才交給敵方continuation與死亡獎勵。ID1–8的
`funcs_2AC25`不能外推ID0專屬演出。ID1現已閉合`0x262EF`的八槽位移、
mode4→target→mode5順序與FDOTHER #19/#21 30-frame資源，並有純indexed
compositor及原始資產回歸。2026-08-24直接指令再勘誤：mode3回傳31，八個
numeric marker分布於step `8..22`的偶數步，八個sample1 marker分布於step
`4..18`的偶數步；舊「九張、無直接sample call」已撤回。它只到
`DATA-READY`。同日後續正式owner已一次預建common actor、每目標31張／八HP
marker、每boundary九張及四張common tail，並接入玩家與敵方producer；MP、HP、
RNG、`Acted`依Draw邊界發布且可整批回復，提升為indexed `RUNTIME-E1`。仍缺
#82取樣率人耳確認及正常未修改玩家／敵方同狀態逐幀、逐音訊`PLAYER-E2`。

ID2的`0x26528`亦已修正為自身state `0x53F7E..0x53F80`、#26/#27
18-frame FIGANI與#83 samples1／2／3；`0x52460/0x52490/0x5249C`實為ID3
資料，舊ID2關聯已撤回。`0x2673F`現由直接指令閉合為單一frame/repeat、原位
effect draw與descriptor-delay推進；29張front、每target12張／六HP marker、
每boundary九張及10張tail皆已轉成typed state／compositor primitive，並以原始
#26/#27資產驗證不越界；正式Game owner現已一次預建29張front、各target
12張、相鄰target九張、10張handler tail與共同四張tail，並接入玩家與敵方
continuation。MP、六段HP、`Acted`、RNG只在Draw確認後發布，失敗整批回復，
達indexed `RUNTIME-E1`；未修改原版同狀態逐幀／音訊仍缺，不宣稱E2。

ID3的`0x26795..0x269D3`已由IDA Pro 9.4、Capstone與原始資產重新閉合。
舊「12個RNG-rotated slots」及JSON `uses_rng=true`／raw side零位移已被直接指令
推翻：mode0確定性初始化12個staggered counter與position，沒有呼叫RNG helper；
raw side零值將`0x52460`的12個X全部加20。`0x52490／0x5249C`分別是vertical-row
及frame-base表；#39／#43均為33幀效果，#84 sub0屬common actor、sub1／sub2
屬handler。mode0／3／6的2／40／20預算、toggle雙張、state範圍、sample條件與
每target 14個raw marker／前13段HP均已保存。typed planner／compositor已原子
預建2張front、每target40張、每boundary9張及20張tail，原始資產回歸通過。
正式Game owner現一次預建common前導／actor、完整handler及共同4張LUT tail；
玩家確認與敵方mode 11共用逐Draw發布／整批回復，提升為indexed `RUNTIME-E1`。
同handle同畫格的raw sample疊音與未修改原版一般玩家逐幀／音訊仍不宣稱E2；主證據見
[`fd2_command3_presentation_ida.txt`](../data/ida/fd2_command3_presentation_ida.txt)。

ID4的`0x269D3..0x26BFC`已由IDA Pro 9.4、Capstone與原始資產閉合：正常敵方
評分器可產生ID4，`0x15311→0x2A6BD→funcs_2AC25[4]`是正式消費鏈；六槽
counter／position／`(rng%2)*7` phase、2／12／8預算、六段HP gate、十個offset、
raw side零值+143、#22／#23十四張效果及#85 sub0／sub1均為`RE-CLOSED`／
`DATA-READY`。正式敵方owner現以raw `+6` selector從選定目的格重建目標陣列，
一次預建2張front、每target12張、每boundary9張、8張handler tail與共同4張tail；
逐Draw發布MP／六段HP，完成後才發布`Acted`與數值RNG，執行失敗整批回復，達
indexed `RUNTIME-E1`。同畫格多次sample1目前合併播放，列為混音近似。
玩家producer仍未知，不猜接；未修改敵方同狀態逐幀／音訊E2另列。主證據見
[`fd2_command4_enemy_presentation_ida.txt`](../data/ida/fd2_command4_enemy_presentation_ida.txt)。
本批另把相同raw-selector target-array契約接回全部敵方ID0..8 presenter；先前
共用玩家confirmed-cursor admission會在實檔`TargetCode=0`下拒絕敵方攻擊目標，
該消費端錯誤已撤回。各ID既有entry證據與畫面排程不因此重開。

ID5的`0x26BFD`已由IDA Pro 9.4完整直接指令、Capstone及原始資產交叉閉合：
mode0／3／6回傳1／12／8；六個counter、十位置循環、六條RNG phase、stop gate、
`0x524D0`水平offset、raw side零值`+143`、#24／#25十二張delay0效果、#86
sub0／sub1與六個直接sample marker均已保存；sub0屬common actor，兩個handler
callee都以sample index1消費sub1。舊JSON把channel3誤寫成counter3，
並漏掉mode2／8與`0x25A96`，現已訂正。typed planner保存跨target與九張boundary
持續state，第一target九個raw marker中只讓前六個發布HP；indexed compositor
保存actor／target／effect層序，工具也不再把合法delay0 frame丟棄，達
`RE-CLOSED`／`DATA-READY`。正式Game owner現已一次預建common前導／actor、
1張front、每target 12張、每boundary 9張、8張handler tail與共同4張LUT tail，
並接入玩家確認及敵方mode 11；MP、六段HP、`Acted`與數值RNG依Draw發布且可整批
回復，提升為indexed `RUNTIME-E1`。
原版process-wide RNG與既有整批damage plan的跨target交錯列為明示近似，不猜測
改寫數值順序。主證據見
[`fd2_command5_presentation_ida.txt`](../data/ida/fd2_command5_presentation_ida.txt)。

ID7的`0x272B8..0x274B0`已由IDA Pro 9.4、Capstone與原始資產閉合：mode0／3／6
回傳2／32／16；四組初始化但只render前三組，toggle令每個counter畫面重複兩張，
state跨target與九張boundary持續。`0x52511`十個offset、raw side零值`+130`、
#37／#38五張效果、#88 sub0 actor／sub1 handler、三種draw mode的六個直接sample
marker，以及第一target七個raw marker／前五段HP均已保存。舊JSON只列mode5的
兩個sample marker，現已補齊mode2／8。typed planner／compositor現由正式
Game owner消費，一次預建common前導／actor、2張front、每target 32張、每boundary
9張、16張handler tail及共同4張LUT tail；玩家確認與敵方mode 11共用逐Draw發布
及整批回復交易，完整Go回歸通過，提升為indexed `RUNTIME-E1`。未修改原版一般
玩家逐幀／音訊比較及外層numeric-marker shake仍是E2缺口；主證據見
[`fd2_command7_presentation_ida.txt`](../data/ida/fd2_command7_presentation_ida.txt)。

ID6的`0x26E39`已由完整直接指令與原始資產交叉驗證：五個local table值、
mode0／3／6回傳7／12／7、#32/#33十幀FIGANI及#87 samples1／2／3均已成
strict typed schedule與實檔回歸，達`DATA-READY`。一次子代理探針曾誤解為
六槽／mode0回傳2；Capstone完整body顯示那是錯誤函式資料流，尚未進版控且已
撤回。`0x3C885/0x3C898`另由直接`fcos/fsin`閉合五點座標純函式；mode4／5完整圖層、外層交易及正常敵方回合
現亦以兩張typed target draw plan與indexed compositor閉合，包含side分支、
十二張mode3 budget、counter numeric marker、frame4負值替代及frame5..9
secondary effect。外層actor/tail、
交易marker及正常敵方回合仍未閉合，因此正式AI繼續使用既有state-only數值
路徑，不冒稱演出E1。

另經AI mode交叉核對，map13 index0雖有command30 bit，但raw mode低四位為8，
`0x13A9A`直接走`0x1317D`，不進`0x14EF0/0x1598A/0x15311`。因此它不是敵方
command30 producer，也不構成缺少AI executor的交付阻擋。

## 四、目前可重生的數字，以及不能怎麼解讀

2026-08-27 以唯讀原版、一次性無網路Docker與合法IDA Pro 9.4重生現有稽核：

- IDA Pro 9.4 對固定雜湊 `FD2.EXE` 辨識1,305個函式；受版控語意索引有67筆，
  其中61筆屬產品程式、6筆屬 runtime。匯出器讓語意索引覆蓋同一函式分類，
  因此沒有重複計數；重生清冊為產品61、runtime175、未知1,069。
- 60份 raw handler script 原有83個 `unknown` call site、23個 target。重生後為
  93個 `native_call`（26個 target），`unresolved_native_call` 與真正
  `unknown` 都為0。`0x2189A`、`0x24BDE`、`0x24D22` 是已有直接證據卻漏登
  語意索引的產品函式；`0x22253`、`0x2BCE5` 則依上游 caller 與正式失敗即關閉
  adapter 升為已分類呼叫。每筆
  具名呼叫皆保存原始位址／PUSH 順序、推論等級與證據檔；編譯器仍逐 caller
  驗證並失敗即關閉，故這是**工具債清除**，不是玩法完成率。
- `campaign_full.json`：121 個 story／cutscene 節點；9 個 scripted、57 個
  handler-bound、55 個 fallback。fallback 可能是撤退、傳聞或尚未接線故事，不能全部
  視為同一種缺口。
- postbattle audit：24 個節點，24 active、0 blocked；mapping gap 為0，
  且已無未分類 native semantics。active 也只表示 admission gate 可通過，不代表
  該章一般玩家 E2 或逐像素一致。

上述數字只能用來定位工作，**禁止相加或換算成遊戲完成百分比**。
重算時只更新本節與上方矩陣；README、SDD、介面矩陣與工作清單只連回本頁，不再
各自保存一份會漂移的「目前總數」。舊日期條目若保留當時數字，必須明寫為歷史快照。

## 五、已知位址的「不要重做」索引

下表指定目前主證據與可重開條件。舊交接、SDD 附錄或 exporter 仍寫 `unknown`，都不能
單獨成為重做理由。

| 位址／家族 | 現有主證據 | 已閉合範圍 | 仍可做的工作 |
|---|---|---|---|
| `0x15F84`、`0x16B43`、`0x16C57`、`0x16559` | [`29`](29-remake-extensible-event-system.md)、[`fd2_story_dialogue_layout_ida.txt`](../data/ida/fd2_story_dialogue_layout_ida.txt)、各 handler 直接指令 | 基本 renderer、四種故事開框碼、`FFFE/FFFD`、`sub_165AC`五階段opening、逐raw glyph寫入、`sub_16C57→sub_16559`等待期嘴型、`sub_16B43`五張snapshot restore／可選游標尾段已`RE-CLOSED`。多rune Unicode映射另以`glyph_pages`保留一個raw word一個16px token，避免13格`ASR-07`被誤算16格；資料模型與compositor測試已通過 | 只開其他 caller-specific binding與E2；不重解基本函式角色、opening、逐字、嘴型或closing順序 |
| `0x1366A` | [`50`](50-cutscene-script-system-design.md)、`chapter_beats` | acting 呼叫原語 | 個別資源、場景時序與畫面；不重解原語本身 |
| `0x11DF2` | [`fd2_11df2_palette_disasm.txt`](../data/fd2_11df2_palette_disasm.txt) | palette range/delta helper | caller 時序與畫面；應回填 exporter |
| `0x1F882`、`0x25052` | [`91`](91-worklist.md) 對應直接指令證據 | 兩種不同 palette ramp | runtime renderer／caller E2；不再稱 vsync |
| `0x13FD4` | [`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt) | raw gate、回復與窄 presentation owner | 同狀態交易／逐幀逐音訊 E2；不重解函式邊界 |
| `0x14EF0` | [`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt) | producer 順序與尾端 dispatch | 未知 command／效果／完整 transaction；不重解既有 dispatch |
| `0x14237` | [`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt) | 物理候選評分窄切片 | 完整 planner、target transaction 與 E2 |
| `0x15311`／`0x1548E` | [`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt) | mode 11 兩段 owner／順序 | 未知 command、完整演出與 E2 |
| `0x28A6C→0x2939D` 命中位移 | [`fd2_battle_impact_displacement_ida.txt`](../data/ida/fd2_battle_impact_displacement_ida.txt) | `RE-CLOSED`／`DATA-READY`：`0x5255F／0x52577` 是六相位水平／垂直位移，不是idle fallback；`0x29F72`未命名輸出、palette33、相位5→0、raw `+6`正負方向與`0x2935B` consumer已閉合 | 只把位移接入既有E1剪影分支並逐幀比較；原始輸出高階名稱、DAC trigger、完整音訊與一般玩家E2仍未知，不重解整個renderer |
| `0x2939D` 終局3%外層預算 | [`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt) | `RE-CLOSED`／`NON-BLOCKING`：`0x2946A..0x29480` 只增加外層預算；終局非零分支跳過 `var_4C` 初始化，卻由 `0x29742→0x2975A→0x29B4C` 讀寫並決定第二輪；第二次配對另於 `0x29B27` 提前返回 | 不把未初始化堆疊（stack）／HP 寫入固化成重製規則；只有新動態追蹤（trace）證明穩定玩家可見契約時才重開 |
| `0x2A289→0x18C6D` 狀態欄 | [`fd2_battle_status_panel_ida.txt`](../data/ida/fd2_battle_status_panel_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：entry22框、23..30 bar cells、31..52／93 digits與FDOTHER#4姓名均已接；普通攻擊不再另畫RGBA bar | 固定命中fixture排除oracle合成邊框後內容區AE=0；仍缺新的未修改一般玩家同狀態E2，不重解既有consumer |
| `0x16F55`、`0x1728C`、`0x117E7`、`0x4E8E1` | [`fd2_system_overlay_options_ida.txt`](../data/ida/fd2_system_overlay_options_ida.txt)、[`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt)、[`empty cursor owner`](../data/ida/fd2_117e7_empty_cursor_system_overlay_ida.txt)、[`END確認框`](../data/ida/fd2_end_turn_confirmation_ui_ida.txt)、[問句同狀態比較](../data/ui-traces/native-end-turn-confirmation-original-vs-remake-e1.json)、[逐字回覆動態配對](../data/ui-traces/native-end-turn-response-progressive-original-vs-remake-e1.json) | `0x12C0D==-1` 空游標 owner；direction2設定、direction3 END及direction1全軍移動均已接正式E1；END另有DATO #75／FDOTHER #5/#2、FDTXT `0x1A3/0x1A4/0x19C`與`0x1A30B`生命週期；`0x4E8E1`由右往左寫入且`0x9017`是右端錨點 | 精確 DOS tick／音訊及收合逐幀同相位、selector0巢狀current-runtime存讀檔、逐章重製PLAYER-E2；正式資料的selector1 event61／75已接通，其他未具owner的90-entry分派仍失敗即關閉；亦不重解設定writer／consumer、END分支與renderer |
| `0x19DF7`、`0x1B1E7`、`0x16FF4` | [`fd2_nested_system_menu_ida.txt`](../data/ida/fd2_nested_system_menu_ida.txt)、[`fd2_system_exit_and_group_march_ida.txt`](../data/ida/fd2_system_exit_and_group_march_ida.txt) | `RE-CLOSED`、`DATA-READY`、`RUNTIME-E1`：nested資訊／存檔／讀檔／離場四分派及四個正式 owner 已接。selector2 LOAD 使用 FDTXT `0x19D/0x19E/0x19C`，在私有 `Game` 完成 current-runtime typed handoff，YES 回覆生命週期後才原子替換；selector1 SAVE 從完整 raw baseline 複製未知 bytes，並只接受有 provenance 的 live／新 JOIN 欄位。外層 selector1 全軍移動使用既有 `0x51B91` 表；event61／75 在途中完成各自已證實事件後續行，未知事件在發布前整批拒絕 | 尚缺未修改同狀態逐幀／tick／音訊 E2；長程存讀檔由使用者人工回報問題。正式資料只有 event61／75 兩筆 selector1 rule；不得外推成90-entry全完成，也不得重解四分派、資訊 schedule、selector3目的地或 `0x51B91` 表身 |
| `0x21AD9`／`0x21B18`（command 13–16 wrapper 家族） | [`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt) | 四個 command literal、wrapper 參數、共同 indexed presentation owner、玩家／AI callers及正式 E1 | 補同狀態逐幀逐音訊 E2；不重解 wrapper |
| `0x21EB1`／`0x22046`（command 13–16 LUT 演出） | [`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt) | FDOTHER #3 LUT provenance、16張排程、visible-cursor中心、兩段200 ms、sample index11、compositor consumer及玩家／敵方 E1 | 補同狀態逐幀／逐音訊 E2；不重解 loop |
| `0x1C4CC`／`0x1C2DA`／`0x1E0DB`／`0x1DF58`（command 13–16 後段） | [`fd2_command_numeric_tail_ida.txt`](../data/ida/fd2_command_numeric_tail_ida.txt) | FDOTHER #6七幀、五組snapshot→mask、`0x4DDD7` write mask、transaction後redraw、FDOTHER #5 queue／22-frame reader與玩家／敵方 E1 | 補同狀態逐幀／逐音訊 E2；不重解函式 |
| `0x20C6F→0x1C4CC→0x1CD17→0x1E0DB/0x1E1DC→0x1DF58`（AI item type 20／24） | [`fd2_ai_item_damage_1cd17_owner.txt`](../data/ida/fd2_ai_item_damage_1cd17_owner.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：command 0／2／3使用#6 `0x31..0x38`、#80 sample6、十張`0x52006[commandID]=0x20` blend、命中bias `0x5E`／失敗glyph `74 75 76 76`及22張結果；正常item79原始資產入口在第18個Draw後才原子發布HP／death bit／RNG，來源物品保留，尾停後才`Acted` | type21另走`0x1CAC7`不得借接；本列只缺同狀態逐幀／逐音訊與一般玩家E2，不重解上述函式 |
| `0x20C6F→0x2111A→0x1C4CC→0x1CAC7→0x1E0DB/0x1E1DC→0x1DF58`（AI item type 21） | [`fd2_ai_item_type21_1cac7_owner.txt`](../data/ida/fd2_ai_item_type21_1cac7_owner.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：正常item29／38／51／99只選command6／1／7／6；command1使用#6 `0x31..0x38`與sample6，6／7使用`0x40..0x49`與sample9；共用`0x4A→0x4B`四組90 ms toggle、命中／失敗queue與22張結果。原始item38正常入口在第16個Draw後發布HP／death bit／RNG，來源保留，尾停後才`Acted` | 只缺同狀態逐幀／逐音訊與一般玩家E2；不得借一般command21的`0x22BC6`，不重解`0x1CAC7` |
| `0x1D4CB`／`0x1D6C8`（玩家 command sound writer／palette owner） | [`fd2_command_sound_handle_53b13_ida.txt`](../data/ida/fd2_command_sound_handle_53b13_ida.txt)、[`fd2_command_modifier_palette_ida.txt`](../data/ida/fd2_command_modifier_palette_ida.txt) | `sub_1D4CB` 以常數0x50載入FDOTHER #80至`[0x53B13]`；`0x1D6C8`唯一caller為`0x1CFF0`，消費#80 selector0、三張36-byte DAC table與四輪color／black；17–23與25–27玩家正式E1均先演出後交易，23並串接兩次`0x22253`離場／入場。ch24 `0x33979` 對同全域的#88覆寫是另一個局部owner | 只補command23同狀態camera／逐幀、精確tick與逐音訊E2；phase-expiry與status panel已由後列主證據關閉；不重解palette loop或`0x22253` callee |
| `0x22C04..0x22E5C`（commands25–27 handler tails） | [`fd2_command25_27_player_ai_presentation_ida.txt`](../data/ida/fd2_command25_27_player_ai_presentation_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：跳表 entries、玩家／敵方間接 consumer、25 failure-only queue、26／27 application queue、三列 raw effect／sample／mask schedule，以及玩家25–27／敵方26–27的逐 Draw owner與回復均已閉合 | 敵方25沒有正常 AI producer時維持失敗即關閉；其餘只補同狀態逐幀／音訊 E2，不重解 handler 或正式 owner |
| `0x1F558`／`0x21527..0x21AD9`（玩家／敵方 commands10–12） | [`fd2_command10_12_presentation_ida.txt`](../data/ida/fd2_command10_12_presentation_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：三wrapper、ID11／12的#80 selector2與`0x2189A`實參、共用三張取樣表、四surface×60張、#80 selector13八個marker及numeric tail順序已閉合；typed fixed-point compositor與正式玩家owner已接；敵方`0x15311→funcs_1541F`現亦使用同一逐Draw owner及MP／HP／RNG／acted原子rollback | 不重解`0x2189A/0x219AD`；一般玩家同狀態逐幀／逐音訊E2另列 |
| `0x27FC9..0x286BD`（玩家 commands32–35 共用演出） | [`fd2_command32_35_presentation_ida.txt`](../data/ida/fd2_command32_35_presentation_ida.txt)、[`fd2_command34_tail_presentation_ida.txt`](../data/ida/fd2_command34_tail_presentation_ida.txt)、[`fd2_command35_tail_presentation_ida.txt`](../data/ida/fd2_command35_tail_presentation_ida.txt) | `RE-CLOSED`／`DATA-READY`／受限class19玩家`RUNTIME-E1`：唯一caller、#65..68效果、#91..94按ID音效、兩段滑入、main／11張可選tail、raw RGB插值、steady restore及四條command-specific tail已閉合。四個正式owner逐Draw消費共用段、0..40 map ramp及專用tail；ID34／35另逐段發布三個writer，中途失敗回復raw／HP／RNG／indexed buffers | IDs32–35一般玩家同狀態逐幀／逐音訊E2另列；score／EXP、AI與其他visual group仍失敗即關閉，不重做正式玩家owner |
| `0x2111A..0x211A4`／`0x1CAC7..0x1CD17`（ID32 command-specific tail） | [`fd2_command32_tail_presentation_ida.txt`](../data/ida/fd2_command32_tail_presentation_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：ID32 #6 `0x40..0x49`、#80 sample9、`0x4A/0x4B`四組90 ms切換、傷害後queue分流、bias `0x5E`與22張數字段均由正式玩家owner消費；HP／RNG只在切換後發布，尾停後才發布`Acted`。非靜音原始資產端到端及晚期rollback回歸已通過 | 精確同狀態逐幀／逐音訊與一般玩家E2另列；不重解tail函式 |
| `0x211A4..0x21206`（ID33 command-specific tail） | [`fd2_command33_tail_presentation_ida.txt`](../data/ida/fd2_command33_tail_presentation_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：函式硬編碼command13，正式玩家owner依序消費#66／#92共用段、#6 `0x39..0x3F`、#80 sample12／1、五組raw mask `0xC0`、`0x1C916(...,0x320)`、bias `0x69`及22張結果；不包含`0x21EB1`。mask後才發布HP／raw／RNG，尾停後才發布`Acted`，晚期錯誤可回復 | 精確逐幀／逐音訊、score／EXP、敵方owner與一般玩家E2另列；不重解tail函式 |
| `0x1A866`／`0x1B750`（transient 到期呈現） | [`fd2_transient_expiry_presentation_ida.txt`](../data/ida/fd2_transient_expiry_presentation_ida.txt) | `RE-CLOSED`：selector `1/0/2` 三個 caller；raw `+0x22..+0x27` 倒數／歸零；`sub_12D7B` 重畫、`sub_1956B(raw +7)` DATO 來源、`sub_15F84` FDTXT `0x1E1..0x1E6`（481..486）文字、`0x4E031` present／input、delay10、`sub_196CB` 關閉與 `sub_1B750` derived recalc 順序 | 正式 UI 已以目前 indexed map、raw +7 DATO、FDTXT 481..486 建立並在完整預建後原子發布；下一步只補精確 tick／音訊、狀態高階名稱與一般玩家 E2；status colors／icons另由 `0x17FC0` 主證據關閉，不猜六個 raw 欄位名稱、不重解 `sub_1A866` 函式本體 |
| `0x17FC0`（角色 status colors／icons） | [`fd2_status_panel_transient_indicators_ida.txt`](../data/ida/fd2_status_panel_transient_indicators_ida.txt) | `RE-CLOSED`／`RUNTIME-E1`：`+0x22..+0x24` 切換 digit base `0x2A/0x77`；`+0x25..+0x27` 非零時消費 FDOTHER #5 entries `0x37..0x39`；typed plan、indexed renderer、church status 正式 owner 與原始資產 regression 均已接 | 只補高階名稱、精確 tick／音訊與一般玩家 E2；不另造六個圖示、不重解函式 |
| `0x24618` | chapter-specific IDA 證據；例如 [`ch22`](../data/ida/fd2_ch22_pre_ida.txt)、[`ch27/28`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt) | indexed transition 核心與部分 caller payload | 新 caller 必須另證參數／view；不得把已知 callee 當全新未知 |
| `0x22253` | [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)、[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) | 共用11＋6＋10、18／24-row bridge、五參數 ABI；battle-state Ebiten presenter已由raw ch28 post及command23兩次離場／入場正式路徑消費，均達`RUNTIME-E1` | 其他 caller-specific focus／story-array adapter與同狀態E2；command23只補camera／逐幀驗收；callee及已閉合caller payload不重解 |
| `0x33F78`（最終戰前 staging wrapper） | [`fd2_ch29_staging_wrapper_ida.txt`](../data/ida/fd2_ch29_staging_wrapper_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：三參數 ABI 固定為 `slot,x,y`；wrapper 先執行 `0x12CEA(x,y)`，再執行 `0x22253(slot,x,y,x,y)`。七個 ch29 pre caller 已由正式 `story_ch30` binding 消費，story slot 只在既有 bridge 邊界發布 | 只補正常第29戰戰後→整備→最終戰前的一般玩家 E2、精確時序與音訊；不得重回已被直接指令否定的 `0x12CEA(slot,x)`，亦不重解 `0x22253` |
| `0x1088D`／`0x33DBA`／`0x35C79`／`0x35C32`／`0x35D60`／`0x35EE6`／`0x2548C`（map28） | [`fd2_map28_runtime_topology_ida.txt`](../data/ida/fd2_map28_runtime_topology_ida.txt)、[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) | 玩家第29戰入口20筆持續隊伍＋group8(56)=76；event75／74／76／79均已成為可編輯資料及正式 `RUNTIME-E1`。post前沿安全集合76／78／80／82／84／87、`0x35BBA→0x1DB65`原資源 presenter、group9＋`0x25535`、ch28 post binding、隊伍同步與 `preparation_ch30` 存讀檔均已達 `RUNTIME-E1`。groups2/3無已證實producer | 補未修改一般玩家逐幀／音訊 E2；高階圖像／sample語意仍unknown；event75／74／76／79僅在新反證下重開，event82只有新producer證據才可重開 |
| `0x24B14` | ch26 post 直接證據與 [`91`](91-worklist.md) | 天空之鑰 inventory gate | 兩臂視覺／效果；不重解搜尋條件 |
| `0x2415B`、`0x24182`、`0x2424B`、`0x24286`、`0x242C1`、`0x24308`、`0x2425F`、`0x2429A`、`0x24336` | [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt) | 玩家第21戰戰後26-slot layout、六組分支呼叫共30句原生對話、ACT63／64與天空之鑰固定演出；正式成功臂消費26句與完整演出，材料不足臂消費14句且不執行演出／授予鑰匙；兩臂均進城鎮／存讀檔 | 只補未修改原版同狀態 E2與第一個動態調色盤相位；不重解函式本體、對話呼叫組、layout或ACT resources |
| `0x24E80..0x25052`、`0x24F43`、`0x24F7E`、`0x24FC4`、`0x24FFF`、`0x2503A` | [`fd2_post26_28_dispatch_ida.txt`](../data/fd2_post26_28_dispatch_ida.txt)、[`fd2_ch25_post_native_dialogue_ida.txt`](../data/ida/fd2_ch25_post_native_dialogue_ida.txt)、[`ch25-post-native-dialogue-e1.json`](../data/ui-traces/ch25-post-native-dialogue-e1.json) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：玩家第26戰戰後由`event_state[12]`選`5/6`與`8/9`，共同`7/10/11`；七個tuple共41句原始control／operand已綁定固定版`FDTXT.DAT`第26項。正常57-slot正式勝利路徑分別只消費18或33句，再走ACTING77–80、同步隊伍、chapter26、`town_ch27`與全新`Game`冷讀。舊唯一70-slot斷言已撤回：正常重製入口為16部署者＋group0 41筆＝57，原函式只讀動態count | 70只保留完整資料形狀相容；不重解`sub_24E80`，未修改同狀態逐幀／音訊另列E2 |
| `0x25464..0x2548C`、`0x231DF..0x231F8`、`0x231E5` | [`fd2_post26_28_dispatch_ida.txt`](../data/fd2_post26_28_dispatch_ida.txt)、[`fd2_ch27_post_native_dialogue_ida.txt`](../data/ida/fd2_ch27_post_native_dialogue_ida.txt)、[`ch27-post-native-dialogue-e1.json`](../data/ui-traces/ch27-post-native-dialogue-e1.json) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：玩家第28戰戰後入口跳往真實共享尾端；`0x231E5#7`唯一消費`FDTXT_028`五句`FFEE`原生對話，再sync與chapter28。正常`story_ch28`讀入map27全60筆來源，但只有groups1..7共44筆與20部署者形成64-slot戰況；group255的16筆保留在source roster。正式勝利、五句具型別輸入、`preparation_ch29`與全新`Game`冷讀均通過；舊80-slot斷言已撤回 | 五句說話者螢幕座標未由handler保存，99%模式保留五階段收框／背景還原但省略無來源額外滑動；未修改同狀態逐幀／音訊另列E2，低位址及共享尾端不重解 |
| `0x24C1E`／`0x24D22`／`0x11EEE` case 23 | [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt) | raw ch23／玩家第24戰的 stage 2..14 先寫後畫、`[0x46c] != [0x539f8]` tick gate、312×192 row rotation、#42 staging、零 transient offset、indexed copy ABI 與正式 E1 adapter | 補未修改一般玩家同狀態逐幀／時序 E2；不得重開入口 latch 或把移動中 offset 外推到 handler |
| `0x24754`／`0x247B4`／`0x1088D`／`0x15F84`／`0x2189A`／`0x219AD`／`0x24B4D`／`0x10652`／`0x4DBFC`（raw ch22 post） | [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)、[`fd2_ch22_post_dialogue_binding.txt`](../data/ida/fd2_ch22_post_dialogue_binding.txt) | layout已閉合為table slots0..16＋special slot17、camera(14,14)；LOADCH先16個persistent slots再append map22 records，完整materialized frontier為16＋70＝86，舊70-slot說法撤回。三實參 unit/radius/step、FDOTHER #3 LUT0..9 與 `0x2189A` typed E1；FDFIELD #69、FDSHAP #46/#47 由 `0x11EEE` 直接消費；`0x24B4D` 的13×9 staging→steady draw→兩列交替30×20ms typed E1 已通過30幀與缺第九列零修改回歸；chapter23 `0x10652` 另建 FDOTHER #42／59904-byte staging，三次ACT73經`0x1366A→0x11CAC→0x11EEE`消費；完整四位元組raw grid已在State保存。十個`0x15F84` caller現把FDTXT_023 index8..17正確展開為56句，不再誤用整檔89句；四種raw control各自保存13／15／14／14 glyph上限。正式可達分支以具型別輸入完成35句後才沿原位置同步隊伍，再由記錄提示存檔、全新`Game`冷讀、15人選取、取消重選及最終肯定進`story_ch24`，達連續`RUNTIME-E1` | event52增援時序、高階畫面名稱、互斥分支的未修改原版動態路徑與一般玩家 E2仍另列；不重解已閉合 helper |
| `0x135DD`、`0x20421`、`0x4DFCC` | 同上及既有 palette／AFM 證據 | `0x24336` 使用的鏡頭移動、全螢幕 AFM 與高色階相位循環窄角色 | 新 caller 另證參數與時序；不可把 `0x20421` 誤稱音訊或把 `0x4DFCC` 推成一般調色盤 API |
| `0x2BCE5` | [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)、[`montage tail`](../data/ida/fd2_ch29_post_montage_tail_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：終局前綴、角色蒙太奇、20段尾段、定格與隊伍回顧已由正式`battle_ch30→ending`消費 | 只補一般玩家原版owner／E2與精確音訊；不重解已閉合前綴、尾段或重製handoff |
| `0x2C39B`／`0x1956B` | [`fd2_ending_dialogue_owner_ida.txt`](../data/ida/fd2_ending_dialogue_owner_ida.txt) | `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：19×5框caller、initial DATO portrait與FDTXT逐句speaker／pages已分層；正式ending逐Draw owner消費19×5 base、`0x1974C`六段opening、四列逐glyph、right-edge DATO mouth overlay與`0x2D31B`五段closing＋source restore，完成後才resume timeline。兩個chapter26文字閘門與chapter29五block原始資產預建均有回歸 | 只補精確時序及一般玩家E2；不得再把block `portrait_id`套給全部台詞，也不得借用ch24三列validator或一般RGBA對話框 |
| `0x28A6C` | [`35`](35-battle-animation-rendering.md)、[`montage tail`](../data/ida/fd2_ch29_post_montage_tail_ida.txt) | 共用雙 runtime-record renderer；直接 caller 含戰鬥、事件與終局 | 重製端冷讀 persistent raw 到定格／回顧已閉合；只補原版 `0x2C2A6` call-time 完整動態狀態、精確輸出與輸入，不得再稱終局專屬 callee |

只有四種情況可重開「已閉合範圍」：原版雜湊不同、原始指令或跳表直接反證、執行期
同狀態結果矛盾，或現有證據缺少它聲稱已具有的 writer／consumer。單純找不到舊筆記、
舊文件仍寫未知、IDA 自訂名稱不同、或 exporter 沒更新，都不是重開理由。

## 六、後續工作必須先分類

### 真正需要局部反組譯

- 依玩家可見 blocker 擴充既有 IDA 清冊的產品／runtime／driver 分類；不為了提高
  百分比替剩餘1,073筆未知函式猜名稱。
- postbattle admission 已無 blocked 節點；只在 E2 實驗出現指令／執行結果矛盾時，
  才重開對應 caller，不因舊 worklist 重讀已閉合 callee。
- 只有代表性抽樣揭露且會改變玩家結果的 command／spell／item 或高階效果缺口；
  沒有正常 producer 的指令不為提高覆蓋率而重開。
- `0x28A6C` 只在原版 `0x2C2A6` call-site 動態狀態與重製結果矛盾時重開；
  `0x2C2A6` 是 `sub_28A6C` 內的呼叫位置，不是另一個待解函式；
  `0x2BCE5` 的正式重製 owner／handoff 已接，剩餘是一般玩家原版 E2與精確音訊。
- global event table 58..89、event82 producer或其他 handler，只有在正常玩家抽樣實際
  到達並形成 blocker 時才局部重開。

### 不需要再反組譯，應轉實作或工具修正

- 維護 `native_call`／`unresolved_native_call`／`unknown` 三態與分級證據；目前
  60份 handler 匯出為93筆已分類呼叫、0筆 unresolved、0筆 unknown。caller 的
  E2與精確時序仍留在 scope，但不得再因此重解已閉合 helper。
- `0x14237`、`0x14EF0`、`0x15311`、`0x1548E`、`0x22253`、`0x2BCE5` 均已有
  caller／consumer 主證據；除非出現本檔定義的直接反證，不得由舊待辦重新開啟。
- 把已知資料接進正式 UI、campaign、save、audio consumer，補原子失敗與 regression。
- 依 [`57`](57-ui-evidence-matrix.md)完成戰場、指令、城鎮、教會、整備的輸入狀態機與畫面。
- 讓完整戰役保留戰後城鎮／商店／整備，不用直接跳下一戰的測試捷徑。

### 需要原版動態驗證，而不是更多靜態反組譯

- 同一 raw save／章節／回合下的原版與重製敵方回合配對。
- 玩家第29戰現有 fixed-hash 第三方存檔的未修改執行檔／普通 CONTINUE／指令環
  候選錨點；仍缺從該戰勝利連續走過 raw ch28 post、整備／存讀檔至第30戰的完整
  provenance E2。其餘章節祕密商店、有效槽 LOAD 與跨章 save/load 仍待抽驗；
  postbattle admission 已無 blocked 節點。
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
## 八、2026-08-25：`0x15055` 完整 target-list consumer（RE-CLOSED）

IDA Pro 9.4 已閉合 `sub_15055 (0x15055..0x15311)` 的 item winner 消費資料流：
它以 `[0x53C37/0x53C3B]` 重跑 `0x14818/0x149F8`，並把完整 target count／list
傳給 `0x20C6F`，不是只消費第一個 target。主證據見
[`fd2_ai_15055_item_target_list_ida.txt`](../data/ida/fd2_ai_15055_item_target_list_ida.txt)。
重開條件限於不同輸入雜湊、上述 call arguments 的原始指令反證，或同狀態執行
結果顯示 target list 順序不同；不可因 runtime 尚缺演出而重做此 RE。

## 九、2026-08-25：AI item type 5／13 `0x211A4` indexed owner

[`fd2_ai_item_restore_presentation_owner.txt`](../data/ida/fd2_ai_item_restore_presentation_owner.txt)
已把`0x15055→0x20C6F→0x211A4` caller鏈與既有尾段證據合併為非破壞性正式規格。
runtime現在只重用`0x211A4`固定command13尾段，不帶入指令33的`0x27FC9`／#66／#92
前導；完整預建7張effect、五組mask、post-state map、22張數字段及音效後才啟動。
HP／inventory／RNG在mask Draw後原子發布，尾停後才發布Acted；晚期取消會回復交易與
indexed buffers。原始資產端到端、未Draw不發布、晚期回復及缺資產失敗即關閉均達
`RUNTIME-E1`。type20／24的`0x1CD17`已由下一節接通；本家族下一步只補type21的
`0x1CAC7`，不重解
`0x211A4`。

## 十、2026-08-25：AI item type 20／24 `0x1CD17` indexed owner

[`fd2_ai_item_damage_1cd17_owner.txt`](../data/ida/fd2_ai_item_damage_1cd17_owner.txt)
以IDA Pro 9.4固定`0x20C6F` caller、command 0／2／3表項、`0x1CD17`十幀及
`0x4DC34`像素consumer。重要勘誤是`0x4DC34`第四參數來自
`0x52006[commandID]`，不是target record index；本切片三個command皆為raw `0x20`。
runtime先在detached transaction預建#6 `0x31..0x38`、十張blend、post-state map及
命中／失敗22張結果，最後blend Draw後才發布HP／death bit／RNG；type24 item79
原始資產正常AI入口證實來源不消耗、尾停後才發布`Acted`。缺素材、selector、raw
狀態或重入均零交易，晚期取消沿共同owner回復完整records／RNG／indexed buffers，
達`RUNTIME-E1`。type21的`0x1CAC7`保持獨立失敗即關閉，不由本排程泛化。

## 十一、2026-08-25：AI item type 21 `0x1CAC7` indexed owner

[`fd2_ai_item_type21_1cac7_owner.txt`](../data/ida/fd2_ai_item_type21_1cac7_owner.txt)
把既有`0x2111A／0x1CAC7／0x1CB94`主證據與正常item→command mapping合成
caller-specific契約，不重解已閉合callee。typed schedule只接受command1／6／7，
依raw表選8張sample6或10張sample9 effect，再以共用framebuffer核心預建
`0x4A→0x4B`四組toggle及22張命中／失敗結果。原始item38正常AI入口證實最後
toggle Draw後才原子發布HP／death bit／RNG，來源物品與MP保持不變，尾停後才
`Acted`；缺素材、selector、raw狀態或重入均零交易並可完整回復，達`RUNTIME-E1`。
至此正常正分type5／13／20／21／24的數值與caller-specific indexed演出均已接通；
後續只做代表性E2或新正常producer，不再以「缺物品演出」重開這三個tail。

## 十二、2026-08-25：玩家 item type 20／21／24 共用 `0x20C6F` indexed owner

上述兩份 canonical owner 的玩家 caller 補證確認 `0x1BE45→sub_20C6F` 與敵方入口
共用完整 raw target list 及 caller-specific indexed tail；`sub_1BBDC` 只在 callee
完整返回後才經 `sub_13512` 設 actor raw `+5` bit7。正式玩家確認入口因此已移除
同步直接改 HP／RNG／`Acted` 的捷徑，改由共同 detached transaction 預建並播放：
type20／24 在最後 `0x1CD17` blend Draw 後、type21 在最後 `0x1CAC7` toggle Draw 後
發布 HP／death bit／RNG，結果尾停後才發布 raw bit7 與 `Acted`。原始 item79／38
正常玩家路徑與既有敵方路徑皆通過；缺 indexed context 時保留目標模式、HP、RNG、
inventory 與 action，達 `RUNTIME-E1`。未修改原版同狀態逐幀／音訊仍是
`PLAYER-E2`，不因此重開兩個已閉合 renderer。

## 十三、2026-08-25：固定非玩家 command producer 覆蓋閉合

[`fd2_nonplayer_command_producer_coverage_20260825.txt`](../data/ida/fd2_nonplayer_command_producer_coverage_20260825.txt)
把固定雜湊 FDFIELD 的 33 圖、1887 筆 roster command mask 與既有
`0x13A9F→0x14EF0→0x1598A→0x15311` IDA 控制流程合併重算。排除不進 scorer 的
mode8 後，正常非玩家 producer 只有 ID0–7、9–18、20–22、26、27，全部已有
caller-specific indexed owner；ID8／19雖無固定初始bit，仍保留已閉合owner。
唯一ID30位於map13 index0 mode8，不構成command30 producer。正式AI admission
因此只接受ID0–22與26／27，並移除ID24／28／29／31借玩家derived-strike helper
同步改HP的捷徑；資產全量覆蓋與拒絕零mutation回歸均通過，達`RUNTIME-E1`。
這關閉「仍有正常敵方command producer缺owner」的舊待辦；剩餘戰鬥阻擋是狀態
高階名稱與代表性同狀態`PLAYER-E2`，不是再猜接靜態mask中不存在的ID。

## 十四、2026-08-28：FDTXT lossless token 與 FDOTHER#4 字型分離

本項不重開已閉合的offset table、`0x15F84`或控制碼語意。固定hash的35筆FDTXT已由
既有parser機械匯出：#0..#33為typed glyph/control JSON，#34零長度保持blocked；
`FDTXT_031`的46條逐word與原始resource相同。`FDOTHER#4`的1,824個16×16 1-bit glyph
亦由PNG atlas逐bit重建一致。UTF-8只是可編輯投影，runtime以lossless tokens為準並拒絕
兩者不一致，故不把未證實FFxx高階名稱寫入資料契約。

LOAD、職業／教會共用資產、戰場姓名、party montage與終局phase0已改讀分離
FDTXT#0/#31及字型，達`DATA-READY`／局部`RUNTIME-E1`。共用道具／轉移面板仍直接消費archive，
所以不得宣稱FDTXT production caller歸零；下一步只遷移這些已知consumer，不重做
文字格式RE。

### 2026-08-28 共用道具／狀態panel接線勘誤

上一段的共用道具／轉移缺口已由後續同日切片關閉。`FDOTHER#5`只准入具caller證據的
58筆entry，按opaque high-run、raw opaque與four-mode frame三種codec分離輸出；全部
逐pixel／mask一致。共用loader同時取得分離FDTXT#0與FDOTHER#4，已接玩家／敵方指令、
教會、商店、整備及道具panel。正式玩家路徑不再呼叫FDTXT archive reader；保留的raw
adapter只供source oracle。此項達`RUNTIME-E1`，其餘80筆FDOTHER#5 entry仍須按consumer
另立切片，不可泛化。

### 2026-08-28 FDICON.B24 全銀行分離

不重開`0x11019`／`0x127e0`已閉合selector ABI。固定hash檔案header為24×24、1,680張；
每張four-mode資料分離成indexed frame、source-write mask與destination-remap mask，合計
5,041檔並逐張三層一致。戰場`loadNativeMapAssets`正式改讀strict separated bank，旁側
archive刻意缺FDICON.B24仍載入成功；整備、職業／教會、城鎮與終局tail baseline亦改接
同一strict bank，正式產品程式的FDICON archive caller歸零，達`RUNTIME-E1`。匯入器與
名稱明確的source-oracle adapter仍保留。不以本項宣稱FDFIELD／FDSHAP或整張戰場畫面E2。
