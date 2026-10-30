# 炎龍騎士團 2 反向工程與重製

## 引言

本專案以合法原版《炎龍騎士團 2：黃金城傳說》作為行為基準，保存它在 DOS
年代的資料格式、介面、音訊與遊戲機制，並以 Go／Ebiten 建立可編輯、可擴充的
潔淨室重製引擎。

我們要做的是重現經典，而不是把玩家困在永無止境的考古工作裡。原版戰役仍是
忠實模式的基準；對話、事件、戰鬥、城鎮、商店、教會、整備、隊伍與存檔則逐步
脫離寫死程式，成為日後能延伸新戰役的資料與規則。

原版程式及美術、文字、音樂等受著作權保護的資產不包含在本倉庫中。玩家必須
自備合法原版；本專案不代表原開發商或權利人。

## 正式第一版下載

[v.1.0.0-20261017](https://github.com/wicanr2/fd2_re/releases/tag/v.1.0.0-20261017) 提供 macOS 通用版 ZIP、Windows x86_64 ZIP 與 Linux x86_64 AppImage，附版本、來源提交、SHA-256 與原生封包自檢紀錄。

公開包只含引擎與可散布資料，遊玩需自備完整合法分離素材。現有單一匯入工具尚不能產生完整可玩包；素材契約與工具範圍見[分離素材規格](docs/knowledge-base/60-editor-separated-assets-spec.md)。30章受控流程與保存例外見[驗收進度](docs/REMAKE-STATUS.md)。HD仍未完成，Windows／macOS未簽署，macOS尚未公證。

開發版將滑鼠移至頂端即可顯示單列工具列，離開後自動隱藏。美術主題與操作介面可分開選擇原版／HD，快捷鍵為 F2／F11；另提供四語 F4、字級 F7、遊戲速度 F8、全螢幕 Alt+Enter 與海報 F10。海報可獨立選擇兩組或關閉，並調整內緣漸層、淡入時間及重播。這些選擇保存在設定檔，不改戰役存檔。戰場檢視角色使用 Home。具備合法的完整 HD 素材包時，故事頭像以
480×480 圖源直接繪製，關閉海報時預設視窗與遊戲區域均為1920×1200；開啟海報後中央維持原比例，兩側另加側欄。浮出列暫時覆蓋頂端，不移動遊戲。故事對話、商店文字與數字、旅館／教會、角色／物品狀態、整備與標題讀檔的穩定畫面採用直接高解析繪製的 Noto Sans CJK 黑體；忠實像素模式保留。整備問句的開收動畫也已接入高解析文字。商店金幣的扣款／入款滾動也直接繪字，出售確認會同步清除被覆蓋的歡迎詞。角色／物品面板的開收與命令滑動，也會同步移動及裁切高解析文字。[戰場角色狀態](docs/figures/hd-battlefield-status.png)只覆蓋面板區域，保留框外現代地圖。戰場人物已接入144×144圖源，沿原生312×192窗格與相同視圖繪製，保留F2與正常走行／指令環。經驗／升級下框的開收動畫也保留HD人物，已用明示章前EXP校準槽驗證正常操作、F2與保存冷讀。可見戰鬥背景與台座圖源已補齊，接入已驗證的共背景、分離背景物理、一般法術、指令24／29及四種召喚流程。物理攻防、法術與召喚的姓名、LV、HP／MP文字直接繪製高解析黑體，戰鬥面板框線與血條也已直接HD繪製；其他介面與完整地形仍待。兩種主題均已驗證首章 JSON 的標題 LOAD 與 F9。
首章原始底圖、前景及寶箱開合圖源已接入手繪HD，包含屋簷遮擋、草土水岸、石路、寶箱與小石塊。正常移鏡頭、F2與保存冷讀已抽驗，開箱呈現另有事件更新後的受控繪圖驗證。地圖資訊窗的細框、A／D及血量數字也直接HD繪製，地形縮圖沿用已登錄的HD來源。未登錄地形與其餘操作介面仍待。

![大頭Q版亞雷斯的實際首章戰場與資訊窗](docs/figures/hd1-ares-q-game.png)

亞雷斯已依原版造型改成更大頭、短身的Q版，戰場與資訊窗共用新版人物圖源。美術與操作介面仍可獨立切換，普通走行、取消及保存讀檔已抽驗。後續小人物依同樣標準逐批修正，索爾、悠妮維持比例參考；可查看[定版比例與144／72／48像素對照](docs/figures/hd1-ares-q-design.png)。其餘人物及轉職形態仍逐批修正，進度見[#286](https://github.com/wicanr2/fd2_re/issues/286)。

![哈諾、鐵諾與哈瓦特的原版及HD1素材對照](docs/figures/hd1-chibi-party-comparison.png)

哈諾、鐵諾與哈瓦特也已接入新版大頭Q版圖源。鐵諾已依原版灰綠布兜帽修正，並保留原版左向被身體遮蔽的直劍；[完整受控戰場與資訊窗](docs/figures/hd1-tino-hood-game.png)可查看這次接線。哈瓦特已再修為成熟五官、收斂眼神與寬厚大鬍子，保留紅褐髮鬚；[完整受控戰場圖](docs/figures/hd1-harvat-mature-game.png)可查看地圖與資訊窗呈現。上圖是已登錄素材的四向及144／72／48像素對照，依序為三人，並非遊戲截圖。地圖與資訊窗已做受控繪圖核對，普通首章另驗完整候選載入與既有角色的切換／保存相容性；三人自然出場與造型由使用者人工確認。全部人物重繪完成後，會再更新完整「原版／HD1」人物總覽。

亞雷斯與哈瓦特的轉職形態也已修回原版裝備：亞雷斯保留人臉、藍軍帽與劍盾，哈瓦特保留成熟大鬍子、斧頭與方盾。可看[原版／HD1局部對照](docs/figures/hd1-native-variants-comparison.png)、[亞雷斯](docs/figures/hd1-ares-variant-game.png)及[哈瓦特](docs/figures/hd1-harvat-variant-game.png)的完整受控戰場與資訊窗。這些畫面以測試用圖組投影核對接線，不代表已驗升階流程或能力值。

![哈諾與貝克威依各自原版頭像重繪的HD1面容](docs/figures/hd1-native-face-reference.png)

哈諾的長鼻、細眼與抿嘴，以及貝克威的平眉、半闔眼與微揚嘴角，分別參考自己的原版大頭像。上圖比較原版頭像與實際144／72／48像素地圖素材，並非遊戲截圖或對話HD頭像驗收；[四向原版／HD1局部對照](docs/figures/hd1-native-faces-comparison.png)另列兩人的裝備與方向。[哈諾](docs/figures/hd1-hano-face-game.png)與[貝克威](docs/figures/hd1-bekwei-face-game.png)的完整受控戰場畫面，可查看地圖與資訊窗接線；測試用圖組投影不代表升階或自然出場驗收。其他角色的面容仍逐人複核，完整人物總覽待全部重繪後更新。

![洛娜與萊汀依各自原版頭像重繪的HD1面容](docs/figures/hd1-knights-face-reference.png)

洛娜的灰綠長髮、杏眼與紅唇，以及萊汀的褐髮頭帶、長鼻與方下巴，分別依自己的原版大頭像重繪。兩人的騎士形態保留原版長兵器、藍鋼甲與方向遮蔽，地圖及資訊窗共用新版圖源。可看[四向及尺寸對照](docs/figures/hd1-knights-comparison.png)、[洛娜](docs/figures/hd1-rona-variant-game.png)與[萊汀](docs/figures/hd1-laiting-variant-game.png)的完整受控戰場圖。面容對照是地圖美術來源比較，戰場圖使用測試圖組投影，不代表升階、自然出場或對話HD頭像驗收。

![索爾與悠妮特殊形態的原版及HD1面容對照](docs/figures/hd1-heroes-face-reference.png)

索爾與悠妮的特殊形態已恢復各自的人臉。索爾保留深藍髮、紅頭帶與收斂眼神；悠妮保留柔和笑容、紅髮及藍金頭飾，衣甲與空手造型沿原版。可看[四向及尺寸對照](docs/figures/hd1-heroes-comparison.png)、[索爾](docs/figures/hd1-sol-variant-game.png)與[悠妮](docs/figures/hd1-unni-special-game.png)的完整受控戰場圖。面容圖是原版頭像與地圖素材比較，戰場圖使用測試圖組投影，不代表自然轉職流程或對話HD頭像驗收。

![戴盔士兵與蒙面劍士的原版及HD1四向對照](docs/figures/hd1-enemies-comparison.png)

戴盔士兵恢復原版十字紋方盾、直劍與灰綠盔甲；蒙面劍士保留遮眼金屬帶、深藍面巾與斜持單劍，去除舊稿新增的骷髏臉。可看[各自原版頭像與實際尺寸](docs/figures/hd1-enemies-face-reference.png)、[士兵戰場](docs/figures/hd1-enemy-076-game.png)及[蒙面劍士戰場](docs/figures/hd1-enemy-079-game.png)。戰場圖使用原生名冊與地圖的受控敵軍群，地圖和資訊窗共用新版圖源，不代表自然章節流程驗收。其他人物仍逐組複核，完整總覽待全部重繪後更新。

![蓋亞原版與新版HD1四向及尺寸對照](docs/figures/hd1-gaia-q-comparison.png)

蓋亞採用上圖的先前HD1版本定案，保留灰綠硬甲、藍色面罩與大頭短身。兩份加重候選不採用。[完整受控戰場及資訊窗](docs/figures/hd1-gaia-q-game.png)、[普通首章遊玩畫面](docs/figures/hd1-gaia-q-normal.png)均使用這版；十二姿態、主題切換與保存冷讀已抽驗。全部125組已做第一輪造型審查，身份與裝備偏移由[#287](https://github.com/wicanr2/fd2_re/issues/287)修正，其他比例由[#286](https://github.com/wicanr2/fd2_re/issues/286)續作；全部完成後才更新完整「原版／HD1」人物總覽。

![系統確認的四組原版與HD介面／美術](docs/figures/hd1-system-confirmations.png)

END、離開、目前戰況保存／讀取與全軍移動的確認視窗已接HD字型、細框及店員頭像。人物隨美術主題，文字與按鈕隨操作介面；問句保留原文，YES／NO選項支援四語。上圖為同一受控戰場的完整畫面縮圖，保留尚未HD的地形與前版細框；現行對話框採古金色，普通END取消返回也已抽驗。HD頭像目前採閉口靜態呈現，造型由使用者確認。來源與限制見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![HD地圖總覽：六張代表戰場](docs/figures/hd1-map-overview.png)

六張代表地圖由同一版遊戲受控擷取，完整保留介面與尚未替換的像素素材。戶外草、土、沙、水與石路已換新版手繪底材，亞雷斯採大頭短身Q版；此總覽保留前批蓋亞，造型修訂見上方說明，洞窟、樹木與建築仍含前批素材。人物採章節測試名冊，因此首章可見蓋亞，不代表正常戰役的加入時點。圖源槽數包含事件有效時的上界，不代表整張地圖的完成比例；完整缺口見[地形覆蓋資料](docs/data/fd2-hd1-terrain-coverage.json)。

![HD1新版戶外地形的普通遊玩畫面](docs/figures/hd1-painted-ground-game.png)

上圖是首章標題LOAD後，正常選取亞雷斯、走行一格再取消的完整視窗。新版地形已接入遊戲，原版／HD介面與美術可獨立切換，保存及新程序讀檔已抽驗。重繪方向採用[已同意的風格原型](docs/figures/hd1-terrain-redesign-style.png)；原型為概念圖，洞窟與其他地形後續沿同一方向續作。

![HD1跨地圖同源地形與前景](docs/figures/hd1-terrain-reuse-map1.png)

上圖為第二張地圖的受控繪製，人物位置用來確認屋頂前景遮擋。三層原生圖源完全相同的木屋、木材及植被已共用現有HD圖片；藍屋頂、石牆與部分介面仍為像素。完整地形缺口由[地形覆蓋資料](docs/data/fd2-hd1-terrain-coverage.json)追蹤，畫面不代表整張地圖已完成HD。

![HD1洞窟地板與石壁](docs/figures/hd1-cave-floor.png)

洞窟石壁、外緣、部分地板與明暗變體已接入手繪HD，原生資料完全相同的其他地圖共用既有圖片，透明範圍保留原生背景。上圖為第二十五張地圖與第25章單位的受控繪製，人物位置與相機經明示設定；未替換的地板、熔岩與部分介面仍為像素，不代表整張洞窟已完成HD。技術範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![獨立介面與美術控制的浮出工具列](docs/figures/hd1-independent-toolbar.png)

上圖為重製端受控城鎮的實際操作畫面，完整縮圖保留海報、工具列與設定選單。海報已接入遊戲；城鎮行走人物仍為像素素材。完整地形與其餘介面依 [HD1 里程碑](https://github.com/wicanr2/fd2_re/milestone/1)繼續完成，造型與實機操作由使用者人工驗證。HD2、HD-2D 留待後續。

![HD1古金雙層對話框的普通父王對話](docs/figures/hd1-gold-dialogue-game.png)

HD1對話框採用古金色雙層框線，保留深藍底、原位置、頭像與文字區；故事開收框及既有提示也已接入。上圖是正常START到父王對話的完整視窗，四組美術／介面切換與續句已抽驗。框線隨F11介面，頭像仍隨F2美術主題。其他介面與集中人工確認仍待，範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![四語系狀態到期提示的HD背景、頭像與文字](docs/figures/hd-transient-expiry-locales.png)

六種狀態到期提示已接上HD背景、480頭像與清晰文字，並支援四語。四格為Linux受控狀態測試的完整遊戲畫面半縮，角色欄位與位置經明示建構；本圖記錄前版框線，現行古金框見上方實際畫面。一般玩家取得狀態後的抽驗與完整HD介面仍待，來源及範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![普通攻擊中的HD戰鬥面板框線與HP／MP條](docs/figures/hd-battle-panel-chrome.png)

戰鬥面板已接上高解析框線與HP／MP條，保留原版位置、當次色盤和逐步扣血。這張是既有首章來源槽經正常鍵盤攻擊的完整遊戲截圖，前景光效依原順序遮擋上方面板。F2、自然返回及保存重開讀檔已抽驗；其他介面與造型人工確認仍待。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![首章正常取景的HD水面、沙地與岸線](docs/figures/hd-shared-grass.png)

這是首章普通讀檔、移鏡頭及F2切回HD後的完整遊戲畫面，水、沙與草岸沿原輪廓合成。其他地形與HUD框線仍含像素素材；各章普通玩家驗收仍分別記錄。來源與範圍依[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![首章正常取景的HD石路與土脊](docs/figures/hd-terrain-relief.png)

這是同一首章的普通方向鍵取景與F2切回HD畫面，石路、草地陰影與土脊沿原生輪廓重繪。完整截圖包含遊戲原有的主題提示，沒有裁切或移除介面。

![首章正常取景的HD寶箱與小石塊](docs/figures/hd-terrain-props.png)

金／銀箱、小石塊與房屋位於同一普通取景，畫面完整保留遊戲提示與介面。開箱圖源已通過既有事件更新後的繪圖與F2；普通取寶仍待集中抽驗。

蓋亞的故事頭像與四方向戰場小人已依原版壯碩機兵重新設計，採用灰綠厚甲與藍色結構。大型戰鬥的待機與拳擊也已接入HD圖組，驗證範圍見下方戰鬥畫面與現代主題規格。
素材要求及驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。此功能尚未加入上述正式第一版。

![HD1地圖資訊窗的細框、數字、地形與人物](docs/figures/hd-map-hud-panel.png)

資訊窗保留原位置、左右切換與血量覆蓋順序，人物和地形縮圖使用同一主題來源。上排是首章普通讀檔與F2切回HD後的畫面，下排是已登錄地形縮圖及血量溢位的受控補樣。四格皆保留完整1920×1200畫面並半縮，沒有補畫；不代表完整章節通關或原版對拍。來源與範圍見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`map_hud_hd_followup_20261026`。

![戰場資訊的像素與HD主題、展開及收合](docs/figures/hd-system-info.png)

戰場資訊窗的細框、章回數、金錢與勝敗條件已接入HD，開收時保留HD地圖。上排是同份存檔、資訊值一致的重製像素與HD兩種主題，下排是明示數值的受控開收測試；完整畫面半縮，沒有補畫，不代表DOS原版對拍或整章通關。來源見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`system_info_hd_followup_20261026`。

![戰場資訊的繁中、簡中、日文與英文實際HD畫面](docs/figures/hd-system-info-locales.png)

資訊窗已接上四語勝敗條件與標籤。四格取自同份章前存檔的普通開窗操作，完整視窗半縮，沒有補畫；像素主題也已抽驗，繁中仍保留原始圖字。正常切換、返回與保存冷讀通過，完整HD1與平台驗收仍待。來源見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`system_info_locales_followup_20261026`。

![四種回復術的正常操作HD畫面](docs/figures/hd-heal-normal.png)

回復術已接入HD光環與綠色數字。四格依序為13～16，取自既有強化存檔經標題讀取、正常選人與施法的實際視窗，完整畫面半縮。取消、扣魔力、演出途中F2及保存冷讀已驗；來源全員滿血，這組只確認演出與滿血上限。受傷後的回復量、完整章節與集中人工驗收不由本圖證明。來源及範圍見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`heal_normal_gui_followup_20261026`。

![HD傳送前導、收縮、逐列轉場與增援入場](docs/figures/hd-unit-present.png)

傳送與增援入場已接入HD光效及人物，保留原有消失、逐列轉場與出現順序。四格取自受控隊伍、位置與道具資料的實際遊戲繪製，完整畫面半縮；地形與邊框仍含像素素材。F2中途切換及原交易已驗，正常玩家與集中人工驗收仍待。來源、近似與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`unit_present_hd_followup_20261024`。

![故事角色入場與直接出現後返回HD世界](docs/figures/hd-story-unit-present.png)

原生故事場景已接入HD人物與已登錄地形，入場光效結束後繼續顯示HD世界。上圖為受控故事／戰場資料的完整遊戲畫面半縮，並非正常通關。驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`story_map_hd_followup_20261024`。

![故事人物走位、轉彎與跟隨移鏡的HD實際畫面](docs/figures/hd-story-motion.png)

故事人物走位、轉彎與插值移鏡也已接上HD，沿既有事件控制器推進。四格依序為往東、往北、轉彎及跟隨移鏡，皆為受控測試的完整遊戲畫面半縮。未登錄地形保留像素圖源，其他背景、特殊演出與集中玩家驗收仍待。繪製及主題切換不改寫故事進度，範圍依同份收據的`story_motion_hd_followup_20261024`。

![HD徑向轉場的起始、末階、加亮與返回世界](docs/figures/hd-indexed-transition.png)

故事的徑向轉場與加亮演出也已接入HD。四格依序為起始、徑向末階、加亮與返回HD世界，取自受控幾何測具的完整遊戲畫面半縮。調色採明示的HD近似，並非正常章節通關或原版對拍；其他特殊演出與集中玩家驗收仍待。來源與範圍依[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`indexed_transition_hd_followup_20261024`。

![戰鬥與故事列位移演出的兩張HD畫面](docs/figures/hd-row-shake.png)

交替列位移演出已接入HD，人物、已登錄地形與資訊窗人物一起移動。上排是戰鬥，下排是故事，皆為受控測試的完整遊戲畫面半縮；草地、水岸與HUD框線仍保留像素素材。此圖不代表正常章節通關，來源與範圍依同份收據的`rowshake_hd_followup_20261025`。

![地圖23特殊背景的HD捲動與調色](docs/figures/hd-map23-stage.png)

地圖23的特殊背景已接入1872×1152手繪紋理，保留原生循環捲動；人物與資訊窗人物也隨當次色盤變暗。上排是起始與捲動，下排是調色中及返回。四格取自實際遊戲繪製的完整畫面半縮，使用受控角色與位置，並非正常章節通關；其他地形與HUD框線仍含像素素材。來源、近似與驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`special_scene_hd_followup_20261025`。

![天空之鑰合成、近景縮遠與返回的 HD 畫面](docs/figures/hd-sky-key.png)

天空之鑰的五色球、金鎖近景、閃白與返回地圖已接入 HD。四格依序為合成、近景、尾段及返回，取自受控測試的完整遊戲畫面半縮。人物、金鎖與元件使用手繪素材，地圖20地形與介面框線仍是像素；金鎖縮遠採美術近似，保留原生演出順序。此圖不代表正常章節通關，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`sky_key_hd_followup_20261025`。

![物品回復、傷害、特殊效果與未命中的 HD 演出](docs/figures/hd-item-map.png)

物品演出已接入 HD 人物、光效與數字。上排為 AI 回復與玩家傷害，下排為特殊效果與未命中，取自受控測試的完整重製畫面半縮。玩家回復道具維持既有直接結算；部分地形及介面仍用像素。此圖不代表正常章節通關，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`item_map_hd_followup_20261025`。

![第9號法術在地圖23保留同次HD背景](docs/figures/hd-player9-stage.png)

第9號法術已保留同次手繪背景，四格依序為調色、命中數字、未命中與返回地圖。畫面取自受控測試的完整重製畫面半縮；綠色背景、人物與數字使用 HD，城堡地形及介面框線仍為像素。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`player9_stage_hd_followup_20261025`，不代表正常章節通關。

![回復法術的HD背景、停格、群體數字與返回](docs/figures/hd-heal-stage.png)

回復法術已接上同次 HD 背景與徑向調色。四格依序為前段、中段停格、群體回復數字與返回，取自受控測試的完整重製畫面半縮。人物、綠色背景與數字使用 HD，城堡地形與介面框線仍為像素。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`heal_stage_hd_followup_20261025`，不代表正常章節通關。

![移位雙段演出保留同次HD背景](docs/figures/hd-unit-stage.png)

移位物品的消失、收縮、出現與返回戰場已接上同次 HD 背景。四格依序為消失前導、收縮、出現尾段與返回，取自兩組受控測試的完整遊戲畫面半縮；人物、光效及綠色背景使用 HD，城堡地形與介面框線仍為像素。F2 切換、扣款與落點已驗證，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`unit_present_stage_hd_followup_20261025`，不代表正常章節通關。

![戰後物品與金錢訊息保留HD背景](docs/figures/hd-reward-stage.png)

戰後成功訊息的開框、等待與關框已保留同次 HD 背景，結束後回到目前戰場。四格依序為物品訊息、金錢訊息、關框與返回，取自受控測試的完整遊戲畫面半縮。文字、頭像與綠色背景使用 HD。本圖保留先前原生框線，現行成功訊息細框見上方提示框總覽；城堡地形仍為像素。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`reward_message_stage_hd_followup_20261025`，不代表正常章節通關。

![物品加成與指令消耗字樣的HD呈現](docs/figures/hd-item-row-labels.png)

物品列表的+AP／+DP／+HP／+MP與指令頁的-MP已直接HD繪字，沿用清晰狀態標籤、血條及細框，保留原位置與滑入滑出裁切。四格為受控流程的完整遊戲畫面半縮；數字與頭像沿既有HD來源。分類／裝備圖示的現況見下圖，其他介面仍待。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![HD1物品分類、裝備與角色狀態圖示](docs/figures/hd-item-panel-icons.png)

分類、裝備與角色狀態圖示已接入HD1，保留原版位置、選取與開收框。上排是首章普通讀檔後的索爾狀態及悠妮指令頁；下排是分類／狀態補樣與滿背包物品選擇的受控畫面。四格均保留完整畫面並半縮。普通攻防、F2、保存與重開讀檔已抽驗；完整HD1的其他地形、HUD與介面仍在整合，尚未發行新版。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![滿背包確認選項的四語HD字型與細框](docs/figures/hd-full-inventory-confirmation-locales.png)

滿背包確認選項已接四語黑體與HD細框，保留原按鈕位置、選中節拍及開收順序。四格為受控測試的完整遊戲畫面半縮，這組只核對選項翻譯與呈現，問句沿既有來源。其他確認流程、圖示與完整介面仍待，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![滿背包獎勵對話的HD細框與既有物品選擇流程](docs/figures/hd-full-inventory-reward-chrome.png)

滿背包獎勵的問句、放棄回覆與取消選擇對話已採用HD細框，保留原版位置與開收框流程。四格為受控測試的完整遊戲畫面半縮，頭像及文字沿既有圖源。本圖保留前版原生YES／NO，現行四語選項見上圖；共用選擇器細框見上圖；圖示與城堡地形仍為像素，完整介面及集中人工確認仍待。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![滿背包獎勵的HD問句、回覆與物品選擇](docs/figures/hd-full-inventory-reward.png)

滿背包領取獎勵時，問句、放棄回覆與物品選擇已接上 HD 字型、頭像及背景。四格依序為問句、放棄回覆、丟棄選擇器與取消後回覆，取自受控測試的完整遊戲畫面半縮。本圖記錄先前原生框線，現行對話細框見上圖；本圖的YES／NO為前版呈現；共用三區細框見上圖；介面圖示與城堡地形仍為像素。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`full_inventory_reward_hd_followup_20261025`，不代表正常章節通關。

![蜜蒂音速刃的普通選目標、像素與HD呈現及戰場返回](docs/figures/hd-miti-sonic-blade-normal.png)

音速刃已接上重製的像素與 HD 主題。四格為普通選目標、施放中的像素主題、HD斬擊與返回戰場，皆保留完整視窗並等比例縮小。同一次施放的不同相位，不是DOS原版對拍。章前來源以正式加入與升級程式建構，帶有明示強化政策；章內經普通鍵盤完成施放、F2與保存冷讀，不代表自然招募或整章通關。取得來源見[音速刃證據](docs/data/ida/fd2_command30_player_acquisition_20261025.json)，畫面與驗證範圍見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`command30_normal_gui_followup_20261026`。完整戰鬥HD與集中造型確認仍待。

## HD2 設計基準

後續HD2戰場沿下列斜角站姿人物與手繪地形設計，保留既有HD及原版主題。操作介面與人物主題獨立，海報從單列浮出工具列選擇；海報邊緣可漸層銜接戰場，並調整淡入時間。

![第一組HD海報的漸層概念：索爾與悠妮朝右，空魔神與四天王在右側](docs/figures/fd2-hd2-poster-blend-set1.png)

![第二組原稿重繪海報的漸層概念](docs/figures/fd2-hd2-poster-blend-set2.png)

以上是已選定美術的瀏覽器概念原型，尚未接入正式遊戲。第二組依使用者提供的原稿重繪，並非原始掃描。下載儲存庫後，可用瀏覽器開啟[海報漸層預覽](docs/figures/hd2-posters-preview.html)，調整邊緣寬度與淡入時間。未修改原稿、提示詞及驗證範圍見[設計清冊](docs/data/fd2-hd2-concept-design-20261024.json)與[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

## 忠實像素模式與多語系

| 繁體中文 | 简体中文 |
|---|---|
| ![忠實模式繁體中文商店](docs/figures/faithful-shop-zh-Hant.png) | ![忠實模式簡體中文商店](docs/figures/faithful-shop-zh-Hans.png) |
| English | 日本語 |
| ![忠實模式英文商店](docs/figures/faithful-shop-en.png) | ![忠實模式日文商店](docs/figures/faithful-shop-ja.png) |

以上為重製引擎的實際商店畫面，保留原版像素美術。多語系由重製引擎提供，並非 DOS 原版執行檔新增語言；可按 F4 切換。四張圖使用相同商品與金幣狀態，擷取來源見[截圖紀錄](docs/data/fd2-readme-locales-20261018.json)。

| HD 旅館保存回覆 | HD 教會復活確認 |
| --- | --- |
| ![HD旅館保存回覆](docs/figures/hd-hotel-saved.png) | ![HD教會復活確認](docs/figures/hd-church-revive.png) |
| HD 角色狀態 | HD 商店裝備 |
| ![HD角色狀態](docs/figures/hd-character-status.png) | ![HD商店裝備](docs/figures/hd-item-equipment.png) |
| HD 整備選人 | HD 標題讀檔 |
| ![HD整備選人](docs/figures/hd-preparation-roster.png) | ![HD標題讀檔](docs/figures/hd-title-load-slots.png) |
| HD 城鎮出發開框 | HD 整備確認收框 |
| ![HD城鎮出發開框中的問句](docs/figures/hd-preparation-opening.png) | ![HD整備確認收框中的問句](docs/figures/hd-preparation-confirmation-closing.png) |
| HD 扣款中的數字局部 | HD 商店出售確認 |
| ![實際視窗的HD扣款數字局部](docs/figures/hd-gold-debit-region.png) | ![HD商店出售確認](docs/figures/hd-shop-sale-confirmation.png) |

以上為開發版的實際遊戲視窗。動態文字與數字直接以高解析繪製，頭像、背景與圖示保留像素素材；操作與保存驗證見[字型收據](docs/data/fd2-modern-hd-fonts-20261018.json)。整備截圖接續具有攻略建構祖先的存檔，不代表自然難度通關。扣款圖是實際視窗的288×54金幣區域，數字仍在滾動。[命令面板轉場局部](docs/figures/hd-status-command-transition-region.png)顯示移動中的HD文字，並非穩定頁。轉場截圖的選項框仍在移動；未列固定畫面、其他轉場與圖示內字樣仍待接入。

![主題工具列、HD城鎮與商店，以及既有HD隊伍頭像](docs/figures/hd-town-shop-toolbar.png)

三種城鎮與三種商店已接入手繪HD背景，店員也採HD造型。既有隊伍HD頭像已接到商店裝備面板，隨開窗動畫移動與裁切。上圖是測試場景經正式 Game.Draw 與最終GPU合成的縮圖；選單框、游標及交易效果保留原圖樣。工具列滑鼠／F2／四語切換已驗證，正常玩家與人工造型確認依全部HD完成後集中抽驗。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。圖中固定列為較早擷取，現行浮出列見下圖。

![單列浮出工具列的繁中、簡中、日文與英文](docs/figures/hd-floating-toolbar.png)

上圖為受控城鎮經正式遊戲更新與繪製、真實滑鼠及快捷鍵操作後的四語工具列畫面，完整遊戲只做等比例縮圖。頂端喚出、自動隱藏、說明頁阻擋與保存保持已驗證；加速、介面獨立切換、HD2及海報控制仍待串接。來源見[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`floating_toolbar_followup_20261025`，不代表完整正常章節或平台驗收。

![四種HD召喚演出](docs/figures/hd-common-summons.png)

由左上順時針為熾天使、風妖精、暗邪鬼、破壞神，新圖已接入正式繪製層。總覽由修改隊伍與指令資料的建構測試取得，是最終合成器 GPU 回讀的縮圖，尚不代表一般玩家流程驗收。此圖擷取時的施法者、背景與介面仍含像素素材。來源及驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![四語系的HD召喚、森林背景與草地台座](docs/figures/hd-summon-battle-scenery.png)

四種召喚也已接入手繪戰鬥背景與台座，保留入場滑動、尾段變色及F2切換。四格取自Linux四語受控測試的完整Game.Draw半縮圖，明示選用既有森林圖塊，面板沿測試隊伍名稱；框線、血條與地圖尾段仍含像素素材。正常玩家、集中人工與平台抽驗仍待，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`compound_scenery_followup_20261024`。

![召喚演出的HD施術者](docs/figures/hd-summon-caster-phases.png)

四種召喚的入場、施術動作與退場已接入已登錄的角色HD圖源，保留原有節拍與F2切換。
此圖取自修改class19／BattleFig4測試隊伍的最終GPU，面板仍顯示來源角色名稱；
不代表亞雷斯可在正常流程施放召喚。此圖只涵蓋這組施術者；背景與介面仍含像素素材。
驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![首章雙方HD普通攻防](docs/figures/hd-ch01-physical-attack.png)

亞雷斯與強盜的待機、攻擊及反擊已使用手繪HD圖源。以上取自首章普通攻防的實際遊戲視窗；
背景與戰鬥面板仍含像素素材。操作、保存冷讀及圖源核對範圍見
[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![索爾HD普通攻防](docs/figures/hd-sol-physical-attack.png)

索爾的待機、蓄力、斬擊與收勢也已接入手繪HD圖源。以上是首章正常END接敵後，
索爾收勢與強盜反擊的實際視窗；沒有改寫隊伍、血量或移動力。背景與面板仍含像素素材，
驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![悠妮HD普通攻防](docs/figures/hd-yuni-physical-attack.png)

悠妮的待機、抬杖、轉身與收勢已使用手繪HD圖源，保留原版紅髮、紫灰護具與金色披布。
以上取自首章正常選取悠妮攻擊的實際視窗，左為展臂，右為收勢；背景與戰鬥面板仍含像素素材。
操作、圖源與已知裁切範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![森林小屋背景接入HD近戰的實際遊戲畫面](docs/figures/hd-battle-forest-background.png)

森林小屋背景與草地台座已分別使用1920×600及924×252圖源，接入共用背景物理戰鬥的預備、主攻、反擊與淡出。上圖是明示換用地形索引與攻防結果的測試場景，取自正式 Game.Draw 最終合成的完整遊戲區縮圖；不是一般遊玩或原版對拍截圖。上圖保留當時的像素面板，現行面板文字已接入高解析字型。更多背景、台座與一般法術接線見下方圖組，地圖尾段與其他效果仍待完成。手繪山線、樹形及材質屬現代詮釋，來源與驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![森林內部與海岸背景、台座的HD實際戰鬥畫面](docs/figures/hd-battle-forest-coast.png)

森林內部與海岸也已接入手繪HD背景及透明台座，沿用原有攻擊、反擊、進退場與淡出順序。
左右取自受控地形索引與固定攻防結果的實際遊戲畫面，完整縮圖未裁切；面板仍為像素素材，正常玩家與集中人工抽驗仍待。
原稿、材質近似與驗證範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`forest_coast_surface_followup_20261024`。

![山地石台、裂谷、冰格與木橋的HD戰鬥畫面](docs/figures/hd-battle-scenery.png)

固定原包的可見戰鬥背景與台座圖源已補齊，接入共背景、分離背景物理及指令24／29的已驗流程。四格是受控地形與攻防結果的完整Game.Draw半縮圖，右下為木橋退場；面板沿用測試槽名稱與像素素材。原有空白來源維持空白，特殊尺寸與透明孔洞保留。構圖、筆觸與台座輪廓屬現代詮釋，正常玩家及集中人工／平台抽驗仍待。來源、限制與重跑入口見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`remaining_scenery_followup_20261024`。

![繁中、簡中、日文與英文的HD戰鬥面板文字](docs/figures/hd-battle-panel-text.png)

物理戰鬥的姓名、LV、HP／MP標籤及數字已直接採用六倍Noto黑體，保留原有框線、血條與逐步扣血。四格取自Linux四語受控攻防的完整Game.Draw半縮圖，語系標籤在遊戲區域外；不是一般玩家或原版同狀態對拍。完整HUD美術及集中人工／平台抽驗仍待，來源與驗證依[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`battle_panel_text_followup_20261024`。

![四語系召喚返回地圖後的HD人物與數字](docs/figures/hd-compound-map-tail.png)

四種召喚返回地圖後也會保留HD人物、已登錄地形與數字，回復光環及角色閃光沿原節拍呈現。四格取自Linux四語受控流程的完整Game.Draw半縮圖，使用測試隊伍與邊界位置；不是一般玩家或原版對拍。其餘地形、HUD與尚未重繪的光效仍含原生素材，集中人工與平台抽驗仍待。來源及範圍依[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的compound_map_tail_followup_20261024。

![狀態法術的HD圓環、黃綠與淡藍菱形](docs/figures/hd-modifier-tail-effects.png)

狀態法術的圓環、黃綠與淡藍菱形已接入手繪HD圖源，保留原節拍、角色閃光、結果數字與主題切換。三格為Linux三條受控流程的完整Game.Draw半縮圖，使用原始敵方紀錄建構位置、職業與狀態，固定種子7與地圖時鐘；非一般玩家或原版對拍。部分地形與HUD仍使用原版素材，玩家23目前只驗調色前導。來源及重跑見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)與[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的modifier_tail_art_followup_20261024。

![四語系召喚地圖尾段的HD光柱、光環、冰晶與交叉斬痕](docs/figures/hd-compound-tail-effects.png)

四種召喚返回地圖後的光效已接入手繪HD圖源。四格取自四語受控邊界場景的完整Game.Draw半縮圖，保留原本的畫面邊界裁切；部分地形與HUD仍為像素素材。光形、材質、羽化與亮度是現代美術近似，正常玩家及人工造型仍待集中確認。來源與重跑見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)及[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的compound_tail_art_followup_20261024。

![四語系的一般法術HD背景、台座與效果](docs/figures/hd-ordinary-battle-scenery.png)

一般法術也已接入手繪戰鬥背景與台座，施術及退場的查表變色採用既有HD四色加權近似。四格為Linux四語受控流程的完整Game.Draw半縮圖，角色圖源與面板沿測試隊伍；框線、血條與部分光效輪廓仍含像素素材。原生模式保留精確查表，正常玩家及集中人工／平台抽驗仍待。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)與[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`ordinary_scenery_followup_20261024`。

![繁中、簡中、日文與英文的HD法術面板文字](docs/figures/hd-command-panel-text.png)

法術、分段打擊與四種召喚的戰鬥面板也已接入高解析黑體，文字隨當幀色盤、扣血與捲動呈現。四格為Linux受控測試的完整Game.Draw半縮圖，人物圖源與面板名稱沿測試隊伍，部分背景、框線與血條仍為像素。正常玩家、集中人工與平台抽驗仍待；來源及範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)與[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`command_panel_text_followup_20261024`。

![非共背景戰鬥的HD人物捲入](docs/figures/hd-physical-scroll.png)

分離背景物理演出已接 HD 人物、森林／海岸背景與台座，保留左右捲動、連射與反擊。
四格依序是退場、兩個捲入階段與主攻，取自明示地形與攻防結果的完整 Game.Draw 縮圖。
三張藍色過渡背景也已採 HD 圖源，沿用原有遮罩歷史、位移與裁切；介面仍為像素素材。手繪條帶與色彩屬現代詮釋，正常玩家與集中人工抽驗仍待。
下方希莉亞的正常射擊抽驗範圍保持，契約與限制見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![希莉亞HD普通射擊](docs/figures/hd-silia-physical-attack.png)

希莉亞的待機、上弦、瞄準、射擊與飛矢，以及敵方劍盾兵，已接入手繪HD圖源。
上圖是第三章普通讀檔、移動待機接敵後的實際射擊畫面。兩次射擊、F2與章節邊界保存冷讀已抽驗；
這份截圖來源的接敵回合使用忠實模式，背景與面板仍為像素素材。來源槽祖先的清敵政策、共享姿勢與驗證範圍見
[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![騎乘劍士劍光與弓手放箭的HD測試畫面](docs/figures/hd-mounted-archer-battle.png)

左為騎乘劍士的藍色劍光，右為弓手放箭。圖組也保留原有飛矢與透明停拍。此圖取自明示換用角色圖源的測試場景最終GPU縮圖，面板仍沿用來源角色名稱；背景、面板與受擊閃色保留原圖樣。正常玩家與人工造型確認仍待全部HD接入後集中抽驗，來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![雙刀武者斬擊與翼龍槍兵突刺的HD測試畫面](docs/figures/hd-ninja-winged-battle.png)

雙刀武者的兩次斬擊與藍色斬光、翼龍槍兵的飛行突刺及透明停拍已接入HD圖源。上圖為測試場景的實際最終GPU縮圖，面板沿用來源角色名稱；背景及受擊閃色仍含像素素材。關鍵姿勢與尾光共用、突刺光寬度等美術近似，保留在全部HD接入後集中人工確認。來源與限定驗證見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![上勾拳、短杖星芒與迴旋打擊的HD測試畫面](docs/figures/hd-monks-wand-battle.png)

左為深藍拳師上勾拳，中為短杖命中星芒，右為灰綠拳師迴旋打擊。此圖取自明示換用圖源的測試場景最終GPU縮圖，面板仍沿用來源角色名稱；背景及受擊閃色保留原圖樣。共用姿勢、袍身比例、光弧高度與碎光形狀的差異已記錄，待全部HD接入後集中人工確認，範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![兩組劍士與重斧人物的HD測試畫面](docs/figures/hd-knights-axeman-battle.png)

左為藍披風劍士迴旋斬，中為重斧人物劈擊，右為白馬尾劍士橫斬。此圖取自明示換用圖源的測試場景最終GPU縮圖，面板仍沿用來源角色名稱；背景及受擊閃色保留原圖樣。共用姿勢、人物與武器比例、光弧及碎光形狀的差異已記錄，待全部HD接入後集中人工確認，範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![兩種劍士分段打擊的HD測試畫面](docs/figures/hd-battle-split-strikes.png)

兩種劍士的分段打擊、森林／海岸場景及原規則適用的台座已接入HD，保留原有蓄光、光束、光柱與多目標續行。四格取自明示地形與固定亂數的完整Game.Draw縮圖，上排為指令24的蓄力與命中，下排為指令29先後兩個目標的場景。面板沿用來源角色名稱，介面與捲動過渡遮罩仍為像素素材。MP／HP標記及透明停拍保持，正常操作與人工造型確認待全部HD接入後集中抽驗；來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![斧盾兵與兩組騎兵的HD測試畫面](docs/figures/hd-shield-riders-battle.png)

左為斧盾兵劈斬，中為長戟騎兵揚馬後的下劈，右為白馬槍騎兵突刺。圖組包含蓄力、轉槍及收勢，保留原有節拍。上圖是測試場景的實際最終GPU縮圖，面板沿用來源角色名稱；背景及受擊閃色仍含像素素材。馬匹與武器比例、光弧寬度及槍影角度有美術近似，待全部HD接入後集中人工確認。來源與限定驗證見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![長戟騎兵、投槍騎兵與三連飛矢的HD測試畫面](docs/figures/hd-riders-archer-battle.png)

上排為長戟下劈與投槍後的藍光揮擊，下排為遠射手準備射擊與三連飛矢。這是測試場景的實際最終GPU縮圖，保留原本離場、命中分段與反擊；面板沿用來源角色名稱，背景與受擊閃色仍含像素素材。人物、馬匹與武器比例、握手及光弧外形有美術近似，正常操作與人工造型確認待全部HD接入後集中處理。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![三位持杖人物的HD戰鬥測試畫面](docs/figures/hd-staff-fighters-battle.png)

紅髮披風、粉紫馬尾與金髮長袍三組人物，已接入原有揮杖與反擊排程。上圖是測試場景的實際最終GPU縮圖，面板沿用來源角色名稱；背景、面板與受擊閃色仍含像素素材。人物比例、握手與衣褶待全部HD接入後集中人工確認，來源及限定驗證見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![拳師、機甲砲手與持杖人物的HD戰鬥測試畫面](docs/figures/hd-fighter-cannon-staff-battle.png)

拳師的上勾拳與金色拳風、機甲砲手的開砲與分段飛彈，以及持杖人物的前刺已接入原有戰鬥排程。上圖是測試場景的實際最終GPU縮圖，面板沿用來源角色名稱；背景、面板與受擊閃色仍含像素素材。姿勢共用、人物／武器比例與握手待全部HD接入後集中人工確認，來源及限定驗證見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![拳師高踢、翼人迴斬與披風劍士劍光的HD測試畫面](docs/figures/hd-fighter-wing-knight-battle.png)

左為拳師高踢，中為翼人迴斬，右為披風劍士劍光。圖組保留翼人飛離畫面的局部與透明停拍，以及兩段／四段命中。上圖由測試場景的實際最終GPU畫面縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。關鍵姿勢共用、光弧形狀與局部裁切差異已記錄，正常操作與造型確認待全部HD接入後集中抽驗。來源與範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

![雙斧交叉斬、紅披風持杖人物與斧盾重甲的HD測試畫面](docs/figures/hd-axes-mage-shield-battle.png)

左為雙斧交叉斬，中為紅披風持杖人物前伸，右為斧盾重甲下劈。上圖由測試場景的實際繪製畫面縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。共用姿勢與人物、武器比例的近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。正常遊玩與造型確認待全部HD接入後集中抽驗。

![藍龍斬擊、灰綠龍展翼與紅龍斬擊的HD測試畫面](docs/figures/hd-dragon-riders-battle.png)

左為藍龍騎者斬擊，中為灰綠龍展翼，右為紅龍騎者斬擊。上圖由測試場景的實際繪製畫面縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。升空與返回保留局部身體取樣，共用姿勢、振翼與比例近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。正常遊玩與造型確認待全部HD接入後集中抽驗。

![連射弓手與兩組持杖人物的HD測試畫面](docs/figures/hd-reused-party-battle.png)

第58組連射弓手及第59、60組持杖人物已接入HD。原版六個銀行與先前圖組完全相同，沿用已驗證的手繪圖源，保留各銀行的動作與命中標記。上圖是測試場景的實際繪製縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。共用姿勢與造型近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，正常遊玩與人工確認待全部HD接入後集中抽驗。

![持杖、拳師與機甲砲手的HD測試畫面](docs/figures/hd-special-reused-party-battle.png)

左上、右上、左下、右下依序為第61組持杖人物、第62組拳師、第63組機甲砲手與第64組持杖人物。八個原版銀行與先前圖組完全相同，沿用原手繪圖源，保留拳光、飛彈與透明停拍。上圖是測試場景的實際繪製縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。共用姿勢與造型近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，正常遊玩與人工確認待全部HD接入後集中抽驗。

![藍衣拳師、劍盾兵、槍兵與披風劍士的HD測試畫面](docs/figures/hd-fighter-guards-battle.png)

左上、右上、左下、右下依序為藍衣拳師的掌光、劍盾兵斬擊、槍兵突刺與披風劍士跪斬，已接入遊戲繪製。上圖為測試場景的實際繪製縮圖，面板沿用來源角色名稱；背景與受擊閃色仍含像素素材。共用姿勢、武器角度與光效裁切見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，正常遊玩與人工確認待全部HD接入後集中抽驗。

![深藍拳師、肩砲人物與有翼蜥蜴人物的HD測試畫面](docs/figures/hd-fighter-cannon-wing-battle.png)

由左至右為深藍拳師回身擊、肩砲射擊與有翼蜥蜴人物斬擊，已接入正式繪製；光束飛行及透明空拍也保留原節拍。上圖是測試場景的實際繪製縮圖，面板沿用來源角色名稱，背景與受擊閃色仍含像素素材。姿勢共用、武器比例與原框裁切見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，正常遊玩與人工確認待全部HD接入後集中抽驗。

![灰綠重甲與綠衣劍士的HD測試畫面](docs/figures/hd-front-knight-fighter-battle.png)

左為灰綠重甲橫斬，右為綠衣劍士下劈，兩組完整動畫已接入正式繪製。上圖是左側敵方主攻、右側我方反擊的測試場景實際繪製，角色名稱沿用測試資料，並非原版對拍或一般遊玩截圖。姿勢共用、原框裁切及殘光近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，正常遊玩與人工確認待全部HD接入後集中抽驗。

![兩種巨劍重甲與錐槍重甲的HD測試畫面](docs/figures/hd-giants-lancer-battle.png)

由左至右為金紋巨劍重甲、紫紋巨劍重甲與錐槍重甲的攻擊演出，已接入正式繪製。上圖是左側敵方主攻、右側我方反擊的測試場景實際畫面，名稱沿用測試資料，背景與面板仍含像素素材。三組保留各自的原生節拍及音效標記；姿勢共用、武器比例、原框裁切與光痕近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)，一般遊玩與人工確認待全部HD接入後集中抽驗。

![巨斧重甲、長戟重甲與飛龍騎乘人物的HD測試畫面](docs/figures/hd-axe-halberd-dragon-battle.png)

由左至右為巨斧下劈、長戟突刺與飛龍騎乘人物攻擊的實際繪製，三組完整動畫已接入。畫面來自左側敵方主攻、右側我方反擊的測試場景，名稱沿用測試資料，背景與面板仍含像素素材。飛龍保留收翼、半展翼及展翼週期，人物比例、武器、光痕與原框裁切的近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)；一般遊玩與人工確認待全部HD接入後集中抽驗。

![兩種肩砲人物的HD測試畫面](docs/figures/hd-cannons88-89-battle.png)

上排為藍甲與灰甲肩砲人物射擊，下排為金色爆裂與藍色光束命中，兩組完整43幀已接入。畫面取自左側敵方主攻、右側我方反擊的測試場景，名稱沿用測試資料；背景、面板與受擊閃色仍含像素素材。砲身比例、光束形狀及裁切等美術近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。一般遊玩與人工確認待全部HD接入後集中抽驗。

![兩種持杖人物的HD攻擊與施術測試畫面](docs/figures/hd-casters91-92-battle.png)

上排為深藍兜帽與紅面罩持杖人物的攻擊，下排為各自施術，完整50幀已接入待機、三段命中與施術路徑。畫面來自修改人物圖源的測試場景，名稱沿用測試資料，背景與面板仍含像素素材。鉤杖、光環生成／懸浮／消散、火焰及姿勢共用的美術近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。一般遊玩與人工確認待全部HD接入後集中抽驗。

![持木杖與持刀人物的HD戰鬥測試畫面](docs/figures/hd-actors94-95-97-98-battle.png)

上排為兩種持木杖人物，下排為持刀人物的下劈與反向斬擊，完整36幀已接入待機與攻防路徑。畫面來自修改人物圖源的測試場景，名稱沿用測試資料，背景與面板仍含像素素材。姿勢共用、刀光形狀與微小殘留塵痕的差異見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。一般遊玩與人工確認待全部HD接入後集中抽驗。

![持刀、拳師與機兵的HD戰鬥測試畫面](docs/figures/hd-actors99-100-101-112-battle.png)

四種人物的65幀素材已接入待機與攻防路徑，機兵一拍依原版零延遲保持不呈現。動畫包含反向劍光、多重拳影與機兵掃射眼光及殘火。畫面來自修改人物圖源的測試場景，名稱沿用測試資料，背景與面板仍含像素素材。姿勢共用、站姿縮放、光痕與拳影的美術近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。一般遊玩與人工確認待全部HD接入後集中抽驗。

![龍首與有翼人物的HD戰鬥測試畫面](docs/figures/hd-actors104-109-battle.png)

六種龍首與有翼人物的122幀素材已接入待機、攻防及已解碼施術路徑，包含吐息、殘火、揮刀、蓄力光束與拳擊。上兩列為攻防，末列為三組普通指令0施術測試；皆取自實際繪製畫面，角色圖源經測試替換，背景與面板仍含像素素材。姿勢與兩種翼型共用、龍首裁切及特效近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。一般遊玩與人工確認待全部HD接入後集中抽驗。

![機兵、砲車、長槍騎士與大型敵人的HD戰鬥測試畫面](docs/figures/hd-remaining-figani-battle.png)

剩餘已解碼人物圖庫的284幀素材已接入，涵蓋機兵、砲車、長槍騎士、鞭手、石像與大型敵人。上三列為十五種圖源的攻防或待機，末列為四組施法測試；均取自實際遊戲繪製，角色圖源經測試替換。原生空拍保留，背景與面板仍含像素素材。共用姿勢、衣物、光形與裁切近似見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。其他效果與完整戰鬥HD仍待，正常遊玩及人工造型依全部串接後集中確認。

| 鐵諾HD普通END攻防 | 蓋亞HD普通攻防 |
| --- | --- |
| ![鐵諾HD普通END攻防](docs/figures/hd-tino-physical-attack.png) | ![蓋亞HD普通攻防](docs/figures/hd-gaia-physical-attack.png) |

左圖是第三章鐵諾在 END 時攻擊，背景與面板仍含像素素材。右圖已換成第23章普通操作的蓋亞手腕修正版，人物、背景與面板文字採 HD，框線與血條仍為像素。
蓋亞沿原框保留待機、蓄力與出拳。新版已經正常進軍、攻擊隊長、死亡演出、戰後整備、F2切換、保存及新程序讀檔，畫面另按實際圖源核對。測試採章前強化槽，沒有章內鎖血；造型自然度仍待全部 HD 後集中人工確認，END參戰另待抽樣。
來源槽祖先、共享稿及驗證範圍見
[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。

| 哈諾HD普通攻防 |
| --- |
| ![哈諾HD普通攻防](docs/figures/hd-hano-physical-attack.png) |

此圖為第二章實際遊戲視窗，使用章前加1000HP的建構槽，再以普通按鍵接敵與攻擊。
哈諾的待機、斧擊與光效已接入HD圖組，背景與面板仍含像素素材。
攻防返回、F2與章節邊界保存冷讀已驗證；此測試槽不代表自然難度。
來源、完整提示與共享姿勢見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`hano_battle_followup_20261020`。

| 女村民HD受擊待機 | 男村民HD受擊待機 |
| --- | --- |
| ![女村民HD受擊待機](docs/figures/hd-villager-133-physical-idle.png) | ![男村民HD受擊待機](docs/figures/hd-villager-134-physical-idle.png) |

兩張取自第二章普通按鍵換手時的實際遊戲視窗，沿用上述章前HP校準槽。
村民各四幀待機已接入HD圖組；原版攻擊與施術圖庫為空，不另造動作。
背景與面板仍含像素素材，呼吸相位共享母稿；來源與驗證範圍見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`villager_idle_battle_followup_20261020`。

| 瑪琳HD普通攻防 | 凱麗HD普通攻防 |
| --- | --- |
| ![瑪琳HD普通攻防](docs/figures/hd-marlene-physical-attack.png) | ![凱麗HD普通攻防](docs/figures/hd-kylie-physical-attack.png) |

兩張皆為第九章普通按鍵攻擊的實際遊戲視窗，使用明示章前HP+1000的建構槽。
瑪琳的揮杖與凱麗的掌擊、旋身已接HD圖組；背景、敵方與面板仍含像素素材。
攻防返回、F2及章節邊界保存冷讀已驗證；共享姿勢、來源政策與驗證範圍見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`early_party_battle_followup_20261020`。
貝克威的端炮與光束也已正式登錄；混合背景反擊的一般能力限制仍由[#265](https://github.com/wicanr2/fd2_re/issues/265)追蹤。

![物品與金錢成功提示的HD文字與頭像](docs/figures/hd-reward-message.png)

戰後物品與金錢成功提示已接上清晰文字、高解析頭像與HD世界，保留原有開收框及結算順序。
左右為受控頭像選擇及獎勵值的實際遊戲畫面，完整縮圖未裁切；面板與部分地形仍為像素素材，正常玩家抽驗仍待。
來源見[HD驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`reward_message_hd_followup_20261023`。

![HD地圖回復的前導、光環、輪廓閃光與綠色數字](docs/figures/hd-heal-map.png)

回復演出已接上 HD 世界、淡藍光環、人物輪廓閃光與綠色數字，保留單體及範圍回復的結算順序。
四格取自指令13～16受控測試的完整遊戲畫面，使用固定種子7與明示角色位置、回復量及MP消耗；尚未作正常玩家抽驗。
光環共用手繪母稿，地形與人物的色表映射為現代呈現近似；HUD及其餘未重繪地形仍含像素素材。
來源與範圍見[HD驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`heal13_16_map_hd_followup_20261023`。

![地圖法術的HD取樣、前導與雙目標結果](docs/figures/hd-map-commands10-12.png)

四格取自指令10～12的實際遊戲繪製，保留完整遊戲區並縮為一半大小：地形取樣、另一取樣位置、徑向前導與雙目標結果。使用固定種子7、受控傷害與命中參數及移位角色，屬測試場景，尚未作正常玩家抽驗。人物與已重繪地形保持HD，其餘地形及HUD仍含像素素材；高解析子像素取樣與變色為現代呈現近似。來源及範圍見[HD驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`commands10_12_map_hd_followup_20261023`。

![地圖23法術前奏、底圖、雙目標傷害與返回的HD畫面](docs/figures/hd-commands1012-stage.png)

指令10～12已保留同次手繪背景，四格依序為前奏、底圖、雙目標傷害與返回。畫面取自固定種子與受控角色位置的完整遊戲繪製半縮；人物、綠色背景與數字使用 HD，城堡及介面框線仍為像素。來源及範圍見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)與[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`commands1012_stage_hd_followup_20261025`，尚未作正常玩家抽驗。

![玩家第九類攻擊的HD地圖效果與結果](docs/figures/hd-player9-map-effects.png)

四格取自重製遊戲實際繪製的測試場景，依序為幽影升起、揮刃、命中與 MISS；只裁切效果周圍區域。固定亂數種子7，角色位置為(1,6)、目標為(2,6)，命中與 MISS 採受控測試參數，尚未作正常玩家抽驗。地圖人物保持HD，HUD與多數地形仍含像素素材；升起裁切、共用母稿與光形是手繪近似。來源與驗證範圍見[HD驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`player9_map_hd_followup_20261023`。

![凱麗攻擊動畫：原版 DOS 與 HD 重製並排](docs/figures/hd-kylie-original-vs-hd.gif)

左側是 dosgolem 執行原版遊戲，右側是 HD 重製的實際遊玩畫面，節錄凱麗的掌擊、旋身與收勢。右腳已依原版踩地站姿修正；右側取自新版正常攻擊，保留完整遊戲區。
兩側沿用上述章前加血槽，屬相近戰況；動畫依動作順序剪輯，血量與亂數不同，不代表逐幀一致。
HD 背景、敵方與面板仍含像素素材。來源與可重跑方式見
[動畫驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`kylie_readme_animation_20261022`。

| 貝克威HD端炮 | 貝克威HD光束 |
| --- | --- |
| ![貝克威HD端炮](docs/figures/hd-bekwei-cannon-ready.png) | ![貝克威HD光束](docs/figures/hd-bekwei-physical-attack.png) |

兩張取自第九章同一場普通攻擊的實際視窗，使用章前加血槽，再正常購弓、立即裝備與出發。
端炮、離場與光束沿原版演出；背景、敵方及面板仍含像素素材。
普通攻擊返回、F2及保存冷讀已抽驗，來源與驗證範圍見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`bekwei_battle_followup_20261020`。

| 鎧甲武士HD待機 | 鎧甲武士HD下劈 |
| --- | --- |
| ![鎧甲武士HD待機](docs/figures/hd-heavyknight-idle.png) | ![鎧甲武士HD下劈](docs/figures/hd-heavyknight-physical-attack.png) |

兩張取自第九章普通敵方回合的實際遊戲視窗，沿用章前加血槽，沒有編修畫面。
厚重尖肩甲、抬劍與下劈沿原版姿勢重繪；背景與面板仍含像素素材。
普通攻擊、反擊、F2及保存冷讀已抽驗，共享姿勢與抬劍裁切見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`heavyknight_battle_followup_20261020`。

| 弓箭手HD瞄準 | HD飛矢抵達希莉亞 |
| --- | --- |
| ![弓箭手HD瞄準](docs/figures/hd-archer87-aim.png) | ![HD飛矢抵達希莉亞](docs/figures/hd-archer87-arrow.png) |

兩張取自第九章普通敵方回合的實際遊戲視窗，沿用章前加血槽，沒有編修畫面。
蒙面弓手的搭箭、滿弓、放箭與飛矢沿原版姿勢重繪；背景與面板仍含像素素材。
普通敵方射擊、我方射擊弓手、掉落訊息、F2及章節邊界保存冷讀已抽驗，範圍見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`archer87_battle_followup_20261020`。

| 突擊騎兵HD蓄勢 | 突擊騎兵HD重擊 |
| --- | --- |
| ![突擊騎兵HD蓄勢](docs/figures/hd-knight83-windup.png) | ![突擊騎兵HD重擊](docs/figures/hd-knight83-impact.png) |

兩張取自第九章普通敵方回合的實際遊戲視窗，沿用章前加血槽，沒有編修畫面。
藍甲、雙頭槍、重擊金光與消退收勢依原姿勢重繪；右圖同時顯示希莉亞的紅色受擊輪廓。
背景與面板仍含像素素材。普通接戰、我方射擊、F2及章節邊界保存冷讀已抽驗，範圍見
[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`knight83_battle_followup_20261020`。

![僧侶的HD戰鬥待機](docs/figures/hd-monk93-idle.png)

第九章希莉亞射擊僧侶時的實際視窗，未編修。灰袍、面布、藍腰飾與羽形杖依原圖重繪，背景與面板仍含像素素材。
待機與揮杖已保留原版六幀節奏；完整攻擊限測試場景，普通流程抽驗受擊、返回、F2及章節邊界冷讀。
抽樣沿章前加血來源；回復法術的地圖特效已有受控 HD 驗證，正常玩家抽驗仍待，範圍見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

![洛娜旋槍前刺的HD弧光](docs/figures/hd-lorna-spear-arc.png)

第九章洛娜原地攻擊突擊騎兵的實際遊戲視窗，未編修。藍披風、銀甲、雙頭槍與弧光已接HD，背景與戰鬥面板仍含像素素材。
抽樣沿章前加血來源，章內無注入；完整繪製、攻防返回、F2與章節邊界冷讀見[驗證範圍](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

| 萊汀 HD 斧擊 | 索菲亞 HD 揮杖 |
| --- | --- |
| ![萊汀HD斧擊](docs/figures/hd-leitin-axe-sweep.png) | ![索菲亞HD揮杖](docs/figures/hd-sofia-staff-strike.png) |

兩張取自第十一章正常移動接敵後的遊戲視窗，未編修。厚甲、長柄斧、金色馬尾與寶石杖依原造型重繪，背景、面板與敵方仍含像素素材。
本次使用章前加血的建構槽，章內沒有狀態注入；攻擊返回、F2與章節邊界保存冷讀已抽驗，範圍見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

| 獸人 HD 待機 | 獸人隊長 HD 待機 |
| --- | --- |
| ![獸人HD待機](docs/figures/hd-beast102-idle.png) | ![獸人隊長HD待機](docs/figures/hd-beast103-idle.png) |

取自第十一章索菲亞與凱麗普通近戰的實際視窗，未編修。豹斑、長尾、尖刃與兩種布料配色依原圖重繪，背景與戰鬥面板仍含像素素材。
跳躍、出框、空白等待與俯衝保留原節奏；完整繪製及正常抽樣的不同範圍見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。抽樣使用明示的章前加血建構槽，章內無注入。

### 普通法術的人物HD呈現

![敵方第九類攻擊的HD幽影與光柱](docs/figures/hd-command9-ai-battle.png)

敵方第九類攻擊已接入手繪幽影、光柱與鬼臉，保留滑入、扣血及消散順序。上圖是修改角色與指令資料的測試場景，取自實際 Game.Draw 的最終GPU縮圖；面板沿用來源角色名稱，背景與介面仍含像素素材。玩家同類指令的地圖演出另列待辦。圖源共用、光柱伸縮與面紋位置的近似，以及限定驗證範圍，見[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。正常玩家與人工造型確認仍待全部HD接入後集中抽驗。

![悠妮普通施法的HD人物](docs/figures/hd-ordinary-command0-yuni.png)

取自首章普通LOAD後的實際施法視窗，未編修畫面。指令0已接入HD人物的滑入、施術、目標及收尾呈現，背景、狀態欄與法術特效仍含像素素材。
正常返回、F2及章節邊界保存冷讀已抽驗；F5／LOAD會重建戰場，未保存本次施法後的魔力與敵方血量。來源與範圍見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`ordinary_command0_followup_20261020`。

![悠妮普通指令1的HD人物](docs/figures/hd-ordinary-command1-yuni.png)

第11章普通鍵盤施法的實際視窗，未編修。指令1已接入既有HD人物，法術特效與戰鬥面板仍含像素素材。
章前建構來源、原地施法、返回及章節邊界保存冷讀的範圍見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的 command1_hd_followup_20261021。

![魔法師的HD普通施法](docs/figures/hd-mage90-cast.png)

第九章正常進軍後的魔法師施法視窗，未編修。人物為HD，背景、戰鬥面板與法術特效保留像素素材。
抽樣使用章前加血來源，章內沒有狀態注入；施法、受擊、返回與保存冷讀見[驗證範圍](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

| 約拿 HD 近戰 | 希爾法 HD 近戰 |
|---|---|
| ![約拿HD近戰](docs/figures/hd-jonah-physical-attack.png) | ![希爾法HD近戰](docs/figures/hd-silfar-physical-attack.png) |

上圖是兩位角色正常移動接敵後的實際攻擊視窗，未編修。寬袍、披領與揮杖順序依原圖重繪。
接敵路線、章前強化與驗證範圍見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`cleric_party_battle_followup_20261021`。

本組畫面只涵蓋已驗證的近戰圖組；完整 HD 仍在製作。

| 機甲隊長受擊時的 HD 待機 | 機甲隊長戰後 HD 頭像 |
| --- | --- |
| ![機甲隊長HD待機](docs/figures/hd-captain116-target.png) | ![機甲隊長HD故事頭像](docs/figures/hd-captain116-story.png) |

兩張皆為正常鍵盤流程的重製視窗，使用明示章前強化槽。戰鬥圖組沿原版十一幀與雙刃節拍重繪，戰後自然接入下一章整備；隊長主動攻擊的正常流程尚未抽驗。詳見[HD 驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的 `captain116_battle_followup_20261021`。

![珊HD普通揮杖攻擊](docs/figures/hd-shan-physical-attack.png)

上圖是珊正常進軍後揮杖攻擊的實際遊戲視窗，未編修。藍色長髮、抬杖與前揮保留原版造型及動作順序。
普通攻擊、返回、F2與保存冷讀已抽驗，完整人物幀另由繪製測試核對。測試使用章前強化槽，人物外效果與背景／面板仍含像素素材；
來源與共享姿勢範圍見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`shan14_battle_followup_20261021`。

![第23章機甲兵HD受擊待機](docs/figures/hd-mechanical111-normal.png)

上圖是蓋亞普通攻擊機甲兵時的實際遊戲視窗，未編修。機甲兵保留原版灰綠重甲、深藍關節與前臂砲造型。
受擊待機、返回、F2及保存冷讀已抽驗；開砲與光束完整繪製另由測試場景驗證，普通開砲仍待。
測試使用章前強化槽，背景與面板仍含像素素材；來源見[驗證紀錄](docs/data/fd2-modern-hd-battlefield-20261019.json)的`mechanical111_battle_followup_20261021`。

普通擊殺與章節事件的死亡演出已接入 HD 煙霧，保留人物轉向、效果累積與消散。

![第23章普通擊殺的HD死亡演出](docs/figures/hd-action-death-normal.png)

上圖來自第 23 章正常進軍、普通擊倒隊長時的實際遊戲視窗，未編修。
測試使用章前強化存檔，章內沒有狀態注入。A／B 效果抽樣、下一章整備、F2 與保存冷讀已驗證；
完整 26 拍另由繪製測試核對。狀態扣血致死也已接入相同呈現，正常玩家抽樣仍待；
範圍見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

![特殊背景下的HD死亡動畫與返回](docs/figures/hd-death-stage.png)

特殊背景下的死亡動畫已保留同次 HD 背景，人物消散期間不再提早疊入狀態人物。四格依序為普通擊殺的最後姿勢、煙霧累積、消散與返回，取自受控繪製測試的完整遊戲畫面半縮。事件與狀態致死也通過相同流程；城堡及部分介面仍含像素。這不取代上方正常章節抽樣。範圍見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)與[HD收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`death_stage_hd_followup_20261025`。

## 遊戲介紹

《炎龍騎士團 2》是 1990 年代華文單機戰略角色扮演遊戲（SRPG）的代表作之一。
玩家在格狀戰場上指揮各具職業、裝備與法術的角色，並在戰鬥之間經歷劇情、城鎮、
商店、教會、轉職、整備及持續成長的隊伍。

它真正難以取代的地方，不只是三十場戰役，而是戰場、人物、音樂與戰間生活共同
形成的節奏。多數戰鬥結束後並非直接跳到下一戰；補給、交易、復活、轉職、存檔與
部署同樣是戰役的一部分。重製因此不能只還原戰鬥公式，也必須保留這條玩家路徑。

## 開發歷史

### 為台灣留一份技術紀念

逆向動畫資料時，我們在原版檔案裡找到一段幾乎被時間掩埋的署名：

> AFM — Animation File Manager Version 1.00<br>
> Copyright (C) 1993 Lo Yuan Tsung

這套由漢堂程式設計師自製的動畫工具，比遊戲發售早兩年。原始碼與說明書如今都
已不可得，但它設計的資料仍在遊戲裡運作。本專案把破解出的技術整理成可重現文件，
不只為了重製，也為了記錄 1995 年台灣團隊如何在 DOS 的限制下完成一款大型戰棋
角色扮演遊戲。

完整總覽：
[`1995 年怎麼做出《炎龍騎士團 2》`](docs/knowledge-base/15-how-fd2-was-made-1995.md)

### DOS 上的中文

DOS 本身不會替遊戲顯示中文。原版自帶點陣字型，文本也不是 Big5 字串：
`FDTXT.DAT` 保存的是 `uint16` 字模索引序列，共 1016 條字串、約 5.8 萬字；
`FDOTHER.DAT` 資源 #4 則包含 1824 個 16×16、1bpp 字模。索引 0–35 是數字與
英文字母，其後才是漢字。

| 字模解碼研究圖 | 對話解碼研究圖 |
|---|---|
| ![FD2 原版字模表研究圖](docs/figures/font_atlas.png) | ![FD2 原版對話與字型解碼研究圖](docs/figures/dialogue.png) |

以上兩圖是原版資料解碼研究產物，不是重製執行期畫面。格式與控制碼詳見
[`文本與字型格式`](docs/knowledge-base/08-text-and-font-format.md)。

### 開場動畫是一台繪圖位元組虛擬機

原版開場不是把每一幀都存成完整點陣圖。AFM 使用 10 個運算碼描述增量繪圖，
每幀只對上一幀留下的畫面補畫、複製或修改局部區域。96 幀的金鎖動畫因此約為
1 MB，而不是 96 份完整 320×200 畫面。

```text
影格：duration + payload_size + bytecode
指令：opcode + 參數；部分指令引用前一幀或外部圖塊
```

原版派發器位於 `0x36C9E`，跳表位於 `0x5276A`；播放器
`0x020421(index, delayMs, skippable)` 管理播放與略過，`0x3DC9F` 以開機測速忙等
校準舊硬體。重製端保留這套增量語意，但不夾帶任何原版畫格。完整證據見
[`ANI／AFM 格式`](docs/knowledge-base/39-ani-afm-format.md)。

### 同一首音樂，兩種玩家記憶

原版同時支援 Roland MT-32 與 Sound Blaster／AdLib。後者使用遊戲自帶的
`SAMPLE.AD` 音色庫，也是當年多數玩家聽見的預設聲音；MT-32 則有不同的樂器、
包絡與空間感。因此「原版音樂」不是唯一一套固定波形。

曲號不是靠聽感猜測，而是從 `play_bgm`（`0x25977`）的 32 處呼叫逐一追查，過程中
也推翻過舊曲目對應。兩條音源都只由玩家自備原版檔案在本機渲染，音色庫與 ROM
不隨倉庫散布。詳見 [`音樂播放與場景`](docs/knowledge-base/12-music-playback-and-scene.md)
及 [`音訊合成`](docs/knowledge-base/16-audio-synthesis-soundfont-mt32.md)。

## 歷史考證

本專案將格式、工具、呼叫關係與玩家可見結果一併保存，並為每個結論標示
「已證實、強推論、假說或未知」。原始位址、位元組與雜湊始終保留；語意名稱只作
導覽，不取代證據。

主要考證入口：

- [原版工具鏈](docs/knowledge-base/04-original-toolchain.md)、[圖像壓縮](docs/knowledge-base/05-image-compression-format.md)、[動畫格式](docs/knowledge-base/06-animation-format.md)、[XMIDI 音樂](docs/knowledge-base/07-music-xmidi-format.md)
- [文本與字型](docs/knowledge-base/08-text-and-font-format.md)、[劇情與對話](docs/knowledge-base/09-story-and-dialogue.md)、[角色與職業名稱](docs/knowledge-base/45-class-name-mapping.md)、[戰鬥規則](docs/knowledge-base/27-combat-rules-and-validation-checklist.md)、[敵方人工智慧](docs/knowledge-base/11-enemy-ai.md)
- [開機、標題與存檔](docs/knowledge-base/23-boot-title-and-scenario-flow.md)、[可擴展事件系統](docs/knowledge-base/29-remake-extensible-event-system.md)、[劇情腳本系統](docs/knowledge-base/19-scenario-script-system-design.md)
- [反組譯覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)與[知識庫總索引](docs/knowledge-base/00-index.md)

這些文件同時是「1995 年台灣怎麼做遊戲」的技術保存紀錄，不只是工程工作清單。

### 匹配式反編譯原始碼

重寫的 C 原始碼與可重跑工具保存在本倉庫的 [tools/](tools/)，隨 Git 提交記錄。
[事件布局來源](tools/fd2_matching_game_event_layout.c)已有完整區間匹配；
[大型事件來源](tools/fd2_matching_game_quake.c)也已通過完整函式匹配。
[建置驅動器](tools/fd2_matching_game_restore.py)與
[匹配主收據](docs/data/ida/fd2_matching_full_20261008.json)保存來源雜湊、編譯旗標與比較結果。
全檔還原狀態見[覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。

## 重要發現

- 原版使用自製 `.DAT` archive、RLE 圖像、索引色盤、XMIDI、AFM 與 FIGANI 等
  多套專用格式，而不是一組現成引擎資源。
- 對話文字、人物動作與大量戰役流程曾寫死在 `FD2.EXE`；重製將它們轉成可編輯
  腳本與具型別規則。
- 敵方人工智慧不是單一「找最近的人」規則。原版先依模式建立候選、評分與目標
  陣列，再由指令或物品消費端執行；重製保留已證實資料流，未知分支不猜接。
- 原版戰後流程包含城鎮、商店、教會、整備與秘密商店等節點；它們是戰役的一部分，
  不能以直接切到下一戰替代。
- 最終演出不是單一影片，而是文字閘門、角色蒙太奇、20 組尾段、定格與隊伍最終
  狀態共同形成的序列。重製端目前會停在經典終局畫面，並可循環回顧隊伍；這是
  來源約束的 `RUNTIME-E1`，未修改原版同狀態 `PLAYER-E2` 仍待抽驗。

### 指令環與角色的繪圖順序

![原版與重製正式Game.Draw的指令環局部總覽](docs/figures/ring-cursor-original-remake-overview.png)

左側是dosgolem原版擷取，右側是正常載入後的重製正式Game.Draw。原版先貼四張圖示，再重繪游標上的角色；重製也保留這個順序，避免後續再貼圖示蓋住角色。
完整畫面已在明示的有限顯示相位下核對，這張圖只展示局部。驗證範圍與來源見[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)及[指令環主契約](docs/data/ida/fd2_parity_ring_open_20261005.json)，不代表整章逐幀一致。

### 原版美術與目前重製畫面總覽

![原版人物肖像、重製對話畫面與第一戰戰場圖示總覽](docs/figures/original-remake-portrait-battle-overview.png)

*這張素材總攬圖由已審查的執行期截圖組成；本機完整分離包現有 40,127 個實體檔案、
約 104 MB，並以清冊逐檔驗證。完整原版衍生素材只保存於私人庫，不在公開 README
提供可重組原作的逐檔 atlas。上排用不同場景展示原版 DOS 對話與目前重製執行期的
人物肖像系統，只比較呈現
方式，不是同狀態忠實度證據；下排則是第一戰同一存檔、鏡頭與回合的原版／重製
戰場圖示抽樣。忠實主題仍使用玩家自備原版資料所分離的美術；現代主題則已有
已登錄角色的獨立閉嘴頭像候選，經尺寸、雜湊與原子載入契約驗證後，可由
顯式主題入口載入。這仍是 `RUNTIME-E1-PARTIAL`，不是全角色或全介面完成。*

![現代重繪角色頭像總攬](docs/figures/modern-portrait-overview-v2.png)

*現代重繪角色頭像總攬：圖中為先前登錄的 28 位候選。每張皆以原版獨立頭像幀確認
角色身分、方向與主要特徵，再以共同主題規格重繪；完整母稿與執行期 PNG 保存於私人
素材庫 `fd2-assets-private`，公開庫只放這張壓平總攬與可追溯 catalog。
`speaker／identity 7、15、28、29` 仍因原始名稱證據衝突而失敗即關閉，不以猜測補圖。*

![原版代表性色盤與現代地圖人物總攬](docs/figures/modern-map-sprites-overview.png)

*地圖人物總攬：上列由雜湊固定的原版 FDICON 索引與 `FDOTHER.DAT` #0 六位元 VGA
DAC 代表性色盤合成，透明度仍取自原版圖形遮罩，不再以灰階索引圖誤導閱讀；個別場景
仍可能依原版排程調整色盤，因此這不是所有關卡的逐幀實機色彩證明。下列才是現代執行期
候選，避免再把人物編號便利輸出誤當 FDICON selector 證據。本總攬是較早94組子集，其中89組保留四方向×三步態、5組維持三相靜態契約。現行清冊已涵蓋原始圖庫全部125組可見來源，共1455張144×144人物圖；來源與範圍見[HD戰場收據](docs/data/fd2-modern-hd-battlefield-20261019.json)。selector 1–14、16–21
由角色表支持為哈諾／鐵諾／哈瓦特／亞雷斯／洛娜／萊汀／蘭斯洛特／希莉亞／悠妮／瑪琳／索菲亞／凱麗／貝克威／珊／凱拉斯／米亞斯多德／蜜蒂／羅德曼／莎拉／約拿的基礎圖組投影；selector 15 僅標為姓名仍待校訂的角色表投影，不與故事頭像身分合併。selector 000、002、008、009、013、032 已依使用者核准的大頭短身現代手繪母風格重新校稿；所有正式候選均為獨立 `24×24` PNG，具逐幀雜湊與
二值透明度契約；selector 76 是第三章普通追兵的獨立候選，保留原版低圓頂深鋼盔、
額鼻護片、護頰、無徽記木盾與量產制式軍裝，刻意降低俊美英雄與精銳騎士感；selector 77 則是同章更厚重的敵方隊長，保留
全罩盔、板甲、大盾與直劍，但不猜其姓名；selector 82 則依第 7–9 戰共 49 筆敵方
劍士資料保留矩形全罩盔、窄眼縫、灰盾、直劍與紅色下擺；selector 78 依 29 筆敵方
劍士資料保留封閉低盔、紅眼、巨大圓肩與敦實重甲，不擅自補入原版小圖看不出的武器；selector 80 因職業欄混合，
只依 38 筆共同圖像索引保留深藍重甲、紅色盔冠與寬圓肩；selector 83 依 42 筆敵方
劍士資料保留鈷藍角盔、黃綠眼光、藍盾與直劍；selector 85 依 36 筆敵方劍士資料
保留完全遮面的低垂大型桶盔、紅眼、方盾、直劍與制式深藍甲，刻意維持普通士兵辨識；selector 86 依第 24–26 戰及終章共 24 筆敵方
劍士資料保留淡灰綠圓頂全罩盔、水平眼縫、深藍十字盾與直立長兵器輪廓，維持量產要塞守衛感；selector 88 依 48 筆敵方
劍士資料保留深藍護甲、紅褐頭巾、遮面眼部與橫持直劍；selector 90 依第 4–9 戰共 26 筆敵方
劍士資料保留低垂紅褐兜帽、黑色面孔、黃綠眼光、寬大深藍肩衣與敦實輪廓，不添加原圖沒有的武器；selector 91 因職業欄混合且
沒有可見武器，只依輪廓近似保留深藍尖垂兜帽、黑色面孔、紅眼與對稱暗紅外緣，
不把外緣斷言成特定服裝。此圖是公開壓平總攬；母稿與逐幀素材只保存於私人素材庫。完整HD包的普通戰場沿原生定位直繪144×144真彩人物，再疊加前景與透明介面；尚未換成HD的完整演出仍保留索引畫面。較早24×24主題包沿既有路徑，忠實主題維持原生索引呈現。圖組 7
只代表角色表與地圖資料投影；故事 `speaker 7`／頭像身分仍有衝突，維持失敗即關閉。希莉亞的
`speaker 8`／`DATO_008` 身分則已有一致證據。*

![依第一關概念稿統一色彩與材質的 33 張現代戰場總覽](docs/figures/modern-battlefields-overview.png)

*現代戰場第一輪總覽：33 張地圖均由正式 `map.json` 重組，保留原版 tile ID、
通行、事件與人物座標，圖集則依第一關概念稿與高解析手繪參考統一自然綠、土黃、
水岸與柔化材質；現代戰鬥狀態面板也已有獨立無文字底板，由程式疊加即時數值。這是
較早可玩的 `RUNTIME-E1` 現代主題基底，現行144人物路徑的地形覆蓋另依上方實際遊戲畫面與[現況矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。完整逐格PNG保存於私人素材庫。後續仍會抽測相鄰圖塊接縫、前景遮擋與各地區辨識度。*

![現代戰場狀態面板與四向行動圖示總覽](docs/figures/modern-battle-ui-overview.png)

*現代戰場 UI 素材總覽：上方是無文字狀態面板，正式執行時才疊加角色名稱、
LV、HP、MP、AP、DP、MV；下方依序為攻擊、法術、物品、待機。四向選單仍顯示在
目前選取方塊附近，不固定到左上角。這是壓平展示圖；正式 loader 會依 catalog
逐檔驗證 149×42 HUD、四張 28×26 圖示與 SHA-256，再由現代主題消費。*

| 33 張原版戰場低解析索引 |
|---|
| ![原版 33 張戰場低解析總覽](docs/figures/original-battlefields-atlas-public.png) |

| 96 組戰場角色代表幀 | 96 組對話頭像代表幀 |
|---|---|
| ![原版戰場角色代表幀總覽](docs/figures/original-sprites-atlas-public.png) | ![原版對話頭像代表幀總覽](docs/figures/original-portraits-atlas-public.png) |

*三張總覽由玩家自備原版的本機分離素材產生：戰場圖依 `map.json` 重組完整地圖，
角色動畫與頭像則每組只取一個代表幀。公開版本經降解析並壓平，只供辨識研究覆蓋，
不是可供遊戲回填的素材表；完整解析素材、全部動畫幀及可重組清冊只留在私人素材庫
與標示 `public_distribution: false` 的本機完整版。*

![ch01 同狀態原版、重製戰場與差異遮罩](docs/figures/battle-field-ch01-scoped-compare-20260810.png)

*同一存檔、鏡頭與回合的戰場比較；此圖只證明標示狀態，不外推全戰役。*

![物理攻擊前導的原版、重製合成與完整畫面差異](docs/figures/physical-attack-prelude-scoped-compare.png)

*各排依序為原版擷取、重製同輸入合成、完整畫面差異。上排是玩家攻擊，下排是獨立
敵方攻擊；只比較零旗標前導末格。這是合成診斷圖，正式 GPU 前導驗證另記於
[場景契約](docs/data/ida/fd2_physical_background_selection_20261004.json)。完整攻擊、
非零旗標捲動與戰後畫面恢復的現況見[覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。*

![物理攻擊首次捲動的原版、同輸入合成與完整差異](docs/figures/physical-scroll-scoped-compare.png)

*各排依序為原版擷取、重製同輸入合成、完整未遮差異。上至下展示非零旗標的前導末格、
捲動中段與目標末格。正式GPU已驗證這段首次轉場；完整攻擊、連擊、反擊與戰後
工作緩衝的現況見[覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。*

![單次未命中物理尾段的原版、同輸入合成與完整差異](docs/figures/physical-tail-scoped-compare.png)

*各排依序為原版擷取、重製同輸入合成、完整未遮差異，展示單次未命中的尾段。
這個有限案例已核對完整畫面及正式GPU呈現。命中、連擊、反擊、音效與戰後工作
緩衝的驗收範圍見[覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。*

![城鎮出發縮放與戰場淡入的原版來源、正式重製GPU及完整差異](docs/figures/town-departure-original-vs-remake.png)

*上至下為縮放首步、末步與淡入完成的戰場。每排依序為原版索引畫面加同時點DAC重建、
重製正式GPU、完整未遮差異。原版欄不冒稱三張寫後擷取；來源與限定驗收見
[城鎮過場契約](docs/data/ida/fd2_town_departure_20261005.json)及[覆蓋矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。*

![確認等待嘴型的原版與正式重製對照](docs/figures/confirmation-mouth-original-vs-remake.png)

*左為dosgolem原版，右為正式GPU；上為DATO第0幀，下為第3幀。兩側使用同一建構槽與
受控時鐘、原版亂數餘數，經正常LOAD及城鎮出發進入等待。完整章與精確硬體時間不由此圖驗收。
來源與限制見[確認等待契約](docs/data/ida/fd2_confirmation_mouth_runtime_20261005.json)。*

![重製端正式開場的漢堂發行商畫面](docs/figures/title-publisher-remake.png)

*重製正常啟動路徑的執行期畫面；美術來自玩家自備原版資料。*

## 重製技術創新

- **可編輯戰役資料**：以 `trigger → when → do` 事件模型與具型別 handler，將原本
  寫死的對話、增援、入隊、戰後與終局流程轉成可審查資料。
- **索引色原生呈現**：保留 DOS 時代的色盤、LUT、RLE、逐幀發布與畫面恢復語意，
  再由現代引擎跨平台顯示，避免先轉成一般 RGBA 動畫後遺失原始行為。
- **原子式失敗即關閉**：複雜法術、物品、AI 與存讀檔先在私有狀態預檢；缺少來源
  或資產時不發布半套交易，避免玩家存檔被猜測性實作污染。
- **範圍法術與演出**：指令 6 依游標格建立作用名單，支援我方格與空格作為中心，
  取消後可重新施法。固定素材的負列寫入採已批准的有界畫面相容處理；
  原版完整記憶體行為仍未知。驗收範圍與完整畫面來源見
  [指令 6 契約](docs/data/ida/fd2_ch24_command6_work_bounds_20261004.json)。
- **持續隊伍與原始存檔橋接**：保留加入順序、裝備、指令、HP／MP、增援與部署
  邊界，使戰場、城鎮、整備與終局能共享同一支隊伍。
  非城鎮整備保留記錄提示、四槽保存與返回選人的流程；
  [局部保存驗證](docs/data/ui-traces/preparation-record-save.json)與整章驗收分開記錄。
- **證據驅動的畫面比較**：把輸入序列、狀態、資產雜湊、畫面差分及證據等級綁在
  一起；不以單張看似相近的截圖宣稱整套介面完成。
- **可延續的潔淨室引擎**：原版戰役是第一個正式資料包；長期目標是讓新戰役能沿用
  同一套戰鬥、事件、城鎮與存檔能力，而不必修改原版 executable。
- **可追溯的四語內容層**：繁中、簡中、日文與英文共用 1,564 句 canonical 劇情身分；
  正式 story、handler dialog 與戰場事件61／75／76已改以 `line_id` 查詢候選譯文，缺鍵或
  身分漂移即停止，不用來源行號或內嵌繁中掩蓋錯誤。非繁中原生 indexed 對話已保留
  開收框、閉嘴頭像、鏡頭與逐字顯示並重建安全分頁。三個譯文包的 5,176 筆全部走過人工
  審校：簡中 5,176 筆定稿，日文與英文各 5,006 筆定稿，另 170 筆說話者欄位在來源就只剩
  單一字模（要與相鄰行拼起來才成名字），拆成來源身分阻擋另計、不猜定譯名，逐條登在
  [`review-blockers.json`](docs/data/localization/review-blockers.json)。200 個道具名
  另有以實體 ID 為鍵的目錄，全部通過原版 67×16 安全矩形；同一個角色在對白、戰鬥面板與
  章節標題只會有一種譯名。斷行以「拉丁單字整串搬、中日韓逐字可斷」為單位，英文與日文
  譯文不會被切在單字中間。這是內容層完成度，不等同全部語系介面或正常玩家路徑 E2。

工程架構移至 [`docs/ENGINEERING.md`](docs/ENGINEERING.md)；第一輪重製門檻、目前
進度與剩餘工作移至 [`docs/REMAKE-STATUS.md`](docs/REMAKE-STATUS.md)。

第十二章敗北已依原版播放提示並返回完整標題；同狀態驗證範圍與限制見
[敗北流程收據](docs/data/ui-traces/ch12-defeat-return-title.json)。

## 逐章對拍進度

重製每章以原版執行結果為基準，兩側載入相同存檔槽並重播等價鍵盤輸入。
原版由dosgolem執行，驗收記錄行為、節點、交易與畫面。必驗項目、作弊政策與保存例外
依[111](docs/goal/111-goal-original-parity-campaign-20260915.md)及各章收據；畫面作參考時，不宣稱逐幀或逐像素一致。

30章均已通過各章規定的受控流程驗收，包含增援、戰後及保存邊界。作弊、未抽樣分支與第1、2章的限定保存差異均逐章揭露；HD與現代主題覆蓋仍在進行。

目前分層現況與接續工作見[重製閉合矩陣](docs/knowledge-base/58-fd2-exe-re-coverage.md)。逐章驗收、章前作弊政策及保存限制見[章台帳](docs/data/parity-campaign-progress.json)與[進度頁](docs/REMAKE-STATUS.md)；各章以自己的收據為準，流程通過不代表逐像素一致。

第十六章已通過同槽戰鬥抽樣、戰後交易與酒店存檔驗收，範圍與限制見[完整章收據](docs/data/ui-traces/parity-ch16.json)。
下圖五個起手畫面逐像素相同；[整章總覽索引](docs/data/ui-traces/parity-ch16-samples.json)另保留所有非零差異。

![第十六章有限起手：原版、重製與完整畫面差異](docs/figures/ch16-native-startup-samples.png)

第十七章已通過四回合抽樣、增援、戰後交易與酒店存檔驗收，詳見[完整章收據](docs/data/ui-traces/parity-ch17.json)與[整章總覽索引](docs/data/ui-traces/parity-ch17-samples.json)。依111／114建構槽例外列PLAYER-E2，收據記錄第5回合清敵與受控亂數比較範圍。下圖仍是五個起手畫面的有限比較。

![第十七章有限起手：原版、重製與完整畫面差異](docs/figures/ch17-native-startup-samples.png)

第十八章的章內護援、兩次回合事件、戰後加入隊伍、買賣、酒店存檔與秘密商店入口已依111／114例外驗收。範圍與非零畫面差異見[完整章收據](docs/data/ui-traces/parity-ch18.json)及[本機總覽來源索引](docs/data/ui-traces/parity-ch18-samples.json)。

讀這張表要知道的限制：建構槽是工具依政策值建出來的（每章固定升級數、強化基底
攻防），不是原版實際遊玩累積的數值，所以收據只能談節點、畫面、介面與存檔閉環，
不能拿來談傷害、命中或敵方選目標；抽樣尾端用修改路徑一次清空敵方 HP 進戰後節點，
那一段不屬於一般玩家路徑。詳細規則見
[`111` 全戰役原版一致目標](docs/goal/111-goal-original-parity-campaign-20260915.md)。


## Remake 里程碑

**`v.1.0.3-20260907` 已撤銷穩定版資格。** 實際玩家啟動揭露預設戰役未載入、
標題素材錯誤會落入預載第一關，以及完整版素材來源選錯等阻斷問題；該版只保留為
可追溯的失敗發行，不應下載作為可玩驗收。修正版必須重新通過「開頭動畫 → 標題
選單 → 第 0 章完整對話／演出 → 第一關」正常輸入與音樂驗收，並以新版本發布，
不得覆寫既有標籤。原附件、雜湊、問題與權利邊界見
[`v.1.0.3-20260907 發行說明`](docs/releases/v.1.0.3-20260907.md)。
[GitHub 歷史發行頁](https://github.com/wicanr2/fd2_re/releases/tag/v.1.0.3-20260907)
保留當時六個附件與 SHA-256 清冊，等待下一個通過正常玩家路徑的修正版取代。

玩家可隨時按 `F1` 查看操作說明；安裝現代素材包後，`F2` 可切換忠實原版／現代
手繪主題；`F3` 切換
Sound Blaster／MT-32 音源，`F4` 切換四種官方語言，`F5`／`F9` 快速存讀檔，
`F6` 開關音樂，`F7` 切換字級（80%–125%）。戰場空白格的系統選單另保留音樂、音效、
速度與狀態欄按鈕；
秘密商店的章節指定修飾鍵＋功能鍵不會被無修飾快捷鍵攔截。公開引擎包不含私人
現代逐幀素材，缺少素材時 `F2` 會保持原主題並顯示原因。
現代手繪主題需另行安裝已驗證的私人素材包；序章頭像與正常操作範圍見
[現代主題規格](docs/knowledge-base/61-modern-theme-runtime-spec.md)。
現行非HD完整版的素材範圍依[封包契約](docs/knowledge-base/41-packaging.md)為準。

先前的 [v0.1.1](https://github.com/wicanr2/fd2_re/releases/tag/v0.1.1) 與
[v0.1.0](https://github.com/wicanr2/fd2_re/releases/tag/v0.1.0)保留為歷史快照；
舊版推廣片規格見 [`docs/promo-v0.1.0-spec.md`](docs/promo-v0.1.0-spec.md)。

第一輪重製的 60／60 個分層代表性抽樣已符合正式路徑資格；這是 95% 信心的
第一輪里程碑，不代表 DOS 逐像素、逐音訊或逐週期一比一。詳細分級與限制以進度頁為準。

- **資產保存**：主要容器、圖像、字型、文字、動畫、地圖與音樂格式已有可重生工具。
- **可操作垂直切片**：標題、讀檔、戰場、玩家指令、敵方回合、城鎮、商店、教會、
  整備、存讀檔與多個戰後節點已接入正式 runtime（多數為 `RUNTIME-E1`）。
- **戰役後段**：第29／30戰、最終整備、終局文字、角色蒙太奇、20段尾段、定格與
  隊伍回顧已有連續重製端路徑；一般玩家同狀態 `PLAYER-E2` 尚未全部抽驗。
- **跨平台交付**：Linux AppImage、Windows ZIP、macOS universal ZIP 以同一完整版號
  重建並執行封包自我檢查；Windows／macOS 實體操作仍待驗收。
- **第一輪完成**：戰役／戰間、戰鬥／AI、介面、存檔／持續隊伍、終局／平台五層
  共60格已達最低配額且零完整性錯誤；後續以玩家回報與發行驗收為主。

里程碑的即時證據、分數與抽樣矩陣不在首頁重複維護，請看
[`FD2 第一輪重製進度`](docs/REMAKE-STATUS.md)。

![正常標題 LOAD、最終整備與戰前劇情後抵達第30戰](docs/figures/native-load-ch30-normal-input-remake-e1.png)

*此圖由正式標題 `LOAD` 的普通鍵盤路徑產生；使用固定雜湊第三方存檔與 dummy
音訊裝置，因此證明重製流程可達，不代表從第一戰完整通關或精確音訊 E2。*

![來源約束的角色終局蒙太奇](docs/figures/ending-campaign-montage-remake-e1.png)

*此圖由正式標題 `LOAD`、普通 Return／Space 與玩家自備原始資產產生；屬
`RUNTIME-E1`，不冒稱未修改 DOS 長程通關或精確硬體音訊。*

## 致謝

感謝漢堂國際資訊與《炎龍騎士團 2》原開發團隊，留下這部至今仍值得研究與遊玩的
作品；也特別記錄 AFM 工具署名者 **Lo Yuan Tsung**，讓三十多年後的人仍能從資料
理解當年的工程設計。

感謝保存實體遊戲、測試存檔、攻略、錄影與硬體聲音的玩家社群。攻略與影片只作
玩家可見行為旁證，不取代原始位元組、合法執行期實驗或同狀態比較。

本專案中的原創程式與文件由各貢獻者完成；原版名稱、程式及資產權利仍屬其權利人。

## 授權與商業使用

本倉庫中由專案貢獻者創作的程式碼與文件，依
[復古重製 source-available 授權條款 1.0（RRSAL-1.0）](LICENSE) 授權：非商業
使用、修改及散布免費；遊戲實況、錄影、評論、教學與平台一般分潤明示允許。
商業使用須事先與 [儲存庫擁有者](https://github.com/wicanr2)另行取得書面授權，
貢獻條件與署名要求以完整條款為準。

此授權不涵蓋、也不重新授權原版《炎龍騎士團 2》的名稱、程式、美術、文字、
音樂、音效或其他資產；這些內容不隨本倉庫散布，權利仍屬其各自權利人。
