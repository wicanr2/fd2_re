# 35 — 全螢幕戰鬥演出繪圖機制(FIGANI 攻守動畫)

> 反組譯《炎龍騎士團2》FD2.EXE(DOS4GW LE,obj1 linear=file offset)的**全螢幕戰鬥演出**:
> 攻擊發生時切到的那張大圖畫面 —— 守方 / 攻方全身 FIGANI、戰場背景、狀態欄、斬擊與閃紅。
> 所有結論附反組譯位址佐證;runtime 才決定的值標「待確認」。
> 相關:doc 06(FIGANI 格式)· doc 31(FDICON 地圖小人,另一套)· doc 13(戰鬥選單)· doc 25(戰鬥事件)。

## 0. 兩個演出函式(別搞混)

| 函式 | 入口 | 參數 | 用途 |
|---|---|---|---|
| 單圖演出 | **0x28784** | 1 個 unit index | 顯示**單一**單位全身圖(施法/單體演出) |
| 攻守演出 | **0x28a6c** | **2 個** unit index(攻方、守方) | 攻擊的**對打**全螢幕演出(本篇重點) |

兩者 prologue 都是 Watcom 風格 `push <frameSize>; call 0x36cd7`(stack-probe):
- 0x28784:`push 0x54; call 0x36cd7`(0x28789),func 範圍 0x28784–0x28a6b `ret`。
- 0x28a6c:`push 0x64; call 0x36cd7`(0x28a71),func 範圍 0x28a6c–0x29116 `ret`。

> ⚠ 既有錨點把 0x287b5(`movzx esi,[ebx+7]`)當成「攻守演出」入口,實際它在**單圖** 0x28784 內;
> 真正的攻守雙圖演出是 **0x28a6c**(0x28ad6 `movzx eax,[ebx+7]` 攻方組、0x28ade `movzx eax,[esi+7]` 守方組)。

### 呼叫鏈(誰觸發演出)

`calls 0x28a6c`(相對 call 來源):

| caller linear | 傳參 | 意義 |
|---|---|---|
| **0x1561f** | `push [0x53c4b]; push ebx; call` → `0x28a6c(ebx, [0x53c4b])` | 攻擊執行流:**arg0=ebx=攻方 idx、arg1=[0x53c4b]=守方 idx** |
| 0x18fc6 | — | 另一觸發點(待確認) |
| 0x2c2aa | `mov [0x540ff],ecx; push 1; push 0; call` → `0x28a6c(0, 1)` | 先設演出 phase 旗標再呼叫 |
| 0x35435 | — | 另一觸發點(待確認) |

`calls 0x28784` → 唯一 caller 0x15195(單圖路徑,與 0x1561f 同一攻擊執行區 0x15xxx,符合「攻擊執行 0x15xxx」推測)。

**[0x540ff] = 演出 phase / 重繪旗標**(`refs 540ff` 找到 writer):函式外 0x25ac0、0x25b6f 設定;函式內 0x28ae8/0x28c15/0x28cdb/…/0x28ef1(設 1)讀寫。
語意:`0` = 第一次進場(載入資源 + 全畫面合成),非 0 = 後續增量幀(只重畫變動層)。動畫靠**重複呼叫 0x28a6c 推進**,phase 旗標決定這一幀做哪些事。

---

## 1. FIGANI 載入(組 × 3)+ buffer

- 攻方組 = `unit_attacker[+7]`(0x28ad6 → `[esp+0x10]`);守方組 = `unit_defender[+7]`(0x28ade → `[esp+0xc]`)。
  與 doc 31/06 一致:`unit[+7]` = FDICON/FIGANI 組號。
- **FIGANI 動畫 index = 組 × 3**:0x28c57 `mov edx,[esp+0x10]; shl eax,2; sub eax,edx`(= 組×4−組 = 組×3),
  再 0x28c78 `組×3`、0x28c99 `inc`(組×3+1)。**每組 3 個動畫**(待機 / 出招 / 受擊,+0/+1/+2;對映待確認),
  確認了既有結論「FIGANI = sprite組 × 3」。
- 載入經 **0x111ba**(資源解碼器,見 §6),descriptor = **0x52388**(FIGANI 動畫容器表):
  `0x111ba(0x52388, prevSlot, 組×3+k)` → 回傳該動畫的 frame 描述子 buffer。
