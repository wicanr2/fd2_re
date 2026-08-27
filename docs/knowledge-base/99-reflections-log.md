# 99 — 逐輪反思日誌(Lessons Learned)

> 每一輪結束寫一則：做了什麼、學到什麼、哪個前輪結論被推翻。錯誤知識要回頭刪。
> 本篇是不可用來計算完成度的歷史時間線；目前分層狀態以
> [`58`](58-fd2-exe-re-coverage.md)為準，系統與 UI 分別看 `56`／`57`，有效工作
> 只看 `91` 檔首。搜尋命中本檔的舊未知或下一輪，不可單獨觸發重新反組譯。

## 第 1 輪 — 素材盤點 + 容器格式破解 + 計畫(2026-06-28)

**做了什麼**
- 盤點 `FLAME2/`：主程式 `FD2.EXE`(DOS4GW LE / Watcom)、12 個 `.DAT` 資產、Miles 音效驅動。
- 破解並驗證 `.DAT` 通用容器格式(`LLLLLL` magic + uint32 offset 目錄)，寫成 `tools/unpack_dat.py`，
  成功解出全部 12 容器共 ~1000 個資源。
- 辨識多種資產 header：320×200/320×100 圖、24×24 圖塊、768B VGA 調色盤、FDTXT 兩層字串結構、
  FDSHAP 地形控制表(0x2422E 與攻略吻合)。
- 把青衫攻略萃取成結構化知識庫(裝備 198 / 法術 35 / 人物 32 / 公式 / EXE 表 offset)。

**學到 / 驗證**
- 漢堂的封裝高度一致 → **一支解包器吃所有 `.DAT`**，省下逐檔逆向。
- 攻略的 modify1/modify2 等於社群已做的「部分 RE 地圖」(EXE offset + 結構)，是極佳 oracle，
  但**有新舊兩版 offset**，不能照抄位址，要用錨定特徵在實檔重新定位。
- 「容器 offset 解出的資源數」與「攻略宣稱的 per-map 指標數」尚未對齊(FDFIELD 100 資源 vs 3 指標/圖)，
  列為 [假設] 待第 2 輪驗證——**避免把未對齊的推論寫成定論**。

**推翻的舊結論**：無(第 1 輪)。

**當時的下一輪起點**：EXE 表 dump（錨定特徵 grep）＋圖像解壓（RLE?）＋FDTXT Big5 確認。
當時引用的早期計畫已於 2026-08-27 刪除；目前工作入口改看 `58` 與 `91`。

**待消除/存疑清單**(後輪確認後更新或刪除)
- [假設] 圖像壓縮為 RLE — ✅ 第 2 輪已證並破解。
- [假設] FDTXT 字串為 Big5 — 未逐字證(第 3 輪)。
- [已更正] `FDICON.B24` 已解為 24×24 four-mode RLE；`FD2.SAV` 已解為 rolling-XOR/checksum envelope（record bytes 仍逐欄位 RE）。
- [假設] DATO=立繪、TAI/ANI/FIGANI 內部結構 — FIGANI 結構已解,像素透明 RLE 與 DATO 未解。

## 第 2 輪 — 開發工具考證 + EXE 表 + 圖像/音樂解碼(2026-06-28)

**做了什麼**
- **開發工具考證**:從 binary 指紋確認 Watcom C/C++32 + Rational DOS/4GW + Miles AIL v3(XMIDI 音樂)。
  並在 `ANI` 資源 #0 找到自製動畫工具 **AFM v1.00,作者 Lo Yuan Tsung,1993** 的版權橫幅(珍貴史料)。
- **EXE 資料表**:當前 EXE 完全對應攻略「舊版」offset(9/9 錨定對齊);dump 物品(215)/法術(36)/敵我(68)/
  升級成長(68)/職業魔抗暴擊(26),對攻略字面值**自驗全通過**。攻略原缺的法術數值編號已從 EXE 還原。
- **圖像壓縮**:破解 RLE(`c≥0x80`→literal `(c&7f)+1`;`c<0x80`→run `c+1`)。以「輸出必為 W×H」為強約束,
  標題畫面、戰鬥背景(山脈/村莊/熔岩)正確渲染。~125 張全幅圖可解。
- **音樂**:確認 XMIDI,寫 `xmi2mid.py` 轉 15 首標準 MIDI;note on/off 平衡、tempo 直通。
- **動畫結構**:FIGANI 每資源 = 一動畫,幀數自描述(同容器手法)+ uint32 offset 表 + 每幀 W,H。

**學到 / 驗證**
- 「正確解碼必輸出剛好 W×H」是破解壓縮格式的**決定性判定**,比肉眼猜更可靠。
- 攻略的 modify 表是極可靠的 oracle:offset 與數值逐筆吻合,連 HP/MP 為 2-byte 都靠它「HP HP」雙寫看出。
- 多種容器(.DAT / FIGANI / FDMUS)都用「自描述目錄(第一個 offset 回推項數)」+「3-byte 分隔標記」的一致手法。

**推翻的舊結論**
- 修正:單位/人物表的 HP、MP 欄為 **2-byte LE**,非 1-byte(第 1 輪 doc 03 已回填修正)。

**下一輪起點**:破 sprite 透明 RLE → 動畫逐幀輸出 PNG(使用者要求);FDTXT Big5 文本傾印。

## 第 3 輪 — 文本+字型全破、動畫像素 codec 受阻(2026-06-28)

**做了什麼**
- **文本格式全破**:FDTXT 是 **uint16 glyph 索引序列**(非 Big5)+ 控制碼(0xFF00+)+ 0xFFFF 結尾;
  1016 條字串、~58000 字、1824 個字模。
- **找到並破解自製字型**:`FDOTHER` 資源 #4 = 58368 = 1824 × 32 = **16×16 1bpp 字型**;
  渲染驗證:索引 0–35 = 數字英文,高位 = 漢字;最常用字模正是 `，`/`的`/`我`。
