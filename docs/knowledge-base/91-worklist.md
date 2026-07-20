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

## 第 3 輪 ✅(核心全完成;2 零星項 2026-07-05 核實已完成補勾)
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
- [x] 各 track 呼叫端對應確切遊戲狀態名(片頭/世界圖/城鎮/戰鬥/劇情)→ doc12「場景切換時的換曲」已列 5 狀態對映(2026-07-05 核實)
- [x] **FDSHAP 圖塊庫解碼**:標頭 count + u32 offset 表 + bg-RLE 24×24;~300 tiles/tileset → `01`§8
- [x] **全 33 張戰場地圖抽取**:FDFIELD×FDSHAP(配對 map N→FDSHAP[2N],索引驗證全通過)→ 本機 `extracted/maps/`;`tools/extract_maps.py`、`render_map.py`
- [x] **FDICON.B24** = 1680 個 24×24 **地圖單位 Q版小人 sprite**(sprite 4-mode RLE 含透明,**非 FDSHAP bg-RLE**;每角色組12=4方向×3幀)→ `31`
- [x] **TAI.DAT** = WxH 圖像(sprite-RLE,如 155×42);多為 UI/特殊圖
- [x] 寫一篇總覽:「1995 年怎麼做出炎龍騎士團2」→ `15`
- [x] 寫一篇總覽:「1995 年台灣怎麼做遊戲 — 炎龍騎士團2 技術全紀錄」→ `docs/knowledge-base/15-how-fd2-was-made-1995.md`(2026-07-05 核實存在)

- [x] **FDFIELD 三段完整解析**:構成(地形)/控制(出場數/回合事件/寶箱/敵我roster)/出場位置;全33圖 metadata → 本機 `extracted/maps/maps_metadata.json`;`tools/parse_field.py`

## 第 4 輪以後(暫定)
- [x] 地圖格式完整解析(FDFIELD 三段)+ 渲染全 33 圖(見上)
- [ ] 反組譯戰鬥/命中/傷害/AI 演算法(Ghidra)，與攻略公式交叉驗證
- [~] **物品系統反組譯**(M1 用)→ `32`:已確認 物品表23B結構、傷害鏈(AP/DP 全域暫存 0x53c27/0x53c2b → 公式 0x15356)、roster 8裝備欄;[阻] 裝備加成精確累加點(夾攻擊大函式,表base-relative)、使用效果碼待續
- [ ] **轉職系統反組譯**(M4):轉職觸發(教會/道具)、職業數值替換、能力繼承、轉職後成長表切換 → 攻略道具表(勇者徽章→英雄…)交叉驗
- [~] **角色名對應**:補全 portrait→角色名 → `49`。核實後「12 個」已過期,實際已定案 38 組
      (0-31 共 32 + 48/66/68/96/97 共 5 + 本輪新增 126=ASR-06);其餘約 97 組多為泛用怪物/路人,
      對話走場景相依 `-19/-20`(見 `40`),**無法只靠對話反推**,需逐圖解 FDFIELD roster 才能繼續補
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
- [~] 敵方 AI 回合:flood-fill + 評分選目標(擊殺×2),對齊 `11`(0x15140)：已補地形 AP/DP 與原版 `dmg≤2` 跳過門檻；情境加成、狀態倍率待 RE，且已證實原版 `0x157B5/0x150F1` 有 SpellID 評分／執行、`0x15B77` 依 spell family 分流目標。remake 已建立 `State.SpellBook`/`AIPlan.SpellID`、item raw K4 (`0x11`) command inventory、`AIAvailableSpells` 與 `AISpellCandidates`（攻擊／補血／增益／解毒祛麻／敵方狀態）；尚未接原版評分與實際施法行動。
- [ ] 勝敗判定 + **回合推進(回合無上限;上限只由劇本事件 turn>=N 設定,見 `27`§1)**
- [ ] headless 確定性回歸:固定種子打一場 → 結果可重現(驗演算法,不靠手玩)

