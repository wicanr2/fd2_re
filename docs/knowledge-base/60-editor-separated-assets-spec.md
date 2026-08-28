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
6. 正式 runtime 仍有多條玩家路徑即時讀取 `FDOTHER.DAT`、`FDTXT.DAT`、
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

### 第一個 runtime 遷移切片：戰場指令格

本切片只搬移已由現行 renderer 正式消費、格式已閉合的 `FDOTHER.DAT #0/#2`：

- 離線匯入命令先驗證 `FDOTHER.DAT` 的固定大小、MD5 與 SHA-256。
- `#0` 的768-byte六位元 DAC 轉成 `palette/fdother_000.json`，保留原始 component，
  runtime 再用既有 `ParseVGAPalette` 建立256色 palette。
- `#2` 的78個 raw cells 各自轉成透明 PNG，檔名固定為
  `ui/action_cells/cell_000.png` 至 `cell_077.png`；index 0 依已證實的 `0x4E9E4`
  destination-preserving 契約轉為透明。
- 正式 `Game` 只能由分離素材根目錄載入上述 JSON／PNG。`FD2_ORIGINAL_FDOTHER`
  只可留給匯入／來源驗證命令，不再是這兩個 renderer input 的 fallback。
- 驗收必須在 archive 路徑不存在時載入完整256色 palette與78 cells；缺任一 cell、
  JSON 格式錯誤或幾何不合法時整批拒絕，不發布半套指令格。

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

同一實際輸出樹再經來源 hash gate 與逐檔 manifest generator 驗證；最初清冊為3,807筆，
後續指令格與 FIGANI 正確分層輸出完成後，現為6,268筆：5,248筆為已匯出、1,005筆
raw 完整列為 `intentionally_raw`，15首 MIDI 明確列為
`blocked`，等待 OGG 輸出。清冊驗證器拒絕重複 ID、斷裂引用、路徑逃逸、來源／輸出
hash 不符；`blocked` 項目不必偽造不存在的輸出 hash。

#### 2026-08-28 指令格遷移結果

`fd2-asset-import` 已以固定 hash `FDOTHER.DAT` 實際輸出1份 palette JSON及78張
action-cell PNG；重新產生的完整 manifest 為3,886筆，其中2,866 exported、15
blocked、1,005 intentionally_raw。正式 `loadNativeUIPalette`／
`loadNativeActionCells` 已不再讀 archive；測試把 `FD2_ORIGINAL_FDOTHER` 指向
不存在檔案，仍從 `FD2_ASSET_PACK` 載入256色與78格。缺第78格或 DAC component
超過63時整批拒絕。本切片達 `RUNTIME-E1`，不外推其他仍讀 `FDOTHER.DAT` 的 UI、
戰鬥演出或終局 consumer。

### 第三個 runtime 遷移切片：FIGANI 索引動畫

264個可由現行已證實 codec 接受的 FIGANI resources，現各自輸出 `animation.json`、
2,118張保留 palette index 的 frame PNG，以及2,118張獨立8-bit mask PNG。frame metadata
保存 x/y、width/height、native delay、raw bytes 4/5/7 與動畫 header bytes 1/2/4；不再
把透明 skip 與不透明 palette index 0 混成同一 alpha。manifest 因此新增264份 metadata
與2,118張 mask，全部幀均含 `source_resource`／`source_frame`。

`figani.LoadSeparatedResource` 只接受上述完整組合；resource 4的7幀已與 archive decoder
逐欄位、逐 indexed pixel、逐 mask 比較一致。終局 party montage 與20段 tail 已改接
分離 loader，缺 metadata／frame／mask 即失敗，不回退 `FIGANI.DAT`。戰鬥通用攻擊、
指令0／1／2／3／5／6／7／8／9的角色與目標待機動畫、指令24／29及32–35的角色、
效果與目標動畫也已改接同一 loader；指令24／29／32的回歸會刻意令原始
`FIGANI.DAT` 不可讀。

尚未遷移的 FIGANI 邊界只剩 `nativeCommand0ActorEffect` 與
`nativeCommand6ActorEffect`：原版會先檢查 `selector*3+2` 的首個 word，為零才退到
前一資源。現行 exporter 未替145個不被 decoder 接受的資源登記「空資源」狀態；在
缺 metadata 時直接套用退回規則會把「未匯出」誤當「原版首 word 為零」，因此維持
失敗即關閉。另有13個 production 檔案仍解碼封裝在 `FDOTHER.DAT` 的戰鬥特效；它們
不是 `FIGANI.DAT` 直接讀取端，須先輸出相同 animation metadata 契約後再遷移。本家族
因此仍是 `RUNTIME-E1-PARTIAL`。

