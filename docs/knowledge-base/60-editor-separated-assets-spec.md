# 60 — 編輯器資料契約與分離素材包規格

> 狀態：**READY（第一階段）**  
> 日期：2026-08-29
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
6. 2026-08-29 的正式 `Game` caller 稽核未再發現直接讀取上述 `.DAT` 的路徑；
   archive adapter 只留在來源 oracle 與測試工具。這不把 93 筆清冊 `unknown`
   自動升格為標準素材，也不代表三平台完整版已完成實機驗收。

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

### 三之一、原始資源覆蓋清冊

> 狀態：**READY**（2026-08-29）
> 證據範圍：固定雜湊原始容器、目前可重生的 raw resource、分離素材 metadata 與
> 音樂 catalog；本節不新增或猜測任何 `FD2.EXE` handler／renderer 語意。

`assets[]` 的 `intentionally_raw` 只表示研究用 raw 已保存，不能回答同一 resource
是否另有可供 runtime／編輯器使用的標準輸出。manifest schema version 2 因此必須
新增 `source_resources[]`，以每一筆可重生 raw resource 為一列，固定包含：

- `source_file`、`source_resource`：與 `source_set` 及 raw 檔名一致的原始定位；
- `raw_asset_id`、`raw_bytes`、`raw_sha256`：可直接回查的 raw 清冊、大小與雜湊；
- `disposition`：只能是 `standardized`、`confirmed_empty`、`blocked` 或 `unknown`；
- `output_asset_ids[]`：同一 manifest 內真正代表該 resource 的標準輸出；
- `runtime_catalog_refs[]`：素材位於經驗證的外部 runtime catalog 時的受控引用；
- `reason_code`：非 `standardized` 狀態的機械式原因，不以散文猜測用途。

分類優先序固定如下：

1. 只要有 `exported` output asset，或有已驗證的 runtime catalog reference，即為
   `standardized`。保留 raw 不會把它降級。
2. 沒有標準輸出且 raw 長度為零，才可標 `confirmed_empty`；僅因 decoder 沒產圖、
   檔案很短或 resource index 位於尾端，不能推成空項。
3. 沒有標準輸出，但有同 resource 的 `blocked` metadata，標 `blocked` 並保存該
   metadata 的 asset ID；不得把 blocked 當作已完成輸出。
4. 其餘一律為 `unknown`。`unknown` 表示目前清冊尚不能證明標準化，不自動等於
   玩家路徑仍需要它，也不自動觸發新的 executable 反組譯。

複合資產必須由自身已版本化 metadata 建立多對一關聯，不能只靠檔名推測：

- `tilesets/fdshap/map_NN/bank.json` 同時覆蓋其已明列的 `image_resource` 與
  `control_resource`；其中圖塊 PNG 仍只屬 image resource。
- `fields/fdfield/selector_30/field.json` 同時覆蓋其已明列的 `map_resource`、
  `control_resource` 與 `positions_resource`。
- `runtime_catalogs.music` 只覆蓋 catalog `tracks[].resource_index` 明列的
  FDMUS resource；三 bytes header、尾端零長度或未列入 track 的 resource 不得
  因同屬 FDMUS 而自動標準化。

validator 必須證明：每個 raw asset 恰有一列、不重複 `(source_file,
source_resource)`、raw ID／大小／雜湊與 `assets[]` 一致、所有 output ID 存在且來源
關聯可由直接 provenance 或上述複合 metadata 證明、所有 runtime catalog reference
存在。清冊不得引用 raw asset 作標準輸出，也不得以不存在的 output 把 unknown 歸零。

2026-08-29 實檔初步診斷（#1／#6 納入勘誤前）：當時 39,520 筆 manifest 含 1,005 筆 raw；若只比較
`assets[].source_resource`，有 142 組 raw pair 沒有非 raw 關聯。這個數字只是舊
schema 的診斷，不是 142 個 decoder 缺口：至少包含 FDSHAP 34 筆 control resource
與 FDFIELD selector 30 的 #91／#92 關聯漏記。schema version 2 重生後才以
`source_resources[].disposition` 作正式統計，舊的 161 筆工作清單數字應同步作廢。

#### 2026-08-29 實作與驗收

manifest schema version 2、generator version 3 與 validator 已實作上述契約。真實
固定來源重生後現有 39,825 筆 asset，其中 38,801 exported、1,005
intentionally raw、19 blocked；新增的 1,005 筆 source-resource ledger 分為：901
`standardized`、11 `confirmed_empty`、0 `blocked`、93 `unknown`。11 筆空項均由
raw 長度零直接證明；93 筆 unknown 精確分布為 FDFIELD 79、FDMUS 5、FDOTHER 9，
不得再引用舊的 161／142 診斷數字。

可版控摘要見
[`fd2-source-resource-coverage-summary.json`](../data/fd2-source-resource-coverage-summary.json)，
它綁定完整 ignored manifest 的 SHA-256，逐來源保留空項、blocked 與 unknown resource
index。validator 會重算完整 ledger 與摘要；漏列 raw、重複 resource、把 raw 冒充
standard output、偽造複合 metadata 關聯、不存在的音樂 catalog track 或摘要漂移皆
拒絕。加入 FDFIELD runtime bridge 後，28 項相關 generator／validator／catalog／schema
測試與真實 39,825 筆 manifest 驗證已在一次性無網路 Docker 容器通過。

這項成果達 `DATA-READY` 的「覆蓋清冊」層，但不把 93 筆 unknown 自動解讀成
93 個玩家功能缺口。下一批須先用 production consumer 清冊交叉比對；FDFIELD #69
已由受控 runtime catalog 回連，不再列入 unknown，也不重做已閉合 decoder。

### 三之二、FDFIELD runtime catalog bridge

> 狀態：**CONFORMED／DATA-READY**（2026-08-29）

目前唯一可直接證明「正式 runtime map JSON 完整等同單一 FDFIELD resource」的是
`assets/maps/map23/map.json` 對 FDFIELD #69。直接 loader／consumer、逐 byte oracle 與
archive 不可讀測試已在本文件「第 23 戰重載的 FDFIELD #69 組合格契約」閉合；本節
只補 manifest provenance，不重做 decoder 或 `ch22_post` handler。

`assets/maps/fdfield_catalog.json` 必須是獨立且可擴充的 runtime catalog，固定包含
FDFIELD 原檔大小／MD5／SHA-256，以及每筆完整資源的：`resource_index`、穩定
`map_id`、包內安全 `path`、JSON 檔大小／SHA-256、重建 raw 大小／SHA-256、
`evidence_level` 與受版控證據路徑。首批只准入 #69；不得由 map index 乘三批次捏造
其他 mapping。

catalog validator 必須重新讀取 `map23/map.json`，以 catalog 的穩定 `map_id` 與受控路徑
識別既有未重複保存 map id 的 JSON；若文件另含 `map` 則必須等於 23。並驗證正整數寬高、
`tiles[]`／`native_composition_event_bytes[]`／`native_tile_blit_modes[]` 都恰為
`width*height`、tile 可表示為 little-endian `uint16`，再依 `u16le width,height` 與
每格 `u16le tile,event,mode` 重建完整 6,072 bytes。檔案或重建 raw 的大小／SHA-256
任一不符即失敗，不得回讀 `.DAT`。

manifest 的 `runtime_catalogs.fdfield` 保存 catalog 路徑、大小、SHA-256、來源檔與
resource 數；source-resource ledger 以 `fdfield/FDFIELD_069` 受控引用把 #69 從
`unknown` 提升為 `standardized`。validator 必須同時證明引用存在，且 catalog 的
raw identity 與 manifest 中 #69 的 raw asset 完全相同。現有
`native_turn_event_controls.json` 只保存各 control resource 的局部投影，不能把 #1、
#4、#7 等完整 raw resource 標成 standardized；未來只有能重建完整 raw 或另有明確
typed 完整契約時才可加入本 catalog。

2026-08-29 實作已建立 catalog、machine-readable schema、獨立嚴格 validator，並同時
接入 manifest generator 與 pack validator。正式 `map23/map.json` 重建 6,072 bytes，
SHA-256 為 `7f5ac77c56e468a50911373284084e376b574f57b3911a9d4eb870603dd21bf8`，與
manifest 的 FDFIELD #69 raw identity 完全相同。缺 catalog、map 漂移、陣列長度、raw
雜湊、偽造 ledger reference 均失敗即關閉；不需要也不會讀取原始 `FDFIELD.DAT`。

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

### 四之一、多語文字、字型與排版規劃

> 狀態：**DECISION-CONFIRMED／SPEC-READY**（2026-08-29）
>
> 產品決策：使用者已確認繁體中文 `zh-Hant`、簡體中文 `zh-Hans`、日文 `ja`、
> 英文 `en` 四語全部隨正式版內建，並以同一受控契約允許外部社群語言包。

目標語系固定使用BCP 47識別碼：原版繁體中文`zh-Hant`、簡體中文`zh-Hans`、日文
`ja`、英文`en`。四語官方包必須完整隨發行版提供；相同 schema 另允許安裝其他
BCP 47語系的外部社群包。以下資料不變量同時適用兩者：

- 遊戲規則、`character_id`、`node_id`、`scene_id`、`line_id`、event／action及存檔
  狀態永遠不使用翻譯後文字作identity。切換語言不得改變戰役進度、隊伍或亂數狀態。
- 原版`FDTXT_NNN`的glyph token、control word、source index及繁中投影保持唯讀
  provenance；簡中／日文／英文不得倒灌成原版glyph index，也不得覆寫原版證據。
- canonical內容只保存一份演出／控制流程；翻譯以穩定`string_id`連結，不能複製四份
  scene後讓handler、portrait、mouth animation或分頁時序分岔。
- 對話的原始 speaker opcode、operand 與場景／單位定位屬不可翻譯 provenance。
  顯示層只有在角色身分已有來源證據時才可使用正式人名；場景相依的敵兵可用
  「刺客／刺客隊長」等明示角色描述，但必須與原始 operand 分欄保存。禁止把未知
  operand 丟入字模表後，將偶然得到的單一字元當人名送入四語翻譯。
- 所有玩家可見字串角色都要可翻譯：章名、地點、角色顯示名、台詞、選項、商店商品、
  系統訊息、戰鬥命令、狀態／道具／法術名稱及錯誤訊息。只翻`Line.Text`不算四語完成。
- 語言與文字顯示偏好屬`fd2_settings.json`，不寫入戰役save；同一存檔可在載入前後切換
  語言。若日後選擇「每槽固定語言」，必須另立產品決策，不可悄悄改變save schema。

建議的共通語言包形狀如下；這是後續machine-readable schema的輸入草案，不是已完成
實作：

```text
locales/<locale>/
  locale.json          # identity、版本、fallback、字型與layout profile
  strings.json         # string_id → UTF-8文字／受控變數
  fonts/               # 具授權紀錄的TTF／OTF或bitmap atlas
  layout-overrides.json# 只保存確有需要的畫面／文字角色覆寫
```

`locale.json`至少需要`schema_version`、`locale_id`、`display_name`、`content_version`、
`base_locale`、`font_stack[]`及`layout_profile`；所有路徑必須是包內安全相對路徑。
`strings.json`每筆至少需要`string_id`與`text`，可選受控`variables[]`及譯者註記，但
不可嵌入任意程式碼或自行新增handler control。正式內建語系必須通過完整key集合；
開發環境可顯示缺字／缺翻譯診斷，發行模式不可靜默顯示空字串。

文字尺寸分成兩層：

1. 每個`layout_profile`提供語系預設`font_size`、`line_height`、`letter_spacing`、
   CJK／Latin換行規則、標點避頭尾及對話框最大行數。
2. 使用者另有全域`text_scale`偏好；runtime先套語系預設，再套有界倍率。倍率上限、
   是否提供離散「小／標準／大」或連續滑桿，須經640×400與手機prototype決定，不能
   只放大glyph卻忽略panel、游標、portrait與分頁容量。

現行有兩條不能混為一談的renderer：現代UTF-8 UI已有TTF／OTF與wrap能力；原版忠實
畫面則固定320×200、16×16二值glyph、310×86對話框及多處16-pixel行距。`zh-Hant`
忠實主題可保留原版indexed呈現；其他語系若要覆蓋native畫面，必須在下列產品分支中
擇一後才能實作：使用現代字型overlay、為每語系製作對應bitmap atlas，或只保證現代
主題完整翻譯。不可把TTF文字直接塞進原版固定格後仍宣稱pixel parity。

已確認與仍待 prototype 裁決的分支：

