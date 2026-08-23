# 58 — `FD2.EXE` 反組譯覆蓋與重製閉合矩陣

> 更新基準：2026-08-22 工作樹。這是判斷「還要不要反組譯」與「重製還缺哪一層」
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
| 開機、標題、LOAD、CONTINUE、存檔 | 部分 | 部分 | 部分 E1 | 原版錨點部分 E2 | 四槽 envelope、checksum、名冊與部分戰間落點已接；標題 selector 的正式確認 owner 現以 checksum-valid 合成槽完整還原 `town_ch02`、typed/raw party、gold、chapter、HUD gate並清除舊 battle state，竄改 envelope 原子留在選槽。current-runtime CONTINUE 與巢狀 SAVE／LOAD 已有正式 E1：LOAD 先完成私有候選 handoff，YES 後才原子替換；SAVE 保留四槽與未命名 bytes，非我方增援不占 persistent slot，具 constructor／identity 證據的新我方 JOIN 追加完整 raw record。尚缺未修改有效四槽與同 raw 狀態 E2；長程遊玩改由使用者人工回報，不列代理工作項目。 |
| 對話、頭像與過場原語 | 部分偏高 | 部分就緒 | 部分 E1 | 部分 | `0x15F84`、`0x1366A`、基本 pan／acting／spawn／join 等不應再從零重解。仍須處理 caller-specific layout、indexed renderer、文字分支與實際 chapter binding。 |
| 30 個 raw chapter 的戰前／戰後處理器 | 部分 | 60 份 handler script；部分 binding | 部分 E1 | 缺完整 E2 | 舊83個 raw unknown 已拆成80個已證實窄呼叫、3個已知但 caller／執行期未閉合的呼叫，已沒有未分類 call site。玩家第29戰 raw ch28 post 現已以綁定的視圖／HUD、`0x35BBA→0x1DB65`、group9、`0x22253`、`0x24B4D`、`0x35E5A`、隊伍同步與 `preparation_ch30` 存讀檔達成 E1；未證實高階圖像／樣本名稱與一般玩家 E2 仍保留。 |
| 可編輯戰役與持續隊伍 | 部分 | 121 個 story／cutscene 節點；9 個 scripted、56 個 handler-bound、56 個 fallback | 部分 E1 | 缺完整 E2 | 24 個 postbattle 節點目前全部 active；admission blocked 為0。玩家第29戰正常 `story_ch29→battle_ch29` 入口現物化76-slot frontier與已證實視圖／HUD，戰果確認後播放 raw ch28 post，追加group9、同步持續隊伍，再進`preparation_ch30`並通過存讀檔 E1。所有 active 仍只代表正式執行期接入，不代表未修改原版 E2 或逐像素一致。 |
| 戰鬥資料、移動、公式、勝敗與成長 | 部分偏高 | 部分就緒 | E1 | 部分 | 多項公式與地形資料已有具型別實作；命中／閃避來源、部分經驗交易、回合事件與原版逐狀態驗證仍不完整。需要針對缺欄位補 producer／consumer，不重解已閉合的 AP−DP 等公式。 |
| 玩家指令、法術、物品與交易 | 部分 | 部分 | 部分 E1／部分失敗即關閉 | 缺完整 E2 | command mask、若干 ID、MP／物品交易與 selector 邊界已解；物品第一階段 raw target field 及 drawable selector 1..5 現由正式 `0x11CAC` 索引畫面消費，缺原生 HUD／LUT／range sprite 時不再顯示綠／青／橘色猜測後備層。未修改第一戰存檔另以正常輸入證實四名我方皆滿 HP 時，item 192 草藥確認後至少4.3秒仍留在物品面板；type 5 現只在已證實候選中存在 `HP < MaxHP` 時發布 target modal，形成窄原版 `PLAYER-E2`／重製 `RUNTIME-E1`。這不包含受傷狀態原版 target modal、其他 item type、indexed item effect、disabled target 外觀、原版取消鍵或 global selector 6 owner。共用 `0x117E7` 在 `0x12C0D==-1` 時進 `0x16F55`。direction3→END 現以原版 DATO #75、FDOTHER #5/#2、FDTXT `0x1A3/0x1A4/0x19C` 跑完6＋4展開、YES／NO、4＋5收合、接受／取消逐字形回覆、來源復原與十二個60 Hz畫格近似，只有YES才進`0x1A30B`；普通鍵盤正式路徑與原版密集擷取已形成動態配對，精確時序／音訊仍未閉合。缺任一資產時在命令框關閉前拒絕。command 13–16 的 `0x21EB1→0x22046` 16張 FDOTHER #3 LUT 演出已轉成 typed schedule，玩家與敵方 mode 11 正式入口均先演出再交易；AI 依 `0x15311` 在移動後重建 raw target array。command 17–19 的 raw modifier transaction已由玩家與敵方mode 11正式消費；ID17依原版由record18扣MP。玩家command 17–22均依`0x1D6C8`先播放#88 sub0與八個commandColor／black DAC phases，第八個Draw acknowledgement後才發布交易；20／21才清`+0x25/+0x26`並借record10 restore，22才經class／RNG gate寫`+0x27`。command 24 的正常 selector32 路徑現依資源98的15幀 raw schedule，在frame4發布MP並播放FDOTHER #53 sub3，在frame10發布單一完整傷害並播放sub2，最後一幀才標記行動完成；兩標記間依raw terrain control選BG，播放`0x29C90`兩段各10次的640-stride viewport滑動。`0x2A289→0x18C6D`的entry22框、HP／MP bar、數字與raw姓名亦已接進actor／target indexed base，轉場不再重疊RGBA雙panel。缺原始FIGANI、BG、terrain control、target idle、palette、panel cells／font／text、sample或raw selector即零交易。升級學習端亦已修正為`unit+7→growth byte10 learn_idx→command_learn`，不再誤用portrait直接查表。`sub_2B659` actor base、扣MP後source snapshot與`sub_2B9A1` target idle reset已接；`sub_29164`九段雙分支角色／TAI滑入與DAC減算已接；精確音訊與一般玩家E2仍缺，故只列partial。AI不套用玩家palette owner。缺baseline／DAC／table／sample／records／target／MP／RNG時交易不發生。ID33／34／35現只對具raw class19及selector4／5／6／7／20的玩家來源開放：record33／34／35分別做52／28／36 MP gate，但已證實來源均不在此分支扣款；33在私有records清`+0x25..+0x27`後以固定`0x320`走`0x211A4`回復，34依`0x22721→0x22866→0x22997`完成三段，35依`0x22D1B`以command26／22／27及`+0x25/+0x27/+0x26`完成三段。三者都只在全部stage成功後原子發布，正式command grid→target confirm回歸已通過。這只關閉state transaction，`0x27FC9` indexed presentation、score／EXP、ID32、AI／其他visual group仍失敗即關閉。其他未知 command、狀態高階名稱、精確 DOS tick／音訊與完整 E2 仍未閉合；phase-expiry caller 與其 FDTXT／DATO／redraw／recalc 消費順序已由 [`fd2_transient_expiry_presentation_ida.txt`](../data/ida/fd2_transient_expiry_presentation_ida.txt) RE-CLOSED，selector 1→0／2 的倒數、歸零重算與 raw 同步已達 `RUNTIME-E1`，indexed 到期訊息已達 `RUNTIME-E1`；status colors／entries `0x37..0x39` 已由正式角色面板消費；精確 tick／音訊、高階名稱與一般玩家 E2 仍待。 |
| 指令28／29／31校正 | 閉合caller分歧；28／31正常取得來源未見 | command29資產與typed schedule就緒 | 28／31校正後數值E1；29玩家indexed owner E1 | 29缺E2；28／31非阻擋 | `0x276EC`已固定三支renderer／分母分歧。command29已由正式玩家confirm消費selector34／resource104並逐target原子發布。另以IDA固定command-mask OR writer只有level-up direct caller；固定learn table與32筆player defaults都不授予28／31，故「一般玩家無已證實取得來源」列強推論，不猜selector、不把它們當交付阻擋，也不冒稱死碼。主證據見[`fd2_command28_29_31_presentation_ida.txt`](../data/ida/fd2_command28_29_31_presentation_ida.txt)與[`fd2_command28_31_reachability_ida.txt`](../data/ida/fd2_command28_31_reachability_ida.txt)。 |
| 敵方人工智慧 | 底層控制流部分偏高；高階交易部分 | mode／候選／部分 fallback 已資料化 | 多個窄 E1 consumer | 只有原版敵方回合邊界 E2 | `0x13FD4`、`0x14EF0`、mode 5／11 等既有函式邊界與窄 owner 不應反覆重解。command 17–19 現已有 raw-selector target array→原子modifier transaction→AI成功動作邊界；selector 1→0 phase-expiry caller 也已依原順序接在合併的非玩家排程前。仍缺其他未接 command／item transaction、到期提示精確 tick／音訊、同一 raw 狀態的重製端配對與一般玩家效果。詳見 [`11`](11-enemy-ai.md)。 |
| 戰場 HUD、指令格、輸入與戰鬥演出 | 部分 | 部分 | 部分 E1 | 少量畫面 E2／多數缺少 | 有 native frame、command overlay、姓名字模、命中色盤與部分 FIGANI consumer。2026-08-22 已以同一未修改存檔由標題正常操作至悠妮 command 0 目標模式：原版四相位動態 LUT 為窄 `PLAYER-E2`，重製普通 X11 路徑達同座標／ID modal 為 `RUNTIME-E1`；時鐘相位未同步，故不是逐像素 parity。command 0 現由正式 Game confirm 接入 `0x2A6BD→0x29164→0x2B659→0x26152` 的完整預建與逐 Draw 發布：九段滑入、施術者效果、28 幀／7 元素錯開目標效果、七段 HP 與 LUT 尾段均達 indexed `RUNTIME-E1`，缺素材或 raw provenance 時在 MP／HP 前失敗即關閉。仍缺 #82 精確取樣率的人耳驗證及原版／重製同狀態逐幀、逐音訊 E2。整體操作狀態機、圖示可用性、其他 commands、相同戰況及演出時序仍未完成。完成度只由 [`57` 介面矩陣](57-ui-evidence-matrix.md)判定。 |

