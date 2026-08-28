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
5. FIGANI、DATO、TAI、BG、介面格、音效與音樂雖已有個別匯出器，尚未形成
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
頭像／544張嘴型 frame、33張地圖、2張字型 atlas 與15首 MIDI；後續再由嚴格匯入器
加入 ANI 等具型別素材。完整機器清冊見
[`asset-export-audit-20260828.json`](../data/asset-export-audit-20260828.json)。

這次結果同時證實現有 `extract_all.py` **不是完成版素材包產生器**：409個 FIGANI
中只有264個被接受，音樂仍是 MIDI，尚未把音效與所有 UI 資源建立
逐檔 `asset_id`／hash／用途。這些缺口已明列，不以「全量試跑成功」冒稱所有素材
皆已轉成正式 PNG／OGG。

同一實際輸出樹再經來源 hash gate 與逐檔 manifest generator 驗證；2026-08-28完成
ANI遷移後，現行清冊為38,679筆：37,655筆為已匯出、1,005筆raw完整列為
`intentionally_raw`，19筆明確列為`blocked`（包含15首等待 OGG 的 MIDI）。清冊驗證器
拒絕重複 ID、斷裂引用、路徑逃逸、來源／輸出
hash 不符；`blocked` 項目不必偽造不存在的輸出 hash。

#### 2026-08-28 指令格遷移結果

`fd2-asset-import` 已以固定 hash `FDOTHER.DAT` 實際輸出1份 palette JSON及78張
action-cell PNG；重新產生的完整 manifest 為3,886筆，其中2,866 exported、15
blocked、1,005 intentionally_raw。正式 `loadNativeUIPalette`／
`loadNativeActionCells` 已不再讀 archive；測試把 `FD2_ORIGINAL_FDOTHER` 指向
不存在檔案，仍從 `FD2_ASSET_PACK` 載入256色與78格。缺第78格或 DAC component
超過63時整批拒絕。本切片達 `RUNTIME-E1`，不外推其他仍讀 `FDOTHER.DAT` 的 UI、
戰鬥演出或終局 consumer。

### 終局 FDOTHER #56..#60 索引素材契約

> 規格狀態：**CONFORMED**
> 證據邊界：固定雜湊 `FDOTHER.DAT` 與既有 `0x2c194..0x2c39a`、`0x2c405..0x2c548`
> caller／consumer；不重新推論 `0x28a6c` 的完整 renderer 語意。

已閉合的原始選擇器為：#56 party cycle 的 320×200 backdrop、#57 的 768-byte VGA DAC、
#58 的 20-entry 四模式 RLE frame table、#59 的最終 320×200 單影格，以及 #60 的前置
320×200 單影格。規格證據見 `56` 的終局段落及既有 `MontageCycleAssets`／
`MontageTailAssets` shape tests；不得把 #57 誤列為 frame source，也不得把 #58 誤列為
palette。

匯入器必須先驗證完整 `FDOTHER.DAT` 大小、MD5 與 SHA-256，再輸出：
`surfaces/FDOTHER_056|059|060/` 各保存 P-mode indexed `frame.png`、L-mode
binary `mask.png` 與 metadata；#57 保存 768 個 `0..63` DAC component 的 JSON；#58
保存於 `animations/FDOTHER_058/` 的 animation metadata、20 張 indexed frame 及20張
binary mask。metadata 必須保留
source resource、frame index、geometry、placement／delay raw 欄位與來源雜湊。一般
RGBA 預覽不得取代 indexed frame＋mask。

共用 strict loader 必須一次預檢 caller 所需的完整集合，驗證來源、resource、frame
連續性、PNG mode、二值 mask、幾何與 #57 DAC 值域；任一缺件時不得發布半套結局資產，
也不得回退讀取 `FDOTHER.DAT`。`LoadMontageCycleAssets` 改接 #56，
`LoadMontageTailAssets` 改接 #57..#60；既有 FDOTHER#5 對話框格另由已分離 item-panel
契約處理，不混入本切片。

驗收須逐 indexed pixel、mask、placement、delay 與最終 blit 對照固定原檔；在
`FDOTHER.DAT` 不可讀時，party cycle 與20段 tail 仍可完成同一 typed preflight，並保留
#59 最終畫面。破損 #57、缺任一 #58 frame 或錯誤 #59／#60 幾何皆須失敗即關閉。
達成後只提升終局素材 `DATA-READY`／`RUNTIME-E1`，不冒稱終局一般玩家 E2。

