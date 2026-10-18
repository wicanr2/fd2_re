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

共 20 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

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

2026-10-18現況：正式序章consumer的97拍已補齊父王48、王后66、強盜96，正常START至首章通過。31位已登錄頭像保留80圖並新增完整480圖集合，catalog共195筆／1187個file與files指向均預檢通過；私人素材已推3f289e7b並核遠端。其餘六個靜態缺頭像宣告仍只是候選，其他章、動態入隊及城鎮consumer尚待確認。本項保持開啟；來源見61、58與fd2-modern-hd-20261018.json。

以下較早記錄保留為歷史；目前以本段與58為準。

排程：P1，#4 的素材子項。先由正式scenario／story／商店／整備與戰鬥consumer建立可重跑的需求清冊，再與modern catalog逐項比對。區分未登錄、缺檔、未接線與刻意保留，不把所有DATO／FDICON索引一律當作需求。

已有33張地圖及94組可繪人物候選；39／57依61是純重映射。故事頭像只登錄28位，正式對話遇未知speaker會拒收，必須查玩家可達來源後補齊。城鎮、商店、整備、戰鬥演出與字型先盤點實際consumer，再將需要新增的美術範圍交使用者確認，不自行擴張61。

重用核准風格與靜態嘴型政策。不得放寬雜湊／幾何／未知speaker拒收以宣稱覆蓋。新增或修訂素材通過清冊與runtime後以受版控allow-list同步唯一私人庫。

2026-10-15 靜態候選盤點已可重跑：tools/audit_modern_theme_coverage.py --verify-private --output <report.json>，只在Docker執行。故事／scenario宣告仍有九個非負speaker未登錄頭像；負speaker另列，地圖與人物宣告無新增缺件。這些含未證明可達的舊資料，不能直接當玩家缺陷或完整素材需求。私人18項Go回歸零略過通過，catalog仍runtime_candidate；下一步逐筆核對正式consumer與正常路徑。來源與限制見docs/data/fd2-remake-toolchain-recovery-20261015.json。

怎樣算做完：受版控需求盤點可重跑，列出來源與consumer；既有已核准現代替換範圍零非預期缺件，其他畫面各有明示分類與決策；缺件拒收與原子切換回歸通過，私人包雜湊及遠端HEAD可核對。

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

2026-10-18物品字型接續：物品操作與商店裝備穩定面板從同份快照繪HD文字／數字及raw slot選取，裝備重建失敗還原完整快照。依實際拷貝矩形遮蔽舊文字並保留框外部分，原12幀轉場與像素素材保持。正常START後F9接續既有hotel_ch02，經城鎮武器店普通選索爾，三筆選取、拒絕裝備藥草、重新裝備已穿戴短劍、F2、縮放／說明遮罩及返回保存通過；13張實際視窗已審查。角色與金幣保持，兩主題及來源完整保存SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

相關cmd介面73.341秒、資料合成器0.313秒及五資料／清冊套件通過；不是完整cmd或battle回歸。戰場物品及死亡交換caller只有相關回歸，未新取得正常GUI。四語各5107筆譯文及狀態保持。主收據item_followup與61為入口，README新增實際HD商店裝備畫面。首次GUI城鎮方向錯誤已分類腳本失敗，同一建置完整r2重跑通過。暫時執行檔已刪除，未錄影，無新美術或Release變更。

原生整備／標題LOAD等固定面板、預建轉場、圖示字樣及平台效能仍待完成，本項保持開啟。以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18角色狀態字型接續：教會與戰場的穩定狀態／命令頁從開啟時同份資料快照合成HD文字與數字，忠實快取及12／14幀轉場保持。正常START後F9接續既有hotel_ch02，經城鎮普通選人開啟索爾狀態與悠妮命令；24張實際視窗已審查，兩頁F2往返PNG SHA各自相同，縮放／說明遮罩與返回保存通過。兩主題及來源完整保存SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

來源提交：70d50d7cf78d9de4770d3e685ee79b347d8f166e。

