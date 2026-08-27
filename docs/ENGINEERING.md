# FD2 重製工程入口

本文件承接原本堆在 GitHub 首頁的工程說明。首頁只保留專案定位、文化保存、
代表畫面與最短入口；會變動的證據與實作細節由此導向權威文件。

## 架構

- 唯一持續整合主線是 Go／Ebiten。
- 原版檔案只作唯讀行為基準，不納入散布物；玩家以合法原版在本機匯出資產。
- 戰役、對話、事件、商店、教會、整備、法術與部分原版處理器已轉成可編輯
  JSON／具型別資料，正式執行期不以 EXE 位址作永久捷徑。
- 引擎採潔淨室實作；未證實語意失敗即關閉，不以猜測補過場或畫面。

每個新切片依固定順序完成：

```text
原版證據 → RE 知識庫 → 可審查規格 → 具型別資料 → runtime／UI／存檔 → 抽樣驗證
```

詳細系統契約與證據分級見
[`56-fd2-remake-sdd.md`](knowledge-base/56-fd2-remake-sdd.md)，原版位址與正式
消費端覆蓋見 [`58-fd2-exe-re-coverage.md`](knowledge-base/58-fd2-exe-re-coverage.md)。

## 主要目錄

| 路徑 | 用途 |
|---|---|
| `remake/` | Go／Ebiten 重製引擎、可編輯資料、封包腳本與測試 |
| `tools/` | 資產匯出、驗證、反組譯與 Docker 工具鏈 |
| `docs/knowledge-base/` | 原版格式、控制流、設計規格與證據矩陣 |
| `docs/data/` | 可重生的位址、追蹤、差分與測試旁車 |
| `docs/figures/` | 已標示證據等級的研究圖與執行期畫面 |
| `org_game/`、`extracted/` | 本機原版與衍生輸出；不加入版本控制 |

## 建置與測試

所有建置、測試、反組譯與抓圖都在 Docker 內執行，主機不安裝 Go、Capstone、
IDA 或遊戲相依項目。最短可維護入口見 [`remake/README.md`](../remake/README.md)
與 [`tools/docker/`](../tools/docker/)；Linux／Windows／macOS 封包契約見
[`41-packaging.md`](knowledge-base/41-packaging.md)。

## 驗證原則

- 第一輪重製採分層代表性抽樣，不要求 DOS 硬體逐週期或全畫面逐像素考古。
- 原版／重製比較必須標出狀態、輸入、資產、計時條件及 E0／E1／E2 等級。
- 除錯直達、修改路徑及第三方存檔可以縮小問題，但不能冒充一般玩家路徑。
- 已閉合反組譯只有在雜湊改變、原始指令反證、同狀態矛盾或主證據缺失時重開。
- 全戰役長程遊玩改由人工回報；每個問題建立窄重現案例，不再以全量自動回歸
  消耗收尾時間。

目前抽樣完成門檻與尚餘工作見 [`REMAKE-STATUS.md`](REMAKE-STATUS.md)。
具型別抽樣清冊由[`check_remake_samples.py`](../tools/check_remake_samples.py)驗證；
不得直接修改`qualifies`把除錯、修改路徑或缺少重製消費端的旁證灌入完成數。
