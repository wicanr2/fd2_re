# 57 — UI evidence matrix（SDD-1 baseline，2026-07-25）

> **文件責任（2026-08-12）**：本檔是玩家可見 UI、輸入、畫面與 E2 差距的唯一
> 狀態表。整體 `FD2.EXE` 反組譯、可編輯資料與正式執行期覆蓋改由
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)統一判定；不能用本檔
> 某個畫面已達 E2，推成整個子系統或戰役完成。

> 這是 SDD 的第一份可執行盤點，不是「已還原」宣告。行號以本輪 `remake/cmd/fd2/main.go` 為準；`partial`／`missing` 必須先補 E0/E1/E2 證據才可改成 verified。

## 2026-07-28 visual-parity audit

這一節回答的是「玩家目前看到的操作畫面與原版相差多少」，不能與
codec、RE 函式數或可編譯測試數混算。分數是依 repo 內 DOSBox／錄影
oracle、目前 source rebuild 截圖、indexed fixture，以及外部原版畫面逐項
審查後的工程估計；它不是 pixel-diff 百分比，也不是遊戲總完成度。

| 畫面／流程 | 視覺還原估計 | 直接證據與主要差距 |
|---|---:|---|
| title/main menu | 穩定選單100%；完整開場E1 | 2026-08-26 由目前原始碼重新建置並以 Docker／Xvfb 擷取 [`title-remake-runtime.png`](../figures/title-remake-runtime.png)，最近鄰縮回320×200後與 DOSBox oracle [`title-original-dosbox.png`](../figures/title-original-dosbox.png) 達整幀 `AE=0/64000`；三列原生座標為164／173／182。`sub_1F894` 的535列捲動、六個插播點與 AFM 3→4→5→6→7→8→0→1 已依 canonical IDA 證據接入。2026-08-27再把AFM 3前的`FDOTHER #74 + #76`漢堂發行商畫面接入119幀60Hz近似；正式runtime第60幀與解碼oracle達640×400 `AE=0/256000`，見 [`title-publisher-remake-e1.json`](../data/ui-traces/title-publisher-remake-e1.json)。舊DOSBox `frame_000`實為AFM 3龍紋過渡、不是發行商標誌。後段兩次`sub_286BD`亦已依IDA直接指令修正為半開區間0..254的索引色盤內插，正式播放紅幕→真實ANI #1→近白標題；[淡入影格](../figures/title-transition-remake-e1.png)與原版frame193的MAE為0.077／255，達99%玩家可見門檻。非原版 F2 常駐提示已移除，逐幕按鍵契約保留。只剩精確音訊與原版完整啟動E2，故不升完整E2。LOAD／CONTINUE 的來源限制另依 `58` 記錄 |
| tactical field/HUD | 50–60%（E1；ch01、ch29及ch30候選錨點） | 目前正式 `story_ch00_handler` 截圖 `native-map-ch01-remake-handler.png` 與 `native-map-ch01-original-video.png` 仍是不同狀態，只能證明原始資源／渲染輸入及隊伍 handoff 已被消費；重製圖已改以唯讀原版 `FDOTHER/FDSHAP/FDICON` 與 IDA 已證實的 FDFIELD b1 selector 產生，修正舊 b0 映射的敵軍圖像錯誤。舊 pair 的場上單位、游標與 HUD 差異不能作為目前渲染器缺陷證據；舊 `native-map-ch01-remake.png` 是直接節點除錯歷史證據，不再作正式比較。較早 E1 raw 相機／游標欄位也不等於畫面像素一致。2026-08-10 另以同一 `FD2.SAV`、相機、游標、回合與單位狀態建立 DOSBox／重製逐幀範圍比較，最近鄰縮放後內容區只剩 22 個畫布邊界差異像素，記為 ch01 scoped E2 candidate。2026-08-26 又以外部末關候選從正式標題普通鍵盤 `CONTINUE` 抵達 `battle_ch30`；旁車配對round12、camera `(16,16)`、cursor `(21,20)`，正式indexed六階段寫入18筆鏡頭內單位，且foreground／HUD未覆蓋unit stage。舊洋紅亂圖已勘誤為未提供`FD2_ORIGINAL_FDOTHER`而走PNG fallback，不是renderer缺陷。合法IDA其後閉合並接通`FDOTHER #55`輔助底面與`0x11CAC(0)→0x4DFCC`的DAC `0xE0..0xEF` cycle；同一typed戰況的合法aux phase10／palette phase0達320×200整幀`AE=0/64000`。另以fixed-hash `fd2004`候選由未修改原版普通CONTINUE抵達玩家第29戰並以Return開啟指令環，重製正式handoff亦還原76筆場上單位、31人持續隊伍及相同round／camera／cursor；因第三方來源仍只列候選。這些候選都不能外推成其他章節或完整操作界面E2，精確音訊與完整存檔provenance仍缺。2026-07-29 稽核發現只有 map0 曾帶 `native_tile_blit_modes/native_terrain_control`，現已從雜湊鎖定的 FDFIELD／FDSHAP 同步至全部 33 圖並有全圖 regression；這只閉合 renderer inputs，不是全遊戲視覺 E2。ch26 又由 pre-handler PAN/FOCUS 與 cursor state machine 閉合 event61 所需 runtime view/HUD E1；ch27 的 selector0→event62→event63 raw camp0 敵軍 AI 前 runner、兩批增援與全白／恢復演出已達重製端 E1，戰前 view／selector0 及 inherited HUD owner 也已閉合並接線。gate A 由存檔保存、anchor 為程序內持續、gate B 由 controller 物化；ch02+ 其餘畫面、一般玩家／CONTINUE 同 roster/event/tick DOSBox 像素差分、該時點角色 raw record 的實際值，以及 `0x12c0d` 的 exact raw lookup predicate/order 仍待補齊 |
| action/command/item/target UI | 45–55% | action skin、command grid、item panel已有原資源 indexed adapters；command grid 另有空 ID／selected 越界失敗即關閉回歸。command0完整`0x2A6BD→0x26152` presenter現由玩家與`Raw53AF9==0`的敵方ID0共用，但各自保留cleanup／continuation。玩家物品type20／21／24正常確認已共用`0x1CD17／0x1CAC7` indexed owner，最後畫面邊界後才發布傷害，尾停後才結束行動；缺素材時保留目標模式並零交易。33圖固定非玩家command mask排除mode8後的正常producer均已有indexed owner；剩餘差距是完整availability動態對照、selector 6/7+、狀態高階名稱與同狀態DOSBox畫面／音訊比較 |
| story dialogue | 60–70%（ch24_post opening＋stable-page＋progressive＋mouth＋closing E1） | `ch24_post` index6/7 的18句已由 raw `FFED/FFEF/FFEE` 固定上／下框，再逐字對照 `FFFE/FFFD` 投影成可編輯 pages；正式第25戰戰後 runner 現使用原版 FDOTHER #5 `(5,2/112)` 的五段開框、DATO portrait、FDOTHER #4字模及原生文字座標。`sub_16C57`等待輸入期間的post-decrement已接成完整頁限定的m0/m3 indexed嘴型，opening／逐字／捲動／closing均重設；`sub_16B43`另以真實snapshot含義收成`16×5→12×4→8×3→4×2→原背景`，非零`var_20`再由raw `+8/+5`查找、可見游標與FDOTHER #5 entry0重播移動尾段。舊句在完整收框前不會pop，正式接縫仍走到 `town_ch26`／祕密商店，達窄 `RUNTIME-E1`。更新分配是可重現近似，不是DOS精確時鐘；同狀態DOS畫面與其他 caller仍缺，其他caller仍走既有RGBA路徑。README 的舊runtime圖不是本切片成果圖 |
| full-screen battle presentation | 40–50% | FIGANI/AFM、部分 status frame與局部 pixel-equal slices可重播；2026-08-10曾宣稱戰鬥姓名已改接FDOTHER#4 16×16 glyph，但2026-08-25實檔稽核發現`unicode_to_glyph.json`的`_comment`讓載入器整批退回TTF；修正解析並鎖定panel`+(5,4)`後才真正達`RUNTIME-E1`。固定命中fixture已把HP顯示改為impact邊界立即提交post-hit值、守方剪影E1色值對齊原版RGB `(190,0,0)`，守方待機幀也改消費descriptor `+6`延遲而不再固定`(prog/6)`。同日以IDA直接consumer閉合`0x5255F／0x52577`六相位位移；目前只把已觀測第一個phase5接入E1剪影，並讓普通攻擊重用既有`0x18C6D` indexed bar／digit核心。完整序列最佳frame76由`AE=4436→1330→903→519`；519全部是舊三欄比較圖oracle的左邊／底邊合成邊框，排除後319×199內容區`AE=0／RMSE=0`，見[`battle-impact-compare-20260825.png`](../figures/battle-impact-compare-20260825.png)與[`battle-impact-no-global-tint.json`](../data/ui-traces/battle-impact-no-global-tint.json)。完整六相位因缺`0x29F72` raw owner仍只到`DATA-READY`，oracle亦非新的未修改DOSBox擷取，故不升E2。command29玩家多目標indexed owner已接。ID32／33／34／35正式command grid與state transaction亦已接，但`0x27FC9` indexed presentation仍缺，不能把綠色交易測試當成畫面完成。28／31無已證實正常取得來源；敵方command29、其他可達command/spell/item完整presentation、精確音效與palette/timing sequence、一般玩家同狀態DOSBox E2仍缺 |
| postbattle/campaign transition | 30–40% | graph與24個標準 postbattle handler均已接；第29戰勝利→戰後→19人整備／冷讀檔→ch29_pre→第30戰已形成連續 E1。2026-08-26 再補齊 ch30 最後 focus 的 view／HUD、party→groups constructor 與完整 END→YES／敵方回合／勝利→終局介面生命週期；原版每章可見轉場仍未逐章 E2 驗收 |
| town hub | ch02 examined slice 85–90%；全章 partial | ch02 variant0 selection0–5 均取得原版 DOSBox E2，且各自能和 production remake 某個 pulse 的 320×200 raw RGB MD5 整幀相同；另有 variant1與variant2 selection0–4 的修改 LOAD E2，兩組五項與指定 pulse 的 640×400 整幀 AE=0。Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape返回selection5皆由原版 input trace 到達。23個town仍缺variant2 selection5 的 BIOS 掃描碼／Enter、未修改一般玩家路徑與其餘章E2；85–90%不可外推成全遊戲town覆蓋率 |
| weapon/item/secret shop | ch02 examined slice 92–95%；全章 partial | 69個shop節點已用variant1/3/5啟用indexed owner，但DOSBox E2只覆蓋ch02若干狀態。購買主選單／清單／Yes-No／不足金與收件者selection0↔1、賣出名冊／物品／Yes-No／成功／加款／返回已有多組整幀AE=0。service2名冊／面板與service3五個穩定子面板另有route-patched partial E2；service3物品清單的AE=2已定位為店員背景動畫兩點相位差，靜態清單內容在該狀態一致。其餘內容與幾何一致但動畫相位未同步。service2裝備現先在私有 unit 完成 raw transaction、重算及完整 panel 重建，最後才一次發布；刻意移除 palette 的深層失敗回歸證實 roster、能力、selection與既有 panel 全部不變，達`RUNTIME-E1`。service3現把Ebiten鍵盤轉成單一typed input consumer；回歸由正式商店四項選單Right×3進入service3，逐一完成opening／closing／restore的Draw acknowledgment，再驗證empty回覆與返回、full原子拒絕，以及目的取消後重入self-transfer。這些分支不再只靠直接寫mode或呼叫transaction helper，均達production-input `RUNTIME-E1`。六名synthetic typed party另由正式menu→purchase→Yes consumer進三列裝備收件者，驗證selection3時start0→1、horizontal no-op、滿欄／無合適角色零交易及正常裝備成功／扣款；每個opening／closing／restore與timeline都經Draw acknowledgment。測試未注入OS鍵盤，也未從完整campaign抵達商店，不能替代原版四人以上存檔或玩家E2。這些E2仍依screenshot-only LOADCH bootstrap；正常JOIN→LOADCH roster另有runtime regression。仍缺四人以上recipient scroll、no-recipient/full、service2動畫相位與原版mutation／restore畫面、service3 empty／full，以及self／destination-cancel與其他章節的原版同狀態E2；92–95%不可外推成69店全覆蓋率 |
| church | 60–70%（已接 slices） | church main/status/transfer/revive/class 多數已有原始 FDOTHER/FDICON/FDTXT indexed畫面與 lifecycle；正式 `church_ch02` 在 `selection=0,pulse=2,gold=1000` 的640×400擷取最近鄰縮回320×200後，與現行原始資源oracle達`AE=0/64000`。舊oracle的`AE=320`已勘誤為過時產物，不是runtime缺陷；此證據仍只是`RUNTIME-E1`，因oracle不是DOSBox原版畫面。transfer的`0x2f8ea`亦由shop service3共用，非church專屬；仍缺 DOSBox side-by-side與完整 persistent/save parity |
| preparation | 50–60% | 舊「兩欄文字核取方塊」與「確認框仍是重製殼層」斷言已失效。城鎮 FDTXT `0x201` 出發提示會保存／還原實際 town frame；無城鎮 FDTXT `0x19a` 記錄提示使用原版黑色來源，存檔延至完整關框後。兩者與 `0x31d3c` 最終確認都接上 6＋4＋兩 tick 脈動＋4＋5＋還原。`0x318ad/0x31e80` 選人主畫面、`0x17fc0` 狀態與 `0x1297d` 待機週期亦已接正式路徑。2026-08-27再把分散鍵盤分支收斂為共用typed consumer；第23戰戰果後已從記錄提示肯定／存檔走到全新`Game`冷讀、提示否定、15人選取、最終取消重選及肯定進`story_ch24`，見[`ch23-post-preparation-ch24-input-e1.json`](../data/ui-traces/ch23-post-preparation-ch24-input-e1.json)。2026-08-26 外部 checksum-valid 晚期槽已由未修改原版普通 LOAD 走過 save-NO、19人選擇、最終確認與戰前劇情至可操作第30戰，形成候選E2；重製再由同一固定槽的正式LOAD入口擷取初始整備，修正第一組數字誤畫已選人數的缺陷。凍結合法相位0並分開FDOTHER #5 entries 31..40／42..51兩套數字字形後，320×200同狀態整幀AE=0/64000。第29戰勝利寫槽的完整來源鏈只保留為證據限制；50–60%不可外推到所有整備章節 |
| save/load | 60–70% | 四槽 input、native save envelope、原版 indexed loadslots 與 chapter-slot→typed party→town/preparation restore owner 已接；current battle 的巢狀 LOAD 也會先完成私有 typed handoff，YES 後才原子替換。SAVE 會保留未知 bytes，並對具證據的新我方 JOIN 同步 persistent raw。重製 JSON LOAD 亦先驗證 JOIN／membership／部署／materialized roster 拓撲；新的 `Game` 已能冷讀 `preparation_ch30` 後繼續第30戰與終局，錯誤拓撲則原子拒絕。2003年外部末關候選存檔通過原版 checksum，固定版原版由普通 `CONTINUE` 抵達第30戰；重製同檔也發布相同回合／鏡頭／游標、33筆場上單位與31人持續隊伍。另一外部 checksum-valid raw chapter `0x1c` 槽已由未修改原版普通 LOAD 連續走過 save-NO、19人整備、最終戰前劇情至可操作第30戰，見 [`native-load-ch29-slot-to-ch30-original-candidate.json`](../data/ui-traces/native-load-ch29-slot-to-ch30-original-candidate.json)。兩者均因第三方來源不完整只列候選E2；仍缺第29戰勝利當下由原版 writer 建槽的完整來源、delete／overwrite與同狀態重製差分。長程往返改由使用者人工回報問題，不列代理工作項目 |