2026-08-24 補證：`0x525AF` 是 command 0..9 的 HP 分段除數表；typed
傷害計畫已不再把 command0 的七段套給全部 ID。command6 使用五段，並通過
決定性發布／越界拒絕回歸；其 12 張 target compositor 尚未接正式 Game owner，
故仍維持 `DATA-READY`，不得提升為畫面 `RUNTIME-E1`。

同日續補：command6 的 7 張前導 orbit、12 張 target、7 張尾段 orbit 已全部
轉成具型別 layer／sequence 與 indexed compositor，包含 side 分流及前五個
marker 才發布 HP 的契約。正式 Game owner、音訊時序與正常 producer 尚未接，
狀態仍是 `DATA-READY`。
| 城鎮、祕密商店、商店、教會與整備 | 部分偏高 | 部分就緒 | 多個正式 E1 consumer | ch02 若干狀態 E2；其餘部分 | 個別 menu、購買、賣出、轉移、復活、轉職與整備已有窄切片；ch02 賣出已由正常商店輸入走完角色／物品／Yes-No、成功、向上金幣滾動及返回名冊，九組 route-patched 原版／正式重製畫面皆整幀 AE=0。獨立裝備 service2 另由正常商店輸入取得名冊與索爾面板，但動畫相位未同步，整幀仍為 AE=1389／1433；正式交易現先在私有 unit 完成 raw 裝備、重算與 panel 重建，最後才發布，深層 renderer 失敗不再污染 roster／能力／既有 panel，達 `RUNTIME-E1`。service3 物品轉移除五個選擇狀態外，現也由正常商店輸入完成索爾短劍→悠妮，返回 loop 後原版索爾只剩皮甲／藥草、悠妮追加未裝備短劍；四個成功交易畫面 AE=1391／82／2／286。可見內容與幾何一致，剩餘差異是角色、翻頁箭頭或選取脈動相位，故仍列 route-patched partial E2。重製同一跨角色交易又穿越 `town_ch02` JSON 冷讀檔，保存雙方 compact/raw 背包、裝備、能力、金幣與隊伍順序。目的名冊取消現由正式 owner 依五幀收合→來源提示六幀展開發布，取消不改角色／金幣；取消後重新進入的 self-transfer 仍走既有 raw remove→append／重算，達 `RUNTIME-E1`。裝備收件者以六名具完整 raw provenance 的 typed party，從正式 menu→purchase→Yes 進三列面板，走過 scroll、滿欄／無合適角色原子返回與成功裝備／扣款，亦達 `RUNTIME-E1`；這些都不是原版 E2。教會轉職與其他商店 mutation 也能返回 town，再穿越重製 JSON 存讀檔。第25戰後的 `town_ch26` 祕密商店 E1 亦已接通。這些不等於未修改一般玩家戰間 E2；其餘章節入口、原版存檔、recipient scroll／no-recipient／full、service2 原版 mutation／restore畫面、transfer empty／full、self／destination-cancel 的原版同狀態畫面、church caller及完整交易仍缺。 |
| 音樂與音效 | 格式閉合；owner／時序部分 | 部分就緒 | 部分 E1 | 逐音訊 E2 缺少 | XMIDI、兩類音源與部分曲目／樣本 owner 已知。重點是精確播放時機、停止／切換、效果同步與三平台播放，不需要重解 XMIDI 格式。 |
| 終局與結局 | 部分偏高 | 來源約束排程已資料化 | E1；正式 `battle_ch30→ending` 已接終端定格 | 缺一般玩家終局 E2 | `0x2BCE5` 前綴、`0x2C548` 角色蒙太奇、20段尾段與 FDOTHER #59 定格已由正式 campaign 以原始資源與持續 raw roster 消費；資產或 provenance 不足時整批失敗即關閉。20段實際80個 FIGANI 已證實全部 header byte1=0；runtime 現逐 raw `+6` inner present、`+7 bit0` 層序、`+4` 位移／palette33與最後 effect 終止、base scheduler 執行兩次交叉配對。尚缺 caller `0x2C2A6` 當下 records／globals 動態連續性、3% RNG重播、精確音訊／終端輸入與原版 owner，不是重解整個 `0x28A6C`。 |
| DOS/4GW、Watcom runtime、Miles 驅動與一般函式庫 | 第一輪分類 | 不適用 | 只在行為外露時處理 | 不適用 | IDA 清冊1305函式中170筆由 Watcom FLIRT 標成 runtime；其餘未分類不能都算產品程式。後續只擴充分級索引，不把函式庫未命名算成 remake 缺口。 |
| 三平台打包與推廣片 | 不適用 | 部分 | 尚未達發行閘門 | 缺完整玩家驗收 | 這不是反組譯問題。待核心戰役、操作 UI、結局與代表性一般玩家路徑關閉後，再做 Linux／Windows／macOS 打包與影片。 |

