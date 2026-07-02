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

---

## 10. 播放器排程(第14輪)[驗]

> 補完 §4.1/§8 留下的排程缺口:0x020421 三參數的**真實語意**、5 個呼叫點各自傳什麼值、
> 開場實際播放序 + 各幕 delay。方法:純靜態反組譯(規則 62),逐一 esp-relative 反推堆疊,
> 未動用 DOSBox。**本節推翻 remake `title.go` 現行 `cutSeq` 猜測**,細節見 §10.6。

### 10.1 0x020421 真實簽名:`play_afm(index, delayMs, skippable)`

逐 esp-relative 反推(cdecl,呼叫端 `push c; push b; push a; call`,故 callee 讀到
`[esp+0x28]=a`(第 1 個 C 參數)、`[esp+0x2c]=b`(第 2 個)、`[esp+0x30]=c`(第 3 個)):

| 參數 | 位址 | 語意 [驗] | 佐證 |
|---|---|---|---|
| **P1 = index** | `[esp+0x28]` | ANI.DAT 容器目錄索引。`0x0204aa`:`eax=[esp+0x2c]`——⚠此處 esp 已因前一條 `push 0`(`0x0204a8`)多減 4,故實際讀到的是 **`esp1+0x28`= P1**(不是 P2,首次反推曾誤判,已用逐指令 esp_rel 校正);`eax=P1*4+6` 當 `fseek` offset,對上 §2 的「6-byte magic + uint32 目錄」。P1==1 時**額外**觸發載入 FDOTHER.DAT 資源 `#0x4e`(78)當 frame0 的隨播「加料」(見 10.3)。 | `0x0204aa-b6` fseek 計算;`0x020446-57` FDOTHER 78 load |
| **P2 = delayMs** | `[esp+0x2c]` | 每幀播完後呼叫 `0x375b2`(jmp thunk → `0x3dccd`)**busy-wait 指定毫秒數**——不是 blit!doc39 §4.1 原標「0x375b2=blit」是誤判:VM opcode(§4.2)已直接寫 VGA `0xA0000`,不需要額外 blit 步驟;`0x375b2` 純粹是**幀間延遲**。 | `0x020555-59` |
| **P3 = skippable** | `[esp+0x30]` | ==0 時每幀跳過 `kbhit`(`0x10620`)判斷、不可按鍵中斷;非0 才允許任意鍵跳出整段播放。 | `0x020561-6f` |

### 10.2 delay 單位 = 真毫秒(有校準,非任意計數)[驗]

`0x3dccd`(`0x375b2` 的實際目標)：
```
eax = P2(=delayMs) * [0x541b0]      ; [0x541b0] = 校準常數
eax = (eax + 500) / 1000            ; 四捨五入
esi = max(eax, 1)                   ; busy-loop 次數
loop: mov ah,0x2c; int 0x21; dec esi; jne loop   ; INT21h AH=2Ch(取系統時間)當忙等時鐘
```
`[0x541b0]` 的寫入處 `0x3dc9f`(獨立校準函式,未找到其呼叫端,推測遊戲初始化時執行一次)：
用 **INT21h AH=2Ch 的 DH(秒)欄位**等到整秒邊界,再數 1 整秒內能執行幾次 `int21h/2Ch`
呼叫,存入 `[0x541b0]`。即「**這台機器 1 秒內能跑幾次 int21h 呼叫**」的即時校準值——
`P2*[0x541b0]/1000` 正是「P2 毫秒該等幾次 int21h 呼叫」的標準忙等換算式。
**結論:P2 的單位就是毫秒(ms),不是「PIT/BIOS tick 計數」,不需再標「單位待動態驗」**——
校準機制本身就是即時時鐘(DOS INT21h),換算式自洽,靜態即可確定物理單位。

### 10.3 companion 資源:兩套獨立的「隨播加料」機制 [驗]

- **`0x020421` 內建(P1==1 才觸發)**:載入 FDOTHER.DAT `#0x4e`(78)→ frame0 呼叫 `0x25a96`
  (SFX/音效觸發,doc23 §3 同一函式用於選單游標音)→ 播放結束後停止(`0x25a96(...,-1,...)`)並釋放。
  **只有 index==1(即 logo「2」那段)符合此條件**——呼應 doc23 §2.4①觀察到的
  「2縮放進場伴隨白色泛光 bloom」:很可能就是這個 FDOTHER #78 companion 資源驅動的閃光/音效。
- **wrapper `0x1f81e` 另一套(獨立於上面)**:第 3 參數 C(≠-1 時)當 FDOTHER.DAT 索引直接
  `load(filename,oldbuf,index=C)` 進快取 `[0x53a65]`,與 P1/P2/P3 轉送給 `0x020421` 無關。
  title_seq 用它在特定 esi 觸發點換底圖/道具貼圖(見 §10.5)。

### 10.4 5 個呼叫點逐一釘死

