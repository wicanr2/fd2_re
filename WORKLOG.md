# 工作歷程

## 2026-10-19 蓋亞原版導向造型修訂

使用者指出舊HD造型與原版壯碩機兵有差距，並確認新版概念稿後要求繼續。
從main的`71a8e473`建立`feat/gaia-hd-redesign-20261019`。依原版DATO30、FDICON30、
普通保存的BattleFig30與FIGANI090／091整理來源板，維持灰綠厚甲、藍色結構、短頸與粗壯四肢。
原版參照是解碼素材，不標成原版執行期截圖。核准概念原圖以原bytes保存，故事頭像直接裁取其右側；
四方向十二幀由內建圖像生成工具沿核准造型製作，逐幀完整輪廓審查後編製24／144輸出。
完整提示、參照hash、母稿與來源沿[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的`gaia_redesign_followup_20261019`，接入前先在61記READY契約。

僅替換蓋亞四個runtime清冊項，另加一份具獨立ID的核准概念圖；其餘360項清冊保持。
離線驗證允許多份不同ID的角色概念圖，重複ID仍拒收；schema及正式consumer不變。
新素材用第二版檔名，公開庫沒有逐幀圖或母稿。十二幀、比例、腳底、覆蓋／二值alpha與來源hash通過。
全部993份Go來源與前批相同；登錄前相關Go回歸19.378秒，登錄後48.429秒，清冊十項測試通過。
沒有遊戲runtime、規則、保存格式或亂數的改動，不重跑完整Go或matching。

正常START的r1誤把兩次牆鐘取樣相同當成文字已停，F2時進度15變24而失敗。
改用60個實際幀的穩定窗後，r2通過F2、97個對話拍點與七名480頭像GPU並到第一章。
尾端F9短按未取得ACK，原執行器r2仍列失敗；離線分段工具只核對已完成的START與原始PNG，
不補造整次通過。獨立34張首章GUI的普通走行、兩次攻擊、F2、F5／F9與新程序LOAD完整通過。
蓋亞另以普通Z選角、北移三格、取消、F2及新程序標題LOAD補抽，來源與保存bytes相同。
沒有座標／狀態注入、強化、鎖血或錄影；其他三個方向只作圖源審查，未冒稱全部正常行走取樣。

GPU首輪因母稿大部分內部alpha251..253，不能套索爾的不透明像素數門檻。
保留原覆蓋alpha，依已知source-over契約驗高覆蓋區；完全不透明RGB仍要求精確相等。
之後查出兩張PNG實際是frame0與2，Draw觀測皆報1，第三張前後相位不同。
本切片沿使用者已定案的流程一致範圍，只辨識同方向三個登錄步態中的唯一圖源，逐筆保留差異；
不修改遊戲時序，不宣稱相同幀，觀測與擷取相位對應的原因仍未知。
兩張分別330／329個不透明像素相符，8157／8053個高覆蓋像素符合混合界限，低覆蓋邊緣不作聲明。
錯收據範圍、錯角色與索爾圖源冒充蓋亞均拒收且不發布報告，原收據保持。

私人包3268檔與runtime-only3063檔核對通過，保留3239項原記錄，新增29檔，
現行包移除被取代的28檔；舊版仍可由私人提交`b0ddae47`回查，本地原圖亦保留。
核准概念稿byte-exact，來源／輸出與UID/GID1000:1000通過；私人提交
`984b284bda84023462785b6260c1bca7dee53117`已推送且真正遠端相同，保存庫保持private。
README只更新穩定現況，既有四語截圖與文化保存內容保持；大型FIGANI、完整HUD與其他地形仍待HD化。
官方#201／#202／#204已回填分段驗證與差異，均保持開啟；官方工作清單20項、13項人工判定、可能完成0項。
修改文件1359個本地連結與新工具入口通過，本輪19檔UID/GID1000:1000。
較廣擁有權掃描發現51個歷史root-owned檔，已核對內容與Git索引相同且本輪未修改，不改無關檔案擁有權。
登錄後另由記錄的基底清冊重生，新29檔bytes與365項素材資料完全相同，正式清冊保持；
`prepare --base-catalog`可重跑該路徑，不需回滾目前清冊。這是來源重生驗證，不重跑已通過GUI。
驗證用執行檔19,174,456 bytes已移除，沒有錄影或FD2容器殘留；正式第一版tag仍為原提交。
來源提交`27afc55c1321d7c6021299df03b2173e1cac503f`已快速前進至main並推送，真正遠端HEAD相同。
本批不宣稱完整HD或新Release完成，後續仍由官方HD工作項追蹤。

## 2026-10-19 相鄰綠色變體與褐色雙塊HD

從已推送main的`f96aabea`建立`hd-terrain-tree-variants-20261019`，先在61寫READY契約。
原生匯出器補map0(13,3)的44／47／46及(9,12)的48；六份物件參照25920像素逐像素核對，
33圖庫與前四份參照保持。兩份母稿由內建imagegen依既有手繪筆觸與原生幾何重畫。
褐色雙塊初稿的右下形體太大且偏左，經定向編修縮小並恢復間隔；拒收稿不登錄或同步。
綠色中段沿3列邊界校準，3／4／5格及正常戰場已檢視，alpha及未校準核心保持。
四張144底圖切片重組零差異，原360項清冊保持；草地及其他地形仍用原版。

GUI增加明示frame與標籤，避免每批新物件都加專用取景旗標。收據保存`terrain_expectation`，
GPU驗每張圖均含全部預期frame；props舊九槽門檻保持，舊收據以新驗證器重跑仍通過。
三項合成損壞範圍拒收測試不發布GPU報告、不修改原始收據，frame參數另驗五種不合法輸入。
正常首章37張GUI包含F2、走行／指令環／取消、兩次攻擊、Home、F5及新程序LOAD，
兩張GPU的全部四槽相符，完整相符圖塊72／74個；來源與完整保存SHA仍為`089b077b…`。
沒有注入狀態、鎖血、錄影、新oracle、matching、整章E2或Release。

初次Go回歸在登錄前通過40.323秒；364項清冊登錄後乾淨重跑相關回歸33.739秒，清冊9項測試通過。
只有原生來源匯出器的Go檔變動，遊戲runtime未改，不重跑完整Go；原有字串處置不需重綁。
私人包3267檔與runtime-only白名單3063檔逐檔驗證，原3261檔保持，僅新增兩母稿與四底圖。
私人提交`b0ddae470a9e6957f82ec3258484df308e3cfaec`已獨立推送並核對真正遠端，儲存庫保持private。
唯一證據、母稿路徑、完整提示、SHA與重跑方式見[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的`rounded_green_and_pair_followup_20261019`。完整HD仍未完成。

最終官方工作清單仍20項、13項人工判定，可能完成0項；250條教訓及131個守衛通過。
修改文件的本地連結與兩份新配方索引通過，原有九個歷史連結／解析器項目保持，不當成新缺陷。
素材及收據UID/GID保持1000:1000，已驗binary移除，保留母稿／圖源／PNG與收據；沒有FD2容器或錄影殘留。
來源提交`0ba78dfdd72e781cc856938b38ce5ee614e58c6b`已快速前進至main並推送，真正遠端HEAD相同。
#201／#202／#204已附來源提交留言，均保持開啟。

## 2026-10-19 尖頂綠色與褐色成組物件HD

從main的`bf280008`建立`hd-terrain-trees-20261019`，先在61寫READY契約，沿正式原生載入器補兩份物件參照。
綠色物件為map0(7,8)的40／43／42，褐色成組物件為(7,6)的203..208；名稱不推定樹種或用途。
33圖庫及先前兩份參照保持，四份物件參照逐像素回讀通過。內建imagegen依原生幾何與既有房屋風格重畫，
綠色初稿因中段第二尖頂拒收，定向編修改成單一連續輪廓。全部提示、母稿SHA與拒收來源沿主收據保存。

frame43會連續重複；先試12列邊界漸變造成條紋，改採3列。只混合新繪圖自身RGB，原生alpha與未修改核心保持。
3／4／5格組合與實際戰場已檢視，邊界列相同；切片零差異明示相對於UV及接縫校準後畫布。
新增九張144底圖及兩份母稿，原356項清冊保持。新增接縫／核心與六項拒收檢查通過，清冊9項測試通過。
相關Go回歸通過，主程式40.267秒；本輪僅改匯出器Go檔，遊戲runtime未改，不重跑完整Go。
最初測試後的build誤指模組根目錄，改用`./cmd/fd2`成功；不將命令錯誤列為遊戲缺陷。

清理空間期間暫停HD，已產生的正常GUI收據保留。恢復後先核對既有37張收據，沒有重跑正常操作。
普通標題LOAD、方向鍵取景、F2、走行／指令環／取消、兩次攻擊、Home、F5及新程序冷讀通過，沒有狀態注入或鎖血。
獨立GPU核對兩張圖全部九個新槽均相符，完整相符圖塊為56／57個。來源與保存SHA同為
`089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2`。
相鄰樹形變體、草地及其他地形仍是原版，混合美術照實保留；沒有新oracle、matching、整章E2、公開圖或錄影。

私人包3261檔與runtime-only白名單3059檔核對通過，原3250檔保持，新增兩母稿與九底圖。
私人提交`dbeff6216594ce59cd1c97a1d3f25045aeffa4f1`已推送並核對真正遠端，儲存庫保持private。
唯一證據、完整提示、SHA及重跑方式見[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的`green_and_brown_followup_20261019`。正式第一版保持原tag與Release，完整HD仍未完成。

收尾檢查250條教訓、131個守衛及20條官方工作清單通過，13項仍需人工，沒有可能完成而未結案的項目。
新增教訓守衛初次指到被掃描器排除的測試檔，改綁實際製圖函式後通過，測試本身另行執行。
本輪修改文件的2769個本地連結有效。較廣掃描4115個目標保留九個原有項目，
其中八個歷史封包缺件、一個陣列呼叫程式碼誤判；四份來源文件與HEAD相同，不回寫歷史。
原生來源、母稿、PNG及收據保留；已驗binary移除，沒有留下FD2容器或錄影。
來源提交`538ea592717b1df8168d41d8f2ff2e95e49d8e23`已快速前進至main並推送，真正遠端HEAD相同。
#201／#202／#204已附來源提交留言，均保持開啟。

## 2026-10-19 暫停HD並清理空間

依使用者要求暫停HD。確認沒有FD2容器仍在執行後，刪除三個可重建的Go建置快取：
`work/remake-recovery-20261015/gocache`、同目錄的`dos-cache`及
`work/parity-slot-ch12/treasure-string-review/gocache`。
逐一核對Go快取README、目錄及全部項目的UID1000，沒有跟隨符號連結。
共釋放5,341,085,696 bytes；刪除後當下可用空間21,931,683,840 bytes。
正式`dist-all/`、原始素材、私人素材庫、研究證據、HD母稿與收據保留，沒有全域清除Docker資源。
清冊保存在本機`work/promo-cadence-20261018/hd-fonts/space-cleanup-20261019.json`。

暫停時工作樹仍在`hd-terrain-trees-20261019`，未提交的新九槽與配方保留。
`props-gui-r1/receipt.json`已產生PASS，但獨立GPU逐圖核對、私人素材同步及正式文件收尾尚未完成。
不得將這批宣稱為完整HD或已完成交付。下一次恢復先核對現有收據，不重跑已完成流程。

## 2026-10-19 屋後遮擋、褐頂物件與AI輸入鎖

從已合併main建立`hd-house-occlusion-20261019`。先補屋後普通路徑契約，未改槽或章內狀態，
seed=1在啟動前固定。索爾依正式可達集合走(7,20)→(7,16)→(7,12)→(7,8)→(10,8)，
普通待機／END經過三次換手與增援對話，保留受傷，第四回合抵達屋後。
兩張實際GPU的前景279各16632像素相符；第一張2752個透明處人物核心相符，GPU色階容許1。
第二張人物來源相位不合，保留不可比。錯hash、錯opaque一像素、錯透明處人物核心、缺前景均會拒收。

前兩輪工具分別把turn當成陣營、把保留的banner文字當播放中，均保留失敗。
第三輪在AI持續行動時碰到180秒期限，不能稱遊戲卡死；另發現Return跨劇情交界後可重入選人，登錄#262。
實際鍵盤收據frame4703有ai_busy=true及move_preview，前一放鍵4699仍有增援對話。
依56既有同步順序先寫READY，再在Update與confirm鎖住AI期間地圖輸入；對話及F2／F9保持。
修復後四回合的4709筆觀測沒有AI期間move_preview，相關回歸與完整Go通過，主程式248.480秒。

第四輪的屋後／F2已完成，但後續F9仍沿用同類banner文字判定，整輪失敗保留。
修正同一工具全部五個同類等待後，以同份binary獨立重跑普通標題LOAD、F2、走行、指令環、取消、
兩次攻擊、Home狀態／命令、保存及新程序冷讀通過。分段證據不包裝成單次長流程PASS。
完整保存及來源SHA均為`089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2`。

原生匯出器沿正式selector補相鄰褐頂物件的2×4格參照，不推定物件用途。
完整33圖庫來源與兩份物件參照均獨立回讀；褐頂物件由內建imagegen完整重畫，明示UV校準後切出八底圖與兩前景。
所有alpha保持原生遮罩，切片重組零差異；兩個前景與base的opaque索引相同，使用相同新母稿。
正式Game.Draw的兩張GPU均包含全部八張新底圖，RGB相符；外緣仍與原版草地混合，未宣稱完整地形HD。
356項清冊及9項工具測試通過，原354項清冊內容保持；新素材只同步私人庫，未新增公開圖或錄影。
私人包3250檔逐檔核對，runtime-only白名單3050檔；遠端HEAD為`4be2f35a4856dd377846692acabf7e8cbacb8175`。

唯一收據與完整提示、SHA、來源及重跑命令見[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的`house_occlusion_and_roof_followup_20261019`。新母稿保存於本機及私人包的`map00-roof-object-hd-v1-master.png`。
其餘前景、地形、完整HUD與早中晚期／平台仍待完成，未新增原版oracle、matching或整章E2。

提交前249條教訓及130個守衛、2758個本地連結與新工具索引正對照通過；993份Go來源SHA保持。
本輪輸出與現代素材沒有root-owned或`.md`目錄，FD2驗證容器已退出。
已清理兩份驗證binary、Go快取與未採用的重複圖源，共321502588 bytes；必要來源、PNG與收據保留。

來源提交`bccba29b9192ce377f7f8d1f4fcc5368d646ec6b`已快速前進至main並推送，真正遠端HEAD相同。
#262已附提交與驗證留言後結案；#201／#202／#204同步現況並保持開啟。
結案後官方快照20項、13項人工判定，可能完成0項；正式Release及tag保持原值。

## 2026-10-19 HD房屋正式世界與正常GPU

從已合併main建立`hd-house-runtime-20261019`。先在61寫READY接線契約，再登錄24底圖及6前景，
沿原生selector、terrainBase／viewportBase與一次時鐘交易接Game.Draw。原生range另疊在新圖上方；
前景保留原排程，只有已登錄槽使用144圖。新RGB依原生調色盤控制點作明示插值近似，alpha及規則不變。
未登錄地形仍沿原生圖，正式Release不變。

正常首章標題LOAD後用普通方向鍵移鏡頭，拍房屋HD、忠實模式及切回HD；再F9回來源繼續原有走行、
取消、兩次攻擊、Home狀態／命令及保存冷讀。兩張實際1920×1200視窗各有22／24個完整圖塊與發布RGB相符。
全程不注入座標／狀態，seed=1於啟動前固定；保存與來源完整bytes相同。
前景正常遮擋與房屋在鏡頭內的走行仍待補，上移只列部件E1，不新增原版對拍或整章E2。

首輪鍵盤工具每100ms才查看按下收據，左鍵在放開前多移一格。正式規則沒有改動；
工具縮短至5ms輪詢後，同一binary乾淨重跑通過。首輪鏡頭沒拍到房屋，也未當成地形通過證據。
房屋外緣及切片接縫已目視審查；原版草地與新房屋的美術差異仍可辨識。

新素材只按catalog白名單同步私人保存庫，另提交；公開庫保留程式、契約與SHA，沒有新增公開圖或錄影。
完整驗證、來源雜湊、私人HEAD及重跑入口見[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的`house_runtime_followup_20261019`。後續接正常前景遮擋及其他地形；#201／#202／#204保持開啟。

完整Go命令中，主程式258秒及其餘遊戲／資料套件通過；只有字串清冊舊來源SHA失敗。
官方遷移工具核對99筆審查ID與決定完全不變，重綁來源SHA後該套件獨立通過；
原始整套命令保留FAIL，沒有為純清冊綁定再重跑全部遊戲。清冊354項及9項工具測試通過。
私人包3239檔逐檔核對，runtime-only白名單3040檔，遠端HEAD為`2d1460d1eaedbd7e62678e4e4ff8ba3c556ca4b6`。

來源提交`0eb9cf6ee9d466e359736fd89419900273001640`已快速前進至main並推送，真正遠端HEAD相同。
#201／#202／#204已更新且保持開啟，#261維持結案；官方快照20項，13項人工判定，可能完成0項。
247條教訓及128個守衛通過；2745個本地連結與新工具索引正對照通過。
992份Go來源SHA保持；輸出與現代素材沒有root-owned或`.md`目錄，本輪FD2容器全部退出。
已清理binary與Go快取239574982 bytes，最終圖片、來源與收據保留，沒有新增錄影。

## 2026-10-19 原生地形製圖來源與HD房屋候選

新增Go離線來源匯出器，直接重用正式FDSHAP載入器、FDOTHER#0調色盤及既有terrain／foreground selector。
全圖庫逐幀保存索引、opaque、destination-remap及基準RGBA，PIL獨立回讀全部產物；不沿用舊正規化RGBA。
來源分層、統計與SHA以[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)及
[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`native_terrain_source_followup_20261019`為準。
舊圖集的歷史解碼根因保留未知，既有檔案與證據不刪除；新製圖來源已由正式bank重生，不重開matching。

內建image_gen依原生房屋參照完整重繪，再用可編輯UV配方對齊屋簷、牆腳與開口，切成144候選。
母稿校正仍有邊界偏移，因此以明示取樣校準處理；不把手工錨點標成原版二進位事實。
初次編製攔下前景279與base278的opaque RGB差異，補獨立母稿後通過，不放寬同源檢查。
實際PNG透明覆蓋保持，切片重組零差異；比較圖已審查，沒有把圖塊間接縫當成新增筆觸。

Go來源工具與fdicon／fdother回歸通過；末張圖庫無效mask、不同前景缺來源及既有目錄保護三項實際拒收通過。
正式遊戲Go、存檔與素材清冊不變，沒有新GPU、正常玩家或原版對拍收據。
房屋仍為DATA-CANDIDATE；下一步接清冊與Game.Draw，再驗第一章正常LOAD、前景遮蔽及外緣接縫。
新圖與原生製圖來源均留本機，未加入公開庫或Release，未同步私人正式主題包，沒有錄影。

來源提交`b086604f077810ce6dc3dda1f86da241f3a86f64`已從新分支快速前進至main、推送並核對真正遠端。
[#261](https://github.com/wicanr2/fd2_re/issues/261)以替代HD製圖來源的DATA-READY範圍結案，
不宣稱舊圖集已修復或HD已接正式世界。#201目前內文改為精確現況，舊批次證據仍在主收據及本工作歷程；#204保持開啟。
本輪官方快照為20項、13項人工判定、可能完成零項；246條教訓、128個守衛通過。
修改文件2625個本地連結通過，9個既存歷史訊號保持。工具版本從容器實讀為Go1.22.12、Python3.12.15及Pillow11.3.0。
清理初始重複圖源、已驗測試binary與Go快取共68752773 bytes；保留最終來源、母稿及候選，本輪FD2容器均已退出。

## 2026-10-19 HD地形圖源審查與草地原型

沿已核准手繪風格以內建image_gen產生草地母稿及一次修訂，原圖、提示與SHA保留本機。
受版控工具逐層核對正式FDSHAP分離圖庫與舊正規化圖集，發現alpha及同索引RGB差異；
唯一現況統計見[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)，逐圖與輸入綁定見
[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的`terrain_source_followup_20261019`。
根因尚未判定，未改原版資料、正式renderer、保存、素材清冊或Release。

草地直接替換的比較有接縫，來源顏色遮罩加細節仍有明顯區塊感，均未採用；
它們只比較固定章資料的排版，不當作正常玩家、GPU或原版對拍。
舊READY圖集契約已限縮為正規化24消費端，不再推定與原生圖層等價；
後續原生HD圖源沿正式bank、既有動畫／前景selector及三個原生圖層閉合。
新增[#261](https://github.com/wicanr2/fd2_re/issues/261)追蹤來源差異，不重開全檔matching。

工單Python主機呼叫被自動審查以Docker-only拒絕後，改在容器使用官方格式器驗證內文，
再由允許的主機gh登錄。後續快照也分成主機gh讀取與容器內官方build_snapshot／render／verify。
沒有錄影或新正式素材包。

錯誤來源指紋、過小材質與既有輸出保護三項拒收檢查通過。官方worklist為21項、14項人工判定、
可能完成零項；245條教訓、127個守衛保持。修改文件2617個本地連結通過；
全儲存庫掃描另有9個既存訊號，涉及未改動歷史交付連結及行內運算元，原文與HEAD一致。
主收據保留各項訊號，未宣稱全庫零斷鏈。正式Go與素材不變，因此未重跑遊戲測試或GPU驗收。

目前分層狀態以 [58](docs/knowledge-base/58-fd2-exe-re-coverage.md)為準，完整匹配範圍依 [118](docs/goal/118-goal-fd2-matching-decompilation-20261008.md)。既有時間序列保留於[工作清單歷史](docs/knowledge-base/91-worklist-history.md)與該頁引用的研究紀錄。

## 2026-10-09 原始 ESI 輸入的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_class_stack.c)與[回歸](tools/test_fd2_matching_input_esi.py)，登錄於既有編譯驅動器的 `game_class_stack`。`sub_12DAC` 完整 140 bytes 匹配。輸入 ESI 在原始指令先讀後寫，保留其未知初值，不補零。
- 14 個表示模型各編譯 3s／4s／5s 與預設／space／speed，共 126 份候選，獨立重建兩輪。所有 OMF、完整報告及產碼相同，126 份另經 objcopy 核對。五份匹配只增加一個原始函式；31385／2DF6B 的模型仍未匹配。
- 全檔組合保留舊 75 份來源報告的順序，新報告最後加入。舊 185 段 C 出處逐項相同；SDK 19 函式的出處保持，SDK 原生連結與全檔收據另重建兩輪。完整 357074-byte EXE 的 SHA-256 與固定輸入相同。其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整原始函式、ESI 先讀後寫、EBX／ESI 保存與 16-bit 有號讀取。缺來源、來源內容與來源雜湊錯誤均於輸出前拒收。既有符號／LE 反例回歸首次誤用未含 190AC 的新物件，改以原有 `treasure-fixed-owner-r1` 乾淨重跑通過，未改守衛。
- 原作參數宣告、輸入 ESI 的來源與實際 GS 修改仍未知。`parm`／`modify exact` 僅為相容產碼表示。已知位址由既有 IDA 9.4 匯出機械核對，不重開既有隊伍／面板語意。正式 Go／Ebiten 與章驗收等級保持。

提交前核對 102 份來源雜湊、1205 條本地連結、191 條教訓與 76 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。原始 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批一次性 FD2 容器均已退出。提交與真正遠端 HEAD 結果回填 [#198](https://github.com/wicanr2/fd2_re/issues/198)。

原始位址、雜湊、工具版本、caller、精確旗標、完整候選與驗證結果保存在[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_class_stack_matches`、`class_stack_original_evidence`、`validation.class_stack` 與 `bootstrap`。公開提交保存自行重寫的 C、工具及整理證據；完整 EXE、SDK 物件與工具二進位留在本機 `work/matching-full-20261008/`。

編譯與連結沿用[驅動器](tools/fd2_matching_game_restore.py)：在既有 Docker 工具鏈內執行 `compile --source-key game_class_stack --output /work/class-stack-final-r1`，再執行 `link --objects /work/class-stack-final-r1 --evidence /work/ida-full-image.json --output /work/class-stack-final-linked-r1`。容器與唯讀編譯器掛載方式沿用主收據的工具鏈紀錄。第二輪使用新的 `r2` 輸出，不覆寫第一輪。

## 2026-10-10 五個 DOS／檔案流程的完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_dos_file.c)與[完整函式回歸](tools/test_fd2_matching_dos_file.py)，登錄於既有驅動器的 `game_dos_file`。361CC／36284／36344／36900／36955 完整 137／109／311／85／173 bytes 匹配，共 815 bytes。
- 36284 保留兩個分開的無號條件。36344 保留 204-byte 局部布局、早期錯誤返回及下一次讀取位置。其餘三項以相容 callee 宣告保留原始 EBX 跨呼叫存活，沒有 inline code 或位元組補丁。原始 unknown 分類、作者宣告與完整間接副作用集合仍分開記錄。
- 14 個模型共 126 份候選，獨立編譯、連結兩輪，OMF、完整報告與產碼相同；126 份全部由 objcopy 獨立核對。17 份正例只計五個原始函式；其餘模型保持負例。
- 舊 76 份來源報告順序保持，新報告最後加入。舊 186 段 C 出處逐項相同；SDK 19 函式／1200 bytes 出處保持，在新 C 基準實際重跑原生連結兩輪，全檔與收據相同。完整 EXE 的大小及 SHA-256 與固定輸入相同，其餘原版 code 尚未還原，#198 與 Goal 保持開啟。
- 回歸核對完整 815 bytes、原始無號分支、堆疊布局與指標跨 close 存活。十一項原始名稱／CDECL 修飾別名錯址及一項原始 LE 矛盾均拒收，CLI 矛盾在輸出前停止。既有符號、明示位移與 LE 守衛通過。
- 第一輪連結曾缺少 CDECL 前置底線別名。直接檢查 COFF relocation 後補上受原始 caller／LE 核對的別名，未放寬守衛。將錯誤返回改回原始布局後，完整函式才收件。已知位址沿既有 IDA 9.4 匯出機械核對，沒有重開玩家語意或提升正式 Go／Ebiten 與 PLAYER-E2。

精確正例旗標、原始函式與 caller、來源雜湊及驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_dos_file_matches`、`dos_file_original_evidence`、`validation.dos_file` 與 `bootstrap`。編譯與連結使用[既有驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_dos_file --output /work/dos-file-final-r1` 與 `link --objects /work/dos-file-final-r1 --evidence /work/ida-full-image.json --output /work/dos-file-final-linked-r1`，第二輪使用新的 `r2` 輸出。公開保存重寫 C、工具與整理證據；完整原版／SDK 二進位留本機。

提交前核對 104 份來源雜湊、1212 條本地連結、192 條教訓與 77 個守衛。真正主機 Issue 清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器已全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。

## 2026-10-10 狀態群組與原始載入位元組守衛

- 新增 [C 候選](tools/fd2_matching_game_state_group.c)及[完整群組／載入位元組回歸](tools/test_fd2_matching_state_group.py)。Watcom 10.0a 與 9.5 各 261 份候選均未完整匹配，獨立編譯與連結兩輪，OMF、完整報告及產碼相同；522 份全部由 objcopy 核對。原有 C 覆蓋、191 段出處及 19 段 SDK 出處保持。
- 22AF6 的原始欄位比較使用 EAX，基準候選使用 ESI，差異限於三個編碼位元組。固定 C 的 3s 預設四函式群組中，9.5 保留原始條件跳躍及 RET；10.0a 額外共用 22AA7 的 RET。最後 27 bytes 局部相同仍不收件，不推定原作精確版本、作者宣告或新玩法。
- 隔離反例只改匯出的三條 `loaded_bytes`，舊比較器會誤報 22AF6 匹配。原版與正式匯出保持不變，偽造輸出不計覆蓋。新增守衛從原始 `file_bytes` 及真正 LE fixup 重算每條載入指令；正式 75,427 條指令與 7,003 個 CODE 重定位位置全部相同，三項改寫與 CLI 偽造在輸出前拒收。
- 舊有 815-byte 五函式正例重新連結及回歸通過，報告除新驅動器雜湊外完全相同。既有符號、LE 與明示位移守衛通過。[56](docs/knowledge-base/56-fd2-remake-sdd.md)、[57](docs/knowledge-base/57-ui-evidence-matrix.md)與[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)同步說明修正範圍，正式 Go／Ebiten 與 PLAYER-E2 保持。
- 261 份 9.5 候選兩輪曾在固定 90 秒內只完成 217／215 份物件，屬批次時限不足。驅動器依 `max(90, min(600, 30 + trial_count))` 設定有界時限，寫入 `runner_timeout_seconds`，同 C 與同候選的兩輪 291 秒批次均完成。失敗目錄保留，不算 C 不匹配。

來源、精確旗標、原始函式、修正前後反例與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_state_group_attempts`、`state_group_original_evidence`、`validation.state_group`、`validation.loaded_image_guard` 及 `validation.compiler_batch_bounds`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_state_group --compiler-version 10.0a` 或 `9.5`，各用新的輸出目錄，再以固定原檔及 IDA 匯出執行 `link`。全檔 C 反編譯尚未完成，#198 與 Goal 保持開啟。

提交前核對 106 份來源雜湊、1229 條本地連結、194 條教訓與 79 個守衛。真正主機清單經官方 pull／render／verify，16 條仍開啟，8 條需人工判定，可能已完成 0 條。正式 IDA 匯出與資料庫雜湊保持，變更檔與新產物 UID／GID 為 1000；歷史 root-owned 共 2811 項保持，沒有 `.md` 目錄。本批 FD2 容器全部退出。提交、推送及真正遠端 HEAD 核對結果回填既有 #198。

## 2026-10-10 開場與戰後較大函式完整 C 匹配

- 新增 [C 來源](tools/fd2_matching_game_long_sequences.c)與[完整序列回歸](tools/test_fd2_matching_long_sequences.py)。3231B 完整 1626 bytes 及 2548C 完整 715 bytes 匹配，共新增兩個原始函式2341 bytes。既有開場／戰後語意沿用原證據，原始分類、作者宣告、正式 Go／Ebiten 與 PLAYER-E2 保持。
- 六個模型各比較三種 CPU 與三種成本策略，共54份候選，獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由 objcopy 核對。25份正例只計兩個原始函式；32-bit開場計數控制未匹配。
- 開場保留 BL 的8-bit計數、15／13次迴圈、所有呼叫與32／31／0全域寫入順序。戰後保留35BBA後讀取指標、原始欄位寫入與兩個淡出／淡入迴圈。初輪指標模型曾提前讀取，已依原始順序修正並新增回歸；缺少32975的外部綁定已按直接call補齊。宣告及volatile只作相容產碼表示，延遲呼叫參數不證明硬體wall-clock。
- 舊77份來源報告順序保持，新報告最後加入。舊191段C出處及SDK19函式／1200 bytes出處逐項相同；新C基準實際原生連結兩輪，全檔與收據相同，完整EXE大小及SHA-256與固定輸入相同。其餘原版code未還原，#198與Goal保持開啟。
- 176列原始具名參照、兩個完整IDA函式及caller保存；五項新符號錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。較早六份9.01單輪候選均未匹配，另經objcopy核對，不納雙輪小計。
- 發布前核對發現IDA的caller清單來自間接派送關係，沒有E8直接呼叫。改以原始LE表項及四個scale4間接call位置保存consumer；初版文件產生器的直接caller假設在寫入主收據前已拒收，沒有提升錯誤證據。

精確旗標、原始函式、來源與驗證結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_long_sequences_matches`、`long_sequences_original_evidence`、`validation.long_sequences` 與 `bootstrap`。單輪9.01比較在 `early_901_comparisons`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_long_sequences`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具及整理證據，完整原版／SDK二進位留本機。

提交前核對108份來源雜湊、1229條本地連結、195條教訓與80個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出及資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 演出／介面C與堆疊實參核對

- 新增[C候選](tools/fd2_matching_game_animation_panel.c)、[實參與保存回歸](tools/test_fd2_matching_animation_panel.py)及[IDA導覽匯出器](tools/ida_export_fd2_local_navigation.py)。28784完整744 bytes與2EBE0完整943 bytes的13個模型，由10.0a／9.5各編譯117份，兩輪OMF、完整報告與產碼相同，234份全由objcopy核對，均未完整匹配。C覆蓋、193段C出處及19段SDK出處保持。
- 原始288E5的`[esp+24h]`與288E9的`[esp+28h]`在不同PUSH之後，都讀caller局部基準+18h。初版C誤傳saved指標的位置已修正為兩個figure來源，原始運算元與有效偏移分列，不用運算元表面差異推定不同來源。
- 演出原始EDI在迴圈初次判斷前沒有局部初始化；來源保留未知，不補成零值。29164兩個已核對返回尾端均還原EDI。三個graphics callee也實際保存EBX；候選pragma允許修改EBX只作保守compiler契約，不能說成實際clobber或原作者宣告。

零值初始化只作`ANIM_LOCAL_2`的未匹配對照，不當作原始初值。
- 原始36／56-byte局部配置、16-bit容量讀取、三個真正E8 caller窗口、107列原始參照與十項錯址拒收回歸保存。既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。兩份WPP前端單輪對照也未匹配，明示前端及固定包裝，另列且不納C前端雙輪小計。
- IDA9.4沿既有py312映像，以正式i64唯讀掛載後的一次性複本輸出導覽。受版控工具重生原檔／DB SHA一致、1305函式、schema及UID核對的JSON；Hex-Rays型別、變數名與推測stack位置只作導覽，不當原作者或位元組證據。技能入口的符號連結指向工具專案，已沿真正權威路徑讀取，沒有另建映像。
- 空的`parm []`後不能再列EDI的編譯語法失敗已修正為EDI輸入在先、堆疊參數在後，機器輸入位置保持。未完成編譯與其後缺收據的診斷不計負例。較早2A2E8的旗標依賴尚未建立C候選，沒有猜補或改列原作組語。

完整來源與結果見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_animation_panel_attempts`、`animation_panel_original_evidence`、`validation.animation_panel` 與 `animation_panel_cpp_supplement`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_animation_panel --compiler-version 10.0a` 或 `9.5`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，完整原版、i64／導覽及SDK物件留本機；#198與Goal保持開啟。

提交前核對111份來源雜湊、1238條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 戰後對話與資源切換完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_post_dialogue.c)及[備份／副作用與派送回歸](tools/test_fd2_matching_post_dialogue.py)，24754完整960 bytes匹配。三個C表示都重現三個17-byte局部備份、分支與章節INC在context推入之後／CALL之前的原始順序，不能由相同產碼推定原作者副作用寫法。
- 27份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全部由objcopy核對。3s預設／speed的六份正例只計一個原始函式；其餘CPU／成本策略按完整區間拒收。
- 舊78份來源報告順序保持，新報告最後加入。舊193段C出處及SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同，357074-byte EXE的SHA-256與固定輸入相同。原版其餘code尚未還原，#198與Goal保持開啟。
- 72列原始具名參照、完整IDA函式與LE表項22／25E23的scale4間接consumer保存。原始60-byte局部配置、4×MOVSD＋MOVSB三次複製、fade步進2／上限64、六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。既有戰後語意、原始分類、正式Go／Ebiten、PLAYER-E2及硬體時序保持。
- 另查237D5為含多個入口與區間外共用尾端的owner，沿既有覆蓋與完整區間政策保留限制，沒有將入口片段裁切成完整C匹配。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_post_dialogue_matches`、`post_dialogue_original_evidence`、`validation.post_dialogue` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_post_dialogue`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對113份來源雜湊、1245條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 六模式分派完整 C 匹配

- 新增[C來源](tools/fd2_matching_game_overlay_dispatch.c)及[條件邊／記錄寫入回歸](tools/test_fd2_matching_overlay_dispatch.py)，122DC完整1051 bytes匹配。if鏈及逐分支返回兩種表示都在3s預設成本策略下匹配；原始分類、型別與作者宣告未知，既有語意保持。
- 18份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全部由objcopy核對。初次switch探針在CODE前置24-byte跳表，函式入口不在起始，工具拒收；該未完成探針不列雙輪總數，也不增加覆蓋。
- 核對六個原始模式條件邊、37個明示helper CALL、兩個返回位置及第五模式共用CALL目標。模式6保留原始四位元組記錄的+7寫入，不把C型別或offset推成新語意。125列原始參照、完整IDA函式及11CF0／18BF7兩個E8 caller窗口保存；六項錯址、既有LE／位移與75427筆載入記錄／偽造拒收回歸通過。
- 舊79份來源報告順序保持，新報告最後加入。舊194段C出處與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256與固定輸入相同，其餘原版code尚未還原，#198與Goal保持開啟。
- 篩選時另外核對32D18、244B6、234BB等完整owner；多入口、額外chunk與區間外共用尾端不裁切計數。這些已知布局限制未改動既有分類，也未重開戰役語意或正式Go／Ebiten。

精確旗標、來源與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_overlay_dispatch_matches`、`overlay_dispatch_original_evidence`、`validation.overlay_dispatch` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_overlay_dispatch`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對115份來源雜湊、1252條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 音效過場、事件尾呼叫與對話備份候選

- 新增[過場C](tools/fd2_matching_game_sound_transition.c)、[事件C](tools/fd2_matching_game_tail_calls.c)與[完整回歸](tools/test_fd2_matching_tail_and_backups.py)。3396A／34C76／35487完整324／61／86 bytes匹配，三個E9均到完整callee入口，由C自然產生。15份正例只計三個原始函式，共471 bytes。
- 另保存[對話備份C](tools/fd2_matching_game_dialogue_backups.c)的九種表示。165AC完整688 bytes的81份候選均未匹配；參數槽、求值順序與暫存器配置仍有差異，聚合及volatile表示不證明作者宣告。後續先核對callee保存契約，再決定相容宣告，不裁切相同片段或提高原始分類。
- 三組108份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。早期探索來源只留本機，正式矩陣由目前受版控來源乾淨重跑，未完成探針不列入小計。
- 核對三筆LE表項：3396A為51D71表項24，由25E3A消費；34C76／35487為51B91表項35／56，由19511消費。表項檔案偏移與IDA線性位址分列。過場只載入#88 selector1，保留一般玩家指令#80的既有證據；三次delay實參600與20／20／20／60等待值不推成硬體wall-clock。
- 原始42列對話參照、兩個E8 caller窗口、兩次有號除法與sum零值旁路、五次26668-byte配置、五次備份／繪製及返回53A18保持。七項備份錯址、三個錯誤尾目標、既有LE／位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊80份來源報告順序保持，兩份正例報告最後加入；195段舊C與SDK19函式／1200 bytes出處保持。新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256仍等於固定輸入；其餘原版code尚未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_sound_transition_matches`、`game_tail_event_matches`、`game_dialogue_backup_attempts`、`tail_and_backups_original_evidence`、`validation.tail_and_backups` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_sound_transition`、`game_tail_events` 或 `game_dialogue_backups`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對119份來源雜湊、1265條本地連結、196條教訓與81個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 START／LOAD完整主流程與對話保存契約

- 新增[主流程C](tools/fd2_matching_game_start_load.c)及[欄位／重試回歸](tools/test_fd2_matching_start_load.py)，25EBB完整663 bytes匹配。槽位址的LEA排列與零實參表示共同重現原始產碼；不能由相同產碼推定作者C表示或型別。
- 22987-byte緩衝、12587-byte header、2600-byte槽、2560-byte記錄複製與七個原始欄位讀取保持。+2欄位為未對齊4-byte載入，其餘指定欄位為byte；255轉selected零值後回26016，原始51D71間接表與25DBD直接caller保留。
- [對話C](tools/fd2_matching_game_dialogue_backups.c)新增五個宣告候選，共45份完整負例，原81份歷史負例保持。原始4E031只改AX並還原ESI；4E96F以PUSHA／POPA保存通用暫存器；15E71保存EBX，15E9E經22BBE共享收尾還原EBX。保守pragma允許修改與實際callee保存分列，不宣稱原作者ABI。
- 兩組90份候選完整編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。七項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 首輪pack(push,1)／pack(pop)被10.0a以E1054／E1009拒收，未產生完整編譯收據；不計成產碼負例。改用pack(1)／pack()後乾淨重跑，sizeof斷言確認10／2600 bytes。教訓與守衛寫入既有[台帳](docs/data/fd2-lessons.json)。
- 舊82份來源報告順序保持，新正例最後加入。198段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持；其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_start_load_matches`、`game_dialogue_backup_contract_attempts`、`start_load_original_evidence`、`validation.start_load` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_start_load`；對話補充用 `game_dialogue_backups --cases DIALOGUE_BACKUP_9 DIALOGUE_BACKUP_10 DIALOGUE_BACKUP_11 DIALOGUE_BACKUP_12 DIALOGUE_BACKUP_13`。各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對121份來源雜湊、1274條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 終局對話完整C與1718-byte owner候選

- 新增[對話C](tools/fd2_matching_game_terminal_dialogue.c)、[owner C](tools/fd2_matching_game_terminal_body.c)與[完整回歸](tools/test_fd2_matching_terminal.py)。2C39B完整106 bytes匹配，保留A9514h目的位址、原始呼叫順序、EBX保存及2BCE5內12個直接caller。兩種保守宣告的四份正例只計一個函式。
- 2BCE5完整1718 bytes仍未匹配。八種表示保留三份20-byte表、84-byte局部布局、40／200／20迴圈、有號除法、原始欄位寫入及兩個外層caller。局部相同不增加覆蓋；原有終局語意、分類與一般玩家驗收保持。
- 兩組99份候選獨立編譯、連結兩輪，OMF、完整報告及產碼相同，全由objcopy核對。72份owner候選均負例，預設對話宣告103 bytes也按完整區間拒收。
- 4E031實際只改AX，1956B及4E63D保存EBX的原始指令另存；保守pragma允許修改不當作實際clobber或作者ABI。九項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 初次連結因sub_20421未登錄被ld拒收，已依原始E8目標補齊，不計成產碼負例。回歸初稿假設預設對照104 bytes，實際物件為103 bytes；已改讀實際收據，完整位元組負例檢查保持。兩者屬工具／驗證問題，不列為產品缺陷。
- 舊83份來源報告順序保持，新正例最後加入。199段舊C與SDK19函式／1200 bytes出處保持；新C基準實際重跑SDK原生連結兩輪，全檔與收據相同。357074-byte EXE的SHA-256保持，其餘原版code未還原，#198與Goal保持開啟。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_terminal_dialogue_matches`、`game_terminal_body_attempts`、`terminal_original_evidence`、`validation.terminal` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_terminal_dialogue` 或 `game_terminal_body`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開保存完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對124份來源雜湊、1283條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 原廠SDK完整群組與額外來源

- 先核對剩餘分布及固定SDK證據，未將未知函式按名稱或相似片段升格。新增[來源工具](tools/fd2_matching_library_extension.py)與[拒收回歸](tools/test_fd2_matching_library_extension.py)，只補既有保留函式庫的來源。
- 原生重建ioalloc／setvbuf／unlink完整119／120／36 bytes，import由既有runtime定位及原始參照逐筆核對。另重建amodf整個302-byte CODE，兩個公開入口0／144及三個完整IDA函式144／75／83 bytes共同驗證。沒有裁切、遮罩、補opcode或改原始名稱。
- 新增六個來源、577 bytes；SDK來源由19函式／1200 bytes增至25函式／1777 bytes。C199函式／29424 bytes、另146-byte未歸屬區塊、175個保留函式庫及931個待還原／分類函式保持。這些來源原本已列為函式庫，不增加C覆蓋。
- 兩輪實際WLINK／WDIS重建的全部CODE、完整報告、收據及357074-byte EXE相同。舊19個SDK來源、200段C出處與完整函式台帳逐筆保持；原始固定EXE SHA-256相同，Goal與#198保持開啟。
- 十項未知／C目標、C出處重疊、裁切邊界、移動或歧義公開入口、偽造載入位元組、跨chunk與物件來源變動均在輸出前拒收。原作精確版本與高階函式語意仍未知。
- 首輪錯把RAW的OFFSET對齊基準當作容器起點，長度守衛拒收。原生map證實唯一CODE從4D7BC起，RAW本身就是完整302 bytes；已依既有工具鏈契約修正並從新目錄重跑，整段比較未放寬。

契約先登記於[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `sdk_extension_spec`，驗證後為CONFORMED；來源與結果在 `sdk_library_extension_origin`、`validation.sdk_library_extension` 與 `bootstrap`。後續C基準先沿用既有library_restore／library_native_restore形成19函式SDK基準，再執行來源工具。CLI的 `--baseline` 與 `--receipt` 指向該中間基準，`--objects`、`--library`、`--evidence`、`--original` 指向固定本機輸入，`--output` 每輪用新目錄。原廠物件、完整CODE／map與EXE留本機；公開工具、雜湊、原始定位與有限caller證據。

提交前核對126份來源雜湊、1290條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品完整清單與執行流程C匹配

- 新增[C來源](tools/fd2_matching_game_ai_item_execute.c)及[清單／byte／夾限回歸](tools/test_fd2_matching_ai_item_execute.py)，15055完整700 bytes匹配。兩種局部整數重用表示同碼，不推定作者的volatile、pragma或局部變數宣告。
- 保留36-byte局部、byte32讀寫、14818／149F8的完整target list與count，以及交給20C6F的原始順序。小command分支在152E5共同接續；兩個14EF0 caller、八位元減16、signed X／Y夾限、64起始漸暗與1..8閃爍保持原始指令證據，沒有新增效果或E2結論。
- 4E56C實際只改EAX／EDX，4DBFC實際保存EBX／ESI／EDI；保守允許修改的C宣告分開標示。物品欄位與record活躍區間、X暫存值及Y局部重用共同重現原始配置。相同700-byte長度的負例仍按完整區間拒收。
- 90份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；兩份正例只計一個原始函式。八項錯址、既有LE／常數位移及75427筆載入記錄／偽造拒收回歸通過。
- 舊84份來源報告順序保持，新正例最後加入。200段舊C出處逐筆保持。SDK完整11／19／25來源鏈在新C基準實際重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項SDK未知／C／裁切等拒收在新基準通過。擴充輸出的六個來源是相對19函式中間層，與上一輪25來源比較沒有新增SDK覆蓋。
- 357074-byte全檔與收據相同，SHA-256等於固定輸入。累計200個原始函式以C匹配，其餘930個函式待還原或分類；Goal與#198保持開啟，正式Go／Ebiten與既有AI效果驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_ai_item_execute_matches`、`ai_item_execute_original_evidence`、`validation.ai_item_execute` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_ai_item_execute`，各輪用新的輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈接續重建；公開C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對128份來源雜湊、1296條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 AI物品評分候選與REGS完整C匹配

- 新增[評分C](tools/fd2_matching_game_ai_item_select.c)、[REGS C](tools/fd2_matching_game_int31_regs.c)與[完整回歸](tools/test_fd2_matching_ai_item_select_regs.py)。36255完整47 bytes匹配，保留56-byte REGS、第三參數32-bit讀取後16-bit遮罩、兩段相距28 bytes與int386原始呼叫。三種C表示的18份正例只計一個函式；四個owner內八個直接caller另存，原分類unknown保持，不推定高階INT31h服務或原作者宣告。
- 1567E先核對既有game_ai_sequences來源與歷史負例，再依已知callee保存契約建立18種表示。162份完整候選均未匹配。部分候選與原版同為514 bytes，只差actor乘80的七個bytes，仍按完整區間拒收；C++四份單輪完整負例另列，不併入C雙輪統計。
- 72-byte局部、400-byte配置先於count、零count略過free、位置表步幅2、14818／149F8完整清單、15880的完整count與strict-greater寫入53C33／37／3B／3F均核對原始指令。4E56C只改EAX／EDX，4DBFC保存EBX／ESI／EDI；保守pragma與volatile只作相容產碼表示。
- 兩組189份候選獨立編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對。六項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。舊85份來源報告順序保持，新正例最後加入，201段舊C出處逐筆保持。
- SDK完整11／19／25來源鏈在新C基準重跑兩輪，25函式／1777 bytes出處與原始來源報告保持；十項未知／C／裁切等拒收通過。首次掛載少一層clib3s.dos而找不到SDK，修正唯讀路徑後用相同命令乾淨重跑。C++初輪malloc隱式轉型E166無完整收據，明確cast後重跑；這些失敗不計產碼負例。
- 357074-byte全檔與兩輪收據相同，SHA-256等於固定輸入。累計201個原始函式以C匹配，其餘929個函式待還原或分類；其餘原版code未還原，Goal與#198保持開啟。正式Go／Ebiten與既有AI效果驗收保持。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_int31_regs_matches`、`game_ai_item_select_attempts`、`ai_item_select_regs_original_evidence`、`ai_item_select_cpp_supplement`、`validation.ai_item_select_regs` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_int31_regs` 或 `game_ai_item_select`，各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。SDK沿既有三層鏈重建；公開完整C、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對131份來源雜湊、1305條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 物品效果與命中完整C候選

- 新增[效果C](tools/fd2_matching_game_item_effect.c)、[命中C](tools/fd2_matching_game_command_hit.c)與[完整回歸](tools/test_fd2_matching_item_effect_hit.py)。20C6F完整1043-byte來源接續已匹配的15055，也保留1BBDC內的玩家caller。1C75E完整193-byte來源保留七個E8 caller；IDA匯出的code xref總數8另列，不把兩種口徑混用。既有物品與職業規則不重解。
- 效果來源保留108-byte局部，將偏移14的16位元欄位擴成32位元存於局部偏移104，將record+60的位元組暫存於局部偏移100再還原；完整目標清單與count、各分支消耗邊界及共用收尾保持。命中來源保留116-byte局部、112-byte表複製、record+32原生職業減一索引、有號16位元讀取、有號除10與餘100、原始命中閘門及1C81F呼叫。
- 12種效果表示與10種命中表示在Watcom 10.0a／9.5、三種CPU與三種成本策略，共396份候選完整編譯、連結兩輪。全部OMF、報告與完整產碼相同，全由objcopy核對，全部仍為負例。型別複製已重現原始REP MOVSD，但同長度193-byte候選仍有六個bytes的暫存器差異，沒有補opcode、遮罩、裁切或計入覆蓋。
- 13項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。1C9DD的兩條退出邊仍經1C916內的1C9C7／1C9D5接22BBE共享收尾；不把172-byte區間單獨當作可返回的完整C函式。保守pragma、volatile、聚合與指標表示只作產碼候選，不推定作者宣告或實際clobber。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準及SDK來源鏈，不把負例放入EXE，不重跑未受影響的SDK重建。固定全檔SHA-256保持，929個函式仍待還原或分類；Goal與#198保持開啟。
- 後續先核對「分開局部變數＋保守的迴圈被呼叫端宣告」及「型別複製＋4E516保守宣告」的交互作用；本批分別驗過各因素，尚未驗證這兩個組合。不重跑本批相同來源矩陣來尋找偶然成功。RND只核對原始產碼，沒有執行或宣稱亂數對拍。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `game_item_effect_attempts`、`game_command_hit_attempts`、`item_effect_hit_original_evidence`、`validation.item_effect_hit` 與 `bootstrap`。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_item_effect` 或 `game_command_hit`，版本用 `--compiler-version 10.0a` 或 `9.5`。各輪使用新輸出目錄，再以固定原檔及IDA匯出執行 `link`。公開完整C候選、工具與整理證據，原版、i64及SDK物件留本機。

提交前核對134份來源雜湊、1314條本地連結、197條教訓與82個守衛。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 ACTING與標題共享返回段C群組

- 前輪預定的兩個宣告組合已核對。20C6F新增分開局部與保守呼叫宣告、16位元欄位／32位元容器、直接真值等六種表示；1C75E新增型別複製與4E516保守宣告的組合。兩個歷史編譯器共126份新候選均負例，舊396份不重跑。1C75E的原始矩陣仍由78a05a7b提交中的來源與驅動器重生，不改寫舊收據。
- 新增[完整群組C](tools/fd2_matching_game_title_acting_group.c)與[回歸](tools/test_fd2_matching_title_shared_group.py)。1366A完整818 bytes與1F894完整1765 bytes共有92-byte局部框架；1FF74以E9跳13994的八位元組收尾。先核對固定原檔、IDA函式邊界、raw bytes與呼叫端，將契約登記為READY，再實作同一C編譯單元，不把1765-byte函式裁成獨立返回函式。
- 來源保留ACTING兩份32-byte局部陣列、原始欄位及更新邊界；標題保留60-byte事件表、五張147列圖、535至0的捲動、原始插播點、存檔校驗、三種選單數量、鍵盤回捲與四次閃爍。聚合、bitfield、volatile與指標重用只作候選，不推定作者宣告。
- 群組六種表示在兩個歷史編譯器共108份候選仍未匹配。編譯器自然產生跨函式共享收尾，既有COFF工具將完整函式分開，只提升跨區間REL32供ld計算，保留指令寬度。共享結構成立不等於完整位元組或入口一致，也不增加C覆蓋。
- 本批六組共234份候選完整編譯、連結兩輪，OMF、完整報告與產碼相同，全由objcopy核對；群組分別擷取兩個完整PE區段再串接。12項錯址、既有LE／常數位移與75427筆載入記錄／偽造拒收回歸通過。
- 既有201個原始函式／30171 bytes、另146-byte未歸屬區塊、202段C出處與SDK25函式／1777 bytes逐項保持。沒有新增匹配，沿用既有全檔基準與SDK來源鏈，負例不放回EXE。929個函式仍待還原或分類，Goal與#198保持開啟；既有語意與Go／Ebiten驗收不提升。

來源、旗標與原始證據見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的 `title_acting_group_spec`、`game_title_acting_group_attempts`、`title_acting_group_original_evidence`、`game_item_effect_mix_attempts`、`game_command_hit_mix_attempts` 與 `validation.title_shared_group`。群組編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的 `compile --source-key game_title_acting_group`；新組合分別使用 `game_item_effect --cases ITEM_EFFECT_12 ITEM_EFFECT_13 ITEM_EFFECT_14 ITEM_EFFECT_15 ITEM_EFFECT_16 ITEM_EFFECT_17` 與 `game_command_hit --cases COMMAND_HIT_10`。版本用 `--compiler-version 10.0a` 或 `9.5`，各輪使用新目錄後執行 `link`。原版、i64、SDK物件與原生產物留本機。

下一步只對1C75E測一次32位元actor的volatile讀取。原版以EAX讀actor、EBX計算80-byte記錄偏移，目前同長度候選的六個差異都在這段完整索引；若新表示沒有改善，就轉向尚未試過且有完整返回段的函式本體。這是尚未驗證的產碼假說，不是作者宣告。

提交前核對136份來源雜湊、1322條本地連結、197條教訓與82個守衛。舊396份矩陣的C及driver雜湊另以78a05a7b的實際Git內容核對。真正主機清單經官方pull／render／verify，16條仍開啟，8條需人工判定，可能已完成0條。原始IDA匯出與正式資料庫雜湊保持，變更檔與新產物UID／GID1000；歷史root-owned共2811項保持，沒有`.md`目錄。本批FD2容器全部退出。提交、推送及真正遠端HEAD核對結果回填既有#198。

## 2026-10-10 暫停匹配試驗，先提交現有C候選

使用者要求暫停後先commit及push。本批只核對並保存暫停前的來源與收據，沒有再編譯或新增試驗。

- 新增[完整C候選](tools/fd2_matching_game_record_resolution.c)，目標為1DB65的857 bytes。保留124-byte局部、record+5整位元組寫入、word+64、視口公式、13幀姿態、6＋6幀效果及153216-byte背景。完整原始指令與八個直接E8 caller由既有IDA9.4匯出保存，既有狀態／死亡演出語意不重開。
- 驅動器登記11種表示。目前最終來源只有RECORD_RESOLVE_8／9／10的Watcom 10.0a、-mf -3s單次收據，產碼863／845／852 bytes，首差偏移17／17／32，皆非完整匹配。較早局部試驗的來源已變動，不作當前來源的正式矩陣。其餘版本、CPU、成本策略與獨立第二輪仍未完成。
- COMMAND_HIT_11的32位元actor volatile讀取已在10.0a／9.5各測一次，產碼皆193 bytes，首差偏移31，沒有改善。收據保留當時driver雜湊，並驗證由本提交driver移除三條record-resolution註冊行可重建原雜湊；不把歷史driver寫成當前版本。
- 五份既有OMF、完整產碼與來源雜湊核對通過，objcopy獨立擷取五份完整區間。當前driver的錯址、常數位移、75427筆載入記錄及偽造拒收回歸通過。零筆路徑的EAX未在函式本體定義，35BEB caller仍傳遞EAX；不推定所有caller忽略回傳，C型別與pragma維持產碼假說。
- 本批沒有新增C匹配。201個原始函式／30171 bytes、另146-byte區塊、202段C出處、SDK25函式／1777 bytes及既有全檔基準保持。五份單次收據另列，不併入116組6405份候選／6375份獨立核對／30份布局拒收的雙輪統計，不把負例放回EXE。929個函式仍待還原或分類，Goal維持暫停，#198保持開啟，Go／Ebiten及PLAYER-E2不提升。

來源、旗標、物件與產碼雜湊見[主收據](docs/data/ida/fd2_matching_full_20261008.json)的record_resolution_candidate_spec、record_resolution_original_evidence、record_resolution_single_pass、command_hit_actor_single_pass及validation.record_resolution_commit。編譯入口為[驅動器](tools/fd2_matching_game_restore.py)的compile --source-key game_record_resolution；單次試驗須明示--cases RECORD_RESOLVE_8 RECORD_RESOLVE_9 RECORD_RESOLVE_10，再用固定原檔及IDA匯出執行link。此入口只供日後重生現有負例，本輪不接續試驗。原版、i64、SDK物件與編譯產物留本機。

提交前核對137份來源雜湊、1328條本地連結、197條教訓與82個守衛。真正主機Issue清單經官方pull／render／verify，16條開啟、8條需人工判定、可能已完成0條。固定原檔、正式IDA匯出與資料庫雜湊保持；變更檔UID／GID1000，歷史root-owned 2811項保持，沒有`.md`目錄。提交前確認本批FD2容器全部退出；推送後核對真正遠端HEAD，結果回填#198。

## 2026-10-15 重製完成與HD排程，匹配研究再次暫停

使用者先要求接續，再明確要求暫停matching decompilation，改查重製缺口、加速重製與HD，並重新調整worklist及GitHub Issues。依復古重製、玩法稽核及工作清單路由載入任務入口；HD架構取捨依grilling確認，沒有新增production規則或素材格式。

- FD2起點2975ecb6，追蹤檔原先乾淨，既有未追蹤work保留。dosgolem已合併；fetch後main與origin/main一致且工作樹乾淨，從81c85be建立fd2-followup-20261015。尚未用整合後程式重生章oracle，分支建立不算行為驗證。
- 暫停前，既有record-resolution來源在Watcom10.0a編譯99份候選；連結容器缺wdis而退出，屬環境失敗。compile-report位於work/matching-full-20261008/record-resolution-resume-20261015/10.0a-r1，SHA-256為5c72a5002fc467bc87d4d196fc77530d705a5915eaf39af4a9f2da74b896a0d0。沒有完整link報告，不列產碼負例、不加雙輪統計；C來源、driver與主收據未改。使用者暫停後未接續編譯，容器已退出。
- 真正主機gh auth成功。讀取遠端Issue後更新16個既有題目的排程、現況與依賴，新增#199～#204六個執行子項。總項#14／#4不與子項重複計算。#198、#52、#197保持開啟，標P3與狀態:暫停；後續平台及編輯器的前置排除暫停研究，保留原驗收內容。
- 遠端22項逐條核對：P0六項、P1八項、P2五項、P3三項，每項恰有一個P級標籤。官方build_snapshot與pull快照完全相同，父子連結及#142→#203依賴通過。工作清單verify沒有可能已完成項；沒有關閉任何未達驗收題目。
- 唯一分層現況58新增本日狀態；56／57同步研究與重製的邊界，61同步HD執行子項，91由官方快照重生。舊匹配研究與章收據保留。沒有重跑已passed章、提高章分級、調整像素預算或恢復第12章目標追蹤。
- 本機缺少fd2-go-test-local:latest及fd2-assets-local:20260829-sfx，受版控Dockerfile仍在；#203記為環境前置。沿用既有eob-remake-release:1.26.7-ebiten2.9.9-20261008-r2容器的Python／Pillow完成清冊機械核對，不用它替代FD2正式Go／GPU驗收，也未建立重複映像。
- 全域sync_native_turn_events.py --check通過，仍有27筆未完整轉寫。現代catalog的161筆結構與私人檔尺寸／SHA／透明度驗證通過；file/files指向的1153檔全部存在。數量與正式章台帳只記在58；原型素材存在不代表全玩家路徑或高解析呈現已完成。
- 工作清單既有19個測試與現代catalog既有1個私人素材測試通過。高解析HD第一階段範圍已提問，#204維持DRAFT；現有核准風格、靜態嘴型、圖塊與存檔語意未改。

本輪可重跑入口，以下Python及測試命令均在上述既有容器內、repo唯讀掛載時執行；render需另將文件輸出所在工作樹可寫掛載並指定UID/GID1000：

```sh
# 只有官方Issue控制面在真正主機執行。
tools/fd2_worklist_issues.py pull
# 以下在Docker內執行。
python3 tools/fd2_worklist.py render
python3 tools/fd2_worklist.py verify
python3 -m unittest discover -s tools -p 'test_fd2_worklist*.py'
python3 -m unittest discover -s tools -p test_modern_theme_catalog.py
python3 tools/sync_native_turn_events.py --check
python3 tools/validate_modern_theme_catalog.py --verify-private
python3 -c 'import json; d=json.load(open("docs/data/parity-campaign-progress.json")); print([(c["chapter"],c["status"]) for c in d["chapters"]])'
python3 -c 'import json,collections; d=json.load(open("remake/assets/themes/modern/catalog.json")); print(collections.Counter((a["role"],a["status"]) for a in d["assets"]))'
```

本輪只重排工作與修正現況文件，未提交或推送程式變更；GitHub Issues已直接更新。歷史root-owned基線2811項未增加，沒有`.md`目錄，變更檔及新研究輸出UID/GID1000。本批容器均退出，其他專案容器未動。

## 2026-10-15 恢復正式工具鏈，補齊IRQ7並回到第24章

使用者選擇先補完整IRQ7鏈，再繼續章對拍。matching decompilation維持暫停。
依復古重製與平台規格優先路由，使用公開DPMI／DOS4GW契約及dosgolem既有305，
先建立429 READY規格，再接真實IVT、DSP確認、PIC EOI、IRET及caller返回。
沒有逐週期ISR考古、輪詢替代或遊戲規則變更。

- 由既有Dockerfile重建fd2-go-test-local及fd2-assets-local，最小context只含五個受版控輸入。
  Go1.22.12與Pillow11.3.0沿用原工具鏈；原版、私人素材、IDA與work未進build context。
  私人現代素材18項通過、零略過；完整Go19套件2383項通過、52項既有選用略過。
- 合併main81c85be的首輪正式原版，在5780804步／LE0x373FE因IRQ7接線缺口拒收。
  actual IVT0F=67FF:06F3，其他兩種0F向量皆零。原始失敗保持。
- dosgolem共用main保持乾淨，專用worktree位於work/dosgolem-fd2-followup-20261015。
  實作e26fb41及驗證規格2c5c104已推送fd2-followup-20261015，遠端HEAD已核對。
  cpu386／machine／FD2 oracle三套件2021項通過；10項略過皆缺MOO2可選原版。
- 真實FD2 ISR每次76條指令。600萬步3次進場／返回；52516893步到標題等待時504次皆返回。
  固定第三方CONTINUE槽及既有鍵盤計畫兩輪皆594524785步，233份完整狀態、233張PNG與3筆trace相同。
  舊／新嚴格比較仍failed，共446筆差異；完整單位、鍵盤讀取與DOS呼叫相同，view只在aux_phase不同。
  211張PNG不同，舊seq0缺圖；不將全部影像差異歸為動畫，不補圖或增加忽略欄位。
- native准入保留舊／合併未修／IRQ7修正版的完整來源組合，修正版另綁派送與實模式橋接兩檔。
  真實Git bytes測試接受三組與當前來源，拒收六檔單獨修改及兩種新舊混搭。oracle相關108項Python回歸通過。
- 現代靜態宣告盤點保留來源SHA與JSON pointer，不複製台詞。九個非負speaker候選缺頭像，負speaker另列；
  尚未證明正式玩家可達，不直接開產品缺陷或生成素材。高解析範圍仍DRAFT，catalog保持runtime_candidate。
- 真正主機Issue更新#142及#201，依限定工具收據關閉#205與#203。官方pull拉回21項；暫停項保持開啟。
  #142接續第23章實際保存槽，SHA-256為22bd63070189e0e6703e14d4bd4db8dc1f66452a21128dd5154579171ecbea63，
  seed4及祖先AP200／DP0／DX60政策保持。原版第24章首輪已到T3、單次清敵、戰後整備及成功DOS保存。
  重播因漏帶既有舞台trace而拒收，屬本輪驗證參數問題；補回既有0x24D48／0x11EED等入口後乾淨重跑。

唯一主證據為[工具恢復收據](docs/data/fd2-remake-toolchain-recovery-20261015.json)，保存輸入、來源、映像及輸出雜湊。
原始原版收據留在work/remake-recovery-20261015；不提升整章PLAYER-E2或公開私人素材。

可重跑命令均在Docker內，repo與原版唯讀、明確輸出可寫、UID/GID1000；Go及素材測試使用恢復的專用映像，
dosgolem使用dosgolem-go-ebiten:1.24-r1。oracle入口與原版／state掛載依[正式驅動器](tools/dosgolem_oracle.sh)。

```sh
FD2_ASSET_PACK=/out/pack FD2_TEST_MODERN_PACK=/repo/remake/generated-assets/modern-theme-prototypes \
  with-xvfb go test ./cmd/fd2 -run 'Test(LoadModern|ModernTheme)' -count=1 -v
FD2_ASSET_PACK=/out/pack with-xvfb go test ./... -count=1 -json
go test ./internal/cpu386 ./internal/machine ./apps/fd2/cmd/oracle -count=1 -timeout 480s -json
python3 tools/test_oracle_native_source_profiles.py --dosgolem <專用worktree>
python3 -m unittest discover -s tools -p 'test_*oracle*.py'
python3 tools/audit_modern_theme_coverage.py --verify-private --output <現代宣告報告>
python3 tools/verify_oracle_prefix.py \
  --prefix work/remake-recovery-20261015/normal-original-irq7-r1 \
  --full work/remake-recovery-20261015/normal-original-irq7-r2 \
  --compare-frames --compare-trace --out <新版兩輪比較報告>
```

舊／新版比較另用原memory197-command-native-original-r1及完整fa25f6e／e26fb41提交對，
`--compare-frames --compare-trace`不變。第24章使用受版控ch24-sample-r3計畫、native近堆及完整舞台／AI trace，
實際初始SAV另拷貝至state覆蓋層；每次運行有界，原始遊戲檔不寫入。

第24章後續結果：補回舞台trace後，r2重播通過，但seq162 PNG被下一個單步控制覆寫。
行為、節點與完整SAV相同，畫面閘門依缺圖拒收；容器exit143與缺少計畫尾端mark也如實保留。
通用單槽pendingFrame問題登記#206、P1，沒有修正或關閉。r4計畫在T3 mark後正常空等待200萬步，
原版r3完整走到退出存檔列表後的整備mark、exit0，211份狀態與211張PNG齊全。
重製側11.10秒通過，16筆AI入口全數消費、零順序分岔；四gate通過、13張完整RGB皆零差異，
兩側完整SAV SHA-256皆a1e6cc85722fc820cf38f820f4454785cb2138450d9c6000226466a340288406。

本次只驗T1～T3、移動、敵方回合與戰後保存，未抽T4／7／10事件、普通攻擊及其他適用戰鬥路徑。
不因四gate或13張零差異直接提升整章111，#142與正式章台帳保持。生成比較收據內嵌於工具主收據的
`chapter24_followup.bounded_receipt`；r4計畫已掛00索引。#206與缺舞台trace的前輪失敗分開記錄。
現代主題未生成新素材、未改格式或正式繪製路徑，高解析範圍仍DRAFT。

本批最後核對：遠端22個未完成Issue，官方pull／render／verify通過，可能已完成0項。
工作清單19項與oracle相關108項Python測試通過；教訓198條、83個守衛通過。
變更文件2238個本地連結通過，20份來源／輸出SHA全部相符。全知識庫掃描另發現9個既有問題，
分別為歷史本機交付物缺失及舊文件中的括號被簡易解析器當連結；均不在本批變更文件，不宣稱全庫連結通過。
原版及私人素材未改，變更檔UID/GID1000，歷史root-owned仍2811項、`.md`目錄0。
dosgolem專用分支及共用main乾淨；FD2提交只加入整理證據、工具與文件，work及大型擷取不入Git。

## 2026-10-15 第24章流程驗收與事件54補齊

使用者允許章前有界 DP 校準，隨後指定作弊過關、參考攻略與流程一致，不要求每個 frame 相同。已同步111／114、56／57／58及 GitHub #14／#142／#33。匹配式反編譯保持暫停。

- 第24章 event54 原先只有登場增援。本輪重用已核對的 IDA 9.4 匯出，與固定 EXE 及 Capstone 5.0.3 交叉確認125 bytes，保留 sub_3535D、sub_35822共用尾端及全域表consumer。規格 READY 後才補四角鏡頭與等待，九項具型別操作同步四筆正式／canonical回合事件。入口為 [event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)。
- 本地青衫攻略第24章記載四角增援，回合為3／4／7／10；固定控制列與原版實測為2／4／7／10。保留差異，攻略只作玩家旁證。
- 新增同章 boost-only 建槽模式，拒收混用推章、升級、事件與金幣。第一輪沿既有 fresh 槽只加DP1000，seed4，25名冊僅100個DP資料bytes及2個checksum bytes不同；HP／MP／物品／金幣與其他槽相同。原最早ch02基底不存在，直接沿已驗fresh產物，沒有猜補來源。
- 流程驗收仍嚴格檢查行為、節點、交易與完整保存。畫面保留原始結果，advisory模式不參與完成判定。測試可省略圖片輸出，正式Game.Update／Draw完成閘門仍執行。31項Python工具測試與建槽、字串Go回歸通過。
- 原有第24章短鏈重新消費新劇本，16筆AI入口全消費且零順序分歧，13張完整RGB零差異，完整SAV相同；這份回歸只驗T1～T3。
- 官方全域事件同步檢查通過，剩23筆未轉寫，沒有將spawn-only提升為完整事件。
- 字串review與新清冊雜湊原已一致，本輪測試Go變更不改production inventory；summary與四語內容落後既有來源，沿官方遷移器補同步，保留原譯文與審查狀態。10筆新來源用明示候選，外語維持machine_draft。四語結構驗證及14項測試通過，其中1項依原環境條件略過。

環境修正：Python建槽測試在唯讀repo先建立cache失敗，改為先拒收錯誤基底檔名再碰cache，乾淨重跑通過。第一次DP位元組核對漏列既有末尾checksum，按fd2save契約補核對後通過，未放寬資料欄位範圍。全docs連結掃描列9個既有歷史／交付路徑缺件，本輪新增連結另行核對，不把它們寫成產品缺陷。

本輪第一輪已正常至T11，T2／4／7／10增援全觸發。重製端重播通過，280筆AI入口全消費且零順序分岔。清敵一次60筆後，原版正常END於9470673688steps異常跳至0x1666E8，next bytes疑似資料，真因未知。沒有戰後SAV，流程收據保留failed，章台帳不提升。已登記[P0 #207](https://github.com/wicanr2/fd2_re/issues/207)，不直接補opcode或再調DP掩蓋。第23章完整流程三閘門回歸通過。

本輪完整Go回歸19個含測試套件全部通過。原版追蹤實際確認event54入口與四個pan、返回均各四次，控制序號134／192／330／538；失敗流程收據已公開索引，章台帳仍20章passed。

提交前已核對本輪31個變更檔UID／GID1000、2259條本地連結與git diff --check。工作清單由真正主機GitHub拉取後render／verify，23項未完成、15項人工判定、可能已完成0項。199條教訓與83個守衛通過。本輪FD2容器均已退出，既有root-owned基線2811項保持，沒有`.md`目錄。FD2專用映像只保留既有可重現版本；未清理其他專案映像或原有IDA工具鏈。

## 2026-10-15 第24章清敵視野與戰後交接接續

接續b8e7ec37與#207。工作樹只有既有work產物，matching維持暫停。原版對拍／規格閘門／文件職責路由及復古重製、IDA技能已核對；dosgolem遠端main仍81c85be，前輪IRQ7分支尚未合併，沿乾淨2c5c104。主機gh auth成功，開工官方pull為23項。

先轉送既有oracle的trace-tail，原預算末端2000條唯讀EIP，並明示寫入runner。診斷重播到第5回合時，已由既有IDA9.4完整匯出找到更窄證據，因此主動停止並保留部分輸出，沒有當作新的完整原版收據。

原版sub_1DB65完整857 bytes與固定EXE／Capstone5.0.3逐指令及邊界一致。局部124 bytes中，30個畫面指標佔0x00..0x77，malloc指標位於0x78；0x1DC29無數量限制地寫入，0x1DEB6..1DEBD依原順序恢復暫存器並RET。失敗前checkpoint631有40個符合畫面與死亡條件的角色，第31..35項同時解釋四個損壞暫存器及0x1666E8返回位址。五項還原同一畫面基底0x14F018，原生heap事件亦有此153216-byte配置。原始writer／consumer已證實，崩潰關聯列強推論，未宣稱取得實際RET trace。

READY修正只改章測試輸入：第11回合後正常方向鍵移游標至(0,0)，再一次HP-only清敵及正常END。沿相同DP1000槽、seed4及T2／4／7／10流程，不改CPU、原版死亡規則或正式重製。原版r2正在執行，通過前章台帳維持todo。重製端新增對應mark，含最後一個方向鍵重播與原版游標／鏡頭四座標核對，不靠座標注入。證據、規格與命令沿[event54主契約](docs/data/ida/fd2_turn_event54_20261015.json)，[視野稽核工具](tools/fd2_clear_viewport_audit.py)可重算。破壞EBP的反例拒收。

來源准入正反例通過，兩個shell入口語法通過；TRACE_TAIL=2在oracle啟動前拒收，實際診斷runner記錄enabled=true。來源准入首輪未掛git worktree共同.git目錄而失敗，補唯讀掛載後用相同容器重跑通過，未改准入清冊。Capstone首次多傳python3造成entrypoint用法錯誤，改用原Dockerfile既有入口後乾淨重跑；均屬環境／命令問題。

原版r2於第6回合seq303被SIGTERM終止，exit143、oracle.log為空，沒有原版錯誤收據。前301份完整JSON bytes與c1相同，沒有忽略任何欄位。改由Docker detached容器自身的2040秒timeout管理生命週期，以相同來源／槽／計畫重跑r3；保留r2，分類環境中止，不再校準DP。第23章以最終重播程式回歸，行為、節點、交易與完整SAV通過。

等待原版期間先核對下一章：攻略第25章第6回合援軍對應全域event56，固定表入口sub_35487，IDA LE linear35487..354DD共86 bytes。原始呼叫依序為pan(6,40)、spawn group1、ACTING74、文字索引5的對白、跳至134E4重設pose。1366A的既有語意索引已證實為ACTING播放器，不能誤當單位聚焦或把74當角色槽。event55是另一條炎龍劍事件，不混入本回合援軍。此處只準備入口，依同步工具契約等第25章原版收據後才接完整正式劇本，剩餘未轉寫數量保持23。

原版r3已完成710份狀態、兩次成功DOS寫入、戰後四槽保存與退出列表，SAV SHA256 e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95。T11改鏡頭前622份完整JSON與c1逐byte相同，原版來源與輸入雜湊在執行期間保持。清敵時實際可見候選只有3個，仍只寫60個敵軍HP，沒有改byte5。

第一輪重製重播在戰後被舊[28,86]前沿白名單拒收，已另列#208。677與685兩個原版快照分別含24C51／24CB2 caller及有效76筆陣列，依既有#145契約限制先寫READY修訂，再僅增加實測76，其他未知count仍拒收。重播再次通過，280筆AI全消費且零順序分岔。行為34點、節點36點、交易與完整SAV均通過；畫面未擷取，frames=false且required=false，不冒稱逐像素一致。

[正式收據](docs/data/ui-traces/parity-ch24.json)與台帳已由官方工具更新，第24章passed，合計21／30，另9章todo。19套件Go回歸通過；其後76槽binding資料修訂再通過正式章與相關Context／Frontier／Binding回歸。123項Python工具測試通過，27項既有Go／字串回歸及第23章完整流程／SAV回歸通過。匹配式反編譯保持暫停。

本批已關閉#142／#207／#208，#199成為下一個P0；正式保存名冊25筆、HP>0為11筆，下一章先驗正常選人並保留原來源。工作清單已由真正主機拉取／render／verify，21項未完成；最後核對18檔UID／GID1000、2262條本地連結零缺件、200條教訓與84個守衛通過。既有root-owned基線2811項保持，`.md`目錄0，本輪FD2容器全部退出。提交及真正遠端HEAD將回填已關閉Issue。

## 2026-10-15 第25章主線流程接續

目前狀態：第24章已passed，台帳21／30；第25章尚未驗收。沿f0c04db3及乾淨dosgolem2c5c104，主機Issue #199為P0。依原版對拍、規格閘門及文件職責路由載入技能與任務入口。使用者詢問必要時啟動matching，現階段先以原版收據定位差異；未啟動編譯試驗。

新輸入是第24章實際保存e67cf23b0c5f72e40742b69544cee572cfb155e01a75d7eff65bf3fbdc20ee95，複製到獨立state。名冊25筆，HP>0為11筆，零額外治療、升級或強化。攻略第25章第6回合己方結束援軍作旁證，依正常鍵盤推至T7後一次明示清敵；炎龍劍支線未列入此主線計畫。畫面advisory，流程三閘門與保存仍必要。

入口：[接續槽清冊](docs/data/parity-slots/ch25-manifest.json)、[主線計畫r1](docs/data/parity-plans/ch25-flow-cheat-r1.jsonl)。原版使用唯一tools/dosgolem_oracle_container.sh，來源與原版唯讀，輸出位於work/ch25-flow-20261015；不覆寫前章。

首輪原版至T5必存角色聖寇拉斯index16／identity26，HP104→0後返回標題，未觸發event56、未保存。保留555份狀態與失敗流程收據，不把正常敗北當執行器缺口。r2沿同SAV／seed，以每回合正常move_unit向(0,43)退避；T1退避前104份完整JSON bytes與r1相同。沒有追加DP或注入HP。

另以原版battle_start seq103建立唯讀前綴供重播診斷。重製原在story_ch25的339F5呼叫24B4D時要求g.st，無法消費LOADCH的故事載體，已開#209。沿既有3396A完整IDA匯出與24B4D／story adapter證據先寫READY，再沿materializeNativeStoryMapState／composeNativeStoryMapFrame組裝場景，輸入組裝完成即恢復g.st=nil。正常四次揭幕參數為20／20／20／60，未略過演出。相同前綴重播通過；18項相關Go回歸及場景成功／缺資產保持非戰鬥狀態通過。字串清冊SHA bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3與review相同，不遷移四語。

第25章event56的完整86-byte靜態轉寫已READY並加入producer，尚未降階到正式劇本。既有35487在9份歷史編譯候選中5份完整匹配，C來源SHA相符；無新matching編譯。依規格等本章原版收據後才接正式consumer。唯一主契約為docs/data/ida/fd2_turn_event56_20261015.json，已掛00索引與58。

第二輪仍於T5因聖寇拉斯死亡返標題，沒有觸發event56。退避不是足以通關的結論，不再單靠調走位重跑。原始LOADCH紀錄證實其自帶兩瓶神聖之水raw槽2／3，沿既有type5、amount999、玩家目標與回復交易，先寫READY後補控制器及重播。第5／6回合正常自用，沒有改初始HP、增道具或鎖血。計畫完成閘門要求兩次item_use；125項Python回歸通過。

第三輪在T5實際完成正常物品使用：seq377→378，HP57→600、byte5=128、第二瓶補到槽2。原版無錯誤，控制器誤以為被用掉的原槽一定空掉而拒收。回查既有已證實sub_1B8E7與原版八格bytes，修正為完整左移補位及尾flag0x80、尾item byte保留；第二次仍選當時槽2。保留r3及其控制器SHA，r4沿相同容器／槽／seed乾淨重跑，不計成DP校準或遊戲缺陷。修正後126項Python測試通過，含取自r3控制鏈／八格內容的正常消耗正例。

進度訊息勘誤：先前所稱退避後第4回合仍600 HP，應為第2回合當下值；第4回合起手原版為199 HP，第5回合起手57 HP。以checkpoint及主契約為準，不用舊訊息推定存活。
原版r4以正常鍵盤完成T6 event56、T7、戰後及酒店保存／離開，共601份checkpoint。T5／6正常使用原有神聖之水，HP57→600及6→600；T7正常移鏡頭後一次HP-only清敵35筆。沒有追加道具、強化或鎖HP。event56固定入口與完整五操作實測後，正式producer、scenario及canonical同步。

前綴揭幕修正後，另查到battle_ch25缺原生view及HUD繼承，開#210並依READY補資料與generator。正式AI首個敵軍施法者record21延後，造成12筆順序漂移；診斷找到ScoreNativeAI1598A的三個caller未傳既有1F183閘門，開#211後先寫READY再接線。相同原版r4完整Game r2消費240筆AI、零順序分歧，正常物品、T6演出及戰後隊伍／酒店都到達。這些缺口皆由既有證據定位，本輪沒有新matching編譯。使用者另允許必要時局部匹配，118與#198已明示界線，全面仍暫停。

完整Go初次只敗在舊測試仍期待event56單一spawn；改成核對完整七個lower動作並保持戰後名冊隔離範圍後，19套件乾淨重跑通過。132項Python回歸通過；第24章以最終Game重新重播，280筆AI零順序分岔，行為／節點／交易及整檔SAV相同。全域sync check通過，剩餘未轉寫22筆；字串inventory SHA仍為bba089ef90272e287c20260b037d3a26805aae7effc1a71fe4ee2ef1d6dd4ac3，未遷移四語。

第25章目前未通過。原版SAV SHA0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb，重製52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。全部明文只有identity26的+0x28..+0x30區間七byte與衍生檢查碼兩byte不同。既有IDA建構器未寫此區，11506整筆複製，本次原版runtime一路保持同值；用途及最初producer仍未知，不能宣稱指標或無用欄位。新#212追蹤，正式失敗收據docs/data/ui-traces/parity-ch25.json、主契約opaque_save_gap保存完整證據。工具先達READY再加--oracle-save，核對實檔SHA與checksum後揭露所有差異，未排除任何byte或放寬交易閘門。台帳第25章blocked，passed仍21／30。

已提出具體九位元組區間驗收例外供使用者決定；尚未收到答覆，完整保存門檻保持，不進下一章、不把oracle常數填進正式runtime。前三項consumer缺陷的驗收範圍與未知存檔題拆開，章驗收由#199／#212繼續追蹤。

提交前官方主機pull／render／verify：22條未完成，可能已完成0條；三項接線缺陷#209／#210／#211已結案，#212與父#199保持。台帳21 passed／1 blocked／8 todo一致，202條教訓及86個守衛通過。40個變更／新檔UID/GID1000；本輪文件2289條本地連結零缺件。全域另有9項缺件／解析誤判，均位於未改動歷史文件，不算本輪新增或產品缺陷。原始輸入及r4控制器／計畫雜湊保持；既有root-owned基線2811項、md目錄0、掃描錯誤0。本輪FD2容器全部退出。提交及真正遠端HEAD核對後回填已結案Issue。

## 2026-10-16 第25章有界保存驗收定案

使用者確認接受逐筆揭露第25章保存差異並繼續。命中原版決定性對拍、規格閘門及文件職責路由，載入既有入口、重製技能與必要reference；開工工作樹只有既有work未追蹤資料，沒有FD2容器。真正主機gh auth成功，官方pull為22項未完成。

先逐byte保留原parity-ch25.json為parity-ch25-flow-r2-save-failed.json，再更新111及event56主契約save_acceptance_policy為READY。正式比較器新增明示--save-policy ch25-opaque-20261016，只接受第25章同流程profile、輸入SAV、計畫、原版EXE／dosgolem、兩側實檔SHA及完整差異清單；七byte與衍生檢查碼兩byte全部保留，bytes_equal=false。未知用途不升格、不回填oracle常數，不改正式遊戲與存檔格式，其他章及預設strict保持。

相同原版r4唯讀，正式Game乾淨重播remake-approved-r3至酒店，240筆AI全消費、零順序分歧，SAV SHA仍為52984034e916259e2c4a45eafd0bb36f1835edabfd8867e007dee9ad38c1996b。使用預設strict同pair仍failed；明示有界政策後行為／節點／交易通過。137項Python工具測試通過，涵蓋不同章／profile／來源／實檔SHA、新增／刪除／改值差異、缺保存writer等拒收；第24章流程與整檔SAV回歸仍passed。此輪沒有Go正式程式修改，未重跑原版或啟動matching編譯。

政策達CONFORMED，第25章按111例外登記PLAYER-E2，官方台帳set／verify更新為22／30。第25章slot_policy明示直接接續實際SAV、零追加升級／強化／event-state改寫，祖先政策獨立保留。58、56、57與索引同步，歷史failed及其SHA保持；#212以使用者接受有界證據限制結案，不宣稱未知語意已解或保存bytes已修成相同。下一個P0為#199第26章。

最終相關回歸141項通過，包含原137項工具與4項台帳測試。真實第25章缺少實檔報告反例及第24章錯套政策反例都交易failed，明示政策沒有一般化為遮罩。原failed收據與前一提交逐byte相同，正式收據來源SHA與原版EXE保持。官方主機pull／render／verify為21條未完成、可能已完成0條；台帳22 passed／8 todo／0 blocked，203條教訓及87個守衛通過。本輪16個變更／新檔UID/GID1000，2294條本地連結零缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0，FD2容器全部退出。提交及真正遠端HEAD核對後回填#212及父#199。

## 2026-10-16 第26章原版主線接續

開工1f8fa562，工作樹只有既有work未追蹤資料，dosgolem專用工作樹2c5c104乾淨，沒有舊FD2容器。命中原版決定性對拍、規格閘門與文件職責，載入既有入口及重製技能。主機gh auth成功，官方pull為21項未完成，#199接續第26章。

沿第25章原版實際SAV 0d420b379a38906b208a5e2a88dd74617b1c9a763010205b5c864f0ff102a9cb複製獨立覆蓋層，名冊27筆，不追加治療／升級／強化／道具。悠妮identity9現HP0，先由正常LOAD／選人核對本章必存角色條件；不能章內鎖HP。第25章保存例外不套用本章。

攻略未知的迴廊與原有控制列表對照：event57在敵方T2／4／6／8／10／12／15／16／17，group由當前53BEF round取得。沿既有IDA9.4完整匯出保存sub_354DD 33bytes及353C4..353DA共用尾段；完整操作為spawn當前round group、pan(9,0)、delay400。現行九列只有spawn，先建立READY契約，等原版收據才接正式完整lower，不重開matching編譯。

95個正常鍵盤控制命令，預估19～29分鐘、2核；原版r1由唯一tools/dosgolem_oracle_container.sh執行，原始素材及dosgolem唯讀，獨立state-r1／original-r1，native近堆、40bn步預算、外層2040秒timeout、記錄AI及event57入口。預定T18後正常移鏡頭再一次HP-only清敵、戰後及酒店保存；台帳仍todo。入口docs/data/ida/fd2_turn_event57_20261016.json、docs/data/parity-slots/ch26-manifest.json及docs/data/parity-plans/ch26-flow-cheat-r1.jsonl，均已掛00索引。

第26章r1於cp78取消：原版LOAD回城鎮，獨立整備前綴誤入教會狀態窗。此為控制計畫問題，未進戰鬥，不列產品缺陷。接續原版槽悠妮HP0，復活費4000高於金幣2000。r2沿已授權建構政策，使用第24章slot-c1完整隊伍基底，以`fd2-chapter-slot -target 25 -levels-per-chapter 6 -seed 4 -boost-base-dp 1000`建槽；累計政策及所有27筆狀態見清冊。正常城鎮選人先選14筆，再以down／right選末位亞奇梅吉。輸入與規格來源已登記索引，保存仍嚴格比較。

第26章r2在戰前探訪取消：`town_probe`固定settle後於轉場狀態反覆送esc，建築選擇偏離，未進戰鬥。r3保留同一建構槽與正常選人輸入，移除戰前探訪，先驗九波與酒店保存；五處探訪與買賣未抽樣，不冒稱完成。預估104個邏輯序列，2核上限，單執行緒原版實測約每秒160萬指令，外層90分鐘有界。

第26章原版r3正常至T6。診斷前綴到T5的Game可執行且176筆AI零順序分歧，但比較器揭露站位／HP不同，不能稱通過。#213定位為缺少raw25的required[9,29]及依序front[9,29]資料；沿既有318AD原始caller與31DBE／321C8閉合helper，READY契約先記於第26章主證據preparation_contract，不重新深挖helper。

原版r3在T13正常敵方範圍法術後悠妮HP272→0，回標題；六波已觀測，零清敵，後三波及戰後／保存未驗，不將敗北當完成。依使用者已明示作弊通關方向，本章採章前一次HP／MaxHP+30000，沿既有signed word欄位，不章內鎖HP、不改正式規則。預檢原子性、保持傷勢與HP0、未指定時完全不改的READY契約先記主證據hp_slot_policy，準備受版控建槽旗標。

章前HP政策工具Go回歸通過，slot-r4由同章slot-r2以`-boost-only -target 25 -boost-max-hp 30000 -seed 4`建構，27筆，零handler／升級，SHA f04c6761f4dd4aa0a1047dc5d18fbfdbcfdfac94cccf0fd62169ca9895a645ff。索爾30420、悠妮30272、亞奇梅吉30416 HP；所有數值低於signed word上限。r4控制命令與r3相同，只有來源槽與政策標示不同，原版source仍乾淨2c5c104，90分鐘有界。

r3敗北診斷Game消費601筆AI零分歧，一般比較behavior=true，但實際end為cursor／T14，不能證明敗北終態。最初誤設只適用第15章的DEFEAT_TAIL測試旗標，已按同容器命令移除並乾淨重跑，失敗log保留，不列產品缺陷。新#214實際缺陷為第26章缺20B3C結果列及原生敗北返回設定。固定EXE54bytes、章25結果表51B7D及既有helper契約已核對，READY先存defeat_contract；只補資料，不重解一般敗北呈現或新增matching。

#214補兩列原生結果與敗北返回資料，移除未證retreat_ch26路由及孤兒節點，保留既有flag。同r3正常Game消費601筆AI，零順序分歧；record2悠妮由272承302傷害至0、raw+5 bit0成立，native_round13觸發record2_inactive，完整原生敗北／publisher／開場至title menu selection0，保存未變。有限工具只新增章26的已觀測終態准入，其他章仍拒收。原版9／36tick影像未逐像素比，不提升完整章；r4HP流程繼續。

第26章正式r4三項流程閘門通過，913筆AI全消費、零順序分歧；SAV 22987 bytes整檔相同，SHA 89a8d7a93f8baae11e2315f9d06c9a979ef5d6d608ee1558ec42ffb198cdbbd8。首次全套Go有三項失敗：覆蓋率舊計數、文字審查舊SHA，以及我加入全域FD2_MUTE造成command17缺sample反例失效。來源稽核先重生為117故事／過場、57handler、51fallback及26retreat，再同步斷言。文字清冊重綁98筆處置不變，四語5094筆全以既有id配對、new=0；canonical-root最初誤指父目錄，按工具規格改story後乾淨重跑。三個針對性Go測試通過，完整同映像回歸移除全域靜音重跑。

#216渲染收尾勘誤：原README仍有舊動態標記，首次渲染改寫了區塊，保留SHA斷言拒收該結果。現以穩定現況連結取代動態表，README區塊外所有文字對HEAD逐字相同，文化保存、圖片與技術導覽未變。進度頁的動態表改由工具產生，前言不再無條件稱四gate通過；CLI重跑與--check保持README SHA。

最終19套件Go回歸通過；第25章用目前Game乾淨重播，240筆AI零分歧，既有有界保存政策仍passed。另審查canonical時發現移除一節點使預設初次匯入規則重編121個ID；#217新增選用既有canonical來源，依legacy_key保留其餘318個ID與引用，不改schema或預設初次匯入。刪除／插入、錯文件／重複ID／錯kind回歸及官方CLI精確重生通過。9項相關Python工具及三相關Go套件通過；最終第26章再重播913筆AI零分歧，整份收據bytes與正式收據相同。README另修正驗收政策段落，不再無條件稱四項全過，其餘文化內容保持。下一章27先核對LOAD owner與已閉合318AD的raw26必出／前置資料，不沿用第26章選人鍵或重解helper。

最後渲染審查補正：非必驗畫面明示非必驗，未比像素不顯示0px，第25章保存明示差異已揭露並接受；10項渲染工具測試通過，相關Python合計85項。官方主機pull／render／verify為21條未完成、可能已完成0條，#213～#217已結案，#199接續第27章。台帳23 passed／7 todo／0 blocked，205條教訓及88個守衛通過。本輪49個變更／新檔UID/GID1000，2750條本地連結零新增缺件；全域既有9項缺件／解析誤判均在未改動歷史文件。root-owned既有基線2811項、md目錄0、掃描錯誤0。README僅更新進度入口與驗收政策，其餘文化／圖片／導覽逐byte保持；原EXE及最終正式收據雜湊保持。最終台帳verify初次誤掛唯讀而拒絕寫回，改為明示台帳可寫後同命令乾淨重跑一致，非產品缺陷。本輪FD2容器全部退出，其他專案容器不動。提交並核對真正遠端HEAD後回填各結案Issue。

## 2026-10-16 第27章主線接續

開工bf4bdab8，追蹤檔乾淨，只有既有work資料，FD2專用dosgolem2c5c104乾淨。路由命中原版對拍／規格閘門／文件職責，主機gh auth成功、官方pull21項未完成。攻略第27章及新浪旁證確認階梯增援、隊長死亡增援與天空之鑰後續，正式 ABI 沿既有IDA證據。直接複製第26章原版實際SAV，零追加升級／強化／事件改寫；祖先HP+30000及AP／DP／DX仍明示。正常LOAD與選人、索爾向y39移動觸發62→63，T9後一次清敵再驗64及戰後記錄，原始素材唯讀。必出缺陷只補raw26既有318AD caller，required9／front9的READY先於資料實作，不重解helper或啟動matching編譯。

#218必出／前置資料已依READY補入生成器及正式／canonical，318個ID保持。缺9拒收、單9通過與完整名冊前置測試通過；66筆IDA指令對固定原始file bytes相同，Capstone5.0.3對齊。原版r1漏設AI／成長trace，於開戰前取消，不作正式收據；r2同輸入同槽，補0x13A9F／1E54A後乾淨重跑。

r1取消邊界補正：最後cp74已載入戰場並處揭幕對白，尚無敵方AI；先前『開戰前』應讀為首次敵方行動前，診斷原始資料不改。r2補齊唯讀決策追蹤並保持同輸入，50分鐘、30bn指令上限。

第27章原版r2完整至T9及非城鎮保存返回選人，424筆AI入口、event62入口13次與event63一次；一次HP-only清敵59筆。
勘誤：最初計畫預期清敵觸發64，實際沒有358EA；64與隊長死亡增援仍未驗。原版SAV22987 bytes，SHA003d0394e1c5d9bb609949830450b4658284501f2d1e94456cbf96ed69b11663。

揭幕舊binding混入死亡對白且漏pan／ACT81／三轉場，依原caller先補READY再接原生19句；來源實際80筆、初始runtime63，撤回loader前錯置context。
故事inventory predicate接回已物化raw roster首16筆，不借持續隊伍。#220接250CC共同前綴與持鑰匙成功臂23句、ACT82..84、完整typed色盤／轉場及清理。
兩層direct patch25116白名單、透明inventory gate視圖交接及indexed palette consumer依READY修正。
先前『鏡頭歸零』只是症狀：真正根因為節點交接捕獲過時battle carrier，未保持當前story view；無原版歸零writer。

Game r7走完整戰後且SAV相同，但正式比較揭露90筆AI順序分歧，沒有因保存相同就標完成。
#222既有IDA直接反證1567E高command第一階段應用mode1／inner1，149F8 selector0應收raw+6非零；舊consumer誤收同陣營造成互傷。
READY先於實作，11／56及舊map0 score8靜態斷言同步勘誤；六項scorer回歸通過。
同原版r2重製r8全424筆零分歧，正式行為／節點／交易通過，完整SAV嚴格相同；frames0張且advisory，白／黑VGA memset尚未轉寫如實列限制。
415筆IDA9.4來源指令與固定EXE原bytes、Capstone5.0.3邊界一致；沒有新matching編譯或硬體RE。

故事稽核重生118節點／9直接腳本／59handler／50fallback，26舊撤退；既有318個canonical ID保持，只新增共同前綴節點。
字串重綁98筆處置保持，inventory SHA39cd7d84193c274946b5f30399f8a66a34e0b6151dd9d044a9b5faaf2f2938cd；四語各5094筆，4860依ID、234依signature，new0。
完整Go與前章正常Game回歸收尾中，台帳及GitHub最後核對後提交。下一步第28章，先核對非城鎮LOAD owner與已閉合必出／前置資料。

最終19套件Go全部通過，cmd/fd2 283.154秒；81項Python回歸零失敗／略過。第26章913筆AI零分歧、三流程閘門與整檔SAV保持。最後indexed色盤發布清除legacy白屏旗標的三項測試通過，第27章r9重播424筆零分歧，正式收據與r8逐byte相同。最後Go修改的文字清冊SHA保持，四語文字／role／variables全與HEAD多重集合相同。官方canonical importer逐byte重生，318個既有ID保留，只新增1個節點。

真正主機gh已更新#199至第28章，#218～#222結案，#33保持13筆未轉寫與未抽樣64限制；官方pull／render／verify為21條未完成、可能已完成0。206條教訓及89守衛通過。README SHA保持6fd6f74bc84a17d4f05bd31cd624b7d91306a3a756bd62b2405516dcbc38dd70。首次衛生檢查發現2812項root-owned，定位為本輪oracle掛載時建立的空dos-cache/gomodcache；只修該空目錄至1000:1000，未遞迴變更。後續所有mount來源先確認存在。歷史map0 item79分數與phase96／8已在11／56及時間序列文件追加勘誤。

最終衛生固定工作樹後乾淨通過：51個變更／新檔UID/GID1000，本輪3282條本地連結零新增缺件；全域9個既有缺件／解析誤判保持，其中交接檔兩項舊封包路徑逐字存在HEAD未變前綴。root-owned回到既有2811項，md目錄0、掃描錯誤0。前一掃描與git暫存並行且錯誤明細未記錄，固定後相同掃描零錯誤，不列產品缺陷。FD2本輪容器全部退出，其他專案容器不動。暫存51檔無原始EXE／SAV／資料庫或私人素材，正式收據strict保持。提交推送並核對真正遠端HEAD後回填#218～#222及父#199。

## 2026-10-16 第27章隊長增援再次抽樣

使用者補充鎖定索爾與敵方隊長血量直到增援抵達，並要求重製也有增援。先回第27章，暫不開第28章。初始工作樹只含work，主機gh登入成功、官方pull21項。路由載入原版對拍、規格閘門及文件職責；既有358EA直接指令顯示第二名死亡生三組、第三名死亡清場。採有界E1修改路徑，既有正式收據不替換。新dosgolem分支從目前main81c85be建立，補回先前未合併的IRQ7兩提交，核心來源tuple與已驗基線一致。430及主契約resample_hp_lock_policy先達READY，再實作選擇性控制及雙側重播。

第27章增援r1已驗前波12筆與保護清敵56筆，但朝15,22的曼哈頓候選停於15,32。成本格確認y31中央三格不可通行，r2明示經x13繞行六回合，候選先以同成本格核對。r1控制465另因牆鐘期限未收到checkpoint而結束，最後trace仍完成三隊長AI入口，未判為引擎缺陷；驅動逾時改用單調時鐘並加回歸。

增援診斷追加：重製r1-fixed已實際追加前波12筆，262筆AI零順序分歧。首輪測試鎖血漏接持續角色NativeIdentity的既有+8相容契約，已按NativeItemPanelRecordForUnit修正，沒有改正式Game。原版r2於雙隊長攻擊前主動停止，因已證實29F72負差額普通傷害為0；依使用者既有作弊授權，用既有fd2-chapter-slot boost-only生成同章全名冊AP+1000槽，SHA256 0eb3de0a84fd2907f28d5112380511a2e05d13423b0b04a66db5b96c9b9836ea。解碼差異僅27筆baseAP／effectiveAP兩word與兩byte校驗碼，共110byte；r3兩側使用同槽，原正式SAV及章台帳維持。

第27章增援r3／Game r3完成：正常兩次物理死亡派送358EA，三路21筆確實追加，新增位置／HP／raw+8一致；270筆AI零分歧，索爾30420及第三隊長3200存活。獨立E1收據passed；正式24／30及parity-ch27維持。第一次驗證腳本誤對不同AP槽要求BOOT暫存器相同，已分離同輸入95邊界嚴格回歸與AP槽EBX完整差異；未知EBX語意未猜補。終點view差異另列#224。最終Go19套件282.332s、102控制器加4章槽Python、來源准入及90教訓guard通過。

## 2026-10-16 驗證錄影清理

依使用者要求節省空間，核對既有收據與實檔雜湊後刪除下列五段本機暫存錄影，共10548497 bytes。保留收據、日誌、必要截圖與受版控重跑腳本。歷史收據中的影片路徑代表當時產物，不代表目前仍保存。原版第一章來源錄影與成品推廣片保留；後續章對拍沿流程模式，不產生非必要錄影。

- `dist/private-promo/fd2-v011-palace-remake-complete-raw.mp4`：3090291 bytes，SHA-256 `eccadf9fe99b76c253b907de6b449b141729c79058a7aa7472ddbcf3bf1960d2`。
- `dist/promo/fd2-ch01-march-remake.mp4`：3581208 bytes，SHA-256 `efe8a83313771e5be2e5e0fade67c4b7b534364f3395a548cd1bd2a8b5941bcf`。
- `extracted/title_re/dragonfx_compare/remake_intro_8s.mp4`：1642688 bytes，SHA-256 `b12937910d13a6ca0a8e1baa50bdeeaba358fba3ed33c6fa0e6580fb32d0eba3`。
- `dist/promo/fd2-v0.1.0-live-gameplay.mp4`：1566124 bytes，SHA-256 `c3fb3c0c07ebb8f05613bae712eefe9a37227452b4da2f0a62d674430b832d10`。
- `dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4`：668186 bytes，SHA-256 `a5e9b98b167d135a7167def90a07aa1450a700db18e158f7180f65f432f96e3c`。

## 2026-10-16 第28章主線接續

開工a3c8669d，保留上一輪錄影清理兩檔；真正主機gh登入與官方pull22項成功。路由命中原版對拍、規格閘門與文件職責。直接複製第27章正式實際SAV，SHA003d0394不變，原來源唯讀；非城鎮LOAD與19人選擇單獨短探針。舊第28章E1聲稱64-slot包含未啟動增援，沿既有IDA及休眠控制列核對，不重解共用helper。

第28章原版正常LOAD短探針完成，34-slot與19人選擇、非城鎮入口已確認。既有222筆IDA指令對固定file bytes及Capstone邊界相同，READY先於補接；新增66／68／70完整轉寫與65／69原子區域啟動，必出前置沿既有318AD。Game同源短前綴的行為／節點／取樣交易通過，0 AI且未驗戰鬥／保存，不登記整章。首次exporter把RET納入對白區間，被完整指令核對拒收；修正半開區間後全58事件及全域回合同步通過，13筆固定回合待轉寫保持。完整原版r1已啟動，T9後才單次清敵；未錄影。

完整r1於T5被實際敵軍與地圖邊界堵住，exit8／cp312，未清敵。校準槽依既有boost-only一次AP+1000，54個word政策值與原檔110byte差異核對，其他byte保持；首次檢查誤將全檔checksum算進第4槽meta，依fd2save.CHECKSUM_OFFSET更正後通過，非產品缺陷。r2先要求完整64筆再清敵。Game r1另在第一移動揭露缺HUD入口，原版cp71與25E5E共同caller先達READY再補typed資料。聚焦晚期測試的原版locator／語系／首張穩定地圖缺件分開修正fixture，沒有放寬正式Game閘門。

#226定位增援candidate回滾PAN六全域，限定66／68／70保留當前view與135F2的selector0；已閉合135DD／35822另68筆原指令獨立核對。修正後Game完整重播r1至原版堵路邊界，74筆AI全消費且零順序分歧。晚期來源測試fixture改從既有JOIN constructor物化持續名冊，語系與穩定地圖用正式helper補齊；環境／fixture失敗不列產品缺陷。r2原版已經正常觸發67死亡及68第二波，等待第三波與戰後保存。

原版r2及Game完整流程完成：初始34筆，66／68／70三波到40／46／64，第三波T9；T13後seq1035一次清敵34個存活敵人，seq1064成功兩次DOS寫入，seq1073返回選人。305筆AI全消費且零順序分歧；正式比較行為／節點／交易通過，整份22987 bytes SAV含checksum相同，SHA df50919111cccb37a00215dea248a069617aa222545d0f4493ab4b00dc5b819f。未錄影，frames advisory不宣稱逐像素。台帳登記25／30，下一章29，不在本章工作單位開新章。字串重新綁定98筆處置不變；完整Go回歸執行中。

2026-10-17收尾：原版與Game完整收據保持passed，13個正常回合／齊備mark視圖四座標相同。首次完整Go額外提供FDOTHER／DATO，揭露六項先前受來源環境閘門略過的舊fixture缺語系；補正式locale，另補第22章持續名冊後的正式部署、第29章冷讀選人素材與非同步END回復畫面。修正僅測試，不改正式規則，也不提升第29章章驗收。137項Python在專用Capstone映像及明示FD2_EXE後全通過；先前兩次容器分別缺Capstone及原版路徑，失敗日誌保留且分類為環境。全域死亡58事件、回合／死亡程式／33地圖同步check通過，固定回合待轉寫13保持。新增PAN candidate教訓後208條、91 guard通過。

來源fixture r7六項全通過，另外保留原生EXP整數byte限制：舊77.5改為77，不放寬正式sync。完整Go以相同FDOTHER／DATO環境乾淨r2重跑，原始org_game另加唯讀掛載。GitHub #225／#226依完整章收據結案，#199排至第29章；官方pull22項，最終render／verify收尾中。最終清冊與runtime修訂後清冊逐byte相同，review SHA3c58908df73ae0a80f3443892e57d6b5e80f2a0f92177df356c83edc461ebc80，98筆處置保持。固定工作樹47檔擁有權1000:1000，新增失效連結0，歷史封包兩項缺件保持；全樹root-owned2811、md目錄0、掃描錯誤0，README與原EXE雜湊不變。

最終完整Go r2於相同來源環境乾淨通過19套件，cmd/fd2 277.634秒；137項Python無失敗／略過，六項舊source fixture乾淨通過。官方worklist verify為22條、14項人工判斷、可能已完成0。所有FD2本輪容器退出，README與原EXE保持；暫存47檔無SAV／EXE／私人素材。提交推送後以真正遠端HEAD核對，並回填#225／#226、#199及#33。

## 2026-10-17 第29章護送與防衛系統流程

開工633d2e43，tracked工作樹乾淨，主機gh登入與官方pull22項成功，FD2容器無殘留。路由命中復古重製、原版對拍與文件職責，沿既有work研究區。原始來源為第28章正式實際SAV df509191，直接複製至新input及state覆蓋層，不追加強化。攻略第29章說明護送悠妮到石碑、四回合增援及三頭龍；ABI與規則以既有map28 IDA主證據為準。先做正常非城鎮LOAD／19人選擇短探針，再安排實際地形路線；章台帳仍25／30，沒有新matching或錄影。

第29章完整r1在round2／checkpoint192主動停止，未清敵。route_cost把敵軍當全程障礙，遠方封鎖列讓Yunie全無候選；這是驗證驅動錯誤，不是原版規則或Game缺陷。修正READY：未來成本按地形，當回合可達集合仍保留占用及ZOC，再由原版Enter裁決。r1計畫與收據保留。

勘誤：r1問題在checkpoint192辨認；實際停止命令完成於checkpoint238，仍round2、未清敵。r2保留同一AP5000槽，地形潛勢改正、正常接戰助手0與18、最多30回合；r1未計整章。
戰後局部回歸先缺原素材環境，再揭露呈現器舊prefix限制。補齊FD2_FDOTHER／原資產包，READY六筆event72插入契約先行後修改正式validator；post-frontier-r3乾淨通過，21種完整組間插入與錯序原子拒收、93筆正式戰後、舊76筆保存冷讀保持。正常動態順序仍以完整章收據為準。

原版r2在2026-10-16 17:24:26 UTC、checkpoint597／round3以143退出，oracle.log無錯誤，工具工作階段收到SIGTERM。分類為環境中斷，未清敵、不計整章。r3沿相同r2凍結計畫與AP5000槽重跑，改由獨立Docker容器內timeout6000s及退出紀錄管理生命週期。
第72號死亡事件兩次35822與第74號同一135DD／10B4E，READY先加入72後補PAN回交；三項局部回歸全部通過，35C18最後range_one保持。#224不因此結案。

戰後首個fixture失敗的原因勘誤：startNativeUnitDeathPresent只在非靜音且g.sfx缺sample3時拒收。direct-entry fixture未初始化音效，也未設FD2_MUTE；明示靜音後可繼續。先前籠統記為素材環境缺件，不代表FD2_FDOTHER或素材包確實缺檔；音訊聆聽驗收未提升。

原版r3超過12分鐘後穩定運作，前598份checkpoint JSON與r2逐bytes相同，已排除生命周期調整改變狀態。round7／8 Yunie卡16,46，raw26=0；前方17,46／16,45守軍尚在，助手仍選後方近敵。主動停止，未清敵、不計整章。READY修正正常接戰策略：每助手依原生MV，先前方敵軍、再距離與索引，無前敵才退回最近其他敵軍；不改原版規則。

r4原版在第30回合觸發75及首波74，3781／round31／84筆以控制器exit19結束，沒有戰敗、清敵或保存。接管shell時容器已退出，未保留原程序；FD2.TMP為207360 bytes工作檔，非完整SAV。r5延至40輪，新增選用正常貼身攻擊解阻，固定同一AP5000來源與seed4，避免再次等後方守軍。
Game r4在正常石碑wait seq3569因raw8來源缺件拒收，已列#229；MaterializeNativePersistentPartyRecord有來源，applyPersistentStats漏帶raw8，READY後補單行，拒收不放寬。

2026-10-17第29章追加勘誤：r5正常護送於T21到碑，T25物化93筆，T26抽79後一次HP清敵28筆，T27仍留戰場。暫停driver並保留原程序查證，確認結果表基址51B19的raw28表項51B89指向20B72；先前誤把單一表項51B51當基址，已由117E7實際call運算元修正。既有IDA完整131-byte handler與Capstone34指令相同，沒有205BE全滅掃描；state18／19／20須由三龍death73／77／78正常派送，單純HP0不設旗標。r5保留failed並以143停止，不計整章、不寫SAV。#230與chapter_result_contract先達READY，才補typed初始code0、三旗標code2及record0／1敗北優先，string9沿正式文字索引lower。r6沿同一AP5000來源及seed4重跑，全部增援與79後正常攻擊三頭龍，不再清敵；原版戰後35BBA自行清場。

工具檢查補記：186項Python回歸在Capstone容器通過但Pillow測試略過1項，已在素材容器補跑35項收據測試全部通過。新正常三龍控制122項driver測試通過。新增純文字canonical guard首次因掃描器未含.txt而失敗；只補.txt支援後同一檢查211條／94guard乾淨通過，不改證據marker。canonical重匯入先被過嚴的零warning斷言及絕對來源路徑拒收，屬操作問題；保留既有extensions.legacy並沿正式相對來源重跑，319節點ID保持。

r6於工具審查初期停止，finish_ui誤加move_unit已改回engage，新增直接接戰收尾測試後123項全過；r7同一plan r4及AP5000來源重新凍結，driver SHA9298cf88不再修改。Game r5負例前兩輪signal killed保留；較大12GB診斷容器取得實際失敗：seq2753的END在76 pulse期間重入，nativeEndTurnRecovery與pulse互等，峰值8792117248 bytes、cgroup OOM kill0。依READY #231補正式Update／END與重播控制權閘，原有frame budget不變。修後同源能完成，但79在END早跑、隨後13536清位，AI多出90／91；#232先依既有1A30B及35EE6達READY，再把79移到清位後。受控事件seed逐筆取原版入口40262／3223，分別得到27345／58716，metadata揭露，不要求整段內部骰序。負例r6乾淨通過963筆AI、零順序差異；原版仍缺三龍死亡旗標，不計整章。局部新fixture首輪跑入缺完整AI前置的mode2拒收，改成只驗本fixture的清位／pair交接；正式AI順序由同源長程重播驗證。

最後正式Go修正後再次重綁字串盤點，98筆候選與各處置內容／數量保持。原版r7實測單核及約400MiB，先前12GiB診斷容器已退出；改用兩CPU、8GiB、Go軟上限4GiB有界並行完整回歸，總容器上限維持12GiB，避免等原版重跑結束才發現舊fixture問題。

第29章原版r7完成：seq3342／T30，349動作／3343 checkpoint；三龍91／90／92依序於seq2986／3068／3258正常死亡，最後同seq寫20B93並進2548C。章內零state_injections、零HP鎖與零force-enemy-clear。post追加group9至94，record_question3323、正常保存3333有兩次成功DOS寫入，最後返回選人。首次Game完整1065 AI零分歧與整檔SAV相同，但3271多出勝利確認而節點失敗；#233先依main25E02／25E23達READY，再補可編輯auto-advance。auto-r2 production已進正確post，重播工具漏既有post分支而未消費劇情；修正由battle_start的正式on_win承接，沒有放寬gate或縮短演出。auto-r3乾淨通過349行為／351節點與strict保存，22,987 bytes含checksum零差異，SHA33c3d74deec1be3ca87b3754a26f2d5ee03799918e237958017be7518b164a6f。章台帳登記26／30，下一章30，本批不開新章。

完整Go r2十九套件已通過；最後auto-advance修正後完整r3另跑。191項Python零失敗／略過，回合與死亡同步check通過且固定回合待轉寫仍13，214條教訓／97guard通過。新增結果／旗標／phase指令另141筆對原始bytes及Capstone邊界相同，不與先前重疊區間相加作覆蓋率。隔離生成器scenario及battle節點逐值相同；最後盤點與上次清冊逐byte相同，98筆處置保持。沒有新matching編譯或錄影。

最後auto-advance與重播交接修正後，完整Go r3在相同FDOTHER／DATO環境乾淨通過19套件，cmd/fd2 205.181秒、exit0、峰值4468736000 bytes、OOM kill0。隔離生成器再次核對最終第29章scenario與battle節點逐值相同，preparation必出／前置欄位相同；正式執行來源雜湊全部相符。正式章收據與完整SAV一致結果保持。

生成器收尾檢查勘誤：新增整個preparation相等的斷言過廣，正式節點另含既有party_limit19，骨架未產生該欄位。未改正式來源；核對限於本輪必出／前置欄位，scenario與battle仍逐值完整相同。

GitHub #227～#233已依完整章與最後回歸結案，#199接第30章。官方pull／render／verify為22項、14項人工判斷、可能已完成0，標籤相符。53個異動檔擁有權1000:1000，66份canonical輸出雜湊相符，9個新證據／計畫入口與37條新增本地連結均有效。README、原EXE、父槽、章前來源、最終SAV與凍結driver雜湊保持；全樹歷史root-owned2811、md目錄0、掃描錯誤0。本輪無錄影、FD2容器無殘留。工作產物留在work且不加入版控。

## 2026-10-17 第30章正常接續與終局驗收

開工0d60760b，tracked乾淨，官方pull22項與主機gh認證通過，無FD2容器殘留。路由命中復古重製、原版決定性對拍與文件職責，沿既有work。來源第29章實際SAV SHA33c3d74d原樣接續，祖先作弊政策保持，沒有新增強化、matching或錄影。先驗正常LOAD、19人選擇與揭幕，不把短探針計整章。主契約DRAFT與既有終局canonical入口同批登錄00-index；專屬勝負與魔神連鎖先查既有證據，不假定清敵即過關。

第30章原版短探針exit0、94 checkpoint，正常20我方＋group0七筆，共27；入口camera16,17／cursor22,23／visible6,6／selector1。Game初輪提前加groups1..3成33筆，短鏈比較失敗；#234依READY修正資料及生成器，canonical319個節點ID保持，r2短鏈行為／節點通過，不計整章。錯帶--oracle-save到沒有保存動作的短探針屬比較命令問題，移除不適用參數乾淨重跑，正式strict未改。既有IDA9.4匯出299指令、5個表項及FDFIELD_088 ff5200原始bytes／hash核對；81 writer使82從休眠啟動，推翻56全33圖無serialized82的舊斷言。82、80與專屬結果各先立READY，才接正式consumer。




第30章完整原版r2到seq1050／T11／25975永久停駐，1051 checkpoint，四次81／82及三波兩人增援、五魔神正常擊倒、完整25757、selector30、2BCE5、27人蒙太奇與20段尾段均實際走過。章內零注入，DOS create／write零次；兩側SAV整檔與來源33c3d74d相同。Game full-r4消費85筆AI、零順序分岔，行為／獨立節點／交易及終局窄契約全部通過，frames advisory。原版18觀測點失敗及Game guard、名冊consumer、mode11漏回復各失敗樣本保留，READY先於修改；#237／#238由正常流程抓到，沒有新matching。

終局補回FDTXT031遺漏的六段以建立60句完整索引，四個caller共42句逐原始controls／glyphs重生。selector30的1筆模板與31格部署容量明示分開，27active／4原始清零格各有來源；2C548實際讀當前scene records，消費四欄與原版相同。原98項字串審查處置不變，新增1項loadErr契約診斷。完整Go回歸另記實際結果，不把fixture、預覽或綠色單元測試提升為整章。

最後Game full-r5再次通過，正式收據及台帳命令已指向此輪。相同FDOTHER／DATO與素材包環境的完整Go r2通過19套件，cmd/fd2 163.702秒、exit0、峰值4304441344 bytes、OOM kill0；相關Python183項及嚴格四語8項均通過，四包各5106筆結構驗證通過。生成器第30章scenario／battle／post逐值相同，整備必出／前置同為9；全域回合／死亡同步、終局字庫與216條教訓的99條guard均通過。68,022,299 bytes失敗log壓至1,778,761 bytes，解壓SHA相同，保留.gz。

GitHub #234～#238及父#199已結案；#239保留新恢復六句三語人工校譯，#200接第1～3章，HD待辦保持。官方pull／render／verify為22項、14項人工判斷、可能已完成0，遠端狀態標籤相符。66個異動檔擁有權1000:1000，66份canonical輸出SHA相符，22條新增本地連結有效，新證據／計畫入口有00-index路由。README、原EXE、父槽、章前來源及完整輪次凍結driver雜湊保持；全樹歷史root-owned2811、md目錄0、掃描錯誤0。本輪無錄影或matching，無非預期FD2容器殘留；work不加入版控。

## 2026-10-17 第1～3章接續驗收

開工961d9525，tracked乾淨，主機gh認證及官方pull22項通過，無FD2容器殘留。路由命中復古重製、原版決定性對拍、規格閘門與文件職責。#200要求目前正式引擎正常START到首章，既有ch01-clear計畫實際走CONTINUE，不能沿用為START收據。沿用已閉合開場／首章事件證據，先建DRAFT主契約與正常START短計畫並掛入00-index，不啟動matching或生成錄影。

正常START prefix-r2原版exit0、seq191、434516894步；正式Game由完整開場動畫／START及既有序章owner到首章可操作，19組來源call、97句、118次按鍵推進一致。首比較失敗只有record9：原版32917→32975寫raw+5=1而HP仍28，Game已正確停用；比較器與driver只看HP造成假差異。#240依READY補地圖caller接納條件，不賦予bit0全域死亡／存活語意。原失敗保留，prefix-r4三閘門通過但不計整章。187項Python通過，28／29／30既有同源收據由新比較器重比仍passed，不新增章驗收。

原版前綴r1的20M等鍵觀測及完整r1的每次攻擊140M額外settle均主動停止；兩者是測試計畫冗餘等待，沒有戰鬥失敗或重選seed。完整r2固定2M觀測、strike_settle1，仍由正常cursor／dialogue owner收尾，保留START、戰鬥至T7四個事件、111清敵及城鎮正常保存；開始後凍結driver與計畫。

完整Go19套件回歸通過，cmd/fd2 410.878秒、exit0；191項Python工具回歸通過。full-r2進一步發現工具預設MV6與原版raw+3B矛盾：四名角色為4／4／7／4，原先只歸因住宅地形不足，保留假說並由raw證據取代。#241新增明示native_move_budget及candidate_map，僅改測試候選，未選用的歷史計畫排序不變。新r3開跑後再次凍結driver及計畫；已停止r1另用正式Game核對首個物理及反擊，不以局部提升整章。

完整r3的raw預算移動已正常推進，索爾7,14→5,16等被原版Enter接受；先前超距重試問題已消除。但前線策略令索爾T2自然死亡，原版返回標題，exit15，該輪不計完整章。r4保留同一未修改SAV與固定起點，只正常將索爾／悠妮退到東側，再抽一次亞雷斯物理，T2索爾／蓋亞退守後正常END至T7。這是第二個自然戰鬥策略，不重選seed、不鎖HP、不做章內座標／狀態注入。

完整原版full-r4正常到T7／seq938，939點與55筆AI，四回合事件均到達；一次物理與正常END抽樣後依111清敵六筆，無章內HP鎖。Game r4首差是原版phase metadata缺漏，#242先READY後補四事件所有action camp；r5消除T3首差但T4新敵軍漏行動。#243沿原版constructor移除GUI額外Acted覆寫，r6／r8均55筆AI零分岔，行為／節點通過。保存首次形成共同狀態後才確定#244：START未建LOAD raw基底，酒店只寫JSON，原版SAV未變。

測試側DRAFT提案保留其他槽、目前戰況及metadata尾段，沿既有零基底與typed隊伍建構首槽。正式標題LOAD冷讀5人能力／物品、章節、金幣與選項一致；與原版實際SAV差2228 bytes，未使用名冊2160、有效紀錄65及檢查碼3。有效紀錄含未用到的AI／死亡欄位，不能全稱不透明；修正兩處原版新開局全零的錯誤註解。依grilling提出限本收據接受差異或繼續來源追查，等待使用者決定，尚未改production或章gate。191項Python乾淨回歸通過；前一命令列錯兩個模組名保留為環境紀錄。

保存政策決策追加：使用者先選逐byte相同，開始沿既有IDA工具鏈追查配置來源，未啟動新matching或新oracle；之後澄清若屬新增欄位則接受建議，再明確回覆「允許差異存在」。確認本案是同尺寸同ABI的既有欄位殘留值，將已展示2228個差異的限本收據方案改為READY。其他來源與章的嚴格政策保持，正式SAV不得抄oracle未知常數。

依使用者明示要求，matching輔助僅重連結既有Watcom10.0a START物件45變體，固定EXE／IDA9.4原始指令與fixup核對通過，G27的25EBB完整663 bytes相同。酒店30012沿用直接memmove2560及metadata十byte證據，完整C未匹配限制保留；65個有效紀錄差異均從同一首章battle首次人物紀錄原值保留到保存。

正式START初次酒店保存以明示可寫容器建立名冊基底，驗證成功及原子取代後才發布；LOAD仍保留raw來源。新增缺START／LOAD來源、壞容器零發布、非目標區域保留及重存決定性測試。冷讀測試重構少右括號造成r9／Go r3在gofmt即停止，修正後預檢成功並乾淨重跑，不當產品缺陷。Game full-r10 exit0，55筆AI全消費、零分岔，正式LOAD冷讀通過。SAV SHAf3942ff2與提案相同；strict仍拒收，明示首章政策後三必要gate通過，正式parity-ch01及台帳更新為28章passed，第2、3章todo。196項Python工具回歸含政策偽造拒收通過，matching契約另由專用CLI實際驗證，回合／死亡同步與218教訓101guard通過；Go r2曾19套件通過，新正式保存完整Go r4另記實際結果。

完整Go r4的18套件通過，cmd/fd2 325.058秒；字串清冊套件因來源位置ID及全清冊SHA過期拒收。最初猜為新增保存診斷，逐檔案／函式／文字／role／參數及重複次序對比後，證實文字多重集合完全相同，99筆go_review處置不變。只將原處置對應到目前行列ID並重生清冊與摘要，受影響套件另作乾淨回歸；不重跑已通過的18套件。

字串清冊套件修正後0.390秒通過，18＋1共19套件的相關回歸閉合；保留完整r4 exit1及修正套件的獨立log。66份canonical SHA相符，2387條本地文件連結有效，40個原異動檔UID/GID1000:1000，歷史root-owned2811、md目錄0、掃描錯誤0。README、原EXE、原始與章前SAV及凍結driver雜湊保持。未產生錄影；全部一次性容器已退出，工作輸出不加入Git。

首章程式與正式證據已提交並推送ace79b15，遠端HEAD核對相同。GitHub #240～#244以該提交與正式收據結案，#200留言保留第2、3章待驗。官方pull／render／verify更新為22項，狀態標籤相符；正式章台帳28 passed、2 todo。最終快照另提交，沒有新增遊戲變更或重跑已通過的章。完整清冊為ignored本機生成物，只提交可重生工具、摘要與原處置映射。

## 2026-10-17 第2章正常接續驗收

接手b38d96fe，tracked乾淨、無FD2容器殘留；主機gh認證與官方pull22項通過，現況28章passed，第2、3章待驗。路由命中復古重製、原版對拍及文件職責，沿既有work。攻略僅作T3六人增援、村民分支與希莉亞JOIN的旁證。沿第1章原版full-r4實際酒店SAV SHA540c3bc1正常LOAD，保留5人及三名HP0，沒有新增強化、復活、HP鎖或matching；先建DRAFT主契約、READY起手與短計畫，全部掛入00-index。

第2章原版startup-r1正常到seq76、21筆、camera9,13／cursor20,14；Game拒收源於計畫把2D11D出戰提示誤標大量選人後的departure_confirmation，改正新r2計畫並保留舊輪次。原版full-r2到T4、seq622、623點，53筆AI與4筆growth；event6一次，六村民raw35/36全為26,15，slot7自然死亡。正常蓋亞物理抽樣後111清敵，戰後希莉亞JOIN及下一城鎮保存成功，沒有HP鎖或新強化。

#245沿既有IDA9.4完整匯出160原始bytes補event6 PAN、ACT13、九句FDTXT002#4及六筆目的地寫入，先READY後降階；既有record_bytes補+36 typed setter及raw投影。起手實測村民mode4反證DRAFT導覽中的mode11，實際map1控制檔004反證002定位，勘誤均保留。純測試先遭舊spawn-only斷言拒收，再查明同步Scenario consumer不消費GUI原語，改測同一ApplyNativeDeathOp規則；三相關套件r3通過，未改正式adapter求通過。196項Python通過，全域尚未完整轉寫事件12筆。

Game完整r2起手名冊相同，第一次移動組圖因HUD來源缺失拒收。#246由原版cp76、33142 focus0與既有LOAD的26112 writer建立READY，再補battle_ch02 view及HUD繼承、生成器和canonical；未關閉renderer預檢。B=1僅用作已證非零顯示，最終raw B未直接擷取仍保留限制。r3同源重播驗證中，章台帳保持28。

Game full-r3正常重播通過53筆AI，零順序分岔；行為／節點passed，保存剩希莉亞+28..30九byte及兩checksum，共11差異。r4新增可明示啟用的正式保存冷讀驗證，實際SAV經新Game loadGame／confirmTitleLoadSlot通過六人能力、8格物品、金幣、章節及選項；strict仍failed，不提升章台帳。使用者要求以matching輔助釐清原因，新增#247，沒有接受第2章政策。既有scene-flow物件126變體重連結，112A5九個完整C候選皆負例；直接IDA指令顯示建構只清+22長6，將以相同SAV／計畫的唯讀memory-change窗口及正對照writer確認建構前殘留來源。205項Python工具回歸通過。canonical66份SHA相符，320節點ID全部保留；先前摘要319為抄錄錯誤，以實檔核對取代。

第2章來源查核完成：原版同一輸入的有界memory-change先讀到runtime27九個保留值，建構時三個正對照writer均捕獲，九byte無值改變。固定EXE／IDA指令核對及工具驗證passed；同輪最終SAV與正式原版d8d534d5相同。正式保存BuildNativeChapterSlot對同步後新入隊紀錄用零基底，與原版未初始化區保留不同；不是新增save欄位。局部副本補九byte重算checksum可重現整份原版，沒有寫回正式SAV。使用者在證據展示後確認本份例外；新政策逐筆綁定全部11差異，負例strict保留failed，正式parity-ch02三必要gate通過，台帳29章，第3章待驗。210項Python工具回歸通過。

數量勘誤：第2章完整原版最終current為seq625／1,698,395,489 steps，共626個checkpoint JSON；早期seq622是保存後較早一點，不是最終停止邊界。主契約已依直接收據修正。完整Go首次18套件通過，cmd/fd2在預設10m的T.Run邊界逾時，依環境分類只重跑該套件、18m上限及json逐項紀錄；未把逾時列為產品缺陷。死亡同步首次用了不存在的舊工具名，改用sync_native_death_programs.py後同容器乾淨通過。擁有權掃描歷史root-owned2811、md目錄0、錯誤0；2343個本地文件連結有效，README／EXE／來源SAV／凍結driver雜湊保持。沒有新增錄影。

依使用者確認接續第3章：攻略只作T3兩端援軍及鐵諾存活JOIN的旁證；起手沿第2章實際原版SAV d8d534d5、六人及三名HP0，零新強化／復活／HP鎖。兩次Left出口由上一章正常town_ch03 probe seq527佐證，先READY起手及DRAFT整章規格，清冊／短計畫／主契約同批加入索引。

第2章完整Go回歸閉合：前輪18套件已通過，唯一逾時的cmd/fd2同工具鏈乾淨重跑643.058秒、exit0，18＋1共19套件。第3章正常LOAD短前綴原版exit0、seq68、15筆；Game完整章前同輸入7.84秒通過，零AI入口抽樣，僅起手E1，不提升第3章台帳。

第2章已提交並推送ab35f6b9，遠端HEAD相同；#245／#246／#247依正式收據結案，#200仍開，第3章待驗，官方快照22條。開始第3章完整r2：正常索爾兩次向橋中退守、蓋亞接戰及三次END，預期T3援軍後27筆，再依111清敵。鐵諾分支由實際狀態決定，無HP鎖或新強化；既有起手cp42..43提供slot6建構窗口，同輪唯讀記錄九個保留值，避免若保存再有差異時重跑整章。

第3章r2在T3已到27筆、索爾29／鐵諾43HP；之後額外第三次END使索爾敗北，最早標題owner為cp365。driver的ally_alive讀到標題留下的舊unit_count與已重用陣列，不能當戰鬥HP；停止無效等待並保留r2。新r3同一SAV／seed／兩次END，在T3援軍抵達且可操作後依111清敵；無HP鎖或新強化。這是有界操作策略調整，不改正式規則或凍結driver。

#248原因勘誤：r2 driver.log在seq341已把round等待判為離開戰場；實際耗盡600次的是後續await_ui cursor。cp365已1FE60標題仍持續至941，exit15。舊raw確已重用且不能判HP，但先前將整段延遲直接歸因ally_alive不成立。Issue內文依直接log修正，原敗北／舊解釋仍留存。


第3章完整r3原版exit0、seq435／1,315,400,367步、436 checkpoint；正常兩次END後T3事件9已到27筆，交回操作即依111清敵20筆。鐵諾存活JOIN、下一城鎮五類服務、酒店保存及返回城鎮通過。成功原版維持f50bd7d4凍結driver，無HP鎖、新強化或重擲seed。既有event9正分支可消費，全域仍12筆待完整轉寫，不新增matching。

#249依原版cp68及LOAD26112既有writer先READY，再補battle_ch03的view／HUD繼承、生成器與canonical；正式移動及完整Game full-r2通過14.79秒，18筆AI全消費、零分岔，正式LOAD冷讀7人／金幣1000。完整strict保存22987 bytes含checksum相同，SHA b942e356f2babb2d936fe7d2daa461f1270175835619f0e59ea4bf43b4ddcd40。戰後先SYNC再JOIN，沒有第二次SYNC；NativeJoinPersistentPending沿同一LOAD基底第6筆，不能套用第2章新入隊零基底結論。正式parity-ch03三閘門通過，工具更新台帳30／30，畫面仍advisory，不推定所有分支、自然難度或HD完成。

#248在成功原版結束後才補do_await_ui guard：wanted UI先接納，title且kbd_pending為整數0即失敗；有待處理鍵、缺metadata及一般對白沿既有界限。cp365真實負例零送鍵即失敗，r3 cp43..68的25筆正常鍵序保持。首fixture省略EXE hash／eip，三個portrait等待分類失去版本脈絡，補齊原始context後斷言通過，未改UI分類或降低斷言。完整215項工具回歸r3只有worklist現況測試失敗：30章已過但#14仍開，屬結案同步，提交後關閉並更新快照重跑。

提交前驗證：正式第3章比較再次三閘門passed；137項driver回歸、219條教訓／102guard、全域回合／死亡同步均通過。66份canonical輸出SHA相符，30份正式章收據均passed。本批只有資料與Python工具變更，既有19套件完整Go及本輪正式Game全章回歸已覆蓋，不重跑無變更Go。全樹連結檢查發現9筆歷史舊交付目標或程式語法誤判，本輪修改文件及README沒有失效目標；保留原歷史文件。初次衛生腳本因素材image未裝git而停止，改由主機git提供唯讀NUL清單，不在主機執行分析。

最終衛生掃描：20個本批提交檔均UID/GID1000:1000，歷史root-owned仍2811，md目錄0，掃描錯誤0。全樹3844個本地目標中保留前述9筆歷史問題，本輪修改文件零失效目標；README文化保存內容與圖片保持，原EXE／第3章來源SAV／實際保存SHA相符。沒有錄影，沒有非預期FD2容器殘留；work不加入版控。

第3章主批36ea3023已推送，遠端HEAD核對相同；#14戰役總項、#200第1～3章及#248／#249依提交與正式證據結案。官方pull20項，render／verify通過且可能已完成0；相同容器乾淨重跑215項Python工具回歸全部通過，原r3的遠端同步失敗保留。第3章及30章台帳完成，現代素材#201、呈現#202與HD#204維持未完成，完整matching #198保持暫停。


## 2026-10-17 正式第一版三平台發行

使用者明示先發布1.0.0，指定macOS、Windows ZIP及Linux AppImage。路由載入版號、dist-all與Windows ZIP契約，完整識別v.1.0.0-20261017；歷史tag不刪不移。公開包沿既有權利邊界只含引擎與可散布資料，HD及實體桌面缺口明示保留。先建立41的READY契約，再補共同stage／verify／attest／collect、三平台原生CI與最後Release發布job。沒有修改遊戲規則或save格式。

actionlint1.7.7語法檢查及6項封包負例回歸通過。初次素材容器單元測試因沒有git而停止，將Git HEAD視為來源比較測試的外部輸入，真實HEAD仍由流水線核對。Linux候選第一輪授權清單取go list -m all，包含未被程式引用、未下載完整source的圖節點而缺Dir；改由go list -deps -json=Module取得實際linked module來源，不刪除必要授權文字。原候選只含stage，沒有正式封包，清除本輪明確部分輸出後同工具鏈重跑。

Linux本機候選已通過：同一Docker image、447份公開資產、實際AppImage從空白cwd進行本版本自檢，沒有原版／私人素材。候選留work且來源工作樹尚未提交，不作正式Release收據。另補type2-runtime固定版本MIT全文及SHA，macOS兩架構的實際模組授權取聯集；正式CI將從乾淨提交重建。

預檢cf9c5cd2的macOS建置步驟失敗，未建立正式tag或Release。公開API可讀狀態與註記，但job log端點回403，未搬運主機憑證；先在受版控macOS流程增加僅含本步驟stdout／stderr的失敗摘要註記，再由公開API讀直接原因，不猜修SDK或規則。Linux候選實際預設啟動已抽frame120，畫面為story_ch00_handler演出，無關卡注入，未用此單張推定完整START。

macOS直接原因已由39041602077的原生註記取得：兩個slice及lipo create成功，Apple lipo將位於-verify_arch後的檔名當成架構旗標，usage明示lipo <input_file> <command>。修正為輸入在前，維持雙架構及後續Mach-O驗證，沒有改SDK、遊戲或放寬gate。cf9c5cd2的Linux／Windows原生建置、自檢及封包均success；正式三平台同來源收據仍待修正後新提交重跑。重複失敗後回查打包路由與封包四層驗證，分類為驗證命令問題。

正式預檢39041776347全數通過後，main快轉至d8c361a8並推送，建立輕量tag v.1.0.0-20261017。Git metadata在sandbox唯讀，改以已授權的unsandboxed Git控制面寫index／refs，沒有覆寫或移動舊tag。正式run39042142693三平台、聚合及發布全success，22:48:16台北時間公開Release 415858640並設為Latest。

受版控verify_fd2_release.py回下載三包至dist-all/v.1.0.0-20261017/patch，與API digest／manifest大小及SHA完全相同。Windows 6,660,970 bytes、macOS 11,329,216 bytes、Linux 6,765,048 bytes；ZIP CRC、UTF-8／BOM／CRLF、兩種Mach-O架構、GUI PE、授權及各447份公開素材全部通過。實際回下載Linux AppImage解包內容及空白cwd自檢亦通過，沒有原版或私人現代素材混入。正式receipt及重跑工具同批加入00-index與41，58新增發行列。未宣稱HD、自然難度、所有分支、簽署／公證或實體平台完成；#9維持實機抽測範圍。

索引檢查勘誤：41原有入口使用打包標題與頁碼，不能只搜尋完整檔名判定缺入口。核對既有標題入口後，同批補上直接可點的41及正式收據連結，沒有新增第二份職責重複文件。

發行收尾依版號契約將工具regex的數字限定為[0-9]，加入全形數字版號拒收案例，6項回歸通過。已發布的ASCII版號及所有附件不受影響，tag／Release不移動、不覆寫。

最後衛生檢查：異動檔與正式交付均UID/GID1000:1000，1496條異動文件本地連結有效。歷史root-owned仍2811、md目錄0、掃描錯誤0；原FD2.EXE SHA保持。清理本輪候選AppImage及stage共38,335,421 bytes，保存SHA／清理紀錄與必要PNG／文字收據，正式三包保留在dist-all。沒有產生錄影；未留下FD2一次性容器。

發行主證據與可重跑工具已提交6ac5616b並推送main；原始正式tag仍固定d8c361a8。GitHub #9回填第一版及三平台原生／回下載證據，保留實機啟動、存讀檔與音訊驗收條件及開啟狀態。官方pull／render／verify仍20項、13項人工判斷、可能已完成0；HD與完整matching的既有狀態保持。

## 2026-10-18 第一版非HD完整包與實際遊玩推廣片

使用者要求從正常開頭、第一章到城鎮購物並介紹技術，完成後另打包三平台非HD完整版。命中推廣影片、原版素材權利、dist-all及Windows ZIP路由，使用reverse-engineer-retro-game-remake技能；公開版號沿已發布v.1.0.0-20261017，發行日與tag來源d8c361a8保持，不更動公開Release。從main建立promo/full-local-v.1.0.0-20261017工作分支。

沿既有Go／X11鏡像補可重現fd2-promo:20261017-r1，替代缺少的歷史game-video工具。正常開場、普通Space略過可略過動畫及START後，沿序章前進到第一章。X11普通鍵盤選亞雷斯、移動、攻擊及反擊；未強化、鎖血或修改敵人。戰後明示以已驗證首章hotel-save.json透過F9接續，普通商店操作買藥草兩件交給索爾，金幣1000→980、道具192新增兩件，F5保存前後核對。這是媒體／重製玩家操作證據，不是新原版同狀態或整章連續自然通關收據。

成片138.035秒、1280×720／30fps／H.264與AAC，SHA aabbf42724f458bf720d14926eee4523dd0d6bcf90f65210434153f91cd5aa92。技術字幕介紹可編輯流程、dosgolem受控驗證、AFM／1824字模及Go／Ebiten三平台；明示尚未HD及城鎮存檔接續。完整解碼及16份抽樣圖通過，字幕無裁切。未調整AAC版本的戰鬥音效峰值達0dB，降低成片3dB並保留視訊stream copy後，平均−20.7dB、峰值−2.7dB。2.5秒遊戲區黑幕為實際反擊後的返回轉場，對話／清單停留與原作靜音窗逐項記錄，不宣稱零黑幕或人耳音色已驗。

三平台完整包沿已發布的相同執行檔，補齊合法原版素材，assets與pack分開。每包43177份素材，原生Windows三份README保留CRLF差異，其餘43174份三平台相同。Windows ZIP 130497727 bytes、macOS通用ZIP 137579176 bytes、Linux AppImage 99240440 bytes；全部逐檔SHA、ZIP CRC、所有非ASCII UTF-8旗標、README BOM／CRLF、GUI PE、雙架構Mach-O、bundle、啟動器與授權通過。實際完整AppImage及Windows ZIP從空cwd自檢並以預設場景擷取通過；Windows為Wine輔助，兩張PNG SHA相同，不推論原版對拍或平台逐幀一致。macOS完整包未執行，#9實機輸入、存讀檔、音訊維持未完成。

環境與腳本勘誤：首輪Docker build旗標不適用buildx，改用專用Dockerfile.dockerignore及既有base；首次有界image匯出到時停止，快取重跑完成。1024個未壓縮影格緩衝下初輪Xvfb／FFmpeg被SIGKILL，沒有該輪cgroup計數，不確定OOM來源；縮至8幀、限制繪圖執行緒後約1GB且OOM計數零。工作樹README不能覆蓋已發布公開基底；Windows三份原生CRLF以平台SHA覆蓋清單精確登錄，不改Game資料。Wine先預建使用者擁有的prefix，再固定C.UTF-8處理中文BAT參數，實際原ZIP啟動器通過；不把Wine環境錯誤列為產品缺陷。跨bind mount的rename回EXDEV，改為複製、核SHA、刪自有暫存；未改正式視訊內容。

全部交付在dist-all/v.1.0.0-20261017/，原公開SHA256SUMS保持，新增DELIVERY-SHA256SUMS列三份公開包、三份本機完整包及影片。完整版與媒體含原版衍生素材，只留本機，不入Git或公開Release。主收據、工具與入口同批登錄00-index及41，58只更新交付列；HD、完整matching與既有20條worklist狀態保持。

驗證後核實檔SHA，刪19份暫存錄影／剪輯共103752637 bytes；正式影片與必要圖、普通輸入、保存前後及音畫收據保留。完整包暫存另移除604960924 bytes，正式三包及原始來源保留。最終Git差異、所有權、連結與Docker清理在本輪收尾核對。

最終核對：七份正式交付物SHA及UID/GID1000:1000通過，1518條本地連結有效，所有新Python語法與兩支shell語法通過。原FD2.EXE SHA保持222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f。222條教訓、105個guard通過；正式worklist仍20項、13項人工、可能完成0。歷史root-owned仍2811、md目錄0，所有本輪FD2容器已退出；舊IDA映像4ac62de83339不屬本輪，不作全域清理。完整包暫存實際移除604960924 bytes，加錄影共708713561 bytes；正式四個新增成品與三份公開包保留。dist-all被.gitignore排除。

本批工具與交付主證據已提交3ddee599並快轉／推送origin/main，遠端HEAD核對相同；正式v.1.0.0-20261017 tag仍為d8c361a8。GitHub #9回填完整包及Wine／Linux抽測證據，留言6236383549，保留實機驗收條件與開啟狀態。交付收據另綁定工具來源提交，未改任何正式成品bytes。

## 2026-10-18 推廣片節奏、序章頭像與1920×1200切片

使用者要求與DOSBox比較影片卡頓，補四語推廣，接續HD。從main 61d2651a建立promo/hd-intro-20261018。先恢復現況、遠端21項worklist與既有Docker工具鏈；全檔matching保持暫停，本輪未改dosgolem原版執行器。路由載入重製／規格閘門、推廣FFmpeg、文件職責與玩家路徑入口。

推廣片直接擷取已發布Go/Ebiten引擎，dosgolem參數不在這條流水線。單FFmpeg同時讀X11與Pulse的代表片段為16.531至18.225fps；拆成獨立音畫後為35.224至49.351fps。DOSBox-X normal core／固定12000 cycles／nosound的輔助片段59.887fps，維持原版來源SHA。比較為相近騎士構圖，未同步動畫相位，不宣稱same-state或硬體wall-clock。完整量測與限制保存於fd2-promo-cadence-20261018.json。

新片136.035秒、1280×720／CFR60，SHA 478d68ff9a4b56254776101d989a52bb86ece37ca48f9e9e286e384fa49c0c42。正常開頭與START、序章、首章普通亞雷斯移動／攻擊及反擊後，以明示F9接戰後槽至城鎮，兩次藥草購物令金幣1000→980、道具192新增兩件。F4實際顯示繁中、簡中、日文及英文商店。全解碼、音畫時長、平均−20.6dB／峰值−3.0dB通過，無超過門檻的黑幕或靜音；八段低變化停留為AFM、標題、對話／交易等鍵或技術字幕。21張中點與字幕邊界已目視，未宣稱人耳驗收或每秒60個不同畫面。12秒DOSBox並列短片SHA 962ae1c06a8fbc4619007ccc8711367743a861c5aef5d307c34b3618b8835d8f。原正式影片aabbf427與公開Release／tag均保持；DELIVERY清冊另列新兩片。

正式序章consumer稽核97筆，需DATO 0／1／4／9／30／48／66／96；缺48父王、66王后、96強盜。正常現代START先在speaker48拒收，補齊後以原正式第一版引擎正常97拍到首章通過。三張1254母稿及80衍生圖經catalog登錄，私人庫fd93f14f已推送並核遠端。原版身分、母稿、產生提示、正常輸入及GPU來源見fd2-modern-portraits-20261018.json，不把母稿存在稱為高解析完成。

使用者先確認父王手繪風格，之後撤回160方案。Ebiten同場景原型比較1280×800／320與1920×1200／480，使用者明確採1920×1200與480×480。先寫61 READY契約，再從31位既有母稿衍生480圖，保留80圖與舊包。新角色集合、契約、SHA、透明度、幾何或部分集合不符即原子拒收。高解析包啟動時使用Ebiten v2.6.6 FinalScreenDrawer，保留640×400遊戲與輸入座標，直接在最終畫面繪製480圖並快取GPU圖片；逐幀清呈現描述，保留幫助、人物資訊與全黑／白遮罩及淡出。背景、字型、戰場及由忠實啟動後F2動態進入HD仍未完成，#204保留。

錄影首輪在人物資訊仍開著時F9讀酒店，節點變更但舊資訊全畫面殘留，新增#250。LOAD成功發布處清nativePlayerStatus，驗證失敗時原指標保持。原有存讀檔回歸先紅後綠；GPU建構槽普通Home→F9酒店→Escape城鎮通過，限RUNTIME-E1，非正常開頭完整E2或原版對拍。重製正常購物片改在資訊完全關閉後讀檔，沒有以後製掩蓋缺陷。

驗證工具與環境勘誤：Go容器login shell會重置PATH，改bash -c。PNG時間戳不能比字幕內容，ImageMagick的%[signature]在此版本回空值；改%#並要求64位實際像素簽章。FFmpeg concat清單相對路徑會重複目錄，改絕對路徑。未交付新片的載入黑幕已剪除，舊正式影片未動。Bash在執行中讀取來源被修改導致EOF，EXIT trap已合併有效影音；擷取器改啟動時複製不可變副本並記SHA。

GPU單元測試不能在RunGame前ReadPixels；改驗真實Draw提交的幾何與細節倍率，真正像素另以正常視窗擷取對清冊480圖MAE核對。Go測試的浮點矩陣用1e-6誤差，不把浮點捨入當產品幾何缺陷。新GUI工具先遇Pulse設定目錄不可寫，再把原版pack錯當公開assets根目錄；以XDG設定目錄及分開的assets／FD2_ASSET_PACK修正，沒有改遊戲規則。故事line在不同caller重起，只用story／line會取錯角色；取樣鍵改保留原始handler_address並拒收重複。中止的不完整輪次不作玩家通過證據。

相關Go回歸含完整私人包、舊包相容、HD缺件／孤兒／重複／壞SHA／尺寸／透明度、上下錨點、細節倍率、GPU快取與遮罩，最終4.340秒通過。Python清冊195項與1187個file／files通過；31張480候選已按allow-list準備私人同步，待完整序章結果後獨立提交。正常1920視窗已確認索爾、父王及王后480區域MAE=0，完整97拍仍執行中。最終收據、私人提交、媒體清理與公開提交於本節後續追加。

高解析最終正常輪hd-normal-r4通過97拍至battle_ch01，七位頭像480區域MAE=0，含父王、王后與強盜；GPU圖已目視。61提升本切片CONFORMED／PLAYER-E2-LIMITED，正式來源與時序觀測綁fd2-modern-hd-20261018.json。F2由忠實啟動後動態進HD及整體介面／戰場、跨平台效能仍未完成。

核對九份現行交付SHA／所有權與不可變原版EXE通過。本輪40份暫存音畫與剪輯426217972 bytes已依使用者要求刪除，正式三片、PNG、操作、存檔與雜湊清冊保留。新HD流程沒有建立錄影。受影響文件2590條本地連結零失效；無範圍全庫掃描額外包含work歷史樹與已歸檔dist，64項既有失效不作本輪產品缺陷。226條教訓及109個guard通過。

私人HD素材獨立提交3f289e7bad8dae340d803f0176843fb2c3b3ea70，推送origin/main後ls-remote完全相同；33個變更檔只含31張480圖及catalog／manifest。1313份檔案SHA與UID1000核對，舊164筆清冊語意不變。

收尾所有權掃描：歷史root-owned檔案2786個、md目錄0個，本輪新檔與私人同步均UID1000；不修改歷史root檔。兩份開發執行檔驗證後刪除37718792 bytes，來源與雜湊留收據。所有本輪FD2容器已退出，未動其他專案容器或映像。

收尾所有權自檢限本輪工作根work/promo-cadence-20261018：root-owned檔案0、md目錄0。全樹掃描已自行完成，結果為2786個root-owned檔案、md目錄0；這次只計檔案，不把歷史不同範圍的2811數字混用。兩份開發執行檔驗證後移除，雜湊與大小留高解析收據。所有本輪FD2容器已退出，未動其他專案容器／映像。提交前受影響文件2595條連結、正式來源SHA與舊161筆清冊語意核對通過。

本批實作與主證據提交55af16c5ecbc8963b102c8283575aa7837a58c67，從promo/hd-intro-20261018快轉main並推送，ls-remote核對相同。原v.1.0.0-20261017 tag／Release未改。#250已依該提交及GPU／回歸證據關閉；官方pull更新為20項。#201／#202保留其他角色／章節與操作驗證，#204改為1920×1200正式入口及後續HD，未重開已驗序章。

## 2026-10-18 F2動態高解析與忠實縮放

由main b2716f8c建立hd/f2-runtime-20261018，沿使用者已確認1920×1200／480×480接續61的READY切片。
所有啟動主題共用最終呈現器；忠實模式保留鎖定Ebiten v2.6.6的整數、非整數、縮小及關閉濾鏡契約。
改寫檔保留Apache署名、修改說明與完整授權，既有封包器仍保存原模組LICENSE。

F2原子預檢目前已解析speaker，僅替換素材與目前頭像，不重建對話或重設文字進度。
視窗模式切入1920×1200，切出恢復原大小；成功預檢的素材於同一程序快取，往返不重複解碼。
相關Go及完整私人包回歸通過11.251s；文字盤點與多語系分別2.285s、2.333s通過。
99筆既有審校只搬座標，從審校原提交ace79b15重建舊盤點後遷移，處置數保持。

正常忠實START途中F2走完97個對話拍點至首章，七位480區域MAE=0。
實際遮罩、視窗大小還原、1600×1000與1920×1080縮放通過。
同份首章JSON以忠實／HD兩次F9載入，單位結果相同，保存SHA保持。
F5沿既有節點保存契約，沒有改為戰中快照。五種實際GPU濾鏡比較均為全畫面零像素差異。
來源、入口、診斷、輸入、截圖及SHA見[F2主收據](docs/data/fd2-modern-hd-f2-20261018.json)。

前兩輪工具在同步初始化時漏送F2或提前讀取舊收據；第三輪完成97拍，卻誤以章內暫態比較節點LOAD，
且說明截圖早於GPU呈現。工具已改為等待完成按鍵收據與遮罩像素變化，第四輪乾淨通過。
沒有放寬正式遊戲或資料規則。

最終LOAD畫面另確認非原生對話仍取80×80現代圖，未接480最終層，沿#202接續。
說明標題與首列重疊另開[#251](https://github.com/wicanr2/fd2_re/issues/251)。
#204仍追蹤介面／字型／戰場及平台驗收，不以本切片關閉整體HD。
私人素材未變，遠端main仍3f289e7b且乾淨。正式第一版tag／Release不變，本輪沒有錄影。

提交前遠端Issue已pull／render／verify，21項開啟、14項人工判定、0項訊號消失。228條教訓與111守衛通過。變更文件2580條本地連結零缺件，13份已測來源SHA相同；本輪44,155,968 bytes測試執行檔已移除，無錄影、root-owned新檔或.md目錄，無FD2容器遺留。

## 2026-10-18 非原生對話HD與首章JSON冷讀

從main d230cfbf建立hd/dialogue-final-20261018，沿61既有1920×1200與480圖源授權，先寫READY再接Game.Draw非原生穩定對話。保留既有168邏輯區域、下框鏡像與上框方向；在1920視窗實際504區域直接取480圖源，原生480區域不變。舊80包、忠實模式、開收框與最終遮罩保持，沒有變更規則、存檔或私人素材。

完整私人包及HD／主題／存讀檔Go回歸2.381秒通過。最終四語盤點、審校與在地化三套件回歸8.831／0.195／0.252秒通過。從base重建盤點遷移99筆既有審校，四語既有譯文／狀態／變數保持；5106→5107來源條目只新增一個已譯相同文字別名，無新譯文。本輪未宣稱新的四語GUI覆蓋。

受版控GUI工具以此前正常97拍後F5的原樣JSON複本，兩個新程序普通START後F9冷讀首章。normal-r5在63.014秒完成F2雙向、F1遮罩、1600×1000／1920×1080與還原、F9重讀及HD啟動冷讀。六個GPU504區域最大正規化MAE為0.00162285，舊80放大參考為0.0251632；獨立參照以Triangle取樣，與GPU線性取樣差異明示，不宣稱零像素或原版parity。實際HD冷讀、忠實還原及遮罩／黑邊已目視。存檔SHA保持089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2；主證據見docs/data/fd2-modern-hd-dialogue-20261018.json。

工具r1／r2與r4的短按鍵或舊audit判斷造成漏ACK、多確認或GPU上傳時漏按；改為按下等實際ACK、finally釋放並等後續Update，不改產品。r3實際發現首章F5 JSON無正chapter，標題nativeLoadSlotMetadata拒收，同份F9可讀。新增缺陷#252，保留正常存檔而不猜填章號；本輪冷讀限START後F9，不能宣稱標題LOAD通過。#251說明間距維持待修。

61提升本切片CONFORMED限首章下框PLAYER-E2；非原生上框僅幾何E1。58／56／57與索引同步，其他章節、介面、字型、戰場與平台效能仍待驗。沒有新錄影；本輪開發執行檔已核SHA後刪除，大小與來源保留主收據，既有work與正式發布不變。

收尾官方pull／render／verify為22項、15項人工判定、可能完成0；230條教訓與113個守衛通過。2588條異動文件本地連結零缺件，13份正式來源與11張GPU截圖SHA核對通過。本輪18887184 bytes開發執行檔已移除，無新增root-owned檔案或.md目錄。私人素材庫乾淨，遠端仍3f289e7b；未留下FD2一次性容器。

## 2026-10-18 首章JSON標題LOAD修正

由main a24ed3c7建立fix/ch01-title-load-20261018。命中規格閘門、正常玩家存讀檔及文件職責路由，先追查saveGameToSlot、loadJSONGameFromSlot、nativeLoadSlotMetadata與確認owner。首章handlerChapter零值經omitempty省略，F9可正常恢復；標題誤以原生文字框metadata投影作為JSON讀取gate，並忽略void包裝背後的實際失敗結果。證據與READY契約先保存於docs/data/fd2-json-title-load-20261018.json，沒有重開原版SAV逆向或全檔matching。

標題JSON改與F9共用實際還原結果，成功才清titlePhase；非法槽與章號、未知節點及原有隊伍／選項拒收保持在選槽。共用JSON讀取者在發布前拒收超出0..255的章號。首章零值、writer、schema與原生SAV交易保持，原生槽metadata仍只負責既有投影；首章沿現有在地化四槽畫面。

新回歸在原程式重現零章號拒收與未知節點失敗後離開選槽。正例手建Game缺官方語系，改最小場景仍失敗；查看loadErr並依既有attachOfficialLocale補齊測試資料後，最終四套件24.099／0.005／0.531／0.748秒通過。覆蓋JSON保存／拒收、原生槽與檢查碼、HD切換及在地化。文字盤點重綁僅三筆來源座標，99筆審校及四語各5107筆譯文／狀態／變數完全保持；無新增譯文。JSON輸出依既有sort_keys格式，沒有整檔格式重排。

受版控GUI工具新增--load-entry title，normal-r1在178.953秒完成忠實與HD兩個新程序普通標題LOAD、F2往返、F1遮罩、縮放與F9。普通F5確實重新寫入，再由第三個新程序普通標題LOAD冷讀。原樣來源、重新保存及冷讀後完整JSON bytes相同，SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2；六個HD區域MAE均0.0016226，插值近似限制保持。實際選槽、HD與第三次忠實冷讀PNG已目視；只提升首章JSON玩家路徑，未新增原版對拍或跨平台實機聲明。

56／57／58／61、README與索引同步限定現況，較早只驗F9的收據不回溯提升。#202／#204保留其他玩家路徑及介面、字型、戰場與平台驗收，#251仍待修。沒有新錄影；本輪18887224 bytes開發執行檔核SHA後已刪除，PNG、普通輸入及存檔保留。

提交前2593條本地連結零缺件，14份正式來源與15張GPU截圖SHA通過，來源與重新F5保存完整bytes相同。230條教訓與113個守衛通過，異動檔UID/GID1000:1000，本輪工作根無root-owned新檔或.md目錄。私人素材庫乾淨，遠端main仍3f289e7b，沒有FD2容器遺留。

修正與玩家主收據已提交8f89be9115f57fc0baca347a5610b38045f863b1，快轉推送main後與遠端核對相同。#252依該提交關閉，#202／#204回填限定標題讀取結果；官方pull／render／verify為21項、14項人工判定、可能完成0。生成工作清單9條本地連結通過，較早的#252負例與歷史pending紀錄保留，現行入口已指向新收據。主契約另綁定實際來源提交，不更動已驗程式、存檔或GPU產物。

## 2026-10-18：HD 清晰黑體主路徑與 README 四語截圖

使用者選定 A 清晰黑體。一般介面、原生故事逐字／捲動與商店文字接最終1920×1200字級；商店金幣、價格與效果數字沿原目的座標直接繪字。Noto TTC依已驗證family選TC／SC／JP；忠實模式與舊包保留。原生固定面板、轉場及圖示內點陣字樣尚未接入，#204保持開啟。

逐步字形差異測試揭露捲動快照與後續寫字共用next緩衝區，已改獨立快照，跨FFFD全部步驟測試通過，缺陷登錄#253。說明頁標題間距與寬度適配修正#251。正常START四語首句、F2往返、F9接續旅店後城鎮／商店及單筆購買1000→990、三種視窗尺寸與最大字級通過；首章標題LOAD、F5／F9及完整保存bytes保持。主收據為docs/data/fd2-modern-hd-fonts-20261018.json。

cmd/fd2、campaign、fdother及localization回歸通過。最初未指定素材包的失敗與非標準4GiB程序被終止保留為環境診斷；依既有8GiB／4CPU契約重跑cmd通過。GUI等待與單筆／兩筆購買期待值修正後，同建置完整重跑通過。四語各5107筆內容、譯文、狀態與變數與base相同，僅遷移來源座標。

README新增四語忠實像素商店截圖，明示重製執行期與F9接續來源，不冒稱DOS原版新增語言。2519個本地連結、文化入口與四張圖片SHA核對通過；231條教訓／114個guard通過。未建立錄影，刪除本輪暫時建置與字型副本，保留PNG、收據與輸出雜湊。正式第一版tag保持，私人美術未變。


## 2026-10-18：原生旅館固定文字 HD 接續

從 main ceb1dc5c 建立 feat/hd-native-facility-fonts-20261018。命中規格閘門、正常玩家操作與文件職責路由；旅館提示、四槽及服務回覆沿既有FDTXT字模合成器，61先加入READY契約。字模包裝只用於當次Draw，保留驗證、背景填色、未知與歧義字形回退。槽框覆蓋提示時清除提示文字描述，共享字模及原始素材不變。原生旅館維持既有文字，沒有新增四語翻譯或改服務規則。

最初字模驗證、原字模不可變、旅館共享資源還原與槽框遮蔽測試通過。四語各5107筆以相同身分重綁，沒有新譯文。hotel-gui-r1與最初完整回歸由代理停止，未列通過；較密觀測先在r2、r3揭露連續Escape沒有新JustPressed，等待ACK+2或放鍵後重新取得frame都不足以證明已處理放鍵。重查實機路由與鎖定Ebiten API後，唯讀收據加入JustReleased事件；GUI須拿到同鍵放開收據才繼續。FD2_INPUT_AUDIT_INTERVAL只調整觀測密度，未設定仍120；遊戲時序不變。失敗樣本與既有教訓保留，不列產品缺陷。

旅館r4機械保存檢查通過，但實際GPU保存回覆重疊，拒收並登錄#254。新增去重斷言在saved的[24 238]重現；清除服務回覆重建前的提示描述後，最終focused回歸cmd/fd2 1.950秒、fdtxt 0.026秒通過。r5以新建置完整重跑171.367秒，13張實際視窗通過，保存與讀取回覆已目視無殘留。兩主題保存及來源槽SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

完整API與觀測器回歸cmd/fd2 643.131秒、campaign 4.899秒、fdtxt 0.460秒、localization 1.480秒通過。其來源指紋保留於hotel-regression-r3-source.json，為服務回覆清除三行與新增去重斷言之前；最終修正另以focused及GUI驗證，不把早版全套外推成最終來源全套。四語各5107筆保持。58／56／57／61與README同步限定範圍，README加入實際HD旅館PNG，原四語忠實截圖保持。無新美術、私人素材或正式Release變更，matching維持暫停。最終暫時執行檔核SHA後已刪除，釋出18946288 bytes，未產生錄影。

來源快照註記：完整測試已編譯後才蒐集來源清冊，其間加入去重斷言，故清冊不能證明該斷言進入完整測試程式。逐檔核對生產來源只差native_hotel_ui.go的服務回覆清除三行；新斷言與最終版仍由focused與r5驗證。2618條異動文件本地連結零缺件，README文化入口與原四語圖保持。

提交前官方pull／render／verify為21項、14項人工判定、可能完成0；#254待來源提交後結案。231條教訓與114個守衛通過，更新放鍵事件守衛。新增產物UID/GID1000:1000，工作根無root-owned檔案或.md目錄；FD2一次性容器均已清除。原版EXE SHA保持，私人素材庫及正式tag未修改。

本切片已提交2e0d17897d6fc440c8e65298882d4c46398887c2並快轉推送main，遠端HEAD核對相同。#254已依該來源結案，#204同步已驗旅館範圍與其餘字型待辦。官方pull／render／verify回到20項、13項人工判定、可能完成0；生成清單9條本地連結通過。主收據綁定來源提交，未再修改Go或GPU產物。前段來源快照的生產／測試界線依後續註記勘誤，不能把編譯後蒐集的測試檔當成完整回歸二進位的來源證明。

## 2026-10-18：教會即時面板 HD 字型

從main ad5797b0建立feat/hd-church-fonts-20261018。命中規格閘門、正常玩家操作、文件職責與README路由，61先加入READY契約。教會即時選單、名冊、轉職／復活列表與問句、交付物品及回覆沿既有FDTXT與具型別資料提交高解析文字，金幣與31／42／119數字沿原座標。共享資產只在Draw期間包裝並還原；下框覆蓋前清除文字描述，保留上方金幣。旅館沿共用包裝，商店數字映射抽出共用函式；規則、保存及原始素材未變。

初次文字清冊將新Draw guard的內部模式字串誤列玩家文字，遷移因缺少翻譯拒收。改用具名內部常數後重新盤點與重綁，沒有修改盤點器或猜補翻譯。四語各5107筆譯文、狀態、變數及審校身分保持；5106筆依id、1筆依signature遷移，新增0筆。

初次focused回歸22.356秒通過；最終來源在建置與回歸之前保存960個Go檔案指紋。相關cmd介面回歸83.485秒、campaign 11.168秒、fdtxt 0.331秒、fdother 16.778秒、localization 1.802秒通過。cmd只跑HD、教會、轉職、商店、旅館、JSON、標題LOAD、存槽與語系相關測試，不宣稱完整cmd測試。轉職、空復活名冊、餘額不足、交付目的名冊及滿袋回覆等fixture分支限RUNTIME-E1。

church-gui-r1在251.157秒完成正常START後F9接續既有hotel_ch02，經城鎮進教會；實際使用原槽倒下隊員與物品，沒有新增狀態注入。四語名冊、復活列表／問句與取消、交付物品列表與取消、F2、縮放及說明遮罩通過，15張實際視窗已審查。66次操作均觀察實際按下及放開事件；整理收據時修正xdotool的Return／Left／Right與Ebiten的Enter／ArrowLeft／ArrowRight名稱映射，GUI收據與測試程式未改。名冊F2往返PNG SHA相同。回旅館普通F5／F9，兩主題保存與來源完整bytes相同，SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。

58／56／57／61、主字型收據及索引同步限定現況。README新增HD教會復活確認實際畫面，原四語忠實截圖與文化保存入口保持。#204更新教會即時面板進度，教會快取狀態、其他固定面板、預建轉場及圖示文字仍待辦；整款HD與字型尚未完成。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18960600 bytes；沒有錄影，保留PNG及收據。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2622條異動文件本地連結零缺件，960個Go檔案來源指紋與15張GUI截圖SHA一致，原版EXE SHA保持。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄；Go／GUI一次性容器均已退出清除。

教會切片來源已提交48ab3e57dc7ea72f3f01fd4726ff26be59748c23並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18：角色狀態與命令穩定頁 HD 字型

從main 8d725032建立feat/hd-status-fonts-20261018。載入規格閘門、正常玩家操作、文件分工及README路由，61先列READY。原先教會及戰場狀態在Update預建像素快取；新增開啟時具型別輸入快照，保留80-byte紀錄、無動態文字底圖、素材、物品列與命令資料，供穩定Draw合成HD字模及數字。忠實快取與12／14幀轉場沿原排程，不改規則、存檔、頭像或圖示。

原合成器完整索引bytes、共享素材與快照不可變、live角色／命令表變動不污染已開面板、教會／戰場穩定消費端與轉場無殘留通過。focused 11.905秒；相關cmd介面96.249秒，battle面板／命令合成器0.689秒；campaign 10.050、fdtxt 0.217、fdother 21.737、localization 1.854秒與文字清冊1.952秒通過。未跑完整cmd或battle套件，不外推。Go編輯結束後保存962檔指紋，蒐集與最終建置／回歸並行；期間沒有再改Go，GUI前後逐檔核對相同，不把快照蒐集時點當成編譯來源證明。

來源行號移動後，4864筆依id、243筆依signature遷移，by_text與新增均0。四語各5107筆文字、審校狀態、變數及身分逐項保持。初次patch因main欄位空白錨點不合未套用，讀取實際欄位後修正；不是產品或原版缺陷。

status-gui-r1在246.771秒完成正常START後F9接續既有旅館，經城鎮普通選人進索爾狀態與ID9悠妮命令。24張實際視窗完成審查，78次輸入均有真實按下／放開事件。兩頁F2往返PNG SHA各自相同；縮放／說明遮罩、四語名冊、復活及交付取消、返回旅館F5／F9通過。兩主題保存及來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，未注入角色狀態。戰場消費端與14幀推進另用fixture列RUNTIME-E1，沒有新正常戰場GUI或原版對拍。

56／57／58／61、字型主收據status_followup與索引同步限定範圍；README增加實際HD狀態截圖，文化保存及四語忠實圖保留。#204仍待其他固定面板、預建轉場與圖示字樣，未宣稱完整HD字型。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18965704 bytes，沒有錄影；收據與PNG保留，Go／GUI容器已退出清除。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2626條異動文件本地連結零缺件，962個Go檔案、24張GUI截圖、回歸日誌與README圖片SHA通過；文化入口保持，原版EXE SHA未變。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪FD2一次性容器已清除。

角色狀態切片已提交70d50d7cf78d9de4770d3e685ee79b347d8f166e並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18：物品操作與商店裝備穩定頁 HD 字型

從main 7062d21c建立feat/hd-item-fonts-20261018。命中規格閘門、正常操作、文件分工及README路由，61先列READY。開啟／裝備重建時保存無動態文字底圖、紀錄、素材、效果列與raw slot選取快照；成功發布像素畫面才發布選取，失敗還原既有快照，關閉清除。穩定Draw提交HD字模與數字，原12幀開收框保持像素。每個實際拷貝矩形只遮蔽其內文字，框外部分保留，新文字也裁在拷貝區域；規則、名冊、保存、頭像與圖示不變。

focused r1把ReadPixels放在Ebiten繪製迴圈前，引發command queue panic，分類測試環境錯誤。改驗CPU索引底圖、image身分與快照不變，GPU另走正式GUI；r2在3.855秒通過。初次盤點誤把新Text helper的內部portraits路徑當玩家文字，遷移拒收；改具名資源目錄常數後重綁，4865筆依id、242筆依signature，新增與by_text均0。四語各5107筆譯文、審校狀態、變數及身分保持。最終964檔來源指紋於建置及相關回歸之前保存，期間沒有再改Go。

最終cmd相關模式73.341秒、battle物品面板／selector／命令合成器0.313秒、campaign 4.140、fdtxt 0.241、fdother 9.975、localization 0.995與文字清冊1.467秒通過；不是完整cmd或battle套件。覆蓋有洞raw slot、空袋、裝備刷新與失敗還原、矩形遮蔽、戰場物品及死亡交換caller。其他caller未新增正常GUI或原版對拍。

item-gui-r1在城鎮誤送Right，從0走到教會4而等待武器店，分類腳本失敗。回查main的原生城鎮Right=-1／Left=+1與商店獨立delta定義，修正受版控工具的前往／返回方向；遊戲Go與binary未改。item-gui-r2以同建置完整重跑100.147秒，13張實際視窗已審查，53次操作均有實際按下／放開事件。短劍、皮甲、藥草選取，拒絕裝備藥草及重新裝備既有短劍，F2、縮放／說明遮罩、返回與F5／F9通過。角色與金幣保持，F2往返PNG SHA相同，兩主題保存與來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，沒有新增狀態注入。

56／57／58／61、字型主收據item_followup與索引同步限定範圍，README加入實際HD商店裝備截圖，保留四語忠實圖與文化入口。#204保持開啟，接續原生整備、原生標題LOAD、預建轉場與圖示字樣，不宣稱完整HD字型。無新美術、私人素材或正式Release變更，matching保持暫停。暫時執行檔核SHA後刪除，釋出18981432 bytes，無錄影；PNG與收據保留，一次性GUI與回歸容器已清除。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；231條教訓與114個守衛通過。2630條異動文件本地連結零缺件，964份Go來源、13張GUI圖片、主收據與回歸日誌SHA通過；README文化入口及原四語圖保持，原版EXE SHA未變。新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，相關一次性容器已清除。

物品操作切片已提交316f3a18a7781082c2eb307959a2b77d5e0bd378並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。


## 2026-10-18：整備與標題讀檔穩定畫面 HD 字型

從main 02933ee7建立feat/hd-preparation-load-fonts-20261018。命中規格閘門、文件職責與正常操作路由，61先列READY。整備在穩定Draw複製資產並沿原字模／已驗數字bank提交HD文字，繪製後還原；標題四槽使用自身palette，完整畫面清除舊文字。原預建開收框、部署規則、槽metadata、JSON後備及SAV交易保持。

最終965份Go來源在建置與回歸前固定。相關介面回歸53.889秒，campaign 11.900、fdtxt 1.133、fdother 20.849、localization 1.441及清冊2.248秒通過，不宣稱完整cmd回歸。必選訊息初測用了不合法identity0，依既有1..31改為1後通過。初次重構移出Draw的錯誤訊息造成清冊少一筆，保留原入口並由修改前基底重綁，四語各5107筆譯文／狀態／變數／身份逐筆相同；5106筆依id、1筆依signature，無新增翻譯。

JSON普通標題LOAD接續既有hotel_ch02，城鎮出發選NO後返回旅館F5／F9，11張GUI、36次按下與放鍵收據通過，兩主題保存及來源SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。第24章原生槽另走普通標題LOAD、保存第二槽、選人15筆、確認取消、F2、縮放與說明遮罩。來源為原版第23章SAV，祖先含攻略建構與清敵；本輪不新增注入，不提升自然難度或整章PLAYER-E2。最終native r3有17張GUI與52次按下／放鍵，兩主題保存SHA均d5f39666ae53f7b55438a1d4658d38b47a301e434c8aaed65c5ed6fac73fae09，後續選人及取消不改保存。

native r1按鍵通過，但縮小視窗後只隔4個Update擷取，GPU仍舊viewport而裁切，不列縮放通過。工具等待20個後續Update並記錄X11幾何。native r2另遇Xvfb尚未就緒，加入getdisplaygeometry有界檢查。相同Go及binary完整r3重跑，實際相隔30幀，1600×1000完整顯示；上方靜態HUD縮放診斷誤差由0.189337降至0.003671，非原版對拍。原失敗日誌保留，兩條可重用教訓與守衛加入清冊。

最終28張實際視窗已審查。56／57／58／61、字型主收據preparation_followup及索引同步；README新增HD整備與標題讀檔圖，保留四語忠實圖及文化內容。剩餘未列固定畫面、預建轉場／金幣滾動、圖示內文字與平台效能由#204追蹤，不宣稱整款HD字型完成。matching暫停，無新美術、私人素材或Release修改。暫時執行檔核SHA後刪除，釋出18990448 bytes；無錄影，PNG與收據保留。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；233條教訓與116個守衛通過。2634條異動文件本地連結零缺件，965份Go來源與新圖片SHA保持，原版EXE與正式第一版tag未變。README文化入口與四語忠實圖保留；新增檔案UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪一次性容器均已清除。

整備與標題讀檔切片已提交cde408b492c4a911965d5df3b0be5f4f3ac59300並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。


## 2026-10-18：整備問句預建轉場 HD 字型

從main ca028679建立feat/hd-preparation-transitions-20261018。命中規格閘門、正常操作與README路由，使用reverse-engineer-retro-game-remake技能，61先列READY。從既有六個建立端確認六張空框、四張問句選項動畫及收框後caller還原；必選訊息為六張空框後一張文字。為登記階段保存獨立底圖、原字模／文本／選項素材、問句種類與必選身份，在Draw重建HD畫面。原索引快取、Draw承認、輸入鎖定、還原與一次性延續點保持；其他job沿原路徑，不在Update包裝字模，沒有新增原版RE或matching工作。

七種開收配方逐張等同原索引畫格，後續來源變更不改既有轉場；中途主題切換及錯誤回退通過。首輪測試誤把收框第5張算作問句，且錯誤回退fixture缺真實palette，依原四張文字階段與素材契約修正，focused r2為12.504秒。最終967份Go來源在建置與回歸前固定，相關介面回歸77.535秒；campaign 4.106、fdtxt 0.122、fdother 10.141、localization 0.787與清冊0.826秒通過，不宣稱完整cmd回歸。四語各5107筆譯文／狀態／變數／身份逐筆保持，5106筆依id、1筆依signature，沒有新增翻譯。

GUI r1的CPU階段判定不足：preparation-transition-gui-json-r1雖回報PASS，frame7與14筆文字的PNG仍空框，另一CPU-contained圖片仍為穩定頁。此PASS由實際GPU審圖推翻，不作轉場完成證據。native r1反因擷取後CPU離開四張文字階段，拒收已呈現HD字與移動中選項的樣本，CLI失敗。回查正常操作路由後，改用實際PNG問句RGB完全等同HD參照，且選項金色邊界須較穩定頁內移。CPU前後狀態照實保留，不宣稱精確同幀；空框／穩定頁負例保留，新增可重用教訓與守衛。

同一Go與binary乾淨重跑兩條r2。JSON普通標題LOAD後城鎮出發選NO、返回旅館F5／F9，有11張普通畫面、2張有效開收轉場及36次按下／放鍵；兩主題與來源完整SHA均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456。晚期第24章原生槽普通LOAD、保存第二槽、選人15筆、確認取消、F2與縮放／說明，有17張普通畫面、3張GPU轉場及52次按下／放鍵。記錄問句開收框與確認問句收框通過，兩主題保存SHA均d5f39666ae53f7b55438a1d4658d38b47a301e434c8aaed65c5ed6fac73fae09；後續選人及取消不改保存。晚期槽祖先有攻略建構與清敵，本輪沒有新增注入，不提升自然難度或整章PLAYER-E2；必選訊息限fixture。

最終33張實際視窗已審查，5張轉場問句區RGB相同，選項金色邊界由穩定1392..1727內移至1416..1703或1440..1679。56／57／58／61、字型主收據preparation_transition_followup及索引同步。README加入城鎮開框與整備確認收框實際圖，保留四語忠實圖與文化入口。其餘設施／角色／物品轉場、金幣滾動、圖示內文字及平台效能由#204追蹤，完整字型與HD仍未完成。無新美術、私人素材或Release變更；核對SHA後刪除暫時執行檔，釋出18996616 bytes，沒有錄影，PNG與收據保留。

提交前官方pull／render／verify為20項、13項人工判定、可能完成0；234條教訓與117個守衛通過。2640條異動文件本地連結零缺件，967份Go來源、GUI工具及圖片SHA通過。README文化入口與原四語圖、原版EXE及正式第一版tag保持。新增來源與圖片UID/GID1000:1000，工作根無root-owned檔案或.md目錄，本輪一次性容器已清除。

整備問句轉場切片已提交5f6934373c0c25a7c97c0302fc0e5e57f8a40a6c並快轉推送main，遠端HEAD核對相同。#204保持開啟並綁定來源提交；主收據補入source_commit，不修改已驗Go、GUI工具或產物。

## 2026-10-18 HD 金幣滾動與出售確認

從main的a586bcc9建立feat/hd-gold-roll-20261018。依復古遊戲路由載入規格先行、正常玩家路徑及README／文件職責契約，沿用重製技能。61先登記READY，再將既有扣款／入款的八位獨立滾動抽成共用窗口計畫；原索引合成及HD使用同份計畫。交易建構時保存背景與窗口，HD Draw只替換金額區，原10毫秒步長、金額／物品提交、輸入鎖定及延續點保持，不重開matching或硬體時序RE。

新測試首輪括號錯誤，修正後r2／r3指出素材假設不符。清冊實際記錄數字1為5×8，其餘為6×8，字距仍6；修正精確來源檢查後focused r4通過。清冊遷移第一次誤用editor-canonical根，查明正式來源在story子目錄後重跑，無產品修改。四語各5107筆，只遷移座標與來源雜湊，譯文、狀態、變數與身份保持。

普通START後F9接續既有hotel_ch02，進武器店購買布衣、拒絕立即裝備，再出售同件，餘額1000→950→987。GUI r1保存預期過嚴：既有RemoveNativeCompactInventory會左移全部raw格，包括未使用格的殘值，造成索爾InventorySlots[5]由255變0。原始規則在native_inventory_transfer.go，未因HD修改。修正預期後，HD r2及忠實r1的完整保存SHA同為32c9cd021b3c7112f1b6f16dae8ffa1c753fd45cd0d1fc9d92331e99c83eed66，新程序標題LOAD通過。這是重製兩主題對照，不新增原版SAV差異放行。

逐張審查另發現既有出售確認殘留HD歡迎詞，登錄#255。原生合成器已重建對話區，但HD上層未清除。依61補READY契約，在Draw清除被覆蓋描述，再由同份原生字模擷取問句，不新增翻譯；歡迎詞負例與忠實畫格保持通過。最終Go相關介面回歸73.288秒，campaign／fdtxt／fdother／localization／清冊套件均通過，非完整cmd或battle回歸。最終來源為gold-roll-source-release.json，binary SHA為23091a2a680499906a43820ac9c7fd2c20f599dbad7cd5d48ec2822b542be7b1。

最終GUI r3整張視窗擷取錯過0.45秒扣款階段，只取得入款有效樣本。CPU候選不足以證明GPU，該輪不通過。改抓實際金幣區域後，同份binary的HD r4通過9張正常畫面、兩張有效GPU區域與新程序標題LOAD。忠實r3的買賣與保存通過，但重開程序時舊audit高幀號按鍵被誤認為新程序收據，冷讀工具逾時。工具改為逐程序保存audit，保留失敗紀錄；不調遊戲速度、不重建binary，接續完整重跑。上述環境及驗證腳本失敗不登錄為遊戲缺陷。

最終HD r5與忠實r4使用同份binary／GUI工具完整通過，包含新程序標題LOAD，audit按程序分檔。共22張實際畫面含4張有效HD金幣區域，GPU閘門拒收空框／前一階段／穩定金額，不宣稱精確同幀。兩側完整保存SHA均32c9cd021b3c7112f1b6f16dae8ffa1c753fd45cd0d1fc9d92331e99c83eed66，僅金額與既有未使用格殘值相對來源改變。README加入最終扣款區域及出售確認，原四語圖與文化入口保持；56／57／58／61／索引及字型主收據gold_roll_followup同步。未錄影，核對SHA後刪除暫時binary，釋出19013056 bytes，PNG與收據保留。正式Release／tag、原版檔及私人素材包未修改；其他轉場、成功演出、圖示文字、戰場與平台效能仍待完成。

提交前236條教訓與119個守衛通過；守衛工具略過測試檔，故出售修正守衛綁正式繪字掛勾。2639條本地連結、969份Go來源、GUI工具、README文化入口、圖片與UID/GID核對通過；工作根無root-owned檔案或.md目錄。GitHub #255待來源提交後結案；完整HD仍沿#204接續。

金幣滾動與出售確認切片已提交d19340be3be13ae1e93e7dd5f4bd95b5515e8804，快轉推送main且遠端HEAD核對相同。主收據補綁source_commit；已驗Go、GUI工具、圖片與保存不再修改。

#255已在遠端確認CLOSED，#204維持開啟並更新金幣驗收與剩餘範圍。官方pull／render／verify回到20項、13項人工判定、可能完成0；236條教訓與119個守衛通過。最後只同步來源綁定與工作清單，不修改已驗程式或產物。

## 2026-10-18 角色與物品面板 HD 轉場

上一輪為有效進度：金幣與#255已推送並結案。本輪由12b269f4建立feat/hd-panel-transitions-20261018；先核對Git／容器／遠端#201／#202及官方pull，沿規格先行、正常玩家路徑與文件職責契約，61先READY再實作。教會與戰場狀態開收／命令轉場保存獨立資料與底圖；物品沿既有raw slot快照。像素與文字共用NativeItemPanelRegion，先裁來源，再移動字形及clip，最後裁目的。原12／14張快取、Draw承認、交易與保存保持，不在Update包裝字型。

首輪focused3.821秒通過；其後追加唯讀audit，最終相關cmd回歸48.994秒與五資料／清冊套件通過，非完整cmd或battle回歸。最終來源panel-transition-source-final.json含971份Go與兩份GUI工具，建置與回歸後未再改動。四語各5107筆僅來源／座標遷移，by_id5105、signature2、new0，內容／狀態／變數／身份保持。

同份binary的教會status r1與商店items r1首次完整GUI均通過，共39張正常圖與11張有效GPU區域，50張已逐張審查。區域RGB等同穩定HD參照的移動後裁切，且不同於未移動位置；CPU只定位候選，不宣稱精確同幀。普通START後F9接續既有hotel_ch02，選人／命令、裝備選取／拒絕藥草／重新裝備、F2、縮放／說明遮罩、F5／F9與新程序標題LOAD通過。完整保存與來源均190b9f8f614c69a40eea7cf3883facf92efd83c32b20dca2a1677289437aa456，不增加原版保存例外。戰場caller限E1，尚未新增正常戰場GPU。

另以內建image_gen生成可丟棄的戰場筆觸參照，保存於work/promo-cadence-20261018/hd-fonts/battlefield-hd-style-prototype-v1.png，提示在battlefield-hd-style-prototype-prompt.json，來源與SHA在同名v1.json。實檔1586×992，非1920正式runtime收據，不登錄catalog或私人正式包，不當作完成證據。61既有手繪配色與大頭短身已核准；提問只收集紋理密度偏好，不阻擋既定HD。原型中的裝備與地形仍須依正式來源校對。

下一主切片先以正常首章LOAD驗現代戰場與狀態返回，避免只做孤立字型。合法來源work/promo-cadence-20261018/f2-runtime/normal-r4/user-data/fd2_re/fd2_save.json，node=battle_ch01、SHA089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。現有Home為正常強制檢視鍵；drawNativePlayerStatus在Game.Draw的戰場／介面之後。須實測完整indexed面板是否蓋回現代底圖，尚不能僅憑程式下結論，再接144角色母稿與地形HD。

README加入實際命令轉場局部連結，文化內容及四語圖保持；56／57／58／61／索引與panel_transition_followup同步。未錄影，核對SHA後刪除暫時binary，釋出19034376 bytes，保留PNG與收據。原版檔、正式第一版tag／Release及私人素材包不變；完整HD尚未完成。

提交前20項工作清單、13項人工判定、可能完成0；236條教訓與119個守衛通過。2657條本地連結、最終Go／GUI來源、README文化入口、圖片及UID/GID核對通過，工作根無root-owned檔案或.md目錄。無新缺陷Issue；本輪只更新#204／#202的具體驗收與未完成範圍。

角色與物品面板轉場已提交ecb006ad2fda7eac97c10d8e3d7c6f364812c4e1，快轉整合並推送main；主收據補綁source_commit，已驗Go、GUI工具與產物保持。#204／#202維持開啟，接續實際戰場與其餘HD範圍。

遠端#204／#202已更新並保持開啟。官方pull／render／verify為20項、13項人工判定、可能完成0；236條教訓與119個守衛通過。來源綁定與工作清單快照同步，未修改已驗程式。

## 2026-10-18 實際戰場角色狀態與現代底圖

前輪面板轉場已推送並同步Issue，屬有效進度。本輪由main e82cb099建立feat/hd-battlefield-panels-20261018。命中正常玩家試玩、規格閘門、文件職責與README路由，沿用重製技能。正常標題LOAD既有普通START／F5首章JSON後，Z聚焦、Home檢視索爾，面板外地圖54000像素有50832被換成原版底圖；由實際PNG及presentNativeClassFrameWithPalette全幀consumer確認，登錄#256，61先READY再實作。

現代戰場只繪原NativeItemPanelRegion目的區域，HD文字沿原來源移動與裁切；HD合成失敗仍使用像素面板，不還原完整原版地圖。忠實與教會路徑保持，原12／14張快取與生命週期不變。focused5.052秒、相關cmd35.408秒、清冊0.226及localization0.188秒通過；不是完整cmd或battle回歸。971份Go在回歸與最終建置之間保持，文字清冊bytes完全相同，四語各5107筆不需遷移。

前測工具錯等索爾命令頁，實際無命令確認會關框；依既有NativeCommandIDs改驗索爾與悠妮。final r1機械通過但冷讀圖片尚在空對話框，不能當可讀完成圖；只修GUI等待後續20個Update並正常結束對話，Go未改，重建binary SHA完全相同。最終r2的14張實際圖逐張審查，六組54000像素面板外地圖零差異，F2、縮放／說明、兩角色返回及新程序標題LOAD通過。完整保存與來源SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。沒有戰鬥、座標／血量注入或新原版對拍。

字型主收據battle_status_followup、56／57／58／61及索引同步，README加入實際首章狀態圖片，文化入口與四語忠實圖保留。下一主切片為144×144角色及地形；已盤點94組母稿，48 RGB、45 RGBA、1缺master_file登錄，89組12幀、5組3相。此盤點不當作HDruntime完成，group68須核對及正式登錄來源，不能憑檔名猜補。未新增美術包或私人同步，原版檔與正式Release保持，matching暫停。

核對SHA後刪除暫時binary，沒有錄影；保留PNG、前測失敗與來源收據。#256待本輪來源提交後結案，#202／#204仍待完整戰場與其他HD範圍。

提交前21項工作清單、14項人工判定、可能完成0；238條教訓與120個守衛通過。2575條本地連結、971份Go與GUI來源、README文化入口及原四語圖保持。全work檢查找到三個2026-09-16既有root-owned空dosgolem-cache目錄，確認無FD2容器使用後只對三目錄逐一修復；重查無root-owned或.md目錄。最終圖片及文件UID/GID1000:1000，FD2一次性容器均已退出。

戰場覆蓋修正已提交7dcce7296aeea708b02cd5f27cc9692ade141b5d並快轉整合main，主收據補入source_commit；已驗Go、GUI工具、圖片及完整保存保持。

遠端HEAD已核對7dcce729，#256確認CLOSED；#202／#204保持開啟並更新本輪來源。官方pull／render／verify回到20項、13項人工判定、可能完成0；238條教訓與120個守衛通過。最後僅同步來源綁定與工作清單，已驗程式及產物保持。

## 2026-10-19 HD戰場角色母稿編製

前輪7dcce729與122ebdfb已驗推送，屬有效進度；本輪從main建立feat/hd-battlefield-sprites-20261019。沿既定1920與手繪風格，先以受版控工具從母稿編製144圖，不放大24成品。四組39張樣本包含行走、靜態與重複政策；靜態48母稿本身仍具像素化外觀，不能因尺寸144就宣稱細節提升。全94組候選正在有界容器編製，DRAFT及工具入口已掛61。

本輪已編製94組1083張首輪144候選，hash／尺寸／alpha通過，但固定格線有435筆碰邊診斷，第4組明顯裁切，拒絕進catalog。不能因資料齊備而宣稱美術或runtime完成。工具改全母稿連通輪廓排序及來源alpha保留，拒收不明額外輪廓；0／4／9／48／52五組51張已逐幀審圖，頭部、武器及四方向完整，52保留重複幀。全組r2另有界重編，結果以實際manifest為準。

內建image_gen沿核准手繪風格補FDICON48銀白機械靜態母稿，與DATO48父王分開。新2172×724 RGBA及三個相位已審，來源26f5b0aea38b2665abc8a57099b2dfb88149a50363efa32ed96ac2a09dd82d35，保存於remake/generated-assets/modern-theme-prototypes/fdicon-048-style-a-v2-hd-master.png，完整prompt及來源記錄在work/promo-cadence-20261018/hd-fonts/hd-static48-v2-prompt.json與generation.json。舊稿保持，尚未登錄catalog或同步正式私人包。

同狀態F2比較先發現#257：全域切主題後又被中立戰場F2／Home舊alias消費，前測忠實圖片其實是狀態頁，不能作取景證據。61先READY後只移除戰場F2舊alias，Home保持；相關cmd16.550秒、文字清冊bytes不變，四語各5107筆文件未改。最終普通首章標題LOAD、Z聚焦、F2往返不開狀態、Home、兩角色／命令、縮放／說明、F5／F9與新程序LOAD通過16張實際圖，完整保存與來源SHA均089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。不是完整cmd／battle回歸或原版對拍。

真正雙主題戰場圖片確認#258：相同units／view仍有現代24邏輯直繪與原生320×200兩倍的取景差異。這是下一個HD世界切片，尚未修正。原生NativeFrameInput／frame／placement及Mask-only單位pass已核對，不重做RE；39／57無來源寫入時保持no-op，不能造人物。完整世界與透明介面分層仍DRAFT，尚未接production。

新HD戰場主收據、61及00索引、56／57／58同步，README補F2／Home既定操作並保留文化與四語圖。核SHA後刪除兩份暫時binary，未錄影；PNG及拒收候選保留。#257待來源提交後結案，#258／#201／#202／#204保持工作範圍；完整HD仍未完成。

完整輪廓r2在第3組拒收，只有7個大輪廓，不能安全宣稱12幀；未調低門檻或挑雜點補數。工具追加--audit-all以一次列完全組拒收，有缺件仍非零退出；r3有界執行中，實際終態與manifest待核對。既有51張樣本的編製函式保持，F2相關Go與GUI來源未改。

提交前22項工作清單、15項人工判定、可能完成0；240條教訓與122個守衛通過。2585條本地連結、F2回歸Go／GUI來源、README文化入口與原四語圖保持，文件與工具UID/GID1000:1000。全組素材audit仍為有界工作，未把執行中或拒收候選當完成。

F2修正與HD候選編製器已提交f5fdb81c1ac86fa969842c0ce6bb578fe63e0d71，快轉整合main；新主收據補綁來源提交，已驗Go／GUI與圖片保持。#257依本提交結案，#258及HD主項保持未完成。

完整輪廓audit r3已完成，實際94組均受檢，84組963張通過hash／尺寸／alpha；10組拒收，exit1為來源不齊的明示結果，不是假產品失敗。拒收group為3,10,12,17,18,20,27,44,49,62。未放寬門檻，未登錄production；全組姿態仍待人工審查。新主收據與61補入實際manifest及拒收清單。所有本輪Docker編製／GUI批次已終止清除；內建image_gen另正補3／10／12母稿，尚未採用。

#257已確認遠端CLOSED，#258及HD主項保持開啟；官方pull／render／verify為21項、14項人工判定、可能完成0，240條教訓與122個守衛通過。三份新母稿已保存，但以非零alpha連通仍拒收。分級診斷顯示64／128／192核心可在三組各分出12人物，真因為低透明度橋接；後續只改拓樸判定並保留原alpha邊緣，不硬放行。group10 frame5方向仍錯，另列補繪。已驗Go／GUI保持。

GitHub #201／#202／#204已回填全組診斷84組963張及10組拒收的實際數字，官方pull／render／verify21項、14項人工判定、可能完成0；原240條教訓及122守衛驗證保持。F2來源971份Go及GUI SHA不變，正式第一版tag保持，所有本輪Docker與內建圖像生成皆已終止，素材仍未登錄。

## 2026-10-19 HD透明度核心與候選校稿

從已核遠端c78072f4建立feat/hd-sprite-alpha-core-20261019，保留接手的核心分離變更。
alpha128只識別人物核心，取回原覆蓋邊緣；單幀明示替換另記來源SHA，重複相位完整沿用canonical來源。
獨立驗證器檢查來源、144尺寸、覆蓋alpha、原墨跡邊界及相位。r6全94組1083幀通過，
48張舊binary-alpha樣本bytes相同，六項拒收／來源守衛通過；只證明候選資料，不是原版對拍或GPU驗收。

內建image_gen補繪來源並保存prompt與實檔SHA，目前22份新母稿明示供28組重用，另3份單幀。
全組候選掃描後，重審29組339張替換幀，修正錯向及第37組花形盾徽。
第19組整稿持盾側與背向錯誤拒收；其單幀只改善朝向，仍像素化。
部分舊母稿比例待校正；第76組中立左向圖顯示不等比填框會把頭盔拉寬，下一步先做框內等比例樣本。
正式catalog、私人包與Go執行期未改，HD世界仍DRAFT，#258保持未修。主收據追加alpha_core_followup_20261019，
61與56／57同步，不以資料齊備宣稱完整HD。來源仍是未整合工作樹，本輪未提交或推送候選。

971份Go及GUI來源共972檔SHA保持，沒有必要重跑未變動的Go回歸；本輪無錄影或遊戲binary。
Python遞迴擁有權掃描逾時屬檢查工具問題，改以同容器GNU find完成；work及本輪可寫目標皆1000:1000。
既有docs研究檔有root-owned檔案，本輪只讀，未做廣域chown。一次性編製／驗證容器均退出。

GitHub #201／#202／#204已同步本機候選進度與未整合限制，官方pull／render／verify維持21項、
14項人工判定、可能完成0；241條教訓與123個守衛通過。正式第一版tag保持d8c361a8，
README及四語忠實截圖未改，沒有新的玩家可見完成聲明。

## 2026-10-19 HD等比例與完整圖庫範圍勘誤

沿既有分支與手繪風格接續。六個縮放原型追到24成品的producer，確認registered ink是現代預覽輪廓，不能當原版FDICON邊界。改用22像素方格等比例、水平置中及已登錄腳底，r9既有94組1083幀資料驗證通過；新增11母稿供14組168幀，19整組取代舊像素化單幀，10及76保留明示方向修正。六頁完整方向掃描及兩頁新稿全尺寸審查已完成；未以此宣稱全部身分或正常GPU完成。

r8序列編製300秒界限留下336張，exit124且沒有完整manifest，保留為工具效能診斷。編製器改同母稿只辨識一次與有界工作程序，r9三程序完整通過；63張r7樣本及336張r8部分輸出bytes相同。正式工具另驗序列／兩程序PNG及manifest完全相同、canonical重複來源、比例、雜湊及拒收等九項守衛。14-v2預覽藍色背景的初次判斷保留勘誤：背景抽查alpha為0，不能稱整份不透明；未選v2，採用v3。

61先READY後增加HD圖源載入、原生單位計畫、前景Mask計畫與候選世界合成器。1568組單位位置及96組前景實際native寫入比較通過，含零色來源、透明洞、位移與原子拒收；候選世界固定四像素邊界並保留既有現代變暗政策。完整Game.Draw／透明UI／HUD與走行仍未接。圖源loader首輪因/pack symlink缺/repo掛載失敗，修正同一repo唯讀掛載後18.497秒通過；後續座標84.983秒、前景0.896秒、世界fixture2.721秒通過。Docker socket沙箱連線失敗未啟動測試，以同一有界隔離命令重跑。980份Go來源快照保持，標準現代主題回歸60.698秒通過，但未給外部候選env而略過該條。

外部候選實測不是綠燈：同一正常首章來源JSON還原後，94組圖源預檢完成，世界因missing group96 frame0拒收，整體回歸39.757秒失敗。沒有用跳過敵人、猜代圖組或像素替代讓它通過，也沒有候選世界圖片、GPU／原版或新E2收據。來源保存SHA仍089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。

固定B24的實際header與既有1680幀loader直接反證舊96組分母。新受版控工具核對原始size／MD5／SHA、前6 bytes及全部分離Mask：140組中125組1455槽可見，現代清冊漏31組372槽，沒有與較早完整12幀組相同的decoded組合。#259登記本缺陷，56／57／58／61及主收據同步；91歷史只追加勘誤。94組子集資料仍有效，驗證器改明示registered_catalog_complete，另明示native_bank_coverage_verified=false，不把清冊完整當全圖庫。

內建image_gen依正式DAC上色的原版96參照及核准0風格補第一份96母稿。1070×1470 RGBA與12方向已審，來源e4839b40733dd56e905acda6f9096b868147ec3bc334ba1799bddb4014a1e9d5；不推定角色／職業或未知持物。原圖、prompt與來源SHA已保存，尚未編製24父項／144圖或登錄私人正式包。下一閘門先補96及其他實際需求、清冊與loader範圍，再接世界／UI與正常GPU；HD地形與平台仍待完成。

本輪沒有錄影或遊戲binary，未變更原版檔、正式catalog、私人正式包與Release。README文化入口及四語忠實圖保持；主收據追加proportional_world_followup_20261019。

提交前官方Issue pull／render／verify為22項、15項人工判定、可能完成0；242條教訓與124個守衛通過。全遞迴Markdown檢查找到64筆既有歷史／oldtree／舊封包缺失，未誤稱整庫通過；README、00、56／57／58／61、91與91歷史共2912筆本地連結及正對照通過。980份已測Go SHA保持，work／現代素材無root-owned或.md目錄，本輪目標1000:1000；FD2容器全部退出。

HD候選層與圖庫範圍勘誤已提交2537cb0811812e13d20a5da77832857f94ba6b42並快轉整合main，遠端HEAD已核同值。#201／#202／#204／#258已綁來源，#259保留開啟並補96候選SHA。主收據補source_commit；980份Go及已驗素材保持，只有來源綁定與官方工作清單快照接續同步。

## 2026-10-19 完整原生人物圖庫與HD清冊

從main的2d6f62c8建立feat/hd-native-sprite-coverage-20261019。61先READY再修正原生140組邊界，
125組可見、15組無來源寫入與固定原始1680幀Mask逐項核對。舊無HD清冊保持相容，
含HD清冊必須完整125組及同組24父項；不再接受94組子集冒充完整。

依已核准風格與原版彩色資料參照補31份母圖及372張24父項。97首稿右向刀刃缺失拒收，
採v2；新增372張HD切幀已逐頁檢閱。r10全125組1455張來源、比例、alpha與相位驗證通過，
舊94組1083張父項及HD輸出bytes保持。正式351筆清冊保留所選母稿、單幀來源與SHA，
編製器可直接依清冊重生；拒收稿與原版參照不入私人allow-list。

首章同份普通START／F5 JSON的CPU世界候選已通過，原始保存089b077b不變。
正式清冊及私人包另驗16張正常GPU、F2／Home、狀態／命令、縮放／說明、F5／F9與新程序標題LOAD；
兩主題與來源保存bytes相同，六組面板外GPU區域零差異。這只驗既有24呈現，144世界仍未接Game.Draw。
相關Go最終回歸通過，fdicon0.341秒、cmd33.060秒；清冊六項與編製九項守衛通過。

兩次環境失敗保留：Go r2誤用FD2_ASSET_DIR，改FD2_ASSET_PACK同image r3通過；
GUI r1在暫存binary建置完成前啟動，沒有建立畫面，待完成後同image r2通過。
登錄後工具測試的隔離複本漏帶10／76單幀來源，補齊既有全域引用後r5通過；正式拒收規則不變。

私人庫維持private。3207檔逐項SHA與allow-list核對，無額外檔；提交1edc3a3b8e1c6582cceaa31a0e97ab99223851c2
已推送且遠端HEAD相同。runtime-only包3010檔，不帶母稿；正式Release與四語README截圖保持。
新主證據追加HD戰場主收據native_coverage_followup_20261019，56／57／58／61同步。
接續#258正式世界與透明介面，再擴HD地形、其他美術及早中晚期／平台驗證；完整HD仍未完成。

登錄後直接由私人包重生的九項工具守衛通過，runtime-only正式載入11.228秒通過。
242條教訓與124守衛、目前八個文件入口2916個本地連結及新工具索引正對照通過。
work與現代素材無root-owned或.md目錄，全部本輪FD2容器退出；已驗暫存binary與runtime-only重複包移除，
保留manifest、雜湊與可重跑工具。沒有新增錄影，沒有改動正式Release。

完整圖庫來源d8d8df214916d9a1523afdffc271ba88e5bd2d26已快轉整合main並核對origin/main。
#259依資料／loader／既有GUI／私人同步驗收結案；#201／#202／#204／#258已更新目前範圍及來源，仍保持開啟。
官方pull後為21個未完成項，主收據綁定來源；後續提交只同步遠端工作清單與證據，不改已驗程式或素材。

## 2026-10-19 正式HD戰場與戰鬥後F9

144人物已接入1920世界，與原生幀共用一次時鐘／DAC交易；地形種子、透明UI與GPU人物分層，缺件拒收不發布原生狀態。普通／第12章上移caller共用定位與terrain觀測，索引輸出保持。

正常標題LOAD既有START/F5首章槽，34張GPU、4張實際走行、指令環、取消、F2、兩次普通攻擊、狀態／命令、縮放／說明、保存及新程序LOAD通過。兩份穩定人物來源各10678核心像素零差異，一份跨相位不可比；同view的960地形樣本零差異。完整保存SHA保持089b077b，沒有錄影。

r4/r5發現#260：F9載入新非原生章首對話，舊經驗owner先攔輸入且原生等鍵條件不成立。較早把移動計畫稱為攔截原因並不充分；r5唯讀owner欄位才確認因果。有效JSON驗證後丟棄經驗owner／待播佇列與移動計畫，不跑舊續行；拒收保持。修正前回歸失敗，修正後成功／拒收通過，GUI亦越過卡點。

r2/r3為驗證工具未等章首對話及固定Z次數錯誤；r6混用讀檔前後不同鏡頭，改為同view比較後同一遊戲來源與輸入r7通過。首次完整測試僅字串清冊綁定過期，99項人工處置以精確舊SHA清冊遷移，其他套件通過；最後完整測試以隔離/tmp乾淨重跑通過，主程式282.386秒，所有套件通過。

現況、來源綁定與重跑命令沿[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的world_runtime_followup_20261019與[61](docs/knowledge-base/61-modern-theme-runtime-spec.md)。地形仍為原版放大，完整HUD、經驗／死亡／法術／故事演出、早中晚期與平台另驗。私人素材及Release保持，沒有新原版oracle、matching或整章PLAYER-E2。提交、Issue與清理結果於本批最後追加。

完整final回歸的後段全部因TempDir建立時空間不足失敗，沒有其他具體斷言錯誤；磁碟剩1.4GB、inode充足。改用有界3GB記憶體/tmp與8GB容器記憶體，保持同一來源、素材、seed條件及測試命令重跑，不清理其他專案。路由未命中額外Docker支線，沿既有規格與AGENTS的隔離驗證契約。

有界tmpfs先明示exec並以清冊套件驗證可執行，最後全套通過。五份GUI暫存執行檔與本任務Go快取已清除，共釋出957156211 bytes；來源、雜湊、PNG與測試收據保留，沒有錄影。

來源提交`3e1abc519fd6a125ea8854db24eaaa0c0b10b967`已快轉整合main、推送並核對真正遠端HEAD相同。#258與#260依提交及驗收結案，#201／#202／#204已改寫目前HD範圍與下一閘門。官方pull／render／verify為20條開啟、13條人工、可能已完成0；243條教訓及125守衛通過。主要文件連結2924條無缺件，986份來源綁定保持，新程式UID/GID1000，輸出無root-owned或.md目錄，無本批FD2容器殘留。Release與私人素材保持；本次後續提交只同步現況及工作清單。完整HD仍未完成。

## 2026-10-19 計時戰場對話保留HD世界

穩定經驗文字原已沿HD黑體，缺口是完整indexed開收框覆蓋HD人物。新增來源格網覆蓋與FFEC計時對話圖層，首次HD Draw保存世界／UI，原生時鐘照常、忠實路徑保持；F9與對話結束隨處理器丟棄。沒有新美術、規則、格式或翻譯。

CPU完整來源格網、零色、背景快照與原子拒收通過。FFEE文字窗口超出格網，因此僅接已證實FFEC，不猜其餘caller。相關現代／原生對話／升級／存讀檔68.504秒與三資料套件通過，未重跑整套Go。章前只把亞雷斯EXP0改99，實際普通攻擊升Lv2／EXP30；37張主要GPU、6張對話、4張走行、F2與來源／兩主題保存bytes相同、新程序LOAD通過。獨立框角驗證2張開框與2張收框，1張跨階段不可比；HD來源8296核心像素零差異。

r1工具誤等第二次攻擊的經驗，改驗固定第一次，不改seed。r2首張開框仍是前段indexed表面，audit不等於GPU fence。r3等待也套在短收框，因而漏擷取；改只等開框交界，先PPM後讀狀態再PNG。r4全部通過；實際框角分級不要求audit索引與畫面逐幀相同。原始失敗與一張不可比樣本均保留。

[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的timed_dialogue_followup_20261019保存來源／衍生槽SHA、唯一差異、來源綁定、測試、GPU與重跑命令。完整HD、地形、其他演出與平台仍待完成；沒有原版oracle、matching或整章E2。私人素材及Release保持，沒有錄影。提交與清理結果於本批最後追加。

提交前官方pull／render／verify為20項開啟、13項人工、可能已完成0。244條教訓、126守衛與2934本地連結通過，991份來源綁定保持；新輸出UID/GID1000，無root-owned與.md目錄。兩份暫存執行檔、PPM及本任務快取已清除，釋出293281361 bytes，沒有錄影與本批容器殘留。

來源提交`b086b2c3db01bfb52673e3d9180b6f1d2233aff0`已從獨立分支快轉main，推送後真正遠端HEAD核同值。#201／#202／#204已同步計時對話成果與尚未驗證caller，三項保持開啟。官方快照再次pull／render／verify為20項、13人工、可能已完成0；後續提交僅同步來源及工單。完整HD仍未完成。

## 2026-10-19 大型戰鬥動畫HD來源與離線候選

使用者將目前Goal改為完成戰鬥動畫HD。從已合併的`b676984d`建立
`hd-battle-animation`，先處理蓋亞；頭像、FDICON、地形、私人包與正式Release保持。
路由命中復古重製、規格閘門、文件職責與戰術幾何；生成沿imagegen技能及已核准手繪造型，
沒有開啟全面matching、硬體時序研究或子代理程式。

新增圖源盤點工具，409筆FIGANI的5355份登錄資產逐檔SHA核對；其中264筆可播放、
2118幀、144筆零標頭，以及408號零長度資源。全部raw依既有LLLLLL容器契約串流重組，
size／MD5／SHA-256與固定FIGANI.DAT相同。既有Go原始解碼器再與全部分離PNG／mask／
描述子逐幀比較，264筆2118幀相同，5.86秒。這是資料來源證據，不是新的原版執行收據。
舊metadata的4236個frame_NNN資產ID與manifest檔名ID不同；正式loader用path與mask_path，
工具沿同一路徑核對本資源allow-list及實際雜湊，保留別名計數，不改原始包。

61先列READY-CANDIDATE，新增未接Game.Draw的1920離線合成器。72組原生參照涵蓋兩側、
RawByte7層序、受擊位移、裁切、index0不透明與填色33；另驗HD細節、coverage alpha與錯尺寸拒收。
蓋亞91號第7幀候選編為906×876，以原生X129/Y46放入1920×1200；全部不透明像素均核對原值，
沒有重定世界位置。候選圖未畫白黃閃光，不能當完整第7幀或15幀動畫已完成。
相關候選與物理回歸12項PASS、24項因未明示正式pack／oracle／GPU條件而SKIP，10.034秒；
不將跳過項目、離線PNG或單元測試提升為GUI／PLAYER-E2。

三姿勢r1/r2的出拳後頭部朝向錯誤，維持拒收。r2提示錯把所有姿勢都要求左朝；放大原版後
更正為待機朝右、出拳後朝左。r3改單姿勢，頭部方向已修正，只保留作位置候選。
r4單張待機的屈臂左右關係仍不符，沒有登錄。四份原圖、完整提示、參照、SHA與拒收原因均
留在既有工作目錄並由主收據索引，沒有把拒收稿同步私人庫，也未新增公開美術。

首次來源拒收測試漏連fixture的images，補齊既有目錄後三項通過，9.036秒；錯固定來源、
錯PNG雜湊、raw竄改且重簽manifest都拒收。首次Go實體候選測試漏掛既有/repo來源連結，
補掛載後同一來源與命令乾淨重跑通過。均屬測試環境問題，不當成遊戲缺陷。

主證據為[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的
`battle_animation_followup_20261019`；重跑工具、原始解碼測試與候選編製均掛入
[61](docs/knowledge-base/61-modern-theme-runtime-spec.md)。[58](docs/knowledge-base/58-fd2-exe-re-coverage.md)
區分來源已驗與正式動畫未接。已依官方formatter新增[工作項#263](https://github.com/wicanr2/fd2_re/issues/263)，
GitHub由真正主機gh操作，快照由Docker內官方函式產生。既有20項不結案。

本批沒有錄影、暫存遊戲執行檔、私人同步或發行操作。Go快取只存在有界容器tmpfs並隨退出清除。
目前僅候選及資料驗證，保留於工作分支，尚未提交或推送；正式待機／攻擊、共同效果、反擊、
入場／返回、清冊與普通玩家路徑仍待接續。工作清單、連結、擁有權及容器清理於本節最後補記。

本批最後官方快照／render／verify為21項開啟、14項人工、可能已完成0；250條教訓與131守衛通過。
830個本地連結、六份來源綁定及12份修改檔擁有權核對通過；候選沒有正式呼叫者，清冊SHA保持。
工作根無root-owned或.md目錄，素材／Go容器均已退出且移除，私人工作樹保持乾淨。
先前70秒工作清單批次在render之後逾時，改用240秒有界命令完成官方驗證；不當成產品缺陷。

## 2026-10-19 蓋亞完整戰鬥候選與按場預載

接續`hd-battle-animation`，補上屈臂方向正確的待機與蓄力、大型命中閃光及拳端小閃光，
搭配既有收勢母稿。受版控編製工具依原始描述子產生完整4／11幀、逐幀配方與367項隔離清冊。
明示重用五張母稿，不聲稱15張獨立重畫或原版逐幀一致。正式365項清冊及私人保存庫保持。
四份新母稿、完整提示、輸入／輸出SHA及拒收窄站姿均由主收據索引，沒有公開新美術。

61先補可選來源READY契約，再新增`battle_animation_hd_set`載入及Python清冊驗證。
原始metadata SHA、header及每幀座標／尺寸／delay／raw欄位逐項核對；錯來源、缺幀、錯PNG、
重複資源、越界路徑及alpha錯誤拒收，不發布部分資源。原清冊不存在此role時保持既有路徑。
首輪候選因`/out`別名改變相對連結根而失敗；改讀相同候選的標準`/repo`路徑後通過，
這是驗證環境問題，沒有放寬圖源檢查。

已驗原始矩形合計47,769,069像素，六倍RGBA為6,878,745,936 bytes；全畫布方案更達
19,519,488,000 bytes。主題預檢逐檔驗證後只保留路徑、指紋與描述子；在物理結算前原子預載
攻守方資源，全部成功才發布並釋放其他常駐圖。失敗不發布新圖或釋放舊圖。這次只解決
常駐記憶體，不宣稱完整圖庫啟動效能已驗；初始alpha預檢仍逐張解碼。

最終Go回歸10個頂層測試、含子項20項通過，無SKIP，30.521秒。合法分離包的fixture只把
BattleFig換成30以命中蓋亞圖源；玩家confirm及AI mode11兩條缺件路徑都保持HP、方向、
原生亂數與Go亂數，且未發布部分圖源。既有多擊、反擊、狀態／暴擊閃光及缺聲音的交易停止
回歸也通過。另有11項清冊測試通過，20.224秒。沒有把fixture當正常玩家路徑或HD GPU收據。

整組排版顯示：較寬待機圖縮進舊框後會比收勢矮，造成大小跳動。嘗試較窄站姿的雙圖改變了
核准頭盔且加入光暈，拒收。受版控預覽工具以同一未修改母稿比較A舊外框與B原版身高／腳底，
B會超出舊框。已依grilling技能展示prototype並請使用者選擇，目前仍待回答；沒有將B寫入
正式幾何或素材清冊。正式HD Draw、入場／返回、其餘FIGANI與FDOTHER效果、正常GUI、
保存冷讀及私人同步均未完成，Goal範圍保持。

完整證據沿主收據`battle_animation_bank_followup_20261019`，58／61及遠端#263同步。
本批未提交或推送，沒有錄影、Release或私人同步。最後清單、連結與清理結果另補記。

本批最後官方工作清單21項、14項人工、可能已完成0；250條教訓與131守衛通過。
834個本地連結、八份來源綁定及14份修改檔擁有權核對通過，正式清冊SHA保持。
工作根無root-owned或.md目錄；素材與Go容器均由--rm退出清理，沒有新錄影。
排版問題仍等待使用者回答，Goal保持active，不將等待當成批准。

## 2026-10-19 共背景物理戰鬥GPU與正常鍵盤診斷

接續同一工作分支，先完成不依賴人物占寬選擇的可選繪製層。
沿61既有原生幾何與主證據的共背景物理契約，將入場、主動／反擊、受擊填色、restore及
64階戰鬥DAC淡出接到1920世界。重建地圖仍沿原有65階淡入，最後續行一次。
沒有修改job、節拍、音效、HP、RNG、存檔格式或原版像素模式。
來源只在結算前或F2發布前預載；GPU快取依scene及theme釋放，不在Draw補讀。
其他背景捲動演出仍整場沿原生呈現，沒有把尚未支援的轉場混入HD。

170個GPU案例最終回讀通過，23.49秒。原生整張1920畫布與CPU合成相同；
HD的alpha、RGB減色、填色及裁切允許1階量化誤差。實際Game.Draw與最終1920呈現的
生命週期另以診斷原版圖通過，32.44秒，涵蓋全部present、caller wait、64階戰鬥淡出與
65階地圖淡入、F2、壞來源拒收、HP與兩種RNG保持及貼圖釋放。
最初測試誤把兩段DAC合成一段，又假設Update與Draw一對一而重複切主題。
依既有CONFORMED返回證據與固定Ebiten版本修正測試，不改產品時序。

新增受版控診斷編製器，把資源12／13的19幀原版索引像素依原始DAC與mask最近鄰放大六倍。
正式清冊及蓋亞母稿保持。首輪把盤點中的PNG檔案hash錯拿解碼buffer比較，在輸出建立前拒收；
依原契約修正後通過。診斷包只留本機，不能計為新HD美術或原版oracle。
GUI工具增加明示的battle-hd門檻，要求兩次普通攻擊真正使用可選戰鬥圖層。
正常標題LOAD既有START／F5保存、走行、指令環、取消、兩次攻擊與返回、Home、F2、
F5／F9及新程序標題LOAD全部通過，200.665秒，沒有座標或狀態注入。
來源保存、兩主題保存及冷讀的完整bytes一致，兩次攻擊至返回各13.085與18.787秒。
這是軟體GPU與擷取期間的觀測，沒有當成正式平台效能或音訊人耳驗收。
首輪在60秒返回等待逾時；同一binary、保存及圖包以有界180秒觀測乾淨重跑通過。
首輪與同時進行的廣泛回歸負載一併保留，不以調整遊戲節拍消除失敗。

廣泛工作樹回歸因原始遊戲檔觸發舊loadGame測試，長時間解碼原有OGG；堆疊停在
TestNativeShopTransferShotLoadGameAdmission的decodeSeparatedSoundBank，音訊程式本輪未改。
為定位而中斷該次回歸，不能列為全專案通過。另一次僅複製Git檔案的乾淨樹漏掉既有
生成素材，造成地圖、字模及原始DAT缺件，該次同樣不作產品缺陷或通過收據。
最後以完整工作樹及合法素材執行本次相關回歸，結果另補記。

字串審查的全域hash因程式行號變動失效。基底Git工作樹還缺少既有忽略檔command_labels.json
的38筆資料；補入同一份生成資料後，舊盤點SHA與現行審查完全相同。
再由既有遷移器搬移座標，99筆分類與理由保持，沒有將新診斷詞硬塞進語言包。
readonly巢狀掛載無法建立新檔的失敗保留，改在可丟棄工作樹補輸入後唯讀使用。

本批來源、收據、失敗分類及重跑入口沿主收據的`battle_animation_gpu_followup_20261019`，
58／57與61同步。蓋亞占寬選擇仍待回答，沒有採用B或登錄新動畫美術。
完整HD Goal保持active；其餘FIGANI、法術、死亡及共同效果、正式素材與私人同步仍未完成。
本批沒有錄影、Release、私人同步或新的原版oracle。最後相關回歸、工單及清理結果另補記。

最後相關Go回歸119個測試／子項通過，19個專用條件項跳過，52.942秒；GPU兩項已分開執行通過，
不把其他跳過項目算通過。字串審查及盤點回歸通過，1.476秒；清冊11項通過，11.757秒。
這是本次相關回歸，前述兩次廣泛測試仍保留為未完成／輸入缺件，沒有宣稱全專案綠燈。
官方#263內文與21項快照已更新，14項人工判定、可能已完成0；252條教訓與133守衛通過。
新守衛最初指向被搜尋器排除的測試檔，改綁正式來源標記與官方遷移器後全部通過，未修改搜尋器。
1271個本地連結、14份來源綁定及36份修改檔擁有權通過，正式清冊SHA保持。
驗證執行檔19,227,648 bytes與臨時基底工作樹已刪除，沒有錄影及FD2容器殘留，私人庫乾淨。
本批形成可選正式consumer到普通玩家保存冷讀的垂直切片，提交範圍不包含待決人物幾何或正式新美術。

## 2026-10-19 共同召喚GPU與熾天使HD五幀

從main的232921a1建立hd-battle-effects，沿復古重製、規格閘門與文件職責路由。
蓋亞占寬A／B仍待使用者回答，本批不採用B、不登錄FIGANI人物候選。
共同召喚沿既有0x27FC9證據，沒有新增RE、matching或硬體時序研究。

FDOTHER全部104筆raw重新組回固定原版DAT，四種召喚37幀的PNG、mask及descriptor核同。
先在61列READY，再接slide／main／tail可選GPU，FIGANI與FDOTHER數字ID分開。
原生delay、hold、音效、HP／MP／兩種RNG與一次續行保持。HD尾段採RGB混色的現代近似，
原生路徑保持真DAC。結算及F2發布前原子預載，Draw不補讀；施術者及地圖尾段保持原生。

依原版後視金髮、雙白翼、白袍及金飄帶，以內建image_gen生成身體、電光材質及近身電光三母稿。
原生身體前四幀25827像素相同，最後幀下移1像素並裁掉11像素，因此身體只縮放一次，FX另編製。
明示三母稿重定向五幀，不稱五張獨立繪畫。r1羽毛與原FX間留空，r2加近身電光後通過功能測試，
但深色像素邊過重；r3改新RGB材質與0.65 native像素柔邊，原生幾何及時序保持。
充電母稿alpha最高254保持原值，成品由身體提供255核心，沒有放寬正式載入器。

273個GPU回讀通過21.514秒，原生全1920畫布相同，HD量化誤差上限1。
候選r3完整RunGame／Game.Draw／最終合成器通過48.095秒，158個原生present、41次HD及1次原版Draw，
保存15張GPU圖片。正式366項清冊及正式圖源再獨立通過32.996秒，158 presents、39次HD及1次原版Draw。
兩次均驗F2、缺件原子拒收、HP／MP／RNG發布邊界、一次續行及貼圖釋放，沒有縮短delay或hold。
此為既有class19／BattleFig4修改fixture，只列RUNTIME-E1，普通召喚、自然存檔及PLAYER-E2仍待驗。

相關回歸55個頂層PASS、17個條件SKIP，7.266秒；既有物理完整Draw及64／65階DAC返回另通過11.775秒。
原始四資源37幀回讀通過，Python清冊11項通過4.790秒；字串盤點由官方遷移器重綁，
99筆分類與理由保持，相關回歸通過0.363秒。本輪未重跑全專案測試，不把條件SKIP列通過。
開發容器最後外層有界逾時124，但獨立命令均已明示result0及PASS，屬容器生命週期結果。
彙整證據時曾以PASS字樣判斷清冊日誌，實際成功格式是validated 366，讀取實檔後修正彙整條件；
首次彙整在寫檔前停止，沒有修改產品或原驗證結果。

正式清冊由365變366，舊365項完整保持。只新增熾天使FDOTHER65五幀，其他三召喚仍用原版。
五幀及三母稿依allow-list同步私人庫，舊3268份逐筆保持；新3276份及runtime-only3068份逐檔核對
SHA、size及UID，母稿不進執行包。私人提交3914bab035acd5e1ebee846b5c7511ada6b81524已推送，
遠端HEAD相同且repository仍private。README新增960×600最終GPU總覽並明示fixture，文化保存內容保持。

主證據為[HD戰場主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的battle_effects_followup_20261019，
包含完整三份提示、輸入／輸出SHA、編製配方、失敗分類及重跑命令，58／57／61同步。
FIGANI人物、其餘法術／死亡、正常召喚及保存冷讀、代表性效能／平台仍待完成，Goal保持active。
沒有錄影、暫存遊戲執行檔或發行操作；Go快取僅在容器tmpfs並已消失。最後工單及清理結果另補記。

最後官方工作清單21項、14項人工、可能已完成0；254條教訓與135守衛通過。
新增archive命名空間及固定身體／獨立FX編製規則，均附正式來源守衛。
1385個本地連結、18份來源指紋與正式／私人清冊核對通過；舊365項清冊保持。
輸出UID/GID1000，工作根無root-owned或.md目錄；沒有本批FD2容器殘留。
#263更新進度但保持開啟，未改寫完整Goal驗收條件。

## 2026-10-19 補齊四種共同召喚HD與跨段Draw確認

上一輪eb257fe6已推送，屬實質進展。從main建立hd-summon-fairy，沿復古重製、規格閘門、
文件職責與圖像技能接續。蓋亞占寬仍待回答；沒有採用未核准幾何，也沒有啟動matching或新RE。

先核對FDOTHER66的14幀：frame0固定身體，1～3是相同手勢差分，4／5另有手勢，
6～13未被索引200風球覆蓋的像素沿frame5。以身體、抬手、伸手與風材質四母稿編製，
主要手臂與遠端小手各自對齊原始可見框，4／5明示共用伸手母稿，不稱14張獨立繪畫。
母稿由內建image_gen依原版參照及核准手繪風格產生，四翼、後視長藍髮與藍金衣著保持。
第一次把世界320寬誤當手勢跨度，修正後再發現淡alpha橋接；回查既有核心128／柔邊3契約，
只用核心分離部位，選中輪廓的原覆蓋保持，母稿不改成二值alpha。

再依原版造型生成破壞神與暗邪鬼身體，各配一份紅色及綠色FX材質。破壞神其餘八幀只有索引32，
固定騎士與球體保持原層序。暗邪鬼前八幀15649個身體像素相同，最後幀下移1像素，
15647個可見像素相同、兩像素裁出底界。各以兩母稿編製九幀，身體只縮放一次。
初版破壞神11×11小球加Gaussian後角落最低alpha1，正式Go載入拒收；直接Lanczos重取樣原遮罩，
保留332個透明輸出像素，最後真正全畫面不透明原幀仍保持255。未放寬載入器，r1拒收候選保留。

風妖精完整RunGame／Game.Draw／最終GPU合成通過156次原生呈現，44.734秒；
F2、最後一幀缺件原子拒收、HP／MP／原生及Go亂數發布邊界、一次續行與貼圖釋放均通過。
同時重跑熾天使158次呈現，27.555秒。原delay與hold保持，不把修改class19／BattleFig4的fixture列為PLAYER-E2。

破壞神及暗邪鬼的完整RunGame發現原有owner缺陷：中間hold期間Draw留下drawn=true，
hold結束切到stageEnd卻沿用舊確認，多次Update可能跳過新的首張畫面。實際分別漏152及164。
以官方formatter建立#264，只在兩個owner換frame處清除drawn，要求新的Draw確認；
原等待值、HP／MP、三段發布、兩種RNG與續行保持。新增交界回歸與多Update的完整RunGame。
修正後破壞神241次呈現通過109.470秒，暗邪鬼255次通過139.275秒，三段發布及hold全部驗到，
F2、缺件原子拒收與一次續行通過。這是原生呈現排程實作修正，沒有宣稱新增原版oracle。

Go登入shell覆寫PATH及/tmp tmpfs禁止執行各造成一次環境失敗，沿同一維護中映像檔修正，
快取只放專用tmpfs。獨立測試二進位首次從module根執行，原素材相對路徑不符而SKIP，
沒有計為通過；改到Go原本的package工作目錄，用同一二進位重跑。清冊回歸曾誤用不存在的檔名，
改用已核對的tools/test_modern_theme_catalog.py，不改產品。正式抽驗與最後回歸結果另補記。

正式清冊由366增至369，舊366項逐筆保持。新增32幀與八母稿，四種共同召喚合計37幀。
依allow-list同步私人庫，3316份及runtime-only3100份逐檔核對SHA／size／UID，舊3276份完整保持，
八母稿排除於執行包。私人提交abb438556d4a516b6dd87eb7b4da56c04ddedb06已推送，
遠端HEAD相同且仍private。README只用四個已通過fixture的GPU縮圖總覽，原始完整母稿留私人包。

統計勘誤：上一輪所寫55個「頂層」實為35個頂層加20子項，17個SKIP及原始結果不變。
本輪修正前相關回歸為39頂層加20子項、17個條件SKIP，74.401秒；修正後結果另補記。
字串盤點及99筆審查在兩行正式修正後仍通過，八項測試1.627秒，不需搬移或改分類。

完整來源、八份提示、編製配方、失敗收據與重跑入口沿[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)
的battle_summon_bank_followup_20261019，58／57／61同步。FIGANI人物與施術者、其他法術／死亡、
非共背景、普通召喚／保存冷讀及代表性效能／平台仍未完成，#263與完整Goal保持開啟。
本批沒有錄影或Release操作；暫存驗證二進位在正式抽驗後刪除，最後工單及清理另補記。

正式369項清冊及正式圖源的獨立抽驗再通過command35完整255次呈現，70.64秒。
修正後相關回歸40個頂層加22子項通過，17個專用條件項SKIP；未宣稱全專案通過。
統計按Go完整結果標記及測試名稱的子項斜線分類，避免ALSA stderr接在行首漏算一個頂層PASS。
清冊11項及256條教訓／137守衛通過。26,162,128 bytes暫存測試二進位已刪除，
Go快取隨有界容器退出消失；沒有錄影。最後工作清單、提交與清理結果另補。

提交前官方快照／render／verify為22項開啟、15項人工、可能已完成0，含待提交結案的#264。
1402個本地連結與15份來源指紋核對通過，輸出UID/GID1000，工作根無root-owned或.md目錄。
已刪runtime-only驗證複本3102檔，釋出132308334 bytes；manifest保留，可由正式清冊及私人完整包重生。

來源提交adf79819d50db78afc30e96bc54b4dc86d3308e3完成後，#264依實際提交與完整驗證結案。
第一次open查詢仍包含state=CLOSED的264，直接核對Issue已關閉後再拉清單；不以列表成員代替state。
官方快照／render／verify回到21項開啟、14項人工、可能已完成0，#263保持開啟，1402個本地連結通過。
本輪六份使用中的原始輸入事後size／MD5／SHA與固定清單相同。
最後交付只同步來源指標及工單；主線推送與遠端HEAD於提交後核對。

## 2026-10-19 亞雷斯物理戰鬥HD與普通玩家抽驗

從main的8f9b73e6建立hd-physical-characters。沿復古remake規格閘門及imagegen技能，
只補身分4的FIGANI12／13，保留蓋亞加寬待決分支。原版metadata、PNG檔案雜湊與描述子
先核對；FD2.EXE及FIGANI.DAT事後size／MD5／SHA也與固定來源相同。

15份母稿編製完整4／15幀，採內建image_gen；全部提示、輸入、接受與拒收版本存入主收據。
初版5／6把雙手放到肩部而拒收；收勢腿距過寬另補窄身，末幀沿前幀姿勢只移除殘光。
初始藍色錨點誤納武器冷色反光，改用本批明示衣色與連通核心。邊界拒收後回讀規格，
收勢四幀共用0.9217171717171717倍率內縮，中心及腳底保持，不放寬正式loader或原矩形。
部分旋轉武器與光效仍沿原描述子裁切，這批是美化重定向，沒有宣稱原版逐幀視覺一致。
原稿alpha最大254，alpha0下的RGB不會顯示；實際合成乾淨，正常LANCZOS成品具255核心，
沒有為此修改透明度。另一次乾淨編製的19張PNG及來源配方完全相同。

普通START／F5取得的首章槽SHA為089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。
正常標題LOAD、移動、指令環／取消、兩次攻擊與返回、經驗對話中F9、F2、Home、保存及
新程序標題LOAD通過。34張主要GPU、115張戰鬥PNG與4張走行圖保留；完整保存bytes與來源相同。
89張戰鬥PNG的藍衣高覆蓋像素符合新增圖源，共114筆相容紀錄。相近身體保留候選集合，
不宣稱唯一相位或武器／光效逐像素通過；武器、光效與完整19幀另經目視審查。
錯收據、錯圖源與舊像素戰鬥擷圖三種負向fixture均拒收且不發布報告。

先以候選通過10個Go頂層加8子項，2個條件SKIP。正式371項清冊與私人包再驗11頂層加8子項，
1項蓋亞預覽SKIP，未跑全專案。清冊11項及256條教訓／137守衛通過。
沒有改正式Go規則、RNG、傷害、音效標記、保存格式或原版oracle。軟體GPU與PNG擷取的
牆鐘時間不當作正式平台效能；代表性效能／平台仍待驗。

原369項保持，新增19張成品及15母稿同步私人庫，完整3350檔及runtime-only3119檔逐筆核對，
舊3316檔保持；15母稿不進執行包。私人提交91c5751e53562d10465a0addde9aab04e243b79a已推送，
遠端HEAD相同，未改私人庫可見性設定。README只新增兩張實際普通攻擊視窗的縮圖總覽。

主收據新增ares_battle_followup_20261019，現況58、介面57、契約61及#263同步。
完整戰鬥HD Goal與#263仍開啟，其餘人物、施術者、非共背景、其他法術／死亡、普通召喚及平台未完成。
本輪沒有錄影或Release操作。已驗暫存二進位刪除，候選重複圖改成同hash的allow-list連結；
runtime-only驗證複本只在容器tmpfs，退出即釋放。官方工作清單與最終清理結果見主收據。

官方工作清單為21項開啟、14項人工、可能已完成0，#263保持開啟。
暫存程式／測試執行檔與候選重複圖共釋出111179528 bytes，圖源SHA及可重跑候選路徑保持。
全版控Markdown掃描另發現四份未修改歷史文件的九個項目：八個已清理的舊dist-all產物，
以及一個程式記法被正則誤判成連結。這些不屬本批新入口，不復原已淘汰的歷史交付包。
本輪修改文件的連結與擁有權結果另存主收據，沒有把全庫舊缺件寫成全部通過。

最後核對本輪修改文件1410個本地連結全部有效。全版控共4186個連結，前述九個舊項保留為限制。
全工作樹rglob衛生掃描達90秒界限後停止，改查本批所有變更檔、ares／figani輸出根及34份正式新圖，
共7172個路徑，UID／GID均1000，沒有root-owned檔案或.md目錄；不宣稱掃完全部歷史工作樹。

## 2026-10-19 首章強盜戰鬥HD與雙方普通攻防

從main 0e83dad0建立hd-ch01-bandit。沿復古重製規格閘門及imagegen技能，
只補figure96的FIGANI288待機4幀與289攻擊8幀；290原始header零，沒有新增動作。
原版紅頭巾、灰綠單肩甲、褐衣、前垂布、藍褲及彎刀保留。
內建image_gen產生八母稿，提示、參照、原稿路徑與SHA存入主收據。

通用配方工具先核盤點、manifest、原始metadata／PNG及描述子，再沿明示身體錨點
等比例編製。修正原幀289/5把紅色光點誤認頭巾，以及兩份新稿把飄帶當頭頂的定位。
原版三個收勢尾幀仍有27／17個非金色像素差異，分別編稿，不宣稱身體完全重用。
四個縮放群組倍率均1.0。維持原矩形，部分武器／光效裁切列為美化限制，蓋亞加寬仍待決。
乾淨重建十二幀與八母稿相同，舊371項逐筆保持。十一項編製拒收通過。

普通START／F5來源槽的標題LOAD、兩次攻防與返回、經驗對話F9、戰場F2、Home、
F5及新程序LOAD通過。保存SHA保持089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。
先F9回來源再F5，不能用相同bytes聲稱受傷後HP持續。沒有章內注入或原版oracle。
保留34張主要GPU、102張戰鬥PNG、4張走行圖；108筆鍵盤操作，擷取耗時457.604秒，
不當作平台效能。70張PNG符合強盜藍褲核心；兩場20／44張同圖同時符合雙方HD核心。
相近圖源只保留相容集合，不唯一判定相位；武器／光效另目視審查。
錯收據、錯人物、錯圖源、舊像素圖及不透明FIGANI五種負向fixture均拒收。

正式373項清冊與新版私人包再驗Go 11頂層加8子項，蓋亞預覽一項條件SKIP。
清冊11項與257條教訓／138守衛通過。只有Go測試工具改為讀實際原幀數，正式Go未改。
新增全來源資料稽核：409位置、264已解碼資源、2118原幀，正式FIGANI四資源31幀，
FDOTHER四效果37幀；408來源仍阻擋。這是資料覆蓋，不能推定caller或普通玩家覆蓋。
稽核初稿誤拒既有全幅不透明召喚。回查正式loader的原生Mask契約後只修稽核，
另驗不透明FIGANI仍拒收，沒有放寬正式人物透明度契約。

十二成品與八母稿已推送私人提交54318cd97ee5d9e0ca054456783b2b10c7620a8d，遠端HEAD相同。
完整3370檔與runtime-only3131檔逐筆核對，舊3350檔保持，八母稿不進執行包。
沒有更改可見性設定，也沒有把先前private狀態冒稱本輪重新查詢。
README新增兩張本批正常雙方攻防的實際視窗縮圖，較早亞雷斯總覽保留。

主收據新增bandit_battle_followup_20261019，現況58、介面57、契約61與#263同步。
完整戰鬥HD Goal與#263仍開啟。其餘FIGANI、施術者、法術／死亡／共同效果、
非共背景、普通召喚保存與效能／平台另驗。本輪沒有錄影或Release操作。
已驗二進位與58份相同hash的候選圖複本共清出93831858 bytes；兩份不同的較早定位原型保留。
runtime-only複本僅在容器tmpfs，退出即釋放。

本批修改文件1416個本地連結通過，README新圖及61新工具入口作正對照。
本批變更、bandit輸出根與20份正式新圖共10759路徑，UID／GID均1000，沒有.md目錄。
全庫舊九項連結限制沿前批紀錄，沒有宣稱掃過全部歷史work。
工作清單render成功後verify與前者共用60秒界限逾時，拆成同容器映像的單獨驗證，
不把驗證腳本時間界限當產品缺陷；最終結果另見主收據。

拆開後的官方verify通過：21項開啟、14項人工、可能已完成0；render與verify原始log及SHA存入主收據。

## 2026-10-19 索爾戰鬥HD與正常END接敵

從main a9026731建立hd-thor-battle。沿復古重製規格閘門與imagegen技能，
補BattleFig0的FIGANI0待機4幀及1攻擊11幀，2為原始空header。保留原版灰綠厚肩甲、
藍衣／護腿、紅腰布、背向視角與單手劍。七份母稿均由內建image_gen產生，完整提示及參照存入主收據。
待機與五拍蓄力明示共享母稿；斬擊及四格收勢各有新稿，不宣稱原版細小頭帶／手臂逐格相同。
原始描述子、raw標記及節拍保持，三個縮放群組均1.0。部分衣緣／劍尖沿舊矩形裁切，蓋亞加寬仍待決。
乾淨重建15幀及七母稿相同，十一種編製錯配方均在建立候選前拒收。

普通首章原槽仍為SHA089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2。
第一輪選索爾時正式可達集合沒有相鄰敵格，保留失敗，沒有改移動力或位置。
按既有正常系統環Down→END→確認，一次換手由回合1至2，兩輪接敵後我方HP均為42／28／31／50。
索爾正常接戰後仍會受傷，最後攻防返回我方HP31／28／31／50，不把接敵時42HP寫成鎖血。
兩次攻防與返回、經驗對話F9、F2、Home、F5及新程序標題LOAD通過。
最後先F9回來源再F5，保存bytes相同不證明受傷後HP持續。沒有章內注入或原版oracle。

保留38張主要GPU、100張戰鬥PNG、4張走行，126筆鍵盤操作，752.095秒只是軟體擷取耗時。
57張索爾與58張強盜PNG符合圖源核心，兩場33／16張同圖同時符合雙方。
相近圖源只列相容集合，不唯一判相位或武器／FX逐像素通過。README另增實際索爾收勢／敵方反擊兩格圖。
新受版控GPU拒收工具驗錯收據、錯人物、錯圖源與舊實際PNG，四種均拒收且不發布報告。
一次診斷讀取把標題狀態當戰場而KeyError units，修診斷後同一GUI工作繼續；不是產品缺陷。

正式375項清冊及私人包Go為11頂層加8子項通過，一項蓋亞預覽條件SKIP。清冊11項、258條教訓／139守衛通過。
正式Go的488份來源與前批快照逐檔相同，沒有改規則、保存格式或測試Go；GUI只新增正常選人與有界END操作。
全來源資料盤點409位置、264已解碼、2118原幀，正式FIGANI六資源46幀與FDOTHER四效果37幀。
408來源仍阻擋；資料覆蓋不推定caller或普通玩家覆蓋。

十五成品與七母稿已推送私人b64bc057f76afcb4fe320d9ffccc279f0dae0729，遠端HEAD相同。
完整3392檔及runtime-only3146檔逐筆核對，舊3370檔保持，七母稿不進執行包。
沒有更改可見性設定，也未宣稱重新查詢live private狀態。主收據新增thor_battle_followup_20261019，
現況58、介面57、契約61及#263同步。#263內文改為現況摘要，舊數據與勘誤仍在歷史主收據及留言。
完整戰鬥HD Goal與#263保持開啟，其餘人物、施術者、法術／死亡／共同效果、非共背景、普通召喚及效能／平台另驗。

本輪沒有錄影或Release操作。已驗二進位與44份同hash候選複本清出82129010 bytes，
原稿、配方、收據與首次失敗保留；runtime-only複本只在容器tmpfs，退出即釋放。
官方工作清單與最終本地連結、擁有權及Docker清理核對另記主收據。

官方工作清單21項開啟、14項人工、可能已完成0。變更文件1421個本地連結通過，
README新圖與61新工具入口作正對照。本批7415路徑UID／GID皆1000，沒有.md目錄；
全庫舊九項連結限制保留，不宣稱掃過全部歷史work。

## 2026-10-19 四召喚施術者HD呈現

從main 98899bef建立hd-caster-stage，沿既有RE／READY契約接prelude、actor及actor-slide。
首章悠妮Class5、指令遮罩僅普通攻擊，不改隊伍冒稱自然施法者。改用既有四召喚修改fixture。
producer明示ActorIndex／TargetIndex，原CPU介面沿同一排程；四正式caller消費同份圖源描述子。
角色／效果先共同驗證，原生演出全部建構成功才共同發布；F2預載整場兩家族，不在Draw讀檔。
原位置、矩形、層序、delay、hold、音效、HP／MP、兩種RNG、發布及續行邊界保持。

最終395組GPU／CPU回歸、15頂層加14子項且零SKIP、既有producer9測試通過。
四caller完整呈現158／156／241／255步，各階段F2、缺件拒收、三段發布及一次續行通過。
148張最終GPU核對HD核心；前景原生mask／HD alpha先核SHA，只排除實際遮擋，剩至少1000核心。
效果PNG替身保持完整收據旗標、錯人物及損壞來源三種負向均拒收，沒有發布通過報告。
自我首目標會產生兩個相同人物圖層，依既有計畫保留，不改測試位置掩蓋。

熾天使擷取兩次18張被終止，第二次memory.events證實2GB oom_kill。
回讀畫布／緩衝重用後共同18張PNG SHA相同；破壞神仍OOM，再提前缺件檢查並延後圖源發布。
最終四流程同2GB上限皆無oom_kill，峰值仍約2GB，不宣稱平台效能合格。
單元測試r3以素材指標暫存後清欄位造成還原失敗，r4改存FDOTHER6欄位，同條件重跑通過。
圖源核對第一版忽略前景而拒actor，修為有雜湊來源的遮擋解釋，未放寬未知像素。
教訓工具排除測試檔，此擷取教訓不掛無效的production guard。

清冊仍375項，FIGANI六資源46幀及FDOTHER四效果37幀；沒有新資產或私人同步。
私人3392檔／runtime-only3146檔沿前批證據，本批工作樹乾淨，HEAD仍b64bc057。
README新增實際GPU施術者單圖，原兩張候選來源SHA相同，取消重複兩格總覽。
58、57、61、#263與主收據compound_caster_followup_20261019同步。
完整戰鬥HD Goal與#263保持開啟；普通法術、其他人物／效果、非共背景、保存冷讀及效能／平台另驗。
本批無錄影、matching、原版oracle、PLAYER-E2或Release。六份已驗程式／測試二進位已記SHA後刪除。
官方worklist、連結、所有權與Docker最終核對另存主收據。

本批暫存二進位釋出143501824 bytes，沒有錄影。259條教訓／139守衛通過；
官方worklist為21項開啟、14項人工、可能完成0。變更文件連結與所有權最終結果存入主收據，
不宣稱掃過全部歷史work；原有九項歷史連結限制保持。

## 2026-10-19 悠妮大型戰鬥HD與金色圖源核對

從main 8e64173a建立hd-yuni-battle，沿規格閘門及imagegen技能補FIGANI27待機4、28攻擊11；29零header。
原版大型造型為紅長髮、紫灰護具、金色長披布與短杖，不從地圖裝備或Class5文字猜戰鬥圖。
內建image_gen產生八母稿，完整提示、參照、原稿／工作區路徑與SHA保存主收據。
展臂、抬杖與少數收勢明示共享，完整15幀沿原位置、層序、delay及hit／cue標記。
兩縮放群組均1.0；右側及少數底邊裁切逐幀揭露，沒有擴張矩形或改寫原始生成alpha。
乾淨重建15幀與八母稿相同，11種編製拒收通過。

普通首章來源槽仍SHA089b077b，Z正常選在場序號1悠妮，兩次各一次END接敵、攻防與返回通過。
F2、經驗對話F9、Home、F5及新程序標題LOAD通過；最後先F9回來源再F5，bytes相同不證受傷後持續。
保留38張主要GPU、118張戰鬥PNG、4張走行及132筆輸入；649.341秒為軟體擷取耗時，不是平台效能。
81張悠妮金色核心、74張強盜藍褲相符，兩場39／32張同圖符合雙方。多候選不唯一判相位。
新gold取樣只選來源服裝，不放寬RGB／alpha比較；預設blue沿舊索爾57張回歸保持。
錯收據、錯人物、錯來源與偽造指紋的舊實際擷取四種拒收，不只核hd_battle旗標。

候選／正式runtime-only的Go各4頂層加8子項通過，零SKIP；正式清冊11測試通過。
Go來源488檔未改，沒有新規則、保存格式或原版oracle。全來源盤點409位置／264已解碼／2118原幀，
FIGANI八資源61幀正式登錄；source408仍阻擋。另按固定SHA分組264來源，249種raw，不推測人物別名或算HD完成。
正式377項與私人3415檔／runtime-only3161檔逐筆核對，原3392檔保持，八母稿不進執行包。
私人提交、公開提交與遠端核對稍後記於議題及主收據。

Go初次從/out別名讀候選相對symlink誤缺頭像；同映像改/repo既有層級後通過。
GUI映像入口為python3，沒有另建映像或退回主機。舊索爾藍衣回歸的兩份清冊SHA不符，查到實際r1後通過，
沒有放寬收據檢查。冷啟動診斷短暫讀不到audit與relative_to錯誤不當產品缺陷，也未重啟GUI。
新增可重用的候選掛載路徑教訓。主收據為yuni_battle_followup_20261019，58／57／61與README同步。

兩份已驗二進位與54份同hash圖片複本清出102743720 bytes；原稿、配方與收據保留，沒有錄影。
runtime-only複本只在容器tmpfs。完整戰鬥HD Goal與#263保持開啟；下一個共用呈現缺口是
modern_hd_battle_runtime.go的非零header背景守衛，沿既有nativePhysicalScroll另立READY再實作。
其他人物、法術／死亡／共同效果、正常施法與代表性效能／平台仍待；蓋亞加寬仍待使用者決定。

私人af9ca1b953510eeda9c70965da926780e0c87eef已推送並核對遠端HEAD相同，工作樹乾淨，未改可見性設定。
260條教訓／140守衛通過；官方worklist為21項開啟、14項人工、可能完成0。
F2普通抽樣在戰場／指令環與面板，沒有新增物理演出中F2證據。

## 2026-10-19 非共背景物理戰鬥HD捲動

從main 860a189b建立hd-scroll-battle，沿既有IDA主證據與READY契約實作。
原生producer附phase、ContentX／BGX／ViewportX及不可變BG歷史；舊pixels介面沿同一排程。
GPU由乾淨base及明示人物圖層組成，Source先限320畫布，TargetIdle保留640工作面／線性base跨列。
透明BG依原順序最後覆蓋，opaque index0保持。目的裁切先驗後發布；錯幾何不清上一張世界。
mode1入場、departure、連射之間返回、反擊、DAC及F2沿原owner；未知counter BG仍拒收。

四producer測試及783 GPU案例通過，含既有395與新增388；原生全畫布exact，HD alpha容差1。
相關回歸17頂層加20子項通過，首跑2條件SKIP。明示人物96及正式清冊後，
HD完整性／玩家及AI結算前拒收再驗2頂層加2子項且零SKIP；six-BIOS-wait需指定原版收據，本輪不重跑。
既有共背景完整Draw回歸通過。非共背景side0診斷139次及正式雙側139／140次呈現通過，
每場六階段F2、兩次射擊及反擊、64／65 DAC、一次續行、HP／MP／RNG與GPU釋放保持。
正式兩側528／531次HD Draw，Draw數隨軟體擷取變動，不當作frame或平台效能。
七張實際最終PNG核對敵方藍褲核心；錯範圍、原生PNG替身與錯幀均拒收，不發布通過報告。

本批是修改BattleFig／攻擊計畫fixture的RUNTIME-E1。FIGANI24／25仍原生，僅已登錄敵方使用HD；
地圖返回沿既有native fixture。沒有新增普通玩家、保存冷讀、原版oracle、matching或PLAYER-E2。
既有不同背景RE不重開。377項清冊及私人af9ca1b9保持，沒有新增資產或同步。
蓋亞占寬仍待決，完整Goal與#263保持開啟。其餘人物、法術／死亡、普通施法與非共背景正常路徑及效能／平台繼續。

r1 tmpfs noexec令測試二進位permission denied，改exec後同映像通過；新增可重用教訓。
r2測試誤用不存在的preludeTicks，改核atk.timer／accumMillis後r3乾淨編譯通過。
七個2GB驗證容器均無OOM，最大峰值1160224768 bytes，不宣稱平台驗收。
README新圖逐byte複製真正捲入中PNG，未改圖；58／57／61與主收據physical_scroll_followup_20261019同步。
沒有錄影或Release操作。已驗二進位清理、工作清單與最終連結／擁有權／容器檢查另記主收據。

官方worklist為21項開啟、14項人工、可能完成0。261條教訓／140守衛通過。
1440個本地連結與三項正對照通過；78個本批路徑UID／GID均1000，無root-owned或.md目錄。
只查變更及scroll輸出根，全庫舊九項連結限制保持。唯一已驗測試程式記SHA後刪除，釋出26319248 bytes。
私人工作樹乾淨、HEAD仍af9ca1b9，沒有新增素材同步。

## 2026-10-20 希莉亞與第三章劍盾兵 HD 候選

沿 main cad8d3ad 的 hd-silia-battle 分支，先完成希莉亞 FIGANI24／25 的20幀候選。
原始 Mask 證實25/14是320×200全透明幀，15..18是四張50×3飛矢。
正式 loader、編製器與來源稽核只在原始 Mask 完整且全0時接受全透明HD圖，
空白 present 與 non-nil 圖源照常保留。缺Mask、短Mask、非空來源及alpha1墨跡拒收。
七種Go條件、空白GPU合成、16項配方／空白稽核與雙側完整Draw通過。
新增四飛矢實際PNG核對，缺幀、錯幀、錯位置及原生像素替身四項拒收通過。
錯幀先被原始垂直描述子擋下，已修正反例的預期層級，未放寬驗證。

希莉亞原生來源沿第2章酒店SAV，SHA d8d534d5；祖先清敵政策見ch03-manifest。
普通標題LOAD、城鎮出發、第三章劇情及F5已通過。GUI首次視窗等待60秒不足，
改明示180秒；章首只在有對話時送鍵會漏掉最後確認，修正普通Return驅動後乾淨通過。
F5只承諾章節邊界保存，LOAD重建戰場。抽驗改以重建結果對冷讀，不為測試新增保存欄位。
獨立原生SAV複本可被F5同步保存，唯讀來源雜湊保持。未把戰中位置或敵方HP視為持續契約。

實際第三章敵方為BattleFig76，既有強盜96不能替代。依原始13幀另產生六母稿；
第一版劍與揮斬過寬，使站姿／長揮斬縮至0.785／0.668，拒收三張並以原始單幀重畫。
修訂後站姿縮放0.976，其餘三群1.0，13幀高覆蓋裁切均0。希莉亞20幀亦無高覆蓋裁切。
33幀乾淨重建、來源描述子與清冊內容相同，重複輸出已刪。合併候選381項，正式仍377項。
新候選Go完整載入、原子發布與來源拒收通過；圖源盤點仍409位置、264已解碼、2118原幀，
候選涵蓋12個FIGANI資源94幀，來源408仍阻擋，不能稱正式覆蓋。

普通攻防抽驗以正式射程2..3與可達集合移動待機，再正常END；NPC鐵諾尚未HD，
END明示以普通F2選忠實模式，回玩家回合再切HD。無HP、座標或狀態注入。
本段寫入時已完成兩輪接敵並進入第三回合HD射擊，重播、返回與冷讀尚待結束收據。
人物本體核對另明示幀範圍，透明幀及飛矢不套服裝核心門檻；舊索爾57張藍衣正向回歸保持。

前置拒收補驗誤用FD2_TEST_MODERN_BATTLE_FIG，沿預設蓋亞而缺銀行；
改正式FD2_TEST_MODERN_BATTLE_FIGURE=8後player／mode11通過。
工作清單原核對掃入remake/packaging/dist的舊封包，現只排除此明確中間產物根；
真實建置腳本仍掃描，新增正反測試後11測試通過，遠端21項、人工14項、可能完成0。
263條教訓／142守衛通過。三個舊程式記SHA後刪除，釋出71977392 bytes，未建立錄影。
主證據沿 fd2-modern-hd-battlefield-20261019.json 的 silia 與 assassin76 後續區塊。
完整戰鬥HD Goal與#263保持開啟；正式登錄、私人同步與本輪普通收據仍待收尾。

後續收尾已通過。兩場正常射擊與F9重建、返回、F2、戰後F5及新程序LOAD完成；
28張主要GPU、128張戰鬥PNG、4張走行、211筆輸入與四次END接敵保留，85張符合希莉亞核心。
來源與兩主題保存bytes相同、章節邊界冷讀重建一致，不升格戰中HP持續或整章E2。
擷取1237.178秒是軟體環境耗時。敵方另以強制反擊計畫完整146次呈現，九張揮劍幀與四飛矢
實際PNG核心相符；六階段F2、64／65 DAC與一次續行保持。六種fixture圖片反例與五種正常圖片反例拒收。

正式381項清冊的11測試通過。首跑誤用缺Pillow的promo映像，改用既有素材映像後同測試乾淨通過。
來源稽核確認正式12個FIGANI資源94幀；四召喚37幀保持。私人完整3464檔與runtime-only3194檔逐筆驗證，
原3415檔保持，新增16母稿不進執行包；runtime-only Go兩項加兩子項通過且零SKIP。
私人45a3a9a0938661dc73e3cfa05aa397c5c9302b5a已推送，實際遠端HEAD相同，工作樹乾淨。

固定來源盤點另存受版控快照，SHA d019e2cb保持；focus與舊清冊欄位明示只屬當批資料。
兩份完整重建配方存主收據，改用私人allow-list母稿再次重建33幀，PNG保持，暫存輸出清除。
README逐位元組複製正常射擊PNG，58／57／61與遠端#263同步。完整Goal仍開啟，NPC鐵諾、
其他人物／法術／死亡／普通施法與平台效能待完成，蓋亞大型戰鬥占寬未擴張。
六份已驗程式累計清出143964136 bytes，沒有錄影、matching、新原版oracle或Release操作。

最終1452個本地文件連結與三項入口正對照通過，29個變更檔UID／GID均1000，
Python語法、JSON及Go格式通過；README文化保存入口保持。遠端worklist更新後仍21項、人工14項、
可能完成0。未留下本輪FD2容器；公開提交與推送版本以GitHub #263收尾留言及Git紀錄為準。

## 2026-10-20 鐵諾 HD 與正常 END

由 main 01aece4b 建立 hd-tino-battle。FIGANI6四待機、7十攻擊、8零header已核固定SHA與Mask；
五份母稿保留白髮、藍披風與三拍斬光，編製14幀，所有高覆蓋裁切為0。
站姿群縮放0.926，斬光群0.982。16項編製／來源反例、候選14資源108幀稽核、
Go正式載入與玩家／AI結算前拒收通過。清冊allow-list母稿另乾淨重建，19檔及清冊項目逐位元組相同，重建暫存已清除。

完整Game.Draw修改fixture取得14張最終1920回讀，14幀核心核對通過；
三階段F2、64／65 DAC、一次續行、HP與兩種RNG保持。既有診斷fixture回歸也通過。
新核對器的五項fixture反例與舊希莉亞85圖回歸保持，scope仍為RUNTIME-E1，無新原版oracle或matching。
第一次缺圖反例誤綁索爾銀行，依實際actor.BattleFig修正後通過，產品規則保持。

普通第三章沿既有相同來源及seed1，取消END的忠實主題切換，僅用正常鍵盤。
r1四次END與兩場射擊已返回，最後Enter跨過自動關框而開狀態面板，Z被面板消費，沒有PASS。
r2先正常關面板，但又錯把nativeMovePlan指標當成活動面板；八次Escape其實已正常循環索爾／蓋亞。
原始控制器顯示g.sel=nil時Escape已是循環選人，move_preview只表示plan存在，不能作取消門檻。
修正GUI後正以fd2-tino-normal-r3乾淨重跑，輸出tino-normal-gui-r3。遊戲binary未變，正式清冊仍381項。
失敗觀測保留OBSERVATION-ONLY，不能混作完整保存冷讀通過；r3結束前GUI來源保持不動。

七張r1 PNG以來源核心證實比觀測job慢一拍，甚至跨待機／收勢邊界。
END核對改只取同圖庫caller位移與clip，再由實際PNG找姿勢；fixture仍同次Draw逐索引核對，容差保持。
r2的56張END樣本有50張實際符合鐵諾圖源，但本項只作診斷，保存冷讀仍待r3完整收據。
所有提示、配方、版本與限制集中在fd2-modern-hd-battlefield-20261019.json的tino區塊。

等待期間備妥哈諾FIGANI3／4的11幀候選及六份母稿；站姿0.983、收勢0.968，斬光1.0，沒有高覆蓋裁切。
來源及16項反例通過，385是以鐵諾隔離候選為基底的候選清冊；沒有正式登錄或執行期／玩家通過聲明。
現有章節槽哈諾HP0，後續須找存活來源或正常教會復活，不直接改HP。契約與入口已掛61及主收據hano區塊。

舊r1測試程式記SHA後刪除26360104 bytes；r1冗餘200張失敗PNG刪除82379675 bytes，
保留七張相位診斷圖、戰後返回圖、audit及operations。另清除14592154 bytes重建輸出。
私人素材庫仍為private，未改寫。正式登錄、私人同步、README及#263收尾仍待完整普通GUI。

勘誤：逐讀r2八筆cleanup，player_status_phase前後皆nil，沒有實際演出關頁。上文『先正常關面板』指已加入關頁路徑，並非該輪有執行。r3也不需額外清理輸入；移除plan假門檻後，完整Z選人、F2、保存冷讀通過。


r3完整通過，1437.072秒為軟體擷取耗時。四次END維持HD、兩場希莉亞射擊、返回、F9、F2及章節邊界保存／新程序LOAD完成；
28張主要GPU、128射擊PNG、56 END、4走行及180筆輸入保留。51張END符合鐵諾核心，73張射擊符合希莉亞核心。
source及兩主題保存bytes相同，冷讀重建單位一致；r3未執行額外cleanup按鍵，不宣稱戰中HP原地持續或整章E2。
完整14幀fixture與五項反例保持，新增六項END反例含錯caller位移亦拒收；舊希莉亞85圖回歸通過。

正式383項清冊11測試、14個FIGANI資源108幀及四召喚37幀來源稽核通過。
私人完整3483檔逐筆SHA核對，原3464檔保持，新增19檔僅14幀與五母稿；runtime-only3208檔不帶新增母稿。
正式Go真實載入與player／mode11前置拒收兩項及兩子項通過，零SKIP。
私人132b2014ff213e3c11608575a82213aac874e406已推送，實際遠端HEAD相同且保存庫仍private。

README新增未編輯的正常鐵諾斬擊PNG，58／57／61與遠端#263更新；官方快照21項、人工14項、可能完成0。
三項新教訓後共266條／145守衛通過，1461個本地連結與三個正對照、Python語法、JSON、Go格式及變更檔UID／GID1000通過。
舊原文限制新增後續指標，歷史WORLD／Silia收據與文化保存入口保持。

三份程式共72036976 bytes、200張冗餘失敗PNG82379675 bytes、重建14592154 bytes與runtime-only副本187139049 bytes已刪，
合計356147854 bytes。七張相位診斷PNG、一次返回PNG、r2觀測與r3完整收據保留；沒有錄影。
工作輸出root-owned與.md目錄檢查為空，沒有非預期FD2容器。
完整HD Goal及#263保持開啟，哈諾11幀仍僅候選，其他人物／法術／死亡／普通召喚及平台效能待；蓋亞大圖占寬未變更。
公開提交與實際遠端核對以Git及#263收尾留言為準。

## 2026-10-20 蓋亞原框戰鬥圖與哈諾繪製驗證

由main 1d36b507建立feat/gaia-battle-original-geometry-20261020。先核對遠端#263、固定來源盤點及61規格。
沿用使用者核准的第二版硬甲機器人造型，重畫朝左側身的緊湊待機與蓄力；未採用舊擴框原型。
新兩母稿完整提示、參考SHA與原始RGBA已保存，impact／flare／follow沿既有三稿，依明示配方編製4／11幀。
standing比例1.0，punch群組0.9809905264420193；91/6有4個高覆蓋邊界裁切，其他十四幀為零。
待機四拍與蓄力五拍共享稿，極小充能亮點及柔化光邊差異明示，不宣稱原版逐幀相同。

16項編製反例通過，385項隔離候選的16個FIGANI資源123幀來源稽核通過。
完整Game.Draw取得15張同次最終1920回讀，15幀核心相符；三階段F2、64／65 DAC、一次續行與HP／RNG保持，
258次HD、3次忠實及326次最終Draw，199.51秒。五項圖源反例拒收。
fixture只換來源BattleFig30／76，其他來源名稱及規則保持，不能當作正常蓋亞存檔或原版傷害對拍。
第三章普通選取蓋亞、移動／待機／END、攻防、F2及保存冷讀正在獨立驗證，尚未正式登錄或私人同步。
私人原3483檔已逐筆SHA及UID核對，既有內容未改寫。正式清冊仍383項。

同一份測試程式另完成哈諾原11幀候選的完整Draw與PNG核心核對，三階段F2、64／65 DAC及一次續行通過；
244次HD、3次忠實及313次最終Draw，175.84秒。五項圖源反例拒收。
哈諾仍只屬隔離候選RUNTIME-E1；來源HP0，正常教會復活／攻防與保存另驗，未正式登錄或私人發布。
已核對churchCandidates依partyJoinOrder及CanRevive取名冊，復活服務選項2及確認0沿既有typed consumer；
不需要為取得存活來源直接改HP。舊385候選正式登錄前須依最新清冊重編，不得覆蓋後續資產。
主證據新增gaia_native_battle_followup_20261020，哈諾沿原區塊追加runtime_followup_20261020。

普通GUI兩次攻防、返回、F2與冷讀已走完，但最後line868指定Gaia END樣本0而失敗，沒有原GUI receipt.json。
四段END觀測群組為4／3／4／3的鐵諾／士兵；本輪Gaia沒有在END參戰，原OBSERVATION-ONLY及128張PNG保留。
沒有重跑整場或把失敗改PASS。先在61定義獨立範圍，再以受版控追蹤工具核175筆按下／放鍵ACK、兩場呈現與返回、
四段END存活、F2狀態、F5／F9兩主題及新程序標題LOAD的audit投影。只有普通攻防與保存追蹤通過，
指定END圖源及原控制器完整通過仍不成立。12項實檔反例包含偽造冷讀布林值與控制追蹤有效的黑圖，均拒收。

128張演出PNG中92張符合Gaia、98張符合士兵核心，87張同圖相符。新版核對器另重跑原15幀fixture與五項反例保持。
戰中Gaia62HP，邊界LOAD回來源68HP，source與兩主題bytes相同；不宣稱戰中HP／位置持續。
兩次序列均同來源FD2_SEED1，章內沒有鎖血、改座標或狀態注入；祖先111抽樣後清敵照舊揭露。

正式385項、16個FIGANI資源123幀及四召喚37幀已核。清冊測試曾因合成90與新正式90撞號而正確拒收；
只隔離測試清冊的同來源項目後，同容器同命令11項乾淨通過，正式重複來源拒收保持。r1失敗log保留。
私人3503檔逐筆SHA、UID核對，原3483保持；runtime-only3223檔不帶新增五母稿。
Go正式真實載入及玩家／mode11結算前拒收兩項加兩子項通過，零SKIP。私人1910e36765aba56b4e6fc00eec84c42ada0dfcc1
已推送，實際遠端HEAD相同，保存庫仍private。README新增未編輯的普通Gaia圖，與原鐵諾圖並列。

本輪先清除19809326 bytes乾淨重建，再清除兩程式、runtime-only副本與可重生的負向影本297620216 bytes，
共317429542 bytes檔案。128張演出圖、15幀fixture、操作／audit、原失敗觀測及最新追蹤都保留，沒有錄影。
清除binary後獨立追蹤再次核對通過；只保留已核指紋，不跳過原始PNG／ACK／冷讀證據。
兩條新教訓後共268條，146個有效guard通過。guard掃描刻意排除測試檔，原先兩條綁測試檔的設定已修正：
END範圍綁正式追蹤器的False欄位，清冊fixture隔離條目以真實11項回歸驗證，不掛不適用訊號。
遠端#263本文及正式worklist快照已更新，仍21項、人工14項、可能完成0。完整Goal及#263保持開啟。
原圖框與Go runtime保持，沒有matching、新原版oracle或Release操作。公開提交、連結／擁有權及Docker收尾另記實際結果。

收尾2042個本地連結、三個README入口正對照、16個變更檔UID／GID1000、JSON／Python語法及1920截圖形狀通過。
README文化保存入口保持，Go runtime零變更，work根與私人包沒有root-owned或.md目錄；沒有非預期FD2容器。
公開提交與推送的最終SHA由Git及GitHub #263收尾留言記錄，私人遠端1910e367已核；完整Goal保持進行。


## 2026-10-20 哈諾HD、教會來源與章前HP校準

由main ddbe7d64建立feat/hano-battle-normal-path-20261020，先依最新385正式項重編哈諾候選，
原11張PNG保持，六母稿與共享待機／末兩收勢、微小餘光差異照舊。完整11幀修改fixture重跑取得
238次HD、3次忠實及322次最終Draw，三階段F2、64／65 DAC、一次續行與HP／RNG保持。
新版圖源工具11幀及五項反例通過，仍限RUNTIME-E1。

第二章沿既有酒店939點原生槽正常LOAD，F5取實際JSON名冊。首跑誤把Go名稱當JSON鍵；
改依has_native_record_byte5／native_record_byte5與omitempty後，教會按正常順序選哈諾，付150金、
HP0→36、gold1000→850，其他名冊不變。該跑尾端又假設story_ch02，實際為story_ch02_pre，
原GUI未通過；獨立復活資料核對及保存後原生槽續接另驗，不補造完整PASS。章首路徑改由authored資料讀取。

續跑在章首遇到speaker134缺圖，補原版朝左閉眼表情後正常章首／F5通過。
接敵首跑在第三回合又缺speaker133，依原版少女表情補圖；兩圖用內建image_gen沿既有手繪風格，
生成母稿及完整提示已保存，80／480不透明靜態圖按既有Lanczos編製，不猜人物姓名或宣稱四嘴形。
133的髮帶與上緣構圖屬手繪重定向，另明示。沒有改缺件拒收或跳過劇情。

補圖後36HP來源在第三回合AI中陣亡，存活斷言正確拒收。停止刷原難度，沿使用者先前允許作弊驗流程，
使用既有fd2-chapter-slot -boost-only -boost-max-hp 1000，在教會保存後同章1→1提高章前HP／MaxHP，
存活者加1000、死亡者維持0。完整解碼SAV僅18個byte不同，均為HP／MaxHP與檢查碼；其餘bytes相同。
沒有AP／DP／DX、金幣、裝備、事件、升級或推章政策，章內不鎖HP或改座標。
Go首跑缺可寫cache屬容器環境問題，指定/tmp/go-build後同工具成功，沒有產品程式修改。

新原生槽正常標題LOAD、章首與F5建JSON來源，再以FD2_SEED1及普通鍵盤完成三次END、第三回合事件，
一場哈諾主動攻防自然返回、F2、保存與新程序標題LOAD完整GUI通過。
21張主要GPU、29張戰鬥取樣、4張走行與197筆輸入保留；10張實際PNG符合哈諾核心。
607.5秒為軟體擷取耗時，不是平台效能。戰中966HP／等級2，邊界冷讀回1036HP／等級1，
source與兩主題保存bytes相同，不宣稱戰中HP、升級、位置持續或新整章E2。
單場模式仍驗完整自然返回與保存，預設兩場不變；九項拒收及舊希莉亞兩場85圖保持。

正式391項、18個FIGANI資源134幀及四召喚37幀，來源與實檔稽核及清冊11測試通過。
私人3526檔逐筆SHA與UID／GID核對，原3503保持，runtime-only3238檔不帶新增八母稿。
正式Go真實載入、玩家／AI結算前拒收與完整HD頭像三項加兩子項通過，零SKIP。
私人fc1931722cb43cb36d7421a28df9b14d22e7ea7e已推送，實際遠端HEAD相同，保存庫仍private。
README新增未編輯的正常鍵盤斧擊PNG，明示章前作弊來源，並修正蓋亞仍用原版大型動畫的過期句。
56／57／58／61與主收據已同步，舊失敗及隔離fixture的證據等級保持。

清除253142683 bytes程式、runtime-only及可重生反例副本，保留runtime manifest；
另將69520216 bytes重複圖檔改指已登錄、相同SHA的正式素材，85檔逐筆核對。
清理後10張正常圖源再次通過。沒有產生錄影，實際GUI、11幀PNG、操作及失敗觀測均保留。
新序列化教訓後共269條、147個guard；最終連結、擁有權、議題與Docker檢查另追加實際結果。
完整Goal與#263保持開啟；其他人物／法術／死亡／普通召喚及效能／平台仍待，沒有matching或Release操作。

收尾2048個本地連結、三個README入口正對照、16個變更檔UID／GID1000、4份Python語法及4份JSON通過。
GitHub #263與#201本文已更新，保留完整驗收與歷史限制；正式快照21項、人工14項、可能完成0。
工作根沒有root-owned或.md目錄，私人工作樹乾淨；沒有非預期FD2容器，公開提交與遠端SHA以Git及議題收尾留言記錄。

## 2026-10-20 第二章村民HD待機與普通END

前批哈諾及故事表情已推送，屬有效進展。從main f109692ffdbffe8ade93afba46001648e8c6ec56
建立本批分支，沿復古重製規格路由及61 READY契約接續完整戰鬥HD Goal。
固定盤點證實FIGANI399／402各四幀，400／401／403／404為zero_header_word，
不補造攻擊或施術，也不因DATO／FDICON同號而合併身分。

內建image_gen依原版背向姿勢與核准手繪風格生成兩份透明母稿。
134初稿過寬拒收，第二稿收窄站距及手臂；133取樣錨點加四個母稿像素緩衝，消除兩個高覆蓋邊緣裁切。
生成alpha不變，完整提示、兩份正式配方、來源SHA及拒收稿留在主收據villager_idle_battle_followup_20261020。
每組四個呼吸相位共享一份母稿，沿原y／height、delay及原本重複相位，非四幅獨立新姿勢或逐幀原版一致。

編製器新增單圖庫範圍前須核對同figure兩項固定零header，未知／decoded／錯SHA均拒收。
完整Go繪製測試新增idle-target兩階段範圍，正式Go規則、renderer與保存未改。
首輪誤配同在右側的索爾圖源，完整流程通過但村民核心被遮擋，像素核對正確拒收；
依原描述子改用左側劍盾兵76，保持完整核心與像素標準後，各四張最終1920回讀通過。
各185次HD呈現、兩階段F2、64／65 DAC、一次續行與HP／兩種RNG保持。
舊哈諾三階段fixture完整十一幀、243次HD呈現及三次F2保持，沒有用待機測試冒充攻擊驗收。

正常來源沿已驗教會復活後章前HP／MaxHP+1000槽及seed1，章內不鎖血或注入位置。
GUI可同次明示133／134；沿原實錄確認前兩次換手無物理戰鬥，移除每回合必須接戰的錯誤門檻。
所有實際戰鬥仍須HD，整份GUI及圖源核對仍須實際END樣本，沒有補造戰鬥。
普通三次END、事件、一場哈諾攻防自然返回、F2及章節邊界保存／新程序LOAD完整GUI通過。
END八張擷取中，兩位村民各三張／兩張符合高覆蓋服裝核心；哈諾另27張符合核心。
source與兩主題保存bytes相同，冷讀單位相同；依章節邊界重建，不宣稱戰中位置／HP／升級持續。
來源含作弊，不新增自然難度、傷害／AI、原版oracle或整章E2。

編製器兩組各20項、待機fixture各12項、普通END各14項反例通過；舊雙圖庫編製16項保持。
負向測試首次指錯不存在silia-normal-gui-r2，屬命令輸入問題；使用既有r1後乾淨重跑，各14項通過。
coverage首跑參數誤用--pack，改為工具既有--original-pack後同容器重跑通過，不列產品缺陷。
舊哈諾／希莉亞10／85張普通像素回歸保持；11項正式清冊測試及來源／實際PNG覆蓋稽核通過。

字串回歸發現舊review指紋漂移。比較其原始盤點證實99筆go_review完整相同，
差異是之前正式Go新增四條繪製診斷字串及UI行號；本批只改_test.go，沒有新增玩家文字。
工作區另有私有衍生command_labels的38條，乾淨git archive沒有，保留輸入差異。
沿既有遷移器重新綁定，處置與99筆ID保持，八項字串回歸全部通過，摘要同步。

正式清冊393項、FIGANI20資源142幀及四召喚37幀；原391項與所有既有圖源保持。
私人3536檔逐筆SHA／UID核對，原3526保持；runtime-only3246檔不帶新增兩母稿。
正式Go真實載入、玩家／AI結算前拒收與完整HD頭像三項加兩子項通過，零SKIP。
私人d072ebc4c61359480424ab60292a9abe536a8119已推送，實際遠端HEAD相同，仍private。
README新增兩張未編輯的普通END視窗並揭露章前來源，56／57／58／61及GitHub #263／#201同步。

清除277797769 bytes已驗程式與runtime-only，另將17937698 bytes重複圖源改指相同SHA正式素材。
runtime manifest、普通END、完整Draw與首輪遮擋失敗證據保留；清理後兩組像素核對通過，沒有錄影。
GitHub真實快照21項、人工14項、可能完成0；269條教訓／147個guard通過。
完整Goal與#263保持開啟，其他人物／施術者／法術／死亡／普通召喚及效能／平台仍待；沒有matching或Release操作。

收尾五份入口文件共1948個本地連結及六個README正對照通過；20個變更檔UID／GID1000，
五份Python語法及五份JSON解析通過，git diff --check無問題。
私人工作樹乾淨，沒有非預期FD2容器；公開提交與實際遠端核對由Git及議題收尾留言記錄。

## 2026-10-20 瑪琳、凱麗與貝克威戰鬥HD

沿main c583540306d8be38c0accdf7f517e51975e02914的獨立分支接續完整戰鬥HD Goal。
先核固定盤點與原版分離資產：30／31共15幀、36／37共20幀、39／40共21幀，32／38／41為零header。
61先寫READY，依原版背向、裝備與核准手繪風格，用內建imagegen逐份生成透明母稿。
相近呼吸與姿勢明示共享，原描述子、delay、cue與空白遮罩保持，不稱逐幀原版一致。
瑪琳橫揮初稿過寬、第二稿後腳抬起均拒收，第三稿收衣擺並讓雙腳落地；
貝克威端帽與尾端初稿過長，各重畫一次，保留拒收來源與完整提示。

三人串成同份399項候選清冊，原393項保持，56幀高覆蓋區均無裁切。
貝克威第一次完整Draw已跑完，但測試只收body／counter，漏掉40/0..13的departure而拒收。
查既有native_physical_body.go的HeaderByte1／HeaderByte2 producer後，只補test capture departure，
並要求非零header的離場F2；不改正式renderer、排程或規則，也不放寬完整21張門檻。
新測試三人完整最終1920回讀15／20／21張、64／65 DAC、F2、一次續行及HP／兩種RNG保持通過。
瑪琳15張紫色核心、凱麗20張藍色核心、貝克威15張本體藍色及5張光束暖色核心逐幀通過；
40/19沿原Mask保持透明。fixture反例5／5／6／6項與編製器各16項通過。

第八章原生來源正常LOAD／出發／F5已驗，但三份普通接敵抽樣都在第二次END遇前方BattleFig5 NPC死亡而拒收。
沒有在章內補血、移位或放寬存活門檻。查第九章typed部署無該前方NPC，改由同一原生基底套既有戰後JOIN5，
建立第九章章前HP+1000來源，無額外升級或金幣政策。未加血同源對照與加血槽解碼差46bytes，
只在11筆HP／MaxHP及檢查碼，其餘bytes完全相同。第九章正常LOAD／出發／F5亦通過，三人普通抽樣接續執行。
本機基底SHA以建構manifest及實檔768a561e..為準；較早ch08-manifest的8113b684..不能冒稱同一份bytes。
來源、全部提示、配方、實測及原失敗已存主收據early_party_battle_followup_20261020，當時仍為候選，尚未正式登錄。

後續第九章瑪琳／凱麗各三次／兩次普通END、各一場攻擊自然返回、F2、F5與新程序標題LOAD完整GUI通過。
兩份收據的source／兩主題保存bytes相同，冷讀單位一致；章節邊界重建，不宣稱戰中HP、升級或位置持續。
瑪琳21張、凱麗33張實際PNG符合圖源核心，各九項普通反例拒收；舊哈諾／希莉亞10／85張保持。
貝克威在第二次END遇既有主動header零／反擊非零的caller BG拒收，另開#265。
此槽含加入時預設裝備，未證明該組合的未修改原版可達性；下一切片先核裝備／射程與既有caller證據。
沒有刪守衛、改無反擊或用完整21張fixture冒充普通攻防；貝克威仍為候選。

正式只登錄瑪琳／凱麗35幀與九母稿，原393項保持；GUI的399項隔離候選中，對應四項entries與44檔逐筆相同。
正式397項、FIGANI24資源177幀及四召喚37幀，固定來源與實際PNG稽核通過；來源408仍阻擋。
原正式private_root仍指上一批村民候選pack，首輪固定prototypes命令找不到新圖，同步前啟動Go亦缺件。
依清冊逐檔核SHA後補齊既有prototypes，正式根固定該處，先同步完成再乾淨重跑，沒有產品規則或renderer修改。
正式Go真實圖源、玩家／AI結算前拒收與完整主題三項加兩子項通過，零SKIP；清冊11項測試通過。
私人3580檔逐筆SHA與UID核對，原3536保持；runtime-only3281檔不含新增九母稿，貝克威候選未混入。
私人提交已推送，實際遠端HEAD相同，保存庫仍private；公開議題只更新公開圖源與驗證範圍。

README新增兩張未編輯的第九章普通攻擊視窗，明示章前政策；56／57／58／61與主收據同步。
GitHub #263／#201保留完整驗收並更新目前範圍，#265初稿verify.path已修正為工具實際使用的paths，遠端重拉後產生及驗證通過。
工作清單22項、人工14項、可能完成0；兩條可重用教訓後共271條、149個guard。
清除293155213 bytes已驗執行檔與runtime-only副本，另將52010228 bytes、100份相同SHA圖檔改指正式素材。
runtime manifest、實際GUI、全部Draw PNG、候選母稿與失敗觀測保持；沒有錄影。清理後圖源另以原收據重驗。
完整戰鬥HD Goal與#263保持開啟，其他人物／法術／死亡／普通召喚及平台仍待；沒有matching或Release操作。

本批收尾1957個本地連結、六個README入口正對照、17檔UID／GID1000、三份Python語法及四份JSON通過。
清理後21／33張普通像素重驗通過；工作根沒有root-owned或誤建.md目錄，私人工作樹乾淨，無非預期FD2容器。

## 2026-10-20 凱麗原版與HD攻擊README動畫

從main 02f29797建立獨立分支，沿README展示、影片處理及規格閘門路由載入契約。
61先寫READY，再製作實際原版與HD並排GIF；正式Go、素材清冊、規則及保存格式沒有修改。
舊oracle工作樹路徑已不存在，首次Docker掛載留下空root-owned目錄；確認空目錄後只移除該路徑。
後續使用專案工作根內的獨立Git工作樹，原始遊戲唯讀，章前FD2.SAV另複製覆蓋層。

來源選擇前三次失敗分別為音訊開發分支未含FD2 IRQ7、舊951不支援heap-profile、主線81c85be未合IRQ7。
查現有修復分支後固定190d944，完整鏈正常越過啟動。沒有新開硬體RE、補寫ISR或恢復輪詢近似。
原版從同源第九章HP+1000槽正常標題LOAD、出發、凱麗兩次移動待機與END，再普通掌擊並返回。
80次BIOS鍵讀、770張唯讀擷取及實際攻擊呼叫／返回追蹤保留；章內沒有鎖血、移位或清敵。
兩側皆遇BattleFig78，但原版攻擊前HP833、HD822，亂數與攻擊到達格亦不同，只列相近戰況。

右側沿已通過的33張普通GUI來源，剪取25組掌擊、旋身與收勢影格，左側依動作順序配對。
合成工具先核所有PNG、runner、控制計畫與收據SHA，拒絕髒來源、章內注入、錯尺寸及無HD實際畫面。
依HD觀測更新幀差剪輯，量化至10ms，另加首尾停留；不宣稱原版wall-clock、逐幀、傷害或亂數一致。
成品960×334、21幀、2.17秒、2130350bytes，25組來源中的重複幀合併而總時長保持。
兩側各21種畫面、循環及Pillow／FFmpeg完整解碼通過，四動作與標題裁切已目視審查。
README加入GIF與範圍圖說；57／58／61及主收據kylie_readme_animation_20261020同步，完整HD仍未完成。

新增來源能力／旗標預檢教訓。清除38801552bytes的本輪程式與字型副本；正式GIF及原版／HD截圖與收據保留。
本輪沒有建立錄影、改私人素材或發布Release；貝克威候選及#265既有缺口保持。

GitHub #263已更新，遠端重拉後22項、人工14項、可能完成0；272條教訓及150個guard通過。
六份入口共1497個本地連結、六個README保存正對照、語法與JSON通過，原版EXE及來源槽SHA保持。
12個變更檔UID／GID1000，工作根沒有root-owned或.md目錄；原版擷取容器已完整退出。

## 2026-10-20 貝克威普通購弓炮擊與原生購買保存修正

從main de98f4a7建立獨立分支，沿正常玩家試玩與規格閘門路由恢復現況。
查#265的實際槽與既有consumer，貝克威class4卻裝type1短劍、射程1，因而取得相鄰反擊資格。
61先寫READY，以正常城鎮購買1800金黑暗弓修正抽樣來源，不刪背景守衛或改反擊規則。
首輪控制器把雙欄Down當逐項，誤選4000金裝備；依正式Right逐項後重跑，保留失敗觀測。

第二輪普通購弓F5發現#266：立即裝備把原始格6／7的0x80旗標清零，兩個殘留item0被啟用。
原生購買直接呼叫正規化EquipItem，未沿獨立裝備已具備的ignored格保留契約。
補充READY後新增正式recipient／equip-confirm／成功生命週期回歸，修正前重現相同旗標毀損。
只將購買確認改接既有EquipNativeCompactSlot，pending候選及原子拒收保持，未改格式或數值規則。
修正後29項真實資產商店回歸零SKIP；同槽正常購弓／立即裝備／F5／出發通過，2000→200金，
HP及其他名冊保持，射程1→2～4，兩個ignored格保持0x80。
建置的登入shell PATH、回歸/tmp noexec及缺FD2_ASSET_PACK分開列環境失敗；完整環境的紅／綠回歸保留。

同來源seed1，普通一次END、由(11,31)移至(11,27)並射程4炮擊(11,23)的BattleFig78，
自然返回、F2及章節邊界保存／新程序LOAD通過。64張有界PNG中，本體14張與光束4張符合核心，九項錯配反例拒收。
原21張完整Draw、離場F2、64／65 DAC及一次續行證據保持；本批不改正式renderer或排程。
圖源報告首次未掛可寫輸出、守衛目錄別名不在工作根的拒收，修正命令後同輸入乾淨重跑，不列產品缺陷。

正式登錄39／40的21幀及八母稿，397項前綴及所有舊圖源保持，清冊399項。
固定來源稽核為FIGANI26資源198幀及四召喚37幀，來源408仍阻擋；清冊11項測試通過。
正式貝克威載入、玩家／AI結算前缺件拒收與完整主題三項加兩子項通過，零SKIP。
私人3609檔、runtime-only3302檔逐筆SHA／UID核對，原3580檔保持，新增八母稿不進執行包。
私人已獨立提交推送，實際遠端HEAD相同且保存庫仍private；公開議題只記公開範圍與驗證。

README新增未編輯的端炮與光束視窗，凱麗原版／HD GIF保持；56／57／58／61及主收據同步。
舊候選與拒收標為前批歷史，完整來源／提示仍可回查；#201／#263更新目前範圍，#266依驗證關閉。
#265保留一般混合header合法caller與BG資料未知，正常購弓來源不再阻塞本輪；不擴稱全部caller已支援。
遠端重拉後22項、人工14項、可能完成0。完整HD、其他人物／法術／死亡／普通召喚與平台仍待，未啟動matching或Release。

清除490484614bytes程式、Go快取及執行包副本，保留manifest；另將16764629bytes、29份同SHA候選檔連到正式來源。
全部普通截圖、完整Draw、來源槽、前後保存與失敗收據保留，本輪沒有建立錄影。清理後14／4張圖源核對保持。

最後Go修改後依既有規則重綁字串審查。舊盤點SHA與已提交綁定相同，5205項文字及來源簽名保持，
99項人工處置及各組數量不變；只更新盤點指紋，摘要同步，八項字串回歸全部通過。
七份入口2082個本地連結、六個README保存正對照、語法與JSON通過，來源槽／原版EXE及凱麗GIF的SHA保持。
273條教訓與151個guard通過，私人工作樹乾淨；本輪容器均已退出，新增字串快取另依主收據清除。

## 2026-10-20 鎧甲武士戰鬥HD與有界END擷取

從已合併main建立feat/heavyknight-battle-hd-20261020。固定原始盤點確認234待機1幀、235攻擊7幀，236為零header；77、79造型不同，沒有混用身分。
先寫READY，再以內建imagegen繪製待機、抬劍、下劈與回收四張透明母稿，完整提示與SHA登記主收據heavyknight_battle_followup_20261020。
明示共用姿勢並沿原描述子編製；抬劍各有9697個劍刃高覆蓋像素位於頂邊之外，身體保持，其餘五幀裁切0。

完整八張同次Draw、三階段F2、64／65 DAC與一次續行通過，五項GPU與16項編製拒收通過。
第九章沿前批正常購弓F5來源，祖先全11人章前HP+1000，固定seed1。正常前進／兩次END、敵方四場物理場景、凱麗主動及鎧甲武士反擊、自然返回、F2及章節邊界保存冷讀通過，章內未注入狀態。
首輪GUI與44張反擊核心通過，但END唯一234:0截圖實際仍是前一回收姿勢，嚴格門檻拒收，沒有補造PASS。
先補READY擷取契約，再加明示每幀上限1／2，預設仍1，第二張另存並保留前後觀測。遊戲與驗證器門檻不改。
第二輪完整GUI、END14張、反擊47張與暖色劍光3張相符，普通PNG符合八幀圖源集合，不判唯一相位，十項錯配反例拒收。

正式登錄兩項，399項前綴保持；唯一現況58與主收據同步，56／57／61更新。正式Go三項加兩子項、零SKIP，清冊11項與固定來源稽核通過。
私人allow-list完整3621檔、runtime-only3310檔逐筆SHA／UID核對，原3609檔保持，新增四母稿不進執行包。私人已獨立提交推送，實際遠端HEAD相同。
README新增兩張未編輯的普通敵方待機／下劈視窗，凱麗並排GIF保持。#201／#263更新並保持開啟，遠端重拉後22項、人工14項、可能完成0。
同次END觀察到87號弓手，原始19幀已閱，尚未HD登錄；其他人物、施術／死亡／普通召喚與平台仍待。#265及來源408保持，不新增原版parity、整章E2、matching或Release。

刪除本批程式、Go快取與runtime-only檔案副本，保留manifest、來源槽、全部截圖與收據。候選與工作母稿以同SHA連到正式素材，沒有新增錄影。清理數量與逐檔SHA見主收據。
公開提交前再核對本地連結、保存文化入口、字串綁定不變、擁有權與Docker清理。公開分支隨本批合入main，遠端HEAD在推送後核對。

最後核對2090個本地連結、六個README保存正對照、圖片與JSON／語法通過；273條教訓及151個guard保持。原生祖先槽、來源JSON及凱麗GIF的SHA不變，未改Go或既有字串審查綁定。
兩輪來源、程式與攻擊選擇相同；按鍵共139／143次，開場跳過與戰後關框ACK依當下觀測不同，不宣稱逐鍵或逐幀重播。清除442715472bytes暫存，另將21587111bytes候選與工作母稿同SHA連到正式素材。工作根無新增root-owned或誤建md目錄，FD2本批容器均已退出。

## 2026-10-20 弓手87戰鬥HD與成功掉落訊息觀測

從main 5983b085建立獨立分支，依固定盤點與原版來源核對FIGANI261待機4張、262攻擊15張、263零header。61先記READY，再以內建imagegen繪製七張透明母稿，完整提示、來源與拒收稿保留在主收據archer87_battle_followup_20261020。
原始262:9全空白，獨立native-empty縮放群組避免320×200空矩形壓縮薄飛矢。首版release／shot身高不連續而拒收，第二版按原窄矩形重繪；兩個待機各有1個高覆蓋像素的取整裁切，其餘17幀為0，未改母稿alpha。

完整19張同次Draw、離場F2、64／65 DAC與一次續行通過。本體13張、飛矢5張與原生空白1張分開核對，十二項fixture反例與十六項編製反例通過；先前鎧甲武士14張END回歸保持。普通擷取補收departure，flow分組嚴格要求END攻擊與我方射擊時的目標待機；sage只選適用的灰綠源色，原有1000點／alpha250與RGB門檻保持。

前兩輪普通GUI在戰後Z選人失敗，保留OBSERVATION-ONLY。僅等待class_ui_job的首修不足：成功掉落的rewardMessage另在nativeSystemEndTurnUI等ACK，既有death_reward_ui只指滿背包互動。先補READY與實作前紅色回歸，再加兩個唯讀audit欄位並驗六狀態綠色回歸。GUI只依真實owner正常Return，不改遊戲規則、排程、背包或保存，不增加Z重試或跳掉落。

同來源seed1、希莉亞own5／target87、至少兩次END的第三輪完整GUI通過，敵方射擊與我方普通射擊87後正常掉落ACK、返回、F2及章節邊界保存／新程序LOAD通過。章前強化與原有高等級明示，章內未注入HP、位置或狀態；不新增自然難度、傷害、AI、原版parity或整章E2。16筆掉落owner觀測中8筆等ACK，攻擊後六次Z均不在掉落／class UI／經驗訊息期間。本體22張含17 END／5待機，灰綠組14張含9飛矢／5待機，十五項普通反例拒收。保存只重建章節邊界，不宣稱戰中戰利品持續。

正式登錄403項，原401項保持；FIGANI30資源225幀及四召喚37幀，來源408仍阻擋。正式Go四項加兩子項零SKIP，清冊十一項通過。私人allow-list完整3647檔、runtime-only3329檔逐筆SHA及UID核對，原3621檔保持，新增七母稿不進執行包。正式private_root保持既有prototypes。
Go修改後重生字串盤點SHA與舊綁定相同，5205項文字及99項人工處置保持，八項字串回歸通過。
README新增兩張未編輯的瞄準／飛矢實際視窗，凱麗並排GIF與文化入口保持。56／57／58／61及主收據同步。速度藥水成功掉落訊息與頭像仍是像素，列入#204，其他人物／施術／死亡／普通召喚與平台仍待，完整HD Goal及#263保持開啟。

清除561321299bytes的本輪程式、Go快取及runtime-only檔案副本，保留manifest；另將25185426bytes的33份同SHA候選與工作母稿連到正式素材。全部截圖、來源、收據、失敗觀測與拒收稿保留，沒有建立錄影。

私人庫已獨立提交推送並核對實際遠端HEAD，保持private。GitHub #201／#263／#204已更新，原驗收條件保留，遠端重拉後22項、人工14項、可能完成0。274條教訓與152個guard通過。六份入口1998個本地連結、六個README文化入口、66份主收據引用SHA、三份Python語法及四份JSON通過；來源槽／JSON與凱麗GIF不變。全部18個變更檔UID／GID1000，工作根無root-owned或誤建md目錄，Go格式保持。
本輪FD2容器均已退出。公開驗證批次將合入main並核對遠端HEAD；完整戰鬥HD仍未完成，不啟動matching或Release。

## 2026-10-20 突擊騎兵83戰鬥HD驗證中

從已合併main fa959343建立獨立分支。沿規格閘門、正常玩家試玩、imagegen與文件職責路由恢復現況；遠端待辦22項，工作樹原只有work未追蹤。固定盤點確認249待機1幀、250攻擊11幀、251零header。61先寫READY，再以內建imagegen繪製七母稿；完整提示及SHA已存主收據knight83_battle_followup_20261020。

首版重擊／四收勢的雙腿太寬、腳底太低，槍尖與腳底間距不足，完整排版拒收。另製原幀單張參考，第二版依原收腿姿勢重畫，腳底約y150、槍尖169／171；未修改母稿alpha。待機／蓄勢共用聲明保持，重擊金光與三段槍尖殘光分開繪製。第二版12幀高覆蓋裁切全0，完整同次Game.Draw 256 HD／3忠實／329最終回讀、三階段F2、64／65 DAC與一次續行通過。12張人物及1張重擊金光符合核心，兩組各5項圖源反例、16項編製反例通過；尚未正式登錄，候選盤點32資源237幀，來源408仍阻擋。

普通GUI首輪--assets誤指runtime/assets，形成assets/assets路徑，屬呼叫環境失敗，未冒稱完整保存成功。第二輪修正為runtime，正常三次END後希莉亞(10,24)距83號(11,15)/(13,15)仍遠，正式可達集合沒有射程2..3的位置而依上限拒收。先追加READY，再只將控制器明示輪數選項擴0..6，預設0與min<=max守衛保持。第三輪固定min4／max6、同來源seed1，章內無狀態注入，正在普通接戰與保存冷讀驗證。

第三輪第四次END已取得250:0..10的實際HD PNG。實際普通視窗名稱為突擊騎兵，與同圖希莉亞受擊；這是玩家可見名稱，不延伸成原始欄位ABI語意。六次為固定停止線；取得普通完整收據及圖源反例後才登錄正式清冊與私人包。完整HD Goal仍開啟，其餘人物、施術／死亡／普通召喚及平台仍待。沒有新增錄影或修改正式Go規則／排程／保存。

第三輪完整GUI已通過：五次END的物理場景數依序0／4／3／5／6，最後希莉亞LV33、HP275，從(9,21)移至(9,22)射擊(8,20)的83號。來源與兩主題保存bytes一致，章節邊界新程序LOAD單位一致。本體40張含37 END／3待機，金光8張含3重擊／5待機，兩組各15項普通反例拒收；不把8張全算金光或宣稱唯一相位。這份軟體GPU抓圖耗時1180.62秒，不作平台效能基準。
正式登錄405項，原403項保持；唯一盤點為FIGANI32資源237幀及四召喚37幀，來源408仍阻擋。正式Go三項加兩子項零SKIP，清冊11項通過。私人完整3666檔與runtime-only3341檔逐筆SHA／UID核對，原3647檔保持，新增七母稿不進執行包。正式根保持prototypes。README新增兩張未編輯的普通蓄勢／重擊圖，凱麗GIF保持；56／57／58／61及主收據同步。本批Go來源與字串審查綁定保持，只擴測試控制器輪數選項。
清除459754412bytes程式、Go快取及已驗執行包副本，保留manifest；另將27846395bytes的26份同SHA候選／母稿連到正式素材。全部截圖、原來源、完整Draw、失敗觀測與拒收稿保留；沒有新增錄影。

私人素材已獨立提交推送，實際遠端HEAD核對相同並維持private。GitHub #201／#263保留原驗收條件並更新，重拉後22項、人工14項、可能完成0。275條教訓與153個guard通過。六份入口2007個本地連結、六個README文化入口、68份主收據引用SHA、Python語法與四份JSON通過；來源槽／JSON與凱麗GIF保持。素材檢查image沒有Git，Go來源不變的核對改由主機必要Git操作取得空差異，再用原容器乾淨重跑其他檢查；不是產品缺陷。
14個公開變更檔及19個新正式素材UID／GID均1000，工作根沒有root-owned或誤建md目錄。本輪FD2容器均已退出，公開批次接續合入main並核對遠端。完整戰鬥HD Goal仍未完成，沒有matching或Release操作。

## 2026-10-20 普通指令0人物HD

先核對主線41222e5b、正式清冊及普通法術consumer。未登錄90號三組新圖前，先補普通指令0的原生前導／施術／目標／尾段HD接點。沿既有command0 IDA與typed producer，先記READY與來源SHA，再附加來源，不改MP／HP／RNG、音效、Draw確認或保存。原版zero-header fallback由載入器回傳實際resource，悠妮29回退28，不拿29猜素材。

兩側完整GPU 953案例通過；完整Game.Draw 74 HD呈現、69張最終1920 PNG、五階段F2、MP一次／HP七次與一次續行通過。缺件／F2原子性已驗。首個fixture缺flash table、第二個GPU fixture未持有LUT屬測試建構問題，補齊既有fixture後同環境乾淨重跑；未變動原版或正式規則。

首章普通GUI r1套物理移動後開法術，命中既有移動後禁用規則而拒收。沿既有lesson與0x18A5F caller定位，READY後只改控制器為原地候選、dist5預篩再由正式目標格接納；r2同未強化普通START／F5來源及seed1，最多一次END，沒有章內注入。首輪觀測保留，不冒稱通過。普通施法、保存冷讀及最終來源稽核仍在進行；90號新圖及完整HD Goal保持未完成。

第二輪普通GUI完整通過，501.43秒、一次END，悠妮原地對(6,21)盜賊施放指令0，MP8至6、敵HP28至0，正常返回與F2。來源未強化，沒有章內注入。22張普通PNG中9張人物核心通過；完整fixture27張核心及五階段F2保持。F5保存bytes與原來源相同，新程序LOAD重建MP8及敵HP28，不宣稱戰中狀態持續。README加入未編修art04視窗。

首輪完整Go在4GB／2CPU及大量HD可選案例下達600秒時限，未列通過。依既有腳本改為8GB／4CPU、原版唯讀locales overlay，20個Go套件乾淨通過；專用HD測試沿獨立收據。字串盤點5205項、99處置保持，只改綁定SHA。

反例命令曾誤傳不存在inventory，舊harness把共同FileNotFound當六項拒收，該輪以invalidated.json及主收據明確撤回。正式harness現先跑有效正對照、逐項核對預期原因，fixture與普通各六項通過，不把路徑錯誤當產品守衛。既有四召喚148張由最終核對器回歸通過。新教訓與守衛已登錄。

本切片CONFORMED-SCOPED／RUNTIME-E1；正式清冊及私人素材完全未變，沒有新增90號圖組。完整HD Goal與#201／#263保持開啟，其他普通指令、角色、VFX／死亡／普通召喚及平台仍待。沒有錄影、原版oracle重跑、matching或Release。

遠端#201／#263已更新並重拉，22項、人工14項、可能完成0。11項清冊測試、三份Python語法與六份入口2019個本地連結通過，276條教訓及153個既有guard保持。新反例教訓以實際收據驗證，現行教訓檢查器刻意排除test_檔，未擴張該工具。私人庫未改動、乾淨且遠端仍符合既有同步；沒有新素材需同步。
清除本輪生成程式、Go快取及原版語系overlay，共386783527bytes。保留全部PNG、完整收據、原始來源及拒收紀錄；工作根沒有root-owned或誤建md目錄。主線提交前最後核對所有權與差異。

## 2026-10-20 BattleFig90 三組戰鬥HD與普通指令4／5

從602098b7新建分支接續。270／271／272固定原始來源確認1／11／11幀，先寫READY再加入明示with-cast及分層配方。首版施術展臂過寬、兩次全景火種位置不符而拒收；改以內建imagegen分開產生人物與獨立火焰，七份接受母稿保持原始RGBA，依原右側Mask位置編製23幀。R3高覆蓋裁切全0，21個編製守衛通過；舊83／133／134全部影格收據相同，舊兩組16項與idle-only20項守衛通過。

普通4／5沿既有typed producer附加同次呈現來源，不重走亂數。共用0的前導／actor／四張尾段，4／5的front、target、boundary及tail保存原層序、base、索引與水平位移。1,123完整GPU案例通過；0／5完整Game.Draw分別74／69 HD繪製、五／七階段F2、MP1與HP7／6及單次續行通過，272全11張均有最終RGB核心證據。缺件／F2原子性、兩組各七項反例通過，舊首章正常指令0六項及83十二張回歸通過。

物理完整221 HD／289最終Draw與64／65 DAC通過。271:4的藍袖被fixture前景228:0遮住，核對器原未排除前景而失敗；沿既有施法來源遮擋方法，核SHA、clip及至少1000可見核心，12張全通，八項錯配／未知前景／錯前景圖與黑圖拒收。首個錯前景反例改到未消費背層，已保留並改取正向報告真正排除的來源，沒有改遊戲或fixture。

20套件標準Go回歸通過；99字串審查處置保持，只重綁來源。初次Go編譯發生在新增測試檔期間，固定來源後乾淨重跑通過，不列產品缺陷。

正常GUI首輪在戰鬥前停止，修正原觀測器只接受物理END樣本的收尾。第二輪三次原地END仍未接敵，90號守原位且MP55，停止無效等待。第三輪用正式可達格前進、兩次END後即因射擊可達而出手，完整返回／F2／保存冷讀走完，但沒有END施法，整份收據拒收。第四輪先補READY，仍用同源、seed1、min2／max4與正常前進／待機，須抽到指定figure的施法才進射擊，目前驗證中。所有失敗觀測保留；尚未正式登錄90素材與私人包，不宣稱整章E2或完整HD。

第四輪完整正常GUI已通過，615.70秒，三次前進格12,30／12,26／10,25與三次END；90號正常指令5消耗MP55至40，再接希莉亞普通射擊、返回、F2及章節邊界新程序LOAD。實際PNG核對4張施法與9張受擊待機，七項正常反例拒收。來源及兩主題F5 bytes保持，冷讀重建原入口，並非戰中狀態持續；沒有新增原版oracle、自然難度或整章E2。

正式登錄270／271／272，原405項保持，唯一盤點為408項、FIGANI35資源260幀及四召喚37幀，來源408仍阻擋。正式Go四實測與兩子項通過；全地圖因少環境變數SKIP後獨立補齊重跑通過。排除母稿的runtime-only又實測四項、零SKIP，11項清冊契約通過。原始20個固定檔案全部SHA保持。

私人完整3696檔、runtime-only3364檔逐筆SHA／UID／GID核對，原3666檔保持；新增七母稿排除執行包。私人獨立提交已推送並核對實際遠端HEAD，儲存庫仍private。README新增未編修的正常魔法師施法圖，凱麗並排GIF保留；56／57／58／61同步。GitHub #201／#263重整為目前範圍，保留原驗收條件與開啟狀態；遠端重拉22項、人工14項、可能完成0。

新增正常取樣停止條件教訓。完整HD Goal保持開啟，其他人物、普通指令／效果與平台仍待。本批沒有Release、matching或新增錄影。

清除本輪已驗程式、專用Go快取、原版語系symlink覆蓋層與runtime-only副本共504763222bytes，另將64260738bytes的同SHA成品／母稿連至正式素材。所有截圖、收據、來源與拒收稿保留，runtime manifest保留；沒有錄影。277條教訓及154個guard通過。

收尾核對60份主收據來源SHA、七份入口2888個本地連結、六個README文化入口、七份Python語法及32檔UID／GID通過；凱麗GIF與HEAD的blob完全相同。正式素材根保持，原20個固定檔SHA保持，工作根沒有root-owned或誤建md目錄。所有本輪Docker工作已退出，公開批次接續推送主線並核對實際遠端。


## 2026-10-20：93號戰鬥HD來源與READY

由origin/main ad6b100c建立feat/sage93-battle-hd-20261020-r1。固定盤點核279／280共六幀與281零header。13／14的現行caller明確走地圖回復演出，修正前置計畫中把它當近身施術銀行的假設；不新增不相干的HD owner。三個原始可見姿勢先製作可丟棄手繪候選，普通受擊及完整Draw通過後才正式登錄。主收據sage93_battle_followup_20261020為唯一當前切片。

93三份透明手繪母稿編製六幀，原始重複姿勢明示共用，六幀高覆蓋核心裁切全0。完整Draw為202次HD／308次final，三階段F2、64／65 DAC及單次續行通過；六張最終PNG與五項fixture拒收通過。編製器十六項守衛通過。普通GUI首輪因相對out導致PulseAudio設定路徑失敗，遊戲未啟動；改絕對容器路徑，沿同來源seed1乾淨重跑通過，不列產品缺陷。三次進軍／END後希莉亞射擊(7,22)僧侶，正常返回／F2／保存與新程序LOAD通過，兩張目標待機核心通過。明示target模式只證受擊待機，十二項反例與83既有四十張普通流程回歸通過；敵方揮杖與回復HD未冒稱普通驗收。

新兩項正式准入後，完整與runtime-only各三項Go測試加玩家／mode11子項均實測通過，零SKIP；清冊十一項通過。本輪無Go來源修改，標準全套沿ad6b100c原二十套件證據，不重跑無關全套。完整私人allow-list與runtime-only逐檔SHA／UID／GID保持，舊檔與二十個原始固定檔SHA一致。README加入未編修普通僧侶待機視窗，凱麗GIF不改。唯一當前覆蓋數量依58；private Git、議題、最終連結與清理另於本段追加。

私人素材庫已獨立提交推送並核實際遠端HEAD，保存庫仍private。遠端#201／#263保留原驗收條件，更新本輪受擊限定範圍並保持OPEN；提交前快照重新由主機gh拉取，22條／14項人工／0可能已完成。清理已驗執行檔、快取與runtime複本共475971072 bytes，核SHA後去重15026730 bytes；PNG、收據與失敗日誌保留，本輪沒有錄影。

最終驗證通過：34份SHA引用、七份文件2140個本地連結、0個README錨點、六項文化保存入口、兩份Python語法與19份UID／GID；正式舊408項entries與凱麗GIF保持。Docker本輪容器已清空，未更換工具鏈映像。


## 2026-10-20：洛娜5號戰鬥HD READY

上一批df2ee6b4已交付僧侶六幀。由最新main建立洛娜分支；原15／16共21幀與17零header核SHA。既有正常LOAD確認5號是洛娜，早期暫定haw5工作目錄已改正，未把猜名寫進正式圖源。先記READY再擴展既有編製器的明示兩銀行分層；不改遊戲、正式資料格式或規則。

洛娜十一份母稿編製原21幀。碎光兩稿alpha均254，回查發現既有母稿契約max>=250與中間層255不一致；停止重畫，補READY後只讓分層中間層沿來源契約，最終完整PNG在out建立前仍要求0／255。母稿不改alpha，新缺最終核心實檔拒收通過。新分層21項守衛，舊僧侶／魔法師／村民33張PNG位元組與57項守衛保持。完整Draw為278次HD／377次final、三階段F2及64／65 DAC、單次續行；本體21張與前刺兩張暖亮核心分開通過，各五項反例。無真實遮擋的首輪proven守衛StopIteration不列成功，改原始無排除核對。

普通首輪因測試控制器將自己原格排除而停在第三回合15,22。補READY及兩個候選條件後，相同來源seed1、min2／max4重跑，實際兩次END後原地攻擊83，HP168至54，正常返回／F2／F5與新程序LOAD通過。本體35張、弧光與碎光各一張通過，兩組18項普通反例拒收。章前HP+1000與既有等級明示，章內無注入；F5只證章節邊界，不談戰中HP持續。原16:3／5鞋底重取樣各裁5／6像素，保留揭露，不稱零裁切。正式兩項asset entries與候選相同，私人allow-list舊檔／原始20固定檔SHA保持；沒有Go來源或正式規則修改，沒有新錄影。README加入未編修普通前刺PNG，凱麗GIF保持。

正式完整與runtime-only各三項Go測試加兩子項零SKIP，清冊十一項通過。本輪Go來源未改，不重跑繼承的二十套件。私人素材已獨立提交推送並核實際遠端HEAD，維持private。遠端#201／#263更新本輪範圍，原驗收條件不改且保持OPEN；主機gh重拉快照為22條／14人工／0可能完成。移除本輪可重建產物490026967 bytes，核SHA去重99198840 bytes；沒有新錄影，PNG與失敗日誌保留。Docker本輪容器已清理；公開交付HEAD另存lorna5-public-git-proof-r1.json。

最終核對46份SHA引用、七份文件2149個本地連結、六項文化入口、三份Python語法與45份UID／GID通過；全部21張最終alpha0／255、原410項entries及凱麗GIF保持。原格攻擊與兩個議題驗收區塊亦逐筆核對。


## 2026-10-21：共用事件死亡演出HD

本批開始於10月20日，基底5912da2b。先核既有0x1DB65證據與原FDOTHER5項68..79，再於61定READY。加入可選LMI效果清冊與獨立來源／SHA／尺寸驗證，四份imagegen原始RGBA編成十二張。13個人物方向、A累積六張、B逐張還原inactive底圖六張及steady保持。HD來源在同一次typed producer預檢，F2只重建繪圖計畫，不重跑規則或亂數；原indexed buffers、等待、音效位置、rollback及一次續行不改。普通擊殺及狀態致死consumer未由本批擴充。

初輪GPU流程flags通過卻被fixture黑DAC遮成黑圖，獨立PNG重組拒收。修測試色盤後定位前25拍現行搬移目的為VGA0,0，最後steady才4,4；HD誤套穩定框造成24 HD像素錯位。補READY勘誤後修逐拍viewport，原像素路徑保持。這是現行producer對照，不新增DOS原版目的位址或PLAYER-E2。Pillow直通alpha在重疊煙霧有78點超門檻，回查Go1.22 drawNRGBASrc預乘原碼後修驗證公式，沒有改圖或放寬RGB門檻。所有拒收與取代輪次保留。

可見診斷、26張Game.Draw與最終1920回讀、四階段F2、原子預檢及一次續行通過。獨立來源重組及八項GPU反例、編製器八項與清冊八項反例通過。正式完整包與不含母稿的執行包各完成26拍，PNG bytes完全相同；完整20套件Go通過，99項字串處置重綁後保持。原20份固定輸入SHA不變；私人3753檔／runtime-only3403檔逐筆SHA與UID/GID通過，舊3737檔保持。私人18b0af20已推送，實際遠端HEAD相同且仍private。

正常世界回歸首輪未帶進軍參數，以預設0輪無射程而停止；未修改遊戲或控制器。沿上一批洛娜own10、同來源seed1、min2/max4/advance重跑，正常近戰、返回、F2及章節邊界新程序LOAD通過。本體42張及弧光16:14一張核對通過，本輪沒有取得16:15普通樣本，不補造另一相位通過。章前HP+1000祖先與章內無注入保持揭露；本回歸沒有觸發章節死亡事件，不能代替該事件正常玩家驗收。

README僅新增明示合成測試場景的未編修畫面連結，凱麗GIF與文化保存內容保持。58、57、61與索引同步；#201/#263保持OPEN，原驗收條件不改。主機gh重拉工作清單為22條、14人工、0可能完成；教訓282條、158守衛通過。本輪不錄影，已移除可重建執行檔、Go/GUI cache及runtime-only實體複本，保留PNG、槽、收據與拒收紀錄；清理量見主收據。公開提交與實際遠端證明另存本批Git核對檔。

收尾再次核對輔助底圖writer：事件前25拍保留seed，最後steady才可重畫ChapterAux。將排除範圍限制於前25拍，並以完整26拍對照驗證事件忽略壞aux宣告、最後steady仍拒收。最終GPU的26張PNG與已獨立重組版本bytes相同，四階段F2與一次續行通過；最終20套件Go再跑通過，主套件204.823秒。字串處置仍99項，盤點SHA保持；最後的重建cache也已清除。正常GUI屬此事件限定修訂前的普通路徑回歸，沒有將它冒充章節死亡事件驗收。

## 2026-10-21：第23章正常死亡HD與章24邊界冷讀

基底32166fbf；不增素材、不改死亡規則。先在61定READY，加入唯讀的最後實際Draw幀、相位、來源及目標觀測，不以Update下一索引冒充畫面。來源從第23章既有原生建構槽boost-only AP+5000／DP+300／DX+60／MaxHP+1000，保留原傷亡、旗標、位置及裝備。原生標題LOAD、整備必選24與正式選人出發後F5；最終工具再跑所得JSON bytes與最初來源相同。

最終固定FD2_SEED=1，正常Z選蓋亞、七次移動／END、一次清除阻擋敵軍，在20,18攻擊20,17隊長。忠實主題完成缺HD頭像的死亡文字，事件啟動後正常F2切HD，A／B取樣後再切回，普通Return確認勝利，戰後自然進章24整備。整備F2前後F5 bytes相同，新程序標題LOAD回相同節點。章前強化及九名實際活躍出戰者如實保留，不稱自然難度、完整章E2或新DOS parity。

早期單調距離進軍被敵軍擋住；加入正常清障後六END仍不夠，依實際可達集合補READY到有界十END，最終只用七次。第三輪已取得A／B但漏按勝利Return而停止等待；第四輪在補固定seed前停止，均保留部分證據，不列正式PASS。沒有注入狀態或修改遊戲來過關。

正常視窗十七張取樣，六張A／B的不透明來源核心符合；候選14的兩側觀測幀相同但實圖核心不符，拒收並保留。初次驗證器侵蝕遮罩會把分散alpha255全刪光；核現行整數HD繪製與來源alpha後補READY，改驗全部不透明點，沒有改美術或放寬RGB3／至少100點。黑圖、錯SHA、缺B、錯origin／viewport／來源順序、章內注入及缺續行八項拒收通過。完整26拍GPU與獨立透明重組通過，PNG與前批逐位元組相同；完整20套件Go通過，主套件218.443秒，99項字串處置及盤點SHA保持，20份原版固定檔不變。

README新增未編修正常章23事件PNG，既有凱麗原版／HD GIF與文化內容保持；57／58／61及主收據同步。完整HD仍未完成，普通擊殺／狀態致死consumer及其他人物／指令／效果繼續列待。GitHub工作議題、提交遠端與清理核對於同批收尾記錄。

收尾：遠端#201／#263內文與verify.note同步，原acceptance逐字保持且仍OPEN；主機gh重拉快照為22條／14人工／0可能完成。教訓283條、158守衛通過；7份文件2422個本地連結、29個SHA引用、四份Python語法、六個文化入口及凱麗GIF保持。最終來源工具SHA與兩份正式GUI收據相同，原生來源重生F5 bytes相同。移除本批可重建binary／Go／GUI cache共252541478 bytes，沒有新錄影，PNG、槽、收據與拒收日誌保留。Docker生命週期與公開遠端HEAD另以提交後收據核對。

## 2026-10-21：普通行動與狀態致死HD consumer

基底a14ebaab。先核110與既有IDA9.4固定SHA的1DB65 caller、active／HP0閘門及狀態1A941順序，再於61定READY。普通物理入口及共用ApplyHPDamage正值到0邊界保留實際原+5，供私有呈現恢復；兩個Unit私有欄位不序列化，JSON bytes與原HP／+5 writer保持，復活與收尾清除。原有行動State／Unit指標、HP／RNG／獎勵保持；新增同步呈現前預檢，結束只續行一次。離屏與早已inactive者不插空演出。

狀態phase沿原HP／inactive／倒數交易，在全部扣血訊息後、到期提示前接同一26拍，不建立killer或死亡獎勵。預檢需覆蓋訊息聚焦後的視圖：私有State／緩衝沿原單格游標step推演，不發正式focus trace。與真實focus的view／HUD／work／VGA相同，缺LMI在HP交易前拒收。普通行動的拒收位於已結算邊界，不聲稱回滾此前HP／RNG；原版逐拍raw並未由此補證。

初輪完整Go只剩AI指令10測試失敗：手工fixture只走原指令owner，沒有後接死亡owner及明示靜音。測試補正常Draw與一次續行／RNG檢查，未為通過測試退回空呈現。補共用HP writer後20套件通過；私有聚焦預檢補強後，最終20套件再次通過，主套件239.264秒，99項字串處置只重綁座標。正式caller、法術來源、JSON不變、狀態訊息交接、缺件原子性、指標與無trace差分皆通過。

action／status／script各26張完整Game.Draw、四階段F2及一次續行通過。最終78張PNG與收據和各自獨立重組基底bytes相同，原script26拍亦與前批保持。GPU及正常PNG核對器各九項錯誤拒收，包含錯consumer。status GPU採明示typed phase建構，正式扣血訊息caller另驗，不推定正常玩家路徑。

正常GUI沿既有章23原生LOAD／整備出發所得F5來源，固定seed1、七次普通END及清除一名阻擋敵軍後，蓋亞在20,18攻擊20,17隊長。新增unit_death_kind區分action與script；只抽第一次普通擊殺，再以忠實模式正常走文字5／事件53、勝利確認與戰後至章24整備。十五張取樣三張A18／B19／B20不透明核心符合；三份其他效果候選跨拍或不足100點不放行，人物／透明尾段不由此證明。F2前後F5 bytes相同及新程序標題LOAD通過。章前強化與既有傷亡明示，章內無注入，不新增自然難度、原版oracle或整章E2。

README改用未編修的普通擊殺視窗，舊章節事件圖與凱麗GIF保持；57／58／61／索引與主收據同步。沒有新素材或私人庫提交，原413項catalog SHA保持。剩餘HD、狀態正常GUI及平台效能仍待；GitHub議題、提交與清理於同批收尾核對。

收尾：#201／#263內文及verify.note同步，原acceptance逐字保持且仍OPEN；主機gh重拉22條／14人工／0可能完成。教訓284條、159守衛通過，新守衛以正式main.go作正對照、移除呼叫的私有副本作反對照。七份文件2428個本地連結、六份Python語法、45項SHA引用、20份原版固定檔與99項字串處置語意保持；README標題與文化尾段、凱麗GIF及舊事件圖bytes相同。移除本批可重建binary／Go／GUI cache共511978682 bytes，PNG同SHA去重44572823 bytes；無新錄影，所有PNG路徑、槽、收據與拒收日誌保留。Docker及實際遠端HEAD另於提交後核對。

## 2026-10-21：機甲兵111戰鬥HD

基底d36339a4，另建分支。先核固定FIGANI333四幀／334十幀及335零header，於61定READY。沿核准手繪風格生成五份採用RGBA母稿；原大型圖為造型主證據，不用FDICON同號圓柱頭猜身份。開砲初稿砲口過靠右，保留拒收並以v2縮回腰前。完整14張按原矩形、delay及raw marker分層編製，共享姿勢明示；機體核心零裁切，兩張光束依原右邊界截斷42791／23545高覆蓋像素，不稱零裁切。

完整Game.Draw為216次HD／334次final，三階段F2、64／65 DAC及一次續行通過。首輪未排除真實228受擊者前景而拒收；沿固定SHA／mask排除後，仍至少1000點與原RGB門檻，本體14張及光束三張暖亮核心通過，兩組各八項拒收。暖亮點分別包含機體外1897／466／25個光束點，不能由相同機體代替。負向工具首輪漏傳inventory的呼叫錯誤保留，未改門檻。編製器21項、清冊12項及284條教訓／159守衛通過。runtime-only各三項Go加兩子項與完整14張GPU通過，PNG與完整包bytes相同；正式路徑另兩項加兩子項零SKIP。本批Go未改，完整20套件沿d36339a4，99項字串處置保持，不冒稱全套新跑。

普通首輪沿最近111在17,29，兩END後蓋亞19,29沒有可達相鄰格，控制器正常停止。補READY後明示row-max25選上方同圖組，不改單位或地形。第二輪四END後攻擊、返回、F2及新程序LOAD已走完，因沒有111主動END圖源仍拒收，不補造PASS。第三輪同來源、seed1、row-max25、min4／max6，以既有target範圍完成正式GUI：四次END抵20,23攻擊上方111、自然返回、F2／保存冷讀通過。26張抽樣中六張符合目標待機，十二項普通拒收通過；共享母稿只列相容集合，不判唯一相位。主動開砲普通路徑仍待，完整開砲限fixture RUNTIME-E1。來源章前AP+5000／DP+300／DX+60／MaxHP+1000與原傷亡明示，章內無注入；F5只重建章節邊界，不保存本次戰中HP／位置，不新增原版parity、自然難度或整章E2。

正式新增兩項，原413項entries保持；實際覆蓋依58及主收據formal_coverage。私人完整3772檔、runtime-only3417檔核SHA，原3753檔保持，拒收稿與原版擷取不混入。私人60aecfb6已推送並核實際遠端HEAD及乾淨工作樹，未改保存庫公開或權限設定。原20份固定檔SHA不變。README加入未編修普通機甲兵視窗，凱麗GIF與文化入口保持；57／58／61與主收據同步。#201／#263原驗收字串不變且保持OPEN，主機gh重拉／純函式重建工作清單為22條／14人工／0可能完成，狀態標籤一致。

清理本輪可重建檔案505896017 bytes，另以硬連結去重PNG 31307366 logical bytes；沒有新錄影，來源／母稿／PNG與拒收紀錄保留。Docker收尾與公開實際遠端HEAD另存本批Git核對檔。

## 2026-10-21：珊14號戰鬥HD

基底2400e21c，從最新main另建分支。正常LOAD確認identity14／BattleFig14／名稱珊，沿固定42四幀／43十幀與44零header先定READY。五份採用母稿保留背向左視角、藍色長髮與低束髮尾、綠褐服裝、紫頭金飾法杖。高舉初稿把杖放在頭左側，前揮初稿過寬，均保留拒收；以單一原姿勢圖修正後編製。近似姿態共用明示，ready／swing等比係數0.9582765／0.968319，全部14張高覆蓋核心零裁切，不稱原輪廓逐像素相同。

生成預覽外圍看似有彩色光暈，四處實測A0，白底合成確認無可見光暈；不是以RGB判alpha或量化母稿。三稿maxA254沿既有母稿門檻，編製後全部14張仍有255核心。候選文字曾把header byte4=6誤稱投射物；查現行nativePhysicalSoundBank的音效來源consumer後在發布前更正，原資料及Go保持，不新增原版語意。兩項教訓分別併入既有alpha規則與新增caller守衛。

完整Game.Draw為225次HD／336次final，三階段F2、64／65 DAC及一次續行通過。14張同次最終1920回讀的藍髮核心核對通過，五項GPU拒收、編製器16項及清冊12項守衛通過；無前景排除或RGB門檻放寬。runtime-only三項Go加兩子項、完整14張GPU與原包PNG bytes相同，正式兩項加兩子項零SKIP。本批無Go來源、規則與保存格式修改，完整20套件沿d36339a4及99項字串處置，不冒稱本批全套新跑。

普通章23用既有F5來源與固定seed1，正常Z選在場序號6珊，兩次移動／END抵21,29，以原射程攻擊23,29的111。20張演出取樣中五張視窗符合人物核心，九項普通拒收、自然返回、F2與章節邊界保存／新程序LOAD通過。共享姿勢只列相容集合，不判唯一相位。章前AP+5000／DP+300／DX+60／MaxHP+1000及原傷亡明示，章內無注入；F5不保存本次戰中HP／位置，不談自然難度、原版傷害、音訊或整章E2。狀態追蹤確認珊HP1309保持，沒有把HUD外觀當作傷害或反擊證據。

正式兩項加入前的415項entries保持；實際覆蓋依58及formal_coverage。私人3791檔完整、3431檔runtime-only核SHA，舊3772檔保持，拒收稿與原版擷取不混入。私人16e5dc97已推送並核實際遠端HEAD、乾淨工作樹與UID/GID，未改公開或權限設定。原20份固定檔SHA不變。README加入未編修普通揮杖圖，原有凱麗GIF與文化內容保持；57／58／61及主收據同步。#201／#263原驗收字串不變，主機gh重拉／純函式重建工作清單為22條／14人工／0可能完成，狀態標籤一致。本輪285條教訓／160守衛通過。其他人物、指令與人物外效果、普通111開砲、狀態正常GUI及平台效能仍待，完整HD未完成。

清理本輪可重建檔案509877375 bytes，另以硬連結去重PNG 22951404 logical bytes；沒有新錄影，來源／母稿／PNG與拒收紀錄保留。Docker收尾與公開實際遠端HEAD另存本批Git核對檔。

## 2026-10-21：約拿／希爾法戰鬥HD與換手觀測

基底6d905f4d，另建feat/hd-cleric-party-battle。沿固定盤點與正常LOAD身分先定READY，再補約拿63／64十幀與希爾法72／73七幀；65／74零header不造施術。七份透明手繪母稿逐張核對原寬袍、背飾／金披領與環首杖，原始RGBA不改。近似姿態共用及明示等比係數保留，全17幀高覆蓋零裁切，原矩形、header、delay、hit／sound marker與保存格式保持。精確配方、原／成品SHA與private母稿名已嵌入主收據cleric_party_battle_followup_20261021。

兩位各完整Game.Draw、三階段F2、64／65 DAC與一次續行通過；灰綠／金色服裝核心全部17張通過，不擴稱武器或半透明全像素相同。完整／runtime-only最終PNG bytes相同，兩位候選與執行包各三Go測試／兩子項、正式各兩測試／兩子項零SKIP。編製器各16項、完整GPU各5項及普通GPU各9項拒收、清冊12項與路線參數3項守衛通過；無前景排除或RGB放寬。

約拿沿既有AP+5000／DP+300／DX+60／MaxHP+1000來源，三次正常END後由21,22近戰20,22的111，四張實圖核心通過。希爾法前四輪高AP會反擊清掉近身目標或首領，分別保留116缺頭像、提前勝利、黑畫面及唯讀版後續缺116的非PASS紀錄。只一次從同一未加值祖先槽用受版控boost-only建構AP+0，其餘強化與原傷亡保持；constructor seed8、game seed1。26筆原生欄位及正常native LOAD／整備必選24／出發／F5所得JSON只差AP／BaseAP 5000。第五輪不限目標列走到已知側翼死路，嚴格停止；第六輪恢復既有row-max25，同槽與seed1，三次正常END後由18,25近戰18,24的111，九張實圖核心通過。兩位均自然返回、F2及章節邊界F5／新程序LOAD；章內無注入，不宣稱自然難度、傷害、原版parity或整章E2。

第三輪黑畫面另列#267：第4回合class_ui_job仍在，AI與transient_ui為false，根因未知。只增唯讀job幀／Draw確認／DAC欄位及不改owner測試，沒有改Draw或演出路由。第四輪frame6836跨過該點但後續缺116仍拒收，第六輪換了章前AP來源，不構成原黑畫面已修復；#267維持OPEN。20套件Go回歸及唯讀測試通過。116故事HD頭像仍沿#201追蹤。源擷取曾因命令使用相對輸出路徑令PulseAudio無法建立config；改絕對路徑、同image乾淨重跑通過，屬環境命令問題，沒有列成產品缺陷。

正式421項、實際FIGANI覆蓋依58及formal_coverage；原417項entries保持。私人3815檔及runtime-only3448檔核SHA／UID／GID，原3791檔保持。私人提交ff9168864342f3f3f57a411d5ba8c56b24132714已推送並核實際遠端HEAD與乾淨工作樹，保存庫權限未改。README加入兩張未編修正常近戰視窗，原凱麗GIF與文化保存bytes保持。56／57／58／61及工作清單同步，#201／#263原驗收不變，#267保留未完成；遠端23條／15人工／0可能完成。本輪286條教訓／160守衛通過。其餘人物、普通指令／效果、召喚、普通111開砲、狀態正常GUI及平台效能仍待，完整HD目標保持active。

清理本輪可重建檔案639308108 logical bytes，PNG硬連結去重43195800 logical bytes。沒有新錄影；固定母稿、來源、PNG及失敗紀錄保留。公開推送與Docker收尾的實際結果另寫本批public-git-proof.json。

## 2026-10-21：第23章黑畫面紀錄勘誤

前段「約拿／希爾法戰鬥HD與換手觀測」將第三輪寫成持續卡死，判斷錯誤。原始JSONL於frame7636的job=true，7638已false，8734已進第5回合，10078停止於116故事HD頭像缺圖；原日誌是RuntimeError而非END逾時。新唯讀稽核器核對完整單調幀序、釋放、後續回合與日誌終態，七項測試通過。原日誌SHA與上一輪保存值相同，舊觀測檔末尾也含該終態；未綁frame的黑PNG不足以定位owner。原圖／收據／日誌不改，主收據ui_black267_correction_20261021取代舊分類，#267按誤判撤回，不宣稱修復引擎或舊run通過。

## 2026-10-21：機甲隊長十一幀與故事頭像

基底679be580，沿固定348／349十一幀及DATO116原四來源先定READY，再以imagegen內建工具繪六份透明戰鬥母稿與一份不透明故事母稿。長面甲、藍球肩、背鰭、綠垂布與前臂雙刃保留；原RGBA不改。兩次灰霧去背稿及雙刃過長／過短稿拒收，r1有39個高覆蓋邊緣裁切；r2擴錨點後全十一幀零高覆蓋裁切。姿勢共用、斬光分層與等比係數明示；原header、矩形、delay、hit／sound marker與350零header保持。完整Draw與runtime-only各三項及子項零SKIP，十一張最終PNG bytes相同，全部alpha≥250來源像素獨立核對。21項編製、5項錯圖、10項正常GUI收據拒收與12項清冊測試通過。正式r1命令漏FD2_ASSET_PACK造成preflight環境SKIP，補正同image後r2兩項／子項全跑；未列產品缺陷，Go遊戲程式未改。

正常章23沿既有AP+5000／DP+300／DX+60／MaxHP+1000與原傷亡來源，啟動前game seed1。蓋亞七次END、一次普通清除阻擋敵軍後攻擊20,17的隊長，三張目標候選有一張348待機2546點通過。戰後speaker116完整480頭像230400點RGB差0，F2像素模式與還原均擷取；自然進第24章整備、F2/F5 bytes相同與新程序標題LOAD通過。章內無注入，隊長主動攻擊正常GUI未提升，不稱自然難度、傷害、原版parity或新E2。

正式425項、唯一工具覆蓋依58；原421項及3815檔保持，新增20檔同步私人庫，完整3835檔與runtime-only3461檔核SHA／UID／GID。私人cdb9373897942ce74e422fad989bf2d037eea709已推送，實際遠端HEAD相同且工作樹乾淨，未改權限。README加入未編修正常目標與故事視窗，凱麗GIF保持。56／57／58／61、#201／#263及工作清單同步；兩項HD議題維持OPEN，#267按前段勘誤撤回。其餘人物、效果、指令、普通召喚與平台效能仍待，完整HD目標保持active。

本輪收尾：22條遠端未完成項、287條教訓／160守衛通過。本輪沒有錄影；移除已驗binary、Go cache及runtime複本共529777535 logical bytes，核SHA後去重39288369 logical bytes。原始母稿、PNG、收據與失敗日誌保留，Docker批次均使用--rm。


## 2026-10-21：萊汀與索菲亞戰鬥HD

由16c0423a主線建立feat/hd-next-party-battle。固定FIGANI18／19與33／34原33幀、20／35零header，先定READY再用內建imagegen重繪。十一份接受母稿原始RGBA保持；萊汀厚背甲、紅角、裸臂、藍垂布與月牙斧，索菲亞金馬尾、紅背衣、紫褲與寶石杖依原圖定位。塵屑／斧弧分層，原矩形、header、delay及音效marker保持。雲霧待機、像素塵屑、裁寶石與過寬揮杖候選拒收保留；最終高覆蓋裁切全0，21／16個編製守衛通過。

完整及排除母稿的runtime-only各兩組完整Game.Draw、三階段F2、64／65 DAC與一次續行通過，零SKIP；全部33張PNG的alpha≥250核心獨立核對，兩包最終PNG bytes相同，各五個GPU錯配反例拒收。正式四項准入後，兩人物各兩項Go載入／原子預檢與兩子項實測，零SKIP；清冊12項通過。Go遊戲來源未改，沿基底既有20套件證據，未重跑或冒稱本批全套。20個固定原始輸入SHA保持。

第12章兩份正常抽樣在首次END後護送友軍HP0並回標題，兩份保持非PASS。原KeyError為控制器在章節終態讀native_round，現改明確拒收；沒有修改NPC或正式遊戲。第11章歷史槽與基底SHA在646份現存SAV均無相符，首次建構缺路徑屬輸入問題，保留日誌。沿試玩入口重選實際ch08來源，用受版控typed8～10章handler新建chapter10槽；constructor seed8、每章升6級、HP+1000與祖先政策完整揭露，不冒稱舊槽boost-only複本。

新來源正常標題LOAD／variant2城鎮出發／章首F5通過。game seed1，萊汀兩次END後在23,23攻擊24,24的102，索菲亞五次END後正常近戰102；各自自然返回、F2、來源／兩主題F5 bytes及新程序LOAD通過。29／37張實際PNG核對圖源、各九項反例拒收。章內無注入，F5只重建章節邊界，不談戰中傷害／位置持續、自然難度、原版逐幀或整章E2。

正式清冊及覆蓋數量由工具重生，唯一現況依58與本批主收據northern_party_battle_followup_20261021。原425項及私人3835檔保持；新44檔allow-list同步後逐檔SHA／UID／GID通過，runtime-only3494檔與正式檔相同，新母稿不進執行包。私人ca78cf22已獨立提交推送並核實際遠端HEAD，未變更公開設定。

README新增未編修普通斧擊／揮杖視窗，凱麗並排GIF與文化保存內容保持。56／57／58／61同步；#201／#263更新並保持OPEN、原驗收條件相同，主機gh重拉22項、人工14項、可能完成0。288條教訓及161個guard通過。本批沒有錄影、matching、Release或新dosgolem收據，完整戰鬥HD Goal仍開啟。

清除本批已驗執行檔、專用Go快取及runtime-only PNG副本共545900614 bytes，核SHA去重47125345 bytes；母稿、截圖、原始槽／JSON、收據及拒收日誌保留。最終入口／來源SHA／所有權與Docker清理檢查接續記錄，公開提交與實際遠端另存本批Git證明。

最後核對97份SHA引用、八份入口3068個本地連結、三份Python語法及58檔UID／GID通過；原3835檔與凱麗GIF保持，README基底內容逐字保留。工作根沒有root-owned或誤建md目錄，本批FD2容器已清空，工具鏈映像未改。公開批次接續合入main並核實際遠端。


## 2026-10-21：豹形獸人102／103戰鬥HD READY

基底77b02272，新建feat/orc-battle-hd。沿固定原始盤點核306／307與309／310共35幀，308／311零header。保留低伏、跳出畫面、三張全透明等待、俯衝與回跳裁切；來源錨點別名沿既有schema1，沒有改格式或Go遊戲。102橘褐／褐布與103金橘／藍布，各待機／跳躍／俯衝，加獨立beam及spark共八份接受母稿，原始RGBA保持。

初先把生成預覽的外圍RGB誤當可見光暈，去背未解決預覽；回查既有alpha教訓後，白底合成及直方圖證實該區A0，撤回光暈分類。四份未採用跳躍稿max254，最終取樣未實測，不推定必然編製失敗。本批固定採無地面陰影的回跳參考所生成的第三稿；其原始alpha含255，人物編製通過，不改來源alpha。103首俯衝稿伸直腿左右相反而拒收，v2沿102正確姿勢與103色系重生。全部舊稿保留。

第一版接觸圖顯示消退相位零散亮點被總外框連成亮條，配方拒收。READY後改按固定峰值色集合及八鄰接元件定位；第二版細長母稿縮至1×1時低於中間250核心，建立out前正確拒收。補獨立緊湊spark母稿，門檻不變；第三版兩組35幀編製通過。307:11／310:10是跳起3的y62起57列，310:9對應3前119列；307:10只差一原像素。腿部局部顏色比對僅作幾何推論，不稱全圖相同。原本出框的四相位裁切明示，其餘人物保持完整。

正常來源沿上批第11章F5 JSON及原章前建構／等級／HP+1000政策，game seed1，章內無注入。索菲亞／凱麗分別正常接敵102／103，最多六END；完整Draw／F2／35張來源、32張非空核心／三張空白及runtime-only、正常攻防返回／保存冷讀正在驗證。尚未正式登錄四項或同步私人庫，不宣稱通過。56／57開頭仍寫116頭像缺件的舊斷言已依現有captain116收據更正，沒有重開該研究。完整HD Goal保持開啟，沒有Release、matching、新原版oracle或錄影。


獸人切片驗收：正常兩份GUI皆PASS，各五END，索菲亞進軍5,10／7,11／7,14／5,15／7,16後近戰102，凱麗27,10／27,15／25,18／22,20／25,22後近戰103。自然返回／F2、來源及兩主題F5 bytes／新程序LOAD通過；22／14張全不透明核心符合，兩側各九項反例拒收。只核到待機、102跳起及出框局部，普通完整相位不外推。

完整GPU首輪因啟動器先建out而在測試Mkdir拒收，兩個空目錄及原log保留，非產品缺陷。R2第一次Docker--name誤放image後，Python用法檢查退出，Go尚未開始；參數修正後同image／候選新out重跑完整GPU，未改Go。完整及runtime-only各兩組Game.Draw／三階段F2／64及65 DAC／一次續行通過，35張最終PNG bytes相同；32張非空全部alpha≥250來源核心符合，三個固定Mask及成品全空另驗。各五項GPU及21項編製守衛通過，正式載入／原子預檢各兩項與兩子項零SKIP，清冊12項通過。Go來源未改，沿基底20套件證據，不冒稱本批全套重跑。

正式四項准入，覆蓋由工具重生，唯一數量依58。私人原3879檔保持，新53檔allow-list與runtime-only3529檔逐筆SHA／UID／GID通過，20個固定原始輸入SHA保持；十八份來源別名對應八份原始母稿，原RGBA不改。README加入兩張未編修正常待機視窗，凱麗並排GIF與文化保存內容保持。私人Git、遠端議題、最終連結及清理另在本段追加。

私人91c93f86已獨立提交推送並核實際遠端HEAD，保存庫未改公開設定。289條教訓及161個guard通過。清除本批可重建執行檔、專用Go快取及runtime-only PNG副本共554671000 bytes，核SHA去重90312705 bytes；包含35張執行包GPU重複圖的原路徑仍可讀，沒有錄影。母稿、原始槽／JSON、完整與失敗收據保留；本批FD2容器已清空，最終入口、遠端議題及公開Git檢查接續記錄。

遠端#201／#263已更新並核OPEN、原驗收條件不變，主機gh重拉22項、人工14項、可能完成0。最終核對107份來源SHA、八份入口3074個本地連結、三份Python語法及66檔UID／GID通過；清理後70張完整／執行包Draw PNG雜湊保持，原3879檔與凱麗GIF保持，README基底逐字保留。工作根無root-owned或誤建md目錄，工具鏈映像未改，公開提交與實際遠端另存本批Git證明。完整戰鬥HD Goal仍開啟，沒有matching或Release操作。


## 2026-10-21：普通法術1人物HD READY

基底4fd467e9，新建feat/hd-command1。58已閉合的0x2A6BD→0x262EF與正式indexed producer作為來源，不重新逆向。只在同次合成保留來源供HD人物繪製，FDOTHER19／21保留像素。規則、RNG、聲音與保存格式不變；規格61先列READY，接續驗整張來源重組、原子預檢、完整Draw及正常GUI。完整HD Goal保持開啟。


指令1執行期驗證：1265個GPU畫布含新增142個雙側／雙目標及boundary來源核對通過；完整Game.Draw五階段F2、MP1／HP8／一次續行與七項GPU錯配拒收通過，包含實際回退28的全部施術來源。0／1／5原子預檢及20套件Go通過；99項字串人工處置重綁，集合與各處置數量保持。正式資產433項、57組411幀未增加，本批重用既有圖。

普通輸入首輪在遊戲啟動前被舊CLI組合限制拒收。第11章指令1第二輪與指令0第一輪皆在兩次END後，悠妮7,18的直線策略遇地形局部最小值而拒收；不是產品崩潰，未取得施法。回查試玩入口及實際audit後，第三回合合法集合含10,19，規格先補READY，工具增加明示路線7,15／7,18／10,19。每格先核正式可達，再正常移動待機／END，下一回合才原地施法；原失敗保持，正常收據仍在執行。

Go直接啟動的login PATH與預設快取兩項環境錯誤已分類，改保留映像PATH及本人工作快取後乾淨通過，沒有改產品邏輯。沒有新母稿、私人素材修改、錄影、Release或matching工作。


普通路徑驗收：兩份皆三次END，明示7,15／7,18／10,19繞路後，悠妮正常取消並重新選法術，在原地施放指令1與0回歸。自然返回／F2／兩主題保存bytes／新程序LOAD通過；施術者核心10／4張，各六項反例拒收。F5只重建章節邊界。README新增未編修指令1畫面，原文化保存內容與凱麗GIF保持；56／57／58／61同步。完整HD Goal及#201／#263仍開啟，清理、遠端與最終檢查接續記錄。


遠端#201／#263已核OPEN、原驗收條件不變。主機gh重拉22項，人工14項、可能完成0；291條教訓及162個guard通過。私人91c93f86與實際遠端一致，本批未新增美術同步。清除本批已驗執行檔、Go快取與舊盤點用來源共301242744邏輯bytes；所有正常／拒收畫面與收據保持，未錄影。最後連結／SHA／所有權、Docker清理與公開Git核對接續記錄。


最後核對43份來源SHA、八份入口3081個本地連結、三份Python語法及24檔UID／GID通過；README基底逐字保留，凱麗GIF雜湊保持。原版包manifest與來源盤點SHA保持，兩份正常收據與全部來源均對目前程式。本批工作根沒有root-owned或誤建md目錄，FD2容器已清空，三個維護中工具鏈映像未改。公開提交與實際遠端另存本批Git證明。完整戰鬥HD Goal仍開啟。


## 2026-10-21：普通法術2／3／7／8人物HD READY

基底04ccee09，建立feat/hd-ordinary-2-3-7-8。58已有四項indexed演出閉合，依規格61先列READY，再將同次producer的底圖、人物與效果位置接入HD。四者保持自身預算、數值／RNG／音效及保存格式，2仍用既有palette entry1；6的packed變色另需視覺契約，不在本批忽略。正常名冊具備第11章悠妮指令2與第23章約拿指令8，3／7的普通來源尚待確認。完整HD Goal保持開啟。


實作與第一階段驗證：2在實際compositor回傳層序，3／7／8在同次typed sequence保存來源，原數值／步進函式未改。新增626個雙側／合法多目標／boundary來源，連既有案例合計1891張GPU畫布通過；四種缺件／F2原子性通過。7首輪因共用fixture只有class2抗性而於數值前拒收，診斷確認缺class7、銀行及狀態保持；恢復正式抗性全表並按完整plan.Results核資源，未移除目標或改範圍。初次trace插入缺換行被gofmt拒收，修正後獨立格式與編譯通過；早期internal套件通過不當成cmd已編譯。

四命令完整Game.Draw及各階段F2通過，2／3／7／8分別MP一次、HP6／13／5／16、一次續行；7實際多目標轉場亦通過。GPU施術者核對增加opaque模式，驗全部alpha≥250可見點，至少1000點與RGB門檻保持，不用服裝色域或抽稀。四組各七項fixture反例通過；99項人工字串處置重綁，集合及各處置數量保持。

使用者暫停後恢復，2026-10-21改排程為先確認`signal: killed`，再完成全部HD串接，最後集中人工確認回報。先前普通2／8的正常GUI及兩組核心拒收已通過，3／7正常GUI延至集中確認；不再逐人物插入長篇進軍。全套R1在4 GB條件下cmd/fd2被killed、其餘19套件通過，沒有OOM事件證據，尚不能判定記憶體不足。新增tools/fd2_go_test_memory_probe.py，於相同4 GB／2 CPU條件重跑，保存cgroup memory.events／peak、程序RSS及Go JSON事件；先取原因再決定修正，不直接調大限制宣稱修復。規格61已先記錄READY排程，執行期來源於診斷期間凍結，所有舊收據與未提交變更保留。

OOM根因已證實：唯讀查主機單一kern.log後取得2026-10-21T10:05:26.867298+08:00的MEMCG事件，核心明記殺掉fd2.test PID2134426，原R1 log mtime為02:05:27.249478 UTC。僅保存該事件必要摘錄與SHA，未保留其他專案日誌。相同4 GB／2 CPU受控R2再次oom_kill=1、exit1，沒有測試斷言失敗；峰值達配額。8 GB／2 CPU R3無OOM，惟650秒總時限到期時新測試才開始0秒、正在PNG解碼，該輪仍列非PASS。按標準8 GB／4 CPU、1200秒只重跑cmd/fd2，763.752秒通過，wrapper總769.987秒，記憶體峰值7558815744 bytes、oom_kill=0。頂層930通過／71專用旗標略過，連比較輪其他19套件的通過證據形成全套回歸；本批完整GPU另有實際執行，不拿略過當繪圖通過。沒有改Go產品來躲OOM，也不據此宣稱正式遊戲記憶體無缺陷。

標準tools/remake_go_test.sh沿8 GB／4 CPU，補20分鐘Go時限、1260秒外層及5秒終止寬限，最後保留Go失敗碼，避免無具名FAIL的SIGKILL被誤當成功。基線解析移入既有Go Docker映像，主機不再執行Python。新probe已驗真實逾時子程序群組清理exit124與拒絕覆寫證據exit2。驗證命令一度誤指/tmp/nonexistent，屬命令錯誤；改用實際檔案後bash -n、timeout工具與AST通過。測試檔文字guard被現有掃描器刻意排除，移除該無效guard而保留原子預檢及完整Draw證據，293條教訓／163個有效guard通過。

同批補BattleFig110，規格先READY，原330／331／332三組4／12／8幀。生成一張六姿勢圖板，保留藍色壯碩身軀、黃眼、肩突、灰腰布與蝠翼；六張來源只做原RGBA矩形切分。R1下排490切線混入鄰圖翼尖而拒收，由alpha投影確認455..485透明間隔後改480；R2把332重複幀推為330來源，被首幀assert拒收。回查固定inventory逐筆SHA，332:0..7確為331:0..7，先勘誤spec再編製R3，不另造施術。最終24張沿原描述子、header／delay／raw markers，六姿勢共用及12個高覆蓋裁切像素明示；來源alpha0..254保持，最終PNG仍有255核心，未改alpha。

110候選完整Game.Draw通過：物理16張、三階段F2／64及65 DAC／一次續行；普通7施術八張來源、多目標轉場、各階段F2／MP1／HP5及一次續行。獨立全部alpha≥250核心核對物理16張與施術21張取樣通過。正式登錄三項並保留此前433項與24張GPU已驗PNG hashes；只另把完整原始RGBA圖板加入source_files，連六張裁切母稿依白名單同步私人庫。舊3932素材逐檔保持，新完整包3963檔、runtime-only3553檔均核SHA／UID；唯一覆蓋由工具重生，數量依58。正常與造型人工確認仍依全部HD接入後集中排程，未宣稱本人物普通流程或新原版oracle。正式載入、私人Git、遠端議題及最終檢查接續記錄。

正式及runtime-only各兩個頂層／四個子項載入與原子預檢通過，零SKIP；正式圖源與已驗GPU PNG hashes保持。私人6cb283ebd99d84471e1ca57e4210cd95a9e1d164已獨立提交推送，真正遠端HEAD相同，保存庫仍private。#201／#263保持開啟及原驗收條件，#4／#202已記錄使用者先全部HD、後集中人工確認順序；主機gh重拉22項、可能完成0。最終核42份來源SHA、五份Markdown的2115個本地連結、30檔UID／GID、原433項清冊與固定原始manifest及凱麗GIF保持；Python AST／shell語法通過，工作根無root-owned或誤建md目錄。未新增錄影，沒有Release或matching操作，FD2容器已清空；公開批次提交與實際遠端另核，全部HD工作繼續。

## 2026-10-21：普通法術6 HD READY

上一批公開90200c3669662486c8b4ad1ee79acceebc66eaa3已快轉main並推送，實際遠端HEAD相同；工作樹僅work未追蹤，無FD2容器。從main新建feat/hd-command6-palette，沿既有工作根建立command6-hd-r1。先回查規格路由與58／指令6CONFORMED主證據，不重解負列、座標、數值或RNG。

規格61先達READY：同次producer來源保留前導target早於效果的層序、可選target與packed值；現行7＋12N＋9(N−1)＋7及StageBases的LUT15保持，不另加尾段。HD變色沿既有地形四近鄰RGB色差插值，保留alpha，標明modern-art palette approximation及相同RGB不同原索引的歧義。只作用target，原生indexed公式與母稿不改；待整張來源重組、七shade GPU、完整owner與F2／一次結算驗證。效果32／33與人工確認另列，尚未接入正式HD呈現。

實作與驗證：同次producer記錄front／target／transition／tail來源，packed僅作用target，快取鍵包含原圖SHA、palette、可見原索引與shade。188張來源重組及八項獨立色差／快取GPU案例，連既有2087張畫布通過。完整Game.Draw有91張HD Draw、實際七種shade、各階段F2／MP1／HP5／RNG與一次續行；原子缺圖及非法shader／guide拒收通過。既有兩項需原版槽或正常GUI旗標的測試略過，不混入實際GPU通過數。99項人工字串處置重綁，集合與各處置保持。

施術者核對首輪因尾段FDOTHER32遮擋而拒收，並非產品GPU斷言失敗。核對器原本只支援FIGANI；加入固定原版manifest核metadata／mask SHA的FDOTHER遮擋，已登錄現代效果則核PNG SHA與alpha。保留RGB／alpha≥250及至少1000點門檻。R2有效基底及八項錯範圍、缺結算、缺七色、缺F2、黑圖、缺actor與缺施術幀均正確拒收。反例工具的錯指令原固定6會對本命令無作用，改為不同於目前命令；七色拒收另核專屬原因。六項直接控制驗opaque／透明／clip、未知archive、竄改mask拒收及HD效果alpha，受版控工具為tools/test_fd2_compound_occlusion.py；舊普通7的27張施術核心亦通過。R1日誌保留，不放寬門檻。

本批重用既有素材，私人清冊與版本保持。正常操作及人工確認留待全部HD串接後；效果32／33與其他HD缺口仍開啟。來源與工具、收據SHA集中在既有HD主收據ordinary6_hd_followup_20261021，沒有新原版oracle、錄影、matching或Release操作。

最終核12份來源SHA、2050個本地連結、24份變更檔UID／GID與三份Python語法通過；本批工作根無root-owned或誤建md目錄，無殘留FD2容器。遠端重拉22條工作項，可能完成0，完整HD及集中人工確認保持未完成。公開提交與遠端驗證另存本批Git證明，私人素材未變。

## 2026-10-21：普通法術HD效果入口 READY

法術6公開80b89acdf5b1dd47a44bdbbe697022aea7aad370已快轉main，推送與實際遠端HEAD相同，工作樹只有work未追蹤。從main建立feat/hd-ordinary-effect-banks，重查規格路由與現行loader後確認battle_effect_hd_set只允許召喚65～68，普通owner預載也只收FIGANI。本批先依既有typed選擇器核普通0～8的18個效果資源，規格61達READY後擴充准入與兩銀行原子預載；不改原版規則、演出、保存或美術。全部HD及人工確認仍未完成。

依READY擴充十八項普通效果專屬consumer／cycle，沿相同描述子及SHA驗證；未知FDOTHER仍拒收，召喚65～68契約不變。四類普通HD幀集保存全相位來源，含pulse，依archive收實際人物／效果並共同stage後publish。普通0～8啟動與F2均走此入口，保存與原動畫規則不變。

Go回歸277.455秒通過，八個頂層與42個子項、零SKIP。九種普通owner注入最後一張效果缺件，兩銀行及既有快取圖保持，MP／HP／Acted／RNG及owner不變；修復後兩銀行發布並淘汰舊銀行，F2失敗及成功維持交易。新效果PNG僅為測試故障替身，未加入清冊，也不當美術完成。舊普通0／1／5、2／3／6／7／8及四召喚人物預載回歸通過；十八項准入、錯consumer／描述子與十四項越界資源拒收通過。Python以既有清冊可丟棄副本驗54個准入／拒收案例，正式清冊不變；字串盤點SHA未變，99項人工處置及對應Go測試0.218秒通過。繪圖器未改，不重跑先前2087張GPU來累積相同證據。

同時依固定9／10原四加六幀與11零header準備斧兵素材，先在規格61列READY。內建image_gen沿原版姿勢與既有手繪風格生成三姿勢／兩斧芒圖板，原RGBA0..254複製進工作區；尚未編製或正式登錄，沒有造型人工通過聲明。提示與參考圖角色、工具、來源及SHA保存在本批figure003-generation-r1.json，私人庫尚未變。

本批最後核10份來源SHA、2050個本地連結與24檔UID／GID，Python語法通過；293條教訓／163個guard通過，遠端工作清單22項且可能完成0。原始包及正式清冊未變，沒有新增錄影；FD2測試容器已退出，工具鏈未重建。公開批次提交與實際遠端另核，私人庫未變。

## 2026-10-21：斧兵BattleFig3正式HD串接

普通法術效果入口公開3dbc5792105f7e2d736b65aa4e7cdc9a251133d9已快轉main並推送，實際遠端HEAD相同；#263已回填兩批HD與集中人工排程。從main建立feat/hd-axeman003，沿上一批既有工作根，不新開同義目錄。規格61先有READY圖源契約，原9／10共十幀、11零header，不重解規則。

R1第三姿勢為橫向跨步，不適合原72×109窄框；沿同一角色與原版收勢生成獨立R2，披布垂直且雙腿靠攏。已用R1兩身體與兩斧芒、R2收勢僅做矩形原RGBA切分，alpha0..254保持。首份配方錯把有序圖層包成layers物件，既有schema在發布前拒收；R3改為既有清單格式，不改編製器或圖源門檻。原十項header／delay／raw markers逐筆保持；五母稿共用、蓄力8個高覆蓋裁切像素、10:5未知細棕線未複製明示。

候選完整Game.Draw通過，220張HD／3張忠實／347張final Draw；攻擊／反擊及三階段F2、64段戰鬥DAC／65段地圖DAC、HP／RNG保持與一次續行通過。十張最終1920 PNG的全部alpha≥250核心逐項符合，每張至少1000點；21項編製器錯來源／配方拒收通過。正式登錄兩項並保持原436項與3963個素材，新增十PNG、五張已用母稿與獨立R2原稿；含拒收姿勢的整張R1圖板排除。正式與不含母稿執行包各完成載入及我方／mode11缺件原子性驗證，零SKIP、零OOM，最終十PNG bytes相同；Go程式與字串盤點本批不變。

全包3979檔與runtime-only3563檔依白名單同步並核SHA與UID／GID，舊3963檔保持，沒有把原版資料或拒收圖板加入現代包。私人素材提交10029c6f獨立推送，遠端HEAD接續驗證。普通來源與造型人工確認仍待全部HD後集中執行，不新增PLAYER-E2或原版oracle；沒有新錄影、matching或Release操作。另已由固定原版包機械式核十八種普通效果共340幀的PNG／mask／描述子，保存技術參考供下一批重繪，不當作HD美術完成。

私人實際遠端10029c6f8ba1ba196dfd86b5c9a0f50b25ca6861已核與本機相同，工作樹乾淨。公開批次最後核23份SHA、2050個本地連結、8檔UID／GID與新增工具語法；本批工作根無root-owned或誤建md目錄，FD2容器已清空。遠端工作清單重拉22項、可能完成0；未刪除來源證據，沒有新增錄影。正式圖源覆蓋由audit_fd2_battle_hd_coverage.py重生，唯一數量見58及主收據formal_coverage。

## 2026-10-21：十六組普通法術HD特效 READY

斧兵公開762380dcbdd85e264807756da14d20459e7c557b與真正遠端main一致，工作樹只有work未追蹤，從main建立feat/hd-ordinary-effect-materials。原十八組340幀來源無空白mask，只有18／19／20／21各兩張完整mask；最大圖幅270×190。規格61先記錄READY，沿既有召喚的遮罩與材質重定向，先補十六組304幀，不把火龍頭26／27當成材質。保持原矩形、全部描述子及演出，兩材質與公式明示為現代美術近似；全部串接後集中人工確認的順序保持。

兩張不透明暖焰／冷白電光材質由內建image_gen生成，原RGB1254×1254複製進工作區。實作前以白色通道255檢查原提案0.84＋0.32L會把亮材質大量clamp成白色，丟失紋理，因此先修READY為0.72＋0.28L，保留0.15色差；Pillow F通道與獨立float64最多1 RGB捨入差明示。原提案未接正式路徑，不以事後文件替已實作內容補背書。

2026-10-21 普通法術十六組HD材質正式接入：304幀原alpha逐像素相同、34,352個獨立float64 RGB抽樣、16個數值控制、十種編製器拒收、全304來源GPU與七owner完整Game.Draw通過。
七owner共646次HD呈現；各相位F2、MP一次、原HP節拍／RNG／一次續行保持，施術全幀核心與代表特效最終像素通過；錯命令、錯最終RGB及損壞原生圖源均拒收。
正式三父十一子、runtime-only兩父兩子零SKIP；全部有界Go程序無OOM。保持舊438項／3979檔，新增304 PNG及兩母稿；私人與執行包逐檔SHA／1000:1000核對，私人獨立提交2cdbfc4e並核遠端。
材質重定向明示modern-art approximation，不稱304張獨立新畫或原版RGB parity。4／5共用owner以5代表；普通GUI與人工美術依全部HD集中排程，完整HD仍未完成。
首輪控制腳本語法錯誤未啟動編製；r2乾淨通過。私人Git局部快照因尚未寫完被拒收，等待完成重讀後308路徑精確符合白名單；未改產品。較早亮部公式於實作前訂正，只有READY的.72+.28進正式。
火龍閉口／張口及尾焰三張原RGBA已生成，36原幀技術參考有版本化工具重生，原型尚未登錄。來源／命令／限制集中HD主收據，不新增IDA、matching或硬體時序研究；本輪未錄影。

本批最後核50份SHA、2054個本地連結、13檔UID／GID與Python AST；工作根無root-owned或誤建md目錄。私人實際遠端2cdbfc4e已核，FD2容器清空。工作清單重拉22條、要人判14條、可能已完成0條，293教訓／163守衛通過；沒有新增錄影。

2026-10-21 火龍26／27先達READY：36原幀來源已核，三張原RGBA按R3明示矩陣及高覆蓋錨點投影，頭部固定虛擬比例／原框裁切，尾焰填入原框；不改原delay、marker或規則。9／15共用張口、16／17共用尾焰及27鏡射明示，集中人工修正仍待。原型未登錄，接下來編製與必要GPU／交易驗證。

2026-10-21 火龍26／27正式接入：36矩陣重生、十三錯件拒收、全36來源GPU與既有304回歸通過；完整指令2為99次HD呈現，全部相位F2、MP1／HP6／RNG／一次續行及全施術來源核心通過。
最終龍頭特效明示alpha≥250與合成區間核RGB，錯命令／錯最終RGB拒收；預設七owner仍取255，舊七份收據重跑通過。正式及runtime-only各一父一子零SKIP、晚期缺件／F2原子性通過，全部有界Go無OOM。
保持舊454項／4285檔，新增36PNG及三原RGBA；私人4324及runtime-only3903逐檔SHA／1000:1000核對，私人獨立提交7ecebb46已核實際遠端。原型投影不混入私人白名單。
9／15共用張口、16／17共用尾焰、27鏡射及原框裁切明示，正常操作／另一側完整owner／人工美術仍待全部HD集中確認。普通0..8的所選效果圖源已齊，完整HD未完成；其餘人物／地形／HUD等仍待。本輪未錄影、未開新RE或matching。

火龍批次最後核33份SHA、2058個本地連結、11檔UID／GID與Python AST；工作根無root-owned或誤建md目錄。私人實際遠端7ecebb46已核，FD2容器清空；遠端工作清單22項、可能已完成0，293教訓／163守衛通過，未錄影。

2026-10-21 火龍898a0e48已推送且核實遠端；議題留言刪去被自動審核拒絕的私人提交資訊後成功回填公開範圍。從新main開隊伍圖源分支，15／17共34幀達READY；六張原RGBA、鏡射衍生與提示已保存，尚未登錄。保留規則與原描述子，普通GUI與人工確認維持最後集中，本輪未錄影。

隊伍圖源R1技術預覽發現15金光方向不符：原圖是左尖右展，誤用了鏡射母稿；R1不登錄。R2改用同向原RGBA，規格先訂正再重編，既有原始描述子與門檻保持。

2026-10-21 隊伍15／17正式接入：全34幀Game.Draw核心、227／229次HD、各相位F2、64／65 DAC、HP／RNG與一次續行通過；來源拒收各21項。正式與runtime-only各兩figure的兩父兩子零SKIP，均無OOM。保持舊456項及4324檔，新34PNG加七母稿依白名單同步，拒收鏡射原型排除。46最後三幀各裁八個高覆蓋像素逐項披露，全部HD及人工確認仍待；未錄影。臨時路徑替換／Docker外層逾時包裝與零裁切誤斷言均屬腳本問題，修正後沿同工具鏈乾淨重跑，不記為產品缺陷。

隊伍批次收尾核42份SHA、2060個本地連結及7檔UID／GID；各五項最終GPU反例通過，工作根無root-owned或誤建md目錄。工作清單重拉核對，私人獨立提交與實際遠端HEAD已確認；無非預期FD2背景程序，未錄影。

2026-10-21 從64b2da33新main建立蓋亞手臂修正分支；確認待機／蓄力母稿的腕拳銜接不自然，#268追蹤。兩張局部修正版原RGBA與提示已保存，90／91十五幀達READY，十幀更新、末五幀必須不變；普通及人工確認集中最後，未錄影。同時準備BattleFig22後續圖源，尚未登錄。

2026-10-21 蓋亞腕拳v2正式接入：十五幀實際Game.Draw核心、217次HD、F2／64及65 DAC／HP與RNG／一次續行通過；十幀更新、末五幀PNG相同，十六項編製拒收及正式／runtime-only各兩父兩子零SKIP通過，無OOM。十二新檔替代私人現行清冊十二舊檔，其他4353檔保持，舊版留Git與本機證據；私人獨立提交及實際遠端已核。#268留待集中人工確認，舊普通收據不替v2背書；未錄影。

蓋亞修正收尾核24份SHA、2071個本地連結及10檔UID／GID；工作清單23項、要人判15項、可能已完成0，294教訓／163守衛通過。工作根無root-owned或誤建md目錄，無非預期FD2容器；未錄影，#268留待集中人工確認。

2026-10-21 蓋亞修正4b76fc78已推送且核實遠端。武者22全21幀達READY：六張原稿及七旋轉衍生、35原生分離FX元件已定；兩母稿原型拒收，色差集的白色漏取／灰衣誤取在DRAFT R1／R2訂正，R3原圖框線已核，尚未登錄。保持原框、節拍與規則，正常／人工最後集中；未錄影。

武者首個編製候選主光束過細、偏低，未登錄；原RGBA裁切保留發射球、截短左段以接近2.27原框比例，R4規格先訂正再重編。16殘光沿完整母稿，不改alpha、原生元件或時序。

武者封包R1保留非PASS：來源載入73.09秒通過，第二預檢仍在原素材NRGBA.At迴圈，210秒Go總時限中止；峰值1042329600 bytes、無OOM。按同image／8GB／4CPU／同測項乾淨R2重跑，只延長Go420／probe450／外層960秒，不改production或驗收。

武者封包R2勘誤：Docker在198秒收到signal15、exit143，早於420秒Go及960秒容器時限；最後cgroup樣本無OOM。發送來源未知，不猜測成因、不登記PASS，runtime尚未啟動。R3沿同image／8GB／4CPU及相同測項，按兩個父測試拆成各自有界容器，正式及runtime-only合計四個命令；兩包仍各兩父兩子零SKIP才可通過，不再調整產品或斷言。

2026-10-21 BattleFig22武者正式接入：FIGANI66／67一加二十幀，68固定空header；六原RGBA、七弧光旋轉衍生與一光束局部裁切均保留來源。R1差集漏白色、R2亮色誤取衣物的DRAFT及過寬掌擊／六弧整板不登錄；R3固定三十五元件，R4只修主光束比例與16殘光來源。21幀最終Game.Draw核心、284次HD／3像素／352最終繪製、F2／64及65 DAC／HP與RNG／一次續行通過；128.626秒、峰值1939435520 bytes、無OOM。二十一項編製拒收及正式／runtime-only各兩父兩子零SKIP通過。新增35檔，舊4365檔保持，私人獨立提交及實際遠端已核；完整4400檔、runtime-only3958檔。無Go、規則或存檔變更，未錄影；僅RUNTIME-E1，集中人工及正常玩家待全部HD後。

武者接入收尾核52份SHA、2073個本地連結及10檔UID／GID；工作清單與教訓守衛通過。工作根無root-owned或誤建md目錄，無非預期FD2容器；未錄影。#263現況已更新，完整HD與集中人工確認保持未完成，#268保持開啟。

所有權範圍勘誤：本輪收尾的『工作根無root-owned』僅指ordinary-effects-r1工作根及本批十檔／新素材。全庫find另見本批未修改的root-owned舊檔，例如remake/web/wasm_exec.js及docs/data/command_labels.json；未因本次HD工作修改這些舊檔或全庫chown。所有FD2一次性容器已退出。

2026-10-21 從ec03a4cf建立騎龍20分支，清冊462項且60／61未登錄，23個遠端工作項／15人工／0可能已完成已核。六張新母稿與固定原圖十八幀完成READY；六幀裁切RGBA全同、一幀差31像素只用幾何。DRAFT差集排除鋼甲灰及衣物綠，暖色二十元件框線已核；保留原框／節拍／規則，正常與人工依原排程最後集中。本輪沿既有Docker image，未錄影。

騎龍R1預覽主斬光太細，候選不登錄；保留舊驗證作歷史。READY追加寬光母稿與兩份原RGBA裁切，R3只換61:8三層來源，二十原生矩形、人物及時序保持；重跑新候選的完整Draw與來源檢查。

2026-10-21 騎龍20十八幀正式接入：六人物／飛龍姿勢及寬斬光七原RGBA、旋轉光與兩主閃光裁切保持。R1細光候選不登錄，R2新PNG全十八幀最終核心、219次HD／3像素／321最終繪製、F2／64及65 DAC／HP與RNG／一次續行通過，65.752秒、峰值1237807104 bytes、無OOM。21項來源拒收及兩包各兩父兩子零SKIP通過；八份來源別名折回原母稿，新28檔、舊4400檔保持；私人獨立提交與實際遠端已核，完整4428／runtime3976檔。無Go／規則／保存變更，未錄影；正常與人工最後集中。

重跑命令勘誤：武者22舊主收據誤寫test_fd2_character_battle_hd.py需--layered，實際argparse沒有此參數；既有成功命令未傳，工具依baseline自動決定。只訂正重跑說明，編製器build仍需--layered，舊驗證與遊戲行為保持。

騎龍20接入收尾核54份SHA、2075個本地連結及9檔UID／GID；工作清單與教訓守衛通過。本批ordinary-effects-r1工作根無root-owned或誤建md目錄，無非預期FD2容器；未錄影。#263現況已更新，完整HD與集中人工確認保持未完成，#268保持開啟。

2026-10-21 龍人16／26完整32幀接入：六人物姿勢及獨立弧光原RGBA保持；49:6與79:5原mask空拍保留四／六等待單位。原0拍與各idle0相同，其餘比例裁切不聲稱exact。R1斬光方向反了而停止剛啟動的fixture，該容器143退出，不列產品缺陷或PASS；R2從父稿裁切／180度整數轉置修正，人物與原節拍不變。兩組各21項來源拒收、30個非空最終Game.Draw核心與兩空拍、各適用F2、64／65 DAC、HP／RNG與一次續行通過，正式／runtime-only共八命令零SKIP。記憶體[{"figure": 16, "seconds": 58.185, "peak_bytes": 1391833088, "oom_kill": 0}, {"figure": 26, "seconds": 62.73, "peak_bytes": 1062273024, "oom_kill": 0}]。新41檔沿白名單同步私人庫，舊4428檔逐項保持，完整版4469／runtime4008檔；七新原稿與兩衍生均保存，私人獨立提交實際遠端一致。無Go／規則／保存變更，未錄影。完整HD、正常操作與人工造型仍待，#263及#268保持開啟。

龍人16／26收尾核65份SHA、2077個本地連結及9檔UID／GID；工作清單與教訓守衛通過，ordinary-effects-r1工作根無root-owned或誤建md目錄。#263已更新，完整HD與集中人工保持未完成；未錄影，無非預期FD2容器。

2026-10-21 使用者追加Zeliard式上方主題工具列與城鎮／商店HD，先保存18／19全部已生成圖稿，該批尚未正式登錄。新工具列在640×400外加24高，1920×1272視窗保留1920×1200遊戲區與原F2；觸控扣除上方偏移，主題請求不越過既有阻擋。六場景沿原FDOTHER11／61／62、12／29／63重繪，店員128／130／132從HD場景原目的矩形取80／480配對；舊頭像接入物品／裝備及共用狀態轉場。城鎮遮罩與成功效果以原生blit雙底色重播取opaque write，無RGB猜測。參考圖首次預覽為灰階索引，改依FDOTHER0取色後才生成；v*255//63比正式bit replication至多一級，明示美術參考近似。首批102例已通過，補十二個裝備開窗後新收據114例全部通過；原收據保留。實際滑鼠／F2／F4、十拒收、十一相關回歸、十二Python清冊及正式／runtime兩包載入通過，來源：HD主收據facility_scene_followup_20261021。測試記憶體{"facility-contract-r2": {"seconds": 42.593, "peak_bytes": 939601920, "oom_kill": 0}, "facility-runtime-r2": {"seconds": 62.691, "peak_bytes": 2110095360, "oom_kill": 0}, "toolbar-input-r1": {"seconds": 41.58, "peak_bytes": 1328107520, "oom_kill": 0}, "facility-regression-r1": {"seconds": 4.008, "peak_bytes": 551960576, "oom_kill": 0}, "facility-formal-load-r1": {"seconds": 29.055, "peak_bytes": 572809216, "oom_kill": 0}, "facility-runtime-load-r1": {"seconds": 30.563, "peak_bytes": 546578432, "oom_kill": 0}}。私人4487／runtime4020檔、原4469保持，獨立私人提交與遠端一致；README只新增實際GPU四格總覽。原生框／游標／交易效果仍像素，全部HD、正常與人工確認仍待；未錄影，本批FD2容器已退出。

工具列／設施收尾核50份SHA、2252個本地連結及29檔UID／GID；工作清單與教訓守衛通過，ordinary-effects-r1工作根無root-owned或誤建md目錄。#204現況已同步，其餘HD與集中人工保持開啟；未錄影，FD2一次性容器已退出。文件收尾工具曾誤讀未掛載的/task，修正為/taskout後以同一容器映像與命令重跑；該失敗在任何文件寫入前，屬工具路徑問題。

工具列／設施公開提交960a7693ae86ab40e170c38d35390718169b4b30已快轉本地main並推送origin/main，直接ls-remote與本機HEAD相同。私人素材另有獨立提交且實際遠端一致；#204及#201已回填公開提交與驗證範圍。18／19的草案、施術空拍守衛修改與已生成原圖保留未提交，未混入本次正式登錄；所有本次FD2容器已退出。

2026-10-21 接續960a7693主線完成戰鬥18／19三銀行準備，原生33+27共60幀，未覆蓋上一輪UI。worklist23開啟／15人工／0可能已完成已核。19舉劍V3出現halo且max254，保留拒收；由乾淨Swing重新生成V4，劍縮短並斜舉，max255。R3 disposable幾何以bbox重建像素可能混入未連通點，未登錄；R4改實際8connected集合。本體manual ROI沿clear pose global座標，55／56多處仍有原始RGBA差異，只稱美術近似。wave_tail局部max254揭露並沿中間層契約保留，最終255及GPU1000門檻不放寬；兩版工具輸出失敗只屬研究顯示與原型過嚴斷言，不列產品缺陷。READY已先回填61與HD主收據，再啟動候選編製；本輪未錄影。

2026-10-22 BattleFig18／19接線：保留原六銀行60幀、header／座標／delay／全部marker與56:13透明空拍，十二原RGBA及裁切來源沿白名單保存。候選、runtime各60幀完整1920畫布對CPU Over至多一級；兩組物理Game.Draw、19／8張非空核心、F2、64／65 DAC、HP／RNG與一次續行通過。每組21編製反例、六空拍來源反例、八項兩包載入／物理缺件原子性零SKIP；共用340幀效果GPU回歸通過，無OOM。普通0強套第三銀行的試驗分別因兩／九個原生標記在預檢拒收，兩份失敗收據保留，不改規則或聲稱合法caller。早期只量alpha255的分析忽略既有alpha>=250混色核對，實際非空幀均達門檻，沒有新增profile或降低門檻。新79檔、舊4487檔保持；完整4566／runtime4080檔、十二原稿與全部配方來源核SHA及1000:1000，私人獨立提交實際遠端一致。無錄影、無新Release；全HD、合法第三銀行生命週期與集中正常／人工仍待，#263及#268保持開啟。

2026-10-22 提交前收尾：主收據85份當時可讀來源／收據雜湊已核；工作根與正式素材無root-owned產物或誤建.md目錄。刻意損壞的反例符號連結以lstat檢查，不改反例。GitHub #263保留開啟並更新限制，權威工作清單23開啟／15人工判定／0可能完成，296教訓中164守衛通過。本批FD2容器均已退出並自動刪除，沒有新增錄影。

2026-10-22 發布收尾：公開24b4b1c53c9df20336165f363b0f2af0e7663f83已快進整合main並推送，以ls-remote重讀確認相同；私人獨立提交遠端亦已核。#263已追加公開提交與限定驗證，#268仍OPEN；本輪沒有新Release或錄影。最終docker ps -a篩選fd2為空。此推送後記錄留待下一批實質提交一併保存，不另作純紀錄提交。

2026-10-22 新批次接手：上一輪為有實際登錄與公開／私人遠端核對的progress。本輪從24b4b1c5新分支恢復，保留推送後WORKLOG；無既有FD2容器。遠端23開啟／15人工／0可能完成。固定圖源盤點與原PNG／mask產生7／23／25／27／28／29／31參考，選7與23完整50幀，包含弓手離場、飛矢與原透明空拍；23／71兩零header實檔已核。61與HD主收據先列DRAFT，不把索引或純來源資料推定合法caller。

2026-10-22 飛矢alpha判斷勘誤：兩張新藍光原稿max254，曾未量最終取樣即推定成品不適用，並提出金屬箭尖分層。重查61既有教訓後，依編製器原公式實測16..23八張均有255核心，故取消未實作的金屬尖方案；不用重生或改alpha補值。原稿extrema不等於取樣後extrema，實際完整編製仍須逐張核對。純藍光V2及RGBA局部保留，拒收／被取代稿不進私人白名單。

2026-10-22 7／23配方審查：兩組完整21／29幀候選均通過編製。R1在22:13的細碎blue_tail重採樣alpha206而拒收；僅補resource／frame／pose／alpha診斷，再以既有22白藍光點呈現原細碎光點，門檻不改。R2來源守衛21項通過；同框預覽發現大C被共用調色盤分離截短、22:8細束過長，R3改全掃光人工包絡與寬弧來源、金束RGBA裁切比例。兩組R3預覽已查，不追加逐人物使用者批准；原規劃正常／人工集中最後。13採用原稿、五衍生及全50幀配方先升READY，未接正式清冊。

2026-10-22 7／23正式接線：50幀候選及runtime-only完整GPU對CPU Over、41本體全部可見高覆蓋點、八飛矢及原70:24空拍同次Draw通過。7三相位及23六相位F2、64／65 DAC、HP／RNG與一次續行通過；各21編製反例、飛矢11拒收與八命令兩包載入／玩家及AI缺件原子性零SKIP，無OOM。初版新飛矢核對器把清冊header陣列誤與盤點字典相比，修正schema後同一收據通過，並依實際六相位加scroll-in/out拒收，不列產品缺陷。新68檔按白名單同步、舊4566保持，完整4634／runtime4130與十三原稿、五RGBA裁切逐筆SHA／UID1000核對，私人獨立提交遠端一致。README總覽為兩張實際最終GPU縮圖，面板仍為fixture來源角色名，不冒稱自然玩家。未錄影或發行新版；完整HD與集中正常／人工仍待，#263及#268保持開啟。

2026-10-22 7／23提交前核對：README新增圖片具實際GPU來源及縮圖SHA，178個本地／外部連結中本地目標存在、77張圖片入口已核，文化保存核心保持。新主收據84份當時可讀檔案雜湊一致；修改檔及工作根UID／GID1000，無root-owned或誤建.md目錄。遠端工作清單已重拉並經工具渲染／verify，教訓guard通過。本批FD2容器均已退出、自動刪除；沒有新增錄影。

2026-10-22 7／23推送收尾：公開2016cd374c8067d903db1e2250e2905a2e42c6cb已快進main並推送，以ls-remote重新核對相同；私人獨立提交遠端亦已核。#263已追加公開提交與限定結果，完整HD及#268集中人工保持開啟。此輪為實際新增正式50幀、完成Draw與兩包驗證及遠端同步的progress。最終FD2容器篩選為空，沒有新增錄影或Release。推送後紀錄留待下一批實質提交保存。

2026-10-22 工具列與城鎮／商店要求覆核：960a7693已包含使用者指定功能，未重複實作。重新唯讀參考Zeliard預覽工具列，核對正式十五份Go來源仍與既有設施收據相同；十八份驗證收據／總覽圖雜湊一致，包含114個GPU案例與實際滑鼠／F2／F4輸入收據。最新main另重跑十一項商店、裝備、文字及轉場回歸全部通過，3.512秒、峰值574431232 bytes、無OOM；紀錄位於既有工作根toolbar-request-recheck-oct22-r1。遠端#204與#201仍OPEN，完整HD及正常／人工確認範圍保持。未新增圖稿、錄影或Release，不以本輪覆核取代集中人工驗收。

2026-10-22 回到戰鬥HD：上一輪工具列覆核有新回歸證據但無新增戰鬥圖源，本輪從2016cd37新分支補25／27完整44幀。固定原PNG／mask／描述子與83零header已核，兩張飛行空拍保留；61與主收據先列DRAFT。兩張待機新原稿生成中，其他流程、規則與保存保持。

2026-10-22 25／27達READY：十二新RGBA及44幀R2配方已核。R1斬光形狀與突刺遮擋拒收，R2補三張弧光並調層序。27返場局部取樣原誤配236點，改用原圖下半身取得2387／2435點，僅作美術幾何強推論；完整位移與來源保留。姿勢／尾光共用、較窄突刺光與原生變形差異逐筆披露，集中人工仍待；未登錄，接下來執行GPU與封包檢查。

2026-10-22 25／27正式接線：五銀行44幀候選與runtime-only來源GPU完整畫布對CPU Over通過；兩組物理Game.Draw的16／18個非空圖源包含飛行局部／投射物，維持1000可見核心。82:5／11固定原mask空拍與同次Draw、八空拍反例、各21編製拒收、八命令兩包正式載入／玩家及AI缺件原子預檢零SKIP通過；無OOM。十二新原RGBA、一張精確刀尖裁切與44PNG依白名單同步，舊4634檔保持，完整4691／runtime4174檔逐筆SHA及UID1000核對。第三銀行77僅來源GPU；姿勢、尾光、光寬及變形近似明示，正常與人工依原排程最後集中。README加入兩張實際最終GPU縮圖，保留fixture標示。無正式Go規則或保存改動，未錄影或發行新版。

25／27追加施術驗證：25第三銀行77的原標記符合指令0既有一次MP契約，既有typed測試場景無需改規則即通過82次HD繪製、五階段F2、MP1／HP7與一次續行；全部八幀核心與35張最終PNG、七拒收通過，48.104秒、峰值1370386432 bytes、無OOM。至此44幀均有明示fixture的實際Draw；原版技能可達性、正常與人工仍待。--timeout550首次在測試啟動前遭argparse拒收，修正為既有--timeout 550乾淨重跑；收據反例產物是report.json，依工具實際輸出記錄，不猜guards.json。

2026-10-22 25／27提交前核對：README新增圖片具實際GPU來源及縮圖SHA，180個本地／外部連結中本地目標存在、78張圖片入口已核，文化保存核心保持。新主收據98份當時可讀檔案雜湊一致；修改檔及工作根UID／GID1000，無root-owned或誤建.md目錄。遠端工作清單已重拉並經工具渲染／verify，教訓guard通過。本批FD2容器均已退出、自動刪除；沒有新增錄影。

2026-10-22 25／27推送收尾：公開df155eff1be9807c0d9655c2d0e9f389029bd57d已快進main並推送，直接ls-remote與本機HEAD相同；私人獨立提交遠端亦已核。全部44幀含第三銀行施術及兩空拍具實際fixture Draw證據，原版玩家可達性、正常及人工仍不提升；完整HD與#268保持開啟。本輪屬新增正式圖源及雙庫交付的progress，無錄影或新Release。推送後紀錄留待下一批實質提交保存。

2026-10-22 28／29／31接手：上一輪新增44幀及雙庫推送為progress，本輪從df155eff新分支恢復，保留推送後紀錄，無既有FD2容器。固定原六銀行44幀與86／89／95零header核對，選兩組拳擊與一組短杖打擊，三待機母稿已發出生成。61與主收據先列DRAFT，規則與保存保持，未錄影。

2026-10-22 28／29／31達READY：十四新原RGBA、六採用衍生及六銀行44幀候選已核。28大C垂直翻轉符合原圖斜向，父稿保持；29實心／空心星芒與後段原生130個FX元件圖框已查，細點沿既有密實光點。三未用光片變體不進清冊。29較窄袍身、31大C較短高度與姿勢／尾光共用明示，集中人工仍待；不改原框／命中／等待／規則或保存。接下來實際GPU與兩包檢查，未錄影。

2026-10-22 28／29／31正式接線：六銀行44幀候選與runtime-only來源GPU完整畫布對CPU Over通過，三組Game.Draw的17／14／13張非空來源維持1000可見核心。各三相位F2、64／65 DAC、HP／RNG與一次續行、各21來源拒收及十二命令兩包載入／玩家與AI缺件原子性零SKIP通過，17項Go均無OOM。十四原RGBA、六精確裁切／正交轉置及44PNG沿白名單保存，ThinGoldSweep未翻轉父稿也保持，三未用光片排除；舊4691檔不變，完整4755／runtime4218檔逐筆SHA與UID1000核對。README新增三張實際GPU縮圖，面板仍為fixture角色名；袍身、光弧高度、姿勢及尾光近似明示，正常與人工依原排程最後集中。無Go正式規則或保存修改，未錄影、未發行新版。已保存32／33／34固定原圖參考與三張待機母稿，入口為同工作目錄party32-34-generation-r1.json，僅供下一批，未登錄或同步私人白名單。

2026-10-22 28／29／31提交前核對：README新增圖片具實際GPU來源及縮圖SHA，182個本地／外部連結中本地目標存在、79張圖片入口已核，文化保存核心保持。新主收據113份當時可讀檔案雜湊一致；修改檔及工作根UID／GID1000，無root-owned或誤建.md目錄。遠端工作清單已重拉並經工具渲染／verify，教訓guard通過。本批FD2容器均已退出、自動刪除；沒有新增錄影。

2026-10-22 32／33／34接手：上一輪7e0ab779新增44幀且公開與私人實際遠端已核，屬progress。本輪自main新分支，無既有FD2容器，遠端23條工作議題已重拉。固定八銀行77幀、兩空拍及101零header，三待機原RGBA已保存，再補揮劍／揮斧姿勢。61與主收據先列DRAFT，98／104兩marker不猜普通0，未錄影。

2026-10-22 32／33／34達READY：21張原RGBA中18張採用，三窄長FX原型拒收；七精確RGBA衍生與八銀行77幀已審查。R1白光與劍刃RGB共用，R2修正圖源分層選色順序；R1部分錨點縮放後裁切5～17個高覆蓋像素，R2留兩個來源像素邊界，全部77幀零高覆蓋裁切。63來源拒收通過。初次批次metadata混入舊三待機的result格式，按已保存欄位篩選再複製，無重生或像素改寫；docker外層timeout需主機Docker權限，以同映像有界執行。姿勢／武器／光效差異保留，未錄影或改規則／保存。

2026-10-22 下一批預備：35／36／37原圖參考與三張待機原RGBA已保存並核SHA，入口為既有ordinary-effects-r1/party35-37-generation-r1.json及party-pending-35-37-native-oct22-r1.json。只屬候選預覽，尚未編製、登錄或同步私人白名單，不列本批交付數量。

2026-10-22 32／34 caller勘誤：先前由普通0雙marker拒收便把caller列未知，漏查已有正式路徑。現有native_command24_presentation.go明示selector32／98及固定15幀schedule；native_command29_presentation.go明示selector34／104及derived-strike分段schedule，58亦已有29的RE／indexed E1入口。這足以訂正未知描述，不需重開EXE。兩owner目前均無modern綁定，故33張第三銀行仍只來源GPU，改列HD消費端缺口。先完成本批77來源／44物理交付，下一切片優先接24／29，再續35／36／37；不改原標記或規則。

2026-10-22 32／33／34正式接線：77幀候選與runtime-only來源GPU完整畫布對CPU Over通過；17／12／15張物理來源全部Game.Draw可見核心至少1000，三相位F2、64／65 DAC、HP／RNG與一次返回通過。單marker25正對照與98／104雙marker原子拒收通過，兩第三銀行共33幀及兩原生空拍僅來源GPU，24／29索引色caller已知但HD尚待綁定，不改標記猜補。各21來源反例與十二項兩包載入／玩家及AI原子預檢零SKIP，18項有界Go通過無OOM。18原RGBA、七精確衍生與77PNG依白名單保存，原4755檔不變，完整4857／runtime4295檔逐筆SHA與UID1000核對，三拒收FX排除。README新增三張實際GPU縮圖，人物比例、劍長與FX差異保留。正式Go規則／保存未改，正常與人工依已定排程集中最後，未錄影或發行新版。

2026-10-22 32／33／34提交前核對：README新增圖片具實際GPU來源及縮圖SHA，184個本地／外部連結中本地目標存在、80張圖片入口已核，文化保存核心保持。新主收據138份當時可讀檔案雜湊一致；修改檔及工作根UID／GID1000，無root-owned或誤建.md目錄。遠端工作清單已重拉並經工具渲染／verify，教訓guard通過。本批FD2容器均已退出、自動刪除；沒有新增錄影。

2026-10-22 32／33／34推送收尾：公開3c4ee6d7993d435cf4cb4ba5c4b781c5e2aa2713已快進main並推送，實際ls-remote與本機HEAD相同，私人獨立提交亦已核實際遠端。本輪屬新增77來源及44物理實際Draw的progress，完整HD及#268保持開啟，24／29的HD呈現優先下一切片；35／36／37僅保存三待機候選。無錄影、無新Release、無殘留FD2容器。推送後紀錄留待下一批實質提交保存。

2026-10-22 指令24／29HD接手：上一輪3c4ee6d7新增77來源／44物理及雙庫推送為progress；新分支保留推送後紀錄，無殘留FD2容器。主機遠端worklist已拉。先查既有原證據與typed selector、mode1 prelude、20拍viewport完整metadata，確認不需新RE或渲染架構。DRAFT經來源hash／介面審查達READY，主收據split_strikes24_29_followup_20261022；目標為24及29多目標全相位HD，不改交易、規則、保存或美術清冊。未錄影。

2026-10-22 24／29預檢R1拒收：396完整native GPU案例通過，兩項預檢在末段RNG斷言失敗。追原Go顯示plan本來就消耗每目標亂數，原測試誤判成功建構應零擲骰；亦確認新增HD錯件檢查在plan之後會額外消耗。保留失敗log，規格回DRAFT補證後READY，沿既有stageBattleHDBanks在plan前暫存、原生建構後才發布；改計數來源比對原有成功擲骰，非降低門檻。

2026-10-22 再次核對使用者指定的工具列與城鎮／商店HD要求：功能已由960a7693接入，未重複修改正式繪製。33份相關來源與既有驗證收據SHA保持，當前六種設施背景與故事頭像素材SHA核對通過；十一項金幣、物品、商店裝備、文字轉場及城鎮pulse回歸零SKIP通過。新收據toolbar-request-recheck-oct22-r2位於既有ordinary-effects-r1工作根，耗時3.010秒、容器峰值583110656 bytes，OOM為零。現有README已含工具列／城鎮／商店／裝備頭像四格GPU圖。原生選單圖樣及集中正常玩家／人工造型確認仍維持原限制，未新增素材、錄影或Release；新增收據UID／GID1000:1000，測試容器已退出。

2026-10-22 指令24／29 HD接線完成限定RUNTIME-E1：396張雙側原生來源矩陣、33張第三銀行含兩原生空拍的全部Game.Draw、57／88張最終1920對獨立CPU預乘合成通過。兩包全部來源與畫面SHA一致，五／八相位F2、MP各一次／HP一與兩次、一次續行通過；兩包缺件與晚期回復、十六項收據拒收、2087共用GPU及空拍通過。最初核對器直通RGB減色錯誤造成兩階差，查Go上傳與shader後修正CPU順序，門檻仍為一階。首輪全套Go的遊戲與規則通過但字串審查拒收；精確舊inventory在既有工作根，保留99筆文字／分類並補前批工具列九筆玩家文字後，同容器同全套命令乾淨重跑通過，152.741秒、峰值5971628032 bytes、無OOM。第一次舊git樹缺精確inventory與兩次受控遷移拒收均保留，未猜補ID；暫時工作樹及自主停止的過廣搜尋容器已清除。README新增四格實際GPU縮圖，背景／面板及美術近似維持。正式catalog與私人美術未變，無新私人提交、錄影或Release。正常與人工依原排程集中全部HD後，完整戰鬥HD仍未完成。

2026-10-22 24／29提交前：官方主機Issue重拉並由既有產生器更新，23條OPEN／15人工／0可能完成；300條教訓、168守衛通過。1095份當前Go／工具／收據雜湊與2180個本地連結通過，README文化入口保持；原圖源歷史已加新consumer取代索引。工作根無root-owned或.md目錄，全部FD2一次性容器退出。無錄影，本輪沒有私人素材修改。

2026-10-22 指令24／29推送收尾：公開b4c298d1c08b6234a10321df07bb2bbcc940d3f4已快進main並推送，實際ls-remote與本機HEAD相同；#263回填提交與限定驗證，保持OPEN。完整戰鬥HD仍未完成，下一批續35／36／37；現存候選原稿未編製或登錄。私人素材樹乾淨，本批無私人修改、錄影或新Release，沒有殘留FD2容器。推送後紀錄留待下一批實質提交保存。

2026-10-22 35／36／37接手：上一輪b4c298d1完成24／29完整HD consumer並推送，屬progress。本輪從main新分支，保留推送後紀錄，無既有FD2容器。固定六銀行47幀與107／110／113原零header已核，三張先前待機母稿SHA保持並重新檢視；61與主收據先列DRAFT。接續補斧盾及兩騎兵攻擊姿勢，規則與保存保持，未錄影。

2026-10-22 35／36／37配方R1在輸出前被既有KEY守衛拒收：生成任務CamelCase誤用為配方pose ID。保留R1，R2只把配方來源鍵與引用轉小寫；原RGBA、錨點及正式工具不變，沿同映像重跑。此為候選配方語法錯誤，非遊戲缺陷。

2026-10-22 35／36候選29幀已編製且零高覆蓋裁切；37的R2整支針形槍影在小元件縮樣後alpha231，既有250分層守衛拒收。查原稿槍杆包絡871..903，再以原RGBA精確裁切槍杆／短片及正交轉置依元件長寬比配置；原針稿保留拒收。所有槍影來源取樣均達250，正式compiler仍須再核每層及最終255，未調alpha、門檻或原描述子。

2026-10-22 使用者再次指定參考Zeliard的上方主題工具列、城鎮／商店HD及既有HD頭像接入。核對現行程式與Zeliard正式入口，這些功能已由960a7693接入，保留當前未提交戰鬥美術，不重複改寫正式路徑。使用目前515項清冊重跑TestModernHDFacilityRuntimeGPU，114個最終GPU案例通過，包含三城鎮、三商店、成功時間軸及十二裝備開窗位置，耗時59.05秒；TestPlayerThemeToolbarRealInputGPU以受版控XTest工具實際點擊與F2／F4通過，五次預期主題切換、四語齊全，耗時46.64秒。兩份新收據分別在既有ordinary-effects-r1工作根的toolbar-facility-request-oct22-r3／receipt.json及toolbar-real-input-request-oct22-r3／receipt.json。README既有總覽保持；選單框、游標及交易效果仍為原圖樣，正常玩家與人工造型確認仍依全部HD後集中排程，未發行新版或錄影。

2026-10-22 35／36／37接線收尾：六銀行47幀零高覆蓋裁切；候選與runtime-only各47張完整GPU對獨立CPU Over、三組全部11／18／18張物理Game.Draw核心、三階段F2、64／65返回DAC、HP／RNG與一次續行通過。63項編製反例、六命令兩包載入及玩家／AI原子預檢零SKIP，合計十一Go命令無OOM；沒有正式Go邏輯變更，只新增專用來源GPU測試。整套Go沿上一批b4c298d1收據，未冒稱本批重跑。十二原RGBA與四精確裁切依白名單保存，拒收針形稿排除，舊4857檔保持；完整／runtime-only全部4920／4342檔SHA與UID核對通過。私人獨立提交推送、實際遠端HEAD一致、樹乾淨且isPrivate=true，公開紀錄不寫私人提交識別。README加入實際GPU三格，馬／武器比例、窄光弧及分段槍影近似保留，人工與普通玩家仍集中全部HD後，非原版parity或PLAYER-E2。

2026-10-22 35／36／37提交前：主機遠端Issue已更新並由正式產生器重拉快照，23條OPEN／15人工／0可能完成；300條教訓、168守衛通過。本批104份引用雜湊、三份inline配方及2188個本地連結通過，README文化與多語入口保持。收尾腳本初讀memory_peak_bytes及來源checks時發現欄位不符，依實際memory.after.memory.peak與frames修正後通過，未更動遊戲或驗收門檻。所有一次性FD2測試容器已退出；工作根root-owned與.md目錄檢查為空，無錄影或新Release。正式狀態依主收據shield35_riders36_37_followup_20261022；下一批38／39／40只備妥原版參考，尚未產生或登錄HD。

2026-10-22 35／36／37推送收尾：公開027a93cd4df7599571b2d691d523849333e80760已快進main並推送，實際ls-remote與本機HEAD相同；#263回填提交，保持OPEN。私人獨立提交與遠端已核，樹乾淨且仍private，不在公開紀錄寫識別。無殘留FD2容器、錄影或新Release。下一批38／39／40的固定原版參考與清單位於既有ordinary-effects-r1，尚無新HD母稿或正式登錄；推送後紀錄留待下一批實質提交保存。

2026-10-22 38／39／40接手：上一輪027a93cd完整47幀及雙庫推送為progress。本輪從最新main建分支，保留推送後紀錄，開工無FD2容器。固定六銀行61幀、116／119／122零header與40純飛矢原始點數已核，主收據及61先列DRAFT。十三張內建image_gen原始RGBA母稿已保存並檢視，generation帳冊包含提示與SHA；姿勢與細光近似明示，READY後編製候選，不改正式規則或保存。核對器只增加固定40設定，舊23九張正對照與十一項反例通過，原人物1000點門檻保持。未錄影。

2026-10-22 38／39／40配方準備：微小片段首次誤找全255的5×5區，實際金色尾光沒有該區；單像素255裁切又正確違反透明背景契約。回查規格並量測原RGBA，改用同時含0及255的精確5×5裁切R2，原稿、alpha與工具門檻不變，拒收gold-chip-r1留本機且不登錄。初次圖像回傳紀錄誤輸出完整工具結果，後續改只記任務名稱與路徑，原圖與prompt均已保存；此為工具輸出錯誤，不列產品缺陷。

2026-10-22 38／39／40接線收尾：六銀行61幀零高覆蓋裁切，候選及精簡包各61張完整來源GPU、全部54張物理本體核心及七張純飛矢通過。三／六階段F2、64／65返回DAC、HP／RNG與一次續行、63編製反例、新40及舊23各十一飛矢反例、兩包玩家／AI原子預檢通過，十一Go命令零SKIP、無OOM。40R1完整C縮成微點造成alpha235而正確拒收，R2只將微點改用已核原RGBA金屬亮點精確裁切，來源alpha與250／255門檻保持；失敗樣本保留。正式清冊與受驗預定清冊bytes相同，六資源及76新檔完成白名單登錄，完整／精簡4996／4403檔SHA與UID核對，十三原RGBA及兩裁切完整、拒收不透明亮點排除，舊4920檔保持。README新增實際GPU四格；共用姿勢、馬／武器比例、握手及曲光近似保留。無正式Go邏輯變更，整套Go沿b4c298d1歷史收據，不冒稱本批重跑。普通／人工集中全部HD後，不提升原版parity或PLAYER-E2。下一批41／42／43只準備原版參考，尚無HD生成或登錄；本批沒有錄影或新Release。

2026-10-22 38／39／40提交前：收尾腳本將sample_count誤當核心數，依核對器程式確認其為22張輸入，matches才是15張本體；只修收尾斷言，產品及核對門檻保持。全部102份引用雜湊、三份inline配方、兩個精確裁切與2196個本地連結通過，README文化／多語／凱麗GIF入口保持。私人素材獨立提交推送，實際遠端HEAD相同、樹乾淨且isPrivate=true，公開紀錄不寫私人提交識別。主機Issue263已更新，正式產生器重拉23條OPEN／15人工／0可能完成；300條教訓、168守衛通過。FD2一次性容器已退出，本批工作根、正式及私人素材root-owned與.md目錄檢查為空。

2026-10-22 38／39／40推送收尾：公開636773e9c6224127a5cbb8db2e52a259126123e7已快進main並推送，實際遠端與本機相同；#263回填提交並保持OPEN。私人保存庫獨立推送與遠端已核，仍private，不公開其提交識別。沒有殘留FD2容器、錄影或新Release。

2026-10-22 41／42／43接手：從最新main建分支，前輪61幀是progress。固定六銀行49幀、三個單次命中及125／128／131零header實檔已核，三份原生參考已檢視；主收據與61先列DRAFT。三張待機原RGBA採內建image_gen生成中，未產生候選或正式登錄，不改規則與保存；正常與人工依全部HD後集中排程。

2026-10-22 41／42／43母稿首輪：三張原RGBA均為alpha0..254，沒有255核心，未升READY或編製。原圖、提示及SHA保留在本機；內建圖像編輯只要求移除外圍光暈並使人物實心，保持造型與站姿，不用程式改alpha或放寬最終255契約。

2026-10-22 41／42／43透明度第二輪：三張編輯稿仍為alpha0..254，均保留拒收。連續兩輪同類失敗後回查規格閘門，改以原版構圖加既有figure029的0..255乾淨圖源作風格參考，先單張重繪42確認輸出；不調整alpha或驗收門檻。讀取尚在執行cell的store未取得資料，改依已回傳實檔路徑核對，此為工具資料可見性，不列產品問題。

2026-10-22 41／42／43透明度勘誤：既有layered-alpha-final-composite-gate教訓已明載來源max254不推定成品254，前兩輪判斷過嚴而偏離既有契約。依原LANCZOS公式量測七張待機稿，全部得到0..255且具1934～2593個255點，無alpha或門檻變更。採R2三份維持服裝一致，R1及42R3是未採用版本，不宣稱編製器曾拒收它們；保留前段錯誤形成記錄。十三選定原稿／三原生參考已檢視，完整49錨點及配方升READY，接著編製候選。

2026-10-22 41／42／43接線收尾：六銀行49幀零高覆蓋裁切，兩包各49張完整來源GPU、三組全部20／15／14張物理Game.Draw核心、三階段F2、64／65返回DAC、HP／RNG與一次續行通過。63編製守衛、舊38分層21回歸與兩包玩家／AI原子預檢通過，十一Go命令零SKIP、無OOM。原錯誤缺旗標測項因單層dict本就合法而成功，屬測試假設錯誤；改為實際加入list再移除旗標，缺最終核心亦支援單層，未改compiler或正式Go。前段透明度誤判已有追加勘誤，保留原RGBA及250／255契約。正式清冊與受驗預定清冊bytes相同，六資源及62新檔完成白名單登錄，完整／精簡5058／4452檔SHA與UID核對、舊4996檔保持。十三原稿採用，四替代稿排除；42／43兩原稿邊緣接觸明示，README新增三格實際GPU，正常／人工集中全部HD後。整套Go仍沿b4c298d1歷史證據，不冒稱本批重跑；沒有錄影或新Release。

2026-10-22 41／42／43私人推送：獨立提交與實際遠端HEAD一致，樹乾淨且isPrivate=true；公開紀錄不寫私人提交識別。本批98份引用雜湊、三份inline配方、十三母稿與2204個本地連結通過，README文化、多語與凱麗GIF入口保持。工作根、正式與私人素材root-owned及.md目錄檢查為空。下一批44／45／46只準備固定原版參考，清單位於work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/party-pending-44-46-native-oct22-r1.json，尚無HD生成或登錄。

2026-10-22 41／42／43提交前：正式產生器重拉23條OPEN／15人工／0可能完成。新教訓首次綁test_檔而被既有掃描器排除，改綁實際compiler的list／旗標分支，不修改掃描規則；301條教訓、169守衛通過。此為守衛資料修正，實測仍依本批63及舊38的21案例。Docker檢查無FD2容器，其他專案容器保持。

2026-10-22 41／42／43推送收尾：公開e36efeb0da5ebe3f848cc353c4343e1b4bdc9d97已快進main並推送，實際遠端一致；#263回填提交且保持OPEN。私人保存庫獨立推送與遠端已核，仍private，不公開其提交識別；無FD2容器、錄影或新Release。

2026-10-22 44／45／46接手：前輪49幀屬progress，本輪從main建分支。固定六銀行56幀及三零header已核，45另有兩空拍與三純飛彈；主收據及61先列DRAFT，飛彈驗證子契約READY。三待機原RGBA保存並檢視，alpha255／255／254均沿既有來源>=250契約，不再從母稿max254推定成品不合格。十二動作／特效母稿採內建image_gen生成中，尚未編製或登錄，不改規則與保存，正常／人工集中全部HD後。

2026-10-22 44／45／46續接：前輪僅覆核工具列，無戰鬥進度；本輪保存已完成兩張R2拳師修稿並檢視，未重啟生成。十六母稿、三精確RGBA裁切、兩既有FX及56錨點審查後升READY；136兩空拍與三純飛彈分開驗，23／40各十一項飛矢守衛先通過。準備腳本首次多一括號，在寫檔前即拒收，修正後同映像重跑通過，屬工具語法問題。未登錄、未錄影，正常／人工保持全部HD後。

2026-10-22 44／45／46接線收尾：六銀行56幀零高覆蓋裁切，兩包各56張完整來源GPU，全部15／21／15張物理本體核心及三純飛彈／兩固定空拍、三／六／三階段F2、64／65返回DAC、HP／RNG與一次續行通過。63編製守衛、舊38分層21回歸、新45飛彈十二反例及舊23／40各十一反例、兩包玩家／AI原子預檢通過，十一Go命令零SKIP、無OOM。缺最終核心反例選到微小金光，軟層250正常重採樣到255而通過；改完整待機構造反例，沒有改compiler或門檻。首次守衛啟動命令引號未閉，在執行Python前失敗；改唯讀腳本同映像重跑。中斷後四命令程序已自然完成且容器退出，沿原handle取回，未重啟。正式清冊與受驗預定清冊bytes相同，六資源及75新檔白名單登錄；完整／精簡5133／4508檔SHA與UID核對、舊5058保持。十六原稿及三裁切採用，兩拳師R1未採用；全部原稿無高覆蓋邊緣接觸。README新增三格實際GPU，比例／握手／砲管灰殘影及共用姿勢留全部HD後集中人工。整套Go仍沿b4c298d1歷史證據，不冒稱本批重跑；沒有錄影或新Release。

2026-10-22 44／45／46私人推送：獨立提交與實際遠端HEAD一致、工作樹乾淨且isPrivate=true，不公開私人提交識別。登錄串接腳本首次誤將sample_count當本體筆數；現有收據26是全取樣，matches21才是本體，寫入前拒收，改核全部26及本體21後重跑通過，未動核對器或門檻。py_compile曾向唯讀掛載寫快取失敗，改PYTHONPYCACHEPREFIX=/tmp後同命令通過，屬環境問題。下一批47／49／50固定原圖及50第三銀行已保存並檢視，48三銀行零header逐實檔核對，不補造；新HD尚未生成。未錄影，本批測試容器均已退出。

44／45／46提交前收尾：137份引用雜湊、三份inline配方、十六原RGBA及三精確裁切、2212個本地連結通過；README文化、多語與凱麗GIF入口保持。官方產生器重拉23條OPEN／15人工／0可能完成，301條教訓及169守衛通過。工作根、正式與私人素材無root-owned及誤建.md目錄；相關一次性容器已退出。最末一般sandbox無Docker socket權限，改由授權的同一隔離映像完成檢查，沒有退回主機執行工作負載。

2026-10-22 凱麗右腳接手：從b6078148主線新分支。逐幀原圖與正式HD對照確認前段本應右靴踩地，母稿卻畫成懸足且露鞋底，後腿額外露出，旋身亦露出原版遮住的雙靴。20幀與六母稿、原GIF收據已定位，不重開EXE。DRAFT先登記，內建image_gen局部修圖，候選／普通GUI與GIF更新前不宣稱修復完成。

凱麗GUI R2流程、返回、F2及保存冷讀PASS，但人工檢視發現舊驗證器強制1920×1200，與既有工具列1272高契約衝突，底部72像素被裁。此為驗證腳本幾何問題，未宣稱遊戲缺陷或擷取完成；回查工具列規格與畫布路由後，先列明示工具列擷取／raw來源及精確game crop的READY子契約，再修改工具並乾淨重跑R3。

2026-10-22 凱麗GUI r3正常流程PASS，350.811秒、記憶體峰1906167808、無OOM；18份原視窗與裁切全數逐像素相等。GIF合成因不同影格不足被拒收，尚未覆寫README；原因是取樣迴圈逐張identify／convert耗時。依61新增READY取樣補充，將精確裁切後移，維持相同來源、seed與32張上限重跑r4。初次r3啟動因promo映像無Pillow被拒，改用既有ImageMagick後通過，非產品缺陷。

2026-10-22 凱麗r4正常GUI PASS，719.639秒、峰2526371840、無OOM；GIF首次合成PASS且全檔解碼及一像素錯裁反例通過。目視總覽發現audit先於PNG約一拍，因此不以audit索引配對旋身；補核實際來源。現有普通GPU核對器納入1272高physical-scene而按舊1200規格拒收，屬驗證工具幾何契約；依61補明示工具列裁切模式，保持匹配門檻再重跑。

2026-10-22 凱麗右腳v2正式接入：六新母稿與20幀，21項編製拒收、五組Go、20幀Game.Draw核心及F2／DAC／HP／RNG／一次續行均通過，無SKIP或OOM。完整視窗第九章同槽seed1普通攻擊、返回、保存與冷讀通過，全部原PNG與精確遊戲區裁切像素一致；README GIF及靜態圖已換新普通v2。原版側重用既有dosgolem固定收據，姿勢映射與剪輯節奏逐筆揭露，不宣稱逐幀一致。私人現行包26換26、5107檔保持，舊版留來源及Git歷史。#269保持集中人工確認，其他HD仍待。未錄影、未發行。

凱麗右腳收尾核166份SHA、2325個本地連結及17檔UID／GID；工作清單及教訓守衛通過，私人獨立提交與實際遠端及private狀態已核。工作根無root-owned或誤建md目錄，無非預期FD2容器。#269與#263仍開啟。

凱麗修正最後核對166份SHA、2325個本地連結、17檔擁有權通過；24項工作清單含16項要人判、可能已完成0，302教訓／170守衛通過。刪除本輪臨時字型複本19484784bytes，原字型保留固定Docker映像，原PNG及README GIF保留；無新錄影及殘留FD2容器。

2026-10-22 接續59818d60，開始47／49／50完整55幀。原版141／142、147／148、150／151／152已檢視，48三銀行全零不補。49原版有飛離畫面局部裁切、固定空拍與兩命中，50攻擊四命中及真實第三銀行，不為套既有測試改標記。已建立DRAFT，先做可撤回母稿及consumer核查。

2026-10-22 本批已保存16份選用原RGBA及完整提示詞；47金踢R1多餘足拒收，R2修正為兩腿。49金弧R2增加圓形碎光且仍觸邊，選較接近原圖的R1並揭露18個高覆蓋光點觸邊，不宣稱原稿完整無裁切。50金光及收勢原稿alpha最高254，臨時保存器誤要求255，改回既有compiler原稿>=250契約，保留原RGBA，正式PNG與GPU門檻不變。母稿仍未進正式包。原版49平移參考完成，六組局部可精確回組、其餘保留姿勢差異；既有物理計畫已支援分段多命中，50第三銀行可沿既有可選人物指令0驗證。

2026-10-22 47／49／50配方READY：完整55幀與原標記保持，49局部採原版平移窗口，50人物／兩色扇光獨立定位，單色純光另生成三張含49橢圓掃光。三份前置複合原稿保留作生成譜系，兩拒收版本排除；49刀尾／布尾、共用手勢及光弧邊緣近似逐筆記錄。正式清冊仍533項。

2026-10-22 三組R1候選完整55幀已編製，540項候選、75新檔與精簡4563檔通過靜態清冊；正式533項保持。目視發現50光效根部與劍尖不連接，原因是按亮區包絡而非手繪實際劍尖定位。依61先定R2契約，再以兩個來源點修正七幀光效錨點，原始資料及RGBA保持，尚未進GPU驗收。

2026-10-22 47／49／50接線：完整55幀兩包來源GPU、14／17／19張物理核心、翼人固定空拍及50第三銀行四幀施術通過。63編製、七空拍、七施術守衛、十二Go命令零SKIP／無OOM，F2／64及65 DAC／HP與RNG／一次續行保持。50七幀光根改接實際劍尖，最大4.153HD像素；新可重跑連接工具拒收真實舊R1的201.180像素間隙。原稿、原框及標記保持，局部裁切及姿勢近似留集中人工。施術守衛輸出本名為report.json，恢復腳本誤查guards.json；已依成功日誌及實檔核正，未重啟驗證。75新檔登錄，原533項與5133檔保持；完整／精簡5208／4563檔全部SHA與UID核對。兩拒收草稿排除，README新增三格實際GPU縮圖，凱麗GIF及文化／多語入口保持。正常與人工按全部HD後集中，未錄影、未發行，完整Go沿歷史證據。

47／49／50最末雜湊檢查拒收動作清單舊引用；舊SHA精確對應現行清單前15筆，後續補入兩色獨立扇光及翼人獨立環光時漏回填清單SHA。全部18筆原RGBA、提示詞及採用／拒收／譜系資料與主收據逐欄相同，故只修正清單引用；原圖、配方、成品與GPU收據保持，不將此歸為遊戲問題。

47／49／50提交前核144份引用SHA、三份inline配方、2327個本地連結及14檔UID／GID通過。官方工具重建遠端Issue快照、工作清單及教訓守衛通過；新增光效實際連接點教訓與可搜尋工具守衛。私人獨立提交／實際遠端及private狀態已核，不公開私人識別。工作根無root-owned及誤建.md目錄，所有本批FD2一次性容器已退出，無新錄影。#263、#268與#269保持OPEN。

47／49／50整庫find衛生掃描exit0，另見既有root-owned研究／extracted檔案；本批14變更檔、工作根、正式與私人素材擁有權已分別通過。後續Python逐檔統計在120秒外層逾時停止，未寫入輸出；不重跑全庫統計，未改寫無關歷史檔或整庫chown。範圍見party47-49-50-hygiene-oct22-r1.json，前段「工作根無root-owned」只指本批工作目錄。一次性容器均已退出。

2026-10-22 接續022bb5b2，從main新分支開始51／52／53完整42幀。固定六銀行原PNG／Mask／描述子與三零header已核，原版參考及DRAFT登記於主收據／61；先檢視原姿勢與consumer，不改規則或保存，正式540項保持。

2026-10-22 51／52／53十三母稿與三純光裁切審查後升READY，42幀保持原框／節拍／標記。52另補披風收束，避免把寬披風縮進窄框。51交叉光503高覆蓋點觸頂緣，對應原光帶離開視窗，明示局部來源；其餘十二稿無高覆蓋觸邊。初裁兩片混入斧柄，R2首次誤取幾乎透明區；回查幾何路由並加原尺寸座標格後修正，舊裁切排除。第一次helper誤要求FX最高255，已依既有來源>=250保留253原alpha，不改compiler或正式PNG門檻。正式清冊540保持，尚未GPU或私人同步。

51／52／53 R1候選42幀編製及精簡包通過，零高覆蓋裁切；目視拒收殘光矩形切邊及持杖前段本體縮小，尚未跑Go或正式登錄。依61先列R2子契約，新增兩份尖端光痕及一份半展臂母稿，原像素資料與正式Go保持，不將編製PASS當美術完成。

51／52／53 R2十六原RGBA與完整42幀配方READY；新增兩向尖端光稿取代全部裁切光片，補半展臂避免前段縮小，收束稿保持。兩光稿max254沿既有來源>=250，原alpha不變；53共用51同SHA光稿，投影折回單名。R1候選／裁切留歷史拒收，尚未正式登錄。

51／52／53等待最後兩份有界素材載入驗證期間，已準備54／55／56固定原版參考，尚無新HD生成或登錄；同manifest及原PNG／Mask雜湊核對，主收據next_native_reference掛入入口，避免下一輪重做素材搜尋。

2026-10-22 51／52／53接線收尾：R2完整42幀零compiler高覆蓋裁切，兩包各42完整來源GPU、9／22／11實際物理本體核心、三階段F2、64／65 DAC、HP／RNG與一次續行通過。63編製守衛、兩包玩家／AI原子預檢及十一Go命令零SKIP／無OOM。R1殘光切邊與前段縮小在GPU前拒收，新增兩向尖端光及半展臂，十六原RGBA採用、全部試作裁切排除；53同SHA光稿折回單名。58新檔登錄、原540項及5208檔保持，完整／精簡5266／4605檔全部SHA及UID核對。51主光503高覆蓋點觸原稿頂緣明示，原生主光亦離開視窗；其餘十五稿無高覆蓋觸邊。README新增三格實際GPU，正常／人工集中全部HD後。完整Go沿b4c298d1歷史證據，無錄影或新Release。

51／52／53提交前核122份引用SHA、三份inline配方、十六原RGBA、2329個本地連結及12變更檔UID／GID通過。官方產生器更新遠端Issue快照，工作清單及既有教訓守衛通過；來源定位、alpha與編製總覽沿既有教訓，未重複新增。私人獨立提交、實際遠端與private狀態已核，不公開私人提交識別。本批工作根find無root-owned或誤建.md目錄，全庫無關歷史root-owned檔保持；FD2一次性驗證容器均已退出。#263、#268及#269仍開啟，無錄影或新Release。

2026-10-22 從9256e8af的main新分支接續54／55／56飛龍騎乘59幀，原版參考沿上批已固定輸出逐項重核PNG／Mask／metadata及零header；DRAFT登記於61與主收據。正式546項保持，先補母稿及局部升降取樣，尚未生成或登錄。

54／55／56十五原RGBA及59幀配方升READY。三份翅尖觸邊草稿拒收後補全，十五採用稿均無高覆蓋觸邊。原尺寸包絡及source anchor區分升空／返回的局部窗口，55揮擊共用與全部比例近似明示；正式清冊仍546項，尚未編製／GPU或私人同步。

54／55／56初輪兩包59完整來源GPU及63編製守衛通過；四Go容器併行時，54兩包資產組均在player預檢階段超過550秒，無OOM，未列通過。保留r1原log／memory，不覆寫；依驗證路由先確認資產組反覆載入完整主題，改單一容器逐包逐人物，沿專案標準Go1200秒／probe1260秒有界重跑，輸入、影像、斷言與RGB門檻保持。新資產收據r2，source／physical仍r1。

54／55／56將資產驗證改為批次，兩份完整主題分開成功預載與冷集合缺件，每位人物保留獨立player／mode11、HP／RNG及全六銀行無部分發布斷言。舊單人物測試保持，明示停止仍在重複載入的r2工作，日誌與docker停止事件留存；後續採兩包新r3批次收據，加兩來源與三物理共七組。下一批原圖匯出遇57三銀行均零header導致零高接觸表PIL報錯，屬工具輸出問題，未新增HD或有效收據；改跳過全空接觸表，保留零header狀態。

下一批參考R2乾淨重跑通過：57三銀行零header只留原狀態，不補圖或建立零高接觸表；58／59／60原圖／Mask／描述子沿同固定包與盤點核對，主收據next_native_reference掛入口，未建立新HD候選或變更正式清冊。

2026-10-22 54／55／56完整59幀接線收尾：兩包各59完整來源GPU與全部17／22／20物理本體核心、三階段F2、64／65返回DAC、HP／RNG及一次續行通過；63編製守衛與七組Go驗證零SKIP／無OOM。十五原稿與59PNG沿白名單登錄，三觸邊拒收稿排除、來源別名按SHA折回單名，舊546項／5266檔保持。完整／精簡5340／4664檔SHA及UID核對，升降局部compiler裁切與共用姿勢近似明示。README新增三格實際Game.Draw，正常／人工／平台仍待集中，完整Go沿b4c298d1歷史證據，無錄影或新Release。

提交前教訓守衛初輪失敗：工具明示排除_test檔，新守衛誤綁測試來源。改綁61中未預載集合的既有契約，工具與遊戲／測試程式保持；保留初輪log，整體核對用r2新輸出重跑。

54／55／56提交前核155份引用SHA、三份inline配方、十五原RGBA、2331個本地連結及14變更檔UID／GID通過。官方產生器更新遠端Issue快照，工作清單及既有教訓守衛通過；新增批次冷集合隔離教訓與可搜尋守衛；來源定位、alpha與編製總覽沿既有原則。私人獨立提交、實際遠端與private狀態已核，不公開私人提交識別。本批工作根find無root-owned或誤建.md目錄，全庫無關歷史root-owned檔保持；FD2一次性驗證容器均已退出。#263、#268及#269仍開啟，無錄影或新Release。

2026-10-22 從e563bcce主線新分支接續58／59／60。先確認完整57幀與40／41／42的header／描述子／PNG／Mask／解碼RGBA相同，改採舊HD原稿及配方原樣重編、核成品SHA後共用檔名，避免無謂重畫。初步比對誤讀resource.json狀態殼而非animation.json，前兩次輸出不採用；正式證據改由固定inventory及load_native重核，不據錯誤診斷宣稱通過。DRAFT先記於61與主收據；正式552項尚未改動。

等待本批有界攻防／預檢時準備61～64固定原圖／Mask／描述子參考，共70幀；八銀行與43～46逐原生SHA及解碼RGBA相同，後續可沿同套共用方法，但尚無HD登錄或執行聲明。新參考掛入目前主收據next_native_reference，不重開RE或製造空header。

2026-10-22 58／59／60接線收尾：六銀行57重編成品逐SHA等於既有40／41／42，原552項保持，共用5340白名單實檔，不增加PNG／母稿或容量。兩包各57完整GPU、50本體核心與七飛矢、六／三／三階段F2、64／65 DAC、HP／RNG及一次續行、63編製與22飛矢反例、七Go組通過且零SKIP／無OOM。候選腳本曾誤用不存在的master_file欄位，修正後從已核重編成品接續，屬工具環境錯誤，正式清冊未在失敗點改動。README三格實際GPU，原銀行metadata保持獨立、身份與caller不推定；既有近似沿用，正常／人工／平台待集中，無錄影或新Release。

舊40飛矢回歸初輪誤配正式預定清冊，與歷史GPU收據原受驗候選清冊SHA不合而被正確拒收。由原收據雜湊定位party40-candidate-oct22-r1/catalog.json後，用同原圖／同PNG及原門檻在r2乾淨重跑，正對照與十一反例通過；失敗r1不覆寫，非遊戲缺陷或門檻放寬。

58／59／60提交前核87份引用SHA、三份inline配方、2333個本地連結及14變更檔UID／GID通過。所有固定零header重新核實檔，工作清單／教訓守衛通過，新增相同原生銀行共用與分開metadata的重用規則。私人獨立提交與實際遠端／private狀態已核，不公開私人提交識別。本批工作根find無root-owned或誤建.md目錄，FD2一次性容器已清理；#263、#268、#269保持開啟，無錄影或新Release。

2026-10-22 從2c622a4a主線新分支接續61～64八銀行70幀。完整原生輸入與43～46逐SHA／RGBA相同，原配方只改figure與resource鍵；四零header已重核，原558項及5340實檔保持。舊45回歸先用原GPU收據SHA定位實際候選清冊，避免混入後期清冊。DRAFT記61與主收據，先核相同成品再接GPU，不重畫相同來源或推定身份／caller。

等待61～64有界GPU驗證時準備65～73固定原圖／Mask／描述子參考；全空銀行只記原狀態，不補圖。下一批參考掛目前主收據next_native_reference，尚不建立HD登錄或新caller聲明。

61～64執行期驗證等待期間，先固定65／68／69／70共51幀的原圖與單命中／音效標記，建立下一批美術DRAFT。66／67全空只保留狀態；藍衣覆面及三種武裝依視覺描述、不猜故事身份。現階段僅原稿候選，不動正式清冊或父批驗證輸入。

下一批十四候選原RGBA與兩份完整生成ledger已保存，站姿、蓄力、突刺、下劈及三光效已檢視，來源高覆蓋邊界均0。兩份max254原alpha保持；幾何工具只存顏色啟發法，不把混入的橘光／膚色直接當身體或FX語意。完整配方與接點待審，仍DRAFT、尚無51幀HD編製／GPU／正式登錄。父批驗證輸入保持，無額外錄影。

2026-10-22 61～64接線收尾：八銀行70重編成品逐SHA等於既有43～46，零高覆蓋裁切、原558項與5340實檔保持，不新增圖檔。兩包各70完整GPU、65本體核心與三飛彈／兩固定空拍、四完整攻防與三／三／六／三階段F2、64／65 DAC、HP／RNG與一次續行、84編製及24投射物反例、兩包八個獨立交易與八Go組通過，零SKIP／無OOM。舊45正對照先核原受驗catalog SHA，正常一次通過；README新增四格實際GPU，各銀行metadata獨立、既有美術近似沿用，正常／人工／平台待全部HD後集中。完整Go沿歷史證據，不新增原版parity或Release。

58～64候選清理勘誤：初輪嚴禁symlink的保護在任何刪除前停止，未改動檔案；檢視發現編製器已對舊素材建連結，不能將約2GB目標大小重算為各包占用。修訂先逐SHA核私人5340白名單與正式來源，再將167個實體重複PNG原子改為相對連結，保留既有連結、所有候選目錄／catalog／名稱與收據可重跑；實檔邏輯大小114854570bytes。原稿、下一批十四DRAFT與正式／私人／精簡來源保持，逐檔證據記duplicate_candidate_cleanup。只使用有界摘要，原診斷列印過多已結束。

61～64提交前核142份引用SHA、四inline配方、2335個本地連結及14變更檔UID／GID通過。四零header重核，工作清單與教訓守衛通過，沿既有相同銀行共用規則，新增候選實體檔／連結容量與安全整理教訓。私人清冊獨立提交、遠端及private狀態已核；本批工作根find無root-owned或誤建.md目錄，FD2有界驗證容器已清理。下一批51幀仍美術DRAFT、無正式來源登錄；#263／#268／#269保持，無錄影或新Release。

2026-10-22 65／68／69／70接線：原版八銀行51幀沿共背景物理consumer，生成十九原RGBA、採十四且排除五拒收稿。65縮窄站姿與改實心掌光；70背景擷取未去除光暈，RGBA實測確認後重新依原圖繪第四稿；原稿不改alpha。R1合成揭露68劍尖光幾乎離框，改可見劍刃742／250並保留R1收據，其餘三組成品不改。兩包來源GPU、全部51本體核心、三階段F2／64及65 DAC／HP／RNG／一次續行、84編製及16連接／四錯置反例、兩包八交易及八Go組通過，零SKIP／無OOM。新65白名單檔與舊5340SHA、正式574項及兩包bytes／UID核對；新增README實際繪製四格，未宣稱完整HD或正常玩家。剩餘inventory96銀行746幀均未命中既有精確來源共用候選，尚需原檔複核及後續美術。

65～70規格收尾把先前DRAFT／READY等待敘述改為目前已驗範圍，失敗成因留本工作歷程與主收據superseded_r1。下一批71／72／73固定50幀、72的217原生header1及14空拍已核，三待機原稿為DRAFT，無正式接線。另將本批五候選包及兩本機精簡測試目錄的9316張同SHA重複PNG改成指向正式白名單的相對連結，移除重複檔內容約2.13GB；原稿、目錄、收據、私人完整包與SHA保持，未按連結的邏輯大小重算容量。

65／68／69／70提交前核179份來源引用SHA、四inline配方與四連接表、2337個本地連結及12變更檔UID／GID通過。9316連結逐SHA重核，五拒收稿與未登錄三草稿不入私人白名單；工作清單與教訓守衛通過。私人素材獨立提交、實際遠端及private狀態已核，無非預期FD2容器。本批仍限RUNTIME-E1，#263／#268／#269保持，無錄影或新Release。

提交前補留四份編製當時清冊：從固定base及compiled來源鍵重建alias前bytes，逐SHA等於原compiled.json的catalog_sha256。後續574項預定清冊與其分開保存，沒有改寫compiled、PNG、配方或GPU受驗輸入。

2026-10-22 71／72／73完整50幀：固定217的離場12、兩光束4331／706及14全透明Mask，沿現有consumer不加規則；原RGBA十五張含73持盾收勢，max254原樣。R1光束棕暗尖部對到砲口但白亮核留間隔，改白核1700／425並保留R1，其他34成品不改。兩包50GPU、47本體核心、兩光束與一空拍、三完整攻防／3-6-3F2／64及65 DAC／HP／RNG／一次續行、63編製、18接點與三反例、72及舊63各十二投射物守衛、兩包六交易與七Go組通過，零SKIP／無OOM。正式580與受驗stage bytes相同，65新白名單檔、舊574項／5405SHA保持，完整5470／精簡4765；README三實際Game.Draw及近似界線已更新。正常／集中人工／平台仍待，不新增Release或錄影。

本批9430張同SHA臨時PNG改成相對連結，移除重複檔內容2175733487bytes；原稿、目錄、收據、正式及私人完整包保持。以lstat只計原先單連結實體檔，沒有把符號連結的目標大小重算容量。

71／72／73提交前核169份引用SHA、三inline配方與三連接表、三原編製清冊SHA、2339個本地連結及14變更檔UID／GID通過。9430連結逐SHA重核；工作清單與教訓守衛通過。私人素材獨立提交、實際遠端與private狀態已核，無非預期FD2容器。本批仍限RUNTIME-E1，#263／#268／#269保持，無錄影或新Release。

2026-10-22 本輪範圍勘誤：初選76～79未先與正式580項清冊交集，編製器拒絕已登錄76；再查76／78四銀行21幀早已存在，正式清冊沒有變更。收斂77／79四銀行22幀，保留十份多餘生成稿及原DRAFT配方／拒收log，排除正式白名單。後續選批以最近formal_coverage.remaining_decoded_resources與即時catalog雙重交集，禁止只按figure順序推算。

2026-10-22 77／79完整22幀：修正重複選批後，只新增231／232／237／238四銀行，八採用原RGBA原樣；76／78舊圖不覆寫，十多餘稿排除。兩包22來源GPU、全部22物理Game.Draw核心、兩完整攻防／三階段F2／64及65 DAC／HP／RNG／一次續行、42編製、五接點最大2.153HD像素與兩錯位反例、兩包四冷交易及既有71預設19核心通過，七Go組零SKIP／無OOM。測試交換既有敵我指標但Camp／side不改，正式規則／保存／原框保持。正式584與stage bytes相同，30新檔、舊580項／5470SHA保持，完整5500／精簡4787；README兩張實際Game.Draw及近似界線已更新。普通／集中人工／平台仍待，無新Release或錄影。

2026-10-22 本批收尾：左側主攻前景的FIGANI填色33由既有GPU確認只改RGB，驗證器補沿相同SHA的HD alpha／原生Mask，未知32／34與特效33仍拒收。初輪77核心拒收log保留，修正後77／79各11與舊71共19核心乾淨重驗通過；另八項來源遮罩守衛通過，不改遊戲。最終134個參考SHA、204個README本地連結、所有既有README行、原編製清冊與inline配方、UID及工作訊號通過。私人素材已獨立提交推送並實查遠端及PRIVATE，工作樹乾淨；不公布私人提交識別碼。只將4719個同SHA候選重複PNG原子改相對連結，減少1091082568實檔bytes，保留所有來源／收據／目錄；沒有新增錄影。FD2素材／Go容器皆已清理，未發現root-owned產物或誤建.md目錄。公開提交與main遠端核對接續執行。

2026-10-22 上批公開提交463e4ed71341a6770fa36de21603b02663aa676f已快轉main並推送，實際ls-remote相符；#263留言6343565592記錄驗收及限制。本輪從該main建立hd/actors80-82-20261022，先核正式remaining與即時清冊交集，240／241、243／244、246／247六銀行確實未登錄，242／245／248零header。既有76／78不重開。

2026-10-22 80／81／82完整31幀：先核剩餘及即時清冊交集，六銀行確實缺少。80／81分別保留音效4／5與末延遲6／2；82保留錐槍姿勢。十二新登錄原RGBA含兩舊草稿新採用，一既有殘光按SHA共用；82待機／81舉劍觸邊與82過寬突刺三稿拒收，82槍尖光根改實際白核600／600，RGBA不改。兩包31來源GPU、全部31Game.Draw核心、三完整攻防／F2／64及65 DAC／HP／RNG／一次續行、63編製、十接點最大3.293HD像素及三錯位反例、八閃色遮罩守衛、兩包六冷交易與七Go組通過，零SKIP／無OOM。正式590與stage bytes相同，43新檔、舊584項／5500SHA保持，完整5543／精簡4818。README三實際繪製與原框裁切／窄光柱／武器比例近似已更新；正常／集中人工／平台仍待，沒有新Release或錄影。

2026-10-23 本批收尾：初次最終核對誤沿用580切片，改從固定基底讀實際長度584後通過，正式素材未改。新增教訓的守衛初輪未逸出正規表示式括號，誤報訊號不見；修正守衛後同映像重跑通過，失敗log保留，未改遊戲程式。私人素材已獨立提交推送並核對實際遠端與PRIVATE。4766個同SHA候選重複PNG原子改為相對連結，移除1139347890重複實檔bytes；原稿、收據與目錄保留。沒有新增錄影，FD2素材及Go容器已清理。公開提交與main遠端核對接續執行。

提交前GitHub帶標籤查詢意外回空，與直接讀#263的OPEN／worklist矛盾；改讀主機GitHub全部開啟議題並按實際worklist標籤篩選，24項及#263／#268／#269均在。修正尚未提交的空快照，再重跑原最終檢查；未關閉任何議題。

2026-10-23 接續eb2f3008主線，84／85／86六銀行43幀與正式清冊及最新remaining交集核對通過，無原生相同共用銀行。原版唯讀素材與全幀參考已固定SHA，三零header保持；先備紅甲巨斧、藍甲長戟及飛龍騎乘人物DRAFT，不推定故事身份或caller。

84～86編製R1總覽發現86的全翼幅來源錨點使編製器按寬度縮小人物，後九拍的翼幅起點也誤排；以實際身體錨點及原生展／半／收／半改R2，只重編86。R1配方、PNG及投影收據保留，不進GPU或正式清冊，84／85保持。

等待84～86有界GPU時備88／89下一批原版參考及兩份待機原RGBA草稿，掛目前主收據next_art_reference。原13／15頭段之外含投射物與空拍，實際解碼共43幀，不以header byte2推算完整幀數。下一批仍DRAFT、未編製或正式登錄，不改目前候選／Go輸入，後續編製前重新核正式基底及缺口。

2026-10-23 84／85／86完整43幀：十二新原RGBA與四既有光效按SHA保存，第一展翼稿近邊排除。飛龍R2修正全翼幅錨點造成縮小及後九拍起點，僅重編86；原框裁切量與盔飾／武器／光痕近似明示。兩包43來源GPU、全部43Game.Draw核心、三完整攻防／F2／64及65 DAC／HP／RNG／一次續行、63編製、八接點最大2.843HD像素及兩錯位反例、八閃色遮罩守衛與兩包六冷交易通過，七Go組零SKIP／無OOM。正式596與stage bytes相同，55新檔、舊590項／5543SHA保持，完整5598／精簡4861；README三實際繪製。正常／集中人工／平台仍待，無新錄影或Release。

84～86收尾：私人素材57項差異逐一等於55個新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人提交識別碼。私庫狀態檔初讀早於git程序結束，保護檢查因此拒收；等待原程序完成後重新核對57項通過，未重啟或改動素材。9566個逐SHA相同、本人擁有的單連結暫存PNG改為相對連結，移除2330678733重複實檔bytes；原稿、R1差異圖、目錄與收據保持。無錄影，素材及Go容器已清理；公開提交與main核對接續執行。

84～86提交前核152份引用SHA、三份inline配方／連接表、原編製清冊與飛龍R2順序、208個README本地連結、既有README全部行及文化內容通過。工作清單與309條教訓守衛通過；本批工作根無root-owned或誤建.md目錄，素材及Go容器已清理。私人保存庫乾淨且遠端已核，#263／#268／#269維持OPEN。

2026-10-23 從b61f7349主線接88／89肩砲人物。原圖及已備兩待機稿重核，四銀行43幀仍缺，正式596項作新基底，保留先前590基底的參考收據。三十本體、十投射物與三空拍分開；沿既有header1分段與六階段consumer，不改規則／保存或推定新caller。先備DRAFT與原RGBA，正常／人工／平台仍待。

88／89編製準備初輪的內嵌腳本引號錯誤未產出腳本，改由獨立檔案語法核對後執行。編製R1於265:11落塵縮為1像素而alpha240被原守衛拒收，未發布候選；R2沿已通過第10拍的2像素高度下限，只改配方、不改RGBA或門檻。另事前修89末段殘光負上界錨點為來源內360高；R1配方及log保留，正式596項保持。

R2的88完整22幀編製通過，89於268:9殘光縮小alpha245拒收，未發布89候選。依路由重讀規格及來源250／合成255契約，實測藍束80／160取樣得255、320為245、360為225。R3只重編89，沿160尺度改用實際不透明尾段1800／365與2010／365收束，原稿RGBA／原時序／門檻保持，更多光束被砲身遮擋的近似明示；R2及88成品保持。

2026-10-23 88／89肩砲完整43幀：十新原RGBA與兩既有光效按SHA保存。R1落塵alpha240、R2藍束245拒收均保留；R3以來源實測縮小尺度及不透明尾點修正，原RGBA／門檻／幀／cue不改。兩包來源GPU、30本體核心、十投射物及三透明空拍、兩完整攻防／六階段F2／64及65 DAC／HP／RNG／一次續行、42編製、十三接點最大3.677HD像素與兩錯位反例、31新投射物／舊72十二反例及八閃色遮罩守衛、兩包四冷交易通過；六Go組零SKIP／無OOM／無逾時。正式600與stage bytes相同，53新檔、舊596項／5598檔SHA保持，完整5651／精簡4904。README四實際繪製，正常／集中人工／平台仍待，無新錄影或Release。

88／89收尾：私人庫55項差異完全等於53新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人識別碼。首次commit早於原git add程序完成，遭index.lock拒絕；等待原程序正常結束後重試通過，未刪鎖或重啟add。4887個逐SHA相同、本人擁有的單連結暫存PNG改相對連結，移除1184932493重複實檔bytes；原稿、失敗配方、目錄及收據保持。

88／89提交前：229份引用SHA、inline配方與連接表、原編製清冊、投射物profile與本體／空拍分段、210個README本地連結及既有全部README行通過；工作清單與310條教訓通過。本批工作根無root-owned或誤建.md目錄；沒有新錄影，素材及Go容器已清理。私人庫乾淨、實際遠端一致且PRIVATE；#263／#268／#269維持OPEN。公開提交與main遠端核對接續執行。

2026-10-23 已核前批公開c9b97ce4與遠端一致，原工作樹只有work暫存。91／92持杖六銀行50幀仍未登錄，原圖／描述子／PNG／Mask已依固定包重生。攻擊三hit與施術單MP分開沿既有consumer，先DRAFT原稿與來源接點，不改規則或caller。正常／集中人工／平台仍待。

91／92原稿共十三份、採用十一份，91前兩展臂稿的額外光暈拒收；第三稿改乾淨參考，原RGBA不修。DRAFT腳本第一版括號語法在寫入前即拒絕，已修。兩物理三hit、兩施術單MP標記保持；懸浮／消散與接掌分開，四十實際接點及全部120圖層縮小透明度探針通過，READY後才編製候選。正式600與5651舊檔保持。

91／92 Go R1兩來源程序因腳本將容器/tmp掛成唯讀，with-xvfb建立/tmp/xvfb.log前失敗，尚未進Go測試。兩份exit1收據保留environment_failure_r1；依路由查試玩契約，改掛/tasktmp唯讀而保留/tmp可寫，同映像檔／測試／清冊／包／預算以R2乾淨重跑，不改產品或門檻。42編製及八閃色遮罩守衛已通過。

等待最後精簡包施術預檢時，先備94／95四銀行十一幀的原生參考及兩待機原RGBA；目前僅DRAFT，不入正式或私人白名單。其原生參考以600基底產生，後續實作前重核即時清冊與資格；不回推本批來源／GPU覆蓋。

2026-10-23 91／92持杖完整50幀：十一新原RGBA與一既有金焰按SHA保存，兩額外光暈稿拒收。兩包50來源GPU、26物理與24實際施術核心、兩三段命中完整攻防／F2／64及65 DAC及兩普通0／五階段F2／MP1／HP7／RNG／一次續行，42編製、40接點最大4.077HD像素與兩錯位反例、14施術與八閃色遮罩守衛、兩包物理四冷交易及各兩施術原子預檢通過。R1掛載問題於Go前拒收，R2同預算十Go組零SKIP／無OOM／無逾時。正式606等於stage bytes，61新檔、原600項／5651檔SHA保持，完整5712／精簡4954。README四實際繪製，正常／集中人工／平台仍待，無新錄影或Release。

91／92收尾：私人庫63項差異完全等於61新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人識別碼。4975個逐SHA相同、本人擁有的單連結暫存PNG改相對連結，移除1285320753重複實檔bytes；原稿、拒收稿、目錄及收據保持。

91／92最終引用檢查先拒收modern_hd_command0_test.go：DRAFT consumer審查釘的是新增測試wrapper前的基底SHA。從公開c9b97ce4取出原檔精確快照，維持舊SHA並附原路徑／基底提交；受驗新版本另由tools欄位釘SHA，兩版本均可回查。不是產品或測試原始碼變更，Go通過收據保持，完整性檢查乾淨重跑。

91／92提交前：223份引用SHA、inline配方／連接表、原編製清冊、三物理命中與單MP及24實際施術核心、下一批DRAFT原生參考、212個README本地連結及既有全部README行通過；工作清單與311條教訓／179守衛通過。本批工作根無root-owned或誤建.md目錄，素材與Go容器已清理。私人庫乾淨、實際遠端一致且PRIVATE；#263／#268／#269維持OPEN。公開提交與main遠端核對接續執行。

2026-10-23 前批42f00ecb公開與遠端相同，主工作樹只有work暫存。重核94／95兩待機草稿及新606基底，合併97／98四人物八銀行36幀以共用驗證載入。原PNG／Mask／header／描述子重生，四個零header保持；94／95／97單hit、98雙hit與cue不改。97header byte2=4與八解碼幀照原資料保存，不猜欄位別義。先DRAFT原稿與配方，正式606／5712舊檔保持。

94／95／97／98十二新原稿含前批兩待機、三份既有刀光；98前兩待機光暈拒收，第三稿重新用原版與乾淨風格參考，原RGBA保持。36幀47圖層縮小透明度與十一實際接點通過，READY後編製。94／95無FX，97單次下劈，98兩反向斬擊；微小遠端塵痕省略與姿勢／刀光形狀近似均已揭露，不更動時序或規則。

2026-10-23 94／95／97／98八銀行完整36幀：兩包來源GPU、全部36物理核心、四完整攻防／三階段F2／64及65 DAC／HP／RNG／一次續行，84編製、十一接點最大2.796HD像素與兩錯位反例、八閃色遮罩及兩包各八冷交易通過；八Go R2組零SKIP／無OOM／無逾時。R1錯環境變數，R2只修容器素材入口同預算重跑，收據保留。十二新原RGBA與三既有刀光，48新檔，原606項／5712舊檔SHA保持；正式614與stage相同，完整5760／精簡4990。README四實際繪製，正常／集中人工／平台仍待，沒有錄影或Release。

94／95／97／98收尾：私人庫50項差異完全等於48新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人識別碼。4946個逐SHA相同、本人擁有的單連結暫存PNG改相對連結，移除1228010200重複實檔bytes；原稿、拒收稿、目錄及收據保持。

94／95／97／98提交前：155份引用SHA、inline配方／連接表、原編製清冊、原命中標記、214個README本地連結與既有全部README行通過；工作清單與教訓守衛通過。工作根無root-owned或誤建.md目錄，所有本輪容器已清理。私人庫乾淨、實際遠端一致且PRIVATE；#263／#268／#269保持OPEN。公開提交及實際遠端核對接續執行。

2026-10-23 前批ab2fc838公開與實際遠端相同，無殘留FD2容器，工作樹只有work。依現行614基底及remaining交集選99／100／101／112八銀行65幀；原PNG／Mask／header／描述子重生，無完全相同已登錄銀行。保留2／1／5／6次命中與112獨立cue，原零header不補造。先DRAFT原稿／分層配方，正式614／5760舊檔保持。

112眼光DRAFT原型：R1把地面火焰混入y130量測並錯假設第16拍仍連眼，於門檻拒收；原始碼與部分輸出保留。逐PNG核得2～15連眼、16分離、17～19無束，R2改真正上段與獨立尾段，R3依完整手繪筆尖及原生六倍線寬投影，十五段最大擬合殘差0.416原生像素。末拍六顆暗色像素另核，不因亮色門檻漏掉。編製子契約READY，僅新增離線資產投影工具，正常PNG／正式包格式／執行期／規則與保存不改；全65幀仍DRAFT，正式614保持。

99／100／101／112全65幀候選配方READY，十五新七共用原稿，184圖層縮小取樣通過。七光暈稿排除，兩斜光稿未採用；第15拍單像素火星原方案alpha88拒收，改緊實金色原紋理縮小，門檻保持。早期配方腳本的證據欄名、透明端點、301第6拍框緣錯誤已修正，未更動遊戲或產生正式素材。

112 R1候選未登錄；目視發現全火焰共用群組而受窄邊緣拍壓小，回DRAFT修為66獨立效果群組，184層取樣仍通過後READY。光痕直接原生座標核對錯把手繪眼部比例視為同身體，修為明示眼部地標映射，8HD門檻保持，R1光束21幀／15走廊已通過。清冊投影腳本誤假設所有資產有files，執行包實已同步5055，改依runtime_candidate的file／files集合計數；非產品缺陷。

100／101候選總覽發現後腳裁切，窄錨點改為完整身體範圍，101末拍14多餘拳影移除；183圖層重新取樣後READY。舊R1/R2候選未正式登錄，受原生框限制的站姿縮放近似明示。

獨立接點抓到全身錨點修改後100／101拳光未移，修連身拳光與身體光環、保留分離拳影世界位置；100末拍光環縮半徑保持中心與原框。183層取樣再過後READY。先前兩包65來源GPU各已PASS，但接點未過，停止後續Go並保留整體143退出與成功來源收據為superseded；不得沿用為最終驗收。

112完整流程在最後驗收報337/19缺GPU截圖，零OOM／逾時；原生第19拍delay0，既有IDA已證實尾段零次呈現，正式規則正確。65來源保留並全GPU核，物理核心改為64，零延遲另強制無job／無capture；不改delay。已讀58不重解閉合位址，READY後修測試假設，其他七Go通過保持，僅112同預算重跑。

2026-10-23 99／100／101／112八銀行完整65幀：兩包來源GPU、全部64物理核心、四完整攻防／三階段F2／64及65 DAC／HP／RNG／一次續行，84編製、九接點最大3.523HD像素及三錯位反例、十二眼光守衛、八閃色遮罩與兩包各八冷交易通過；九Go組零SKIP／無OOM／無逾時。十五新原稿與七既有光效、80新白名單檔，舊614項／5760檔SHA保持；正式622與stage相同，完整5840／精簡5055。README四實際繪製，正常／集中人工／平台仍待，沒有錄影或Release。

99／100／101／112收尾：私人庫82項差異完全等於80新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人識別碼。19991個逐SHA相同、本人擁有的單連結暫存PNG改相對連結，移除5021557625重複實檔bytes；原稿、拒收稿、目錄及收據保持。

收尾教訓guard初綁測試檔，但既有檢查器scannable刻意排除_test.go，改綁主收據zero_delay_capture_spec；不放寬檢查器。全專案擁有權掃描另見extracted、舊docs及wasm等歷史root-owned檔，未觸碰；本批新產物及ordinary-effects工作根擁有權檢查通過，不對全庫chown。

2026-10-23 接續85a0d9f1，前輪屬PROGRESS；重新讀主線、24遠端worklist及唯一現況。選104～109完整15銀行122幀，逐原包manifest／metadata／PNG／Mask核對、既有銀行無完全相同候選；原圖六總覽已檢視，三零header保持，DRAFT後製作手繪候選。物理96與施術26、122來源及兩包／私人保存均為本批閘門，未因難度拆掉龍首或施術。正式622及5840檔保持，無新錄影／Release。

104～109候選READY：31新原RGBA、一既有金環，122幀269圖層取樣通過，77接點。四拒收稿完整保存；兩翼關鍵姿勢共用、龍首視窗裁切、分帶殘火與旋流環形近似明示。107右邊界尾段由原PNG核為第8幀，第9幀無效果。正式622項保持，下一步編製及兩包實際GPU驗證。

104～109候選READY：31新原RGBA、一既有金環，122幀269圖層取樣通過，77接點。四拒收稿完整保存；兩翼關鍵姿勢共用、龍首視窗裁切、分帶殘火與旋流環形近似明示。107右邊界尾段由原PNG核為第8幀，第9幀無效果。正式622項保持，下一步編製及兩包實際GPU驗證。

104～109本批CONFORMED-SCOPED／RUNTIME-E1：122幀，兩包來源GPU、96物理／26施術核心、六攻防及三普通0交易／F2／返回，126編製、77接點及六錯根、21施術反例／三正對照、八閃色遮罩與兩包預檢通過，十五Go零SKIP／無OOM／無逾時。31新原RGBA與既有金環依白名單保存；正式637項／完整5993／精簡5177，舊622項／5840檔SHA保持。姿勢／翼型／裁切／光效近似明示，README九格為實際測試Draw。私人遠端、去重與收尾檢查另記，沒有新原版parity、錄影或Release，完整HD與集中正常／人工／平台仍待。

104～109收尾：私人庫155項差異完全等於153新白名單檔及catalog／manifest，獨立提交推送後實際遠端相同且PRIVATE，不公開私人識別碼。10546個逐SHA相同、本人擁有的單連結暫存PNG改相對連結，移除2987341825重複實檔bytes；原稿、拒收稿、目錄及收據保持。

104～109補記：R1只作編製審查；發現龍息接在上顎與殘火大光球後回DRAFT，R2改口內接點及小碎光，269圖層重新取樣再READY。其他72幀逐SHA保持，最終十五Go及全部來源／核心均依R2清冊驗證，沒有沿用R1 GPU結果。

104～109最終核對通過568個實際SHA參照、218個README本地連結與全部舊README行；312教訓的180個守衛及24條遠端工作清單訊號通過，沒有可自動關閉項目。GitHub #263已同步且驗收條件不變，#268／#269保持OPEN。私人庫獨立提交推送後實際遠端相同且PRIVATE，沒有公開私人提交識別碼；本批FD2容器皆已結束並清除。公開庫本批包含正式清冊、兩個專用測試入口、主收據、現況規格、README九格與遠端工作清單快照，完整HD及集中正常／人工／平台仍待。

2026-10-23 接續b5f7dbe1，前輪屬PROGRESS；主線乾淨，無FD2容器，主機gh登入及24項遠端worklist核對通過。新分支hd/remaining-figani-20261023納入全部剩餘33個已解碼圖庫284幀，十五份原圖總覽已看。343非共背景、121純爪光／空待機、124地塵及135僅待機均保留，不縮成較易通過子集；四施術也列入DRAFT。637項／5993檔保持，先做配方與驗證分型，無新原版parity、錄影或Release。

2026-10-23 剩餘33圖庫284幀配方READY。60份內建生成原稿保存，選50新稿與11既有來源；580圖層取樣、55接點與舊72十二投射物守衛通過，其他舊八profile逐項保持。R1縮小失敗108項保留，R2緊實微光／左右塵簇與窄站姿修正後零失敗；直接預覽顯示alpha0底下RGB導致誤判光暈，黑白Over合成證明實際透明。117／119金光ROI亦排除衣髮，343爆點、126柱身與127頭部修正。第二版腳本先因縮排在產檔前停止，修正後同容器／同預算乾淨重跑；未列產品缺陷。27個Go驗證尚待，正式637項未改，無錄影、新Release或完整HD聲明。

編製113通過，114在產檔前拒收camelCase母稿鍵；既有schema1只接受小寫KEY。回DRAFT將配方pose／group及接點名稱正規化後再READY，原RGBA／SHA、580變換與55接點數值及全部預覽PNG保持，未放寬編製器。113實檔保留並核對新配方SHA相同；114失敗log保留，後續編製用r2輸出目錄。

來源284與113攻防首輪通過，耗時分別566.764／598.705秒，無OOM／逾時，峰值約2.05／1.91GB。檢查發現battle HD載入的三個alpha OR旗標即使已全true仍全圖掃描，加入等價提前停止，原空白與PNG／SHA／幾何／Mask守衛保持；補末像素alpha1／255墨跡反例。當時已啟動的原程式二進位保留為基準，不混稱新程式；同命令重跑與效能量測待完成，尚未聲稱提速。

檢查攻防fixture發現完整主題載入後，程式原本即丟棄非battleAnimations欄位，演出只消費scene.modernResources。新增測試專用選取契約READY，所有請求圖庫仍經正式逐組loader完整驗SHA／PNG／原描述子；全284兩包與兩包完整冷預檢保持。這是測試載入範圍調整，收據明示，不宣稱正式啟動提速；正常遊戲不採此篩選。

126普通0實際Game.Draw／F2／MP／HP／RNG已通過，但Python獨立全像素核對兩次在原90秒限制逾時，均未產正對照或守衛通過收據；R1完整log及R2新輸出目錄與結構化觀察保留。重查知識路由及重製驗證reference，先量18個待機／施術來源約930萬個高覆蓋點的成本，以有界獨立程序觀察，不減少幀或像素、不改規則／素材或門檻。Go演出通過不能抵銷此尚未通過的獨立檢查。

126成本診斷以相同2GiB／2CPU與固定圖源核全部14施術來源通過：wall147.565秒、CPU約128秒，峰值1692954624 bytes，OOM／timeout皆零。這足以證明原90秒不足；測試驅動器新增明示--verifier-timeout，預設90與1～600有界範圍，本份126用300重跑全部七守衛。幀數、全像素及1000點／RGBA門檻不變；正式遊戲不採此測試參數，兩次失敗及當時驅動器快照保持，尚不以診斷正對照代替完整守衛。

成本點數精算勘誤：finalfigs-cast126-cost-oct23-r1.json的18張來源合計9491399個高覆蓋點，先前約930萬為初估，現況採約950萬。原逐幀量測不變；新增可重用的有界驗證預算教訓，入口與守衛綁定主收據。

126與127的Python獨立施術核對在明示300秒後，各完整七個守衛與全幀正對照通過，記憶體探針均無OOM／外層逾時。126兩次及127首次90秒失敗保留；124／125仍是原90秒通過，範圍0／601拒收。這是測試預算修復，全部34施術幀與全高覆蓋像素／1000點門檻保持，正式遊戲不採此參數。

完整精簡包四施術冷預檢R1在第四人物的完整主題PNG解碼觸及Go20分鐘上限，無OOM。查明wrapper每人物重讀一次670項完整主題，而測試交易原本已用深冷clone隔離。新增純測試批次契約READY：共用一份未預載完整template，每人物成功set及F2 reload仍各自冷clone，所有缺末幀／F2／HP／RNG／owner assert保持；一般單人物入口仍讀完整主題，正式遊戲不改。重跑保持原1200秒／8GiB／4CPU，保留R1失敗。

已主動停止仍逐人物重讀完整主題的候選包R1測試，Docker stop結束後client137，保留部分log與取消紀錄。這是被新測試取代的受控終止，不當成OOM或產品失敗；未完成memory終態不補猜。目標保持active，兩包改用原1200秒及獨立冷clone重跑，沒有省略人物、幀或交易。

2026-10-23 剩餘33圖庫284幀正式登錄，限定CONFORMED-SCOPED／RUNTIME-E1。兩包來源GPU、230攻防／待機與34施術核心、18FX／兩空拍；十四攻防、四普通0及135僅待機、適用F2／結算／DAC／一次返回，兩包完整主題冷預檢與三項等價alpha回歸，共三十項必要Go檢查通過，無SKIP或OOM。319編製、55接點與五錯根、48FX與28施術反例、八閃色遮罩及舊72十二反例通過。50新原稿與11既有來源沿白名單保存，正式stage bytes相同，舊637項／5993檔SHA保持；新增334檔，完整6327／精簡5461逐SHA及UID核對。README十九格來自實際Draw，58由正式audit重生264銀行2118幀／670項；正常／人工／平台、其他效果及408仍待，無新原版parity、錄影或Release。

等價alpha重跑完整284來源及完整主題113通過；113r2仍用完整主題，先前預期它使用selected-bank的文字更正。118起及135才在收據明示測試選取，正式遊戲沒有此篩選。共享主機／cache下時間不作正式啟動效能結論，舊基準與新收據分開保留。115編製後實際346/7 Draw另已目視確認，沒有以來源圖代替遊戲畫面。

本批私人庫336項差異已逐項等於334新白名單檔及catalog／manifest，獨立提交推送後實際遠端HEAD與本機相同，主機gh回讀PRIVATE，不公開私人提交識別碼。5985個本人擁有、單連結且逐SHA相同的暫存PNG改相對連結，移除2084764368重複實檔bytes，非淨可用空間量測；原稿、失敗、目錄與收據保持。全部本批測試容器已結束並刪除，後續文件與最終檢查使用一次性容器。

本批最終檢查通過875個實際SHA參照、220個README本地連結及全部舊文化保存行，僅兩個已核實現況句精確修正。315教訓的183個守衛及24條最新遠端worklist訊號通過，可能已完成0條，沒有可自動關閉項目。主機gh已更新#263且驗收條件保持，#268／#269仍OPEN；官方快照及91產生區塊已重生。來源264圖庫2118幀全部登錄只限素材覆蓋，其他效果／caller／背景／HUD、正常玩家／集中人工與平台仍待；目標持續active，沒有完整HD、新Release或新原版parity聲明。公開提交前確認本批FD2容器均已結束並清除。

2026-10-23 接續bc8259af，前輪為PROGRESS；主線及實際遠端相同，無FD2容器，主機gh登入與24條worklist核對通過。本輪hd/command9-effects-20261023先補敵方指令9的獨立FDOTHER44，既有十八普通特效與四召喚不重做。固定31來源、原描述子及PNG／Mask已依manifest核對並輸出總覽，existing IDA／typed producer為主證據，沒有新反組譯。DRAFT保留玩家9地圖owner及其他HD缺口，正式670項不改，不宣稱完整戰鬥HD或新PLAYER-E2。

2026-10-23 指令9補充：初版暫用圖源稽核誤設31幀皆非空，於第30幀中止且未產出收據。固定Mask確認第30幀是原生空拍後，以受版控稽核器重跑通過；不補畫空拍。新增五份原始RGBA保留不變，分開記錄母稿與正式素材；runtime來源橋接子規格審查提升READY，美術子規格仍DRAFT。

2026-10-23 指令9：31幀美術原型第二版補回末三幀的原生消散輪廓後，已檢視並提升美術子規格READY。來源重組測試119幀逐筆相同；首次末幀計數斷言誤設2，既有60目標幀只呈現一次effect30，訂正測試預期為1，不改產品排程。

2026-10-23 敵方9完成限定RUNTIME-E1：31來源兩包完整GPU、119幀獨立來源重組、完整Game.Draw199次HD呈現、九階段F2、MP1／HP20／RNG／一次續行通過。八項Go閘門、33編製與15GPU核對器反例、12清冊守衛及完整figani／battlepresent回歸通過，無SKIP／OOM。正式清冊671項，新增31圖層及四原RGBA，舊670項／6327檔SHA保持。README六格為實際Draw，原生留白與手繪近似明示；玩家9地圖、其他效果／背景／HUD及集中正常／人工／平台仍待，#263／#268／#269保持OPEN。初次連結CLI誤用得0項不計驗收，逐檔同parser核對已通過。完整收據、配方、反例與重跑入口依主收據command9_battle_hd_followup_20261023。私人同步、遠端與清理狀態於收尾核對。

2026-10-23 收尾：私人素材庫37檔差異經審查後獨立提交推送，實際本機／遠端main相同且PRIVATE，未公開私人提交識別碼。遠端#263已更新但保持原驗收，24項worklist快照／render／verify重生，#268／#269保持OPEN。5527個本人單連結暫存PNG逐SHA相同後改相對連結，移除1632604288重複實檔bytes，非檔案系統淨空間量測；原稿、拒收稿、截圖及收據保留。私人git add尚在執行時首次commit遇index.lock，沒有移除鎖或改檔，等add完成並檢查37檔後重新提交成功。無新錄影或Release。

2026-10-23 最後檢查通過：本批82項SHA引用／76個檔案、README222個本地目標及56／57／58／61／91全部本地連結通過。初版稽核範圍錯納歷史已刪執行檔、舊工具指紋及53份未修改root-owned舊檔；依現況／時間序列文件契約訂正，只嚴格核對本批完整SHA與全部修改檔／輸出1000:1000，不改歷史、不全庫chown。舊670項／6327檔保持另有正式收據。Go與素材Docker工作已退出，公開庫即將提交本批已驗證切片，work暫存產物不加入提交。

2026-10-23 玩家9地圖HD開工：前批f095ce9d已推送且tracked乾淨。路由命中spec-gated與文件職責，沿既有IDA閉合證據；不重開RE。新分支hd/player9-map-effects-20261023，遠端24項worklist核對保持。受版控稽核比對27效果及13數字全部PNG與原raw，發現5的bank metadata未列manifest，改以已列raw的既有LMI格式獨立重解；初灰階參考保留，r2依正式#0色盤。草案與輸入SHA集中player9_map_hd_followup_20261023，尚未接正式或登錄素材。

2026-10-23 依使用者要求先清理workspace：115714份已與正式素材逐SHA相同的暫存PNG改成相對連結，移除重複檔配置量20254834688 bytes；另刪舊凱麗驗證執行檔19632128配置bytes及兩個空快取目錄。目前HD使用中的Go快取、四張玩家9生成原圖、原版、正式／私人素材、Release、收據、截圖與獨有草稿保留。1458筆有界批次交界紀錄以4KiB取整回補配置量，配置量不宣稱共享主機淨空間增量。清冊671項實檔／尺寸／SHA驗證及98筆去重連結抽查通過；明細壓縮保存於work/cleanup-20261023.json.gz，入口work/cleanup-20261023.summary.json。去重後歷史runtime-only包不再是獨立實檔副本，後續獨立性測試需重建新包。Docker按唯一ID移除已被取代的FD2 Watcom快取33.87MB；六個現行工具映像檔、共用IDA映像檔及用途未確認的匿名磁碟區保留，沒有過期FD2 image／volume可刪，不做全域prune。唯讀全庫盤點超時後已停止相關容器，改以有界精準目錄清單完成；本輪沒有產品規則變更。

2026-10-23 玩家9接續：前輪清理為PROGRESS，原稿及現行工具保留。24條遠端worklist重新核對；玩家9完整圖層與typed快照契約提升READY，美術仍DRAFT。已確認同次HUD在LMI之下與事件copy(0,0)，原生閉合證據不重開；Noto字型固定b76b0433、Debian fonts-noto-cjk 1:20220127+repack1-1。五份原RGBA保存且alpha保持，尚未接正式圖源。

2026-10-23 依使用者要求再次優先清理：最新玩家9實檔包的單元、完整Game.Draw及來源GPU收據均為returncode=0且未逾時，核對5532份來源後，將其中5492份與正式素材逐SHA相同的副本改成相對連結，移除重複檔配置量1614417920 bytes。40份尚未登錄的新圖、全部原稿／配方／收據／截圖與現用Go快取保留。原runtime-manifest保留測試當時的獨立性證據，現況已改為連結，日後獨立包驗證須重建實檔。明細見[清理收據](work/cleanup-20261023-followup.json)。配置量不等於共享主機淨釋放量。Docker再次核對六個FD2現行映像檔、磁碟區與快取，沒有新增可確定過期的FD2項目；保留用途不明匿名磁碟區及其他專案，不做全域prune。大範圍唯讀盤點耗時已停止，改用清冊精準核對。清理後671項正式素材清冊驗證、100份連結SHA抽查、40份新圖與1000:1000擁有權檢查通過；輸出目錄沒有root-owned檔或誤建的.md目錄，git diff --check通過，本批FD2容器均已退出並移除。

2026-10-23 玩家9地圖HD完成限定RUNTIME-E1：27個LMI6效果與13份LMI5數字／MISS，兩包各120來源GPU與完整命中／MISS的176次HD呈現、16次F2、MP80→50一次／HP100→0或100／seed7／hold／一次續行通過。98張原生VGA由獨立覆寫器重組一致；正式路徑另用永久driver完整重跑，214.392秒、峰值2853666816 bytes，無SKIP／OOM／逾時。八項Go成功、五項編製測試及21個拒收案例均有真實正對照，最終編製守衛7.023秒通過。三次舊360/420秒有界逾時的oom均0，20秒剖析只定位掃描；SIGQUIT嘗試時程序已退出，沒有信號送達，不混為產品崩潰。PNG完整解碼與SHA守衛不變，alpha掃描等價微基準僅限掃描，不宣稱全遊戲效能倍數。

2026-10-23 正式清冊673項，除private_root回復外與受驗候選相同；舊671項／6362檔SHA保持，58新白名單檔含五原RGBA與13字形源。完整覆蓋重跑確認264個FIGANI／2118幀、23個FDOTHER／408幀及三組LMI／52張；來源408仍BLOCKED，資料數量不推定caller。私人6420實檔逐SHA與UID核對，60檔獨立提交推送，實際本機／遠端HEAD相同且保存庫仍PRIVATE，不公開私人提交識別碼。README四格為實際Game.Draw精確裁切，seed7、角色移位與受控命中明示；手繪共用／裁切／消散／光形近似保留。八Go與本批來源、工具、原提示及重跑命令集中player9_map_hd_followup_20261023，21項實作指紋及53份SHA引用已核對。

2026-10-23 #263已更新並保持原驗收，#268／#269保持OPEN；24項遠端worklist由官方函式pull／render／verify重生，0項可能已完成。README／WORKLOG／56／57／58／61／91共2384個本地目標核對通過；搜尋命中的舊待辦均為時間序列紀錄或其他HD缺口，沒有重開已完成玩家9。首次最後編製環境預檢用錯CATALOG變數，在執行測試前拒收，依程式的MAP_CATALOG修正後同工具乾淨重跑；未知下一切片檔名查無後改實際目錄定位，未改產品。repo-local作者原已符合wicanr2個人信箱，sandbox禁止重寫設定後唯讀確認，不改全域。新程式gofmt通過，所有本批修改與輸出1000:1000，未動53份歷史root-owned文件。本批FD2容器已退出，先前完成的1.61GB暫存去重保持，沒有新錄影。其他效果／10～12／地圖回復／背景／HUD及集中正常玩家／人工／平台仍待，完整目標保持active。公開庫即將提交推送本批，work及完整素材不加入公開提交。

2026-10-23 新增兩條可重用教訓：LMI同銀行按索引集合辨別角色，以及alpha掃描格式／Stride等價與量測範圍。首次guard指向測試檔，因既有工具明確排除_test而拒收；改綁正式modernMapSpriteAlpha入口，未改掃描器或測試，318條教訓／186個guard全數通過。

2026-10-23 接續087417ac，前輪為PROGRESS，公開本機／遠端已同HEAD，私人保存PRIVATE且乾淨。新分支hd/commands10-12-map-20261023從origin/main建立，24項遠端worklist核對。已沿閉合IDA確認10～12三surface raw取樣、60拍及11／12十輪前導，不重開RE；typed資料到HD分層、fixed-point子像素與LUT近似規格審查READY後才進實作，主收據commands10_12_map_hd_followup_20261023。

2026-10-23 指令10～12完成限定RUNTIME-E1：兩包各1083次主要Game.Draw與90次F2往返，三指令單目標命中／MISS及雙目標，完整十輪前導與restore／60主段／22結果／500ms hold、MP80→50一次／HP／seed7／Acted／一次續行通過。九項最終Go閘門無SKIP／逾時／OOM，必要測試逐名核對；舊2189A四項、原生敵方10及玩家9／死亡／alpha格式等共15個必要頂層回歸通過。

2026-10-23 驗證範圍勘誤：C10／11正式GPU先於新增13圖尺寸拒收；C12正式、三指令runtime及兩包最終單元使用補守衛後正式程式，有效圖源與排程不變。另將角色移到HD地形旁(8,5)，以獨立範圍核對11的58387內點／57708變色及12的101025／100331，兩包相同；該來源檢查與actor(1,6)完整Game.Draw分開，不冒稱正常玩家。永久驅動器後補必要測試逐名／OOM及非空HD覆蓋拒收，對九份不可覆寫的真實log再驗通過，未重寫舊收據或宣稱再執行Go。全畫面跨程序SHA不作相同聲明，初始地圖clock未控制；各張GPU均對同次獨立CPU Over逐RGBA核對至容差2。

2026-10-23 本批只接既有consumer，673項清冊與私人素材SHA保持，沒有新增圖稿或私人提交。HD子像素取樣、既有四色差近似與未HD地形／HUD明示；README四格為實際Game.Draw完整遊戲區半縮圖。#263更新仍OPEN，#268／#269保持集中人工；24項遠端worklist重生，0項可能已完成。所有修改與輸出1000:1000，本批測試容器已退出。5532份獨立暫存圖經測試及逐SHA確認後改相對連結，移除重複實檔配置1615429632 bytes，明細及測試前／清理後獨立性見主收據runtime_pack_after_cleanup；原稿、正式素材、收據與截圖保留，沒有錄影或新Release。完整戰鬥HD及13～16回復、其他效果／背景／HUD、集中正常／人工／平台仍待，目標保持active。

2026-10-23 接續385f12b2，從origin/main建立hd/battle-background-toolbar-20261023。依使用者最新指示先核對mm3工具列與戰鬥背景；13～16回復保留。路由命中規格閘門與文件職責，使用grilling及imagegen。使用者選擇加入遊戲加速但先看操作原型，並看過同狀態A／B後選定A頂端浮出、自動隱藏。正式程式及673項素材未改，新工具列／加速仍DRAFT。

工具列以受版控HTML、建構器及既有Playwright固定image重生。r1發現100%英文溢出，緊湊標籤訂正後r2通過；r3補來源／工具SHA及純工具列圖，12語系／字級、六速度、F2／F4、顯隱、39操作及七圖通過。這是固定Game.Draw截圖上的可操作原型，速度只改選單與示意條，不證明遊戲加速。使用者選定的純UI圖保存docs/figures/fd2-toolbar-floating-prototype.png並掛入61，不公開嵌入私人畫面的HTML產物。

固定原包BG／TAI各57資源逐manifest／raw／PNG／Mask SHA核對，末項各零長raw保持；BG十個／TAI十六個10×3均全透明。46可見BG逐indexed／mask不同；40可見TAI去重29種，33地形表第三欄涉及40索引，只描述表內容，不冒稱正常章內可達。320×101／318×100與台座各原框保持，總覽以正式FDOTHER#0色盤，不當作原版執行圖。稽核首次Docker socket被sandbox拒絕，改用既有容器的host權限後同命令成功，是環境邊界，不是產品失敗。

內建image_gen依BG_004原框及正確色盤總覽生成2243×701 RGB森林草稿，未修改原檔完整複製到既有work目錄，提示／SHA／輸入與限制集中background_toolbar_prototype_20261023。尚未轉正式1920×600、登錄清冊、同步私人包或接Game；其他背景／台座、正式工具列／加速、13～16與集中正常／人工／平台仍待，無新原版parity、錄影或Release。純原型與盤點不構成重大可玩更新，本輪不提交推送。

本批收尾：15項SHA引用、六個新增規格連結、既有入口正對照、新工具語法與全部修改檔1000:1000通過。主機gh已追加[#204](https://github.com/wicanr2/fd2_re/issues/204#issuecomment-6364115060)與[#263](https://github.com/wicanr2/fd2_re/issues/263#issuecomment-6364116053)進度，未改驗收或關閉；24項遠端worklist經官方純函式產生快照、render及verify，可能已完成0項。Docker檢查無本批遺留容器，沒有新增錄影需清除。正式遊戲程式未改，因此本輪未重跑Go回歸，瀏覽器原型檢查不替代正式玩家驗證。

2026-10-23 延續前輪原型實際前進至背景consumer：BG_004共用物理場景先補READY契約，新增battle_surface_hd的固定metadata／PNG／原框／alpha驗證與BaseScenery同次來源，再接兩側預備、body／counter／restore及DAC。工具列與實際加速未接；規則／保存格式不改。候選674項已編製，正式清冊仍673項。

兩包各三個必要單元閘門、24個獨立CPU Over逐RGBA GPU世界、我方／敵方完整Game.Draw通過，共八份Go gate，無SKIP／逾時／OOM。兩側124個原生預備／攻防／restore畫布獨立重組相同；候選與5533份全新實檔包的15／13張實際截圖SHA全部相同。呈現Draw次數因host排程不同，不宣稱同幀數。每案例64個戰鬥DAC及65個地圖DAC、三階段F2、HP／RNG及一次返回保持；強制攻防結果與地形4的fixture明示，非正常玩家或原版RGB對拍。完整共用GPU與七項舊路徑回歸仍在同一live程序執行，須通過才登錄正式素材。主收據bg004_physical_followup_20261023；README候選縮圖取實際完整遊戲區，不裁切重畫，台座／HUD仍像素。

2026-10-23 共用回歸勘誤：r1的GPU本體通過，隨後原生測試NewImage觸發鎖定版Ebiten v2.6.6的isRunGameEnded守衛，整包FAIL不計驗收，非產品缺陷或OOM。依實際module原碼與試玩路由核對後，永久driver拆為legacy-gpu／regression兩個新程序r2，保留舊失敗log與memory。彙整腳本首次誤讀stdout摘要鍵memory_peak_bytes，改讀原始JSON的after.memory.peak，舊收據不改；同程式重跑成功。新增可重用程序隔離教訓，guard綁正式driver。

2026-10-23 BG_004十份Go gate均已通過，包含分開程序的原生七項與完整共用GPU回歸；失敗r1不計驗收。正式清冊新增一項至674，與受驗候選僅private_root不同，舊673項保持，新增未修改原稿與1920×600圖源各一檔。兩包28張實際攻防截圖逐SHA相同，Draw次數不作相同聲明。私人同步、正式覆蓋與收尾仍待。

2026-10-23 BG_004限定CONFORMED-SCOPED／RUNTIME-E1：正式及獨立包驗收完整，native七項與2087共用GPU各在乾淨程序通過，全部十份Go閘門無SKIP／逾時／OOM。正式覆蓋重生為674項，264個FIGANI／2118幀、23個FDOTHER／408幀及三LMI／52張保持，新增BG_004一份可見單圖，不推定其他caller。私人6422實檔逐SHA／UID通過，舊6420檔保持；兩份新圖沿catalog白名單保存，其他原始研究BG／TAI不混入。README加入實際完整Game.Draw半縮圖，文化保存段落保持；56／57／58／61與唯一現況資料同步。工具列與實際加速仍為原型，其他BG／TAI／捲動／指令、13～16／HUD及集中正常／人工／平台仍待。Git同步、worklist與空間清理在最後收尾核對。

2026-10-23 BG_004收尾：私人素材庫四檔差異獨立提交推送後，實際本機／遠端main相同、工作樹乾淨且PRIVATE；公開資料只記核對結果，不公開私人提交識別碼。5533個本人單連結暫存PNG在四份獨立包PASS收據及逐SHA核對後改相對連結，移除重複配置1619103744 bytes，非共享主機淨可用空間量測；原稿、正式／私人素材、所有截圖、log、失敗樣本與原manifest保持。歷史收據描述驗證當時的獨立實檔，現在的包不可再當獨立驗收，未來需重建。明細與最後檢查依主收據runtime_pack_after_cleanup及work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/bg004-final-check-oct23-r1.json。

2026-10-23 最後檢查通過58項本批SHA引用、12份來源指紋、2406個本地Markdown目標及README全部舊文化行／標題保持，所有修改及輸出1000:1000，gofmt無輸出；320條教訓／188個guard通過。主機gh已更新[#263](https://github.com/wicanr2/fd2_re/issues/263#issuecomment-6365410806)，保持總驗收與OPEN；官方函式從最新遠端資料pull／render／verify共24條，可能已完成0條，#268／#269仍待集中人工。全部本批Docker容器已退出並清除，未建新image／volume，未錄製新影片。此批已有真實可編輯來源到Game.Draw的切片及真實回歸，符合重大更新條件，即將提交並推送公開主線；推送後主機Git核對輸出於bg004-public-git-verification-oct23-r1.json。完整HD目標保持active。

2026-10-23 接續6b4e2679，前輪森林背景為實際PROGRESS；tracked乾淨、無FD2容器，從最新origin/main建立hd/tai004-pedestal-20261023。沿spec-gated路由與既有TAI4 metadata／原框／遮罩、NativeBattlePanelOrigin及零底圖證據，不重開RE。RGBA台座原稿2170×725保留不變，alpha64核心裁切框與完整提示已記tai004_physical_followup_20261023；來源橋接READY，美術編製尚待端點及正式合成核對。計畫先去掉HD底圖中已烘的原生台座，再沿同次來源補靜態／移位層，避免舊像素殘影；正式清冊仍674項，其他caller與集中人工保持待辦。

TAI_004候選675項編製及單元通過，原框154×42轉924×252透明圖，保留未修改母稿。GPU首輪在prelude side0 stage6的byte7293841得到3而舊CPU期望1，收據保留並拒收；原因是舊CPU直接對straight RGB取整，漏了鎖定Ebiten v2.6.6 imagetobytes.go的draw.Src RGBA8預乘量化。比較器改為先依相同公開貼圖格式量化，再獨立計算subtract及Over，原容差1與正式shader不改。第二輪50個世界通過；另把TAI-only測例清除所有人物／背景HD集合，避免僅靠既有HD人物啟用而空泛通過，後續乾淨重跑。工具列A定案保持，新工具列及實際加速仍未接入。

2026-10-23 TAI_004限定CONFORMED-SCOPED／RUNTIME-E1：兩包各完整675項主題與原子拒收、124張原生重組、50個GPU世界及雙側完整Game.Draw通過；九份必要Go閘門含七項舊回歸，無SKIP／逾時／OOM。補強前source r2另為通過樣本，最終採只含TAI的r3，不以重複案例增加覆蓋。每案例三階段F2、64／65 DAC、HP／RNG及一次返回保持，28張跨包PNG逐SHA相同，Draw數不作相同聲明。未改shader，2087共用核心沿前批，未冒稱本批重跑。正式清冊675項，舊674項保持，新增未修改原稿與924×252圖源；264 FIGANI／2118幀、23 FDOTHER／408幀與三LMI／52張保持。

私人6424實檔逐SHA／UID通過，舊6422檔保持，四檔獨立提交推送；主機核對本機／遠端main相同、工作樹乾淨且PRIVATE，公開資料不記私人提交識別碼。README實際完整Game.Draw半縮圖換成森林及台座，面板保持像素，文化內容不改。5534份暫存runtime圖在四份獨立包閘門及清理前後SHA驗證後改相對連結，移除重複實檔配置1619914752 bytes，不宣稱共享檔案系統淨可用空間量。去重後包不能再當獨立實檔；原稿、截圖、manifest、log及失敗樣本保留，沒有新錄影或Release。#263已更新並保持OPEN，其他背景／台座／caller、13～16回復／HUD、正式浮出工具列／速度與集中正常／人工／平台仍待，完整HD目標保持active。

最後檢查：45項本批SHA引用、九份來源指紋、2297個本地Markdown目標、gofmt及全部本批輸出1000:1000通過；README僅替換HD圖說與實際縮圖，其餘文化行／標題保持。320條教訓／188 guard通過，24項worklist、可能已完成0項。全庫find的r1雖已產出無本批root檔報表，外層退出124，不計成功命令；縮到本批輸出與程式／工具工作根的r2乾淨成功，既有root檔未chown。無新映像／volume／錄影，最後核對沒有FD2容器遺留後提交推送。

2026-10-23 前輪530fd6a4已推送主線，屬實際PROGRESS；從最新main建立hd/heal13-16-20261023。接續回復13～16：既有21EB1／tail schedule及LMI源已核對，17來源逐固定raw／PNG／metadata通過。先補61 READY橋接，藍環新母稿1254×1254 RGBA原檔保留，最高alpha254不人工量化；編製／正式串接／驗收仍待，其他HD與工具列不外推完成。

回復HD候選677項／17圖已編製；正式清冊仍675。新原稿alpha最高254沿既有LMI coverage契約保存，不人工加255。首輪單元的actor(1,6)到target(10,7)超出明示SelectionMode1，原生候選守衛正確拒收；只改測試為相鄰actor(9,7)，多目標依法包含施術者共三人，規則未改。r1已有FAIL而仍在完整主題讀取，人工停止以釋放資源，Docker stop的137不是SignalKilled產品結論，保留partial log且不算驗收。r2已通過425984個獨立raw-work幾何點與116張完整LMI畫布，完整主題及八案例GPU仍執行中。七項舊路徑回歸已通過，無SKIP／OOM。

候選單元r2與完整主題原子F2通過，無逾時／OOM。補充source gate通過180張前導／效果／數字來源重組、425984個獨立幾何點及17圖／18個冷載入錯件反例。第一次八案例GPU走到首例交易時，延遲閉包錯抓range最後的id16，因go.mod為go1.21而非工具鏈go1.22的新語意；只在測試建立command區域副本，正式引擎不改。失敗整包保留、不列驗收，乾淨r2重新跑。

候選GPU r2第一個單體流程完成，但多目標疊光時image/draw的16-bit整數Over與RGBA8逐層取整累積三階差，整包拒收且不放寬容差2。新增僅供回復測試的獨立RGBA8 scalar Over，比照實際上傳先量化預乘RGBA，再以浮點Over每層取整；共用GPU與遊戲程式不改，r3待驗。另依既有nonvacuous教訓，把fixture移至actor(10,7)、target(11,7)、第三目標(11,6)，確保最小半徑13也掃到HD樹林；新增四指令各至少100個HD範圍內／變色點、範圍外零變色的必要單元訊號。先前單元成功只列其舊位置範圍，不取代新增閘門。

2026-10-23 回復13～16限定CONFORMED-SCOPED／RUNTIME-E1：兩包七份最終Go閘門通過，無SKIP／逾時／OOM。每包完整677項主題、180張前導／LMI畫布、425984個獨立幾何點、17來源GPU及18個冷拒收、八完整Game.Draw與88次F2通過，48張跨包PNG逐SHA相同。四指令HD範圍內／變色數兩包相同，範圍外0；計數為16幀HD texel訪問總和，不當作唯一畫面座標數。MP／HP上限／RNG／Acted與原owner一次交易／續行保持，固定seed7及受控紀錄／位置明示；不是正常玩家或原版RGB對拍。峰值約5.97／5.85GB包含八個保留測試Game與比較快取，不當一般玩家效能量測。

正式清冊677項，舊675項保持；17成品、一原稿與十個直接Noto字形來源共28新白名單檔。私人6452實檔逐SHA／UID通過，舊6424檔保持，獨立Git提交／遠端核對另記。264 FIGANI／2118幀、23 FDOTHER／408幀及兩BG／TAI保持，LMI擴為五群69圖，量由formal_coverage重生。README四格取候選最終PASS的實際完整Game.Draw半縮圖加外部標籤，不裁遊戲區，文化內容保持。

5551份runtime暫存圖在三份最終獨立包閘門與清理前後SHA核對後改相對連結，移除重複配置1619963904 bytes；另核固定SHA刪19,484,784 bytes的臨時Noto字型複本。不是共享主機淨可用空間量測，已去重包不能再當獨立實檔；原稿、字形來源、manifest、全部PNG／log與拒收樣本保持。沒有錄影或新Release。其他owner、背景／台座／HUD／地形、浮出工具列／速度與集中正常／人工／平台仍待，完整HD目標保持active。

本批收尾：私人素材已獨立提交推送，主機核對本機／遠端main完全相同、工作樹乾淨且PRIVATE；公開檔只記核對結果，不公開私人提交識別碼。#263已追加本批進度且保持OPEN，#268／#269集中人工仍待；遠端24項worklist已pull／render／verify，可能已完成0項。44項SHA引用、九來源指紋、2413個本地Markdown目標、gofmt、321條教訓／188個guard及所有修改／輸出UID1000:1000通過。僅在本批工作根執行find，無root產物或.md目錄，不掃全庫或chown舊檔。全部FD2容器已退出；沒有新映像／volume或影片。相關收尾依heal1316-final-check-oct23-r1.json；接著提交並推送公開main，實際HEAD／遠端核對保存於同批public-git-verification。完整HD目標維持active。

2026-10-23 從93ce12ec主線建立hd/reward-message-20261023，續接#204已知物品／金錢成功訊息像素缺口。原版來源沿58已閉合死亡效果與既有caller，不開新RE。61先記READY，再以凍結typed地圖／work／色盤串接HD世界、Noto文字與既有480頭像；六開、等鍵、五收與restore仍沿原owner。清冊及美術不變，目前驗證待完成，不能列CONFORMED或正常玩家完成。重跑入口tools/verify_fd2_reward_message_hd.py及HD主收據reward_message_hd_followup_20261023。

2026-10-24 成功訊息初輪兩項單元與八項舊回歸通過，補強金額0與123456789逐字HD檢查後單元r2亦通過；兩項HUD回交／唯讀觀察另通過。完整Game r1在開框第0拍框外30,594差3拒收，r2同門檻保存原始離屏／最終PNG。原始離屏在本輪面板以外有96720個非透明點，明確顯示resetBattle留下的開戰對話；這是fixture初始狀態，不作產品缺陷。rewardHDTestGame明示初始開戰對話與橫幅已結束，正式引擎／色差2不變，r3乾淨重跑。舊失敗收據與PNG保留，不計驗收。

2026-10-24 成功訊息限定CONFORMED-SCOPED／RUNTIME-E1：最終unit r3、game r3、八項regression及兩項hud四批13項必要Go測試通過，無SKIP／逾時／OOM。物品192與金額123456各六開、等ACK、五收與restore，共26完整Game.Draw與26次F2；頭像selector9為受控fixture，原生面板框外HD世界、鏡向／移動裁切、金額0及123456789逐字、HP／MP／RNG／owner／原final與一次續行通過。關框後才加金幣與HUD回交保持。清冊677項及全部美術／私人白名單未改；README雙圖由兩張完整1920×1200等待畫面半縮並排，不裁切或重畫。其他成功問句、到期訊息、HUD／地形、背景／caller、浮出工具列／速度及正常／集中人工／平台仍待，不宣稱完整HD或新原版parity。

收尾核對：#204與#263已更新遠端內文、原驗收條件保持且皆OPEN，提交前重新拉24項worklist並render／verify，可能已完成0項。322條教訓／188 guard通過；41項SHA引用、八份來源指紋、2431個本地Markdown目標、gofmt與本批輸出UID1000:1000通過。完整主題加兩個保留測試Game的峰值3430088704 bytes，只作測試資源量測，不外推正常玩家效能。本批沒有錄影、Release、新映像／volume或大型素材複本，十張通過PNG、兩張拒收PNG與原始log／收據保存。FD2測試容器已退出；公開提交／遠端核對另記reward-hd-public-git-oct24-r1.json。

2026-10-24 從73bca93e主線建立hd/forest-coast-20261024，回到戰鬥HD背景缺口。新增BG005森林內部／006海岸與TAI005／006四原稿，沿已核准手繪風格。原始P圖預覽為灰階，構圖參考原圖、配色沿固定色盤總覽與明示材質；原RGBA、黑白alpha合成與縮小配方保留。61先DRAFT再依審圖／透明端點升READY，既有編製器追加固定配方批次，候選681項、舊677項保持；8新增白名單檔，舊檔連結共用。獨立runtime建立5555實檔。

森林source GPU首批50世界／CPU Over容差1已通過；unit5首輪冷來源／錯件與完整681主題已通過，但原生重組仍含source.Resource!=4的舊斷言而整包拒收。只把測試來源與155寬度改為實際所選5／6與native.Width，正式renderer／shader不改，舊失敗保留。接著依序驗兩來源、兩包、雙側完整Game.Draw與舊4回歸，最多兩Go工作並行；未完成前不登錄正式清冊。

2026-10-24 森林內部／海岸場景限定CONFORMED-SCOPED／RUNTIME-E1：19份候選／5555獨立實檔包／舊4來源Go閘門，以及同步後正式unit與GPU兩份，共21份通過，無SKIP／逾時／OOM。八完整Game.Draw、24次F2、64／65 DAC與一次返回，56組跨包PNG逐SHA相同；兩包各兩來源原生重組與50世界GPU、冷錯件原子性通過。原Go renderer／shader／規則／保存不改，新增4個battle_surface_hd項目，舊677項及6452私人檔保持，新四原稿／四成品皆私人白名單，6460實檔／SHA／UID通過。README雙圖取兩個同段落的完整最終Game.Draw半縮並排，不裁切／重畫，像素HUD及手繪構圖／比例近似明示。正常／集中人工／平台及其他來源／caller仍待，完整HD目標保持active。

私人6460實檔已獨立提交推送，主機核對本機／遠端main相同、工作樹乾淨；公開資料只存核對結果，不公開私人提交識別碼。編製器以固定基底／原稿重跑，八份新檔及正式assets陣列完全相同，inline配方與前677項基底均可精確重建原SHA。5555份runtime實檔與兩份候選各八新實檔，在正式／私人來源逐SHA核對後原子改為同SHA相對連結，移除重複配置1626701824 bytes，不當共享主機淨可用空間量測。原稿、全部PNG、recipe／manifest／log／拒收與通過收據保留，已去重runtime不能再作獨立驗收。沒有新影片、Release、映像或volume。

森林／海岸最後檢查通過101項SHA引用、九份來源指紋、2443個本地Markdown目標、gofmt及本批輸出1000:1000。README除明示現況句及新雙圖外逐字保持；323條教訓／188個guard通過。遠端24項worklist已pull／render／verify，#263保持OPEN，集中人工#268／#269不關閉。FD2測試容器全部退出，沒有新映像或volume；最終收據為forest-coast-final-check-oct24-r1.json。公開提交／遠端核對保存於forest-coast-public-git-oct24-r1.json，完整HD目標保持active。

2026-10-24 從e9cdc37e主線建立hd/scroll-scenery-20261024。沿已閉合29510／29C90／29DED及連射證據，61先記READY，準備把分離背景物理的同次BG／TAI身份接入預備、departure、scroll、連射與反擊。原生計畫與BG0..2遮罩歷史保持，不改清冊或美術，不新增原版parity或正常玩家聲明；驗收尚待。

2026-10-24 分離背景場景限定CONFORMED-SCOPED／RUNTIME-E1：三組4:5／5:6／6:4、雙側各1038張原生畫布及120世界GPU／CPU Over容差1通過；兩包十二完整Game.Draw、72次F2、64／65 DAC、HP／RNG與一次返回保持，36組跨包PNG逐SHA相同。24份Go閘門／32必要頂層測試含舊4／5／6及原生／分段指令／完整GPU核心回歸通過，無SKIP／逾時／OOM。首輪driver漏帶兩項分段指令素材環境，整包依SKIP拒收；r2補明示環境乾淨成功，不改正式規則。清冊681項與私人美術保持，沒有新圖稿或同步需求。README四格沿實際完整Game.Draw半縮，不裁切／重畫，原生BG0..2過渡及HUD明示。正常／集中人工／平台及完整HD仍待；字串重綁、worklist與清理另核。

5555份新runtime實檔在十二攻防／來源閘門完成及正式來源逐SHA核對後，原子改為相對連結，移除重複配置1600884736 bytes；不宣稱共享主機淨可用空間量測，清理後不能作獨立實檔。108項字串人工處置以受控遷移器重綁並通過目前盤點測試，處置數保持。可由公開b4c298d1重建的舊字串來源樹及臨時tar已刪，兩份inventory與測試／拒收收據、原稿、PNG及正式／私人素材全部保留；沒有新影片或Release。

本批收尾通過181項SHA引用、九份來源指紋、2450個本地Markdown目標、gofmt與1000:1000擁有權。README除明示現況句與捲動圖說外逐字保持；323條教訓／188個guard保持。主機gh更新#263並再次取得遠端24項worklist，於Docker呼叫官方build_snapshot／render／verify更新快照，可能已完成0項；#268／#269保持OPEN。所有FD2容器已退出，沒有新Docker映像、volume或錄影。最終檢查依scroll-scenery-final-check-oct24-r1.json；公開提交／遠端核對保存於scroll-scenery-public-git-oct24-r1.json，完整HD目標保持active。
