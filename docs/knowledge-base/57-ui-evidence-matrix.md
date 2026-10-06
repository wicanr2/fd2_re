# 57 — UI evidence matrix（SDD-1 baseline，2026-07-25）

> 2026-09-15 起，全戰役原版一致依 [111](../goal/111-goal-original-parity-campaign-20260915.md) 由代理程式以 dosgolem 逐章推進；v.1.0.19 完整包與當時修正／未通過項目仍以 [94](94-ch01-town-parity-20260908.md) 為準。

2026-09-08 最新玩家路徑：[94](94-ch01-town-parity-20260908.md) 優先於下列歷史
進度。新 AppImage 已由普通 START 到第三回合援軍，發現 JOIN 阻塞；修正重跑中。
dosgolem 的戰場黑底已證實來自大型 FDSHAP 讀檔截斷，修正後自行重生完整地形。
v.1.0.14 第二輪圖片因視窗位置被裁切，排除完整畫面評分；旁車只用於定位流程。
v.1.0.15 改以完整尺寸檢查重生。兩側首關至城鎮仍未達 PLAYER-E2。

2026-09-13 死亡掉落物品滿欄 UI 已依 IDA 9.4 重核改為原版「丟棄擊殺者舊物品」
流程，不再錯掃隊友空格。FDTXT `0x1B1/0x1B2`、YES／NO／Escape、八格 item panel、
方向鍵與面板收合後原子交易均由正式 `Game.Update`／indexed `Draw` 消費並通過逐幀
acknowledgement 回歸，列 `RUNTIME-E1`；尚無未修改原版同狀態畫面，故不升 E2。

2026-09-08 延長對拍：v.1.0.13 已修鏡頭終點交接、跨頁保留前文、翻頁箭頭
與捲動矩形，實際 START 前38頁完成；dosgolem 原版收據延伸至65頁。
省略號字模碰撞與剩餘背景差異仍未關閉，全段一致未提升。
唯一目前結果與各頁未遮罩收據見 [93](93-dialogue-cause-20260908.md)。

2026-09-08 最新局部反證：[王宮對話原因](93-dialogue-cause-20260908.md)。前兩句
框內畫面匹配，第二句原有 29,093 像素差異來自漏接說話者聚焦。v.1.0.10
已補聚焦及逐字頭像，正常 START 第二句完整零差異；第一句仍有 482 個框外
像素不同，不將此兩句抽樣提升為全段 PLAYER-E2。

> **2026-09-07 開場操作複核：** 既有 v.1.0.3 完整包與原版 START 不一致；
> 實際原版主題操作還揭露選人視野、資訊框、圖示間距及回合後誤判敗退。
> 修正規格與唯一逐項驗收狀態見 [`92`](92-opening-ch01-input-audit.md)。
> 下方舊局部畫面錨點不能推成「從開場到父子登場前操作一致」。DOSBox
> 合法 CONTINUE 的 TURN 003 結束後已觸發父子對話，沒有第五回合前仍未登場的證據。
> v.1.0.7 已以正常 START 補驗中立換人、F2／已行動狀態與新回合聚焦；
> v.1.0.8 另補普通物品寶箱頭像、YES／NO 與物品名稱回覆。記為
> `RUNTIME-E1` 加實際封包普通輸入收據，DOSBox 輔助圖不冒充正式 dosgolem E2。

> **第一輪抽樣策略（2026-08-27）**：介面收尾改採早／中／晚期正常輸入的代表性
> 抽樣，與戰役、戰鬥、存檔及終局合計至少60個樣本，用來建立95%產品信心。
> 這不會把partial E1自動升為E2，也不會把未抽到的畫面宣稱逐像素一致；本矩陣仍
> 保留每個畫面的證據等級與已知差異。樣本配額見[`REMAKE-STATUS.md`](../REMAKE-STATUS.md)。

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

> **2026-08-28 標題素材來源重驗：** 正式標題已改由分離pack消費#69..#73、#101、
> #7／#8、#100／#99及#75／#76，不再讀舊title PNG或FDOTHER archive。由目前原始碼
> Docker／Xvfb重擷取的640×400 PNG與既有`title-remake-runtime.png`檔案SHA-256同為
> `17e6fc41cee0ce706aaf1d809e1f1b986053cea670f7065c1bd00c62cbe1afa8`；最近鄰縮回
> 320×200後仍與DOSBox oracle達`AE=0/64000`、raw RGB SHA-256
> `eb5957077536a840050e2ee52f4a3001156f6c95814c0c2d0dd6977e5929a342`。此重驗保留
> 原有畫面E2錨點，不外推完整開場或ANI archive遷移。

| 畫面／流程 | 視覺還原估計 | 直接證據與主要差距 |
|---|---:|---|
| title/main menu | 選單狀態與指定畫面已驗證；完整開場 E1 | 2026-08-26 由目前原始碼重新建置並以 Docker／Xvfb 擷取 [`title-remake-runtime.png`](../figures/title-remake-runtime.png)，最近鄰縮回320×200後與 DOSBox oracle [`title-original-dosbox.png`](../figures/title-original-dosbox.png) 達整幀 `AE=0/64000`；三列原生座標為164／173／182。`sub_1F894` 的535列捲動、六個插播點與 AFM 3→4→5→6→7→8→0→1 已依 canonical IDA 證據接入。2026-08-27再把AFM 3前的`FDOTHER #74 + #76`漢堂發行商畫面接入119幀60Hz近似；正式runtime第60幀與解碼oracle達640×400 `AE=0/256000`，見 [`title-publisher-remake-e1.json`](../data/ui-traces/title-publisher-remake-e1.json)。舊DOSBox `frame_000`實為AFM 3龍紋過渡、不是發行商標誌。後段兩次`sub_286BD`亦已依IDA直接指令修正為半開區間0..254的索引色盤內插，正式播放紅幕→真實ANI #1→近白標題；[淡入影格](../figures/title-transition-remake-e1.png)與原版frame193的MAE為0.077／255，達99%玩家可見門檻。2026-08-29 ANI#0..#8共289幀與逐幀六位元DAC已分離，固定archive oracle逐indexed pixel／DAC相同，正式標題不再讀`ANI.DAT`；此為資料owner遷移，不另提升畫面等級。非原版 F2 常駐提示已移除，逐幕按鍵契約保留。只剩精確音訊與原版完整啟動E2，故不升完整E2。LOAD／CONTINUE 的來源限制另依 `58` 記錄 |
| tactical field/HUD | `RUNTIME-E1`；ch01、ch29及ch30為範圍受限的 E2 候選錨點 | 目前正式 `story_ch00_handler` 截圖 `native-map-ch01-remake-handler.png` 與 `native-map-ch01-original-video.png` 仍是不同狀態，只能證明原始資源／渲染輸入及隊伍 handoff 已被消費；重製圖已改以唯讀原版 `FDOTHER/FDSHAP/FDICON` 與 IDA 已證實的 FDFIELD b1 selector 產生，修正舊 b0 映射的敵軍圖像錯誤。舊 pair 的場上單位、游標與 HUD 差異不能作為目前渲染器缺陷證據；舊 `native-map-ch01-remake.png` 是直接節點除錯歷史證據，不再作正式比較。較早 E1 raw 相機／游標欄位也不等於畫面像素一致。2026-08-10 另以同一 `FD2.SAV`、相機、游標、回合與單位狀態建立 DOSBox／重製逐幀範圍比較，最近鄰縮放後內容區只剩 22 個畫布邊界差異像素，記為 ch01 scoped E2 candidate。2026-08-26 又以外部末關候選從正式標題普通鍵盤 `CONTINUE` 抵達 `battle_ch30`；旁車配對round12、camera `(16,16)`、cursor `(21,20)`，正式indexed六階段寫入18筆鏡頭內單位，且foreground／HUD未覆蓋unit stage。舊洋紅亂圖已勘誤為未提供`FD2_ORIGINAL_FDOTHER`而走PNG fallback，不是renderer缺陷。合法IDA其後閉合並接通`FDOTHER #55`輔助底面與`0x11CAC(0)→0x4DFCC`的DAC `0xE0..0xEF` cycle；同一typed戰況的合法aux phase10／palette phase0達320×200整幀`AE=0/64000`。另以fixed-hash `fd2004`候選由未修改原版普通CONTINUE抵達玩家第29戰並以Return開啟指令環，重製正式handoff亦還原76筆場上單位、31人持續隊伍及相同round／camera／cursor；因第三方來源仍只列候選。這些候選都不能外推成其他章節或完整操作界面E2，精確音訊與完整存檔provenance仍缺。2026-07-29 稽核發現只有 map0 曾帶 `native_tile_blit_modes/native_terrain_control`，現已從雜湊鎖定的 FDFIELD／FDSHAP 同步至全部 33 圖並有全圖 regression；這只閉合 renderer inputs，不是全遊戲視覺 E2。ch26 又由 pre-handler PAN/FOCUS 與 cursor state machine 閉合 event61 所需 runtime view/HUD E1；ch27 的 selector0→event62→event63 raw camp0 敵軍 AI 前 runner、兩批增援與全白／恢復演出已達重製端 E1，戰前 view／selector0 及 inherited HUD owner 也已閉合並接線。gate A 由存檔保存、anchor 為程序內持續、gate B 由 controller 物化；ch02+ 其餘畫面、一般玩家／CONTINUE 同 roster/event/tick DOSBox 像素差分、該時點角色 raw record 的實際值，以及 `0x12c0d` 的 exact raw lookup predicate/order 仍待補齊 |
| action/command/item/target UI | 45–55% | action skin、command grid、item panel已有原資源 indexed adapters；command grid 另有空 ID／selected 越界失敗即關閉回歸。死亡掉落滿欄流程已接 FDTXT `0x1B1/0x1B2`、YES／NO／Escape、擊殺者八格 selector 與收合後原子 replace，達 `RUNTIME-E1`，不再錯轉交隊友。command0完整`0x2A6BD→0x26152` presenter現由玩家與`Raw53AF9==0`的敵方ID0共用，但各自保留cleanup／continuation。玩家物品type20／21／24正常確認已共用`0x1CD17／0x1CAC7` indexed owner，最後畫面邊界後才發布傷害，尾停後才結束行動；缺素材時保留目標模式並零交易。33圖固定非玩家command mask排除mode8後的正常producer均已有indexed owner；剩餘差距是完整availability動態對照、selector 6/7+、狀態高階名稱與同狀態DOSBox畫面／音訊比較 |
| story dialogue | 十九個正式切片（E1；全章仍partial） | 十九個切片均直接由原始說話者控制碼與`FFFE/FFFD`建立可編輯頁面，完整清冊以`58`為準。2026-08-27新增`ch25_post`七組分支呼叫共41句：固定版`FDTXT.DAT`第26項為5,004 bytes，逐句control／operand與編輯行由raw等值驗證；正常57-slot勝利路徑依`event_state[12]`分別播放18或33句，兩臂都完成ACTING77–80、同步、`town_ch27`與全新`Game`冷讀。`ch27_post`另由正式64-slot第28戰勝利路徑播放`FDTXT_028`五句，再同步隊伍並進`preparation_ch29`冷讀；舊80-slot說法已勘誤。`ch20_post`成功／不足臂則分別播放26／14句並完成各自天空之鑰契約。多字元字形仍以`glyph_pages`保存，見[`native-story-multirune-glyph-token-e1.json`](../data/ui-traces/native-story-multirune-glyph-token-e1.json)。2026-09-14 dosgolem 三臂證實第一關事件對白與第二關回合起手對白的 Escape／Enter 單次推進同義，無輸入反對照分流；正式戰鬥兩分支已共用 typed gate，見[`battle-dialogue-esc-vs-enter.json`](../data/ui-traces/battle-dialogue-esc-vs-enter.json)。各切片共用已閉合的開框、逐字形、嘴型及快照收框；同狀態DOS畫面與其餘呼叫端仍缺，未綁定呼叫端仍走既有RGBA路徑 |
| full-screen battle presentation | 40–50% | FIGANI/AFM、部分 status frame與局部 pixel-equal slices可重播；2026-08-10曾宣稱戰鬥姓名已改接FDOTHER#4 16×16 glyph，但2026-08-25實檔稽核發現`unicode_to_glyph.json`的`_comment`讓載入器整批退回TTF；修正解析並鎖定panel`+(5,4)`後才真正達`RUNTIME-E1`。固定命中fixture已把HP顯示改為impact邊界立即提交post-hit值、守方剪影E1色值對齊原版RGB `(190,0,0)`，守方待機幀也改消費descriptor `+6`延遲而不再固定`(prog/6)`。同日以IDA直接consumer閉合`0x5255F／0x52577`六相位位移；目前只把已觀測第一個phase5接入E1剪影，並讓普通攻擊重用既有`0x18C6D` indexed bar／digit核心。完整序列最佳frame76由`AE=4436→1330→903→519`；519全部是舊三欄比較圖oracle的左邊／底邊合成邊框，排除後319×199內容區`AE=0／RMSE=0`，見[`battle-impact-compare-20260825.png`](../figures/battle-impact-compare-20260825.png)與[`battle-impact-no-global-tint.json`](../data/ui-traces/battle-impact-no-global-tint.json)。完整六相位因缺`0x29F72` raw owner仍只到`DATA-READY`，oracle亦非新的未修改DOSBox擷取，故不升E2。command29玩家多目標indexed owner已接。ID32／33／34／35正式command grid與state transaction亦已接，但`0x27FC9` indexed presentation仍缺，不能把綠色交易測試當成畫面完成。28／31無已證實正常取得來源；敵方command29、其他可達command/spell/item完整presentation、精確音效與palette/timing sequence、一般玩家同狀態DOSBox E2仍缺 |
| postbattle/campaign transition | 30–40% | graph與24個標準 postbattle handler均已接；第28戰現由正式`story_ch28`保留64-slot戰況，勝利後依raw control播放五句、同步隊伍並進`preparation_ch29`及全新`Game`冷讀，見[`ch27-post-native-dialogue-e1.json`](../data/ui-traces/ch27-post-native-dialogue-e1.json)。第29戰勝利→戰後→19人整備／冷讀檔→ch29_pre→第30戰亦已形成連續 E1。2026-08-26 再補齊 ch30 最後 focus 的 view／HUD、party→groups constructor 與完整 END→YES／敵方回合／勝利→終局介面生命週期；原版每章可見轉場仍未逐章 E2 驗收 |
| town hub | ch02 examined slice 85–90%；全章 partial | ch02 variant0 selection0–5 均取得原版 DOSBox E2，且各自能和 production remake 某個 pulse 的 320×200 raw RGB MD5 整幀相同；另有 variant1與variant2 selection0–4 的修改 LOAD E2，兩組五項與指定 pulse 的 640×400 整幀 AE=0。Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape返回selection5皆由原版 input trace 到達。23個town仍缺variant2 selection5 的 BIOS 掃描碼／Enter、未修改一般玩家路徑與其餘章E2；85–90%不可外推成全遊戲town覆蓋率 |
| weapon/item/secret shop | ch02 examined slice 92–95%；全章 partial | 69個shop節點已用variant1/3/5啟用indexed owner，但DOSBox E2只覆蓋ch02若干狀態。購買主選單／清單／Yes-No／不足金與收件者selection0↔1、賣出名冊／物品／Yes-No／成功／加款／返回已有多組整幀AE=0。service2名冊／面板與service3五個穩定子面板另有route-patched partial E2；service3物品清單的AE=2已定位為店員背景動畫兩點相位差，靜態清單內容在該狀態一致。其餘內容與幾何一致但動畫相位未同步。service2裝備現先在私有 unit 完成 raw transaction、重算及完整 panel 重建，最後才一次發布；刻意移除 palette 的深層失敗回歸證實 roster、能力、selection與既有 panel 全部不變，達`RUNTIME-E1`。service3現把Ebiten鍵盤轉成單一typed input consumer；回歸由正式商店四項選單Right×3進入service3，逐一完成opening／closing／restore的Draw acknowledgment，再驗證empty回覆與返回、full原子拒絕，以及目的取消後重入self-transfer。這些分支不再只靠直接寫mode或呼叫transaction helper，均達production-input `RUNTIME-E1`。六名synthetic typed party另由正式menu→purchase→Yes consumer進三列裝備收件者，驗證selection3時start0→1、horizontal no-op、滿欄／無合適角色零交易及正常裝備成功／扣款；每個opening／closing／restore與timeline都經Draw acknowledgment。測試未注入OS鍵盤，也未從完整campaign抵達商店，不能替代原版四人以上存檔或玩家E2。這些E2仍依screenshot-only LOADCH bootstrap；正常JOIN→LOADCH roster另有runtime regression。仍缺四人以上recipient scroll、no-recipient/full、service2動畫相位與原版mutation／restore畫面，以及service3 empty／full、self／destination-cancel與其他章節的**原版同狀態 E2**；這些分支的重製正式輸入 `RUNTIME-E1` 已完成。92–95%不可外推成69店全覆蓋率 |
| church | 60–70%（已接 slices） | church main/status/transfer/revive/class 多數已有原始 FDOTHER/FDICON/FDTXT indexed畫面與 lifecycle；正式 `church_ch02` 在 `selection=0,pulse=2,gold=1000` 的640×400擷取最近鄰縮回320×200後，與現行原始資源oracle達`AE=0/64000`。舊oracle的`AE=320`已勘誤為過時產物，不是runtime缺陷；此證據仍只是`RUNTIME-E1`，因oracle不是DOSBox原版畫面。transfer的`0x2f8ea`亦由shop service3共用，非church專屬；仍缺 DOSBox side-by-side與完整 persistent/save parity |
| preparation | 50–60% | 舊「兩欄文字核取方塊」與「確認框仍是重製殼層」斷言已失效。城鎮 FDTXT `0x201` 出發提示會保存／還原實際 town frame；無城鎮 FDTXT `0x19a` 記錄提示使用原版黑色來源，肯定結果在完整關框後進入四槽，Enter保存後保留列表，Escape才返回選人；見[有限保存收據](../data/ui-traces/preparation-record-save.json)。兩者與 `0x31d3c` 最終確認都接上 6＋4＋兩 tick 脈動＋4＋5＋還原。`0x318ad/0x31e80` 選人主畫面、`0x17fc0` 狀態與 `0x1297d` 待機週期亦已接正式路徑。2026-08-27再把分散鍵盤分支收斂為共用typed consumer；第23戰戰果後已從記錄提示肯定／存檔走到全新`Game`冷讀、提示否定、15人選取、最終取消重選及肯定進`story_ch24`，見[`ch23-post-preparation-ch24-input-e1.json`](../data/ui-traces/ch23-post-preparation-ch24-input-e1.json)。2026-08-26 外部 checksum-valid 晚期槽已由未修改原版普通 LOAD 走過 save-NO、19人選擇、最終確認與戰前劇情至可操作第30戰，形成候選E2；重製再由同一固定槽的正式LOAD入口擷取初始整備，修正第一組數字誤畫已選人數的缺陷。凍結合法相位0並分開FDOTHER #5 entries 31..40／42..51兩套數字字形後，320×200同狀態整幀AE=0/64000。第29戰勝利寫槽的完整來源鏈只保留為證據限制；50–60%不可外推到所有整備章節 |
| save/load | 60–70% | 四槽 input、native save envelope、原版 indexed loadslots 與 chapter-slot→typed party→town/preparation restore owner 已接；current battle 的巢狀 LOAD 也會先完成私有 typed handoff，YES 後才原子替換。SAVE 會保留未知 bytes，並對具證據的新我方 JOIN 同步 persistent raw。重製 JSON LOAD 亦先驗證 JOIN／membership／部署／materialized roster 拓撲；新的 `Game` 已能冷讀 `preparation_ch30` 後繼續第30戰與終局，錯誤拓撲則原子拒絕。2003年外部末關候選存檔通過原版 checksum，固定版原版由普通 `CONTINUE` 抵達第30戰；重製同檔也發布相同回合／鏡頭／游標、33筆場上單位與31人持續隊伍。另一外部 checksum-valid raw chapter `0x1c` 槽已由未修改原版普通 LOAD 連續走過 save-NO、19人整備、最終戰前劇情至可操作第30戰，見 [`native-load-ch29-slot-to-ch30-original-candidate.json`](../data/ui-traces/native-load-ch29-slot-to-ch30-original-candidate.json)。兩者均因第三方來源不完整只列候選E2；仍缺第29戰勝利當下由原版 writer 建槽的完整來源、delete／overwrite與同狀態重製差分。長程往返改由使用者人工回報問題，不列代理工作項目 |