**2026-08-23 ID32 現況勘誤**：上表「玩家指令、法術、物品與交易」列尾端仍保留
本批開始時「ID32失敗即關閉」的快照文字；該單一敘述已由
[`fd2_command32_transaction_ida.txt`](../data/ida/fd2_command32_transaction_ida.txt)
取代。ID32現與33／34／35同為受限class19玩家state transaction `RUNTIME-E1`，
正式grid→confirm已接；仍失敗即關閉的是`0x1C4CC→0x1CAC7` indexed演出、
`0x1E0DB/0x1E1DC`結果畫面、score／EXP、AI、其他visual group與一般玩家E2。

**2026-08-23 敵方物理提交補強**：mode 2與mode 11 `0x1548E`正式入口現先預檢
攻方attack FIGANI、守方idle FIGANI與descriptor delay，全部可建立排程後才消耗
RNG並發布HP／死亡獎勵；缺素材維持零交易。另確認敵方17–19由`0x15311`直接進
effect table，原版不經玩家專用`0x1D6C8` palette owner，故不把玩家八相位演出
誤接到AI。此為`RUNTIME-E1`原子邊界，仍缺正常敵方回合同狀態E2。

**2026-08-23 敵方 command0 演出接線**：`0x15311`在ID `<10`且
`Raw53AF9==0`時進`0x2A6BD`；正式敵方ID0已改用既有完整
`0x2A6BD→0x26152` presenter，不再直接發布state-only傷害。預建失敗時MP、HP、
RNG與`Acted`均不變；完成後才交給敵方continuation與死亡獎勵。ID1–8的
`funcs_2AC25`不能外推ID0專屬演出。ID1現已閉合`0x262EF`的八槽位移、
mode4→target→mode5順序與FDOTHER #19/#21 30-frame資源，並有純indexed
compositor及原始資產回歸；勘誤：此entry沒有直接sample call。它只到
`DATA-READY`，因外層MP／HP／音訊發布順序與正常producer E2未閉合，正式AI仍是
既有state-only數值路徑而非完整演出；不得把「數值可執行」誤寫成「renderer
失敗即關閉」或`RUNTIME-E1`畫面完成。

