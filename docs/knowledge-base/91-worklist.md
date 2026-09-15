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

共 14 條未完成項。權威是 GitHub Issues（標籤 `worklist`），[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。

`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。
新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`](../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。

## re — 原版證據還沒閉合

### sub_112A5 加入紀錄的空物品格 item byte 原版是 0xff，轉寫是 0x00

`join-constructor-empty-cell-item-byte` · RE待解 · [#23](https://github.com/wicanr2/fd2_re/issues/23) · 仍未完成 · 自承還在 remake/cmd/fd2-chapter-slot/main.go

重製端 `campaign.MaterializePersistentRecord`（sub_112A5 轉寫）對物品格 6／7 只寫旗標 `+0x16`／`+0x18`＝0x80，item byte `+0x17`／`+0x19` 留 0；真實原版存檔（ch01-cleared 剛加入的 id 8、ch02-cleared 同一筆）這兩格是 `80 ff`。四格 defaults 那邊，defaults 為 0xff 時旗標寫 0x80、item 寫 0xff，與觀察一致；只有固定的兩格不同。消費端只看旗標 bit7，所以玩法不受影響，但建槽工具的輸出與原版 bytes 差這兩個 byte。要回 IDA 看 0x112A5 是否另有寫 `+0x17`／`+0x19`＝0xff 的指令，或紀錄區在 JOIN 前被 0xff 填過。

怎樣算做完：IDA 9.4 直接指令證實 +0x17／+0x19 的來源（明寫 0xff 或前置填充），轉寫與建槽工具同步修正，正對照這兩個 byte 歸零。

## data — 可編輯資料還沒就緒

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 工作 · [#4](https://github.com/wicanr2/fd2_re/issues/4) · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 四筆角色身份的多名稱已分類，剩人複核

`editor-identity-ambiguity` · 工作 · [#6](https://github.com/wicanr2/fd2_re/issues/6) · 仍未完成 · 自承還在 remake/assets/editor-canonical/character-identity.json

native-0／native-1／native-7／native-96 的多個候選名稱已由資料本身分成兩類：章節互不重疊的是同一身份在不同段落的稱呼（索爾／索爾(少年)、刺客／蘭斯洛特），同章且共同前綴加單一編號的是多個雜兵共用一個 sprite（強盜 B/C/L/M/N）。診斷都帶著章節依據，severity 從 error 降為 note，canonical 已無未分類衝突。剩下的是人複核那個分類對不對——判準是從資料算的，不是從劇情知識來的。

怎樣算做完：人複核四筆的分類；若有誤判就調整判準並重生 bundle。

## runtime — 還沒接進正式執行期

### 酒店存檔沒有寫原版 FD2.SAV 章節槽 bytes

`hotel-save-native-slot-bytes` · 缺陷 · [#24](https://github.com/wicanr2/fd2_re/issues/24) · 仍未完成 · 自承還在 docs/data/ui-traces/parity-ch04.json

原版酒店存檔（0x30012）把 32 筆 0x50 持續紀錄與 +0..+9 metadata 寫進 FD2.SAV 的四槽區（0x312B + slot×0xA28），標題 LOAD 讀回同一區。重製端 `saveGameToSlot` 只寫自有 JSON；只有戰場系統選單 SAVE 走 `buildNativeCurrentSaveStored` 寫 current 區。111 的交易 gate 要比兩側酒店寫出的槽 bytes，目前重製側沒有可比的輸出，第四章起每章收據的存檔項都會 blocked。要做：以 `campaign.BuildNativeCurrentPersistentRecords` 同一套紀錄投影＋`fdsave.WriteSlot`／`Encode` 在酒店存檔時同步寫原版四槽（FD2_NATIVE_SAVE 有指定時），metadata +0 chapter、+1 count、+2..+5 gold、+6..+9 依 0x30012 的 writer。

怎樣算做完：酒店存檔後 FD2_NATIVE_SAVE 的對應槽 bytes 與原版同狀態存檔相同（第四章收據交易 gate 的 save 項由 blocked 轉 ok）。

### 第四章敵方回合 mode 0 移動終點與原版分岔（17 個中 11 個）

`enemy-mode0-destination-divergence-ch04` · 缺陷 · [#25](https://github.com/wicanr2/fd2_re/issues/25) · 仍未完成 · 還沒出現

111 第四章對拍：同一份建構槽、同一回合（第 1 回合我方全員不動、END），原版 17 個 group 1 敵人的第 1 回合終點與重製端只有 6 個相同。修正 MV（見同批提交：Load 時以 0x10C50 建構器 record 的移動力覆寫 authored `mv`）之後仍差 11 個，型態像是同分候選的掃描順序或先動單位佔格後的連鎖：例如 index 8 (9,1) 原版到 (8,3)、重製端到 (9,4)；index 9 (10,2) 原版到 (9,4)、重製端到 (8,3)，兩者對調。所有敵人 +0x34=0（mode 0）、+0x3b=4/3。原版逐單位終點在 `docs/data/ui-traces/parity-ch04.json` 的 behavior gate 點（after_enemy_phase）與 `work/parity-slot-ch04/sample-r*/checkpoint-*.json`。要回 `0x14121`／`0x13E9C`／`0x4E1A6` 對照 native_ai_destination.go 的候選排序與佔格更新順序。

怎樣算做完：第四章收據 after_enemy_phase 第 1 回合 17 個敵人終點全部相同（behavior gate 該點 ok）。

### 敵方施法時 native AI command damage target array is empty 失敗即關閉

`ai-command-damage-empty-target-array` · 缺陷 · [#26](https://github.com/wicanr2/fd2_re/issues/26) · 仍未完成 · 自承還在 docs/data/ui-traces/parity-ch04.json

111 第四章重播：第 3 回合敵方回合（法師 fig 90／93 在場）重製端執行期錯誤 `native AI action: native AI command damage target array is empty`（remake/internal/battle/native_command0.go PlanNativeAICommandDamage）。0x14EF0 路由已選定 command 與目的地，但執行時以 actor 的 NativeMapPresentation 座標重建目標陣列得到空集合；懷疑移動與施法之間 presentation 座標未更新，或目的地選擇與目標判定用了不同座標。原版同一回合正常施法（第四章 r2 收據悠妮被火炎術打中的畫面）。重現：`tools/chapter_parity.sh 4 work/parity-slot-ch04 <sample-run> <out>`，replay.log 最後一筆 runtime_error。

怎樣算做完：第四章重播能走完敵方回合不出 loadErr；收據不再出現 runtime_error 點。

## player — 缺未修改一般玩家路徑的驗收（PLAYER-E2）

### 戰鬥交易缺代表性玩家路徑驗收

`battle-transaction-e2` · 工作 · [#12](https://github.com/wicanr2/fd2_re/issues/12) · 仍未完成 · 要人判

正常 producer 的物品、command 與敵方 AI 都已達 `RUNTIME-E1`。缺的是代表性玩家與敵方回合的精確演出、音訊與未修改一般玩家路徑的 `PLAYER-E2`。

怎樣算做完：挑代表性的玩家與敵方回合，以未修改路徑取得同狀態逐幀與音訊對照。

### 戰後節點缺完整玩家路徑驗收

`campaign-postbattle-e2-full` · 工作 · [#14](https://github.com/wicanr2/fd2_re/issues/14) · 仍未完成 · 還沒出現

已綁定的章節各自有窄 `RUNTIME-E1`，但沒有每一章都以同狀態走過戰前對白、戰鬥節拍、戰後節點、城鎮與存檔邊界並與原版比較。

2026-09-15 決議（取代 2026-08-23 的「交人工遊玩回報」）：由代理程式依 [`111` 目標提示詞](https://github.com/wicanr2/fd2_re/blob/main/docs/knowledge-base/111-goal-original-parity-campaign-20260915.md) 用 dosgolem 逐章推進 30 章。起點用受版控工具依攻略校準的建構槽跳關，章內全程正常鍵盤輸入，抽樣完戰鬥節拍後以 `force-enemy-clear` 進戰後節點；這條路徑視同該章 `PLAYER-E2`。四個 gate（行為、節點、交易零容忍；代表畫面 ≤1% 差異像素）都過才算一章完成，進度台帳是 `docs/data/parity-campaign-progress.json`。

怎樣算做完：30 章各自依 111 的章工作單元取得原版／重製收據，四個 gate 全過；parity-campaign-progress.json 的 all_chapters_passed 為 true。

### 曲號與音效還需要人耳確認

`bgm-sfx-listening` · 工作 · [#15](https://github.com/wicanr2/fd2_re/issues/15) · 仍未完成 · 要人判

開場配樂曲號、戰鬥曲與勝利曲的對應、以及 UI 音效 index 2..0xb 的語意畫面，都要實際聽辨才能定案。容器內無音訊裝置，驗不了。

怎樣算做完：逐項聽辨後修正曲號對映與音效語意記錄。

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