> **晚期 slot 0 重製端補證：** 同一固定雜湊檔現經正式 LOAD selector 還原29人、
> 60金幣與四個已證實 metadata byte，並進 `preparation_ch30`。這關閉原版候選與
> 重製節點間的路由差分。後續正常輸入回歸又從標題 LOAD 事件、slot 0、連續19次
> 選取與最終確認抵達 `battle_ch30`；選取後游標自動前進，20名部署者直接來自已驗證
> persistent records，包含 authored ch30 scenario 漏列的 identity 3。逐像素整備比較、
> writer 來源及其他章節差分仍未閉合。同一固定槽又由正式 END／YES 進入未刪減的
> 第30戰敵軍回合；原始敵軍完成行動並交回 `PLAYER PHASE`。這是重製端
> `RUNTIME-E1`，不是新增原版逐幀或音訊 E2。
| ending | 60–70%（E1；來源約束終局） | prefix 已跑到 `0x2c548`；`sub_2C39B`的19×5框、caller initial portrait與FDTXT逐句speaker來源已閉合，timeline保存每句control／operand／pages。正式ending已用原版indexed owner逐Draw呈現六段opening、最多四列逐glyph、完整頁嘴型、Enter／Space分頁／換句、五段closing與source restore；不再使用一般RGBA框，完成收框後才resume timeline。chapter26兩個文字閘門與chapter29第一閘門五個blocks均有原始資產預建／生命週期回歸；缺資產整批零發布。正式 `battle_ch30→ending` 現以 persistent JOIN roster 的 raw `+6/+7/+8/+0x20` 同步最後隊伍並執行原資源角色蒙太奇；全新 `Game` 的冷讀連續回歸已由 `preparation_ch30` 經一般戰果接縫抵達此處，部署成員採最終戰更新，未部署成員保留冷讀狀態，回顧循環依完整 JOIN 時序涵蓋全隊。缺戰場、零筆身分符合、缺任一原始資產或 raw provenance 時不發布部分終局，而是整批回到可編輯結語。其後 `MontageTailPlayer` 預檢 20 組共80個 TAI／BG／FIGANI selector；實檔全部 header byte1=0，因此正式 E1 現逐 raw `+6` inner present、`+7 bit0` 層序、base scheduler、`+4` 位移與 palette33 override執行兩次交叉配對，第二次在最後 effect frame 結束；再依 caller 的20／78 ticks疊 FDOTHER #58，完成後保持 #59。[20 組總覽](../figures/ending-tail-20-segments-approximate-remake-e1.png)仍只是 E1，因為 `0x1088d(0x1e)` 的31-record baseline 穿過 `0x2c548` 的位元連續性、聲音 owner、原版輸入時序與一般玩家路徑尚未閉合。正式成功路徑現不再疊加來源等級／按鍵說明等現代提示；僅玩家主動開啟除錯HUD時可見。`Game.Update`與冷讀長鏈回歸現共用單一終局輸入owner，完成raw-change pending、定格進回顧與回顧返回；這只證明重製端正常輸入接線，不外推原版scan code。2026-08-28 新增普通標題LOAD抽樣：來源完整四名隊伍依序通過`0x2be44`多block文字、`0x2c548`／cue、Space raw-change 4／4與20筆tail admission，最長重播自然抵達segment5；[輸入與狀態證據](../data/ui-traces/ending-montage-tail-normal-input-remake-e1.json)只提升重製端E1，不冒稱本次已到永久終端。`0x2939D` 的3%外層預算在終局非零分支讀取未初始化區域值，不能當作穩定重播契約，已降為非阻擋原版考古限制。外部片尾錄影把 #59 對應為 `THE END` 是**強推論**，見 [`ch30-ending-youtube-visual-side-evidence.json`](../data/ui-traces/ch30-ending-youtube-visual-side-evidence.json)，不是一般玩家 E2。Enter／空白鍵開啟、Enter／空白鍵／Esc 關閉的角色回顧循環是重製版延伸；一般玩家 E2 仍未完成 |

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

> **2026-08-27 產品停止線勘誤：** 依使用者接受的 99% 玩家可見相似門檻，第29戰
> 正常 `CONTINUE`、戰場物化、`END→YES`、敵方回合與控制權交回，以及重製端
> 勝利後接往第30戰整備的垂直鏈，已足以列為產品完成。第三方存檔的完整 writer
> 來源、逐幀／逐音訊 `PLAYER-E2` 仍是證據限制或可選精修，不再列為第29戰重製缺陷，
> 也不得單獨觸發重新反組譯。

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
| UI-01 title/menu | 原版 `0x1fe2c` scan-code loop（↑/↓ wrap；Enter/Space/`0xe0`/`0x52` confirm）、`0x25ebb` return dispatcher、DOSBox oracle `docs/figures/title-original-dosbox.png` 已固定 START／LOAD／CONTINUE 與 title cursor；四槽 selector、valid-save typed restore 與 indexed 畫面已接。CONTINUE 的 FDFIELD 控制映像、battle-local event state、current-runtime-order selector rebuild，以及標題 caller 的 opening／interactive range mode、HUD gate B／anchor 已閉合成唯讀 preflight；`TitleMenuContinue`＋`loadNativeContinueFromCurrentSnapshot` 已提供 chapter0 的正式 `Game` controller handoff | partial（CONTINUE 為 `RUNTIME-E1`） | 刪除／覆寫、完整 boot 畫面差分，以及一般玩家同狀態 E2；不再把 CONTINUE production owner 列為缺口 |

| UI-02 field | map/camera/cursor/unit/HUD Draw 約 3441–3568、4571、4595；camera、absolute/visible cursor、HUD anchor/gates 與 FDOTHER #130 panel 已有直接原版資料流；ch26 event61 所需 view/HUD 已達 E1。ch27 event62 已接向左一步第七拍 selector0，能由完整 raw row 與 `0x2066E` 已證實的新戰鬥回合初始值1啟用 event63；`sub_1A813(0)` 的敵軍 AI 前 owner、兩次 0x35822 增援、delta255 全白／delta0 恢復及 AI continuation 已接正式 runner。ch26_pre 返回 battle_ch27 的 view `(camera 9,49; cursor 14,54; visible 5,5)`、selector0 與 inherited HUD 已由 IDA／Capstone 閉合並接線；gate A 從存檔、anchor 從程序持續狀態、gate B 從 controller 取得，不猜章節常數。event63 的 indexed regression 由雜湊綁定 `NativeJoinConstructorTable` 建立凱麗 fresh raw `+0x42=151`，不再手填 fixture，也不由章節近似 HP 反推。ch00 的 `0x32999` 已以 FDOTHER #9 接12次索引呈現、pass6/7/8 snapshot 重建及 pass1 #95，兩次各12幀後能沿正式 handler 進入戰鬥、戰後、城鎮與整備。另以截圖專用快速時鐘逐拍跑完 `story_ch00_handler` 的 73 拍並擷取 `native-map-ch01-remake-handler.png`；完整隊伍已出現；`native-map-ch01-original-video.png` 是不同狀態的歷史參考，不能以其單位、游標、HUD 或尺度差異判定目前 renderer 缺陷。2026-08-10 另以同一 `FD2.SAV` 狀態做 DOSBox／重製最近鄰比較，內容區只剩 22 個畫布邊界差異像素，記為 ch01 scoped E2 candidate；ch01 以外仍只是 E1。ch01 global event1/2 又驗證 turn4/5 各12次呈現後才執行 ACTING(3/4)，event2 對話不會越過 acting；缺 acting 資源時不發布 roster/cache/turn continuation。這些目前仍不是完整戰場 UI 或全章節 DOSBox E2 | partial | 除 ch26／ch27 event62/63／ch00／ch01 event1/2 E1 切片與 ch01 scoped E2 candidate 外，ch02+ 的逐章 dynamic view/gates/anchor producer、ch27 一般玩家／CONTINUE 同 roster/event/tick DOSBox 像素差分、該時點角色 raw record 的實際值，以及 `0x12c0d` 的 exact raw lookup predicate/order；另補 ch00 與 ch01 event1/2 同 camera/roster/pass 的原版逐幀比較 |
| UI-03 action menu | Docker Capstone `0x18890` + `0x18d8c`：↑0 attack、←1 spell、→2 item、↓3 wait/field interaction；native command grid每欄四列。共用空游標 `0x16F55` 中，direction2設定與direction3 END已有正式E1。direction0另由`0x19DF7`開nested cells `36、39/41、42/44、45`；四分派及`0x1B1E7`資訊畫面的12開／等待任意鍵／12關、entries `0x85..0x88`、兩行FDTXT、六個數值欄與`0x4DFCC` palette cycle現已由正式鍵盤路徑消費。nested selector1 SAVE 已以完整 CONTINUE raw baseline、FDTXT `0x19A/0x19B/0x19C`、checksum、XOR envelope與原子取代接入；只有 YES 寫入明確的可寫複本，NO 與所有來源矛盾保持原檔不變。nested selector3亦依完整`-1`回傳鏈正常離開。外層selector1現以第一筆runtime `+7` DATO肖像、FDTXT `0x1A1/0x1A2/0x19C`確認，依record順序播放全軍移動並收束回合；正式資料僅有的selector1 event61／75會在行走途中暫停，分別完成59幀／JOIN31與對話後turn chain再續行。動態count、raw或事件資產不完整時均在發布前失敗即關閉。battle command 0、13–27已有多個來源約束E1。`0x1A866` selector 1→0／2 已依原版順序接入 `+0x25` 扣 `floor(MaxHP/10)`、`sub_1DB65` 全記錄 HP 0 標記、存活者 raw 狀態倒數、歸零重算與 runtime record 同步；狀態致死沒有 killer，也不分派行動專屬死亡效果。`sub_17FC0` status colors／FDOTHER entries `0x37..0x39` 已由正式角色面板消費。 | partial（設定、END、nested資訊／SAVE／離場、含event61／75的全軍移動與狀態 phase 交易為重製E1；END另有原版窄E2） | 上述系統路徑與commands同狀態逐幀／逐音訊E2；未來若資料出現其他selector1 handler仍須獨立接線；status高階名稱、FDTXT `0x1E7` 扣血回覆、狀態 phase 死亡動畫、到期提示精確 tick／音訊／E2及其他effect presentation |
| UI-04 target/range | `0x1cff0` + `0x149f8` 證實 command record `+3/+4/+6` 參與 target-candidate geometry；`0x1bbdc` item case 0 的 two-stage targets、observed type5–24 effect dispatch 已閉合。item entry materialize `row[+0x12]+2`；first selector return後grid reset且selector回1。type23 destination把literal target code6傳給`0x115b6`，不是global selector6；兩層取消都回item panel。remake已接tracked transaction、occupancy/class/race/29×20 cost/terrain gate。2026-08-22 同一未修改 `FD2.SAV` 已由標題 CONTINUE 正常操作至悠妮 command 0：原版四相位擷取為窄 `PLAYER-E2`，重製普通 X11 路徑的sidecar明列 `native_command_targeting=true`／ID0／游標(10,15)，為 `RUNTIME-E1`；[四相位圖](../figures/native-command0-target-original-phase-sequence.png)、[原版／重製／差異圖](../figures/native-command0-target-original-vs-remake-e1.png)與[擷取紀錄](../data/ui-traces/native-command0-target-original-vs-remake-e1.json)已固定動態LUT與modal presentation。兩側時鐘相位未同步，不列逐像素一致。正式 Game confirm 現完整預建並消費 command 0 的九段角色滑入、施術者效果、28 幀／7 元素錯開目標效果、七段 HP 與 LUT 尾段，所有階段需 Draw acknowledgement 才發布 MP／HP／acted，列 indexed `RUNTIME-E1`；缺 raw provenance、FIGANI、BG／TAI、FDOTHER、palette 或 panel input 時維持原子不變。command 1亦已接正式玩家／敵方owner：每目標31張、八段HP、八個sample1 marker、九幀目標間過場與common tail均依Draw發布，列indexed `RUNTIME-E1`。command 6 亦以相同原子邊界接入玩家與敵方：common 前導／施術者、全目標 orbit、九幀目標間過場、每目標五段 HP 與尾段均需 Draw acknowledgement，列 indexed `RUNTIME-E1`；#87 單幀多呼叫混音仍只近似。指令6的指定12張target全幅與正常鍵盤施法已於2026-10-05限定驗收，詳見本頁最新節；三者完整演出逐幀／逐音訊E2仍未宣稱。物品第一階段 raw target field 與 selector 1..5 已由正式索引組合器消費；缺原生素材時不再顯示無證據的綠／青／橘半透明後備層。另以同一未修改存檔正常選到索爾草藥：四名我方皆滿 HP 時，Return 後 4.3 秒仍保留物品面板、未進 target modal，形成 type5 無可回復候選的窄 `PLAYER-E2`；[三狀態圖](../figures/native-item-herb-fullhp-original-e2.png)與[擷取紀錄](../data/ui-traces/native-item-herb-fullhp-original-e2.json)已固定 | partial | 受傷角色的 type5 目標 modal、native argument↔weapon min/max mapping、AOE/LOS、不可用目標灰化、command0／1／6 同狀態逐幀／逐音訊 E2、其餘 indexed command／item effect presentation、物品取消鍵原版語意、其他command同狀態比較；global selector6的production owner仍待 |

> **2026-08-28 UI-04 普通取消交易補證：** 同一章節0 current-runtime由正式標題
> CONTINUE與普通X11鍵盤完成兩條不同owner的取消：移動selection按Escape後清除
> selection且角色／回合不變；command0目標modal按Escape後回到同角色action overlay，
> 保留selection且不扣MP／不acted。證據見
> [`native-move-cancel-ch01-remake-e1.json`](../data/ui-traces/native-move-cancel-ch01-remake-e1.json)及
> [`native-command0-target-cancel-ch01-remake-e1.json`](../data/ui-traces/native-command0-target-cancel-ch01-remake-e1.json)。

> **2026-08-28 戰鬥普通輸入與AI旁車補證：** `FD2_SHOT_STATE.battle.units`現在依
> runtime array順序輸出純scalar index／camp／座標／HP／MP／acted／on-field；index
> 不冒稱角色身分。章節0普通END→YES且`FD2_SHOT_AI=1`的前後旁車證明七名場上敵軍
> 移動、待命敵軍不動、我方狀態不變、round只加一，完成後普通Return取得玩家控制。
> 證據見[`native-enemy-turn-units-ch01-remake-e1.json`](../data/ui-traces/native-enemy-turn-units-ch01-remake-e1.json)。
| UI-05 dialog | dialog Draw 約 3590–3686；`dlgAdvance` 有 page/scroll state；ch01 original oracle `docs/figures/ch01-dialogue-original-dosbox.png` 固定左肖像下框、文字、page indicator。2026-09-14 由 dosgolem normal START 重生王座廳第一句，鎖定原版 control boundary 12 與重製 frame 362：兩側鏡頭`(3,20)`、索爾焦點`(8,21)`及21筆角色FIG／座標一致；重製另固定`story_ch00_handler` beat4、`0x32382`、`FDTXT_033#0`。最近鄰正規化後全畫面63518／64000（99.246875%）相同，對白overlay 28160／28160及上／左／右邊界逐像素一致；差異482像素只在`y=21..111`人物sprite。原版兩個穩定boundary乘重製三個sprite相位的六張差異遮罩雜湊相同，排除錯配動畫時點。第二次Enter後原版正常BIOS讀鍵與重製production輸入owner皆移至國王焦點`(7,5)`、鏡頭`(3,4)`。見[`storybg-dialogue-original-vs-remake-e1.json`](../data/ui-traces/storybg-dialogue-original-vs-remake-e1.json)；2026-09-05 near-state收據只保留為已被本次同相位比較取代的歷史基準 | partial（王座廳第一句 lower／left 為同狀態 `RUNTIME-E1`） | 補其他說話者的 upper／right、控制碼、跨頁及完整戰役 storyBG 同狀態差分；上半部人物sprite仍有482差異像素。重製端使用決定性截圖鉤子，因此本切片不冒稱完整 PLAYER-E2 |
| UI-06 HUD | native map HUD `0x1acf3` 的 panel→terrain→AP→DP→optional unit icon→HP 已由 `BlitNativeMapHUD→ComposeNativeFrame` 接入 ch01 production full frame；display gates、persistent anchor、LMI1 #130／hex #0x83/#0x84、digit banks與FDICON selector均有 regression。`0x11cfa`證實HUD base是`work+0x8088`。FD2.SAV 初始快照為 camera `(1,13)`／absolute cursor `(8,17)`／visible `(7,4)`；原版錄影434.5秒的較晚比較幀則與remake對齊 `(1,13)`／`(8,15)`／`(7,2)`、tree icon及`A -05/D +10`。全 33 圖現具雜湊驗證後同步的 composition byte+3 與 terrain control；ch01／ch26／ch27 已改用 `native_map_hud_inherited`：gate A 由 custom save／native chapter restore 保存，anchor 在程序內持續，gate B 只接受已證實的 controller entry 1。event63 production regression 已由正式 JOIN table 提供 persistent raw record 並進 indexed path；#22仍只在 native admission 失敗時 fallback | partial | 除 ch26／ch27 E1 切片外，ch02+逐章動態 view/gates/anchor provenance、ch27 未修改一般玩家／CONTINUE 同一 roster/event state 的 pixel diff、`0x12c0d` exact raw lookup predicate/order；raw globals高階名稱仍不猜 |
| UI-07 postbattle | `campInput` battle result 約 2394；campaign node 可表達 post node；`campaign_full` 30 戰 transition matrix 已逐列展開。主迴圈直接指令、scenario `chNN→map(N-1)` 與 handler `chNN_post→set_chapter(N+1)` 共同證實玩家戰鬥 N 使用 raw `ch(N-1)_post`。13個既有同號錯接已全數清除；目前稽核為24 active／0 blocked。raw ch06→玩家ch07 已閉合 map6 六格 selector0 event26 的 raw `+6` gate、slots9..27 mode寫入與 state16 producer；enemy turn10 event25 只有在 state16==1 時才建立34→44 runtime、寫state17，戰後再經slot43 raw gate、唯一JOIN12 persistent record進 `town_ch08`。未踏格反例維持34 slots；先前「第10回合必定增援」與96-slot空白 frontier斷言均已撤回。raw ch07→玩家ch08 已撤回無 producer 的初始 groups1／8／9／10，正常入口為party10＋group0共29 slots；event27回合2..7逐組追加兩筆，戰後接受29..41奇數 frontier，依序執行layout、ACTING33／34、完整全黑、JOIN5、sync與chapter8，再進 `town_ch09`。raw ch09→玩家ch10 現保留60／61兩種強推論 frontier，依原始位址執行 DAC delta 0→63 淡出、sparse record/view patch、delta 64→0 淡入、FDTXT_010 index4／5、ACTING37、JOIN11／6、sync與chapter10，再進 `town_ch11`。raw ch15→玩家ch16 現正式接入76-slot persistent-first topology，四條 raw branch（round>18、inactive>4、word42<0x140、JOIN18 arm）均以 Docker/Xvfb E1 regression 進`town_ch17`；raw ch16→玩家ch17現以 map16的兩條 roster_has(18) branch接入layout、ACTING50–53、FDTXT_017 index5–8、JOIN16，60／61→61／62 frontier並進`town_ch18`，另驗證save/load；raw ch17→玩家ch18 現以 map17的55-slot runtime接入layout、ACTING56／57／58、FDTXT_018 index7–10、JOIN21／7，進`town_ch19`並驗證save/load；raw ch19→玩家ch20現以固定record0＋選15人和map19 group0建立83-slot入口，round15執行group1→84與JOIN28，round16精確略過，兩路共同JOIN25後進`town_ch21`。raw ch23→玩家ch24現以70筆FDFIELD＋16筆LOADCH建立86-slot runtime，依stage-before-draw與BIOS tick gate執行240＋60次indexed draw，完成隊伍同步後進`preparation_ch25`並驗證save/load；raw ch24→玩家ch25現以party16＋group0(46)=62開場，event56追加group1(8)成70，戰後追加group2成71供ACT75操作slot70，JOIN26／29後進`town_ch26`並驗證save/load。raw ch12→玩家ch13由table bytes固定interior entry `0x2389f`並接`town_ch14`。raw ch05／ch25／ch27則分別屬玩家ch06／ch26／ch28；第27戰天空之鑰成功分支不重用raw ch27。玩家第22至25戰已提升為 E1；玩家第29戰現已完成 raw ch28 post、隊伍同步、preparation_ch30 與存讀檔 E1；所有已接切片仍缺未修改一般玩家 DOSBox E2，因此不宣稱完整一致。位址證據見[`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)、[`fd2_ch15_post_ida.txt`](../data/fd2_ch15_post_ida.txt)、[`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)、[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)、[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)及各切片證據檔。 | partial | 以原版 handler offset／DOSBox input 差分核對每章是否進 town/shop/rest/preparation/ending；玩家第22至25戰與第29戰均已提升為 E1，但仍缺一般玩家 DOSBox E2；第7／8／10／16／17／18／20／22／24／25戰同樣尚缺一般玩家 DOSBox E2；ch00 `0x3241f` 尚缺 raw FDICON key，仍是明示的 RGBA E1 近似 |
> **2026-08-27 第6戰戰後原生對話補證：** raw ch05 post 的唯一caller現由
> FDTXT_006 index6建立19句原生版面。正式輸入依序完成JOIN13、group3、PAN、
> ACTING27、19句、sync、40→41 slots、`town_ch07`與存讀檔，列`RUNTIME-E1`。
> 未修改DOSBox同狀態與精確音訊仍缺，但依99%門檻不阻擋。
>
> **2026-08-27 第22戰戰後原生對話補證：** raw ch21 post 的三個caller現由
> FDTXT_022 index4／5／6建立3＋1＋7句原生版面。73／79-slot兩條正式節點入口
> 均保存戰場視圖，以具型別輸入完成11句後才執行indexed transition、同步、
> `preparation_ch23`與存讀檔，列`RUNTIME-E1`。未修改DOSBox同狀態與精確音訊
> 仍缺，但依99%門檻不阻擋。
>
> **2026-08-27 第24戰戰後原生對話補證：** raw ch23 post 的兩個caller現由
> FDTXT_024 index2／3建立3＋8句原生版面；index2保存scene0→scene1的有序跨場景
> 映射。86-slot正式戰果路徑以具型別輸入完成11句、stage2..14、同步、
> `preparation_ch25`與存讀檔，列`RUNTIME-E1`。未修改DOSBox同狀態、精確BIOS tick
> 與音訊仍缺，但依99%門檻不阻擋。
>
> **2026-08-27 第10戰戰後原生對話補證：** raw ch09 post 的兩個caller現由
> FDTXT_010 index4／5建立19＋16句原生版面。60／61兩個正式輸入路徑都依序完成
> DAC淡出、sparse patch、淡入、35句、ACTING37、JOIN11／6、`town_ch11`與存讀檔，
> 列`RUNTIME-E1`。未修改DOSBox同狀態與精確音訊仍缺，但依99%門檻不阻擋。
>
> **2026-08-27 第17戰戰後原生對話補證：** raw ch16 post 的四個caller現由
> FDTXT_017 index5／7／6／8建立4＋3＋1＋18句原生版面。roster有／無角色18
> 兩條正式輸入路徑分別完成23／22句、60→61／61→62 frontier、JOIN16、
> `town_ch18`與存讀檔，列`RUNTIME-E1`。未修改DOSBox同狀態與精確音訊仍缺，
> 但依99%玩家可見門檻不阻擋。
>
> **2026-08-27 第16戰戰後原生對話補證：** raw ch15 post 的三個caller現由
> FDTXT_016 index2／3／4建立3＋5＋15句原生版面。round>18、inactive>4、
> word42<`0x140`與word42>=`0x140`四條正式輸入路徑分別完成8／8／0／15句、
> JOIN18正反例、`town_ch17`與存讀檔，列`RUNTIME-E1`。未修改DOSBox同狀態與
> 精確音訊仍缺，但依99%玩家可見門檻不阻擋。
>
> **2026-08-27 第7戰戰後原生對話補證：** raw ch06 post 的`0x2337F`／index4
> 與`0x233B2`／index5現由固定raw控制碼建立8＋4句原生版面；同一`0x233B2`
> 出現在互斥分支時只計一個caller。active／inactive兩條正式輸入路徑分別完成
> 8／4句、JOIN12正反例、`town_ch08`與存讀檔，列`RUNTIME-E1`。未修改DOSBox
> 同狀態與精確音訊仍缺，但依99%玩家可見門檻不阻擋。
>
> **2026-08-27 第18戰戰後原生對話補證：** raw ch17 post 的四個 caller 現以
> `FDTXT_018` index7／8／9／10直接建立5／2／2／12句原生版面；index10保持
> scene3→scene4的有序分段。正式具型別輸入播放全部21句並完成opening、逐字、嘴型、
> closing後，才JOIN21／7、進`town_ch19`及驗證存讀檔。原始控制碼相等與正式路徑
> Docker／Xvfb回歸均通過，列`RUNTIME-E1`；未修改DOSBox同狀態E2仍缺，但依99%
> 玩家可見門檻不再阻擋，也不得因E2缺口重開已閉合的`0x23cd5`。
> **2026-08-27 第13戰戰後原生對話補證：** raw ch12 post 的唯一 caller
> `0x238c8`現以`FDTXT_013` index9建立12句原生版面，保持scene3／4兩段順序。
> 正式具型別輸入完整播放後，才同步15人、JOIN3、進`town_ch14`及驗證存讀檔；
> raw equality與Docker／Xvfb回歸通過，列`RUNTIME-E1`。未修改DOSBox同狀態E2
> 仍缺，但依99%玩家可見門檻不再阻擋，也不得重開`0x2389f` dispatch。
> **2026-08-27 第20戰戰後原生對話補證：** raw ch19 post 六個caller的29句已
> 由`FDTXT_020`固定raw建立原生版面。正式round15路徑播放全部29句、83→84 slots、
> JOIN25／28；round16播放共同15句、維持83 slots且只JOIN25。兩路都以具型別輸入
> 完成收框後進`town_ch21`並通過存讀檔，列`RUNTIME-E1`。未修改DOSBox同狀態E2
> 仍缺，但依99%門檻不再阻擋，也不得重開`sub_23E74`。
> **2026-08-27 第8戰戰後原生對話補證：** raw ch07 post兩個caller／8句已由
> FDTXT_008固定raw建立原生版面。正式29與41 slots邊界均以具型別輸入播放完整8句，
> 再執行ACTING、全黑、JOIN5、同步、`town_ch09`與存讀檔，列`RUNTIME-E1`。
> 31至39由相同typed frontier契約接受；未修改DOSBox E2仍缺但不阻擋99%門檻。
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
> 存檔／讀檔。2026-08-27 又依 `0x2415B` 三張25-byte表接入26-slot layout與
> camera `(336,240)`，並在三段FDTXT之間完整消費ACT63／64。這解除的是先前
> 「鑄造動畫及相鄰擺位／演出未接」的缺口，不包含第一個程序內相位或未修改原版同狀態 E2，故 UI-07
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
| UI-10 church | `0x2d7bd` 左右四項循環；`0x3072f` dispatch `0→0x2ffa5` status、`1→0x2f8ea` item transfer、`2→0x30dc3` revive、`3→0x31385` class。class path已接 exact list/confirmation lifecycle；正式 `applyChurchClassChange→leaveChurch→town→JSON save/load` 整合回歸現保存 portrait/class/raw class、selector、成長、裝備重算、背包、隊伍順序與節點，並清除讀檔前教會暫態。raw0已接兩欄 roster與完整唯讀`0x17aed` status/items→command/MP lifecycle。raw1與shop service3共用`0x2f8ea`：FDTXT510/511/512、source/item/destination roster、`0x2dc55(mode1)`、FDTXT506滿欄與raw remove→append/recalc均已接；destination roster保留source本人，self-transfer依原指令做unequipped尾端重排。shop caller已有五個route-patched partial E2畫面，但不可外推church caller。revive已接 raw byte5 bit0候選、raw class×level費用、三列名單與完整feedback；成功animation/BGM lifecycle亦已接。2026-08-27正式`church_ch02`主選單固定相位與現行oracle達`AE=0/64000`，狀態旁證見[`native-church-menu-ch02-remake-e1.json`](../data/ui-traces/native-church-menu-ch02-remake-e1.json)。同日再由正式標題LOAD與普通X11鍵盤完成`town_ch02→church_ch02→town_ch02`，入口／返回旁車見[`native-town-church-roundtrip-ch02-remake-e1.json`](../data/ui-traces/native-town-church-roundtrip-ch02-remake-e1.json)；起點為合法重製節點存檔，故只提升正常重製輸入E1，不提升原版長程E2。 | partial/fail-closed | 原版FD2.SAV與church caller DOSBox E2 visual diff；不重做已閉合的四項production-input owner |
| UI-11 preparation | `0x2d0d1` 城鎮出發提示使用 FDTXT `0x201`／`(95,119)` 與原 town source；`0x2cc04..0x2cc87` 無城鎮提示先清 VGA、使用 FDTXT `0x19a`／`(100,119)`，肯定才在關框後呼叫存檔。`0x318ad` 清除30旗標；`0x31a7c..0x31b08` 左右±1、上下±10；`0x31e80` 接三區背景、10欄角色格、游標、彩色／灰色角色及 `0x17fc0` 狀態。`0x31ea9..0x31ec6` 第一組數字直接取quota，`0x31edb..0x31efb` 第二組才取quota減已選，舊重製初始`00／19`已修成原版`19／19`。`0x320fc`直接證實record0固定、旗標i對應record i+1；重製已修正為固定1人＋可選15／19人，總上場16／20。`0x1297d` 待機週期與 `0x31d3c` 最終確認的完整 Draw 確認生命週期均已接；原始圖像索引、記錄或資源缺值即退回。同一固定晚期槽的初始狀態與合法相位0現有AE=0/64000成果圖。2026-08-28再由正式標題LOAD普通輸入完成早期`town_ch02→preparation_ch02`前置確認及Escape取消返回；三態旁車與限制見[`native-town-preparation-cancel-ch02-remake-e1.json`](../data/ui-traces/native-town-preparation-cancel-ch02-remake-e1.json)。空名冊起點不提升選人或出戰完成。 | partial（E1） | 依99%忠實度門檻關閉這個固定初始狀態；只抽測其他章節與交互狀態；`0x1f42d` 已更正為戰場進入演出，不屬此選人視窗 |
| UI-12 save/load | F5/F9 是重製自有快捷路徑，不得外推為原版戰場存檔；save package 自有 schema。原版 `FD2.SAV` 的 `0x59cb` boundary、rolling-XOR/u32 byte-sum checksum、4×logical `0xa28` records at `+0x312b`（metadata `0x28` + roster `0xa00`）已由真實 sandbox decode、`tools/fd2save.py` 與 `internal/fdsave` regression 覆蓋。合法 IDA 9.4 已固定 reader `0x2602c..0x26098` 與 writer `0x30012`：兩者只處理 metadata `+0..+9`；writer 只由 `0x2cad7` 直接整備與酒店呼叫。production 以雜湊綁定的 `0x526b9` gate table 把 raw chapter 1..29 還原到 `town_ch02..27` 或 `preparation_ch23..30`，先完整驗證 persistent record→typed party、節點型別與重複 identity，再原子套用 campaign cursor、gold、party 與 raw metadata 保存值；ch21/ch27 postbattle inventory gate 不會重播。標題 Enter／Space 現只經正式確認 owner：checksum-valid 合成槽完整還原 `town_ch02`、悠妮 typed/raw record、join order、789金幣、chapter1、HUD gate並清除舊 battle state／selection；竄改 envelope 留在 `loadslots`，campaign／party／gold／battle state 零修改且不落入 JSON loader。空槽及修改存檔chapter1有效槽畫面均與DOSBox全幀相同；未修改CONTINUE戰場按F5後存檔雜湊不變，確認一般玩家有效槽必須先從酒店／整備建立 | partial（空槽 E2；有效槽排版與 restore 為修改／合成路徑 E1，不升為一般玩家 E2） | 正常完成戰鬥後由酒店／整備建立槽位，再走標題LOAD的successful native-load E2；metadata `+10..+39` 其他可能 consumer、CONTINUE current-battle owner、delete/overwrite |

