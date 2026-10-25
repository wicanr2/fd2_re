# 91 — FD2 remake 目前未完成項

未完成項的唯一權威是 GitHub Issues（[`wicanr2/fd2_re`](https://github.com/wicanr2/fd2_re/issues?q=is%3Aissue+label%3Aworklist)，
標籤 `worklist`）。[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是
`tools/fd2_worklist_issues.py pull` 拉下來的快照；下面那一節由
[`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 從快照產生，**不要手改**，
改了下一次 pull／render 就會被蓋掉。

每一條掛一個 `verify`：跑起來為真代表這一條仍然未完成，為假就是東西做好了而條目
沒改。這樣「動手前先確認那條還成不成立」不必靠人記得。

2026-10-15 使用者要求重製完成與HD優先。GitHub Issues已標示P0～P3，
暫停項保持開啟並加上`狀態:暫停`；開啟不代表本輪應執行。
總項只彙整子項，不以Issue數量換算完成度。實際程式、章台帳與證據分層見[58](58-fd2-exe-re-coverage.md)。

| 優先級 | 執行順序與範圍 |
|---|---|
| P0 | [#203](https://github.com/wicanr2/fd2_re/issues/203)工具鏈及[#205](https://github.com/wicanr2/fd2_re/issues/205)IRQ7鏈已限定驗收，接續[#142](https://github.com/wicanr2/fd2_re/issues/142)第24章 → [#199](https://github.com/wicanr2/fd2_re/issues/199)第25～30章與結局 → [#200](https://github.com/wicanr2/fd2_re/issues/200)補第1～3章。事件[#33](https://github.com/wicanr2/fd2_re/issues/33)隨相關章收斂；[#14](https://github.com/wicanr2/fd2_re/issues/14)為總項 |
| P1 | 與主線交錯處理[#201](https://github.com/wicanr2/fd2_re/issues/201)現代素材需求與[#202](https://github.com/wicanr2/fd2_re/issues/202)正常路徑，總項[#4](https://github.com/wicanr2/fd2_re/issues/4)。[#204](https://github.com/wicanr2/fd2_re/issues/204)先確認高解析範圍。HUD #41、戰鬥 #12、身份 #6、音訊 #15併入代表性驗收；造成實際玩家阻擋時優先修正 |
| P2 | 平台與交付#7／#8／#9，最後處理編輯器#10／#11。原先最後處理的前置排除暫停研究與非阻擋診斷，避免永久等待 |
| P3 | #52／#197第12章單像素目標追蹤維持暫停；#198全檔C匹配依使用者要求暫停。保留證據與未完成狀態，不作P0／P1／P2前置 |

```sh
tools/fd2_worklist_issues.py pull   # 從 GitHub 拉下開著的條目，重寫快照（主機，要 gh 登入）
tools/fd2_worklist.py verify        # 逐條檢查，發現訊號消失的條目就以 exit 1 收場
tools/fd2_worklist.py render        # 重寫下面的產生區塊
python3 -m unittest discover -s tools -p 'test_fd2_worklist.py'
```

條目做完就在 GitHub 關掉那個 issue（`tools/fd2_worklist_issues.py close <id> <留言>`），不是在這裡打勾，也不是改快照。

其他入口：

- 每一題的 RE／資料／正式執行期／E2 分層現況：[`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)
- 介面覆蓋率與尚未關閉的關卡：[`57-ui-evidence-matrix.md`](57-ui-evidence-matrix.md)
- 各輪的工作記錄、勘誤與位址出處：[`91-worklist-history.md`](91-worklist-history.md)
- 已閉合位址的「不要重做」索引在 `58`；重開條件是輸入雜湊不同、原始指令或跳表
  直接反證、同狀態執行結果矛盾，或主證據缺少它聲稱具備的 writer／consumer。

<!-- BEGIN fd2_worklist.py render；不要手改這一段 -->

共 24 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

## re — 原版證據還沒閉合

### P1 混合背景反擊：主動header零、反擊非零的caller圖層待證

`native-physical-mixed-counter-bg` · RE待解 · [#265](https://github.com/wicanr2/fd2_re/issues/265) · 仍未完成 · 自承還在 remake/cmd/fd2/native_physical_body.go

原始拒收保留：第九章章前HP+1000建構槽的貝克威為class4弓兵，卻裝著type1短劍item0、射程1。相鄰受擊因此嘗試反擊，遇到主動header零／counter非零的caller BG守衛。這份建構槽不能證明未修改原版可達性。

後續已正常購買並立即裝備type5黑暗弓，金幣扣1800、射程2～4；一次普通END及射程4炮擊、返回／F2／邊界保存與新程序LOAD通過，貝克威21幀已正式登錄。此正常購弓來源不再阻塞本輪HD。購物另發現的原生ignored格毀損獨立列#266，已修正並實機驗證。

本項保留的一般能力限制：prepareNativePhysicalBodyResources仍拒收attack.HeaderByte1==0且counter.HeaderByte1!=0。未證明所有這類組合均不可達，也未補齊原版caller保存的BG與必要預載；缺件仍在結算前拒收，不刪守衛或把反擊關掉。原版oracle未知與已完成的正常貝克威流程分開。

原背景主證據docs/data/ida/fd2_physical_background_selection_20261004.json；前批失敗沿主收據early_party_battle_followup_20261020，購弓修正與普通炮擊沿bekwei_battle_followup_20261020。既有已閉合位址不重做，後續只有實際合法caller需要時才開窄切片。

怎樣算做完：確認該組合的原生裝備／射程與可達範圍，補齊固定雜湊、原始位址及caller／consumer證據；依READY規格實作必要背景預載或修正錯誤測試來源，同狀態與普通玩家抽樣通過，缺件仍在結算前拒收。不能只刪守衛或改成無反擊。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

## data — 可編輯資料還沒就緒

### P1 HD／現代主題總項：素材覆蓋、正常路徑與交付驗收

`modern-theme-prototype-status` · 工作 · [#4](https://github.com/wicanr2/fd2_re/issues/4) · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

排程：P1，與P0戰役主線交錯推進。沿用61已核准的modern-handpainted-a手繪風格、靜態閉嘴與原子預檢契約。

目前是原生尺寸重繪候選，不宣稱已完成高解析呈現。清冊、實體檔案及consumer核對結果只見 [58](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/58-fd2-exe-re-coverage.md) 與 [61](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/61-modern-theme-runtime-spec.md)。selector39／57是純目的地重映射，不列缺人物圖。

拆成可驗收子項：全玩家路徑素材覆蓋與缺件報告；早／中／晚期正常操作及戰場返回的現代呈現；高解析呈現範圍確認與可丟棄原型。高解析倍率、幾何與素材資料格式未定案前不改production；不把目前24×24／80×80圖直接放大當作新增HD細節。

公開庫只存引擎、清冊、證據及已獲准總覽；素材沿allow-list同步唯一私人保存庫，維持private。

執行子項：#201 素材需求、#202 正常玩家驗收、#204 高解析範圍／原型；#203 提供正式回歸環境。

怎樣算做完：素材覆蓋及正常路徑子項完成；高解析範圍經使用者確認並依其規格驗收；catalog／loader／同步工具的正式狀態一致；私人包與封包驗收通過後才將總項結案。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 工作 · [#6](https://github.com/wicanr2/fd2_re/issues/6) · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

排程：P1，併入代表性玩家路徑與素材驗收；保持原驗收條件，不以綠色單元測試代替人工聽辨或正常路徑。

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

### P0 補齊剩餘回合事件：增援、AI、鏡頭、演出與對白

`turn-event-handlers-partial-transcription` · 缺陷 · [#33](https://github.com/wicanr2/fd2_re/issues/33) · 仍未完成 · 要人判

2026-10-16目前：第26章event57九列已完整轉寫並消費，操作為依當前round生成group、pan(9,0)、delay400。沿固定EXE／既有IDA9.4 sub_354DD 33 bytes及353C4..353DA共用尾段建立READY契約，再實作與同狀態重播，沒有新matching編譯。

原版r4正常觀測T2／4／6／8／10／12／15／16／17九波，之後才一次清敵；完整Game 913筆AI零順序分歧、行為／節點／交易通過，22987 bytes整檔SAV嚴格相同。按111章前作弊政策列PLAYER-E2，畫面參考，不證明自然戰鬥難度。正式收據docs/data/ui-traces/parity-ch26.json；主契約docs/data/ida/fd2_turn_event57_20261016.json。

全域tools/sync_native_turn_events.py --check通過，尚未轉寫13筆；本項保持開啟。現況數量與分布唯一入口為58及可重跑工具，不由單章passed推定其他事件完成。後續隨相應正常章路徑補足，不能以清敵略過未驗事件。

以下保留較早歷史，文中22／23筆與第26章待驗均已被上述新收據取代。

2026-10-16目前：第25章event56正式完整流程已按111明示保存政策驗收，240筆AI零分歧。#212不再阻塞；下一章26的event57待正常原版收據後接完整consumer。全域剩餘數量仍依58及sync工具，不從本次驗收政策推定其他事件完成。下列為歷史。

2026-10-15最新：第25章T6 event56已按READY契約接上完整pan(6,40)、group1、ACTING74、text5及134E4重設pose。正常原版r4及完整Game重播確實消費，240筆AI順序零分歧；全域--check通過，未轉寫22筆。第25章完整保存尚因#212未過，不稱本章PLAYER-E2；下一章26事件待本章驗收後接續。主契約docs/data/ida/fd2_turn_event56_20261015.json。下列記錄保留為歷史。

2026-10-15 更新：第24章event54完整九操作已轉寫，四筆T2／4／7／10控制列同步到正式及canonical劇本。來源為既有IDA9.4與固定EXE的125 bytes核對；補足四角鏡頭與等待，規格先達READY再實作。主契約docs/data/ida/fd2_turn_event54_20261015.json。既有短鏈13張完整RGB、行為／節點／完整SAV回歸通過，本輪T2／4／7／10至T11、戰後、記錄／保存及退出列表已通過流程三閘門，兩側完整SAV相同；280筆AI入口零順序分岔。#207／#208已限定修正並關閉，正式收據docs/data/ui-traces/parity-ch24.json。下一個事件為第25章T6 event56，仍須取得該章正常原版收據後接完整劇本。依使用者最新定案，畫面只作參考，不因逐幀差異阻塞流程；未轉寫事件仍須逐章補足，不能用清敵掩蓋。

排程：P0，與相關章的正常玩家路徑一起處理。2026-10-15 容器執行全域 sync --check 通過，但仍有未轉寫事件；唯一數量與分布見 [58](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/58-fd2-exe-re-coverage.md)。

既有完整轉寫與已通過章保持。優先補第24～26章會被正常路徑消費的事件，再補第1～3章及第21／22章剩餘事件。每個handler須保留完整動作與參數、共享尾段、READY審查及正式consumer，不能只把增援資料視為完整事件。

FDFIELD回合事件由完整具型別處理器降成正式劇本，保留增援、AI模式、鏡頭、演出與對白；不得只保留spawn。原始表為0x51B91，工具為tools/extract_native_death_events.py與tools/sync_native_turn_events.py。

2026-10-15較早快照：當時全域--check通過，仍有23筆未轉寫；現況見本文首段及58。原先「其餘57筆、只有第四／五章」是歷史快照，不能作目前待辦。現已轉寫的章由工具直接列出；其餘依111正常章收據逐章處理，不先批次替換。

本輪補第十七章event40：group2後鏡頭(17,37)與text1三句。固定EXE／IDA9.4全handler與共享尾段經READY審查，再由正式lower同步到scenarios及canonical。正常Game確實消費原先缺失三句，124筆AI零順序分歧，98張畫面與酒店SAV保持。#195／#196工具補正後同來源四gate皆通過。既有第17章PLAYER-E2保持，未重新執行原版或新增章收據。

主契約：docs/data/ida/fd2_turn_event40_20261006.json；唯一分層現況見58。剩餘未轉寫列分布於第1／2／3／21／22／25／26章，本題維持開啟。

怎樣算做完：tools/sync_native_turn_events.py --check（不帶 --chapters）通過，報告裡「尚未轉寫的回合事件」為 0，且每一章的擴充都有該章 111 收據。

### P1 HD素材覆蓋：由正式玩家consumer列出缺件並補齊

`modern-theme-player-asset-coverage` · 工作 · [#201](https://github.com/wicanr2/fd2_re/issues/201) · 仍未完成 · 要人判

全部HD仍未完成，本項保持開啟。使用者2026-10-21定案：先確認signal: killed，完成全部HD串接後，再集中人工確認與回報，不逐人物插入人工進軍。原驗收條件保持。唯一覆蓋數量及缺口依[58現況矩陣](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/58-fd2-exe-re-coverage.md)，契約見61。

普通2／3／7／8人物已沿同次typed producer接入，保持數值、亂數、聲音與保存格式；四者完整Game.Draw／F2、MP一次／HP6、13、5、16段與一次續行通過，來源重組1891張GPU畫布相符。正常2／8施法、取消／返回、F2與章節邊界冷讀在暫停前已通過；3／7及集中人工確認待全部HD串接後執行。FDOTHER特效仍有像素來源，沒有把fixture升格為正常玩家或新原版parity。

全套Go原失敗已由同時刻核心MEMCG事件及4 GB受控重跑oom_kill=1確認。8 GB比較輪無OOM但碰650秒總時限，保留非PASS；標準8 GB／4 CPU、1200秒的主程式重跑通過，峰值約7.04 GiB，其他19套件沿比較輪通過。這是分輪彙整，沒有冒稱單一完整命令全部成功；專用GPU略過與獨立執行另列。標準測試入口補足總時限、外層終止、真實失敗碼及Docker內基線分析。

BattleFig110的330／331／332三組原24幀已正式登錄。物理16張及施術21張最終GPU取樣核心符合，包含全部八張施術來源；332:0..7原始PNG／Mask／尺寸逐項等於331:0..7，來源誤判已在spec追加勘誤。六姿勢共用、12個高覆蓋裁切像素與原RGBA保持均明示；完整及runtime-only載入／缺件與F2原子性通過。完整圖板與六張來源沿清冊白名單存入私人庫6cb283ebd99d84471e1ca57e4210cd95a9e1d164，實際遠端HEAD已核相同，維持private。

其餘人物、指令6及其他未列指令／效果、地形／前景、背景、完整介面與平台效能仍待。來源集中於[HD主收據](https://github.com/wicanr2/fd2_re/blob/main/docs/data/fd2-modern-hd-battlefield-20261019.json)的ordinary_remaining_hd_followup_20261021、winged110_battle_followup_20261021及hd_integration_order_20261021。沒有新Release或完整HD完成聲明。

怎樣算做完：受版控需求盤點可重跑，列出來源與consumer；既有已核准現代替換範圍零非預期缺件，其他畫面各有明示分類與決策；缺件拒收與原子切換回歸通過，私人包雜湊及遠端HEAD可核對。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

### P2 校譯新恢復終局六段的簡中、英文與日文候選

`ending-six-restored-lines-locale-review` · 工作 · [#239](https://github.com/wicanr2/fd2_re/issues/239) · 仍未完成 · 要人判

FDTXT031 strings40..45原版尾聲先前未納入可編輯劇本，完整60句索引已補回。三語新候選依既有遷移工具加入，明示machine_draft，六個canonical line_id及來源SHA凍結於supplements；不冒稱人工審校。原版繁中終局以分離原始字模播放，不由此推定三語已驗收。

怎樣算做完：由使用者或譯者核對六段來源、專名與語氣後逐筆reviewed，保存校譯紀錄並更新受控六句待審範圍；其他已審校譯文保持。

證據：`docs/data/localization/ending-story-supplements.json`

## runtime — 還沒接進正式執行期

### P1 補齊剩餘 HUD 重繪：玩家確認、物理返回與劇情關框

`hud-anchor-attack-and-message-redraws` · 缺陷 · [#41](https://github.com/wicanr2/fd2_re/issues/41) · 仍未完成 · 自承還在 docs/knowledge-base/56-fd2-remake-sdd.md

排程：P1；若在P0章路徑造成差異，當場提升為該章阻擋。

死亡最後steady的0x1DEAE及已證實196CB戰場END／獎勵closing已接線並有有限回歸。剩餘為玩家指令確認的0x1D3FF、普通物理0x1563B／0x1565C與尚未覆蓋的劇情／升級closing。0x15510屬簡略演出，不能接到普通full攻擊。逐caller證據與目前consumer以 [HUD主契約](https://github.com/wicanr2/fd2_re/blob/main/docs/data/ida/fd2_hud_redraw_20261006.json) 及56為準。

利用後續正常章收據確認，不重跑已passed章，不重解已閉合caller，也不以額外評估點消除未知差異。

怎樣算做完：以原版 eip-trace（或同狀態擷取）證明這三類重繪發生的時機與當時的閘門／可見游標，接到同一個入口。不為了驗收重跑已 passed 的章（使用者 2026-09-18 定案）：以之後正常排程的新章收據順帶確認即可，沒有出現對應時點就在 56 §#40 記錄仍未被覆蓋。

### P1 HD高解析呈現：原生面板字型、介面、戰場與平台驗收

`modern-theme-highres-prototype` · 工作 · [#204](https://github.com/wicanr2/fd2_re/issues/204) · 仍未完成 · 要人判

戰後物品／金錢成功訊息已保留同次map23背景。訊息沿既有密封場景保存before-acted的typed input／work／VGA／palette與stage，F2驗來源及原生重組後才發布。開框、等待ACK、關框、文字／480頭像裁切與原交易保持；關框後才加金幣，返回時恢復當下人物灰化。

兩包各兩類、28張主要Game.Draw邊界與28次F2，完整收據與12張PNG相同；獨立CPU世界、框外非空背景、像素控制組與一次續行通過。八種密封來源破壞、錯素材冷暖拒收、一般地圖26張／26次F2、HUD及滿背包原交易回歸通過。源頭缺口已有修改前反例，沿既有RE，無新素材／清冊／私人提交；字串盤點及109項處置保持。數量及範圍依58、61與HD主收據reward_message_stage_hd_followup_20261025，私人測試觸及6GiB，memory.events.max=228、OOM為零，記憶體事件原樣保存。

實圖測具曾混用索爾地圖key0與法術測具BattleFig9，已只修測具為DATO0並加入一致性斷言，重跑受影響流程。480圖源與1254母稿推導相符，沒有80像素放大回退。這是受控RUNTIME-E1，像素控制組是同一remake，未新增FD2.EXE對拍、正常通關或平台聲明。

先前移位雙段、指令10～12及前奏、回復13～16與玩家9等切片保持。滿背包問句、其他提示／完整HUD、其餘地形、HD2、介面獨立控制、單列浮出工具列／加速／海報，以及全部HD後集中正常玩家／人工／平台仍待。蓋亞手腕與凱麗右腳沿原工單，完整HD總項保持開啟。

怎樣算做完：既有HD頭像／黑體／穩定頁／整備問句／金幣／角色物品轉場及首章角色狀態覆蓋已抽驗。接續其他設施轉場、成功演出、圖示與未列畫面，再完成角色／地形真正高解析素材、戰場交界及跨平台HD效能；完整HD仍未完成。 工具列須分開控制操作介面原版／HD與人物主題原版／HD，原版／HD／HD2三種人物主題與兩種介面共六組合互不改寫；介面內人物圖示、頭像與戰鬥動畫跟主題切換，HD介面樣式先經原型確認。 新增使用者已選的HD2斜角站姿／HD地形主題，游標下敵人的地圖與HUD人物圖示一致；單列頂端浮出、自動隱藏工具列提供獨立海報控制。第一組左索爾＋悠妮朝右、右空魔神＋四天王；第二組依使用者給定原稿重繪並保留來源。 海報內緣漸層及淡入可由單列工具列調整，切海報與漸層不得影響戰場／HUD；保存已選概念至GitHub作HD2設計基準。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

### P1 大型戰鬥動畫HD：人物、效果與正常玩家呈現

`modern-battle-animation-hd` · 工作 · [#263](https://github.com/wicanr2/fd2_re/issues/263) · 仍未完成 · 要人判

戰後物品／金錢成功訊息已保留同次map23背景。訊息沿既有密封場景保存before-acted的typed input／work／VGA／palette與stage，F2驗來源及原生重組後才發布。開框、等待ACK、關框、文字／480頭像裁切與原交易保持；關框後才加金幣，返回時恢復當下人物灰化。

兩包各兩類、28張主要Game.Draw邊界與28次F2，完整收據與12張PNG相同；獨立CPU世界、框外非空背景、像素控制組與一次續行通過。八種密封來源破壞、錯素材冷暖拒收、一般地圖26張／26次F2、HUD及滿背包原交易回歸通過。源頭缺口已有修改前反例，沿既有RE，無新素材／清冊／私人提交；字串盤點及109項處置保持。數量及範圍依58、61與HD主收據reward_message_stage_hd_followup_20261025，私人測試觸及6GiB，memory.events.max=228、OOM為零，記憶體事件原樣保存。

實圖測具曾混用索爾地圖key0與法術測具BattleFig9，已只修測具為DATO0並加入一致性斷言，重跑受影響流程。480圖源與1254母稿推導相符，沒有80像素放大回退。這是受控RUNTIME-E1，像素控制組是同一remake，未新增FD2.EXE對拍、正常通關或平台聲明。

先前移位雙段、指令10～12及前奏、回復13～16與玩家9等切片保持。滿背包問句、其他提示／完整HUD、其餘地形、HD2、介面獨立控制、單列浮出工具列／加速／海報，以及全部HD後集中正常玩家／人工／平台仍待。蓋亞手腕與凱麗右腳沿原工單，完整HD總項保持開啟。

怎樣算做完：建立由正式consumer到FIGANI人物及共同效果的可重跑覆蓋清冊；完成可達戰鬥動畫HD素材、原始位置／層序／節拍／命中與音效標記的正式呈現，缺件明示；正常玩家抽驗攻擊、受擊／反擊、法術／效果、死亡與戰場返回，F2、保存冷讀與代表性效能通過；已登錄素材同步私人庫，保留原版像素模式。資料測試及離線PNG不能取代正式GUI。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

排程：P1，併入代表性玩家路徑與素材驗收；保持原驗收條件，不以綠色單元測試代替人工聽辨或正常路徑。

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 工作 · [#15](https://github.com/wicanr2/fd2_re/issues/15) · 仍未完成 · 要人判

排程：P1，併入代表性玩家路徑與素材驗收；保持原驗收條件，不以綠色單元測試代替人工聽辨或正常路徑。

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

### P3 第12章單像素差異：保留證據，非章驗收阻擋

`ch12-hud-terrain-single-pixel` · 缺陷 · [#52](https://github.com/wicanr2/fd2_re/issues/52) · 仍未完成 · 要人判

排程：P3，保留精確視覺差異；原先第12章目標追蹤仍暫停。不影響已通過章台帳，也不作其他章、HD或發行預覽的前置條件。

正式native六回合seq3146的(11,85)原版118、Game116仍是已證實差異，writer未知。不改像素、不遮罩、不調門檻，也不稱為逐像素完成。重啟需使用者明確指定此診斷，或出現新的玩家可見缺陷證據。

2026-10-01 定位勘誤：目前正式 treasure-original-r2／treasure-formal-r2 收據的17個差異點，均位於地圖 tile27 的局部像素(7,9)，不是HUD地形縮圖。13張完整畫面只差1px，另4張與指令環差異共存；完整四gate仍通過，本項維持可選像素修飾、不阻塞111。

原版IDA 9.4既有解碼契約核對：0x11EEE的0x1220C／0x12220分別呼叫raw 0x4DEDA及LUT 0x4DCC6。mode3在raw路徑保留目的底色，在LUT路徑讀目的像素並映射。每張殘差的實際分支及底色寫入來源尚未知，不能用archive byte+3推論runtime分支，也不能改成固定色。

主證據：docs/data/ida/fd2_terrain_mode3_review_20261001.json（輸入雜湊、工具／位址空間、原始bytes、分級與17點影像雜湊）。舊HUD解釋保留於Issue歷史與專案交接，新增證據以未遮罩座標及camera→world tile對照否定該定位；不宣稱根因已閉合。

2026-10-06 原生來源勘誤：使用正式native近堆模式、同固定槽及受版控六回合前綴，原版seq3146 attack_result在(11,85)為索引118，正式Game為116，完整差異仍1px。舊adapter索引0只解釋舊收據，不能宣稱目前差異僅由工具清零造成。253畫面點中246點完整0px，整體行為／節點／畫面比較仍未通過，跨同步HP與unknown UI限制保留。下一步只追必要copy plane／map work的writer，不填固定色或改容差。來源、命令及限制：主契約native_profile_recheck。

怎樣算做完：定位單像素來源的原生writer／繪圖契約，依READY規格修正並以相同狀態點驗證，不遮蔽或調預算。

### P1 HD正常路徑：早中晚期操作、戰場返回與存讀檔

`modern-theme-normal-player-validation` · 工作 · [#202](https://github.com/wicanr2/fd2_re/issues/202) · 仍未完成 · 要人判

蓋亞已依使用者核准的原版導向第二版造型重製，恢復灰綠厚甲、藍色結構與壯碩四肢。僅替換身分30的80/480故事頭像與第30組24/144四方向十二幀，核准概念原圖保存在私人庫；其他360項清冊保持，清冊現為365項。大型FIGANI仍為原版。

正常START已到首章97拍，七名480頭像GPU核對相同；尾端F9短按未取得ACK，原執行器r2失敗保持。分段工具只核對START與原始PNG，保存冷讀另由完整34張首章GUI通過，不冒稱同次長流程全通。蓋亞另抽普通Z選角、北移三格／取消、F2及新程序標題LOAD；來源與保存bytes保持，沒有狀態注入或鎖血。

角色GPU只辨識同方向三個登錄步態中的唯一圖源。兩張實際frame0/2而觀測皆報1，第三張跨相位列不可比，原因未知，沒有修改遊戲時序或宣稱同幀。不透明像素330/329個精確相同，高覆蓋像素8157/8053個符合source-over界限，低覆蓋邊緣不作聲明。錯範圍、錯角色及索爾冒充圖源均拒收。

登錄後相關Go回歸48.429秒，十項清冊測試及250條教訓／131守衛通過；全部993份Go來源保持。私人包3268檔逐檔核對，3239項原記錄保持，29檔新增、28舊檔自現行包移除但保留於Git歷史；runtime-only3063檔。私人提交984b284bda84023462785b6260c1bca7dee53117已推送且真正遠端相同，保持private。

前批地形仍為map0六組53槽，原本圓頂綠色／褐色雙塊的四底圖GPU與72/74完整圖塊證據保持。草地、其他地形、完整HUD、其他索引演出與平台HD仍待完成。唯一現況依58，契約見61，完整來源、提示、差異與重跑入口在docs/data/fd2-modern-hd-battlefield-20261019.json的gaia_redesign_followup_20261019與WORKLOG。沒有錄影、新公開圖片、新oracle、matching、整章E2或Release。

本項保持開啟：其他前景、上移、早中晚期、故事／法術／死亡與跨平台仍待正常玩家抽驗。

怎樣算做完：受版控計畫及目前正式runtime的正常鍵盤／GPU收據涵蓋早中晚期與上述主要交界；存檔可冷讀回，忠實主題抽樣回歸通過；已登錄素材顯示與catalog一致，未驗範圍明示。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

### P1 第27章增援後游標與鏡頭回交邊界待核對

`ch27-reinforcement-view-handoff` · RE待解 · [#224](https://github.com/wicanr2/fd2_re/issues/224) · 仍未完成 · 要人判

第27章修改路徑補抽已驗原版與重製都實際追加三路21筆，270筆AI零順序分歧，新增單位位置／HP逐筆相同。既有正式章收據與24／30台帳維持。來源為 docs/data/ui-traces/ch27-captain-reinforcement-resample.json，READY與限定CONFORMED契約在 docs/data/ida/fd2_ch27_flow_20261016.json#resample_hp_lock_policy。

增援完成的游標／鏡頭交回點仍不同：重製回死亡隊長附近，原版停於最後援軍附近。完整兩側view保存在收據advisory_view。不宣稱這是單一frame相位差，也不先斷言正式Game缺陷；先分清35822／35992演出返回邊界與重播器後續焦點移動。

排程P1可選呈現，非增援或章流程阻擋。使用者只要求流程一致、畫面參考，這項不重開已通過章，也不啟動全面matching或硬體時序研究。

怎樣算做完：由既有IDA與目前同源執行收據定位原版回交邊界，辨明正式Game或測試重播差異；READY契約後修正對應consumer，以同源局部重播核對終點並保留舊差異歷史。

### P1 蓋亞HD待機與蓄力手腕反折及修正版確認

`gaia-hd-guard-wrist-review` · 缺陷 · [#268](https://github.com/wicanr2/fd2_re/issues/268) · 仍未完成 · 要人判

使用者指出蓋亞大型戰鬥站姿的手有反折感。檢視現行待機與蓄力母稿後，近側前臂、手腕及拳頭的朝向銜接不自然，需修正母稿並重編相關影格。保留原版厚甲造型、朝向、原始描述子、節拍及規則。修正版的來源與實際遊戲呈現自動驗證先完成，造型人工確認依既有決定留待全部HD串接後集中處理，不另設逐角色批准流程。

技術修正已接入v2：待機／蓄力十幀更新，出拳後五幀PNG保持；完整十五幀實際繪製、F2與返回地圖通過。正式及精簡素材包的載入與缺件原子性通過，這輪沒有OOM。這是技術接入驗證，新造型人工確認仍按全部HD後集中處理，本議題保持開啟。

怎樣算做完：修正待機與蓄力母稿的連續肩肘、前臂、手腕與拳頭結構，接入全部受影響影格，完整Game.Draw、F2及缺件原子性通過；集中人工確認時檢查站姿自然度，確認後才關閉。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

### 凱麗HD戰鬥右腳落點與README動畫修正

`kylie-hd-right-foot-review` · 缺陷 · [#269](https://github.com/wicanr2/fd2_re/issues/269) · 仍未完成 · 要人判

凱麗舊HD母稿的可見右靴誤畫為懸空並露鞋底，與固定FIGANI36／37的踩地承重姿勢不同。六母稿及完整20幀已修為v2，原框、節拍、標記、規則與保存保持。20幀來源及Game.Draw核心、五組Go、21項來源拒收、兩包載入／缺件原子性均通過，無SKIP或OOM。第九章同槽seed1普通攻擊、自然返回、F2、保存與新程序冷讀通過；25張普通PNG實際圖源核心通過。README GIF與靜態圖改用新普通v2，完整工具列原PNG及精確遊戲區裁切逐筆保留。剪輯配對依實際PNG核心相容集合，audit先行及多候選歧義明示；相近戰況不宣稱逐幀一致。新造型仍依全部HD後集中人工確認，保持OPEN。

主收據kylie_right_foot_followup_20261022及kylie_readme_animation_20261022；規格61「凱麗右腳與README動畫修正」。

怎樣算做完：修正髖／膝／踝與腳底的連續結構及原版右腳承重落點，六母稿與20影格接入；完整來源GPU、Game.Draw／F2／返回及兩包載入／缺件原子性通過，以新普通GUI擷取更新README GIF；集中人工確認自然度後才關閉。

證據：`docs/data/fd2-modern-hd-battlefield-20261019.json`

## release — 發行、平台與封包

### 網頁版前景玩家路徑待最終自動驗收

`wasm-web-release` · 工作 · [#7](https://github.com/wicanr2/fd2_re/issues/7) · 仍未完成 · 還沒出現

排程：P2，維持先前「最後處理」決議；本輪只修正依賴定義。已暫停的全檔C研究 #198 與非阻擋診斷 #52／#197 不算本項前置，避免永久等待。

建置、資產與存檔都已通過；剩餘缺口是前景瀏覽器中的開場至第一關操作，以及重新載入後的存檔讀回。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式在 Docker／Xvfb 啟動非 headless 瀏覽器，以 CDP 將頁面帶到前景並自動送正常輸入、截圖與檢查 localStorage；不再要求使用者人工確認。若 Chromium 仍把頁面標成 hidden，先修正可重現的前景瀏覽器工具鏈，不得把 hidden 分頁結果冒充通過。

卡在：完成目前啟用的P0戰役與P1重製／現代主題工作後執行；P3考古、暫停研究與非阻擋像素診斷不在依賴內。

怎樣算做完：目前啟用的P0／P1重製與現代主題工作完成後，在 Docker／Xvfb 的非 headless Chromium 中由 CDP `Page.bringToFront` 確認 `visibilityState=visible`，從目前 WASM 產物正常走到第一關可操作，完成一次存檔、重新載入與讀回；保存輸入、畫格、console、localStorage 前後值與截圖到 docs/data/ui-traces/wasm-browser-player-path-e1.json，並由自動測試驗證。

### Android 封包沒有建置

`android-package` · 工作 · [#8](https://github.com/wicanr2/fd2_re/issues/8) · 仍未完成 · 還沒出現

排程：P2，平台交付階段；保留原平台與實機驗收範圍，不因Linux結果宣稱其他平台完成。

觸控輸入已支援，但沒有 `ebitenmobile bind` → `.aar` → APK 的建置流程。

怎樣算做完：產出可安裝的 APK，並在實機確認啟動、存檔與音訊。

### Windows 與 macOS 還沒有實機抽測

`release-platform-acceptance` · 工作 · [#9](https://github.com/wicanr2/fd2_re/issues/9) · 仍未完成 · 要人判

排程：P2，平台交付階段；保留 Windows 與 macOS 實機驗收範圍。

2026-10-17 正式第一版 [v.1.0.0-20261017](https://github.com/wicanr2/fd2_re/releases/tag/v.1.0.0-20261017) 已公開為 Latest。macOS 通用 ZIP、Windows x86_64 ZIP、Linux x86_64 AppImage 均從同提交完成原生程序自檢；回下載 SHA、架構、授權與每包 447 份公開素材核對通過。來源、原生 CI 及限制見[正式發行收據](https://github.com/wicanr2/fd2_re/blob/main/docs/data/fd2-release-v.1.0.0-20261017.json)。

仍待 Windows 與 macOS 實機的啟動、存檔／讀檔與音訊抽測。本版 Windows／macOS 未簽署，macOS 尚未公證，已在公開發行說明中揭露。這項 Issue 保持開啟。

怎樣算做完：兩個平台各自在實機完成啟動、存檔／讀檔與音訊抽測並記錄結果。

## tooling — 工具、編輯器與工作流程

### 戰場編輯器待最終自動端到端驗收

`battlefield-editor-mvp` · 工作 · [#10](https://github.com/wicanr2/fd2_re/issues/10) · 仍未完成 · 還沒出現

排程：P2，維持先前「最後處理」決議；本輪只修正依賴定義。已暫停的全檔C研究 #198 與非阻擋診斷 #52／#197 不算本項前置，避免永久等待。

戰場編輯器已具備地圖與單位編輯、部署格、波次總覽、復原及格式保真測試；缺的是由瀏覽器編輯正式複本、存回、重生 canonical，再由正式 runtime 載入的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做，屆時由代理程式以瀏覽器自動化操作可丟棄複本並驗證，不再要求使用者人工畫圖或授權目錄。

卡在：完成目前啟用的P0戰役與P1重製／現代主題工作後執行；P3考古、暫停研究與非阻擋像素診斷不在依賴內。

怎樣算做完：目前啟用的P0／P1重製與現代主題工作完成後，在 Docker 中啟動 editor server 與非 headless 瀏覽器，對一張原版地圖的可丟棄複本執行圖塊、單位與部署格各一項編輯，經正式 save API 存回並重生 canonical；正式 runtime 載入後核對地形成本、單位、部署格與畫面，最後還原來源零差異。保存 docs/data/ui-traces/battlefield-editor-roundtrip-e1.json 並由自動測試驗證。

### 劇情編輯器待最終自動端到端驗收

`campaign-editor-ui` · 工作 · [#11](https://github.com/wicanr2/fd2_re/issues/11) · 仍未完成 · 還沒出現

排程：P2，維持先前「最後處理」決議；本輪只修正依賴定義。已暫停的全檔C研究 #198 與非阻擋診斷 #52／#197 不算本項前置，避免永久等待。

劇情編輯器五個分頁與節點圖均已可用，現有原版路線 round-trip 已通過；缺的是建立一條原版沒有的敗北／choice 路線，再由正式引擎走通首尾的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式以瀏覽器自動化建立可丟棄測試路線並驅動引擎驗收，不再要求使用者人工設計或試玩。

卡在：完成目前啟用的P0戰役與P1重製／現代主題工作後執行；P3考古、暫停研究與非阻擋像素診斷不在依賴內。

怎樣算做完：目前啟用的P0／P1重製與現代主題工作完成後，在 Docker／非 headless 瀏覽器中用編輯器對可丟棄複本新增一條 battle.on_lose 敗北路線與一個依旗標過濾的 choice 分支，透過正式 save API 存回、重生 canonical，並由正式引擎以決定性輸入分別走通兩個分支首尾；還原來源零差異。保存 docs/data/ui-traces/campaign-editor-custom-route-e1.json 並由自動測試驗證。

### P3 有界記憶體觀察：工具已備，目標像素追蹤待重啟

`oracle-bounded-memory-change-observation` · 工作 · [#197](https://github.com/wicanr2/fd2_re/issues/197) · 仍未完成 · 要人判

排程：P3，依附 #52 目標診斷；不是重製主線或HD的前置。

既有工具、參數拒收、正常短前綴與觀察開關不擾動結果已具有限收據。尚缺第12章目標窗口的正式正常來源，不得把READY工具誤標為pixel writer已解。使用者重啟 #52 或新的實際玩家阻擋需要此能力時再執行。

父項#52。正式native第12章seq3146像素仍118對116，現有EIP／map_state只有配置回傳與16項全域，沒有原始目標byte的改變來源；不能用位址重用或舊adapter清零推定最後writer。只擴充既有受版控apps/fd2/cmd/oracle，預設停用、宣告固定最多16個LE線性byte位址、既有追蹤窗口及筆數上限，每次成功instruction後唯讀觀察，記錄baseline或value change與原始指令定位。未變同值寫入不宣稱已觀察；不改CPU、heap、RNG、鍵盤、畫面或遊戲檔案。輸出留本地work，公開庫只存來源／收據雜湊。READY規格先於實作，wrapper明示參數與來源審查。

2026-10-06 READY工具實作已由dosgolem fa25f6e7d7f6be399b6563b7df6a55995daba77b提交推送，真正遠端相同。1058項平台測試零失敗零略過；原版百萬步啟用／停用的完整報告及PNG一致，除新增觀察欄位及輸出路徑。53筆觀察含52次變更。FD2兩層wrapper傳遞旗標並綁來源清冊，CPU／heap來源保持。BOOT時目標pixel尚未配置的false收據保留，不使用0代替；正向BOOT使用當時已存在全域／stack。首次fixture漏DS descriptor已修正並乾淨重跑。正常短槽已通過：233份JSON、232張PNG、輸入與EIP追蹤逐檔相同。FD2驅動器0734838e已推送。第12章目標觀察於2026-10-06依使用者要求暫停，保存到第1回合control_seq1110、2,900,397,260步；1111份JSON與1111張PNG與既有前綴相同，尚未到byte觀察窗口。停止程序143不是產品失敗，無target writer新結論。本案保持開啟，下次以同槽／計畫及新輸出與新覆蓋層重跑，保留中止收據。唯一來源／命令／雜湊：地形主契約memory_observation。

卡在：第12章目標追蹤維持暫停；既有觀察工具供需要時使用。

怎樣算做完：bounded參數與越界拒收、disabled原metadata保持、原始記憶體／CPU不變；同固定槽native正常計畫重生，既有完整checkpoint／PNG／controls一致後確認原生byte變更定位，不用新觀察放寬#52比較。

證據：`['docs/data/ida/fd2_terrain_mode3_review_20261001.json#native_profile_recheck', 'tools/dosgolem_oracle_container.sh']`

### 【暫停】全檔 C 位元組匹配與原始碼保存

`fd2-full-matching-decompilation` · 工作 · [#198](https://github.com/wicanr2/fd2_re/issues/198) · 仍未完成 · 要人判

排程：P3，使用者於 2026-10-15 明確暫停。保留既有成果與原始驗收條件；使用者本輪補充允許必要時局部匹配確認已重現差異；全面工作仍暫停。每個局部任務先定玩家阻塞點、固定輸入、待驗函式與停止條件，既有證據足夠就不重編。

本題是工具鏈與原始碼保存研究，不是 Go／Ebiten 重製版、HD 主題、章節驗收或封包的前置條件。一般原版行為對拍繼續使用正式 dosgolem oracle；遇到具體玩家缺陷才做最小充分 RE，不重開全檔匹配。

使用者2026-10-08定案：完成固定版 FD2.EXE 的 matching decompilation，遊戲邏輯還原成 C；原作組語與第三方函式庫可保留，覆蓋分列。

主契約：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。原版固定357074 bytes，SHA-256 222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f。

已完成的前置試驗不等於全檔完成；不得以原版 EXE 複製或未知遊戲機器碼回填冒稱 C 還原。

卡在：全檔工作依使用者指示暫停；必要時有界局部匹配已獲允許，不能擴張為全檔重啟。

怎樣算做完：從受版控來源及明示本機依賴乾淨建置完整 FD2.EXE，全檔大小及 SHA-256 相同；遊戲 C 函式逐一產碼一致；原作組語、第三方函式庫、資料與未知分列，未知不得冒稱已還原；正式 dosgolem 原版入口抽樣啟動。

證據：`docs/goal/118-goal-fd2-matching-decompilation-20261008.md`

### P1 原版收據連續延後截圖不得覆寫待輸出檔名

`oracle-deferred-frame-overwrite` · 缺陷 · [#206](https://github.com/wicanr2/fd2_re/issues/206) · 仍未完成 · 要人判

固定FD2.EXE、dosgolem 2c5c104的第24章ch24-sample-r3兩輪皆缺seq162 PNG。該點在runtime memcpy／11ED9→12103，frame_deferred=true；下一個force-enemy-clear只跑1步，capture把單一pendingFrame覆為163，162沒有輸出。原始失敗及完整保存留work/remake-recovery-20261015/ch24-original-r1與r2，主證據docs/data/fd2-remake-toolchain-recovery-20261015.json。

這是工具收據缺件，不是重製畫面錯誤。章計畫r4明示在mark後正常空等待200萬步，先完成原輸出再清敵；不補圖、不遮罩或放寬比較器。通用工具修正另走READY規格，保存每個待輸出名稱與其狀態邊界，禁止把較晚畫面冒稱同狀態。

怎樣算做完：受版控READY規格先於修正；相鄰兩個insideScreenWriter邊界的PNG都不丟失且來源／延後步數可追；保留CPU與輸入時序、不以中途畫面充數。正反向測試及固定原版重跑通過，既有前綴與畫面比較保持。

證據：`docs/data/fd2-remake-toolchain-recovery-20261015.json`

<!-- END fd2_worklist.py render -->
