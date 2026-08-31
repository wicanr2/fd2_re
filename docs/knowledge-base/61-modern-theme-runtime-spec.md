# 61 — 現代美術主題正式執行期規格

狀態：`READY-PARTIAL`（2026-08-31）

## 範圍

本切片只定義索爾故事對話的獨立現代頭像。它不授權從合成概念稿反切戰場
sprite、地圖圖塊或 HUD，也不改變忠實原版主題。

## 原始證據與既有消費端

- 原版分離資產：`assets/portraits/DATO_000_m0.png`；`DATO` resource 0、frame 0。
- `dato.Frame` 已證實為 `80×80`，故事說話者必須具備四幀；正式載入端位於
  `remake/cmd/fd2/native_story_dialogue.go` 的 `loadNativeSeparatedPortrait`。
- 原生上框頭像右向合成起點是線性位址 `0x728`，即 `(232,5)`；下框是
  `0x9017`，即 `(87,115)`。兩者皆覆蓋 `80×80`。
- `ComposeNativeStoryDialogueBaseFrame` 目前會把 DATO 頭像直接寫進
  `320×200` 索引畫面；`presentNativeClassFrame` 再套用原版調色盤並放大兩倍。
- 證據等級：上述形狀、座標、載入者與消費端皆為**已證實**；來源是受版控
  原始碼、分離資產測試及既有原版位址證據。現代畫風本身是使用者核准方向，
  不是原版行為證據。

## 型別與呈現契約

1. 現代故事頭像是獨立 `80×80` 真彩色 PNG，以穩定 `asset_id`、SHA-256、
   原始 frame 身分及嘴型狀態登錄於 `assets/themes/modern/catalog.json`。
2. 現代頭像不得先量化進原版 16 色索引畫面；正式 renderer 應先呈現原生框與
   文字，再在相同邏輯座標以真彩色層覆蓋完整 `80×80` 頭像區。
3. 上／下框錨點與 `NativeStoryDialogueTextGeometry` 不得改動。
4. 第一個候選只提供索爾閉嘴 frame 0。正式啟用現代主題前，必須明確選擇並
   驗證靜態閉嘴策略，或補齊四幀嘴型；不可偷偷混用原版 frame 3。
5. 主題必須整組預檢並原子切換。檔案缺漏、尺寸不符、雜湊不符或 speaker
   沒有對應資產時，現代主題失敗即關閉，不得在同一畫面混搭原版與現代頭像。

## 目前候選

- `modern.sol.portrait.style_a.frame0`
- 執行期檔：`sol-portrait-style-a-v2-80.png`，`80×80`，SHA-256
  `6ecc692c30973c62900dc9983ea878df29394d30bfe28d7bf6931dd223131e82`
- 母稿：`sol-portrait-style-a-v2-master.png`，`1254×1254`。
- 兩檔皆留在忽略版控的私人素材目錄；公開儲存庫只保存契約與雜湊。

## 驗收與停止線

- catalog validator 必須驗證候選檔尺寸與 SHA-256。
- renderer 單元測試須驗上／下框矩形、缺檔、錯誤尺寸及未知 speaker 均拒絕。
- 正常故事路徑至少抽測索爾上框與下框各一張，確認文字安全區、縮放與無混搭。
- 完成上述 consumer 前狀態維持 `runtime_candidate`，不可寫成 `runtime_ready`。
