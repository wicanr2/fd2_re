# 60 — 編輯器資料契約與分離素材包規格

> 狀態：**READY（第一階段）**  
> 日期：2026-08-28  
> 範圍：先建立可驗證契約、來源驗證、素材清冊及逐家族遷移；不在本規格中猜測
> 尚未證實的 handler／renderer 語意，也不授權散布原版或轉製素材。

## 一、實檔稽核結論

現有資料足以證明大量內容已可由 JSON 載入，但**不足以直接建立安全的完整編輯器**。
目前主要缺口如下：

1. campaign node 有 map key，但 campaign、map、scenario、story、unit、event、line、
   beat、animation 與 frame 沒有一致的版本及穩定識別碼。
2. 多處 runtime 仍用陣列 index、raw slot 或 resource number 互相引用。編輯器插入或
   重排一筆資料，可能靜默改變對話、增援、動畫或角色身份。
3. 現有測試主要證明 JSON 可被 loader 消費，尚未證明「載入→編輯模型→寫回→再載入」
   保持未知欄位、原始來源與語意。
4. `portrait`、`fig`、`map_selector_key`、`battle_fig` 與 persistent identity 是不同
   概念；目前缺少面向創作者的角色身份關聯表。
5. FIGANI、AFM、DATO、TAI、BG、介面格、音效與音樂雖已有個別匯出器，尚未形成
   一份可驗證、可定位來源、供 runtime 與編輯器共同使用的完整素材清冊。
6. 正式 runtime 仍有多條玩家路徑即時讀取 `FDOTHER.DAT`、`FDTXT.DAT`、`DATO.DAT`、
   `FIGANI.DAT`、`BG.DAT`、`TAI.DAT` 與 `FDFIELD.DAT`。因此目前不能宣稱遊戲只
   消費分離素材。

## 二、權利與執行邊界

- 公開儲存庫只保存引擎、schema、匯出器、來源雜湊與不含原版內容的清冊模板。
- 玩家合法自備原版。匯出器先依
  [`fd2-reference-files.json`](../data/fd2-reference-files.json) 驗證檔名、大小、MD5
  與 SHA-256，再於本機建立不入版控的分離素材包。
- 遊戲啟動只驗證「原版來源／已匯出素材包的 provenance」。一旦進入正式 runtime，
  renderer、音訊、對話與規則只能讀分離素材包；不得偷偷回退讀 `.DAT`。
- 缺檔、hash 不符、schema 不符或交叉引用失效時採失敗即關閉（fail-closed），
  並回報精確 `asset_id`。不可用看似成功的猜測畫面掩蓋缺口。
- 原版忠實主題與現代美術主題共用相同 gameplay／story data，只替換 presentation
  catalog。現代素材不得覆蓋或改寫忠實素材的身份與來源證據。

## 三、分離素材包契約

素材包根目錄固定包含 `manifest.json`，至少具有：

- `schema_version`：目前為 `1`。
- `pack_id`：例如 `fd2-original-b97caf22`；不可只靠目錄名推斷版本。
- `source_set`：每個原始檔的檔名、大小、MD5、SHA-256；不得包含主機絕對路徑。
- `assets[]`：每筆含穩定 `asset_id`、`kind`、相對 `path`、輸出 SHA-256、來源檔、
  原始 resource／frame／offset、尺寸或時長、palette、透明規則及證據等級。
- `relationships[]`：角色、嘴型、動畫、音效 cue、地圖 tileset 等跨檔引用。
- `generated_by`：工具版本與命令契約；不得記錄授權檔或私人 ROM 路徑。

第一階段標準輸出家族：

| 家族 | 標準檔 | 必要附檔 |
|---|---|---|
| 地圖與圖塊 | PNG＋JSON | cell、terrain、event raw provenance |
| 地圖人物 | PNG sprite sheet／獨立 frame | direction、frame、selector provenance |
| 頭像與嘴型 | PNG | identity、portrait resource、mouth frame |
| 戰鬥／過場動畫 | PNG frame | animation JSON、placement、delay、raw markers |
| 背景與介面 | PNG | palette、indexed/LUT/blit mode metadata |
| 文字與對話 | UTF-8 JSON | raw chapter、control bytes、line stable ID |
| 音效 | OGG | sample rate、channels、cue provenance、duration |
| 音樂 | OGG | source cue、音源版本、loop／stop metadata |
| 字型 | PNG atlas＋JSON | glyph mapping、cell geometry、source bank |

PNG／OGG 是 runtime 消費格式；raw `.bin` 只可留在研究中間目錄，不算完成的獨立
素材。若某資源尚無足夠證據轉成上述格式，manifest 必須列為 `blocked`，不可省略後
仍宣稱全量完成。

