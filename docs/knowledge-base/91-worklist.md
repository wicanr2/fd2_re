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
- [ ] **動畫逐幀拆解**:sprite 像素 codec 未破(已排除多假設,改走反組譯 oracle)← 使用者明確要求,接續中
- [ ] 建 glyph id → Unicode 對照表(讓文本可全文檢索 / 翻譯)
- [ ] `DATO`(立繪)/`FDSHAP` 24×24 圖塊集解碼
- [ ] 寫一篇總覽:「1995 年台灣怎麼做遊戲 — 炎龍騎士團2 技術全紀錄」

## 第 4 輪以後(暫定)
- [ ] 地圖格式完整解析(FDFIELD 三段對齊)→ 渲染第一張可視地圖
- [ ] 反組譯戰鬥/命中/傷害/AI 演算法(Ghidra)，與攻略公式交叉驗證
- [ ] `FD2.SAV` 存檔加密/格式破解、`FDICON.B24` 格式
- [ ] SoundFont 試聽 + tempo 校準、TIMB→GM 配器對應
- [ ] 選定首個重製技術棧做「讀真資料 → 畫面」垂直切片
- [ ] 反組譯完整性盤點

## 完成定義(反組譯研究)
全部資產格式可解(解包+解壓+轉現代格式)、核心數值表全 dump 並驗證、
主要遊戲規則演算法(戰鬥/移動/升級/AI)有反組譯依據、地圖可渲染、文本可讀可改。
