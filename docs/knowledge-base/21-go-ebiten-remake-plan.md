# 21 — Go / Ebiten 重製架構規劃

> 規劃用 **Go + Ebiten** 重製 FD2,目標**一套程式碼跑桌面 / 網頁(WASM)/ 手機(Android·iOS)**。
> 參考魔法大帝(Master of Magic)Ebiten 重製。可行性見 `20`;腳本系統見 `19`;資產格式見 `01`–`16`。

## 為什麼 Go / Ebiten

- **單一程式碼、三平台**:Ebiten 原生支援桌面、**WebAssembly(瀏覽器)**、**行動裝置**(`ebitenmobile`)。
  正合「以後放到網頁與手機」的需求。
- **純 Go、依賴少**:Web 版 `GOOS=js GOARCH=wasm` 免 CGO;桌面版才需 CGO(OpenGL/X11)。
- **已證可行**:魔法大帝(同型策略遊戲)已用 Ebiten 做成跨桌面/Web/Android。

## 資產管線(玩家自備原版 → 產生 assets)

引擎**不內含原版資產**(著作權)。玩家放入合法原版 → 跑既有抽取工具 → 產生 `assets/`:

```
原版 FLAME2/ ──tools──▶ assets/
  unpack_dat / render_map / extract_maps   → maps/*.png 或 tile 資料 + 地圖 JSON
  decode_dato                              → portraits/*.png
  decode_figani                            → anim/*(戰鬥動畫)
  export_music_ogg(munt+MT-32 ROM)        → music/*.ogg
  decode_story_text --script-json(待做)   → dialogue/*.json(UTF-8)
  dump_exe_tables / parse_field            → tables/*.json + maps_metadata.json
  (TTF 字型,OFL 授權)                      → font.ttf
  campaign.json(19)                        → 關卡流程
```
- **發行兩種**:① 引擎+工具(玩家自備原版自行產資產);② 衍生內容(新劇本/戰場 = 自製,可隨引擎附)。
- 載入方式:WASM 版用 `embed` 或 fetch;桌面/手機用檔案。

## 引擎分層

```
cmd/fd2/main.go            進入點(Ebiten RunGame;WASM/桌面/mobile 共用)
game/
  core/    Game(Update/Draw/Layout)、固定步進迴圈、資源管理
  scene/   ScenarioRunner 狀態機(19):title→story→battle→shop→…(節點/轉場/旗標)
  render/  hi-res 畫布(640×400,nearest 放大)、tilemap、sprite、TTF 文字、對話框
  audio/   OGG 串流(ebiten/audio + vorbis);BGM 迴圈、SFX
  input/   鍵盤+滑鼠+觸控(統一成抽象動作;手機 on-screen 控制)
  battle/  戰棋:格子、單位、移動(flood-fill)、AI(11)、選單(13)、戰鬥公式(02/03)
  data/    載入 JSON(campaign/maps/tables/dialogue)+ 圖/音資產
```

## 各子系統對應 RE 成果

| 子系統 | 來源 | 章節 |
|---|---|---|
| 地圖渲染 | tileset PNG + 地圖 JSON(地形索引) | `01`§8 |
| 單位/動畫/頭像 | PNG sprite/anim/portrait | `06`/`01`§7 |
| 文字 | UTF-8 dialogue JSON + TTF | `08`/`14`/`18` |
| 音樂 | OGG(MT-32)/ 可選 SoundFont | `16` |
| 數值/戰鬥 | tables JSON(物品/法術/單位/成長)+ 公式 | `02`/`03` |
| AI | 評分式(可達格×目標) | `11` |
| 選單/行動 | 行動狀態機 + 游標 | `13` |
| 關卡流程 | campaign.json + ScenarioRunner | `19` |

## 跨平台建置(Docker first)

| 目標 | 指令(docker golang) | 備註 |
|---|---|---|
| **Web(WASM)** | `GOOS=js GOARCH=wasm go build -o fd2.wasm ./cmd/fd2` + `wasm_exec.js` + HTML | 免 CGO;放網頁即玩 |
| **桌面** | `CGO_ENABLED=1 go build`(裝 libGL/X11/asound)→ AppImage/.exe | 沿用魔法大帝 docker-scripts |
| **Android** | `ebitenmobile bind` → `.aar` → Gradle 打 `.apk` | 觸控輸入 |
| **iOS** | `ebitenmobile bind` → `.framework` | Mac 簽章 |

## 輸入:鍵盤 + 觸控(手機)

抽象成動作(上下左右/確認/取消/選單),三種輸入映射到同一套動作:
- 桌面:方向鍵 + Enter/ESC(對應原版 `13`);滑鼠點格。
- 手機:**on-screen 控制**(方向鍵盤 + 確認/取消鈕)或直接點格移動;對話點擊推進。
- (戰棋是回合制、格子操作,對觸控很友善,不像動作遊戲難移植。)

## 階段(里程碑)

1. **MVP 垂直切片**(本輪起步):Ebiten 載入序章地圖(tileset+地圖 JSON)→ 畫出 → 游標移動。
   先證「Go/Ebiten 跑得起來 + 讀得到我們的資料」。WASM build 確認可上網頁。
2. **戰棋核心**:部署我方 → 移動範圍(flood-fill)→ 攻擊 → AI 回合 → 勝敗判定。
3. **文字/對話**:TTF + dialogue JSON + 頭像對話框。
4. **音訊**:OGG BGM/SFX + 音源切換。
5. **ScenarioRunner**:title→story→battle→shop 串起來(用 campaign.json),先跑原版線性。
6. **跨平台**:WASM 上網頁、Android APK。
7. **擴充**:分支/敗北路線/新戰場(`19`)、字型/音源雙模式。

## 風險/注意
- **WASM 體積/載入**:資產(圖/音)較大 → 壓縮(OGG/PNG)、按需載入、loading 畫面。
- **觸控 UI**:手機需重新設計可點區大小(參考「鍵盤→觸控」方法論)。
- **效能**:tilemap/ sprite 用 Ebiten 批次繪製;TTF 字形快取。
- **資產版權**:引擎/工具/腳本公開;原版資產玩家自備,不隨庫散布(同既有鐵則)。