#### FIGANI resource 狀態文件

2026-08-28 以固定雜湊來源唯讀盤點全部409筆：264筆已有完整 `animation.json`；其餘
145筆中，144筆恰為3 bytes `00 00 0a`，首個 little-endian word 已證實為零；#408則是
0-byte resource。落在 `selector*3+2` 的119筆未解碼資源全部屬前述零標頭形狀。這只
證實原版 helper 的機械退回條件，不替空資源命名高階語意。

匯出器現已替每個 `FIGANI_000..408` 產生
`animations/FIGANI_NNN/resource.json`，固定包含：

- `schema_version=1`、`kind=animation_resource_status`、穩定 `resource_id`；
- `source.file=FIGANI.DAT`、`source.resource`、raw size、最多16 bytes的十六進位前綴；
- `status=decoded | empty_header_zero | blocked`；
- `header_word_le`：只有至少2 bytes時存在；
- `reason_code=decoded | zero_header_word | empty_resource | unsupported_shape`；
- `evidence=confirmed`。

`animation.json` 仍只存在於完整可解碼且幀數與 header count 一致的資源。runtime 只有
在 `resource.json` 完整符合來源、ID、`empty_header_zero`、`header_word_le=0`、
`zero_header_word` 與 `confirmed` 時，才可依既有 caller 契約退到前一資源；缺文件、
欄位矛盾、#408空檔或其他 blocked 狀態一律失敗即關閉，不得把「缺匯出」當成零標頭。

實際重生後409份狀態為264 `decoded`、144 `empty_header_zero`、1 `blocked`；manifest
由6,268增為6,677筆（5,657 exported、1,005 intentionally_raw、15 blocked）。
`nativeCommand0ActorEffect`／`nativeCommand6ActorEffect` 已改用
`LoadSeparatedResourceWithZeroHeaderFallback`，指令0正常游標確認測試會刻意令
`FIGANI.DAT` 不可讀；resource2→1另與原 archive decoder 逐欄位相同。正式
`FIGANI.DAT` decoder caller 因此歸零。仍待遷移的是13檔 `FDOTHER.DAT` 內嵌動畫，
不可把它們誤列為 FIGANI archive caller。

#### FDOTHER 內嵌戰鬥動畫

production caller 全量掃描固定為13檔、23個唯一 resource：
`18,19,20,21,22,23,24,25,26,27,28,30,32,33,37,38,39,43,44,65,66,67,68`。
每一筆均由 `fdother.ReadResource` 取得 raw 後直接交給 `figani.Parse`；這證實可以共用
相同 frame／mask／metadata 契約，但來源必須保留為 `FDOTHER.DAT`，目錄與穩定 ID
使用 `FDOTHER_NNN`／`animation/fdother_NNN`，不得標成 FIGANI。

匯出器只對上述已知消費集合產生完整動畫輸出；其他 FDOTHER resource 仍依各自的
LMI1、字型、調色盤、音效或未知格式處理。通用 loader 必須驗證 metadata 的
`source.file` 與目錄 prefix 一致，缺 frame／mask／來源矛盾時失敗即關閉，且不可
回退 `FDOTHER.DAT`。

實際匯出23組共408幀，每幀各有 indexed PNG 與mask，另有23份animation metadata與
23份resource status，共新增862筆標準輸出；manifest 因此由6,677增為7,539筆
（6,519 exported、1,005 intentionally_raw、15 MIDI blocked）。
`LoadSeparatedArchiveResource` 明確驗證 `FDOTHER.DAT`／`FDOTHER_NNN`／
`animation/fdother_NNN` 三者一致；resource18的16幀已與原 archive decoder逐欄位、
逐pixel、逐mask相同。指令0／1／2／3／5／6／7／8／9與32／33／34／35均已改接，
production `figani.DecodeResource` caller 歸零。本子家族達 `RUNTIME-E1`；其餘
FDOTHER palette、音效、LMI1及介面 consumer仍是不同待辦，不因動畫遷移自動閉合。

### 第四個 runtime 遷移切片：BG／TAI 索引單幀素材

