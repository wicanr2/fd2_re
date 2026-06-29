# 91 — Worklist(逐輪更新，依序執行)

> 目標:完成《炎龍騎士團2》反組譯研究，並考證當年開發工具。
> 每輪結束更新本表(打勾 / 補新項 / 調整順序)，與 `99-reflections-log.md` 互補。
> 圖例:✅ 完成 · 🟡 進行中 · ⬜ 待辦 · ❌ 放棄(註明原因)

## 第 1 輪 ✅
- [x] 素材盤點(`FD2.EXE` + 12 `.DAT` + 音效驅動)
- [x] 破解 `.DAT` 容器格式 + 寫 `tools/unpack_dat.py`
- [x] 辨識圖像/調色盤/文本/地形表 header
- [x] 攻略萃取成知識庫
- [x] 建知識庫骨架 + RE 計畫 + 反思 + README + git push

## 第 2 輪 ✅
- [x] **當年開發工具考證**(Watcom C/C++32 + DOS/4GW + Miles AIL v3 / XMIDI + AFM 動畫工具/作者 Lo Yuan Tsung)→ `04-original-toolchain.md`
- [x] 建立本 worklist
- [x] **EXE 資料表 dump**:`tools/dump_exe_tables.py`,9 表全對齊「舊版」offset,5 表 dump 並自驗全通過 → `docs/data/exe_tables/`、`03-…`
- [x] **圖像解壓**:破解 RLE(c≥0x80 literal / c<0x80 run),`tools/decode_image.py` 渲染標題+背景驗證 → `05-image-compression-format.md`
- [x] **音樂解析**:確認 XMIDI,`tools/xmi2mid.py` 轉 15 首標準 MIDI(note 平衡、tempo 直通)→ `07-music-xmidi-format.md`
- [x] **動畫機制結構**:AFM 容器 + FIGANI 幀封裝(幀數自描述 + offset 表)→ `06-animation-format.md`

## 第 3 輪 🟡(進行中)
- [x] **文本解碼**:破解 FDTXT(uint16 glyph 索引 + 控制碼 + 0xFFFF)+ 找到自製字型(FDOTHER_004,16×16 1bpp,1824 字模),**還原可讀中文** → `08-text-and-font-format.md`、`tools/decode_text.py`
- [x] **動畫逐幀拆解**:✅ **完整破解**!反組譯參數化解碼器 0x4F43D + 解出 13-byte 幀標頭(realW/H 在 +9/+11)+
      4 模式 RLE → `tools/decode_figani.py` 把 **264 動畫 2118 幀**全部解出(騎士揮劍動畫視覺驗證)← 使用者明確要求,完成
- [x] **持久素材抽取**:`tools/extract_all.py` → 本機 `extracted/`(raw/images/animations/music/fonts);**不入版控**
- [x] **劇情/對話結構解出**:[控制碼][說話者肖像ID][『][對白][』];全 35 章渲染成可讀 PNG(`extracted/story/`)→ `09-…`
- [x] **序章(FDTXT_001)逐章轉錄完成**(`extracted/story/序章_transcript.md`,本機)
- [x] **敵/我方動畫機制文件**:解碼器變體家族(全彩/remap調色/silhouette/dither)+ 陣營/面向 → `10-…`
- [x] **敵人/NPC 戰場 AI** 反組譯文件(0x15140 評分決策)→ `11-…`
- [x] **音樂播放與場景切換**機制(AIL XMIDI 序列)→ `12-…`
- [x] **戰場選單與行動系統**(行動狀態機/選單游標/Get_EasyMagic)→ `13-…`
- [x] README 知識庫總索引(可點選分類)
- [x] **glyph→Unicode 對照表完成(1824/1824,100%)** → `docs/data/glyph_map.json`(含數字/英文/漢字/標點/機器人雙字元代號)
- [x] **全 35 章劇情轉錄完成**:自動解碼成含說話者的 UTF-8 → 本機 `extracted/story/full_story_auto.md`(1450 句);序章~第3章另有人工精校
- [x] **按鍵綁定**(Enter/Space 確認、ESC 取消、方向鍵)反組譯 → `13-…`
- [x] **Get_EasyMagic** 法術面板反組譯(0x18ED0)→ `13-…`
- [x] **場景→曲號對映**:play_bgm(0x26777)+ 32 處呼叫 track 反組譯 → `12-…`
- [x] **LE fixup xref 工具**(`tools/le_xref.py`)解開 DOS4GW 重定位,可做 data xref
- [x] **控制碼語意還原**(反組譯文本渲染器 0x16D00-0x17200):FFEF/EE/ED/EC=開對話框(FFEF 帶 DATO 頭像)、
      FFFE=換行、FFFD=翻頁等鍵、FFFF=結束 → `09`;副產物確認 **DATO.DAT=人物頭像**