- 解出的動畫描述子存:**[0x54117]=攻方、[0x5411b]=守方**(0x28e4a / 0x28e5b,經 `0x2bc9a` 後處理)。
  另 [0x53a49] / [0x53a5d] 為單圖路徑(0x28784)用的 FIGANI buffer(對映既有錨點)。
- 龍騎兵 / 飛行特例:0x28b72 檢查 `unit[+0x20]==0x13`(職業 0x13=龍騎士)或 `unit[+0x1f] in {4,5}` 且 `unit[+7]==0x1c` → 走特殊組路徑(`call 0x12e38` 換組)。

> 動畫描述子格式(0x2939d / 0x2935b 讀法):`byte[ebp]` = frame 數;`[ebp + i*4 + 8]` = 第 i 幀相對 offset;
> `byte[ebp+1]` = 類型旗標(0=靜態單幀走 BG 路徑,非 0=多幀動畫)。

---

## 2. 守 / 攻 blit 座標 + 縮放(最關鍵)

### 2.1 blit 原語 0x4e63d(原生尺寸,無縮放)

`0x4e63d(src, X, Y, dst, stride, transp)`(由 `ebp+8..ebp+0x1c` 取參,0x4e643 起):
```
esi = src                         ; 來源(自帶尺寸)
word[src+0]→[0x627b4] = 寬          ; 0x4e646 lodsw
word[src+2]→[0x627b6] = 高          ; 0x4e64e lodsw
ecx = X (ebp+0xc)
eax = Y (ebp+0x10);  edx = stride (ebp+0x18)
edi = dst (ebp+0x14);  edi += Y*stride + X      ; 0x4e663 mul / 0x4e665-667
transp = ebp+0x1c   (-1 = 用 RLE 透明跳過;否則色鍵)
```
**關鍵結論:`dst 位址 = dst + Y*stride + X`,圖以 src header 自帶的寬高原生繪製。整條 blit 路徑沒有任何 `imul`/`fild`/`fmul` 縮放運算。**
→ **守方較小、攻方較大不是 runtime 縮放,而是 FIGANI 美術本身就畫成不同尺寸**(景深感燒進素材)。
remake 對不準的根因即在此:該照各 frame header 的寬高 + 下面的座標貼,不要自己 scale。

### 2.2 座標來源

- **螢幕錨點常數 (X=0xa4=164, Y=0x9d=157)**:出現在 0x28f55(`0x4e63d(src, 0xa4, 0x9d, edi, 0x280, -1)`)、
  0x29164 狀態欄(0x291b0/0x29268/0x29295)。這是演出區/單位圖的**固定錨點**(160×100 半屏中央偏下)。
- **每個 figure 的 X 來自 `word[unit+0x40]`**:0x294ad `movzx eax, word[eax+0x40]`(eax=守方 unit ptr)→ `[esp+0x50]`,
  再 push 進 figure 繪製(0x29582 `push [esp+0x50]` → 0x2935b)。
  → **figure 水平位置 = unit 結構 +0x40 的 word 欄位**(戰場格位投影到螢幕的 X;攻守不同因 unit 不同)。
- **frame 自帶 (dx,dy)**:figure 繪製 wrapper **0x2935b** 解單幀:
  ```
  eax = frameIdx*4 + descriptor          ; 0x2936a
  edx = descriptor + [eax+8]             ; 該幀資料 ptr
  word[edx+0]=幀X偏移, word[edx+2]=幀Y偏移 ; 0x2937a/0x2937d
  src 像素 = edx+9                        ; 0x2938f add eax,9
  → 0x4e63d(edx+9, 幀X, 幀Y, dst, stride, transp)
  ```
  → 每幀內嵌自己的 (dx,dy),**斬擊弧 / 出招前傾 / 受擊後仰就是逐幀換 (dx,dy)**。