1. **已確認**：官方四語內建＋可選外部社群語言包。外部包不得修改戰役、規則、
   handler或存檔，只能提供翻譯、受控字型與layout override。
2. 非繁中語系是否要求原版忠實320×200介面也全面翻譯，或只要求現代主題完整翻譯。
3. 缺翻譯時是整包拒絕、回退`zh-Hant`，或僅外部包允許回退；官方包建議完整key後才
   發行，避免遊玩途中混語。
4. 文字縮放採離散三段或連續倍率；須先製作繁中／英文／日文最長台詞與戰鬥HUD的
   可丟棄prototype，再決定界線。

後續垂直切片固定為：language／layout JSON Schema → key extractor與完整度validator →
`fd2_settings.json`向後相容欄位 → UTF-8 renderer切換 → 對話、商店、戰鬥HUD三種最長
字串prototype → 四語切換／缺字／換行／存檔不變測試。忠實主題翻譯範圍與文字縮放
界線尚未由 prototype 裁決，不得先把建議值寫進正式玩家預設。

#### 現行程式盤點與遷移邊界

2026-08-29 實檔盤點確認，多語系不能只修改故事 JSON：

- `campaign.Line.Text`、`battle.Action.Text`、`battle.DialogLine.Text`與結局節點原先均為
  單一字串。角色、物品、戰鬥raw姓名及原生指令名稱現已各自具穩定 ID 語系契約；職業名稱
  與其餘現代介面字串仍待搬移，不可把已完成的實體目錄外推成全介面完成。
- 現代介面的標題、法術、物品、商店、整備、教會、戰鬥提示與結局仍有大量繁中
  字串直接寫在`remake/cmd/fd2/main.go`及`title.go`。第一階段必須由可重跑extractor
  產生清冊，逐筆搬到受schema約束的字串catalog；不可用人工搜尋一次後宣稱完整。
- 現代`Font`可繪製UTF-8及依寬度換行，但基準大小固定、行高固定，且仍會尋找主機
  字型。正式四語包必須改用包內鎖定版本與雜湊的字型，不得讓平台安裝字型改變
  字寬、分頁或截圖。
- 原生`native_dialogue`的`pages`／`glyph_pages`與FDOTHER #4字模是原版繁中證據。
  它們不得被翻譯覆寫；其他語系使用獨立UTF-8 layout projection，除非日後另有經
  驗證的語系點陣字模主題。

為避免破壞現有內容，遷移期保留舊`text`／`label`作`zh-Hant`來源投影，另加
`string_id`；編譯器負責證明兩者一致。正式canonical資料不得在每個scene內嵌四份
翻譯，也不採`map[locale]text`散落各檔，統一由語言包catalog提供翻譯，避免事件與
演出資料因翻譯而分叉。

#### 可實作契約與驗收順序

1. **字串清冊**：掃描JSON schema、資料檔及Go玩家介面，輸出`string_id`、文字角色、
   來源位置、變數簽章與目前繁中值；同一ID的變數集合不同即拒絕。
2. **語言包schema**：建立`locale.json`、`strings.json`及`layout-overrides.json`的
   machine-readable schema；路徑逃逸、重複ID、未知變數、缺正式key與錯誤BCP 47
   識別碼均失敗即關閉。
3. **字型與排版profile**：每個語系固定包內字型資產、SHA-256、基準字級、字距、
   行距、斷行規則與缺字政策；每種文字角色另定`text_safe_rect`、最大行數及溢位
   政策。使用者的`text_scale`只在profile允許範圍內套用。
4. **相容載入與設定**：舊資料的`text`／`label`仍可匯入為`zh-Hant`，但writer只輸出
   canonical ID。`fd2_settings.json`保存`locale_id`、`text_scale`與主題；戰役save
   不保存語言，切換語言不得改變遊戲狀態。
5. **渲染切片**：依序接對話、商店、戰鬥HUD，再擴到標題、城鎮、整備、教會、
   道具／法術及結局。每條正式路徑不得殘留直接硬編碼的玩家文字。
6. **原型與抽樣**：以最長實際繁中／簡中／日文／英文內容，製作對話、商店與
   戰鬥HUD在640×400及手機縮放下的截圖；檢查字形覆蓋、標點禁則、英文長詞、
   分頁、游標與頭像避讓。這些只證明排版，不宣稱非繁中語系原版逐像素一致。

第一階段完成門檻是：四個官方語系的正式key集合完整；每個字型實際覆蓋所有使用
字元；切換語言及字級後，標題、對話、商店、戰鬥HUD與結局均無裁切；存檔內容逐欄
不變；移除主機字型後仍可重現相同量測。外部社群包允許宣告base locale作開發期
fallback，但正式官方包仍不得靠fallback掩蓋缺譯。

#### 字串候選清冊（2026-08-29）

> 狀態：**DIAGNOSTIC／可重跑；尚非正式翻譯鍵集合**

`remake/cmd/fd2-string-inventory`已把第一階段盤點變成可重跑工具。它只掃描
`remake/assets/story`、`scenarios`、`data`的受控文字欄位，以及正式
`remake/cmd/fd2`中的直接繪製／提示脈絡；測試檔、研究註記、raw證據欄位與純內部
錯誤不列入。完整輸出刻意不入版控，精簡結果見
[`fd2-string-inventory-summary.json`](../data/fd2-string-inventory-summary.json)，重生命令為：

```sh
cd remake
go run ./cmd/fd2-string-inventory -repo .. -output ../docs/data/fd2-string-inventory.json
go run ./cmd/fd2-string-inventory -repo .. -summary -output ../docs/data/fd2-string-inventory-summary.json
```

目前固定結果為5,225筆候選：4,708筆來自JSON欄位、436筆屬直接介面脈絡、2筆由
明確名稱函式辨識，另有79筆只因正式Go來源含非ASCII而列入人工審查。角色名與台詞
等重複值仍逐來源保存，所以這不是5,225個互異譯文；所有ID也仍是來源位置型暫定ID，
不能直接承諾給翻譯者。
完整清冊的SHA-256由摘要綁定，兩次重生必須逐位元相同。

去除完全相同文字後共有2,489個互異繁中字串，145筆候選含格式變數；這仍不能直接
當成翻譯工作量，因為同一文字在不同角色可能需要不同語境，而角色名等實體文字則應
合併到穩定catalog。79筆`go_review`已另以
[`fd2-string-review.json`](../data/fd2-string-review.json)綁定同一清冊SHA-256：30筆確認
玩家可見、42筆為內部診斷、3筆只供開發、4筆維持未知。Go測試要求四類不重複且完整
覆蓋當前79筆；任何來源行移動或內容改變都會讓雜湊失配，禁止沿用過期人工判讀。

已知限制與下一個收斂步驟：

- JSON目前以目錄＋欄位白名單分類，仍須對各文件schema與JSON pointer再做路徑約束；
  `field_rule`表示「符合欄位規則」，不是已證實玩家可見。
- Go直接介面掃描會納入繪圖／渲染函式內的間接ASCII文字，但跨非繪圖函式再送往
  UI的值仍可能漏列；本輪已人工分流79筆，後續新候選仍須重新審查，不能自動升格。
- 同一物件有多個文字欄位時，不會把一個既有`string_id`錯套給全部欄位。正式遷移
  必須分開例如speaker display name與line text的穩定鍵，並把相同實體名稱合併到
  character／entity catalog，而不是依來源位置永久複製。
- `%d`、`%s`、`%w`等格式變數已保存簽章；正式語言包validator仍須比較每個語系的
  變數集合與型別，禁止譯文遺失或新增變數。

2026-08-29 已新增受控語言包 schema、validator 與四語共同的 4-key 戰鬥訊息
垂直切片。官方包必須具有相同 key 集與變數簽章，社群包只能覆寫既有 key 並明列
官方 base locale；不安全路徑、未知頂層欄位與缺官方 key 均失敗即關閉。這只把
契約與四筆真實翻譯提升為 `RUNTIME-E1`：正式 Go loader 與 Python validator 交叉
驗證同一契約，玩家與敵方三條物理攻擊路徑均由 semantic key 格式化；F4 先驗證
完整下一包再原子切換，`fd2_settings.json`保存 BCP 47 `locale_id`，戰役存檔不含
語系。`packageSelfCheck`亦要求四個官方包可載入。這仍**未**把其餘約 2,493 個互異
來源字串、日文字形覆蓋或排版 prototype 冒稱完成。

2026-08-30 下一個正式切片固定先收斂跨場景都會遇到的系統訊息：語系／音源切換、
SAVE／LOAD成功、無存檔、無效節點、戰後禁止存檔與非戰役模式禁止存檔。這些 key
使用穩定的 `runtime.<subsystem>.<role>` provenance，不再綁定會因程式編輯漂移的 Go
行號；既有四筆原版攻擊訊息仍保留原來的 `legacy.go...` 回查來源。Go loader 與
Python validator 必須共享同一 key／變數／provenance 契約，四個官方包缺任一筆即
整包拒絕。正式訊息只能經 catalog 格式化；切換失敗時保留舊 catalog 與遊戲狀態。
這個切片關閉「跨存檔系統訊息會隨 F4 切換」，不外推成完整劇情翻譯。

#### 全量四語內容契約（2026-08-30）

> 狀態：**SPEC-READY**；本節規定全量翻譯的身分、產生、審查與執行期邊界。

全量翻譯不得把目前診斷清冊的來源行號直接當成永久鍵。劇情台詞以
canonical story 的 `line_id` 為身分；場景標籤、章名與地點以 `document_id` 加受控欄位為
身分；角色、物品、法術與固定介面文字以各自的 entity／semantic ID 為身分。舊 JSON
pointer 與 Go 來源位置只保留為 provenance，不得因重排一行程式就使翻譯鍵全部漂移。

四個語系各有一份受版控目錄；每筆至少包含：`string_id`、`id_status`、`role`、
該目錄語言的 `text`、`variables[]`、`source` 與 `status`。原始繁中可由相同
`string_id` 在 `zh-Hant/content.json` 回查，不在每一筆重複保存另一份 `source_text`。
`status` 只能是 `source`、`machine_draft`、`reviewed` 或 `blocked`。離線模型產生的英文與日文只能標成
`machine_draft`；簡繁規則轉換也不自動升為人工審定。契約可以證明「每筆都有完整候選譯文」，
但只有 `reviewed` 才可對外宣稱經人工校譯。

產生器必須在翻譯前保護 printf 變數、控制碼與術語表中的專名，並在後處理復原；
任一 token 遺失、多出、順序或類型改變都必須失敗。繁體中文是唯一來源文本；簡體中文可以
使用 OpenCC 的可重現初稿，英文使用固定版本的離線繁中→英文模型，日文初稿可使用英文→日文模型。
模型名稱、版本、授權、輸入目錄雜湊與術語表雜湊必須記錄在產物。

四個官方包必須具有完全相同的鍵集、變數簽章與來源身分。官方包不允許回退；
任一筆 `blocked`、空文字、未復原 token、英文包殘留 CJK 劇情、或日文包整筆等於英文，
都不得通過內容完整度閥門。執行期只能以 `string_id` 取得文字；在 canonical 內容完成
鍵化前，全量譯文只是 DATA-READY，不得因語言包有文字就提升為 RUNTIME-E1。

2026-08-30 已以目前版本重生 5,225 筆候選清冊，排除 42 筆內部診斷、3 筆開發文字與
4 筆未知後，四語內容目錄各有 5,176 筆玩家內容出現位置。其中 3,333 筆劇情欄位
已以 canonical `document_id`／`scene_id`／`line_id` 作穩定身分；其餘尚無 entity ID 的
項目繼續標成暫定來源身分，不冒充已完成編輯器契約。

簡體中文由 OpenCC 0.1.7 `t2s` 產生初稿，英文由 Argos
`translate-zt_en-1_9` 產生初稿。舊 Argos `en_ja 1.1` 實測會將擇號誤譯成
「お問い合わせ」並大量保留英文，已拒絕為不合格輸出。日文改用固定 revision
`facebook/nllb-200-distilled-600M@f8d333a098d19b4fd9a8b18f94170487ad3f821d`
直接由 `zho_Hant`→`jpn_Jpan`；執行使用 CTranslate2 4.8.1 INT8，以 beam 2、
repetition penalty 1.2、no-repeat trigram 與 128-token 上限防止長句退化。NLLB 模型授權為
CC BY-NC 4.0，只用於本非商業專案的本地候選譯文產生；模型權重不進入儲存庫或發行包。