| 呼叫點(linear) | 所在函式 | index | delayMs | skippable | 呼叫方式 | 上下文 |
|---|---|---|---|---|---|---|
| `0x01f9b2` | title_seq `0x1f894` 本體(捲動迴圈**之前**) | **3** | **90** | 1(可跳) | 直接呼叫 | 開場第一段(守護者),播放前已載入 FDOTHER `#0x63`(99)當紅底背景 |
| `0x01f87a` | wrapper `0x1f81e`(**同一條指令被 6 次邏輯呼叫共用**,見 §10.5) | 依呼叫端 A 參數 | 依呼叫端 B 參數 | **恆 0(不可跳)** | 經 wrapper 轉呼叫 | title_seq 捲動迴圈內,esi 觸發 |
| `0x01fd14` | title_seq 本體(擦除轉場**之後**) | **1** | **15** | 1(可跳) | 直接呼叫 | logo「2」+ FLAME DRAGON 揭示(見 10.3 companion) |
| `0x024404` | `0x024336`(caller `0x0242c9`) | **0** | **15** | **0(不可跳)** | 直接呼叫 | **不是開場**——`0x0242c9` 緊接 `call 0x15f84`(章節劇情面板渲染,doc23 §4 同一函式)+ `inc dword ptr[0x53c03]`(章節變數自增!)。此為**某中期章節的劇情過場**重用金鎖素材,call-graph 從 main 不可達(進一步上層呼叫鏈未追完,推測經章節跳表深層間接呼叫)。 |
| `0x02bdce` | `0x02bce5` | **2** | **100** | **0(不可跳)** | 直接呼叫 | **不是開場**——`0x02bce5` 的呼叫端為 `0x2545d`/`0x25970`,兩者分別落在**戰後跳表 `0x51de9` 第 26/29 項**(章節 26、29 的勝利/戰後 handler)內。呼叫前另有獨立 `push 0x3e8; call 0x375b2`(先忙等 1000ms)與 `call 0x1f525`(調色盤淡入)。**確認 ANI_002(月亮)非開場資源,是章節 26/29 通關過場素材**,doc39 §6 原表該行需更正。 |

### 10.5 title_seq 完整排程(esi = 捲動列數,535→0,直接反組譯讀出)[驗]

`title_seq 0x1f894` 內部有一段**資料驅動的觸發表**:主捲動迴圈以 `esi` 從 `0x217`(535)遞減到 0,
每步呼叫 `0x11eb0` 複製一列(doc23 §2.1 已載明機制),**每步固定 delay 30ms**(`push 0x1e;call 0x375b2`),
迴圈中用一串 `cmp esi,常數` 硬編碼比對,命中即觸發播放,播完才繼續遞減:

```
迴圈前(esi 未進入迴圈時,setup 階段):
  → play_afm(index=3, delay=90, skip=1)         ; 直接呼叫 0x1f9b2,搭配 FDOTHER#99 紅底

esi==0x1c2(450) → call 0x1f73f(esi,edi,0x63,0x64)   ; ★不是 ANI.DAT!另一套「FDOTHER 靜態全螢幕
                                                        blit+淡入」機制(見 10.7),疑對應 doc23 §2.4③
                                                        未歸位的分鏡⑥/⑨
esi==0x14a(330) → wrapper(A=4,B=90,C=0x63) → play_afm(index=4,delay=90,skip=0)
                   接著 jmp 復用同一 call 指令 → wrapper(A=5,B=50,C=0) → play_afm(index=5,delay=50,skip=0)
esi==0xd2 (210) → wrapper(A=6,B=90,C=0x63) → play_afm(index=6,delay=90,skip=0)
                   接著 → wrapper(A=7,B=50,C=0) → play_afm(index=7,delay=50,skip=0)
esi==0x6e (110) → call 0x1f882(輔助/同步,細節未展開) → wrapper(A=8,B=90,C=0x63) → play_afm(index=8,delay=90,skip=0)
esi==0x19  (25) → wrapper(A=0,B=15,C=0)     → play_afm(index=0,delay=15,skip=0)     ; 金鎖(96幀)
esi==0xa   (10) → call 0x1f73f(esi,edi,0x4c,0x4b)   ; 同上,另一套靜態 blit 機制
esi==0     → 額外 +1000ms 停留(push 0x3e8;call 0x375b2)

第一段擦除轉場(esi: 0x28→0,每步 delay 8ms,call 0x286bd,即 doc23 §2.3 已載明的抹除轉場):
  → 結束後額外 +100ms(push 0x64)

  → 載入 FDOTHER #7(標題畫面)/#8,play_afm(index=1, delay=15, skip=1)   ; logo「2」+ FLAME DRAGON

第二段擦除轉場(esi: 0→0x28,每步 8ms,call 0x286bd,不同常數 0x38/0x3c/0x3f)
  → 存檔存在性檢查 → 進主選單迴圈(0x1fe2c,doc23 §3)
```