2026-08-28 實作已完成上述輸出，並把 FDOTHER#5 對話框格的 caller-proven raw entries
由23筆擴為40筆（新增 #1..17），使 party cycle 也不再讀 archive。固定原檔逐 indexed
pixel、mask、placement、delay 與非零 destination blit 對照均一致；#57 的768 bytes逐 byte
一致。正式 `ending_preview` 在只指定 `FD2_ASSET_PACK`、沒有提供 `FDOTHER.DAT` 給 tail
loader 的 Xvfb 抽測通過，缺完整 separated roots 會在建立 player 前失敗即關閉。完整
manifest 現為38,092筆：37,068 exported、1,005 intentionally raw、19 blocked。本切片達
`DATA-READY`／`RUNTIME-E1`；`LoadMontageTailAssetsArchive` 只保留給 source oracle。

### 商店 FDOTHER #12／#29／#63 索引素材契約

> 規格狀態：**CONFORMED**
> 證據邊界：固定雜湊 `FDOTHER.DAT`、`0x2e341` 的 raw hub variant 分派，及
> `DecodeNativeShopAssets` 已有 caller-proven entry contract；不替未被 caller
> 消費的 entry 猜測用途。

固定原檔大小為3,382,481 bytes，MD5為
`22f56e5027edc7c766ad34ca4e5aca93`，SHA-256為
`a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce`。
`0x2e341` 的 raw hub variant 1／3／5分別選 resource 12／29／63；12與63是
`LLLLLL` 外層目錄，29是含 terminal boundary 的場景型 `LMI1`。三者共同的
caller-proven entry為：#0 320×200背景、#1 63×15 opaque decoration、#2 6×99
gold-roll strip、#3..#10八個 service cells、#15 price cell、#16 panel、#18..#22
五個 compare cells。#23起的success frame數依序為5／1／7；其已證實呈現計畫
分別是`(169,45)`每幀2 ticks並restore portrait、`(148,39)`前1後8 ticks並restore，
以及`(131,28)`每幀2 ticks且不restore。

匯入器須為每個resource輸出獨立 `shop/FDOTHER_NNN/resource.json`，保存來源檔
身份、resource、container kind、entry count，以及每項標準輸出的原始entry index、
codec與geometry；完整pack manifest另保存每個輸出檔的內容雜湊。背景、各格與success frame均保存P-mode indexed PNG；
codec具有透明語意者另存L-mode二值mask。success另有 `animation.json`，保存嚴格
幀數、destination、ticks及portrait restore raw契約。一般RGBA預覽不能取代indexed
frame＋mask；未被caller證實的entry只保留在來源oracle，不升格成正式typed用途。

正式loader必須一次預檢12／29／63完整集合，驗證schema、固定來源身份、resource、
container kind、entry index、P／L mode、二值mask、geometry、內容雜湊及success計畫；
任一缺件即整批失敗，不回退讀取`FDOTHER.DAT`。`DecodeNativeShopAssets`改為明示的
archive source oracle，正式`loadNativeShopUIAssets`只接受分離包。

驗收至少包含：三資源與固定原檔逐indexed pixel／mask／geometry／success timeline
比較；破壞來源身份、entry index、6×99或320×200幾何、PNG mode、mask及success幀數
皆須失敗即關閉；在`FDOTHER.DAT`不可讀且只指定`FD2_ASSET_PACK`時，正常城鎮進商店、
服務選單、一次購買成功／扣款及返回原城鎮仍可完成。達成後只提升商店素材
`DATA-READY`與正式路徑`RUNTIME-E1`，不冒稱未修改原版`PLAYER-E2`。

2026-08-28 已依本契約輸出67張indexed PNG、43張二值mask與3份metadata，共113筆
標準素材。三個resource的背景、格件與5／1／7張success frames均與固定原檔逐indexed
pixel及mask一致；正式`loadNativeShopUIAssets`一次預檢三種variant，已不呼叫archive
decoder。建立尚未遷移的共用設施bundle後，把`FD2_ORIGINAL_FDOTHER`改指向不存在檔案，
三種商店專屬資源仍能載入；正式選單與購買流程Xvfb抽測通過。完整manifest現為38,092筆：37,068
exported、1,005 intentionally raw、19 blocked。本切片達`DATA-READY`／`RUNTIME-E1`；
`DecodeNativeShopAssets`只保留作來源oracle；共用設施bundle的其他FDOTHER consumer仍是
獨立缺口，正常城鎮進出商店也不外推為原版E2。