`validate_full_locale_content.py` 現在會比對四包的 key／role／variables／source、輸入雜湊、
英文 CJK 殘留、日文 Argos 污染詞、異常膨脹及敘事文字整句英文殘留。當前 20,704 筆
四語出現位置通過這些結構與退化閥門，但英文與日文仍全數是 `machine_draft`；抽樣可見
語氣、專名與短感嘆句仍需人工校譯。這是全量候選譯文 `DATA-READY`，不是「已完成官方人工翻譯」，
也不是 runtime 已消費所有 5,176 鍵。

##### Canonical 劇情執行期切片

> 狀態：**SPEC-READY／RUNTIME-E1-PARTIAL**（2026-08-30）；正式 story、handler dialog
> 與戰場內嵌事件61／75／76已接通，只授權劇情台詞，不外推到商店、戰鬥 HUD 或實體名稱。

editor canonical 劇情 JSON 是正式 `line_id` 身分來源；既有 `assets/story` 劇本繼續保存原版
數值 speaker 與執行 metadata。載入器只能在檔名、scene index、line index、繁中 `text` 與
canonical 的 legacy speaker 全部一致時把兩者合併；任一不一致即停止整批，不用陣列位置
猜接。`text` 繼續保存原始繁中來源，`line_id` 才是語系查詢身分；翻譯鍵固定為
`<line_id>/text`。如此不在兩份檔案複製同一永久身分，也不因 legacy 格式仍供原版 handler
使用而放棄 canonical 契約。

執行期啟動及 F4 切換時，必須先完整載入同語系的 `pack.json` 與 `content.json`，兩者都成功
才可原子替換目前語系。故事載入後，每句具 `line_id` 的正式劇情一律從內容目錄取得
`role=dialogue` 的文字；缺鍵、`blocked`、非 canonical 身分、角色不符或空文字都要停止該
對話交易，不可靜默回退內嵌繁中。沒有 `line_id` 的節點內嵌 fallback 仍只屬既有繁中相容
路徑，不得對外計入四語劇情覆蓋。

本切片先保證語言切換後「下一句／下一個節拍」使用新語言。已經 materialize 的目前對話頁
及原生 indexed progressive frames 不在 F4 當幀重建，避免破壞逐字進度、嘴型、開收框與
鏡頭時序；完整即時重排須另經文字安全矩形、字型與分頁規格後才可開啟。原生對話仍保留
原有頁面與動畫載體，不能因翻譯而換成一般 renderer；未完成各語重排前維持失敗即關閉。

2026-08-31 使用者裁決：非繁中原生對話不要求嘴型動畫。繁中仍保留既有原版嘴型收據；
簡中、日文與英文使用閉嘴頭像完成整段 progressive frame，不能因此省略開框、收框、鏡頭、
對話框、頭像或逐字發布。多語分頁固定使用原生 control 的13／14／15 glyph 列寬、三列文字
安全窗口與目前官方字型的實際像素寬度雙重約束；超出三列即建立下一頁，不依賴 painter clipping，
也不把長句截斷。缺字、空頁或任何列超界都停止該對話，不回退原始繁中頁面。

2026-08-31 `RUNTIME-E1` 收據：localized compositor 以官方 TTF 直接產生 palette index
`0xCD` 前景與 `0x4C` 陰影，並共用原生 fully-opened frame、閉嘴頭像、opening／closing
與鏡頭資料。所有目前可由正式 handler binding 編譯的 native utterance 均對簡中、日文與
英文執行完整分頁、字形存在及安全矩形檢查；合成測試也確認 frame0 到最後逐字幀確實改變。
這仍是 renderer E1，不是正常玩家路徑截圖 E2；無法編譯的 handler 本來就由正式入口拒絕，
不納入此語系收據。

事件61／75／76雖由 battle action runner 呈現，文字來源仍是同一批 canonical story line；
它們必須先以 `line_id` 解析當前官方內容，再建立 action。trigger speaker、事件狀態寫入、
palette pulse 與提交時序仍由既有 typed event plan 擁有，語系層只替換文字；解析失敗時不得
先提交事件 mutation。四語測試須各抽一條事件文字，並保留原有事件延遲提交回歸。

### 10.6 商店商品名稱四語契約（READY）

2026-08-31 以 `campaign_full.json` 與四份 `content.json` 交叉核對：正式商店共有337個
商品名稱出現位置、120個互異商品 ID；繁中、簡中、日文與英文各自都沒有「同一商品 ID
對應多種譯名」的衝突。因此商品顯示身分固定為 `item.<id>.name`，不可再以商店節點、陣列
位置或原文字串作執行期身分。節點仍擁有商品供應清單，`item.json`／戰役資料仍擁有價格與
規則；語系資料只能提供名稱，不得改寫 ID、價格、秘密商店條件或庫存交易。

每個官方語系必須提供 `entities.json`，其 `items` 恰好涵蓋目前正式商店會出現的120個 ID。
產生器只接受四份現有 occurrence 翻譯對同 ID 達成一致；缺項、blocked、非 `entity_name`、
同 ID 譯名衝突或來源商品 ID 不存在時失敗即關閉。正式 runtime 查不到目前商品 ID 時停止
商店文字合成，不回退繁中或 `Good.Name`。

機器初稿若超出安全矩形，可在受版控的 `entity-overrides.json` 以語系＋商品 ID 提供人工
短名稱；override 必須保留完整語意，且只能指向清冊已有的 ID。2026-08-31 首批校正日文
item 95，以及英文 item 9／92／95／151／175／189；原因均為原稿冠詞、句號或冗長直譯
無法在最低可讀字級完整呈現，不是任意改寫商品規則。

原版商品列的名字安全矩形為每列 `(x+28,y+3,67,16)`，其中 `x=10+148*column`、
`y=119+26*line`，右側從 `x+95` 起保留能力圖示與數值。繁中保留原版 FDTXT 索引字形；
其餘語系先由既有 compositor 畫出不含名稱的圖示、屬性與價格，再於同一安全矩形使用官方
字型重畫名稱。字型須縮放到矩形內；若到規格下限仍放不下或缺字，整個畫面停止，不可靠
painter clipping、截斷或繁中殘影掩蓋錯誤。

2026-08-31 `DATA-READY／RUNTIME-E1` 收據：四份 `entities.json` 均由同一可重跑產生器
輸出120個 `item.<id>.name`，正式啟動、F4 原子切換與封包自檢會一併驗證。購買清單的
繁中路徑保持原版 FDTXT；簡中、日文與英文改走無名稱的原版商品列 compositor，再以目前
語系名稱填入原版安全矩形。全120個名稱已對三個翻譯語系執行實際字形、寬度與缺鍵測試；
缺名稱、缺字或超界皆拒絕合成。這一批只關閉購買清單名稱，不代表商店固定提示、角色名、
出售／轉移持有物清單或正常玩家截圖 E2 已完成。

### 10.7 商店固定提示四語契約（READY）

來源固定為 `fd2-reference-files.json` 登記的 `FDTXT.DAT`，並以完整本機包的唯讀
`FDTXT_000/resource.json` 回查原始 token：weapon／item greeting 分別為440／501；購買問句
439／502；金額不足438／504；無可裝備者437／505；裝備確認507；角色滿欄506；轉移目的、
空來源、來源提示510／511／512。這些 index 與 variant 選擇已由既有 native shop caller／
consumer 閉合，本批只把玩家可見字串正規化，不重開交易控制流。

官方 `pack.json` 為上述13個語意各保存獨立鍵與原始 `FDTXT_000/string_NNNN` provenance；
weapon 與 item 店員語氣不可合併。`%s` 商品／角色名與 `%d` 價格必須維持變數次序及型別。
非繁中畫面先重建同一店員頭像與空白原版對話框，再在 `(12,119)` 起、292像素寬、19像素
列距、最多三列的安全矩形繪製；不足金只可寫入 caller 已保留的第三列 `(12,157)`。
缺鍵、缺字、超過三列或變數不符皆停止目前 UI transaction，不回退繁中。

2026-08-31 `RUNTIME-E1`：非繁中的 greeting、購買問句、金額不足、無可裝備者、
裝備確認、轉移來源與目的提示已接正式 compositor；三個翻譯語系的固定提示及所有正式商店
商品＋價格問句均通過字形與三列矩形測試。506／511 的角色滿欄與空來源提示也已由
`native_identity` 名稱目錄填入。服務按鈕圖像文字及 E2 截圖仍未完成。

### 10.8 可玩角色四語名稱契約（READY）

可玩隊伍的顯示身分固定為 `native_identity` 0–31，原版名稱固定回查
`docs/data/exe_tables/characters.json`、`docs/data/portrait_names.json` 與 FDTXT 名稱索引
`native_identity+1`。這個目錄只描述隊伍／商店／戰鬥介面的角色顯示名稱，不可拿來合併
劇情中青年版頭像、偽裝身分或敵方同圖像角色。特別是 identity 9 只有「悠妮」一個角色，
不得因字形或機器翻譯另建「尤妮」。

2026-08-31 HUD 前置勘誤：直接回查 FDTXT_000 `string_0001..0032` 後，隊伍 UI 的
identity15／28／29 原版顯示字形分別是「塞可邦勒／達克塞／亞齊梅吉」，不是早期
`characters.json`／劇情索引使用的「賽可邦勒／達克賽／亞奇梅吉」。角色身分不變；
本目錄只修正 UI 顯示字形，歷史劇情名稱證據保留原樣並加此勘誤，不批次改寫來源筆記。

繁中名稱屬原版已證實資料；簡中名稱是繁簡正規化；日文與英文原版並不存在官方本地化，
因此重製版採人工審查的名稱轉寫，必須在資料中標示 `curated_remake_transliteration`，不可在
文件或介面宣稱為 1995 年官方譯名。機器內容中的「謝多→ありがとうございました／Thank you」
及「巴拿羅西亞→Panama」明確排除。查不到角色、名稱為空、來源缺失或狀態不在允許清單時
失敗即關閉，不回退成錯誤機器稿。

商店 FDTXT506／511 的 `%s` 必須使用同一 `native_identity` 名稱目錄；角色名只替換文字，
原版店員頭像、開收框、等待模式、物品轉移與滿欄判定仍由既有 typed transaction 擁有。
非繁中嘴型同步不列入第一版驗收；閉嘴頭像、逐字文字、鏡頭與開收框仍須保留。

2026-08-31 `DATA-READY／RUNTIME-E1` 收據：四份官方 `entities.json` 均含完整 32 筆
角色 ID、名稱、FDTXT／native identity provenance 與名稱狀態；產生器拒絕缺號、空名稱或
未知狀態。非繁中購買滿欄與物品轉移空來源／滿目的地已改由 `CharacterName` 填入既有
對話框；全 32 名角色、三個翻譯語系與兩種動態訊息均通過實際字形及三列安全矩形測試。
繁中仍走原版 indexed compositor。正常玩家截圖 E2 尚未由本批關閉。

角色名冊沿用已閉合 `0x2ea90` 的兩欄六列配置：圖示位於 `(14+132*c,117+26*r)`，
名稱原點為 `(40+132*c,121+26*r)`。多語名稱安全矩形固定為 `100×16`，不侵入下一欄
圖示或下列；出售、物品轉移、裝備、消耗品購買對象與教會共用名冊只清空這個矩形，
再以 `native_identity` 目錄重畫。
三列裝備比較名冊的姓名右側從 `x=122` 開始放 AP／DP 比較，因此姓名矩形另固定為
`(40,121+26*r,80,16)`；只清空姓名，四組 current→candidate 數字、箭頭、顏色與裝備
試算仍由 `0x2efb7` typed compositor 擁有。三個翻譯語系的32名角色都必須同時通過
兩欄 `100×16` 與裝備比較 `80×16` 兩種矩形。
出售／轉移持有物重用商品清單既有 `67×16` 名稱矩形，價格模式仍由 caller 保持原版
四分之三價，語系層不得改價；缺名稱、缺字或超界時整個 frame 拒絕發布。

2026-08-31 `RUNTIME-E1` 收據：非繁中的三列裝備比較名冊已改由無姓名原版 compositor
保留人物圖示、AP／DP／HIT／EV current→candidate、箭頭及比較色，再以角色目錄重畫姓名。
裝備矩形使用獨立0.55字級；賽可邦勒與聖寇拉斯的日文長名亦完整落在80×16內，沒有截字或
改名。原版合成器回歸與三語全32名矩形測試通過；正常玩家截圖仍待四語 E2 抽樣。

