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

## 階段 0 結果（2026-11-15）

命令：`FD2_DATA_ROOT=<主工作區> tools/rust_r5_golden_frames.sh work/r5-stage0-20261115`。產出在主工作區 `work/r5-stage0-20261115/`，PNG 與覆蓋率檔不進版控。環境是 `fd2-go-test-local`（Go 1.22.12、Ebiten v2.6.6），Xvfb 下的 Mesa 22.3.6 llvmpipe。

新增的工具都只輸出資料，既有 Go 檔沒有改：

| 檔案 | 用途 |
|---|---|
| `remake/cmd/fd2/rust_golden_frames_test.go` | `TestRustGoldenFrames`，未設 `FD2_GOLDEN_FRAMES_OUT` 時略過 |
| `remake/cmd/fd2-r5-ebiten-rounding/` | Ebiten 半透明疊色與 `ColorScale` 的讀回實測 |
| `tools/rust_r5_draw_inventory/` | 以 go/types 建 `Game.Draw` 的靜態呼叫樹，與覆蓋率交集 |
| `tools/rust_r5_golden_frames.sh` | Docker 驅動：兩種模式各跑兩次、比雜湊、再跑實測與盤點 |

### Go 黃金畫格

| 模式 | 範圍 | 擷取 | 內容改變的幀 | 相異畫面 | 序列 SHA-256 |
|---|---|---|---|---|---|
| `opening` | 標題開場動畫 → 選單 → START → 序章故事 → 第一次進入 `battle_ch01`，每個 Update 之後一張 | 10372 | 7445 | 6334 | `eeed99ea…0be771` |
| `chapter` | 第一章受控重播（輸入同 `tools/rust_r4_ch01_coverage.sh`）的每個 pump 幀 | 13751 | 8589 | 7463 | `2f26390d…ff7c4f` |

- 兩次執行的序列雜湊、`frames.json`（`7a9c8226…`、`d71aad6a…`）與 16034 張 PNG 逐檔相同。
- Go 的 HD 載入修正（`a399da4a`，v.1.1.13）改了 `cmd/fd2` 的 `Update` 與 `loadGame`。合併後以 `FD2_R5_RUNS=r1 FD2_R5_SKIP_ANALYSIS=1` 重跑一輪（`work/r5-golden-recheck-20261115`），兩種模式的序列雜湊與 16034 張 PNG 都與上表相同，`checkpoints.jsonl` 仍是 `1c846f57…`，黃金畫格不必重產。
- 加上每幀 `Draw` 之後，`chapter` 模式的 `checkpoints.jsonl` 兩次都是 R4 Go 黃金 `1c846f57…`。
- 擷取方式：`chapter` 模式原樣呼叫 `TestChapterParityReplay`，只掛共用 `pump` 既有的 `ch01FrameObserver`，在上一個 Update 之後、下一個 `ackPresents` 之前 `Draw` 到離屏 640×400 影像並 `ReadPixels`。`Draw` 在 Ebiten 的 `Update` 內執行，每次擷取都走完一個 Ebiten 畫格，否則每幀新建的影像不會釋放。
- 牆鐘：遊戲時間都經過 `playerClock.sample`。擷取時把 `playerClock.wall` 設在遠未來，再依 `g.frame` 的增量把 `playerClock.game` 推進 1/60 秒，等於 60 Hz 虛擬時鐘。
- 亂數：嘴型動畫用 `math/rand` 全域來源，Go 1.20 起它隨機播種，正式程式沒有 `rand.Seed`。沒固定時檢查點相同，但兩次執行的畫格序列不同，第一個分歧在戰場事件對白的嘴型時點。擷取前以 `rand.Seed(4)` 固定。Rust 版的嘴型要照 Go `math/rand` 的序列。
- `chapter` 模式沒擷到的 Update 共 7 段 9282 個，記在 `frames.json` 的 `gaps`。序章 `driveStory` 佔 8430 個，由 `opening` 模式補上。武器店直接迴圈佔 817 個。另有 5 段各 7 個，推定是指令環開合的 `closeRing`／`settleActionOverlay`。城鎮、商店、整備多半不經 `pump`，只擷到 1 到 21 幀，要照驗證第 2 點以 `FD2_SHOT_STATE` 另補。