- [x] **劇情校對**:解碼自驗 + 上下文揪出 14 處形近字模誤判並修正
      (脅/實/黨/費/鍛/輩/辭/摸/牢/樁/紮/襲/態/責)
- [x] **陣營/狀態 remap 配色**:確認 LUT 來源=FDOTHER 資源#3(LMI1,23張256-byte LUT),dump 並套用展示(LUT0灰=已行動…)→ `10`;BB→LUT索引精確對應待續
- [x] **DATO 頭像全解**(136×4嘴型幀)→ `01`§7;**Unicode→glyph 反向表+編碼器**(round-trip 100%)→ `tools/encode_text.py`
- [ ] 各 track 呼叫端對應確切遊戲狀態名(片頭/世界圖/城鎮/戰鬥/劇情)
- [x] **FDSHAP 圖塊庫解碼**:標頭 count + u32 offset 表 + bg-RLE 24×24;~300 tiles/tileset → `01`§8
- [x] **全 33 張戰場地圖抽取**:FDFIELD×FDSHAP(配對 map N→FDSHAP[2N],索引驗證全通過)→ 本機 `extracted/maps/`;`tools/extract_maps.py`、`render_map.py`
- [x] **FDICON.B24** = 1680 個 24×24 **地圖單位 Q版小人 sprite**(sprite 4-mode RLE 含透明,**非 FDSHAP bg-RLE**;每角色組12=4方向×3幀)→ `31`
- [x] **TAI.DAT** = WxH 圖像(sprite-RLE,如 155×42);多為 UI/特殊圖
- [x] 寫一篇總覽:「1995 年怎麼做出炎龍騎士團2」→ `15`
- [ ] 寫一篇總覽:「1995 年台灣怎麼做遊戲 — 炎龍騎士團2 技術全紀錄」

- [x] **FDFIELD 三段完整解析**:構成(地形)/控制(出場數/回合事件/寶箱/敵我roster)/出場位置;全33圖 metadata → 本機 `extracted/maps/maps_metadata.json`;`tools/parse_field.py`

## 第 4 輪以後(暫定)
- [x] 地圖格式完整解析(FDFIELD 三段)+ 渲染全 33 圖(見上)
- [ ] 反組譯戰鬥/命中/傷害/AI 演算法(Ghidra)，與攻略公式交叉驗證
- [~] **物品系統反組譯**(M1 用)→ `32`:已確認 物品表23B結構、傷害鏈(AP/DP 全域暫存 0x53c27/0x53c2b → 公式 0x15356)、roster 8裝備欄;[阻] 裝備加成精確累加點(夾攻擊大函式,表base-relative)、使用效果碼待續
- [ ] **轉職系統反組譯**(M4):轉職觸發(教會/道具)、職業數值替換、能力繼承、轉職後成長表切換 → 攻略道具表(勇者徽章→英雄…)交叉驗
- [ ] **角色名對應**:看 character_summary.png 補全 140 組 portrait→角色名(目前已知 12 個)
- [x] `FDICON.B24`=1680個24×24地圖單位sprite(sprite-RLE,見 `31`);`TAI.DAT`=WxH圖像(sprite-RLE)
- [~] `FD2.SAV` 存檔:熵7.99=強加密/壓縮,無結構;破解需反組譯存檔加解密常式(大工程)。重製用自有存檔格式 → 低優先
- [x] **音色合成評估+MT-32實證**(SoundFont/MT-32/版本切換,munt渲染15首)→ `16`
- [x] **擴充劇本/玩法可行性評估**(戰場/對話/商店/機制)→ `17`
- [~] SoundFont/MT-32 → 見 `16`(MT-32 已渲染);SoundFont 試聽 + TIMB 配器對映待補
- [ ] 選定首個重製技術棧做「讀真資料 → 畫面」垂直切片
- [ ] 反組譯完整性盤點