**AFM 實際播放序(依 esi 由大到小 = 時間先後):3 → 4 → 5 → 6 → 7 → 8 → 0 →(轉場)→ 1**
(共 7 段 ANI.DAT 播放 + 2 段擦除轉場,**index=2 完全不在 title_seq 內**——§10.4 已證實它是
章節 26/29 專用)。這與 doc39 §6 依 dosbox 視覺比對排出的順序(3,4,5,6,〔2〕,7,8,0,1)幾乎一致,
**只差 index=2 這一項要拿掉**;dosbox 實拍中「明月」畫面(item⑤)其實已由 ANI_007(騎馬夜行,
doc39 §6 已獨立列出)提供,不需要 ANI_002 補位。

### 10.6 對照 remake `title.go` 現行 `cutSeq` 猜測 —— 需要的修正

`remake/cmd/fd2/title.go` 現行:
```go
var cutSeq = []int{3, 4, 5, 6, 2, 7, 8, 0, 1}
const cutTicksPerFrame = 6   // 全域固定 6 tick/幀(≈100ms@60fps)
```

**應改為**(60fps→1 tick≈16.67ms,四捨五入;`skip`=能否按鍵跳過整段):

| 順序 | index | delayMs | tick/幀(60fps) | skippable |
|---|---|---|---|---|
| 1 | 3 | 90 | 5 | 是 |
| 2 | 4 | 90 | 5 | 否 |
| 3 | 5 | 50 | 3 | 否 |
| 4 | 6 | 90 | 5 | 否 |
| 5 | 7 | 50 | 3 | 否 |
| 6 | 8 | 90 | 5 | 否 |
| 7 | 0 | 15 | 1 | 否 |
| — | (轉場,8ms/步×41步+100ms) | | ~1 tick/步 | — |
| 8 | 1 | 15 | 1 | 是 |

**移除 index=2**(非開場資源,§10.4/10.5 已證實)。**每段各自的 delay 不同**,不是全域常數
`cutTicksPerFrame=6`;且只有播 index=3(第一段)與 index=1(logo)可按鍵跳過,其餘 5 段**不可跳**
——若要重製「按鍵跳過整段開場」,需模擬原版只在特定段落生效,或簡化為「跳過即整段全跳」
(玩家體感差異小,但如實記錄原版行為供決策)。

粗略總時長估算(538 步×30ms 捲動 + 7 段 AFM 播放時長總和 + 2 段轉場 + 額外停留)
≈ 16.05s(捲動)+ 9.8s(7 段 AFM:28×90+12×90+35×50+12×90+17×50+12×90+96×15)+ 1.0s(esi=0 停留)
+ 0.33s×2(兩段轉場)+ 0.1s(轉場間停留)+ 0.77s(index1×51幀×15ms)≈ **28.4 秒**,
與 doc23 §2.4③ dosbox 實拍量測的 **~32.3 秒**同量級(差距 ~12%,合理落在「初始 FDOTHER
載入/blit 磁碟耗時」「頭尾未計入的靜態畫面(0x1f73f 分鏡、片頭 logo)」誤差範圍內),
**佐證 delay=毫秒的判讀正確,不是量級錯誤**。

### 10.7 待補(本輪新發現的缺口)

- **[待補]** `0x1f73f`(esi==0x1c2 與 esi==0xa 觸發)是**另一條獨立的顯示機制**:接受
  `(esi,edi,FDOTHER_idxA,idxB)`,做 FDOTHER 全螢幕 blit(`0x4e63d`)+ 淡入(`0x1f525`)+
  BGM/SFX(`0x17aa9`),**不經過 ANI.DAT/VM**。極可能就是 doc23 §2.4③ 未歸位的分鏡⑥(浮空城)
  與⑨(惡魔臉特寫)的真正來源(FDOTHER 靜態圖,非 ANI.DAT 動畫)。下一輪可用同一手法
  (規則 62)反推 `0x1f73f` 內部載入的 FDOTHER 資源索引(0x63/0x64/0x4c/0x4b)並解圖比對。
- **[待補]** `0x024404`(index=0 重用)的完整呼叫鏈只追到 `0x0242c9`,再上層(哪個章節、
  透過哪個跳表項間接呼叫)未展開;不影響本輪「排除非開場呼叫點」的結論,但若要在 remake
  重現該中期章節過場,需要補這段。
- **[待補]** `0x1f882`(esi==0x6e 觸發前的輔助呼叫)內部只追到 `jmp 0x1f51e`,語意未展開
  (推測與遊戲速度設定/vsync 同步有關,不影響排程結論)。

**方法論**:全程 esp-relative 靜態反推(規則 62),**首次推導 `[esp+0x2c]` 時因漏算前一條
`push 0` 而誤判 P1/P2 對調**,靠逐指令記錄 `esp_rel` 表格才抓出並修正——再次印證「反推堆疊
偏移必須逐指令累計,不能憑印象跳算」。
