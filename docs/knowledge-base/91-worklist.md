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

共 22 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

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

排程：P0，與相關章的正常玩家路徑一起處理。2026-10-15 容器執行全域 sync --check 通過，但仍有未轉寫事件；唯一數量與分布見 [58](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/58-fd2-exe-re-coverage.md)。

既有完整轉寫與已通過章保持。優先補第24～26章會被正常路徑消費的事件，再補第1～3章及第21／22章剩餘事件。每個handler須保留完整動作與參數、共享尾段、READY審查及正式consumer，不能只把增援資料視為完整事件。

FDFIELD回合事件由完整具型別處理器降成正式劇本，保留增援、AI模式、鏡頭、演出與對白；不得只保留spawn。原始表為0x51B91，工具為tools/extract_native_death_events.py與tools/sync_native_turn_events.py。

2026-10-06現況：全域--check通過，仍有27筆未轉寫。原先「其餘57筆、只有第四／五章」是歷史快照，不能作目前待辦。現已轉寫的章由工具直接列出；其餘依111正常章收據逐章處理，不先批次替換。

本輪補第十七章event40：group2後鏡頭(17,37)與text1三句。固定EXE／IDA9.4全handler與共享尾段經READY審查，再由正式lower同步到scenarios及canonical。正常Game確實消費原先缺失三句，124筆AI零順序分歧，98張畫面與酒店SAV保持。#195／#196工具補正後同來源四gate皆通過。既有第17章PLAYER-E2保持，未重新執行原版或新增章收據。

主契約：docs/data/ida/fd2_turn_event40_20261006.json；唯一分層現況見58。剩餘未轉寫列分布於第1／2／3／21／22／24／25／26章，本題維持開啟。

怎樣算做完：tools/sync_native_turn_events.py --check（不帶 --chapters）通過，報告裡「尚未轉寫的回合事件」為 0，且每一章的擴充都有該章 111 收據。

### P1 HD素材覆蓋：由正式玩家consumer列出缺件並補齊