`Draw` 後依狀態推定的主路徑（`frames.json` 的 `draw_paths`，近似分類）：

| 路徑 | opening | chapter |
|---|---|---|
| `title`（`drawTitle`） | 1940 | 24 |
| `story-viewport`（storyBG 或戰場事件：正規化圖塊與 sprite 畫到 320×200 離屏，再 `GeoM.Scale(storyZoom)`） | 8431 | 1837 |
| `native-map`（`drawNativeMapFrame` 蓋上整幀索引畫面） | 1 | 4077 |
| `battle-normalized`（戰場但沒有原生整幀，正規化圖層可見） | 0 | 1548 |
| `fullscreen-native`（`Draw` 前段 early-return 的原生全螢幕呈現與非戰場節點） | 0 | 6265 |
| 其中帶 `+ttf-msg`（尾段以 TTF 畫 `g.msg`） | 0 | 5763 |
| 其中帶 `+fade`（舊 fade 黑色疊層） | 72 | 34 |

### 執行到的繪圖函式

從 `(*Game).Draw` 沿靜態呼叫邊走，只經過覆蓋率有執行的函式。結果在 `work/r5-stage0-20261115/inventory/executed-draw-functions.json`（`60e0a421…`）與 `draw-inventory.json`（`4be7ce9d…`）。

| 分組 | 函式 | 行數 |
|---|---|---|
| 城鎮與設施（含 486 行的 `drawCampaignUI`，第一章只跑到守門與少數分支） | 28 | 1046 |
| `Game.Draw` 本體（分派、正規化地圖層、舊對話框與 fade） | 1 | 834 |
| HUD 與選單 | 35 | 790 |
| 對話與文字 | 16 | 383 |
| 戰鬥演出 | 7 | 318 |
| 地圖與單位 | 14 | 283 |
| 標題與開場 | 7 | 257 |
| 轉場與調色盤 | 4 | 229 |
| 素材路徑與時鐘（非繪圖） | 14 | 160 |
| 共用與其他 | 1 | 7 |
| 非 HD 合計 | 127 | 4307 |
| HD 檔（`modern_*` 等，原版模式下多為守門） | 69 | 2431 |

- 這些函式直接用到的 Ebiten API：`DrawImage`、`GeoM.Translate`／`Scale`、`NewImage`、`NewImageFromImage`、`Fill`、`SubImage`、`Clear`、`WritePixels`、`Set`、`Dispose`、`ColorScale.Scale`／`ScaleWithColor`、`text.DrawWithOptions`／`BoundString`、`ebitenutil.DebugPrint*`。`ColorM`、`vector`、shader 都沒有出現在執行到的函式裡。
- 限制：介面呼叫無法靜態解析，本輪為 0 處。經 func 型別變數或欄位的呼叫有 21 處，多是函式內的閉包。`internal/` 的合成器不在 `-coverpkg=./cmd/fd2` 內，這裡只算 `cmd/fd2`。

### 舊後備與 RGBA 路徑