> **晚期 slot 0 重製端補證：** 同一固定雜湊檔現經正式 LOAD selector 還原29人、
> 60金幣與四個已證實 metadata byte，並進 `preparation_ch30`。這關閉原版候選與
> 重製節點間的路由差分。後續正常輸入回歸又從標題 LOAD 事件、slot 0、連續19次
> 選取與最終確認抵達 `battle_ch30`；選取後游標自動前進，20名部署者直接來自已驗證
> persistent records，包含 authored ch30 scenario 漏列的 identity 3。逐像素整備比較、
> writer 來源及其他章節差分仍未閉合。同一固定槽又由正式 END／YES 進入未刪減的
> 第30戰敵軍回合；原始敵軍完成行動並交回 `PLAYER PHASE`。這是重製端
> `RUNTIME-E1`，不是新增原版逐幀或音訊 E2。
| ending | 60–70%（E1；來源約束終局） | prefix 已跑到 `0x2c548`；`sub_2C39B`的19×5框、caller initial portrait與FDTXT逐句speaker來源已閉合，timeline保存每句control／operand／pages。正式ending已用原版indexed owner逐Draw呈現六段opening、最多四列逐glyph、完整頁嘴型、Enter／Space分頁／換句、五段closing與source restore；不再使用一般RGBA框，完成收框後才resume timeline。chapter26兩個文字閘門與chapter29第一閘門五個blocks均有原始資產預建／生命週期回歸；缺資產整批零發布。正式 `battle_ch30→ending` 現以 persistent JOIN roster 的 raw `+6/+7/+8/+0x20` 同步最後隊伍並執行原資源角色蒙太奇；全新 `Game` 的冷讀連續回歸已由 `preparation_ch30` 經一般戰果接縫抵達此處，部署成員採最終戰更新，未部署成員保留冷讀狀態，回顧循環依完整 JOIN 時序涵蓋全隊。缺戰場、零筆身分符合、缺任一原始資產或 raw provenance 時不發布部分終局，而是整批回到可編輯結語。其後 `MontageTailPlayer` 預檢 20 組共80個 TAI／BG／FIGANI selector；實檔全部 header byte1=0，因此正式 E1 現逐 raw `+6` inner present、`+7 bit0` 層序、base scheduler、`+4` 位移與 palette33 override執行兩次交叉配對，第二次在最後 effect frame 結束；再依 caller 的20／78 ticks疊 FDOTHER #58，完成後保持 #59。[20 組總覽](../figures/ending-tail-20-segments-approximate-remake-e1.png)仍只是 E1，因為 `0x1088d(0x1e)` 的31-record baseline 穿過 `0x2c548` 的位元連續性、聲音 owner、原版輸入時序與一般玩家路徑尚未閉合。正式成功路徑現不再疊加來源等級／按鍵說明等現代提示；僅玩家主動開啟除錯HUD時可見。`Game.Update`與冷讀長鏈回歸現共用單一終局輸入owner，完成raw-change pending、定格進回顧與回顧返回；這只證明重製端正常輸入接線，不外推原版scan code。`0x2939D` 的3%外層預算在終局非零分支讀取未初始化區域值，不能當作穩定重播契約，已降為非阻擋原版考古限制。外部片尾錄影把 #59 對應為 `THE END` 是**強推論**，見 [`ch30-ending-youtube-visual-side-evidence.json`](../data/ui-traces/ch30-ending-youtube-visual-side-evidence.json)，不是一般玩家 E2。Enter／空白鍵開啟、Enter／空白鍵／Esc 關閉的角色回顧循環是重製版延伸；一般玩家 E2 仍未完成 |

> **2026-08-25 第30戰同狀態反證：** 來源可追溯的末關候選存檔現可由原版普通
> `CONTINUE` 與重製普通 X11 鍵盤分別抵達相同 round／camera／cursor。重製側仍用
> `FD2_NOCUT=1` 與固定 title tick，故只列 E1；實際畫面已暴露 map29 外圍地形錯誤
> 圖樣及可見單位配置差異。此反證不降低 ch01 scoped 候選結果，但禁止把 ch01
> 外推為晚期戰場完成。證據見
> [`native-battle-ch30-original-candidate.json`](../data/ui-traces/native-battle-ch30-original-candidate.json)。

> **2026-08-26 第30戰同相位閉合：** 合法IDA／Capstone
> 證實一般`0x11CAC(0)`在terrain前還會呼叫`0x4DFCC`，依BIOS兩tick gate把
> 93-byte滑動表寫到DAC `0xE0..0xEF`。正式runtime已接此owner；同一typed戰況的
> 合法aux phase10／palette phase0與原版候選整幀`AE=0/64000`。這關閉該候選相位的
> renderer差異，但第三方存檔、固定title tick與非全程來源仍只容許候選E2，不能
> 外推成所有第30戰時點、其他章節或完整一般玩家通關E2。

> **2026-08-26 正常標題路徑補驗：** 另一次正式GUI重播完整播放重製開場，未設定
> `FD2_NOCUT`或`FD2_NATIVE_TITLE_TICK`，只在選單以普通X11事件輸入
> `Down、Down、Return`；frame7202旁車證實抵達相同`battle_ch30`／round12／camera／
> cursor，且沒有dialog、battle event或turn staging。這關閉重製端兩項測試夾具，
> 不改變第三方存檔provenance與完整通關E2限制。

> **同日可操作邊界：** 相同時間線在CONTINUE發布後再送一次普通`Return`；frame6300
> 旁車證實`opening_confirm=false`、`action_overlay_open=true`、
> `native_continue_cursor_overlay=true`，原版indexed空游標面板實際可見。這證明正常
> GUI不只載入第30戰，也把操作權交給玩家；仍不外推為完整來源或從第一戰通關E2。

> **同日敵方回合補證與勘誤：** 先前原版證據只分開證明可操作第30戰與有界
> `END→YES`診斷，不能合併成單次連續鏈。2026-08-26 重新由雜湊一致的第三方候選
> 存檔建立乾淨tmpfs副本；固定版原版未修改執行檔、記憶體、章節、路由或畫面，
> 同一DOSBox程序以普通鍵盤完成`CONTINUE→END→YES→ENEMY PHASE`，執行敵方演出後
> 再接受普通Return並開啟索爾狀態面板，直接證實控制權交回玩家。三格圖與時間線見
> [`native-battle-ch30-original-candidate.json`](../data/ui-traces/native-battle-ch30-original-candidate.json)。
> 這提升為未修改原版執行與輸入的單次連續候選E2；第三方存檔來源與停用音訊仍使
> 它不能證明從頭通關、完整來源（provenance）或精確音訊，重製敵方回合本身仍為E1。

> **同日第29戰候選錨點：** Player Lin 保存站 `fd2004.zip` 的固定雜湊
> `FD2.SAV` current snapshot 為原始章節值 `0x1c`。固定版未修改原版以普通鍵盤
> `Escape×8→Down→Down→Return` 進入第29戰；戰場穩定後再按 Return 會開啟原版
> 指令環，直接證實操作權。兩次 Down 刻意分離，較早誤入 START／LOAD 的畫面已拒收。
> 完整雜湊、畫面與限制見
> [`native-battle-ch29-original-candidate.json`](../data/ui-traces/native-battle-ch29-original-candidate.json)。
> 這補上第29戰未修改執行檔／普通輸入的候選錨點，但第三方存檔來源不完整，且尚未
> 從該戰勝利連續走到第30戰，因此不升格為完整第29→30戰一般玩家 E2。

> **同日第29戰單次回合循環補證：** 以上述相同固定雜湊存檔重新建立乾淨沙箱；同一
> DOSBox程序由普通`CONTINUE`進場後，連續完成`END→YES→ENEMY PHASE`，等待約150秒
> 完成敵方演出，再以普通Return開啟索爾狀態面板。這把第29戰候選由「可操作錨點」
> 提升為一輪玩家／敵方控制權交接的單次連續候選E2。三格畫面與完整時間線仍沿用
> [`native-battle-ch29-original-candidate.json`](../data/ui-traces/native-battle-ch29-original-candidate.json)；
> 第三方來源、停用音訊及尚未完成第29戰勝利→第30戰仍是限制。

> **同日重製第29戰回合交接：** 同一 `fd2004` 固定雜湊槽由重製正式標題事件
> `CONTINUE` 進入 `battle_ch29`，再由正式 END／YES 介面驅動完整敵方回合；沒有
> 清空敵軍、預設 `Acted` 或直接呼叫回合捷徑。至少一名敵軍實際行動，回合數增加並
> 回到 `PLAYER PHASE`，76筆 runtime／31人 persistent 拓撲保持不變。這關閉的是
> 重製 `RUNTIME-E1`；外部來源與停用音訊不允許提升為完整 `PLAYER-E2`。

> **同日晚期有效槽 LOAD→第30戰補證：** Player Lin `fd2021.zip` 的 checksum-valid
> `fd2last.sav` 保存 raw chapter `0x1c` slot 0。固定版未修改原版在同一 DOSBox 程序
> 由普通標題 LOAD 讀槽，選 NO 跳過再次記錄，連按19次 Return 完成晚期整備，接受
> `0x31D3C` 出戰確認，再逐頁走完最終戰前劇情至可操作第30戰。四格圖、輸入、雜湊
> 與拒收路徑見
> [`native-load-ch29-slot-to-ch30-original-candidate.json`](../data/ui-traces/native-load-ch29-slot-to-ch30-original-candidate.json)。
> 這關閉有效晚期槽、preparation-only save prompt、19人整備及第30戰控制權的候選E2；
> 保存站註記全隊等級全滿，且沒有第29戰勝利→writer建槽的完整來源鏈，故不提升為
> 未修改全程通關或完整第29→30戰E2。

> **同日晚期整備畫面勘誤：** 舊的9255／3763／9255結果是截圖入口
> 未凍結圖像相位，不是三個固定相位的有效比較。IDA `0x1088D→0x11019`
> 與 `0x31FBB` 證實名冊依 persistent raw key 首見順序消費並略過固定隊長；
> 28個可選格在相位0逐格 `AE=0`。`0x31EA9..0x31EFB` 另證實兩組
> 數字分別使用 FDOTHER #5 entries 31..40與42..51。凍結相位並接通
> 第二套字形後，該固定初始狀態為 `AE=0/64000`。依使用者99%
> 忠實度停止線，本狀態不再追 DOS 逐週期差異；其他章節與交互狀態仍須抽測。

