# 61 — 現代美術主題正式執行期規格

狀態：`RUNTIME-E1-PARTIAL`（2026-09-01）

## 範圍

本規格分別定義現代故事頭像，以及 FDICON selector 0–21、68 的二十三組 12 格候選。
selector 是圖像索引，不等於故事說話者或頭像身分；兩者不得混用。
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
- `modern.celia.portrait.style_a.frame0`
- 執行期檔：`celia-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `e0f8b3644662c10044bb0d6082ed8553c54aa26b50df7084120c9100c96102f1`
- 母稿：`celia-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_008_m0`、86 筆 `speaker=8 / speaker_name=希莉亞` 與繁中實體清冊
  三方一致確認。人工影像檢視保留正面構圖、棕色波浪長髮、藍色額飾與中央
  紅寶石、紅唇及灰藍高領；洛娜現代母稿只作畫風參考。
- `modern.yuni.portrait.style_a.frame0`
- 執行期檔：`yuni-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `f8d28f8ed44f6a4063dd4375aafe965c7b688a92fd33a18963adc7f0efd535b0`
- 母稿：`yuni-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_009_m0`、107 筆 `speaker=9 / speaker_name=悠妮` 與繁中實體清冊
  三方一致確認。人工影像檢視保留正面構圖、深紅長髮與頭帶、中央藍寶石、
  狹長眼及暗紅高領。初遇倒地昏迷屬場景演出，不混入正常對話頭像。
- `modern.marin.portrait.style_a.frame0`
- 執行期檔：`marin-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `75701dc5fa307e09aa393d46f863465b3a06bbb64e1c97d34073d39c7d96d4e1`
- 母稿：`marin-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_010_m0`、18 筆 `speaker=10 / speaker_name=瑪琳` 與繁中實體清冊
  三方一致確認。人工影像檢視保留正面構圖、紫紅齊肩短髮、中央金色髮飾、
  狹長眼、紅唇與金紅華麗高領；悠妮現代母稿只作畫風參考。
- `modern.sophia.portrait.style_a.frame0`
- 執行期檔：`sophia-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `39da886f78193f0ff6082f10aa7d65c1f201771ad5ea4193ceed664cd640c2ff`
- 母稿：`sophia-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_011_m0`、19 筆 `speaker=11 / speaker_name=索菲亞` 與繁中實體清冊
  三方一致確認。人工影像檢視保留灰綠短髮、兩側長鬢髮、紅唇、黑色細頸圈
  與藍灰肩甲；瑪琳現代母稿只作畫風參考。
- `modern.kelly.portrait.style_a.frame0`
- 執行期檔：`kelly-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `b92b5118a7b17b455854e8eca92f19fd8a78af614e876af902d7880d5333a393`
- 母稿：`kelly-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_012_m0`、16 筆 `speaker=12 / speaker_name=凱麗` 與繁中實體清冊
  三方一致確認，識別字採英文實體目錄的 `Kelly`。人工影像檢視保留微向右側、
  金色短捲髮、垂眼、紅唇、金色耳飾與深藍高領。
- `modern.beckway.portrait.style_a.frame0`
- 執行期檔：`beckway-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `cd6ccdfb8b8df27b01aa5a0461203f8394d5a83eb737a707ad479af334682049`
- 母稿：`beckway-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_013_m0`、60 筆 `speaker=13 / speaker_name=貝克威` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Beckway`。人工影像檢視保留
  正面金色長直髮、寬深藍頭帶、微瞇眼、淡笑及深藍頸圈。
- `modern.shan.portrait.style_a.frame0`
- 執行期檔：`shan-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `9123e7894502efab46b062c4c10138ee9d7849ac06a06cc13c7ddb9e0946f970`
- 母稿：`shan-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_014_m0`、23 筆 `speaker=14 / speaker_name=珊` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Shan`。人工影像檢視保留
  微向左側、深藍長髮、橙金細額飾與紅寶石、狹長眼、紅唇及藍色高領。
- `modern.kylas.portrait.style_a.frame0`
- 執行期檔：`kylas-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `8334058bcee062d77b6cc93641c057b12c2ef1002aa34bae2b0807b9dde2e8f1`
- 母稿：`kylas-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_016_m0`、30 筆 `speaker=16 / speaker_name=凱拉斯` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Kylas`。原版影像確認它是向左
  的棕黑龍首，具長吻、尖牙、綠眼、後掠角與深色頸部，不得套用人類頭像模板。
- `modern.miasdord.portrait.style_a.frame0`
- 執行期檔：`miasdord-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `9487e75795103c8a5c98bfd79a3e264dc6780fec8fccfb2b6634989aa250404c`
- 母稿：`miasdord-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_017_m0`、22 筆 `speaker=17 / speaker_name=米亞斯多德` 與繁中實體
  清冊三方一致確認，識別字採 `Miasdord`。原版影像確認它是正面暗紅龍／魔獸首，
  具亮紅眼、多根尖角、尖長側耳、黑色鬚棘與灰藍肩部，不得套用人類頭像模板。
