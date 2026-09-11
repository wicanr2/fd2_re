# 91 — FD2 remake 目前未完成項

未完成項的唯一權威是 [`docs/data/fd2-worklist.json`](../data/fd2-worklist.json)。
下面那一節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生，**不要
手改**；改了下一次 render 就會被蓋掉。

每一條掛一個 `verify`：跑起來為真代表這一條仍然未完成，為假就是東西做好了而條目
沒改。這樣「動手前先確認那條還成不成立」不必靠人記得。

```sh
tools/fd2_worklist.py verify   # 逐條檢查，發現訊號消失的條目就以 exit 1 收場
tools/fd2_worklist.py render   # 重寫下面的產生區塊
python3 -m unittest discover -s tools -p 'test_fd2_worklist.py'
```

條目做完就從 JSON 移走，不是在這裡打勾。

其他入口：

- 每一題的 RE／資料／正式執行期／E2 分層現況：[`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)
- 介面覆蓋率與尚未關閉的關卡：[`57-ui-evidence-matrix.md`](57-ui-evidence-matrix.md)
- 各輪的工作記錄、勘誤與位址出處：[`91-worklist-history.md`](91-worklist-history.md)
- 已閉合位址的「不要重做」索引在 `58`；重開條件是輸入雜湊不同、原始指令或跳表
  直接反證、同狀態執行結果矛盾，或主證據缺少它聲稱具備的 writer／consumer。

<!-- BEGIN fd2_worklist.py render；不要手改這一段 -->

共 18 條未完成項。權威是 [`docs/data/fd2-worklist.json`](../data/fd2-worklist.json)，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。

## re — 原版證據還沒閉合

### 故事場景對白的原版收據還沒取

`storybg-dialogue-receipt` · 仍未完成 · 要人判

戰場對白已確認原版不為對白換世界層，重製端照改。故事場景（`storyBG`）的對白仍走正規化管線，那條路徑沒有取過原版收據，所以不知道它與戰場對白是不是同一個契約。

怎樣算做完：取一段 storyBG 對白的原版逐幀收據，判定它是否也共用地圖層的重繪集合。

證據：`docs/knowledge-base/100-dialogue-viewport-20260909.md`

### 戰鬥中的對白接不接受 ESC 還沒量

`battle-dialogue-esc-receipt` · 仍未完成 · 要人判

故事對白已量到 ESC 與 Enter 完全同義（三臂逐格 sha256 相同），重製端已接。戰鬥事件對白與回合起手對白只取過 Enter／Space 的樣本。

怎樣算做完：對這兩處各取一次 ESC 與 Enter 的對照收據，相同就接線，不同就記錄差異。

證據：`docs/knowledge-base/98-opening-esc-and-length-20260909.md`

### ch21／ch22 增援的 eax 來源還沒追

`reinforcement-eax-source` · 仍未完成 · 要人判

六筆增援的來源運算元是暫存器或間接記憶體，靜態掃描停在 `$reg_or_mem`，沒有追到寫入端。

怎樣算做完：六筆各自找到寫入端與消費端，或明確記錄它們是同一個 producer 的不同分支。

### 死亡效果型態 3 與型態 2 的其他 id 還不可執行

`native-death-effect-unresolved` · 仍未完成 · 自承還在 tools/export_units.py

FDFIELD b22..b24 抄進 runtime `+0x31..+0x33`。型態 0（物品）、1（金幣）與型態 2 的 id 39／41 已資料化；型態 3 與型態 2 的其他 id 沒有閉合的 handler，地圖檔只保留原始 `death_effect`，不產生 `death_reward`。第一關就有兩筆：海盜頭目 `[3, 8]` 與哈諾 `[2, 4]`。原版進城金幣 1000 只說明頭目那筆不給金幣。

怎樣算做完：追到型態 3 與其餘型態 2 handler 的 writer／consumer，資料化成可執行獎勵或明確的無效果，並加測試。

證據：`docs/knowledge-base/109-title-to-town-journey-20260911.md §3`

## data — 可編輯資料還沒就緒

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 素材清冊還有 93 筆 unknown 沒有交叉核對

`asset-manifest-unknown-dispositions` · 仍未完成 · 自承還在 docs/data/asset-disposition-summary.json

manifest v2 的 source_resources 有 1,005 筆，disposition 為 unknown 的 93 筆（FDFIELD.DAT 79、FDOTHER.DAT 9、FDMUS.DAT 5，reason_code 都是 no_standard_output）。正式 Game caller 稽核沒有發現它們有直接 archive consumer，所以不阻擋第一版；要確認的是有沒有尚未登記的玩家 consumer。清單與定位資訊在 docs/data/asset-disposition-summary.json，由 tools/summarize_asset_dispositions.py 從 pack 的 manifest 重生。

怎樣算做完：93 筆逐筆確認沒有玩家 consumer，或找到 consumer 後補 provenance 並重開對應切片。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

## runtime — 還沒接進正式執行期

### 升級上限沒有實作

`native-level-cap` · 仍未完成 · 還沒出現

`0x1E2E0..0x1E2F2`：記錄 `+7` 為 0x1E／0x1F 時等級比 99，其餘比 40，相等就跳離升級處理。重製端 `gainExp` 沒有這道判斷，等級可以無限上升。第一關等級到不了上限，不影響目前的整段對照。

怎樣算做完：`gainExp` 依 `+7` 套上 40／99 上限並加測試；跳離時經驗是否仍累積要先讀 `0x1E2F2` 的跳躍目標再決定。

證據：`docs/knowledge-base/109-title-to-town-journey-20260911.md §5`

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 仍未完成 · 要人判

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### ch02 以外的戰間介面還沒抽樣

`town-shop-ui-e2-sampling` · 仍未完成 · 要人判

城鎮、商店、教會、整備與祕密商店的原版 E2 目前只覆蓋 ch02。早、中、晚期章節都還沒以正常輸入抽樣。

怎樣算做完：早、中、晚期各挑一章，以正常輸入取得原版與重製端的同狀態對照。

### 戰後節點缺完整玩家路徑驗收

`campaign-postbattle-e2-full` · 仍未完成 · 要人判

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以未修改一般玩家路徑走過戰後節點、城鎮與存檔邊界。長程漂移依 2026-08-23 的決定改由人工遊玩後回報。

怎樣算做完：人工遊玩回報的缺陷各自建立窄重現案例並修掉。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 仍未完成 · 要人判

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

### 原版側對拍只走到第二關入口

`parity-ch02-ch04-original-side` · 仍未完成 · 還沒出現

第一至三關都通關並在城鎮存了檔（work/parity-state/chNN-cleared/）；第四關用同一份 ch02-clear.jsonl 從第三關的續跑點起跑中。這條路徑上的每一輪都開著 FD2_ORACLE_LOCK_ALLY_HP（修改路徑），取得的收據不得當成 PLAYER-E2。

怎樣算做完：第二至四關各有一份可重跑的閉環序列與收據，並在 96 記錄命令、輸入與日期。

## release — 發行、平台與封包

### 網頁版還沒在前景瀏覽器由人確認可玩

`wasm-web-release` · 仍未完成 · 要人判

建置、資產與存檔都通了：`tools/build_wasm.sh` 產出 `fd2.wasm`、資產打包檔（43047 個檔案、124 MB）與索引；`index.html` 的檔案系統轉接層讀走資產包、寫走 localStorage，實測跑完 Go 的存檔流程（建立暫存檔→寫→關→rename→讀回），而且寫過的檔案會蓋過資產包。瀏覽器實測跑到漢堂國際的開場 logo。剩下的是**互動驗證**：自動化分頁的 `visibilityState` 是 hidden，瀏覽器會暫停 requestAnimationFrame，Ebiten 主迴圈因此不推進，所以「開場走到第一關可操作」只能在前景視窗由人確認。

怎樣算做完：在前景瀏覽器開 tools/build_wasm.sh 的產物，確認開場走到第一關可操作，存檔能寫入並在重新載入後讀回。

### Android 封包沒有建置

`android-package` · 仍未完成 · 還沒出現

觸控輸入已支援，但沒有 `ebitenmobile bind` → `.aar` → APK 的建置流程。

怎樣算做完：產出可安裝的 APK，並在實機確認啟動、存檔與音訊。

### Windows 與 macOS 還沒有實機抽測

`release-platform-acceptance` · 仍未完成 · 要人判

三平台公開封包已由 CI 產出，Linux 經啟動與解包驗證。Windows 與 macOS 只做過 ZIP 內容驗證，沒有實機啟動、存檔與音訊抽測。

怎樣算做完：兩個平台各自在實機完成啟動、存檔／讀檔與音訊抽測並記錄結果。

## tooling — 工具、編輯器與工作流程

### 戰場編輯器還沒有

`battlefield-editor-mvp` · 仍未完成 · 要人判

tools/editor/battlefield.html 已可用：圖塊筆刷／矩形／填充／橡皮擦、單位擺放與表單、部署格、波次總覽、復原，存回 map.json 與 mapN_units.json。移動成本換算與匯出管線一致（tools/test_editor_terrain_cost.py 逐格對照所有受版控地圖），存回的格式保真也有測試（tools/test_editor_server.py）。缺的是人實際畫一張地圖並在引擎裡載入——那是美術與關卡設計判斷。

怎樣算做完：人用編輯器開一張原版地圖、改動後存回，並由正式 runtime 讀得起來。

### 劇情編輯器缺人實際編一章玩過

`campaign-editor-ui` · 仍未完成 · 要人判

五個分頁都已可用：對白、戰場事件、商店品項、節點轉場，以及 doc 38 Phase 3 的節點圖（依章節分層、改轉場、旗標管理、敗北路線紅虛線、choice 選項的旗標條件）。32 個章節都畫得出來、零失敗。編輯來回已由 tools/editor/serve.py 走完並驗過：改 battle_ch01.on_win → 存回 → 重生 canonical → 引擎讀到新值 → 還原零 diff。缺的是**人實際做出一條原版沒有的路線並玩過**——那是設計判斷，不是機器驗得掉的。

怎樣算做完：人用編輯器做出一條非原版路線（至少含一條 battle.on_lose 敗北路線與一個依旗標過濾的 choice 分支），存回並重生 canonical 後在引擎裡跑通首尾。

<!-- END fd2_worklist.py render -->