### 城鎮 FDOTHER #10／#11／#61／#62 索引素材契約

> 規格狀態：**CONFORMED**
> 證據邊界：`0x2cd46..0x2d05a`、現有三variant整幀E2及
> `DecodeNativeTownAssetsArchive`；不重新推論設施高階語意。

固定原檔的#11／#61／#62是三張320×200 caller-selected背景；#10是由opaque high-run
codec解出的62×26目前選項標籤。FDICON sprites 0／1／2及其`0,1,2,1` pulse已由既有
分離bank閉合，不重複輸出。本切片只新增三個`surfaces/FDOTHER_NNN/`標準P-mode背景
與各自L-mode mask，以及`ui/fdother_010_town_label/`的P-mode opaque label和metadata。

匯入器須先驗證完整FDOTHER身份，metadata保留resource、raw size、codec、geometry與
已證實證據等級；manifest保存逐檔雜湊。strict loader必須一次預檢三背景、label與既有
FDICON bank，驗證來源身份、PNG mode、二值mask及固定geometry；任一缺件整批失敗，
不得回退archive。正式`loadNativeTownUIAssets`只接受分離pack；archive decoder只供oracle。

驗收須逐indexed pixel／mask對照固定原檔，並在`FDOTHER.DAT`不可讀時抽測三variant、
selection pulse及城鎮進設施再返回的正式路徑。完成只提升本素材`DATA-READY`／
`RUNTIME-E1`；既有route-patched視覺E2證據維持原範圍，不外推未修改長程玩家E2。

2026-08-28 已輸出三背景的3張indexed PNG、3張二值mask、3份surface metadata，及
#10 opaque label的1張indexed PNG與1份metadata，共11筆。固定原檔逐indexed pixel／
mask、三variant×六selection×四pulse共216組完整composite均一致；正式town loader在
`FD2_ORIGINAL_FDOTHER`指向不存在檔案時仍可預檢。manifest現為38,092筆：37,068
exported、1,005 intentionally raw、19 blocked。本切片達`DATA-READY`／`RUNTIME-E1`。
variant2既有contact sheet另重新裁切驗證五組raw RGB AE=0，舊不明語意MD5保留為legacy
欄位，新的逐像素MD5／SHA-256已寫回證據JSON。

### 標題發行商 FDOTHER #74／#76 索引素材契約

> 規格狀態：**CONFORMED**
> 證據邊界：`fd2_title_scroll_schedule_ida.txt`及
> `decodeNativeTitlePublisher`既有caller／codec／shape tests；不包含後續捲動、標題淡入或ANI。

固定FDOTHER #74是320×200四模式RLE單影格，#76是768-byte VGA六位元DAC；兩者合成
開場最前端發行商畫面。匯入器須輸出`surfaces/FDOTHER_074/`的P-mode indexed frame、
L-mode binary mask與metadata，以及`palette/fdother_076.json`的768個`0..63` components。
metadata與manifest保留固定來源身份、resource、raw size、codec、geometry及逐檔雜湊。

正式loader一次驗證surface與palette契約後建立320×200 `image.Paletted`；任一缺件、錯
resource、PNG mode／geometry、非二值mask或DAC越界均失敗，不回讀`FDOTHER.DAT`。
`decodeNativeTitlePublisher`只保留作archive source oracle，正式`loadTitleAssets`只走
分離pack。驗收須逐indexed pixel、mask、DAC及最終paletted RGBA對照固定原檔，並在
FDOTHER不可讀時完成publisher preflight。本切片完成只達`DATA-READY`／`RUNTIME-E1`；
#69..#73/#101、#7/#8與ANI 0/1/3..8另依後續契約遷移。

2026-08-28 已輸出#74的indexed PNG、binary mask與metadata，以及#76 DAC JSON，共4筆。
分離loader與固定原檔的320×200 indexed pixels、mask及256項palette逐項一致；缺完整pack
失敗即關閉。正式`loadTitleAssets`的publisher分支只呼叫分離loader，archive decoder保留
為oracle。manifest現為38,092筆：37,068 exported、1,005 intentionally raw、19 blocked。
本切片達`DATA-READY`／`RUNTIME-E1`，不外推後續title phases。