| 路徑 | 位置 | 執行次數（opening／chapter） | 結論 |
|---|---|---|---|
| 正規化圖塊與 sprite 的 320×200 離屏放大 | `main.go:9815`..`9909` | 序章與戰場事件的主要路徑 | 會走到，畫面可見，要移植 |
| 已行動單位 `ColorScale.Scale(0.55,0.55,0.6)` | `main.go:11545` | 0／9071 | 會走到；有原生整幀時被蓋掉，`battle-normalized` 時可見 |
| 移動範圍半透明格 `(0x40,0x80,0xff,0x66)` | `main.go:9844` | 0／135 | 會走到，同上 |
| 白框游標 `drawCursor` | `main.go:11571` | 4／15308（每次 4 條） | 會走到，同上 |
| 舊 fade 黑色疊層 | `main.go:10180` | 72／34 | 會走到，畫在最上層，可見 |
| TTF 戰場短訊 `g.msg` | `main.go:10122` | 0／2650 | 會走到，畫在原生整幀之上，可見 |
| TTF 勝負字樣 | `main.go:10137`、`10143` | 0／1 | 會走到 |
| publisher 亮度 `ColorScale` | `title.go:739` | 119／0 | 會走到 |
| 色塊單位 `drawUnit`、標題 `ColorM`（`titlePaletteBlend`）、`DebugPrint` 35 處、scroll 淡入（`title.go:790`）、白閃（`title.go:812`）、舊對話框（`main.go:10000`..`10118`） | | 0 | 開場與第一章沒走到 |

HD 檔 `modern_hd_confirmation.go:122` 在第一章執行 8 次，那一行是 `freezeModernHDConfirmation` 預先量「是／否」字框的 `text.BoundString`，只量尺寸、不畫到影像，原版模式的畫面不受影響。呼叫端是 `native_death_reward_ui.go:108` 等建構 HD 確認框的地方，屬 R6。

### Ebiten 取整（llvmpipe）

`cmd/fd2-r5-ebiten-rounding` 跑 30 組實驗，每組 196608 個通道值。結果在 `work/r5-stage0-20261115/rounding/ebiten-rounding.json`（`1dc87b3a…`）。下列公式與 Go 實際讀回逐值相同：

| 操作 | 逐值相同的公式 |
|---|---|
| `Fill` 色塊以 `DrawImage` source-over 疊到不透明底 | `out = min(255, s + mul255(d*(255-a)))`，`mul255(x) = (x + (x>>8) + 0x80) >> 8` |
| `ColorScale.Scale(k)` 畫到不透明底 | `out = f2u(u2f(v) * k)` |
| `ColorScale.ScaleWithColor(c)`（TTF 字的覆蓋率） | 來源 rgb 與 alpha 各自 `f2u(u2f(v) * k)`，再照第一列疊合 |

- `u2f(v) = float32(float32(v)/256) * float32(256/255)`，是 llvmpipe 讀 8 位元貼圖的換算。
- `f2u(x)`：x≤0 為 0，x≥1 為 255，其餘取 `float32(float32(x*float32(255/256)) + 32768)` 的 IEEE 位元低 8 位，float32 加法以最近偶數取整。
- 每一步都要明寫 float32 轉換，避免乘加融合成 FMA。
- 常見的四捨五入公式不對：`(d*(255-a)+127)/255` 在每組 alpha 掃描的每個通道有 24 個 (d, a) 組合不同，例如 d=191、a=2 時實際是 189，不是 190。`round(v*k)` 在 30×0.55 時實際是 16，不是 17。
- R/G/B 大於 A 的非預乘色（main.go 的範圍格、對話框底）也走同一公式，結果在 255 截頂。
- 這是 llvmpipe 的行為，換 GPU 驅動可能不同。黃金畫格在同一環境產生，CPU 合成照上列公式實作；之後的 GPU 模式以 CPU 畫格為準。

### 對下一階段的輸入

- 階段 1 的 Ebiten 語意子集要包含：最近鄰 `DrawImage`（平移、整數與 `storyZoom` 縮放、對話框的非整數縮放）、`Fill`、`SubImage`、`Set`、上列 source-over 與 `ColorScale` 公式，以及 TTF 文字的位置與外框。
- 階段 2 的分組照上表。`story-viewport` 的正規化圖層與 `drawCampaignUI` 是行數最多的兩塊，城鎮與設施要先補 `FD2_SHOT_STATE` 黃金才能驗。
- Rust 對比 `chapter` 畫格時，要照同一套 pump 擷取點、60 Hz 虛擬時鐘與 `rand.Seed(4)` 重現。

