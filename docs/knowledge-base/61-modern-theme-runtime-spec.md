# 61 — 現代美術主題正式執行期規格

狀態：現代主題仍為部分覆蓋；目前分層與缺口依[58](58-fd2-exe-re-coverage.md)。

2026-10-15 接續現況：使用者已要求加速HD化；本檔既有手繪風格、靜態嘴型、幾何與原子預檢契約保持。
清冊與私人檔檢查通過，不代表完整玩家路徑或高解析呈現完成；唯一數量與缺口見[58目前狀態](58-fd2-exe-re-coverage.md)。
[#4](https://github.com/wicanr2/fd2_re/issues/4)為總項，
[#201](https://github.com/wicanr2/fd2_re/issues/201)盤點正式consumer素材覆蓋，
[#202](https://github.com/wicanr2/fd2_re/issues/202)驗證正常路徑，
[#204](https://github.com/wicanr2/fd2_re/issues/204)追蹤高解析範圍與原型。
正式路徑已接480×480故事頭像、高解析黑體及144×144戰場人物；下方各切片分別記錄驗證範圍。
真正HD地形、其他完整演出與平台仍待完成。新增範圍先經確認與READY審查。

素材缺口先用[靜態宣告盤點工具](../../tools/audit_modern_theme_coverage.py)產生候選來源與JSON pointer，
再查正式consumer或正常玩家路徑。工具用法見檔首；它包含可能不可達的故事資料，零缺件也不代表完整玩家覆蓋。
負speaker宣告另列，不直接視為缺頭像。此次靜態候選、來源雜湊及18項私人素材回歸結果見
[工具與素材主收據](../data/fd2-remake-toolchain-recovery-20261015.json)；尚未改變catalog的候選狀態。

## 範圍

### 2026-10-19 原始圖庫範圍勘誤（已證實，#259）

第一章既有JSON存檔的具型別名冊需要selector96，HD世界候選拒收`missing group 96 frame 0`。
回查固定`FDICON.B24`及既有`fdicon.SeparatedSpriteCount`：實際header為1680幀／140組，
不是96組。先前94組／1083張只涵蓋selector0..95的可繪子集；其資料驗證仍有效，
不能當成全部原版圖庫覆蓋。這項勘誤沒有重開原版函式或matching。

[圖庫盤點工具](../../tools/audit_modern_fdicon_bank.py)在Docker執行，接受`--original <FDICON.B24>`、
`--bank <分離bank.json>`、`--catalog <catalog.json>`、`--reference <固定版本清單>`及`--report <新檔>`。
它核對原始size／MD5／SHA、header與全部分離mask，輸出未登錄可見selector及逐層雜湊；
`--palette <FDOTHER#0 JSON> --render-selector 96`另產生明示的原版彩色參照，不能標成執行期擷圖。
完整數量、輸入指紋及來源見[58目前狀態](58-fd2-exe-re-coverage.md)與
[HD戰場主收據](../data/fd2-modern-hd-battlefield-20261019.json)的`proportional_world_followup_20261019`。

[#259](https://github.com/wicanr2/fd2_re/issues/259)追蹤漏列圖組、既有24與新增HD載入範圍。
先補第一章實際需求，再核對其他名冊；不猜代96、不略過敵人、不以像素替代讓候選通過。
上述拒收時只有94組loader；其後由下節補齊125組，正式世界另見本檔「正式HD戰場接線」。

#### 完整原生圖庫的清冊、父項與HD載入（CONFORMED，限資料與既有呈現）

沿上節已證實的固定B24指紋、1680幀header、全部分離Mask與正式selector consumer，
修正資料容量假設，不改玩法、名冊、selector cache、方向、座標或保存格式。
可見圖組名單只取原始Mask盤點；無來源寫入的15組不造人物，raw單位pass保持既有無寫入行為。

- Go以既有`fdicon.SeparatedSpriteCount/12`定義原生圖組上界；可見資格另由固定圖庫盤點定義，
  並逐組對照真實分離Mask回歸。Python清冊工具使用同一已證實名單，另以完整盤點驗證。
- 沒有HD人物的舊清冊仍可載入既有子集；新24人物可登錄實際可見的96..139圖組。
  只要有`map_sprite_hd_set`，必須完整覆蓋125組及同份125組24父項，排除全部零來源寫入組。
  任一缺組、重複、孤兒、越界、錯誤週期／hash／腳底，均原子拒收。不能以95組局部清冊放行首章。
- 新父項只從已審手繪母稿完整輪廓產生。沿已核准alpha核心提取、22像素等比例方格、水平置中與
  腳底23；24輸出使用最近鄰與alpha128二值門檻，HD輸出保留原覆蓋alpha。不得固定格線裁切。
  既有94組24成品bytes保持；新31組的12幀皆有原始來源，不重用未證實的別組或新增身分語意。
- 清冊schema維持1；HD角色使用既有已核准role／consumer_contract／layout_policy，另以
  `master_file`／`frame_overrides`保留已驗的來源選擇，`source_files`與`source_sha256`逐項保留
  實際採用的母稿及單幀來源。來源檔名安全、互異且hash相符；
  來源只供完整私人包重生，runtime-only包只需正式逐幀PNG。拒收草稿與原版參照不列allow-list。
- 先產生可丟棄完整父項清冊與候選包，再編製／驗證125組HD資料與全姿態；完整資料預檢及
  第一章同份存檔候選成功後才登錄正式清冊。已登錄來源與逐幀檔沿現有allow-list工具同步私人庫。
  這項資料接入只驗載入；後續Game.Draw驗收見「正式HD戰場接線」，完整HD地形仍另列。
- 回歸覆蓋真實140組Mask分類、125組完整預檢、96與最後可見135的有效載入、15組空來源及140越界、
  缺組／孤兒／重複／父項錯誤／來源hash拒收、舊清冊相容、原始保存不變與第一章候選。
  正式現代主題另抽正常F2／Home／保存冷讀；其餘章節與完整HD世界仍保持未驗分級。

[完整清冊候選工具](../../tools/prepare_modern_hd_map_catalog.py)只在Docker執行。
`prepare`接受`--catalog`、`--prior-manifest`、`--native-audit`、`--new-masters`、`--source`及新`--output`，
建立完整24父項候選包、選定母稿／單幀映射及輸入指紋；`--workers 1..4`限制編製程序。
再用既有HD編製器及獨立驗證器產生完整HD候選。`register`另要求`--parent-catalog`、`--manifest`、
`--world-receipt`、`--art-review`、`--source`與`--destination`，預檢正式清冊未被其他工作修改、既有成品bytes保持，
且同份完整第一章候選與原始保存檢查通過，最後才原子登錄。工具不啟用Game.Draw或發布Release。

125組／1455張HD及31組／372張父項已登錄，新增372幀逐姿態檢閱通過。
舊94組的1083張父項與HD輸出bytes相同；完整原生Mask分類、清冊與九項編製守衛通過。
首章實檔CPU候選通過；正式清冊與私人包另取得16張普通GUI、F2／Home、保存及新程序LOAD，
兩主題與來源保存bytes相同。CONFORMED只限本段資料、載入與既有24呈現；當時144世界尚未接線，
後續正式接線由本檔獨立切片及主收據的`world_runtime_followup_20261019`驗收。
來源、重跑命令與限制見[HD戰場主收據](../data/fd2-modern-hd-battlefield-20261019.json)的`native_coverage_followup_20261019`。

### 2026-10-19 F2戰場重複消費修正（CONFORMED，限首章抽驗）

F2已是核准的全域主題切換，Home為正常角色檢視。正式首章LOAD、Z聚焦後F2的
實際PNG與兩個KeyF2讀取端確認：同一鍵也開啟角色狀態，前測圖片因此不能證明取景比較。
修正只從中立戰場角色檢視移除F2舊alias，保留Home；全域F2、修飾鍵、序章／設施、
動畫推進、選取、角色／view與保存不變。正常F2往返須維持player_status_phase空且GPU仍是
戰場，之後Home才開啟狀態；並驗兩角色、縮放／說明、F5／F9與新程序標題LOAD。
來源沿現行Go及字型GUI工具；不新增原版RE或parity聲明。


正常首章16張GPU圖、兩次F2不開狀態、Home檢視索爾／悠妮、縮放／說明、保存及新程序LOAD通過。
完整保存與來源SHA相同；相關cmd16.550秒及清冊bytes保持。來源見
[HD戰場主收據](../data/fd2-modern-hd-battlefield-20261019.json)。#257已依來源提交結案。
真正雙主題戰場比較也確認取景倍率不同，#258另列，不能用F2修正宣稱已完成HD世界。

### 2026-10-19 母稿透明度核心分離（CONFORMED，僅候選編製）

三份新RGBA母稿以alpha>0連通時拒收；實檔分級64／128／192在三組均得到12人物，
證實低覆蓋像素橋接了各人物，不能以此宣稱實際身體重疊。來源、完整histogram與輸入SHA
見HD戰場主收據的redraw_batch_3_10_12，原始PNG保持不可變。

- alpha≥128只作四鄰接實例拓樸判定，排序及拒收條件保持，不把此二值mask當輸出alpha。
- 對每個核心保留原RGB／alpha；只在核心外3來源像素內取回原覆蓋邊緣，排除非該實例的淡橋接。
  無partial alpha的舊來源沿原編製法，應逐檔保持既有PNG bytes。原24墨跡矩形及腳底不變。
- 驗新3／10／12各12幀與原0／4／9／52等48張binary來源，前者alpha保留並各自完整，後者bytes相同。
  靜態與重複政策另維持。方向仍須人工審，group10 frame5不得因分離成功而放行。
- 本契約僅適用候選編製，不授權正式HD世界或不完整catalog，其他主線規則與存檔不變。

r6全94組1083幀已通過獨立資料驗證，48張舊binary-alpha樣本的PNG bytes保持；
單幀替換、canonical重複相位的來源傳遞及六項拒收守衛通過。CONFORMED只限本段候選工具，
不包含美術或正常玩家路徑；版本、命令輸入與雜湊沿主收據的alpha_core_followup_20261019。

#### 單幀來源替換（READY，僅候選編製）

group10 frame5的實際候選面向右，與該列左向契約矛盾。單獨補繪左向步態後，
候選工具可接收明示的group／frame／安全來源檔名映射，不覆寫整張母稿。
只接受清冊內圖組及有效幀，單幀來源須能辨認唯一完整人物；來源SHA、裁切範圍與
替換索引寫入manifest。其他幀須逐檔保持，重複相位不得另設衝突替換。
驗收先核對方向、角色與步態，再驗144尺寸、覆蓋alpha、原墨跡矩形及未替換幀的bytes。
未通過人工審查仍不能登錄正式清冊，也不改變遊戲規則或保存。

[候選驗證器](../../tools/verify_modern_hd_map_sprites.py)在Docker執行，接受
`--manifest <hd-map-sprites.json> --catalog <catalog.json> --source <母稿目錄>`，
檢查目前編製器／清冊與逐幀來源SHA、尺寸、原墨跡定位及重複政策；預設要求全94組，
有界樣本須明示`--allow-subset`。只產生資料驗證收據，不替代人工姿態審查或正常GUI。

### 2026-10-19 144×144 戰場角色與世界層（較早DRAFT與候選證據）

本節保留候選形成與94組子集的歷史限制；現行125組及Game.Draw分別依本檔完整圖庫與正式HD戰場接線。

沿使用者已核准的1920×1200、手繪風格、原方格與人物位置，從既有母稿編製144×144圖。
[候選編製器](../../tools/build_modern_hd_map_sprites.py)保留24×24已審墨跡矩形及腳底，
色彩與細節直接取自母稿；12幀行走、3相靜態及明示重複幀保持。這是技術編製，
不重新決定畫風，不把尺寸變大當作細節完成。group68的母稿尚未登錄，候選明示override。

目前已證實現代正規化世界以24邏輯像素直繪，忠實原生世界則由320×200放大兩倍；
同一NativeMapViewState的取景與人物大小因此不同。原生世界來源為
`buildNativeMapFrameInput`、`fdicon.NativePlacementOffset`、`NativeFrameIndex`及
`indexedmap.ComposeFrame`，原版雜湊、位址基準與證據沿既有固定清單及原生地圖契約。
新增HD層不得重猜方向、可見名冊、移動偏移、selector與前景遮擋。

擬採獨立1920×1200世界層及透明640×400介面層，在最終螢幕先貼世界、再貼介面與HD文字。
世界使用原生(4,4)、312×192窗格及同份具型別狀態；忠實路徑保持，早返演出清除本幀世界。
新增`map_sprite_hd_set`只附加144圖源，完整94組原子預檢，24圖與存檔格式保持。
首輪94組1083張雖通過檔案與alpha檢查，固定格線會裁切，已拒絕登錄；總覽不是全部姿態驗收。
編製器改以整張母稿的連通輪廓排序，保留來源alpha並拒收不明額外輪廓；五組51張完整樣本已審。
靜態48另由內建image_gen沿核准風格重繪，三相與原銀白機械身份保持，原檔不覆寫。
新圖與其餘像素化母稿仍為候選，不能用144尺寸宣稱細節齊備。來源及下一閘門見同一HD戰場主收據。
r3歷史audit為84組963張通過、10組拒收。新透明度核心及明示來源替換後，r6全94組1083幀
通過來源、幾何、覆蓋alpha與相位驗證。22份新母稿供28組明示重用，另有3份單幀；
全組候選已掃描，29組339張替換幀已重審，但完整美術驗收尚未通過，仍不進catalog。
第19組單幀只解決朝向，仍像素化；部分舊母稿頭身比例待校正。第37組花形盾徽已拒收並
另製十字修正版；第19組持盾側錯誤的整張新稿也拒收，沒有因解析度高就採用。
r6不等比填滿舊墨跡框曾拉伸人物，第76組中立左向頭盔尤其明顯。
後續等比例r9已通過既有94組資料驗證及姿態掃描；新母稿與來源沿主收據。
真正第一章候選另因selector96拒收，見本檔範圍勘誤；資料PASS仍不推定完整美術或HD世界READY。

#### 保留來源比例的HD編製（READY，僅候選工具）

六個可丟棄樣本比較舊填框、框內等比例及22像素方格內等比例；第76組左向中立頭盔
在舊填框被拉寬。追查既有`process_modern_map_sprite_sheet.py`確認原24成品本來就以
`min(22/width,22/height)`保持比例，並置中、將腳底對齊23；它不是原版FDICON墨跡。
r6的`native_ink`因此只是已登錄現代24成品的輪廓，不能解讀為原版位元組的權威邊界。
樣本與來源見主收據的後續比例紀錄；r6收據保留，不重寫歷史。

- 新候選沿同一22像素方格契約，最大132×132，整張144×144；只等比例縮放完整母稿人物。
- 水平置中於144方格，腳底沿已登錄24成品的alpha下界乘6；可用高度不得超過該腳底座標，
  不越出畫布。整數尺寸採最近整數，半數進位，最小1像素；不量化原覆蓋alpha。
- manifest明示`contain_22px_cell_feet_v1`、`registered_ink`及實際`fit_rect`。
  原24 PNG、selector／frame／cycle、圖格、世界位置與保存不變。重複幀連同fit來源傳遞。
- 獨立驗證來源比例的整數誤差、最大尺寸、置中、腳底、實際alpha範圍及重複相位。
  全94組重編後重審完整方向，尤其第76組中立幀、靜態48與特殊馬形73。
- 本項修正候選編製拉伸，不將既有二值來源bytes保持要求套到新的縮放政策。
  透明度核心提取仍沿上一節；造型與正式HD世界的驗收保持分開。

#### HD人物清冊載入（READY，世界繪製與美術驗收另列）

本節只覆蓋既有selector0..95清冊子集；#259已證實實際圖庫更大，尚未擴充正式範圍。
新增`map_sprite_hd_set`附加圖源，既有24人物與正式保存格式不改。
12幀契約為`fdicon_hd_map_sprite_12x144_v1`，3相契約為
`fdicon_hd_static_sprite_3x144_v1`；尺寸144×144、`alpha_contract: coverage`，
`layout_policy: contain_22px_cell_feet_v1`，逐幀PNG檔名及SHA安全檢查。
每幀須具有透明背景及不透明核心，柔邊保留；空白、全幅不透明背景或僅半透明圖拒收。
只要清冊有HD人物，就必須完整94組，排除已證實無來源寫入的39／57；同份清冊也須有
完整對應24組。逐組frame_count、cycle_policy及全部重複幀等價關係須與24父項相同。
重複圖組、缺件、錯誤幾何、來源雜湊或週期均原子拒收整個主題，不部分發布。
沒有HD人物的既有清冊仍沿原路徑；本項只載入圖源，不能自行啟用DRAFT世界層。
驗證完整清冊、缺組／重複、父項不符、錯誤來源、透明度與相位，並跑既有現代主題回歸。

#### HD單位的原生座標計畫（READY，尚未啟用世界層）

輸入只沿`indexedmap.FrameInput`的UnitBank、SelectorCache、raw Units、相機、週期與pixelShift，
及該caller的viewport來源位移。定位與可見閘門沿`fdicon.BlitNativeUnitLayerWithBaseShift`、
`NativeFrameIndex`和`NativePlacementOffset`既有已證實契約；原版雜湊與位址基準沿固定清單，
不重新推定u.Dir、人物ID或已行動狀態。正式世界啟用仍依上節DRAFT與美術／GUI閘門。

- 使用原版cache slot取得group，按pose×3加原週期選幀，forceBase、cycle3→1及bit7原樣保留。
- 原始work位移先通過同一非負及工作緩衝邊界；以絕對位移和viewport base各自除／餘456，
  兩者差值才是座標。不能對兩者差值直接取餘數，否則負向邊界會錯行。
- HD目的矩形為原生viewport的(4,4)加相對座標，乘6；尺寸144，裁於312×192 viewport乘6。
  caller可提供已證實的上移viewport base，不改世界或單位座標。
- 先檢原版選中Sprite.Mask；沒有來源寫入的39／57與靜態未用方向不生成HD人物。
  不把RemapMask當人物輪廓。有效來源卻缺HD group／frame時原子拒收計畫。
- 對比既有原生單位pass的實際work寫入，涵蓋四方向、步長0..6、相機四邊、forceBase、
  週期、cache順序、純重映射／靜態保留幀與拒收；這是重製端座標轉接驗證，不新增oracle或E2。
#### HD前景遮擋計畫（READY，尚未啟用世界層）

沿`fdicon.BlitNativeForegroundLayer`的資格、排程、相機／地圖邊界、control及frame selector，
把同份原版Sprite的來源寫入轉成透明RGBA前景。只以Mask決定覆蓋，包含palette index 0；
不用前後畫面差分判定遮擋，因為相同顏色的寫入仍會遮住HD人物。
`BlitMode==0xff`沿原色，其餘沿同份LUT轉換來源；`BlitLUTTransparentAtOffset`已證實
保留mode-3目的地，故RemapMask不產生前景覆蓋。座標沿上一節絕對work位移與viewport差值，
只在最終畫布裁切。任何缺bank、control、LUT、幾何或palette均原子拒收。
以既有native foreground blitter的實際寫入驗證順序、移動鄰格、負邊界、資格、flip、
LUT、零色來源及透明洞。這項只增加繪製計畫，不新增地形美術或啟用正式HD世界。

#### HD世界候選合成器（READY，限較早CPU候選）

此有界切片只驗正常穩定世界的地形／範圍、HD單位與前景排序。使用同份FrameInput與
明示原生work初始快照；複製cells後讓既有ComposeFrame合成地形／範圍，單位、前景、
SelectionOverlay與HUD由本切片分開。沿原生312×192取景乘6，外圍四像素邊界保持黑色。
HD單位依上一節計畫與名冊順序合成，再貼來源Mask形成的前景。兩個計畫及全部144圖源
先預檢，錯誤不發布部分畫面。原始cells、work、銀行與HD圖源保持不變。
raw bit7沿既有現代主題的變暗係數`0.55／0.55／0.6`呈現，不宣稱原版palette-band逐色一致。
39／57在本raw單位pass只有mode-3透明跨度，沿已證實的`BlitForNativeFlagsAtOffset`
保持無來源寫入；不把它誤套成地形BlitLUT的目的地重映射。
地形與前景仍是原生素材放大，本項不聲稱完成HD地形。UI、HUD、走行caller、橫幅、
戰鬥返回及Game.Draw啟用仍由上層DRAFT追蹤；候選合成圖只列重製端驗證。

正式READY前須完成全部來源／透明度與姿態審查、純目的地重映射的保留方法，並界定正常世界、
範圍、前景、HUD與戰鬥返回。尚未接正式runtime，不宣稱整體HD完成。

[HD圖源載入](../../remake/cmd/fd2/modern_hd_map_sprite_assets.go)、
[單位計畫](../../remake/cmd/fd2/modern_hd_map_unit_plan.go)、
[前景計畫](../../remake/cmd/fd2/modern_hd_map_foreground_plan.go)與
[候選世界合成器](../../remake/cmd/fd2/modern_hd_map_world.go)已依上述READY契約實作。
重製端回歸涵蓋1568組單位定位與96組前景寫入比較、透明洞／零色來源及原子拒收，
既有現代主題回歸通過；不宣稱完整cmd或正常GPU驗收。

[候選工具守衛](../../tools/test_modern_hd_map_sprites.py)使用`--catalog`、`--source`、`--report`在Docker執行，
驗序列／平行bytes、canonical重複來源與比例等九項守衛。
[實檔候選測試](../../remake/cmd/fd2/modern_hd_map_candidate_test.go)由
`FD2_HD_CANDIDATE_DIR`、`FD2_HD_CANDIDATE_SOURCE`、`FD2_HD_WORLD_SAVE`與`FD2_HD_WORLD_REPORT`
明示本機輸入／輸出，再於既有Go測試容器執行`go test ./cmd/fd2 -count=1 -run TestModernHDMapActualCandidateWorld`。
尚未登錄的完整父項清冊另以`FD2_HD_PARENT_CATALOG`明示，不覆寫正式清冊來測候選。
此次94組實檔預檢後，第一章世界因96拒收，失敗與來源快照保留；標準回歸略過此外部候選測試。
下一閘門是#259素材／清冊範圍，再接正式世界、透明介面與正常玩家GPU。

### 2026-10-19 戰鬥後JSON LOAD暫態（CONFORMED-SCOPED，#260）

正常標題LOAD及普通攻擊後，`hd-world-gui-r5`觀測到F9已載入新的非原生章首對話，
但`nativeLevelUpDialogue`與`nativeMovePlan`仍在。`Update`先呼叫經驗對話owner，
`nativeStoryDialogueAtInputWait`拒絕非原生章首對話，輸入無法到達普通對話。
這是已證實的重製端生命週期缺陷，不推論原版ABI。

成功JSON LOAD在存檔欄位驗證後、發布新節點前丟棄舊經驗對話、待播佇列與移動計畫／面板，
不得執行舊owner的續行。拒收存檔保留目前owner。驗收包含修正前失敗的成功LOAD案例、
拒收案例，以及普通物理戰鬥後F9、章首對話、保存與新程序LOAD。
存檔格式、原生槽與正式經驗／死亡規則保持；這個窄修正不宣稱完整暫態覆蓋。

### 2026-10-19 正式HD戰場接線（CONFORMED-SCOPED）

沿已確認1920×1200、144人物、原方格與位置、F2及透明介面方案。完整125組來源已通過上一節，
不新增畫風、玩法、保存或發行決策。原版指紋沿固定清單；本節是現有重製消費端的已證實接線。
來源為`composeNativeMapFrameAtWithPaletteCycle`、`buildNativeMapFrameInput`、
`indexedmap.ComposeNativeFrame`／`ComposeNativeStepFrame`／`ComposeNativeUpStepFrame`、
`nativeMapSelectionOverlay`、`BlitNativeMapHUD`與`MosaicNativeMapViewport`。
原版位址、遮罩與caller證據沿上述現有契約及本檔HD單位／前景計畫，不重開matching。

- 正式HD戰場共用一次原生幀交易的candidate state、時鐘、DAC與FrameInput。
  HD預檢失敗不得發布新原生work／VGA、時鐘、狀態或部分HD幀。忠實及無HD舊包沿原路徑。
- HD獨立保存只含地形的work種子；不得把上一幀原版人物、range、HUD或選取像素當地形來源。
  地形、parallax、chapter auxiliary與第23章staging沿原生blitter及相同caller；原生work另保持原本語意。
  普通幀保留range，走行／橫幅不進range或HUD；第12章上移沿既有9列地形及viewport位移。
- 地形原生320×200來源乘6；144人物依原生計畫直繪，前景依Mask含零色來源，順序保持。
  原地形放大只是過渡呈現，真正HD地形仍未完成。raw bit7沿已核准現代變暗係數。
- Selection／HUD均已證實為來源覆寫或透明跳過。用兩個不同的完整初始填值執行同一純寫入callback：
  兩結果相同者為覆寫，兩者各保留原填值者透明，其餘拒收destination-dependent操作。
  這是覆寫契約的覆蓋證明，不能用原畫面前後色差、零色或黑色推定遮罩。
  UI以320原生位置乘2貼透明640介面；全畫面面板、說明、對話、淡幕及HD字型沿既有繪製順序。
- 現代指令圖示沿原`ActionOverlayOrigin`與各階段offset乘6貼世界，之後重繪游標上的HD人物，
  HUD及後續不透明UI保持在上層。輸入、blink、Draw承認與availability保持。
- 回合橫幅凍結進場HD世界。馬賽克原生block乘6、從viewport左上取樣；字樣沿原glyph／位置在UI。
  真彩人物無原版palette index，橫幅真彩減亮僅列現代呈現近似，不稱原版逐色parity。
- 每次Draw清除可見世界引用；早返、標題、設施、戰鬥演出、完整DAC黑白及LOAD不能露出舊世界。
  最終螢幕先世界、再透明介面、HD頭像與文字；邏輯截圖亦須包含同一世界。
- 驗收比較原生幀／時鐘／保存不變、普通及兩種走行、遮罩同色／零色、缺件原子拒收、
  F2、選取、指令環、走行／取消、HUD、角色／戰鬥返回及普通GPU。完整HD地形、其他演出及平台另列。

正式實作入口：[HD世界幀與UI](../../remake/cmd/fd2/modern_hd_map_runtime.go)、
[HD世界GPU](../../remake/cmd/fd2/modern_hd_map_gpu.go)；正常首章34張GPU與4張走行通過；兩次普通攻擊分驗經驗對話中F9及正常返回，保存與新程序LOAD bytes保持。
兩份穩定144來源各10678個核心像素零差異，一份跨相位明示不可比；同view的960地形樣本零差異。
範圍限本節與[主收據](../data/fd2-modern-hd-battlefield-20261019.json)的`world_runtime_followup_20261019`，不宣稱完整HD或原版對拍。
本輪GPU限首章普通走行與物理攻擊。第12章上移caller限CPU回歸，回合橫幅限shader編譯；
正常END橫幅、AI、其他章節及故事世界仍沿#202／#204後續抽驗，不能由這34張圖提升分級。
[GPU圖源驗證](../../tools/verify_fd2_hd_world_images.py)在Docker以`--receipt`、`--catalog`、`--source`及
新`--report`核對預先指定樣本的144不透明核心、前後相位、原生位移與雙主題同view；不挑選碰巧相同的動畫幀。

### 角色／物品面板轉場字型（CONFORMED，限已列GUI與fixture範圍）

延伸已核准的狀態、命令與物品字型。原版定位沿用`0x17AED`／`0x18409`，
位址基準及FD2.EXE雜湊沿[固定清單](../data/fd2-reference-files.json)。
既有具型別來源為[面板區域計畫](../../remake/internal/battle/native_item_selector.go)、
[角色合成器](../../remake/cmd/fd2/native_church_status_ui.go)與
[物品繪製](../../remake/cmd/fd2/native_item_panel_ui.go)，不重開RE或新增原版逐幀聲明。

- 教會／戰場狀態開收框與狀態到命令的下框移動，在建立動畫時保存獨立來源與角色資料快照。
  原12／14張索引快取與Draw承認保持，正式規則及保存格式不變。
- 物品及商店裝備沿已發布的raw slot快照，Draw各階段合成同份HD文字來源；不在Update包裝字型。
- 像素及文字使用同一份NativeItemPanelRegion。文字先裁來源矩形，再按來源到目的的差值移動，
  最後裁目的矩形。遮蔽只影響實際目的區域，保留面板外文字，不能把原位置文字留在移動畫面上。
- 原生模式仍使用索引快取；無效快照或區域退回原畫格。中途切換主題不推進動畫、改寫選取或重做交易。
- 回歸涵蓋全12／14階段、上下／左右裁切、原索引相關結果、快照獨立性、原生回退及一次性延續點。
  普通START後F9接續既有旅館槽，經教會狀態／命令與商店裝備取得GPU移動文字樣本，
  返回旅館保存並冷讀；CPU只定位候選，不能代替實際圖或宣稱精確同幀。不錄影。

實作見[面板移動文字](../../remake/cmd/fd2/modern_hd_panel_transition.go)。
正常驗證沿[字型GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--status --panel-transitions`
或`--items --panel-transitions`；[區域探針](../../tools/verify_fd2_hd_panel_transitions.py)
比對實際GPU與移動後穩定HD參照，不以CPU階段替代。

最終50張實際畫面含11張移動區域，兩路徑均通過兩主題／來源完整保存相同與新程序標題LOAD。
相關介面48.994秒及五資料／清冊套件通過。戰場caller限E1，未新增正常戰場GPU。
來源與分級見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的`panel_transition_followup`。
其他設施對話／清單轉場、圖示內文字、成功演出、完整戰場美術與平台效能仍另列。

### 商店金幣滾動字型（CONFORMED，限已列GUI與fixture範圍）

延伸已核准的黑體與商店數字，僅改變扣款／入款滾動的繪字來源。
原版定位沿用 `0x2d516`／`0x2d3ff`，位址基準為 IDA flat-loader linear address，
輸入雜湊沿[固定清單](../data/fd2-reference-files.json)的 FD2.EXE。
既有證據與消費端見[系統設計](56-fd2-remake-sdd.md)、
[金額合成器](../../remake/internal/campaign/native_gold_debit.go)及其原有測試。
本輪不重開逆向，不新增原版逐幀或硬體時鐘一致聲明。

- 將既有八位十進位獨立滾動順序抽成具型別窗口計畫，原索引合成與 HD 使用同一計畫。
  扣款向下、入款向上，每位每輪九個窗口，6×99來源與6×9目的窗口保持。
- 時間軸建立時保存獨立的數字背景、來源窗口與色盤索引。不由畫面辨識或目前金額猜階段。
  HD Draw 只替換 `(16,98,48,9)` 的數字區，採現有20邏輯像素黑體及每位獨立裁切。
  一個來源窗口跨兩個數字時，兩個字形隨相同偏移滾動；原模式仍直接顯示索引快取。
  分離素材 `ui/fdother_005_item_panel/resource.json` 的31..40為6×8，只有32數字1為5×8；
  字距仍為6。滾動條每格9列，穩定數字在99列，滾動窗口自98列開始。
- 不把字型包裝帶入 Update，不改金額／物品提交時點、10毫秒步長、輸入鎖定或延續點。
  無效來源保留原畫格，主題中途切換不重建交易或推進動畫。
- 驗收包括扣款／入款、借位／進位、零交易、範圍拒收、索引結果保持、裁切與一次性延續點。
  普通 START 後 F9 接續已知旅館槽，正常購買及出售，觀測 GPU 滾動數字、F2往返、
  普通保存及冷讀；實際畫面與 CPU 候選分列，不宣稱精確同幀。不錄影。

其他設施／角色／物品轉場、成功演出中的金額、圖示內文字、戰場及平台效能仍另列。

實作入口為[金額文字快照](../../remake/cmd/fd2/modern_hd_gold_roll.go)，
驗證沿[字型GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--gold-roll`；
忠實交易對照另加`--gold-roll-theme faithful`。完整素材與輸出參數沿本檔既有入口。

同輪 GUI 發現出售確認穩定頁殘留 HD 歡迎詞，見[#255](https://github.com/wicanr2/fd2_re/issues/255)。修正契約：
`ComposeNativeShopSellQuestionBase` 已重建對話區，Draw 在提交原生問句前同步清除該區 HD 文字，
再用現有字模反查包裝擷取同份原生問句。忠實模式保持原畫格，未辨識字形保留原字形；
不新增翻譯，不改交易。歡迎詞殘留負例及實際出售確認畫面已驗。

最終HD與忠實普通買賣、F2、F5／F9及新程序標題LOAD通過，完整保存bytes相同。
22張實際畫面含4張有效HD金幣區域，所有圖已逐張審查；來源及分級見
[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的`gold_roll_followup`。
保存與來源的差異只有金額及既有raw移格造成的一個未使用格殘值，不新增原版保存放行。

### 整備預建問句轉場（CONFORMED，限已列GUI與fixture範圍）

延伸已核准的整備 HD 字型，保留原開收框與輸入時序。已證實的重製端來源為
`native_preparation_ui.go` 的六個建立端，以及 `native_class_ui_lifecycle.go` 的
逐次 Draw 承認排程。`NativePreparationConfirmationOpeningFrames` 先呈現六張空框，
再呈現四張含問句的選項動畫；收框依序為四張問句與五張空框，再由 caller 還原。
必選角色訊息另在六張空框後呈現一張文字。原版定位與資料雜湊沿既有整備契約及
[固定版本清單](../data/fd2-reference-files.json)，不新增原版語意。

- 建立轉場時保存獨立的底圖、原始字模／文本／選項素材及問句種類與必選身份。
  原始索引畫格仍為權威快取；不在 Update 包裝 HD 字模，也不由像素或 hash 猜來源。
- 只有已登記的文字階段在 Draw 從同份來源合成 HD 底圖與文字，再使用原選項動畫。
  空框、還原及未登記的 job 沿原畫格。未知字形保留像素，合成失敗回到原畫格。
- 中途 F2 可切換呈現；不得改畫格索引、Draw 承認、輸入鎖定、延續點或保存格式。
  原始素材與來源快照保持，不能讓後續游標、名冊或問句狀態改寫已建立的轉場。
- 驗收逐張比較來源配方與原索引快取，檢查開收階段、未知回退、中途主題切換與
  延續點只執行一次。正常城鎮與來源明示晚期槽另以普通鍵盤抽樣 GPU 問句轉場，
  收據區分擷取前後狀態，避免把穩定頁冒稱轉場；兩主題保存須相同。
  CPU 狀態僅定位候選；最終 PNG 的問句區須與 HD 穩定參照相同，選項框位置須位於
  開收途中。擷取可能跨過 CPU 階段邊界，前後狀態照實保留，不宣稱精確同幀。
  其他設施、角色／物品開收框、金幣滾動與圖示內文字仍另列，不錄影。

正式實作見[轉場來源快照](../../remake/cmd/fd2/modern_hd_preparation_transition.go)。
七種開收配方逐張與原索引畫格相同，來源變更不改已建立的轉場；中途主題切換、
失敗回退與延續點一次性測試通過，必選訊息仍限 RUNTIME-E1。
正常 JSON 城鎮出發問句的開收框、第24章原生槽記錄問句開收框與確認問句收框，
共5張實際 GPU 轉場通過問句 RGB 及內移選項位置驗證；另28張正常路徑、
F2／縮放／說明與保存畫面已審查。JSON 與來源 bytes 相同；原生槽兩主題保存相同，
建構祖先明示，不提升自然難度或整章 E2。
首輪 CPU 階段判定曾把空框／穩定頁算作通過，又誤拒 GPU 已呈現的跨階段樣本，
保留原失敗並以同一建置乾淨重跑兩條路徑。完整來源與勘誤見
[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的 `preparation_transition_followup`。
重跑[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加 `--preparation --transition-frames`，
晚期樣本另加 `--native-save <既有SAV>`。本切片不新增原版對拍或發行驗收。

### 整備與標題讀檔穩定畫面（CONFORMED，限已列GUI與fixture範圍）

沿用已核准的 A 黑體、1920×1200 與既有座標。資料鏈為
`nativePreparationUIAssets.status` 的字模／數字素材與正式名冊，經既有整備合成器
進入 `drawNativePreparation`；標題四槽由已驗證的存檔 metadata 經
`ComposeNativeLoadSlotsFrame` 進入 `drawNativeLoadSlots`。不新增原版語意。

- 整備只在穩定 Draw 暫時複製資產、包裝字模與已驗證的數字 bank；繪製後還原。
  包含選人狀態、確認／必選角色訊息、城鎮出發／非城鎮保存問句及四槽。
- 標題讀檔使用自己的 palette 與字模擷取文字；完整索引畫面替換時清除先前 HD 文字。
  未知、歧義或字型缺字沿用像素。原生四槽與 JSON 後備路徑保持既有邊界。
- 原像素快取保持；整備問句轉場另依上節來源重建 HD 文字。不得將 Draw 包裝帶入 Update、變更部署規則、
  存檔 bytes、checksum 或交易驗證。圖示、烘焙標籤與頭像不在本輪字型範圍。
- 驗收包含穩定畫面文字、資產還原、原像素 bytes 不變、空槽／選取色與拒絕路徑，
  相關整備／讀檔回歸及實際視窗普通鍵盤操作。GUI 收據明示存檔來源與續玩邊界；
  非城鎮建構 fixture 不提升為正常玩家路徑。F2 往返及保存／讀取逐筆核對。
  不錄影，不修改正式發行包或私人美術。

正常標題 LOAD 既有 JSON 槽後，城鎮出發選 NO、返回旅館 F5／F9 通過，
完整保存與來源相同，限 PLAYER-E2-LIMITED。既有第24章原生槽另驗普通標題 LOAD、
保存第二槽、選人15筆、確認取消、F2、說明與縮放。它有攻略建構與清敵祖先，來源明示，
不提升自然難度或整章 PLAYER-E2；必選訊息等其他分支限 RUNTIME-E1。
最終28張實際視窗已審查，兩種主題保存各自相同，未錄影。
初輪縮放擷取仍為舊 viewport，不能列通過；沿同一建置等待後續 Update，
並確認 X11 幾何後重跑，完整畫面通過。Xvfb 就緒競態另保留環境失敗日誌。
來源與勘誤見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的 `preparation_followup`。
沿[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加 `--preparation` 重跑；
原生晚期樣本另加 `--native-save <既有SAV>`，工具只寫複本。

### 物品操作與商店裝備穩定面板（CONFORMED，限商店PLAYER-E2／其他caller相關回歸）

沿用A黑體、1920×1200及原版版面。已證實的重製端資料鏈為
`prepareNativeItemPanelMode`、`refreshNativeItemPanelMode`及`rebuildNativeItemPanelContents`，
從80-byte角色紀錄、分離素材與物品效果列產生像素快取；`drawNativeItemPanel`由戰場物品、
商店裝備與滿袋交換共同消費。原始輸入沿[固定版本清單](../data/fd2-reference-files.json)，
不新增原版二進位語意或物品規則。

- 開啟／裝備重建時保存無動態文字底圖與同份紀錄、資產、效果列快照；原像素快取保持。
  只有成功發布選取畫面才發布對應raw slot。裝備重建失敗須還原原快照與既有所有cache，
  不改持續名冊；關閉時清除快照。
- 各Draw階段沿原字模及31／42／119數字合成HD文字與選取色；開收轉場依上節移動與裁切。
  預建12幀開收框仍用像素快取；不在Update包裝字模，不猜快取的來源。
- 每個實際拷貝區域只遮蔽該區域的既有HD文字，保留外部文字；新文字同樣裁在拷貝區域。
  不讓商店歡迎詞／金幣或戰場文字穿過不透明物品面板。頭像、圖示及烘焙標籤仍是像素。
- 驗收包括與原合成器完整索引bytes相同、有洞raw slot選取、空名冊、快照及素材不可變、
  選取／裝備刷新、失敗還原、區域遮蔽及原12幀排程。戰場與交換caller另跑相關回歸列E1。
  正常START後F9接續既有旅館槽，經城鎮商店普通裝備選人、移動物品、重新裝備既有短劍，
  驗F2、縮放／說明遮罩、返回與F5／F9；保存兩主題須一致，來源差異逐筆檢查。
  不錄影，不改正式發行包、私人美術、存檔格式或翻譯。

正常START後F9接續既有旅館，經城鎮進武器店裝備服務，13張實際視窗已審查。
三筆選取、拒絕藥草及重新裝備短劍、F2、縮放／說明遮罩與返回保存通過；角色資料及金幣保持，
F2往返PNG SHA相同，兩主題與來源完整保存bytes相同。相關回歸涵蓋戰場物品與死亡交換caller，
不新增其正常GUI或原版對拍。來源、負例與驗證分級見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)
的`item_followup`；重跑既有[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--items`。

### 角色狀態與命令穩定面板（CONFORMED，限教會PLAYER-E2／戰場RUNTIME-E1）

沿用已核准A黑體、1920×1200及原版座標。已證實的重製端資料鏈為
`prepareNativeUnitStatus`的80-byte紀錄、分離素材與物品效果列，經
`RenderNativeItemPanelBaseResources`、`RenderNativeItemPanelData`及物品／命令列合成器，
由教會`drawNativeChurchStatus`及戰場`drawNativePlayerStatus`消費。
原始資產沿[固定版本清單](../data/fd2-reference-files.json)，不新增二進位語意。

- 開啟時保存具型別的面板輸入快照：紀錄、無字底圖、字模／數字素材、物品效果列與命令資料。
  同份輸入產生原有像素快取，忠實模式及12／14幀開收框沿原排程。
- Draw從同份快照合成HD底圖與文字，沿既有字模唯一反查與未知回退；轉場依上節來源／目的區域。
  共用文字及31／42／119數字包裝只在Draw生效，不進Update，不以指標或雜湊猜快取來源。
  原始素材與快照不變，切換F2不重新取得或修改角色資料。
- 覆蓋全畫面前清除前一層HD文字；教會狀態面板同時覆蓋歡迎詞與金幣區，不保留被遮蔽描述。
  角色頭像及圖示字樣仍是像素素材，原轉場快取保留，HD文字依上節接續；不新增翻譯或改版面。
- 驗收包括與原合成器完整索引bytes相同、快照不可變、未知字形回退、兩主題往返及
  教會／戰場兩個消費端。正常START後F9接續既有旅館槽，經城鎮進教會，
  普通選人進狀態／命令、F2、縮放及說明遮罩，再返回並保存；兩主題與來源bytes須一致。
  戰場消費端另以fixture穩定畫面及正式推進介面測試列RUNTIME-E1，不提升連續戰役或原版像素對拍。

正常START後F9接續既有旅館，經城鎮普通選人開啟索爾狀態與悠妮命令頁，
F2往返、縮放／說明遮罩及24張實際視窗審查通過。兩頁切回HD的PNG SHA各自相同，
返回旅館F5／F9的兩主題保存與來源完整bytes相同。戰場消費端與14幀推進另以fixture驗證，
不宣稱新取得正常戰場GUI。範圍、來源與回歸見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)
的`status_followup`；重跑既有[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--status`。

### 教會即時文字面板（CONFORMED，限已驗玩家路徑／其餘RUNTIME-E1）

沿用A黑體、1920×1200、原版座標與語系路由，只換呈現。已證實的重製端消費端為
`native_church_ui.go`、`native_church_roster_ui.go`、`native_class_ui.go`、
`native_church_revive_ui.go`與`native_church_transfer_ui.go`；它們從既有FDTXT、具型別隊伍
及費用／物品資料即時合成。原始輸入依[固定版本清單](../data/fd2-reference-files.json)，
字模及十進位字庫契約沿下節與既有商店規格，不新增原版二進位語意。

- 當次Draw使用獨立的共享資產包裝，原字模、數字素材與名冊不變。保留字模驗證及
  未知字形回退，文字提交最終層；普通金幣及已驗31／42／119字庫以原目的座標提交。
- 教會穩定選單、名冊、轉職列表／問句、復活列表／問句／空名冊與餘額不足回覆、
  交付物品列表及滿袋回覆使用此路徑。原有在地化名冊沿原字串及安全矩形提交，
  其他原生文字維持既有語言，不新增翻譯。
- 下方框從原生y112覆蓋文字區；列表或對話重建前清除下方文字描述，保留y99金幣。
  `NativeChurchMenuBase`只清除(201,169)至(305,189)服務圖示，不自行當成已清除提示。
- 其他預建開收框與復活動畫保留像素文字；角色狀態開收框與命令轉場已由上節接入HD。字型包裝不進Update，
  不以byte slice指標或雜湊推測快取幀歸屬；F2下一次Draw立即依主題重新選擇呈現。
- 驗收包括所有即時分支的覆蓋文字去重、金幣保留、共享資產不可變、忠實模式
  索引結果保持，以及既有轉職／復活／交付正式輸入測試。正常START後F9接續已驗旅館槽，
  由城鎮進教會，抽驗四語名冊、既有倒下隊員的復活列表／問句與取消、交付物品列表與取消、F2、離開及保存一致性；其餘fixture分支
  僅列RUNTIME-E1。使用既有Docker工具鏈與實際視窗擷取，不錄影，不變更正式發行包。

正常START後F9接續既有旅館槽，經城鎮進教會；四語名冊、復活列表／問句及取消、
交付物品列表及取消、F2、縮放／說明遮罩與15張實際視窗審查通過。
名冊F2往返PNG SHA相同；回旅館普通F5／F9，兩主題與來源槽完整bytes相同，金幣保持1000。
沒有新增狀態注入；轉職、空名冊、餘額不足、交付目的名冊與滿袋等fixture分支仍限RUNTIME-E1。
相關cmd介面及campaign／fdtxt／fdother／localization回歸通過，不宣稱完整cmd測試。
來源、輸入與分級見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的`church_followup`。
重跑既有[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--church`，其餘參數沿字型主路徑；不錄影。

### 原生旅館固定文字（CONFORMED，限旅館JSON玩家路徑）

沿用已確認的A黑體與1920×1200，不改版面或服務。已證實的重製端來源為
`native_hotel_ui.go`、`campaign/native_church_scene.go`與`campaign/native_load_slots.go`：
提示從FDTXT索引選取，FFFE換行；槽列表展開FFFA數字、章名與標題，再呼叫
`fdtxt.Font.BlitNativeGlyph`。既有原版來源與位址見[酒店LOAD證據](../data/ida/fd2_hotel_load_20261004.json)；
本次只替換呈現，不新增二進位語意，輸入仍依[固定版本清單](../data/fd2-reference-files.json)。

- 只於旅館當次Draw使用獨立字模包裝，原字模資料不變。完成原有範圍驗證及背景填色後，
  可唯一對回單一Unicode且Noto有字的字模，改提交最終層文字；其餘保留原始點陣。
- 字模目的地沿原合成器，16原生像素一格，字級28邏輯像素；選取色沿色盤。
  槽框與服務回覆重建面板時先清除提示描述，不依[]byte指標或內容雜湊猜測圖層歸屬。
- 只涵蓋即時合成的旅館提示、保存／讀取回覆及四槽固定字樣。原有文字與各語系路由保持，
  不新增翻譯。預建動畫、圖示內文字及其他面板仍待辦。
- 驗收包括未知字模回退、原字模不可變、無效目的地拒收、槽框遮蔽提示，以及正常F9接續
  既有旅館存檔後選槽、保存、服務LOAD與F2往返。保存狀態須兩主題一致；不宣稱原版像素對拍。
  使用既有Docker工具鏈與實際視窗擷取，不錄影，不改公開發行包或私人美術。
  GUI可用唯讀`FD2_INPUT_AUDIT_INTERVAL=2`加密收據；未設定仍每120次Update取樣，
  按下與放開事件均記錄，測試須確認實際放鍵，不以幀數替代。此觀測不影響Update頻率、動畫時鐘或輸入處理。

正常START後F9接續既有旅館存檔，四語上下文、四槽選取、保存、服務LOAD及F2往返通過。
13張實際視窗完成審查，兩主題保存完整bytes一致，來源槽未變；拒收回覆與特殊字模回退另驗E1。
GUI的r4雖通過保存檢查，服務回覆卻殘留歡迎詞，已拒收並登錄[#254](https://github.com/wicanr2/fd2_re/issues/254)。
先以重複目的座標回歸重現，再清除重建面板前的提示描述；最終r5截圖不再重疊。
來源、負例與驗收見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的`hotel_followup`。
重跑既有[GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--hotel`，其他參數沿下節；不錄影。

### HD 清晰黑體主路徑（CONFORMED，限已驗切片）

使用者選定 A 清晰黑體，沿用 Noto Sans CJK。正式目標為1920×1200直接繪字，
涵蓋對話、商店與一般介面；忠實模式保留。此項不改規則或保存格式。

- 邏輯排版仍為640×400；字級、斷行與逐字時鐘沿既有消費端。新文字層在最終
  畫布以3倍字級柵格化，依最終幾何縮放與黑邊定位，不把低解析字形放大。
- 一般介面的文字依繪製順序收集；完整原生畫面覆蓋時清除先前文字。說明頁
  先清除文字層再提交自己的文字；全畫面黑白與淡出同時作用於文字。
- 原生對話從具型別頁面及既有逐字進度重建可見字形，清除原文字窗口後才疊新字。
  多頁、捲動與特殊字模須有明示處理；未知字模保留原字形，不猜Unicode。
- 商店沿既有資料、商品與語系字串產生文字，使用既有無字底圖。每個覆蓋文字
  窗口的面板先清除被覆蓋的文字描述，不殘留歡迎詞或上一個商品名稱。
  金幣與商品數字沿既有31／42／119字庫的0～9契約及實際Blit目的座標替換，
  保留既有補零、價格、顏色與未知值標誌。原圖先完整驗證，再建立只於本次繪製
  使用的透明文字占位圖；不修改分離素材與正式規則。數字以字形墨跡頂端對齊。
  HD主文字採已展示的淺色黑體與一個輸出像素的陰影，選取色仍沿既有色盤。
- 舊80×80現代包與忠實模式不啟用新文字層。原版圖中烘焙的標誌與裝飾仍屬美術。
- 驗收須包括直接高解析字形、縮放與裁切、F2往返、四語對話／商店及普通保存冷讀。
  分層記錄已接入與未接入的消費端，未驗證分支不宣稱整款字型完成。

已確認的[A／B字型原型](../figures/hd-font-style-comparison.png)為純字型樣式比較，不是遊戲執行期證據。實際呈現、來源雜湊與範圍見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)。

未接入部分：原生整備／原生標題LOAD等其餘固定索引面板、預建原生轉場job中的文字，以及FDOTHER圖示內的HP／貨幣等字樣。這些維持原有點陣；特殊字模或已綁定的不同原始字形若不能無損轉成Unicode，也明示保留原字形。本切片不關閉#204的完整字型／介面與戰場閘門。

正常START序章首句四語、F2同狀態往返、明示F9接續旅店存檔後的城鎮與商店、單筆購買1000→990、1280／1920與1600縮放說明頁及最大字級均通過實際GPU驗證。首章JSON正常標題LOAD、F5／F9與重新保存bytes保持。cmd/fd2、campaign、fdother及localization回歸通過；四語各5107筆譯文、狀態與變數保持。本節只將上述路徑列PLAYER-E2，其他消費端及完整字型覆蓋仍沿#204。

重跑時先在fd2-go-test-local:latest容器內建置go build ./cmd/fd2，再使用[字型GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)的--binary、--assets、--original-pack、--modern-pack、--town-save與新的--out。GUI使用fd2-promo:20261017-r1，原始資料唯讀、輸出獨立可寫；本輪未錄影，暫時建置與字型副本已刪除，雜湊保留於主收據。

逐字驗證發現既有`ComposeNativeStoryDialogueProgressiveFrames`把`next`直接加入捲動
快照，再將它用於下一行寫字；最後一張捲動畫格因此被後續字形污染。這是已證實的
重製端記憶體別名缺陷，非新原版語意。READY修正為發布獨立快照，保留既有十幀捲動
與逐字時鐘；以同一具型別頁面逐步比較索引字形位置，另保留忠實模式的靜態終點。

### 2026-10-18 非原生對話高解析切片（CONFORMED，限首章下框PLAYER-E2／上下框RUNTIME-E1）

本切片沿用已確認的手繪風格、480×480來源與最終呈現器，補齊正常JSON節點LOAD後的
非原生對話。既有玩家證據見[F2收據](../data/fd2-modern-hd-f2-20261018.json)的
首章LOAD截圖；正式消費端為`Game.Draw`中`!nativeStoryDialogueDrawn`的穩定對話分支。
以上為已證實的重製端行為，不新增原版二進位語意，也不重開匹配式反編譯。

- 已解析的`DialogLine.Speaker`選取同一清冊頭像；完整HD包直接提交480×480原圖，
  舊80×80包與忠實模式維持既有繪圖。缺speaker仍拒收，不猜角色對應。
- 保留既有`ps=2.1`及168×168邏輯區域：下框左上角(16,213)，上框(456,19)。
  下框水平鏡像，上框不鏡像；`Upper`覆蓋仍由既有消費端解析。
  因保留版面，1920×1200視窗中的此分支區域為504×504，來源細節為480×480；
  原生故事對話仍為既有480×480顯示區域。
- 只在`dlgPhase==0`且目前對話真的繪出頭像時提交HD描述；開／收框期間及下一幀
  沒有對話時不殘留。說明、狀態面板、黑白DAC與淡出沿既有最終呈現遮罩契約。
- 不改對話頁數、文字進度、框位置、規則、隊伍或存檔。原圖、清冊與私人庫均不變。
- 驗收包括上下框及鏡像的繪圖變換、舊包與忠實模式、遮罩與開收框回歸；
  使用上一輪正常START實際F5存檔的原樣複本，從新程序的正常START後以F9冷讀，
  驗證實際GPU頭像、F2往返、縮放與F9重讀。存檔SHA與兩種主題LOAD結果須保持。
  普通LOAD只驗實際可達的首章下框；其餘上下框幾何測試另標RUNTIME-E1。
  不建立錄影，不提升全戰役、全介面、跨平台或原版視覺對拍狀態。

本節當輪驗證入口勘誤：首章實際JSON沒有正`chapter`值，`nativeLoadSlotMetadata`拒收，
故標題LOAD仍停在選槽；同份F9可讀。這是既有JSON槽顯示與保存編號契約的缺口，
當輪獨立列為[#252](https://github.com/wicanr2/fd2_re/issues/252)，未猜填章號或放寬原版SAV驗證。
其後標題JSON確認已改用實際還原結果，忠實／HD及重新F5後的標題冷讀通過，見[獨立主收據](../data/fd2-json-title-load-20261018.json)。
本節較早收據仍只驗START後F9，不回溯提升其範圍。

兩個新程序正常START後F9、F2往返、說明遮罩及縮放通過；六個504區域GPU比較最大正規化MAE為0.00162285，
參照為480原圖經Triangle插值至504並鏡像，與GPU線性取樣存在明示插值差異，不宣稱零像素。
原樣存檔SHA保持，上框與鏡像幾何回歸通過。來源及限制見[非原生對話主收據](../data/fd2-modern-hd-dialogue-20261018.json)。
重跑[冷讀GUI工具](../../tools/verify_fd2_hd_dialogue_gui.py)時提供正式建置的--binary、公開--assets、
合法--original-pack、完整--modern-pack、既有F2主收據的--source-receipt及新的--out；
使用既有fd2-promo:20261017-r1容器，來源唯讀、輸出獨立可寫，不建立錄影。

### 2026-10-18 F2 動態高解析切片（CONFORMED，限原生故事對話與節點存讀檔）

本切片接續已核准的1920×1200／480×480呈現。現有F2只替換素材，啟動後無法新增
`FinalScreenDrawer`，且正在顯示的原生對話仍保留舊頭像指標。來源為
`player_shortcuts.go`、`main.go`及`native_story_dialogue.go`的正式消費端。

- 所有啟動主題共用最終呈現器。忠實模式的整數放大、縮小、非整數放大及關閉濾鏡，
  均沿鎖定Ebiten v2.6.6的預設契約。來源為容器內該版本`gameforui.go`的
  `DrawFinalScreen`與`screenShaderSrc`；附加HD層不改640×400座標及黑邊。
  改寫檔`ebiten_screen_filter.go`保留Apache署名與修改說明，完整授權見
  [Ebiten授權](../../remake/packaging/ebiten-LICENSE)；既有封包工具另保留模組原始授權。
- F2先預檢整包與目前對話的已解析speaker，再發布主題。失敗只顯示訊息；成功只換
  繪圖資源及目前頭像指標，不重建對話、不重設文字進度、頁數、戰況、隊伍或存檔。
  切回忠實模式立即清除現代頭像與上一幀HD描述。
- 完整預檢成功的素材在同一程序內快取；往返切換重用已驗證圖片，不重複解碼整包。
  重新啟動仍完整預檢。這不是檔案熱更新；失敗的候選不進快取。
- 視窗模式切入完整HD包採1920×1200，切出時恢復切入前大小；HD期間可自由縮放。
  全螢幕切換不改視窗大小。舊80×80現代包不觸發HD視窗轉換。
- 驗收包括壞包與缺speaker的原子拒收、對話進度保持、忠實濾鏡實際GPU對照、
  正常START途中雙向F2、縮放／黑邊／遮罩，以及第一章普通F5／F9保存回歸。
  F5 JSON沿既有`save.go`節點邊界契約，F9重建章起始狀態；比較同份存檔在兩種主題
  LOAD的結果與保存SHA，不要求還原章內暫態戰況，也不修改存檔格式。
  不建立錄影，不宣稱整款HD、跨平台效能或原版視覺一致。

正常忠實START途中雙向F2、視窗恢復、非整數縮放、黑邊及實際說明遮罩通過。
正常97拍至首章，七位原生故事頭像480區域MAE=0；同份首章JSON在兩種主題LOAD的單位結果相同，
保存SHA不變。五種GPU濾鏡對照全畫面零像素差異。來源與限制見
[F2主收據](../data/fd2-modern-hd-f2-20261018.json)。

本節當輪F9重建章起始時的非原生對話由`Game.Draw`取`modern.image`的80×80圖，未接480最終層，
已由上節非原生切片補齊首章下框，其他玩家路徑沿[#202](https://github.com/wicanr2/fd2_re/issues/202)驗證。
F1標題與首列間距另列[#251](https://github.com/wicanr2/fd2_re/issues/251)；其後已由本檔HD黑體切片修正並通過忠實／HD說明頁驗證。
重跑[正常HD驗證工具](../../tools/verify_fd2_hd_gui.py)加`--switch-theme`；
`--switch-only`只補抽正常START至父王對話的切換／遮罩，不算完整97拍或首章保存。
[濾鏡GPU工具](../../tools/verify_fd2_screen_filter_gpu.py)以`--binary`指定
`go test -c ./cmd/fd2`的測試程式，`--out`指定新目錄；均在既有`fd2-promo:20261017-r1`容器執行。

### 2026-10-18 高解析序章切片（CONFORMED，限啟動時高解析）

使用者撤回160×160方案，檢視1280×800／320×320與1920×1200／480×480同場景原型後，
明確選擇1920×1200、480×480頭像，維持版面比例與對話位置。此決定取代較早的倍率選擇。

- 第一階段只提升故事頭像的實際細節。遊戲狀態與640×400座標維持，現代主題在最終呈現層
  直接繪製480×480圖，不先縮成80×80或160×160再放大。背景、原版字模及戰場仍待後續切片。
- 依既有母稿產生31位已登錄頭像的480×480完全不透明PNG；保留80×80清冊與檔案，
  以可選 `story_portrait_hd_frame`、`native_story_dialogue_rgba_overlay_hd6_v1` 登錄。
  母稿須至少480×480；缺母稿、身分、尺寸、SHA、透明度任一不符即拒收，不以放大80圖補洞。
- 有高解析宣告時，speaker集合須與既有80×80集合完全一致，frame0、靜態閉嘴；
  重複、孤兒、部分集合或壞檔拒絕整個主題。沒有高解析宣告的舊包仍沿原有路徑。
- 採用專案鎖定的Ebiten v2.6.6 `FinalScreenDrawer`，沿其 `geoM` 保留比例及黑邊。
  480圖以原生上框(232,5)、下框(8,115)的80×80區域換算；1920×1200實際輸出為480×480。
  此API契約來源為容器 `/go/pkg/mod/github.com/hajimehoshi/ebiten/v2@v2.6.6/run.go`。
- 只對完整高解析現代包啟用新呈現器與1920×1200預設視窗；忠實模式與舊包沿原有呈現器。
  遊戲視野、鍵盤／指標座標、文字安全區、規則與存檔格式不變。縮放視窗時依相同比例縮放。
- 每張高解析頭像只建立一次GPU圖片。開／收框、非對話畫面、幫助與全畫面黑／白效果不得殘留頭像，
  淡出須覆蓋頭像；每次Draw重設當幀呈現描述，不能直接沿用上一句狀態。
- 驗收包括完整私有包原子預檢、缺件／壞檔反例、上下錨點及一像素細節GPU測試、
  黑邊／縮放／淡出測試、正常START的序章至首章流程及1920×1200實際視窗擷取。
  這是核准的美化呈現，不宣稱原版視覺對拍；三平台效能與整款HD尚未完成。
- 素材完成驗證後依catalog allow-list同步私人庫並核對遠端HEAD，公開庫不加入完整美術。

正常START已以1920×1200視窗走完97個序章對話拍點並進入第一章。七位抽樣頭像的480區域與清冊圖MAE=0，
含父王、王后及強盜；相關Go回歸、完整私人包與Python清冊檢查通過。
正式來源、原型、GPU、普通輸入、時序觀測與限制見[高解析收據](../data/fd2-modern-hd-20261018.json)。
本節原驗證限啟動時高解析，現況已由上節F2切片接續。非原生首章下框已由上節冷讀切片接續；其餘玩家路徑、介面／字型／戰場及跨平台效能仍由#204／#202追蹤。

重跑入口為[正常HD驗證工具](../../tools/verify_fd2_hd_gui.py)，在既有
`fd2-promo:20261017-r1`容器執行。`--binary`指定由`fd2-go-test-local:latest`建置的程式，
`--assets /repo/remake`指定公開資料根目錄，`--original-pack`指定含manifest的合法分離素材根目錄，
`--modern-pack`指定清冊登錄的私人圖檔目錄，`--out`指定不存在的新驗證目錄。
原版pack與公開assets是兩個命名空間，不得把pack作為assets根目錄。
工具保留普通按鍵、實際視窗PNG及JSON收據，不建立錄影；此入口不驗全戰役或跨平台。

### 2026-10-18 序章頭像切片（CONFORMED，限序章頭像）

使用者已確認父王原型的手繪風格，先補齊序章頭像，再接高解析呈現；忠實原版主題保留。
正式 `campaign_full` 起點、`ch00_pre` handler、原生對話映射及 LOADCH roster
需要 DATO 48（父王）、66（王后）及96（強盜）。
[需求收據](../data/fd2-modern-intro-coverage-20261018.json)保留97筆來源與解析方法。
現代主題正常 START 先前在父王對話拒收 speaker48，證實這是實際玩家阻擋。
補圖後同一正式第一版引擎以普通按鍵走過97個對話拍點、父王／王后／強盜及首章接縫，
沒有缺圖錯誤；[製作與正常路徑收據](../data/fd2-modern-portraits-20261018.json)保留來源、GPU與私人同步雜湊。

- 三個新增頭像沿既有 `native_story_dialogue_rgba_overlay_v1` 登錄；80×80、完全不透明、
  frame0、靜態閉嘴、各自SHA-256與原生DATO身分都須通過原子預檢。
- 保存使用者確認的父王原圖與後續同風格母稿，衍生80×80執行期圖。
  母稿與PNG留私人素材保存庫；公開庫只記清冊、來源、決策及雜湊。
- 不改原生對話錨點、文字安全區、隊伍、規則或存檔。
  正常START須越過父王、王后與強盜對話並進入第一章，才列本切片玩家路徑通過。
- 本節只授權既有尺寸的缺件補齊。高解析檔及最終視窗繪製仍須獨立READY契約與GPU驗證，
  母稿存在不代表高解析呈現完成。

本規格分別定義現代故事頭像、已登錄的九十四組可繪 FDICON 候選，以及依概念稿建立、
但保持原版圖塊索引的戰場圖集。九十四組中有八十九組四方向十二幀行走人物，另有
五組依原版證據保留三相靜態契約；selector 39／57 是純目的地重映射，不虛構人物圖。
selector 是圖像索引，不等於故事說話者或頭像身分；兩者不得混用。
它不授權從合成概念稿反切地圖圖塊或 HUD，也不改變忠實原版主題。

## 全戰役現代戰場圖集契約（READY）

- `map0/map.json` 已固定邏輯圖塊為 `24×24`、每列 16 格；原版分離圖集為
  `384×432`，共 288 格。地圖資料中的 tile ID、通行規則、事件座標與人物座標
  均不因主題改變。
- `ch01-battlefield-style-a.png` 只提供地貌色彩、材質與光影方向。它包含烘焙人物，
  因此不得直接作背景，也不得由合成畫面反切碰撞或前景遮罩。
- 正式候選 `modern.map0.tileset.style_a` 至 `modern.map32.tileset.style_a` 各為
  單一 RGBA PNG，依 `map_tileset_indexed_geometry_v1` 登錄 `map_id`、
  `tile_width=24`、`tile_height=24`、`columns=16`、實際 `tile_count` 與 SHA-256。
  寬度均為 384；高度依原圖集為 144／288／432／576，格數相應為
  96／192／288／384，不可硬套第一關尺寸。
- 產生器逐格處理原版分離圖集，只從概念稿取得色彩與平滑繪法；不得重排格子、
  增刪圖塊或混入人物。這是經使用者核准的現代呈現，不是原版視覺 parity 證據。
- `modern.battlefield.tileset.style_a.reference` 是依原 atlas 與第一關概念稿生成的
  高解析手繪風格參考。直接縮放該稿後，以正式 `map0` 重組會出現格線漂移、斷岸與
  錯接屋頂，已拒絕作 runtime atlas；正式 33 張圖集只擷取它的色彩／材質分布，
  每一格的結構仍由原版分離圖集控制。純黑洞窟與畫布外區域必須保留純黑，避免
  中性色轉換產生假地板。
- 載入時須先驗證清冊身分、雜湊、PNG 尺寸與圖塊幾何，再依地圖目錄原子替換
  `map0` 至 `map32` 圖集。
  任一條件不符即拒絕整個現代主題，不得混搭半套現代圖塊。
- 現代主題使用正規化地圖 renderer 疊加現代人物；原版完整 indexed frame 不得再
  覆蓋它。忠實原版主題的 indexed compositor 行為維持不變。

### 2026-10-18 戰場角色狀態覆蓋區域修正（CONFORMED，限首章抽驗）

正常標題 LOAD 普通首章 JSON 後，以 Z 聚焦、Home 檢視索爾的實際 GPU 圖片，
確認面板外現代底圖被原版 indexed 全幀覆蓋。前測來源為 ecb006ad 的正式遊戲，
`battle-status-gui-before-r1` 保留實際圖片及輸入；索爾無命令頁，工具等待命令的
逾時另列驗證腳本錯誤。這是重製端已證實缺陷，不新增原版 RE 或保存例外。

- 現代主題的戰場角色狀態只繪製 `NativeItemPanelRegion` 的目的區域。
  原開收與命令換頁的來源矩形、移動、遮蔽及文字裁切沿既有配方；框外保留已繪現代地圖。
- Draw 可從獨立快照合成 HD 文字；失敗時使用已驗像素面板，仍不得用原版完整底圖覆蓋。
  教會的完整設施來源與忠實戰場快取路徑保持；不改 12／14 張排程、Draw 承認或規則。
- 正常標題 LOAD 既有首章槽，以普通 Z／Home 抽有／無命令兩種角色，F2、返回與縮放。
  實際 PNG 在面板外的地圖區域須等於開啟前參照。單位、游標與完整保存須保持，
  另以同份檔案正常 F9／新程序標題 LOAD 驗證，不把節點保存當戰中快照。
- 單元驗證逐階段快取不變、文字在目的區域內、框外 HD 文字保留與框內遮蔽，
  並跑角色／面板／保存相關回歸。仍不宣稱早中晚全部交界或原版視覺 parity。

正式實作與正常首章抽驗通過：索爾無命令、悠妮有命令，14張實際畫面已審查。
六組面板外54000像素區域與各自開啟前參照完全相同，修正前50832像素不同。
完整保存與來源SHA相同，F2／F5／F9與新程序標題LOAD通過；沒有原版對拍或戰鬥驗收。
相關介面35.408秒及文字清冊／localization通過，四語清冊bytes完全保持。
最終工具與回歸Go相同；冷讀空框的過早擷取已分類工具問題，同份binary完整重跑。
來源見[字型主收據](../data/fd2-modern-hd-fonts-20261018.json)的`battle_status_followup`，
重跑[正常GUI工具](../../tools/verify_fd2_hd_fonts_gui.py)加`--battle-status`。
戰場轉場外區GPU、早中晚其他交界與真正高解析美術仍待完成。

## 現代戰場狀態面板契約（READY）

- `modern.battle.hud.style_a` 只作整體構圖參考；正式 runtime 不得把概念稿中的
  地圖、人物、選單或文字裁入 HUD。
- `modern.battle.hud.panel.style_a` 是獨立 `149×42` RGBA 背景，依
  `battle_status_panel_149x42_v1` 登錄。母稿只含深藍底板、金色雙框與角飾，
  不含頭像、文字、數值、圖示或填滿的 HP／MP 條。
- 現代主題沿用既有 `×2` 邏輯尺寸與左下方錨點；角色名、LV、HP、MP、AP、DP、MV
  仍由 runtime 依目前單位狀態繪製，不可烘焙進圖片。
- 面板檔名、SHA-256、尺寸與完全不透明契約任一不符，即拒絕整個現代主題。
  忠實原版主題仍使用已證實的 FDOTHER／indexed HUD，不受此素材影響。

## 現代四向行動選單契約（READY）

- 原版已證實的結果順序維持 `attack／spell／item／wait`，畫面位置維持
  上／左／右／下，錨點仍由目前選取單位或游標方塊換算；不得固定到左上角。
- `modern.battle.action-icons.style_a` 依 `battle_action_icons_4x28x26_v1`
  登錄四張 `28×26` RGBA PNG，語意順序固定為攻擊、法術、物品、待機。
  圖像分別使用銀劍、魔法杖與星芒、藥水與布袋、沙漏；不再以未證實的
  `status` fallback 圖示冒充法術。
- 四個檔名、逐檔 SHA-256、尺寸、完全不透明、語意順序任一不符，即拒絕整個
  現代主題。選單開合四幀、可用性、鍵盤輸入及命令結果仍由既有 runtime 控制。
- 忠實原版主題繼續使用 FDOTHER action cells；完整通過 catalog 驗證的現代主題
  則由四枚重繪圖示擁有 action overlay 呈現，不得再被原版 FDOTHER 圖層蓋回去。
  圖示集合不完整時整組失敗即關閉，不得混搭原版與現代按鈕；此切換不改變原版
  命令語意或位置證據。

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

1. 每個 `modern.fdicon.group_NNN.style_a` 必須一次提供 12 個完整 `24×24` PNG 槽位，
   保留 frame 0..11、pose 0..3 與 cycle 0..2 的穩定映射。原版若明確重複某幀，
   現代素材也必須保留該重複時序，不得為追求雜湊不同而虛構新姿勢。
2. 每張 alpha 只能是 0 或 255；catalog validator 在私人素材驗證模式逐像素檢查。
3. 生成的第 11 格曾因雙劍與比例漂移被拒收，後續暫以第 9 格佔位。現行候選
   以第 9 格的上半身及第 10 格的下半身做確定性像素合成，保持人物尺度、裝備
   側別、方向與 `24×24` 邊界，形成獨立第三步態。這是**現代近似**，不是原版
   第 11 格的逐像素重繪。
4. catalog 的 `cycle_policy` 預設為 `three_distinct_cycles`，要求 12 個幀雜湊皆不同；
   只有原版逐位元證據明確出現重複幀時，才可使用 `source_exact_repeats`，且清冊中
   至少要有一組相同雜湊並附原版 source ref。兩種政策都可列為 `runtime_candidate`；
   renderer 尚未接線、正常玩家尚未擷圖，所以不可標為 `runtime_ready`。
5. 原生 indexed bank 與正規化 RGBA loader 是兩條不同 consumer；接線時兩條
   路徑必須共同抽測，且現代主題缺任一格即整組失敗即關閉。

### selector 0–31、68、76、77、78、80、82、83、85、88 與 91 候選

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
- `modern.fdicon.group_022.style_a`：只依原版 `fig_022_f00..f11` 建立中性
  selector 投影；保留藍黑色巨大長髮、頭頂小紅飾、灰綠護甲、深藍腰帶與
  褲、淺灰護肩與護腕，不增加原小圖無法證實的武器、盾、披風或魔法效果。
  目前尚無角色表或劇本資料足以把 group 22 固定命名為特定角色，故 catalog
  保留原 selector 身分；現代母稿只負責四方向三步態的美術投影。
- `modern.fdicon.group_023.style_a`：同樣只依 `fig_023_f00..f11` 建立中性
  selector 投影；保留體積巨大的棕色長髮、深藍寬額帶、深藍甲衣與護臂、金色
  腰腹飾帶、背後暗紅披風，以及側向幀可見的窮銀色短直刃；不增加盾、頭盔、
  第二把武器或魔法。角色身分尚無足夠資料閉合，catalog 故不硬編人名。
- `modern.fdicon.group_024.style_a`：只依 `fig_024_f00..f11` 建立中性
  selector 投影；保留光亮禿頭、耳側灰白髮、濃密灰白八字鬍、巨大金黃立領／披肩、
  深紅寬大長袍、中央深藍黑內衣與棕金飾邊。原圖無武器，故不新增法杖、盾、
  皇冠、帽子或魔法效果；角色身分未閉合，catalog 仍只保留 selector 24。
- `modern.fdicon.group_025.style_a`：只依 `fig_025_f00..f11` 建立中性
  selector 投影；保留低伏深藍巨大兜帽、完全藏在黑影的臉、窮淺灰亮眼、深藍布甲、
  藍黑腰帶、少量紫藍下裝與水平持握的銀灰直刃。第三列固定為純背面，不顯示
  眼縫或刀刃；不新增裸露臉孔、第二把刀、盾、披風、重甲或魔法。身分未閉合，
  catalog 只保留 selector 25。
- `modern.fdicon.group_026.style_a`：只依 `fig_026_f00..f11` 建立中性
  selector 投影；保留低伏修長的深藍灰龍首與長吻、頭後短角／硬棘、暗灰綠胸腹與
  四肢、背後成對藍灰摺疊翼片，以及身側窮長金黃直刃。第三列背向的前兩步不顯示
  刀刃，第三步才在右側露出少量金刃；翼片不解釋為圓盾，也不追加第二把劍、火焰、
  坐騎、披風或魔法。身分未閉合，catalog 只保留 selector 26。
- `modern.fdicon.group_027.style_a`：只依 `fig_027_f00..f11` 建立中性
  selector 投影；保留低伏橄欖綠龍首與長吻、灰綠胸腹與四肢、成對橄欖綠摺疊翼片、
  灰銀直刃，以及中央有清楚黑十字的白色鳶形盾。背向三幀不顯示劍、盾或十字；
  不把白盾改為圓盾，也不追加第二把劍、火焰、坐騎、披風或魔法。身分未閉合，
  catalog 只保留 selector 27。
- `modern.fdicon.group_028.style_a`：只依 `fig_028_f00..f11` 建立中性
  selector 投影；保留巨大低壓的深藍圓頂全罩頭盔、黑色狹縫中的兩枚紅眼、灰綠矩形
  口部護甲、帶短黑刺的厚重肩甲、深藍胸臂與灰綠腹甲／腰前護片。背向幀不顯示
  紅眼或正面面甲；不新增裸露皮膚、頭髮、劍、斧、盾、披風、角或魔法。身分未閉合，
  catalog 只保留 selector 28。
- `modern.fdicon.group_029.style_a`：只依 `fig_029_f00..f11` 建立中性
  selector 投影；原小圖無法判定是臉、面具或頭盔，故只保留可見的巨大光滑藍灰橢圓
  頭部、下半部兩枚小型淺藍眼光與少量深藍短紋、厚重灰綠環狀披肩／立領、鈷藍寬袖
  長袍及中央黑色直襟。不增加鼻口、耳朵、頭髮、人類皮膚、法杖、武器、盾、角、翼或魔法；
  catalog 只保留 selector 29，不宣稱種族。
- `modern.fdicon.group_030.style_a`：只依 `fig_030_f00..f11` 建立中性
  selector 投影；保留頭部、領肩、胸腹與手臂的灰白偏灰綠舊布帶、橫跨眼部的亮鈷藍
  帶狀眼縫、深藍內衣與腿部、紫藍腰前布及纏帶彎曲爪手。背向保留灰綠頭帶、深藍
  後腦帶區與腰布，不顯示正面眼縫。不畫成骷髏、裸臉殭屍或金屬騎士，也不新增
  劍、盾、法杖、披風、角、火焰或魔法。身分未閉合，catalog 只保留 selector 30。
- `modern.fdicon.group_031.style_a`：只依 `fig_031_f00..f11` 建立中性
  selector 投影；原小圖無法確認頭部是齊瀏海髮型、帽或兜帽，故只保留巨大方正、
  上緣平直的深海軍藍頭部輪廓、棕膚小臉與藍眼、蓬鬆灰綠領巾／肩衣、深灰綠上衣、
  深色下裝與藍色腰布。不把頭部改成圓頭盔，也不追加劍、盾、法杖、弓、披風、角、
  魔法或職業徽記。身分未閉合，catalog 只保留 selector 31。
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
  `speaker=77` 消費；研究投影中的單字「約」已證實只是 operand 77 被誤當字模，
  不是人物姓名。現代母稿只依
  `fig_077_f00..f11` 可見的全罩深灰綠鋼盔、厚重板甲、大型圓盾、直劍與背面暗紅
  腰帶建立，採中性「第三章敵方重裝隊長」身分，不與約拿 speaker 21、蘭斯洛特
  group 7 或普通追兵 group 76 合併。
- `modern.fdicon.group_078.style_a`：`map7／8／17` 共 29 筆敵方劍士均一致使用
  `fig／portrait／map_selector_key／battle_fig=78`。現代母稿只依原版十二幀保留
  低矮封閉深藍頭盔、面罩內紅眼、極寬圓厚雙肩甲、敦實重甲與少量棕色下擺；
  原版小圖沒有足以可靠辨認的武器或盾，故不額外添加，也不推測姓名或軍階。
- `modern.fdicon.group_090.style_a`：`map3` 至 `map8` 共 26 筆敵方劍士均一致使用
  `fig／portrait／map_selector_key／battle_fig=90`。現代母稿只依原版十二幀保留
  低垂紅褐兜帽、完全藏在陰影中的面孔、黃綠眼光、寬大深藍肩衣與敦實輪廓；
  原圖沒有可靠可見的武器或盾牌，故不額外添加，也不推測姓名或軍階。
- `modern.fdicon.group_080.style_a`：`map9／16／21／31` 共 38 筆敵方記錄使用
  `fig／portrait／map_selector_key／battle_fig=80`，但職業欄有「劍士」與原始
  `r0c1` 兩種值，故只登錄為中性重甲單位。現代母稿依正規化原版十二幀保留
  深藍重甲、紅色直立盔冠、紅橙胸甲紋與寬圓肩，不追加武器、盾牌、姓名或職級。
- `modern.fdicon.group_082.style_a`：第 7–9 戰的單位資料共有 49 筆敵方劍士使用
  `fig／portrait／map_selector_key／battle_fig=82`。現代母稿只依原版十二幀可見的
  矩形全罩盔、窄眼縫、灰盾、直劍、暗灰甲與紅色下擺重繪，不推測姓名或故事身分。
- `modern.fdicon.group_083.style_a`：第 6–9、18–19 戰的單位資料共有 42 筆敵方
  劍士使用 `fig／portrait／map_selector_key／battle_fig=83`。現代母稿保留原版
  鈷藍矩形全罩盔、兩側短角、黃綠眼光、藍盾、直劍與紅色背面下擺，不推測姓名。
- `modern.fdicon.group_085.style_a`：`map16／17／19／20／21` 共 36 筆敵方劍士
  使用 `fig／portrait／map_selector_key／battle_fig=85`。現代母稿依原版十二幀
  保留低垂大型深藍桶盔、完全藏在陰影中的臉與紅眼、方盾、簡單直劍及暗紅腰布；
  不加入裸臉、羽飾、皇冠或英雄式裝甲，讓量產士兵身分保持清楚。
- `modern.fdicon.group_086.style_a`：`map23／24／25／31` 共 24 筆敵方劍士一致使用
  `fig／portrait／map_selector_key／battle_fig=86`。現代母稿依原版十二幀保留
  淡灰綠圓頂全罩盔、水平眼縫、層疊頸甲、深藍肩甲、帶深藍十字記號的大盾及
  背側直立長兵器輪廓；臉部保持完全遮蔽，不添加羽飾、披風或英雄式比例。
- `modern.fdicon.group_088.style_a`：第 10、12–16、22 戰的單位資料共有 48 筆
  敵方劍士使用 `fig／portrait／map_selector_key／battle_fig=88`。現代母稿保留原版
  深藍護甲、紅褐頭巾、遮面眼部與橫持直劍，不推測姓名或陣營內職級。
- `modern.fdicon.group_091.style_a`：九個地圖資料共有 44 筆敵方記錄使用
  `fig／portrait／map_selector_key／battle_fig=91`，職業欄包含「劍士」與 `r0c1`，
  原版十二幀也沒有可見武器。現代母稿只依輪廓近似為深藍尖垂兜帽、黑色面孔、
  紅眼與對稱暗紅外緣；「兜帽」屬強視覺推論，正式身分仍是中性敵方單位，不猜
  職業、種族、服裝結構或姓名。
- `modern.fdicon.group_032.style_a`：**2026-09-05 勘誤**：舊稿只看灰階匯出，誤把
  原版 `FDICON.B24` 的 384..395 幀判成全罩巨盔。按 FDOTHER #0 正式 DAC 色盤重新
  檢查後，可見的是藍髮、紅額帶、裸露臉部、灰綠肩甲、深藍衣裝與淡灰綠短刃。
  現代候選已依彩色證據重畫；身分仍未閉合，只以 selector 32 登錄，不從外觀猜姓名。

### 2026-09-05 核准母風格擴展

- 使用者核准 selector 000／032 所呈現的現代日系手繪 Q 版方向：頭部約占全高
  55–60%、短身短肢、乾淨深色外輪廓與柔和高光；方向與三步態仍由原版圖組約束。
- selector 002、008、009、013 已依此母風格重新校稿並降採樣為正式 `24×24` 幀；
  分別保留巨塔盾與窄劍、弓手長髮與箭袋、悠妮朱紅高馬尾與無武器輪廓，以及
  金髮弓手的黑額帶與非尖耳限制。
- 這是美術風格定稿，不提升角色身分或原版逐像素證據等級；所有色彩、裝備與
  方向仍以各 selector 的既有原版證據及拒收限制為準。
- `modern.fdicon.group_033.style_a`：原版彩色匯出幀 396..407 可直接確認為紅褐短髮、
  黑色額帶、灰綠輕甲與淡冰藍短刃的年輕戰士；正面及兩側露臉，背向只顯示後腦、
  額帶後緣與背甲。專案地圖資料尚未找到 selector 33 的直接角色或關卡身分，故不套用
  人名或職業；現代母稿只依可見造型與十二幀方向重繪。
- `modern.fdicon.group_034.style_a`：`class_change_targets.json` 的已證實預設映射為
  current portrait 2 → target portrait 34；原版 selector 2 與 34 彩色幀皆顯示同一位
  棕髮、黑額帶、背負巨大淡灰綠塔盾並持銀白直刃的人物。首版現代稿把正面背盾誤畫成
  第二面手持盾，已拒收；修正版只保留單一背盾，正面露出深灰綠胸甲，背向由盾背遮住
  大部分人物。角色姓名仍不由 portrait 數值猜定。
- `modern.fdicon.group_035.style_a`：轉職表證實 current portrait 3 → default target 35；
  原版 selector 3 與 35 的彩色幀保持相同紅褐巨髮、濃密大鬍與矮壯身形，target 35
  新增銀灰短柄戰斧、方形灰綠盾及重甲。現代稿保留斧盾的方向可見性，背向以後腦
  巨髮遮住頭頸；角色姓名只由另有來源的身分表處理，不從轉職數值反推。
- `modern.fdicon.group_036.style_a`：轉職表證實 current portrait 4 → default target 36；
  原版 selector 4／36 的彩色幀是同一位棕髮、寬方形深藍帽、銀白直劍的年輕劍士，
  target 36 只加深藍肩披／短背披與較厚護甲。現代稿沿用已定稿 selector 4 的人物比例、
  帽形與武器，只增加這些可見轉職層次，不新增盾牌或改變人物身分。
- `modern.fdicon.group_037.style_a`：轉職表證實 current portrait 5 → default target 37；
  原版 selector 5／37 彩色幀保留相同灰綠長形頭部披覆、銀白直劍、小型深藍方盾與
  纖細身形，target 37 的主要差異是紅褐背披改為鈷藍背披並加深衣甲藍色。現代稿延續
  selector 5 的人物比例；頭部披覆究竟是長髮或兜帽仍不強行命名。
- `modern.fdicon.group_038.style_a`：角色表證實 selector 6 是萊汀，轉職表證實
  current portrait 6 → default target 38。原版 6／38 彩色幀保留相同深紅褐束髮、
  黑額帶與銀白槍刃；target 38 把灰綠甲改為深鈷藍重甲並加入金色腰胸飾。現代稿依此
  延續萊汀基礎造型。一次誤用 selector 36 母稿產生的未登錄檔已在清冊前刪除。
- `modern.fdicon.group_040.style_a`：角色表證實 selector 8 是希莉亞，轉職表證實
  current portrait 8 → default target 40。原版 8／40 彩色幀保留相同棕色長髮、
  紅額帶、木弓與箭袋；target 40 將深藍衣甲改成深酒紅／黑褐色，背向三幀新增可見的
  淺青灰色斜向箭袋帶。現代稿延續 selector 8 的人物比例，不新增披風、盾牌或魔法效果。
- `modern.fdicon.group_041.style_a`：角色表證實 selector 9 是悠妮，轉職表證實
  current portrait 9 → default target 41；特殊道具分支另會覆寫成 target 52，故本組
  只代表預設轉職。原版 9／41 保留鮮紅高馬尾與藍寶石額帶；target 41 將藍紅布裝
  升級為炭黑／深灰甲與銀灰肩臂護甲。現代稿只沿用這些可見差異，不混入特殊分支造型。
- `modern.fdicon.group_042.style_a`：角色表證實 selector 10 是馬琳，轉職表證實
  current portrait 10 → default target 42。原版 10／42 保留粉紫齊頸短髮與金色額飾；
  target 42 將紅金長衣改成深藍綠／炭黑衣甲，肩臂加入淡青綠／銀灰金屬輪廓。原版
  小圖不足以證明該輪廓是法杖，現代稿不增加未證實武器。
- `modern.fdicon.group_043.style_a`：角色表證實 selector 11 是索菲亞，轉職表證實
  current portrait 11 → default target 43。原版 11／43 保留灰金齊頸短髮、頭頂紅色
  髮飾與短杖；target 43 將深藍／紅衣裝改成棕紅、赭橙、深褐與少量深綠金邊長衣。
  現代稿維持短杖比例，不放大成未證實的長槍。
- `modern.fdicon.group_044.style_a`：角色表證實 selector 12 是凱麗，轉職表證實
  current portrait 12 → default target 44。固定雜湊原版的 selector 12／44 不只套色後
  12 幀逐像素相同，逐幀 `frame.png`、`mask.png`、`remap_mask.png` 也全部位元組相同。
  因此現代 group 44 明確複用 group 12 的已核准母稿與十二幀；不憑轉職索引硬造原版
  不存在的外觀差異。
- `modern.fdicon.group_045.style_a`：角色表證實 selector 13 是貝克威，轉職表證實
  current portrait 13 → default target 45。原版兩組的 mask／remap 逐位元相同，12 張
  frame 則全部不同；套色後輪廓、長金髮、額帶、弓與箭袋相同，主要差異是深墨綠衣甲
  改成明亮黃綠／草綠。因此現代稿只更新有證據的配色，不改變角色與裝備。
- `modern.fdicon.group_046.style_a`：角色表證實 selector 14 是珊，轉職表證實
  current portrait 14 → default target 46。原版兩組的 frame／mask／remap 十二幀皆
  不同；target 46 保留鮮藍高馬尾與綠寶石額帶，將深藍金邊長袍改成深綠內衣甲、
  紅褐外袍／披肩與金色邊飾。現代稿依四方向獨立輪廓重建，不作單純換色複用。
- `modern.fdicon.group_047.style_a`：轉職表證實 current portrait 15 → default
  target 47，但角色姓名仍未閉合。原版 15／47 的 12 張 frame 全部不同；mask 與
  remap 只有最後一幀不同。兩組都呈現深藍全罩重甲、窄眼縫與黃眼，target 47 將紅色
  腰腿點綴改成冷藍紫。現代稿不賦予未證實身分，只保留可見轉職差異。
- `modern.fdicon.group_049.style_a`：轉職表證實 current portrait 17 → default
  target 49、class 15、移動力增加 1；角色姓名仍未閉合。原版 group 17／49 的
  12 張 frame 逐位元完全相同，因此現代稿直接重用 group 17，不另創造不存在的
  服裝、方向或動作差異。
- `modern.fdicon.group_050.style_a`：角色表與轉職表共同證實這是索爾 current
  portrait 0 使用 item 89 才能選擇的 optional target 50；target class 17、移動力
  增加 2，但職業名稱不由映射自行猜定。原版 12 幀為完整四方向三步態，且沒有與
  0–49 任一幀逐位元相同。現代稿依 FDOTHER #0 正式 DAC 色盤保留深鈷藍長髮、紅色
  披巾／短披風、深綠灰裝甲與四方向輪廓，不新增武器或未證實徽章。
- `modern.fdicon.group_051.style_a`：角色表與轉職表共同證實這是哈諾 current
  portrait 1 使用 item 93 才能選擇的 optional target 51；target class 18、移動力
  不變，職業名稱仍不猜。原版只有 f06–f08 分別與 sprite 402–404 逐位元相同，其餘
  九幀獨立，故不可整組複用。現代稿依正式色盤保留栗紅短髮、深綠灰斗篷式裝甲與
  原版方向性持物遮蔽：f03–f05 不露短刃，正面與反向側面才保留冰藍短刃輪廓。
- `modern.fdicon.group_052.style_a`：角色表與轉職表共同證實這是悠妮 current
  portrait 9 使用 item 90 的特殊 target 52；target class 21、移動力增加 2。原版
  f06 與 f08 逐位元相同，故使用 `source_exact_repeats` 並讓現代兩槽保留相同 PNG
  位元組；其餘十槽獨立。現代稿按正式色盤保留朱紅兜帽／披風、紫灰領口、金扣與
  側面少量深藍髮束，不與 default selector 41 混用。
- `modern.fdicon.group_053.style_a`：角色表與轉職表共同證實這是哈瓦特 current
  portrait 3 使用 item 93 的 optional target 53；target class 18、移動力不變。
  原版只有 f06–f08 分別與 selector 35 對應幀相同，其餘九幀獨立，故不可整組複用。
  現代稿按正式色盤保留栗紅長髮／鬍鬚、灰綠大盾與銀灰短柄斧，四方向固定盾斧側別。
- `modern.fdicon.group_054.style_a`：角色表與轉職表共同證實這是亞雷斯 current
  portrait 4 使用 item 205 的 optional target 54；target class 19、移動力增加 2。
  原版 12 幀皆為獨立圖像。現代稿不猜職業名稱，只按正式色盤保留大型鈷藍矩形
  盔帽／護面輪廓、深藍甲衣、灰藍圓形肩背甲，以及各方向可見的窄銀色直刃。
- `modern.fdicon.group_055.style_a`：轉職表證實 current portrait 5 使用 item 205
  的 optional target 55；target class 19、移動力增加 2，但角色姓名仍未閉合。原版
  12 幀皆為獨立圖像。現代稿按正式色盤保留大型灰綠兜帽／髮罩形構件、鈷藍與
  鋼藍甲衣、灰藍圓形肩背甲與方向性窄銀直刃，不將未證實身分寫入素材名稱。
- `modern.fdicon.group_056.style_a`：轉職表證實 current portrait 6 使用 item 205
  的 optional target 56；target class 19、移動力增加 2，但角色姓名仍未閉合。原版
  12 幀皆有實際像素，且未與 selector 0–55 任一完整圖組逐位元相同。現代稿按
  FDOTHER #0 正式 DAC 色盤保留深紅褐大型束髮、鈷藍／鋼藍甲衣、金色小扣件與
  方向性窄銀直刃，不將未證實身分或職業寫入素材名稱。
- `modern.fdicon.group_058.style_a`：原版 `FDICON.B24` 的 sprite 696–707 與
  sprite 480–491（selector 40）十二幀逐位元相同，透明遮罩與換色遮罩亦完整。
  因此現代清冊保留獨立 selector 58 索引，但十二個輸出位元組精確重用 selector 40；
  不另畫一套原版不存在的差異，也不由索引猜測角色姓名、職業或劇情身分。
- `modern.fdicon.group_059.style_a`：原版 `FDICON.B24` 的 sprite 708–719 與
  sprite 492–503（selector 41）十二幀、透明遮罩及換色遮罩皆逐位元相同。
  現代清冊保留獨立 selector 59 索引，但十二個輸出位元組精確重用 selector 41；
  不另創不同造型，也不由索引猜測角色姓名、職業或劇情身分。
- `modern.fdicon.group_060.style_a`：轉職資料證實 current portrait 10 使用 item 88
  的 optional target 60，class 22、移動力增加 1。原版 sprite 720–731 與
  sprite 504–515（selector 42）的圖像、透明遮罩及換色遮罩全數逐位元相同；
  因此現代清冊保留獨立索引並精確重用 selector 42，不另造原版不存在的外觀差異。
- `modern.fdicon.group_061.style_a`：轉職目標表證實 target 61 為 class 22、移動力
  增加 1。原版 sprite 732–743 與 sprite 516–527（selector 43）的圖像、透明遮罩
  及換色遮罩全數逐位元相同；現代清冊保留獨立索引並精確重用 selector 43。
  目前沒有足夠證據把它綁定特定姓名或劇情身分，故不猜測命名。
- `modern.fdicon.group_062.style_a`：轉職資料證實 current portrait 12 使用 item 92
  的 optional target 62，class 24、移動力增加 1。原版 sprite 744–755 與
  selector 12／44 的圖像、透明遮罩及換色遮罩全數逐位元相同；現代清冊保留
  獨立 selector 62 索引並精確重用 selector 44，不另造不存在的轉職外觀差異。
- `modern.fdicon.group_063.style_a`：轉職資料證實 current portrait 13 使用 item 88
  的 optional target 63，target 表直接記錄 class 20、移動力增加 1。原版十二張
  可見圖皆為獨立圖像，只有透明遮罩及換色遮罩逐幀沿用 selector 13，故不可複製
  selector 13 的金髮可見外觀。現代稿依原版巨大圓頂深色罩體、正面黑色面罩與
  雙亮眼、銀灰側構件及四方向三步態另製十二幀；姓名與正式職業仍不猜。
- `modern.fdicon.group_064.style_a`：轉職資料證實 current portrait 14 使用 item 88
  的 optional target 64，class 22、移動力增加 1。原版 sprite 768–779 與
  selector 46 的圖像、透明遮罩及換色遮罩全數逐位元相同；現代清冊保留獨立
  selector 64 索引並精確重用 selector 46，不另造不存在的轉職外觀差異。
- `modern.fdicon.group_065.style_a`：轉職資料證實 current portrait 15 使用 item 91
  的 optional target 65；target 表直接記錄 class 24、移動力增加 1。原版
  sprite 780–791 與 selector 47 的圖像、透明遮罩及換色遮罩全數逐位元相同；
  現代清冊保留獨立 selector 65 索引並精確重用 selector 47，不另造外觀差異。
- `modern.fdicon.group_069.style_a`：原版 sprite 828–839 為完整四方向三步態，
  十二張可見圖均無整組重用。正式色盤顯示大型灰綠矩形重甲、十字盾、窄銀直劍、
  深紅手套／關節與無可見人臉的平整背甲；現代稿保留這些輪廓，不猜姓名或職業。
- `modern.fdicon.group_070.style_a`：原版 sprite 840–851 為完整四方向三步態，
  三層素材沒有整組重用。正式色盤顯示深海軍藍軟帽、淺灰綠圓形護目鏡／面罩、
  斜持窄銀直劍與深棕斗篷；現代稿保留方向側別，物件語意與身分仍不猜。
- `modern.fdicon.group_071.style_a`：原版 sprite 852–863 為完整四方向三步態，
  可見幀皆為獨立圖像；部分遮罩與後段槽位重用，但不構成可見外觀別名。正式
  色盤顯示大型藍灰角狀／獸首裝甲輪廓、深色面罩、分段軀幹與棕色腰帶；現代稿
  保留這些特徵與換色區，不猜為特定種族、角色或職業。
- `modern.fdicon.group_072.style_a`：原版 sprite 864–875 為完整四方向三步態，
  可見幀沒有整組重用。套用正式色盤後顯示深鈷藍髮／頭巾、藍灰上衣、紅棕靴、
  大型棕色背包與紫灰背負部件；現代稿依彩色參考保留輪廓，不沿用灰階檢視曾造成
  的白灰重甲誤判，也不猜武器或職業。
- `modern.fdicon.group_073.style_a`：原版 sprite 876–887 為完整四方向三步態，
  可見幀皆為獨立圖像，遮罩則與 selector 16 的相對槽位重用。正式色盤顯示棕色
  馬形／騎乘輪廓、直立長槍、淺灰十字盾與紅色鞍具；現代稿保留裝備側別，但不猜
  特定種族、角色或職業。
- `modern.fdicon.group_074.style_a`：原版 sprite 888–890 是三張可見相位，
  891–899 是空白保留槽。正式色盤顯示紫色寬帽、紫袍、深藍領口、紅色腰腹布片、
  棕色圓形背負物與橙棕杖端；現代稿只保留三張正面相位，不補造其他方向。
- `modern.fdicon.group_075.style_a`：原版 sprite 900–902 是三張可見相位，
  903–911 是空白保留槽。正式色盤顯示紅髮／紅披風包覆的側向低矮人形、膚色臉與
  手臂、藍色衣甲及棕／橙／紫部件；現代稿只保留可見輪廓，不猜倒臥原因或場景。
- `modern.fdicon.group_079.style_a`：原版 sprite 948–959 為完整四方向三步態，
  三層素材均無整組重用。正式色盤顯示深橄欖綠寬帽／兜帽、象牙白面罩、深藍
  上衣、灰棕護具與斜持銀劍；現代稿保留背向不露面罩及左右持劍側別，不猜身分。
- `modern.fdicon.group_081.style_a`：原版 sprite 972–983 為完整四方向三步態，
  可見幀均為獨立圖像；部分遮罩在本組內重用。正式色盤顯示深藍甲殼／重甲、
  雙角、中央紅色眼縫、巨大圓肩與暗紅垂布；現代稿保留背面不露眼縫，不猜種族。
- `modern.fdicon.group_084.style_a`：原版 sprite 1008–1019 為完整四方向三步態，
  可見幀均為獨立圖像。正式色盤顯示鮮紅方正重甲、短米灰側角、黃色眼縫、
  橙紅旋紋方盾與直立銀劍；現代稿保留背面大片紅甲及左右裝備側別，不猜身分。
- **核准的共通戰場人物風格（2026-09-05）**：使用者以 selector 000 修訂母稿確認
  統一方向；人物頭部約佔全高 55–60%，身體與腿部短小，保留原版可辨識的髮色、
  頭帶、盔甲、武器、盾牌與主要配色，再施以現代手繪光影。四方向三步態、腳底
  基準與 `24×24` 消費契約不變。禁止改成一般寫實頭身、單純像素放大，或加入原版
  證據沒有的裝備與身分符號。既有候選先做比例、配色與戰場可讀性稽核；不符合者
  逐組重畫，不因風格統一而推倒已符合的素材。
- `modern.fdicon.group_001.style_a`：依上述核准風格重畫第一批擴展候選，保留原版
  栗紅短髮、黑額帶、裸露臉部、斜向灰綠肩胸護具、深棕衣裝與深藍腰腿區；不新增
  原圖沒有的盾、長劍或披風。四方向三步態均維持短腿、相同腳底基準與背向不露臉。
- `modern.fdicon.group_002.style_a`：第二組核准風格擴展保留棕髮、黑額帶、深藍衣甲、
  單一巨大淡灰綠塔盾與窄銀直劍。正面盾在人物後方形成高聳框形，側面遮住大部分
  身體，背向由同一面盾遮住人物；不可把方向差異誤畫成兩面盾，也不新增披風或徽章。
- `modern.fdicon.group_008.style_a`：希莉亞的核准風格修訂保留棕色長髮、紅額帶、
  紫色頸巾、深綠衣甲、深藍腰腿、木弓與箭袋。背向以長髮與箭袋為主且不露臉；
  四方向維持弓與箭袋的裝備側別，不新增劍、盾或魔法效果。
- `modern.fdicon.group_009.style_a`：悠妮的核准風格修訂以正式彩色幀與角色表為準，
  保留朱紅高馬尾、深藍額帶中央藍寶石、藍眼、深鈷藍與紅棕布裝及少量紫灰護具。
  灰階輪廓曾被誤讀為全罩頭盔，該解釋已拒收；背向是大面積紅髮，不是背甲。
  手持小輪廓語意未閉合，不放大成未證實的劍、弓、長杖或盾牌。
- `modern.fdicon.group_013.style_a`：貝克威的核准風格修訂保留巨大金黃長髮、黑額帶、
  深墨綠衣甲、木弓與背側箭袋。側面及背向的大塊黃色區域是長髮，不改畫為兜帽或
  披風；背向不露臉，四方向維持弓與箭袋的裝備側別。
- `modern.fdicon.group_015.style_a`：核准風格修訂保留低矮深藍全罩重甲、黑色面罩、
  黃綠眼光、寬圓肩、灰綠腹帶與鮮紅腰布。側面只露單眼光，背向不露眼光；身分
  未閉合，不新增裸臉、武器、盾牌或職業徽章。
- `modern.fdicon.group_021.style_a`：角色表證實為約拿；核准風格修訂明確保留銀白
  蓬髮、濃密白色大鬍鬚、棕褐長袍與木質短杖，維持聖者老人的辨識特徵。背向只見
  後髮、棕袍與木杖，不加入皇冠、盔甲、翅膀或未證實神聖光效。
- `modern.fdicon.group_022.style_a`：核准風格修訂保留藍黑高馬尾、紅色髮扣、灰綠
  護肩與胸甲、深藍下裝及短靴。原版未證實身分、職業或武器，因此不附加姓名、
  徽章、盾牌或手持武器。
- `modern.fdicon.group_023.style_a`：核准風格修訂保留巨大棕色長髮、深藍額帶與
  甲衣、金色飾帶、暗紅披布及側向短直刃。原版藍黑輪廓不足以證實為箭袋；正式
  重繪不加入弓、箭、箭袋或未證實職業，背向以髮幕遮臉。
- `modern.fdicon.group_087.style_a`：依原版十二幀新增；保留深灰綠寬簷頭盔與
  頸甲、窄暗面部區、藍灰披風及棕色背側長形輪廓。該輪廓用途未證實，正式圖不
  加入箭矢、箭羽、弓弩、徽章或角色姓名。
- `modern.fdicon.group_089.style_a`：依原版十二幀新增；保留方正灰綠金屬頭體、
  深紅圓頂、窄黑面窗、紅與黃綠亮點、圓肩、中央藍色腰布及側向網格圓板。原版
  沒有足夠證據命名單位或內部機構，因此只登錄可見輪廓。
- `modern.fdicon.group_092.style_a` 至 `group_095.style_a`：依各自原版十二幀新增。
  092 保留深藍頭殼、暗面與深紅面部區；093 保留灰綠方體、紅頂與紅色格柵；
  094 保留深藍方體、灰綠頂蓋與青綠燈；095 保留高大深藍方頭、紅色面窗與灰綠
  肩臂。四組均只登錄可見輪廓，不由地圖暫名猜測角色、職業或機械功能。
- `modern.fdicon.group_032.style_a`：2026-09-05 依原版十二幀與核准的 selector 000
  頭身比例重製第二版。保留深藍長髮、紅額帶、裸露臉部、灰綠肩甲、深藍衣裝與
  淡灰綠短刃；第一個誤讀成金髮銀盔的生成候選已拒收，未進 catalog 或私人正式包。
- `modern.fdicon.group_048／050–056／063.style_a`：2026-09-05 依固定原版分離包的
  `group×12+frame` 輪廓重新校稿並正式同步。048 撤回人物編號誤投射後改為三相
  方形全罩機甲；050、052、054、055、063 保留各自全罩裝甲／面窗差異，051、053
  保留盾劍與頭盔／棕髮差異，056 保留棕紅頭罩與長劍。052 依原版重複相位維持
  `f08=f06`。九組均由 24×24 二值透明輸出回算清冊雜湊，私人母稿與逐幀檔同步
  保存；未再使用未版控的 `remake/assets/sprites` 便利輸出作造型證據。
- `modern.fdicon.group_066–068.style_a`：2026-09-05 依相同固定分離包重新校稿。
  066／067 只保留原版三個可見靜態相位，分別維持紫灰大型髮罩配橙金衣甲、灰藍
  蓬髮配墨綠肩衣；068 維持完整四方向三步態、方正全罩制式頭盔、白十字盾與
  直劍，刻意保持普通量產士兵辨識。三組均已同步 runtime prototype 與私人包，
  清冊雜湊由受測工具從 24×24 二值透明輸出重算。
- `modern.fdicon.group_069–073.style_a`：2026-09-05 將已完成戰場比例檢視的核准
  風格候選正式同步。069 保留十字盾重甲與窄劍，070 僅採修正版 3×4 的深藍帽
  面罩劍士，071 保留藍灰角狀重甲，072 保留深藍頭罩與背負輪廓，073 保留馬身、
  長兵器與盾牌。各組均以固定原版分離幀核對四方向，不猜角色／職業名稱；舊版
  070 候選仍拒收且未進正式包。
- `modern.fdicon.group_077–079.style_a`：2026-09-05 依同一核准風格重校。
  077 將偏寫實的高身比重甲稿改為大頭、短身、短腿，保留灰綠全罩盔、
  圓盾與直劍；078 將放大像素稿替換為深藍手繪重甲，保留紅眼、寬肩與
  胸甲 V 形反光，不增加武器；079 統一為大頭短身的深綠兜帽遮面單位，
  保留白色包覆面部、深藍面巾與單一彎刃。三組均以原版十二幀輪廓與
  已登錄彩色證據交叉限制，24×24 二值透明輸出不猜角色身分。
- `modern.fdicon.group_082–083／085–086.style_a`：2026-09-05 將四組方塊化或
  放大像素舊稿重製為核准的大頭短身手繪風格。082 保留無徽矩形盔、素面方盾、
  直劍與紅色下擺；083 保留鈷藍矩形角盔、黃綠眼光、凹槽盾與直劍；085 保留
  深藍低垂角盔、紅眼、素面方盾與直劍，並拒收一版擅自加入菱形徽飾的候選；
  086 保留淡灰綠圓頂盔、深藍肩甲、原尺寸盾面圖形與直立長兵器。四組的
  四方向遮擋關係均依原版十二幀保留，不新增陣營徽章或角色身分。
- `modern.fdicon.group_087／089–090／092–095.style_a`：2026-09-05 完成本輪
  selector 風格收尾。087 保留灰綠寬簷盔、深藍披衣與用途未知的背側短筒，不補畫
  箭羽或弓；089 保留灰綠方體、深紅頂罩、雙色面窗與側向網格圓板；090 保留
  紅褐兜帽、黑面黃綠眼光與無武器輪廓。092 保留深藍頭殼、黑面藍眼光、深紅面巾與
  棕灰肩衣；093–095 分別保留灰綠紅頂格柵、深藍青綠燈方體、深藍紅面窗高方頭。
  七組均從放大像素或不一致舊稿改為核准手繪比例；機械格窗與面窗不人格化，
  不猜功能或身分。
- 九十四組目前共有 1083 個可見 `24×24`、二值 alpha 槽位：89 組一般圖組各有
  12 幀，selector 48、66、67、74、75 各有 3 張靜態相位。除 selector 52 依原版證據保留
  一組相同雜湊外，一般圖組維持 12 個不同 SHA-256；五個靜態圖組各維持三個
  不同 SHA-256，全部列為
  `runtime_candidate`。私人母稿與逐格 PNG 不進公開 Git，公開 catalog 只保留
  可重現契約與雜湊。

### 地圖人物 consumer 現況

#### 三幀靜態 selector 契約（READY）

- 原版 selector 48、66、67、74、75 都只有槽位 0–2 具有可見 `frame`／`mask`；槽位 3–11
  是零可見圖形加全幅 `remap_mask`，不是遺失的行走動畫，也不可生成九張假動作。
- 清冊以 `fdicon_static_sprite_3x24_v1`、`frame_count: 3` 與
  `cycle_policy: static_three_phase` 保存三張可見相位；九個原版空保留槽留在原始
  分離素材與本段證據中，不包成透明 PNG 冒充現代完稿。
- 正式繪製端既有有界索引規則在方向索引超出三幀時回落至 `phase % 3`，因此三張
  靜態相位不依方向切換，也不會因角色方向值而消失。載入器與清冊驗證器只接受
  三張互異、`24×24`、二值 alpha 且 SHA-256 相符的 PNG；任何第四張或重複相位
  都失敗即關閉。
- selector 48、66、67、74、75 的身分與前三幀語意仍分別依自身原版素材記錄；本契約只
  證實素材形狀與正式消費方式，不把未知圖像命名為角色或職業。
- 2026-09-05 勘誤：selector 48 必須取固定原版包的 FDICON 幀 576–578；其可見
  輪廓是方形全罩灰白機甲，不是 DATO 人物 48 的藍帽紅披風王者。舊
  `remake/assets/sprites/fig_048_f00.png` 屬未版控便利輸出，曾混入人物編號投射，
  不再作原版造型證據；正式候選只能由 `asset:FDICON.B24.group_048` 回查。
- selector 74 的原版 sprite 888–890 同樣只有三張連續可見相位；891–899 是
  `frame`／`mask` 空白且 `remap_mask` 全滿的保留槽。它適用相同的
  `fdicon_static_sprite_3x24_v1` 契約，禁止把九個空白槽補成其他方向。
- selector 75 的原版 sprite 900–902 也是三張連續可見相位；903–911 是相同
  形狀的空白保留槽。它同樣適用三相位契約，特殊低矮／倒臥輪廓不可被工具
  自動補成站立人物或四方向行走動畫。

#### 可繪圖組與純目的地重映射圖組

- 2026-09-05 以私人原版固定包 `FDICON.B24`（大小 `624010`、MD5
  `46f793540209a063ea73a5373ca14bf4`、SHA-256
  `7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44`）逐幀檢查
  selector0..95子集、每組 12 幀的 `mask.png` 與 `remap_mask.png`。此子集只有 selector 39
  （原始幀 468..479）及 selector 57（原始幀 684..695）在全部 12 幀皆為
  `source-write=0`、`destination-remap=576`。
- 因此當時盤點的96組子集分成94組可繪地圖人物，以及2組純目的地色彩重映射
  操作。後兩組沒有可供現代重繪的角色輪廓，不得輸出全透明 RGBA 圖後冒稱完成，
  也不得因轉職資料中的 `target portrait=39/57` 猜造人物。原版轉職只改寫人物記錄
  `+0x20` 類別欄位；正式戰場圖組仍由建構時的 `unit+2` selector cache 決定，
  portrait 編號不是 FDICON selector 的全域等價關係。
- 現代主題目前只承諾替換 94 組具來源像素的可繪圖組。selector 39／57 維持原版
  indexed 合成語意；若日後要讓正規化現代 renderer 消費它們，必須另立具型別的
  destination-remap 契約與正常玩家消費端證據，不得塞入目前只接受二值 alpha RGBA
  的 `fdicon_map_sprite_12x24_v1`。
- 2026-09-05 catalog 已涵蓋上述全部 94 組可繪圖組；selector 39／57 的缺席是
  已證實的純重映射契約，不是待補人物圖或素材缺漏。
- 2026-10-19 勘誤：上述96組只是0..95子集，舊文字把它當成FDICON全圖庫不成立。
  固定原始header與現行1680幀loader直接反證；額外可見需求及#259沿本檔範圍勘誤與58。

- `loadModernStoryPortraitSet` 現會同時預檢 catalog 中的地圖人物組：group
  0..95、12 個安全檔名、逐格 SHA-256、`24×24`、二值 alpha、互異雜湊及三週期
  policy，任何一項不符即拒絕整個現代主題。
- `loadGame` 在完整預檢後，才依 catalog 的 `source_group` 逐組以 12 張真彩色圖原子
  取代對應的正規化 `g.sprites[group]`；未登錄 group 不變，忠實主題
  預設路徑也不變。
- 現代主題改用正規化 renderer 疊加真彩圖集與人物，並拒絕原生 indexed 完整幀
  覆蓋；忠實原版主題仍直接消費 `NativeMapSelectorCache`。兩條呈現路徑互斥，
  不量化或混搭；地圖人物目前是 `RUNTIME-E1-PARTIAL`，待正常玩家抽測後再升級。

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