### 標題捲動、靜態幕、主選單與調色盤素材契約

> 規格狀態：**CONFORMED**
> 主證據：[`fd2_title_scroll_schedule_ida.txt`](../data/ida/fd2_title_scroll_schedule_ida.txt)
> 的 `sub_1F894`、`sub_1F73F`、`sub_1F81E`、`sub_286BD` caller／consumer；輸入
> 身份沿用本文件固定 `FDOTHER.DAT` 雜湊。舊研究 PNG 只作交叉檢查，不是資料來源。

本切片把仍由正式標題程式直接讀取的 FDOTHER 資料轉成可編輯的索引素材：

- #69..#73：五張 320×147 單影格，依資源順序組成 320×735 捲動面；#101 是其
  768-byte VGA DAC。
- #100＋#99：`esi=450` 插播的 320×200 靜態幕與專屬 DAC；#75＋#76：`esi=10`
  插播的 320×200 靜態幕與專屬 DAC。#76 已由發行商切片輸出，不重複建立身份。
- #7 是 23,377-byte 巢狀 `LLLLLL`，directory count為8：entry 0 是 320×200 標題
  base；entry 1..6 依序是 61×7、61×7、62×7、62×7、62×8、62×8 的三個主選單項
  未選／選中影格；entry 7 是零長度尾項，不得輸出成假影格。#8 是共同的768-byte
  VGA DAC。巢狀 entry 必須保留 outer resource與nested index，不能攤平成看不出來源的
  任意 PNG 名稱。

根資源單影格沿用 `surfaces/FDOTHER_NNN/{frame.png,mask.png,resource.json}`；#7 巢狀
影格使用 `surfaces/FDOTHER_007/entry_NNN/{frame.png,mask.png,resource.json}`。每份 metadata
都須保留 outer archive 固定身份、outer resource、nested index（根資源不得假造 nested）、
raw entry size、codec、geometry與證據等級。調色盤使用 `palette/fdother_NNN.json`，完整
保留 768 個 `0..63` component。

正式 loader 必須一次預檢五張捲動面、兩張靜態幕、#7 的七個有效巢狀影格，並驗證
其原始directory count為8及entry 7空尾項，再載入 #8／#99／#101（並重用已驗證 #76）
建立 `titleAssets`。任一 metadata、PNG mode、geometry、binary mask、nested count／index
或 DAC 契約不符即失敗，不回讀 `FDOTHER.DAT`，也不把
`remake/assets/title/*.png` 當 production fallback。組合與 palette 插值仍沿用已閉合的
runtime 排程；本切片只遷移資料 owner，不重新解釋時序。

驗收至少包含：所有根／巢狀影格逐 indexed pixel 與 mask 對照 archive oracle、三份新
DAC逐 byte 對照、320×735 組合與兩組20幀 palette transition逐像素對照、六張主選單項
與兩張靜態幕的最終 paletted 畫面對照，以及沒有原始 archive／缺任一分離檔時的失敗即
關閉測試。ANI.DAT 0／1／3..8 仍是下一份獨立動畫契約，不因本節通過而宣稱標題所有
archive consumer 歸零。

2026-08-28 已輸出7張根資源影格、#7 sub0..6七張巢狀影格與3份新DAC；每張影格各有
indexed PNG、binary mask與metadata，共45筆。#7 directory count=8及sub7零長度尾項均由
匯入器與loader共同驗證。固定archive oracle的14張影格逐indexed pixel／mask一致，#8／
#99／#101逐byte一致；正式標題捲動、兩段palette transition、主畫面、六張選單字樣與兩張
靜態幕只從分離pack建立。缺pack會在`loadGame`發布任何標題狀態前留下明確錯誤，不再跳過
標題或回退舊PNG／archive。manifest現為38,092筆：37,068 exported、1,005 intentionally raw、
19 blocked。本切片達`DATA-READY`／`RUNTIME-E1`。

### ANI.DAT／AFM 全螢幕增量動畫素材契約