- `modern.mitty.portrait.style_a.frame0`
- 執行期檔：`mitty-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `4230f203f3fefb576edcfa46dc5e768898d9784073e584187e687eea09fb64a1`
- 母稿：`mitty-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_018_m0`、29 筆 `speaker=18 / speaker_name=蜜蒂` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Mitty`。人工影像檢視保留
  正面構圖、蓬鬆金色長捲髮、深色眉毛、紅唇與沉著表情；蘇菲亞現代母稿
  只作畫風與完成度參考。
- `modern.rodman.portrait.style_a.frame0`
- 執行期檔：`rodman-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `4abe00cc62692a6a122bf26334cfc54d9e66056153021a6c0ae4634817db105e`
- 母稿：`rodman-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_019_m0`、12 筆 `speaker=19 / speaker_name=羅德曼` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Rodman`。人工影像檢視保留
  向左三分之二側面、棕色後梳長髮、濃密八字鬍與下巴鬍鬚、銳利眼神及
  灰藍肩甲；蜜蒂現代母稿只作平滑手繪完成度與背景參考。
- `modern.sarah.portrait.style_a.frame0`
- 執行期檔：`sarah-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `4e738256880aa93af830e45354f80cf2b199463fb43d03bc320a306823ed57e7`
- 母稿：`sarah-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_020_m0`、6 筆 `speaker=20 / speaker_name=莎拉` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Sarah`。人工影像檢視保留
  正面精靈尖耳、灰綠長直髮、細紅色頭帶、纖細眉眼、紅唇與深色高領；
  蜜蒂現代母稿只作平滑手繪完成度與背景參考。
- `modern.jonah.portrait.style_a.frame0`
- 執行期檔：`jonah-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `d739a4e499b02529da54062761de9bd61eac7e6d00cab4b67a8714f56bb498c9`
- 母稿：`jonah-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_021_m0`、84 筆 `speaker=21 / speaker_name=約拿` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Jonah`。人工影像檢視保留
  老年人類聖者、貼合頭部的白色頭巾／白髮、巨大白色鬍鬚與鬍髭、飽經風霜
  的面孔、堅毅眼神及素色棕灰聖者衣袍；不得畫成戴深藍尖帽的巫師。
  2026-09-01 經角色資料、使用者勘誤與人工圖像再次核對後，
  約拿只保留 `DATO_021_m0` 為身分來源；不得再以其他老年男性角色作角色來源，
  避免把風格參考誤讀成身分證據。
- `modern.carlos.portrait.style_a.frame0`
- 執行期檔：`carlos-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `8ebbb556ed3462f0efe824344702fc59fd5219359d5fc29c42d9185861d71ebe`
- 母稿：`carlos-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_022_m0`、16 筆 `speaker=22 / speaker_name=卡里斯` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Carlos`。人工影像檢視保留
  正面略向左、深藍黑高馬尾、兩側長鬢髮、冷峻微瞇眼、黑色高領與暗紅肩部；
  莎拉現代母稿只作平滑手繪完成度與背景參考。