戰鬥面板姓名不是只有 0–31 隊員。固定雜湊的 FDTXT_000 `string_0001..0139`
對應 raw unit record `+8` 的 selector 0–138；其中94筆有名稱，32–67、75、128–132及
136–138共45筆為已證實空字串。`BattleName(raw_record_byte_8)` 因此是獨立於
`CharacterName(native_identity)` 的資料契約；即使0–31目前對齊，也禁止用角色身分替代
raw selector。四語目錄必須恰含94筆，空字串 selector、缺譯、缺字或超過姓名矩形皆失敗即
關閉，不回退成繁中或臆造敵人名稱。

2026-08-31 `DATA-READY／RUNTIME-E1` 收據：雜湊綁定匯出器已保存94筆原文與45筆空洞，
非隊伍的62筆敵人／村民名稱另有簡中確定性轉換及日／英人工短譯。指令0／1／2／3／5／6／
7／8與敵方9共27個呼叫，以及指令24／29／32–35共18個呼叫，已統一由同一多語 owner
消費 raw `+8`。非繁中只省略原版姓名，保留 `0x18C6D` 的149×42框、HP／MP、等級、
raw `+6`上下定位及chapter24 unit17例外，再於 `(panel.x+5,panel.y+4,120,16)` 重畫姓名；
繁中仍走原版 FDTXT compositor。三個翻譯語系全部94筆均通過實際字形及安全矩形測試。
一般玩家四語戰鬥截圖仍列 E2，不以本批單元測試取代。

原生法術／戰技選單採 raw command ID 0–35，不採舊 `Spell.Name` 作名稱身分。原版
`0x1ceed→0x15f84` 以 `FDTXT_000 string_(441+commandID)` 畫標籤；ID31是已證實空槽，
因此官方語系目錄必須恰有35筆且查詢31失敗。來源解碼的ID21「社麻術」與ID27「麻庫術」
仍原樣保存在 `command_labels.json`；正式繁中顯示依已接效果與既有術名作受控字形校正為
「祛麻術／麻痺術」，標為 `curated_source_glyph_correction`，不可回寫或隱藏原始證據。

2026-08-31 `DATA-READY／RUNTIME-E1` 收據：四語 `CommandName(raw_command_id)` 已接一般玩家
四列原生指令格、`SpellBook`相容顯示及F4原子切換。標籤沿用原版每欄100像素、MP數字起點
`x+0x49`的幾何，在140個邏輯像素內以0.02級距縮放且保留2像素餘量；35筆四語名稱均通過
實際包內字型量測。ID31、缺譯或小於0.6倍率仍失敗即關閉。這不改 command availability、
MP gate、target/effect或演出 owner，也不把舊直列 `drawSpellMenu` 提升為原版玩家路徑。

2026-08-31 服務按鈕勘誤：三種商店資源的entries3–10均為24×20成對 raster圖示，實際畫面
使用購買／出售／裝備／離開圖像，沒有需翻譯的嵌入文字，故不再列為四語缺項。教會同尺寸
entries3–10在目前 indexed 畫面呈現橙藍亂碼紋理；這是原版 raster codec／palette或素材
消費端的介面缺陷，不能用猜測文字遮罩冒充翻譯。修正前須先取得原版同狀態畫面或直接
codec反證，再依RE→規格→實作處理。

同批另將七個高頻戰鬥提示加入官方語意 pack：MP不足、指令不可用、攻擊選擇目標、原始
指令不可用、封咒、無法術及物品欄位選擇。四語 pack均以穩定key、原始string inventory
provenance及零變數簽章驗證；正式玩家輸入的九個寫入點已改查目錄，缺鍵時不回退繁中。

下一批法術操作以七個語意鍵接通選單標題、目標提示、麻痺阻擋、MP／封咒阻擋、傷害
結算、回復結算與未命中尾碼。結果訊息由玩家與敵方AI共用格式化owner；「造成／回復」
不再以繁中動詞變數傳入其他語言。戰場寶物與擊破獎勵另以六個鍵涵蓋金幣、物品及滿欄
分支。此批只證明正式consumer與四語資料契約的RUNTIME-E1，不等於四語戰場畫面已取得
正常玩家E2。

教會正式輸入另接通「無物品可轉交」、物品轉交成功與復活成功三個四語鍵。轉交與復活
都在 mutation 前預先格式化必要訊息；官方目錄缺失時不先修改背包、金幣或角色狀態。
29筆ClassID職業名稱已加入官方實體目錄；26／27／28以`original_placeholder`保留原版
問號／全形空白，不升格成正常職業。轉職成功、確認標題、空名冊、目標與是／否後備介面
均查四語目錄；正式轉職在 mutation 前預取角色名、職業名與成功訊息，缺任一資料不修改
成長值、裝備或持續隊伍。

完整原版物品名稱區固定為 FDTXT_000 `string_0181..0395`，對應 raw item ID 0–214。
雜湊綁定匯出確認其中 200 筆有名稱，108–122 共15筆為原版空字串；空洞必須標成
`confirmed_empty_item_ids`，不可翻譯或造名，也不可在正常背包中顯示。正式背包目錄的
驗收數因此是200，不是256；若存檔或 runtime 把空洞／範圍外 ID 當成可顯示物品，應視為
資料錯誤並失敗即關閉。

完整清冊與既有商店資料只有三筆字形差異，產生器採精確 allowlist：item93 由
「白金勳章」校正為 FDTXT「白金徽章」；item139 由常用字形「神秘裝」校正為 FDTXT
「神祕裝」；item94 的 metadata 解碼為語義不成立的「生命之貫」，保留既有「生命之實」
並標成受控人工字形勘誤。除這三組原文對以外，任何商店名稱與 FDTXT 清冊不一致仍拒絕產生。

2026-08-31 `DATA-READY／RUNTIME-E1` 收據：四份官方 `entities.json` 已由120筆商店商品
擴充為200筆完整可顯示物品；80筆非商店缺項以繁中原文、確定性簡中及人工審查的日／英短譯
補齊，item125／126 保留原始字面值 `11`／`22`，不假造語意。四語嚴格 loader 拒絕
108–122 confirmed-empty 與範圍外 ID；全部200筆在三個翻譯語系均通過字形與 `67×16`
安全矩形測試。出售與轉移持有物正式路徑現使用無名稱原版 compositor，再依物品 ID 重畫
名稱；圖示、能力、四分之三價格、選取與交易規則不變。

### 四之二、legacy 匯入與 canonical 往返

> 狀態：**DATA-READY（第一個實檔切片）**（2026-08-29）

2026-08-30 的全戰役批次契約使用 `tools/export_editor_canonical.py`，固定以
`campaign_full.json`、30 份 scenario 與 35 份 story 建立獨立 canonical bundle，
不得覆寫 legacy 目錄。若本機存在已驗證的私人動畫 metadata，可另顯式加入 297 份
animation 文件；乾淨 clone 的受版控測試不得把被 `.gitignore` 排除的私人產物當成
必要輸入。每份輸出與 bundle summary 都採 deterministic writer，並保存來源、診斷、
document ID 與 SHA-256；兩次獨立輸出必須逐位元相同。這先關閉全戰役資料化與往返，
跨文件 identity validator 及正式 runtime consumer 仍是下一個 gate。

`tools/import_editor_legacy.py` 現可將 legacy campaign、scenario、story 與既有動畫
metadata 投影成四份 canonical 文件。穩定 ID 只由來源路徑、legacy key、原始數值與
索引建立，不依賴顯示文字；未映射欄位完整保存在 `extensions.legacy`，並產生可機讀
診斷。scenario 的 legacy `do[]` 已明確轉成 canonical `actions[]`；具非負
`native_identity` 的隊員與台詞 speaker 共用 `character/native-N`，避免同一角色因
台詞位置不同而被拆成多個身份。

writer 採固定 UTF-8、排序 key 與縮排，load→write→reload 不改動來源、擴充欄位、
穩定 ID 或戰役轉場。實際 `campaign_full.json`、ch01 scenario、ch01 story 與 AFM
metadata 均已進 Docker 往返測試，並與既有跨文件 validator 一起通過。這一層仍是
匯入／寫回基礎，不等於完整角色 identity catalog、全 35 章 canonical 產物、編輯器
圖形介面或 runtime 已改讀 canonical 文件。

## 五、往返與相容性

### FDOTHER #13 讀檔欄位固定資產

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_load_slots_ui_ida.txt`](../data/fd2_load_slots_ui_ida.txt)。

`ui/fdother_013_load_slots/resource.json` 是讀檔四槽底框的 canonical 資產文件。
它只包含已證實由 `0x30437` 消費的 entry 16：`opaque_high_run`、310×86、
`entry_016/frame.png`。文件必須綁定 `FDOTHER.DAT` 的固定雜湊及 resource 13 的
53,210-byte raw size；loader 只接受單一 entry、正確 codec／幾何與 indexed PNG。
正式遊戲不得回退到 `.DAT`，任何不完整或不一致均失敗即關閉。這項契約只關閉
圖像來源，不把 native save restore、刪除或覆寫提升為完成。

現行 `fd2-asset-import`、`fdother.LoadSeparatedLoadSlotsFrame` 與正式
`loadNativeLoadSlotsUIAssets` 已符合此契約；原始 entry16／分離 PNG 逐位元組測試、
缺包拒絕及 LOAD 畫面聚焦回歸均通過。production 的存檔欄位底框不再讀
`FDOTHER.DAT`。

### FDOTHER #14 教會／轉職固定資產

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_church_ui_assets_ida.txt`](../data/fd2_church_ui_assets_ida.txt)。

`ui/fdother_014_church/resource.json` 固定保存教會／轉職正式玩家路徑已證實的
21筆 mixed-codec entry：four-mode frames 0與23..31、opaque entries 1／3..10／16，
以及 raw entry15。每筆保留原始 index、codec、幾何與標準 indexed PNG；four-mode
frame另需 binary mask。metadata 綁定固定 `FDOTHER.DAT` hash、resource14及
51,157-byte raw size。loader 必須要求完整且無重複的精確集合，不接受未證實 entry，
也不得回退 archive。

此切片只關閉 resource14。`native_class_ui` 同時使用的 FDOTHER #5 dialogue／digits
與 #2 choices 仍是後續獨立契約；在它們遷移前，不宣稱整個 loader 已與 archive
斷開。

現行 importer 與 `fdother.LoadSeparatedChurchUIAssets` 已符合契約；21筆原始／
分離 pixels／masks、缺包拒絕、底層 compositor，以及正式轉職與四項教會服務
聚焦回歸均通過。production 的 `native_class_ui` 已無 resource14 archive caller。

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

### 12.1 現代美術主題目錄（2026-08-31）

現代美術的本機工作稿位於被版控忽略的
`remake/generated-assets/modern-theme-prototypes/`；通過清冊與執行期驗證後，正式
私人保存位置是 `https://github.com/wicanr2/fd2-assets-private.git` 的
`packs/fd2-modern-handpainted-a/`。公開儲存庫保存
`remake/assets/themes/modern/catalog.json` 的可追溯目錄，以及使用者允許公開的
總攬圖。每筆必須有
穩定 `asset_id`、用途 `role`、成熟度 `status`、原版／重製參考 `source_refs`、
私有相對路徑、寬高與 SHA-256。目錄不得把概念合成圖冒充可直接替換的 portrait、sprite
sheet、tileset 或 HUD 九宮格，也不得宣稱能從合成圖可靠反切原始 frame／tile／cell。
正式新素材必須逐一對照原始獨立資產的ID、方向、遮罩與幾何重繪；只有完成透明邊界、
縮放與 runtime 合成驗收後，狀態
才能由 `concept` 提升為 `runtime_ready`。

`tools/validate_modern_theme_catalog.py` 有兩種閘門：公開庫模式驗證 schema、角色集合、
路徑限制、雜湊格式，以及受版控來源檔或 `asset:` 穩定來源識別；本機完整版加
`--verify-private`，逐檔核對私有 PNG
尺寸與 SHA-256。私有檔缺失不得影響忠實主題啟動，但現代主題必須失敗即關閉，不得悄悄
混用原版素材並宣稱現代主題完整。

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

本段最初只關閉`loadNativeUIPalette`與78格command grid owner，**不代表當時所有戰鬥
演出都已停止回讀FDOTHER #0**。2026-08-29 production caller稽核曾找到command 0／1／
2／3／5／6／7／8／9／24／29／32／33／34／35、native map bundle、LOAD slots與
class／church共18條直接`ReadResource(..., 0)`；下列同日遷移已取代該歷史現況。

#### 共用 #2 action cells 與 #5 dialogue 補全契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_shared_ui_banks_evidence.txt`](../data/fd2_shared_ui_banks_evidence.txt)。