> 狀態：**CONFORMED**（2026-08-29）
> 固定來源：`ANI.DAT`，2,437,547 bytes，MD5
> `81315bcbb78764361c5137ab0f714f7e`，SHA-256
> `be909c71d0f1121b6632ae931d978e990f6d54c830f4e0509cd6862187c4d963`。
> 原版程式證據：播放器 `0x020421`、VM 派發 `0x036c9e`、10-entry opcode 跳表
> `0x05276a`；位址皆為固定版本 `FD2.EXE` 物件 2 的線性位址。完整格式證據見
> [`39-ani-afm-format.md`](39-ani-afm-format.md)。

`ANI.DAT` 是含10個目錄項的 `LLLLLL` 容器；#0..#8為有效 AFM v1.00 資源，#9是
零長度尾項，不可匯出成假動畫。AFM 是延續前一幀 framebuffer／palette 的10-opcode
增量繪圖 VM，不是可獨立解碼每一筆 script 的圖片集合。固定實檔標頭如下：

| 資源 | raw bytes | 幀數 | 已知正式 owner |
|---:|---:|---:|---|
| #0 | 1,002,800 | 96 | 標題過場；第20戰天空鑰匙演出亦共用 |
| #1 | 635,952 | 51 | 標題過場 |
| #2 | 97,726 | 26 | 原版結局預覽／合成器 |
| #3 | 35,566 | 28 | 標題過場 |
| #4 | 36,113 | 12 | 標題過場 |
| #5 | 411,039 | 35 | 標題過場 |
| #6 | 43,553 | 12 | 標題過場 |
| #7 | 137,859 | 17 | 標題過場 |
| #8 | 36,893 | 12 | 標題過場 |
| #9 | 0 | 0 | 容器空尾項；不建立 asset |

每個有效資源固定輸出到 `animations/ANI_NNN/`：

- `animation.json`：`schema_version`、`kind=afm_indexed_animation`、穩定 `asset_id`、
  `status`、`evidence`、固定來源 identity、resource index／raw size、AFM title、
  `codec=fd2_afm_vm_v1`、`width=320`、`height=200`、header frame count及有序
  `frames[]`。
- 每幀 `frame_NNN.png`：320×200、PNG palette mode，像素值必須逐 byte 等於 AFM VM
  執行後的64,000-byte framebuffer snapshot；不可只保存轉成 RGBA 後的結果。
- 每幀 `palette_NNN.json`：恰有768個 `dac_6bit_components`，逐 byte保存該幀 VM
  執行後的 VGA 六位元 palette snapshot。PNG 內的八位元顯示 palette只是預覽，
  runtime 的忠實輸入仍以這份六位元資料為準。
- `frames[]` 每筆使用穩定 `frame_id` 並引用對應 indexed PNG及 DAC JSON；不得把
  呼叫端的 delay 或可略過旗標冒充 AFM 檔案欄位。

呼叫端播放契約仍由具名場景資料保存。現行標題 #3/#4/#5/#6/#7/#8/#0/#1 的
tick-per-frame依序為5/5/3/5/3/5/1/1，只有 #3及#1可略過；第20戰及結局各自保留
既有 owner 的時序與合成規則。這些數值只證實原版呼叫選擇／目前 owner，不代表
逐週期 DOS wall-clock parity。

離線匯出器必須先驗證整個 `ANI.DAT` identity、目錄項數10、#9零長度、每個有效
資源標頭幾何320×200、宣告幀數及完整 script 邊界。任一 VM opcode、script、frame
record、幀數或 palette／frame長度不一致時整批拒絕，不可沿用舊 decoder「保留前面
成功影格」的寬鬆行為。正式 loader 同樣一次驗證完整 animation 後才發布 `Clip`，且
不得在缺檔或 metadata 錯誤後回退讀取 `ANI.DAT`。

驗收至少包含：九個有效資源共289幀的 indexed framebuffer及六位元 palette逐幀與
archive oracle一致；#9沒有輸出；缺 frame／palette、錯誤 PNG mode／geometry、錯誤
frame count／source identity全部失敗即關閉；標題、第20戰與結局在原始 `ANI.DAT`
不可讀時仍從分離pack工作。完成同狀態原版畫面比較前，本切片最高只可標
`RUNTIME-E1`，不可冒稱`PLAYER-E2`。