- `modern.roland.portrait.style_a.frame0`
- 執行期檔：`roland-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `eecbc0dc0688e7bc46233f0a6e40c50a1d54d12e4bbb292f9ec19cc0888554c9`
- 母稿：`roland-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_023_m0`、3 筆 `speaker=23 / speaker_name=羅蘭` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Roland`。人工影像檢視保留
  正面略向左、灰棕長捲髮、寬深藍頭帶、狹長眼、淡紅唇色與小型金色耳飾；
  希莉亞現代母稿只作平滑手繪完成度與背景參考。
- `modern.sylpha.portrait.style_a.frame0`
- 執行期檔：`sylpha-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `ecd5b1f299bb9f0e1b1dc233a31b9fe5652d9bcac20eacad2935839fba424ccb`
- 母稿：`sylpha-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_024_m0`、10 份故事檔共 41 筆 `speaker=24 / speaker_name=希爾法` 與繁中實體清冊
  三方一致確認，識別字採重製英文實體目錄的 `Sylpha`。人工影像檢視保留
  完全禿頭、高額頭、兩側長尖耳、瘦削嚴肅面孔、細黑八字鬍、中央尖山羊鬍
  與金橙高聖職衣領；約拿現代母稿只作完成度與背景參考，希爾法不得帶入約拿的
  巨大白鬍鬚或貼頭白色頭巾。
- `modern.seymour.portrait.style_a.frame0`
- 執行期檔：`seymour-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `cda1be8693662eb478b9462d9b3a9e8239fb1c5b1164c9828c6ffd29fc868ff7`
- 母稿：`seymour-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_025_m0`、2 份故事檔共 5 筆 `speaker=25 / speaker_name=謝多` 與繁中
  實體清冊三方一致確認，識別字採重製英文實體目錄的 `Seymour`。2026-09-01
  依使用者角色設定勘誤確認謝多是豹人，不是鼠人；原版 `40×40` 頭像單靠像素
  輪廓不足以判定種族，先前把圓角耳、口鼻與亮色嘴部解讀為鼠耳、鼠吻及門牙
  是過度斷言，現已撤銷。修正版保留原版可見的深藍灰毛、向右三分之二側面、
  警覺淺色眼與深色忍者高領，改用小型貓耳、短寬貓科口鼻、強壯顴骨及低調豹斑；
  不再帶入大型圓鼠耳、細長鼠吻或門牙。
- `modern.saint-colas.portrait.style_a.frame0`
- 執行期檔：`saint-colas-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `4c1de0e53513751b81bf6ccf3dd632be04ac01ab5d86ebbf1e89dc6d073266ad`
- 母稿：`saint-colas-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_026_m0`、角色資料表的龍人／龍劍士欄位，以及四語隊伍姓名清冊共同確認
  為聖寇拉斯（`Saint Colas`）。人工影像檢視保留原版向左的深鋼藍龍首、狹長
  金色眼、後掠長角、頰部與頸部棘刺、外露尖牙及藍色護甲。故事資料中的
  `speaker=26` 尚存在跨場景身分混用，不能把其總筆數當成聖寇拉斯對話證據；
  正式載入仍依 catalog 的明確 `speaker_id` 契約失敗即關閉。
- `modern.banalosia.portrait.style_a.frame0`
- 執行期檔：`banalosia-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `ed12d32c99a2e1058f9cf2c20865a8bd4d0fc079d62b9ee7c2b8a13100b18167`
- 母稿：`banalosia-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_027_m0`、角色資料表的龍人／龍劍士欄位，以及四語隊伍姓名清冊共同確認
  為巴拿羅西亞（`Banalosia`）。人工影像檢視保留原版向左的橄欖綠龍首、狹長
  黃眼、頭頂與頸後淡色尖角、分層鱗片及外露尖牙；不從職業名稱臆造盔甲或飾品。
  身分標籤統計與展開後故事台詞筆數屬不同量尺，不得混稱。