## 階段 1 結果（2026-11-15）

逐項落地見 [`fd2-game/PORTING.md`](../../remake-rs/crates/fd2-game/PORTING.md)「R5 階段 1」。

### 畫布型別

- `Picture` 改存預乘 alpha（Ebiten `ReadPixels` 的輸出），不另立畫布型別。R4 的轉換函式逐一改成 `NewImageFromImage` 之後的值：
  `*image.Paletted` 原值、`color.NRGBA` 調色盤與 PNG（Go `*image.NRGBA`）依 `image/draw` 預乘。原版美術模式的素材都是不透明或全透明，
  位元組不變。
- Ebiten 語意子集在 `remake-rs/crates/fd2-game/src/game/canvas.rs`：`NewImage`、`NewImageFromImage`（`Paletted`、`NRGBA`）、`Clear`、
  `Fill`、`Set`、`WritePixels`、`ReadPixels`、`SubImage`（來源與目標）、`DrawImage`（`GeoM` 平移與縮放、最近鄰、`ColorScale.Scale`／
  `ScaleWithColor`、source-over）。放在 `game` 底下，因為 R4 已把 `*ebiten.Image` 定成 `Picture`，繪圖函式的簽名都收 `Picture`。
- 單元測試 8 項讀 `src/game/canvas/ebiten_vectors.json`（`tools/rust_r5_canvas_vectors.sh` 從兩份實測輸出抽出，`5ade78bf…`）：
  重建每個實驗的輸入、以畫布畫一次、比整張讀回的 SHA-256。階段 0 混色 30 組、階段 1 取樣 89 組與語意 11 組全部相同；常見公式會算錯的
  546 個值逐筆相同（含 d=191、a=2 → 189 與 30×0.55 → 16）。竄改公式後對應測試失敗。

### 取樣與 Fill 實測

命令：`FD2_GOCACHE=<主工作區>/work/gocache tools/rust_r5_ebiten_sampling.sh <主工作區>/work/r5-stage1-20261115/sampling`，
環境同階段 0。實驗加在 `remake/cmd/fd2-r5-ebiten-rounding/sampling.go`（`-sampling-out`）；`remake/cmd/fd2` 與其他既有 Go 檔沒有改。
兩次輸出相同，`ebiten-sampling.json` SHA-256 `096f7972595345223ebc3e4335cbed53da5c96ca1429a3d99b4c20a3e17bdce8`；工具改成逐步執行
後重跑階段 0 的實驗仍是 `1dc87b3a…`。

| 項目 | 結論（逐值相同） |
|---|---|
| 最近鄰取樣（89 組：原生整幀與 storyView 2 倍、標題捲動、倍率 1 的小數平移 32 組、圖塊鏡頭、舊對話框 `Scale(2, 2*sc)`、2.1 倍頭像、1×198 漸層、標題 logo 縮放 30 組） | 目標頂點先經 Ebiten `adjustDestinationPixel` 對齊到整數加 0、5/16、11/16、1；像素中心 `p+0.5` 落在 `[x0, x1)` 才覆蓋；來源索引 `floor((p+0.5-x0)*n/(x1-x0))`。兩軸可分開算。不經對齊的像素中心公式在平移小數約 0.3、0.5 與舊對話框的部分縮放不符 |
| `Fill` 半透明色 | 覆蓋，不疊合；讀回等於 `color.RGBA` 的四個位元組，含 R/G/B 大於 A |
| `SubImage` 後 `Fill`／`DrawImage` | 只寫入與父影像相交的矩形；座標是父影像座標；子影像的子影像再取交集 |
| `Set`、`NewImageFromImage`、`WritePixels`／`ReadPixels`、疊到半透明底 | `clr.RGBA()>>8`；`color.RGBAModel` 與 `image/draw` 預乘；原樣；四個通道同一 source-over 公式 |