ID2的`0x26528`亦已修正為自身state `0x53F7E..0x53F80`、#26/#27
18-frame FIGANI與#83 samples1／2／3；`0x52460/0x52490/0x5249C`實為ID3
資料，舊ID2關聯已撤回。mode4→target→mode5單張純indexed compositor與原始
資產回歸達`DATA-READY`；mode1／2／7／8 helper座標、外層交易及正常敵方回合
仍未閉合，不宣稱正式演出。

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

2026-08-21 以唯讀 Docker 實跑現有稽核：

- IDA Pro 9.4 對固定雜湊 `FD2.EXE` 辨識1,305個函式；受版控語意索引有32筆，
  其中31筆屬產品程式、另1筆與 FLIRT runtime 分類重疊；去重後清冊仍為
  產品31、runtime170、未知1,104。`0x37416` 同時由 FLIRT 命名為
  `free` 並由索引標成 runtime；沒有因此重複計數。
- 60份 raw handler script 原有83個 `unknown` call site、23個 target。重生後為
  80個 `native_call`（22個 target）與3個 `unresolved_native_call`
  （`0x22253` 1次、`0x2BCE5` 2次），真正 `unknown` 為0。每筆
  具名呼叫皆保存原始位址／PUSH 順序、推論等級與證據檔；編譯器仍逐 caller
  驗證並失敗即關閉，故這是**工具債清除**，不是玩法完成率。