- `modern.gaia.portrait.style_a.frame0`
- 執行期檔：`gaia-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `bc5a12b1e6e0d61dcdf0b48fee2061ad7ca7fba3f81f7bfaaa144257ae390398`
- 母稿：`gaia-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_030_m0`、角色資料表的機兵職業、四語隊伍姓名清冊及故事中的機能／導航
  敘述共同確認為蓋亞（`Gaia`）。人工影像檢視保留原版向右的非人機械頭、白灰
  分段裝甲、深色水平視窗及藍色關節／線路；種族欄位仍未解碼，不另加種族斷言。
- `modern.wood.portrait.style_a.frame0`
- 執行期檔：`wood-portrait-style-a-v1-80.png`，`80×80`，SHA-256
  `225455dd1e6f3678b1376889f546c110d41a46c48415476cf58425e07cf50805`
- 母稿：`wood-portrait-style-a-v1-master.png`，`1254×1254`；角色身分由
  `DATO_031_m0`、機兵職業、四語姓名及故事的系統／記憶庫／機甲戰鬥團敘述共同
  確認為渥德（`Wood`）。人工影像檢視保留原版向左的仿人機械臉、古銅色人造
  表皮、黑色硬質頭冠、藍黑側板、紅橙眼與強壯下顎；不把它改畫成蓋亞式全罩頭。
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
- 上述母稿與執行期檔先在忽略版控的本機工作目錄產生；通過 catalog 驗證後同步至
  私人庫 `wicanr2/fd2-assets-private` 的 `packs/fd2-modern-handpainted-a/`。
  公開儲存庫保存契約、雜湊與經使用者允許的總攬圖，不保存完整原版分離包。

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

1. 每個 `modern.fdicon.group_NNN.style_a` 必須一次提供 12 張獨立 `24×24` PNG，保留
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

### selector 0–21、68、76 與 77 候選

- `modern.fdicon.group_000.style_a`：以原版 `fig_000_f00..f11` 為動作基準生成
  3×4 母稿，再以確定性背景分割、逐格裁切及二值 alpha 轉成 12 張 `24×24`
  PNG。索爾的基礎建構資料使用 selector 0，因此這組可驗證第一關索爾；但
  catalog 仍以原始 selector 命名，不宣稱 group 0 永遠只供索爾使用。
- `modern.fdicon.group_001.style_a`：角色資料表 `index=1` 已證實哈諾的基礎
  `sprite_group=1`；以原版 `fig_001_f00..f11` 的四方向與三步態作姿勢基準，
  生成 3×4 母稿後，以連通背景分割、最大人物元件保留、最近鄰縮放及二值
  alpha 輸出 12 張 `24×24` PNG。這是現代美術候選，不冒稱原版逐像素重繪；
  catalog 仍按 selector 命名，避免未來 writer 改寫 selector 時把角色身分硬編進 loader。
- `modern.fdicon.group_002.style_a`：角色資料表 `index=2` 已證實鐵諾的基礎
  `sprite_group=2`。現代母稿依原版巨盾、直劍、灰綠兜帽與四方向遮擋關係生成，
  再由通用確定性工具完成背景分割、最大人物元件保留、最近鄰縮放與二值 alpha
  輸出；12 幀均保持獨立雜湊。
- `modern.fdicon.group_003.style_a`：角色資料表 `index=3` 已證實哈瓦特的基礎
  `sprite_group=3`。母稿保留原版由巨大紅棕頭髮／鬍鬚形成的圓厚輪廓、黑灰
  重甲、披肩與重型戰斧，再由同一通用工具輸出 12 幀。母稿原生透明通道由工具
  先轉為二值 alpha，不再套用棋盤背景分割。
- `modern.fdicon.group_004.style_a`：角色資料表 `index=4`、多章 `fig=4` 與
  `native_identity=4` 已證實亞雷斯的基礎 `sprite_group=4`。母稿保留原版巨大
  深藍軟帽、暗綠頭帶、藍色輕甲、直立長劍與小型圓盾，以及四方向各三步態的
  遮擋關係；同一通用工具由原生透明通道輸出 12 幀二值 alpha。
- `modern.fdicon.group_005.style_a`：角色資料表 `index=5`、多章 `fig=5` 與
  `native_identity=5` 已證實洛娜的基礎 `sprite_group=5`。母稿保留原版銀灰
  長髮、藍灰騎士裝甲、直立長劍，以及背面可見的紅棕披風；同一通用工具由
  原生透明通道輸出四方向各三步態。洛娜肖像只保留 `DATO_005_m0` 為身分來源，
  不再把亞雷斯的現代肖像列為角色來源。
- `modern.fdicon.group_006.style_a`：角色資料表 `index=6` 與多章角色身分資料
  已證實萊汀的基礎 `sprite_group=6`。母稿保留原版深棕束髮、深色頭帶、灰綠
  重甲、高聳銀色肩甲輪廓與貼身短直刃；同一通用工具由原生透明通道輸出四方向
  各三步態。萊汀肖像只保留 `DATO_006_m0` 為身分來源，不再把亞雷斯列為來源。
- `modern.fdicon.group_007.style_a`：角色表 `index=7 / sprite_group=7`、隊伍
  實體目錄與多章地圖角色資料共同支持這是蘭斯洛特的基礎地圖圖組投影。母稿
  保留原版巨大橙紅長髮、遮住口鼻的深藍高領巾與深色裝甲，不加入原版小圖未
  顯示的披風或武器；同一通用工具輸出四方向各三步態。這項證據只授權地圖
  sprite group 7；`ch03` 與 `ch18` 的 `speaker=7` 名稱衝突仍未閉合，因此不產生、
  不接通 `DATO_007` 現代頭像。
- `modern.fdicon.group_008.style_a`：角色表 `index=8 / sprite_group=8`、四語
  隊伍姓名、多章地圖角色資料與 `DATO_008 / speaker=8` 一致確認希莉亞。母稿
  依原版大片棕色波浪髮、藍色弓兵服裝、側／背面可見的樸素木弓與少量箭袋
  輪廓生成，不帶入 FIGANI 或對話頭像未在地圖小圖顯示的細節；通用工具輸出
  四方向各三步態。這只支持 group 8 的地圖投影，不能泛化成所有角色欄位全域等價。
- `modern.fdicon.group_009.style_a`：角色表 `index=9 / sprite_group=9`、四語
  隊伍姓名、多章地圖資料與 `DATO_009 / speaker=9` 一致確認悠妮。母稿依原版
  深紅長髮與窄頭帶、深藍上衣、紅色下身及纖細法師輪廓生成，不加入原版小圖
  未顯示的武器。這 12 幀只代表正常行走；初遇倒地昏迷仍由獨立場景 acting
  與專用演出素材負責，不能以本組宣稱已完成倒地演出。
- `modern.fdicon.group_010.style_a`：角色表 `index=10 / sprite_group=10`、
  四語隊伍姓名、多章地圖資料與 `DATO_010 / speaker=10` 一致確認瑪琳。母稿
  依原版粉紫短髮、金色額飾、紅色僧侶上衣與深色下身生成，不加入原版地圖
  小圖未顯示的武器；四方向各保留三個不同步態。
- `modern.fdicon.group_011.style_a`：角色表 `index=11 / sprite_group=11`、
  四語隊伍姓名、多章地圖資料與 `DATO_011 / speaker=11` 一致確認索菲亞。母稿
  依原版灰綠短髮與長鬢髮、紅色頭飾、深色上衣、紫色袖套、紅色腰飾與短金色
  法杖生成；法杖依方向與步態自然遮擋，不從頭像或戰鬥動畫補入其他裝備。
- `modern.fdicon.group_012.style_a`：角色表 `index=12 / sprite_group=12`、
  四語隊伍姓名、多章地圖資料與 `DATO_012 / speaker=12` 一致確認凱麗。母稿
  依原版巨大橙金側馬尾、深藍武者服、藍灰腰帶與淺灰綠護腕生成；逐幀檢視
  沒有可證實武器，橙色長輪廓是頭髮，不加入刀劍或長柄武器。
- `modern.fdicon.group_013.style_a`：角色表 `index=13 / sprite_group=13`、
  四語隊伍姓名、多章地圖資料與 `DATO_013 / speaker=13` 一致確認貝克威。母稿
  依原版巨大金色長直髮、黑色寬頭帶、深綠弓兵服、樸素棕弓與箭袋生成；原表
  種族仍標「精靈？」且沒有定論，因此不把精靈尖耳加入現代稿或規格斷言。
- `modern.fdicon.group_014.style_a`：角色表 `index=14 / sprite_group=14`、
  四語隊伍姓名、多章地圖資料與 `DATO_014 / speaker=14` 一致確認珊。母稿
  依原版巨大深藍不對稱長髮、綠色額帶與額側飾物、深藍法師袍、棕橙肩領及
  紫紅腰胸飾生成；小圖沒有可證實武器，也看不出尖耳。原表種族仍標
  「精靈？」，因此不把問號或精靈特徵寫成定論。
- `modern.fdicon.group_015.style_a`：角色表 `index=15 / sprite_group=15` 與
  多章 roster 支持同一地圖投影；原版 12 幀只能證實封閉式深藍頭盔與甲胄、
  狹窄黃色眼縫、紅色腰腳點綴及無可見武器。角色表／故事使用「賽可邦勒」，
  四語隊伍目錄繁中則是「塞可邦勒」，且 `speaker=15` 頭像仍在既有衝突停止線，
  因此本組只採中性 selector 身分，不用頭像補臉、種族、髮型或武器，也不宣稱
  canonical 人名已定案。
- `modern.fdicon.group_016.style_a`：角色表 `index=16 / sprite_group=16`、
  四語隊伍姓名、多章 roster 與 `DATO_016 / speaker=16` 一致確認凱拉斯。
  母稿依原版 12 幀可見的棕色龍首、深藍甲胄、紅色腰飾、藍灰圓盾及紅白長刃
  生成；頭像只協助確認長吻、綠眼與後掠角的角色識別，不把頭像細節冒充地圖
  小圖逐像素證據，也不從其他素材追加翅膀或第二件武器。
- `modern.fdicon.group_017.style_a`：角色表 `index=17 / sprite_group=17`、
  四語隊伍姓名、多章 roster 與 `DATO_017 / speaker=17` 一致確認米亞斯多德。
  母稿依原版 12 幀可見的暗紅龍首與身體、灰綠圓盾、深色三角盾紋及藍白長刃
  生成；頭像只協助確認紅眼與多角龍人身分，不把尖耳、鬚棘等頭像細節冒充
  地圖小圖逐像素證據，也不追加翅膀或第二件武器。
- `modern.fdicon.group_018.style_a`：角色表 `index=18 / sprite_group=18`、
  四語隊伍姓名、多章 roster 與 `DATO_018 / speaker=18` 一致確認蜜蒂。母稿依
  原版 12 幀可見的蓬鬆金橙長髮、紅色長袖外衣、淺灰胸甲、藍灰腰部與深色下裝
  生成；原版地圖小圖沒有足以可靠辨認的武器，因此即使職業表記為劍聖，也不
  從職業名稱猜加長劍。頭像只協助角色識別，不把臉部細節冒充地圖逐像素證據。
- `modern.fdicon.group_019.style_a`：角色表 `index=19 / sprite_group=19`、
  四語隊伍姓名、多章 roster 與 `DATO_019 / speaker=19` 一致確認羅德曼。母稿
  依原版 12 幀清楚可見的棕色長髮與鬍鬚、藍灰甲胄、深色十字圓盾及短直銀劍
  生成；頭像只協助確認成熟男性角色識別，不把臉部角度或甲胄細節冒充地圖
  小圖逐像素證據，也不追加頭盔、披風或雙手大劍。
- `modern.fdicon.group_020.style_a`：角色表 `index=20 / sprite_group=20`、
  四語隊伍姓名、多章 roster 與 `DATO_020 / speaker=20` 一致確認莎拉。母稿依
  原版 12 幀可見的深綠長髮、暗灰綠衣甲與紅色腰肩點綴生成；小圖手部物件
  無法可靠辨識，因此沒有依龍騎士職業名稱追加長槍、龍、翅膀或坐騎。
- `modern.fdicon.group_021.style_a`：角色表 `index=21 / sprite_group=21`、
  四語隊伍姓名、多章 roster 與 `DATO_021 / speaker=21` 一致確認約拿。母稿依
  原版 12 幀清楚可見的蓬鬆白髮、覆蓋胸肩的巨大白鬍鬚與樸素棕袍生成；不再
  帶入已撤銷的狼族、年輕法師或尖帽造型，也不追加法杖、劍或盾。
- `modern.fdicon.group_068.style_a`：沿用第 68 組原版輪廓的現代候選。它不綁
  固定角色身分；第 11 格是第 9 格上半身與第 10 格下半身的現代近似。
- `modern.fdicon.group_076.style_a`：第三章實際敵軍資料的普通追兵使用
  `fig／map_selector_key／battle_fig=76`；隊長則為 77，兩者均不是角色表的
  蘭斯洛特 group 7，也不是第十八章才登場的約拿 speaker 21。現代母稿依原版
  `fig_076_f00..f11` 保留封閉式深鋼盔、低額甲、護頰、頸部防護、小盾、短劍與
  制式軍裝。依使用者確認，臉部不是視覺焦點，不加入裸頭、英雄披風或華麗裝甲；
  這是士兵職能的現代近似，不冒稱原版逐像素重繪。
- `modern.fdicon.group_077.style_a`：第三章 `map2_units.json` 的敵方隊長在同一筆
  單位記錄使用 `fig／portrait／map_selector_key／battle_fig=77`，故事也由
  `speaker=77` 消費；原始名稱只有單字「約」，不足以證實人物姓名。現代母稿只依
  `fig_077_f00..f11` 可見的全罩深灰綠鋼盔、厚重板甲、大型圓盾、直劍與背面暗紅
  腰帶建立，採中性「第三章敵方重裝隊長」身分，不與約拿 speaker 21、蘭斯洛特
  group 7 或普通追兵 group 76 合併。
- 二十五組都具有 12 個不同 SHA-256、`24×24`、二值 alpha 與三個不同週期，列為
  `runtime_candidate`。私人母稿與逐格 PNG 不進公開 Git，公開 catalog 只保留
  可重現契約與雜湊。

### 地圖人物 consumer 現況

- `loadModernStoryPortraitSet` 現會同時預檢 catalog 中的地圖人物組：group
  0..95、12 個安全檔名、逐格 SHA-256、`24×24`、二值 alpha、互異雜湊及三週期
  policy，任何一項不符即拒絕整個現代主題。
- `loadGame` 在完整預檢後，才以各組 12 張真彩色圖原子取代正規化
  `g.sprites[0]`／`g.sprites[1]`／`g.sprites[2]`／`g.sprites[3]`／`g.sprites[4]`／`g.sprites[5]`／`g.sprites[6]`／`g.sprites[7]`／`g.sprites[8]`／`g.sprites[9]`／`g.sprites[10]`／`g.sprites[11]`／`g.sprites[12]`／`g.sprites[13]`／`g.sprites[14]`／`g.sprites[15]`／`g.sprites[16]`／`g.sprites[17]`／`g.sprites[18]`／`g.sprites[19]`／`g.sprites[20]`／`g.sprites[21]`／`g.sprites[68]`／`g.sprites[76]`／`g.sprites[77]`；其他 group 不變，忠實主題
  預設路徑也不變。
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