> **守 vs 攻 的水平分離量**:由 `word[unit+0x40]`(runtime 戰場位置)+ 各幀內嵌偏移決定 → **精確像素差待確認**
> (需 runtime 兩個 unit 的 +0x40 值);但「攻方在右、守方在左、各自 X 由 unit+0x40 給、尺寸由素材決定」已由上面靜態確認。
> 守方/攻方在程式裡靠 0x28b51 `movzx eax,[esi+6]`(守方 +6 旗標)與 0x28dfd 同檢查決定**畫的先後與左右 buffer 交換**
> (0x28e05-0x28e16 視 [esi+6] 交換 [0x54107]↔[0x54103])。

---

## 3. 背景 BG 繪製 + 戰場→BG 對應

### 3.1 BG 多層載入 + blit

- 演出進場前,BG.DAT 由 **0x22d1b** 載入(既有錨點;0x2866x 區三次 `0x22d1b` 以 index 對載地形圖,在前置函式內)。
- 演出函式內,BG 分**多層**經 0x111ba(descriptor = **0x52381**)解出:
  - [0x54107](0x28cd4)、[0x54103](0x28daa, idx 0)、[0x5410b](0x28dc4, idx 0)、[0x5410f](0x28dde, idx 1)、[0x54113](0x28df8, idx 2)。
  - → 至少 **3–5 層**(idx 0/1/2),疑似遠景 / 近景 / 土台分層(攻方「站土台」的土台可能是其中一層)。**各層內容待確認**。
- **BG blit 座標**:`0x4e63d(BGsrc, X=0, Y=0x32=50, dst, stride, -1)`:
  - 0x28d36(`[0x54107]`,stride 0x140=320)、0x28e1b(`[0x54103]`,stride 0x140)。
  - → **背景貼在 X=0, Y=50,寬 320**(整屏寬;上方 50px 與下方留給狀態 UI)。

### 3.2 戰場 → BG 參數對應

- **章節參數表 0x52363**:`[0x53c03]`(= 章節 byte,既有錨點)當 index 取 `byte[chapter + 0x52363]`:
  0x28b61 `mov edx,[0x53c03]; movzx edx, byte[edx+0x2363]` → `[esp+0x18]`。
  表值 = `[4, 9, 14, 18, 14, 0, 2, 4, 6, 8, 10, …]`(0x52363 起,前 6 個是章節用)。
- 此值參與 figure / BG 選擇分支(0x28b89 與龍騎兵特例聯動),**確切語意(選哪張 BG / 色盤)待確認**;
  但「**戰場→演出參數由章節 [0x53c03] 索引 0x52363 表**」這條對應已確認。
- 既有錨點 BG_004 森林 320×100:與本處「BG 寬 320、貼在 y=50」尺寸吻合。地形→BG index 的細部對應待續(追 0x22d1b 的 index 參數來源)。

---

## 4. 狀態欄繪製(0x29164)

`0x29164(...)` 為單一單位畫**全身圖 + 狀態條**(攻守各呼叫一次,0x28e76 / 0x288f3):

- prologue `push 0x2c; call 0x36cd7`;`esi = [0x53a45]+idx*80`(0x2917c-0x29184,**單位 80=0x50 byte stride**,`idx*5*16`),確認既有錨點。
- 0x2918a `movzx ebx, byte[unit + 6]` → 分支(+6 旗標,同 §2 守方判定):
- **全身圖**:0x4e63d 於 **(X=0xa4=164, Y=0x9d=157)** 貼圖(0x291b0 / 0x29268 / 0x29295)。
- **8 次迴圈**(`esi=8 → 0`,0x29197 / 0x2920d):每圈
  - `imul ebp, esi, 0xa`(×10):層 / 行間距;
  - **HP/狀態條 = 色盤寫入** `0x11d40(0, 0xff, esi*6)`(0x291f9 `imul ebx,esi,6`):**0x11d40 直接寫 VGA DAC 埠**(見 §5)。
  - 每圈 `0x11eb0` 把 work buffer 段 present 到 VGA 0xa0000(0x291ec)。
- **數字 / 名字 / LV**:經 0x2935b(0x291cf / 0x29247)以幀方式貼(name/LV/HP/MP 為預渲染圖塊)。
- 座標體系:work stride 0x280=640、螢幕寬 0x140=320、高 0xc8=200、VGA 0xa0000(全部出現,§6)。