既有 `ui/action_cells/cell_000.png` 至 `cell_077.png` 必須新增
`ui/action_cells/resource.json`，以固定 provenance、完整0..77集合、每格幾何及
`raw_indexed_transparent` codec 使 raw-cell runtime 可嚴格載入。既有
`ui/FDOTHER_005/item_panel` 則在不更換穩定 identity、不複製 PNG 的前提下，補入
dialogue raw entries0／18／19；如此同一 separated loader 能提供教會／商店所需的
0..19 dialogue與31..40 digits。兩包任一 metadata、indexed PNG、幾何或完整集合錯誤
均失敗即關閉，不回退 archive。

現行 importer 已輸出 #2 metadata 並補齊 #5 entries；兩個 strict loader、原始逐
位元組比較、缺包拒絕與正式指令格／轉職／教會／商店聚焦回歸均通過。
`native_class_ui` 不再取得 FDOTHER archive path。

#### 整備 #1 range overlay 與 #5 panel 補全契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_preparation_assets_evidence.txt`](../data/fd2_preparation_assets_evidence.txt)。

`sprites/fdother_001_range_overlay/bank.json` 保存 FDOTHER #1 的20個24×24 descriptors，
每筆使用 indexed frame、binary source mask與binary remap mask；穩定 identity是
`sprites/FDOTHER_001/range_overlay`，同時供整備游標與戰場 range overlay使用。
既有 `ui/FDOTHER_005/item_panel` 則加入 opaque entries20／21與 four-mode entry137，
並直接共用已分離的兩組數字31..40／42..51。正式整備 loader 只接受這兩包與完整
FDICON separated bank，任一不完整即失敗即關閉，不得回退 archive。

實作結果：importer 已輸出20筆 range overlay 的 indexed frame／source
mask／remap mask 與固定 provenance；`ui/FDOTHER_005/item_panel` 亦補齊
entries20／21／137。正式 `loadNativePreparationUIAssets` 現在只讀取 separated
pack，不再取得 `FDOTHER.DAT` 路徑。逐 indexed pixel／mask 原版對照、
缺包拒絕、`internal/fdother` 回歸及有界 Xvfb 整備界面聚焦測試均通過。

#### Event61 FDOTHER #45 幀表演出契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；canonical IDA 證據見
> [`fd2_event61_fdother45_ida.txt`](../data/ida/fd2_event61_fdother45_ida.txt)。

`animations/fdother_045_event61/bank.json` 使用穩定 identity
`animation/FDOTHER_045/event61`，將原版 resource45 的59筆 frame-table descriptor
輸出為逐幀 indexed PNG 與 binary source mask。metadata 保留原始 index、
X／Y、width／height，並固定 FDOTHER 版本、resource45、raw size 2073與
raw SHA-256。正式 event61 presentation 與全軍移動 preflight 必須共用
同一 strict loader；缺檔、缺幀、重複索引、非 indexed PNG、非 binary mask、
幾何或 provenance 不符即失敗即關閉，不回退 `FDOTHER.DAT`。

驗收必須將 separated loader 與固定原版 `DecodeResource(...,45)` 進行
59幀逐筆 X／Y／尺寸／indexed pixels／mask 對照，並證明兩個 production
consumer 的 source 不再取得 archive path。本契約不改 event61 的觸發、
道具、JOIN、文字與時序語意，也不自動提升 PLAYER-E2。

實作結果：importer 已輸出59組 indexed PNG／binary mask與 strict
metadata；`LoadSeparatedEvent61Frames` 與固定 archive 的59幀逐筆對照一致。
event61 演出 owner 與全軍移動 preflight 現只消費分離 bank，resource45
的 production archive caller 歸零；缺包拒絕、`internal/fdother` 回歸與
有界 Xvfb 正式 event61／group-march 聚焦測試均通過。

#### Chapter-23 staging FDOTHER #42 單幀契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_ch23_stage_assets_evidence.txt`](../data/fd2_ch23_stage_assets_evidence.txt)。

`surfaces/FDOTHER_042/resource.json` 沿用標準 `indexed_surface` 契約與
`surface/FDOTHER_042` 穩定 identity；`frame.png`與`mask.png`固定為
312×192 indexed pixels／binary source mask。source 必須綁定固定 FDOTHER
版本、resource42與raw size 59412。`LoadSeparatedNativeCh23Stage` 只接受
這份完整契約；任一 metadata、codec、幾何、PNG類型或mask值不符
即失敗即關閉，不回退 `FDOTHER.DAT`。

正式 `startNativeCh23Loop` 與 `prepareNativeCh22Aux` 必須共用該 loader。
驗收包含原版／分離 geometry／indexed pixels／mask 對照、缺包原子
拒絕，以及 ch23 post／ch22 auxiliary reload 兩條正式路徑聚焦回歸。
這不重解 loop／DAC／BIOS tick，也不自動提升 PLAYER-E2。

實作結果：importer 已輸出312×192 indexed PNG／binary mask與帶raw
entry 雜湊的metadata；strict loader 與固定原版的geometry／indexed
pixels／mask對照一致。ch23 post 與 ch22 auxiliary reload 現只消費此分離
surface；缺包原子拒絕與兩條有界 Xvfb 聚焦回歸均通過。

#### 戰場資訊 FDOTHER #5 entries `0x85..0x88` 契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；原版 caller、索引、幾何與
> 座標主證據見
> [`fd2_nested_system_menu_ida.txt`](../data/ida/fd2_nested_system_menu_ida.txt)。

既有 `ui/FDOTHER_005/item_panel` mixed-codec bank 必須追加原始 entries
`133..136`，保留穩定原始 index及 LMI1 解碼後的 indexed PNG。四筆固定幾何依序
為 `102x17`、`170x117`、`170x16`、`63x15`；不替它們猜測高階美術名稱。

`LoadSeparatedSystemInfoPanels` 必須以單一交易載入既有 strict #5 bank，確認四筆
完整存在且幾何符合 `0x1B41D`；任一 metadata、entry、PNG或幾何不符即失敗即關閉。
正式 `loadNativeSystemInfoAssets` 只能組合這四筆與已分離的數字、FDTXT及字型，
不得再以 `nativeFDOTHERPath()` 檢查或讀取原始 archive。驗收包含四筆與固定原版
`DecodeLMI1Resource(...,5)` 的逐 indexed pixel oracle、缺包拒絕及正式 loader 在
沒有 `FD2_ORIGINAL_FDOTHER` 時成功；不重解 BIOS／DAC 時序或提升 `PLAYER-E2`。

實作結果：共用 #5 bank 已追加 entries `133..136`，四筆分離 PNG 與固定原版
逐 indexed pixel 一致；`LoadSeparatedSystemInfoPanels` 會一次驗證完整 bank 與
四筆精確幾何。正式 `loadNativeSystemInfoAssets` 已移除 `nativeFDOTHERPath()`，
在清空 `FD2_ORIGINAL_FDOTHER` 時仍通過 loader、12 段展開／收合及巢狀選單
聚焦回歸；缺包仍由共用 strict loader 拒絕。

#### 共用物品／教會／戰場短狀態面板 FDOTHER #5 契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_battle_status_panel_ida.txt`](../data/ida/fd2_battle_status_panel_ida.txt)
> 與
> [`fd2_status_panel_transient_indicators_ida.txt`](../data/ida/fd2_status_panel_transient_indicators_ida.txt)。

既有 `ui/FDOTHER_005/item_panel` 已包含 `0x17EEF／0x17FC0` 所需的 raw、
opaque與four-mode entries。正式 `RenderNativeItemPanelBaseResources` 必須從
strict separated bank取得 raw `1..17`與opaque `20/21`，並從分離 portrait pack
取得 raw record `+7`所選 DATO frame0；`RenderNativeItemPanelResources` 不再接受
archive path。物品初次開啟、成功交易後重建及教會狀態三個 caller必須共用此路徑。

`LoadNativeBattlePanelValueAssets` 同樣改從分離 bank取得opaque `22`、raw
`23..30`及four-mode `31..52/93`，不得為了只畫數值而回讀整個 resource #5。
兩個 archive adapter保留明確 `Archive` 後綴，只供固定原版逐像素 oracle。
缺 bank／entry／portrait／codec／幾何時，在私人320×200 buffer發布前拒絕；
驗收須證明分離與原版完整 base／data panel逐 indexed pixel相等，且四個正式
caller不再出現 `nativeFDOTHERPath()`或 archive loader。

實作結果：`RenderNativeItemPanelBaseResources` 與
`LoadNativeBattlePanelValueAssets` 現只消費共用 strict #5 bank；原始版本分別
保留 `Archive` 後綴供 oracle。物品初次開啟、成功交易後重建、教會狀態與
戰場短狀態欄四個 production caller均移除 `nativeFDOTHERPath()`。完整 base＋data
panel與短狀態欄的分離／固定原版逐 indexed pixel對照通過；缺包原子拒絕、
chapter1正常物品面板及教會狀態生命週期在清空三個原版 archive環境變數時通過。

#### 戰鬥共用 FDOTHER #0 DAC owner 遷移契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

這18條正式路徑不得各自取得archive path再讀同一768-byte資源；統一經
`loadNativeBattlePalette`載入既有`palette/fdother_000.json`，同時回傳原始六位元DAC
與256色顯示palette。loader沿用已閉合的固定FDOTHER identity、resource 0、raw size 768、
component範圍0..63及`asset_id=palette/fdother_000`驗證；任一條件不符即失敗，不回退
`FDOTHER.DAT`。

本切片只遷移palette owner，不刪除同一command因FDOTHER內嵌動畫、巢狀音效、HUD／panel
或LUT仍需的archive path。驗收為：上述production檔案對`ReadResource(*,0)`歸零；共用
helper與固定archive #0逐byte／逐RGBA一致；至少抽測一條玩家command、一條AI command、
compound command及native map bundle；令`FD2_ORIGINAL_FDOTHER`指向缺檔時，只能證明
palette helper本身仍工作，不能冒稱整條尚有其他archive資源的演出已完全分離。

實作已移除`separated_ui_assets.go`內原本只驗`schema_version`／`asset_id`／長度的重複
寬鬆JSON解析，所有owner改共用`fdother.LoadSeparatedFDOTHERPalette`。正式
`remake/cmd/fd2`對`ReadResource(..., 0)`的呼叫現為0；分離DAC與固定archive #0的768
bytes及256色RGBA逐項相同。無archive測試、損壞DAC拒絕，以及玩家command 0、AI
command 9、native map、class／church代表路徑皆通過。command 34既有end-to-end fixture
在進入本切片helper前即因HUD drawable selector狀態不足失敗，故不拿它冒充本切片失敗；
複合命令正式碼已編譯並共用同一精確helper，但完整舊fixture仍是獨立測試債。

#### 第一批戰鬥巢狀音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片只使用[`36`](36-sfx-audio-data.md)已證實的FDOTHER outer／subresource及caller時序，
涵蓋resource 82／83／84／85／86／87／88／90的22筆非空sample。每個resource輸出：

```text
sfx/FDOTHER_NNN/
  resource.json
  sample_MMM.ogg
```

`resource.json`固定保存`schema_version=2`、`kind=fd2_pcm_sound_bank`、穩定`asset_id`、
固定FDOTHER來源identity、outer resource、原始container count、zero-length tail index、
`sample_rate=11025`、`channels=1`、`sample_format=unsigned_u8`、
`timing_evidence=hardware-spec_approximation`，以及每筆sample的subresource、raw byte count、
`source_pcm_sha256`、OGG相對路徑與cue evidence。cue evidence只允許raw caller／typed schedule
證實的角色，不可把人耳猜測名稱升格成資料identity。

離線匯入器先驗證固定FDOTHER大小與雜湊，再驗證兩層`LLLLLL`directory、所有offset、非空
sample數及空尾項；以可重現、鎖版Docker內編碼器將raw unsigned-8 PCM轉成mono OGG。OGG是
runtime格式，raw PCM只作容器內oracle，不寫入正式pack。因Vorbis有損，驗收分兩層：raw
provenance以原始bytes的長度／SHA-256閉合；OGG則驗證可解碼、mono、宣告取樣率、非靜音、
duration與`sample_count/11025`在一個output sample內一致，不聲稱逐sample parity。

嚴格runtime loader只能由`FD2_ASSET_PACK/sfx`依`asset_id`載入metadata與OGG；拒絕錯來源、
錯resource／subresource、重複或缺sample、非空尾項、路徑逃逸、錯channels／rate、損壞OGG
及metadata漂移，不回退`FDOTHER.DAT`或`remake/assets/sfx/*.wav`。command 0／1／2／3／4／
5／6／7／8及敵方9的admission gate與播放bytes須改用同一已預檢bank，避免「archive只負責
存在性、舊WAV負責播放」的雙來源。

