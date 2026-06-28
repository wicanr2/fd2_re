# 01 — 容器與資產格式

> 第 1 輪逆向工程成果(2026-06-28)。所有結論皆以原版實檔解碼驗證。

## 1. `.DAT` 通用容器格式 [已驗證]

漢堂把幾乎所有資產打包成同一種極簡歸檔容器。`FLAME2/` 下 12 個 `.DAT` 全部符合：

```
偏移   型別          說明
+0     char[6]       magic = "LLLLLL" (0x4C ×6)
+6     uint32[N] LE  offset 目錄;N = (offsets[0] - 6) / 4
                     每個 offset 指向一個 sub-resource 的起點(相對檔頭)
                     單調遞增。第 i 個資源長度 = offsets[i+1] - offsets[i]
                     最後一個資源延伸到檔尾。
```

判別與解包：先讀 magic，再以「第一個 offset 值」回推目錄項數。`tools/unpack_dat.py` 即此演算法。

### 各容器資源數(實測)

| 檔案 | 大小 | 資源數 | 內容(推定) |
|---|---:|---:|---|
| `FIGANI.DAT` | 15.3 MB | 409 | 戰鬥動畫(最大宗) |
| `DATO.DAT` | 1.98 MB | 137 | **人物頭像**(已解:每資源 4 幀講話嘴型 80×80,見下 §7) |
| `FDOTHER.DAT` | 3.38 MB | 104 | 雜項(含調色盤、UI) |
| `FDFIELD.DAT` | 243 KB | 100 | 地圖資料 |
| `FDSHAP.DAT` | 3.56 MB | 67 | 地形 / sprite 圖塊 + 地形控制表 |
| `BG.DAT` | 624 KB | 57 | 戰鬥背景 |
| `TAI.DAT` | 95 KB | 57 | 待定(小資源居多) |
| `FDTXT.DAT` | 120 KB | 35 | 章節文本 / 對白 |
| `FDMUS.DAT` | 80 KB | 21 | 音樂(曲目對映?) |
| `ANI.DAT` | 2.44 MB | 10 | 動畫(含 "AFM " chunk) |
| `TITLE.DAT` | 23 KB | 8 | 標題 / 片頭圖 |

> `FDICON.B24`(624 KB)**無** LLLLLL magic，是另一種格式 [假設] 可能是 24-bit 圖或圖示集，待後輪解。
> `FD2.SAV`(存檔, 22987 B)高熵、無 magic，疑似加密或壓縮 [假設]，是重製存檔相容性的 RE 標的。

## 2. 圖像格式 [已驗證]

圖像資源結構：

```
+0  uint16 LE  width
+2  uint16 LE  height
+4  像素資料,8-bit VGA 調色盤索引(mode 13h, 320×200 基準)
```

像素資料兩型：
- **未壓縮**：body 長度 == W×H(實例 `FDOTHER_015` / `FDOTHER_055`，size=64004=4+320×200)。
- **RLE 壓縮**(第 2 輪破解，視覺驗證通過)：

```
讀 byte c:
  c >= 0x80 : 文字串(literal),其後 (c & 0x7F)+1 個 byte 原樣輸出
  c <  0x80 : 連續執行(run),下一個 byte 重複 (c+1) 次
重複到輸出滿 W×H。
```

判定強約束：正確解碼必輸出**剛好 W×H** 個像素。此演算法在 `TITLE`(320×200)、`BG`(320×100)、
`FDOTHER` 背景上精確命中並渲染出正確畫面(標題 logo「FLAME DRAGON 2 — Legend of Golden Castle」、
山脈/村莊/熔岩等戰鬥背景)。工具：`tools/decode_image.py`。

**調色盤**：`FDOTHER` 資源 #0 = 768B(256×RGB, 6-bit)，×4 轉 8-bit;對標題與背景皆正確。
不同畫面可能切換調色盤，後輪對應。

| 樣本 | 寬×高 | 類型 | 內容 |
|---|---|---|---|
| `TITLE_000` | 320×200 | RLE | 遊戲標題畫面 |
| `BG_003` | 320×100 | RLE | 戰鬥背景(山脈) |
| `FDOTHER_015` | 320×200 | 未壓縮 | 熔岩材質 |
| `FDSHAP_000` | 24×24 起 | 待解(sprite) | 24×24 圖塊集，~256 格 |

**覆蓋**：背景/標題類全幅圖 ~125 張已可解。**sprite 類**(FDSHAP 圖塊、`FIGANI` 戰鬥動畫、
`DATO` 立繪)多數用**透明/skip 變體**的 RLE(非矩形)，尚未解 → 第 3 輪。**24×24** 為基本圖塊尺寸。

## 3. 調色盤 [已驗證]

`FDOTHER_000` = 768 bytes，值域 ≤ 0x3F → 標準 VGA 256 色調色盤(256 × RGB，每色 6-bit 0–63)。
重製時需 ×4(或 <<2 補低位)轉成 8-bit RGB。其他容器可能各自帶調色盤，待清點。

## 4. 文本格式 [已驗證 兩層結構]

`FDTXT.DAT`(35 資源)為章節文本。每個資源**內部再分條目**：資源開頭是一張 uint16 LE 的次目錄
(指向各字串)，例如 `FDTXT_000` 開頭 `2a 05 74 05 7a 05 …` = 0x052a, 0x0574, …。
字串本體**不是 Big5**,而是**自製字型的 uint16 glyph 索引** + 控制碼 + `0xFFFF` 結尾(第 3 輪已破解,
見 `08-text-and-font-format.md`)。這是中文化的核心改寫點。

## 5. 地形控制表 [已驗證，交叉印證攻略]

`FDSHAP_001` = 1200 bytes = 300 格 × 4 byte，其檔內偏移 `0x2422E` **正好等於**青衫攻略
modify2 宣稱的「地形控制資料 2422Eh」。每格 4 byte：

```
byte0  寶箱資訊  bit5(0x20)=寶箱  bit6(0x40)=隱藏物品
byte1  移動資訊  0=正常(AP+5) 1/5=不可移動 2=騎士減速 3=全體減速 4=全體大減速
byte2  戰鬥背景編號  uint16 LE
```

實測前段多為 `00 00 04 00`(無寶箱、正常地形、背景#4)，與格式相符。

## 6. 音效驅動(非遊戲資料)

`*.MDI` / `*.DIG` / `SAMPLE.*` / `DOS4GW.EXE` / `SETSOUND.EXE` 屬 Miles Sound System(AIL)
驅動與 DOS extender，非遊戲內容；重製時以現代音訊管線取代，毋須逆向。

## 7. 人物頭像格式(DATO.DAT)[已驗證]

`DATO.DAT`(137 資源)= 對話頭像。每資源:

```
+0  uint32[4]   4 個子圖 offset(= 講話時的 4 種嘴型幀)
各子圖: uint16 W, uint16 H(多為 80×80), 然後 RLE 像素
```

RLE codec(反組譯 `0x4F716`,比 sprite 簡單,無透明):
```
讀 byte b:  b <= 0xC0 → 字面像素(值=b);  b > 0xC0 → run,重複 (b-0xC0) 次下一 byte
```

頭像索引 = 肖像 ID(`DATO_000`=索爾、`001`=哈諾…對上 memory.md 肖像表,已逐張驗證)。
對話框由控制碼 `0xFFEF` 依說話者肖像 ID 載入對應頭像(見 `14-text-control-codes`)。
工具 `tools/decode_dato.py`;全 136 頭像 ×4 幀已匯出本機 `extracted/portraits/`。