> 規格狀態：**CONFORMED**
> 證據邊界：固定雜湊 `BG.DAT`／`TAI.DAT` 的 `LLLLLL` resource，以及已由
> `fdother.ParseSingleFrame`／`Frame.BlitAt` 閉合的 `0x4E63D` 四模式 RLE。這個切片
> 只搬移像素、透明位置及原始 resource selector，不替 selector 命名地形、角色或
> 法術等尚未證實的高階語意。

戰鬥指令0／1／2／3／5／6／7／8／9／24／29／32–35目前以動態 raw selector
成對載入 `BG.DAT` 與 `TAI.DAT`。既有 `images/BG_NNN.png`／`images/TAI_NNN.png`
是研究用 RGB 輸出：它們已遺失 palette index，也無法區分透明 skip 與不透明 index
0，因此不得接到正式 indexed compositor。

正式分離契約固定為：

- 根目錄：`surfaces/BG_NNN/` 或 `surfaces/TAI_NNN/`；`NNN` 是三位十進位原始
  resource number，不能由顯示名稱取代。
- `frame.png`：P-mode PNG，尺寸必須與 raw frame 的 little-endian width／height
  完全一致；每個不透明像素保存原始 palette index。
- `mask.png`：L-mode PNG，同尺寸；`255` 表示原版 RLE 會寫入 destination，`0`
  表示 mode 3 skip 或 mode 1 dither 中保留 destination 的位置。mask 不能由
  `frame.png` 的 index 0 反推。
- `resource.json`：`schema_version=1`、`kind=indexed_surface`、穩定 `asset_id`、
  `status=decoded | blocked`、`codec=fd2_4e63d_single_frame`、width／height、
  `source.file`、`source.resource`、來源 size／MD5／SHA-256、raw resource size、
  `frame=frame.png`、`mask=mask.png`、`evidence=confirmed`。blocked 項目不可附
  假 frame；必須保存 reason code 與 raw size。
- 匯入器必須先驗證 `BG.DAT`／`TAI.DAT` 的固定 size、MD5 與 SHA-256，再遍歷
  archive directory 的實際 resource count。每筆都必須有 `resource.json`；不能只
  匯出目前測試碰到的 selector。
- runtime loader 必須同時驗證目錄 prefix、`source.file`、resource number、尺寸、
  PNG 色彩模型、mask 值域與 metadata。任一不一致即失敗，不回退 `.DAT`，也不
  發布半套 surface。
- loader 回傳的 typed frame 保留 `Width`／`Height`、indexed pixels 與逐像素 mask；
  runtime 合成必須由 mask 決定 destination-preserving 位置，不能把透明處改寫成0。
- source archives、raw resource 與轉製 PNG 均屬玩家自備原版衍生素材，不受引擎的
  PolyForm Noncommercial 1.0.0 授權；不得加入公開儲存庫或發行包。

驗收條件：固定版本所有 BG／TAI resource 均有 decoded／blocked 狀態；至少抽樣一個
含不透明 index 0、mode 1 dither及mode 3 skip 的 resource，以原 archive decoder 與
分離 loader 比較全部 indexed pixels、mask及最終 blit；至少一條正常戰鬥指令在
`BG.DAT`／`TAI.DAT` 路徑不可讀時仍從 `FD2_ASSET_PACK` 完成呈現。缺檔、錯來源、
錯 resource、RGB frame、非二值 mask及尺寸不符測試均須失敗即關閉。

#### 2026-08-28 BG／TAI 遷移結果

固定版本 `BG.DAT`／`TAI.DAT` 各有57筆 resource；兩者#0..55皆以既有四模式 RLE
成功輸出，#56皆為0-byte並列為blocked，不猜測補圖。112筆 decoded resource 共輸出
112張P-mode frame、112張L-mode二值mask與114份狀態文件；包含BG的46個透明skip、
TAI的2,598個透明skip與29個dither run。archive decoder與分離loader已逐 resource
比較尺寸、全部indexed pixels、全部mask及以固定destination完成的最終blit，112筆
全數相同。

戰鬥指令0／1／2／3／5／6／7／8／9／24／29／32–35已全改讀
`FD2_ASSET_PACK/surfaces`；正常玩家指令0測試會刻意令原始 `BG.DAT`／`TAI.DAT`
不可讀。終局20段tail的BG／TAI transaction也已改接同一loader，四項tail player
測試覆蓋20筆全 preflight、決定性重播、最終定格與部分播放前失敗即關閉。
production已無BG／TAI single-frame archive decoder caller；party montage仍直接核對
`TAI.DAT#3`的原始透明placeholder bytes，屬另一個raw resource contract，不能因本次
surface遷移而誤列為已移除。

