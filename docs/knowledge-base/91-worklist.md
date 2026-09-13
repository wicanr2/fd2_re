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

共 15 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

## re — 原版證據還沒閉合

### 戰鬥對白 ESC 待 dosgolem 三臂自動對拍

`battle-dialogue-esc-receipt` · RE待解 · [#2](https://github.com/wicanr2/fd2_re/issues/2) · 仍未完成 · 還沒出現

故事對白已量到 ESC 與 Enter 完全同義（三臂逐格 SHA-256 相同），重製端已接。戰鬥事件對白與回合起手對白仍缺各自的 ESC／Enter 原版對照收據。

2026-09-14 決議：不再等待人工操作。由代理程式用 dosgolem 對兩個正式入口各跑 ESC、Enter 與無輸入反對照，依收據直接決定是否共用輸入契約；若不同則保留各自 typed 規則，不可猜接。

怎樣算做完：以 dosgolem 對戰鬥事件對白與回合起手對白各產生 ESC、Enter、none 三臂收據，固定起點、步數、鍵盤閘門與 EXE 雜湊；自動比較每臂的畫面 SHA-256、輸入消耗、控制邊界與後續狀態。相同就接線並測試，差異就分開實作並記錄；結果寫入 docs/data/ui-traces/battle-dialogue-esc-vs-enter.json，且 receipt 測試通過。

證據：`docs/knowledge-base/98-opening-esc-and-length-20260909.md`

### ch21／ch22 六筆增援來源待 RE 證據閉合

`reinforcement-eax-source` · RE待解 · [#3](https://github.com/wicanr2/fd2_re/issues/3) · 仍未完成 · 還沒出現

ch21／ch22 六筆增援的來源運算元是暫存器或間接記憶體，舊靜態掃描停在 `$reg_or_mem`，尚未追到 writer。

2026-09-14 決議：由代理程式以攻略只定位玩家可見觸發情境，再以既有 RE 文件、固定雜湊 IDA 9.4 資料庫的交叉參照與資料流閉合六筆來源；攻略不得單獨當 ABI 或欄位語意證據，也不再要求人工複核。

怎樣算做完：先由攻略與既有 battle event／chapter handler 文件列出 ch21、ch22 六筆增援的觸發情境，再用 IDA Pro 9.4 逐筆追到 eax 或間接來源的 writer、呼叫者與 consumer；高影響分支核對 raw bytes／jump table，間接 writer 不得只靠直接 xref。產出 docs/data/ida/fd2_reinforcement_eax_sources.json，逐筆保存原始名稱、線性位址、bytes、推論等級、證據出處與固定 EXE 雜湊；同步 58 與相關事件文件。六筆全閉合或誠實分級後由 schema／coverage 測試裁決，不需人工。

證據：`docs/data/ida/fd2_reinforcement_eax_sources.json`

## data — 可編輯資料還沒就緒

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 工作 · [#4](https://github.com/wicanr2/fd2_re/issues/4) · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 素材清冊還有 93 筆 unknown 沒有交叉核對

`asset-manifest-unknown-dispositions` · 工作 · [#5](https://github.com/wicanr2/fd2_re/issues/5) · 仍未完成 · 自承還在 docs/data/asset-disposition-summary.json

manifest v2 的 source_resources 有 1,005 筆，disposition 為 unknown 的 93 筆（FDFIELD.DAT 79、FDOTHER.DAT 9、FDMUS.DAT 5，reason_code 都是 no_standard_output）。正式 Game caller 稽核沒有發現它們有直接 archive consumer，所以不阻擋第一版；要確認的是有沒有尚未登記的玩家 consumer。清單與定位資訊在 docs/data/asset-disposition-summary.json，由 tools/summarize_asset_dispositions.py 從 pack 的 manifest 重生。

怎樣算做完：93 筆逐筆確認沒有玩家 consumer，或找到 consumer 後補 provenance 並重開對應切片。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 工作 · [#6](https://github.com/wicanr2/fd2_re/issues/6) · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 戰間介面待早中晚期自動原版對拍

`town-shop-ui-e2-sampling` · 工作 · [#13](https://github.com/wicanr2/fd2_re/issues/13) · 仍未完成 · 還沒出現

城鎮、商店、教會、整備與祕密商店的原版 E2 目前只覆蓋 ch02；早、中、晚期還缺同狀態抽樣。

2026-09-14 決議：不再等待人工操作。由代理程式以 dosgolem 作原版權威執行器，對早、中、晚期代表章節各取得正常玩家路徑收據，再由重製端重播等價輸入並自動比較。不得用 direct-entry、修改 HP 或鄰近畫面冒充 E2；若 dosgolem 缺能力，先補能力再重生收據。

怎樣算做完：選早期、中期、晚期各一章，以未修改正常玩家路徑進入城鎮，對商店、教會、整備與該章可達的祕密商店做風險導向抽樣；dosgolem 原版與重製端固定存檔／狀態、輸入、畫格、版本與素材雜湊，自動比較節點、交易、存檔邊界及代表畫面。結果寫入 docs/data/ui-traces/town-shop-early-mid-late-e2.json，三章 coverage 與 receipt schema／差異 gate 測試全通過才完成。

### 戰後節點缺完整玩家路徑驗收

`campaign-postbattle-e2-full` · 工作 · [#14](https://github.com/wicanr2/fd2_re/issues/14) · 仍未完成 · 要人判

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以未修改一般玩家路徑走過戰後節點、城鎮與存檔邊界。長程漂移依 2026-08-23 的決定改由人工遊玩後回報。

怎樣算做完：人工遊玩回報的缺陷各自建立窄重現案例並修掉。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 工作 · [#15](https://github.com/wicanr2/fd2_re/issues/15) · 仍未完成 · 要人判

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

### 原版側已到第四關續跑點，待第四關閉環收據

`parity-ch02-ch04-original-side` · 工作 · [#16](https://github.com/wicanr2/fd2_re/issues/16) · 仍未完成 · 還沒出現

既有 `work/parity-state/` 已保存 ch01、ch02、ch03 cleared 的原版 `FD2.SAV`；這證明原版側已完成第三關並至少到達第四關前的續跑點，舊標題「只走到第二關入口」已失效。受版控計畫目前仍只有 `ch01-clear.jsonl` 與 `ch02-clear.jsonl`，第四關的可重跑閉環序列與正式收據尚未登錄。

2026-09-14 決議：先稽核上次錄影／checkpoint 是否已涵蓋第四關；不足時允許 `FD2_ORACLE_LOCK_ALLY_HP=1` 鎖定原版所有我方 HP，並允許驅動器強制清場以完成通關。這是修改路徑，只能證明關卡節點、畫面、介面與存檔閉環，不得用來宣稱傷害、生存、戰鬥結果或一般玩家 `PLAYER-E2`。

怎樣算做完：先稽核 work/ 的既有錄影、runner、checkpoint 與 ch03-cleared/FD2.SAV；若已有第四關完整收據就驗證並登錄，否則以該存檔的可寫複本續跑。允許鎖定 camp 2 全體 HP 與強制清場，但 runner／每個 checkpoint 必須記錄 state_injections、強制清場方式、輸入、版本、原始素材雜湊與證據降級。完成 docs/data/parity-plans/ch04-clear.jsonl、第四關戰場→戰後→城鎮存檔的可重生收據，並在 96 記錄命令、輸入、日期及不可主張的範圍。

證據：`docs/knowledge-base/96-parity-toolchain-20260909.md`

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

<!-- END fd2_worklist.py render -->
