# 91 — FD2 remake 目前未完成項

未完成項的唯一權威是 GitHub Issues（[`wicanr2/fd2_re`](https://github.com/wicanr2/fd2_re/issues?q=is%3Aissue+label%3Aworklist)，
標籤 `worklist`）。[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是
`tools/fd2_worklist_issues.py pull` 拉下來的快照；下面那一節由
[`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 從快照產生，**不要手改**，
改了下一次 pull／render 就會被蓋掉。

每一條掛一個 `verify`：跑起來為真代表這一條仍然未完成，為假就是東西做好了而條目
沒改。這樣「動手前先確認那條還成不成立」不必靠人記得。

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

共 16 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

## data — 可編輯資料還沒就緒

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 工作 · [#4](https://github.com/wicanr2/fd2_re/issues/4) · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 工作 · [#6](https://github.com/wicanr2/fd2_re/issues/6) · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

### 回合事件處理器只轉寫了 spawn：其餘章的 AI 模式改寫、鏡頭、演出與對白待逐章轉寫

`turn-event-handlers-partial-transcription` · 缺陷 · [#33](https://github.com/wicanr2/fd2_re/issues/33) · 仍未完成 · 要人判

FDFIELD回合事件由完整具型別處理器降成正式劇本，保留增援、AI模式、鏡頭、演出與對白；不得只保留spawn。原始表為0x51B91，工具為tools/extract_native_death_events.py與tools/sync_native_turn_events.py。

2026-10-06現況：全域--check通過，仍有27筆未轉寫。原先「其餘57筆、只有第四／五章」是歷史快照，不能作目前待辦。現已轉寫的章由工具直接列出；其餘依111正常章收據逐章處理，不先批次替換。

本輪補第十七章event40：group2後鏡頭(17,37)與text1三句。固定EXE／IDA9.4全handler與共享尾段經READY審查，再由正式lower同步到scenarios及canonical。正常Game確實消費原先缺失三句，124筆AI零順序分歧，98張畫面與酒店SAV保持。#195／#196工具補正後同來源四gate皆通過。既有第17章PLAYER-E2保持，未重新執行原版或新增章收據。

主契約：docs/data/ida/fd2_turn_event40_20261006.json；唯一分層現況見58。剩餘未轉寫列分布於第1／2／3／21／22／24／25／26章，本題維持開啟。

怎樣算做完：tools/sync_native_turn_events.py --check（不帶 --chapters）通過，報告裡「尚未轉寫的回合事件」為 0，且每一章的擴充都有該章 111 收據。

## runtime — 還沒接進正式執行期

### 攻擊演出收尾、0x1DEAE 與訊息關框三類重繪還沒經過 HUD anchor 入口

`hud-anchor-attack-and-message-redraws` · 缺陷 · [#41](https://github.com/wicanr2/fd2_re/issues/41) · 仍未完成 · 自承還在 docs/knowledge-base/56-fd2-remake-sdd.md

#40 把 anchor 評估收成單一入口 `redrawNativeMapHUD`（游標鍵步、`0x12CEA` 開頭與逐步、`0x1A30B` 返回後、戰場進場），第十章章收據 268 個畫面點全部一致。還沒接的是三類原版 `0x11CAC` 呼叫點：攻擊演出收尾（`0x1CFF0` 的 `0x1D3FF`、`0x1548E` 的 `0x15510`／`0x1563B`／`0x1565C`）、`0x1DB65` 的 `0x1DEAE`、訊息關框 `0x19742`。呼叫點清單在 `docs/data/ida/fd2_redraw_callers_hud_gates_ida.txt`，對照表在 `docs/knowledge-base/56` §#40（那三列標「未接」）。它們對應到重製端哪一段整幀重組、當時閘 B 與可見游標是什麼，都還沒逐點證明；在拿到收據之前不要為了收斂差異自行加評估點。

2026-09-18 定案：不為了這條重跑已 passed 的章；等之後排程到的新章收據順帶驗收。

2026-10-06 部分接線：死亡最後steady candidate與已證實196CB戰場END／獎勵closing，在原版指定時點經redrawNativeMapHUD評估。局部紅／綠測試及正常第24章同槽原生T5前綴四閘門已通過，54畫面點完整0px；存檔未抽測，不增加章PLAYER-E2。兩次原版2076份JSON及PNG bytes相同。玩家1D3FF已由IDA與窄正常指令收據更正定位為演出前重繪，15510屬簡略演出且正式仍拒收。物理1563B／1565C、玩家指令與劇情closing仍待接齊，本案保持開啟。來源、命令及限制：docs/data/ida/fd2_hud_redraw_20261006.json。

怎樣算做完：以原版 eip-trace（或同狀態擷取）證明這三類重繪發生的時機與當時的閘門／可見游標，接到同一個入口。不為了驗收重跑已 passed 的章（使用者 2026-09-18 定案）：以之後正常排程的新章收據順帶確認即可，沒有出現對應時點就在 56 §#40 記錄仍未被覆蓋。

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 戰後節點缺完整玩家路徑驗收

`campaign-postbattle-e2-full` · 工作 · [#14](https://github.com/wicanr2/fd2_re/issues/14) · 仍未完成 · 還沒出現

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以同狀態走過戰前對白、戰鬥節拍、戰後節點、城鎮與存檔邊界並與原版比較。

2026-09-15 決議（取代 2026-08-23 的「交人工遊玩回報」）：由代理程式依 [`111` 目標提示詞](https://github.com/wicanr2/fd2_re/blob/main/docs/goal/111-goal-original-parity-campaign-20260915.md) 用 dosgolem 逐章推進 30 章。起點用受版控工具依攻略校準的建構槽跳關，章內全程正常鍵盤輸入，抽樣完戰鬥節拍後以 `force-enemy-clear` 進戰後節點；這條路徑視同該章 `PLAYER-E2`。四個 gate（行為、節點、交易零容忍；代表畫面 ≤1% 差異像素）都過才算一章完成，進度台帳是 `docs/data/parity-campaign-progress.json`。

怎樣算做完：30 章各自依 111 的章工作單元取得原版／重製收據，四個 gate 全過；parity-campaign-progress.json 的 all_chapters_passed 為 true。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 工作 · [#15](https://github.com/wicanr2/fd2_re/issues/15) · 仍未完成 · 要人判

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

### 第十二章地圖 tile27 的 mode3 單像素殘差

`ch12-hud-terrain-single-pixel` · 缺陷 · [#52](https://github.com/wicanr2/fd2_re/issues/52) · 仍未完成 · 要人判

2026-10-01 定位勘誤：目前正式 treasure-original-r2／treasure-formal-r2 收據的17個差異點，均位於地圖 tile27 的局部像素(7,9)，不是HUD地形縮圖。13張完整畫面只差1px，另4張與指令環差異共存；完整四gate仍通過，本項維持可選像素修飾、不阻塞111。

原版IDA 9.4既有解碼契約核對：0x11EEE的0x1220C／0x12220分別呼叫raw 0x4DEDA及LUT 0x4DCC6。mode3在raw路徑保留目的底色，在LUT路徑讀目的像素並映射。每張殘差的實際分支及底色寫入來源尚未知，不能用archive byte+3推論runtime分支，也不能改成固定色。

主證據：docs/data/ida/fd2_terrain_mode3_review_20261001.json（輸入雜湊、工具／位址空間、原始bytes、分級與17點影像雜湊）。舊HUD解釋保留於Issue歷史與專案交接，新增證據以未遮罩座標及camera→world tile對照否定該定位；不宣稱根因已閉合。

2026-10-06 原生來源勘誤：使用正式native近堆模式、同固定槽及受版控六回合前綴，原版seq3146 attack_result在(11,85)為索引118，正式Game為116，完整差異仍1px。舊adapter索引0只解釋舊收據，不能宣稱目前差異僅由工具清零造成。253畫面點中246點完整0px，整體行為／節點／畫面比較仍未通過，跨同步HP與unknown UI限制保留。下一步只追必要copy plane／map work的writer，不填固定色或改容差。來源、命令及限制：主契約native_profile_recheck。

怎樣算做完：定位單像素來源的原生writer／繪圖契約，依READY規格修正並以相同狀態點驗證，不遮蔽或調預算。

### 第二十四章整章原版／重製對拍與收尾

`ch24-full-chapter-parity` · 工作 · [#142](https://github.com/wicanr2/fd2_re/issues/142) · 仍未完成 · 要人判

接續已驗收第23章非城鎮記錄SAV，SHA256 22bd63070189e0e6703e14d4bd4db8dc1f66452a21128dd5154579171ecbea63。依111／114例外，保持同一槽與seed4，不新增強化、治療、金幣或道具。先核對正常整備／選人／起手與58已閉合來源，再正常抽樣章內節拍、清敵一次、戰後與實際保存。遇反證另立缺陷／RE待解，READY先於修正，不放寬四gate。

怎樣算做完：['固定EXEhash、dosgolem commit、同源SAV與受版控計畫／清冊，正常章路徑可重跑。', '四gate與完整SAV通過，相關Go及第23章回歸通過。', '同步56／57／58、索引、正式台帳與產生現況，保留注入、敗北及未抽範圍。']

證據：`['docs/data/ui-traces/parity-ch23.json', 'docs/goal/111-goal-original-parity-campaign-20260915.md', 'docs/data/parity-campaign-progress.json']`

## release — 發行、平台與封包

### 網頁版前景玩家路徑待最終自動驗收

`wasm-web-release` · 工作 · [#7](https://github.com/wicanr2/fd2_re/issues/7) · 仍未完成 · 還沒出現

建置、資產與存檔都已通過；剩餘缺口是前景瀏覽器中的開場至第一關操作，以及重新載入後的存檔讀回。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式在 Docker／Xvfb 啟動非 headless 瀏覽器，以 CDP 將頁面帶到前景並自動送正常輸入、截圖與檢查 localStorage；不再要求使用者人工確認。若 Chromium 仍把頁面標成 hidden，先修正可重現的前景瀏覽器工具鏈，不得把 hidden 分頁結果冒充通過。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker／Xvfb 的非 headless Chromium 中由 CDP `Page.bringToFront` 確認 `visibilityState=visible`，從目前 WASM 產物正常走到第一關可操作，完成一次存檔、重新載入與讀回；保存輸入、畫格、console、localStorage 前後值與截圖到 docs/data/ui-traces/wasm-browser-player-path-e1.json，並由自動測試驗證。

### Android 封包沒有建置

`android-package` · 工作 · [#8](https://github.com/wicanr2/fd2_re/issues/8) · 仍未完成 · 還沒出現

觸控輸入已支援，但沒有 `ebitenmobile bind` → `.aar` → APK 的建置流程。

怎樣算做完：產出可安裝的 APK，並在實機確認啟動、存檔與音訊。

### Windows 與 macOS 還沒有實機抽測

`release-platform-acceptance` · 工作 · [#9](https://github.com/wicanr2/fd2_re/issues/9) · 仍未完成 · 要人判

三平台公開封包已由 CI 產出，Linux 經啟動與解包驗證。Windows 與 macOS 只做過 ZIP 內容驗證，沒有實機啟動、存檔與音訊抽測。

怎樣算做完：兩個平台各自在實機完成啟動、存檔／讀檔與音訊抽測並記錄結果。

## tooling — 工具、編輯器與工作流程

### 戰場編輯器待最終自動端到端驗收

`battlefield-editor-mvp` · 工作 · [#10](https://github.com/wicanr2/fd2_re/issues/10) · 仍未完成 · 還沒出現

戰場編輯器已具備地圖與單位編輯、部署格、波次總覽、復原及格式保真測試；缺的是由瀏覽器編輯正式複本、存回、重生 canonical，再由正式 runtime 載入的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做，屆時由代理程式以瀏覽器自動化操作可丟棄複本並驗證，不再要求使用者人工畫圖或授權目錄。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker 中啟動 editor server 與非 headless 瀏覽器，對一張原版地圖的可丟棄複本執行圖塊、單位與部署格各一項編輯，經正式 save API 存回並重生 canonical；正式 runtime 載入後核對地形成本、單位、部署格與畫面，最後還原來源零差異。保存 docs/data/ui-traces/battlefield-editor-roundtrip-e1.json 並由自動測試驗證。

### 劇情編輯器待最終自動端到端驗收

`campaign-editor-ui` · 工作 · [#11](https://github.com/wicanr2/fd2_re/issues/11) · 仍未完成 · 還沒出現

劇情編輯器五個分頁與節點圖均已可用，現有原版路線 round-trip 已通過；缺的是建立一條原版沒有的敗北／choice 路線，再由正式引擎走通首尾的端到端收據。

2026-09-14 決議：本項延到其餘非「最後處理」worklist 完成後才做。屆時由代理程式以瀏覽器自動化建立可丟棄測試路線並驅動引擎驗收，不再要求使用者人工設計或試玩。

卡在：依使用者 2026-09-14 裁定，排在所有非「優先級:最後」worklist 完成之後。

怎樣算做完：其餘非最後處理 worklist 完成後，在 Docker／非 headless 瀏覽器中用編輯器對可丟棄複本新增一條 battle.on_lose 敗北路線與一個依旗標過濾的 choice 分支，透過正式 save API 存回、重生 canonical，並由正式引擎以決定性輸入分別走通兩個分支首尾；還原來源零差異。保存 docs/data/ui-traces/campaign-editor-custom-route-e1.json 並由自動測試驗證。

### 為單一地形像素補正式 oracle 的有界唯讀記憶體變更收據

`oracle-bounded-memory-change-observation` · 工作 · [#197](https://github.com/wicanr2/fd2_re/issues/197) · 仍未完成 · 要人判

父項#52。正式native第12章seq3146像素仍118對116，現有EIP／map_state只有配置回傳與16項全域，沒有原始目標byte的改變來源；不能用位址重用或舊adapter清零推定最後writer。只擴充既有受版控apps/fd2/cmd/oracle，預設停用、宣告固定最多16個LE線性byte位址、既有追蹤窗口及筆數上限，每次成功instruction後唯讀觀察，記錄baseline或value change與原始指令定位。未變同值寫入不宣稱已觀察；不改CPU、heap、RNG、鍵盤、畫面或遊戲檔案。輸出留本地work，公開庫只存來源／收據雜湊。READY規格先於實作，wrapper明示參數與來源審查。

2026-10-06 READY工具實作已由dosgolem fa25f6e7d7f6be399b6563b7df6a55995daba77b提交推送，真正遠端相同。1058項平台測試零失敗零略過；原版百萬步啟用／停用的完整報告及PNG一致，除新增觀察欄位及輸出路徑。53筆觀察含52次變更。FD2兩層wrapper傳遞旗標並綁來源清冊，CPU／heap來源保持。BOOT時目標pixel尚未配置的false收據保留，不使用0代替；正向BOOT使用當時已存在全域／stack。首次fixture漏DS descriptor已修正並乾淨重跑。正常短槽已通過：233份JSON、232張PNG、輸入與EIP追蹤逐檔相同。FD2驅動器0734838e已推送。第12章目標觀察於2026-10-06依使用者要求暫停，保存到第1回合control_seq1110、2,900,397,260步；1111份JSON與1111張PNG與既有前綴相同，尚未到byte觀察窗口。停止程序143不是產品失敗，無target writer新結論。本案保持開啟，下次以同槽／計畫及新輸出與新覆蓋層重跑，保留中止收據。唯一來源／命令／雜湊：地形主契約memory_observation。

怎樣算做完：bounded參數與越界拒收、disabled原metadata保持、原始記憶體／CPU不變；同固定槽native正常計畫重生，既有完整checkpoint／PNG／controls一致後確認原生byte變更定位，不用新觀察放寬#52比較。

證據：`['docs/data/ida/fd2_terrain_mode3_review_20261001.json#native_profile_recheck', 'tools/dosgolem_oracle_container.sh']`

### 完成 FD2 全檔位元組一致反編譯與遊戲 C 原始碼覆蓋

`fd2-full-matching-decompilation` · 工作 · [#198](https://github.com/wicanr2/fd2_re/issues/198) · 仍未完成 · 要人判

使用者2026-10-08定案：完成固定版 FD2.EXE 的 matching decompilation，遊戲邏輯還原成 C；原作組語與第三方函式庫可保留，覆蓋分列。

主契約：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。原版固定357074 bytes，SHA-256 222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f。

已完成的前置試驗不等於全檔完成；不得以原版 EXE 複製或未知遊戲機器碼回填冒稱 C 還原。

怎樣算做完：從受版控來源及明示本機依賴乾淨建置完整 FD2.EXE，全檔大小及 SHA-256 相同；遊戲 C 函式逐一產碼一致；原作組語、第三方函式庫、資料與未知分列，未知不得冒稱已還原；正式 dosgolem 原版入口抽樣啟動。

證據：`docs/goal/118-goal-fd2-matching-decompilation-20261008.md`

<!-- END fd2_worklist.py render -->
