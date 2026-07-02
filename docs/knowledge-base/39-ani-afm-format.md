# 39 — ANI.DAT 完整 AFM 格式解碼(開場過場動畫)

> 目標:破解 `ANI.DAT`(2.44MB,10 個資源)——它驅動doc23 §2.4 記載的「32.3 秒開場多幕過場」
> (漢堂 logo 後、標題「2」縮放進場前,守護者→索爾→拔劍屠龍→騎馬夜行→浮空城→金鎖→合照→惡魔臉)。
> 方法:**靜態反組譯 VM 指令集(規則 62)+ Python 重寫解碼器 + 逐幀輸出 PNG 用實拍幀當 oracle 視覺比對(規則 64)**,
> 全程未用 DOSBox debugger。工具:`tools/le_xref.py`、`tools/disasm_le.py`、新增 `tools/decode_ani.py`。

---

## 0. 一句話結論

**AFM 不是逐幀點陣圖動畫,是一個 10-opcode 的「增量繪圖 VM」**:每幀是一段 bytecode script,
對上一幀遺留在 framebuffer / palette 的內容疊加操作(填色 / RLE /局部貼圖 / 局部調色盤更新),
不整幀清空重畫。這是 1993 年典型的「差分壓縮動畫」設計——用小體積 script 驅動 320×200
全螢幕輸出(如 resource #0 僅 96 幀 ~1MB,遠小於 96×64000=6MB 的全幀陣列)。

九個資源逐一解碼並用**已知 dosbox 實拍分鏡序列**(`extracted/title_re/dosbox_seq/`,doc23 §2.4③)
肉眼比對,**全數命中**,見 §4 對照表。

---

## 1. 容器層(已知,doc06 記錄過)

`ANI.DAT` 是標準 `LLLLLL` 容器(見 `tools/unpack_dat.py`),10 個目錄項,第 10 項長度 0(終止符),
故實際 **9 個有效資源**:

| idx | 長度(bytes) |
|---|---|
| 0 | 1,002,800 |
| 1 | 635,952 |
| 2 | 97,726 |
| 3 | 35,566 |
| 4 | 36,113 |
| 5 | 411,039 |
| 6 | 43,553 |
| 7 | 137,859 |
| 8 | 36,893 |

每個資源本身就是一份**完整的 AFM v1.00 檔案**(資源 #0 開頭即版權橫幅,見 doc06)。

---

## 2. 資源(AFM 檔)標頭 [驗,自洽位元組核算]

```
+0x00  80 bytes   版權橫幅 "AFM - Animation File Manager Version 1.00 Copyright (C) 1993 Lo Yuan Tsung 09/29"
+0x50  1 byte      0x1A(SUB/EOF 標記,DOS 文字檔慣例)
+0x51  0x50 bytes  標題欄位(space-padded ASCII;未設定時內容固定為 ".Empty Title." )
+0xA1  1 byte      0x00(標題結尾 null)
+0xA2  3 bytes     未知(版本/旗標?未逐位解出)
+0xA5  uint16 LE   frameCount(反組譯證實:0x020421 讀 [buf+0xA5] 當幀迴圈上界)
+0xA7  uint16 LE   螢幕寬度(9 個樣本恆為 320)
+0xA9  uint16 LE   螢幕高度(9 個樣本恆為 200)
+0xAB  2 bytes     未知
+0xAD  起          frameCount 個「幀記錄」(見 §3)
```

**驗證方式**:對 resource #0(96 幀)逐筆累加 `8 + compSize` 剛好等於
`資源總長 - 0xAD`(1,002,800 − 173 = 1,002,627,實測完全吻合),對其餘 8 個資源同樣核算通過——
證明此標頭 + 幀記錄框架的位元組切分**完全正確**,不是巧合。

---

## 3. 每幀記錄 + AFM Script [驗]

```
+0 uint16 LE  compSize   本幀 script 的位元組數(fread 用)
+2 uint16 LE  cmdCount   本幀要執行的 VM 指令數
+4 uint16 LE  (保留,9 個資源樣本恆為 0,用途未明)
+6 uint16 LE  (保留,同上)
+8 ... compSize bytes 的 script(bytecode,見 §4 VM)
```

少數幀 `compSize=0, cmdCount=0`(如 resource #0 的 frame 1–9):**合法的「空幀」**——
不執行任何繪圖指令,畫面沿用上一幀內容原樣顯示,純粹用來拉長該畫面的停留時間(逐幀播放但無變化)。

---

## 4. VM 指令集(10 opcode,反組譯完整還原)[驗]

### 4.1 反組譯位址

| linear | 角色 |
|---|---|
| `0x020421` | **AFM 播放器主函式**:`fopen("ANI.DAT","rb")` → seek 目錄項 → 讀 173-byte 標頭 → 逐幀迴圈:讀 8-byte 幀記錄→讀 script→呼叫 VM→`blit`(`0x375b2`)→ `kbhit`(`0x10620`)可跳過。3 個參數:`(index, delay, skippable)`。 |
| `0x36c9e` | **VM 指令派發器**:`lodsb` 讀 1 byte opcode,`shl 2` 查表 `[0x5276a + op*4]`,`call` handler,重複 `cmdCount` 次。 |
| `0x36c7d` | VM 狀態初始化:設定 `[0x52760]`=畫面寬度(word)、`[0x52762]`=framebuffer 指標(dword)、`[0x52766]`=palette 暫存指標(dword)。**唯一呼叫端 `0x02048a`,傳入 `width=0xfa00(64000)、framebuffer=0xA0000(VGA mode13h 顯示記憶體本身!)、palette=malloc(768)`** ——證實**畫面操作是直接寫 VGA 顯示記憶體,無雙緩衝**。 |
| `0x5276a` | opcode 跳表(fixup 解出 10 個有效函式指標,第 11 項起變成別的資料段內容「Stack...」字串,故 **opcode 只有 0–9**)。 |

### 4.2 十個 opcode 語意

| op | 目標緩衝 | 語意 | 消耗位元組 |
|---|---|---|---|
| 0 | palette(768B) | 整包填滿同一 byte 值 | 1 |
| 1 | palette | 整包字面載入(768 bytes 原樣拷貝) | 768 |
| 2 | palette | RLE 解壓(2-mode:控制byte高2bit==11→run,否則→literal),填滿 768 bytes | 變動 |
| 3 | palette | 局部貼補:1 byte 記錄數 N,N×(colorIdx, count, RGB×count),offset=idx×3、length=count×3 | 變動 |
| 4 | framebuffer(64000B=320×200) | 整片填滿同一 byte 值(= 純色清屏) | 1 |
| 5 | framebuffer | 整片字面載入(64000 bytes 原樣拷貝,未壓縮全幀) | 64000 |
| 6 | framebuffer | RLE 解壓(同 op2 的 2-mode),填滿 64000 bytes——**全螢幕主力解碼器** | 變動 |
| 7 | framebuffer | N×(offset16, value8) 單點繪製(sparse pixel plot) | 2+3N |
| 8 | framebuffer | N×(offset16, length8, value8) 區段填色(run-fill,如純色色塊) | 2+4N |
| 9 | framebuffer | N×(offset16, length8, rawBytes...) 區段貼圖(sparse block copy / 局部貼花) | 2+N×(3+length) |

RLE 文法(op2/op6 共用,**與 FIGANI 的 4-mode 文法不同**,這裡只有 2-mode):
```
高2bit==11 (0xC0)  →  count=低6bit;讀1byte值;寫 count 個該值(run)
其餘(全byte值)      →  literal:直接寫該 byte(不遮罩)
```

**設計意義**:opcode 0-3 操作 palette(調色盤淡入/局部變色如霓虹閃爍),opcode 4-9 操作
framebuffer(整屏填色/RLE背景/貼圖角色/局部特效點),兩者混合、跨幀疊加,即可用極小 script
合成複雜過場(如角色逐漸淡入、鏡頭緩慢位移、火光閃爍)。

---

## 5. 解碼器與驗證結果

`tools/decode_ani.py`(純 Python,不依賴反組譯工具,可獨立跑):
```
python3 tools/decode_ani.py info   <ANI_NNN.bin>              # 印標頭/幀表摘要
python3 tools/decode_ani.py frames <ANI_NNN.bin> <out目錄>    # 逐幀存 PNG(320x200)
```

**全 9 個資源、合計 289 幀,全部解碼成功、無 VM 例外**(即所有 opcode 分支與運算元切分
在完整 2.44MB 資料流上自洽,非單幀巧合)。已存 PNG 於本機 `extracted/re_tmp/ani_frames_NNN/`
(未 commit,依 heritage-preservation 紀律)。

---

## 6. 視覺比對:9 個資源 ↔ doc23 §2.4③ 分鏡 [驗,肉眼比對]

| 資源 | 幀數 | 內容(解碼結果肉眼判讀) | 對照 doc23 §2.4③ 分鏡 | dosbox 實拍幀 |
|---|---|---|---|---|
| ANI_003 | 28 | 紅底黑色守護者剪影,發光藍色項圈/紋章逐漸浮現(frame5→27 漸顯過程) | ① 守護者盔甲全身像 | frame_003–028 |
| ANI_004 | 12 | 藍髮主角(索爾)半身特寫,持劍,深藍披風,藍天背景 | ② 索爾半身像特寫 | frame_046–051 |
| ANI_005 | 35 | 索爾拔劍蹲姿,火山洞穴橘紅光背景,劍光一閃 | ③ 主角與巨龍對峙、拔劍、劍光一閃 | frame_052–059 |
| ANI_006 | 12 | 另一位深藍盔甲角色半身特寫,堅毅表情(第二段落人物) | ④「①②③重複一輪」的第二段落 | frame_077–101 |
| ANI_002 | 26 | 藍天白雲 + 滿月(frame10);後段近乎全白特寫(frame25,月輪拉近至滿版) | ⑤ 一輪明月(月亮背景素材) | frame_083–101 附近 |
| ANI_007 | 17 | 藍色鎧甲騎士騎馬,弧形月光背景 | ⑤ 騎馬夜行的隊伍剪影 + 明月 | frame_083–086/095–101 |
| ANI_008 | 12 | 索爾+夥伴 合照式半身群像 | ⑦ 主角一行人合照式半身群像 | frame_124–130 |
| ANI_000 | 96 | 金色鑲寶石鎖頭圖案,橢圓變形逐步收斂成正圓(frame20 側轉/frame50 橢圓/frame90+ 正圓定格) | ⑧ 金色鑲寶石鎖頭(黃金城展示) | frame_144–163 |
| ANI_001 | 51 | 「2」數字逐幀成形放大 + 紅色鋸齒 FLAME DRAGON 字樣,最終畫面即標題主選單 logo | doc23 §2.4① 「2」縮放進場動畫本體 | frame_188–194 |

**doc23 §2.4③ 未在 9 個資源中明確找到獨立對應的分鏡**:⑥ 浮空城疊在滿月前(可能是 ANI_002
或 ANI_007 較晚幀的延伸內容,本輪未逐幀窮盡掃描全部 26/17 幀確認)、⑨ 紅光惡魔臉特寫轉紅閃光
(可能是 ANI_002 白色/亮部特寫幀組的更早段落,或轉場閃光本身由外層播放器的調色盤操作產生而非
獨立 AFM 資源)。留待下一輪逐幀窮舉。

**重大附帶發現**:**doc23 §2.3/§2.4① 原本用 FDOTHER #7 靜態 blit 解釋的「FLAME DRAGON 2」標題
logo,其「2」數字縮放進場動畫實際也是由 `ANI.DAT` 資源 #1(AFM VM 動畫)驅動**,不是預存
5-6 張不同尺寸點陣圖的簡單 blit 序列(doc23 §2.4①原猜測),而是同一套 VM script 增量繪製 ——
這解答了 doc23 留下的「這組多尺寸『2』資源」缺口,直接更正該處推測。

---

## 7. 對重製(Go/Ebiten)的意義

- **AFM VM 可完整移植**:10 個 opcode 語意單純(填色/RLE/貼圖/調色盤更新),Go 端可直接用
  `image.Paletted` + `[]byte` framebuffer 複刻(`decode_ani.py` 的 Python 實作可逐行翻譯)。
- **不需保留原始 VM 執行順序的「增量疊加」語意**做即時播放——remake 可以離線把每個資源的
  289 幀全部展開成獨立 PNG(如本輪產出),播放端只需逐張貼圖,不必重新實作 VM state machine。
- **9 個資源即 9 段可獨立播放的片段**,可對應到 remake 的 `cutscene` 系統做逐段觸發,
  不需要一次性播完整 32 秒(如需要跳過/加速功能,原生的 `skippable` 參數已提供設計參考)。

---

## 8. 受阻 / 待補

- **[待驗]** `+0xA2`(3 bytes)、`+0xAB`(2 bytes)欄位語意未解出(9 個樣本恆為固定值組合,
  無變異可供歸納,可能是版本號/保留位)。
- **[待驗]** 8-byte 幀記錄中 `+4`/`+6` 兩個保留 word 欄位(9 個資源樣本全為 0),用途未明,
  推測可能是原編輯器(AFM Tool)的每幀 metadata(如原始繪製時間戳)、對播放邏輯無影響。
- **[待補]** doc23 §2.4③ 分鏡⑥(浮空城)、⑨(惡魔臉特寫)未逐幀窮盡確認落在哪個資源。
- **[待補]** 轉場閃光(紅/白 bloom)機制:是 AFM opcode 0/4(純色填滿)驅動,還是外層播放器
  在呼叫 `0x020421` 前後另外呼叫 `0x11d40`(VGA DAC 操作,doc06 提及)疊加——播放器 5 個
  呼叫點(`0x1f87a`/`0x1f9b2`/`0x1fd14`/`0x24404`/`0x2bdce`)週邊確實常伴隨 `0x11d40` 呼叫,
  但因這 5 個呼叫點分散在不同函式脈絡(不只是開場,可能也含其他過場如通關動畫),
  完整「哪個呼叫點對應開場哪一段」尚未逐一釘死,本輪只釘死了 ANI.DAT 內部格式與 9 個
  資源內容,呼叫端排程留下一輪。

---

## 9. 方法論註記

全程**靜態反組譯 + Python 重寫解碼器 + 視覺比對**(規則 62/64),未使用 DOSBox debugger。
關鍵轉折:一開始以為 8-byte 幀記錄的兩個 word 是「壓縮大小」與「畫面寬度」等圖像參數,
靠純推測差點卡住;改為**追蹤這兩個值在 `0x020421`/`0x36c9e` 裡實際的暫存器/堆疊用法**
(哪個被拿去 `fread` 當長度、哪個被拿去當 VM 的 `cmdCount`)才精確釘死語意——
再次印證規則 62「先追 sink 用法,別用格式直覺猜欄位」。