驗收至少包含：八個bank的22筆raw hash重生；OGG全數probe；缺archive時command 0玩家路徑、
command 6多目標及敵方command 9仍可建立presentation；缺pack或任一必需sample時在MP／HP／
狀態交易前失敗即關閉。這批當時只關閉第一批command sound banks，尚未外推UI #31、一般物理攻擊
動態bank、resource 80／91..95或完整音訊E2。

實作結果：`tools/export_sfx.py --separated-pack`現先驗證固定FDOTHER identity及104筆outer
directory，再嚴格驗證八個巢狀bank、22筆非空sample與各自空尾項；Docker內固定使用
`vorbis-tools 1.4.3`的`oggenc`產生OGG，並由`ogginfo`／`oggdec`逐筆驗證。相同輸入連續
兩次輸出的30筆檔案逐byte一致。`internal/fdother.LoadSeparatedSoundBank`再驗證metadata、
來源雜湊、resource形狀、PCM hash格式、安全相對路徑及OGG magic；正式Ebiten adapter
必須成功解碼全部八個bank後才發布。

command 0／1／2／3／4／5／6／7／8及敵方command 9已移除所有
`ReadNestedResource`入場檢查，播放bytes與admission gate統一來自分離OGG。舊
`battle_82/83/84/85/86/87/88/90_*.wav`正式引用歸零，#88的既有death／ch24 transition
別名也改指同一已解碼bank。原始FDOTHER路徑刻意不存在時，玩家command 0、command 6、
敵方command 9及其餘command 1..8代表演出測試均通過；缺bank／sample及不支援resource
則失敗即關閉。清冊當時因此增加22筆OGG與8筆metadata，為38,709筆：37,685 exported、
1,005 intentionally raw、19 blocked。

#### UI 共用 FDOTHER #31 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片沿用前節相同`fd2_pcm_sound_bank`契約，新增`asset_id=sfx/FDOTHER_031`。固定
resource大小31,771 bytes、container count 14、zero-length tail index 13；只輸出#0..#12
共13筆OGG。每筆metadata保存[`36`](36-sfx-audio-data.md)列出的raw長度與SHA-256，取樣格式
仍為unsigned-u8 mono／11025 Hz hardware-spec approximation。匯出器必須和第一批command
bank共用相同來源雜湊、原子輸出、可重現serial與OGG probe，不可另留較寬鬆的legacy模式
作正式pack來源。

正式`loadSFX`改由`FD2_ASSET_PACK/sfx/FDOTHER_031`一次嚴格載入並解碼#0..#12；缺metadata、
缺任一sample、壞OGG或來源漂移時整批拒絕，不回退`remake/assets/sfx/sfx_*.wav`。既有
`playSFX(index)`及所有typed schedule可繼續消費同一map，但正式碼不得再讀舊WAV。驗收至少
涵蓋：13筆raw hash重生、兩次OGG輸出一致、全部OGG probe、原始FDOTHER及舊WAV不可讀時
標題cursor／confirm、AI idle recovery、AI mode 5與command heal代表路徑仍能預檢；缺
index 4／11／12時在對應狀態或交易之前失敗即關閉。這不外推物理攻擊動態bank、#80、
#91..95或完整人耳E2。

實作結果：canonical匯出現在一次產生#31及既有八個command bank，共9份metadata／35筆
OGG；兩次完整輸出逐byte一致，#31的13筆OGG均通過probe。`loadSFX`已改為嚴格載入並解碼
`FDOTHER_031`，舊`assets/sfx/sfx_*.wav`正式碼與測試引用歸零；OGG loader明確建立播放
context，沒有遺失舊WAV loader原本隱含的初始化責任。缺原始FDOTHER時，真實播放器、標題
游標／確認、ch28 post及AI mode 5代表路徑通過；缺pack則整個#31 bank拒絕。清冊增加13筆
OGG與1份metadata，當時為38,723筆：37,699 exported、1,005 intentionally raw、19 blocked。

#### 指令共用 FDOTHER #80 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片沿用`fd2_pcm_sound_bank`，新增`asset_id=sfx/FDOTHER_080`。固定resource大小
116,165 bytes、container count 17、zero-length tail index 16；只輸出#0..#15共16筆
OGG。每筆metadata必須保存[`36`](36-sfx-audio-data.md)的raw長度與SHA-256，sub3／sub4
即使bytes相同也保留各自selector。高階名稱保持未知，11025 Hz仍是hardware-spec
approximation。

正式runtime須在入場時一次完整載入#80 bank，所有`battle_80_*.wav` consumer改由同一
嚴格map取得selector；缺metadata、缺任一sample、壞OGG、來源漂移或selector越界時整批
拒絕，不回退舊WAV或原版archive。驗收至少涵蓋command 9、commands 10–12、AI item、
玩家／敵方modifier與commands 32–35代表演出，並在原版`FDOTHER.DAT`不可讀時通過；缺
pack必須在任何MP／HP／狀態交易前失敗即關閉。本切片不包含#91..95，也不猜一般物理
攻擊動態bank的`index2`對照。

實作結果：canonical匯出現為10份metadata／51筆OGG；兩次完整輸出逐byte一致，#80的
16筆OGG全數通過probe。正式整批loader在`loadGame`入場時驗證並解碼完整#80，再安裝
selector 0／2／13／14／15具名欄位及供AI item、modifier、commands 32–35使用的raw map；
所有`battle_80_*.wav`正式碼與測試引用歸零。完整`internal/fdother`／`cmd/fd2`回歸通過，
缺pack測試維持失敗即關閉，且刻意移除selector的原子性測試不會被演出途中重載補回。
清冊增加16筆OGG與1份metadata，現為38,740筆：37,716 exported、1,005 intentionally raw、
19 blocked。

#### 指令 32–35 與增援 FDOTHER #91..95 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片沿用`fd2_pcm_sound_bank`並保留原始selector，不重新編號。#91/#94各有三筆
非空PCM與空尾3，#92/#93各有兩筆非空PCM與空尾2，#95有一筆非空PCM與空尾1。
command32正式消費#91/1、/2；command33消費#92/1；command34消費#93/1；command35
消費#94/1、/2；`0x32999` pass1消費#95/0。#91..94的selector0雖具PCM位元分布且位於
sound resource，但尚無已證實caller；仍輸出標準OGG以完成素材分離，metadata必須標
`strong_inference`及`no_confirmed_consumer`，正式runtime不得自行播放。
因本批新增每筆必填`classification_evidence`，sound-bank metadata由版本1升為版本2；
同次重生全部15個bank，舊版本1 pack明確拒絕，不以同一版本號靜默改變契約。

正式loader一次完整驗證並解碼五個bank；四條command owner須在建立交易plan前預檢
各自必需selector，增援owner由同一分離bank安裝#95/0。缺件、壞OGG、來源漂移或原始
selector不符時失敗即關閉，不回退`battle_91..95_*.wav`或原版archive。驗收包含兩次
完整匯出逐檔一致、11筆OGG解碼／聲道／取樣率／時長、原始PCM hash、無archive正式
載入、四條command及增援代表回歸，以及舊WAV引用歸零。

實作結果：canonical匯出現為15份metadata／62筆OGG；兩次完整輸出逐檔一致。
#91..95的11筆OGG均通過解碼與完整素材清冊驗證；四筆無caller的selector0保留
`strong_inference`，沒有接入播放。commands32–35改由分離bank取得#91..94已證實
selector，`sfxSpawnIntro`改由#95/0安裝；所有`battle_91..95_*.wav`正式引用歸零。
Python契約／清冊測試、完整`internal/fdother`及`cmd/fd2`分離bank聚焦回歸通過。清冊增加11筆OGG及
5份metadata，現為38,756筆：37,732 exported、1,005 intentionally raw、19 blocked。

#### command24 固定 FDOTHER #53 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片只接[`36`](36-sfx-audio-data.md)已證實的FIGANI resource98 header6→FDOTHER #53
與actor selector3／target selector2。#53的四筆非空PCM全部保留原selector轉OGG，空尾4
不輸出；selector0／1只有PCM形狀與bank歸屬，標`strong_inference`且不接入播放。

正式整批loader新增#53，`sfxCommand24Actor`／`sfxCommand24Target`只由分離bank安裝；
command24在建立交易plan前預檢2／3。缺bank、缺selector、壞OGG或來源漂移時零MP／HP
交易，不回退`battle_53_02.wav`／`battle_53_03.wav`。驗收為兩次完整匯出一致、四筆
OGG probe、原始PCM hash、無archive載入、command24 marker聚焦回歸與兩筆舊WAV引用歸零。
本切片不建立物理攻擊`index2`對照，也不宣稱selector0／1高階語意。

實作結果：canonical匯出現為16份metadata／66筆OGG；兩次完整輸出逐檔一致。#53
四筆OGG通過probe與素材清冊驗證，0／1保留強推論且未接播放。正式loader安裝3／2，
command24在建立交易plan前依typed schedule預檢；`battle_53_02.wav`／`_03.wav`正式引用
歸零。9項Python測試、`internal/fdother`及command24／分離bank聚焦Go回歸通過。清冊
增加4筆OGG與1份metadata，現為38,761筆：37,737 exported、1,005 intentionally raw、
19 blocked。

#### command29 固定 FDOTHER #50 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）

本切片只接已證實的一般學招selector34→FIGANI resource104→FDOTHER #50、actor sample1
與target sample4。#50五筆非空PCM全部保留原selector轉OGG，空尾5不輸出；0／2／3明標
`strong_inference`且不接播放。

正式整批loader新增#50；command29由typed schedule在建立交易plan前預檢1／4，播放
資料只取分離bank。缺件時MP／HP／acted保持不變，不回退動態`battle_50_*.wav`。驗收
包含兩次完整匯出一致、五筆OGG probe、原始PCM hash、無archive載入、command29聚焦
回歸及正式`fmt.Sprintf("battle_%02d_%02d.wav")`引用歸零。本切片不猜0／2／3用途或
一般物理攻擊`index2`。

實作結果：canonical匯出現為17份metadata／71筆OGG；兩次完整輸出逐檔一致。#50五筆
OGG通過probe與清冊驗證，0／2／3維持強推論且未接播放。command29正式owner依typed
schedule預檢並消費1／4，動態舊WAV路徑已移除。9項Python、`internal/fdother`及
command29／分離bank聚焦Go回歸通過。清冊增加5筆OGG及1份metadata，現為38,767筆：
37,743 exported、1,005 intentionally raw、19 blocked。

### FDMUS FM／MT-32 OGG 與音源 catalog 契約

> 規格狀態：**CONFORMED**
> 主證據：[`fd2_music_render_catalog_evidence.md`](../data/fd2_music_render_catalog_evidence.md)。

編輯器與 runtime 的穩定入口為 `assets/music_catalog.json`。schema version 1 固定
`fd2_music_catalog`、原版 `FDMUS.DAT` identity、15個 track及 `fm`／`mt32` 兩個
正式 profile。每個 track 保留原始resource index，不新增曲名；每個 render必須
保存相對路徑、bytes、SHA-256、雙聲道、sample rate、PCM sample count與duration。
profile另保存render pipeline、provenance完整度及權利備註，但不得包含或散布MT-32 ROM。

loop契約只接受`whole_file_runtime_repeat`：原版`loop_count=0`時循環完整decoded
stream，`1`時只播一次；catalog明列`seam_evidence=unknown`。這是E1音訊近似，不宣稱
無縫波形、Miles timer或真實硬體parity。正式loader須拒絕錯schema、FDMUS identity、
track集合、profile集合、重複track、非固定相對路徑、錯hash、非Vorbis、非雙聲道、
sample rate／sample count漂移。runtime不得再把未分級`assets/music/`當靜默fallback；
缺選定render時只保持不播放，不修改`bgmCur`或campaign state。

驗收包含catalog可重生、30份OGG逐檔identity／Vorbis geometry、FM與MT-32各至少兩首
正常播放抽測、錯hash與缺track失敗即關閉、`loop_count=0/1` consumer回歸，以及搜尋
確認正式`musicPath`不再拼接未驗證路徑。現有render provenance不完整，故本切片只把
既有bytes納入嚴格catalog；未來重render須建立新版profile或更新完整工具／ROM私有
hash紀錄，不可沿用舊hash冒充相同產物。