開場與第一章實際用到的非整數倍只有倍率 1 的小數平移（鏡頭內插、行軍位移）；舊對話框、頭像、logo 縮放沒有執行到，一併量了。

### 骨架

`tools/rust_r5_skeleton.sh`：`tools/rust_r4_skeleton -r5` 以 R4 的型別閉包與命名，依階段 0 盤點（受版控複本
`docs/data/rust-r5-executed-draw-functions-20261115.json`，`60e0a421…`）產生簽名，`tools/rust_r5_skeleton_apply.py` 依 crate 現況
套用，重跑不改任何檔，從 `eeb024d7` 重新套一次結果相同。

| 項目 | 數量 |
|---|---|
| 盤點的函式 | 196（非 HD 127、HD 69） |
| 非 HD 標 `r5` | 58（其中 5 個是更正錯誤的 `ported`：`Font.Draw` 在 crate 裡沒有函式，另 4 個本體只有 `todo!("R5: …")`） |
| 新增骨架 `todo!("r5:<檔>.go:<函式名>")` | 52 個函式，分在 20 個子模組 |
| 非 HD 已 `ported` | 69 |
| HD 檔 `r8` 改 `r6` | 42（另 27 個已 ported） |

`Game::draw` 照 Go `Draw` 的 early-return 順序寫成分派：開場與第一章執行到的分支（標題、ch28 post present、調色盤斜坡、登場轉場、
END 回收、AI 閒置回收、攻擊畫面、尾段地圖與介面）呼叫對應函式，HD 分支 `todo!("r6:…")`，沒有執行到的分支 `todo!("r8:…")`；分支是否
執行到依階段 0 的覆蓋率。尾段的正規化地圖層與舊 fade 疊層是 Draw 本體的 r5 `todo!`。R5 對照表（每組函式、Go 位置、行數、畫布操作、
局部黃金來源）在 `PORTING.md`，r5 函式合計約 2200 行。

### 畫格擷取介面

- `parity_replay/rust_golden_frames_test.rs`：設 `FD2_GOLDEN_FRAMES_MODE=chapter`、`FD2_GOLDEN_FRAMES_OUT` 時，章重播的 `pump` 每一幀
  在上一個 `update` 之後、`ack_presents` 之前呼叫 `Game::draw` 到 640×400 畫布並記 SHA-256，與 Go `ch01FrameObserver` 同一位置；結束時
  寫與 Go 同名欄位的 `frames.json`。預設關閉。
- 關閉時 `tools/rust_parity_ch01.sh` 的 `checkpoints.jsonl`（`1c846f57…`）、`FD2.SAV`（`f3942ff2…`）、`hotel-save.json`（`190b9f8f…`）
  與 Go 黃金逐位元組相同。
- 重現規格：60 Hz 虛擬時鐘照 Go 黃金的 `pinClock`（第一次擷取時牆鐘設在遠未來，之後每次擷取依幀數增量推進
  `16666666 ns × speed`，pump 以外的 `Update` 期間凍結）已實作；gaps 以同一格式記錄，階段 3 先比 gaps 再比畫格；全域亂數
  `rand.Seed(4)`（故事嘴型 `rand.Intn(30)` 與確認框嘴型的 `rand.Int63()` 共用一個來源）**還沒接上**，規格寫在 `PORTING.md`，階段 3
  比對前要先做。
- 目前 r5 函式都是 `todo!`，打開擷取會停在第一個擷取點；沒有做畫格比對。

### 驗證

`tools/rust_game_test.sh` 320 passed、4 ignored（加畫布 8 項）；`tools/rust_campaign_test.sh` 530 passed、`tools/rust_rules_test.sh` 654 passed；`Cargo.lock` 沒有改動，沒有新增套件。

## 階段 2 結果：r5t 標題與開場（2026-11-15）

逐項落地見 [`fd2-game/PORTING.md`](../../remake-rs/crates/fd2-game/PORTING.md)「R5 階段 2：r5t」。

