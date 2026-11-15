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