實作結果：`tools/generate_music_catalog.py`可由兩套既有render重生catalog，並拒絕
非Ogg/Vorbis、非雙聲道及未審查loop tag。`internal/musiccatalog`只接受固定schema、
FDMUS identity、兩個profile、15個排序track及固定相對路徑，載入時逐一核對30份檔案
identity與Vorbis geometry；任一漂移即整批失敗。正式`playBGMCount`只由該catalog
解析路徑，舊`assets/music/`fallback已移除。Python測試、完整30份loader測試、錯來源／
錯render hash、未知profile／track，以及FM／MT-32各兩首runtime解碼／切換均已通過。
此狀態只關閉「既有render的可編輯catalog與正式consumer」，不提升其不完整render
provenance、無縫loop或人耳／三平台音訊為E2。

### 完整分離清冊的外部音樂根橋接契約

> 規格狀態：**CONFORMED**
> 前置證據：generated pack內15份MIDI仍按既有規則列為`blocked`；正式runtime的
> `music_catalog.json`與30份OGG位於另一個assets root，且已由上一節閉合。

完整manifest不複製大型OGG，也不保存`../../assets`等會穿越pack root的實體路徑。
頂層可選`runtime_catalogs.music`只保存：固定kind `fd2_music_catalog`、邏輯根
`runtime_assets`、安全catalog檔名`music_catalog.json`、catalog bytes／SHA-256、
schema version、`FDMUS.DAT`來源、2個profile、15個track及30個render計數。產生器只有
在呼叫者明確提供`--music-assets-root`時才建立bridge，並先完整驗證catalog與30份OGG；
驗證器看到bridge後則必須取得明確`--runtime-assets`，不能從manifest相對猜路徑。

bridge admission必須重新核對catalog identity、固定track／profile集合、每份render的
安全固定相對路徑、bytes／SHA-256與Ogg Vorbis geometry。缺外部root、缺catalog、錯
catalog hash、缺OGG或render漂移時整份manifest驗證失敗。15份MIDI保持`blocked`，因為
它們仍不是runtime格式；統計另列一份catalog、2個profile、15個track、30個已驗證外部
render，不得把兩種狀態相加冒充pack內OGG。乾淨clone沒有render時音樂失敗即關閉，
不得回退MIDI、legacy `assets/music/`或原始archive。

實作結果：`music_catalog_contract.py`成為manifest產生器與驗證器共用的外部音樂契約；
產生器新增明確`--music-assets-root`，驗證器新增`--runtime-assets`。五項bridge測試涵蓋
無複製成功路徑、缺外部root、render漂移、catalog hash漂移及路徑穿越，並與原有九項manifest／
catalog測試共同通過。現行實檔manifest已重生並完整驗證：pack本身現為39,825筆
（38,801 exported、1,005 intentionally raw、19 blocked），另有一份已驗證外部bridge：
2個profile、15個track、30個render；catalog為12,719 bytes、SHA-256
`8f2d2835971861e5fa97c8f33536022b53209b42d53bdd9fbfa062de9b880311`。

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

### 私人素材保存庫與本機完整版契約

> 決策狀態：**CONFIRMED（2026-08-29）**

- 完整分離包已由私人儲存庫提交 `21c9d775a6aa0b7035147313b2080610906cfa07` 保存；
  pack manifest SHA-256 為
  `94d90aee84f316dd7cbfcce21416a5aaf327a42a75342f9e531299ee35e06c38`。公開 `fd2_re`
  只保留解碼器、schema、雜湊清冊摘要、
  驗證工具與經審查的執行期比較圖，不提交可重組原作的完整逐檔素材。
- 私人儲存庫固定為 `https://github.com/wicanr2/fd2-assets-private.git`。原版分離包與
  現代重繪包分別置於 `packs/fd2-original-b97caf22/` 與
  `packs/fd2-modern-handpainted-a/`，不可混用來源、manifest 或權利聲明。
- 私人包固定根為 `packs/fd2-original-b97caf22/`，必須保留 `manifest.json` 的原始相對
  路徑。`raw/` 只供研究追溯與重新驗證，正式 `Game` 不得讀取。
- 開發端以 `FD2_ASSET_PACK` 指向私人包根；正式 consumer 只可讀分離 PNG、OGG、JSON、
  mask、palette 與具型別資料。必要素材缺失或 provenance 不符時失敗即關閉，不回退
  原版 `.DAT` 即時解碼。
- 公開引擎包不含私人素材。本機完整版組裝時，先驗證私人包 manifest、固定來源身份與
  `remake/assets` 外部 runtime catalog，再把兩者放入同一交付根；封包仍須保留來源
  HEAD、manifest SHA-256、內容清單與最終封包 SHA-256。
- 本機完整版只供庫擁有者或另有合法授權的交付情境使用；私人庫不能被 README、公開
  發行工作流或匿名下載 URL 當成公開素材來源。

實作：`remake/packaging/assemble-local-full.sh` 接受平台 payload、私人 pack 根及安全
輸出名，在一次性無網路 Docker 內先驗證 manifest／外部 runtime catalog，再合併
`remake/assets`與私人 pack。輸出同時包含 Linux／AppImage、Windows啟動器、
`FD2-LOCAL-FULL.json`及逐檔`SHA256SUMS`。2026-08-29以私人提交`21c9d775`及假 Linux
payload實跑，產生42,253個檔案；全部雜湊、pack結構與啟動器均通過。此收據證明組裝
契約，不替代三平台真實 binary／安裝生命週期驗收。

正式執行期邊界另於2026-08-29收緊：`nativeFDOTHERPath`／`nativeDATOPath`與
`FD2_ORIGINAL_FDOTHER`／`FD2_ORIGINAL_DATO`只存在於 `_test.go` source-oracle adapter，
不再編入正式遊戲。正式 binary 建置後已檢查不含這兩個 locator；原版 archive decoder
仍可留在匯入器與 oracle工具中，但不得由`Game` caller呼叫。

同一實際輸出樹再經來源 hash gate 與逐檔 manifest generator 驗證；2026-08-28完成
ANI遷移後，當時清冊為38,679筆：37,655筆為已匯出、1,005筆raw完整列為
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
由23筆擴為40筆（新增 #1..17），使 party cycle loader 不再讀 archive。固定原檔逐 indexed
pixel、mask、placement、delay 與非零 destination blit 對照均一致；#57 的768 bytes逐 byte
一致。當時的 Xvfb 抽測只證明 tail loader 在不提供 `FDOTHER.DAT` 時可由
`FD2_ASSET_PACK` 完成；它沒有涵蓋 `ending_preview` 建構器仍直接解碼的 #54，也沒有
涵蓋 `prepareNativeEndingDialogue` 仍直接讀取的 #5。先前把該結果概括成整個正式
`ending_preview` 不需 archive 是錯誤斷言；本勘誤保留其形成原因，且不得用該測試替
#54／#5 正式 caller 的缺原檔驗收背書。完整
manifest 現為38,092筆：37,068 exported、1,005 intentionally raw、19 blocked。本切片達
`DATA-READY`／`RUNTIME-E1`；`LoadMontageTailAssetsArchive` 只保留給 source oracle。

### 終局 FDOTHER #54 前綴動畫契約

> 規格狀態：**CONFORMED**
> 主證據：`docs/data/ida/fd2_ch29_terminal_body_ida.txt` 的
> 「FDOTHER #54 分離素材契約（2026-08-29）」；不重新推論 renderer 語意。

匯入器必須由固定雜湊 `FDOTHER.DAT` 的 resource54 輸出完整111格
`fd2_2935b_frame_table`，保存每格原始 placement／geometry、paletted indexed PNG
與 binary mask PNG，並在 `bank.json` 綁定 archive 與 raw resource 的大小、MD5、
SHA-256。穩定目錄為 `animations/fdother_054_ending_prefix/`，穩定 `asset_id` 為
`animation/FDOTHER_054/ending_prefix`。

嚴格 loader 必須拒絕錯 schema、來源、resource、raw identity、格數、重複／缺漏索引、
越出320×200的幾何、非固定相對路徑、非 indexed frame 或非二值 mask；不得回退讀取
archive。`ending_preview` 建構器改為只消費此 frame bank；對話準備則改用已分離
FDOTHER #5 entry0..19，不再保留 `fdotherPath`。固定原檔只留在 importer 與 oracle
測試。驗收包含111格逐 indexed pixel／mask／placement 對照，以及在
`FD2_FDOTHER`／`FD2_ORIGINAL_FDOTHER` 都不可讀時仍能由完整 `FD2_ASSET_PACK` 建立
前綴並準備對話；缺 #54 或 #5 任一必要資料時，必須在發布半套 player 前失敗即關閉。

2026-08-29 實作已完成：匯入器輸出111張indexed frame、111張binary mask及一份
`bank.json`；嚴格loader固定archive／raw identity、格數、路徑及幾何，且固定原檔
oracle逐格一致。正式`ending_preview`及其對話準備已移除`fdotherPath`與
`DecodeResource`／`ReadResource`，缺原版FDOTHER／FDTXT／DATO的Docker／Xvfb抽測
通過。重生後完整manifest共39,157筆：38,133 exported、1,005 intentionally raw、
19 blocked，並通過來源／輸出hash驗證。本切片達`DATA-READY`／`RUNTIME-E1`。

### 第21戰戰後 FDOTHER #34 天空之鑰動畫契約

> 規格狀態：**CONFORMED**
> 主證據：`docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt`；既有`0x24336`
> caller／排程為`RE-CLOSED`，本切片不得重新推論DAC首相位或DOS時鐘。

匯入器必須由固定雜湊`FDOTHER.DAT` resource34輸出完整101格
`fd2_2935b_frame_table`，保存每格原始placement／geometry、paletted indexed PNG
與binary mask PNG。metadata固定raw size 102,345、MD5
`84dca404546a3a407d72f139cb934a40`、SHA-256
`53f120fa4b1fab74c6b3998ec3ef8a9a2363461980ad38a7ffef2400e79b0c4d`；穩定目錄
為`animations/fdother_034_ch20_sky_key/`，`asset_id`為
`animation/FDOTHER_034/ch20_sky_key`。

嚴格loader須拒絕錯schema、來源、resource、raw identity、格數、重複／缺漏索引、
界外幾何、路徑、PNG mode或mask；並固定frame0 `(148,94,24,24)`、frame68／69
`(146,95,32,23)`與frame100 `(148,94,24,36)`。正式
`startNativeCh20SkyKeySequence`只由此bank與已分離ANI #0建立job，不保留
`nativeFDOTHERPath`或archive fallback。驗收包含101格固定原檔oracle，以及在
`FD2_FDOTHER`／`FD2_ORIGINAL_FDOTHER`不可讀時從正常ch20 post owner完成資產預檢；
缺bank時不得先移動鏡頭、修改campaign state或發布半套演出。

2026-08-29 實作已完成：101張indexed frame、101張binary mask及`bank.json`皆由
固定原檔重生，逐格oracle一致。正式`startNativeCh20SkyKeySequence`已無archive
path／decoder；第21戰完整成功臂測試在map20初始化後、`0x24336`啟動前撤掉原版
FDOTHER，仍完成演出、JOIN、城鎮與存檔邊界。重生後manifest共39,360筆：38,336
exported、1,005 intentionally raw、19 blocked，並通過來源／輸出hash驗證。本切片
達`DATA-READY`／`RUNTIME-E1`。這一批當時尚未處理的map初始化
#1／#3／#5／#6／#9／#55依賴，已由下一節的後續切片取代；不得再以本段的歷史
狀態重開工作。

### 戰場初始化 FDOTHER #1／#3／#5／#6／#9／#55 契約

> 規格狀態：**CONFORMED**
> 主證據：`docs/data/ida/fd2_map_runtime_assets_separation_evidence.txt`；本節只把
> 已閉合 archive/resource ABI 轉為分離素材契約，不新增 renderer 語意。

正式 `loadNativeMapAssets` 必須只消費分離素材。一般戰場原子預檢包含既有
FDSHAP、FDICON、FDOTHER #1 range-overlay、#3 完整23×256 LUT、#5 map HUD
mixed-codec entries、#5完整138-entry LMI1與#6完整230-entry LMI1；#9完整12-entry
spawn-intro bank也必須在資產 bundle 建立前驗證，避免增援／入隊事件走到一半才
發現缺件。map 28／29另要求#55的320×200 opaque indexed surface。任一必要 pack
缺失或 metadata／PNG／entry topology 不符時，初始化須失敗即關閉，且不得讀取
`FDOTHER.DAT` 作 fallback。

穩定輸出如下：