> 攻方欄 / 守方欄的左右分屬:0x29164 被呼叫兩次,各傳不同 unit idx + 不同 buffer base;**各欄絕對 X 偏移待確認**
> (主錨點 164,157 是共用 figure 位;狀態條相對位由迴圈 `esi*10` / `esi*6` 決定)。顏色來源 = 色盤 index(0x11d40 寫 0x3c9)。

---

## 5. 動畫階段機制(windup → swing → impact → standoff)

- **驅動**:phase 旗標 **[0x540ff]** + **重複呼叫 0x28a6c**(每幀一次);非「一次畫完整段」。
  進場 phase=0(載資源 + 全合成,0x28e3e 區算 [0x54117]/[0x5411b]),之後 0x28ef1 設 [0x540ff]=1 走增量。
- **進度百分比**:figure renderer **0x2939d** 內 0x2946a `call 0x4e893` → 0x2947b `idiv 100`(`mov ebx,0x64; idiv`),
  取餘數判斷階段(`cmp edx,3` < 3 時 `[esp+0x4c]=2`,多畫一層)→ **動畫進度以 0–99% 表示,百分比決定當前幀 / 疊層**。
- **幀迴圈**:0x2939d 以 `byte[ebp]`=幀數迴圈(0x29409-0x29424),逐幀經 0x29f72(單幀子繪製)+ 0x2935b 貼;
  幀的 (dx,dy) 內嵌(§2.2)→ **swing 斬擊弧 = 逐幀位移 + 換幀**。
- **idle / fallback 描述子**:0x2939d 進場 `rep movsd` 從 **0x5255f**(6 dword)與 **0x52577**(6 dword)複製預設描述子到區域 frame
  (0x293cf / 0x293df)→ 沒有真實動畫時的**待機姿態 fallback**。
- **閃紅 / 抽 HP(impact)= 色盤操作,不是重畫像素**:
  - **0x11d40** 是 VGA DAC 寫入迴圈:`push 0x3c8 / push 0x3c9; call 0x37795`(0x11d5c / 0x11d73)→
    out 到埠 **0x3c8(palette index)/0x3c9(palette data)**。0x37795 = DAC 埠寫入原語。
  - 同手法在 0x28784 / 0x286dd 的 fade-in 迴圈(用 0x53a65 色表插值)。
  - → **守方受擊閃紅 = 改色盤**(把該圖用色暫時拉紅再復原);HP 條變化亦走 0x11d40 色盤,效能極低成本。
    精確的「閃紅幀數 / 色值序列」待確認(需追 0x37795 的色值來源表)。
- **standoff**:演出結束 0x290xx 釋放所有 buffer(0x28fc1-0x2900e 連續 `0x37416` free)、復原色盤(0x290b8 `0x375c0`)、`0x11cac` 還原畫面。

---

## 6. 螢幕座標系(確認)

| 量 | 值 | 出處 |
|---|---|---|
| 螢幕寬 | 0x140 = 320 | 0x28f3c / 0x4e63d stride 參數 |
| 螢幕高 | 0xc8 = 200 | 0x28f37 / 0x11eb0 rows |
| VGA framebuffer | 0xa0000 | 0x28945 / 0x28fb4 |
| **work buffer stride** | **0x280 = 640** | 0x28f47 / 0x28f57 / 0x2935b |
| present 來源寬 | 0x140 = 320 | 0x11eb0 bytesPerRow |

- **work buffer 是 640 寬、但只 present 左半 320**:`0x11eb0` 每列 memcpy 320 byte、來源 stride 640、200 列 → VGA 320×200。
  雙倍寬 work buffer(`lea ebx,[edi+0x140]` 0x2929 系列存取右半)疑作**off-screen 預備區**(下一幀 / 滑入的 figure 先畫右半再捲入),
  具體用途待確認,但「**work stride 640、可視 320**」這條已確認 → remake 若用單寬 buffer 要注意座標換算。
- present 原語 **0x11eb0**(`rows, dstStride, src/dst, srcStride …`,逐列 `0x373c4` memcpy):BG→work、work→VGA 都走它。

---

## 7. 函式 / 位址速查