- 移植 `title.go` 的 `titleMenuY`、`titlePublisherBrightness`、`titleScrollUsesPulsePalette`、`drawTitle`，`PORTING.json` 以
  `tools/rust_r5_mark_ported.py` 改成 `ported`。`drawTitle` 的 HD 守門（`modernOpeningEnabled`、`drawModernHDTitleMenuItem`）在忠實模式
  為 false，HD 時 `todo!("r6:…")`；沒有執行到的 `ColorM` 後備（`titlePaletteBlend`）與讀檔槽畫面標 `todo!("r8:…")`。canvas 沒有補語意。
- Go 測試 3 條（`title_menu_state_test.go` 的 `:110`、`:129`、`:349`）逐條移植。`drawTitle` 在 Go 只有 HD 的 GPU 測試（R6）。
- Rust 加 opening 模式擷取（`parity_replay/rust_golden_frames_opening_test.rs`，`tools/rust_r5_opening_frames.sh`），步驟與 Go
  `goldenFramesOpening` 相同；`Game::draw` 停在未移植的 `todo!` 時記在 `frames.json` 的 `stopped`。

### 局部黃金

比對工具 `tools/rust_r5_frames_compare.py`：兩邊的 rows 依 `capture`／`count` 展開成每次擷取的 RGBA SHA-256，逐次比；另要求 Go 黃金
`draw_path` 為 `title` 的每一次都在 Rust 範圍內且相同。不遮蔽像素。輸出在主工作區 `work/r5-stage2-r5t-20261115/`（不進版控）。

| 項目 | 結果 |
|---|---|
| Go 黃金 | `work/r5-stage0-20261115/r1/opening/frames.json`（`7a9c8226…`），`title` 擷取 1940 次，序號 0..1939 連續 |
| Rust 擷取 | 1940 次，停在第 1940 次：序章第一幀（`story_ch00_handler`）走到 Draw 本體的正規化地圖層 `todo!("r5:main.go:9814..9950")`，不屬本組 |
| 逐次比對 | 1940／1940 相同；前 1940 次的序列雜湊兩邊都是 `8768d920a135e001a53a2bed8f1b230d402fd07b2dd49d72fa76b875d7e3fa06` |
| 重跑 | 三次 `frames.json` 逐位元組相同；暫時拿掉 publisher 的 `ColorScale` 後比對失敗（1926／1940，第一個不同在第 0 次），比對有效 |

序章故事段（Go 黃金第 1940..10371 次共 8432 次，`story-viewport` 8431 次、`native-map` 1 次）要等正規化地圖層移植後才能比。`scroll`、`logozoom` 兩個 `titlePhase`
與 `ColorM` 後備不在 Go 黃金範圍內。

### 驗證

`tools/rust_game_test.sh` 323 passed、5 ignored；`tools/rust_campaign_test.sh` 530 passed、`tools/rust_rules_test.sh` 654 passed；`tools/rust_parity_ch01.sh`（擷取關閉）的 `checkpoints.jsonl`
`1c846f57…`、`FD2.SAV` `f3942ff2…`、`hotel-save.json` `190b9f8f…` 與 Go 黃金相同；`Cargo.lock` 沒有改動。共用檔只改
`parity_replay` 的擷取器（`FrameCapture` 加 opening 模式與 `stopped`，chapter 模式行為不變），`Game::draw`、`canvas.rs` 沒有改。
## 階段 2 結果：r5m（2026-11-15）

地圖與單位組。逐項落地見 [`fd2-game/PORTING.md`](../../remake-rs/crates/fd2-game/PORTING.md)「R5 階段 2：r5m 組」。

### 移植

- 分組表的 7 個函式（`tileAt`、`nativeMapFrameAdmission`、`mapSpriteGroup`、`drawUnitSprite`、`drawNativeMapFrame`、
  `nativeMapFrameImage`、`drawNativeSpawnIntro`）照 Go 逐行移植，只用階段 1 的畫布操作，`canvas.rs` 沒有改。
