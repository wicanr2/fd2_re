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

## data — 可編輯資料還沒就緒

### 四語內容還有機器初稿沒有人工審校

`locale-machine-draft` · 仍未完成 · 自承還在 remake/assets/locales/en/content.json

繁中全部是來源語，簡中、日文與英文仍有標 `machine_draft` 的條目。沒有人工審校的譯文不得冒稱正式翻譯。另有一批說話者名稱源自單字或 `?`，屬來源身分阻擋，不是未翻譯，不可為了全綠而猜定。

怎樣算做完：三語的 `machine_draft` 全部經人工審校改為定稿，或明確拆出「來源身分阻擋」另計。

### 現代美術主題仍是原型狀態

`modern-theme-prototype-status` · 仍未完成 · 自承還在 remake/assets/themes/modern/catalog.json

現代主題 catalog 目前登錄 `"status": "prototype"`。逐組審查頭身比例、原版配色、輪廓與戰場尺寸可讀性尚未走完，早／中／晚期地圖的接縫、前景遮擋、人物、游標與 HUD 也還沒抽測。

怎樣算做完：catalog 轉為正式狀態，且抽測涵蓋早／中／晚期地圖與戰場尺寸可讀性。

### 素材清冊還有 93 筆 unknown 沒有交叉核對

`asset-manifest-unknown-dispositions` · 仍未完成 · 要人判

manifest v2 的 `source_resources` 有 1,005 筆，其中 `disposition` 為 `unknown` 的有 93 筆。正式 `Game` caller 稽核沒有發現它們有直接 archive consumer，所以不阻擋第一版；要確認的是有沒有尚未登記的玩家 consumer。

怎樣算做完：93 筆逐筆確認沒有玩家 consumer，或找到 consumer 後補 provenance 並重開對應切片。

### 四筆角色身份歧義還沒解

`editor-identity-ambiguity` · 仍未完成 · 要人判

`native-0`／`native-1`／`native-7`／`native-96` 有名稱衝突，目前保留直接來源、拒絕猜選。

怎樣算做完：四筆各自解決或明確拆分，並在 canonical schema 記錄依據。

## runtime — 還沒接進正式執行期

### 增益走重製端的共用計時器，不是原版的六個暫時狀態

`remake-buff-uses-normalized-timer` · 仍未完成 · 自承還在 remake/internal/battle/magic.go

原版六個暫時狀態各有各的剩餘回合數（`+0x22..+0x27`，由 `0x1A866` 逐一遞減），`sub_1B750` 只看「非零與否」決定要不要套固定效果：`+0x22` 攻擊力 ×1.15、`+0x23` 防禦力 ×1.15、`+0x24` 命中與迴避各 +15（見 107）。重製端的 `EffectiveAP()`／`EffectiveDP()` 走可累加的 `BuffAPPct`／`BuffDPPct` 加一個共用的 `BuffTurns`（`applyBuff` 自承是重製簡化）。數值對得上，但施兩次魔刃在重製端會變成 +30%，原版不會；六個狀態的到期時間在重製端也是綁在一起的。機制層（保存、遞減、回復閘門、法術停用、反擊資格）都已經走原版那條。

怎樣算做完：`EffectiveAP`／`EffectiveDP`／`EffectiveHIT`／`EffectiveEV` 在有原版 raw 的單位上改由 `+0x22..+0x24` 決定，效果照 `sub_1B750` 的固定倍率與加法，六個狀態各自到期；normalized 那條只留給沒有 raw 的舊劇本。

證據：`docs/knowledge-base/107-transient-status-semantics-20260910.md`

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

## release — 發行、平台與封包

### 網頁版還載不到資產

`wasm-web-release` · 仍未完成 · 自承還在 tools/build_wasm.sh

`tools/build_wasm.sh` 與 `.github/workflows/build-wasm.yml` 已經能產出可重現的 `fd2.wasm`（約 20 MB）、`wasm_exec.js` 與 `index.html`，WASM 目標也編得過。**還不能玩**：`assetGlob` 走 `filepath.Glob`，js/wasm 底下沒有檔案系統可掃，所以遊戲在缺資產時失敗即關閉。資產共 81 MB（其中 52 MB 是三套音樂），直接 `go:embed` 進 WASM 不切實際；HTTP 取用又沒有「列目錄」這回事，所以需要一層 `fs.FS` 抽象加一份受版控的資產清單。目前有 15 個檔案直接用 `os.ReadFile`／`filepath.Glob`／`os.Open`。

怎樣算做完：網頁版能載到資產並跑起來：資產存取走可替換的 `fs.FS`，wasm 端由清單驅動 HTTP 取用，並在瀏覽器確認開場到第一關可玩。

### Android 封包沒有建置

`android-package` · 仍未完成 · 還沒出現

觸控輸入已支援，但沒有 `ebitenmobile bind` → `.aar` → APK 的建置流程。

怎樣算做完：產出可安裝的 APK，並在實機確認啟動、存檔與音訊。

### Windows 與 macOS 還沒有實機抽測

`release-platform-acceptance` · 仍未完成 · 要人判

三平台公開封包已由 CI 產出，Linux 經啟動與解包驗證。Windows 與 macOS 只做過 ZIP 內容驗證，沒有實機啟動、存檔與音訊抽測。

怎樣算做完：兩個平台各自在實機完成啟動、存檔／讀檔與音訊抽測並記錄結果。

## tooling — 工具、編輯器與工作流程

### canonical 文件還沒有通往正式執行期的 compiler

`editor-canonical-runtime-compiler` · 仍未完成 · 要人判

版本化 bundle 已含 1 份 campaign、30 份 scenario、35 份 story 與 38 筆角色身份候選，deterministic exporter 與跨文件 validator 都已接。缺的是讓正式戰役規則直接消費編輯後文件的那一段。

怎樣算做完：至少一條正式 runtime 路徑直接讀 canonical 文件，且 load→write→reload 不遺失資料。

### 戰場編輯器還沒有

`battlefield-editor-mvp` · 仍未完成 · 要人判

計畫是網頁單檔工具：圖塊繪製、單位擺放、部署格，讀寫 `assets/maps`。

怎樣算做完：能開啟一張原版地圖、改動後存回，並由正式 runtime 讀得起來。

### 劇情與節點圖編輯器還沒有

`campaign-editor-ui` · 仍未完成 · 要人判

包含對白與事件表單、商店設定，以及 campaign 節點圖（拖線、旗標、敗北路線）的可視化編輯。

怎樣算做完：能編輯一個章節的對白與節點連線，存檔後由正式 runtime 走得通。

<!-- END fd2_worklist.py render -->