> **2026-08-28 UI-09 早期道具店普通輸入補證：** 合法重製節點存檔由正式標題
> LOAD與普通X11鍵盤進入`town_ch02`，原生selector選項3進`shop_ch02_item`，再以
> Escape沿正式商店返回邊界回到城鎮；節點、gold與transient旁車見
> [`native-town-item-shop-roundtrip-ch02-remake-e1.json`](../data/ui-traces/native-town-item-shop-roundtrip-ch02-remake-e1.json)。
> 此證據只提升重製正常輸入入口／返回E1，不取代購買／出售交易或原版同狀態E2。

> **2026-08-28 UI-11 中期整備入口補證：** 合法`town_ch17`節點存檔由正式標題
> LOAD與普通X11鍵盤進native variant1城鎮，再以選項2抵達`preparation_ch17`
> 前置確認；limit15與三態旁車見
> [`native-mid-preparation-entry-ch17-remake-e1.json`](../data/ui-traces/native-mid-preparation-entry-ch17-remake-e1.json)。
> 取消返回重播未通過標題起點而排除，故只列中期入口E1，不外推選人或往返。

> **2026-08-28 UI-11 晚期整備往返補證：** 合法`town_ch26`節點存檔由正式標題
> LOAD與普通X11鍵盤進晚期城鎮，再以選項2抵達`preparation_ch26`前置確認；第二次
> 相同前綴送Escape後返回`town_ch26`。入口固定town-backed、limit15，返回保留
> gold12000且無暫態；證據見
> [`native-late-preparation-cancel-ch26-remake-e1.json`](../data/ui-traces/native-late-preparation-cancel-ch26-remake-e1.json)。
> 空名冊與合法節點起點使本格只達重製E1，不證明第25戰長程來源、選人或進戰。

> **2026-08-28 UI-10 晚期教會往返補證：** 合法`town_ch27`節點存檔由正式標題
> LOAD與普通X11鍵盤進城鎮，以選項4抵達`church_ch27`穩定menu，再以Escape返回
> `town_ch27`；selection0、gold13000及暫態收束見
> [`native-late-church-roundtrip-ch27-remake-e1.json`](../data/ui-traces/native-late-church-roundtrip-ch27-remake-e1.json)。
> 本格只達重製E1，不證明第26戰長程來源、四項服務交易或原版同狀態。

> **2026-08-28 UI-09 第26章神秘商店普通組合鍵補證：** 正式標題LOAD進
> `town_ch26`後，普通Left×4抵達選項4；Shift+F5建立原版scan0x58並揭露選項5，
> 另按Return進`shop_ch26_secret`，Escape返回仍顯示`???`入口。證據見
> [`native-secret-shop-roundtrip-ch26-remake-e1.json`](../data/ui-traces/native-secret-shop-roundtrip-ch26-remake-e1.json)。
> 這是章節特定正常玩家入口，不是debug shortcut；只達重製E1，不外推其他章節、
> 商品交易、長程來源或原版同狀態。

> **2026-08-28 UI-09 第27章神秘商店普通組合鍵補證：** 正式標題LOAD進
> `town_ch27`後，保持酒店選項0並送Ctrl+F6；原版章節表的scan0x63揭露選項5，
> 另按Return進`shop_ch27_secret`，Escape返回仍顯示`???`入口。證據見
> [`native-secret-shop-roundtrip-ch27-remake-e1.json`](../data/ui-traces/native-secret-shop-roundtrip-ch27-remake-e1.json)。
> 它與第26章的selection4／Shift+F5／0x58是不同章節風險；只達重製E1。

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

2026-08-31 四語勘誤：正式四列 grid 已不再以玩家另供 `command_labels.json` 作語系 admission；
官方四語 `CommandName(raw command ID)` 目錄恰含35個非空slot，ID31維持原版空槽。一般玩家
raw mask、有分離 palette及語系目錄時直接進原生 grid；35筆均通過同一欄 MP 起點前的字型寬度
測試。`command_labels.json`仍是繁中原始證據與相容載入來源，不再是多語 runtime 的唯一名稱層。
這提升四語名稱到 `RUNTIME-E1`，不提升 command effect或一般玩家截圖到E2。

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
  DOSBox 畫面與精確音訊，不是重製端 menu／callee 接線。2026-08-31畫面稽核另發現
  教會24×20服務格目前呈現亂碼紋理；它是renderer／codec視覺缺陷，不能因輸入callee已接
  就列為介面完成，也不能用四語文字猜蓋。
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

> **2026-09-11 範圍再更正：** MAP/TURN/ENEMY/FRIEND/NPC 資訊畫面與「決定要行軍嗎?」YES/NO
> 不屬於 battle-entry。前者只能經空游標系統選單 `0x16F55` → 巢狀選單 `0x19DF7` → `0x19F03`
> 進 `0x1B1E7`，後者是 `0x16F55` 外層 selector 1（全軍行軍）的 FDTXT `0x1A1`；原版開局不會
> 自動顯示（[doc46 §5.4](46-ch1-opening-timeline.md)、[doc109 §2](109-title-to-town-journey-20260911.md)）。
> 兩者由既有的系統選單條目承接，D8 只剩 `0x1a30b` 的 indexed choreography，上一段列的欄位
> 不再是 D8 的缺口。

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

### 2026-08-30：第一關推廣片選單錨點勘誤（RUNTIME-E1）

舊 v0.1.0 實機錄影揭露非原生畫面、空游標開啟系統選單時，
`actionOverlayAnchor(nil)` 會退回 `(0,0)`，使四格選單固定出現在左上角。依既有
`0x1741c` 游標來源契約，後備畫面現改用目前 `g.curX/g.curY`；選取單位路徑仍用
單位格，原生畫面仍由 `NativeMapViewState` 消費。Docker 回歸與重新錄影確認選單
出現在目前游標附近。證據、規格與限制見
[`ch01-promo-menu-correction-20260830.json`](../data/ui-traces/ch01-promo-menu-correction-20260830.json)
及 [`promo-ch01-dosbox-compare-spec.md`](../../docs/promo-ch01-dosbox-compare-spec.md)。

同批推廣片加入 DOSBox 原版與重製版第一關動態並排；兩側不是同一動畫 tick，故
明確標成「相近狀態比較」，不提升為逐幀或逐像素 E2。現有影片模板仍支持第一關
四名角色欄序 `7,8,10,11`，因此本批沒有因舊錄影觀感猜測性改寫部署格。

### 2026-08-30：共用選單錨點與第一關四人移動（RUNTIME-E1）

錨點修正不是第一關特例：同一 `actionOverlayAnchor` 以 map0、map25、map28 三組
不同 camera／cursor 狀態回歸，均依目前可見游標計算，不讀章節。第一關正常
`ch00_pre` 則由 `0x3281D LOADCH→0x32830 PAN→0x3283A ACTING(0)` 與
`map32.json` resource 0 固定 slots `0,1,2,3` 同步向上六格；正式測試逐格驗證
`(7,20),(10,21),(8,22),(11,23)` 到 `(7,14),(10,15),(8,16),(11,17)`，並走完整
handler 進第一戰。七拍格線動作另由約 18.2065 Hz 硬體規格近似投影到 60 Hz，
42 個來源拍需 139 次更新，不再誤壓成 42 次更新。

原版影片已定位 332–338 秒的相近狀態；本輪重製正常路徑錄影的自動按鍵未可靠
命中同段，因此動態並排仍為 READY，不提升成 E2。證據與可重播驗收見
[`action-overlay-and-ch01-march-20260830.json`](../data/ui-traces/action-overlay-and-ch01-march-20260830.json)。

# 2026-09-14 早中晚戰間收據入口

早／中／晚三份 dosgolem 原版收據、交易／存檔邊界、設施節點與祕密商店的聚合
資料見
[`town-shop-early-mid-late-e2.json`](../data/ui-traces/town-shop-early-mid-late-e2.json)，
2026-09-15 在 `fd2-go-test-local` 重跑 `TestTownShopEarlyMidLateE2Receipt` 三章全過，
六張原版 town／secret 幀逐 index 相符，狀態 `passed`。本輪 IDA 9.4 勘誤也確認
城鎮游標取 selector-cache 首槽，不是固定 FDICON archive group 0。外部存檔的
建立歷程與 dosgolem 未推送分支仍是收據內的證據限制。

第四章整章收據 [`parity-ch04.json`](../data/ui-traces/parity-ch04.json)（2026-09-16，
r58，原版側 r13、dosgolem `caa9ee8`；2026-09-17 以第七章之後的重製端重跑 remake-r14，差異
像素數逐點相同）：狀態 `passed`，55 個畫面比較點全部在 640 px 預算內、38 點逐像素相同，17 點的差
（66–201 px）全是指令環畫面殘差那一類（56 §五章回顧、#34）——
城鎮、出發、戰鬥開始，五個回合全部的 select／move／attack_armed／attack_result／stay／
wait、換手橫幅（狀態小窗、指令環、攻擊射程染色、橫幅都在 indexed composer 內，#28
關閉）、敵方回合的鏡頭（游標協定，#30 關閉）、酒店四圖示／存檔槽列表／「記錄儲存
完畢」（`native_hotel_ui.go`，#27 關閉）；`after_enemy_phase` 點只比行為。這一輪修掉
的兩個假差異：原版側 checkpoint 落在整幀 memcpy 中間的撕裂（dosgolem 延後 PNG，#31），
與重播端 idle 相位被 `0x1297D` 推進而少一張變體。抽樣對照表分兩張
[`parity-ch04-samples-p1.png`](../figures/parity-ch04-samples-p1.png)、
[`p2`](../figures/parity-ch04-samples-p2.png)（27 列：每種 kind 第一點加
全部 `diff_pixels>0` 的點，三格「原版｜重製｜差異遮罩」，每張 ≤1.5 MB，index
[`parity-ch04-samples.json`](../data/ui-traces/parity-ch04-samples.json)；`tools/parity_sample_sheet.py`）。