- **還原可讀中文**:`tools/decode_text.py` 把文本配字型畫出原版系統訊息(「要記錄戰況嗎？」等),完全可讀。
- **動畫 sprite codec 調查**:用視覺回饋(rulebook 64)確認背景 RLE 不適用(渲染為亂條);
  排除多種 escape/run 假設;確認 0xFE 是 escape 但連續長度僅 1–2、後接正常像素 → 非 bulk 透明。

**學到 / 驗證**
- 台灣老遊戲文本=「自製字型 + 內部索引」,不是 Big5——**先找字型再讀文本**是正確路徑。
- **視覺回饋是破壓縮格式最強的 oracle**:一張渲染圖立刻判生死,勝過反覆湊長度。
- **靜態猜 codec 要設上限**(rulebook 35/64):sprite 像素 codec 猜了 ~8 種假設仍不中,
  正解應改用反組譯 EXE 解碼迴圈當 oracle(`3C FE` byte 搜尋為假命中,需 Ghidra 級反組譯)。

**推翻的舊結論**
- 修正:`FDTXT 字串為 Big5` [假設] → **錯**,實為自製字型 glyph 索引(doc 01/99 對應修正)。
- 修正:`sprite 用背景 RLE 的透明變體` 的樂觀假設 → 已知是**另一套逐列格式**,codec 待反組譯。

**下一輪起點**:反組譯 FIGANI sprite 解碼器 → 動畫逐幀輸出(使用者要求);glyph→Unicode 對照表。

## 第 3 輪(續)— 反組譯定位 sprite 解碼器家族(2026-06-28)

**做了什麼**
- capstone(docker uv)反組譯 `FD2.EXE`。用 **`rep stosb`/`rep movsb` 叢集**當錨點,
  定位 sprite 解碼器家族於 **`0x4E000`–`0x4F800`**(動畫播放器呼叫的函式群)。
- 逐指令還原 `0x4EB52`(24×24 sprite 解碼器)的 **4 模式 RLE 文法**:
  高 2 bit=模式(色彩run/dither/literal/透明skip),低 6 bit=count−1,`[ebp+eax]` 調色 remap。
- 排除誤判:`0x12CB0`=純矩形 blit、`0x4EE31`=flood-fill、`3C FE`=call 位移假命中。

**學到 / 驗證**
- **`rep stosb`/`rep movsb` 是定位 RLE blitter 的最佳靜態錨點**(比 `cmp al,imm` 可靠;
  RLE 解壓器幾乎必用串指令做 run/literal)。
- 反組譯當 oracle 確實突破了純資料猜測的死局:從「codec 完全未知」→「家族定位 + 24×24 文法逐指令還原」。
- **byte 消耗對齊 ≠ 解碼正確**:24×24 文法套 FIGANI 能精確消耗位元組,但垂直相關顯示像素未對齊
  → FIGANI 用同家族**另一參數化變體**,不可因「長度對」就斷定 codec 正確(需視覺 + 相關雙重驗證)。

**推翻的舊結論**
- 修正:`0xFE 是 sprite 透明 escape`(第 3 輪前段假設)→ 實際透明是**控制 byte 高 2 bit = 11**,
  非單一 0xFE 值;sprite codec 是 2-bit 模式 RLE,非 escape-byte 制。

**下一輪起點**:反組譯 `0x4E000–0x4F800` 內 FIGANI 用的參數化變體(確認其模式/位元表),
完成逐幀 PNG 輸出;glyph→Unicode 對照表;FDSHAP tile stride 收尾。

## 第 3 輪(完結)— FIGANI 戰鬥動畫 codec 完全破解(2026-06-28)

**做了什麼**
- 反組譯參數化解碼器 **`0x4F43D`**(用 `[0x27B4]` 每列重設寬、`[0x27B6]` 列數)= FIGANI 用的解碼器。
- 解出 **13-byte 幀標頭**:realW 在 +9、realH 在 +11(呼叫端傳 frame+9 給解碼器);RLE 從 +13。
- 確立 4 模式 RLE(色彩run / dither陰影 / literal / 透明skip,count=低6bit+1)。
- `tools/decode_figani.py`:把 **264 個動畫、2118 幀**全部解出。視覺驗證:騎士(藍灰盔甲+紅披風)
  揮劍攻擊動畫含黃色斬擊特效,地面 dither 陰影正確 → **完全正確**。

**學到 / 驗證(關鍵方法論)**
- **垂直相關分析是定位真實列寬的利器**:橫條亂圖 → 對解出像素做「相鄰列相似度 vs 寬度」掃描,
  峰值(104, 0.728)立刻指出真實寬 ≈103 而非標頭首欄 167 → 回頭挖出 13-byte 幀標頭。
- **標頭首欄未必是解碼參數**:FIGANI 首欄 167 是「外框寬」,真正解碼寬 103 在 +9;
  解碼器讀的是呼叫端傳入的指標(frame+9),不是 frame+0。套規則前要追「解碼器實際讀哪個位址」。
- 反組譯 oracle + 視覺回饋 + 相關分析三者合擊,才攻下純資料靜態猜測攻不下的硬 codec。

**推翻的舊結論(已於 doc 06 修正/刪除)**
- `0xFE 為透明 escape` → 錯,透明是控制 byte 高 2 bit=11。
- `FIGANI 首欄 167 為解碼寬` → 錯,是外框寬;真實寬 103 在 +9。
- `24×24 文法直接套 FIGANI` → 需用參數化版 0x4F43D + 正確幀標頭。

**歷史快照（已過時）**：這段當時僅涵蓋資產容器／codec，不能延伸成「反組譯研究主體完成」。
後續已證實 native command、transient lifecycle、player selector、battle UI、campaign/town flow 仍有大量未閉合資料流；反組譯覆蓋現況以 `58`，系統、畫面與待辦分別以 `56`、`57`、`91` 為準。

## 第 4 輪 — 遊戲機制反組譯 + LE 重定位工具 + 劇情轉錄(2026-06-28)

