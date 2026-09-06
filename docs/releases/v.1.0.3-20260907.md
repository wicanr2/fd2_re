# v.1.0.3-20260907 發行說明

這是《炎龍騎士團 2》潔淨室重製的第一個穩定里程碑。它延續前三個保留但未建立
GitHub Release 的不可移動標籤，修正私人素材邊界、跨平台供應鏈、Windows 換行，
以及完整原版素材遮蔽現代戰場圖示的問題。版本字串、Git tag、GitHub Release、
程式建置資訊、封包檔名與 `dist-all/` 目錄一律使用 `v.1.0.3-20260907`。

## 可公開附件

- `FD2-v.1.0.3-20260907-linux-x86_64.AppImage`
- `FD2-v.1.0.3-20260907-windows-x86_64.zip`
- `FD2-v.1.0.3-20260907-macos-universal.zip`
- `FD2-v.1.0.3-20260907-promo.mp4`
- `SHA256SUMS`
- `MANIFEST.json`

公開附件只包含重製引擎、RRSAL-1.0 授權、可公開資料與現代主題 catalog，不包含
原版素材或私人現代逐幀 PNG。玩家必須自備合法原版；私人現代素材缺件時，`F2`
保持目前主題並顯示原因。

## 本機完整版

`dist-all/v.1.0.3-20260907/full/` 保存真正含完整遊戲資料與現代 runtime 素材的
Linux AppImage、Windows ZIP 與 macOS universal ZIP。這些封包只供本機／已授權
私下使用，標示 `public_distribution: false`，不得加入公開 Release。

## 玩家入口

- `F1`：暫停式操作說明；`Esc` 或再次按 `F1` 關閉。
- `F2`：忠實原版／現代手繪主題；只替換繪圖資源，不重設戰況。
- `F3`：Sound Blaster／MT-32；`F4`：四種官方語言。
- `F5`：快速存檔；`F6`：音樂開關；`F9`：快速讀檔；`F12`：除錯資訊。
- 戰場系統選單仍提供音樂、音效、速度與狀態欄按鈕；現代 HUD 與攻擊／法術／
  物品／待機圖示須通過 catalog 尺寸及 SHA-256 驗證，並在現代主題擁有呈現權。

## 證據範圍與限制

- 第一輪 60／60 個分層代表性抽樣已達最低門檻，屬 95% 信心里程碑；不代表 DOS
  逐像素、逐音訊、逐週期一致，也不冒稱完整長程 `PLAYER-E2`。
- Linux 在隔離 Xvfb 中執行封包自我檢查；Windows／macOS 由原生 GitHub Actions
  runner 建置與自我檢查，實體機操作仍屬發行後抽測。
- 原版音樂對拍影片只留本機，不列入公開附件。

即時技術狀態見 [`docs/REMAKE-STATUS.md`](../REMAKE-STATUS.md)。

## 發布收據

- GitHub Release：<https://github.com/wicanr2/fd2_re/releases/tag/v.1.0.3-20260907>
- 標籤提交：`3dd8ee1a1e87c114d324a67e6804a759a4fb3379`
- Linux Actions：`34052073433`，成功
- Windows Actions：`34052073455`，成功
- macOS Actions：`34052073434`，成功
- 六個公開附件已回讀為 `uploaded`；三平台封包、47 秒 H.264／AAC 推廣片、
  `SHA256SUMS` 與 `MANIFEST.json` 均已存在。
- 本機 `dist-all/v.1.0.3-20260907/full/` 另有三個不公開完整版；Linux AppImage
  經啟動與解包檢查，Windows／macOS ZIP 經逐項清冊檢查，皆含原版 manifest、
  `FD2-LOCAL-FULL.json` 與現代 action icon。