第五章整章收據 [`parity-ch05.json`](../data/ui-traces/parity-ch05.json)（2026-09-16，
remake-r5，原版側 r5、dosgolem `caa9ee8`；2026-09-17 重跑 remake-r10，差異像素數逐點相同）：狀態
`passed`，58 個畫面比較點全部在預算內、40 點逐像素相同，18 點的差（60–628 px）是指令環畫面
殘差與 YES pulse 兩類（#34、#35）——
城鎮、出發、戰鬥開始、第 3 回合八個單位移動待機、第 4 回合友軍 group 2 登場（鏡頭、演出、
四句對白，回合事件處理器整筆轉寫，#33）與兩次攻擊、第 5 回合清場、戰後對白、城鎮出售、
五棟建築、酒店存檔（整檔 sha256 相同：`+3` 朝向與 `+2` FDICON 快取槽照原版抄）、祕密商店；
`after_enemy_phase` 與落在 `0x1A30B` 換手處理裡的兩個 wait 只比行為。細節見
[56 §第五章章工作單元](56-fd2-remake-sdd.md#第五章章工作單元回合事件處理器與敵方回合順序2026-09-16)。
抽樣對照表 [`parity-ch05-samples.png`](../figures/parity-ch05-samples.png)（28 列，index
[`parity-ch05-samples.json`](../data/ui-traces/parity-ch05-samples.json)）。

第六章整章收據 [`parity-ch06.json`](../data/ui-traces/parity-ch06.json)（2026-09-16，
remake-r11，原版側 r4、dosgolem `50a3b47`；2026-09-17 重跑 remake-r10，seq 619 attack_armed
由 0 變 8 px：同一個 DAC 循環色相位差一步的形狀，重製端的循環相位每次重跑不一定相同，#38）：狀態
`passed`，58 個畫面比較點全部在預算內、39 點逐像素相同，19 點的差（8–618 px）分屬指令環畫面
殘差、YES pulse、DATO 嘴型、DAC 循環色四類（56 §五章回顧、#34／#35／#38）——
普里茲港城鎮、出發、戰鬥開始、第 1～9 回合守位（移動、待機、三次攻擊）、第 5 回合 event 20
對白（selector 2 回合事件）、第 10～11 回合接戰、清場、戰後對白與 group 3 登場、往王城的
途中出售、五棟建築、酒店存檔（整檔 sha256 相同）、酒店前 Alt+F6 祕密商店；`after_enemy_phase`
與落在 `0x1A30B` 換手處理裡的 `ai_order`／wait 只比行為，`ai_order` 分岔 0。這一輪修掉的
行為差異：敵方單位的暫時狀態 +0x22..+0x27 在正常進戰場時沒有遞減（法師不再放強化、MP 剩
太多）、指令 0／4 的演出擲骰沒有接在同一條 `0x4E893` 序列上（傷害 36 對 39）；畫面差異：
佈陣格跟槽位、HUD 小窗只在真的畫時前進、待機前先重播游標鍵。細節見
[56 §第六章章工作單元](56-fd2-remake-sdd.md#第六章章工作單元selector-2-回合事件記錄-8-守衛與戰後-spawn2026-09-16)。
抽樣對照表 [`parity-ch06-samples.png`](../figures/parity-ch06-samples.png)（29 列，index
[`parity-ch06-samples.json`](../data/ui-traces/parity-ch06-samples.json)）。

第七章整章收據 [`parity-ch07.json`](../data/ui-traces/parity-ch07.json)（2026-09-17，
remake-r8，原版側 sample-r6、dosgolem `f57c23d`；槽由 `fd2_chapter_slot.py` 從 ch02-cleared 建到
「已通關第 6 章」、9 人）：狀態 `passed`，241 個原版動作、228 個畫面比較點全部在預算內、164 點
逐像素相同，64 點的差（52–255 px）是指令環畫面殘差 63 點（#34）與祕密商店店主眨眼 1 點（#38）——
往王城的途中城鎮、出發、戰鬥開始（HUD 小窗 anchor 依初始可見游標翻到右側）、第 1～10 回合
全隊北上接戰（63 次選取、43 次移動、24 次攻擊：掉落訊息、經驗與升級對話、指令 4／13 的敵方
施放）、第 7～10 回合 step_into 踏事件格 26（記錄 9..27 AI 模式清 0）、第 10 回合 event 25
（group 2 登場、鏡頭、ACTING 30、四句對白，`after_event25` 356→0 px：0x134E4 全員姿勢歸零）、
清場、戰後 JOIN12 凱麗、出售、五棟建築、酒店存檔（整檔 sha256 相同：JOIN 記錄沒寫到的 byte
是 LOAD 時槽的殘值）、武器店前 Shift+F7 祕密商店；`after_enemy_phase` 與 `ai_order` 只比行為，
`ai_order` 分岔 0，金幣 2000→5500（seq 1304 掉落 3500：關框後才加）→5537。這一輪修掉的行為
差異：mode 0 敵人 0x14121 落點在原格時沒改走 0x13E9C（第 7 回合記錄 23）；畫面差異：掉落訊息
底圖攻擊者變灰與面向目標、升級對話底圖的 idle 相位、升級後 HUD HP 的綠字（+0x42 沒同步）、
ACTING 之後凱麗的朝向、暫時狀態掃描換掉 Unit 指標；收尾順序照 `0x11985`：`0x13565` 自動換手在
`0x1198A` 格子事件分派之前。細節見
[56 §第七章章工作單元](56-fd2-remake-sdd.md#第七章章工作單元格子事件分派掉落訊息與-step_into2026-09-16)。
抽樣對照表分五張 [`parity-ch07-samples-p1.png`](../figures/parity-ch07-samples-p1.png)、
[`p2`](../figures/parity-ch07-samples-p2.png)、[`p3`](../figures/parity-ch07-samples-p3.png)、
[`p4`](../figures/parity-ch07-samples-p4.png)、[`p5`](../figures/parity-ch07-samples-p5.png)
（81 列：每種 kind 第一點、全部 `diff_pixels>0`、指定的 1926／1955／1973／2011／2209，每張
≤1.5 MB，index [`parity-ch07-samples.json`](../data/ui-traces/parity-ch07-samples.json) 記每列
在哪一張）。城鎮進戰場的淡出另有輔助基準
[`parity-ch07-town-fade-probe.png`](../figures/parity-ch07-town-fade-probe.png)（非 gate：出口 YES
之後 `0x2D1CB..0x2D275` 十步縮放＋DAC 暗化到全黑，再由 `0x1F544` 淡入戰場；index
[`parity-ch07-town-fade-probe.json`](../data/ui-traces/parity-ch07-town-fade-probe.json)；
此為歷史輔助基準；現行正式接線與原版收據見下方#39勘誤）。

第八章整章收據 [`parity-ch08.json`](../data/ui-traces/parity-ch08.json)（2026-09-17，
remake-c3e，原版側 sample-c3、dosgolem `f57c23d`；槽由 `fd2_chapter_slot.py` 從 ch02-cleared 建到
「已通關第 7 章」、10 人，`--seed 4 --event-states 7:17=1`，再依 114 強化政策加基底 AP+200／DP+0／
DX+60）：狀態 `passed`，197 個原版動作、179 個畫面比較點全部在預算內、127 點逐像素相同，52 點的差是
指令環畫面殘差 50 點（41 move＋9 stay，最大 629 px，#34）、出口 YES pulse 1 點（60 px，#35）與
攻擊結果 1 點（4 px）——王城前的戰鬥城鎮、出發、戰鬥開始、
第 1～6 回合北上接戰（event 27 登場、敵方施法與回復）、第 6 回合擊倒騎士 slot 10（死亡程式 29 對白
seq 1452 逐像素相同）、第 7～15 回合停手（第 15 回合 event 28）、第 16 回合開頭清場、戰後 JOIN5 洛娜、
騎士的抉擇城鎮出售與買入（金幣 4800→4837→4827）、教會與酒店探訪、酒店存檔（整檔 sha256 相同）、
祕密商店；`after_enemy_phase` 與 `ai_order` 只比行為、`ai_order` 分岔 0。強化收據修掉的畫面差異：
死亡程式對白底圖沒重繪（HUD 小窗仍是攻擊前、屍體還在）、整幀重繪時攻擊者被提早畫灰、對白「．．」
畫成一點的字模 347（原版 584 兩點）、敵方攻擊前多聚焦自己一次與重播端 END 前沒重走推游標的方向鍵
（兩者都讓鏡頭差一列）。細節見
[56 §114 強化槽與第八章強化收據](56-fd2-remake-sdd.md#114-強化槽與第八章強化收據2026-09-17)。
抽樣對照表分三張 [`parity-ch08-samples-p1.png`](../figures/parity-ch08-samples-p1.png)、
[`p2`](../figures/parity-ch08-samples-p2.png)、[`p3`](../figures/parity-ch08-samples-p3.png)（68 列：每種
kind 第一點、全部 `diff_pixels>0`、指定的 1452／1799／1929／2187，每張 ≤1.5 MB，index
[`parity-ch08-samples.json`](../data/ui-traces/parity-ch08-samples.json)）。
限制：強化值不是原版證據，收據不談傷害、命中、存活與敵方選目標；洛娜（FDFIELD 記錄，不吃強化）
第 2 回合、凱麗第 9 回合陣亡，使用者同意接受。未強化槽的舊收據改名保留為
[`parity-ch08-unboosted-r1.json`](../data/ui-traces/parity-ch08-unboosted-r1.json)（第 10 回合提早清場）。

第九章整章收據 [`parity-ch09.json`](../data/ui-traces/parity-ch09.json)（2026-09-17，remake-r2，原版側
sample-r1、dosgolem `f57c23d`；槽由 `fd2_chapter_slot.py` 建到「已通關第 8 章」、11 人，強化 AP+200／DP+0／
DX+60）：狀態 `passed`，298 個原版動作、286 個畫面比較點全部在預算內、203 點逐像素相同，83 點的差是指令環
開啟中途 81 點（64 move＋17 stay，最大 627 px，#34）、出口 YES pulse 1 點（60 px，#35）與事件 30 對白底圖
1 點（seq 924，地圖上方單位 101 px）——騎士的抉擇城鎮、出發、戰前（ch08_pre）、第 1～9 回合北上接戰、
第 4 回合擊倒 Boss（事件 30 對白兩句、倒戈成友軍、登場 group 1）、第 5／6 回合 END 之後事件 31 登場
group 2／3、第 10 回合開頭清場、戰後 ch08_post、洞窟中的激戰城鎮出售與買入（金幣 3000→3037→3027）、
教會與酒店探訪、酒店存檔（整檔 sha256 相同）、Alt+F9 祕密商店；`ai_order` 分岔 0。這一輪修掉的畫面差異：
回合開頭聚焦時提早打開閘 B，小窗翻到右側（seq 814 起 26 點各約 4656 px）。細節見
[56 §第九章章工作單元](56-fd2-remake-sdd.md#第九章章工作單元boss-倒戈休眠回合事件列與回合開頭聚焦的閘-b2026-09-17)。
抽樣對照表分四張 [`parity-ch09-samples-p1.png`](../figures/parity-ch09-samples-p1.png)、
[`p2`](../figures/parity-ch09-samples-p2.png)、[`p3`](../figures/parity-ch09-samples-p3.png)、
[`p4`](../figures/parity-ch09-samples-p4.png)（100 列：每種 kind 第一點、全部 `diff_pixels>0`、指定的
909／924／1438／1702／2667，index [`parity-ch09-samples.json`](../data/ui-traces/parity-ch09-samples.json)）。
限制：強化值不是原版證據；事件 31 的鏡頭巡視與戰後 ACTING 36 落在兩個 checkpoint 之間，畫面 gate 只比到
之後的點。

第十章整章收據 [`parity-ch10.json`](../data/ui-traces/parity-ch10.json)（2026-09-17，remake-r6，原版側
sample-r4、dosgolem `a9bcd62`；槽建到「已通關第 9 章」、11 人，強化 AP+200／DP+0／DX+60）：狀態 `passed`，
280 個原版動作、行為 281 點一致、268 個畫面比較點全部在預算內、174 點逐像素相同，其餘以指令環畫面殘差為主
（最大 428 px，#34）——幻之森林前的洞窟中的激戰城鎮、出發、戰前（ch09_pre）、第 1～9 回合北上接戰、第 5 回合
結束事件 32 登場友軍、兩次寶箱取得金錢（2000→12000→22000）、第 10 回合開頭清場、戰後 ch09_post、幻之森林
城鎮出售與買入（22037→22027）、探訪、酒店存檔（整檔 sha256 相同）、Shift+F10 祕密商店。這一輪修掉的畫面
差異：中毒扣血訊息的聚焦、分派器停留聚焦（鏡頭錯一欄）、chapter 9 熔岩輔助底面（FDOTHER #15）、友軍道具
原地使用、#40 途中翻邊。細節見
[56 §第十章章工作單元](56-fd2-remake-sdd.md#第十章章工作單元狀態扣血訊息友軍休眠停留聚焦與輔助底面2026-09-17)。
抽樣對照表 [`parity-ch10-samples-p1.png`](../figures/parity-ch10-samples-p1.png)…[`p5`](../figures/parity-ch10-samples-p5.png)
（113 列，index [`parity-ch10-samples.json`](../data/ui-traces/parity-ch10-samples.json)）。
限制：強化值不是原版證據；輔助底面相位由收據推回（強推論）；Boss 與事件 33 不在抽樣回合內。

第十一章整章收據 [`parity-ch11.json`](../data/ui-traces/parity-ch11.json)（2026-09-18，remake-r3，原版側
sample-r2、dosgolem `a9bcd62`；槽建到「已通關第 10 章」、13 人，強化 AP+200／DP+0／DX+60）：狀態 `passed`，
240 個原版動作、行為 241 點一致、231 個畫面比較點全部在預算內、157 點逐像素相同，其餘以單位動畫相位與
指令環畫面殘差為主（最大 489 px，#34）——幻之森林城鎮、出發、戰前（ch10_pre：登場友軍 group 1 與
ACTING 38／39）、第 1～6 回合全隊南下接戰、敵方 mode 5 沿途取寶箱（記錄 37 撿走 slot 7 的 10000 金，
被擊倒時掉給玩家：2000→12000）、第 7 回合開頭清場、戰後 ch10_post 的 JOIN 14（珊）、北山道城鎮出售與
買入（12037→12027）、探訪、酒店存檔（整檔 sha256 相同）、Ctrl+F1 祕密商店。這一輪修掉的畫面差異：
敵方撿走寶箱之後那一格要換成打開的箱子（`0x12263` 就地把 tile word +1，繪圖端跟著讀同一份可變緩衝），
以及 mode 5 撿到的東西要能在死亡時掉出來。細節見
[56 §第十一章章工作單元](56-fd2-remake-sdd.md#第十一章章工作單元mode-5-取寶箱與打開的箱子2026-09-18)。
抽樣對照表 [`parity-ch11-samples-p1.png`](../figures/parity-ch11-samples-p1.png)…[`p5`](../figures/parity-ch11-samples-p5.png)
（91 列，index [`parity-ch11-samples.json`](../data/ui-traces/parity-ch11-samples.json)）。
限制：強化值不是原版證據；玩家法術／物品與第二次踏事件格沒有驅動端指令，抽樣不涵蓋。

### 2026-09-09：我方被攻擊時台座蓋住腳步（RUNTIME-E1）

全螢幕戰鬥演出的左右是資料決定的：FIGANI 幀標頭內嵌的絕對螢幕座標按陣營分邊
（亞雷斯 12／13 在 x=89..178，盜賊 288／289 在 x=6..28），同一單位不論攻守都
留在自己那一側。`TAI.DAT` 台座固定在右側 slice，因此會壓到台座的永遠是我方那
張圖。`drawBattleScene` 先前固定畫成「守方 figure → 台座 → 攻方 figure」，敵方
攻擊我方時我方成了守方、被畫在台座之前，腳步就被蓋住。現在由
`battleSceneLayerOrder` 保證**台座緊接在我方 figure 之前**，兩個方向都有回歸釘。

原版側的對照來自敵方攻擊我方那一段的 dosgolem 收據（我方索爾在右、背影、踩在
台座上、腳沒有被蓋住），量測、擷取條件與前後畫面見
[`fd2-battle-pedestal-zorder-20260909.json`](../data/ui-traces/fd2-battle-pedestal-zorder-20260909.json)
與 [`battle-pedestal-zorder-20260909.png`](../figures/battle-pedestal-zorder-20260909.png)，
成因與規則見 [103](103-battle-pedestal-zorder-20260909.md)。我方攻擊方向的既有
fixture 逐位元組不變，本項不提升演出時序或逐幀分鏡為 E2。

### 2026-09-09：地圖走行幀數與單位上下位移（RE-CLOSED／RUNTIME-E1）

原版每格發佈六幀 `+4`（1..6），跨格那一幀 raw 座標換成新格、`+4` 同時重設為
1，中間沒有 `+4 = 0`；只有整段抵達才是 pose 0／`+4` 0。`stepBattleWalk` 依此
改為 `nativeMapGridMotionFrames = 6`，第七次呼叫提交新格之後在同一幀直接接上
下一段的第一拍。

同一條線修正 `fdicon.NativePlacementOffset`：上下方向位移原本寫成一整格高度
（24×456），已閉合的 byte equation 是 `unit[+4] × {+0x720,-4,-0x720,+4}`，
`0x720` 是 456-byte stride 的四列＝四像素。四像素乘六拍剛好一格。修正前垂直
移動每拍跳一整格，單位整段離開可見範圍、抵達才回到終點格。逐幀對照、收據與
出處見 [`fd2-walk-frames-and-placement-20260909.json`](../data/ui-traces/fd2-walk-frames-and-placement-20260909.json)
與 [99](99-move-confirm-cursor-20260909.md)；圖見
[`native-walk-placement-20260909.png`](../figures/native-walk-placement-20260909.png)。
本項不宣稱兩側逐像素一致，也不涵蓋劇情走位的每格幀數。

### 2026-09-09：可見游標的寫入端與走行步進的視圖跟隨（RE-CLOSED／RUNTIME-E1）

`[0x53AB9]`／`[0x53ABD]` 的完整寫入端已列出（IDA 9.4 data xref，Capstone 的
LE fixup 清單一致）：四個鍵盤游標處理器 `0x11B48`／`0x11B9B`／`0x11BFA`／
`0x11C59`、四個走行步進 `0x12EAA`／`0x1300D`／`0x13185`／`0x13315`，加上
`0x10010..0x10620` 的初始化與 `0x205DA`、`0x233C6`、`0x235F9`、`0x23E74`、
`0x25757` 的歸零。沒有一處由 `cursor - camera` 重算；劇情捲動 `0x135DD` 與
直接設定游標的 `0x149F8` 都不碰它，所以確認移動之後可見游標會合法地停在舊值。

安全帶規則兩家族相同（上 `>= 2`、下 `<= 5`、左 `>= 2`、右 `<= 0x0A`，
超出改捲鏡頭），但判準來源不同：鍵盤處理器讀已存的可見游標，走行步進算單位
自己的相對列。消費端 `0x1741C` 以 `visible_x * 24 + visible_y * 24 * 0x1C8`
定位指令環，證實它就是視窗內的格座標。

重製端據此拿掉 `visible == cursor - camera` 的檢查（改檢查 13×8 視窗界線）、
補上 `FocusNativeMapCursor`（`0x12CEA`：先 X 後 Y 逐格走鍵盤處理器，確認移動
由 `0x18A26` 走這條）與 `AdvanceNativeMapWalkStepView`，走行每提交一格視圖跟著
走一格。證據見
[`fd2_visible_cursor_writers_ida.txt`](../data/ida/fd2_visible_cursor_writers_ida.txt)，
接線與界線見 [99](99-move-confirm-cursor-20260909.md)。本項不宣稱兩側逐像素
一致，也沒有涵蓋 `0x53B0B`／`0x53AF1`／`0x53AF5` 的語意。

劇情 pan `0x135DD` 同一輪也已閉合：迴圈對每一格同時位移鏡頭與絕對游標且位移量
相同（`0x13606`／`0x1360C`、`0x13614`／`0x1361A` 與 Y 軸同一對），X 先走完再走 Y，
每格一次 `0x11CAC(0)`，進入時 `mov [0x51A83], 0` 且返回前不還原。dosgolem 從
START 走完序章取到三次 pan 共 126 格逐格對上，收據見
[fd2-story-pan-cursor-20260909.json](../data/ui-traces/fd2-story-pan-cursor-20260909.json)。
重製端 `syncStoryNativeMapPanView` 據此改成平移鏡頭差量，四個發動 pan 的地方
（beat `pan`、battle event `pan`、回合登場演出、截圖快轉）共用同一個 `camPanJob`。

同一份收據另量到原版容許可見游標暫時離開 13×8 視窗（15 格之後 `visible_y = −1`）。
原版的界線寫在寫入端自己的分支條件裡：`0x11B9B` 以 `[0x53AC5]-1` 夾絕對游標、
以 `[0x53AC5]-8` 夾鏡頭，可見游標則只有 `inc`／`dec`，八個寫入端都不檢查範圍。
重製端因此分三層：執行期狀態只夾鏡頭與絕對游標；節點常數走
`campaign.NativeMapViewConfig.Validate`（進場即繪，都在界內且
`visible = cursor − camera`）；13×8 由
`NativeMapViewState.VisibleCursorInViewport()` 在三個把它當畫面格座標的消費端把關
（`fdother.ActionOverlayOrigin`／`ActionOverlaySnapshotOrigin`、
`native_unit_present`、`native_command_heal_presentation`）；`native_current_save`
另有存檔標頭的 byte 範圍與恆等式閘門。
走行捲動 `0x13185` 家族的發布也改成逐格套用寫入端規則，與戰鬥走行共用
`battle.AdvanceNativeMapWalkStepViewState`；序章 15 格捲動的端點
（camera_y 34→20、cursor_y 34→19、visible_y 0→−1）由同一份收據背書。

### 2026-10-01：玩家取箱／可變 HUD 與第十二章（PLAYER-E2，111例外）

原版 sample-r5（dosgolem a9bcd62、固定 FD2.EXE SHA-256、tracked dirty0）與 remake-r8 完整重播通過：286 原版動作、289 重製檢查點、287 行為比較點、275 畫面點，176 點0px、最大215px；所有計畫節點齊全，酒店存檔整檔 SHA-256 6e8823cae90191bcf7813841a5e9a514f119a717be3a24762231c8a6de8af821 相同。 寶箱slot0／物品56／(16,15)由玩家第10回合取得，seq3853同格HUD與打開的箱子逐像素相同；第1回合事件35、第五回合事件36、保護友軍、戰後JOIN17、買賣、酒店存檔與祕密商店均納入。

[正式收據](../data/ui-traces/parity-ch12.json)、[抽樣索引](../data/ui-traces/parity-ch12-samples.json)與[7張對照圖入口](../figures/parity-ch12-samples-p1.png)保存115列。這是#53修正前的歷史抽樣（2026-10-01，以下差異數已失效；目前見本檔#53節）：非零差異有99點，指令環相位為主，HUD縮圖13個1px點及取寶提問游標框134px當時仍留#52／#53；不得宣稱全章逐像素一致。敗北返回標題#47、玩家法術／物品操作與#41未完成；第四～十一章本輪未重跑。

## 2026-10-01：第十二章敗北返回標題

[正式收據](../data/ui-traces/ch12-defeat-return-title.json)列PLAYER-E2（114建構槽限制）：正常LOAD、整備、五回合章內鍵盤輸入進敗北，兩張提示各0差異像素且indexed／PNG SHA一致；9／36 BIOS ticks順序不可由確認鍵略過，完整既有標題owner自行交出menu，原生存檔不變。START重開章0另列RUNTIME-E1；本收據未比較完整標題所有幀，不宣稱自然難度／傷害／存活或硬體時鐘相同。勝利分支同日現行程式重跑四項驗收仍通過。有限規格與消費端勘誤見[58](58-fd2-exe-re-coverage.md)。

### 2026-10-01 建槽 JOIN 驗證範圍

#23 的工具殘值修正只有 RUNTIME-E1：四種殘值與存檔往返通過，未新增介面或一般玩家驗證。前輪第二章最終存檔的兩個 item byte 尚不同；本輪以新原版實驗及戰後尾格投影消除差異，有限驗收通過。[58](58-fd2-exe-re-coverage.md#2026-10-01建槽-join-殘值與戰後複製分層23)保存現況與工具收據，既有章 PLAYER-E2 不由此提升。

本輪 #23 九項驗收及正式 JOIN→sync→酒店存檔 E1 回歸見[有限收據](../data/fd2_join_copyback_verification_20261001.json)；原版明示清場，只驗證物品尾格，既有章 PLAYER-E2 與介面畫面等級不變。

## 2026-10-01：#53取寶提示底圖有限驗收

正式收據與抽樣圖已由treasure-original-r2／treasure-formal-r2重生，入口仍為上列[收據](../data/ui-traces/parity-ch12.json)與[抽樣索引](../data/ui-traces/parity-ch12-samples.json)。
提問seq3820的游標框差異134→0像素；取得後seq3853仍為0，酒店存檔整檔雜湊一致。
275張中177張0像素，98個非零點最大215，未改像素預算。取寶底圖契約CONFORMED，
原始selector不變；完整caller內部解釋仍為強推論，不能推廣至其他對話。
#52的地圖tile27殘差（原稱HUD縮圖，見下方勘誤）與其他指令環相位差異仍保留，不宣稱全章逐像素；玩家法術／物品操作與#41仍未完成。


## 2026-10-01 #52 地圖殘差定位勘誤

[主證據](../data/ida/fd2_terrain_mode3_review_20261001.json)以正式第十二章
收據逐張核對17個殘差點：13張全畫面1px、4張與指令環差異共存。
camera→world tile對照均為tile27局部(7,9)，畫面座標y85／109／133／157，
不在底部HUD；因此否定前輪「HUD地形縮圖」定位。歷史抽樣及舊Issue名稱保留作追溯。
IDA LE 0x1220C／0x12220分別呼叫raw 0x4DEDA／LUT 0x4DCC6；既有mode3
契約分別保留目的底色／讀目的色經LUT寫回。這些指令契約已證實，不重開既有RE。
每張殘差的實際runtime分支與目的底色writer仍未知；不可從archive byte+3猜測，
也不可固定色、遮罩或調高640px預算。#52維持開啟；正式runtime未改，本項未CONFORMED。


## 2026-10-01 第十三章起手有限驗收與目前阻塞

[#58](https://github.com/wicanr2/fd2_re/issues/58)依READY規格修正可編輯scenario的
原生handler承接與group0，正式battle節點接入已證實入口view及持續HUD。
[同槽預檢](../data/ui-traces/parity-ch13-preflight.json)seq48名冊共59筆
（15我方／12友軍／32敵方）、原生順序／座標／HP及320×200未遮罩畫面對上原版，RGB差0px。
限於這個起手切片標CONFORMED／RUNTIME-E1；未提升整章PLAYER-E2。
[原始位址、完整chunks與有限驗收](../data/ida/fd2_ch13_handoff_20261001.json)保留出處。
完整Go回歸19套件通過；編輯器canonical由受版控exporter重生及驗證清冊雜湊。

目前原版r1第六回合回標題，沒有完成計畫。seq2110仍15名我方及3名友軍HP>0，
seq2111後戰場緩衝失效，不得把title時ally_alive=0當作全滅。前77動作凍結預檢
行為gate通過，但35個畫面點超過640px（最大11901px），節點／交易拒絕截短計畫。
[#60](https://github.com/wicanr2/fd2_re/issues/60)追畫面；
[#61](https://github.com/wicanr2/fd2_re/issues/61)追原版結果判定與正常敗北路徑。
戰役台帳第十三章目前BLOCKED，完成章仍9／30；不重跑已通過第四～十二章。
#59探針已驗證5個requested地址均在完整chunks匯出內，原始名稱／位址／bytes與分級保留。

## 2026-10-01 #60 第十三章畫面前綴勘誤

[同槽收據](../data/ui-traces/parity-ch13-preflight.json)已由preflight-remake-r4重生：
75張、51張0px、最大215px，行為與畫面gate通過。這取代上文r3的35個超標點現況；
剩餘24個非零點位於指令環圖示，不能宣稱逐像素。正式raw色盤表與測試相位條件修正
列有限CONFORMED，詳見[58](58-fd2-exe-re-coverage.md)及[IDA證據](../data/ida/fd2_palette_cycle_table_20261001.json)。
完整第十三章仍因#61計畫未完成而BLOCKED，節點／交易拒絕，9／30章現況不變。


## 2026-10-01 #61／#62 條件勘誤的介面範圍

[58](58-fd2-exe-re-coverage.md)與[IDA主證據](../data/ida/fd2_ch13_result_conditions_20261001.json)
已閉合第十三章兩個code1條件及event5共用尾段，來源工具清冊通過六項回歸。
本輪沒有修改正式介面或新增同狀態畫面收據；第六回合原因仍為強推論，
返回標題／對白接線尚未達RUNTIME-E1，整章PLAYER-E2與9／30章保持不變。

#61本輪僅新增第四回合event5資料接線，原生append與動作順序達RUNTIME-E1；
獨立原版record59的HP／座標／身份與名冊順序通過。沒有新增正常UI同狀態畫面
收據；對白畫面與整章等級不提升。上段「沒有修改正式介面」只指未新增畫面驗收，
目前資料接線以本段及58末節為準。


## 2026-10-01 #61 第十三章有限敗北閉合

[58現況](58-fd2-exe-re-coverage.md)與[正式有限收據](../data/ui-traces/ch13-defeat-return-title.json)
取代此前「結果DRAFT／原因僅強推論」的現況：原版207EC→20815→22E5C
已重生，正式逐條raw結果與阻塞對白按READY契約實作；同槽第六回合只命中記錄59分支，
兩張提示各0px且indexed／PNG SHA相等，完整標題返回與SAV不變通過。
限於這條敗北路徑標CONFORMED／RUNTIME-E1；原版建構槽的normal_player_path_verified=false
及受控決策點RNG均明示，不提升整章PLAYER-E2，不宣稱自然難度、傷害、存活或硬體時鐘相同。
19個Go套件與文字審查綁定通過。第九回合完整event7、友軍保護與戰後／存檔仍未閉合，
#61保持開啟，9／30不變；已通過第四～十二章未重跑。


## 2026-10-01 #61 event7 接線與 #63 戰後測試勘誤

[58 現況](58-fd2-exe-re-coverage.md)與[IDA 主證據](../data/ida/fd2_ch13_result_conditions_20261001.json)
記錄第九回合完整 pan／spawn／ACTING46／reset／文字8，READY 後接入可編輯資料。
GUI 逐幀回歸確認演出完成後才出對白，收框後才繼續；建構狀態列 DATA-READY／RUNTIME-E1，
沒有第九回合原版同狀態畫面，不提升 CONFORMED 或整章 PLAYER-E2。

#63 修正章13戰後測試的假缺件 SKIP：FDICON.DAT 改為固定清單的 FDICON.B24，
補正式語系與 group1／group2 前置狀態。現在實際通過12段原生對白、JOIN3、town14與存讀檔；
舊套件綠燈包含被略過的測試，不能回溯宣稱戰後已驗證。其餘同類測試另登記 #64。
原版保護計畫 r2 第八回合 slot59 陣亡返回標題，尚未到 event7／戰後；失敗收據保留。
全套首輪18個 Go 套件通過，修正後同容器整個 cmd/fd2 乾淨重跑通過，合計19個套件。
#61 保持開啟，戰役台帳9／30不變，已驗收第四～十二章沒有重跑。詳細重生入口與限制以58為準。


## 2026-10-01 #64 戰後測試勘誤與 #61 東側護援

[58 現況](58-fd2-exe-re-coverage.md) → [測試收據](../data/fd2_post_fixture_verification_20261001.json)。
其餘錯用FDICON.DAT的戰後測試改用固定FDICON.B24與正式語系，揭露並修正舊前置：
event25陣營入口、第六／十章名冊形狀，以及收框後游標飛回的精確幀數。
全部相關目標測試實際PASS而非SKIP；正式規則未改，僅建構E1，不能回溯把舊套件綠燈當已驗證。
完整Go回歸與第十三章東側護援仍在有界執行；終態以58及遠端Issue為準，戰役9／30不變。

#64本輪完整Go回歸已結束：19個套件全部通過，遊戲套件178.586秒；八條戰後測試的16分支明確PASS而非SKIP。正式收據已保存全套日誌雜湊。這取代上段「完整回歸仍在執行」，原版第十三章護援程序仍有界執行，現況與限制由58承載。


#61東側護援原版程序已終止（exit15）：第七回合返回標題，未到event7／戰後。
[主證據](../data/ida/fd2_ch13_result_conditions_20261001.json)的east_guard_attempt
保存最後有效checkpoint3183：初始紀錄15～26全部HP0／bit0=1，後到紀錄59仍HP52／bit0=0，
我方0／3／14仍活著。依已閉合20765條件，原因屬初始友軍分支的強推論；本run未追207A4 writer，
沒有把條件相符提升成直接執行證據。保留完整失敗樣本，本輪不再重啟整章或縮減第九回合門檻。
111明定抽樣該章事件／增援回合，不能以提早清敵代替；下一步先用有效戰場狀態調整局部正常輸入。
本輪曾將全Go回歸與oracle並行，違反111執行紀律；結果如實保留，後續兩者串行。
所有本輪FD2工作均已退出，沒有待續執行程序；第十三章仍BLOCKED，9／30不變。


## 2026-10-02 #65 原版掃描末名自動換手勘誤

第十三章東側護援的有效失敗狀態保留；目前收據另外直接證實工具跳過玩家回合：
seq1253末名行動使round1→2，隨後seq1257～1266仍送END；seq2333的round3→4也有相同情況。
`tools/dosgolem_oracle_drive.py` 的 `do_sweep_round` 只在迴圈開頭檢查 `stop_on_auto_end`，
最後一次行動剛好耗盡 `max_units` 時漏檢；此缺陷登記於 [#65](https://github.com/wicanr2/fd2_re/issues/65)。
因此此前「只需調整兩側護援戰術」的解釋不充分，應先修正工具回合邊界。

修正只增加最後一次行動後的回合進位檢查，未啟用旗標的舊計畫仍沿原語意；
沒有修改正式遊戲規則、槽、章內狀態或第九回合門檻。舊程式在新增末名案例失敗，
修正後77項工具測試通過。受版控[單回合計畫](../data/parity-plans/ch13-auto-end-boundary.jsonl)
沿相同首回合操作，驗證範圍只到第二回合起手；終態與來源雜湊將記入
[工具收據](../data/fd2_oracle_sweep_auto_end_20261002.json)。完整章仍由58與#61管理。

單回合正式終態已通過：exit0，checkpoint1254停在第二回合cursor，沒有end_turn動作或章內注入。
前1254筆（seq0～1253）的指令位置、亂數字組、名冊、視圖及輸入鏈與舊收據全部相同，
因此差異確定落在末名自動換手後的工具收尾。詳細雜湊、來源與重生入口見上述工具收據。
本輪沒有重跑已通過章節；此工具驗證不提升第十三章PLAYER-E2。


## 2026-10-02 #66 第八回合麻痺選取工具缺陷

#65修正後的相同完整計畫 `east-guard-original-r2` 已終止，exit9；
checkpoint5360仍在第八回合角色狀態面板，原始record20 HP95／bit0=0、後到record59 HP119／bit0=0。
兩個敗北writer 0x207A4／0x207EC都沒有trace命中；這次停止是工具缺陷，不能寫成護援失敗。
record14的raw+0x26=3，原版依0x1191D..0x1192B進sub_17AED面板，
input_chain的0x17B0B等鍵caller被工具泛用對白判準誤判，接戰仍等target而退出。

[#66](https://github.com/wicanr2/fd2_re/issues/66)修正候選與顯式操作跳過麻痺單位、
將該caller優先分類status；舊工具三條行為回歸全部失敗，修正後81項工具測試通過。
[原指令與工具收據](../data/fd2_oracle_paralyzed_selection_20261002.json)保存固定原版雜湊、
既有IDA來源、有效原版畫面與raw投影、日誌雜湊及測試範圍。正式遊戲規則沒有改動。
修正後完整原版計畫尚未重跑；第九回合／戰後四gate未驗收，第十三章不提升PLAYER-E2。
本輪不再重啟整章；先保留有效終態，後續重生沿維護工具與相同計畫，原版／全Go仍串行。


## 2026-10-02 #61／#67 第十三章最新驗收

原版r3已完成第九回合event7與戰後酒店存檔；行為、交易與258張畫面通過，
最後秘密商店計畫誤用戰前鍵使節點失敗。三份計畫改為town_ch14的selection2／Shift+F3；
合法酒店存檔短程LOAD的城鎮／秘密商店各0px，列有限RUNTIME-E1。正式遊戲沒有改動。
完整章仍未通過；目前真相、收據與重生入口統一見[58最新段落](58-fd2-exe-re-coverage.md)。


## 2026-10-02 #61 第十三章完整四項驗收通過

修正後同槽完整r4原版／重製重播通過；[正式收據](../data/ui-traces/parity-ch13.json)
行為、節點、交易與259張畫面全部達標，最後秘密商店0px，酒店SAV雜湊相同。
依111／114例外列PLAYER-E2，保留建構槽、清敵與亂數設定限制；不拼接舊收據。
目前真相、範圍與重生入口統一見[58最新段落](58-fd2-exe-re-coverage.md)。

## 2026-10-02 #68 首頁對拍表格勘誤

第十三章正式收據入庫後，上一輪未執行既有產生工具，README與REMAKE-STATUS仍列第4～12章。
現已由 tools/render_parity_progress.py 依台帳與收據重生兩處區塊，並以 --check 驗證一致；
區塊外文字逐位元組保持不變。此修正不新增玩家驗證範圍，現況與限制沿用58的第十三章完整收據。

## 2026-10-02 #70 第十四章原生起手有限驗收

依READY規格，Scenario改為group0與runtime-append、補第16部署格；原生鏡頭與繼承HUD接入canonical正式戰役，legacy來源同步。
[原版IDA與逐欄名冊](../data/ida/fd2_ch14_startup_20261002.json)與[同槽收據](../data/ui-traces/ch14-native-startup.json)證明67筆前沿、三張完整RGB畫面各0px；完整Go回歸19套件通過。
本切片列CONFORMED／RUNTIME-E1；整章12張最低量不變，有界比較器仍回傳failed，不借三張起手畫面宣稱PLAYER-E2。
#69的受版控整章計畫另抽樣三回合後清敵，再驗戰後與交易存檔；正式收據尚未產生。

## 2026-10-02 #69 第十四章完整原版與重製失敗診斷

[58最新現況](58-fd2-exe-re-coverage.md)保存完整原版終態、重製失敗點與分層限制；
[完整首次診斷](../data/ui-traces/parity-ch14-r1.json)四項未通過，整章不列PLAYER-E2。
#71事件10規格仍DRAFT、#72較早AI順序差異與#73戰後前沿待修；本次未修改正式事件行為。

## 2026-10-02 #69 第十四章完整四項驗收

[正式收據](../data/ui-traces/parity-ch14.json)的行為、節點、64張整幀畫面與全檔酒店存檔皆通過。
事件10、mode8、戰後前沿與排列依READY規格完成；原版168筆AI入口無順序分岔。
依111／114例外列PLAYER-E2／有限CONFORMED，建構槽與清敵限制不變。
唯一分層現況、完整重生入口與歷史勘誤統一見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-02 #77 第十五章原生起手有限驗收

[原版IDA與逐欄名冊](../data/ida/fd2_ch15_startup_20261002.json)及[同槽收據](../data/ui-traces/ch15-native-startup.json)
證明16我方＋58筆group0＝74筆前沿，16格部署、原生視圖與繼承HUD接入canonical正式戰役。
正常LOAD、出戰確認、戰場起手三張完整RGB畫面差異均為0；完整Go回歸19套件通過，遊戲套件111.014秒。
有限切片列CONFORMED／RUNTIME-E1；整章仍保持12張最低量與戰後／存檔門檻，#76尚未完成。
第4／7／9回合event13／38／18尚缺完整動作，先登記#78再有界補證；不以spawn-only代表完整事件。

## 2026-10-02 第十五章完整首次診斷（#76／#79／#80）

起手三張0px的有限結論保持；完整原版sample-original-r1於round6敗北返回標題，尚未達round7／9與戰後。
最後有效名冊我方16人存活、record64 bit0=1，直接敗北條件及READY規格見[58](58-fd2-exe-re-coverage.md)
與[完整IDA主證據](../data/ida/fd2_ch15_result_conditions_20261002.json)。
重製首次同槽重播另停止於狀態到期提示畫面來源缺漏，#80先分類正式繪製／離屏前置；
本章不列PLAYER-E2，也不以原版釋放後的陣列推算全隊死亡。

### 2026-10-02 第十五章有限修正驗證

#78／#79有限實作與#80離屏前置通過完整Go回歸19套件（遊戲113.367秒）；44事件原版bytes／覆蓋核對與6項轉寫負向測試通過。
#80[可重查收據](../data/fd2_ch15_transient_replay_20261002.json)列有限CONFORMED；正式raw狀態／到期規則未改。
#78仍待round7／9，第十五章同槽敗北仍受#81分岔阻擋，#76四gate保持failed，不能以綠色Go測試宣稱原版一致。
#79分級語意已回填自動匯出索引；1305原始函式邊界／名稱／caller未變，機械重生清冊只更動20822一列。
目前分類62 product／175 runtime／1068 unknown、68條函式註記；來源與命令見[結果主證據](../data/ida/fd2_ch15_result_conditions_20261002.json)。

## 2026-10-02 第十五章 #81 修正後畫面限制

[最新嚴格診斷](../data/ui-traces/parity-ch15-r2.json)保留舊失敗歷史：AI順序分岔0、66個行為點通過，61張完整RGB中60張通過。seq849索引像素全同，水面E0色槽不符合完整raw phase5，仍差16098px；#82已登記，writer未知，不複製原版色值、不遮罩、不放寬640px。這不是對白排版差異。

原生經驗／成長consumer修正列有限RUNTIME-E1；#81、敗北返回標題#79、第7／9回合#78及整章#76仍開啟。唯一目前分層、重生入口與證據限制見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-02 #82 中間窗口修正後驗收

[新嚴格診斷](../data/ui-traces/parity-ch15-r3.json)的61張完整RGB全部通過，最大199px，seq849原版／重製PNG SHA-256相同。640px、最少12張及無遮罩契約不變。原始EIP4E01F與暫存器證實phase6已寫四槽，不再把返回位址16D05誤當目前EIP；#82只修對拍私有候選，不改正式Game DAC。

66個行為點與297AI入口一致，#81／#82有限CONFORMED；節點、交易與#79敗北返回標題未驗，整章不列PLAYER-E2。權威現況與重生入口仍見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-02 #79 正常章回放的敗北返回標題

[58唯一現況](58-fd2-exe-re-coverage.md)與[同槽敗北主證據](../data/ida/fd2_ch15_result_conditions_20261002.json)
取代前述返回標題未驗：原版r1的seq1431已在標題選單，重製r6由相同建構槽正常LOAD及章內動作，
round6 record64 bit0觸發既有敗北提示，完整開場自行回menu，清理戰鬥暫態且SAV不變。
回放此前在result=lose停住，補驗尾端後通過；沒有修改正式敗北呈現或結果規則。

提示兩幀僅保存重製收據；與已閉合第十三章共用FDOTHER79索引雜湊相同屬跨章旁證，
不宣稱第十五章同狀態提示逐像素。本分支列有限CONFORMED／RUNTIME-E1。
sample-verify-r6的61張戰場完整RGB與66行為點仍通過，但節點／交易failed；
整章不提升PLAYER-E2，#76／#78的成功護援、第7／9回合與戰後存檔繼續。

## 2026-10-02 第十五章完整章畫面與存檔驗收（#76／#78）

[正式收據](../data/ui-traces/parity-ch15.json)的完整RGB、行為、節點及全檔酒店SAV均通過；
維持640px、最少12張與無遮罩。護援走過第4／7／9回合事件，依111／114例外列PLAYER-E2。
建構槽、清敵、既有自動換手比較點例外與自然難度限制如實保存；舊失敗／部分收據不覆寫。
唯一分層現況、重生入口與統計依[58](58-fd2-exe-re-coverage.md)與正式台帳，不另保存會漂移的章數。

## 2026-10-02 第十六章有限起手畫面（#84／#85／#86）

[起手收據](../data/ui-traces/ch16-native-startup.json)的城鎮、出戰提示、選人、最終確認、戰場起手五個同槽完整RGB均0px。
選人板外保留caller城鎮，最終確認恢復caller；獨立整備仍為既有黑底。原始素材測試實際PASS，缺來源拒收。
[總覽及來源](../data/ui-traces/ch16-native-startup-samples.json)列兩側完整影像與雜湊；有限CONFORMED／RUNTIME-E1。
整章最低12張、640px與無遮罩契約不變，戰鬥／戰後／交易／SAV仍待#83，不能列PLAYER-E2。
唯一分層現況與重生入口見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-02 第十六章完整畫面與存檔驗收（#83／#87）

[正式收據](../data/ui-traces/parity-ch16.json)行為、節點、76張完整RGB與全檔酒店SAV皆通過。
[固定抽樣來源](../data/ui-traces/parity-ch16-samples.json)保留所有非零差異，最大200px；640px／最少12張／無遮罩維持。
前三回合、戰後18人名冊、買賣、酒店與秘密商店同一完整run走到，依111／114例外列PLAYER-E2。
舊SAV失敗與有限起手收據保留，不宣稱全部動畫相位逐像素；現況及限制見[58](58-fd2-exe-re-coverage.md)。

### 第十七章有限起手（#89，2026-10-02）

正式 `battle_ch17` 已加入有來源的視圖與繼承 HUD。正常讀檔、15次選人與戰前對話後，五點完整 RGB 皆為0像素差異。53筆起手單位狀態一致，分級為RUNTIME-E1。詳見[主規格](../data/ida/fd2_ch17_startup_20261002.json)、[有限收據](../data/ui-traces/ch17-native-startup.json)與[五點總覽](../figures/ch17-native-startup-samples.png)。完整回合、增援、戰後及全檔SAV仍由[#88](https://github.com/wicanr2/fd2_re/issues/88)驗收；不提升整章PLAYER-E2。

### 第十七章完整同槽章收據（#88／#92／#93，2026-10-02）

[正式收據](../data/ui-traces/parity-ch17.json)行為、節點、交易與98張全幅RGB通過，72張逐像素相同，最大295px；酒店SAV整檔相同。固定初始槽沿111／114政策，NPC52與記錄0／2正常移動護援，四回合後第5回合才清敵23筆。turn4 event40追加8名camp1友軍，frontier53→61；原版124筆AI順序與重製全部一致。依111例外列PLAYER-E2，收據保留建構槽、狀態注入與亂數比較範圍，不能外推自然戰鬥、傷害、存活或敵方選目標。

#92共用FIGANI右半畫布末列界線已修正；#93只擴充4E014完整triplet邊界的測試私有色表窗口。兩份主規格列有限CONFORMED；原始失敗收據仍可回查。最終Go19套件、第15章156張與第16章76張四項回歸通過，既有fixture／證據漂移#91仍開啟。

### 第十八章有限起手與山景（#95／#96，2026-10-02）

[有限收據](../data/ui-traces/ch18-native-startup.json)在正常LOAD、15次選人與戰前對話後，五點完整RGB皆0px，53筆起手單位一致。正式戰役已保存有來源的鏡頭／HUD，FDOTHER16/17嚴格分離圖面以462×226鋪底，再依原版鏡頭公式取312×192；透明地形保留山景。缺來源或越界原子拒收，不以oracle PNG作資產。

兩份規格列有限CONFORMED／RUNTIME-E1。最終Go19套件、三條山景測試及第十七章98張四項與全檔酒店SAV通過；私人素材清冊驗證通過。原版／重製PNG與總覽留本機，公開庫保存[抽樣索引](../data/ui-traces/ch18-native-startup-samples.json)。不外推逐步捲動動畫。第3／8回合、戰後、交易與SAV由#94續驗，整章不提升PLAYER-E2；唯一分層現況與統計依[58](58-fd2-exe-re-coverage.md)。

### 第十八章四輪有限介面回歸（#97／#98）

模式9修正後，同一原版r1的正常四輪完整RGB通過既定預算；不新增遮罩或改判準。[有限收據](../data/ui-traces/ch18-turns1-4.json)保留每張差異與兩側雜湊，原圖留本機。原版計畫round5停止，沒有T8／戰後／酒店SAV驗收，整章#94不列PLAYER-E2。唯一現況與回歸入口依[58](58-fd2-exe-re-coverage.md)。

第十八章r2最初153張比較未通過，歷史拒收保留；#100／#101修正後的現況見下節。原版round7敗退，沒有T8／戰後／SAV。這不取代前四輪有限收據，整章#94仍未驗收，拒收範圍及工具入口見[58](58-fd2-exe-re-coverage.md)。

### 第十八章六輪有限回歸（#100／#101，2026-10-02）

同一原版r2的[六輪有限收據](../data/ui-traces/ch18-turns1-6.json)通過153張完整RGB，其中107張0px，最大496px；640px門檻與無遮罩不變。所有已完成行為與213筆AI入口順序一致。只有原版已觀測的正常輸入範圍列RUNTIME-E1，不宣稱中途演出逐張一致。原版round7敗退，T8、戰後、交易與SAV仍缺，整章#94保持未驗收；唯一現況入口見[58](58-fd2-exe-re-coverage.md)。

本批最終Go19套件與第十七章98張、四項及整檔酒店SAV回歸通過，命令與輸出雜湊保存在[六輪收據](../data/ui-traces/ch18-turns1-6.json)。

### 2026-10-03 工具快照勘誤（#91）

[有限工具收據](../data/ui-traces/tooling-closeout-20261003.json)已修正第十七章記錄的舊fixture／診斷摘要漂移，63項相關檢查通過。這次沒有新增玩家畫面或E2證據；第十八章完整驗收仍缺正常T8、戰後與SAV，現況見[58](58-fd2-exe-re-coverage.md)。

### 2026-10-03 第十八章原版停止前綴（#102）

[主證據與有限前綴](../data/ida/fd2_ch18_oracle_stosb_20261003.json)保存r3全部已完成行為、240筆AI順序與152張全幅RGB通過。原版round7繪圖停止，node／transaction拒收；原版r5的1926點前綴一致只證明唯讀觀測未改執行，不提高介面或整章證據等級。

### 第十八章整章驗收（2026-10-03）

[完整章收據](../data/ui-traces/parity-ch18.json)取代先前缺正常T8、戰後與SAV的現況。正式event43／42按原始控制列觸發；T8先增援再完整對白，戰後使用有來源的75槽前沿並清暫態後重算持續裝備能力值。舊55建構形狀保留，來源不完整仍拒收。[前沿契約](../data/ida/fd2_ch18_postbattle_slots_20261003.json)與[同步契約](../data/ida/fd2_ch18_postbattle_equipment_20261003.json)為CONFORMED。

完整行為、節點、無遮罩RGB、交易及全檔酒店SAV通過；第17章與全部Go回歸亦通過。依111／114建構槽及抽樣後一次清敵例外列本章PLAYER-E2，不宣稱自然戰鬥、逐幀演出或全章逐像素一致。[總覽索引](../data/ui-traces/parity-ch18-samples.json)保留所有非零差異；原圖與SAV留本機。唯一統計與#102尚未解決的原版r3停止見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-03 第十九章整章與必出角色拒收（#106至#111）

[正式收據](../data/ui-traces/parity-ch19.json)包含正常LOAD、方向鍵選人、起手、五回合抽樣、清敵後正常END的T6事件、戰後、買賣、教堂、酒店與秘密商店。四項通過；54張完整RGB有39張逐像素相同，最大215px，沒有遮罩或放寬640px。

[拒收前綴收據](../data/ui-traces/ch19-required-party-rejection.json)另驗缺少凱拉斯的原生訊息及返回城鎮，五點差異0／0／0／275／0px。這是有限RUNTIME-E1；原本完整章verifier的failed診斷保留，不能把沒有進戰場的拒收前綴稱為整章通過。

整章依111／114建構槽例外列PLAYER-E2；清敵seq1119在T6事件seq1138之前，槽來源與注入如實記錄。T10、自然戰鬥及逐幀相位不外推；唯一現況與統計見[58](58-fd2-exe-re-coverage.md)。

### 2026-10-03 第二十章原生起手（#112／#113／#114）

[有限起手收據](../data/ui-traces/ch20-native-startup.json)驗證83筆前沿與五點完整RGB，全部0px。正式LOADCH初始group0按既有0x10C50逐列避讓；原始位置列保持18,30，第二筆record35在執行期為18,31。composition取binding明示Map，包含與roster分離的開場場景。共用writer／consumer契約見[主證據](../data/ida/fd2_ch20_initial_placement_20261003.json)。

第十九章r8四項與全檔SAV回歸通過；原生開場、正常封包與初始重疊整合測試通過。完整Go19套件通過，兩份起手規格列有限CONFORMED。第二十章章內／戰後仍在驗證，本段僅RUNTIME-E1；唯一整章統計仍依58與正式台帳。

### 2026-10-03 節點假通過訂正（#117）

舊章比較器的原版序列複製重製UI，舊 `nodes=true` 不能當作獨立介面證據。[新契約](../data/fd2-chapter-node-comparison-contract.json)使用原版動作與輸入鏈；未知來源拒收，跨時序點明列未比較。共用服務對話僅比較家族，不證明酒店／教會／一般或秘密商店身分。

[回歸收據](../data/ui-traces/parity-node-source-regression-20261003.json)確認第十九章四項仍通過；第二十章r2的target／cursor及shop／town兩點被節點判準拒收，三個原有RGB差異保留，沒有遮罩或改640px預算。歷史原始收據及其餘證據保留，第二十章仍未通過整章。

### 第二十章物理確認畫面（#115）

[有限RUNTIME-E1收據](../data/ui-traces/ch20-physical-target-confirmation.json)保留原版r2，以補正正式0x115B6 consumer及取消／待機按鍵owner的重播r2比較。seq593目標等待與seq612後續選人兩張完整RGB皆0px，獨立節點判準一致；64筆AI及全檔酒店SAV不變。第十九章54張／四項與Go19套件回歸通過。沒有注入原版視圖或遮罩；商店錯選由#116追查，第二十章仍未新增PLAYER-E2。


### 2026-10-03 第二十章四項對拍（#112／#115／#116）

[正式章收據](../data/ui-traces/parity-ch20.json)使用同源第十九章酒店SAV、原版r3及重製r1；原版1095檢查點、35動作，64筆AI全消費且零順序分岔。獨立原版節點比較version2、行為、交易與畫面四項通過。31張完整RGB有25張0px，六張差異最大181px；[抽樣索引](../data/ui-traces/parity-ch20-samples.json)包含全部非零點，圖面留本機。酒店22987 bytes SAV SHA49f00f95…兩側整檔相同。

先前攻擊自身無效目標seq593與後續選人seq612，現均完整RGB0px。商店計畫r3由酒店selection0按right到4，再Ctrl-F10／Enter，seq1094完整RGB0px；正式gate規則未改。原r1謝多自然陣亡及r2錯選一般店的拒收仍保留，較早「整章未驗收」是當時狀態，由本節與正式章收據取代。

PLAYER-E2限111／114建構槽例外：本章不再次升級、強化、補血或改金幣，seq800一次清敵56筆，沒有lock_ally_hp。不得外推自然難度通關、傷害、存活或敵方選目標。未抽樣及not_comparable節點仍未知。完整Go r5共19套件、第十九章r9／比較r11四項與54張RGB、全檔SAV通過。唯一整章統計依58與正式台帳。

### 2026-10-03 第二十一章有限起手（#119／#120）

[收據](../data/ui-traces/ch21-native-startup.json)包含獨立原版節點及五張完整RGB，全部0px。原版75筆前沿、record1 identity21；重製快照只序列化72筆active，未序列化的HP0 slots3／7／13如實列為限制，不宣稱其全部raw欄位相同。camera26,16／cursor37,17來自固定原版狀態，沒有擷取後注入視圖。

缺約拿的原版拒收及Enter返回城鎮另有五點RGB，最大275px。原章比較器在此拒收前綴仍有兩個節點來源缺項，保留failed診斷；此有限owner驗證不稱整章四項。r2最後PNG缺落檔，r3只增加兩次有界空白等待，前108點CPU／輸入／RNG一致。兩份規格限RUNTIME-E1列CONFORMED；章內、戰後、交易與酒店存檔仍待#118。

### 2026-10-03 第二十一章四項對拍（#118）

[正式章收據](../data/ui-traces/parity-ch21.json)包含987個原版檢查點、34個動作與54筆AI入口，全部消費且順序零分岔。行為、獨立原版節點、交易及完整RGB四項通過；酒店22987 bytes SAV兩側SHA-256均為22399b547dc8457ecbf5df982da99cc594817482981d90ea9dd84a50fd4a81bd。較早「戰後與存檔待驗」由本節取代。

30張完整RGB有24張0px，其餘六張最大375px，維持640px預算。[抽樣索引](../data/ui-traces/parity-ch21-samples.json)納入全部非零點，原版圖面留本機。人工檢視差異位於人物與指令環局部動畫／遮擋區域；精確相位仍未知，未以遮罩消除。

PLAYER-E2僅依111／114建構槽例外。本章沿用第二十章酒店SAV，不再次強化或治療；正常敵方階段後於T2清敵一次。材料不足分支、正常城鎮販售／購物／服務探查／酒店存檔／Alt-F1秘密入口已抽樣。天然難度、未抽樣畫面、死亡slots3／7／13完整raw欄位、HUD B精確值及製作成功臂均不外推。唯一現況與統計見[58](58-fd2-exe-re-coverage.md)。

### 2026-10-03 第二十二章有限起手（#124／#126）

[收據](../data/ui-traces/ch22-native-startup.json)含正常LOAD、提示、選人、最終確認與起手五張完整RGB，全部0px，獨立原版節點及行為通過。原版66筆、record1 identity24，camera16,27／cursor22,32／visible6,5／range1；重製只序列化63筆active，未外推三筆死亡raw欄位。

缺希爾法r1的拒收與Enter返回城鎮另五張完整RGB，全部0px；通用章比較器仍有兩個節點缺項，failed診斷保留。B=1只適配nonzero HUD，原版精確byte未知。本段限RUNTIME-E1，不稱整章四項；正式台帳維持18／30。

### 2026-10-03 非城鎮整備四槽保存（#125／#127）

[主規格](../data/ida/fd2_preparation_record_save_20261003.json)與[有限收據](../data/ui-traces/preparation-record-save.json)限局部RUNTIME-E1列CONFORMED。2CC76問題YES→2CCBB／3009C四槽；Enter寫完整SAV後保留列表，ESC才到31A2E零勾選選人。重製已補上正式owner，NO不保存；素材／來源或寫入失敗即停止。

四個完整RGB皆0px，22987 bytes SAV兩側SHA-256相同。原版成功寫入22528+459 bytes，內容相同也屬合法覆寫。driver、replay及比較器新增preparation_save；缺planned save、錯owner、無成功寫入均拒收，沒有跨時序畫面豁免。相關三套件與第二十一章30畫面／全檔SAV回歸通過。建構raw22槽不代表第二十二章戰後可達；#123仍開啟，整章數以正式台帳為準。

本批最終回歸：19套件／2293個Go測試全部通過；97個Go審查候選文字與各處置數不變，只遷移原始碼定位。第一次清冊綁定失敗保留於有限收據，不能視為玩法缺陷。

### 2026-10-03 第二十二章四項與非城鎮戰後保存（#123）

[正式章收據](../data/ui-traces/parity-ch22.json)含975個原版檢查點、42個動作、212筆AI入口，全部消費且順序零分岔。行為、獨立原版節點、交易與完整RGB四項通過。正常戰後11句到非城鎮整備，保存seq965的22987 bytes SAV兩側SHA-256為3c7298cd30221a4220656f77e7676bc89b9002aa02b793de2cc16e0491585db1；本章不是戰後城鎮。

34張實際RGB比較有28張0px，其餘六張最大185px，640px預算及無遮罩判準不變。人工檢視差異限人物輪廓與指令環局部，精確動畫相位仍未知。原r2 seq915延後PNG未計入；同源r1該點CPU／輸入／units／view均相同，另有有限T5完整RGB0px收據，不包裝成r2畫面。#132產生表只計具有實際diff_pixels的點。

PLAYER-E2限111／114例外。沿用第21章SAV，T3／T5正常抽樣後seq916一次清敵54筆，不鎖HP、不追加治療或強化。T7未抽樣，原r1第五回合敗北保留診斷；raw HUD B、三筆死亡槽中途raw及未比較點不外推。較早本章未驗收由本節取代，唯一統計依[58](58-fd2-exe-re-coverage.md)。

### 2026-10-03 第二十三章有限開場

[起手收據](../data/ui-traces/ch23-native-startup.json)保存同一第22章SAV、seed4、正常LOAD／NO／選人與零章內注入。合法分支的記錄問題、選人、確認及開場四張完整RGB各0px；缺希爾法訊息與返回重選分支五張各0px。camera(14,29)、cursor(19,35)、visible(5,6)與存活名冊對齊；九筆HP0不由active快照外推逐欄一致。

拒收通用nodes仍因departure_confirmation來源標籤及seq76沒有action而失敗，原報告保留。有限CONFORMED只依31E65／31A2E、正常正式輸入與完整RGB，不宣稱該分支四項或整章PLAYER-E2。正式HUD B精確值、晚期event52、戰後與本章保存另驗。完整Go、前章與保存probe回歸通過，唯一現況入口為[58](58-fd2-exe-re-coverage.md)。

#135城鎮返回補驗：同一原版拒收前綴正常Enter後為2CE08，重播已回town；錯用出發caller2D170會拒收。額外城鎮分支只驗正式API／source owner，非城鎮五張0px與合法四張0px回歸通過；不將城鎮前綴稱整章驗收。

### 2026-10-03：第23章T2戰後有限診斷（#139／#140）

[有限收據](../data/ui-traces/ch23-post-frontier.json)沿用第22章正式SAV與seed4，正常T2後清敵一次，走戰後對白到整備保存。原版有效前沿42直接反證exact86；binding只接受42及既有明示86局部fixture，其他數量仍拒收。

[視圖契約](../data/ida/fd2_ch23_post_view_20261003.json)沿用233C6的23465..23493原始writer。247B4布局同時重設camera／absolute cursor，可見cursor清0，range gate清0；native focus／pan在post兩個載體同步發布。沒有放寬徑向幾何或夾中心。離屏測試補Draw回報後走完2189A與戰後，不改正式演出節拍。

14張完整RGB中12張0px，兩張move為179／70px，依111既定640px上限通過，精確動畫相位未知。24筆AI入口零分岔；保存22987 bytes兩側SHA-256均eab7665b117d44e355157fa5501a677dc192a2e5301669bc2cb2560d6616e279。第22章34張與全檔SAV回歸通過。特效中間幀未逐幀對拍，晚期event52／#138與#133整章仍開啟，不改正式章台帳。完整Go r6：19套件／2298個測試通過、27項略過。#139／#140限本批RUNTIME-E1列CONFORMED。

### 2026-10-03：第23章事件52與正常戰後保存（#133／#138／#141）

[主契約](../data/ida/fd2_ch23_event52_20261003.json)與[正式收據](../data/ui-traces/parity-ch23.json)已列CONFORMED。52於T13呼叫254／255空群組仍PAN與白閃，T15追加2／3、T18追加8／9；producer依已證低byte算式代入控制列，沿用既有staging格式。#141只修本caller：發布預檢roster時保留PAN後六全域，避免視圖回滾。

同源SAV與seed4，正常T18移動後到T19玩家游標清敵一次，62筆進正常戰後與整備四槽保存。現行ch22_post只接受42／62／86；62有原版24962 consumer與完整保存直接證據，T2有限42／86契約保留原適用範圍。正式台帳、測試與限制統一引用[58](58-fd2-exe-re-coverage.md)；PLAYER-E2限111／114建構槽例外，T22未抽樣，不宣稱傷害、存活、AI選目標或特效中間幀parity。原r2／r3敗北與#141失敗RGB保留。

### 2026-10-03 第24章兩種正常選人與T3戰後有限比較

[有限收據](../data/ui-traces/ch24-finite-parity.json)保存兩條同源正常LOAD分支。前15位選人五張完整RGB皆0px；存活15位選人至T3正常戰後保存的13張有12張0px，移動seq121差88px，範圍為人物局部。沿111既有640px上限，不遮罩；精確動畫相位未知。

存活分支四項門檻與22987-byte完整SAV一致。舞台取raw trace最後完成的11EB0 copy相位，工作列偏移與已發布列偏移各自記錄；選人造成的camera9／10分支由正式對白與尾端聚焦繼承。原r1／r2 T4敗北及各次失敗重播保留。本段限RUNTIME-E1，未比較特效中間幀、T4／7／10或自然難度；#142整章工單仍開啟，正式章台帳不增加。

2026-10-04：第24章晚期原版抽樣先受CPU指令缺口#146阻擋。修正與同槽重跑現況引用[58](58-fd2-exe-re-coverage.md)及[D8主證據](../data/ida/fd2_ch24_oracle_d8_20261004.json)；本工具切片不提升UI或整章驗收等級。

2026-10-04：FCOS有限驗收與後續PF分支阻擋#148引用[58](58-fd2-exe-re-coverage.md)及[三角函數主證據](../data/ida/fd2_ch24_oracle_fcos_20261004.json)。CPU指令切片不提升玩家介面或整章驗收。

2026-10-04：PF分支有限驗收及後續FILD16阻擋#149引用[58](58-fd2-exe-re-coverage.md)與[主證據](../data/ida/fd2_ch24_oracle_parity_branch_20261004.json)。指令支援不提升整章或UI驗收等級。

2026-10-04：FILD16有限驗收與T5正常路徑返回標題的限制引用[58](58-fd2-exe-re-coverage.md)與[主證據](../data/ida/fd2_ch24_oracle_fild16_20261004.json)。CPU切片已解除阻擋，完整章與UI驗收保持原等級。

2026-10-04：#150 修正Go重播及Python獨立比較器的固定來源指標假設。只有已證0x11D3B與完整viewport參數能重綁src；其他caller不同src仍未知。正常r6 seq2100兩側已發布列49一致，完整RGB仍差946px。第23章24張與第24章13／5張既有有限回歸皆通過。新晚期特效越界#151、T4狀態差異#152及seq206 RGB差異#153仍未完成；不提升整章驗收。主契約與命令見[舞台證據](../data/ida/fd2_ch24_stage_runtime_20261003.json)，唯一現況見[58](58-fd2-exe-re-coverage.md)。

2026-10-04：#151已依[原版工作配置契約](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)將shared indexed work修正為0x2A300 bytes／640-byte列距＝270列。原#33 frame9底端263合法，不裁切或改排程，預建交易與真正越界拒收保留。正常r6重播不再中止，消費56筆AI但順序仍分岔7筆；獨立65張完整報告仍failed。第23章24張與第24章13／5張有限回歸通過，完整Go19套件2306項通過、27項既有略過。#154的#32負列、#152的AI與#153的RGB仍未完成；限RUNTIME-E1，不提升整章或逐幀特效證據。唯一現況見[58](58-fd2-exe-re-coverage.md)。

2026-10-04：#152 指令7已依[亂數交錯主契約](../data/ida/fd2_command7_target_rng_20261004.json)達有限CONFORMED／RUNTIME-E1。正式玩家與AI owner逐目標resolve，命中的全部NumericMarker消耗共用亂數，超過五段HP上限仍計入；miss推進handler但不抖動。首次T3差異與T4入口狀態已修正，已完成節點序列一致。完整章未驗收，後續攻擊比較點的stage copy未完成另登錄[#155](https://github.com/wicanr2/fd2_re/issues/155)，#153／#154仍開啟；不提升傷害、存活、選目標或整章PLAYER-E2。唯一現況及完整驗證依[58](58-fd2-exe-re-coverage.md)。

2026-10-04：#155已依[stage主契約的補正](../data/ida/fd2_ch24_stage_runtime_20261003.json)關閉驗證工具的錯誤拒收。sub_24D22只旋轉0x53AFF記憶體，不撤回已完成的VGA發布；Go重播與獨立Python比較仍依最後完整copy承接相位，未知來源／參數、未發布與截斷拒收保留。正常攻擊seq1840恢復對拍，整份正常重播可完成且AI順序一致。整份影像報告仍failed，#153及#142不提升；數字、雜湊與範圍依[58](58-fd2-exe-re-coverage.md)。

2026-10-04：#153已依[stage主契約的延後圖片補正](../data/ida/fd2_ch24_stage_runtime_20261003.json)達有限CONFORMED。原版按排程記錄狀態，PNG延至VGA copy完成；Go與獨立Python現在依同一呼叫者完整發布推導圖片相位。正式繪圖流程與原版狀態不改，未知邊界／來源／參數仍拒收。正常選取全RGB及整份影像門檻通過，原版未知介面、晚期計畫與保存仍使整章failed。#142／#154保持未完成，驗證、命令、雜湊與範圍依[58](58-fd2-exe-re-coverage.md)。

2026-10-04：#156依[指令6主契約補正](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)達有限CONFORMED。正式共用owner保存首次mode3座標搬移、非零側mode4／5的0/1與2/3/4分層，以及target／九張轉場後的counter與secondary。固定#33雙目標序列及既有正常收據回歸通過；影像門檻通過不代表本次所有target影格已逐幀對拍。#154負列仍原子拒收，#157數值亂數交錯另待RE，整章#142及DP政策不變。驗證數字與限制由[58](58-fd2-exe-re-coverage.md)承載。


2026-10-04 #157 補正：指令6命中數字抖動與下個目標判定已依[主契約](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)接入正式玩家／AI路徑，達有限 CONFORMED。每個命中 marker 都消耗亂數，超過五段 HP 後仍繼續；落空及九張轉場只傳遞動畫 state。全序列預建後才准許 Draw 發布 MP／HP，完成時才提交 RNG。前述「#157另待RE」已由正常原版逐次 trace 與受控規則測試取代；章驗收與負列限制仍由[58](58-fd2-exe-re-coverage.md)承載。


2026-10-04 #158：原版分類器依[節點主契約](../data/fd2-chapter-node-comparison-contract.json)補足sub_16C57等待期的頭像巢狀返回配對。固定EXE指紋、mouth owner與portrait helper同時成立才辨識dialogue，保持既有外層owner優先與unknown拒收。正常r6 seq2254已獨立對上重製dialogue；前述「原版未知介面造成節點拒收」由此補正取代，整章計畫與保存仍未達。驗證數字與範圍見[58](58-fd2-exe-re-coverage.md)；#38像素相位、#154負列與#142章驗收不變。

2026-10-04 #159 READY：#154窄原版試驗在施法前因工具漏收await_ui:spell退出。等待清單補既有spell，main先驗整份計畫再送鍵；新分類、原版狀態與遊戲規則未改。失敗樣本與修正契約見[指令6主證據](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)的normal_probe；實測與範圍依[58](58-fd2-exe-re-coverage.md)。

2026-10-04 #159 CONFORMED：法術等待正常收據與整份計畫送鍵前檢查已通過；主契約保留115項Python回歸、1774/1774原版前綴與正常spell seq22。#154已取得真實負列與12個原版target viewport，正式compositor仍拒收。另[玩家範圍中心主證據](../data/ida/fd2_player_command6_cursor_center_20261004.json)證實#160的直接enemy中心限制過嚴，達RE-CLOSED／DRAFT；正式UI實作與#154獨立，驗證數字、未知consumer與章層級見[58](58-fd2-exe-re-coverage.md)。

2026-10-04 #160：經[玩家中心主證據](../data/ida/fd2_player_command6_cursor_center_20261004.json)的READY審查，正式指令6白名單與confirm已改用游標Cell；先驗selection field／cursor gate，再從中心建effect名單。共用field分支原已允許空名單，但白名單缺6，先前「指令6selection UI已可進target」由本次直接反例補正。既有AI、其他指令及演出交易沿用。驗證範圍見[58](58-fd2-exe-re-coverage.md)：#160有限RUNTIME-E1，正常非零側演出仍由#154原子拒收，章台帳不變。

2026-10-04 #161／#162：[#161](https://github.com/wicanr2/fd2_re/issues/161)登記指令6完整target組圖consumer缺口；[#162](https://github.com/wicanr2/fd2_re/issues/162)修正被__CHP反證的取整假設。同源隔離原型12張完整索引／RGB相同，不使用遮罩；它仍是局部prototype，沒有正式非零側confirm全程演出或PLAYER-E2。正式#154負列拒收、#160未關閉，完成數與驗證只引用[58](58-fd2-exe-re-coverage.md)。

2026-10-04 #161：已審查caller組圖接入正式LUT15、actor末幀、packed target色調／pose與持續base。Shade／Pose／Jitter只在施法開頭初始化，跨目標落空與轉場保留；display只消耗已證numeric marker，末端RNG須等於damage plan。HP marker先更新持續base，下一張才顯示新HP。全序列預建、失敗零交易及嚴格工作區guard維持。

同源固定槽探針的11張正式composer完整影格與dosgolem原版索引／RGB全0差異；第7張仍按#154負列拒收。12張全圖一致是隔離原型，未證明正式非零側全程施法或新增PLAYER-E2。#161／#160／#154保持開啟，章台帳與既有隊伍政策不變。唯一分層現況見[58](58-fd2-exe-re-coverage.md)。


### 2026-10-04 #32：正式酒店LOAD與既有傳聞返回

[酒店主契約](../data/ida/fd2_hotel_load_20261004.json)達有限CONFORMED／RUNTIME-E1。原版301F4的302BF gate先拒絕整備槽，合法槽沿原roster／metadata交易載入；1DE確認後回24A及服務2，ESC才回loaded town。正式酒店UI、既有四槽reader與canonical戰役已接入，23條傳聞返回邊與旗標驗證通過。

固定同源槽、三個停點的完整320×200索引／RGB在合法相位均0差異，不遮罩；這是相位等價，未同步原版與重製頭像／游標時間。原版普通鍵盤零章內注入，重製城鎮酒店邊使用authored opt0；不新增章PLAYER-E2。原版傳聞角色列表／內容及gate1提示畫面未在本探針比較。驗證數字、失敗分類與來源只見[58](58-fd2-exe-re-coverage.md)及主契約。


### 2026-10-04 #29：END 回復與升級回歸

[END 主契約](../data/ida/fd2_end_turn_recovery_20261004.json)經READY審查達有限CONFORMED／RUNTIME-E1。正式END／YES與自動END共用兩輪索引發布，第一輪候選Mask填C8，sample4一次；第二輪才寫HP與bit7，raw sprite恢復及Draw完成後接回合事件。缺素材、來源變動或過期計畫均拒收，演出期間不能保存中間狀態。

[完整收據](../data/ui-traces/end-recovery-20261004.json)固定原版925／926／927三張320×200，索引與RGB均0差異，不遮罩。原版是同一ch04建構槽的正常五次END；重製是正常LOAD／出戰後匯入原版raw狀態的E1夾具，合法相位列舉未同步時間，不新增章PLAYER-E2。既有正式升級owner的第七章seq1836也以目前remake通過dialogue及完整畫面比較；其餘五個節點拒收另由#164追蹤。#163 AI Mask仍開啟，唯一分層現況與驗證數字見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-04 #163 AI回復完整畫面符合規格

[單一caller狀態](../data/ida/fd2_ai_idle_recovery_20261004.json)已CONFORMED／RUNTIME-E1。正式consumer發布初始畫面、C8 Mask、mode0原圖恢復，sample4在初次Draw後，HP在第三Draw後才提交。RLE source0、透明span、raw恢復與缺資產／非法tuple零交易測試通過。

[完整收據](../data/ui-traces/ai-idle-recovery-20261004.json)來自目前dosgolem正常BIOS輸入的接受分支；同源raw E1三張完整索引／RGB均0差異，不遮罩。原版與重製HP尾端一致。合法idle0及terrain0..3相符，沒有同步時間；重製夾具只讀raw狀態，不讀原版像素。這取代前一節「#163仍開啟」所指的未修正狀態；[#163已有限結案](https://github.com/wicanr2/fd2_re/issues/163#issuecomment-5977566171)，不新增章PLAYER-E2。唯一數字與目前限制見[58](58-fd2-exe-re-coverage.md)。

## 2026-10-04 #164：第七章節點拒收補正

[回歸收據](../data/ui-traces/parity-ch07-node-regression-20261004.json)已依[節點主契約 extension164](../data/fd2-chapter-node-comparison-contract.json)驗收。目前正式 Game 重播同一固定來源，四項 gate 與完整 SAV 相符。四組 wait／mark 的來源改為唯一語意配對，1894 的原版活動 EIP 已由閉合等待 owner 獨立分類，受影響完整畫面也通過。

這取代較早 END 回歸所述的五個節點拒收。原版仍是歷史 sample-r6，沒有冒稱目前 oracle 重生；跨時序點仍未比較，不新增 PLAYER-E2。唯一數字與來源限制見[58](58-fd2-exe-re-coverage.md)。


## 2026-10-04 #52：起手底色 writer 與正式故事緩衝診斷

[主證據追加 followup](../data/ida/fd2_terrain_mode3_review_20261001.json)保留舊17點定位。以目前乾淨951cb55 dosgolem、同一建構槽及正常鍵盤前兩名單位操作重跑兩次，五個受影響抽樣PNG雜湊相同。原版0x4DF2C在step445652335從固定FDSHAP_022.bin的tile21 literal span複製byte138到0x1587D3。三個來源指標交叉核對同一資源基址；其後tile27的0x1220C實際走raw，0x4DF39只前移、保留該底色。這是起手觀測，不外推完整17點。

目前正式Game同輸入短重播的行為、節點與金幣相同。九張抽樣各自在640px內，但整章畫面gate要求12張，短探針未通過樣本數；存檔未抽樣。正常LOAD→城鎮→整備→故事→戰場的純觀察診斷，分別呼叫完整Draw與既有離屏方式，兩者進場前後沒有nativeMapWork，第一次完整合成目標index為0。第十二章正式ch11_pre binding尚無原生對白附加資料；原生故事背景與工作緩衝交接是強推論候選，未達READY。這不是GUI PLAYER-E2，正式程式未改，不用固定色、遮罩或原版像素注入。#52維持開啟，下一步補證原生故事繪圖及交接，再驗完整17點。

r1 trace達200000上限，只引用截斷前的具體row；r2 writer窗口有界且未截斷。腳本參數／引號修正僅屬驗證工具執行問題，沒有主機分析或原檔變更。


## 2026-10-04 #52：第十二章故事背景與起手交接有限驗收

正式第十二章已接回11句原生對白版面，並依已證實聚焦重繪及同一LOADCH交接工作緩衝。正常路徑起手及方向鍵第二次選取完整畫面一致，達有限RUNTIME-E1。後續移動殘差未閉合，#52仍開啟；整章及PLAYER-E2門檻不變。唯一現況、回歸與限制見[58的本節](58-fd2-exe-re-coverage.md#2026-10-04-52第十二章故事背景與起手交接有限驗收)，來源與雜湊見[主證據validation](../data/ida/fd2_terrain_mode3_review_20261001.json)。


### 2026-10-04 #52：向上移動地形有限驗收

[walk_terrain_writer.runtime_spec及validation](../data/ida/fd2_terrain_mode3_review_20261001.json)已有限CONFORMED／RUNTIME-E1，取代上一節READY現況。正式第12章正常玩家向上walk每拍合成13×9偏移地形與既有單位／前景，按原始camera判準複製viewport。正常LOAD→城鎮→整備→故事→戰場→兩次移動，有／無Draw都保留原版tile41 literal118；mode3的一般重繪不改寫它。motion0/7、缺bank、負相機、aux／parallax六種非法輸入均原子拒收。

目前951cb55f原版短探針九張均在640px內，行為／節點／交易通過；100的完整圖0差異，94的完整圖仍210px、65仍88px，均如實保留。九張未達完整章12張門檻，存檔未抽樣，不宣稱全章或六拍全部畫面一致。

歷史a9bcd621的完整來源重播155／155 AI入口、零順序分岔；兩側22987-byte存檔SHA-256仍同為6e8823cae90191bcf7813841a5e9a514f119a717be3a24762231c8a6de8af821。原17點現15個地形像素相同，3137／3152仍各一像素未閉合；其他完整畫面殘差不遮罩。1528／2030原版未知wait仍嚴格拒收，沒有冒稱新oracle完整章通過。完整Go19套件最終回歸通過，字串98項處置不變，教訓guard通過。父#52、#154與章台帳20/30不變，下一步只追後段兩點的有界writer。


### 2026-10-04：#52 後段游標重繪修正，16／17 地形點一致

現況以 [mode3 主證據](../data/ida/fd2_terrain_mode3_review_20261001.json) 的 late_attack_work_writer.cursor_helper 為準，取代前段15／17的現況數字。原版12CEA逐格消費鍵盤重繪；cameraY21→20→19時，固定FDSHAP_022的tile37 literal118先被tile82 literal116覆寫，tile27再保留。原始位元組分別位於檔案偏移0x5BAA／0xC49D，透明header在0x446D。正常LOAD診斷有無Draw均得到相同結果；截圖輔助函式原略過camera20，修正後共用FocusNativeMapCursorSteps與nativeCursorStepHUD，不注入原版像素。

正式歷史來源重播155／155 AI入口、零順序分岔；原3137整張畫面零差異，原17點現16點地形相同，只有3152仍差一像素。275張完整比較的behavior與transaction通過，1528／2030的原版unknown wait仍依既有規則拒收。22987-byte存檔SHA-256仍同為6e8823cae90191bcf7813841a5e9a514f119a717be3a24762231c8a6de8af821。Go19套件回歸通過，98筆字串處置不變。

IDA LE 28AFD..28B08與29063..290C7證實物理攻擊釋放並重新配置0x25680-byte地圖work，再進11CAC(1)。這不證明malloc初值為0，故3152不以清緩衝猜補，後段原版有界trace另記於同一主證據。此切片只列有限CONFORMED，父#52、#154及20／30章台帳維持原狀。

本輪六回合原版trace已完成，106121筆未封頂，控制3132／3134／3149的PNG與現行951來源逐位元組相同。原版在同一0x18BEB4位址重用地圖work與戰鬥copy plane；28F4D→11EB0覆蓋目標0x19CF8B，之後仍有4E63D／2935B寫入。這排除整塊清零的修法，3152來源仍列DRAFT；詳見主證據buffer_reuse_trace。

### 2026-10-04 #167：透明底色的工具政策限制

[主證據 oracle_heap_policy_correction](../data/ida/fd2_terrain_mode3_review_20261001.json)更正前節「基址重用排除空白緩衝」：目前dosgolem配置器在reuse時clear整段，原版實體malloc初值仍未知。3152保持一像素差異，列ORACLE-POLICY-LIMIT；不把工具清零接入正式remake或稱17／17通過。16點已相同的有限驗證保留，#52仍開啟。

正式Docker入口的runner.json已記錄near_heap_policy與配置器／caller雙來源hash；未知來源標unknown，原版配置器parity標unverified。工具驗證通過，未修改引擎、dosgolem配置器、章門檻或#154政策。唯一數字、來源及目前狀態見[58](58-fd2-exe-re-coverage.md)。#166固定BG／TAI缺陷獨立處理。

### 2026-10-04 #166：正常物理背景與台座

| 介面範圍 | 分層 | 正式consumer與剩餘驗收 |
|---|---|---|
| 正常物理BG／TAI來源與座標 | RUNTIME-E1，選擇有限CONFORMED | 玩家、一般AI與mode11持有raw／mutable terrain場景快照，反擊沿用；完整影格、雙BG滑入與work續接仍在#166。[主契約](../data/ida/fd2_physical_background_selection_20261004.json)及[58現況](58-fd2-exe-re-coverage.md)。 |

本輪沒有新增整章PLAYER-E2，也沒有以局部素材測試取代完整物理演出對拍。

### 2026-10-04 #166：雙向前導有限對拍

| 介面範圍 | 分層 | 已驗與剩餘 |
|---|---|---|
| 零header物理panel、BG與29164前導 | 有限CONFORMED、RUNTIME-E1 | 正常玩家及獨立敵方的完整末格，正式GPU九次Draw與DAC；同輸入合成與GPU診斷，不外推完整攻擊或整章E2。主證據與唯一現況見[58](58-fd2-exe-re-coverage.md)及[主契約](../data/ida/fd2_physical_background_selection_20261004.json)。 |
| 非零header與戰後work | 尚待驗收 | 原版獨立敵方來源已觀察；雙BGscroll、完整counter影格、work續接仍在#166。 |

### 2026-10-04 #166：非零旗標首次轉場有限對拍

| 介面範圍 | 分層 | 已驗與剩餘 |
|---|---|---|
| 非零旗標首次前導、攻方departure、雙BG捲動 | 有限CONFORMED、RUNTIME-E1 | 正式consumer已接線；raw+6零的完整indexed／RGB及GPU prefix通過，來源見[主契約](../data/ida/fd2_physical_background_selection_20261004.json)，唯一數字見[58](58-fd2-exe-re-coverage.md)。 |
| 完整物理演出及戰後work | 尚待驗收 | 相反raw-side非零影格、2939D尾段、連擊、counter與工作緩衝續接仍在#166；沒有新增整章PLAYER-E2。 |

本節取代前節「非零header只觀察來源，首次scroll仍待實作」的現況，不擴張有限收據的範圍。


### 2026-10-05 #166：原生逐揮尾段有限對拍

| 介面範圍 | 分層 | 已驗與剩餘 |
|---|---|---|
| 非零header的單次MISS完整場景 | 有限CONFORMED、RUNTIME-E1 | 前導到逐揮尾段的完整indexed／RGB及正式GPU owner通過。僅同輸入合成診斷，來源與唯一數字見[58](58-fd2-exe-re-coverage.md)及[主契約](../data/ida/fd2_physical_background_selection_20261004.json)。 |
| 命中、多揮、counter、DAC0與音效 | RUNTIME-E1，原版oracle待補 | 正式owner與缺銀行原子拒收已測；供值fixture不能提升為原版影格驗收。 |
| 戰鬥到地圖work | 尚待驗收 | #167配置器工具政策限制保持，完整章unknown wait與影像gate仍拒收。 |

本節取代先前完整2939D尾段未接入的現況。早期表格的固定全屏戰鬥／raw owner未實作描述是當時狀態，現在由本節與58判定。新圖[有限比較](../figures/physical-tail-scoped-compare.png)保留完整畫布；不新增章PLAYER-E2。


### 2026-10-05 #166：零旗標主攻與counter完整畫面

| 介面範圍 | 分層 | 已驗與剩餘 |
|---|---|---|
| 零header單次主攻命中／counter未命中及最後恢復 | 有限CONFORMED、RUNTIME-E1 | 固定兩個phase邊界的完整indexed／RGB及正式GPU通過；來源與唯一數字見[58](58-fd2-exe-re-coverage.md)及[主契約](../data/ida/fd2_physical_background_selection_20261004.json)。 |
| 非零相反raw側、連擊、命中反擊、DAC0／音效、work | 尚待驗收 | 仍在#166，不由這次有限收據外推。 |

此項是建構槽同輸入演出診斷，保留政策加成限制；不驗收傷害、存活或敵方選目標，不新增章PLAYER-E2。本節取代前段counter全部原版影格尚缺的現況。


### 2026-10-05 #168：原生物理演出返回的 VGA 收尾

正常owner已接回原版明寫的VGA清除，時點在最後Draw完成後、地圖續行前。原版指令、回歸及目前驗證見[主契約physical_map_return_validation](../data/ida/fd2_physical_background_selection_20261004.json)與[58](58-fd2-exe-re-coverage.md)。

這是VGA寫入的有限範圍。map work配置初值、完整返回地圖圖像及#166其他演出收據仍未驗收；章門檻、#167政策與PLAYER-E2分級不變。

VGA寫入切片已有限CONFORMED，原始work與整章限制保持；相稱回歸的完整範圍見58及主契約。


### 2026-10-05 #166：弓手首擊的非零側別完整畫面

| 介面範圍 | 分層 | 已驗與剩餘 |
|---|---|---|
| raw+6=2、header1=1的正常首次射擊 | 有限CONFORMED／RUNTIME-E1 | 完整前導、射擊起手、左向捲動與逐揮末格及正式GPU通過；唯一數字與來源見[58](58-fd2-exe-re-coverage.md)及[physical_opposite_side_validation](../data/ida/fd2_physical_background_selection_20261004.json)。 |
| 非零連擊、命中反擊、DAC0／音訊、工作緩衝 | 尚待驗收 | 後續地圖原版影格保留，不由這次演出驗收外推；#166保持開啟。 |

本節取代前段「非零相反原始側別全部尚待」的現況。只驗建構槽同源演出輸入，normal_player_path_verified及原限制保持，不增加章PLAYER-E2或傷害／存活／敵方選目標聲明。


### 2026-10-05 #169：正常物理尾端停留

六刻度等待的最後畫面持有與續行時點已有限CONFORMED／RUNTIME-E1。完整GPU核對等待中的原始最後畫面，既有演出與正常章重播保持。

[主契約physical_return_wait_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)保存完整命令與限制；唯一驗證數字及目前狀態見[58](58-fd2-exe-re-coverage.md)。本節補足#168的caller等待，未驗收map work。#170現已有限CONFORMED，見後文；完整map work、章門檻與PLAYER-E2保持。


### 2026-10-05 #170：物理音效與正常返回順序

[physical_sound_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已有限CONFORMED／RUNTIME-E1。正式body以專屬PCM owner替換同通道聲音；已宣告原生map在結算前預檢，正常返回先執行普通11CAC合成，再stop、清atk與after。來源失效禁止續行；一般UI、title與nil bundle相容範圍保持。

這閉合既有map consumer的呼叫順序，未驗收重新malloc的work初值或完整返回影格。平台PCM生命周期不提升人耳、硬體波形或章PLAYER-E2。唯一驗證數字、失敗修正及目前狀態見[58](58-fd2-exe-re-coverage.md)；#166的完整map work、DAC／人耳及其餘演出仍待驗收。

### 2026-10-05 #166：敵方主攻與友軍命中反擊

[主契約physical_counter_hit_validation](../data/ida/fd2_physical_background_selection_20261004.json)以既有固定輸入重生當前oracle，零旗標主攻與命中反擊、最後恢復的完整索引像素／RGB及正式GPU已有限CONFORMED／RUNTIME-E1。前導取DAC更新後入口，攻擊與恢復畫面取複製出口；原始呼叫端固定演出／地圖界線，完整原圖保留，無遮罩或候選搜尋。

本節取代前段「命中反擊全部待驗收」的現況，僅覆蓋本次零header敵方／友軍案例。其餘非零連擊、DAC0／人耳音訊與完整地圖工作緩衝仍在#166。唯一數字、命令、同源診斷限制及目前狀態見[58](58-fd2-exe-re-coverage.md)，不新增PLAYER-E2。

### 2026-10-05 #171：返回地圖來源可用

逐格PNG現在可取得同時點的完整單位原始列，工具已限定CONFORMED。原始11D3B返回影格的selector仍為1，下一停點已為0，不得混用。原始呼叫端與完整畫面保持；[主契約](../data/ida/fd2_physical_background_selection_20261004.json)及[58](58-fd2-exe-re-coverage.md)保存來源、命令及唯一驗證數字。

#166完整返回地圖仍待正式consumer比較。本項只補觀測來源，不新增RUNTIME-E1地圖或PLAYER-E2。

### 2026-10-05 #172：物理返回保留steady色盤

正常物理返回不再錯走11CAC(0)的DAC週期。固定時鐘驗收同時確認地圖／單位仍合成，普通重繪仍推進色盤，候選預檢與正式返回使用同一入口。已限定CONFORMED／RUNTIME-E1，來源與唯一數字見[主契約](../data/ida/fd2_physical_background_selection_20261004.json)和[58](58-fd2-exe-re-coverage.md)。

四條完整演出／GPU及正常第十二章保持；完整map work與返回圖像仍在#166，#173需補同時點週期／HUD／色盤來源。沒有新增整章PLAYER-E2。

### 2026-10-05 #173：地圖原始狀態來源

[physical_map_runtime_spec／tool_validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定工具CONFORMED。既有IDA欄位寬度先複核為RE-CLOSED／READY，再改唯一oracle；clean來源提交推送後，以原槽／seed／輸入與窗口重生，原控制、停點、trace欄位、完整PNG及metadata保持。唯一數字與來源見[58](58-fd2-exe-re-coverage.md)。

這補齊#166完整map消費端的raw globals／view／units／palette來源，未驗收重製map或PLAYER-E2。入口與copy出口BIOS已有不同，不能以凍結單一刻度替代各層clock來源，也不由output反推input。#171舊收據與#167工具政策保持。首次wrapper缺掛載與收尾計數含子案例的診斷已訂正，原日誌保留；本批沒有修改正式引擎。

### 2026-10-05 #166：完整返回地圖的限定驗收

[physical_map_return_acceptance_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1。正常LOAD／整備／出戰取得正式地圖資料，再匯入唯一11CAC入口的raw units／view／globals與獨立DAC。正式finishAttackPresentation走合成、停止與續行；原版PNG只在合成後供比較，不作輸入。此節取代前段「返回地圖所有正式consumer尚未驗收」的接手狀態，範圍限本次第十二章enemy23→Ally14命中反擊的普通返回。

完整畫面與色盤通過；單位保持。BIOS單次採樣仍屬hardware-spec approximation，原版後一shift latch與重製不同，不宣稱逐時鐘一致。正式Game保留work，本例通過不證明一般配置器初值或生命週期。其餘返回分支、非零連擊與DAC0／人耳音訊保持在[#166](https://github.com/wicanr2/fd2_re/issues/166)，不新增PLAYER-E2。

DRAFT診斷與READY先於正式assertion；正式程式未修改。唯一數字、目前狀態及完整命令見[58](58-fd2-exe-re-coverage.md)與主契約；#167及章台帳保持。

### 2026-10-05 #174／#175：HUD週期與弓手返回地圖

[主契約physical_map_hud_cycle_validation／physical_own_map_return_acceptance_validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1。原版1297D先推進idle，1ACF3再讀取；正式HUD曾在推進前取得舊快照，現改讀本次候選idle。兩個caller與拒收原子性均有反例與通過收據。

弓手的選幀指標與B24來源一致。剩餘差異來自驗收工具在raw匯入後呼叫新單位constructor，把姿勢重設；現於constructor後恢復原始呈現資料並逐筆核對。正式unit blitter保持。原先拒收收據保留，#175的「正式弓手選幀尚未閉合」診斷由原版指標與測試資料流反證取代。

我方弓手與既有敵方／友軍反擊的普通返回完整畫布通過。原版弓手首次copy的DAC全黑，驗收另檢查indexed資料與整份色盤，不能單靠RGB相等。時鐘仍為hardware-spec approximation，原版配置器與一般work生命週期未證明。正常章重播的行為與存檔保持，整章節點／畫面仍按既有門檻拒收；不新增PLAYER-E2。唯一數字與命令見[58](58-fd2-exe-re-coverage.md)，父項[#166](https://github.com/wicanr2/fd2_re/issues/166)保持開啟。

### 2026-10-05 #166：主攻命中、反擊MISS的普通地圖返回

[physical_counter_miss_map_return_acceptance_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1。原版固定槽、seed與正常鍵盤輸入保持；新收據只延長擷取窗口並補同時點raw。caller區分了演出最後一次copy與普通11CAC(1)返回，原版PNG只於正式合成後比較。

完整indexed、RGB與色盤通過；單位與palette phase／tick保持，after續行。本例DAC全黑，indexed閘門另行通過。前段「此分支返回尚未驗收」由本節取代；其餘歷史失敗與原始收據保留。正式引擎未改，只新增經READY授權的固定案例wrapper。

本例Game合成前work未配置，因此仍不能驗收一般保留工作緩衝的生命週期。時鐘沿既有hardware-spec approximation；特殊分支、音訊人耳確認、#167工具政策及20/30章台帳保持，不新增PLAYER-E2。唯一數字與命令見[58](58-fd2-exe-re-coverage.md)；[#166](https://github.com/wicanr2/fd2_re/issues/166)保持開啟。

### 2026-10-05 #176：正常物理owner完整DAC與工作緩衝返回

[physical_owner_return_dac_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1。原版正常caller在最後body漸暗後重建map，停止音效，再將map漸亮才續行。正式Game重用既有palette ramp，分別使用最後body與重新合成的map索引畫布；最後Draw後才交接after。

固定ch08正常Game在進場後同步一次原版輸入，resolver只結算一次；已有work storage持續沿用，preflight未發布candidate。前置map、黑map中間時點、最終可操作map、完整DAC序列及GPU通過。舊孤立helper不能證明一般owner交接的限制，現由這條有限垂直鏈補足；原版heap重新配置的生命週期仍未宣稱已知。先前r1只驗黑map的收據保留為中間時點，不冒稱完整caller返回。

章重播的離屏helper現依正式Draw優先序，避免在map漸亮時繪製已dispose body。標準Go、既有完整物理場景／map返回、章AI順序及存檔保持。章節點／畫面仍按既有門檻拒收，時鐘保持hardware-spec approximation，不新增PLAYER-E2。唯一數字、命令與雜湊見[58](58-fd2-exe-re-coverage.md)。特殊旗標／非零連擊補驗與人耳音訊維持原限制；#167及#154未決方案不變。

本輪[#176](https://github.com/wicanr2/fd2_re/issues/176)完整DAC交接與[#166](https://github.com/wicanr2/fd2_re/issues/166)指定正常物理抽樣／work交接接受條件已滿足，遠端均已結案；提交結果另追加。這不代表整個戰役或所有戰鬥分支已驗收。


### 2026-10-05 #39 城鎮過場補齊

正式town-backed preparation的首次YES及超額選人後YES共用[城鎮出發owner](../../remake/cmd/fd2/native_town_departure.go)。
[主契約](../data/ida/fd2_town_departure_20261005.json)及[驗收入口](../../remake/cmd/fd2/native_town_departure_test.go)
已限定CONFORMED／RUNTIME-E1。定點十步、全黑、LOADCH的65步淡入與最後caller皆通過，
完整畫布、DAC與正式GPU結果引用[58現況表](58-fd2-exe-re-coverage.md)。
舊「64步」與「尚未接」不再作現況；舊輔助基準及形成過程保存。
來源phase由正常2D010暫存器同步一次，60Hz時序仍為硬體規格近似，不增加PLAYER-E2。


## 2026-10-05 #35 確認框共享相位補驗

[主證據](../data/ida/fd2_confirmation_pulse_20261005.json)與[有限驗收入口](../../remake/cmd/fd2/native_confirmation_pulse_oracle_test.go)
已限定CONFORMED／RUNTIME-E1。原版19953入口只重設53C57；53C13與53C17由確認框／17898共用，
接受／取消才把53C13歸零。選中圖格取計數器除二，沒有固定cell49的起始契約。
19B21先sign-extend BIOS低字，再以32位減53C17；差值0或1才等候，負差值也進一次。
19B51保存第二次讀值，不能與比較讀值混用。正式host-time adapter的短間距仍為硬體規格近似。

第4、5章指定四個確認框已由受版控工具補驗為0；原單相位60及完整章來源雜湊保存在主證據。
正常LOAD→提示→ESC與正式GPU通過；戰後兩點只比較排版，沒有重跑長章、隊伍、交易或存檔，
沒有增加PLAYER-E2。唯一數字與目前驗收結果引用[58](58-fd2-exe-re-coverage.md)。
第6章及其他章的舊YES差異保留為歷史觀測，不因四點補驗改寫其他章數字。


## 2026-10-05 #38 嘴型與循環色候選勘誤

[主契約](../data/ida/fd2_parity_mouth_cycle_20261005.json)與[有限候選](../../remake/cmd/fd2/chapter_parity_preparation_mouth_test.go)
已限定工具CONFORMED。seq1603的原始返回鏈證明是下一章整備確認框，舊酒店／16C57分類失效；
19953自身的初態、post-decrement與閉合重新取值沿既有IDA直接指令，不能套用16C57的倒數。

第六章指定嘴型及DAC三點已完整RGB補驗，兩次重製重播勝出PNG雜湊一致。
原版DAC循環caller為4DFCC；只使用16色完整匹配raw窗口的私有候選，不改正式時鐘或原版像素。
原始三點及章來源雜湊保留；其他章與第七章祕密商店的歷史觀測不由此次更新外推。
正式確認框缺少嘴型owner已另登記[#177](https://github.com/wicanr2/fd2_re/issues/177)。
唯一數字與完整結果引用[58](58-fd2-exe-re-coverage.md)，不新增PLAYER-E2。

## 2026-10-05 #177 正式確認等待嘴型

| 原版owner／正式消費端 | 分層 | 驗證與限制 |
|---|---|---|
| 19953／整備出發prompt | RE-CLOSED／DATA-READY／限定CONFORMED／RUNTIME-E1 | 現行dosgolem與正常Game.Update/Draw完整索引、RGBA及倒數通過；建構槽與受控實際RNG餘數如實保留。[主契約](../data/ida/fd2_confirmation_mouth_runtime_20261005.json)。 |
| END及現行共享戰場YESNO | RUNTIME-E1 | caller保存自有DATO0／3、等待／開收框、缺幀拒收與既有接受／取消／交易回歸通過；本輪未增加這些caller的動態原版對拍。 |
| 教會與未抽樣caller | 維持原有證據 | 不由共用19953推定已完成mouth接線或原版動態parity。 |

目前統計只引用[58](58-fd2-exe-re-coverage.md)，PLAYER-E2章台帳保持。


## 指令環游標角色與正式呈現勘誤（2026-10-05，#34／#179／#180／#181）

[主契約](../data/ida/fd2_parity_ring_open_20261005.json)限定第四、五章move／stay與正常LOAD第一個指令環。原版四圖示後游標角色已由正式consumer重繪；Game.Draw保留完整indexed的圖示位置，省略重複貼圖且仍標記本幀已繪。完整indexed／RGB、真正GPU、兩次重播及四個章閘門的目前統計統一見[58](58-fd2-exe-re-coverage.md)。

以下各章段落保存當時採樣的數字與來源。以前稱為「開框中途」的成因是假說，已由原版等待返回鏈與四步候選無效的結果否定；未重算的其他章數字不能當成本輪結果。第四、五章正式JSON只補30個指定點，並保留#35先前確認框驗證與其他歷史來源。沒有新增章PLAYER-E2。

## 2026-10-05 指令6限定隔離驗收（#154／#160／#161）

使用者已採用#154的限定隔離方案。正式組圖只對固定FDOTHER #32的完整具型別內容雜湊、Mode5／channel2／frame9／secondary及已證實座標保留一個前置寫入。獨立640-byte前置列不與背景、heap或存檔共用；其他負列與普通actor／target仍嚴格拒收。全序列預建及失敗零交易沿用。

#161的正式組圖與#160的正常游標確認已連成完整施法路徑，達限定CONFORMED／DATA-READY／RUNTIME-E1。正常鍵盤由標題CONTINUE進法術選單、我方中心、Escape取消，再確認施法至操作權返回；施法入口一次承接RNG3473。這取代前述「第7張仍拒收」、「12張只屬原型」及「#154待使用者決策」的現況說法，歷史證據與失敗成因保留。

固定第三方槽不增加章PLAYER-E2。目前dosgolem配置器政策仍標unknown，原版malloc與完整heap consumer未證實；雙目標轉場、演出其他階段及音訊不由12張target影格外推。唯一數字、完整命令、來源與產物雜湊見[指令6主契約](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)的isolation_validation及[58目前狀態](58-fd2-exe-re-coverage.md)。


2026-10-05 #102局部診斷：[#183](https://github.com/wicanr2/fd2_re/issues/183)的CMC CPU缺口已修正並通過有限原版helper及正式啟動前綴。原生近堆首次配置仍停止，#184保留後續CPU缺口。局部人工free list、自然LE entry與章oracle分級保留；正式_nmalloc／_nfree政策、同r3到T8與既有章PLAYER-E2不由這項結果外推。現況及來源只引用[58](58-fd2-exe-re-coverage.md)與[#102主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)。


2026-10-05 #184／#185：原始配置器六項人工helper回傳，有限CPU契約CONFORMED；自然原生配置首次回傳仍未取得，#186保留SBB缺口。取代前述AND／PUSH GS未支援的現況，保留舊收據。人工free list不作章oracle，近堆正式政策、#102同r3至T8與章分級保持。唯一現況及完整命令見[58](58-fd2-exe-re-coverage.md)與[#102主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)。


2026-10-05 #186／#187返回鏈補驗：限定SBB19與XCHG87已通過平台回歸及原始入口，
取代前述「#186保留SBB缺口」的現況。#188的POP GS平台回歸通過，
自然返回仍因未知GS selector阻塞，段映射另登記#189；不任意登錄descriptor。
原版中間非零EAX不當作完整配置回傳。正式近堆政策、#102同r3到T8及章分級保持。
唯一現況、數字及重生命令見[58](58-fd2-exe-re-coverage.md)與
[#102主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)。


2026-10-05 #188／#189：已由同源受版控自然入口找到啟動GS合法還原漏項，
原始首次caller與合法非零指標／ABI已通過，兩案結案。這取代前述首次返回阻塞現況，
歷史收據保留。GS一般位址仍未知；目前只需已設定值的保存／還原，不建立假Descriptor。
原生配置／釋放整合及#102同r3到T8尚未完成，不外推全heap或章PLAYER-E2。
唯一數字與重生命令見[58](58-fd2-exe-re-coverage.md)與
[#102主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)的allocator_first_return。


2026-10-06 #190／#191：原始近堆在完整既有平台與固定唯讀資料下，
自然配置／釋放／已釋放指標再配置已通過有界診斷。#190限定TEST契約結案，
#191保留正式oracle原生profile整合。取代前述僅首次配置已驗的現況，保留舊收據；
rootless／缺IO的診斷結果不當正式對拍。正式近堆、#102同r3到T8及章PLAYER-E2未提升。
唯一數字、命令與來源見[58](58-fd2-exe-re-coverage.md)及
[#102主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)的allocator_natural_lifecycle。


2026-10-06 #191已結案：唯一正式oracle／wrapper可明示原生近堆模式，
BOOT與正常標題方向鍵有限CONFORMED，取代前述正式整合DRAFT。
預設adapter保持，周邊沿既有硬體近似；#102同r3到T8尚未重跑，章分級不升。
來源、模式、可重跑命令與唯一數字見[58](58-fd2-exe-re-coverage.md)及
[主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)的allocator_formal_native_profile。


2026-10-06 #102／#192／#193／#194：正式 native 同 r3 到 T8，原版 T8 END
另遇 word ADD 83 缺口；#192 的CPU與此前native前綴接續通過，同r3原版計畫已走完戰後及酒店存檔。
舊 adapter 前綴與 native 的 RNG／殘值不同，父 #102 保持。
正式 Game 新來源重播的首個行為分岔是 T6 敵方換手後，登記 #194；
後續 BG 預檢拒收 #193 屬分岔後狀態，尚非原版同狀態 caller。
既有 r4 章驗收保持，不新增 PLAYER-E2。唯一現況、來源、收據與下一 gate 見
[58](58-fd2-exe-re-coverage.md)及
[主證據](../data/ida/fd2_ch18_oracle_stosb_20261003.json)的 allocator_native_chapter。


### 第十八章 #194 同來源回傳修正（2026-10-06）

0x14EF0 三者同分但達門檻時回1，正式Game原地進共用收尾。
同native完整重播的行為、節點、交易與畫面閘門已通過，酒店存檔整檔相同。
#193的BG拒收來自修正前分岔狀態，修正後未再出現；其BG保護未修改，
不推定該舊caller的原版記憶體語意已證實。既有第18章PLAYER-E2保持。
原版三score及return1的正常局部追蹤亦通過。唯一數字、來源與命令見[58](58-fd2-exe-re-coverage.md)及
[回傳主契約](../data/ida/fd2_ai_14ef0_return_20261006.json)。

### 2026-10-06 #33／#195／#196 第十七章event40與訊息分類

event40由增援後接鏡頭(17,37)及text1三句，原始ranges與既有lower完整保留。正常Game已消費對白並保持同來源四gate及酒店整檔SAV，限定CONFORMED／RUNTIME-E1，章PLAYER-E2保持。原版死亡獎勵訊息的16D05＋1ACEE由固定EXE直接指令補分類，未知來源仍拒收。測試環境補真正os與明示adapter，正式政策未改。唯一現況、數字及限制見[58](58-fd2-exe-re-coverage.md)、[event40主契約](../data/ida/fd2_turn_event40_20261006.json)及[節點契約](../data/fd2-chapter-node-comparison-contract.json)的issue195；#33整體仍未完成。

### 2026-10-06 #102 第十八章原生oracle驗收定案

使用者採用同native前綴一致、T8接續及完整Game四閘門／存檔驗收，取代舊adapter前綴相同要求。舊RNG／殘值差異保留，不宣稱兩profile一致；既有章PLAYER-E2保持。唯一證據、數字及限制見[58](58-fd2-exe-re-coverage.md)與[主契約acceptance_decision](../data/ida/fd2_ch18_oracle_stosb_20261003.json)。

### 2026-10-06 #41／#52 有界原生畫面核對

#41兩項最小owner已由READY接入，死亡最後candidate與END／獎勵最後restore後評估HUD；
未覆蓋指令、物理前後及劇情closing保持未完成。正常新章第五回合前綴的行為與完整像素已通過，
存檔未抽樣，不新增章PLAYER-E2，不用通過前綴替代owner抽樣。

#52原生來源仍有單一地形像素不同，舊adapter的0不再作現況解釋。
跨同步HP漂移、unknown UI與整體比較失敗均保留，沒有遮罩、固定色或放寬預算。
唯一數字及證據只引用[58](58-fd2-exe-re-coverage.md)、
[HUD主契約](../data/ida/fd2_hud_redraw_20261006.json)與
[地形主契約](../data/ida/fd2_terrain_mode3_review_20261001.json)。

### 2026-10-06 #197 原始byte觀察工具READY

唯一正式oracle已依012-fd2-parity-capture §10補選用的成功指令前後byte觀察。固定原版BOOT的啟用／停用完整狀態與PNG一致，平台及驅動器回歸通過。未知來源、越界、同值寫入及控制注入限制明示，不改CPU、近堆或遊戲規則；#52最後pixel writer仍未知，#197待正常收據。來源提交、唯一數字、命令與雜湊見[地形主契約](../data/ida/fd2_terrain_mode3_review_20261001.json)的memory_observation。
