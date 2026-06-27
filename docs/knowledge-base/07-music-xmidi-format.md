# 07 — 音樂格式(XMIDI)紀錄

> 《炎龍騎士團2》的背景音樂系統。第 2 輪逆向工程整理。
> 與圖像、動畫併為 1990 年代台灣 DOS 遊戲技術的保存紀錄。

## 技術選型:Miles Sound System + XMIDI

原版音訊走 **Miles Design Audio Interface Library(AIL)v3**(見 `04-original-toolchain.md`)——
當年商用音訊中介層,讓一份音樂資料能在 AdLib / OPL3 / Sound Blaster / MPU-401 / Roland MT-32 /
Gravis UltraSound 等各種音效卡上播放(故有整套 `.MDI` / `.DIG` 驅動)。

音樂資料格式為 **XMIDI(.XMI)**:Miles 為 AIL 設計的 MIDI 擴充格式,以 IFF(EA 85)chunk 封裝,
打包在 `FDMUS.DAT` 容器內。

## FDMUS.DAT 內容

21 個資源中:
- **15 首 XMI 樂曲**(以 `FORM…XDIR` 開頭)。
- 數個 **3-byte 分隔標記**(`20 0D 0A` = 空白+CRLF),作群組分隔,非音樂。

## XMI 檔結構(IFF chunk)

```
FORM XDIR
  INFO            2 byte:序列(曲)數
CAT  XMID
  FORM XMID
    TIMB          樂器音色表(每 2 byte:patch, bank/ch)
    EVNT          事件流(見下)
  (可有多個 FORM XMID = 多序列)
```

所有多位元組長度為 **big-endian**(IFF 慣例),與容器/圖像的 little-endian 相反——解析時要切換位元序。

## XMIDI 與標準 MIDI 的兩大差異

這是把 XMI 轉成現代可播放 `.mid` 的關鍵:

1. **延時用「間隔累加」**而非標準 MIDI 的 variable-length:
   事件之間連續的 `< 0x80` 位元組逐一**相加**成 delta tick(每 byte 貢獻 0–127)。

2. **Note On 自帶持續長度**:`0x9n note vel` 之後緊跟一個 **VLQ(標準 7-bit 可變長)** 表示音長;
   XMIDI **不發 Note Off**,播放器須在「現在時間 + 音長」排程關閉該音。

其餘(program change、controller、pitch bend、tempo meta `FF 51`、end-of-track `FF 2F`)與 MIDI 相同。

## 轉換為標準 MIDI

`tools/xmi2mid.py` 實作上述還原:
- 累加間隔得 delta tick;
- Note On 讀內嵌音長 → 排程對應 Note Off;
- 其餘事件直通;
- 輸出 SMF type 0,PPQN=60。

```bash
python3 tools/xmi2mid.py --info  FDMUS_008.bin          # 只分析
python3 tools/xmi2mid.py --batch FDMUS/  midi_out/       # 全部轉檔
```

**驗證**:輸出 SMF 經回讀,**note-on 與 note-off 完全平衡**(如 FDMUS_008:2846 = 2846);
每首皆含 tempo meta 事件(直通,故速度正確,毋須臆測 XMIDI 的 120Hz 基準)。

## 15 首樂曲分析

| 資源 | 音符數 | 使用聲道 | 樂器數 | 長度(拍) |
|---|---:|---|---:|---:|
| FDMUS_001 | 1337 | 1-4, 9 | 9 | 90 |
| FDMUS_003 | 2234 | 1-5, 9 | 13 | 115 |
| FDMUS_004 | — | — | — | — |
| FDMUS_006 | 2234 | 1-5, 9 | 13 | 115 |
| FDMUS_008 | 2846 | 1-6, 9 | 10 | 250 |
| FDMUS_010 | 979 | 1-7, 9 | 15 | 96 |
| FDMUS_011 | 513 | 1-3 | 3 | 228 |
| FDMUS_012 | 572 | 1-5, 9 | 8 | 24 |
| FDMUS_013 | 1591 | 1-6, 9 | 10 | 128 |
| FDMUS_014 | 588 | 1-7, 9 | 11 | 77 |
| FDMUS_015 | 877 | 1-7, 9 | 10 | 88 |
| FDMUS_016 | 91 | 1-5, 9 | 7 | 5 |
| FDMUS_017 | 56 | 1-3 | 4 | 15 |
| FDMUS_018 | 2256 | 1-10 | 10 | 137 |
| FDMUS_019 | 329 | 1-4, 9 | 7 | 35 |

(**聲道 9** = General MIDI 打擊樂軌,印證遊戲音樂有鼓組;短曲如 FDMUS_016/017 應為勝利 / 事件提示樂句。)

## 重製對應

XMI 已轉標準 MIDI,C++(SDL2_mixer / 自帶合成)與 Go/Ebiten 皆可用 SoundFont 合成播放;
亦可重新編曲。原始樂曲著作權屬漢堂,轉出的 MIDI 不隨倉庫散布(僅保留工具與分析)。

## 待辦(後輪)

- TIMB 音色表逐曲對應 GM program,標出各曲配器。
- 以 SoundFont 渲染試聽,校準 XMIDI tempo 換算(目前直通 tempo meta,聽感校準待補)。
- 對應每首曲在遊戲中的使用場景(片頭 / 戰鬥 / 城鎮 / 勝敗)。