2026-08-29 已依本契約輸出9份animation metadata、289張indexed PNG及289份六位元
DAC JSON，共587筆；#9空尾項沒有輸出。嚴格 loader 與固定archive oracle已逐一比較
九個資源的289組64,000-byte framebuffer及768-byte palette，全數相同。標題、第20戰
天空鑰匙與結局預覽均改讀`animations/ANI_NNN`；production對`afm.DecodeResource`、
`FD2_ANI`及ANI archive path的引用歸零。缺pack會在發布標題狀態前失敗；第20戰與結局
聚焦回歸亦已從分離pack建立AFM clip。這些是`DATA-READY`／`RUNTIME-E1`，沒有提升
原版完整啟動或逐幕時序為`PLAYER-E2`。

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

### FDICON.B24 戰場人物銀行契約（CONFORMED）

#### 證據與範圍

固定版本 `FDICON.B24` 為624,010 bytes，MD5
`46f793540209a063ea73a5373ca14bf4`、SHA-256
`7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44`。Docker內
直接讀取header為 `{width=24,height=24,count=1680}`。IDA Pro 9.4證據
[`fd2_fdicon_selector_constructor_ida.txt`](../data/ida/fd2_fdicon_selector_constructor_ida.txt)
已證實 `0x11019`按raw key建立每組12張的快取，`0x127e0`再以
`group*12+pose*3+cycle`消費；既有typed decoder逐張保留four-mode RLE的indexed
pixel、source-write mask與mode-3 destination-remap mask。本切片不重新命名raw key、
pose、cycle或圖像身份，也不把同值推成角色身份。

#### 標準輸出與loader

標準輸出根為 `sprites/fdicon/`：`bank.json`逐筆列出穩定sprite index、24×24幾何、
`frame.png`、`mask.png`及`remap_mask.png`。frame必須是256色indexed PNG；兩張mask
必須是8-bit grayscale PNG且只含0／255。三層不可合併成單一RGBA：mode-3不寫source
pixel，而是在特定consumer中映射既有目的地，普通alpha無法表示。

正式loader必須一次驗證來源檔名、大小、雜湊、1680筆完整且不重複的連續index、固定
24×24幾何、PNG mode與binary mask，再發布一個完整`fdicon.Bank`。缺檔、多檔、錯誤
index、非indexed frame、非binary mask或任何幾何不符均整批失敗，不得回退
`FDICON.B24`。

#### 驗收與邊界

真實固定版本的1680張sprite須逐張比較pixels／mask／remap mask；正式戰場載入在
`FD2_ORIGINAL_FDICON`及相鄰archive路徑不可讀時仍須取得同一bank。此切片只關閉
FDICON人物銀行，不宣稱FDFIELD、FDSHAP、FDOTHER HUD／LUT／palette或完整戰場素材組合
已分離；也不把重製端等價比對提升成原版畫面E2。

#### 2026-08-28 實作與驗收

固定版本已輸出1,680張indexed frame、1,680張source mask、1,680張remap mask及
`bank.json`，合計5,041個標準檔。strict loader逐張重建後，全部pixels／mask／remap
mask與原始four-mode decoder一致；空bank及不完整pack會失敗即關閉。正式
`loadNativeMapAssets`改讀此bank，測試沙箱只在相鄰目錄提供FDOTHER／FDSHAP、不提供
FDICON.B24，仍能載入完整1,680張戰場人物素材。

完整manifest現為13,048筆：12,024 exported、1,005 intentionally_raw、19 blocked。
本切片達`DATA-READY`與`RUNTIME-E1`。戰場bundle、整備、職業／教會、城鎮及終局tail
loader baseline均改接同一strict bank；正式產品程式的FDICON archive caller歸零。
`fd2-asset-import`及名稱明確的source-oracle adapter仍可讀原檔，不屬runtime fallback。

### 第 23 戰重載的 FDFIELD #69 組合格契約

> 狀態：**CONFORMED**
> 範圍：`ch22_post` 尾端切換至 map23 時的 `FDFIELD.DAT` 資源 69；不包含
> `FDSHAP.DAT` 資源 46／47、終局蒙太奇使用的 FDFIELD 資源 90／91／92，亦不
> 宣稱一般玩家路徑 E2。

