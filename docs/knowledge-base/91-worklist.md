# 91 — FD2 remake 有效工作佇列與歷史工作記錄

> **文件責任（2026-08-12）**：只有下方「有效佇列」決定下一步；其後兩千多行
> 是已完成切片與歷史工作記錄，保留證據連結但不再用 `[x]` 推算完成度。
> 每一題的 RE／資料／正式執行期／E2 分層狀態，以
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)為準。
>
> 舊 `[x]` 只表示該段標題所述的有界產物曾通過當時驗證；它不等於子系統完成。
> 新工作必須使用 `RE-CLOSED`、`DATA-READY`、`RUNTIME-E1`、`PLAYER-E2` 或
> `BLOCKED` 明列關閉哪一層，不再新增沒有層級的 `[x]`。

> **2026-08-30 發行核實：** GitHub `v0.1.1` 已是正式、非草稿、非預發行版本，
> Linux AppImage、Windows ZIP、macOS DMG／tar.gz 與 `SHA256SUMS` 五個附件均已上傳。
> 原版音樂對拍片依使用者決定只留本機，不加入 Git 或公開 Release。後續發行工作
> 是 Windows／macOS 實機、簽章／公證與完整四語內容，不是重複建立 v0.1.1。

## 有效佇列（2026-08-31）

> **本輪執行順序（2026-08-30）：**依使用者指定，先完成 A5 四語正式切換，再完成
> A1 編輯器資料契約。A5 的驗收是四個官方包具有相同穩定 key／變數簽章、正式
> runtime 能原子切換並持久化 locale、對話／商店／戰鬥 HUD 具代表性最長字串通過
> 字形與邊界檢查；沒有人工審校的譯文不得冒稱正式翻譯。A1 的驗收是全戰役 legacy
> 資料可批次轉成 deterministic canonical 文件、跨檔身份與素材連結通過 validator、
> load→write→reload 不遺失資料，且至少一條正式 runtime 路徑直接消費 canonical
> 文件。兩項皆維持 RE／來源證據→規格→實作順序。

> **新工作前沿**：第一輪代表性可玩抽樣已閉合後，下一個產品目標改為
> 「可安全編輯的資料契約＋原版素材完全分離」。實檔稽核已確認現有 JSON 缺穩定
> ID／統一版本／往返契約；正式 runtime 的最新 caller 稽核則已排除直接 `.DAT`
> consumer，不能再沿用較早的相反斷言。正式規格見
> [`60-editor-separated-assets-spec.md`](60-editor-separated-assets-spec.md)。現代美術
> 只在分離 catalog 上建立獨立 theme，不覆蓋忠實原版主題。

| 順序 | 工作 | 現況 | 下一個可驗收結果 |
|---:|---|---|---|
| A1 | 編輯器 canonical schema 與穩定身份層 | `DATA-READY（全戰役 legacy projection）／RUNTIME-E1（封包 admission）`：版本化 bundle 現含1份campaign、30份scenario、35份story與38筆角色身份候選；deterministic exporter、跨文件節點／speaker validator、角色身份schema、逐檔SHA-256與完整package self-check已接。Linux AppImage與Windows ZIP重建後均帶入bundle，Linux空白cwd自檢通過。4筆名稱衝突保留直接來源並拒絕猜選 | 解決或明確拆分`native-0/1/7/96`身份歧義；建立canonical→runtime compiler，使正式戰役規則直接消費編輯後文件；再補編輯器UI修改→驗證→執行→存讀檔往返。私人297份animation metadata可顯式匯入，但不作乾淨clone必要輸入 |
| A2 | 原版素材全量分離與清冊 | `RELEASE-DATA-READY／RESEARCH-PARTIAL`：manifest v2現有39,825筆asset與1,005筆source-resource ledger；901 standardized、11個零長度confirmed-empty、0 blocked、93 unknown。完整本機包有40,127個實體檔案、約104 MB，已於私人庫保存。2026-08-30 `v0.1.1` Linux／Windows本機完整版均為42,258個檔案，macOS為42,332個檔案；三者均綁定engine head `be9a2a77`、通過manifest驗證並標示不可公開。三平台原生公開候選與雜湊也由GitHub Actions重建成功。公開README新增33張完整戰場低解析索引、96組sprite與96組portrait代表幀，不公開可重組逐檔素材。正式`Game` caller未發現93筆unknown有直接archive consumer，因此它們不阻擋第一版 | Windows／macOS實機抽測。只有新證據找到正式consumer才重開unknown，FDOTHER #47／#49不猜接 |
| A3 | runtime 移除 `.DAT` 即時讀取 | `RUNTIME-E1-PARTIAL`：FDTXT、字型、FDICON、FDSHAP、ANI、FDFIELD主要玩家路徑，以及城鎮、商店、標題、LOAD、整備、教會與戰場初始化均已遷移。19個巢狀音效bank已分離；標題#77選單音與ANI #1的#78 companion均由正式runtime消費。2026-08-29 caller稽核未發現正式`Game`仍直接讀原版archive，`FD2_ORIGINAL_FDOTHER`／`DATO` locator亦已移到測試專用檔，正式binary字串檢查通過；archive adapter只留source-oracle。BGM只從完整驗證的30份OGG catalog解析，物理攻擊亦由分離FIGANI provider原子預檢 | 以version 2 ledger的93筆unknown交叉核對是否有尚未登記的玩家consumer；已具標準資料但缺manifest bridge者只補provenance。沒有正式caller的oracle helper不再列為runtime缺口；只有新證據證明正式consumer存在才重開RE→spec→runtime切片 |
| A4 | 現代美術主題 | `PROTOTYPE-SET／DATA-CONTRACT-READY／RUNTIME-E1-PARTIAL`：索爾頭像、第一關戰場與戰鬥HUD三張style A本機私有稿均已產生；受版控`themes/modern/catalog.json`保存穩定asset ID、來源、尺寸與SHA-256。索爾獨立閉嘴頭像母稿及80×80候選已接入顯式`FD2_THEME`正式對話consumer；身分／speaker／雜湊／尺寸／不透明契約任一不符即原子拒絕。本輪勘誤`0x9017`為下框右緣錨點，實際左上角是`(8,115)`，不再沿用舊的`(87,115)`斷言；規格見`61-modern-theme-runtime-spec.md`。戰場與HUD仍是concept | 先抽測索爾上／下框正常故事路徑擷圖，再以穩定 speaker ID逐角色產生獨立頭像；後續分別以fig_068十二幀、map0 tile ID與FDOTHER HUD cells作獨立重繪基準，不能從合成概念圖反切 |
| A5 | 繁中／簡中／日文／英文與可調文字顯示 | `DATA-READY（5,176筆／語）＋RUNTIME-E1-PARTIAL`：35份劇本共1,564句具穩定 `line_id`；正式 story、handler dialog及事件61／75／76均查詢四語內容。非繁中原生 indexed 對話已重建安全分頁並依裁決固定閉嘴。FDTXT清冊現涵蓋200個可顯示物品與15個confirmed-empty ID；32名角色、94筆戰鬥raw姓名、35筆非空指令與29筆職業均有四語實體目錄，職業26／27／28保留原版占位。轉職成功與後備確認介面會在mutation前預取四語角色／職業／句型。玩家／敵方共用指令與戰鬥姓名已由共用owner重畫；高頻戰鬥提示、法術結算、寶物／擊破獎勵及教會轉交／復活也已接官方pack。商店購買成功、出售成功、裝備詢問與無可收件角色四條正常交易路徑也已移除硬編碼繁中。F4會原子同步正式grid與SpellBook相容顯示。英文／日文其餘內容仍是機器初稿，尚無正常玩家截圖 E2 | 商店服務格已證實是純圖示，不列翻譯缺口；教會服務格的亂碼紋理改列UI codec／palette缺陷，須先補原版同狀態證據。接著做四語正常玩家畫面、其餘硬編碼玩家提示與人工語文審校。4筆劇情身份歧義不與0–31隊伍顯示目錄合併；raw戰鬥姓名也不以`native_identity`替代 |

> **2026-08-31 A5 商店四語追加：**除購買／出售結果四條交易路徑外，
> 正規化商店的購買／出售標題、裝備標記、收件角色欄位與操作提示另有
> 9 個 renderer 鍵已接入四個官方語系。Go 的 `officialLocaleContract` 與 Python
> validator 同時固定 67 鍵、來源身分與變數簽章，避免 pack 比正式 runtime
> 多鍵時到啟動才失敗。

> **2026-08-30 v0.1.0 發行批次：**目前 AppImage 已由正式標題以一般鍵盤進入
> 第一關，並實際錄得 F4 依序顯示簡中、日文、英文與繁中橫幅。2026-08-30
> 勘誤版另修正空游標系統選單誤落左上角，並加入 DOSBox 原版／重製版第一關
> 12秒動態對拍，明示為相近狀態比較。80.5秒推廣片使用目前 runtime、原版保存
> 短片與已分級比較圖，不含原版音樂；H.264／AAC、
> 1280×720／30fps、雙聲道48kHz、長黑與長靜音檢查均通過。公開 Release 只含
> 不帶原版素材的三平台封包、雜湊與影片；本機 local-full 不得上傳。規格與收據見
> [`promo-v0.1.0-spec.md`](../../docs/promo-v0.1.0-spec.md)及
> [`fd2-v0.1.0-promo-20260830.json`](../data/video/fd2-v0.1.0-promo-20260830.json)。

> **2026-08-30 第一關操作勘誤：**四格選單錨點已以 map0／25／28 證明為全戰役
> 共用契約，不再固定左上角。`ch00_pre` 的四人隊伍由正式 `ACTING(0)` 同步向上
> 六格，且七拍來源時基已由 60 Hz 誤用修成約 18.2065 Hz 硬體規格近似；正常
> handler、逐格座標與時基回歸均通過，列 `RUNTIME-E1`。原版 332–338 秒已定位，
> 可靠的同段重製動態對拍仍列 READY，不把自動輸入失敗的影片當完成證據。收據見
> [`action-overlay-and-ch01-march-20260830.json`](../data/ui-traces/action-overlay-and-ch01-march-20260830.json)。

> **2026-08-29 A2／A3 追加：** event61 FDOTHER #45 的59幀演出已達
> `DATA-READY／RUNTIME-E1`；59組 indexed PNG／mask 逐幀對照原版通過，
> 正式演出與全軍移動 preflight 均只消費分離 bank。下一個 A2／A3
> 切片仍依 production caller 盤點選擇，不重做 event61 handler 語意。

> **2026-08-29 A2／A3 第二批追加：** FDOTHER #42 chapter-23 staging 已達
> `DATA-READY／RUNTIME-E1`；312×192 indexed surface／mask與原版逐像素一致，
> ch23 post 與 ch22 auxiliary reload 兩個production owner只消費分離資產，
> 缺包在發布前原子拒絕。不重做已閉合的ch23 loop／BIOS／DAC證據。

> **2026-08-29 A2／A3 第三批追加：** 戰場資訊 FDOTHER #5 entries
> `0x85..0x88` 已達 `DATA-READY／RUNTIME-E1`；四筆分離 indexed PNG 對固定原版
> 逐像素一致，正式 `sub_1B1E7` adapter 不再需要原始 `FDOTHER.DAT`。下一批仍由
> 真正 production archive caller 清冊選擇，不把 archive oracle 誤列為產品缺口。

> **2026-08-29 A2／A3 第四批追加：** 共用物品／教會／戰場短狀態面板已達
> `DATA-READY／RUNTIME-E1`；`0x17EEF／0x17FC0` 的完整 base＋data與`0x18C6D`
> value panel均只讀分離 #5 bank及portrait／text／font，四個production caller的
> FDOTHER archive path歸零。Archive adapter只留固定原版逐像素 oracle。

> **策略更新**：第一輪 remake 改以[`REMAKE-STATUS.md`](../REMAKE-STATUS.md)的
> 60格分層代表性抽樣建立95%信心。下表既有E2與精確音訊項目先登記到抽樣矩陣；
> 只有抽樣揭露的重大玩家缺陷才回到實作佇列。全戰役長程遊玩、逐週期DOS硬體、
> 無正常producer的指令及完整EXE覆蓋不列第一輪阻擋。公開發行的實體平台、簽章與
> 推廣片另行追蹤，不混入第一輪完成判定。
> 2026-08-27 已完成66秒「開場到第一關」安全母版：原版／重製開場動態、標題、
> 對話、戰場與命令格比較，加上可編輯流程、1995技術保存及跨平台引擎三段貢獻。
> 成片在 `dist/promo/`，不含原版音樂且不入Git；規格、重生腳本與雜湊見
> `docs/promo-opening-ch01-spec.md`、`tools/build_fd2_opening_promo.sh`及影片metadata。

> **抽樣清冊現況**：已登記65筆證據，60筆符合門檻（戰役／戰間12、戰鬥／AI 18、
> 介面12、存檔／持續隊伍10、終局／平台8），2026-08-28 Docker 完整檢查為
> `first_round_complete=true`、0 integrity errors。終局新增正式標題LOAD、文字
> 生命週期、`0x2c548` admission／cue、普通Space角色蒙太奇與20筆尾段排程抽樣；
> 尾段只跑到segment5，不冒稱本次已抵達永久THE END。章節0 current-runtime 的空游標命令格、
> HUD、command0調色盤與END接受逐字，以及Windows城鎮F5存檔和第30戰前冷讀持續隊伍
> 已按不同驗收不變量登錄；沒有以同一結果跨層重複計數。空游標命令格
> 已由原版與重製的普通鍵盤路徑配對；正常標題LOAD另已由普通X11鍵盤走過
> slot 0、記錄提示NO、19人整備、戰前劇情並抵達第30戰，關閉戰間層零樣本。
> 現行 Linux AppImage 另由普通 X11 鍵盤走正式標題 LOAD，從合法重製槽位抵達
> `town_ch02`，並完成 `town_ch02→rumor_ch02→town_ch02` 的酒店消息往返；這只
> 提升封包互動、戰間節點與單句對話 E1，不冒稱長程通關、音訊或實體桌面。
> 既有 Windows／Wine 普通輸入另按不同驗收不變量登錄城鎮至武器店節點轉移與
> 正式商店畫面；Wine限制與方向對映限制原樣保留。
> 晚期正常 LOAD 長鏈另登錄戰前 staging 收束後的 turn 1 玩家控制權與正式HUD；
> 兩者不重複既有第30戰敵方回合循環。
> 早期 AppImage 城鎮主畫面與第30戰空游標操作面板補足介面層12／12；後續不再
> 為湊數新增同類UI格，轉向戰鬥／AI、戰役、存檔與終局缺額。
> 存檔層另把runtime／party名冊與map view分開驗收，補登chapter0、第29戰、第30戰
> 的round／camera／cursor還原；普通標題LOAD→F5→F9再確認金錢、HP／MP、AP／DP、
> 背包、裝備與原始物品槽一致；截斷JSON槽位亦由正式標題普通輸入拒絕且零部分發布。
> 存檔／持續隊伍層已達10／10，不再新增同類格，後續轉向戰鬥／AI、戰役與終局。
> 戰鬥層已把END問句開啟、YES接受與NO取消列為三種不同交易；NO普通鍵盤回程
> 維持turn1／round1及同一視圖，沒有殘留modal或turn staging，不以同一結果重複計數。
> 滿HP草藥亦由原版／重製普通CONTINUE同座標重播：兩側都留在第三列物品面板，
> 重製HP、草藥raw slot與turn／round不變，且沒有錯誤發布目標modal。
> 第30戰固定雜湊current-runtime另由普通CONTINUE抵達受傷角色；確認第一列item143
> 與角色目標後，HP由445發布為619、acted轉為true，交易面板完整收束。此格只列
> 重製`RUNTIME-E1`；item207未被選到，不誤標為MP回復或原版精確消耗行為。
> 早期城鎮另由合法節點存檔經正式標題LOAD普通操作完成
> `town_ch02→church_ch02→town_ch02`；入口與返回旁車固定mode／selection／gold及
> transient收束。此格不外推四項教會服務、長程來源或原版E2。
> 同一起點另由普通輸入完成`town_ch02→preparation_ch02前置確認→Escape取消→town_ch02`；
> 旁車固定三態與limit15。空名冊起點不驗證選人、最終確認或進戰。
> 早期道具店另由同一合法起點經正式LOAD、原生selector選項3進
> `shop_ch02_item`，Escape返回town；它不重複武器店，也不外推購買／出售交易。
> 中期另由正式LOAD普通輸入從native variant1的`town_ch17`進
> `preparation_ch17`前置確認；取消返回未通過起點而排除，本格只計中期入口。
> 晚期則由正式LOAD普通輸入完成`town_ch26→preparation_ch26→town_ch26`；入口
> 與返回旁車固定town-backed、limit15、gold12000及暫態收束。空名冊與合法節點
> 存檔不外推第25戰長程來源、選人或進戰。
> 第27章城鎮另以普通輸入完成`town_ch27→church_ch27→town_ch27`；入口固定menu、
> selection0與gold13000，返回沒有殘留暫態。這只計教會主選單往返，不外推四項服務。
> 第26章神秘商店也已由普通Shift+F5完成：選項4＋scan0x58揭露選項5，確認後進
> `shop_ch26_secret`並以Escape返回；它是章節特定正常入口，不是debug shortcut。
> 第27章則由酒店選項0＋普通Ctrl+F6送出scan0x63，完成`shop_ch27_secret`往返；
> selection與scan皆不同，故不是重複樣本。戰役／戰間層已達12／12，不再新增同類格。
> 戰鬥層另由章節0普通CONTINUE驗證移動selection的Escape取消，以及command0目標
> modal的Escape取消；前者清除角色selection，後者返回同角色action overlay，兩者
> 都不改acted、HP／MP、turn或round，分屬不同輸入owner。
> 唯讀`battle.units`旁車再固定同一runtime index的AI前後狀態：七名on-field敵軍
> 移動、off-field index9不動、我方index0..3不變、round只由1增至2，AI完成後普通
> Return開啟第二回合操作面板。加上action overlay及item panel取消，戰鬥層達18／18。
> 下一步優先把已接正式consumer的晚期教會／商店及終局邊界改由普通輸入重播，形成
> 玩家路徑，而不是再新增無消費端RE。權威清冊見
> [`first-round-remake-samples.json`](../data/verification/first-round-remake-samples.json)。

| 順序 | 工作 | 現況 | 下一個可驗收結果 |
|---:|---|---|---|
| 0 | 原始碼註解與 Markdown 現況斷言稽核 | `DATA-READY`：已撤回「合成有效槽 fixture 證明未修改原版四槽 LOAD E2」、「轉職尚未實作」、「所有 `0x13FD4` 消費端均未接入」及把購買success/debit誤讀成sell成功E2等舊說法。2026-08-22 第五輪又修正開場 cutScript「反組譯真值／完整」誇大、天空之鑰 fixture「等價一般玩家路徑」、戰鬥動畫完整、所有繁中字必有字形及 SETSOUND 等註解；同步撤回 `42/56/57/91` 把 service3 五個 partial E2 子面板列為未做。歷史勘誤保留可追溯；這是持續性品質閘門，不是一次掃描即永久完成 | 每個玩家功能關閉時，以程式、測試與 `58` 現況核對完成度、節點、slot、handler、renderer；錯誤現況直接訂正，歷史證據追加勘誤 |
| 1 | 維護 handler 三態與 IDA 函式清冊 | `RE-CLOSED`、`DATA-READY`：2026-08-27由合法IDA 9.4重生1,305函式；全專案unknown位址足跡已成為可重跑清冊。補登已有直接證據與正式消費端的`0x2189A`／`0x24BDE`／`0x24D22`後，語意索引現67筆，結果為產品61／runtime175／未知1,069；剩餘485個精確起點、503個range命中，322筆同時有canonical與direct artifact而列第一優先。`0x36CD7`的541個prologue callers及`0x3EEDA`的160個AIL callers現可從產品原語統計排除。舊83 unknown call sites現重生為93筆已分類、0 unresolved、0 unknown | 依 `fd2_unknown_footprints.json` 審322筆第一優先候選；先做compiler／linker／extender／middleware指紋與runtime樣板分流，再審會改變玩家結果的writer／consumer，不為歸零猜名 |
| 2 | 玩家第21戰天空之鑰固定演出 | `RE-CLOSED`、`DATA-READY`、`RUNTIME-E1`：`0x2415B`的三張25-byte表與special slot已接26-slot layout；六組分支呼叫共30句保存原始control／operand／pages。成功臂正式消費26句、ACT63／64與`0x24336`；材料不足臂正式消費14句且不執行演出／授予鑰匙。兩臂均完成JOIN24／23、城鎮與存讀檔。無來源的3,231-byte紀錄已撤回，權威第21項為4,660 bytes | 只補未修改原版同狀態 E2與第一個動態調色盤相位；不重解函式本體、對話呼叫組、layout或ACT resources |
| 3 | 玩家第29戰 raw ch28 post（產品結案） | `RE-CLOSED`、`DATA-READY`、`RUNTIME-E1`；`0x1DB65`、group9→`0x25535`、持續隊伍與19人整備存讀檔均已接。全新 `Game` 冷讀後會消費ch29 pre的21句／七次staging；2026-08-26 再修正 ch30 缺最後focus view／HUD及錯誤非runtime constructor，正式 entry 現依 raw origin 去重 LOADCH party＋group0，再補 groups1–3，33筆全具 indexed presentation。相同長鏈已消費完整 END→YES、敵方回合、勝利、終局文字閘門與角色蒙太奇。外部 ch30 候選已由未修改原版同一程序以普通鍵盤完成`CONTINUE→END→YES→ENEMY PHASE`、敵方演出並交回玩家操作權；renderer同狀態合法相位達整幀AE0。fixed-hash `fd2004` 候選也由未修改原版普通 CONTINUE 進第29戰並完成一輪控制權交接；重製正式標題事件現以同檔完成 END／YES、實際敵軍行動及回合交還，並保留76筆 runtime／31人 persistent。checksum-valid `fd2021` raw chapter `0x1c` 槽則由普通 LOAD 連續走過save-NO、19人整備、戰前劇情至可操作第30戰。依玩家可見99%相似門檻，本項不再位於有效工作前沿 | remake 已完成；完整provenance、原版writer建槽、逐幀／精確音訊只列證據限制與可選polish。三份第三方存檔仍只列候選E2；高階圖像與sample 3語意仍為unknown，但均不得重開已閉合RE或阻擋交付 |
| 4 | 玩家指令／法術／物品與敵方 AI 完整交易 | 敵方ID0／4、玩家／敵方ID1／2／3／5／6／7／8／9／10／11／12、玩家／敵方IDs17–22、玩家25–27及敵方26／27均已有caller-specific indexed owner；受限class19玩家ID32–35亦達`RUNTIME-E1`。ID4現使用#22／#23／#85逐Draw發布六段HP；敵方ID0..8也已全部改依raw `+6` selector與選定目的格重建目標陣列，不再錯走玩家confirmed-cursor admission。同時修正ID4／5實檔`EffectMode=1` gate。`0x15055`也已保存並消費`0x1567E` winner的完整raw target list；正常正分item type5／13／20／21／24的數值與caller-specific indexed演出均達`RUNTIME-E1`：分別走`0x211A4`、`0x1CD17`、`0x1CAC7`，具Draw發布與完整回復邊界。玩家正常item38／79也已共用後兩個owner，不再於確認時同步跳過畫面。33圖全量mask稽核再證實，排除mode8後的正常非玩家command producer只有ID0–7、9–18、20–22、26、27，全部已有indexed owner；唯一ID30位於mode8，不進scorer。敵方mode2無候選也已接`0x13C0F→0x13FD4`恢復／零修改共用收尾。2026-08-26末關動態診斷再修正物理helper完整item table、`0x15311`原地effect destination與FIGANI 379／168嚴格按需載入；固定版原版現於同一程序從普通`CONTINUE`走完`END→YES`、敵方演出並交回玩家操作權，撤回「兩次證據不能合併」的舊限制，但第三方存檔與停用音訊仍只列候選E2。共用短音效播放器現保留疊播 voice 至自然結束、逐幀回收並在退出時關閉，既有 raw cue 不再依賴被立即丟棄的 player；這是重製播放可靠性 E1，不提升精確音訊。敵方25缺正常AI producer，維持失敗即關閉；ID4無已證實正常玩家producer；28／30／31沒有已證實正常玩家取得來源。六個raw transient的玩家可見到期文字已由FDTXT#481..486正式消費，不把缺高階enum名稱列為runtime阻擋。 | 正常producer的`RUNTIME-E1`交易已閉合，正式owner不重做；下一步只做精確音訊與代表性同狀態E2 |
| 4a | 玩家／敵方指令 `28／29／31` 原版演出 | `RE-CLOSED`（caller分歧與取得路徑窄稽核）／`RUNTIME-E1`：29玩家正式confirm已接多目標indexed owner與整批回復。28／31的固定learn table與32筆player defaults均無command bit，已知mask OR writer又只有level-up direct caller；因此「無已證實一般玩家取得來源」為強推論，不再把尋找selector列成交付阻擋，也不宣稱死碼。現有章節原始遮罩亦未找到ID29敵方producer，不猜接敵方owner | 下一個可驗收結果是command29未修改玩家同狀態逐幀／音訊E2。28／31只在動態原版出現command bit、找到新mask writer或同actor raw `+7` provenance時重開；敵方29只在出現正常producer時重開 |
| 5 | 戰場與戰間 UI 收尾 | UI-03、UI-07～12 仍 partial；故事對話現接十九個切片。`ch27_post`已由第28戰正式64-slot勝利路徑消費`FDTXT_028`五句，再同步隊伍、進`preparation_ch29`並冷讀；舊80-slot斷言已訂正為混淆60筆source與44筆已物化groups1..7。 | 十九個已接故事切片只剩未修改同狀態E2；`ch25_post`與`ch27_post`不再列為未接caller。下一個故事切片只有在canonical caller／consumer證據充分時才開啟；第29戰已結案。 |
| 6 | 原版終局精確鏈 | `RUNTIME-E1`：第27戰缺天空之鑰現在沿正式typed gate進chapter26 `0x2BCE5`前綴與`FDTXT_027`兩個原版文字閘門，不再只顯示通用結語；chapter29 `sub_2C39B`文字把caller initial portrait與14句FDTXT speaker拆開，正式19×5 indexed owner逐Draw消費六段opening、四列逐glyph、mouth、Enter／Space、五段closing與source restore，收框後才resume，不再使用一般RGBA框。正式`battle_ch30→ending`則消費chapter29來源約束前綴／角色／20段尾段並停在#59，可選隊伍最終狀態循環已接。全新`Game`冷讀最終整備槽後，JOIN順序與persistent raw `+6/+7/+8/+0x20`已連續保存到定格、回顧及返回。三筆具位址音訊cue由typed `runtime_stage`正式消費；玩家自備MT-32 OGG的`FDMUS_004→stop→FDMUS_018`已實際解碼、建立與切換播放器，但無聲Docker不證明人耳輸出。正式成功畫面已移除來源等級／按鍵說明等現代疊圖，僅除錯HUD保留；`Game.Update`與第29戰後冷讀長鏈回歸也已共用raw-change／定格／回顧的單一輸入owner。80個實際FIGANI的header-zero `0x2939D` raw `+4..+7`、base scheduler與兩次配對已接；3%外層預算依賴未初始化區域值，降為非阻擋考古限制；未達E2 | 下一步只補原版 caller `0x2C2A6` 當下完整動態狀態、精確音訊時序／原版終端輸入及第27／30戰E2。不重做重製端連續性、speaker mapping、19×5 owner、兩個文字臂、定格、回顧循環、未初始化堆疊實作殘留或三筆raw cue |
| 7 | 三平台打包與推廣片 | `RELEASE-v0.1.0`：Linux AppImage、Windows ZIP、macOS universal DMG／tar.gz 均已由原生工作流程建立、通過封包自我檢查與可攜雜湊；71.5秒推廣片重新錄製目前 AppImage 的正式標題、第一關及四語切換，影音與逐幕目視驗收通過。公開發行物不含原版EXE／DAT／存檔或本機完整素材 | 補Windows與macOS實體玩家桌面的視窗／輸入／存檔／音訊抽測；處理簽章／Gatekeeper。Wine、Docker與CI自我檢查均不取代真機 |

> **2026-08-27 全專案 Markdown 斷言稽核：** 已刪除無可重跑來源的90／100自評，
> 以合法IDA 9.4重生並完成足跡、Watcom、Miles與三筆既有handler產品語意回填後，函式清冊為產品61／runtime175／未知1,069，將故事切片由舊17筆訂正為
> 19筆，並撤回Linux ELF可外推三平台、C++／SDL2第二runtime、AI／終局舊未接狀態、
> 「全劇情轉錄即完整接線」及歷史`✅`代表完成等說法。歷史證據保留但加上取代
> 關係；現況只由`58`、`57`、本檔有效佇列及可執行抽樣清冊裁決。

> **2026-08-27 第26戰後兩分支41句原生故事對話：** 固定版`FDTXT.DAT`
> 第26項5,004 bytes已與受版控raw逐位元組核對；七個caller的41句保存control、
> operand、換行與分頁。正式`battle_ch26`勝利確認以正常57-slot入口，依
> `event_state[12]`分別消費18／33句，再完成ACTING77–80、同步、chapter26、
> `town_ch27`與全新`Game`冷讀。這同時撤回把70筆完整單位資料當唯一runtime
> frontier的舊斷言。兩臂均達`RUNTIME-E1`；未修改原版同狀態與精確音訊另列E2。

> **2026-08-27 第28戰後五句原生對話與64-slot勘誤：** `0x25464`只建立
> `FDTXT_028` index7參數後跳往共享`0x231E5`；五句`FFEE`已保存operand、分頁與
> 具型別輸入。`ch28.json`現以`runtime_append_groups`接收前置handler狀態；正式
> 戰況是20名部署者＋groups1..7共44筆＝64 slots，group255的16筆只留source roster。
> 勝利後依序完成五句、sync、chapter28、`preparation_ch29`與全新`Game`冷讀，達
> `RUNTIME-E1`。說話者逐句螢幕座標缺來源，99%模式保留五階段收框及背景還原，
> 不以玩家游標冒充額外滑動；未修改原版逐幀／音訊仍另列E2。

> **2026-08-26 service2 正式輸入補證：** 獨立裝備已由正式 service menu→角色→
> item→成功／不相容／空背包→收合→同角色名冊→menu 完整往返，達 production-input
> `RUNTIME-E1`。後續只留原版 mutation／restore 畫面與代表性章節 E2，不再重做交易。

> **2026-08-26 教會正式輸入補證：** `0x3072F` 四項服務現由正式鍵盤與回歸
> 共用 typed consumer；status／transfer／revive／class 均在關框與 source restore
> 後才發布。status 又沿正式 consumer 走完名冊→狀態→指令→同一名冊的完整面板
> 生命週期。menu dispatch 與 raw index 0 已達 `RUNTIME-E1`，後續只留 church caller
> 原版同狀態 E2。
> raw index 1 的 source／item／destination／full 也已共用正式 consumer；跨角色成功、
> 自我 remove→append、取消與滿欄零交易均達 `RUNTIME-E1`，不再把 church transfer
> production input 列為缺口。
> raw index 2 復活也已覆蓋成功、取消、不足金與最後一名後 empty→menu；深層交易
> 失敗不再誤播成功演出。此項達 production-input `RUNTIME-E1`，精確音訊與原版畫面
> 仍留 E2。
> raw index 3 轉職也完成 class list／confirm production input，並與既有 JSON 冷讀檔
> 邊界共同驗證完整角色發布。教會四項 service 的 remake 正常輸入至此均達
> `RUNTIME-E1`；後續只做代表性原版同狀態畫面／音訊 E2，不再重做 menu 或 callee。

> **2026-08-27 第23戰後整備正式輸入：** 鍵盤與測試現在共用單一 preparation
> typed consumer。正式戰果確認跑完`postbattle_ch23_persist`後，記錄提示肯定會在
> indexed關框後存檔；全新`Game`冷讀同槽，再以提示否定進15人選取、最終確認取消
> 重選及肯定，最後抵達`story_ch24`。節點、JOIN順序、HP／MP、native identity、
> raw +5／+6／class與command mask均跨冷讀保持；取消不離開節點且清空部署旗標。
> 此項達正常重製輸入`RUNTIME-E1`，未修改原版畫面／音訊E2仍另列，不重開raw ch22
> post或`0x318AD／0x31D3C`。

> **2026-08-27 第2戰前原生故事對話：** `ch01_pre`四個原始caller的20句已由
> `FDTXT_002` raw control建立逐句typed版面；`preparation_ch02`存檔後以全新
> `Game`冷讀，再用正式整備／故事輸入完成12句上框、8句下框及完整原生生命週期，
> 正常物化`battle_ch02`。資料／編譯聚焦回歸通過；同批整合測試先前聚焦執行通過，
> 本次冷快取重跑在編譯階段達外層期限而無斷言失敗。此項為`RUNTIME-E1`，不冒稱
> DOSBox同狀態`PLAYER-E2`；其餘故事caller需各自綁定，不重解通用renderer。

> **2026-08-27 序章97句原生故事對話：** `ch00_pre`十九個原始caller已由
> `FDTXT_033` 41句、`FDTXT_032` 37句與`FDTXT_001` 19句建立可重生typed版面。
> 正式具型別輸入會消費全部97句，再連續通過第1戰、戰後、`town_ch02`、
> `preparation_ch02`、第2戰前劇情與`battle_ch02`。speaker由raw slot／identity
> 導回raw `+7`；map32兩筆缺constructor來源的特殊劇情角色只在私有背景clone採
> 99%保守前景近似，不污染戰鬥或存檔。Docker／Xvfb資料、runtime與長鏈回歸均
> 實際通過，見[`ch00-pre-native-dialogue-e1.json`](../data/ui-traces/ch00-pre-native-dialogue-e1.json)。
> 此項達`RUNTIME-E1`；未修改DOSBox同狀態、精確音訊及作業系統鍵盤注入仍未提升。

> **2026-08-27 第1戰後13句原生故事對話：** `ch00_post`的`0x22F1F`已由
> `FDTXT_001` string9建立13份typed版面；binding明示12／14／18／23／27-slot
> frontiers與`story_viewport`交接。正式第1戰勝利後使用typed story input完成全部
> opening、逐字、多頁、四列窗口、嘴型、snapshot與游標尾段，再執行`sync_party`
> 並進`town_ch02`；原先直接清除`g.dialog`的測試捷徑已移除。相同長鏈續走整備、
> 全新`Game`冷讀、第2戰前20句至`battle_ch02`，見
> [`ch00-post-native-dialogue-e1.json`](../data/ui-traces/ch00-post-native-dialogue-e1.json)。
> 此項達`RUNTIME-E1`；未修改原版同狀態與精確音訊仍另列。

> **2026-08-25 最終戰前現況：** `story_ch30` 已由兩句通用後備改接正式
> `ch29_pre` binding；`0x33F78` 的 `(slot,x,y)` wrapper、`0x12CEA(x,y)` focus、
> `0x22253` story-array bridge、21句對話及七個 caller 均達窄 `RUNTIME-E1`。
> 同日正式連續回歸已由第29戰勝利走過 raw ch28 post、`preparation_ch30`
> 19人選擇與存讀檔、完整 ch29 pre binding，最後物化 `battle_ch30`；並修正三個
> 只在連續路徑暴露的 LOADCH／story view 狀態錯誤。一般玩家 E2、精確時序／音訊
> 仍歸順序3與6驗收，不再把 wrapper owner 或 generic compositor 列為待實作。

> **2026-08-26 晚期四槽 LOAD 修正：** `fd2021` slot 0 已由重製正式 selector
> 還原29人、60金幣與 raw metadata，並由來源綁定的 editable override 進
> `preparation_ch30`。舊 `raw+1` 會誤進 `preparation_ch29` 的斷言與實作已由聚焦
> 回歸取代。後續正式標題事件、19次自動前進選取、最終確認與 ch29 pre 亦已用同檔
> 抵達 `battle_ch30`；20名部署者直接來自 persistent records，不受 authored scenario
> 漏列 identity 3／19 影響。不再把這個固定槽的重製正常輸入接線列為待辦。

> **2026-08-26 晚期槽第30戰回合交接：** 同一固定雜湊槽現已由正式 LOAD、19人
> 整備、END／YES，在不清空敵軍或直接呼叫 `endTurn` 的條件下完成一輪敵軍人工智慧，
> 並交回 `PLAYER PHASE`。首輪回歸找出 persistent raw `+0x34/+0x35/+0x36` 未傳入
> scorer 的實作缺口；三欄現以原始位元組保留，不新增高階語意。此題達
> `RUNTIME-E1`，下一步只剩同 raw 狀態逐幀／音訊 E2，不再重開 AI owner。

### 阻擋完整 remake 的剩餘工作

> **2026-08-25 終局3%外層勘誤：**後段歷史項目仍出現的「3% RNG重播待接」已失效。
> IDA／Capstone 已證實終局非零分支跳過 `var_4C` 初始化，卻仍以
> `var_4C→var_44→record+0x40` 決定外層第二輪；第二次配對又在
> `[0x540FF]==1` 的最後效果幀提前返回。正式重製不模擬未初始化堆疊（stack）／HP 寫入，
> 此題降為非阻擋原版考古限制。檔首有效佇列與 `58` 取代後段舊措辭。

2026-08-25 戰場命中位移已閉合並接上E1錨點：IDA直接consumer證實
`0x5255F／0x52577`是六相位水平／垂直位移，不是idle descriptor。玩家攻擊的
第一個相位5接入`(-14,-10)`後，完整序列最佳frame76由`AE=4436`降至`AE=1330`；
修正姓名索引的`_comment`解析、真正啟用FDOTHER#4字模後再降至`AE=903`。
普通攻擊再重用既有`0x18C6D` indexed bar／digit核心，全幅降至519；519恰為舊
三欄比較圖oracle的左邊與底邊合成邊框，排除後319×199內容區`AE=0／RMSE=0`。
此固定fixture不再列renderer像素缺口；後續只需新的未修改一般玩家E2抽驗。
完整六相位因缺`0x29F72` raw owner仍只到`DATA-READY`，不得猜接；後續只補代表性
一般玩家E2。可重跑入口為`tools/docker/fd2-battle-series-compare.sh`。

依2026-08-23使用者決定，全戰役長程通關改由人工遊玩後回報問題，不再作為代理程式工作項目。現有節點與章節邊界回歸仍保留；人工回報的缺陷再建立窄重現案例。其餘阻擋交付項目如下：

1. **戰鬥交易完整性**：正常producer的物品、command與敵方AI `RUNTIME-E1` 已閉合；mode2無候選不再卡回合。command28／31無已證實一般玩家取得來源、敵方25無正常producer，均維持失敗即關閉且不列阻擋。六個raw transient已由原版到期文字與indexed owner正式消費，高階enum名稱不是runtime缺口。後續只做代表性玩家／敵方回合的精確演出、音訊與E2。
2. **存檔與持續隊伍窄回歸**：current-runtime LOAD、動態增援／JOIN 後的 persistent raw 同步，以及全新 `Game` 由最終整備冷讀到終局的完整 JOIN 時序均達窄 `RUNTIME-E1`；拓撲矛盾會在發布前原子拒絕。維持已知章節、金錢、HP／MP、物品、裝備與入隊順序的決定性邊界測試；不再把「同一 `Game` 清欄位後重載」當作冷啟動證據。長程漂移改由使用者人工回報後重現。
3. **戰場與戰間介面**：修正玩家已指出的戰場排版／動畫差異，將ch02以外的早、中、晚期城鎮、商店、教會、整備與祕密商店以正常輸入抽樣。
4. **終局與音訊**：重製端冷讀 persistent raw 到最後定格／回顧的連續性已驗證；後續只抽驗原版呼叫當下完整動態狀態、精確音訊／輸入與第27／30戰 E2。已接的蒙太奇與20組尾段不再重做；3%外層預算依賴未初始化區域值，只保留為非阻擋考古限制。
5. **發行驗收**：核心路徑關閉後，完成Linux／Windows／macOS封包、平台實機啟動／存檔／音訊抽測，最後才製作推廣影片；全戰役遊玩問題由使用者人工回報。

不阻擋交付的項目：整支 `FD2.EXE` 每個函式命名、DOS BIOS 本身、無玩家路徑的helper、以及不影響操作或可讀性的像素級微差。

> **command30取得路徑勘誤（2026-08-25）**：32筆player constructor defaults與
> 固定command-learn table都不授予ID30；已知command-mask OR writer仍只有level-up
> direct caller，因此正常玩家無已證實取得來源屬強推論，不冒稱死碼。map13 index0雖有bit30，但raw AI
> mode低四位是8；原版`0x13A9A`直接分派`0x1317D`，不進command scorer或
> `0x15311`。它也不是正常敵方command30 producer，故不列玩家／AI交付阻擋；
> 動態出現新mask writer或未修改正常路徑時才重開正式UI。

> **2026-08-22 current-runtime SAVE 閉合：** `sub_19DF7` selector1 已達
> `RE-CLOSED`／`DATA-READY`／窄 `RUNTIME-E1`。純函式 writer 保留四個
> chapter slots、未使用 runtime 容量與其他未命名 bytes；Game 只從
> CONTINUE-projected runtime 回填有 provenance 的座標、raw flags、物品、指令與數值欄位。
> UI 只在 YES 後以同目錄暫存檔原子取代明確的 `FD2_NATIVE_SAVE`；NO、缺檔、拓撲或
> raw矛盾都保持原檔。後續批次已讓非我方增援不消耗 persistent slot，並以已證實
> constructor／identity 為新我方 JOIN 追加完整 raw record；其他拓撲變化仍拒絕混寫。
> 已通過 snapshot round-trip、raw overlay、FDTXT index、YES／NO
> 與原子寫檔聚焦回歸；依使用者指示未再執行最後一輪完整 `go test ./...`，不把此批宣稱為全套綠燈。

> **2026-08-22 command 0 閉合：** 順序4先前所述的正式消費端缺口已關閉。
> `0x26152` 的 28 幀／7 元素錯開排程、FDOTHER #18／#20、#82/sub1、
> back→target→front primitive 與七段 HP plan，現已由正式 `Game`
> 接入 `0x2A6BD` actor／target battle scene owner，達 indexed
> `RUNTIME-E1`。下一個門檻是精確音訊與同狀態逐幀／逐音訊 `PLAYER-E2`；主證據見
> [`fd2_command0_presentation_ida.txt`](../data/ida/fd2_command0_presentation_ida.txt)。

### 反組譯重開規則

已在 `58`「不要重做索引」列出的位址，只有原版雜湊不同、直接指令／跳表反證、
同狀態執行結果矛盾，或主證據其實缺少聲稱的 writer／consumer 時才能重開。
handoff 的舊說法、raw exporter 尚寫 `unknown`、或新工作階段找不到筆記，都不是理由。

---

### 2026-08-22 名稱勘誤

歷史條目中的 chapter0 `CONTINUE→Return` 空游標四格畫面應稱為**系統／行動覆蓋層**，
不是原生 command grid。原生 command grid 是選取場上單位後取得 raw command mask 的
另一條路徑；兩者不可再用同一名稱推導 action owner。

## 歷史工作記錄（不可用來計算完成度）

### 2026-08-21：玩家第29戰 event75→74 可編輯事件鏈

- `RE-CLOSED`／`DATA-READY`：IDA Pro 9.4 固定 `sub_35C79` 的 raw `+6/+8`
  branches、FDTXT_029 indices 0／1、state index17／16 writes與兩筆live-row
  activations；`sub_35C32`固定groups4..7、鏡頭(10,29)及逐次reschedule。
  map28事件格以實際31格寬重算為(15,21)；錯誤暫算(28,22)已明確撤回。
- `RUNTIME-E1`：正式成功動作owner已接event75；index1五句對話完成後才原子
  啟動event74／76。event74沿用indexed pan／白閃staging，每次只發布一個動態
  group快照，失敗不污染公開battle state。
- 決定性回歸覆蓋成功、identity mismatch、缺raw provenance、對話提交邊界、
  dynamic group preflight與既有event63非對稱回歸。當時的event76待辦已由下段
  取代；event79、post及E2仍待辦。

### 2026-08-21：玩家第29戰 event76 raw-camp2 progression

- `RE-CLOSED`／`DATA-READY`：`sub_35D60` 的state17 repeat/final branches、
  `0x13512(slot1)`、row1 reschedule、group1三筆／gate0、state21 base、event79 row，
  以及六次`0x35E5A`與indices2..6順序已寫入typed scenario。
- `RUNTIME-E1`：`completeTurn`在native round increment後、player phase前執行
  raw-camp2；repeat transaction與CONTINUE raw views保持原子，final group1／row2
  先在私有state完整preflight。indexed pulse／delay／dialogue全部結束後才恢復輸入。
- 聚焦回歸覆蓋缺slot1 raw provenance零修改、phase order、private group append，
  以及六pulse＋兩delay＋五組editable text。當時的event79待辦已由下段取代；
  post binding與E2仍待辦。

### 2026-08-21：玩家第29戰 event79 pair mutation

- `RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`：`sub_35EE6`的row2 reschedule、
  process-wide單步`0x4E893`、state21 base、`rng%3`與`(rng+1)%3`兩個
  `0x13512` targets已接正式raw-camp0 owner。
- 回歸以seed `0xFFFF`區分零擴展加一與錯誤uint16 overflow；缺第三筆group1
  raw `+5` provenance時RNG、row與units全部零修改。post binding與E2仍待辦。

### 2026-08-21：玩家第29戰 post raw prelude

- `RE-CLOSED`／`DATA-READY`：合法 IDA Pro 9.4 固定 `sub_35BBA` 清除
  slots20..tail的raw word `+0x40`，caller再把slot20 raw `+7/+8`寫成`0x7E`；
  `sub_1DB65`不是generic redraw，而有13＋6＋6次indexed呈現與raw `+3/+5` writes。
- 窄`RUNTIME-E1`：frontier validator只接受已證實producer形成的
  76／78／80／82／84／87，拒絕跳序group或多出groups2/3；raw transaction
  當時曾要求完整0x50 projection；後續證據已改為「所有消費欄位必須有 typed provenance，
  CONTINUE 若另帶完整projection則必須逐欄一致」。原資源`0x1DB65` presenter、group9與正式binding
  已由本日後續條目完成；本段僅保留當時歷史。

### 2026-08-21：raw ch28 `0x25535` indexed presenter 與斷言稽核

- `RE-CLOSED`／`DATA-READY`：固定版 caller 已證實為
  `0x22253([0x53BEB]-1,15,10,15,10)`；腳本只對 `0x25535` lower 成動態最後
  runtime slot，不把強推論 slot93 寫入資料。此段記錄的是當時狀態；map28入口
  topology及event75→74後續已由本日較新的條目取代。
- `RUNTIME-E1`：battle-state presenter 預先驗證並播放
  11＋6＋18/24＋10 段，座標只在 bridge 邊界提交；`0x35E5A` 另按
  0..63／400 ms hold／62..0 執行127次 indexed DAC 寫入。缺 asset、raw
  provenance 或 compositor 時零修改，執行中錯誤會 rollback。
- 斷言稽核修正將所有 `0x22253` 概括為「renderer 未完成」的舊說法，並區分
  已接的 `0x25535` battle-state caller、仍 blocked 的 `0x33F78` story/focus
  wrapper 與無 caller ABI 的 legacy `unit_present`。handler manifest 產生器也已
  修正重複 JSON key，新增冪等回歸。
- Docker 驗證：Go `go test ./...` 全套、Python 工具56項、918份 JSON（含重複鍵）、
  722個 Markdown 本地目標均通過；handler 為80 classified／3 unresolved／0 unknown，
  本段是 presenter 接線前的歷史快照；現況為24 active／0 blocked，玩家第29戰正式 binding 已達 E1，仍缺 E2。

### 2026-08-21：raw ch23／玩家第24戰戰後垂直切片

- `RE-CLOSED`：IDA Pro 9.4 對固定雜湊 `FD2.EXE` 證實 `sub_24C1E` 每次 inner
  loop 都先呼叫 `sub_24D22(stage)` 再 draw；`sub_11EEE` case23 只在 BIOS
  tick 改變時旋轉312-byte staging rows。三個 transient offset 的 producer
  都在自己的共同尾端清零，且不由此 handler 呼叫；舊「入口 latch／offset
  尚未知而阻擋第一幀」斷言已撤回。
- `DATA-READY`／`RUNTIME-E1`：production binding 現以 FDOTHER #42、70筆
  FDFIELD＋16筆 LOADCH runtime、240＋60次 indexed draw、ESI 0..59、兩拍
  palette gate與原子失敗回復接入 `postbattle_ch24_persist`。正常戰果確認完成
  `sync_party` 後進 `preparation_ch25`，並驗證該節點存檔／讀檔。
- `PLAYER-E2`：仍缺未修改原版同狀態逐幀／時序，以及 handler 入口的程序內
  palette phase；不得宣稱逐像素或 DOS 時鐘一致。主證據見
  [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)，規格見
  [`56-fd2-remake-sdd.md`](56-fd2-remake-sdd.md)。

### 2026-08-21：command 13–16 `0x21EB1` indexed 演出

- `RE-CLOSED`：IDA Pro 9.4／Hex-Rays 與 Docker Capstone 固定四個 wrapper 的
  `(start,step)`、sample index11、`0x21EB1` 的FDOTHER #3 LUT 9→1／3→9、
  visible-cursor中心、5 ms逐張與兩段200 ms；`0x22046` consumer沿用既有
  strict compositor，不重解 callee。
- `DATA-READY`／`RUNTIME-E1`：四組排程已資料化，玩家 command 13–16 先完整
  preflight 16張 indexed frame，全部呈現後才扣MP、回復HP、提交action並重設range。
  缺正常畫面baseline、原始LUT、palette或visible cursor時不 mutation。
- 後續批次已接 `0x1C4CC/0x1C2DA→0x1E0DB/0x1DF58` 後段與敵方 owner；本段
  原有待辦已失效，現只缺同狀態原版逐幀／逐音訊 E2。主證據見
  [`fd2_command13_21eb1_presentation_ida.txt`](../data/ida/fd2_command13_21eb1_presentation_ida.txt)。

### 2026-08-21：command 13–16 後段 indexed／數字演出

- `RE-CLOSED`：固定雜湊 FD2.EXE 的 IDA Pro 9.4 與 Capstone 共同證實
  `0x1C4CC` 的 FDOTHER #6 descriptors `0x39..0x3F`、sample12，`0x1C2DA` 的
  五組 snapshot→mask與raw `0xC0`，`0x4DDD7` 的24×24四模式write-mask consumer，
  以及 `0x21190` steady redraw→`0x1DF58` 22張數字／500 ms尾停。
- `DATA-READY`／`RUNTIME-E1`：`NativeCommandHealTailSchedule` 保存所有 raw table；
  玩家與敵方 mode 11 共用完整視覺工作但保留各自 target builder。開始前一次驗證
  #3/#5/#6、baseline、mask與所有 digit位置；缺 #5／#6 的回歸證實 MP、HP、Acted
  均不變。transaction後才重畫steady frame、建立四欄queue並播放22張，完成尾停後
  才進cleanup／AI continuation。
- `PLAYER-E2`：仍缺未修改原版同狀態逐幀／逐音訊比較；60 Hz每raw frame至少一個
  update，不宣稱2 ms逐時鐘一致。主證據見
  [`fd2_command_numeric_tail_ida.txt`](../data/ida/fd2_command_numeric_tail_ida.txt)。

### 2026-08-21：END 直接控制流程與 command 13 演出 owner

- `RE-CLOSED`：合法 IDA Pro 9.4／Hex-Rays 與 Docker Capstone 對固定雜湊
  `FD2.EXE` 共同證實 `0x16F55` direction3→FDTXT `0x1A3`→YES choice0→
  FDTXT `0x1A4`→`delay(0xC8)`→`0x1A30B`；`0x117E7` 是直接 caller。
- `RUNTIME-E1`：正式 battle 共用空游標 END 現顯示兩段原文，YES 後先等待
  十二個60 Hz幀才進 `ENEMY PHASE`。回歸固定四幀 overlay close、確認、延遲、敵方
  回合與玩家回合返回；這不是 DOS BIOS tick 逐時鐘等價。
- 2026-08-22 勘誤：上述「顯示兩段原文」是當時的泛用文字層快照；目前已由
  caller-specific indexed確認生命週期取代，現況以檔首與`57` UI-03為準。
- 2026-08-22 `RE-CLOSED`／`RUNTIME-E1`：巢狀 selector3 的`-1`已由 IDA caller
  chain閉合為main程式出口，不是回標題／城鎮；正式鍵盤路徑現於接受回覆、BGM停止、
  約200毫秒及完整收合後發布`ebiten.Termination`。取消與缺資產均不退出。
- `RE-CLOSED`／`RUNTIME-E1`：command 13–16 wrapper 與共同 indexed presentation
  owner 已定位於 `0x21AD9/0x21B99/0x2211C/0x22153→0x21B18`；command 13
  玩家 cursor→MP扣除→HP回復→action提交→range reset 已通過 production 回歸。
  畫格、調色盤、音效與同狀態 E2 尚未移植，詳見
  [`fd2_end_turn_command13_owner_ida.txt`](../data/ida/fd2_end_turn_command13_owner_ida.txt)。

### 2026-08-20：chapter0 空游標 END→YES→敵方回合正式入口（歷史快照）

- `RE-CLOSED`／`PLAYER-E2` 原版錨點沿用既有
  [`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)：
  `CONTINUE→Return→Down(END)→Return→Return(YES)`。沒有重解 `0x16f55`，也沒有
  將當時尚未閉合的三格由圖示外觀猜成玩法 owner。
- `RUNTIME-E1`：正式輸入現在只允許 direction3 開 END；完整呈現四個 action overlay
  關閉畫面後才顯示確認，YES 才進 `ENEMY PHASE`，敵方計畫耗盡後返回
  `PLAYER PHASE`。取消不改回合；這是2026-08-20的歷史狀態。2026-08-22已由
  IDA／Capstone直接證實direction2為`0x1728C`設定面板並接成E1，direction0／1仍
  失敗即關閉。
- `PLAYER-E2` 差距：原版一側已有 E2；重製端確認提示仍是文字層，尚缺同一 raw
  roster／鏡頭／游標／tick 的逐幀畫面配對，所以不可宣稱完整 END UI 或 AI parity。
- 2026-08-22 勘誤：重製端已不再使用文字層，而是接上原版資源的完整確認生命週期；
  2026-08-22 後續又完成同一存檔的問句穩定畫面配對，肖像／文字子區域差異為0；
  接受、取消、收合、精確 tick與音訊仍缺，因此整體 E2 限制不變。

### 2026-08-20：玩家第25戰後第26章祕密商店正常戰間切片

- `RE-CLOSED`／`DATA-READY`：沿用已閉合的 `0x2cd16→0x2cef7` 章節表與
  `native_secret_gate`，未重新反組譯。`town_ch26` 的原始條件是 selection4＋
  Shift+F5（BIOS scan `0x58`）；掃描碼只揭露selection5，下一次確認才進variant5。
- `RUNTIME-E1`：`TestChapter25PostMaterializesSlot70JoinsPartyAndReachesTown26SaveBoundary`
  現由玩家第25戰勝利、62→70→71、JOIN26／29及town save/load繼續走到
  `shop_ch26_secret`，使用原版城鎮／商店資源驗證隱藏項重畫、商品195／207／40、
  四幀離店與返回`town_ch26`。錯用ch02 Shift+F1（`0x54`）會拒絕；返回後selection5
  與持續隊伍仍保留。
- `PLAYER-E2`：尚缺未修改原版第25戰勝利後同狀態輸入與畫面擷取；本切片不得
  外推成23個城鎮或所有商店交易已達E2。

### 2026-08-13：玩家第21戰天空之鑰演出垂直切片

- [x] **RE-CH20-SKY-KEY-SEQUENCE-20260813**：合法 IDA Pro 9.4 與 Docker
  Capstone 對固定雜湊 `FD2.EXE` 閉合 `0x242C9→0x24336`。原版依序平移鏡頭、
  播放 `FDOTHER.DAT #34` 第0至68幀、`ANI.DAT #0`、兩次全域調色盤變換，最後
  播放第69至100幀；直接指令、資產雜湊、幾何與推論等級見
  [`fd2_ch20_sky_key_sequence_ida.txt`](../data/ida/fd2_ch20_sky_key_sequence_ida.txt)。
- [x] **RUNTIME-CH21-SKY-KEY-TOWN-SAVE-E1-20260813**：可編輯劇本只接受精確
  source／target／零參數合約，正式執行期預先驗證全部幀與 AFM 後才改變畫面。
  完整 `campaign_full.json` 回歸由玩家第21戰正式勝利確認走過六素材配方、鑄造演出、
  JOIN24／23、隊伍同步、`town_ch22` 與存檔／讀檔。map20 另固定原版
  `75→79→83→87→91` 的群組追加前沿，群組255不物化。
- [ ] **PLAYER-E2-CH21-SKY-KEY**：尚缺未修改一般玩家路徑的原版同狀態逐幀／音訊
  配對；原版 `0x4DFCC` 第一個程序內相位仍是動態狀態。重製端目前只保存相對相位，
  且相鄰 `layout_units`、ACT63／64 尚未接入，不宣稱逐像素或完整戰後演出一致。

## 目前驗收（2026-08-09）

## 本輪收束（2026-08-11）

- [x] **CAMPAIGN-CH16-TOWN17-BOUNDARY**：`battle_ch16→postbattle_ch16_persist→town_ch17`
  已由實際 `campaign_full.json` 回歸走過武器店返回、整備取消、再次整備確認，最後
  才進 `battle_ch17`。這關閉一個可編輯的戰後城鎮／商店／整備垂直切片，不代表
  30 章全部戰役或一般玩家 E2 已完成；測試為
  `TestCampaignFullChapter16BattlePreservesTownShopPreparationBoundary`。
- [x] **RE-PLAYER-CURRENT-RUNTIME-PAIRED-BASELINE**：同一份固定雜湊 `FD2.SAV` 的
  未修改原版 CONTINUE E2 與重製端普通 X11 輸入 E1 已保存為
  [`native-continue-current-runtime-remake-e2.json`](../data/ui-traces/native-continue-current-runtime-remake-e2.json)。
  原版 320×200 放大至 640×400 的比較 AE `164`、RMSE `50.2631`；重製端仍受
  `FD2_NATIVE_TITLE_TICK=0` 夾具限制，故不宣稱逐像素或重製端 E2。
- [x] **UI-ACTION-ANCHOR-NATIVE-PRESENTATION-SCALE**：`drawRing` 已依完整 native
  map frame 的呈現縮放修正 320×200→640×400 指令環座標；
  `TestNativeActionOverlayAnchorUsesPresentationScale` 通過。圖示可用性、原始
  command／spell／item presentation 與敵方回合配對仍維持失敗即關閉。
- [x] **RE-AI-ENEMY-PHASE-ORIGINAL-E2-ANCHOR**：以固定雜湊的未修改
  `FD2.EXE`／`FD2.SAV`，在一次性 Docker DOSBox 中由 `CONTINUE` 進入
  current-runtime 戰場，開啟 command grid，選擇 `END` 並以 `YES` 確認；約
  1 秒看見 `ENEMY PHASE`，約 10 秒仍在敵方回合，約 20 秒回到玩家操作狀態。
  三張 320×200 client crop、按鍵時間線與 PNG／原版雜湊見
  [`native-enemy-turn-original-e2.json`](../data/ui-traces/native-enemy-turn-original-e2.json)。
  這是原版一般玩家 E2 的回合邊界，不是重製端 parity，也不單獨證明目標選擇、
  移動評分或 command／spell／item 語意。
- [x] **RE-AI-MODE11-RAW-STAGE-TRANSACTION**：`NativeAIMode11Stage`、
  `Stages` 與 `ExecuteNativeAIMode11Transaction` 保存 IDA 證實的呼叫順序，
  並由 `NextAIPlan`／`startNativeAIMode11` 接成可執行的兩段 owner：
  `0x15311` 交給已閉合的 raw command／item executor，`0x1548E` 交給已閉合
  的 typed physical damage／FIGANI owner；`0x14121` 找不到 blocked cell 時
  進入下一項 `0x13FD4` owner。缺少 command、target、path、FIGANI 或 raw
  provenance 時停止，不把 route 猜成玩法名稱。Docker focused regression 已
  通過；一般玩家敵方回合 E2 與未知 command／spell presentation 仍另列待辦。
- [x] **RE-AI-MODE11-GAME-CONSUMER-20260811**：當時新增的遊戲層回歸後來拆為
  現行 `TestNextAIPlanMode11BuildsOrderedDirectStages`、
  `TestNativeAIMode11StagesPreserveNativeOrder`、
  `TestAIStepMode11RejectsMissingCommandPresentationWhileStationary` 與
  `TestAIStepStopsMode11WithoutVerifiedProducerTables`。Docker/Xvfb 實際由
  `NextAIPlan` 產生兩段 raw stage，經 `aiStep` 先消費 `0x15311` 指令，再沿
  continuation 消費 `0x1548E` 物理／FIGANI；缺 command book 或 item／movement
  producer 時停止且不消耗行動。這只把 mode 11 的遊戲層 owner 提升為 E1 回歸，
  不宣稱原版目標選擇、完整敵方回合或一般玩家 E2。
- [x] **RE-AI-MODE5-GAME-CONSUMER-20260811**：新增
  `TestAIStepConsumesVerifiedMode5EventPlan` 與
  `TestAIStepStopsMode5WithoutMovementProvenance`。Docker/Xvfb 實際由
  `NextAIPlan` 產生帶完整 raw event grid、field-control row 與 movement-cost
  provenance 的 mode 5 計畫，經 `aiStep` 實際移動至事件格、提交原始 event state
  `0→1`、清除 map event、寫入 raw `+0x34=7` 並完成回合；缺少原始來源仍失敗即關閉。
  `FD2_MUTE=1` 只隔離測試環境沒有可播放 AIL sample 的外部資產，不命名事件的玩法、
  物品或法術效果，也不宣稱一般玩家 E2。
- [x] **RE-AI-MODE7-GAME-CONSUMER-20260811**：新增
  `TestAIStepConsumesVerifiedMode7DestinationPlan` 與
  `TestAIStepStopsMode7WithoutMovementProvenance`。Docker/Xvfb 實際由
  `NextAIPlan` 消費 raw `+0x35/+0x36` 目的地，完成 movement-only 行走後才寫入
  raw `+0x05=1` 與 map-range provenance；缺少 movement rows 時不建立 walk／attack、
  不改寫 raw byte 或回合。這只提升 `0x32975` writer 的遊戲層 owner 為 E1，不命名
  mode 7 的高階玩法，也不宣稱原版一般玩家 E2。
- [x] **RE-AI-MULTI-ACTOR-LOOP-20260811**：新增
  `TestAIStepConsumesTwoVerifiedMode7ActorsBeforeFinishingTurn` 與
  `TestAIStepStopsTwoMode7ActorsWithoutMovementProvenance`。兩個 raw mode-7 actor
  分別沿已證實 `+0x35/+0x36` 目的地由 `NextAIPlan`→`aiStep` 依序消費；第一個
  actor 完成後才建立第二個 actor 的行走，兩者都提交 `+0x05=1`，最後只增加一次
  `Turn`。缺少 movement rows 時第一個 actor 即停止，第二個 actor、位置、raw byte
  與回合均不變。這只驗證重製端敵方回合 loop 的 E1 編排，不命名 mode 7 玩法，也不
  宣稱原版多單位目標選擇 E2。
- [x] **RE-AI-EDITABLE-SPELL-FALLBACK-CONSUMER-20260811**：原始 AI 路徑沒有建立
  計畫且沒有回傳錯誤時，`NextAIPlan` 現可依可編輯 `SpellBook`／單位 `Spells`
  產生正規化（normalized）治療、攻擊、輔助、狀態與行動術計畫；`aiStep` 以既有
  `CastArea`、注入亂數、移動動畫與共同回合提交正式消費。`TestNextAIPlanSelectsEditableHealSpellBeforePhysicalFallback`、
  `TestNextAIPlanAllyTargetsEnemyWithEditableAttackSpell`、
  `TestNextAIPlanApproachesEditableAttackSpellTarget`、
  `TestNextAIPlanUsesEditableInventorySpellMapping` 與
  `TestNextAIPlanSpellPathUsesStableCellTieBreak` 另鎖定治療優先、友軍 NPC 不誤攻我方
  （Own）陣線、背包命令映射可建立法術計畫、攻擊法術朝可施放格移動及同分可施放格的穩定選擇；
  `TestAIStepConsumesEditableHealSpellThroughProductionLoop` 與
  `TestAIStepConsumesEditableAttackSpellAndMovesIntoRange` 覆蓋治療、移動入施法距離、數值
  結算與回合完成；`TestAIStepStopsSpellWithoutRNGBeforeMutation` 確認缺亂數時 MP、HP、
  行動與回合均不變。任何原始路徑的來源錯誤仍優先失敗即關閉，這是可玩
  重製端 E1，不是原版 `0x1598A` 評分、命令格或法術演出，也不是一般玩家 E2。
- [x] **RE-AI-MODE3-9-GAME-CONSUMER-20260811**：新增
  `TestAIStepConsumesVerifiedMode3AndMode9RawTargetPlans` 與
  `TestAIStepStopsMode3AndMode9WithoutMovementProvenance`。Docker/Xvfb 實際由
  `NextAIPlan` 依 raw `+0x08` 首筆查找建立 movement-only 路徑；mode 3 只提交
  明示的 map-range write，mode 9 保留不寫入的分支，兩者都不把目標當成攻擊消費。
  缺少 movement rows 時均失敗即關閉，不改寫位置、回合或 map-range；這是 E1 raw
  consumer，不命名 `0x12C60` 的高階玩法，也不宣稱一般玩家 E2。
- [x] **RE-AI-MODE4-10-GAME-CONSUMER-20260811**：新增
  `TestAIStepConsumesVerifiedMode4AndMode10DestinationPlans` 與
  `TestAIStepStopsMode4AndMode10WithoutMovementProvenance`。Docker/Xvfb 實際由
  `NextAIPlan` 消費 raw `+0x35/+0x36` 目的地，完成 movement-only 行走並提交
  map-range write；兩者不寫入 raw `+0x05`、不建立攻擊。缺少 movement rows 時
  不改寫位置、回合或 map-range，維持失敗即關閉；不命名未證實的高階玩法。
- [x] **RE-AI-MODE0-1-8-GAME-CONSUMER-20260811**：新增
  `TestAIStepConsumesVerifiedMode0NearestFallback`、
  `TestAIStepStopsMode0WithoutMovementProvenance` 與
  `TestAIStepConsumesVerifiedMode8Completion`。Docker/Xvfb 實際驗證 mode 0 的
  raw nearest fallback movement-only owner（不寫入 map-range）及 mode 8 的共同
  行動完成分支；mode 0 缺少 movement rows 時不改寫位置、回合或 raw 狀態。
  同批另以 `TestAIStepConsumesVerifiedMode1BlockedCoordinate` 與
  `TestAIStepStopsMode1WithoutMovementProvenance` 驗證唯一 raw blocked-coordinate
  的 movement-only owner。這只關閉已證實 raw 分支的重製端 E1 消費邊界，不命名高階
  玩法，也不宣稱原版一般玩家 E2。
- [x] **CAMPAIGN-APPROXIMATE-INTERMISSION-20260811**：新增明確的
  `FD2_APPROXIMATE=1` 可玩近似模式。對尚未有正式 handler 的
  `postbattle_*` 節點，只同步已物化戰場隊伍、顯示「戰後整理」提示，等待玩家
  確認後沿 authored `next` 進入城鎮／整備；不猜 JOIN、獎勵、章節或原版分支。
  預設忠實模式仍失敗即關閉，且新增 `TestApproximatePostbattlePreservesAuthoredIntermissionBoundary`
  驗證 town／preparation 邊界與戰場狀態清除；另以 `campaign_full.json` 的
  `postbattle_ch23/24/25/29` 實際節點通過近似確認與預設失敗即關閉矩陣，確認四個
  未綁定 handler 都沿 authored `next` 進入準備／城鎮而不跳過戰間段落。這批
  unbound campaign 測試在 24 個標準 postbattle binding 全部 active 後移除；目前由
  `TestCampaignFullPostbattleBindingsUseVerifiedRawOwner` 與逐章 production-boundary
  回歸取代。這是歷史可玩近似切片，不是目前的原版 E2 證據。
- [x] **CAMPAIGN-RESULT-CONFIRM-CH01-20260811**：`TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne`
  改由實際 `battle.State.Result`→`checkResult`→`confirmBattleResult` 生產邊界
  確認戰鬥勝利，再沿已編譯戰後節點進入 `town_ch02` 與整備；不再用
  `Runner.Advance("win")` 取代玩家的結果確認。測試仍以完成的戰場 fixture 清除
  敵方與 pending groups，這只驗證生產結果消費端，不宣稱未修改 DOSBox E2。
- [x] **CAMPAIGN-CH22-RESULT-PREPARATION-SAVELOAD-20260811**：新增
  `TestChapter22BattleResultPreparationSaveLoadUsesProductionBoundaries`。以 map21、
  真實 ch22 runtime roster 與已證實 73-slot group1／2 frontier，經正式
  `confirmBattleResult` 進入 `postbattle_ch22_persist`，消費既有 `ch21_post` binding
  後抵達 `preparation_ch23`；接著只在整備節點以隔離 XDG 目錄存檔，再由
  `loadGameFromSlot` 還原節點、chapter=22、隊伍名冊／加入順序／部署狀態，並確認
  不殘留 battle array。這是重製端 E1 的戰鬥結果→戰後→整備→存檔邊界，不是未修改
  一般玩家 DOSBox E2；group 66／72、未綁定 ch23/24/25/29 與同狀態逐幀差異仍待。
- [x] **GAME-TEST-REPORT-20260811**：獨立子代理在 Docker 內實際驗證重製端完整
  Go 回歸、mode 2／mode 11、戰後城鎮／整備與 shop/preparation contract，並以
  未修改 `FD2.EXE`／固定 `FD2.SAV` 完成 DOSBox 啟動／CONTINUE／END／YES 敵方回合
  擷取；報告見 [`game-test-2026-08-11.md`](../reports/game-test-2026-08-11.md)。
  完整時間線已證實原版章節0 敵方回合 E2 可達，但尚未取得重製端同一 raw 狀態配對，
  故 mode 11 與多 actor loop 仍只列重製端 E1，不宣稱 parity。
- [x] **APPROXIMATE-POSTBATTLE-SCREENSHOT-20260811**：以實際
  `FD2_CAMPAIGN=assets/scenarios/campaign_full.json`、`FD2_CAMP_NODE=postbattle_ch23_persist`
  與 `FD2_APPROXIMATE=1` 在 Docker／Xvfb 擷取戰後整理提示
  [`postbattle-approximate-remake.png`](../figures/postbattle-approximate-remake.png)，
  SHA-256 `dfdd3248f0d653a97c95ac1ea17cb8e884436fc4a473a78f2b9bb3d5b4967abe`；這是
  可玩近似模式畫面，不提升為原版 E2。
- [x] **RE-AI-13FD4-RAW-PRESENTATION-COMMIT**：正式 runtime owner 已接上
  `0x13FD4` 的 indexed／音訊窄切片：消費 `[0x53EEC]` index `4`／loop `1`、
  `0x12D7B`、兩段 `0x1DA16`（`(2,0xFD)`→`(0,0)`）、兩段修正後的
  `0x11EB0` 312×192 copy 與三次 wait，使用既有 FDICON／palette／map
  compositor 產生三個可確認的 320×200 indexed frame，並在繪圖確認後才提交
  `+0x40` HP。缺樣本、資產、tuple、frame 或 raw record 變動即回復並停止。
  這是 E1 owner，不宣稱 index 4 的高階音效名稱、兩個 decode mode 的玩法語意、
  逐幀原版 parity 或一般玩家 E2。
- [x] **RE-PLAYER-TURN-ORIGINAL-E2-ANCHOR**：以固定雜湊的未修改
  `FD2.EXE`／`FD2.SAV` 複本，在 Docker DOSBox 沙箱中由標題、開場對話走到
  第一戰第一個我方單位的玩家指令格；保存 320×200 原版畫面、按鍵時間線、
  MD5／SHA-256 與限制於 [`native-player-turn-original.json`](../data/ui-traces/native-player-turn-original.json)
  及 [`battle-player-turn-original-dosbox.png`](../figures/battle-player-turn-original-dosbox.png)。
  這是一般玩家「玩家回合可操作」的 E2 原版錨點，不是重製端同狀態 parity，
  也不代表敵方 AI 回合或整個戰場 UI 已完成。

### 2026-08-11 進度勘誤：演出 owner 已接線，E2 仍未宣稱

本節取代本表較早「尚無 `0x15311`／`0x1548E` owner」與「`0x13FD4` 仍只有
state-only」的現況敘述；那些段落保留作時間序列證據，不再當作目前狀態。

- `remake/internal/battle/native_ai_mode11_runtime.go` 會要求完整 raw record、
  command book、item row 與物理 producer provenance，建立兩段可編輯 stage。
  `remake/cmd/fd2/native_ai_mode11_execute.go` 以 continuation 保持同一單位的
  stage 順序：`0x15311` 重用已驗證 command／item 消費端，`0x1548E` 重用 typed
  physical damage／FIGANI 演出，`0x14121` 的 blocked-search fallback 完成後才
  進 `0x13FD4`。任一路徑缺證據、目標、路徑或演出資源就停止。
- `remake/cmd/fd2/native_ai_idle_recovery.go` 是 `0x13FD4` 的正式 indexed／
  音訊 owner：先驗證 raw sample `[0x53EEC]` index `4`／loop `1`、
  `0x12D7B`、兩個 decode tuple、修正後的 `0x11EB0` ABI、FDICON selector、
  DAC palette 與 320×200 frame，再播放 `sfx[4]` 並逐 frame 等待繪圖確認；
  HP 僅在第三次確認後提交。Draw／資產／sample／record 任一失敗都回復
  `[0x51A83]` 快照並停止。
- 此批是「原始證據→可見窄切片」的 E1 完成，不是原版高階語意完成：index 4
  音色名稱、`0x1DA16` mode 的玩法名稱、完整 command／spell／item 表現、同一
  未修改原版狀態的逐幀／逐音訊 E2 仍列為待辦。

## 本輪優先工作（2026-08-10）

- [x] **RE-AI-MODE2-PHYSICAL-RUNTIME**：把合法 IDA Pro／Docker Capstone 固定的
  `0x14237` 物理候選窄契約接到 `battle.State.NextAIPlan`。執行期現在會要求
  原始 mode 2、`0x4e555` 29×20 移動表、FDFIELD 地形／組成來源、物品幾何、
  `0x1DEBE` 與 `0x14237` 評分輸入；缺任何來源就停止，不退回另一套目標選擇。
  Docker 真實測試已涵蓋選目標、路徑與缺表失敗即關閉。`0x14237` 無候選現在會
  依 `0x13C0F` caller接到既有 `0x13FD4` indexed／音訊 owner：accepted gate在
  第三個Draw後才加HP，HP已滿或raw gate拒絕則零修改完成單位；缺資產仍零交易。
  這關閉正常 mode 2 的回合卡死，不代表一般玩家 E2：
  `0x14EF0` 前置選擇與 mode 3／9／5 的窄消費端已另行接線；mode 11 的兩段
  owner 與 `0x13FD4` indexed／音訊 owner 已達 E1；精確音訊與一般玩家 E2 仍未閉合。
- [~] **NATIVE-TOWN-SECRET-GATE-MATRIX**：`campaign_full.json` 現以可編輯
  `native_secret_gate` 保存 23 個城鎮的選項／BIOS 掃描碼（scan code）／祕密商店
  目的地；`ch02` 的「精確組合鍵揭露→再次確認 selection 5」與其餘章節的差異表
  已由決定性回歸（deterministic regression）鎖定。這只證明資料與 Runner 邊界，不宣稱每章
  未修改一般玩家 DOSBox E2 輸入已完成。
- [x] **ENDING-AUDIO-WIRING（RUNTIME-E1）**：戰鬥 BGM 以原版 `0x51e63` 30-entry table、
  城鎮／商店以已證實的 `FDMUS_010` 做資料回歸；IDA 已直接證實結局事件
  `0x2c5cf→FDMUS_004`、`0x2c1ac→play_bgm(-1)`、`0x2c1f5→FDMUS_018`。
  runtime 現以 typed `runtime_stage` 在精確 `0x2c548` 消費 `FDMUS_004`；來源約束
  E1 的 party montage 成功後，`MontageTailPlayer` 只在兩筆 tail cue 與所有資產
  均通過 admission 後，依 `0x2c1ac` 停曲、`0x2c1f5` 接上 `FDMUS_018`，再依序消費
  TAI／BG／FIGANI 與 #58，完成後保持 #59。raw 曲目、參數與順序已正式接線；精確停曲、
  呼叫間隔與畫面同步仍未閉合；素材或 raw provenance 失敗才確認回到可編輯結語。
  可達的 header-byte1-zero `0x2939D` 配對迴圈已接；仍未閉合的是 call-time
  records／globals、raw terminal owner、精確音訊／輸入與一般玩家 E2。3%外層預算
  依賴未初始化區域值，不接入正式重製。證據見
  [`fd2_ending_audio_ida.txt`](../data/ida/fd2_ending_audio_ida.txt)。
- [~] **RE-AI-14EF0-RUNTIME-CONSUMER-20260810**：raw producer→`0x14EF0`
  route→command／item state-only executor 已接上；當時的兩個大型遊戲層測試後來拆為
  `TestNextAIPlanUses14EF0CommandWinnerAndRetainsRawTarget`、
  `TestNextAIPlanUses14EF0ItemWinnerAndRetainsRawTarget` 與各自的正式 consumer
  fail-closed 回歸。原測試名稱只保留於 Git 歷史，不再當目前可執行入口。
  缺 item rows 的負向測試維持失敗即關閉；mode 3／9 raw `+0x08` 查找與 mode 5
  mutable event grid／state tail 也有 Docker regression。
  mode 11 現另有 `SelectNativeAIMode11Transaction` 與 runtime stage owner，
  已把 `0x15311`／`0x1548E`／`0x14121→0x13FD4` 接到可編輯執行路徑；
  `0x13FD4` 已有 indexed／音訊消費端。mode 5 raw AIL sample 的遊戲名稱、
  未知 command／relocation、完整原版逐幀／逐音訊比對與未修改原版 E2 仍待補證。

- [x] **RE-AI-14EF0-RAW-DISPATCH**：合法 IDA Pro 9.4 與 Docker Capstone
  5.0.3 交叉固定 `0x14ef0..0x15055` 的完整 raw 尾端契約：三個 producer
  的固定順序、三個 signed score、record `+0x34 & 0x40`、actor／target
  `+0x48/+0x4a`、必要時 `0x4e516([0x53c2f])`，以及
  `0x1548e/0x15311/0x15055` 路由與共用收尾。六個 direct callers 亦已列出。
  `battle.SelectNativeAI14EF0Tail` 保存 provenance 閘門下的無副作用位址路由；
  runtime consumer 另以 `NextAIPlan` 接線，但兩者不得混稱為完整 turn/camp、
  交易或 E2。
  證據見 [`fd2_ai_14ef0_dispatch_ida.txt`](../data/ida/fd2_ai_14ef0_dispatch_ida.txt)。
- [x] **玩家第18戰／raw ch17 post 垂直切片**：已修正 raw handler 與玩家戰鬥
  編號的偏移，`ch17_post` 是玩家第18戰戰後，不是第17戰。IDA 直接指令閉合
  `0x23cd5`、sub_233C6 版面、演出56–58與 FDTXT_018 index10 跨場景分段；
  55-slot map17 runtime、JOIN21／7、`town_ch19` 及 town save/load 已在
  Docker/Xvfb E1 回歸通過。角色 7／21 的 map17 ally base 只以明示的
  `native_join_base_units.json` 強推論索引提供，未知 JOIN 仍失敗即關閉。證據見
  [`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)。
- [x] **玩家第16戰戰後垂直切片**：raw `ch15_post` 已正式綁定
  `postbattle_ch16_persist`。76-slot persistent-first topology、四條 raw 分支、
  FDTXT_016 index2／3／4的3＋5＋15句、JOIN18、`town_ch17` 與 town save/load
  均以正式輸入及 Docker/Xvfb 真實回歸通過（E1）。四路可見句數為8／8／0／15；
  證據見[`fd2_ch15_post_native_dialogue.md`](../data/ida/fd2_ch15_post_native_dialogue.md)。
- [x] **玩家第17戰／raw ch16 post 垂直切片**：已把 raw handler `0x23b5f`
  正確綁定 `postbattle_ch17_persist`，保留 roster_has(18) 的兩條分支、
  layout、PAN、ACTING50–53、FDTXT_017 index5／7／6／8的4＋3＋1＋18句、JOIN16 與
  `town_ch18`。60／61 的前置 runtime frontier 與 61／62 的戰後 frontier
  均以正式輸入在 Docker/Xvfb E1 測試中驗證，兩路可見句數為23／22，並覆蓋
  JOIN後的town save/load；未具備raw provenance的條件仍失敗即關閉。證據見
  [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)與
  [`fd2_ch16_post_native_dialogue.md`](../data/ida/fd2_ch16_post_native_dialogue.md)。
- [x] **ch23 post 原始 staging 原語（E1）**：IDA／Capstone 已證實
  `0x11eee` case 23 會在 tick 變化時間接呼叫 `0x24d22(0)`；重製端新增
  `RotateNativeCh23Rows` 與 `ApplyNativeCh23PaletteCycle`，保留 312-byte
  列旋轉、`0x60003` palette table 與位址來源；`0x10652` 分支另已固定
  `FDOTHER.DAT` #42→312×192 單幀、`0xea00` staging 配置與透明 blit，重製端以
  `DecodeNativeCh23Stage`／`BlitNativeCh23Stage` 保存此窄原語；另以兩個嚴格驗證的
  `native_ch23_loop` beats 保留第一段 30 次、第二段 12 次的完整 call-site、
  stage 值與 register-shaped arguments。2026-08-21 IDA 函式資料流勘誤證實
  每個 inner loop 都是 setter-before-draw，故入口 latch 不影響第一個呈現；
  `0x53aed`／`0x53af1`／`0x53af5` 只在四個移動／呈現 producer 內暫寫，
  返回前清零，且 `sub_24C1E` 不呼叫它們。固定版 raw seed `0x01` 不進 runtime；
  production adapter 已依 SDD 接入虛擬 BIOS tick、正式綁定
  `postbattle_ch24_persist`，並通過70＋16槽位、`preparation_ch25` 與存讀檔 E1；
  現只缺一般玩家同狀態畫面／時序 E2，不再重解入口 latch。證據見
  [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)。
- [x] **RE-CH23-STAGING-GLOBAL-XREF**：IDA data-xref／Capstone 已固定 `0x53aff`
  的 raw loader owner、`0x51a10` 的 `0x24d22` local owner、`0x539f8` 的
  `0x11eee` tick snapshot owner；`0x53aed`／`0x53af1`／`0x53af5` 的寫入
  分散於 `0x12eaa`、`0x1300d`、`0x13185`、`0x13315` 共用呈現函式。這只
  關閉 offset-state 的靜態 owner 邊界，不把欄位命名成 camera／framebuffer，
  本條只關閉 offset-state 的靜態 owner；indexed／campaign gate 已由同日正式
  adapter 另行解除，不得把本條舊語句當成仍 blocked。完整位址與輸入雜湊見
  [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)。
- [x] **ch21 post 證據勘誤**：補明 `0x1f882` 是 0–63、每步 2 ms 的原生
  palette 淡出，不是同步／等待輔助器；`postbattle_ch22_persist` 仍因
  frontier runtime trace 與 indexed 畫面狀態未閉合而失敗即關閉。證據見
  [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)。
- [x] **ch21 post layout／演出候選（E1）**：IDA 已閉合 `0x233c6` 的 16 格
  layout、special slot72、raw camera `(16,18)` 與 `0x244b6` 的參數順序；
  Docker exporter 已把 acting 65／66 轉成可編輯 frame 表，並保留
  `native_relative_cursor` provenance。候選 runtime frontier 是
  **66→72→73→79**；原版 `[0x53a45]` 實際配置 96 個 `0x50`-byte 槽，
  `[0x53beb]` 是追加 count，不是 66-slot 物理容量。由於尚未證明 slot72
  在每個宣告入口都已 materialize，編譯器以最小已 materialize frontier
  保守失敗即關閉，不產生可執行 `runtime_context`；這不是原版 buffer 越界
  判定，也不刪除 slot72。這只關閉資料消費候選，不解除
  正式戰役節點；各 frontier 的 record、indexed 畫面與 E2 仍待補足。詳見
  [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)、
  [`ch21_post_candidate.json`](../../remake/assets/cutscenes/bindings/ch21_post_candidate.json)。
- [x] **玩家第23戰／raw ch22 pre 正式戰前切片（E1）**：IDA／Capstone 已固定
  `0x336a0..0x338c0` 的 16 次停用迴圈、三段 PAN、`0x336e5→0x24618` 的
  raw push、FDTXT_023 0..4、ACTING 68..70、group1 spawn 與 focus/reset 順序。
  正式 binding 保留 map22 的 70 slots 與 group1 24 rows，並以來源位址區分
  `0x336e5` 的 Y+5 與 ch21 `0x245ce` 的 Y+3；compiler/runtime fail-closed
  回歸已通過。`loadch` 後的六個原始視圖全域已由 IDA 證實重設為零，
  `0x135dd` 只同步鏡頭／絕對游標，因此第一次 PAN 後 indexed tile 是 `(0,5)`；
  重製端以場景專用載體保存，並以 Docker＋Xvfb 實測進入 `battle_ch23`。
  這不是逐像素 E2；`postbattle_ch23_persist`、戰後城鎮／商店／整備／存檔與
  未修改一般玩家 DOSBox E2 仍待。證據見
  [`fd2_ch22_pre_view_reset_ida.txt`](../data/ida/fd2_ch22_pre_view_reset_ida.txt) 與
  [`ch22_pre.json`](../../remake/assets/cutscenes/bindings/ch22_pre.json)。
- [x] **玩家第23戰／raw ch22 post `0x2189a` 原語（E1）**：IDA／Capstone 已固定
  `0x24754` 的三個 `0x2189a` caller、十次外層迴圈、`work+0x8088`、456 stride、
  13×8 raw 場景建立、312×192 呈現與五個巢狀 call-site。三個呼叫點已在
  `ch22_post.json` 轉成 `native_2189a_loop`，compiler regression 保留
  push-shaped raw arguments。2026-08-21 補證後，typed runner 只接受
  `(slot10,15,1)`／`(slot16,30,1)`，開始前驗證 LUT0..9 與完整 indexed state，
  再依 `work+0x8088` 發布十個決定性 frame；缺 LUT0 的回歸證實不修改既有 buffer。
  三次 `0x111ba` 已另補證為 FDFIELD #69、FDSHAP #46/#47；相鄰 `0x24b4d`
  也已以13×9 staging、穩態 draw、兩張 row-shifted viewport 與30×20 ms排程
  完成 typed E1。後續又閉合 `0x10652`／FDOTHER #42 consumer、三資源 reload、
  18-slot layout 與16＋70 constructor，正式接通
  `postbattle_ch23_persist→preparation_ch24` 及存讀檔 E1。尚缺 event52 精確
  增援時序與一般玩家 E2。證據見
  [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。
- [x] **玩家第23戰／raw ch22 post 分支條件（E1）**：`0x247c6` 的
  `0x24b14(100)` 回傳方向、`0x24840` 的 `0x24bde(18)` persistent `+8`
  比較，以及 `0x248b5` 的 `[0x53bef] < 15` 已轉成巢狀 editable `if`。
  compiler／BeatRunner 只接受完整 raw inventory、persistent identity 與 raw
  round provenance；缺欄位失敗即關閉，不把它們命名成角色或一般物品欄。這只
  改善 ch22 handler CFG；正式 postbattle binding 與整備／存檔 gate 現已由
  後續86-slot回歸解除，但一般玩家 E2 尚缺。證據見
  [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)；canonical
  `diagnostics.unknown_ops` 已由歷史 10 校正為 5。
- [x] **RE-CH22-2189A-GLOBAL-XREF**：IDA data-xref 已固定 `0x2189a` 直接
  讀取 `0x53a49`／`0x53aa9`／`0x53aad`／`0x53a6d`，但這些 globals 同時由
  `0x11cac`、`0x11eee`、`0x127a9`、`0x21eb1`、`0x22046`、`0x24618` 等
  共用 caller 消費；`0x53b03` 仍屬 raw resource loader handle，
  `0x53b07`／`0x53b0b` 寫入則分散於共用呈現函式。這只關閉 ch22 工作區的
  靜態 owner 邊界，不把欄位命名成 portrait／effect；indexed 原語已另由 typed
  runner 消費，但完整 campaign 仍因相鄰未閉合尾段失敗即關閉。證據見
  [`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。
- [x] **本輪稽核基線**：以目前 `tools/audit_postbattle_binding_gates.py --json`
  當時實際稽核為24節點中23 active／1 blocked；後續 raw ch28 post 接線後現況為 **24 active／0 blocked**；
  story/cutscene 為121節點、9個獨立 script、55個 handler binding、57個
  fallback；目前處理器映射缺口是對話6、捲動畫面1，
  原生語意缺口為 0。這些數字只表示已知處理器呼叫能否映射為可編輯原語，
  不代表演出消費端、介面還原或戰役節點已完成。
  這些是覆蓋統計，不是原版完成百分比。現時只封鎖玩家第29戰；
  第25戰 `0x24df2` 的兩個 `0x112a5` 參數已依中途跳入堆疊 ABI 修正為角色26
  「聖寇拉斯」與29「亞奇梅吉」，並由62→70→71 runtime／存讀檔 E1 消費。
- [x] **RE-4DBFC-RAW-MASK-CONTRACT**：合法 IDA Pro 9.4 重新固定
  `0x4dbfc..0x4dc34` 的 `count=u8[base+0]*u8[base+2]` 與逐格
  `cell[+3]=0xff`、`cell[+2]&=0x1f`、`cell[+1]&=0x03`；`0x24a92` 只是其中一個
  呼叫者，完整函式共有34個直接呼叫點。這只關閉原始變更契約（raw mutation contract），
  不是 ch22 專屬清理器。【2026-08-21 勘誤】`State.NativeMapEventGrid` 已保存
  完整四位元組 header／cell，故原始資料已達 DATA-READY；ch22 戰後尾段的
  typed 原子重設與正式接線現已完成。這仍不可把 raw mutation 泛化成
  target-field reset。
- [x] **ch29 candidate `0x35bba(20)` raw 邊界**：IDA／Capstone 已固定
  `0x35bba` 只從 runtime index20 起清除每筆 0x50-byte record 的
  `+0x40`，再呼叫多 caller 共用的 `0x1db65`。後者讀取 raw `+0/+1/+5/+0x40`
  並進入共用 indexed 呈現／更新消費鏈；欄位與高階 renderer 語意維持未知，
  不命名成 HP／狀態，也不接 `postbattle_ch29_persist`。這只刪除「0x35bba
  完全未知」的過時斷言，未增加戰役覆蓋率。證據見
  [`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt)。
- [x] **DOSBox 原版抓圖工具校正**：`tools/docker/fd2-dosbox-screenshot.sh`
  的 `shot:` 現在擷取實際 DOSBox 用戶端視窗（client window）；Xvfb 根視窗只保留給固定座標
  就緒探測（readiness probe）。Docker 實測輸出 320×200 原版標題選單與戰場對話畫面，避免
  把根視窗左上角的局部黑畫面誤列為介面證據；另保存目前重製端執行期畫面
  [`title-remake-runtime.png`](../figures/title-remake-runtime.png)，不把原版重複圖再加入。
- [~] **BATTLE-VISUAL-GAP-CH01**：重新稽核 GitHub 上的戰場對照圖，確認
  `native-map-ch01-original-video.png` 是 320×200 原版參考。`native-map-ch01-remake-handler.png`
  （640×400）現由 Docker／Xvfb 以正式 `story_ch00_handler` 的 73 拍快速時鐘執行
  LOADCH、JOIN、SPAWN 與 battle handoff 後擷取，並唯讀掛載使用者提供的
  `FDOTHER.DAT`／`FDSHAP.DAT`／`FDICON.B24`，並以 `FD2_SHOT_DETERMINISTIC=1`
  固定動畫時鐘；同時套用 IDA 已證實的 FDFIELD b1 selector。這取代舊 b0 映射造成的敵軍友軍圖像錯誤，也排除舊
  `native-map-ch01-remake.png` 直接跳 `battle_ch01` 的單角色除錯入口假象。它與舊
  `native-map-ch01-original-video.png` 參考不是同一狀態，不能用兩張舊圖的可見差異
  推論目前渲染器仍有相同缺陷；尺寸／雜湊／可見觀察與舊圖歷史證據已保存於
  [`battle-visual-gap-ch01.json`](../data/ui-traces/battle-visual-gap-ch01.json)。
  2026-08-10 已另外用同一份 `FD2.SAV` 在 DOSBox 擷取原版 oracle，並以最近鄰縮放
  對齊重製端 handler 畫面；IDA `0x1adbf` 證實地形圖示與可選單位圖示都寫入
  `base + stride*5 + 6`，修正後內容區只剩 22 個畫布邊界差異像素。此項已關閉
  「戰場 HUD 地形圖示列位址」子門檻，但整體仍維持 `[~]`：其他章節、一般玩家
  CONTINUE、動作／指令覆蓋層、動畫時序與完整戰場 E2 尚未完成，不能宣稱全遊戲
  戰場畫面已與原版一致。首頁主比較圖為
  [`battle-field-ch01-scoped-compare-20260810.png`](../figures/battle-field-ch01-scoped-compare-20260810.png)，
  右欄只標出這 22 個左下邊界像素；舊不同狀態圖片仍只作歷史參考。主圖已在
  Docker 內重建並驗證左欄為原版、中欄為重製端最近鄰縮放，避免把重複欄位誤當成
  視覺證據；最新檔案雜湊與修正說明見 `battle-visual-gap-ch01.json`。
- [ ] **RELEASE-TRIPLE-PLATFORM-PROMO-GATE**：只有在 30 章一般玩家路徑、
  戰場／戰後／城鎮／商店／整備／存檔等畫面都取得同狀態原版 DOSBox 與重製端
  逐幀證據，且 UI 矩陣不再有未解除的視覺或流程封鎖時，才允許製作
  Windows、Linux、macOS 三平台套件與推廣影片；在閘門前不得以 E1、素材合成或
  不同狀態截圖代替完整驗收。
- [~] **下一個戰後切片**：玩家第22戰已完成 raw `ch21_post` 的 E1 production
  binding、73／79-slot runtime frontier、持久隊伍同步與 `preparation_ch23`
  邊界；玩家第25戰已完成62→70→71 runtime、JOIN26／29、`town_ch26`與存讀檔
  E1；玩家第24戰亦已完成 raw ch23 adapter、70＋16槽位、`preparation_ch25`
  與存讀檔 E1；玩家第23戰後續亦已接線，目前只剩玩家第29戰 blocked。每關都必須保留
  town／shop／整備／連戰與存檔邊界，不可只把節點接到下一場戰鬥。
- [x] **UI-05 重製端對話框執行期擷取**：以 `FD2_CAMPAIGN=1` 的可編輯序章腳本，
  在 Docker／Xvfb 實際產生 640×400 [`dialogue-remake-runtime.png`](../figures/dialogue-remake-runtime.png)。
  這是 E1 消費端證據；縮放至原版 320×200 後與
  [`ch01-dialogue-original-dosbox.png`](../figures/ch01-dialogue-original-dosbox.png)
  的 AE=60414，未解除上／右肖像位置、控制碼、排版與一般玩家 E2 缺口。
- [ ] **E2 與完整戰役**：取得未修改一般玩家路徑的 DOSBox 同狀態逐幀證據，
  並完成剩餘 UI、戰鬥 AI、對話資料化與 30 章無除錯破關鏈；目前仍不可宣稱
  《炎龍騎士團 2》重製完成。

## 歷史檢查點（2026-08-02，提交 `030f1d2`；後續回填另有日期標記）

- [x] **玩家第20戰／raw ch19 post E1垂直切片**：固定record0加選15人、
  map19 group0形成83-slot入口；round15追加group1至84並執行JOIN28，round16
  依直接跳躍略過，兩路共同JOIN25後進`town_ch21`。舊86-slot authored拓撲
  已撤回。完整證據、測試與雜湊見本檔後段及
  [`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)。
- [x] **本週驗證基線**：882個JSON可解析；戰後稽核為24節點中17 active／
  7 blocked；story/cutscene為121節點、9個獨立script、47個handler binding、
  65個inline／generic fallback；`go test ./... -count=1`與235個本地文件／圖片
  目標均通過。這些是覆蓋統計，不是完整度或原版等價百分比。
- [x] **玩家第16戰／raw ch15 post（本段舊待辦，2026-08-09 已完成）**：原本的
  candidate 四分支已升為 production binding；76-slot persistent-first、raw
  round／byte+5／word+0x42、JOIN18、`town_ch17` 與 town save/load 均有
  Docker/Xvfb E1 回歸。未修改一般玩家 DOSBox E2 仍另列於 UI-07，不把本切片
  說成完整 parity。
- [~] **其餘標準戰後節點**：玩家第22戰的 raw `ch21_post` 已接正式 E1 binding，
  只接受已追加 group1+2（73）或 group1+2+3（79）的 materialized frontier，
  缺 group2 或更早的 66／72-slot 狀態會停止；本歷史段的 blocked 清單已由
  2026-08-21 raw ch22／ch23 adapter 勘誤，目前只剩玩家第29戰。第17、18戰已由
  本文件上方的 raw `ch17_post` 切片解除。完成
  handler後仍需逐關驗證城鎮／商店／整備／存檔邊界，不可只接下一場戰鬥。
- [x] **玩家第22戰／raw ch21 post E1 垂直切片**：授權 IDA Pro 9.4 已固定
  `0x244b6`、`0x24512→0x233c6`、文字索引4/5/6、ACTING raw 65/66、
  PAN raw `(16,16)`／`(16,14)`、`0x245ce→0x24618` 與尾端
  `0x1f882→sync→chapter22`；三組 16-byte layout 表及原生 loader
  `sub_1088d→sub_10b4e→sub_10c50` 已匯出。map21 的 raw header／group
  分布與事件追加順序給出候選 runtime frontier **66→72→73→79**（強推論），
  不再把槽位數寫成完全未知。證據見
  [`fd2_ch21_post_ida.txt`](../data/ida/fd2_ch21_post_ida.txt)。layout、special
  slot72 與 acting 65/66 的可編輯轉錄已有 E1 候選；原版 buffer 是 96 槽，
  `[0x53beb]` 是追加 count；編譯器 regression 會對最小宣告的 66-slot
  已 materialize frontier 拒絕 slot72 layout，避免把候選誤當可執行 binding；
  `0x24618` 的九段 LUT 9→1、5ms／段、500ms 尾端、0..62／4ms 調色盤序列與
  固定目的地 `0xA0504` 已由 IDA／Capstone 閉合；仍缺各 frontier 的一般玩家
  runtime record、indexed 畫面狀態與正式 `postbattle_ch22_persist` binding。
  `0x24618` 的 raw 相對游標 globals
  （`0x53ab9/0x53abd`）與 ch21 呼叫點 `0x245ce` 的 Y+3 變換已由 IDA 證實；
  重製端已有帶 provenance 且依呼叫位址白名單核對偏移的 fail-closed 動態欄位橋接，因此
  不接 `town_ch23`，維持失敗即關閉。2026-08-11 新增正式
  `assets/cutscenes/bindings/ch21_post.json`，只接受 73／79-slot（layout 需要
  slot72），並將 `postbattle_ch22_persist` 接到既有 `preparation_ch23`；
  `ch22.json` 改為 runtime append groups，Docker/Xvfb 回歸實際跑過 group1+2
  與 group1+2+3、`sync_party`、chapter22 及整備節點。66／72-slot、未知
  indexed 資產或未修改一般玩家路徑仍失敗即關閉，未提升為 E2。
- [ ] **介面與完整戰役驗收**：UI-01至UI-12目前全部仍為partial；城鎮與商店
  只有部分ch02狀態達E2。仍須同狀態DOSBox／重製逐幀比較，以及無debug的30章
  一般玩家可破關鏈。不要以asset／codec、文件行數或通過的重製端單元測試代替。

## Visual parity correction（2026-07-28）

- [x] **RE/FUTURE-GROUP-CONSTRUCTOR-10C50-1B750**：合法 IDA Pro 9.4 固定
  `0x1B750..0x1B83D`、12個直接 caller、即時名冊base與四個raw destination；
  Capstone重核八格`0x40`裝備、`+0x22/+0x23` binary64 1.15＋x87朝零及
  `+0x24` HIT/EV各加15。撤回「`0x1B750`等同`0x1145A`」斷言。production
  現將table base、inventory、exact重算、placement及selector全數在私有candidate
  預檢後原子發布，來源roster失敗不改。三套件Docker回歸通過；完整0x50-byte
  identity、其他caller、phase expiry與DOSBox E2仍待。
- [x] **DOC-ASSERTION-GOVERNANCE-20260728**：repo-wide掃描66個Markdown，撤回會污染remake的現行斷言：攻略不能取代逐章handler/postbattle route；`battle_events.json`與`gen_campaign.py`只是candidate scaffold；DATO／FDICON cache slot／runtime identity不全域恆等；cutscene DSL不是33關完整oracle；historical decision freeze已標superseded。同步修正README、42/56/57目前town/shop E2範圍，並保留「悠妮」與`DATO_075=商店店員`的已核實對應。
- [x] 逐項重審 title、field/HUD、action/target、dialogue、battle、postbattle、town、shop、church、preparation、save/load、ending；當時曾提出40–45%的操作界面工程估計，後因缺少完整狀態與玩家路徑分母而撤回。現況只看 `57` 的逐列證據狀態。
- [x] README撤回將 `docs/figures/title.png`／`dialogue.png` 標成 remake runtime對照；兩張是raw decode／字型研究圖。
- [~] **UI-VIS-TOWN**：`0x2cd16/0x2cf71/0x11eb0`已閉合3個FDOTHER背景variant、#10 label、FDTXT `0x1ef+selection`、FDICON `0,1,2,1` pulse、6組variant座標與312×192→VGA `(4,4)`；23個town保存raw `native_town_variant`。ch02 variant0 [`selection0–5`](../figures/town-hub-six-selections-original-vs-remake.png) 都達原版／remake raw RGB 整幀相同；另以固定雜湊原版的修改 LOAD 副本取得 variant1與variant2，兩者正常 selection0–4 都與對應 production node 指定 pulse 逐幀 AE=0，見 [`native_town_variant1_e2.json`](../data/native_town_variant1_e2.json)、[`native_town_variant2_e2.json`](../data/native_town_variant2_e2.json) 與兩張對照圖。Left/Right wrap、Shift+F1 reveal、Enter進variant5及Escape返回selection5亦有input E2；共用 glyph shadow 與誤 reset pulse 已修。仍缺variant2 selection5 的 BIOS 掃描碼／Enter，以及未修改一般玩家城鎮路徑 E2。
- [~] **UI-VIS-SHOP**：四個callee與secret selection+BIOS-scan gate皆已接production；ch02 variant1/3/5主選單、purchase list／Yes-No／不足金／recipient selection0↔1、success與debit的既有E2維持。賣出正常商店輸入現從角色名冊、物品與Yes-No走完五個成功原子影格、`0x2D3FF`向上金幣滾動及返回原角色名冊；新增九組完整320×200 RGB AE=0，見[`shop-sell-success-return-ch02-e2.json`](../data/ui-traces/shop-sell-success-return-ch02-e2.json)。route patch與screenshot-only typed/raw bootstrap限制仍有效。當時下一gate列為四人recipient scroll、no-recipient/full、equip/transfer child panel、其他章節與未修改一般玩家路徑；此清單已由下方勘誤取代。
  **2026-08-22 service3 勘誤**：上句是取得 service2／3 原版畫面前的歷史狀態。
  service2名冊／面板與service3五個穩定子面板現已有route-patched partial E2；
  剩餘是recipient其餘分支、service2動畫相位與mutation／restore、service3
  mutation／empty／full、其他章節及未修改一般玩家路徑。
- [x] **CAMPAIGN-JOIN-LOADCH-PERSISTENT-PARTY-BOOTSTRAP**：修正正常ch00 JOIN只記`partyMembers/partyJoinOrder`、首次LOADCH未建`partyRoster`，導致帶native identity的第一個`sync_party`因找不到既有record而全數skip。`applyLoadCH`現只在已有JOIN chronology時依typed scenario補缺少record，既有進度優先，direct/debug replay不造persistent state。真實ch00 scenario/order `[0,9,4,30]` regression驗證ch02布衣候選為`[0,9,4]`且首次native-identity sync可命中；這是remake runtime bridge，不宣稱native FD2.SAV或完整campaign E2。
- [x] **UI-SHOP-STANDALONE-EQUIP-PRODUCTION**：Docker Capstone重讀`0x2f883/0x1bffe/0x17e0b/0x1b9de`，撤回「獨立裝備沿用purchase商品／收件者widget」的假設。production現走service2→兩欄角色roster→11→0完整item/status panel；相容item經`0x1c1c3→0x1c142→0x1b750`原地更新flags/能力並重畫，incompatible無發明feedback，離開0→11 restore shop再回roster。`EquipNativeCompactSlot`驗證raw occupied order與compact inventory/equipped一致，保留ignored raw hole/stale byte，divergence原子拒絕。Docker/Xvfb production regression通過；DOSBox E2仍待。
- [x] **UI-SHOP-TRANSFER-PRODUCTION**：`0x2f8ea`同時由shop service3與church raw1呼叫，不是任一場景專屬。shop production已接FDTXT512 source prompt→全party roster→FDTXT511 empty或`0x2dc55(mode1)` item list→FDTXT510→全party destination roster→FDTXT506 full或raw remove/append/recalc→512 loop。重核撤回「destination排除source」的高階假設：source本人保留為候選，未滿欄時self-transfer會把item以unequipped狀態移到尾端。`ValidateNativeInventoryProjection`與full raw-flag gate原子拒絕投影分歧；Docker/Xvfb production、empty/full/self regression通過。
- [~] **UI-SHOP-RECIPIENT-INPUT-E2**：原版已實測裝備收件者由selection0按Down到selection1、再按Up回selection0，Left/Right不改selection。production共用純`advanceNativeShopEquipmentRecipient`：bounded Up/Down、horizontal no-op、同tick Up後Down順序與`NativeThreeRowWindow`stateful origin均有直接input regression；helper-level invalid count/selection/start rejection亦有測試，production caller會在索引recipient前fail closed回purchase list。以乾淨原始SAV/TMP加三處已驗route patch重跑，`waitpixel(175,90)=101,121,121`在Down前同步人物動畫相位；0.05／0.20／0.40秒原版樣本分別與remake cycle1／cycle1／cycle0取得整幀AE=0，另兩張對上cycle2，沒有遮罩。故selection0↔1 input/E2已關閉；項目仍為partial是因四人以上scroll尚無原版E2。
- [x] **RE/UI-TOWN-SECRET-GATE**：Docker Capstone閉合`0x2cd16→0x4e4b9`與`0x2cde0..0x2cef7`：每章0x1f-byte town record `+1`必須等於目前五項selection，`+2`必須等於BIOS Shift/Ctrl/Alt-F1..F10 scan，才把selection寫5。新證據撤回「chord立即進店」：hub先重畫selection5 icon/label，後續Enter才由`0x2d093→0x2d28c`進variant5 shop。23筆已資料化為editable `native_secret_gate`並接runtime；modified F2/F3/F5/F9不再誤觸remake全域shortcut。撤回`found_secret_*`永久顯示第六項等同原版的斷言；ch02 E2已由後續項目閉合，其餘town仍待逐章驗收。
- [~] **UI-VIS-PREPARATION**：`0x318ad..0x321c8` 已修正
  `0x32004` 的輸入正規化，並證實選滿後先由 `0x320fc` 重排隊伍、再走
  `0x31d3c..0x31db4` 最終確認。外層 `0x2d093` 另證實先顯示 FDTXT
  `0x201`「要進入戰場嗎？」；可選記錄不超過15／19時完全略過 `0x318ad`，超過才進
  30個全零選取旗標。城鎮重製流程已改為出發確認→按名冊門檻略過或選人→最終
  確認，任一取消依可編輯 `cancel` 回城；另以 `0x2cad7` 分開無城鎮路徑的
  FDTXT `0x19a`「要記錄戰況嗎？」與可選存檔，拒絕時仍進選人。已刪除預先全選與小隊不足按Escape
  強行出發。正常序章→第1章哈諾加入→戰後同步五人→羅德鎮→整備回歸已通過，
  並補上後加入角色的持久快照。2026-07-29 更新：選人主畫面、角色狀態、
  待機週期及最終確認已接原版索引色正式路徑；`0x1f42d` 已更正為戰場進入
  演出，不是選人 slide。城鎮 FDTXT `0x201` 提示現保存實際 town frame，
  無城鎮 FDTXT `0x19a` 提示依 `0x2cc04` 使用黑色來源；兩者都接
  6＋4＋兩 tick 脈動＋4＋5＋還原，肯定存檔／轉場只在還原幀呈現後執行。
  2026-08-02 IDA／Capstone勘誤：`0x320fc` 的目的index從1開始，selection
  byte `i` 對應persistent record `i+1`，record0固定且不消耗quota；重製已改為
  固定1筆＋可選15／19筆，正常戰場上限為16／20。舊「最多只上場15／19人」
  行為已撤回，證據見[`fd2_preparation_fixed_record_ida.txt`](../data/ida/fd2_preparation_fixed_record_ida.txt)。
  保存 record 與 ch02 departure 生命週期證據圖。仍須晚期合法存檔、
  跨畫面初始相位與 DOSBox 同狀態實機差分，故維持部分完成。
- [~] **UI-VIS-LOAD**：合法 IDA 9.4 證實 `0x25F48` 載入 FDOTHER #13，
  `0x30437` 使用 entry16（310×86）於 `(5,112)`，不是 FDOTHER #5
  對話框。production 已改走 FDTXT #0／原版字型／palette 的 indexed
  compositor；空槽 320×200 與 DOSBox oracle 全幀 RGB 相同，並新增目前
  source 的 [`load-empty-remake.png`](../figures/load-empty-remake.png)。
  `/tmp` 修改存檔的 chapter1 有效槽與 production 也全幀 RGB 相同；這只
  關閉有效槽排版，成功 native restore、delete/overwrite 與 roster ABI
  仍待 E2。
- [ ] **UI-VIS-DIFF-HARNESS**：固定同一FD2.SAV／roster／camera／cursor／tick，輸出DOSBox與remake 320×200 pair及pixel diff；現有ch01兩張角色狀態不同，只證明compositor slice。
- [ ] **ENGINE-REPOSITORY-EXTRACTION-GATE**：待 FD2 忠實模式的核心垂直
  路徑穩定後，建立獨立 GitHub 引擎倉庫。抽離範圍只包含可由第二個真實
  戰役消費的網格、回合排程、事件虛擬機、索引色渲染、輸入、存檔介面與
  跨平台層；FDFIELD／FD2.SAV、handler ABI、原版戰役、位址證據與
  FD2 專屬相容規則留在本倉庫。驗收門檻為：第二個可編輯戰役不含
  `fd2` product branch 即可啟動、遊玩、存讀檔及建置，兩倉 CI 均可由
  Docker 重生。本倉庫原創程式碼與文件已採 PolyForm Noncommercial 1.0.0，商業
  用途須另向儲存庫擁有者取得書面授權；未來抽離引擎時仍須決定新倉庫是否沿用
  此授權，並補定貢獻規範與兩倉版本相依方式。

## 文件狀態入口（更新至 2026-08-11）

目前統計：`[x]=576`、`[~]=119`、`[ ]=70`；只計算本文件的 checklist 行，且僅代表工程項目數，不是原版完成百分比。

- [x] 根目錄 `README.md` 改為「資產／RE／引擎切片／原版差距」四欄狀態表，加入已驗證成果圖片；不再宣稱全 30 章 parity。
- [x] `remake/README.md` 改為垂直切片與失敗即關閉差距說明；此處記錄的是當時閱讀順序，目前已改為 README → `58` 覆蓋矩陣 → `56` SDD → `57` 介面矩陣 → 本工作清單。
- [x] `20`／`22` 的「所有必要能力已完成／只剩工程整合」過強斷言曾先降級；兩份早期可行性／技術驗證快照後於 2026-08-27 因已被現行工程與證據文件完全取代而刪除。
- [~] 專題 RE 文件仍保留各自證據與歷史修正；不直接合併成單一長文，避免丟失位址層證據。現況衝突依 `58`、`56`、`57` 與本文件頂端有效佇列裁決。沒有獨有原始證據的舊落差表 `42` 已於 2026-08-27 刪除。
- [x] 2026-07-27 README/KB review：README 改正「跨平台已完成」「EXE 全部表已閉合」「SDL2 第二條 runtime」等過強敘述，補上原版／重製對話圖與可驗證差距說明；當時仍保留 `90`、`30`、`51`、`SESSION-HANDOFF-*`。2026-08-27 再依獨有價值複核，只刪除已被取代的 `90`、`30`，保留具玩家實測與錯誤形成證據的 `51`、`SESSION-HANDOFF-*`。
- [x] **DOC-NO-VALUE-SNAPSHOT-REMOVAL-20260827**：逐檔複核 87 份受版控
  Markdown 後，刪除沒有獨有原始位元組、執行期擷取、文化考證或錯誤形成證據，
  且已由現況文件完全取代的 `20`、`21`、`22`、`30`、`42`、`90`。工程入口改由
  `docs/ENGINEERING.md` 承載，完成與落差判定只看 `56`／`57`／`58`／`91`；原文
  仍可從 Git 歷史回查。`README-RESTORE`、`51`、`99`、handoff、實測報告、資料
  證據與本機 donate 構想均有獨有價值，明確保留。
- [x] **DOC-SECOND-PASS-LIVE-ASSERTION-AUDIT-20260827**：在刪除舊快照後重新複核
  81 份受版控 Markdown。修正 `57` 同時把 CONTINUE／service3 寫成已接與未接的
  內部矛盾，移除 title／戰場局部成果看似整體完成度的百分比，並把商店、教會
  剩餘工作限定為未修改原版同狀態 `PLAYER-E2`；`58` 與 README 同步區分
  `RUNTIME-E1`、候選封包及 13／60 抽樣門檻。全專案本地連結檢查為 0 斷鏈，
  圖片引用均存在；沒有殘留的已刪檔引用，也沒有其他符合刪除門檻的受版控文件。
- [x] **DOC-THIRD-PASS-REPRODUCIBLE-CLAIMS-20260827**：改以機器清冊、現行 Go
  符號與 Git 歷史交叉稽核，修正 `58` 檔首／後段殘留的產品31、未知1,104舊數字；
  canonical IDA 9.4 清冊為函式1,305、產品37、runtime170、未知1,098。`56／91`
  另有七組已被後續精確回歸取代的測試名稱仍用現在式，現已換成目前原始碼存在的
  planner／consumer／postbattle／montage 測試，或明示只屬 Git 歷史。未知函式依
  玩家影響分類；DOS／PIT／DAC與裝飾性 RNG 不阻擋，玩法 RNG 仍須保留契約。
- [x] **IDA-INVENTORY-EXACT-REBUILD-20260827**：使用授權 IDA Pro 9.4 Docker、
  唯讀固定雜湊 `FD2.EXE` 與 tmpfs 從零重生清冊。為避開 IDAPython 對中文輸入
  路徑的 ASCII `UnicodeEncodeError`，把同一檔案唯讀綁定為 `/input/FD2.EXE`；
  匯出器仍驗證 size／MD5／SHA-256。2,299,687-byte 完整 JSON 壓縮後與版控清冊
  逐欄位相同：函式1,305、產品37、runtime170、unknown1,098、語意註記38。
  `1,089` 為數字顛倒，不另建清冊版本；這項重生不替 unknown 猜測語意。
- [x] **IDA-UNKNOWN-MARKDOWN-FOOTPRINT-AUDIT-20260827**：將上述1,098個 unknown
  函式範圍與全專案 Markdown／文字證據交叉比對，找到508個精確函式起點、524個
  函式範圍曾被提及。位址命中不等於語意閉合；只把具直接 caller、consumer、
  writer／控制流與受版控證據的十筆窄語意加入非破壞性索引。授權 IDA 9.4 從
  固定雜湊原檔重新匯出後為函式1,305、產品46、runtime171、unknown1,088、語意
  註記48。封包成功只驗證 engine／資料包／玩家垂直切片的交付完整性，不證明
  未分類原版 helper，也不把 `RUNTIME-E1` 提升為 `PLAYER-E2`。
- [x] **IDA-UNKNOWN-FOOTPRINT-REVIEW-BATCH2-20260827**：新增可重跑產生器與完整
  `fd2_unknown_footprints.json`，將前批後的1,088筆分成337筆現況＋直接產物、
  43筆直接產物、86筆現況斷言、48筆歷史／raw 線索及574筆無受版控文字足跡。
  第一批人工複核接受十二筆窄產品語意，涵蓋 FDFIELD／FDICON、玩家 controller、
  AIL wrappers、postbattle gate、raw/event constructors、input cleanup、RNG 與上下框
  portrait blitters；`0x36CD7` 等證據不足者明確拒絕。授權 IDA 9.4 重生結果為
  產品58、runtime171、unknown1,076、語意註記60；剩餘第一優先候選325筆。
- [x] **RE-WATCOM-STACK-CHECK-36CD7-20260827**：使用者追問後重開前批保守拒絕，
  授權IDA直接證實`0x36CD7`保留EAX並把stack demand交給`0x36CEA`；後者比較
  prospective ESP、process lower limit與SS selector，`0x36D07`則以
  `Stack Overflow!`／code 1退出。這是stack limit／overflow check，不是舊稱的逐頁
  guard-page probe。三段均歸runtime，541個prologue callers可由產品原語統計排除；
  清冊成為產品58／runtime174／unknown1,073／語意63。專案與`~/.codex`均新增可重用
  pattern、判讀步驟及停止線。
- [x] **RE-TOOLCHAIN-FINGERPRINT-AND-AIL-BACKGROUND-20260827**：固定雜湊證實
  Watcom C/C++32家族與LE產物，但精確compiler／WLINK版本仍未知；同版隨附檔則
  直接證實DOS/4GW 1.92、Miles AIL 3.02及AFM 1.00。依此分流160-call-site的
  `0x3EEDA`，以IRQ 8 timer writer、初始化owner與AIL consumer閉合為背景中斷
  巢狀狀態runtime；精確`AIL_background`名稱仍為強推論。IDA重生為產品58／
  runtime175／unknown1,072／語意64。通用方法已同步`~/.codex`與`~/my_skill`。
- [x] 2026-07-27 stale dialogue-operand assertion cleanup：`09`、`01`、`18` 不再把控制碼第二 word 一律稱為固定肖像/DATO ID；依 `0x15f84→0x12c60` 分開 identity lookup、runtime unit `+7` 與 direct-DATO fallback，並將 `FFFA/FFFB` 統一修正為遞迴名稱／數值插入碼，不是特效。
- [x] 2026-07-27 second-pass dialogue wording audit：`14` §4 的組合說明與 `-17/-18` 讀取步驟仍殘留「直接肖像 ID」舊斷言，已改成 identity lookup／record `+7`／direct-DATO fallback 三路 provenance；未修改任何未證實的 story operand。
- [x] 2026-07-27 expansion-doc assertion audit：`17-scenario-expansion-evaluation.md` 原稱「原版評分式 AI 已還原、可照搬」已撤回，改以 `11` 的 raw dispatcher/candidate/score slices 與完整 runtime 未閉合為準；`50` 的 persistence 句也限定為 remake 自有 JSON projection，不冒稱 `FD2.SAV` byte identity。
- [x] **DOC-EVENT-DSL-ASSERTION-AUDIT-20260728**：將`29-remake-extensible-event-system.md`明確降級為歷史設計草案；刪除「handler只管勝負／動作全在FDFIELD」「record +5 bit0全域等同存活」「第一章主角含妮雅」「示例已完整重現30關」等會污染忠實模式的斷言。同步將第3/6/7/8輪的「核心全完成／通用1:1／像素級收官／魔法SFX補完」標題限定為當時codec或fixture範圍；當時沿用的40–45%視覺估計現已撤回，shop recipient production接線明列為E1而非DOSBox lifecycle parity，並刪除已被後續closure取代的ch29 cleanup重複待辦。
- [x] **DOC-REPO-WIDE-ASSERTION-AUDIT-20260728**：擴大審核歷史專題文件並修正會被當成規格的現行矛盾：`00/28/53`不再把攻略稱為handler ground truth；`19/29/30`把自動campaign與Registry降為尚未閉合的scaffold／設計提案；`25`保留raw byte5 caller predicate；`35`刪除BG=TAI台座與`0x53ec8`=縮放X；`39`區分AFM resource decode與caller schedule；`44`撤回「序章無單位移動」「group10/11全遊戲死資料」及過時兩行直進戰鬥live-state；`47/50`撤回所有章NPC永遠dir0；`50`的campaign graph test不再冒稱全戰役原版route E2；`99`的資產全解改為當時base-codec範圍。
- [x] **AGENT-MEMORY-AND-DOCKER-HYGIENE-20260729**：新增根目錄
  `AGENTS.md`，統一專案目標、文件權威順序、E0–E3、known corrections、
  fail-closed、Docker-only Capstone、subagent review與重大更新才commit/push等
  跨session規則；`CLAUDE.md`改為指向單一操作契約。另建立
  `~/.codex/AGENTS.md`共用Docker鐵則。實際停止四個已跑21小時的
  `fd2-go-test-local` containers，確認沒有FD2 container殘留，並刪除已被
  authorized workflow取代、repo無引用的3.6GB `fd2-ida-local` image；保留目前
  cap/go-test/dosbox/authorized-IDA各一份可重現image。
- [x] **RE-POSTBATTLE-HUB-ROUTE-2D093**：依合法 IDA Pro 9.4／Capstone 的
  `0x2CAD7/0x2D093` 與 `0x526B9` raw table，新增
  `fdother.ResolveNativePostbattleRoute`；保存 preparation-first gate、
  hub selector→raw callee mapping、invalid fail-closed。IDA 再閉合
  `0x2CAD7` 回傳值：子流程 raw 0 會內部重複；直接整備／option2 的非零
  結果使 gate 回傳0，其餘 option 的非零結果使 gate 回傳1。
  `ResolveNativePostbattleOutcome` 只保存這個 raw 契約，不把 option 或
  raw 1 自動命名成酒店／商店／教會／結局，也不直接呼叫 scene。
- [x] **RE-TOWN-SHOP-SERVICE-2E341**：Docker Capstone 固定 resource與selector後續已完成callee dataflow：`0→0x2f0b0` purchase、`1→0x2f642` sell、`2→0x2f883` equip、`3→0x2f8ea` inventory transfer。命名依insert/remove/equip/gold writer與FDTXT，不依icon猜測；`ResolveNativeShopServiceRoute`現保存typed kind但仍不呼叫scene。
- [x] **RE-TOWN-HOTEL-SERVICE-2FC85**：Docker Capstone 固定 `0x2fc85` raw resource `13`、selector `0/1/2→0x2ffa5/0x30012/0x301f4`，selector3→`0x19953→0x197e5`；新增 `fdother.ResolveNativeHotelServiceRoute` raw plan/regression。只保存 address-level order，不命名服務、不執行 scene。
- [x] **RE-PREPARATION-CAP-318AD**：Docker Capstone 重核 `0x318ad`：`[0x53c03] <= 0x1a` 時 cap=15，`>0x1a` 時 cap=19；新增 `fdother.NativePreparationPartyLimit` 與 boundary regression。明確以 native index 為輸入，不把 late cap 猜成顯示章號或直接改寫 JOIN roster。
- [x] **RE-PREPARATION-PREVIEW-31E80**：Docker Capstone 完整 trace 固定 `0x31e80` 讀 caller-owned 30-byte selection table、以 `0x320ce` 計數，依 flag 分支 `0x4deda/0x4de56` 做 indexed preview；body 未寫 selection table／persistent roster。撤回把它當 Enter/toggle mutation，remake 保持 `partyDeploy` mutation 與 renderer boundary 分離。
- [x] **PROGRESS-AUDIT-E0-TO-E2**：重新檢視近期對話、commit 與權威文件，確認停滯主因是 E0 raw slice 沒有同步 runtime consumer／UI input trace／E2 screenshot、`main.go` 仍是 monolithic scene owner，以及 30 章 postbattle graph 未逐章驗收；新增停止孤立 offset 擴張的門檻，下一里程碑改為 title→dialog→battle→postbattle hub→preparation/town 垂直鏈。
- [x] **UI-01-TITLE-TRACE**：新增純 `TitleMenuState`／`TitleSlotState`，保存原版主選單三項 wrap、24-tick confirm flash、load branch 與 `0x30550` 四槽 bounded/no-wrap/cancel contract；`titleUpdate` 的 Ebiten input 已改走同一 state transition，Docker/Xvfb regression 可重播 selection/action trace。仍不宣稱原版逐幀 visual parity 或 FD2.SAV 相容。
- [x] **UI-07-08-CAMPAIGN-MENU-TRACE**：新增 `campaign.MenuState`，將 `choice/town` hub 的 bounded cursor、空選項 fail-closed 與 confirm→`optN` transition 抽成純 state contract；`campInput` 已共用該 contract，internal/campaign 與 Docker/Xvfb focused regression 通過。未命名 town service、未跳過 handler，逐章 route/E2 仍是 partial。
- [x] **UI-08-TOWN-HUB-SOURCE-SCREENSHOT**：用目前 source 在 `fd2-go-test-local` 內重新 build（不使用舊 `fd2-linux`），以 `FD2_CAMP_NODE=town_ch02`、frame 30 產生 [`town-hub-remake.png`](../figures/town-hub-remake.png) 並加入 README；這是 remake current artifact，不是原版 visual parity。
- [x] **UI-08-TOWN-HUB-CH02-E2-SLICE**：在隔離 `/tmp` game sandbox 只以 route patch 跳過戰鬥勝負判定，原版 postbattle handler／campaign gate／town resources 均照常執行；走完20次戰後對話確認，第21次取得 ch02 town。初次 diff 抓到 `BlitNativeGlyph` 把 `0x4ea2a` 的 shadow 誤寫同列左側；修正為下一列左下／正下後，selection0/pulse2 與 Left→selection1/pulse2 的320×200 raw RGB MD5分別整幀相同。這不證明 route patch 是原版玩法，也不解除其他 variant/input 的 E2 gate。
- [x] **RE-FDTXT-GLYPH-SHADOW-4EA2A**：Docker Capstone 指令級確認 foreground→`edi`、shadow→`edi+(stride-1)`／`edi+stride`；撤回同列`edi-1` shadow 的錯誤 ABI 與兩個保護錯誤位址的 roster/class tests。共用 `BlitNativeGlyph`、相鄰筆畫 regression 與 consumer tests 已修；town selection0/1 E2 整幀 hash 是直接 visual oracle。
- [x] **UI-08-TOWN-DETERMINISTIC-SHOT-STATE**：新增 screenshot-only `FD2_SHOT_TOWN_STATE=selection,pulse`，只接受 native town 的 selection0..5／pulse0..3，非法值 fail closed；正常 input 不讀此 hook。`0x2ce7a/0x2ceac/0x2cef7` 無 `[0x54133]` writer，故方向鍵／secret reveal 不再猜測性 reset pulse。
- [x] **UI-08-TOWN-VARIANT0-SIX-SELECTION-E2**：以新 `waittown0:key,delay,max` 背景簽章同步避免固定 Return 次數過按，原版依 input 到達0酒店、1武器店、2出口、3道具店、4教會、5「???」；六幀各自與 deterministic remake pulse 做320×200 raw RGB全幀hash相同。另實測Right 0→4、Left 4→0、Shift+F1 0→5；secret reveal production helper有pulse/clock continuity regression。`waittown0`只辨識variant0，不冒稱可同步variant1/2。
- [x] **UI-09-CH02-SECRET-SHOP-SERVICE0-E2**：由同一原版 sandbox 走 `Shift+F1→selection5→Enter`，捕捉 variant5/resource63/DATO#0x84 秘密商店。新增 strict screenshot-only `FD2_SHOT_SHOP_STATE=service,pulse,gold`；E2 抓出 production service phase 被 caller 與 compositor 雙重 `/2`、導致 selected sprite 永不出現，已只修正該 service-menu call site並補 consumer regression。gold0 時原版/rebuild phase0、phase2 raw RGB MD5 分別 `12fad3c03096aae48098c8f9074370c7`、`e5654e8ed03d1e4fd30b2c76106bb7a1`，兩組皆整幀 AE=0。
- [x] **UI-09-CH02-SECRET-SHOP-FOUR-SERVICES-RETURN-E2**：原版實際 Right `0→1→2→3→0`、Left `0→3` 六幀各與同service remake pulse全幀AE=0；Escape closing後回town hidden selection5，亦與town pulse1全幀AE=0。E2抓出`leaveShop→enterNode`把selection誤重設0；production現只對已證實native variant1/3/5恢復同值，custom shop仍預設0，並有boundary regression。新增[兩列對照圖](../figures/secret-shop-ch02-services-return-original-vs-remake.png)。
- [x] **UI-09-CH02-SHOP-VARIANTS-1-3-5-E2**：由同一原版town分別以selection1進weapon variant1、selection3進item variant3；各取10個steady樣本，variant1全部、variant3除首張transition外九張均在phase0/2交替並與production全幀AE=0。selected phase raw RGB MD5為variant1 `69003be54f47c221916c1ed89cf1d26f`、variant3 `dd5d80bb761cc87980dff066773f6763`、variant5 `e5654e8ed03d1e4fd30b2c76106bb7a1`，原版/remake成對相同。新增[三變體對照圖](../figures/shop-variants-1-3-5-original-vs-remake.png)；主選單variant gate關閉，child panels仍待。
- [x] **UI-09-CH02-WEAPON-PURCHASE-LIST-E2**：新增 strict screenshot-only `FD2_SHOT_SHOP_PURCHASE_STATE=selection,start,gold`，只接受production已claim的native purchase mode、合法goods selection與正規化偶數window start，其他狀態fail closed。原版service0 Enter後實測Right `0→1`、Down `1→3`、Left `3→2`；延長等待排除進場最初2像素與portrait animation transient後，四個stable 320×200幀均與production全幀AE=0。raw RGB MD5為selection0 `1589cee3c068936f0beb6058cfd63991`、1 `7480dbb0284b033b4e9ad8c8c7a8b78e`、2 `48d6182e261ebce574b08c4778b8a072`、3 `3c0a2c935260b8ca80432b25b3600111`；新增[兩列對照圖](../figures/shop-purchase-ch02-selections-original-vs-remake.png)。這只關閉購買清單steady/input，不推廣到確認、收件者或交易結果。
- [x] **UI-09-CH02-WEAPON-PURCHASE-CONFIRM-E2**：新增 strict screenshot-only `FD2_SHOT_SHOP_CONFIRM_STATE=good,choice,pulse,gold`；只接受production已claim且shared assets完整的native shop、真實editable good、choice0/1、pulse0..3與合法gold，其他狀態fail closed。原版由good0 Enter開「布衣／50元／要不要啊？」、Right到No、Left回Yes；高頻取樣的可見selected Yes/No raw RGB MD5分別`7a07b1c064ca2c431bc97c798dcfd51e`／`56f6ffb003e87cbc63d7a915ac4b5dd0`，normal frame `b8cce25df13447e73e1750a8b2edaf0f`，三者均與production全幀AE=0。新增[兩列對照圖](../figures/shop-purchase-confirm-ch02-original-vs-remake.png)；只關閉confirmation steady/input，不推廣到recipient或transaction result。
- [x] **UI-09-CH02-WEAPON-PURCHASE-INSUFFICIENT-E2**：新增 strict screenshot-only `FD2_SHOT_SHOP_INSUFFICIENT_STATE=good,gold`；只接受production已claim且shared assets完整的native shop、真實editable good及`gold<price`，不扣金、不改recipient，final compositor失敗即原子回復。原版gold0對good0選Yes後顯示「錢不夠！」及等待標記。最初誤用第四個inward choice frame造成整幀AE=563；Docker Capstone重核證實原版`0x197e5`四次present後由`0x19913..0x1994c`恢復310×86 question region，`0x16c57(1)`再以FDOTHER#5 cell18/19畫等待標記。remake目前以ch02限定的deterministic recomposition取得相同pixels，尚非generic saved-buffer restore owner；原版／production raw RGB MD5皆為`6babcedfe2017a7457924c4df65ba7dc`、整幀AE=0。新增[左右對照圖](../figures/shop-purchase-insufficient-ch02-original-vs-remake.png)。不外推到recipient/no-recipient/full/success。
- [x] **UI-09-CH02-EQUIPMENT-RECIPIENT-E2**：`FD2_CAMP_NODE=shop_ch02_weapon`本身不建立persistent party，故新增screenshot-only `FD2_SHOT_PARTY_BINDING`：只接受compile無issue、同時提供`PartyScenario+PartyOrder`的LOADCH binding，依該binding記錄的order materialize typed roster並要求identity/selector/race/class/byte6/raw inventory/equipment-base provenance；hook不獨立重證JOIN來源位址，姓名與順序也不硬編。`FD2_SHOT_SHOP_EQUIPMENT_RECIPIENT_STATE=good,selection,start,cycle,gold`再嚴格驗證商品、價格、三列window與FDICON cycle。E2抓出ch01 party遺漏DX projection；2/2/1/2由可見HIT/EV與已知equipment rows交叉約束，不是直接raw `+0x3e` dump。另修正HIT/EV整組相對AP/DP右移3px及arrow Y anchor；修正後good0/selection0/start0/cycle1/gold1000原版與production raw RGB MD5皆`28258fb3ce5bc42eb1c701a7792d193b`、整幀AE=0。新增[左右對照圖](../figures/shop-equipment-recipient-ch02-original-vs-remake.png)；此bridge不代表正常campaign persistence，亦不外推到input/scroll、FD2.SAV相容、no-recipient/full/success。
- [x] **UI-VERTICAL-CH02-TOWN-PREPARATION**：將 `Game.stepCampaignMenu` 接到 `campInput`，新增 `TestCampaignTownPreparationInputTrace`，以 `down,down,enter(opt2)` 驗證 `town_ch02→preparation_ch02→story_ch02_pre→battle_ch02`；保存可編輯 trace [`town-preparation-ch02.json`](../data/ui-traces/town-preparation-ch02.json) 與目前 source rebuild 的 [`preparation-current-remake.png`](../figures/preparation-current-remake.png)。這是第一個 campaign/UI state vertical closure，仍不等於逐章原版 parity。
- [x] **UI-VERTICAL-CH02-TOWN-SHOP-RETURN**：新增 `Game.leaveShop` 與 `TestCampaignTownShopPurchaseReturnTrace`，驗證 `town_ch02→shop_ch02_weapon`、reserve 不先扣金、finalize 後回 town；保存 [`town-shop-ch02.json`](../data/ui-traces/town-shop-ch02.json) 與 source rebuild 的 [`shop-current-remake.png`](../figures/shop-current-remake.png)。這只閉合 remake shop/campaign boundary，不命名 native shop callee 或宣稱原版 parity。

- [x] **UI-VERTICAL-CH02-TOWN-CHURCH-REVIVE**：新增 `Game.reviveChurchUnit`／`Game.leaveChurch` 與 `TestCampaignTownChurchReviveReturnTrace`，驗證 `town_ch02→church_ch02→revive(level3,class1,fee7)→town_ch02`，gold 100→79、HP restore、OnField restore；保存 [`town-church-revive-ch02.json`](../data/ui-traces/town-church-revive-ch02.json)。原檔名 [`church-current-remake.png`](../figures/church-current-remake.png) 已於2026-08-27替換為正式`church_ch02` indexed runtime擷取；目前四項service具typed owner，但仍不宣稱原版caller E2。
- [x] **UI-VERTICAL-CH02-TOWN-CHURCH-CLASS-CHANGE**：`Game.applyChurchClassChange` 與 `TestCampaignTownChurchClassChangeReturnTrace` 驗證悠妮（native identity/portrait09）依 special override 解析唯一 portrait34/class21、Yes 確認、MV 5→7、Exp reset、消耗 item `0x5a`，並可 Escape 回 `town_ch02`；保存 [`town-church-class-change-ch02.json`](../data/ui-traces/town-church-class-change-ch02.json)。舊「可編輯三分支」斷言已撤回。
- [x] **UI-VERTICAL-CH02-SAVE-LOAD-BOUNDARY**：新增 `TestCampaignSaveLoadRestoresTownBoundaryAndParty`，驗證 town 節點 F5 存檔→清除 transient runtime→F9 讀檔後恢復 campaign cursor、gold、items、party roster/deploy/join order/chapter，並由 `enterNode` 清除 battle/shop/church state；保存 [`save-town-boundary-ch02.json`](../data/ui-traces/save-town-boundary-ch02.json)。這是 remake JSON boundary，不是 native `FD2.SAV` 相容性。
- [x] **UI-VERTICAL-CH02-TOWN-HOTEL-RAW-RETURN**：新增 `hotel` campaign node、`Game.applyHotelServiceSelection`／`Game.leaveHotel` 與 `TestCampaignTownHotelRawRouteReturnTrace`，驗證 `town_ch02→hotel_ch02→town_ch02`，selector 0/1/2/3 保留 raw resource13 與 `0x2ffa5/0x30012/0x301f4/0x19953→0x197e5` order；未命名服務、不做 party/gold mutation，未知 selector fail-closed。保存 [`town-hotel-raw-return-ch02.json`](../data/ui-traces/town-hotel-raw-return-ch02.json)。
- [x] **POSTBATTLE-UNBOUND-FAIL-CLOSED**：`Game.enterNode` 對沒有 active handler 的 `postbattle_*` cutscene 拒絕空 beats auto-advance，新增 `TestUnboundPostbattleCutsceneFailsClosed`；流程停在原 node、保留 `loadErr/msg`，避免未完成 persistence/reward handler 被誤當成直接回 town。
- [x] **POSTBATTLE-SAVE-FAIL-CLOSED**：`saveGameToSlot` 對所有 `postbattle_*` 節點拒絕 F5，新增 `TestSaveRejectsUnboundPostbattleBoundary`；未完成 persistence handler 不會產生假 save。
- [x] **POSTBATTLE-BINDING-GATE-AUDIT**：新增唯讀 `tools/audit_postbattle_binding_gates.py`，逐一依 handler source address 檢查 generated binding 的 `loadch/pan/dialog/act/layout` 覆蓋；歷史快照曾列18至22 active，**歷史快照曾達23 active／1 blocked，現況以 Docker 實際稽核的24 active／0 blocked為準**。ch09/ch10/ch12/ch18、玩家第23戰raw ch22 post、玩家第24戰raw ch23 post與玩家第25戰raw ch24 post已通過正式 regression並提升為active handler，其餘skeleton禁止自動啟用。

## 第 1 輪 ✅
- [x] 素材盤點(`FD2.EXE` + 12 `.DAT` + 音效驅動)
- [x] 破解 `.DAT` 容器格式 + 寫 `tools/unpack_dat.py`
- [x] 辨識圖像/調色盤/文本/地形表 header
- [x] 攻略萃取成知識庫
- [x] 建知識庫骨架 + RE 計畫 + 反思 + README + git push

## 第 2 輪 ✅
- [x] **當年開發工具考證**(Watcom C/C++32 + DOS/4GW + Miles AIL v3 / XMIDI + AFM 動畫工具/作者 Lo Yuan Tsung)→ `04-original-toolchain.md`
- [x] 建立本 worklist
- [x] **EXE 資料表 dump**:`tools/dump_exe_tables.py`,9 表全對齊「舊版」offset,5 表 dump 並自驗全通過 → `docs/data/exe_tables/`、`03-…`
- [x] **圖像解壓**:破解 RLE(c≥0x80 literal / c<0x80 run),`tools/decode_image.py` 渲染標題+背景驗證 → `05-image-compression-format.md`
- [x] **音樂解析**:確認 XMIDI,`tools/xmi2mid.py` 轉 15 首標準 MIDI(note 平衡、tempo 直通)→ `07-music-xmidi-format.md`
- [x] **動畫機制結構**:AFM 容器 + FIGANI 幀封裝(幀數自描述 + offset 表)→ `06-animation-format.md`

## 第 3 輪（歷史素材／codec round；勾選不代表核心引擎或 AI 全完成）
- [x] **文本解碼**:破解 FDTXT(uint16 glyph 索引 + 控制碼 + 0xFFFF)+ 找到自製字型(FDOTHER_004,16×16 1bpp,1824 字模),**還原可讀中文** → `08-text-and-font-format.md`、`tools/decode_text.py`
- [x] **動畫逐幀拆解**:✅ **完整破解**!反組譯參數化解碼器 0x4F43D + 解出 13-byte 幀標頭(realW/H 在 +9/+11)+
      4 模式 RLE → `tools/decode_figani.py` 把 **264 動畫 2118 幀**全部解出(騎士揮劍動畫視覺驗證)← 使用者明確要求,完成
- [x] **持久素材抽取**:`tools/extract_all.py` → 本機 `extracted/`(raw/images/animations/music/fonts);**不入版控**
- [x] **劇情/對話結構解出**:[控制碼][說話者肖像ID][『][對白][』];全 35 章渲染成可讀 PNG(`extracted/story/`)→ `09-…`
- [x] **序章(FDTXT_001)逐章轉錄完成**(`extracted/story/序章_transcript.md`,本機)
- [x] **敵/我方動畫機制文件**:解碼器變體家族(全彩/remap調色/silhouette/dither)+ 陣營/面向 → `10-…`
- [~] **敵人/NPC 戰場 AI** 反組譯文件：舊 `0x15140/0x15356` 位址已撤回；
  真正的物理候選評分入口 `0x14237` 與法術 `0x15AD8→0x15B77` 已分離，
  仍需閉合候選格順序、raw helper 語意、turn/camp 與 runtime execution → `11-…`
- [x] **RE-AI-CALLER-15AD8**：Docker Capstone 閉合 `0x15A1E..0x15B76` 的 bounded candidate→`0x14818` target builder→`0x15B77` score→best-score/tie-break/write globals 邊界；`0x15B77` 的 command `<0x0d`、recovery `0x0d..0x10`、raw flag `0x14..0x16` branches 已寫入 `11`，不把它升格成完整 AI turn。
- [x] **RE-AI-DISPATCH-14EF0**：`0x14EF0` 已由 `RE-AI-14EF0-RAW-DISPATCH` 的
  IDA／Capstone 證據取代舊的 candidate-only 敘述；此歷史條目保留索引，現況
  以完整 raw 尾端契約、六個 caller 與 fail-closed 診斷路由為準。
- [x] **RE-REFERENCE-FILE-HASHES**：固定目前反組譯版本的 `FD2.EXE` 大小
  `357074`、MD5 `b97caf2239a27a896069d03549d96e1e` 與 SHA-256，另為
  12 個實際解析資產建立可重算清單
  [`fd2-reference-files.json`](../data/fd2-reference-files.json)；
  `disasm_le.py` 每次執行會在標準錯誤輸出顯示來源指紋。不同雜湊不得沿用位址。
- [~] **RE-BATTLE-AI-SPECIAL-TOPIC**：已把 `0x1A4EB/0x1A58F→0x1D80B/0x1D8BA
  →0x13A9F→0x14EF0` 整理為可信拓撲，並分開 `0x14237` 物理評分與
  `0x15AD8→0x15B77` 法術評分。`0x1548E` 已更正為選擇結果執行，不是
  pathfinder；`0x145CD→0x4E040→0x146D1→0x14B16` 已閉合 raw 落點產生與
  row-major 順序，`0x14B78→0x4E1A6→0x13488` 已閉合路徑方向與實際落點
  排序；無 action 的一般 mode 0 備援已定位到 `0x14121→0x13E9C`，舊
  `0x15192` 假說撤回。2026-07-29 又固定 `0x1D80B` 的 raw `+6==1`
  單遍，以及 `0x1D8BA` 對 raw `+6==0` 的「分數門檻預選＋無門檻第二遍」；
  每筆均依序走結構已閉合的 90 筆全域事件表、30 筆章節戰場事件 handler 表與
  `[0x53ECC]` pending 碼，round counter 在第二掃描返回後才增加。既有
  constructor＋`0x14818` 消費證據已固定 raw camp code 敵0／友1／己2，
  故前者是友軍單遍、後者是敵軍兩遍；具型別
  `PlanNativePhaseUnitScans` 已分開三遍、保留 signed threshold 與缺 score
  fail-closed regression；`ExecuteNativePhaseUnitScans` 另保存每遍動態
  重判、兩張表順序與 pending 提前退出。`0x1598A→53C23` 命令遮罩候選、`0x1567E→53C33`
  item-command 候選與 `0x13512` bit7 已串成「分數門檻優先遍後排除雙動」；
  不替門檻命名成高階玩法。
  尚未接 production `NextAIPlan`。下一步以固定存檔 trace 驗證實際選中
  command／目標與畫面順序；不得重複把陣營碼、兩張表或 pending 碼降回未知。
- [x] **RE-AI-PATH-FALLBACK-14B78**：Docker Capstone 閉合 `0x4E1A6`
  mode 0/1/2、方向碼、成本與 `0x40/0x80` gate；`0x14B78` 依
  Manhattan→軸差→逐列順序選落點，`0x13E9C` 才是最後的 Manhattan
  最近 opposite-group 座標備援。新增 raw-only path／blocked-coordinate／
  destination ranking／nearest-coordinate adapters 與決定性測試。
- [x] **RE-AI-PHYSICAL-SCORE-14237**：Docker Capstone 閉合候選格×目標枚舉、
  actor/target `+0x48/+0x4A` 地形百分比修正、差值`<=2`拒絕、嚴格
  `score>target +0x40`時×2/priority18、`0x1DEBE`及raw `+8`調整，以及
  priority→score→先出現者同分規則。新增 raw-only
  `ScoreNativePhysicalAttackCandidate`／`SelectNativePhysicalAttackCandidate`
  與門檻、嚴格HP比較、priority及穩定同分測試；合法 IDA 9.4 另交叉確認
  函式邊界、三個 callers 與 `0x1DEBE` 唯一 caller。不接 normalized planner。
  完整 IDA／Capstone 位址級證據見
  [`fd2_ai_physical_score_ida.txt`](../data/ida/fd2_ai_physical_score_ida.txt)。
- [x] **RE-AI-PHYSICAL-CANDIDATE-BRIDGE**：新增
  `BuildNativeAIPhysicalAttackCandidates`，以 raw records 串起
  `NativeAIPhysicalDestinations` 的 row-major 落點、caller 明示的
  `0x14818` target geometry、raw `+5/+6` 目標篩選與 detached actor／target
  record snapshots，再交給 caller-owned score resolver。測試確認候選順序、
  raw 群組篩選、穩定選擇與 resolver 失敗即關閉；這是 E0 資料鏈，不代表
  `0x1B83D` actor 物品列來源、地形修正 writer、正式回合執行或
  尚未接入原版 native AI 的 production planner；現有 `NextAIPlan` 仍是
  normalized approximation 的執行分離，不代表原版 AI parity。
- [x] **RE-AI-PHYSICAL-ITEM-SOURCE**：合法 IDA 9.4 釐清
  `0x14237` 的 `0x1B83D(unit,0)` 只是 equipped low-item slot lookup；
  `0x1B722(unit,slot)` 讀 runtime `+0x0B` item ID，`0x4E56C` 再以
  `0x602AD + id*0x17` 取得 item row，`0x142B2..0x142BE` 讀 row
  `+0x0B/+0x0C` 供 `0x14818`。新增
  `battle.ResolveNativeAIPhysicalItemSource` 與 raw bounds／detached-row
  regression；這修正「0x1B83D command record 來源」斷言，但仍是 E0 provenance，
  不接 score resolver、選中目標、正式回合或 `NextAIPlan`。
- [x] **RE-AI-PHYSICAL-EXECUTION-1548E**：Docker Capstone 證實唯一 callers
  `0x13E39/0x14F9B`；callee 消費 `0x53C43/47/4B`，經 `0x14B78` 後依
  `0x53AF9` 選地圖呈現或 `0x28A6C(actor,target)`，收尾固定回1。沒有
  `0x4EE40/0x4F355` call，故撤回「pathfinder／移動決策入口」斷言。
  合法 IDA 9.4 已交叉確認函式邊界與兩個 callers。
- [x] **RE-AI-UNIT-DISPATCH-13A9F**：Docker Capstone 閉合 `0x13A9F` 的 unit `0x50`-byte record、raw `+5 & 0x05` gate、`record+0x34 & 0x0f` command nibble 與 `0x14EF0/0x1598A/0x15311/0x1548E` 分支；保留 nibble 語意未命名。
- [x] **RE-AI-UNIT-SCANS-1D80B**：Docker Capstone 閉合 `0x1D80B/0x1D8BA/0x1D988` 三段 `[0x3BEB]` record scans、raw `+6/+5/+0x26` gates、`0x13A9F`／`0x1598A→0x1567E` 呼叫與 `[0x51A8F]/[0x53C03]` table dispatch；保留 raw table/loop semantic 未命名。
- [x] **RE-AI-PHASE-CALLBACK-ORDER**：合法 IDA Pro 9.4 優先確認
  `0x51B91` 為 90 筆、`0x51B19` 為 30 筆，並固定逐筆順序為
  可選全域事件→無條件章節 handler→pending 檢查；不合格 record 仍走尾段，
  第二遍則重新讀取第一遍可能改寫的 `record+5 bit7`。
  `fdother.ExecuteNativePhaseUnitScans` 保存動態重判、表界與提前退出的 E0
  契約；未知 handler 效果仍由呼叫端提供，不接正式 `NextAIPlan`。
  `[0x53ECC]` 碼 1 只證實固定 `0x22E5C` 資源 #79 呈現，碼 2 只證實
  章節戰後表→`0x2CAD7` gate；舊「世界地圖／中場／勝利」通用名稱已撤回。
- [x] **RE-AI-PHASE-CALLS-1A4EB**：Docker Capstone 固定 `0x1A4EB` 的 `0x1A813(1)→0x1A866(1)→0x1A7BD→0x1D80B→0x1A7F1` 與 `0x1A58F` 的 selector-0 對應鏈；只記 phase-specific raw callsites，不命名回合開始／結束。
- [x] **RE-AI-COMMAND-ENUM-1567E**：Docker Capstone 閉合 `0x1567E` 的 `record+0x0B+2*slot` item ID→row `+0x10` command、`command<=0x0F→0x14818`、`command>0x0F→0x149F8(command-0x10)`、`0x15880` score 與 `0x53C33/37/3B/3F` best writes；`0x53C3F` 是 inventory slot，執行端再由 `0x1B722` 解 item。撤回混用 `0x15B77` 與 command-list 的說法。
- [x] **RE-AI-SCORE-15880**：Docker Capstone 閉合 `0x15880` 的 item row `+0x0D/+0x0E` type/word 分支：type5/0x0D 的 current HP `<=max/3→8`、`<=max/2→3`、其餘0，raw `+0x34 bit7` ×3；type0x14/0x15 經 `0x4E516`、type0x18 直用 row word，target HP `<=threshold→0x12`、其餘8。`ScoreNativeAIItemCommandTargets` 已接並驗證邊界；不命名效果或 status。
- [x] **RE-AI-ITEM-PRODUCER-1567E**：保存帶雜湊的 `0x14818/0x149F8/0x14B16` 指令窗口，閉合 count-sized slot scan、row-major destinations、低 command area targets、高 command actor→destination cardinal targets、strict best 與 `[0x53C3F]=raw slot`。`ScoreNativeAI1567E` 已接；map0＋tracked item79 fixture 固定 score8／`(19,15)`／slot0。這是 E0 交叉 fixture，不宣稱一般玩家 map0 持有 item79。
- [x] **RE-AI-CANDIDATE-149F8**：Docker Capstone 閉合 `0x149F8` 的 cardinal ±X/±Y cursor steps、map bounds、`0x12C0D` unit lookup、raw `+6` selector gate、supplied byte-buffer writes 與 cursor restore；明確標為 candidate scanner，不命名 damage/hit/LOS/spell effect。
- [x] **RE-AI-MODE-SOURCE-10FB6**：Docker Capstone 閉合 FDFIELD 名冊 `b17/b18/b19` → runtime `+0x34/+0x35/+0x36`，33 圖 1887 筆低四位分布已保存為 `docs/data/fdfield_native_ai_modes.json`，資料管線與 `Unit` 保留原始來源；高四位不誤命名成 mode。
- [x] **RE-AI-MODE-WRITER-3419C**：閉合 `0x3419C` inclusive range writer 的保留高四位規則，以及 `0x13D20`／章節處理器的 whole-byte writes；新增 fail-closed materializer 與 writer regression。
- [x] **RE-AI-MODE2/11-BRANCHES**：原始 Capstone 與合法 IDA 9.4 交叉勘誤 mode 2 為 `0x14EF0` 失敗後 `0x14237→0x13B1E→0x13C06→0x13FD4`；`0x14237` 尾端固定回傳 0，因此會走共用零分支，但不走 `0x13E9C`。mode 11 依 `[0x53C23]`／`[0x53C4F]` 兩個獨立 signed `>=6` gate，第一段後仍評估物理第二段。`PlanNativeUnitMode2` regression 已同步修正，證據見 `fd2_ai_mode_dispatch_ida.txt`；先前「mode 2 不呼叫 `0x13FD4`」已撤回。
- [x] **RE-AI-MODE11-FULL-IDA-20260810**：合法 IDA Pro 9.4 與 Docker Capstone 重新核對
  `0x13E02..0x13E57`，證實 mode 11 直接 `0x1598A→(gate)0x15311→0x14237→(gate)0x1548E/0x14121→0x13FD4`，
  不先經 `0x14EF0`。同時保存 `0x1598A` 的 `[0x53C23]` producer、`0x14237` 的
  `[0x53C4F]` priority producer、`0x15311`／`0x1548E` consumer 邊界與 direct callers；
  `[0x53C4F]` 不命名成算術 score。完整輸入雜湊與位址基準見
  [`fd2_ai_mode11_full_ida_20260810.txt`](../data/ida/fd2_ai_mode11_full_ida_20260810.txt)。
  重製端仍只保留 E0 路由選擇，transaction owner、演出與一般玩家 E2 尚未完成。
- [x] **RE-AI-MODE0/1-BRANCHES**：同一份 `0x13A9F` 原始控制流已資料化為 `PlanNativeUnitMode0`／`PlanNativeUnitMode1`；mode 0 保留 `0x14EF0→0x14121→0x13E9C` 巢狀備援，`0x13E9C` 零回傳才到 `0x13FD4`；mode 1 只保留 `0x14EF0→0x14121→0x13FD4`。原 helper 仍只保存 E0 位址順序與 caller-supplied 回傳旗標；2026-08-10 的 `NextAIPlan` bridge 另以 raw provenance 接上 mode 0／1，缺來源仍失敗即關閉。
- [x] **RE-AI-MODE3-10-BRANCHES**：同一份 `0x13A9F` raw CFG 已資料化為 `PlanNativeUnitMode3/4/5/7/9/10`；保留 `0x12C60` 的 `-1`／索引分支、`0x12D7B→0x14B78→0x13FD4`、`0x51A83` 清零、mode 5 的 `+0x31/+0x32`／`0x53AD5`／`+0x34=7` writes 與 mode 7 的 `0x32975`。原 helper 的 caller-supplied 邊界仍保留；2026-08-10 `NextAIPlan` bridge 已接 mode 3／4／5／7／9／10 的 raw destination／event state，mode 5 raw AIL sample tuple 當時接到舊 `sfx_12.wav` 並在缺樣本時失敗即關閉；2026-08-29 已由 `FDOTHER_031/sample_012.ogg` 取代。
- [x] **RE-AI-MODE5-FULL-IDA-20260810**：重新固定 mode 5 的 `0x13C19..0x13D24`
  分支、`0x15DF3` 的 return `0=命中／-1=無命中`、`0x12263` 整圖 state tail，
  並以 IDA helper 字串證實 `0x25B45([0x53EE8],12,1)` 是 AIL sample
  stop/init/address/loop/start，不是 indexed renderer。`0x17AA9` 不是 mode 5
  direct caller；`0x25B45` raw sample tuple 已接同一 FDOTHER #31 導出的
  當時的 `sfx_12.wav`；2026-08-29 已由 `FDOTHER_031/sample_012.ogg` 取代，缺樣本時
  維持失敗即關閉，未知 sample 名稱不命名。完整證據見
  [`fd2_ai_mode5_full_ida_20260810.txt`](../data/ida/fd2_ai_mode5_full_ida_20260810.txt)。
- [x] **RE-AI-IDLE-RECOVERY-13FD4**：`0x13FD4` 只在 currentHP≠maxHP 且 raw `+0x25/+0x26==0` 時回復 `floor(maxHP/5)` 並封頂；新增 state-only adapter，玩家休息正式路徑同步刪除錯誤的最少回復 1 並接 raw transient gates。
- [x] **RE-AI-13FD4-FULL-IDA-20260810**：重新以合法 IDA Pro 9.4／Docker Capstone
  固定 `0x13FD4..0x14120` 的三次 `0x17AA9(1)` 等候、兩次 `0x1DA16` raw
  24×24 解碼、`0x11EB0` 312×192 緩衝區拷貝，以及 caller `0x19082` 的原始
  `a3==0` gate。這是 E0 raw presentation 邊界，不宣稱影格、色彩、音效或
  `0x12D7B` 的高階玩法；重製 renderer 仍失敗即關閉。證據見
  [`fd2_ai_13fd4_full_ida_20260810.txt`](../data/ida/fd2_ai_13fd4_full_ida_20260810.txt)。
- [x] **RE-AI-MODE11-WRITER-35F92**：`[0x53AD5]+0x10==4` 時，`0x36078→0x3419C(20,20,11)` 改寫單位 20 低四位；它是全域 90-entry 表的 event 82，不是第二張 30-entry 表的 entry 22。一般玩家觸發尚未閉合，且 33 張格子事件表沒有 event 82，不猜章節或人物。
- [~] **REMAKE-AI-MODE-RUNTIME**：模式 0/1/2/3/4/5/7/8/9/10 已有 `NextAIPlan` 與 game-layer E1 窄消費端（mode 1 僅接受唯一 raw blocked-coordinate，其他模式分別保留 nearest fallback、raw table／event／destination provenance、mode 8 common completion 等邊界）；mode 11 的雙段 owner（`0x15311`、`0x1548E`／`0x14121→0x13FD4`）已接 E1 continuation，`0x13FD4` 另有 indexed／音訊 owner；`0x14EF0` 也已閉合 command route 與資產 type-5 item route 的窄消費端。mode 1 的完整 raw producer、完整 command／item producer 與高階回合 orchestration、event 82 觸發、未知 command／relocation／spell／其他 item 演出與一般玩家 E2 仍未完成；所有未具 raw provenance 的分支維持失敗即關閉。`set_ai:berserk` 仍只是 inert 事件標記。
- [x] **RE-FIELD-EVENT-13A44**：閉合地圖 event-word low5 的 1-based slot、FDSHAP `0x20/0x40` 寶箱 gate、FDFIELD 控制段 16×2 `(event_id,selector)` 與 `0xFF` gate；33 張地圖已同步為可編輯資料並有失敗即關閉查詢。
- [~] **REMAKE-GLOBAL-EVENT-DISPATCH**：全域 `0x51B91` 已由錯誤的 58 entries 更正為 90 entries；回合事件使用 0..57，格子事件只覆蓋另一子集合。58..89 handler 的高階語意與各 dispatcher 的 selector 生產路徑仍須逐一閉合，未知 handler 不接正式流程。
- [x] **RE-POST-RESOLUTION-1AA1D**：閉合 `{kind:u8,payload:u16le}`，kind0/1 為物品／金錢、kind2 dispatch 全域事件、kind3 為另一呈現分支；建構器只採 FDFIELD b22+b23..24，撤回 b23..25 24-bit payload。
- [x] **REMAKE-NATIVE-TREASURE-ASSETS**：33 圖 composition+FDSHAP 寶物格及 16 槽控制列已選擇性同步；type0/1 可執行，其他型態保存 event/native_type 並失敗即關閉，不再誤給一般物品。
- [~] **RE-EVENT82-REACHABILITY**：turn、field、treasure、unit effect 與四個 EXE 硬編碼後處理列均無 payload82；目前無已知資料 producer。仍須稽核 runtime `+0x31..+0x33` 的其他 writer，未證實 dead code。
- [x] **REMAKE-TREASURE-EVENT58**：閉合 map25 slots0..4 共用 event58；空欄時依 slot 給 `[1D,2B,33,3D,47]` 並共同關閉五槽，滿八格不改狀態。規則已綁定 EXE 雜湊、匯出成 editable JSON 並接正式 ClaimTreasure。
- [x] **RE-PHASE-RESOURCE-1A7BD**：Docker Capstone 固定 `0x1A7BD` 是 `[0x53AF9]` gate 下的 `0x111BA(0x1A4D,0,0x40)` resource-handle setup，`0x1A7F1` 釋放 `[0x53B0F]`；已從 transient selector／campaign phase 語意中分離。
- [x] **音樂播放與場景切換**機制(AIL XMIDI 序列)→ `12-…`
- [x] **戰場選單與行動系統**(行動狀態機/選單游標/Get_EasyMagic)→ `13-…`
- [x] README 知識庫總索引(可點選分類)
- [x] **glyph→Unicode 對照表完成(1824/1824,100%)** → `docs/data/glyph_map.json`(含數字/英文/漢字/標點/機器人雙字元代號)
- [x] **全 35 章劇情轉錄完成**:自動解碼成含說話者的 UTF-8 → 本機 `extracted/story/full_story_auto.md`(1450 句);序章~第3章另有人工精校
- [x] **按鍵綁定**(Enter/Space 確認、ESC 取消、方向鍵)反組譯 → `13-…`
- [x] **Get_EasyMagic** 法術面板反組譯(0x18ED0)→ `13-…`
- [x] **場景→曲號對映**:play_bgm(0x26777)+ 32 處呼叫 track 反組譯 → `12-…`
- [x] **LE fixup xref 工具**(`tools/le_xref.py`)解開 DOS4GW 重定位,可做 data xref
- [x] **控制碼語意還原**(反組譯文本渲染器 0x16D00-0x17200):FFEF/EE/ED/EC=開對話框(FFEF 帶 DATO 頭像)、
      FFFE=換行、FFFD=翻頁等鍵、FFFF=結束 → `09`;副產物確認 **DATO.DAT=人物頭像**
- [x] **劇情校對**:解碼自驗 + 上下文揪出 14 處形近字模誤判並修正
      (脅/實/黨/費/鍛/輩/辭/摸/牢/樁/紮/襲/態/責)
- [x] **陣營/狀態 remap 配色**:確認 LUT 來源=FDOTHER 資源#3(LMI1,23張256-byte LUT),dump 並套用展示(LUT0灰=已行動…)→ `10`;BB→LUT索引精確對應待續
- [x] **DATO 頭像全解**(136×4嘴型幀)→ `01`§7;**Unicode→glyph 反向表+編碼器**(round-trip 100%)→ `tools/encode_text.py`
- [x] 各 track 呼叫端對應確切遊戲狀態名(片頭/世界圖/城鎮/戰鬥/劇情)→ doc12「場景切換時的換曲」已列 5 狀態對映(2026-07-05 核實)
- [x] **FDSHAP 圖塊庫解碼**:標頭 count + u32 offset 表 + native 24×24 four-mode RLE；2026-07-26 直接掃 FDSHAP_000 的 288 tiles，mode `[0,2,3]` 全部完整解碼，撤回「僅不透明 bg-RLE」錯誤。`render_map.py` 依 `0x4deda` ABI 保留 transparent spans 為 index0；多層 foreground composition 仍不得由單張 export 推論。→ `01`§8
- [x] **全 33 張戰場地圖抽取**:FDFIELD×FDSHAP(配對 map N→FDSHAP[2N],索引驗證全通過)→ 本機 `extracted/maps/`;`tools/extract_maps.py`、`render_map.py`
- [x] **FDICON.B24** = 1680 個 24×24 **地圖單位 Q版小人 sprite**（four-mode RLE；撤回「與 FDSHAP 為不同 bg-RLE codec」，兩者共享 ABI、renderer branch 不同；每角色組12=4方向×3幀）→ `31`
- [x] **TAI.DAT** = WxH 圖像(sprite-RLE,如 155×42);多為 UI/特殊圖
- [x] 寫一篇總覽:「1995 年怎麼做出炎龍騎士團2」→ `15`
- [x] 寫一篇總覽:「1995 年台灣怎麼做遊戲 — 炎龍騎士團2 技術全紀錄」→ `docs/knowledge-base/15-how-fd2-was-made-1995.md`(2026-07-05 核實存在)

- [x] **FDFIELD 三段完整解析**:構成(地形)/控制(出場數/回合事件/寶箱/敵我roster)/出場位置;全33圖 metadata → 本機 `extracted/maps/maps_metadata.json`;`tools/parse_field.py`

## 第 4 輪以後(暫定)
- [x] 地圖格式完整解析(FDFIELD 三段)+ 渲染全 33 圖(見上)
- [ ] 反組譯戰鬥/命中/傷害/AI 演算法(Ghidra)，與攻略公式交叉驗證
- [~] **物品系統反組譯**(M1 用)→ `32`:已確認物品表23B結構、roster 8裝備欄與 AP/DP raw temporary leads；舊 `0x15356` 傷害公式地址未由 canonical scan 證實。裝備加成精確累加點（夾攻擊大函式、表 base-relative）與使用效果碼待續。
- [ ] **轉職系統反組譯**(M4):轉職觸發(教會/道具)、職業數值替換、能力繼承、轉職後成長表切換 → 攻略道具表(勇者徽章→英雄…)交叉驗
- [~] **角色名對應**:補全 portrait→角色名 → `49`。核實後「12 個」已過期,實際已定案 38 組
      (0-31 共 32 + 48/66/68/96/97 共 5 + 本輪新增 126=ASR-06);其餘約 97 組多為泛用怪物/路人,
      對話走場景相依 `-19/-20`(見 `40`),**無法只靠對話反推**,需逐圖解 FDFIELD roster 才能繼續補
- [x] `FDICON.B24`=1680個24×24地圖單位sprite(sprite-RLE,見 `31`);`TAI.DAT`=WxH圖像(sprite-RLE)
- [~] `FD2.SAV` 存檔：Docker static trace 已固定 `rb/wb FD2.SAV`、全檔 `0x59cb`、四槽 record `+0x312b+i*0xa28`（`0x28` metadata + `0xa00` persistent roster）；真實 sandbox decode 與 `tools/fd2save.py` round-trip/tamper regression 已固定 `0x4dbd8` rolling-XOR、`0x4dbb9` byte-sum checksum。合法 IDA 9.4 閉合 reader `0x2602c..0x26098`、writer `0x30012` 及兩個戰間 caller；兩端對稱處理 roster 與 metadata `+0..+9`。production `FD2_NATIVE_SAVE` 已從 indexed selector 正式接到 `BuildNativeChapterSlotRestorePlan`：依雜湊綁定的 `0x526b9` table，把 raw chapter 1..29 原子還原為 fresh campaign flags、gold、typed persistent party 與 town／preparation node；ch21/ch27 的 postbattle inventory gate 不重播，錯誤不部分套用或誤轉 JSON loader。四槽 LOAD 仍不是 `0x10010` CONTINUE；尚缺一般玩家有效槽 E2、metadata `+10..+39` 其他可能 consumer、delete/overwrite 及 current-battle restore。不得再稱「強加密／無結構」；重製自有存檔仍為另一格式。
  `0x112A5→0x1145A→0x17FC0` 的 writer／consumer 已再由合法 IDA 固定
  item cells、command mask、race/class/level、transient、base AP/DP、
  MV/EXP、DX、HP/MP 與衍生 AP/DP/HIT/EV offsets；新增
  `PersistentRecord.View` 唯讀投影及 signed-word regression。下一步是
  證據化 name/class/resource resolver，再接 normalized party；不可直接
  把 raw `+7/+8` 都當 portrait／character id。
- [x] **RE-SAVE-ENVELOPE-ADAPTER**：新增 `remake/internal/fdsave` typed raw adapter，依 `0x4dbd8/0x4dbb9` 保存 rolling-XOR、u32 byte-sum、四槽 bounds 與 writer／reader 已證實的 metadata `+0..+9`；`+10..+39` 與未投影 roster bytes 保持 opaque，Go Docker round-trip/tamper/bounds regression 通過。
- [x] **RE-SAVE-WRITE-SLOT-30012**：官方 IDA 9.4 閉合 `0x30012` confirmed-slot write order：2560-byte roster→`record+0`、metadata `+0..+9` globals→record、checksum over `0x59c7` bytes、rolling-XOR、完整 `0x59cb` write。新增 `fdsave.WriteSlot` opaque replacement adapter/regression；仍不宣稱 native roster/opaque metadata 已接入 remake campaign。
- [x] **NATIVE-CHAPTER-SLOT-RESTORE**：官方 IDA 9.4 固定 `0x30012` 只由 `0x2ccb6/0x2fd93` 戰間流程呼叫；新增雜湊綁定的 `native_intermission_gate.json`、side-effect-free restore plan 與 production title owner。完整 `campaign_full` raw chapter 1..29 均驗證落到既有 town／preparation；合成有效槽驗證 typed party、gold、raw option bytes 保存及錯誤無部分 mutation。這是 E1，不取代一般玩家有效槽 E2，也不接 CONTINUE。
- [x] **音色合成評估+MT-32實證**(SoundFont/MT-32/版本切換,munt渲染15首)→ `16`
- [x] **擴充劇本/玩法可行性評估**(戰場/對話/商店/機制)→ `17`
- [~] SoundFont/MT-32 → 見 `16`(MT-32 已渲染);SoundFont 試聽 + TIMB 配器對映待補
- [ ] 選定首個重製技術棧做「讀真資料 → 畫面」垂直切片
- [ ] 反組譯完整性盤點

## 重製前置(規劃/實作)
- [x] **音樂預錄 OGG**(MT-32 音源):15 首 → 本機 `extracted/music_ogg/`;`tools/export_music_ogg.sh`
- [x] **字型現代化規劃**(UTF-8 + TTF render)→ `18`(計畫:文字資料化 + TTF + 雙字型模式)
- [x] **劇本/關卡腳本系統設計**(可分支節點圖/敗北路線/商店/旗標)→ `19` + `docs/data/campaign_sample.json`
- [ ] 實作:`decode_story_text.py --script-json`(35 章 → UTF-8 script);重製文字層 TTF render
- [ ] 實作:從原版資料自動生成「線性 campaign.json」(parse_field + 劇情 + 商店)→ 原版模式
- [ ] 實作:引擎 ScenarioRunner 狀態機(節點/轉場/旗標)
- [x] **第一性原理可行性確認** → `20`(9 項必要能力全具備,降為工程整合)
- [x] **Go/Ebiten 重製架構規劃** → `21`(桌面/Web/手機)
- [x] **重製 MVP 垂直切片**:Ebiten 載入序章地圖+渲染+游標(方向鍵/WASD/觸控)→ `remake/`
- [x] **技術驗證報告** → `22`(桌面 ELF 10.8MB + WASM 10.5MB + 資產管線,三項全通)

---

# 重製 worklist(Go/Ebiten,本機優先;依序執行)

> 現行工程入口見 [`docs/ENGINEERING.md`](../ENGINEERING.md)；有效工作只看本檔檔首，
> 下列里程碑為歷史工作記錄。

> 當時策略：**先把本機桌面執行檔做成能完整玩，再回頭處理網頁／手機打包**。
> 這是歷史里程碑，不是目前跨平台完成宣稱；現行驗收以 `58`、`57` 與 `91` 檔首為準。

## M0 — 引擎骨架 ✅(已完成)
- [x] Ebiten 專案 + Docker 建置流程(`remake/`,`go.mod`/`go.sum`)
- [x] 載入地圖渲染(tileset.png + map.json,offset 表定位無漂移)
- [x] 游標移動 + 相機跟隨(方向鍵 / WASD / 觸控)
- [x] hi-res 畫布(640×400,CJK 拉畫布不縮字)
- [x] **本機桌面執行檔建成(Linux ELF)** + WASM 編譯成功 → `22`

## M1 — 戰棋核心(下一個,讓它「能玩一場戰鬥」)
> 驗收:能部署我方、選單下指令、移動(flood-fill 範圍)、攻擊結算傷害、敵方 AI 回合、判定勝敗。
- [x] 資料模型:Unit(HP/攻防/移動力/陣營/位置/alive/acted)、BattleState(回合/單位) → `remake/internal/battle/model.go`
- [x] 單位資料管線 `tools/export_units.py`(roster+座標+EXE數值→units.json)+ 引擎載入並渲染(陣營色塊+HP bar+選中資訊)+ headless test 全綠
- [x] 移動:flood-fill 可達範圍 + 高亮 + 選取/移動/待機(`move.go`);地形成本待接
- [x] **地圖單位 sprite=FDICON Q版小人**(24×24 待機動畫)→ `31`(取代誤用的 FIGANI 全身)
- [~] 戰場選單狀態機(移動/攻擊/待機/道具/結束),對齊 `13`(游標/Enter/ESC)：原始 action wrapper、四向環、command grid、物品列與目標游標已有可編輯／失敗即關閉切片；完整 end-turn 入口、indexed effect presentation、原版畫面差分仍未關閉。
- [~] 攻擊結算:玩家選單路徑現在以注入的遊戲亂數呼叫 `AttackWithRNG`，保留未命中／暴擊／經驗結果並交給演出與訊息層；公式仍以**青衫公式**(物理/劍技/法術/恢復+命中+暴擊+經驗,doc 02 §4 = 實作依據)+ EXE 數值表(`03`)為目前重製依據，原版 raw 攻擊 ABI／一般玩家 E2 尚未關閉。
- [~] 敵方 AI 回合：normalized flood-fill/評分與 raw evidence 對照；`aiStep` 現與玩家攻擊共用注入亂數的型別結算邊界，但仍是 normalized approximation。舊 `0x15140` 地址已由 canonical recheck 撤回。`0x13A9F/0x14EF0/0x15B77` 的 dispatcher／尾端路由／score slices 已各自有 evidence/adapter，mode 11 與 `0x13FD4` 也已有 E1 窄 owner；完整權重、turn/camp、target selection、未閉合效果與一般玩家 native runtime E2 仍待補證。`0x1598A` 使用 `0x14818`、`0x1567E` 的 item-command spell branch 才使用 `0x149F8`；raw `+0x22..+0x27` 不命名 AP/DP/HIT/status。2026-08-11 已補 mode 2 的 game-layer `aiStep` E1 夾具：完整 movement provenance 可實際走到 FIGANI 攻擊 owner 並完成回合，缺 movement rows 則失敗即關閉；這不提升為原版 E2。
- [x] **RE-AI-MODE2-AISTEP-CONSUMER-E1**：`TestAIStepConsumesVerifiedMode2PhysicalPlan` 以完整原始記錄、構成／地形位元組、道具資料列、移動成本資料列及 FIGANI 資產，實際驗證 mode 2 `NextAIPlan`→`aiStep`→行走→攻擊演出擁有者→回合完成；`TestAIStepStopsMode2WithoutMovementProvenance` 驗證缺少移動來源證據時不會標記單位已行動。這是重製端 E1 消費端回歸，不是原版目標選擇語意或一般玩家 E2。
- [~] 敵方 AI 雙預選 bridge：`BuildNativeAIPhaseDiagnosticPlan` 已依
  `0x1D8BA` 原序將 `0x1598A→0x15B77→[0x53C23]` 與
  `0x1567E→0x15880→[0x53C33]` 兩個具型別 producer 接入
  `PlanNativePhaseUnitScans` 的 signed `>=6` 門檻；每個合格 selector-zero
  單位都要求明確成本列，缺漏／重複／額外輸入失敗即關閉。map0 修改狀態的
  E0 交叉夾具固定96／8並保留第二遍，且驗證輸入記錄不變。它不執行
  `0x13A9F`、回呼表或 pending code；尚缺同一原版 runtime 動態 trace，
  故仍不接正式 `NextAIPlan`。
- [x] **RE-AI-COMPOSITION-EVENT-BYTES-ALL-MAPS**：修正同步工具漏掉
  FDFIELD 構成格 `+2` 的管線缺口，同時撤回把它直接命名成完整 target
  flags 的錯誤斷言。33 張 map 現均同步
  `native_composition_event_bytes`；`0x4DBFC` 的 low5 基底與
  `0x145CD→0x14625/0x146A7` 的 caller-specific `0x40/0x80` writer 已分層。
  合法 IDA Pro 9.4 已優先確認函式邊界與交叉參照，Capstone 再覆核
  直接指令；`0x4E040` 只建立搜尋狀態，實際旗標消費端是 `0x4E16E`。
  證據保存於
  [`fd2_field_composition_lifecycle_disasm.txt`](../data/fd2_field_composition_lifecycle_disasm.txt)。
  map19 1600格中7格非零，真實 unit55 的兩個 producer 均為零分且不創造
  勝者。`0x145CD` 直接呼叫者的短生命週期執行期旗標（live flags）仍維持
  明確傳入。
- [x] **RE-COMMAND-FLAG-LIFETIME**：合法 IDA Pro 9.4 優先確認
  `0x1598A/0x1567E/0x1CFF0/0x1BBDC` 的候選生命週期，Capstone 再覆核每次
  `0x4E040/0x14818` 後都呼叫 `0x4DBFC`，且這些函式沒有呼叫 `0x145CD`。
  `State.NativeTargetFlags` 已刪除；正式命令改由
  `NativeCompositionEventBytes` 每次重建獨立低五位切片（low5 slice），測試鎖定跨呼叫
  不共享 mutation，缺來源仍失敗即關閉。直接證據見
  [`fd2_ai_composition_flag_lifetime_disasm.txt`](../data/fd2_ai_composition_flag_lifetime_disasm.txt)。
- [x] **RE-AI-RAW-RECORD-1598A**：`NativeAIScoringRecords` 以完整來源建立分離的 `0x50` runtime 快照，補齊 presentation、`+5/+6/+34..+36/+42/+46` 並拒絕不完整 roster；map0 與 map19 真實資產錨點通過。
- [x] **RE-AI-CANDIDATES-1598A**：`NativeAIScoredCommandCandidateGroups` 已以 command `+3/+4`、原版 cost row、exact grid flags 與 raw `+5/+6` 建立 row-major destination/target-index groups；selector target-code transform 與空 target skip 已保存，map0 identity103→ally `(23,14)` 真實資產 regression 通過。群組、單位級最大分數與三遍門檻均已接入唯讀診斷；下一步是原版同狀態動態 trace。
- [x] **RE-AI-SCORE-GROUPS-15B77**：`ScoreNativeAIScoredCommandGroups` 已依完整 ID 家族分派攻擊、恢復、旗標與原始零分支；map0 command0 的四個友軍目標各得24、群組合計96。IDs10..12 缺 `0x1F183` caller gate 時拒絕執行。`[0x53C23]` 的數值最大值可由零開始比較，但零分時的命令字區域變數初值仍未知，尚不可聲稱已閉合勝者。
- [x] **RE-AI-UNIT-SCORE-1598A**：`ScoreNativeAI1598A` 已串接命令可用性→候選幾何→群組評分→正分 strict winner，並交叉驗證 actor mask／MP 與 runtime record；map0 command0 固定最大分96與 `(23,14)`，全零分不創造勝者。純數值結果已接到三遍 phase planner 的 `[0x53C23]` 證據欄；正式動作執行、逐單位回呼與 pending early-exit 仍未接。
- [x] **RE-AI-MAP-ASSET-INPUTS**：撤回「地圖已有初始命令遮罩」的錯誤前提；同步前 33 張圖共 1887 個單位全數缺欄位。現在已由 FDFIELD b13..b16 補齊，並依 `0x10d7f..0x1100c` 同步 constructor `word42/word46`；1885 筆具有完整 MP 來源，map32 兩筆未覆蓋 selector 保持缺值。現有 263 筆非零遮罩中，261 筆通過原始 MP gate。
- [x] **RE-AI-SPELL-SCORE-15B77**：Docker Capstone/Hex-Rays 釘死 `0x15b77` 的 attack IDs0..12 score（HP `<` spell value→24，否則8；record `+0x08==0` 時乘 1.5 並 toward-zero）與 recovery IDs13..16 score（HP `<` max/3→8、否則 `<` max/2→3、否則0；`+0x34 bit0` 再×2）；新增 raw-only `ScoreNativeAISpellAttack`／`ScoreNativeAISpellRecovery`，ID10..12 嚴格要求 caller-supplied `0x1f183` gate。未接 AI runtime、command inventory、target UI 或效果名稱。
- [x] **RE-AI-SPELL-FLAGS-15B77**：Hex-Rays 釘死 ID20/21→raw `+0x25/+0x26` nonzero flag score，每筆各加6；ID26/27→同兩 offsets zero flag score，每筆各加4；新增 `ScoreNativeAISpellFlag`／`ScoreNativeAISpellZeroFlag`，不清除、不命名 flag，也不接施法 runtime。
- [x] **RE-AI-SPELL-MODIFIERS-15B77**：Hex-Rays 釘死 ID17/18/19→raw `+0x22/+0x23/+0x24` zero flag score，每筆各加3；`ScoreNativeAISpellZeroFlag` 保存該 raw helper，未命名 transient 欄位或接 AI runtime。
- [x] **RE-AI-DISPATCH-1598A**：合法 Hex-Rays pseudocode 釘死 `0x1598a` 的 caller order：`+0x27==0` gate→`0x1c269` command bytes→record `+5 <= unit+0x44` MP gate→target resolver→`0x15b77` score；最高 score 勝，平手比較 command record `+0`，再保存 raw `(x,y,command)`。新增 `SelectNativeAISpellCandidate`，只做 score/tie-break，不接 MP、target、UI 或施法執行。
- [x] **RE-AI-DISPATCH-GATE-1598A**：`NativeAvailableAICommandIDs` 將 `0x1598a` 的 raw `+0x27==0` gate 加在既有 bounded command scan 前；unknown physical IDs36..39 仍 fail-closed，不接 target resolver 或 runtime action。
- [x] **RE-AI-SPELL-ID22-15B77**：`0x15d30` 先 gate raw `+0x27==0`，再呼 `0x1c269(unit,nil)` 掃 `+0x1a..+0x1e` 五 bytes；任一 bit set 即累加6。新增 `ScoreNativeAISpell22`，不命名欄位、不接 ID22 effect/status runtime。
- [ ] 勝敗判定 + **回合推進(回合無上限;上限只由劇本事件 turn>=N 設定,見 `27`§1)**
- [ ] headless 確定性回歸:固定種子打一場 → 結果可重現(驗演算法,不靠手玩)

## M2 — 文字 / 對話層
> 驗收:對話框能顯示 UTF-8 劇情、帶頭像、翻頁;字用 TTF render(不再靠點陣字模)。
- [ ] 工具:`decode_story_text.py --script-json`(35 章 → UTF-8 `script.json`,控制碼→結構)
- [ ] 引擎 TTF 文字渲染(接 `18` 字型現代化:資料化 + TTF + 雙字型模式)
- [x] 對話框 UI ✅(debd52d):原版框素材(LMI1 #21 310×99)+ orig 佈局(下框(5,112)@320/上框鏡射)+
      大側臉頭像(我方左面右/對方右鏡像面左,對映 0x4E8AF/0x4E8E1)+ 白字『』框內換行(≤3行);
      翻頁=campaign story 逐句 Enter。LMI1 #20=單位詳細狀態面板(待用)
- [~] DATO 頭像接入：已新增 `internal/dato.MouthState`，以 native `0x16D00` 的每 2 frame tick、開嘴 1 tick、閉嘴 `rand()%30+2` cadence 驅動 m0/m3；完整 DATO frame/grid、speaker layout 與 runtime dialogue parity 仍未閉合。

## M3 — 音訊層
> 驗收:戰鬥/城鎮/劇情切場景時 BGM 正確切換,用預錄 OGG(MT-32 音源)。
- [ ] OGG 串流播放(15 首,來源 `extracted/music_ogg/`)
- [ ] 場景→曲號對映(對齊 `12`,play_bgm 邏輯)
- [ ] (選配)SoundFont/MT-32 版本切換開關 → `16`
- [ ] 音效(SFX)接入

## M4 — 腳本系統 / 流程串接
> 驗收:序章→商店→分支→下一關 能一條龍跑完;戰敗走不同路線而非 game over。
- [ ] 工具:原版資料自動生成「線性 campaign.json」(parse_field + 劇情 + 商店)→ 原版模式
- [ ] 引擎 ScenarioRunner 狀態機(節點/轉場/旗標),對齊 `19` + `campaign_sample.json`
- [~] 商店節點：目前可編輯 `item.json`／shops fixture 保存 215 個 numeric item ID（0..214）與價格；較早「337 筆商品」說法無現行 fixture 支持，已撤回。祕密商店與 town 回返已接、`ClassID`／item type／class equip 白名單、指定收件者與兩階段裝備 prompt 已接；賣出 UI 已接成「Tab→角色→欄位」，`SellSlot` 鎖定原價 75 折並同步移除 equipped flag；`0x1145a/0x1c142` RE 已接入 base+flag 重算與 `<0x80`/`>=0x80` 同類替換；raw `inventory_slots` 保留 source 8 bytes，Load/PartyUnits 依 `0x10f06..0x10f31` materialize 成 runtime 8 slots，內部空槽不再錯移。runtime `0x602ad` item table 的完整邊界／215 rows 對應仍未閉合，故不把它當作 attack UI 的真相；`0x14237→0x14818` 僅鎖定 caller-specific geometry 用途；待：完整 item multiplier/效果碼與原版 UI 對照。
- [~] 戰後 town/整備流程：campaign_full 的 postbattle→town、連戰 preparation 路線與 shop/rumor return 已盤點；城鎮 `0x2d093` 是進戰場確認／小名冊略過選人／取消回城，無城鎮 `0x2cad7` 則是記錄詢問後必進選人；兩路共用 `0x318ad` 的30-byte全零勾選表、一般cap15／late cap19。重製已接分流及 `partyDeploy`，永久 JOIN roster 不被改寫；選人面板仍是重製殼層，尚非原版介面。church `0x3072f` 已證實四個 raw index→address dispatch（不是四個已命名服務）；`0x2d7bd` 只接受左右鍵並在四項循環。revive fee table、原子 `ReviveUnit`、church selector 與 class-change 候選→唯一 target→確認 mutation 已接；尚待 indexed renderer 與原版數值對照（無免費一般治療）。
- [~] 戰後 town/整備流程：preparation 與 church selector UI 已接；`docs/figures/church-selector.png` 為 xvfb 實機畫面。revive 與 class-change 單一 target mutation 已可保存 roster/gold；尚待完整 xvfb 轉職操作，以及原版 `+0x22/+0x23/+0x24` DX/race/multiplier 欄位資料化。
- [~] class-change church：已鎖定 `0x3151a..0x3152d` portrait→item gate、`0x31860` inventory 掃描、`0x1b8e7` item 移除與 `0x31571..0x3157a` class/portrait 寫回；`0x526a7` mapping、`0x2a2e8` 成長重算與 editable target resolution 已接，待 raw race/multiplier 欄位與實機回歸。
- [~] class-change church：`class_change_targets.json` 已校正為兩層可編輯資料：current portrait 0..0x11→default target 與 optional/special override inputs（`0x526a7` 以 current portrait 索引，raw `0xff` 不啟用 optional override），以及 target portrait 0x20..0x41→class/mobility increment (`0x615fe`)；portrait 9 持 item 0x5a 時覆寫為 target 0x34。這些不是玩家同時可選的分支。
- [~] class-change church：核心 `campaign.ApplyClassChange` 已依 `0x31602` 實作可重現 RNG（row `[min,max)`）、將新職 AP/DP/DX/MaxHP/MaxMP growth **累加**既有值、MV(+0x3b) 累加、保留 Lv、清 EXP、HP/MP 回滿與轉職道具移除；persistent party 已同步保存 MV。自動 target 解析與左右 Yes/No confirmation 已接，仍需原版實機數值回歸。
- [~] campaign town/shop 外部交叉盤點：攻略頁明列第4、7、9、14、16、18、19、21章的武器店／道具店／教會／神秘店（來源連結已記入 handoff）；不能由攻略頁推論戰後立即順序，仍以 EXE table 與 `campaign_full.json` 的 postbattle→town/preparation 節點為準，後續測試不得把勝利直接串成下一戰。
- [x] ch02/ch03 story handler slices：ch02_pre/ch02_post 依 count-aligned scene line 範圍播放；ch03_post 接已證實的 ch04 scene3 lines0–3；ch03_pre 已由 jump-table/loadch/FDTXT_004 direct evidence 完成 binding，idx0→scene0 lines0–3、idx1→scene1 lines0–4，`story_ch04` 不再只播兩句 generic fallback。
- [x] ch04/ch05 pre-handler slice：`ch04_pre` 的 FDTXT_005 idx0/1/2 已接 `ch05.json` scene0/1（3+3+9句），map4 50-slot、pan、acting22/21 皆有 binding；`story_ch05` 已由空 cutscene 接回可編輯 handler。
- [x] ch05/ch06 pre-handler：新增 `HandlerDialog.Segments[]` 跨-scene adapter，依 FDTXT_006 #0 的 scene0→1→2→3 targets 展開 18 句；`ch05_pre` 完整 binding，`story_ch06` 已接回可編輯 handler。
- [x] ch06/ch07 pre-handler：FDTXT_007 index0/1（2+6句）與 map6/acting28/29 已接 binding，`story_ch07` 已接回 editable handler。
- [x] ch07/ch08 pre-handler：FDTXT_008 index0/1（跨 scene 15句+2句）與 map7/acting31/32 已接 binding，`story_ch08` 已接回 editable handler。
- [x] ch08/ch09 pre-handler：FDTXT_009 index0/1（2+5句）與 map8/acting35 已接 binding，`story_ch09` 已接回 editable handler。
- [x] ch09/ch10 pre-handler：FDTXT_010 index0 跨 scene0/1（6+6句）與 map9/60-slot 已接 binding，`story_ch10` 已接回 editable handler。
- [x] ch10/ch11 pre-handler：FDTXT_011 index0 跨 scene0/1/2（4+6+2句）、index1/2 scene2 延續，map10/acting38/39 已接 binding，`story_ch11` 已接回 editable handler。
- [x] ch11/ch12 pre-handler：FDTXT_012 index0 跨 scene0/1（2+9句）與 map11/acting40/41 已接 binding，`story_ch12` 已接回 editable handler。
- [x] ch12/ch13 pre-handler：FDTXT_013 index0（6句）與 map12/70-slot 已接 binding，`story_ch13` 已接回 editable handler。
- [x] ch13/ch14 pre-handler：FDTXT_014 index0（4句）與 map13/70-slot、pan 20,20 已接 binding，`story_ch14` 已接回 editable handler。
- [~] ch14/ch15 handler：Docker Capstone 已證實 pre `0x334f3..0x334f7` 的 `roster_has(12)`→FDTXT_015「有 12：0/1/2；無：3/4/5」，以及 raw `ch14_post` `0x239d1..0x239d3` 的「有：12；無：13」。主迴圈與前三戰既有測試證實玩家戰鬥 N 對應 raw `ch(N-1)_post`；因此 `postbattle_ch14_persist` 應使用 `ch13_post`，`postbattle_ch15_persist` 才使用 `ch14_post`（JOIN15→set_chapter15→town_ch16）。pre binding 含 map14/80-slot、pan、acting48；runtime 只讀 permanent party roster，缺此資料 fail-closed。
- [x] **ch14/ch15 postbattle campaign index correction**：撤回「同號 postbattle node 對同號 raw post handler」的錯誤斷言；目前已驗證並回歸 `postbattle_ch14_persist→ch13_post→town_ch15`、`postbattle_ch15_persist→ch14_post→town_ch16`。其他仍採同號 binding 的章節必須逐一用直接指令複核，不能機械式整批平移。
- [x] **既有 postbattle 索引錯接稽核**：稽核工具現將 active binding 與已證實的 `battle N→raw ch(N-1)_post` 關係逐筆比較，不再把「欄位非空」當成 active 正確。原有13個同號錯接已清除；IDA 直接指令再閉合 raw ch25→玩家ch26、raw ch27→玩家ch28、raw ch05→玩家ch06、raw ch06→玩家ch07、raw ch07→玩家ch08、raw ch09→玩家ch10、raw ch12→玩家ch13、raw ch16→玩家ch17、raw ch17→玩家ch18與raw ch19→玩家ch20。ch24 candidate 曾因錯誤86-slot拓撲從正式 binding撤回；2026-08-13以62→70→71追加式runtime重新閉合並恢復正確玩家第25戰owner。歷史統計保留，現況以本文件頂端與`58`為準。
- [x] **玩家第 8 戰 raw ch07 post 垂直切片（E1）**：撤回 `ch08.json` 把 groups 1／8／9／10 預先 materialize 的舊設定。`0x1088D` 只建立 party＋group0（10＋19＝29 slots），event27 回合2..7才逐組追加兩筆，合法戰後 frontiers為29..41的奇數。address-keyed binding 已接 slot28 raw JOIN5身分、layout、ACTING33／34、FDTXT_008 index3／4、精確全黑與 framebuffer clear，再走JOIN5／sync／chapter8進 `town_ch09`；負向測試拒絕其他 `0x11D40` call site／參數。event28 slots10..27 raw `+0x34 &= 0x80` 的正式回合接線及 DOSBox E2仍待完成。證據見 [`fd2_ch07_post_ida.txt`](../data/ida/fd2_ch07_post_ida.txt)。
- [x] **玩家第 10 戰 raw ch09 post 垂直切片（E1）**：IDA Pro 9.4 與 Docker Capstone 固定 `0x235F9..0x23790`。正式 binding 保留有／無凱麗造成的60／61兩種強推論 frontier，依位址執行 delta 0→63 共64次 DAC 淡出、只寫明列 offset 的 sparse record/view patch、delta 64→0 共65次 DAC 淡入、FDTXT_010 index4／5的19＋16句、ACTING37、JOIN11／6、sync與chapter10，最後進 `town_ch11`。兩個frontier均以正式輸入完成35句並通過存讀檔；執行期對非法值、缺 raw provenance 或 frontier 不足會在任何寫入前原子拒絕。尚缺未修改 DOSBox 一般玩家逐幀 E2。證據見 [`fd2_ch09_post_ida.txt`](../data/ida/fd2_ch09_post_ida.txt)與[`fd2_ch09_post_native_dialogue.md`](../data/ida/fd2_ch09_post_native_dialogue.md)。
- [ ] **ch00 `0x3241f` 原生淡入閉合**：追查 map32 runtime roster 的 raw FDICON key producer，讓 title/story indexed compositor 不再依賴 `Fig==key` 假設；完成前只保留此 exact call site 明示的 RGBA E1 可玩近似，不得泛化為 `0x1F525` fallback。
- [x] **raw ch15 postbattle layout audit（玩家第 16 戰）**：Docker Capstone 與合法 IDA Pro 9.4 固定 `0x23a0a→0x233c6` 的 slots `0..15`、special raw slot65=`(28,30,pose2)`、camera raw `(22,25)`，X=`[28,27,28,29,30,25,26,27,26,29,30,31,25,26,30,31]`、Y=`[28,27,27,27,27,28,28,28,27,28,28,28,29,29,29,29]`、constant pose0；acting resource49 只有 slot65 pose0/5 beats。IDA 另證實 `sub_3453E(index)` 讀 `runtime[index*0x50+5]&1`，故 handler 的 `0x42..0x49` 確為 slots66..73。`sub_1088D` 直接指令與 map15 資料固定入口為16個 persistent slots加60個 group0 rows，即76 slots；四條分支、JOIN18、`town_ch17` 與 town save/load 已通過 Docker/Xvfb E1，未修改一般玩家 DOSBox E2 仍待。
- [x] **native inactive-count condition primitive**：新增 `native_inactive_count_gt` editable condition，compiler 僅接受明確 slots、非負 threshold 並保留 raw byte5 bit0；runtime 對缺 slot／缺 raw provenance 直接 fail-closed，測試覆蓋 count 5/4 與缺 raw。這只提供 ch15 branch 所需的一個純條件 primitive，未替 `[0x53bef] >18` 或 record `+0x42 >=0x140`。
- [x] **ch15 raw round/word provenance**：Docker Capstone 確認 `[0x53bef]` writer=`0x1a5b9`、ch15 gate 嚴格 `>0x12`，以及 `[0x53a45]+0x42` raw u16 gate `>=0x140`；新增 `NativeRoundCounter`、`NativeRecordWord42` 與 `native_round_gt`／`native_record_word_gte` strict compiler/runtime regression，不再把 raw word 直接命名成 normalized MaxHP。
- [x] **raw ch15 runtime roster producer**：合法 IDA Pro 9.4 直接指令證實 `sub_320FC` 只重排完整 persistent records、不改總數；`sub_1088D` 先逐筆複製 persistent roster，再以 `sub_10B4E(0)` 逐列附加 raw group0，沒有 fig15 的略過、替換或提升分支。raw ch15 的擁有者是玩家第16戰，故應對照 map15：16個 persistent slots加60筆 group0，入口恰為76 slots；撤回較早以 map14 推導74／78的假說。
- [x] **ch15 +0x42/+0x46 constructor producer/export**：Docker Capstone 固定 constructor 的 HP／MP 公式與 `+0x40/+0x42`、`+0x44/+0x46` 寫入；新增 `native_record_word42/46` 投影並同步 33 張地圖。具備來源的列以原始值初始化 HP/MP，舊列沿用正規化數值；格式錯誤或未覆蓋 selector 維持缺值與失敗即關閉。
- [x] **ch15 compound predicate primitive**：新增受限 `native_any_of`，只接受已驗證的 raw round／inactive-count 子條件；任一子條件可證實為真才通過，全部缺 provenance 仍 fail-closed，並已用於 production `ch15_post` handler；未修改一般玩家 DOSBox E2 仍另列為獨立 gate。
- [x] **ch15 +0x42 persistence bridge**：`syncPartyFromBattle` snapshot 與 `applyPersistentStats` 現保留 `NativeRecordWord42/HasNativeRecordWord42`，測試覆蓋 raw word `0x140`；不由 normalized HP 推導，units source 缺 constructor provenance 時仍 fail-closed。
- [x] **ch15 candidate／production editable CFG**：保留 `handlers/candidates/ch15_post_cfg.json` 的 raw source addresses 與 nested OR/else branch；2026-08-02 IDA 重核修正 JOIN18 錯置：`0x23b1f` 跳到章節尾端，JOIN18 只屬於 `else word42>=0x140` arm。正式 `handlers/ch15_post.json`／binding 已依同一證據接入 production，candidate 僅作研究對照，不是另一套執行入口。
- [x] 執行 raw ch15 四條 branch trace，補 JOIN18 當下的 typed persistent record、`battle_ch16→postbattle_ch16→town_ch17` 與 save regression；四條路徑已通過 Docker/Xvfb E1，一般玩家原版 runtime capture 仍是獨立 E2 gate。
- [x] ch16/ch17 pre-handler：`0x335bb` 的 `roster_has(18)` 接 `test/jne 0x3344d`；有角色18直接進 shared tail，沒有才 `spawn(group 1)`。已轉為 editable `if roster_has`，map16/60-slot/FDTXT_017 binding 接入 `story_ch17`。compiler branch 現繼承前置 LOADCH slot frontier，但 merge 後不假設分支新增 slots。
- [x] ch17 battle initial-group correction：原版 ch16 pre 只在 char18 缺席時 append group1，group3 是 ch16 post 才 spawn；`ch17.json` 不再把 1/3 固定 initial。Scenario 加入可編輯 `initial_groups_if_party_absent`，只控制戰前 `OnField` visibility；它不宣稱已還原 native append-slot identity。raw ch16 post 現已由正式 binding 消費，不能把 `ch17_post` 錯接到第17戰。
- [x] **raw ch16 post／玩家第17戰 E1**：IDA Pro 9.4 固定 handler `0x23b5f..0x23cd5` 的 roster_has(18) 分支、共用 ACTING call site 的 immediate 分流、layout table、PAN、FDTXT_017 index5–8 及 JOIN16；資源50–53已轉為可編輯演出稿。`postbattle_ch17_persist` 現以 60／61 slot-count binding 接入，兩條分支分別驗證 61／62 戰後 frontier、`town_ch18` 與 save/load；原始位址與雜湊見 [`fd2_ch16_post_ida.txt`](../data/fd2_ch16_post_ida.txt)。本切片為 E1，未修改一般玩家 DOSBox E2 仍待。
- [x] ch17/ch18 pre-handler：FDTXT_018 index0/1/2（7+4+13句）與 map17/70-slot、acting54/55 已接 binding，`story_ch18` 已接回 editable handler。
- [x] **raw ch17 post／玩家第18戰 E1**：`0x23cd5` 的 layout、ACTING56/57/58、FDTXT_018 index7/8/9/10（index10 以兩段 scene mapping）與 JOIN21/7 已轉為可編輯 binding；map17 55-slot runtime、`town_ch19`、save/load 均通過 Docker/Xvfb。`native_join_base_units.json` 只列出 map17 的 raw +8=7/21 base，證據等級為強推論；不存在明示 base 的 JOIN 不得自動猜測。證據見 [`fd2_ch17_post_ida.txt`](../data/fd2_ch17_post_ida.txt)。
- [x] ch18/ch19 pre-handler：`ch18_pre` 實際 index0（8句）與 map18/70-slot 已接 binding，`story_ch19` 已接回 editable handler；未把未呼叫的 FDTXT_019 其他 strings 硬播。
- [x] ch19/ch20 pre-handler：FDTXT_020 index0（17句）與 map19/70-slot 已接 binding，`story_ch20` 已接回 editable handler。
- [x] **玩家第20戰 raw ch19 post 垂直切片（E1）**：IDA Pro 9.4與Docker Capstone固定`0x23E74..0x240FA`；四張table精確寫slots0..15、52..60與view `(26,31)`。固定record0＋選15人、map19 group0共67筆，形成83-slot入口；round15執行group1→84、ACTING60–62、index14–16與JOIN28，round16依`0x24005 jg`全部略過；JOIN25、index13、chapter20為共同路徑。舊scenario預載70筆再加16人的86-slot拓撲已撤回，改用runtime append只建立group0。正式binding已接`postbattle_ch20_persist→town_ch21`，兩條分支、persistent JOIN與城鎮銜接回歸通過；尚缺未修改一般玩家DOSBox E2。現況稽核統計以本文件頂端為準。證據見[`fd2_ch19_post_ida.txt`](../data/ida/fd2_ch19_post_ida.txt)。
- [x] ch20/ch21 pre-handler：FDTXT_021 index0（17句）與 map20/80-slot 已接 binding，`story_ch21` 已接回 editable handler。
- [~] class-change data/UI bridge：`LoadClassChangeTable`、`NativeClassChangeTarget`、`LoadClassChangeGrowth` 已接；church 先在三列視窗選角色，再依 special>optional>default 自動解析唯一 target，最後以左右 Yes/No confirmation 決定 mutation。`0x31019` row、FDOTHER#14 entry16 panel、`0x1974c` 六幀 opening，以及 `0x19953/0x197e5` 動態名字＋FDOTHER#2 Yes/No normal/pulse＋四幀 choice open/close 均已成 fail-closed indexed compositor。official IDA 重核補正完整順序：候選清單先以 `0x2d31b` 五幀關閉＋source restore；確認結束再跑 choice close 4 幀、dialogue close 5 幀＋source restore，才執行 mutation／返回。每幀都由 Draw acknowledgement 推進。`0x19953` BIOS delta>=2 時 counter mod4 前進、選中 variant=counter/2。已重生 [`native-class-list-indexed.png`](../figures/native-class-list-indexed.png)／[`native-class-confirm-indexed.png`](../figures/native-class-confirm-indexed.png)，兩者現在包含原版教會 scene source；raw service0 status/command renderer、HIT/EV/DX 實機數值差分仍待。
- [~] class-change synthesis：`0x31602` 五組 `0x1e529` 先把新職成長加到 raw AP/DP/DX/MaxHP/MaxMP，隨後呼叫 `0x1b750`；該 routine 讀 raw `+0x37/+0x39/+0x3e`、item table 23-byte row 的 `+1/+3/+5/+7`，寫 derived AP/DP/HIT/EV `+0x48/+0x4a/+0x4c/+0x4e`。`RecomputeAfterClassChange` 已恢復並防止既有裝備重複計算；`+0x22/+0x23/+0x24` 是 constructor 清零後由其他 transient/effect writer 使用的旗標，class path 本身不寫入，不能臆測成 class modifiers。
- [~] headless class-change fixture：`FD2_CAMP_CLASS_FIXTURE=1` 現正確注入 native identity/portrait9 的悠妮（先前誤寫索爾已修正）＋item0x58/0x5a，供 xvfb 驗證「教會→轉職→角色→自動解析 special target0x34→Yes/No」；正常遊戲不改變。舊 `church-class-targets.png` 是 remake 自創三分支選單，已撤回並刪除。
- [~] 分支與敗北路線：campaign runner 已有 on_lose→retreat 非 game-over 路徑與測試；battle Node 新增可編輯 `protect` 目標（空值沿用索爾），main 不再硬編碼唯一保護角色；待逐關核對原版保護目標與 retreat 後整備語意。
- [~] 存檔/讀檔(自有格式,非破解原版 `FD2.SAV`)：節點／旗標／金幣／道具／persistent party 已保存；2026-07-20 新增同目錄暫存檔+rename 原子寫入與清理測試，避免 town/shop/preparation 存檔被截斷。仍待完整 GUI/Xvfb 讀檔回歸。

## M5 — 內容完整化(原版可破關)
> 驗收:從序章玩到結局,全 33 戰場 + 全劇情 + 商店,正常玩法可達(無 debug hook)。
- [ ] 匯出全 33 戰場為引擎資產 + 全單位/數值表接入(對齊 EXE 表 `03`)
- [ ] 全劇情/對話接入(35 章)
- [ ] 完整性盤點:對照原版,缺漏列冊(`83` 完整性 > 投報)
- [ ] 正常玩法可達性驗證(連通/可破關鏈,參考 skill 踩雷)

## M6 — 跨平台打包(回頭做網頁/手機)
> 驗收:Windows/macOS/Linux 桌面包 + 網頁 + Android APK。
- [ ] 桌面交叉編譯 + 打包(Windows `.exe` / macOS `.app` / Linux AppImage)
- [ ] WASM 上網頁(資產載入 + `index.html` 完整化)
- [ ] Android:`ebitenmobile bind` → `.aar` → Gradle APK(觸控已支援)
- [ ] 玩家向 README(圖文並茂,突顯貢獻)+ 工程文件分離

## 擴充(M4 之後,擺脫原版固定 33 路線)
- [x] **可擴展事件系統規劃** → `29`:trigger/when/do DSL + 文本事件控制碼 `{{}}`;條件/動作 Registry 可註冊;原版 30 關可表達+自創戰役
- [ ] 實作 EventSystem(ConditionRegistry/ActionRegistry)+ DialoguePlayer 解析 `{{}}`
- [ ] 自創戰場 + 自訂劇本(用 `19`+`29` 系統)
- [ ] 多分支劇情線 / 多結局
- [ ] 編碼器回寫中文(`encode_text.py`)做在地化/二創

## 第 5 輪 ✅(開場流程反組譯 — 使用者指定)
- [x] **建反組譯器** `tools/disasm_le.py`(capstone 解 DOS4GW LE,docker)+ 確認 entry/main/狀態機
- [x] **頂層狀態機反組譯**:真 main=0x25bf4(雙層迴圈),核心狀態變數 `[0x53c03]`=章節,兩張章節跳表(0x51d71 戰前劇情 / 0x51de9 戰後)→ `23`
- [x] **標題序列**:角色立繪 5 幀(FDOTHER #0x45-0x49,320×147)垂直捲動(非旋轉)+ FLAME DRAGON logo(#7 sub0)+ 主選單;**解碼器當 oracle 解圖視覺驗證** → `23`
- [~] **主選單機制**:輸入迴圈/scancode dispatch(↑0x48/↓0x50/Enter/Space)/游標 wrap、return `0=新遊戲`、`1=0x30550` 四槽 selector 已由 Docker Capstone 重跑；第三 return branch 直進 `0x10010`。2026-08-02 合法 IDA 重核已撤回「`0x4E031` 是戰鬥驅動器」：它只複製 BIOS 鍵盤緩衝 word；第三分支返回0後由 main `0x25DCE` 呼叫並循環重入真正的共享控制器 `0x117E7`。remake 也已刪除 selection2 誤讀 JSON slot0 的高風險行為；在未修改一般玩家有效槽 E2、delete/overwrite 與 CONTINUE 的 pending-group／`Game` handoff 完成前，未證實狀態仍停在 title。四槽 LOAD 已接 checksum-valid selector→typed party→town/preparation production owner，故仍不能稱完整 native LOAD/CONTINUE 相容 → `23`、`56`、`57`
- [x] **新遊戲→開場對話→自動進戰場**:[0x53c03] 章節驅動,cutscene 0x3231b(與前代主角對話)→ 戰場地圖=章節*3+2(自動串接)→ `23`
- [x] **call-graph 遞迴反組譯工具** `tools/callgraph_le.py`(可達集/callers/rpath/funcof/jtab)→ `24`
- [x] **cutscene→戰場控制流勘誤**：`0x10010` 真 caller 仍是 `0x1a251/0x26130`，但展開 `0x25ebb` 證實 `0x26130` 只屬第三主選單分支；新遊戲與四槽讀檔各自跑 pre-handler 後從 `0x25ebb` 返回，main `0x25dce` 才呼叫 `0x117e7`。舊「handler ret 後在同 driver 線性落入 `0x10010`」已撤回；callgraph 工具排除 `0x1b051/0x26f30` 偽命中的成果仍有效 → `23`、`24`
- [x] **[0x53ecc] 戰後/事件狀態機**：章節 handler 的 raw code 由戰役迴圈消費；`0x205be` 另有直接閉合的 0/1/2 roster 規則。不得再把 `0x205c9–0x20c64` 寫成單一事件解譯器，或把所有 code1/2 writer 先統稱同一玩法語意 → `24`§6
- [x] **挖完事件 handler 原語** → `25`：第三張章節跳表 `0x51b19`（30章／18個特殊 handler），FD2 事件是每章 C handler 而非 byte-code；`0x3453e` 查 raw bit，`0x205be` 是三值結果規則，`0x205da` 才是重設／章節載入入口。
- [x] **RE-BATTLE-RESULT-205B4-BOUNDARY**：Docker Capstone 與合法 IDA 9.4 交叉確認完整函式起點 `0x205b4`、共享 direct-call 入口 `0x205be`，以及 code2→camp0 active code0→record0 bit0 code1 覆寫順序；`0x205d5→0x2067e` 跳過相鄰 `0x205da`。新增 `NativeBattleResultCode205B4`、map0 真實 roster 錨點與失敗即關閉測試；撤回舊「`0x205be` 清碼並呼 `0x1088d`」混線斷言。
- [x] **逐關挖 18 特殊 handler** → `26` + `tools/event_handler_dump.py` + `docs/data/battle_events.json`(30章條件→動作,供 remake 去 hardcoding)
- [x] **補完 battle-event raw predicates（2026-07-27 勘誤）**：`0x3453e(idx)` 只閉合為 `[0x53a45]+idx*0x50+5 & 1`；remake 的 `unit_inactive` 是 caller-specific projection，不是全域死亡／存活欄位。`0x33499`=roster_has（查 `[0x53bf7]` 我方名冊）。`battle_events.json` skeleton 未記錄 action_fns，不代表 postbattle/cutscene handler 無動作；舊 bit0 高階命名已撤回。
- [~] **handler raw-byte5 runtime bridge**：`0x3453e` raw adapter、constructor、已知 damage/death writer、revive writer 與 `deactivate_unit` 已有 raw propagation/regression；完整 raw roster 時 `cmd/fd2` 的 `any_unit_inactive` 已 strict 讀 raw bit0，只有舊／混合 JSON 才使用 `OnField/Alive` 相容 projection。需補 zero-HP 初始 record／所有 LOADCH 分支並讓 strict binding 缺 raw 時 fail-closed，才可把 ch01/ch02 handler 全面升為 E0。
- [~] **persistent raw-byte5 bridge**：`syncPartyFromBattle`／`applyPersistentStats` 已保存 `NativeRecordByte5/6`，並在 raw bit0 有 provenance 時依 native branch 決定 HP refill；缺 raw 仍保留 E1 projection。需完成 LOADCH raw record materialization 才能移除 fallback。
- [x] **反思日誌補第 7-10 輪** → `99`
- [x] **挖完 `[0x53bf7]` 表語意**:不是 tile,是**我方隊伍名冊**(32槽×0x50B);`0x33499(id)=roster_has(id)` 查 byte[+8]==角色ID(章16 用)→ `25`/`26` 回填;兩單位陣列釐清([0x53a45]96槽全場 / [0x53bf7]32槽名冊)
- [x] **回合計數釐清**:`[0x53bef]`=回合/進度 counter（開始1/inc/cmp N），`[0x53ec8]`=累積計數（非回合）；**修正前輪把 [0x53ec8] 當回合**。byte+5 bit0 的歷史高階命名已撤回，僅保留 raw mask。
- [x] **戰鬥規則來源盤點 + 動態驗證清單** → `27`:青衫公式=remake 實作依據+交叉驗證;列出 10 項需 DOSBox 實機驗證(核心 #1-4=戰鬥狀態機旗標/計數語意);新增「回合無上限」需求
- [x] **動態驗證清單更新** → `27`§3：byte+5 bit0／bit7 的 caller-specific raw tests 已列出；回合/換邊完成條件仍未由完整 state-machine 關閉。7-8 用青衫攻略；9-10 對 normalized projection 可簡化，但 raw persistence/handler 仍需 `0x50B` slot 與 `+8` provenance；3(`[0x53ec8]`)低優先。舊「bit0=1 是存活」使用者記憶已撤回。
- [x] **撤回 `[0x53ad5]`=opened-treasure／unit-pointer 斷言**：`0x10322` 初始化時複製 0x20 bytes 到 `[0x53ad5]` 指向的 buffer，`0x13d00` 以 event index 寫其 byte；ch25 post `0x24f30/0x24fb1` 讀 entry #12 來選 FDTXT index（base+5/base+8）。它是 battle-local state table，但高階 event 意義未命名；`OpenedTreasure` 保留 remake-owned state，不再聲稱原版位址。
- [x] **state table entry12 writer closure**：`0x356bc..0x35821` 先 gate table[12]，成功臂以 actor class 查 item `0xd0`、`0x1b8e7` 消耗它、完成 presentation 後才設 table[12]=1，接 `spawn(1)→JOIN(31)→FDTXT #4`。因此 ch25 post 的 table[12] base+5/+8 有直接來源；尚未完成兩臂 runtime 資產，不能以 treasure／party condition 取代。
- [x] **entry12 dispatch-scope audit（FDFIELD 勘誤）**：IDA 的八個 generic indirect xrefs 本身無法定位章節，但 map25 FDFIELD field slot2 已直接證實 selector1、座標 `(1,46)` → event61/`0x356B7`。舊「沒有 map25-local caller」斷言撤回；item D0、entry12、spawn1、JOIN31 與 text2/3/4 已資料化。59 幀呈現與 selector 時機的後續接線狀態以 `REMAKE-FIELD-EVENT61-SELECTOR1` 項為準。
- [x] **RE-FIELD-EVENT59/60**：map25 y36/y22 selector0 邊界及 trigger record byte6 非零 gate 已閉合；分別把 ranges39..44、23..24+53..56 低四位模式設0，規則已嵌入 editable map asset。
- [x] **REMAKE-FIELD-EVENT59/60-SELECTOR0**：合法 IDA 9.4 與 Docker Capstone 固定 `0x13488` 只有 path byte1 進 `0x1300D`，七拍提交 `x-1` 後以新座標呼叫 selector0；重製已在每個向左格步驟相同提交點執行 event59/60，向右反例與第1..6拍不觸發測試通過。
- [~] **REMAKE-FIELD-EVENT61-SELECTOR1**：event61 handler 已解碼並資料化；selector1 位於 AI `0x13E77` 收尾及玩家 `0x18890→0x18D8C` 三個 action handler 成功返回臂，不可簡化成任意 walk completion。FDTXT_026 63 句與 `ch26.json` 全量對齊；presentation 保存 archive/resource/frame/base/stride/transparent/delay。共同成功閘門現已接入待命、攻擊、一般法術、原始指令、三種已閉合物品交易與 AI；攻擊及攻擊型法術延至全螢幕演出結束，取消、目標不合法與 executor 錯誤不觸發。正式路徑完成 FDTXT2／3／4、59 幀兩 tick cadence、native compact remove D0、entry12、append group1 與 persistent JOIN31，缺 D0 不突變。ch25 pre-handler `PAN(9,39)→FOCUS_UNIT(0)`、ch26 slot0 `(15,46)` 及原版 cursor state machine 又推得 event 格 runtime view `(camera 0,39; cursor 1,46; visible 1,7)`，`battle_ch26` 已資料化初始 view/HUD，真實資產 regression 不再手工注入。剩餘門檻是同 roster/event/tick 的未修改 DOSBox 一般玩家比較；production 已達 E1，未達 E2。
- [x] **NATIVE-MAP-RENDERER-INPUTS-ALL-MAPS**：稽核證實舊資產只有 map0 帶 composition byte+3 與 FDSHAP terrain control，map1–32 全缺，這會讓後期所有 indexed map frame 在資料入口失敗。新增 `sync_native_map_renderer_inputs.py`，先驗證 FDFIELD.DAT／FDSHAP.DAT 大小、MD5、SHA-256，再只同步 `native_tile_blit_modes/native_terrain_control`、保留既有寶物／事件／手改欄位；33 圖 `--check` 與 Go loader regression 通過。這關閉 E0 input completeness，不代表 ch02+ view/HUD runtime 或畫面 E2。
- [x] **撤回 ch27 `0x24618`=acting 的暗示**：official IDA 定義 `sub_24618=0x24618..0x24754`（含 post-handler `0x33af1/0x33c9d` callers）；Docker Capstone 證實它是 13×8 offscreen terrain + 固定 9-pass strip composite + 0..62 palette 收束的地圖轉場。四個參數是 tile/strip geometry/progression，不能降成 actor `act`、pan 或任意 fade；strict indexed runtime adapter 已於 2026-07-28 接通。
- [x] **全 30 關卡目標表(攻略 ground truth)** → `28`:每關勝利/失敗/加入條件;**失敗條件=護衛目標**證實 unit_state 機制;加入=roster_has;ch30 魔神連鎖=回合事件;remake 關卡規格直接可用
- [x] **撤回章17 alive 誤讀**：依 caller-specific raw bit0 branch 重新解讀；舊「指定單位存活→設碼」已撤回 → `25`/`26` 回填
- [~] 單位 0x50B 結構：`+5` raw bit0/bit7 predicates、`+8`角色ID、`+0/+1/+2/+6/+0x31` 已解；完整逐欄佈局 [阻]（remake 用自有 struct，不需）
- [ ] (補)更新 doc 12:修正「main=0x10000」、補章節→BGM 表 0x51e63 精確曲號

## 第 6 輪（歷史單一戰鬥演出 fixture 對照；不代表通用戰鬥 renderer 1:1）
> 目標:remake 戰鬥攻擊演出(orig_05)像素級對齊原版。方法=**密集網格疊圖 oracle**(見記憶 pixel-align)+ 反組譯確認機制,**無 dosbox debugger**(0.74 vanilla 不能 dump)。
- [~] **正式戰鬥的雙 record 演出繪圖機制** → `35`：`0x28a6c`、blit `0x4e63d`（無縮放／無翻轉，尺寸朝向燒進素材）、固定錨點 `(164,157)`、`[0x540ff]` 的 0／非零 branch、BG 多層（`0x52381=BG.DAT`）、戰場→BG 表 `0x52363` 都有直接證據。`0x28a6c` 另有終局與事件 caller，故不能宣稱「完整」或把正式戰鬥的 phase／resolver 語意外推至所有 caller；完整解析與限制見 `35`。
- [x] **figure 幀/姿態**:我方亞雷斯=攻擊動作1 `FIGANI_013_f01`(組×3+1,人眼確認);幀序播放;守方不翻轉(FIGANI_288 原圖面右)
- [x] **白斬擊弧 = FIGANI 攻擊幀自帶**(燒 sprite),移除程式 vector 補弧
- [x] **[設計鐵則] 我方=背影+腳下台座 / 敵方=正面**(使用者確認,與攻守無關純陣營)→ `35`§3.2.5
- [x] **figure 位置對齊**(密集網格+程式量土台中心):我方土台中心 x≈238、敵方腳 y≈135(@320)
- [x] **狀態欄對齊**:名字放大(16px視覺)、血條加長(緊接標籤到數字)、bevel 立體框、HP/MP淺藍標籤、暗槽色暗版、上下欄位置/間隔(我方離頂、敵方離150線空隙)
- [x] **z-order RE**:演出順序狀態欄(0x28ce7/0x28d62)先、figure(0x28e76/9a/ee0)後 → figure 蓋住狀態欄;remake 改 BG→狀態欄→figure → `35`§4.-1
- [x] **狀態欄機制 RE**(agent):真繪製器 0x18c6d(非 0x29164);框=素材sprite、HP=逐欄cell(len=curHP×101/maxHP)、名=16×16點陣字、數值=6px digit cell → `35`§4
- [x] **清除錯誤斷言**:土台正名 FIGANI自帶→**TAI.DAT 台座**(0x29164 載 0x28c46);figure-X=word[unit+0x40] 誤讀;對話框開框碼 0x16F40
- [x] **① TAI.DAT 台座解碼 + remake 貼上** ✅(v23):TAI_004=154×42 綠草橢圓台座(decode_sprite 解 body[4:],index0透明17%);remake 載 tai_004.png 貼我方腳下(z:狀態欄<台座<figure);對齊 orig 取代偏灰 dither。確切 entry↔職業/地形對映待後續
- [x] **② 複查 `+0x40`／figure displacement**：**`+0x40=當前HP`**（`0x18c98` 血條 `word[+0x40]×101/word[+0x42]`=HP%）與 `+0x44/+0x46=current/max MP` 已閉合，舊「戰場格 X」已清。`0x29f72` 是 derived AP/DP/HIT/EV、HP、item、terrain/RNG 的 hit/crit/damage resolver，**不是** lunge；`+0x48/+0x4a` 亦是 derived AP/DP。`0x2935b` 直接以 `frameIndex*4+8` descriptor pointer 讀 header `u16 X/Y` 再交 `0x4e63d`，故逐幀 figure displacement 已閉合為 frame metadata；caller schedule／攻守配對仍另列待辦。
  - [x] runtime export guard：`cmd/fd2` regression 對 `meta.json` 全部 22 個 FIGANI resource、每一 frame 的 `(X,Y)` 與 player archive `FIGANI.DAT` header 比對；不再只憑 exporter 意圖宣稱 runtime 位置資料正確。
- [x] **③ 狀態欄框/HP 用真素材** ✅(v25-26):破解 FDOTHER#5 LMI1 sub-sprite codec(反組譯 0x4e916:c≤0xC0 literal/c>0xC0 run,新 codec,`tools/decode_lmi.py`);框=#22(149×42 含bevel+標籤+槽)貼 panel.png、血條 cell=#27-30;修 HP靠左(槽 x21-123)/提亮/數字對位。doc35 §4.2.5。盜賊 y 軸對齊(276→296,頭頂偏上一排)
- [x] **④ BG 草地延伸到 figure 腳下** ✅(2026-07-05 使用者確認 `docs/figures/battle_restore_grid.png` 網格對照:左原版/右 remake 兩邊草地都延伸到 figure 腳下、台座疊綠草非黑底,一致)

## 第 7 輪（歷史戰鬥演出資料化 round；「像素級」只限當時 fixture）
> 從「手調對齊」進化到「原版資料驅動」;README 對外展示;全部 push(commit 至 a42ee4a+)。
- [x] **[重大] FIGANI 幀標頭 +0/+2 = 每幀絕對螢幕座標 (dx,dy)@320**(修 doc06 錯誤標註「boundW/boundH」):
      f01=(141,3)/盜賊=(16,41) 與模板匹配 orig 落點完全一致 → 走位/伸擊/突刺全在資料,引擎照貼即可
- [x] **戰鬥演出資料驅動重寫**:meta.json(22 個 FIGANI 全幀 dx,dy)+ loadFigMeta;刪 lunge/錨點手調;
      FIGANI_013 15幀=f01-f10旋轉蓄力/f11黃劈擊/f12-14突刺;盜賊 4 幀待機呼吸
- [~] **打擊感（歷史 fixture，已勘誤）**：舊實作曾把命中寫成全紅剪影與
      VGA 色盤等價；目前只由 `orig_05_attack_03_impact.png` 支持守方剪影、
      攻方維持 FIGANI 原色，原始 DAC 輸出欄位與通用時序仍未接入。HP 命中窗
      也只算重製端 E1 近似，詳見本檔 2026-08-10 勘誤段。
- [~] **通用化（重製端管線）**：`newAtkAnim` 已可依組別與配對 FIGANI 建立
      動畫，但「所有角色的命中幀都是倒數第 4 幀」仍只是歷史推定；未取得各
      frame flag、傷害步進與 raw presentation 欄位前，不提升為原版通用規則。
- [x] **播放速度接口**:FD2_BATTLE_FPT 環境變數(tick/幀,預設3)+ atkAnim.fpt
- [x] **像素級對齊(模板匹配法)**:三 figure+台座+狀態欄框+三處數字 全部 err=0 且 dx=dy=0;
      狀態欄=原生 149×42 blit(敵(0,154)/我(171,4))、數字=LMI1 #31-40 素材(#42-51綠/#119-128黃=滿血變色)、
      LMI1 混雙 codec(框 0x4e916/小cell 4-mode)、VGA 6-bit palette ×4(decode_lmi 修正)
- [x] **README 對外展示（局部 fixture）**：`battle_restore.gif`、
      `battle_storyboard.png`、`battle_restore_grid.png` 仍保留作歷史證據；
      2026-08-10 首頁改用 `battle-impact-compare-20260810.png`（原版正規化／
      重製／逐 RGB 差異遮罩），不得把任何局部圖稱為整體「像素級 1:1 還原」。
- [x] **FD2_SHOT_SERIES 逐幀截圖鉤子**(GIF/分鏡素材管線)
- [x] **戰鬥狀態欄姓名改接原版字模（2026-08-10）**：撤回「名字=TTF 28px」舊決策。
      `0x18c6d→0x15f84→0x4ea2a` 已證實使用 FDOTHER#4 16×16 1bpp glyph；重製端
      以 `FDOTHER_004.bin`＋`unicode_to_glyph.json` 逐字 render，未知字元才保留
      TTF fallback，並以 Docker/Xvfb 產生 [`battle-native-name-remake.png`](../figures/battle-native-name-remake.png)。
      這是戰鬥狀態欄 E1 消費端修正，不代表全螢幕演出或一般玩家 DOSBox E2。
      **2026-08-25勘誤**：舊載入器因頂層`_comment`字串整批失敗，上述2026-08-10
      圖實際仍是TTF fallback；現已修正解析、`+(5,4)`原點與實檔「盜賊」回歸，
      並重抓同名圖。E1狀態以本勘誤後產物為準。

## 第 8 輪（歷史玩法盤點 round；魔法／SFX／campaign 目前仍是 partial）
> 使用者指示:檢視腳本系統一路到移動/觸發戰鬥/魔法,盤點缺口逐項補。
- [x] 盤點完成(見下缺口清單)
- [x] **腳本系統 campaign(M4 骨架)** ✅(74bf386):internal/campaign(節點圖 Runner:story/battle/
      choice/event/ending + 旗標 + 敗北路線 + choice 條件選項;單測3條);引擎接線(FD2_CAMPAIGN=1、
      enterNode/campInput/drawCampaignUI、勝敗 Enter 轉場、resetBattle 重試);campaign.json 第一章示範
      (敗北→撤退設旗標→再戰)。待續:商店節點、存檔、原版 33 關自動生成 campaign
- [x] **移動動畫** ✅(74bf386):battle.Path(BFS 路徑)+ walkAnim 沿路徑逐格走(方向幀+OffX/Y 內插,
      ~4-5 tick/格,走完進攻擊/待命,期間鎖輸入);AI 移動沿用瞬移(待接同管線)
- [x] internal/battle 測試失敗已修 ✅(e09c68c):部署格斷言=舊設計殘留,對齊現行(部署格屬 spawn_party)
- [~] **魔法系統** (第7-8輪完成資料與部分 runtime,commit 3c618c4/74366fa:暫定四向 action UI+法術+MP+青衫公式;code: ringInput/castSp/spells.json)——`0x18d8c` 已證實方向 result order，但 `0x1cff0` command table、完整 native 演出仍待；
      不存在獨立 spell-id→FIGANI 特效索引（doc37）；僅已證實施法者自身 FIGANI 組動畫，其他 spell runtime 保持 partial
- [x] **音樂** ✅(e09c68c；2026-08-29 catalog收線):audio.go(ebiten/audio+vorbis；`play_bgm`
      正確位址`0x25977`：同曲不重播／換曲釋放／loop count 0整檔循環)；campaign節點BGM驅動；
      FM／MT-32 30份OGG由嚴格catalog驗證，舊fallback已移除。待：非campaign模式場景→曲號自動對映(doc12表)
- [x] **音效 SFX** ✅(第8-11輪完成,cmd/fd2/audio.go;commit e09c68c 音樂+SFX 收線)。資料位置 RE(doc36):`FDOTHER.DAT` 資源 #31(巢狀 `LLLLLL` 容器，14 個目錄項目／13 個非空 8-bit
      unsigned mono raw PCM 子樣本)+ 戰鬥音效動態 index(同檔案,依攻擊資料決定 index);播放走
      `AIL_init/set_sample_address/set_sample_loop_count/start_sample`(0x26896/0x26945)。
      待:14 子樣本→UI事件對照、戰鬥動態 index 表還原、remake 端接入(SDL_mixer/ebiten audio)
- [~] **native action overlay／現行四向 approximation**：官方 IDA 9.4／Capstone 已更正 `0x1741c` 的 raw cell index 為 `3*firstArgumentWord+2*secondArgumentWord`。battle wrapper `0x18d8c` 與空游標 `0x16f55` 必須分開；chapter0 `CONTINUE→Return` 使用 cells=`[21,15,18,12]`。direction2設定、direction3／END、direction0巢狀資訊與離場均已達正式E1；direction1全軍移動亦達E1，正式資料僅有的selector1 event61／75已接入途中暫停與續行，未知handler仍在確認前整批失敗即關閉。剩餘空格缺口是巢狀current-runtime存讀檔及同狀態E2，不再把「所有全軍移動事件效果」籠統列為未做。完整證據見[`fd2_system_overlay_options_ida.txt`](../data/ida/fd2_system_overlay_options_ida.txt)、[`fd2_system_exit_and_group_march_ida.txt`](../data/ida/fd2_system_exit_and_group_march_ida.txt)及[`fd2_continue_action_overlay_ida.txt`](../data/ida/fd2_continue_action_overlay_ida.txt)。open／close各四次present、72×72 snapshot primitive與原始FDOTHER#2圖格均有回歸；目前Ebiten每幀整幅重畫，尚未正式消費snapshot restore，設定面板也尚缺同狀態逐幀／音訊E2。
- [x] **RE-ATTACK-EQUIPPED-PREDICATE-1B83D**：官方 IDA 9.4 閉合 `0x1b83d(unit,a2)`：八格依序檢查 `flag&0x40`，`a2==0` 僅接受 item `<0x80`，`a2!=0` 僅接受 item `>=0x80`，回第一個 raw slot／`-1`。新增 `battle.NativeEquippedInventorySlot` regression，action overlay 在 constructor raw flags 存在時採用此 predicate；target geometry、damage/effect、renderer 仍 partial。
- [x] **RE-ITEM-AVAILABILITY-GATE-1B8A6**：`0x1b8a6` 的 raw occupied count 已有 record adapter；新增 Unit-facing `NativeInventoryAvailableCount`，overlay 在八格 constructor flags 存在時以 bit7-clear count 決定 item disabled，`len(Inventory)` 僅是 legacy fallback。

- [x] **native command target flag/runtime-grid bridge**：`0x14818→0x4e040` 的 raw target resolver 已有純資料層與 runtime producer（camp predicates、cross、cardinal flood-fill）。修正舊斷言：bit40 block；bit80 是扣 terrain cost 後強制 remaining budget=0 的可達終點，不是 zero-cost chain。FDFIELD composition event word low byte（entry+2）只是不可變來源；每次 command caller 先取 low5 基底、使用 byte+3 grid，結束即依`0x4dbfc`重建。缺 exact event bytes/grid 仍 fail-closed。
- [x] **native command MP transaction**：官方 IDA `0x21227→0x1CA89` 已證實 generic command 在 candidate array 建立後、逐 target effect 前以 record `byte+5` 從 actor runtime `+0x44` 扣 MP，前段 selector 已 gate `currentMP >= cost`。`SpendNativeCommandMP` 以 raw 0..255 cost 實作該成功交易並在 invalid/MP不足時不變更；刻意不接受 normalized `Spell`、不搶接 legacy cast/UI。
- [x] **generic native command two-stage data contract**：`NativeCommandEffectTargets` 固定 `0x1cff0` generic path：actor/`record+3` candidate list → confirmed candidate → confirmed cell/`record+4` final-effect list；non-candidate confirmation 拒絕。它不涵蓋 `0x17/0x1e` special branches、MP/effect/renderer，且尚未接 UI。
- [x] **native command record loader**：`NativeCommandRecord` 明確表示 verified IDs 0..35 的 raw `+3/+4/+5/+6` 為 selection/effect mode、MP cost、target code；從現有 physical `spells.json` 讀取時逐 row 重解 `raw` 七 bytes，欄位不符、缺洞或非 36 rows 均拒絕，避免 normalized Spell 名稱／效果編輯污染 native ABI。
- [x] **SDD native command family matrix**：`56` 現以 IDs 0..35 的 dataflow、strict engine slice、UI/renderer gate 三欄固定已證實 family 與 fail-closed 邊界；不得以 label、raw record 或 generic dispatch 把未知 ID 借接 legacy `CastArea`。ID24 已更正為玩家 `2A6BD→276EC→1C81F` 的 derived-stat special route；AI table 的 `funcs_1541f[24]=22153` 是另一分派，不能誤接為 ID16 heal。
- [x] **commands 0..8 direct compositor + numeric route**：`0x1cff0` 對 IDs0..8 直入 `0x2a6bd`，確實不是 handler table 的 `0x21227/0x213b7` wrappers；但 direct `0x2a6bd` 的 `sub_2b659` MP event 和 final-target loop `1C75E(targetSlot, commandID)` 已重新逐行確認。故 ID0 executor／target UI 恢復為 bounded state slice，renderer/post-resolution 仍未宣稱完成。
  - [x] generic renderer schedule：`funcs_2ac25[0]=0x26152`；`0x2a6bd` 以 handler mode0 取 step count，再逐 step 走 mode2→`0x11eb0` 320×200 present→`0x17aa9(1)` tick→mode1，收尾另走 mode4/double-buffer path。`0x2b9a1` 依 descriptor frame byte+6 delay 推進 `0x540fc/0x540fd` subframe counters。這是 schedule ABI，不為 handler 視覺命名。
  - [x] generic BG selector boundary：`0x2a6bd→0x2b5e1(finalCount, finalTargetArray)` 倒序 target scan，經 `0x12e38`／`0x1f183`，只有 gate 不通或累積 selector=0 才以 control byte+2 取代 selector，再載 `BG.DAT`。`NativeCommandBackgroundSelector` unit regression 固定這條 raw ABI；command ID 只先選 generic/special presentation branch，不直接選 BG resource。selector semantic 保持 raw。
  - [x] BG archive input：`BG.DAT` #0/#1/#2 為 320×100 的 `0x4e63d` four-mode single-frame payload，新增 `fdother.DecodeArchiveSingleFrame` 與 player-archive decode regression；它只提供 indexed frame，layer selection／schedule 仍由 native caller evidence 決定。
- [x] **shared native damage route IDs0..12**：IDs0..8 經 `2A6BD→2B659/1C75E`，ID9 direct `1CA89→1C75E`；IDs10..12 的 `0x21548` 專用 compositor 尾端也直接 `1CA89→per-target 1C75E`。同樣扣 MP、逐 target numeric writer、success-only raw completion writer；engine bounded support 0..12，UI 仍僅 ID0。不得從 numeric 共用推論 visual equivalence。
- [~] **native command IDs13..16 healing route**：IDA `0x21AD9/0x21B99/0x2211C/0x22153→0x21B18` 已閉合 generic final target array、`0x1CA89(actor,id)` MP debit、`0x1C8ED→0x1C916` per-target HP restore 及 `+0x42` cap；其 amount 公式是 `floor(amount*9/10)+floor((rand%100)*amount/1000)`。`ExecuteNativeCommandHeal` 已接 strict engine slice。玩家與敵方 mode 11 現都走 `0x21EB1→0x22046` 的FDOTHER #3 16張前段、`0x1C4CC` #6七張、`0x1C2DA` mask、transaction後steady redraw及`0x1DF58` #5數字22張；缺raw資產不 mutation。只剩同狀態E2，且禁止誤用 IDs0..9 damage executor。
- [~] **native command ID24 player special route**：玩家 confirm 的 `0x1CFF0` 對 `0x18` 直入 `0x2A6BD→0x276EC`；`0x276EC→0x2B659` 以 `0x1CA89(actor,0x18)` 扣 record24 MP，並以 `trunc(actor derived +0x48 × 15/10) - target derived +0x4a` 呼叫 `0x1C81F`。原版為多段演出暫時復原 HP 後等份遞減，`ExecuteNativeCommand24` 已接相同 final delta 的 strict non-UI slice。`funcs_1541F[24]` 雖為 `0x22153`，但只在 AI／自動 `0x15311` dispatcher 使用，且傳 ID16 給 heal tail，不能拿來推導玩家 ID24。multi-hit／presentation/SFX/UI 未接。
  **2026-08-22 勘誤：** 新的IDA caller／resource98逐幀證據否定「等份遞減」；
  command24的分母為1，frame10一次發布完整傷害。selector32正式演出已接
  `RUNTIME-E1 partial`；後續已接`0x29C90`兩段20次BG滑動，但完整indexed base與一般玩家E2仍缺。現況以檔首有效佇列、
  `56`與`58`為準，舊段保留只為解釋錯誤形成原因。
- [~] **native commands28/29/31 derived-strike siblings**：同一 `0x276EC` 對玩家 ID28／29／31 分別選 20／12／18 倍率，並經 `0x2B659→0x1CA89(actor,id)` 與同一 final HP delta path；其 ordinary record geometry 可走 `NativeCommandEffectTargets`，`ExecuteNativeCommandDerivedStrike` 已接 strict state-only slice。ID29玩家正式confirm另已接selector34／resource104多目標indexed owner與全批rollback；敵方caller及28／31仍不可借接。ID30 的 special route 亦已收斂：`0x1CFF0` 先確認 record+3 candidate，`0x149F8` 從 saved pre-confirm cursor 朝 confirmed cursor 走 `record+3-0x10`（record30=4）格；X-first，僅 X 相同走 Y，selector=1 只收 enemy，然後 `0x2A6BD→0x276EC` default倍率18。`ExecuteNativeCommand30` 已接顯式 cursor state slice；不將其隱藏接入 current UI，cursor lifecycle／multi-hit／SFX／indexed renderer 仍待。32..35 走 `0x27FC9`。
  **2026-08-23 command28勘誤：** 上一項「同一final HP delta」對command28不成立。直接writer以分母8發布，固定FIGANI全檔的可達effect每個target都只有一個impact marker，且函式尾端沒有補差；28的正式delta因此是roll/8，重製端已修正。29／31仍為分母1。caller-specific base／`sub_29164` mode／`sub_29C90`分歧見[`fd2_command28_29_31_presentation_ida.txt`](../data/ida/fd2_command28_29_31_presentation_ida.txt)。
- [x] **RE-COMPOUND-PLAN-27FC9**：Docker Capstone 重新固定唯一 caller `0x2A7CE`（`0x2A6BD` selector `>=0x20`），並把 ID32/33/34/35 的 raw helper 順序、ID33 direct clear offsets `+0x25/+0x26/+0x27`、固定 amount `0x320` 資料化為 `battle.NativeCompoundCommandPlan`；此為 editable evidence-only plan，`Callee==0` 明確代表 inline byte clear，不執行 transaction/MP/target/UI。
- [~] **native commands 17..19 transient modifiers**：ID17/18/19 handlers 已直接定位 `+0x22/+0x23/+0x24` nonzero gate 與 writer：17 對 derived `+0x48`、18 對 `+0x4a` 做 `__CHP(value*0.15+1)` **toward-zero** increase 並設 2..5 duration，19 對 `+0x4c/+0x4e` 各加 15 並設 duration。`0x377A4` 暫存 control word、設 RC=11b、`frndint` 後 restore，故撤回 FPU-rounded／未知 round-mode 說法。ID17/18 的 wrappers 都以 `0x1CA89(actor,0x12)` debit，且 records17/18 的 raw 7 bytes 相同；因此禁止泛化成「每 handler 必傳自身 ID」。這撤回 doc35 將 `+0x48..+0x4e` 稱作 screen coordinates 的衝突斷言。此段的「engine integration未閉合」是當時狀態；2026-08-22後玩家與AI交易、玩家`0x1D6C8` palette／SFX演出均已接入，仍缺phase-expiry caller、status labels與E2。
- [x] **RE-COMMAND17-19-RAW-DISPATCH**：新增 `battle.ApplyNativeCommandModifier`，嚴格映射 ID17→`0x22721`、ID18→`0x22866`、ID19→`0x22997`，回傳 branch-specific raw result/RNG/accumulator；不接 MP、target、presentation 或 stat 名稱，unsupported ID fail-closed。
- [~] **native commands 20..21 flag-clear/restore route**：`0x22A85/0x22BC6→0x22AA8→0x22AF6` 分別對 `+0x25/+0x26` 做 nonzero gate；成功時以 command record 10 呼 `0x1C916` HP writer 後清 flag，零 flag 只顯示失敗。MP debit 仍以 command20/21 record。`ExecuteNativeCommandClearRestore` 已接 strict non-UI core（record10 amount、raw clear、cap-aware restore、empty gate仍 successful completion）。兩個 status 名稱與 UI 未閉合；ID22 的 `+0x27` application route 不可混入。
- [~] **native command 22 application route**：`0x22BE1→0x22CDA→0x22D1B` 在 `+0x27==0`、class `+0x20∉{0x19,0x1a}` 且 `rand()%100<50` 時，固定以 `0x1C81F(target,10)` 扣 10 HP，寫 `rand()%4+2` 至 `+0x27`；其他路徑僅失敗顯示。它已接入 `ExecuteNativeCommandApplication` 的 strict raw core；status name/tick、UI、expiry recompute integration 未閉合。
- [~] **transient command duration lifecycle**：official IDA/Capstone 固定 `0x1A866` gate 為 `record+6 == selector` 且 `(record+5 & 1)==0`，direct callers 為 `0x1a4d1→push1`、`0x1a55e→push0`、`0x1a797→push2`；另有 `0x1a30b` 內部 `record+6==2` sweep，不能混成同一 phase。通過 raw gate 後，六個 bytes `+0x22..+0x27` 才逐一 decrement，歸零才發 expiry feedback 並 `0x1B750` 重算 derived stats。remake 現以 `NativeRecordByte5/6` 保存 provenance，`TickNativeTransientsRaw` 失敗即關閉；selector 1→0 已按原順序接在合併的非玩家掃描前，selector 2 接在玩家輸入前，歸零時才重算並同步 raw，整批任一失敗不發布。舊 `TickNativeTransients(camp)` 不再猜測映射。FDTXT 481..486／DATO indexed 到期訊息已達 `RUNTIME-E1`；`sub_17FC0` status colors／entries `0x37..0x39` 已由正式面板消費；尚缺高階名稱、精確 tick／音訊與一般玩家 E2，故不可稱完整介面復刻。
- [~] **native command 23 special relocation**：`0x2218A→0x22253` 已確認
  先把 selected unit `+0/+1` 寫 `0xff/0xff` 作離場演出，再直接寫 selector
  cursor globals `0x51CF9/0x51CFD` 進場；這是 direct coordinate
  relocation，非 path movement。mode6 legality 已釘為 other-active-unit
  occupancy gate 與 target-dependent terrain code20。2026-08-22正式玩家路徑已接
  `0x1D6C8`八段palette、第一次`0x22253(...,0xff,0xff,current)`離場、第二次
  `0x22253(...,destination,destination)`入場，再原子發布MP／座標／action；兩段
  renderer及交易皆先完整preflight，缺件不發布第一幀。狀態提升為`RUNTIME-E1`；
  尚缺原版同狀態camera、逐幀與逐音訊`PLAYER-E2`。
- [x] **RE-RAW-BYTE6-FDFIELD-CONSTRUCTOR**：`parse_field.py` 保存 FDFIELD roster b0 的 `native_record_byte6`，`export_units.py` 將其寫入 units JSON；`battle.Load`／`Scenario.PartyUnits` materialize raw `+6`（own party=2，map selector key 亦保留 direct provenance）。這只閉合 constructor source，不替 `+6` 命名 camp 或 phase 語意。
- [~] **native commands 25..27 closure**：ID25 `0x22C04` 以 record25
  MP debit，僅在 target `+5 bit0x80` 已設時清 raw bit；
  `ExecuteNativeCommand25` 已接 strict non-UI slice。ID26/27 復用
  `0x22d1b` 到 `+0x25/+0x26`；舊「固定10 HP／兩 RNG」已修正為
  gate RNG→damage RNG（base10 實際9 HP）→duration RNG 三 draws。
  `ExecuteNativeCommandApplication` 已同步修正。UI/status labels 與其餘
  engine integration 待。
  **2026-08-25 後續勘誤：** 玩家25–27與敵方26／27已接正式indexed handler tail，
  mask Draw後才原子發布交易，必要的22張數字段與500 ms尾停後才完成行動；25全成功
  時不建立空queue。敵方25仍因缺正常AI producer失敗即關閉。高階status名稱與E2仍待。
- [~] **native command IDs10..12 compositor family**：`RE-CLOSED`／`DATA-READY`／`RUNTIME-E1`。IDA Pro 9.4 已固定 ID10 `0x21527`、ID11 `0x2185F`、ID12 `0x21A9E` wrappers；11／12先播放#80 selector2並各呼`0x2189A(actor,15,10)`／`(actor,30,16)`，再進共用`0x21548`。typed schedule、`0x1F558`等價fixed-point compositor、四surface×60張、selector13八個marker、#5結果佇列及正式玩家／敵方owner均已接；敵方`0x15311→funcs_1541F`也只在Draw邊界發布MP／HP／RNG／acted，晚期失敗完整rollback。剩同狀態逐幀／逐音訊E2；`0x2189A/219AD`不重解，不可從數值共用推論visual equivalence。
- [x] **scenario native command-mask bridge**：`PartyMember.initial_command_mask` 已接 exact four-byte source，loader 對 malformed length fail-closed；`gen_campaign.py` 從 EXE `character_defaults.json` 依角色 index 合併至 ch01..ch30 而不覆寫既有手工 scenario 欄位。戰後 persistent snapshot 也保留完整五-byte runtime mask，level-up OR 不會跨 town/preparation 消失。ch01 悠妮 `[1,0,0,0]` 有 per-scenario materialization regression；不可由 normalized `Spells` 反造 raw bytes。待：逐章真機 availability 對照、未知 command effect／frame renderer。
- [~] **魔法系統**（資料表與基礎 Cast 已接，native command/effect 尚未閉合）: `magic.go`
      讀取 `spells.json` 的 36 條 EXE dump 與 normalized 名稱；`CastArea` 的範圍、命中、
      治療／輔助／狀態效果均有 deterministic regression，玩家法術選單可走射程高亮、
      結算、扣 MP 與既有戰鬥演出。敵方／友軍 NPC 在 raw route 未處理且無錯誤時也能經
      editable spell fallback 走 `NextAIPlan→aiStep→CastArea`；這些皆是重製端近似。
      ✅ 法術特效對映已 RE 定論(f8fffba 後,doc37):**不存在獨立法術id→FIGANI對映**——施法演出=施法者
      自己的組×3/×3+1(火花燒在 sprite 幀,`0x28784` 不讀 spell_id)。這僅閉合 FIGANI 手勢選擇；
      `0x2a6bd` command-specific presentation、SFX、命中與多段畫面仍待，現行角色攻擊動畫只是局部 adapter，
      不得稱完整原版一致。
- [~] **商店+祕密商店**: 69個shop節點已用`native_hub_variant` 1/3/5啟用indexed production owner；四項service與23筆secret chord gate已接。`found_secret_ch*`／legacy `SecretIf`只保留editable擴充，不再當原版gate。sell高階adapter仍會canonicalize ignored stale tail，不宣稱FD2.SAV byte parity。後續E2已閉合ch02三種主選單、secret chord/return與weapon purchase list；town variant1/2正常五項的修改 LOAD E2另已固定，剩餘為purchase後續、sell/equip/transfer、variant2 selection5／未修改玩家路徑及其他章節。
  **2026-08-22 後續勘誤**：上句「剩餘 sell/equip/transfer」是當時狀態，已被檔首有效佇列取代。sell success／credit／return 現有 route-patched E2；service2 名冊／面板與service3五個穩定子面板為 partial E2；真正未閉合的是service2 動畫相位與mutation／restore、service3 mutation／empty／full、recipient其餘分支的原版 E2 及未修改一般玩家路徑。
- [x] 存檔/讀檔 ✅(e09c68c):save.go 自有 JSON(節點/旗標/金幣/道具),F5/F9,節點邊界語意

## 第 9 輪 ✅(3-subagent 成本分工;haiku=資料/sonnet=RE·套件/旗艦=架構·驗收)
> 策略(rulebook/45):簡單工作派便宜模型,旗艦只做架構與把關;每件交付先抽驗再 commit。
- [x] **商店品項表**(haiku): `docs/data/shops.json`保存攻略來源的69家／23祕密商店品項與進入提示，campaign已資料化；這是外部攻略／editable authored資料，不是EXE gate真值。原版modifier/key→selection5仍列E0缺口。
- [x] **SFX 破案**(sonnet):FDOTHER#31=14個目錄項目／13個非空8-bit PCM＋AIL 鏈 → doc36；歷史WAV導出(export_sfx.py,11025Hz 負向證據);
      **index0=游標音確認**(5處方向鍵分支);戰鬥音效=另一獨立池([0x5411f])待導出
- [~] **法術 FIGANI 手勢邊界**：`0x28784` 不讀 spell id，沒有另一段 FIGANI 由 spell id 選擇（火花在角色幀）→ doc37；
      但 `0x2a6bd` command-specific presentation／SFX／命中分支未閉合，remake 角色動畫不可稱完整原版施法演出，
      不結案。
- [~] **歷史 legacy magic snapshot（不可作 native completion claim）**：舊條目所稱「魔法完整版」僅指
      `CastArea` AoE/命中擲骰與 normalized 輔助法術；2026-07-26 已由 SDD56/UI-03 取代為逐 command
      E0 matrix。native target/effect/transaction/presentation 未閉合者維持 fail-closed。
      (魔刃/魔鎧/風行 doc02 明文值)/毒麻封咒行動術/combo;13 單測;引擎:Buff 進 Attack、TickStatus、
      AoE 指空地、FD2_SEED。缺口列冊:風妖精 dmg=0 矛盾、劍技倍率表、傳送 UI
- [x] **全 33 戰場匯出**(haiku):remake/assets/maps/map1-32(96 檔,抽驗 3 圖合法);
      旗艦接線 loadMap(dir)+campaign battle.map 欄位(map3 實測換圖)
- [x] **AI 行走+敵攻我演出（歷史 normalized approximation；非 native parity）**：
      `NextAIPlan` 決策執行分離+`aiStep`；`atkOwn` 欄位按陣營。這只代表重製端可玩
      近似路徑，不代表原版敵方 AI 已完成
- [x] **SFX 引擎接入**(旗艦):loadSFX/playSFX+游標/確認/命中掛點(命中暫代,待戰鬥池)
- [ ] 戰鬥音效池([0x5411f] 動態子容器)導出+逐招對照
- [ ] 非 map0 角色 sprite 組匯出(換圖後 fallback 色塊)
- [ ] 33 關 campaign 自動生成(parse_field+劇情+商店串鏈,M4 工具)
- [ ] UI 音效 index 2-0xb 語意畫面實測

## 第 10 輪（歷史快照；不代表目前 parity）
- [x] **全 30 章 campaign 生成器**(sonnet):gen_campaign.py→campaign_full.json(183 節點,
      雙重驗證 python+真 campaign.Load;章→map 順序對應依據誠實);旗艦修 resetBattle fallback
      (scenario 空不再錯載 ch01 → roster 全員登場);ch02/map1 實測 33 單位 ✓
- [x] **sprite/頭像滿覆蓋**(haiku):96 組×12 幀 sprite(全 33 圖需求);旗艦補 5 敵方頭像→384 全滿;
      map3 實測全真 sprite
- [x] **戰鬥音效池 RE**(sonnet):FDOTHER #48-53/64/78/88 九候選 42 WAV(七池 sub0 相同=共用
      揮擊音,md5 抽驗 ✓);[0x5411f] 載入點 0x028110(index=招式id→byte陣列動態);
      **位址勘誤:doc36 全篇 0x11fba→0x111ba**(對齊 doc35)
- [x] **全域文字銳利化**(旗艦):font.go per-尺寸 face 快取,所有 Draw 呼叫自動銳利(糊字根因=非整數縮放)
- [x] **BGM 舊聽辨勘誤**：曾以單獨實聽把`FDMUS_018`判成商店；後續原始boot caller
      `0x25db5`已證實它是boot／標題消費曲，商店／城鎮固定曲為`FDMUS_010`。戰鬥曲仍待逐曲實聽。
- [x] **派工 SOP 入 rule**:rulebook/45 新節(haiku=資料/sonnet=RE·套件/旗艦=架構·把關;prompt 要素;把關不可省)
- [ ] **每章 scenario stub**(ch2-30「能玩」關鍵):party 延續+deploy_cells+initial_groups 全開
      (gen_campaign 擴充,回合增援事件之後疊)← 下輪首位
- [ ] 戰鬥曲號聽辨(使用者)+ 各 track 逐曲實聽修正 doc12
- [ ] 戰鬥 SFX:index 陣列填值上游、#48-64 逐招對照、remake 接入(atkAnim 命中掛 battle 池)
- [ ] UI 音效 index 2-0xb 語意畫面實測

## 第 11 輪（歷史快照；「全 30 章一條龍」斷言已由後續誠實揭露降級）
- [x] **ch2-30 scenario stub**(sonnet):29 個 chNN.json(party 4 人/deploy=own_deploy 真資料
      (9 章資源瑕疵 spiral fallback)/groups 全開排除 group==255 padding);campaign_full 30/30
      掛 scenario(含修 ch01 campaign 模式沒主角隊的壞點);三層驗證+3 章實跑
      → 產生全 30 章 authored campaign 節點；當時把它寫成「一條龍可玩」的句子已撤回，
      因為節點圖與實際劇情、戰後城鎮／整備、持續隊伍、存檔及介面驗收不是同一件事。
- [x] **戰鬥命中音接真素材**(旗艦):battle 池共用揮擊音(#48 sub0)接命中幀;loadWav/playRaw
- [x] **SFX index2 追蹤**(sonnet,部分解出誠實標記):真路徑=0x01cff0
      `[esp+計數+0xd0]`（填值待追）;
      **意外收穫:0x1c269 從 unit+0x1a 起掃描 5 bytes/40 bits 並輸出 byte index；欄位語意尚未定案**；`+0x22..+0x24` 是另一路 raw transient/modifier bytes;
      battle_sfx_map.json 骨架。依「夠用就停」:+0xd0 續追降低優先(共用音已可用)
- [x] 聽辨清單(extracted/music_ogg/聽辨清單.md,待使用者逐曲填)
- [ ] 戰鬥曲/勝利曲聽辨(使用者)
- [ ] party 數值成長/招募(doc28 加入條件)、回合增援事件疊到 stub
- [ ] ch10 等圖少數 tile 雜色查因
- [x] unit+0x1a vs +0x22 offset：constructor trace 已定案為 initial command mask vs raw transient/modifier bytes（舊稱 `magic_raw` 已撤回）
- [ ] +0xd0 陣列填值(逐招音效對照,低優先)

## 第 12 輪 ✅(招募成長/劇情文本/編輯器規劃/政策更新)
- [x] **gen_campaign v3**(sonnet):26 角色 21 章招募累積(ch30 全 30 人)+ 成長(HP 真表值,
      AP/DP 近似明標);**增援誠實跳過**(battle_events.json 實為勝負 metadata；此處的
      「event_id→group 未反組譯 0x22e5c」是本輪歷史快照，已由第 13 輪的 `0x1a813`／`0x51b91`
      證據取代，不得再當現況)→
      docs/data/turn_events.json 真資料 dump + doc26 防誤用註記
- [x] **story 管線**(sonnet):story_to_script.py,ch01-03 精校文本 156 句(speaker 對映 78-85%);
      引擎 story script 載入(旗艦:Node.Script+loadStoryScript,無檔 fallback)
- [x] **著作權政策更新(使用者 2026-07-03)**:FD2 版權過期,**對白文本開放入庫**
      (assets/story/ 例外;ch01.json 恢復原文);圖像/音樂/binary 仍本機
- [x] **tile 雜色結案**(sonnet):非 bug——map9 黑塊紫紋=原版地底裂谷美術;
      全 33 圖 index 零越界、匯出 vs oracle 逐像素 0 差異
- [x] **編輯器規劃**(sonnet)→ `38`:選型=獨立網頁單檔編輯器(File System Access API;
      不做 Ebiten 內建=避免編輯器複雜度混入引擎/不外包 Tiled=劇情事件無對應工具);
      MVP=戰場編輯(產物零轉換直接引擎載入);地基發現:MoveCost 未接地形、
      event.go 實作僅 doc29 願景子集(表單以實作為準+--dump-registry 同步)
- [ ] **戰場編輯器 MVP**:網頁單檔 HTML/JS,tile 繪製+單位擺放+部署格;FSA API 讀寫 assets/maps;
      驗收=引擎零轉換載入(細節 `38`)
- [ ] 劇情編輯器:對白+事件表單+商店(下拉=event.go 現行能力,`38` §3.3)
- [ ] 編輯器能力清單同步:Go --dump-registry
- [ ] campaign 節點圖編輯器(拖線/旗標/敗北路線可視化)
- [x] **地形屬性接線**:地形控制表 per-tile 確認(300~400 格不等,非固定 300;
      `tools/dump_terrain_table.py` → `docs/data/exe_tables/terrain.json`,33 tileset 全 dump)。
      移動代碼(byte1,0-5)語意用 references/text/notes.md 玩家攻略「地形移動力/攻防影響」表
      交叉驗證 AP/DP 數值全吻合(森林 code2/3 = -5%/+10%、沼澤 code4 = -5%/-5%)。
      `tools/export_engine_assets.py` 換算 per-tile 步行成本寫入 map.json `"cost"[]`;
      `battle.State.Cost` + `MoveCost` 查表(`remake/internal/battle/move.go`),`Load()` 自動讀
      units.json 同目錄 map.json 接上(main.go 未改動)。全 33 圖 + 頂層 map0 重新匯出。
      新增 6 個測試(`move_test.go`)。**限制**:僅步行成本,騎兵/飛行差異(notes.md 另有數字)
      待 Unit 加兵種欄位才能接;地形 AP/DP 戰鬥加成本輪未接。
- [x] **0x22e5c 語意更正**：已確認它是固定載入 `FDOTHER.DAT` #79
      並呈現的路徑，不讀章節索引，也不是 `turn_events.event_id→group` 消費點；
      真正增援消費鏈為 `0x1a813`（turn/camp filter）→`0x51b91`（全域 90-entry 表中的 FDFIELD 子集合 0..57）→spawn 原語。
      舊「待反組譯 0x22e5c→接增援」與「章1專屬中場」名稱已撤回，詳見
      `25` §6.1；玩家可見場景名稱仍須原版執行期證據。
- [ ] ch04-33 劇情文本精校(30 章,PNG 人眼轉錄;對白已可入庫)
- [ ] 視窗縮放 filter 查證(可能 linear 暈染,tile-debug 提醒)

## 第 13 輪 ✅(增援打通/地形/開場實機裁決/文本流水線)
- [x] **回合增援機制全解**(sonnet):0x51b91 全域 90-entry 跳表中的 FDFIELD 子集合 0..57(0x22e5c 排除);map0 4/4 ground truth;
      extract_event_id_groups.py;turn_events.json 補 groups
- [x] **gen v4 增援疊入**(sonnet):18 章 35 筆 spawn_group(turn 精確比對=原版語意);
      \$turn_counter 展開(3 圖核對);6 筆 \$reg_or_mem 列冊待解;ch08 T0/T4 實跑增援登場 ✓
- [x] **地形接線**(sonnet):FDSHAP 2N/2N+1 配對地形表(4B:寶箱/移動代碼/**戰鬥背景編號**
      =doc35 地形→BG 對應解!);MoveCost 查表+6 測試;main.go 零改動。騎兵/飛行差異待兵種欄位
- [x] **ch04-08 文本**(sonnet):177 句入庫;speaker 編碼文獻化(0-9,A-V→face_portrait)
- [x] **dosbox 開場實機裁決**(sonnet):logo=縮放進場(使用者記憶證實,推翻 doc23 [驗]);
      開場實為 32.3 秒多幕過場(疑 ANI.DAT 驅動,新缺口);選單座標/硬切閃光轉場
- [x] **title 修正**(旗艦):logozoom phase(紅閃→縮入→白閃)+選單實拍座標
- [x] **ANI.DAT 完整 AFM 格式 RE**(sonnet):9 資源=10-opcode 增量繪圖 VM(palette 4 op+
      framebuffer 6 op,直寫 VGA 0xA0000);173B 標頭+8B 幀記錄,289 幀全解無例外(位元組自洽);
      `tools/decode_ani.py`;9 資源逐一視覺比對 doc23 §2.4③ 分鏡全數命中(守護者/索爾/拔劍/
      騎馬夜行/明月/合照/金鎖);**「2」logo 縮放亦由 ANI.DAT(資源#1)驅動**,更正 doc23 猜測。
      見 doc39。待補:⑥浮空城/⑨惡魔臉未逐幀窮舉、轉場閃光呼叫端排程。
- [ ] 開場配樂曲號實聽驗證(容器 nosound 無法驗;使用者聽辨)
- [ ] ch21/22 \$reg_or_mem 增援 eax 來源 RE(6 筆)
- [ ] ch09-33 文本(批次進行中:09-13 執行中)

## 第 14 輪 ✅(AFM 完全破解+開場過場端到端+文本過半)
- [x] **AFM 格式完全破解**(sonnet):10-opcode 增量繪圖 VM(Lo Yuan Tsung 1993);
      派發 0x36c9e/跳表 0x5276a/framebuffer=VGA 0xA0000;289 幀(9 資源)逐位元組驗證;
      decode_ani.py;視覺全命中 dosbox 分鏡(屠龍/logo/金鎖…)→ doc39
- [x] **Go AFM VM 移植+開場接入**(旗艦):internal/afm(容器+VM);執行期解玩家 ANI.DAT
      (不夾帶版權幀);title cutscene 9 幕串接進選單;afm_test 驗幀數 96/51/35;
      無 ANI.DAT 退回 FDOTHER 捲動 fallback
- [x] **AFM 播放器排程 RE**(sonnet):play_afm(index,delayMs,skippable);毫秒校準 0x3dc9f;
      5 呼叫點釘死(開場 3/4/5/6/7/8/0/1,delay 90-15ms;idx0/2=章節過場非開場);
      title.go 換真值排程(拿掉月亮 idx2、各幕 delay、skippable 旗標)
- [x] **ch14-18 文本**(sonnet):229 句;ch01-18 累計 747 句;ch18 永久劇情死亡標記
- [x] **0x1f73f FDOTHER 靜態幕 RE**(sonnet):開場 2 幕靜態=①守護者(#100+pal#99,esi=0x1c2)
      +⑥滿月浮空城(#75+pal#76,esi=0x0a,dosbox frame168-173 逐像素吻合);機制 memset黑→
      載調色盤→blit→淡入→BIOS tick 忙等(修正原 KB「BGM/SFX」誤判);⑨惡魔臉排除是 0x1f73f(待下輪)
- [x] **開場過場插 2 靜態幕**(旗艦):cutScript AFM+static 交錯腳本;frame165 守護者/frame645 浮空城驗證
- [x] **歷史文本轉錄快照（非可玩覆蓋）**(sonnet 流水線 6 批):當時記錄
      ch01-33 共 1452 句與 speaker 場景本地表現象；這只代表文字資產曾被轉錄，
      不代表 33 章 handler、對話索引、一般玩家路徑或重製已完成。現況以本檔頂端
      121 個 story/cutscene、9 個獨立 script、50 個 handler binding、62 個
      fallback 稽核為準；未證實的身世標籤不作為 runtime 語意。
- [x] **speaker→頭像機制 RE**(sonnet):0xFFEF operand→0x12C60 查[0x53A45]/[0x53BF7] byte[+7]=DATO;
      三推論裁決(①部分成立=陣列重填+雙定址②怪物表不成立③字母碼是 render_story.py operand 洩漏 bug);
      **story JSON 零修改**(現行最忠實);修 render_story.py operand-skip;doc14 修正
- [ ] **開場配樂曲號 RE**(bgm-title 執行中):play_bgm 開場鏈曲號→FDMUS 檔(取代猜測 FDMUS_004)
- [ ] 開場分鏡⑨惡魔臉來源 RE(疑另一機制或 ANI.DAT)
- [ ] ch21/22 \$reg_or_mem 增援 eax 來源 RE(6 筆)
- [ ] 待展開(位址已釘):0x3453E 額外檢查、tag==0x27 sentinel、[0x53BF7] 表用途

## ⚠ 歷史快照：全 33 章劇情文本「大部分轉錄完成但尚未接進遊戲」（2026-07-27；已由本文件頂端 2026-08-09 稽核取代）

以下數字與缺口是當時的記錄，只保留用來追溯「轉錄」與「接線」曾被混為一談的原因；
不要把它們當成目前覆蓋率。現況一律以[`58`](58-fd2-exe-re-coverage.md)的可重生
數字為準；2026-08-27 的現況是121／9／57／55，postbattle為24 active／0 blocked。

**症狀**:尚未接 script 的 remake story 節點仍會使用短佔位文字；已接的節點則會載入對應劇本。
**查證**:目前實際 `campaign_full.json` 有 **121 個 story/cutscene 節點：9 個有 direct `script`、33 個有 `handler_binding`**；
其餘 79 個沒有 script/handler，仍依 node lines/default fallback，不能把 33 章轉錄宣稱成已接入。
`assets/story/ch01~33.json` 的 33 章 1452 句轉錄存在，但不等於每個 story node 都有正確 scene/line 對映。
**根因**:各自完成、接線沒人做——
- 「全 33 章文本完成」(story 流水線 6 批)✓ 真的轉錄好了
- 「全 30 章 campaign 節點生成」✓ 節點圖生成了，但不代表 story 對映完成
- **部分節點接 script、其餘仍依 placeholder/default fallback** → 兩者尚未完全連起來 ✗
**教訓**:子系統各自報「完成」不等於整合完成;跨模組「接線」要獨立驗(truth-in-code,
配 rulebook/63)。使用者實玩才揭露——沒實玩/沒查,文件會一直顯示「完成」。
**2026-07-27 當時修法狀態**：9個節點有 direct script、44個由 handler binding 供應
過場資料；其餘分類與14 active／10未綁定只保留為歷史。後續已達57個
handler-bound與24 active／0 blocked；不得再用本段重開已接節點。
- [x] **story-script coverage audit tool**：`tools/audit_story_script_coverage.py` 以唯讀方式列出 story/cutscene、coverage role、script、scene、handler、generated skeleton 與 next。2026-08-02 實測為121個 story/cutscene、9個 direct script、44個 handler-bound、68個 fallback；fallback 再分為30個 retreat、23個 rumor、10個 unbound postbattle與5個 generic。24個 postbattle skeleton 中14個已啟用、10個未綁定；postbattle skeleton 依主迴圈證實的零起算關係定位。數字變動時必須以工具輸出與對應回歸一起更新，不可只改文件。
- [x] **raw ch03 post binding（玩家第4戰）**：handler `0x231bc` 只有 FDTXT_004 dialog `0x231e5`、persistent sync `0x231ed` 與 `set_chapter4` `0x231f2`，無 unknown 或 runtime layout 需求；由既有 generated mapping 提升為 authored binding，接入 `postbattle_ch04_persist→town_ch05` 並納入全章 owner regression。
- [x] **raw ch04 post binding（玩家第5戰）**：Docker Capstone 證實 `0x2324c→0x233c6` 的 X/Y/pose 陣列（各 7 bytes）、slots0–6、special slot41 `(12,8,pose0)`、camera raw `(6,4)`；map4 raw roster frontier=50，FDTXT_005 index9 count-aligned 為 scene5 lines0–16 加 scene6 lines0–1。authored binding 現接 `postbattle_ch05_persist→town_ch06`；未宣稱 renderer parity。
- [x] **raw ch05 post binding（玩家第6戰，E1）**：Docker acting exporter 解碼 resource27 為3 beats、slot34/pose2；raw map5 `enemy_ally_total=40` 且 group3僅一筆，handler `0x10b4e(3)` 因而保存 `spawn_groups[3]=1`。`0x232b8` pan `(5,14)` 依既有 tile ABI materialize `(120,336)`，FDTXT_006 index6的19句已由raw control重生成typed版面。IDA Pro 9.4 直接確認 table index5=`0x23296`，並沿 `0x232e3→0x231df` 真實共享尾段固定 JOIN13→spawn3→pan→ACT27→dialog→sync→chapter6順序；正式輸入已完成40→41 slots、`postbattle_ch06_persist→town_ch07`與存讀檔。`0x2cad7` 原版戰間 outcome與一般玩家畫面仍未達E2。證據見[`fd2_ch05_post_native_dialogue.md`](../data/ida/fd2_ch05_post_native_dialogue.md)。
- [x] **raw ch12 post binding（玩家第13戰，E1）**：IDA Pro 9.4 與 table bytes 確認 index12=`0x2389f`，即使 IDA 導覽把它併在 `sub_237D5` 內也不可改寫真實入口。FDTXT_013 index9 經 address context 依序展開 ch13 scene3 lines0..5、scene4 lines0..5，再由 `0x238d0` sync、`0x238d7→0x237c8` JOIN3、`0x237d0→0x231f2` chapter13；authored binding 已接 `postbattle_ch13_persist→town_ch14`，跨 scene與尾段順序均有永久回歸。`0x2cad7` outcome及一般玩家畫面仍未達E2。
- [x] **raw ch08 post binding（玩家第9戰）**：Docker acting exporter 解碼 resource36 為 5 beats、slot47/pose0；raw map8 `enemy_ally_total=60` 且 group4 僅一筆，handler `0x10b4e(4)` 保存 `spawn_groups[4]=1`。`0x235d8` pan `(6,1)` materialize `(144,24)`，FDTXT_009 index4 對 ch09 scene4 lines0–4；authored binding 現接 `postbattle_ch09_persist→town_ch10`，未宣稱 renderer parity。
- [x] **raw ch11 post binding（玩家第12戰）**：Docker Capstone 證實三個 14-byte arrays、slots0–13，special slot2 最終覆寫 `(10,4,pose0)`，camera raw `(14,0)`→`(336,0)`；map11 60-slot frontier。Docker acting exporter 解碼 resource45 的 slot8 special frames（0 與 6 beats），FDTXT_012 index3/4 對 ch12 scene3 lines0–2、scene3 lines3–9；authored binding 現接 `postbattle_ch12_persist→town_ch13`，未宣稱 renderer parity。
- [x] **raw ch13 post binding（玩家第14戰）**：Docker Capstone 證實三個 16-byte arrays、slots0–15，special slot0 最終覆寫 `(0,0,pose0)`，camera raw `(12,10)`→`(288,240)`；map13 70-slot frontier、group1 僅一筆。Docker acting exporter 解碼 resource47 為 4 beats、slot67/pose2，FDTXT_014 index2/3 對 ch14 scene0 lines8–17、scene1 lines0–6；authored binding 現接 `postbattle_ch14_persist→town_ch15`，未宣稱 renderer parity。
- [x] **raw ch24 post／玩家第25戰 E1 垂直切片（owner 與拓撲勘誤）**：固定版FD2.EXE的IDA Pro 9.4／Docker Capstone已共同證實table index24的raw entry`0x24df2`、FDTXT_025 index6/7、PAN raw`(4,16)`→`(96,384)`、raw`0x10b4e(2)`、ACT resource75與共享`0x112a5`建構器呼叫；ACT解碼為4 beats、slot70/pose2。`0x24e7b push 0x1d→jmp 0x237c8`使共享尾段實際消費角色29，不是direct-entry角色14。舊86-slot說法已撤回：`ch25.json`現以party16＋group0(46)=62開場，第6回合event56追加group1(8)成70；戰後再追加唯一group2成71，group255不物化，正好讓ACT75操作slot70。正式binding已接`postbattle_ch25_persist→town_ch26`；Docker／Xvfb回歸通過完整勝利交接、JOIN26／29順序、持續隊伍與town save/load。尚缺未修改一般玩家原版E2。完整位址與雜湊見[`fd2_ch24_post_ida.txt`](../data/ida/fd2_ch24_post_ida.txt)。
- [x] **ch24 raw role-argument correction（已由正式 E1 消費）**：`remake/assets/cutscenes/handlers/ch24_post.json`以原始`source.addr`保留`join 26`（`0x24e6c`）與堆疊參數`join 29`（`0x237c8`），不再保留錯誤的`raw_append(14)`；上述正式垂直切片驗證兩筆持續紀錄與存讀檔，但不提升為一般玩家E2。
- [x] **raw ch25 post binding（玩家第26戰，E1）**：Docker Capstone 證實 `0x24e80` 的 `0x233c6` caller 以16 slots、camera raw `(9,5)`→`(216,120)` 寫入 map25；Docker acting exporter 解碼 resource77(slot1/pose2)、78(slot2 pose2→1→special pose0)、79(slot0/special pose2)、80(slot0 pose3→2→slot2 pose2)。FDTXT_026 string5–11 已由 raw glyph/control stream 對到 ch26 scene2 lines0–14、scene3 lines0–17、scene4 lines0–7；IDA Pro 9.4 又固定主表 index25=`0x24e80`、event state entry12 的兩個分支及共同 sync/chapter26 尾段。authored binding 已接正確 owner `postbattle_ch26_persist→town_ch27` 並有編譯與 campaign regression；2026-07-29 的63/63 count-aligned勘誤仍有效，未宣稱 renderer parity或一般玩家 E2。
- [x] **raw ch06 event26→event25→post conditional frontier（玩家第7戰，E1）**：IDA Pro 9.4 固定 map6 六格 selector0 event26=`0x3499B`：觸發單位 raw `+6 != 0` 才以 `0x3419C(9,27,0)` 清 slots9..27 的 `+0x34` 低四位並寫 state16=1。enemy turn10 event25=`0x34924` 先要求 state16==1，才依序 spawn group2→pan `(16,10)`→ACTING30→FDTXT_007 index2→寫 state17=1；未踏格反例不增援。ACTING30 直接引用 slots34..43。先前「slot43 是96-slot空白 record」已撤回：runtime 是 party9＋group1 25=34，再 append group2 10=44。authored scenario 現保存完整 gate／事件順序，已追蹤的 ch06 post handler 亦由錯誤線性稿修成雙層 CFG：state17==1 精確細化為44 slots，再讀 slot43 raw byte5 bit0；active 才 layout→index4→JOIN12，否則 index5。FDTXT_007 index4／5的8＋4句現由raw control重生；兩條正式輸入路徑分別驗證JOIN12正反例、`town_ch08`與存讀檔。`0x233B2`雖位於兩個互斥分支仍只計一個caller；尚缺一般玩家 DOSBox E2。證據見 [`fd2_ch06_post_event25_ida.txt`](../data/ida/fd2_ch06_post_event25_ida.txt)與[`fd2_ch06_post_native_dialogue.md`](../data/ida/fd2_ch06_post_native_dialogue.md)。
- [x] ch01 開場三幕(王城父子/草地悠妮蓋亞/遇海盜)手動接線+轉錄 FDTXT_033/032(intro-scenes)
- [x] **ch01 開場三幕背景圖 RE+接線**(使用者實測發現對白疊在戰場地圖上,非王座廳/草地,2026-07-04):
      RE 修正 doc23 §4 誤記(「FDTXT 序幕『影像』資源」不存在,FDTXT 純文字)——真正背景是
      **暫借章節 32 時 `0x1088d` 順帶載的 FDFIELD 組32(資源96/97/98)= 18×51 複合地圖**(王座廳→長廊→
      草地),與戰場同一 tile 渲染器;已渲染驗證(`extracted/maps/map32.png`)逐像素對齊 dosbox 參考圖
      + 使用者原版錄影。序幕尾端 `[0x53c03]=0` 還原真章節,「遇海盜」對白疊在**真戰場地圖 map0**(非另一
      張圖)。remake 加 `campaign.Node.Map/CamX/CamY`(story 節點固定鏡頭背景圖)+ `main.go` `storyBG` 模式
      (鏡頭不跟游標、不畫單位/游標/HUD);`campaign_full.json` 三節點接線(palace/meadow→map32,
      pirate→map0)。截圖驗證王城幕=雙王座紅毯廳(對照 orig_02_dialog_02_king.png)。
      **教訓**:另一 agent 曾提案「背景已在 BG_BG_\*.png,只需配對」,經抽樣檢視(320×100 全景走廊,
      無王座/紅毯任何痕跡)證偽——套用前先驗證,不可盲信「已抽出」的斷言(rulebook 62)。
      另踩雷:`~/.local/share/fd2_re/assets/`(玩家/測試用資產覆蓋目錄,`assetPath()` 優先讀它)有舊版
      campaign_full.json 快取(缺 ch00_palace/meadow 分幕),測試前先同步 repo 最新版才看得到真結果
      (使用者已驗收+ commit;team-lead 另修 play.sh 每次啟動先清 XDG scenarios/story 影子,一勞永逸)。
- [x] **王座廳 NPC 擺位**(使用者驗收背景後指出「王座是空的、索爾沒出現」,2026-07-04):RE 出 FDFIELD 組32
      出場位置段(資源98)直帶場景 NPC 座標+肖像,同戰場單位 roster 格式;**國王 portrait48@(7,5)+
      王后 portrait66@(10,5)** 頭像圖核對(`DATO_048/066_m0.png`=戴冠鬍鬚男/紫髮女)完全對上
      `f_006.png` 左王/右后。索爾在該格出場位置表無對應項(原版走 0x3231b 內 `push1/3/5;call 0x10b4e`
      另一條登場路徑,未逐一 RE),故索爾位置(fig0 @(8,8) dir2)是目視 f_006 定位、非 FDFIELD 直讀,
      已在 doc23/campaign_full.json 誠實標記。remake 加 `campaign.Actor{Fig,X,Y,Dir}`+`Node.Actors`
      (story+Map 節點靜態擺位,複用 battle.Unit/drawUnitSprite 畫法、無戰鬥邏輯),`story_ch01_palace`
      接 3 actor。截圖對照 f_006 吻合(國王/王后坐正確王座、索爾紅毯中央背對鏡頭)。
      **順帶發現**(未實作,留給 ch02-33 接線時參考):同一出場位置表在草地段(row42/46/47)另有
      portrait0×2(索爾+疑似另一己方角色)+portrait4(亞雷斯)+16 個 portrait68/69 走廊守衛,
      可比照本次做法補草地/走廊 NPC。
- [x] **ch01 開戰隊形 deploy_cells 核對**(使用者指出「索爾隊伍站位都是錯的」,2026-07-04):
      格子座標本身(FDFIELD `own_deploy` 直讀)已驗證正確,問題出在**逐人分配順序**——用 fig sprite
      外觀(fig4=藍盔=亞雷斯/fig9=紅髮=悠妮/fig30=機甲=蓋亞)逐一核對 `orig_03_battle_start.png`/
      `f_029.png`,發現影片是「索爾+亞雷斯緊鄰、悠妮稍右、蓋亞最右」,但 `ch01.json` 原
      `deploy_cells` 陣列順序配上 `party` 順序會把亞雷斯/悠妮的格子配反。交換
      `deploy_cells[1]`/`[2]` 修正,隔離 Xvfb + xdotool 送 Enter 清對白後截圖(before/after 對照)
      確認吻合。**除錯插曲**:FD2_SHOT_CUR 測試一度看似「怎麼設都沒用」,查出是地圖只有 24 格高
      (576px)但視窗 400px,camY clamp 上限只有 176,導致 curY=20/21.5/23 全部 clamp 到同一個畫面
      (誤判無效);換更小的 curY(如 15)才看出真的有作用——clamp 邊界會讓「看似無效的截圖測試」
      其實只是撞到同一個 clamp 上限,不是機制真的沒用,下次遇到「怎麼測都一樣」先檢查 clamp 範圍。
      → doc44 §2.5 定案(信心分級:格子=FDFIELD 直讀高信心,逐人配對=影片外觀反推中高信心非鐵證)。
- [ ] ch02-33 全章 story 節點接 script(gen_campaign 修+重生成)— 等 ch01 落地後做
- [~] ch02-33 全章 story fallback：runtime 對精確 `story_chNN` generic node 自動掛 `assets/story/chNN.json`，讓已匯出的可編輯完整劇本取代節點短 fallback lines；named/pre/post cutscene 不套用此 heuristic，避免重播整章。ch02/03 handler 仍待逐段 beats 接線。
- 🟡 **ch01 開場 Phase 2 實作(doc46 D1-D6,2026-07-04,待使用者驗收才打勾)**:使用者三輪回報後
      team-lead 先做「原版開場逐幕時間軸」(doc46)才動手,這輪照時間軸把 D1-D6 全部實作:
      **D1/D2 背景重構**:`story_ch01_palace` 拆成 `story_ch01_palace_throne`(map32 王座廳)+
      `story_ch01_palace_path`(map32 草地小徑,原「meadow」節點誤用棚)兩幕,`story_ch01_meadow`
      **改名為 `story_ch01_forest_duel`+`story_ch01_forest_discover`,背景從 map32 改指 map31 密林**
      (先前張冠李戴的核心 bug);map31 actor 用 FDFIELD roster 直讀(索爾19,46/亞雷斯19,47/
      蓋亞5,43/悠妮5,44);`portrait75` 是商店店員 NPC，不在 00-41 可入隊角色範圍，**未擺放**。
      **D4 行軍蒙太奇**:新增 `story_ch01_march`(map0,無對白,`auto_advance`
      180 幀自動轉場,索爾走位代表隊伍,簡化版,doc 誠實標「近似非逐幀重現」)。
      **D5 分段播放(核心)**:`campaign.Node` 加 `Scene` 欄(只取 Script 檔 `scenes[]` 裡 label
      對映的那一段,不再攤平全部劇本);改「每段一個 story 節點」而非 Node 內 sub-scenes,
      保留 `FD2_CAMP_NODE` 可跳任一幕驗證。**D6 走位動畫**:`campaign.Actor` 加
      `FromX/FromY/WalkFrames`(進場走位,重用 `battle.Unit.OffX/OffY` 插值)、`Node.ExitWalk`
      (退場走位,索爾沿紅毯走下場~1.5s);新增泛用**淡出/淡入轉場**(`storyFade`,0.6s/次,
      story 節點間一律套用,不再硬切)。**除錯插曲**:forest_duel 一度以為亞雷斯(fig4)沒畫出來,
      加 debug 座標印字才確認兩個 actor 都在正確位置、只是 FDFIELD 給的座標剛好只差 1 格(y46/47)
      造成兩張 24×24 sprite 緊貼——不是 bug,是資料本身就這麼緊,已移除 debug 印字。
      驗證:每幕獨立截圖 + 相鄰幕轉場(throne→path 含退場走位+淡出淡入全程截圖)+
      discover 幕走位動畫三階段截圖(進場遠/中/抵達)+ march 幕靜默→自動轉場→抵達海島全程截圖,
      build+test 綠、gofmt 乾淨。**D8(戰前 MAP/TURN 資訊畫面+行軍確認 UI)不在本輪範圍,已登記獨立項**。
      → doc25 §7.5.1 已修正範圍(戰場進場直接定位仍成立;cutscene 幕內走位是另一機制,已推翻舊結論)。
- [~] **D8:戰前 UI**(doc46 附帶發現):Docker/Capstone 已釘 `0x1a30b` battle-entry choreography：
      `0x1f1cc(0x52)`→20ms→`0x1f30a(0x52)`、64000-byte indexed surface；`0x1f42d` 已釘為
      LMI1 entry #0x52 的雙側五幀 slide-in/restore（x=85−offset、165+offset；offset=100..0）、
      後續 `0x1a813/0x1a866` dispatch；`0x15f0e` frame ABI 亦已釘為 offset-table + RLE
      decode + stride blit；`[0x53a81]` 已由既有 loader trace 對位為 `FDOTHER.DAT#5`
      的 `LMI1`，remake `fdother.ParseLMI1`／`LMI1Entry.BlitAt`（透明 preserve + mirror）
      與 player-asset regression 已補（#0x52=72×14；directory offset 不是 RLE 結尾）。
      證實不是 `resetBattle` 直接跳過的空白階段。仍待釘死
      MAP/TURN/ENEMY/FRIEND/NPC 欄位與 YES/NO input 的資源／字串 ABI，再做 remake shell 與截圖，
      不把 resource `0x52` 或 `0x51e81` 猜成畫面名稱。
- [x] **D8 scope correction / raw entry step**：重新檢查官方 `0x1a30b`，確認本體沒有 `0x15f84` 文字呼叫；只對 selector/gate 通過的 raw record 做 `+0x40 += +0x42/5`、上限 clamp 與 indexed redraw，之後才呼叫 `0x1f1cc/#0x52`。新增 `NativeBattleEntryStep` regression；MAP/TURN/ENEMY/FRIEND/NPC 資源與 YES/NO input 仍未證實，不能由 `0x52` 命名。
- [x] **0x1a30b shared-caller correction**：Docker Capstone xrefs 固定 callers=`0x135c5/0x17154/0x17272`；`0x1716f/0x17241` 旁有 FDTXT_000 `0x19c/0x1a4` 互動訊息，`0x1728c` 處理 selector flags。故 `NativeBattleEntryStep` 保持 reusable raw record primitive，不接成 D8-only preparation action。
- [x] **UI-11 remake shell screenshot artifact**：Docker `fd2-go-test-local` + Xvfb 實跑 `FD2_CAMP_NODE=preparation_ch02 FD2_SHOT_FRAME=30`，產生 [`preparation-remake.png`](../figures/preparation-remake.png)（640×400，2× 320×200）；只證明 editable preparation node 與地圖背景/overlay 可呈現，不取代原版 DOSBox 差分，也不關閉 MAP/TURN/YES-NO evidence gate。
- [x] **native raw action-bit writer**：Docker Capstone 固定 `0x13512(index)` 設 `record[index*0x50+5] |= 0x80`、`0x13536` 全表清 bit7；新增 `battle.SetNativeRecordBit7`／`ClearNativeRecordBit7All` 與 bounds/other-bit regression，保留 raw offset，不把 bit7 強行命名成回合狀態。
- [x] **post-resolution inventory occupied-count ABI**：Docker Capstone 固定 `0x1b8a6(unit)` 掃 `record+0x0a+2*i` 八個 cell；bit7 clear 只增加 occupied count，函式不驗證 compact prefix。caller 再以 count 作 slots `0..count-1` 上界；洞會讓 stale item byte 進入掃描。bit7 set 是 `0x1bb8c` 使用的 reserved 空格。已更正 free-slot／prefix 斷言並以 `battle.NativeInventoryOccupiedCount` 保存 exact count。
- [x] **post-resolution inventory reservation writer**：Docker Capstone 固定 `0x1bb8c(unit,item)` 取第一個 flag bit7 cell、清 flag、寫入 item byte、回傳 1/-1；新增 `battle.AssignNativeReservedItem` 與 first-cell/none regression，保持 raw item/category opaque。

## 待辦:實測回饋(使用者 playtest,2026-07-03)
- [ ] **開場過場節奏 3x 太快 RE**(dragon-fx2 DOS 對比發現,doc39 §10.8):原版魔王立繪捲動
      (esi535→0)貫穿全開場、與各 AFM 幕交錯(暫停播幕→續捲),貢獻 ~16s 延遲;remake 把捲動
      搬到最後單播→開場 5s vs 原版 14.7s。修需先補 0x11eb0/0x1f894 逐指令(捲動如何在 AFM
      直寫 framebuffer 後接回)。使用者已 OK 開頭閃光(#9),此為獨立節奏落差,低優先
- [x] **序章劇本 staging 機制 RE**(使用者指出 #3=劇本機制沒 RE 完整,2026-07-03 反組譯+dosbox 220+ 張連拍
      複驗收尾)→ **定論:主角隊直接定位,原版無行軍動畫**。0x3231b 使用直接 spawn(`0x10b4e`)、
      攝影機平移（`0x13185`/`0x135dd`）與新增群組索引轉場（`0x32999`）；這些都不是單位行走。DOSBox 全程重跑序章開場
      未見任何單位行走動畫或世界地圖段落;玩家記憶「走到地圖中央」疑與攝影機平移視覺效果混淆。
      remake 現行 focusOnParty(純鏡頭對準)+ spawn_party(直接定位)已忠實,#3 非 bug,不需補行軍動畫。
      → `docs/knowledge-base/25-battle-event-system.md` §7.5.1
- [~] **playtest 8 項修正**（前次批次已提交，#7 已判定不是 bug；目前無執行中代理）：
      #1 方向鍵按住持續移動、#2 預設沒開場動畫、#4 移動後 ESC 取消退回、
      #5 action overlay／command grid 的原版 side-by-side 視覺對照、#6 地圖狀態欄還原原版、#8 單位走完轉回正面朝向
      → **batch1 已 commit(0f32d25)**;#7 非 bug(kill 誤殺);#3 部分(鏡頭對準部隊)
- [ ] **#9 法術特效時序**(待使用者釐清):playfix 靜態審查=攻擊系法術路徑乾淨無殘留;
      真根因疑「治療系法術 target=1 無全螢幕演出(只文字)」→ 打治療咒後緊接敵方攻擊演出,
      被誤認成法術效果延遲出現。修法需先 RE 原版治療咒視覺(閃光/數字浮現/僅改血條),
      或使用者釐清實際現象,不瞎編視覺
- [x] **序章場景轉換打通**(2026-07-04,使用者驗收 OK,commit 2c5adda):王座廳/草地/遇海盜改用
      真 tile 地圖背景(map32/map0)+ 固定鏡頭,非戰鬥圖。RE 定論:0x3231b 暫借章節32 載 FDFIELD
      組32(紅毯雙王座→長廊→草地縱向拼接),與戰場共用 tile 渲染器。story.Node 加 Map+CamX/CamY,
      main.go 加 storyBG 鎖鏡頭/擋游標。→ `23-boot-title-and-scenario-flow.md` §4
- [x] **援軍 stale-cache bug 根因修復**(2026-07-04,使用者報「援軍不該一開始出現在地圖上」):
      根因非 code bug——`~/.local/share/fd2_re/assets/scenarios/ch01.json` 是舊版 `initial_groups=[1,2,10,11]`,
      XDG 快取層優先蓋掉 repo 已修正的 `[1,2]` → group 10/11 開場即 OnField=true 出現。**治本**:XDG 是給
      版權衍生素材(sprites/maps/music)+ 玩家編輯版覆蓋用,scenarios/story 是原創內容不該進 XDG;已刪 XDG
      scenarios/story 影子 + play.sh 每次啟動先清,dev 一律以 repo 為真相。→ 記憶 `fd2-intro-cutscene-bg-and-userdata-cache`
- [x] **過場腳本機制第一性原理解答(doc47)**(2026-07-04,使用者問「RE 為何沒還原 staging」,旗艦親做):
      方案 b 證偽=FDTXT 純對話碼無 staging;方案 a=序章 handler 0x3231b 逐 beat 全轉錄。
      原語翻新:0x135dd=平滑鏡頭平移、0x15f84=對白播放器(doc23 舊判「逐格貼圖」誤)、
      0x1366a=演出(acting)播放器（direct bank 格式與 106 筆資源已解出；normal=逐格搬移、
      special=原地 pose，zero-special 保留三 tick 時序）、0x112a5=入隊(0/9/4/30)。
      重大:王城→草地=同 map32 鏡頭平移轉場非淡出換景;對白與演出逐條交錯;海島幕 3 個平移點。
      → remake 修正指示 doc47 §4；尚待逐章把 direct acting 資源接入可編輯 cutscene 節點，
      並以實機截圖核對 renderer/presentation 差異（不猜測性補 handler 語意）。
- [~] **王座廳 NPC 擺位**（已有部分盤點，目前無執行中代理）：國王/王后坐王座 + 索爾站紅毯中央,對照 f_006.png;
      story 節點加 actor 擺位欄。RE 查 FDFIELD 組32 是否帶 NPC roster(sprite id/cell 直接來自原版)
- [x] **ch21/ch22 pre-handler**：FDTXT_022 index0（11句）與 map21/70-slot、pan(16,28)、acting67 已接 editable binding；`story_ch22` 已接回原版 pre-handler，compiler/campaign/battle regression 通過。
- [x] **ch21 合成不足分支的戰間存檔邊界（2026-08-09）**：`inventory_recipe` 材料不足時實際走可編輯（editable）的 insufficient→town 節點，清除戰場暫態但保留持續隊伍；Docker/Xvfb regression 會在 town22 等價節點存檔、清空暫態後讀回金幣、隊伍成員與加入順序。這只補足戰後城鎮／存檔回歸，不猜 raw `ch21_post` handler，也不宣稱一般玩家 E2。
- [x] **外部資源／城鎮流程交叉盤點**：公開資料確認 `FDFIELD.DAT` 是可替換的外部場景層，且章節間存在 preparation、商店、教會、存讀檔流程；後續以 DAT provider + battle→town/prep graph 實作，未將網路資料當 binary 格式硬證據。
- [ ] **社群行為 oracle 對照**：逐項把 FD2.EXE 修改表中的入隊、隨時存檔、等級上限、寶箱持久化轉成可編輯規則與 regression；先挑 save/chest 兩項和目前 persistent flow 最相關者。
- [x] **ch22_pre control-flow／視圖來源**：固定 16-slot deactivate loop、`0x11df2`
  immediate `palette_update`、三段 tile-step PAN 與 `0x336e5→0x24618` 的
  Y+5 indexed cursor。IDA 已證實 `0x205da` 將鏡頭／絕對游標／可見游標全數
  重設為零，`0x135dd` 只同步前兩者；正式 binding 已接 `story_ch23`，並以
  Docker＋Xvfb 回歸進入 `battle_ch23`。`ch23.json` 未宣告
  `runtime_append_groups`，所以 handoff 只採正式戰場重建，不猜測部分 handler
  陣列共享；未修改一般玩家 DOSBox E2、renderer parity 與 postbattle_ch23
  城鎮／商店／整備／存檔仍未關閉。證據見
  [`fd2_ch22_pre_view_reset_ida.txt`](../data/ida/fd2_ch22_pre_view_reset_ida.txt)。
- [x] **ch23/ch24 pre-handler**：FDTXT_024 index0/index1（14句）與 map23/70-slot、spawn group1、四段鏡頭已接 binding；`story_ch24` 已接回原版 pre-handler，compiler/campaign/battle regression 通過。
- [x] **ch23 post mapping boundary（歷史段落，已由2026-08-21垂直切片取代）**：raw table index23=`0x24c1e`、FDTXT_024 index2／3、兩段30×8與12×5迴圈、312×192 staging、BIOS tick gate、ESI 0..59及DAC `0xe0..0xef` 的主證據保留在 [`fd2_ch23_post_ida.txt`](../data/ida/fd2_ch23_post_ida.txt)。本段原有「入口 latch／transient offset 尚未知、僅候選且不能接 campaign」的斷言已被後續 IDA data-flow 與正式 E1 adapter 直接推翻，故刪除其重複細節；現況以本檔有效佇列、[`58`](58-fd2-exe-re-coverage.md)與同日歷史條目為準。
- [~] **ch24/ch25 pre-handler**：`0x24b4d` 四段 transition count 已 lower 為 `transition_reveal`（20/20/20/60、20ms/frame），FDOTHER#88 sub1 四次 SFX、index=-1 stop、handle release 已接，FDTXT_025 跨 scene 對白已接 `story_ch25`；尚待 indexed double-buffer visual adapter。
- [~] **ch25/ch26 pre-handler**：FDTXT_026 string0 已以 direct scene0 12-line mapping 接 binding（map25/70-slot、pan、acting76），`story_ch26` 已接回 handler。2026-07-29 已修正未加引號訊息計數，FDTXT_026 全量 63/63 count-aligned；這只關閉文字索引，不自動證明每個條件分支或 event61 玩家路徑。
- [~] **ch26/ch27 pre-handler**：FDTXT_027 idx0/3/4/5/6/7 已高信心對到 ch27 scene0 全部 21 句，新增六組 editable direct overrides 並接 `story_ch27`；共用 `0x24618` renderer 已完成。IDA／Capstone 又閉合 ch26_pre 返回時的 view `(camera 9,49; cursor 14,54; visible 5,5)` 與 selector0，`battle_ch27` 已資料化並由正式 runtime 消費。HUD 持續擁有者亦已閉合為 save-persistent gate A、process-persistent anchor 與 controller entry gate B=1，`battle_ch27` 已改用 `native_map_hud_inherited`，不猜章節常數；仍缺未修改一般玩家／CONTINUE 的同狀態 E2，以及 `0x24b14` item `0x64` branch 的其餘視覺行為，故不能視為完整章節流程完成。
- [x] **raw ch27 post／玩家第28戰流程（RUNTIME-E1）**：FDTXT_028 string7 已精確對到 ch28 scene1 lines11–15。IDA Pro 9.4 直接確認主表 index27=`0x25464`，該入口準備對話參數後跳到 `0x231df` 共用尾段，依序執行 dialog／sync_party／set_chapter28；低位址來源是真實共享程式碼，不是 exporter 污染。正式`story_ch28`以`runtime_append_groups`物化20名部署者＋groups1..7共44筆，group255的16筆只留source roster；64-slot勝利路徑已用具型別輸入完成五句、`preparation_ch29`及全新`Game`冷讀。舊80-slot斷言與玩家第27戰天空之鑰錯接均已撤回；證據見[`ch27-post-native-dialogue-e1.json`](../data/ui-traces/ch27-post-native-dialogue-e1.json)。未宣稱逐幀renderer parity或一般玩家E2。
- [x] **玩家第 28／29 戰前置處理器 owner 勘誤**：IDA Pro 9.4 的
  `0x51D71` 分派表與 `0x1088D(chapter)` 資源公式共同證實 raw index27
  `0x33C9D` 屬玩家第 28 戰（map27／FDTXT_028），raw index28 `0x33DBA`
  才屬玩家第 29 戰（map28／FDTXT_029）。舊版把 `ch28_pre` 接到
  `story_ch28`，且錯配 map27／slot70／ch28 party，已改為
  `story_ch28→ch27_pre`、`story_ch29→ch28_pre`。前者新增固定 20 筆
  deactivate→`+0x40!=0` 才清 byte `+5` 的受限原語，以及 `0x33CE2` 的
  relative cursor X+6/Y+5；後者以 slot76、group8=56 進場，群組實際數不符
  立即停止。兩條皆由正式 campaign handler 走到正確 battle node；這是 E1，
  尚非未修改一般玩家 E2。`0x35822` 的 pan→spawn→300ms→全白→200ms→
  baseline restore→redraw 與 `(group,y,x)` PUSH 勘誤仍有效。證據見
  [`fd2_ch27_ch28_pre_owner_ida.txt`](../data/ida/fd2_ch27_ch28_pre_owner_ida.txt)。
- [x] **FDFIELD 部署尾端匯出勘誤**：`0x1088D` 從
  `2+6*enemy_ally_total` 起讀 control 宣告的 `own_deploy` 筆 X/Y，完全不讀
  第三個 raw key。exporter 已停止用 `raw_key==0` 掃全表：map28 由16修為20
  筆重疊位置，map31／map32 的捏造部署格由1／2修為0；33張地圖的
  `positions=units+own_deploy` 由 regression 鎖定。scenario 的去重／螺旋補位
  仍明列為重製直達戰鬥近似，不冒充原始位置表。
- [~] **ch26 post item-gate branch**：`0x25186→0x24b14(0x64)` 是前 16 個 runtime slots 的 exact inventory search，無 camp/activity filter；成功臂無 `0x1b8e7`，天空之鑰不消耗，之後才 sync→chapter increment→persistent cleanup。FDTXT_027 idx8–12 / idx13–16 對應兩臂；仍需把 visual/effect calls 與缺匙 editable branch 資料化，不能只保留 generic ending。
- [x] **ch26 success palette-ramp lowering**：Docker Capstone 定義 `0x25052(start,delay)` 為 inclusive `delta=start..0` 的 `0x11df2(0,255,delta)`＋每步 delay；compiler 已 lower immediate start 0..63。synthetic descending/zero/invalid 與真實 `ch26_post.json` 六個 5/4/3/2、80ms calls 均有 regression。這是 palette ramp，不是 generic fade；`0x24618` 已有專用 adapter，其餘 renderer effects仍各自 fail-closed。
- [x] **撤回 `0x1f882`=vsync/sync helper**：Docker Capstone 展開 `ebx=0..63`、每次 `0x11d40(0,255,ebx)`＋2ms wait，故是 64-step native palette fade-out。compiler 現保留 exact `native_palette_fade_out(0..63,2ms)` payload；它與 `0x25052/0x11df2` 的 delta ramp 不同，runtime 在 indexed DAC adapter 未完成前有 regression-protected fail-closed。
- [x] **native palette pulse (`0x35e5a`)**：Docker Capstone 完整 body 固定 `0x11df2(0,255,delta)` 的 inclusive 0→63（8ms/step）、400ms hold、再 62→0（8ms/step）。compiler 以 exact editable `native_palette_pulse` payload 保存不對稱端點，並拒絕帶參數變體；runtime 現以 immutable baseline、127次 draw acknowledgement 與最後 baseline restore 執行 indexed DAC，schedule drift 在發布前拒絕。這是窄 `RUNTIME-E1`，正式 ch28 post binding 與原版 E2仍缺。
- [x] **ch29 staging wrapper (`0x33f78`)**：2026-08-25 的合法 IDA Pro 9.4 完整 body 與 Docker Capstone 共同推翻舊 `0x12cea(slot,x)` 解讀，固定 raw push-order `[y,x,slot]`→`0x12cea(x,y)`→`0x22253(slot,x,y,x,y)`。compiler 只保存 `NativeStagingPresent{Slot,X,Y}`；`story_ch30` 正式 binding 已接 `LOADCH`、21句對話與七個 caller，focus 前完整預建，story slot 只在 bridge 邊界發布，缺資產／視圖／來源時零可見修改。Docker／Xvfb 已由正式節點擷取 E1 畫面；一般玩家 E2、精確 tick／音訊仍待。
> **終局歷史段落勘誤（2026-08-27）**：下列 ch29 terminal 條目保存當時分段接線
> 過程；其中「正式 owner／campaign handoff／終局仍未接」已被本檔有效佇列第3、6項
> 與`58`終局列取代。正式`battle_ch30→ending`已消費前綴、蒙太奇、20段尾段、
> 定格與隊伍回顧；剩餘限制是一般玩家原版E2與精確音訊。不得由下列舊現在式句子
> 重開已閉合RE或把重製終局降回未接。
- [~] **ch29 post staged mapping**：四組對白已精確接到 ch29/ch30 authored lines；`0x12cea` focus、`0x25089` persistent cleanup、`0x17aa9` tick、dynamic palette loop、terminal `loadch` 與 `0x24618` indexed transition 均已有 runtime adapter/regression。`0x2bce5` 的前綴、party cycle 與 `0x2c194` raw tail 已分別保存；完整 indexed owner、輸入事件與 campaign handoff 仍未閉合，因此整支 terminal handler 仍不接 campaign runtime。
- [~] **ch29 post focus lowering**：`0x12cea` 已安全 lower 成 tile-step pan(22,23) 並通過 regression；cleanup與`0x24618`已接，仍待完整 ending owner／indexed handoff。
- [~] **ch29 post persistent cleanup**：`0x25089` 已 lower 為 editable `reset_persistent_roster_state`，並以 runtime/campaign regression 鎖定清 transient、回填 MaxHP/MaxMP；本 handler 的主要剩餘終局 gate 是 `0x2bce5` 的完整 indexed owner／輸入／campaign handoff。
- [~] **ch29 post tick wait**：`0x17aa9(1)` 已 lower 成一個 editable delay tick 並通過 compiler regression；`0x24618` 已接，仍待完整 ending owner／indexed handoff。
- [~] **ch29 post dynamic palette loop**：`0x11df2(EBX,255,0)` 已依 direct 0x3e→0 loop materialize 成 63 組 palette/delay beats 並通過 regression；`0x1088d` 的舊文字-only 說法已由完整 `loadch` 取代。`0x24618` 已接，party cycle／raw tail 亦已保存，仍待完整終局 owner。
- [~] **ch29 post terminal handler**：`0x25870 → 0x1088d` 不是純文字載入：它會載 FDTXT/FDFIELD、重建 unit buffer、從 persistent roster 複製 records、寫 map29 deployment 並 spawn groups。現已 lower 為完整 editable `loadch`（chapter30/map29/roster70/ch30 story+scenario），而非文字-only operation；`0x112a5` 已證實 persistent records 依 JOIN 呼叫 append，因此正常遊戲 slot order 可用 `partyJoinOrder` 表示。layout、動態 pan、`0x24618` indexed adapter、前綴／party cycle／raw tail 均已分段保存；完整 indexed owner、輸入事件與 campaign handoff 未閉合，故整支 handler 維持 fail-closed。`0x25970 → 0x2bce5` 返回後是 self-loop；map29／LOADCH 只能支持 internal chapter 29 終局候選，不能把它升格成玩家戰次或 campaign owner，**也不是** map28 戰後可接 `preparation_ch30` 的 handler。近似模式已有獨立 ending 節點執行前綴／蒙太奇／尾段與下列最終隊伍同步；這仍不是把整支 raw handler 接入正式 campaign。
- [x] **ch29 post layout data**：`0x257b4 → 0x233c6` 的 20 slots X/Y/pose 與 camera `(16,18)` 已存入 editable binding，並有 compiler regression；`0x112a5` 已補證 persistent ordinal=JOIN chronology。完整終局 handler 尚未接 campaign，不表示終局已可播放。
- [x] **ch29 post final pan**：`0x25937 → 0x135dd(11,12)` 已依 X-first/Y-second native ABI lower 為 tile-step `(264,288)`，compiler regression 通過；終局`0x24618`已接，完整 ending owner／indexed handoff 仍待。
- [x] **0x24618 indexed transition runtime closure**：editable schema與compiler保存tile/radial-radius、9-pass LUT `9..1`、5ms/500ms/4ms schedule及32-step `0x11df2` DAC ramp。Docker重讀證實`0x11df2`每次從immutable `[0x53a65]`取RGB再加delta／upper-clamp63，非對current DAC累積。runtime現在all-or-nothing preflight原始field、tile-aligned camera、actor provenance、selector cache、FDOTHER#3 LUT與FDOTHER#0 768-byte baseline DAC；`ComposeNativeTransitionFrame`逐pass執行terrain→first LUT→unit/foreground→second LUT→rect LUT→312×192 present。每個pass與baseline-derived DAC step皆需真實Draw acknowledgement，500ms tail固定30 ticks；拒絕時不改既有work/VGA。60Hz host無法重現5/4ms每次寫入的原始wall-clock，故只宣稱完整狀態／順序，不宣稱DOS timing parity。2026-08-09 合法 IDA 的直接交叉參照又固定 `0x24618` 的 8 個直接呼叫者（`0x245ce`、`0x252ee`、`0x25848`、`0x336e5`、`0x33bb9`、`0x33c09`、`0x33c66`、`0x33ce2`）及 `0x22046` 的 6 個直接呼叫者；這只證實共享消費端邊界，不把它命名成單一戰後角色演出。ch29 terminal仍由後續`0x2bce5` gate阻擋。
- [x] **RE-22046-INDEXED-PASS-SEQUENCE**：Docker Capstone 重讀 `0x22046`，新增 `fdother.ApplyIndexedTransitionPass` 保存第一 radial LUT→`0x127a9` middle redraw→第二 radial LUT→centered rectangle LUT 的不可省略順序；三段 geometry 先完整 preflight，缺 redraw/invalid second pass 不修改 buffer。LUT bank、double-buffer與Ebiten presentation已由strict runtime adapter消費。
- [x] **ch29 final 0x24618 arguments**：依 layout→focus 的 native scroll-offset writes，`0x25848` dynamic args 已定案為 tile `(6,6)`、radial radius `(10,step8)`，已寫入 binding/compiler regression並由runtime adapter消費；terminal handler仍因`0x2bce5`維持fail-closed。
- [x] **0x24618 pass-range/runtime boundary**：`0x22046` 的固定最後兩參數是 row range `[start_y,end_y)=[0,0xc0)`，不是 source_y 或 blit width；clip `0x138×0xc0`、radial step、`0x53a6d` LUT bank、`0x219ad` row clip均已接入strict indexed adapter。
- [~] **ch29 pre native unit presentation**：舊「6×(render+present+10ms)+2 ticks」結論已撤回。完整 `0x22253` trace 是前段 `0x22470` 11 次 LMI present/tick、中央 `0x22547` 6 次 10ms remap present+2 ticks、後段 `0x22656` 10 次 remap present/tick，合計 27 次 present；既有 `unit_present` metadata 不完整，維持 fail-closed。
- [x] **`0x22253` machine-readable schedule boundary**：`fdother.NativeUnitPresentSchedule` 嚴格保存三段 11+6+10 的27個 full present；後續已接 exact geometry、18／24-row bridge 與 battle-state Ebiten presenter。舊 six-frame shortcut 仍禁止；本項的「尚無 renderer adapter」是歷史狀態，已由 2026-08-21 `0x25535` 窄 `RUNTIME-E1` 取代。
- [x] **`0x22470` first-phase destination ABI**：direct arithmetic 已保存為 `NativeUnitPresentByteOrigin(x,y,camX,camY)=0x8088+24*(x-camX)+24*456*(y-camY)+456`。它是 456-stride indexed work-buffer byte offset，最後 `+456` 不可漏；raw helper 保留 offscreen signed result，clip 仍屬 caller/renderer boundary。LMI decoder／unit-layer/present adapter 尚待組合。
- [x] **`0x22470→0x4e85b` LMI write primitive**：`0x4e85b` 逐像素透過 `0x4e916` decode，僅非零寫 destination，等同既有 `LMI1Entry.BlitAt` 的 preserve-zero 規則。`BlitNativeUnitPresentLMI` 已將 #6 cell 與 verified byte origin 組合，對 offscreen origin fail-closed；其後 unit redraw/present/tick 與其餘 phase 仍待 adapter。
- [x] **`0x22470` eleven-pass intro executor**：`RunNativeUnitPresentLMIIntro` 固定走 #6 entries `0x72..0x7c`，每一 entry blit 後強制要求 caller 執行一次 redraw/present/tick callback；不得折疊為一張最終畫面。short table／nil callback 均 fail-closed。尚未接 GUI renderer。
- [x] **`0x22547` LUT geometry correction**：重做 `0x22046` argument
  mapping，撤回舊「dynamic radius」斷言。radius其實固定11、scale固定16；
  dynamic值是`startY=trunc((24*raw53ABD+15)/5)*lutIndex`。新增
  `NativeUnitPresentContractStartY`、`NativeUnitPresentLUTPass`與完整
  6+10 `NativeUnitPresentLUTFrames`，保存兩radial及中間rectangle的
  split-row geometry；raw globals仍不猜玩法名。
- [x] **`0x22547/0x22656` BUFFER-TRANSACTION**：
  `RunNativeUnitPresentLUTFrame`嚴格要求完整`0x25680` work/snapshot與
  256-byte LUT；每frame先restore snapshot，再執行first radial、
  mandatory full-buffer object redraw、second radial、rectangle及present。
  防止錯誤累積16次LUT或略過中間sprite mutation；Ebiten snapshot producer
  與present scheduler仍待。
- [x] **`0x22253` INDEXED-FRAME-COMPOSERS**：新增exact
  `0x25680` terrain-only snapshot、atomic object-only `0x127a9/0x129ec`
  redraw、source`+0x8088` stride456→destination stride320的312×192
  viewport copy，以及`0x22470` intro／`0x22547/0x22656` LUT frame
  composers。剩餘runtime gate縮小為Unit raw pose/motion、cycle globals與
  phase-specific snapshots，不再籠統寫成「沒有indexed buffer」。
- [~] **ch29 post BIOS tick wait**：`0x17aa9` 已證實比較 raw `word[0x46c]` 的相對差值；DOS BIOS 約54.9ms是強推論。舊 handler compiler 的「每 tick 3 個 remake frames」只保留為 editable beat 近似，不是原版精確時間；終局 `MontageCycle` 另以 55ms 作近似排程，仍須一般玩家 E2 才能提升 timing parity。
- [x] **native `0x22253` renderer adapter**：`0x22547→0x22046`、FDOTHER #3／#6、shared snapshot、object redraw 與 bridge 已閉合。2026-08-21 新增 battle-state 原子 presenter：預算11＋6＋18/24＋10個可見邊界，第六張 contract 畫出後才寫 `record+0/+1`，preflight／runtime failure 回復 unit、work、VGA。`0x25535` 動態最後槽位、command23離場／入場與 `0x33F78` story-array/focus caller 均已接正式 E1；後續只補各 caller 同狀態 E2，不再重解 generic compositor。
- [~] **chapter ending renderer (`0x2bce5`)**：IDA／Capstone 已完整固定 `0x2bce5..0x2c39b` 之前綴（FDOTHER `#0x36`、320×200雙 buffer、frame0/9、palette ramp、三輪重複 ramp、frame12..108、40/200次 composite）與 `0x2c405` 的 `0x1088d(0x1e)`、500次 raw staging、`0x2c548` montage 入口；`0x2c548` party-cycle 與 `0x2c194` raw tail 現已有原資源 executor／schedule。`0x29164` 已更正為 raw `+6==0`／非零分支；`0x2c194` 的三組表也已校正為每輪寫 record0、record1 各自的 `+7`、並以 `<0x4c` 即時計算各自 `+6`，第三組寫 `[0x540ff]`，不能再誤寫成單一 record 的高位欄位或首輪例外。`0x10620→0x4e031` 已證實在 portrait loop 讓當前角色完成後改走 final loop，但尚無確切按鍵映射。**2026-08-22 勘誤：**正式 `battle_ch30→ending` 現直接以來源約束 E1 合約播放前綴、原資源 party montage 與20段尾段，並停在 #59；不再需要 `FD2_APPROXIMATE=1`。FIGANI header 已改按 byte0／1／2 解析；實際80個終局資源的 byte1 全為0，故正式執行期已接可達的 `0x2939D` raw `+4..+7`、base scheduler、palette33覆寫及兩次交叉配對。任一資產或 raw roster provenance 缺失時整批回到可編輯結語。`0x25970→0x2bce5` 仍不可命名為 raw campaign owner；未閉合的是呼叫時 records/globals 的動態連續性、3% RNG重播、音訊／輸入及一般玩家 E2。完整 body 證據見 [`fd2_ch29_terminal_body_ida.txt`](../data/ida/fd2_ch29_terminal_body_ida.txt) 與 [`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt)。

- [x] **第29／30戰終局持續隊伍邊界（E1）**：第29戰勝利現由正式 raw ch28 post 同步隊伍→`preparation_ch30`→隔離存檔／讀檔回歸；第30戰以 campaign JSON 的 `ending_party_snapshot_on_win` 在直達 ending 的勝利邊同步最後戰場，避免角色結局仍讀戰前 roster。缺戰場或零筆身分符合時不移動游標且保留勝利結果；載入器拒絕非終局用途。這解除了重製正式 campaign 的終局資料邊界，但不證明 raw terminal owner、精確 renderer／輸入或一般玩家 E2。

- [~] **shared object redraw compositor**：`0x127a9` 的 `0x127e0` 不是單純 loop bookkeeping：active roster entry 以 camera-relative placement 選 24×24 descriptor，走 `0x4deda` raw indexed-RLE 或 `0x4de56` palette-band-RLE 寫 `0x53a49`；尾端 `0x129ec` 又在同 buffer 疊 map/object layer。`+5 bit7` clear→raw、set→band 已由 direct branch 關閉。`BlitNativeUnitLayer` 現以 raw slot／pose／movement／base-frame／active gate、camera bounds、cycles 及 pixel shift 完整表達 steady unit layer，且 preflight 失敗不寫半張 frame；它不接 GUI。`0x53a61` 是 global raw-key cache 的 pointer blocks，runtime index 是回傳 `slot×12 + pose×3 + cycle`，而非角色 group。仍待將 terrain→range→unit→foreground→HUD→viewport copy 組成 caller adapter；在此之前不得把 `0x22046` passes 或 `unit_present` 接成 native UI。
- [x] **`0x11cac` range-layer provenance**：Docker Capstone 釘住 redraw order 為 `0x11eee terrain → 0x122dc range overlay/mutation → 0x127a9 unit+foreground → 0x1acf3 HUD → 0x11eb0 viewport copy`。修正舊斷言：只有 modes1..5 展開固定 calls 到 `0x126f7`；mode6直接清 selected cell byte+3，7+直接return。`0x126f7` camera-bound 後以 `0x4deda` 寫 `buffer+0x8088`。
- [x] **`0x122dc` range call-table／asset closure**：Docker Capstone 完整直讀 modes 1..5，`fdother.NativeRangeOverlayPlacements` 保留原始 call order 的 1/1/5/13/21 個 `(x,y,descriptor)`；特別固定 mode3 centre=`#14`、mode5 的重複座標／不同 descriptor，禁止圖形化 normalize。`0x25c7d..0x25c92` 已證明 `FDOTHER#1→0x53a4d`；實檔 header 是 20-entry 24×24 four-mode-RLE bank，`0x126f7` 以 `base+6+4*descriptor` 選 #0..18 後 `0x4deda`。`DecodeNativeRangeOverlayBank`／`BlitNativeRangeOverlay` 以真實 resource regression 固定 456 stride、0x8088、camera clip 和 preflight。mode6 不呼 `0x126f7`，而是算 `4*(x+y*raw53ac1)+7` 後清 `[0x53a51]` 指向資料的一個 byte；drawable API 明確拒絕 mode6。native buffer/grid lifetime已接；mode6 production caller仍待。
- [x] **RE-RANGE-MODE-ZERO-NOOP**：官方 IDA 9.4 重讀 `0x122dc`
  證實 switch default不draw；Capstone另固定bootstrap `0x10483`
  先寫`[0x51a83]=0`再呼`0x11cac(1)`。故raw mode0是transient opening frame，
  不是persistent steady state。`BlitNativeRangeOverlay`現以byte-for-byte
  regression接受0為exact no-op；pure modes1..5 placement API仍拒絕0，
  mode6仍只走獨立field mutation。
- [x] **`0x122dc` mode6 raw-field／scheduler closure**：`0x108f0..0x10932` 載 FDFIELD composition 至 `0x53a51`、讀 signed `u16 width/height`；`0x4dbfc` 由 header 後的 4-byte cells 逐筆將 byte+3 初始化為`0xff`，再對 byte+2 mask `0x1f`。所以 mode6 的 `4*(x+y*width)+7` 正是 selected cell byte+3（event-high／raw blit-mode byte），不是 overlay sprite或抽象grid。`ComposeFrame`現按原順序在terrain後清selected cell、再畫unit/foreground，且只有full-frame成功才commit caller field；bounds/HUD failure regression保證atomic。這只閉合compositor primitive，尚未把未證實的global-selector6 owner接進production。
- [x] **`[0x51a83]` full-domain correction**：合法 IDA 9.4 完整 data xrefs 已保存於 [`fd2_51a83_xrefs.txt`](../data/ida/fd2_51a83_xrefs.txt)。撤回把它限制為`0..5`或稱「戰鬥訊息索引」：`0x15140/0x153b1/0x1bd14/0x1d188` 都是 zero-extended record byte `+2`；原 spell table 的 range 5/7/9 會產生7/9/11。`0x122dc`對>6不畫圖，但`0x115b6`仍以selector-1進`0x14742` target legality。battle raw state現保留writer可達的`0..0x101`；campaign JSON只允許有直接入口證據的 selector 0／1：ch26_pre 返回 battle_ch27 時為0，CONTINUE／post-bootstrap `0x1060c` 為1，避免靜態資料冒充其他互動writer。
- [x] **RE-INTERACTIVE-SELECTOR-LIFECYCLE**：Docker Capstone固定setup `0x10483=0→0x11cac(1)→0x105eb:0x11cac(0)→0x1060c=1`，並固定`0x1cff0` target entry寫`record+4+2`、cancel/effect期間暫寫0、exit恢復1。remake campaign/production frame現以selector1和FDOTHER#1 descriptor0呈現原生steady cursor，移除白框 approximation；target modal亦直接呈現完整call-table已閉合的selectors2–5，regression要求每一 selector 實際改變indexed VGA frame。selector6 mutation、7+ no-draw target visual、flash與indexed effect仍維持partial。
- [x] **`0x115b6→0x14742` cursor-confirm closure**：Capstone證實`0x14742`唯一caller為`0x1175f`。code5先拒絕；cell byte+3=`0xff`也在code4前拒絕；非`0xff`的code4接受；codes0..3以`[0x51a83]`（>1才減1）作strict Manhattan `< radius` roster count，count非零才確認。code6維持獨立relocation branch。新增`NativeCursorConfirmationAllowed` fail-closed raw-roster regression。同步撤回target code2=`camp!=1`舊斷言；direct branches是`camp==1`，即只選友軍。
- [x] **steady native indexed map-frame scheduler**：新增 `internal/indexedmap.ComposeFrame`，強制順序 `0x11eee terrain → 0x122dc range → 0x127a9 unit → 0x129ec foreground → HUD callback → 0x11eb0`。**更正舊 320×192 斷言**：direct `0x11d12..0x11d36` 是 width312、height192、dst `A0504`／stride320，即 VGA `(4,4)` 四邊留4px；compositor/regression已照此修正。HUD callback 缺失即拒絕，private work clone 讓任一 layer/HUD 失敗不污染 caller 的 work/VGA。
- [x] **native map HUD panel subpass**：`indexedmap.BlitNativeMapHUDPanel` 直接接 `0x1acf3` 已證實的雙 raw gate、FDOTHER#5 LMI1 #130（69×34）、`stride*157+anchorX`；entry geometry 不符即拒絕且不寫 destination。**撤回**把它當一般 LMI1 cell 的實作：#130/#0x83/#0x84 必走 `ParseLMI1FrameEntry→0x4e63d` four-mode `Frame`，`DecodeNativeMapHUDFrames` 已以真實 FDOTHER regression 固定。它只畫 panel，terrain/unit icon 與 AP/DP/HP digit paths 仍分離，不能把完整 HUD 標成完成。
- [x] **native HUD signed-number selector**：`indexedmap.BlitNativeMapHUDSignedNumber` 固定 `0x1aeb1` 的 raw LMI #0x83（非負 6×7）／#0x84（負值 6×5）選擇、absolute value、`origin+8` digit callback。callback 必填且 failure atomic，不把 sign 留半張；font、table value、AP/DP/HP來源與語意仍未命名。
- [x] **native HUD two-digit renderer**：`0x1aeb1→0x187d6` call-site 固定 glyph base #0x1f、width=2；`%0.5d` 被 patch 成 `%0.2d`，每位以 six-pixel advance 走 `0x16886→0x4e63d` Frame。`BlitNativeMapHUDTwoDigitNumber` 接上完整 #0x1f..#0x28（實檔 #0x20=5×8、其餘6×8）與 sign selector，超過99 fail-closed，不讓 native truncation變成可編輯資料的隱性行為。數值來源／AP/DP/HP語意仍不命名。
- [x] **native HUD terrain-icon subpass**：`0x1acf3` 在 panel 後以 `0x12e38` local word0（masked terrain descriptor）直接選 selected FDSHAP bank descriptor，`0x4deda` raw blit 到 panel row-5 的 `stride*5+6`（與 optional unit icon 同列）。`BlitNativeMapHUDTerrainIcon` 固定此 10-bit selector／anchor並拒絕 bank 外資料；不以 PNG preview 或 terrain 名稱代替原始 source。
- [x] **native HUD unit-icon subpass**：`0x12c0d` 成功後，`0x1acf3` 以 unit+2 selector-cache slot、raw global state（3→1 alias）選 cached FDICON 12-frame block，raw blit 至 panel `stride*5+6`。`BlitNativeMapHUDUnitIcon` 有 cache/state/bounds regression；slot 不命名為角色或 portrait。
- [x] **native HUD terrain AP/DP subpass**：resolver raw control byte+1 經 `0x51a12/0x51a2a` 固定 0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5)。`NativeMapHUDTerrainAPDP`／`BlitNativeMapHUDTerrainAPDP` 用 exact AP/DP layout origin、sign和兩位 glyph renderer，invalid code／render失敗 atomic；不替 control byte 命名。
- [x] **native HUD persistent anchor branch**：Docker Capstone 實讀 `0x1ad2a..0x1ad5f`：raw `0x53abd>5 && 0x53ab9<3` 才寫 anchor `0xf2`，`0x53abd>5 && 0x53ab9>9` 才寫 `1`，其餘座標保留既有 `0x51a0c`；`indexedmap.AdvanceNativeMapHUDAnchor` 覆蓋兩臨界值與 retention，不把 globals 命名為未證實 UI 語意。
- [x] **native HUD HP subpass**：`0x1ae8e→0x1875d→0x187d6` 的 incoming stack 已逐項驗：unit unsigned `+0x40/+0x42` 傳 current/max 和 mode3；current==max 選 glyph base #0x1f，否則 #0x2a，畫 current 三位、每位 advance6；current>999 改畫 base+10。真實 #5 #0x29/#0x34 均18×8，兩 digit bank 僅 digit1=5×8。`BlitNativeMapHUDHP` 覆蓋 equal/unequal／overflow和 invalid-resource atomic，未將 unequal 命名成 damage。
- [x] **native full HUD assembly**：Docker Capstone `0x1ad72..0x1aea9` 確認順序 panel→terrain→AP→DP→optional icon→optional HP；`BlitNativeMapHUD` 以 `NativeMapHUDInput` 將所有已證實 subpass atomic 組裝。`OptionalUnit=nil` 嚴格代表 `0x12c0d` 或後續 raw unit-byte gate 的 skip，helper 不猜測 gate 角色語意；display gates 關閉時 no-op 且不要求資源。
- [x] **native HUD optional-unit gate**：`0x1ae2a..0x1ae47` 已直接固定：raw `unit+7==0x79` skip；否則僅 raw `unit+0x1f==0x0a && unit+6==1` skip。`NativeMapHUDOptionalUnitEligible` 覆蓋兩 skip 與兩放行，供 caller 正確產生 `OptionalUnit=nil`；三 byte 不命名。
- [x] **strict native indexed-frame entrypoint**：`ComposeNativeFrame` 把 `FrameInput` 與 `NativeMapHUDInput`/frame/terrain/unit/cache 綁為單一 `NativeFrameInput`，直接以 `BlitNativeMapHUD` 填滿 `0x11cac` 的 HUD slot，不再把完整 native frame 交給任意 callback。regression 驗 HUD bytes 進 work buffer 及 `0x11eb0` viewport copy；PNG/Ebiten presentation 仍是下一個獨立 asset/palette bridge。
- [x] **FDSHAP archive sprite-bank bridge**：`fdother.DecodeSpriteBankResource` 以 LLLLLL `ReadResource→fdicon.Parse` 解 FDSHAP even image resource，不混入相鄰 terrain-control resource；player-provided FDSHAP#0 regression 固定 288 個24×24 four-mode frames。這提供 native HUD terrain icon／indexed compositor 的真實 archive input，但 map↔resource pairing仍由上層明示。
- [x] **FDSHAP map resource pairing**：`DecodeMapTerrainResources(mapIndex)` 只讀已證 map N→image #`2N`、control #`2N+1`，並驗 bank frame count 不超 control-record count；FDSHAP map0 真實 regression=288 frame/1200 control bytes。明確拒絕從 tile count/cost 猜 map 資源。
- [x] **exported map-path binding**：`MapIndexFromAssetPath` 僅接受 legacy `assets`=map0 或 basename 精確 `mapN`，拒絕 suffix/負數/任意目錄；runtime 將用此 explicit N 餵 FDSHAP pair loader，不以檔名近似猜配。
- [x] **production native-map asset gate**：`Game.loadMap` 載入 HUD FDOTHER frames、FDOTHER #1 range bank、明示 FDSHAP pair、FDICON.B24、FDOTHER #3 LUTs與VGA palette為all-or-nothing `nativeMapAssets`；任一缺失/解碼失敗保持既有PNG renderer。bundle regression明確拒絕缺range bank；indexed-to-Ebiten presentation仍待camera/global-state bridge。
- [x] **indexed→Ebiten native HUD strict bridge**：撤回舊hardcode
  `gateA=true/gateB=true/anchor=1` partial presentation；`0x10010`證實
  gateA由native save plaintext `0x30d2`恢復。`drawNativeMapHUD`現只在
  `NativeMapHUDRuntimeState`、selector/cache cycle、unit+7/race/+6、
  raw maxHP全部有provenance時，一次畫panel/terrain/AP/DP +
  optional unit/HP；缺任一在draw前回legacy。`battle_ch01`、`battle_ch26`、
  `battle_ch27` 已用可編輯 view 與 `native_map_hud_inherited` 接上
  save-persistent gate A、process-persistent anchor 及 controller gate B=1；
  其他章仍須逐章 view 來源，且角色 raw record 不完整時照舊失敗即關閉，
  故此項是 strict bridge 而非全戰役畫面已 native。
- [x] **HUD unit-gate constructor provenance**：Docker Capstone `0x10d7f..0x10efc` 固定 runtime `+6=FDFIELD b0`、`+7/+8=FDFIELD b1`；IDA Pro 9.4 進一步固定 `0x10ed6` 將同一 b1 傳給 `0x11019`，回傳 slot 寫入 `unit+2`。editable `map_selector_key`／`battle_fig` 現以 b1 對齊；`+0x1f` 改由 portrait/resource branch 寫入，不能拿 portrait/class 直接代替。缺少該 resource byte 時 optional icon/HP 繼續 fail-closed。

- [x] **FDICON indexed asset primitive**：`internal/fdicon` 現直接 decode `FDICON.B24` header/offset table/24×24 four-mode RLE，保留透明與 dither spans；`Sprite.BlitAt` 是 raw `0x4deda`，`BlitPaletteBand` 是 `0x4de56` 的 `(index&7)+0x18`。**撤回 256-byte LUT 對應說法**（那是其他 renderer path）；fixture 與 player-provided 原始 1680-sprite regression 通過；仍未替代 roster/frame/timing/layer adapter。

- [x] **FDICON native selector primitive**：`Bank.SpriteFor(key,pose,cycle)` 嚴格表達已解析 B24 raw key 的 `key×12 + pose×3 + cycle` lookup（pose 0..3、cycle 0..2）並 regression；`0x127e0` 則先取 runtime `unit+2` cache slot 選對應 12-pointer block。它與 `0x287b5..0x2884c` 的 battle `unit+7 × 3` FIGANI selector 是不同 raw field；現有 exported visual id 的相等只在已驗證 roster 記錄成立，不能當 ABI alias。`NativeFrameIndex` 依 +4 movement offset 選 `0x3C0B/0x3C07`，將 global cycle 3 正規化為 1，`+0x26` 則強制 0；撤回「runtime +4 frame」說法，故沒有把它隱式接入 GUI。
  - [~] battle selector bridge：`battle_fig`→`Unit.BattleFig`→全螢幕 `newAtkAnim` 已可承載 split ABI，loader regression 固定它可與 legacy map `fig` 不同；constructor `0x10d7f..0x10efc` 已閉合 FDFIELD `b1→unit+7`，正式 exporter 已寫入該欄、舊 JSON 才 fallback。`fig` 不宣稱原版 field。
  - [~] map selector provenance audit：`0x10c50→0x11019` 是 global raw-key FDICON cache path；IDA Pro 9.4 已釘 FDFIELD **b1** 傳入 `0x11019`，回傳 slot 寫入 `unit+2`，而 b0 另寫 native `+6`。`0x11019` 只比對全域 key table，僅新 key 使用 caller archive pointer 建 block；player `0x10a25` 與 scripted `0x10b69` 都開啟 `FDICON.B24`。parser/exporter 現輸出 raw `map_selector_key=b1`；33 份版本化 map assets 已以 `--rewrite-map-selector-key` 修正 1886 筆 stale b0 值，Docker `--check` 與 1887/1887 逐筆比對通過。舊 scripted `native_identity` 已移除，避免把 FDFIELD `b1→runtime +8` 錯稱角色身分，且不覆寫其他人工校正數值。Scenario 現在以 party-first／group-order batch materialize，battle draw 只在整場成功時 slot→key；malformed editable input 會保留 legacy append 並禁用全場 native selector。撤回把角色表/DATO/素材 index 的相等值當成全域 mapping。下一步是以重新抓圖驗證單位圖像／前景／HUD，仍不得把目前 PNG/Ebiten selector adapter 寫成完整原版 renderer。
    - [x] player-party source split：`0x1088d→0x10a77` 先 copy persistent `[0x53bf7]` 0x50-byte record，再用 copied `+7` 作 `0x11019` key，回傳 slot 寫 `unit+2`。它不是 FDFIELD b1 路徑；slot allocation 順序必須保留這條 roster loop 的順序。
      Official IDA 9.4 address-only xref report再確認 `0x10a77` 屬於 `sub_1088d`，而 `sub_1088d` 的 callers 是 `0x205ff`、`0x25870`、`0x2c437`；不得將 selector initialization 當成只有一般 battle setup 才會做的步驟。
      `JOIN` constructor `0x112a5(join_id)` 直接寫 persistent `+7=join_id` 且 `+8=join_id`；`0x33499` 已閉合 `+8` character-ID lookup。因此 fresh player 的 map raw key=character ID，但只限這個 writer；不得回推 FDFIELD/NPC/general `fig` identity。另 `0x314a7..0x3157a` class-change flow 對 live roster `+7` 寫 UI-selected raw target，故 equality 不是 immutable；`0x11506` 的 full 0x50 runtime→persistent copy 會在任何 `sync_party` caller 保存它，唯 class-change 是否立即進這條 flow 待追。
    `fdicon.NativeSelectorCache` 已以 first-seen key→slot regression 表達 cache 部分；resource/key decoder 尚未接入 runtime。
    `KeyForSlot`／`SpriteForNativeSlot` 已閉合 slot→raw B24 key→`key×12+pose×3+cycle` 的 pointer-block lookup；runtime materialization 仍待。
    - [x] explicit raw-key materializer：`map_selector_key` 與 `battle.MaterializeNativeMapSelectorSlots` 現可按 caller supplied native order 建 first-seen slots；preflight 要求每筆有 0..255 key，missing/invalid 不改 unit/cache，絕不從 `Fig` fallback。`spawn_party` 已先物化玩家隊伍，後續 `AppendGroup` 共用同一 cache，正式 scenario 已保留 party-first→scripted order；不完整 batch 會禁用全場 native selector，story/direct-start/retry 不在此 E1 範圍。
    - [x] player JOIN/class-change split persistence：fresh `PartyMember.Fig` 僅在 verified JOIN initialization 種入 `BattleFig`／raw key；`ApplyClassChange` 依 `0x3157a` 更新後兩者、清舊 slot，保留 stable `Fig`（native `+8` identity），跨關 persistent overlay 完整帶回 split fields。battle/campaign/cmd regression 通過；renderer 仍不由 legacy `Fig` fallback。
    - [x] state atomic construction-order seam：`State.AppendNativeMapSelectorBatch` 持有單一 global raw-key cache，只有整批 explicit key preflight 成功才 materialize+append；regression 固定 party `[9,4]`→scripted `[0,2,0]` slots `[0,1,2,3,2]` 和 failure 不污染 state/cache。33 份 map asset 已有 raw keys；Scenario 已於 `spawn_party`／`AppendGroup` 接上 mixed construction order，缺 key 時保留 legacy append 但整場 native selector 失敗即關閉。
    - [x] persistent overlay boundary：`syncPartyFromBattle`／`applyPersistentStats` 保存 raw `+0x42` 與 `+0x46`，且不由正規化 HP／MP 反推。`MapSelectorSlot` 是每戰由 `0x11019` 重建的 battle-local cache result，不得由 persistent snapshot 覆蓋；回歸以既有 slot 7 證明 overlay 不修改它。
    - [x] **JOIN-112A5-PERSISTENT-RECORD-MATERIALIZATION**：IDA Pro 9.4 主判讀與 Docker Capstone 覆核固定 fresh JOIN 的 default/growth table 公式；`sync_native_join_constructor.py` 現從 32×0x18 defaults、32×0x0B growth 與 `fd2-reference-files.json` 產生雜湊綁定的 `native_join_constructor.json`。Go loader 驗證 EXE identity、row order、file offsets、raw strides；正常 JOIN→LOADCH 與 `scenario join_party` 共用 materializer，建立 identity／raw key、race/class、level/MV、command mask 及 `+0x42/+0x46`。凱麗 id12 初始 `+0x42=151`，event63 fixture 不再手填，也不從 ch27 近似 `hp=90` 反推。另修正友軍尚未轉為 Own 時被錯誤 camp gate 阻止建 persistent record 的舊缺陷。
    - [x] **JOIN-1145A-EQUIPMENT-RECOMPUTATION**：撤回「`sub_1145A` 尚未閉合」的舊斷言；raw helper、八格 `0x40` gate、item row `+1/+5/+3/+7`、signed/wrapping destinations 早已具直接證據。本輪把它接入正常 `JOIN 0x112A5` materializer：局部 0x50-byte transaction 先寫 raw inventory、base AP／DP／DX、HP／MP，再原子重算 typed AP／DP／HIT／EV；缺 item row 失敗即關閉。凱麗 fresh JOIN regression 固定 base `80/69/10`、items `0x3E/0xAC` 與 derived `100/79/110/15`。同輪修正 native save restore 遺漏 `+0x42/+0x46` provenance，以及商店 current stats 把 signed words 誤讀成 unsigned。這不宣稱保留原版 record 的未觸及 bytes，也不是 ch27 一般玩家時點 E2。
- [x] **FDICON native placement primitive**：`NativePlacementOffset` 逐指令重現 `0x127e0` 的 456-byte buffer stride、`0x75d8` origin、24-pixel tile 與 `unit+4 × {+0x720,-4,-0x720,+4}` direction offset；`+0x26` 才加入 native 0/1 pixel shift。它回傳 framebuffer byte offset，不把未證實的 framebuffer origin/layer/UI 自動轉成 remake screen coordinate。
- [~] **native foreground-terrain occlusion layer**：Docker trace 已把 `0x129ec` 定為 unit-sprite 後的前景補畫，但修正「每個可見 unit」的簡化說法：它先跳過 `0x1f183(slot)` true 的 raw gate（`unit+7==0x1c` 放行；其他 `unit+7` 的 class `0x13` 或 race `4/5` 跳過），再跳過 `0x3453e` inactive slot。**撤回**剛才將 `unit+7` 稱為 group 的錯誤：map sprite group 是 `unit+2`。`fdicon.NativeForegroundRedrawEligible` 以 regression 固定兩 gate，`NativeForegroundRedrawCells` 保留 eligible slot 的精確 `(x,y)`、`(x,y-1)`、移動 pose-neighbour 順序。`BlitNativeForegroundLayer` 現以 raw roster inputs 接上 steady `0x129ec→0x12ac6`：camera interval、bit7／bit8 descriptor、`index+1`、`0x8088` offset、raw/LUT-transparent branch 全部 byte-level regression；只在 supplied map 外的座標 fail-closed skip，不讀 unchecked native memory。Official IDA 9.4 再證明 `0x1366a` 的 scripted-step redraw 也做 `0x11eee` terrain→per-slot `0x127e0`→`0x129ec`，並在 `0x129ec` 後才進 `0x11eb0`／present；故不可只把 occlusion 掛在 steady `0x127a9`。range/HUD/VGA/Ebiten adapter 尚待。
- [~] **FDSHAP raw-transparency / LUT branch boundary**：four-mode decoder 保留 opacity mask，`export_engine_assets.py` 以它輸出 raw `0x4deda` preview 的 RGBA tileset（map0 alpha `(0,255)`，opaque palette index0 不被猜透明）。**撤回**「mode3 一律等價 alpha」：`0x11eee` 的 entry `+3!=0xff` 走 `0x4dcc6`，其 mode3 對既有 destination 做 LUT remap，不是 skip。exporter 已保留 event high byte `native_tile_blit_modes`，供未來 indexed adapter；完整 palette/LUT compositor 與 `0x129ec` schedule 仍 fail-closed。
- [x] **native terrain frame selector**：`0x11eee` 對 visible FDFIELD cell 取 10-bit tile ID、讀 FDSHAP terrain-control byte；priority 為 bit8→`+2*flip(0x53a40)`，否則 bit10→`+trunc(0x3c0b/2)`，否則 bit4→`+flip`，其餘 base tile，隨後才選 `0x4deda/0x4dcc6`。`fdicon.NativeTerrainFrameIndex` strict regression 覆蓋 priority/negative truncation/bounds。這是 raw animation ABI，不替 bit 命名；GUI frame scheduler 尚未接。
- [x] **native `0x4dcc6` LUT primitive**：`fdicon.Sprite.BlitLUT` 精確保留 source write→`lut[source]`、mode3→`lut[existing destination]`、mode1 dither holes 不改寫三種行為，short LUT fail-closed 並 regression。它不選 LUT／不管 map camera/layer，避免把原始 pure blitter 誤接成完整 terrain renderer。
- [x] **native single-cell terrain compositor**：`Bank.BlitNativeTerrainCell` 組合 exact frame selector 與 FDFIELD `entry+3==0xff` raw／否則 LUT branch，regression 覆蓋兩支及 mode3 destination remap。camera-visible loop、LUT phase、foreground `0x129ec` 不在此 pure adapter 範圍。
- [x] **native visible terrain pass**：`Bank.BlitNativeTerrainRegion` 以 raw FDFIELD cell、FDSHAP 4-byte control records、map origin／explicit LUT 做 `0x11eee` row-major visible region，bounds fail-closed、regression 覆蓋 raw/LUT cell order。正常 `0x11cac` ABI 已釘為 `(buffer+0x8088,456,13,8,camX,camY)`，其後 range→unit→foreground passes 仍分離。
- [x] **native indexed viewport copy**：official IDA/Capstone 關閉 `0x11eb0` 為逐列 `memmove`；`0x11cac` 明確以 source `buffer+0x8088`／stride456、width312、height192 複製到 VGA `0xA0504`／stride320。regression 覆蓋 row stride、source/destination offset、4px border 與 fail-closed bounds；ch01 已接 Ebiten production presentation。
- [~] **native terrain/unit map HUD (`0x1acf3`)**：它在 `0x11cac` 的 terrain/range/unit+foreground 後、viewport copy 前執行，且須 raw gates `0x51aab`、`0x51aac` 都非零。`BlitNativeMapHUD→ComposeNativeFrame→drawNativeMapFrame` 已把 panel、terrain、AP/DP、optional FDICON unit與HP依原順序接入ch01 production frame；#130、hex #0x83/#0x84、digit/overflow banks、persistent anchor與raw cycle均有resource/runtime regression。FDOTHER#5 full-screen #22只在native admission失敗時作playable fallback，不再代表ch01現況。ch26 event61 所需 view/HUD 已另達 E1；此項仍為partial，因其餘 ch02+ 缺逐章view/gates/anchor來源、`0x12c0d` exact raw lookup predicate/order尚未閉合，且沒有原版DOSBox 320×200 HUD pixel oracle；高階global與resource artwork名稱仍不猜。
  - [x] HUD runtime provenance：data初值anchor=1、gateA/gateB=1；
    load `0x10010`由plaintext `0x30d2`覆寫gateA。anchor只由visible
    cursor row/column `[0x53abd]/[0x53ab9]`兩條branch改0xf2/1；doc14
    舊「框寬高」斷言已刪。`NativeMapHUDRuntimeState`保存raw bytes與
    persistent anchor，未materialize時不畫native HUD。
  - [x] camera/cursor runtime provenance：`0x11b48..0x11cac`四個direct
    helpers證實absolute cursor、camera及visible cursor identity。
    上/左在visible `<2`且camera非零時捲動；下在visible `>5`且未達
    `height-8`時捲動；右在visible `>10`且未達`width-13`時捲動。
    `NativeMapViewState`與keyboard/touch bridge保存原版13×8 window，
    並拒絕broken identity或diagonal raw move。
  - [x] inherited HUD state vertical slice：`battle_ch01` campaign 節點保存
    真實 FD2.SAV 的 camera `(1,13)`、cursor `(8,17)`、visible `(7,4)`；
    ch26／ch27 各自使用 pre-handler 閉合的 view。三者都由
    `native_map_hud_inherited` 取得 save-persistent gate A、process-persistent
    anchor 與 controller gate B=1；loader 要求 HUD 必須有 view，且原子拒絕
    不合法 raw bounds。固定 `(1,1,1)` 只保留給明確的存檔 fixture，不再冒充
    所有戰鬥入口初值。
  - [x] codec boundary：#130／hex #0x83／#0x84 不走 `ParseLMI1` 的 `0x4e916` cell codec；native `0x1aeb1` 有 literal `mov ebx,0x83/0x84`，明確走 four-mode `0x4e63d`。`ParseLMI1FrameEntry`／`DecodeLMI1FrameResource` regression 驗證 geometry 69×34、6×7、6×5 及 transparent decode。撤回剛才將 hex immediate 誤改成 decimal #83/#84（44×12／45×12）的錯誤斷言。
  - [x] optional unit selector：`0x1ae4d` 以 raw `unit+2*12 + state` 選 FDICON，state=3 alias 1，並在 panel `stride*5+6` raw blit；HP `+0x40/+0x42` 經 `0x1875d` 畫至 `stride*21+9`（mode3）。`NativeMapHUDUnitFrameIndex` regression 保留 selector，不替 state 命名。
  - [x] strict compositor layout／production bridge：`NativeMapHUDLayoutFor(anchor,456)` 固定 frame／terrain／AP／DP／unit／HP 的六個 byte destinations，拒絕非 native stride 與 69-pixel frame 出 320-pixel viewport 的 anchor；`BlitNativeMapHUD`已由`ComposeNativeFrame`接入Ebiten production full frame。此項不證明其他章節的raw runtime來源或DOSBox visual parity。
  - [x] HUD viewport-base／原版位置 oracle：Docker Capstone固定
    `0x11cfa`將`[0x53a49]+0x8088`與stride456傳入`0x1acf3`；舊adapter
    錯把HUD offsets套在allocation base，測試也固定了錯位。`ComposeNativeFrame`
    現改傳`work[0x8088:]`，HUD panel由同一source經`0x11eb0`落到VGA
    `(anchor+4,161)`，並保留full-frame atomic failure。
    `extract_fd2_video_frame.sh`先裁錄影的1408×880 centered viewport再回復
    320×200，撤回直接縮整張1440×1080影片的失真oracle。原版434.5秒與
    remake現對齊camera `(1,13)`、absolute/visible cursor `(8,15)/(7,2)`、
    tree terrain與`A -05/D +10`；screenshot hook也改走native cursor/camera/
    HUD anchor state machine。roster/event仍不同，完整pixel diff仍待。
  - [x] pre-handler→battle runtime roster handoff：原版`ch00_pre`
    `LOADCH(map0)`後由ACT0將party slots0..3從部署Y
    （scenario UI順序`0,4,9,30`為`[20,22,21,23]`）先依JOIN順序
    `0,9,4,30`重排成`[20,21,22,23]`，再上移六格成`[14,15,16,17]`，
    並於同一runtime array
    append initial groups；舊`resetBattle`卻清掉全部`storyActors`並重播
    `on_battle_start`，是原版錄影與remake可見roster不一致的根因。
    現僅在handler roster／party-scenario paths與battle node完全相等時
    carry已ACT/SPAWN的array、重建native selector slots、保留pending roster
    並consume已完成opening；direct start／retry／mismatch仍走部署重建。
    regression同時鎖定carry與direct兩條座標。新增完整runtime regression
    實際compile並跑完`ch00_pre`至`battle_ch01`：frontier精確為12
    （party4 + group1四筆 + group2四筆），party座標為
    `0:(7,14),9:(10,15),4:(8,16),30:(11,17)`；slot9的raw `+5`
    whole-byte writer結果為1，pending groups 3..7仍保留。
- [x] **native terrain renderer export bridge**：`export_engine_assets.py` 在帶 FDSHAP terrain resource 時輸出完整 `native_terrain_control` raw bytes 加既有 per-cell `native_tile_blit_modes`。map0 實測為 576 cell modes、1200 control bytes；因此 region adapter 不必把 normalized `cost` 當 native renderer input。
- [x] **native terrain renderer runtime bridge**：`battle.Load` 以 serialized `native_tile_blit_modes` 驗證 exact map provenance，但依`0x4dbfc`將 live `State.NativeTileBlitModes`全填`0xff`；`native_terrain_control`維持原始資料。dimensions/cell count/control alignment/tile bounds任一失敗即fail-closed。舊版把archive zeroes直接當live renderer state、造成整張圖走LUT的斷言已撤回。
- [x] **FDOTHER#3 LUT bank loader**：`fdother.ParseLUTBank`／`DecodeLUTResource` 嚴格解析 LMI1 directory 的 23×256-byte remap tables（非 UI LMI cell），fixture 與 player-provided archive regression 通過。現可把確證 LUT 交給 `BlitLUT`；map selector、palette timing、renderer layer 仍不猜接。
- [x] **native terrain LUT phase selector**：EXE `0x51A97` 的 20 bytes 直接讀得 `0..10..1` 往返序列；`NativeTerrainLUTIndex(0..19)` 並 regression。`0x11eee` 預設取此 phase 對 FDOTHER#3 LUT；explicit override state仍只保留 raw，不命名效果。
- [~] **indexed ending compositor core**：`internal/ending.IndexedCompositor` 現提供原版尺寸的 VGA/offscreen/work buffers、透明 `fdother` in-place blit、64000B copy、baseline-derived DAC、ANI、frame12..108、40/200-pass schedule與Ebiten獨立preview。`internal/ending.MontageCycle` 現可在原資源上執行 `0x2c548` 的九段 fade、20 次 secondary、primary FIGANI、DATO/FDTXT portrait 與 64 段 palette 收尾；`MontageTailAssets` 負責 #57/#58/#60/#59 來源驗證，`MontageTailPlayer` 已按實際 header-byte1-zero 資源執行 raw `+6` inner present、`+7 bit0` 層序、base scheduler、`+4` 位移／palette33及兩次交叉配對，完成後保持 #59。`MontageTail.Plan` 保存 record0／record1 的 `+6/+7` 與 `[0x540ff]` raw schedule；正式最終節點以 persistent raw roster／shared native RNG 接線，且只在 portrait loop 消費新輸入的 raw-change 效果。證據見 [`fd2_ch29_montage_ida.txt`](../data/ida/fd2_ch29_montage_ida.txt) 與 [`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt)。尚未解除的是呼叫時 records/globals 連續性、3% RNG重播、音訊／輸入與一般玩家 E2，不再把整個 `0x28a6c` renderer 列為未知。
- [x] **ending compositor asset preflight**：正確圖源是 `FDOTHER_054.bin`（263655B、111-frame table），不是 `FDOTHER_036.bin`（408×138 的無關資源）；ANI #2 已完整分離。#54已輸出完整111格indexed frame＋binary mask＋placement metadata，strict loader與固定原檔逐格一致；正式`ending_preview`及#5對話格不再讀取`FDOTHER.DAT`，缺原版FDOTHER／FDTXT／DATO的Xvfb測試通過。`native_2c548.json`描述的FDOTHER#56、TAI#3、FIGANI/DATO party montage亦由分離素材載入；缺任一必要資料即失敗即關閉。精確輸入與raw handoff仍是獨立E2 gate，不重開本素材切片。
- [~] **ending `#0x36` frame decoder contract**：`0x2935b` 以 `base+8+frame*4` offset table取 descriptor；`+0/+2` 是內嵌目的地 dx/dy，`+9/+11` 是 real w/h，payload 自 `+9` 以 transparent `-1` RLE blit。玩家素材 regression 現對 #054 全111幀逐一做 320×200 in-place decode。`0x2bce5` 的 frame0、frame9、frame12..108、兩段 frame-pair composite與palette/delay loop均已有runtime；`0x2c39b`已定案為DATO portrait ID＋current FDTXT string index並接兩段preview dialogue。`0x2c548` 已有原資源 party-cycle executor，但不代表終局 campaign／輸入路徑完成。
- [~] **editable ending prefix timeline（歷史載入器狀態）**：新增 `assets/endings/native_2bce5.json`，把已證實的 #054 blit、copy、delay、palette ramps、兩段 native composite loops 存成可編輯 IR。`0x2c39b` 第二 arg 已依 `0x15f84` direct ABI 定案為 current-FDTXT string index：final route idx2..7 → `ch30.json` scene1 lines0..13；chapter26 bad ending idx17..20 → `ch27.json` appendix scene lines1..4（原始 FDTXT_027 逐 string decode 實證）。第一 arg已依 `0x1956b → 0x111ba(0x51a70)` 與 doc14 定案為 `DATO.DAT` portrait ID，timeline 改用 `portrait_id`。`internal/ending` 的 JSON loader 仍驗證歷史 raw status `recovered_prefix_only_fail_closed`；此字串是 timeline 資產狀態，不是現行 campaign mode。現行正式終局接線由上方 2026-08-21 勘誤與有效佇列為準。
- [x] **天空之鑰缺失對話分支**：新增 `ch27.json` 分支 scene（FDTXT_027 idx13–16 共17句）並接 `inventory_gate_ch27_sky_key → story_ch27_post_sky_key_missing → ending_ch27_no_sky_key`。**2026-08-25勘誤：**後者已接chapter26來源約束`0x2BCE5`前綴與FDTXT_027 index17..20兩個原版文字閘門，故「視覺效果仍待direct RE」舊句已失效；剩餘的是呼叫當下動態連續性、精確輸入與一般玩家E2。
- [x] **戰後 town/shop/preparation 外部交叉盤點（2026-07-20）**：subagent 查得公開攻略逐章列出羅德鎮、塞拉村、普里茲港等戰間商店／教會／整備，並有「第2章戰後獎勵」與「第6章戰後貝克威加入」等 persistent event 證據；只作流程旁證，不取代 EXE branch 證據。後續保持 battle→postbattle→town/shop/preparation→next battle 可編輯節點，禁止把 postbattle 直接接下一場戰鬥當完成。
- [x] **戰後 town/rest 反例盤點（2026-07-20）**：GameFAQs 明載第14章途中小鎮休息，且第22章至第26章前沒有 rest/buy/sell；因此 campaign 需保留 battle→town/rest 的可編輯節點，也要允許 ch23–25 連戰區間不插 town/shop。攻略只作外部旁證，仍須以 EXE/資產驗證觸發時機。

## 對話框 / 過場打磨(2026-07-05,使用者實玩逐項校正)
- [x] **對話框文字不覆蓋頭像**:上框(頭像在右)文字右緣止於頭像左緣前(commit 57c0e30)→ doc09
- [x] **對話框上下移入畫面**:下框上移(底邊可見)、上框下移(頂邊可見)、頭像置中框內(dc5ebb1)→ doc09
- [x] **框內底色=頭像底色漸層**:框內疊 40,69,138→56,85,154 漸層消接縫色差(dc5ebb1)→ doc09
- [x] **長對白分頁不截斷**:>3 行分頁,Enter 先翻頁翻完才換句;dlgWrap/dlgPageCount/dlgAdvance + dlgPage;
      Go 測試 dlg_test.go 驗全文保全(b81268d)→ doc09
- [x] **進場走位面向修正**:走完面向 actor 目標 dir(亞雷斯走到索爾旁面向他),storyWalkJob.finalDir(aaf5020)→ doc47 §11
- [x] **對話分頁捲動動畫**(2026-07-20)：原版確認有文字往上捲；remake 已實作 10 幀上捲（舊頁上移、新頁由底部進入、框內 clip），Enter 於動畫期間不跳頁，並有 dlg regression。GUI 實機截圖待補圖形依賴容器。
      目前翻頁是「瞬間切換」;要改成**平滑往上捲動**——按 Enter 翻頁時,當前內容往上捲出、下一頁從底部捲入。
      **使用者明示:不用依賴原版機制,自己寫平滑捲動即可**(原版有此效果,但捲動速度/幀數自訂,非 RE 值)。
      實作方向:翻頁時啟一個 `dlgScrollT` 計時器(數幀),繪製時把文字整體 y 偏移從 0 平滑插到 -行高×3、
      同時畫「當前頁下移出 + 下一頁自底部進」,期間 clip 在框內矩形;捲完才定位到新頁。
      動 `cmd/fd2/main.go` 對話繪製區 + `dlgAdvance`(翻頁時觸發捲動而非瞬間 dlgPage++)。
- [ ] **⬜ 自動結束回合（可選改善，不是 blocker）**：正式 battle 已可由空游標
      原資源面板的 Down→END 換回合，Tab 只保留為重製端快速鍵；是否在全員完成後
      自動換邊仍未由原版證據要求。native end-turn 的完整 team predicate／AI completion timing尚未閉合。
      native end-turn 的完整 caller／team predicate／AI completion timing 尚未閉合。`+5 bit7` 只能作 raw
      set/test mutation，不在此 work item 命名 acted/turn，也不能直接宣稱「全員完畢→換邊」。需補 native
      state-machine evidence 後，才決定是否自動 endTurn、是否保留 Tab 提前結束。
- [~] **handler 後半段 beats 解碼**（前次分析未形成可驗證提交，目前無執行中代理）：庭院/森林段走位/對話/fade 編排,
      供重建 palace_path/forest 節點(Ares 進場對話框位置、逐段走位轉向、索爾練劍、領頭跟隨、fade 換場)

## 完成定義(反組譯研究)
全部資產格式可解(解包+解壓+轉現代格式)、核心數值表全 dump 並驗證、
主要遊戲規則演算法(戰鬥/移動/升級/AI)有反組譯依據、地圖可渲染、文本可讀可改。

## 2026-07-25 SDD gate（使用者要求先重審反組譯與 UI）

- [x] **可重現 UI/core regression container**：`fd2-go-test-local` 在 Docker build 時取得 Go modules、在 runtime 使用 `--network=none`；已實跑 `go test ./cmd/fd2 ./internal/... -count=1` exit 0。image 內含 Ebiten 所需 ALSA/X11/GL headers；這只驗 source build/test，並非原版 UI 畫面對照。
- [x] **UI-01 original title screenshot oracle**：新增隔離 `fd2-dosbox-screenshot-local` runner（SVGA/Xvfb/xdotool，原始 FLAME2 不掛載、只用 `/tmp` sandbox），連續 Escape 跳過 opening 後得到 320×200 `docs/figures/title-original-dosbox.png`。畫面證實 START／LOAD／CONTINUE 及初始 cursor；title input/save semantics 仍未關閉。
- [~] **UI-12 LOAD selector contract**：原版 DOSBox screenshot 與合法
  IDA 9.4 已固定四槽、slots `0..3`、↑↓ bounded（不 wrap）、
  Enter/Space confirm、Esc cancel；IDA 並固定 FDOTHER #13 entry16、
  FDTXT 索引、row/座標與 selected/normal 色碼。production 空槽已與
  DOSBox 全幀 RGB 相同；修改存檔 chapter1 有效槽排版亦全幀相同。
  native save boundary=`0x59cb`、
  record=`+0x312b+i*0xa28`（metadata=`0x28`、roster=`0xa00`）及
  rolling-XOR/checksum已有 adapter；JSON 仍是自有格式，尚未實作
  native 有效槽 restore／完整 roster 相容。
- [x] **UI-05 ch01 dialogue screenshot oracle**：START 分支得到 320×200 `docs/figures/ch01-dialogue-original-dosbox.png`，鎖住一種 lower/left DATO portrait、藍框、兩行文字與 page indicator；upper/right/control code/pagination 尚未由這張圖宣稱完成。
- [x] **UI-04 native command-grid remake oracle**：Docker/Xvfb 以 player-provided FDOTHER.DAT、ch01 materialized 悠妮 `initial_command_mask=[1,0,0,0]` 捕捉 [`native-command-grid-remake.png`](../figures/native-command-grid-remake.png)。畫面確證 command0 label「火炎術」與 selected-unit HUD 同時存在，故 raw mask→grid cell `(18,103)`→editable label→palette/font renderer 已接通；這是 remake runtime smoke，**不是**原版 DOSBox visual diff、full command gate 或 effect/UI 完成證明。
- [~] **UI-03 原版 command grid E2（2026-08-09，2026-08-11 勘誤）**：已取得一份未修改、checksum-valid 的 `FD2.SAV` current-runtime 快照（chapter0、12 筆 runtime records），並在 Docker DOSBox 以正常 `CONTINUE`→Enter 輸入取得原版 command grid E2 錨點；原始輸入、雜湊與兩張 320×200 圖見 [`native-continue-current-runtime-e2.json`](../data/ui-traces/native-continue-current-runtime-e2.json)。這只關閉章節0的「完全沒有一般玩家 command grid oracle」舊說法，不外推到 ch22／ch23、敵方回合、完整 command effect 或重製端 parity；重製 CONTINUE handoff 仍失敗即關閉。

- [x] **UI-03-ORIGINAL-CONTINUE-CURRENT-RUNTIME-E2（2026-08-11）**：未修改原版 `FD2.EXE`／`FD2.SAV` 在 Docker DOSBox 中由開場、標題 `CONTINUE` 到 current-runtime 戰場，再以 Enter 開啟游標單位的 command grid；截圖是實際 320×200 client crop，不是 route patch、debug shortcut 或 remake fixture。這是一般玩家 UI-02／UI-03 的 E2 原版錨點；因快照是 chapter0，不能提升第23／24／25／29戰或正式 CONTINUE handoff。
- [x] **UI-E1-CH01-REMAKE-RUNTIME-CAPTURE（2026-08-11）**：以既有 `story_ch00_handler` BeatRunner 的截圖快進器實際走到 `battle_ch01`，產生 [`native-battle-ch01-remake-e1.png`](../figures/native-battle-ch01-remake-e1.png)；完整執行條件與 PNG 雜湊見 [`native-battle-ch01-remake-e1.json`](../data/ui-traces/native-battle-ch01-remake-e1.json)。這只證明重製端 E1 執行期畫面，不是一般玩家輸入或原版同狀態 E2，不能解除 CONTINUE／戰場逐幀差異。
- [x] **FD2 remake SDD**：新增 `56-fd2-remake-sdd.md`，定義 UI contracts、battle→postbattle→town/shop/church/preparation flow、persistent party/save、native indexed renderer、E0–E3 證據分級與 milestone gates。
- [~] **SDD-1 UI evidence matrix**：以 Ghidra/IDA + Docker Capstone 重審 title/menu/action/target/HUD/dialog input dispatch；矩陣與 Capstone E0 已建立。2026-07-26 使用者合法 IDA Docker image 已實跑 `idat -A`／Hex-Rays，輸出 address-only [`fd2_xrefs.json`](../data/ida/fd2_xrefs.json)；script 已修正 IDA 9.4 移除的 xref-type API。分析 database 與 IDAPython config 均留 `/tmp`，repo 不含 license／binary／database，也絕不用 `kg_patch`。report 只補 call graph，未有資料流或 E2 不解除語意 gate。
- [x] **SDD-1 baseline matrix**：新增 `57-ui-evidence-matrix.md`，以目前 runtime 行號把 UI-01…UI-12 的 partial/missing 與下一個 E0/E1/E2 問題固定下來；這不是原版 verified。
- [x] **UI-03 action caller recheck**：Docker Capstone 重審 `0x18890`，確認它呼叫 `0x18d8c` 取得 action result 並串接 `0x13488` path-walk／`0x13a44` target path；撤回「只是繪圖」類推，`0x18d8c` 本體仍是下一個 RE gate。
- [x] **UI-03 action switch closure**：Docker Capstone 完成 `0x18d8c`：`↑0=攻擊、←1=法術、→2=物品、↓3=待機／格子互動`；同步修正 `main.go` ring mapping 與 13/14/57 文件，撤回舊 screenshot-derived mapping。
- [~] **UI-03 native command ABI**：Docker Capstone 完成 `0x1c269→0x1cff0→0x4e516`：unit `+0x1a..+0x1e` 的
      五個 bitmask 逐 bit 展開為 command ID `0..39`，再索引 `0x619fd + 7*id` 的靜態 record。現行四格 ring
      只是 partial interaction；`0x159fa` 再證實 record `+5 <= unit+0x44`（current MP）的 availability gate。
      `0x10f7f/0x11399` construction 各 copy source 的 4 bytes 到 `+0x1a..+0x1d` 並清 `+0x1e`，
      `0x1d7fb` 可按 commandID/8 OR 回 runtime bit，故 40-bit ABI 初始為 32-bit source、可動態擴充。
      官方 IDA 再釘 confirm dispatch：IDs `0..8/0x18/>=0x1c` 走 `0x2a6bd` generic pipeline，
      `0x09..0x17/0x19..0x1b` 才經 `0x1d6c8` palette flicker→`funcs_1541f[id]`；這不把 command 0
      升格成 legacy spell effect。ID0 的 `0x2a6bd` entry 也只閉合 compositor（`funcs_2ac25[0]=0x26152`），
      尚未定位 HP/status writer。`0x1b6b7→0x1aa1d` 已排除為該 writer：前者只收集符合 post-resolution
      條件的 runtime record 三-byte資料，後者處理其後續訊息／掉落／互動。
      `0x1c75e→0x1c81f` 現已釘 command0 hit/damage：record `u16+0`×target class-ID（unit`+0x20`）multiplier/10，
      `rand()%100 < record+2`，命中後以 90..99.9% base 扣 `unit+0x40` 並 clamp0（`+0x42` 是 HP cap）。
      multiplier table 已與 file `0x51d96` 的職業魔抗 `resist_raw` 對齊（base=`dmg*raw/10`）；完整 target family
  仍待，尚不可直接替換為 legacy magic formula。
      selector `0x1d51d` 已鎖每欄四列的 variable-column grid：↑/↓ linear wrap、←/→ ±4、Enter/Space 重查
      MP gate、Esc cancel；`0x1ceed` 再鎖 x/y formula 與 label index=`0x1b9+commandID`。常駐 table
      已對齊 FDTXT_000，40 個 physical label slots 已由 `tools/export_command_labels.py` 匯出為
      `docs/data/command_labels.json`；label 不等於可達／有 gameplay effect。待 command producer、完整
      renderer/effect stack 後才可重製原版 menu。2026-07-25 再釘 `0x18d8c` wrapper 的 item-side raw
      preconditions：`0x1b83d` 找八格 inventory 中 equipped(bit0x40) 且 ID<0x80 的項目，失敗寫 output+0；
      `0x1b8a6==0`（八格全 empty bit0x80）寫 output+8。它們對應哪個圖示仍未有 callee/E2，禁止猜接 UI。
      2026-07-26 再以完整 `0x18d8c` dataflow 固定四個 disabled words 的順序為 attack/native-command/item/wait，
      `0x173e7/0x177fc` 僅選值 0；`0x1c269==0` 或 `unit+0x27!=0` 都寫 native-command `+4=1`。
      remake 已以 `NativeTransient[5]` gate raw command；撤回任何「`+0x27` 就是 legacy `Sealed`」的斷言。command22
      已知寫入此 duration；status 名稱與其他 writer 仍未知。
      confirm input 同步拒絕任何 nonzero disabled word，避免僅 render 灰 cell 卻仍執行該 action。
      action chooser 本體亦已完成 E0：availability=0 才可用 ↑/←/→/↓ 選 action 0/1/2/3，Enter/Space 確認、
      Esc 取消；四張 indexed asset 自中心做 4-frame 十字 slide，72×72 backup 每幀 restore。尚缺 resource
      anchor 與畫面 oracle，故仍不可將現有文字/ring UI 當成 original renderer。resource provenance 已補：
      `[0x53a89]` = `FDOTHER.DAT#2` 的 78-cell raw offset bank，`0x4e9e4` 直接貼 index pixels（0 preserve）；
      strict decoder/regression 已加入，仍未接 runtime renderer。
- [x] **UI-03 指令格（command grid）失敗即關閉契約（2026-08-09）**：新增純資料 regression，確認空 command ID、selected 越界／負值不 panic、不改變 grid 長度或標記任何 cell；未接未知 command effect、renderer 語意或 DOSBox E2。
- [x] **UI-03 command-record/table identity**：`0x4e516` 的 IDs 0..35 與 EXE spell table 7-byte rows
      byte-for-byte 相同，故 record `+3/+4/+5/+6` 可安全正名為 `dist/range/mp/target`；全 FDFIELD 和
      character-default initial masks 的已見 ID 範圍為 0..30。36..39 僅是 pointer 可達的相鄰 data、label
      為空／系統訊息，未被升格為 spell。
- [x] **UI-03 level-up command producer**：`0x1e292→0x1d79c` 已釘為升級習得 command；growth row 的
      `learn_idx` 經 `0x4e4a2` 查 20×12-byte、最多六組 `(required_level,command_id)` 表，命中即 OR bit
      並顯示 FDTXT_000 #587「學會了！」。已導出 `command_learn.json`，保留 FF/FF sentinel；portrait→growth
      row 是 `0x4e4d1(unit+7)=0x620a1+unit[+7]*11` direct ABI；constructor 已證實 FDFIELD `b1→unit+7`，
      `State.GainExp` 已以 injectable table 接線，
      runtime asset `assets/data/command_learn.json` 已在每個新 battle state bind，不能用 legacy `Spells`
      偽造結果。
- [x] **UI-03 raw command-mask pipeline**：FDFIELD roster `b13..b16` 已由 parser/exporter 保留為
      `initial_command_mask`；battle runtime materialize 為可持久的 5-byte `NativeCommandMask`，並有原版
      order 的 ID expansion／`0x1d7fb` bounded-OR regression。舊 `Spells` 是 normalized approximation，
      不再冒充 raw source；尚未將未知 command effect 接入。
- [x] **UI-03 command availability gate**：新增 `battle.NativeCommandAvailable`／`NativeAvailableCommandIDs`，只接受 raw command bit、完整 0..35 `0x4e516` record 與 `record+5 <= unit.MP`（`0x159fa`）；36..39 physical bits、malformed book 與負 cost fail-closed。未併入 `+0x27` action-direction gate、target geometry 或 command/status 語意。
- [x] **UI-04 target-candidate provenance**：Docker Capstone 延伸 `0x1cff0→0x149f8`，確認 local command record `+3/+4/+6`、`command=0x1e` 傳 selector14、`0x149f8` 沿格步進並輸出符合 selector 的 unit index，另有 `0x17` special geometry 與 `0x2a6bd/0x1d6c8` effect paths；不再把 `0x149f8` 誤稱成傷害／完整 spell priority。
- [~] **UI-03 native target/effect state dispatch**：battle command grid confirm 現以 verified two-stage target contract 開啟原始 target cursor，並只白名單已有 state executor 的 IDs `0,13–16,20–22,24–29,31`；`main.go` 的 cursor highlight/confirm 共用 selected raw command ID 的 record `+3/+4/+6`，target entry materialize `record+4+2` selector，Enter 另以selected cell byte+3／selector／target code／完整raw roster執行 exact cursor-confirm gate，cancel與成功exit恢復selector1。未知 IDs、ID30 special cursor、缺 raw flags/record/resistance 仍 fail-closed。這接通 state/UI boundary，不宣稱 indexed effect renderer、SFX、完整 target visual 或所有 command semantics。
- [~] **UI-04 range geometry**：Docker/Capstone 已直讀 `0x14818`：它以固定 `0x61646` record 0 和原始 `(x,y,mode)`
      呼叫 `0x4e040`，mode 作 seed grid byte 並經 terrain cost gate 建立／更新 target grid；後續再有
      `|x-cx|+|y-cy| < caller radius` 的 marker 層、unit active flag／
      camp selector 輸出 slot；mode>=`0x10` 有另一路十字 clear。這只證實其中的曼哈頓幾何，
      另一 caller `0x14344` 證實同 helper 會以 unit `+0x20`（fallback 0x13）的 record 和 terrain table
      作格點 gate，故 SDD 必須保留 table+terrain+marker，而不能以單一 diamond 實作。
      `0x1cff0` stack-dataflow 亦已固定參數為 `(x,y,out,mode,radius,campSelector)`：special `0x17` 用
      `record+3`/radius 1，一般 path 用 `record+4`/radius 0 並消費既有 marker。尚未將這些 producer
      同武器 `range_min/range_max` table 完整對位；record producer 已鎖為 `0x4e516(id)=0x619fd+7*id`，
      故 `+3/+4/+6` 是 command ABI raw fields，仍不改寫為「所有武器 max inclusive」或 LOS 定論。
- [x] **RE-ATTACK-GEOMETRY-14237**：官方 IDA 9.4 `0x14237→0x14818` 閉合 caller-specific raw geometry：item row `+0x0b/+0x0c` 作 `a5/a4`；`mode<0x10` 時排除 Manhattan `<a5` 的 marker cells，`mode>=0x10` 走 cross 且不套 inner marker。新增 `battle.NativeAttackCandidates` regression；欄位、LOS、item effect 與 UI 仍不命名／不接猜測。
- [x] **RE-NATIVE-TARGET-BYTE5-GATE**：完整 raw roster 時，`NativeCommandTargets`／`NativeAttackCandidates`／`NativeCommandEffectTargets`／command-30 cardinal resolver 已以 raw byte+5 bit0 作唯一 active gate，新增 HP/OnField 相反值 regression；缺 raw 的舊 JSON／測試資料保留 E1 projection，避免猜測性擴大 native binding。
- [x] **RE-ITEM-ROW-CALLER-AUDIT-20260727**：官方 IDA 9.4 交叉檢查 `0x1145a/0x14237/0x1567e/0x1bbdc` 的 `0x4e56c` row consumers；確認 `+1/+3/+5/+7` 是裝備合成輸入，`+0x0b/+0x0c` 只在攻擊 caller 作 geometry inputs，`+0x0d` 另作 effect type dispatch。runtime table 邊界、其餘欄位語意與 normalized row 的一一對應仍未證實，維持 fail-closed。
- [x] **RE-ITEM-WORD-DELTA-TYPE8-10**：Docker Capstone dispatch、
  `0x1145a` base/equipment data flow 與 215-row cross-check 共同閉合
  type8/9/0xa：row `+0xe` 永久增加 base AP/DP/DX
  (`+0x37/+0x39/+0x3e`)，重算後移除來源 slot；IDs198/199/200 amount
  為 9/9/7。`NativeItemWordDeltaRouteForType` 現回傳 typed stat。
  presentation selector、道具名稱與 type17–19 不在此項證據範圍；
  type17–19 已由下方獨立 producer/consumer 證據閉合。
- [~] **UI-03 battle selector input**：Docker/Capstone 重檢 `0x19953`，確認它呼叫 `0x36d98` 讀 ASCII/scancode；Enter/Space/`0xe0`/`0x52` family 走確認回傳、`0x01`/`0x53` family 走取消回傳，`0x4b`/`0x4d` 更新左右選擇狀態。這是 battle selector 的 E0 input ABI，不等於已閉合 action enable/end-turn 或 D8 行軍確認。
- [~] **SDD-2 campaign transition matrix**：已從 `campaign_full.json` 逐一展開 30 個 battle 的 `on_win`，
      明確保留 town/shop/church/preparation/inventory-gate/ending 節點與連戰例外，表格已寫入
      `56-fd2-remake-sdd.md` §5.1（E1 editable graph）。仍待逐列補原版 handler E0／DOSBox E2 證據與 save/reload regression，
      未把 authored graph 當作原版已驗證。
- [x] **RE-CAMPAIGN-LOOP-25DE5**：官方 IDA 9.4 閉合外層戰役順序：phase1→固定 `0x22e5c` interlude；phase2→停止 BGM→`funcs_25e23[chapter]` post-handler→`0x2cad7` gate；僅 gate 回傳 zero 才進 `funcs_25e3a[chapter]` 與下一戰 BGM `0x51e63[chapter]`。table entries／`0x2cad7` 的具體 town/shop label 仍 opaque，但已明確撤回「勝利直接接下一戰」的錯誤模型。
- [x] **RE-TOWN-HUB-GATE-2D093**：官方 IDA 9.4 閉合 postbattle hub 分派：`[0x5412b]` option0→`0x2fc85` hotel/inn、1/3→`0x2e341` shop family、4→`0x3072f` church、2→save/confirm 後 `0x318ad` preparation；各 facility 回 hub 並恢復 track10，下一戰 BGM 仍由外層 `0x25de5` 選取。Docker raw bytes 確認 `byte_526b9[22..24]`、`[27..29]`=`1`（preparation-only），`[0..21]`、`[25..26]`=`0`（selectable town hub）；逐章文字與 E2 畫面仍待。
- [x] **RE-HUB-SUBSCENE-CALLEES**：官方 IDA 9.4 閉合 `0x2e341`／`0x2fc85` 的 raw subscene boundaries：shop family 依 selection/resource 走 `0x2f0b0/0x2f642/0x2f883/0x2f8ea`，hotel family 走 `0x2ffa5/0x30012/0x301f4/0x197e5` preparation path；各自保存 indexed fade/return-to-hub。callee 高階服務名未證實者只留 address-level，不接 normalized menu 語意。
- [x] **RE-CHURCH-SERVICE-SELECTOR-2D7BD**：官方 IDA 9.4 閉合教會 selector：`0x2d7bd` 只讀左右 raw scancode `75/77`，四項 `0..3` 循環；Enter/Space (`28/57`) 回 `1`、Esc (`1`) 回 `-1`。`0x3072f` 的 confirmed raw dispatch 為 `0→0x2ffa5`、`1→0x2f8ea`、`2→0x30dc3`、`3→0x31385`；`0x2d669` 的 transition 已補證清除區、`0x526da` signed cell offsets `[-39,-13,13,39]`、open/close divisor 與 stride `0x140`，仍不猜服務名稱或畫面排列。remake `AdvanceNativeChurchServiceSelection` 與 UI 已改用左右循環，未知 callee/renderer 仍 fail-closed。
- [x] **UI-CHURCH-MENU-INDEXED-2D669**：合法 IDA 9.4 重讀固定 FDOTHER#14 entries3/5/7/9 為四個 normal cell，3/4、5/6、7/8、9/10 為 steady normal/pulse pair，原版實檔均24×20；目的基準 `(240,169)`，兩份位移表 `0x526da/0x526ea` 都是 `[-39,-13,13,39]`。`0x2d669` 每 pass 先 restore 104×20、palette74 的 cleared source，再用 opening divisor4/3/2/1或closing1/2/3/4透明 blit並present；更正舊斷言，只有closing在第四幀後restore cleared source。`0x2d85f(0)→0x2d9fe` 以兩BIOS tick gate令counter mod4前進、selected variant=counter/2。`0x3072f` stable base亦已exact合成FDOTHER#5 raw dialogue grid/four-mode digits、#14 entry1、DATO#131 frame0及FDTXT585/586。runtime用獨立draw-ack job接四幀opening／closing＋cleared-source restore，route side effect延至close完成；保存[`native-church-menu-indexed.png`](../figures/native-church-menu-indexed.png)。2026-08-27重生過時oracle並以正式`church_ch02`固定相位擷取達`AE=0/64000`；此為`RUNTIME-E1`，不是DOSBox E2。四項service現均有typed owner，不再沿用「raw service0未接」的舊現況。
- [~] **RE-CHURCH-RAW-SERVICE-LISTS-2E6B8**：官方 IDA 9.4 閉合 `0x2e6b8/0x2df6b` bounded two-column selector。`0x2f8ea`是church raw1與shop service3共用callee；掃八格 signed flag非負 item，FDTXT510/511/512＋`0x1b8e7→0x1bb8c→0x1b750`閉合物品轉交且不改gold。來源／目的原版 roster、mode1 item list、full feedback與6-open/5-close已接；目的全party roster不排除source，self-transfer現依raw順序重排。保存[`native-transfer-item-indexed.png`](../figures/native-transfer-item-indexed.png)與[`native-transfer-full-indexed.png`](../figures/native-transfer-full-indexed.png)。shop caller的提示／來源／物品／目的五狀態已有route-patched partial E2；church caller、empty/full、mutation與未修改玩家路徑仍待 DOSBox E2，故維持 `[~]`。
- [x] **RE-CHURCH-INVENTORY-TRANSFER-1BB8C**：官方 IDA 9.4 閉合 `0x1bb8c(destination,item)`：掃八格、第一個 flag byte `<0` 的 cell 寫 flag `0` 與 item byte，成功回 `1`，滿格回 `-1`；配合 `0x2f8ea` 的 `0x1b8e7(source)` 前置，raw topology 是來源→目的角色的物品轉移，目的格為未裝備。新增 atomic `battle.TransferNativeInventoryItem` 與 full-destination regression；constructor `0x10c50` 的八格 flags 已資料化，church source gate 不再把 `Equipped` 當成 native signed predicate；未將 raw index 1 命名成高階服務或宣稱 native renderer parity。
- [~] **UI-CHURCH-REVIVE-30DC3**：official IDA 閉合 `0x30dc3→0x309ff/0x30c22/0x30a47`。候選只看 roster record `+5 bit0`，不再以 `HP<=0` projection 猜測；費用為 `word_52669[raw class +0x20] × raw level +0x21`，不再自行把 Lv0 提升為1。remake 已接 stateful 三列名單（sprite/name/race/class/currency/五位數fee）、FDTXT590 的 FFFC名字／FFFA費用／FFFE換行、Yes/No，以及 list 6-open/5-close。第二次 IDA caller 重核刪除「確認一律choice4+dialogue5關閉」的錯誤斷言：原版 `0x197e5` 先只關choice四幀；不足金在仍開著的question第三行 `(12,157)` 寫FDTXT504、`0x16c57(1)`等待後才`0x2d31b`五幀關框；無候選則FDTXT588、`0x16c57(0)`。兩條 indexed message lifecycle已接。成功 `0x2f4c6` 因 hub selector固定4而只走case4：FDOTHER#14 entries23..31 sequential transparent blit到 `(147,32)`、每幀2 BIOS ticks，DAC 0→62/62→0每步4ms，`0x17aa9(10/5)`是相對前次latch而非額外hold；monotonic indexed timeline與原版資源 regression已接。再次指令級重核刪除「`sub_25977(17/11)`是PCM SFX」錯誤斷言：它是`play_bgm(track,loop_count)`，直接載FDMUS track17/11並各設loop count1；`playBGMCount`已依動畫前／後時序接入。仍需DOSBox E2視聽diff，故維持 `[~]`。
- [x] **RE-INVENTORY-CONSTRUCTOR-FLAGS-10C50**：官方 IDA 9.4 閉合 constructor 八格 flag writes：cell0=`0x40`；source byte0=`0xff` 時 cell1=`0x80` 並將 source byte1 放入 cell0，否則 cell1=`0x40`；cell2..7 依 source `0xff` 為 `0x80` 否則 `0x00`。新增 `NativeInventoryFlagsFromSource`／signed gate regression，`Load`/`PartyUnits` 持有 raw flags；修正 `0x2f8ea`「只選未裝備」錯誤斷言，`0x40` equipped 亦是非負可選 cell。
- [x] **RE-CHURCH-RAW-0-17AED**：官方 IDA 9.4 與 Docker 指令級重讀固定 `0x2ffa5→0x17aed(actor)` 是單 stack actor（Hex-Rays 第二參數為 artifact）：`0x17e0b(actor)` item/status staging→key wait→`0x1c269(actor,0)` gate→可選 `0x1ceed(actor,-1,...)` command/MP overlay→key wait→restore；body 無 persistent writer，因此撤回「能力服務」命名並定為唯讀角色資訊／狀態 presentation。remake 已接 `0x2e6b8/0x2ea90` 兩欄六人 roster、6-open/5-close+restore，以及 status 12-open、bottom 7-close/7-open、`0x1ceed` FDTXT441+ID/cell92/MP digits、12-close+restore後重開名冊；全部由 Draw acknowledgement 推進。保存 [`native-status-command-indexed.png`](../figures/native-status-command-indexed.png)。command effect/target 不屬此唯讀 service，仍由 UI-03 fail-closed。
- [x] **戰場進入分割滑動原語**：官方 IDA 確認 `0x1f42d/0x1f1cc` 使用 FDOTHER#5 LMI1 第 `0x52` 項，以 456 列距在 `(85-offset,82)`／`(165+offset,81)` 執行 `100,75,50,25,0` 五步，每步呈現後還原；新增 `NativeBattleEntrySplitSlideSteps`、邊緣裁切繪製、回呼執行器與回歸測試。2026-07-29 直接重讀呼叫者 `0x1a30b`，確認它處理戰場記錄與 456 列距戰場緩衝；撤回把這項列為 UI-11 選人視窗動畫的錯誤分類。未命名 MAP/TURN，也未接未證實的行軍輸入。
- [x] **RE-RAW-1A866-FIRST-LOOP**：Docker Capstone 重新讀 `0x1a866` 的第一個 raw loop，固定 `+0x25!=0`、`+0x06==selector`、`+0x05 bit0==0` 三個 gate，以及 `+0x40 -= +0x42/10`、負值 clamp、global divisor write；新增 `ParseNativePreparationRecord`／`NativePreparationEligible`／`NativePreparationAdjustedWord40` 與 malformed/gate/clamp regression。此項只保存 address-level raw branch，不命名 preparation/UI/deployment；同函式第二段 `+0x22..+0x27` decrement 另由 transient lifecycle 條目追蹤。
- [x] **UI-11 preparation dispatch table**：Docker Capstone 固定 `0x1a813` 的 `base+3*i`、16 slots、`+3/+5` gate 與 `+4` function-table index；新增 `FindNativePreparationDispatch`，保留重疊 3-byte raw layout、short-table fail-closed 與多重命中 regression，不執行未命名 callback。
- [x] **UI-11 preparation timer transition**：Docker Capstone 固定 `0x1a941` 對 0x50-byte record 的 selector/inactive gates、六個 `+0x22..+0x27` counter decrement，以及僅 1→0 才產生 `0x1e1+index` downstream source；新增 `TickNativePreparationTimers` in-place raw planner/regression，不命名狀態或效果。
- [x] **UI-11 preparation input ABI**：Docker Capstone 固定 `0x19953` 的 raw scancode branches：`E0/52/1C/39→1`、`01/53→-1`、`4B→cursor0`、`4D→cursor1`，其他輸入繼續等待；新增 `ApplyNativePreparationInput` 與 regression，不把 return 1/-1 猜成 YES/NO。
- [x] **post-resolution raw command stream correction**：重新對照 FDTXT_000，確認 `0x1aa1d` 的 `0x1b0..0x1b3` 是掉落／互動訊息，不是 preparation UI；撤回 `UI-11 preparation command stream` 命名。保留 `0x1ac62` 的 `base+3*i` `{kind byte,payload word}` raw parser（kind 0/1/2/3 observed branches），改名 `ParseNativePostResolutionCommands`，不接 D8。
- [x] **RE-PREPARATION-INPUT-32004**：Docker Capstone 重核 `0x32004` 的雙位元組輸入介面：`0xe0/0x52` 與 `[0x53a8d]==0x20` 都正規化為 `0x1c`；只有未走前述分支且 `[0x53a8e]==0x53` 時才回傳 `1`，其餘保留初始值 `0x10`。呼叫端 `0x31a29` 對 `1`／`0x1c` 的後續分支亦已記錄。先前「`0xe0/0x52` 原樣回傳」的錯誤已修正；`NormalizeNativePreparationKey` 只保存位元組介面，不替按鍵或畫面命名。
- [~] **UI-11 原版整備選人主畫面**：Docker Capstone 重讀
  `0x318ad..0x32004`，固定三區背景、兩組二位數、10 欄角色格、游標先畫、
  已選角色上移三列且走 `0x4deda`、未選走 `0x4de56`，以及左右 ±1／
  上下 ±10 的邊界。新增混合解碼資源包、原子合成器、原始 selector key
  生產接線與真實 FDOTHER／FDICON 回歸；局部證據圖為
  [`preparation-roster-compositor-partial.png`](../figures/preparation-roster-compositor-partial.png)。
  此圖以原始圖像索引 0～19 建立，明確不是 DOSBox 截圖、正常戰役名冊或
  `FD2.SAV` 證據。游標角色的右上狀態已直接重用既有、真實資源驗證的
  `0x17fc0` 合成器與完整 0x50 位元組記錄；缺原始欄位即整張退回。
  `0x1297d` 待機週期已重用 `fdicon.AdvanceNativeMapSpriteCycles`：
  有號 BIOS 低字差值 `>4` 或回繞才前進 raw state，繪圖再由
  `NativeFrameIndex` 把 3 映成1；生產可見序列為 `0,1,2,1`。
  `0x31d3c` 穩定最終確認亦已接：`0x1956b(0x4b)` 的 FDOTHER #5 對話框與
  DATO #75、FDTXT `0x292` 的 `(95,119)` 起筆、`0x16559(0)` 最後肖像覆蓋，
  再疊 `0x19953` 的 FDOTHER #2 Yes／No。保存
  [`preparation-confirmation-compositor-partial.png`](../figures/preparation-confirmation-compositor-partial.png)；
  這是 E1 原始資源合成，不是原版實機。完整生命週期也已接正式路徑：
  `0x1956b` 六幀開框、`0x19953` 四幀展開與兩 tick 脈動、
  `0x197e5` 四幀關選項、`0x2d31b` 五幀關框，再呈現原畫面後才執行結果；
  每一步只在 Draw 確認後前進。保存
  [`preparation-confirmation-lifecycle.png`](../figures/preparation-confirmation-lifecycle.png)。
  下一門檻是跨畫面的行程全域初始相位，以及合法晚期存檔的同狀態實機差分。
- [~] **SDD-3 UI shell vertical slice**：已新增 `TestUIShellVerticalTraceKeepsPostbattleTownAndShopBoundary`，以 title confirm、story→battle、battle win→editable postbattle、town→shop→town 的同一 state trace 固定「戰後不可直跳下一戰」；既有 town/shop/preparation 截圖 artifact 與 Docker/Xvfb regression 可重跑。battle field/action/dialog 的同一條畫面 trace、原版 DOSBox pixel differential 仍待補齊。
- [ ] **SDD-4 native renderer re-audit**：完成 resource provenance 與 indexed buffer contract 前，不得把 finale figure-fade／ending prefix 宣稱為完成。
- [x] **RE-UNIT-STATIC-TABLES**：以 Docker 實際 FD2.EXE 產生/驗證 raw fixture：高 branch `b1-0x44 → 0x61af9` 68×10；lower branch `0x61da1` 32×24／`0x620a1` 68×11。constructor caller 的 level 公式與 `+0x42` join 已由 Capstone 固定；`export_units.py`／`sync_native_selector_fields.py` 將 raw provenance 輸出到 33 張 editable map asset。未被 table 覆蓋的 selector 與 HUD renderer consumer 仍維持 fail-closed；`0x619fd` 不屬於 constructor。
- [x] **INDEXED-FRAME-TEST-CONTRACT**：修正 native compositor fixture 的 `work+0x8088` 來源邊界、range descriptor bank 與 viewport copy 座標；Docker indexedmap regression 通過，未放寬 production fail-closed 條件。
- [x] **REGRESSION-BLOCKERS-2026-07-26**：Docker image 內建 Xvfb 已納入完整 regression command；ch14 final dialogue line mapping 依 FDTXT_015 count-aligned continuation 修正，ch16 conditional spawn 僅 branch-local after LOADCH。完整 suite 通過，未刪除有效 assertion 或放寬 fail-closed compiler。

## 2026-07-20 ending prefix playback slice

- [x] **0x2bce5 可播放前綴（2026-07-20 歷史切片）**：`internal/ending.Player` 以毫秒 clock 依原序執行 frame0/copy/hold、ANI#2、baseline-derived ramp、兩段 native text、frame12..108、40/200-pass composite，再銜接 `0x2c405` 的500-pass scroll。本條當時仍只在 `FD2_APPROXIMATE=1` 後繼續；現行狀態已由下方 2026-08-21 勘誤取代。
- [x] **獨立畫面 oracle**：`FD2_ENDING_PREFIX=1` 會讀玩家自備 DAT，將 indexed VGA DAC 轉為 320×200、2× 顯示於 Ebiten；它不接 campaign，故無法假裝原版終局已完成。可用 `FD2_FDOTHER=/path/FDOTHER.DAT`、`FD2_FDTXT=/path/FDTXT.DAT`、`FD2_ANI=/path/ANI.DAT` 指定素材（FDTXT預設同FDOTHER目錄），並沿用 `FD2_SHOT` 截圖。

> **2026-08-21 勘誤：**上述兩段描述是當時的獨立預覽／受限近似入口。
> 最終 `ending` 節點現由正式 `battle_ch30` 以嚴格 `native_ending_prefix`
> 來源約束 E1 合約建立，不再依賴 `FD2_APPROXIMATE=1`；它使用
> 相同的前綴播放器；它抵達 `0x2c548` 時消費 `FDMUS_004`，來源約束路徑隨即播放
> 原資源 party montage。成功完成後會以 20 組 TAI／BG／FIGANI 與 #58 疊圖播放
> 近似尾段，再停在 #59；只有 admission
> 失敗才等待確認回到可編輯結語。`FDMUS_018` 在近似尾段開始時接線，但不主張其
> 精確停曲、間隔與畫面同步等同原版。
> 這不接 raw ch29 terminal handler，也不改變一般玩家 E2、精確 BIOS key mapping 或
> 精確 `0x28a6c` renderer 未完成的狀態。
- [~] **下一個 ending gate**：`MontageCycle` 已完成 FDOTHER#56 backdrop、FIGANI/DATO、dialogue-frame grid、mirror/non-mirror figure fade、primary descriptor delay 與 64 段 palette 收尾的 standalone indexed adapter。IDA 已閉合 `0x10620` 的兩個 raw word 比較與 `0x4e031` 的 `0x41a→0x41c` 複製；`fdsave.NativeBIOSKeyboardState` 只提供這個原語，不解碼按鍵。`0x28a64` 已校正為共用清理 epilogue，不是返回 owner；`MontageTailAssets` 驗證 `0x2c194..0x2c39a` 的四項 FDOTHER 資源，`MontageTailPlayer` 已接實際80個 FIGANI 全部可達的 header-byte1-zero `0x2939D` 主迴圈與 #58 疊圖並保持 #59。`MontageTail.Plan` 的三組 20-byte table 正確表示 record0／record1 各自的 `+7`、`<0x4c` 推得的 `+6` 與 `[0x540ff]`。`MontageTailLoaderBaseline` 也會以真實 FDFIELD／FDICON 建立 `0x1088d(0x1e)` 後的 31-record 基線。下一個 gate 是精確呼叫時 records/globals 動態連續性、3% RNG重播、sound owner、按鍵映射及一般玩家 E2；已接的可達 renderer 不再列為整體缺口。證據見 [`fd2_ch29_input_cleanup_ida.txt`](../data/ida/fd2_ch29_input_cleanup_ida.txt)、[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt) 與 [`fd2_ch29_tail_nonzero_renderer_ida.txt`](../data/ida/fd2_ch29_tail_nonzero_renderer_ida.txt)。

- [x] **終局隊伍回顧循環（重製延伸）**：預設流程現為角色最終狀態 → 20 組近似尾段 → #59 定格；玩家以 Enter／空白鍵才重播已 admission 的 `MontageCycle`，每次完整隊伍最終狀態播放後重新開始，Enter／空白鍵／Esc 回到 #59。它不改寫戰役、不模擬 DOS BIOS input，也不宣稱原版存在該循環；精確 `0x28a6c` 與一般玩家終局 E2 仍是獨立 gate。

### 2026-07-20 native ending dialogue bridge

- [x] **0x2c39b 原生對話（2026-08-25 speaker／owner勘誤）**：`internal/ending.Player.BlockedDialogue(chapter)` 僅在 `native_text_branch_opaque` 取出 chapter26 then 或 final else blocks；`FD2_ENDING_PREFIX=1` 以 `FD2_ENDING_CHAPTER=26|29` 明確選 branch。`ch27.json`／`ch30.json` 的 scene／line／count 只保留文字來源對照，不是現行 renderer 的一般逐行輸入；正式可編輯顯示資料是 timeline 的 `native_utterances`。舊版把 block `portrait_id` 強制套給每句的做法已被 FDTXT_030 index2..7 內嵌 `FFEE/FFEF` operand 直接否定；runtime 現逐句使用4／24／126／0等原始 speaker，chapter26 index17..20 沒有內嵌 speaker 才沿用 caller initial portrait。19×5 索引 owner 已接六段 opening、最多四列逐字、嘴型、Enter／Space、五段 closing 與 source restore；每段完整收框後才 resume 當前 text gate，第二段 `0x2bf1c` 也可重新排入。一般玩家 E2 仍另列。
- [x] **native text gate resume**：對話所有頁／句皆完成後，preview 只可呼叫 `ResumeBlockedDialogue()` 恢復該一個 `native_text_branch_opaque` segment；任何 composite 或其他 opaque gate 都被拒絕，避免 UI 誤跳過未 RE renderer。每次成功resume會清除preview queue latch，讓後續已證實的第二個text gate可獨立排入。
- [x] **text 後 palette repeat**：`palette_ramp_repeat` 的 native `repeat=3`、63→0、4ms、`tail_delay_ms=200` 現由 player 展開成三組明確 `palette_ramp + delay`，不以普通 fade 代替；其後frame12..108 sequence與第二text gate均已有executor。
- [x] **frame12..108 sequence**：`blit_frame_sequence` 現展開 frame12 到108 的 transparent VGA blit 與每幀20ms wait；第一段 text 後 resume 可走到第二個已知 native text gate。composite 的 string formula 改名 `first_frame_formula`，避免與 sequence integer `first_frame` 的 JSON schema 衝突。
- [x] **ending composite frame selection regression**：`0x2bf60` 與 `0x2c0c5` 的兩張角色 frame 都是 `(i%4)+1` / `(i%4)+5`，不是 `floor(i/4)+1`；timeline 與測試已鎖定。200-pass scheduler 完成後會停在 `0x2c172` 的未恢復 montage gate，不會誤報 ending complete。
- [x] **first 40-pass composite primitive**：新增 640×200 work buffer、`CopyRect`、帶 byte-origin bounds 的 `Frame.BlitAt`，並以 native primary/secondary offsets + viewport x=160 實作 `Composite40(i)`；尚待 scheduler 接線，第二 loop 的 palette helper 繼續封閉。
- [x] **first 40-pass composite scheduler**：player 現以每輪20ms 驅動 `Composite40(i=0..39)`，完成後精確落在第二段 native text gate；200-pass loop 仍因 `0x11d40` 未證實而封閉。
- [x] **second 200-pass composite scheduler**：baseline palette loop 已恢復為200×20ms（0..135 base、136..199 base−1）；其後 `0x2c172` 明確標為 unrecovered montage gate，禁止 player 回報完整 ending。
- [x] **finale 0x2c405 phase-0 map**：已確認 chapter30 text load、`0x36b00` staging/clear、`+0x12c30` text-composite destination、500×(1ms) 的 320×200 row-scroll 與 baseline palette 40→0→上升 cadence。strict FDTXT_031/#44 glyph staging 已恢復；仍只在後續 `0x2c548` montage gate 停止，禁止用 generic fade 或空白畫面替代。
- [x] **finale phase-0 editable script node**：新增 `assets/endings/native_2c405.json` 和嚴格 loader；正確呼叫點是 `0x2c435 push 0x1e`、`0x2c437 call 0x1088d`，loader 依 raw selector 規則選 FDTXT_031，故 `0x2c` 是其合法實體 string #44（46 strings）。內容是後日談跑馬燈前言，對位跨資源重用的 `ch32.json` scene0 line0；staging/layout/timing/palette cadence 均資料化。asset 可編輯但 `Ready()==false`，直到完整 finale montage 都恢復。
- [x] **native FDTXT/font decoder foundation**：`internal/fdtxt` 現嚴格讀原始 offset-table、保留所有 FFxx 控制字、精確解 `FDOTHER_004` 的 16×16 1bpp（MSB-left）glyph。尚未猜 palette／框／控制碼行為；下一步把已知 `0x15f84` layout 接成 compositor。
- [x] **native glyph staging primitive**：`Font.BlitGlyph` 將 FDOTHER_004 的 set bits 以明確 caller palette index 寫入 indexed buffer、zero bits 完全透明，且有 pixel regression。`0x4ea2a` 的實際色彩／layout 參數仍未假設，不能因此解除 finale gate。
- [x] **0x4ea2a glyph ABI closure**：Docker Capstone 確認 native glyph renderer 是 16×16 前景 + 左／下陰影，background 非零才清 cell；finale `0x2c469` 的 caller 展開為 stride320、foreground `0xCD`、shadow `0x4C`、background0。`BlitNativeGlyph` 已 pixel-test 這個 ABI；FFxx flow/staging backdrop 尚未完成。
- [x] **finale phase-0 raw glyph composition**：`ComposePhase0Text` 現真正把 FDTXT_031 physical #44 的121個 glyph、9個 `FFFE` soft line breaks，以 `staging+0x12c30`、16-byte advance、每行 `25×320` rows、CD/4C/transparent native style 寫入。實機資源 regression 逐 bit 驗 foreground/left/down shadow；除已證實 FFFE 外任一 FFxx 仍拒絕，整段 finale 仍 fail-closed。
- [x] **finale phase-0 scroll scheduler**：`Phase0Player` 精確執行500 passes：每輪 baseline palette→`staging+i×320` 的320×200 copy→wait1ms→i++；i=0..199 每5輪（含0）將40逐步降至0，i=301後每5輪（首個305）升回。完成只回傳 phase done，不會跨 `0x2c548` montage gate。
- [x] **FDTXT archive provenance**：Capstone 直接證實 `0x2c435 push 0x1e`、`0x2c437 call 0x1088d`，loader 先將 raw selector 遞增再傳 `0x111ba`，所以載 archive resource31；其 bytes 已 byte-for-byte 對照 extracted `FDTXT_031.bin`。先前 resource30 mismatch（5762 vs 6756）是 off-by-one，phase preview 可安全取 resource31。
- [x] **finale phase-0 bridge**：`ending.Player.EnableRecoveredPhase0` 只在精確 `0x2c172` hand-off 執行；它取前段 `PresentANI` 已捕捉的原版 DAC baseline（無 provenance 即拒絕）、清 VGA、以 FDTXT_031/#44 與 FDOTHER#4 生成 staging，逐毫秒跑完 500-pass scroll。完成後精確停在未恢復的 `0x2c548`，不會誤宣告 ending complete；regression 覆蓋首幀、baseline 缺失拒絕與 montage gate。
- [x] **finale `0x2c548` first party-cycle map**：Docker Capstone 切出三個 native buffers（128000、64000、64000）、TAI#3 與 FDOTHER#56 backdrop；更正：TAI#3 raw 是 `10×3`、三列 `C9` 的全透明 placeholder，不能誤稱可見 platform。loop index 從 `[0x53bfb]-1` 向下，但必做 `i=0→slot1、i=1→slot0` swap，才以 unit stride80／visual group `+7` 載 `FIGANI group*3+1` 與 `group*3`。`0x29164` 對 `unit+6` 是 zero/nonzero test，不可把非零限制為1；其後先有 `0x2b9a1` 的20×1 BIOS-tick loop，再跑 primary FIGANI descriptor `+6` tick frames；舊 `20×1ms` 斷言已刪除。已入 `assets/endings/native_2c548.json`，並由近似模式的 `MontageCycle` 以原始資源執行；精確輸入、後續 owner與一般玩家 campaign handoff仍 fail-closed。
- [x] **finale party portrait/text executable slice**：DATO=`unit+7`；FDTXT_031 的 #10/#11/ending epilogue 與 FDTXT_000 的角色名／職業名，五個 destination 與 CD/4C glyph style 均已直接對齊。DATO 固定貼 `staging+0x0c88`；countdown=0 時重設 `(random&31)+40` 且本輪不減，非零先減，結果 `<2` 用 frame3、其他用 frame0。一般 loop 220 ticks，只有 loop index0（swap後slot1）跑440 ticks並自tick220改用FDTXT_031 #45。`ComposeMontagePortraitFrame` 已以玩家 FDOTHER/FDTXT/DATO 做原資源 regression；未知控制碼仍拒絕。`MontageCycle` 現在只在近似最終節點以 persistent raw roster 接線，並受限消費 portrait-loop 的 raw input-change 效果；精確 key mapping、tail owner 與一般玩家 campaign handoff 仍是獨立 gate。
- [x] **finale dialogue-frame layout call ABI**：IDA/`14-text-control-codes.md` 交叉確認 `[0x53a81]` 是 `FDOTHER.DAT#5`，不是 DATO；`0x2c773→0x168b6` 實參為 `(destination=C, stride=0x140, arg8=5, argC=7, arg10=5, arg14=5)`，先建立 dialogue frame/grid，後續才由 DATO `[0x53a85]` 經 `0x4e8af` 貼 portrait。已撤回 `dato_layout` 錯誤命名，schema 改為 `dialogue_frame_layout`。
- [x] **DATO indexed decoder foundation**：新增 `internal/dato`，按 `0x4e8af→0x4e916` 高值-run codec 解四個 80×80 mouth frames，零值保持 opaque（不套 transparent sprite 規則），並提供 strict bounds checked indexed blit；synthetic RLE/opaque-zero 與玩家 DATO#37 regression 已加入。mouth cadence 已由 `MouthState` 接入對話更新迴圈，但不宣稱完整 dialogue parity。
- [x] **dialogue-frame `0x168b6` raw grid plan（2026-07-27 correction）**：舊 `Montage.PlanDialogueFrameGrid()` 漏掉 `v6` 的 `a3=5` 並混用 byte/stride 項，所謂 exact arithmetic 斷言錯誤。新增共用 `fdother.PlanNativeDialogueFrameGrid` 逐一保存49次呼叫；首 offsets 修為2245/2328、portrait grid origin=3208、尾格=23752，ending改委派此 planner。
- [x] **FDOTHER#5 raw-cell codec correction**：`0x1685c→0x4e9bb` 只讀 width/height 後逐 row `rep movsb`，不使用 `0x4e916` high-run；新增 `fdother.ParseLMI1RawEntry/DecodeLMI1RawEntry`，真實 #5 entry1 (`3×3`, literal `60 be bd...`) regression 固定此 path，避免把 dialogue frame bank 誤套 LMI1 RLE。
- [x] **dialogue-frame raw compositor**：`RenderDialogueFrameGrid` 依 49 個 verified placements 直接將 `FDOTHER#5` raw cells 寫入 C buffer，明確使用 opaque `rep movsb`（包含 zero bytes），不接 DATO/text/input；synthetic overlap/zero regression 通過。
- [x] **dialogue-frame resource-backed compositor**：`RenderDialogueFrameGridResource` 現實際載入玩家 `FDOTHER.DAT#5` entries 1..17，再按 native placement/overwrite order 寫入 C buffer；缺檔仍 fail-closed，asset regression 驗證非空輸出。
- [x] **DATO opaque paste primitive**：`RenderDATOFrameAt`／`dato.Frame.BlitAtOffset` 對應 `0x4e8af` 的 stride-320 opaque frame paste，destination offset 必須由 caller 明確提供（native 常見 `staging+[0x53c67]`）；不把該 global 猜成固定 anchor，也不接 mouth cadence。
- [x] **finale `0x2c548` official-IDA gate recheck (2026-07-26)**：IDA 9.4 ASM 直接確認 phase-0 的 `0x2c548` gate 先 free 500-pass staging，再配置 `0x1f400`、`0xfa00`、`0xfa00` 三塊 indexed buffer；以 `sub_111ba("TAI.DAT",3)` 與 `sub_111ba("FDOTHER.DAT",0x38)` 載入 montage inputs，FDOTHER #0x38 先以 stride `0x140`、transparent `-1` blit 到第一個 64000-byte buffer，接著由 `[0x53bfb]-1` 反向取 party record。這是專用 indexed montage 的資源/緩衝 ABI，不是 generic fade 或可直接替換的 PNG scene；DATO、FIGANI schedule、mirror branch 與 standalone executor 已閉合，輸入／campaign owner 仍 fail-closed。
- [x] **indexed FIGANI decoder foundation**：新增 `internal/figani`，直接讀 FIGANI LLLLLL resource、13-byte frame header（signed X/Y、delay、real W/H）和 4-mode RLE，透明 span 以 mask 保留而非轉 palette0；`BlitAt` 寫入 indexed surface，實機 `FIGANI.DAT` #13 regression 通過。下一步是 TAI frame 與 native 0x29164 fade/composite，不能改走 RGBA PNG。
- [x] **native FIGANI scheduler `0x2b9a1` (2026-07-26)**：官方 IDA 確認 `arg4==0` 僅清 `byte_540fc`（subframe）且不 render；非零路徑先以目前 `byte_540fd` frame 呼叫 `0x2935b`，再讀 descriptor `+6` delay，累加 subframe，達 delay 才換 frame 並於 frame count wrap。`internal/figani.NativeScheduler.Step` 已照此實作與 regression；renderer 仍由 caller 顯式提供，未猜測 `0x2935b` 的 presentation semantics。
- [x] **戰鬥 FIGANI 幀延遲呈現橋（2026-08-09，E1）**：從固定版本
  `FIGANI.DAT` 擷取 22 個已匯出動畫的 descriptor `+6`，保存為
  `remake/assets/figani/delays.json`；`cmd/fd2` 回歸逐幀比對原始延遲與幀數。
  `internal/figani.DisplayScheduler` 現供全螢幕攻擊 PNG 幀選擇與停留時間使用，
  `FD2_BATTLE_FPT` 只作明示顯示倍率。缺少配對資料即不建立演出，不再補猜固定
  15 幀。此項不解除命中／傷害／音效／台座／完整原版畫面或 E2 gate。
- [x] **0x29164 first fade closure**：第一參數是 party loop unit index（讀 `[0x53a45]+unit×80+6`），不是 TAI；TAI#3 是尾端 aux argument，7-byte transparent raw 不可餵 `0x2935b`。兩條 native path 都做 `esi=8..0` 共9次 present，每次 DAC baseline delta=`esi×6`（48→0）。2026-08-28 runtime gate已改由來源綁定的10×3全透明分離surface驗證，不再即時讀raw；這不改寫原版7-byte證據或把它當可見平台。
- [x] **non-mirrored figure-fade schedule**：`native_2c548.json`／`Montage.PlanFigureFade(1)` 現嚴格記錄 final caller 的 `unit+6==1` branch：work stride640、320×200 left viewport、stage byte offset `8..0 ×10`、palette delta `48..0`，TAI#3@164,157 explicit transparent no-op、secondary FIGANI frame0。`unit+6==0` mirrored branch已有獨立 planner／pixel primitive，未把非鏡像公式套用。
- [x] **non-mirrored indexed fade primitive**：`RenderFigureFadePass` 現真正執行每輪 B→A（320→640）restore、secondary FIGANI 在 `stage×10` 的 indexed blit、A left viewport→VGA 與 baseline DAC delta；TAI#3必須是來源綁定的10×3全透明 no-op。像素 regression 鎖住 backdrop 保留、stage shift 與 48→2 palette；B→C 的 post-figure `memmove(64000)` 亦已記錄，供下一段 portrait renderer 使用。
- [x] **`0x29164` mirror-branch ABI transcription (2026-07-26)**：官方 IDA 釘出 `unit[+6]==0` 的 `0x2927e..0x29357` 路徑：仍為 `stage=8..0`、每步 palette `stage*6`，但 primary FIGANI source 是 `staging+0x140-stage*10`；只有 `arg4==0` 才額外畫 TAI#3 與 secondary FIGANI。`native_2c548.json` 已保存 `mirror_branch` editable metadata，`MontageCycle` 依 caller 的 `arg4=1` 逐步消費；不因此解除輸入／campaign owner gate。
- [x] **mirror fade planner**：`Montage.PlanMirrorFigureFade(unitSide,sideFlag)` 現輸出 9 個 exact stage/offset/palette pass，並明確保存 `arg4==0` 的 secondary/platform gate；`MontageCycle` 只在證實的 final caller `arg4=1` 路徑消費 pixel pass，不把 planner 當成通用 renderer。
- [x] **mirror indexed fade primitive**：`RenderMirrorFigureFadePass` 依 `0x292ad` 的 caller-preseeded 640-stride right viewport，先 present `work+0x140`，再以 `staging+0x140-stage*10` 畫 primary、按 `arg4==0` 畫 secondary，最後 present 同一 viewport；TAI#3僅做來源綁定的全透明surface validation。pixel regression通過，並已由`MontageCycle`與DATO/FDTXT portrait階段串接。
- [x] **RE-UNIT-RAW-SCHEMA**：`export_units.py` 與 `battle.NativeConstructorTable` 已保存已證實 branch/index/raw records，嚴格拒絕 malformed dimensions；此項只完成資料邊界，不代表 renderer/gameplay 已接通。
- [x] **RE-MAP-SPRITE-RAW-CYCLES**：閉合 `sub_1297d` 的完整 mutation：
  moving `[0x53c07]` 每次 call 必定 0..3 循環；idle `[0x53c0b]`
  只有 signed `BIOS-tick-last<0 || >4` 才循環並更新 `[0x53c0f]`。
  `AdvanceNativeMapSpriteCycles` 保存兩條 pure ABI；舊 HUD-only helper
  委派同一實作。`[0x46c]` 已由 `0x17aa9` 的 0x10000 wrap busy-wait
  及 `0x16d00` 的 two-tick gate閉合為 low 16-bit BIOS timer tick，不是
  VGA scanline；runtime monotonic-clock materialization/call timing尚未接。
- [x] **RE-RUNTIME-POSE-MOTION-LIFECYCLE**：player materializer
  `0x10a77..0x10aad` 與 FDFIELD constructor 都建立 raw `+3/+4=0/0`。
  四向 movement entries 固定寫 `+3=方向`，每格 draw loop 寫
  `+4=1..6`，第七拍更新 X/Y 後寫 `+4=0` 且保留 pose；`0x1366a`
  normal acting 相同，special 只寫 pose。doc54 已刪除錯位 acting dump
  的影片推測，改成 direct writer/consumer lifecycle。remake 現以獨立
  `NativeMapPresentationState` 保存 `+0/+1/+3/+4`；selector materialize
  同時初始化 raw X/Y、pose0、motion0，一般移動與 acting 都跑來源格
  motion1..6／第七拍提交。`NativeUnitLayerEntry` 缺 presentation、
  selector slot 或 record byte5 任一 provenance 即 fail-closed。
- [~] **RUNTIME-NATIVE-MAP-CLOCK-AND-FRAME-INPUT**：成功建立 native
  selector batch 後，`battle.State.NativeMapCycleState` 已擁有
  `[0x53c0b]/[0x53c07]/[0x53c0f]`，且只接受 signed low BIOS word；
  legacy state 不會猜值。official IDA/Capstone另關閉`0x11eee`：
  `[0x51a93]==-1`時signed BIOS delta>2或wrap才令`[0x53c1f]`
  modulo20前進並更新`[0x539f4]`；override0..19直接選phase且不改latch。
  terrain `[0x53a40/0x53a00]`與unit pixel shift
  `[0x53a04/0x53a08]`則是兩組獨立「新BIOS word翻轉」state，均已由
  State持有。`nativeBIOSClock`現以PIT `1193182/65536≈18.2065Hz`
  的battle-local monotonic low word，在每次steady redraw只呼一次
  `0x1297d`並更新terrain phase／兩個binary latch；signed 16-bit wrap
  有regression。七拍movement仍由Ebiten Update驅動，command/target
  range-mode writers也尚未materialize，故整項仍partial。
- [x] **RUNTIME-NATIVE-MAP-RAW-ROSTER**：新增
  `NativeMapFrameRoster`，一次性建立unit/foreground arrays與cycle
  snapshot。foreground另外要求explicit `unit+7`、race、class；
  舊JSON的`BattleFig=Fig` fallback以`HasBattleFig=false`隔離，不能混入
  native compositor。任一record缺provenance即整批拒絕，不回傳半張
  native frame。它已供strict `NativeFrameInput` builder使用；尚待的是
  clock、camera/range/HUD globals的production caller ownership。
- [x] **RUNTIME-NATIVE-FRAME-INPUT-ADMISSION**：
  `buildNativeMapFrameInput`已將original banks、FDFIELD cells、
  selector cache、完整raw roster、terrain LUT phase、idle/moving
  cycles、terrain flip與unit pixel shift組成單一
  `indexedmap.NativeFrameInput`。editable control table必須與實際
  FDSHAP bytes完全相等；camera/range/cursor/HUD globals必須由caller
  明示，禁止從640×400 camera、normalized reach或PNG UI猜值。
  下一步是原版320×200 camera與HUD gate/anchor/clock runtime ownership，
  不是再做一個renderer primitive。
  - [x] 2026-07-28 production neutral-frame bridge：ch01 map JSON重新由
    合法原版輸出576個封存 byte+3、1200-byte control table與event low bytes；
    ch01改走已證實party-first、initial-groups-append constructor順序。
    玩家原始DAT integration test可完整通過`ComposeNativeFrame`並由Ebiten
    呈現，artifact為`docs/figures/native-map-ch01-remake.png`。raw range
    mode只接受campaign明示0；其他UI modes未接前回playable renderer。
- [x] **RE-UNIT-PRESENT-SNAPSHOT-OWNERSHIP**：`0x22253` 只配置一塊
  `0x25680` snapshot：terrain-only狀態供11個intro frames restore；
  `0x22547` entry再把final LMI `#0x7c`畫進同一塊，後續6 contract +
  10 release全部restore這份terrain+LMI snapshot。coordinate rewrite與
  strip-copy bridge不改它。新增atomic
  `ComposeNativeUnitPresentLUTSnapshot`及invalid-input regression，撤回
  contract/release可任意提供不同snapshot的舊註解。
- [x] **RE-UNIT-PRESENT-DIRECT-VGA-BRIDGE**：更正「總共27 presents」
  的不完整斷言：27只計full-viewport `0x11eb0`。contract後另以FDOTHER
  #3 entry0 pointer+1做一次不present的`0x22046`，再逐row從456-stride
  work buffer直接memmove 24 bytes到320-stride VGA，每row delay10ms。
  targetY==cameraY時從target row寫18 rows；否則從上方6 pixels起寫24
  rows。新增layout/progressive-copy/bounds regressions；故可觀察schedule
  是27 full presents + 18/24 direct row reveals。
  `ComposeNativeUnitPresentStripBridge`另將snapshot restore→bridge-only
  LUT/object redraw→direct rows接成單一transaction，並以untouched VGA
  regression防止誤插full viewport copy。
- [x] **RE-UNIT-PRESENT-SHIFTED-LUT**：真實FDOTHER #3 offsets
  `0x66/0x166/0x266...`證實每table連續256 bytes；`0x22547` shared
  epilogue回傳entry0 pointer+1，而`0x22046`仍讀256 entries，故bridge
  table精確為`LUT0[1:256]+LUT1[0]`，不是LUT0或LUT1。
  `NativeUnitPresentBridgeLUT`與player archive regression固定跨entry
  boundary，禁止aligned LUT近似。
- [x] **RE-UNIT-PRESENT-FIVE-ARG-ABI**：五個direct callers固定
  `0x22253(unit,newX,newY,visualX,visualY)`；intro/contract先用visual
  pair，之後才寫record `+0/+1=new pair`再跑bridge/release。command23
  先`new=ff/ff,visual=current`消失、再`new=visual=destination`出現；
  ending只做unit1消失，script helpers用兩pair相等。新增
  `PlanNativeUnitPresentCall` byte-boundary regression，不再泛稱兩pair為
  source/destination。
- [x] **CH29-POST-FLOW-WIRING 歷史勘誤（已由 2026-08-21 E1 閉合取代）**：本項當時撤回 `postbattle_ch29_persist→ch29_post` 的錯接，並在 renderer／E2 gate 尚未具備時保持未綁定；目前正式 `ch28_post` binding、presenter、group9、持續隊伍及 `preparation_ch30` 存讀檔已達 `RUNTIME-E1`，只剩一般玩家 `PLAYER-E2`。原始 handler table 證據仍有效：index26→`0x250cc`、index28→`0x2548c`、index29→`0x25757`；index29 的 `0x25970→0x2bce5→0x25975` 仍是獨立 self-loop。證據見 [`fd2_ch28_post_ida.txt`](../data/ida/fd2_ch28_post_ida.txt) 與 [`fd2_ch29_terminal_callers_ida.txt`](../data/ida/fd2_ch29_terminal_callers_ida.txt)。
- [x] **RE-PHASE-DISPATCH-GATE**：Docker Capstone 重讀 `0x1d80b` 第一個 phase loop，固定 0x50-byte record stride、`count=[0x53beb]`、raw gates `record+6==1`、`record+5&0x81==0`、`record+0x26==0`；新增 `fdother.FindNativePhaseDispatchCandidates` 與 short-input/opaque-byte regression。只回傳 raw unit/selector，不執行 `0x13a9f` 或命名 event effects。
- [x] **RE-INVENTORY-COMPACTION-AUDIT**：官方 IDA 9.4 decompiler 直接閉合 `0x1b8e7(int unit,int slot)`：`memmove(record+0x0a+2*slot, record+0x0c+2*slot, 2*(7-slot))`，再寫最後 cell flag `record+0x18=0x80`；新增 `battle.RemoveNativeInventorySlot`，保留 stale tail item byte，並覆蓋 slot0/slot2/slot7/short-input regression。先前「第三個 stack argument 未閉合」斷言已刪除。
- [x] **RE-UNIT-MODE-DISPATCH**：Docker Capstone 重讀共享 `0x13a9f`，固定 raw gate `record+5&5==0` 與 mode/argument reads `+0x34&0x0f`、`+0x35`、`+0x36`、`+0x3d`；新增 `fdother.PlanNativeUnitMode`，short/gate/masked-mode regression 通過。只保存 mode plan，不呼叫 `0x14ef0/0x14b78/...` 或命名效果；mode 6/8/其他仍保留未命名分支。
- [~] **RE-ITEM-EFFECT-DISPATCH**：Docker Capstone 固定 `0x20c6f` 的
  type→callee/argument 全 map；`NativeItemEffectRouteForType` 保留 raw
  topology。observed effect branches 5–24 的 typed post-confirm closures
  已完成；item selector UI、indexed presentations 與 normalized engine
  integration 仍是獨立缺口。
- [x] **RE-ITEM-TYPE67-MUTATION**：重讀 `0x22af6` 修正舊 adapter：
  marker 位於 target `record+a5`，不是 parallel `flags[]`。type6/7 用
  `+0x25/+0x26`，nonzero 時 base10→actual9 HP restore、清 record marker，
  再消耗來源 slot。新增 `ApplyNativeItemMarkerClearRestore` 與
  IDs196/197 regression；status/UI 名稱仍未知。
- [x] **RE-ITEM-WORD-DELTA**：官方 IDA 9.4 decompile `0x21082` 固定 `word(record[index]+a3) += low16(a2)`、16-bit wrap，隨後呼叫 `0x1b8e7(a1,a4)`；新增 `battle.ApplyNativeWordDeltaAndRemove`，explicit target/removal units 與 bounds/atomic regression 完成。欄位仍不命名，renderer/effect callback 不接。
- [x] **RE-RNG-GROWTH-MARKER**：官方 Capstone 固定 `0x4e893` 為 `rol16(state+0x9014,3)`；`0x22721/0x22866/0x22997` 的 `idiv 4` 取 EDX remainder，再 `+2` 寫 marker。新增 `fdother.NativeRNGStep/NativeRNGMarker` regression，刪除 quotient-based 誤讀；成長欄位與 FPU multiplier 仍未命名。
- [x] **RE-RAW-WORD-GROWTH-22721**：官方 IDA 9.4 固定 `+0x22` zero gate、RNG marker、`+0x48` 的 `trunc(word*0.15+1)` 與 `2*effective(+0x21)` raw accumulator；新增 `battle.ApplyNativeRawWordStep`，覆蓋 marked skip、RNG consumption、rounding、word update、score 與 preflight bounds。未命名成長效果，未接 presentation/tail。
- [x] **RE-RAW-WORD-GROWTH-22866**：Docker Capstone 固定 `0x22866` 與 `0x22721` 同構，僅 offsets 改為 marker `+0x23`、word `+0x4a`；`ApplyNativeRawWordStepAtOffsets` 共用實作並有 variant regression，未命名欄位。
- [x] **RE-RAW-PAIR-22997**：Docker Capstone 固定 `0x22997` marker
  `+0x24` zero gate、同 RNG marker、`+0x4c/+0x4e` 各 `+0x0f`、score
  `2*effective(+0x21)`；新增 `ApplyNativeRawPairStep` 覆蓋
  wrap/marked skip/preflight。後續 equipment cross-check 已將兩 words
  定案為 derived HIT/EV，type12 caller closure 見下方。
- [x] **RE-ITEM-22D1B-BRANCH**：Docker Capstone 重核撤回舊「兩次 RNG/
  固定10 damage」：gate RNG 成功後 `0x1c81f(base=10)` 消耗第二 RNG，
  實際減9 HP；第三 RNG 才寫 marker。type14/22 item caller 使用
  `+0x26/+0x27` 並保留來源；`ApplyNativeItemMarkerApplication`
  保存 class gate、三次 RNG 與 atomic mutation。status 名稱仍未知。
- [x] **RE-COMMAND23-COORD-WRITE**：官方 IDA 9.4 閉合 `0x22253` 尾端 `record[+0]=a13`、`record[+1]=a14`；`0x2218a` caller 的 `0xff/0xff` 是 pre-render pair，最後寫 cursor-derived pair。新增 `battle.SetNativeUnitCoordinateBytes` raw writer/regression；renderer/pathfinding 仍未接。
- [x] **RE-PERSISTENT-IDENTITY-LOOKUP-24BDE**：Docker Capstone 閉合 `0x24bde` 的 raw lookup：掃 caller-supplied persistent count（native capacity 32），stride `0x50`，只比較 record `+0x08` unsigned byte，命中即回 1、否則 0。新增 `battle.FindNativePersistentIdentity`，保留 first-index/read-only/bounds regression；不把 `+8` 泛化成 portrait、Fig 或 NPC identity。
- [~] **SYNC-PARTY-RAW-IDENTITY-GATE**：`PartyMember.native_identity`/`Unit.NativeIdentity` 已可選地攜帶 native persistent `+0x08`，`syncPartyFromBattle` 有 raw-key matching 與 unknown-key fail-closed regression；缺欄位時才保留 Fig projection。仍未完成全 roster/save/export 的 raw record 接線，故不宣稱 byte-identical。
- [x] **RE-PERSISTENT-COPY-MUTATION-11506**：Docker Capstone 閉合 `0x11506` 配對後 mutation core：runtime→persistent copy `0x50` bytes；清 persistent `+0x22..+0x27`；`+0x05 &= 1`；若結果非1，`+0x40 = +0x42`；固定 `+0x44 = +0x46`。新增 `battle.ApplyNativePersistentRecordCopy` read/write/bounds regression；`0x3453e` zero-identity gate 與 `0x1145a` tail 保留 caller-owned，未猜測性接入 sync runtime。
- [x] **RE-RAW-BYTE5-BIT0-3453E**：Docker Capstone 閉合 `0x3453e(index)`：回傳 selected record `+0x05 & 1`，不改寫資料。新增 `battle.NativeRecordByte5Bit0` mask/bounds regression；保持 raw predicate，不命名 acted/alive/active。
- [x] **RE-EQUIPMENT-RECALC-1145A**：Docker Capstone 閉合 `0x1145a(persistentIndex)` raw arithmetic，並由 `battle.ApplyNativeEquipmentRecalc` 保存 signed base words `+0x37/+0x39/+0x3e`、八格 `0x40` flag gate、`0x4e56c` row stride、四個 raw destinations 與 16-bit wrap。normalized `campaign.RecomputeEquipment` 仍是 projection-only；四個 row 欄位後由全表 cross-check 命名，fresh JOIN production 接線則由 `JOIN-1145A-EQUIPMENT-RECOMPUTATION` 完成。其餘 effect bytes 與完整 campaign byte identity仍未閉合。
- [x] **RE-EQUIPMENT-RAW-ADAPTER-1145A**：新增 `battle.ApplyNativeEquipmentRecalc`，依 raw `[flag,item]` 八格、bit `0x40` gate、`0x602ad+item*0x17` row 與 signed/wrapping word arithmetic 寫入四個 raw destinations；bounds preflight atomic、unequipped/missing-row regression 通過。此項落地時 row 欄位尚未命名；後續全表 cross-check 已在下一項閉合四個 equipment words，但仍未接完整 native campaign record。
- [x] **RE-EQUIPMENT-ROW-WORDS-CROSSCHECK**：215 個已知 runtime rows
  的 `+1/+3/+5/+7` little-endian words 已逐筆與 normalized `item.json`
  AP/HIT/DP/EV 比對，全部一致；Go regression 鎖定 fixture 間 contract，
  `0x1145a` 的 native 寫入順序可定案為 AP/DP/HIT/EV。其餘 effect bytes
  與 table 最終邊界不因本項而開放。
- [~] **RE-ITEM-EFFECT-ROW-4E56C**：已用 Docker Capstone 閉合
  `0x602ad + item*0x17`，並逐 byte 證實 EXE file view 從 `0x540ad`
  起、比 normalized `item.json` 的 `0x540ac` 起點偏移一 byte。新增
  `native_item_effect_rows.json` 保存 215 個已知 selector 的 raw prefix，
  exporter 與 Go loader regression 固定跨 normalized-row 邊界的 byte
  行為。實際 table 終點、未命名欄位與 normalized equipment 接線仍待證據，
  保持 fail-closed。
- [x] **RE-ITEM-EFFECT-211A4-ABI**：官方 IDA 9.4 閉合
  `0x211a4(actor,count,targetBytes,amount)`；item caller `0x20c6f` 直接傳
  `a3/a4` count/list 與 row `+0x0e` HP amount。type5 返回後經
  `0x1b8e7` 消耗來源 slot，type13 直接 cleanup 並保留來源。
  `ApplyNativeItemHPRestore` 保存 list 順序、sequential RNG、raw score、
  target/source atomic preflight 與 consumption 分歧。renderer/SFX/道具名稱
  仍 fail-closed。
- [x] **RE-ITEM-TWO-STAGE-TARGET-1BBDC**：官方 IDA/Capstone 閉合 case0 的兩次 `0x14818`：actor-origin first stage用 row `+0x10/+0x15`（type0x17 才 inner marker=1），`0x115b6` 確認後以 confirmed-origin row `+0x12/+0x15`、inner=0 產 final list，傳 `0x20c6f(actor,slot,count,list)`。新增 `NativeItemTargetPlanFromRow`／`NativeItemEffectTargets` 與 confirmed-candidate/short-row regression；runtime row producer、renderer與 gameplay 名稱仍 fail-closed。
- [x] **RE-ITEM-211A4-SHARED-CALLERS**：canonical Docker Capstone
  核對 direct callers `0x20ce0` item path 與 opaque selector `0x21`
  path `0x285ed`；後者以 caller-owned list、amount `0x320` 重用 helper，
  故 callee 本身不是 type5/13 專屬 routine。這不否定已由 dispatcher
  caller 閉合的 type5/13 HP restore/consumption；opaque caller 的上層語意
  與 renderer 仍 fail-closed。
- [x] **RE-ITEM-PRESENTATION-1E0DB**：官方 IDA 9.4 閉合 `0x1e0db(value,digitBias,target)` 的 camera gate、四位十進位格式化、queue position codes `2,7,12,17`、target index 與 digit-byte 寫入；`0x1e1dc` 保留 parallel raw queue writer。新增 `battle.AppendNativePresentationDigits` raw adapter/regression。這只關閉 presentation queue ABI，不命名 HP/MP/damage/heal，也不接 normalized item UI。
- [x] **RE-ITEM-ADJACENCY-GATE-1DEBE**：官方 IDA 9.4 閉合 `0x1debe(actor,x,y)` 的 active gate、曼哈頓相鄰一格與 equipped row `+0x0b<=1` 條件；此為 caller-specific precondition，不宣稱 `+0x0b` 是通用 weapon max range。
- [x] **RE-ITEM-PRESENTATION-1C4CC**：官方 IDA 9.4 pseudocode 閉合 `0x1c4cc/0x1c2da` caller ABI：兩者都接 `(opaque actor, raw subcommand, target count, target-byte list)`；`1c4cc` 依三張 33-byte frame table 逐 frame 做 456-stride target redraw、312×192 present、subcommand/frame SFX 分支與 BIOS tick，`1c2da` 以 native cycle/visual bank 做 target blit，再做五次 restore/present pair。這只關閉 presentation ordering/camera bounds/restore cadence，不命名 item effect、frame asset、SFX 或 target producer；`RE-ITEM-EFFECT-211A4` 仍保持 partial。
- [x] **RE-ITEM-20-24-PRESENTATION-1CD17**：官方 IDA 9.4 閉合
  type20/24 共用的 `0x1cd17`：30-byte remap table、固定十幀、每幀
  restore saved indexed buffer、camera-visible target redraw、
  `7-(frame%8)` raw blend argument、312×192 present、單 BIOS tick，
  再恢復原 buffer。此 helper 本身不做 gameplay mutation；其後獨立的
  row-selected command-damage loop 已由下方 caller closure 定案。
- [x] **RE-ITEM-COMPAT-1C1C3**：官方 IDA 9.4 閉合 item selector compatibility predicate：`0x1c1c3(actor,item)` 取 actor class 對應的六-byte raw table，逐一比較 item row `+0`；只保存 six-byte table／row-byte ABI，不命名 class 或 equipment 語意。
- [~] **UI-ITEM-8SLOT-SHELL**：舊 shell保留八個 raw holes並只支援↑↓，
  現已證實這不是 original parity。`0x1b9de` 依 signed flag非負 compact
  occupied prefix成兩欄四列；↑/↓ linear wrap、←/→±4，battle-use
  Enter拒絕effect type0。`0x184c0` 固定 label `(42+150*col,
  103+22*row)`、FDTXT `itemID+181`、selected/unselected raw color
  201/205、category icon59–61/equipped+3及stat icon64–67/41。
  `AdvanceNativeItemSelector`／`NativeItemSelectorCells` regression完成；
  GUI shell/Enter/indexed animation仍待依此改寫。
- [x] **RE-ITEM-SELECTOR-12FRAME-PANELS**：`0x17e0b` opening固定
  frames11→0，`0x1b932` closing固定0→11；每幀由 saved 64000-byte
  buffer重建。`0x18409→0x182ad/0x18312/0x1839b` 的三區已資料化：
  left `(src5,7,86×86)` frame6後left-clip16px；upper
  `(92,7,223×86)` frame3後up-clip16px、frame9消失；bottom
  `(5,94,310×102)` 每幀y+16、frame6消失。`NativeItemPanelSchedule`
  exact reverse/clip regression通過；indexed source已由下方完整
  compositor閉合，GUI animation adapter仍待。
- [x] **RE-ITEM-PANEL-SOURCES-17EEF-17FC0**：official IDA 9.4 確認
  `0x17eef` 以 `0x168b6(dst,320,5,7,5,5)` 建 `(5,7)` 的5×5框，
  unit record `+7` 選 DATO portrait貼 `(8,10)`；FDOTHER#5 directory
  offsets `+86/+90` 即 entries20/21，貼 `(92,7)`／`(5,94)`。
  `0x17fc0` 的2 bar、4 compared-number、8 raw-number、3 FDTXT與
  base/flag icons之 exact destination/record-offset schedule 已落入
  `NativeItemPanelBaseLayoutFor`／`NativeItemPanelDataPlanFor` regression。
  尚未證實的 raw offsets 不命名；下一步是 indexed renderer/Ebiten bridge。
- [x] **UI-ITEM-PANEL-BASE-INDEXED**：`RenderNativeItemPanelBaseResources`
  現從玩家 FDOTHER/DATO archive 原子化執行完整 `0x17eef`：corrected
  49-cell raw grid→DATO frame0→FDOTHER#5 entries20/21。新增
  `LMI1Entry.BlitOpaqueAt` 修正舊「`0x4e8af` index0 transparent」錯誤；
  synthetic overwrite/atomic failure與玩家資產 regression通過。
  `0x17fc0` dynamic overlays已由下方項目閉合；Ebiten bridge仍待。
- [x] **RE-ITEM-TEXT-HELPER-15F84**：official IDA重核
  `0x15f84/0x16559`，刪除 doc35 舊「`[0x53a85]` 是 CJK glyph容器」
  斷言。普通文字實際走
  `0x4ea2a([0x53a75]=FDOTHER#4,fontGlyph,...)`；`0x16559`只從目前
  DATO `[0x53a85]` 取 mouth frame重貼 portrait。item panel三段文字
  style固定 foreground205/shadow76/background0；含控制碼時仍須
  fail-closed。
- [x] **UI-ITEM-PANEL-DYNAMIC-17FC0**：新增
  `RenderNativeItemPanelData/Resources`，完整執行2 bar、4
  compared-number、8 raw-number、3 FDTXT與4 icon calls；精確保存
  `0x18795/0x17d6f` zero/nonzero bar、`0x1875d/0x187d6` padding/
  overflow/color、raw/four-mode/font三 codec與 record `+7` DATO selector。
  整張 atomic commit，control word fail-closed；synthetic與玩家 archive
  regression通過。新增可重現工具 `cmd/fd2-item-panel-oracle` 與
  [`item-panel-native-indexed.png`](../figures/item-panel-native-indexed.png)。
  後續 Ebiten bridge見下一項。
- [~] **UI-ITEM-PANEL-ROWS-EBITEN**：新增
  `RenderNativeItemPanelRows` 完成 `0x184c0` category/stat icons、
  FDTXT `itemID+181`、color201/205及stat number；oracle更新為有實際
  item rows。`NativeItemPanelRecordForUnit` 僅在 raw
  `+6/+8/+0x1f/+0x20`、DATO與8格inventory provenance齊全時建立
  80-byte輸入。
  `cmd/fd2` 已有 complete indexed image adapter、compact四方向input、
  opening11→0與closing0→11；缺證據/archives才用legacy shell。
  Docker/Xvfb玩家資產 regression走完整12+12幀。FDFIELD map roster的
  raw欄位已同步；JOIN `0x112a5` lower record與class-change只改
  `+0x20/+7`的lifecycle亦已接進30章scenario/persistence，正常ch01
  campaign asset可開啟原版面板。tracked IDs198/199/200（type8/9/10）
  及 IDs94/95/96（type17/18/19）已接完整Enter transaction：兩段
  self-target驗證、raw base AP/DP/DX或MaxHP/MaxMP/MV加值、compact消耗、
  必要的equipment recompute、`+5 bit7`及action結束；其他effect/target
  transaction仍fail-closed。
- [x] **RE-ITEM-COMPAT-TABLE-4E53E**：官方 IDA 9.4 閉合 `0x4e53e(class)=0x6188a+class*7`；新增 `battle.NativeClassCompatibilityRowOffset` 與 `NativeClassItemCompatible`，嚴格保留 row+0..+5 比對及 row+6 opaque、bounds/short-row regression，不接 normalized class/equipment。
- [x] **RE-RAW-HP-RESTORE-1C916**：新增
  `battle.ApplyNativeRawHPRestore`，保存 RNG step、amount arithmetic、
  current/max HP `+0x40/+0x42` clamp 與 `record+0x07`/class-derived
  score gate；shared primitive 本身不冒充 item，但 type5/13 caller 已另行
  閉合成 HP restore。UI/presentation 仍未完成。
- [x] **RE-RAW-MP-RESTORE-1C9DD**：新增
  `battle.ApplyNativeRawMPRestore` 保存 current/max MP `+0x44/+0x46`
  clamp 與無 class bonus 的 score gate；type11 caller 另由
  `ApplyNativeItemMPRestore` 保存 zero-max target 不消耗 RNG、list order
  與來源 slot consumption。IDs206/207 amounts=80/200；UI/presentation
  仍待完成。
- [x] **RE-ITEM-TYPE12-HIT-EV-22997**：結合 dispatcher tail、
  `0x22997` 與已定案的 derived words，閉合 type12：marker `+0x24`
  非零跳過且不耗 RNG；成功寫 `(rng%4)+2`、HIT/EV
  `+0x4c/+0x4e` 各加15，來源 slot 保留。新增
  `NativeItemHITEVStepRoute`／`ApplyNativeItemHITEVStep` 與 ID210
  fixture regression；marker UI 名稱仍未知。
- [x] **RE-ITEM-EFFECT-COMMAND-DAMAGE-20-21-24**：official IDA 9.4
  固定三 type 都將 row word 當 command ID，逐 target 呼
  `0x1c75e(target,commandID)`；20/24 用 `0x1cd17` 十幀 presentation，
  21 用 `0x2111a→0x1cac7`。dispatcher 不呼 `0x1ca89`、不移除來源。
  type20 IDs11/56/60→commands2/0/2，type21 IDs29/38/51/99→6/1/7/6，
  type24 ID79→command3；typed executor 保存 presentation 分歧與 transaction。
- [x] **RE-ITEM-TYPE23-RELOCATION**：official IDA/Capstone 閉合
  `0x1bbdc→0x2218a`：actor gate 是 raw identity `+8==24` 與 max MP
  `+0x46>=20`，不是舊稱 class/level；只取第一 target，以 command23
  cost20 對 current MP `+0x44` 做 16-bit subtract，按 target
  class/level 加 raw accumulator，再由兩次 `0x22253` 寫
  `0xff/0xff→destination cursor`。dispatcher 保留 item ID101；
  `NativeItemRelocationRoute`／executor 與 MP-wrap/preflight fixture 已補。
- [x] **RE-RELOCATION-MODE6-LEGALITY**：`0x115b6` mode6 Enter predicate
  已資料化：selected target 不算 occupant；其他同座標且 raw
  `+5 bit0==0` 的 record 阻擋。target selector 通常取 class `+0x20`，
  `+7==0x1c` 改1，class `0x13`／race `4,5` 改19；`0x4e555` 的
  29×20 row 在 resolved terrain index 必須為 literal20。新增 editable
  `native_movement_cost_rows.json`、strict loader、pure adapter 與 fixture
  regression；cursor UI／27 full-present + 18/24 direct-row renderer仍未接。
- [x] **RE-RAW-WORD-SUBTRACT-ADDRESS-CORRECTION**：Docker Capstone 證實
  word `+0x44` subtract 位於 `0x1ca89`，`0x1cac7` 是 allocation、
  `0x1cb94` drawing 與四輪 320×192 present helper。修正 adapter attribution
  並刪除 type21 MP-subtract 斷言；兩個地址不再混用。
- [x] **RE-RAW-FLAG-RESTORE-22AF6**：
  `ApplyNativeRawFlagRestore(records,targets,markerOffset,rng)` 現正確保存
  record-local marker read/clear、conditional HP restore、sequential RNG、
  target preflight 與 accumulator；錯誤 detached flags API 已刪除。
- [x] **RE-RAW-APPLICATION-22D1B**：`ApplyNativeRawApplication`／
  `ApplyNativeRawHPDamage` 保存 marker-zero/class gate、gate/damage/
  marker 三次 RNG、base10→actual9 HP subtract、marker `(rng%4)+2`
  與 accumulator；normalized command executor 亦修正為消耗三個 random
  draws。presentation/status 維持 fail-closed。
- [x] **RE-ITEM-TYPE15-16-AP-DP**：type15 marker `+0x23`／derived
  DP `+0x4a`，type16 marker `+0x22`／derived AP `+0x48`；成功增
  `trunc(current×0.15+1)`，marked target 不耗 RNG，來源保留。新增
  `NativeItemAPDPStepRoute`／`ApplyNativeItemAPDPStep` 與
  IDs213/214 fixture regression；marker UI 名稱仍未知。
- [x] **RE-ITEM-TYPE17-19-CAPACITY-MV**：type17/18 將 row amount20
  加到 max HP `+0x42`／max MP `+0x46`；type19 對 word `+0x3b`
  加1，但 caller 保存並恢復 `+0x3c` EXP，故 net effect 是 MV byte +1、
  EXP 不變。三條都由 `0x21082→0x1b8e7` 消耗來源；新增
  `NativeItemCapacityStepRoute`／`ApplyNativeItemCapacityStep` 與
  IDs94/95/96 fixture、atomic removal regression。
- [x] **RE-RAW-BUFFER-LATCH-24D22**：Docker Capstone 重讀 `0x24d22(arg)`：`arg!=0` 只把低 byte 寫入 global `0x51a10` 後返回；`arg==0` 配置 `latch*0x138` bytes，從 `0x53aff+(0xc0-latch)*0x138` 複製，接著以 `0xbf-latch` 向下做 `0x138` bytes row copy，最後再 copy 一列並經 `0x37416` free。沿 `0x11cac→0x11eee` case 23 的間接呼叫已證實 `0x24d22(0)` 是 312-byte staging 列旋轉消費端；`0x11EB0` 的共用尾端另固定將 312×192 bytes 複製到 `0xA0504`。tick gate 是 `[0x46c] != [0x539f8]`；`sub_24C1E` 在每個 draw 前先寫 stage 2..14，且三個 transient offset producer 都在返回前清零並不由此 handler 呼叫。故原版 RE gate 已閉合，下一步分類為依 SDD 接 production adapter 與 E1／E2，而不是再次反組譯入口 latch。
- [x] **RE-RAW-MARKER-REWRITE-24E80**：Docker Capstone 閉合 `0x24e80` 的 raw mutation：從 runtime slot `0x10` 到 caller count，若 record `+0x07==0x1f`，寫 `+0=0x10`、`+1=0x06`。新增 `battle.RewriteNativeMarker1F` 與 prefix/nonmatching/bounds regression；欄位仍不命名，不接 renderer 或 roster identity。
- [~] **RE-CHAPTER-CALLER-24838**：Docker Capstone 重讀唯一 `0x24bde` caller `0x24838`：先以 `0x24b14(0x64)` 分支，成功臂 `dialog #8→join(0x16)`；接著 `0x24bde(0x12)` 命中才走 `dialog #10→acting #0x48→0x32975(0x11)`，缺失時再依 global count `0x53bef<0x0f` 分成 `dialog #13→join(0x13)` 或 `dialog #12→0x32975(0x11)`，共同 sync/presentation 後才進後續 handler。只保存 raw call order；不把 `0x64`、`0x12`、`0x16/0x13` 命名成道具／角色／章節語意，runtime campaign binding 仍 fail-closed。
- [x] **RE-RAW-RECORD-BYTE5-32975**：Docker Capstone 閉合 `0x32975(index)`：直接覆寫 selected runtime record `index*0x50+0x05 = 1`，不保留其他 bit。新增 `battle.SetNativeRecordByte5One` overwrite/bounds regression；與 `SetNativeRecordBit7` 分離，不把 byte5 命名成 acted/turn/action。
- [x] **RE-COMMAND23-CALLER-SCOPE-CORRECTION**：Docker Capstone 重讀 `0x250cc→0x22253`，確認 `0x22253` 不是 command-23 專屬：chapter-ending/post handler 在 `0x1c2da` 後也以 unit `1`、pre-render `0xff/0xff`、record `+0/+1` 呼叫同一 indexed routine，隨後才進 `0x25089` cleanup 與 `0x2bce5` ending renderer。故 `SetNativeUnitCoordinateBytes` 僅是 shared raw writer；command-23 selector、ch29 ending layout、renderer/campaign semantics 仍分開且 fail-closed。
- [x] **CHAPTER-ENDING-250CC-BRANCH-AUDIT**：Docker Capstone 對齊 `0x25348` 分支確認：ending path 先送 FDOTHER frame `#0x0d/#0x0e/#0x0f`，呼 `0x1c2da`，再以 shared `0x22253` 寫 unit `1` 的 raw `+0/+1`，送 frame `#0x10`，最後 `0x25089→0x2bce5` 並 self-loop。這只固定 call order/終局邊界，不把 `0x24b14` 回傳或 frame IDs 命名成 town/shop/gameplay；一般戰後 flow 仍不得接此 self-loop。
- [~] **RE-INVENTORY-ITEM-GATE-24B14**：Docker Capstone 閉合 `0x24b14(item)`→`0x31860(unit,item)`→`0x1b8a6/0x1b722`：只掃 runtime unit `0..15`；每 unit 先取 bit7-clear count，再比對 raw slots `0..count-1` 的 item bytes，沒有額外 compact 驗證。成功回 native `1`，缺失回 `-1`。新增 `battle.FindNativeInventoryItemInUnit`／`FindNativeInventoryItem` 與 `NativeInventoryRecords` regression；campaign `partyHasItemID` 在完整 raw provenance 時已走同一 count-sized gate，缺資料才 fallback normalized。
- [x] **RE-NATIVE-RNG-LIFECYCLE-627B8**：Docker LE object/fixup audit
  確認 shared RNG word `0x627b8` 位於 initialized object 3，image初值
  `0x0000`；全EXE只有 `0x4e893` 自身load/store兩個reference，save/load
  與chapter handler都不讀寫。因此生命周期是process-wide、初值0、
  不進FD2.SAV。runtime新增獨立`uint16` state，不混用Go RNG。
- [x] **REMAKE-ITEM-HP-MP-TARGET-TRANSACTION**：Ebiten item Enter已將
  types5/13 HP與type11 MP接到兩階段`0x14818` target planner；確認目標後
  才atomic materialize/commit 0x50-byte records、依list order消耗native
  RNG、按type保留或compact移除來源，最後設raw `+5 bit7`並結束action。
  任一unit缺raw provenance即fail-closed；indexed effect presentation與
  types6/7/12/14–16/20–24 runtime接線仍待。
- [x] **RE-COMMAND-DAMAGE-RNG-CORRECTION**：Docker Capstone直讀
  `0x1c75e/0x1c81f`，確認`0x1c7ed`命中與`0x1c869`變異都呼
  `0x4e893`；miss耗1 step、hit耗2 steps。刪除舊
  `math/rand`替代，player command0與item types20/21/24改用同一
  process-wide uint16 state，並補state-sequence regression。
- [x] **REMAKE-ITEM-TARGETED-EFFECT-BATCH-2**：兩階段item target runtime
  已新增types6/7 marker-clear+conditional HP、type12 HIT/EV、
  types15/16 DP/AP、types14/22 marker application+damage，以及
  types20/21/24 command damage。raw transient、HP、derived words、
  retained/consumed inventory皆同步；indexed effect presentations仍待。
- [x] **REMAKE-ITEM-TYPE23-DESTINATION-CURSOR**：item101完成first-target後
  不立即改座標，而是進獨立destination cursor；逐格使用完整raw roster、
  `NativeTerrainMoveCodes`與29×20 cost rows執行literal target-code6的
  occupancy/terrain predicate，合法格才扣command23 MP、寫target raw
  `+0/+1`、保留來源並結束action。原版first-target與destination兩層
  Escape都直接回caller-owned item panel；舊destination→first-target
  行為已刪除。27 full-present + 18/24 direct-row indexed renderer仍待。
- [x] **REMAKE-ITEM-TARGET-SELECTOR-LIFECYCLE**：item target entry以
  `row[+0x12]+2` materialize global selector，first target field仍使用
  `row[+0x10]`／type23 inner marker／`row[+0x15]`。第一次`0x115b6`返回後
  reset所有cell byte+3並恢復selector1；final target list後再reset。
  type23 destination維持global selector1，只把literal code6傳給
  `0x115b6`，不再把兩個「6」混成同一狀態。focused Docker/Xvfb regression
  覆蓋兩層cancel、重新進入、成功commit與grid/selector reset。
- [x] **UI-ITEM-TARGET-INDEXED-FAIL-CLOSED**：物品第一階段 target field 與
  global selector 1..5 已由正式 `0x11CAC` 索引組合器消費；移除原生畫面
  失敗時仍會露出的綠色 item target、青色 relocation 與橘色 command 0
  半透明後備層。截圖旁車現可記錄 item targeting／raw ID／relocation modal，
  缺 HUD、LUT、range sprite 或 raw provenance 時不再以猜測介面冒充成功。
  物品效果 indexed presentation、disabled target 外觀、原版取消鍵與
  global selector 6 production owner 仍未知，不因本項升格。
- [x] **UI-SHOP-PURCHASE-CONFIRM-E1**：完整重讀`0x2f0b0`後保存四組
  六variant FDTXT表；購買問題展開`FFFC`商品名與`FFFA`十進位價格，
  並接原版`0x19953` Yes/No selected pulse。2026-07-28再以指令順序重核
  修正 framebuffer 斷言：`0x2f2a9`先完成`0x197e5`四幀choice closing，
  再由`0x19913..0x1994c`恢復保存的question region；`0x2f2d3`才在literal
  VGA`0xac44c`／`(12,157)`追加第三行並等待。不再錯誤保留steady
  Yes/No cells或使用第四個inward frame。production已接list close→confirmation
  open/steady/close→cancel或不足金wait→dialogue close→list reopen。
  真實FDOTHER/FDTXT/DATO regression與更正後indexed fixture已補。recipient
  selector與inventory-full後續已有E1 production實作；recipient input/scroll的DOSBox E2、
  no-recipient/full/success仍無DOSBox E2，不能由production接線推論原版操作
  驗收。下一步是optional-equip/success/debit lifecycle及同狀態E2。
- [x] **UI-SHOP-CONSUMABLE-RECIPIENT-E1**：`0x2f30a`分流已釘死：
  item type≥`0x20`走`0x2e6b8`兩欄六人名冊；type<`0x20`走相容性篩選後
  的`0x2e8cf→0x2ebe0`三列能力比較面板。新增strict consumable wrapper，
  裝備type誤用即fail-closed；真實shop entry16、FDICON與FDTXT regression/
  fixture已補。八格滿分支另保存`word_5265f={1,506,1,506,506,506}`、
  `unit[+7]+1`動態姓名與mode1 wait，未插入也未扣金。下一gate是裝備比較
  renderer、success/equip/debit與production lifecycle。
- [x] **UI-SHOP-EQUIPMENT-RECIPIENT-E1**：完整`0x2e8cf/0x2ebe0/
  0x2ef8f/0x2efb7`閉合type<`0x20`的filtered三列面板。candidate以raw
  base AP/DP/DX＋item `+1/+5/+3/+7`，只保留另一`type<=0x14`類別的
  已裝備貢獻；對derived AP/DP/HIT/EV選digit bank31/42/119
  （equal/increase/decrease）。shop entries16/18..22、FDICON、FDTXT姓名、
  三列geometry、6-open/5-close均已有strict compositor、真實資源regression
  與indexed fixture。下一gate縮為成功insert→optional equip→`0x2f4c6`
  →debit、production owner與E2。
- [~] **UI-SHOP-PURCHASE-SUCCESS/DEBIT**：`0x2f4c6`不可沿用church case4
  當通用動畫。shop variant1/resource12為entries23..27、`(169,45)`、
  每幀2 ticks後portrait mode0 restore；variant3/resource29為entry23、
  `(148,39)`、pre1/post8 ticks後restore；variant5/resource63為
  entries23..29、`(131,28)`、每幀2 ticks且不restore。三條strict plan/
  compositor與真實資源regression已補。DOSBox交易抓圖撤回舊fixture保留
  藍色問句框的錯誤：success必須從已關閉dialogue的bare shop framebuffer開始；
  修正後variant1前四個採樣依序對上source-built frame0/1/2/3。2026-07-29
  再重核`0x1956b/0x2d31b/0x2f4c6/0x16559`，確認裸畫面不得先覆蓋
  DATO第0幀；撤回該覆蓋後四組未遮罩整幀均為AE=0。
  caller順序固定insert→optional equip/recalc→success→`0x2d516` debit
  →product loop。Docker Capstone重讀`0x2d516..0x2d620`後，production已接
  先commit新balance、再用FDOTHER current resource entry2的6x99 strip做八位數
  downward odometer：每個不同digit同步減一、0→9、每值9個opaque 6x9 window、
  每phase `0x375b2(10)`。DOSBox內建320×200 capture進一步抓出roll destination
  是literal`0xa7a90=(16,98)`，不是stable gold的`(16,99)`；修正一列offset後，
  `1000→950`的21張debit樣本有16張分別與45個source phases整幀AE=0，另5張
  中斷在`0x2d620`逐列memmove的partial write。新增
  [`shop-purchase-debit-ch02-original-vs-remake.png`](../figures/shop-purchase-debit-ch02-original-vs-remake.png)
  五phase上下對照，再回六幀product list。
  wall-clock 60Hz會依elapsed取樣10ms phase，不保證每個source phase皆實體present；
  扣款的原子影格E2已關閉。購買成功動畫的25/26個DOSBox樣本也各自找到
  整幀AE=0來源影格；唯一第15張在`0x16886`效果寫入途中只差
  `(184,47)/(184,49)`兩點，下一張同一來源影格即AE=0，不列為原子畫面。
  成功動畫與扣款合成切片可升E2；完整商店仍因其他子面板與正常campaign/save
  路徑未閉合而維持部分完成。
- [~] **NATIVE-CURRENT-SNAPSHOT-ROSTER**：合法 IDA Pro 9.4 閉合
  `0x10010` 的 plaintext `0x0000` `0x8a3` FDFIELD 控制映像、
  `0x08a3` persistent roster、`0x12a3` runtime roster、`0x30a3`
  32-byte battle-local event state 與 `0x30c3` 18-byte header。撤回 header `+0` 是 persistent
  count 的錯誤工具斷言；正確為 `+0=turn counter`、`+1=runtime count`、
  `+9=persistent count`。`fdsave.InspectCurrentSnapshot` 已保存兩份 raw
  records、field control、battle-local event state，限制原生容量並有
  聚焦回歸；使用者
  checksum-valid 原版快照
  實測 persistent identities `[0,9,4,30]`。strict identity/class
  catalog 與單筆 `battle.Unit` materialization 已由下一項閉合。IDA 與
  Capstone 證實 `0x10010` 自己載資源、建 selector、恢復畫面；但
  `0x10616→0x4E031` 只複製 absolute `0x41A` word 到 `0x41C`。共享
  epilogue 返回後，main `0x25DCE` 才呼叫 `0x117E7` 控制器；舊「`0x4E031`
  是戰鬥驅動」及「不存在另一個 CONTINUE owner」說法已撤回。後續 IDA／Capstone 已證實
  `[0x53ad5]` 是 `malloc(0x20)` pointer，writer／reader 對稱保存，
  並由 indexed event paths 消費；`Raw30A3` 因而提升為
  `NativeEventState[32]`。`0x0000..0x08a2` 也由 FDFIELD 資源來源、
  對稱 copy 與 `0x1a813/0x13a44/0x10b4e` consumers 閉合為
  `NativeFieldControl[0x8a3]`。控制映像、runtime records/selectors、timing
  與 future-group constructor 已由下列具型別交易分別閉合；chapter0 未改寫
  live 排程已有嚴格 pending roster consumer 與原版快照測試。真正缺的是
  動態 turn-writer／group-formula 的通用 pending-group binding，以及整組 `battle.State` 到正式
  `Game`／controller 的原子 handoff，故正式 CONTINUE 仍維持失敗即關閉；本輪已
  新增重製端 `battle.State→Game` 原子發布契約與真實快照回歸，但標題呼叫端
  （caller）尚未提供帶符號 BIOS 計時值（signed BIOS tick），不能因此解除正式擁有者
  （owner）
  → `fd2_current_snapshot_ida.txt`、`fd2_current_event_state_ida.txt`、
  `fd2_current_field_control_ida.txt`
- [~] **NATIVE-CONTINUE-RUNTIME-PREFLIGHT**：合法 IDA Pro 9.4 與
  Capstone 閉合 `0x1035c` 清 selector cache count，接著
  `0x1036a..0x1039c` 依 current runtime record order 取每筆 `+7`
  呼叫 `0x11019`，並覆寫該筆 `+2`。撤回「CONTINUE 必須重播新章
  persistent→FDFIELD group construction order」與「存檔 `+2` 必須等於
  cache slot」兩個錯誤模型。`BuildContinueRuntimeInput` 現以明確
  chapter／field dimensions／FDICON group count 原子驗證 counts、
  FDFIELD unit capacity、camera-cursor identity、active record
  presentation 與 first-seen slots；所有 raw 區域深複製，不改
  `battle.State`。後續 IDA/Capstone 又證實標題 caller 的 range mode
  為開場 `0`／返回 `0x117E7` 控制器前 `1`，資料映像 gate B／anchor seed 均為
  `1`，且 anchor 只依已恢復 visible cursor 精確推進；這些值已收入
  `ContinueMapPresentation`。runtime unit、map timing seed adapter 與完整
  future-group constructor transaction 均已閉合；chapter0 靜態 live
  turn/event 已能只綁 groups3..7 共15筆；saved turn 只保留尚未掃描的
  selector0/1，已於上一輪尾端掃描的 selector2 不重綁。preflight
  仍保留動態 pending-group binding 與 `battle_controller_handoff` 兩個
  待 caller 接管的 owners，
  `ReadyForContinue=false`，故正式 CONTINUE 仍失敗即關閉
  → `fd2_continue_selector_rebuild_ida.txt`、
  `fd2_continue_map_presentation_ida.txt`、
  `fd2_continue_pending_schedule_ida.txt`
  - [~] **CONTINUE pending groups 靜態排程切片**：IDA/Capstone 直接證實
    `0x117E7→0x16F55→0x19DF7` 的存檔分支不先進 `0x1A30B`；後者只由
    `0x13565` 玩家階段收束門檻進入，依 raw selector1/0 掃目前回合後才
    `inc [0x53BEF]`，後段才掃 selector2；故 saved turn 只納入 selector0/1，
    selector2 已消費。新增
    `MaterializeNativeContinuePendingGroups`，只在 live `(turn,event_id)` 與
    scenario 完全相符時深複製 future rows／item table；chapter0 原版快照
    綁定 groups3..7 共15列，排除已出場1/2與未排程10/11。map0 舊31×24
    測試註解已依資產更正為24×24；map25/event61 另以真實資產固定
    selector1／slot／once-state12，live selector 不符即拒絕。動態
    event27/54/57、event47/49 formula、
    ch03 slot條件與多個 turn-byte writer 尚未資料化，故 owner 不移除
  - [x] **CONTINUE map timing seed**：IDA 完整 data xrefs 與 Capstone
    raw data 證實 cycles／terrain phase／兩組 binary latch 初值全零，
    terrain override 為 `-1`；唯 `[0x53C0F]` 由 main
    `0x25D83..0x25D8B` 擷取標題入口 signed BIOS low word。
    `ContinueRuntimeContext.TitleTimerTick` 與 `ContinueMapTimingSeed`
    現嚴格保存這個邊界；`MaterializeNativeContinueMapTiming` 原子安裝
    seed，map compositor 只在實際成功合成時取樣並同時發布 timing/pixels，
    已撤回每次 `Game.Update` 推進的錯誤排程。`0x10494`／`0x105ED`
    redraw 間的固定演出／delay 仍由正式 handoff 排程，但不再列為未知
    `map_timing` owner
    → `fd2_continue_map_timing_seed_ida.txt`
  - [x] **CONTINUE FDFIELD control typed view**：依 `0x53A55` 已證實
    layout，`ContinueFieldControlView` 原子拆出 raw header、16 筆
    turn events、16 筆 field events、16 筆 chest controls 與
    count-delimited 26-byte unit rows，並驗證 caller mutation 不會別名
    到輸出。IDA `0x10BCC` 的 exclusive compare 與 chapter0 current
    snapshot／FDFIELD resource1 全同前綴，另固定 raw `+2=30` 只解出
    30 列；資源第31列與容量尾端不冒充 live unit。控制資源不含
    `[0x53A51]` composition live byte `+3`；
    後者由資源 `3N` 另載並經 `0x4DBFC` 重設。下一步仍是把 typed
    control、saved runtime roster 與 chapter asset bundle 一次映射到
    `battle.State`；不可讓 `battle.Load` 的 serialized map provenance
    冒充 current runtime
    → `fd2_current_field_control_ida.txt`
  - [x] **CONTINUE live control mutation boundary**：IDA 完整 data xrefs
    與 Capstone 直接指令證實 `[0x53A55]` 會在戰鬥中被改寫：
    `0x19357` 更新 chest value，`0x34AB4/0x34AC5` 及多個 chapter
    handler 更新 turn event bytes。故 current snapshot 的
    `NativeFieldControl` 是唯一 live control 來源，不可由原始 FDFIELD
    resource 或 map JSON 覆寫；control rows 只供未來
    `0x10B4E→0x10C50` group append，現有單位仍由 saved runtime
    records 決定。同步刪除 doc26 把 party/FDFIELD constructors 拼成
    單一路徑及把 row bytes 錯套 runtime offsets 的舊表
    → `fd2_current_field_control_mutations_ida.txt`
  - [x] **CONTINUE live field boundary adapter**：
    `MaterializeNativeContinueFieldBoundary` 會從公開 input 重建 snapshot、
    重跑完整 preflight 並逐欄比對，故 marker 存在但內容被竄改也會拒絕；
    再配合相符的 chapter asset，檢查 dimensions／field-event topology 後
    原子安裝 exact control、turn/field/chest/future-unit rows、event
    state、raw round、view、HUD 與 opening range mode 0。輸入與輸出
    均不別名；拒絕路徑不改 state。它明確不碰現有 Units、timing、
    interactive mode 1 或正式 `Game`／`0x117E7` 控制器轉接。原本籠統的
    field-runtime owner 已由後續的 runtime-unit、map-timing 與
    future-group 具型別交易逐項關閉；正式 `Game`／controller handoff 現已有
    失敗即關閉發布契約（fail-closed publication contract），但標題呼叫端的帶符號
    BIOS 計時值與泛用待處理群組產生器（pending-group producer）仍是呼叫端擁有的
    邊界（caller-owned）。
  - [x] **CONTINUE saved runtime unit projection**：
    `MaterializeNativeContinueRuntimeUnits` 只接受已重驗 input 與逐筆相符的
    live field boundary；先在 detached roster 驗證 raw camp 0/1/2、
    class、active presentation 及 first-seen selector cache，再依 saved
    runtime record 順序原子替換 `State.Units`。它完整保存
    `+0/+1/+3/+4/+5/+6/+7/+8`、command mask、race/class、
    transient、`+34..+36`、`+42/+46`、八格 inventory 與 signed stats；
    saved `+2` 永不採信。撤回「所有 runtime +8 都是 identity」：只有
    native camp2 player record 依 persistent 契約提升
    `NativeIdentity`，其餘只保存 `NativeRecordByte8`。
    checksum-valid 原版 chapter0 current snapshot 的12筆 records 已在
    Docker 整合測試全數通過，前四名為索爾、悠妮、亞雷斯、蓋亞，
    enemy `+8=96` 不具 identity。adapter 不設定 timing、不 append future
    group、不切 interactive mode、不發布正式 `Game`／controller handoff；
    正式 CONTINUE 仍失敗即關閉。
    同輪同步遷移33份 map unit assets：scripted FDFIELD `b1` 現輸出
    `native_record_byte8`，不再輸出 `native_identity`；同步工具
    `--check` 全數零 pending，AI／item-panel raw record consumers 仍取得
    相同位元，但不再攜帶錯誤角色語意。
  - [~] **CONTINUE future group constructor inputs**：完整 Docker
    Capstone `0x10B4E..0x11018` 固定 group row order、6-byte position
    record、`b2→runtime +0x3D`、`b13..16→+0x1A..1D`、
    `b17..19→+0x34..36`、`b22..24→+0x31..33`；`b3/b20/b25`
    在 constructor 內無 reader。撤回 `b2/b3` 是 runtime race/class 的
    暗示，後者來自 b1-selected EXE tables。33份 map assets 已保存 exact
    position record、raw +3D、death triple、未讀 source bytes 與
    b1-selected constructor table record，loader 及 CONTINUE projection
    也保留 runtime raw 欄。`NativeFutureGroupPlacement` 已精確轉寫
    `0x145CD(0/1)` occupancy、raw `[0x53AFA]` gate、全圖 row-major
    Manhattan 與同距離後者勝出；舊半徑環狀 `nearestFree` 已明確降為
    legacy 呈現。`DecodeNativeFutureConstructorBase` 已轉寫兩條 table
    分支，33圖1,885筆 record 的 race/class/HP/MP 交叉驗證通過。
    table dump 已自帶 FD2.EXE size/MD5/SHA-256，sync 在使用前強制對照
    reference manifest；Docker 重生檔與版本化 JSON 逐位元組相同。
    official IDA 9.4 已將 `[0x53AFA]` 完整關閉為唯一 reader＋11組
    set1/call/reset0；25筆 handler spawn 與34筆 global event call 現保存
    source/via/`raw_placement_gate`，缺欄位的原版 handler fail-closed。
    `AppendGroupWithNativePlacement` 已把 handler Beat 接到 exact gate、position
    row、逐列 occupancy 與 group append，且 batch preflight 失敗不改 roster／
    units；runtime Xvfb regression 通過。global turn-event 的45筆 schedule 現
    產生46個 editable actions，全部保存 `native_event_id` 與逐 call
    source/via/gate；排程內六筆 gate=1 固定。正式 runner 對
    `runtime_append_groups` 逐 call 走 exact placement，缺 roster 失敗即關閉；
    未遷移情境仍是正規化相容（normalized compatibility），不是忠實度證據。
    產生器對同回合
    多 schedule／event 改綁拒絕合併，`--scenarios-only` 可避免舊生成拓撲覆蓋
    權威 `campaign_full.json`。ch02 turn3/event6 的版本化 action 已驗證六名
    group3 友軍採 gate=1 原始位置；錯誤不再呼叫回合完成回呼（continuation）。
    official IDA 9.4 已固定 `0x32999` 的四個 caller、本體不含 `0x1366A`、
    FDOTHER #9 固定12次 indexed compositing/presentation；global event1/2
    的 editable call metadata 另保存後續 ACTING(3/4) 與 call-site。handler
    的 `0x32999` adapter 已接 FDOTHER #9 的12次呈現、舊／新增槽位邊界、
    pass6/7/8 快照重建與 pass1 的 FDOTHER #95；完整預檢成功才發布 roster，
    每次 Draw 確認後才前進，再由下一個 beat 執行 caller ACTING。ch00 真實
    handler 回歸已驗證兩次各12幀後仍進入 battle_ch01、戰後、城鎮與整備。
    ch01 global event1/2 亦已由 battle-event runner 承接：turn3 建立14槽 frontier，
    turn4／5 分別 preflight group4／5、各12次呈現、ACTING(3／4)，event2 對話只在
    acting 完成後出現。缺 acting 資源的回歸固定 units／roster／selector cache／
    turn continuation 均不變；低階 `ExecuteActionChecked` 繼續拒絕無畫面擁有者的
    直接呼叫。`0x10C50` 的 table-base、八格 inventory 與 `0x1B750` 即時
    equipment／modifier 重算現已由合法 IDA＋Capstone 閉合，並原子接入
    future-group append；來源 roster 在失敗時不被改寫。正式 CONTINUE 的其他
    owner、完整 0x50-byte identity 與 DOSBox E2 仍維持 fail-closed
    → `fd2_future_group_constructor_capstone.txt`、
      `fd2_future_group_raw_gate_ida.txt`、
      `fd2_runtime_equipment_recalc_1b750_ida.txt`、`fd2_spawn_intro_32999_ida.md`
- [x] **NATIVE-CONTINUE-BATTLE-PUBLICATION-E1（2026-08-11）**：新增
  `MaterializeNativeContinueInteractiveBoundary` 與
  `ValidateNativeContinueBattleHandoff`，在所有 typed adapter 完成後才切換
  opening selector mode `0`→interactive mode `1`；新增
  `Game.publishNativeContinueBattle`，以複製 runner、清除殘留 UI／轉場／戰鬥
  暫存、同步保存鏡頭／游標後一次發布 `st`／`sc`／node，且不呼叫
  `resetBattle`／`Scenario.Setup`。真實 `FD2.SAV` chapter0 current-runtime
  快照的 `TestNativeContinueBattlePublicationFromRealCurrentSnapshot`、
  `TestMaterializeNativeContinueInteractiveBoundaryInstallsControllerMode` 及
  不完整 adapter 失敗即關閉回歸均在 Docker 通過。此項只閉合重製端 E1 publication
  契約（contract）；呼叫端帶符號 BIOS 計時值、泛用待處理群組寫入器／公式
  （pending-group writer／formula）、未修改一般玩家同狀態 E2 與戰後城鎮／商店／整備／存檔仍未完成。
- [x] **NATIVE-CONTINUE-TITLE-CALLER-E1（2026-08-11）**：`TitleMenuContinue` 現以
  `FD2_NATIVE_SAVE`／`FD2_NATIVE_TITLE_TICK` 為明確輸入，從可編輯戰役圖唯一解析
  `scenario.chapter` 相符的 battle node，在私有 state 完成 field/runtime/pending/
  timing/view/HUD adapters 後才呼叫 `Game.publishNativeContinueBattle`；缺存檔、signed
  BIOS tick、資產或對映含糊時停在標題並失敗即關閉。Docker／Xvfb 實際以一般按鍵
  Escape×8、Down×2、Enter 走到 chapter0 current-runtime battle，畫面與雜湊見
  [`native-continue-current-runtime-remake-e1.png`](../figures/native-continue-current-runtime-remake-e1.png)
  與 [`native-continue-current-runtime-remake-e1.json`](../data/ui-traces/native-continue-current-runtime-remake-e1.json)。
  這項只關閉重製端 E1 publication／輸入邊界；原版 BIOS 時鐘逐幀、action 選取擁有者、
  status/equipment panel、同狀態 E2、戰後 town/shop/preparation/save 全路徑與敵方 AI
  正式 caller／目標決策仍未完成。
- [x] **RE-CHAPTER-AUX-GRAPHICS-10652**：合法 IDA Pro 9.4 與 Docker
  Capstone 固定 `0x10652..0x1088d` 只有 CONTINUE、完整章節 loader、
  ch22 post 三個 caller。函式先釋放 `[0x53aff]/[0x53b03]`，再只對 raw
  chapter `9/17/21–25/27–29` 載入或展開特定 FDOTHER 輔助圖形；它不負責
  FDFIELD/FDSHAP/FDTXT/roster 的完整章節載入。撤回 exporter 的
  `load_ch_bg`，改為 `prepare_chapter_aux_graphics`，並以 compiler
  regression 固定尚無 runtime lowering 時必須失敗即關閉。全量重生另
  暴露 exporter 尚無法重建 ch14/ch16/ch25 後續人工閉合的 structured
  branches；在 generator 補齊前不得用全量輸出覆蓋那些 canonical assets
  → `fd2_chapter_aux_graphics_10652_ida.txt`
- [x] **HANDLER-LOADCH-OBSOLETE-NAME-GATE**：全量機械重生確認
  `0x25870→0x1088d` 現由 exporter 正確輸出 `loadch`，同步 ch29 raw／editable
  artifact 與統計。刪除 compiler 對舊 `load_ch_text` 名稱的相容降階；
  現在即使提供完整 binding 也會失敗即關閉，只有 `loadch` 可在 map、
  roster、slot count、story context 完整時降階。
- [~] **NATIVE-PERSISTENT-PARTY-MATERIALIZATION**：新增與參考 EXE
  SHA-256 綁定的可編輯 32 人 identity／class 0–28 catalog，以及嚴格
  `PersistentRecord→battle.Unit` 投影。保留 raw inventory flags、command
  mask、transient、race/class、base/effective stats，並依
  `0x10a77→0x11019` 投影 record `+7` 為 `MapSelectorKey`；不把它推導成
  portrait、Fig、identity、座標或章節。合法 IDA Pro 9.4 已證實
  class 顯示直接使用 `150+raw class`；固定雜湊 FDTXT 的 class 27 是
  兩個全形空格、class 28 是「？？？」，舊 `cls28`／`?`／「職業28」
  占位已移除。`FD2_NATIVE_SAVE_FIXTURE` Docker 整合測試已
  唯讀走完 current snapshot 四筆 record，實得索爾、悠妮、亞雷斯、蓋亞。
  下一步是閉合上述 current battle runtime；目前不接 CONTINUE。四槽 LOAD
  的 `0x2cad7` 戰間 restore owner 已由後續項目接入，尚缺一般玩家有效槽 E2。
- [x] **MAP26-EVENT62-DORMANT-TURN-ACTIVATION**：完整33圖×16列
  `native_turn_event_controls.json` 已由 FDFIELD raw resource 決定性重生；
  目錄同時保存固定 FD2.EXE 雜湊與 `0x2066E` 已證實的戰鬥回合初始值1，
  新戰鬥由此取得 event62 所需回合來源，CONTINUE 則保留快照即時值；
  執行期鎖定完整目錄 SHA-256 與 map 0–32 唯一集合，竄改控制列／寫入端／
  地圖身分的負向回歸均失敗即關閉；
  `0xff` 休眠列不再被 parser 丟棄，也不會進入第255回合排程。event62 的
  selector0／state17／slot0 event63 raw-camp0／native-round+1 已成可編輯規則，
  並接到向左一步第七拍的正式 selector0 owner；raw/typed 不一致與重複觸發
  在 mutation 前拒絕。同步更正 `0x35822` 來源
  `PUSH` 順序為 `(group,y,x)`，以 ch27 `[6,16,0]` 及 ch28 `[8,19,9]`
  非對稱回歸鎖定；event63 兩個 staging calls 已進固定雜湊的
  `event_id_groups.json`；同一擷取器亦保存 event64／66／68／70／72 的
  staging calls，但都仍不冒充一般 spawn。
- [x] **MAP26-EVENT63-DYNAMIC-RUNNER（重製端 E1）**：IDA／Capstone 直接
  順序已閉合 `sub_1A813(0)` 位於 `0x1D8BA` 敵軍 AI 前。ch27 的
  `native_turn_events` 將 live row0 精確交給獨立 raw camp0 owner，並把
  group1／2 從開局 initial roster 移到待增援 roster。runner 依
  `0x358C7..0x358E5` 執行 group1@(3,27)、group2@(15,27) 的 pan、native
  constructor、300ms、全 DAC 白閃、200ms、baseline restore、redraw，完成後
  才啟動 AI；兩批先在私人 state 完整預演，錯誤第二批不會部分發布第一批。
  gate A／anchor 的持續擁有者與 controller gate B=1 已接成
  `native_map_hud_inherited`；production regression 以明確帶 persistent raw
  `+0x42` 的凱麗 fixture 走 indexed DAC，不由 ch27 近似 HP 反推。缺完整 raw
  unit record 時仍只對既有 RGB 戰場使用數學上等價的全白覆蓋，不宣稱一般
  RGB palette adapter。
- [ ] **MAP26-EVENT63-E2-PLAYER-PATH**：從未修改 ch27 一般玩家路徑完成
  event62 向左一步、跨到下一 native round、觸發 event63，再以 DOSBox 同
  camera/roster/tick 逐幀比對兩次白閃與增援。ch27 戰前 view／selector0 已
  閉合並接線，persistent HUD 擁有者也已達 E1；本項剩餘該時點真實 roster
  raw record、CONTINUE 邊界及原版逐幀 oracle。完成前 event63 仍不可標成 E2。

## 2026-08-09：UI-03 完整 native frame 疊加修正

- [x] **UI-03-NATIVE-FRAME-OVERLAY**：Docker／Xvfb 真實抓圖發現 action overlay／native command grid
  被排除在完整 `drawNativeMapFrame` 之外，短地圖留下黑帶；現已在完整資源可用時允許
  modal 疊加，缺資源仍回退。新增 admission regression 與目前 source 的
  [完整指令環畫面](../figures/action-overlay-native-remake-fullframe.png)；只代表重製端
  E1，未提升為原版 DOSBox E2。

## 2026-08-09：戰後 runtime 邊界回歸

- [x] **CAMPAIGN-POSTBATTLE-RUNTIME-SEAM-E1**：新增 production
  `confirmBattleResult`，讓實際 `endTurn → aiStep → finishTurn → checkResult`
  的結果停在 battle node，Enter 才進入可編輯 postbattle；postbattle 先
  `sync_party`、再設章並經淡出進 town。Docker／Xvfb regression
  `TestEndTurnEnemyPhaseResultEntersPostbattleCutsceneThenTown` 已驗證持續隊伍快照、
  結果清除與 town 邊界，避免把戰後補給／商店入口誤接成下一戰。
- [ ] **CAMPAIGN-POSTBATTLE-E2-FULL-PATH**：仍需每一已綁定章節以未修改一般玩家
  路徑驗證 handler、戰後城鎮／商店／整備／存檔，以及原版與重製同狀態逐幀證據；本輪
  E1 fixture 不解除各章 fail-closed gate。

  **2026-08-27 範圍勘誤：** 此項是代表性證據提升，不再要求逐章執行，也不阻擋
  第29戰產品完成。第29戰已依使用者接受的99%玩家可見相似門檻關閉；其第三方存檔
  writer 來源與完整逐幀／逐音訊 E2 只保留為證據限制，不可據此重開已閉合 RE。

## 2026-08-10：戰場命中色盤脈衝與全畫面紅罩

- [x] **BATTLE-IMPACT-DAC-E0**：以固定雜湊的 `FD2.EXE`、合法 IDA Pro 9.4
  與 Docker Capstone 5.0.3 固定 `0x2939d` 的條件式 DAC 分支、`0x17aa9(1)`
  子幀等待，以及 `0x298a8..0x299b9` 的 20/40 毫秒脈衝；原始 local 欄位仍
  保留未知等級。證據：`docs/data/ida/fd2_battle_impact_pulse_ida.txt`。
- [x] **BATTLE-IMPACT-FULLSCREEN-TINT-CORRECTION-E1**：移除重製端沒有 raw
  provenance 支持的 RGBA 全畫面紅罩；只保留原版 impact 參考圖支持的守方紅
  剪影，攻方維持 FIGANI 原色，未把
  `AttackResult` 欄位猜成原始 DAC 輸出。
- [x] **BATTLE-IMPACT-DEFENDER-SILHOUETTE-E1**：以
  `orig_05_attack_03_impact.png` 的可見狀態修正攻方不應對稱染紅；這只代表
  impact 參考圖的重製端畫面近似，不提升為所有攻擊／技能的原版規則。
- [x] **BATTLE-IMPACT-HP-BOUNDARY-E1（2026-08-10）**：固定命中 fixture 證實
  原版在 impact 開始時立即顯示 post-hit HP；重製端撤回未有 raw provenance
  依據的 8 tick 中間值，並以原版擷取 RGB `(190,0,0)` 校正守方剪影近似色。
  [`battle-impact-compare-20260810.png`](../figures/battle-impact-compare-20260810.png)
  的正規化逐 RGB 差異仍為 3933 像素，不能提升為完整戰鬥 UI E2。
- [x] **BATTLE-DEFENDER-FIGANI-SCHEDULE-E1（2026-08-10）**：守方待機幀撤回
  固定 `(prog/6)` 猜測，改依各資源 descriptor `+6`／`FD2_BATTLE_FPT` 的純排程
  橋選幀；攻守任一延遲表不完整即失敗即關閉。這只收緊 renderer 輸入，不證實
  命中、傷害或 DAC 語意。
- [ ] **BATTLE-IMPACT-DAC-RUNTIME-BRIDGE**：仍需建立帶 frame `+4`、傷害步進、
  `0x29f72` 原始輸出與 palette baseline 的正式轉接器，並以未修改一般玩家
  同狀態逐幀比較；完成前不得宣稱攻擊閃紅或整體戰場 UI E2。