## 重製前置(規劃/實作)
- [x] **音樂預錄 OGG**(MT-32 音源):15 首 → 本機 `extracted/music_ogg/`;`tools/export_music_ogg.sh`
- [x] **字型現代化規劃**(UTF-8 + TTF render)→ `18`(計畫:文字資料化 + TTF + 雙字型模式)
- [x] **劇本/關卡腳本系統設計**(可分支節點圖/敗北路線/商店/旗標)→ `19` + `docs/data/campaign_sample.json`
- [ ] 實作:`decode_story_text.py --script-json`(35 章 → UTF-8 script);重製文字層 TTF render
- [ ] 實作:從原版資料自動生成「線性 campaign.json」(parse_field + 劇情 + 商店)→ 原版模式
- [ ] 實作:引擎 ScenarioRunner 狀態機(節點/轉場/旗標)
- [x] **第一性原理可行性確認** → `20`(9 項必要能力全具備,降為工程整合)
- [x] **Go/Ebiten 重製架構規劃** → `21`(桌面/Web/手機)
- [x] **重製 MVP 垂直切片**:Ebiten 載入序章地圖+渲染+游標(方向鍵/WASD/觸控)→ `remake/`
- [x] **技術驗證報告** → `22`(桌面 ELF 10.8MB + WASM 10.5MB + 資產管線,三項全通)

---

# 重製 worklist(Go/Ebiten,本機優先;依序執行)

> **詳細工作拆解(WP/輸入/產出/驗收/衝刺)見 `30-remake-work-breakdown.md`**。本表為里程碑總覽。

> 策略:**先把本機桌面執行檔做成能完整玩,再回頭處理網頁/手機打包**(`22` 已證三平台都可建)。
> 每個里程碑 = 一個可執行、可驗收的切片。完成才往下一個。架構見 `21`,可行性見 `20`。

## M0 — 引擎骨架 ✅(已完成)
- [x] Ebiten 專案 + Docker 建置流程(`remake/`,`go.mod`/`go.sum`)
- [x] 載入地圖渲染(tileset.png + map.json,offset 表定位無漂移)
- [x] 游標移動 + 相機跟隨(方向鍵 / WASD / 觸控)
- [x] hi-res 畫布(640×400,CJK 拉畫布不縮字)
- [x] **本機桌面執行檔建成(Linux ELF)** + WASM 編譯成功 → `22`

## M1 — 戰棋核心(下一個,讓它「能玩一場戰鬥」)
> 驗收:能部署我方、選單下指令、移動(flood-fill 範圍)、攻擊結算傷害、敵方 AI 回合、判定勝敗。
- [x] 資料模型:Unit(HP/攻防/移動力/陣營/位置/alive/acted)、BattleState(回合/單位) → `remake/internal/battle/model.go`
- [x] 單位資料管線 `tools/export_units.py`(roster+座標+EXE數值→units.json)+ 引擎載入並渲染(陣營色塊+HP bar+選中資訊)+ headless test 全綠
- [x] 移動:flood-fill 可達範圍 + 高亮 + 選取/移動/待機(`move.go`);地形成本待接
- [x] **地圖單位 sprite=FDICON Q版小人**(24×24 待機動畫)→ `31`(取代誤用的 FIGANI 全身)
- [ ] 戰場選單狀態機(移動/攻擊/待機/道具/結束),對齊 `13`(游標/Enter/ESC)
- [ ] 攻擊結算:套**青衫公式**(物理/劍技/法術/恢復+命中+暴擊+經驗,doc 02 §4 = 實作依據)+ EXE 數值表(`03`)
- [ ] 敵方 AI 回合:flood-fill + 評分選目標(擊殺×2),對齊 `11`(0x15140)
- [ ] 勝敗判定 + **回合推進(回合無上限;上限只由劇本事件 turn>=N 設定,見 `27`§1)**
- [ ] headless 確定性回歸:固定種子打一場 → 結果可重現(驗演算法,不靠手玩)

