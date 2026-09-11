# 編輯器

獨立的網頁工具，`remake/internal/` 一行都不用改（選型與理由見
[`38`](../../docs/knowledge-base/38-editor-design.md) §4）。

| 檔案 | 範圍 | 狀態 |
|---|---|---|
| [`battlefield.html`](battlefield.html) | 戰場：圖塊繪製、單位擺放、部署格 | Phase 1 MVP |

## 怎麼開

`file://` 開不了——Chrome 的 File System Access API 需要 secure context，而且瀏覽器
自動化也擋 `file:`。起一個只服務這個目錄的本機 server：

```bash
cd tools/editor
python3 -m http.server 8765 --bind 127.0.0.1
# 瀏覽器開 http://127.0.0.1:8765/battlefield.html
```

`--bind 127.0.0.1` 不能省：不加會把目錄開給整個區網。用完把 server 關掉。

需要 Chrome 或 Edge。Firefox 與 Safari 目前沒有 File System Access API，編輯器會在
按下「開啟地圖資料夾…」時直接說明，不會靜默失敗。

## 素材不經過任何伺服器

`tileset.png` 由瀏覽器從你自己授權的本機目錄讀取，只留在分頁裡。這個目錄下的檔案
不含任何原版素材，server 也只服務這個目錄。

## 戰場編輯器

開啟時選 `remake/assets/maps/mapN`（裡面要有 `map.json`、`tileset.png`，有
`mapN_units.json` 才能編單位）。

| 工具 | 行為 |
|---|---|
| 筆刷 | 單擊或拖曳塗選中的 tile |
| 矩形 | 拖出範圍，鬆開後整片填入 |
| 填充 | 把相連的同 tile 區域整片換掉 |
| 橡皮擦 | 還原成 tile 0 |
| 單位 | 點空格新增、點單位選取；右側表單改數值 |
| 部署格 | 點一下切換 `own_deploy` |

存回會寫 `map.json` 與 `mapN_units.json`，**引擎直接讀得起來，不需要任何轉換步驟**。

### 移動成本是算出來的，不是填出來的

畫完 tile 之後 `cost` 由地形控制表（`map.json` 的 `native_terrain_control`）按 tile
index 重算，用的是和 [`tools/export_engine_assets.py`](../export_engine_assets.py)
同一張表。兩份常數任何一邊被單獨改到，
[`tools/test_editor_terrain_cost.py`](../test_editor_terrain_cost.py) 會開口——它除了
比對兩張表，還用規則重算每一張受版控地圖的每一格，和已匯出的 `cost` 逐格對照。

屬性面板因此是唯讀的：讓使用者逐格填移動成本，會讓同一張 tile 在不同地圖上行為不同。

### 沒列在表單裡的欄位原樣保留

單位表單只列引擎會讀的那些欄位。原檔的其他欄位在存回時原樣送回去——編輯器少寫一個
欄位不會報錯，只會讓那張地圖在遊戲裡少掉某個效果，而那要玩到那一格才會發作。