| 位址 | 角色 |
|---|---|
| 0x28784 | 單圖演出(1 unit) |
| **0x28a6c** | **攻守演出主函式(2 unit)** |
| 0x29164 | 單位全身圖 + 狀態條繪製 |
| 0x2935b | 單幀 figure 貼圖 wrapper(解 frame header dx/dy → 0x4e63d) |
| 0x2939d | figure 動畫 renderer(幀迴圈 + 百分比進度) |
| 0x29f72 | 單幀子繪製(0x2939d 內,細節待續) |
| 0x4e63d | blit 原語(原生尺寸 RLE,dst+Y*stride+X) |
| 0x11eb0 | 矩形 present(逐列 memcpy,work↔VGA) |
| 0x11d40 | VGA DAC 色盤寫(閃紅 / HP 條 / fade,埠 0x3c8/0x3c9) |
| 0x111ba | 資源解碼器:`(descriptor, prevSlot, index)` → 解 entry[index],釋放 prevSlot,回新 buffer |
| 0x22d1b | BG.DAT 載入(前置) |
| 0x4e893 | 動畫進度來源(被 idiv 100) |
| 0x37795 | VGA DAC 埠寫入原語 |

### 關鍵 descriptor / 變數
| 符號 | 意義 |
|---|---|
| 0x52381 | BG 多層 descriptor(0x111ba 用) |
| 0x52388 | FIGANI 動畫 descriptor(index = 組×3+k) |
| 0x52393 | 演出輔助 descriptor(index 含章節值;語意待確認) |
| 0x52363 | 章節→演出參數表 `[4,9,14,18,14,0,…]`(`[0x53c03]` 索引) |
| 0x5255f / 0x52577 | idle / fallback 動畫描述子(各 6 dword) |
| [0x53a45] | 單位陣列基底(每單位 80 byte) |
| [0x540ff] | 演出 phase / 重繪旗標 |
| [0x54117] / [0x5411b] | 攻方 / 守方 FIGANI 動畫描述子 buffer |
| [0x54107]/[0x54103]/[0x5410b]/[0x5410f]/[0x54113] | BG 各層 buffer |
| [0x53c03] | 當前章節 |
| [0x53c4b] | 守方 unit idx(0x1561f 傳入) |
| unit[+7] | FIGANI/FDICON 組號 |
| unit[+0x40] (word) | figure 螢幕 X(戰場位置投影) |
| unit[+6] | 守方 / 旗標(決定左右 buffer 交換) |

---

## 8. 六項成果摘要 + 待確認

1. **入口 + 呼叫鏈** ✅:單圖 0x28784(caller 0x15195)、攻守 0x28a6c(caller 0x1561f 傳 `攻方ebx, 守方[0x53c4b]`,另 0x18fc6/0x2c2aa/0x35435)。phase = [0x540ff]。
2. **守/攻 blit + 縮放** ✅(重點):blit 原語 0x4e63d 原生尺寸、`dst+Y*stride+X`,**全程無縮放運算 → 大小燒進 FIGANI 素材**;figure X = `word[unit+0x40]`;每幀內嵌 (dx,dy)(0x2935b);共用錨點 (164,157)。**待確認**:攻守精確像素分離量(runtime unit+0x40)。
3. **BG 繪製** ✅:多層 descriptor 0x52381 → [0x54107…54113],`0x4e63d(X=0,Y=50,寬320)`;戰場→章節 [0x53c03] 索引 0x52363。**待確認**:各 BG 層內容、地形→BG index 細節。
4. **狀態欄 0x29164** ✅:全身圖 (164,157)、8 次迴圈、HP/狀態條走色盤 0x11d40、present 0x11eb0。**待確認**:攻欄/守欄絕對 X、名字/數字相對排版細節。
5. **動畫階段** ✅:[0x540ff] phase + 重複呼叫驅動;0x2939d 幀迴圈 + `idiv 100` 百分比進度;幀 (dx,dy) = swing 斬擊弧;**閃紅/HP 抽乾 = VGA DAC 色盤 0x3c8/0x3c9(0x11d40)**;idle fallback 0x5255f/0x52577。**待確認**:閃紅色值序列、各階段確切幀數。
6. **座標系** ✅:320×200、VGA 0xa0000、**work stride 640 但只 present 左半 320**(雙寬 off-screen 預備區,用途待確認)。