相關cmd介面96.249秒、資料合成器0.689秒、四資料套件與清冊回歸通過，不宣稱完整cmd或battle回歸。戰場消費端及轉場推進以fixture驗證，限RUNTIME-E1，未新取得正常戰場GUI。四語各5107筆譯文／狀態／變數保持。主收據status_followup與61為入口，README新增實際HD角色狀態畫面。暫時執行檔已刪除，未錄影；沒有新美術或Release變更。

物品操作／整備／原生標題LOAD等固定面板、預建轉場、圖示文字及平台效能仍待完成，本項保持開啟。以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18教會字型接續：即時選單、四語名冊、轉職／復活列表與問句、交付物品及提示已接HD黑體與十進位數字。正常START後F9接續既有hotel_ch02，經城鎮進教會，復活及交付均走取消；四語名冊、F2、縮放／說明遮罩與15張實際GPU截圖審查通過。回旅館F5／F9，兩主題保存與來源槽完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

來源提交：48ab3e57dc7ea72f3f01fd4726ff26be59748c23。

相關cmd介面回歸83.485秒及campaign／fdtxt／fdother／localization回歸通過；不是完整cmd測試。轉職、空名冊、餘額不足與滿袋等fixture分支限RUNTIME-E1。四語各5107筆譯文及狀態保持，沒有新翻譯。[主收據](https://github.com/wicanr2/fd2_re/blob/main/docs/data/fd2-modern-hd-fonts-20261018.json)的church_followup與61為入口，README新增HD教會復活確認實際畫面。教會快取狀態、預建轉場、圖示文字及其他固定面板仍待完成，本項保持開啟。

以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18原生旅館字型接續：即時提示、四槽編號／章名／標題與保存／讀取回覆已接HD黑體。正常START後F9接續既有hotel_ch02，四語上下文、四槽選取、保存、服務LOAD、F2與13張實際視窗審查通過；兩主題保存及來源槽完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。原生旅館文字保持，沒有新增翻譯。#254服務回覆殘留歡迎詞已修正，負例及去重回歸保留；候選完整回歸與最終focused驗證的來源界線另列於主收據。

來源提交：2e0d17897d6fc440c8e65298882d4c46398887c2。[主收據](https://github.com/wicanr2/fd2_re/blob/main/docs/data/fd2-modern-hd-fonts-20261018.json)的hotel_followup及61為本切片入口。README已加入HD旅館實際保存畫面，原四語忠實截圖保持。暫時執行檔已刪除，未錄影；沒有新美術、私人素材或正式Release變更。

接續物品／狀態／教會／整備／標題LOAD等固定面板、預建轉場與圖示內字樣；再接完整介面、戰場與平台效能。本項保持開啟，不宣稱整款字型或HD完成。

以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18接續：HD清晰黑體主路徑已接入最終1920×1200繪製，涵蓋一般Font.Draw、原生故事逐字／捲動、商店文字與31／42／119十進位數字。正常START首句四語、F2同狀態往返、F9接續旅店後城鎮／商店及單筆購買1000→990、三種尺寸與最大字級說明頁通過；首章JSON標題LOAD、F5／F9及完整保存bytes保持。cmd/fd2、campaign、fdother與localization回歸通過，四語各5107筆譯文／狀態／變數保持。README已加入四語忠實模式實際商店截圖，原始素材包未公開，正式第一版tag保持。

來源提交：ca1d0a13f8d5dc39ee40f027c29de38b6759ee90。主收據：[HD字型](https://github.com/wicanr2/fd2_re/blob/main/docs/data/fd2-modern-hd-fonts-20261018.json)。本項保持開啟，下一步補原生物品／狀態／教會／旅店／整備等固定索引面板、預建轉場與圖示內字樣；再接完整介面、戰場與平台效能。特殊字形fallback、未驗章節與正常流程限制如實保留。#251說明間距及#253捲動快照缺陷已修正驗證，並依來源提交結案。

以下較早記錄保留為歷史；目前以本段、58、61及新主收據為準。

2026-10-18接續：#252首章JSON標題LOAD已依READY契約修正。忠實／HD兩個新程序普通標題LOAD、F2/F1/縮放/F9及重新F5後第三個新程序標題冷讀通過；原樣與重新保存完整bytes相同，SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。原生SAV交易與格式保持，首章JSON仍用既有在地化槽畫面。來源：docs/data/fd2-json-title-load-20261018.json。只驗首章JSON，其他章節與玩家交界、介面／字型／戰場及跨平台效能仍待驗；#251說明間距保持開啟。

以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18接續現況：非原生對話已直接使用480×480圖源，保留既有168邏輯區域及鏡像，1920視窗實際顯示504×504。首章正常F5原樣JSON在兩個新程序正常START後以F9冷讀，F2雙向、F1遮罩、縮放與F9重讀通過；六個GPU區域最大正規化MAE為0.00162285，Triangle參照與GPU線性插值差異已揭露。存檔SHA不變。來源：docs/data/fd2-modern-hd-dialogue-20261018.json及61。首章下框限PLAYER-E2，上下框幾何為RUNTIME-E1；未宣稱完整戰役或跨平台HD。標題LOAD對此JSON因chapter拒收另列#252，同份F9可讀；F1間距仍#251。 原生序章、F2及本切片保留已驗狀態，接續介面、字型、戰場與跨平台效能，本項保持開啟。

以下較早記錄保留為歷史，目前以本段、58及61為準。

2026-10-18現況：已完成1920×1200／480×480原生故事頭像的啟動與F2動態入口。由忠實START途中雙向切換，完整97拍至首章，七位480區域MAE=0；視窗還原、非整數縮放、黑邊及實際說明遮罩通過。兩種主題LOAD同份首章JSON的單位結果相同、保存SHA不變；F5沿既有節點契約，不是戰中快照。五種忠實GPU濾鏡對照全畫面零像素差異，成功預檢素材於程序內快取。來源：61、58及docs/data/fd2-modern-hd-f2-20261018.json。非原生對話仍取80×80圖，先由#202補最終層；再接介面／字型／戰場與跨平台效能。F1間距缺陷另列#251。未宣稱整款HD或穩定60fps，本項保持開啟。

以下較早記錄保留為歷史；目前以本段與61、58為準。

2026-10-18使用者已確認父王原型手繪風格，先補齊序章頭像，再接高解析呈現，忠實原版保留。三個缺少頭像已完成正常97拍序章。已展示同狀態80×80與160×160原型，固定頭像位置、對話框及文字區；160倍率與更高畫布方案等待本次選擇。未改production高解析renderer或資料格式；本項仍DRAFT。來源與已完成切片見61及docs/data/fd2-modern-portraits-20261018.json。

以下較早記錄保留為歷史；目前以本段與58為準。

排程：P1，#4 的高解析分支，目前DRAFT。現有61規格只授權原生尺寸的真彩色重繪候選；24×24人物及80×80頭像不等於高解析貼圖，已有高解析母稿也不代表已接到正式renderer。

使用者已要求HD化；第一階段是先補齊現代主題並做原型，或同步完成高解析繪製，正在確認。依grilling先展示相同遊戲狀態的可丟棄原型與現有呈現，保留核准的風格與操作語意；倍率、圖層、文字安全區、素材格式及效能成本由原型查證後確認。不要自行將插值放大當成細節提升，也不要在未READY前改正式資料格式、存檔或production繪製。

怎樣算做完：1920×1200故事頭像、F2、首章JSON冷讀、HD黑體主路徑、旅館／教會、角色狀態／命令與物品操作穩定頁已驗。接續原生整備／原生標題LOAD固定面板、預建轉場與圖示字樣，再完成介面／戰場與跨平台效能；完整字型及整體HD仍未完成。

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

2026-10-18接續：#252首章JSON標題LOAD已依READY契約修正。忠實／HD兩個新程序普通標題LOAD、F2/F1/縮放/F9及重新F5後第三個新程序標題冷讀通過；原樣與重新保存完整bytes相同，SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。原生SAV交易與格式保持，首章JSON仍用既有在地化槽畫面。來源：docs/data/fd2-json-title-load-20261018.json。只驗首章JSON，其他章節與玩家交界、介面／字型／戰場及跨平台效能仍待驗；#251說明間距保持開啟。

以下較早記錄保留為歷史，目前以本段、58與61為準。

2026-10-18接續現況：非原生對話已直接使用480×480圖源，保留既有168邏輯區域及鏡像，1920視窗實際顯示504×504。首章正常F5原樣JSON在兩個新程序正常START後以F9冷讀，F2雙向、F1遮罩、縮放與F9重讀通過；六個GPU區域最大正規化MAE為0.00162285，Triangle參照與GPU線性插值差異已揭露。存檔SHA不變。來源：docs/data/fd2-modern-hd-dialogue-20261018.json及61。首章下框限PLAYER-E2，上下框幾何為RUNTIME-E1；未宣稱完整戰役或跨平台HD。標題LOAD對此JSON因chapter拒收另列#252，同份F9可讀；F1間距仍#251。 早中晚地圖、戰鬥返回、法術／物品、其他章節與正式標題冷讀仍待驗，本項保持開啟。

以下較早記錄保留為歷史，目前以本段、58及61為準。

2026-10-18現況：原生現代頭像與啟動高解析序章已通過。新增正常忠實START途中F2，完整97拍至首章、七位480區域MAE=0，雙向切換／縮放／黑邊／遮罩及同份首章JSON在兩種主題LOAD的結果均通過。來源：docs/data/fd2-modern-hd-f2-20261018.json與61。最終LOAD截圖確認非原生對話分支仍在Game.Draw取modern.image的80×80圖，未提交480最終層；這是下一個有實際玩家證據的HD切片。早中晚戰場、戰鬥返回、法術／物品、正式冷讀與其他圖層仍待驗，本項保持開啟。

以下較早記錄保留為歷史；目前以本段與58為準。

排程：P1，#4 的執行期驗收子項，可先用已登錄素材做有界抽測。用目前正式Game從標題正常START／LOAD進場，抽早／中／晚期地圖、走行遮擋、游標、HUD、四向指令、取消、故事上下框、攻擊／施法返回、戰後商店／整備及存讀檔。

沿61核准範圍驗證現代圖層沒有被原版完整indexed幀蓋回，語意結果、座標、存檔與忠實主題相容。現代畫面驗收與忠實原版pixel parity分開。缺圖或未知consumer如實拒收，轉交素材子項；不以現有loader單測或總覽圖代替正常玩家收據。

怎樣算做完：受版控計畫及目前正式runtime的正常鍵盤／GPU收據涵蓋早中晚期與上述主要交界；存檔可冷讀回，忠實主題抽樣回歸通過；已登錄素材顯示與catalog一致，未驗範圍明示。

### P1 第27章增援後游標與鏡頭回交邊界待核對

`ch27-reinforcement-view-handoff` · RE待解 · [#224](https://github.com/wicanr2/fd2_re/issues/224) · 仍未完成 · 要人判

第27章修改路徑補抽已驗原版與重製都實際追加三路21筆，270筆AI零順序分歧，新增單位位置／HP逐筆相同。既有正式章收據與24／30台帳維持。來源為 docs/data/ui-traces/ch27-captain-reinforcement-resample.json，READY與限定CONFORMED契約在 docs/data/ida/fd2_ch27_flow_20261016.json#resample_hp_lock_policy。

增援完成的游標／鏡頭交回點仍不同：重製回死亡隊長附近，原版停於最後援軍附近。完整兩側view保存在收據advisory_view。不宣稱這是單一frame相位差，也不先斷言正式Game缺陷；先分清35822／35992演出返回邊界與重播器後續焦點移動。

排程P1可選呈現，非增援或章流程阻擋。使用者只要求流程一致、畫面參考，這項不重開已通過章，也不啟動全面matching或硬體時序研究。

怎樣算做完：由既有IDA與目前同源執行收據定位原版回交邊界，辨明正式Game或測試重播差異；READY契約後修正對應consumer，以同源局部重播核對終點並保留舊差異歷史。

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