##### Party montage `TAI.DAT#3` typed adapter

> 規格狀態：**CONFORMED**

原版固定resource恰為7 bytes：`0a 00 03 00 c9 c9 c9`。既有證據只證實它在
`0x29164`／`0x292AD` 路徑擔任10×3 destination-preserving透明no-op；它不是可見
台座，也不可當成FIGANI frame-table。正式runtime可以由已綁定固定來源雜湊、
`source.file=TAI.DAT`、`source.resource=3`與`raw_size=7`的分離surface loader取得
typed frame，但 admission 必須再驗證 width=10、height=3、30個mask值全為0。
renderer只保存這個已證實的no-op gate，不執行blit、不依indexed pixel猜語意。

缺metadata、來源／resource／raw size不符、幾何不符或任一mask為不透明時，party
montage必須在建立player前整批拒絕；不得回退讀取`TAI.DAT`。驗收以正常campaign
montage consumer在`TAI.DAT`不可讀時完成至少一輪figure fade，以及相鄰的非透明
mask fixture被拒絕。這個adapter只移除#3的raw archive依賴，不外推FDOTHER、FDTXT
或其他TAI用途。

實作後 `MontageCycleAssets.TAI003` 改為typed `fdother.Frame`；non-mirror與mirror兩條
fade renderer只驗證全透明no-op，不把它畫入surface。內部loader、雙分支cycle與
非透明相鄰fixture測試通過；正式campaign montage測試另將`FD2_TAI`指向不存在檔案，
仍由分離pack依persistent LOADCH順序進入figure fade、portrait與可選隊伍結果回顧。
本adapter達 `RUNTIME-E1`，production `TAI.DAT` direct reader歸零。

該TAI批次完成時manifest為7,877筆：6,854 exported、1,005 intentionally_raw、18 blocked；
目前總數已由後文FDTXT／字型批次取代。
18筆blocked包含15首尚未轉OGG的MIDI、`FIGANI.DAT#408`及兩個零長度BG／TAI#56；
manifest generator現會讀取`resource.json`的blocked狀態，不再把狀態文件本身誤列
為可用素材。本切片達 `RUNTIME-E1`。

### 第二個 runtime 遷移切片：故事對話 DATO 頭像

- `decode_dato.py --batch` 必須輸出保留原始 palette index 的 PNG，不得先轉 RGB；
  否則重複 RGB palette entry 可能讓 indexed compositor 無法無損反推原值。
- 每個 `portrait_id` 固定包含 `DATO_NNN_m0..m3.png` 四張；四幀必須都是 paletted
  PNG、幾何合法且完整。缺一幀時整組拒絕。
- story dialogue 正式 consumer 由 `FD2_ASSET_PACK/portraits` 讀取四幀 indexed
  pixels，不得呼叫 `DATO.DAT` decoder。原始 resource number 仍由 typed dialogue
  speaker／raw selector 提供，不把角色 identity 與 portrait ID 混為一談。
- 聚焦驗收以同一 `DATO` resource 比較 archive decoder 與四張 PNG 的 width、height
  及全部 indexed pixels，並在 archive 不可讀時完成 story portrait 載入。

#### 2026-08-28 故事頭像遷移結果

136組／544張 DATO PNG 已全數重生為 P-mode indexed PNG。resource 26的四幀經
archive decoder 與 PNG loader 比較，幾何及全部6,400個 indexed pixels／幀一致；
story consumer 在 `FD2_ORIGINAL_DATO` 指向不存在檔案時仍可載入，分離 pack 缺失
時則拒絕，不回退原版 archive。同一 loader 已再接入暫態狀態提示、group march、
整備、教會轉職主介面與商店，並以真實 pack 跑過相關既有 consumer 測試。production
直接 `dato.DecodeResource` 已由10處降至0處；物品／狀態面板、終局 preview與
montage 均改讀同一分離 loader，相關 battle／ending／正式 `cmd/fd2` 測試已使用真實
素材包通過。這只關閉 DATO 家族，其他 archive consumer 仍未全數移除，不可冒稱
runtime 已完全只讀分離素材。