## M2 — 文字 / 對話層
> 驗收:對話框能顯示 UTF-8 劇情、帶頭像、翻頁;字用 TTF render(不再靠點陣字模)。
- [ ] 工具:`decode_story_text.py --script-json`(35 章 → UTF-8 `script.json`,控制碼→結構)
- [ ] 引擎 TTF 文字渲染(接 `18` 字型現代化:資料化 + TTF + 雙字型模式)
- [ ] 對話框 UI(開框/翻頁/換行,對齊 `14` 控制碼語意)
- [ ] DATO 頭像接入:**4 嘴型幀 m0~m3 對話動畫(嘴開合+眨眼)**,非單張(對齊 `01`§7 / `31`§7);播放時機待反組譯 0x16D00

## M3 — 音訊層
> 驗收:戰鬥/城鎮/劇情切場景時 BGM 正確切換,用預錄 OGG(MT-32 音源)。
- [ ] OGG 串流播放(15 首,來源 `extracted/music_ogg/`)
- [ ] 場景→曲號對映(對齊 `12`,play_bgm 邏輯)
- [ ] (選配)SoundFont/MT-32 版本切換開關 → `16`
- [ ] 音效(SFX)接入

## M4 — 腳本系統 / 流程串接
> 驗收:序章→商店→分支→下一關 能一條龍跑完;戰敗走不同路線而非 game over。
- [ ] 工具:原版資料自動生成「線性 campaign.json」(parse_field + 劇情 + 商店)→ 原版模式
- [ ] 引擎 ScenarioRunner 狀態機(節點/轉場/旗標),對齊 `19` + `campaign_sample.json`
- [ ] 商店節點(買賣/裝備)
- [ ] 分支與敗北路線(on_lose → 敗北關卡,非結束)
- [ ] 存檔/讀檔(自有格式,非破解原版 `FD2.SAV`)

## M5 — 內容完整化(原版可破關)
> 驗收:從序章玩到結局,全 33 戰場 + 全劇情 + 商店,正常玩法可達(無 debug hook)。
- [ ] 匯出全 33 戰場為引擎資產 + 全單位/數值表接入(對齊 EXE 表 `03`)
- [ ] 全劇情/對話接入(35 章)
- [ ] 完整性盤點:對照原版,缺漏列冊(`83` 完整性 > 投報)
- [ ] 正常玩法可達性驗證(連通/可破關鏈,參考 skill 踩雷)

## M6 — 跨平台打包(回頭做網頁/手機)
> 驗收:Windows/macOS/Linux 桌面包 + 網頁 + Android APK。
- [ ] 桌面交叉編譯 + 打包(Windows `.exe` / macOS `.app` / Linux AppImage)
- [ ] WASM 上網頁(資產載入 + `index.html` 完整化)
- [ ] Android:`ebitenmobile bind` → `.aar` → Gradle APK(觸控已支援)
- [ ] 玩家向 README(圖文並茂,突顯貢獻)+ 工程文件分離

## 擴充(M4 之後,擺脫原版固定 33 路線)
- [x] **可擴展事件系統規劃** → `29`:trigger/when/do DSL + 文本事件控制碼 `{{}}`;條件/動作 Registry 可註冊;原版 30 關可表達+自創戰役
- [ ] 實作 EventSystem(ConditionRegistry/ActionRegistry)+ DialoguePlayer 解析 `{{}}`
- [ ] 自創戰場 + 自訂劇本(用 `19`+`29` 系統)
- [ ] 多分支劇情線 / 多結局
- [ ] 編碼器回寫中文(`encode_text.py`)做在地化/二創