## 四、編輯模型契約

所有可編輯文件都必須有 `schema_version`、`document_id`、`kind` 與 `source`。
穩定識別碼在同一 campaign pack 內不得重用；顯示名稱可改，識別碼不可因排序改變。

最低識別層如下：

- `character_id`：連結 persistent identity、portrait set、map sprite set、battle
  animation set 與顯示名稱；raw selector 分別保存，不能合併成一個 `fig`。
- `node_id`、`battle_id`、`map_id`、`scenario_id`：戰役與戰場的穩定引用。
- `unit_id`、`event_id`、`action_id`：增援、死亡條件及腳本不可依陣列位置引用。
- `scene_id`、`line_id`、`beat_id`：插入台詞或演出後，binding 仍指向同一內容。
- `asset_id`、`animation_id`、`frame_id`、`audio_cue_id`：presentation 只引用 catalog
  identity，不引用 `.DAT` 路徑或未分型 resource number。

編譯器負責把穩定 ID 解析成 runtime 所需的陣列、raw slot 與 index，並產生可追溯
映射。未知 `kind`、未知 action、重複 ID、斷裂引用、陣列尺寸不符及不合法欄位組合
必須拒絕；不能由 Go 的零值靜默吸收。

## 五、往返與相容性

- 舊 JSON 只作 legacy import。匯入時產生穩定 ID 與 provenance，並輸出診斷；不在
  原檔上就地猜改。
- 編輯器 canonical JSON 必須通過：decode → validate → encode → decode → validate，
  且 stable ID、未知 extension、raw provenance 與交叉引用不變。
- runtime 可暫時由 compiler 產生舊形狀，讓遷移逐家族進行；但正式編輯器不得把
  runtime index 當內容身份。
- capability catalog 必須由程式碼／schema 產生，列出實際可用的 node、trigger、
  condition、action 與 renderer。舊文件願景詞彙不得出現在可選清單中。

## 六、實作順序與驗收

1. **契約基礎**：加入 manifest schema、來源驗證器、穩定 ID validator 與 legacy
   import 診斷。驗收：錯誤 hash、重複 ID、斷引用及未知 action 全部拒絕。
2. **全量匯出清冊**：統一呼叫既有 decoder，輸出 PNG／OGG／JSON 與逐檔 hash。
   驗收：每個原始 archive resource 都是 exported、intentionally-raw 或 blocked，
   不允許無紀錄遺漏。
3. **runtime 遷移**：依 title／UI、story／portrait、map、battle animation、ending、
   audio 分批改讀 catalog。每批加入「原始 archive 不可讀、分離素材仍可運作」測試。
4. **編輯器最小垂直切片**：編輯一張地圖、一個 unit、一個 event、一句 dialogue、
   一個 portrait 與一段 animation，編譯後由正式 runtime 顯示並可存讀。
5. **現代美術原型**：從已分離且具 `asset_id` 的一組頭像、戰場人物、圖塊與介面框
   各做忠實版／現代版對照；經使用者選定風格後才建立正式 theme pack。

### 2026-08-28 第一輪全量匯出實測

固定版本原版已在 `fd2-assets-local:20260828` 一次性容器內完成全量試跑，實際輸出
寫入被版控忽略的 `remake/generated-assets/fd2-original-b97caf22/`。結果為1,005個
archive subresources、125張一般 PNG、264組 FIGANI／2,118張動畫 frame、136組
頭像／544張嘴型 frame、33張地圖、1張字型 atlas 與15首 MIDI。完整機器清冊見
[`asset-export-audit-20260828.json`](../data/asset-export-audit-20260828.json)。

這次結果同時證實現有 `extract_all.py` **不是完成版素材包產生器**：409個 FIGANI
中只有264個被接受，音樂仍是 MIDI，尚未把音效、AFM／ANI 與所有 UI 資源建立
逐檔 `asset_id`／hash／用途。這些缺口已明列，不以「全量試跑成功」冒稱所有素材
皆已轉成正式 PNG／OGG。

## 七、完成定義

只有同時成立才可宣稱「素材已完全分離、JSON 足以建立編輯器」：

- 原始資源清冊零無紀錄遺漏；blocked 項目有明確原因且不在正常玩家路徑。
- production Go 程式除來源驗證／匯入命令外，不再呼叫原始 archive decoder。
- 沒有 `.DAT` 可讀權限時，既有代表性玩家抽樣仍能由分離素材包通過。
- canonical editor 文件通過 schema、跨檔引用與往返測試。
- 編輯器最小垂直切片能改變正式畫面／流程，存檔後重開仍保持結果。