### FDTXT 文字與 FDOTHER#4 字型契約（READY）

本切片沿用已閉合的 offset-table 與 16-bit word parser，不重新猜測控制碼語意。
固定來源是 `FDTXT.DAT`（120,502 bytes，MD5
`fe5c487ce4313485f1da9d48d35b05f9`，SHA-256
`a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69`）及
`FDOTHER.DAT#4` 字型。FDTXT 共35筆 resource；`#0..#33` 可由既有 parser 接受，
`#34` 是0-byte resource，必須輸出 `blocked/empty_resource` 狀態，不可猜補文字。

每筆 `text/FDTXT_NNN/resource.json` 固定包含來源檔、resource、archive hash、raw size、
`status`、`reason_code` 與證據等級。`decoded` resource 另含 `strings[]`；每筆字串至少有：

- 穩定 `string_id=FDTXT_NNN/string_NNNN` 及不受排序影響的原始 `source_index`；
- 有型別 `tokens[]`：普通 word 是 `glyph` 並保留 `glyph_index`，`0xFF00..0xFFFE`
  是 `control` 並保留四位十六進位值；終止字 `0xFFFF` 由容器契約表示，不冒充文字；
- `text` 是依受版控 `glyph_map.json` 產生的 UTF-8 投影。無映射字形須以明確占位 token
  保留索引，不能遺失或替換；
- loader 必須由 tokens 重建既有 `fdtxt.Strings`。若 `text` 與 glyph token 的可重生投影
  不一致、token 種類未知、控制碼範圍錯誤、ID／來源 index 重複或缺號，整筆拒絕。

這份雙層契約讓編輯器顯示與修改 Unicode，同時保留原版逐 word 控制流。編譯器只有在
Unicode 能唯一或依明載策略映射回 glyph 時才可更新 tokens；不能把 UTF-8 顯示字串直接
交給原版 renderer，也不能讓 stale tokens 靜默蓋掉使用者文字。

`fonts/fdother_004/{atlas.png,font.json}` 是獨立契約：atlas 為固定 16×16 cells 的1-bit
遮罩投影，metadata 保存 `glyph_count`、cell geometry、source bank／hash 與 glyph-to-cell
索引。FDTXT resource 不內嵌字型像素；theme 可以替換 atlas，但不可改變忠實資料包的
glyph identity。正式 LOAD、職業／教會／道具面板與終局消費端，必須在原始
`FDTXT.DAT` 不可讀時仍由這兩組分離素材完成；任一 JSON、token、atlas 或來源證明缺失
時採失敗即關閉，不得回退 archive。

#### 2026-08-28 FDTXT／字型第一批遷移結果

固定版本真實匯出得到34筆 `decoded` word table 與1筆 `blocked/empty_resource`
（`FDTXT_034`）。`FDTXT_000` 的661條及 `FDTXT_031` 的46條字串均具穩定ID、UTF-8
投影與lossless typed tokens；#31已逐word與原始resource相同。loader測試另證實會拒絕
UTF-8投影與token不一致的文件。

`FDOTHER.DAT#4`已輸出512×912二值atlas及metadata；1,824個16×16 glyph逐bit與
58,368-byte原始bank一致，灰階中間值、錯誤geometry及來源hash均失敗即關閉。正式
LOAD、職業／教會共用資產、戰場姓名、party montage及終局phase0已改讀分離
FDTXT#0/#31與字型，不回退這兩項archive resource。道具／轉移面板仍有FDTXT archive consumer，故本家族
狀態是 `DATA-READY`／`RUNTIME-E1-PARTIAL`，不是完成。

重生後完整manifest為7,914筆：6,890 exported、1,005 intentionally_raw、19 blocked；
其中35份 `text/FDTXT_NNN/resource.json` 為34 exported＋1 blocked，字型atlas／metadata
另為2筆exported。舊 `images/FDTXT_FDTXT_015.png` 只保留為研究圖像輸出，kind不是
`text`，不可作為runtime文字契約。

### FDOTHER#5 共用道具／狀態面板契約（CONFORMED）

固定 `FDOTHER.DAT#5` 是44,181-byte、138-entry的 `LMI1` directory；directory只提供
entry邊界，不能把相鄰 entry 猜成同一 codec。本切片只輸出已有 caller／consumer
證據的面板集合：