## 第 5 輪 ✅(開場流程反組譯 — 使用者指定)
- [x] **建反組譯器** `tools/disasm_le.py`(capstone 解 DOS4GW LE,docker)+ 確認 entry/main/狀態機
- [x] **頂層狀態機反組譯**:真 main=0x25bf4(雙層迴圈),核心狀態變數 `[0x53c03]`=章節,兩張章節跳表(0x51d71 戰前劇情 / 0x51de9 戰後)→ `23`
- [x] **標題序列**:角色立繪 5 幀(FDOTHER #0x45-0x49,320×147)垂直捲動(非旋轉)+ FLAME DRAGON logo(#7 sub0)+ 主選單;**解碼器當 oracle 解圖視覺驗證** → `23`
- [x] **主選單機制**:輸入迴圈/scancode dispatch(↑0x48/↓0x50/Enter/Space)/游標 wrap → `23`
- [x] **新遊戲→開場對話→自動進戰場**:[0x53c03] 章節驅動,cutscene 0x3231b(與前代主角對話)→ 戰場地圖=章節*3+2(自動串接)→ `23`
- [x] **call-graph 遞迴反組譯工具** `tools/callgraph_le.py`(可達集/callers/rpath/funcof/jtab)→ `24`
- [x] **釘死 cutscene→戰場鏈**:0x10010 真 caller=0x1a251/0x26130,路徑 main→0x25ebb→0x10010,獨立驗證章節跳表(修 data 段 fixup)→ `24`;排除偽命中 0x1b051/0x26f30
- [x] **[0x53ecc] 戰後/事件完整狀態機**:事件解譯器(0x205c9-0x20c64,28處設1/2)↔戰役迴圈(==1進世界圖/中場 0x22e5c、==2勝利→戰後跳表+結局判定+下一章)→ `24`§6
- [x] **挖完事件指令集** → `25`:第三張章節跳表 0x51b19(戰場事件,30章/18 handler)、FD2 事件=每章 C handler 非 byte-code、事件原語(0x3453e 查單位/0x205be prologue/回合數=[0x53bef])
- [x] **逐關挖 18 特殊 handler** → `26` + `tools/event_handler_dump.py` + `docs/data/battle_events.json`(30章條件→動作,供 remake 去 hardcoding)
- [x] **補完事件語意**:`0x3453e(idx)=unit_state`([0x53a45]+idx*0x50+5 bit0=**存活**;使用者確認,非陣亡);`0x33499`=roster_has(查 [0x53bf7] 我方名冊);**handler 無動作函式**(只條件→設碼+繪圖)→ `25`/`26` 回填
- [x] **反思日誌補第 7-10 輪** → `99`
- [x] **挖完 `[0x53bf7]` 表語意**:不是 tile,是**我方隊伍名冊**(32槽×0x50B);`0x33499(id)=roster_has(id)` 查 byte[+8]==角色ID(章16 用)→ `25`/`26` 回填;兩單位陣列釐清([0x53a45]96槽全場 / [0x53bf7]32槽名冊)
- [x] **回合計數釐清**:`[0x53bef]`=回合數(開始1/inc/cmp N),`[0x53ec8]`=累積計數(非回合);**修正前輪把 [0x53ec8] 當回合**+**撤回 byte+5=陣亡**(初始化=1=狀態旗標)→ `25`/`26` 回填
- [x] **戰鬥規則來源盤點 + 動態驗證清單** → `27`:青衫公式=remake 實作依據+交叉驗證;列出 10 項需 DOSBox 實機驗證(核心 #1-4=戰鬥狀態機旗標/計數語意);新增「回合無上限」需求
- [x] **動態驗證清單經使用者領域知識定案** → `27`§3:byte+5 bit0=存活(HP>0)/bit7=已行動;回合=我方全動+敵方AI全動完;7-8用青衫攻略;9-10不需要;3([0x53ec8])低優先 → **DOSBox 驗證實質取消**
- [x] **全 30 關卡目標表(攻略 ground truth)** → `28`:每關勝利/失敗/加入條件;**失敗條件=護衛目標**證實 unit_state 機制;加入=roster_has;ch30 魔神連鎖=回合事件;remake 關卡規格直接可用
- [x] **撤回「擊敗Boss=勝利」誤讀**:章17 反組譯確認 unit_state=「指定單位存活→設碼」(非擊敗),早先 bit0=陣亡錯誤已清 → `25`/`26`/記憶回填
- [~] 單位 0x50B 結構:+5(bit0存活/bit7已行動)/+8角色ID/+0/+1/+2/+6/+0x31 已解;完整逐欄佈局 [阻](remake 用自有 struct,不需)
- [ ] (補)更新 doc 12:修正「main=0x10000」、補章節→BGM 表 0x51e63 精確曲號

## 第 6 輪 🟡(戰鬥全螢幕演出畫面 1:1 還原 — 使用者逐項對照)
> 目標:remake 戰鬥攻擊演出(orig_05)像素級對齊原版。方法=**密集網格疊圖 oracle**(見記憶 pixel-align)+ 反組譯確認機制,**無 dosbox debugger**(0.74 vanilla 不能 dump)。
- [x] **完整 RE 戰鬥演出繪圖機制** → `35`:演出主函式 0x28a6c、blit 0x4e63d(無縮放/無翻轉,尺寸朝向燒進素材)、固定錨點(164,157)、phase [0x540ff]、BG 多層(0x52381=BG.DAT)、戰場→BG 表 0x52363
- [x] **figure 幀/姿態**:我方亞雷斯=攻擊動作1 `FIGANI_013_f01`(組×3+1,人眼確認);幀序播放;守方不翻轉(FIGANI_288 原圖面右)
- [x] **白斬擊弧 = FIGANI 攻擊幀自帶**(燒 sprite),移除程式 vector 補弧
- [x] **[設計鐵則] 我方=背影+腳下台座 / 敵方=正面**(使用者確認,與攻守無關純陣營)→ `35`§3.2.5
- [x] **figure 位置對齊**(密集網格+程式量土台中心):我方土台中心 x≈238、敵方腳 y≈135(@320)
- [x] **狀態欄對齊**:名字放大(16px視覺)、血條加長(緊接標籤到數字)、bevel 立體框、HP/MP淺藍標籤、暗槽色暗版、上下欄位置/間隔(我方離頂、敵方離150線空隙)
- [x] **z-order RE**:演出順序狀態欄(0x28ce7/0x28d62)先、figure(0x28e76/9a/ee0)後 → figure 蓋住狀態欄;remake 改 BG→狀態欄→figure → `35`§4.-1
- [x] **狀態欄機制 RE**(agent):真繪製器 0x18c6d(非 0x29164);框=素材sprite、HP=逐欄cell(len=curHP×101/maxHP)、名=16×16點陣字、數值=6px digit cell → `35`§4
- [x] **清除錯誤斷言**:土台正名 FIGANI自帶→**TAI.DAT 台座**(0x29164 載 0x28c46);figure-X=word[unit+0x40] 誤讀;對話框開框碼 0x16F40
- [x] **① TAI.DAT 台座解碼 + remake 貼上** ✅(v23):TAI_004=154×42 綠草橢圓台座(decode_sprite 解 body[4:],index0透明17%);remake 載 tai_004.png 貼我方腳下(z:狀態欄<台座<figure);對齊 orig 取代偏灰 dither。確切 entry↔職業/地形對映待後續
- [x] **② 複查 `+0x40` 衝突** ✅(第一性原理):**+0x40=當前HP**(0x18c98 血條 `word[+0x40]×101/word[+0x42]`=HP% 鐵證);figure lunge 位置實際讀 **+0x48/+0x4a 螢幕投影**(0x29f72 不用+0x40);agent-A「戰場格X」誤標已清。+0x44/46=MP/maxMP。doc35 §2.2/§4/§7 全修正
- [x] **③ 狀態欄框/HP 用真素材** ✅(v25-26):破解 FDOTHER#5 LMI1 sub-sprite codec(反組譯 0x4e916:c≤0xC0 literal/c>0xC0 run,新 codec,`tools/decode_lmi.py`);框=#22(149×42 含bevel+標籤+槽)貼 panel.png、血條 cell=#27-30;修 HP靠左(槽 x21-123)/提亮/數字對位。doc35 §4.2.5。盜賊 y 軸對齊(276→296,頭頂偏上一排)
- [ ] **④ BG 草地延伸到 figure 腳下**(讓台座/陰影疊綠草,非黑底)

## 完成定義(反組譯研究)
全部資產格式可解(解包+解壓+轉現代格式)、核心數值表全 dump 並驗證、
主要遊戲規則演算法(戰鬥/移動/升級/AI)有反組譯依據、地圖可渲染、文本可讀可改。