- `nativeMapFrameImage` 每次都呼叫回合橫幅的 `phaseBannerStep`、`phaseBannerSlideOffset`、`blitPhaseBannerGlyphs`
  （連同 `phaseBannerElapsedMillis`、`blitPhaseBannerGlyph`、`phaseBannerGlyphsAvailable`），這 6 個列在「HUD 與選單」，
  一併移植；`drawPhaseBanner` 的 TTF 後備沒有動。
- `PORTING.json` 13 個函式 `r5` 改 `ported`（`tools/rust_r5_mark_ported.py`，記在 `r5_stage2_r5m`）。`drawUnitSprite` 缺
  sprite 時的 `drawUnit` 色塊在開場與第一章沒有走到，留 `todo!("r8:…")`。
- `drawUnitSprite` 的 `u` 改收 `&battle::Unit`：Go 也傳 `&g.storyActors[i]`，那不是 `st.units` 的索引。
- Go 測試 11 項移植（回合橫幅 7 項、守門 3 項、`TestDrawNativeMapFramePresentsCorrectedVGABorder`），另加 4 項；
  `tools/rust_game_test.sh` 335 passed、4 ignored。

### 局部黃金

`chapter` 整張畫格混有別組圖層，所以新增只輸出資料的 `remake/cmd/fd2/rust_golden_map_layers_test.go`（未設
`FD2_GOLDEN_MAP_LAYERS_OUT` 時略過，既有 Go 檔沒有改）與 Rust 端 `parity_replay/rust_golden_map_layers_test.rs`。兩邊照
階段 0 `chapter` 模式的掛點與 60 Hz 虛擬時鐘跑第一章受控重播（Go 另有 `rand.Seed(4)`；Rust 的全域亂數還沒接上，只影響嘴型，不進本組圖層），每個 pump 幀不呼叫整個 `Draw`，只照 `Draw`
的分支順序呼叫本組函式：spawn intro 畫格、正規化世界層（圖塊、單位、storyActors）、`nativeMapFrameAdmission` 與原生地圖
整幀（含 320×200 索引畫格與調色盤），各自畫到空白畫布取雜湊。驅動與比對是 `tools/rust_r5_map_layers.sh`、
`tools/rust_r5_map_layers_compare.py`。

| 項目 | 結果 |
|---|---|
| 擷取數 | Go、Rust 都是 13751（與階段 0 `chapter` 相同） |
| 逐圖層比對 | spawn 24、world 6404、admission 6404、map 6370、索引畫格 6370，全部相同 |
| 序列雜湊 | 兩邊都是 `bb359b4d5a0d0c69500b12adde78e4217ad3d45c3105fc7394c49b93c6eb1238`；r1、r2 兩次的 `rows` 逐筆相同 |
| `checkpoints.jsonl` | 兩邊都是 R4 黃金 `1c846f57…` |

- 涵蓋已行動單位 `ColorScale` 9071 次、回合橫幅馬賽克 1472 張與字樣 1999 張；沒有遮蔽像素。
- 旁證：與階段 0 整張黃金同一擷取點對照，spawn 24 張全部相同，map 6370 張中 1407 張相同（其餘整張畫格上另有別組圖層）。
- 正規化世界層的移動範圍格、`storyZoom` 放大、游標與 HUD 屬 `Draw` 本體與 HUD 組，不在這份比對裡；`Draw` 尾段的正規化
  地圖層 `todo!`（`main.go:9814..9950`）沒有接上，整張畫格的比對在階段 3。

### 驗證

`tools/rust_game_test.sh` 335 passed、4 ignored（fmt、clippy 通過）；`tools/rust_campaign_test.sh` 530 passed、
`tools/rust_rules_test.sh` 654 passed；`tools/rust_parity_ch01.sh`（擷取關閉）的 `checkpoints.jsonl`、`FD2.SAV`、`hotel-save.json`
與 Go 黃金逐位元組相同；`Cargo.lock` 沒有改動。