固定來源 `FDFIELD.DAT` 大小為 243,169 bytes，MD5 為
`ecdb0436d26adfe5d107f2713fa7e9a2`，SHA-256 為
`b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513`。
原版 `FD2.EXE`、IDA Pro 9.4 線性位址、caller／consumer 與直接指令證據見
[`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)：
`0x24A3E..0x24A53` 載入資源 69，`0x24A58..0x24A87` 接續載入 FDSHAP
資源 46／47，`0x24A8C` 再把組合格交給 `0x4DBFC`；證據等級為**已證實**。

正式分離來源是 [`map23/map.json`](../../remake/assets/maps/map23/map.json)。每格
四 bytes 必須依序重建為：`tiles[]` 的 little-endian `uint16`、
`native_composition_event_bytes[]`、`native_tile_blit_modes[]`；檔頭是 little-endian
`uint16 width` 與 `uint16 height`。loader 必須在發布候選狀態前驗證正尺寸、三個陣列
皆恰為 `width*height`、tile 落在 `0..65535`，並重建完整 `4+4*w*h` bytes。
任何缺欄、長度、範圍或 JSON 錯誤均失敗即關閉，不得回退讀 `FDFIELD.DAT`，也不得
只保留顯示用的十位元 tile 而遺失原始高位元。

驗收條件是以固定原檔逐 byte 比較重建結果與 FDFIELD 資源 69，並在正式重載測試的
原版資料目錄刻意不提供 `FDFIELD.DAT` 時，仍完成候選建立、`0x4DBFC` reset 與原子
提交；`FDSHAP.DAT` 和 `FDOTHER.DAT` 在本切片仍由玩家原版唯讀提供。

2026-08-28 實作已讓 `MapData` 正式接收三個組合格陣列，strict loader 會重建完整
6,072 bytes。固定原檔逐 byte oracle、缺 `FDFIELD.DAT` 的正式 staging，以及缺欄／
超出 `uint16` tile 的失敗即關閉回歸均在 Docker／Xvfb 通過。本切片因此達
`DATA-READY` 與窄 `RUNTIME-E1`；原始 beat 上的 `FDFIELD.DAT`／69 名稱仍保留作
原版 provenance，不代表執行期回讀封存檔。

### 全 33 張戰場的 FDSHAP 圖塊與控制銀行

> 狀態：**CONFORMED**

固定來源 `FDSHAP.DAT` 大小為 3,557,794 bytes，MD5 為
`9b0d356074f57cc27aebf3bb89aae247`，SHA-256 為
`901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c`。
已證實的配對契約是 map N 使用圖塊銀行資源 `2*N` 與四位元組控制表資源
`2*N+1`；`0x11EEE` 的正式 renderer consumer、`0x4DEDA` raw blit 與
`0x4DCC6` LUT／目的地重映射分支見 `56` 的 native terrain-frame contract，
第 23 戰 `#46/#47` 的直接 loader／consumer 另見
[`fd2_ch22_post_ida.txt`](../data/ida/fd2_ch22_post_ida.txt)。證據等級為**已證實**；
本切片不重新命名四個控制 bytes 的高階地形語意。

現有 `tileset.png` 是一般 RGBA 預覽，會把 four-mode RLE 的 mode-3 span 當透明，
無法保存 `0x4DCC6` 對既有目的地像素套 LUT 的語意。因此正式標準輸出另建
`tilesets/fdshap/map_NN/`，每張圖塊分別保存 24×24 indexed `frame.png`、binary
`mask.png` 與 binary `remap_mask.png`；`bank.json` 同時保存 map index、原始 image／
control resource、完整四位元組 `controls[]`、來源檔雜湊及每張圖塊的穩定 index。
RGBA `tileset.png` 保留給一般編輯預覽，但不得作 native compositor 的唯一來源。

strict loader 必須驗證 schema、固定來源雜湊、map `0..32`、resource `2*N`／`2*N+1`、
24×24 幾何、連續且不重複的 sprite index、三層 PNG mode、binary mask、controls 非空且
長度為四的倍數。圖塊數與 control record 數不是同一個上限：全量實檔中 map17 為
384／330、map24 為192／180，但兩張地圖實際 base tile 最大值分別為325／120；圖塊銀行
包含 renderer 可選的追加 frame。故 loader 不得要求 `sprite_count<=control_count`，而應由
map composition 驗證每個 base tile 可索引 control，再由 frame selector 個別驗證最終圖塊
index。任一 map 缺檔或不一致時整個該 map 銀行失敗，不得回退 `FDSHAP.DAT`。

驗收分三層：全 33 張銀行逐圖塊比較 pixels／source mask／remap mask，controls 逐 byte
比較相鄰原始資源；正式 `loadNativeMapAssets` 與第 23 戰 reload 在相鄰目錄沒有
`FDSHAP.DAT` 時仍取得相同 typed bank；至少保留 map0、map23、map32 三個早／中／晚
consumer 抽樣。本切片不宣稱 FDOTHER HUD／range／palette 已分離，也不提升戰場畫面 E2。

2026-08-28 實作輸出 33 份 bank metadata、8,256 張 indexed frame、8,256 張 source
mask 與 8,256 張 remap mask，共 24,801 個標準檔。全 33 張銀行及 controls 已與固定
原檔逐層一致；map0／map23／map32 的正式戰場 consumer、以及第 23 戰重載均在測試
目錄沒有 `FDSHAP.DAT` 時通過。正式產品程式的 FDSHAP archive caller 歸零，僅離線
匯入器與名稱明確的 source oracle adapter 保留原檔 reader。本切片達 `DATA-READY` 與
`RUNTIME-E1`；現行38,092筆素材清冊中37,068 exported、1,005 intentionally raw、19 blocked。

### 終局 selector `0x1e` 的 FDFIELD 部署契約

> 狀態：**CONFORMED**
> 範圍：`0x1088d(0x1e)` 建立 31 筆終局部署 record 所消費的 `FDFIELD.DAT`
> 資源 90／91／92；不包含 `0x2c548` 後續狀態變化、`0x28a6c` 精確 renderer admission
> 或一般玩家路徑 E2。

固定來源 `FDFIELD.DAT` 大小為 243,169 bytes，MD5 為
`ecdb0436d26adfe5d107f2713fa7e9a2`，SHA-256 為
`b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513`。原版
`FD2.EXE` 固定雜湊、IDA Pro 9.4 線性位址、caller／consumer 與直接指令證據見
[`fd2_ch29_post_montage_tail_ida.txt`](../data/ida/fd2_ch29_post_montage_tail_ida.txt)：
`0x2c435` 將 selector `0x1e` 傳入 `0x1088d`；後者消費資源 90／91／92、建立
31 筆 `0x50` bytes runtime record。證據等級為**已證實**。

正式分離資料是素材包內的 `fields/fdfield/selector_30/field.json`。它以 `tiles[]`、
`event_bytes[]` 與 `blit_modes[]` 重建資源 90：little-endian `uint16` 寬高後，每格依序為
tile、event byte、blit mode，固定幾何為 35×45，完整長度為 6,304 bytes。同一文件以
`control_bytes[]` 保存資源 91 的完整控制 bytes，並以 32 筆具型別的 `x_word`、`y_word`、
`raw_key` 保存資源 92。文件含 schema、document ID、固定來源雜湊、資源索引、各資源
SHA-256；位置第 0 列保留 control row，其後 31 列保留部署座標及 raw key，不以未證實
的角色或結局語意重新命名。

strict loader 必須先完整解碼並驗證兩份文件，再一次發布三個重建資源。資源 91 header
必須為 `(30,31,1)`、長度 157 bytes、唯一 unit row 的 group byte 必須為 255；資源 92
必須為 little-endian count 32 加 32×6 bytes，第 0 列 key 為 30，其後 31 列 raw key
皆為 0。缺檔、未知 schema、來源雜湊或 resource index 不符、欄位溢位、列數或幾何
錯誤均失敗即關閉，不得回退讀取 `FDFIELD.DAT`。

2026-08-28 實作與驗收已完成：匯入器新增 `-fdfield`，三個重建資源與固定原檔逐 byte
相同；正式 montage baseline 在原版資料目錄
沒有 `FDFIELD.DAT` 時仍建立相同 31 筆 record，並保留首兩筆座標 `(17,18)`、`(1,43)`；
破損分離文件會在任何 runtime record 發布前被拒絕。清冊現為 37,850 筆，其中
36,826 exported、1,005 intentionally raw、19 blocked。本切片達 `DATA-READY` 與窄
`RUNTIME-E1`，但不得據此宣稱終局畫面或 campaign E2。

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