> **同日正式戰役長鏈補證：** 冷讀 `preparation_ch30` 後的正式
> `story_ch30→battle_ch30` 先前只驗證單位數與結果接縫，實際缺少 HUD／range entry，
> 且非 runtime-append setup 使 FDFIELD 單位留在 selector cache 外。現行資料以
> ch29_pre 最後 `sub_12D7B(slot0)` 的 typed view `(camera 16,14；cursor 23,18；visible
> 7,4)`、`0x33F69` gate B=1 與 selector0 建立 entry；handler 已物化 rows 依 immutable
> raw origin 扣除，再補 groups1–3。跨全新 `Game` 回歸現連續走完原生 map、END→YES、
> 敵方回合、勝利、ending 文字閘門、角色蒙太奇、20 組尾段、永久定格、隊伍回顧
> 及返回同一定格；JOIN 順序與 persistent raw `+6/+7/+8/+0x20` 全程保持一致。這提升重製端 `RUNTIME-E1`，不把
> 敵軍全滅 fixture 或未送 OS 鍵盤的測試冒稱 `PLAYER-E2`。

終局 19×5 owner 的最新玩家可見 E1 證據見
[`ending-dialogue-native-indexed-remake-e1.json`](../data/ui-traces/ending-dialogue-native-indexed-remake-e1.json)；
旁車鎖定 `0x2BE44`、`waiting`、第 1／5 區塊且一般對話佇列為零。這只關閉重製端
正式 renderer，不提升第 30 戰一般玩家 E2。

> **2026-08-25 最終戰前劇情勘誤（E1）：** `story_ch30` 已停止使用兩句通用後備，
> 改由正式 `ch29_pre` binding 消費 `LOADCH`、21句 FDTXT_030 對話及七個
> `0x33F78` staging caller。wrapper 先聚焦 `(x,y)`，再由既有 `0x22253` owner
> 於 bridge 邊界發布 story slot 座標；任何必要資產或原生視圖缺失時，focus 前即
> 失敗即關閉。Docker／Xvfb 正式節點畫面與限制見
> [`story-ch30-pre-remake-e1.json`](../data/ui-traces/story-ch30-pre-remake-e1.json)。
> 單張圖不證明七次 staging 的每一幀，也不提升第29戰戰後→整備→最終戰前的一般玩家 E2。

### 2026-08-25：代表性戰間介面抽測契約

早期、中期與終局前的抽測固定使用 `campaign_full.json`，並由正式 `Game` 邊界
消費選單、服務返回、整備取消／確認、持續隊伍及存讀檔；自建 campaign fixture
或直接指定節點只可協助定位，不能列入這批完成證據。最小代表路徑為：

- `town_ch02→church_ch02→town_ch02`：正式城鎮選項及教會返回；
- `town_ch17→preparation_ch17→town_ch17/story_ch17`：正式城鎮輸入、城鎮來源
  畫面與整備取消／確認；
- `battle_ch29→postbattle_ch29_persist→preparation_ch30→story_ch30→battle_ch30`：
  正式戰果確認、raw ch28 post、持續隊伍、存讀檔、最終戰前 binding 及戰場物化。

2026-08-25 回歸已通過上述三段，且晚期路徑確實執行19人選擇、存讀檔、21句對話、
七次 staging 及第30戰物化。測試也抓出並修正 LOADCH 未同步一般游標、story view
六欄混用及新 scene actor 缺原生呈現載體三個只在連續路徑出現的缺陷。這仍只提升
重製端正式資料與執行期的代表性 `RUNTIME-E1`；未注入作業系統鍵盤的測試、快速
略過可見等待，以及直接節點擷圖都不提升未修改原版一般玩家 `PLAYER-E2`。三張
直接節點擷圖目前只作本地診斷，不加入版控或進度百分比。

### 2026-08-11：未修改原版敵方回合 E2 錨點

