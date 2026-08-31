# 61 — 現代美術主題正式執行期規格

狀態：`RUNTIME-E1-PARTIAL`（2026-08-31）

## 範圍

本規格分別定義索爾故事對話的獨立現代頭像，以及 FDICON selector 0、68 的
兩組 12 格候選。selector 是圖像索引，不等於角色身分；兩者不得混用。
它不授權從合成概念稿反切地圖圖塊或 HUD，也不改變忠實原版主題。

## 原始證據與既有消費端

- 原版分離資產：`assets/portraits/DATO_000_m0.png`；`DATO` resource 0、frame 0。
- `dato.Frame` 已證實為 `80×80`，故事說話者必須具備四幀；正式載入端位於
  `remake/cmd/fd2/native_story_dialogue.go` 的 `loadNativeSeparatedPortrait`。
- 原生上框頭像的一般左上角是線性位址 `0x728`，即 `(232,5)`，覆蓋
  `x=232..311, y=5..84`。下框的 `0x9017` 是 `0x4E8E1` 右向左 blit 的每列
  **右緣**，不是左上角；`80×80` 頭像實際覆蓋 `x=8..87, y=115..194`。
- `ComposeNativeStoryDialogueBaseFrame` 目前會把 DATO 頭像直接寫進
  `320×200` 索引畫面；`presentNativeClassFrame` 再套用原版調色盤並放大兩倍。
- 證據等級：上述形狀、座標、載入者與消費端皆為**已證實**；來源是受版控
  原始碼、分離資產測試及既有原版位址證據。現代畫風本身是使用者核准方向，
  不是原版行為證據。

### 2026-08-31 勘誤

本規格初稿曾把 `0x9017 % 320 = 87` 誤寫成下框左上角。直接 consumer
`blitNativeDialoguePortraitAt` 對此位址呼叫 `BlitRightToLeftAtOffset`，而該函式
逐列由 right edge 遞減目的位址；因此新證據直接否定舊座標，修正為左上角
`(8,115)`。原始位址 `0x9017` 保留不變。

## 型別與呈現契約

1. 現代故事頭像是獨立 `80×80` 真彩色 PNG，以穩定 `asset_id`、SHA-256、
   原始 frame 身分及嘴型狀態登錄於 `assets/themes/modern/catalog.json`。
2. 現代頭像不得先量化進原版 16 色索引畫面；正式 renderer 應先呈現原生框與
   文字，再在相同邏輯座標以真彩色層覆蓋完整 `80×80` 頭像區。
3. 上／下框錨點與 `NativeStoryDialogueTextGeometry` 不得改動。
4. 第一個候選只提供索爾閉嘴 frame 0。依使用者既有裁決「嘴型不考慮」，
   現代主題採靜態閉嘴策略；正式 consumer 在原版嘴型 phase 仍覆蓋同一張現代
   frame 0，不可露出或混用原版 frame 3。
5. 主題必須整組預檢並原子切換。檔案缺漏、尺寸不符、雜湊不符或 speaker
   沒有對應資產時，現代主題失敗即關閉，不得在同一畫面混搭原版與現代頭像。

## 目前候選

- `modern.sol.portrait.style_a.frame0`
- 執行期檔：`sol-portrait-style-a-v2-80.png`，`80×80`，SHA-256
  `6ecc692c30973c62900dc9983ea878df29394d30bfe28d7bf6931dd223131e82`
- 母稿：`sol-portrait-style-a-v2-master.png`，`1254×1254`。
- `modern.hano.portrait.style_a.frame0`
- 執行期檔：`hano-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `89d108d4ccea833ab0f2167aa5c220b9cea2694976347fff241cf016ec48203d`
- 母稿：`hano-portrait-style-a-v1-master.png`，`1254×1254`。2026-08-31
  勘誤：`DATO_001_m0`、多章 `speaker=1` 與繁中實體清冊三方皆指向哈諾，
  不是亞雷斯；原先 `modern.ares...` 名稱及私有檔名已更正，像素內容與雜湊
  未改。保留棕紅長髮、深色頭帶與右向側面，索爾現代母稿只作畫風參考。
- `modern.ares.portrait.style_a.frame0`
- 執行期檔：`ares-portrait-style-a-v2-80.png`，`80×80`，SHA-256
  `23e12ae24d8f67bde815c32c82544d99cd2f8af1b844bc691422a3d7ed7cfa39`
- 母稿：`ares-portrait-style-a-v2-master.png`，`1254×1254`；角色身分由
  `DATO_004_m0`、多章 `speaker=4 / speaker_name=亞雷斯` 與繁中實體清冊
  三方一致確認。保留原版左向、深藍包頭帽、綠額帶、紅棕短髮及肩後長柄武器；
  哈諾現代母稿只作畫風參考。
- `modern.lorna.portrait.style_a.frame0`
- 執行期檔：`lorna-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `46d339af0f1aa08c95f5fb1e47a5b434590f961b119640b045fda70c8dfa6ffc`
- 母稿：`lorna-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_005_m0`、第 8／11／16／24／31 章的 `speaker=5 / speaker_name=洛娜`
  與繁中實體清冊三方一致確認。保留正面構圖、灰綠長髮、銳利眉眼、紅唇與
  深色高領；亞雷斯現代母稿只作畫風參考。