- `entry22`：`0x4E8AF` opaque high-run，149×42；zero pixel覆寫目的地；
- `entries23..30,53..57,59..67,92`：raw width×height bytes，由既有
  `BlitOpaqueAtOffset`消費，zero同樣是實際像素；
- `entries31..52,93,119..129`：`0x4E63D` four-mode frame，必須保存indexed pixels
  與destination-preserving mask。

標準輸出根為 `ui/fdother_005_item_panel/`，包含 `resource.json` 及每個已准入entry的
`entry_NNN/frame.png`；four-mode entry另有 `mask.png`。metadata逐筆保存entry index、
codec、geometry、檔名及來源hash。未列入本集合的其餘entry仍維持未遷移，不因同在
LMI1 directory就自動分類或輸出錯誤codec。

正式loader必須一次驗證完整entry集合、來源 `FDOTHER.DAT#5`、raw size、PNG mode／
geometry、binary mask與無重複索引，再組回既有 `LMI1Entry`／`RawCell`／`Frame`。
缺任一entry、codec不符、opaque entry帶mask、four-mode缺mask或metadata多出未准入
entry時整批拒絕，不發布半套panel。`FDTXT#0`及`FDOTHER#4`則分別由前節typed text／
font loader取得；原始archive不可作fallback。

本切片的驗收是共用道具panel在 `FDTXT.DAT` 與 `FDOTHER#4/#5` 即時reader不可用時，
仍由分離pack完成相同record的indexed輸出；並逐pixel／mask比較所有58筆准入entry。
這只關閉道具／狀態／轉移共用資料，不外推FDOTHER#5其餘80筆entry或所有UI家族。

#### 2026-08-28 實作與驗收

真實固定版本已輸出58筆准入entry：1個opaque、23個raw opaque及34個four-mode frame；
合計58張indexed PNG、34張binary mask與1份metadata。全部entry已依各自原始codec逐
geometry、indexed pixel及mask比對一致；缺完整pack會失敗即關閉。

`LoadNativeItemPanelDataAssets`現在只接受素材包根目錄，並一次載入上述FDOTHER#5、
分離FDTXT#0及分離FDOTHER#4字型。source-oracle另有明確命名的archive adapter，正式
玩家runtime不呼叫它。玩家道具面板在`FD2_ORIGINAL_FDTXT`指向不存在檔案時完成開啟、
12幀生命週期與關閉；教會status／transfer／revive、整備、商店及指令0–3／5–9／24／
29／32–35均改接同一loader。FDTXT正式玩家consumer因此歸零；FDOTHER#5其餘entry與
其他直接consumer仍待各自typed切片。

完整manifest更新為8,007筆：6,983 exported、1,005 intentionally_raw、19 blocked；
其中 `ui/fdother_005_item_panel/` 有92份PNG與1份metadata。此切片達
`RUNTIME-E1`，不外推未修改原版逐像素E2。

## 七、完成定義

只有同時成立才可宣稱「素材已完全分離、JSON 足以建立編輯器」：

- 原始資源清冊零無紀錄遺漏；blocked 項目有明確原因且不在正常玩家路徑。
- production Go 程式除來源驗證／匯入命令外，不再呼叫原始 archive decoder。
- 沒有 `.DAT` 可讀權限時，既有代表性玩家抽樣仍能由分離素材包通過。
- canonical editor 文件通過 schema、跨檔引用與往返測試。
- 編輯器最小垂直切片能改變正式畫面／流程，存檔後重開仍保持結果。

### 2026-08-28 canonical 編輯契約第一批

新增 campaign、scenario、story 與 animation 四份 Draft 2020-12 JSON Schema，要求
`schema_version`、`document_id`、各層穩定 ID、`source` 與受控 `extensions`。另以
`validate_editor_documents.py` 檢查文件 ID 唯一性、戰役轉場、事件／動作／台詞／動畫
frame 重複 ID、對話 mouth animation 及素材 manifest 的 `asset_id` 引用。11項 Schema、
清冊與跨檔 fixture 測試已在素材 Docker 映像通過。

這批是新 canonical 格式的契約基礎，尚未把現有 campaign／scenario／story／FIGANI
資料自動轉入，也尚無未知欄位 round-trip writer；因此狀態是 `SPEC-READY`，不是完整
編輯器或既有 JSON 已全部相容。下一批須提供 legacy importer 診斷、canonical writer
與 load→write→reload 測試，才可提升為 `DATA-READY`。
