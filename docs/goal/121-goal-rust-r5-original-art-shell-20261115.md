# 121 R5 原版美術外殼：CPU 合成與 Bevy 呈現

依 [119](119-goal-rust-bevy-rewrite-20261113.md) 的 R5（工單 [#311](https://github.com/wicanr2/fd2_re/issues/311)）。R5 把凍結 Go 版 `remake/cmd/fd2` 原版美術模式的繪圖移植到 Rust，並由 `fd2-app` 以 Bevy 開視窗、接輸入與音訊。HD 主題（F2 美術、F11 介面、poster、HD 對話框）屬 R6。

## 現況量測（2026-11-15，靜態閱讀）

| 項目 | 結果 | 出處 |
|---|---|---|
| 邏輯畫布 | 640×400；原生畫面是 320×200 索引緩衝，固定 2 倍放大 | `main.go:53`、`main.go:11586` |
| 原版畫面的合成 | 在 CPU 合成 8-bit 索引緩衝與調色盤，轉成 `image.Paletted` 後 `NewImageFromImage`，`GeoM.Scale(2, 2)` 以最近鄰貼上 | `main.go:12442`、`native_palette_ramp.go:180` |
| 調色盤效果 | 斜坡、加亮、白閃、pulse 都在 CPU 改 DAC 再重建調色盤；6-bit 轉 8-bit 是 `(v<<2)|(v>>4)` | `internal/fdother/palette.go:13`..`76` |
| `Game.Draw` | `main.go:9355`..`10188`，約 834 行，依序 early-return 約 35 個全螢幕原生呈現 | 同左 |
| 原版繪圖量 | `cmd/fd2` 非 HD 的 draw／present／compose 函式 175 個約 6500 行，另有 `title.go`、`ending_preview.go`；合成器在 `internal/`（indexedmap、battlepresent、fdother、fdicon、figani、fdtxt、ending 等），R1～R3 已移植 | 盤點估計 |
| 走 RGBA 的少數路徑 | 半透明色塊（範圍格、游標條、舊 fade 疊層）以 Ebiten 預乘 alpha 疊合；`ColorScale` 用在已行動單位變暗、標題 publisher 亮度與 scroll 淡入；`ColorM` 只是標題缺預建幀時的後備 | `main.go:10173`、`main.go:11545`、`title.go:496`、`title.go:739`、`title.go:790` |
| 非整數縮放 | 標題 logo 縮放（6 倍縮到 2 倍，最近鄰）；舊對話框的垂直縮放與 2.1 倍頭像 | `title.go:803`、`main.go:10013`、`main.go:10053` |
| 中文字 | 繁中原生畫面用原版 FDOTHER #4 的 16×16 點陣字模，直接寫進索引緩衝；其他語系的原生畫面用 TTF 畫到 alpha 遮罩再以 A≥32 二值化；舊後備畫面用 Ebiten `text`（v1）反鋸齒 TTF，`HintingFull`、72 DPI | `internal/fdtxt/fdtxt.go:158`、`native_story_dialogue_locale.go:155`、`font.go:164` |
| 只在外框 | `vector.DrawFilledRect`、`FilterLinear`、screen shader 只用在工具列、速度選單、poster 與視窗縮放，不進邏輯畫面 | `ebiten_screen_filter.go:52`、`player_toolbar.go:207` |
| 音訊 | `audio.NewContext(44100)`；BGM 以 `vorbis.DecodeWithSampleRate` 解碼並無限循環；SFX 是 FDOTHER 的 OGG bank 先解成 PCM | `audio.go:27`..`307` |

## 架構（使用者 2026-11-15 定案）

- **CPU 合成。** 繪圖移植到 `fd2-game`，在 CPU 產生 640×400 RGBA 畫格；無畫面也能跑，黃金差分直接比畫格位元組。`fd2-app` 只把畫格上傳成貼圖、整數倍最近鄰放大，並接輸入、音訊與視窗。
- Ebiten 繪圖語意只實作原版模式用到的子集：`DrawImage`（平移、整數與上列非整數縮放、最近鄰）、`Fill`、`SubImage`、預乘 alpha 的 source-over、`ColorScale`。Ebiten 在 GPU 上做 alpha 與 `ColorScale` 的取整方式要先實測（階段 0），以 Go 版實際畫出的像素為準。
- **文字。** 繁中畫面用原版點陣字模，必須逐像素相同。TTF 路徑（其他語系、舊後備畫面）以 `Cargo.lock` 已鎖定的字型套件點陣化，只比文字位置與外框，像素差異寫成 `KNOWN_DIFFERENCES`；不移植 Go 的字型點陣化器。
- **貼圖與語系分開。** 美術選擇只由美術設定（F2）決定，所有語系一致；語系只影響文字。Go 版非繁中語系的系統資訊畫面會不管 F2／F11，以 6 倍解析度合成含當地文字的 HD 圖再縮成 1/3 貼上（`native_system_info_ui.go:208`..`229`）。Rust 版不照做：原版美術時所有語系都用原版外框，文字走該語系的一般文字路徑；HD 美術由 R6 對所有語系（含繁中）一致處理。這一處列為使用者定案的刻意差異，Go 黃金的非繁中系統資訊畫格不作為比對基準。
- **GPU 加速模式（使用者 2026-11-15 要求）。** CPU 合成通過驗收後再加。GPU 模式在原版美術下的輸出必須與 CPU 合成的畫格逐像素相同，以 CPU 結果作黃金；CPU 合成保留為預設與驗證基準。

## 範圍

| 項目 | R5 | 之後 |
|---|---|---|
| 原版模式的全部畫面繪圖、調色盤效果、fade | 移植 | |
| `Game::acknowledge_present` 剩下的 R5 分支 | 接上 | |
| `fd2-app`：視窗、整數倍放大、鍵盤／滑鼠／觸控對應到 `TickInput`、BGM 與 SFX | 實作 | |
| 播放器外框（工具列、速度選單、說明畫面） | 說明畫面移植；工具列與速度選單只做原版需要的部分 | 完整工具列屬 R6 |
| HD 主題、poster、HD 對話框、HD 系統資訊 | | R6 |
| GPU 加速模式 | | CPU 驗收後 |
| 第一章以外的章節流程 | 只移植開場與第一章走到的畫面 | R8 |

## 驗證

1. **Go 測試移植**：Go 的繪圖測試（建 `ebiten.NewImage` 再 `ReadPixels` 的那些）逐條移植成比 CPU 畫格的測試。
2. **Go 黃金畫格**：
   - 章重播 `FD2_PARITY_CAPTURE_FRAMES=1` 輸出的 320×200 索引畫格（每個相位一張，不經 GPU），與 Rust 逐位元組比對。
   - `FD2_SHOT`／`FD2_SHOT_STATE` 在 Xvfb 下擷取的 640×400 畫格，涵蓋標題、開場、城鎮、商店、整備等章重播檢查點以外的畫面；同一狀態的 Rust 畫格逐像素比對。
   - 範圍：開場與第一章全部相位（使用者 2026-11-15 定案）。TTF 路徑與上述刻意差異以外，不得遮蔽像素。
3. **外殼**：`fd2-app` 在 Xvfb 下的視窗截圖，必須等於 CPU 畫格的整數倍最近鄰放大；R0 已證明 Bevy 可以做到。
4. **原版收據層**：沿用 `verify_chapter_parity.py`；R5 不新增原版對拍，`58` 的證據分層不變。

## 階段

| 階段 | 內容 |
|---|---|
| 0 | 盤點與實測：列出原版模式會走到的繪圖函式（以覆蓋率確認「舊後備」路徑在開場與第一章是否觸發）；實測 Ebiten 的 alpha 與 `ColorScale` 取整；產生 Go 黃金畫格並固定雜湊 |
| 1 | 骨架：CPU 畫布與 Ebiten 語意子集、`Game::draw` 的函式骨架（`todo!` 標 R5／R6／R8），對照表寫進 `fd2-game/PORTING.md` |
| 2 | 分組平行移植繪圖函式，每組附 Go 測試移植與局部黃金 |
| 3 | 整合：開場與第一章全部畫格與 Go 黃金比對到相同 |
| 4 | `fd2-app`：Bevy 視窗、輸入、音訊（OGG 解碼與 Go 逐樣本比對，見 119 R0 的待查） |
| 5 | GPU 加速模式（CPU 驗收後） |