**做了什麼**
- **機制文件 5 篇**:`09` 劇情/對話結構、`10` 敵我/狀態繪製、`11` 戰場 AI、`12` 音樂播放/場景切換、`13` 戰場選單。
- **AI（歷史假說，已撤回地址）**：舊筆記曾以 `0x15140` 描述 flood-fill／逐落點評分；canonical Docker recheck 已證該地址不是可確認的 AI entry。當時只足以稱 `0x14EF0` 為 candidate boundary；2026-08-09 的 IDA／Capstone 複核已補上其 `0x14ef0..0x15055` raw 尾端契約。這是當時的未閉合狀態；目前正常producer的E1 consumer已接，剩餘限制以`58`與`11`為準，不得由本歷史段落重開AI本體。
- **選單**:Enter/Space 確認、ESC 取消、方向鍵游標(`[0x3C57]`);`Get_EasyMagic` caller 已定位，但 magic raw/command schema 以 `unit+0x1a..+0x1d` 與 `0x1cff0` 為準，不能把 `+0x22..+0x24` 當 bitfield。
- **音樂**:`play_bgm`(0x26777),`[0x1A11]`=目前曲、track=−1 停、否則載 `FDMUS[track]`;32 處呼叫得場景→曲號。
- **LE 重定位 xref 工具**(`tools/le_xref.py`):解析 LE object/fixup 表,解開 DOS4GW 絕對位址重定位。
- **劇情轉錄**:序章+第2+第3章(本機 `extracted/story/`),含說話者標註與主線伏筆(悠妮公主身份、葛雷/卡蘿娜)。
- README 加知識庫總索引(分類可點選)。

**學到 / 驗證(方法論)**
- **DOS4GW LE 的 data xref 必須先套 fixup**:字串/表的絕對位址在檔內是重定位佔位,raw 搜不到;
  解析 LE fixup 表後才能「字串→參照處」追碼(scene→track 即靠此追到 play_bgm)。
- **洩漏的函式名是金礦**:`Get_EasyMagic` 直接點出法術選單函式位置。
- **攻略 patch ↔ 反組譯互證**:AI「傷害≤2 不打」正是攻略「不管防禦都攻擊」patch 改的判斷式。
- **劇情**:台灣老 SRPG 對話 = [控制碼][說話者肖像ID][『][字模索引][』],說話者直接對應角色。

**狀態**:7 大資產 + 5 大機制(AI/選單/音樂場景/敵我繪製/劇情結構)皆有反組譯文件。
知識庫 13 篇 + 工具 9 支。後續:劇情續轉錄、glyph→Unicode、陣營配色、重製垂直切片。

## 第 5 輪 — glyph→Unicode 全表 + 全劇情 + 控制碼 + 校對(2026-06-28)

**做了什麼**
- **glyph→Unicode 對照表 100%**(1824/1824):多模態逐批讀字模網格(`font_grid.py`)建表,全 35 章可解。
- **全劇情轉錄**:`decode_story_text.py --all` → 含說話者的 UTF-8(本機,1450 句)。
- **控制碼語意**(反組譯文本渲染器 0x16D00-0x17200):FFEF/EE/ED/EC=開對話框(4 變體,FFEF 從 DATO 載頭像)、
  FFFE=軟換行(每框 3 行)、FFFD=翻頁等鍵、FFFF=結束。**副產物:確認 DATO.DAT=人物頭像**。
- **校對**:用「解碼 + 上下文」揪出 14 處形近字模誤判修正(如 952 襲≠態、749 責≠青、1320 輩≠暈…)。

**學到 / 驗證(方法論)**
- **建 glyph 表最有效是「解碼自驗回圈」**:讀一批→解碼→讀解出的文字→形近誤判在上下文立刻露餡
  (襲/態、責/青、黨/嘗、費/曹…都是這樣抓到)。比孤立逐字判讀可靠得多。
- **字型是大雜燴**(使用者點出):數字/英文/漢字/標點/雙字元機器人代號(73/C2…)同擠一套;含**重複槽**
  (同字多 index)與形近字,判讀要靠上下文不能只看字形。
- **控制碼用 sign-extend int8 比較**(0xFFFE=-2…),渲染器靠負值分派;追文字渲染碼即解出語意。

**推翻的舊結論**
- 修正:DATO.DAT「立繪待定」→ 確認為**人物頭像**(0xFFEF 對話框由此載)。
- 修正 14 個字模誤判(形近/重複槽)。

## 第 6 輪 — worklist 收尾 + 音訊/擴充評估(2026-06-28)

**做了什麼**
- **FDSHAP 圖塊 + 地圖渲染**:發現圖塊 offset 表(原逐塊累積解會漂移→地圖亂掉);修正後全 33 戰場正確渲染。
- **FDFIELD 三段完整解析**:構成/控制/出場 → 全 33 圖 metadata(出場數/回合事件/寶箱/敵我roster/座標)。
- **FDICON.B24**=1680 個 24×24 **地圖單位 sprite**（four-mode RLE 含透明；後續 native scan 已撤回「非 FDSHAP bg-RLE codec」說法：FDSHAP 也共享 ABI，但 renderer branch 不同）；**TAI.DAT**=WxH 圖像(sprite-RLE)。
- **DATO 頭像**=4 嘴型幀(codec 0x4F716);**Unicode→glyph 反向表 + 編碼器**(round-trip 100%)。
- **remap LUT**=FDOTHER#3(LMI1,23 張),LUT 索引走全域狀態表(地形/場景 tint)。
- **音訊評估 + MT-32 實證**(doc 16):解釋 SoundFont、回答 MT32MPU.MDI(經 MPU-401 驅動外接 MT-32);
  munt+真 ROM **渲染全 15 首成 MT-32 WAV**(本機);版本切換架構(MT-32/SoundFont/FM)。
- **擴充劇本/玩法評估**(doc 17):資料層障礙已清,建議重製資料驅動+雙模式。
- **總覽 doc 15**:「1995 年怎麼做出炎龍騎士團2」。

**學到 / 驗證**
- **offset 表 vs 逐塊累積**:RLE tile 庫一定要用顯式 offset 表定位,否則單塊微小溢出會累積、整圖錯位
  (這就是「地圖全亂掉」的根因,使用者一眼看出)。