- `campaign_full.json`：121 個 story／cutscene 節點；9 個 scripted、56 個
  handler-bound、56 個 fallback。fallback 可能是撤退、傳聞或尚未接線故事，不能全部
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
| `0x15F84` | [`29`](29-remake-extensible-event-system.md)、各 handler 直接指令 | 對話呼叫與索引基礎 | caller-specific 分頁、版面與 E2；不重解基本函式角色 |
| `0x1366A` | [`50`](50-cutscene-script-system-design.md)、`chapter_beats` | acting 呼叫原語 | 個別資源、場景時序與畫面；不重解原語本身 |
| `0x11DF2` | [`fd2_11df2_palette_disasm.txt`](../data/fd2_11df2_palette_disasm.txt) | palette range/delta helper | caller 時序與畫面；應回填 exporter |
| `0x1F882`、`0x25052` | [`91`](91-worklist.md) 對應直接指令證據 | 兩種不同 palette ramp | runtime renderer／caller E2；不再稱 vsync |
| `0x13FD4` | [`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt) | raw gate、回復與窄 presentation owner | 同狀態交易／逐幀逐音訊 E2；不重解函式邊界 |
| `0x14EF0` | [`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt) | producer 順序與尾端 dispatch | 未知 command／效果／完整 transaction；不重解既有 dispatch |
| `0x14237` | [`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt) | 物理候選評分窄切片 | 完整 planner、target transaction 與 E2 |
| `0x15311`／`0x1548E` | [`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt) | mode 11 兩段 owner／順序 | 未知 command、完整演出與 E2 |
| `0x16F55`、`0x1728C`、`0x117E7`、`0x4E8E1` | [`fd2_system_overlay_options_ida.txt`](../data/ida/fd2_system_overlay_options_ida.txt)、[`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt)、[`empty cursor owner`](../data/ida/fd2_117e7_empty_cursor_system_overlay_ida.txt)、[`END確認框`](../data/ida/fd2_end_turn_confirmation_ui_ida.txt)、[問句同狀態比較](../data/ui-traces/native-end-turn-confirmation-original-vs-remake-e1.json)、[逐字回覆動態配對](../data/ui-traces/native-end-turn-response-progressive-original-vs-remake-e1.json) | `0x12C0D==-1` 空游標 owner；direction2設定、direction3 END及direction1全軍移動均已接正式E1；END另有DATO #75／FDOTHER #5/#2、FDTXT `0x1A3/0x1A4/0x19C`與`0x1A30B`生命週期；`0x4E8E1`由右往左寫入且`0x9017`是右端錨點 | 精確 DOS tick／音訊及收合逐幀同相位、selector0巢狀current-runtime存讀檔、逐章重製PLAYER-E2；正式資料的selector1 event61／75已接通，其他未具owner的90-entry分派仍失敗即關閉；亦不重解設定writer／consumer、END分支與renderer |
| `0x19DF7`、`0x1B1E7`、`0x16FF4` | [`fd2_nested_system_menu_ida.txt`](../data/ida/fd2_nested_system_menu_ida.txt)、[`fd2_system_exit_and_group_march_ida.txt`](../data/ida/fd2_system_exit_and_group_march_ida.txt) | `RE-CLOSED`、`DATA-READY`、`RUNTIME-E1`：nested資訊／存檔／讀檔／離場四分派及四個正式 owner 已接。selector2 LOAD 使用 FDTXT `0x19D/0x19E/0x19C`，在私有 `Game` 完成 current-runtime typed handoff，YES 回覆生命週期後才原子替換；selector1 SAVE 從完整 raw baseline 複製未知 bytes，並只接受有 provenance 的 live／新 JOIN 欄位。外層 selector1 全軍移動使用既有 `0x51B91` 表；event61／75 在途中完成各自已證實事件後續行，未知事件在發布前整批拒絕 | 尚缺未修改同狀態逐幀／tick／音訊 E2；長程存讀檔由使用者人工回報問題。正式資料只有 event61／75 兩筆 selector1 rule；不得外推成90-entry全完成，也不得重解四分派、資訊 schedule、selector3目的地或 `0x51B91` 表身 |
| `0x21AD9`／`0x21B18`（command 13–16 wrapper 家族） | [`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt) | 四個 command literal、wrapper 參數、共同 indexed presentation owner、玩家／AI callers及正式 E1 | 補同狀態逐幀逐音訊 E2；不重解 wrapper |
| `0x21EB1`／`0x22046`（command 13–16 LUT 演出） | [`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt) | FDOTHER #3 LUT provenance、16張排程、visible-cursor中心、兩段200 ms、sample index11、compositor consumer及玩家／敵方 E1 | 補同狀態逐幀／逐音訊 E2；不重解 loop |
| `0x1C4CC`／`0x1C2DA`／`0x1E0DB`／`0x1DF58`（command 13–16 後段） | [`fd2_command_numeric_tail_ida.txt`](../data/ida/fd2_command_numeric_tail_ida.txt) | FDOTHER #6七幀、五組snapshot→mask、`0x4DDD7` write mask、transaction後redraw、FDOTHER #5 queue／22-frame reader與玩家／敵方 E1 | 補同狀態逐幀／逐音訊 E2；不重解函式 |
| `0x1D6C8`（command 17–23／25–27 玩家 palette owner） | [`fd2_command_modifier_palette_ida.txt`](../data/ida/fd2_command_modifier_palette_ida.txt) | 唯一caller `0x1CFF0`、#88 sub0、三張36-byte DAC table、四輪color／black；17–23與25–27玩家正式E1均先演出後交易，23並串接兩次`0x22253`離場／入場 | 只補command23同狀態camera／逐幀、精確tick與逐音訊E2；phase-expiry與status panel已由後列主證據關閉；不重解palette loop或`0x22253` callee |
| `0x1A866`／`0x1B750`（transient 到期呈現） | [`fd2_transient_expiry_presentation_ida.txt`](../data/ida/fd2_transient_expiry_presentation_ida.txt) | `RE-CLOSED`：selector `1/0/2` 三個 caller；raw `+0x22..+0x27` 倒數／歸零；`sub_12D7B` 重畫、`sub_1956B(raw +7)` DATO 來源、`sub_15F84` FDTXT `0x1E1..0x1E6`（481..486）文字、`0x4E031` present／input、delay10、`sub_196CB` 關閉與 `sub_1B750` derived recalc 順序 | 正式 UI 已以目前 indexed map、raw +7 DATO、FDTXT 481..486 建立並在完整預建後原子發布；下一步只補精確 tick／音訊、狀態高階名稱與一般玩家 E2；status colors／icons另由 `0x17FC0` 主證據關閉，不猜六個 raw 欄位名稱、不重解 `sub_1A866` 函式本體 |
| `0x17FC0`（角色 status colors／icons） | [`fd2_status_panel_transient_indicators_ida.txt`](../data/ida/fd2_status_panel_transient_indicators_ida.txt) | `RE-CLOSED`／`RUNTIME-E1`：`+0x22..+0x24` 切換 digit base `0x2A/0x77`；`+0x25..+0x27` 非零時消費 FDOTHER #5 entries `0x37..0x39`；typed plan、indexed renderer、church status 正式 owner 與原始資產 regression 均已接 | 只補高階名稱、精確 tick／音訊與一般玩家 E2；不另造六個圖示、不重解函式 |
| `0x24618` | chapter-specific IDA 證據；例如 [`ch22`](../data/ida/fd2_ch22_pre_ida.txt)、[`ch27/28`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt) | indexed transition 核心與部分 caller payload | 新 caller 必須另證參數／view；不得把已知 callee 當全新未知 |
| `0x22253` | [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt)、[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) | 共用11＋6＋10、18／24-row bridge、五參數 ABI；battle-state Ebiten presenter已由raw ch28 post及command23兩次離場／入場正式路徑消費，均達`RUNTIME-E1` | 其他 caller-specific focus／story-array adapter與同狀態E2；command23只補camera／逐幀驗收；callee及已閉合caller payload不重解 |
| `0x1088D`／`0x33DBA`／`0x35C79`／`0x35C32`／`0x35D60`／`0x35EE6`／`0x2548C`（map28） | [`fd2_map28_runtime_topology_ida.txt`](../data/ida/fd2_map28_runtime_topology_ida.txt)、[`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) | 玩家第29戰入口20筆持續隊伍＋group8(56)=76；event75／74／76／79均已成為可編輯資料及正式 `RUNTIME-E1`。post前沿安全集合76／78／80／82／84／87、`0x35BBA→0x1DB65`原資源 presenter、group9＋`0x25535`、ch28 post binding、隊伍同步與 `preparation_ch30` 存讀檔均已達 `RUNTIME-E1`。groups2/3無已證實producer | 補未修改一般玩家逐幀／音訊 E2；高階圖像／sample語意仍unknown；event75／74／76／79僅在新反證下重開，event82只有新producer證據才可重開 |
| `0x24B14` | ch26 post 直接證據與 [`91`](91-worklist.md) | 天空之鑰 inventory gate | 兩臂視覺／效果；不重解搜尋條件 |
| `0x24336` | [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt) | 玩家第21戰戰後天空之鑰固定演出：`FDOTHER #34`、`ANI #0`、調色盤與延遲順序；正式重製端 E1 已消費 | 補未修改原版同狀態 E2、第一個動態調色盤相位及相鄰 `layout_units`／ACT63／64；不重解函式本體 |
| `0x24C1E`／`0x24D22`／`0x11EEE` case 23 | [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt) | raw ch23／玩家第24戰的 stage 2..14 先寫後畫、`[0x46c] != [0x539f8]` tick gate、312×192 row rotation、#42 staging、零 transient offset、indexed copy ABI 與正式 E1 adapter | 補未修改一般玩家同狀態逐幀／時序 E2；不得重開入口 latch 或把移動中 offset 外推到 handler |
| `0x247B4`／`0x1088D`／`0x2189A`／`0x219AD`／`0x24B4D`／`0x10652`／`0x4DBFC`（raw ch22 post） | [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt) | layout已閉合為table slots0..16＋special slot17、camera(14,14)；LOADCH先16個persistent slots再append map22 records，完整materialized frontier為16＋70＝86，舊70-slot說法撤回。三實參 unit/radius/step、FDOTHER #3 LUT0..9 與 `0x2189A` typed E1；FDFIELD #69、FDSHAP #46/#47 由 `0x11EEE` 直接消費；`0x24B4D` 的13×9 staging→steady draw→兩列交替30×20ms typed E1 已通過30幀與缺第九列零修改回歸；chapter23 `0x10652` 另建 FDOTHER #42／59904-byte staging，三次ACT73經`0x1366A→0x11CAC→0x11EEE`消費；完整四位元組raw grid已在State保存。正式binding已由一般戰果確認進`preparation_ch24`並通過save/load E1 | event52增援時序、高階畫面名稱與未修改一般玩家 E2仍另列；不重解已閉合 helper |
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
- postbattle admission 已無 blocked 節點；只在 E2 實驗出現指令／執行結果矛盾時，
  才重開對應 caller，不因舊 worklist 重讀已閉合 callee。
- 未知 command／spell／item transaction 與高階效果 owner。
- `0x28A6C` 精確 renderer、終端輸入、`0x2BCE5` 正式 owner／handoff。
- global event table 58..89、event82 producer，以及其他會阻擋一般玩家流程的 handler。

### 不需要再反組譯，應轉實作或工具修正

- 維護 `native_call`／`unresolved_native_call`／`unknown` 三態與分級證據；目前
  handler 匯出已沒有真正 `unknown` call site，剩餘3筆只追 caller／執行期 gate，
  不重解已知 helper。
- 把已知資料接進正式 UI、campaign、save、audio consumer，補原子失敗與 regression。
- 依 [`57`](57-ui-evidence-matrix.md)完成戰場、指令、城鎮、教會、整備的輸入狀態機與畫面。
- 讓完整戰役保留戰後城鎮／商店／整備，不用直接跳下一戰的測試捷徑。

### 需要原版動態驗證，而不是更多靜態反組譯

- 同一 raw save／章節／回合下的原版與重製敵方回合配對。
- 玩家第29戰 raw ch28 post 與其餘章節祕密商店的一般玩家 E2、有效槽 LOAD
  與跨章 save/load；postbattle admission 已無 blocked 節點。
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