## M2 — 文字 / 對話層
> 驗收:對話框能顯示 UTF-8 劇情、帶頭像、翻頁;字用 TTF render(不再靠點陣字模)。
- [ ] 工具:`decode_story_text.py --script-json`(35 章 → UTF-8 `script.json`,控制碼→結構)
- [ ] 引擎 TTF 文字渲染(接 `18` 字型現代化:資料化 + TTF + 雙字型模式)
- [x] 對話框 UI ✅(debd52d):原版框素材(LMI1 #21 310×99)+ orig 佈局(下框(5,112)@320/上框鏡射)+
      大側臉頭像(我方左面右/對方右鏡像面左,對映 0x4E8AF/0x4E8E1)+ 白字『』框內換行(≤3行);
      翻頁=campaign story 逐句 Enter。LMI1 #20=單位詳細狀態面板(待用)
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
- [~] 商店節點：原版337筆商品 numeric ID／價格已驗、祕密商店與 town 回返已接、`ClassID`／item type／class equip 白名單、指定收件者與兩階段裝備 prompt 已接；賣出 UI 已接成「Tab→角色→欄位」，`SellSlot` 鎖定原價 75 折並同步移除 equipped flag；`0x1145a/0x1c142` RE 已接入 base+flag 重算與 `<0x80`/`>=0x80` 同類替換；raw `inventory_slots` 保留 source 8 bytes，Load/PartyUnits 依 `0x10f06..0x10f31` materialize 成 runtime 8 slots，內部空槽不再錯移。`0x14237→0x14818` 已鎖定 `range_min` 幾何用途；待：完整 item multiplier/效果碼。
- [~] 戰後 town/整備流程：campaign_full 的 postbattle→town、連戰 preparation 路線與 shop/rumor return 已盤點；`0x318ad` RE 已鎖定 30-byte 勾選表、一般 cap15／late cap19，remake 已接 `party_limit`、`partyDeploy`、save persistence 與可操作 preparation UI，永久 JOIN roster 不被改寫。church `0x3072f` 已證實四個入口，revive fee table、原子 `ReviveUnit`、church selector 與 class-change candidate/branch mutation 已接；尚待完整 GUI 轉職實機回歸與原版數值對照（無免費一般治療）。
- [~] 戰後 town/整備流程：preparation 與 church selector UI 已接；`docs/figures/church-selector.png` 為 xvfb 實機畫面。revive 與 class-change branch/mutation 已可保存 roster/gold；尚待完整 xvfb 轉職操作，以及原版 `+0x22/+0x23/+0x24` DX/race/multiplier 欄位資料化。
- [~] class-change church：已鎖定 `0x3151a..0x3152d` portrait→item 分支、`0x31860` inventory 掃描、`0x1b8e7` item 移除與 `0x31571..0x3157a` class/portrait 寫回；`0x526a7` mapping、`0x2a2e8` 成長重算與 editable branch 已接，待 raw race/multiplier 欄位與實機回歸。
- [~] class-change church：`class_change_targets.json` 已校正為兩層可編輯資料：current portrait 0..0x11→default/optional target（`0x526a7` 以 current portrait 索引，raw `0xff` 不產生 optional branch），以及 target portrait 0x20..0x41→class/mobility increment (`0x615fe`)；portrait 9 的 item 0x5a→target 0x34 special branch 明列。資料完整性與 mutation 已接，下一步追原版 race/multiplier 欄位。
- [~] class-change church：核心 `campaign.ApplyClassChange` 已依 `0x31602` 實作可重現 RNG（row `[min,max)`）、將新職 AP/DP/DX/MaxHP/MaxMP growth **累加**既有值、MV(+0x3b) 累加、保留 Lv、清 EXP、HP/MP 回滿與轉職道具移除；persistent party 已同步保存 MV。target 選擇與 equipment/UI 已接，仍需原版實機數值回歸。
- [~] campaign town/shop 外部交叉盤點：攻略頁明列第4、7、9、14、16、18、19、21章的武器店／道具店／教會／神秘店（來源連結已記入 handoff）；不能由攻略頁推論戰後立即順序，仍以 EXE table 與 `campaign_full.json` 的 postbattle→town/preparation 節點為準，後續測試不得把勝利直接串成下一戰。
- [x] ch02/ch03 story handler slices：ch02_pre/ch02_post 依 count-aligned scene line 範圍播放；ch03_post 接已證實的 ch04 scene3 lines0–3；ch03_pre 已由 jump-table/loadch/FDTXT_004 direct evidence 完成 binding，idx0→scene0 lines0–3、idx1→scene1 lines0–4，`story_ch04` 不再只播兩句 generic fallback。
- [x] ch04/ch05 pre-handler slice：`ch04_pre` 的 FDTXT_005 idx0/1/2 已接 `ch05.json` scene0/1（3+3+9句），map4 50-slot、pan、acting22/21 皆有 binding；`story_ch05` 已由空 cutscene 接回可編輯 handler。
- [x] ch05/ch06 pre-handler：新增 `HandlerDialog.Segments[]` 跨-scene adapter，依 FDTXT_006 #0 的 scene0→1→2→3 targets 展開 18 句；`ch05_pre` 完整 binding，`story_ch06` 已接回可編輯 handler。
- [x] ch06/ch07 pre-handler：FDTXT_007 index0/1（2+6句）與 map6/acting28/29 已接 binding，`story_ch07` 已接回 editable handler。
- [x] ch07/ch08 pre-handler：FDTXT_008 index0/1（跨 scene 15句+2句）與 map7/acting31/32 已接 binding，`story_ch08` 已接回 editable handler。
- [x] ch08/ch09 pre-handler：FDTXT_009 index0/1（2+5句）與 map8/acting35 已接 binding，`story_ch09` 已接回 editable handler。
- [x] ch09/ch10 pre-handler：FDTXT_010 index0 跨 scene0/1（6+6句）與 map9/60-slot 已接 binding，`story_ch10` 已接回 editable handler。
- [x] ch10/ch11 pre-handler：FDTXT_011 index0 跨 scene0/1/2（4+6+2句）、index1/2 scene2 延續，map10/acting38/39 已接 binding，`story_ch11` 已接回 editable handler。
- [x] ch11/ch12 pre-handler：FDTXT_012 index0 跨 scene0/1（2+9句）與 map11/acting40/41 已接 binding，`story_ch12` 已接回 editable handler。
- [x] ch12/ch13 pre-handler：FDTXT_013 index0（6句）與 map12/70-slot 已接 binding，`story_ch13` 已接回 editable handler。
- [x] ch13/ch14 pre-handler：FDTXT_014 index0（4句）與 map13/70-slot、pan 20,20 已接 binding，`story_ch14` 已接回 editable handler。
- [~] ch14/ch15 pre-handler：ch14 的 `roster_has(12)` 動態分支仍待控制流證據；ch15 已接 FDTXT_016 index0（16句）與 map15/60-slot，`story_ch16` 已接回 editable handler。
- [x] ch17/ch18 pre-handler：FDTXT_018 index0/1/2（7+4+13句）與 map17/70-slot、acting54/55 已接 binding，`story_ch18` 已接回 editable handler。
- [~] class-change data/UI bridge：`LoadClassChangeTable`、`ClassChangeTargets`、`LoadClassChangeGrowth` 已接；church 現在先選角色再列 default/optional/special target，Enter 依 branch 消耗物品、套用 RNG stat reset、重算裝備並保存 persistent roster。runtime assets 已補入；待實機 xvfb 走完整轉職流程與校正 HIT/EV/DX synthesis。
- [~] class-change synthesis：`0x31602` 五組 `0x1e529` 先把新職成長加到 raw AP/DP/DX/MaxHP/MaxMP，隨後呼叫 `0x1b750`；該 routine 讀 raw `+0x37/+0x39/+0x3e`、item table 23-byte row 的 `+1/+3/+5/+7`，寫 derived AP/DP/HIT/EV `+0x48/+0x4a/+0x4c/+0x4e`。`RecomputeAfterClassChange` 已恢復並防止既有裝備重複計算；`+0x22/+0x23/+0x24` 是 constructor 清零後由其他 transient/effect writer 使用的旗標，class path 本身不寫入，不能臆測成 class modifiers。
- [~] headless class-change fixture：新增僅由 `FD2_CAMP_CLASS_FIXTURE=1` 啟用的 Lv20 portrait9＋item 0x58/0x5a roster，供 xvfb 依「教會→轉職→角色→target branch」操作驗證；正常遊戲不改變。實機截圖 [`church-class-targets.png`](../figures/church-class-targets.png) 已確認 default 0x29、optional 0x3b、special 0x34 三分支。
- [~] 分支與敗北路線：campaign runner 已有 on_lose→retreat 非 game-over 路徑與測試；battle Node 新增可編輯 `protect` 目標（空值沿用索爾），main 不再硬編碼唯一保護角色；待逐關核對原版保護目標與 retreat 後整備語意。
- [~] 存檔/讀檔(自有格式,非破解原版 `FD2.SAV`)：節點／旗標／金幣／道具／persistent party 已保存；2026-07-20 新增同目錄暫存檔+rename 原子寫入與清理測試，避免 town/shop/preparation 存檔被截斷。仍待完整 GUI/Xvfb 讀檔回歸。

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
- [x] **補完事件語意（2026-07-16 勘誤）**:`0x3453e(idx)=unit_inactive`([0x53a45]+idx*0x50+5 bit0；1=死亡/隱藏/inactive，0=active/alive)；`0x33499`=roster_has(查 [0x53bf7] 我方名冊);**handler 無動作函式**(只條件→設碼+繪圖)→ `25`/`26` 回填。舊記「使用者確認 bit0=存活」已被 constructor/death-path 反組譯推翻並撤回。
- [x] **反思日誌補第 7-10 輪** → `99`
- [x] **挖完 `[0x53bf7]` 表語意**:不是 tile,是**我方隊伍名冊**(32槽×0x50B);`0x33499(id)=roster_has(id)` 查 byte[+8]==角色ID(章16 用)→ `25`/`26` 回填;兩單位陣列釐清([0x53a45]96槽全場 / [0x53bf7]32槽名冊)
- [x] **回合計數釐清**:`[0x53bef]`=回合數(開始1/inc/cmp N),`[0x53ec8]`=累積計數(非回合)；**修正前輪把 [0x53ec8] 當回合**。byte+5 歷史判讀經 2026-07-16 完整反組譯定案為 bit0=inactive。
- [x] **戰鬥規則來源盤點 + 動態驗證清單** → `27`:青衫公式=remake 實作依據+交叉驗證;列出 10 項需 DOSBox 實機驗證(核心 #1-4=戰鬥狀態機旗標/計數語意);新增「回合無上限」需求
- [x] **動態驗證清單更新** → `27`§3:byte+5 bit0 已由反組譯定案為 inactive(1)/active(0)，bit7=已行動；回合=我方全動+敵方AI全動完;7-8用青衫攻略;9-10不需要;3([0x53ec8])低優先。舊「bit0=1 是存活」使用者記憶已撤回。
- [x] **全 30 關卡目標表(攻略 ground truth)** → `28`:每關勝利/失敗/加入條件;**失敗條件=護衛目標**證實 unit_state 機制;加入=roster_has;ch30 魔神連鎖=回合事件;remake 關卡規格直接可用
- [x] **撤回章17 alive 誤讀**:依 `unit_inactive` 重新解讀，指定單位 inactive 才依 jcc 設碼；舊「指定單位存活→設碼」已撤回 → `25`/`26` 回填
- [~] 單位 0x50B 結構:+5(bit0=inactive/bit7=已行動)/+8角色ID/+0/+1/+2/+6/+0x31 已解;完整逐欄佈局 [阻](remake 用自有 struct,不需)
- [ ] (補)更新 doc 12:修正「main=0x10000」、補章節→BGM 表 0x51e63 精確曲號

## 第 6 輪 ✅(戰鬥全螢幕演出畫面 1:1 還原 — 使用者逐項對照;①-④ 全完成 2026-07-05)
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
- [x] **④ BG 草地延伸到 figure 腳下** ✅(2026-07-05 使用者確認 `docs/figures/battle_restore_grid.png` 網格對照:左原版/右 remake 兩邊草地都延伸到 figure 腳下、台座疊綠草非黑底,一致)

## 第 7 輪 ✅(戰鬥演出資料驅動 + 像素級收官,2026-07-02)
> 從「手調對齊」進化到「原版資料驅動」;README 對外展示;全部 push(commit 至 a42ee4a+)。
- [x] **[重大] FIGANI 幀標頭 +0/+2 = 每幀絕對螢幕座標 (dx,dy)@320**(修 doc06 錯誤標註「boundW/boundH」):
      f01=(141,3)/盜賊=(16,41) 與模板匹配 orig 落點完全一致 → 走位/伸擊/突刺全在資料,引擎照貼即可
- [x] **戰鬥演出資料驅動重寫**:meta.json(22 個 FIGANI 全幀 dx,dy)+ loadFigMeta;刪 lunge/錨點手調;
      FIGANI_013 15幀=f01-f10旋轉蓄力/f11黃劈擊/f12-14突刺;盜賊 4 幀待機呼吸
- [x] **打擊感**:命中=全紅剪影交替閃(redSilhouette,orig=VGA色盤閃紅)+ HP 命中窗快抽;5 階段對照 orig 全對上
- [x] **通用化**:newAtkAnim 建構器(所有角色同管線:攻=組×3+1/守=組×3;演出長度隨幀數;命中幀=倒數第4幀通用推定)
- [x] **播放速度接口**:FD2_BATTLE_FPT 環境變數(tick/幀,預設3)+ atkAnim.fpt
- [x] **像素級對齊(模板匹配法)**:三 figure+台座+狀態欄框+三處數字 全部 err=0 且 dx=dy=0;
      狀態欄=原生 149×42 blit(敵(0,154)/我(171,4))、數字=LMI1 #31-40 素材(#42-51綠/#119-128黃=滿血變色)、
      LMI1 混雙 codec(框 0x4e916/小cell 4-mode)、VGA 6-bit palette ×4(decode_lmi 修正)
- [x] **README 對外展示**:battle_restore.gif(orig|remake 同步+網格)、battle_storyboard.png(5階段分鏡)、
      battle_restore_grid.png(網格驗證);新增「戰鬥演出:像素級 1:1 還原」節
- [x] **FD2_SHOT_SERIES 逐幀截圖鉤子**(GIF/分鏡素材管線)
- [x] 名字=TTF 28px+深藍描邊(~85%,既定決策:只有狀態欄數字用點陣素材,其他文字 TTF)

## 第 8 輪 ✅(remake 玩法系統盤點與補完 — 魔法/SFX 已於第7-11輪補完,2026-07-05 核實補勾)
> 使用者指示:檢視腳本系統一路到移動/觸發戰鬥/魔法,盤點缺口逐項補。
- [x] 盤點完成(見下缺口清單)
- [x] **腳本系統 campaign(M4 骨架)** ✅(74bf386):internal/campaign(節點圖 Runner:story/battle/
      choice/event/ending + 旗標 + 敗北路線 + choice 條件選項;單測3條);引擎接線(FD2_CAMPAIGN=1、
      enterNode/campInput/drawCampaignUI、勝敗 Enter 轉場、resetBattle 重試);campaign.json 第一章示範
      (敗北→撤退設旗標→再戰)。待續:商店節點、存檔、原版 33 關自動生成 campaign
- [x] **移動動畫** ✅(74bf386):battle.Path(BFS 路徑)+ walkAnim 沿路徑逐格走(方向幀+OffX/Y 內插,
      ~4-5 tick/格,走完進攻擊/待命,期間鎖輸入);AI 移動沿用瞬移(待接同管線)
- [x] internal/battle 測試失敗已修 ✅(e09c68c):部署格斷言=舊設計殘留,對齊現行(部署格屬 spawn_party)
- [x] **魔法系統** ✅(第7-8輪完成,commit 3c618c4/74366fa:radial 指令環+法術+MP+青衫公式;code: ringInput/castSp/spells.json)——原盤點:法術選單(radial 指令環,doc13 0x18ED0)、MP 消耗、青衫法術公式、
      法術特效動畫(FIGANI 內含法術特效,可沿用資料驅動管線)
- [x] **音樂** ✅(e09c68c):audio.go(ebiten/audio+vorbis;忠實 play_bgm 0x26777:同曲不重播/換曲釋放/
      無限迴圈);campaign 節點 bgm 驅動;FD2_MUTE 靜默。待:非 campaign 模式場景→曲號自動對映(doc12 表)
- [x] **音效 SFX** ✅(第8-11輪完成,cmd/fd2/audio.go;commit e09c68c 音樂+SFX 收線)。資料位置 RE(doc36):`FDOTHER.DAT` 資源 #31(巢狀 `LLLLLL` 容器,14 個 8-bit
      unsigned mono raw PCM 子樣本)+ 戰鬥音效動態 index(同檔案,依攻擊資料決定 index);播放走
      `AIL_init/set_sample_address/set_sample_loop_count/start_sample`(0x26896/0x26945)。
      待:14 子樣本→UI事件對照、戰鬥動態 index 表還原、remake 端接入(SDL_mixer/ebiten audio)
- [x] **radial 指令環** ✅(3c618c4):orig_04 截圖裁 4 圖示(道具/攻擊/狀態/待機),十字繞單位+選中橘框;
      ↑←→↓+Enter/ESC(doc13 [0x3C57]);移動到位自動開環。待:方向↔指令原版精確配對 dosbox 驗證、
      道具 stub、左下 A+05/D+00 攻防預覽小欄
- [x] **魔法系統** ✅(3c618c4):magic.go(spells.json=EXE dump 36條+M1-M5名稱表;InCastRange/Cast
      固定表值傷害/治療capMax);悠妮火炎/電擊/治療;法術選單→射程紫高亮→施放接戰鬥演出+扣MP。
      待:AoE(range>0)、命中率、輔助系(魔刃/風行…)效果。
      ✅ 法術特效對映已 RE 定論(f8fffba 後,doc37):**不存在法術id→FIGANI對映**——施法演出=施法者
      自己的組×3/×3+1(火花燒在 sprite 幀,0x28784 不讀 spell_id;0x2a6bd 跳表=武器屬性UI已排除)
      → remake 現行「施法用角色攻擊動畫」與原版行為一致,不需改
- [~] **商店+祕密商店**: campaign shop 節點與真實 EXE 品項/價格、收件者相容性、兩階段裝備詢問已接線；
      待：賣出、裝備後 AP/DP/HIT/EV 重算、同類舊裝替換、原版祕密商店進入方式 RE(攻略#16 方向鍵位置)
- [x] 存檔/讀檔 ✅(e09c68c):save.go 自有 JSON(節點/旗標/金幣/道具),F5/F9,節點邊界語意

## 第 9 輪 ✅(3-subagent 成本分工;haiku=資料/sonnet=RE·套件/旗艦=架構·驗收)
> 策略(rulebook/45):簡單工作派便宜模型,旗艦只做架構與把關;每件交付先抽驗再 commit。
- [x] **商店品項表**(haiku):docs/data/shops.json 69家/23祕密(含進入方式「酒店前Shift+F1」等);campaign 換真值
- [x] **SFX 破案**(sonnet):FDOTHER#31=14×8bit PCM+AIL 鏈 → doc36;WAV 導出(export_sfx.py,11025Hz 負向證據);
      **index0=游標音確認**(5處方向鍵分支);戰鬥音效=另一獨立池([0x5411f])待導出
- [x] **法術特效定論**(sonnet):不存在法術id→FIGANI對映(0x28784 不讀 spell_id;火花燒在角色幀)→ doc37;
      remake 施法用角色動畫=與原版一致,結案
- [x] **魔法完整版**(sonnet battle 層+旗艦接線):CastArea AoE/命中擲骰(hit=0必中規則)/輔助法術
      (魔刃/魔鎧/風行 doc02 明文值)/毒麻封咒行動術/combo;13 單測;引擎:Buff 進 Attack、TickStatus、
      AoE 指空地、FD2_SEED。缺口列冊:風妖精 dmg=0 矛盾、劍技倍率表、傳送 UI
- [x] **全 33 戰場匯出**(haiku):remake/assets/maps/map1-32(96 檔,抽驗 3 圖合法);
      旗艦接線 loadMap(dir)+campaign battle.map 欄位(map3 實測換圖)
- [x] **AI 行走+敵攻我演出**(旗艦):NextAIPlan 決策執行分離+aiStep;atkOwn 欄位按陣營
- [x] **SFX 引擎接入**(旗艦):loadSFX/playSFX+游標/確認/命中掛點(命中暫代,待戰鬥池)
- [ ] 戰鬥音效池([0x5411f] 動態子容器)導出+逐招對照
- [ ] 非 map0 角色 sprite 組匯出(換圖後 fallback 色塊)
- [ ] 33 關 campaign 自動生成(parse_field+劇情+商店串鏈,M4 工具)
- [ ] UI 音效 index 2-0xb 語意畫面實測

## 第 10 輪 ✅(3-subagent 續批:全流程骨架/素材滿覆蓋/戰鬥音效)
- [x] **全 30 章 campaign 生成器**(sonnet):gen_campaign.py→campaign_full.json(183 節點,
      雙重驗證 python+真 campaign.Load;章→map 順序對應依據誠實);旗艦修 resetBattle fallback
      (scenario 空不再錯載 ch01 → roster 全員登場);ch02/map1 實測 33 單位 ✓
- [x] **sprite/頭像滿覆蓋**(haiku):96 組×12 幀 sprite(全 33 圖需求);旗艦補 5 敵方頭像→384 全滿;
      map3 實測全真 sprite
- [x] **戰鬥音效池 RE**(sonnet):FDOTHER #48-53/64/78/88 九候選 42 WAV(七池 sub0 相同=共用
      揮擊音,md5 抽驗 ✓);[0x5411f] 載入點 0x028110(index=招式id→byte陣列動態);
      **位址勘誤:doc36 全篇 0x11fba→0x111ba**(對齊 doc35)
- [x] **全域文字銳利化**(旗艦):font.go per-尺寸 face 快取,所有 Draw 呼叫自動銳利(糊字根因=非整數縮放)
- [x] **BGM 修正(使用者實聽 oracle)**:FDMUS_018=商店(推翻 doc12「戰鬥」推定);戰鬥曲撤下待聽辨
- [x] **派工 SOP 入 rule**:rulebook/45 新節(haiku=資料/sonnet=RE·套件/旗艦=架構·把關;prompt 要素;把關不可省)
- [ ] **每章 scenario stub**(ch2-30「能玩」關鍵):party 延續+deploy_cells+initial_groups 全開
      (gen_campaign 擴充,回合增援事件之後疊)← 下輪首位
- [ ] 戰鬥曲號聽辨(使用者)+ 各 track 逐曲實聽修正 doc12
- [ ] 戰鬥 SFX:index 陣列填值上游、#48-64 逐招對照、remake 接入(atkAnim 命中掛 battle 池)
- [ ] UI 音效 index 2-0xb 語意畫面實測

## 第 11 輪 ✅(campaign 全 30 章能玩 + SFX 收線)
- [x] **ch2-30 scenario stub**(sonnet):29 個 chNN.json(party 4 人/deploy=own_deploy 真資料
      (9 章資源瑕疵 spiral fallback)/groups 全開排除 group==255 padding);campaign_full 30/30
      掛 scenario(含修 ch01 campaign 模式沒主角隊的壞點);三層驗證+3 章實跑
      → **全 30 章一條龍可玩**(FD2_CAMPAIGN=assets/scenarios/campaign_full.json)
- [x] **戰鬥命中音接真素材**(旗艦):battle 池共用揮擊音(#48 sub0)接命中幀;loadWav/playRaw
- [x] **SFX index2 追蹤**(sonnet,部分解出誠實標記):真路徑=0x01cff0 [esp+計數+0xd0](填值待追);
      **意外收穫:0x1c269=單位 40-bit 招式遮罩解碼器(unit+0x1a)**=「單位學會哪些招式」真實結構;
      battle_sfx_map.json 骨架。依「夠用就停」:+0xd0 續追降低優先(共用音已可用)
- [x] 聽辨清單(extracted/music_ogg/聽辨清單.md,待使用者逐曲填)
- [ ] 戰鬥曲/勝利曲聽辨(使用者)
- [ ] party 數值成長/招募(doc28 加入條件)、回合增援事件疊到 stub
- [ ] ch10 等圖少數 tile 雜色查因
- [ ] unit+0x1a(招式表)vs doc03 +0x22(法術表)offset 差異查證
- [ ] +0xd0 陣列填值(逐招音效對照,低優先)

## 第 12 輪 ✅(招募成長/劇情文本/編輯器規劃/政策更新)
- [x] **gen_campaign v3**(sonnet):26 角色 21 章招募累積(ch30 全 30 人)+ 成長(HP 真表值,
      AP/DP 近似明標);**增援誠實跳過**(battle_events.json 實為勝負 metadata、
      event_id→group 卡未反組譯 0x22e5c,經 ch01 ground truth 反測拒絕硬湊)→
      docs/data/turn_events.json 真資料 dump + doc26 防誤用註記
- [x] **story 管線**(sonnet):story_to_script.py,ch01-03 精校文本 156 句(speaker 對映 78-85%);
      引擎 story script 載入(旗艦:Node.Script+loadStoryScript,無檔 fallback)
- [x] **著作權政策更新(使用者 2026-07-03)**:FD2 版權過期,**對白文本開放入庫**
      (assets/story/ 例外;ch01.json 恢復原文);圖像/音樂/binary 仍本機
- [x] **tile 雜色結案**(sonnet):非 bug——map9 黑塊紫紋=原版地底裂谷美術;
      全 33 圖 index 零越界、匯出 vs oracle 逐像素 0 差異
- [x] **編輯器規劃**(sonnet)→ `38`:選型=獨立網頁單檔編輯器(File System Access API;
      不做 Ebiten 內建=避免編輯器複雜度混入引擎/不外包 Tiled=劇情事件無對應工具);
      MVP=戰場編輯(產物零轉換直接引擎載入);地基發現:MoveCost 未接地形、
      event.go 實作僅 doc29 願景子集(表單以實作為準+--dump-registry 同步)
- [ ] **戰場編輯器 MVP**:網頁單檔 HTML/JS,tile 繪製+單位擺放+部署格;FSA API 讀寫 assets/maps;
      驗收=引擎零轉換載入(細節 `38`)
- [ ] 劇情編輯器:對白+事件表單+商店(下拉=event.go 現行能力,`38` §3.3)
- [ ] 編輯器能力清單同步:Go --dump-registry
- [ ] campaign 節點圖編輯器(拖線/旗標/敗北路線可視化)
- [x] **地形屬性接線**:地形控制表 per-tile 確認(300~400 格不等,非固定 300;
      `tools/dump_terrain_table.py` → `docs/data/exe_tables/terrain.json`,33 tileset 全 dump)。
      移動代碼(byte1,0-5)語意用 references/text/notes.md 玩家攻略「地形移動力/攻防影響」表
      交叉驗證 AP/DP 數值全吻合(森林 code2/3 = -5%/+10%、沼澤 code4 = -5%/-5%)。
      `tools/export_engine_assets.py` 換算 per-tile 步行成本寫入 map.json `"cost"[]`;
      `battle.State.Cost` + `MoveCost` 查表(`remake/internal/battle/move.go`),`Load()` 自動讀
      units.json 同目錄 map.json 接上(main.go 未改動)。全 33 圖 + 頂層 map0 重新匯出。
      新增 6 個測試(`move_test.go`)。**限制**:僅步行成本,騎兵/飛行差異(notes.md 另有數字)
      待 Unit 加兵種欄位才能接;地形 AP/DP 戰鬥加成本輪未接。
- [ ] **0x22e5c RE**(world-map handler:event_id→group 對應)→ 接回合增援
- [ ] ch04-33 劇情文本精校(30 章,PNG 人眼轉錄;對白已可入庫)
- [ ] 視窗縮放 filter 查證(可能 linear 暈染,tile-debug 提醒)

## 第 13 輪 ✅(增援打通/地形/開場實機裁決/文本流水線)
- [x] **回合增援機制全解**(sonnet):0x51b91 58-entry 跳表(0x22e5c 排除);map0 4/4 ground truth;
      extract_event_id_groups.py;turn_events.json 補 groups
- [x] **gen v4 增援疊入**(sonnet):18 章 35 筆 spawn_group(turn 精確比對=原版語意);
      \$turn_counter 展開(3 圖核對);6 筆 \$reg_or_mem 列冊待解;ch08 T0/T4 實跑增援登場 ✓
- [x] **地形接線**(sonnet):FDSHAP 2N/2N+1 配對地形表(4B:寶箱/移動代碼/**戰鬥背景編號**
      =doc35 地形→BG 對應解!);MoveCost 查表+6 測試;main.go 零改動。騎兵/飛行差異待兵種欄位
- [x] **ch04-08 文本**(sonnet):177 句入庫;speaker 編碼文獻化(0-9,A-V→face_portrait)
- [x] **dosbox 開場實機裁決**(sonnet):logo=縮放進場(使用者記憶證實,推翻 doc23 [驗]);
      開場實為 32.3 秒多幕過場(疑 ANI.DAT 驅動,新缺口);選單座標/硬切閃光轉場
- [x] **title 修正**(旗艦):logozoom phase(紅閃→縮入→白閃)+選單實拍座標
- [x] **ANI.DAT 完整 AFM 格式 RE**(sonnet):9 資源=10-opcode 增量繪圖 VM(palette 4 op+
      framebuffer 6 op,直寫 VGA 0xA0000);173B 標頭+8B 幀記錄,289 幀全解無例外(位元組自洽);
      `tools/decode_ani.py`;9 資源逐一視覺比對 doc23 §2.4③ 分鏡全數命中(守護者/索爾/拔劍/
      騎馬夜行/明月/合照/金鎖);**「2」logo 縮放亦由 ANI.DAT(資源#1)驅動**,更正 doc23 猜測。
      見 doc39。待補:⑥浮空城/⑨惡魔臉未逐幀窮舉、轉場閃光呼叫端排程。
- [ ] 開場配樂曲號實聽驗證(容器 nosound 無法驗;使用者聽辨)
- [ ] ch21/22 \$reg_or_mem 增援 eax 來源 RE(6 筆)
- [ ] ch09-33 文本(批次進行中:09-13 執行中)

## 第 14 輪 ✅(AFM 完全破解+開場過場端到端+文本過半)
- [x] **AFM 格式完全破解**(sonnet):10-opcode 增量繪圖 VM(Lo Yuan Tsung 1993);
      派發 0x36c9e/跳表 0x5276a/framebuffer=VGA 0xA0000;289 幀(9 資源)逐位元組驗證;
      decode_ani.py;視覺全命中 dosbox 分鏡(屠龍/logo/金鎖…)→ doc39
- [x] **Go AFM VM 移植+開場接入**(旗艦):internal/afm(容器+VM);執行期解玩家 ANI.DAT
      (不夾帶版權幀);title cutscene 9 幕串接進選單;afm_test 驗幀數 96/51/35;
      無 ANI.DAT 退回 FDOTHER 捲動 fallback
- [x] **AFM 播放器排程 RE**(sonnet):play_afm(index,delayMs,skippable);毫秒校準 0x3dc9f;
      5 呼叫點釘死(開場 3/4/5/6/7/8/0/1,delay 90-15ms;idx0/2=章節過場非開場);
      title.go 換真值排程(拿掉月亮 idx2、各幕 delay、skippable 旗標)
- [x] **ch14-18 文本**(sonnet):229 句;ch01-18 累計 747 句;ch18 永久劇情死亡標記
- [x] **0x1f73f FDOTHER 靜態幕 RE**(sonnet):開場 2 幕靜態=①守護者(#100+pal#99,esi=0x1c2)
      +⑥滿月浮空城(#75+pal#76,esi=0x0a,dosbox frame168-173 逐像素吻合);機制 memset黑→
      載調色盤→blit→淡入→BIOS tick 忙等(修正原 KB「BGM/SFX」誤判);⑨惡魔臉排除是 0x1f73f(待下輪)
- [x] **開場過場插 2 靜態幕**(旗艦):cutScript AFM+static 交錯腳本;frame165 守護者/frame645 浮空城驗證
- [x] **全 33 章劇情文本完成**(sonnet 流水線 6 批):ch01-33 共 1452 句;
      speaker 場景本地表現象文獻化;身世真相(悠妮=ASR-07/大魔王=ASR-06)
- [x] **speaker→頭像機制 RE**(sonnet):0xFFEF operand→0x12C60 查[0x53A45]/[0x53BF7] byte[+7]=DATO;
      三推論裁決(①部分成立=陣列重填+雙定址②怪物表不成立③字母碼是 render_story.py operand 洩漏 bug);
      **story JSON 零修改**(現行最忠實);修 render_story.py operand-skip;doc14 修正
- [ ] **開場配樂曲號 RE**(bgm-title 執行中):play_bgm 開場鏈曲號→FDMUS 檔(取代猜測 FDMUS_004)
- [ ] 開場分鏡⑨惡魔臉來源 RE(疑另一機制或 ANI.DAT)
- [ ] ch21/22 \$reg_or_mem 增援 eax 來源 RE(6 筆)
- [ ] 待展開(位址已釘):0x3453E 額外檢查、tag==0x27 sentinel、[0x53BF7] 表用途

## ⚠ 誠實揭露:全 33 章劇情文本「轉錄完成但從未接進遊戲」(2026-07-03 使用者質疑後查證)

**症狀**:remake 每章開場只顯示 2 句佔位(「第N章:.../目標:...」),1452 句真對白全沒播。
**查證**:campaign_full.json 的 **83 個 story 節點,`script` 欄全部是空的**(0/83 接真對白檔),
而 `assets/story/ch01~33.json` 的 33 章 1452 句轉錄**全都在、全躺著沒用**。
**根因**:各自完成、接線沒人做——
- 「全 33 章文本完成」(story 流水線 6 批)✓ 真的轉錄好了
- 「全 30 章可玩 / campaign 183 節點」(gen_campaign)✓ 節點生成了
- **但 gen_campaign 生成 story 節點時從沒接 `script` 欄** → 兩者從未連起來 ✗
**教訓**:子系統各自報「完成」不等於整合完成;跨模組「接線」要獨立驗(truth-in-code,
配 rulebook/63)。使用者實玩才揭露——沒實玩/沒查,文件會一直顯示「完成」。
**修法**:story_chNN 節點加 `script:assets/story/chNN.json`;gen_campaign 修+重生成 → 全章接通。
- [x] ch01 開場三幕(王城父子/草地悠妮蓋亞/遇海盜)手動接線+轉錄 FDTXT_033/032(intro-scenes)
- [x] **ch01 開場三幕背景圖 RE+接線**(使用者實測發現對白疊在戰場地圖上,非王座廳/草地,2026-07-04):
      RE 修正 doc23 §4 誤記(「FDTXT 序幕『影像』資源」不存在,FDTXT 純文字)——真正背景是
      **暫借章節 32 時 `0x1088d` 順帶載的 FDFIELD 組32(資源96/97/98)= 18×51 複合地圖**(王座廳→長廊→
      草地),與戰場同一 tile 渲染器;已渲染驗證(`extracted/maps/map32.png`)逐像素對齊 dosbox 參考圖
      + 使用者原版錄影。序幕尾端 `[0x53c03]=0` 還原真章節,「遇海盜」對白疊在**真戰場地圖 map0**(非另一
      張圖)。remake 加 `campaign.Node.Map/CamX/CamY`(story 節點固定鏡頭背景圖)+ `main.go` `storyBG` 模式
      (鏡頭不跟游標、不畫單位/游標/HUD);`campaign_full.json` 三節點接線(palace/meadow→map32,
      pirate→map0)。截圖驗證王城幕=雙王座紅毯廳(對照 orig_02_dialog_02_king.png)。
      **教訓**:另一 agent 曾提案「背景已在 BG_BG_\*.png,只需配對」,經抽樣檢視(320×100 全景走廊,
      無王座/紅毯任何痕跡)證偽——套用前先驗證,不可盲信「已抽出」的斷言(rulebook 62)。
      另踩雷:`~/.local/share/fd2_re/assets/`(玩家/測試用資產覆蓋目錄,`assetPath()` 優先讀它)有舊版
      campaign_full.json 快取(缺 ch00_palace/meadow 分幕),測試前先同步 repo 最新版才看得到真結果
      (使用者已驗收+ commit;team-lead 另修 play.sh 每次啟動先清 XDG scenarios/story 影子,一勞永逸)。
- [x] **王座廳 NPC 擺位**(使用者驗收背景後指出「王座是空的、索爾沒出現」,2026-07-04):RE 出 FDFIELD 組32
      出場位置段(資源98)直帶場景 NPC 座標+肖像,同戰場單位 roster 格式;**國王 portrait48@(7,5)+
      王后 portrait66@(10,5)** 頭像圖核對(`DATO_048/066_m0.png`=戴冠鬍鬚男/紫髮女)完全對上
      `f_006.png` 左王/右后。索爾在該格出場位置表無對應項(原版走 0x3231b 內 `push1/3/5;call 0x10b4e`
      另一條登場路徑,未逐一 RE),故索爾位置(fig0 @(8,8) dir2)是目視 f_006 定位、非 FDFIELD 直讀,
      已在 doc23/campaign_full.json 誠實標記。remake 加 `campaign.Actor{Fig,X,Y,Dir}`+`Node.Actors`
      (story+Map 節點靜態擺位,複用 battle.Unit/drawUnitSprite 畫法、無戰鬥邏輯),`story_ch01_palace`
      接 3 actor。截圖對照 f_006 吻合(國王/王后坐正確王座、索爾紅毯中央背對鏡頭)。
      **順帶發現**(未實作,留給 ch02-33 接線時參考):同一出場位置表在草地段(row42/46/47)另有
      portrait0×2(索爾+疑似另一己方角色)+portrait4(亞雷斯)+16 個 portrait68/69 走廊守衛,
      可比照本次做法補草地/走廊 NPC。
- [x] **ch01 開戰隊形 deploy_cells 核對**(使用者指出「索爾隊伍站位都是錯的」,2026-07-04):
      格子座標本身(FDFIELD `own_deploy` 直讀)已驗證正確,問題出在**逐人分配順序**——用 fig sprite
      外觀(fig4=藍盔=亞雷斯/fig9=紅髮=悠妮/fig30=機甲=蓋亞)逐一核對 `orig_03_battle_start.png`/
      `f_029.png`,發現影片是「索爾+亞雷斯緊鄰、悠妮稍右、蓋亞最右」,但 `ch01.json` 原
      `deploy_cells` 陣列順序配上 `party` 順序會把亞雷斯/悠妮的格子配反。交換
      `deploy_cells[1]`/`[2]` 修正,隔離 Xvfb + xdotool 送 Enter 清對白後截圖(before/after 對照)
      確認吻合。**除錯插曲**:FD2_SHOT_CUR 測試一度看似「怎麼設都沒用」,查出是地圖只有 24 格高
      (576px)但視窗 400px,camY clamp 上限只有 176,導致 curY=20/21.5/23 全部 clamp 到同一個畫面
      (誤判無效);換更小的 curY(如 15)才看出真的有作用——clamp 邊界會讓「看似無效的截圖測試」
      其實只是撞到同一個 clamp 上限,不是機制真的沒用,下次遇到「怎麼測都一樣」先檢查 clamp 範圍。
      → doc44 §2.5 定案(信心分級:格子=FDFIELD 直讀高信心,逐人配對=影片外觀反推中高信心非鐵證)。
- [ ] ch02-33 全章 story 節點接 script(gen_campaign 修+重生成)— 等 ch01 落地後做
- [~] ch02-33 全章 story fallback：runtime 對精確 `story_chNN` generic node 自動掛 `assets/story/chNN.json`，讓已匯出的可編輯完整劇本取代節點短 fallback lines；named/pre/post cutscene 不套用此 heuristic，避免重播整章。ch02/03 handler 仍待逐段 beats 接線。
- 🟡 **ch01 開場 Phase 2 實作(doc46 D1-D6,2026-07-04,待使用者驗收才打勾)**:使用者三輪回報後
      team-lead 先做「原版開場逐幕時間軸」(doc46)才動手,這輪照時間軸把 D1-D6 全部實作:
      **D1/D2 背景重構**:`story_ch01_palace` 拆成 `story_ch01_palace_throne`(map32 王座廳)+
      `story_ch01_palace_path`(map32 草地小徑,原「meadow」節點誤用棚)兩幕,`story_ch01_meadow`
      **改名為 `story_ch01_forest_duel`+`story_ch01_forest_discover`,背景從 map32 改指 map31 密林**
      (先前張冠李戴的核心 bug);map31 actor 用 FDFIELD roster 直讀(索爾19,46/亞雷斯19,47/
      蓋亞5,43/悠妮5,44);`portrait75` 是商店店員 NPC，不在 00-41 可入隊角色範圍，**未擺放**。
      **D4 行軍蒙太奇**:新增 `story_ch01_march`(map0,無對白,`auto_advance`
      180 幀自動轉場,索爾走位代表隊伍,簡化版,doc 誠實標「近似非逐幀重現」)。
      **D5 分段播放(核心)**:`campaign.Node` 加 `Scene` 欄(只取 Script 檔 `scenes[]` 裡 label
      對映的那一段,不再攤平全部劇本);改「每段一個 story 節點」而非 Node 內 sub-scenes,
      保留 `FD2_CAMP_NODE` 可跳任一幕驗證。**D6 走位動畫**:`campaign.Actor` 加
      `FromX/FromY/WalkFrames`(進場走位,重用 `battle.Unit.OffX/OffY` 插值)、`Node.ExitWalk`
      (退場走位,索爾沿紅毯走下場~1.5s);新增泛用**淡出/淡入轉場**(`storyFade`,0.6s/次,
      story 節點間一律套用,不再硬切)。**除錯插曲**:forest_duel 一度以為亞雷斯(fig4)沒畫出來,
      加 debug 座標印字才確認兩個 actor 都在正確位置、只是 FDFIELD 給的座標剛好只差 1 格(y46/47)
      造成兩張 24×24 sprite 緊貼——不是 bug,是資料本身就這麼緊,已移除 debug 印字。
      驗證:每幕獨立截圖 + 相鄰幕轉場(throne→path 含退場走位+淡出淡入全程截圖)+
      discover 幕走位動畫三階段截圖(進場遠/中/抵達)+ march 幕靜默→自動轉場→抵達海島全程截圖,
      build+test 綠、gofmt 乾淨。**D8(戰前 MAP/TURN 資訊畫面+行軍確認 UI)不在本輪範圍,已登記獨立項**。
      → doc25 §7.5.1 已修正範圍(戰場進場直接定位仍成立;cutscene 幕內走位是另一機制,已推翻舊結論)。
- [ ] **D8:戰前 UI**(doc46 附帶發現,team-lead 裁定另開項不與本輪合併):原版每場戰鬥開局有
      「MAP·NN TURN·NNN,勝利/失敗條件,ENEMY/FRIEND/NPC 數」資訊畫面 + 「決定要行軍嗎?YES/NO」
      確認,remake `resetBattle` 目前直接進場沒有這兩個畫面。低優先,等 ch1 開場核心問題(D1-D6)
      使用者驗收後再排。

## 待辦:實測回饋(使用者 playtest,2026-07-03)
- [ ] **開場過場節奏 3x 太快 RE**(dragon-fx2 DOS 對比發現,doc39 §10.8):原版魔王立繪捲動
      (esi535→0)貫穿全開場、與各 AFM 幕交錯(暫停播幕→續捲),貢獻 ~16s 延遲;remake 把捲動
      搬到最後單播→開場 5s vs 原版 14.7s。修需先補 0x11eb0/0x1f894 逐指令(捲動如何在 AFM
      直寫 framebuffer 後接回)。使用者已 OK 開頭閃光(#9),此為獨立節奏落差,低優先
- [x] **序章劇本 staging 機制 RE**(使用者指出 #3=劇本機制沒 RE 完整,2026-07-03 反組譯+dosbox 220+ 張連拍
      複驗收尾)→ **定論:主角隊直接定位,原版無行軍動畫**。0x3231b 本體只有直接 spawn(`0x10b4e`)+
      攝影機平移 reveal(`0x13185`/`0x32999`,鏡頭動非單位動)兩種登場原語,dosbox 全程重跑序章開場
      未見任何單位行走動畫或世界地圖段落;玩家記憶「走到地圖中央」疑與攝影機平移視覺效果混淆。
      remake 現行 focusOnParty(純鏡頭對準)+ spawn_party(直接定位)已忠實,#3 非 bug,不需補行軍動畫。
      → `docs/knowledge-base/25-battle-event-system.md` §7.5.1
- [~] **playtest 8 項修正**(playfix agent 執行中,#7=我 kill 誤殺非 bug 已排除):
      #1 方向鍵按住持續移動、#2 預設沒開場動畫、#4 移動後 ESC 取消退回、
      #5 指令環白框沒對齊+中央錯放索爾頭、#6 地圖狀態欄還原原版、#8 單位走完轉回正面朝向
      → **batch1 已 commit(0f32d25)**;#7 非 bug(kill 誤殺);#3 部分(鏡頭對準部隊)
- [ ] **#9 法術特效時序**(待使用者釐清):playfix 靜態審查=攻擊系法術路徑乾淨無殘留;
      真根因疑「治療系法術 target=1 無全螢幕演出(只文字)」→ 打治療咒後緊接敵方攻擊演出,
      被誤認成法術效果延遲出現。修法需先 RE 原版治療咒視覺(閃光/數字浮現/僅改血條),
      或使用者釐清實際現象,不瞎編視覺
- [x] **序章場景轉換打通**(2026-07-04,使用者驗收 OK,commit 2c5adda):王座廳/草地/遇海盜改用
      真 tile 地圖背景(map32/map0)+ 固定鏡頭,非戰鬥圖。RE 定論:0x3231b 暫借章節32 載 FDFIELD
      組32(紅毯雙王座→長廊→草地縱向拼接),與戰場共用 tile 渲染器。story.Node 加 Map+CamX/CamY,
      main.go 加 storyBG 鎖鏡頭/擋游標。→ `23-boot-title-and-scenario-flow.md` §4
- [x] **援軍 stale-cache bug 根因修復**(2026-07-04,使用者報「援軍不該一開始出現在地圖上」):
      根因非 code bug——`~/.local/share/fd2_re/assets/scenarios/ch01.json` 是舊版 `initial_groups=[1,2,10,11]`,
      XDG 快取層優先蓋掉 repo 已修正的 `[1,2]` → group 10/11 開場即 OnField=true 出現。**治本**:XDG 是給
      版權衍生素材(sprites/maps/music)+ 玩家編輯版覆蓋用,scenarios/story 是原創內容不該進 XDG;已刪 XDG
      scenarios/story 影子 + play.sh 每次啟動先清,dev 一律以 repo 為真相。→ 記憶 `fd2-intro-cutscene-bg-and-userdata-cache`
- [x] **過場腳本機制第一性原理解答(doc47)**(2026-07-04,使用者問「RE 為何沒還原 staging」,旗艦親做):
      方案 b 證偽=FDTXT 純對話碼無 staging;方案 a=序章 handler 0x3231b 逐 beat 全轉錄。
      原語翻新:0x135dd=平滑鏡頭平移、0x15f84=對白播放器(doc23 舊判「逐格貼圖」誤)、
      0x1366a=演出(acting)播放器(格式已破,資源容器未定位)、0x112a5=入隊(0/9/4/30)。
      重大:王城→草地=同 map32 鏡頭平移轉場非淡出換景;對白與演出逐條交錯;海島幕 3 個平移點。
      → remake 修正指示 doc47 §4;未解(acting 容器/0x627d8 填表)doc47 §5
- [~] **王座廳 NPC 擺位**(cutscene-bg 執行中):國王/王后坐王座 + 索爾站紅毯中央,對照 f_006.png;
      story 節點加 actor 擺位欄。RE 查 FDFIELD 組32 是否帶 NPC roster(sprite id/cell 直接來自原版)

## 對話框 / 過場打磨(2026-07-05,使用者實玩逐項校正)
- [x] **對話框文字不覆蓋頭像**:上框(頭像在右)文字右緣止於頭像左緣前(commit 57c0e30)→ doc09
- [x] **對話框上下移入畫面**:下框上移(底邊可見)、上框下移(頂邊可見)、頭像置中框內(dc5ebb1)→ doc09
- [x] **框內底色=頭像底色漸層**:框內疊 40,69,138→56,85,154 漸層消接縫色差(dc5ebb1)→ doc09
- [x] **長對白分頁不截斷**:>3 行分頁,Enter 先翻頁翻完才換句;dlgWrap/dlgPageCount/dlgAdvance + dlgPage;
      Go 測試 dlg_test.go 驗全文保全(b81268d)→ doc09
- [x] **進場走位面向修正**:走完面向 actor 目標 dir(亞雷斯走到索爾旁面向他),storyWalkJob.finalDir(aaf5020)→ doc47 §11
- [ ] **⬜ 對話分頁捲動動畫**(使用者要求 2026-07-05;**原版確認有:文字往上捲**):
      目前翻頁是「瞬間切換」;要改成**平滑往上捲動**——按 Enter 翻頁時,當前內容往上捲出、下一頁從底部捲入。
      **使用者明示:不用依賴原版機制,自己寫平滑捲動即可**(原版有此效果,但捲動速度/幀數自訂,非 RE 值)。
      實作方向:翻頁時啟一個 `dlgScrollT` 計時器(數幀),繪製時把文字整體 y 偏移從 0 平滑插到 -行高×3、
      同時畫「當前頁下移出 + 下一頁自底部進」,期間 clip 在框內矩形;捲完才定位到新頁。
      動 `cmd/fd2/main.go` 對話繪製區 + `dlgAdvance`(翻頁時觸發捲動而非瞬間 dlgPage++)。
- [ ] **⬜ 自動結束回合**(使用者要求 2026-07-05,不急):目前 remake 要**手動按 Tab** 才換回合;
      原版是**全員行動完自動結束回合**——我方(玩家操作的 + NPC/友軍)全部移動/行動完 → 自動換敵方;
      敵方全部移動完 → 自動換回我方。實作:每次單位行動後檢查該陣營是否還有「可行動」單位
      (未行動 flag,見 doc25/單位 +5 bit7=已行動),若無則自動 endTurn 換陣營;移除或保留 Tab 當「提前結束」快捷。
      需對照原版:①是否有「跳過剩餘我方單位」的手動提前結束 ②敵方 AI 動完的判定時機。動 `battle` 回合狀態機 + main.go。
- [~] **handler 後半段 beats 解碼**(sonnet subagent 執行中 acb94c2):庭院/森林段走位/對話/fade 編排,
      供重建 palace_path/forest 節點(Ares 進場對話框位置、逐段走位轉向、索爾練劍、領頭跟隨、fade 換場)

## 完成定義(反組譯研究)
全部資產格式可解(解包+解壓+轉現代格式)、核心數值表全 dump 並驗證、
主要遊戲規則演算法(戰鬥/移動/升級/AI)有反組譯依據、地圖可渲染、文本可讀可改。