- **MT32MPU.MDI 的存在 = 音樂為 MT-32 而寫**:故 munt 渲染是還原原意,非腦補;沿用 dq3 既有 munt 流程即可。
- **勘誤**：`FD2.SAV` 並非未解的「強加密」；已由 `0x4dbd8/0x4dbb9` 關閉為 rolling-XOR/checksum，並有 `tools/fd2save.py` round-trip/tamper regression。重製使用自有存檔是產品選擇，不是工具鏈無法讀取原版。

**當時狀態**:已完成列舉的base codecs與exports（容器／圖／動畫／音樂／
文字字型／頭像／圖塊／地圖／圖示／remap／TAI），並抽出33地圖、
136頭像、15首音樂與劇情素材。這不代表所有mixed-resource entry、
caller binding或scene composition已閉合；反組譯覆蓋看 `58`，系統與畫面分別看 `56`／`57`。

## 第 7 輪 — 重製開工:Go/Ebiten + 本機執行檔(2026-06-28)

**做了什麼**
- 第一性原理可行性確認(doc 20):當時以 9 項資產／工具能力判定可以開始整合；
  後續實測已證明這不等於戰役、介面、戰後城鎮或存檔流程完成；覆蓋、系統、畫面與待辦依序看 `58`／`56`／`57`／`91`。
- Go/Ebiten 架構(doc 21)+ MVP 垂直切片(`remake/`:載序章地圖→hi-res 渲染→游標)。
- 技術驗證(doc 22):**本機桌面 Linux ELF(10.8MB)+ WASM(10.5MB)+ 資產管線**三項實證可建。
- 建立 M0–M6 重製工作分解（本機優先，網頁／手機打包延後）；這是當時的規劃基線，
  不是可玩完成度或原版等價宣稱。

**學到 / 驗證**
- 使用者校正方向:**本機執行檔優先**,不急著上網頁;WASM 只證「未來路沒被堵死」。
- 資產走中間格式(png+json)解耦:Python 工具側解 .DAT,引擎只認穩定中間檔。

## 第 8 輪 — 開場流程反組譯:標題/主選單/自動過場(2026-06-28)