- `modern.leidin.portrait.style_a.frame0`
- 執行期檔：`leidin-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `f26171b353719930c905b534c6bd8b69db82a3d15c958c1530538a5ddca58bb1`
- 母稿：`leidin-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_006_m0`、多章 `speaker=6 / speaker_name=萊汀` 與繁中實體清冊三方
  一致確認。保留左向側面、深棕束髮、窄深色頭帶、狹長眼及左下淺色尖角；
  亞雷斯現代母稿只作畫風參考，不能把藍帽／綠額帶帶入萊汀。
- `modern.tino.portrait.style_a.frame0`
- 執行期檔：`tino-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `f0d350e52c9d67250b1f63e8775e7eb9a99e030eb815bca4b37ae6f2f887b64d`
- 母稿：`tino-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_002_m0`、`ch03.json` 的 `speaker=2 / speaker_name=鐵諾` 與繁中實體
  清冊三方一致確認。保留淺灰綠兜帽、深色長髮、額帶、右向側面及深藍灰護甲；
  哈諾現代母稿只作畫風參考。
- `modern.harvat.portrait.style_a.frame0`
- 執行期檔：`harvat-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `709f889fa71c36644c279af2aefd889fd46098545c345dd0223d7a748ffd8109`
- 母稿：`harvat-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_003_m0`、多章劇本的 `speaker=3 / speaker_name=哈瓦特` 與繁中實體
  清冊三方一致確認。保留原版左向、巨大圓鼻、紅棕濃鬍與緊湊臉部特寫；鐵諾
  現代母稿只作像素密度、光影及背景參考。
- 上述母稿與執行期檔皆留在忽略版控的私人素材目錄；公開儲存庫只保存契約與雜湊。

## FDICON 地圖人物 12 格契約

### 原始證據

- 原始來源是 `FDICON.B24` 第 68 組，不是 `FIGANI_068`；後者已由清冊證實為
  `empty_header_zero`，不可混作戰鬥動畫來源。
- 角色建構資料把索爾基礎 selector 設為 0；轉職等 writer 可另行改寫 selector，
  且既有晚期悠妮資料曾使用 `0x44`。因此第 68 組只能稱為原始 selector 圖組，
  不能永久命名為索爾或悠妮。這項勘誤直接取代 2026-08-31 初稿的「索爾
  fig_068」錯誤斷言。
- 固定版 `FDICON.B24`：624,010 bytes，MD5
  `46f793540209a063ea73a5373ca14bf4`；完整 SHA-256 與版本身分仍以
  `docs/data/fd2-reference-files.json` 為準。
- 每組固定 12 格、每格 `24×24` RGBA；索引公式
  `cache_slot × 12 + pose × 3 + cycle` 已由正式載入與測試證實。
- `f00..02／f03..05／f06..08／f09..11` 對應下／左／上／右，格內是三個
  步態週期。這個方向順序是 exporter 實測與既有程式資料流的**強推論**，不是
  單憑檔名升格的二進位精確結論。

### 現代候選契約

1. `modern.fdicon.group_068.style_a` 必須一次提供 12 張獨立 `24×24` PNG，保留
   frame 0..11、pose 0..3 與 cycle 0..2 的穩定映射。
2. 每張 alpha 只能是 0 或 255；catalog validator 在私人素材驗證模式逐像素檢查。
3. 生成的第 11 格曾因雙劍與比例漂移被拒收，後續暫以第 9 格佔位。現行候選
   以第 9 格的上半身及第 10 格的下半身做確定性像素合成，保持人物尺度、裝備
   側別、方向與 `24×24` 邊界，形成獨立第三步態。這是**現代近似**，不是原版
   第 11 格的逐像素重繪。
4. catalog 現以 `three_distinct_cycles` 登記，整組升為 `runtime_candidate`；
   renderer 尚未接線、正常玩家尚未擷圖，所以不可標為 `runtime_ready`。
5. 原生 indexed bank 與正規化 RGBA loader 是兩條不同 consumer；接線時兩條
   路徑必須共同抽測，且現代主題缺任一格即整組失敗即關閉。

### selector 0 與 68 候選

- `modern.fdicon.group_000.style_a`：以原版 `fig_000_f00..f11` 為動作基準生成
  3×4 母稿，再以確定性背景分割、逐格裁切及二值 alpha 轉成 12 張 `24×24`
  PNG。索爾的基礎建構資料使用 selector 0，因此這組可驗證第一關索爾；但
  catalog 仍以原始 selector 命名，不宣稱 group 0 永遠只供索爾使用。
- `modern.fdicon.group_068.style_a`：沿用第 68 組原版輪廓的現代候選。它不綁
  固定角色身分；第 11 格是第 9 格上半身與第 10 格下半身的現代近似。
- 兩組都具有 12 個不同 SHA-256、`24×24`、二值 alpha 與三個不同週期，列為
  `runtime_candidate`。私人母稿與逐格 PNG 不進公開 Git，公開 catalog 只保留
  可重現契約與雜湊。

### 地圖人物 consumer 現況

- `loadModernStoryPortraitSet` 現會同時預檢 catalog 中的地圖人物組：group
  0..95、12 個安全檔名、逐格 SHA-256、`24×24`、二值 alpha、互異雜湊及三週期
  policy，任何一項不符即拒絕整個現代主題。
- `loadGame` 在完整預檢後，才以各組 12 張真彩色圖原子取代正規化
  `g.sprites[0]`／`g.sprites[68]`；其他 group 不變，忠實主題預設路徑也不變。
- 原生 indexed 戰場 compositor 仍直接消費 `NativeMapSelectorCache`，尚未加入
  真彩色覆蓋層。此路徑保持原版 sprite，不偷偷量化或混搭；因此地圖人物目前
  是 `RUNTIME-E1-PARTIAL`，待原生與正規化同狀態抽測後才能升級。

## 正式 consumer 現況

- `FD2_THEME=modern-handpainted-a` 是目前顯式候選入口；預設不設定時完全
  不改變忠實原版主題。
- `FD2_MODERN_THEME_PACK` 指向本機私人或完整版素材目錄；正式 loader
  同時檢查 catalog 身分、speaker ID、frame 0、閉嘴策略、`80×80`、
  SHA-256 與逐像素完全不透明。
- `prepareNativeDialogueFrames` 在每句對話發布前預取當前 speaker。主題
  缺少該 speaker 時整句失敗即關閉，不回退原版頭像。
- 穩定頁、逐字頁與原版嘴型 phase 都由同一真彩色 frame 0 覆蓋完整
  `80×80` 區域；開框與收框本來就沒有頭像，維持原生 indexed 畫面。

## 驗收與停止線

- catalog validator 必須驗證候選檔尺寸與 SHA-256。
- 地圖人物候選另須驗證 12 個穩定檔名、逐格 SHA-256、`24×24`、二值 alpha
  與三個不同週期；現代第三步態的像素合成 provenance 必須保留。
- renderer 單元測試已固定上／下框矩形；loader 已驗證身分、尺寸、
  雜湊與不透明契約。缺檔與未知 speaker 仍由正式預取路徑原子拒絕。
- 正規化故事下框已於 ch00 frame 90 實際擷圖，確認新頭像位於左側、
  面向文字且無原版嘴型混搭；收據見
  `docs/data/ui-traces/modern-sol-portrait-ch00-e1.json`。原生 indexed 上／下框
  仍需各抽一張。
- 現行 consumer 已達 `RUNTIME-E1-PARTIAL`，但正常故事上／下框擷圖與其他
  speaker 素材尚未完成，catalog 仍維持 `runtime_candidate`，不可冒稱完整主題。
- 地圖人物 loader 的完整／柔邊拒絕測試已通過；正規化 consumer 已接，原生
  indexed consumer 與第一關四方向正常玩家擷圖仍待完成。