`modern-theme-player-asset-coverage` · 工作 · [#201](https://github.com/wicanr2/fd2_re/issues/201) · 仍未完成 · 要人判

排程：P1，#4 的素材子項。先由正式scenario／story／商店／整備與戰鬥consumer建立可重跑的需求清冊，再與modern catalog逐項比對。區分未登錄、缺檔、未接線與刻意保留，不把所有DATO／FDICON索引一律當作需求。

已有33張地圖及94組可繪人物候選；39／57依61是純重映射。故事頭像只登錄28位，正式對話遇未知speaker會拒收，必須查玩家可達來源後補齊。城鎮、商店、整備、戰鬥演出與字型先盤點實際consumer，再將需要新增的美術範圍交使用者確認，不自行擴張61。

重用核准風格與靜態嘴型政策。不得放寬雜湊／幾何／未知speaker拒收以宣稱覆蓋。新增或修訂素材通過清冊與runtime後以受版控allow-list同步唯一私人庫。

2026-10-15 靜態候選盤點已可重跑：tools/audit_modern_theme_coverage.py --verify-private --output <report.json>，只在Docker執行。故事／scenario宣告仍有九個非負speaker未登錄頭像；負speaker另列，地圖與人物宣告無新增缺件。這些含未證明可達的舊資料，不能直接當玩家缺陷或完整素材需求。私人18項Go回歸零略過通過，catalog仍runtime_candidate；下一步逐筆核對正式consumer與正常路徑。來源與限制見docs/data/fd2-remake-toolchain-recovery-20261015.json。

怎樣算做完：受版控需求盤點可重跑，列出來源與consumer；既有已核准現代替換範圍零非預期缺件，其他畫面各有明示分類與決策；缺件拒收與原子切換回歸通過，私人包雜湊及遠端HEAD可核對。

## runtime — 還沒接進正式執行期

### P1 補齊剩餘 HUD 重繪：玩家確認、物理返回與劇情關框

`hud-anchor-attack-and-message-redraws` · 缺陷 · [#41](https://github.com/wicanr2/fd2_re/issues/41) · 仍未完成 · 自承還在 docs/knowledge-base/56-fd2-remake-sdd.md

排程：P1；若在P0章路徑造成差異，當場提升為該章阻擋。

死亡最後steady的0x1DEAE及已證實196CB戰場END／獎勵closing已接線並有有限回歸。剩餘為玩家指令確認的0x1D3FF、普通物理0x1563B／0x1565C與尚未覆蓋的劇情／升級closing。0x15510屬簡略演出，不能接到普通full攻擊。逐caller證據與目前consumer以 [HUD主契約](https://github.com/wicanr2/fd2_re/blob/main/docs/data/ida/fd2_hud_redraw_20261006.json) 及56為準。

利用後續正常章收據確認，不重跑已passed章，不重解已閉合caller，也不以額外評估點消除未知差異。

怎樣算做完：以原版 eip-trace（或同狀態擷取）證明這三類重繪發生的時機與當時的閘門／可見游標，接到同一個入口。不為了驗收重跑已 passed 的章（使用者 2026-09-18 定案）：以之後正常排程的新章收據順帶確認即可，沒有出現對應時點就在 56 §#40 記錄仍未被覆蓋。

### P1 HD高解析呈現：範圍確認、並列原型與READY規格

`modern-theme-highres-prototype` · 工作 · [#204](https://github.com/wicanr2/fd2_re/issues/204) · 仍未完成 · 要人判

排程：P1，#4 的高解析分支，目前DRAFT。現有61規格只授權原生尺寸的真彩色重繪候選；24×24人物及80×80頭像不等於高解析貼圖，已有高解析母稿也不代表已接到正式renderer。

使用者已要求HD化；第一階段是先補齊現代主題並做原型，或同步完成高解析繪製，正在確認。依grilling先展示相同遊戲狀態的可丟棄原型與現有呈現，保留核准的風格與操作語意；倍率、圖層、文字安全區、素材格式及效能成本由原型查證後確認。不要自行將插值放大當成細節提升，也不要在未READY前改正式資料格式、存檔或production繪製。

卡在：HD第一階段範圍待使用者確認；不阻塞現有核准主題的素材盤點與正常路徑驗收。

怎樣算做完：使用者確認本階段HD範圍；受版控工具可重跑同狀態並列原型，記錄素材實際解析度、輸出尺寸、文字與游標幾何、效能與權利；證據審查後將61相關高解析契約提升READY，正式實作另依該契約驗收。原型本身不算HD完成。

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

排程：P1，併入代表性玩家路徑與素材驗收；保持原驗收條件，不以綠色單元測試代替人工聽辨或正常路徑。

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 全戰役驗收總項：完成剩餘章節與戰後／存檔邊界

`campaign-postbattle-e2-full` · 工作 · [#14](https://github.com/wicanr2/fd2_re/issues/14) · 仍未完成 · 還沒出現

排程：P0。這是戰役驗收總項，不與各章工作重複計算完成度。唯一章狀態仍由 [parity-campaign-progress.json](https://github.com/wicanr2/fd2_re/blob/main/docs/data/parity-campaign-progress.json) 與 [58](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/58-fd2-exe-re-coverage.md) 判定。

先完成 #142 第24章，再按章推進25～30章及結局，最後補齊1～3章。回合事件 #33 與 HUD #41 隨相關章一起收斂。不為非阻擋像素 #52 或已暫停 #198 等待，不重跑已通過章來追求函式數或無關畫面。

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以同狀態走過戰前對白、戰鬥節拍、戰後節點、城鎮與存檔邊界並與原版比較。

2026-09-15 決議（取代 2026-08-23 的「交人工遊玩回報」）：由代理程式依 [`111` 目標提示詞](https://github.com/wicanr2/fd2_re/blob/main/docs/goal/111-goal-original-parity-campaign-20260915.md) 用 dosgolem 逐章推進 30 章。起點用受版控工具依攻略校準的建構槽跳關，章內全程正常鍵盤輸入，抽樣完戰鬥節拍後以 `force-enemy-clear` 進戰後節點；這條路徑視同該章 `PLAYER-E2`。四個 gate（行為、節點、交易零容忍；代表畫面 ≤1% 差異像素）都過才算一章完成，進度台帳是 `docs/data/parity-campaign-progress.json`。

執行子項：#142 第24章、#199 第25～30章、#200 第1～3章；#203 是環境前置。總項保持原30章驗收標準，暫停研究不在依賴內。

怎樣算做完：30 章各自依 111 的章工作單元取得原版／重製收據，四個 gate 全過；parity-campaign-progress.json 的 all_chapters_passed 為 true。

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

### P0 第24章：完成四閘門、戰後城鎮與完整存檔

`ch24-full-chapter-parity` · 工作 · [#142](https://github.com/wicanr2/fd2_re/issues/142) · 仍未完成 · 要人判

排程：P0，下一個原版戰役垂直切片。

以目前 dosgolem 最新 main 建立的乾淨分支，先用同一既有控制序列核對整合後的前綴，再取得正式章收據。先核對主契約與既有有限通過結果，僅補缺失的事件、演出、戰後與存檔鏈，不重做已閉合物理／指令6 RE。

第24章 DP 校準先查現有114政策與既有使用者定案；有新政策取捨才提問。不得自行提高數值、鎖HP或反覆刷長局。第12章 #52 是既有非111阻擋項；#198保持暫停。

接續已驗收第23章非城鎮記錄SAV，SHA256 22bd63070189e0e6703e14d4bd4db8dc1f66452a21128dd5154579171ecbea63。依111／114例外，保持同一槽與seed4，不新增強化、治療、金幣或道具。先核對正常整備／選人／起手與58已閉合來源，再正常抽樣章內節拍、清敵一次、戰後與實際保存。遇反證另立缺陷／RE待解，READY先於修正，不放寬四gate。

2026-10-15 接續：#203工具鏈及#205完整IRQ7鏈已限定驗收。dosgolem專用分支2c5c104已推送，正常鍵盤短鏈兩輪233份完整狀態／PNG及trace一致。舊／新版嚴格比較仍failed，不把動畫相位或CPU步數差異隱藏；不再以工具恢復阻塞本章。同源第23章實際SAV的ch24-sample-r4短鏈已通過四gate：T1～T3、移動與敵方回合、單次清敵、戰後整備、完整SAV及退出存檔列表。13張完整RGB皆零差異，16筆AI入口零順序分岔；原版211份狀態與PNG齊全。完整SAV SHA-256為a1e6cc85722fc820cf38f820f4454785cb2138450d9c6000226466a340288406。

本輪沒有驗T4／7／10事件、普通攻擊與其他適用戰鬥抽樣，不能把短鏈四gate直接提升整章111，#142與正式台帳保持未完成。既有物理／指令6 RE不重開。r3計畫的seq162延後PNG覆寫另列#206，r4只加一次正常空等待，不補圖或放寬門檻；通用工具缺陷仍未修。主證據docs/data/fd2-remake-toolchain-recovery-20261015.json的chapter24_followup。下一步只補晚期事件及剩餘戰鬥鏈，保留現有起手／保存成果。

怎樣算做完：['固定EXEhash、dosgolem commit、同源SAV與受版控計畫／清冊，正常章路徑可重跑。', '四gate與完整SAV通過，相關Go及第23章回歸通過。', '同步56／57／58、索引、正式台帳與產生現況，保留注入、敗北及未抽範圍。']

證據：`['docs/data/ui-traces/parity-ch23.json', 'docs/goal/111-goal-original-parity-campaign-20260915.md', 'docs/data/parity-campaign-progress.json', 'docs/data/fd2-remake-toolchain-recovery-20261015.json']`

### P0 完成第25～30章與結局的正常章驗收

`campaign-ch25-ch30-finish` · 工作 · [#199](https://github.com/wicanr2/fd2_re/issues/199) · 仍未完成 · 要人判

排程：P0，承接 #142 第24章，屬 #14 的執行子項。逐章追原始資料→typed事件→規則→正常鍵盤→戰後→持續隊伍→存檔／結局，不另設一份章完成台帳。

優先重用已閉合RE與既有引擎；遇到實際差異才開窄RE。25／26章剩餘回合事件與 #33 一起處理。遵守111／114的槽、seed與注入限制，不刷長局、不鎖HP；最後一章以實際結局／終止邊界驗收，沒有城鎮就不虛造城鎮交易。

卡在：先完成 #142 第24章；若揭露玩家阻擋，與其所屬規則／事件Issue一起修正。

怎樣算做完：parity-campaign-progress.json 的25～30章各有正式dosgolem來源、計畫、來源槽、四閘門與實際保存邊界；第30章有結局及隊伍狀態結果。失敗保留，既有passed章不重算。

### P0 補齊第1～3章正式驗收與新遊戲入口

`campaign-ch01-ch03-closeout` · 工作 · [#200](https://github.com/wicanr2/fd2_re/issues/200) · 仍未完成 · 要人判

排程：P0，與後段章驗收同屬 #14；後段完成後補齊，若新遊戲有玩家阻擋則立即處理。重用既有標題至第一章研究，不把舊RUNTIME-E1或不同來源收據直接提升為111。

從標題START正常輸入覆蓋開場／第一章；各章依111取得戰鬥抽樣、事件、戰後、城鎮與存檔。第1～3章剩餘回合事件由 #33 同步收斂。

怎樣算做完：正式台帳第1～3章均passed，四閘門及存檔來源完整；另有目前正式引擎標題START到第一章可操作的正常鍵盤收據，明示建構槽與注入限制。

### P1 HD正常路徑：早中晚期操作、戰場返回與存讀檔

`modern-theme-normal-player-validation` · 工作 · [#202](https://github.com/wicanr2/fd2_re/issues/202) · 仍未完成 · 要人判

排程：P1，#4 的執行期驗收子項，可先用已登錄素材做有界抽測。用目前正式Game從標題正常START／LOAD進場，抽早／中／晚期地圖、走行遮擋、游標、HUD、四向指令、取消、故事上下框、攻擊／施法返回、戰後商店／整備及存讀檔。

沿61核准範圍驗證現代圖層沒有被原版完整indexed幀蓋回，語意結果、座標、存檔與忠實主題相容。現代畫面驗收與忠實原版pixel parity分開。缺圖或未知consumer如實拒收，轉交素材子項；不以現有loader單測或總覽圖代替正常玩家收據。

怎樣算做完：受版控計畫及目前正式runtime的正常鍵盤／GPU收據涵蓋早中晚期與上述主要交界；存檔可冷讀回，忠實主題抽樣回歸通過；已登錄素材顯示與catalog一致，未驗範圍明示。

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

排程：P2，平台交付階段；保留原平台與實機驗收範圍，不因Linux結果宣稱其他平台完成。

三平台公開封包已由 CI 產出，Linux 經啟動與解包驗證。Windows 與 macOS 只做過 ZIP 內容驗證，沒有實機啟動、存檔與音訊抽測。

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

排程：P3，使用者於 2026-10-15 明確暫停。保留既有成果與原始驗收條件；只有使用者明確重啟才接續。

本題是工具鏈與原始碼保存研究，不是 Go／Ebiten 重製版、HD 主題、章節驗收或封包的前置條件。一般原版行為對拍繼續使用正式 dosgolem oracle；遇到具體玩家缺陷才做最小充分 RE，不重開全檔匹配。

使用者2026-10-08定案：完成固定版 FD2.EXE 的 matching decompilation，遊戲邏輯還原成 C；原作組語與第三方函式庫可保留，覆蓋分列。

主契約：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。原版固定357074 bytes，SHA-256 222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f。

已完成的前置試驗不等於全檔完成；不得以原版 EXE 複製或未知遊戲機器碼回填冒稱 C 還原。

卡在：使用者明確要求暫停；不得以 remake 尚未完成或個別畫面差異自行重啟。

怎樣算做完：從受版控來源及明示本機依賴乾淨建置完整 FD2.EXE，全檔大小及 SHA-256 相同；遊戲 C 函式逐一產碼一致；原作組語、第三方函式庫、資料與未知分列，未知不得冒稱已還原；正式 dosgolem 原版入口抽樣啟動。

證據：`docs/goal/118-goal-fd2-matching-decompilation-20261008.md`

### P1 原版收據連續延後截圖不得覆寫待輸出檔名

`oracle-deferred-frame-overwrite` · 缺陷 · [#206](https://github.com/wicanr2/fd2_re/issues/206) · 仍未完成 · 要人判

固定FD2.EXE、dosgolem 2c5c104的第24章ch24-sample-r3兩輪皆缺seq162 PNG。該點在runtime memcpy／11ED9→12103，frame_deferred=true；下一個force-enemy-clear只跑1步，capture把單一pendingFrame覆為163，162沒有輸出。原始失敗及完整保存留work/remake-recovery-20261015/ch24-original-r1與r2，主證據docs/data/fd2-remake-toolchain-recovery-20261015.json。

這是工具收據缺件，不是重製畫面錯誤。章計畫r4明示在mark後正常空等待200萬步，先完成原輸出再清敵；不補圖、不遮罩或放寬比較器。通用工具修正另走READY規格，保存每個待輸出名稱與其狀態邊界，禁止把較晚畫面冒稱同狀態。

怎樣算做完：受版控READY規格先於修正；相鄰兩個insideScreenWriter邊界的PNG都不丟失且來源／延後步數可追；保留CPU與輸入時序、不以中途畫面充數。正反向測試及固定原版重跑通過，既有前綴與畫面比較保持。

證據：`docs/data/fd2-remake-toolchain-recovery-20261015.json`

<!-- END fd2_worklist.py render -->
