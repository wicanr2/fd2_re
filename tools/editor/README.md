# 編輯器

獨立的網頁工具，`remake/internal/` 一行都不用改（選型與理由見
[`38`](../../docs/knowledge-base/38-editor-design.md) §4）。

| 檔案 | 範圍 | 狀態 |
|---|---|---|
| [`battlefield.html`](battlefield.html) | 戰場：圖塊繪製、單位擺放、部署格 | Phase 1 MVP |
| [`campaign.html`](campaign.html) | 對白、戰場事件、商店品項、節點轉場、節點圖 | Phase 2＋3 |

## 怎麼開

```bash
python3 tools/editor/serve.py          # 預設 127.0.0.1:8765
# 瀏覽器開 http://127.0.0.1:8765/battlefield.html 或 campaign.html
```

`serve.py` 同時做兩件事：服務編輯器頁面，以及提供一個**白名單檔案橋**。有了它，
Firefox 與 Safari 也能存檔——File System Access API 只有 Chrome 與 Edge 有。頁面會
自己偵測 server 在不在：在的話多一顆「用本機 server 開啟」。

`file://` 開不了（File System Access API 需要 secure context，瀏覽器自動化也擋
`file:`），所以一定要透過 server。用完把它關掉。

### 檔案橋的邊界

| 限制 | 為什麼 |
|---|---|
| 只綁 `127.0.0.1` | 不加會把目錄開給整個區網 |
| 讀寫都限制在 `remake/assets/` 底下 | 解析後的真實路徑要仍在那裡，符號連結指出去一律拒絕 |
| 寫入只允許 `.json`；`.png` 只讀不寫 | 戰場編輯器要拿 tileset，但不該改它 |
| 只覆寫既有檔案 | 打錯路徑就多一個沒人讀的檔案 |
| 寫入前驗 JSON 解析得動 | 寫進半份壞檔會讓遊戲在啟動時才失敗 |
| **保不住格式就回 409** | 見下 |

受版控的 JSON 格式不一致：story 用 1 空格縮排、scenario 用 2、`map.json` 是單行且
完全緊湊、`map0_units.json` 連結尾換行都沒有。統一格式會讓每次存檔產生整份 diff，
真正改到的那一行就埋在裡面。所以寫回時逐檔沿用原本的縮排、分隔符與結尾換行。

有些檔案是**手工排版**的（`cutscenes/acting/*.json` 把 `{ "slot": 34, "pose": 2 }`
寫在同一行），程式化的 dump 重現不了。那些一律拒絕寫入而不是硬寫——
[`tools/test_editor_server.py`](../test_editor_server.py) 釘住「每一份要嘛保真、
要嘛被拒，沒有第三種」。

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

## 劇情編輯器

開啟時選 `remake/assets`（裡面要有 `story/` 與 `scenarios/`）。四個分頁：

| 分頁 | 改什麼 | 寫回 |
|---|---|---|
| 對白 | `scenes[].lines[]` 的說話者與台詞 | `story/chNN.json` |
| 戰場事件 | `events[]` 的 id、觸發、回合與動作 | `scenarios/chNN.json` |
| 商店 | shop 節點的 `goods[]` | `scenarios/campaign_full.json` |
| 節點轉場 | 每個節點的 `next`／`on_win`／`on_lose`（表格） | `scenarios/campaign_full.json` |
| 節點圖 | 同上的圖形版，加上旗標、choice 選項與敗北路線 | `scenarios/campaign_full.json` |

說話者下拉從那份檔案現有的台詞收集，不另外維護一張會漂的名單。轉場下拉只列得出
現有節點——`campaign.Decode` 會拒絕斷裂的轉場，在這裡擋住比在遊戲啟動時失敗好。

事件動作只有 `dialogue`、`spawn_group`、`spawn_party`、`pan`、`delay` 有表單；其餘
型別給原始 JSON 編輯，欄位名不猜。猜錯會寫出引擎讀不動的事件，而那要玩到那一關
才會發作。

### 節點圖

依章節分層畫出節點與轉場：next 灰、on_win 綠、**on_lose 紅虛線**、options 黃；會設
旗標的節點標 ⚑；指到別章的轉場畫成節點右邊的文字（不畫會讓人以為那一章是死路）。

改轉場用「按『改連到…』再點目標節點」，不是自由拖曳——299 個節點擠在一起時，拖到
隔壁節點的機率比拖對還高，而接錯的轉場要玩到那一關才會發作。

屬性面板還能勾選這個節點要設哪些旗標、新增旗標、編 choice 選項的顯示條件，以及直接
加一個 choice 節點或替 battle 接一條敗北路線——那是 doc 38 Phase 3 驗收要的兩樣東西。

### 改到 campaign_full.json 之後要重生 canonical

正式執行期讀的是 `remake/assets/editor-canonical` 這份 bundle（見
[`60`](../../docs/knowledge-base/60-editor-separated-assets-spec.md) 四之二），不是
legacy JSON。改完要重跑：

```bash
python3 tools/export_editor_canonical.py \
  --output remake/assets/editor-canonical --without-animations
```

不重生的話遊戲裡看不到這次改動，而回歸會以「canonical 編出的戰役與 legacy 原檔
不一致」失敗。編輯器存回時也會提醒。