以固定雜湊的 `FD2.EXE`／`FD2.SAV`，在一次性 Docker DOSBox 中由
`CONTINUE` 進入 current-runtime 戰場，開啟 command grid，選擇 `END` 並以
`YES` 確認。約 1 秒畫面明確顯示 `ENEMY PHASE`，約 10 秒仍在敵方回合，約
20 秒回到玩家操作狀態。三張 320×200 client crop 與完整輸入、PNG 雜湊見
[`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)。

這關閉的是「原版完全沒有一般玩家敵方回合 E2 輸入／畫面錨點」的舊缺口；它不證明
目標選擇、移動評分、命令／法術／道具決策，也不是重製端同一 raw 狀態的 parity。
`REMAKE-AI-MODE-RUNTIME`、`CAMPAIGN-POSTBATTLE-E2-FULL-PATH` 與重製端同狀態
逐幀比較仍維持 partial／open。

### 2026-08-11：CONTINUE battle handoff E1 邊界

重製端已新增 `MaterializeNativeContinueInteractiveBoundary`、
`ValidateNativeContinueBattleHandoff` 與 `Game.publishNativeContinueBattle`：所有
已驗證的欄位／執行期／待處理群組／計時／視圖／HUD 型別化轉接器
（field/runtime/pending/timing/view/HUD adapter）完成後，才從開場選擇器
（selector）mode `0` 原子切換到互動 mode `1`，並以一次發布清除舊對話／轉場／戰鬥暫存。真實
`FD2.SAV` chapter0 快照的 Docker 回歸（regression）已通過；測試明確以呼叫端提供的零值
計時種子（timer seed）驗證資料契約，不是標題時鐘或畫面 E2 證據。

同日已補上正式標題呼叫端：`TitleMenuContinue` 只接受明確提供的
`FD2_NATIVE_SAVE` 與 `FD2_NATIVE_TITLE_TICK`，從可編輯戰役圖唯一解析
`scenario.chapter` 相符的 battle node，在私有 state 完成四個 adapter 後才發布；缺少
存檔、signed BIOS tick、資產或章節對映含糊時，標題保持不動並失敗即關閉。真實
`FD2.SAV` chapter0 的標題 Escape／Down／Down／Enter 路徑已在 Docker／Xvfb 取得
[`native-continue-current-runtime-remake-e1.png`](../figures/native-continue-current-runtime-remake-e1.png)，
條件與雜湊見
[`native-continue-current-runtime-remake-e1.json`](../data/ui-traces/native-continue-current-runtime-remake-e1.json)。
這是重製端 E1 publication／輸入邊界，不是 BIOS 時鐘逐幀或原版畫面 E2。

**2026-08-27 勘誤：**上段之後的實作已解除其中多項閘門，不能再用這份舊清單重開
工作。33圖待處理群組現有正式 materializer；正常 producer 的敵方 AI caller、目的格、
目標陣列及命令／法術／物品交易已達 `RUNTIME-E1`；戰後24節點、town／shop／
preparation、巢狀SAVE／LOAD、service2／3與最終戰長鏈也都有正式 owner。剩餘是
其他章節與未修改原版的代表性 E2、精確音訊／動態相位及三平台實機驗收，不是缺少
重製端 consumer。依使用者接受的99%玩家可見門檻與長程遊玩人工回報政策，這些
原版 E2 抽樣不得再阻擋引擎收尾或觸發重做已閉合 RE；當前產品阻擋以`91`的發行列為準。

2026-08-10 的音訊邊界：戰鬥節點使用原版 `0x51e63` 章節曲表，城鎮／商店節點使用
已證實的 `FDMUS_010`；這些是資料回歸，不代表每章一般玩家 E2。`ending` 的三個
已證實事件與位址、檔案雜湊見 [`fd2_ending_audio_ida.txt`](../data/ida/fd2_ending_audio_ida.txt)。
可編輯結語的空白 BGM 只呼叫已證實的 `play_bgm(-1)` 停曲；`0x2BCE5` 的 indexed
前綴現由正式 `battle_ch30→ending` 與第27戰缺天空之鑰的
`0x250CC→0x2545D` 分支，以各自 chapter29／26 的嚴格來源約束 E1
`native_ending_prefix` 啟動，不再依賴 `FD2_APPROXIMATE=1`；後者會消費
`FDTXT_027` index17..20 的兩個原版文字閘門，不再只顯示通用結語。最終戰路徑
在 `0x2c548` 消費 `FDMUS_004`，並以 persistent raw roster
播放 `MontageCycle`；cycle 成功完成後，`MontageTailPlayer` 會消費 20 組原版
TAI／BG／FIGANI、descriptor `+6` 延遲與 FDOTHER#58 疊圖，再保持 `FDOTHER#59`；只有
素材／raw provenance admission 失敗才回到可編輯結語。`FDMUS_018` 在近似尾段開始時
接線，但精確停曲、間隔與畫面同步不宣稱與原版相同。新輸入只在 portrait loop 實現已證實的「完成本輪後
進 final loop」效果；另有可選的現代隊伍回顧，離開時回到 #59。精確 BIOS 按鍵對映、
精確 `0x28a6c` 20-entry renderer、停曲與一般玩家 E2 仍維持失敗即關閉；已記錄的
record0／record1 `+6/+7` raw writes 不構成 renderer 完成宣稱。

2026-08-10 ch01 HUD 位址勘誤：官方 IDA 直接指令證實 terrain icon 與 optional unit
icon 都寫入 `base + stride*5 + 6`；重製端已修正原先把 terrain icon 寫在 `base+6`
造成的向上偏移。以同一 `FD2.SAV` 的 Docker DOSBox oracle 與 handler 截圖做
320×200 最近鄰比較，內容區只剩 22 個邊界差異像素；詳見
[`battle-field-ch01-scoped-compare-20260810.png`](../figures/battle-field-ch01-scoped-compare-20260810.png)、
[`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json) 與
[`fd2_map_hud_geometry_ida.txt`](../data/ida/fd2_map_hud_geometry_ida.txt)。這是 ch01
單一狀態的範圍 E2 候選，不可外推至其他章節或完整操作界面。

舊版曾把少數已檢視切片換算成整體百分比；這種算法沒有完整的畫面狀態、章節與
玩家路徑分母，現已撤回。town／shop 的 ch02 同狀態證據不能外推為 23 個 town
或 69 個 shop 狀態的覆蓋率，資產可解碼也不能換算成操作界面完成度。現況只能以
下表逐列的 `partial`／E0／E1／E2 狀態陳述，不再對外提供單一整體百分比。

外部交叉證據只用來辨認原版畫面結構，不取代本機 DOSBox oracle：

- [巴哈文章搜尋摘要](https://home.gamer.com.tw/artwork.php?sn=1432264)
  明確包含「進入教會的畫面」；頁面目前會拒絕自動抓取，故不作像素證據。
- [小黑盒原版回顧](https://api.xiaoheihe.cn/maxnews/app/share/detail/2265131)
  的原版 shop screenshot 可見店員、店內背景、藍色對話框、gold counter
  與圖示選單，直接排除「地圖上通用半透明商品清單」是原版等價 UI。
- [百度原版攻略畫面](https://jingyan.baidu.com/article/597a0643385421312b5243cf.html)
  可見戰場 action menu 是原生像素 overlay，不是一般現代文字 panel。
- [圖文攻略](https://egameinsider.com/p/dko871470c83/)
  顯示章間服務並非每戰後同一流程；例如第22–25章是連續戰、沒有村落
  補給，支持 campaign 必須逐章保存 town／shop／連戰節點。

### 視覺優先順序

1. 先把 town、weapon/item shop、preparation、loadslots 從 generic
   `drawCampaignUI` 分離，建立 320×200 original-indexed scene owner。
2. 每個 owner 必須有同一 state/input 的 DOSBox screenshot；不能只用
   原資源 fixture 宣稱 E2 parity。
3. 戰場驗收要固定同一 save／roster／camera／cursor／animation tick，再
   做 palette-index或RGB pixel diff；目前兩張 ch01 圖只能證明 compositor
   slice，不能證明整幀等價。
4. README 只展示並列且標清 `original DOSBox`、`remake runtime`、
   `indexed fixture`、`raw decode` 的圖片，禁止再把 raw decode 說成 remake。

## 現有 runtime evidence

| Contract | 現有 code evidence | 判定 | 下一個證據問題 |
|---|---|---|---|
| UI-01 title/menu | 原版 `0x1fe2c` scan-code loop（↑/↓ wrap；Enter/Space/`0xe0`/`0x52` confirm）、`0x25ebb` return dispatcher、DOSBox oracle `docs/figures/title-original-dosbox.png` 已固定 START／LOAD／CONTINUE 與 title cursor；四槽 selector、valid-save typed restore 與 indexed 畫面已接。CONTINUE 的 FDFIELD 控制映像、battle-local event state、current-runtime-order selector rebuild，以及標題 caller 的 opening／interactive range mode、HUD gate B／anchor 已閉合成唯讀 preflight；後續 map timing、live field、runtime units、future-group transaction 與未改寫 chapter0 pending roster 亦有嚴格 consumer | partial | 第三主選單 CONTINUE 的 production owner 仍缺動態 turn-writer／group-formula 的通用 pending-group binding，及 `0x117E7` 對應的正式 `Game` controller handoff；另缺刪除／覆寫與完整 boot 畫面差分 |
> **UI-01 表格勘誤（2026-08-11）**：上列「CONTINUE production owner 仍缺」是接線前的歷史描述，現由 `TitleMenuContinue`＋`loadNativeContinueFromCurrentSnapshot` 補上 chapter0 E1；它仍不是一般玩家 E2。詳細輸入與失敗即關閉條件以本頁下方同日段落及 [`native-continue-current-runtime-remake-e1.json`](../data/ui-traces/native-continue-current-runtime-remake-e1.json) 為準。

| UI-02 field | map/camera/cursor/unit/HUD Draw 約 3441–3568、4571、4595；camera、absolute/visible cursor、HUD anchor/gates 與 FDOTHER #130 panel 已有直接原版資料流；ch26 event61 所需 view/HUD 已達 E1。ch27 event62 已接向左一步第七拍 selector0，能由完整 raw row 與 `0x2066E` 已證實的新戰鬥回合初始值1啟用 event63；`sub_1A813(0)` 的敵軍 AI 前 owner、兩次 0x35822 增援、delta255 全白／delta0 恢復及 AI continuation 已接正式 runner。ch26_pre 返回 battle_ch27 的 view `(camera 9,49; cursor 14,54; visible 5,5)`、selector0 與 inherited HUD 已由 IDA／Capstone 閉合並接線；gate A 從存檔、anchor 從程序持續狀態、gate B 從 controller 取得，不猜章節常數。event63 的 indexed regression 由雜湊綁定 `NativeJoinConstructorTable` 建立凱麗 fresh raw `+0x42=151`，不再手填 fixture，也不由章節近似 HP 反推。ch00 的 `0x32999` 已以 FDOTHER #9 接12次索引呈現、pass6/7/8 snapshot 重建及 pass1 #95，兩次各12幀後能沿正式 handler 進入戰鬥、戰後、城鎮與整備。另以截圖專用快速時鐘逐拍跑完 `story_ch00_handler` 的 73 拍並擷取 `native-map-ch01-remake-handler.png`；完整隊伍已出現；`native-map-ch01-original-video.png` 是不同狀態的歷史參考，不能以其單位、游標、HUD 或尺度差異判定目前 renderer 缺陷。2026-08-10 另以同一 `FD2.SAV` 狀態做 DOSBox／重製最近鄰比較，內容區只剩 22 個畫布邊界差異像素，記為 ch01 scoped E2 candidate；ch01 以外仍只是 E1。ch01 global event1/2 又驗證 turn4/5 各12次呈現後才執行 ACTING(3/4)，event2 對話不會越過 acting；缺 acting 資源時不發布 roster/cache/turn continuation。這些目前仍不是完整戰場 UI 或全章節 DOSBox E2 | partial | 除 ch26／ch27 event62/63／ch00／ch01 event1/2 E1 切片與 ch01 scoped E2 candidate 外，ch02+ 的逐章 dynamic view/gates/anchor producer、ch27 一般玩家／CONTINUE 同 roster/event/tick DOSBox 像素差分、該時點角色 raw record 的實際值，以及 `0x12c0d` 的 exact raw lookup predicate/order；另補 ch00 與 ch01 event1/2 同 camera/roster/pass 的原版逐幀比較 |
| UI-03 action menu | Docker Capstone `0x18890` + `0x18d8c`：↑0 attack、←1 spell、→2 item、↓3 wait/field interaction；native command grid每欄四列。共用空游標 `0x16F55` 中，direction2設定與direction3 END已有正式E1。direction0另由`0x19DF7`開nested cells `36、39/41、42/44、45`；四分派及`0x1B1E7`資訊畫面的12開／等待任意鍵／12關、entries `0x85..0x88`、兩行FDTXT、六個數值欄與`0x4DFCC` palette cycle現已由正式鍵盤路徑消費。nested selector1 SAVE 已以完整 CONTINUE raw baseline、FDTXT `0x19A/0x19B/0x19C`、checksum、XOR envelope與原子取代接入；只有 YES 寫入明確的可寫複本，NO 與所有來源矛盾保持原檔不變。nested selector3亦依完整`-1`回傳鏈正常離開。外層selector1現以第一筆runtime `+7` DATO肖像、FDTXT `0x1A1/0x1A2/0x19C`確認，依record順序播放全軍移動並收束回合；正式資料僅有的selector1 event61／75會在行走途中暫停，分別完成59幀／JOIN31與對話後turn chain再續行。動態count、raw或事件資產不完整時均在發布前失敗即關閉。battle command 0、13–27已有多個來源約束E1。`0x1A866` selector 1→0／2 的 raw 狀態倒數、歸零重算與 runtime record 同步亦已接到正式非玩家／玩家階段 owner；`sub_17FC0` status colors／FDOTHER entries `0x37..0x39` 已由正式角色面板消費。 | partial（設定、END、nested資訊／SAVE／離場、含event61／75的全軍移動與狀態倒數為重製E1；END另有原版窄E2） | 上述系統路徑與commands同狀態逐幀／逐音訊E2；未來若資料出現其他selector1 handler仍須獨立接線；status高階名稱、到期提示精確 tick／音訊／E2及其他effect presentation |
| UI-04 target/range | `0x1cff0` + `0x149f8` 證實 command record `+3/+4/+6` 參與 target-candidate geometry；`0x1bbdc` item case 0 的 two-stage targets、observed type5–24 effect dispatch 已閉合。item entry materialize `row[+0x12]+2`；first selector return後grid reset且selector回1。type23 destination把literal target code6傳給`0x115b6`，不是global selector6；兩層取消都回item panel。remake已接tracked transaction、occupancy/class/race/29×20 cost/terrain gate。2026-08-22 同一未修改 `FD2.SAV` 已由標題 CONTINUE 正常操作至悠妮 command 0：原版四相位擷取為窄 `PLAYER-E2`，重製普通 X11 路徑的sidecar明列 `native_command_targeting=true`／ID0／游標(10,15)，為 `RUNTIME-E1`；[四相位圖](../figures/native-command0-target-original-phase-sequence.png)、[原版／重製／差異圖](../figures/native-command0-target-original-vs-remake-e1.png)與[擷取紀錄](../data/ui-traces/native-command0-target-original-vs-remake-e1.json)已固定動態LUT與modal presentation。兩側時鐘相位未同步，不列逐像素一致。正式 Game confirm 現完整預建並消費 command 0 的九段角色滑入、施術者效果、28 幀／7 元素錯開目標效果、七段 HP 與 LUT 尾段，所有階段需 Draw acknowledgement 才發布 MP／HP／acted，列 indexed `RUNTIME-E1`；缺 raw provenance、FIGANI、BG／TAI、FDOTHER、palette 或 panel input 時維持原子不變。command 1亦已接正式玩家／敵方owner：每目標31張、八段HP、八個sample1 marker、九幀目標間過場與common tail均依Draw發布，列indexed `RUNTIME-E1`。command 6 亦以相同原子邊界接入玩家與敵方：common 前導／施術者、全目標 orbit、九幀目標間過場、每目標五段 HP 與尾段均需 Draw acknowledgement，列 indexed `RUNTIME-E1`；#87 單幀多呼叫混音仍只近似。三者仍缺同狀態逐幀／逐音訊 E2。物品第一階段 raw target field 與 selector 1..5 已由正式索引組合器消費；缺原生素材時不再顯示無證據的綠／青／橘半透明後備層。另以同一未修改存檔正常選到索爾草藥：四名我方皆滿 HP 時，Return 後 4.3 秒仍保留物品面板、未進 target modal，形成 type5 無可回復候選的窄 `PLAYER-E2`；[三狀態圖](../figures/native-item-herb-fullhp-original-e2.png)與[擷取紀錄](../data/ui-traces/native-item-herb-fullhp-original-e2.json)已固定 | partial | 受傷角色的 type5 目標 modal、native argument↔weapon min/max mapping、AOE/LOS、不可用目標灰化、command0／1／6 同狀態逐幀／逐音訊 E2、其餘 indexed command／item effect presentation、物品取消鍵原版語意、其他command同狀態比較；global selector6的production owner仍待 |
| UI-05 dialog | dialog Draw 約 3590–3686；`dlgAdvance` 有 page/scroll state；ch01 original oracle `docs/figures/ch01-dialogue-original-dosbox.png` 固定左肖像下框、文字、page indicator。重製端另以可編輯序章腳本擷取 [`dialogue-remake-runtime.png`](../figures/dialogue-remake-runtime.png)（640×400，E1）；縮放至320×200與原版比較 AE=60414，僅證明重製端消費端（runtime consumer），未達同狀態 E2 | partial | 每種上／下框與肖像錨點、控制碼渲染器（control-code renderer）、原生裁切與原版同狀態差分 |
| UI-06 HUD | native map HUD `0x1acf3` 的 panel→terrain→AP→DP→optional unit icon→HP 已由 `BlitNativeMapHUD→ComposeNativeFrame` 接入 ch01 production full frame；display gates、persistent anchor、LMI1 #130／hex #0x83/#0x84、digit banks與FDICON selector均有 regression。`0x11cfa`證實HUD base是`work+0x8088`。FD2.SAV 初始快照為 camera `(1,13)`／absolute cursor `(8,17)`／visible `(7,4)`；原版錄影434.5秒的較晚比較幀則與remake對齊 `(1,13)`／`(8,15)`／`(7,2)`、tree icon及`A -05/D +10`。全 33 圖現具雜湊驗證後同步的 composition byte+3 與 terrain control；ch01／ch26／ch27 已改用 `native_map_hud_inherited`：gate A 由 custom save／native chapter restore 保存，anchor 在程序內持續，gate B 只接受已證實的 controller entry 1。event63 production regression 已由正式 JOIN table 提供 persistent raw record 並進 indexed path；#22仍只在 native admission 失敗時 fallback | partial | 除 ch26／ch27 E1 切片外，ch02+逐章動態 view/gates/anchor provenance、ch27 未修改一般玩家／CONTINUE 同一 roster/event state 的 pixel diff、`0x12c0d` exact raw lookup predicate/order；raw globals高階名稱仍不猜 |
| UI-07 postbattle | `campInput` battle result 約 2394；campaign node 可表達 post node；`campaign_full` 30 戰 transition matrix 已逐列展開。主迴圈直接指令、scenario `chNN→map(N-1)` 與 handler `chNN_post→set_chapter(N+1)` 共同證實玩家戰鬥 N 使用 raw `ch(N-1)_post`。13個既有同號錯接已全數清除；目前稽核為24 active／0 blocked。raw ch06→玩家ch07 已閉合 map6 六格 selector0 event26 的 raw `+6` gate、slots9..27 mode寫入與 state16 producer；enemy turn10 event25 只有在 state16==1 時才建立34→44 runtime、寫state17，戰後再經slot43 raw gate、唯一JOIN12 persistent record進 `town_ch08`。未踏格反例維持34 slots；先前「第10回合必定增援」與96-slot空白 frontier斷言均已撤回。raw ch07→玩家ch08 已撤回無 producer 的初始 groups1／8／9／10，正常入口為party10＋group0共29 slots；event27回合2..7逐組追加兩筆，戰後接受29..41奇數 frontier，依序執行layout、ACTING33／34、完整全黑、JOIN5、sync與chapter8，再進 `town_ch09`。raw ch09→玩家ch10 現保留60／61兩種強推論 frontier，依原始位址執行 DAC delta 0→63 淡出、sparse record/view patch、delta 64→0 淡入、FDTXT_010 index4／5、ACTING37、JOIN11／6、sync與chapter10，再進 `town_ch11`。raw ch15→玩家ch16 現正式接入76-slot persistent-first topology，四條 raw branch（round>18、inactive>4、word42<0x140、JOIN18 arm）均以 Docker/Xvfb E1 regression 進`town_ch17`；raw ch16→玩家ch17現以 map16的兩條 roster_has(18) branch接入layout、ACTING50–53、FDTXT_017 index5–8、JOIN16，60／61→61／62 frontier並進`town_ch18`，另驗證save/load；raw ch17→玩家ch18 現以 map17的55-slot runtime接入layout、ACTING56／57／58、FDTXT_018 index7–10、JOIN21／7，進`town_ch19`並驗證save/load；raw ch19→玩家ch20現以固定record0＋選15人和map19 group0建立83-slot入口，round15執行group1→84與JOIN28，round16精確略過，兩路共同JOIN25後進`town_ch21`。raw ch23→玩家ch24現以70筆FDFIELD＋16筆LOADCH建立86-slot runtime，依stage-before-draw與BIOS tick gate執行240＋60次indexed draw，完成隊伍同步後進`preparation_ch25`並驗證save/load；raw ch24→玩家ch25現以party16＋group0(46)=62開場，event56追加group1(8)成70，戰後追加group2成71供ACT75操作slot70，JOIN26／29後進`town_ch26`並驗證save/load。raw ch12→玩家ch13由table bytes固定interior entry `0x2389f`並接`town_ch14`。raw ch05／ch25／ch27則分別屬玩家ch06／ch26／ch28；第27戰天空之鑰成功分支不重用raw ch27。玩家第22至25戰已提升為 E1；玩家第29戰現已完成 raw ch28 post、隊伍同步、preparation_ch30 與存讀檔 E1；所有已接切片仍缺未修改一般玩家 DOSBox E2，因此不宣稱完整一致。位址證據見[`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)、[`fd2_ch15_post_ida.txt`](../data/fd2_ch15_post_ida.txt)、[`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)、[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)、[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)及各切片證據檔。 | partial | 以原版 handler offset／DOSBox input 差分核對每章是否進 town/shop/rest/preparation/ending；玩家第22至25戰與第29戰均已提升為 E1，但仍缺一般玩家 DOSBox E2；第7／8／10／16／17／18／20／22／24／25戰同樣尚缺一般玩家 DOSBox E2；ch00 `0x3241f` 尚缺 raw FDICON key，仍是明示的 RGBA E1 近似 |
> **2026-08-21 最新勘誤：玩家第23戰戰後已提升為 E1。** raw ch22 handler
> 現依原版 persistent-first constructor 建立16筆持續隊伍＋70筆 map records，
> 以86-slot正式 binding 消費18-slot layout、三個 raw predicate、`0x2189A`、
> `0x24B4D`、FDFIELD #69、FDSHAP #46/#47、`0x4DBFC` 與 FDOTHER #42 staging。
> 正常戰果確認完成後進 `preparation_ch24`，並通過隊伍同步與存檔／讀檔回歸。
> event52 精確增援時序與未修改原版同狀態畫面仍缺，故只列 RUNTIME-E1；
> 目前24個標準戰後節點為24 active／0 blocked；這是 E1 admission 現況，不代表全章 E2。

> **2026-08-21 歷史快照（已由下方正式 E1 勘誤取代）：** raw ch28 `0x25535` 對
> `0x22253([0x53BEB]-1,15,10,15,10)` 的來源限定 lowering、indexed unit
> presenter 與 `0x35E5A` 127-step palette pulse 已達 RUNTIME-E1。這不解除
> 本段當時仍未解除玩家第29戰的 fail-closed：map28 materialize 順序、group9 後的實際 runtime
> frontier、對話／視圖 owner、正式 binding、存檔邊界與一般玩家 E2 仍未閉合；
> 尤其不得把強推論的固定 slot93 寫入正式資料。

> **2026-08-21 map28 runtime 拓撲勘誤：** IDA 9.4 已固定正常 pre-handler
> 入口為20筆持續隊伍後追加group8的56筆，正式 battle seam 現保留該76筆順序，
> 不再把groups1..9全部當作開場單位。event75 selector1 的可編輯對話／live-row
> activation、event74逐回合groups4..7 staging，以及event76的raw-camp2 repeat／
> group1／六次palette pulse與indices2..6，以及event79單步RNG／兩個group1
> targets現已達 `RUNTIME-E1`；post的合法pre-frontier集合、`0x35BBA` raw清除與
> slot20 `+7/+8=0x7E` writer亦有窄`RUNTIME-E1`。group9 producer/order已閉合，
> `0x1DB65`原資源presenter、group9 transaction、正式戰後binding、隊伍同步與`preparation_ch30`存讀檔現已達`RUNTIME-E1`；仍缺一般玩家`PLAYER-E2`。
> groups2/3沒有已證實producer，故保持source-only。
> 此切片現已解除第29戰正式執行期的 fail-closed 並達 `RUNTIME-E1`，但尚未提升為一般玩家 `PLAYER-E2`。

> **2026-08-11 歷史勘誤；blocked 清單已由 2026-08-21 取代。** 玩家第22戰
> 當時由 fail-closed 提升為 E1；玩家第24戰又於2026-08-21以 raw ch23 indexed
> adapter 提升為 E1。此歷史清單又由上方玩家第23戰接線勘誤取代。

> **2026-08-13 勘誤：玩家第21戰天空之鑰鑄造固定演出已達 E1。**
> `0x242C9→0x24336` 已以真實 `FDOTHER #34`、`ANI #0`、原始幀順序與相對
> 調色盤相位接入正式 `campaign_full.json`；完整勝利路徑能回到 `town_ch22` 並
> 存檔／讀檔。這解除的是先前「鑄造動畫未接」的缺口，不包含相鄰
> `layout_units`、ACT63／64、第一個程序內相位或未修改原版同狀態 E2，故 UI-07
> 整列仍為 partial。證據見
> [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt)。

> **2026-08-21 勘誤：玩家第24戰戰後已提升為 E1。** raw ch23 handler
> `0x24C1E` 的 stage-before-draw、BIOS tick row gate、312×192 staging copy、
> ESI 0..59 DAC subtraction 與兩拍 palette gate 已由正式 indexed adapter
> 消費。正常戰役回歸保留 map23 的70＋16槽位，經戰果確認、`sync_party` 進入
> `preparation_ch25`，並在該節點完成存檔／讀檔。未修改原版同狀態逐幀與
> 程序入口相位仍缺 PLAYER-E2，因此 UI-07 仍為 partial。

> **2026-08-11 追加勘誤：UI-07 的舊總結已被本段取代。** 玩家第23戰戰前
> `ch22_pre` 現已由 `0x205da`／`0x135dd` 的 LOADCH 視圖證據接到
> `battle_ch23`（E1）；此段當時的 blocked 清單已失效。目前玩家第24、25戰均為
> E1；當時所列玩家第23、29戰 blocked 已由上方最新勘誤取代。
> 這只修正狀態分類，不代表未修改一般玩家 DOSBox E2 或逐像素 parity。

> **2026-08-11 追加勘誤：raw ch24 post 的共享角色參數。** `0x24e7b` 的
> `push 0x1d→jmp 0x237c8` 會跳過 direct-entry 的 `push 0x0e`，所以可編輯
> handler 的共享尾段角色是29，不是14；另一個直接建構器輸入是26。這只修正
> `source.addr` 保留的腳本／證據索引，沒有解除玩家第25戰的70→86 roster
> handoff、`Roster`／selector provenance或一般玩家 E2 gate。
> 詳見 [`fd2_ch22_pre_view_reset_ida.txt`](../data/ida/fd2_ch22_pre_view_reset_ida.txt)。

> **2026-08-13 追加勘誤：玩家第25戰已解除 runtime 阻擋。** 上段70→86
> handoff是錯誤重製拓撲；現已改為party16＋group0(46)=62、event56追加
> group1(8)=70、戰後追加group2=71。ACT75操作slot70，JOIN26／29後進
> `town_ch26`並通過存讀檔 E1。本段 blocked 清單已再由2026-08-21 raw ch23
> adapter 勘誤；該時點仍失敗即關閉的是玩家第23、29戰。此清單已由上方
> 最新勘誤取代；第23至25戰均尚缺
> 未修改一般玩家E2。

> **2026-08-11 可玩近似模式勘誤：** `FD2_APPROXIMATE=1` 只為尚未有正式 handler 的
> `postbattle_*` 提供可見的戰後整理提示；確認後沿 authored `next` 進入既有 town／preparation，
> 並先同步已物化隊伍。它不猜 JOIN、獎勵、章節或原版分支；未設定旗標的忠實模式仍失敗即關閉。
> 因此本矩陣的 blocked／E1／E2 分類不因近似路徑而升級。

> **2026-08-12 玩家第 28／29 戰前置 owner 勘誤（E1）：** IDA 分派表與
> `0x1088D` 資源公式證實 `0x33C9D`（raw index27）屬 `story_ch28`／map27／
> FDTXT_028；`0x33DBA`（raw index28）屬 `story_ch29`／map28／FDTXT_029。
> 舊版把後者錯接到前者，導致 map、slot count、party scenario 與 group8 全部
> 錯一章，現已拆成兩份正式 binding，並由 Docker／Xvfb 走到相應 battle node。
> map28 部署尾端也由錯誤的 raw-key 篩選 16 筆修為 control 宣告的20筆；map31／
> map32 的假部署格由1／2修為0。這關閉的是可執行 owner／資料流，不是 DOSBox
> 一般玩家逐像素 E2；證據見
> [`fd2_ch27_ch28_pre_owner_ida.txt`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt)。

| UI-08 town | `0x2cd16/0x2cf71/0x11eb0`；FDOTHER#11/#61/#62背景、#10 label、FDTXT `0x1ef+selection`、FDICON pulse、三variant×六selection座標；23筆raw variant已接production。ch02 postbattle 以 `/tmp` sandbox route patch 走完原版 handler，variant0 [`selection0–5 contact sheet`](../figures/town-hub-six-selections-original-vs-remake.png) 的每格都能和指定 remake pulse 做 raw RGB 整幀 hash 配對。另以固定雜湊原版的修改 LOAD 副本取得 variant1與variant2，兩者正常 selection0–4 都與對應 production node 的指定 pulse 逐幀整幀 AE=0，證據與限制見 [`native_town_variant1_e2.json`](../data/native_town_variant1_e2.json)、[`native_town_variant2_e2.json`](../data/native_town_variant2_e2.json) 及兩張對照圖。input trace另證實 Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape回selection5；`0x2ce7a/0x2ceac/0x2cef7` 不寫 pulse counter，已刪除方向鍵／secret reveal reset | partial（E1 + ch02 variant0 E2 + variant1/2 selection0–4 modified-LOAD E2） | variant2 selection5 的 BIOS 掃描碼／Enter；未修改一般玩家路徑與其他城鎮 |
| UI-09 shop | purchase、sell、standalone equip與transfer均有original-resource regression及production owner；strict adapters在raw projection不完整時fail-closed。ch02主選單、purchase list／Yes-No／不足金／收件者，以及賣出名冊／物品／Yes-No／成功／加款／返回均有多組同狀態AE=0。service2名冊與索爾item/status panel亦由正常route-patched商店輸入取得，整幀AE=1389／1433。service3來源提示／名冊、物品、目的提示／名冊也由同一路徑取得，AE依序為88／1391／2／88／321；其中物品清單的2點只在店員背景`(175,90)/(176,90)`，不是清單renderer。再沿普通鍵盤完成索爾短劍→悠妮，返回提示、索爾移除後清單與悠妮追加後清單均取得，成功交易四畫面AE=1391／82／2／286。可見內容與幾何一致，剩餘差異是店員／角色、翻頁箭頭或選取脈動相位，故兩者只列partial E2。重製同一跨角色交易穿越`leaveShop→town_ch02→JSON save/load`保存雙方compact/raw背包、裝備、能力、金幣與隊伍順序。六名具完整native provenance的typed party又從正式menu→purchase→Yes走到三列裝備收件者，經同一production input consumer完成scroll、滿欄／無合適角色原子返回與正常裝備／成功／扣款；所有UI job由實際Draw acknowledgment推進，達`RUNTIME-E1`。這些party／E2仍使用synthetic或screenshot-only bootstrap，不是完整campaign/native save E2。第25戰後`town_ch26`祕密商店E1也已接通。 | partial（E1 + ch02 多個商店狀態與賣出成功／加款／返回 route-patched E2 + service2／service3成功交易 partial E2） | ch26與ch02仍缺未修改原版完整玩家路徑；recipient scroll、no-recipient/full、service2動畫相位與mutation／restore、service3 self／empty／full／destination-cancel、church caller；其他章節route/state與原版存檔 |
| UI-10 church | `0x2d7bd` 左右四項循環；`0x3072f` dispatch `0→0x2ffa5` status、`1→0x2f8ea` item transfer、`2→0x30dc3` revive、`3→0x31385` class。class path已接 exact list/confirmation lifecycle；正式 `applyChurchClassChange→leaveChurch→town→JSON save/load` 整合回歸現保存 portrait/class/raw class、selector、成長、裝備重算、背包、隊伍順序與節點，並清除讀檔前教會暫態。raw0已接兩欄 roster與完整唯讀`0x17aed` status/items→command/MP lifecycle。raw1與shop service3共用`0x2f8ea`：FDTXT510/511/512、source/item/destination roster、`0x2dc55(mode1)`、FDTXT506滿欄與raw remove→append/recalc均已接；destination roster保留source本人，self-transfer依原指令做unequipped尾端重排。shop caller已有五個route-patched partial E2畫面，但不可外推church caller。revive已接 raw byte5 bit0候選、raw class×level費用、三列名單與完整feedback；成功animation/BGM lifecycle亦已接。2026-08-27正式`church_ch02`主選單固定相位與現行oracle達`AE=0/64000`，狀態旁證見[`native-church-menu-ch02-remake-e1.json`](../data/ui-traces/native-church-menu-ch02-remake-e1.json)。 | partial/fail-closed | 原版FD2.SAV與church caller DOSBox E2 visual diff；不重做已閉合的四項production-input owner |
| UI-11 preparation | `0x2d0d1` 城鎮出發提示使用 FDTXT `0x201`／`(95,119)` 與原 town source；`0x2cc04..0x2cc87` 無城鎮提示先清 VGA、使用 FDTXT `0x19a`／`(100,119)`，肯定才在關框後呼叫存檔。`0x318ad` 清除30旗標；`0x31a7c..0x31b08` 左右±1、上下±10；`0x31e80` 接三區背景、10欄角色格、游標、彩色／灰色角色及 `0x17fc0` 狀態。`0x31ea9..0x31ec6` 第一組數字直接取quota，`0x31edb..0x31efb` 第二組才取quota減已選，舊重製初始`00／19`已修成原版`19／19`。`0x320fc`直接證實record0固定、旗標i對應record i+1；重製已修正為固定1人＋可選15／19人，總上場16／20。`0x1297d` 待機週期與 `0x31d3c` 最終確認的完整 Draw 確認生命週期均已接；原始圖像索引、記錄或資源缺值即退回。同一固定晚期槽的初始狀態與合法相位0現有AE=0/64000成果圖 | partial（E1） | 依99%忠實度門檻關閉這個固定初始狀態；只抽測其他章節與交互狀態；`0x1f42d` 已更正為戰場進入演出，不屬此選人視窗 |
| UI-12 save/load | F5/F9 是重製自有快捷路徑，不得外推為原版戰場存檔；save package 自有 schema。原版 `FD2.SAV` 的 `0x59cb` boundary、rolling-XOR/u32 byte-sum checksum、4×logical `0xa28` records at `+0x312b`（metadata `0x28` + roster `0xa00`）已由真實 sandbox decode、`tools/fd2save.py` 與 `internal/fdsave` regression 覆蓋。合法 IDA 9.4 已固定 reader `0x2602c..0x26098` 與 writer `0x30012`：兩者只處理 metadata `+0..+9`；writer 只由 `0x2cad7` 直接整備與酒店呼叫。production 以雜湊綁定的 `0x526b9` gate table 把 raw chapter 1..29 還原到 `town_ch02..27` 或 `preparation_ch23..30`，先完整驗證 persistent record→typed party、節點型別與重複 identity，再原子套用 campaign cursor、gold、party 與 raw metadata 保存值；ch21/ch27 postbattle inventory gate 不會重播。標題 Enter／Space 現只經正式確認 owner：checksum-valid 合成槽完整還原 `town_ch02`、悠妮 typed/raw record、join order、789金幣、chapter1、HUD gate並清除舊 battle state／selection；竄改 envelope 留在 `loadslots`，campaign／party／gold／battle state 零修改且不落入 JSON loader。空槽及修改存檔chapter1有效槽畫面均與DOSBox全幀相同；未修改CONTINUE戰場按F5後存檔雜湊不變，確認一般玩家有效槽必須先從酒店／整備建立 | partial（空槽 E2；有效槽排版與 restore 為修改／合成路徑 E1，不升為一般玩家 E2） | 正常完成戰鬥後由酒店／整備建立槽位，再走標題LOAD的successful native-load E2；metadata `+10..+39` 其他可能 consumer、CONTINUE current-battle owner、delete/overwrite |

> **2026-08-26 UI-09 service2 正式輸入補證：** 四項商店選單 Right×2、角色名冊、
> 原版 item scan code、相容／不相容交易、空背包、0→11收合、同角色名冊重開及返回
> service selection 2 現由正式鍵盤與回歸共用 typed consumer。Down（scan80）才會在
> 目前兩筆直排物品由selection0移到1；Right（scan77）保持0是原版幾何，不是缺陷。
> 此項達 production-input `RUNTIME-E1`，原版 mutation／restore 畫面仍留 E2。

> **2026-08-26 UI-10 正式輸入補證：** 教會 `0x3072F` 的 status／transfer／revive／class
> 四分派現由 Ebiten 鍵盤與決定性回歸共用單一 typed consumer；四項皆在四段關框
> 與 source restore 後才發布各自 roster／文字／開框 owner。這關閉 menu dispatch
> 的 `RUNTIME-E1`。raw index 0 又由同一 typed status consumer 走完
> 名冊→十二段狀態開框→十四段指令面板切換→十二段關框／source restore→同一名冊；
> persistent ID 與 raw transient panel 全程沿正式 owner。這仍不替代 church caller
> 的未修改原版畫面 E2。
> raw index 1 亦已由單一 typed consumer 走完 source／item／destination／full：
> 跨角色成功、自我 remove→append、目的取消及八格滿欄原子拒絕均經完整關框／restore，
> 不再只有 shop caller 的 production-input 回歸。
> raw index 2 現也由 typed consumer 走完候選、費用確認、No／Escape、不足金、
> 成功 indexed timeline／BGM cue、最後一名後 empty feedback 與返回 menu。只有
> `reviveChurchUnit` 真正成功才播放成功演出；取消／不足金維持金錢與角色零修改。
> raw index 3 亦由 typed consumer 走完 class list→confirmation→取消／成功→重建名冊；
> default／special target table 缺列時零修改，成功才一次發布 portrait、class、raw class、
> selector、成長、裝備重算與背包消耗。既有 town→JSON 冷讀檔回歸保持通過。

> **2026-08-22 UI-03 勘誤：** 上表 UI-03 的17–22範圍已擴充為玩家
> command 17–23及25–27 `RUNTIME-E1`。25–27同樣依`0x1D6C8`播放#80 selector0與
> 八個DAC phases，最後一幀前不交易；ID25只清raw`+5 bit7`且不改target
> `Acted`，26／27經完整preflight後才走application。ID23另接mode-6目的地、八段
> palette及兩次`0x22253`離場／入場，第二段結束才發布MP／座標／action交易。
> 仍缺同狀態逐幀／逐音訊E2、status／expiry UI與command23精確camera核對；
> 這項勘誤不提升上述E2缺口。

> **2026-08-22 UI-03 command24 補充：** 正常學習鏈已更正為原始
> `unit+7` selector查11-byte growth row，再由byte10 `learn_idx`查12-byte command
> row；selector32的row4在Lv4授予command24。此前把portrait直接當command row
> index的重製端實作已撤回。這項資料與正式演出接線仍不證明未修改原版的一般玩家
> 連續E2，也不把既有battle background冒稱`0x29C90`原版轉場。

### UI-03 dispatch-wrapper recheck（2026-07-25，E0 partial）

Docker/Capstone 重新從 `0x18d8c` 入口線性追到 return，確認這是 action dispatch 的
**wrapper**，不能誤當 command-grid renderer：它先清 caller output 的 `+0` 與 global
`[0x53ec8]`。先前把 `0x1b83d(unitSlot,0)` 寫成「前序選擇」是錯的，現已刪除：它精確掃
unit `+0x0a + slot*2` 的八個 inventory slots，找 `bit0x40` 已設且 item ID `<0x80` 的第一格；
找不到時回 `-1`，wrapper 只設 output `+0=1`。命中時才經 `0x1b722 → 0x4e56c` 取該 slot 的
item record `+0xb/+0xc`，再呼叫 `0x14818(x,y,0,record+0xc,record+0xb,0)` 建立前序 target state。

其後 `0x1b8a6(unitSlot)` 精確計數八格中 `bit0x80` **未**設的 slots，因此它為零（所有 slots
空）時設 output `+8=1`；`0x1c269(unitSlot,0)` 為零及 `unit[+0x27] != 0` 都設 output
`+4=1`。前兩個 raw precondition 已閉合，三個 caller-visible flags 對應哪個可見 action／disabled
icon 仍未由 callee 或實機畫面閉合，SDD 保留 raw offsets，不能擅自畫圖示。`0x177fc`
是 wrapper 等待的選擇 loop，回傳 `-1` 則直接取消；非取消才按 `[0x53c57]` 分派：0 走
attack pipeline、1 走 `0x1cff0` command selector、2 走 `0x1bbdc` item selector，其他值才走
`0x13fd4/0x190ac` 的休息回復／格子互動路徑；`0x13fd4` 已由直接指令固定為
raw `+0x25/+0x26` 零值 gate 與 `floor(maxHP/5)` 回復，不是泛稱的 wait helper。
這補強 UI-03 的取消階層與 dispatch 邊界，但不增加
任何 renderer 或 flag 語意斷言。

`unit+0x27` 的 action effect 已額外由 `0x1598a` 固定：它先取 `0x1c269` command count，隨即讀
`unit+0x27`；count 為零或此 byte 非零都在**任何** command record、MP gate、target-grid 建立之前
直接走 zero return。因此 `+0x27` 是整個 native command submenu 的 gate，不只是 wrapper 的一個
局部 flag。`0x1eb64` 的 `lea [ebx+0x27]` 是 UI resource frame index，並非 unit access。後續已定位
command 22 的 `0x22BE1→0x22D1B` 會寫入 `rand()%4+2`；狀態名稱與所有 producer 仍未閉合，故不得稱其為
沉默、封魔或任一 status effect。

### UI-03 action overlay/input closure（2026-07-25，E0 partial）

`0x173e7` 先由四個 availability words 找第一個零值，寫 global current action `[0x53c57]`。
`0x177fc` 的 input loop 再以同一四-word state 拒絕不可用方向：scancode `0x48/0x4b/0x4d/0x50`
分別只在 word `0/1/2/3 == 0` 時選擇 `↑/←/→/↓` action `0/1/2/3`；`0x1c`/`0x39`
（Enter/Space）回 confirm，`0x01` 回 `-1` cancel。這是 command-grid `0x1d51d` 以外的 action
chooser ABI，現有 remake ring 的四向 mapping 只可作 interaction approximation。

renderer `0x1741c` 以 `[0x53a89]` 的 relative asset table 選四張 state-dependent images，透過
`0x4e9e4` 寫入 indexed overlay。它不是瞬間顯示：四張都從 shared origin `+0x390` 開始，每次
present 後 4-frame slide 分別更新 offset `up -= 0x8e8`（5 native rows）、`left -= 6`、
`right += 6`、`down += 0x8e8`。`0x175a9` 在開啟前備份 72×72 bytes（`0x1440`）到 private buffer，
`0x17643` 在每幀 restore。Docker Capstone 重讀 `0x176b4` 後，撤回「單純反向」的過度概括：它的
四幀 close 初始 byte offset 是 `[−0x23a0,0x378,0x3a8,0x2ac0]`，每幀改為
`[+0x8e8,+6,−6,−0x8e8]`。這證實十字狀 indexed overlay、方向與節奏。asset provenance 現已閉合：boot
`0x25c97..0x25cac` 將 `FDOTHER.DAT #2` 交給 `0x111ba`
並寫入 `[0x53a89]`。raw #2 是 untagged 78-cell offset bank（首 `u32=0x138` 即 directory end），cell
為 `{u16 width,u16 height,width*height indexed pixels}`；`0x4e9e4` 逐列 direct blit，index 0 preserve。
實測為 74 個 24×20、4 個 24×16 cells，strict `fdother.ParseRawCellBank` 與 player asset regression 已覆蓋。
`0x1741c` 的 relative table index ABI 已在 2026-08-11 由合法 IDA／Capstone 重讀：每個方向取
第一個四字表與第二個四字表，cell index=`3*firstArgumentWord +
2*secondArgumentWord`，再讀 `u32 relativeOffset=base[index]`、貼 `base+relativeOffset`。早期將兩個
乘數顛倒的 `3*availabilityWord + 2*directionState` 是錯誤斷言，已由
[`fd2_continue_action_overlay_ida.txt`](../data/ida/fd2_continue_action_overlay_ida.txt) 取代。battle
wrapper `0x18d8c` 的**第一**表固定 `[0,1,2,3]`，所以第二表為 0 時 cells=`[0,3,6,9]`，為 1 時
cells=`[2,5,8,11]`。chapter0 current-runtime 的一般 X11 `CONTINUE→Return` 空游標入口是
系統／行動覆蓋層，**不是原生 command grid**；其畫格**強推論**使用
`0x16f55` 初始表 `[7,5,6,4]`／`[0,0,0,0]`，顯示 cells=`[21,15,18,12]`；重製已完整載入 78 cells，
並保存 [原版／重製／差異 E1 比較](../figures/native-continue-current-command-compare-e1.png)，但確認
效果仍失敗即關閉，不能從圖塊推測四格 action 語意。先前把 `0x1728c` 的
`[0x12+(byte_51e61==0),0x14+(byte_51e62==0),0x16+(byte_53af9!=0),0x18+(byte_51aab==0)]`
套到 battle action 是錯誤；該 caller 選中方向後只切換這些 byte state 並重畫自己的巢狀四向 menu。
`fdother.BattleActionOverlayState` 現以 unit test 固化真正 battle table；它不替這個另一個 submenu
的四個 byte 命名。remake runtime 現可選擇性讀玩家自己的 `FD2_ORIGINAL_FDOTHER`／
`assets/original/FDOTHER.DAT`：FDOTHER#0 的 6-bit VGA palette 轉為透明 index-0 palette，#2 的完整 raw
78 cells 由 caller-owned lifecycle 依 opening `0..3`／closing `0..3` 幾何貼到 cursor。輸入在兩段
四-present 序列中被鎖定，confirm/cancel 的 child state 只在 close frame3 已呈現後提交；沒有把
`0x1741c/0x176b4` 未提供的 delay 猜成毫秒值。這不包含原版 asset，也不把 current remake 的
attack/spell/item availability approximation 說成 native `0x1b83d/0x1c269/0x1b8a6` 全等價。
[8-frame Xvfb artifact](../figures/action-overlay-open-close-remake.png) 與
[settled overlay screenshot](../figures/action-overlay-native-remake.png) 已證實 loader、palette、
cell geometry、frame order 與 font-independent draw path 實際出畫；它們不是原版 DOSBox 畫面對照。

重製端現已補上 `fdother.CaptureActionOverlaySnapshot`／
`fdother.RestoreActionOverlaySnapshot` 的固定 `72×72 = 0x1440` indexed 快照原語與
失敗即關閉測試。API 要求 caller 明確給出矩形左上角，故不把 cursor、camera 或
relative blit offset 猜成備份 owner；`ActionOverlaySnapshotOrigin` 現已由 IDA／Capstone
固定為游標各減一個 24-pixel cell 的 flat byte address（完整位址與雜湊見
[`fd2_action_overlay_snapshot_ida.txt`](../data/ida/fd2_action_overlay_snapshot_ida.txt)）。
現行 Ebiten adapter 仍由整幅場景重畫取代 private-buffer restore，待正式 runtime
consumer 與 DOSBox 同狀態畫面證據閉合後再接 renderer。

2026-07-25 renderer gate 縮小：native skin adapter 現至少直接套用 `0x1b83d` 的「equipped 且
ID `<0x80`」attack 前提，並在 raw `NativeCommandMask` 非零時以其作 spell availability；沒有 raw
mask 的舊 editable scenario 才退回 normalized `Spells`。attack target geometry、`unit+0x27` 的名稱及
item effect 仍未閉合，因此這不是 native gate 全等價。

2026-07-26 official IDA 9.4 重讀 `0x1741c/0x176b4`：open/close 都是四次 cell blit、present
(`0x11eb0`) 與 72×72 backup restore 的直線迴圈；迴圈本體沒有顯式 delay/wait call。因此 offset
sequence 是 E0，但每一幀應停留多少 presentation ticks 尚未由這兩個函式證實；remake 不得自行把
它命名或硬編成 60ms 等固定動畫時間。

### UI-03 native command-grid renderer closure（2026-07-26，E0）

official IDA 9.4 的 `0x1d51d→0x1ceed` 證實 command submenu 是 320×200 indexed-buffer 的四列 grid，
不是 remake 的單列 spell list：對第 `i` 個由 `0x1c269` 輸出的 command ID，`column=i/4`、`row=i%4`；
label 由 FDTXT_000 的 `0x1b9+commandID` 畫於
`x=0x12+0x64*column, y=0x67+0x16*row`。選中項 text palette index=`0xc9`，其他項=`0xcd`；同一欄的
MP/record `+5` 數字使用右側 `x+0x49`／`y+5` 的 numeric renderer。↑/↓在完整 list 頭尾 wrap，←只在
index≥4 時減4，→只在 `index+4<count` 時加4，故水平不 wrap；Enter/Space 還會以 unit `+0x44` 與 command
record `+5` 的 MP gate 再確認一次。這閉合 layout/input ABI，但不命名 `+5` 以外的 command effect，也不使
normalized `Spells` list 自動成為原版 command grid。

2026-07-26 label bridge：若玩家提供 editable `assets/data/command_labels.json`（FDTXT_000 的
`0x1b9+commandID` export），remake 會只覆蓋已載入 EXE spell rows的 presentation label；缺檔或
malformed JSON 維持 normalized labels。這改善既有 spell presentation 的原始文字 fidelity，並沒有把
legacy vertical spell UI 宣稱成 `0x1ceed` command grid，也沒有擴大 effect semantics。

2026-07-26 native command-grid runtime slice：當 player-provided FDOTHER VGA palette 與 editable
`command_labels.json` 都存在，ring 的 command branch 以 `NativeCommandMask` 開 native four-row grid，
label 直接採原始 `0xc9/0xcd` palette entries，↑↓／←→採 recovered ABI。confirm 現一律明確停在未接 native
two-stage target/effect，**不再**因 ID 剛好有 EXE spell row 就送入 legacy `CastArea`；缺任一 asset 則退回 legacy
spell UI。這是可視 layout/input slice，不是所有 command effect 或 native frame/background renderer 的完成宣告。

2026-08-11 補充：上述玩家（player）命令格確認（command-grid confirm）的限制不延伸到
敵方／友軍 NPC 的可編輯法術後備（fallback）。後者僅在原始 AI 路徑未處理且無錯誤時，
於無玩家 UI 的 `NextAIPlan→aiStep→CastArea` 路徑消費；它不是 `0x1ceed` 的命令確認、
不是原始命令效果／渲染器（renderer）證據，也不提升本矩陣的 UI-03 E0／E1／E2 等級。

runtime audit（2026-07-26，更新）：chapter `Scenario.Party` 現已保存 exact
`initial_command_mask`；產生器從 EXE `character_defaults.json` 依角色 index 帶入，並已重產 ch01..ch30。
loader 僅接受空值或四 bytes，避免以截斷值製造假 command inventory；persistent roster 亦保留 runtime
fifth byte。這只閉合 raw availability bridge，不以 normalized `Spells` 填補，也不證明 command effect、frame
background 或全部原版 input state。

2026-07-25 重讀 `0x1741c` 並以 `0x179d5` 交叉驗證後，收斂了一層 framebuffer anchor：四張 cell
的共同地址為 `framebuffer + 0x8088 + 0x18*cursorColumn + (0x18*0x1c8)*cursorRow`。
`0x11bfa/0x11c59` 的 cursor movement 證明 `[0x53ab9]/[0x53abd]` 是這對可視 cursor coordinates：
在右／下邊界時分別改寫 `[0x53aa9]/[0x53aad]` 的 camera scroll，否則才遞增它們。因此撤回「A/B
語意未證實」；`fdother.ActionOverlayOrigin` 已把命名後的 byte-address expression 獨立測試。剩餘的是
將 native indexed framebuffer 接到 runtime，以及 DOSBox visual-diff 驗證實際 skin。

## 明確缺口（不可用 fallback 掩蓋）

- `item` action 仍是提示字串，不能宣稱道具 UI 完成。
- touch 目前只移動游標，不能 confirm/cancel；沒有 gamepad/key-binding UI。
- `unit_present` 與 `indexed_transition` 尚未有 native indexed adapter；RGBA／色塊 fallback 僅供診斷。
- church 四項服務都已有正式具型別輸入 consumer；raw index 0／1 不再顯示
  「尚待原版 callee 完整接線」。仍缺的是未修改原版 church caller 的同狀態
  DOSBox 畫面與精確音訊，不是重製端 menu／callee 接線。
- battle `Tab` 可結束回合是現有配置，不代表已證實原版是 Tab 或可見選單；需 E0/E2。

## 可重跑盤點命令

```sh
rg -n 'func \(g \*Game\) (enterNode|campInput|Draw)|ringInput|尚未實裝|尚待原版' remake/cmd/fd2/main.go
git diff --check
test ! -e /tmp/fd2cap
```

### UI-01 DOSBox title oracle（2026-07-25，E2 partial）

`tools/docker/fd2-dosbox-screenshot.Dockerfile` 以既有 Xvfb/xdotool/ImageMagick image 建立隔離 runner；
它只接受可寫的 **`/tmp` game sandbox** 掛載與明確 `/tmp` shots mount，原始 `FLAME2` 不掛進容器。
以 `svga_s3`、`fixed 18000` 跑 `wait:2; Escape ×4; wait:8` 後取得
`docs/figures/title-original-dosbox.png`（320×200 crop）。畫面直接證實 title 的 START／LOAD／CONTINUE
縱列與 START cursor；這是 UI-01 的 E2 畫面 oracle，不證明 title input dispatch、存讀檔語意或 remake
title renderer 已完成。

同一 timeline 在 title 選 LOAD 後可重現 `docs/figures/load-empty-original-dosbox.png`：原版在空 save
sandbox 顯示四列 `1)` 到 `4)`、每列「無儲存記錄」，第一列有 selection outline。這是 UI-12 的空槽
E2 oracle；它沒有有效存檔資料，因此不證明 record layout、LOAD 成功路徑或 SAVE overwrite confirmation。

START 分支首個可重現對話 crop 為 `docs/figures/ch01-dialogue-original-dosbox.png`：第一章場景中可見
左側 DATO portrait、下方藍框、兩行文字與框底中央 page indicator。這提升 UI-05 的一個 lower/left
E2 anchor；它不涵蓋 upper/right speaker、FFxx control code、完整 pagination timing 或 remake renderer。

重製端標題選單的 Docker／Xvfb 實際擷取為 `docs/figures/title-remake-runtime.png`
（640×400 輸出、2× 原生內容；2026-08-26由目前原始碼重建）。三列改為原生
`y=164／173／182` 後，最近鄰縮回320×200與上述 DOSBox oracle 達整幀
`AE=0/64000`。這關閉穩定主選單畫面，但它仍只是重製端 E1 執行期證據；不外推
完整開場幕序、`logozoom`、所有 LOAD／CONTINUE 狀態或一般玩家 E2。

### D8 native trace（2026-07-25，E0 partial）

Docker/Capstone 直讀 `0x1a30b`：battle-entry 先掃 unit buffer、以 `0x1da16` 更新 320×200
offscreen surface，再呼叫 `0x11eb0` present；接著呼叫 `0x1a813`／`0x1a866`，並在 phase
`[0x53ecc]==0` 時進入 `0x1a7bd → 0x1d80b → 0x1a7f1`。其中 `0x1a4c7` 明確呼叫
`0x1f1cc(0x52)`、20ms、`0x1f30a(0x52)`，完成 redraw 後才進後續 dispatch；`0x1f1cc`
與 `0x1f30a` 都配置 64000-byte indexed buffer、呼叫 `0x15f0e` 取資源並逐幀
`0x11d40` palette/present。進一步 trace `0x15f0e` 可確定它以 `base + 6 + frame*4`
取 frame offset，descriptor 前兩個 signed words 是 width/height，先配置
`width*height+8` 再經 `0x4e96f` 解壓、`0x4e85b` 以 stride 寫入 indexed surface；
這是可重用的 frame-resource ABI；`[0x53a81]` 的 loader provenance 已由 UI trace
確認為 `FDOTHER.DAT` resource #5 的 `LMI1` 容器（doc35 §4.2.5），remake 已新增
strict `fdother.ParseLMI1` 與 codec regression。
Codec/blit correction：`0x4e8af` 對每個 decoded pixel 都直接 store，index 0
也是 opaque overwrite；舊「index-0 transparent preserve」斷言已撤回。
`LMI1Entry.BlitOpaqueAt` 保存此路徑，`BlitAt` 則只留給另有證據會 preserve zero
的 caller；兩者都要求顯式 surface/anchor，未擅自接入 D8 layout。
實際玩家 `FDOTHER.DAT#5` regression（138 entries，#0x52=72×14）另證實 directory
offset 只標示 entry start：`0x4e916` 的 repeat 可跨下一個 offset，原版依 width×height
停止，因此 parser 不得把 next offset 誤當壓縮 stream 結尾。
`0x1f42d` 不是文字 helper：`0x1f1cc` 以 offset `100,75,50,25,0` 各呼叫一次，
每幀把 LMI1 **entry #0x52** 貼到 offscreen `(85-offset,82)` 與
`(165+offset,81)`（stride 456），present 一 tick，再以 `0x15e71` restore；這是
兩側 UI cell 的五幀滑入。它的反向 path 由 `0x1f30a` 使用同一 helper。這只閉合
indexed cell/座標/節奏，不足以命名 MAP/TURN 欄位或確認其為「行軍確認圖」，故 UI-11
仍 partial。

下一輪先處理 UI-03／UI-04 的原版 dispatch 與 weapon reach provenance，再補 D8 的
MAP/TURN text source 與 YES/NO input ABI；在此之前不新增猜測性 renderer。

#### D8 scope correction (2026-07-26)

官方 `0x1a30b` 本體沒有 `0x15f84` 呼叫；它先以 raw unit-record gates 做 `+0x40` 向 `+0x42` 的 `max/5` transition，再進 indexed redraw 與 `0x1f1cc/#0x52` slide。故目前 D8 證據只支持 battle-entry indexed choreography，不支持 MAP/TURN/ENEMY/FRIEND/NPC 字串或 YES/NO input；那些欄位仍是缺口。

### UI-04 geometry slice（2026-07-25，E0 partial）

`0x14818` 先以固定的 table record 0（`0x61646`，20 bytes）呼叫 `0x4e040`，並將原始
`(x,y,mode)` 傳入，建立／更新 target grid；`0x4e040` 以 mode 作 seed grid byte，內層再依
tile flag 與 record byte table 的 cost gate 擴張。此 raw mode 的玩法名稱尚未確定。其後才有可獨立
證實的一層幾何：以 source cell `(cx,cy)` 掃全格、
對每一格算 `abs(x-cx)+abs(y-cy)`，只有嚴格小於 caller radius 的格寫入 `0xff` marker。
最後掃 0x50-byte unit buffer：死亡／inactive unit 跳過、非 marker cell 跳過，再依 caller selector
對 `unit+6` camp 過濾，將 slot index 寫入可選 target output。當另一個 mode argument 大於等於
`0x10` 時另走一條十字形 clear path；它的玩法語意與 weapon `min/max` 欄位尚未完成 caller-dataflow
對照，不能把這個 raw `radius` 直接等同 remake `AtkMax` 或宣稱已解 LOS。

補作 `0x1cff0` caller 的 stack-dataflow 後，`0x14818` 的參數順序已可固定為
`(x, y, output, mode, radius, campSelector)`：`mode` 是第 4 參數、上述嚴格曼哈頓比較使用
第 5 參數，unit filter 使用第 6 參數。特別 command `0x17` 傳入 `record+3` 作 mode、`1`
作 radius、`record+6` 作 selector；一般 command 則傳 `record+4` 作 mode、`0` 作 radius、
`record+6` 作 selector。因此一般 path 不會在這一 call 新畫 diamond，而是消費前序已建立的
marker grid。`record+3/+4` 仍不能在未追到 producer 前命名為 weapon min/max。

該 record 的 producer 已定位：`0x1cff0` 將選單結果 ID 傳給 `0x4e516`，而
`0x4e516(id) = 0x619fd + 7*id`。因此 `+3/+4/+6` 是靜態 7-byte command ABI 的欄位，
不是這個 handler 自行組出的暫存結構；在有 field-name 或實機資料對照前，仍以 raw offset
記錄，不擅自命名成攻擊／法術的 min/max range。

command ID 並非 four-way ring 的固定索引：`0x1c269(unitIndex, out)` 讀取該 0x50-byte unit
record 的 `+0x1a..+0x1e` 五個 byte，逐 bit 把 set bit 寫出成 `byteIndex*8 + bitIndex`（0..39）。
`0x1cff0` 以這份 list 的目前選項取得 ID、再呼叫 `0x4e516`。因此 UI-03 的完整 SDD 必須資料化
command bitmask、ID→label/rendering、enable gate 與 cancel hierarchy；現行四格 `ringInput` 只能保留
為 provisional interaction，不能冒充原版完整 command menu。

bitmask 的 construction ABI 也已定位：`0x10f7f` 將 source record `+0x0d..+0x10` 的 4 bytes
copy 到 unit `+0x1a..+0x1d`，並清 unit `+0x1e`；另一 construction path `0x11399` 同樣 copy
4 bytes（其 source `+8..+0xb`）再清 `+0x1e`。後續 `0x1d7fb` 以 `commandID/8` 選 byte、OR
對應 bit 寫回 `unit+0x1a` 起的 array。因此 40-bit 是真實 runtime ABI，但初始 source 只有 32 bits，
第 5 byte 由後續流程擴充；source record 的遊戲語意仍不可未證實地命名。

原版另有已證實的可用性 gate：`0x159fa` 先取得同一份 `0x1c269` list，逐個取 command record
`+5`，僅當該 byte `<= word[unit+0x44]` 時保留；`+0x44` 已由 battle HUD 證實為 current MP。
因此 `command+5` 是 MP cost/requirement 的 E0 ABI，而不是 UI 的任意排序值。bitmask 的寫入
producer、每個 ID 的名稱與其他 enable gate 尚未閉合。

`0x1d51d` 是這份 command list 的 input loop（不是 `0x19953`）：每次先 call `0x1ceed` render，
再取 `0x1c269` count。scancode `0x48/0x50` 對線性 cursor 做 -1/+1 並在 `[0,count-1]` wrap；
`0x4b/0x4d` 分別在 index >=4 時 -4、在 index+4<count 時 +4；renderer 座標證實每欄四列（不是四欄）。
`0x1c/0x39`（Enter/Space）重新查 `command+5`，只有 current MP 足夠回傳 confirm；`0x01`（Esc）回傳
cancel。`0x1ceed` 的 list index `i` 使用 `x=0x12+0x64*floor(i/4)`、`y=0x67+0x16*(i%4)`，以
`0x15f84([0x53a7d], 0x1b9+commandID, ...)` 顯示 label，並以 `0x187d6` 顯示 command `+5`。這鎖定
label index ABI 與 geometry。常駐 `[0x53a7d]` table 已由其他 callsite 的 direct trace 對齊為
FDTXT_000；raw strings `0x1b9..0x1e0` 已匯出為 `docs/data/command_labels.json`。其中空字串與
系統訊息 slot 證實文字不等於可達指令；cursor cell／不可用 command 的可見表現仍待 resource／實機畫面，
不得猜作四方向 ring。

補充 record evidence：`0x4e516` 的 `0x619fd+7*id` 對 IDs 0..35 byte-for-byte 等同 EXE spell table，
所以 command record 的 `+3/+4/+5/+6` 可分別沿用 spell row `dist/range/mp/target` 的已證實欄位；資料掃描中
FDFIELD/character default initial masks 只出現 IDs 0..30。36..39 的鄰接 bytes 與 FDTXT 系統訊息不能被當成
可選技能。

動態 command producer 亦已定案：level-up routine `0x1e292` 讀 portrait growth row 的 `learn_idx`，以
`0x4e4a2` 取固定 12-byte learning row，掃最多六組 `(level, commandID)`，level 命中便呼叫
`0x1d79c(commandID, runtimeSlot)` OR bit 並顯示 FDTXT_000 #587。20 rows 已原樣導出；這不是一般
selector effect trace，故不代表所有已學 command 已有可執行 remake effect。

`0x4e040` 並非僅由這個 target caller 使用：`0x14344` 先以 unit `+0x20`（fallback record
`0x13`）透過 `0x4e555` 取另一個 20-byte record，再把 map grid、terrain table 一併傳入。
其內層 `0x4e16e` 讀 tile flag 與該 record 的 byte table 後決定是否擴張。故目前可用的
E0 模型是 **seed mode + table + terrain/cost gate + marker + unit filter**；尚不可把 target highlight
reducer 成單一菱形或宣稱其完整路徑／LOS 規則。

### 2026-08-09：玩家近戰結算消費鏈（E1 重製端）

戰場 action menu 的近戰確認現在進入 `battle.State.AttackWithRNG`，使用由
`Game` 注入的固定種子亂數，並把 `AttackResult` 的未命中／暴擊／傷害／經驗交給
訊息與 FIGANI 演出。缺少亂數來源時先停止，測試也確認不會先改寫攻方或守方狀態。
這是重製端消費鏈的 E1 回歸，不是原版 raw 攻擊 ABI、完整 indexed settlement
或一般玩家 E2；命中表、劍技與完整經驗介面仍維持未關閉。

### 2026-08-09：完整 native map frame 與 action overlay 疊加（E1 重製端）

`drawNativeMapFrame` 的 admission 現允許已 materialize 的 action overlay／native
command grid 疊加；原先這些 modal 被排除時，短地圖會露出非原版黑帶。Docker／Xvfb
目前 source 的 [完整 640×400 畫面](../figures/action-overlay-native-remake-fullframe.png)
已保存供審查。資源缺失仍回退，這不是原版 DOSBox E2。

### 2026-08-09：戰後結果 Enter 與城鎮邊界（E1 重製端）

實際 runtime 回歸確認敵方回合完成後先停在 battle 結果畫面；Enter 才進入含
`sync_party` 的 postbattle cutscene，淡出完成後才進 town。這修正了只測
`Campaign.Advance` 而未測玩家輸入邊界的缺口。證據為
`TestEndTurnEnemyPhaseResultEntersPostbattleCutsceneThenTown`，不提升為原版
DOSBox E2，也不替未綁定的章節 handler 猜測 renderer 或 campaign 語意。

### 2026-08-10：戰場命中畫面全螢幕紅罩勘誤（E0／E1）

IDA／Capstone 已證實原版命中效果是受原始 frame flag、傷害步進與
`0x29f72` 輸出欄位控制的 DAC 脈衝，不是每次命中都套用 RGBA 全畫面紅罩。
重製端已移除沒有 raw provenance 的全畫面紅罩，並只保留原版 impact 參考圖支持
的守方紅剪影；攻方維持 FIGANI 原色，避免背景、狀態欄、台座與攻方一起泛紅。
完整位址、雜湊與埠序列見
[`fd2_battle_impact_pulse_ida.txt`](../data/ida/fd2_battle_impact_pulse_ida.txt)。
本項只關閉一個已確認的重製端視覺偏差，未提升戰鬥幀時序、傷害演出或一般玩家
同狀態比較為 E2。修正後代表性畫面與擷取條件見
[`battle-impact-no-global-tint.png`](../figures/battle-impact-no-global-tint.png)／
[`battle-impact-no-global-tint.json`](../data/ui-traces/battle-impact-no-global-tint.json)。

### 2026-08-11：未修改原版一般玩家回合錨點（E2 partial）

Docker DOSBox 以未修改 `FD2.EXE`／`FD2.SAV` 的複本，從標題與開場正常按鍵
走到第一戰第一個我方單位的玩家指令格；原始輸入時間線、執行檔／存檔雜湊、
320×200 PNG 的 MD5／SHA-256 與限制見
[`native-player-turn-original.json`](../data/ui-traces/native-player-turn-original.json)，
畫面見 [`battle-player-turn-original-dosbox.png`](../figures/battle-player-turn-original-dosbox.png)。
這只閉合 UI-02／UI-03 的「一般玩家回合可操作畫面」原版錨點；尚無同一
raw runtime 狀態的重製端配對，也沒有證明敵方 AI 回合、攻擊演出或全戰場畫面
已與原版一致。

### 2026-08-11：UI-03／UI-12 原版 current-runtime E2 錨點勘誤

工作區內的原版 `FD2.SAV` 已由 Docker `tools/fd2save.py` 驗證為 checksum-valid
current-runtime 快照（chapter0、12 筆 runtime records），不應再寫成「沒有任何
可用原版存檔」。未修改 `FD2.EXE`／`FD2.SAV` 從開場正常進入 `CONTINUE`，再以
Enter 開啟玩家指令格；兩張 320×200 client crop、輸入時間線與雜湊見
[`native-continue-current-runtime-e2.json`](../data/ui-traces/native-continue-current-runtime-e2.json)。

證據等級為原版一般玩家 UI-02／UI-03 的 E2 partial：它只覆蓋 chapter0 current
runtime，不能外推到 ch22／ch23／ch24／ch25／ch29，也不代表重製端 CONTINUE
handoff 或同狀態逐像素 parity 已完成。UI-03 與 UI-12 的正式重製 owner 仍保持
失敗即關閉。

同日另保存一張重製端 `story_ch00_handler`→`battle_ch01` 的 E1 執行期畫面
[`native-battle-ch01-remake-e1.png`](../figures/native-battle-ch01-remake-e1.png)，
以及可重生條件與雜湊 [`native-battle-ch01-remake-e1.json`](../data/ui-traces/native-battle-ch01-remake-e1.json)。
截圖快進器只在明確的 `FD2_SHOT_FAST_FORWARD=1` 模式執行，故不提升 UI-02／UI-03
的一般玩家 E2，也不表示與上面的原版 current-runtime 已是同一 raw roster、鏡頭、
游標或 tick。

### 2026-08-11：重製端 current-runtime E1 配對基準與原生指令環座標修正

同一份固定雜湊的 `FD2.SAV` 已在 Docker／Xvfb 以普通 X11 鍵盤事件走過重製端
`CONTINUE`，並保存重製端原生戰場畫面
[`native-continue-current-runtime-remake-e2.png`](../figures/native-continue-current-runtime-remake-e2.png)。
它與原版 E2 crop 的最近鄰 2×比較為 AE `164`、RMSE `50.2631`；這是原版 E2
與重製端 E1 的同存檔配對基準，不是逐像素一致。重製端的
`FD2_NATIVE_TITLE_TICK=0` 是明確提供的計時夾具，故尚未宣稱一般玩家 BIOS 時鐘
或敵方回合 E2。

本輪另修正 `drawRing` 將原生 320×200 地圖座標直接拿到 640×400 畫布的偏移：
`actionOverlayAnchor` 只在完整 native map frame admitted 時套用 2×呈現縮放，
normalized 路徑仍維持 1×。回歸測試
`TestNativeActionOverlayAnchorUsesPresentationScale` 固定 `(8,16)` 單位的
native anchor `(336,144)`，但不替尚未閉合的 icon availability／command semantics
猜測接線。

完整輸入、雜湊、限制與 helper 來源見
[`native-continue-current-runtime-remake-e2.json`](../data/ui-traces/native-continue-current-runtime-remake-e2.json)。
