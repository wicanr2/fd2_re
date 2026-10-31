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

2026-10-22 32／34 caller勘誤：先前由普通0雙marker拒收便把caller列未知，漏查已有正式路徑。現有native_command24_presentation.go明示selector32／98及固定15幀schedule；native_command29_presentation.go明示selector34／104及derived-strike分段schedule，58亦已有29的RE／indexed E1入口。這足以訂正未知描述，不需重開EXE。24／29兩owner後續已接HD，現況見61的split_strikes24_29_followup_20261022；原始33張第三銀行來源證據保留，不改原標記或規則。

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

2026-10-24 從5da8971b main建立hd/command24-29-scenery-20261024。前輪捲動HD為實際PROGRESS，tracked乾淨且無FD2容器。沿原指令24／29與29C90閉合證據，61先READY，準備補actor／各target的HD場景、原規則適用TAI、RNG前及F2來源預檢；不改清冊或美術，驗收尚待。

2026-10-24 指令24／29場景限定CONFORMED-SCOPED／RUNTIME-E1：三組地形與雙側，兩包各3018原生畫布、36組Go規劃前錯件及36組F2拒收、72項純場景／原生回退含48實際GPU對CPU Over通過。24完整Game.Draw、156次F2、MP一次／逐目標HP一次／原擲骰及一次續行保持，870完整1920畫布對固定FIGANI／BG／TAI及獨立Pillow合成容差1，跨包PNG雜湊相同。Runtime逐張回讀與編碼、僅不留重複PNG；兩指令各20收據反例、舊396／2087 GPU、物理與冷載入及晚期回復保持，36份Go閘門／44必要頂層測試零SKIP／逾時／OOM。清冊681及私人美術／shader／規則／保存保持，README四格取實際完整畫面半縮，不裁切／重畫。其他場景／過渡／HUD與正常／集中人工／平台仍待，不稱完整HD。字串綁定、worklist與清理另核。

5555份runtime新實檔完成全部閘門及正式來源逐SHA核對後，原子改同SHA相對連結，移除重複配置1600626688 bytes；不宣稱共享主機淨可用空間量測，清理後不可再當獨立實檔。108項人工字串處置以受控遷移器重綁並通過目前盤點，處置數保持；可由5da8971b重建的舊來源樹及臨時tar已刪。正式870畫布、runtime編碼雜湊、base／native PNG、log、manifest與通過／反例收據及原版／私人美術保留；沒有新影片或Release。

本批最後核對195項SHA引用、十份程式來源指紋、2460個本地Markdown目標、gofmt及1000:1000擁有權通過。README僅更新分段打擊圖說及同名實際縮圖，其餘內容逐字保持；323條教訓／188 guard保持。主機gh更新#263並再次取得24項worklist，由Docker內官方build_snapshot／render／verify更新快照；#263、#268及#269皆保持OPEN。所有FD2容器已退出，沒有新Docker映像、volume或錄影。最終收據為cmd2429-scenery-final-check-oct24-r1.json；公開提交／遠端核對另存cmd2429-scenery-public-git-oct24-r1.json，完整HD目標維持active。

2026-10-24 從ee0900c6 main建立hd/transition-bg-20261024。前輪指令24／29場景已推送，屬PROGRESS；tracked乾淨且無FD2容器。本輪補BG0..2過渡背景，固定原圖三份RGBA色盤參考與320×100／全不透明來源已核對，非單一循環位移不作假設。61先記來源橋接READY／美術DRAFT，三份生成原稿各以原圖參考製作；審查／編製／正式登錄與驗收尚待。

過渡背景十二完整物理流程／72次F2通過。跨包比較第一版把整份收據相等當成條件，因排程造成的final_draws／hd_draws／phase_hd_draws不同而拒收；改為明示排除這三項計數，其他欄位及60組完整PNG逐SHA相同。正式renderer與測試閘門不改，來源／每階段實際繪圖仍必要。舊Git快照解包初次使用容器Python尚未支援的tarfile filter參數失敗，檢查自有Git archive全部路徑與連結後以既有API解開；屬驗證環境問題。

跨包物理比較最終勘誤：首段的60組及第二版已通過敘述不成立。六組逐欄與逐圖檢查確認，排程計數共四項可不同；左右轉場的最後歷史來源不同，實際擷取鍵為左10／右6，不能兩側均假設10。依原phase／來源集合核對後，共48組完整PNG逐SHA相同，四項以外收據欄位相同，全部必要來源計數皆正值。三份過渡圖實際參與Draw不等於每個歷史層同時可見；不改正式producer、遮罩、規則或Go閘門。最終證據transition-physical-paired-oct24-r1.json。

GitHub連接器無權查私人素材庫；沿用主機Git已登入路徑fetch成功，兩庫本機與origin/main皆無分歧。另在不帶主機憑證的有界Docker查匿名GitHub REST，私人庫回404；與登入Git可達並列記錄，不搬運憑證、不改公開方式。

2026-10-24 BG0..2過渡背景限定CONFORMED-SCOPED／RUNTIME-E1：三份新原稿／成品登錄，清冊684項，固定配方六檔逐SHA可重編，舊681項及私人6460檔保持。兩包36完整Game.Draw、228次F2，918組最終PNG跨包雜湊相同；分段870畫布對固定來源／Pillow合成容差1。兩包原生重組、來源拒收、僅過渡95 GPU及舊681原生回退通過，67項收據反例通過，共59份Go閘門／79必要頂層測試無SKIP／逾時／OOM。README四格取實際完整畫面半縮，未裁切或重畫；手繪條帶／色彩／比例近似明示。原producer／規則／音效／保存／shader保持，其他場景／效果／HUD及普通／人工／平台仍待，完整HD目標保持active。字串盤點、Git及清理另核。

5558份runtime實檔及兩份候選共12新檔，在全部閘門與正式來源逐SHA核對後原子改為同SHA相對連結，移除重複配置1617747968 bytes；不是共享主機淨可用空間量測。清理後runtime不再是獨立實檔。108項字串人工處置經官方遷移器重綁／盤點測試通過，處置數保持；可由ee0900c6重建的舊樹及臨時Git tar已刪。原稿、正式／私人美術、最終PNG、來源／配方與全部收據保留，沒有新影片、Release、映像或volume。

本批最後核對262項SHA引用、13份來源指紋、2471個本地Markdown目標、gofmt及1000:1000擁有權通過。README只更新既有捲動圖與限定圖說，其餘內容逐字保持；323條教訓／188 guard保持。私人庫已獨立提交推送，登入Git核對本機／遠端main相同且工作樹乾淨，匿名API404並列記錄，不公開私人提交識別碼。主機gh更新#263並再次取得24項worklist，由Docker內官方build_snapshot／render／verify更新快照；#263／#268／#269保持OPEN。所有FD2容器已退出，沒有新映像、volume、錄影或Release。最終證據為transition-final-check-oct24-r1.json；公開提交及遠端核對另存transition-public-git-oct24-r1.json，完整HD目標保持active。

2026-10-24 上輪56a527c0已推送、tracked乾淨且無FD2容器，屬實際PROGRESS。從目前main建立hd/remaining-battle-scenery-20261024，本輪整批補剩餘77個可見BG／TAI來源，共66份新原稿，其餘依原raw相同關係共用。來源稽核與既有caller定位支持318×100／320×101橋接，61先READY來源／DRAFT美術。原mask／UI邊界／空槽與美術近似明示；圖稿、正式登錄及驗收尚待。主機sandbox首次建branch遭.git唯讀拒絕，改授權Git執行成功，非產品問題。

全部46張可見BG的三種尺寸、雙側前導／主攻／反擊／restore已由正式native producer驗5704張完整畫布重組通過，10個空Mask保持；首輪source-native閘門無SKIP／逾時／OOM，cgroup峰值1162260480 bytes。此處只證明新尺寸橋接保留native輸出，尚未新增HD素材或普通玩家驗收。

66份新原稿及77份六倍成品已審查，八張實際alpha總覽保留；TAI14／47的生成器周圍顏色在實際合成中不可見，沒有重畫alpha。配方及來源逐SHA核對後升READY。先前DRAFT編製反例已確認拒收且不建立輸出；正式清冊仍684項，接著才編候選與執行期驗證。

總覽首稿誤取淡入首拍，技術組版通過但目視過暗；原稿及來源保留，第二版改用完成淡入的departure拍。遊戲畫面不裁切或重畫，不改正式演出。

字串重綁首檢以純Git快照當舊輸入，因缺少目前生成輸入而拒收，未修改review。追查確認目前盤點逐bytes等於前批已接受的盤點，108筆候選與人工處置已正確綁定，本批無需遷移。差異清單與拒收原因記於allscenery-string-review-oct24-r1.json，正式Go盤點測試另驗。

2026-10-24 剩餘場景限定CONFORMED-SCOPED／RUNTIME-E1：66新原稿、77來源登錄，正式761項／46 BG／40 TAI；舊684項及6466私人檔保持。兩包各86冷來源與172 GPU世界、三種原框5704原生畫布、52完整Game.Draw／276次F2及1030組跨包PNG通過，分段870張另對Pillow容差1，兩指令各22反例。舊684主題八來源16世界原圖回退通過；74份HD Go閘門／91必要測試及字串八項無SKIP／逾時／OOM。原producer／規則／音效／保存／shader保持，README新四格為實際完整畫面半縮；普通／集中人工／平台及其他caller／HUD／浮出工具列／速度仍待。私人6609檔已依白名單同步，遠端及清理另核；完整HD目標保持active。

5635份runtime、兩份候選各143份新檔及66原稿／77預覽，在全部閘門、私人遠端與清理前後SHA核對後改同SHA相對連結，移除重複配置2448732160 bytes，不當共享主機淨可用空間量測。去重後runtime不再是獨立實檔。可由56a527c0重生的本批字串舊樹及Git tar已刪；原稿、正式／私人美術、最終PNG、來源／配方、通過／拒收收據與log保留，沒有新錄影或Release。

最後審查發現冷拒收測試的保留marker55與TAI55碰撞，可能只驗到重覆拒收；只移至未用56，補跑正式全部86來源通過。完整Game.Draw及正式引擎保持，舊測試版本快照保留；最終75份HD Go閘門／92必要測試。最後Go變更後重生的字串盤點仍逐bytes等於既有審查，不改108筆處置。

最終檢查通過386項SHA引用、10份來源指紋、4675個有效本地Markdown目標、gofmt與本批1000:1000擁有權。三份未改動的歷史交付筆記有八個舊輸出連結不可用，與base56a527c0逐bytes相同，分列於收據；初版檢查器另誤把行內／縮排程式碼當連結，修正後無本輪新增缺鏈。收尾封裝命令的引號錯誤在執行前拒收，改用檔案腳本，沒有修改正式程式或驗證門檻。323條教訓／188 guard通過；主機gh更新#263後重新拉24項worklist，由官方build_snapshot／render／verify同步，可能已完成0項，#268／#269保持OPEN。私人已獨立提交推送，登入Git核對HEAD／遠端相同且乾淨，匿名REST及HTML404，不公開私人提交識別碼。最終收據allscenery-final-check-oct24-r2.json；公開提交／遠端核對另記，完整HD目標維持active。

本批Go／Xvfb及素材／Pillow容器已全部退出並刪除；沒有新映像、volume、錄影或Release。公開提交後核對HEAD／遠端與乾淨工作樹，結果保存allscenery-public-git-oct24-r1.json。

2026-10-24 物理戰鬥面板HD文字，限定CONFORMED-SCOPED／RUNTIME-E1。沿核准1920／Noto／原位置，保存每次typed數值、raw名稱與原生149×42，逐byte匹配source base才附加。原框、bar、overflow符號、層序與保存保持，其他caller及完整HUD仍待。四語228姓名ink、24 GPU畫布、六完整Game.Draw／24次F2、72實際PNG及README四語圖組重生通過。11 HD閘門／24必要測試、七原生面板及八字串測試通過，峰值2611900416 bytes，沒有SKIP、逾時或OOM；這是受控fixture與測試快取範圍，不是正常玩家／平台效能。

本輪三種工具／審查失敗均保留：首個compile-only命令缺DISPLAY，改同工具鏈Xvfb乾淨重跑；字形測試最初誤設80個有效姓名，實際目錄為57×4；字串遷移保留舊108分類後，新LV標籤仍須審查，已補player_visible一筆至109並重跑八項。實際畫面另抓到原字模左下陰影，依fdtxt.Font.BlitNativeGlyph寫入足跡修清除區，重跑四語／GPU與完整流程；舊失敗圖保留。Pillow image未提供DejaVu，外部標籤改內建字型，遊戲圖未重畫。新增教訓入口與guard，沒有新RE或原版RGB聲明。

重跑沿tools/verify_fd2_battle_surface_hd.py的panel-unit／panel-source／panel-game／panel-scroll與既有regression／scroll-regression／legacy-gpu，在有界Go Docker內給正式catalog、pack、原包與out前綴；四語由--panel-locale，分離場景--pair 34:36 --transition與--enemy。原生面板由go test ./internal/battle -run "^(TestRenderNativeBattlePanel|TestNativeBattlePanelRecord)"；字串由fd2-string-inventory、官方migrate_string_review.py及./cmd/fd2-string-inventory測試。固定來源、源碼SHA與全部最終收據見HD主收據battle_panel_text_followup_20261024及hud-final-verification-oct24-r1.json。正式catalog761項與私人美術保持，不需新私人素材提交；原正常／集中人工／平台與浮出工具列／速度排程保持。公開提交、Issue及容器清理於收尾另核。

收尾只刪本輪被陰影修正版取代的55份r1 PNG，共132336154 bytes；逐檔核對原收據SHA後刪除，保留一份陰影缺陷原圖、全部72份最終PNG、收據及log，不當作共享主機淨可用空間。彙整器先誤認分離背景收據也帶PNG SHA；確認其只記檔名與圖層後，72份最終PNG均核完整尺寸並另存首份SHA清單，未冒充全部上游均有逐SHA比對。

新教訓guard首檢拒收，因fd2_lessons.py明確排除_test檔；改綁正式raster的陰影清除矩形，保留實際字形回歸，未修改產品或放寬測試。

最終收尾通過17份來源指紋、72 PNG雜湊、2390個本輪文件本地目標、153個來源／輸出擁有權，324教訓／189 guard及24條遠端worklist。#263／#204已更新且原acceptance保持，#268／#269仍OPEN，可能已完成0條；最終核對hud-final-check-oct24-r3.json。廣域自檢另見51份歷史root-owned原資料／圖片，均未列本輪寫入，受版控者與HEAD逐bytes相同，其他只留SHA；沒有全域chown或刪除。所有本輪Go／Xvfb／Pillow容器均已退出且--rm，私人庫沒有本輪變更，沒有新image、volume、錄影或Release。

提交前程式審查補修拒收分支：bindModernPhysicalScenery的defer原會在來源失敗時也附加typed面板；改為只有成功才綁定，補真實尺寸但缺raw的BG拒收案例。最終panel-unit r6三項、physical-regression r2七項及字串r3八項乾淨重跑通過，109筆分類保持。有效繪製、捲動與shader完全未改，因此保留六場成功GPU收據並保存其原始碼快照，不重新擴跑無關畫面；最終來源與差異界線見hud-final-verification-oct24-r2.json。

拒收分支補修後最終核對hud-final-check-oct24-r4.json通過：17來源、72 PNG、2390本地目標、176本輪來源／輸出擁有權；51歷史root檔保持。提交前host gh重新拉24項，官方snapshot／render／verify及324教訓／189 guard通過；最新origin/main仍為本輪基底aa156fa0。

2026-10-24 共用戰鬥指令面板HD文字接線，限定CONFORMED-SCOPED／RUNTIME-E1。依路由的規格先行入口，沿核准Noto與1920畫布，先將61及主收據列READY；重用已閉合面板writer／origin與指令producer，不開新RE。15個presenter涵蓋0～8、敵方9、24／29與32～35；4／5共用presenter。collector本次owner私有，base建構與巢狀stage helper明示傳遞，失敗不發布。逐次typed數值與raw姓名凍結、原生base逐byte匹配後附加，24／29先附加再捲動／裁切。全色盤演出的文字取當幀palette，人物專用subtract／mix不作用於面板。原框、bar、overflow、交易、RNG、音效與保存保持。

16來源分片1993幀含1486戰鬥幀、147快照，原生圖／單位／RNG對照相同；16完整Game.Draw共1590次文字GPU回讀、598330匹配像素、104次F2及346 PNG通過。39 HD閘門／53必要頂層測試、八字串回歸，最大記憶體峰值3461398528 bytes，無SKIP、逾時或OOM；是受控Linux測試，不推定正常／人工／其他平台效能。24／29全部145畫布另對Pillow容差1，各30拒收通過；物理雙側回歸與四語字形／GPU保持。官方字串遷移只更新inventory SHA與LV標籤來源行號，109分類保持。README新增四語完整半縮圖，逐SHA重生且標籤位於遊戲區外；測試隊伍名稱與替換圖源差異明示。固定清冊761項及私人美術保持，沒有新素材或私人同步需求。

過程失敗如實保留：初次編譯因command6巢狀helper未明示傳collector而拒收，補完整傳遞後全16來源／Game.Draw重跑；24首次擷取被舊exporter誤判typed面板為原生負resource，修正測試序列化及獨立驗證器後同流程通過；新增錯位／尺寸反例先被既有矩形guard拒收，修反例為內部一致矩形，再驗面板來源guard，未放寬產品或色差；初次字串測試在官方遷移前綁舊SHA而拒收，遷移後八項通過。來源、重跑及最終收據依主收據command_panel_text_followup_20261024與command-hud-verification-oct24-r1.json。完整HUD美術、其餘背景caller／效果、浮出工具列／速度及集中正常／人工／平台仍待；完整HD目標保持active。Issue、清理與公開推送另於收尾核對。

最終核對command-hud-final-check-oct24-r1.json通過490項SHA引用、39份來源指紋、346 PNG、2506個本地Markdown目標與本輪1000:1000擁有權。325教訓／190 guard通過；host gh更新#263／#204並拉24條，官方snapshot／render／verify同步，可能已完成0條，原acceptance保持，#268／#269仍OPEN。廣域自檢仍為51份歷史root-owned原資料／圖片，與本輪寫入無交集，另存SHA並保持未動，沒有遞迴chown。全部本輪Go／Xvfb／Pillow容器已退出且--rm；私人庫乾淨，沒有新image、volume、錄影或Release。通過及拒收收據、346最終PNG與兩份小型初版失敗PNG保留；未刪原始參考來源。公開提交與遠端HEAD另核。

2026-10-24 四種召喚戰鬥場景HD接線，限定CONFORMED-SCOPED／RUNTIME-E1。前一批758e299e已由main推送並核對遠端相同，host gh於#263留言提交；完整HD目標保持active。新分支從該main建立，61與主收據先列READY，沿既有原生producer、完整已登錄BG／TAI圖源及HD扣色／插色契約，不新增RE、美術或清冊／保存欄位。32～35在圖庫及owner發布前，先在私有副本拆出來源並以原生圖逐byte重組base，整批成功才發布。前導原生台座層只標記已驗來源，保留stage位移；其餘戰鬥階段沿原位置拆出場景，地圖尾段保持。

八來源分片1620幀含606戰鬥幀及303可見TAI來源幀，八完整Game.Draw／720實際回讀、2630763場景像素含1787台座核心像素、44次F2與44最終PNG通過。18個HD閘門／21必要頂層測試及八字串回歸無SKIP／逾時／OOM，最大測試峰值4548190208 bytes，不宣稱正式遊戲或其他平台效能。背景及台座取樣只涵蓋未被後層遮擋的不透明核心；原生base重組檢查涵蓋全部戰鬥來源。字串官方遷移只改inventory綁定，109處置保持；README新四語完整半縮圖與既有圖組重生相同SHA均驗證。唯一來源、源碼指紋與重跑依主收據compound_scenery_followup_20261024及compound-scenery-verification-oct24-r1.json。清冊761項及私人美術保持，沒有新私人同步需求；完整HD未完成。

失敗與勘誤保留：R1來源是BG0及合法空白TAI0，metadata出現台座不能當作可見HD；R1 side2的Go繪製通過但永久驗證器因缺可見TAI拒收，不改寫成成功。R1 side0在command32 BG0通過後，為改用可見fixture而停止自有批次，外層137屬有意取消，已完成command32收據無OOM；未冒稱產品signal killed。R2直接覆寫NativeTerrainControl被既有editable-field／FDSHAP一致性guard拒收，連續失敗後重新載入規格先行入口，修fixture為選原控制表已有圖塊，保留正式檢查。R2 GPU在嘗試停止前已自行以同錯誤結束，查無容器，沒有刪到其他工作。R3以BG4／TAI4、raw side0／2完整重跑通過，seed與Camp保持既有fixture政策。初版失敗log及收據不刪；沒有因此修改原版語意或放寬色差。普通0～9的LUT caller、地圖尾段／其他效果、完整HUD與浮出工具列／速度及集中正常／人工／平台仍待；Issue及清理於收尾另核。

召喚場景收尾核對compound-scenery-final-check-oct24-r1.json通過112項SHA引用、九份來源指紋、44 PNG、2521本地Markdown目標與本輪1000:1000擁有權。326教訓／191 guard及24遠端worklist通過，可能已完成0條；#263／#204已更新且原acceptance保持，#268／#269仍OPEN。51歷史root檔與前批逐SHA相同，沒有chown或刪除。所有本批Go／Xvfb／Pillow容器已退出且--rm，保留失敗記錄與最終畫面，沒有新image、volume、錄影或Release；私人庫無本輪變更。公開提交與遠端HEAD另核。

2026-10-24 一般指令背景／台座與查表HD接線，限定CONFORMED-SCOPED／RUNTIME-E1。由乾淨f0145bca主線建分支，載入規格先行路由及逆向重製技能，61與主收據先列READY。審查九presenter涵蓋0～8及敵方9，4／5共用，確認actor用11、尾段12～14、6的stage用15；原始raw／查表僅在當幀base全部可見mask逐byte相符後附加，完整重組成功才發布。原frame不覆寫，native fallback精確映射索引；HD沿terrain既有四色差、直通alpha及黑白端點近似，owner私有共享權重與最多兩個內容圖，預備於發布／F2前，不在Draw補讀或算圖。

19合法來源／1970戰鬥幀、19完整Game.Draw／1829實際回讀、1802422場景像素含474686變色像素、129次F2及328 PNG通過。43 HD閘門／58必要頂層測試、十完整原生fallback世界及八字串回歸無SKIP／逾時／OOM，峰值3627274240 bytes為本批容器測試範圍，不外推正式／平台效能。清冊761項及私人美術保持，沒有新素材或私人同步需求。README新四語完整半縮圖逐SHA記錄；字串處置沿官方遷移，109分類保持。唯一來源、源碼指紋及重跑依主收據ordinary_scenery_followup_20261024與ordinary-scenery-verification-oct24-r1.json。

失敗及勘誤保留：R1指令0 Go完整繪製通過，但固定31×23網格沒有抽到TAI4的584個alpha255點，永久驗證器拒收；Pillow確認483可見點而網格命中0，改依實際alpha選點並保容差1，R2同流程通過。來源矩陣的AI9 side2被既有raw-side-zero actor guard拒收，確認0x15311既有owner範圍後，規格改列19合法來源加非法側原子拒收，未解鎖原版沒有的路徑。反側R2遇TAI3未登錄，核對固定原包10×3、raw_size7及全零mask，改分列已證實原生空白；任何可見但缺HD的來源仍拒收，R3反側完整重跑通過。F2快取另補兩內容項與最後繪製者保留，驗多候選不會移除當前內容；不借用不同原圖SHA／查表／palette的圖。兩次apply_patch空白context不符在寫入前停止，依gofmt後實檔修補，沒有產品變更遺失。全部原始失敗log／收據保持。完整HD、其他效果尾段／HUD及集中正常／人工／平台仍待，目標保持active。

2026-10-24 本批收尾：507個收據／源碼／圖片引用、2425個文件本地連結、761清冊、109字串分類及gofmt通過。官方快照由主機gh讀回24個開啟Issue後以官方build_snapshot／write_snapshot產生，render／verify顯示可能已完成0，#204／#263驗收保持，#268／#269未關閉。327教訓／191 guard通過。51歷史root檔逐SHA保持，新檔與修改檔1000:1000，沒有新增異常md目錄。私人庫乾淨，兩種FD2測試image無殘留容器；沒有新錄影、image、volume或美術候選。最終完整性首次在登入shell遺失Go PATH時未找到gofmt，改用同一Go image的直接Python入口後乾淨通過；彙整PNG沿現有Pillow image，未裝主機套件。完整性證據依主收據final_integrity。

2026-10-24 接續a6fe4734，從main建hd/compound-map-tail-20261024；上一輪分類為已推送進度。載入規格先行路由、重製技能與交接reference，主機gh auth通過，官方快照24個開啟Issue。61與HD主收據先列READY，再接32～35全部地圖尾段。原生WithSources、image、delay、聲音、publishAt、500ms hold、RNG及一次續行保持。33七環與既有數字直接HD，其餘未登錄效果仍原生。沒有新美術／清冊或保存欄位。

每階段凍結input／work／VGA／palette，逐byte重組後才發布候選。F2失敗保留當前圖；GPU依凍結scene清理紋理，普通地圖nil owner回收本次尾段內容。34／35偶數mask用初始完整VGA，奇數用該階段work，不提前顯示後段狀態。

{"source_cases": 8, "all_native_frames": 1620, "map_frames": 1014, "source_atomic_rejections": 16, "actual_flows": 8, "audited_map_draws": 3069, "matched_pixels": 49963320, "registered_overlay_pixels": 106179, "changed_overlay_pixels": 106179, "f2_roundtrips": 100, "screens": 264, "hd_gates": 21, "required_tests": 36, "peak_bytes": 6442450944, "required_lmi_effects": 84, "registered_required_lmi_effects": 7, "remaining_visible_lmi_effects": 77}。HD閘門無SKIP／逾時／OOM，字串八項保持；原生三套件日誌另記通過與略過，不混稱HD必要測試。記憶體峰值只代表本批Linux容器，不外推平台效能。來源、源碼指紋與重跑依主收據compound_map_tail_followup_20261024。

固定原包110項LMI逐raw／Pmode／manifest驗證，84光效與26數字相關來源均非空。尚缺64～75、126～190共77份光效，下一步優先補美術。README新增四語完整半縮圖，取R1真實流程；32～34主側另R2補強HD圖層核心，較弱收據保持。清冊761與私人庫未改，不需私人同步。

修正記錄：未用import在編譯時拒收，移除後同工具鏈通過。相對out被remake cwd的驗證器拒收，改絕對out乾淨重跑。完整GPU格點通過後，再補已登錄alpha255核心及原生像素差。盤點工具固定四列與stdout27／13讓110項參考圖截斷；原數據仍有效，改動態列數／實際筆數後R2重生，預設玩家9入口另通過。apply_patch空白context與腳本新增hunk格式錯誤均於寫入前拒收，修正後再執行，無產品狀態遺失。

其餘owner／HUD、來源408、18／19第三銀行合法caller及集中正常／人工／平台仍待；#268／#269依原排程。完整HD目標保持active。

2026-10-24 記憶體限制：既有完整回復GPU回歸在6GiB限制碰到memory.max並回收389次，419.444秒完成，oom／oom_kill均0；未因觀測時間延遲重啟。這是測試工作負載的壓力證據，不稱平台效能或可用餘裕已驗，入口見主收據memory_context。原生三套件365個必要頂層pass、0 skip；八字串維持109分類。

2026-10-24 最終完整性：373份收據／源碼／圖片引用、2437本地連結、761清冊、109字串分類、24個遠端開啟Issue與gofmt通過。#204／#263驗收保持，#268／#269未關閉；工作清單可能已完成0。328教訓／192 guard通過。51歷史root檔逐SHA保持，新檔與修改檔1000:1000，無新異常md目錄。私人庫乾淨；本批兩種FD2 image無殘留容器，無新增image／volume或錄影。README四語圖已逐SHA回查來源並目視，人物與部分地形／數字HD，其餘地形與HUD原生如圖說。完整性入口依主收據final_integrity。

## 2026-10-24 四召喚77份尾段光效HD

路由沿spec-gated入口及README標準，61／主收據先READY後實作。內建imagegen以固定原版形狀及核准回復環風格生成九格母稿；R1因邊界裁切／相鄰光暈拒收，R2原RGBA與alpha保存。首份編製出現冰晶／彎刃／交叉斬痕平色缺口，先補READY材質契約，再以固定八方向貼合、最近墨跡RGB延拓與鏡像第二道斬痕修正。原生支撐、1.1HDpx羽化、2HDpx邊緣、亮度及棕區調色均明示近似，不稱原版RGB／alpha相同。77份從舊761清冊可重生且逐SHA相同。

{"new_lmi_frames": 77, "catalog_assets": 763, "lmi_groups": 7, "lmi_frames": 146, "required_compound_lmi_registered": 84, "required_compound_lmi_native_fallback": 0, "hd_gates": 16, "required_top_level_tests": 29, "source_gpu_worlds": 462, "cold_group_rejections_per_source_run": 18, "actual_independent_flows": 8, "formal_path_flows": 1, "legacy_theme_flows": 1, "audited_map_draws": 3051, "map_grid_pixels": 49670280, "registered_overlay_pixels": 153856, "changed_overlay_pixels": 153840, "f2_roundtrips": 100, "actual_pngs": 264, "peak_bytes": 3426471936, "producer_tests": {"pass": 365}, "string_tests": {"pass": 8}}。全部HD閘門無SKIP／逾時／OOM；三套件365與字串8頂層pass。八雙側四語流程保留結算／RNG／一次續行且無可見LMI回退，正式路徑另驗32、舊761另驗34原圖回退。正式開發包既有symlink不符合獨立實檔旗標，首次啟動於檢查階段拒收；依正式契約重跑，獨立5712實檔包另列，不放寬檢查器。README圖R1冰晶128貼邊不清，R2改取同真實流程132；整幀半縮、邊界裁切如實保留。

正式763清冊新增77PNG與原母稿，原761項／6609檔保持。私人6687檔逐SHA與正式相同且UID/GID1000，私人提交7015f54d6f8467ae4daa36e2a3104aa5846ca26e已推送並核對local／origin／remote HEAD。GitHub connector對私人庫404，僅用認證Git讀取與未認證不能讀取核對既有私有邊界，沒有取得API visibility欄位或更動公開設定。

收據、完整提示詞、源碼指紋與重跑入口依主收據compound_tail_art_followup_20261024。其他owner／HUD、408、18／19合法caller、集中正常／人工／平台及#268／#269保持待辦，全部HD目標仍active。

2026-10-24 空間清理：獨立實檔包完成全部驗證且正式／私人包逐SHA相同後，核對5712實檔雜湊，保留catalog／manifest與全部測試收據，再刪除可重生的單一測試副本，共1712476731 bytes。未產生錄影；母稿、必要PNG、原版來源及正式圖包保留。重生入口與逐檔清理清冊見主收據runtime_copy_cleanup。

2026-10-24 最終完整性：{"status": "PASS", "sha_references": 80, "local_links": 2453, "historical_root_files_unchanged": 51, "new_files_uid_gid": 1000, "output_root_root_owned_or_md_directories": [], "readme_cultural_entries_present": true, "gallery_sources_verified": 4, "python_sources_compile": true, "issues": {"open": 24, "manual": 16, "possibly_done": 0, "parents_204_263_and_art_268_269_open": true}, "lessons": 328, "guards": 192}。私人庫乾淨且兩種FD2工具image無殘留容器；無新image／volume。Issue讀回只規範末尾換行差異，原驗收／id／verify.kind保持，官方工具產生快照與render，未手改產生區。來源與重跑依主收據final_integrity。

2026-10-24 正式覆蓋重生：audit_fd2_battle_hd_coverage.py對固定原包、正式763清冊與實際PNG通過，LMI七組146張。264個已解碼FIGANI／2118幀保持，來源408仍阻擋；資料覆蓋不提升caller或玩家證據。完整參數與SHA見本批formal_coverage。gofmt、git diff --check通過。

## 2026-10-24 狀態法術HD地圖接線

從652e36c2的main建立hd/modifier-map-20261024，沿規格先行路由與重製技能恢復。61及主收據先列READY，再將玩家獨立調色owner與玩家／敵方狀態尾段接入共享HD地圖。原生WithSources只補定位資料，原始像素、聲音、delay、publishAt、500ms hold、交易、取消、RNG及保存格式保持。玩家23只涵蓋調色前導，後續消失／出現仍待；沒有新增敵方25入口。

固定原包127個LMI來源逐raw／Pmode／manifest核對。5:74～76已屬既有死亡HD組的火焰，不能誤稱MISS字母或待補美術；失敗queue仍保留原始十進位索引。正式763清冊保持，只有6:191～229共39份尚未登錄。以內建imagegen生成兩份透明菱形母稿，原RGBA與完整提示詞保存於既有work根。39幀可丟棄編製候選仍為DRAFT，部分次要實心菱形有條紋狀材質接縫，未登錄或同步私人庫。目視時一度懷疑192右側褐色直線，固定原圖也有相同筆畫，因此撤回該疑慮並保留原位置。候選、原圖與勘誤見主收據next_art_candidates。

初輪獨立GPU流程在map-palette出現像素差，曾懷疑多Game批次；改獨立程序仍重現。原生控制再次compose也差1018點，查到composeNativeMapFrameForCaller會取樣真實時鐘並推進地形／待機／色盤。測試改為初始化前固定兩側Unix1000，額外compose與獨立控制相同，推翻先前批次假說；正式遊戲時鐘不變。來源及GPU矩陣固定seed7，另沿既有AI26正例預先指定seed0補驗成功，不重擲或取代seed7的失敗結果。錯誤及修正收據分開保存。

其他驗證環境修正：AI直接建構缺少正式派發脈絡，改走executeNativeAIActionWithContinuation；原始己方紀錄缺AI raw欄位，改用原敵方4／5紀錄並明示位置、Camp、class與狀態，目的地同步建構位置。這些只改測試條件。六案例GPU分片在750秒到期，峰值4228853760 bytes，oom／oom_kill均0；保留失敗收據，拆成三案例且不降低檢查條件。先前未固定時鐘的通過收據不算本批最終矩陣。

Docker恢復時首次受sandbox拒絕連線，改用已授權的隔離Docker控制面；既有素材image的entrypoint是Python，補明示entrypoint後讀取成功，沒有另建image。教訓守衛預設排除_test檔，首次掛測試檔時被拒收，改掛既有永久驗證器後329教訓／193守衛通過。完整HD目標仍active，其他owner、39份光效、完整HUD、未知來源與集中正常／人工／平台仍待。

本批最終矩陣：21來源／846原生幀、42原子拒收、21固定seed7流程及一固定seed0套用成功流程；1333次Game.Draw回讀、21701240格點像素、210193已登錄HD圖層核心含210178異於原生像素、84次F2往返、278張完整1920×1200 PNG通過。12份HD閘門／21項必要頂層測試，另365原生producer及8字串回歸通過，最終必要矩陣沒有SKIP／逾時／OOM。117次原生圖層回退是未登錄39份來源在本矩陣的繪製次數，不是117份缺件。Linux容器最高4098408448 bytes；同容器順序測試的memory.peak為累計值，不外推平台效能。失敗六案例分片另列，不混入通過矩陣。

字串回歸首次只有審核綁定SHA失敗。使用已保存且匹配舊審核SHA的盤點，經官方migrate_string_review.py遷移座標；逐分類比較role／text／file／function與數量保持，109筆仍為34玩家可見／62內部診斷／9開發／4未知，沒有drop或新分類。八項字串測試R2全部通過。原始失敗log與遷移收據保留，未放寬測試。README新增未修圖的完整敵方17 Game.Draw與明示fixture圖說；正式清冊與私人素材庫保持，沒有新Release、錄影、image或volume。

字串差異進一步定位：完整5226筆的語意簽名及數量相同，真正移位的是modern_hd_compound_map與native_command_modifier_presentation六筆runtime_ui座標。109筆go_review的ID本身未變，review只更新其綁定的完整inventory SHA，分類與處置內容保持。Issue #204／#263已讀回確認新進度，原驗收／id／verify區塊保持；#268／#269內文未動且仍開啟。官方快照與render／verify更新後，24個開啟項、可能已完成0。完整性核對涵蓋本批收據、來源、圖片、文件連結與擁有權，51個歷史root檔逐SHA保持；既有README文化入口保持。

最終完整性通過350項SHA引用、2574個選定文件本地連結、329教訓／193守衛及Go格式。新增與修改檔、本批modifier輸出均1000:1000，沒有異常md目錄；私人庫乾淨，兩種FD2工具image均無執行中或停止殘留容器。沒有新增image／volume或錄影，不需刪除正式或原始素材。唯一重跑與收據入口為HD主收據modifier_map_hd_followup_20261024；其他HD待辦保持，目標active。


## 2026-10-24 使用者暫停HD並清理空間

HD目標維持paused，只執行空間清理。主機衛生依AGENTS與文件職責路由，沒有額外Docker專用知識入口。保留hd/modifier-tail-art-20261024的未提交程式、764清冊、原稿、主候選、存檔、原版來源與dist-all。私人素材提交a8516596617f2255410532fbdb647a9fb1793a92已推送且工作樹乾淨；公開HEAD仍為de45ee02。39份來源、七條消費端、正式／私人實檔及舊763回退已有通過收據，尚未整理公開提交與現況文件。恢復HD時沿modifier_tail_art_followup_20261024接續收尾，不重做這批素材。

以go clean -cache刪除5619474432 bytes已確認建置快取，保留本人擁有的快取目錄與README；正式素材依賴與快取沒有交集。兩份驗證過的重生副本移除13374個重複符號連結及5962020 bytes重複檔案，保留各自catalog／compiled收據、主候選與正式／私人包；沒有刪除符號連結目標。

歷史章節PNG只在完整SHA256、尺寸、模式、UID/GID與穩定inode／mtime一致後，以原子硬連結合併副本，路徑及內容保持，排除正式來源與受版控檔案。Docker控制面unexpected EOF使去重容器以125中止，未盲目重啟。保留原始壓縮日誌，另復原有效gzip；逐SHA驗證168813個完成連結，回收2633994240 bytes資料區塊，沒有殘留暫存連結。不宣稱全部歷史擷取均已去重。歷史收據PNG保持不可變，重跑輸出使用新目錄。

舊fd2-promo:20261017-r1與fd2-build-appimage:latest已移除，重建入口分別為tools/docker/fd2-promo.Dockerfile與remake/packaging/Dockerfile.appimage。Go、素材、Capstone及Watcom現行映像保留。FD2無殘留容器或具名volume；依FD2儲存庫／Dockerfile建置紀錄與時間窗核對，沒有可單獨歸屬的未共享舊快取，未全域prune。舊建置紀錄刪除呼叫後曾斷線，讀回仍有3筆，沒有冒稱刪除完成。其他專案資源保留。

清理後6728份正式素材逐SHA與私人清冊相同，未提交HD來源仍在，原版資料、保存與正式交付未動。本批可核對回收下限8253468672 bytes，另有兩個舊映像與重生副本回收；主機可用空間更大幅度的變動不歸因於本批。清理總收據與來源快照見[workspace-cleanup-report-20261024-r1.json](work/workspace-cleanup-report-20261024-r1.json)。HD仍暫停，沒有追加實作或發行。

## 2026-10-24 恢復HD並完成39份狀態光效

使用者明確要求開工繼續，恢復原HD工作。沿用已確認手繪風格與規格61的READY狀態光效契約，完成FDOTHER6第191～229幀的圓環、黃綠及淡藍菱形三組光效。未修改母稿；可重跑編製器以原生連通元件及輪廓限制材質，羽化與亮度為明示美術近似。R1矩形疊接、R2極低alpha邊緣缺RGB，R3只延伸340個最大alpha為1的邊緣RGB，聯集alpha保持。原生192幀旁的褐色直條確屬來源，保留；不再誤判為HD瑕疵。R3舊prototype.json狀態字樣沿用R2，主收據以label_note勘誤並綁定39份成品SHA。

正式清冊由763增至764項，LMI八組185張；只開放resource6／count39對應191～229。既有763項及6687個檔案SHA保持，新增39份PNG與兩份未修改母稿。私人白名單共6728檔，提交a8516596617f2255410532fbdb647a9fb1793a92已推送並核對遠端HEAD。主候選、正式與私人包保持；清理只移除已驗證的兩份重生圖檔副本及別名，catalog／compiled收據保留。

17份HD閘門、26項必要頂層測試、365項原生及8項字串回歸全數通過，沒有SKIP、逾時或OOM。三輪各39來源／78完整GPU世界及九個冷啟動拒收通過，舊77來源／154世界與18拒收保持。21個既有owner來源共846原生幀及42拒收通過。七條受影響流程加正式與私人抽樣，共581次Game.Draw回讀、51110個光效核心像素、41次F2與149張1920×1200 PNG，無可見LMI原圖回退；舊763主題另保留16張PNG，13個原圖圖層回退符合契約。跨包78個來源世界及13張效果PNG的SHA完全相同。

兩側均在初始化前固定種子7及時鐘1000，使用明示建構的原始敵方紀錄、位置、職業與狀態。規則、交易、亂數、存檔及原生節拍保持；這批只宣稱CONFORMED-SCOPED／RUNTIME-E1，不新增正常玩家或原版RGB對拍。記憶體最高3138428928 bytes為序列容器累積峰值，只代表此次Linux工具鏈，不代表個別測試峰值或跨平台驗收。清理後以保留的收據與來源指紋完成核對，沒有為文件修改重跑已通過的GPU流程。

README新增三條實際完整Game.Draw的半縮圖，保留受控狀態與證據限制；61、58、56、57及HD主收據同步。#204／#263保持開啟；#268／#269未關閉。玩家23的消失／出現、其他光效、完整HUD／地形、未知來源、正常玩家流程與集中人工／平台仍待。沒有新Release、錄影或Docker映像。唯一收據入口為[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的modifier_tail_art_followup_20261024。

最終核對通過260項SHA引用、165張完整擷取、正式與私人各6728檔逐SHA及擁有權、2894個本地連結與README文化入口。Go格式及Python語法通過，330條教訓／194守衛與764項正式清冊有效；工作清單24項仍開啟、16項待人工、可能已完成0。#204／#263進度讀回相符，#268／#269原欄位與內文保持；兩次查詢多出url欄位的驗證腳本差異已修正，不列為產品缺陷。本批沒有遺留FD2容器。

## 2026-10-24 操作介面與人物主題分離原型

使用者要求操作介面也HD，工具列分開選操作介面原版／HD與主題原版／HD，並明示介面中的人物icon跟主題切換；頭像與戰鬥動畫同屬主題。沿既有A頂端浮出、自動隱藏決定製作四組合原型，沒有接入正式Game或更動保存／素材清冊。新分支hd/ui-theme-controls-20261024以b5953ff4主線建立。

重用原型建構器與Playwright1.56.1容器，加入--display-modes，同一索爾DATO_000及正式480頭像示範人物選擇；原版面板沿保存的重製端索引圖，HD面板為SVG樣式，保留LV／EX／DX／MV／HIT／EV／AP／DP／HP／MP及兩項物品資訊。藍底與主要配置的視覺方向尚待使用者確認；新F11介面快捷鍵僅在原型有效。四語只驗工具列，面板固定繁中，不宣稱戰鬥動畫已接入此原型。

第一輪日文窄版主題字寬溢出後縮短標籤；第二輪全頁擷取使浮出列隱藏，測試繼續點擊隱藏控制而停滯。只讀CDP事件確認原因，停止自有容器，修正檢查器先用頂端熱區喚回工具列，字級重試最多三次；這是原型驗證腳本問題，不列為遊戲缺陷。修後12種語系／字級、六種倍率選項、四種介面／主題組合、人物圖示只依主題、F2／F11互不改寫、F4、顯隱與固定版邊界均通過；舊模式另驗12版面。倍率仍只改示意計時條，不是遊戲加速。最後補齊HIT／EV並調整字寬，收據為ui-theme-split-browser-oct24-r5；前輪輸出保留，不重寫歷史。

使用者表示找本機檔案不便，改為直接在對話附概念圖，沒有要求使用者尋找路徑。樣式確認問題仍待答，不把沉默當同意。61與58登記DRAFT，唯一來源與收據入口是[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的ui_theme_split_20261024。完整HD、工具列正式串接與正常玩家／人工／平台仍未完成。

既有HD工作也已定位指令23共享nativeUnitPresentJob：11個FDOTHER6前導、六個查表收縮、條帶bridge與十個展開，相同mutationAt17及回滾必須保留；消費端也供戰役staging，不另造指令23專用分身。這輪沒有重開已閉合0x22253或實作其HD。概念圖產生於既有工作根，沒有新image、volume或錄影；原始與私人素材保持。

## 2026-10-24 單列浮出、雙組海報與HD2風格定向

使用者重申A浮出，要求工具列盡量單列、游標下敵人的地圖與資訊窗圖示一致HD、地圖素材HD及兩側海報。內建image_gen先產生含HD海岸與斜角站姿人物的整體概念，使用者明確選其中間戰場作為獨立HD2，不覆蓋現有HD。原基準完整保存，SHA a18f75cf8a72a11367fb26a11098a2dc11b282a1063acd9991980d6c87c36f5b；原先索爾／凱麗海報配置由後續指示取代。

第一組改為左索爾＋悠妮兩人向右、右空魔神＋四天王；以原版DATO122～126及正式名稱表核對地水風火空五位，HTML原圖參考板不修改來源。第二組採使用者提供三張原稿重繪，第三張為紅色封面的局部，原圖逐SHA另存。曾上網搜尋官方新聞、收藏海報與角色資料，最終圖源依使用者指定，不用搜尋誤配的其他Dragon Knight作品。內建圖片工具最多五份參考，將五個原生頭像排成單張未改圖參考板後生成，未改用CLI。兩張1672×941整體概念已直接顯示並存入本專案，提示詞及輸出SHA完整保存；海報重繪不稱原始掃描。第一組五位角色均可見，左二人朝右；中間保持HD2方向與同角色敵人HD資訊窗，工具列均為單列。

本機新入口work/hd2-posters-preview.html可切兩組及下載，圖片來源在既有ordinary-effects-r1工作根。正式Game、清冊與私人正式素材庫未改，沒有生成執行檔或錄影。此批只屬視覺概念；尚未驗證HD2重用圖源、六組合GUI、正式單列浮出或海報功能。唯一目前入口為HD主收據ui_poster_hd2_followup_20261024及規格61，58登記同一狀態。既有三列HTML只保留較早控制驗證，不能外推新單列功能。

HUD差異由目前來源定位：modern_hd_map_runtime.go獨立畫HD世界人物，UI仍以BlitNativeMapHUD的原生HUDUnits合成；main.go的optional unit沿SelectorSlot、raw idle及顯示條件。後續補主題圖示接線，不固定塞某個敵人頭像。右下放大資訊窗屬新介面提案，原座標、typed資料及玩法規則仍要驗證。沒有重開原版RE或宣稱新原版parity。

本批收尾核對25項來源／輸出SHA、1057個本地連結、原始參考圖擁有權、預覽頁JavaScript語法及兩張圖路徑均通過。#204新範圍讀回一致，官方快照與工作清單已更新，24個開啟項仍未關閉。沒有殘留本批瀏覽器容器；工作樹保留可審閱原型及規格變更，未以純概念圖批次提交或推送正式功能。

## 2026-10-24 海報漸層原型與GitHub設計基準

使用者接受雙組方向，要求海報淡入與戰場漸層銜接可由工具列設定，並明確要求完成後將概念圖保存GitHub，後續HD2戰場依此設計。此授權包含本批概念封存提交；不把純概念發布冒稱一般玩家垂直切片或正式遊戲功能。沿已選HD2中間戰場，保留三張未修改概念原稿及兩張最終漸層PNG在docs/figures，原始使用者參考圖與素材包未加入公開庫。

先將瀏覽器切片寫成READY，再實作單列浮出工具列的海報組別、0～96px空間漸層與0／300／600／1200ms時間淡入，預設48px／600ms。漸層合成只改兩側海報，調寬度不重新淡入；全局關閉、重播、熱區喚回及窄窗橫向捲動保持單列。速度／介面／主題按鈕仍是概念位置，不宣稱遊戲加速或正式主題載入完成。

第一輪Canvas像素讀回受file不透明來源限制，驗證器改以容器loopback白名單提供HTML及兩張PNG，維持network none，沒有關閉瀏覽器安全機制。第二輪控制及核心像素通過後，視覺審閱發現HUD倒影；兩張圖移到本機拒收證據，不進公開基準。改只延伸600列前乾淨地形，並新增每組22560個邊緣像素核對；第三輪兩組0／48／96px、1200ms早期／完成及預設600ms、關閉、1800／720單列與顯隱全過，每次中央核心3218400 bytes逐像素保持。未修改原圖SHA保持，正式Go與素材清冊未變。

README保留文化保存入口，新增HD2設計基準及兩組概念，61限定瀏覽器CONFORMED／正式DRAFT，56／57／58同步範圍。公開提示詞與圖源／輸出SHA在docs/data/fd2-hd2-concept-design-20261024.json，瀏覽器收據另存同目錄。正式HD2圖源、第三主題載入、六組合GUI、HUD接線、持久化設定、正常玩家／人工／平台仍待，沒有Release或錄影。

發布前核對43項SHA引用、2607個本地連結、331條教訓／195守衛及新檔擁有權通過。教訓掃描器既有副檔名不含mjs，將新守衛綁到已驗證乾淨地形收據訊號，未擴大掃描工具範圍。#204新驗收條件已讀回確認，24項工作清單保持開啟；原版及私人素材庫乾淨，本批沒有殘留瀏覽器容器、錄影或新Docker映像。

## 2026-10-24 地圖HUD人物圖示HD串接

從ecabe2e2重新建立hd/hud-unit-icons-20261024，沿58與61的現況，選定世界人物HD但資訊窗仍為原圖的正式缺接線。先將原版HUD選圖、24像素框與HP層序寫成READY，重用已閉合證據，未重開IDA。固定FDOTHER5圖框130的圖示區遮罩全部不透明；原selector slot回查實際圖組，HUD狀態3轉用第1幀，保持方向0與無palette-band。

正式實作以原生框／地形／AP／DP為底、144像素人物居中、原HP在上，只清除原UI的24×24人物區。事件視窗同步移除四個原生像素的邊距，明示清空UI的取樣幀同時清空HUD描述；指令環後恢復不透明HUD區塊，避免重複alpha。圖源缺件、尺寸、slot／state與非不透明底層在發布前拒收。無新美術、清冊、規則或保存欄位，私人素材庫保持乾淨。

最初單元拒收樣本誤把原生色碼0當成透明；直接檢查four-mode與分離Mask契約後，只修測試為有效的透明Mask，沒有改正式規則。r2通過後補驗取樣幀清空HUD，最終hud-unit-r3的四項必要測試通過。正式圖庫125組、500個HUD狀態，以及四個Game.Draw／F2、兩個層序GPU樣本共六張完整PNG通過，像素容差2保持；README取右下強盜樣本縮成960×600，原圖與SHA保存在主收據。22項原生HUD、8項字串回歸，以及玩家9完整命中／未命中、狀態17與七項交易／凍結／回滾回歸通過。

治療八流程的單程序驗證主動中止，容器退出137由本輪docker stop造成，不是引擎被OOM殺掉；未把部分擷取計入通過。本輪原預算840秒短於既有917秒成功收據，改由受版控驗證器明示start／limit分成四批，每批兩條，兩個有界容器順序執行；全部八個身份、每條11次F2與一次交易／續行須由合併入口再核對，不降低原驗收。既有全八條預設保持，批次收據改由實際計數產生。

本輪成果只到受控RUNTIME-E1，正常玩家與全HD人工／平台仍待。HD2圖源、六組介面／主題、單列浮出工具列、速度與海報正式串接沒有關閉。蓋亞手腕與凱麗右腳議題保持。

收尾：治療四批及嚴格合併通過全八條／88次F2，每條一次交易／續行；缺項與重複的合併負例均拒收。九份HD閘門共18次必要頂層測試，另22原生HUD、8字串回歸，共48次執行／45個唯一測試，無SKIP／逾時／OOM。來源與40份收據SHA綁入既有HD主收據；初次GPU來源另存，後續只調整測試與分批工具，正式繪製來源SHA保持。61改為限定CONFORMED，56／57／58及README同步；未宣稱完整HD。

主機gh讀回#204內文及正式worklist註記一致，官方build_snapshot／write_snapshot更新快照，render／verify保持24個開啟項、零可能已完成。私人素材庫main乾淨，本輪沒有新素材可同步。24張中止驗證的部分PNG逐檔核SHA後刪除，釋放29992977 bytes，清理manifest及log保留；沒有錄影或新Docker映像。全部本輪FD2容器已由--rm移除，其他專案容器未動。README文化保存入口與新增圖入口保持，331條教訓／195守衛通過。提交以本節所屬Git提交為準，完成檢查後推送origin/main並核對遠端HEAD。

提交前核對58項來源／收據SHA、2510個本地連結及縮圖與原圖的完整像素均通過。新來源、資料、圖片與工作清單擁有權為1000:1000；工作根的root-owned及誤建.md目錄掃描均為零。

## 2026-10-24 單位消失／出現與玩家23完整HD轉場

從已合併5b2f5d34建立hd/unit-present-20261024，沿既有0x22253與玩家23兩次呼叫的已閉合證據，先寫工程READY規格，再接共享消失／出現工作。保留11前導、6收縮、18／24逐列橋接、10展開與17號座標寫入；不新增原版RE、規則或保存格式。visual定位與可見游標中心分開凍結，原生全部work／VGA及等待逐幀重組後才發布HD。

既有回復是在完整work套徑向pass，單位轉場則以work+0x8088為基底。抽出共用繪製時明示WorkOffset，保留兩者語意；獨立幾何核對兩種基底全部312×192點。條帶依456→320位元組位址投影，保留跨列與窗外原生像素；獨立GPU控制組從完整after世界取像素，不使用正式條帶拼貼器驗證自己。F2預檢涵蓋活動工作、調色前導持有的消失／出現兩段，以及故事／戰場focus待啟動工作，全部通過才換圖。

內建imagegen依原版11張輪廓與已核准手繪光效風格產生透明母稿，原RGBA及SHA完整保存。第一版放大遮罩有階梯，留本機拒收；第二版改四倍超取樣連通筆畫與圓接，保持原支撐、孔洞及黃／紅／暗紅節拍，檢視11張總覽與實際Game.Draw後才升美術READY。中心線、兩HD像素外緣、材質、高光及HD查表屬明示美術近似，不稱原版RGB一致。編製來源初次比較只因tuple／list序列化型別不同拒收，確認JSON內容相同後統一JSON表示；沒有放寬固定來源SHA。

候選、正式及私人各驗11來源的22個GPU世界，完整玩家23與戰場staging每次299次繪製、30次F2，固定時鐘1000與原生seed7。MP／Acted、一次續行、原單位／RNG／work／VGA控制組逐步保持；兩包16張完整PNG及完整收據相同。兩種45／51幀、原子缺件／錯尺寸／metadata拒收、晚失敗回滾、待啟動focus及舊主題缺整組光形回退通過。故事呼叫者只完成共享接線與原生回歸，沒有新增故事GPU或正常玩家聲明。

共用回復13單／多目標兩條Game.Draw與22次F2通過，其餘六條既有回復收據未重跑。首輪漏設FD2_ASSET_PACK，未找到FDOTHER65 metadata；補環境後同工具鏈乾淨r2通過，屬驗證環境失敗。九項共用交易／凍結／HUD／回滾回歸，以及八項字串測試通過；109筆人工處置只遷移座標。正式計入十份HD閘門32次必要測試執行、24個唯一測試，另八字串測試，無SKIP／逾時／OOM。候選GPU峰值約5.31GB、正式3.71GB、私人3.02GB，共用回復5.60GB，仍不能代替平台效能驗收。

正式清冊765項、LMI九組196張。11成品及未修改母稿共12新檔依白名單同步私人庫，原764項／6728檔逐SHA保持；私人現6740檔，5762個runtime檔為獨立實檔。私人庫維持private，提交e9f5403961415e4a785f9e1b31d3dffa33620ce7已推送並核對遠端HEAD。由私人母稿及內嵌固定來源報告重編，11份成品記錄相同，不依賴候選稿。

61限定CONFORMED-SCOPED／RUNTIME-E1，56／57／58及README同步，四格圖沿完整最終RGBA雜湊後半縮，沒有裁切或重畫遊戲內容。新來源與收據61項SHA已核對；舊清冊雜湊另綁不可變快照，避免誤指向擴充後正式清冊。#204／#263已讀回內文與原驗收條件，其他22個開啟議題保持；官方快照與渲染更新後仍24項開啟、16項待人工、零可能已完成。來源408、18／19 caller、其他效果、完整HUD、HD2／六組合、浮出工具列／速度／海報及正常／集中人工／平台仍待，#268／#269不關閉。

文件檢查以本輪修改範圍為準，2525個本地連結與README文化入口通過。全歷史掃描另發現九筆舊交付路徑／反組譯符號的缺項，未據此重開歷史工單。Pillow映像不含git，檢查器改用主機git輸出的受版控檔案清單，未將分析移到主機。Go格式與Python語法核對，沒有新增Docker映像或錄影；收據與必要PNG保留。

新增工作紀錄後，本輪文件共2631個本地連結通過；新增徑向基底規則後為332條教訓／196守衛，全部通過。全儲存庫擁有權掃描超出120秒，外層timeout回傳124但Docker容器仍在，已明確停止並移除，未列為引擎或Go測試逾時。改核對本輪unit-present工作根、正式來源及輸出，避免遍歷既有百萬檔暫存樹；其他專案容器保持。

## 2026-10-24 故事入場及返回HD世界

上一輪為具體進度：71cebf6c已推送並核對主線。接續從該main建立hd/story-transitions-20261024，依目前58／61與遠端#204／#263恢復；未重新開啟已閉合RE。核對目前Draw發現故事的0x22253效果雖已HD，穩定世界的admission只接受battle.State，返回時會退回正規化像素世界。另定位0x24618、0x24B4D、天空之鑰與第23章特殊背景仍整幀原圖，列為具體後續，沒有以此清單宣稱其他caller都已完成。

先將故事世界切片寫成READY。只接受已有完整storyNativeMapState、storyBG與六欄視圖的場景，沿既有故事compositor建立私有state及selector cache，僅發布HD畫面。原生clock、DAC、work／VGA與故事角色保持，既有race／class近似只在私有複本；不擴大ch12原生緩衝交接契約。F2換圖前同次預檢故事穩定世界與活動／待啟動轉場，失敗時原主題、圖框及故事來源保持。原生故事證據在docs/data/ida/fd2_terrain_mode3_review_20261001.json的followups[0].story_frame_continuity；程式註解省略ida目錄，文件與SHA入口已修正並查驗，沒有把根層不存在的同名檔當證據。

單元r1的快照誤序列化含函式callback的battle.State，改記實際渲染欄位、角色、selector cache及時鐘／緩衝。r2誤把空Viewport視為錯誤，且破壞的地形來自未被消費的正規化field；依目前renderer的合法預設窗與實際故事carrier修正測試。r3三項通過後補上整個applyTheme的缺圖原子拒收，最終四項通過。上述為測試來源假設修正，正式規則、色差容差2與拒收條件未放寬。

既有GPU執行器新增獨立story-game程序，實際呼叫故事staging及LastRuntimeSlot直接出現，沿正式Draw、原始等待與drawn確認，不略過節拍。正式與私人各198次繪製、兩個返回GPU世界、19次F2及22來源世界通過，12張完整PNG與全部收據相同；固定時鐘1000、seed7，17號座標寫入與一次續行保持。故事私有Draw前後render來源、原生緩衝／時鐘不變。舊玩家23與戰場staging另重跑299次繪製、30次F2與22來源世界，原生控制組及一次返回保持。正式GPU395.241秒／4.20GB峰值、私人220.285秒／3.45GB，舊流程351.393秒／3.76GB，皆無SKIP／逾時／OOM；不是跨平台效能或正常玩家驗收。

最終七份HD閘門共28次必要頂層測試、27個唯一測試；另八字串回歸通過，109筆人工處置保持。新來源、工具與全部收據綁入既有HD主收據story_map_hd_followup_20261024。清冊765項與私人6740檔保持，沒有新素材可同步，私人庫main乾淨。README四格為實際入場及返回的完整半縮圖，逐張先對最終RGBA雜湊，不重畫遊戲內容；61改限定CONFORMED，56／57／58同步新範圍並保留前批收據。

#204／#263最新內文與verify註記已讀回，原驗收條件保持，其他22個開啟議題未變；官方快照與渲染維持24項開啟、16項待人工、零可能已完成。其他故事、0x24618／0x24B4D／天空之鑰／第23章特殊效果、全部caller、完整HUD、HD2／六組合、浮出工具列／速度／海報與集中正常／人工／平台仍待，#268／#269不關閉。下一步先補共用0x24618徑向及完整調色尾段，完整戰鬥HD目標保持。

提交前核對35項SHA引用及六份Go來源快照、七份必要閘門的實際PASS／階段身份、2648個本輪文件連結與README文化入口通過。99個本輪輸入／輸出及來源項目均為1000:1000，沒有root-owned或誤建.md目錄；Go格式、Python語法、333條教訓／197守衛通過。所有本輪FD2 Go及Pillow容器均已移除，沒有新映像、錄影或未整理執行檔。

提交前另追到stepStoryWalks：插值期間先寫正規化終點及OffX／OffY，完成後才SetMapPlacement同步原生座標。為避免中間拍誤畫成原生靜止站位，本批明示限沒有storyWalkJob且鏡頭與raw view對齊的故事畫面；走位與未對齊移鏡保持既有呈現，完整HD仍待。先補規格邊界，再加入走位／鏡頭反例；初次測試用了值清單而正式型別為指標清單，改成正式型別後最終四項通過。正式／私人均用最終Go來源重跑，r2分別277.191秒／5.31GB與274.962秒／3.40GB，全部通過無SKIP／逾時／OOM；12張跨包及前後版本穩定PNG完全相同。最終收據重生README四格SHA仍為1ea0475d21f57697050d5ab9660eff8bfe880d67e973a842043bb0ac61cb8635，不覆寫原收據或圖片來源。

來源快照及字串盤點更新為r2，109處置與八字串回歸保持。前版完整玩家／戰場回歸保留在不進入故事條件的未變分支，不把它冒稱最終故事邊界重跑。#204／#263已再次補上最終顯示邊界並讀回，官方快照與原驗收條件保持。接續先補故事插值走位／移鏡的HD位置、選圖與裁切，再回共用0x24618；main.go的現行位置／幀選擇已記入主收據作下一切片輸入，未改原controller或提前宣稱故事移動HD完成。

最終r2收尾核對38項SHA引用、六份Go來源、2648個本地連結及150個本輪來源／輸出項目的1000:1000擁有權通過。333條教訓／197守衛、Go格式與Python語法有效。原版與私人素材保持，沒有新增錄影、映像或遺留FD2容器；本節隨已驗證垂直切片提交、快轉合併main並推送後核對遠端HEAD。


## 2026-10-24 故事走位與插值移鏡HD

從已推送的473dcf19建立hd/story-motion-20261024，先依重製規格閘門把本切片寫成READY，再沿既有故事控制器補正式HD消費端。沒有重開原版反組譯，也沒有新增美術、清冊或保存欄位。

- 私有故事視圖讀X／Y、OffX／OffY及當前像素相機；保留原生靜止錨點、選圖與旗標。走位沿既有三幀節拍，轉彎、跟隨捲動與一次續行由既有控制器負責。整格私有相機加右／下補畫，圖面裁切保持；來源、native緩衝、時鐘、DAC與RNG不因Draw／F2改變。
- 地形從新增的TerrainUnderlay開始，避免對已完成地形再次套目的色重映射。視差以輸出像素取樣，保留每格3／2比例，來源Mask與目的色RemapMask分開。縮放僅由新底圖的ScaleToRect明示啟用，既有演出沿原繪圖契約；暫時圖面紋理有界回收。
- 最終八份Go閘門34次必要頂層測試通過，另八字串測試與109項原處置保持。兩包各八完整Game.Draw、112次繪製與24次F2，涵蓋四方向、兩種轉彎、雙軸移鏡與跟隨捲動；同一重製程式的原圖主題控制組及一次續行通過；本批未重跑原版FD2.EXE。兩包八張PNG與完整收據相同，私人5762獨立runtime檔核對通過。舊故事兩流程198次繪製、19次F2與22來源世界回歸，12張PNG及完整收據與前批相同。無SKIP、逾時或OOM。唯一數量依58；可重跑入口為tools/verify_fd2_unit_present_hd.py的motion-unit／motion-game及原有五類閘門，具體來源與收據依HD主收據story_motion_hd_followup_20261024。
- 早期拒收包含舊admission斷言、跟隨測具漏設第一格安全帶，以及測具DAC與新HD圖面尺寸斷言。逐項核對來源後修正測具，未改遊戲規則；失敗log與記憶體收據保留。私有地形重映射及輸出精度視差的教訓已追加，335項教訓／199個guard通過。
- README四格由最終完整畫布先核RGBA雜湊再半縮，PNG SHA為2a1a760132cab6d5e6d2081c5632399b734ec84c2fd2177a6f31fd874a799f73。規格61由READY提升CONFORMED-SCOPED／RUNTIME-E1，56／57／58同步；前批靜態限制保留為初始收據範圍，由本節新契約取代。GitHub #204及#263精確回讀，acceptance與其他22項保持，24開啟／16人工／0可能已完成。官方工作清單重新產生。
- 核SHA後清理44張本輪重複／失敗測具暫存圖，釋放81930089 bytes，紀錄在story-motion-cleanup-oct24-r1.json；保留正式畫面、全部log及收據，沒有新錄影。765清冊與私人6740檔保持，私人main仍e9f5403961415e4a785f9e1b31d3dffa33620ce7，沒有新私人同步或Release。FD2一次性容器均已移除，沒有重建工具映像。

本切片是重製顯示近似，只到受控RUNTIME-E1。視差測具將原版FDOTHER16／17接到map0，不代表第18章正常玩家可達性；未登錄地形仍用原圖。下一閘門接0x24618完整徑向／調色，再補0x24B4D與特殊背景；HD2、完整HUD、浮出工具列／加速及全部HD後的正常玩家／集中人工／平台仍待。程式、規格與README以本段所在Git提交及origin/main追溯，未另發版。


## 2026-10-25 九階徑向轉場與baseline加亮HD

從dfee7fad的main建立hd/indexed-transition-20261024，跨日完成本批。先依重製規格閘門將既有0x24618繪製切片寫成READY，再接HD消費端；原始caller解析、九階排程、停留、DAC與一次續行由既有owner保留，沒有重開RE或新增玩法。

- 冷啟動凍結同次地圖／selector／色盤、九個LUT與每階work／VGA。HD預檢逐byte重建原生九階，背景從前一階完整work續接，保留目的色重映射歷史。徑向遮罩使用完整work基底0，沿已核准四色差近似繪HD人物及已登錄地形。新檔modern_hd_indexed_transition.go承接消費端。
- 核對ApplyVGAPaletteDelta後確認是從baseline加亮、飽和於63。HD的32階只改PaletteAdd6，世界重組後以GPU飽和加delta/63，alpha保持；每階由同一最後徑向圖面開始，避免累加前一張圖。這是HD色彩近似，原圖主題保留精確DAC與bit replication。Draw只承認顯示，F2全部預檢後一起發布，回到正常世界時新欄位為零。
- 十份Go閘門49次必要頂層測試通過，另八字串回歸及109個既有審校處置保持。兩包各四完整Game.Draw、288次繪製與32次F2，完整九階／31停留更新／32個DAC／一次續行及HD返回通過；各三個GPU世界另核alpha0／128／255。兩包16張PNG與完整收據相同，5762私人獨立runtime檔通過。原圖主題控制組為同一重製程式，本批未重跑原版FD2.EXE；三個既有caller及一個直接幾何測具分列，不外推新caller或PLAYER-E2。
- 8原生indexed、7HD來源／拒收、7原生入場、5HD轉場、4故事來源、6故事走位及9共用回歸通過。舊故事完整GPU兩流程198次繪製、19次F2與22來源世界保持，12張PNG及完整收據與前批相同。全批無SKIP、逾時或OOM；峰值5396979712 bytes低於6GiB容器限額。這些數字是受控回讀測試，不是正式玩家效能測量。
- 首輪快取F2測具只備原生bank，缺少正規化baseMapTileset而panic；中間一次欄位名稱誤寫造成編譯失敗。回查refreshThemeVisuals後補齊測具，同命令乾淨重跑通過，正式路徑沒有猜補底圖。失敗log與記憶體收據保留，新增可重用教訓；336教訓／200個guard通過。
- README四格從已驗證完整RGBA畫布半縮，順序為幾何起始、徑向末階、加亮及返回，PNG SHA為be41988ccf560a495693aa9c58bf79a0ecca366a24bd918ce597ab2eec27b0c2。唯一數量及現況依58，契約61提升CONFORMED-SCOPED／RUNTIME-E1；56／57與GitHub #204／#263同步，acceptance及其他22項保持。24開啟／16人工／0可能已完成，未關閉整體HD議題。
- 依跨包SHA清理16張私人重複PNG，釋放30686124 bytes，紀錄在indexed-hd-cleanup-oct24-r1.json；正式PNG、全部log／收據保留，沒有新增錄影。765清冊與私人6740檔保持，私人main仍e9f5403961415e4a785f9e1b31d3dffa33620ce7，沒有新私人同步、工具映像或Release。一次性FD2容器完成後移除，收尾另檢查清理狀態。

主證據為HD主收據indexed_transition_hd_followup_20261024，原生owner的實作前SHA以dfee7fad綁定，實作後來源以indexed-hd-source-snapshot-oct24-r1.json定位。永久入口沿tools/verify_fd2_unit_present_hd.py的indexed-native／indexed-unit／indexed-game，完整GPU帶--catalog ABS --pack ABS --out FRESH_PREFIX --timeout 1200，私人包另帶--require-real-runtime；舊故事沿story-game。下一閘門接0x24B4D及天空之鑰／第23章特殊背景；其餘地形、HD2、完整HUD、浮出工具列／加速及全部HD後的正常／集中人工／平台仍待。程式與規格由本段所在提交及origin/main追溯，完整HD目標保持。

## 2026-10-25 交替列位移演出HD

- 從be687d30主線建立hd/row-shake-20261025。沿58既有0x24B4D證據先升規格61為READY，再接戰鬥與故事兩列HD，保留第九列、HUD人物圖示、原生節拍與一次續行；起動失敗補回故事初始化，F2先預備全部owner再發布。
- 十份最終Go閘門39次必要頂層測試與八字串測試通過，109項處置保持。兩包各四流程、100次Draw與12次F2，12張PNG及完整收據相同；舊故事12張及收據保持，原徑向README總覽重生SHA保持。無SKIP／逾時／OOM；峰值4937691136 bytes只作有界測試量測。
- 最初兩次單元拒收來自合成測具沒有同步Game游標及缺前景控制；改測具明示輸入，正式規則與門檻保持。最後補強未物化故事角色的冷失敗回復並重跑六項單元，正式程式與GPU測項保持。
- 清冊765、私人6740檔與原始素材保持，沒有新美術／Release／原版執行器或正常玩家聲明。現況依58、規格61及docs/data/fd2-modern-hd-battlefield-20261019.json的rowshake_hd_followup_20261025。天空之鑰、第23章特殊背景、其餘地形／HD2／HUD／浮出工具列及集中正常／人工／平台仍待。
- 核SHA後刪41份重複PNG，釋放67450093 bytes；保留正式擷取、原始log與完整收據，未產生錄影。清理逐檔紀錄在work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/rowshake-hd-cleanup-oct25-r2.json。Docker收尾及遠端HEAD於提交前後核對。

- 末輪審查修復驗證器的既有indexed-game旗標誤改，另真實重跑288次Draw／32次F2，16張PNG及完整收據保持；六模式與新guard正反控制通過。舊九閘門以最後檢查器另存audit，未重計Go次數。正式Go及字串來源保持，最後來源快照為rowshake-hd-source-snapshot-oct25-r2.json。

## 2026-10-25 地圖23特殊背景與完整DAC下的HD人物

- 從00e300be的main建立hd/special-scenes-20261025。沿既有FDOTHER42／raw ch23 post證據，先寫DRAFT及READY-ART-REVIEWED，再實作原生循環列位移的HD紋理與同次來源綁定。raw ch23對應玩家第24戰，沒有新增故事名稱或重開RE。天空之鑰只生成金鎖母稿，FDOTHER34／ANI0仍DRAFT。
- 新增1872×1152背景與未修改1599×984母稿，正式清冊766項、私人6742檔，舊765項及6740檔逐SHA保持。完整提示、原圖／母稿／配方與SHA依HD主收據special_scene_hd_followup_20261025；固定基底重編兩檔相同。私人獨立提交已推送main並核對遠端HEAD，公開庫只加規格、工具與總覽圖。
- 正式與5763檔獨立runtime各300次完整Game.Draw、6次F2、兩段各一次續行及正常HD返回通過。八張PNG與完整收據跨包及對候選相同；literal／目的色映射／透明區分權、循環0／1／191、alpha、HUD人物、故事私有隔離、冷拒收與晚失敗回復通過。控制組是同一重製程式的原圖主題，本批未重跑FD2.EXE，不新增PLAYER-E2、每frame或硬體時鐘一致聲明。
- 十一份選定最終Go閘門43次必要頂層測試、42個不同測項通過，另八字串回歸與109處置保持。舊列位移四流程100次Draw／12次F2、十二張PNG與完整收據保持。全批無SKIP／逾時／OOM；最終峰值3060375552 bytes，低於6GiB測試限額，不能當正式玩家效能。最後程式／測試／工具1102檔指紋在special-scenes-source-snapshot-oct25-r1.json。
- 早期精簡測具只有LUT0..9，第33次需索引10；補到正式20相位所需0..10，另以十份截短bank作晚失敗反例。GPU測具曾誤要求32份，實際原包23份；後續又抓到改尺寸卻保留map0事件緩衝，改用battle.Load完整載入map23幾何。只修測具，不改正式規則，失敗log與記憶體收據保留。彙整腳本另依正式memory.peak欄位修正，沒有重計Go測試。
- 58唯一狀態表、56／57／61與README同步。README四格從已驗證完整1920畫面半縮，SHA為e0101f719ea8363e7004bbe28fd614af98f841e67484b7fd6adaa643843fdf71。GitHub #204／#263精確回讀後只改最新進度與verify.note，驗收條件及其他22項保持；官方快照／91由工具重生，24開啟／16人工／0可能已完成。339教訓／203個guard通過。
- 核逐檔SHA後刪除5763個獨立runtime檔、28張重複擷取、兩個重編副本及6740個重編連結，釋放1748223812 bytes。保留正式與私人素材、正式畫面、所有log／manifest／收據及未使用金鎖草稿，沒有新錄影。明細在special-scenes-cleanup-oct25-r1.json；已清理包不能重用作獨立實檔證據，需由私人同步工具新建。一次性FD2 Go容器全部移除，沒有重建工具映像或新增Release。

下一步為天空之鑰FDOTHER34／ANI0及其他凍結caller的地圖23來源綁定。其餘地形、HD2、完整HUD、操作介面獨立控制、單列浮出工具列／加速／海報，與全部HD後的正常玩家、蓋亞手腕／凱麗右腳人工確認及平台效能仍待。完整HD目標保持。

- 收尾擁有權掃描列出2808個既有歷史項，均不在本批變更內；本批31個變更檔全部UID1000。只記錄舊檔清單，未遞迴改權限。入口為special-scenes-ownership-oct25-r1.json。


## 2026-10-25 天空之鑰完整HD演出

- 從008a93f9的main建立hd/sky-key-20261025。沿既有0x24336、FDOTHER34及ANI0證據，不重開RE；先審核人物抬手、五色球／藍色物件與金鎖母稿，DRAFT候選修正衣袍誤判金鎖及近景錨點跳變後，依READY契約接入正式owner。
- 新增100張FDOTHER人物／元件與96張ANI近景，三份母稿與兩份配置合計201份新檔。清冊768、私人白名單6943檔，舊766項保持；固定基底清冊、原包與母稿重編全部新檔逐SHA相同。冷載入完整銀行後發布唯讀RGBA快取；每張準備仍核PNG與像素SHA，開始／F2預檢整組。兩份配置JSON是runtime依賴，母稿不隨獨立runtime包。
- 正式與5961檔獨立runtime各202次完整Game.Draw、12次F2與一次返回，13張PNG及完整收據跨候選／正式／runtime相同。鏡頭四格、68張前段、96張ANI、閃白、還原及32張尾段保持；控制組是同一remake的原圖主題。地圖20地形／介面框線仍是像素，仿射、早期縮遠延拓、新姿態及RGB屬美術近似，未重跑FD2.EXE，不新增正常玩家、每frame或硬體時鐘聲明。
- 五份選定最終Go閘門24次必要測試、23個不同測項通過；來源／快取完整性、冷拒收、晚失敗回復、所有owner預備後才換主題、故事私有隔離與舊地圖23保持。舊地圖23八張PNG及完整收據逐SHA相同。另八字串測試及109項處置保持，340教訓／204個guard通過。最後來源快照1107檔在sky-key-source-snapshot-oct25-r1.json。最終閘門無SKIP／逾時／OOM；不把容器峰值當作正式玩家或平台效能。
- 測試首次缺DISPLAY、原版DATO環境變數與相對輸出路徑，均屬啟動設定；補齊後以相同隔離工具鏈重跑。最初單元輪次重複冷載完整主題而逾時，改為同程序唯讀模板。合併來源與流程回歸後抓到反例json.Unmarshal重用共用slice，只修成零值解碼，正式素材與播放器保持；失敗log保留。guard工具排除測試檔，因此綁定同程序來源／流程的正式驗證入口，沒有改掃描器契約。
- 58唯一狀態表、56／57／61、README與GitHub #204／#263同步。README四格來自實際完整1920畫面半縮，SHA為7b38d3d4810bdbd23022505a648a9bfc470353a91322028c9f4bddd1c4b5123f。兩議題驗收條件及其他22項保持；遠端回讀、快照、91 render／verify通過，24項仍開啟。完整HD與集中正常／人工／平台閘門保持，沒有新Release。
- 自動審查拒絕從私人session擷取三次生成呼叫的完整提示詞備份，理由是私人來源轉存需明確授權。已提出限定範圍授權問題，未讀取或轉存該session內容；其他來源與輸出SHA照常保存，這不阻塞功能驗證。
- 清理逐檔核SHA，移除6791份已驗證可重生檔案與26968個候選連結，合計2508221485 bytes。保留正式／私人成品、母稿、正式13張畫面及全部log／收據；沒有新錄影。第一次清理白名單漏掉僅master_file宣告的舊連結而停止，正式目標未改；補用完整清冊白名單，先保存計畫再完成，明細在sky-key-cleanup-plan-oct25-r2.json及sky-key-cleanup-oct25-r2.json。已清理獨立包不能重用作實檔驗收，須從正式清冊重建。

目前現況與後續入口為docs/data/fd2-modern-hd-battlefield-20261019.json的sky_key_hd_followup_20261025及58。其他凍結caller來源綁定、其餘地形、HD2、完整HUD、操作介面獨立控制、單列浮出工具列／加速／海報，以及全部HD後的正常玩家、蓋亞手腕／凱麗右腳人工確認及平台效能仍待。

- 收尾掃描確認本批27個變更檔UID／GID均為1000，6943個私人清冊檔的SHA與擁有權通過。2808個歷史root-owned項均不在本批變更內，沒有遞迴改權限；明細為sky-key-ownership-oct25-r2.json。全樹find第一次90秒逾時，改用有界串流記錄完成同一掃描。FD2一次性Go容器已全部清除，工具映像保留目前版本。私人獨立提交8720011fa0c665dfcbe8d0e663c2a6b3283b87b8已推送main，本機及遠端HEAD一致。公開實作與總覽將隨本批同一次有效更新推送main，版本與Release保持。

## 2026-10-25 物品地圖 HD、凍結背景與未命中字母勘誤

- AI 回復、玩家與 AI 傷害及三種特殊效果，共十四條既有演出接入 HD。新增八張白光效果與未修改母稿，固定配方重編相同。正式清冊、覆蓋數及來源以 58 與 HD 主收據 `item_map_hd_followup_20261025` 為準。私人素材已提交 `06e09c2750c8be449d187a80807e0159930adebe` 並核對遠端 HEAD；公開庫只保存工具、清冊與已獲准總覽。
- 共用 compound 場景凍結同次地圖 23 背景、快取與來源簽章。非恆等 LUT 反例證明必須先附上 staging 再重播；移除提早套色後，同一反例及四種召喚、狀態效果與天空之鑰回歸通過。玩家回復 type5／13 保持既有直接結算，未新增動畫。
- 目視總覽發現 MISS 誤畫煙霧。既有 IDA 主證據早已指出線性 `0x5204A` 的 `74 75 76 76` 是十六進位。本輪只機械式複核四個位元組及既定 EXE 雜湊，修正五個重製迴圈，沒有重開完整 RE。六項共用佇列及實際狀態／召喚分支通過，驗證器改查十進位 116／117／118，拒絕煙霧索引。#270 已關閉，提交後補確切連結。
- 正式與獨立素材包各十四條完整 Game.Draw 流程，抽驗 556 張畫格、42 次 F2 與十四次續行。完整收據與五十張 PNG 逐檔相同。十三份選定 Go 閘門共 41 次必要頂層測試、37 個不同測項通過，無 SKIP／逾時／OOM；其中包含八項文字回歸，109 項既有處置保持。清冊十三項通過，344 條教訓與 207 個 guard 通過。初次清冊測試僅被 45 秒外層時限中止，原紀錄保留；同映像、命令及輸入乾淨重跑通過。
- README 總覽以修正後實際遊戲畫面重建，SHA-256 為 `48646f62744b7fa911b5471cfad97b4341e00af54c1d384b9bb1113fae8e00ad`。舊煙霧總覽及收據降為歷史診斷，不能作最終驗收。#204／#263 同步本切片及剩餘範圍，驗收條件與其他 22 項不變；工作清單由遠端資料及既有工具重生，維持 24 項開啟。
- 逐檔 SHA 與擁有權核對後，刪除 6,144 個重複檔及 20,829 個連結，檔案邏輯大小共 2,070,819,356 bytes。保留正式與私人素材、五十張正式畫面、原始來源、所有收據與必要診斷樣本，沒有新增錄影。明細為 `work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-map-cleanup-oct25-r1.json`；已清理的獨立包重跑時須重新同步。

本批限於受控 RUNTIME-E1。RGB 映色與材質屬美術近似，未重跑 FD2.EXE，未新增正常玩家、硬體時鐘或平台效能聲明。接續其他凍結背景消費端及其餘 HD；地形、HD2、完整 HUD、介面獨立控制、浮出工具列／加速／海報，以及集中正常玩家、蓋亞手腕與凱麗右腳等人工確認仍待。

收尾核對 2,597 個本地文件連結及文化保存入口，39 個變更檔皆為目前使用者擁有，最終程式指紋保持。本輪工作根無 root-owned 檔與錯誤 `.md` 目錄；兩種專案工具映像皆無殘留容器。沒有重建映像、增加 Release 或更動其他專案資源。
## 2026-10-25 玩家第9號法術銜接特殊背景

- 從主線 `1868fc9f` 建立分支。沿既有玩家第9號與地圖23證據及61的READY契約，封存同次typed input、原生work／VGA、palette、快取及staging。所有調色、效果、結果與停留使用封存背景，F2重驗來源及原生重組；不讀後續live stage。共用凍結輸入先恢復同次staging，避免重複套用目的映射。
- 修正前的非恆等LUT／BlitMode0及實際地圖反例均重現缺口；修正後三項來源、隔離與原子拒收測試通過。完整命中／MISS各58張不同畫格，兩組共116張，完成16次F2及兩次續行與HD返回。像素主題控制組的原生狀態、交易及RNG保持。
- 正式與私人實體圖包的18張PNG與完整收據相同。六份Go閘門27次必要頂層測試、26個不同測項通過，包含五項一般玩家9、九項物品共用來源及八項文字回歸；109項既有文字判定保持。無SKIP／逾時／OOM，最大測試峰值5714178048 bytes，不推論正常玩家或其他平台效能。
- 沒有新美術或清冊修改，直接唯讀私人庫既有實體檔驗證，沒有重建大型執行包。原包與私人素材均保持唯讀；確認跨包SHA後刪除18張重複擷取，共10054996 bytes，正式18張與兩側收據／log保留，沒有錄影。明細依HD主收據 `player9_stage_hd_followup_20261025`。
- README新增實際遊戲四格，SHA-256為 `a1d6886a46530a45f1c0afde48a15de2d8cae8e3d6ba0a55bd0301d9fe4db23f`。已目視命中986與MISS；手繪綠色背景可見，城堡原生圖塊與框線仍為像素。這一限制未以圖層存在或綠色測試遮掩。

本批限受控RUNTIME-E1，未重跑FD2.EXE，不新增正常玩家、硬體時鐘或平台聲明。接續回復、10／12前奏與其他徑向／凍結消費端的背景遮罩，其餘HD地形、HD2、完整HUD／操作介面、工具列與集中人工仍待；蓋亞手腕、凱麗右腳工單保持開啟。

收尾核對2604個本地文件連結與文化保存入口，16個變更檔皆為目前使用者擁有，最終程式指紋保持。344條教訓及207個guard通過；工作根無新增root-owned檔或錯誤`.md`目錄，兩種專案工具映像無殘留容器。#204／#263已同步，驗收條件與其他22項保持；工具重生的工作清單仍24項開啟、16項人工、0項訊號消失。

## 2026-10-25 回復演出的特殊背景與中段停格

- 從主線`6e671bbb`建立`hd/heal-stage-20261025`。沿既有`sub_21EB1`與`sub_11EEE`證據及61的READY契約，修正回復13～16從零緩衝建立前段而漏掉地圖23背景的缺陷。原生工作區先恢復同次staging，HD整張背景明示圖層身分，依原生遮罩套用0～3次調色。保存交易前後兩份密封來源，F2重驗，不讀後續live stage。
- 同一主證據顯示第九張之後只等待200ms，十六張後才呼叫普通地圖重繪。兩種主題的中段停格改保留第九張，直接核對原生320×200像素；既有等待、音效、五次閃爍、交易時點及續行不改。冷來源缺失原子拒收，交易後重繪失敗恢復單位欄位、HP／MP、RNG及原生工作區。
- 正式八組完整Game.Draw驗472張畫格、88次F2、八次交易／續行及HD返回。私人實體素材只重跑13單目標與16多目標兩端，共118張、22次F2及兩次返回；這兩組完整收據與18張PNG逐SHA相同，不宣稱兩包完整八組。64張原生前段、受控0～3次映色、背景隔離、來源修改拒收與AI正式入口通過。
- 十份選定閘門32次必要頂層測試、31個不同測項通過，包含八項文字回歸、七項共用徑向與既有一般地圖／原生玩家流程；109項文字處置保持。無SKIP／逾時／OOM，最大測試峰值6052179968 bytes。345條教訓與207個guard通過。初測把停格要求新的Draw確認，另少算五次閃爍的兩張畫格；AI測具也曾混入缺原生AI欄位的玩家記錄。均依既有契約修正測具後乾淨重跑，失敗log保留，不改正式規則。
- README四格取自已驗證完整畫面半縮，SHA-256為`ecc57a8799353e144b82ade3a3c97214d161900bbd711c015e70a154a5f4c160`。已目視前段、中段、群體回復69／69及返回，城堡地形及框線仍為像素。56／57／58／61、#204／#263同步，#271關閉；驗收條件及其他22項保持，官方工具重生工作清單為24項開啟、16項人工、0項訊號消失。來源快照、閘門、跨包比較與範圍依HD主收據`heal_stage_hd_followup_20261025`。
- 沒有新美術或清冊修改，直接唯讀私人庫既有實體素材，私人提交保持`06e09c2750c8be449d187a80807e0159930adebe`。核對逐檔SHA後刪除96張重複PNG，共137173186 bytes；保留正式72張、兩側收據／log、原始與私人素材，沒有錄影或大型素材複本。清理明細為`work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-stage-cleanup-oct25-r1.json`。

本批限受控RUNTIME-E1，未重跑FD2.EXE，不新增正常玩家、硬體時鐘或平台聲明。接續10／12前奏、其他徑向／單位轉場／獎勵等背景消費端，以及其餘HD地形、HD2、完整HUD與介面獨立控制、浮出工具列／加速／海報。全部串接後再集中正常玩家、蓋亞手腕與凱麗右腳人工確認及平台驗收。

收尾核對本批文件及權威入口的2719個本地連結與文化保存項目，21個變更檔皆為UID／GID1000；十個正式來源／測試／工具指紋保持。本批輸出無root-owned檔或錯誤`.md`目錄，兩種專案工具映像無殘留容器。整樹連結掃描曾碰到已清理的歷史work連結，另全歷史文件有九個未修改的文字或舊交付連結；最終檢查明示本批文件範圍，不把這些診斷當產品失敗。沒有重建映像或新增Release。


## 2026-10-25 指令10～12的特殊背景與AI重繪來源

- 從主線818e832c建立hd/command1012-stage-20261025。沿21548／2189A及11EEE已閉合證據，先補61的READY契約，再將主段底圖、結果與11／12前奏接上同次map23手繪背景。前奏的地形快照及最後普通重繪先恢復staging，HD背景沿單次徑向遮罩調色；三張1F558衍生surface仍只取原始地形，不猜補背景或改取樣表。
- 原生非恆等LUT反例先重現已合成工作區的重複目的映色，修正後十張前奏及尾段重建通過。主段與前奏密封typed input、work／VGA、LUT、幾何及stage，F2重驗來源與原生重組；冷缺件與來源修改原子拒收，原交易時點、HP／MP／RNG、音效、等待及一次續行保持。
- AI首輪缺class19抗性資料，依既有AI command10測具改用明示class5／抗性10，不改正式表。第二輪抓到最後重繪誤用演出前HUD，另保存hdSteady來源後通過。R3實測AI11／12的物件名冊相同、HUD不同，先前姿勢假說已訂正；舊formal R1有界停止，未拿部分流程作最終驗收。全部失敗log保留。
- 正式九組涵蓋三指令各命中、MISS及雙目標，共1083次完整Game.Draw、90次F2與九次返回。私人實體素材只重跑10單目標與12雙目標兩端，237次Draw、19次F2及兩次返回；完整兩組收據與13張PNG對正式逐SHA相同。六份最終閘門32次必要頂層測試、31個不同測項，含五項核心／AI、十五項原生、兩項一般地圖與八項字串回歸；109項文字處置保持。
- 最終閘門無SKIP／逾時／OOM，最大峰值6159302656 bytes，最終兩GPU程序memory.events.max均為0。較早私人R1曾觸及6GiB並回收，max增加147但無OOM，記錄保留；不由最終測試推論正式玩家或平台餘裕。345條教訓與207個guard保持。
- README四格由最終完整遊戲畫面半縮，SHA-256為8e344c77fedc5359e9aa84aabd63594d641576e9e8ecba457f65aef7df7416f1，已檢視前奏、底圖、雙目標916／986及返回。綠色背景、人物與數字為HD，城堡及框線仍為像素。56／57／58／61及#204／#263同步，#272關閉；總項驗收條件與其他22項保持，工作清單由主機遠端資料及官方工具重生，24項開啟、16項人工、0項訊號消失。GitHub初次列表仍快取272，改用no-cache排序查詢並直接GET核對closed，不當成產品或憑證問題。
- 無新美術、清冊或私人提交，直接唯讀既有私人實體素材，未複製大型圖包。逐檔SHA核對後刪除76張重複擷取，共35948151 bytes，保留正式63張、所有收據／log與原版／私人素材，沒有錄影。完整來源、命令前綴、峰值與清理依HD主收據commands1012_stage_hd_followup_20261025。

本批限受控RUNTIME-E1，未重跑FD2.EXE，不新增正常玩家、硬體時鐘或平台聲明。接續其餘特殊背景消費端的可達性與串接稽核，再完成地形、HD2、完整HUD／介面獨立控制與浮出工具列／加速／海報。全部HD後集中正常玩家、蓋亞手腕與凱麗右腳人工確認及平台驗收。完整HD目標保持。

收尾核對2728個本地文件連結與文化保存入口，18個變更檔的UID／GID均為1000，十個來源／測試／工具指紋保持。本批輸出無root-owned檔或錯誤`.md`目錄；一次性Go與素材容器已退出並移除，沒有重建映像或新增Release，也未變更其他專案資源。


## 2026-10-25 移位雙段單位轉場的同次HD背景

從main的df96572d建立hd/unit-stage-20261025。先查58不要重做索引，既有22253主證據沒有其聲稱的地形snapshot writer，因此只重開這個缺口。沿既有IDA9.4 Python3.12映像，不新建工具鏈；一次smoke與154指令狹義匯出有效，固定EXE雜湊、原名、位址及推論等級保持。222aa呼叫11EEE，222a4加8088h，456／13／8參數由直接指令與Capstone交叉核對。原始資料唯讀，臨時i64隨容器移除。新canonical由58索引，先在61寫READY，再接runtime。222aa新增instruction索引，22253既有說明補證據；清冊註記不一致守衛先拒收，經官方refresh_compact_annotations重綁後11項契約通過，1305函式及分類數保持。原始匯出保留取得當時註記，不改寫歷史證據。

原生22253建立snapshot前恢復同次map23 staging；兩段各自密封，HD整張背景與徑向來源明示傳入。F2已有快取仍驗素材與來源；座標寫入、18／24條帶、MP／道具與一次續行保持。未更動美術、清冊、私人庫或保存格式。其餘caller僅按實際可達性稽核，不擴成幾何測具矩陣。

before-r1先暴露測具camera與NativeView不同步，修測具後before-r2才取得缺少stage種子的有效反例。core-r1下方目標超出物品距離，修正施術者站位；core-r2將通用徑向的第三次疊色錯套到互斥區域，依既有排程修測試為0～2。兩項均屬測具條件，未調正式規則。GPU-r3完整繪製及結算已過，但尾端分支門檻發現VisualY16並非cameraY14，兩段都是24列。依已閉合StripLayout將一端放到14，另一端19，確實覆蓋18→24及24→18；未改production。core-r4四項乾淨通過。

正式與私人實體素材各跑兩種雙段完整Game.Draw，每組分別200／200次Draw、20次F2及一次HD返回，兩包全部收據與24張PNG相同。共9份閘門、32次必要測試、29個不同測項通過，無SKIP、逾時或OOM。CPU／GPU獨立條帶、原圖控制組、落點／MP／物品／RNG、隔離與拒收通過，私人反向測試觸及6GiB上限，memory.events.max=585、oom與oom_kill為零；這是容器壓力事件，不推論平台餘裕，也不為追求max零而重跑。所有memory.events原樣保存。字串盤點bytes與前批相同，原處置不需重綁。這是受控RUNTIME-E1；新IDA不代表新原版執行器或正常通關。

README新增四格實際遊戲畫面，保留文化入口與多語系圖片。城堡地形、框線、其他可達來源、HD2、完整介面／工具列及全部HD後集中正常／人工／平台仍待。來源、版本、來源快照、精確前綴及雜湊依HD主收據unit_present_stage_hd_followup_20261025，唯一數量表為58。

跨包24張私人擷取副本逐SHA核對正式保留檔後刪除，釋放13064276 bytes。完整JSON／log／memory及正式24張PNG保持；本批沒有錄影、素材同步、新映像或Release，未觸及其他專案容器。清理逐檔雜湊依同份主收據cleanup。

官方#273依完整收據關閉，#204／#263更新目前來源與限制且驗收文字保持，其他22項標題／內文／標籤不動。主機gh核對遠端，官方純函式重生快照；24項仍開啟、16項人工判定，標籤同步計畫無差異。完整HD總項仍開啟。

本批21份變更與專屬輸出擁有權1000:1000，2740個本地文件入口與README文化正對照通過；1129份來源快照雜湊保持。官方工作清單可能完成0項，345條教訓／207個守衛通過。Go／Xvfb容器已退出，沒有新工具鏈映像或非預期FD2容器，其他專案不動。


## 2026-10-25 戰後成功訊息的同次HD背景

從main的5cf81a74建立hd/reward-stage-20261025。目前消費端稽核發現rewardMessage仍呼叫沒有stage的普通helper，修改前真實map23與正式圖包反例重現「reward lost frozen map23 stage」。反例的Go測具另依先前記錄SHA保存原bytes；生產程式由base commit回查。沒有重開RE，沿既有成功訊息與11EEE背景契約，先在61寫READY再實作。

訊息改用既有modernHDCompoundMapScene，刪除重複的四個背景欄位。新場景保存同次stage與before-acted來源，F2驗簽章、原生重組與面板底圖後才發布。開收框、字形／頭像、ACK、原交易及灰化時點保持，滿背包原生規則未改。

來源隔離、八種密封資料破壞、錯圖源冷暖拒收與原圖主題回復通過。兩包各28張主要Game.Draw邊界、28次F2與兩次HD返回，完整JSON及12張PNG相同。獨立CPU世界、框外非空背景、480頭像／文字裁切、原圖控制組、物品／金幣及一次續行通過。一般地圖26張／26次F2與HUD／滿背包交易回歸保持，共7份Go閘門、16次必要測試、15個不同測項，無SKIP／逾時／OOM，所有memory.events保留。四個核心／舊回歸共用容器，memory.peak是累計峰值，不拿來推論各項或平台餘裕。

README新增物品、金錢、關框及返回四格實圖，原文化入口與四語圖片保持。實圖檢視發現共用法術測具把BattleFig覆寫為9，地圖仍為索爾key0；r2功能驗證雖通過，該組畫面不作最終展示。只在rewardStageGame恢復DATO0並加入key0一致性斷言，r3重跑受影響來源與兩包Game.Draw；正式程式未再改。一般地圖、文字／HUD與滿背包回歸不依賴此測具，沿r2收據與來源快照保留，不重跑。清冊、私人素材、字串盤點及109項處置保持，沒有新圖、私人提交、錄影、原版執行器、正常玩家或Release。城堡、框線、滿背包問句、其他提示／HUD、地形、HD2、浮出工具列與全部HD後集中正常／人工／平台仍待。來源、精確前綴、雜湊與重跑方式沿主收據reward_message_stage_hd_followup_20261025，唯一數量表為58。

最終私人測試觸及6GiB上限，memory.events.max=228、oom／oom_kill為零；較早錯配測具r2的max=1941另保留診斷收據。不為取得max零而重跑，不推論平台餘裕。文件彙整配方曾因變數遮蔽中斷，修正後先核對既有總覽PNG bytes再接續，遊戲測試未重跑。

核對逐檔SHA後刪除35張擷取，釋放22935605 bytes：12張已驗私人副本與23張已被一致性修正取代的r2測具圖；保留r2錯配代表圖一張、正式12張與一般地圖10張，全部JSON／log／memory保持。無錄影、來源素材或交付包被刪。

官方#274依完整收據關閉，#204／#263更新已驗範圍且原驗收不變，其他22項標題／內文／標籤保持。主機gh核對遠端，官方純函式重生快照：24項仍開啟、16項人工判定，標籤無差異。完整HD總項保持開啟。

最終16份變更與本批專屬輸出擁有權1000:1000；2749個本地文件入口與README文化正對照通過，1129份最終來源雜湊保持。官方工作清單可能完成0項，345條教訓／207個守衛通過。沒有新工具鏈映像或非預期FD2容器，其他專案不動。

## 2026-10-25 滿背包獎勵HD串接

- 以e6168f2e與明示私人素材重現問句缺少HD文字／頭像，沿61 READY及既有1AA56證據實作。問句、逐字回覆、YES／NO與選擇器取消以明示工作階段重組；原生清底同步裁掉舊字，頭像／文字隨面板裁切，問句保留同次密封世界，選擇器沿正常地圖。
- 兩包各291次完整Game.Draw與40個F2邊界，五分支及五次續行、獨立CPU背景與像素控制通過；完整收據及10張PNG相同。七份選定Go檢查25次頂層測試，無SKIP／逾時／OOM；全部峰值與memory.events依主收據。既有109筆字串審查只重綁來源，判定與筆數保持。
- 初次未明示圖包的測試SKIP不採用。第一次完整GPU測具因go.mod仍為1.21而共用迴圈閉包，續行計數失敗；依既有教訓改區域副本，未改正式遊戲。已啟動的舊測具診斷提前停止，兩包用相同配額乾淨重跑。
- 受控RUNTIME-E1，未重跑FD2.EXE或提升正常玩家／平台。無新美術、清冊或私人提交。README保留文化與四語入口，新增正式遊戲總覽。#275與父工單依本批收據更新；完整HD及集中驗收保持開啟。清理與提交狀態於本節後續追加，來源入口為HD主收據full_inventory_reward_hd_followup_20261025。

本輪清理：核對SHA後刪除33張已驗證重複或失效測具擷取，釋放20,507,983 bytes。正式十張圖、總覽、兩包完整收據、日誌及記憶體事件保留；本輪未錄影。逐檔記錄為full-reward-cleanup-r4.json。私人包最終觸及6GiB，memory.events.max=614，OOM與oom_kill均為零；不推論平台餘裕。

收尾核對：官方worklist為24條，16條要人判、可能已完成0條；其餘22個工單未變。2,758個本批文件本地目標、四語圖片與文化保存入口通過；22個變更檔及本輪輸出擁有權正常。FD2測試容器已清理，未重建映像或變更私人素材庫。確切提交與推送狀態由Git紀錄及#275結案留言回查。

## 2026-10-25 共用死亡動畫的特殊背景與HUD

- 上一輪8e36718d為已推送實質進展。本輪先查#265，確認混合header只保留錯誤裝備來源與正常可達性未知，未刪守衛、未重開RE。轉向已證實1DB65／11EEE缺口，依61 READY補十三拍姿勢、B底圖及steady的同次stage；密封來源並在HD／F2獨立重組26份native work／VGA。
- 修改前B段缺stage反例及實圖提前HUDUnit反例成立；前25拍不再額外疊穩定HUD人物。首輪完整測具誤將steady的空Viewport當作空視窗，修測具採正式預設後重跑，不列產品缺陷。
- 兩包各三caller、231次主段Draw與20張狀態訊息前綴、78個死亡邊界及三次返回、24次F2通過，完整收據及18張PNG相同。七份選定Go檢查29次頂層測試、28個不同測項；九類來源破壞、live隔離、冷拒收及舊一般地圖逐RGB診斷通過，無SKIP／逾時／OOM。所有峰值及memory.events原樣保留，不推論平台餘裕。
- 普通行動與狀態保留State／Unit身分，不重播HP／RNG／獎勵，事件私有狀態與原先回復保持。109筆字串審查與盤點SHA完全不變。無新美術、清冊或私人提交，#276與父工單依收據更新；完整HD及集中正常／人工／平台仍待。正式來源入口為HD主收據death_stage_hd_followup_20261025，清理與Git交付狀態於本節後續記錄。

本輪清理：逐SHA核對後刪除21張重複或失效擷取，釋放11,592,061 bytes；正式18張圖、總覽、兩包收據、原始日誌及一張提前HUD代表圖保留，未錄影。完整正式／私人測試均觸及6GiB，memory.events.max分別1287／358，OOM與oom_kill皆零，不推論平台餘裕。

收尾核對：官方worklist維持24條，16條要人判、可能已完成0條，其他22工單與#265內容未變。2,767個本地文件目標、四語圖片、文化保存入口及舊正常死亡圖均通過；17個變更檔與本輪輸出擁有權正常。新增HUD圖層教訓及正式程式守衛，全部教訓檢查通過。FD2測試容器已清理，私人素材庫未改；確切提交與推送狀態由Git紀錄及#276結案留言回查。

## 2026-10-25 單列浮出工具列的主題與四語

- 從已同步d086b9db建立分支，先重跑完整圖源覆蓋，264組2118幀已登錄，408來源阻擋保持。未依舊剩餘欄位重開已完成天空之鑰或凍結consumer；改正主收據頂層過期摘要，唯一現況仍依58。
- 依61 READY把正式入口改成640×400／1920×1200頂端浮出列，最終遊戲層不平移；主題與四語按鈕沿F2／F4相同閘門，觸控不誤傳地圖。保留明示fixed相容模式，舊--player-toolbar擷取器設定該模式。
- 實際字界反例顯示舊中文字低一像素，改按墨跡置中。兩次GUI失敗分別是測具語系順序及用wall-clock推定Update數；讀取既有localeIDs、90次Update與Xvfb契約後修測具，沒有放寬產品規則。失敗日誌及記憶體證據保留。
- 16步真實滑鼠／F2／F4／F1、16張GPU逐byte控制、完整13780 bytes保存保持及四語通過；兩項unit、十二項既有回歸與固定列真實輸入通過，全部選定測試無SKIP、逾時或OOM。新增三個語系按鈕字串，109既有處置由官方工具遷移至112。無新美術／私人提交，沒有重跑原版或提升完整正常玩家與平台聲明。
- 正式來源為HD主收據floating_toolbar_followup_20261025，重跑入口tools/verify_fd2_floating_toolbar.py與規格61。README保留文化及四語原版圖片，新增實際浮出列四語總覽。完整HD、其餘控制及集中人工驗收保持未完成。清理、Issue與Git核對於本節後續追加。

收尾核對：2661個本地文件目標、README文化與四語入口、19個變更檔及本輪產物擁有權通過。逐SHA刪除30份失效擷取與可重建輸入執行檔，釋放81,828,831 bytes；未錄影，正式圖與全部日誌／記憶體證據保留。另修舊固定列測具的連續F4同步，逐筆等待實際GPU確認後乾淨重跑。新增GUI同步教訓，產品顯隱政策保持。Issue及Git交付狀態由本輪留言與提交回查。

官方Issue提交前再次拉取並由工具產生快照／頁面，24條維持開啟、16條要人判、可能已完成0條；父工單204／263保留原驗收與標籤，其餘22工單未變。全部本輪FD2容器已退出；未重建映像，原版EXE雜湊與私人素材庫保持。

## 2026-10-25 蜜蒂音速刃的玩家來源與HD播放

- 依目前9f386e80與遠端工單恢復，原始bytes反證舊ID30無取得來源停止線：growth18→learn5於Lv2授予30，map16 slot36為我方蜜蒂。保留原始研究證據；過期停止線已依使用者後續指示刪除。28／31、AI mode8及#265守衛不重開。主證據為docs/data/ida/fd2_command30_player_acquisition_20261025.json，重跑入口已掛58、61及README。
- 依61 READY先補保存游標四格選取與正式target入口，再重用原生／HD派生打擊播放器；讀取已登錄54／56、14幀、九張前導與各目標20張轉場。補載既有音效49，保留frame2的獨立音效、frame4耗魔及frame11逐目標命中。死亡發布沿ApplyHPDamage，晚期回復同步還原死亡暫態。沒有新美術、規則或保存格式。
- 正式升級seed7一次300 EXP取得技能，current-save完整原子保存後以有界新程序冷讀。runtime遮罩保留30；persistent依原有baseline契約保持。目標確認／取消、特殊直線與state-only下一次Go RNG、負傷害、死亡、冷缺件及晚期回復通過。核心23項頂層回歸與HD原子性三指令、29完整GPU保持。
- 雙側各108次HD繪製、78張完整最終GPU與八相位F2，MP一次／兩目標HP各一次、續行與owner釋放通過。獨立CPU逐完整畫面最大單色差1，22項反例拒收。全FIGANI及compound來源、字串審查通過；首輪兩項分離包測試因缺明示FD2_TEST_ORIGINAL_PACK而SKIP，補參數後兩項重跑PASS。無未解的SKIP、逾時或OOM。
- 失敗測具與原因保留：最初未載FDOTHER49為正式缺口；存檔測具誤要求persistent同步、缺field前置與raw roster旗標，已依既有契約修正；反例工具末幀固定17不適用14幀音速刃，改來源對應後重跑。主收據分列失敗前與修正後，不把測具失敗列為產品缺陷。
- 右側藍框與血條僅可丟棄原型，尚待使用者回覆，未接正式Game.Draw。README新增四張實際遊戲畫面的總覽並保留原版四語與文化保存。完整正常鍵盤、真正地圖返回、人工造型／聽感及平台仍待全部HD後集中驗收；不新增原版動態或PLAYER-E2聲明。私人包與清冊無變更。本輪未錄影，收尾清理、Issue及Git結果於後續追加。

本輪清理：逐SHA核對後刪除已被r2取代的原型PNG，釋放147,845 bytes；保留待選r2、實際GPU圖、原始失敗與記憶體收據。未錄影、未建映像；本輪FD2容器已退出且無遺留。全部檢查無OOM／逾時，不據此推論平台餘裕。

提交前核對：2,795個變更文件本地目標、16個正式來源綁定與本輪輸出擁有權通過，文化保存與原版四語入口保留。文件檢查器預設會掃入大型work，已停止該容器並改為50秒有界的變更文件檢查，結果通過。公開／私人素材清冊未變，無原版EXE改寫。當前工單快照25條含本輪#277，結案與父工單更新由提交後遠端再拉回。

補充字型驗證：最後核對指出初輪未啟用HD面板字型。沿現有驗證器新增--with-panel-text後補跑同一音速刃全流程，99次HD繪製、八相位17,632個文字RGBA採樣通過，README總覽改採此實際GPU版本。未改正式字型、血條或框線，獨立CPU及面板反例結果於後續追加。

高解析字型版本的獨立CPU完整78畫面及30項反例拒收通過，最大單色差1。現況文件與新增歷史勘誤共2797個本地目標通過。舊handoff的歷史本機封包已清理而連結失效，本次只驗追加段落，不重建過期交付或列為產品缺陷。

實質提交bda7119e15b3914a29c4a1bbac5fcb15e9d165f7已推送並核對遠端main。#277按明示RUNTIME-E1範圍結案，父工單204／263保留原驗收與完整HD剩餘工作；其餘22個開啟工單未變。官方工具重拉產生24條、16條要人判、可能已完成0條，結案後文件2797個本地目標通過。來源綁定保持，最終同步提交由Git紀錄回查。

## 2026-10-25 蓋亞v2普通攻擊、死亡與戰後補驗

- 從已同步f4185672建立分支，讀取現行#268／#269／#263；v2技術接入已完成，普通操作仍缺現行圖包證據，人工確認維持全部HD後集中。依61 READY補唯讀actor-art取樣與通用normal-action核對分組，沒有改Go、素材、清冊、規則或保存格式。
- 既有promo映像已清理。核對現有映像與Dockerfile後確認Go映像缺xdotool／ImageMagick／PulseAudio，從原Dockerfile恢復同一fd2-promo:20261017-r1，未建同功能新名稱。實際image ID記主收據；執行UID1000、網路關閉、來源唯讀與資源有界。
- 以既有第23章已審查來源槽及seed1，正常進軍八回合、清除擋路敵軍、攻擊116隊長、觸發死亡與戰後、進第24章整備，F2保存相同並由新程序正常LOAD。GUI595.558秒、峰值5445234688 bytes，沒有OOM或逾時；僅2CPU軟體GL條件，不外推平台效能。
- 五張實際PNG均核對目前v2完整高覆蓋核心，涵蓋91:0待機姿勢、蓄力、出拳與收勢；每張至少270878點。擷取前frame5的實際PNG相容2／3／4，不強迫採audit索引；保留所有相容集合。人物與隊長／冷讀各十項反例拒收，死亡三張適用A／B核心通過，十五張其他觀測不提升。
- README舊圖屬第三章91:9／10收勢，這兩幀在v2保持。改展示本次第23章新版91:0完整PNG，不把不同姿勢說成手腕前後對照；同步56／57／58與61。來源祖先、章前AP／DP／DX／HP政策及無章內注入照舊揭露。不新增自然難度、原版動態、PLAYER-E2、完整十五幀普通擷取或造型人工完成聲明。
- 主收據gaia_v2_normal_followup_20261025保存來源、工具、映像、正常按鍵與保存、來源核心、拒收及記憶體，並回鏈原v2與舊README證據。#268人工確認仍開啟，#269與完整HD剩餘範圍保持。本輪未錄影；清理、Issue及Git交付於後續追加。

### 2026-10-25：刪除失效停止線，保留後續傷害調整方向

- 依使用者明示刪除ID30無學招來源、玩家選取／分段演出未接與ID32–35缺indexed演出的過期斷言。同步修正13、56、57、原始證據註記及舊交接／歷史快照，不再只在末尾追加勘誤。原始bytes、位址、敵方mode8與尚未證實的28／31證據保留。
- 專案AGENTS及61記錄新指示：已驗HD絕招不因舊停止線重開；傷害公式允許後續調整，忠實模式保留原版預設。資料格式及調整介面未先行定案，本批規則不變。

- 補舊fixture模式回歸：現行蓋亞十五來源幀完整Game.Draw／F2通過，258次HD繪製，64個戰鬥／65個返回DAC保持；逐圖不透明核心十五張全數通過。兩輪49.557／12.020秒，峰值1329004544／728256512 bytes，無OOM／逾時。五張普通GUI取樣與十五幀fixture分開聲明。
- 原始GUI執行檔核對SHA後刪除，釋放20618216 bytes；來源、必要PNG、收據與記憶體紀錄保留。本批沒有錄影。
- 遠端HD總項與正常路徑工單同步刪除高解析未定案、僅有原生尺寸候選及蓋亞FIGANI仍為原版等過期限制；目前驗收範圍與尚待項分開記錄。

- 傷害公式後續擴充已開正式工單#278，DRAFT規格掛回56、58及61；HD總項#4、#201、#202、#204、#263及#268內文已同步。#268／#269維持集中人工確認，沒有誤關完整HD。

- 同族補查另刪除ID24／29分段演出／SFX未接、32–35原生演出仍全面拒收及ID35舊畫格未知停止線；對照58、61及已閉合共用／尾段證據，不新增原版RE。

- 最終檢查：四份Python語法、22份來源／收據綁定、2705個本地連結及七項舊斷言刪除檢查通過；25條worklist無殘留誤開訊號，348條教訓／209守衛檢查通過。本批輸出逐項擁有權正確，無root產物或.md目錄。廣域歷史work遍歷主動停止後改查本批輸出，未把停止碼混入產品驗收。

- Docker收尾已核對：本批一次性容器皆退出且已移除，無非預期FD2容器；保留目前五項工具鏈，沒有重複GUI映像。其他專案容器及映像未動。私人素材庫乾淨，沒有新資產或清冊變更。

### 2026-10-25：十一圖共用純草地HD正式接入

- 前輪為實質進展，從a234b1c7乾淨main開hd/shared-grass-20261025。依現況58及61選實際缺件；物品演出的舊DRAFT已由既有CONFORMED取代，本輪刪除過期缺接說法，不重做已接HD絕招或開matching。
- map0:144的原生索引、opaque及remap內容與另外十圖逐byte相同，且均有正式selector消費。重用原生來源匯出及33bank／8256幀獨立回讀，不猜地形語意。新手繪母稿沿既有房屋草地風格，縮至144後做八像素邊帶接縫近似；原稿保持，RGB不混原版。
- READY先記來源、scope及驗收，再用受版控編製／登錄工具。正式清冊由769擴至781，新增一母稿及十一稀疏集合共用一PNG；稀疏地形64槽，正式Go與規則／RNG／保存格式未變。
- 十來源反例、五登錄原子拒收及正對照通過；既有四項地形測試、66個實際bank／selector GPU世界與一原樣Game.Draw／F2對獨立CPU通過，另十一份錯遮罩拒收。首輪編譯誤用NativeFrameInput成員，修測具後同預算重跑；反例GPU輸入與目標清冊撞名，改gpu-前綴後重跑通過，沿既有教訓更新。GPU首次通過後另補清冊／RGBA綁定再驗，未把舊收據補欄冒充新執行。
- 首章正常標題LOAD、游標(23,10)、F2、走行與兩次攻擊取樣、來源槽新程序LOAD通過。37張PNG／126操作；兩張地形PNG各兩塊新草地，共82944個RGBA像素，總44／46匹配包含既有物件。來源保存bytes保持；F9回載不等於自然戰鬥返回或戰後變更保存，相關布林仍False。GUI447.993秒，峰值5394677760 bytes，6GiB／2CPU無OOM／逾時，不外推平台效能。
- 新母稿及成品同步唯一私人庫，6954檔逐項SHA，舊6952檔保持；私人清冊實檔驗781項通過。README新增完整普通視窗PNG，沒有裁切、去字或補畫；文化保存及四語原版展示保持。其他草地／地形、HD2、完整介面／工具列、全部HD後集中人工與平台仍待，目標不結案。

- 私人提交9618f47e22eeb971b7196ac6cff25328bd963459已推送並核對真正遠端相同，保持private。正常實檔包與私人6954檔逐SHA相同後，保留manifest並刪除重複實檔包、可重建執行檔及兩張一次性參照，移除3199469251 bytes。本批無錄影；必要正常PNG與收據保留。

- 同步刪除61中「原生HD契約仍為DRAFT、LUT與renderer尚待」的過期停止線，改引用已接的稀疏地形及本批證據。正規化圖集不等於原生圖層的真實限制保留，不因此重開無玩家依賴的來源考古。

- 提交前官方遠端快照25項、worklist verify及348條教訓／209守衛通過；四份Python語法、八來源綁定、全部本地連結、目前輸出擁有權與無.md目錄通過。最後三份地形／GPU／GUI收據均無SKIP／OOM／逾時。來源槽及讀檔bytes保持，README文化保存入口及原版四語圖未改。

- Docker最終檢查：本批所有一次性容器皆已移除，FD2 Go／GUI／Pillow容器清單為空；沿既有工具鏈，未新建映像或volume。私人庫main與遠端9618f47e相同且乾淨；其他專案資源未動。

### 2026-10-25：十四種草泥交界HD接入十一圖

- 從f76e24e4建立hd/grass-earth-edges-20261025，沿既有33bank／8256幀來源匯出與原生稀疏地形，補map0實際使用145～151、153～159及其他十圖同內容共141槽。先記61 READY契約，再編製與登錄；沒有新增原版RE、Go正式邏輯、規則或保存格式。
- 新泥土母稿沿既有草地／房屋手繪風格目視，完整保存。RGB只用兩份新母稿，原生索引分組僅作製圖遮罩，不新增玩法語意。邊帶接縫與半徑2柔化屬明示美術近似，alpha沿原生opaque六倍；十四圖290304個RGBA像素及相同材質端點核對通過。
- 十一圖66個完整GPU世界及一原樣首章Game.Draw／F2對獨立CPU、十一錯遮罩、十來源／五登錄拒收通過。共享登錄器的舊純草地模式另跑正反例保持相容。GPU267.319秒、峰值4400558080 bytes；無OOM／逾時。測具首輪Path下標錯誤已修正，同候選／同GPU重跑通過，沒有改正式程式調參。
- 正式清冊793項、稀疏地形205槽；舊781項保持。十五圖源同步唯一私人庫，6969檔核對，舊6954檔bytes及mtime保持。同步工具新增同inode跳過，仍先驗清冊SHA及manifest；別名重跑、mtime與錯SHA反例通過，不另建重複完整包。私人提交1f5b75f18e4d50e6c611b85ebec7305a1c2e1e3d已推送，實際遠端相同且仍為private。
- 首章正常標題LOAD、游標(6,23)、F2、走行／取消／兩次攻擊取樣及來源槽新程序LOAD通過，41張PNG、118操作。主取景兩張各20塊完整相符，148候選被人物遮擋而拒收；同輪指令環F2前後各27塊相符且各含一完整148。補驗明示三圖參數，驗同次來源SHA、單位與視圖、HD／原圖／HD；十二項反例及舊預設正對照通過。原取景不相符仍逐筆保留，預設不放寬；沒有重錄或改原PNG／收據。
- 補驗初稿把原圖視窗誤要求1920×1200，依實檔及視窗audit修為1280×800；只有主題差異摘要採最近鄰同尺寸，HD完整圖塊仍原尺寸核SHA。GUI536.904秒、峰值4687286272 bytes，6GiB／2CPU無OOM／逾時；不推論平台餘裕。F9來源回載不宣稱本批自然返回或戰後變更保存，原版對拍與十一章E2不提升。
- 依使用者直接刪除過期停止線：移除56／57／58中首章保存仍待接受、第二章尚未接受差異的舊限制，以及蓋亞新版普通操作仍未驗的說法。61刪除HD世界／原生座標／前景尚未啟用及#259仍待串接的舊停止線，保留來源、原始失敗與幾何契約。傷害後續調整仍由#278追蹤，本批數值未改。
- README改採同輪完整實際視窗，未裁切、去字或補畫；原四語與文化保存入口保持。主收據為grass_earth_edges_followup_20261025，56／57／58／61及五條父工單同步。目前優先補平地底材，再道路／水岸；HD2、完整介面／工具列及全部HD後集中人工／平台仍待，總項維持開啟。
- 核對GUI收據的執行檔SHA後刪除可重建binary，釋放20618216 bytes。本批未錄影，必要原始PNG、母稿與候選、來源、失敗及記憶體收據保留。沒有新增Docker image／volume，其他專案資源未動。最終來源／連結／擁有權、工單與Docker檢查結果於下段追加。

- 提交前31份來源／收據綁定、9份Python語法、2832個本地目標、七項舊斷言刪除、清冊與本批擁有權通過；README文化保存與原版四語入口保持。新增教訓守衛初稿漏跳脫正則括號，修正後349教訓／210守衛通過。官方工單快照25條、18條要人判、可能已完成0條，五個HD父工單原驗收保持。FD2 Pillow／Go／GUI容器清單均為空，私人庫已同步且乾淨。公開提交與遠端狀態由Git紀錄回查。

### 2026-10-25：十八圖平地底材與三種補接交界HD

- 上輪完成十四交界，屬實質進展。從982839c4主線建立hd/ground-bases-20261025，按現況58選剩餘原生平地52／53及160／164／165；52與244內容相同。五份原生RGBA先目視，再將十八圖52個實際selector槽及三層SHA寫入底材配方，61 READY先於編製。沿既有草／土母稿及柔化，不新增玩法語意、母稿或正式Go。
- 共用編製／登錄器及GPU測具增加明示底材模式，固定五來源／52槽／十八圖；既有模式保持。五張新PNG、十八稀疏集合登錄，原793項保持；正式811清冊、257地形槽。103680個合成像素、52純草／53純土全圖一致、十來源反例、來源引用原子拒收及五登錄拒收通過，兩舊模式的來源／登錄正反例亦通過。
- 修正三項本輪問題：新模式的maps計數遮蔽路徑參數，改map_count後重建；清冊驗證器要求每幀不同檔名，與Go按frame唯一及同源52／244共圖的實際契約不符，改驗同檔SHA一致，仍拒收重複frame／錯SHA／危險路徑；本輪source_refs誤加不支援的#片段，改回正式檔案路徑，編製器在原子發布前新增完整清冊預檢及實際拒收反例。原配方重建後SHA核對保存，不靠放寬來源檢查通過。
- 第一份GPU及普通GUI在像素上已通過，但屬修正前中繼資料，未用來宣稱修正清冊完成。原GUI已自行PASS／退出，停止要求返回無容器，不標成遊戲故障。以修正候選重新取得108來源GPU世界與一個原樣Game.Draw／F2，十八錯遮罩、全部五圖源及獨立CPU通過；GPU137.896秒、峰值4151083008 bytes，無OOM／逾時。完整十四項清冊測試含私人實檔與同圖共用正反例通過。
- 修正清冊另跑正常標題LOAD、方向鍵取景(6,23)、F2、狀態／命令面板、兩主題保存及新程序LOAD。十九張PNG，兩地形PNG各54塊完整相符，52與53均完整可比，不需補取景。GUI258.182秒、峰值4116852736 bytes，6GiB／2CPU無OOM／逾時，不推論平台餘裕；本批沒有抽驗攻擊、自然返回或戰後變更保存，無新原版parity／十八章E2。
- 五新圖及清冊按白名單同步唯一私人庫，6974檔核SHA，原6969檔bytes及mtime保持。重同步檢查器初稿誤將本批五新圖複製的mtime也列為舊檔，修正驗證範圍後原6969檔保持要求通過；未放寬舊素材保留。私人提交86e59b00c81d68f4575564c0c7fd584716eabe1c已推送，真正遠端相同且仍為private。
- README改採本輪完整原始普通視窗，不裁切、去字或補畫；56／57／58／61及五父工單同步，完整HD總項不關閉。主收據為ground_bases_followup_20261025；下一步仍須其他地形／水岸、HD2、完整介面／工具列及全部HD後集中正常／人工／平台。已驗絕招不重開。
- 核對SHA後移除可重建binary、兩份已被正式候選取代的相同PNG／preview及13938個舊候選symlink，移除21378822個實檔bytes。保留兩舊候選的catalog／manifest、原始錯配方、全部GPU／GUI／失敗及記憶體紀錄，相同PNG由正式r3候選及私人庫保存。本批未錄影、未建重複完整包或新Docker image／volume。最終工單、來源、連結、擁有權及容器收尾於下段追加。

- 最終35份來源／收據綁定、七份Python語法、2838個本地目標、793舊清冊項目保持及本輪擁有權通過，無root產物或.md目錄。官方快照25條、18條要人判、可能已完成0條，350教訓／211守衛通過。Pillow／Go／GUI容器清單皆空，私人庫乾淨且真正遠端已核對；README文化保存與原版四語圖保持。公開實質提交與遠端狀態由Git紀錄回查。

### 2026-10-25：水面、沙地與首章草沙／沙水交界HD

- 上輪完成十八圖底材，屬實質進展。從707fa898主線建立hd/shore-20261025，沿原生來源匯出與實際selector，先目視首章49／50、64～79及96～119中十五實際沙水來源，共33種、九圖41槽。首章control第一byte為零，不新增換幀／波浪時鐘；原生selector、palette／LUT與規則保持。
- 使用內建圖像工具各新繪水面、沙地母稿，完整提示及原始輸出路徑寫主收據，兩張1254正方形不透明原稿保存專案。草沿既有母稿；三材質索引只作本來源藝術分類，沙內灰褐小點併入新材質、半徑2柔化及八像素邊帶均屬明示近似。61 READY先於編製，沒有改正式Go、原岸線、角色位置、RNG或保存格式。
- 共用編製／登錄器及測具新增明示水岸模式，不新增正式資產格式。原生33bank／8256幀完整來源預檢及候選清冊預檢通過；兩母稿、33成品與九稀疏集合登錄，原811項保持。正式清冊822、地形298槽。684288個新RGBA像素、純水／沙全圖、相同材質端點、十來源及來源引用原子拒收、五登錄拒收通過。
- 首輪受控場景沿舊(6,15)鏡頭，缺49／50；第二輪沿前批普通南岸(0,16)，有49但沒有純沙50。兩次後回讀戰術幾何路由，查實際map0：50位於北岸(1..5,3..6)，南岸僅有混合交界。以資料證明同時含49與50的合法(0,0)視圖重跑，沒有放寬像素條件或改遊戲。第三輪54來源GPU世界、一原樣Game.Draw／F2及九錯遮罩通過，115.58秒、峰值3841740800 bytes，無OOM／逾時。新增取景來源規則，失敗與記憶體紀錄保留。
- 舊底材與十四交界兩模式完整重編，清冊、PNG與preview逐byte等於既有已驗候選，原來源／登錄正反例通過；純草模式另跑最小正反例。十四項清冊測試含完整私人實檔通過。另目視三組不同的藍色來源，包含平藍形態，保留map7／26～29索引及來源圖，不只憑色盤猜用途或併入本配方。
- 正常首章標題LOAD，以方向鍵至(6,0)，F2、狀態／命令面板、兩主題來源保存及新程序LOAD通過。兩張完整PNG各75／78塊相符，純水49及純沙50均完整可驗，不使用補取景。GUI290.157秒、峰值4013977600 bytes，6GiB／2CPU無OOM／逾時；本批未抽攻擊／自然返回或戰後變更保存，不增加原版RGB、九章E2或平台餘裕聲明。
- 35份新PNG依白名單同步唯一私人庫，7009檔核SHA，舊6974檔bytes及mtime保持。私人提交ffce8d9e969e41460551eb0243b7f0f521c39c66已推送並核對真正遠端，保持private。README採本輪完整F2回HD視窗，不裁切、去字或補畫；56／57／58／61與五父工單同步。主收據為shore_followup_20261025，其他地形、HD2、完整介面／工具列與全部HD後集中人工／平台仍待。
- 核對SHA後刪除可重建binary，35份候選重複圖改成同內容私人別名，釋放26865252個實檔bytes；兩個空的失敗GPU輸出目錄移除。原始母稿與提示、正常PNG、候選清冊與全部收據／失敗／記憶體紀錄保留，未錄影、未建重複完整包或新Docker image／volume。逐幀audit合計約5.6MB，屬小型必要來源紀錄，依比例原則保留。最終工單、連結、來源與Docker收尾於下段追加。

- 最終43份來源／收據綁定、十份Python語法、2844個本地目標、811舊清冊項目保持、35個同SHA別名與本批擁有權通過，無root產物或.md目錄。官方快照25條、18條要人判、可能已完成0條，351教訓／211守衛通過。三份實際Docker image ID已記錄，Pillow／Go／GUI容器清單皆空；私人庫乾淨且真正遠端已核對。README文化保存與原版四語入口保持，公開實質提交與遠端狀態由Git紀錄回查。

## 2026-10-25 首章陰影、土脊與石路HD

從56f550e1的乾淨main建立hd/map0-terrain-finish-20261025。路由載入重製技能、spec-gated workflow與文件職責；普通取景連續遇環境問題後另查試玩路由。原生三層來源依既有native-terrain-sources-r2清冊與固定FDSHAP／palette綁定，不重開原版RE。先寫READY配方與61，再擴充既有材質編製、登錄、反例及來源GPU測具。35首章槽去除37／38相同內容後為34來源，實際覆蓋十四圖139槽。沿已登錄草／土與內建image_gen新灰板岩母稿，以藝術分類及明示暗部合成144圖；原版RGB不混入，alpha沿原opaque六倍。遮罩、羽化、暗部與材質為美術近似。正式Go、規則、位置、RNG及保存格式保持。

84來源GPU世界與一原樣Game.Draw／F2、十四錯遮罩、十來源拒收、兩暗部／材質拒收及五登錄拒收通過；705024合成像素與完整重編相同，舊底材／水岸清冊、PNG與preview重編逐byte相同。完整私人包十四項清冊測試通過。GUI r4從普通標題LOAD與方向鍵至(17,21)，F2、狀態／命令、來源保存與新程序LOAD通過；19張PNG、77操作，兩完整地形PNG匹配88／91塊，51與120均可驗且無補取景。本批未抽攻擊、自然戰後返回或新增PLAYER-E2。GPU217.824秒／4232544256 bytes；GUI422.223秒／4982108160 bytes，6GiB／2CPU下無OOM／逾時，不推論平台餘裕。

失敗與勘誤：新增模式首輪覆寫masters局部變數，於候選發布前TypeError拒收，修正後重編；正式前再限制bicubic暗部下界，舊草稿不登錄。私人同步先誤將7009清冊檔當PNG數，實為7007PNG與兩JSON；後續同步舊工具只跳過samefile，部分普通副本以相同內容重寫mtime。Git基底逐SHA確認舊7009檔bytes全保持；補SHA相同普通副本跳過及回歸後7044檔重跑bytes／mtime保持，初次mtime不冒稱保持。GUI r1錯估焦點對應鏡頭，把y15來源120移出視窗，據audit改合法焦點；r2在180秒啟動搜尋內未見視窗，無退出／OOM，完整載入較慢僅列強推論；r3的360秒值被舊argparse拒收，新增明示360選項後r4通過，預設與舊選項不變。所有失敗日誌保留，未降像素或保存門檻。教訓守衛初指向被掃描器略過的測試檔，已改綁正式同步工具；352教訓／212守衛通過。

正式清冊837項、稀疏地形437槽，舊822項保持。新35PNG已保存唯一私人庫f3c5bdc49241633be082a5d3bb9beba04a63de13，遠端main相符且仍private；原版固定包未改。README保留北岸與全部舊圖，新增本輪完整未裁切GUI；修正舊「現行路徑沿原生放大」敘述。母稿、完整提示、來源／輸出SHA、命令、記憶體與全部收據集中docs/data/fd2-modern-hd-battlefield-20261019.json的relief_followup_20261025；61已為CONFORMED-SCOPED／RUNTIME-E1／NORMAL-GUI。#4／201／202／204／263已更新並保持OPEN，重新拉取／render／verify；25未完成項及人工閘門保持。首章底圖尚餘248／250兩寶箱及274／275兩小石塊，前景無缺；其他地形、HD2、完整介面／工具列及集中人工／平台仍待。

清理核SHA後刪未登錄草稿重複pack、候選35PNG改既有私人包別名、刪驗證執行檔，共31105904 regular bytes；母稿、接觸表、PNG、原始資料及收據保持，沒有錄影。全部一次性Docker均--rm，批次收尾確認三個FD2工具image無殘留容器，無新增image／volume或全域清理。公開批次通過最終來源、連結、清冊、擁有權與差異檢查後提交並推送main，精確HEAD以Git為準。

## 2026-10-25 首章寶箱開合與小石塊HD

- 從c3ff379c乾淨main建立hd/map0-props-20261025。沿已載入重製技能、spec-gated workflow與文件職責，先寫READY，再編製與登錄。原先四原始槽之外，既有0x12263／CompleteTreasureMapUpdate還會產生249／251開箱；同源map14:180～183一起納入，共六來源、兩圖十槽。原始selector不當成事件後覆蓋證據，不重開原版RE。
- 內建image_gen產生金／銀箱四格與小物件兩格透明母稿。原RGBA保留，>=250高覆蓋主體完整裁切，依原bbox六倍與既有草地合成；陰影、裁切、比例與新RGB屬美術近似。alpha沿原opaque六倍，正式Go、規則、事件、座標、RNG及保存格式保持。來源、提示、SHA、配方與重跑CLI集中docs/data/fd2-modern-hd-battlefield-20261019.json的props_followup_20261025。
- 十二來源GPU世界與閉／開兩完整Game.Draw／F2、原事件二次更新不再跳圖、單位與RNG保持通過。十來源／七登錄拒收、124416合成像素、完整重編一致、舊起伏地形登錄相容與十四項私人實檔清冊檢查通過。GPU99.320秒／4788817920 bytes；正常GUI357.237秒／5440737280 bytes，6GiB／2CPU下無OOM／逾時，不推論平台餘裕。普通標題LOAD、焦點(6,9)、F2、面板及來源保存／新程序LOAD通過；兩PNG各93／96完整格相符，四原始物件每張可驗，無補取景。開箱呈現為受控事件狀態，正常取寶／戰後返回與PLAYER-E2不由此新增。
- 反例測具首輪GPU檔名catalog.json撞上自己的正式清冊樣本，產生假失敗；改gpu-前綴後原候選／GPU收據通過，未改正式路徑。像素驗證器原以props前綴套舊九物件，與明示四來源衝突；改按terrain_expectation，無明示清單的舊props才保留九物件守衛。同PNG通過，十五新舊／補驗守衛均通過。收尾蒐證腳本漏tools匯入路徑，中止後只續寫未完成metadata，沒有重錄或重做已完成清理。
- 正式清冊841項、稀疏地形447槽，舊837項保持。八新PNG已推送私人庫377197837c6566d553ca2895335becba1067b482，遠端main相同且仍private；原7044檔bytes／mtime保持。README保留全部舊圖與文化入口，新增完整普通GUI。58／56／57／61及#4／201／202／204／263已同步；工單維持開啟，快照重新拉取／render／verify。354教訓／214守衛通過。
- 核SHA後將八份候選素材改私人包別名，刪驗證執行檔，共23196075 regular bytes。母稿、原始資料、預覽、PNG及收據保留，無錄影。Docker工作均--rm、UID1000及有界，批次收尾沒有殘留FD2容器，沒有新增image／volume或全域清理。公開批次在最終差異與來源檢查通過後提交並推送main，精確HEAD以Git為準。其他地形、HD2、完整HUD／介面與工具列、普通取寶及集中正常／人工／平台仍待，完整HD目標保持。

## 2026-10-25 清除與目前定案衝突的HD停止線

- 依使用者明確指示清理現行文件，命中知識路由的舊結論回填入口。56／57／58／61共22處刪除或改正：普通法術32／33及正式效果美術、召喚地圖尾段、首章四種物件、水岸、滿背包問句與玩家回復已由後續切片接上；浮出主題／語系列已驗，剩餘項改為工具列其餘控制。原始來源、既有主收據及時間序列工作紀錄保留。
- 刪除逐幀／逐音訊一致的硬性完成門檻。遠端#12改驗代表性正常玩家與敵方回合的流程、演出順序、結算、返回及音訊抽樣；已接HD且有目前原版流程證據的絕招沿用已驗範圍。傷害公式後續擴充仍依#278，不加入本輪HD閘門。
- 透過真正主機gh更新並重新取得遠端Issue，再由Docker內官方build_snapshot／write_snapshot及render更新快照。25開啟項、工作清單verify及354條教訓的214項守衛通過；#12仍開啟，集中人工確認仍依全部HD串接後排程。只修文件與驗收條件，未修改正式規則、圖源、程式或保存格式，沒有重跑遊戲或新增parity聲明。
- 逐項核對目前主收據及文件差異，git diff --check通過。Docker均有界、--rm、UID1000及network none，收尾核對FD2容器與檔案擁有權。無新增映像、volume、錄影或執行檔。
- 全庫擁有權掃描在45秒上限停止；改查本輪異動目錄的直接項目，找到docs/data下八個既有root-owned資料檔，均不在本輪異動範圍，未修權或改寫。未發現誤建.md目錄；七個異動檔案均為UID1000。

## 2026-10-26 戰鬥面板HD框線與HP／MP條

- 從d5f149d8建立feat/hd-battle-panel-chrome-20261025，沿介面路由、重製技能及使用者「全部HD先接上，再集中人工確認」排程。先寫READY，再接共用149×42面板的894×252直接繪製；原型只作清晰藍框構圖參考，正式取同次原色盤，不用固定RGB。三像素外框、兩像素分隔線與一像素bar框屬HD呈現近似；不改原生原點、字區、層序、捲動、規則、音效、HP／MP／RNG或保存格式。
- 血條保留既有整數量化及左右端帽佔位；八組空／滿／部分／極小／負值／超上限／零或負上限，以實際indexed輸出驗1648個橫向採樣位置，不用複製公式自證。正式實作補上早期原型未計入的右端帽佔位。原生溢位marker仍從同次凍結面板保留，字型沿既有Noto與四語字區；來源／色盤凍結、拒收及原生fontless路徑保持。
- 十六指令來源入口、24來源GPU畫布、31份Go閘門／36頂層測試與十二完整Game.Draw流程通過，零SKIP／失敗／逾時／OOM。涵蓋雙側物理／捲動、普通0／6／敵方9、分段24／29／音速刃30與召喚32／34，四語、F2與一次返回保持。八指令流程849次面板回讀，框／槽／條6316223像素與字形260463像素相符。
- 24／29／30共223個最終1920畫布對獨立Pillow合成通過，完整畫面容差1；獨立框／條核對分別12／14／14個面板與色盤組，5113440幾何像素通過。十五反例涵蓋錯框／條／凍結HP、透明度、尺寸、SHA、路徑、缺保存／自然返回旗標與正確SHA的粗框；另以舊像素面板拒收為負對照。沒有遮蔽最終畫面差異，未重跑原版執行器或新增原版RGB／PLAYER-E2。
- 正常GUI沿既有首章JSON來源槽，由標題LOAD、鍵盤移動／攻擊、F2、自然返回與保存／新程序LOAD通過；19主要PNG及兩面板72未遮擋幾何點。F5維持章節邊界保存語意，不保存戰中位置、Acted與敵方HP，不宣稱跨章通關。295.292秒／4775550976 bytes，無OOM／逾時。Go流程共用容器的累計峰值達6GiB限額並有快取回收，非單一測試獨占峰值，不推論平台餘裕。
- 58／56／57／61及README同步，新增完整未裁切的普通攻擊PNG，原139張圖及文化保存／原版多語入口保留。舊文字切片改指向現行框／條；獨立介面選項仍待實作，過期「等看圖確認才繼續」改依全部HD後人工排程。主收據battle_panel_chrome_followup_20261026綁定來源、工具鏈、命令、SHA與所有收據，素材清冊及私人庫保持。
- 彙整腳本先誤計套件層pass，再誤寫來源檔名；兩次均於登錄前停止。改為只計具名測試，並從實際consumer定位native_item_panel_data_render.go後，沿原收據完成登錄，未重跑遊戲。核SHA刪20618256 bytes驗證執行檔，錄影未產生；來源槽、PNG與證據保留。異動檔及目錄UID1000，窄範圍掃描的既有root-owned檔未改寫或修權。Docker工作均有界、--rm、network none與UID1000，收尾無殘留FD2容器，未新增image／volume。
- 收尾的hd-text-clear-native-shadow-footprint守衛仍硬綁舊erase矩形，與整張底板先重繪的現行程式衝突。已刪除舊pattern，改綁實際p.chrome.paint入口，保留陰影／溢位與獨立像素驗證；原有教訓形成原因以追加說明保留。早期樣式比較測具的左側標籤也改為現行HD，避免重跑時把新框線誤標為像素。
- 更新後354條教訓／214項守衛通過。比較原型R3另跑一次具名Go測試通過，零SKIP／OOM；歷史R2與測具原SHA由d5f149d8重新核對保留，不改寫舊圖。這次只修標籤，不改正式繪製。私人庫本機與遠端main均為377197837c6566d553ca2895335becba1067b482，工作樹乾淨。
- 最終來源／收據140項SHA、2778個本機連結、140張README圖片與文化入口、三個Python語法檢查通過；正式清冊與來源槽SHA保持，主證據全數仍在，驗證執行檔已清理。GitHub #4／201／202／204／263已同步並保持OPEN，官方快照／render／verify通過，25項中沒有可直接關閉者。整體HD目標保持進行。\n- 最終證據檔仍在寫入時，一次git diff --check退出135（SIGBUS）；等寫入程序完成後，同命令乾淨重跑退出0。讀寫重疊為疑因，不記為遊戲產品缺陷；後續文件寫入與Git檢查改為依序執行。

## 2026-10-26 使用者指定的空間清理

- 先處理空間，暫不新增HD實作。依交付目錄契約保留dist-all；原版資料、私人素材庫、來源存檔與獨有證據保持。所有分析與清理在UID1000、無網路、有資源限制的一次性Docker容器執行。
- 刪除兩處可重建Go編譯快取，釋放6743859200配置bytes。清除舊完整版打包暫存、AppDir與平台中間包，釋放2251898880配置bytes；保留appimagetool與linuxdeploy工具。正式交付98檔逐檔重新核對大小與SHA-256，全部保持。
- 正式現代素材7052份重複實檔逐檔核對私人清冊與實際SHA後，改為既有私人包的符號連結。私人庫唯讀，工作樹乾淨，HEAD保持377197837c6566d553ca2895335becba1067b482。正式清冊841項的--verify-private通過。這些別名不是獨立驗證輸入包，需要獨立實檔的驗證仍須重新同步。
- 歸檔輸出PNG核對673433檔，只有內容SHA相同的548909檔改用硬連結，減少11202146304配置bytes；所有路徑與內容保留，獨有截圖保持。素材輸入包不參與硬連結。共用inode可能改變mtime，重新擷取須使用全新輸出目錄，避免覆寫舊證據。完成後另抽查275組的SHA、inode、大小與擁有權，全部通過。
- 四項清理的配置容量合計約22.2GB，實際淨釋放須扣除稽核紀錄與符號連結。work從73026330624降至55360491520 bytes，工作區合計61313396736 bytes，檔案系統可用187308851200 bytes。容量依同次du -sx -B1計算，共用inode只計一次。
- 清單與摘要集中既有work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-final-r1.json及其引用的四份清理紀錄。548909筆PNG稽核壓縮檔SHA-256為3e0289c297a88a0caac393ae4d5dd1f3f42a11f8216ff6e745c73d1ef971447b。正式清冊、README面板圖片與來源保存槽SHA保持。本輪沒有重跑遊戲或新增原版一致聲明。
- 第一輪掃描大量符號連結耗時，確認原程序仍活著且尚未刪除後停止，改用scandir略過連結續做。一次診斷誤把python3重複傳入既有entrypoint，未執行清理；明示entrypoint後同項檢查通過。未重複啟動去重程序。
- 沒有找到過期FD2映像或volume，保留現用工具鏈，不清理其他專案資源。一次性清理容器已退出並移除。完整HD目標保持進行，本輪不改正式程式、規則或保存格式。

## 2026-10-26 狀態到期提示HD與四語

- 從07c1f6d7的main建立feat/hd-transient-expiry-20261026，保留前輪空間清理紀錄。沿復古重製、規格閘門與文件職責路由，先在61寫READY，再接現行sub_1A866到期owner。原版證據沿固定EXE與既有IDA9.4匯出，不重開已閉合RE。
- 到期job封存聚焦後的typed input、work、VGA及色盤；HD世界、480頭像與文字沿同次快照。共用class UI只為此具型別job放行，其他transient頭像守衛保持。交易前私有預檢新圖源，F2完整預備後才發布。原生倒數、扣血／死亡／到期順序、Draw確認、恢復與一次續行保持，保存格式及正式規則不改。
- 六則FDTXT0x1E1..0x1E6加入既有四語pack與來源契約；繁中保留字模原點，其他語系使用原(99,127)文字區，205寬、最多三行、行高19。翻譯只供顯示，不參與counter或保存鍵。精簡JSON格式後逐包語意相同，原有內容保持。
- 四語完整Game.Draw共288拍、36次代表階段F2、三種封存來源變造拒收及一次恢復通過。原生框覆寫768000點、框外HD509276160像素、頭像4147200點及文字實際墨跡577438像素核對；36張1920×1200 PNG逐RGBA雜湊獨立驗證，十項收據反例拒收。GPU207.025秒、峰值6363865088 bytes，6GiB／2CPU下memory.events的OOM與kill皆零；接近上限，不推論平台餘裕。
- 最終選定Go通過70筆具名測試：cmd/fd2 27、localization 20、battle／campaign 22與獨立GPU程序1；沒有SKIP或失敗，包含原有狀態／死亡、原生字框、四語套件與保存原子寫入／冷讀。初始seed7、selector2、六counter=1、HP100/100、BaseAP100／BaseDP60／DX20明示於主收據；實際扣血至90後播六提示。DATO9與地圖位置沿受控fixture，不宣稱正常角色配置、一般取得狀態、PLAYER-E2或原版逐幀一致。
- 首輪原生測試漏掛FD2_ASSET_PACK，分類為環境；補正式私人原包後重跑通過。首輪GPU的typed槽缺EquipmentBaseSet，補明示基礎值後先跑完整原生正對照，反例改核對modern HD expiry的錯因。回填既有negative-tests-require-positive-control-and-reason教訓，未放寬正式來源守衛。
- 第二輪GPU逐拍F2重做相同主題預備，確認程序仍活著、6分25秒後明確停止，stop後退出137，不記為遊戲signal killed或OOM。第三輪維持288張完整畫布核對，F2改每個提示完整框及首提示開／收／restore代表階段，共36次通過。全部命令與失敗日誌保留於既有ordinary-effects-r1的expiry-commands-r1.json。
- 58／56／57／61同步到CONFORMED-SCOPED／RUNTIME-E1，刪兩處已過期的死亡待補及精確tick／逐音訊停止線。README新增四語完整Game.Draw半縮總覽，原140張圖與文化入口保持。收尾文化位址檢查誤用小寫0x36c9e，核對原文0x36C9E後續做剩餘檢查，沒有改寫文化內容或重做清理。
- 來源16項、2896個本地連結、141張README圖片及文化入口通過；正式清冊、EXE、來源保存槽、原包manifest與私人庫保持。私人本機／遠端main均為377197837c6566d553ca2895335becba1067b482且工作樹乾淨，無新美術或私人提交需求。永久入口為tools/verify_fd2_hd_transient_expiry.py、61與HD主收據transient_expiry_hd_followup_20261026。
- GitHub #4／201／202／204／263已讀回核對並保持OPEN，官方解析器重建快照及render／verify通過；25項中18項待人判、0項機器訊號消失。354教訓／214守衛通過。全部HD先串接後集中人工的排程保持；其他地形、HD2、完整介面、獨立切換及工具列／加速／海報仍待。
- 一份中止輪的PNG與新PNG逐SHA相同，改相對別名保留原路徑，減少1753088配置bytes；其他不同內容的診斷PNG保留。沒有錄影或留存驗證執行檔。Docker均--rm、init、UID1000、network none及有界，收尾確認沒有殘留FD2容器，未新建image／volume或清理其他專案。


## 2026-10-26 再次清理過期圖片與測試資料

- 依使用者指示優先清理空間，未新增HD實作。沿復古重製技能、驗證交接與文件職責路由；保留未提交的HD程式與文件、原版來源、正式交付、存檔、現代素材清冊及目前收據。
- 完成13900份文件與收據的路徑盤點，另以受版控全文逐字排除被引用目錄。僅處理104個未引用的舊候選及反例目錄中的生成圖片，刪1033份；JSON、文字收據與腳本保留。刪2013份可重建Go快取。兩次唯讀深入掃描達時間上限，均未刪檔；實際刪除沿已完成的目錄彙總與引用盤點，不重啟未完成的掃描。
- 243個測試pack目錄中，6675份與私人清冊及實檔SHA相同的素材改為私人包符號連結。不同內容的素材保持。此類歸檔輸入已是別名，需獨立輸入的重新驗證應以同步工具重建實檔，並用新輸出目錄，不能宣稱別名是獨立驗證包。
- 刪除及替換按最後一個實體連結計數，合計釋放4425678848配置bytes，淨減少量與收尾容量以[清理摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r2.json)為準。該檔索引逐檔SHA稽核壓縮檔、受保護檔案清冊與本輪清理腳本，皆沿用既有工作目錄。
- 3741份受保護檔案逐大小與SHA複查通過，含正式交付及目前HD畫面與收據；6675個別名及4762份私人目標、3046個刪除路徑複查通過。私人庫工作樹乾淨，未寫入私人素材或原版資料。這是清理驗證，沒有重跑遊戲或新增一致性聲明。
- Docker均為有界、--rm、無網路與UID1000的一次性容器。沒有可確認過期的FD2映像、容器或具名volume，保留現用工具鏈與其他專案資源。完整HD目標與既有未提交工作保持，後續仍須完成串接及集中人工確認。


## 2026-10-26 戰鬥提示框HD細框

- 從45b64235的main建立feat/hd-battle-dialogue-chrome-20261026。沿現行到期與戰後物品／金錢成功訊息，將六倍放大的18像素原框改為1920×1200目標下三像素HD細框。四邊、四角與底色取同次FDOTHER#5來源及色盤，不加固定RGB；內部頭像、文字原點、開收框、恢復、交易與一次續行保持。這是明示美術近似，不新增原版逐像素聲明。
- READY規格先於實作。來源形狀、單色底、色帶與不透明色盤先凍結，HD在倒數／HP交易或F2發布前拒收壞來源；原生路徑保持。框區先裁切再沿contentGeometry定位，幫助、狀態頁、黑白與淡出保持遮擋。沒有新圖包、保存欄位、規則或平衡變動。
- 32筆具名Go驗證通過，零SKIP／失敗；當前11份實作及測具SHA與原通過收據一致。六到期訊息四語288次完整Game.Draw／36次F2，物品與金錢26次／26次F2，共314次完整畫面及62次切換。另有八種最終繪製層情況，該測具不冒稱完整Game.Draw或正常玩家。
- Go核對13646880個框線及舊邊帶清除點；46張完整PNG以獨立Pillow核對2834784個框線RGB點。22項收據反例及實際舊18像素原框拒收通過；舊到期收據預設仍相容，require-chrome明示要求新框。來源與完整命令由HD主收據battle_dialogue_chrome_followup_20261026定位，不重錄或重跑已通過的GPU。
- 到期GPU峰值6442450944 bytes觸及6GiB限制，memory.events.max=429、oom=0、oom_kill=0；獎勵峰值5905375232 bytes，max／oom／kill皆零。不由此推論平台餘裕。沿seed7與既有typed受控狀態／獎勵槽，DATO9與地圖位置如實標示；沒有新增正常取得狀態、原版對拍或PLAYER-E2。
- README保存四語完整遊戲畫面總覽，142張圖片與文化入口保留。56／57／58／61同步，GitHub #4／201／202／204／263已讀回核對且保持OPEN。提交前再由主機gh取得遠端工單，Docker內官方解析器重建快照及render／verify，不用本地快照代替遠端。
- JSON跨語言嵌入及JavaScript替換字串兩次產生命令失敗，修正後同輸入通過，分類為驗證腳本問題；追加可重用教訓script-generation-keeps-language-boundaries。完整錯誤及修正命令仍在dialogue-chrome-commands-r1.json。
- 一份總覽逐SHA相同後改指向公開圖的相對別名，減少1269760配置bytes；46份獨有完整PNG與兩張最終層圖保留。未產生錄影或留存驗證執行檔。之後依使用者要求完成另一批空間清理，記於上一節，當前程式與收據3741份保護檢查通過。
- 正式清冊841項及447稀疏地形槽、來源保存、原包manifest與私人庫保持。私人庫本機工作樹乾淨，HEAD為377197837c6566d553ca2895335becba1067b482；無新素材同步需求。Docker均有界、--rm、UID1000與network none，收尾無殘留FD2容器，保留現用工具鏈。重大且驗證成功的本批提交／推送main，精確HEAD以Git為準。
- 完整HD仍持續：其餘地形、HD2、完整操作介面、主題／介面獨立切換、工具列字級／加速／海報及全部串接後集中正常／人工／平台確認仍待。本批限定CONFORMED-SCOPED／RUNTIME-E1，不關閉完整HD總項。


## 2026-10-26 滿背包獎勵對話HD細框

- 從0590f27d建立feat/hd-full-reward-dialogue-chrome-20261026，先寫READY規格，再將既有滿背包問句、放棄及選擇器取消owner接共用三像素細框。來源由實際整備對話格與class色盤凍結；HD來源不符在佇列移出與互動發布前拒收，原生正對照保持。不改物品替換、裝備旗標、輸入、RNG或保存格式，無新素材及RE。
- 四來源／問句及八既有對話／原生交易回歸通過；完整GPU跑五分支291次Game.Draw、40個F2邊界及五次續行，共13具名Go通過，沒有SKIP／失敗。全部開收與restore核對16601760框線及清除點，物品面板／返回無殘留，Draw未確認不推進。直接唯讀使用私人6062份獨立runtime檔，未再複製圖包。
- 十張完整PNG獨立核對421200個框線RGB點，九收據反例及先前粗框真實PNG拒收。畫面上的索爾像素感經回查確為已登錄480圖源本身；五份對話PNG共1152000個頭像RGBA點完全相同，不把尺寸或掛勾存在當作新重繪風格證據。最終頭像檢查已收進受版控工具的--portrait-pack，舊到期／成功訊息的收據路徑回歸保持。
- GPU348.386秒、峰值5696036864 bytes；單元203.237秒／4964741120 bytes，既有回歸23.107秒／681779200 bytes。8GiB GPU／單元與4GiB回歸、各2CPU，memory.events皆零，沒有OOM／逾時，不推論平台餘裕。沿既有map23、DATO0、八槽16..23、物品211／212、seed7及clock1000受控建構，不新增原版、正常玩家或冷讀GUI聲明。
- 完整來源與命令由HD主收據full_reward_chrome_followup_20261026定位。Go來源與實測快照相同；後續只補獨立Python頭像檢查並用相同PNG驗證，未為此重跑GPU。首次規格命令因Markdown反引號令JavaScript在執行前解析失敗，沒有寫入；格式化首用登入shell重設PATH，改非登入shell後沿原映像通過。兩者皆為工具組態，沒有Go產品測試失敗。
- 56／57／58／61修正過期對話框線敘述，先前主收據保留來源並指向目前細框。README新增滿背包四場景總覽，舊圖與文化內容保留；YESNO、八格選擇器框線／圖示與城堡地形仍為像素。完整HD、HD2、獨立控制與工具列其餘功能，以及全部串接後集中正常／人工／平台仍待。
- 總覽逐SHA一致後改成公開圖的相對別名，釋放量依full-reward-chrome-cleanup-r1.json；十張獨有PNG及收據保留，無錄影與留存執行檔。保留本輪必要Go快取以免反覆重建。所有容器有界、--rm、UID1000與無網路，使用既有映像，沒有新image／volume或其他專案清理；私庫與正式清冊保持。


## 2026-10-26 滿背包四語HD確認選項

- 從484658fc建立feat/hd-full-reward-confirmation-20261026，沿復古重製與文件路由先寫READY，再接共用選項元件到滿背包owner。原格16／17為24×20、48／49／51／52為24×16，沿原生四步x=248±spread及y=168。原格內以同色盤清底、三像素金框及40HD像素Noto畫common.yes／common.no四語字串；文字安全區12像素，pulse以6HD像素下移，均為明示美術近似。
- 每格先合成文字與底圖再按左後右繪製，保留開收重疊遮擋，避免晚畫的左文字蓋過右格。六原格加入owner簽章；來源、色盤、字形與完整墨跡預檢後才發布，原生格／輸入／庫存／RNG／保存未改。兩源／幾何測試、四滿背包來源測試、八原生／交易回歸及一完整GPU，共15具名Go通過，零SKIP／失敗；十二組typed位置重新畫原格對實際native producer bytes相同。
- 五分支291主流程畫格、40主題切換邊界及五次續行保持。另控制UI localeID／choice／pulse驗48組完整Game.Draw，獨立CPU整圖合成110592000像素、16次額外F2往返與狀態保持通過；測具不推進原生job，完成後恢復控制值。此為受控RUNTIME-E1，不新增一般玩家、正常F4操作或原版oracle聲明。
- 十八張完整PNG獨立核對選項幾何78336點及實際墨跡11543點，四語標籤、字形置中／pulse、安全區與重疊保持；同次480頭像1152000點及對話細框421200點保持。十九收據反例、同穩定幾何／pulse0的舊原生YESNO實圖拒收通過。四語總覽只驗選項翻譯，問句沿既有受控來源，不宣稱整個原生確認流程已翻譯。
- GPU313.043秒、峰值8589934592 bytes觸及8GiB，memory.events.max=243、oom=0、oom_kill=0。48組內嵌畫面與16次額外往返的集中測試不推論一般平台餘裕。來源單元14.726秒／1116639232 bytes、滿背包單元72.699秒／3382808576 bytes、回歸13.539秒／839790592 bytes，這三組memory.events皆零。直接唯讀使用私人6062實檔，無新圖包、SAV格式或RE。
- 來源、完整命令與核對入口由HD主收據full_reward_confirmation_followup_20261026定位。Go及測具在四組Go測試間保持，後續只擴充獨立Pillow選項核對及總覽並沿同一PNG驗證。56／57／58／61與README勘誤滿背包YESNO舊限制，其他確認caller及選擇器仍待；舊圖、來源與文化入口保持。
- 總覽逐SHA相同後改成公開圖別名，配置容量釋放依confirmation-cleanup-r1.json，十八份獨有PNG及收據保持。未產生錄影或留存執行檔，保留必要Go快取。Docker均有界、--rm、UID1000及network none，沿既有映像；未新增image／volume或清理其他專案。完整HD、HD2、獨立UI／theme、工具列及全部串接後集中正常／人工／平台目標保持。


## 2026-10-26 優先清理重複圖片與診斷資料

- 依使用者要求先清理空間，保留尚未提交的四語HD確認選項工作。命中復古重製與文件職責路由，沿用既有工作區，沒有新增HD實作或重跑遊戲。
- 完成歸檔圖片與JSON輸出逐SHA盤點，合併241387份相同副本，包含134171張PNG、106984份JSON、230份JSONL與2張GIF。全部路徑與檔案內容保持；硬連結共用實體檔，歸檔mtime可能沿用保留檔。重新驗證使用全新輸出目錄，不就地改寫歸檔。
- 排除目前HD、正式交付、測試pack、玩家存檔、來源與現用快取，審查後另跳過6個輸入路徑。獨有圖片、逆向證據與原版素材保持。首份完整Python盤點達120秒上限，未刪檔；改以有界find／du及按inode統計完成。首次測試pack抽查漏掛私人別名目標，補唯讀掛載後同命令通過，屬盤點環境問題。
- 移除重複實體占用11692879872 bytes；扣除約15.7MB清冊與稽核資料後，專案實際配置由57945214976降為46268084224 bytes，淨減11677130752 bytes。數值以GNU du實體配置計算，不把硬連結的重複路徑重複加總。
- 收尾獨立重讀241387個路徑，確認與保留檔共用inode，再逐SHA核對77808個實體檔；3878份受保護檔案逐大小及SHA驗證通過，包含目前程式、HD畫面、保存槽與正式交付。沒有內容差異、擁有權異常或意外的.md目錄。
- [清理摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r3-final.json)索引逐檔壓縮稽核、去重計畫、保護清冊與[可重跑腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r3.py)。腳本依plan、apply、verify三階段執行，既有報告不覆寫。
- Docker均為有界、--rm、init、UID1000、network none的一次性容器。未找到可確認過期的FD2映像、容器或volume，保留現用工具鏈與其他專案資源，未執行全域prune。完整HD目標及現有分支未提交工作保持。


## 2026-10-26 四語HD確認選項收尾

- 清理後重新核對本批來源、15項頂層Go結果與NO／Escape兩個子測試、18張PNG及3878份保護清冊，沿既有通過收據，不重跑GPU。GitHub #4／201／202／204／263已更新並讀回，原驗收及OPEN狀態保持；官方解析器重建遠端快照並render／verify通過，25項中18項待人判、0項訊號消失。355教訓／214守衛通過。
- 私人素材庫本機與遠端main均為377197837c6566d553ca2895335becba1067b482，工作樹乾淨。沒有新素材或私人提交需求。來源、文化入口、README圖片與連結、存檔及原包保持的最終結果見[收尾核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/confirmation-final-checks-r1.json)。重大且驗證成功的本批同步main，精確提交與遠端一致性以Git收尾核對為準；完整HD仍持續。


## 2026-10-26 共用三區物品與狀態面板HD細框

- 從bb10dd9b的main建立feat/hd-item-panel-chrome-20261026。沿既有0x17EEF、0x17FC0與0x18409資料與consumer，先登錄READY，再將同來源三像素細框接到八槽物品、共用狀態及戰場狀態頁；保留三區原生來源／目的位移、裁切、文字、480頭像與交易。原生模式保持，沒有新美術、保存格式、平衡或RE。
- 先前以原生310×102複製區當成下框，首版Go與同假設像素檢查通過，目視仍見舊邊帶。回查固定原包resource.json與entry21 PNG確認310×99，分開99列美術與102列複製padding後修正。新獨立工具於48,1140拒收首版真實PNG；亦於31,42拒收更早原生粗框，保留錯圖及來源，不把自洽測具當作原圖形狀證據。
- 最終31項頂層Go通過：三來源／細框、十一既有介面、十五原生資料／布局及兩個獨立GPU程序，無SKIP／失敗。物品五分支291畫格、40個F2及五次續行；狀態／指令兩頁41畫格、9個F2，另四語確認48組／16次F2保持。直接唯讀私人6062份獨立runtime，不另複製測試包。
- 33張完整PNG獨立核對，Go框線8984952點、PNG框線2023920點；穩定物品畫面框外2115648個RGBA像素對前版完全相同。37個收據反例及兩張真實錯框拒收，舊對話細框、四語選項、同次頭像與原生控制保持。受控map23、selector0、DATO0、八槽16..23、seed7與clock1000明示，狀態指令9來自既有State行並置於Game清單索引9，不宣稱自然取得、正常鍵盤或PLAYER-E2。
- GPU295.819秒／7154675712 bytes，狀態98.887秒／4808806400 bytes；兩者各8GiB／2CPU。最終來源16.682秒／1172656128 bytes、介面回歸5.012秒／548790272 bytes、原生回歸3.007秒／255221760 bytes。五組memory.events皆零，沒有OOM或逾時，不推論平台餘裕。
- 狀態前三輪為測具診斷：先缺Game指令清單，再把ID9放進長度1陣列，最後以fmt比較結構；按實際book[commandID] consumer及既有JSON值快照修正，未改正式規則。收尾原生計數曾誤估十項及抄錯一個名稱，實際正規式選到十五項，依原始日誌核對後建閘門；不報為產品失敗。三項可重用規則寫入fd2-lessons.json。
- 來源、全部命令、記憶體、兩份最終收據與獨立核對均由HD主收據item_panel_chrome_followup_20261026索引；永久入口為tools/verify_fd2_reward_message_hd.py及tools/verify_fd2_hd_item_panel_chrome.py。Go來源快照與最終實檔一致，後續只擴充獨立圖片反例並沿原PNG重跑，未重跑GPU。
- README保存四場景完整畫面總覽，56／57／58／61同步已過期框線限制。23份逐SHA相同圖片改相對別名，釋放配置bytes依item-panel-chrome-cleanup-r1.json；不同內容的診斷圖保持。沒有錄影或留存驗證執行檔，現用Go快取保留，所有容器--rm、init、UID1000、network none及有界。
- 完整HD仍持續：其他確認caller、圖示、英數標籤／狀態條、其餘地形、HD2、獨立UI／theme、工具列字級／加速／海報，以及全部HD後集中正常／人工／平台確認仍待。沒有新Release或完整HD完成聲明。

## 2026-10-26 清理舊追蹤日誌與反例圖片

- 依使用者要求先清理空間，保留目前分支所有未提交的HD修改。載入復古重製技能與文件職責，沿用既有工作區。
- [清理計畫](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r4-plan.json)限定舊大型JSONL與反例測試圖片。JSONL先無損壓縮，再逐解壓SHA核對後移除展開檔；檔名對照及原內容SHA保存於[逐檔稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r4-audit.jsonl.gz)。往後讀取這批歸檔使用對照中的.gz路徑並先解壓，不把壓縮檔當作原JSONL直接讀取。
- [保護清冊](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r4-protected.json)包含受版控檔案、原版來源、正式交付、存檔與目前HD收據。保留原版畫面、正式素材、現用Go快取及文字測試結果。[清理腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r4.py)依plan、apply、verify執行，禁止覆寫既有報告。
- 刪1468張過期反例圖片，72份大型JSONL壓縮後逐解壓SHA驗證。4430份保護檔案逐大小與SHA複查通過。GNU du相同命令的專案配置從46567473152降至44051001344 bytes，扣稽核開銷後減少2516471808 bytes，約2.52GB；數值不把相同實體檔的硬連結路徑重複加總。完整結果見[清理摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r4-final.json)。
- 一次舊HD目錄的Python細部分類到90秒上限，未寫入或刪除；沿已完成的有界find、du與明確反例目錄建立逐檔計畫，不拿未完成掃描當刪除依據。沒有新增遊戲功能或重跑GPU，也未提交尚待收尾的HD修改。
- Docker使用既有工具映像，全部清理容器均--rm、init、無網路、UID1000及有界。沒有確認過期的FD2映像、容器或volume，保留現用工具鏈及其他專案資源。擁有權、意外.md目錄及殘留壓縮暫存檔核對皆無異常。

- 收尾讀回GitHub #4／201／202／204／263，五項維持OPEN；遠端25項由官方解析器重建，18項待人判、0項機器訊號消失。358教訓／216守衛通過。私人庫本機與遠端main均為377197837c6566d553ca2895335becba1067b482且工作樹乾淨，沒有新素材同步需求。31項Go及33張PNG沿既有收據重新核對來源與SHA，沒有重跑GPU；[最終核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-chrome-final-checks-r1.json)保存結果。原包清冊首用錯誤檔名，改依實檔manifest.json後同雜湊重跑，不是素材缺件。清理後目前HD證據保持；本批提交與推送後另核對本機／遠端HEAD。


## 2026-10-26 共用狀態面板固定標籤與血條HD

- 從b100468a的main建立feat/hd-item-panel-values-20261026。先寫61的READY，再沿已閉合0x17EEF／0x17FC0、typed plan與現行上框實圖接十個固定縮寫、兩斜線及HP／MP條。既有數字與溢位符號保持；本批不包含圖示、物品列加成縮寫或完整介面。沒有新素材、平衡、保存格式或原版RE。
- 上框223×86補繪只覆蓋量測安全區。Noto沿既有有界48到24目標像素，十標籤／兩斜線取各自原圖最亮非底色；條長沿signed原生規則，只裁切呈現。record、原框、raw23..30與色盤封存，F2先全部預備後發布；補繪跟同一原生copy區位移及裁切。
- 首版Go自洽核對通過，實際PNG卻有深色方塊與低對比；entry13對話底色與entry20上框不同，一般字模205亦不代表固定標籤墨色。按原圖改取上框local(100,70)及實際glyph bbox，補直接背景回歸；原錯圖保留作反例，不把首版當成果。兩個仍存活的r1 GPU明確停止後退出143，不報產品signal killed或OOM。
- 首兩次容器預檢使用不存在的runtime子目錄與相對輸出路徑，Go尚未執行；沿先前正式包根與絕對路徑修正。修正後的[命令與診斷](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-commands-r1.json)及[Go來源快照](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-go-sources-r2.json)保留。最終環境明示GOMAXPROCS=2與LP_NUM_THREADS=2配合2CPU容器，屬驗證資源設定，不是玩家規則或平台效能聲明。
- 最終28項頂層Go通過，零SKIP／失敗，含兩項正式JSON原子保存／新實例LOAD。兩條完整Game.Draw共332畫格／49個主要F2邊界及五次續行，另48組四語確認／16次F2；只讀私人6062份獨立runtime，無新圖包。
- 33張完整PNG獨立核對，Go血條2910600點、PNG血條582120點與標籤墨跡110860點。25張圖替換區外56350368個RGBA像素與前版相同；53個收據反例、兩張真實舊圖及首版錯底色圖均拒收。獨立反例曾選到條被裁掉的零點畫格，改選可見條的正對照後同PNG通過，只改Python檢查器，未重跑GPU。
- 集中GPU560.709秒、峰值8589934592 bytes觸8GiB配額，memory.events.max=243，OOM／kill零；狀態411.081秒／4983775232 bytes，8GiB／2CPU且events全零。來源104.519秒／1424879616 bytes、介面105.723秒／1401401344 bytes、交易146.031秒／1511899136 bytes，4GiB／2CPU且events全零。保存回歸另依主收據，沒有由集中測試推論平台餘裕。
- 61由READY轉CONFORMED-SCOPED／RUNTIME-E1；56／57／58與README同步現況。README將同一物品／狀態總覽入口換成當前標籤／條，舊圖仍由先前主收據保存，文化內容與145張圖片數量保持。正式清冊、原包、EXE、來源槽及私人庫保持，沒有新美術同步需求。
- [本批主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的item_panel_values_followup_20261026索引來源、完整命令、兩條收據、獨立PNG核對、實圖反例及[清理紀錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-cleanup-r1.json)。相同歸檔輸出按SHA合併，33路徑內容驗證保持；沒有錄影或留存測試執行檔，現用Go快取保留。圖示、物品／指令列縮寫與其他HD、全部串接後集中正常／人工／平台仍待，完整目標保持。
- 收尾五張GitHub工單已逐內文讀回並維持OPEN，原acceptance／evidence／verify未改；官方解析器重建25項，18項待人判、0項機器訊號消失。359教訓／217守衛通過。工單準備曾誤把parse_issue的(item,error)視為字典，只續跑準備段修正，收據、文件登錄與清理未重做。私人庫本機／遠端main均為377197837c6566d553ca2895335becba1067b482且工作樹乾淨。
- [最終核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-final-checks-r1.json)保存來源、28項Go、33張PNG、文化入口與連結結果。現況過期標籤／條限制搜尋為零，僅一筆當時工作歷程仍記舊待辦，不據此重開。沒有非預期FD2容器；本批重大更新提交／推送main後另核對本機／遠端HEAD，完整HD目標保持。
- 工單最終差異審查發現首版更新覆蓋了原說明後段，已由更新前遠端原文恢復，只替換本批進度前兩段。五張工單各保留9／8／9／10／10段既有說明與使用者定案，逐段讀回相同；原驗收欄位未變。官方重新pull／render／verify通過，[工單讀回](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-issue-readback-r2.json)及[取代前版的最終核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-values-final-checks-r2.json)保存結果。Go、PNG及原始輸入保持，不重跑GPU。

## 2026-10-26 清理過期連結與舊GPU中間截圖

- 依使用者要求優先清理空間。載入復古重製技能、文件職責與交付目錄契約，保留分支目前未提交的HD工作。沒有新增遊戲功能、重跑測試、提交或推送。
- [連結清單](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-plan.json.gz)限定2026-10-25前反例測試pack／runtime內的符號連結。刪139180個連結，實檔及連結目標不刪；[逐筆稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-audit.jsonl.gz)保留原路徑及目標，可據以重建。計畫、刪除與核對使用[腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5.py)的plan／apply／verify入口，既有報告不覆寫。
- [最終圖片計畫](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-captures-r4-plan.json)完整納入受版控文字與9.7MB的HD主證據，以檔名及輸出前綴排除仍有引用的驗證組。45組舊GPU輸出刪929張中間PNG，保留各組首／中／末抽樣、直接引用圖與全部文字收據；原版、目前HD及正式交付不刪。[圖片稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-captures-r4-audit.jsonl.gz)逐檔保存原SHA-256；[腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-captures.py)提供plan／apply／verify。較早三份圖片計畫只供診斷，未執行刪除。
- [保護清冊](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-protected.json)4143份目前程式、存檔與HD證據逐大小及SHA-256複查通過。核對全部已刪路徑與保留抽樣，擁有權1000:1000；工作根未見root-owned檔案或意外.md目錄。原始資料與現用工具鏈、Go快取保持。
- GNU du相同命令的專案實體配置由44356284416降至42893697024 bytes，扣除計畫與稽核開銷後淨減1462587392 bytes，約1.46GB；目前實體占用約42.89GB。硬連結只計一次，不把路徑大小加總當成實際占用。完整結果與可重跑核對入口見[最終摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-final.json)及[核對腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-request-20261026-r5-final.py)。
- 兩個廣域盤點達有界逾時，均未據此刪檔。圖片索引初版限制文件大小而漏載HD主證據，刪除前已取消限制，並補齊輸出前綴及多重副檔名引用；最終只執行r4圖片計畫。保護雜湊與既有未提交差異保持，不把環境或盤點腳本問題列為遊戲缺陷。
- Docker均使用既有映像、有界、--rm、init、UID1000及無網路。未找到可確認過期的FD2映像、容器或volume，不清理其他專案或全域build cache；沒有殘留本批容器。完整HD目標保持。

## 2026-10-26 物品加成與指令消耗縮寫HD

- 從6edb334c的main建立feat/hd-panel-row-labels-20261026。沿已閉合0x184C0物品列及0x1CEED指令列，先登錄61的READY，再接raw64..67的+AP／+DP／+HP／+MP與raw92的-MP。固定字格取自己的底色與最亮墨色，以Noto沿既有48到24目標像素的有界字區直接繪製；名稱、數字、圖示、緊密排列與原生規則保持。
- 補繪使用entry21的310×99完整美術座標，沿同次三區位移與102列複製範圍裁切。物品與指令各有密封圖，空列無補繪；record、typed表、下框、字格與色盤加入簽章，F2完整預檢後才發布。共用物品、狀態及指令頁接入既有繪製層，沒有新美術、資料包、保存格式或原版RE。
- 最終六組共30項頂層Go通過，無SKIP／失敗，含兩項既有JSON原子保存／新實例LOAD。物品五分支291畫格／40個F2與五次續行、狀態41畫格／9個F2保持；另三個typed補樣涵蓋八格四類物品、十二指令與空列，各驗完整Game.Draw及F2。四語確認48樣本／16次F2保持。直接唯讀私人6062份獨立runtime，未複製測試包。
- 補樣只裝備首件武器及防具，回復道具保持未裝備。三張before PNG是同typed狀態、只抑制新字樣的HD控制組，未冒充舊版或原版畫面。既有建構槽、seed7、clock1000與Game清單索引9的指令來源如實記錄；不新增一般鍵盤、自然取得、原版oracle或PLAYER-E2聲明。
- 39張完整PNG由獨立原始23-byte列／7-byte指令表核對，450792個字區點與78912個實際墨跡點通過；字區外64061208個RGBA像素對前版或同狀態HD控制組相同。55收據反例與兩張真實舊縮寫圖拒收，既有細框、四語選項及480頭像保持。目視確認四類物品、三欄指令與滑入裁切，集中造型人工驗收仍依全部HD後排程。
- 集中GPU425.581秒／7845679104 bytes、狀態183.095秒／5140938752 bytes，各8GiB／2CPU；來源44.753秒／1274863616 bytes、介面44.763秒／1352708096 bytes、交易57.271秒／1335476224 bytes、保存6.028秒／609255424 bytes，各4GiB／2CPU。六組memory.events皆零，無OOM或逾時，不推論平台效能餘裕。
- unit-r1誤用assets/data/spells.json，改為既有assets/spells.json後unit-r2通過；審查補樣旗標後沿最終來源跑unit-r3與其餘五組。獨立狀態工具首版把關閉指令頁視為物品頁，依nativeStatusTransitionPass的commands來源修正，只重驗同PNG。彙整器曾誤用full收據的f2_roundtrips欄位，改讀實際f2_boundaries後續跑；既有保存閘門未覆寫，沒有重跑GPU或改正式規則。
- 61轉為CONFORMED-SCOPED／RUNTIME-E1，56／57／58與README同步現況。[HD主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的item_row_labels_followup_20261026索引來源、完整命令、建構政策、測試、獨立核對與[最終檢查](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-row-label-final-checks-r1.json)。20份凍結來源SHA保持，README145張圖片、文化入口及2848個本地連結通過；舊總覽仍由前批主收據保存。
- GitHub #4／201／202／204／263只更新前兩段進度，原說明後段各9／8／9／10／10段、acceptance／evidence／verify、標籤與OPEN狀態逐張讀回相同。官方解析器重建25項，18項待人判、0項機器訊號消失；359教訓／217守衛通過。私人庫本機及遠端main皆為377197837c6566d553ca2895335becba1067b482且乾淨，無新素材同步需求。
- [本批清理](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-row-label-cleanup-r1.json)將16份相同PNG合併為硬連結，釋放11816960配置bytes，39路徑內容SHA保持。未產生錄影或留存測試執行檔，保留現用Go快取；所有FD2容器皆有界、--rm、init、UID1000及network none，沿既有映像，收尾沒有非預期FD2容器。
- 本批重大且已驗證的更新提交／推送main後另核對本機與遠端HEAD。分類／裝備及角色狀態圖示、其他介面／地形、HD2、獨立UI／theme、工具列其餘控制及全部HD串接後集中正常／人工／音訊／平台驗收仍待；完整目標保持。

## 2026-10-26 暫停HD與清理舊素材鏡像

- 依使用者「本週先這樣，先暫停」指示，完整HD目標已設paused。狀態GPU恰好自然完成，293.079秒／5052145664 bytes，memory.events皆零；整段物品GPU由本次docker stop停止，退出143，沒有最終收據，不列產品signal killed或OOM。沒有啟動後續驗證、私人同步、提交或推送。
- 暫停前的新圖示工作保存在feat/hd-panel-icons-20261026，基底main為ea4fb3bdc78ea61e90b4ad8db7f90187c4a0cb9e。imagegen內建工具依原版產生[八圖母稿](remake/generated-assets/modern-theme-prototypes/item-panel-icons-hd6-master-20261026-r1.png)，SHA-256為5508cb81f19e8d273dad44f3c039931bad2d7daa88f218de8ac49fc9dc988662；[提示詞摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-prompt-r1.txt)與十張六倍圖示保留。正式清冊仍841項，851項候選與程式未定版，私人庫未更新。
- 單元十項、介面十一項、交易八項、狀態GPU一項與保存兩項已通過。來源單元峰值觸4GiB、max=117、OOM／kill零，不推論平台餘裕。新載入器曾將素材子目錄再傳入會自行附加子目錄的API，依實際ui根契約修正後通過，教訓已記錄。完整限制與恢復步驟見[暫停交接](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-paused-20261026.json)，[命令](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-commands-paused-20261026.json)及27份來源快照保持。使用者恢復後，整段流程須用全新game-r2輸出；獨立PNG、正式清冊／私人保存、README／工單及提交仍待。
- 清理沿既有work/，沒有新建workplace同義目錄。前兩個廣域計畫達有界逾時，未據此刪檔；實際改按445個已停用的pack／runtime／runtime-only鏡像目錄逐段處理，每個目錄先完整保存連結清冊再刪除。正式資產對這些目錄的依賴為零，原始資料、正式交付及目前新圖工作保留。
- 清除1222913個PNG符號連結，[r3逐筆稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-r3-audit.tsv.gz)保存原路徑及目標；[r3腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-r3.py)的setup／clean／verify提供分段進度及核對。另刪40份與私人正式包逐SHA相同的實體PNG副本，[r4稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-r4-audit.jsonl.gz)保存原SHA、大小與還原來源，[r4腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-r4.py)提供相同三入口。獨有或不同內容的實檔不刪。
- [保護清冊](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-r3-protected.json)3827份逐大小及SHA核對通過，兩批已刪路徑與40份保留來源亦複查通過。若要重跑舊鏡像，先於Docker使用[還原工具](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-restore.py)的--directory work/原鏡像相對目錄列出數量，再加--apply；只還原指定目錄，衝突或來源SHA不同時拒收。私人來源須唯讀掛為/private。
- 已停止的專案Go編譯快取用維護中Go容器執行go clean -cache，由841564160降至12288配置bytes；下次恢復重新編譯。模組下載與現用Docker工具映像保留，沒有全域prune或清理其他專案。
- 相同GNU du命令量測，專案實體配置由43009339392降至37383409664 bytes，淨減5625929728 bytes，約5.63GB，目前實體占用約37.38GB。含硬連結重複計算的檔案總大小為57548999890 bytes，約57.55GB。數值為收尾摘要寫入前量測，摘要與腳本僅增加少量記錄；[最終摘要](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-cleanup-pause-20261026-final.json)保存命令口徑與結果。
- 清理容器均有界、--rm、init、UID1000及network none。工作根未見root-owned檔案或意外.md目錄，私人素材庫仍乾淨，沒有非預期FD2容器。所有未提交HD變更保持；目標維持paused，清理不代表恢復HD工作。

## 2026-10-26 繼續合併重複工作資料

- 依使用者「繼續刪除重複的」指示，只清理舊工作資料，HD目標維持paused。路由載入`~/.codex/knowledge-base/local/project-document-responsibilities.md`，沿用本檔及既有work輸出目錄，未修改正式清冊、私人素材庫或GitHub工單，沒有提交／推送。
- [可重跑腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r5.py)先以大小、檔首尾篩選，再逐SHA-256確認。掃描work內至少4096 bytes的圖片、歷史報告、紀錄及音畫等檔案；排除符號連結、已共用inode的副本、快取、程式來源、執行檔，以及名稱含save／state／overlay／manifest／catalog的可變資料與原保護清冊。初次較窄盤點未套用，最後按擴查後清冊處理。
- [計畫](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r5-plan.json)共1519組、2023份重複副本。逐份再核對雜湊、大小、擁有權、權限及修改時間，將多餘儲存以原子替換合併成硬連結；所有路徑保留。主要節省來自46份相同contradictory-evidence.json及重複截圖。[逐筆稽核](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r5-audit.jsonl)保留來源、原mtime、大小與SHA。舊證據若需再寫入，先複製成獨立檔案，或使用全新輸出目錄，避免修改共用inode。
- [核對結果](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r5-verified.json)為PASS：2023個替換路徑仍存在且與保留來源共用inode，1519組SHA及3827份受保護檔案逐份通過。GNU du實體配置由37383794688降至36337438720 bytes，實際淨減1046355968 bytes，約1.05GB，目前約36.34GB；計畫、稽核及摘要也計入實測。
- 全部工作在既有素材工具Docker映像執行，--rm、init、network none、UID1000、1GiB、2CPU、pids-limit128及有界逾時。收尾檢查FD2容器與輸出擁有權，保留原本未提交HD變更。未重新啟動HD或遊戲測試。

## 2026-10-26 第三輪重複副本清理

- 依使用者「再做一輪」指示，沿用文件職責路由、既有work目錄與3827份保護清冊。HD維持paused，沒有修改正式程式、清冊或私人素材庫，沒有提交／推送。第一次純Python盤點55秒逾時，未改檔；改用容器內find分目錄盤點後正常完成。
- [r6腳本](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r6.py)補查上輪未涵蓋的小型檔案、舊matching編譯／連結／反組譯輸出及jsonl。extracted與remake/generated-assets僅作保留來源，替換目標全在work，原始遊戲org_game及正式交付未納入。快取、資料庫、存檔／狀態／覆蓋層及目前圖示工作仍排除。
- [計畫](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r6-plan.json)確認29002組、161570份逐SHA相同副本，來源與目標須同裝置、擁有權及權限。先完整保存並fsync[還原清冊](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r6-audit.jsonl.gz)，再分兩批以原子替換合併成硬連結，分別136888及24682份；中斷後以inode判定進度，不將僅列在清冊視為已完成。所有舊路徑保留，未跟隨符號連結。這些歷史輸出維持唯讀；需重跑時使用新目錄，需修改既有檔案時先複製成獨立檔案。
- [驗證](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/space-duplicates-20261026-r6-verified.json)PASS：161570個路徑與保留來源共用inode，29002組SHA及3827份保護檔案逐份一致。GNU du實體配置由36337455104降至35291066368 bytes，含本輪計畫與稽核後，淨釋放1046388736 bytes，約1.05GB；目前約35.29GB。
- 沿用既有素材工具映像，容器均--rm、init、UID1000、network none、1GiB／2CPU／pids-limit128及有界逾時。收尾檢查輸出UID/GID、意外root-owned檔案與FD2容器；未重啟HD測試，原有未提交工作保留。

## 2026-10-26 恢復HD1正式接線與共用面板圖示

- 使用者指定恢復HD1，先完成接線作正式版。依現有重製技能、規格閘門及正常玩家試玩路由恢復，唯一狀態表為HD主收據`hd1_release_integration_20261026`。HD2既定方向保留，新增工具列控制是否須隨HD1首版完成已提單一範圍問題；既有接線照常推進。完整HD1仍未完成，沒有修改版號、tag或Release。
- 暫停前27份程式來源SHA一致，沿用已通過的單元／介面／交易／保存與狀態GPU收據；以全新game-r2完成使用者曾停止的物品五分支。十個raw53／54／55／56／57／59..63圖示進正式清冊851項，保留原生分類、裝備、旗標、座標及移動裁切，不重開RE。正式資料與完整結果見主收據`item_panel_icons_hd_followup_20261026`，規格61改CONFORMED-SCOPED／RUNTIME-E1／NORMAL-GUI。
- 選定34項不同頂層Go測試共48次通過，含私人實際包直接載入；受控五分支291畫格、狀態41畫格與三補樣通過。31PNG獨立核對1542672個圖示像素及矩形外62969328像素；19收據反例、兩張真實舊圖示及12個清冊反例拒收。物品GPU362.332秒、峰值8GiB、max=651、OOM／kill零，狀態GPU293.079秒／5052145664 bytes；單元曾觸4GiB，不能推論平台餘裕。
- 正常GUI前兩次在啟動遊戲前因我設定不合法的採樣數及repetitions退出，不列產品缺陷。第三輪真的抓到[#279](https://github.com/wicanr2/fd2_re/issues/279)：Unit.NativeCommandIDs回傳非nil空陣列，snapshotNativeStatusTransition的append到nil複製將[]變null，rowLabels封存簽章因而誤拒。起初將其描述為舊快照殘留，後以直接資料流更正為空表示改變。改用slices.Clone保留兩種指令切片表示，不刪或放寬來源守衛。
- 新回歸最初漏帶HD主題環境，兩個診斷輪次的nil錯誤不作修法證據；修正測試環境後，無指令及空commandBook、舊來源拒收、狀態快照／兩頁與教會生命週期五項通過。教訓新增為空切片快照規則，361條／219守衛通過；十四Python清冊回歸通過。字串盤點另補登先前已發布的十個LV／EX／MU／DX／HIT／EV／AP／DP／HP／MP中點縮寫為player_visible，122候選完整綁定，最終Go來源重建SHA一致。
- [正常r4收據](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-normal-r4-capture/receipt.json)沿既有普通START／F5首章JSON，標題LOAD、走行／指令環／取消、兩場攻防、經驗對話中F9、Home無／有法術兩頁、F2、縮窗／說明頁、自然返回與保存／新程序LOAD通過；來源與兩主題保存bytes保持。34張主要擷取，388.105秒、峰值4950855680 bytes，6GiB限制下events全零。章前來源沿既有SHA，不注入座標或狀態，不新增完整章節或原版對拍聲明。
- [正常視窗獨立核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-normal-independent-r2.json)四張圖的480頭像與頭徽共1021824像素逐來源一致。核對器起初把隊伍序號1當DATO1，改依實際游標下單位battle_fig導出0及9後通過，沒有改圖或放寬像素要求。[README總覽來源](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-readme-r1.json)上排用普通面板，下排用受控分類／旗標與滿背包，完整畫面半縮，原提示及來源頭像保持；不以總覽宣稱完整HD1。
- 私人包7052舊檔SHA與mtime逐份保持，只新增十成品與一母稿，合計7063檔／6072獨立runtime；白名單同步及直接載入通過。私人提交fc70c3cdc0f0e35024c977f4fdcab0b8d09923ba已推送並核對遠端main，與公開引擎分開保存。公開庫只新增總覽、清冊／來源SHA及程式，不加入完整素材。
- 已同步56／57／58／61、README及五張HD工單#4／201／202／204／263。工單只更新目前摘要與過期verify說明，原id／acceptance／evidence、標籤與OPEN狀態讀回一致。#279驗證後關閉，提交後補連結。官方解析器更新25項，18項待人判、0項機器訊號消失；3011個修改文件本地連結通過。最終程式來源見[來源快照](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-sources-final-r1.json)。
- 全部工作沿既有Docker映像，UID1000、network none、有界預算與--rm／init；GUI背景程序由容器擁有，收尾沒有FD2容器或意外root-owned檔案。沒有錄影，[核雜湊後移除舊測試執行檔](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/item-panel-icons-cleanup-r1.json)，保留目前r4與必要收據。前面清理的3827份保護SHA是恢復HD前的快照；後續清理須依目前來源重建，不能把這輪合法修改誤判為檔案毀損。
- 本批重大且已驗證的接線將提交／合入main並核對遠端。其餘HD1地形、HUD及操作介面、工具列新增控制與集中正常／人工／平台仍依唯一狀態表推進，不宣稱HD1正式版已發行。

## 2026-10-26 HD1原生地圖資訊窗接線

- 從main的8a8a2adb建立feat/hd-map-hud-20261026，先在61記READY，再接正式執行期。知識路由沿spec-gated workflow、戰術固定骨架、正常試玩與README標準；不重開已閉合的HUD反組譯。唯一目前狀態仍是58與[主收據](docs/data/fd2-modern-hd-battlefield-20261019.json)的map_hud_hd_followup_20261026。
- 實際FDOTHER5 entry130為69×34，不套用早期149×42概念面板。以原圖自身底色／框色直接繪製414×204細框、A／D、正負兩位數及HP三位數；原圖41／52經目視確認為三個問號，保留大於999的溢位字樣。地形沿實際descriptor與raw取色使用已登錄144圖源，人物沿既有selector與3轉1，HP最後覆蓋。六個地形control及數值規則、原生work／VGA、時鐘、RNG與保存格式保持。
- 正常、行動環復原、事件平移、逐列位移、走行省略與死亡末拍回交均接入；取樣物件及死亡前25拍省略整個HUD。CPU只快取最近的不可變框圖，GPU使用固定可重寫紋理，不把逐幀HUD放入地形指標快取。
- [單元閘門](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-unit-final-r1.gate.json)、完整GPU、同來源裁切、owner／保存、死亡來源拒收及字串盤點共30項不同頂層Go通過。完整GPU驗125組／500原selector狀態及18組畫面、四次F2；實際已登錄地形20736像素、六個control、HP邊界、來源反例及凍結快取均通過。原圖字格與Noto的有界字形另作獨立期望，不以HUD自身格式器產生測試答案。
- [獨立PNG核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-game-independent-r2.json)通過細框、數字子像素、登錄地形與七種變造收據，兩張實際舊HUD拒收。舊hud-game-r1使用較早地形清冊，不能拿整張圖作本輪框外基準；另建同來源、同畫面的原生HUD控制組，實際GPU與PNG均確認矩形外2219544像素相同。此控制組是RUNTIME-E1幾何驗證，不是新原版對拍。
- 早期診斷包含漏Xvfb／分離包環境、fixture未在場、穩定八列不足九列staging，以及誤把既有材質差值映射當成完全換色，均依實際來源修正測試，未改遊戲規則。補上字型也暴露受控Game.Draw槽殘留的開場對話；測試明示中立地圖狀態後，HUD最終顯示比較保持原條件。完整GPU最終562.954秒、峰值4930498560 bytes，6GiB下memory.events全零。
- [正常面板流程](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-normal-r1-capture/receipt.json)從原普通首章槽標題LOAD，走行／行動環／取消、兩場演出取樣及F9回載、Home狀態／指令、F2、保存與新程序LOAD通過。34張主擷取，四張HUD獨立核對；470.604秒、峰值4800806912 bytes，6GiB下events零。此組不宣稱自然返回。
- [自然返回補驗](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-normal-natural-r4-capture/receipt.json)沿同一來源、一場普通攻擊，自然返回、戰後F5、F2及新程序標題LOAD通過。19張主擷取，三張HUD獨立核對；392.414秒、峰值5597827072 bytes，6GiB下events零。F5仍是既有章節邊界保存，不宣稱保存戰中位置、Acted或敵方HP，不新增完整章節或原版parity。
- 自然返回r1在啟動前缺battle-status參數；r2在冷啟動被外層900秒測試器終止，OOM零。r3加長外層後暴露內層整輪仍固定660秒。GUI工具新增明示且有界的--run-timeout，r4以1500秒預算乾淨重跑通過，遊戲SHA不變。死亡來源拒收初輪同樣在既有PNG載入期超過240秒，600秒有界重跑通過；不是新HUD錯誤。owner初輪兩項缺顯式主題環境而SKIP，不列通過；死亡已補驗，指令10～12 stage來源拒收不新增本輪驗證範圍。
- **勘誤前一批自然返回敘述**：item-panel-icons-normal-r4的實際natural_attack_return_checked=false；前批主收據的true與本歷程「自然返回」文字過度推論。已修正目前主收據及58，原收據與歷史文字保留可追查。前批面板、保存／冷讀仍成立；本批另以真正自然返回收據補驗，不能倒推成前批已驗。
- 本輪沒有新增素材或修改私人圖包，只新增公開README完整視窗總覽。清冊與私人HEAD保持。新的[README來源清單](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-readme-r1.json)上排為普通讀檔／F2，下排為受控地形與溢位；全部保留完整視窗並半縮，沒有補畫。字串122候選重綁並通過八項檢查；目前來源乾淨重建與已驗binary SHA逐位元組相同。
- 56／57／58／61及README同步；五張HD工單#4／201／202／204／263已更新，完整舊尾段、驗收條件、標籤及OPEN狀態讀回一致。官方解析器從主機gh現況更新25項、18項待人判、0項訊號消失。新增GUI預算教訓，362條／220守衛通過。未發布新版，未登錄地形、其餘操作介面與HD1集中人工／平台仍待。

- 收尾依[清理記錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/map-hud-cleanup-r1.json)核SHA移除已被取代的前批binary，保留目前已驗程式與收據；未建立錄影。Docker沒有FD2容器殘留。本批修改與產物均為UID/GID1000，沒有錯型態md目錄；全庫另有2808個既有root-owned檔，不對全庫修權限。修改文件3029個本地檔案／圖片連結通過。

## 2026-10-26 回復術13～16正常鍵盤與HD演出補驗

- 從`0dad7c6`建立`feat/hd-heal-normal-20261026`，沿重製驗證技能與規格流程，先在61寫READY契約，再補唯讀audit及鍵盤工具。沿已閉合的原生回復排程與既有HD圖源，不改規則、時序、傷害公式、素材或存檔格式。來源及結果集中於HD主收據`heal_normal_gui_followup_20261026`。
- 使用已審查第23章來源，SHA為`5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871`；章前AP+5000／DP+300／DX+60／MaxHP+1000、保留傷亡沿舊manifest。本輪沒有新注入。希爾法identity24已出擊並學會13～16，來源HP1253／MP276。全員滿血，因此限定正常演出與滿血上限，不外推正回復量、自然難度或完整章節。
- [正常GUI收據](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-normal-gui-r3-capture/receipt.json)從標題LOAD，以Z循環、原位置移動確認、法術環及正式目標格施放。四種完整演出各一次，另一次13專驗途中F2；案例間明示F9回載，固定FD2_SEED=1，不重擲挑選。取消不扣MP，正式目標數為1／4／4／5；四種扣MP為3／10／20／40。逐拍audit確認五例各只有一個MP消耗邊界，滿血Actual均0，結束後已行動並自然返回地圖。
- F2往返時回復owner保持；F5章節邊界保存、原版／HD保存bytes一致、新程序標題LOAD及單位冷讀保持。未宣稱保存戰中MP、位置或Acted。正常GUI442.467秒、峰值4663193600 bytes，6GiB下memory.events全零。19張主擷取、46張動畫取樣與兩程序audit保留，沒有錄影。
- [獨立核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-normal-independent-r2.json)驗正式17份光環／綠字圖源、Go預乘RGBA雜湊、實際矩形、半透明混色區間及多目標層序。46張觀測中26張有相容圖源核心，共103286點，四種皆有光環及數字。其餘為跨CPU／GPU相位觀測，不強求每frame相同。14項來源、案例、MP、主題、矩形、圖源、冷讀、F2 owner、黑視窗及六倍像素化反例拒收。像素化反例由實際視窗降採樣建構，不冒稱原版擷取。
- 來源、完整主題、原生交易、死亡來源及字串盤點合計19項不同頂層Go、21次通過，無SKIP／失敗。來源測試驗425984原生幾何點及180畫布；四種HD替換區域外保持，6072份獨立runtime直接載入。完整主題[重跑閘門](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-normal-unit-r2.gate.json)349.933秒、峰值5925679104 bytes，6GiB下events零。既有八條受控Game.Draw／F2收據保持，不冒稱本輪重跑全部八條。
- 初輪13在F2像素模式期間走完HD效果段，施放與保存通過但缺HD取樣。改成完整演出與獨立切換案例，沒有放慢遊戲或改排程。第二輪命令誤用相對out，PulseAudio在啟動遊戲前失敗，依既有絕對路徑教訓修正；第三輪通過。完整主題初輪4GiB觸memory.max=535並在Go的270秒界線逾時，OOM／kill零，未列通過；同來源改6GiB／600秒有界重跑通過，不記為產品缺陷。
- README新增四種回復術完整視窗半縮總覽，來源依[圖片收據](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-normal-readme-r1.json)，無補畫。56／57／58／61同步。61舊「血條／框線仍DRAFT」已由目前正式接線直接反證，改指向既有整合證據；原型及取得研究仍可回查，集中人工狀態保持。
- GitHub #4／201／202／204／263已更新並讀回，完整舊尾段、驗收、標籤與OPEN狀態保持。官方解析器從主機gh現況重生25項快照，18項待人判、零項訊號消失。新增動畫取樣教訓後363條／221守衛通過。字串盤點122候選及審查SHA保持，八項字串測試通過。
- 素材清冊851項、私人7063檔及私人HEAD保持，無新圖包同步或正式發行。乾淨重建與已驗程式逐位元組相同，SHA為`74863ad9688fb845aaf8a0477f61b73468095488920d0d239053e6fe08857ca8`。[清理記錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/heal-normal-cleanup-r1.json)核SHA移除舊HUD程式、重建複本及49張被取代的初輪PNG，共63211087 bytes；保留正式第三輪圖片、收據、audit與目前程式。HD1其餘操作介面、未登錄地形及集中人工／平台仍待。
- 收尾核對3053個本地文件／圖片連結及235個本輪輸出路徑通過，UID/GID均1000，沒有新增root-owned檔或錯型態md目錄。2808個舊root-owned路徑維持不動。三個FD2工具鏈映像沒有執行中或停止後殘留容器；私人庫乾淨且HEAD保持。

## 2026-10-26 戰場資訊四面板HD接線

- 從`03163469`建立`feat/hd-system-info-20261026`，沿重製驗證技能、規格流程與既有工作根。先核對`0x19F03→0x1B1E7→0x1B41D`的閉合證據、四圖133～136、60條原FDTXT條件及原區域copy，再將61寫為READY。沒有重開RE、改正式數值或保存格式；來源與結果集中於主收據`system_info_hd_followup_20261026`。
- 四面板改用三像素原色細框及直接Noto繪字；保留MAP／TURN、ENEMY／FRIEND／NPC、貨幣、原前導零及兩行條件。原12字／FFFE換行保持，字區逐行量測及縮字，不截字。原章號、回合、金錢、raw camp0／2／1及glyph words在開窗前凍結，章16分支保持。GPU紋理與CPU圖只保留目前使用字型及色彩，不因不相干DAC pulse逐幀重建。
- 新唯讀區域轉接器直接消費原十二拍開收算式，沒有再抄一套幾何。HD只覆蓋原copy目的區域，矩形外保留當次HD地圖；移動先移整幅來源再裁切，裁切邊不生成新框。像素模式保留原生buffer，F2進HD先預檢完整圖源、字型與顏色；失敗不發布半份視窗。
- [來源及回歸](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-unit-r2.memory.json)與字串盤點共23項不同頂層Go通過，沒有SKIP／失敗。四圖像素seal、60條文字／零缺字、30章及章16分支、來源反例、原生十二開／等待／十二收、快取與凍結保持通過。六數字另以原位置、明示值、前導零及camp順序獨立畫期望，不調用HD自身格式器。
- [完整GPU](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-game-r2-capture/receipt.json)驗25個位置及25次F2。每個g.frame暫不掛資訊owner，走完整Game.Draw及DrawFinalScreen產生HD地圖控制，再掛回同owner重畫；原生work／VGA、RNG與phase／frame保持。最終GPU及Pillow對原copy區域合成57600000像素，RGBA色差至多1。這是明示章0／回合255／金錢99999999／camp數12、34、56的受控測試，不是普通玩家或原版對拍。294.468秒、峰值4563124224 bytes，6GiB下memory.events零。
- [普通鍵盤](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-normal-r2-capture/receipt.json)沿已審查第23章來源`5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871`，章前強化政策沿舊manifest，本輪沒有新注入。普通移游標至空地、系統環、巢狀選單及資訊入口共開窗三次；HD／像素資訊值一致，單位HP／MP／位置／Acted不變，原任意鍵收合保持。實際F2也會收窗，沒有將受控Draw的F2宣稱為按鍵可維持等待頁。
- 普通流程的返回、F5章節邊界、兩主題保存bytes與新程序標題LOAD通過。7張主擷取，兩張HD四區重建1820808像素，與當次CPU預檢圖SHA完全相同。344.257秒、峰值5610127360 bytes，6GiB下events零。不宣稱自然難度、整章E2或戰中位置／Acted保存。
- [獨立PNG核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-independent-r2.json)另拒收十種變造／缺幀、真實忠實像素主題及舊暗字HD實圖。兩個實圖反例與正式資訊值相同；它們是重製兩主題及前版HD，不是新跑的DOS原版執行收據。完整來源、工具與程式SHA均在主收據。
- 第一輪來源與GPU自比對雖綠，實圖發現標題誤取陰影76。由原圖文字區及同次忠實像素視窗確認169／173／177為亮字面；現行取169，沒有另定RGB。新增獨立取色及數字期望後重跑全部來源、GPU與正常流程。初輪不列最終通過證據；保留一張錯字色HD圖作反例。教訓守衛原誤指測試檔，既有掃描契約排除測試檔，已改綁實際產品取色程式，不放寬掃描器。
- README新增完整遊戲視窗半縮總覽，上排為資訊值一致的重製像素／HD主題普通流程，下排為明示數值的受控開收，無補畫。來源依[圖片清單](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-readme-r1.json)。56／57／58／61同步；原中文條件的四語內容另外以官方格式開[#280](https://github.com/wicanr2/fd2_re/issues/280)，未把HD繪製當成翻譯完成。
- GitHub #4／201／202／204／263更新並讀回，舊內文、驗收區塊、標籤與OPEN保持。官方解析器以主機gh現況重生26項快照，19項待人判、零項訊號消失。新增取色教訓後364條／222守衛通過。七個原生標籤字串登錄為玩家可見；129候選重綁，八項字串測試通過。
- 沒有新美術包，清冊851項、私人7063檔及私人HEAD保持。當前程式SHA為`7b301ab992d06742fab2a693d3c22cf8136112e9c51c8044d8957668ad7aebd3`，乾淨重建逐位元組相同。[清理記錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/system-info-cleanup-r1.json)核SHA移除62個舊程式、重建複本、初輪重複PNG及audit，共184121437 bytes；正式第二輪完整保留，沒有錄影。未發布新版，HD1其餘介面／地形與集中人工／平台仍待。
- 收尾核對3071個本地連結及224個本輪輸出路徑通過，UID/GID均1000，沒有新增root-owned檔或錯型態md目錄；2808個歷史root-owned路徑不動。首次find碰到掃描途中移除的環境隱藏目錄，改用GNU find的`-ignore_readdir_race`重跑，不略過目前檔案的擁有權檢查。三個FD2工具鏈沒有容器殘留，私人庫乾淨。

## 2026-10-26 戰場資訊四語內容與兩主題

- 從main `83941aada4236653a2831de8550b0e394be75ff9`建立`feat/hd1-system-info-localization`。沿既有資訊窗來源，先寫READY規格，再接官方四語條件及標籤；不重開RE、不改勝敗規則或存檔。各章保留來源ID，章16未有identity18仍選0627／0628。既定譯名沿entities，原密蒂字形保留在繁中來源，不另創角色。
- 像素繁中保留原圖字，其他語言保留原框；HD沿既有細框及字區。來源、缺翻譯、錯來源、空行、長譯文與語言切換快取均有實際字型回歸。失敗不發布半份新圖；原生buffer及owner節拍保持。第一輪測試把不同章的第二條誤當相同來源，已改正測試資料，未放寬產品守衛。
- 59項不同頂層Go、64次通過，含完整8項字串測試；三個專用GPU／晚期整備入口未啟用，不列通過。129候選重新綁定，處置集合及數量保持。
- [正常收據](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/info-locale-normal-r2-capture/receipt.json)以已核對第23章章前強化來源經標題LOAD、空地系統環及巢狀資訊入口，完成四語兩主題開關、F2／F4、返回、F5章節邊界與新程序標題LOAD。沒有新狀態注入或戰鬥。436.502秒、峰值5469519872 bytes，6GiB下events全零。
- [獨立核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/info-locale-independent-r1.json)核HD四區實際像素、官方內容與像素框，七種變造／實圖反例均拒收。README以受版控展示工具合成四個完整視窗半縮圖，沒有補畫。數量依58，來源SHA、命令與限制集中HD主收據`system_info_locales_followup_20261026`。
- 當前程式SHA `edd0e8f5a636d93387ca5c3f422e4cfaed0d5d7bc076aef4d086de46c16b3671`，重建完全相同。沒有新美術包，851項清冊、私人7063檔及私人HEAD保持。沒有錄影或新Release；HD1其餘介面、未登錄地形與集中人工／平台仍待。
- 清理前核對SHA，移除三個已取代或重建複本程式，共62517920 bytes；保留本輪驗證程式、正式收據與必要PNG。掃描5298個本地連結，本輪新連結沒有缺檔；八個既有缺檔均為未修改歷史文件所指的舊dist-all版本，另列衛生收據，不冒稱全庫連結皆存在。初版掃描另將inline／fenced程式表示式誤判為連結，排除程式碼後才作此分類，沒有改產品或歷史來源。114個目前來源／輸出擁有權通過，歷史root-owned路徑2808個不動，沒有新增root-owned或錯型態md目錄。
- 實作提交7bbce98f。GitHub #280附提交與限制後關閉，#4／201／202／204／263讀回確認OPEN，原驗收區塊、標籤與內文保留。主機gh現況經官方解析器重生25項快照，18項待人判，零項訊號消失；364條教訓與222守衛保持。私人庫乾淨，FD2一次性容器已清除。

## 2026-10-26 音速刃普通施放、HD圖源與返回

- 前輪已提交資訊窗四語接線，本輪回到持續目標「完成 fd2 戰鬥動畫hd」。從main 94d6360d建立`feat/hd-command30-normal-gui`，目標完整範圍未縮小；本輪補尚缺的音速刃玩家流程，完整戰鬥HD仍未宣稱完成。
- 以目前清冊及私人實檔重跑[圖源覆蓋](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-completion-coverage-r1.json)，264已解碼資源／2118幀全登錄。408仍SOURCE-BLOCKED，資料齊全不推定全部consumer與玩家。發現原觀測器漏掉共用分段owner且固定標29，官方格式登錄#281；改讀typed schedule並納入scene閘門，未改交易、排程或規則。
- 既有177份JSON來源皆沒有蜜蒂。先寫READY規格，再從固定map16第36筆經正式JOIN與原生升級writer，用一次EXP300／seed7取得30，正式回復後套明示章前AP／DP／DX／HP政策。新增第23章隊員，逐筆斷言其他角色及章節保持。[來源](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/command30-normal-source-r2/fd2_save.json)SHA為`56aac676d2c8bd4da3c9315c700e721a72f658eb03577f3d3e658563907799a2`，不是自然招募或完整章節。
- [正常GUI](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/command30-normal-r3-capture/receipt.json)從標題LOAD，普通進軍兩個END、十字選目標、取消／重選、施放、F2、自然返回、F5及新程序LOAD通過。MP29→5、目標HP680→0各一次；原始audit與摘要獨立核對，已學30保存不失。沒有章內注入、鎖血或改敵方，沒有錄影。303.751秒、峰值4423667712 bytes，6GiB／2CPU下events零。
- 首輪工具以菱形預篩誤選斜角敵人，正式目標格拒收；第二輪施放／返回已過，但全域指定角色套到戰後，Z不能選Acted蜜蒂，未完成保存。兩次都保留記錄並列工具問題；同一程式、來源及seed修正後乾淨重跑，沒有把它們列為產品缺陷或通過收據。
- [獨立PNG核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/command30-normal-independent-r3.json)以正式56銀行確認實際斬擊5264核心點，唯一相容為frame6，當時audit先行到7。另兩張跨相位只作觀測，不要求每frame一致。七種變造與實際像素主題反例均拒收；像素反例只在必要時最近鄰等比例至1920，不冒稱同frame或DOS原版。初版核對器誤用顯式FIGANI.DAT辨別值，依正式空archive分派修正後重驗原PNG，核心門檻保持。
- 18項不同頂層Go、23次通過，包含完整8項字串；專用匯出與隔離子程序冷讀分列。129候選綁定保持。程式SHA`62b6c940bc757d18ca690599de1a9d177d6ca1a6d7e7607ec52b9a6d9584e03c`與重建完全相同。README以四個完整實際視窗更新音速刃展示，來源與縮放步驟由受版控工具保存，刪除「框線血條仍像素／尚未正常返回」的過期現況描述。
- 沒有新增美術包，851項清冊、私人7063檔及私人HEAD保持。#263完整consumer、集中造型與效能等原驗收繼續；#268／269待集中自然度確認，#265合法caller原版oracle未知另列。未發布新版，Goal保持active。
- 新增十字候選、動作範圍選人及typed排程識別教訓，367條／225守衛通過。核SHA後清掉重建複本、前版程式、重複來源及失敗輪次多餘PNG／audit，共95600006 bytes；保留失敗記憶體／log、末狀態診斷與必要PNG，正式第三輪完整保存。掃描5310個本地連結，本輪無新增缺檔；既有8個舊dist-all缺檔另列，不冒稱全庫連結存在。256個目前路徑UID/GID1000，歷史root-owned 2808個未動，沒有錯型態md目錄。
- 2026-10-27收尾：實作6ac91c8c附證據後關閉#281，五個父項讀回保持OPEN，原驗收與標籤不變。#263觀測說明回到原戰鬥驗收範圍，未拿介面／工具列擴張完成閘門。官方解析器重生25項快照，18項待人判，零項訊號消失。

## 2026-10-27 四種美術主題與HD-2D原型

- 使用者明示保留原版／HD／HD2／HD-2D四個獨立選項；HD2定版斜角站姿與海報不改名、不合併。操作介面原版／HD獨立，目標八組合。61記錄已確認方向及未定的HD-2D人物、投影、鏡頭與架構，56／57／58同步；正式遊戲仍只有原版／HD，未新增production接點。
- [可操作原型](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/hd2d-theme-prototype-20261027-r1.html)並列原像素人物與既有HD人物，地景、鏡頭與光影相同。以既有Playwright映像內Chromium的真實點擊核對四按鈕、1920／1024單列、敵方人物／右下圖示同步、光照與景深；[瀏覽器收據](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/hd2d-theme-prototype-20261027-r1.browser.json)標DRAFT、game_connected=false、style_confirmed=false。這是示意地景，不是原版地圖、玩法或正式八組合驗收。比較PNG原圖保留1920×1200，另等比例縮1152×720供對話預覽。
- 來源、圖像雜湊及兩個原型工具留在既有工作根，沒有新增素材清冊或私人包。已展示兩候選並發出單題風格選擇，未以沉默採用建議。HD1與戰鬥驗收沿既定範圍繼續。
- 主機gh更新#204的四主題／八組合要求，保留其餘驗收、標籤與OPEN狀態，遠端讀回一致；官方解析器重生25項快照並驗證。文件與原型工作留在feat/four-theme-hd2d-prototype，沒有新Release。
- 收尾核對17個目前檔案UID/GID1000、三張瀏覽器PNG及HTML的SHA一致；工作清單25項／18項待人判／0項訊號消失。diff檢查通過，本批FD2及Playwright一次性容器沒有殘留。未新增錄影，未刪除其他專案容器。

## 2026-10-27 戰鬥HD正式消費端圖源清冊

- 上輪四主題規格及議題有實際更新，本輪回到原持續目標「完成fd2戰鬥動畫HD」。沿重製驗收與規格流程，先在61寫READY，再新增受控正式job匯出器及獨立核對器；沒有改正式交易、排程、素材、存檔或原版回退。
- [生成清冊](docs/data/fd2-battle-hd-consumers.json)涵蓋物理共／分背景雙側、普通0～9、分段24／29／30與召喚32～35，唯一數量依58。普通與分段從正式owner逐拍取來源，物理與召喚明示預建來源；預載sourceFrames不冒稱完整畫格。分段初始targets以值複製，Go種子與原生seed明示，不把施放後HP當初始狀態。
- [最終觀測](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-consumers-observed-r4.json)與第二次重播逐位元組相同，SHA為059a6f771901dbeba365c19219f2662f0d3550d470ead471c974f41232078947。兩輪各21個子案例，頂層30.56／44.06秒，4GiB／2CPU／PID256下無SKIP／失敗；不由容器成功推論平台記憶體餘裕。來源樹、正式清冊及原版manifest指紋綁定。
- 獨立核對原生座標／尺寸／delay／raw4、5、7、像素與遮罩，以及實際HD檔案。414組HD成品及兩組原全透明TAI分列；12種錯命名空間、資源、畫格、像素、遮罩、缺案例、假GPU、缺HD、錯來源碼、指令、packed與描述子反例拒收。召喚地圖尾段、非archive圖層／Base像素、全部角色與分支、正常玩家、GPU最終畫面、人工造型與平台未由本清冊提升。完整戰鬥HD仍未完成。
- 初版核對器將背景0／255遮罩當成動畫loader的0／1，另未區分全透明TAI與packed轉色；由正式loader及原始PNG確認後修正，未放寬來源SHA。來源樹另遇Path元件排序與Go完整字串排序不同，改為同一完整路徑字序；不是遊戲或來源異動。早期JSON只作診斷，最終收據以第四輪為準。
- #263同步已驗範圍及限制，讀回保持OPEN、原驗收與標籤不變，明示工具／文件仍在本機工作樹，未冒稱已併main。官方解析器重生25項快照，工作清單18項待人判、零項訊號消失。61與58及HD主收據battle_consumer_inventory_followup_20261027同步。沒有新美術包、私人同步或Release，也沒有新的DOS動態對拍或逐幀要求。
- [清理記錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-consumers-cleanup-r1.json)核SHA移除七份中間／重複JSON，共10773603 bytes；保留第四輪完整觀測、兩輪log、原始SHA及生成清冊。沒有新增錄影。HD-2D風格題仍待回覆，未阻擋此批HD1工作。
- 收尾核對14個目前路徑擁有權、清冊／工具／log SHA、Go來源樹及新增文件入口通過；工作根find未發現root-owned檔或錯型態md目錄。私人庫乾淨且HEAD保持fc70c3cdc0f0e35024c977f4fdcab0b8d09923ba，FD2與Playwright工具鏈沒有殘留容器。現行變更留工作樹供後續完整驗收，未提交／推送或發布。

## 2026-10-27 召喚地圖尾段來源與GPU32補驗

- 前輪archive消費端清冊已落地，本輪接原持續目標的地圖尾段缺口。沿重製驗收／規格流程，先在61列READY擴充，再加入凍結scene／source核對、LMI1與同selector人物遮罩觀測。沒有改正式規則、來源選擇、交易、時序、素材或存檔。
- [現行清冊](docs/data/fd2-battle-hd-consumers.json)涵蓋四召喚地圖尾段，唯一數量依58。第六輪與第七輪[原始觀測](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-consumers-observed-r6.json)逐位元組相同；21個子案例各跑一次，頂層51.67／38.64秒，無SKIP／失敗。原版LMI raw獨立重解、實際HD檔案與原人物來源核對，新增七個地圖／LMI／遮罩反例，與既有反例共19種均拒收。
- 四次TestModernCompoundMapAllSources完整主題預備皆通過，原生重建、狀態／RNG保持與各兩個原子拒收保持；所列LMI沒有原生回退。此helper統一native seed7，而來源匯出35沿既有fixture seed1，故額外數字116／117／118也獨立核對。兩組狀態分列，不把不同數字當錯誤或宣稱同狀態對拍。
- [GPU32](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-consumers-map32-gpu-r1-capture/receipt.json)走實際Game.Draw與最終繪製、十階段F2、158個present及一次續行；地圖網格／HD覆蓋核心與面板實際RGBA核對通過。233.922秒、峰值6424424448 bytes，6GiB／2CPU／PID256下memory.events全零。峰值接近上限，不能推論平台餘裕；網格容差2，不稱全畫面逐像素或DOS動態parity。保留21張受控GPU PNG，畫面邊界裁切是測具位置，未冒稱正常玩家。
- 工具原碼另留SHA相同的追溯快照，正式重跑仍走repo入口；此快照不是第二個執行器。主收據新增battle_consumer_map_inventory_followup_20261027，舊archive-only摘要標歷史並保留原始第四輪觀測／log。現行同名生成摘要已更新，舊摘要SHA只記錄當時輸出，不宣稱現在檔案仍有舊SHA。
- 58／61與#263同步，遠端讀回保留原驗收／標籤及OPEN；官方解析器重生25項快照，18項待人判、零項訊號消失。沒有新增原版對拍、正常玩家、私人美術包或Release。完整戰鬥HD仍未宣稱完成；接續核對原驗收的正常反擊證據與造型，不重複盤點已驗同族圖源。
- [清理記錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-consumers-map-cleanup-r1.json)核SHA移除三份重播／中間JSON，共3585326 bytes；保留第六輪完整觀測、兩輪log、四個預備收據、GPU／記憶體收據及必要PNG。沒有新增錄影，當前變更仍留工作樹，未推送。
- 收尾核對53個目前路徑、Go來源樹、工具快照、GPU PNG與收據SHA通過，UID/GID均1000；工作根未見root-owned或錯型態md目錄，FD2與Playwright工具鏈沒有殘留容器。私人庫乾淨且HEAD保持fc70c3cdc0f0e35024c977f4fdcab0b8d09923ba。28／31沿58既有非阻擋分類，本輪未重開RE，也未將未授予指令補成新正式動畫。

## 2026-10-27 正常反擊與戰鬥呈現接點收尾

- 以既有首章來源SHA `089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2`、seed1及目前正式程式重跑普通END、我方攻擊、自然返回、F2、章節邊界保存及新程序標題LOAD。[正常GUI](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/counter-normal-20261027-r2-capture/receipt.json)通過，沒有章內鎖血或狀態注入；不宣稱自然難度、DOS原版或整章E2。
- [獨立PNG核對](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/counter-normal-core-20261027-r2.json)有七張敵96發動／我4反擊的攻擊圖源，17903個可見核心點；前後同owner與首程序74筆audit相符。十二張不同PNG共有十四個相容來源，不以audit幀號作GPU畫格真相。八項角色／owner／階段／主題與缺證據反例拒收。
- 初輪攻防已自然返回，測具卻強制選已行動索爾而錯報失敗。戰後改沿原生Z選下一個可行動者，指定角色只作用於攻擊，同binary／來源／seed／策略重跑通過；正式選人規則不變。既有動作範圍教訓更新，367條／225守衛通過。
- [AST盤點](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-owner-coverage-20261027-r1.json)將16個正式快取owner連到17筆建構案例及正常物理證據。第6指令由明示return轉接到cursor建構器，工具依實際呼叫關係補讀；缺第32指令、錯建構器與缺反擊三項反例拒收。這是結構覆蓋，不推論全部參數或其他地圖owner。
- 蓋亞六及凱麗八個原圖／目前HD姿勢做助手離線抽看，未見明顯反折手腕或分離靴。[圖源與對照圖](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/pose-review-20261027-r1.json)保留，不提升為正式GUI或最終人工接受；#268／269保持OPEN。
- 主收據新增`normal_counter_followup_20261027`，56／57／58／61同步。GUI298.292秒、峰值觸6GiB、memory.events.max=59且OOM／kill零；不能由此宣稱正式繪製效能。原始#263仍需代表性效能與集中造型，沒有將HD2或其他介面加入其原驗收。
- 接續先於61寫READY，加入預設關閉的頻率觀測轉接器及GUI雙程序輸出；既有Ebiten 2.6.6註解明示ActualFPS的虛擬垂直同步限制。程式保留原Update／Draw／最終繪製與Layout，兩項觀測窗／覆寫拒收／IO失敗測試通過。正常同來源重跑及實測仍在進行，未先宣稱效能通過。

- 效能觀測接線完成：目前程式重新跑正常反擊、十張攻擊PNG／25542核心點、八反例、F2／自然返回與保存冷讀通過；[GUI第三輪](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/counter-normal-20261027-r3-capture/receipt.json)282.645秒、峰值4988821504 bytes，6GiB下events全零。前一輪工具原碼已封存，舊收據仍有效，最新狀態改指battle_performance_followup_20261027。
- [密集PNG觀測](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-performance-normal-r1.json)與[關閉連續PNG觀測](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-performance-no-art-r1.json)皆核兩程序、來源SHA、時間窗及各四反例。純物理Draw平均9.86／14.00Hz、更新14.87／35.56Hz。第二輪252.988秒、峰值5536264192 bytes，events零，另有GODEBUG回收紀錄。兩輪不是只差PNG的嚴格效能實驗，不從差值認定唯一原因；CPU與軟體GPU負載待profile，效能未通過。本機/dev/dri不存在，沒有硬體GPU驗收。
- 新增觀測來源後重跑21個consumer案例，1168份Go來源，26.205秒，案例內容對r6完全相同；[更新清冊](docs/data/fd2-battle-hd-consumers.json)及19反例通過。先前同名摘要封存並修正主收據引用，避免舊SHA誤指目前輸出。16個快取owner／17建構索引也改用目前來源及反擊PNG，未重開原版RE或提升全部可達性。
- GitHub #263更新讀回，原驗收、標籤與OPEN保持；官方快照25項／18待人判／零訊號消失，367教訓／225守衛通過。目前及前輪187個完整路徑參照、SHA與UID/GID1000核對通過，工作根無root-owned或錯型態md目錄。
- [清理紀錄](work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/battle-performance-hygiene-r1.json)移除本批失敗PNG／audit，扣回保留尾段後共109781958 bytes；另以相同內容共用inode保留全部有效PNG路徑，估計少配置31711232 bytes。清理後的衛生檢查誤跟隨既有斷鏈，中斷了晚寫的清冊；由保留觀測與capture log補回，16份失敗PNG的原始SHA未落盤，明示未知。原始素材及正式有效收據保持，沒有錄影或新Release。後續清理先落盤再變動，斷鏈以lstat分列。
- 來源提交1467242f已推送main，遠端HEAD逐字一致。#263改指已發布來源並再次讀回，原驗收／標籤／OPEN保持，#268／269保持OPEN；25項快照重新產生。新增清理落盤教訓後368條／225守衛通過。私人庫無變更，本批FD2工具鏈容器已退出，效能profile定位是下一工作，未宣稱Goal或HD正式版完成。

### 2026-10-27 戰鬥HD CPU定位與撤回未證實優化

- 沿既有效能觀測加入預設關閉的30秒CPU profile，獨占新檔、到期／提前Close關檔及正常流程收尾已驗；六項CLI錯用在建立輸出前拒收。
- 同一HD攻防owner的兩秒執行緒增量顯示llvmpipe約165% CPU、全遊戲185%，另記cgroup節流；Go profile不能代表全部原生負載。
- 原繪製的2／4／8CPU軟體GPU普通攻擊、返回、F2、章界保存及新程序LOAD通過，物理Draw約15.34／24.18／35.66Hz。4與8CPU只跳過走行PNG重驗並明示world驗證false，不修改正式規則或品質門檻。6GiB各輪無OOM，硬體GPU仍未驗。
- 候選DrawImage／預乘alpha著色器通過2087 GPU案例及正常GUI，但無可靠加速證據，正式繪製已恢復7e4cdd70。早期交錯測具因不對稱的計時外逐RGBA比對而有偏差，數字作廢；修正後同碼負控制RGBA精確、兩側中位數接近，仍不作玩家FPS。
- 第一輪4CPU在短走行擷取無有效PNG而停止，列驗證觀測不足；首次無audit執行緒取樣落在冷啟動，排除戰鬥推論。沒有新增RE或改動私人素材。
- 唯一目前入口：docs/data/fd2-modern-hd-battlefield-20261019.json 的 battle_cpu_diagnostics_followup_20261027，原始與衍生證據逐SHA登錄。來源清冊與遠端議題待本批最終核對；未宣稱完整戰鬥HD或人工批准。
- 最終Go來源清冊已凍結：1171份來源、21案例與前批相同，獨立核對19反例拒收；AST仍為16快取owner／17建構。四項觀測測試與六項CLI守衛通過。新正常GUI衍生對照已由代理程式目視抽驗，未見先前明顯手腕反折或右靴露底；這不等同人工批准，#268／269仍依全部HD後集中確認。
- 教訓檢查初輪失敗屬新守衛指定測試檔的格式錯誤：工具原本排除_test.go。保留排除政策，守衛改查61內的對稱量測契約，實際GPU負控制仍由獨立收據證明，不改遊戲程式迎合檢查。
- 最終工作清單25項OPEN／18項人工／無消失訊號，370條教訓與227守衛通過；遠端#263／#204原acceptance與labels保持。逐SHA與UID檢查本批目標及收據全部通過。清理前清冊已完整落盤，兩個未使用build釋放42031840 bytes；即使後續衛生檢查失敗，執行結果仍可回查。
- 衛生檢查初輪把整庫既存root-owned一律視為本批失敗，範圍過廣。改核對本批變更／證據UID1000，並另存2808個未觸及root-owned路徑；不批次chown。lstat共2044450項，無.md目錄。正式profile、候選終版binary、原始素材與有效收據保留，沒有錄影。

### 2026-10-27 戰鬥相同畫面重用

- 上輪9230d379已推送main並核對遠端；本輪從main建立feat/hd-battle-frame-reuse。沿規格61的READY等價重構，先驗完整計畫，再對同一成功畫面的所有消費輸入建立指紋；owner／theme／reset／close失效。不調時鐘、音效、規則或保存。
- 指紋可變輸入及五項效能觀測測試通過。首輪獨立目的畫布比較在非整數縮放邊緣出現16個色彩bytes差異，最大14；同目標強制重畫精確。共享來源texture仍有同樣差異，不能將差異直接歸因來源貼圖或重用。失敗測具原碼與log分別保存。
- 對照改為同一目的畫布與來源綁定下，實際重用及強制完整重畫逐byte精確；26組消費參數／切片原地修改／無效計畫／reset／close通過。原2087個來源GPU案例保留原生精確及HD容差1，另逐案驗證重用仍精確，正常GUI及完整Game.Draw／F2待本批後續收尾。
- 本批最終正常GUI274.559秒通過，首章同來源seed1／2CPU／6GiB／llvmpipe；完整物理窗203次合成、165次重用，約45%重複合成省略，Draw22.44Hz、更新51.09Hz。max事件101、無OOM／kill，保留記憶體壓力與平台限制，不宣稱固定加速或餘裕。效能模式明示未重驗走行PNG。
- 蓋亞正式圖源的完整生命週期185次HD／3次像素／253次最終繪製，64拍戰鬥淡出及65拍返回、F2及失敗預檢保持HP／RNG通過。新1172份Go來源重生21案例與前批相同，19反例拒收，16快取owner／17建構保持。唯一入口為HD主收據battle_frame_reuse_followup_20261027。私人素材與清冊未改，沒有錄影或新版Release；集中造型與平台仍待。
- 遠端#263讀回原acceptance／labels與OPEN保持，25項快照由官方工具更新、render／verify通過；371條教訓與228守衛通過。當批來源／收據逐SHA通過，147項本批變更與產物UID1000，無誤建.md目錄；沒有錄影或待刪影片。既存root-owned基線不批次改權限。實際攻防PNG已由代理目視核對，不代替集中人工驗收。

### 2026-10-27 HD解碼圖片唯讀共享

- 前輪d4a27e0a已推送main；本輪從main建立perf/hd-decoded-image-sharing。戰鬥圖片原已按需預備，沒有全2118幀常駐。最新6GiB觸限樣本含約4.54GB匿名記憶體與1.88GB檔案快取，不能把整個cgroup峰值都算成圖像。
- 同一正式來源、Go1.22.12的CPU配置實測，天空之鑰100／96張及代表攻防27張，共223份獨立圖片。共享後為158份；RGBA等價容量少150645888 bytes，回收後Go heap由985042368降為834142096 bytes。容量／heap與全遊戲RSS分列；前後測具條件與圖源一致，沒有另設GC參數或降低圖質。
- 依61的READY規格，在天空之鑰單庫冷載入、戰鬥單次預備交易內共享新解碼唯讀圖片；每路徑重新讀檔核SHA，每畫格仍驗尺寸／黑幕／透明度。發布原子性、未請求戰鬥圖庫釋放與像素SHA守衛保持。十組共享／錯誤路徑／改檔／形狀／黑幕／coverage控制通過；正常GUI與完整演出待本批收尾。
- 最終新增獨立來源與跨交易釋放控制後，共11項共享控制通過；天空之鑰三項CPU／12具名控制、完整202拍GPU／12次F2／一次續行，以及蓋亞真實圖源完整攻防／F2／返回通過。正常首章252.514秒、峰值6245777408 bytes、memory.events全零；沒有新增GC參數，仍只作當批軟體GPU觀測。
- 1173份Go來源重生21案例與前批相同、19反例拒收，16快取owner／17建構保持。唯一目前入口為HD主收據battle_decoded_storage_followup_20261027；私人素材、清冊、正式規則與保存未改，沒有新錄影或Release。
- 遠端#263讀回原acceptance／labels及OPEN保持；官方快照、render／verify與372條教訓／229守衛通過。當批程式與收據逐SHA、13張天空之鑰PNG解碼／1920×1200與三張正常截圖SHA通過，152項本批變更／產物UID1000，無.md目錄；既存root-owned基線不觸及。沒有錄影或待刪影片，私人素材與公開圖源保持。

### 2026-10-27 使用者暫停與提交交接

- 使用者要求暫停後整理進度、commit與push；Goal狀態已確認為paused。本批已完成解碼共享及驗證，後續維持暫停。
- 收尾已核對本輪來源SHA、遠端Issue快照與官方render／verify；程式與證據入口為HD主收據battle_decoded_storage_followup_20261027。平台驗收與全部HD後集中造型確認保留，#263維持OPEN。
- 本批測試／GUI容器均已退出，沒有新增錄影；驗證收據與必要圖片留本機work/，不加入Git。私人素材庫維持原狀。

### 2026-10-27 恢復HD1並補狀態致死正常操作

- 依使用者「繼續」恢復HD1，從main建立feat/hd-status-death-gui；既有Go程式及正式素材保持13e8196a的已驗版本。
- 建構工具只在啟動前改首章非隊長身分4的HP及原生暫態六byte，差異清冊可重生。正常Z選角、正式可達格退避／待機、END／YES後，由selector2扣血及status死亡owner返回；完整HD與途中F2分兩次，章內無狀態寫入或鎖血。
- 獨立核對2張A／B不透明來源核心，26項反例拒收；既有章23普通擊殺三張核心回歸通過。章界保存bytes及新程序標題LOAD一致，不宣稱保存章內死亡、正常取得中毒、自然難度、整章E2或新DOS對拍。
- 首次錯誤地圖目錄屬腳本問題，改讀campaign節點；第二次敵軍先普通擊殺，保留57筆action audit並明確停止，未當成status收據。第三次正常退避後通過，231.4秒、峰值5474689024 bytes，memory.events全零；僅限本批軟體GPU條件。
- 唯一入口為HD主收據status_death_normal_gui_followup_20261027及58／61。回復術current_state的舊「正常待驗」一併改指既有2026-10-26收據，避免重新開已驗工作。私人素材、清冊與發布版本未變，沒有錄影；集中造型及平台仍待。
- 最終遠端#263的acceptance／labels及OPEN讀回保持；官方快照render／verify為25項、18項人工、無消失訊號，372條教訓／229守衛通過。32個來源及證據SHA、15張PNG解碼與本批輸出UID1000通過，沒有誤建.md目錄。實際A／B視窗已由代理程式目視抽驗，不代替集中人工批准；全部FD2容器已退出。

### 2026-10-28 HD接線與使用者人工驗證交接

- 使用者明示「HD接線完成即可，剩下我人工驗證」。同步AGENTS、規格61及現況58／56／57：造型、逐章及平台人工試玩交由使用者，接線保留必要載入、缺件、建置與切換／保存相容性。未製作地形、其餘介面、獨立控制與HD2／HD-2D仍屬實作，不改稱人工工作。
- 既有正式清冊及私人包全量核對通過；全部264組已解碼FIGANI的2118幀皆已登錄，23組FDOTHER效果、10組LMI及86背景／台座來源保持。來源408保留既有不能解碼的限制，不開新RE或猜補caller。受控21消費端與19個反例核對通過，不外推所有可達性或原版parity。
- 8項必要Go載入／原子性檢查分兩批取得PASS、零SKIP。首批七項通過，最後全地圖載入因420秒整批預算耗盡，僅拆出該項補驗，84.191秒通過；兩批無OOM。沒有重錄各場GUI、調效能參數、修改正式Go程式、美術、私人庫或發布版本。既有F2／保存冷讀收據沿用。
- 首次清冊工具因未掛至catalog指定private_root而找不到檔案，列容器來源問題。按原路徑唯讀掛同一私人包重跑通過，不當成產品缺陷。必要檢查與逐SHA收據見HD主收據hd1_wiring_handoff_20261028。
- 遠端#204／263／268／269已同步使用者人工分工，labels與OPEN保持；官方快照render／verify及373條教訓／230守衛通過。來源與證據SHA、輸出UID1000及本次新增連結均通過；沒有錄影、私人素材變更或額外FD2容器。
### 2026-10-28 HD1里程碑與獨立控制接線

- 使用者將地形、操作介面、獨立控制與完整單列工具列定為HD1 milestone。建立GitHub milestone1與#282～285，官方遠端快照更新為29項，沒有以人工移交關閉實作缺項。
- 從main建立feat/hd1-independent-controls。F2只改美術、F11只改操作介面，任一HD保留1920×1200呈現；人物圖源依美術，字體／框線／操作圖示依介面。既有活躍owner沿F2預檢重備，錯誤不發布半套。原版美術的狀態／資訊窗與穩定HUD另補介面接點，模態HUD及其餘UI仍待。
- 工具列接四語、A+、1／2／3／4倍速與全螢幕；倍率以連續遊戲時鐘及Ebiten TPS實作，音訊沿真實音高／完成旗標。設定改成先載入再更新單一欄位，修正語系及音源抹掉字級／介面／美術的問題。戰役存檔不加顯示設定。
- 原生固定字區與確認按鈕放大後按實際字界縮入原矩形，包含日文否定標籤、陰影與選取下移；移動裁切不再影響排版字級。34項頂層Go通過，峰值1747165184 bytes，無SKIP／逾時／OOM。字串審查重綁為128項，移除六個舊標籤定位，加入三個原版標籤、縮略符及效能診斷分類；既有語意分類保持。
- 浮出列32步實際滑鼠／鍵盤與列外GPU保持、四組合、四語、字級、倍率及全螢幕通過；13780 bytes保存保持。固定模式依正式啟動的明示介面模式重跑10步通過。初輪時間預算及舊測具失敗原件保留，不當成正式產品或平台結論。沒有重新錄製章節或效能校準。
- 最後1179份Go來源重生21案例與416個archive來源，19反例拒收；Linux建置及字串盤點測試通過。現況及精確命令集中HD主收據hd1_controls_followup_20261028。README更新實際工具列半縮圖，如實標示城鎮行走人物仍像素。
- 地形、其餘介面／模態HUD與海報仍是實作工作，#282～285維持OPEN；HD2／HD-2D另期。使用者接手人工造型與平台試玩，私人素材／清冊未改，沒有錄影或新Release。
- 收尾核對1179份目前Go來源、17筆來源／收據SHA、README本地連結與233項本批來源／輸出擁有權。29項工作清單無消失訊號，373條教訓／230守衛通過；本批FD2容器均退出，私人庫乾淨。

### 2026-10-28 HD1雙側海報與工具列收尾

- 從079368a1建立feat/hd1-poster-controls，沿已核准兩組原稿及61的READY規格接入正式Go。工具列補海報下拉設定、F10、0～96內緣滑桿、四種真實時間淡入與重播。兩側各132邏輯像素，中央保持原比例；偏好獨立保存，戰役格式不變。
- 背景只從乾淨設施圖或原生／HD地形來源重組，排除人物、範圍、對話及HUD；沒有來源時用深藍底。滑鼠／觸控與設定區擁有輸入，兩側拒絕地圖輸入；來源缺失、SHA及不完整組別保持原狀與設定。
- 正式853項、私人7065檔完整驗證，舊7063檔SHA保持；兩張批准原稿未改像素，來源與裁切契約留清冊。私人庫a6513f06已推送並核對遠端HEAD，仍為private；公開庫只保留既有批准原稿與總覽縮圖。
- 32項頂層回歸含八個字串測試、Linux建置通過。正式包海報載入與13種拒收通過；76組完整Game.Draw／最終GPU、原生及HD乾淨地形通過。36步真實輸入、72個按下／釋放邊緣、四語、四倍速四組介面／美術及13780 bytes戰役保存保持。GUI142.826秒、峰值5904281600 bytes，7GiB下events全零；不宣稱整章、原版或平台驗收。
- input-r1在第15步F10未切換，舊測具未記錄實際邊緣，原因無法回溯。本批未改遊戲判斷，改由測具確認按下／釋放後才送下一事件，r3通過。r2缺FD2_ASSET_PACK在載入前失敗，補齊掛載後乾淨重跑。既有Go image被清理後，沿原Dockerfile重建同名工具鏈，沒有另建重複image。
- 最後1185份Go來源重生21案例、416張archive來源及19反例通過；155個字串處置含27個新海報標籤，沿官方遷移工具保存既有分類。現況、SHA與重跑集中HD主收據hd1_posters_followup_20261028；README使用實際GUI完整縮圖，城鎮人物仍為像素。
- #285工具列接線與必要檢查完成；#282地形、#283其餘介面及#284完整組合仍是實作工作。使用者接手人工造型與實機操作，HD2／HD-2D另期，沒有新Release。沒有錄影或待刪影片。
- 收尾核對1185份目前Go來源、24筆收據／來源SHA、338個README本地連結與7336個本批路徑UID/GID1000；無誤建.md目錄。28項工作清單無消失訊號，374條教訓／231守衛通過；本批FD2容器已退出，沒有錄影。

### 2026-10-28 HD1全地圖來源盤點與同源接線

- 從f7fbe7b1建立feat/hd1-terrain-coverage。沿61的READY規格擴充既有Go圖源工具：正式selector、33圖真實bank及事件writer共用，事件全有效時只作來源上界。生成覆蓋資料已接58／61與README，不另建工作清單。
- 原生8256幀、每層4755456像素獨立回讀通過。新盤點分出美術與合成缺項，避免把部分覆蓋／目的地重映射誤稱只差畫圖；不重開原版RE、逐章或已驗戰鬥。
- 11圖42地形／4前景槽三層完全相同且HD成品唯一，補登錄現有PNG。候選與正式864項、私人7065檔通過，原實檔及private-manifest bytes保持；沒有新圖片或重複素材。私人庫661cd99a已推送並核對遠端，仍為private。
- 全部新槽184個正式來源producer／GPU對獨立CPU、四種色盤，以及map1實際資料／章2單位的兩個完整Game.Draw通過。人物與鏡頭屬明示受控設定，測屋頂遮擋；原版／HD介面及程式F2往返、單位／RNG與保存bytes保持，不宣稱真實鍵盤、整章或平台驗收。GPU120.724秒、峰值6623518720 bytes，7GiB下events全零。
- 28項頂層Go、Linux建置與字串回歸通過；三個Python測試含九來源反例、四登錄反例、拒覆寫與完整重編通過。gpu-r1是測具未使用import，gpu-r2缺原生frame banks；補齊後先跑1.28秒CPU預檢，再乾淨重跑r3。正式產品Go未改，所有新增呈現沿既有正式consumer。
- 目前1188份Go重生21個戰鬥受控案例、416張archive來源及19反例通過。主收據為hd1_terrain_reuse_followup_20261028；README新增完整、未裁切的受控map1畫面，明示其他素材與部分介面仍為像素。#282、#283與#284保持OPEN，#285保持CLOSED。人工造型與實機操作由使用者接手，HD2／HD-2D另期，無新Release或錄影。
- 收尾核對1188份Go、18筆來源／收據SHA、340個README本地連結及7126個本批路徑UID/GID1000，無誤建.md目錄。28項工作清單無消失訊號，375條教訓／232守衛通過；本批FD2容器已退出，沒有錄影。

### 2026-10-28 HD1來源遮罩與洞窟石壁

- 從6ae3b554建立feat/hd1-terrain-compositing。沿既有原生blit證據及61 READY規格，確認部分來源的阻擋是HD載入器過度不透明限制；刪除該斷言，來源／目的地mask須二元互斥，HD透明區保留已完成合成的原生或已登錄HD背景，不重複LUT。不建立通用HD work鏡像，未HD背景仍揭露。
- 全306部分遮罩／純目的地來源612個GPU案例、獨立CPU及透明區精確檢查通過。內建image_gen依原生4×4參考繪製洞窟石壁，實際母稿1254方形完整保存；原生覆蓋保持，新RGB補邊與形狀是美術近似。最初小圖誤認瀑布已核圖修正，未將錯誤用途接入正式資料。
- 16張map9:32～47實圖、32來源與兩組完整Game.Draw／F2、單位／RNG及166 bytes邊界保存保持通過。actual-r1為測具缺HUD／range初始化，補齊後乾淨r2通過；原生規則未改。r2耗時100.169秒，峰值4171952128 bytes，7GiB下events全零。synthetic批212.231秒、峰值2196570112 bytes，4GiB下events全零。
- 正式866項與私人7082檔驗證通過，舊7065檔SHA保持，只增加母稿及16成品。私人庫45169fa2已推送並核對遠端HEAD，仍為private。最後六個Python頂層測試含來源／登錄拒收與固定基底重編通過；修正圖片handle釋放後無ResourceWarning。
- 26項頂層Go、既有地圖23背景／原生映射、獨立介面／海報、字串及Linux建置通過。1189份目前Go重生21個戰鬥受控案例、416張archive來源及19反例通過。覆蓋JSON由Go重生，509槽已HD、3479缺美術、10純目的地來源另列；新22個同源可重用槽留下一批驗證。
- 目前唯一入口為主收據`hd1_terrain_partial_followup_20261028`、58與61。README新增完整未加工洞窟實際畫面，明示相機受控、章10單位，熔岩及地板仍為像素。#282／283／284保持OPEN，#285保持CLOSED；使用者接手造型與實機操作，無新Release或錄影。
- 收尾核對1189份Go、22筆來源／收據SHA、943個README及本次規格本地連結與7134個本批路徑UID/GID1000；無誤建.md目錄。28項worklist無消失訊號，375條教訓／232守衛通過。本批FD2容器已退出，無錄影。

### 2026-10-28 HD1洞窟石壁跨圖接線

- 從b3bda747建立feat/hd1-cave-cross-map。依61 READY同源契約，補map24:32～47及map27:100／102／104／108～110，共22個地形槽。來源三層、相同角色與selector見證均核對，沿map9現有PNG；不新增圖片或背景RGB，不改規則。
- 既有GPU測具增加明示EXPECTED=22與map24／章25完整場景；46槽舊測具預設及來源／登錄反例保留。88來源GPU與兩組完整Game.Draw、程式F2、單位／RNG及169 bytes保存保持通過；受控鏡頭與第一單位位置，不宣稱正常玩家或平台。GPU154.477秒、峰值5197500416 bytes，7GiB下events全零。
- 正式868項、私人7082檔與完整SHA／UID/GID驗證通過，圖片及private-manifest bytes保持。私人提交af05b4d5已推送並核對遠端，仍為private。兩批六個Python頂層測試各含九來源及四登錄反例、完整重編及既有輸出拒覆寫通過。
- 27項Go、地圖23背景、獨立介面／海報、字串及Linux建置通過。1189份Go重生21個戰鬥來源案例、416張archive來源及19拒收反例通過。Go生成資料為531槽HD、3457缺圖、10純目的地來源，無剩餘已知同源可重用槽。
- README換為完整未加工map24實際畫面，保留前批證據圖片；熔岩、地板及部分介面仍為像素。現況入口為58／61及HD主收據`hd1_cave_reuse_followup_20261028`。#282仍OPEN，完整HD地形、其餘介面及組合繼續，人工由使用者接手，HD2／HD-2D另期。無Release或錄影。
- 收尾核對1189份Go、19筆來源／收據SHA、943個README／規格本地連結與7135個本批路徑UID/GID1000。28項worklist無消失訊號，375條教訓／232守衛通過；無誤建.md目錄或非預期FD2容器，沒有錄影。

### 2026-10-28 HD1洞窟地板與59槽接線

- 從8fc04ea7建立feat/hd1-cave-floor，依61 READY規格補map9:80～95共16張地板，另接map24／25／26的43個同源槽。原始frame PNG只作索引層，彩色來源atlas核圖後才定案80～95；不把圖號31光柱當地板。新母稿由內建image_gen製作，逐字提示、1254原稿、配方及SHA保存；RGB只取新稿，分格與明暗屬美術近似。
- 地板32來源GPU與兩組map9完整Game.Draw／F2及保存保持通過；跨圖172來源GPU／四種色盤與兩組map24完整Game.Draw／F2及保存保持通過。單位／RNG保持，鏡頭與map24第一單位位置明示受控，不新增整章、正常玩家、原版oracle或平台驗收。兩批memory.events全零，時長與峰值依主收據。
- 洞窟工具保留石壁分支，新增明示floor契約；同源工具同圖追加集合以新槽內容摘要建立獨立身分，不改舊集合。兩類工具的新舊四批共12項Python頂層測試通過，含完整重編與來源／GPU／保存拒收；候選後只修正floor登錄狀態文字，圖片保持。
- 正式873項／私人7099檔通過完整SHA、UID/GID及清冊驗證，舊7082檔保持；只新增母稿與16成品。私人提交20d874763ebcb981ecac67df5c87e6a0c6179ee2已推送並核對遠端，仍為private。
- 27項Go、背景與地形來源／載入／時鐘、獨立介面／海報、字串與Linux建置通過。1189份Go重生21個戰鬥來源案例、416張archive來源及19反例通過。Go生成覆蓋資料為590槽HD、3398缺圖、10純目的地來源，沒有剩餘已知同源可重用槽。
- README保存完整未加工map24新地板畫面，前批圖片保留。現況依58／61與HD主收據`hd1_cave_floor_followup_20261028`；#282仍OPEN，其他地形、熔岩、介面及完整組合繼續，人工由使用者接手。無Release或錄影。
- 收尾核對1189份Go、44筆來源／收據SHA、944個README／規格本地連結與14285個本批路徑UID/GID1000；新配方已掛61入口，逐字提示一致。28項worklist無消失訊號，375條教訓／232守衛通過；無誤建.md目錄、非預期FD2容器或錄影。

### 2026-10-28 README的HD地圖總覽

- 使用者追加要求總覽直接放README。從768ad313建立feat/hd1-map-overview，先完成此審閱交付；沒有新增本輪地形素材或改動正式遊戲。依61 READY規格，map0／1／9／24／25／26從同一正式清冊重新擷取完整Game.Draw，鏡頭按HD地形覆蓋取景，不改人物位置。
- 六組GPU對獨立CPU、程式F2與各圖邊界保存保持通過，188.841秒、7GiB峰值6325538816 bytes，events全零。章節JSON採直接進場名冊，首章含蓋亞；已核實來源且在圖說明示，不把受控畫面當正常加入歷程。
- 總覽為六張1920×1200完整PNG等比例縮至640×400，排版1352×1662，保留像素區域。Noto Sans CJK TC的15個實際字形墨跡安全區與五個來源拒收反例通過。圖下槽數由正式coverage重算，含事件上界，不宣稱整圖完成率。新擷取與總覽工具已掛61入口。
- 1190份目前Go重生21個戰鬥來源案例、416張archive來源及19反例，八項字串回歸通過；正式873項／私人7099檔保持。現況與重跑集中HD主收據`hd1_map_overview_followup_20261028`。
- 暫存字型work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/hd1-overview-font.ttc的SHA b76b0433203017ca80401b2ee0dd69350349871c4b19d504c34dbdd80541690a及大小19484784 bytes已保存，刪除後保留可從既有Go image重取的來源。六張完整PNG與收據保留，沒有錄影或新封包。
- 收尾核對1190份Go、38筆來源／收據SHA、18份未修改的地圖／章節／名冊輸入、948個README／規格本地連結與28個本批路徑UID/GID1000。28項worklist無消失訊號、375條教訓／232守衛通過；無誤建.md目錄、非預期FD2容器或錄影。

### 2026-10-29 洞窟地板明暗變體與同步總覽

- 從e0d7ebf5建立feat/hd1-cave-floor-details。依61 READY規格與已核原生彩色圖，內建image_gen產生2×2新母稿，map9:74／75／78／79四張圖接map9／24／25／26共12槽。只共用相同角色、三層原生bytes與selector見證，不改規則或幾何。原稿、逐字提示及SHA保存於新配方與私人庫。
- 為避免反覆冷載主題，把48個原色／LUT／黑／白來源及六個完整Game.Draw合併驗證；184.564秒、8GiB峰值5159514112 bytes，events全零。六次程式F2、單位／RNG及保存保持通過；受控相機與直接進場章節名冊不提升普通玩家或原版oracle。原包與org_game明示唯讀掛載。
- 新來源三項與舊地板三項Python回歸通過，包含六種來源及五種登錄拒收。候選後只加嚴來源格式與母稿角色守衛，最終重編PNG及catalog.assets保持，原GPU收據不改寫。
- 正式878項與私人7104檔通過SHA／UID/GID1000驗證，舊7099檔保持，只增母稿及四成品；私人提交42ff40e747078eafe2b8f061b2d91ddf1e768d5a已推送並核對遠端，仍為private。27項Go、Linux建置與1190份目前Go／21個戰鬥案例／416張來源／19反例通過。
- 總覽工具新增嚴格正式搬移等價：兩清冊除private_root、兩覆蓋除清冊引用外必須全等。舊總覽重編逐byte相同、新總覽七個拒收通過；以同份候選已驗PNG更新README，未冒稱正式重拍。新覆蓋為602槽HD、3386缺圖、10純目的地來源。
- 現況與重跑依58／61及HD主收據`hd1_cave_details_followup_20261029`。核SHA後刪除暫存字型及舊總覽重建複本，共21306532 bytes；六張來源PNG與收據保留。#282／283／284仍待，人工由使用者接手，沒有錄影、三平台重打或新Release。
- 收尾核對1190份Go、26筆來源／收據SHA、54個GPU案例與六張完整PNG、951個README／規格本地連結與7171個本批路徑UID/GID1000。新配方已掛61，逐字提示一致；28項worklist無消失訊號、375條教訓／232守衛通過。無誤建.md目錄、非預期FD2容器或錄影。


### 2026-10-29 洞窟外緣石地與總覽

- 從b3ef9611建立feat/hd1-cave-rim。先核原包manifest、色盤、selector及map9:0～15／map26:80～95三層一致，依61 READY規格以內建image_gen產生4×4新母稿，新增16張144圖接32槽。新RGB延拓後以原生索引0恢復不透明黑洞輪廓，不把黑洞當透明背景。
- 新批三項、舊明暗三項與舊地板三項Python通過；原生黑洞逐像素核對、九種來源與五種登錄缺項拒收。第一次模組式啟動缺少tools搜尋路徑，未進測試；改用原檔入口及同容器重跑通過，正式程式／來源不因環境失敗改動。
- 128來源GPU及六組完整Game.Draw／程式F2／單位、RNG與保存保持通過。GPU195.015秒、8GiB峰值6605848576 bytes、events全零；首批26項Go與Linux建置60.142秒、4GiB峰值1466249216 bytes、events全零。同源交易首次缺明示開關而跳過，設FD2_HD_TERRAIN_REUSE_CPU=1後只補該項並通過，合計27項，獨立收據為hd1-rim-transaction-r1。1190份目前Go重生21戰鬥案例、416來源及19反例通過。
- 正式881項清冊與私人白名單同步已驗，舊素材逐SHA保持，只新增母稿及16成品。私人提交／遠端HEAD與認證可讀、匿名拒收及公開庫正對照，保存本機hd1-rim-private-git-r1.json；公開進度不複製私人提交資訊。
- README六圖總覽使用同份已驗候選PNG；正式清冊及覆蓋除private_root與清冊引用外全等，七個總覽反例拒收。受控鏡頭／章節測試名冊仍屬RUNTIME-E1，不新增正常玩家或原版oracle聲明。
- 核SHA後刪除暫存字型及全洞窟診斷索引圖，逐項記於hd1-rim-cleanup-r1.json；必要參考、六張完整PNG及收據保留，無新錄影、封包或Release。現況依58／61及HD主收據hd1_cave_rim_followup_20261029；#282／283／284保持，造型與平台人工由使用者接手。
- 收尾核對1190份Go、30筆來源／收據SHA、134個GPU案例與六張完整PNG、957個README／規格本地連結及7201個本批路徑UID/GID1000。編製器SHA與候選一致，新配方已掛61，逐字提示一致；28項worklist無消失訊號、375條教訓／232守衛通過。無誤建.md目錄，既有root-owned基線未改。

### 2026-10-29 一般系統確認的HD接線

- 從8640e7b1建立feat/hd1-system-confirmations，依61 READY切片接五個原生確認owner。原問句／回覆與交易保持，人物隨美術，字型／細框／按鈕隨介面；問句未新增翻譯。十拍開框、四拍收按鈕、逐字回覆及五拍關框沿原時序。
- 原生核對DATO75為棕髮雙辮店員，原清冊沒有對應HD；沿已選手繪風格補一張母稿，產80／480。母稿與逐字提示保存於配方與私人白名單；原生0／3嘴形與亂數保持，HD為閉口靜態，不宣稱嘴形逐幀一致。
- 147完整Game.Draw、15張PNG、五次途中F2／F11及世界／單位／保存保持通過。GPU289.267秒、8GiB峰值6573273088 bytes，events全零；獨立核對3456000頭像像素及11種反例。README新增1352×1136四組總覽。
- 三項Python回歸包含四種來源與七種登錄拒收；候選後加嚴登錄守衛，重編證明PNG與catalog.assets不變。15項原生與介面回歸、41個不同頂層Go、21個正式消費端案例／416來源／19反例及Linux建置通過。目前1193份Go來源指紋集中HD主收據`hd1_system_confirmation_followup_20261029`。
- 首批字串審查因行號位移失敗，以Git基底重生舊inventory，官方遷移工具按相同函式／字串保留原審查及rationale。只補跑八項字串、coverage及build，全通過；第一批不整批冒稱綠色。測具缺字模／campaign及DATO75缺圖原因分開保存。GPU R2因重複預檢有界停止，不算通過；R3維持147案例與容差。
- 普通驗證首次因Go測試映像缺xdotool，尚未進遊戲。清理後原promo映像已不存在，沿既有Dockerfile恢復同名工具鏈；相同參數R2從標題LOAD第23章來源，普通END、四組切換、NO自然返回與保存bytes通過。來源章前強化政策明列，章內無注入、無新戰鬥或冷讀聲明。
- 正式883項與私人白名單同步，舊7121檔SHA保持，只增加母稿及80／480三圖；Git與可見性核對留本機收據，公開進度不複製私人提交資訊。地形內容與前批完全相同，只重生coverage的清冊指紋；六圖地圖總覽保留。#282／283／284仍OPEN，人工由使用者接手，HD2及HD-2D另期。無新Release或錄影。
- 普通GUI168.643秒、8GiB峰值5026856960 bytes，events全零；另核五組視窗頭像1152000像素。核SHA後清除暫存字型、十張與R3逐byte相同的R2圖片及舊Go來源tar，共31859642 bytes；必要圖源與收據保留。收尾核對43筆SHA、3049個本地連結與7178個本批路徑UID/GID1000，既有root-owned基線2808不變，無誤建.md目錄。28項worklist無消失訊號、375條教訓／232守衛通過；無非預期FD2容器。

### 2026-10-29 HD1戰場大頭Q版方向與亞雷斯原型

- 使用者指定以原版造型優化整套HD1小人物，維持大頭Q版；索爾與悠妮可作參考，其他角色不符合期待，亞雷斯優先。從b928ff89建立feat/hd1-chibi-map-refresh。共同決策與圖片生成技能用於新比例原型，沒有把普通繼續當作新造型核准。
- 由首章名冊確認亞雷斯native_identity4／group4，清冊有125可見圖組、1455幀。原版分離PNG儲存色盤索引，第一次參考誤用其灰階palette；已依既有總覽工具的FDOTHER0代表色盤重建r2彩色參考，r1不作配色證據。原版與私人素材均唯讀。
- 內建image_gen生成十二格新稿。首稿右向誤畫兩把武器，第二次生成修成單一武器；不宣稱編輯時前九幀bytes保持。較大頭初稿展示144／72／48與索爾／悠妮對照後，使用者明選頭再大、身體再短，第三稿將頭臉再放大、身體縮短。現行母稿SHA3246972a818d1edb1570469ff3572678c4533660ae66a43dce95d51ced21f9ab；完整提示與來源在hd1-ares-q-prototype-r2/prototype.json。
- 新舊兩版都沿既有master_components／compile_group編出十二幀，完整輪廓、144尺寸、透明核心、既有腳底與比例編製檢查通過。更大頭對照圖SHA dffa304cb1c675cbf4005b4cb8d1c878ce7d47c030eaf2c4ceca99e81714a12e；這是素材原型，不是Game.Draw、方向全驗或正常玩家證據。最新比例等待使用者選定，未替換正式清冊、引擎、私人包或存檔。
- 依官方fd2_worklist_issues.py cmd_new做純欄位檢查與格式化，再由主機gh建立#286，分層data／類別缺陷。遠端Issue為權威，快照由正式pull函式依主機gh讀取結果產生。61記錄DRAFT及原型入口，58區分新造型待確認與既有技術證據；不以舊總覽宣稱美術完成。
- 收尾核對兩版24幀、30份本批檔案SHA及UID/GID1000、1255個規格／工作清單本地連結；正式883項清冊與私人庫保持。暫存Noto字型與錯誤灰階參考已核SHA後刪除，共19633099 bytes；彩色參考、兩版原型及完整提示保留。無錄影或新發行，未做遊戲替換，因此不跑與本次原型無關的Go建置或整章回歸。

### 2026-10-29 定版大頭Q版亞雷斯接入遊戲

- 使用者同意v3更大頭短身原稿，要求後續都用相同方式處理。將定版對照原樣掛入版控，AGENTS／61記錄持續標準；原身份與新疑義先查證，不再重問已定比例。61 READY後才編製正式候選。
- 新工具沿原版FDICON及代表色盤、完整輪廓與既有腳底，輸出版本化24父項及144十二幀；只替換兩個清冊項目，其他881項與舊檔保持。三項來源回歸、八拒收及登錄後舊清冊白名單重編逐byte通過；舊compile_group有ResourceWarning，未改像素或把它當遊戲缺陷。
- 首輪GPU測具逐案完整F2預檢過重，有界停止並留診斷。只改測具以保留24畫面／十二姿態／兩種介面，四向錨點做F2；舊測試預設行為保持。R2為513.231秒、8GiB峰值6996967424 bytes，events全零，單位／RNG／保存保持通過。
- 獨立核對器初版只取alpha255，錯漏多為253的有效核心，並把HP合法覆蓋列差異。依實際alpha統計與既有HUD相對16列契約，改SrcOver界限及前96列，完整GPU容差2保持。180225近不透明點與十種反例含真黑PNG通過。
- 普通GUI首次1080秒選項不受支援，未進遊戲。沿既有900秒預算、同程式／來源R2通過：首章標題LOAD、選亞雷斯、四組呈現、合法走行與取消、F5及新程序LOAD；355.962秒、7GiB峰值5978558464 bytes，events全零。本輪無章內狀態注入；來源名冊沿既有JSON，不新增原版自然加入或整章聲明。
- 正式清冊21個頂層CPU／字串測試及Linux建置通過；舊的專用全圖庫HUD GPU未啟用不算通過，本批HUD另有24案例及普通視窗實圖證據。目前1194份Go、21個戰鬥來源案例／416來源／19反例通過。唯一收據與重跑依HD主收據`hd1_chibi_ares_followup_20261029`。
- 新母稿及24成品已白名單同步私人庫，舊已登錄檔SHA與UID/GID保持；私人提交／遠端及匿名不可讀有本機收據，不在公開進度複製私人提交資訊。地形內容不變，只重生清冊指紋。README新增定版尺寸與普通取消後完整1920×1200原圖，舊六圖總覽明示前版人物。
- 核SHA後移除五張與通過PNG相同的停止輪次圖片，共11454596 bytes；定版原稿、普通PNG及診斷保持，沒有錄影或Release。#286納入既有HD1 milestone，移除過期等待比例斷言並保持OPEN；其餘人物／轉職、地形／UI續做，集中人工由使用者接手。
- 收尾再次核對1194份Go、38筆來源／收據SHA、3088個本地連結及147個本批路徑UID/GID1000；完整圖與普通GUI來源再驗通過。29項worklist無消失訊號，376條教訓／233守衛通過。私人庫已推送且仍不可匿名讀取，正式與私人清冊相同；沒有錄影或新發行。

### 2026-10-29 HD1全人物總覽收尾要求

- 使用者指定全部HD1人物重繪完成後，再重生「原版／HD1」人物總覽並更新README。將要求寫入AGENTS、61及遠端#286同一驗收條件，保留人物、轉職與敵方範圍及144／72／48像素分組展示。
- 既有總覽工具仍讀固定舊檔名與24像素父圖，規格明示收尾時改為從正式map_sprite_hd_set與來源SHA取最新圖。此輪只更新交付要求，不提前重生混合新舊圖、不變更正式清冊或私人素材，也不重跑無關遊戲測試。
- 遠端#286讀回保持OPEN且新驗收條件一致；官方pull／render／verify通過，29項未完成、零消失訊號。本輪660個本地連結及五檔UID/GID1000核對通過。純交付要求調整留在工作樹，未建立新提交或發布；一次性Docker結束，無錄影。

### 2026-10-29 HD1哈諾、鐵諾與哈瓦特接線

- 從cd787716恢復並保留前輪全人物總覽要求，建立feat/hd1-chibi-party。沿61 READY及使用者已定比例，核原版圖組1／2／3與代表色盤。使用內建image_gen重繪三組，鐵諾首稿錯劍位與兩份多餘手臂局部修稿拒收，最後依原版重生；不宣稱局部編輯保留其餘幀bytes。選定三原稿與逐字提示完整保存。
- 同一編製器產生36張24父圖與36張144圖源，六項替換、877項保持。三項回歸核全部72PNG逐byte重編、八種來源拒收及拒覆寫通過。第一次out相對路徑被工具拒收且scratch移除，改絕對路徑乾淨通過；舊compile_group的ResourceWarning記為工具生命週期提示，不當成產品缺陷。
- 完整測具初輪在Go package cwd找不到新場景，未入繪圖。改為既有../../assets，先三組建構預檢通過再r2重跑。21個完整Game.Draw、18張PNG、四向中性及正面三拍／三次F2、單位／RNG／保存保持通過；142.315秒，8GiB峰值8056541184 bytes、events全零。獨立核241359近不透明核心點，九反例含真黑PNG拒收。Harvat只投影Hano範本圖組／身份，不用於能力值或自然加入結論。
- 普通首章來源未改，只含原有四人；正常LOAD、亞雷斯走行取消、四組控制、F5及新程序LOAD通過，375.208秒、7GiB峰值4515057664 bytes，events全零。只證明本批完整候選載入及交易相容，不冒稱新三人正常出場。22個CPU／字串頂層測試通過；舊全圖庫HUD專用GPU未啟用不計通過，實際新人物HUD另由本批21案核圖。正式Go未改，沿用已核SHA的相同Linux程式，沒有重建相同production程式或新Release。
- 登錄前消費端收據綁前版正式清冊，拿新清冊核對被拒收；重新由1195份目前Go匯出綁定新清冊的21案例／416來源，19反例通過，沒有改寫舊收據雜湊。地形由Go重生、內容保持，只變正式清冊指紋。
- 正式883項與私人包全量驗證通過，先前7149個實檔SHA保持，新三母稿與72成品已白名單同步與推送，私人庫乾淨且仍拒絕匿名讀取；細節只留本機收據。README新增本批清晰分組圖，完整原版／HD1總覽仍待全部重繪後重生。唯一入口為58／61及HD主收據`hd1_chibi_party_followup_20261029`，#286及既有地形／介面／獨立控制缺項保持OPEN。
- 初版分組圖樣本跨列重疊，核SHA後刪除，由受版控工具的r2圖取代，共443399 bytes；原稿、正式PNG、診斷與收據保持。無錄影；集中造型、逐章及平台人工由使用者接手。
- 收尾核對1195份Go、47筆來源／收據SHA、3096個本地連結及235個本批路徑UID/GID1000；21完整圖與普通相容收據再核通過，沒有誤建.md目錄。遠端#286保持OPEN，官方29項worklist零消失訊號，376條教訓／233守衛通過；私人庫乾淨且遠端一致，本批FD2容器已全部退出，無錄影或新發行。

### 2026-10-29 哈瓦特成熟造型與古金框定版

- 使用者修訂Harvat要老成、有威嚴、大鬍子。內建image_gen沿v1與原版參考一次修成v2，原稿SHAe565358f19e65e5276f5e3f7a3d07ac5b6956612c9d988fd0b5cfa7f92f43c6c原樣保存；不推定劇情年齡。十二張24父圖與十二張144圖源重編通過，只換兩項、881項保持。
- 核對器支援明示單組配方，完整21案不減；舊三組預設與九拒收回歸通過。三項來源測試核逐byte重編、八拒收與拒覆寫通過。分組工具context須與當前清冊逐欄一致，最初漏/modern掛載拒讀舊alias，補唯讀來源後重生三人圖，只更新哈瓦特。
- 21完整Game.Draw／18PNG、三次F2、地圖／HUD、單位／RNG／保存保持通過；獨立243196核心點與九反例含真黑圖。GPU298.213秒，8GiB峰值8589934592 bytes、max342、無OOM；不稱平台餘裕。普通首章相容303.617秒、峰值5360009216 bytes／7GiB且events全零；來源沒有Harvat，只驗完整候選載入及原有亞雷斯走行取消、四組控制／保存冷讀。
- 正式Go未改，來源仍1195份／相同d9fbebbc指紋，沿用前批22個CPU／字串回歸而非聲稱重跑。新正式清冊消費端由Go重生，21案例／416來源／19反例通過。原版規則、動畫時鐘及存檔不改，沒有新Release。
- 正式883項與私人包全量r2驗證通過，7224個舊實檔SHA保持，25個新版白名單素材已保存並推送，私人庫仍拒絕匿名讀取。第一次45秒全量流程逾時未出收據，先保存私人版本但延後公開提交；同輸入r2乾淨通過27.616秒，1GiB達上限max3261、無OOM，不算平台餘裕。
- 使用者新增HD1對話金框要求。以SVG／HTML疊既有完整HD畫面，FFmpeg librsvg呈現；初次Pillow缺件與ImageMagick SVG delegate失敗列工具環境，不列遊戲缺陷。獨立核兩幅21060個差異點只在原框邊帶，文字／頭像／地圖不變。使用者選定古金雙層框，下半部原圖SHA2251195701b96682199571bc9e0d456dc8bc9a6a539be59fbf6496724bc46187原樣保存；61轉READY，下一切片補故事與既有提示，F11介面／F2美術保持獨立。此時尚未改Gold runtime。
- README三人對照及Harvat完整受控圖同步，唯一入口為58／61及主收據`hd1_harvat_mature_followup_20261029`；#286其餘人物、#283金框接線與其他UI保持未完成。沒有錄影或可刪影片，集中造型及平台人工由使用者接手。
- 本切片收尾核對1195份Go、47筆來源／收據SHA、3104個本地連結及176個本批路徑UID/GID1000，無誤建.md目錄。官方29項worklist零消失訊號、376條教訓／233守衛通過；私人庫已推送且仍private。古金原型已READY，接續實作，未宣稱此切片已完成金框。

### 2026-10-29 HD1古金故事框接線

依使用者定版原型接原生FFEC／FFEE／FFED／FFEF與一般故事框，既有提示共用古金材質。F11介面獨立，物品／狀態面板保留原色；字區、深藍底、動畫時計、規則與保存格式保持。74完整受控案例、14張PNG與五反例、八種最終層、26拍獎勵及26次F2／10PNG、27個CPU與八字串回歸通過。155個字串審查只按同簽名重綁，處置數保持；目前1196份Go消費端21案例／416來源幀與19反例通過。

普通START到父王，以實際按下／放開收據驗四組呈現及續句，五張PNG與五反例通過。原樣首章來源另經標題LOAD、亞雷斯一格走行取消、四組切換、F5與新程序LOAD，實際保存SHA及隊伍保持。來源沒有哈瓦特，不作其自然出場證據。README換成完整父王實際畫面，原型與灰綠來源保留於歷史證據。

診斷保留：故事r1重複預備主題主動終止；r2不同GPU目的地的一般頭像分數邊緣差異，改同一畫布後74案通過。GUI r1的60秒視窗等待不足，改沿既有360秒明示上限；r2讀latest漏ACK，r3沒有等實際放鍵，r4連續Confirm重設標題閃爍；沿既有放鍵教訓與原標題狀態機修正測具，r5通過。這些不列產品缺陷。精確SHA與資源依主收據hd1_gold_dialogue_followup_20261029，沒有OOM／kill或錄影，不宣稱全部HD1、原版逐幀或平台效能。

本輪暫時舊版字串工作樹已刪除；私人素材與正式883項清冊未改，不另產私人提交。其餘地形／介面／人物與轉職仍列遠端worklist，#283／286保持OPEN。
- 本切片收尾核對1196份Go、61筆來源／收據SHA、3111個本地連結及417個本批路徑UID/GID1000，無誤建.md目錄。官方29項worklist零消失訊號，376條教訓／233守衛通過。


## 2026-10-29 HD1戶外五底材版本化刷新

- 使用者同意清爽手繪原型後，先完成草／土／沙／水／石路五母稿，沿既有編製與原生遮罩刷新87成品、21圖384槽。883項只換68項，其他815項逐欄保持；正式Go、規則與保存格式未改。
- 428來源GPU及六張完整Game.Draw／F2／保存保持、三項逐byte重編、九前置／四結構拒收、十二獨立反例與28頂層Go通過。舊partial GPU因未啟用而跳過，未計通過；本批GPU另有完整收據。現行1196份Go、21戰鬥來源案例／416幀與19反例重生。
- 普通首章LOAD、四組呈現、合法一格走行取消、F5與新程序LOAD通過；五張PNG固定頂列62格及保存實檔核對。來源槽保持089b077b…，沒有狀態注入；不擴張為整章或原版parity。
- 母稿與成品92檔已按白名單存入指定私人庫，舊7249實檔及inode保持。候選缺既有非runtime母稿使初次完整同步停止，改沿完整包重跑通過；首次消費端核對的catalog綁定不符，登錄後重生來源通過。兩者是工具輸入範圍問題，未放寬檢查。
- GPU354.626秒／峰值8120061952 bytes，正常333.490秒／5666205696 bytes，events全零；編製2GiB達上限max1797，OOM與kill零，沒有平台餘裕聲明。全部分析、編製、GUI與測試都在有界非root容器執行，沒有建立錄影。
- README六圖總覽及普通遊玩圖已更新；原型明示為概念圖。洞窟、物件與其餘缺圖仍由#282追蹤，人物由#286續作。完整SHA、命令、資源與診斷依HD主收據`hd1_painted_ground_followup_20261029`，收尾清理與最終Git驗證另接本節。

- 收尾：私人遠端HEAD與本機一致，匿名私人庫404／公開庫200正對照通過，未搬移憑證。3103個本地連結、README內部錨點與文化保存入口通過。核對總覽字型SHA後移除臨時字型19484784 bytes，保留來源、必要PNG與收據；無新錄影、無殘留FD2容器，本批輸出UID/GID保持1000。擁有權掃描另找到51個歷史root-owned圖片／研究資料，本批未寫入或修權。
- 提交檢查修正三個新工具各一個結尾空行，前版完整來源及精確一LF轉換另存tool-format-rebind-r1。以現行核對器重生independent-r3及十二反例，登錄歷史收據保持；Go及素材未變。


## 2026-10-29 蓋亞小人物重繪與125組造型初審

- 使用者要求蓋亞也修改、其他人物全數review，完成後更新README完整總覽。第30組沿原版灰綠硬甲與藍面罩重繪大頭短身，保留特殊機械結構；只替換兩項，其他881項保持，正式Go與玩法／存檔不變。
- 第二次內建image_gen產出定稿，第一稿中性在中欄不符原位而拒收；新稿第一欄中性、兩欄走行，原稿原樣保存。25檔版本化圖源已白名單保存私人庫，舊7341檔與inode保持。
- 24完整Game.Draw、十二姿態、資訊窗、四向F2及保存保持通過；獨立核173017核心點與十一反例，三項逐byte重編與八來源拒收通過。普通首章四組控制、Ares走行取消與保存／新程序LOAD保持；Gaia三張PNG共22434核心點核對。
- normal audit每兩次更新一次，首次只看擷取兩端漏掉中間idle相位而拒收。查明FD2_INPUT_AUDIT_INTERVAL=2及記錄器契約後，改核同方向／同矩形的三拍，保留原像素門檻並明示不逐幀；未改遊戲、未重錄。編製初輪漏prepare而在argparse拒收，修正同一工具命令重跑通過。
- 全125組已初審四向中性與正面三拍，靜態組保留三拍；1455來源SHA／尺寸／alpha另核。12組身份偏移及6組待核列新缺陷#287，包含Tino兜帽誤銀髮、人形轉職誤機械、Harvat無鬚少年與盾／武器偏移。69組需統一新比例，32組可保留候選仍待整套風格統一；數量依主收據，未稱全部重繪或1455幀逐幀人工完成。
- Tino經DATO_002及四向原生來源證實為灰兜帽；舊技術收據維持，造型忠實推定撤回。g48舊機械式圖說及g73馬形推定已撤回，不靠名稱猜補身份。README新增蓋亞局部對照／實際畫面，完整人物總覽依指示仍待全數重繪後更新。
- 21頂層Go含八字串、21消費端來源／416幀與19反例、完整883項實檔通過。消費端首次錯傳candidate清冊導致private_root SHA不同而拒收，改用正式清冊對同一匯出核對通過，未修改收據。兩個舊GPU未啟用而跳過，未計通過；Gaia完整GPU另有收據。
- GPU302.471秒／峰值7235362816 bytes／8GiB，正常325.324秒／4731174912 bytes／7GiB，events全零；沒有平台餘裕或新Release聲明。全部工作在有界非root容器，原始資料唯讀，沒有錄影。SHA、重跑、逐組評語與收尾依HD主收據`hd1_gaia_chibi_review_followup_20261029`；公開commit／push後的遠端HEAD核對保存在本批public-git-r1.json。

- 收尾核對3114個本地連結、README錨點、文化保存入口及三張新圖通過。移除未引用的中間審查輸出並將前／後相同12頁改為相同內容硬連結，SHA與必要證據路徑保持，共回收45557475 bytes；無錄影，本批輸出UID/GID1000。私人庫匿名404與公開庫200正對照通過，遠端HEAD一致。

### 2026-10-29 鐵諾兜帽修正與蓋亞厚重第二候選

- 從c038ec46建立feat/hd1-gaia-heavy-v2，依原版DATO_002及十二姿態，把鐵諾銀髮誤畫修回灰綠布兜帽；左向直劍維持身體遮蔽。兩輪內建圖片生成原稿、逐字提示與拒收原因依獨立配方，未以程式補畫。
- 鐵諾與不變哈諾／成熟哈瓦特共21完整繪圖、18PNG、三次F2與保存保持通過；獨立239645核心點及九反例、三項逐byte重編／八來源拒收通過。普通首章來源不含這三人，僅驗完整候選載入、四組控制、既有角色走行取消、F5及新程序LOAD相容，不宣稱鐵諾自然出場。
- 正式883項只換兩項，881項保持；25新檔已私人白名單保存，原7366美術檔的inode／大小／mtime／UID保持。私人提交13ac6229a322b989730568d9d7ba78d31681e79a已推送並核實遠端HEAD與private。正式11頂層Go通過，兩個未啟用專用GPU不計通過；正式21消費端案例／416來源／19反例、全125組1455來源、379教訓／233守衛與30開啟工單檢查通過。
- 使用者拒絕前批蓋亞渾厚感，第一份加重稿仍要求胸肩拳腳更厚。第二稿已低頭盔、寬厚胸肩、巨拳大腳與短腿，編成四向及144／72／48實際尺寸對照；等待體格確認，尚未登錄，也未以舊GPU替新稿放行。其餘人類依亞雷斯、哈諾與成熟哈瓦特畫風；新審查只更新第2／30組評語，其餘124組來源逐SHA相同。#287仍餘11已見身份／裝備偏移及6待核，#286全套及完整README人物總覽保持未完成。
- README三人局部圖已換本批鐵諾，附完整受控遊戲PNG；蓋亞舊圖說明尚待加重，56／57／58／61與遠端兩工單同步。GPU144.311秒／峰值6804111360 bytes，正常222.021秒／4044849152 bytes，events全零。39份SHA與3118本地連結通過。沒有錄影或Release，無FD2殘留容器；全repo歷史root-owned檔2808項不在本批目標，未批次改權限，本批兩工作根均為目前UID。主收據hd1_tino_hood_and_gaia_bulk_followup_20261029保存所有輸入、命令、來源限制與清理紀錄。

### 2026-10-30 蓋亞定版與第53／54組身份、裝備修正

- 從0ef907e3建立feat/hd1-native-identity-review-r2，使用者選回先前已登錄Gaia HD1版定案。核母稿3370cb36及25檔、兩清冊項保持；兩份加重候選不採用，不再列體格待確認。AGENTS／README／56／57／58／61及遠端#286同步決定。
- 第53／54組各以原生十二姿態、代表色盤與成熟Harvat3／Ares4畫風作參考，內建圖片工具各產一份原稿。Harvat53修回成熟大鬍子、右手斧與左臂T記號盾；Ares54修回人臉、藍軍帽、右手劍與左臂鋼藍盾。左向遠側兵器保留原版遮蔽，原稿不經程式補畫；提示與來源依版本化配方。
- 新variant矩陣固定53／54共14完整Game.Draw、12PNG與兩次F2，170399核心點及十二反例通過。只作身份3／4基底槽的明示selector key／raw byte6／cache slot投影，不作升階、能力值、自然加入或原版動態證據。HD介面取四向中性與正面三拍，原版介面只取正面中性。預設三人矩陣完整21案例／18PNG、239645核心點與十反例另行通過，沒有縮減舊驗收。
- 首輪party JSON的map_selector_key不被玩家建構器讀取，GPU／CPU正確拒收；改為測試狀態投影，正式程式不改。第二輪GPU本體239.89秒已PASS並寫完整收據，後續CPU在同Go程序RunGame結束後NewImage觸既有守衛，原批次262.092秒／returncode1保留。CPU獨立程序11.605秒補驗通過，沿既有程序隔離教訓，不把排程失敗當產品缺陷或覆寫失敗碼，也不重跑已通過且經獨立像素核對的GPU。
- 三項逐byte重編／八來源拒收、21頂層Go含字串、正式21來源案例／416幀／19反例及全125組1455來源通過。普通首章300.291秒／峰值4792803328 bytes，十PNG、原有角色切換／走行取消／F5與新程序LOAD相容通過；source與正式規則不變。新GPU原批次峰值6197035008 bytes、events全零；舊三人GPU216.575秒峰值8GiB、max298而無OOM，不作平台餘裕聲明。
- 正式883項只換四項，879項保持。50新檔已私人白名單保存，原7391美術檔inode／size／mtime／UID保持，私人提交db4ec96489837f00f6243cfe73cb91d767094945已推送並核遠端HEAD及private。公開庫只保存配方、清冊、受控完整PNG及部分原版／HD1對照，未放母稿或完整原版包。
- 本批重看53／54並依使用者決定更新Gaia30評語，其餘123組來源逐SHA保持。#287餘48／50／51／52／55／56／63／76／79九組已見偏移及六組待核，其他Q比例與全套人物總覽仍待；#286／287保持OPEN。47份SHA、3143本地連結、30工單及380教訓／234守衛通過。正式Go未改，1196份Go現行SHA549d031f56d179947b95c08a1f907a7d64eb8dabbb765078909f6a46c43dba4a只因測試改動而更新。無錄影／Release／FD2殘留容器，本批工作根無root-owned；全repo既有2808項保持。主收據hd1_native_variants_followup_20261030保存限制、來源、所有診斷與重跑方法。

### 2026-10-30 磁碟全滿後的工作區清理

- 使用者中止人物編製，要求先清理工作區。初查根檔案系統100%且可用0，sandbox連暫存marker都無法建立；以主機Docker／git控制面恢復盤點。FD2約43GiB，work約37GiB，最大可安全刪除項是Go編譯快取。未提交Go／Python測試修改、原版資料、正式素材、私人庫及dist-all保持。
- 先以唯讀工具保存一個可重建cache object的路徑、56281666 bytes及SHA238cf4d6c045876dc99c5cb961074ff026e3917f5d58d2723931fa7cd84d1b70，刪除後立即落地journal。之後先保存其餘4176檔完整路徑／大小／SHA計畫，再逐檔核對並刪除，逐項保存結果；合計4177檔，配置量估計5629562880 bytes，約5.63GB，保留空cache根目錄供下次建置重生。同期其他檔案系統變動也使全碟空間上升，不把全碟差額算成本輪成果。
- 清理後FD2配置量39555600384 bytes，約39.6GB；硬碟可用已超過300GB。現有FD2兩套專用工具鏈及共用Pillow保留，沒有可確認過期的FD2卷或映像；共享建置快取及其他專案資源保持，無FD2清理容器殘留。
- 哈諾51與貝克威63已生成的個別五官修稿、五份完整提示及原版頭像來源保存於work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/hd1-identity-review-r3/face-generation-pending.json；全滿造成的部分複本已由原始生成檔逐SHA修復，零byte中斷輸出記錄後移除。尚未編製、GPU驗證或登錄，不宣稱這兩組完成。人物臉部須逐人參考原版大頭像，保留臉型、眼型、鼻形、眉形與表情，不能只換髮型沿用同一張臉。
- 本輪清理明細與驗證見[清理收據](work/fd2-space-cleanup-20261030-r1.json)、[逐檔刪除紀錄](work/fd2-space-cleanup-20261030-r1.jsonl)及[刪除前快取清冊](work/fd2-go-cache-removal-20261030-r1.jsonl)。沒有重跑遊戲、建置、錄影或Release；私人庫保持乾淨。
### 2026-10-30 依原版五官完成哈諾51與貝克威63接線

- 使用者要求逐人參考原版大頭像，保留不同臉型、五官及表情。沿已定Q比例，哈諾51依DATO001修回長鼻細眼抿嘴、棕髮頭帶及灰綠甲單斧；貝克威63依DATO013恢復金長髮人臉、平眉半闔眼微笑與弓箭筒。五份逐字生成提示、未採稿原因、兩份原稿與來源SHA記於面容配方，沒有程式補畫。
- 兩人十二姿態依原版裝備遮蔽編製。固定identity-pair profile明示身份1／13的selector、byte6及cache槽投影51／63，14完整Game.Draw、12PNG、兩次F2、保存／RNG保持、172469核心點與十二反例通過。原版介面只取正面中性，HD介面四向中性與正面三拍；不證明升階、自然加入、能力或原版動態。party21與variants14矩陣保持，本批不重跑其歷史GPU。
- 三項逐byte重編／八來源拒收、21頂層Go含字串、正式21消費端案例／416來源／19反例與883項完整清冊通過。兩項專用GPU未啟用的skip不計通過。來源測試的兩個既有BufferedReader警告保留，容器退出已回收。首次來源命令-e位置錯誤已修；獨立核對曾先於normal收據完成啟動而FileNotFoundError，等原命令完成後只重跑核對，沒有重跑已通過的遊戲或改圖源。
- 普通首章十PNG驗完整候選載入、四組控制、原有Ares走行取消及保存／新程序LOAD；來源沒有51／63，其本人像素只由上述受控投影驗證。GPU385.963秒／6412660736 bytes，正常438.523秒／4355256320 bytes，CPU回歸88.221秒／1480503296 bytes，memory.events全零。清理後Go冷快取已重建；正式Go未改，1196份Go現行SHA60c58702147ee034b54ebddc237086d5471d68b8504c2775333280f5a0b91ab1只因測具更新而改變。
- 正式883項只換四項，879項保持。50新檔依清冊白名單保存私人庫，原7441美術檔的內容／inode／size／mtime／UID保持，Gaia兩項及25圖源未改。私人提交118a3527268361a54c173c07d1e546aafa437b14已推送並核遠端HEAD，保存庫仍private。
- README新增原版頭像與實際144／72／48面容對照、四向局部對照及兩人的完整受控戰場圖。其餘123圖組來源SHA保持；已見身份／裝備偏移餘48／50／52／55／56／76／79七組，另六組身份待核。原哈諾1、亞雷斯4／54優先複核五官，其餘人類逐人續審；不以相同Q畫風或舊技術收據推定全部面容通過。#286／287已更新且保持OPEN，全人物重繪與完整總覽尚未完成。
- 主收據hd1_native_faces_followup_20261030保存來源、命令、限制、排程診斷及資源報告。沒有錄影、Release或FD2殘留容器；新增工作產物均user1000，歷史root-owned不動。
- 最後核對33份來源SHA及本批修改文件3362個本地目標，沒有新增斷鏈。全庫5583目標另有九項既有結果：一項inline code誤判、八項舊dist-all產物路徑；四份歷史文件與HEAD相同，不重建已清理的舊包。首次全庫零斷鏈斷言及Python逐檔stat逾時均保留診斷，改用指定find於25.256秒完成root-owned／md目錄自檢，既有2808項未動；收據final-validation-r3.json已掛入主收據。
### 2026-10-30 洛娜55與萊汀56原版身份及個別五官修正

- 從origin/main 11fedb32建立feat/hd1-native-identity-review-r4。沿既有Q比例與逐人原版五官要求，第55組依DATO005恢復洛娜灰綠長髮、杏眼紅唇女性，第56組依DATO006恢復萊汀褐髮深色頭帶、長鼻方下巴。原版十二姿態決定藍鋼甲、窄尖／寬側刃長兵器及左右遮蔽，不沿舊機器人／紅頭盔／劍稿。三份完整內建圖像提示與首稿未採原因保存在騎士形態配方；最終母稿未經程式補畫。
- 新增固定knights profile只驗55／56共14完整Game.Draw，HD介面四向中性／正面三拍，原版介面正面中性；兩次F2、單位／RNG／保存保持，12PNG與136653核心點、十二反例通過。從章9／10 authored身份5／6注入selector、byte6及cache槽，僅驗呈現，不證明升階、自然入隊或能力。party21／variants14／identity-pair14矩陣保持，本批未重跑其歷史GPU。
- 三項逐byte重編／八來源拒收、九組CPU fixture、21頂層Go含字串與正式21消費端案例／416來源／19反例通過；兩項未啟用專用GPU的skip不計通過。來源log保留既有compile_group兩個BufferedReader ResourceWarning，三項通過且容器退出已回收。正式883項清冊與125組1455來源重核通過，其餘123圖組逐SHA保持。
- 普通首章十PNG驗完整候選、四組控制、原有Ares走行取消及保存／新程序LOAD。來源沒有55／56，本人像素僅由上述受控投影驗證。GPU245.964秒／6052110336 bytes，正常364.733秒／5216739328 bytes，CPU回歸44.590秒／1451167744 bytes，memory.events全零。正式Go未改，1196份Go現行SHA59a5a9715bae96163cb3c0589f6f93acf6029a6ee913a8fa9bf0bcf280b3d311只因測具更新而改變。
- 正式清冊只換四項，879項保持。50新檔依清冊白名單保存私人庫，原7491美術檔的內容／inode／size／mtime／UID保持，Gaia兩項及25圖源未改。私人提交fe197121e74736e0426ff92a7fb8dc5e878c9c56已推送且核遠端HEAD，保存庫仍private。
- README新增兩人原版頭像／實際144／72／48面容對照、四向局部對照及完整受控戰場圖。#286／287續追48／50／52／76／79五組已見身份／裝備偏移、六組身份待核與其他Q比例／個別五官；原1／4／54優先面容複核。全人物重繪及完整總覽未完成，工單保持OPEN。
- find在19.444秒完成工作區擁有權與錯置md目錄自檢，既有2808項root-owned未動，本批無新增；沒有錄影、Release或FD2殘留容器。主收據hd1_knights_native_followup_20261030保存來源、提示、命令、驗證範圍與限制。
- 提交前35份來源引用SHA、兩原版頭像SHA及四公開PNG／來源一致性通過，本批3348個本地連結目標沒有新增斷鏈。全庫5588目標維持前輪九項既有結果，未改四份歷史文件；30工單、380教訓／234守衛通過。受限sandbox的bwrap掛載一度失敗且git回135，改用契約允許的host Git重跑相同唯讀檢查通過，未重跑已通過的遊戲。完整稽核final-validation-r1.json已掛入主收據。
### 2026-10-30 索爾50與悠妮52原版人形及明示身份

- 從origin/main 6d9a517c建立feat/hd1-native-identity-review-r5。原先按圖組編號考慮鐵諾52，查既有class_change_targets後確認0的optional_target50與9的special_target52；LABELS亦已標Yuni special promotion item90，身份在編製與登錄前已訂正。這是既有資料消費，不新增轉職規則或重開RE。
- 索爾50依DATO000恢復深藍髮紅頭帶、收斂眼神、較長直鼻與灰綠甲／藍衣／單側紅布；悠妮52依DATO009恢復柔和眉眼與笑容、紅色頭部輪廓／藍金飾帶、紅外袍紫內衣。兩人原生十二姿態皆空手，不新增兵器或機械面具。三份完整內建圖像提示、未採首稿與來源SHA保存於主角形態配方，最終原稿未經程式補畫。
- heroes固定50／52共14完整Game.Draw，HD介面四向中性／正面三拍、原版介面正面中性；12PNG、兩次F2、單位／RNG／保存保持、155665核心點與13反例通過。新反例明示52誤綁2，CPU／獨立核對器另讀class_change_targets與配方綁0／9，不再按減50泛推身份。舊固定profile矩陣及身份保持，本批未重跑其歷史GPU。
- 三項逐byte重編／八來源拒收、11組CPU fixture及明示資料對照、21頂層Go含字串、正式21來源案例／416幀／19反例、883項完整清冊與125組1455來源核對通過，兩個專用GPU skip不計通過。52的f08／f06三層原生輸入、24父圖與144圖源逐SHA等價保持。來源log的既有兩個BufferedReader ResourceWarning保留，測試通過且容器退出回收。
- 普通首章十PNG包含基底索爾0／悠妮9，驗完整候選、四組控制、Ares走行取消及保存／新程序LOAD；不含selector50／52，轉換後本人像素僅由受控投影驗證，不作自然轉職、能力或原版動態聲明。GPU180.008秒／6345076736 bytes，正常261.711秒／5263183872 bytes，CPU回歸44.085秒／1515745280 bytes，memory.events全零。正式Go未改，1196份Go新SHA6f0ac320f888aea34272a87d7f0175242d396f7967c7a91ef4a03502032b7edb只因測具更新。
- 正式883項只換四項、879保持；50新檔按白名單私人保存，原7541美術檔的內容／inode／size／mtime／UID保持。基底0／9及Gaia30共六項75檔未改。私人提交9a82738eb481184f10c98fd5e800086fda1be7a2已推送並核遠端HEAD，保存庫仍private。
- README新增兩人原版頭像／144／72／48面容、四向對照及完整受控戰場圖。其餘123組來源SHA保持，已見身份／裝備差異餘48／76／79三組、另六組身份待核；原1／4／54及其他五官續審，全人物Q重繪與完整總覽未完成，#286／287保持OPEN。新增可重用身份對照教訓與guard，source明示target不按圖號猜人。
- find於16.469秒完成root-owned與錯置md目錄自檢，舊2808項未動，本批無新增。沒有錄影、Release或FD2殘留容器。主收據hd1_heroes_native_followup_20261030保存所有來源、提示、命令、限制與回歸。
- 提交前38份來源引用SHA、兩原版頭像SHA及公開PNG／來源一致性通過，本批3353個本地目標沒有新增斷鏈。全庫5593目標維持前輪九項既有結果，四份歷史文件未改；30工單、381教訓／235守衛通過。身份舊說法搜尋只命中61的明示訂正，完整稽核final-validation-r1.json已掛入主收據。


## 2026-10-30 HD1士兵與蒙面劍士原版身份修正

- 來源：從origin/main 9be26422另開feat/hd1-native-identity-review-r6。原版76／79十二姿態、色盤、DATO076／079及map0／18名冊綁SHA；內建image_gen.imagegen兩份提示與定版原稿依敵方配方保存。
- 實作：76恢復十字方盾與直劍，79移除骷髏並保留金屬遮眼帶／面巾；各組只換兩清冊項與25新檔，其他879項及Gaia保持。正式Go未改，新enemy測具使用原生名冊及完整群建構器。
- 驗證：14完整繪圖／12 PNG、兩次F2、163874核心點、16反例；三項重編／八來源拒收、22頂層Go含八字串、正式21來源案例／416幀／19反例、完整883項及125組1455來源通過。普通首章十PNG驗四組控制、走行取消、保存／冷讀，敵方像素限受控原生群，不宣稱第19章自然到達。
- 資源：GPU202.223秒、峰值6487482368 bytes；正常368.28秒、4085157888 bytes；CPU109.05秒、1532633088 bytes，三份memory.events全零。首兩次敵方測具因空快取未建呈現資料失敗，已依既有完整AppendGroupWithNativePlacement修正，同容器命令乾淨重跑通過；分類為測具問題，非產品缺陷。既有編製器兩次BufferedReader警告原樣記錄。
- 保存：私人庫提交76b0f66bab53a6d691bc77b216e55ca0a4f5a99a已推main並核遠端與private。原7591美術檔的SHA／inode／mtime／擁有權保持，50新檔user1000，現行catalog白名單7121圖源加兩metadata。公開只存配方與四張整理圖，README／58／56／57／61及#286／287同步；主收據hd1_enemies_native_followup_20261030。
- 範圍：已見偏移剩48與六組身份待核；69組Q比例與其他五官、全人物總覽仍待，集中人工由使用者接手。沒有錄影或發行產物；2808既有root-owned項目不碰，本批無新增。Docker一次性容器已結束，交付前再核。完整重跑命令與收據見主收據references及commands-r1.json。
- 收尾：final-validation-r1.json核35引用SHA、兩頭像／兩名冊與四公開PNG來源，5598本地目標及3358變更文件目標，無新增壞連結；九項既有歷史發現與四舊文件保持。382教訓／236 guard及30遠端工單驗證通過，23人工／0可能完成。Docker交付前核無非預期FD2容器。公開提交與推送於本段後完成，提交身分wicanr2；沒有新tag或Release。


## 2026-10-30 HD1父王原生三拍人形修正

- 從origin/main 5d22a7c2另開feat/hd1-static-human-review-r7。第48組由map32名冊、王座廳actor、speaker48父王與DATO048核身份；大頭像證明藍色輪廓為頭髮，訂正初審藍帽導覽標籤。此場景名稱不泛化成轉職target48的持續身份。
- 內建image_gen.imagegen一次產生三拍原稿，完整提示與SHA依三拍配方保存；藍髮金額飾、成熟濃眉長鼻、藍鬍鬚長下巴及紅披風空手保持。只換兩清冊項、三張24父圖／三張144圖源與母稿，共七新檔。既有編製器本來支援三拍，這批只補對照工具與固定故事驗證矩陣。
- 六完整故事繪圖／三PNG、一次F2、20554核心點及16反例，三項重編／八來源拒收、23頂層Go含八字串、正式21來源案例／416幀／19反例、883項清冊及125組1455來源通過。正常首章十PNG驗四組控制、走行取消與保存／新程序冷讀。GPU預存測試首章保存保持，不新增故事保存功能。
- 第48組不具完整raw race／class構造，沿既有native_story_map.go故事載體與私有前景近似；不注入selector／持續身份或升格戰鬥，原本隱藏HUD。正式Go程式、規則、保存schema及其餘124組不改。本批全程沒有失敗重跑，編製器既有一個BufferedReader警告原樣保存。
- 資源：GPU193.174秒／4244652032 bytes，正常433.278秒／5126135808 bytes，CPU72.251秒／1469452288 bytes；三份memory.events全零。私人庫提交ab7993d761a831e31ca853178092e1054f3da05b已推main並核遠端與private，原7641美術檔SHA／inode／mtime／擁有權保持；七新檔user1000，現行圖源白名單7120加兩metadata。
- README／58／56／57／61與#286／287同步；主收據hd1_king_static_followup_20261030。初審十二組已見偏移均處理，六組身份、69組Q比例及其他五官／完整總覽仍待，集中人工由使用者接手。沒有錄影或發行產物；2808既有root-owned不碰，本批無新增。一次性Docker工作結束，交付前再核；完整重跑命令與來源見主收據references。
- 收尾核對：final-validation-r1.json驗34引用SHA、一原版頭像、三故事來源、一原版對照板與三公開PNG逐byte相同；5602本地目標、3362變更文件目標，無新增壞連結。九項既有歷史發現與四舊文件保持；383教訓／237 guard、30遠端工單驗證通過，23人工／0可能完成。公開提交與推送於本段後完成，提交身分wicanr2，沒有新tag或Release。


## 2026-10-30 HD1六組視覺身份與裝備複核

- 從origin/main 63c2ea64另開feat/hd1-six-identity-review-r8。六組70／73／77／89／125／133由map5／24／2／17／24／1名冊及各自DATO綁身份、位置、陣營與SHA；73龍頭雙足、77無圓盾、89露頸面甲、125無白鬚、133平瀏海，70露下臉且斜白線為劍。70藍色頭部材質不提升為已證實。
- 內建image_gen.imagegen六組生成加兩次單點修訂，八份逐字提示與兩未採原因依六組配方；73正面偏側向已修正，77盔頭比例／右向劍位已修正。六母稿未經程式補畫。只換12清冊項與150新檔，其餘871項及Gaia保持。
- 三個獨立GPU程序各14繪圖，共42完整Game.Draw／36PNG、六次F2、481439核心點與21反例；合併工具核固定配對、完整來源／記憶體報告SHA、PNG實檔。缺份／重複份／錯來源均拒收。CPU測具另程序，三項重編／八來源拒收、24頂層Go含八字串、正式21來源案例／416幀／19反例、883項與125組1455來源通過。其餘119組圖源SHA保持，原7648美術檔SHA／inode／mtime／擁有權保持。
- 正常首章十PNG驗完整候選、四組控制、走行取消與保存／新程序冷讀；六角色本人限原生名冊群控制，保留原陣營／位置／portrait／selector，不作自然章節、AI或原版玩法語義證據。正式Go程式、規則及保存schema未改；20組CPU來源測具通過，七個舊GPU profile保留原範圍，不稱本批全套重驗。
- 資源：banded120.728秒／4933586944 bytes，draconic116.207秒／5625122816 bytes，robed107.203秒／5796909056 bytes；正常222.964秒／5232463872 bytes，CPU43.586秒／1353052160 bytes，五份memory.events全零。未跑單程序42，不宣稱它必然OOM。全批無測試失敗重跑，來源編製既有六個BufferedReader警告保留，程序結束清理。
- 私人庫提交5055411c931fe0af1efe5a274ace3f351c3cd265已推main並核遠端及private；150新檔user1000，現行白名單7118圖源加兩metadata。公開只存配方、來源雜湊與12整理圖；README／58／56／57／61與#286／287同步，主收據hd1_six_identities_followup_20261030。導覽標籤另將55／56／63舊unknown同步既有已閉合資料。
- 初審十二組偏移與六組視覺待核均處理，仍69組Q比例及其他五官、完整人物總覽未完成，集中人工由使用者接手。384教訓新增分段GPU完整來源規則；沒有錄影或發行產物。2808既有root-owned不碰，本批無新增；Docker一次性容器交付前再核，完整命令與來源見主收據references。
- 收尾：final-validation-r1.json核64引用SHA、六頭像／六名冊綁定／六原版對照板、12公開PNG與36合併GPU圖；5615本地目標、3375變更文件目標，無新增壞連結。九項既有歷史發現及四舊文件保持，384教訓／238 guard與30遠端工單驗證通過，23人工／0可能完成。公開提交與推送於本段後完成，身分wicanr2，無新tag或Release。


### 2026-10-30 哈諾與亞雷斯基底面容

- 目前狀態以58首列與主收據hd1_base_faces_followup_20261030為準。本批只修1／4的臉型、五官與原生持物側；54面容與其他69組Q比例仍待，Gaia30保持已定v1。
- 先核ch02身份1／fig1／portrait1及ch01身份4／fig4／portrait4，再綁DATO001／004與原生十二姿態三層、代表色盤。READY後用內建繪圖各出一稿及一次面容修訂；初稿偏通用圓臉，兩份不採。定稿保留長瘦／寬方臉，原稿、逐字提示、SHA與版本化編製依hd1-base-faces-refresh.recipe.json。原版頭像不拿來猜不可見裝備。
- 固定base-faces不用selector／身份投影。14完整Game.Draw、12PNG、兩次F2、169923核心點與15反例通過；普通首章十PNG、三張亞雷斯地圖／資訊窗47002核心點、四組控制、走行取消、保存及新程序LOAD通過。哈諾本人限受控party，未跑FD2.EXE，不新增自然加入或原版動態聲明。
- 三項逐byte重編、八來源拒收、24頂層Go含字串與21來源案例／416幀／19反例通過，兩項可選GPU略過不計通過。全883清冊及125組1455來源核對，其他123組逐SHA相同。正式只換四項、879保持；50新圖源私人保存，原7798美術檔的SHA／inode／mtime／owner及0／9／30六項保持。private allow-list7118，保留舊版本不等於目前白名單。
- GPU158.400秒／峰6556475392 bytes，普通238.507秒／3841421312 bytes，正式回歸62.162秒／1299337216 bytes，memory.events全零，沒有OOM／kill。編製初次相對out被relative_to拒收，未發布候選；同一容器改絕對/repo路徑乾淨重跑通過，屬命令問題。編製器既有兩次BufferedReader ResourceWarning留log，程序結束清理，不改列為產品錯誤。
- 私人庫提交3e0d4f36ada6db6b58c53a3b7e49301eeec71a2c已push，52路徑精確為50新圖加兩份metadata。README新增兩人原版面容／實際尺寸與受控圖，首頁亞雷斯改接本批普通遊玩圖；早期三人比例圖明標舊來源。沒有建立新Release或錄影。一次性容器完成即退出，最後清理與引用檢查另記本節。
- 收尾：24份來源雜湊、五公開PNG與12受控／10普通PNG核對；5631個Markdown目標中，變更文件3381個目標沒有新增缺檔。其餘舊檔掃描結果包含行內程式表示式與歷史封包路徑，未改動。新配方已掛61入口；384教訓／238guard與30條worklist驗證通過，23要人判／零可能完成。2808個既有root-owned不動，沒有本批新root-owned或.md目錄；私人遠端HEAD及PRIVATE已核實。


### 2026-10-30 三名基底與亞雷斯轉職面容

- 使用者要求沿既有定案持續推進，不再詢問相同比例。狀態以58首列與主收據hd1_knights_bow_followup_20261030為準。洛娜5、萊汀6、貝克威13由redraw-q-proportion改為new-q-review-keep；69→66。54完成個別面容，其他人類五官仍逐人複核，不能把優先項清空稱全人物通過。
- 先核ch09／10／07名冊5／6／13及class_change_targets的portrait4／optional_target54，再綁各自DATO與原生十二姿態三層、代表色盤。READY後用內建繪圖各出一份選定原稿，共四份；逐字提示、參考原稿SHA及來源依hd1-knights-bow-refresh.recipe.json。6修回原生鉤刃長兵器、短褐髮及灰綠護具，不沿舊馬尾／垂劍；5保留橘褐披風無盾，13保留棕弓箭筒，54沿基底4的成年人臉並保持劍盾。
- 同一完整四組候選以base-knights5／6與bow-and-ares13／54兩程序驗GPU。28完整Game.Draw、24PNG、四次F2、326002核心點與21反例通過。逐actor保留scenario SHA及身份／portrait／selector，只有54投影key／byte6／派生cache，保留4身份及能力；不作自然加入／轉職證據。舊六組三份收據讀取契約通過，舊GPU不冒用目前Go指紋。
- 普通首章十PNG驗完整候選、四組控制、走行取消、F5及新程序標題LOAD；沒有狀態注入，不作四人自然出場證據。來源三項逐byte重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例與883清冊通過，兩可選GPU略過不計通過。編製器原有四次BufferedReader ResourceWarning留log，程序結束清理，沒有改正式編製器。
- 正式只換八項，875保持；100新檔私人保存。原7848美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組逐SHA保持，全125組1455來源重核。白名單7116是現行圖源：5／6將原本分開的24父圖與HD母稿統一為各一份，所以比前版7118少兩份，舊檔仍完整保留。
- GPU424.040／424.760秒，峰5540544512／6200733696 bytes；普通371.393秒／5553324032 bytes，回歸119.133秒／1600634880 bytes，memory.events全零。所有程序returncode0且無timeout／OOM／kill，不以本輪共享主機牆鐘時長推論產品效能。
- 私人庫提交7de80609459404d22b0645444541b72c4aa6c277已push，102路徑精確為100新圖及兩份metadata。README補兩組原版／HD1面容、四向與144／72／48實際尺寸及四張完整受控戰場；原版文化保存段落保持。未跑FD2.EXE、未新增Release或錄影。最終來源雜湊、引用及容器清理另記本節。
- 收尾：26份來源雜湊、八公開PNG、24受控及十普通PNG核對；5640個Markdown目標中，變更文件3390個目標沒有新增缺檔。既有程式表示式與歷史封包路徑留原紀錄。新配方已掛61入口；384教訓／238guard、30條worklist／23要人判／零可能完成通過。原2808個root-owned不動，沒有本批新root-owned或.md目錄；私人遠端HEAD及PRIVATE已核實，容器收尾檢查未留下非預期FD2工作。


### 2026-10-30 蘭斯洛特、希莉亞與兩名僧侶基底

- 沿使用者已定比例與逐人原版頭像方式自主推進7／8／10／11，沒有重問相同比例。來源為ch19／03／06／11的明示identity／fig／portrait、各自DATO、十二原生三層與代表色盤；先寫READY再編製與接線。當前唯一狀態依58首列及主收據hd1_lance_bow_clerics_followup_20261030。
- 內建繪圖五次，採四份原稿。7保留長髮嚴肅成人臉，8柔和紅唇微笑，10粉紫高馬尾及紫唇，11灰金後馬尾及成熟紅唇。7與10保持原生空手，8棕弓箭筒沿原版；11初稿左向仍露出遠側持杆，單點修正後採用。配方保留逐字提示、實際參考圖呼叫順序／SHA、選稿及未採理由，正式母稿原樣私人保存。
- 同一完整候選分longhair-and-bow7／8與clerics10／11兩個GPU程序。28完整Game.Draw、24PNG、四次F2、269843核心點與21反例通過；scenario SHA、原生身份與false投影旗標逐actor核對。無selector／身份投影，不作自然加入或原版動態證據。舊audited-six／party-next固定分份讀取契約另驗，舊GPU不冒用目前Go指紋。
- 普通首章十PNG驗整包、四組切換、走行取消、F5及新程序標題LOAD，沒有狀態注入。三項來源重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例與883清冊通過，兩可選GPU略過不計通過；編製器既有四次BufferedReader ResourceWarning保留，程序結束清理。沒有修改正式Go／玩法、重新跑FD2.EXE或平台發布。
- 只換八項、875保持，100新檔白名單私人保存。原7948美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組逐SHA相同，全125組1455來源重核。四組由待重繪改為已重繪，66→62；其他個別面容及完整總覽仍待，Gaia維持使用者定版。完整125組目前決定與面容狀態保存在主收據，不只依暫存圖表。
- GPU320.233／317.231秒，峰6218760192／6041300992 bytes；普通436.952秒／5923516416 bytes，回歸91.318秒／1495019520 bytes，memory.events全零且無timeout／OOM／kill。牆鐘時長不作產品效能聲明。
- 私人庫提交0f35e6ef447cebedb3eeffdfe82470b0cebd4c9f已push，102路徑精確為100新圖及兩份metadata，現行白名單7113。README新增兩組原版／HD1面容、四向及144／72／48尺寸與四張完整受控戰場；文化保存內容保持。沒有錄影或新Release，最終引用與容器清理另記本節。
- 收尾：26份來源雜湊、八公開PNG、24受控及十普通PNG核對；5649個Markdown目標中，變更文件3399個目標沒有新增缺檔。既有程式表示式與歷史封包路徑留原紀錄。新配方已掛61入口；384教訓／238guard、30條worklist／23要人判／零可能完成通過。原2808個root-owned不動，沒有本批新root-owned或.md目錄；私人遠端HEAD及PRIVATE已核實，容器收尾檢查未留下非預期FD2工作。


### 2026-10-30 凱麗、珊與獸形同伴基底

- 依已定比例及逐人原版面容續作12／14／15／16，先核ch08／12／16／18明示identity／fig／portrait、DATO及十二原生三層與代表色盤，再寫READY。沒有重新推測EXE或用名字補武器。唯一現況依58首列與主收據hd1_martial_beasts_followup_20261030。
- 內建繪圖四次採四稿。凱麗保留金側瀏海遮眼、成熟斜眉與自信紅唇；珊保留藍低馬尾、綠頭帶與暖褐杏眼；賽可邦勒保留深藍獸形長吻、黑鼻及直立耳；凱拉斯保留褐龍長吻、黑角及原生紅劍藍紋盾。前三人空手，未把頭像額帶加進地圖；四向持物側及遮蔽按原生。配方保存逐字提示、實際參考順序與SHA，四份母稿原樣私人保存。
- 同一完整候選分martial-and-mage12／14與beast-and-dragon15／16兩個GPU程序，共28完整Game.Draw、24PNG、四次F2、243544核心點及21反例通過。逐actor來源SHA與false投影旗標核對，沒有selector或身份投影。三種舊分份讀取契約通過，不重跑舊GPU或升格其Go來源。
- 普通首章十PNG驗完整候選、四組獨立控制、走行取消、F5及新程序標題LOAD，沒有狀態注入。三項來源重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊與125組1455來源通過，兩可選GPU略過不計；既有四次BufferedReader ResourceWarning保留，程序結束清理。正式Go、玩法與保存沒有改動。
- 正式只換八項、875保持；100新檔按白名單私人保存，原8048美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組SHA相同。Q比例餘項62→58；其他五官與完整總覽續作，Gaia維持定版。125組目前決定與面容狀態保存主收據。
- GPU243.564／241.396秒，峰5902868480／5666541568 bytes；普通419.895秒／5187969024 bytes，回歸60.64秒／1520222208 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作產品效能聲明。
- 私人提交0325da70c49579d9628cf295a9a025dce456a5e3已push，102路徑精確為100新圖與兩份清冊，白名單7111。README補兩組面容／四向／實際尺寸及四張完整受控戰場，文化保存保持。本輪未跑原版動態、自然加入或平台發布，沒有錄影；最終引用與清理另記。
- 收尾：26份來源雜湊、八公開PNG、24受控與十普通PNG核對；5658個Markdown目標中，變更文件3408個目標無新增缺檔，歷史程式表示式及舊封包路徑保持原紀錄。新配方已掛61入口；384教訓／238guard及30條worklist／23要人判／零可能完成通過。原2808個root-owned不動，沒有新增root-owned或.md目錄。私人遠端HEAD與PRIVATE已核實，未留非預期FD2容器。受限主機git diff一次以訊號135結束，原因未定；相同命令以host權限乾淨通過，不列為產品測試失敗。


### 2026-10-30 紅龍、蜜蒂、莎拉與約拿基底

- 沿已定比例與各自原版頭像續作17／18／20／21。先核ch13／17／23／19明示identity／fig／portrait、DATO、十二原生三層及FDOTHER0代表色盤，再寫READY。19未在party名冊找到明示身份，本批不為湊數推定，仍留待辦。唯一現況依58首列及主收據hd1_dragon_elder_followup_20261030。
- 內建繪圖五次採四稿。17保留紅褐窄龍面、紅眼、長角、藍劍灰盾；18保留金長髮、成熟高眉窄眼、紅唇、紅披風與空手；20保留綠髮尖耳、紅額帶、收斂紅唇及背側藍灰窄片；21保留白藍髮、大白鬚、高齡皺紋、白眉窄眼、長鉤鼻及棕袍。21初稿正面偏側，單點修第一列後採用，原稿與修正版SHA及逐字提示保留。
- 20背側配件與21低位棕色斜杆只按可見形狀；用途未定，不把舊HD曲杖、職業或名字當證據。舊初審的曲杖稱法留歷史，現行逐組notes已改為用途未定，不回填成已證實裝備語義。原版動態、480對話頭像與自然加入沒有新增驗收。
- 同一完整候選分dragon-and-swordswoman17／18及greenhair-and-elder20／21兩GPU，共28完整Game.Draw、24PNG、四次F2、266338核心點與21反例通過。身份、scenario SHA及false投影逐actor核對，沒有selector／身份投影；四種舊分份讀取契約另驗，不重跑或升格舊GPU。
- 普通首章十PNG驗完整候選、四組獨立控制、走行取消、F5及新程序標題LOAD，沒有狀態注入。三項來源重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊與125組1455來源通過；兩可選GPU略過不計。既有四次BufferedReader ResourceWarning保留，程序結束清理。正式Go、玩法與保存不變。
- 正式只換八項、875保持，100新檔白名單保存；原8148美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組SHA相同。Q比例餘項58→54，其他五官及完整總覽續作，Gaia保持定版。125組目前決定及面容狀態保存主收據。
- GPU225.652／222.727秒，峰5128077312／6387851264 bytes；普通319.594秒／5751398400 bytes，回歸132.171秒／1533669376 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作產品效能聲明。
- 私人提交5ecf5873c21a3aab16fdb64cc7a1d5c438c98348已push，102路徑精確為100新圖及兩份清冊，白名單7109。README補兩組面容、四向與實際尺寸及四張完整受控戰場，文化保存保持。沒有錄影或新Release，最終引用及容器清理另記。
- 收尾：26份來源雜湊、八公開PNG、24受控及十普通PNG核對；5667個Markdown目標中，變更文件3417個目標無新增缺檔，歷史表示式及舊封包路徑保持原紀錄。配方已掛61入口；384教訓／238guard、30條worklist／23要人判／零可能完成通過。原2808個root-owned不動，沒有新增root-owned或.md目錄；私人遠端HEAD及PRIVATE已核實，未留下非預期FD2容器。


### 2026-10-30 卡里斯、羅蘭、希爾法與謝多基底

- 依已定比例與逐人原版頭像續作22／23／24／25，先核ch24／22／22／21明示identity／fig／portrait、DATO、十二原生三層及FDOTHER0代表色盤，再寫READY。唯一現況依58首列及主收據hd1_stubble_bow_robes_followup_20261030。
- 內建繪圖四次採四稿。22保留深藍綁髮、紅髮束、成熟窄眼長鼻及鬍渣，原生空手；23保留長褐波髮、額帶、暖褐眼及開口紅唇微笑，藍紫弓與藍灰箭筒；24保留高額禿頂、紅褐側髮、尖耳、綠窄眼、細鬚及紅袍金黃大領，黃色背布不作頭髮；25保留深藍獸形、灰綠窄眼、微揚嘴角與小獠牙，藍紫腰布及腰際橫持短劍。持物側與遮蔽按原生，不加面罩、兜帽、背刀或職業推定武器。逐字提示、實際參考順序與SHA保存在配方。
- 同一完整候選分stubble-and-bow22／23與bald-and-beast24／25兩GPU，共28完整Game.Draw、24PNG、四次F2、293677核心點及21反例通過。原生身份、scenario SHA與false投影逐actor核對，無selector或身份投影；五種舊分份讀取契約另驗，不重跑或升格舊GPU。
- 普通首章十PNG驗完整候選、四組控制、走行取消、F5與新程序標題LOAD，無狀態注入。三項來源重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊與125組1455來源通過；兩可選GPU略過不計。編製器既有四次BufferedReader ResourceWarning保留，程序結束清理。正式Go、規則及保存沒有變更，不新增原版動態、自然加入或480頭像驗收。
- 只換八項、875保持，100新檔白名單私人保存。原8248美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組SHA相同。Q比例餘項54→50，其他個別面容及完整總覽續作，Gaia保持定版。125組目前決定及面容狀態保存主收據。
- GPU204.104／206.111秒，峰5793329152／6092795904 bytes；普通277.319秒／5663723520 bytes，回歸67.669秒／1386237952 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作產品效能聲明。
- 私人提交ae85634ce42e1995dbb3d60468a6d8c38685a1b3已push，102路徑精確為100新圖及兩份清冊，白名單7108。README補兩組面容、四向及實際尺寸與四張完整受控戰場，文化保存保持。沒有錄影或新Release；最終引用與容器清理另記。
- 2026-10-31收尾核對26份引用、八張公開圖、24張GPU及十張普通圖；5676個文件目標、其中修改文件3426個，新增失效連結零。19筆既有文字誤判／歷史交付路徑不改；既有root-owned2808項保持，本輪新增零。遠端工單30項／人工23／可能完成零，教訓384／238檢查通過。私人遠端HEAD及PRIVATE已核，FD2測試／GUI容器零殘留，未產生錄影。


### 2026-10-31 兩名龍人、達克賽與亞奇梅吉基底

- 依已定比例及逐人原版面容續作26／27／28／29，先核ch26／20／21／26的native_identity／fig／portrait、各自DATO、FDICON十二姿態三層及FDOTHER0代表色盤，再寫READY。唯一現況依58首列及主收據hd1_dragons_blue_robes_followup_20261031。
- 內建繪圖四次採四稿。26藍灰長吻、藍紫窄眼、冠角及小獠牙，金黃直劍與窄藍灰盾；27橄欖綠寬吻、黃綠細瞳及重頰板，銀劍白盾；28深藍裸面、尖耳、厚重額面、鉤鼻及微揚薄唇，空手與灰綠腰帶；29藍灰高額禿頂、尖耳、深陷窄眼、長鼻及下垂薄唇，空手、藍紫袍及灰綠大領。持側與遮蔽依原生，逐字提示、參考順序與SHA在配方。28眼色依自身DATO藍紫，原生地圖紅眼另留原版參考，不宣稱眼部逐像素相同。
- 同一完整候選分blue-and-green-dragons26／27與blue-and-robed28／29兩GPU，共28完整Game.Draw、24PNG、四次F2、207126核心點及21反例通過。逐actor身份、scenario SHA與false投影核對，無身份／selector投影；六種舊分份讀取契約通過，不重跑或升格舊GPU。
- 普通首章十PNG驗完整候選、四組控制、走行取消、F5及新程序標題LOAD，無狀態注入。三項來源重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊及125組1455來源通過；兩可選GPU略過不計。正式Go、玩法與保存不改，不新增EXE動態、四人自然加入或480頭像驗收。
- 原8348美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組SHA相同；100新檔私人保存。125組決定及面容狀態寫入主收據，Q比例餘項50→46，其餘個別面容及完整總覽續作。
- GPU281.994／281.523秒，峰5718200320／6230306816 bytes；普通583.516秒／5314236416 bytes，回歸187.303秒／1573339136 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作效能聲明。
- 私人提交973735ecfd2e906cc69b7b6f15cddcf58d0333b4已push，102路徑精確為100新圖與兩清冊，白名單7106。README補兩組面容、四向實際尺寸及四張完整受控戰場，文化保存保持；沒有錄影或新Release。
- 對照圖第一次在候選編製尚未確認退出時啟動；確認完成後首圖已通過，批次重跑因既有圖被fresh-output拒收。保留首圖及收據，只補餘三張；未改工具或放寬驗證。舊分份讀取腳本的欄位拼名修為既有source_parts後，同一來源乾淨重跑通過，均屬驗證調度／腳本問題。編製器既有四次BufferedReader ResourceWarning保持，程序退出清理。
- 最終核對26份引用、八張公開圖、24張GPU及十張普通圖，5685個文件目標、其中修改文件3435個，新增失效連結零。19筆既有文字誤判／歷史交付路徑不改；既有root-owned2808項保持，本輪新增零。工單30／人工23／可能完成零，教訓384／238檢查通過；私人遠端HEAD與PRIVATE已核，FD2測試／GUI容器零殘留。


### 2026-10-31 希莉亞、悠妮、瑪琳與索菲亞轉職圖組

- 依已定比例及逐人面容續作58／59／60／61，先核optional_target明示8／9／10／11、ch03／01／06／11原生名冊、各自DATO及轉職FDICON十二姿態三層／FDOTHER0色盤，再寫READY。現況依58首列及主收據hd1_bow_mages_promoted_followup_20261031。
- 內建繪圖五次採四稿。希莉亞長褐波髮與柔和紅唇、金弓灰綠箭筒；悠妮紅長馬尾、藍紋額帶與收斂神情，紫粉頭金杆及紫胸布；瑪琳粉紫馬尾、成熟窄眼與紫唇，藍衣灰綠長直側片，用途不另推定；索菲亞灰金馬尾與紅唇，紅褐衣與金褐側飾，不從基底11補杆。悠妮初稿背向露金杆，按原版三拍遮蔽修正後採用；原版黃色域計數0／0／0只作視覺旁證，不提升武器機制。逐字提示、實際參考順序及SHA保存在配方。
- 同一完整候選分bow-and-unni58／59與two-clerics60／61，共28完整Game.Draw、24PNG、四次F2、304038核心點及25反例通過。逐actor核原生身份8／9／10／11、scenario SHA與true投影；只在測具改selector／raw byte6／cache，不改身份、BattleFig、能力或正式規則。四個新增反例直接驗配方來源、optional欄位、身份與揭露旗標；七種既有分份契約保持，不重跑或升格舊GPU。
- 普通首章十PNG驗完整候選、四組控制、走行取消、F5與新程序標題LOAD，無狀態注入。三項來源逐byte重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊及125組1455來源通過；兩可選GPU略過不計。原版EXE動態、自然轉職及480對話頭像沒有新增驗收，正式Go／保存格式保持。
- 原8448美術檔SHA／inode／mtime／owner及0／9／30六項保持，其餘121組SHA相同；100新檔私人保存。125組決定及面容狀態保存主收據，Q比例餘項46→42，其他面容及完整總覽續作，Gaia保持定版。
- GPU156.399／153.844秒，峰5687599104／5841080320 bytes；普通276.776秒／5544259584 bytes，回歸112.339秒／1501642752 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作效能聲明。
- 私人提交0765ceca536906f066aba688de645d65c49fa170已push，102路徑精確為100新圖及兩清冊，白名單7106。README補兩組面容、四向實際尺寸與四張完整受控戰場，文化保存保持。來源回歸既有四次BufferedReader ResourceWarning保持，程序結束清理；沒有錄影或新Release。
- 最終核對27份引用、八張公開圖、24張GPU與十張普通圖，5694個文件目標、其中修改文件3444個，新增失效連結零。19筆既有文字誤判／歷史交付路徑不改；既有root-owned2808項保持，本批新增零。工單30／人工23／可能完成零，教訓384／238檢查通過；私人遠端HEAD及PRIVATE已核，FD2測試／GUI容器零殘留。


### 2026-10-31 索爾、哈諾、鐵諾與凱麗轉職圖組

- 開工查復古遊戲路由，載入spec-gated-workflow與project-document-responsibilities。依已定大頭短身、逐人原版面容及持續推進授權續作32／33／34／62；先核轉職表、ch01／02／04／08原生名冊、自身DATO與十二姿態三層／FDOTHER0色盤，再寫READY。現況依58首列及主收據hd1_first_promotions_followup_20261031。
- 內建繪圖四次採四稿。索爾深藍髮、紅額帶、短馬尾、濃眉窄眼，空手藍衣灰綠護具；哈諾紅褐髮、藍灰額帶、長窄臉與低眉細眼，冰藍短斧；鐵諾灰綠布兜帽及背布、近鼻藍色遮眼區、長鼻角形下顎，單一直劍；凱麗遮一眼的金色瀏海、褐色杏眼與成熟紅唇，空手。遮眼區只記視覺形狀，不推定機制。逐字提示、參考順序及SHA保存在版本化配方。
- 第一輪圖形擷取後，獨立稽核拒收鐵諾來源SHA：實際選ch04，測具觀測欄位遺漏identity2而記成ch01。補回case2→ch04後，以同一候選及同一命令另存r2重跑兩組；舊收據不改寫，一般流程與圖片編製來源未變。
- 同一完整候選分sol-and-hano32／33與tino-and-kylie34／62，共28完整Game.Draw、24PNG、四次F2、247051核心點及25反例通過。前三組核default_target，62核optional_target，逐actor揭露selector／raw byte6／cache投影；原身份0／1／2／12、BattleFig及能力保持。八種舊分份及party-nine配方契約保持，不重跑或升格舊GPU。
- 普通首章十PNG驗整包候選、四組控制、走行取消、F5與新程序標題LOAD，無狀態注入。三項來源逐byte重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊及125組1455來源通過。兩可選GPU略過不計。沒有新增原版EXE動態、自然轉職或480對話頭像驗收；正式Go與保存格式保持。
- 原8548美術檔SHA／inode／mtime／owner與0／9／30六項保持，其餘121組SHA相同。100新檔私人白名單保存，Q比例餘項42→38，其他面容及完整總覽續作，Gaia保持定版。
- GPU 269.347／271.286秒，峰6149541888／6094913536 bytes；普通277.303秒／5258829824 bytes，回歸96.981秒／1510744064 bytes。memory.events全零，均returncode0且無timeout／OOM／kill；共享主機牆鐘不作效能聲明。
- 私人提交a53b9d9007dd8e91eff57fffbfbba28a6d250dcf已push，102路徑精確為100新圖及兩清冊，白名單7106。README補兩組面容、四向實際尺寸與四張完整受控戰場，文化保存保持。來源回歸既有四次BufferedReader ResourceWarning保持，程序結束清理。沒有錄影或新Release。
- 最終核對27份引用、八張公開圖、24張GPU與十張普通圖，5703個文件目標、修改文件3453個，新增失效連結零。19筆既有文字誤判／歷史交付路徑保持；舊root-owned2808項、本批新增零。工單30／人工23／可能完成零，教訓385／239通過。來源修正前後24張PNG逐byte相同。私人遠端HEAD及PRIVATE已核，FD2測試／GUI容器零殘留。


### 2026-10-31 亞雷斯、洛娜、萊汀與希莉亞預設轉職圖組

- 開工查復古遊戲路由，載入spec-gated-workflow、project-document-responsibilities及內建繪圖技能；依已定比例、個別面容與持續推進授權處理36／37／38／40。先核default_target明示4／5／6／8、ch01／09／10／03名冊、自身DATO及原生十二姿態三層／FDOTHER0色盤，再寫READY。現況依58首列及主收據hd1_default_knights_bow_followup_20261031。
- 內建繪圖六次採四稿。亞雷斯藍帽綠帶與方正臉；洛娜灰綠長髮、成熟杏眼紅唇及藍背布；萊汀方下顎長下巴、微揚薄唇及深藍甲金黃件；希莉亞長褐髮、藍飾、柔和微笑、灰綠領與金弓箭筒。前三組兵器按原生形狀與持側，不推定玩法種類。希莉亞初稿步行幀靴色跳變，先修正十二姿態深褐靴色，再依原版修正背向上半部金弓由長髮遮蔽後採用。逐字提示、實際參考順序、拒收原因及SHA存配方。
- 候選第一輪因背向遮蔽複核主動停止兩GPU，均returncode143，不作通過、OOM或自發signal-killed證據；原因與初稿雜湊另存cancelled-r1.json。補齊六份繪圖的來源／參考SHA後，最終候選獨立編製到candidate-final，驗收只採第二輪GPU。來源測試新增可選FD2_CHIBI_PARTY_CANDIDATE，未設時保持work/candidate，不覆寫前輪資料。
- 同一完整候選分ares-and-lona36／37及lyting-and-cylia38／40，共28完整Game.Draw、24PNG、四次F2、324328核心點及25反例通過。逐actor核原身份4／5／6／8、實際章名冊SHA、default_target及true投影，只改測具selector／raw byte6／cache，保留BattleFig、能力與保存。九種既有分份與party-nine／party-ten配方契約保持，不重跑或升格舊GPU。
- 普通首章十PNG驗整包四組控制、走行取消、F5及新程序標題LOAD，無狀態注入。三項逐byte重編／八拒收、24頂層Go含字串、21來源案例／416幀／19反例、883清冊及125組1455來源通過；兩可選GPU略過不計。沒有新增原版EXE動態、自然轉職或480對話頭像驗收，正式Go與保存格式保持。
- 100新檔及八項清冊已登錄，875項、原8648美術檔SHA／inode／mtime／owner、0／9／30六項與其餘121組來源保持。Q比例餘項38→34，其他面容及完整總覽仍待，Gaia保持定版。
- GPU 161.479／161.576秒，峰5952286720／6152331264 bytes；普通270.792秒／5722402816 bytes，回歸52.092秒／1358680064 bytes。memory.events全零，均returncode0、無timeout／OOM／kill。共享主機牆鐘不作效能聲明。
- 私人提交d3af38e186f0c2ea4f7f7a6a8bf227066aadddb4已push，102路徑精確為100新圖及兩清冊，白名單7102。README補兩組面容、四向實際尺寸與四張完整受控戰場，文化保存保持。編製器既有四次BufferedReader ResourceWarning保持，程序結束清理；沒有錄影或新Release。
- 最終核對29份引用、八張公開圖、24張GPU與十張普通圖，5712個文件目標、修改文件3462個，新增失效連結零。19筆既有文字誤判／歷史交付路徑保持；既有root-owned2808項、本批新增零。工單30／人工23／可能完成零，教訓386／240通過，新增修正版候選位置一致性的教訓。私人遠端HEAD及PRIVATE已核，FD2測試／GUI容器零殘留。


### 2026-10-31 悠妮、瑪琳、索菲亞與凱麗預設轉職共用圖組

- bad request後由工作樹、遠端工單及第19批實檔恢復。載入復古遊戲路由、reverse-engineer-retro-game-remake技能、規格閘門、驗證及文件職責；沿既有READY與持續推進授權完成41／42／43／44，沒有重畫已確認的同角色圖源。現況依58首列及主收據hd1_shared_default_mages_followup_20261031。
- default_target明示原身份9／10／11／12，名冊ch01／06／11／08及各自DATO保持。四對原生十二姿態三層與59／60／61／62逐檔相同，共144對；共用100個既有母稿與24／144圖檔。編製器核原生圖層、既有兩項資產摘要、母稿、幀SHA及HD重編結果，保留目標身份與原版來源索引。沒有新增繪圖或修改正式Go、規則與保存格式。
- 中斷前兩GPU已通過；接手後核對同一候選與來源指紋，完成獨立稽核。共28完整Game.Draw、24PNG、四次F2、301298核心點與28反例。逐actor揭露selector／raw byte6／cache投影，原身份、BattleFig與能力保持。十種舊分份及三批轉職配方契約保持，不重跑或升格舊GPU。
- 普通首章十PNG驗完整候選、四組控制、走行取消、F5及新程序標題LOAD，無狀態注入。共用模式三項逐byte重編／11拒收，前批新繪模式三項重編／八拒收；24頂層Go含字串、21來源案例／416幀／19反例、883清冊及125組1455來源通過。兩可選GPU略過不計，既有BufferedReader ResourceWarning保留。不新增EXE動態、自然轉職或480頭像驗收。
- 正式只換八項、875保持，100檔全部已存在；原8748美術檔SHA／inode／mtime／owner及0／9／30／59／60／61／62十四項保持，其餘121組SHA相同。Q比例餘項34→30，其餘面容與完整總覽續作，Gaia維持定版。
- GPU 250.876／251.644秒，峰6435373056／6149996544 bytes；普通662.132秒／5924442112 bytes，回歸81.407秒／1570566144 bytes。memory.events全零，均returncode0、無timeout／OOM／kill；共享主機牆鐘不作效能聲明。
- 私人提交2b4af2dc01d277010aaef11a7a0d73f29b6ac933已push，只改catalog與private-manifest兩份清冊，白名單6998。README新增兩組面容、四向實際尺寸及四張完整受控戰場，文化保存內容保持。沒有錄影或新Release；最終引用、工單與Docker清理另記。
- 最終核對30份引用、八張公開圖、24張GPU與十張普通圖；5721個文件目標、修改文件3471個，新增失效連結零。19筆既有文字誤判／歷史交付路徑保持；既有root-owned2808項，本批新增零。遠端工單30／人工23／可能完成零，教訓386／240通過。私人遠端HEAD及PRIVATE已核，FD2容器零殘留。文件腳本的共用章節名稱斷言及稽核容器缺Pillow均已修正後重跑通過，屬收尾腳本／環境問題。