**做了什麼**
- 建 `tools/disasm_le.py`(capstone 解 LE);反組譯開機→標題→主選單→開場對話→自動進戰場(doc 23)。
- 解圖驗證(規則 64):標題立繪 5 幀(FDOTHER #0x45–0x49)+ FLAME DRAGON logo(#7 sub0)。

**推翻的舊結論**(回填修正)
- ❌ `main = 0x10000` → ✅ **真 main = 0x25bf4**(0x10000 是場景載入子程式)。
- ❌ 標題「旋轉動畫」→ ✅ 實為**角色立繪垂直捲動**(解圖兩面證實)。
- ❌ logo = FDOTHER #0x65 → ✅ #0x65 是調色盤,logo 在 **#7 容器 sub0**。

## 第 9 輪 — call-graph 工具 + 釘死呼叫鏈 + [0x53ecc] 狀態機(2026-06-28)

**做了什麼**
- 建 `tools/callgraph_le.py`(遞迴可達反組譯:reach/callers/rpath/funcof/jtab)。
- 當時以 call graph 找到 `0x10010` 的兩個真 caller 並修復 data 段跳表；2026-07-29 進一步展開
  `0x25ec8..0x26151` 後撤回「一般進戰場鏈」說法：`0x10010` 只在第三主選單分支與戰內重載分支
  恢復 FD2.SAV current-runtime snapshot，一般 pre-handler 返回 main 後直接進 `0x117e7`。
- 追完 `[0x53ecc]` 戰役迴圈狀態機(doc 24)。

**學到 / 推翻**
- **線性 sweep 會給偽 caller**(le_xref 回報 0、agent 猜 0x1b051/0x26f30 皆偽);**遞迴可達**才可靠。
- ❌「相位機串接 cutscene→戰場」仍不成立；但當時進一步稱「同 `0x25ebb` 內線性串接到
  `0x10010`」也已於 2026-07-29 撤回。正確邊界是 pre-handler 從 `0x25ebb` 返回 main，
  main 再呼叫 `0x117e7`；`[0x53ecc]` 是戰後/事件碼而非進場中介。
- **方法**:定位具名全域變數所有讀寫,用 LE fixup refs + opcode 模式(繞線性漂移)。

## 第 10 輪 — 戰場事件系統 + 逐關 handler 資料化(2026-06-28)

**做了什麼**
- 第三張章節跳表 `0x51b19`(戰場事件 handler)+ 事件原語(doc 25)。
- 逐關挖完 18 個特殊 handler(doc 26)→ 機器可讀 `docs/data/battle_events.json`(30 章條件→動作);
  建 `tools/event_handler_dump.py`。

**學到 / 推翻**
- ❌「事件腳本解譯器(byte-code)」→ ✅ **每章一個編進 EXE 的 C handler 函式**(非 VM);用詞已正名。
- 11章使用 default raw 三值結果規則，18個特殊章另查單位／回合設碼；
  舊「default=殲滅即勝」及把指定 raw bit 一律命名成存活／死亡的說法，
  已由 `0x205be` 與 `0x3453e` 直接指令勘誤取代。
- **對重製**:事件可資料驅動(battle_events.json + roster + FDTXT),引擎不為任一關寫死分支。

**下一輪起點**:挖各章動作函式(0x33499…)+ 單位 idx→角色對應,補完事件語意;接 ScenarioRunner。

## 第 11 輪 — remake M1 開工:單位資料模型 + 渲染(2026-06-28)

**做了什麼**
- `tools/export_units.py`:序章單位 = FDFIELD roster(camp/cls/lv)+ 出場座標(positions)+ EXE 數值表(hp/ap/dp/mv)→ `remake/assets/map0_units.json`(本機)。
- `remake/internal/battle/model.go`:Unit/Camp/State + `Load()`；`Alive()`=HP>0、`Acted` 是 remake 的 normalized projection。這兩者不可宣稱為原版 `byte+5` 的全域對映；native byte+5 目前只在已確認的 caller 邊界以 raw mask 讀取。
- `cmd/fd2/main.go`:在 MVP 地圖上畫單位層(陣營色塊 我方藍/友綠/敵紅 + HP bar),游標選中顯示 Lv/HP/AP/DP/MV + 回合與各陣營存活數。
- headless test(`model_test.go`)驗證序章:own=2 ally=4 enemy=24,我方落部署格、HP/MV 合理、UnitAt 排除陣亡 → 全綠。桌面 ELF 建置成功。

**學到 / 驗證**
- 三方資料(FDFIELD roster + 出場座標 + EXE 數值)組成引擎 units.json,引擎只認 JSON → 資料/引擎解耦(doc 30 管線)。
- Ebiten 邏輯層(battle package 不 import ebiten)可純 headless 測試,不需 display → M1-8 回歸基礎。

**下一輪起點**:M1-3 flood-fill 移動範圍 + M1-4 戰場選單(移動/攻擊/待機)。

## 第 12 輪 — 地圖單位 sprite 找對來源:FDICON Q 版小人(2026-06-28)

**做了什麼**
- 使用者提供**原版實機截圖**(`real_pic/`,rulebook 64 oracle)→ 確認地圖單位是 **Q 版大頭小人**,非 FIGANI 全身。
- `0x4EB52`(24×24 場景單位解碼器)caller 是間接呼叫追不到 → 改 screenshot oracle 反推資源。
- 發現 **`FDICON.B24` = 1680 個 24×24 地圖單位 sprite**(header tw/th/count+offset 表;tile 用 **sprite 4-mode RLE 含透明**);`tools/decode_fdicon.py` 解出全部 Q 版小人。每角色組=12(4方向×3待機幀,手擺)。
- 引擎改用 FDICON:24×24 直貼格 + 待機循環 + 陣營腳標 → `31`。

**學到 / 推翻**
- **解碼失敗 ≠ 斷言錯**:FDICON「1680×24×24」斷言是對的,錯在用 bg-RLE 解(該用 sprite-RLE)→ 換對 codec 即解;一度想把對的斷言改「待確認」,被使用者打斷糾回。
- **FDICON=地圖單位 / FIGANI=戰鬥全身**,兩套分工別混(之前誤用 FIGANI 當地圖單位)。
- 原版實機截圖是最強 oracle:一眼定位「該找什麼 sprite」。

**★ sprite / 動畫機制定論(反組譯，2026-07-26 勘誤)**:① 地圖 sprite index = **`unit+2`×12 + 方向×3 + 幀**(0x128e0–0x12932,經 0x11019)。② 已驗證的玩家 roster 可讓角色 id／肖像／map visual id 相等，但不能把它寫成所有 runtime 的欄位 alias。③ **戰鬥 FIGANI = `unit+7`×3**(0x2884c)，與地圖 selector 是不同 raw field；全螢幕演出主函式 0x28a6c **無 runtime 縮放**(守小攻大是 FIGANI 美術本身尺寸,景深燒進素材);完整繪圖機制見 doc 35。

> 教訓(高代價,踩過兩次):撞到「打破規律的特例」先查權威資料(攻略 memory.md / 實機)再下手,**別憑自己未證實的角色↔檔名對應建例外映射表** → 循環論證一錯到底。sprite 誤判 DATO_067=凱拉斯、FIGANI 亂建 figaniID,都是同一坑。

## 經驗補記 — 為什麼單一反組譯函式看不到完整 UI 排版（2026-07-28）

### 問題

單看一個 renderer 或 menu handler，往往只能看到局部座標與一次 blit，
無法直接看出玩家最後看到的完整畫面。這不代表排版不能從反組譯還原，
而是 UI 的責任被拆散在多個 caller、callee、資源、global state 與前一張
framebuffer 之間。

典型畫面可能由以下來源共同組成：

- caller：決定場景流程、資源編號與 draw order；
- renderer callee：決定 destination、stride、clip、transparent index；
- `FDOTHER`／`FDICON`／`FDTXT`／`DATO`：保存背景、框、圖示、文字與肖像；
- global variables：保存 camera、cursor、selected item、animation phase；
- palette function：決定 indexed pixels 最終顯示的色彩；
- input loop：決定 opening、steady、confirm、closing 的時序；
- caller 進入前的 framebuffer：可能已經包含不能在目前函式內重新找到的背景。

因此原版通常不會有一個容易辨認的
`draw_shop(background, shopkeeper, menu)`。更常見的是：

```text
載入背景
→ 貼店員
→ 建立對話框
→ 畫金額
→ 開啟服務圖示選單
→ 備份局部 framebuffer
→ 滑入商品框
→ 畫商品／游標
→ restore／close／return
```

這條鏈可能散落在五至十個函式。只破解其中一個文字函式或一個 frame
decoder，不能宣稱商店畫面已還原。

### 反組譯可以精確回答的部分

追完整 caller/callee dataflow 後，靜態反組譯通常能固定：

- framebuffer 是 320×200、320-stride 或 640-stride work surface；
- 元件 destination byte offset，以及換算後的 `(x,y)`；
- width、height、clip rectangle、transparent index；
- archive 名稱、resource index、nested entry index；
- glyph foreground／shadow／background、line advance；
- menu row／column、cursor wrap、confirm／cancel；
- opening／closing 每幀 displacement；
- copy、blit、present、palette、wait 的實際順序。

所以「幾何排版無法反組譯」是錯誤結論。真正的限制是：需要追完整
composition graph，不能只看一個函式。

### 反組譯單獨不容易回答的部分

只看 assembly 通常不足以可靠命名：

- 某個畫面究竟是武器店、道具店、酒店或秘密商店；
- 一張透明 sprite 在成品畫面中的美術角色；
- callee 進入前 framebuffer 已留下什麼；
- 哪一組 DAC palette 才是該畫面的正確顏色；
- 某一幀是短暫 transition，還是可接受輸入的 steady state；
- 畫面只在何種 chapter、save flag、party state 下出現。

這些問題需要 DOSBox screenshot／video、攻略畫面、動態輸入 trace 或原始
save state 交叉驗證。網路圖片只能協助辨認畫面結構，不能取代本機可重播
oracle。

### 已經踩過的錯誤

1. **把 raw decode 當 runtime screenshot**
   `docs/figures/title.png` 是錯 palette 的 resource decode，不是目前 remake
   title 畫面；`dialogue.png` 也是文字／字型研究圖，不是對話 runtime。

2. **把局部 primitive 當完整 scene**
   解出 dialogue frame、shop transaction 或 selector input，不等於背景、
   店員、金額、圖示、開關動畫都已組合。

3. **用 generic UI 補掉未知排版**
   `drawCampaignUI` 的現代半透明框能讓流程可玩，但不能成為原版 town、
   shop、preparation、load 或 ending 的完成證據。

4. **忽略 stack push 期間 ESP 位移**
   caller 連續 `push [esp+N]` 時，每次 push 都會改變後續 `[esp+N]` 指向。
   不做 symbolic stack trace，容易把 restore buffer、FIGANI、TAI、unit slot
   對錯。

5. **只比兩張不同狀態的截圖**
   ch01 原版／remake 若 roster、camera、cursor、animation tick 不同，只能
   證明素材與局部 compositor，不能用來宣稱整幀 parity。

### 後續 UI 還原的必要流程

每個完整操作界面都應採三層 evidence gate：

1. **E0：反組譯完整 composition contract**
   - 從 scene entry 追到所有 draw/present/input/close calls；
   - 保存 address、resource、buffer、offset、stride、order、timing；
   - 未證實的美術／玩法名稱保留 raw name。

2. **E1：原始素材重建 indexed frame**
   - 使用玩家自備 archive；
   - 在 320×200 indexed buffer 中按原順序合成；
   - 不用現代 panel、手調座標或 PNG convenience layer 補缺口；
   - 不支援的 branch 必須 fail-closed。

3. **E2：同狀態 DOSBox 對照**
   - 固定同一 `FD2.SAV`、chapter、party、cursor、camera、menu selection；
   - 固定 opening／steady／closing 的相同 presentation tick；
   - 同時輸出 original、remake、pixel diff；
   - geometry、palette、layer、input lifecycle 均通過後才標記 visual parity。

### 工程單位必須從 primitive 改為完整畫面 owner

後續不應只排「再解一個 frame／一個文字函式」。應以玩家可見 scene 為單位：

```text
town entry
  ├─ service background
  ├─ shopkeeper / church character
  ├─ dialogue frame + text
  ├─ gold / status
  ├─ service selector
  ├─ child buy/sell/revive/class panel
  └─ close / return / campaign transition
```

每個 owner 都要擁有完整 resource binding、indexed buffers、state machine、
input trace 與 screenshot oracle。只有這樣，address-level RE 才會真正收斂成
原版畫面，而不是持續累積彼此沒有組起來的 primitives。

### 本次結論

反組譯能提供原版排版的精確資料，但「全貌」存在於完整 composition graph，
不在單一函式中。FD2 現階段 asset/codec 進度高，操作界面視覺 parity 仍低，
根因正是過去以 primitive 為工作單位，且過早用 generic UI 完成可玩流程。
後續優先級應改成 town／shop／preparation／load 等完整 scene owner，再以
DOSBox 同狀態 pixel diff 驗收。

## 經驗補記 — 商店 stable scene 證明 composition graph 方法有效（2026-07-28）

以 `0x2e341` 為 owner 往下追，而不是再孤立解一個 codec，這次一次串起了
FDOTHER variant background、`0x1956b` 的 dialogue grid／DATO portrait、
entry1 decoration、八位 gold 與 FDTXT greeting，產生第一張原版資源商店
stable fixture。過程也抓到兩個若只看檔頭很容易犯的錯：

1. 同一 scene resource 內有四模式 background、高位 run opaque cell，以及
   `0x2d669` 透過 `0x4e9e4` 消費的另一類 entries，不能用一個 decoder
   強解全部。
2. nested `LLLLLL` 與 scene-flavoured `LMI1` 都有 terminal boundary，但
   count/offset layout 不同；必須用真實三個 variants 做回歸，不能從單一
   resource 外推。

因此 asset parser 只 decode 已由 caller 證實的 background／decoration，
其餘 service entries 保留 raw；這比「解得出一張看似合理的圖」更重要。
目前畫面仍只是 opening 完成後的 stable target，不含服務圖示、輸入 pulse、
商品／接收者子面板與 closing，所以 production owner 仍保持 partial。

後續重讀 `0x2d669/0x2d9fe` 又證明「保留 raw」是正確決策：entries3–10
不是前面 entry1 的 high-run opaque codec，而是 `0x4e9e4` 消費的
width×height literal transparent cells。四個 normal/selected pairs 以
`[-39,-13,13,39]` 做四步spread，selected cell再依phase/2切換。若上一輪
為了讓所有entries都「成功decode」而放寬high-run parser，這批圖示很可能
產生看似有圖、實際錯位或截斷的假成果。mixed scene的正確策略是讓direct
caller決定每個entry的codec，而不是讓container magic決定。

同樣原則也適用服務名稱：四個icon的圖案只適合提出假說，真正定名必須追
到mutation writer。這次 `0x2f0b0` 的insert／optional equip／gold debit、
`0x2f642` 的3⁄4 credit／remove、`0x2f883` 的compatibility／same-type
replacement，以及`0x2f8ea`的source→destination writer，才足以把四項
定名為purchase／sell／equip／transfer。這也修正了文件兩個相反問題：
早期一處把normalized transaction標成完整✅，另一處卻在writer已充分時仍
  把四項全寫成unknown。證據門檻不是永遠保守，而是在證據到齊時精確升級。

## 經驗補記 — 畫面 parity 必須保存 framebuffer ownership（2026-07-28）

購買流程再次說明「文字 index 與座標正確」仍不足以代表 GUI 正確。
`0x2f0b0` 的金額不足分支不是呼叫 `0x1956b` 建立新對話。更完整的
instruction-order audit 也推翻本段早先「保留 steady Yes/No cells」的說法：
`0x2f2a9` 先呼叫 `0x197e5` 呈現四個 choice-closing frames，接著
`0x19913..0x1994c` 恢復保存的310×86 question region；`0x2f2d3` 才把
FDTXT 504/438 寫到 literal VGA `0xac44c`（`(12,157)`），並由
`0x16c57(1)`畫等待標記。若只抽象成 `showMessage(text)`，截圖仍可能看似
合理，卻會遺失上一層 prompt 的restore ownership、等待動畫與返回狀態。

因此 scene RE 除了 resource、offset、draw order，還必須記錄：

1. 目前 framebuffer 是 caller-owned stable target 還是新配置 buffer；
2. callee 是 replace、overlay、append、restore 哪一種 mutation；
3. input owner 與 pulse 是否仍存活（本例在不足金文字出現前已關閉）；
4. 錯誤回饋後回到同一 prompt，還是重新進入 scene owner。

本輪把 insufficient-gold API 從 generic fresh overlay 移出，並在再次
核對 caller 後收緊為只能使用與`0x197e5` restore後question相同的pixels；
目前ch02 screenshot path是deterministic recomposition，尚未成為generic
saved-buffer restore owner。這種 fail-closed contract 比共用一個方便的
message renderer 更能防止後續 remake 悄悄偏離原版 lifecycle。

## 經驗補記 — 同一個「裝備分類」可能有兩種原版 key（2026-07-28）

equipment recipient preview 的 `0x2efb7` 以 item row type `<=0x14` 判斷
要保留哪一類已裝備加成；真正寫回的 `0x1c142` 則直接以 item ID
`<0x80`／`>=0x80` 決定卸下哪個 raw slot。兩者在原版資料上對應，但不能
因此把 type、ID 與 raw flag 合併成一個抽象欄位。重製時應分別保存：

1. item type：相容性與 preview 數值路徑；
2. item ID：`0x1c142` replacement category；
3. raw inventory flag：實際 equipped writer（0或`0x40`）。

這次 production integration 也暴露 compact `Equipped[]` 只更新自己、
未同步八格 raw flags 的舊缺口。現在 writer 完成後會按含內部空洞的
`InventorySlots` 映射回 raw slot；否則當前畫面看似正確，存檔、下一次
preview與native item command卻會讀到另一套裝備狀態。

## 經驗補記 — 高階語意 parity 不等於 stale byte parity（2026-07-28）

`0x1b8e7` 移除物品時會把後續raw pairs左移，只把最後一格flag寫成
`0x80`；最後item byte保留stale值。重製的`Unit.InventorySlots`同時也是
後續`AddInventoryItem`尋找空位的高階adapter，因此目前把該ignored tail
item canonicalize成`0xff`。這能保持「下一次插入找到尾格」的語意，但不能
被文件寫成整個0x50 record byte-identical。

遇到這類狀態必須分開聲明：

1. writer順序／active flag／後續玩法語意是否一致；
2. adapter是否正規化原版不再讀取的stale bytes；
3. 是否真的能以FD2.SAV或memory dump逐byte比較。

本輪因此把sell標成production gameplay parity，但明確保留save-byte E2/E3
缺口，避免用綠色GUI regression支持一個它根本沒有覆蓋的斷言。

### 同名操作不一定共用 widget

購買後的「選收件者」依 item type 分成兩個 owner。消耗品走兩欄六人
roster；裝備先過 compatibility，再用三列 AP/DP/HIT/EV 現值→候選值面板。
只看兩者都回傳 actor index，會錯把裝備比較資訊刪掉。實作因此先讓
consumable compositor 拒絕 equipment type，再獨立閉合
`0x2e8cf→0x2ebe0/0x2efb7`。這是資料 discriminator 應進入 renderer API、
不能只留在上層選單文字的實例。

### 入口叫「裝備」，不代表它是商店商品面板

重讀 `0x2f883` 後，原先最自然的實作方向被推翻。這個入口只負責角色
roster；真正 child 是 `0x1bffe`，它先由 `0x17e0b` 保存目前商店 VGA，
再開啟戰場物品指令也共用的完整 item/status panel。它沒有商品價格、購買
收件者、金錢 writer或成功對話。相容 item 由 `0x1c1c3` 判斷，成功才走
`0x1c142→0x1b750` 並在同一 panel 原地重畫；不相容只回 selector。

### Fixture 通過不代表 raw projection 足以做 E2（2026-07-28）

裝備收件者 indexed fixture 曾讓三列面板看似完成，但真正DOSBox對照立刻暴露
兩種不同錯誤。第一，direct shop node沒有persistent party；若硬塞三個姓名，
只會掩蓋JOIN chronology與raw inventory/class來源。新的screenshot bridge只從
compile通過且同時帶`PartyScenario+PartyOrder`的LOADCH binding建立typed
projection；順序來自該binding所記錄的`PartyOrder`，hook本身不重新證明JOIN
來源位址。這只是screenshot bridge，不代表正常campaign persistence或
native FD2.SAV載入；缺任一必要provenance即拒絕。

第二，scenario只保存derived HIT/EV卻漏了`0x2efb7`真正讀的DX projection
`+0x3e`，fixture仍能用零值產生「合理」數字。ch01的2/2/1/2是以可見
HIT/EV和已知equipment rows交叉約束出的projection，不是獨立raw dump。
加入明確DX projection後，E2又抓出HIT/EV欄位相對
AP/DP右移3px及arrow Y anchor錯誤。最後以FDICON idle cycle1對齊原版，
整幀AE才從數千降為0。經驗是：renderer fixture、typed party provenance、
global animation phase與DOSBox同狀態像素缺一不可；任何單項綠燈都不能宣稱
操作界面已還原。

### JOIN chronology 不是 persistent record 本身（2026-07-28）

recipient E2的screenshot bootstrap也暴露正常campaign的另一層缺口：JOIN beat
只保存membership/order，而首次LOADCH只建cutscene/battle unit；之後
`sync_party`遇到native identity時卻只接受既有typed roster match，因此可能
把開局全隊skip。修正位置應在JOIN後的LOADCH record materialization，而不是
放寬`sync_party`接受未知identity，也不是把screenshot hook接進production。
新bridge只補缺少record、永不覆蓋既有進度，且direct/debug replay不建立
persistence。這個案例說明：已證實的順序、typed record生命週期、以及畫面E2
是三個不同證據層；文件若把其中一個寫成另外兩個，會直接導致錯誤引擎設計。

這輪的工程教訓是把共享範圍切在「資料 mutation／原版 panel primitive」，
而不是切在玩家語彙。購買後可選擇裝備與城鎮獨立裝備都會改 equipped flag，
所以可共用 strict mutation；但兩者的 scene owner、input、framebuffer restore
與 feedback 完全不同。若只看最後副作用，會做出操作可用但畫面資訊架構錯誤
的重製。

另一個容易被高階模型遮蔽的缺口是 slot index。`0x1b9de` 顯示的是跳過
raw ignored cells 後的 compact order，`0x1c142` 最終仍寫 raw cell；remake
的 `EquipItem` 卻接受 compact ordinal。因此 production adapter 必須先證明
raw occupied order、`Inventory`與`Equipped`三者一致，並保留 ignored cell
可能存在的 stale item byte。這個 gate 比「item ID 看起來相同」更重要，
因為 duplicate ID與internal hole都會讓錯誤 index 靜默裝到另一格。

### 共用 callee 不屬於單一設施；目的名冊也不一定排除來源

`0x2f8ea` 的高階語意確實是物品轉交，但 direct caller 有兩個：
shop `0x2e341` service3與church `0x3072f` raw1。只查其中一個 caller
會做出「現有另一設施接錯」的錯誤勘誤；本輪用兩個 direct caller
交叉裁決，保留共享 service body，僅替各自接正確 framebuffer owner。

另一個高階便利假設是「目的角色不應是來源本人」。原版第二次
`0x2e6b8` 仍使用完整 party，沒有刪除 source。若來源未滿八格，選自己會先
`0x1b8e7` compact-remove，再由`0x1bb8c`插入第一個空格，因此 item移到尾端
且清除 equipped bit；滿八格則在 remove前就顯示 FDTXT506。這個分支提醒我們：
合理的現代 UX 篩選不能取代 raw candidate evidence，即使看起來只是多餘選項。

### 攻略說明了按鍵，EXE 才說明 gate

秘密商店攻略的「酒店前 Shift+F1」容易被重製成先看傳聞、寫永久 flag、再顯示
第六個選項。這能玩，但不是原版 input topology。`0x2cde0..0x2cef7`證明攻略中
的「前」其實是目前五項 selection；每章 town record 另存一個 BIOS function-key
scan，只有兩者同時命中才把當次 selection 改為 hidden value5。

因此外部資料適合提供可搜尋的按鍵提示與商品表，不能單獨決定 runtime state
model。反組譯閉合後，schema應保存最小 raw contract
`{selection,scan_code,to}`；persistent unlock flag只能標成擴充功能。

這輪也再次證明生成器不是天然權威。現有`gen_campaign.py`雖能重產完整JSON，
但它的拓撲落後於後續人工 handler integration；直接執行會大幅刪回新版節點。
任何生成器都要先比較semantic diff，不能因命令成功與JSON合法就覆蓋較新的
campaign。

### GUI 排版不是一張背景；selector lifecycle 也不是一次 transition

城鎮 hub 這輪把「已解出背景」與「完整 scene owner」的差異具體化。
town record byte0只決定FDOTHER#11/#61/#62之一；真正畫面還需要#10 label、
FDTXT `0x1ef+selection`、FDICON pulse、三個variant各六組X/Y座標，以及
stride456 work scene裁成312×192後貼到VGA `(4,4)`。若只把背景換進現代
清單，資源雖然正確，layout ownership仍完全錯。

同一段input loop也推翻本輪稍早的API：secret chord命中後只把selection寫5，
沒有立刻呼叫shop。hub先以第六組座標／文字重畫，玩家後續confirm才dispatch。
因此`detect→reveal→redraw→confirm→dispatch`必須拆開測；把它壓成
`AdvanceSecret()`會讓最終目的地正確，卻把玩家實際看見的中間狀態刪掉。

最後，第一次Xvfb命令表面exit 0但截圖timestamp未變，畫面仍是舊檔。改用自行
啟動Xvfb、等待socket、`exec` binary的可觀察runner後才真的產生新圖。GUI artifact
驗證不能只信process exit：還要查mtime、尺寸並人工看圖，否則README很容易長期
展示與source不一致的舊成果。

### 72×72 快照原語不等於 native renderer 完成

原版 `0x175a9/0x17643` 的 `0x1440` bytes 很容易被誤讀成「只要在 Ebiten
畫面上貼四張 cell 就完成」。本輪先把它收斂成兩個明確的 indexed API：
`CaptureActionOverlaySnapshot` 與 `RestoreActionOverlaySnapshot`。矩形左上角由
caller 明確傳入，任何尺寸／stride／邊界錯誤都在寫入前拒絕；測試同時驗證
非均質資料的完整還原與外部像素不被碰觸。這保留了原始位元組契約，又不把
cursor、camera 或 relative offset 猜成 private-buffer owner。

目前重製的 Ebiten 路徑每幀重畫整幅場景，所以視覺上不會累積殘影，但這不是
原版 backup/restore 的證據。只有在補齊 native owner、同狀態 DOSBox 畫面與
正式 renderer 的消費端後，才可把 UI-03 提升；單純把 `[x]` 測試項寫進工作清單
會重新造成「資料原語＝玩家畫面」的記憶混亂。

### 反向追查快照 owner 要看逐列來源，而不是只看配置大小

第一次只看到 `malloc(0x1440)`，只能證實 72×72 大小，不能知道它對應畫面哪一塊。
IDA／Capstone 逐行重核後才閉合真正 owner：來源從 `0x8088` framebuffer base
開始，欄列分別是可視游標各減一個 24-pixel cell，來源每列走 `0x1c8`、快照每列
走 `0x48`，共 `0x48` 列。這讓重製端可以安全提供
`ActionOverlaySnapshotOrigin`，但仍不應把 flat byte address 直接當成 Ebiten
像素座標；正式 renderer 還要證明誰持有 snapshot 以及何時 restore。