| resource | 穩定目錄／檔案 | 固定拓撲 |
|---|---|---|
| #1 | `sprites/fdother_001_range_overlay/bank.json` | 20個24×24 indexed frame／mask／remap mask |
| #3 | `palette/fdother_003_luts.json` | 23項，每項256個0..255元件 |
| #5 HUD | `ui/fdother_005_item_panel/resource.json` | caller-proven mixed-codec entries，含#130..#132與#31..#52 |
| #5完整bank | `ui/fdother_005_lmi1_opaque/bank.json` | 138項 `opaque_high_run` indexed PNG |
| #6 | `effects/fdother_006_lmi1_opaque/bank.json` | 230項 `opaque_high_run` indexed PNG |
| #9 | `animations/fdother_009_spawn_intro/bank.json` | 12項 `opaque_high_run` indexed PNG及固定幾何 |
| #55 | `surfaces/FDOTHER_055/resource.json` | 320×200 `raw_indexed_opaque` PNG |

每份metadata固定`schema_version=1`、穩定`asset_id`、原始archive及resource raw
identity、codec、幾何、entry index與相對路徑。完整LMI1 bank的逐項PNG必須能由
固定原檔oracle還原成相同width／height／indexed pixels；#3須逐byte相同；#5 HUD
另需對照mixed-codec frame與mask；#55須逐pixel相同。驗收還必須在
`FD2_FDOTHER`／`FD2_ORIGINAL_FDOTHER`均不可讀時，抽測一般地圖與map 28／29的
正式初始化。缺任一必要分離檔時，不能發布半套`nativeMapAssets`。

2026-08-29實作已完成：匯入器固定raw identity並輸出#3的23組LUT、#5的
138-entry ordinary LMI1、#6的230-entry ordinary LMI1、#9的12-entry LMI1、
#55的320×200 indexed surface，且#5 mixed-codec bank補齊HUD #130..#132。
所有新增loader均拒絕錯schema、來源、hash、索引、codec、幾何與PNG；逐項
oracle對固定原檔的width／height／indexed pixels（HUD另含mask）一致。正式
`loadNativeMapAssets`只讀分離#1／#3／#5／#6／#9／#55，command 0..8、敵方9、
24、29、32..35及action overlay的過期`nativeFDOTHERPath()`存在檢查亦已移除。
一般map 0／23／32及map 28／29在兩個FDOTHER環境路徑都指向不存在檔案時仍完成
初始化；缺分離pack則維持失敗即關閉。重生後manifest共39,520筆：38,496
exported、1,005 intentionally raw、19 blocked，來源與輸出hash驗證通過。本切片

#### #1／#6 manifest 納入勘誤

> 狀態：**CONFORMED／LEDGER-DATA-READY**（2026-08-29）

上述 runtime 與逐項 oracle 已證明 #1、#6 是完整標準 bank，但 manifest classifier
只識別 `sprites/fdicon` 與既有 `ui` 路徑，漏列
`sprites/fdother_001_range_overlay` 的 `bank.json`／20組 frame、mask、remap mask，以及
`effects/fdother_006_lmi1_opaque` 的 `bank.json`／230張 frame。這是清冊 provenance
缺口，不是 decoder、runtime 或原版語意缺口。

generator 必須只對這兩個已閉合穩定路徑新增精確納入規則：#1 文件分類為 metadata、
PNG 分類為 map sprite；#6 文件分類為 battle animation、`bank.json` 分類為 metadata。
每筆皆固定 `source_file=FDOTHER.DAT`、resource 1 或 6，並由目錄中的
`sprite_NNNN`／`entry_NNN` 記錄 `source_frame`。不得泛化成任意 `effects/` 或
`sprites/fdother_*` 都已證實。重生後 ledger 應以實際輸出 asset IDs 將 #1／#6 從
unknown 提升為 standardized；既有嚴格 loader、raw oracle 與正式 consumer 證據繼續
作為完整性依據，不新增重複 catalog。

實作已精確納入 #1 的61份 metadata／PNG 與 #6 的231份 metadata／PNG；相似但未經
證實的 `effects/fdother_007*` 路徑不會被分類。合成測試與真實重生共同證明兩筆 ledger
轉為 standardized，當時 manifest 增至39,812筆，unknown 降至97筆；後續 #77
音效切片已再更新現況數字，pack validator 仍會
逐檔驗證相對路徑、來源關聯與 SHA-256。
達`DATA-READY`／`RUNTIME-E1`，不提升一般玩家`PLAYER-E2`或未命名entry語意。

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

#### FDOTHER #102 捲動列調色盤切換

> 規格狀態：**CONFORMED**（2026-08-29）
> 主證據：[`fd2_title_fdother102_palette_pulse_ida.txt`](../data/ida/fd2_title_fdother102_palette_pulse_ida.txt)

匯入器須另輸出 `palette/fdother_102.json`，沿用 palette schema v1 並嚴格驗證
固定 `FDOTHER.DAT` 身份、resource 102、raw size 768 與全部 component `0..63`。
正式 loader 應把 #101 與 #102 同時預檢；任一缺少或不合契約即不發布標題資產。

runtime 保留同一份320×735 indexed pixels，只切換 palette。觸發列固定為
`520,430,410,340,310,300,240,180,150,130,110,87,64,22`；命中列由 #101
切至 #102，11 個原版30ms列迴圈後回復 #101。60 Hz 可以依現有連續 `scrollY`
投影，但必須測試每個觸發邊界、觸發前後與區間外皆選到正確 palette。不得把
未證實高階色號用途寫入 typed contract。

實作後匯入器輸出第四份標題捲動DAC，正式loader原子預檢#101／#102，runtime依
上述14筆列值切換同一indexed surface。固定archive逐byte oracle、缺包失敗即關閉、
完整0..535列與浮點邊界測試均通過；resource ledger把FDOTHER #102由unknown提升為
standardized。本切片達`DATA-READY／RUNTIME-E1`，不提升逐硬體時間或人耳音效E2。

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

#### 一般物理攻擊動態 pair 的 consumer 規格

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）；主證據見
> [`fd2_physical_attack_separated_provider_20260829.txt`](../data/ida/fd2_physical_attack_separated_provider_20260829.txt)。

上述archive caller歸零只證明沒有直接讀`FIGANI.DAT`，不代表普通玩家攻擊已完整
消費分離pack。本批開始時，敵方一般／mode 11會在結算前以
`ensureNativeAttackPresentation`載入`selector*3+1`攻擊與`selector*3`守方待機；玩家
普通攻擊卻只查啟動時的22-resource legacy cache，cache miss仍先提交傷害。這是正式
consumer缺口，不是需要重做FIGANI格式或selector RE。

本切片固定要求玩家與敵方在方向、亂數、HP、EXP、死亡獎勵及acted改變前共用同一
分離provider。provider必須完整驗證兩個resource的frame／mask／位置／delay及排程後
一次發布；缺任一輸入即零遊戲狀態修改並保留玩家目標選擇，不回退archive或固定delay。
`newAtkAnim`只消費通過預檢的cache。原始archive不可讀的普通玩家攻擊、缺pack原子
拒絕與mode 11既有回歸是最低驗收；法術與特殊劇情攻擊另立切片。

實作現已在玩家普通攻擊的方向、亂數與結算之前呼叫同一provider；缺pair會保留玩家
選取狀態，且HP、acted、面向與注入RNG均不變。敵方一般路徑也把面向改到provider
成功之後。以啟動cache為空、原始`FIGANI.DAT`指向不存在路徑的測試，已從完整分離包
動態載入玩家selector4→resource13與守方selector96→resource288，建立演出後才提交
傷害；mode 2既有聚焦回歸同批通過。一般物理攻擊動態pair至此達`RUNTIME-E1`，不再
列為A2／A3下一批；法術及截圖專用直接入口沒有因此提升。

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

### 標題 FDOTHER #77 音效分離契約

> 狀態：**CONFORMED／RUNTIME-E1**（2026-08-29）。RE 主證據為
> [`fd2_title_fdother77_sound_bank_ida.txt`](../data/ida/fd2_title_fdother77_sound_bank_ida.txt)。

本切片沿用既有 `fd2_pcm_sound_bank`，新增 `asset_id=sfx/FDOTHER_077`。固定
resource 大小 52,031 bytes、container count 5、zero-length tail index 4；只輸出
selector 0..3 四筆 OGG。每筆 metadata 保存 raw byte count／SHA-256、固定來源 identity、
`sample_rate=11025`、`channels=1`、`sample_format=unsigned_u8` 與
`timing_evidence=hardware-spec_approximation`。selector 0／3 雖有 caller，但高階
演出名稱與精確重製排程仍未知，因此只保存，不接播放。

正式 runtime 在既有分離音效整批 admission 中載入 #77；任一 metadata／OGG 缺漏、
來源漂移、selector 越界或 OGG 解碼失敗時整批拒絕，不回退 #31 或原始 archive。
標題主選單及 LOAD 槽位選擇的上／下移動消費 #77/2，確認消費 #77/1；其他 UI
仍維持各自已證實的 #31 owner。驗收至少包含：固定 raw hash 重生、四筆 OGG probe、
無原始 `FDOTHER.DAT` 時 #77 完整解碼、主選單／槽位 typed event 使用專用 bytes，
以及缺 pack 的失敗即關閉。這不宣稱 selector 0／3 已接、完整開場音訊或人耳 E2。

實作結果：canonical 匯出現為 18 份 metadata／75 筆 OGG；#77 四筆 OGG 均由
固定 raw bytes 重生並通過解碼 probe。嚴格 loader 接受且只接受
`container_count=5`、`zero_length_tail_index=4` 與四筆連續 selector；正式整批
loader 安裝 #77/2、#77/1 到標題專用欄位，主選單與 LOAD 槽位不再借用 #31。
匯出器／清冊 14 項測試、完整 pack validator、`internal/fdother` 及標題／分離音訊
有界 Go 回歸均通過。整個 `cmd/fd2` 套件另有三個既有 fixture 因直接尋找未掛入
`remake/assets` 的字型／發行商／教會分離資產而失敗，未列為本切片通過證據。

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

### FDOTHER #78：ANI #1 companion 音效（READY）

- 原版證據：[`fd2_ani1_fdother78_sound_ida.txt`](../data/ida/fd2_ani1_fdother78_sound_ida.txt)。
- typed data：`sfx/FDOTHER_078/resource.json` 必須是 schema 2、兩筆目錄項、恰好一筆
  非空 `sample_000.ogg` 與 index 1 的 0-byte 尾哨兵；來源雜湊固定為參考版 FDOTHER.DAT。
- cue：只允許 `ani1_frame0_companion`。ANI #1 第一幀啟動 selector 0；自然結束或略過
  ANI #1 時停止專用 voice。不得把它分類成畫面泛光或一般戰鬥音效。
- runtime：標題素材預檢時整庫解碼；缺 metadata、OGG、來源契約或 sample 時失敗即關閉，
  不回退原始 archive。音效設定關閉及截圖模式維持靜音。
- 時序：11025 Hz 只標記硬體規格近似（hardware-spec approximation），不追 DOS DAC／PIT
  逐週期一致。
- 驗收：exporter 形狀測試、嚴格 loader 測試、標題排程觸發／停止測試，以及正式分離包
  無原始 archive 解碼抽測。

### FDOTHER #79：raw pending code 1 固定兩幀呈現（READY／runtime blocked）

- 原版證據：[`fd2_pending_code1_fdother79_ida.txt`](../data/ida/fd2_pending_code1_fdother79_ida.txt)。
- typed data：`animations/fdother_079_pending_code1/bank.json` 使用既有
  `fdother_frame_bank` schema，固定來源 #79、6801 bytes、兩幀及完整 raw MD5／SHA-256；
  frame 0 固定 `(69,61,181,75)`，frame 1 固定 `(130,141,55,8)`。
- 每幀輸出 indexed PNG 與 binary mask；loader 必須驗證來源、count、index、座標、
  幾何、安全相對路徑及完整 PNG／mask。缺任一項即失敗即關閉，不回退原始 archive。
- typed schedule 固定為：停止 BGM → palette 前置 → 清 320×200 → frame0 → 65 phase
  palette 淡入（2ms／phase）→ 等待9 tick → frame1 → 等待36 tick → 釋放。
- 高階名稱、逐章觸發原因與當下 palette source 均保持 unknown。正式 Game 接線維持
  blocked，直到 raw pending-code producer／consumer 能從一般玩家路徑交付此 schedule；
  direct-entry 或測試注入不得提升為 `RUNTIME-E1`。
- 驗收：固定原檔逐幀 indexed／mask oracle、malformed／missing pack 拒絕、schedule
  順序測試與 manifest provenance。這一階段只達 `DATA-READY`。
