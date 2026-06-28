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
- [x] **FDICON.B24** = 1680 個 24×24 圖示 tileset(同 FDSHAP 格式,無 LLLLLL 外殼)
- [x] **TAI.DAT** = WxH 圖像(sprite-RLE,如 155×42);多為 UI/特殊圖
- [x] 寫一篇總覽:「1995 年怎麼做出炎龍騎士團2」→ `15`
- [ ] 寫一篇總覽:「1995 年台灣怎麼做遊戲 — 炎龍騎士團2 技術全紀錄」

- [x] **FDFIELD 三段完整解析**:構成(地形)/控制(出場數/回合事件/寶箱/敵我roster)/出場位置;全33圖 metadata → 本機 `extracted/maps/maps_metadata.json`;`tools/parse_field.py`

## 第 4 輪以後(暫定)
- [x] 地圖格式完整解析(FDFIELD 三段)+ 渲染全 33 圖(見上)
- [ ] 反組譯戰鬥/命中/傷害/AI 演算法(Ghidra)，與攻略公式交叉驗證
- [ ] `FD2.SAV` 存檔加密/格式破解、`FDICON.B24` 格式
- [ ] SoundFont 試聽 + tempo 校準、TIMB→GM 配器對應
- [ ] 選定首個重製技術棧做「讀真資料 → 畫面」垂直切片
- [ ] 反組譯完整性盤點

## 完成定義(反組譯研究)
全部資產格式可解(解包+解壓+轉現代格式)、核心數值表全 dump 並驗證、
主要遊戲規則演算法(戰鬥/移動/升級/AI)有反組譯依據、地圖可渲染、文本可讀可改。
